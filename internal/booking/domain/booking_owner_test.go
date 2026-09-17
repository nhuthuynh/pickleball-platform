package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/nhuthuynh/white-label/internal/booking/domain"
)

// T55.1 (DECISION D1, ADR-0015 option (a)) — a Booking now records who owns
// it, and only that owner may cancel it.
//
// These are the domain-side tests. They are the mirror of the DB-side guard
// (owner_user_id NOT NULL REFERENCES identity_users, migration 0027) exactly
// as CLAUDE.md rule 4 requires for the no-double-booking invariant: the
// column is authoritative for existence, this file is authoritative for the
// rule, and both must say the same thing.

func ownerTestRange(t *testing.T) domain.TimeRange {
	t.Helper()
	start := time.Date(2026, 9, 4, 10, 0, 0, 0, time.UTC)
	r, err := domain.NewTimeRange(start, start.Add(time.Hour))
	if err != nil {
		t.Fatalf("NewTimeRange: %v", err)
	}
	return r
}

// NewBooking must reject a booking with no owner, for the same reason
// NewRecurringHireTemplate rejects an empty RequestedByUserID: an
// unattributed row is exactly the hole #144 reported.
func TestNewBookingRequiresOwner(t *testing.T) {
	r := ownerTestRange(t)

	tests := []struct {
		name        string
		ownerUserID string
		want        error
	}{
		{
			name:        "empty owner is rejected",
			ownerUserID: "",
			want:        domain.ErrEmptyOwnerUserID,
		},
		{
			name:        "owner present is accepted",
			ownerUserID: "11111111-1111-1111-1111-111111111111",
			want:        nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b, err := domain.NewBooking(
				"b-1",
				"22222222-2222-2222-2222-222222222222",
				domain.SourceIndividual,
				r,
				"",
				tc.ownerUserID,
			)
			if !errors.Is(err, tc.want) {
				t.Fatalf("NewBooking error = %v, want %v", err, tc.want)
			}
			if tc.want == nil && b.OwnerUserID != tc.ownerUserID {
				t.Fatalf("OwnerUserID = %q, want %q", b.OwnerUserID, tc.ownerUserID)
			}
		})
	}
}

// EnsureOwner is the rule #144 asked for. It mirrors
// socialplay/domain.Game.EnsureHost and facilities' Facility.EnsureOwner —
// same shape, same empty-actor handling, so there is one idea of "owner" in
// the codebase and it behaves identically everywhere.
func TestBookingEnsureOwner(t *testing.T) {
	const owner = "11111111-1111-1111-1111-111111111111"
	const stranger = "33333333-3333-3333-3333-333333333333"

	r := ownerTestRange(t)
	b, err := domain.NewBooking(
		"b-1",
		"22222222-2222-2222-2222-222222222222",
		domain.SourceIndividual,
		r,
		"",
		owner,
	)
	if err != nil {
		t.Fatalf("NewBooking: %v", err)
	}

	tests := []struct {
		name        string
		actorUserID string
		want        error
	}{
		{
			name:        "the owner may act",
			actorUserID: owner,
			want:        nil,
		},
		{
			name:        "a different user may not",
			actorUserID: stranger,
			want:        domain.ErrNotBookingOwner,
		},
		{
			// The case #144 is actually about: no token at all. An empty
			// actor must never satisfy the check, even against a booking
			// whose owner is somehow also empty — hence the explicit arm
			// rather than relying on string equality.
			name:        "an unidentified caller may not",
			actorUserID: "",
			want:        domain.ErrNotBookingOwner,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := b.EnsureOwner(tc.actorUserID); !errors.Is(err, tc.want) {
				t.Fatalf("EnsureOwner(%q) = %v, want %v", tc.actorUserID, err, tc.want)
			}
		})
	}
}

// The empty-owner booking cannot be built through NewBooking, but a
// zero-value struct can still be constructed in-process (a repository
// hydrating a row, a test fixture). EnsureOwner must not let an empty actor
// match an empty owner and pass.
func TestBookingEnsureOwnerRejectsEmptyAgainstEmpty(t *testing.T) {
	var b domain.Booking
	if err := b.EnsureOwner(""); !errors.Is(err, domain.ErrNotBookingOwner) {
		t.Fatalf("EnsureOwner(\"\") on zero-value Booking = %v, want %v", err, domain.ErrNotBookingOwner)
	}
}

// testOwner is the owner these pre-D1 tests pass so they keep exercising
// what they were written to exercise. Their subject is time-range, source
// and overlap logic — not ownership, which booking_owner_test.go covers —
// so they supply a valid owner and otherwise stay as they were.
const testOwner = "11111111-1111-1111-1111-111111111111"
