// T7.7 — object-level authorization regression tests for Facilities'
// write endpoints, run through the real gRPC handler (not just the
// domain-level unit tests internal/facilities/domain/facility_test.go and
// internal/facilities/app/service_test.go already have). This is the
// "does the guarantee survive the full stack" test the T7 sprint plan's
// T7.7 ticket asks for (docs/process/t7-sprint-plan.md, T7.7) — the third
// instance of this pattern after T5.5 (Social Play) and T6.7 (Payments).
//
// Scope note: T7.3 shipped CreateFacility, GetFacility, ListFacilities,
// AddCourt, and AddCameraLink — no UpdateFacility RPC exists. Per the
// ticket's own instruction ("if only CreateFacility/ListFacilities/
// GetFacility/AddCourt exist, scope this ticket to AddCourt only"),
// AddCameraLink is also included here because T7.3 did in fact ship it
// (proto/pickleball/facilities/v1/facilities.proto) — it's a second real
// write RPC beyond CreateFacility, not an invented one. CreateFacility
// itself has no cross-owner rejection to prove (there's no existing
// Facility to be scoped against yet — the caller is the one setting
// owner_id), so this file covers the two RPCs that mutate an
// *existing* Facility: AddCourt and AddCameraLink.
//
// This is a handler-level test (real grpcapi.Handler + real app.Service +
// real domain, with an in-memory fake standing in for
// internal/facilities/adapter/postgres) rather than a `-tags=integration`
// testcontainers-go test like internal/booking/adapter/postgres/
// concurrency_integration_test.go. Two independent reasons, mirroring
// T5.5's authz_regression_test.go exactly:
//
//  1. The ticket explicitly allows "a handler-level or -tags=integration
//     test (implementer's choice, same reasoning T5.5 used for picking
//     handler-level over a full Postgres round trip when the check itself
//     has no SQL involved)" — the ownership check under test lives
//     entirely in domain.Facility.EnsureOwner/AddCourt/AddCameraLink (see
//     internal/facilities/domain/facility.go and
//     internal/facilities/app/service.go), which port.Repository doesn't
//     influence; a real Postgres round trip would add infrastructure, not
//     proof.
//  2. This environment has no Docker daemon (the same gap
//     concurrency_integration_test.go's and T5.5's own package comments
//     already document for this repo). CLAUDE.md rule 10 means this
//     ticket's own regression test needs to actually run in an
//     environment we can execute it in, not just be plausible in CI.
package grpcapi_test

import (
	"context"
	"fmt"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/nhuthuynh/white-label/internal/facilities/adapter/grpcapi"
	"github.com/nhuthuynh/white-label/internal/facilities/app"
	"github.com/nhuthuynh/white-label/internal/facilities/domain"

	facilitiesv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/facilities/v1"
)

// --- in-memory port.Repository fake -----------------------------------
//
// Stands in for internal/facilities/adapter/postgres for this test only.
// Implements the exact same port.Repository interface the real Postgres
// adapter does, so app.Service and grpcapi.Handler run unmodified, real
// production code — only the persistence boundary is faked. Mirrors
// internal/facilities/app/service_test.go's inMemoryRepo and
// internal/socialplay/adapter/grpcapi/authz_regression_test.go's
// fakeGameRepo/fakeRegistrationRepo.
type fakeRepo struct {
	facilities map[string]domain.Facility
	courts     map[string]domain.Court
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		facilities: make(map[string]domain.Facility),
		courts:     make(map[string]domain.Court),
	}
}

func (r *fakeRepo) CreateFacility(_ context.Context, f domain.Facility) (domain.Facility, error) {
	r.facilities[f.ID] = f
	return f, nil
}

func (r *fakeRepo) GetFacilityByID(_ context.Context, id string) (domain.Facility, error) {
	f, ok := r.facilities[id]
	if !ok {
		return domain.Facility{}, domain.ErrFacilityNotFound
	}
	return f, nil
}

func (r *fakeRepo) ListFacilities(_ context.Context, nameFilter string) ([]domain.Facility, error) {
	var out []domain.Facility
	for _, f := range r.facilities {
		if nameFilter == "" || f.Name == nameFilter {
			out = append(out, f)
		}
	}
	return out, nil
}

