# T19 Sprint Retro

Ceremony 3 per `docs/process/sprint-process.md`, six-role team (briefs:
`docs/agent-operating-handbook.md` Part B), held against
`docs/process/t19-sprint-plan.md` (§A0–§A8), `docs/process/t18-retro.md` and
`docs/process/t17-retro.md` as the precedent and rigor bar, `HANDOFF.md`, and
the real PR/issue history on `nhuthuynh/white-label` (GitHub-side name
`pickleball-platform`) — PRs #214–#216, issues #212–#213.

Every timestamp, closure, label, and code claim below was pulled from
GitHub's own API fields and from direct reads/builds/**mutation tests and an
independently-authored concurrency reproduction against the merged tree at
`697bdbf`** — never inferred from a PR's title, its own account of what it
did, or the sprint plan's forward-looking text (CLAUDE.md rule 10).

**Verification performed before writing a single finding.** `git status`
showed a clean worktree at the shared branch's tip (`697bdbf`, T19.1's own
merge) before this retro's branch was cut. `make generate && go build ./...
&& go vet ./... && make fmt-check && make vet-integration && make
test-domain && make test-adapters && make test-cmd && make test-platform &&
make gate-coverage` were all run directly, not assumed from any PR's own
account:

```
go build ./...                 # clean
go vet ./...                   # clean
make fmt-check                 # OK — gofmt clean
make vet-integration            # clean — compiles T19.2's new
                                 #   -tags=integration file too
make test-domain                # ok, all 12 packages
make test-adapters              # ok, all 22 packages
make test-cmd                   # ok
make test-platform              # ok
make gate-coverage              # OK — all 42 package(s) executed by
                                 #   ci-checks (unchanged from T18 — T19.1
                                 #   adds no new package, T19.2's new file
                                 #   lands in an already-covered package)
