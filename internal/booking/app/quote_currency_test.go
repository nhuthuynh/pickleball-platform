// T68.2 (ADR-0005, closes #357) — a Quote says what currency it is in, and
// its two currencies can never disagree.
//
// The app-layer half of the ticket. The domain half (pricing_currency_test.go,
// discount_currency_test.go) proves the rules; this proves GetQuote carries
// them to the caller — the field is populated from the resolved band, and the
// discount check runs before the arithmetic rather than after.
package app_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nhuthuynh/white-label/internal/booking/app"
	"github.com/nhuthuynh/white-label/internal/booking/domain"
)

// currencyQuoteSvc builds a Service whose single court is priced in
// bandCurrency, with discountRules on its Facility. It mirrors quoteSvc in
// quote_discount_test.go but parameterises the band's currency, which is the
// field under test here.
func currencyQuoteSvc(t *testing.T, bandCurrency string, discountRules ...domain.DiscountRule) *app.Service {
	t.Helper()
	pricing := &fakePricingRepo{rulesByCourt: map[string][]domain.PricingRule{
		courtID(1): {{
			ID: "weekday-rule", CourtID: courtID(1), Band: domain.BandWeekday,
			Weekdays: []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday},
			Start:    mustClock(t, 6, 0), End: mustClock(t, 17, 0),
			PriceCents: 2000, Currency: bandCurrency,
		}},
	}}
	discounts := newFakeDiscountRepo()
	discounts.rulesByFacility[facilityID(1)] = discountRules
	return app.NewService(app.ServiceOptions{
		Bookings:       newInMemoryRepo(),
		PricingRules:   pricing,
		DiscountRules:  discounts,
		RecurringHires: newFakeRecurringHireRepo(),
		Facilities:     zonedLookupFor(courtID(1)),
		Identity:       &fakeIdentityLookup{},
		Zones:          &fakeZones{},
		IDs:            &sequentialIDs{},
	})
}

// fixedDiscount builds a fixed_amount rule in a named currency. It goes
// through domain.NewDiscountRule like discountRule does, so the fixture is a
// rule that could really have been created.
func fixedDiscount(t *testing.T, currency string, cents int64) domain.DiscountRule {
	t.Helper()
	return discountRule(t,
		domain.DiscountAmount{Fixed: domain.Money{Cents: cents, Currency: currency}},
		domain.DiscountTypeFixedAmount,
		[]domain.Source{domain.SourceIndividual},
	)
}

// mondaySlot is 09:00-10:00 on a Monday, which resolves the 06:00-17:00
// weekday band at 2000 in every fixture in this file.
func mondaySlot(t *testing.T) domain.TimeRange {
	t.Helper()
	return mustTimeRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")
}

// TestGetQuote_NamesTheBandCurrency is REQUIREMENT 1: a quote reports the
// currency its price_cents and band_price_cents are denominated in. One
// number with no unit is #357's complaint; two numbers with one unit between
// them is the fix, because both come from the same resolved band by
// construction.
func TestGetQuote_NamesTheBandCurrency(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		currency string
	}{
		{name: "the seeded launch market", currency: "VND"},
		{name: "a second market on the same path", currency: "USD"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := currencyQuoteSvc(t, tt.currency)

			quote, err := svc.GetQuote(context.Background(), app.GetQuoteInput{
				CourtID: courtID(1), Source: domain.SourceIndividual, Range: mondaySlot(t),
			})
			if err != nil {
				t.Fatalf("GetQuote: %v", err)
			}
			if quote.Currency != tt.currency {
				t.Errorf("Currency = %q, want %q", quote.Currency, tt.currency)
			}
			if quote.PriceCents != 2000 || quote.BandPriceCents != 2000 {
				t.Errorf("price/band = %d/%d, want 2000/2000 — carrying a currency must not change the amounts",
					quote.PriceCents, quote.BandPriceCents)
			}
		})
	}
}

