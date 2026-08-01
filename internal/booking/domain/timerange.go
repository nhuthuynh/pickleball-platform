package domain

import "time"

// TimeRange is a half-open interval [Start, End) — the same semantics as the
// Postgres tstzrange the `during` column is generated from. Half-open means a
// booking ending at 10:00 and one starting at 10:00 on the same court do NOT
// overlap; this must stay true in both the domain and the DB constraint.
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

// Contains reports whether other is fully inside r (inclusive of shared
// boundaries), used by pricing-rule window matching.
func (r TimeRange) Contains(other TimeRange) bool {
	return !other.Start.Before(r.Start) && !other.End.After(r.End)
}

func (r TimeRange) Duration() time.Duration {
	return r.End.Sub(r.Start)
}
