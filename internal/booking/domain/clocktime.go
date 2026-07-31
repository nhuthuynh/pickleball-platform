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

// ClockTimeOf extracts the ClockTime of the given instant.
func ClockTimeOf(t time.Time) ClockTime {
	return ClockTime(t.Hour()*60 + t.Minute())
}

func (c ClockTime) String() string {
	return fmt.Sprintf("%02d:%02d", int(c)/60, int(c)%60)
}
