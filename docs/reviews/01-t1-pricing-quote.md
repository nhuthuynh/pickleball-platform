# Review — T1: Wire Pricing into a Quote use case

## Phase goal
HANDOFF.md T1: expose the pricing-band resolution logic already proven in
T0 (`domain.ResolvePrice`) through a real use case and API — `Service.GetQuote`,
backed by a `port.PricingRuleRepository` + Postgres adapter, reachable via the
`GetQuote` gRPC/REST endpoint already defined in `booking.proto`.

## Process
Implemented TDD-first: `internal/booking/app/quote_test.go` written and run
red (compile failure — `GetQuote` didn't exist) before `Service.GetQuote` was
added. Once green, two independent adversarial subagent reviews were run in
parallel against the diff, loading their briefs from
`docs/agent-operating-handbook.md` Part B: a QA review (B4 — actively try to
break the result) and a Principal Engineer review (B1 — challenge complexity
and one-way doors). Findings below; no consensus was manufactured — each
review ran blind to the other.

## QA review findings and resolution

| # | Finding | Sev | Resolution |
|---|---|---|---|
| 1 | Cross-midnight/multi-day slots could silently match the wrong pricing rule — a doc comment stated the precondition but nothing enforced it | **High** | **Fixed.** Added `fitsSingleCalendarDay` guard in `domain.ResolvePrice`, returning new `domain.ErrPricingSlotSpansMultipleDays`. Regression tests added for the rejected cross-midnight case, the rejected multi-day case, and the exact-midnight-end case that must keep working (`pricing_test.go`). Mapped to `codes.InvalidArgument` (400) in the gRPC handler — this is a caller input error. |
| 2 | The exact-midnight boundary case (`clockTimeOfEnd`'s own worked example) had zero test coverage before this phase | Med | **Fixed as part of #1** — `TestResolvePrice_SlotEndingExactlyAtMidnightStillResolves` now exercises it directly, including the `ClockTime(1440)` value only reachable via a DB-sourced rule. |
| 3 | No Postgres-level guard against overlapping `pricing_rules` windows, contradicting CLAUDE.md rule 4 | Med | **Accepted gap, documented, not fixed.** There is no write path in T1 — `pricing_rules` is migration-seeded only, so there's nothing yet to guard against a concurrent bad write. Recorded explicitly in HANDOFF.md T1 and `docs/LESSONS.md` as a closure item for when `CreatePricingRule` is built, rather than engineering a DB constraint for a table nothing writes to yet. |
| 4 | `ErrAmbiguousPricingRule -> codes.Internal` doesn't distinguish "bad data" from "server bug" | Med | **Fixed** (also raised independently by the PE review — see below) — mapped to `codes.FailedPrecondition`. |
| 5 | `pricing_rules.weekdays` has no `CHECK` constraining values to 0-6 or rejecting an empty array | Low | **Fixed** — added `CHECK (cardinality(weekdays) > 0 AND weekdays <@ ARRAY[0,1,2,3,4,5,6]::smallint[])` to `db/migrations/0003_pricing_rules.sql`. |
| 6 | Postgres adapter reused Booking's `translateErr`, which maps `pgx.ErrNoRows` to the Booking-specific `ErrBookingNotFound` | Low | **Fixed** (also raised independently by the PE review) — see below. |

QA also verified as sound and did not flag: weekday numbering consistency
between the migration comment, seed data, and `time.Weekday`; the 0/1/many
match-counting in `ResolvePrice`; the `ErrNoPricingRule -> NotFound` mapping
against the AC's "clear error" wording; sqlc struct-field plausibility in the
adapter; no concurrency issue (GetQuote is a pure read).

## Principal Engineer review findings and resolution

| # | Finding | Sev | Resolution |
|---|---|---|---|
| 1 | `GetQuote` sits on Booking's `app.Service` rather than a standalone Pricing bounded context per the operating handbook's A1 table | Med, not blocking | **Accepted, noted for later.** Pricing has no aggregate/CRUD/lifecycle yet; `GetQuote` is a 6-line `ListForCourt` + `ResolvePrice` pass-through, trivially extractable later. Noted in HANDOFF.md cross-cutting. |
| 2 | `port.PricingRuleRepository` scoping | — | **No issue found** — single read method, correctly scoped, explicitly documented as read-only for T1. |
| 3 | `NewService` constructor growing positional args (3 now) | Low | **Accepted, noted for later.** Not worth a builder/options pattern at 3 args; revisit at a 4th (likely T5/T6). Noted in HANDOFF.md cross-cutting. |
| 4 | Postgres pricing adapter reused Booking's `translateErr` (`pgx.ErrNoRows -> ErrBookingNotFound`), a latent mismap risk for a future `:one` pricing query | Med | **Fixed** — added `translatePricingErr`, deliberately not doing the Booking-specific `ErrNoRows` mapping, with a comment explaining why it's separate. |
| 5 | Go-specific weekday numbering leaks into the SQL schema | Low | **Accepted, noted for later** in HANDOFF.md cross-cutting; not worth reworking for a solo v1 with proto-generated (not SQL-generated) mobile clients. |
| 6 | Dependency-rule compliance (adapter/domain import direction) | — | **No issue found** — clean in both new files. |
| 7 | `ErrAmbiguousPricingRule -> codes.Internal` identical to the default branch, no client-visible distinction despite ADR-0002's intent | Low | **Fixed** — same fix as QA finding #4 above (`codes.FailedPrecondition`). |

## Alignment against the whole picture
- **Bounded contexts / ubiquitous language:** `Quote`, `PricingRule`, `Band`
  are used consistently with the glossary in
  `docs/agent-operating-handbook.md` A2. The PE-flagged context-boundary
  question (finding 1) is real but explicitly deferred, not ignored.
- **DDD/TDD correctness:** every invariant now has a test that would fail if
  the invariant were removed, including the two the QA pass surfaced that
  weren't there before this review round. `GetQuote` stayed a thin
  orchestration method; no business logic leaked into the app or adapter
  layers.
- **Industry standards checklist:** the QA round is exactly the "strong
  automated-testing culture" + "actively try to break it" standard this
  project holds itself to; the PE round is the "challenge complexity/one-way
  doors" standard. Neither review found anything that blocks the phase —
  the one still-open item (DB-level pricing-rule overlap guard) is a
  documented, reversible gap tied to the absence of a write path, not a
  silently accepted invariant violation.

## Genuine disagreement
None between the two reviews — QA and PE independently converged on the same
two fixable issues (`ErrAmbiguousPricingRule` status code, adapter error
translation reuse) and no direct contradictions. The one place with residual
tension is PE's "is Pricing its own context?" question (finding 1), which
remains genuinely open — recorded as a forward-looking note rather than
resolved, since resolving it now would mean building Pricing/Facilities CRUD
that's out of T1's scope.

## Gate
- `make test-domain` — green (`-race`), including 3 new pricing tests from
  the QA round.
- `go vet` / `golangci-lint run` — clean on every package that compiles
  without generated code.
- `gofmt -l .` — clean.
- `go build ./...` — fails only on the two `internal/gen`-dependent
  packages, exactly as expected (not run in this environment; requires
  `buf`/`sqlc`).
- HANDOFF.md T1 Definition of Done: AC met (table-driven quote tests pass;
  `GetQuote` RPC wired; no-rule case returns `codes.NotFound`); ADR not
  required (no new architectural decision beyond what ADR-0001/0002 already
  cover — the cross-midnight fix is a correctness fix to an already-decided
  design, not a new decision).
