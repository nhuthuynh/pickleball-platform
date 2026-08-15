// Boundary validation for the caller-supplied per-session court_ids
// collections on ScheduleCompetition (T14.8, issue #156).
//
// The Competitions side of the same defect Social Play has, with one extra
// dimension: courts are nested inside Sessions, so the input is a collection
// of collections and a malformed entry can hide in the second session behind
// a first session that reserves perfectly well.
package app_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/nhuthuynh/white-label/internal/competitions/domain"
)

// courtID mints a deterministic, UUID-shaped court id — the shape the real
// courts.id column holds and the only shape bookingapp.CreateBooking accepts
// across port.CourtReservation. See the twin helper in
// internal/socialplay/app/court_ids_shape_test.go.
func courtID(n int) string {
	return fmt.Sprintf("00000000-0000-4000-c000-%012d", n)
}

func malformedCourtIDs() []string {
	return []string{
		"",
		"not-a-uuid",
		"court-1", // the old fixture shape
		"0",
		"'; DROP TABLE bookings;--",
		"../../etc/passwd",
		"6ba7b810-9dad-11d1-80b4-00c04fd430c\x00",
		"6ba7b810-9dad-11d1-80b4-00c04fd430c",
		"zzzzzzzz-9dad-11d1-80b4-00c04fd430c8",
		"{6ba7b810-9dad-11d1-80b4-00c04fd430c8}",
		"urn:uuid:6ba7b810-9dad-11d1-80b4-00c04fd430c8",
	}
}

// TestScheduleCompetition_MalformedCourtIDRejectedBeforeReservingCourts puts
// the malformed entry in the SECOND session, behind a whole session that
// would otherwise reserve successfully — the nested-loop equivalent of Social
// Play's "malformed entry last" case, and the shape a first-element-only
// guard would fail to catch.
//
// Assertions are on observable state (no booking held, no Competition
// persisted), following
// TestScheduleCompetition_RollsBackEverySessionWhenLaterCourtUnavailable's
// stated reasoning: an implementation that reserved and then "rolled back"
// with a release that freed nothing would still fail this test.
func TestScheduleCompetition_MalformedCourtIDRejectedBeforeReservingCourts(t *testing.T) {
	t.Parallel()

	for _, malformed := range malformedCourtIDs() {
		t.Run(malformed, func(t *testing.T) {
			t.Parallel()

			repo := newFakeRepository()
			reservation := newFakeReservation()
			svc := newTestService(repo, reservation, newFakeFacilityLookup(), &fakeShareTokens{})

			in := validInput(t,
				session(t, "2026-09-01T09:00:00Z", "2026-09-01T12:00:00Z", courtID(1)),
				session(t, "2026-09-02T09:00:00Z", "2026-09-02T12:00:00Z", courtID(2), malformed),
			)

			_, err := svc.ScheduleCompetition(context.Background(), in)
			if !errors.Is(err, domain.ErrMalformedCourtID) {
				t.Fatalf("ScheduleCompetition(session 2 court_ids=[valid, %q]) error = %v, want domain.ErrMalformedCourtID", malformed, err)
			}
			if len(reservation.reserveCalls) != 0 {
				t.Fatalf("reserved %v; a malformed court id in ANY session must be rejected before ANY court is reserved", reservation.reserveCalls)
			}
			if len(reservation.active) != 0 {
				t.Fatalf("holding %d bookings, want 0", len(reservation.active))
			}
			if len(repo.competitions) != 0 {
				t.Fatalf("persisted %d competitions, want 0 — no partial state may survive a rejected call", len(repo.competitions))
			}
		})
	}
}

// TestScheduleCompetition_EmptyCourtIDsStillReportsEmptyNotMalformed is the
// no-regression rail: an empty per-session court list was already a clean
// ErrEmptyCourtIDs and must stay one.
func TestScheduleCompetition_EmptyCourtIDsStillReportsEmptyNotMalformed(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	svc := newTestService(repo, newFakeReservation(), newFakeFacilityLookup(), &fakeShareTokens{})

	in := validInput(t,
		session(t, "2026-09-01T09:00:00Z", "2026-09-01T12:00:00Z", courtID(1)),
		session(t, "2026-09-02T09:00:00Z", "2026-09-02T12:00:00Z"),
	)

	_, err := svc.ScheduleCompetition(context.Background(), in)
	if !errors.Is(err, domain.ErrEmptyCourtIDs) {
		t.Fatalf("error = %v, want domain.ErrEmptyCourtIDs", err)
	}
}

// TestScheduleCompetition_WellFormedCourtIDsStillReserve is the too-strict
// rail, including an upper-case-hex id, which is canonical and must be
// accepted.
func TestScheduleCompetition_WellFormedCourtIDsStillReserve(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	reservation := newFakeReservation()
	svc := newTestService(repo, reservation, newFakeFacilityLookup(), &fakeShareTokens{})

	in := validInput(t,
		session(t, "2026-09-01T09:00:00Z", "2026-09-01T12:00:00Z", courtID(1), "6BA7B810-9DAD-11D1-80B4-00C04FD430C8"),
	)

	if _, err := svc.ScheduleCompetition(context.Background(), in); err != nil {
		t.Fatalf("well-formed court ids must still schedule, got err: %v", err)
	}
	if len(reservation.active) != 2 {
		t.Fatalf("holding %d bookings, want 2", len(reservation.active))
	}
}
