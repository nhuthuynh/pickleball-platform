# ADR-0005: Every monetary amount carries an explicit ISO 4217 currency code

## Status
Accepted (requirements-research phase, pre-T5)

## Context
`docs/requirements/research-accessibility-i18n.md` §3 found that
`PricingRule.PriceCents`/`pricing_rules.price_cents` already follow half
of Fowler's Money pattern correctly (integer minor units, no float money
anywhere in the domain — confirmed by reading `pricing.go` end to end),
but the other half is missing: there is no currency identifier anywhere
in the schema or domain. The spec's own header states "single launch
market, built i18n/currency-ready for later expansion" — the intent was
already there, just not the column.

## Decision
Add an ISO 4217 `currency_code` (e.g. Postgres `char(3)`) alongside every
`price_cents`-style column — starting with `pricing_rules`, and carried
into the future `payments` table (T6) — even while v1 only ever populates
it with one constant value (the launch market's currency). Amount and
currency are coupled in the domain type from day one (a
`Money{Cents int64; Currency string}`-shaped value, or equivalent),
matching Stripe's own integer-minor-units-plus-currency-code convention
(this project's payment processor, so this isn't just an analogy).

## Consequences
**Pros:** adding a second currency later is a data-population change
(start writing real values into the existing column) rather than a
schema migration touching every table and call site that handles money;
matches the payment processor's own representation, reducing translation
bugs at the Stripe adapter boundary when T6 builds it.
**Cons:** one more column to populate correctly on every pricing-rule and
payment row from the start, for a value that's constant in v1 — pure
carrying cost until multi-currency is real, no functional payoff yet.
**Alternative considered and rejected:** leave currency implicit
(single-market assumption baked into the code, no column) until a second
market is actually being built. Rejected because Fowler's Money pattern
specifically warns against decoupling amount and currency even
temporarily — the risk isn't the missing column today, it's every future
call site written against a bare `int64` cents value that then has to be
found and fixed one by one when currency-readiness is finally needed for
real.
