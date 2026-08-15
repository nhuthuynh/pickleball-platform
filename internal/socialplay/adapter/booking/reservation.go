// Package booking is the one place Social Play code is allowed to import
// internal/booking/* (CLAUDE.md rule 5 / docs/process/t5-sprint-plan.md's
// kickoff note on the socialplay/booking context boundary): it's the
// adapter boundary that implements internal/socialplay/port.CourtReservation
// against the real Booking context's app.Service, not domain or app code.
// internal/socialplay/domain and internal/socialplay/app never see a
// bookingdomain.Booking or bookingapp.Service directly.
package booking

import (
	"context"
	"errors"
	"fmt"
	"time"

	bookingapp "github.com/nhuthuynh/white-label/internal/booking/app"
	bookingdomain "github.com/nhuthuynh/white-label/internal/booking/domain"
	"github.com/nhuthuynh/white-label/internal/socialplay/domain"
)

// Reservation implements port.CourtReservation by calling the Booking
// context's real app.Service.CreateBooking/CancelBooking, translating
// bookingdomain.ErrCourtDoubleBooked into Social Play's own, context-local
// domain.ErrCourtUnavailable (CLAUDE.md rule 5) — callers on the Social
// Play side of this boundary must never see a bookingdomain error type.
type Reservation struct {
	bookingSvc *bookingapp.Service
}

func NewReservation(bookingSvc *bookingapp.Service) *Reservation {
	return &Reservation{bookingSvc: bookingSvc}
}

// ReserveCourt reserves courtID for [start, end) against referenceID (a
// Game's ID) as a game-source Booking (bookingdomain.SourceGame's
// underlying value, "game" — confirmed against
// internal/booking/domain/booking.go, per port.CourtReservation's doc
// comment).
func (r *Reservation) ReserveCourt(ctx context.Context, courtID string, start, end time.Time, referenceID string) (string, error) {
	rng, err := bookingdomain.NewTimeRange(start, end)
	if err != nil {
		// Social Play's own domain.NewGame already validated this range
		// (end after start) before ScheduleGame ever calls this port, so
		// this should be unreachable in practice — but if it ever isn't,
		// don't leak the bookingdomain error type across the boundary.
		return "", fmt.Errorf("socialplay booking adapter: invalid range for court %s: %s", courtID, err)
	}

	b, err := r.bookingSvc.CreateBooking(ctx, bookingapp.CreateBookingInput{
		CourtID:     courtID,
		Source:      bookingdomain.SourceGame,
		Range:       rng,
		ReferenceID: referenceID,
	})
	if err != nil {
		if errors.Is(err, bookingdomain.ErrCourtDoubleBooked) {
			return "", domain.ErrCourtUnavailable
		}
		// T15.6 (closes #185). courtID names no court: either it is
		// malformed (bookingapp.CreateBooking's uuidShape guard — not
		// reachable from ScheduleGame, whose own T14.8 shape guard runs
		// first, but reachable from any other caller of this port) or it is
		// well-formed and names no courts row, which only the FK on
		// bookings.court_id can detect and which Booking's Postgres adapter
		// now translates rather than leaving as a raw 23503.
		//
		// Translated to this context's own sentinel, never propagated as the
		// bookingdomain type (CLAUDE.md rule 5) — the same shape as the
		// ErrCourtDoubleBooked arm above it. Without this arm the error
		// falls to the %s-stripped wrap below, arrives at toStatus with no
		// sentinel to match, and answers Internal: the defect itself.
		if errors.Is(err, bookingdomain.ErrInvalidCourtReference) {
			return "", domain.ErrCourtNotFound
		}
		// Any other error (e.g. a wrapped infra failure) is stripped of its
		// original type before crossing the boundary — %s, not %w — so a
		// caller can never accidentally errors.Is() against a bookingdomain
		// sentinel from the Social Play side (CLAUDE.md rule 5).
		return "", fmt.Errorf("socialplay booking adapter: reserve court %s: %s", courtID, err)
	}
	return b.ID, nil
}

// ReleaseCourt is the compensating action for a ReserveCourt call that must
// be undone (app.Service.ScheduleGame's rollback path). It cancels the
// underlying Booking; best-effort per port.CourtReservation's doc comment,
// so its own failure is returned but callers are expected to not let it
// mask the original error that triggered the rollback.
func (r *Reservation) ReleaseCourt(ctx context.Context, bookingID string) error {
	if _, err := r.bookingSvc.CancelBooking(ctx, bookingID); err != nil {
		return fmt.Errorf("socialplay booking adapter: release booking %s: %s", bookingID, err)
	}
	return nil
}
