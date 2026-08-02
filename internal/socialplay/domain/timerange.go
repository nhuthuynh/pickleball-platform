package domain

import "time"

// TimeRange is a half-open interval [Start, End) — the same semantics as
// internal/booking/domain.TimeRange (and the Postgres tstzrange the
// Booking's `during` column is generated from). Half-open means a Game
// ending at 10:00 and one starting at 10:00 on the same court do NOT
// overlap.
//
// This type is deliberately a local reimplementation, not an import of
// internal/booking/domain — see CLAUDE.md and the T5 sprint plan kickoff
// note on the socialplay/booking context boundary.
type TimeRange struct {
	Start time.Time
	End   time.Time
}

// NewTimeRange validates and constructs a TimeRange.
func NewTimeRange(start, end time.Time) (TimeRange, error) {
	if !end.After(start) {
		return TimeRange{}, ErrInvalidTimeRange
	}
	return TimeRange{Start: start, End: end}, nil
}

// Overlaps reports whether two half-open ranges share any instant.
// [a.Start, a.End) and [b.Start, b.End) overlap iff a.Start < b.End && b.Start < a.End.
func (r TimeRange) Overlaps(other TimeRange) bool {
	return r.Start.Before(other.End) && other.Start.Before(r.End)
}

func (r TimeRange) Duration() time.Duration {
	return r.End.Sub(r.Start)
}
