package port

import (
	"context"

	"github.com/nhuthuynh/white-label/internal/socialplay/domain"
)

// GameRepository is Social Play's persistence boundary for the Game
// aggregate (T5.1). The domain and app layers only ever see this interface;
// internal/socialplay/adapter/postgres implements it against the real
// database (T5.4), and tests implement it in-memory. Mirrors
// internal/booking/port.Repository's shape.
type GameRepository interface {
	// Create persists a new scheduled Game.
	Create(ctx context.Context, g domain.Game) (domain.Game, error)

	// GetByID returns a single Game, or domain.ErrGameNotFound.
	GetByID(ctx context.Context, id string) (domain.Game, error)
}

// RegistrationRepository is Social Play's persistence boundary for the
// Registration aggregate (T5.2).
type RegistrationRepository interface {
	// Create persists a new registration. Implementations backed by
	// Postgres rely on a unique constraint on (game_id, player_id) scoped
	// to non-cancelled rows as the authoritative guard against double
	// registration under concurrency; domain.Register's own-scope check is
	// a pre-check only, the same relationship booking.EnsureNoConflict has
	// with the EXCLUDE constraint (CLAUDE.md rule 4).
	Create(ctx context.Context, r domain.Registration) (domain.Registration, error)

	// GetByID returns a single registration, or domain.ErrRegistrationNotFound.
	GetByID(ctx context.Context, id string) (domain.Registration, error)

	// ListActiveForGame returns the non-cancelled registrations for gameID —
	// the capacity-safe read path domain.Register needs to re-derive the
	// active count/players before enforcing the capacity invariant.
	ListActiveForGame(ctx context.Context, gameID string) ([]domain.Registration, error)

	// Update persists changes to an existing registration (e.g. a status
	// transition from Cancel).
	Update(ctx context.Context, r domain.Registration) (domain.Registration, error)

	// UpdatePaymentStatus persists a PaymentStatus change for the
	// Registration identified by id (T6.5) — a dedicated, single-purpose
	// write path (mirroring UpdateRegistrationStatus/UpdateBookingStatus's
	// one-method-per-updatable-field convention) rather than overloading
	// Update, so the Postgres adapter's query for it is scoped to exactly
	// the payment_status column and can't accidentally clobber Status (or
	// vice versa). Returns domain.ErrRegistrationNotFound for an unknown id.
	UpdatePaymentStatus(ctx context.Context, id string, status domain.PaymentStatus) (domain.Registration, error)
}
