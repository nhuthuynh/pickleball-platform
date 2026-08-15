package app_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/nhuthuynh/white-label/internal/facilities/app"
	"github.com/nhuthuynh/white-label/internal/facilities/domain"
)

// --- actor fixtures ------------------------------------------------------
//
// UUID-shaped, and that is the point rather than a detail. Until T13.3 these
// tests used "owner-1" and "attacker", which are values production can never
// hold: facilities.owner_id is `uuid NOT NULL`
// (db/migrations/0010_facilities.sql) and the Postgres adapter converts it
// with a mustUUID() that panics on anything else.
//
// This is the identical move sequentialIDs' own doc comment records making
// for Facility/Court IDs — "It used to return \"id-1\", \"id-2\", ... That was
// a fixture lying about the shape production uses" — applied to the actor
// field, which ADR-0014 has now placed in the same identifier space. A fixture
// that lies about the actor's shape is what let #154 sit undetected through
// T12.7's whole migration: every app-level test passed, because none of them
// used a value the real column could reject.
func userUUID(n int) string { return fmt.Sprintf("00000000-0000-4000-b000-%012d", n) }

var (
	ownerUserID    = userUUID(1)
	attackerUserID = userUUID(2)
)

// stubIdentity is a port.IdentityLookup fake for the app-layer tests, which
// exercise methods *below* the resolution seam and therefore never resolve a
// subject at all — every one of them is handed an already-resolved User.ID,
// exactly as ADR-0014 says the app layer always is.
//
// It returns ErrUserNotFound unconditionally, deliberately. These tests must
// not depend on resolution succeeding, and a stub that resolved anything
// would quietly let a future app method start calling the seam without any
// test noticing. If one ever does, it fails here, loudly.
//
// The genuinely unmocked coverage of the seam lives two packages over, in
// internal/facilities/adapter/identity (the adapter over the real
// identityapp.Service) and internal/facilities/adapter/grpcapi's
// subject_owner_seam_test.go (the full end-to-end path).
type stubIdentity struct{}

func (stubIdentity) UserIDBySubject(_ context.Context, _ string) (string, error) {
	return "", domain.ErrUserNotFound
}

// TestCreateFacility_RejectsANonUUIDOwner is the app-layer half of #154's
// regression coverage: the fail-closed guard that makes ADR-0014's invariant
// ("below the grpcapi boundary an actor is always a User.ID") enforced rather
// than merely intended.
//
// The values below are the two ways the invariant could be broken — a real
// IdP subject, and an arbitrary caller-shaped string — and both must be
// refused *before* the repository is reached, because the repository's
// mustUUID() answers them with a panic rather than an error (pinned in
// internal/facilities/adapter/postgres/owner_subject_panic_test.go).
//
// ErrUserNotFound, which grpcapi maps to PermissionDenied per ADR-0014 §6.
func TestCreateFacility_RejectsANonUUIDOwner(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		ownerID string
	}{
		{name: "a real IdP subject", ownerID: "auth0|abc123"},
		{name: "a google subject", ownerID: "google-oauth2|10769150350006150715"},
		{name: "an arbitrary string", ownerID: "owner-1"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			repo := newInMemoryRepo()
			svc := app.NewService(repo, stubIdentity{}, &sequentialIDs{})

			_, err := svc.CreateFacility(context.Background(), app.CreateFacilityInput{
				OwnerID: tc.ownerID, Name: "Riverside Courts", Address: "123 Main St",
			})
			if !errors.Is(err, domain.ErrUserNotFound) {
				t.Fatalf("CreateFacility(OwnerID=%q) = %v, want ErrUserNotFound — an actor "+
					"value that is not a User.ID must fail closed here, not panic the "+
					"Postgres adapter's mustUUID() (#154)", tc.ownerID, err)
			}
			if n := len(repo.facilities); n != 0 {
				t.Fatalf("repo holds %d facilities, want 0 — a rejected CreateFacility "+
					"must never reach the repository", n)
			}
		})
	}
}

// TestResolveActorUserID_DelegatesToTheIdentityPort pins that the seam method
// exists on this service and goes through port.IdentityLookup rather than
// re-deriving an actor some other way.
//
// It asserts the unresolvable direction because stubIdentity resolves nothing;
// the resolving direction is proven against the real identityapp.Service in
// internal/facilities/adapter/identity/lookup_test.go, which is where it
// belongs — a fake resolving a subject would only prove the fake works.
func TestResolveActorUserID_DelegatesToTheIdentityPort(t *testing.T) {
	t.Parallel()

	svc := app.NewService(newInMemoryRepo(), stubIdentity{}, &sequentialIDs{})

	_, err := svc.ResolveActorUserID(context.Background(), "auth0|never-registered")
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("ResolveActorUserID(unregistered) = %v, want ErrUserNotFound", err)
	}
}
