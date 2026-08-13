package domain_test

import (
	"errors"
	"testing"

	"github.com/nhuthuynh/white-label/internal/identity/domain"
)

// TestUser_UpdateSelfReportedLevel_RejectsMismatchedActor proves the
// ownership check runs first: a non-self actor is rejected with
// domain.ErrNotSelf without ever learning whether the new level itself
// would have been valid, mirroring
// internal/facilities/domain.Facility.AddCameraLink's check ordering
// (EnsureOwner before the domain-specific rule).
func TestUser_UpdateSelfReportedLevel_RejectsMismatchedActor(t *testing.T) {
	t.Parallel()

	u, err := domain.NewUser("user-1", "Ada Lovelace", []domain.Role{domain.RolePlayer}, domain.SelfReportedStartingLevel(3))
	if err != nil {
		t.Fatalf("unexpected err constructing fixture: %v", err)
	}

	_, err = u.UpdateSelfReportedLevel("user-2", domain.SelfReportedStartingLevel(4))
	if !errors.Is(err, domain.ErrNotSelf) {
		t.Fatalf("got err %v, want %v", err, domain.ErrNotSelf)
	}
}

// TestUser_UpdateSelfReportedLevel_RejectsOutOfRangeLevel proves the
// bounded range invariant applies on update too, not just at construction.
func TestUser_UpdateSelfReportedLevel_RejectsOutOfRangeLevel(t *testing.T) {
	t.Parallel()

	u, err := domain.NewUser("user-1", "Ada Lovelace", []domain.Role{domain.RolePlayer}, domain.SelfReportedStartingLevel(3))
	if err != nil {
		t.Fatalf("unexpected err constructing fixture: %v", err)
	}

	_, err = u.UpdateSelfReportedLevel("user-1", domain.SelfReportedStartingLevel(6))
	if !errors.Is(err, domain.ErrInvalidSelfReportedStartingLevel) {
		t.Fatalf("got err %v, want %v", err, domain.ErrInvalidSelfReportedStartingLevel)
	}
}

// TestUser_UpdateSelfReportedLevel_Valid proves the happy path: the owning
// actor updates their own level and the returned User carries it.
func TestUser_UpdateSelfReportedLevel_Valid(t *testing.T) {
	t.Parallel()

	u, err := domain.NewUser("user-1", "Ada Lovelace", []domain.Role{domain.RolePlayer}, domain.SelfReportedStartingLevel(3))
	if err != nil {
		t.Fatalf("unexpected err constructing fixture: %v", err)
	}

	updated, err := u.UpdateSelfReportedLevel("user-1", domain.SelfReportedStartingLevel(5))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if updated.SelfReportedStartingLevel != 5 {
		t.Fatalf("SelfReportedStartingLevel = %d, want 5", updated.SelfReportedStartingLevel)
	}
	// The original aggregate is untouched — UpdateSelfReportedLevel returns
	// a new value, mirroring domain.Facility.AttestCameraConsent's pointer
	// receiver (which does mutate) is NOT the pattern here: this uses a
	// value receiver + return, so the caller must use the returned User.
	if u.SelfReportedStartingLevel != 3 {
		t.Fatalf("original u.SelfReportedStartingLevel = %d, want unchanged 3", u.SelfReportedStartingLevel)
	}
}
