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

			svc := app.NewService(newInMemoryRepo(), &sequentialIDs{})

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
	svc := app.NewService(repo, &sequentialIDs{})

	created, err := svc.CreateFacility(context.Background(), app.CreateFacilityInput{
		OwnerID: "owner-1",
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
