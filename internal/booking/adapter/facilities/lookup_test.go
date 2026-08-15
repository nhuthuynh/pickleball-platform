// Package facilities_test covers internal/booking/adapter/facilities, the one
// place Booking is allowed to call into the Facilities context.
//
// Before T13.1 this file held a single compile-time assertion and nothing
// else — 12 lines that could not fail for any behavioural reason. That is the
// exact shape of coverage gap T12 retro finding 2 identified: issue #146
// shipped through two individually well-tested tickets because the adapter
// joining the two contexts had no test that could fail.
//
// These tests drive the REAL facilitiesapp.Service over an in-memory
// port.Repository fake, following
// internal/payments/adapter/socialplay/registration_updater_test.go and
// internal/booking/adapter/identity/lookup_test.go. No Docker, no
// testcontainers. Faking Facilities' app.Service wholesale would prove only
// that a stub returns what it was told to; driving the real service means the
// uuidShape guard, domain.Facility.EnsureOwner and the real sentinels are all
// in the loop.
package facilities_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	bookingfacilities "github.com/nhuthuynh/white-label/internal/booking/adapter/facilities"
	"github.com/nhuthuynh/white-label/internal/booking/domain"
	"github.com/nhuthuynh/white-label/internal/booking/port"
	facilitiesapp "github.com/nhuthuynh/white-label/internal/facilities/app"
	facilitiesdomain "github.com/nhuthuynh/white-label/internal/facilities/domain"
)

// Compile-time proof that *bookingfacilities.Lookup satisfies
// port.FacilityLookup — asserted here rather than left for the cmd/server
// wiring to discover, mirroring the identical assertion in
// internal/competitions/adapter/facilities' own lookup_test.go.
//
// Kept deliberately (T13.1 instruction 6): it is cheap and correct. It is
// simply not a test — everything below it is.
var _ port.FacilityLookup = (*bookingfacilities.Lookup)(nil)

// Fixture ids are uuid-shaped because facilitiesapp.Service guards every
// caller-supplied Facility/Court id on uuidShape and answers a malformed one
// exactly like an unknown one. A non-uuid fixture here would therefore
// short-circuit before reaching any of the behaviour under test.
const (
	fixtureFacilityID = "3f2504e0-4f89-11d3-9a0c-0305e82c3301"
	fixtureCourtID    = "3f2504e0-4f89-11d3-9a0c-0305e82c3302"
	otherCourtID      = "3f2504e0-4f89-11d3-9a0c-0305e82c3303"
	unknownFacilityID = "3f2504e0-4f89-11d3-9a0c-0305e82c33ff"
)

// fixtureOwnerSubject is deliberately NOT uuid-shaped. Of the five packages
// T13.1 covers, this adapter is the only one whose surface carries an actor at
// all (EnsureFacilityOwner's actorUserID), and since T12.7 the value the
// production call site passes is auth.RequireSubject(ctx) — an opaque IdP
// subject, not a server-minted uuid. Issue #146 was exactly a test suite that
// never once used a realistically-shaped actor here.
const fixtureOwnerSubject = "auth0|facility-owner"

// errRepoUnavailable stands in for an infrastructure failure with no
// context-local meaning at all — the "any other error" arm of the adapter's
// translate().
var errRepoUnavailable = errors.New("facilities postgres: connection refused")

// inMemoryRepo is a minimal facilities port.Repository fake. Only the two
// reads this adapter can reach (GetFacilityByID, GetCourtByID) are
// meaningfully implemented; the rest exist to satisfy the interface.
//
// failWith, when set, is returned from both reads — the seam that lets a test
// drive an upstream error the adapter has no translation for.
type inMemoryRepo struct {
	facilities map[string]facilitiesdomain.Facility
	courts     map[string]facilitiesdomain.Court
	failWith   error
}

func newInMemoryRepo() *inMemoryRepo {
	return &inMemoryRepo{
		facilities: make(map[string]facilitiesdomain.Facility),
		courts:     make(map[string]facilitiesdomain.Court),
	}
}

func (r *inMemoryRepo) CreateFacility(_ context.Context, f facilitiesdomain.Facility) (facilitiesdomain.Facility, error) {
	r.facilities[f.ID] = f
	return f, nil
}

func (r *inMemoryRepo) GetFacilityByID(_ context.Context, id string) (facilitiesdomain.Facility, error) {
	if r.failWith != nil {
		return facilitiesdomain.Facility{}, r.failWith
	}
	f, ok := r.facilities[id]
	if !ok {
		return facilitiesdomain.Facility{}, facilitiesdomain.ErrFacilityNotFound
	}
	return f, nil
}

func (r *inMemoryRepo) ListFacilities(context.Context, string) ([]facilitiesdomain.Facility, error) {
	return nil, nil
}

