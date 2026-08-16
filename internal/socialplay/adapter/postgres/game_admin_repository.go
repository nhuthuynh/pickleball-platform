package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	socialplaydb "github.com/nhuthuynh/white-label/internal/gen/socialplaydb"
	"github.com/nhuthuynh/white-label/internal/socialplay/domain"
)

// GameAdminRepository implements port.GameAdminRepository against the
// game_admins table (T14.4, partial fix for #168). Mirrors
// MatchRepository/GameRepository's shape in this package exactly.
type GameAdminRepository struct {
	q *socialplaydb.Queries
}

func NewGameAdminRepository(pool *pgxpool.Pool) *GameAdminRepository {
	return &GameAdminRepository{q: socialplaydb.New(pool)}
}

// Assign inserts a Game-Admin assignment. The composite primary key on
// (game_id, user_id) is the authoritative duplicate guard under concurrency —
// domain.AssignGameAdmin's own check is a fail-fast pre-check (CLAUDE.md
// rule 4) — and translateGameAdminErr turns its violation into the same
// domain.ErrAlreadyGameAdmin that pre-check returns.
//
// UserID and AssignedBy are NOW (T29.2, closing the Social Play third of
// #164) passed through mustUUID too — game_admins.user_id/assigned_by are
// `uuid NOT NULL REFERENCES identity_users (id)`
// (db/migrations/0026_socialplay_identity_conformance.sql), no longer the
// `text` subjects ADR-0014 §5a originally ruled for this table. Safe because
// every caller of app.Service.AssignGameAdmin now hands this a resolved
// User.ID on both fields: ActorUserID via the grpcapi handler's actor()
// funnel (which becomes AssignedBy), and AdminUserID via that same handler's
// resolveUserID step on the caller-supplied target subject (see
// adapter/grpcapi.Handler.resolveUserID's doc comment for why that second
// resolution step exists — this table's target-user field is the one place
// in this migration that needed it). GameID is a real uuid and is converted
// the same way every other Game id in this package is — app.Service applies
// its uuidShape guard before this method is ever reached, so mustUUID cannot
// be handed a malformed value from the wire.
func (r *GameAdminRepository) Assign(ctx context.Context, a domain.GameAdmin) (domain.GameAdmin, error) {
	row, err := r.q.AssignGameAdmin(ctx, socialplaydb.AssignGameAdminParams{
		GameID:     mustUUID(a.GameID),
		UserID:     mustUUID(a.UserID),
		AssignedBy: mustUUID(a.AssignedBy),
		AssignedAt: toTimestamptz(a.AssignedAt),
	})
	if err != nil {
		return domain.GameAdmin{}, translateGameAdminErr(err)
	}
	return gameAdminFromFields(row.GameID, row.UserID, row.AssignedBy, row.AssignedAt), nil
}

// Revoke deletes the assignment for (gameID, userID), answering
// domain.ErrGameAdminNotFound when the DELETE matched no row.
//
// The query RETURNINGs precisely so that distinction survives: a bare DELETE
// reports success whether or not anything was removed, and a revoke that
// silently removed nothing would confirm a Host's belief about who holds
// authority that the store does not support (see
// port.GameAdminRepository.Revoke's doc comment). pgx.ErrNoRows is the shape
// "deleted nothing" arrives in for a `:one` query.
func (r *GameAdminRepository) Revoke(ctx context.Context, gameID, userID string) error {
	// userID (T29.2): also now mustUUID, for the same reason Assign's does.
	// app.Service.RevokeGameAdmin guards against a blank userID before this
	// method is ever reached (see that method's own doc comment), so mustUUID
	// here is never handed the one value that would otherwise panic it.
	_, err := r.q.RevokeGameAdmin(ctx, socialplaydb.RevokeGameAdminParams{
		GameID: mustUUID(gameID),
		UserID: mustUUID(userID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrGameAdminNotFound
		}
		return translateGameAdminErr(err)
	}
	return nil
}

// ListGameAdmins returns every assignment for gameID, oldest first. An unknown
// Game yields an empty slice rather than an error — the app layer checks the
// Game's existence separately (see app.Service.ListGameAdmins) because "this
// Game has no admins" and "there is no such Game" are different facts that an
// authorization decision must not conflate.
func (r *GameAdminRepository) ListGameAdmins(ctx context.Context, gameID string) ([]domain.GameAdmin, error) {
	rows, err := r.q.ListGameAdmins(ctx, mustUUID(gameID))
	if err != nil {
		return nil, translateGameAdminErr(err)
	}
	out := make([]domain.GameAdmin, 0, len(rows))
	for _, row := range rows {
		out = append(out, gameAdminFromFields(row.GameID, row.UserID, row.AssignedBy, row.AssignedAt))
	}
	return out, nil
}

// translateGameAdminErr maps infrastructure failures onto domain errors
// (CLAUDE.md rule 5) — mirrors translateGameErr/translateMatchErr's shape.
//
//   - 23505 (unique violation) is game_admins' composite primary key firing:
//     the DB-level mirror of domain.ErrAlreadyGameAdmin, and the guard that
//     actually closes the two-concurrent-assignments race the domain
//     pre-check alone cannot.
//   - 23503 (foreign-key violation) is game_id's REFERENCES games (id)
//     firing. app.Service.AssignGameAdmin checks the Game exists first, so
//     this is defensive rather than routine — the same relationship
//     translateMatchErr's own 23503 case documents.
//
// pgx.ErrNoRows is deliberately NOT handled here: it means different things to
// different callers of this repository (nothing to revoke, vs. no such Game),
// so Revoke translates it itself rather than having one mapping guess.
func translateGameAdminErr(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgUniqueViolation:
			return domain.ErrAlreadyGameAdmin
		case pgForeignKeyViolation:
			return domain.ErrGameNotFound
		}
	}
	return fmt.Errorf("socialplay postgres adapter (game admins): %w", err)
}

// gameAdminFromFields builds a domain.GameAdmin from the 4 columns every
// game_admins query selects — mirrors gameFromFields/matchFromFields' pattern
// (CLAUDE.md gotcha: sqlc emits a distinct ...Row type per query, not a shared
// table model, so this converts from the shared columns rather than from one
// row struct).
//
// userID/assignedBy are pgtype.UUID, not string, as of T29.2 (closing the
// Social Play third of #164): game_admins.user_id/assigned_by are now
// `uuid NOT NULL REFERENCES identity_users (id)`
// (db/migrations/0026_socialplay_identity_conformance.sql). Converted back to
// Go strings via .String() — mirrors gameFromFields'/registrationFromFields'
// identical conversion.
func gameAdminFromFields(gameID, userID, assignedBy pgtype.UUID, assignedAt pgtype.Timestamptz) domain.GameAdmin {
	return domain.GameAdmin{
		GameID:     gameID.String(),
		UserID:     userID.String(),
		AssignedBy: assignedBy.String(),
		AssignedAt: assignedAt.Time,
	}
}
