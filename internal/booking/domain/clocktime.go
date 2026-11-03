package domain

import (
	"fmt"
	"time"
)

// ClockTime is minutes since midnight, local to the court's facility. It lets
// PricingRule windows be expressed independently of any calendar date.
type ClockTime int

const minutesPerDay = 24 * 60

// NewClockTime builds a ClockTime from an hour (0-23) and minute (0-59).
func NewClockTime(hour, minute int) (ClockTime, error) {
	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return 0, fmt.Errorf("%w: %02d:%02d", ErrInvalidClockTime, hour, minute)
	}
	return ClockTime(hour*60 + minute), nil
}

// ClockTimeOf extracts the ClockTime that t reads **in t's own Location**.
//
// It cannot do anything else: a time.Time carries one wall clock, the one its
// Location gives it, and this function has no way to know which zone the
// caller meant. That is the trap ADR-0004 named and T68.1 (#356) closed — the
// type comment above has said "local to the court's facility" since T0 while
// adapter/grpcapi fed this the UTC-normalised instant off a protobuf
// Timestamp, quoting an 18:00 Ho Chi Minh City slot as 11:00.
//
// So **convert first**: TimeRange.In(loc) re-expresses a slot in the
// facility's zone, and ResolvePrice — the only caller in this package — does
// that before any clock-time comparison. Treat this as a reader of an
// already-local value, never as the place a zone is applied.
func ClockTimeOf(t time.Time) ClockTime {
	return ClockTime(t.Hour()*60 + t.Minute())
}

func (c ClockTime) String() string {
	return fmt.Sprintf("%02d:%02d", int(c)/60, int(c)%60)
}
