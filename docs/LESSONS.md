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

## T5 (in progress) — a capacity invariant with no DB-level backstop

- **Mistake:** T5.4's first pass (PR #14) implemented `RegisterForGame` as
  a get-active-registrations-then-count-then-insert sequence in
  `app.Service`, with only a domain-level check
  (`count >= game.Capacity -> ErrGameFull`) and a Postgres unique index on
  `(game_id, player_id)` — which correctly prevents the *same* player
  double-registering, but does nothing to stop two *different* players
  both passing the in-process count check for the last open slot and both
  succeeding. This is exactly the class of bug CLAUDE.md rule 4 exists to
  prevent ("invariants are enforced in Postgres AND expressed in the
  domain"), and unlike booking's `EXCLUDE` constraint (ADR-0001), there
  was no unconditional DB-level guard on capacity at all — only
  uniqueness. It was also the sprint goal's own headline invariant
  ("capacity... enforced"), not an edge case.
  **Fix:** loop 2 of T5.4 (PM+PE review caught it before merge, per the
  sprint's execution-loop mechanics) added a `BEFORE INSERT/UPDATE`
  trigger on `registrations` that locks the `games` row (`SELECT ... FOR
  UPDATE`) and counts non-cancelled registrations against `games.capacity`
  before allowing the insert, translated to `domain.ErrGameFull` in the
  adapter per rule 5. Verified with a real concurrency test (20 concurrent
  `RegisterForGame` calls, capacity 5) run 6 times including 2 cold
  starts, consistently 5 successes / 15 `ErrGameFull` / 0 unexpected
  errors — not a single run, per rule 10.
  **Lesson:** a domain-level "count existing rows, then insert" check is
  not an invariant under concurrency, no matter how correct the counting
  logic looks in a unit test with a fake repository — it has a TOCTOU
  window the moment two real requests interleave. Any future "N of these,
  capped at M" invariant (this project's waitlist work in T6 is a likely
  next case, per ADR-0006) needs to ask the same question this review
  asked: what actually closes the race at the database level, not just in
  application code? A unique index closes a *distinctness* race; it does
  not by itself close a *counting* race — those need a lock, a trigger, or
  an equivalent DB-enforced check, and this is worth checking explicitly
  every time a ticket introduces a capacity-style limit, not just the
  first time.

## T5 sprint retro

Ceremony 3 per `docs/process/sprint-process.md`, six-role team, held against
`docs/process/t5-sprint-plan.md`, the T5.4 capacity-invariant entry above,
and PRs #11–#15 (`nhuthuynh/white-label`, GitHub-side name
`pickleball-platform`). Findings below, not a single voice — recorded
disagreements are left as disagreements per the "do not manufacture
consensus" rule.

