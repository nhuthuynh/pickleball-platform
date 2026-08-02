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

	// ErrCourtUnavailable is Social Play's own, context-local conflict
	// error (T5.3) — the translated equivalent of
	// bookingdomain.ErrCourtDoubleBooked, but never that type itself, so
	// internal/socialplay/domain and internal/socialplay/app never need to
	// import internal/booking/domain (CLAUDE.md rule 2 / T5 sprint plan
	// kickoff note). port.CourtReservation implementations (the T5.4
	// adapter, or an in-memory test fake) return an error satisfying
	// errors.Is(err, ErrCourtUnavailable) when the requested court/time
	// already has a confirmed Booking of any source.
	ErrCourtUnavailable = errors.New("socialplay: court unavailable for the requested time range")
)
