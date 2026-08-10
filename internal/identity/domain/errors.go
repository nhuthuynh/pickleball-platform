package domain

import "errors"

// Domain errors. Adapters must translate infrastructure failures into these
// — upper layers only ever see errors from this file. Mirrors
// internal/booking/domain/errors.go's sentinel-error pattern; deliberately
// not shared with that package (CLAUDE.md rule 2/3: internal/identity/domain
// must not import internal/booking/domain, or any other context's domain
// package).
var (
	ErrEmptyID                          = errors.New("identity: id is required")
	ErrEmptyDisplayName                 = errors.New("identity: display name is required")
	ErrEmptyRoles                       = errors.New("identity: at least one role is required")
	ErrInvalidRole                      = errors.New("identity: invalid role")
	ErrInvalidSelfReportedStartingLevel = errors.New("identity: self-reported starting level must be between 1 and 5")
)
