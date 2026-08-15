// Package facilities_test covers internal/socialplay/adapter/facilities, the
// one place Social Play is allowed to call into the Facilities context.
//
// Before T13.1 this package had no test file at all — not a vacuous test, an
// empty set (T12 retro finding 2). It is one of the five cross-context
// adapters that shipped with zero behavioural coverage, which is the gap that
// let issue #146 through on a sibling seam.
//
// These tests drive the REAL facilitiesapp.Service over an in-memory
// port.Repository fake, following
// internal/payments/adapter/socialplay/registration_updater_test.go. No
// Docker, no testcontainers. A stubbed Facilities service would prove only
// that the stub returns what it was told to; the real service keeps the
// uuidShape guard and the real sentinels in the loop.
package facilities_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	facilitiesapp "github.com/nhuthuynh/white-label/internal/facilities/app"
	facilitiesdomain "github.com/nhuthuynh/white-label/internal/facilities/domain"
	socialplayfacilities "github.com/nhuthuynh/white-label/internal/socialplay/adapter/facilities"
	"github.com/nhuthuynh/white-label/internal/socialplay/domain"
	"github.com/nhuthuynh/white-label/internal/socialplay/port"
)

// Compile-time proof that *socialplayfacilities.Lookup satisfies
// port.FacilityLookup, mirroring the identical assertion in
// internal/competitions/adapter/facilities and
// internal/booking/adapter/facilities. Cheap and correct — and, as T13.1
// instruction 6 puts it, simply not a test. Everything below it is.
var _ port.FacilityLookup = (*socialplayfacilities.Lookup)(nil)

// Fixture ids are uuid-shaped. This seam carries no actor at all —
// FacilityExists takes a facility id and nothing else — so T13.1's
// "non-uuid subject where the seam carries one" condition does not apply
// here, and a subject-shaped fixture would exercise a path production never
// takes: facilitiesapp.Service guards this id on uuidShape. Stated as a
// checked negative rather than left to inference.
const (
	fixtureFacilityID = "3f2504e0-4f89-11d3-9a0c-0305e82c3301"
	unknownFacilityID = "3f2504e0-4f89-11d3-9a0c-0305e82c33ff"
	fixtureOwnerID    = "auth0|facility-owner"
)

// errRepoUnavailable stands in for an infrastructure failure with no
// context-local meaning — the "any other error" arm of FacilityExists.
var errRepoUnavailable = errors.New("facilities postgres: connection refused")

// inMemoryRepo is a minimal facilities port.Repository fake. Only
// GetFacilityByID — the single read this adapter can reach — is meaningfully
// implemented; the rest exist to satisfy the interface. failWith, when set,
// drives an upstream error the adapter has no translation for.
type inMemoryRepo struct {
	facilities map[string]facilitiesdomain.Facility
	failWith   error
}

func newInMemoryRepo() *inMemoryRepo {
	return &inMemoryRepo{facilities: make(map[string]facilitiesdomain.Facility)}
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
	return c, nil
}

func (r *inMemoryRepo) AddCameraLink(context.Context, string, facilitiesdomain.CameraLink) (facilitiesdomain.CameraLink, error) {
	return facilitiesdomain.CameraLink{}, nil
}

func (r *inMemoryRepo) AttestCameraConsent(context.Context, string) error { return nil }

func (r *inMemoryRepo) GetCourtByID(context.Context, string) (facilitiesdomain.Court, error) {
	return facilitiesdomain.Court{}, facilitiesdomain.ErrCourtNotFound
}

func (r *inMemoryRepo) ListCourtsForFacility(context.Context, string) ([]facilitiesdomain.Court, error) {
	return nil, nil
}

// stubIDs satisfies facilitiesapp.NewService's port.IDGenerator. Fixtures are
// seeded straight through the repository so ids stay fixed, so it is never
// called.
type stubIDs struct{}

func (stubIDs) NewID() string { return "unused" }

// stubIdentityLookup satisfies facilitiesapp.NewService's port.IdentityLookup,
// which Facilities acquired in T13.3 (ADR-0014's subject -> User.ID seam).
//
// Nothing in this file resolves an actor: this adapter exercises Facilities'
// *read* side only, which sits below the seam and is handed no subject. The
// stub therefore resolves nothing, so a future change that made one of these
// reads depend on actor resolution would fail here rather than pass against
// an accommodating fake.
type stubIdentityLookup struct{}

