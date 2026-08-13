package port

import (
	"context"

	"github.com/nhuthuynh/white-label/internal/identity/domain"
)

// Repository is the Identity/Users context's persistence boundary. The
// domain and app layers only ever see this interface; adapter/postgres
// implements it against the real database, and tests implement it
// in-memory. Mirrors internal/facilities/port.Repository's shape.
// Implementors must translate infrastructure errors into domain errors
// (CLAUDE.md rule 5) — e.g. a Postgres pgx.ErrNoRows on a user lookup
// becomes domain.ErrUserNotFound.
type Repository interface {
	// Create persists a new User.
	Create(ctx context.Context, u domain.User) (domain.User, error)

	// GetByID returns a single User, or domain.ErrUserNotFound.
	GetByID(ctx context.Context, id string) (domain.User, error)

	// UpdateSelfReportedLevel persists a new SelfReportedStartingLevel for
	// id and returns the updated User, or domain.ErrUserNotFound. Callers
	// must have already checked domain.User.EnsureSelf before calling
	// this — this method only persists, it does not re-check authorization.
	UpdateSelfReportedLevel(ctx context.Context, id string, level domain.SelfReportedStartingLevel) (domain.User, error)
}
