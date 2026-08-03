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
)
