// T68.2 (ADR-0005, closes #357) — a fixed-amount discount may only be taken
// off a price in its own currency.
//
// This is requirement 2's domain half, and it is a real arithmetic bug rather
// than a tidiness point: DiscountRule.ApplyToCents does
// `priceCents - r.Amount.Fixed.Cents` with no reference to either currency,
// so a 500-cent USD rule taken off a 200000-dong band price silently
// subtracts 500 dong. Nothing in the repository noticed, because before this
// ticket the band price had no currency to compare against.
//
// Percent rules are deliberately exempt. A percentage is dimensionless — 15%
// off is 15% off in any currency — and `discount_rules.currency` is
// documented by 0017 as "Empty for a percent rule, which has no currency".
// Requiring a match there would refuse every percent rule in the table.
package domain_test

import (
	"errors"
	"testing"

	"github.com/nhuthuynh/white-label/internal/booking/domain"
)

func TestDiscountRule_EnsureCurrency(t *testing.T) {
	t.Parallel()

	fixed := func(currency string, cents int64) domain.DiscountRule {
		return domain.DiscountRule{
			DiscountType: domain.DiscountTypeFixedAmount,
			Amount:       domain.DiscountAmount{Fixed: domain.Money{Cents: cents, Currency: currency}},
		}
	}
	percent := func(currency string) domain.DiscountRule {
		return domain.DiscountRule{
			DiscountType: domain.DiscountTypePercent,
			Amount:       domain.DiscountAmount{Percent: 15, Fixed: domain.Money{Currency: currency}},
		}
	}

	tests := []struct {
		name         string
		rule         domain.DiscountRule
		bandCurrency string
		wantErr      error
		why          string
	}{
		{
			name:         "a fixed-amount rule in the band's own currency applies",
			rule:         fixed("VND", 500),
			bandCurrency: "VND",
			wantErr:      nil,
		},
		{
			name:         "a fixed-amount rule in another currency is refused",
			rule:         fixed("USD", 500),
			bandCurrency: "VND",
			wantErr:      domain.ErrDiscountCurrencyMismatch,
			why: "ApplyToCents would subtract 500 from a dong amount as though the 500 were dong; the two " +
				"currencies on the response would then disagree, which is the one thing requirement 2 forbids",
		},
		{
			name:         "a fixed-amount rule carrying no currency at all is refused",
			rule:         fixed("", 500),
			bandCurrency: "VND",
			wantErr:      domain.ErrDiscountCurrencyMismatch,
			why: "0017 made discount_rules.currency NOT NULL DEFAULT the empty string, with no CHECK, so an unlabelled " +
				"fixed-amount rule is a row that exists today; it is not treated as 'same currency as whatever " +
				"it is applied to', because that assumption is exactly #357",
		},
		{
			name:         "a percent rule is exempt, and its empty currency is not a mismatch",
			rule:         percent(""),
			bandCurrency: "VND",
			wantErr:      nil,
			why:          "a percentage is dimensionless, and 0017 documents the empty string as the correct stored value",
		},
		{
			name:         "a percent rule that somehow carries a foreign code is still exempt",
			rule:         percent("USD"),
			bandCurrency: "VND",
			wantErr:      nil,
			why: "the field is meaningless for this discount type (0017, and the proto says so too), so reading " +
				"it would make a stray value change the arithmetic — the opposite of the fix",
		},
		{
			name:         "the no-discount zero value is exempt",
			rule:         domain.DiscountRule{},
			bandCurrency: "VND",
			wantErr:      nil,
			why: "ResolveDiscount returns the zero value for a no-match with a nil error, and a quote with no " +
				"discount is the common outcome; refusing it would break every undiscounted quote",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.rule.EnsureCurrency(tt.bandCurrency)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("EnsureCurrency(%q) = %v, want %v\nwhy: %s", tt.bandCurrency, err, tt.wantErr, tt.why)
			}
		})
	}
}
