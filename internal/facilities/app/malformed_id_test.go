// Boundary validation for caller-supplied Facility IDs.
//
// Same class of bug as internal/competitions/app/malformed_id_test.go documents
// in full: GetFacility took an unvalidated HTTP path parameter and handed it to
// a Postgres adapter whose mustUUID panics on a non-UUID, and grpc installs no
// recover() of its own, so the panic killed the server process. The guard makes
// a malformed ID resolve to the same domain.ErrFacilityNotFound an unknown ID
// already produces — a 404, never a 500.
package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nhuthuynh/white-label/internal/facilities/app"
	"github.com/nhuthuynh/white-label/internal/facilities/domain"
)

// malformedFacilityIDs mirrors the competitions corpus. The braced and urn:
// entries are the ones that matter: github.com/google/uuid's Validate accepts
// both, pgtype.UUID.Scan rejects both, so a guard built on uuid.Validate would
// still have panicked on them.
var malformedFacilityIDs = []string{
	"",
	"not-a-uuid",
	"0",
	"'; DROP TABLE facilities;--",
	"../../etc/passwd",
	"6ba7b810-9dad-11d1-80b4-00c04fd430c\x00",
	"6ba7b810-9dad-11d1-80b4-00c04fd430c",
	"zzzzzzzz-9dad-11d1-80b4-00c04fd430c8",
	"{6ba7b810-9dad-11d1-80b4-00c04fd430c8}",
	"urn:uuid:6ba7b810-9dad-11d1-80b4-00c04fd430c8",
	" 6ba7b810-9dad-11d1-80b4-00c04fd430c8 ",
}

func TestGetFacility_MalformedIDIsNotFound(t *testing.T) {
	t.Parallel()

	for _, id := range malformedFacilityIDs {
		t.Run(id, func(t *testing.T) {
			t.Parallel()

			svc := app.NewService(newInMemoryRepo(), stubIdentity{}, &sequentialIDs{})

			_, err := svc.GetFacility(context.Background(), id)
			if !errors.Is(err, domain.ErrFacilityNotFound) {
				t.Fatalf("GetFacility(%q) error = %v, want %v", id, err, domain.ErrFacilityNotFound)
			}
		})
	}
}

// TestGetFacility_WellFormedIDStillResolves is the too-strict guard rail: a
// validator that rejects real IDs turns a loud crash into a silent 404 on every
// Facility, which is worse.
func TestGetFacility_WellFormedIDStillResolves(t *testing.T) {
	t.Parallel()

	repo := newInMemoryRepo()
	svc := app.NewService(repo, stubIdentity{}, &sequentialIDs{})

	created, err := svc.CreateFacility(context.Background(), app.CreateFacilityInput{
		OwnerID: ownerUserID,
		Name:    "Riverside Courts",
		Address: "1 River Rd",
	})
	if err != nil {
		t.Fatalf("CreateFacility: %v", err)
	}

	got, err := svc.GetFacility(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetFacility(%q) on a real, freshly-created Facility: %v", created.ID, err)
	}
	if got.ID != created.ID {
		t.Fatalf("GetFacility returned %q, want %q", got.ID, created.ID)
	}
}

// --- T10.7: AddCourt's own malformed FacilityID guard (closes #97) ---------
//
// AddCourt calls Repository.GetFacilityByID(facilityID) first, before
// EnsureOwner, and already returns the bare domain.ErrFacilityNotFound for
// an unknown-but-well-formed id (see TestAddCourt_RejectsNonOwner's sibling
// tests above and TestAttestCameraConsent_UnknownFacilityReturnsNotFound for
// the same pattern on a different method) — a malformed id must answer
// identically rather than reaching the Postgres adapter's mustUUID, which
// panics on non-UUID input. Unlike TestGetFacility_MalformedIDIsNotFound
// above (a return-value-only assertion, pre-dating this ticket), these tests
// use inMemoryRepo.getFacilityByIDCalls to assert the call never happens at
// all — the property that actually matters, per docs/process/t9-retro.md
// finding 3 and this ticket's own instructions.
func TestAddCourt_MalformedFacilityIDIsNotFoundAndNeverReachesRepository(t *testing.T) {
	t.Parallel()

	for _, id := range malformedFacilityIDs {
		t.Run(id, func(t *testing.T) {
			t.Parallel()

			repo := newInMemoryRepo()
			svc := app.NewService(repo, stubIdentity{}, &sequentialIDs{})

			_, err := svc.AddCourt(context.Background(), id, ownerUserID, "Court 1")
			if !errors.Is(err, domain.ErrFacilityNotFound) {
				t.Fatalf("AddCourt(%q) error = %v, want %v", id, err, domain.ErrFacilityNotFound)
			}
			if calls := repo.getFacilityByIDCalls.Load(); calls != 0 {
				t.Errorf("malformed FacilityID %q reached the repository (%d calls); it must be rejected at the boundary", id, calls)
			}
		})
	}
}