// TestGetQuote_CurrencyAgreesWithTheNestedDiscount is REQUIREMENT 2, stated as
// the invariant rather than as an example: **whenever a discount declares a
// currency, it is the quote's own currency.**
//
// The assertion is deliberately written against the two values on the quote
// and not against a constant, so it is the agreement being tested and not two
// independent equalities that happen to hold. That is the same shape
// error_mapping_test.go's TestBookingService_SameSentinelSameCodeAcrossRPCs
// uses for #131: compare the two outputs to each other.
//
// Percent rules are the "does not declare a currency" case: 0017 stores ""
// for them and the proto says the field is meaningful only for fixed_amount,
// so there is nothing to agree with and the row asserts exactly that.
func TestGetQuote_CurrencyAgreesWithTheNestedDiscount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		discount  func(t *testing.T) domain.DiscountRule
		wantPrice int64
	}{
		{
			name:      "a fixed-amount discount in the band's currency",
			discount:  func(t *testing.T) domain.DiscountRule { return fixedDiscount(t, "VND", 500) },
			wantPrice: 1500,
		},
		{
			name: "a percent discount, which declares no currency at all",
			discount: func(t *testing.T) domain.DiscountRule {
				return discountRule(t, domain.DiscountAmount{Percent: 15},
					domain.DiscountTypePercent, []domain.Source{domain.SourceIndividual})
			},
			wantPrice: 1700,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := currencyQuoteSvc(t, "VND", tt.discount(t))

			quote, err := svc.GetQuote(context.Background(), app.GetQuoteInput{
				CourtID: courtID(1), Source: domain.SourceIndividual, Range: mondaySlot(t),
			})
			if err != nil {
				t.Fatalf("GetQuote: %v", err)
			}
			if !quote.Discount.Applies() {
				t.Fatal("the fixture's discount did not apply, so this test would assert nothing")
			}
			if quote.PriceCents != tt.wantPrice {
				t.Errorf("PriceCents = %d, want %d — the discount must actually have been applied",
					quote.PriceCents, tt.wantPrice)
			}
			declared := quote.Discount.Amount.Fixed.Currency
			if declared != "" && declared != quote.Currency {
				t.Errorf("discount currency %q disagrees with the quote's %q — one response, two conventions is #357",
					declared, quote.Currency)
			}
		})
	}
}

// TestGetQuote_RefusesADiscountInAnotherCurrency is the test REQUIREMENT 2
// asks for: *the one that would fail if the two currencies could disagree.*
//
// Without the guard this fixture does not error — it answers 1500 VND, having
// subtracted 500 US cents from a dong amount as though they were dong, and
// hands back a response whose own currency says VND while its nested discount
// says USD. That is the defect, not a theoretical one: it is what
// ApplyToCents did for every fixed-amount rule before this ticket, with no
// currency on the band price to compare against.
//
// Verified by mutation, not by reading: removing the EnsureCurrency call from
// app.Service.GetQuote makes this test fail with `GetQuote err = <nil>` and a
// 1500 price. See the PR body for the full matrix.
func TestGetQuote_RefusesADiscountInAnotherCurrency(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		bandCurrency string
		discount     func(t *testing.T) domain.DiscountRule
	}{
		{
			name:         "a dollar discount against a dong band",
			bandCurrency: "VND",
			discount:     func(t *testing.T) domain.DiscountRule { return fixedDiscount(t, "USD", 500) },
		},
		{
			name:         "the same mismatch the other way round",
			bandCurrency: "USD",
			discount:     func(t *testing.T) domain.DiscountRule { return fixedDiscount(t, "VND", 500) },
		},
		{
			name:         "a fixed-amount rule with no currency recorded at all",
			bandCurrency: "VND",
			discount:     func(t *testing.T) domain.DiscountRule { return fixedDiscount(t, "", 500) },
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := currencyQuoteSvc(t, tt.bandCurrency, tt.discount(t))

			quote, err := svc.GetQuote(context.Background(), app.GetQuoteInput{
				CourtID: courtID(1), Source: domain.SourceIndividual, Range: mondaySlot(t),
			})
			if !errors.Is(err, domain.ErrDiscountCurrencyMismatch) {
				t.Fatalf("GetQuote err = %v (quote %+v), want %v — a discount in another currency must be "+
					"refused, never subtracted", err, quote, domain.ErrDiscountCurrencyMismatch)
			}
		})
	}
}