func (r *inMemoryRepo) AddCourt(_ context.Context, c facilitiesdomain.Court) (facilitiesdomain.Court, error) {
	r.courts[c.ID] = c
	return c, nil
}

func (r *inMemoryRepo) AddCameraLink(context.Context, string, facilitiesdomain.CameraLink) (facilitiesdomain.CameraLink, error) {
	return facilitiesdomain.CameraLink{}, nil
}

func (r *inMemoryRepo) AttestCameraConsent(context.Context, string) error { return nil }

func (r *inMemoryRepo) GetCourtByID(_ context.Context, courtID string) (facilitiesdomain.Court, error) {
	if r.failWith != nil {
		return facilitiesdomain.Court{}, r.failWith
	}
	c, ok := r.courts[courtID]
	if !ok {
		return facilitiesdomain.Court{}, facilitiesdomain.ErrCourtNotFound
	}
	return c, nil
}

func (r *inMemoryRepo) ListCourtsForFacility(_ context.Context, facilityID string) ([]facilitiesdomain.Court, error) {
	var out []facilitiesdomain.Court
	for _, c := range r.courts {
		if c.FacilityID == facilityID {
			out = append(out, c)
		}
	}
	return out, nil
}

// stubIDs satisfies facilitiesapp.NewService's port.IDGenerator. Nothing here
// creates a Facility through the service — fixtures are seeded straight
// through the repository so the Facility can carry its Courts — so it is never
// called.
type stubIDs struct{}

func (stubIDs) NewID() string { return "unused" }

// newLookup builds the adapter over a REAL facilitiesapp.Service and returns
// it alongside the repo fake, so a test can both seed and fail the upstream.
func newLookup(t *testing.T) (*bookingfacilities.Lookup, *inMemoryRepo) {
	t.Helper()

	repo := newInMemoryRepo()
	return bookingfacilities.NewLookup(facilitiesapp.NewService(repo, stubIDs{})), repo
}

// seedFacility writes a Facility carrying courts straight through the repo.
// Seeding via the repository rather than Service.CreateFacility is required,
// not a shortcut: CreateFacility mints its own id from the IDGenerator and
// cannot attach Courts, and both are needed here.
func seedFacility(t *testing.T, repo *inMemoryRepo, ownerID string, courts ...facilitiesdomain.Court) {
	t.Helper()

	f, err := facilitiesdomain.NewFacility(fixtureFacilityID, ownerID, "Ada Courts", "", "1 Test Way", nil)
	if err != nil {
		t.Fatalf("building fixture facility: %v", err)
	}
	f.Courts = courts
	repo.facilities[f.ID] = f
	for _, c := range courts {
		repo.courts[c.ID] = c
	}
}

func mustCourt(t *testing.T, id, facilityID string) facilitiesdomain.Court {
	t.Helper()

	if facilityID == "" {
		// domain.NewCourt rejects an empty FacilityID, but a Court with a NULL
		// facility_id is a real row in this schema (0010_facilities.sql's
		// pre-Facilities seeded courts), and it is precisely the case
		// FacilityIDForCourt must answer ErrFacilityNotFound for.
		return facilitiesdomain.Court{ID: id, Name: "Unattached Court"}
	}
	c, err := facilitiesdomain.NewCourt(id, facilityID, "Court 1")
	if err != nil {
		t.Fatalf("building fixture court: %v", err)
	}
	return c
}

// TestEnsureFacilityOwner covers the actor-carrying method with a
// subject-shaped actor throughout.
func TestEnsureFacilityOwner(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		facilityID string
		actor      string
		wantErr    error
	}{
		{
			name:       "owner resolved by non-uuid IdP subject",
			facilityID: fixtureFacilityID,
			actor:      fixtureOwnerSubject,
			wantErr:    nil,
		},
		{
			name:       "a different subject is not the owner",
			facilityID: fixtureFacilityID,
			actor:      "auth0|someone-else",
			wantErr:    domain.ErrNotFacilityOwner,
		},
		{
			name:       "empty actor is never the owner",
			facilityID: fixtureFacilityID,
			actor:      "",
			wantErr:    domain.ErrNotFacilityOwner,
		},
		{
			name:       "unknown facility",
			facilityID: unknownFacilityID,
			actor:      fixtureOwnerSubject,
			wantErr:    domain.ErrFacilityNotFound,
		},
		{
			name: "malformed facility id is answered like an unknown one",
			// facilitiesapp's uuidShape guard fires before the repository is
			// touched; the adapter must still surface Booking's own sentinel.
			facilityID: "not-a-uuid",
			actor:      fixtureOwnerSubject,
			wantErr:    domain.ErrFacilityNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			lookup, repo := newLookup(t)
			seedFacility(t, repo, fixtureOwnerSubject)

			err := lookup.EnsureFacilityOwner(context.Background(), tt.facilityID, tt.actor)

			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("EnsureFacilityOwner(%q, %q) = %v, want nil", tt.facilityID, tt.actor, err)
				}
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("EnsureFacilityOwner(%q, %q) = %v, want %v", tt.facilityID, tt.actor, err, tt.wantErr)
			}
		})
	}
}

