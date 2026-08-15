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

- **Correction, added once #94 and #95 had merged (this entry is
  append-only in spirit, so amending it now is cheaper than a later
  correction entry the way T4's follow-up was):** the coverage table above
  was itself incomplete in two ways this PR's own review process found.
  First, `booking.GetQuote` — a second public, unauthenticated read,
  reaching the same `mustUUID` panic via `PricingRuleRepository.ListForCourt`
  — was missed by PR #89's Layer 2 pass entirely; it shipped with the
  vulnerability this whole entry describes and was only closed by PR #94,
  a follow-up fix mirroring `ListCourtBookings`'s guard exactly. Second,
  PR #89's own `ListCourtBookings` regression test was vacuous in exactly
  the shape the T9 retro's finding 3 names (`docs/process/t9-retro.md`):
  the in-memory fake can't reproduce Postgres rejecting a non-UUID against
  a `uuid` column, so the original assertion passed identically whether or
  not the guard existed. Unlike T9.5's instance of the same bug, this one
  *shipped* rather than being caught in review — the first instance of
  this fake-fidelity family to reach the shared branch instead of being
  stopped by it. PR #94 fixed both: it added `GetQuote`'s guard and
  retrofitted `ListCourtBookings`'s test with a real
  "never reaches the repository" assertion alongside the original. The
  write-handler gap this entry's "Explicitly not fixed" bullet describes
  is unaffected and remains open. Separately, SCRUM-6 (PR #95) has since
  landed a real CI/CD pipeline definition — see `docs/adr/0011-*` — which
  is the structural fix the retro's "candidate finding" on CI-gating
  points at, though as of this correction it is repo-side only (no
  Jenkins job/webhook/branch-protection configured yet, per HANDOFF.md),
  so it does not yet actually run these checks automatically.

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

## T10 (2026-08-10/13) — verified the working tree, shipped the git index

Incident postmortem for the mechanism behind `docs/process/t10-retro.md`
finding 1. Full investigation (exact red-branch duration, who built on
the broken commit, and a second independent instance of the same claim-
vs-reality mismatch this retro caught by directly re-running the test
suite against the commit in question) lives in that retro; this entry is
the reusable mechanism, not a restatement of the sprint-level analysis.

- **Mistake:** during PR #101's (T10.7) merge-conflict resolution against
  the already-merged PR #102 (T10.6) — both PRs had added a guard clause
  to the same line of `internal/payments/app.Service.CreateOnlinePayment`
  — the coordinating session also fixed an unrelated fixture-infidelity
  bug the merge exposed: T10.6's new competition-entry tests in
  `internal/payments/app/service_test.go` used non-UUID literals
  (`"entry-1"`, plus stray `"booking-1"`/`"reg-1"`) that T10.7's new
  `uuidShape` guard on `PayableID` correctly rejects. The fix was made
  correctly: the file was edited, `gofmt -w` was run, `make test-domain`
  was run and came back green, and the fix was reported as verified in
  the PR's own review. **What didn't happen: the edited file was never
  `git add`ed again.** Git's own automatic merge resolution had already
  staged the file with its pre-fix content; the commit went through
  without re-staging, so the commit that was actually pushed and merged
  (`306df838`) silently kept the broken, un-fixed content. The
  `make test-domain` run that passed was real — it read the working tree
  from disk, which had the fix — but the working tree was never what got
  committed. Verifying the working tree proved nothing about what shipped.
  The shared branch was red (`make test-domain` failing, 9 tests in
  `internal/payments/app`, all `"payments: payable id is required"`) from
  `306df838`'s merge (2026-08-10T11:02:38Z) until PR #104's fix merged
  (2026-08-13T08:15:14Z) — **2 days, 21 hours, 12 minutes, 36 seconds**,
  computed from the two merges' own timestamps, not estimated.
- **Fix (PR #104):** identical content to the original intended fix, but
  verified three different ways this time, each closing a different gap
  the first attempt had: (1) `git diff --cached` — the **staged** diff,
  not the working tree, confirming the fix was actually part of what
  would be committed; (2) a full local `make test-domain` run; (3) a
  completely fresh `git worktree add` off the **pushed remote branch**,
  independent of any local working-directory state at all — the only one
  of the three that would have caught the original mistake, since it
  cannot see anything that wasn't actually committed and pushed.
- **What the retro's own re-verification found, beyond the incident PR
  #104 already disclosed:** PR #103 (T10.3) branched from `306df838` —
  the exact commit that was red — and its own PR body claims
  `make test-domain` was green "across the entire repo." Checked directly
  in this retro by checking out `306df838` in an isolated worktree and
  re-running the domain/app suite: 9 failures, same file, same error
  string PR #104 already reported for that commit. PR #103's claim does
  not hold at the commit it says it was checked against. No defect
  shipped from this second instance (T10.3's own diff never touches the
  broken code), but it is a second, independent occurrence of the same
  underlying mistake this entry's title names — a verification claim
  describing a different state than the one actually checked — caught
  only because this retro re-ran the test itself instead of trusting
  either PR's prose.
- **Lesson.** "Tests pass" is a claim about *some* state; after a
  merge-conflict resolution that includes a manual edit, the state that
  matters is the one that gets pushed, which is not automatically the one
  a plain test run just checked. Two concrete, generalizable checks:
  (a) after resolving a conflict with a manual edit, run `git diff
  --cached` (or read the resulting commit directly) before pushing — not
  just re-run the tests, since a passing test run says nothing about
  whether the edit that made it pass was actually staged; (b) when
  auditing a claim that a specific commit was green, check that specific
  commit — a fresh `git worktree` off the actual pushed ref is the only
  one of the three verification layers used here that cannot be fooled by
  stale local state, and it is the cheapest of the three to make
  mandatory for exactly the situations (merge-conflict resolutions, and
  audits of another PR's green claim) where the other two have now both
  independently failed to catch this same mistake once each.

## T10 sprint retro

Held as `docs/process/t10-retro.md`, following the convention T5/T9 set
(see the `## T5 sprint retro`/`## T9 sprint retro` entries above) and
CLAUDE.md's **Docs index & naming convention**.

Six findings, two recorded as unresolved disagreements (PE vs. QA on
whether the three-way "staged diff + local suite + fresh worktree"
verification from finding 1 becomes a mandatory step or stays a cheap
`git diff --cached` floor; PdE vs. QA on how far to generalize finding
4's shared-namespace dispatch-isolation gap beyond migration numbers).
The widest-reaching adopted changes: **verify the staged/pushed state,
not the working tree, after any merge-conflict resolution that includes
a manual edit** (finding 1, full mechanism in the entry directly above);
**`make test-domain`'s scope excludes `adapter/**` by design, so a
fixture-infidelity regression living there can survive indefinitely
unless a full `go test ./...` run happens to hit it** (finding 2, this
project's third and fourth instance of the fixture-fidelity bug class in
one sprint alone); **creation RPCs need their own adversarial checklist
item, distinct from this project's well-worn mutation-on-existing-object
one**, since a caller-supplied ID that becomes a permanent primary key is
a new failure shape T10.2's `CreateUser` was the first to expose (finding
3); and **A7's dispatch-isolation checklist covers file-level overlap but
not shared numeric-sequence resources like migration numbers**, which
collided for the second time in this project's history, unpredicted by
planning both times (finding 4). Finding 6, found while checking finding
4's "closes #N" claims rather than as an assigned investigation point,
is a project-wide discovery: no issue has ever actually auto-closed on
this branch's topology, for any sprint since T5, because GitHub's
`Closes #N` only fires against the default branch and this project has
never merged into it.

## T11 (2026-08-13/14) — a build tag hides code from every gate you can actually run

Incident postmortem for the mechanism behind `docs/process/t11-retro.md`
finding 2. The sprint-level analysis (including the fact that the same
file broke twice in one sprint, and that the learning propagated within
the sprint but is encoded nowhere durable) lives in that retro; this
entry is the reusable mechanism.

- **Mistake:** T11.5 (PR #120) added two parameters to
  `bookingapp.NewService` and did not update the call site in
  `internal/booking/adapter/postgres/concurrency_integration_test.go`.
  That file is gated behind `//go:build integration`, so **every gate the
  session actually ran was blind to it**: `go build ./...`,
  `go vet ./...`, `make test-domain`, and `go test ./... -race -count=1`
  all passed against a file that would not compile. The omission is
  visible in the implementer's own commit message, whose verification
  list names all of those commands and no `-tags=integration` variant.
  The same file, in the same sprint, one PR earlier, had been correctly
  kept in step by T11.2 (PR #118), whose verification list *does* name
  `go vet -tags=integration ./...` — so this was a reintroduction of a
  break class the sprint had already demonstrated it knew about.
- **Root cause, which is not "an implementer forgot":** checked directly
  against the `Makefile`, no target that can run in this environment
  compiles integration-tagged code. `make test-domain` is scoped to
  `domain/`+`app/`. `make ci` runs `generate tidy lint test-domain
  test-tools generate-client lint-web test-web build-web` plus
  `go build ./...` — no `-tags=integration` step anywhere. The only
  target that compiles the file is `make test`, which *runs* the
  testcontainers suite and is therefore hard-guarded by
  `make ci-integration` behind `docker info`, and no session in this
  project's history has had a Docker daemon. So the file was
  structurally unverifiable by every command an agent can run, and the
  two PRs that did check it (T11.2, and T11.6 afterwards) each did so by
  independently remembering an ad-hoc command that appears in no target,
  no checklist, and no doc.
- **Fix:** the reviewer caught it in a fresh worktree off the pushed
  branch, fixed it on the source branch (commit `fdb0ff5`) with
  fail-closed stand-ins mirroring the file's existing
  `unusedFacilityLookup{}` pattern, and re-verified with the unmasked
  exit code (`go vet -tags=integration ./...; echo $?` → `0`, having
  first confirmed the failing case really returned `1` rather than a
  pipe-masked false pass). Broken state lived on the PR branch for
  7m 26s and never reached the shared branch.
- **Lesson.** A build tag is not just a way to skip slow tests — it
  removes the file from the *type-checker's* view too, so "the build is
  green" is a claim about the untagged subset of the repository only.
  Two generalizable rules: (a) whenever a build-tag-gated file exists,
  at least one gate that can run **without** the tag's heavyweight
  dependency must still compile it — `go vet -tags=<tag> ./...`
  type-checks without executing anything and without Docker, which is
  the cheap version of this; (b) changing a constructor's signature
  obliges you to find its call sites **under every build tag in the
  repo**, not just the ones a default build walks. A convention that
  lives only in the memory of whoever wrote the last PR is not a gate,
  which is the same conclusion T10's fixture-ID entry reached about
  shared constant blocks.

## T11 (2026-08-14) — two agents finished their work and the block ended without noticing they never opened a PR

Incident postmortem for the mechanism behind `docs/process/t11-retro.md`
finding 1.

- **Mistake:** two of Wave 1's five tickets (T11.4, T11.9) produced no
  PR from their own implementer session. Both had done correct,
  independently-verifiable work — T11.9 had committed *and pushed* its
  branch and stopped before opening the PR; T11.4 had committed only,
  with the work existing nowhere but a local worktree. The coordinating
  session discovered this ~23 hours later on resume, found both agents
  unreachable, and had to independently verify and publish both
  (`docs/process/t11-retro.md` finding 1 has the exact gaps: 23h 23m 49s
  and 23h 20m 33s from final commit to PR open, against 35s and 56s for
  the two Wave-1 siblings that opened their own).
- **What makes this a process failure rather than an agent-runtime
  failure:** the coordinating session did not stop when the agents did.
  It kept working for another **48 minutes** after the last silent
  agent's final commit — merging two PRs and opening and merging a third
  — and then ended the work block without ever comparing the list of
  tickets it had dispatched into Wave 1 against the list of PRs that
  existed. Both silent tickets had in fact finished their engineering
  work *before* the two that succeeded, so this was not work running out
  of budget mid-task; it was completed work failing at the purely
  mechanical publish step, in a window where a live session could have
  caught it trivially. The true root cause inside the agent sessions is
  not determinable from repository evidence (session exhaustion and an
  interruption leave identical traces), which is exactly why the
  mitigation must not depend on diagnosing it.
- **Fix / lesson.** Treat **absence of a completion notification as
  unknown state, never as "still working" and never as "nothing to
  do."** Before ending a work block, do a roll-call against the sprint
  plan's own dispatch table: for every dispatched, unmerged ticket, state
  whether a PR exists, a remote branch exists, or neither, and name any
  ticket in the "neither" state as an open item. Note specifically that
  polling agent-liveness or listing remote branches would *not* have
  caught T11.4, whose branch was never pushed — only comparing against
  the dispatch list catches the case where the work exists solely as an
  unpushed local commit, which is also the case where it can be lost
  outright.

## T11 sprint retro

Held as `docs/process/t11-retro.md`, following the convention T5/T9/T10
set (see the `## T5 sprint retro`/`## T9 sprint retro`/`## T10 sprint
retro` entries above) and CLAUDE.md's **Docs index & naming convention**.

Six findings, three recorded as unresolved disagreements (PE vs. PdE on
whether to poll background agents or roll-call the dispatch list before
ending a block; QA vs. PE on whether the `-tags=integration` gate needs
CLAUDE.md/DoD prose alongside the Makefile target; PM vs. BA on whether
T12 restores GitHub issues as the board of record or amends
`sprint-process.md` to name the sprint plan instead). The widest-reaching
adopted changes: **add a Docker-free `vet-integration` target
(`go vet -tags=integration ./...`) to the Makefile and to `make ci`**,
since no runnable gate currently compiles build-tag-gated code and the
same file broke twice in this sprint (finding 2, mechanism in the entry
two above); **roll-call the dispatch list before ending a work block**,
since a dispatched ticket with no PR is a finding rather than silence
(finding 1); **extend the dispatch-isolation check from the files each
ticket *adds* to the existing files both will *append to***, naming
`internal/<context>/domain/errors.go`, after T11.1/T11.4 collided there
in the third instance of the shared-namespace class (finding 3); and
**a disclosed gap a reviewer declines to block on must produce a durable
record in that same review**, since T11.5→T11.6's otherwise-exemplary
handoff lived only in a PR body and the coordinating session's memory,
which the sprint plan's own A3 rule forbids (finding 4). Two findings are
about the plan's own accuracy rather than execution: A5's migration
pre-assignment is **the first process fix in this project to fully
prevent its own incident class** (0017/0018, no collision, where T6 and
T10 both collided), A9's placement rulings held and A6 was exceeded — but
A6's four prescribed absence-check signals **would all have missed the
very control T11.6 was told to check**, and A8's dependency table carried
a precedent citation (`socialplay/port.IdentityLookup`) that has never
existed in this repo, in the one cell that carried no evidence marker
(finding 5). Finding 6, found while verifying T11.8's issue closures
rather than as an assigned point: T11 closed all 42 historical issues and
opened **zero** for its own nine tickets, so the board of record
`sprint-process.md` names now contains no evidence T11 happened — and
T11.8's own PR disclosed that exact gap for T10 while standing inside a
live instance of it.

## T12 (2026-08-14) — a platform ticket shipped without the one primitive all three of its consumers needed, and two of them built it independently 68 seconds apart

Incident postmortem for the mechanism behind `docs/process/t12-retro.md`
finding 1.

- **Mistake:** T12.2 built the auth platform (`internal/platform/auth`)
  in deliberate **observe-only** mode — it populates a `Principal` and
  enforces nothing — while A11 Ruling 2 gave each context its own
  `AuthenticatedMethods()` list. Nobody owned the **consumer** of those
  lists: the primitive that turns a method list into a rejected call.
  Verified after the fact with `git log --diff-filter=A`:
  `internal/platform/auth/require.go` was created by **T12.7**, not by
  T12.2. So both Wave-2 tickets independently discovered they could not
  proceed without it and each built their own — T12.7's `require.go`
  (`MethodSet`/`RequireUnaryInterceptor`/`RequireStreamInterceptor`/
  `RequireSubject`) and T12.9's `enforce.go`
  (`RequireAuthentication`/`RequireAuthenticationStream`) — functionally
  identical, architecturally separate, in the same package. Their PRs
  opened **68 seconds apart** (17:09:09Z and 17:10:17Z), so neither
  implementer could see the other's branch, and both said so truthfully
  in their own bodies.
- **What kept the cost to minutes:** both implementers *predicted the
  collision in their own PR bodies before it was adjudicated*, each
  reasoning that the other ticket would need the same primitive. It was
  caught in review of the second PR and resolved on that PR's own source
  branch by deleting `enforce.go`/`enforce_test.go` and consolidating on
  the already-merged `require.go`, with post-merge re-verification
  captured in the merge commit. The sprint plan's stated resolution rule
  for a *different* file (`cmd/server/main.go`: whichever merges second
  resolves on its own branch) generalised correctly to a file the plan
  never named — with the necessary adaptation that "keep both entries" is
  wrong when the two entries are functionally identical.
- **Why the existing safeguard could not have caught it, which is the
  whole lesson.** T11's retro had already extended the dispatch-isolation
  check from "which files does each ticket **create**?" to "which existing
  files will both **append to**?" — the right fix for T11's `errors.go`
  collision, which was a *visibility* failure. T12's is a **provenance**
  failure: at planning time the honest answer to "which files will T12.7
  and T12.9 create in `internal/platform/auth`?" was *"none — T12.2
  provides that package."* Both plans were correct as written. No
  file-overlap question, asked of new files or existing ones, reaches a
  capability that no ticket was assigned to build.
- **Fix / lesson.** **Check the dependency graph for capability
  completeness, not just the file system for overlap.** For every arrow in
  a wave dependency graph, state in one line what the upstream ticket's
  acceptance criteria *deliver* and what each downstream ticket's
  acceptance criteria *consume*, and name any capability that appears on a
  consumer's side and no producer's side. Assign it to exactly one ticket
  before dispatch. Corollary, from the same sprint's recorded PdE/PE
  disagreement (`t12-sprint-plan.md` A14): when a brand-new platform
  capability has three or more first-time consumers, merging and reviewing
  the *first* consumer before dispatching the rest would have prevented
  this outright — the third consumer (T12.8), dispatched one wave later
  with the primitive already present, consumed and extended it correctly
  instead of re-inventing it.

## T12 (2026-08-14) — a whole capability shipped broken because the adapter joining two contexts had no test file at all

Incident postmortem for the mechanism behind `docs/process/t12-retro.md`
finding 2.

- **Mistake:** T12.7 made Booking's actor a verified IdP subject
  (`auth0|abc123`). T12.9 split `User.ID` (a server-minted uuid) from
  `User.Subject` (the IdP string) and added `UserBySubject` for exactly
  that translation — wiring it only into Identity's own handler. Booking's
  cross-context adapter still called `GetUser`, which opens with
  `if !uuidShape.MatchString(id) { return ErrUserNotFound }`. A subject can
  never match a uuid, so **`RequestRecurringHire` returned `NotFound` for
  every caller alive** — a shipped capability, non-functional on the shared
  branch, from two individually well-tested and individually correct
  tickets in the same wave.
- **Why no test caught it, stated precisely, because the obvious diagnosis
  is wrong.** It is *not* that "the calling context fakes the port" —
  faking `port.IdentityLookup` in `internal/booking/app`'s tests is
  correct; that is what a port is for. The gap is one level down:
  `internal/<ctx>/adapter/<other-ctx>/` packages are the *only* code where
  two contexts actually meet, and **`internal/booking/adapter/identity/`
  contained exactly one file — `lookup.go` — from its creation in T11.5
  until the hotfix (PR #151) added the first test.** Nothing at all was
  pointed at the code where the bug had to live. Measured across the repo
  at the time of this entry: of 8 cross-context adapter packages, 3 have a
  real behavioural test, 3 have only a compile-time
  `var _ port.X = (*T)(nil)` assertion, and 2 have no test file at all.
- **What actually found it, which changes the lesson.** Not luck. The
  sprint plan's T12.8 instruction item 2 *ordered* the investigation, in
  writing: *"check the existing relationship in `internal/socialplay` and
  `internal/identity` before deciding, and record the finding either
  way."* T12.8 checked, concluded the two actor names were one identifier
  space, noted its conclusion was safe "because none of them calls into
  Identity," and — in its own words — *"that last clause is load-bearing,
  and checking it surfaced a REAL BUG in already-merged code."* The same
  sprint produced a second instance of the same effect: T12.4's ticket text
  asserted a false sentinel mapping *and* told the implementer to "check
  the existing mapping before adding one," which is why the false premise
  cost nothing.
- **Fix / lesson.** Two, and the cheap one is the second. **(a)** Every
  cross-context adapter needs at least one *behavioural* test driving the
  real other-context `app.Service` over in-memory repository fakes — the
  Docker-free pattern this repo already had in
  `internal/payments/adapter/socialplay/registration_updater_test.go` — and
  it must use a realistically-shaped identifier, because the entire bug was
  that `auth0|abc123` never appeared in a test. A compile-time interface
  assertion is not coverage, and three of these packages currently hold one
  where a reader would expect a test. **(b) The cheapest thing a sprint
  plan can do is tell an implementer to verify the plan's own claim.** Two
  independent defects in one sprint — one live, one latent in a ticket's
  own text — were caught by a written "check X before relying on it"
  instruction rather than by any gate. That instruction costs one clause.
- **Related trap, worth its own line.** The first diagnosis of this bug was
  accurate, complete-looking, and **incomplete**: a second, independent
  guard (`uuidShape` in `internal/booking/app/recurring_hire.go`) fires
  *earlier* on the same call path and returns the *identical*
  `ErrUserNotFound` → `NotFound`. Reading a call chain from either end
  yields a true and sufficient explanation when two independent gates
  return the same error; only executing the whole path distinguishes them.
  The durable rule: **when two or more independent guards on one call path
  can return the same sentinel, a fix that removes one of them is not
  proven by a unit test at that guard's level — it needs a test that
  traverses the whole path.** The remaining half is tracked as issue #152
  with a test that asserts the still-defective behaviour so it cannot go
  quiet, and the original issue was deliberately left open rather than
  closed on the strength of a partial fix.

## T12 sprint retro

Held as `docs/process/t12-retro.md`, following the convention T5/T9/T10/T11
set (see the `## T5 sprint retro`/`## T9 sprint retro`/`## T10 sprint
retro`/`## T11 sprint retro` entries above) and CLAUDE.md's **Docs index &
naming convention**.

Six findings, three recorded as unresolved disagreements (QA vs. PE on
whether a port-contract-change rule is needed on top of the new
dependency-completeness check; PO vs. BA on whether the implementer or the
reviewer is the primary party obliged to open an issue for a disclosed
gap; and A9(a)'s roll-call-vs-polling question, which T12 could not score
because it had no silent agent). All 9 tickets merged in a single
1h47m work block — the fastest sprint in this project's history — plus one
unticketed hotfix. Two of T11's recorded disagreements were **resolved**
this sprint with real evidence: the `vet-integration` prose question
(A9(b)) closed in PE's favour after the sprint's own PR record showed 1
of 8 PRs skipping the check with a reviewer catching it and no break
occurring, and the board-of-record question (A9(c)/T12.6) completed the
first full carry→resolve→implement→verify loop in this project's history.

The widest-reaching adopted changes: **check the wave dependency graph for
capability completeness, not just the file system for overlap** — a new
collision class that the two existing file-overlap questions structurally
cannot see, after T12.2 shipped without the enforcement primitive all
three of its consumers needed (finding 1, mechanism in the entry two
above); **backfill behavioural tests for the five cross-context adapter
packages that lack one**, after a whole capability shipped broken through
the one adapter that had no test file at all (finding 2, mechanism in the
entry above); **extend T12's evidence-marking discipline from the
cross-context dependency table to any factual claim about existing code in
ticket Instructions**, since this sprint's false plan claim was in an
Instruction rather than the table and was harmless only because the same
sentence also said to check it (finding 5); and **a partial fix must say
"partial fix for #N", never "closes #N"** (finding 3).

Finding 4 is the sprint's honest self-assessment and the one most likely
to be misread later: the verified-principal **mechanism** is real,
tested, and consumed by all six bounded contexts (24 RPCs, wire fields
ignored and proven so by mutation, the `CreateUser` squatting DoS closed
and proven closed) — but the goal's literal phrasing, "every
authorization check," does not hold. **Eleven tracked exceptions**,
including several RPCs that have no authorization check at all (#144,
#147, #148), Payments still comparing a verified actor against
caller-supplied ownership facts (#149 — "it closes impersonation; it does
not close fact-fabrication"), one migrated capability still
non-functional (#146/#152), and no way to verify a token from a real
identity provider until a remote JWKS source exists (#137). Finding 6:
A5's disclosed-gap rule produced a durable issue for **19 of 19**
disclosed gaps, a dramatic improvement on T11's zero — but 18 of those
19 were opened by the reviewing session rather than by the PR A5's text
names, so the outcome rests on one reviewer's attention with nothing in
the record signalling the dependency.

## T13 (2026-08-15) — nine PRs said "closes #N", nine issues stayed open, and the sprint's one countable goal failed on the step nobody owns

Incident postmortem for the mechanism behind `docs/process/t13-retro.md`
finding 1.

- **Mistake:** T13 shipped and verified the code for nine tracked issues —
  #123, #129, #135, #136, #138, #146, #148, #152, #154 — and closed **none**
  of them. Six were the "residual auth issues" the sprint's goal promised to
  close *"rather than carry"*, and closing them was the whole point of the
  Ceremony 1 exercise (`docs/process/t13-sprint-plan.md` A1) that ranked
  eleven issues and took eight. The open-issue count went **up**, 19 → 28.
  Verified by arithmetic before being checked individually: 19 inherited + 9
  opened in-sprint = 28 open, so nothing had been closed. #123's
  `updated_at` still equals its `created_at` from T12's Ceremony 1, and its
  `closed_by_pull_requests` count is 0, despite PR #172 being titled
  "(closes #123)".
- **Why this is a repeat, not a first.** `sprint-process.md` DoD step 5
  already documents this exact failure, in bold: *"Writing `Closes #N` in the
  PR body is still good practice … but **it is not sufficient by itself and
  must not be treated as satisfying this step**."* That text exists because
  GitHub's auto-close structurally cannot fire on this project's branch
  topology — every PR merges into `claude/go-backend-pickleball-7up34j`, not
  the default branch — and because it silently never fired for any ticket
  from T5 through T10 until T11.8 repaired the backlog retroactively
  (issue #111, `docs/process/t10-retro.md` finding 6). The rule was written,
  the mechanism was diagnosed, the backlog was repaired, and two sprints
  later it happened again in full.
- **What makes it a process failure rather than forgetfulness.** The same
  session that skipped nine one-line API calls performed nine independent
  mutation checks, nine fresh-worktree toolchain runs, and caught more test
  failures than two PRs claimed. Attention was not scarce; **the step has no
  slot.** T12's A6 added *"every review enumerates the issues it **opened**"*
  and nobody added the symmetric half for issues **closed**, so a reviewer
  working their own checklist finds nothing missing. Every review ends
  "Merging per CLAUDE.md rule 9" and moves to the next ticket — and closing
  happens *after* merge, at the exact moment attention has already moved to
  dispatching the next wave. The one review that did think about it, PR
  #170's, wrote *"This closes issue #154"* as an accomplished fact while the
  call was never made: the record asserted the act because someone wrote that
  it was done.
- **Why it costs more than tidiness.** The issue list is this project's board
  of record for everything outliving a sprint, and Ceremony 1 ranks the
  backlog off it — which is exactly what T13's A1 did, well. That list now
  misdescribes the codebase in six places: #138 claims the auth spine runs in
  no gate (it runs in `make test-platform`), #123 claims booking takes 7
  positional parameters (it takes a `ServiceOptions`), #148 claims
  `ConfirmOnlinePayment` has no owner check (it has one, mutation-proven). A
  future ceremony reading it will re-rank finished work.
- **Fix / lesson.** **A merge is not done until the close call is made** —
  treat "PR merged" and "issue closed" as one indivisible step, never two,
  because the second one lands in the attention gap between tickets. Two
  mitigations, adopted together because this sprint proved either alone is
  insufficient: (i) every review states the issues the PR **closes**, and the
  reviewer performs the close before moving on — the symmetric half of A6;
  (ii) a sprint-level Definition-of-Done check that no issue remains open
  whose fix merged this sprint, verifiable in one API call **by a party other
  than the merger**, so it fails loudly instead of silently. Note
  specifically that getting the *wording* right is not evidence the *act*
  happened: T13 followed A5's "partial fix for #N, never closes #N" rule
  perfectly across all nine PRs while closing nothing. The judgement-laden
  half was performed correctly and the mechanical half was skipped entirely.

## T13 sprint retro

Held as `docs/process/t13-retro.md`, following the convention
T5/T9/T10/T11/T12 set (see the `## T5 sprint retro` … `## T12 sprint retro`
entries above) and CLAUDE.md's **Docs index & naming convention**.

Six findings against the sprint's own plan, the merged code, and the live
PR/issue record (PRs #155–#172, issues #123–#168), with every claim
re-derived here rather than taken from the coordinating session's summary.
**Sprint outcome: all 9 tickets merged on their first loop, no defect reached
the shared branch, and the Wave-1.5 checkpoint held in its strongest form** —
all three Wave-2 branches and the Wave-3 branch are git descendants of the
checkpoint's merge commit, not merely later in time. The two subject↔uuid
seams are fixed under one recorded decision (ADR-0014: translate at each
context's grpcapi `actor()` funnel, never widen), so `RequestRecurringHire`
and `CreateFacility` work for a real caller for the first time.

The sprint's one failed clause is finding 1, in the entry above: six residual
auth issues were fixed in code and never closed on GitHub. Finding 2 is its
structural sibling — T13.1 wrote five new adapter test packages and T13.4
built the Docker-free gate **in the same wave**, and the gate does not run
them; 22 adapter packages' tests are executed by no gate (#157), the third
consecutive sprint of that class after T11's build-tag hole and T12's
`internal/platform/**` hole. The dependency-completeness check T12's retro
introduced is defined over capabilities a downstream ticket *consumes*, so it
structurally cannot see a gate whose coverage set its own siblings are
changing in parallel — A13 arrow 7 dismissed T13.4 with *"nothing consumes
it; it is a gate, not a capability."* Finding 5 is the same shape in
miniature: a two-line `gofmt` fix survived nine PRs and three reviews that
each re-observed it, because `make ci` has no formatting gate and every
ticket's scope discipline *correctly* forbade touching another context's
file — correct rules producing a stuck outcome, with no janitor lane.

Finding 4 records a complete inversion of T12 finding 6: **all eight
execution-opened issues were opened by implementers before their own PRs**
(T12: 1 of 19), resolving the PO/BA disagreement in PO's favour and producing
markedly better-specified issues — while BA's half recurred exactly where BA
predicted, with #167 unlabelled and #168 carrying three labels that are not in
the taxonomy at all. All four carried-forward questions were scored and none
deferred (finding 6): **A9(a)'s roll-call is closed as unfalsifiable in
practice** with a stated reopening trigger (a sprint spanning more than one
work block) rather than a fourth deferral; **QA's port-contract-change rule is
closed in PE's favour** — the dependency-completeness check caught the
semantic half before dispatch and the Go compiler caught the mechanical half
at test-merge, leaving only semantic-only drift, tracked as #164;
**T13.6's Host-only roster scope is resolved in PE's favour**, with QA's
overclaim concern fully honoured by A5's "partial fix" title rule and the
substantive blocker tracked as #168; and **the Wave-1.5 checkpoint's cost is
deliberately left unscored** — PdE's objection was conditional on a second
review loop, T13.2 merged first-loop in 5m14s, and recording "the checkpoint
is cheap" from one fast run would be CLAUDE.md rule 10 violated at the process
level.

## T14 (2026-08-15) — two implementer sessions died mid-ticket; one had pushed, one had not, and the session that recovered both became their author, reviewer and merger

Incident postmortem for the mechanism behind `docs/process/t14-retro.md`
findings 2 and 3. This is the T11 incident's successor: same class
(an agent stops without a PR), opposite detection outcome, new failure
introduced by the recovery itself.

- **What happened.** Two of T14's nine tickets — T14.1 and T14.4, the
  sprint's two 8-point items — had their implementer sessions terminated
  by an **account-level session limit** mid-run. T14.1 had committed
  *and pushed* its branch with no PR opened. **T14.4 had committed only,
  in its worktree, never pushed** — the state `LESSONS.md`'s T11 entry
  names as the one that "can be lost outright" and that neither
  agent-liveness polling nor remote-branch listing can detect. The
  coordinating session found both **inside the same sprint**, recovered
  T14.4's work from the interrupted agent's leftover worktree directory,
  and published both as PRs #181 and #182.
- **The detector worked, and this is the first evidence it has ever
  had.** T11's lesson prescribed one specific mitigation — *compare the
  dispatched-ticket list against the PRs that exist, because that is the
  only check that catches an unpushed local commit*. T14 produced one
  ticket of each kind at once, so the two candidate detectors are
  cleanly separated: branch-listing would have found T14.1 and **missed**
  T14.4. Treat that as settled by evidence now, not by argument.
- **The new failure is in the recovery, not the detection.** The
  recovering session became the recovered work's **author of record,
  its reviewer, and its merger**. PRs #181 and #182 hold the two
  shortest open-to-merge windows in the sprint — **14 and 13 seconds**,
  against a sprint median of 92 seconds — with the review submitted 9
  and 8 seconds after the PR opened. The verification was genuinely
  performed (full toolchain, fresh-worktree re-run, a mutation check)
  but it was performed *as the author, before opening the PR*, so the
  review step had nothing left to check. Both PR bodies additionally
  asserted **"per CLAUDE.md rule 9, I am not merging this myself
  either"** — and both were merged by that same session seconds later.
  A stated safeguard that does not exist is worse than an acknowledged
  absence.
- **Fix / lesson, in three parts.**
  1. **When an implementer session ends without a PR, inspect its
     worktree for unpushed commits *and uncommitted changes* before
     re-dispatching.** Re-dispatching from scratch silently discards
     completed work; listing remote branches does not see it.
  2. **The recovering session may commit, push and open the PR, and must
     say so in the PR's first line** (PRs #181/#182's provenance note is
     the template — that part was done right, voluntarily, and should be
     copied).
  3. **A recovered PR is reviewed by a different party than the one that
     recovered it. Where no independent reviewer can be dispatched, say
     that plainly instead of claiming a safeguard, and hand the
     re-derivation to the sprint's retro** — which must then re-derive
     the PR's headline claim itself and record what it did and did not
     re-check. T14's retro discharged this for T14.1 (re-ran
     `make gate-coverage`, re-derived its package count by hand, and
     mutation-tested the check from scratch against
     `internal/booking/port`) and recorded explicitly that it did **not**
     re-perform T14.4's Host-only mutation check.

## T14 sprint retro

Held as `docs/process/t14-retro.md`, following the convention
T5/T9/T10/T11/T12/T13 set and CLAUDE.md's **Docs index & naming
convention**.

Six findings against the sprint's own plan, the merged code, and the live
PR/issue record (PRs #174–#183, issues #123–#168), with every claim
re-derived at the retro — including a full Docker-free gate run
(`fmt-check`, `test-domain`, `test-platform`, `test-tools`,
`test-adapters`, `test-cmd`, `vet-integration`, all green) and an
independent mutation check of the sprint's marquee deliverable.
**Sprint outcome: all 9 tickets merged, no ticket took a second loop, no
defect reached the shared branch, six issues closed and none opened — the
open count fell 19 → 13, and T14 is the first sprint since the
board-of-record split (T12) to open zero issues.**

**Finding 1 declines to score the merged-fix issue sweep as either a pass
or a repeat of T13, because it is neither.** All six closes are correct
and cite their PRs — and all six landed in an **eleven-second batch**
(13:49:42–13:49:53Z), 65 seconds after the last ticket merged and
immediately before the retro was dispatched, up to **6h45m** after the PR
that earned them. So the per-PR half of the DoD step T14 itself adopted
scored **0/6**, and neither of the sweep's two sanctioned moments (the
retro; the next Ceremony 1) ran — the *merger* swept its own work at
sprint end. That is a **third state the amendment does not name**, correct
in outcome and lacking the "party other than the merger" property the
sprint-level half exists to provide. PO's position is upheld on which
moment gets skipped; PE's cost argument is confirmed by measurement
(the issue list misdescribed the codebase for 6h45m *inside* the sprint)
while PE's remedy is not. The sweep's second disposition — *a partial fix
stays open and someone writes down why* — ran **1 of 4**, and the one
instance (#144) was the issue nobody was required to comment on;
#147, #168 and #149 received no comment at all.

**Finding 2 is the incident above.** Finding 3 records that T13's retro's
reopening trigger for A9(a) (the wave roll-call vs. polling) fired on its
own stated terms and **resolved positively** for the first time, on the
one case no alternative detector covers.

**Finding 4 credits T14.1 and ranks its three proofs.** The mechanical
gate-coverage check found a category **#157 and T14's own Ceremony 1 had
both missed** — neither enumeration scanned `cmd`, so
`cmd/server/main_test.go`'s startup-refusal tests (#136's regression
proof) ran in no gate. All 41 test-holding packages are now executed by
`ci-checks`. The weakest of its three proofs is the one the sprint itself
supplied: T14.4/T14.5's new test files all landed in already-gated
packages, so the dynamic-enumeration property was exercised at file
granularity by siblings and at package granularity only by `tools/devtoken`
(easy case) — the hard case was proven by **mutation**, performed twice,
once independently at the retro.

**Finding 5 is the same shape as T13's finding 1 in miniature, three
times over:** T14.6's label taxonomy was applied to a hand-written list of
three and left #147 and #149 **entirely unlabelled** under a rule that
makes `role:` mandatory; **no review** performed the label-conformance or
closure-enumeration check the same ticket added (0/9 reviews; the single
instance is in PR #175's *body*); and PR #178 and its review both
described **#97 as "still-open"** when it had been closed two days
earlier and was never about the gap they cited — leaving that residual
tracked nowhere. Finding 6 scores all three scorings the plan owed —
**drop A5's dual coverage question, the scheduled-removal condition is
met**; the PE/PO sweep disagreement (finding 1); and the QA/PdE partial-fix
disagreement, correctly **not yet scoreable**, with PdE's premise verified
(no Competition-Admin store exists and nothing blocks building one) so
T15's outcome decides it. **ADR-0012's Q1/Q2 and ADR-0015's D1 remain
blocked on the user**, both re-verified untouched in code, for the sixth
and second consecutive sprint respectively.

## T15 (2026-08-15) — a dependency check verified the producer's capability existed and never checked whether the consumer could reach it

Incident postmortem for the mechanism behind `docs/process/t15-retro.md`
finding 1.

- **Mistake:** T15's Ceremony 1 dependency-completeness check (§A12, GAP B)
  verified that T15.5 (Payments consuming Social Play's new Game-Admin store)
  needed no edit to `internal/socialplay`, because
  `internal/socialplay/app/service.go:1009` already exports
  `ListGameAdmins(ctx context.Context, gameID string) ([]domain.GameAdmin, error)`.
  That is true, and the check correctly concluded T15.5 needed no producer-side
  work. What it never asked is whether **T15.5's own inputs contained a
  `gameID` to pass to that function.** They do not:
  `RecordOfflinePaymentInput` and `RefundPaymentInput`
  (`internal/payments/app/service.go`) carry only `PayableID` — a
  Registration's or CompetitionEntry's own id, never its parent Game's or
  Competition's — and neither `socialplay.app.Service` nor
  `competitions.app.Service` exports a read that resolves one back to the
  other. T15.5 discovered this after being dispatched, built the read-side
  ports and adapters anyway (real, tested, reusable infrastructure), and
  correctly stopped short of wiring them in rather than inventing a new
  caller-supplied `game_id` field that would have relocated the same
  forgery one level up. #168 — the sprint's headline authorization goal —
  stayed open as a direct result.
- **Why this is a distinct mistake from the prior "capability completeness"
  entries, not a restatement of them.** T12's entry (two above) is about a
  capability that **no ticket built at all** — three consumers each
  independently discovering the gap and two of them building duplicate
  primitives. This entry is the opposite shape: the capability **was**
  built, correctly, exactly where the plan said it would be, and the
  dependency check correctly confirmed its existence. The miss is one hop
  further down the same chain: **a capability existing and being callable by
  its own signature is not the same as the specific ticket that needs it
  having, on hand, the value its signature requires.** GAP B's own text —
  *"T15.5 therefore needs no edit to `internal/socialplay` at all"* — is
  true and was the wrong question to have stopped at.
- **Why this was cheaply avoidable, not merely hindsight.** The check that
  would have caught it is not an implementation attempt; it is reading
  `RecordOfflinePaymentInput`'s own field list and asking "does this struct
  already hold a `gameID`?" — one grep, performed by T15.5's own review in
  about one paragraph, entirely after the fact. Nothing about this required
  building anything to discover.
- **Fix / lesson.** When a dependency-completeness check confirms a producer
  capability exists for a downstream ticket to consume, add the symmetric
  question before clearing the dependency: **does the consuming ticket's own
  inputs already contain every argument that capability's signature
  requires?** A capability check that stops at "the function exists and is
  exported" is verifying reachability of the wrong endpoint — the walk needs
  to run from the consumer's actual call site back to the producer, not from
  the producer's export list forward. Threaded into
  `docs/process/sprint-process.md`'s dependency-completeness check as one
  added clause (T15 retro recommendation 1), not a new ceremony step or a
  mechanical tool — this is the first occurrence of exactly this shape, and
  this project's own standing preference is one sweep or one sentence before
  a mechanism, not pre-emptive tooling.

## T15 (2026-08-15) — two PRs said "closes #N," in the same sprint that rewrote the closure rule around exactly that failure, and neither call was made until the retro

Incident postmortem for the mechanism behind `docs/process/t15-retro.md`
finding 2. This is the third sprint in a row this project's issue-closure
bookkeeping has failed in a new shape — T13: the per-PR step scored 0/9 with
every wording rule followed perfectly; T14: the per-PR step scored 0/6, and
the merging session self-swept its own work in an eleven-second batch at
sprint end, correctly but with no independent check; T15: the same sprint
that rewrote the rule to make the sweep primary and the per-PR step optional
produced two PRs whose own titles claimed a close that never happened, and
nobody — not the reviewer, not a self-sweep — caught it before the retro did.

- **Mistake:** PR #191 (T15.6, "closes #185") and PR #189 (T15.7, "closes
  #137") were both merged with fully correct, independently mutation-verified
  fixes for the issues they named. Neither PR's review ever called the
  GitHub issue-close API, and neither PR's review even mentions the target
  issue's live state — contrast PR #192 (T15.4)'s review, which states
  *"will close #147 per instruction 5"* and then does. #185 and #137 both
  sat open, `updated_at` unchanged since their own creation, until this
  retro closed them — verifying each PR's fix independently first, then
  closing with a comment citing the resolving PR, exactly as
  `sprint-process.md`'s sweep mechanics specify.
- **Why this is worse in one specific way than either prior sprint's
  failure, and better in another.** Worse: T14 at least self-swept — the
  merging session made an attempt, however late and however unverified by
  anyone else, to catch its own misses. T15 made **zero** attempt on 2 of
  its 3 "closes" claims; the only reason the codebase's issue tracker is
  accurate today is that a ceremony happened to run before the next
  sprint's Ceremony 1 would have. Better: this is the first time in this
  project's history that the merged-fix sweep — moment 1, "the retro runs
  it and reports it" — actually ran and actually found something, rather
  than reporting a clean-by-coincidence result. The mechanism worked exactly
  as designed; the discipline it exists to backstop did not improve.
- **The correlation worth generalizing:** every T15 review that explicitly
  cited an issue's state as read from the API (T15.1's own new clause, in
  force for this exact sprint) also got the closure right; every review
  that skipped that citation also skipped the close. Two data points is not
  proof, but it names a mechanism cheap enough to lean on: a review that is
  actually looking at the tracker is the review that remembers to update it.
- **Fix / lesson.** A PR titled "closes #N" is a stronger claim than a PR
  titled "partial fix for #N," and treating both under one *optional* early
  close (as T15.1's own same-sprint amendment did) conflates them. The
  former should be a mandatory close-or-explain at review time, not an
  optional convenience — a title that already asserts the fact is false the
  moment the PR merges without the corresponding API call, regardless of
  whether a later sweep eventually catches it. Threaded into
  `docs/process/sprint-process.md` as T15 retro recommendation 2. Separately:
  a prior ceremony's comment on an issue can itself go stale when the
  sprint's actual outcome contradicts it (#149's mid-sprint prediction that
  T15.5 would resolve two of its five facts, corrected by this retro after
  T15.5 resolved zero) — the same shape as T14's #97 misattribution,
  recurring a third time. Neither the closure sweep nor the "read state from
  the API" clause currently obliges anyone to correct content, only to check
  state before closing or leaving open; recommendation 4 closes that gap
  with one added sentence rather than a fourth restatement.

## T15 sprint retro

Held as `docs/process/t15-retro.md`, following the convention
T5/T9/T10/T11/T12/T13/T14 set and CLAUDE.md's **Docs index & naming
convention**.

Seven findings against the sprint's own plan, the merged code, and the live
PR/issue record (PRs #186–#193, issues #124–#185), with every claim
re-derived at the retro rather than taken from any PR's or the plan's own
prose — including three live corrections to the issue tracker performed
during the ceremony itself (closing #185 and #137, both fully resolved in
code but never closed on GitHub despite their own PRs' titles; correcting
#149's stale mid-sprint prediction). **Sprint outcome: all 7 tickets (34
points) merged in one unbroken 1h09m10s work block, no session interruption,
no reviewer-authored gap-fix anywhere in the commit record (ADR-0016's
interim rule held for its first full sprint), wave sequencing exact.**

**Finding 1 is the sprint's central result, argued rather than
both-sided.** T15.5 built real, tested read-side infrastructure onto both
admin stores and then discovered — independently re-verified by this retro
via its own grep of both cross-context `app.Service`s — that Payments has no
way to obtain the `gameID`/`competitionID` its new readers need for the
payable types that require one. #168, the sprint's headline authorization
target, stayed open. This was a genuine, well-verified engineering finding,
not a shortcut, but it was also a real, narrow, cheaply-avoidable planning
miss: the dependency-completeness check that cleared T15.5 for dispatch
verified that the producer capability existed and never checked that the
consumer's own inputs could reach it — full mechanism in the incident entry
above.

**Finding 2 is the second incident above** — two PRs claimed a close their
own reviews never performed, caught only by this retro's sweep, one ceremony
earlier than the amended rule's own stated failure condition would have
caught it. **Finding 3** names the mechanism: every review that cited an
issue's live state from the API also closed correctly; every review that
skipped that citation also skipped the close. **Finding 4** is a second,
unrelated instance of a disclosed-but-unfiled residual (T15.6's FK-race
class, nine other write paths on the same unmapped-`23503` shape #185 was
just fixed for) — the identical shape that produced #185 in the first place,
recurring inside the very PR that closed it, and carried to T16's Ceremony 1
per the board-of-record rule rather than lost a second time. **Finding 5**
is the #149 stale-prediction correction, the same shape as T14's #97
misattribution recurring a third time. **Finding 6** states plainly that D1
and D2 both remain unanswered by the user — third and first deferral
respectively — and that ADR-0016 being authored and merged this sprint is
not progress on the question it asks. **Finding 7** verifies ADR-0016's
interim rule held for its first full sprint: zero reviewer-authored
gap-fixes across six PRs, with one merge-conflict resolution (PR #191,
against a moving base branch) correctly distinguished from the class the ADR
governs, per established T14.1/T14.9 precedent.

Seven recommendations bind T16's Ceremony 1 and 2, most concretely: add one
clause to the dependency-completeness check (walk from the consumer's actual
call site back to the producer, not just confirm the producer's export
exists); make "closes #N" a mandatory close-or-explain at review time rather
than folding it into the same optional-early-close treatment as "partial fix
for #N"; file T15.6's FK-race residual as an issue; and add one sentence
requiring a correction, not just a state-check, when a later fact
supersedes an earlier issue comment. `HANDOFF.md` is deliberately not
touched by the retro PR, per the standing convention that a retro cannot
correctly write the Docs-index row that cites its own merge number — T16's
Ceremony 1 corrects it, as it has every prior sprint.

## T16 (2026-08-15) — a review's own "no test-merge needed" claim was false, and the shared branch's own tip sat uncompilable for 15m21s between two merges that both reported green

**What happened, reconstructed from the commit graph, not from either PR's
prose (full mechanism: `docs/process/t16-retro.md` finding 1).** T16.2 and
T16.3 were both Wave 1 — correctly dispatched with no functional dependency
on each other — and both branched from the same commit before either had
merged. T16.3 merged first, widening `internal/competitions/port.Repository`
with a new method and patching every existing test fake in
`internal/payments/adapter/competitions/` that implemented it — a correct,
complete blast-radius search for the tree it could see. T16.2, working
concurrently in an isolated worktree, was simultaneously authoring a brand
new file in that same package containing a **third** implementer of the same
interface — one T16.3 could not have found, because it did not exist in any
tree T16.3 ever read. When T16.2 merged second, onto the tip T16.3 had
already updated, the new fake was missing the method the interface now
required. `go vet ./internal/payments/adapter/competitions/...` genuinely
failed on the shared branch's own tip from that moment.

**The part that makes this a real process gap and not an unavoidable
timing accident**: T16.2's own review explicitly claimed to have checked
this — *"base already included T16.3 (mergeable_state clean, no test-merge
needed)... all green"* — and that claim is falsifiable, and false, from the
commit graph alone: the reviewed commit's only parent is the pre-T16.3
plan-doc commit, not T16.3's merge. `mergeable_state: clean` certifies the
absence of a *textual* conflict; it says nothing about whether the combined
tree still *type-checks*, and a new file plus a widened interface in
disjoint lines of disjoint files is exactly the shape that produces a clean
merge and a broken build at the same time. The reviewer treated "no
conflict" and "already tested against the merged tree" as the same fact.
They are not.

**Contrast, in the same sprint, with the discipline that would have caught
it**: T16.4's review, reviewing the next PR onto the same branch, checked
out the *bare* shared-branch tip in a separate worktree with none of its own
changes and ran `go vet` directly against it — and found and reported the
break, correctly, before folding in the one-line fix. That is what T16.2's
review claimed to have done and had not.

**Same shape as T10's finding, different mechanism.** T10's retro named "a
green claim checked against the wrong commit" for a fixture fix verified
against a working tree that was never `git add`'d. This is the same failure
class — a verification step that is true of the tree it was actually run
against, misreported as true of a different tree it was never run against —
recurring in a new form: not an unstaged file this time, but an inferred
merge state substituted for a constructed one.

**Fix / lesson, threaded into `docs/process/t16-retro.md` recommendation
1.** When two same-wave tickets both touch the blast radius of one shared
interface in files that will not textually conflict — one widening it, one
authoring a new implementer — the second to merge must verify against an
actually reconstructed post-merge tree (a real local merge, or GitHub's
`refs/pull/<n>/merge` ref), never against `mergeable_state: clean` alone.
This is a narrow, specific verification-method gap — not a dispatch or
wave-design failure; the wave design that let T16.2 and T16.3 run
concurrently was correctly reasoned and is not what this entry is about.
**The broken window itself was short (15m21s, `cd31f86` to `c0e9c7d`) and
was caught and fixed within the same sprint, by the very next PR's own
mandated toolchain run** — worth stating plainly so this entry is not read
as "the shared branch was down for a long time and nobody noticed." It was
down for a short time, on the record, because the DoD's per-ticket testing
step actually requires the toolchain to run, and running it (correctly, at
T16.4) is exactly what surfaced this.

## T16 sprint retro

Held as `docs/process/t16-retro.md`, following the convention
T5/T9/T10/T11/T12/T13/T14/T15 set and CLAUDE.md's **Docs index & naming
convention**.

Seven findings against the sprint's own plan, the merged code, and the live
PR/issue record (PRs #196–#200, issues #124–#198), with every claim
re-derived at the retro rather than taken from any PR's or the plan's own
prose — including a from-the-commit-graph investigation of a genuine defect
that reached the shared branch's own tip this sprint (finding 1, see the
incident entry above). **Sprint outcome: all 3 tickets (16 points) merged in
one unbroken 59m14s work block, no session interruption; both "closes #N"-
titled PRs performed the mandatory close within seconds of merging (2/2,
after T15's 0/2 on the identical shape); the merged-fix sweep reconciles
exactly (`12 − 2 + 1 = 11`) and is the first fully clean sweep — nothing left
for the retro to close — in this project's history.**

**Finding 1 is the sprint's central result, argued rather than accepted at
face value.** A real defect reached the shared branch's own tip: T16.2 and
T16.3, both correctly-dispatched Wave-1 tickets with no functional
dependency between them, concurrently touched the blast radius of the same
widened interface in files that produced no textual conflict. T16.3's own
blast-radius search was complete for what it could see; T16.2's review
explicitly claimed to have verified against the post-T16.3 merged state and
had not — a claim this retro proved false directly from the commit graph,
not inferred. The shared branch sat genuinely uncompilable for 15m21s,
caught and fixed by the next PR's own mandated toolchain run. **Finding 2**
scores all three of T16 Ceremony 1's own process amendments individually on
their first live test — all three held, a genuinely different outcome from
T15's mixed record on its own first-sprint amendments. **Finding 3** is the
merged-fix sweep, clean and reconciled. **Finding 4** confirms #198 (a new,
accurately-described gap in `CreateOnlinePayment`'s competition_entry
branch) is real and flags it for T17. **Finding 5** argues that D1's
unanswered status has now shaped ticket *scope* a second time (T16.3's
court-Bookings half deferred specifically because of it), which is a
materially worse trajectory than being merely re-named each sprint.
**Finding 6** argues T15's own retro — not its Ceremony 1 — had, in its own
session, in writing, everything needed to draft T16.2's exact ticket shape
twenty minutes before it wrote a generic "flag for T16 planning" instead.
**Finding 7** finds nothing to score on ADR-0016's interim rule this sprint
(zero reviewer-authored gap-fixes, second consecutive clean sprint, still a
small sample).

Five recommendations bind T17's Ceremony 1 and 2: verify against a
reconstructed merged tree, not `mergeable_state`, when same-wave tickets
share an interface's blast radius; rank #198; escalate D1 a fifth time and
say its footprint grew; transcribe an already-known unblocking shape into
the next ceremony's ticket text rather than re-deriving it; and re-run the
merged-fix sweep in full regardless of this retro's clean result.
`HANDOFF.md`'s T16 row is corrected by this same retro PR, per this
ceremony's own explicit instruction distinguishing "correcting the prior
ceremony's now-knowable placeholder" from the disallowed "citing this PR's
own future number."

## T17 sprint retro

Held as `docs/process/t17-retro.md`, following the convention
T5/T9/T10/T11/T12/T13/T14/T15/T16 set and CLAUDE.md's **Docs index & naming
convention**.

Five findings against the sprint's own plan, the merged code, and the live
PR/issue record (PRs #202–#207, issues #124–#198), with every claim
re-derived at the retro rather than taken from any PR's or the plan's own
prose. **Sprint outcome: all 5 tickets (17 points) merged in one unbroken
31m46s work block, no session interruption; the merged-fix sweep reconciles
exactly (`11 − 2 + 0 = 9`), a third consecutive clean sweep; the new
same-wave shared-interface rule correctly found zero opportunities to fire,
confirmed by diffing every PR's actual files rather than trusting the plan's
own claim.**

**Finding 1** is the merged-fix sweep, clean and reconciled, both closes
(#198, #195) independently re-verified against the merged code. **Finding 2**
is the sprint's most novel positive result: #195's close depended on
coordinating four independent PRs correctly, not one, and all four got it
right on the mechanism's first live test at that shape — three declined the
close explicitly and correctly, the fourth performed it exactly as promised,
citing all four resolving PRs. **Finding 3** confirms the same-wave rule
found nothing to do because all five tickets were genuinely file-disjoint
(checked via diff, not assumed), and notes every review ran the
reconstructed-merge-tree check anyway as standing practice regardless.
**Finding 4** is this sprint's one real process-gap: Ceremony 1's own ticket
text for T17.4 named the wrong bounded context (`facilities`, not `booking`)
for `discount_rules.facility_id`, a fact carried forward unchecked from
#195's own filing a sprint earlier, when the migration file's own header
comment has said "for the Booking context" since T11 — checked and found to
be the first occurrence of this specific shape in this project's history,
not a repeat, and caught by implementation-time diligence before anything
shipped wrong. No LESSONS entry is warranted for it beyond this index
pointer: unlike T16's finding 1, nothing reached the shared branch's tip in
a broken state. **Finding 5** finds D1's footprint held steady rather than
growing further this sprint (a contrast to T16), and D2 had a third
consecutive sprint with no reviewer-authored gap-fix to score.

Four recommendations bind T18's Ceremony 1 and 2: verify a ticket's target
table's owning bounded context against its migration file before finalizing
the ticket's file path, not only before merging; re-run the merged-fix sweep
in full regardless of this retro's clean result; name the multi-PR
coordinated-close pattern explicitly as the expected shape for any future
issue whose fix spans more than one PR; and D1/D2 stay with the user, not
re-decided here. `HANDOFF.md`'s T17 row is left untouched by this retro, per
the ordinary convention (T16's retro's own self-correction was an
explicitly-argued one-off, not a new standing practice) — T18's Ceremony 1
corrects it.

## T18 sprint retro

Held as `docs/process/t18-retro.md`, following the convention
T5/T9/T10/T11/T12/T13/T14/T15/T16/T17 set and CLAUDE.md's **Docs index &
naming convention**.

Four findings against the sprint's own plan, the merged code, and the live
PR/issue record (PRs #209–#210, issue #167), with every claim re-derived at
the retro rather than taken from the PR's own prose — including three
independent mutation checks re-performed against the actual merged tree
(removing the already-paid guard, the idempotency-claim short-circuit, and
reverting `PublicMethods()` to `nil`), each reproducing exactly the failure
both the PR and its independent reviewer had already claimed. **Sprint
outcome: the sprint's one ticket (8 points) merged as PR #210 in a normal
3-minute review window; issue #167 closed 5 seconds after merge, citing the
PR; the merged-fix sweep reconciles exactly (`9 − 1 + 0 = 8`), a fourth
consecutive clean sweep.**

**Finding 1** is the merged-fix sweep, clean and reconciled, #167's close
independently re-verified against the merged code (`PublicMethods()` now
carries the RPC; `HandleStripeWebhookEvent` is wired end-to-end through
`cmd/server`). **Finding 2** is the sprint's one real, narrow miss: the PR's
own summary states `ConfirmOnlinePayment` is "unchanged, byte-for-byte,"
but the merged diff shows its body was refactored (extracted into a shared
`captureAndMarkPaid`) — exactly as the ticket's own instruction 6 required.
Every property the claim was meant to protect does hold (the authorization
gate, the doc comment, and the RPC handler are all genuinely untouched, line
for line), so this is an overstatement of an otherwise-true functional
guarantee, not a defect — caught here, zero shipped consequence, and
argued in full against DoD item (b), which used the same "byte-for-byte"
wording the ticket's own instructions made impossible to satisfy literally.
**Finding 3** independently reproduces three of the PR's six claimed
mutation checks (the same three the reviewing session's own review comment
already re-checked), all three matching exactly at every layer, and states
plainly which three were and were not independently re-verified rather than
letting the convergence read as broader than it is. **Finding 4** confirms
D1's footprint held steady (no T18 ticket touches Booking) and D2 had its
fourth consecutive sprint with nothing to score — the exact outcome T18's
own Ceremony 1 predicted, now confirmed rather than merely repeated.

Five recommendations bind T19's Ceremony 1 and 2: state a DoD item's
narrower, achievable form up front when a ticket's own instructions make the
strongest form of that claim impossible; re-run the merged-fix sweep in full
regardless of this retro's clean result; a retro re-performing only some of
a review's claimed mutation checks should say exactly which ones; D1/D2 stay
with the user, not re-decided here; and a future review should flag a PR's
own summary prose overstating a claim its own commit message states more
narrowly and accurately, the same "verify the PR's own claim against its own
diff" discipline already applied to code. `HANDOFF.md`'s T18 row is left
untouched by this retro, per the ordinary convention — T19's Ceremony 1
corrects it.
