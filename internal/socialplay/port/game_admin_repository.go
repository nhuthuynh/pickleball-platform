package port

import (
	"context"

	"github.com/nhuthuynh/white-label/internal/socialplay/domain"
)

// GameAdminRepository is Social Play's persistence boundary for
// domain.GameAdmin (T14.4, partial fix for #168) — the durable store that
// replaces the caller-supplied `assigned_game_admin_user_ids` list this
// codebase used to have no alternative to. Mirrors
// GameRepository/RegistrationRepository/MatchRepository's shape exactly: the
// domain and app layers only ever see this interface,
// internal/socialplay/adapter/postgres implements it against the real
// database, and tests implement it in-memory.
//
// # Why the read is a list rather than a membership test
//
// The sprint plan (T14.4 instruction 5, resolving §A13 GAP A) requires a read
// path here and asks for a stated choice between `IsGameAdmin(ctx, gameID,
// userID) (bool, error)` and `ListGameAdmins(ctx, gameID)`. This interface
// ships the list, for three reasons:
//
//  1. **It satisfies both shapes the consumer's own ticket describes.** T14.5
//     instruction 1 says its domain change "receives the resolved set or a
//     resolved boolean". A boolean is derivable from the set
//     (domain.HasGameAdmin does exactly that, and is exported for it); the set
//     is not derivable from a boolean. Picking the narrower read would have
//     forced T14.5 to add the wider one, which is the shape of gap this
//     instruction exists to close, not to move by one ticket.
//  2. **The write path needs it anyway.** AssignGameAdmin's
//     already-assigned pre-check (domain.AssignGameAdmin's `existing`
//     argument) reads the Game's current assignments, exactly as
//     RegisterForGame's capacity pre-check reads its current registrations.
//     One read serving both the write pre-check and the consumer's
//     authorization resolution is one query to keep correct, not two.
//  3. **The membership rule stays in the domain.** A repository-level
//     IsGameAdmin would push the blank-entry guard (domain.HasGameAdmin's
//     `a.UserID != ""` check, mirroring Game.EnsureHostOrGameAdmin's) into
//     SQL, where CLAUDE.md rule 2 says it does not belong and where no unit
//     test without Docker can reach it.
//
// The cost — loading a Game's admin rows to answer a yes/no question — is
// bounded by design: assignments are per-Game and made one at a time by a
// single human Host.
type GameAdminRepository interface {
	// Assign persists a new Game-Admin assignment. Callers
	// (app.Service.AssignGameAdmin) have already run domain.AssignGameAdmin,
	// so the value arrives validated and authorized; this method's own job is
	// the round trip and the CLAUDE.md-rule-4 backstop.
	//
	// Implementations backed by Postgres rely on game_admins' composite
	// primary key (game_id, user_id) as the authoritative guard against a
	// double assignment under concurrency, translating its unique violation
	// into domain.ErrAlreadyGameAdmin — the same relationship
	// RegistrationRepository.Create has with its own unique index, and the
	// reason domain.AssignGameAdmin's duplicate check is described as a
	// pre-check rather than the guard.
	//
	// A game_id referencing no Game returns domain.ErrGameNotFound (the FK
	// firing), though app.Service checks existence first, so that path is
	// defensive rather than routine.
	Assign(ctx context.Context, a domain.GameAdmin) (domain.GameAdmin, error)

	// Revoke removes the assignment for (gameID, userID), returning
	// domain.ErrGameAdminNotFound when there was none.
	//
	// The not-found answer is an error rather than a silent no-op on purpose:
	// a Host revoking an admin is asserting a belief about who currently
	// holds authority, and answering "done" to a revoke that removed nothing
	// would confirm a belief the store does not support. This package's
	// standing "reject, don't guess" stance, same as
	// ExpireWaitlistPromotion's premature-call rejection.
	Revoke(ctx context.Context, gameID, userID string) error

	// ListGameAdmins returns every Game-Admin assignment for gameID, oldest
	// assignment first.
	//
	// An unknown gameID returns an empty slice, not an error — the same
	// convention RegistrationRepository.ListActiveForGame follows, and for
	// the same reason: existence of the Game is the caller's question to ask
	// (app.Service does, via GameRepository.GetByID) and a list read has a
	// natural empty answer. Callers must not treat "no admins" as "no such
	// Game"; the two are genuinely different facts and the app layer
	// distinguishes them.
	ListGameAdmins(ctx context.Context, gameID string) ([]domain.GameAdmin, error)
}