**1. The capacity-invariant gap was a ticket-writing gap first, a review-loop
catch second.** PE/QA: T5.4's ticket text (`t5-sprint-plan.md`) names the
DB-mirror requirement explicitly for *uniqueness* ("a unique constraint on
`(game_id, player_id)`... per CLAUDE.md rule 4") but never names it for
*capacity*, even though the sprint goal's headline clause is "capacity...
enforced" and ADR-0001 already established the precedent (invariant in
Postgres, not just the domain) two sprints ago. The PM+PE review loop caught
it, which is the mechanism working as designed — but it caught it after a
full implement pass, not before one started. BA: this is the same shape of
gap the BA role exists to catch (spec clauses that sound consistent but
don't compose) applied to ticket-writing rather than spec-reading — a
"capacity is enforced" AC and a "translate any new Postgres constraint"
non-functional note read as compatible without anyone checking whether the
second actually covers the first. **Process change (the one this retro
recommends for T6):** Ceremony 1 (backlog refinement) adds an explicit
checklist question PM+PE must answer per ticket, not just per review: *does
this ticket introduce a count-limited invariant ("N of these, capped at
M")? If so, does the ticket text itself require a DB-level guard (lock,
trigger, or equivalent), not only a unique-key guard?* Put in the ticket at
refinement time, this class of bug is caught before loop 1 opens a PR, not
during it.

**2. Every one of PRs #11–#15 hit the identical self-approval error, and
none had merged by retro time.** PdE: all five review passes independently
discovered "GitHub rejected APPROVE — Can not approve your own pull
request" and fell back to COMMENT-with-recommended-verdict, because the
MCP session's credentials and the PR-author account are the same
`nhuthuynh` identity. That's a correct, rule-9-compliant workaround each
time, but it repeated five times in one sprint with no process adjustment
— and as of this retro, `pull_request_read` shows **all five PRs still
open, zero merged**, `mergeable_state` clean on four and "dirty" on #15
(needs a rebase once #11 actually lands). PO: by the sprint-level
Definition of Done ("all in-scope tickets merged... `HANDOFF.md` state
updated for the next sprint to resume from"), T5 is not actually done —
every ticket is review-approved but human-merge-blocked, and the stack
means #12–#15 are all sequentially gated behind #11. PM: this isn't a
quality problem — the content of every review was substantive and
adversarial, not rubber-stamped — but a sprint whose entire output sits
unmerged is a real process risk if it repeats: work piles up faster than a
human reviewer can clear it, and later sprints will keep stacking on an
unmerged base (T6 tickets would stack on top of an already-5-deep unmerged
queue). **Recommended for T6:** either get a second, distinct reviewing
identity so an agent review can actually land as a real GitHub APPROVE, or
treat "COMMENT with recommended-event: APPROVE" as the documented, expected
terminal state in `sprint-process.md` itself (not a workaround) and add an
explicit sprint-level checkpoint for clearing the human-merge queue before
the next sprint's stack grows on top of it.

**3. The 5-loop cap was exercised meaningfully exactly once.** QA/PE: T5.1,
T5.2, T5.3, and T5.5 each converged in loop 1 with a straight recommended
APPROVE; only T5.4 used a second loop (2 of 5) to close the capacity-guard
finding. Notably, ticket size didn't predict loop count — T5.3 (8 points, a
real architecture judgment call on rollback-vs-single-court) also converged
in loop 1, while T5.4 (also 8 points) was the one that needed rework, and
the discriminator was "touches a new DB invariant," not story points. One
sprint and one instance of loop 2 is too little data to conclude 5 is
correctly calibrated — it's neither confirmed too generous nor too tight
yet. **Not changing the cap for T6**; watching whether any T6 ticket needs
3+ loops (which would suggest either the cap is fine and doing its job, or
that ticket was mis-scoped per the loop-mechanics rule) before revisiting
the number.

**4. Ticket sizing held up on the two small tickets; the stacked-diff
`additions` count is not a usable estimation signal.** PO: T5.1 (3 pts, 339
additions, 1 commit) and T5.5 (3 pts, small own-commit diff) both landed
clean in one loop, consistent with their points. T5.2 (5 pts) also
converged cleanly. The two 8-point tickets diverged in outcome (T5.3 clean,
T5.4 needed loop 2) for a reason unrelated to points, per finding 3.
PdE: raw PR `additions`/`changed_files` are misleading under this stacking
model — T5.4 shows 2,751 additions and T5.5 shows 3,058, but both totals
are inflated by carrying every earlier unmerged ticket's diff along; the
reviews themselves correctly scoped to "this ticket's own commit" rather
than the visible diff size, but anyone estimating T6 tickets from these PR
stats alone would badly overestimate T5.4/T5.5's actual size.
**Recommendation:** when using T5 as a reference class for T6 estimation,
use each ticket's own-commit diff (stated in each PR body's "Stacking"
section), not the PR's total `additions`, and treat "does this ticket touch
a new DB invariant or cross a context boundary for the first time" as a
better risk signal than point count alone.

**5. The PM/PE waitlist disagreement's resolution held up, and was tested
harder than expected.** PE: the kickoff note's promise — "a future
waitlist ticket can hook a promotion trigger onto that boundary without an
API-shape change" — got an unplanned early test when loop 2's DB-level
capacity trigger was added: the fix mapped its new `P0001` case onto the
*same* `domain.ErrGameFull` sentinel already wired through to a 409 in
T5.4's gRPC handler, with no proto change, no new error type, and no
handler change. That's about as direct a validation of the "stable hook"
design bet as this sprint could have produced. `docs/adr/
0006-waitlist-data-model-direction.md`'s status line was in fact updated to
point at T6, honoring PM's stated condition for accepting the compromise.
PM: agreed the technical hook is validated — but flags that the underlying
product gap PM raised (a real player hitting a hard, un-queued rejection
when a Game is full) is still fully live in what shipped; nothing about
the trigger fix changes the user-facing behavior, it only made the
rejection *reliable* instead of racy. This is not a reason to revisit the
T5/T6 split now — the scoped compromise did exactly what it said it would
— but it is a reason for T6 planning to treat the waitlist ticket as a
real commitment already validated as buildable, not a nice-to-have that
can slip again; PM's original condition ("not allowed to silently slip
again") should be checked explicitly at T6 Ceremony 1, not assumed.

**6. Several findings are flagged in PR bodies/`HANDOFF.md` prose but have
no tracked ticket yet, across two sprints now.** BA/PO: by count — (a)
Game-cancellation not cascading to Bookings/Registrations (flagged T5.1,
PR #11), (b) `domain.Register` not checking `Game.Status` (flagged T5.2
review, PR #12, now in `HANDOFF.md`), (c) `CancelBooking`'s missing
actor-ownership check, carried since T3 and re-flagged in T5.2 without a
fix, and (d) `CreateGame`/`Game.Cancel()` having no actor parameter at all,
found while scoping T5.5 (PR #15) and explicitly split out rather than
built. All four are real, all four are documented somewhere, and none of
the four is a GitHub issue with a `sprint:t*` label — they exist only as
prose. This is exactly the "flagged, not silently skipped" discipline the
tickets ask for working correctly at the PR level, but it puts the burden
on someone re-reading old PR bodies to not lose them. **Recommended for
T6:** Ceremony 1 for T6 should open these four as real tickets (in or out
of T6 scope, PM/PE's call) rather than let a third sprint pass with them
still living only in prose.

**No finding on:** UX/UI Designer and Product Owner had limited independent
material to contribute this sprint specifically — T5 shipped no
client-facing surface (no Vue/Swift/Kotlin work touched) and the backlog
mechanics PO owns (DoD, ticket ambiguity) were largely clean aside from
finding 6. Noted rather than manufactured, per the same rule this retro
follows for disagreements.
