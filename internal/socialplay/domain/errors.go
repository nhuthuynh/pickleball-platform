package domain

import "errors"

// Domain errors. Adapters must translate infrastructure failures into these
// — upper layers only ever see errors from this file. Mirrors
// internal/booking/domain/errors.go's sentinel-error pattern; deliberately
// not shared with that package (CLAUDE.md: socialplay/domain must not
// import internal/booking/domain).
var (
	ErrInvalidTimeRange        = errors.New("socialplay: invalid time range")
	ErrInvalidCapacity         = errors.New("socialplay: capacity must be greater than zero")
	ErrEmptyCourtIDs           = errors.New("socialplay: at least one court id is required")
	ErrIllegalStatusTransition = errors.New("socialplay: illegal status transition")
	ErrEmptyPlayerID           = errors.New("socialplay: player id is required")
	// ErrGameFull, ErrAlreadyRegistered, and ErrNotRegistrationOwner are
	// deliberately distinct, stable sentinels (not a generic validation
	// error) — T5.4's gRPC/REST layer maps each to its own status code, and
	// T6's waitlist promotion trigger needs a clean hook on ErrGameFull
	// specifically (see docs/process/t5-sprint-plan.md T5.2 kickoff note).
	ErrGameFull             = errors.New("socialplay: game is at capacity")
	ErrAlreadyRegistered    = errors.New("socialplay: player is already registered for this game")
	ErrNotRegistrationOwner = errors.New("socialplay: only the registering player may cancel this registration")
)
