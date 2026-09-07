package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nhuthuynh/white-label/internal/socialplay/domain"
)

// T55.2 (issue #124's court half) — CancelGame releases the courts the Game
// was holding.
//
// The release itself (that a freed court is genuinely re-reservable) is
// proven end to end against a real Booking service in
// internal/socialplay/adapter/booking/release_for_reference_test.go. What
// this file pins is the wiring, which is where the silent mistakes live:
// that the cascade fires at all, that it names THIS Game, that it acts as
// the Game's host (the owner of game-source Bookings since DECISION D1), and
// that its failure is surfaced rather than swallowed.

func TestCancelGame_ReleasesTheCourtsItHeld(t *testing.T) {
	ctx := context.Background()
	svc, g, _, _ := newMatchTestService(t)
	reservation := newFakeReservation()

	if _, err := svc.CancelGame(ctx, g.ID, g.HostID, reservation); err != nil {
		t.Fatalf("CancelGame: %v", err)
	}

	if len(reservation.releasedForReference) != 1 {
		t.Fatalf("cascade fired %d times, want exactly 1", len(reservation.releasedForReference))
	}
	if got := reservation.releasedForReference[0]; got != g.ID {
		t.Fatalf("cascade released reference %q, want this Game's id %q", got, g.ID)
	}
	// The owner matters as much as the reference: since D1, releasing a
	// Booking requires being its owner, and game-source Bookings are owned
	// by the host. Passing anything else here would be refused in
	// production while still passing a test that only checked the reference.
	if got := reservation.releaseOwners[0]; got != g.HostID {
		t.Fatalf("cascade acted as %q, want the Game's host %q", got, g.HostID)
	}
}

// A cascade failure is surfaced, not swallowed — same treatment the T16.3
// Registration cascade already gets, and for the same reason: courts still
// held for a cancelled Game is never an expected outcome.
func TestCancelGame_CourtReleaseFailureIsSurfaced(t *testing.T) {
	ctx := context.Background()
	svc, g, _, _ := newMatchTestService(t)
	reservation := newFakeReservation()

	boom := errors.New("booking service unavailable")
	reservation.releaseForReferenceErr = boom

	cancelled, err := svc.CancelGame(ctx, g.ID, g.HostID, reservation)
	if !errors.Is(err, boom) {
		t.Fatalf("CancelGame with a failing cascade = %v, want it to wrap %v", err, boom)
	}

	// The Game itself is still returned as cancelled: the parent status
	// write committed before the cascade ran, and that ordering is
	// deliberate (see CancelGame's doc comment). A caller seeing this error
	// knows the Game is cancelled and the courts may still be held, which is
	// the repairable state; the reverse ordering's failure mode is not.
	if cancelled.Status != domain.StatusCancelled {
		t.Fatalf("returned Game status = %q, want cancelled even though the cascade failed", cancelled.Status)
	}
}

// A caller who is not the host never reaches the cascade at all — the
// authorization check runs first, so a refused cancellation must not release
// anybody's courts.
func TestCancelGame_NonHostNeverReachesTheCascade(t *testing.T) {
	ctx := context.Background()
	svc, g, _, _ := newMatchTestService(t)
	reservation := newFakeReservation()

	if _, err := svc.CancelGame(ctx, g.ID, "not-the-host", reservation); !errors.Is(err, domain.ErrNotGameHost) {
		t.Fatalf("CancelGame(non-host) = %v, want %v", err, domain.ErrNotGameHost)
	}
	if len(reservation.releasedForReference) != 0 {
		t.Fatalf("cascade fired %d times for a refused cancellation, want 0", len(reservation.releasedForReference))
	}
}
