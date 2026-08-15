# T20 Sprint Plan — Ceremonies 1 + 2

Backlog refinement (Ceremony 1) and sprint planning (Ceremony 2) per
`docs/process/sprint-process.md`, six-role team (briefs:
`docs/agent-operating-handbook.md` Part B). Held against
`docs/process/t19-retro.md` (PR #217, merged 2026-08-15T19:36:40Z),
`HANDOFF.md` including its Docs index, `CLAUDE.md`, and the live PR/issue
state of `nhuthuynh/white-label` (GitHub-side name `pickleball-platform`).

**Every factual claim below was re-derived against the repository and the
GitHub API during this ceremony**, at the merged branch tip (`ff7233a`, T19
retro's own merge), rather than taken from the retro's or T19's plan's
prose — CLAUDE.md rule 10 applied to planning. Where this ceremony repeats a
claim it could not independently re-check, it says so.

## §A0 — Correcting T19's Docs-index row and Task-backlog narrative: this
ceremony's first job, per the retro's own instruction

Per `sprint-process.md`'s standing convention (a retro PR never updates the
Docs-index row that points at it, since that row must cite the retro's own
merge PR number, which does not exist until it merges) and per
`docs/process/t19-retro.md`'s own closing line — *"T20's Ceremony 1 corrects
T19's row, including its real PR merge order and the honest-form sentence
above, as its first job"* — **performed directly in `HANDOFF.md` by this
same PR**, not summarized twice here. What was corrected, and how each fact
was independently re-verified rather than copied from the retro's prose:

| Field | What the row now says | Independently re-verified this ceremony |
|---|---|---|
| Retro link | `docs/process/t19-retro.md` (no incident-grade finding; two mutation checks and a fourth, independently-authored concurrency reproduction re-performed against the merged tree; names T19.2's status precisely as "manually proven, CI-unexecuted"; 5 recommendations for T20) | File exists, read in full this ceremony; its own characterization matches its content |
| Reviews cell | PRs #214 (Ceremony 1/2 doc) → #215 (T19.2) → #216 (T19.1) → #217 (retro doc), in that merge order | Re-fetched all four via `list_pull_requests`: `merged_at` **19:16:06Z → 19:24:28Z → 19:26:36Z → 19:36:40Z** — ascending; T19.2 (#215) merged before T19.1 (#216) despite the opposite *dispatch* order, checked against the live field rather than presumed from the plan's own dispatch table |
| Task-backlog narrative | The retro's own agreed sentence, quoted verbatim from `docs/process/t19-retro.md`'s "The sprint goal, scored" section, per `sprint-process.md` Ceremony 1 item 3 ("the retro's form, not a stronger one") | Cross-checked word-for-word against the retro's own blockquote — copied, not paraphrased |

**A new T20 row is added** in the same before-the-sprint form, Retro and
Reviews honestly marked "not yet written" / "not yet opened", for T21's
Ceremony 1 to correct in turn — the same convention every prior ceremony has
followed.

## §A1 — The merged-fix issue sweep, run as this ceremony's first substantive
act (per DoD, T20's Ceremony 1 is the authoritative moment)

**Step 1 — list the open issues, live.** `list_issues(state: OPEN)` at
ceremony start → **`totalCount: 8`**: #124, #126, #130, #134, #144, #145,
#149, #164. Matches `docs/process/t19-retro.md`'s own sweep exactly —
re-fetched live rather than trusted, per the standing rule that a prior
ceremony's clean result does not discharge the next one.

**Step 2 — reconcile arithmetically, before reading any issue
individually.** `open_at_end_of_T19's_own_retro − closed_since_then +
opened_since_then`. T19's own retro (its sweep moment 1,
`docs/process/t19-retro.md` finding 1) left the count at **8**. Since that
retro's own PR (#217) merged, zero PRs have merged (confirmed by `git log`,
below) — `closed_since = 0`, `opened_since = 0`. `8 − 0 + 0 = 8`. **Matches
the live `totalCount: 8` read at this ceremony's start exactly.**

**Step 3 — cross-reference merged PRs against the open list.** Only one PR
has merged since T19's retro closed the tree: #217 itself (the retro's own
doc-only PR, which its own closing line states performed no tracker action —
*"Issue-tracker actions this ceremony: none... Open count now: 8"*).
Re-verified directly rather than assumed: `git log --oneline -3` shows
`ff7233a` (PR #217's merge) as the current tip with no descendants, and
`git status` is clean, before this ceremony's branch was cut.

**Sweep result: clean, sixth sprint running** (after T15, T16, T17, T18,
T19). **T21's Ceremony 1 still re-runs this sweep in full**, per the
standing rule that a prior ceremony's clean result does not discharge the
next one.

**Recommendation 1's shape does not recur this sweep, stated explicitly.**
T19 retro's recommendation 1 covers the case where a sprint's own Ceremony 1
files an issue that the same sprint's execution then closes — this
ceremony's own §A5/§A6 below conclude that **no new issue is filed this
ceremony**, so the arithmetic above needed no special handling for that
shape. Recommendation 1 remains live guidance for whichever future ceremony
next hits it, not something this sweep had to apply.

## §A2 — T19 retro's five recommendations, each given a disposition

| # | Recommendation | Disposition |
|---|---|---|
| 1 | When a sprint's own Ceremony 1 files an issue that the same sprint's execution closes, count the Ceremony-1 filing as part of `opened_during_sprint` | **Not exercised this sprint, by construction** — §A5/§A6 conclude no new issue is filed. Carried forward as standing guidance for whenever the shape recurs, not restated as a new rule (this project's own "don't accumulate two shapes of one fix" discipline, `sprint-process.md`'s Scheduled-removals table) |
| 2 | Name a manually-verified-but-CI-unexecuted concurrency claim precisely ("manually proven, CI-unexecuted"), never rounded either direction | **Not exercised this sprint** — no T20 ticket touches a concurrency claim (zero tickets, §A6). Carried forward: the next ticket that does touch a `-tags=integration` concurrency claim applies this naming discipline |
| 3 | Continue treating the merged-fix sweep as authoritative regardless of the prior retro's clean result | **Executed here** (§A1) — independently re-derived the open count and re-verified from the API rather than the retro's own table |
| 4 | D1 and D2 stay with the user; no T20 ticket implements `CancelBooking`/`CreateBooking` authorization or a reviewer-authorship carve-out | **Executed here** (§A7) — neither implemented nor guessed at; moot in the strongest possible sense this sprint, since zero tickets exist to check |
| 5 | A future retro should check a Ceremony 1's file-scope prediction against the actual diff, file by file | **Not a Ceremony-1 action item this sprint, and cannot be — there is no ticket to make a prediction about** (§A6). Binds whichever future ceremony next writes a file-scope prediction; noted here so it is not silently dropped for having had nothing to attach to this time |

## §A3 — What this ceremony verified, and how

| Claim | How it was checked | Result |
|---|---|---|
| Open-issue count at ceremony start | live `list_issues(state: OPEN)` | `totalCount: 8`, matching #124/#126/#130/#134/#144/#145/#149/#164 exactly |
| Every one of the 8 open issues' blockers are unchanged since T19's retro | `list_issues` with labels/comment counts, live; cross-checked against `docs/process/t19-sprint-plan.md` §A3/§A5's own per-issue table | #144 still exactly 1 comment (D1, unchanged); #149 still 3 comments (unchanged); #145/#164 still 1 comment each (unchanged); #124 still 1 comment (unchanged); #126/#130/#134 still 0 comments (unchanged). **Zero blockers changed state, zero new comments on any of the 8.** |
| D1 (#144) still has exactly one comment | `issue_read`-equivalent live fetch | confirmed — T14.3's original escalation only, unchanged across T14 through T19, now T20 (eighth sprint) |
| D2 (ADR-0016) still unanswered | read `docs/adr/0016-*.md`'s `## Status` | unchanged: "Escalated — awaiting decision (D2)." |
| No PR merged since T19's retro's own PR | `git log --oneline -3`, `git status` | clean, up to date with `origin/claude/go-backend-pickleball-7up34j`, HEAD at `ff7233a` (T19 retro's own merge) before this ceremony's branch was cut |
| `HANDOFF.md`'s Cross-cutting section re-read in full for any new disclosed-but-never-filed gap, the same check T19's own Ceremony 1 ran | Read the entire Cross-cutting section (`HANDOFF.md` lines ~936–1600) line by line | **No new candidate of #212/#213's shape found.** Every open thread in that section is one of: already closed (with a closing note in place — ServiceOptions/#123, CreateUser squatting/T12.9, RefundPayment wiring/T12.3, CancelGame authz/T12.4, panic recovery/#89+T10.7, Competitions PayableType/#96, facility_id reconciliation/T8.3, courts-list/T8.2, camera consent/T8.4, npm peer-deps/#79); already tracked as one of the 8 open issues (#126 price field, #130 no-show refund, #134 WCAG); explicitly and structurally blocked on something outside this environment (Jenkins server-side wiring — no reachable instance/admin credentials, same class as the IdP-tenant blocker on #164/#145); or a deliberate, named, low-urgency roadmap item, one level down |
| The one candidate that looked promising on a first pass — the `golang-migrate`/`goose` migration-tooling swap (`HANDOFF.md`'s "Swap docker initdb.d for golang-migrate or goose before production") — checked against this project's own settled precedent, not treated as new | `grep -rn "golang-migrate\|goose" HANDOFF.md docs/process/*.md`, then read the T11 (line ~893), T12 (~1315), T13 (~1336), T14 (~1416) sprint plans' own Cross-cutting dispositions for it in full | **Four separate prior ceremonies (T11, T12, T13, T14) already considered this exact item and explicitly ruled it out of the issue-filing obligation — on the record, in the same words each time**: T13's plan states outright, "**No issues opened: these are roadmap items, not disclosed gaps**," grouping it with Observability and the ISO-8601 weekday encoding note. This is a **different classification from #212/#213**, which were live, unguarded correctness/coverage gaps with a concrete, bounded fix — not open-ended infra debt with no fixed scope (which tool, what migration path for the already-applied `0001`–`0023` files, whether the `0005` payments/socialplay collision gets renumbered). T19's plan's own aside recommending "a future Ceremony 1's own scoping item" was PdE/BA's forward-looking suggestion in a "why not bigger" footnote, not a reopening of four sprints' settled classification — and reopening a settled classification on the strength of one footnote, without a materially new fact, is exactly the kind of scope-manufacturing this section exists to avoid. **Disposition: still correctly unticketed, unchanged.** |
| `go vet ./internal/socialplay/... ./internal/payments/...` — checking whether T7.7's old disclosed `go vet` gap (a stale 3-arg `socialplayapp.NewService` call in `registration_updater_test.go`) is still real | ran directly; also read `internal/socialplay/app/service.go`'s `NewService` signature and all `socialplayapp.NewService` call sites in `internal/payments/adapter/socialplay/` | **Moot, not live** — `NewService` has taken a `ServiceOptions` struct since T13.8, and every call site in `internal/payments/adapter/socialplay/` already uses the struct form; `go vet` on both packages is clean. This T7-era note is stale from an intervening refactor, not a currently-real gap; no action needed and none taken |
| `make generate && make test-domain && make gate-coverage` still green on the unmodified tree | ran directly | `go vet ./...`: clean; `make test-domain`: all 12 packages green; `make gate-coverage`: `OK — all 42 package(s) … executed by "ci-checks"` |
| Next free migration / ADR numbers, checked in case a ticket needed one | `ls db/migrations`, `ls docs/adr` | `0023` last migration, `0016` last ADR → `0024`/`0017` would be free — **not consumed this sprint**, no ticket needs one |
| Local worktree matches the shared branch's real tip | `git status`, `git log --oneline -3` | clean, up to date with `origin/claude/go-backend-pickleball-7up34j`, HEAD at `ff7233a` before this ceremony's branch was cut |

**Not re-verified by this ceremony, and named as such:** the full `make
test`/`ci-integration` Docker path (no Docker daemon reachable here, same
standing gap as every prior sprint); any claim about what Jenkins would run
(no Jenkins job exists).

---

# Ceremony 1 — Backlog refinement

## §A4 — Migration-header-ownership check: not applicable this sprint

No T20 ticket exists (§A6), so no ticket names a database table or a
migration file. Stated explicitly rather than silently omitted, per this
project's own "silence is indistinguishable from not having checked"
standard — the same treatment T18's plan gave the same-wave verification
rule when it had only one ticket, generalized here to zero.

## §A5 — The whole open backlog, ranked, with a disposition for each

All 8 open GitHub issues, re-verified live (§A3):

| Issue | Ranked | Disposition |
|---|---|---|
| **#144** `CancelBooking`/`CreateBooking` have no authz | **Escalated** | D1 unanswered; §A7. Eighth sprint carrying this |
| **#149** Payments' remaining caller-supplied fact (`booking_host_id`) | **Untouched, correctly** | Blocked on D1 exactly as its own text says |
| **#124** court-Bookings half of the cascade | **Untouched, correctly** | Blocked on D1, same reason as #149; re-verified still open with its T16.3 comment intact |
| **#164** ADR-0014 actor-column conformance | **Deferred, still blocked** | Needs a real IdP tenant, unreachable from this environment; unchanged since T14 |
| **#145** pre-existing UUID rows vs. `Principal.Subject` | **Deferred** | Same real-IdP-tenant blocker as #164; unchanged |
| **#126** real per-Game price field | **Deferred** | Needs Product Owner input on whether the price is per-Game, per-Registration, or per-head; unchanged since T8.10 |
| **#130** refunding a `no_show_fee` Payment | **Deferred** | Carries its own stated open product question ("confirm first that reversing a no-show fee is the product behaviour actually wanted"); unchanged |
| **#134** WCAG manual screen-reader pass | **Deferred** | Needs a `role:ux-ui-designer` pass with real assistive-technology hardware this environment cannot provide; T20 ships no UI change |

**Every one of the 8 tracked issues remains genuinely blocked, independently
re-verified rather than assumed** (§A3) — the ninth consecutive time this
exact set of blockers has been re-checked and found unchanged (T12 through
T20). Zero of the 8 open issues are actionable this sprint without guessing
at D1, a real IdP tenant, a Product Owner decision, or assistive-tech
hardware.

**Unlike T19, this ceremony's own re-scan of `HANDOFF.md`'s Cross-cutting
section (§A3) found nothing new to promote into an issue.** T19's own
Ceremony 1 found two items that had been miscategorized for over a decade
of sprints — genuinely live, unguarded correctness/coverage gaps sitting in
prose rather than in the tracker, which the board-of-record rule's
"mandatory, not discretionary" language required be filed. This ceremony
ran the identical check and did not find a third: every remaining
Cross-cutting thread is either already closed, already tracked as one of
the 8 issues above, blocked on something structurally outside this
environment, or — checked specifically, §A3 — a roadmap-debt item four
prior ceremonies (T11–T14) already and explicitly classified as **not** a
disclosed gap requiring the mandatory-issue treatment, a classification
this ceremony found no new fact to overturn.

## §A6 — Why zero tickets, and why that is the honest answer rather than a
gap in this ceremony's diligence

**The two alternatives this project has repeatedly named and rejected are
still rejected, for the same reasons:**

1. **Manufacture a ticket against one of the 8 blocked issues by guessing
   at its blocker.** Not done — every blocker was independently
   re-verified live this ceremony (§A3/§A5), and none has moved.
2. **Take the migration-tooling item (or Observability, or the ISO-8601
   weekday-encoding note) as new scope.** Not done, for the reason §A3's
   table states in full: this is settled, on-the-record roadmap debt, not
   a disclosed gap, per four separate prior ceremonies' own explicit
   ruling — reopening that classification on the strength of one sprint's
   footnote, absent a new fact, would be exactly the kind of manufactured
   scope this project's "take only what's genuinely unblocked" discipline
   (established T16, reaffirmed every sprint since) exists to prevent.

**The third alternative — the one this ceremony takes — is a 0-ticket
sprint, stated with its reasoning in full rather than left implicit.** This
is not new to this project: `docs/process/t19-sprint-plan.md` §A5 named a
0-ticket sprint as the genuine alternative it considered and rejected only
because #212/#213 existed to take instead; the task that set up this
ceremony names it explicitly as an acceptable, even preferable, outcome
over fabricated scope. **No new #212/#213-shaped gap surfaced this time.**
A sprint that manufactures a ticket to avoid reporting zero is a worse
outcome than a sprint that reports zero honestly — the same "an invariant
with no test that could fail is untested, not proven" standard QA applies
to code, applied here to sprint scope itself.

**Dependency-completeness check: not applicable.** With no ticket, there is
no producer/consumer pair to check either question of (`sprint-process.md`'s
two-question form) — stated explicitly per this project's own
silence-is-not-diligence standard, not left to be inferred from the
section's absence.

## §A7 — DECISION D1 and DECISION D2 remain unanswered

**Re-verified this ceremony, not assumed** (§A3): #144 carries exactly one
comment, T14.3's original escalation, unchanged across T14 through T19, now
T20. ADR-0016's own `## Status` field is unchanged: "Escalated — awaiting
decision (D2)."

> **DECISION D1 (for the user / Product Owner) — eighth deferral.** When
> somebody books a court through the public flow *without an account* —
> which is how the flow works today — who should be allowed to cancel that
> booking later, and should booking without an account remain possible at
> all? `docs/adr/0015-booking-ownership-for-public-bookings.md` lays out
> four options with their costs and recommends none. No T20 ticket exists
> to implement this (there are no T20 tickets at all), and per ADR-0015's
> restriction list no T20 PR may guess at it regardless. Its footprint is
> unchanged from T19 retro's own re-check: still the same two named
> instances of shaped scope (`CancelBooking`'s own missing check; the
> court-Bookings half of #124), neither grown nor shrunk.

> **DECISION D2 (for the user) — fifth deferral, sixth sprint open.** When
> the same session both reviews and merges a pull request, may it also
> write the code on that pull request — and if so, under exactly what
> limits? `docs/adr/0016-reviewer-authored-code-on-a-reviewed-pull-request.md`
> lays out four options and recommends none. T19 shipped no
> reviewer-authored gap-fix (its own retro finding 6 confirmed T19's own
> Ceremony 1 prediction correct) — its fifth consecutive sprint with
> nothing to score either way. **T20 has no tickets at all**, so there is
> no PR under review this sprint in the first place — the strongest
> possible null result, not merely a sixth prediction that happens to hold.
> The retro is asked to record this precisely (no PR existed to score, not
> "a PR existed and produced no reviewer-authored fix") rather than folding
> it into the same sentence as T15–T19's five null results, which is a
> different and weaker claim.

**Neither is implemented, decided, or guessed at by this ceremony.** Both
remain exactly as blocked as `sprint-process.md`'s own restriction lists
require.

## §A8 — Shared-file pre-assignment, and same-wave verification: not
applicable this sprint

| Artifact | Owner | Notes |
|---|---|---|
| **`HANDOFF.md`** | **this ceremony only** | An implementer that finds a stale line flags it for T21's Ceremony 1 and does not edit it — the standing rule, unchanged. No execution ticket exists this sprint to collide with it |
| **`docs/process/sprint-process.md`** | **not touched this ceremony** | No amendment is landed this ceremony |

**Same-wave shared-interface verification rule: does not apply — trivially,
by construction.** The rule's own precondition is *two* same-wave tickets
touching one shared interface's blast radius. T20 has zero tickets. There is
no wave to have a collision within, the same honest "no opportunity to
fire" answer T18's plan gave for its own single-ticket sprint, one level
further down.

---

# Ceremony 2 — Sprint planning

## Sprint goal

> **Confirm, rather than assume, that the tracked backlog remains genuinely
> blocked and that no new disclosed-but-unfiled gap has surfaced since T19
> — and take no ticket this sprint rather than manufacture one.** All 8
> open issues were independently re-verified live and found exactly as
> blocked as T19 left them: three on DECISION D1, two on a real IdP tenant
> this environment cannot provision, two on genuine Product Owner input,
> one on real assistive-technology hardware. `HANDOFF.md`'s Cross-cutting
> section was re-read in full for a third time running (T18, T19, T20) and
> produced no third #212/#213-shaped find — the one candidate that looked
> promising, the `golang-migrate`/`goose` migration-tooling swap, is
> confirmed to be settled, on-the-record roadmap debt rather than a
> disclosed gap, per four separate prior ceremonies' own explicit ruling,
> and reopening that ruling on no new fact would itself be the scope
> manufacturing this discipline exists to prevent. D1 and D2 go back to the
> user unanswered, the latter with its strongest possible null result yet:
> no PR exists this sprint to have tested the interim rule against.

**What this sprint does not claim** (the half PM insists on):

- **This is not evidence the project has run out of real work**, and this
  plan does not claim that. It is evidence that this project's own
  discipline — take only what's genuinely unblocked, file what's genuinely
  disclosed, don't reopen a settled roadmap-debt classification without a
  new fact — produced zero this time, the same way it produced 1 at T18 and
  2 at T19. The size of a sprint is an output of that discipline, not an
  input PM/PE negotiate toward a target.
- **This does not mean the 8 tracked issues, or the roadmap-debt items
  (`golang-migrate`/`goose`, Observability, ISO-8601 weekday encoding),
  have become less real or less worth doing eventually.** Every one of them
  is exactly as real and exactly as blocked (or exactly as
  correctly-unticketed) as it was at T19 — nothing about their disposition
  changed, only that this ceremony re-verified it rather than assuming it.
- **D1 and D2 remain unanswered**, and no T20 artifact — because there are
  no T20 tickets — implements, narrows, or guesses at either.
- **This is not a precedent for skipping the Cross-cutting re-scan in future
  sprints.** The check ran in full (§A3/§A5) and is exactly what produced
  this sprint's honest zero — a ceremony that stopped checking because a
  prior sprint found zero would be the failure mode this discipline exists
  to prevent, mirrored onto planning diligence itself.

## Tickets — 0 items, 0 points

**None.** No ticket is dispatched this sprint. §A6 states the reasoning in
full: every tracked issue is genuinely blocked (§A5), and the one
Cross-cutting candidate that looked promising on a first read is settled
roadmap debt, not a disclosed gap, per four prior ceremonies' own explicit
ruling that this ceremony found no new fact to overturn (§A3).

## Waves

**None.** No ticket exists to wave-assign.

## Recorded disagreements (Ceremony 2 rule 3 — not smoothed over)

**PM's residual concern, recorded rather than smoothed over — a sharper
version of the same concern PM raised at T18.** PM's mandate is protecting
product value and market timing, and a 0-ticket sprint is smaller than any
sprint this project has run, including T18's 1-ticket, 8-point sprint PM
already flagged as thin. PM's position: two consecutive small sprints
(T18, T19) followed by a zero-ticket sprint reads, from outside this
process's own reasoning, as a stalled backlog — and PM asks that this
concern be carried to the user explicitly rather than only defended inside
this document, since a genuinely blocked backlog and a mismanaged one are
indistinguishable from the outside without the per-issue verification this
ceremony performed.

**PdE and PE's position, which governs, restated rather than merely
repeated:** the check this concern asks for is not a future action item —
it is §A3/§A5's own per-issue live re-verification, already performed and
already the artifact PM's own concern can be checked against. Every one of
the 8 issues' blockers was independently re-fetched from the API this
ceremony, not carried forward from memory, and none has moved since T18.
PdE/PE do not dispute that a zero-ticket sprint is a signal worth
surfacing to the user — they hold that the correct response to that signal
is exactly what this document already does (name each blocker precisely,
by issue number, with its own re-verified state), not to manufacture
ticket count to make the signal go away. **BA and QA concur with PdE/PE on
the specific question of whether the migration-tooling item should be
promoted to a ticket** — BA's read of the four-ceremony precedent (§A3) is
that reopening it now, on the strength of one prior sprint's aside and no
new fact, would itself be an unrecorded process change, which is a
different and worse failure than a small sprint.

**Not overridden by the discomfort of the number — PdE/PE's position that
no genuinely unblocked ticket exists this sprint is what governs, and that
claim is checkable** (§A5's per-issue blocker table, re-verified live),
not asserted. PM's concern is carried forward as a standing flag for the
next ceremony that finds the same shape twice running, per this project's
own "don't restate a mechanism a third time, change its shape" discipline
(`sprint-process.md`'s Scheduled-removals table) — if T21 also lands at
zero or near-zero tickets with the same blockers unchanged, that is the
point to escalate the backlog's overall shape to the user directly, not to
manufacture a ticket at T20 to defer the conversation one more sprint.

## Sprint-level Definition of Done

All of `sprint-process.md`'s standing DoD, adapted for a sprint with no
tickets, stated now so it is not improvised at the retro:

1. **No ticket to merge; sprint goal met as stated** — confirm-and-report,
   not build-and-ship — or explicitly descoped with reasoning recorded (it
   is not: the goal as stated is fully met by this ceremony's own
   verification work).
2. **The merged-fix issue sweep run and reported with its count** — by the
   retro (reporting, not blocking) and again by T21's Ceremony 1
   (authoritative). Expected to be trivially clean, since no PR merges this
   sprint to have closed anything — the retro should still report the exact
   count rather than assume it, per this project's own "silence is
   indistinguishable from not having checked" standard.
3. **Scoring owed at the retro:**
   - **(a)** Did this ceremony's own claim — that all 8 open issues remain
     genuinely blocked, re-verified live rather than assumed — hold for the
     whole sprint, i.e. did none of their blockers resolve mid-sprint
     without anyone noticing (a live re-check at retro time, not a re-read
     of this document)?
   - **(b)** Is the `golang-migrate`/`goose` roadmap-debt classification
     (§A3) still correctly unticketed at retro time, or did anything
     surface mid-sprint that would change that read?
   - **(c)** Did D1 or D2 get answered mid-sprint? (Not expected, per §A7 —
     scored regardless, per this project's own discipline of scoring a
     prediction whichever way it lands.)
4. **Not scoreable by T20 and deliberately not pre-empted:** D1 and D2
   remain the user's. If either is answered mid-sprint, the answer's own
   trigger takes over and this plan does not constrain it.
5. Retro in `docs/process/t20-retro.md`, indexed by a `## T20 sprint retro`
   stub in `docs/LESSONS.md`. `HANDOFF.md`/`CLAUDE.md` state updated —
   noting that **T21's Ceremony 1**, not the retro, corrects T20's
   Docs-index row (the ordinary convention).
