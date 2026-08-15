# T19 Sprint Plan — Ceremonies 1 + 2

Backlog refinement (Ceremony 1) and sprint planning (Ceremony 2) per
`docs/process/sprint-process.md`, six-role team (briefs:
`docs/agent-operating-handbook.md` Part B). Held against
`docs/process/t18-retro.md` (PR #211, merged 2026-08-15T19:05:45Z),
`HANDOFF.md` including its Docs index, `CLAUDE.md`, and the live PR/issue
state of `nhuthuynh/white-label` (GitHub-side name `pickleball-platform`).

**Every factual claim below was re-derived against the repository and the
GitHub API during this ceremony**, at the merged branch tip (`0bcdeb6`, T18
retro's own merge), rather than taken from the retro's or T18's plan's
prose — CLAUDE.md rule 10 applied to planning. Where this ceremony repeats a
claim it could not independently re-check, it says so.

## §A0 — Correcting T18's Docs-index row and Task-backlog narrative: this
ceremony's first job, per the retro's own instruction

Per `sprint-process.md`'s standing convention (a retro PR never updates the
Docs-index row that points at it, since that row must cite the retro's own
merge PR number, which does not exist until it merges) and per
`docs/process/t18-retro.md`'s own closing line — *"T19's Ceremony 1 corrects
T18's row, including its real PR merge order and the honest-form sentence
above, as its first job"* — **performed directly in `HANDOFF.md` by this
same PR**, not summarized twice here. What was corrected, and how each fact
was independently re-verified rather than copied from the retro's prose:

| Field | What the row now says | Independently re-verified this ceremony |
|---|---|---|
| Retro link | `docs/process/t18-retro.md` (mutation checks independently reproduced; one narrow "byte-for-byte" overclaim caught, zero shipped consequence) | File exists, read in full this ceremony; its own characterization matches its content |
| Reviews cell | PRs #209 (Ceremony 1/2 doc) → #210 (T18.1) → #211 (retro), in that merge order | Re-fetched all three via `list_pull_requests`: `merged_at` **18:32:34Z → 18:56:41Z → 19:05:45Z** — ascending, matches numeric order (checked, not presumed) |
| Task-backlog narrative | The retro's own agreed sentence, quoted verbatim from `docs/process/t18-retro.md`'s "The sprint goal, scored" section, per `sprint-process.md` Ceremony 1 item 3 ("the retro's form, not a stronger one") | Cross-checked word-for-word against the retro's own blockquote — copied, not paraphrased |

**A new T19 row is added** in the same before-the-sprint form, Retro and
Reviews honestly marked "not yet written" / "not yet opened", for T20's
Ceremony 1 to correct in turn — the same convention every prior ceremony has
followed.

## §A1 — The merged-fix issue sweep, run as this ceremony's first substantive
act (per DoD, T19's Ceremony 1 is the authoritative moment)

**Step 1 — list the open issues, live.** `list_issues(state: OPEN)` at
ceremony start → **`totalCount: 8`**: #124, #126, #130, #134, #144, #145,
#149, #164. Matches `docs/process/t18-retro.md`'s own sweep exactly —
re-fetched live rather than trusted, per the standing rule that a prior
ceremony's clean result does not discharge the next one.

**Step 2 — reconcile arithmetically, before reading any issue
individually.** `open_at_end_of_T18's_own_Ceremony_1 − closed_during_T18 +
opened_during_T18`. T18's own Ceremony 1 (§A1 there) left the count at **9**.
During T18's execution: **#167** closed (T18.1, PR #210) — `9 − 1 = 8`. Zero
issues opened during T18's execution (confirmed by re-reading PR #210's own
body and its instruction-11 sibling sweep). **Matches the live `totalCount:
8` read at this ceremony's start exactly.**

**Step 3 — cross-reference merged PRs against the open list.** Only one PR
has merged since T18's retro closed the tree: #211 itself (the retro's own
doc-only PR, which the retro's own closing line states performed no
tracker action — *"Issue-tracker actions this ceremony: none"*). Re-verified
directly rather than assumed: `git log` shows `0bcdeb6` (PR #211's merge) as
the current tip with no descendants before this ceremony's branch was cut.

**Sweep result: clean, fifth sprint running** (after T15, T16, T17, T18).
**T20's Ceremony 1 still re-runs this sweep in full**, per the standing rule
that a prior ceremony's clean result does not discharge the next one.

## §A2 — T18 retro's five recommendations, each given a disposition

| # | Recommendation | Disposition |
|---|---|---|
| 1 | When a sprint plan's DoD item and a ticket instruction make claims about the same code that can't both be true in their strongest form, state the narrower, achievable form up front | **Applied to both T19 tickets below** — neither ticket's instructions ask for a claim its own other instructions would falsify; checked explicitly for each (§A6) |
| 2 | Continue treating the merged-fix sweep as authoritative regardless of the prior retro's clean result | **Executed here** (§A1) — independently re-derived the open count and re-verified from the API rather than the retro's own table |
| 3 | When a later retro re-performs some of a review's claimed mutation checks, it should state exactly which it re-checked and which it did not | Not a Ceremony-1 action item — binds a future retro, not this ceremony. Noted for T19's own retro to apply if it re-performs any of T19's mutation checks |
| 4 | D1 and D2 stay with the user; no T19 ticket implements `CancelBooking`/`CreateBooking` authorization or a reviewer-authorship carve-out | **Executed here** (§A7) — neither implemented nor guessed at; both tickets below independently checked to touch neither surface (§A6) |
| 5 | A future review should flag a PR's own stronger phrasing when a weaker, equally-cheap, and actually-true phrasing was available in the same PR | Not a Ceremony-1 action item — binds a future review. Threaded into both tickets' instructions below as an explicit reminder to state findings at their true strength, not rounded up |

## §A3 — What this ceremony verified, and how

| Claim | How it was checked | Result |
|---|---|---|
| Open-issue count at ceremony start | live `list_issues(state: OPEN)` | `totalCount: 8`, matching #124/#126/#130/#134/#144/#145/#149/#164 exactly |
| Every one of the 8 open issues' blockers are unchanged since T18 retro | `issue_read(get)`/`issue_read(get_comments)` on all 8, live | #144/#149/#124 still D1-blocked (comment counts and content unchanged — #144 still exactly one comment); #164/#145 still need a real IdP tenant this environment cannot provision; #126/#130 still need Product Owner input on a live product question, unchanged bodies; #134 still needs real assistive-technology hardware. **Zero blockers changed state.** |
| D1 (#144) still has exactly one comment | `issue_read(get_comments)` | confirmed — T14.3's original escalation only, unchanged across T14 through T18, now T19 (seventh sprint) |
| D2 (ADR-0016) still unanswered | read `docs/adr/0016-*.md`'s `## Status` | unchanged: "Escalated — awaiting the user's decision. This ADR decides nothing." |
| No PR merged since T18 retro's own PR | `git log --oneline -3`, `git status` | clean, up to date with `origin/claude/go-backend-pickleball-7up34j`, HEAD at `0bcdeb6` (T18 retro's own merge) before this ceremony's branch was cut |
| **A genuinely new finding: two disclosed cross-sprint gaps were never filed as GitHub issues**, in violation of the board-of-record rule (`sprint-process.md`, "an item deferred out of a sprint without an issue is a process violation, not a judgement call") | Read `HANDOFF.md`'s Cross-cutting section in full; cross-checked each disclosed-but-not-fixed item against the live open-issue list (§A3 above) and against the current code | Two items were never filed: (1) `domain.Register`/`JoinWaitlist` never check `Game.Status` (disclosed T5.2, its own stated closing trigger — "when Game-cancellation cascading is built" — fired at T16.3 with nobody acting on it); (2) Payments' 20-way concurrent-duplicate-recording proof was only ever run via an uncommitted throwaway program (disclosed T6.4). Both independently re-verified still-live against the current tree (§A5). **Filed this ceremony as #212 and #213** — the process-violation fix and the backlog-ranking input happen in the same step, per the rule's own "mandatory, not discretionary" wording |
| `domain.Register`'s gap, verified directly | Read `internal/socialplay/domain/registration.go:111` and `waitlist.go:81` in full; read `internal/socialplay/app/service.go`'s `RegisterForGame`/`JoinWaitlist` (lines 389, 554) | confirmed: neither checks `game.Status` anywhere in the call chain; `db/migrations/0006_socialplay_capacity_guard.sql`'s trigger function read in full, no `games.status` reference; `0009_socialplay_waitlist_join_position.sql`'s `join_waitlist_entry()` read in full, same absence |
| `ErrGameCancelled` already exists and is already gRPC-mapped | Read `internal/socialplay/domain/errors.go:239` and `adapter/grpcapi/handler.go`'s `toStatus` (line 419) | confirmed: sentinel exists, used today only by `RecordMatchResult`'s precondition, already mapped to `FailedPrecondition` — reusing it needs no new error type and no new gRPC mapping |
| Payments' concurrency-proof gap, verified directly | Read `internal/payments/adapter/postgres/smoke_integration_test.go` in full; `grep -rn "concurrent\|Concurrent" internal/payments` | confirmed: the only committed test touching `payments_payable_unique_idx` asserts against exactly two **sequential** calls (AC1/AC2), not a concurrent burst; zero other matches for "concurrent" anywhere in `internal/payments/**` |
| Next free migration / ADR numbers | `ls db/migrations`, `ls docs/adr` | `0022` last migration, `0016` last ADR → **`0023`**/**`0017`** free (0023 consumed by T19.1, §A4; no ADR needed — neither ticket makes an architectural decision) |
| `0006_socialplay_capacity_guard.sql` and `0009_socialplay_waitlist_join_position.sql`'s owning context, per their own headers | Read both files' opening comments in full | Both explicitly Social-Play-owned ("Social Play capacity guard", waitlist join position under the same context) — confirmed before assigning T19.1's new migration to `internal/socialplay/**` (§A4) |
| `make generate && make test-domain && make gate-coverage` still green on the unmodified tree | ran directly | `go vet ./...`: clean (exit 0); `make test-domain`: all 12 packages green; `make gate-coverage`: `OK — all 42 package(s) … executed by "ci-checks"` |
| Local worktree matches the shared branch's real tip | `git status`, `git log --oneline -3` | clean, up to date with `origin/claude/go-backend-pickleball-7up34j`, HEAD at `0bcdeb6` before this ceremony's branch was cut |

**Not re-verified by this ceremony, and named as such:** the full `make
test`/`ci-integration` Docker path (no Docker daemon reachable here, same
standing gap as every prior sprint); any claim about what Jenkins would run
(no Jenkins job exists).

---

# Ceremony 1 — Backlog refinement

## §A4 — Migration-header-ownership check, applied for real (T17 retro
recommendation 1, applied every sprint since)

T19.1 (below) is the only T19 ticket that touches a migration, and it does
so by **redefining**, not editing, two already-applied functions.

**Both target functions' owning migrations, checked against their own
header comments — not assumed Social-Play-owned from the ticket's own
framing:**

```
-- Social Play capacity guard (T5.4 loop-2 fix, PM+PE review on PR #14):
```
(`0006_socialplay_capacity_guard.sql:1`) and `0009_socialplay_waitlist_join_position.sql`'s own header, under the same Social Play migration sequence. Both settle it: `enforce_game_capacity()` and `join_waitlist_entry()` are Social-Play-owned, unambiguously, and T19.1's target file path
(`internal/socialplay/domain`, `internal/socialplay/app`, new migration
`0023_socialplay_registration_status_guard.sql`) is correctly scoped. **This
check performing its job on the easy case** (a function already inside the
same context the ticket's issue names) rather than skipped because it
seemed obvious — the same discipline every ceremony since T18 has applied.

**T19.2 touches no migration** — it is a new committed test file only, no
schema or function change (the invariant it proves is already correctly
enforced; only the committed proof is missing). Stated explicitly rather
than silently omitted, per this project's own "silence is indistinguishable
from not having checked" standard.

## §A5 — The whole open backlog, ranked, with a disposition for each — plus
two genuinely new, genuinely unblocked items this ceremony's own
re-verification surfaced

All 8 open GitHub issues, re-verified live (§A3):

| Issue | Ranked | Disposition |
|---|---|---|
| **#144** `CancelBooking`/`CreateBooking` have no authz | **Escalated** | D1 unanswered; §A7. Seventh sprint carrying this |
| **#149** Payments' remaining caller-supplied fact (`booking_host_id`) | **Untouched, correctly** | Blocked on D1 exactly as its own text says |
| **#124** court-Bookings half of the cascade | **Untouched, correctly** | Blocked on D1, same reason as #149; re-verified still open with its T16.3 comment intact |
| **#164** ADR-0014 actor-column conformance | **Deferred, still blocked** | Needs a real IdP tenant, unreachable from this environment; unchanged since T14 |
| **#145** pre-existing UUID rows vs. `Principal.Subject` | **Deferred** | Same real-IdP-tenant blocker as #164; unchanged |
| **#126** real per-Game price field | **Deferred** | Needs Product Owner input on whether the price is per-Game, per-Registration, or per-head; unchanged since T8.10 |
| **#130** refunding a `no_show_fee` Payment | **Deferred** | Carries its own stated open product question ("confirm first that reversing a no-show fee is the product behaviour actually wanted"); unchanged |
| **#134** WCAG manual screen-reader pass | **Deferred** | Needs a `role:ux-ui-designer` pass with real assistive-technology hardware this environment cannot provide; T19 ships no UI change |

**Every one of the 8 tracked issues remains genuinely blocked, independently
re-verified rather than assumed** (§A3). This is the outcome the task
handed this ceremony explicitly flagged as possible — **and this ceremony's
own re-verification confirms it is exactly what has happened**: zero of the
8 open issues are actionable this sprint without guessing at D1, a real IdP
tenant, a Product Owner decision, or assistive-tech hardware.

**Rather than manufacture a ticket by guessing at one of those blockers, or
scope a 0-ticket sprint, this ceremony's own backlog re-verification (§A3)
found real, disclosed, genuinely unblocked work that had never gone through
the mandatory issue-filing step** — the same category `sprint-process.md`'s
board-of-record rule exists for, just never actually filed. Filed this
ceremony as **#212** and **#213** (§A3), both re-verified against the live
code rather than trusted from `HANDOFF.md`'s own prose, and both taken:

| Issue | Ranked | Disposition |
|---|---|---|
| **#212** `Register`/`JoinWaitlist` admit a cancelled Game with no error anywhere in the stack | **Taken** | **T19.1** — a live, unguarded correctness gap (not a blocked one): the sentinel and its gRPC mapping already exist, this sprint only needs the check that was never wired to them (§A3) |
| **#213** Payments' concurrency proof was never committed | **Taken** | **T19.2** — a disclosed test-coverage gap with no blocker at all: the invariant is already correctly enforced, only the committed regression proof (mirroring booking's/socialplay's own committed pattern) is missing |

**Why these two and not a bigger sprint.** No other item in `HANDOFF.md`'s
Cross-cutting section is both real and unblocked at this size: the
`golang-migrate`/`goose` migration-tooling swap is real and genuinely
unblocked but is an undersized-ticket risk in the other direction — its own
note gives no fixed scope (which tool, what migration path for the existing
`0001`–`0022` files, whether the `0005` payments/socialplay collision gets
renumbered in the same pass) and taking it without first scoping it would
repeat the exact "manufacture scope" failure mode this section exists to
avoid. Recommended as a **future Ceremony 1's own scoping item**, not
guessed at here. The waitlist `Position` count-vs-monotonic question
(T6.6-loop-2) is explicitly a **product**-semantics question by its own
disclosure text, not an engineering one — it joins #126/#130's blocker
class, not this ceremony's take list.

## §A6 — T19.1 and T19.2: dependency-completeness check, both questions, run
against the code

**T19.1 — does the producer's capability exist?** **Yes, entirely — the
unusual shape here, stated explicitly rather than reflexively re-run as if
it were the usual "build a new port" case.** `domain.Game.Status` and
`StatusCancelled` already exist (`game.go:11`); `domain.ErrGameCancelled`
already exists and is already mapped to `FailedPrecondition` in
`adapter/grpcapi.toStatus` (§A3). Nothing new needs to be built at the
producer end — the gap is that `Register`/`JoinWaitlist` never call the
check that would use what already exists. **Does the consumer's own input
contain what it needs?** N/A in the cross-context sense (this is
single-context, Social-Play-only); the consumer here is `domain.Register`/
`JoinWaitlist` themselves, and both already receive a full `Game` value
(including `Status`) as their first parameter — confirmed by reading both
signatures (§A3) — so no signature change is needed anywhere in the call
chain, only a new check inside two function bodies plus the DB-level
mirror.

**T19.2 — does the producer's capability exist?** **Yes, in full.**
`paymentsapp.Service.RecordOfflinePayment`, `paymentspg.Repository`, and the
`testcontainers-go` + `applyMigrations` scaffolding this ticket needs all
already exist and are already exercised by
`smoke_integration_test.go` in the same package — this ticket adds a new
test function to an already-proven harness, mirroring
`internal/booking/adapter/postgres/concurrency_integration_test.go`'s and
`internal/socialplay/adapter/postgres/capacity_concurrency_integration_test.go`'s
pattern exactly. No new port, adapter, or production code. **Consumer
input**: N/A, single-context, test-only.

**Neither ticket touches D1 or D2's surface** — confirmed directly: T19.1
touches `internal/socialplay/{domain,app,adapter/postgres-migration}`, T19.2
touches `internal/payments/adapter/postgres` (test file only); neither
touches `internal/booking/**`, `CancelBooking`, `CreateBooking`, or any
reviewer-authorship question. Grepped to confirm: zero occurrences of
`CancelBooking`/`CreateBooking` in either ticket's target file list.

## §A7 — DECISION D1 and DECISION D2 remain unanswered

**Re-verified this ceremony, not assumed** (§A3): #144 carries exactly one
comment, T14.3's original escalation, unchanged across T14 through T18, now
T19. ADR-0016's own `## Status` field is unchanged: "Escalated — awaiting
the user's decision. This ADR decides nothing."

> **DECISION D1 (for the user / Product Owner) — seventh deferral.** When
> somebody books a court through the public flow *without an account* —
> which is how the flow works today — who should be allowed to cancel that
> booking later, and should booking without an account remain possible at
> all? `docs/adr/0015-booking-ownership-for-public-bookings.md` lays out
> four options with their costs and recommends none. No T19 ticket
> implements this, and per ADR-0015's restriction list no T19 PR may guess
> at it. Its footprint is unchanged from T18 retro's own re-check: still the
> same two named instances of shaped scope (`CancelBooking`'s own missing
> check; the court-Bookings half of #124), neither grown nor shrunk, because
> no T19 ticket touches either surface.

> **DECISION D2 (for the user) — fourth deferral, fifth sprint open.** When
> the same session both reviews and merges a pull request, may it also
> write the code on that pull request — and if so, under exactly what
> limits? `docs/adr/0016-reviewer-authored-code-on-a-reviewed-pull-request.md`
> lays out four options and recommends none. T18 shipped no
> reviewer-authored gap-fix (its own retro finding 4 confirmed T18's own
> Ceremony 1 prediction correct) — its fourth consecutive sprint with
> nothing to score either way. T19's two tickets are both implemented and
> reviewed by the ordinary two-role loop; nothing in either ticket's design
> calls for a reviewer to author code on a PR under its own review, so this
> sprint is not expected to produce a fifth data point either — stated as a
> prediction, not a guarantee, and the retro is asked to score it
> regardless of which way it lands.

**Neither is implemented, decided, or guessed at by this ceremony.** Both
remain exactly as blocked as `sprint-process.md`'s own restriction lists
require.

## §A8 — Shared-file pre-assignment, and same-wave verification

| Artifact | Owner | Notes |
|---|---|---|
| `internal/socialplay/domain/registration.go`, `waitlist.go`, `errors.go` (doc-comment only — no new sentinel) | **T19.1 only** | No other T19 ticket touches Social Play |
| `internal/socialplay/app/service.go` (no functional change expected — `RegisterForGame`/`JoinWaitlist` already propagate `domain` errors unchanged; touched only if instruction 3 finds otherwise) | **T19.1 only** | Same |
| `db/migrations/0023_socialplay_registration_status_guard.sql` | **T19.1 only** | Next free number, confirmed (§A3/§A4); pre-assigned so no other future-dispatched ticket can claim it out from under this one |
| `internal/payments/adapter/postgres/` (new test file only) | **T19.2 only** | No other T19 ticket touches Payments |
| **`HANDOFF.md`** | **this ceremony only** | An implementer that finds a stale line flags it for T20's Ceremony 1 and does not edit it — the standing rule, unchanged |
| **`docs/process/sprint-process.md`** | **this ceremony only** | No T19 execution ticket touches process; no amendment is landed this ceremony |

**Same-wave shared-interface verification rule: does not apply, verified
rather than assumed.** T19.1 touches `internal/socialplay/**` exclusively;
T19.2 touches `internal/payments/adapter/postgres/**` exclusively (a new
test file, not an interface change at all). Fully disjoint packages, no
shared interface either ticket widens or implements — the rule's own
precondition (two same-wave tickets touching **one shared interface's**
blast radius) has no instance here, checked by reading both tickets' target
file lists rather than inferred from ticket count alone (T18's own honest
distinction between "no sibling exists" and "siblings exist but are
file-disjoint" — this sprint is the latter).

---

# Ceremony 2 — Sprint planning

## Sprint goal

> **Two real, disclosed, genuinely unblocked gaps that had never been
> tracked as GitHub issues get built and closed — not because the tracked
> backlog offered anything actionable (it did not: all 8 open issues remain
> exactly as blocked as T18 left them, independently re-verified), but
> because a genuine re-verification of `HANDOFF.md`'s own disclosed-gap
> record found two items whose own stated closing conditions had already
> been met or were never conditional at all.** A Player can no longer
> register for, or join the waitlist of, a Game that is already cancelled
> (#212) — closing a gap disclosed at T5.2 whose own stated trigger fired at
> T16.3 with nobody acting on it. Payments' concurrent-duplicate-recording
> invariant, already correctly enforced by Postgres today, gets the
> committed regression proof it has been missing since T6.4 (#213), mirroring
> booking's and Social Play's own committed concurrency-test pattern. Every
> other open issue stays exactly where T18 left it, each for a restated,
> re-verified reason rather than a silent re-deferral. D1 and D2 go back to
> the user unanswered.

**What this sprint does not claim** (the half PM insists on):

- **This is not a bigger sprint manufactured to avoid an 8-point precedent.**
  Both tickets were found by re-verifying a disclosed-gap record that
  already existed, not by inventing new scope — the alternative genuinely
  considered and rejected was a 0-ticket sprint (§A5), and PM/PdE/PE agreed
  that filing and closing real, already-disclosed, unblocked gaps is the
  more honest use of a sprint than either manufacturing a fabricated ticket
  against a blocked issue or doing nothing while two real gaps sit unfiled.
- **Neither ticket resolves, narrows, or touches D1 or D2** — confirmed
  directly (§A6).
- **T19.1 does not change what happens to an *already-active* Registration
  when its Game is later cancelled** — that is T16.3's cascade, already
  shipped. T19.1 only stops a *new* Registration/waitlist join from being
  created against a Game that is *already* cancelled.
- **T19.2 does not change the invariant it proves.** The DB-level unique
  index already correctly enforces "exactly one Payment per payable" today;
  this ticket adds the committed regression proof that invariant has been
  missing, it does not alter payment-recording behaviour in any way.
- **#144/#149/#124's remainder/#164/#145/#126/#130/#134 are untouched**,
  each with a reason recorded in §A5.
- **This sprint (2 tickets, 8 points) is the same size as T18's, not
  bigger** — not a loosening of scope discipline: every open tracked issue
  remains genuinely blocked (§A5), and manufacturing scope to hit a bigger
  number would mean guessing at one of those blockers.

## Tickets — 2 items, 8 points

### T19.1 — `Register`/`JoinWaitlist` reject a cancelled Game, at both domain and DB layers (closes #212)

- **Story:** As a Player, I want the platform to refuse my attempt to
  register for, or join the waitlist of, a Game that has already been
  cancelled, so that I never end up holding a Registration or waitlist slot
  in something that no longer exists.
- **Points:** 5 · **Role:** `role:principal-engineer` · **Type:** `type:bug`
- **Description:** Closes **#212** (filed this ceremony, §A3/§A5 — a gap
  disclosed at T5.2 whose own stated closing trigger, "when Game-cancellation
  cascading is built," fired at T16.3 with nobody acting on it since). The
  reused sentinel (`domain.ErrGameCancelled`) and its gRPC mapping already
  exist (§A3/§A6) — this ticket wires the check that was never built, not a
  new error concept.
- **DoD-claim discipline (T18 retro recommendation 1, applied):** this
  ticket does **not** claim `RegisterForGame`/`JoinWaitlist`'s existing
  behaviour is otherwise unchanged in the strong "byte-for-byte" sense —
  say instead, up front: **behaviourally additive only** (a new, earlier
  rejection branch; every currently-passing case that doesn't hit a
  cancelled Game is untouched, and that is the claim instruction 6 asks be
  verified, not a stronger one).

**Instructions**

1. **Add the check to `domain.Register`** (`internal/socialplay/domain/
   registration.go`): if `game.Status == StatusCancelled`, return
   `Registration{}, ErrGameCancelled` — checked **first**, before the
   guest-allowance/double-registration/capacity checks (the game not being
   in a bookable state at all is the more fundamental fact, mirroring
   `Register`'s own existing ordering rationale for its other checks).
2. **Add the identical check to `domain.JoinWaitlist`** (`internal/
   socialplay/domain/waitlist.go`), same ordering rationale, same sentinel.
3. **Verify `app.Service.RegisterForGame`/`JoinWaitlist` propagate the new
   error unchanged** — both already call `domain.Register`/`JoinWaitlist`
   and return their error directly (confirmed by reading both, §A3); if
   either wraps or swallows the domain error, fix it to propagate bare, the
   same discipline `adapter/grpcapi.toStatus`'s existing `errors.Is`-based
   dispatch already assumes throughout this codebase.
4. **Add migration `0023_socialplay_registration_status_guard.sql`**
   (confirmed the next free number, §A3/§A4), with a header comment in the
   same form `0006_socialplay_capacity_guard.sql`'s own header uses — naming
   the owning context (Social Play), this ticket (T19.1, closes #212), and
   explicitly stating this is a `CREATE OR REPLACE FUNCTION` **redefinition**
   of `enforce_game_capacity()` and `join_waitlist_entry()`, not an edit to
   the already-applied `0006`/`0009` files (CLAUDE.md's migration-append-only
   gotcha). Both functions already lock the owning `games` row `FOR UPDATE`
   before their existing capacity/position logic runs (§A3) — add, at the
   top of each locked section, a check that raises (mirroring
   `enforce_game_capacity`'s own `RAISE EXCEPTION ... USING ERRCODE =
   'P0001'` pattern) when the locked row's `status = 'cancelled'`. This is
   the dual-invariant pattern (`docs/adr/0001-dual-invariant-enforcement.md`)
   applied to a *third* Social Play invariant using the *same* lock two
   existing functions already take — no new lock, no new race window.
5. **Non-functional — TDD-first, mutation-checked, at all three layers this
   codebase's established object-level-check pattern uses (T5.5/T7.7/T8.5):**
   - Domain-level table-driven tests on `Register`/`JoinWaitlist` asserting
     `ErrGameCancelled` for a cancelled Game and unchanged behaviour for
     every other existing case (capacity, guest allowance, double
     registration) — run the **existing** test suites for both functions
     unmodified alongside the new cases, proving instruction set 1–2 is
     additive per this ticket's own DoD-claim discipline above.
   - A regression test at the `app.Service` level covering the same two
     call sites.
   - Verified non-vacuously (CLAUDE.md rule 10): temporarily disable both
     new checks, confirm the new tests fail (and only the new tests — the
     pre-existing suites for both functions must still pass, proving the
     new checks are additive, not a behaviour change to the untouched
     paths), restore, confirm the full `internal/socialplay/...` suite green
     again.
   - **The DB-level guard is not independently executable here** (no Docker
     daemon, the standing gap every prior sprint's tickets have carried) —
     state this plainly rather than claiming a run that didn't happen; the
     migration's own header names the exact manual-verification fallback
     this project's LESSONS.md methodology already established (apply
     `0001`–`0023` in order against a local Postgres instance, attempt a
     `RegisterForGame`/`JoinWaitlist` call against a cancelled Game's row
     directly via the repository, confirm the trigger raises).
6. **Close #212 after merge**, per DoD step 5, with a comment naming this
   PR and stating plainly what the fix covers and what it deliberately does
   not (per the sprint goal's "does not claim" list): new registrations/
   waitlist joins against an already-cancelled Game are now rejected at both
   layers; an already-active Registration on a Game that gets cancelled
   *after* the fact is unaffected by this ticket (that's T16.3's cascade,
   already shipped).

### T19.2 — Commit Payments' 20-way concurrent-duplicate-recording proof (closes #213)

- **Story:** As a Principal Engineer reviewing this codebase's concurrency
  claims, I want the "exactly one Payment per payable, even under
  concurrent contention" invariant proven by a committed, repeatable test —
  the same standard booking's and Social Play's own concurrency invariants
  already meet — rather than trusting a claim that was only ever manually
  verified once, years ago, via a script nobody can re-run.
- **Points:** 3 · **Role:** `role:qa` · **Type:** `type:chore`
- **Description:** Closes **#213** (filed this ceremony, §A3/§A5 — a gap
  disclosed at T6.4, carried in `HANDOFF.md`'s prose for 13 sprints without
  a committed test or a tracked issue). **This ticket changes no production
  code** — the invariant is already correctly enforced by
  `payments_payable_unique_idx` (§A3); only the committed regression proof
  is missing.

**Instructions**

1. **Add a new test file** in `internal/payments/adapter/postgres/`
   (e.g. `concurrency_integration_test.go`, mirroring
   `internal/booking/adapter/postgres/concurrency_integration_test.go`'s
   filename convention exactly), `//go:build integration`, following the
   same testcontainers-go + `applyMigrations` scaffolding
   `smoke_integration_test.go` already uses in this same package (reuse
   its `waitForReady`/`applyMigrations` helpers rather than duplicating
   them, if Go's package-private visibility allows — otherwise a minimal,
   explicitly-justified duplication, same package).
2. **Fire N (20, matching T6.4's original claim) concurrent
   `RecordOfflinePayment` calls** via `paymentsapp.Service`, all targeting
   the same `(payable_type, payable_id)` pair. Assert **exactly 1** success
   (`Status == domain.StatusPaid`) and **exactly N−1**
   `domain.ErrPaymentAlreadyRecorded` — not "at least one success," the
   precise count, the same discipline `concurrency_integration_test.go`'s
   own assertion already uses for the booking invariant.
3. **Non-functional — CLAUDE.md rule 10 applied directly, since this
   ticket's entire subject is a concurrency claim:** run the new test
   **repeatedly**, not once — at minimum 3 runs, including at least one
   true process cold start (container recreated, not reused), before
   calling the invariant "proven" anywhere in the PR body. State the exact
   run count and outcomes in the PR, the same way T4's own concurrency work
   and its `docs/LESSONS.md` entry set the bar for this project.
4. **Docker-daemon caveat, stated plainly rather than silently assumed
   away:** this environment has no Docker daemon (the standing gap every
   prior sprint's committed `-tags=integration` tests have carried, per
   CLAUDE.md's own "11 test files... gated behind `//go:build integration`"
   gotcha). This ticket cannot itself execute the committed test here. Per
   this project's own T4/T6.4 LESSONS.md fallback methodology, verify the
   identical scenario manually against a local Postgres instance instead
   (apply `0001`–`0023` in order, run the same 20-concurrent-call scenario
   via a throwaway program, confirm exactly 1 success and 19 conflicts,
   repeated across multiple runs including a cold start), and state that
   fallback explicitly in the PR rather than claiming the committed test
   itself was run. `make vet-integration` (compiles, does not run, no
   Docker needed) must still pass — run it directly, not assumed.
5. **Close #213 after merge**, per DoD step 5, with a comment naming this
   PR, the exact run count and outcomes (per instruction 3), and stating
   plainly that this ticket changed no production code — only added the
   committed proof of an invariant that was already correctly enforced.

## Waves

**Wave 1 — both tickets, dispatched together (2 tickets, 8 points)**
`T19.1` (#212), `T19.2` (#213)

Confirmed file-disjoint (§A8) — `internal/socialplay/**` vs.
`internal/payments/adapter/postgres/**` (a new test file, no interface
touched) — with independent producer capabilities already existing for both
(§A6). No functional dependency between them in either direction. No
Wave-1.5 checkpoint condition (a new cross-cutting decision with three or
more first-time in-sprint consumers) can fire: neither ticket introduces a
new cross-cutting interface at all.

## Recorded disagreements (Ceremony 2 rule 3 — not smoothed over)

**None this sprint, stated explicitly rather than left to be inferred from
silence.** Both tickets close a disclosed, previously-unfiled gap with an
already-agreed suggested shape (from their own `HANDOFF.md` disclosure
text, now carried into the filed issue and this ticket's instructions);
PM's residual-concern pattern from T18 (a small sprint's value framing) was
raised and did not recur here, because — unlike T18.1, which PM
characterized as reliability-hardening rather than new user-facing
capability — T19.1 closes a **live, unguarded correctness gap** (a Player
can today register for a cancelled Game with no error anywhere in the
stack), which every role agreed is squarely in scope regardless of sprint
size. No PE/PM/PdE/QA/PO/BA disagreement was recorded on either ticket's
inclusion, scope, or sizing.

## Sprint-level Definition of Done

All of `sprint-process.md`'s standing DoD, plus the scorings T19 owes,
stated now so they are not improvised at the retro:

1. T19.1 and T19.2 merged per the per-ticket DoD; sprint goal met or
   explicitly descoped with reasoning recorded.
2. **The merged-fix issue sweep run and reported with its count** — by the
   retro (reporting, not blocking) and again by T20's Ceremony 1
   (authoritative).
3. **Scoring owed at the retro:**
   - **(a)** Did T19.1 actually close #212, scored directly against the
     merged code (does a `RegisterForGame`/`JoinWaitlist` call against a
     cancelled Game actually get rejected, at both the domain and DB
     layers, without any new production-code regression on the untouched
     cases), not against the PR's own account?
   - **(b)** Is T19.1's own behaviourally-additive-only claim (the DoD-claim
     discipline stated in the ticket text itself, per T18 retro
     recommendation 1) actually true — did every pre-existing
     `Register`/`JoinWaitlist` test case keep passing unmodified?
   - **(c)** Did T19.2's new concurrency test actually get run repeatedly,
     including a cold start, with the exact counts stated in the PR — not
     merely asserted once?
   - **(d)** Did the migration-header-ownership check (§A4, T17 retro
     recommendation 1) get applied correctly for T19.1's migration — does
     `0023_socialplay_registration_status_guard.sql`'s own header let a
     future ceremony read both the ownership answer and the
     redefinition-not-edit distinction in one read?
4. **Not scoreable by T19 and deliberately not pre-empted:** D1 and D2
   remain the user's. If either is answered mid-sprint, the answer's own
   trigger takes over and T19's plan does not constrain it.
5. Retro in `docs/process/t19-retro.md`, indexed by a `## T19 sprint retro`
   stub in `docs/LESSONS.md`. `HANDOFF.md`/`CLAUDE.md` state updated —
   noting that **T20's Ceremony 1**, not the retro, corrects T19's
   Docs-index row (the ordinary convention).
