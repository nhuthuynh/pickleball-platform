package domain

import "errors"

// Domain errors. Adapters must translate infrastructure failures (e.g. a
// Postgres 23P01 exclusion-constraint violation) into these — upper layers
// only ever see errors from this file.
var (
	ErrInvalidTimeRange             = errors.New("booking: invalid time range")
	ErrInvalidClockTime             = errors.New("booking: invalid clock time")
	ErrInvalidSource                = errors.New("booking: invalid source")
	ErrEmptyCourtID                 = errors.New("booking: court id is required")
	ErrCourtDoubleBooked            = errors.New("booking: court already booked for an overlapping time range")
	ErrIllegalStatusTransition      = errors.New("booking: illegal status transition")
	ErrBookingNotFound              = errors.New("booking: not found")
	ErrNoPricingRule                = errors.New("booking: no pricing rule matches the requested slot")
	ErrAmbiguousPricingRule         = errors.New("booking: more than one pricing rule matches the requested slot")
	ErrPricingSlotSpansMultipleDays = errors.New("booking: pricing slot must fall within a single calendar day")
)
