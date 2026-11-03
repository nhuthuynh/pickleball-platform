// T68.2 (ADR-0005, closes #357) — a resolved price band says what currency
// it is denominated in, and refuses to answer when it cannot.
//
// ADR-0005 is Accepted and decided that an ISO 4217 code sits beside every
// `price_cents`-style column, **"starting with `pricing_rules`"**. The column
// reached payments (0005), games (0013), competitions (0014) and
// DiscountRule (0017) and never reached the table the ADR named first, so
// GetQuoteResponse carried two conventions in one message: a bare
// `price_cents` and a nested DiscountRule that does declare its currency.
//
// The three cases below are the three answers ResolvePrice can now give about
// currency, and they are tested here (not only through the app layer) because
// every one of them is a pure function of the rule set:
//
//  1. the happy path — the resolved rule's own code comes back;
//  2. a rule set whose rows disagree — refused, never silently picked;
//  3. a rule carrying no usable code at all — refused, never defaulted.
package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/nhuthuynh/white-label/internal/booking/domain"
)

// currencyRule builds one minimal weekday rule for court-1 carrying currency,
// so each case below differs in exactly the field under test. Everything else
// is the same 06:00-17:00 Monday-Friday window pricing_test.go's own fixtures
// use, and the fixture calendar is shared with that file (2024-01-01 is a
// Monday).
func currencyRule(t *testing.T, id, currency string, priceCents int64) domain.PricingRule {
	t.Helper()
	start, err := domain.NewClockTime(6, 0)
	if err != nil {
		t.Fatalf("fixture clock time: %v", err)
	}
	end, err := domain.NewClockTime(17, 0)
	if err != nil {
		t.Fatalf("fixture clock time: %v", err)
	}
	return domain.PricingRule{
		ID: id, CourtID: "court-1", Band: domain.BandWeekday,
		Weekdays:   []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday},
		Start:      start,
		End:        end,
		PriceCents: priceCents,
		Currency:   currency,
	}
}

// TestResolvePrice_ReportsTheResolvedRuleCurrency is requirement 1's domain
// half: the code is carried on PricingRule and comes back on the resolved
// rule, so the app layer has something to put on the response beside
// price_cents rather than leaving a client to guess.
//
// VND is the fixture value on purpose rather than USD: 0035 seeds the demo
// facility's rules in VND (its zone is Asia/Ho_Chi_Minh, 0034), and ADR-0009
// answered the market-scope question on 2026-10-07 as *both* markets — so a
// USD-shaped fixture would quietly re-assert the single-market assumption
// this ticket exists to remove.
func TestResolvePrice_ReportsTheResolvedRuleCurrency(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		currency     string
		wantCurrency string
	}{
		{name: "the launch market's code", currency: "VND", wantCurrency: "VND"},
		{name: "a second market's code on the same shape", currency: "USD", wantCurrency: "USD"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rules := []domain.PricingRule{currencyRule(t, "weekday-rule", tt.currency, 2000)}
			slot := mustRangeAt(t, dt(t, 2024, time.January, 1, 9, 0), dt(t, 2024, time.January, 1, 10, 0))

			got, err := domain.ResolvePrice(rules, "court-1", slot, time.UTC)
			if err != nil {
				t.Fatalf("ResolvePrice: %v", err)
			}
			if got.Currency != tt.wantCurrency {
				t.Errorf("Currency = %q, want %q", got.Currency, tt.wantCurrency)
			}
			if got.PriceCents != 2000 {
				t.Errorf("PriceCents = %d, want 2000 — the amount must be untouched by carrying a currency", got.PriceCents)
			}
		})
	}
}

