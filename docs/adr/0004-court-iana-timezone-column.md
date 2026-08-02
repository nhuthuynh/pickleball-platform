# ADR-0004: Courts carry an explicit IANA timezone identifier

## Status
Accepted (requirements-research phase, pre-T5)

## Context
`docs/requirements/research-accessibility-i18n.md` §2 found that
`courts` has no timezone column at all (only `id`/`name` today), while
`domain.ClockTimeOf` (`internal/booking/domain/clocktime.go`) already
documents itself as "local to the court's facility" and is used by
`PricingRule.covers` to resolve weekday/time-band pricing. Nothing pins
the `time.Time` a caller passes to the court's actual local zone — it
silently trusts whatever `Location` happens to be attached to the value in
memory. `bookings.starts_at`/`ends_at` are correctly stored as UTC
`timestamptz`, so the gap is upstream of storage, not in it.

This becomes a correctness bug, not just a design nicety, the moment
either of two things happens: (a) a booking falls near local midnight,
where a caller resolving `ClockTimeOf` from an unconverted UTC `time.Time`
can resolve the wrong calendar day/weekday for pricing; (b) T5's
`recurring_hire` generator (D3b, not yet built) needs to expand a weekly
template — per RFC 5545/Google Calendar guidance (same research doc),
anchoring that template to a fixed UTC offset instead of a real IANA zone
causes every occurrence to drift an hour across a DST transition, a
well-documented recurring bug class in scheduling systems (Mozilla
Lightning, FullCalendar, node-ical all have public issues for exactly
this).

## Decision
Add an IANA timezone identifier (e.g. `America/Los_Angeles`) as a
required column on `courts` (or `facilities`, if courts within one
facility are never in different zones — the common case — in which case
the column lives on `facilities` and courts inherit it). Domain code that
needs a court-local clock time or weekday (`ClockTimeOf`, the future
recurring-hire generator) must take this identifier as an explicit input
and convert before computing, never assume the caller already converted.

## Consequences
**Pros:** closes a real, already-latent bug (local-midnight pricing
resolution) before it ships, not after a report from a real user; gives
the recurring-hire generator (T5+) a correct foundation from day one
instead of a retrofit; matches the same "store UTC, render/compute local"
principle the project's booking storage already correctly follows, just
extended one layer further to where local-calendar computation actually
happens.
**Cons:** one more required field on facility/court creation (owner
onboarding, §3.2); needs a migration if courts already exist in a
deployed environment (none do yet — this project is pre-launch).
**Alternative considered and rejected:** defer the timezone column until
the recurring-hire generator is actually built (T5+). Rejected because
`ClockTimeOf`/pricing resolution already has the same silent-assumption
bug today, independent of recurring hire — deferring would mean shipping
a real bug now and calling it a future improvement.