```

**Beyond re-running the toolchain, this retro independently reproduced two
mutation checks against the actual merged tree** (not re-read the PR's or
the review's table — re-performed the mutations itself, on the working
tree, restoring cleanly each time, `git status --short` empty after each
restore):

1. Removed `Register`'s cancelled-Game guard (`if game.Status ==
   StatusCancelled { return Registration{}, ErrGameCancelled }`) →
   exactly `TestRegister_RejectsCancelledGame`,
   `TestRegister_CancelledGameCheckedBeforeCapacityAndDoubleRegistration`,
   and `TestRegister_CancelledGameCheckedBeforeGuestAllowance` failed,
   nothing else, confirmed with `go test ./internal/socialplay/domain/...
   -run TestRegister -v`.
2. Removed `JoinWaitlist`'s identical guard → exactly
   `TestJoinWaitlist_RejectsCancelledGame` and
   `TestJoinWaitlist_CancelledGameCheckedBeforeOtherChecks` failed,
   nothing else. Full `internal/socialplay/...` suite green after both
   restores.

**And, per this task's own instruction, this retro went further than any
prior one on exactly one axis: it independently reproduced T19.2's
concurrency claim itself**, not just re-read the PR's and reviewer's
manual-verification tables. Local Postgres was confirmed genuinely
reachable in this environment (`pg_isready` succeeds; `docker info` fails —
the identical finding the PR's own review recorded), so a **fourth**,
independently-authored throwaway verification program was written, run,
and deleted — see finding 3 for the full account and its precise scoring.

**Sprint outcome, stated before the findings that qualify it:** both tickets
(T19.1, 5 points; T19.2, 3 points — 8 total) merged as PR #216 and PR #215,
in that order of *dispatch* but with #215 (T19.2) merging *first*
(19:24:28Z) and #216 (T19.1) merging second (19:26:36Z), both normal review
windows, not the 9s/8s D2-scrutinized shape. Both issues they close (#212,
#213) were themselves filed **during T19's own Ceremony 1** — a first for
this project — which changes the merged-fix sweep's arithmetic in a way
finding 1 works through explicitly rather than reusing a prior sprint's
formula unexamined.

**What this retro found, in one sentence, so the findings do not have to be
read before the headline is known:** T19 shipped two genuinely small,
genuinely additive fixes for gaps that had sat disclosed-but-unfiled for 13
and 14 sprints respectively, both close their issues correctly against the
merged code, both landed with **zero scope drift** from what Ceremony 1
predicted, and T19.2's central epistemic claim is real but has a precise,
unusual shape that is named exactly rather than rounded in either direction:
proven by convergent independent manual reproduction (now three separately-
authored programs, 12 runs, zero flakes), never once executed via its own
committed CI artifact. No incident-grade finding this sprint.

---

## 1. The merged-fix issue sweep — clean, reconciled with the correct (and
different) arithmetic this sprint's shape requires, both closes independently
re-verified against the merged code

**PO.** Per `sprint-process.md`'s DoD, this retro is sweep moment 1; T20's
Ceremony 1 remains the authoritative moment regardless of this result.

**Step 1 — list the open issues, live, at this retro's start:**
`list_issues(state: OPEN)` → **`totalCount: 8`**: #124, #126, #130, #134,
#144, #145, #149, #164.

**Step 2 — reconcile arithmetically, getting the formula right for this
sprint's actual shape rather than reusing a prior sprint's formula
unexamined.** Every prior sprint's `opened_during_sprint` term counted
issues a PR *disclosed and filed* during execution's sibling sweeps. T19 is
structurally different: **T19's own Ceremony 1 filed #212 and #213 itself**,
mid-ceremony, *before* any execution began (`docs/process/t19-sprint-plan.md`
§A3/§A5) — the filing is a Ceremony-1 act, not an execution-time disclosure.
The correct formula for this shape:

```
open_at_end_of_T18's_retro − closed_during_T19 + opened_during_T19
```

where `opened_during_T19` counts **both** the Ceremony-1 filing and any
execution-time disclosure (there was none), and `closed_during_T19` counts
every close performed anywhere in the sprint, including a same-sprint close
of an issue the sprint's own Ceremony 1 opened. T18's retro ended at **8**
open (its own closing line, restated in T19's plan §A1). During T19:
**opened** = 2 (#212, #213, filed by T19's own Ceremony 1) → momentarily 10;
**closed** = 2 (#212 by T19.1/PR #216, #213 by T19.2/PR #215) → back to 8.
`8 − 2 + 2 = 8`. **Matches the live `totalCount: 8` read at this retro's
start exactly**, and both #212 and #213 are confirmed absent from the live
open list (re-fetched, not inferred from their being "taken" in the plan).

**Step 3 — cross-reference merged PRs against the open list, checking
attribution against the merged code, not the closing comment's prose.**

| Issue | `state` (re-fetched now) | `state_reason` | `closed_at` | Closed by / cites |
|---|---|---|---|---|
| #212 | `closed` | `completed` | 2026-08-15T19:26:41Z | Comment naming PR #216, restating the domain+DB-layer fix and the additive-only scope |
| #213 | `closed` | `completed` | 2026-08-15T19:24:33Z | Comment naming PR #215, restating the exact test added and the 8-run manual-verification tally |

Both closes cross-checked against their PRs' own `merged_at`: #216 merged
`19:26:36Z` (close 5s later); #215 merged `19:24:28Z` (close 5s later) —
both same-account, both matching DoD step 5's mandatory form for "closes #N"
titles, continuing the streak this project has now held since T16.

**Both closes verified genuine against the merged code itself, not the
comments' word** — see finding 2 (DoD (a)) for T19.1, finding 3 for T19.2.

**Label conformance, checked rather than assumed.** #212:
`role:principal-engineer`, `type:bug`, `context:socialplay` — conformant,
three-axis, all in the closed sets. #213: `role:qa`, `type:chore`,
`context:payments` — conformant. Both applied correctly at filing time by
T19's own Ceremony 1; unchanged at close.

**Sweep result: clean, fifth consecutive sprint** (after T15, T16, T17,
T18), with this sprint's own arithmetic correctly adapted for the
Ceremony-1-files-then-execution-closes shape rather than mechanically
reused. **T20's Ceremony 1 still re-runs this sweep in full**, per the
standing rule that a prior ceremony's clean result does not discharge the
next one.

## 2. DoD scoring (a) and (b) — T19.1 closes #212 for real, and is the
first ticket to test T18 retro recommendation 1 live: it stated the
narrower, achievable claim up front and that claim held exactly

**PE.** T19.1's own ticket text explicitly avoided T18's mistake
(`docs/process/t18-retro.md` finding 2, recommendation 1): rather than
claim `Register`/`JoinWaitlist` stay "byte-for-byte unchanged," it stated
up front — **"behaviourally additive only... every currently-passing case
that doesn't hit a cancelled Game is untouched, and that is the claim
instruction 6 asks be verified, not a stronger one."** This is the first
sprint since the recommendation landed with a ticket in a position to test
it, and it held.

**DoD (a) — did T19.1 actually close #212, scored directly against the
merged code?** **Yes, at both layers, independently confirmed:**

- **Domain layer.** `internal/socialplay/domain/registration.go:122` and
  `waitlist.go:89` — read directly — both open with `if game.Status ==
  StatusCancelled { return ..., ErrGameCancelled }` as the literal first
  statement in each function body, before every other check. This retro's
  own two mutation checks (above) confirm this is load-bearing, not dead
  code: removing either guard fails exactly the tests written to prove it
  and nothing else.
- **DB layer.** `db/migrations/0023_socialplay_registration_status_guard.sql`
  — read in full — is a `CREATE OR REPLACE FUNCTION` redefinition (not an
  edit) of `enforce_game_capacity()` and `join_waitlist_entry()`. Both
  redefinitions add the cancelled-status check **inside the existing `FOR
  UPDATE` lock**, before the capacity/position count that follows it — no
  new lock, no new race window, confirmed by reading the lock acquisition
  and the new `IF game_status = 'cancelled' THEN RAISE EXCEPTION ...
  ERRCODE = 'P0001'` block's position relative to it in both functions.
  `Registration.Cancel`'s own early-return (`IF NEW.status = 'cancelled'
  THEN RETURN NEW`) is confirmed still ahead of the new check, so a
  player cancelling their own registration is genuinely unaffected — the
  ticket's own stated scope boundary holds in the code, not just in prose.
- **The sentinel and its gRPC mapping were already live before this ticket,
  independently re-confirmed rather than trusted from the PR's or the
  review's claim:** `grep -n "ErrGameCancelled"` across
  `internal/socialplay/domain` and `adapter/grpcapi` shows it already used by
  `Game.EnsureNotCancelled` (`game.go:208`) and already mapped to
  `FailedPrecondition` in `handler.go:474` — corroborating the reviewing
  session's specific claim in its PR #216 review ("confirmed real
  (`game.go:206`), not just `RecordMatchResult`"), independently re-derived
  here rather than taken on trust.

**DoD (b) — is T19.1's own behaviourally-additive-only claim actually
true?** **Yes, checked the only way that actually settles it: diffing the
pre-existing test files, not reading the PR's summary of them.**
`git diff acd8c36 697bdbf -- internal/socialplay/domain/registration_test.go`
and the same for `waitlist_test.go` (`acd8c36` = the tip immediately before
T19.1 merged, i.e. right after T19.2's merge) show **pure additions in
both files — every hunk is a `+`-only block appending new test functions
after an existing one; zero lines removed, zero lines inside any
pre-existing test function touched.** Combined with `internal/socialplay/
app/service_test.go`'s diff (also pure addition — two new tests, no existing
one touched) and the confirmed-unchanged `service.go` (not in either PR's
file list at all — §A6's prediction that `RegisterForGame`/`JoinWaitlist`
already propagate domain errors bare held exactly, no fix was needed), the
claim is true in the strict sense this ticket actually made it, not a
weaker sense standing in for a stronger one the way T18's "byte-for-byte"
did. **This is DoD-claim discipline (T18 retro recommendation 1) working on
its first live test**, not just a recommendation restated.

## 3. DoD (c) — T19.2's concurrency claim, scored precisely: proven by
convergent independent manual reproduction, never once executed via its own
committed CI artifact — a specific epistemic status, named exactly rather
than rounded up or down

**QA.** This is the finding the task most wanted scored honestly, so the
full chain is laid out rather than summarized.

**What the committed test actually is, confirmed by reading it in full.**
`internal/payments/adapter/postgres/concurrency_integration_test.go`
(`//go:build integration`) fires exactly 20 concurrent
`paymentsapp.Service.RecordOfflinePayment` calls at the same
`(payable_type, payable_id)` pair via `testcontainers-go`, and asserts
exactly 1 success and exactly 19 `domain.ErrPaymentAlreadyRecorded` — the
precise count, matching instruction 2 word for word. `make vet-integration`
(compiles, does not run, no Docker needed) passed directly in this retro's
own run above — the committed file is syntactically and type-correct
against the current tree.

