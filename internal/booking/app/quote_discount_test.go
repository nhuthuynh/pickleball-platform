package app_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nhuthuynh/white-label/internal/booking/app"
	"github.com/nhuthuynh/white-label/internal/booking/domain"
)

// quoteSvc wires a Service whose court courtID(1) belongs to facilityID(1),
// priced by the weekday/peak bands from quote_test.go, with rules as the
// facility's configured DiscountRules.
func quoteSvc(rules ...domain.DiscountRule) *app.Service {
	pricing := &fakePricingRepo{rulesByCourt: map[string][]domain.PricingRule{
		courtID(1): weekdayPricingRules(),
	}}
	discounts := newFakeDiscountRepo()
	discounts.rulesByFacility[facilityID(1)] = rules
	lookup := &fakeFacilityLookup{
		ownerByFacility: map[string]string{facilityID(1): userID(1)},
		facilityByCourt: map[string]string{courtID(1): facilityID(1)},
		// courtID(2) exists as a Court but belongs to no Facility — the
		// nullable courts.facility_id case (0010_facilities.sql) — so it is
		// deliberately absent from this map.
	}
	return app.NewService(newInMemoryRepo(), pricing, discounts, newFakeRecurringHireRepo(), lookup, &fakeIdentityLookup{}, &sequentialIDs{})
}

func discountRule(t *testing.T, amount domain.DiscountAmount, dType domain.DiscountType, appliesTo []domain.Source) domain.DiscountRule {
	t.Helper()
	r, err := domain.NewDiscountRule(
		"rule-fixture", facilityID(1), dType, amount, appliesTo,
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), domain.NoEnd(),
	)
	if err != nil {
		t.Fatalf("NewDiscountRule fixture: %v", err)
	}
	return r
}

// TestGetQuote_AppliesDiscount is T11.2's headline AC: a discount the Facility
// Owner configured actually changes the number a Player is quoted, rather than
// being decorative. The band price is still reported alongside it, so the UI
// (T11.3) can show what was taken off rather than silently swapping one
// number for another.
func TestGetQuote_AppliesDiscount(t *testing.T) {
	t.Parallel()

	// 2026-08-03 is a Monday; 09:00-10:00 resolves the weekday band at 2000.
	slot := mustTimeRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")

	tests := []struct {
		name             string
		rules            []domain.DiscountRule
		courtID          string
		source           domain.Source
		wantPrice        int64
		wantBandPrice    int64
		wantDiscountAppl bool
	}{
		{
			name:             "no discount configured leaves the band price untouched",
			rules:            nil,
			courtID:          courtID(1),
			source:           domain.SourceIndividual,
			wantPrice:        2000,
			wantBandPrice:    2000,
			wantDiscountAppl: false,
		},
		{
			name: "percent discount for this source is applied",
			rules: []domain.DiscountRule{
				discountRule(t, domain.DiscountAmount{Percent: 15}, domain.DiscountTypePercent, []domain.Source{domain.SourceIndividual}),
			},
			courtID:          courtID(1),
			source:           domain.SourceIndividual,
			wantPrice:        1700,
			wantBandPrice:    2000,
			wantDiscountAppl: true,
		},
		{
			name: "fixed-amount discount for this source is applied",
			rules: []domain.DiscountRule{
				discountRule(t, domain.DiscountAmount{Fixed: domain.Money{Cents: 500, Currency: "USD"}}, domain.DiscountTypeFixedAmount, []domain.Source{domain.SourceIndividual}),
			},
			courtID:          courtID(1),
			source:           domain.SourceIndividual,
			wantPrice:        1500,
			wantBandPrice:    2000,
			wantDiscountAppl: true,
		},
		{
			name: "a discount scoped to another source does not apply",
			rules: []domain.DiscountRule{
				discountRule(t, domain.DiscountAmount{Percent: 15}, domain.DiscountTypePercent, []domain.Source{domain.SourceCompetition}),
			},
			courtID:          courtID(1),
			source:           domain.SourceIndividual,
			wantPrice:        2000,
			wantBandPrice:    2000,
			wantDiscountAppl: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := quoteSvc(tt.rules...)
			quote, err := svc.GetQuote(context.Background(), app.GetQuoteInput{
				CourtID: tt.courtID, Source: tt.source, Range: slot,
			})
			if err != nil {
				t.Fatalf("GetQuote: %v", err)
			}
			if quote.PriceCents != tt.wantPrice {
				t.Errorf("PriceCents = %d, want %d", quote.PriceCents, tt.wantPrice)
			}
			if quote.BandPriceCents != tt.wantBandPrice {
				t.Errorf("BandPriceCents = %d, want %d", quote.BandPriceCents, tt.wantBandPrice)
			}
			if quote.Discount.Applies() != tt.wantDiscountAppl {
				t.Errorf("Discount.Applies() = %v, want %v", quote.Discount.Applies(), tt.wantDiscountAppl)
			}
			if quote.Band != domain.BandWeekday {
				t.Errorf("Band = %q, want %q", quote.Band, domain.BandWeekday)
			}
		})
	}
}