// TestFacilityIDForCourt covers the court->facility resolution, including the
// two distinct situations port.FacilityLookup deliberately collapses into one
// sentinel.
func TestFacilityIDForCourt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		courtID string
		want    string
		wantErr error
	}{
		{
			name:    "court belonging to a facility",
			courtID: fixtureCourtID,
			want:    fixtureFacilityID,
		},
		{
			name: "court belonging to no facility",
			// courts.facility_id is nullable; from Booking's side "no Facility
			// to scope a discount to" is the same answer as "no such Court".
			courtID: otherCourtID,
			wantErr: domain.ErrFacilityNotFound,
		},
		{
			name:    "unknown court",
			courtID: "3f2504e0-4f89-11d3-9a0c-0305e82c33ee",
			wantErr: domain.ErrFacilityNotFound,
		},
		{
			name:    "malformed court id",
			courtID: "not-a-uuid",
			wantErr: domain.ErrFacilityNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			lookup, repo := newLookup(t)
			seedFacility(t, repo, fixtureOwnerSubject,
				mustCourt(t, fixtureCourtID, fixtureFacilityID),
				mustCourt(t, otherCourtID, ""),
			)

			got, err := lookup.FacilityIDForCourt(context.Background(), tt.courtID)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("FacilityIDForCourt(%q) err = %v, want %v", tt.courtID, err, tt.wantErr)
				}
				if got != "" {
					t.Fatalf("FacilityIDForCourt(%q) = %q, want empty on error", tt.courtID, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("FacilityIDForCourt(%q) unexpected err: %v", tt.courtID, err)
			}
			if got != tt.want {
				t.Fatalf("FacilityIDForCourt(%q) = %q, want %q", tt.courtID, got, tt.want)
			}
		})
	}
}

// TestCourtIDsForFacility proves only ids cross the boundary, and in the
// Facility's own court order.
func TestCourtIDsForFacility(t *testing.T) {
	t.Parallel()

	t.Run("returns every court id", func(t *testing.T) {
		t.Parallel()

		lookup, repo := newLookup(t)
		seedFacility(t, repo, fixtureOwnerSubject,
			mustCourt(t, fixtureCourtID, fixtureFacilityID),
			mustCourt(t, otherCourtID, fixtureFacilityID),
		)

		got, err := lookup.CourtIDsForFacility(context.Background(), fixtureFacilityID)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		want := []string{fixtureCourtID, otherCourtID}
		if len(got) != len(want) {
			t.Fatalf("CourtIDsForFacility = %v, want %v", got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("CourtIDsForFacility = %v, want %v", got, want)
			}
		}
	})

	t.Run("facility with no courts yields an empty slice and no error", func(t *testing.T) {
		t.Parallel()

		lookup, repo := newLookup(t)
		seedFacility(t, repo, fixtureOwnerSubject)

		got, err := lookup.CourtIDsForFacility(context.Background(), fixtureFacilityID)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("CourtIDsForFacility = %v, want empty", got)
		}
	})

	t.Run("unknown facility", func(t *testing.T) {
		t.Parallel()

		lookup, repo := newLookup(t)
		seedFacility(t, repo, fixtureOwnerSubject)

		if _, err := lookup.CourtIDsForFacility(context.Background(), unknownFacilityID); !errors.Is(err, domain.ErrFacilityNotFound) {
			t.Fatalf("CourtIDsForFacility = %v, want %v", err, domain.ErrFacilityNotFound)
		}
	})
}

