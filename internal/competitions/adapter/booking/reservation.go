// Package booking is the one place Competitions code is allowed to import
// internal/booking/* (CLAUDE.md rule 5; mirrors
// internal/socialplay/adapter/booking's identical role for Social Play's
// side of the same context boundary): it's the adapter that implements
// internal/competitions/port.CourtReservation against the real Booking
// context's app.Service, not domain or app code.
// internal/competitions/domain and internal/competitions/app never see a
// bookingdomain.Booking or bookingapp.Service directly — see
// port.CourtReservation's doc comment for why that boundary exists and why
// the port is expressed purely in primitives.
package booking

import (
	"context"
	"errors"
	"fmt"
	"time"

	bookingapp "github.com/nhuthuynh/white-label/internal/booking/app"
	bookingdomain "github.com/nhuthuynh/white-label/internal/booking/domain"
	"github.com/nhuthuynh/white-label/internal/competitions/domain"
)

// Reservation implements port.CourtReservation by calling the Booking
// context's real app.Service.CreateBooking/CancelBooking, translating
// bookingdomain.ErrCourtDoubleBooked into Competitions' own, context-local
// domain.ErrCourtUnavailable (CLAUDE.md rule 5) — callers on the
// Competitions side of this boundary must never see a bookingdomain error
// type.
type Reservation struct {
	bookingSvc *bookingapp.Service
}

func NewReservation(bookingSvc *bookingapp.Service) *Reservation {
	return &Reservation{bookingSvc: bookingSvc}
}

// ReserveCourt reserves courtID for [start, end) against referenceID (a
// Competition's ID) as a competition-source Booking
// (bookingdomain.SourceCompetition, underlying value "competition"). The
// source is fixed here rather than accepted as a parameter, per
// port.CourtReservation's doc comment: this port exists to serve
// Competitions' use case, and a caller able to choose its own source could
// write a Booking that misreports what is holding the court.
//
// Nothing in the Booking context needed changing to support this. The
// `competition` source already exists in bookingdomain, and Booking's
// no-double-booking invariant (the Postgres EXCLUDE constraint, plus
// domain.EnsureNoConflict) is source-agnostic — so a Competition's
// reservations conflict with games, recurring hires, individual bookings,
// and other competitions automatically.
func (r *Reservation) ReserveCourt(ctx context.Context, courtID string, start, end time.Time, referenceID, ownerUserID string) (string, error) {
	rng, err := bookingdomain.NewTimeRange(start, end)
	if err != nil {
		// Competitions' own domain.NewCompetition already validated every
		// session's range (end after start) before ScheduleCompetition ever
		// calls this port, so this should be unreachable in practice — but
		// if it ever isn't, don't leak the bookingdomain error type across
		// the boundary.
		return "", fmt.Errorf("competitions booking adapter: invalid range for court %s: %s", courtID, err)
	}

	b, err := r.bookingSvc.CreateBooking(ctx, bookingapp.CreateBookingInput{
		CourtID:     courtID,
		Source:      bookingdomain.SourceCompetition,
		Range:       rng,
		ReferenceID: referenceID,
		OwnerUserID: ownerUserID,
	})
	if err != nil {
		if errors.Is(err, bookingdomain.ErrCourtDoubleBooked) {
			return "", domain.ErrCourtUnavailable
		}
		// T15.6 (closes #185). courtID names no court: either malformed
		// (bookingapp.CreateBooking's uuidShape guard — not reachable from
		// ScheduleCompetition, whose own T14.8 shape guard runs first, but
		// reachable from any other caller of this port) or well-formed and
		// naming no courts row, which only the FK on bookings.court_id can
		// detect and which Booking's Postgres adapter now translates rather
		// than leaving as a raw 23503.
		//
		// Translated to this context's own sentinel, never propagated as the
		// bookingdomain type (CLAUDE.md rule 5) — the same shape as the arm
		// above it, and the twin of
		// internal/socialplay/adapter/booking.Reservation.ReserveCourt's.
		// Without this arm the error falls to the %s-stripped wrap below,
		// reaches toStatus with no sentinel to match, and answers Internal:
		// the defect itself.
		if errors.Is(err, bookingdomain.ErrInvalidCourtReference) {
			return "", domain.ErrCourtNotFound
		}
		// Any other error (e.g. a wrapped infra failure) is stripped of its
		// original type before crossing the boundary — %s, not %w — so a
		// caller can never accidentally errors.Is() against a bookingdomain
		// sentinel from the Competitions side (CLAUDE.md rule 5). Same
		// deliberate non-%w wrapping as
		// internal/socialplay/adapter/booking.Reservation.ReserveCourt.
		return "", fmt.Errorf("competitions booking adapter: reserve court %s: %s", courtID, err)
	}
	return b.ID, nil
}

// ReleaseCourt is the compensating action for a ReserveCourt call that must
// be undone (app.Service.ScheduleCompetition's rollback path). It cancels
// the underlying Booking; best-effort per port.CourtReservation's doc
// comment, so its own failure is returned but the caller is expected not to
// let it mask the original error that triggered the rollback.
func (r *Reservation) ReleaseCourt(ctx context.Context, bookingID, ownerUserID string) error {
	if _, err := r.bookingSvc.CancelBooking(ctx, bookingID, ownerUserID); err != nil {
		return fmt.Errorf("competitions booking adapter: release booking %s: %s", bookingID, err)
	}
	return nil
}
