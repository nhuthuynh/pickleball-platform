package domain_test

import (
	"testing"

	"github.com/nhuthuynh/white-label/internal/booking/domain"
)

// TestDiscountRule_ApplyToCents is T11.2's domain half: T11.1 built
// ResolveDiscount (which rule applies) but never the arithmetic of actually
// applying it, because nothing consumed a resolved rule yet. GetQuote is now
// that consumer, and "band price minus this discount" is a business rule, so
// it lives here rather than in the app layer (CLAUDE.md rule 2).
//
// Two properties this pins deliberately:
//   - a discount can never produce a negative price (a 100%-off or an
//     oversized fixed_amount rule floors at 0, it does not owe the player
//     money);
//   - the zero-value DiscountRule that ResolveDiscount returns for "no
//     match" (a valid, common outcome per its own doc comment) leaves the
//     price untouched rather than being mistaken for a 0%-off rule.
func TestDiscountRule_ApplyToCents(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		rule  domain.DiscountRule
		price int64
		want  int64
	}{
		{
			name:  "no matching rule (ResolveDiscount's zero value) leaves the price untouched",
			rule:  domain.DiscountRule{},
			price: 2000,
			want:  2000,
		},
		{
			name:  "percent takes the stated percentage off",
			rule:  domain.DiscountRule{DiscountType: domain.DiscountTypePercent, Amount: domain.DiscountAmount{Percent: 15}},
			price: 2000,
			want:  1700,
		},
		{
			name:  "fractional percent rounds to the nearest cent (half away from zero)",
			rule:  domain.DiscountRule{DiscountType: domain.DiscountTypePercent, Amount: domain.DiscountAmount{Percent: 12.5}},
			price: 2001, // 12.5% of 2001 = 250.125 -> 250 off
			want:  1751,
		},
		{
			name:  "percent rounding at exactly half goes away from zero",
			rule:  domain.DiscountRule{DiscountType: domain.DiscountTypePercent, Amount: domain.DiscountAmount{Percent: 50}},
			price: 2001, // 50% of 2001 = 1000.5 -> 1001 off
			want:  1000,
		},
		{
			name:  "100 percent off floors at zero, never negative",
			rule:  domain.DiscountRule{DiscountType: domain.DiscountTypePercent, Amount: domain.DiscountAmount{Percent: 100}},
			price: 2000,
			want:  0,
		},
		{
			name:  "fixed amount takes the stated cents off",
			rule:  domain.DiscountRule{DiscountType: domain.DiscountTypeFixedAmount, Amount: domain.DiscountAmount{Fixed: domain.Money{Cents: 500, Currency: "USD"}}},
			price: 2000,
			want:  1500,
		},
		{
			name:  "fixed amount larger than the price floors at zero, never negative",
			rule:  domain.DiscountRule{DiscountType: domain.DiscountTypeFixedAmount, Amount: domain.DiscountAmount{Fixed: domain.Money{Cents: 9999, Currency: "USD"}}},
			price: 2000,
			want:  0,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.rule.ApplyToCents(tt.price); got != tt.want {
				t.Errorf("ApplyToCents(%d) = %d, want %d", tt.price, got, tt.want)
			}
		})
	}
}

// TestDiscountRule_Applies proves the zero value is distinguishable from a
// real rule without inspecting fields — the predicate GetQuote uses to decide
// whether it has a discount to report at all, so a "no discount" quote is
// never reported as a 0%-off discount that was applied.
func TestDiscountRule_Applies(t *testing.T) {
	t.Parallel()

	if (domain.DiscountRule{}).Applies() {
		t.Error("zero-value DiscountRule reports Applies() = true; ResolveDiscount returns it for a no-match")
	}
	real := domain.DiscountRule{DiscountType: domain.DiscountTypePercent, Amount: domain.DiscountAmount{Percent: 10}}
	if !real.Applies() {
		t.Error("a resolved percent rule reports Applies() = false")
	}
}
