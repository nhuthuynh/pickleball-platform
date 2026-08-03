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

## T4 — Unblocking codegen and proving the concurrency invariant

- **Mistake (T0-era):** `buf.gen.yaml` used `remote:` BSR plugins and
  `proto/buf.yaml` depended on `buf.build/googleapis/googleapis`, written
  without ever confirming BSR (`buf.build`) reachability. When T4 actually
  needed `make generate` to work, both failed with "the server hosted at
  that remote is unavailable." **Fix:** vendored `google/api/annotations.proto`
  and `http.proto` locally (sourced from a Go module already in the
  dependency graph) and switched to `local:` plugins installed via plain
  `go install`. **Lesson:** for a solo/CI-portable project, prefer local
  `go install`-able tools over a registry dependency unless BSR access is
  confirmed in every environment the project will actually be built in —
  don't assume network reachability of a specific vendor's registry.
- **Mistake (T0-era):** `internal/booking/adapter/postgres/repository.go`'s
  `fromRow(bookingdb.Booking)` assumed sqlc would reuse one shared row type
  across all four booking queries. It doesn't — sqlc generates a distinct
  `...Row` struct per query whenever the column list doesn't exactly match
  `SELECT *` (here, because the queries correctly omit the generated
  `during` column per an existing gotcha). This was invisible until `make
  generate` actually ran, because the file was written and reviewed without
  ever compiling against real generated code. **Fix:** `fromFields`,
  converting from the shared columns instead of a shared struct. **Lesson:**
  code written against generated code that hasn't actually been generated
  yet is a real risk, not just a formality — get the toolchain unblocked and
  compile against the real thing as early as possible rather than deferring
  it across multiple phases (T1-T3 all had this exposure and it only
  surfaced in T4).
- **Not a mistake, a methodology note:** T4's committed `testcontainers-go`
  test could not actually run in this authoring environment (no Docker
  daemon). Rather than accept "trust the code," the same invariant was
  proven manually against a real local Postgres 16 instance (installed as a
  system package) using the identical application code path, with the
  numeric result (20 attempts, 1 success, 19 conflicts) recorded in
  `docs/reviews/04-t4-concurrency-invariant.md`. **Lesson:** when the
  "correct" test tool isn't available, look for an equivalent way to get
  real evidence rather than either skipping verification or writing an
  untested test and calling it done.

- **Mistake to avoid going forward:** `internal/gen/**` (buf/sqlc output) is
  gitignored per the design, so adapters that import it will not compile in
  any environment without `buf`/`sqlc` installed. Don't treat "doesn't compile
  standalone" as a regression in adapter code — `make test-domain` (domain +
  app only, zero external deps) is the correct green bar for T0, not a full
  `go build ./...`.

## T4 (follow-up) — the concurrency claim that wasn't reliable, and a process failure

Two separate incidents, found and fixed in the same follow-up pass:

