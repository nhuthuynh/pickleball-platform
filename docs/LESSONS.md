# LESSONS

Running log of mistakes made and how they were fixed, so later phases don't
repeat them. Append, don't rewrite history — each entry is a small postmortem.

## T0 — Bootstrap

- **Mistake:** the session was originally briefed to "resume" an in-progress Go
  backend, but the repository actually contained an unrelated TypeScript
  sample project with none of the described docs or code. Proceeding as if the
  briefing's assumed state existed would have silently fabricated locked
  decisions and history that were never actually agreed.
  **Fix:** stopped and asked the user before writing anything; confirmed this
  is a genuine from-scratch bootstrap using uploaded planning docs as the
  source of truth, and got explicit sign-off to remove the unrelated project
  rather than leaving it alongside the Go code.
## T1 — Pricing Quote use case

- **Mistake:** `domain.PricingRule.covers` compared clock-times only
  (`ClockTimeOf`), with a doc comment stating the slot "must not cross
  midnight" as a precondition — but nothing enforced it. QA's adversarial
  review (docs/reviews/01-t1-pricing-quote.md) found a concrete failure: a
  2-hour slot spanning midnight (Mon 23:00 -> Tue 01:00) would silently match
  a 1-hour "23:00-24:00 Monday only" rule, because clock-time comparison
  alone can't distinguish "01:00 the same day" from "01:00 the next day".
  **Fix:** added `fitsSingleCalendarDay` as an explicit guard in
  `ResolvePrice`, returning `domain.ErrPricingSlotSpansMultipleDays` instead
  of guessing, with regression tests for both the rejected case and the
  exact-midnight-end case that must keep working.
  **Lesson:** a precondition stated only in a doc comment is not a
  precondition — the QA role brief (agent-operating-handbook.md B4) exists
  precisely to catch "invariant asserted but not proven." Any future
  "caller must X" comment on a domain function should get an explicit test
  proving what happens when a caller doesn't.
- **Deliberate scope decision, not a mistake:** `pricing_rules` has no
  DB-level EXCLUDE-style guard against overlapping rule windows, which is a
  literal gap against CLAUDE.md rule 4. Accepted for T1 because there's no
  write path yet (rules are migration-seeded only) — recorded explicitly in
  HANDOFF.md T1 rather than silently skipped, to be closed when
  `CreatePricingRule` is built.

- **Mistake to avoid going forward:** `internal/gen/**` (buf/sqlc output) is
  gitignored per the design, so adapters that import it will not compile in
  any environment without `buf`/`sqlc` installed. Don't treat "doesn't compile
  standalone" as a regression in adapter code — `make test-domain` (domain +
  app only, zero external deps) is the correct green bar for T0, not a full
  `go build ./...`.
