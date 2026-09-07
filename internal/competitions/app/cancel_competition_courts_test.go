package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nhuthuynh/white-label/internal/competitions/app"
	"github.com/nhuthuynh/white-label/internal/competitions/domain"
)

// T55.2 (issue #124's court half, mirrored onto Competitions) —
// CancelCompetition releases the courts the Competition was holding.
//
// #124 was opened against Social Play's Game.Cancel, but T16.3 found the
// identical gap on Competitions and fixed the Registrations/entries half in
// lockstep. The court half mirrors the same way, and is wired here for the
// same reason: a cancelled Competition holding courts nobody can rebook is
// the same bug whatever aggregate reserved them.
//
// As on the Social Play side, what this file pins is the wiring — that the
// cascade fires, names THIS Competition, acts as its host (the owner of
// competition-source Bookings since DECISION D1), and surfaces its failures.

// cancelCourtsFixture mirrors scheduleFixture but hands back the reservation
// so a test can inspect what the cascade asked it to do.
func cancelCourtsFixture(t *testing.T) (*app.Service, domain.Competition, *fakeReservation) {
	t.Helper()

	reservation := newFakeReservation()
	svc := newTestService(newFakeRepository(), reservation, newFakeFacilityLookup(), &fakeShareTokens{})

	in := validInput(t)
	in.Capacity = 16

	c, err := svc.ScheduleCompetition(context.Background(), in)
	if err != nil {
		t.Fatalf("fixture ScheduleCompetition failed: %v", err)
	}
	return svc, c, reservation
}

func TestCancelCompetition_ReleasesTheCourtsItHeld(t *testing.T) {
	t.Parallel()

	svc, c, reservation := cancelCourtsFixture(t)

	if _, err := svc.CancelCompetition(context.Background(), c.ID, c.HostID); err != nil {
		t.Fatalf("CancelCompetition: %v", err)
	}

	if len(reservation.releasedForReference) != 1 {
		t.Fatalf("cascade fired %d times, want exactly 1", len(reservation.releasedForReference))
	}
	if got := reservation.releasedForReference[0]; got != c.ID {
		t.Fatalf("cascade released reference %q, want this Competition's id %q", got, c.ID)
	}
	// The owner matters as much as the reference: since D1, releasing a
	// Booking requires being its owner, and competition-source Bookings are
	// owned by the host. Passing anything else would be refused in
	// production while still passing a test that only checked the reference.
	if got := reservation.releaseOwners[0]; got != c.HostID {
		t.Fatalf("cascade acted as %q, want the Competition's host %q", got, c.HostID)
	}
}

// A cascade failure is surfaced, not swallowed — the same treatment T16.3's
// entry cascade already gets.
func TestCancelCompetition_CourtReleaseFailureIsSurfaced(t *testing.T) {
	t.Parallel()

	svc, c, reservation := cancelCourtsFixture(t)

	boom := errors.New("booking service unavailable")
	reservation.releaseForReferenceErr = boom

	cancelled, err := svc.CancelCompetition(context.Background(), c.ID, c.HostID)
	if !errors.Is(err, boom) {
		t.Fatalf("CancelCompetition with a failing cascade = %v, want it to wrap %v", err, boom)
	}

	// Still returned as cancelled: the parent status write committed before
	// the cascade ran, deliberately (see CancelCompetition's doc comment).
	// A caller seeing this error knows the Competition is cancelled and the
	// courts may still be held — the repairable state.
	if cancelled.Status != domain.StatusCancelled {
		t.Fatalf("returned Status = %q, want cancelled even though the cascade failed", cancelled.Status)
	}
}

// A non-host never reaches the cascade: authorization runs first, so a
// refused cancellation must not release anybody's courts.
func TestCancelCompetition_NonHostNeverReachesTheCascade(t *testing.T) {
	t.Parallel()

	svc, c, reservation := cancelCourtsFixture(t)

	if _, err := svc.CancelCompetition(context.Background(), c.ID, "not-the-host"); !errors.Is(err, domain.ErrNotCompetitionHost) {
		t.Fatalf("CancelCompetition(non-host) = %v, want %v", err, domain.ErrNotCompetitionHost)
	}
	if len(reservation.releasedForReference) != 0 {
		t.Fatalf("cascade fired %d times for a refused cancellation, want 0", len(reservation.releasedForReference))
	}
}