// TestResolvePrice_RefusesAMixedCurrencyRuleSet is REQUIREMENT 3's answer,
// asserted rather than described: when one court's pricing rules name more
// than one currency, ResolvePrice refuses with ErrMixedPricingCurrency. It
// does not pick the matching rule's code and carry on.
//
// # The two cases that matter, and why the second one is the whole point
//
// Row 1 is the obvious one: the slot matches a rule, and the court's other
// rule disagrees about currency. Row 2 is the one a narrower check would get
// wrong — the disagreeing rule is on a *different weekday*, so a check that
// only looked at the matching rule would answer 2000 VND perfectly happily
// and nobody would ever learn that Saturday on this court is priced in
// dollars. That is why the check is over the court's whole rule set and not
// over the matches: a misconfiguration visible only on the slots that happen
// to resolve cleanly is the exact shape of every defect in this repository's
// history (the band that resolved fine in UTC until #356; the CHECK that
// never widened until 0031).
//
// Row 3 pins the scope boundary: a *different court's* rule in another
// currency is not this court's problem. Currency agreement is per-court
// because pricing_rules is per-court, and a platform serving both Vietnam and
// a global market (ADR-0009, answered 2026-10-07) must be able to price two
// facilities' courts in two currencies.
func TestResolvePrice_RefusesAMixedCurrencyRuleSet(t *testing.T) {
	t.Parallel()

	otherCourt := func(t *testing.T, currency string) domain.PricingRule {
		t.Helper()
		r := currencyRule(t, "other-court-rule", currency, 9900)
		r.CourtID = "court-2"
		return r
	}
	saturdayRule := func(t *testing.T, currency string) domain.PricingRule {
		t.Helper()
		r := currencyRule(t, "weekend-rule", currency, 3000)
		r.Band = domain.BandWeekend
		r.Weekdays = []time.Weekday{time.Saturday, time.Sunday}
		return r
	}

	tests := []struct {
		name    string
		rules   []domain.PricingRule
		wantErr error
	}{
		{
			name: "two rules on the same court disagree, and one of them matches",
			rules: []domain.PricingRule{
				currencyRule(t, "weekday-rule", "VND", 2000),
				saturdayRule(t, "USD"),
			},
			wantErr: domain.ErrMixedPricingCurrency,
		},
		{
			name: "the disagreeing rule does not match the slot at all",
			rules: []domain.PricingRule{
				currencyRule(t, "weekday-rule", "VND", 2000),
				// A second weekday rule, non-overlapping in clock time, so
				// this set is NOT ambiguous — exactly one rule matches the
				// 09:00 slot. Only the currency disagrees.
				func() domain.PricingRule {
					r := currencyRule(t, "evening-rule", "USD", 3500)
					r.Band = domain.BandPeak
					start, _ := domain.NewClockTime(17, 0)
					end, _ := domain.NewClockTime(22, 0)
					r.Start, r.End = start, end
					return r
				}(),
			},
			wantErr: domain.ErrMixedPricingCurrency,
		},
		{
			name: "another court's rule in another currency is not this court's problem",
			rules: []domain.PricingRule{
				currencyRule(t, "weekday-rule", "VND", 2000),
				otherCourt(t, "USD"),
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			slot := mustRangeAt(t, dt(t, 2024, time.January, 1, 9, 0), dt(t, 2024, time.January, 1, 10, 0))

			got, err := domain.ResolvePrice(tt.rules, "court-1", slot, time.UTC)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("ResolvePrice err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ResolvePrice: %v", err)
			}
			if got.Currency != "VND" {
				t.Errorf("Currency = %q, want VND", got.Currency)
			}
		})
	}
}

// TestResolvePrice_RefusesAnUnusableCurrency is the domain half of 0035's
// `NOT NULL` plus its `pricing_rules_currency_shape` CHECK (CLAUDE.md rule 4
// — the invariant is expressed on both sides). An amount with no unit is the
// defect #357 reported; a column that merely *exists* and holds "" would
// relocate that defect rather than fix it, so the domain refuses it too.
//
// Unset and malformed share one sentinel deliberately. errors.go already
// collapses exactly this distinction twice — ErrFacilityNotFound ("an unknown
// or malformed FacilityID") and ErrRecurringHireTemplateNotFound — and the
// test it is NOT modelled on is T68.1's ErrFacilityTimezoneUnset /
// ErrInvalidFacilityTimezone pair, which split because one is fixed by an
// operator recording data and the other by a runtime missing its tz database.
// A missing currency code and a malformed one are the same fix by the same
// person in the same row of the same table.
func TestResolvePrice_RefusesAnUnusableCurrency(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		currency string
	}{
		{name: "empty, which is what a column with no value looks like", currency: ""},
		{name: "lowercase", currency: "vnd"},
		{name: "two letters", currency: "VN"},
		{name: "four letters", currency: "VNDX"},
		{name: "a symbol rather than a code", currency: "$"},
		{name: "a numeric ISO 4217 code, which is not the alphabetic one", currency: "704"},
		{name: "padded", currency: " VND"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rules := []domain.PricingRule{currencyRule(t, "weekday-rule", tt.currency, 2000)}
			slot := mustRangeAt(t, dt(t, 2024, time.January, 1, 9, 0), dt(t, 2024, time.January, 1, 10, 0))

			_, err := domain.ResolvePrice(rules, "court-1", slot, time.UTC)
			if !errors.Is(err, domain.ErrInvalidPricingCurrency) {
				t.Fatalf("ResolvePrice err = %v, want %v", err, domain.ErrInvalidPricingCurrency)
			}
		})
	}
}

