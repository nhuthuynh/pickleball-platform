package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nhuthuynh/white-label/internal/competitions/domain"
	competitionsdb "github.com/nhuthuynh/white-label/internal/gen/competitionsdb"
)

// pgForeignKeyViolation is competition_admins.competition_id's
// REFERENCES competitions (id) firing. Declared here rather than alongside
// pgUniqueViolation/pgCapacityExceeded in repository.go because this file
// introduces the first code in this package that can provoke it — Repository's
// own writes either run inside Create's transaction (where the Competition is
// being inserted in the same statement batch) or reference a Competition the
// app layer has already loaded.
const pgForeignKeyViolation = "23503"

// CompetitionAdminRepository implements port.CompetitionAdminRepository
// against the competition_admins table (T15.3, partial fix for #168). Mirrors
// internal/socialplay/adapter/postgres.GameAdminRepository exactly, and
// Repository's shape in this package.
//
// A separate type from Repository, matching the separate port interface — see
// port.CompetitionAdminRepository's doc comment for why the interface is split
// rather than widened. It needs no pool of its own beyond the Queries handle:
// unlike Repository.Create, none of these three operations spans two tables,
// so none needs a transaction.
type CompetitionAdminRepository struct {
	q *competitionsdb.Queries
}

func NewCompetitionAdminRepository(pool *pgxpool.Pool) *CompetitionAdminRepository {
	return &CompetitionAdminRepository{q: competitionsdb.New(pool)}
}

// Assign inserts a Competition-Admin assignment. The composite primary key on
// (competition_id, user_id) is the authoritative duplicate guard under
// concurrency — domain.AssignCompetitionAdmin's own check is a fail-fast
// pre-check (CLAUDE.md rule 4) — and translateCompetitionAdminErr turns its
// violation into the same domain.ErrAlreadyCompetitionAdmin that pre-check
// returns.
//
// UserID and AssignedBy are real uuids as of T29.1
// (db/migrations/0025_competitions_identity_conformance.sql) — mustUUID,
// not nullableUUID, for both: a.UserID is resolved and non-blank-checked by
// the time domain.AssignCompetitionAdmin returns a value to store
// (ErrEmptyCompetitionAdminUserID rejects a blank one first), and
// a.AssignedBy is always the caller's own resolved actor from
// h.actor(ctx) — neither is ever blank on this path. CompetitionID is a
// real uuid and is converted the same way every other Competition id in this
// package is — app.Service applies its uuidShape guard before this method is
// ever reached, so mustUUID cannot be handed a malformed value from the wire.
//
// Before T29.1, UserID and AssignedBy were subjects held in `text` columns
// per ADR-0014 §5a (see db/migrations/0021_competitions_competition_admins.sql
// for the pre-migration shape) — parsing THOSE as uuids would have panicked
// on every real IdP subject, which is exactly why they were excluded from
// mustUUID before this ticket.
func (r *CompetitionAdminRepository) Assign(ctx context.Context, a domain.CompetitionAdmin) (domain.CompetitionAdmin, error) {
	row, err := r.q.AssignCompetitionAdmin(ctx, competitionsdb.AssignCompetitionAdminParams{
		CompetitionID: mustUUID(a.CompetitionID),
		UserID:        mustUUID(a.UserID),
		AssignedBy:    mustUUID(a.AssignedBy),
		AssignedAt:    toTimestamptz(a.AssignedAt),
	})
	if err != nil {
		return domain.CompetitionAdmin{}, translateCompetitionAdminErr(err)
	}
	return competitionAdminFromFields(row.CompetitionID, row.UserID, row.AssignedBy, row.AssignedAt), nil
}