// TestNeverLeaksFacilitiesSentinels holds CLAUDE.md rule 5 at this boundary,
// across every method on the port.
//
// Two properties, both required by T13.1 instruction 3:
//
//  1. The sentinels Booking has its own names for are translated —
//     facilitiesdomain.ErrFacilityNotFound must arrive as
//     domain.ErrFacilityNotFound and must no longer match the upstream one.
//     (Covered by the tables above for the translated arm; asserted here for
//     the non-leak half.)
//  2. Everything else is wrapped with %s, not %w, so a Booking-side caller
//     cannot errors.Is() across the boundary at all — not against the upstream
//     error it came from, and not accidentally against one of Booking's own
//     sentinels either.
func TestNeverLeaksFacilitiesSentinels(t *testing.T) {
	t.Parallel()

	// Upstream errors the adapter has no translation for on ANY of these
	// paths: one foreign-context sentinel, one plain infra failure.
	//
	// ErrCameraConsentRequired is chosen precisely because it is a real
	// facilitiesdomain sentinel that this adapter's translate() does not name.
	// ErrCourtNotFound would NOT work here and the first draft of this test
	// caught that: FacilityIDForCourt translates it deliberately, so using it
	// would assert the opposite of what this test is for. The point is not
	// this particular sentinel — it is that any Facilities error Booking has
	// no name for must arrive untyped.
	upstreams := []struct {
		name string
		err  error
	}{
		{name: "untranslated facilities sentinel", err: facilitiesdomain.ErrCameraConsentRequired},
		{name: "infrastructure failure", err: errRepoUnavailable},
	}

	calls := []struct {
		name string
		call func(*bookingfacilities.Lookup) error
	}{
		{
			name: "EnsureFacilityOwner",
			call: func(l *bookingfacilities.Lookup) error {
				return l.EnsureFacilityOwner(context.Background(), fixtureFacilityID, fixtureOwnerSubject)
			},
		},
		{
			name: "FacilityIDForCourt",
			call: func(l *bookingfacilities.Lookup) error {
				_, err := l.FacilityIDForCourt(context.Background(), fixtureCourtID)
				return err
			},
		},
		{
			name: "CourtIDsForFacility",
			call: func(l *bookingfacilities.Lookup) error {
				_, err := l.CourtIDsForFacility(context.Background(), fixtureFacilityID)
				return err
			},
		},
	}

	for _, up := range upstreams {
		for _, c := range calls {
			t.Run(up.name+"/"+c.name, func(t *testing.T) {
				t.Parallel()

				lookup, repo := newLookup(t)
				seedFacility(t, repo, fixtureOwnerSubject, mustCourt(t, fixtureCourtID, fixtureFacilityID))
				repo.failWith = up.err

				err := c.call(lookup)
				if err == nil {
					t.Fatalf("%s: want an error when the upstream fails", c.name)
				}
				if errors.Is(err, up.err) {
					t.Fatalf("%s leaked the upstream error across the boundary: %v", c.name, err)
				}
				if errors.Is(err, domain.ErrFacilityNotFound) || errors.Is(err, domain.ErrNotFacilityOwner) {
					t.Fatalf("%s misclassified an unrelated upstream failure as a Booking sentinel: %v", c.name, err)
				}
				// The wrapping still has to say what failed — an untyped error
				// that also loses its message would be worse, not better.
				if !strings.Contains(err.Error(), up.err.Error()) {
					t.Fatalf("%s dropped the upstream message: %v", c.name, err)
				}
			})
		}
	}
}

// TestFacilityNotFoundIsTranslatedNotForwarded is the other half of rule 5 at
// this boundary: the one sentinel that IS translated must arrive as Booking's
// own and must NOT still satisfy errors.Is against the Facilities one. Split
// out from the table above because "translated" is a two-sided claim and a
// test asserting only the positive half would pass against an adapter that
// returned the upstream error wrapped with %w alongside its own.
func TestFacilityNotFoundIsTranslatedNotForwarded(t *testing.T) {
	t.Parallel()

	lookup, repo := newLookup(t)
	seedFacility(t, repo, fixtureOwnerSubject)

	err := lookup.EnsureFacilityOwner(context.Background(), unknownFacilityID, fixtureOwnerSubject)

	if !errors.Is(err, domain.ErrFacilityNotFound) {
		t.Fatalf("err = %v, want domain.ErrFacilityNotFound", err)
	}
	if errors.Is(err, facilitiesdomain.ErrFacilityNotFound) {
		t.Fatalf("err = %v, still matches the facilitiesdomain sentinel", err)
	}
}

// TestNotFacilityOwnerIsTranslatedNotForwarded is the same two-sided claim for
// the ownership rejection, which reaches Booking through
// facilitiesdomain.Facility.EnsureOwner rather than through translate().
func TestNotFacilityOwnerIsTranslatedNotForwarded(t *testing.T) {
	t.Parallel()

	lookup, repo := newLookup(t)
	seedFacility(t, repo, fixtureOwnerSubject)

	err := lookup.EnsureFacilityOwner(context.Background(), fixtureFacilityID, "auth0|someone-else")

	if !errors.Is(err, domain.ErrNotFacilityOwner) {
		t.Fatalf("err = %v, want domain.ErrNotFacilityOwner", err)
	}
	if errors.Is(err, facilitiesdomain.ErrNotFacilityOwner) {
		t.Fatalf("err = %v, still matches the facilitiesdomain sentinel", err)
	}
}