func (r *fakeRepo) AddCourt(_ context.Context, c domain.Court) (domain.Court, error) {
	r.courts[c.ID] = c
	return c, nil
}

func (r *fakeRepo) AddCameraLink(_ context.Context, facilityID string, link domain.CameraLink) (domain.CameraLink, error) {
	f, ok := r.facilities[facilityID]
	if !ok {
		return domain.CameraLink{}, domain.ErrFacilityNotFound
	}
	f.CameraLinks = append(f.CameraLinks, link)
	r.facilities[facilityID] = f
	return link, nil
}

// fakeIDs is a deterministic, dependency-free port.IDGenerator stand-in,
// mirroring internal/socialplay/adapter/grpcapi/authz_regression_test.go's
// fakeIDs.
type fakeIDs struct{ n int }

func (f *fakeIDs) NewID() string {
	f.n++
	return fmt.Sprintf("id-%d", f.n)
}

// newTestHandler wires the real app.Service and the real grpcapi.Handler —
// exactly what cmd/server wires in production — against the in-memory
// fakeRepo above.
func newTestHandler() (*grpcapi.Handler, *fakeRepo) {
	repo := newFakeRepo()
	svc := app.NewService(repo, &fakeIDs{})
	return grpcapi.NewHandler(svc), repo
}

func seedFacility(t *testing.T, h *grpcapi.Handler, ownerID string) *facilitiesv1.Facility {
	t.Helper()
	resp, err := h.CreateFacility(context.Background(), &facilitiesv1.CreateFacilityRequest{
		OwnerId: ownerID,
		Name:    "Riverside Courts",
		Address: "123 Main St",
	})
	if err != nil {
		t.Fatalf("failed to seed fixture facility: %v", err)
	}
	return resp.GetFacility()
}

// --- AddCourt: object-level (BOLA) regression ------------------------------

// TestAddCourt_RejectsMismatchedActor is the ticket's required test:
// create a Facility owned by owner-1, then attempt AddCourt as a
// different actor_user_id, through the real handler -> app -> domain
// path, and assert the request is rejected with the correctly mapped
// status — not a 500, not a silent success.
func TestAddCourt_RejectsMismatchedActor(t *testing.T) {
	ctx := context.Background()
	h, repo := newTestHandler()

	facility := seedFacility(t, h, "owner-1")

	// The BOLA attempt: "attacker", a different actor_user_id than the
	// Facility's owner_id, tries to add a Court to owner-1's Facility.
	_, err := h.AddCourt(ctx, &facilitiesv1.AddCourtRequest{
		FacilityId:  facility.GetId(),
		Name:        "Court 1",
		ActorUserId: "attacker",
	})
	if err == nil {
		t.Fatal("AddCourt(attacker) succeeded silently — a non-owner was able to add a Court to owner-1's facility (BOLA regression)")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("AddCourt(attacker) returned a non-gRPC-status error: %v (a client can't map this to a clean HTTP status)", err)
	}
	if st.Code() == codes.Internal {
		t.Fatalf("AddCourt(attacker) mapped to Internal (500-shaped) — want PermissionDenied (403-shaped): %v", err)
	}
	if st.Code() != codes.PermissionDenied {
		t.Fatalf("AddCourt(attacker) status code = %v, want PermissionDenied (403-shaped)", st.Code())
	}

	// Belt-and-braces, per the ticket's "not a silent success": prove no
	// Court was persisted, not just that an error came back on the wire.
	if len(repo.courts) != 0 {
		t.Errorf("repo.courts has %d entries after a rejected AddCourt, want 0 (the attacker's rejected attempt must not have any side effect)", len(repo.courts))
	}
}

