// End-to-end coverage for ADR-0014's actor-resolution seam applied to
// Facilities (T13.3), and the regression test for issue #154.
//
// **What makes this file different from every other test in this package:**
// it wires the REAL internal/facilities/adapter/identity.Lookup over the REAL
// internal/identity/app.Service, instead of the fakeIdentityLookup the other
// files use. That distinction is the whole point. #154 shipped because every
// Facilities test faked the identity boundary — or, before T13.3, had no
// identity boundary at all — so no test ever carried a real, non-uuid IdP
// subject across the Facilities↔Identity seam, which is the only actor value
// production has carried since T12.7 made the actor auth.RequireSubject(ctx).
//
// The path exercised here is the production one, end to end and unmocked at
// every layer that has an opinion about identifiers:
//
//	auth.Principal{Subject: "auth0|..."}    (a real IdP subject shape)
//	  -> grpcapi.Handler.CreateFacility
//	  -> Handler.actor(ctx)                 (the ADR-0014 seam)
//	  -> app.Service.ResolveActorUserID
//	  -> port.IdentityLookup.UserIDBySubject
//	  -> adapter/identity.Lookup            (REAL)
//	  -> identity/app.Service.UserBySubject (REAL)
//	  -> identity port.Repository           (in-memory, a fake)
//	  -> app.Service.CreateFacility         (receives a uuid, as ADR-0014 rules)
//	  -> the persisted Facility.OwnerID
//
// Only the two repositories are fakes, and that is deliberate rather than
// convenient: the defect class this file exists to catch lives entirely in
// which *identifier space* a value belongs to, and a Postgres round trip
// cannot influence that. What a Postgres round trip WOULD add is the
// mustUUID() panic — which is why the assertion below on OwnerId being
// uuid-shaped is not cosmetic: it is the Docker-free proxy for "this value
// can be written to a uuid column at all". The panic itself is pinned
// directly, one package over, in
// internal/facilities/adapter/postgres/owner_subject_panic_test.go.
package grpcapi_test

import (
	"context"
	"regexp"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/nhuthuynh/white-label/internal/facilities/adapter/grpcapi"
	facilitiesidentity "github.com/nhuthuynh/white-label/internal/facilities/adapter/identity"
	"github.com/nhuthuynh/white-label/internal/facilities/app"
	facilitiesv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/facilities/v1"
	identityapp "github.com/nhuthuynh/white-label/internal/identity/app"
	identitydomain "github.com/nhuthuynh/white-label/internal/identity/domain"
)

// uuidShaped is the canonical 8-4-4-4-12 form pgtype.UUID.Scan accepts, which
// is what "can be stored in facilities.owner_id" means in practice. Kept
// local to the test rather than exported from app: a test that imported the
// production regexp would pass whenever both were wrong together.
var uuidShaped = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// realIdentityRepo is a minimal internal/identity/port.Repository fake. It is
// the only fake between the handler and the User record on the Identity side
// — every layer above it is the real implementation.
type realIdentityRepo struct {
	users map[string]identitydomain.User
}

func newRealIdentityRepo() *realIdentityRepo {
	return &realIdentityRepo{users: make(map[string]identitydomain.User)}
}

func (r *realIdentityRepo) Create(_ context.Context, u identitydomain.User) (identitydomain.User, error) {
	for _, existing := range r.users {
		if existing.Subject == u.Subject {
			return identitydomain.User{}, identitydomain.ErrUserAlreadyExists
		}
	}
	r.users[u.ID] = u
	return u, nil
}

func (r *realIdentityRepo) GetByID(_ context.Context, id string) (identitydomain.User, error) {
	u, ok := r.users[id]
	if !ok {
		return identitydomain.User{}, identitydomain.ErrUserNotFound
	}
	return u, nil
}