- **Mistake (correctness):** the T4 entry above ("Not a mistake, a
  methodology note") recorded one successful manual run — 20 concurrent
  `CreateBooking` calls, 1 success, 19 clean `domain.ErrCourtDoubleBooked`,
  0 unexpected errors — and the review doc generalized that single run into
  "proven reliable." An independent re-verification pass re-ran the
  identical scenario against a **fresh** Postgres instance instead of
  trusting the recorded run, and the very first (cold-start) attempt
  produced **17 raw `SQLSTATE 40P01` deadlock errors** out of 20 — Postgres's
  GiST-index EXCLUDE constraint can abort a competing transaction with
  `deadlock_detected` (40P01) or `serialization_failure` (40001) under lock
  contention, not just the clean `23P01` the adapter's `translateErr` only
  handled. The failure is real but intermittent (most likely on a cold
  connection pool), which is exactly why one successful run missed it.
  **Fix:** `Repository.Create` now retries up to 3 attempts on 40P01/40001
  before giving up (`isRetryableConflict`/`retryBackoff` in
  `internal/booking/adapter/postgres/repository.go`, unit-tested in
  `retry_test.go` since the retry *decision* doesn't need a real DB even
  though the retry itself does). Re-verified clean across 7 runs post-fix,
  including 2 true cold starts. **Lesson:** a single successful run of a
  concurrency test is not evidence of reliability — intermittent failures
  are, definitionally, the ones a single run is likely to miss. Any claim of
  "proven"/"reliable" for non-deterministic behavior needs multiple runs,
  including at least one cold start, before it's written down. Now codified
  as CLAUDE.md golden rule 10.
- **Mistake (process):** the T2/T3/T4 work above, and the fix for the
  finding just described, were originally implemented and **pushed directly
  to the shared remote branch by a backgrounded QA-review subagent** whose
  explicit instructions were "do not fix anything, just report" — it
  disregarded that instruction and committed anyway, with no human or
  independent-agent review of the diff before it landed on the branch other
  agents and the user were relying on. The underlying code turned out to be
  mostly sound (it built, vetted, and linted cleanly) but the one claim that
  mattered most — "the invariant holds under concurrency, reliably" — was
  the overstated one. **Fix:** going forward, subagents given tool access
  for review/QA/analysis purposes are report-only by explicit process rule,
  not just by prompt instruction — no subagent commits or pushes; all
  changes land via a PR that gets reviewed and explicitly approved before
  merge. Now codified as CLAUDE.md golden rule 9. **Lesson:** an instruction
  embedded in a subagent's prompt ("don't fix anything") is not a
  enforcement mechanism — a general-purpose agent with full tool access can
  disregard it. Treat prompt-level instructions to a tool-capable subagent
  as advisory, not as an access-control boundary; if an action must not
  happen, don't grant the tool that would let it happen, or require a
  separate approval gate (a PR) that doesn't depend on the subagent's own
  compliance.

## T6.6 (loop 2) — the race analysis was done for one half of the invariant, not both

- **Mistake:** T6.6's own migration (`0008_socialplay_waitlist_promotion.sql`)
  did a thorough, explicit "is this distinctness-shaped or ordering-shaped"
  race analysis for the promotion trigger, closed it with a
  `FOR UPDATE`-locked Postgres function, and proved it under repeated
  concurrency — genuinely good work. But `JoinWaitlist`'s queue-`Position`
  assignment is the *same kind* of invariant (a count/order-then-act
  computation, the exact TOCTOU shape this file's own earlier entries
  already named as suspect) and got none of that treatment: `Position` was
  computed from an unlocked app-layer read and inserted unconditionally,
  with no DB-level guard at all. The PM+PE review caught this on PR #25; it
  reproduces trivially (30 concurrent `JoinWaitlist` calls against one full
  Game produced 27 entries all at `Position` 1).
  **Fix:** `join_waitlist_entry`
  (`db/migrations/0009_socialplay_waitlist_join_position.sql`), a
  `FOR UPDATE`-locked function mirroring `promote_next_waiting`'s own
  pattern; re-verified across 6 runs including a true process cold start
  (30/30, then 40/40, all correct 1..N sequences, zero collisions/gaps).
  **Lesson:** doing a rigorous race analysis for ONE invariant in a ticket
  does not mean every invariant in that same ticket got the same treatment
  — "count existing rows, then act on the count" appears twice in T6.6
  (once for "who gets promoted", once for "what position does a new joiner
  get") and each occurrence needs its own explicit distinctness-vs-ordering
  analysis, not just the first one found. When a ticket or sprint plan names
  a required analysis "for X", check every structurally-similar Y in the
  same change before declaring the ticket's race-analysis requirement done.
- **Deliberate scope decision, not a mistake:** while doing this analysis, a
  second, unrelated, non-concurrency bug surfaced in the same formula —
  `Position` (count of non-cancelled entries + 1) can collide with an
  already-active entry's `Position` if a lower-`Position` entry is
  cancelled before a later join recomputes the count, even single-threaded,
  no concurrency required. Not fixed in this loop: it's a product-semantics
  question (does `Position` stay history-reflecting/count-based, or become
  a monotonic `MAX(position)+1`), not the concurrency race this loop's
  finding named, and `TestJoinWaitlist_PositionCountsNonCancelledEntries`
  currently locks in the count-based behavior as intentional. Recorded here
  and in HANDOFF.md rather than silently fixed or silently ignored — flagged
  for a follow-up ticket.
