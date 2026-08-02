package domain

// Money is an amount in integer minor units ("cents") paired with its
// ISO 4217 currency code. This is ADR-0005's decision reused directly, not
// reopened: amount and currency are coupled in the domain type from day
// one (docs/adr/0005-currency-code-column.md), matching the payment
// processor's own integer-minor-units-plus-currency-code convention. v1
// only ever populates Currency with one constant value (the launch
// market's currency) — see the ADR for the full reasoning.
type Money struct {
	Cents    int64
	Currency string
}

// isValidCurrencyCode is a cheap ISO 4217 *format* check — three uppercase
// letters — not a full currency registry. Matches ADR-0005's explicitly
// accepted scope ("v1 only ever populates it with one constant value");
// validating against the real ISO 4217 list is not needed until a second
// currency is actually supported.
func isValidCurrencyCode(code string) bool {
	if len(code) != 3 {
		return false
	}
	for _, r := range code {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return true
}
