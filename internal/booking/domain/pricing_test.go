package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/nhuthuynh/white-label/internal/booking/domain"
)

// Fixture calendar: 2024-01-01 is a Monday, 2024-01-06 a Saturday, 2024-01-07
// a Sunday — used so weekday-matching in the table tests below is obvious
// from the rule definitions without recomputing Weekday() by hand.
func dt(t *testing.T, y int, m time.Month, d, hour, min int) time.Time {
	t.Helper()
	return time.Date(y, m, d, hour, min, 0, 0, time.UTC)
}

func weekdayRules(t *testing.T) []domain.PricingRule {
	t.Helper()
	weekdayStart, err := domain.NewClockTime(6, 0)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	weekdayEnd, err := domain.NewClockTime(17, 0)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	peakEnd, err := domain.NewClockTime(22, 0)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	weekdays := []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday}
	weekend := []time.Weekday{time.Saturday, time.Sunday}

	return []domain.PricingRule{
		{
			ID: "weekday-rule", CourtID: "court-1", Band: domain.BandWeekday,
			Weekdays: weekdays, Start: weekdayStart, End: weekdayEnd, PriceCents: 2000,
		},
		{
			ID: "peak-rule", CourtID: "court-1", Band: domain.BandPeak,
			Weekdays: weekdays, Start: weekdayEnd, End: peakEnd, PriceCents: 3500,
		},
		{
			ID: "weekend-rule", CourtID: "court-1", Band: domain.BandWeekend,
			Weekdays: weekend, Start: weekdayStart, End: peakEnd, PriceCents: 3000,
		},
	}
}

func TestResolvePrice_BandsAndBoundaries(t *testing.T) {
	t.Parallel()
	rules := weekdayRules(t)

	tests := []struct {
		name       string
		courtID    string
		slot       domain.TimeRange
		wantRuleID string
		wantPrice  int64
		wantErr    error
	}{
		{
			name: "weekday morning resolves to weekday band", courtID: "court-1",
			slot:       mustRangeAt(t, dt(t, 2024, 1, 1, 9, 0), dt(t, 2024, 1, 1, 10, 0)),
			wantRuleID: "weekday-rule", wantPrice: 2000,
		},
		{
			name: "weekday evening resolves to peak band", courtID: "court-1",
			slot:       mustRangeAt(t, dt(t, 2024, 1, 1, 18, 0), dt(t, 2024, 1, 1, 19, 0)),
			wantRuleID: "peak-rule", wantPrice: 3500,
		},
		{
			name: "saturday resolves to weekend band", courtID: "court-1",
			slot:       mustRangeAt(t, dt(t, 2024, 1, 6, 10, 0), dt(t, 2024, 1, 6, 11, 0)),
			wantRuleID: "weekend-rule", wantPrice: 3000,
		},
		{
			name: "sunday resolves to weekend band", courtID: "court-1",
			slot:       mustRangeAt(t, dt(t, 2024, 1, 7, 9, 0), dt(t, 2024, 1, 7, 10, 0)),
			wantRuleID: "weekend-rule", wantPrice: 3000,
		},
		{
			name:       "slot ending exactly at the weekday/peak boundary stays in weekday band",
			courtID:    "court-1",
			slot:       mustRangeAt(t, dt(t, 2024, 1, 1, 16, 0), dt(t, 2024, 1, 1, 17, 0)),
			wantRuleID: "weekday-rule", wantPrice: 2000,
		},
		{
			name:       "slot starting exactly at the weekday/peak boundary is peak band",
			courtID:    "court-1",
			slot:       mustRangeAt(t, dt(t, 2024, 1, 1, 17, 0), dt(t, 2024, 1, 1, 18, 0)),
			wantRuleID: "peak-rule", wantPrice: 3500,
		},
		{
			name:    "slot straddling the weekday/peak boundary matches no rule",
			courtID: "court-1",
			slot:    mustRangeAt(t, dt(t, 2024, 1, 1, 16, 30), dt(t, 2024, 1, 1, 17, 30)),
			wantErr: domain.ErrNoPricingRule,
		},
		{
			name:    "court with no rules at all",
			courtID: "court-9",
			slot:    mustRangeAt(t, dt(t, 2024, 1, 1, 9, 0), dt(t, 2024, 1, 1, 10, 0)),
			wantErr: domain.ErrNoPricingRule,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := domain.ResolvePrice(rules, tt.courtID, tt.slot)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("got err %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if got.ID != tt.wantRuleID {
				t.Errorf("resolved rule ID = %q, want %q", got.ID, tt.wantRuleID)
			}
			if got.PriceCents != tt.wantPrice {
				t.Errorf("resolved price = %d, want %d", got.PriceCents, tt.wantPrice)
			}
		})
	}
}

