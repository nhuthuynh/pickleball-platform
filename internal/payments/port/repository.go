package port

import (
	"context"

	"github.com/nhuthuynh/white-label/internal/payments/domain"
)

// Repository is the Payments context's persistence boundary (T6.4). The
// domain and app layers only ever see this interface; adapter/postgres
// implements it against the real database, and tests implement it
// in-memory. Implementations must translate infrastructure errors into
// domain errors — most importantly a Postgres 23505 unique-violation on
// (payable_type, payable_id) becomes domain.ErrPaymentAlreadyRecorded
// (CLAUDE.md rule 5), mirroring how internal/booking/port.Repository's
// Create translates a 23P01 exclusion violation into
// domain.ErrCourtDoubleBooked.
type Repository interface {
	// Create persists a new Payment. The Postgres adapter's
	// UNIQUE (payable_type, payable_id) constraint is the authoritative
	// guard against a second Payment for the same payable action under
	// concurrency (CLAUDE.md rule 4); domain.EnsureOnePaymentPerPayable is
	// a pre-check only. A violation surfaces here as
	// domain.ErrPaymentAlreadyRecorded, not a raw pgconn.PgError.
	Create(ctx context.Context, p domain.Payment) (domain.Payment, error)

	// GetByID returns a single Payment, or domain.ErrPaymentNotFound.
	GetByID(ctx context.Context, id string) (domain.Payment, error)

	// Update persists changes to an existing Payment (e.g. a status
	// transition from MarkPaid or Refund).
	Update(ctx context.Context, p domain.Payment) (domain.Payment, error)
}
