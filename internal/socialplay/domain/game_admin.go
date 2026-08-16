package domain

import "time"

// GameAdmin is a durable record that a Game's Host delegated Game-Admin
// authority over that Game to another user (T14.4, partial fix for #168).
//
// # Why this type exists at all
//
// Before this ticket the concept "Game Admin" existed in this codebase only as
// a **caller-supplied repeated string field** on the two requests that needed
// it (socialplay's RecordMatchResultRequest.assigned_game_admin_user_ids and
// payments' RecordOfflinePaymentRequest equivalents). Nothing persisted it, so
// any authorization rule wanting to include admins had to either trust a list
// the caller wrote — letting a caller name themselves an admin and satisfy the
// check — or exclude admins entirely, which is what T13.6 did to Social Play's
// roster read. This type is the durable fact those rules were missing.
//
// # Identifier space (ADR-0014/ADR-0017, issue #164 — UPDATED T29.2)
//
// UserID and AssignedBy hold **User.ID** (uuid) as of T29.2, closing the
// Social Play third of #164: db/migrations/0026_socialplay_identity_conformance.sql
// converts game_admins.user_id/assigned_by from `text` to
// `uuid NOT NULL REFERENCES identity_users (id)`, backfilling every existing
// row by joining its stored subject against identity_users.subject. This is
// the same identifier space Game.HostID holds (both resolved once, at write
// time, by the grpcapi handler's actor()/resolveUserID funnel — see
// adapter/grpcapi.Handler.actor's and .resolveUserID's doc comments for the
// full mechanism, including why AssignGameAdmin's/RevokeGameAdmin's
// caller-supplied user_id target needed its own resolution step no other
// context's identical migration required).
//
// BEFORE T29.2, UserID and AssignedBy held **subjects** — the value
// internal/platform/auth.RequireSubject returned, unresolved. ADR-0014 §5a
// ruled that deliberately (Social Play's stored actor facts were
// non-conformant-but-self-consistent text subjects, with conformance
// deferred to #164), and db/migrations/0020_socialplay_game_admins.sql's own
// header comment carried the identical note — both are now historical: this
// paragraph is kept, dated, rather than deleted, mirroring
// internal/payments/adapter/grpcapi.Handler.actor's "kept all versions
// dated" convention, so a reader tracing an old reference to "subjects" here
// finds the history rather than a silently vanished claim.
//
// # What this type is not
//
// It is not a profile, a role catalogue, or a permission set. It records one
// fact — that this user holds Game-Admin authority over this Game, granted by
// that assigner at that time — and every rule reading it is expressed as a
// function in this package.
type GameAdmin struct {
	// GameID is the Game this assignment is scoped to. A Game Admin's
	// authority never spans Games: the assignment is per-Game by
	// construction, which is what CLAUDE.md's locked "per-game Game Admins"
	// decision means.
	GameID string
	// UserID is the subject holding Game-Admin authority over GameID.
	UserID string
	// AssignedBy is the subject of the Host who made the assignment. Stored
	// rather than inferred: Game.HostID can only ever answer "who hosts this
	// Game now", and an audit of a delegated authority needs to record who
	// actually granted it at the time.
	AssignedBy string
	// AssignedAt is when the assignment was made. Supplied by the caller
	// (app.Service passes time.Now()) rather than read here, keeping this
	// package free of ambient clock reads — the same convention
	// domain.RecordMatch already follows for Match.RecordedAt.
	AssignedAt time.Time
}