// TestGetQuote_SurfacesThePricingCurrencySentinels carries the two
// rule-set-level refusals through the real app path, so they are not only
// domain-level facts. Both are pricing-data misconfigurations an operator
// fixes; see domain.ensureOneUsableCurrency for the reasoning behind each.
func TestGetQuote_SurfacesThePricingCurrencySentinels(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		rules   []domain.PricingRule
		wantErr error
	}{
		{
			name: "two rules on one court in two currencies",
			rules: []domain.PricingRule{
				{ID: "a", CourtID: courtID(1), Band: domain.BandWeekday,
					Weekdays: []time.Weekday{time.Monday}, Start: mustClock(t, 6, 0), End: mustClock(t, 17, 0),
					PriceCents: 2000, Currency: "VND"},
				{ID: "b", CourtID: courtID(1), Band: domain.BandPeak,
					Weekdays: []time.Weekday{time.Monday}, Start: mustClock(t, 17, 0), End: mustClock(t, 22, 0),
					PriceCents: 3500, Currency: "USD"},
			},
			wantErr: domain.ErrMixedPricingCurrency,
		},
		{
			name: "a rule whose currency column is empty",
			rules: []domain.PricingRule{
				{ID: "a", CourtID: courtID(1), Band: domain.BandWeekday,
					Weekdays: []time.Weekday{time.Monday}, Start: mustClock(t, 6, 0), End: mustClock(t, 17, 0),
					PriceCents: 2000, Currency: ""},
			},
			wantErr: domain.ErrInvalidPricingCurrency,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := app.NewService(app.ServiceOptions{
				Bookings:       newInMemoryRepo(),
				PricingRules:   &fakePricingRepo{rulesByCourt: map[string][]domain.PricingRule{courtID(1): tt.rules}},
				DiscountRules:  newFakeDiscountRepo(),
				RecurringHires: newFakeRecurringHireRepo(),
				Facilities:     zonedLookupFor(courtID(1)),
				Identity:       &fakeIdentityLookup{},
				Zones:          &fakeZones{},
				IDs:            &sequentialIDs{},
			})

			_, err := svc.GetQuote(context.Background(), app.GetQuoteInput{
				CourtID: courtID(1), Source: domain.SourceIndividual, Range: mondaySlot(t),
			})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("GetQuote err = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// TestGetQuote_UnpricedCourtIsUnaffectedByTheCurrencyChecks is the regression
// this ticket's ordering decision most easily breaks. GetQuote has answered
// ErrNoPricingRule for an unknown, malformed or unpriced court since T1, and
// the currency checks run BEFORE matching — so an empty rule set has to stay
// an empty rule set rather than becoming a currency complaint.
func TestGetQuote_UnpricedCourtIsUnaffectedByTheCurrencyChecks(t *testing.T) {
	t.Parallel()

	svc := currencyQuoteSvc(t, "VND")

	// courtID(9) has no rules in the fake at all, which is what an unpriced
	// or unknown court looks like through this port.
	_, err := svc.GetQuote(context.Background(), app.GetQuoteInput{
		CourtID: courtID(9), Source: domain.SourceIndividual, Range: mondaySlot(t),
	})
	if !errors.Is(err, domain.ErrNoPricingRule) {
		t.Fatalf("GetQuote err = %v, want %v", err, domain.ErrNoPricingRule)
	}
}
