package domain

import "time"

// CompetitionAdmin is a durable record that a Competition's Host delegated
// Competition-Admin authority over that Competition to another user (T15.3,
// partial fix for #168).
//
// The exact mirror of internal/socialplay/domain.GameAdmin (T14.4), in the
// context T14 did not reach. It is deliberately a separate type in a separate
// package rather than a shared one: each bounded context owns its own domain
// vocabulary, and internal/competitions/domain never imports
// internal/socialplay/domain (CLAUDE.md rules 2 and 7 — "one ubiquitous
// language" means a Competition Admin is a Competition Admin everywhere, not
// that two contexts share a Go type).
//
// # Why this type exists at all
//
// Before this ticket the concept "Competition Admin" existed in this codebase
// only as a **caller-supplied repeated string field** on the requests that
// needed it (payments' RecordOfflinePaymentRequest.
// assigned_competition_admin_user_ids and its refund twin). Nothing persisted
// it, so any authorization rule wanting to include admins had to either trust
// a list the caller wrote — letting a caller name themselves an admin and
// satisfy the check — or exclude admins entirely, which is what T13.6 did to
// Competitions' roster read (see app.Service.ListEntriesForCompetition, whose
// doc comment says so in as many words). This type is the durable fact those
// rules were missing.
//
// # Identifier space (ADR-0014/ADR-0017, T29.1 — closes the Competitions
// third of #164)
//
// UserID and AssignedBy hold **User.ID** uuids as of T29.1, the same
// identifier space Competition.HostID holds. AssignedBy is minted from the
// grpcapi handler's actor(ctx), which resolves the caller's verified subject
// to a User.ID before app.Service.AssignCompetitionAdmin ever runs (ADR-0014's
// boundary rule) — never from the wire. UserID names a DIFFERENT user (the
// one being granted authority, not the caller), so it cannot come from
// actor(ctx); the handler instead resolves the wire-supplied subject through
// the identical port.IdentityLookup seam before it ever reaches this package
// (see internal/competitions/adapter/grpcapi.Handler.resolveTargetUserID).
//
// Before T29.1, both fields held plain IdP **subjects** instead — ADR-0014
// §5a named Competitions explicitly, alongside Social Play and Payments, as a
// context whose stored actor facts were non-conformant-but-self-consistent
// text subjects, with conformance deferred to #164. That gap is what T29.1
// closes: competitions.host_id, competition_entries.player_id, and this
// table's user_id/assigned_by are all backfilled to real `uuid` foreign keys
// into identity_users(id) in the same migration
// (db/migrations/0025_competitions_identity_conformance.sql), per ADR-0017's
// ruling — so a uuid is never compared against a subject on either side of
// any check in this file. db/migrations/0021_competitions_competition_admins.sql
// carries its own historical note for the pre-migration shape.
//
// # What this type is not
//
// It is not a profile, a role catalogue, or a permission set. It records one
// fact — that this user holds Competition-Admin authority over this
// Competition, granted by that assigner at that time — and every rule reading
// it is expressed as a function in this package.
type CompetitionAdmin struct {
	// CompetitionID is the Competition this assignment is scoped to. A
	// Competition Admin's authority never spans Competitions: the assignment
	// is per-Competition by construction, the same scoping CLAUDE.md's locked
	// "per-game Game Admins" decision gives Social Play.
	CompetitionID string
	// UserID is the User.ID (uuid, as of T29.1 — see the identifier-space
	// note above) holding Competition-Admin authority over CompetitionID.
	UserID string
	// AssignedBy is the User.ID (uuid, as of T29.1) of the Host who made the
	// assignment. Stored rather than inferred: Competition.HostID can only
	// ever answer "who hosts this Competition now", and an audit of a
	// delegated authority needs to record who actually granted it at the
	// time.
	AssignedBy string
	// AssignedAt is when the assignment was made. Supplied by the caller
	// (app.Service passes time.Now()) rather than read here, keeping this
	// package free of ambient clock reads — the same convention every other
	// constructor in this package follows.
	AssignedAt time.Time
}