func (r *realIdentityRepo) GetBySubject(_ context.Context, subject string) (identitydomain.User, error) {
	for _, u := range r.users {
		if u.Subject == subject {
			return u, nil
		}
	}
	return identitydomain.User{}, identitydomain.ErrUserNotFound
}

func (r *realIdentityRepo) UpdateSelfReportedLevel(_ context.Context, id string, level identitydomain.SelfReportedStartingLevel) (identitydomain.User, error) {
	u, ok := r.users[id]
	if !ok {
		return identitydomain.User{}, identitydomain.ErrUserNotFound
	}
	u.SelfReportedStartingLevel = level
	r.users[id] = u
	return u, nil
}

// identityStubIDs satisfies identity's port.IDGenerator. Users are seeded
// through the repository directly (see seedIdentityUser), so it is never
// called — it exists because identityapp.NewService requires the port.
type identityStubIDs struct{}

func (identityStubIDs) NewID() string { return ownerUserID }

// seedIdentityUser writes a User straight through the repository fake. Going
// through identityapp.Service.CreateUser instead would mint its own id from
// the IDGenerator, and these tests need to control which uuid a given subject
// resolves to in order to assert on it.
func seedIdentityUser(t *testing.T, repo *realIdentityRepo, id, subject string) {
	t.Helper()

	u, err := identitydomain.NewUser(id, subject, "Ada Lovelace",
		[]identitydomain.Role{identitydomain.RolePlayer}, identitydomain.SelfReportedStartingLevel(3))
	if err != nil {
		t.Fatalf("building fixture user (id=%q subject=%q): %v", id, subject, err)
	}
	if _, err := repo.Create(context.Background(), u); err != nil {
		t.Fatalf("seeding identity repo: %v", err)
	}
}

// realIdentityHarness is newTestHandler's twin, with the ONE difference that
// matters: port.IdentityLookup is the real adapter over the real Identity
// service, not fakeIdentityLookup.
type realIdentityHarness struct {
	handler    *grpcapi.Handler
	facilities *fakeRepo
	users      *realIdentityRepo
}

func newHandlerWithRealIdentity(t *testing.T) *realIdentityHarness {
	t.Helper()

	users := newRealIdentityRepo()
	lookup := facilitiesidentity.NewLookup(identityapp.NewService(users, identityStubIDs{}))

	facilities := newFakeRepo()
	svc := app.NewService(facilities, lookup, &fakeIDs{})
	return &realIdentityHarness{handler: grpcapi.NewHandler(svc), facilities: facilities, users: users}
}

// TestCreateFacility_SubjectShapedOwnerResolvesEndToEnd is the #154
// regression test: the RPC that issue reports as fatal for every caller
// alive, driven by a caller shaped exactly like a real one.
//
// Before T13.3 this stored OwnerId = "auth0|owner-1" — the raw subject —
// which is the value that panics the Postgres adapter's mustUUID() and cannot
// be written to `facilities.owner_id uuid NOT NULL` at all. Against a real
// database the process would have gone down; against the fakes it silently
// "succeeded" with a value production can never hold, which is exactly why
// every pre-existing test in this package passed.
//
// The OwnerId assertions are the load-bearing half. Asserting only "no error"
// would still pass if the seam persisted the raw subject.
func TestCreateFacility_SubjectShapedOwnerResolvesEndToEnd(t *testing.T) {
	h := newHandlerWithRealIdentity(t)
	seedIdentityUser(t, h.users, ownerUserID, ownerSubject)

	resp, err := h.handler.CreateFacility(ctxAs(ownerSubject), &facilitiesv1.CreateFacilityRequest{
		Name:    "Riverside Courts",
		Address: "123 Main St",
	})
	if err != nil {
		t.Fatalf("CreateFacility as a subject-shaped actor (%q) = %v, want success — "+
			"this is the #154 defect: a verified IdP subject must resolve to the "+
			"caller's User.ID at the grpcapi boundary (ADR-0014)", ownerSubject, err)
	}

	got := resp.GetFacility().GetOwnerId()
	if got == ownerSubject {
		t.Fatalf("Facility.OwnerId = %q — the raw subject was persisted. "+
			"facilities.owner_id is `uuid NOT NULL` and the Postgres adapter converts "+
			"it with a mustUUID() that PANICS on this value (#154).", got)
	}
	if !uuidShaped.MatchString(got) {
		t.Fatalf("Facility.OwnerId = %q, want a uuid — anything else cannot be written "+
			"to facilities.owner_id", got)
	}
	if got != ownerUserID {
		t.Fatalf("Facility.OwnerId = %q, want %q (the resolved User.ID for subject %q)",
			got, ownerUserID, ownerSubject)
	}

	// And the same value reached persistence, not just the response.
	stored, ok := h.facilities.facilities[resp.GetFacility().GetId()]
	if !ok {
		t.Fatal("the Facility was not persisted")
	}
	if stored.OwnerID != ownerUserID {
		t.Fatalf("persisted OwnerID = %q, want %q", stored.OwnerID, ownerUserID)
	}
}

