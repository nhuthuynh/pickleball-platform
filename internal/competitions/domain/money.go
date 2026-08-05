package domain

// Money is an amount in integer minor units ("cents") paired with its
// ISO 4217 currency code — ADR-0005's decision (amount and currency coupled
// in the domain type, never a bare cents value) applied to
// Competition.EntryFee.
//
// This is a *context-local* value type that deliberately duplicates
// internal/payments/domain.Money rather than importing or sharing it. Do
// not "fix" the duplication by extracting a shared money package: each
// bounded context defining its own small value type is correct DDD, and a
// shared cross-context money type would be exactly the shared-kernel
// coupling agent-operating-handbook.md A1 forbids. The precedent is already
// documented on the proto side (proto/pickleball/payments/v1/payments.proto:
// "Money mirrors internal/payments/domain.Money") and is restated in
// docs/process/t9-sprint-plan.md §A4.
//
// Field names differ from Payments' (AmountCents/CurrencyCode vs
// Cents/Currency) because T9.1's ticket specifies these; the *semantics*
// are identical, which is what the ubiquitous language requires (CLAUDE.md
// rule 7 — a fee is minor units plus a currency code everywhere).
//
// A zero amount is a real, legal value meaning a **free competition** — not
// a placeholder or an "unset" marker. Callers rendering an entry fee must
// show "Free" for it rather than an empty price (see IsZero).
type Money struct {
	AmountCents  int64
	CurrencyCode string
}

// Validate reports whether this is a well-formed Money value:
//
//   - a negative amount is never legal (an entry fee that pays the entrant
//     is not a modelled concept);
//   - a zero amount with an empty currency code is legal — the free
//     competition case above, which must not be forced to carry a currency
//     it doesn't need;
//   - a non-zero amount with an empty currency code is rejected, since that
//     is precisely the bare-cents value ADR-0005 exists to prevent;
//   - a currency code that is present at all must be well-formed, whatever
//     the amount, so a typo can't be stored and silently mean nothing.
func (m Money) Validate() error {
	if m.AmountCents < 0 {
		return ErrInvalidMoney
	}
	if m.CurrencyCode == "" {
		if m.AmountCents != 0 {
			return ErrInvalidMoney
		}
		return nil
	}
	if !isValidCurrencyCode(m.CurrencyCode) {
		return ErrInvalidMoney
	}
	return nil
}

// IsZero reports whether this is a free entry fee. Exists so callers ask
// "is this competition free?" instead of comparing AmountCents to 0 inline
// and inviting the "zero means unset" misreading Money's doc comment warns
// against.
func (m Money) IsZero() bool {
	return m.AmountCents == 0
}

// isValidCurrencyCode is a cheap ISO 4217 *format* check — three uppercase
// letters — not a full currency registry, mirroring
// internal/payments/domain's helper of the same name and ADR-0005's
// explicitly accepted scope ("v1 only ever populates it with one constant
// value"). Validating against the real ISO 4217 list is not needed until a
// second currency is actually supported.
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
