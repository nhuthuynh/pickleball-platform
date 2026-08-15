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
	// courtOrder tracks insertion order — see
	// internal/facilities/app/service_test.go's inMemoryRepo.courtOrder for
	// why (Go map iteration order is unspecified).
	courtOrder []string
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

// GetFacilityByID merges in this Facility's Courts (T8.2), mirroring the
// Postgres adapter's GetFacilityByID.
func (r *fakeRepo) GetFacilityByID(_ context.Context, id string) (domain.Facility, error) {
	f, ok := r.facilities[id]
	if !ok {
		return domain.Facility{}, domain.ErrFacilityNotFound
	}
	f.Courts = r.courtsForFacility(id)
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

// AddCourt does not model courts.facility_id's FK
// (db/migrations/0010_facilities.sql), unlike AddCameraLink below — checked
// while investigating T17.4 (issue #195), and deliberately left as-is rather
// than "fixed" for consistency. app.Service.AddCourt always calls
// GetFacilityByID as its own object-level-ownership guard before ever
// reaching this method (app/service.go), and every test in this package
// drives AddCourt through the real Handler/app.Service, never this fake
// directly — so an unknown FacilityID here is unreachable in practice: the
// guard answers domain.ErrFacilityNotFound first every time, and adding an
// FK check that a mutation check then proves no test can ever fail would be
// the same "no production code without a test that demanded it" violation
// CLAUDE.md rule 1 names, just committed inside a fixture instead of
// production code.
func (r *fakeRepo) AddCourt(_ context.Context, c domain.Court) (domain.Court, error) {
	r.courts[c.ID] = c
	r.courtOrder = append(r.courtOrder, c.ID)
	return c, nil
}

// ListCourtsForFacility is T8.2's read path fake, mirroring
// internal/facilities/adapter/postgres.Repository.ListCourtsForFacility.
// GetCourtByID is T11.2's Court read (the one Booking's port.FacilityLookup
// needs); this handler package never calls it, but port.Repository requires it.
func (r *fakeRepo) GetCourtByID(_ context.Context, courtID string) (domain.Court, error) {
	c, ok := r.courts[courtID]
	if !ok {
		return domain.Court{}, domain.ErrCourtNotFound
	}
	return c, nil
}

func (r *fakeRepo) ListCourtsForFacility(_ context.Context, facilityID string) ([]domain.Court, error) {
	return r.courtsForFacility(facilityID), nil
}

func (r *fakeRepo) courtsForFacility(facilityID string) []domain.Court {
	out := make([]domain.Court, 0)
	for _, id := range r.courtOrder {
		if c := r.courts[id]; c.FacilityID == facilityID {
			out = append(out, c)
		}
	}
	return out
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

func (r *fakeRepo) AttestCameraConsent(_ context.Context, facilityID string) error {
	f, ok := r.facilities[facilityID]
	if !ok {
		return domain.ErrFacilityNotFound
	}
	f.CameraConsentAttested = true
	r.facilities[facilityID] = f
	return nil
}

// fakeIDs is a deterministic, dependency-free port.IDGenerator stand-in,
// mirroring internal/socialplay/adapter/grpcapi/authz_regression_test.go's
// fakeIDs.
type fakeIDs struct{ n int }

// NewID mints deterministic but *UUID-shaped* IDs. It used to return "id-1",
// which no real ID generator produces and no real Postgres adapter can store —
// see the same note on internal/facilities/app's sequentialIDs.
func (f *fakeIDs) NewID() string {
	f.n++
	return fmt.Sprintf("00000000-0000-4000-8000-%012d", f.n)
}

// --- port.IdentityLookup fake -----------------------------------------

// userUUID mints the server-side identifier space's fixture values: a
// resolved identity_users.id, which facilities.owner_id (`uuid NOT NULL`)
// can actually hold. subjectOf's values are the other space — see
// fakeIdentityLookup.
func userUUID(n int) string { return fmt.Sprintf("00000000-0000-4000-b000-%012d", n) }

var (
	ownerUserID    = userUUID(1)
	attackerUserID = userUUID(2)
)

// fakeIdentityLookup stands in for internal/facilities/adapter/identity,
// returning the same context-local sentinel that adapter translates
// Identity's ErrUserNotFound into (T13.3).
//
// It models BOTH identifier spaces ADR-0014 distinguishes, because a fake
// that collapsed them would reintroduce the exact blind spot #154 came
// through: `subjects` maps a verified IdP subject to the User.ID this
// platform stores, and those two strings are never equal. Before this ticket
// the tests in this package used "owner-1" as the subject AND as the stored
// owner_id, which is why every one of them passed against a handler that
// wrote a subject into a uuid column.
//
// The genuinely unmocked end-to-end coverage of this seam is
// subject_owner_seam_test.go, which drives the real adapter over the real
// Identity service. This fake exists so the authorization cases in this
// package do not each need an Identity fixture.
type fakeIdentityLookup struct {
	subjects map[string]string
}

func newFakeIdentityLookup() *fakeIdentityLookup {
	return &fakeIdentityLookup{subjects: map[string]string{
		ownerSubject:    ownerUserID,
		attackerSubject: attackerUserID,
	}}
}

// UserIDBySubject is ADR-0014's translation, faked: the subject the token
// carries in, the User.ID this platform stores out. An unregistered subject
// is domain.ErrUserNotFound, which the handler maps to PermissionDenied.
func (l *fakeIdentityLookup) UserIDBySubject(_ context.Context, subject string) (string, error) {
	id, ok := l.subjects[subject]
	if !ok {
		return "", domain.ErrUserNotFound
	}
	return id, nil
}

// newTestHandler wires the real app.Service and the real grpcapi.Handler —
// exactly what cmd/server wires in production — against the in-memory
// fakeRepo above.
func newTestHandler() (*grpcapi.Handler, *fakeRepo) {
	repo := newFakeRepo()
	svc := app.NewService(repo, newFakeIdentityLookup(), &fakeIDs{})
	return grpcapi.NewHandler(svc), repo
}

// seedFacility creates a Facility owned by the caller authenticated as
// subject. Note the parameter is a SUBJECT, not an owner id: as of T13.3 the
// handler resolves it to a User.ID and stores that, so the value this
// function is given and the value that lands in owner_id are deliberately
// different strings (ADR-0014).
func seedFacility(t *testing.T, h *grpcapi.Handler, subject string) *facilitiesv1.Facility {
	t.Helper()
	// T12.7: the owner is minted from the verified principal, so the fixture
	// authenticates as the owner rather than declaring them in the request body.
	resp, err := h.CreateFacility(ctxAs(subject), &facilitiesv1.CreateFacilityRequest{
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
	h, repo := newTestHandler()

	facility := seedFacility(t, h, ownerSubject)

	// The BOLA attempt: attackerSubject, a verified caller who is not the
	// Facility's owner, tries to add a Court to owner-1's Facility. T12.7
	// moved this identity from an actor_user_id request field to the
	// verified principal — see principal_authz_test.go for the case that
	// proves the request field is now ignored outright.
	ctx := ctxAs(attackerSubject)
	_, err := h.AddCourt(ctx, &facilitiesv1.AddCourtRequest{
		FacilityId: facility.GetId(),
		Name:       "Court 1",
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
	h, repo := newTestHandler()

	facility := seedFacility(t, h, ownerSubject)

	ctx := ctxAs(ownerSubject)
	resp, err := h.AddCourt(ctx, &facilitiesv1.AddCourtRequest{
		FacilityId: facility.GetId(),
		Name:       "Court 1",
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
	h, repo := newTestHandler()

	facility := seedFacility(t, h, ownerSubject)
	ctx := ctxAs(attackerSubject)
	// Attest consent directly against the fake repo's fixture, bypassing
	// the wire API (there is no AttestConsent RPC) — mirrors
	// internal/facilities/app/service_test.go's own fixture setup.
	stored := repo.facilities[facility.GetId()]
	stored.CameraConsentAttested = true
	repo.facilities[facility.GetId()] = stored

	_, err := h.AddCameraLink(ctx, &facilitiesv1.AddCameraLinkRequest{
		FacilityId: facility.GetId(),
		Url:        "https://example.com/cam1.m3u8",
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
	h, repo := newTestHandler()

	facility := seedFacility(t, h, ownerSubject)
	stored := repo.facilities[facility.GetId()]
	stored.CameraConsentAttested = true
	repo.facilities[facility.GetId()] = stored

	ctx := ctxAs(ownerSubject)
	resp, err := h.AddCameraLink(ctx, &facilitiesv1.AddCameraLinkRequest{
		FacilityId: facility.GetId(),
		Url:        "https://example.com/cam1.m3u8",
	})
	if err != nil {
		t.Fatalf("AddCameraLink(owner-1) (the owner) should succeed, got: %v", err)
	}
	if len(resp.GetFacility().GetCameraLinks()) != 1 {
		t.Errorf("CameraLinks = %v, want one link", resp.GetFacility().GetCameraLinks())
	}
}

// --- AttestCameraConsent: object-level (BOLA) regression -------------------
//
// T8.4's required regression test: mirrors TestAddCourt_RejectsMismatchedActor
// and TestAddCameraLink_RejectsMismatchedActor exactly, for the new
// AttestCameraConsent RPC. Verified per CLAUDE.md rule 10 / T7.7's own
// verification pattern: temporarily commented out the EnsureOwner call
// inside domain.Facility.AttestCameraConsent, confirmed
// TestAttestCameraConsent_RejectsMismatchedActor failed (the attacker's
// call succeeded and CameraConsentAttested flipped to true), then restored
// the check and confirmed the test passes again.

// TestAttestCameraConsent_RejectsMismatchedActor is the ticket's required
// test: create a Facility owned by owner-1, then attempt
// AttestCameraConsent as a different actor_user_id, through the real
// handler -> app -> domain path, and assert the request is rejected with
// the correctly mapped status — not a 500, not a silent success — and that
// CameraConsentAttested is left untouched.
func TestAttestCameraConsent_RejectsMismatchedActor(t *testing.T) {
	h, repo := newTestHandler()

	facility := seedFacility(t, h, ownerSubject)

	ctx := ctxAs(attackerSubject)
	_, err := h.AttestCameraConsent(ctx, &facilitiesv1.AttestCameraConsentRequest{
		FacilityId: facility.GetId(),
	})
	if err == nil {
		t.Fatal("AttestCameraConsent(attacker) succeeded silently — a non-owner was able to attest consent on owner-1's facility (BOLA regression)")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("AttestCameraConsent(attacker) returned a non-gRPC-status error: %v (a client can't map this to a clean HTTP status)", err)
	}
	if st.Code() == codes.Internal {
		t.Fatalf("AttestCameraConsent(attacker) mapped to Internal (500-shaped) — want PermissionDenied (403-shaped): %v", err)
	}
	if st.Code() != codes.PermissionDenied {
		t.Fatalf("AttestCameraConsent(attacker) status code = %v, want PermissionDenied (403-shaped)", st.Code())
	}

	// Belt-and-braces, per the ticket's "not a silent success": prove
	// CameraConsentAttested is still false, not just that an error came
	// back on the wire.
	if repo.facilities[facility.GetId()].CameraConsentAttested {
		t.Error("CameraConsentAttested = true after a rejected AttestCameraConsent, want false (the attacker's rejected attempt must not have any side effect)")
	}
}

// TestAttestCameraConsent_AllowsOwningActor is the symmetric positive-path
// case: without it, TestAttestCameraConsent_RejectsMismatchedActor alone
// couldn't tell "the ownership check correctly rejects a mismatched actor"
// apart from "AttestCameraConsent is broken and rejects everyone."
func TestAttestCameraConsent_AllowsOwningActor(t *testing.T) {
	h, repo := newTestHandler()

	facility := seedFacility(t, h, ownerSubject)

	ctx := ctxAs(ownerSubject)
	resp, err := h.AttestCameraConsent(ctx, &facilitiesv1.AttestCameraConsentRequest{
		FacilityId: facility.GetId(),
	})
	if err != nil {
		t.Fatalf("AttestCameraConsent(owner-1) (the owner) should succeed, got: %v", err)
	}
	if !resp.GetFacility().GetCameraConsentAttested() {
		t.Error("CameraConsentAttested = false in response, want true")
	}
	if !repo.facilities[facility.GetId()].CameraConsentAttested {
		t.Error("CameraConsentAttested = false in repo after the owner's successful AttestCameraConsent, want true")
	}
}

// TestAttestCameraConsent_ThenAddCameraLink_EndToEnd is the ticket's
// headline acceptance criterion, exercised through the full real handler
// stack: attesting consent as the owner, then adding a camera link, both
// succeed — proving the previously-dead-ended AddCameraLink flow is now
// reachable end-to-end by a real user for the first time.
func TestAttestCameraConsent_ThenAddCameraLink_EndToEnd(t *testing.T) {
	h, _ := newTestHandler()

	facility := seedFacility(t, h, ownerSubject)

	ctx := ctxAs(ownerSubject)

	if _, err := h.AttestCameraConsent(ctx, &facilitiesv1.AttestCameraConsentRequest{
		FacilityId: facility.GetId(),
	}); err != nil {
		t.Fatalf("AttestCameraConsent: unexpected err: %v", err)
	}

	resp, err := h.AddCameraLink(ctx, &facilitiesv1.AddCameraLinkRequest{
		FacilityId: facility.GetId(),
		Url:        "https://example.com/cam1.m3u8",
	})
	if err != nil {
		t.Fatalf("AddCameraLink after AttestCameraConsent should succeed, got: %v", err)
	}
	if len(resp.GetFacility().GetCameraLinks()) != 1 {
		t.Errorf("CameraLinks = %v, want one link", resp.GetFacility().GetCameraLinks())
	}
}
