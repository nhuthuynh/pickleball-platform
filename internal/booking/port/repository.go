package port

import (
	"context"

	"github.com/nhuthuynh/white-label/internal/booking/domain"
)

// Repository is the Booking context's persistence boundary. The domain and
// app layers only ever see this interface; adapter/postgres implements it
// against the real database, and tests implement it in-memory. Implementors
// must translate infrastructure errors into domain errors (e.g. a Postgres
// 23P01 exclusion-constraint violation becomes domain.ErrCourtDoubleBooked).
type Repository interface {
	// Create persists a new confirmed booking. Implementations backed by
	// Postgres rely on the EXCLUDE constraint as the authoritative guard
	// under concurrency; the domain's EnsureNoConflict is a pre-check only.
	Create(ctx context.Context, b domain.Booking) (domain.Booking, error)

	// ListActiveForCourt returns non-cancelled bookings on courtID that
	// overlap the given range, used both for conflict pre-checks and for
	// the ListCourtBookings use case (T2).
	ListActiveForCourt(ctx context.Context, courtID string, r domain.TimeRange) ([]domain.Booking, error)

	// ListActiveForReference returns non-cancelled bookings made against
	// referenceID — a Game's, Competition's, or RecurringHireTemplate's id.
	//
	// It exists for the cancellation cascade (#124): releasing the courts a
	// cancelled Game held means finding the Bookings that hold them, and
	// reference_id is the only link between the two. An empty referenceID
	// must match nothing (reference_id is empty for plain individual
	// bookings, so matching it would select the whole table); an unknown one
	// is not an error, it simply has no bookings.
	ListActiveForReference(ctx context.Context, referenceID string) ([]domain.Booking, error)

	// ListForOwner returns every booking owned by ownerUserID — the
	// player-facing "my bookings" read (T67.1, part of issue #353), keyed on
	// the bookings.owner_user_id column 0027 added and served by the
	// bookings_owner_user_id_idx index that migration created for exactly
	// this query ("which bookings does this user own", in its own words).
	//
	// **Cancelled bookings are INCLUDED, which is deliberately the opposite
	// of ListActiveForCourt above.** The two reads answer different
	// questions and the difference is not an inconsistency: the court read
	// answers "is this slot free?", where a cancelled booking is noise, while
	// this one answers "what did I book?", where a cancelled booking is
	// information the user needs. A list that dropped them would leave a
	// player unable to tell "I cancelled that" from "my booking was lost" —
	// and the second is the bug #353 was filed about, so a read that can
	// masquerade as it is not an acceptable answer to it.
	//
	// Ordered by starts_at ascending, ties broken by id so the order is
	// total rather than merely sorted. An actor with no bookings is an empty
	// slice and no error; an unknown owner is not an error here, because
	// "you own nothing" and "you do not exist" are the same answer to this
	// question and distinguishing them would make the read an enumeration
	// oracle.
	ListForOwner(ctx context.Context, ownerUserID string) ([]domain.Booking, error)

	// GetByID returns a single booking, or domain.ErrBookingNotFound.
	GetByID(ctx context.Context, id string) (domain.Booking, error)

	// Update persists changes to an existing booking (e.g. a status
	// transition from Cancel).
	Update(ctx context.Context, b domain.Booking) (domain.Booking, error)
}