// TestResolvePrice_RejectsCrossMidnightSlot proves the precondition
// documented on PricingRule.covers ("the slot must not cross midnight") is
// actually enforced, not just assumed. Without the guard, a 2-hour slot
// spanning midnight (Mon 23:00 -> Tue 01:00) would satisfy a 1-hour
// "23:00-24:00 Monday only" rule's covers() check, because clock-time
// comparison alone can't tell "01:00 the same day" from "01:00 the next
// day" — silently pricing two hours of court time as if they were both
// covered by a rule that should apply to Monday night only.
func TestResolvePrice_RejectsCrossMidnightSlot(t *testing.T) {
	t.Parallel()

	nightStart, _ := domain.NewClockTime(23, 0)
	nightEnd := domain.ClockTime(1440) // 24:00 — only reachable via a DB-sourced rule, see clockTimeOfEnd.
	rules := []domain.PricingRule{
		{ID: "night-rule", CourtID: "court-1", Band: domain.BandPeak, Weekdays: []time.Weekday{time.Monday}, Start: nightStart, End: nightEnd, PriceCents: 5000},
	}

	// Monday 23:00 -> Tuesday 01:00: genuinely crosses midnight.
	slot := mustRangeAt(t, dt(t, 2024, 1, 1, 23, 0), dt(t, 2024, 1, 2, 1, 0))

	_, err := domain.ResolvePrice(rules, "court-1", slot)
	if !errors.Is(err, domain.ErrPricingSlotSpansMultipleDays) {
		t.Fatalf("got err %v, want %v", err, domain.ErrPricingSlotSpansMultipleDays)
	}
}

func TestResolvePrice_RejectsMultiDaySlot(t *testing.T) {
	t.Parallel()

	rules := weekdayRules(t)
	// Monday 09:00 -> Wednesday 09:00: spans multiple full days.
	slot := mustRangeAt(t, dt(t, 2024, 1, 1, 9, 0), dt(t, 2024, 1, 3, 9, 0))

	_, err := domain.ResolvePrice(rules, "court-1", slot)
	if !errors.Is(err, domain.ErrPricingSlotSpansMultipleDays) {
		t.Fatalf("got err %v, want %v", err, domain.ErrPricingSlotSpansMultipleDays)
	}
}

// TestResolvePrice_SlotEndingExactlyAtMidnightStillResolves is the
// regression guard for the exact-midnight case the cross-midnight fix must
// not break: a slot ending precisely at 00:00 the next day is still a
// same-day slot (clockTimeOfEnd's whole reason for existing).
func TestResolvePrice_SlotEndingExactlyAtMidnightStillResolves(t *testing.T) {
	t.Parallel()

	lateStart, _ := domain.NewClockTime(22, 0)
	lateEnd := domain.ClockTime(1440)
	rules := []domain.PricingRule{
		{ID: "late-rule", CourtID: "court-1", Band: domain.BandPeak, Weekdays: []time.Weekday{time.Monday}, Start: lateStart, End: lateEnd, PriceCents: 4000},
	}

	// Monday 22:00 -> Tuesday 00:00 (exactly midnight): still a same-day slot.
	slot := mustRangeAt(t, dt(t, 2024, 1, 1, 22, 0), dt(t, 2024, 1, 2, 0, 0))

	got, err := domain.ResolvePrice(rules, "court-1", slot)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got.PriceCents != 4000 {
		t.Errorf("resolved price = %d, want 4000", got.PriceCents)
	}
}

func TestResolvePrice_AmbiguousRulesRejected(t *testing.T) {
	t.Parallel()

	weekdayStart, _ := domain.NewClockTime(6, 0)
	weekdayEnd, _ := domain.NewClockTime(17, 0)
	weekdays := []time.Weekday{time.Monday}

	overlapping := []domain.PricingRule{
		{ID: "rule-a", CourtID: "court-1", Band: domain.BandWeekday, Weekdays: weekdays, Start: weekdayStart, End: weekdayEnd, PriceCents: 2000},
		{ID: "rule-b", CourtID: "court-1", Band: domain.BandWeekday, Weekdays: weekdays, Start: weekdayStart, End: weekdayEnd, PriceCents: 2500},
	}

	slot := mustRangeAt(t, dt(t, 2024, 1, 1, 9, 0), dt(t, 2024, 1, 1, 10, 0))
	_, err := domain.ResolvePrice(overlapping, "court-1", slot)
	if !errors.Is(err, domain.ErrAmbiguousPricingRule) {
		t.Fatalf("got err %v, want %v", err, domain.ErrAmbiguousPricingRule)
	}
}

func mustRangeAt(t *testing.T, start, end time.Time) domain.TimeRange {
	t.Helper()
	r, err := domain.NewTimeRange(start, end)
	if err != nil {
		t.Fatalf("bad fixture range: %v", err)
	}
	return r
}