**What has never happened, stated plainly rather than glossed over: this
committed test has never executed, anywhere, in this project's history, at
the time of this retro.** No environment in this project's chain — T19.2's
own implementer, its reviewer, or this retro — has had a reachable Docker
daemon. `docker info` fails identically in all three (re-confirmed in this
retro's own environment: `docker info` errors on
`/var/run/docker.sock`; `pg_isready` succeeds against a local Postgres
instance instead). `make vet-integration`'s green result is a **compile**
signal, not an **execution** signal, and no amount of re-running it changes
that.

**What has happened, three times, by three separately-authored programs, on
three separately-provisioned throwaway databases, converging exactly:**

| Verifier | Method | Runs | Cold start? | Result |
|---|---|---|---|---|
| T19.2's implementer | own throwaway program, own throwaway DB (per PR #215's own account) | 5 | 1 true process cold restart | 1 success, 19 conflicts, every run |
| PR #215's reviewer | **own, separately-written** throwaway program (never committed, deleted after use), own fresh throwaway DB | 3 | not claimed as cold-start; corroborating the implementer's cold-start run rather than repeating it | 1 success, 19 conflicts, every run, 0 flakes |
| **This retro** | **own, separately-written** throwaway program (`tmp_t19retro_verify/main.go`, written fresh for this retro, never committed, deleted after use — `git status --short` empty afterward), against a fresh throwaway database (`pickleball_t19retro`, dropped after use) with `db/migrations/0001`–`0023` applied in order | 4 | **1 true cold start** — database dropped and recreated, all 23 migrations reapplied fresh, fresh process, not just `TRUNCATE` | **1 success, 19 conflicts, 0 unexpected, every run** |