// TestAddCourt_AllowsOwningActor is the symmetric positive-path case:
// without it, TestAddCourt_RejectsMismatchedActor alone couldn't tell "the
// ownership check correctly rejects a mismatched actor" apart from
// "AddCourt is broken and rejects everyone" — this pins down that the
// real owner's AddCourt call still succeeds through the same handler
// path.
func TestAddCourt_AllowsOwningActor(t *testing.T) {
	ctx := context.Background()
	h, repo := newTestHandler()

	facility := seedFacility(t, h, "owner-1")

	resp, err := h.AddCourt(ctx, &facilitiesv1.AddCourtRequest{
		FacilityId:  facility.GetId(),
		Name:        "Court 1",
		ActorUserId: "owner-1",
	})
	if err != nil {
		t.Fatalf("AddCourt(owner-1) (the owner) should succeed, got: %v", err)
	}
	if resp.GetCourt().GetFacilityId() != facility.GetId() {
		t.Errorf("Court.FacilityId = %q, want %q", resp.GetCourt().GetFacilityId(), facility.GetId())
	}
	if len(repo.courts) != 1 {
		t.Errorf("repo.courts has %d entries, want 1 after the owner's successful AddCourt", len(repo.courts))
	}
}

// --- AddCameraLink: object-level (BOLA) regression -------------------------

// TestAddCameraLink_RejectsMismatchedActor mirrors
// TestAddCourt_RejectsMismatchedActor for AddCameraLink, the other write
// RPC T7.3 shipped beyond CreateFacility. Camera consent is attested first
// so this test proves the ownership check specifically, not conflated
// with the pre-existing ErrCameraConsentRequired check (T7.2) — see
// domain.Facility.AddCameraLink's doc comment on check ordering.
func TestAddCameraLink_RejectsMismatchedActor(t *testing.T) {
	ctx := context.Background()
	h, repo := newTestHandler()

	facility := seedFacility(t, h, "owner-1")
	// Attest consent directly against the fake repo's fixture, bypassing
	// the wire API (there is no AttestConsent RPC) — mirrors
	// internal/facilities/app/service_test.go's own fixture setup.
	stored := repo.facilities[facility.GetId()]
	stored.CameraConsentAttested = true
	repo.facilities[facility.GetId()] = stored

	_, err := h.AddCameraLink(ctx, &facilitiesv1.AddCameraLinkRequest{
		FacilityId:  facility.GetId(),
		Url:         "https://example.com/cam1.m3u8",
		ActorUserId: "attacker",
	})
	if err == nil {
		t.Fatal("AddCameraLink(attacker) succeeded silently — a non-owner was able to add a camera link to owner-1's facility (BOLA regression)")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("AddCameraLink(attacker) returned a non-gRPC-status error: %v (a client can't map this to a clean HTTP status)", err)
	}
	if st.Code() == codes.Internal {
		t.Fatalf("AddCameraLink(attacker) mapped to Internal (500-shaped) — want PermissionDenied (403-shaped): %v", err)
	}
	if st.Code() != codes.PermissionDenied {
		t.Fatalf("AddCameraLink(attacker) status code = %v, want PermissionDenied (403-shaped)", st.Code())
	}

	if len(repo.facilities[facility.GetId()].CameraLinks) != 0 {
		t.Errorf("CameraLinks = %v after a rejected AddCameraLink, want empty (the attacker's rejected attempt must not have any side effect)", repo.facilities[facility.GetId()].CameraLinks)
	}
}

// TestAddCameraLink_AllowsOwningActor is the symmetric positive-path case
// for AddCameraLink.
func TestAddCameraLink_AllowsOwningActor(t *testing.T) {
	ctx := context.Background()
	h, repo := newTestHandler()

	facility := seedFacility(t, h, "owner-1")
	stored := repo.facilities[facility.GetId()]
	stored.CameraConsentAttested = true
	repo.facilities[facility.GetId()] = stored

	resp, err := h.AddCameraLink(ctx, &facilitiesv1.AddCameraLinkRequest{
		FacilityId:  facility.GetId(),
		Url:         "https://example.com/cam1.m3u8",
		ActorUserId: "owner-1",
	})
	if err != nil {
		t.Fatalf("AddCameraLink(owner-1) (the owner) should succeed, got: %v", err)
	}
	if len(resp.GetFacility().GetCameraLinks()) != 1 {
		t.Errorf("CameraLinks = %v, want one link", resp.GetFacility().GetCameraLinks())
	}
}
