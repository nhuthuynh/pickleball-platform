package domain

import "time"

const oneDay = 24 * time.Hour

// Band names the kind of pricing window a PricingRule represents. Purely
// descriptive — matching is driven by Weekdays/Start/End, not by Band itself,
// but the name is part of the ubiquitous language (see the operating
// handbook glossary) and is surfaced to hosts/owners when a Quote is shown.
type Band string

const (
	BandWeekday Band = "weekday"
	BandPeak    Band = "peak"
	BandWeekend Band = "weekend"
)

// PricingRule resolves to a price for any slot on CourtID that falls fully
// within [Start, End) on one of Weekdays. Rules are per-court (facility-level
// inheritance, if any, is an app-layer concern of copying/looking up rules —
// the domain only resolves against whatever rule set it's given).
type PricingRule struct {
	ID         string
	CourtID    string
	Band       Band
	Weekdays   []time.Weekday
	Start      ClockTime
	End        ClockTime
	PriceCents int64
}

func (r PricingRule) appliesToWeekday(w time.Weekday) bool {
	for _, d := range r.Weekdays {
		if d == w {
			return true
		}
	}
	return false
}

// covers reports whether the rule's window fully contains the slot. The
// slot must not cross midnight for clock-time comparison to be meaningful;
// callers resolving multi-day slots must split them per day first.
func (r PricingRule) covers(slot TimeRange) bool {
	if !r.appliesToWeekday(slot.Start.Weekday()) {
		return false
	}
	slotStart := ClockTimeOf(slot.Start)
	slotEnd := clockTimeOfEnd(slot)
	return r.Start <= slotStart && slotEnd <= r.End
}

// clockTimeOfEnd handles the common case of a slot ending exactly at
// midnight (e.g. 22:00-24:00), which time.Time represents as 00:00 the next
// calendar day — that must read as end-of-day (1440), not start-of-day (0),
// for boundary comparisons against a rule's End to work.
func clockTimeOfEnd(slot TimeRange) ClockTime {
	end := ClockTimeOf(slot.End)
	if end == 0 && slot.End.After(slot.Start) {
		return ClockTime(minutesPerDay)
	}
	return end
}

// fitsSingleCalendarDay reports whether slot lies within one calendar day —
// clock-time comparison (ClockTimeOf) is only meaningful within a single
// day, so this is the guard that makes PricingRule.covers's documented
// precondition actually hold rather than just assumed. A slot ending
// exactly at midnight the next day still counts (that's the same case
// clockTimeOfEnd exists to handle); anything else spanning into a second
// calendar day does not.
//
// **"One calendar day" means one day in slot's own Location**, which is why
// ResolvePrice converts to the facility's zone before calling this (T68.1,
// ADR-0004) rather than after. Applied to UTC instants it answers a different
// question than the one pricing asks: a 06:30-07:30 slot in a +07:00 zone is
// 23:30-00:30 UTC the day before — one local day, two UTC ones — and would be
// refused as multi-day although no clock-time comparison on it is ambiguous
// at all.
//
// The next local midnight is computed as day+1 at 00:00 (time.Date normalises
// an out-of-range day), NOT as local midnight plus 24 hours. On a DST
// transition day those are different instants: in America/Los_Angeles on
// 2026-03-08, midnight + 24h is 01:00 on the 9th, so a 22:00-24:00 local slot
// that night compared against it was rejected as spanning two days. A test in
// pricing_timezone_test.go pins that case.
func fitsSingleCalendarDay(slot TimeRange) bool {
	// A 24-hour cap on top of the date comparison below. It is redundant for
	// every slot whose local calendar day is 24 hours long, and on a
	// fall-back day (a 25-hour local day) it is deliberately conservative:
	// a full-day slot there is refused rather than priced by a band whose
	// ClockTime window cannot express the repeated hour.
	if slot.Duration() > oneDay {
		return false
	}
	y1, m1, d1 := slot.Start.Date()
	y2, m2, d2 := slot.End.Date()
	if y1 == y2 && m1 == m2 && d1 == d2 {
		return true
	}
	startOfNextDay := time.Date(y1, m1, d1+1, 0, 0, 0, 0, slot.Start.Location())
	return slot.End.Equal(startOfNextDay)
}

// ResolvePrice finds the single PricingRule (scoped to courtID) whose window
// fully contains slot, resolved against the wall clock of loc — the IANA zone
// of the Facility that owns courtID. Returns ErrNoPricingRule if none match (a
// slot straddling two rule windows, e.g. spanning a weekday/peak boundary, is
// a no-match rather than a partial price) and ErrAmbiguousPricingRule if more
// than one rule matches, since that means the rule set itself is
// misconfigured (overlapping windows) — the caller must fix the data, the
// domain won't silently pick one.
//
// # Why loc is a parameter and not an assumption (T68.1, ADR-0004, #356)
//
// A PricingRule's Start/End are ClockTimes: minutes since midnight, local to
// the court's facility. Nothing about a time.Time pins it to that zone — a
// TimeRange built by adapter/grpcapi from a protobuf Timestamp is normalised
// to UTC, so before this parameter existed an 18:00 Ho Chi Minh City slot
// arrived as 11:00 and was quoted the weekday band instead of peak. The
// conversion happens here, once, rather than at each caller, which is
// ADR-0004's "convert before computing, never assume the caller already
// converted" in code: every clock-time, weekday and calendar-day question
// below is asked of the converted range.
//
// loc is a *time.Location rather than an IANA name on purpose. Resolving a
// name means reading the system's tz database, which is I/O this package must
// not do (CLAUDE.md rule 2) — the app layer loads it through
// port.LocationLoader and hands the result down.
//
// A nil loc is ErrFacilityTimezoneUnset, never UTC. The domain will not price
// a slot it cannot place on a local clock; see that sentinel's doc comment.
func ResolvePrice(rules []PricingRule, courtID string, slot TimeRange, loc *time.Location) (PricingRule, error) {
	if loc == nil {
		return PricingRule{}, ErrFacilityTimezoneUnset
	}
	local := slot.In(loc)

	if !fitsSingleCalendarDay(local) {
		return PricingRule{}, ErrPricingSlotSpansMultipleDays
	}

	var matches []PricingRule
	for _, r := range rules {
		if r.CourtID != courtID {
			continue
		}
		if r.covers(local) {
			matches = append(matches, r)
		}
	}

	switch len(matches) {
	case 0:
		return PricingRule{}, ErrNoPricingRule
	case 1:
		return matches[0], nil
	default:
		return PricingRule{}, ErrAmbiguousPricingRule
	}
}