// Revoke deletes the assignment for (competitionID, userID), answering
// domain.ErrCompetitionAdminNotFound when the DELETE matched no row.
//
// The query RETURNINGs precisely so that distinction survives: a bare DELETE
// reports success whether or not anything was removed, and a revoke that
// silently removed nothing would confirm a Host's belief about who holds
// authority that the store does not support (see
// port.CompetitionAdminRepository.Revoke's doc comment). pgx.ErrNoRows is the
// shape "deleted nothing" arrives in for a `:one` query.
//
// userID uses nullableUUID, NOT mustUUID, unlike Assign's identical-looking
// parameter above — deliberately: the grpcapi handler's
// resolveTargetUserID passes an EMPTY string through here unresolved when
// the wire user_id was blank (see that method's own doc comment), and
// mustUUID panics on "". nullableUUID converts a blank userID into an
// explicitly-invalid (Valid: false) parameter, which the query's `user_id =
// $2` comparison then matches against nothing — the identical "blank never
// matches a real row" answer the pre-migration `text` column already gave,
// now expressed as a real NULL-shaped comparison instead of an empty-string
// one.
func (r *CompetitionAdminRepository) Revoke(ctx context.Context, competitionID, userID string) error {
	_, err := r.q.RevokeCompetitionAdmin(ctx, competitionsdb.RevokeCompetitionAdminParams{
		CompetitionID: mustUUID(competitionID),
		UserID:        nullableUUID(userID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrCompetitionAdminNotFound
		}
		return translateCompetitionAdminErr(err)
	}
	return nil
}

// ListCompetitionAdmins returns every assignment for competitionID, oldest
// first. An unknown Competition yields an empty slice rather than an error —
// the app layer checks the Competition's existence separately (see
// app.Service.ListCompetitionAdmins) because "this Competition has no admins"
// and "there is no such Competition" are different facts that an authorization
// decision must not conflate.
func (r *CompetitionAdminRepository) ListCompetitionAdmins(ctx context.Context, competitionID string) ([]domain.CompetitionAdmin, error) {
	rows, err := r.q.ListCompetitionAdmins(ctx, mustUUID(competitionID))
	if err != nil {
		return nil, translateCompetitionAdminErr(err)
	}
	out := make([]domain.CompetitionAdmin, 0, len(rows))
	for _, row := range rows {
		out = append(out, competitionAdminFromFields(row.CompetitionID, row.UserID, row.AssignedBy, row.AssignedAt))
	}
	return out, nil
}

// translateCompetitionAdminErr maps infrastructure failures onto domain errors
// (CLAUDE.md rule 5) — mirrors translateErr/translateEntryErr's shape in this
// package and translateGameAdminErr's in Social Play.
//
//   - 23505 (unique violation) is competition_admins' composite primary key
//     firing: the DB-level mirror of domain.ErrAlreadyCompetitionAdmin, and the
//     guard that actually closes the two-concurrent-assignments race the domain
//     pre-check alone cannot.
//   - 23503 (foreign-key violation) is competition_id's REFERENCES
//     competitions (id) firing. app.Service.AssignCompetitionAdmin checks the
//     Competition exists first, so this is defensive rather than routine.
//
// pgx.ErrNoRows is deliberately NOT handled here: it means different things to
// different callers of this repository (nothing to revoke, vs. no such
// Competition), so Revoke translates it itself rather than having one mapping
// guess.
func translateCompetitionAdminErr(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgUniqueViolation:
			return domain.ErrAlreadyCompetitionAdmin
		case pgForeignKeyViolation:
			return domain.ErrCompetitionNotFound
		}
	}
	return fmt.Errorf("competitions postgres adapter (competition admins): %w", err)
}

// competitionAdminFromFields builds a domain.CompetitionAdmin from the 4
// columns every competition_admins query selects — the per-query "fromFields"
// pattern this package already uses (CLAUDE.md gotcha: sqlc emits a distinct
// ...Row type per query, not a shared table model, so this converts from the
// shared columns rather than from one row struct).
//
// userID and assignedBy are pgtype.UUID as of T29.1, converted via
// fromNullableUUID for the identical orphan-tolerance reason
// repository.go's competitionFromFields documents for HostID.
func competitionAdminFromFields(competitionID pgtype.UUID, userID, assignedBy pgtype.UUID, assignedAt pgtype.Timestamptz) domain.CompetitionAdmin {
	return domain.CompetitionAdmin{
		CompetitionID: competitionID.String(),
		UserID:        fromNullableUUID(userID),
		AssignedBy:    fromNullableUUID(assignedBy),
		AssignedAt:    assignedAt.Time,
	}
}
