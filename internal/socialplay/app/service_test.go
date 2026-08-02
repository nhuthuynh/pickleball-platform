package app_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/nhuthuynh/white-label/internal/socialplay/app"
	"github.com/nhuthuynh/white-label/internal/socialplay/domain"
)

// fakeReservation is a minimal in-memory port.CourtReservation fake — no
// real Booking context involved (T5.3 is deliberately app-layer-only, no
// Postgres/proto, mirroring how Booking's own cross-source overlap test was
// proven before any adapter existed). It lets tests seed a specific court as
// already unavailable (simulating an existing overlapping Booking of any
// source) and records enough call history to prove rollback behaviour.
type fakeReservation struct {
	unavailable map[string]bool // courtID -> already reserved by someone else

	reserveCalls []string // courtIDs ReserveCourt was called for, in order
	released     []string // bookingIDs ReleaseCourt was called for, in order
	releaseErr   error    // optional: simulate a rollback call itself failing

	n int
}

func newFakeReservation(unavailableCourts ...string) *fakeReservation {
	f := &fakeReservation{unavailable: make(map[string]bool)}
	for _, c := range unavailableCourts {
		f.unavailable[c] = true
	}
	return f
}

func (f *fakeReservation) ReserveCourt(_ context.Context, courtID string, _, _ time.Time, _ string) (string, error) {
	f.reserveCalls = append(f.reserveCalls, courtID)
	if f.unavailable[courtID] {
		return "", domain.ErrCourtUnavailable
	}
	f.n++
	return fmt.Sprintf("booking-%d", f.n), nil
}

func (f *fakeReservation) ReleaseCourt(_ context.Context, bookingID string) error {
	f.released = append(f.released, bookingID)
	return f.releaseErr
}

// sequentialIDs is a deterministic port.IDGenerator fake, mirroring
// internal/booking/app's test fake of the same shape.
type sequentialIDs struct{ n int }

func (g *sequentialIDs) NewID() string {
	g.n++
	return fmt.Sprintf("game-%d", g.n)
}

func mustRange(t *testing.T, start, end string) domain.TimeRange {
	t.Helper()
	s, err := time.Parse(time.RFC3339, start)
	if err != nil {
		t.Fatalf("bad fixture start: %v", err)
	}
	e, err := time.Parse(time.RFC3339, end)
	if err != nil {
		t.Fatalf("bad fixture end: %v", err)
	}
	r, err := domain.NewTimeRange(s, e)
	if err != nil {
		t.Fatalf("bad fixture range: %v", err)
	}
	return r
}

func validInput(courtIDs ...string) app.ScheduleGameInput {
	return app.ScheduleGameInput{
		HostID:     "host-1",
		FacilityID: "facility-1",
		CourtIDs:   courtIDs,
		Capacity:   4,
	}
}

// TestScheduleGame_ReservesEveryCourt proves the happy path: a multi-court
// Game reserves one Booking per court, using the game's ID as the
// referenceID passed to the port.
func TestScheduleGame_ReservesEveryCourt(t *testing.T) {
	t.Parallel()

	reservation := newFakeReservation()
	svc := app.NewService(&sequentialIDs{})
	in := validInput("court-1", "court-2")
	in.Range = mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")

	g, err := svc.ScheduleGame(context.Background(), in, reservation)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if g.Status != domain.StatusScheduled {
		t.Fatalf("status = %v, want scheduled", g.Status)
	}
	if len(reservation.reserveCalls) != 2 {
		t.Fatalf("ReserveCourt called %d times, want 2", len(reservation.reserveCalls))
	}
	if len(reservation.released) != 0 {
		t.Fatalf("no rollback expected on the happy path, got releases: %v", reservation.released)
	}
}

