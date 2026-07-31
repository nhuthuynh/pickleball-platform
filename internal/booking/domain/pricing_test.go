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