// TestResolvePrice_CurrencyChecksPrecedence pins the ORDER of the three
// refusals a currency can cause against the three that already existed, so
// the message a misconfigured facility gets is the one naming the fault it
// should fix first. Each row states which answer wins and why.
//
// This exists because the checks are not commutative and the choice was a
// real one: putting the currency check before the match is what makes case
// "unpriced court" below a question at all.
func TestResolvePrice_CurrencyChecksPrecedence(t *testing.T) {
	t.Parallel()

	slot := mustRangeAt(t, dt(t, 2024, time.January, 1, 9, 0), dt(t, 2024, time.January, 1, 10, 0))
	twoDaySlot := mustRangeAt(t, dt(t, 2024, time.January, 1, 9, 0), dt(t, 2024, time.January, 3, 10, 0))

	tests := []struct {
		name    string
		rules   []domain.PricingRule
		slot    domain.TimeRange
		wantErr error
		why     string
	}{
		{
			name:    "a court with no rules at all still answers ErrNoPricingRule",
			rules:   nil,
			slot:    slot,
			wantErr: domain.ErrNoPricingRule,
			why: "an empty set has no currencies to disagree about and no row to malform, so the currency checks " +
				"are vacuous here — this is the case that must NOT change, because GetQuote has answered " +
				"ErrNoPricingRule for an unknown, malformed or unpriced court since T1",
		},
		{
			name:    "a malformed currency beats the no-match answer",
			rules:   []domain.PricingRule{currencyRule(t, "weekday-rule", "", 2000)},
			slot:    mustRangeAt(t, dt(t, 2024, time.January, 6, 9, 0), dt(t, 2024, time.January, 6, 10, 0)), // Saturday: matches nothing
			wantErr: domain.ErrInvalidPricingCurrency,
			why: "the court IS priced and the row is broken; answering ErrNoPricingRule would send an operator " +
				"looking for missing band rows that are right there",
		},
		{
			name: "a malformed currency beats a mixed rule set",
			rules: []domain.PricingRule{
				currencyRule(t, "weekday-rule", "", 2000),
				currencyRule(t, "second-rule", "USD", 2000),
			},
			slot:    slot,
			wantErr: domain.ErrInvalidPricingCurrency,
			why: "a row with no usable code is a more basic fault than two rows disagreeing, and fixing it may " +
				"resolve the disagreement too — reporting 'mixed' first would name a symptom",
		},
		{
			name: "the single-calendar-day guard still runs before any currency check",
			rules: []domain.PricingRule{
				currencyRule(t, "weekday-rule", "VND", 2000),
				saturdayRuleInUSD(t),
			},
			slot:    twoDaySlot,
			wantErr: domain.ErrPricingSlotSpansMultipleDays,
			why: "that guard is about the REQUEST, not the data; a caller whose slot cannot be placed on one " +
				"clock gets told about their slot, and would get the same answer against a flawless rule set",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := domain.ResolvePrice(tt.rules, "court-1", tt.slot, time.UTC)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ResolvePrice err = %v, want %v\nwhy: %s", err, tt.wantErr, tt.why)
			}
		})
	}
}

func saturdayRuleInUSD(t *testing.T) domain.PricingRule {
	t.Helper()
	r := currencyRule(t, "weekend-rule", "USD", 3000)
	r.Band = domain.BandWeekend
	r.Weekdays = []time.Weekday{time.Saturday, time.Sunday}
	return r
}