// TestCreateFacility_ThenOwnerCanActOnIt is what makes #154 "fully resolved"
// rather than "half resolved", and it is the case the ticket's instruction 6
// asks to check rather than assume.
//
// AddCourt, AddCameraLink and AttestCameraConsent do not write owner_id; they
// compare the actor against the stored one via domain.Facility.EnsureOwner. A
// seam that resolved on the write path but not on the read path would pass
// the test above and still lock every owner out of their own Facility —
// storing a uuid and then comparing it against a subject. This drives all
// three through the same real seam, as the same caller.
func TestCreateFacility_ThenOwnerCanActOnIt(t *testing.T) {
	h := newHandlerWithRealIdentity(t)
	seedIdentityUser(t, h.users, ownerUserID, ownerSubject)

	created, err := h.handler.CreateFacility(ctxAs(ownerSubject), &facilitiesv1.CreateFacilityRequest{
		Name: "Riverside Courts", Address: "123 Main St",
	})
	if err != nil {
		t.Fatalf("CreateFacility: %v", err)
	}
	facilityID := created.GetFacility().GetId()
	ownerCtx := ctxAs(ownerSubject)

	if _, err := h.handler.AddCourt(ownerCtx, &facilitiesv1.AddCourtRequest{
		FacilityId: facilityID, Name: "Court 1",
	}); err != nil {
		t.Fatalf("AddCourt as the Facility's own creator = %v, want success — the write "+
			"path and the ownership check disagree about the actor's identifier space", err)
	}

	if _, err := h.handler.AttestCameraConsent(ownerCtx, &facilitiesv1.AttestCameraConsentRequest{
		FacilityId: facilityID,
	}); err != nil {
		t.Fatalf("AttestCameraConsent as the Facility's own creator = %v, want success", err)
	}

	if _, err := h.handler.AddCameraLink(ownerCtx, &facilitiesv1.AddCameraLinkRequest{
		FacilityId: facilityID, Url: "https://example.com/cam1.m3u8",
	}); err != nil {
		t.Fatalf("AddCameraLink as the Facility's own creator = %v, want success", err)
	}
}