// TestGetQuote_CourtWithNoFacilityStillQuotes is the case the table above
// could not express cleanly: a Court that exists, has pricing, but belongs to
// no Facility (courts.facility_id is nullable for the pre-Facilities seeded
// courts). The quote must still resolve — at the undiscounted band price —
// rather than failing because the discount lookup found nothing.
func TestGetQuote_CourtWithNoFacilityStillQuotes(t *testing.T) {
	t.Parallel()

	pricing := &fakePricingRepo{rulesByCourt: map[string][]domain.PricingRule{
		courtID(2): {{ID: "r", CourtID: courtID(2), Band: domain.BandWeekday,
			Weekdays: []time.Weekday{time.Monday}, Start: mustClock(t, 6, 0), End: mustClock(t, 17, 0), PriceCents: 2000}},
	}}
	discounts := newFakeDiscountRepo()
	discounts.rulesByFacility[facilityID(1)] = []domain.DiscountRule{
		discountRule(t, domain.DiscountAmount{Percent: 50}, domain.DiscountTypePercent, []domain.Source{domain.SourceIndividual}),
	}
	lookup := &fakeFacilityLookup{
		ownerByFacility: map[string]string{facilityID(1): userID(1)},
		facilityByCourt: map[string]string{},
	}
	svc := app.NewService(newInMemoryRepo(), pricing, discounts, newFakeRecurringHireRepo(), lookup, &fakeIdentityLookup{}, &sequentialIDs{})

	quote, err := svc.GetQuote(context.Background(), app.GetQuoteInput{
		CourtID: courtID(2),
		Source:  domain.SourceIndividual,
		Range:   mustTimeRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z"),
	})
	if err != nil {
		t.Fatalf("GetQuote: %v", err)
	}
	if quote.PriceCents != 2000 {
		t.Errorf("PriceCents = %d, want the undiscounted 2000", quote.PriceCents)
	}
	if quote.Discount.Applies() {
		t.Error("a court belonging to no facility picked up another facility's discount")
	}
}

// TestGetQuote_AmbiguousDiscountSurfaces pins ADR-0002's precedent through the
// quote path: two rules matching the same (FacilityID, Source, time) is a
// data-configuration error the caller must fix, never silently resolved by
// picking one. It reaches grpcapi as FailedPrecondition, exactly like
// ErrAmbiguousPricingRule already does.
func TestGetQuote_AmbiguousDiscountSurfaces(t *testing.T) {
	t.Parallel()

	svc := quoteSvc(
		discountRule(t, domain.DiscountAmount{Percent: 15}, domain.DiscountTypePercent, []domain.Source{domain.SourceIndividual}),
		discountRule(t, domain.DiscountAmount{Percent: 25}, domain.DiscountTypePercent, []domain.Source{domain.SourceIndividual}),
	)

	_, err := svc.GetQuote(context.Background(), app.GetQuoteInput{
		CourtID: courtID(1),
		Source:  domain.SourceIndividual,
		Range:   mustTimeRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z"),
	})
	if !errors.Is(err, domain.ErrAmbiguousDiscountRule) {
		t.Fatalf("error = %v, want %v", err, domain.ErrAmbiguousDiscountRule)
	}
}

func mustClock(t *testing.T, h, m int) domain.ClockTime {
	t.Helper()
	c, err := domain.NewClockTime(h, m)
	if err != nil {
		t.Fatalf("NewClockTime(%d,%d): %v", h, m, err)
	}
	return c
}
