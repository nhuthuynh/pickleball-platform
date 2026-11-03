package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nhuthuynh/white-label/internal/facilities/app"
	"github.com/nhuthuynh/white-label/internal/facilities/domain"
)

// TestCourtTimezone is T68.1's read into this context (ADR-0004, closes
// #356): Booking's quote path needs the IANA zone of the Facility that owns a
// Court, because a PricingRule's windows are clock times local to that
// facility. The zone lives on facilities (0034), so the read lives here —
// Booking does not reach into the courts or facilities tables.
//
// Three answers, and the difference between the second and third is the whole
// point:
//   - a facility with a zone: the zone;
//   - a facility whose timezone column is the empty string (0034's default,
//     since no creation path sets it yet): that empty string and no error,
//     because "nobody has set one" is a real state of the data;
//   - no such Court, or a Court belonging to no Facility: ErrCourtNotFound,
//     since there is no facility to have a zone at all.
//
// Booking turns both of the last two into its own ErrFacilityTimezoneUnset at
// the adapter boundary; this context reports the data as it is rather than
// pre-deciding for one caller.
func TestCourtTimezone(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := newInMemoryRepo()
	svc := app.NewService(repo, stubIdentity{}, &sequentialIDs{})

	owner := seqUUID(930)
	zoned, err := svc.CreateFacility(ctx, app.CreateFacilityInput{
		OwnerID: owner, Name: "Saigon Courts", Address: "1 Le Loi",
	})
	if err != nil {
		t.Fatalf("CreateFacility: %v", err)
	}
	zonedCourt, err := svc.AddCourt(ctx, zoned.ID, owner, "Court 1")
	if err != nil {
		t.Fatalf("AddCourt: %v", err)
	}
	// Set through the repo, not an app method: nothing writes
	// facilities.timezone yet (0034 defaults it to '' and making it a
	// required field on facility creation is ADR-0004's remaining half), so
	// this is how a seeded or operator-updated row looks.
	repo.timezones[zoned.ID] = "Asia/Ho_Chi_Minh"

	unzoned, err := svc.CreateFacility(ctx, app.CreateFacilityInput{
		OwnerID: owner, Name: "Nameless Courts", Address: "2 Le Loi",
	})
	if err != nil {
		t.Fatalf("CreateFacility: %v", err)
	}
	unzonedCourt, err := svc.AddCourt(ctx, unzoned.ID, owner, "Court 1")
	if err != nil {
		t.Fatalf("AddCourt: %v", err)
	}

	// A Court with facility_id NULL — the shape both courts of 0002_seed.sql
	// had before 0034 attached them. domain.NewCourt rejects an empty
	// FacilityID, so it is written straight to the fake, exactly as
	// internal/booking/adapter/facilities' own fixture does.
	detachedCourt := seqUUID(931)
	repo.courts[detachedCourt] = domain.Court{ID: detachedCourt, Name: "Unattached Court"}

	tests := []struct {
		name     string
		courtID  string
		wantZone string
		wantErr  error
	}{
		{name: "a court whose facility has a zone", courtID: zonedCourt.ID, wantZone: "Asia/Ho_Chi_Minh"},
		{name: "a court whose facility has no zone recorded", courtID: unzonedCourt.ID, wantZone: ""},
		{name: "a court belonging to no facility", courtID: detachedCourt, wantErr: domain.ErrCourtNotFound},
		{name: "an unknown court", courtID: seqUUID(999), wantErr: domain.ErrCourtNotFound},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			zone, err := svc.CourtTimezone(ctx, tt.courtID)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("got err %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if zone != tt.wantZone {
				t.Errorf("zone = %q, want %q", zone, tt.wantZone)
			}
		})
	}
}

// TestCourtTimezone_MalformedIDNeverReachesRepository is the same T10.7 guard
// every other id-taking method here carries, proved the only way it can be:
// by the repository call never happening. The return value cannot show it —
// a malformed id and an unknown one both answer ErrCourtNotFound — and the
// in-memory fake cannot reproduce Postgres rejecting a non-UUID against a
// `uuid` column, which is where the real adapter's mustUUID panics.
func TestCourtTimezone_MalformedIDNeverReachesRepository(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := newInMemoryRepo()
	svc := app.NewService(repo, stubIdentity{}, &sequentialIDs{})

	if _, err := svc.CourtTimezone(ctx, "not-a-uuid"); !errors.Is(err, domain.ErrCourtNotFound) {
		t.Fatalf("got err %v, want %v", err, domain.ErrCourtNotFound)
	}
	if got := repo.courtTimezoneCalls.Load(); got != 0 {
		t.Errorf("repository CourtTimezone called %d times for a malformed id, want 0", got)
	}
}
