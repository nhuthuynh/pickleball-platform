package facilities_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/nhuthuynh/white-label/internal/booking/domain"
	facilitiesdomain "github.com/nhuthuynh/white-label/internal/facilities/domain"
)

// T68.1 (ADR-0004, closes #356) — the boundary half of the fix.
//
// The zone a price band resolves in is a fact the Facilities context owns, so
// it crosses this adapter, and the adapter's job is the one CLAUDE.md rule 5
// describes: translate. Three different upstream answers mean the same thing
// to Booking and must arrive as one Booking-side sentinel; a fourth (an
// infrastructure failure) must NOT be flattened into it, because "this
// facility has no zone" is an operator's problem and "the database is down"
// is not.
//
// These tests drive the REAL facilitiesapp.Service over the repo fake, like
// every other test in this package, so the uuidShape guard and the real
// facilitiesdomain sentinels are in the loop.
func TestTimezoneForCourt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		// seed, when non-nil, runs before the call.
		seed     func(t *testing.T, repo *inMemoryRepo)
		courtID  string
		wantZone string
		wantErr  error
	}{
		{
			name: "a court whose facility has a zone",
			seed: func(t *testing.T, repo *inMemoryRepo) {
				seedFacility(t, repo, fixtureOwnerSubject, mustCourt(t, fixtureCourtID, fixtureFacilityID))
				repo.timezones[fixtureFacilityID] = "Asia/Ho_Chi_Minh"
			},
			courtID:  fixtureCourtID,
			wantZone: "Asia/Ho_Chi_Minh",
		},
		{
			// facilities.timezone = '' — 0034's default, which every Facility
			// created through the API carries until a zone is set. The empty
			// string must NOT reach Booking: time.LoadLocation("") answers
			// UTC, which is the mispricing #356 reported.
			name: "a facility with no zone recorded is the unset sentinel, not an empty string",
			seed: func(t *testing.T, repo *inMemoryRepo) {
				seedFacility(t, repo, fixtureOwnerSubject, mustCourt(t, fixtureCourtID, fixtureFacilityID))
			},
			courtID: fixtureCourtID,
			wantErr: domain.ErrFacilityTimezoneUnset,
		},
		{
			// courts.facility_id IS NULL — both courts of 0002_seed.sql before
			// 0034 attaches them, and any court added outside a facility.
			name: "a court belonging to no facility",
			seed: func(t *testing.T, repo *inMemoryRepo) {
				repo.courts[otherCourtID] = mustCourt(t, otherCourtID, "")
			},
			courtID: otherCourtID,
			wantErr: domain.ErrFacilityTimezoneUnset,
		},
		{
			name:    "an unknown court",
			courtID: otherCourtID,
			wantErr: domain.ErrFacilityTimezoneUnset,
		},
		{
			// facilitiesapp.Service's uuidShape guard answers a malformed id
			// exactly like an unknown one, so this must not reach the
			// repository (where the Postgres adapter's mustUUID panics).
			name:    "a malformed court id",
			courtID: "not-a-uuid",
			wantErr: domain.ErrFacilityTimezoneUnset,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			lookup, repo := newLookup(t)
			if tt.seed != nil {
				tt.seed(t, repo)
			}

			zone, err := lookup.TimezoneForCourt(context.Background(), tt.courtID)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("got err %v, want %v", err, tt.wantErr)
				}
				if zone != "" {
					t.Errorf("zone = %q alongside an error, want empty", zone)
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

// TestTimezoneForCourt_InfrastructureFailureIsNotTheUnsetSentinel is the
// negative half, and the one that would catch an over-eager translation: an
// upstream failure with no context-local meaning must stay distinguishable
// from "no zone recorded". Collapsing them would tell an operator to go and
// set a timezone that is already set.
//
// It must also not leak the facilitiesdomain error type — translate() wraps
// with %s rather than %w for exactly that reason (CLAUDE.md rule 5).
func TestTimezoneForCourt_InfrastructureFailureIsNotTheUnsetSentinel(t *testing.T) {
	t.Parallel()

	lookup, repo := newLookup(t)
	seedFacility(t, repo, fixtureOwnerSubject, mustCourt(t, fixtureCourtID, fixtureFacilityID))
	repo.timezones[fixtureFacilityID] = "Asia/Ho_Chi_Minh"
	repo.failWith = errRepoUnavailable

	_, err := lookup.TimezoneForCourt(context.Background(), fixtureCourtID)
	if err == nil {
		t.Fatal("TimezoneForCourt with a failing repository returned no error")
	}
	if errors.Is(err, domain.ErrFacilityTimezoneUnset) {
		t.Errorf("an infrastructure failure was translated to ErrFacilityTimezoneUnset: %v", err)
	}
	if errors.Is(err, errRepoUnavailable) {
		t.Errorf("the upstream error type crossed the boundary (should be %%s-wrapped, not %%w): %v", err)
	}
	if !strings.Contains(err.Error(), errRepoUnavailable.Error()) {
		t.Errorf("err %q does not report the upstream cause", err)
	}
}

// TestTimezoneForCourt_FacilityNotFoundAlsoTranslates covers the other
// facilitiesdomain sentinel this method names. A repository that reports
// ErrFacilityNotFound — rather than ErrCourtNotFound — for a court whose
// facility row is gone is a legal implementation of that port, and the
// adapter must answer the same Booking-side sentinel either way rather than
// letting it fall through to the generic wrap.
func TestTimezoneForCourt_FacilityNotFoundAlsoTranslates(t *testing.T) {
	t.Parallel()

	lookup, repo := newLookup(t)
	seedFacility(t, repo, fixtureOwnerSubject, mustCourt(t, fixtureCourtID, fixtureFacilityID))
	repo.failWith = facilitiesdomain.ErrFacilityNotFound

	_, err := lookup.TimezoneForCourt(context.Background(), fixtureCourtID)
	if !errors.Is(err, domain.ErrFacilityTimezoneUnset) {
		t.Fatalf("got err %v, want %v", err, domain.ErrFacilityTimezoneUnset)
	}
	if errors.Is(err, facilitiesdomain.ErrFacilityNotFound) {
		t.Error("the facilitiesdomain sentinel crossed the boundary")
	}
}
