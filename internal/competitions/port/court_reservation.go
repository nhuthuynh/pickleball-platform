package port

import (
	"context"
	"time"
)

// CourtReservation is Competitions' outbound port for turning a scheduled
// Competition's (session, court) pairs into real court reservations in the
// Booking bounded context. It is expressed entirely in primitive types
// (court ID, start/end time.Time, a reference ID) so that
// internal/competitions/domain and internal/competitions/app never import
// internal/booking/domain or internal/booking/app — the context map
// (docs/agent-operating-handbook.md A1) requires that dependency to run
// through a port, not a direct import. Only the adapter that implements
// this interface (internal/competitions/adapter/booking) is allowed to see
// Booking's real types.
//
// This is deliberately Competitions' *own* port rather than a reuse of
// internal/socialplay/port.CourtReservation, whose method set is
// identical. Sharing one interface across two bounded contexts would make
// Competitions depend on Social Play for a reason that has nothing to do
// with Social Play, and would mean the two contexts' court-reservation
// needs could never diverge without breaking each other — the same
// shared-kernel coupling A1 forbids, and the same reasoning
// internal/competitions/domain.TimeRange and .Money already apply to their
// Social Play namesakes.
//
// Every reservation made through this port is implicitly Booking-source
// "competition" (bookingdomain.SourceCompetition's underlying value,
// "competition" — confirmed against internal/booking/domain/booking.go).
// The port is scoped to Competitions' own use case, so the source is not a
// parameter callers choose: a caller that could pick its own source could
// write a `game`-source Booking from the Competitions context, and the
// source field would stop being a trustworthy answer to "what is holding
// this court?".
//
// Note that the Booking context needs no change to support this: the
// `competition` source already exists and is already covered by Booking's
// cross-source conflict tests, so a Competition inherits the
// no-double-booking invariant against games, hires, and individual bookings
// simply by reserving through this port.
type CourtReservation interface {
	// ReserveCourt reserves courtID for the half-open range [start, end)
	// against referenceID (a Competition's ID). It returns the ID of the
	// Booking created to hold the reservation. If the court is already
	// reserved for an overlapping time — by any Booking source (individual,
	// recurring hire, game, or another competition) — implementations
	// return an error that satisfies errors.Is(err,
	// domain.ErrCourtUnavailable); callers must not depend on any other
	// error type crossing this boundary.
	//
	// One Competition produces one call per (session, court) pair: a
	// Competition runs across dates, so a single ScheduleCompetition can
	// mean many more reservations than Social Play's single-range
	// equivalent. Each call is independent — there is no "reserve this
	// whole competition atomically" operation, which is why the caller owns
	// the compensating rollback described on ReleaseCourt.
	// ownerUserID is the **User.ID** (uuid) the Booking is created as owned
	// by — the Competition's host. Required since DECISION D1 (ADR-0015
	// option (a)): every Booking has exactly one owner, and only that owner
	// may cancel it. Passing the host here is what makes the compensating
	// ReleaseCourt below able to cancel the Booking it just made; a
	// reservation owned by nobody could not be rolled back at all.
	ReserveCourt(ctx context.Context, courtID string, start, end time.Time, referenceID, ownerUserID string) (bookingID string, err error)

	// ReleaseCourt is the compensating action for a ReserveCourt call that
	// must be undone because a later (session, court) pair in the same
	// Competition failed, or because persisting the Competition itself
	// failed after every court reserved (app.Service.ScheduleCompetition's
	// rollback path — see its doc comment).
	//
	// Implementations should treat this as best-effort: ScheduleCompetition
	// surfaces the original reservation failure to its caller regardless of
	// whether ReleaseCourt itself succeeds, since the original conflict is
	// the actionable error and a rollback failure shouldn't mask it. The
	// cost of that choice is that a failed release can leave a dangling
	// Booking holding a court for a Competition that doesn't exist; that
	// residue is a known gap, not an oversight, and the real adapter is
	// where logging/alerting on it belongs.
	//
	// ownerUserID must be the same owner the matching ReserveCourt call
	// passed: since D1, cancelling a Booking requires being its owner, so a
	// rollback that supplied a different actor would be refused and the
	// court would stay held.
	ReleaseCourt(ctx context.Context, bookingID, ownerUserID string) error
}
