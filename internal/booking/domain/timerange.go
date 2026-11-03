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

// In re-expresses both ends of the range in loc. The instants are unchanged —
// this is time.Time.In, twice — so Overlaps, Contains and Duration all answer
// exactly the same before and after. What changes is the *wall clock* the
// range reads as: Date(), Weekday() and Hour() all become the facility's
// local ones.
//
// That distinction is the whole of ADR-0004 (T68.1, closes #356). A
// PricingRule's window is a ClockTime, local to the court's facility, while
// a TimeRange built from a protobuf Timestamp is normalised to UTC — so the
// band of an 18:00 Ho Chi Minh City slot can only be resolved after a
// conversion, and ResolvePrice is where it happens.
//
// A nil loc is deliberately not handled here: time.Time.In panics on nil, and
// every caller in this package rejects a nil Location with
// ErrFacilityTimezoneUnset before getting this far. Silently treating nil as
// UTC is the exact bug this method exists to fix.
func (r TimeRange) In(loc *time.Location) TimeRange {
	return TimeRange{Start: r.Start.In(loc), End: r.End.In(loc)}
}