func (stubIdentityLookup) UserIDBySubject(_ context.Context, _ string) (string, error) {
	return "", facilitiesdomain.ErrUserNotFound
}

// newLookup builds the adapter over a REAL facilitiesapp.Service, seeded with
// one Facility, and returns the repo fake so a test can fail the upstream.
func newLookup(t *testing.T) (*socialplayfacilities.Lookup, *inMemoryRepo) {
	t.Helper()

	repo := newInMemoryRepo()
	f, err := facilitiesdomain.NewFacility(fixtureFacilityID, fixtureOwnerID, "Ada Courts", "", "1 Test Way", nil)
	if err != nil {
		t.Fatalf("building fixture facility: %v", err)
	}
	repo.facilities[f.ID] = f

	return socialplayfacilities.NewLookup(facilitiesapp.NewService(repo, stubIdentityLookup{}, stubIDs{})), repo
}

// TestFacilityExists covers the adapter's whole surface: the yes answer, and
// both no answers.
func TestFacilityExists(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		facilityID string
		wantErr    error
	}{
		{
			name:       "existing facility",
			facilityID: fixtureFacilityID,
			wantErr:    nil,
		},
		{
			name:       "unknown facility",
			facilityID: unknownFacilityID,
			wantErr:    domain.ErrFacilityNotFound,
		},
		{
			name: "malformed facility id is answered like an unknown one",
			// facilitiesapp's uuidShape guard fires before the repository is
			// touched; Social Play's own sentinel must still be what surfaces.
			facilityID: "not-a-uuid",
			wantErr:    domain.ErrFacilityNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			lookup, _ := newLookup(t)

			err := lookup.FacilityExists(context.Background(), tt.facilityID)

			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("FacilityExists(%q) = %v, want nil", tt.facilityID, err)
				}
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("FacilityExists(%q) = %v, want %v", tt.facilityID, err, tt.wantErr)
			}
		})
	}
}

// TestFacilityNotFoundIsTranslatedNotForwarded is the two-sided half of
// CLAUDE.md rule 5: the translated sentinel must arrive as Social Play's own
// AND must no longer satisfy errors.Is against the Facilities one. Asserting
// only the positive half would pass against an adapter that returned the
// upstream error wrapped with %w alongside its own.
func TestFacilityNotFoundIsTranslatedNotForwarded(t *testing.T) {
	t.Parallel()

	lookup, _ := newLookup(t)

	err := lookup.FacilityExists(context.Background(), unknownFacilityID)

	if !errors.Is(err, domain.ErrFacilityNotFound) {
		t.Fatalf("err = %v, want socialplay domain.ErrFacilityNotFound", err)
	}
	if errors.Is(err, facilitiesdomain.ErrFacilityNotFound) {
		t.Fatalf("err = %v, still matches the facilitiesdomain sentinel", err)
	}
}

// TestNeverLeaksFacilitiesSentinels holds the other half of rule 5: every
// error this adapter has no name for is wrapped with %s, not %w, so a Social
// Play caller cannot errors.Is() across the context boundary at all.
func TestNeverLeaksFacilitiesSentinels(t *testing.T) {
	t.Parallel()

	upstreams := []struct {
		name string
		err  error
	}{
		// A real facilitiesdomain sentinel this adapter deliberately does not
		// name. The point is not this particular sentinel — it is that any
		// Facilities error Social Play has no translation for arrives untyped.
		{name: "untranslated facilities sentinel", err: facilitiesdomain.ErrCameraConsentRequired},
		{name: "infrastructure failure", err: errRepoUnavailable},
	}

	for _, up := range upstreams {
		t.Run(up.name, func(t *testing.T) {
			t.Parallel()

			lookup, repo := newLookup(t)
			repo.failWith = up.err

			err := lookup.FacilityExists(context.Background(), fixtureFacilityID)
			if err == nil {
				t.Fatal("want an error when the upstream fails")
			}
			if errors.Is(err, up.err) {
				t.Fatalf("leaked the upstream error across the boundary: %v", err)
			}
			if errors.Is(err, domain.ErrFacilityNotFound) {
				t.Fatalf("misclassified an unrelated upstream failure as a Social Play sentinel: %v", err)
			}
			// Untyped must not also mean unreadable.
			if !strings.Contains(err.Error(), up.err.Error()) {
				t.Fatalf("dropped the upstream message: %v", err)
			}
		})
	}
}
