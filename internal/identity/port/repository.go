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
	//
	// Implementors must return domain.ErrUserAlreadyExists when u.Subject
	// is already registered to another User. That check is NOT the app
	// layer's to make with a read-then-write: two concurrent registrations
	// for one subject would both see "not registered" and both insert. The
	// Postgres implementation therefore relies on identity_users.subject's
	// UNIQUE constraint and translates the resulting 23505 (CLAUDE.md rules
	// 4 and 5) — the same authoritative-constraint discipline the Booking
	// context's EXCLUDE constraint gets. In-memory fakes must emulate the
	// rejection so tests exercise the same contract.
	Create(ctx context.Context, u domain.User) (domain.User, error)

	// GetByID returns a single User, or domain.ErrUserNotFound.
	GetByID(ctx context.Context, id string) (domain.User, error)

	// GetBySubject returns the User registered to a verified IdP subject,
	// or domain.ErrUserNotFound.
	//
	// This is the lookup that lets the grpcapi boundary translate an
	// auth.Principal into the actor value the domain already understands
	// (a User ID), which is what keeps domain and app free of any
	// internal/platform/auth import (T12 sprint plan A11 Ruling 3). The
	// subject is a verified claim by the time it reaches here, never a
	// value read off a request message.
	GetBySubject(ctx context.Context, subject string) (domain.User, error)

	// UpdateSelfReportedLevel persists a new SelfReportedStartingLevel for
	// id and returns the updated User, or domain.ErrUserNotFound. Callers
	// must have already checked domain.User.EnsureSelf before calling
	// this — this method only persists, it does not re-check authorization.
	UpdateSelfReportedLevel(ctx context.Context, id string, level domain.SelfReportedStartingLevel) (domain.User, error)
}