// TestCreateFacility_StillRejectsANonOwnerThroughTheRealSeam proves the seam
// resolves identity WITHOUT weakening authorization — that ADR-0014 answers
// "who are you", never "what may you do".
//
// The actor here is a real, registered, resolvable User who simply does not
// own the Facility. Resolution succeeds; the ownership check still refuses.
// Without this case, a seam that resolved every subject to the same User
// would pass the two tests above and hand every caller ownership of every
// Facility.
func TestCreateFacility_StillRejectsANonOwnerThroughTheRealSeam(t *testing.T) {
	h := newHandlerWithRealIdentity(t)
	seedIdentityUser(t, h.users, ownerUserID, ownerSubject)
	seedIdentityUser(t, h.users, attackerUserID, attackerSubject)

	created, err := h.handler.CreateFacility(ctxAs(ownerSubject), &facilitiesv1.CreateFacilityRequest{
		Name: "Riverside Courts", Address: "123 Main St",
	})
	if err != nil {
		t.Fatalf("CreateFacility: %v", err)
	}

	_, err = h.handler.AddCourt(ctxAs(attackerSubject), &facilitiesv1.AddCourtRequest{
		FacilityId: created.GetFacility().GetId(), Name: "Court 1",
	})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("AddCourt(resolvable non-owner) = %v (code %v), want PermissionDenied",
			err, status.Code(err))
	}
	if len(h.facilities.courts) != 0 {
		t.Fatalf("courts = %d, want 0 — a rejected call must have no side effect",
			len(h.facilities.courts))
	}
}

// TestCreateFacility_UnregisteredSubjectIsPermissionDenied pins ADR-0014 §6's
// answer to the error case: a caller whose token verified but who has no User
// row.
//
// PermissionDenied, not NotFound and not Unauthenticated:
//   - Unauthenticated is wrong by ADR-0013 §5 — we know exactly who they are,
//     their token verified.
//   - NotFound would make a 404 ambiguous between "the Facility you named does
//     not exist" and "you do not exist", and turn every actor-taking RPC on
//     this service into a user-enumeration oracle.
func TestCreateFacility_UnregisteredSubjectIsPermissionDenied(t *testing.T) {
	h := newHandlerWithRealIdentity(t)
	// Deliberately seed nobody: the subject verifies, but no User is registered.

	_, err := h.handler.CreateFacility(ctxAs("auth0|never-registered"), &facilitiesv1.CreateFacilityRequest{
		Name: "Ghost Courts", Address: "1 Nowhere",
	})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("CreateFacility(unregistered subject) = %v (code %v), want PermissionDenied",
			err, status.Code(err))
	}
	if n := len(h.facilities.facilities); n != 0 {
		t.Fatalf("facilities = %d, want 0 — an unresolvable actor must never reach the repository", n)
	}
}

// TestActorSeam_RawSubjectNeverReachesTheAppLayer is the structural assertion
// behind ADR-0014's ruling, and the one that would catch a future RPC wired
// past the seam.
//
// It states the invariant directly: below grpcapi, no actor value is ever a
// subject. Passing the raw subject into app.Service is therefore a caller
// error and stays rejected — deliberately NOT "fixed" — because that guard
// failing closed is what stands between a bypassed seam and a panicking
// Postgres adapter.
func TestActorSeam_RawSubjectNeverReachesTheAppLayer(t *testing.T) {
	h := newHandlerWithRealIdentity(t)
	seedIdentityUser(t, h.users, ownerUserID, ownerSubject)

	svc := app.NewService(h.facilities,
		facilitiesidentity.NewLookup(identityapp.NewService(h.users, identityStubIDs{})),
		&fakeIDs{})

	// The seam, called directly: subject in, User.ID out.
	resolved, err := svc.ResolveActorUserID(context.Background(), ownerSubject)
	if err != nil {
		t.Fatalf("ResolveActorUserID(%q) = %v, want the caller's User.ID", ownerSubject, err)
	}
	if resolved != ownerUserID {
		t.Fatalf("ResolveActorUserID(%q) = %q, want %q", ownerSubject, resolved, ownerUserID)
	}

	// The raw subject, handed to the app layer as an actor: refused, not stored.
	_, err = svc.CreateFacility(context.Background(), app.CreateFacilityInput{
		OwnerID: ownerSubject, Name: "Bypassed Courts", Address: "1 Main St",
	})
	if err == nil {
		t.Fatal("CreateFacility(OwnerID = a raw subject) succeeded — the app layer's " +
			"uuid guard is what makes a bypassed seam fail closed instead of panicking " +
			"the Postgres adapter (#154)")
	}
}