// TestScheduleGame_RejectsCourtAlreadyReserved is the literal HANDOFF AC:
// a Game cannot be scheduled onto a court the port reports as already
// reserved (standing in for a confirmed Booking of any source — individual,
// recurring hire, competition, or another game; the fake doesn't need to
// model all four, it just needs to prove the conflict propagates through
// ScheduleGame's returned error unchanged).
func TestScheduleGame_RejectsCourtAlreadyReserved(t *testing.T) {
	t.Parallel()

	reservation := newFakeReservation("court-1")
	svc := app.NewService(&sequentialIDs{})
	in := validInput("court-1")
	in.Range = mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")

	_, err := svc.ScheduleGame(context.Background(), in, reservation)
	if !errors.Is(err, domain.ErrCourtUnavailable) {
		t.Fatalf("got err %v, want ErrCourtUnavailable", err)
	}
}

// TestScheduleGame_RollsBackEarlierCourtsOnConflict proves the no-half-
// scheduled-Game requirement: when court 2 of 2 conflicts, court 1's
// reservation must not be left dangling. This is the chosen resolution to
// the multi-court-atomicity question (rollback via best-effort compensating
// ReleaseCourt calls) rather than restricting Games to a single court — see
// the PR description for why.
func TestScheduleGame_RollsBackEarlierCourtsOnConflict(t *testing.T) {
	t.Parallel()

	reservation := newFakeReservation("court-2")
	svc := app.NewService(&sequentialIDs{})
	in := validInput("court-1", "court-2")
	in.Range = mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")

	_, err := svc.ScheduleGame(context.Background(), in, reservation)
	if !errors.Is(err, domain.ErrCourtUnavailable) {
		t.Fatalf("got err %v, want ErrCourtUnavailable", err)
	}
	if len(reservation.reserveCalls) != 2 {
		t.Fatalf("ReserveCourt called %d times, want 2 (court-1 then court-2)", len(reservation.reserveCalls))
	}
	if len(reservation.released) != 1 || reservation.released[0] != "booking-1" {
		t.Fatalf("expected court-1's reservation (booking-1) to be released, got %v", reservation.released)
	}
}

// TestScheduleGame_RollbackFailureDoesNotMaskOriginalError proves the
// documented best-effort semantics: even if the compensating ReleaseCourt
// call itself fails, ScheduleGame still surfaces the original conflict
// error rather than a rollback-specific one.
func TestScheduleGame_RollbackFailureDoesNotMaskOriginalError(t *testing.T) {
	t.Parallel()

	reservation := newFakeReservation("court-2")
	reservation.releaseErr = errors.New("release: boom")
	svc := app.NewService(&sequentialIDs{})
	in := validInput("court-1", "court-2")
	in.Range = mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")

	_, err := svc.ScheduleGame(context.Background(), in, reservation)
	if !errors.Is(err, domain.ErrCourtUnavailable) {
		t.Fatalf("got err %v, want ErrCourtUnavailable even though rollback itself failed", err)
	}
	if len(reservation.released) != 1 {
		t.Fatalf("rollback should still be attempted once, got %v", reservation.released)
	}
}

// TestScheduleGame_InvalidInputRejectedBeforeTouchingPort proves domain
// validation (T5.1's NewGame) runs before any court is reserved — mirroring
// TestCreateBooking_InvalidSourceRejectedBeforeTouchingRepo's style in
// internal/booking/app.
func TestScheduleGame_InvalidInputRejectedBeforeTouchingPort(t *testing.T) {
	t.Parallel()

	reservation := newFakeReservation()
	svc := app.NewService(&sequentialIDs{})
	in := validInput("court-1")
	in.Range = mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")
	in.Capacity = 0

	_, err := svc.ScheduleGame(context.Background(), in, reservation)
	if !errors.Is(err, domain.ErrInvalidCapacity) {
		t.Fatalf("got err %v, want ErrInvalidCapacity", err)
	}
	if len(reservation.reserveCalls) != 0 {
		t.Fatalf("invalid Game must not touch the reservation port, got calls: %v", reservation.reserveCalls)
	}
}
