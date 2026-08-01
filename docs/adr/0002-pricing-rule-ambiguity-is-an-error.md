# ADR-0002: An ambiguous pricing-rule match is a domain error, not resolved by priority

## Status
Accepted (T0)

## Context
`domain.ResolvePrice` matches a slot against a court's `PricingRule`s. If
rule windows are configured to overlap (e.g. two "weekday" rules both cover
Monday 09:00-10:00 at different prices), there are two reasonable designs:
(a) pick a winner via an implicit priority (e.g. "most specific band wins",
or "first rule in table order wins"), or (b) refuse to resolve and surface
an error.

## Decision
Treat more than one matching rule as `domain.ErrAmbiguousPricingRule` — a
data-configuration error, not something the domain silently resolves.

## Consequences
**Pros (business/domain):** an owner who misconfigures overlapping pricing
windows (e.g. forgets to end the "weekday" band before "peak" starts) gets a
loud, immediate error surfaced through `GetQuote` (HANDOFF.md T1) rather than
a silently-wrong price charged to a player — money correctness is exactly the
kind of thing that must fail loudly (Microsoft-style security/correctness
rigour on money paths, per the operating handbook's industry-standards
checklist).
**Cons (product):** without an admin-facing pricing-rule editor that
prevents overlapping windows at entry time, an owner could hit this error at
quote time with no clear UI explanation yet — that UI is out of scope for
T0/T1 and is a product-engineering (PdE) follow-up once Facilities/Pricing
CRUD exists.
**Alternative considered and rejected:** "last rule wins" / table-order
priority. Rejected because it makes pricing bugs silent and
order-of-insertion-dependent, which is worse than a hard error for a feature
that directly determines what a player is charged.
