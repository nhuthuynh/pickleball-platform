package domain

// Money is a context-local money value for the booking domain. It is
// deliberately NOT a shared/imported cross-context type — each bounded
// context that needs money owns its own copy of this shape (T9's ruling
// against a shared-kernel Money type, restated for T11.1's DiscountRule,
// which is the first thing in this context that needs one). PricingRule
// (T1) predates this convention and still uses a plain PriceCents int64;
// left as-is here, since changing it is out of this ticket's scope.
type Money struct {
	AmountCents  int64
	CurrencyCode string
}

// NewMoney validates and builds a Money. A valid Money has a positive amount
// and a non-empty currency code — either missing makes the value meaningless.
func NewMoney(amountCents int64, currencyCode string) (Money, error) {
	if amountCents <= 0 || currencyCode == "" {
		return Money{}, ErrInvalidMoney
	}
	return Money{AmountCents: amountCents, CurrencyCode: currencyCode}, nil
}

// IsPositive reports whether m is a valid, positive amount of money. Used by
// DiscountRule's constructor to bound-check a fixed_amount discount without
// forcing every caller through NewMoney (e.g. values already loaded from
// storage, where validity was already enforced on write).
func (m Money) IsPositive() bool {
	return m.AmountCents > 0 && m.CurrencyCode != ""
}