// AssignCompetitionAdmin validates and constructs a Competition-Admin
// assignment.
//
// existing is the Competition's current set of assignments (app.Service reads
// it through port.CompetitionAdminRepository.ListCompetitionAdmins), used only
// for the already-assigned pre-check — the composite primary key on
// (competition_id, user_id) is the authoritative guard under concurrency,
// exactly the relationship domain.Enter has with
// competition_entries_active_player_idx (CLAUDE.md rule 4).
//
// Order of checks, and why:
//
//  1. **Host-only authorization, first.** c.EnsureHost, and there is
//     deliberately no host-or-admin variant to reach for, so an assigned
//     Competition Admin is refused. #168 puts the reason plainly: an admin
//     must not be able to appoint an admin, or the Host-only distinction
//     ErrNotCompetitionHost exists to protect is worthless. Checking it before
//     any input validation means an unauthorized actor learns nothing about
//     whether the request would otherwise have been valid, the ordering
//     internal/payments/app.authorizeOfflineRecording established.
//  2. A blank adminUserID is rejected (ErrEmptyCompetitionAdminUserID). A
//     blank stored row would match a blank actor at read time — the accident
//     HasCompetitionAdmin's `a.UserID != ""` guard defends the read end
//     against, closed here at the write end too so the bad row never exists.
//  3. Assigning the Host themselves is rejected
//     (ErrHostCannotBeCompetitionAdmin). The Host is already entitled to
//     everything a Competition Admin is, so the row would grant nothing;
//     worse, it would make a later revoke appear to remove the Host's own
//     authority, which no assignment row can do. Keeping Host and Competition
//     Admin disjoint is also what makes check 1 complete by construction: an
//     admin is never the Host, so the Host-only gate is the whole enforcement,
//     with no separate "is the actor merely an admin" branch to forget.
//  4. A duplicate assignment is rejected (ErrAlreadyCompetitionAdmin) rather
//     than silently re-applied, this package's standing "reject, don't guess"
//     stance (see ErrAlreadyEntered, ErrIllegalStatusTransition).
//
// Deliberately NOT checked here: whether the Competition is cancelled. A
// cancelled Competition can still need its roster read and its outstanding
// cash entry fees reconciled — the two things a Competition Admin exists to do
// — so gating assignment on Competition status would strand exactly the work
// that outlives the cancellation. The operation where a cancelled Competition
// genuinely forbids the action (Enter) keeps its own cancelled check, where
// the rule actually belongs. This mirrors AssignGameAdmin's identical call.
func AssignCompetitionAdmin(c Competition, existing []CompetitionAdmin, actorUserID, adminUserID string, now time.Time) (CompetitionAdmin, error) {
	if err := c.EnsureHost(actorUserID); err != nil {
		return CompetitionAdmin{}, err
	}
	if adminUserID == "" {
		return CompetitionAdmin{}, ErrEmptyCompetitionAdminUserID
	}
	if adminUserID == c.HostID {
		return CompetitionAdmin{}, ErrHostCannotBeCompetitionAdmin
	}
	if HasCompetitionAdmin(existing, adminUserID) {
		return CompetitionAdmin{}, ErrAlreadyCompetitionAdmin
	}

	return CompetitionAdmin{
		CompetitionID: c.ID,
		UserID:        adminUserID,
		AssignedBy:    actorUserID,
		AssignedAt:    now,
	}, nil
}

// EnsureMayRevokeCompetitionAdmin returns ErrNotCompetitionHost unless
// actorUserID is the Competition's Host — the revoke half of
// AssignCompetitionAdmin's check 1.
//
// Granting and withdrawing a delegated authority are the same authority, so
// they are gated identically: an assigned Competition Admin who could revoke
// their peers (or, with a self-targeted call, could not usefully be
// constrained at all) would defeat the Host-only rule from the other end.
// Stated as its own exported function rather than left implicit at the app
// layer so the rule is testable in this package, alongside the assignment rule
// it mirrors.
//
// It deliberately does not take the target user id: whether the target is
// currently assigned is a persistence fact
// (port.CompetitionAdminRepository.Revoke answers it with
// ErrCompetitionAdminNotFound), not a domain rule, and folding it in here
// would require this pure function to be handed the whole assignment set for
// no rule of its own.
func EnsureMayRevokeCompetitionAdmin(c Competition, actorUserID string) error {
	return c.EnsureHost(actorUserID)
}

// HasCompetitionAdmin reports whether userID holds a Competition-Admin
// assignment in admins.
//
// This is the resolution helper the consumer side (T15.4's roster read, T15.5's
// Payments authorization) turns
// port.CompetitionAdminRepository.ListCompetitionAdmins' result into an
// authorization answer with, kept here so the membership rule — including the
// blank-entry guard — lives in the domain rather than being re-derived at each
// call site.
//
// A blank userID never matches, even against a blank row: it mirrors
// domain.HasGameAdmin's identical guard, so an unidentified caller can never be
// resolved into an entitled one. Both ends are guarded —
// AssignCompetitionAdmin refuses to create such a row at all — because a rule
// that depends on the other end having been enforced is a rule that breaks the
// first time a row arrives from somewhere else.
func HasCompetitionAdmin(admins []CompetitionAdmin, userID string) bool {
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
