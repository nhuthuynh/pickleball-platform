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

	// GetByID returns a single booking, or domain.ErrBookingNotFound.
	GetByID(ctx context.Context, id string) (domain.Booking, error)

	// Update persists changes to an existing booking (e.g. a status
	// transition from Cancel).
	Update(ctx context.Context, b domain.Booking) (domain.Booking, error)
}