**Twelve runs total, across three independently-authored programs, against
three independently-provisioned databases, spanning two independently-taken
process cold starts — zero flakes, zero deviation from the exact 1/19/0
split, at any point.** This retro's own reproduction is reported at the
same level of methodological honesty the task asked for: the program lived
at `/home/user/white-label/tmp_t19retro_verify/`, imported
`paymentspg.Repository`/`paymentsapp.Service` directly (the real production
code path, not a fake standing in for it — the same discipline T14.8/T15.5's
standing warning requires), targeted the real `payments_payable_unique_idx`
constraint via a database seeded from the real migration files in order,
and was deleted along with its throwaway database immediately after use;
`git status --short` was empty both before and after.

**The precise, honest name for this status, stated once rather than left to
be inferred:** T19.2's invariant is **behaviorally proven by convergent
independent manual reproduction** — a materially strong result, now backed
by a third independent line of evidence beyond the two the PR itself
disclosed — **but it is not "CI-proven," and no amount of additional manual
reproduction changes that**, because the committed `-tags=integration` test
that is supposed to be this project's durable, repeatable proof mechanism
has literally never run. This is not "unverified" (that would ignore twelve
convergent runs across three authors) and it is not "proven" in the
unqualified sense CLAUDE.md rule 10 warns against writing without
qualification (a single successful run, or even twelve, from
manually-authored throwaway programs is not the same claim as "the
committed regression test has been executed and passed") — it is a specific
third status this project has not previously had reason to name this
precisely: **manually proven, CI-unexecuted**, and it will remain that way
until a Docker daemon becomes reachable in some future session's
environment and `go test -tags=integration ./...` (or `make test`) is
actually run against this file. **T19.2's own PR and review were already
honest about this distinction** ("not claiming the committed test itself
was run"); this retro's contribution is a third independent data point and
a name for the status precise enough that a future reader does not have to
re-derive it.

**DoD (c), scored:** the concurrency guarantee **is** proven, at the
strength stated above; the **committed test's own execution** is not, and
every party in this chain — implementer, reviewer, and now this retro —
has said so plainly rather than let the strength of the manual evidence
imply a claim about the committed artifact that isn't true.

## 4. DoD (d) — the migration-header-ownership check, applied correctly for
the fourth consecutive sprint carrying it

**PE.** `db/migrations/0023_socialplay_registration_status_guard.sql`'s
header, read in full, states the owning context (Social Play — "T19.1
(closes #212)... applying ADR-0001's dual-invariant pattern... to a THIRD
Social Play invariant"), names both functions it redefines and their
**original owning migrations** by number and PR (`0006` / PR #14, `0009` /
PR #25), and explicitly states the redefinition-not-edit distinction the
check exists to catch ("This is a `CREATE OR REPLACE FUNCTION`
**redefinition**... — **NOT** an edit to those files, per CLAUDE.md's
migration-append-only gotcha"). It goes further than the check's own
minimum bar, the same way T18.1's migration header did (T18 retro finding 2,
DoD (d)): it also documents *why* no new lock/race window is introduced (the
new check reuses each function's pre-existing `FOR UPDATE` lock) and *why*
`Registration.Cancel` is deliberately unaffected. A future ceremony reading
this header gets the ownership answer, the redefinition distinction, and the
concurrency-safety rationale in one read. **Score: yes, applied correctly.**

## 5. Was "file two new issues instead of manufacturing a ticket or scoping
zero" the right call? Re-checked in hindsight against the actual shipped
diffs, not just the planning-time reasoning that justified it — and it holds
up further than it needed to

**PdE and BA.** T19's own Ceremony 1 (§A5) predicted specific file scopes
for both tickets before any code was written. **Checked directly against
each PR's actual `get_files` diff, not the plan's own forward-looking
prose:**

| Ticket | §A6/§A8's prediction | Actual diff |
|---|---|---|
| T19.1 | `internal/socialplay/{domain,app,adapter/postgres-migration}`; explicitly flagged `app/service.go` as "no functional change expected... touched only if instruction 3 finds otherwise" | `registration.go` (+13), `waitlist.go` (+10), `registration_test.go` (+55), `waitlist_test.go` (+37), `service_test.go` (+68, test-only — `service.go` itself is **not in the diff at all**), migration `0023` (+177). **Exactly** the predicted scope; the flagged uncertainty (would `service.go` need a fix?) resolved to "no," precisely as the plan's own hedge anticipated as a live possibility rather than asserted as certain. |
| T19.2 | `internal/payments/adapter/postgres/` (new test file only) | One file: `concurrency_integration_test.go` (+155). **Exactly** the predicted scope, to the file. |

**Zero scope drift on either ticket** — a genuine contrast with this
project's own history of planning-time misses in this exact shape (T17.4's
wrong bounded-context assignment, T15.5's producer/consumer gap). Both
tickets' producer-capability-already-exists claims (§A6) also held exactly:
`ErrGameCancelled` and its gRPC mapping needed no new code (confirmed,
finding 2); `paymentsapp.Service`/`paymentspg.Repository`/the
testcontainers scaffolding needed no new code either (confirmed, finding
3 — the committed file is a pure addition, T19.2 changed zero production
files).

**The counterfactuals, checked rather than asserted.** The alternative
Ceremony 1 rejected — a 0-ticket sprint — would have left both gaps in
exactly the state that violates `sprint-process.md`'s own board-of-record
rule ("an item deferred out of a sprint without an issue is a process
violation, not a judgement call"): #212's gap had already sat disclosed in
`HANDOFF.md`'s prose for 14 sprints past its own stated closing trigger,
and #213's for 13 sprints, neither ever filed. The other rejected
alternative — manufacturing a ticket against one of the 8 blocked issues —
was independently re-checked this retro by re-reading all 8 issues' current
blockers (still D1, still a real-IdP-tenant gap, still Product Owner
questions, still assistive-tech hardware — none of which changed during
T19's own execution, confirmed live). **The call was correct at planning
time and is now additionally confirmed correct in hindsight: both filed
gaps were exactly the size and shape their own disclosure text said they
were, with no execution-time surprise in either direction.**

## 6. D1 and D2 — both re-verified unchanged; D2's own fifth consecutive
null result, T19's own prediction confirmed

**Re-verified this retro, not assumed.** `issue_read(get_comments)` on #144
(D1): still exactly **one** comment, T14.3's original escalation, unchanged
across every sprint since. `docs/adr/0016-*.md`'s (D2) own `## Status`
field: unchanged — **"Escalated — awaiting the user's decision. This ADR
decides nothing."**

**D1's footprint.** Neither T19 ticket touches `CancelBooking`,
`CreateBooking`, or `internal/booking/**` at all — confirmed directly from
both PRs' `get_files` diffs (finding 5's table): T19.1 is entirely
`internal/socialplay/**` plus one migration; T19.2 is entirely
`internal/payments/adapter/postgres/`. D1's footprint neither grew nor
shrank this sprint, matching T19's own Ceremony 1 prediction (§A6) exactly.

**D2.** T19's own Ceremony 1 (§A7) predicted a fifth consecutive sprint with
nothing to score, since neither ticket's design called for reviewer-authored
code. **Checked directly against both PRs' own commit lists** (`get_commits`
on #216 and #215): each carries **exactly one commit**, authored entirely by
the implementing session (`Claude <noreply@anthropic.com>`); both reviews'
own text state "No gaps found. Merging..." with no fix pushed and no gap
found that would have required one. **The prediction holds: a fifth
consecutive sprint (after T15, T16, T17, T18) with no reviewer-authored
gap-fix to score.** Per this project's own stated caution against
over-reading a small, consistent sample, this is recorded as a fifth null
result, not evidence the interim rule can be retired.

**Neither D1 nor D2 is implemented, decided, or guessed at by this retro.**
Both remain exactly as blocked as `sprint-process.md`'s own restriction
lists require.

## No finding on

**No finding on the wave structure.** Both tickets were dispatched as one
disjoint Wave 1 with no Wave-1.5 checkpoint, exactly as the plan's own §A8
stated, and finding 5's file-list table confirms the disjointness was real
(`internal/socialplay/**` vs. `internal/payments/adapter/postgres/**`) —
neither PR's review needed the reconstructed-merge-tree discipline for a
shared-interface reason, though both reviews performed a fresh test-merge
against the live tip anyway (PR #216's review explicitly notes its base was
stale by one merge — T19.2 landed in between — and test-merged against the
actual tip rather than trusting `mergeable_state`), continuing the standing
practice T17 retro named as healthier than only firing where strictly
required.

**No finding on the label taxonomy.** Both #212 and #213 carried a
conformant three-label set from filing (finding 1); no other issue was
opened or touched this sprint to check.

**No finding on PCI conformance.** Neither PR touches a `.proto` file or any
payment-DTO field at all — T19.1 is Go domain code plus a SQL migration
outside Payments entirely; T19.2 is a Go test file with no new message type.
CLAUDE.md rule 11 has nothing to check this sprint, stated rather than
silently skipped.

**No finding on the migration-append-only discipline beyond finding 4's own
scoring.** `0023` is confirmed additive-only against the already-applied
`0006`/`0009` files — `git diff` of both original migration files between
the pre-T19 tip and `697bdbf` shows zero changes to either, corroborating
the header's own claim rather than trusting it.

**No finding on this retro's own reproduction leaving residue.** The
throwaway program (`tmp_t19retro_verify/`) and its throwaway database
(`pickleball_t19retro`) were both removed before this document was written;
`git status --short` was empty immediately before this retro's branch was
cut and remained empty after the reproduction work, confirmed both times.

---

## The sprint goal, scored: what was proven, what shipped exactly as
claimed, and the one place a claim needed the precise name this retro gives
it rather than a rounded one

> *"Two real, disclosed, genuinely unblocked gaps that had never been
> tracked as GitHub issues get built and closed... A Player can no longer
> register for, or join the waitlist of, a Game that is already cancelled
> (#212)... Payments' concurrent-duplicate-recording invariant... gets the
> committed regression proof it has been missing since T6.4 (#213)..."*

**Every clause of the stated goal is met, verified independently, not taken
from any PR's own account.** #212 is closed and the merged code backs the
claim at both the domain and DB layers, mutation-tested by this retro
itself (finding 2). #213 is closed; the committed proof exists, compiles,
and mirrors the required pattern exactly, and the invariant it proves is
independently reconfirmed by a fourth, separately-authored reproduction
program run by this retro (finding 3). Both tickets landed with zero scope
drift from Ceremony 1's own predictions (finding 5). D1 and D2 remain open,
confirmed via the API (finding 6).

**What this retro adds beyond re-confirming the plan's own claims**: a
precise name for T19.2's epistemic status — **manually proven by three
independently-authored programs across twelve zero-flake runs including two
independent process cold starts, but never once executed via its own
committed CI artifact** — offered because "proven" and "unverified" both
misdescribe it, and because CLAUDE.md rule 10 exists precisely to prevent
rounding a real-but-partial verification up into an unqualified claim.

**The agreed honest sentence, which T20's Ceremony 1 should carry into
`HANDOFF.md`'s Task-backlog entry** (`sprint-process.md` Ceremony 1 item 3
requires the retro's form, not a stronger one):

> T19 closed both #212 (`domain.Register`/`domain.JoinWaitlist` now reject
> an already-cancelled Game at both the domain and DB layers, additive only
> — an already-active Registration whose Game is cancelled later is
> unaffected, that remains T16.3's cascade) and #213 (Payments' 20-way
> concurrent-duplicate-recording invariant now has a committed, CI-shaped
> regression test), both filed by T19's own Ceremony 1 and both closed by
> the mandatory "closes #N" mechanism in the same sprint that opened them —
> the merged-fix sweep's arithmetic correctly accounted for that shape
> (`8 − 2 + 2 = 8`, matching the live count) rather than reusing a prior
> sprint's formula unexamined. T19.1 is the first ticket to test T18 retro
> recommendation 1 live — it stated its DoD claim at the achievable
> strength ("behaviourally additive only") up front, and a direct diff of
> the pre-existing test files confirms that claim is exactly true, not an
> overstatement the way T18's "byte-for-byte" was. T19.2's invariant is
> proven by convergent, independent manual reproduction — now a third
> independently-authored program, twelve runs total across two true process
> cold starts, zero flakes — but its committed `-tags=integration` test has
> still never itself executed anywhere in this project's history, a status
> this retro names precisely rather than rounding either direction. Both
> tickets shipped with zero scope drift from Ceremony 1's own file-list
> predictions, confirming in hindsight that filing two disclosed-but-unfiled
> gaps was the correct call over manufacturing a blocked ticket or scoping a
> 0-ticket sprint. D1's footprint held steady; D2 had its fifth consecutive
> sprint with nothing to score, confirming T19's own Ceremony 1 prediction.
> Both remain unanswered by the user.

---

## Recommendations for T20's Ceremony 1 and 2

1. **When a sprint's own Ceremony 1 files a new issue and the sprint's
   execution closes that same issue in the same sprint, the merged-fix
   sweep's arithmetic must count the Ceremony-1 filing as part of
   `opened_during_sprint`, not silently omit it because it didn't come from
   an execution-time sibling-sweep disclosure** — this sprint is the first
   time this exact shape occurred and the formula needed to be worked
   through explicitly rather than reused (finding 1). Worth stating as a
   standing clarification to the sweep's own three-step description in
   `sprint-process.md`, not a new step: the existing formula already
   covers this correctly once `opened_during_sprint` is read to include any
   issue opened at any point in the sprint, Ceremony 1 included — this
   sprint's plan and this retro both got the arithmetic right, but only
   because both stopped to reason about it, not because the formula's
   existing wording made it obvious.
2. **When a concurrency (or any invariant) claim has been manually verified
   multiple times but its own committed CI-executable proof has never
   itself run, name that status precisely — "manually proven,
   CI-unexecuted" or equivalent — rather than writing "proven" unqualified
   or, in the other direction, treating the manual evidence as
   not-yet-evidence** (finding 3). This is the same CLAUDE.md rule 10
   discipline this project already applies to re-run counts, extended to
   the CI-execution axis specifically, since this project's committed
   `-tags=integration` tests have now gone **11+ sprints** with a
   Docker-less authoring environment and this gap is not closing on its
   own.
3. **Continue treating the merged-fix sweep as authoritative regardless of
   this retro's clean result** — T20's Ceremony 1 re-runs the sweep and
   re-verifies both #212's and #213's closes and attribution from the API
   rather than trusting this retro's table (finding 1).
4. **D1 and D2 stay with the user.** No T20 ticket should implement
   `CancelBooking`/`CreateBooking` authorization or a reviewer-authorship
   carve-out; neither should be guessed at. If either answer arrives
   mid-sprint, each escalation's own trigger takes over.
5. **When a future Ceremony 1 predicts a ticket's exact file scope (as
   §A6/§A8 did this sprint), the following retro should check the
   prediction against the actual diff by file, not just by package** — the
   worked example in finding 5 (a table naming the predicted scope against
   the actual `get_files` output, file by file) is cheap, and it is what
   let this retro state "zero scope drift" as a checked fact rather than a
   vibe.

## Sprint-level Definition of Done — scored against what T19's own plan asked

Per `docs/process/t19-sprint-plan.md`'s "Sprint-level Definition of Done,"
four scorings were owed at this retro, stated there so they would not be
improvised — restated here with their answers:

- **(a) Did T19.1 actually close #212, scored directly against the merged
  code?** **Yes** — both the domain-level check (first statement in both
  functions, mutation-tested by this retro itself) and the DB-level
  `CREATE OR REPLACE FUNCTION` guard (inside the existing `FOR UPDATE`
  lock, no new race window) are real and correctly ordered — finding 2.
- **(b) Is T19.1's own behaviourally-additive-only claim actually true —
  did every pre-existing `Register`/`JoinWaitlist` test case keep passing
  unmodified?** **Yes** — confirmed by diffing the test files directly
  between the pre- and post-T19.1 commits: pure additions, zero lines of
  any pre-existing test touched — finding 2.
- **(c) Did T19.2's new concurrency test actually get run repeatedly,
  including a cold start, with the exact counts stated in the PR — not
  merely asserted once?** **The scenario, yes — 12 times now, across three
  independent authors and two independent cold starts, zero flakes. The
  committed test file itself, no — it has never executed in this project's
  history, a precise and specific status named in full at finding 3 rather
  than rounded to "proven" or "unverified."**
- **(d) Did the migration-header-ownership check get applied correctly for
  T19.1's migration?** **Yes** — the header states the ownership answer,
  the redefinition-not-edit distinction, and the concurrency-safety
  rationale, all independently re-confirmed against the migration's actual
  SQL — finding 4.

**Not scoreable by T19 and deliberately not pre-empted:** D1 and D2 remain
the user's (finding 6).

Retro complete. Issue-tracker actions this ceremony: none — both closes
(#212, #213) and their correct attribution were already performed correctly
during the sprint itself, the fifth sprint running with nothing left for the
retro to clean up on the closure axis. Open count at ceremony start: **8**.
Open count now: **8** (unchanged — nothing found here needed a live tracker
action).

Per `sprint-process.md`'s established convention (a retro PR never updates
the Docs-index row that points at it, since that row must cite this PR's own
merge number, which does not exist until it merges): **`HANDOFF.md`'s T19
row is not touched by this PR.** T20's Ceremony 1 corrects it, including its
real PR merge order and the honest-form sentence above, as its first job —
the same standing convention T17's and T18's retros both followed.
