package domain

import "time"

// TimeRange is a half-open interval [Start, End) — the same semantics as
// internal/booking/domain.TimeRange (and the Postgres tstzrange the
// Booking's `during` column is generated from) and
// internal/socialplay/domain.TimeRange. Half-open means a session ending at
// 12:00 and one starting at 12:00 on the same court do NOT overlap.
//
// This type is deliberately a local reimplementation, not an import of
// either sibling context's version — see CLAUDE.md rule 2/3 and the T5
// sprint plan's kickoff note on the socialplay/booking context boundary,
// which applies unchanged to Competitions.
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
// [r.Start, r.End) and [other.Start, other.End) overlap iff
// r.Start < other.End && other.Start < r.End.
func (r TimeRange) Overlaps(other TimeRange) bool {
	return r.Start.Before(other.End) && other.Start.Before(r.End)
}

// Duration is the length of the range.
func (r TimeRange) Duration() time.Duration {
	return r.End.Sub(r.Start)
}
