package domain

// User is the Identity/Users bounded context's aggregate (T10.1,
// docs/process/t10-sprint-plan.md), mirroring internal/booking/domain's
// shape per CLAUDE.md: an ID, the fields this context owns, and a
// constructor that validates the invariants that don't require calling out
// to another context.
//
// Explicitly not built here, per ADR-0012
// (docs/adr/0012-identity-users-and-match-built-rating-and-matching-algorithm-blocked-on-escalated-decisions.md):
// no PlayerRating field, no derived Level, no Gender field, no
// matching-mode flag. SelfReportedStartingLevel below is the raw
// self-reported input the locked CLAUDE.md decision specifies for
// cold-start seeding ("new players seeded by a self-reported starting
// level") — it is NOT an answer to ADR-0012's still-open Q1 (the
// tenure+win-rate-weighted, history-derived Level formula); a future
// reader must not conflate the two self-reported-1..5-scale-in vs.
// computed-score-out. PlayerRating, any computed Level, Gender, and any
// matching-mode flag remain blocked on ADR-0012's Q1/Q2 and are deliberately
// absent from this aggregate, not an oversight.
type User struct {
	// ID is this platform's own identifier for the User: a uuid this
	// backend mints (T12.9, via port.IDGenerator like every other
	// aggregate in this codebase). Before T12.9 it was the caller-supplied
	// actor_user_id, which is the squatting hole HANDOFF.md's T10.2 bullet
	// disclosed and T12.9 closed — see Subject below.
	ID string

	// Subject is the identity provider's verified `sub` claim for the
	// person this User represents, as carried by auth.Principal.Subject
	// (internal/platform/auth). It is the external identity; ID is the
	// internal one, and they are deliberately separate columns rather than
	// one — a subject is an arbitrary provider-specific string like
	// `auth0|abc123`, not a uuid, so it cannot be this row's uuid primary
	// key. The full reasoning lives in
	// db/migrations/0019_identity_subject.sql and is not restated here.
	//
	// This field is what makes a User claimable only by the person who can
	// actually authenticate as it. Note that domain code never *verifies* a
	// subject — verification is internal/platform/auth's job, at the
	// grpcapi boundary (T12 sprint plan A11 Ruling 3). By the time a
	// subject reaches this package it is already verified, and this package
	// must not import auth to double-check it.
	//
	// Uniqueness across Users is NOT enforced here: it is a cross-aggregate
	// fact a single User cannot see. Postgres owns it (a UNIQUE constraint,
	// CLAUDE.md rule 4's authoritative half) and the adapter translates the
	// violation to ErrUserAlreadyExists.
	Subject string

	DisplayName               string
	Roles                     []Role
	SelfReportedStartingLevel SelfReportedStartingLevel
}

// NewUser constructs a User, validating the invariants that don't require
// knowledge of other Users or another context: ID, Subject, and DisplayName
// must be non-empty, Roles must be non-empty and every element must be one
// of Role's closed-enum values, and SelfReportedStartingLevel must fall
// within its bounded 1..5 range.
//
// subject is a *verified* IdP subject by the time it arrives here (see the
// Subject field's doc comment). An empty one is rejected rather than
// tolerated: a User with no subject is a row nobody can ever authenticate
// as, and — worse — a later ownership comparison against "" could match
// another empty value and succeed. That is the same class of bug
// auth.ContextWithPrincipal refuses a zero-Subject principal for.
func NewUser(id, subject, displayName string, roles []Role, level SelfReportedStartingLevel) (User, error) {
	if id == "" {
		return User{}, ErrEmptyID
	}
	if subject == "" {
		return User{}, ErrEmptySubject
	}
	if displayName == "" {
		return User{}, ErrEmptyDisplayName
	}
	if len(roles) == 0 {
		return User{}, ErrEmptyRoles
	}
	for _, r := range roles {
		if !r.IsValid() {
			return User{}, ErrInvalidRole
		}
	}
	if !level.IsValid() {
		return User{}, ErrInvalidSelfReportedStartingLevel
	}
	return User{
		ID:                        id,
		Subject:                   subject,
		DisplayName:               displayName,
		Roles:                     roles,
		SelfReportedStartingLevel: level,
	}, nil
}

// EnsureSelf returns ErrNotSelf unless actorUserID matches u.ID exactly (an
// empty actorUserID is always rejected). This is T10.2's object-level (BOLA)
// authorization check for UpdateSelfReportedLevel — mirrors
// internal/facilities/domain.Facility.EnsureOwner/internal/socialplay/domain.
// Registration's actorPlayerID-vs-PlayerID check applied to a User's own
// identity fact instead: only the User themself may update their own
// self-reported level.
//
// T12.9 UPDATE — the caveat this comment used to carry no longer applies to
// this context. actorUserID was a caller-supplied claim taken straight off
// the wire; it is now resolved at the grpcapi boundary from the verified
// auth.Principal's subject (the handler looks up the User owning that
// subject and passes THAT user's ID), and the wire field is ignored
// entirely. This function's signature and rule are unchanged — which is the
// point of A11 Ruling 3: the domain keeps expressing "only the User
// themself", and what got fixed is that the actor handed to it is now a
// verified fact rather than an assertion. See ErrNotSelf's doc comment.
func (u User) EnsureSelf(actorUserID string) error {
	if actorUserID == "" || actorUserID != u.ID {
		return ErrNotSelf
	}
	return nil
}

// UpdateSelfReportedLevel returns a copy of u with a new
// SelfReportedStartingLevel, but only once the caller has been proven to be
// this User (EnsureSelf, checked first — mirrors
// internal/facilities/domain.Facility.AddCameraLink's check ordering: a
// mismatched actor is rejected with ErrNotSelf without ever learning
// whether the new level itself would have been valid) and only once level
// falls within the bounded range NewUser enforces at construction
// (ErrInvalidSelfReportedStartingLevel otherwise) — the range invariant
// applies on update too, not only at construction.
func (u User) UpdateSelfReportedLevel(actorUserID string, level SelfReportedStartingLevel) (User, error) {
	if err := u.EnsureSelf(actorUserID); err != nil {
		return User{}, err
	}
	if !level.IsValid() {
		return User{}, ErrInvalidSelfReportedStartingLevel
	}
	u.SelfReportedStartingLevel = level
	return u, nil
}
