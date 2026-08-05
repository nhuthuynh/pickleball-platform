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
## T5 sprint retro

Moved to `docs/process/t5-retro.md` — retro-ceremony output (six-role
findings against a sprint plan) is a distinct artifact type from this
file's incident-postmortem entries, and gets its own phase-tagged file per
sprint going forward (`docs/process/t{N}-retro.md`) rather than living as a
growing section here. See that file for the T5 retro in full.

## T6 — Direct-push-for-docs

- **Mistake:** after the T4 incident that created rule 9, the next three
  artifacts correctly followed it — real branches, real PRs, real "Merge
  pull request #N" merge commits, verified directly against `git log
  --graph` rather than trusted from any prior draft's description of it:

  | Commit | Date | Content | Via |
  |---|---|---|---|
  | `86df8f9` | 07-31 17:51 | T4 deadlock fix + corrected reliability claim | PR #1 |
  | `4c1194c` | 07-31 20:00 | `sprint-process.md` + six role dossiers | PR #2 |
  | `88a0e06` | 08-01 03:53 | README project overview | PR #3 |

  Starting with the four requirements-research docs on 2026-08-01 and
  continuing through both T5/T6 sprint plans, ADRs 0004-0008, the T5 retro,
  `HANDOFF.md` cross-cutting notes, and all 11 docs across the 10-round
  design review, every further process/planning commit went directly to
  `claude/go-backend-pickleball-7up34j` with no branch and no PR — a
  multi-day, multi-sprint pattern, not an isolated slip.

  This entry needed four correction passes to get even its own commit-level
  provenance claims right (miscounted design docs; wrongly attributed a
  PR-merged commit to the direct-push list; then wrongly widened that
  correction too far; then mis-scoped which commits actually touched this
  shared branch versus `main`) — each caught by a fresh review that
  re-checked `git log` rather than trusting the prior pass's claim.
  Deliberately not re-attempting exact commit-level precision here after
  that: four consecutive loops each fixing one small error while
  introducing another is itself the signal that this paragraph's value is
  the pattern it documents, not a hash-by-hash reconstruction, and further
  precision-chasing on a historical footnote stopped being worth its cost.
  Full verified detail, to the extent it matters later, lives in this PR's
  own review comments on GitHub (`chore/docs-governance-and-naming` ->
  PR #29). CLAUDE.md rule 9 ("No direct commits/pushes to the
  shared branch — PR only")
  draws no exception for documentation; the reasoning applied in the
  moment each time was an unstated, unreviewed judgment call — "this is
  just a doc, it's low-risk, a PR would slow down a fast iteration loop"
  — made unilaterally by whichever agent was writing the doc. That is
  structurally the same failure shape as the incident rule 9 was
  originally written to prevent (an agent deciding on its own that a
  change was safe enough to skip review), just generalized from one
  subagent's one bad call to a whole category of file, repeated across
  two sprints.
  **Fix:** surfaced directly to the user rather than resolved
  unilaterally — this file's own guidance ("only take risky actions
  carefully, and when in doubt, ask before acting") applies to a
  governance question about this project's own rules as much as to a
  `git push --force`. The user chose strict enforcement (option B, no
  category exceptions) over formalizing the exception. Rule 9 was
  tightened to say so explicitly (no implied carve-out for docs), and this
  entry plus a docs index/naming convention (`HANDOFF.md`'s **Docs
  index**, CLAUDE.md's **Docs index & naming convention**) landed via a
  real branch + PR — the first artifact under the tightened rule to do so.
  **Lesson:** a written rule with an unstated, self-granted exception is
  not actually the rule — it's the rule until an agent finds a
  low-stakes-looking reason not to follow it, which is exactly the
  category of decision no single agent-in-the-moment should be trusted to
  make alone. If a category of change is genuinely meant to be exempt from
  a governance rule, the exemption needs to be written into the rule
  itself, deliberately, before it's relied on — not inferred from what
  happened to be convenient in each prior instance.

## T9 (2026-08-05) — grpc installs no panic recovery; `net/http` intuition does not transfer

Incident postmortem for PR #89, the critical out-of-band fix in T9. Owed
since that PR, which flagged the entry as warranted and deliberately did
not write it in order to keep a severity-critical fix small and reviewable.
The T9 sprint retro (`docs/process/t9-retro.md`) references this entry
rather than restating it.

- **Mistake:** `cmd/server/main.go` called `grpc.NewServer()` with no
  options at all — **zero interceptors, and therefore no panic recovery
  anywhere in the process**. Separately, all five contexts' Postgres
  adapters carry a `mustUUID(s string) pgtype.UUID` that panics on a
  non-UUID, documented as "fail loudly on a caller bug." That reasoning is
  sound *only* if a malformed ID cannot reach it. It could:
  `Service.GetCompetition` passed `req.GetCompetitionId()` — an HTTP path
  parameter, straight off the wire — through to `Repository.GetByID`, on a
  read that is deliberately public and unauthenticated. The two faults
  lined up into an unauthenticated, total-outage denial of service:

  ```
  curl http://host/v1/competitions/not-a-uuid
  ```

  did not return a 500. **The server process died** — every bounded
  context, every tenant, every in-flight request went with it. No
  credentials, no rate limit to exhaust, no state to set up; a single
  `curl` in a loop is a permanent platform-wide outage. Reproduced before
  fixing on a real server against a real Postgres: the process exited on
  the panic, and `/v1/facilities` — a *different* bounded context —
  returned connection-refused immediately afterward. That cross-context
  death is the point: this was never a Competitions bug, it was a platform
  bug that Competitions happened to expose. A second, independent instance
  of the identical crash was found in `booking.ListCourtBookings`
  (`GET /v1/courts/not-a-uuid/bookings`) while investigating the first.

- **What was missed, and why.** Three separate reasons this survived nine
  phases, all worth naming because each is reusable:

  1. **The framework assumption.** `net/http` recovers panics
     per-connection and keeps serving, so a panicking handler is a 500 and
     nothing more. **grpc installs no `recover()` of its own** — a panic
     in any handler unwinds past the server and terminates the process.
     Review intuition carried over from HTTP handlers is simply wrong
     here, and nobody checked it against grpc's actual behaviour.
  2. **The test fixtures made the bug invisible.** Several fakes minted
     IDs like `"id-1"`, `"court-1"`, `"g-1"` — shapes the real
     `idgen.UUID` never produces and the real Postgres adapter cannot
     store. No existing test could see this crash, because the fixtures
     were the only place those values were ever valid IDs. CI would not
     have caught it either; the fixture infidelity, not the absence of CI,
     is what hid it.
  3. **It was found by accident.** The bug surfaced as a side effect of an
     unrelated PE+QA review of PR #88 (T9.5) — the reviewers were looking
     at the share-token read path, noticed the ID-keyed read next to it
     had no equivalent guard, and pulled the thread. It was not found by
     planned testing, and no ticket in T9 (or T0–T8) would have found it.

- **Fix (PR #89), deliberately two layers because either alone is
  insufficient:** *Layer 1* — a global panic-recovery interceptor
  (`internal/platform/grpcrecovery`, hand-written with stdlib `recover()`,
  no new dependency) wired via `grpc.ChainUnaryInterceptor` /
  `ChainStreamInterceptor`. It protects every handler in all five
  contexts, present and future, against every panic; logs a
  `runtime/debug.Stack()` trace; and returns a constant `codes.Internal`
  **without echoing the panic value**, since `mustUUID`'s panic string
  embeds the caller's raw input and the adapter's package path and these
  endpoints are public — otherwise the fix converts a crash into an
  information disclosure. The stream interceptor is registered too, even
  though no streaming RPC exists, because registering only the unary one
  would leave the first streaming method anyone adds silently unprotected,
  re-opening this exact hole. *Layer 2* — app-layer UUID shape validation
  on the public read paths, so the panic never happens: malformed IDs
  return each context's own not-found answer (`NotFound` for get-shaped
  reads, empty list for list-shaped ones), matching what an
  *unknown-but-well-formed* ID already returns, byte-identically — a
  distinguishable "that isn't even a valid ID" is an enumeration oracle.
  **The validator choice was load-bearing and nearly went wrong:**
  `github.com/google/uuid` was already a dependency and `uuid.Validate`
  was the obvious pick, but it *accepts* `{...}`-braced and `urn:uuid:`
  forms that `pgtype.UUID.Scan` rejects — a guard built on it would have
  passed both through the "validated" boundary and panicked anyway. The
  shipped guard is a strict canonical 8-4-4-4-12 check, deliberately
  *narrower* than what the adapter accepts. **A validator wider than the
  thing it protects is not a validator.**

- **Explicitly not fixed, disclosed rather than assumed:** every
  write/mutating handler taking a caller-supplied ID (`CancelCompetition`,
  `EnterCompetition`, `AddCourt`, `RecordOfflinePayment`,
  `CreateOnlinePayment`, `ConfirmOnlinePayment`) still relies on Layer 1
  alone — a panic there is now a contained 500 with the process alive, not
  a crash, but it is not the specific not-found answer the reads get. This
  was verified rather than assumed (`POST /v1/competitions/not-a-uuid:cancel`
  still panics, returns 500, process still serving). Lower severity since
  these are intended to require real auth once it exists, but each
  unguarded panic still logs a full goroutine stack, so an unauthenticated
  caller can drive attacker-controlled log volume. Follow-up recommended,
  not yet ticketed.

- **Lesson.** *When wiring `cmd/server` for a new bounded context — or
  auditing an existing one — check for panic-recovery coverage
  explicitly. It is not implied by the framework the way it is for
  `net/http`.* More generally: a "this can only happen on a programmer
  error" panic is a claim about reachability, and that claim needs to be
  re-checked every time a new caller appears — `mustUUID`'s reasoning was
  correct when written, and became false the moment a handler passed wire
  input to it. Two concrete checks to carry forward: (a) any helper whose
  contract is "panics on bad input" should have every call site traced to
  a source, and any call site fed from the wire is a bug regardless of how
  well the process recovers; (b) a defence added at a boundary must be
  *narrower* than the thing it defends, and that relationship should be
  measured, not assumed from a library's name.

## T9 sprint retro

Held as `docs/process/t9-retro.md`, following the convention T5 set (see
the `## T5 sprint retro` entry above) and CLAUDE.md's **Docs index &
naming convention**: retro-ceremony output — six-role findings against a
sprint plan — is a distinct artifact type from this file's
incident-postmortem entries, and lives in `docs/process/t{N}-retro.md`.
See that file for the T9 retro in full.

Eight findings, three of them recorded as unresolved disagreements (PE vs.
QA on the shared-checkout collision; PE vs. PM on cross-context planning
timing and on throttling fast merges; PE/QA vs. PdE on whether one-loop
convergence signals quality). The two process changes with the widest
reach: **dispatch isolation becomes a Ceremony 2 checklist item** after
five batch-1 implementers collided in one unisolated working directory and
each rediscovered the same worktree fix independently without anyone
writing it down; and **a regression test for an infrastructure-originated
bug must either run against that infrastructure or assert a property
observable without it**, after T9.5's first regression-test attempt proved
vacuous against an in-memory fake. That second finding is this project's
third encounter with the fake-fidelity boundary (after T4's single-run
concurrency claim and T5.4's TOCTOU-invisible unit tests) and the first
time it has been stated as a general rule — the T9 incident entry directly
above is a fourth instance of the same family, since its own root cause
was test fixtures minting IDs the real system could never produce.
