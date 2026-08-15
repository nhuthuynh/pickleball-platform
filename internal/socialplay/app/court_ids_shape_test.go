// Boundary validation for the caller-supplied court_ids collection on
// ScheduleGame (T14.8, issue #156).
//
// Sibling of malformed_id_test.go, which covers the caller-supplied *scalar*
// Game ID. The difference that made #156 possible: CourtIDs is a slice, and
// it is consumed by a *different* bounded context reached through
// port.CourtReservation, so T10.7's sweep — which covered ids consumed
// within this context — never reached it. domain.NewGame checks only that
// the slice is non-empty; every entry's shape was unchecked until this
// ticket.
package app_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/nhuthuynh/white-label/internal/socialplay/app"
	"github.com/nhuthuynh/white-label/internal/socialplay/domain"
)

// courtID mints a deterministic, UUID-shaped court id — the shape a real
// courts.id column actually holds, and the only shape
// bookingapp.Service.CreateBooking accepts on the other side of
// port.CourtReservation (its own uuidShape guard, T10.7/issue #97).
//
// The pre-T14.8 fixtures in this package used "court-1", a value the real
// Booking context rejects outright: the fakes never noticed, which is the
// fixture infidelity docs/LESSONS.md' T9 entry names, in the specific form
// that hid #156 for four sprints.
func courtID(n int) string {
	return fmt.Sprintf("00000000-0000-4000-c000-%012d", n)
}

// malformedCourtIDs is the corpus malformed_id_test.go uses for Game IDs,
// applied here to a court id. The braced and urn: forms matter for the same
// reason they do there: github.com/google/uuid's Validate accepts them and
// pgtype.UUID.Scan does not, which is why the guard is a canonical-form
// check and not a permissive parse.
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

// TestScheduleGame_MalformedCourtIDRejectedBeforeReservingCourts is #156's
// core fix, and it deliberately puts the malformed entry LAST, behind a
// perfectly good court.
//
// A guard that only checked CourtIDs[0] would pass a code-only assertion
// while still reserving court 1 and leaving a dangling Booking behind, so
// the assertions here are about observable state — zero ReserveCourt calls,
// zero persisted Games — not merely about the returned error. This mirrors
// TestScheduleGame_UnknownVenueFacilityRejectedBeforeReservingCourts (T8.3),
// which is the established pattern for "rejected before anything is
// reserved" in this package.
func TestScheduleGame_MalformedCourtIDRejectedBeforeReservingCourts(t *testing.T) {
	t.Parallel()

	for _, malformed := range malformedCourtIDs() {
		t.Run(malformed, func(t *testing.T) {
			t.Parallel()

			reservation := newFakeReservation()
			games := newFakeGameRepository()
			svc := app.NewService(app.ServiceOptions{
				IDs:           &sequentialIDs{},
				Games:         games,
				Registrations: newFakeRegistrationRepository(),
				Waitlist:      newFakeWaitlistRepository(),
				Matches:       newFakeMatchRepository(),
			})

			in := validInput(courtID(1), malformed)
			in.Range = mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")

			_, err := svc.ScheduleGame(context.Background(), in, reservation, newFakeFacilityLookup())
			if !errors.Is(err, domain.ErrMalformedCourtID) {
				t.Fatalf("ScheduleGame(court_ids=[valid, %q]) error = %v, want domain.ErrMalformedCourtID", malformed, err)
			}
			if len(reservation.reserveCalls) != 0 {
				t.Fatalf("reserved %v; a malformed court id must be rejected before ANY court is reserved", reservation.reserveCalls)
			}
			if len(games.games) != 0 {
				t.Fatalf("persisted %d games, want 0 — no partial state may survive a rejected call", len(games.games))
			}
		})
	}
}

// TestScheduleGame_MalformedCourtIDInFirstPositionRejected is the mirror of
// the test above: the malformed entry first, so neither element ordering can
// pass by accident.
func TestScheduleGame_MalformedCourtIDInFirstPositionRejected(t *testing.T) {
	t.Parallel()

	reservation := newFakeReservation()
	games := newFakeGameRepository()
	svc := app.NewService(app.ServiceOptions{
		IDs:           &sequentialIDs{},
		Games:         games,
		Registrations: newFakeRegistrationRepository(),
		Waitlist:      newFakeWaitlistRepository(),
		Matches:       newFakeMatchRepository(),
	})

	in := validInput("not-a-uuid", courtID(2))
	in.Range = mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")

	_, err := svc.ScheduleGame(context.Background(), in, reservation, newFakeFacilityLookup())
	if !errors.Is(err, domain.ErrMalformedCourtID) {
		t.Fatalf("error = %v, want domain.ErrMalformedCourtID", err)
	}
	if len(reservation.reserveCalls) != 0 {
		t.Fatalf("reserved %v, want no reservations", reservation.reserveCalls)
	}
	if len(games.games) != 0 {
		t.Fatalf("persisted %d games, want 0", len(games.games))
	}
}

// TestScheduleGame_EmptyCourtIDsStillReportsEmptyNotMalformed is the
// no-regression rail on #156's own baseline: an empty list was ALREADY a
// clean InvalidArgument via domain.ErrEmptyCourtIDs, and the new guard must
// not swallow that distinct, better-worded error. It also pins the guard's
// position relative to domain.NewGame — a guard placed before the
// constructor would answer an empty slice with nothing at all (no entries to
// iterate) or, worse, re-order the domain's own validation precedence.
func TestScheduleGame_EmptyCourtIDsStillReportsEmptyNotMalformed(t *testing.T) {
	t.Parallel()

	for name, courts := range map[string][]string{"nil": nil, "empty": {}} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			svc := app.NewService(app.ServiceOptions{
				IDs:           &sequentialIDs{},
				Games:         newFakeGameRepository(),
				Registrations: newFakeRegistrationRepository(),
				Waitlist:      newFakeWaitlistRepository(),
				Matches:       newFakeMatchRepository(),
			})

			in := validInput(courts...)
			in.Range = mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")

			_, err := svc.ScheduleGame(context.Background(), in, newFakeReservation(), newFakeFacilityLookup())
			if !errors.Is(err, domain.ErrEmptyCourtIDs) {
				t.Fatalf("error = %v, want domain.ErrEmptyCourtIDs", err)
			}
		})
	}
}

// TestScheduleGame_WellFormedCourtIDsStillReserve is the too-strict rail: a
// guard that rejected real court ids would break every Game in the system,
// and no amount of correct 400s on bad input would make that acceptable.
func TestScheduleGame_WellFormedCourtIDsStillReserve(t *testing.T) {
	t.Parallel()

	reservation := newFakeReservation()
	svc := app.NewService(app.ServiceOptions{
		IDs:           &sequentialIDs{},
		Games:         newFakeGameRepository(),
		Registrations: newFakeRegistrationRepository(),
		Waitlist:      newFakeWaitlistRepository(),
		Matches:       newFakeMatchRepository(),
	})

	// Upper-case hex is canonical too — pgtype accepts it — so a guard
	// written with a lower-case-only character class would fail here.
	in := validInput(courtID(1), "6BA7B810-9DAD-11D1-80B4-00C04FD430C8")
	in.Range = mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")

	if _, err := svc.ScheduleGame(context.Background(), in, reservation, newFakeFacilityLookup()); err != nil {
		t.Fatalf("well-formed court ids must still schedule, got err: %v", err)
	}
	if len(reservation.reserveCalls) != 2 {
		t.Fatalf("ReserveCourt called %d times, want 2", len(reservation.reserveCalls))
	}
}
