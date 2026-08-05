package domain

// Money is an amount in integer minor units ("cents") paired with its
// ISO 4217 currency code. This is ADR-0005's decision reused directly, not
// reopened: amount and currency are coupled in the domain type from day
// one (docs/adr/0005-currency-code-column.md), matching the payment
// processor's own integer-minor-units-plus-currency-code convention. v1
// only ever populates Currency with one constant value (the launch
// market's currency) — see the ADR for the full reasoning.
//
// This is a deliberate, context-local duplicate of
// internal/payments/domain.Money, not an oversight and not a candidate for
// extraction into a shared package: CLAUDE.md rules 2/3 keep each bounded
// context's domain self-contained, so internal/socialplay/domain must not
// import internal/payments/domain (nor the reverse). The two types are the
// same shape today and are still allowed to diverge tomorrow — and they
// already differ in one rule that matters (see IsFree and Validate below):
// a payments.Payment of zero is meaningless and rejected, whereas a Game's
// EntryFee of zero is a real product state (a free Game). Sharing one type
// would have forced one of those two rules onto the other context.
type Money struct {
	Cents    int64
	Currency string
}

// IsFree reports whether this amount costs the player nothing. Zero is a
// real, Host-chosen value — a free Game — never a sentinel for "no price
// set" (T9.2 retired exactly such a sentinel: T8.10's
// PLACEHOLDER_REGISTRATION_FEE_CENTS, which had no way to express "free" at
// all). Exposed as a named predicate rather than leaving `fee.Cents == 0`
// comparisons scattered across the app, adapter, and UI layers, so the
// "free" product state has one definition everywhere (CLAUDE.md rule 7).
func (m Money) IsFree() bool {
	return m.Cents == 0
}

// Validate reports whether this amount is a well-formed entry fee.
//
// The rules, and why each one:
//   - A negative amount is rejected. There is no Game that pays a player to
//     attend. This is the domain-side twin of
//     db/migrations/0013_socialplay_entry_fee.sql's
//     CHECK (entry_fee_cents >= 0) — CLAUDE.md rule 4 requires the invariant
//     to be expressed in both places, and the two must be kept in sync if
//     either ever changes.
//   - A non-zero amount with an empty currency code is rejected: ADR-0005's
//     entire point is that a bare cents figure with no currency must never
//     exist in the domain. A *zero* amount with an empty currency is
//     accepted, because a free Game genuinely has no currency to name —
//     nothing is being charged in any currency. Both representations of free
//     (Money{0, ""} from a Host who set no fee, and Money{0, "USD"} from the
//     migration's backfill default) satisfy IsFree, so they behave
//     identically everywhere downstream.
//   - A non-empty currency code must be three uppercase letters — the same
//     cheap ISO 4217 *format* check internal/payments/domain applies, for
//     the same reason: validating against the real ISO 4217 registry is not
//     needed until a second currency is actually supported (ADR-0005
//     explicitly scopes v1 to one constant value).
func (m Money) Validate() error {
	if m.Cents < 0 {
		return ErrInvalidMoney
	}
	if m.Cents == 0 && m.Currency == "" {
		return nil
	}
	if !isValidCurrencyCode(m.Currency) {
		return ErrInvalidMoney
	}
	return nil
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