// AssignGameAdmin validates and constructs a Game-Admin assignment.
//
// existing is the Game's current set of assignments (app.Service reads it
// through port.GameAdminRepository.ListGameAdmins), used only for the
// already-assigned pre-check — the composite primary key on
// (game_id, user_id) is the authoritative guard under concurrency, exactly
// the relationship domain.Register has with the registrations unique index
// (CLAUDE.md rule 4).
//
// Order of checks, and why:
//
//  1. **Host-only authorization, first.** g.EnsureHost — not
//     EnsureHostOrGameAdmin — so an assigned Game Admin is refused. #168 puts
//     the reason plainly: "an admin must not be able to appoint an admin, or
//     the Host-only distinction ErrNotGameHost exists to protect is
//     worthless." Checking it before any input validation means an
//     unauthorized actor learns nothing about whether the request would
//     otherwise have been valid, the ordering
//     internal/payments/app.authorizeOfflineRecording established and
//     app.Service.RecordMatchResult already follows.
//  2. A blank adminUserID is rejected (ErrEmptyGameAdminUserID). A blank
//     stored row would match a blank actor at read time — the accident
//     Game.EnsureHostOrGameAdmin's existing `adminID != ""` guard exists to
//     prevent, closed here at the write end too so the bad row never exists.
//  3. Assigning the Host themselves is rejected (ErrHostCannotBeGameAdmin).
//     The Host is already entitled to everything a Game Admin is, so the row
//     would grant nothing; worse, it would make a later RevokeGameAdmin
//     appear to remove the Host's own authority, which no assignment row can
//     do. Keeping Host and Game Admin disjoint is also what makes check 1
//     complete by construction: an admin is never the Host, so the Host-only
//     gate is the whole enforcement, with no separate "is the actor merely an
//     admin" branch to forget.
//  4. A duplicate assignment is rejected (ErrAlreadyGameAdmin) rather than
//     silently re-applied, this package's standing "reject, don't guess"
//     stance (see ErrAlreadyRegistered, ErrAlreadyOnWaitlist).
//
// Deliberately NOT checked here: whether the Game is cancelled. A cancelled
// Game can still need its roster read and its outstanding cash payments
// reconciled — the two things a Game Admin exists to do — so gating
// assignment on Game status would strand exactly the work that outlives the
// cancellation. The one operation where a cancelled Game genuinely forbids
// the action (RecordMatchResult) keeps its own Game.EnsureNotCancelled check,
// where the rule actually belongs.
func AssignGameAdmin(g Game, existing []GameAdmin, actorUserID, adminUserID string, now time.Time) (GameAdmin, error) {
	if err := g.EnsureHost(actorUserID); err != nil {
		return GameAdmin{}, err
	}
	if adminUserID == "" {
		return GameAdmin{}, ErrEmptyGameAdminUserID
	}
	if adminUserID == g.HostID {
		return GameAdmin{}, ErrHostCannotBeGameAdmin
	}
	if HasGameAdmin(existing, adminUserID) {
		return GameAdmin{}, ErrAlreadyGameAdmin
	}

	return GameAdmin{
		GameID:     g.ID,
		UserID:     adminUserID,
		AssignedBy: actorUserID,
		AssignedAt: now,
	}, nil
}

// EnsureMayRevokeGameAdmin returns ErrNotGameHost unless actorUserID is the
// Game's Host — the revoke half of AssignGameAdmin's check 1.
//
// Granting and withdrawing a delegated authority are the same authority, so
// they are gated identically: an assigned Game Admin who could revoke their
// peers (or, with a self-targeted call, could not usefully be constrained at
// all) would defeat the Host-only rule from the other end. Stated as its own
// exported function rather than left implicit at the app layer so the rule is
// testable in this package, alongside the assignment rule it mirrors.
//
// It deliberately does not take the target user id: whether the target is
// currently assigned is a persistence fact (port.GameAdminRepository.Revoke
// answers it with ErrGameAdminNotFound), not a domain rule, and folding it in
// here would require this pure function to be handed the whole assignment set
// for no rule of its own.
func EnsureMayRevokeGameAdmin(g Game, actorUserID string) error {
	return g.EnsureHost(actorUserID)
}

// HasGameAdmin reports whether userID holds a Game-Admin assignment in admins.
//
// This is the resolution helper the consumer side (T14.5) turns
// port.GameAdminRepository.ListGameAdmins' result into an authorization answer
// with, kept here so the membership rule — including the blank-entry guard —
// lives in the domain rather than being re-derived at each call site.
//
// A blank userID never matches, even against a blank row: it mirrors
// Game.EnsureHostOrGameAdmin's identical `adminID != ""` guard, so an
// unidentified caller can never be resolved into an entitled one. Both ends
// are guarded — AssignGameAdmin refuses to create such a row at all — because
// a rule that depends on the other end having been enforced is a rule that
// breaks the first time a row arrives from somewhere else.
func HasGameAdmin(admins []GameAdmin, userID string) bool {
	if userID == "" {
		return false
	}
	for _, a := range admins {
		if a.UserID != "" && a.UserID == userID {
			return true
		}
	}
	return false
}
