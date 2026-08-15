package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nhuthuynh/white-label/internal/facilities/app"
	"github.com/nhuthuynh/white-label/internal/facilities/domain"
)

// TestGetCourt is T11.2's read path into the Facilities context: Booking's
// new port.FacilityLookup has to answer "which Facility owns this Court?" so
// GetQuote can resolve a facility-scoped DiscountRule from the CourtID it
// already has. Courts belong to Facilities, so that read lives here rather
// than Booking reaching into the courts table behind the context boundary.
//
// A malformed courtID is answered exactly like an unknown one
// (domain.ErrCourtNotFound), matching GetFacility/AddCourt's own T10.7 guard
// on the same class of caller-supplied ID — and proved by the repository
// never being called, since the in-memory fake cannot reproduce Postgres
// rejecting a non-UUID against a `uuid` column.
func TestGetCourt(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := newInMemoryRepo()
	svc := app.NewService(repo, stubIdentity{}, &sequentialIDs{})

	owner := seqUUID(900)
	f, err := svc.CreateFacility(ctx, app.CreateFacilityInput{
		OwnerID: owner, Name: "Riverside Courts", Address: "123 Main St",
	})
	if err != nil {
		t.Fatalf("CreateFacility: %v", err)
	}
	court, err := svc.AddCourt(ctx, f.ID, owner, "Court 1")
	if err != nil {
		t.Fatalf("AddCourt: %v", err)
	}

	t.Run("known court resolves to its facility", func(t *testing.T) {
		got, err := svc.GetCourt(ctx, court.ID)
		if err != nil {
			t.Fatalf("GetCourt: %v", err)
		}
		if got.FacilityID != f.ID {
			t.Errorf("FacilityID = %q, want %q", got.FacilityID, f.ID)
		}
		if got.ID != court.ID {
			t.Errorf("ID = %q, want %q", got.ID, court.ID)
		}
	})

	t.Run("unknown but well-formed court is ErrCourtNotFound", func(t *testing.T) {
		if _, err := svc.GetCourt(ctx, seqUUID(4242)); !errors.Is(err, domain.ErrCourtNotFound) {
			t.Fatalf("GetCourt(unknown) error = %v, want %v", err, domain.ErrCourtNotFound)
		}
	})

	t.Run("malformed court id is ErrCourtNotFound and never reaches the repository", func(t *testing.T) {
		for _, id := range malformedFacilityIDs {
			before := repo.getCourtByIDCalls.Load()
			if _, err := svc.GetCourt(ctx, id); !errors.Is(err, domain.ErrCourtNotFound) {
				t.Fatalf("GetCourt(%q) error = %v, want %v", id, err, domain.ErrCourtNotFound)
			}
			if after := repo.getCourtByIDCalls.Load(); after != before {
				t.Fatalf("GetCourt(%q): a malformed court id reached the repository — against real Postgres that hits a `uuid` column and panics", id)
			}
		}
	})
}
