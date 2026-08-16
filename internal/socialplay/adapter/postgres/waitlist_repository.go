package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	socialplaydb "github.com/nhuthuynh/white-label/internal/gen/socialplaydb"
	"github.com/nhuthuynh/white-label/internal/socialplay/domain"
)

// WaitlistRepository implements port.WaitlistRepository (T6.6).
type WaitlistRepository struct {
	q *socialplaydb.Queries
}

func NewWaitlistRepository(pool *pgxpool.Pool) *WaitlistRepository {
	return &WaitlistRepository{q: socialplaydb.New(pool)}
}

// Create persists a new waiting WaitlistEntry. Per T6.6-loop-2's PM+PE
// review finding: e.Position (domain.JoinWaitlist's own count-based
// computation, from an unlocked app-layer read) is NOT trusted as
// authoritative here and is discarded, not echoed back — it exists only for
// in-memory/test implementations of this port that have no concurrent
// writers to race against. This Postgres implementation instead calls
// join_waitlist_entry (db/migrations/0009_socialplay_waitlist_join_position.sql),
// a FOR UPDATE-locked function mirroring PromoteNext's own race-closing
// pattern, and returns whatever Position it authoritatively assigns — see
// that migration's doc comment for the full race analysis (ordering-shaped,
// not distinctness-shaped, same conclusion 0008 reached for the promotion
// trigger).
func (r *WaitlistRepository) Create(ctx context.Context, e domain.WaitlistEntry) (domain.WaitlistEntry, error) {
	row, err := r.q.JoinWaitlistEntry(ctx, socialplaydb.JoinWaitlistEntryParams{
		PID:     mustUUID(e.ID),
		PGameID: mustUUID(e.GameID),
		// PPlayerID (T29.2, closing the Social Play third of #164):
		// waitlist_entries.player_id is now
		// `uuid NOT NULL REFERENCES identity_users (id)`
		// (db/migrations/0026_socialplay_identity_conformance.sql), and
		// join_waitlist_entry's own p_player_id parameter is now `uuid` too
		// (that migration recreates the function) — safe via mustUUID because
		// app.Service's callers only ever hand this a resolved User.ID.
		PPlayerID: mustUUID(e.PlayerID),
	})
	if err != nil {
		return domain.WaitlistEntry{}, translateWaitlistErr(err)
	}
	return waitlistEntryFromFields(row.ID, row.GameID, row.PlayerID, row.Position, row.Status, row.PromotedAt), nil
}

func (r *WaitlistRepository) GetByID(ctx context.Context, id string) (domain.WaitlistEntry, error) {
	row, err := r.q.GetWaitlistEntryByID(ctx, mustUUID(id))
	if err != nil {
		return domain.WaitlistEntry{}, translateWaitlistErr(err)
	}
	return waitlistEntryFromFields(row.ID, row.GameID, row.PlayerID, row.Position, row.Status, row.PromotedAt), nil
}

func (r *WaitlistRepository) ListForGame(ctx context.Context, gameID string) ([]domain.WaitlistEntry, error) {
	rows, err := r.q.ListWaitlistEntriesForGame(ctx, mustUUID(gameID))
	if err != nil {
		return nil, translateWaitlistErr(err)
	}
	out := make([]domain.WaitlistEntry, 0, len(rows))
	for _, row := range rows {
		out = append(out, waitlistEntryFromFields(row.ID, row.GameID, row.PlayerID, row.Position, row.Status, row.PromotedAt))
	}
	return out, nil
}

// PromoteNext calls promote_next_waiting (db/migrations/
// 0008_socialplay_waitlist_promotion.sql) — the DB-level operation that
// actually closes the promotion-ordering race, not this method itself; see
// that migration's doc comment. A zero-row result (pgx.ErrNoRows) means the
// game currently has no waiting entry, translated to
// domain.ErrNoWaitingEntries — an expected outcome, not an infrastructure
// failure.
func (r *WaitlistRepository) PromoteNext(ctx context.Context, gameID string, now time.Time) (domain.WaitlistEntry, error) {
	row, err := r.q.PromoteNextWaiting(ctx, socialplaydb.PromoteNextWaitingParams{
		PGameID: mustUUID(gameID),
		PNow:    toTimestamptz(now),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.WaitlistEntry{}, domain.ErrNoWaitingEntries
		}
		return domain.WaitlistEntry{}, translateWaitlistErr(err)
	}
	return waitlistEntryFromFields(row.ID, row.GameID, row.PlayerID, row.Position, row.Status, row.PromotedAt), nil
}

// ExpirePromotion runs the compare-and-swap ExpireWaitlistPromotion query
// (WHERE status = 'promoted'). A zero-row result is ambiguous on its own
// (id doesn't exist vs. exists but isn't promoted anymore), so a zero-row
// result triggers a follow-up GetByID to disambiguate which domain error to
// return — mirroring how this package already treats "not found" as a
// distinct case from other failures elsewhere (translateWaitlistErr).
func (r *WaitlistRepository) ExpirePromotion(ctx context.Context, id string, now time.Time) (domain.WaitlistEntry, error) {
	row, err := r.q.ExpireWaitlistPromotion(ctx, mustUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Disambiguate: does the row exist at all?
			if _, getErr := r.GetByID(ctx, id); getErr != nil {
				return domain.WaitlistEntry{}, getErr
			}
			return domain.WaitlistEntry{}, domain.ErrIllegalStatusTransition
		}
		return domain.WaitlistEntry{}, translateWaitlistErr(err)
	}
	return waitlistEntryFromFields(row.ID, row.GameID, row.PlayerID, row.Position, row.Status, row.PromotedAt), nil
}

// translateWaitlistErr maps infrastructure failures onto domain errors —
// the only errors allowed to cross out of this package (CLAUDE.md rule 5).
// A 23505 unique_violation on waitlist_entries_active_player_per_game_idx
// (db/migrations/0007_socialplay_waitlist.sql) becomes
// domain.ErrAlreadyOnWaitlist, the DB-level guard's counterpart to
// domain.JoinWaitlist's own pre-check (CLAUDE.md rule 4).
func translateWaitlistErr(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgUniqueViolation:
			return domain.ErrAlreadyOnWaitlist
		case pgCapacityExceeded:
			return domain.ErrGameFull
		}
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrWaitlistEntryNotFound
	}
	return fmt.Errorf("socialplay postgres adapter (waitlist): %w", err)
}

// waitlistEntryFromFields builds a domain.WaitlistEntry from the 6 columns
// every waitlist_entries query selects — see gameFromFields's doc comment
// in repository.go for why this pattern exists (sqlc emits a distinct Row
// type per query).
//
// playerID is pgtype.UUID, not string, as of T29.2 (closing the Social Play
// third of #164): waitlist_entries.player_id is now
// `uuid NOT NULL REFERENCES identity_users (id)`
// (db/migrations/0026_socialplay_identity_conformance.sql). Converted back to
// a Go string via .String() — mirrors gameFromFields'/registrationFromFields'
// identical conversion.
func waitlistEntryFromFields(id, gameID, playerID pgtype.UUID, position int32, status string, promotedAt pgtype.Timestamptz) domain.WaitlistEntry {
	e := domain.WaitlistEntry{
		ID:       id.String(),
		GameID:   gameID.String(),
		PlayerID: playerID.String(),
		Position: int(position),
		Status:   domain.WaitlistStatus(status),
	}
	if promotedAt.Valid {
		e.PromotedAt = promotedAt.Time
	}
	return e
}
