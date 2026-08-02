# ADR-0007: Account deletion anonymizes PII, it does not cascade-delete rows other records depend on

## Status
Accepted (requirements-research phase, pre-Auth/T5)

## Context
`docs/requirements/research-security-compliance.md` §4 found the spec has
no deletion/erasure mechanism or retention policy anywhere (§4, §5, §8):
no `deleted_at`/anonymization strategy for `users`, no stated retention
period for `payments`/`analytics_events`, and no resolution for the
direct conflict between GDPR Art. 17 (right to erasure) and financial/tax
record-retention law, which requires `payments` rows to survive account
deletion for a jurisdiction-dependent retention period. A naive
`DELETE FROM users WHERE id = ...` with cascading foreign keys would
either violate that retention obligation (if it cascades) or throw a
constraint error at deletion time (if it doesn't) — neither is a real
answer.

## Decision
Account deletion **anonymizes** the `users` row rather than deleting it:
PII fields (name, email, contact info, self-reported level if considered
identifying in combination with other fields) are overwritten/cleared,
but the row and its ID persist so that `payments`, `bookings`, and
`matches` foreign keys stay intact and those tables' own, separately
justified retention (financial/tax record-keeping law) is unaffected by
a player's deletion request. This is the standard "delete vs. anonymize"
resolution the research doc names as the common industry pattern for this
exact GDPR-vs-retention conflict.

A concrete retention period per table (especially `payments` and
`analytics_events`) is a jurisdiction-dependent legal question, not an
engineering one — left open here, to be set once a launch market/legal
counsel input exists (spec §12's still-open "launch market" decision
already gates this).

## Consequences
**Pros:** resolves the erasure-vs-retention conflict before any real user
data exists, rather than discovering it the first time a real user
requests deletion; keeps `payments`/`bookings`/`matches` referential
integrity intact with no special-case FK handling.
**Cons:** "anonymize" requires a defined field-by-field policy (which
`users` columns count as PII to scrub) that doesn't exist yet and needs
BA/PO input once the `users` schema is fully built out (T5+ auth work);
until then this ADR fixes the *strategy*, not the field list.
**Alternative considered and rejected:** hard-delete `users` and let
cascades handle dependents. Rejected outright — it either destroys
financial records a legal obligation requires keeping, or requires ad hoc
per-table exceptions to a cascade that would need to be re-derived and
re-verified every time a new FK to `users` is added, which is a worse
long-term maintenance burden than anonymizing once, centrally.
