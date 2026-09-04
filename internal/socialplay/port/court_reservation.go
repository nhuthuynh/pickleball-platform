package port

import (
	"context"
	"time"
)

// CourtReservation is Social Play's outbound port for turning a scheduled
// Game's courts into real court reservations in the Booking bounded context.
// It is expressed entirely in primitive types (court ID, start/end
// time.Time, a reference ID) so that internal/socialplay/domain and
// internal/socialplay/app never import internal/booking/domain or
// internal/booking/app — the context map (agent-operating-handbook.md A1)
// requires that dependency to run through a port, not a direct import. Only
// the adapter that implements this interface (internal/socialplay/adapter/
// booking, built in T5.4) is allowed to see Booking's real types.
//
// Every reservation made through this port is implicitly Booking-source
// "game" (bookingdomain.SourceGame's underlying value, "game" — confirmed
// against internal/booking/domain/booking.go) — the port is scoped to Social
// Play's own use case, so the source isn't a parameter callers choose.
type CourtReservation interface {
	// ReserveCourt reserves courtID for the half-open range [start, end)
	// against referenceID (a Game's ID). It returns the ID of the Booking
	// created to hold the reservation. If the court is already reserved for
	// an overlapping time — by any Booking source (individual, recurring
	// hire, competition, or another game) — implementations return an error
	// that satisfies errors.Is(err, domain.ErrCourtUnavailable); callers
	// must not depend on any other error type crossing this boundary.
	// ownerUserID is the **User.ID** (uuid) the Booking is created as owned
	// by — the Game's host. Required since DECISION D1 (ADR-0015
	// option (a)): every Booking has exactly one owner, and only that owner
	// may cancel it. Passing the host here is what makes the compensating
	// ReleaseCourt below able to cancel the Booking it just made; a
	// reservation owned by nobody could not be rolled back at all.
	ReserveCourt(ctx context.Context, courtID string, start, end time.Time, referenceID, ownerUserID string) (bookingID string, err error)

	// ReleaseCourt is the compensating action for a ReserveCourt call that
	// must be undone because a later court in the same multi-court
	// reservation failed (app.Service.ScheduleGame's rollback path — see
	// its doc comment). Implementations should treat this as best-effort:
	// ScheduleGame surfaces the original reservation failure to its caller
	// regardless of whether ReleaseCourt itself succeeds, since the
	// original conflict is the actionable error and a rollback failure
	// shouldn't mask it.
	//
	// ownerUserID must be the same owner the matching ReserveCourt call
	// passed: since D1, cancelling a Booking requires being its owner, so a
	// rollback that supplied a different actor would be refused and the
	// court would stay held.
	ReleaseCourt(ctx context.Context, bookingID, ownerUserID string) error
}