// TestAddCourt_WellFormedUnknownFacilityIDStillReachesRepository is the
// too-strict guard rail: a well-formed but unknown FacilityID must still
// reach the repository and get the repository's own ErrFacilityNotFound, or
// every real Facility's AddCourt would silently fail.
func TestAddCourt_WellFormedUnknownFacilityIDStillReachesRepository(t *testing.T) {
	t.Parallel()

	repo := newInMemoryRepo()
	svc := app.NewService(repo, stubIdentity{}, &sequentialIDs{})

	unknown := "6ba7b810-9dad-11d1-80b4-00c04fd430c8"
	_, err := svc.AddCourt(context.Background(), unknown, ownerUserID, "Court 1")
	if !errors.Is(err, domain.ErrFacilityNotFound) {
		t.Fatalf("AddCourt(%q) error = %v, want %v", unknown, err, domain.ErrFacilityNotFound)
	}
	if calls := repo.getFacilityByIDCalls.Load(); calls != 1 {
		t.Fatalf("well-formed unknown FacilityID did not reach the repository (%d calls)", calls)
	}
}

// TestAddCameraLinkAndAttestCameraConsent_MalformedFacilityIDNeverReachesRepository
// covers the other two write handlers this ticket's required inspection
// sweep found taking a caller-supplied FacilityID with the identical
// unguarded GetFacilityByID-first shape AddCourt had: AddCameraLink and
// AttestCameraConsent. Both already return the bare domain.ErrFacilityNotFound
// for an unknown-but-well-formed id (TestAddCameraLink_UnknownFacilityReturnsNotFound
// and TestAttestCameraConsent_UnknownFacilityReturnsNotFound in
// service_test.go pin that today).
func TestAddCameraLinkAndAttestCameraConsent_MalformedFacilityIDNeverReachesRepository(t *testing.T) {
	t.Parallel()

	for _, id := range malformedFacilityIDs {
		t.Run("AddCameraLink/"+id, func(t *testing.T) {
			t.Parallel()

			repo := newInMemoryRepo()
			svc := app.NewService(repo, stubIdentity{}, &sequentialIDs{})

			_, err := svc.AddCameraLink(context.Background(), id, ownerUserID, "https://example.com/cam1.m3u8")
			if !errors.Is(err, domain.ErrFacilityNotFound) {
				t.Fatalf("AddCameraLink(%q) error = %v, want %v", id, err, domain.ErrFacilityNotFound)
			}
			if calls := repo.getFacilityByIDCalls.Load(); calls != 0 {
				t.Errorf("malformed FacilityID %q reached the repository (%d calls); it must be rejected at the boundary", id, calls)
			}
		})

		t.Run("AttestCameraConsent/"+id, func(t *testing.T) {
			t.Parallel()

			repo := newInMemoryRepo()
			svc := app.NewService(repo, stubIdentity{}, &sequentialIDs{})

			_, err := svc.AttestCameraConsent(context.Background(), id, ownerUserID)
			if !errors.Is(err, domain.ErrFacilityNotFound) {
				t.Fatalf("AttestCameraConsent(%q) error = %v, want %v", id, err, domain.ErrFacilityNotFound)
			}
			if calls := repo.getFacilityByIDCalls.Load(); calls != 0 {
				t.Errorf("malformed FacilityID %q reached the repository (%d calls); it must be rejected at the boundary", id, calls)
			}
		})
	}
}

// TestAddCameraLinkAndAttestCameraConsent_WellFormedUnknownFacilityIDStillReachesRepository
// is the too-strict guard rail for both, mirroring
// TestAddCourt_WellFormedUnknownFacilityIDStillReachesRepository above.
func TestAddCameraLinkAndAttestCameraConsent_WellFormedUnknownFacilityIDStillReachesRepository(t *testing.T) {
	t.Parallel()

	unknown := "6ba7b810-9dad-11d1-80b4-00c04fd430c8"

	t.Run("AddCameraLink", func(t *testing.T) {
		t.Parallel()
		repo := newInMemoryRepo()
		svc := app.NewService(repo, stubIdentity{}, &sequentialIDs{})

		_, err := svc.AddCameraLink(context.Background(), unknown, ownerUserID, "https://example.com/cam1.m3u8")
		if !errors.Is(err, domain.ErrFacilityNotFound) {
			t.Fatalf("AddCameraLink(%q) error = %v, want %v", unknown, err, domain.ErrFacilityNotFound)
		}
		if calls := repo.getFacilityByIDCalls.Load(); calls != 1 {
			t.Fatalf("well-formed unknown FacilityID did not reach the repository (%d calls)", calls)
		}
	})

	t.Run("AttestCameraConsent", func(t *testing.T) {
		t.Parallel()
		repo := newInMemoryRepo()
		svc := app.NewService(repo, stubIdentity{}, &sequentialIDs{})

		_, err := svc.AttestCameraConsent(context.Background(), unknown, ownerUserID)
		if !errors.Is(err, domain.ErrFacilityNotFound) {
			t.Fatalf("AttestCameraConsent(%q) error = %v, want %v", unknown, err, domain.ErrFacilityNotFound)
		}
		if calls := repo.getFacilityByIDCalls.Load(); calls != 1 {
			t.Fatalf("well-formed unknown FacilityID did not reach the repository (%d calls)", calls)
		}
	})
}
