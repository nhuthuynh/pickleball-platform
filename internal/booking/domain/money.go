package domain

// Money is a context-local money value for the booking domain — deliberately
// NOT imported from internal/payments/domain (T9's ruling against a
// shared-kernel Money type, restated by T11.1's own instructions: each
// bounded context owns its own copy of this shape rather than sharing one).
// Field names/shape mirror internal/payments/domain.Money (ADR-0005:
// integer minor units paired with an ISO 4217 currency code) so the concept
// reads the same across contexts even though the types are independent.
// PricingRule (T1) predates this TYPE and deliberately still does not use it:
// it carries PriceCents int64 plus its own Currency string (T68.2, ADR-0005,
// #357). So it follows ADR-0005's convention — integer minor units paired
// with an ISO 4217 code — without using this struct, which is also what
// DiscountRule does on the wire (booking.proto's fixed_amount_cents +
// currency) and what every *_cents/*_currency column pair in db/migrations
// does. Collapsing PricingRule onto Money was explicitly out of T68.2's
// scope, so the half of the old sentence here that still holds is kept: this
// type is not retrofitted onto PricingRule. The half that stopped holding was
// "PricingRule predates this convention" — it no longer does, because the
// convention is the amount/code pairing and PricingRule now has one.
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
