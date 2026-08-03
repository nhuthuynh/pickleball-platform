package port

import (
	"context"
	"time"

	"github.com/nhuthuynh/white-label/internal/socialplay/domain"
)

// WaitlistRepository is Social Play's persistence boundary for the
// WaitlistEntry aggregate (T6.6). Mirrors GameRepository/
// RegistrationRepository's shape and role.
type WaitlistRepository interface {
	// Create persists a new waiting WaitlistEntry. Implementations backed
	// by Postgres rely on a unique constraint on (game_id, player_id)
	// scoped to active (waiting/promoted) rows as the authoritative guard
	// against a player joining the same Game's waitlist twice under
	// concurrency — domain.JoinWaitlist's own pre-check is a fast pre-check
	// only, the same relationship domain.Register has with
	// RegistrationRepository.Create (CLAUDE.md rule 4).
	//
	// e.Position (T6.6-loop-2): callers pass domain.JoinWaitlist's own
	// count-based computation, but that value is only safe to trust as-is
	// in a single-goroutine/in-memory implementation with no concurrent
	// writers. A Postgres implementation MUST NOT echo it back as
	// authoritative — two concurrent Create calls for the same Game can
	// otherwise be handed the same, or a misordered, Position (reproduced
	// directly: 30 concurrent calls against one Game produced 27 entries
	// all at Position 1). The Postgres adapter instead derives Position
	// itself inside a FOR UPDATE-locked function (join_waitlist_entry,
	// db/migrations/0009_socialplay_waitlist_join_position.sql) and returns
	// whatever it authoritatively assigns, discarding the caller's value —
	// mirroring how PromoteNext below never lets the caller pick which
	// entry gets promoted either.
	Create(ctx context.Context, e domain.WaitlistEntry) (domain.WaitlistEntry, error)

	// GetByID returns a single entry, or domain.ErrWaitlistEntryNotFound.
	GetByID(ctx context.Context, id string) (domain.WaitlistEntry, error)

	// ListForGame returns every entry for gameID, of any status, ordered by
	// Position ascending — the read path domain.JoinWaitlist needs to
	// derive the next Position and duplicate-join check before enforcing
	// its own invariants.
	ListForGame(ctx context.Context, gameID string) ([]domain.WaitlistEntry, error)

	// PromoteNext atomically promotes the oldest still-waiting entry for
	// gameID (setting PromotedAt to now), or returns
	// domain.ErrNoWaitingEntries if there isn't one. This is the DB-level
	// race-closing operation backing app.Service's auto-promotion
	// orchestration — see db/migrations/0008_socialplay_waitlist_promotion.sql
	// and the PR description's race analysis for why a plain
	// application-level "list then update" read/write pair isn't enough to
	// close this race under concurrent cancellations.
	PromoteNext(ctx context.Context, gameID string, now time.Time) (domain.WaitlistEntry, error)

	// ExpirePromotion transitions a promoted entry to expired, but only if
	// it is still promoted at the moment of the call — a compare-and-swap
	// UPDATE ... WHERE status = 'promoted', so a concurrent confirm (the
	// promoted player registering) or a second expiry sweep can't
	// double-act on the same entry. Returns domain.ErrWaitlistEntryNotFound
	// if id doesn't exist, or domain.ErrIllegalStatusTransition if it
	// exists but is no longer promoted.
	ExpirePromotion(ctx context.Context, id string, now time.Time) (domain.WaitlistEntry, error)
}
