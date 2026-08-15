package port

import (
	"context"

	"github.com/nhuthuynh/white-label/internal/competitions/domain"
)

// CompetitionAdminRepository is Competitions' persistence boundary for
// domain.CompetitionAdmin (T15.3, partial fix for #168) — the durable store
// that replaces the caller-supplied `assigned_competition_admin_user_ids` list
// this codebase used to have no alternative to. Mirrors Repository's shape
// exactly: the domain and app layers only ever see this interface,
// internal/competitions/adapter/postgres implements it against the real
// database, and tests implement it in-memory.
//
// It is a **separate interface from Repository** rather than three more
// methods on it, which is the one shape decision here that is not simply
// inherited from T14.4's GameAdminRepository. Two reasons: Repository is
// already a wide interface covering Competitions and their entries, and every
// in-memory fake in service_test.go would have to grow three methods it does
// not use; and T15.5 consumes the read from *outside* this context, where a
// narrow interface is the whole difference between depending on one fact and
// depending on the Competitions repository.
//
// # Why the read is a list rather than a membership test
//
// The sprint plan (T15.3 instruction 4, resolving §A12 GAP A) requires a read
// path here and points at port.GameAdminRepository's doc comment for the
// argument. It applies unchanged, so this interface ships the list for the
// same three reasons:
//
//  1. **It satisfies both shapes the consumers describe.** A boolean is
//     derivable from the set (domain.HasCompetitionAdmin does exactly that,
//     and is exported for it); the set is not derivable from a boolean.
//     Picking the narrower read would have forced T15.4 to widen it, which is
//     the shape of gap this instruction exists to close, not to move by one
//     ticket.
//  2. **The write path needs it anyway.** AssignCompetitionAdmin's
//     already-assigned pre-check (domain.AssignCompetitionAdmin's `existing`
//     argument) reads the Competition's current assignments, exactly as
//     EnterCompetition's capacity pre-check reads its current entries. One
//     read serving both the write pre-check and the consumer's authorization
//     resolution is one query to keep correct, not two.
//  3. **The membership rule stays in the domain.** A repository-level
//     IsCompetitionAdmin would push the blank-entry guard
//     (domain.HasCompetitionAdmin's `a.UserID != ""` check) into SQL, where
//     CLAUDE.md rule 2 says it does not belong and where no unit test without
//     Docker can reach it.
//
// The cost — loading a Competition's admin rows to answer a yes/no question —
// is bounded by design: assignments are per-Competition and made one at a time
// by a single human Host.
type CompetitionAdminRepository interface {
	// Assign persists a new Competition-Admin assignment. Callers
	// (app.Service.AssignCompetitionAdmin) have already run
	// domain.AssignCompetitionAdmin, so the value arrives validated and
	// authorized; this method's own job is the round trip and the
	// CLAUDE.md-rule-4 backstop.
	//
	// Implementations backed by Postgres rely on competition_admins' composite
	// primary key (competition_id, user_id) as the authoritative guard against
	// a double assignment under concurrency, translating its unique violation
	// into domain.ErrAlreadyCompetitionAdmin — the same relationship
	// Repository.CreateEntry has with its own unique index, and the reason
	// domain.AssignCompetitionAdmin's duplicate check is described as a
	// pre-check rather than the guard.
	//
	// A competition_id referencing no Competition returns
	// domain.ErrCompetitionNotFound (the FK firing), though app.Service checks
	// existence first, so that path is defensive rather than routine.
	Assign(ctx context.Context, a domain.CompetitionAdmin) (domain.CompetitionAdmin, error)

	// Revoke removes the assignment for (competitionID, userID), returning
	// domain.ErrCompetitionAdminNotFound when there was none.
	//
	// The not-found answer is an error rather than a silent no-op on purpose:
	// a Host revoking an admin is asserting a belief about who currently holds
	// authority, and answering "done" to a revoke that removed nothing would
	// confirm a belief the store does not support. This context's standing
	// "reject, don't guess" stance, same as domain.Competition.Cancel's
	// rejection of an already-cancelled Competition.
	Revoke(ctx context.Context, competitionID, userID string) error

	// ListCompetitionAdmins returns every Competition-Admin assignment for
	// competitionID, oldest assignment first.
	//
	// An unknown competitionID returns an empty slice, not an error — the same
	// convention Repository.ListActiveEntriesForCompetition follows, and for
	// the same reason: existence of the Competition is the caller's question
	// to ask (app.Service does, via Repository.GetByID) and a list read has a
	// natural empty answer. Callers must not treat "no admins" as "no such
	// Competition"; the two are genuinely different facts and the app layer
	// distinguishes them.
	ListCompetitionAdmins(ctx context.Context, competitionID string) ([]domain.CompetitionAdmin, error)
}
