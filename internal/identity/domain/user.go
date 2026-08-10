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
	ID                        string
	DisplayName               string
	Roles                     []Role
	SelfReportedStartingLevel SelfReportedStartingLevel
}

// NewUser constructs a User, validating the invariants that don't require
// knowledge of other Users or another context: ID and DisplayName must be
// non-empty, Roles must be non-empty and every element must be one of
// Role's closed-enum values, and SelfReportedStartingLevel must fall within
// its bounded 1..5 range.
func NewUser(id, displayName string, roles []Role, level SelfReportedStartingLevel) (User, error) {
	if id == "" {
		return User{}, ErrEmptyID
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
		DisplayName:               displayName,
		Roles:                     roles,
		SelfReportedStartingLevel: level,
	}, nil
}
