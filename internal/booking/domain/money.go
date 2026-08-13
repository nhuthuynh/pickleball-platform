package domain

// Money is a context-local money value for the booking domain — deliberately
// NOT imported from internal/payments/domain (T9's ruling against a
// shared-kernel Money type, restated by T11.1's own instructions: each
// bounded context owns its own copy of this shape rather than sharing one).
// Field names/shape mirror internal/payments/domain.Money (ADR-0005:
// integer minor units paired with an ISO 4217 currency code) so the concept
// reads the same across contexts even though the types are independent.
// PricingRule (T1) predates this convention and still uses a plain
// PriceCents int64; left as-is, changing it is out of this ticket's scope.
//
// Like payments.Money, this type has no constructor of its own — validity
// (a positive amount) is enforced where Money is used, e.g.
// NewDiscountRule's ErrInvalidDiscountAmount check for a fixed_amount
// DiscountRule, mirroring how payments.NewPayment validates its own Money
// parameter rather than Money validating itself.
type Money struct {
	Cents    int64
	Currency string
}
