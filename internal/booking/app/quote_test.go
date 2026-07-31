package app_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nhuthuynh/white-label/internal/booking/app"
	"github.com/nhuthuynh/white-label/internal/booking/domain"
)

// fakePricingRepo is an in-memory port.PricingRuleRepository, scoped by
// court like the real Postgres adapter will be.
type fakePricingRepo struct {
	rulesByCourt map[string][]domain.PricingRule
}

func (r *fakePricingRepo) ListForCourt(_ context.Context, courtID string) ([]domain.PricingRule, error) {
	return r.rulesByCourt[courtID], nil
}

func weekdayPricingRules() []domain.PricingRule {
	weekdayStart, _ := domain.NewClockTime(6, 0)
	weekdayEnd, _ := domain.NewClockTime(17, 0)
	peakEnd, _ := domain.NewClockTime(22, 0)
	weekdays := []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday}

	return []domain.PricingRule{
		{ID: "weekday-rule", CourtID: "court-1", Band: domain.BandWeekday, Weekdays: weekdays, Start: weekdayStart, End: weekdayEnd, PriceCents: 2000},
		{ID: "peak-rule", CourtID: "court-1", Band: domain.BandPeak, Weekdays: weekdays, Start: weekdayEnd, End: peakEnd, PriceCents: 3500},
	}
}

// TestGetQuote_ResolvesPriceForSlot is the app-layer proof (HANDOFF.md T1 AC:
// "table-driven quote tests pass") that GetQuote wires domain.ResolvePrice
// to a real repository round trip.
func TestGetQuote_ResolvesPriceForSlot(t *testing.T) {
	t.Parallel()

	pricingRepo := &fakePricingRepo{rulesByCourt: map[string][]domain.PricingRule{
		"court-1": weekdayPricingRules(),
	}}
	svc := app.NewService(newInMemoryRepo(), pricingRepo, &sequentialIDs{})
	ctx := context.Background()

	tests := []struct {
		name      string
		courtID   string
		slot      domain.TimeRange
		wantPrice int64
		wantErr   error
	}{
		{
			name:      "weekday morning resolves the weekday band",
			courtID:   "court-1",
			slot:      mustTimeRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z"), // 2026-08-03 is a Monday
			wantPrice: 2000,
		},
		{
			name:      "weekday evening resolves the peak band",
			courtID:   "court-1",
			slot:      mustTimeRange(t, "2026-08-03T18:00:00Z", "2026-08-03T19:00:00Z"),
			wantPrice: 3500,
		},
		{
			name:    "court with no pricing rules returns a clear error",
			courtID: "court-9",
			slot:    mustTimeRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z"),
			wantErr: domain.ErrNoPricingRule,
		},
		{
			name:    "slot outside every rule window returns a clear error",
			courtID: "court-1",
			slot:    mustTimeRange(t, "2026-08-08T09:00:00Z", "2026-08-08T10:00:00Z"), // Saturday: no weekend rule configured
			wantErr: domain.ErrNoPricingRule,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			quote, err := svc.GetQuote(ctx, tt.courtID, tt.slot)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("got err %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if quote.PriceCents != tt.wantPrice {
				t.Errorf("price = %d, want %d", quote.PriceCents, tt.wantPrice)
			}
		})
	}
}
