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

// pgUniqueViolation is the Postgres error code identity_users' PRIMARY KEY
// raises on a second CreateUser for the same caller-claimed id — a real,
// reachable case for this context specifically (see domain.
// ErrUserAlreadyExists's doc comment for why User's id differs from every
// other server-generated aggregate id in this codebase). Mirrors
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
		DisplayName:               u.DisplayName,
		Roles:                     rolesToStrings(u.Roles),
		SelfReportedStartingLevel: int16(u.SelfReportedStartingLevel),
	})
	if err != nil {
		return domain.User{}, translateErr(err)
	}
	return fromFields(row.ID, row.DisplayName, row.Roles, row.SelfReportedStartingLevel), nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (domain.User, error) {
	row, err := r.q.GetUserByID(ctx, mustUUID(id))
	if err != nil {
		return domain.User{}, translateErr(err)
	}
	return fromFields(row.ID, row.DisplayName, row.Roles, row.SelfReportedStartingLevel), nil
}

func (r *Repository) UpdateSelfReportedLevel(ctx context.Context, id string, level domain.SelfReportedStartingLevel) (domain.User, error) {
	row, err := r.q.UpdateSelfReportedLevel(ctx, identitydb.UpdateSelfReportedLevelParams{
		ID:                        mustUUID(id),
		SelfReportedStartingLevel: int16(level),
	})
	if err != nil {
		return domain.User{}, translateErr(err)
	}
	return fromFields(row.ID, row.DisplayName, row.Roles, row.SelfReportedStartingLevel), nil
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

// fromFields builds a domain.User from the 4 columns every identity_users
// query selects. sqlc generates a distinct Row struct per query
// (CreateUserRow, GetUserByIDRow, UpdateSelfReportedLevelRow, ...) rather
// than reusing a shared table model (CLAUDE.md gotcha) — a shared
// field-level converter avoids duplicating this mapping across every query,
// mirroring internal/facilities/adapter/postgres's fromFacilityFields.
func fromFields(id pgtype.UUID, displayName string, roles []string, level int16) domain.User {
	return domain.User{
		ID:                        id.String(),
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
		// UpdateSelfReportedLevel's uuidShape guard, and CreateUser's
		// domain.NewUser id-required check) has already validated id is
		// non-empty and, for the read/update paths, UUID-shaped before
		// reaching here — a malformed id here means an upstream invariant
		// was already violated, which is a programmer error, not a
		// runtime one — mirrors facilities' adapter.
		panic(fmt.Sprintf("identity postgres adapter: invalid uuid %q: %v", s, err))
	}
	return u
}
