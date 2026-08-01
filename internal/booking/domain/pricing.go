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
func fitsSingleCalendarDay(slot TimeRange) bool {
	if slot.Duration() > oneDay {
		return false
	}
	y1, m1, d1 := slot.Start.Date()
	y2, m2, d2 := slot.End.Date()
	if y1 == y2 && m1 == m2 && d1 == d2 {
		return true
	}
	startOfNextDay := time.Date(y1, m1, d1, 0, 0, 0, 0, slot.Start.Location()).Add(oneDay)
	return slot.End.Equal(startOfNextDay)
}

// ResolvePrice finds the single PricingRule (scoped to courtID) whose window
// fully contains slot. Returns ErrNoPricingRule if none match (a slot
// straddling two rule windows, e.g. spanning a weekday/peak boundary, is a
// no-match rather than a partial price) and ErrAmbiguousPricingRule if more
// than one rule matches, since that means the rule set itself is
// misconfigured (overlapping windows) — the caller must fix the data, the
// domain won't silently pick one.
func ResolvePrice(rules []PricingRule, courtID string, slot TimeRange) (PricingRule, error) {
	if !fitsSingleCalendarDay(slot) {
		return PricingRule{}, ErrPricingSlotSpansMultipleDays
	}

	var matches []PricingRule
	for _, r := range rules {
		if r.CourtID != courtID {
			continue
		}
		if r.covers(slot) {
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
