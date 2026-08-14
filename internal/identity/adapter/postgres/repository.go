// Package postgres is the Identity/Users context's Postgres adapter. It
// implements port.Repository against sqlc-generated queries and translates
// Postgres errors into domain errors — CLAUDE.md rule 5. It only compiles
// after `make generate` has produced internal/gen/identitydb (see CLAUDE.md
// gotchas); that package is gitignored and not committed. Mirrors
// internal/facilities/adapter/postgres/repository.go's shape.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nhuthuynh/white-label/internal/identity/domain"

	identitydb "github.com/nhuthuynh/white-label/internal/gen/identitydb"
)

// pgUniqueViolation is the Postgres error code identity_users.subject's
// UNIQUE constraint raises on a second CreateUser for an already-registered
// subject (db/migrations/0019_identity_subject.sql). That constraint is the
// authoritative half of CLAUDE.md rule 4's dual enforcement for "one User
// per verified subject" — the app layer deliberately does NOT pre-check
// with a read-then-write, because two concurrent registrations for one
// subject would both see "not registered" and both insert. Translating the
// violation here (rule 5) is what keeps that race correct.
//
// Before T12.9 this same code fired on the PRIMARY KEY instead, for a
// caller-claimed id — the identity-squatting DoS HANDOFF.md's T10.2 bullet
// disclosed. The id is server-minted now, so a PK collision is as
// unreachable here as it is for every other aggregate. Mirrors
// internal/payments/adapter/postgres's identical constant.
const pgUniqueViolation = "23505"

type Repository struct {
	q *identitydb.Queries
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{q: identitydb.New(pool)}
}

func (r *Repository) Create(ctx context.Context, u domain.User) (domain.User, error) {
	row, err := r.q.CreateUser(ctx, identitydb.CreateUserParams{
		ID:                        mustUUID(u.ID),
		Subject:                   u.Subject,
		DisplayName:               u.DisplayName,
		Roles:                     rolesToStrings(u.Roles),
		SelfReportedStartingLevel: int16(u.SelfReportedStartingLevel),
	})
	if err != nil {
		return domain.User{}, translateErr(err)
	}
	return fromFields(row.ID, row.Subject, row.DisplayName, row.Roles, row.SelfReportedStartingLevel), nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (domain.User, error) {
	row, err := r.q.GetUserByID(ctx, mustUUID(id))
	if err != nil {
		return domain.User{}, translateErr(err)
	}
	return fromFields(row.ID, row.Subject, row.DisplayName, row.Roles, row.SelfReportedStartingLevel), nil
}

// GetBySubject resolves a verified IdP subject to its User (T12.9).
//
// No mustUUID here, deliberately: a subject is NOT a uuid — it is an
// arbitrary provider-specific string like `auth0|abc123` (see
// db/migrations/0019_identity_subject.sql) — so it goes to the `text`
// column as-is. Passing it through mustUUID would panic on every real
// subject the platform will ever see.
func (r *Repository) GetBySubject(ctx context.Context, subject string) (domain.User, error) {
	row, err := r.q.GetUserBySubject(ctx, subject)
	if err != nil {
		return domain.User{}, translateErr(err)
	}
	return fromFields(row.ID, row.Subject, row.DisplayName, row.Roles, row.SelfReportedStartingLevel), nil
}

func (r *Repository) UpdateSelfReportedLevel(ctx context.Context, id string, level domain.SelfReportedStartingLevel) (domain.User, error) {
	row, err := r.q.UpdateSelfReportedLevel(ctx, identitydb.UpdateSelfReportedLevelParams{
		ID:                        mustUUID(id),
		SelfReportedStartingLevel: int16(level),
	})
	if err != nil {
		return domain.User{}, translateErr(err)
	}
	return fromFields(row.ID, row.Subject, row.DisplayName, row.Roles, row.SelfReportedStartingLevel), nil
}

// translateErr maps infrastructure failures onto domain errors — the only
// errors allowed to cross out of this package (CLAUDE.md rule 5).
func translateErr(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
		return domain.ErrUserAlreadyExists
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrUserNotFound
	}
	return fmt.Errorf("identity postgres adapter: %w", err)
}

// fromFields builds a domain.User from the 5 columns every identity_users
// query selects. sqlc generates a distinct Row struct per query
// (CreateUserRow, GetUserByIDRow, UpdateSelfReportedLevelRow, ...) rather
// than reusing a shared table model (CLAUDE.md gotcha) — a shared
// field-level converter avoids duplicating this mapping across every query,
// mirroring internal/facilities/adapter/postgres's fromFacilityFields.
func fromFields(id pgtype.UUID, subject, displayName string, roles []string, level int16) domain.User {
	return domain.User{
		ID:                        id.String(),
		Subject:                   subject,
		DisplayName:               displayName,
		Roles:                     stringsToRoles(roles),
		SelfReportedStartingLevel: domain.SelfReportedStartingLevel(level),
	}
}

func rolesToStrings(roles []domain.Role) []string {
	out := make([]string, len(roles))
	for i, r := range roles {
		out[i] = string(r)
	}
	return out
}

func stringsToRoles(roles []string) []domain.Role {
	out := make([]domain.Role, len(roles))
	for i, r := range roles {
		out[i] = domain.Role(r)
	}
	return out
}

func mustUUID(s string) pgtype.UUID {
	var u pgtype.UUID
	if err := u.Scan(s); err != nil {
		// Every caller into this adapter (app.Service.GetUser/
		// UpdateSelfReportedLevel's uuidShape guard, and — since T12.9 —
		// CreateUser's server-minted id, which is well-formed by
		// construction) has already validated id is non-empty and
		// UUID-shaped before reaching here — a malformed id here means an upstream invariant
		// was already violated, which is a programmer error, not a
		// runtime one — mirrors facilities' adapter.
		panic(fmt.Sprintf("identity postgres adapter: invalid uuid %q: %v", s, err))
	}
	return u
}
