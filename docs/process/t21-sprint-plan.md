# T21 Sprint Plan — Ceremonies 1 + 2

Backlog refinement (Ceremony 1) and sprint planning (Ceremony 2) per
`docs/process/sprint-process.md`, six-role team (briefs:
`docs/agent-operating-handbook.md` Part B). Held against
`docs/process/t20-retro.md` (PR #219, merged `2026-08-15T19:52:17Z`),
`HANDOFF.md` including its Docs index, `CLAUDE.md`, and the live PR/issue
state of `nhuthuynh/white-label` (GitHub-side name `pickleball-platform`).

**Every factual claim below was re-derived against the repository and the
GitHub API during this ceremony**, at the merged branch tip (`c3f7aac`, T20
retro's own merge), rather than taken from the retro's or T20's plan's
prose — CLAUDE.md rule 10 applied to planning. Where this ceremony repeats a
claim it could not independently re-check, it says so.

## §A0 — Correcting T20's Docs-index row and Task-backlog narrative: this
ceremony's first job, per the retro's own instruction

Per `sprint-process.md`'s standing convention (a retro PR never updates the
Docs-index row that points at it, since that row must cite the retro's own
merge PR number, which does not exist until it merges) and per
`docs/process/t20-retro.md`'s own closing line — *"T21's Ceremony 1 corrects
T20's row, including the honest-form sentence above, as its first job — the
same standing convention every prior ceremony has followed"* — **performed
directly in `HANDOFF.md` by this same PR**, not summarized twice here. What
was corrected, and how each fact was independently re-verified rather than
copied from the retro's prose:

| Field | What the row now says | Independently re-verified this ceremony |
|---|---|---|
| Retro link | `docs/process/t20-retro.md` (no incident-grade finding; independently re-verified live that all 8 issues' blockers held and the migration-tooling classification stayed unticketed; sharpened PM's stalled-backlog concern into D1's eight-sprint silence specifically; 4 recommendations for T21) | File exists, read in full this ceremony; its own characterization matches its content |
| Reviews cell | PRs #218 (Ceremony 1/2 doc) → #219 (retro doc), in that merge order | Re-fetched both via `list_pull_requests`: `merged_at` **19:46:10Z → 19:52:17Z** — ascending, matching the retro's own dispatch-then-retro order (no reordering surprise this sprint, unlike T19/T20's own history) |
| Task-backlog narrative | The retro's own agreed sentence, quoted verbatim from `docs/process/t20-retro.md`'s "The sprint goal, scored" section, per `sprint-process.md` Ceremony 1 item 3 ("the retro's form, not a stronger one") | Cross-checked word-for-word against the retro's own blockquote — copied, not paraphrased |

**A new T21 row is added** in the same before-the-sprint form, Retro and
Reviews honestly marked "not yet written" / "not yet opened", for T22's
Ceremony 1 to correct in turn — the same convention every prior ceremony has
followed.

## §A1 — The merged-fix issue sweep, run as this ceremony's first
substantive act (per DoD, T21's Ceremony 1 is the authoritative moment)

**Step 1 — list the open issues, live.** `list_issues(state: OPEN)` at
ceremony start → **`totalCount: 8`**: #124, #126, #130, #134, #144, #145,
#149, #164. Identical set to T20's own Ceremony 1 sweep and T20's retro
sweep before it.

**Step 2 — reconcile arithmetically, before reading any issue
individually.** `open_at_end_of_T20's_own_retro − closed_since_then +
opened_since_then`. T20's own retro (its sweep moment 1,
`docs/process/t20-retro.md` finding 1) left the count at **8**. Since that
retro's own PR (#219) merged, zero PRs have merged (confirmed by `git log`,
below) — `closed_since = 0`, `opened_since = 0`. `8 − 0 + 0 = 8`. **Matches
the live `totalCount: 8` read at this ceremony's start exactly.**

**Step 3 — cross-reference merged PRs against the open list.** No PR has
merged since T20's retro closed the tree. Re-verified directly rather than
assumed: `git log --oneline -3` shows `c3f7aac` (PR #219's merge, "docs: T20
sprint retro (Ceremony 3)") as the current tip with no descendants, and
`git status` was clean (up to date with
`origin/claude/go-backend-pickleball-7up34j`) before this ceremony's branch
was cut.

**Sweep result: clean, seventh sprint running** (after T15, T16, T17, T18,
T19, T20's own Ceremony-1 sweep, T20's own retro sweep). **T22's Ceremony 1
still re-runs this sweep in full**, per the standing rule that a prior
ceremony's clean result does not discharge the next one.

## §A2 — T20 retro's four recommendations, each given a disposition

| # | Recommendation | Disposition |
|---|---|---|
| 1 | Continue treating the merged-fix sweep as authoritative regardless of the prior retro's clean result — re-verify from the API, not from the retro's table | **Executed here** (§A1) — the open count was independently re-derived from the live API and reconciled arithmetically, not read off the retro's own table |
| 2 | Carry PM's stalled-backlog concern forward, but in the sharper form the T20 retro derived — D1's eight-sprint silence specifically, not the sprint-size trend, which has an ordinary explanation | **Fires this sprint**: T21 also lands at zero tickets (§A6), which is exactly the trigger condition the recommendation named. Per the recommendation, D1's silence — now its ninth deferral, this sprint included — is what is named (§A5/§A7), not a "sprints are shrinking" framing. See recommendation 3's disposition immediately below for how the escalation half of this concern was actually discharged, rather than only restated a third time |
| 3 | T21's Ceremony 1 should explicitly re-examine whether D1's eight-sprint silence changes the cost-benefit of escalating harder (e.g. a direct, short message to the user rather than another ADR comment nobody reads until the next ceremony) — without implementing D1 itself | **Re-examined and closed, not left open.** This exact question was put to the user directly, outside this repository, by the coordinating session, ahead of this ceremony. The user's explicit answer: keep the sprint loop running and let both D1 and D2 continue to be re-deferred each sprint, rather than escalate further. This ceremony does not propose a new escalation mechanism on top of that answer, does not implement or guess at D1/D2, and does not treat the recommendation as still-open guidance requiring further action from this ceremony — the cost-benefit question it raised has been asked and answered by the only party with standing to answer it. D1's ninth deferral is named plainly in §A5/§A7 below, at the same accuracy every prior ceremony has used — neither softened by this disposition nor amplified past what the retro itself found |
| 4 | D1 and D2 stay with the user; no T21 ticket implements `CancelBooking`/`CreateBooking` authorization or a reviewer-authorship carve-out, and neither is guessed at | **Executed here** (§A7) — neither implemented nor guessed at; both remain formally open per ADR-0015/ADR-0016 regardless of recommendation 3's disposition above, which closes the *escalation question*, not the *decisions themselves* |

## §A3 — What this ceremony verified, and how

| Claim | How it was checked | Result |
|---|---|---|
| Open-issue count at ceremony start | live `list_issues(state: OPEN)` | `totalCount: 8`, matching #124/#126/#130/#134/#144/#145/#149/#164 exactly |
| Every one of the 8 open issues' blockers are unchanged since T20's retro | `list_issues` with comment counts and `updated_at`, live; cross-checked against `docs/process/t20-retro.md`'s own per-issue table (finding 2) | #144 (D1) still exactly 1 comment, `updated_at 2026-08-15T07:01:03Z`; #149 still 3 comments, `2026-08-15T16:56:58Z`; #145/#164 still 1 comment each, `2026-08-15T05:01:29Z`/`2026-08-15T14:16:28Z`; #124 still 1 comment, `2026-08-15T16:25:34Z`; #126/#130/#134 still 0 comments, `2026-08-14T16:12:26Z`/`2026-08-14T16:30:25Z`/`2026-08-14T16:37:49Z`. **Every field matches the retro's own live-fetched table exactly, byte-for-byte — zero blockers changed state, zero new comments on any of the 8** |
| D1 (#144) still has exactly one comment | live fetch | confirmed — T14.3's original escalation only, unchanged across T14 through T20, now T21 (ninth sprint) |
| D2 (ADR-0016) still unanswered | read `docs/adr/0016-*.md`'s `## Status` | unchanged: "Escalated — awaiting the user's decision. This ADR decides nothing." |
| No PR merged since T20's retro's own PR | `git log --oneline -3`, `git status` | clean, up to date with `origin/claude/go-backend-pickleball-7up34j`, HEAD at `c3f7aac` (T20 retro's own merge) before this ceremony's branch was cut |
| `HANDOFF.md`'s Cross-cutting section re-read in full for any new disclosed-but-never-filed gap — the same check run for the fourth sprint running (T18, T19, T20, T21) | Read the entire Cross-cutting section (`HANDOFF.md`, the full block between "## Cross-cutting / later" and "## Definition of Done") line by line | **No new candidate of #212/#213's shape found.** Every open thread is one of: already closed with a closing note in place (ServiceOptions/#123, CreateUser squatting/T12.9, RefundPayment wiring/T12.3, panic recovery/#89+T10.7, Competitions PayableType/#96, `facility_id` reconciliation/T8.3, courts-list/T8.2, camera consent/T8.4, npm peer-deps/#79, `npm`/CI partial via SCRUM-6); already tracked as one of the 8 open issues (#126 price field, #130 no-show refund, #134 WCAG); explicitly and structurally blocked on something outside this environment (Jenkins server-side wiring — no reachable instance/admin credentials); or a deliberate, named, low-urgency roadmap item (Observability, ISO-8601 weekday encoding, the `golang-migrate`/`goose` swap — checked separately below) |
| The `golang-migrate`/`goose` migration-tooling swap, checked against this project's own settled precedent, not treated as new | `grep -rn "golang-migrate\|goose" HANDOFF.md docs/process/*.md`, then re-read the T11/T12/T13/T14 sprint plans' own dispositions in full | **Same result as T20's own check**: four separate prior ceremonies (T11–T14) already and explicitly ruled this out of the mandatory-issue treatment — "roadmap items, not disclosed gaps." No new fact surfaced this sprint (zero PRs merged) that could overturn that classification. **Disposition: still correctly unticketed, unchanged** |
| `go test ./internal/.../domain/... ./internal/.../app/... -race` (`make test-domain`) still green on the unmodified tree | ran directly | all 12 packages green |
| `make gate-coverage` still green | ran directly | `OK — all 42 package(s) with runnable tests are executed by "ci-checks"` — unchanged from T20, no new package landed since no PR merged |
| Next free migration / ADR numbers, checked in case a ticket needed one | `ls db/migrations`, `ls docs/adr` | `0023` last migration, `0016` last ADR → `0024`/`0017` would be free — **not consumed this sprint**, no ticket needs one |
| Local worktree matches the shared branch's real tip | `git status`, `git log --oneline -3` | clean, up to date with `origin/claude/go-backend-pickleball-7up34j`, HEAD at `c3f7aac` before this ceremony's branch was cut |

**Not re-verified by this ceremony, and named as such:** the full `make
test`/`ci-integration` Docker path (no Docker daemon reachable here, same
standing gap as every prior sprint); `make generate`/`go build ./...`/`go
vet ./...` (no `buf`/`sqlc`-generated `internal/gen` in this session's
environment, same standing gap named by every prior planning-only
ceremony); any claim about what Jenkins would run (no Jenkins job exists).

---

# Ceremony 1 — Backlog refinement

## §A4 — Migration-header-ownership check: not applicable this sprint

No T21 ticket exists (§A6), so no ticket names a database table or a
migration file. Stated explicitly rather than silently omitted, per this
project's own "silence is indistinguishable from not having checked"
standard.

## §A5 — The whole open backlog, ranked, with a disposition for each

All 8 open GitHub issues, re-verified live (§A3):

| Issue | Ranked | Disposition |
|---|---|---|
| **#144** `CancelBooking`/`CreateBooking` have no authz | **Escalated** | D1 unanswered; §A7. Ninth sprint carrying this |
| **#149** Payments' remaining caller-supplied fact (`booking_host_id`) | **Untouched, correctly** | Blocked on D1 exactly as its own text says |
| **#124** court-Bookings half of the cascade | **Untouched, correctly** | Blocked on D1, same reason as #149; re-verified still open with its T16.3 comment intact |
| **#164** ADR-0014 actor-column conformance | **Deferred, still blocked** | Needs a real IdP tenant, unreachable from this environment; unchanged since T14 |
| **#145** pre-existing UUID rows vs. `Principal.Subject` | **Deferred** | Same real-IdP-tenant blocker as #164; unchanged |
| **#126** real per-Game price field | **Deferred** | Needs Product Owner input on whether the price is per-Game, per-Registration, or per-head; unchanged since T8.10 |
| **#130** refunding a `no_show_fee` Payment | **Deferred** | Carries its own stated open product question; unchanged |
| **#134** WCAG manual screen-reader pass | **Deferred** | Needs a `role:ux-ui-designer` pass with real assistive-technology hardware this environment cannot provide; T21 ships no UI change |

**Every one of the 8 tracked issues remains genuinely blocked, independently
re-verified rather than assumed** (§A3) — the tenth consecutive time this
exact set of blockers has been re-checked and found unchanged (T12 through
T21). Zero of the 8 open issues are actionable this sprint without guessing
at D1, a real IdP tenant, a Product Owner decision, or assistive-tech
hardware.

**This ceremony's own re-scan of `HANDOFF.md`'s Cross-cutting section (§A3)
found nothing new to promote into an issue, the fourth sprint running this
exact check has come up empty (T18, T19, T20, T21)** — T19's Ceremony 1 is
still the only one that found a genuine #212/#213-shaped gap in this
section; every scan since (including this one) has confirmed the section
holds no third instance, not merely repeated the prior finding unchecked.

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
   ruling, with no new fact this sprint to overturn it.

**The third alternative — the one this ceremony takes — is a 0-ticket
sprint, the second in this project's history, stated with its reasoning in
full rather than left implicit.** T20 already established that a 0-ticket
sprint is an acceptable, honestly-reported outcome over fabricated scope
when the backlog genuinely supports no other conclusion; this ceremony
independently re-ran every check that produced that conclusion at T20
(§A3/§A5) and reached the identical result on freshly-fetched facts, not by
assuming T20's diligence still holds. **No new #212/#213-shaped gap
surfaced this time either.** A sprint that manufactures a ticket to avoid
reporting zero is a worse outcome than a sprint that reports zero honestly
— the same "an invariant with no test that could fail is untested, not
proven" standard QA applies to code, applied here to sprint scope itself.

**A second consecutive 0-ticket sprint is not treated as evidence this
discipline has stopped working.** Per §A2's disposition of recommendation
3, the shape of PM's underlying worry — a repeatedly-shrinking, then-zero
backlog reading as a stalled process from outside this document's own
reasoning — has already been put to the user directly by the coordinating
session, and the user's answer affirms the current operating mode (keep the
loop running, keep re-deferring D1/D2) rather than asking for a change to
it. This ceremony does not treat that as license to stop independently
re-verifying the backlog every sprint — §A3/§A5 ran in full, exactly as
before — only as the reason the *escalation* half of the concern is closed
rather than open.

**Dependency-completeness check: not applicable.** With no ticket, there is
no producer/consumer pair to check either question of
(`sprint-process.md`'s two-question form).

## §A7 — DECISION D1 and DECISION D2 remain unanswered

**Re-verified this ceremony, not assumed** (§A3): #144 carries exactly one
comment, T14.3's original escalation, unchanged across T14 through T20, now
T21. ADR-0016's own `## Status` field is unchanged: "Escalated — awaiting
the user's decision. This ADR decides nothing."

> **DECISION D1 (for the user / Product Owner) — ninth deferral.** When
> somebody books a court through the public flow *without an account* —
> which is how the flow works today — who should be allowed to cancel that
> booking later, and should booking without an account remain possible at
> all? `docs/adr/0015-booking-ownership-for-public-bookings.md` lays out
> four options with their costs and recommends none. No T21 ticket exists to
> implement this (there are no T21 tickets at all), and per ADR-0015's
> restriction list no T21 PR may guess at it regardless. Its footprint is
> unchanged from T20's own re-check: still the same two named instances of
> shaped scope (`CancelBooking`'s own missing check; the court-Bookings half
> of #124), neither grown nor shrunk. **Per §A2's disposition of T20 retro
> recommendation 3, the question of whether this silence's length changes
> the cost-benefit of escalating harder than another ADR comment has itself
> been put to the user directly, outside this repository, and answered: keep
> re-deferring.** That answer governs how this ceremony treats the
> *escalation* question, not the underlying decision — D1 itself remains
> exactly as open and exactly as un-guessed-at as every prior sprint.

> **DECISION D2 (for the user) — sixth deferral, seventh sprint open.** When
> the same session both reviews and merges a pull request, may it also
> write the code on that pull request — and if so, under exactly what
> limits? `docs/adr/0016-reviewer-authored-code-on-a-reviewed-pull-request.md`
> lays out four options and recommends none. T20 shipped no reviewer-authored
> gap-fix (there were no PRs at all this sprint to test the interim rule
> against) — its sixth consecutive sprint with nothing to score, of the
> structurally weaker "no PR existed" shape T19's retro distinguished from
> T15–T19's five "a PR existed and needed no fix" instances. **T21 also has
> no tickets**, so this sprint is a second instance of that same weaker null
> result, not a seventh instance of T15–T19's stronger one — named precisely
> rather than folded into the same count. Per the same disposition above,
> the user has been asked whether the length of this deferral changes
> anything and has chosen to continue it.

**Neither is implemented, decided, or guessed at by this ceremony.** Both
remain exactly as blocked as `sprint-process.md`'s own restriction lists
require.

## §A8 — Shared-file pre-assignment, and same-wave verification: not
applicable this sprint

| Artifact | Owner | Notes |
|---|---|---|
| **`HANDOFF.md`** | **this ceremony only** | An implementer that finds a stale line flags it for T22's Ceremony 1 and does not edit it — the standing rule, unchanged. No execution ticket exists this sprint to collide with it |
| **`docs/process/sprint-process.md`** | **not touched this ceremony** | No amendment is landed this ceremony |

**Same-wave shared-interface verification rule: does not apply — trivially,
by construction.** T21 has zero tickets. There is no wave to have a
collision within.

---

# Ceremony 2 — Sprint planning

## Sprint goal

> **Correct the record for T20, confirm rather than assume that the tracked
> backlog remains genuinely blocked, and close the loop on T20 retro's
> most consequential recommendation rather than leaving it open a second
> time.** All 8 open issues were independently re-verified live and found
> exactly as blocked as T20 left them: three on DECISION D1, two on a real
> IdP tenant this environment cannot provision, two on genuine Product Owner
> input, one on real assistive-technology hardware. `HANDOFF.md`'s
> Cross-cutting section was re-read in full for a fourth sprint running
> (T18, T19, T20, T21) and produced no third #212/#213-shaped find. The
> `golang-migrate`/`goose` roadmap-debt classification remains settled on
> no new fact. D1's escalation question — whether eight (now nine) sprints
> of silence change the cost-benefit of pushing harder than an ADR comment
> — was put to the user directly outside this repository and answered:
> continue the standing re-deferral pattern for both D1 and D2. This
> ceremony records that answer once, plainly, rather than re-raising the
> question or proposing a new mechanism.

**What this sprint does not claim** (the half PM insists on):

- **This is not evidence the project has run out of real work**, and this
  plan does not claim that. Two zero-ticket sprints running is the honest
  output of a discipline that takes only what is genuinely unblocked and
  files what is genuinely disclosed — not a target this team engineered.
- **This does not mean the 8 tracked issues, or the roadmap-debt items,
  have become less real or less worth doing eventually.** Every one is
  exactly as real and exactly as blocked (or exactly as
  correctly-unticketed) as it was at T20.
- **D1 and D2 remain formally open and unanswered as ADR decisions.** The
  user's answer discharges the *escalation-mechanism* question T20 retro
  raised — it is not itself an answer to D1 or D2, and no T21 artifact
  reads it as one.
- **This is not a precedent for skipping the Cross-cutting re-scan, the
  live blocker re-check, or the merged-fix sweep in future sprints.** All
  three ran in full this ceremony (§A1/§A3/§A5) and are exactly what
  produced this sprint's honest zero.

## Tickets — 0 items, 0 points

**None.** §A6 states the reasoning in full: every tracked issue is
genuinely blocked (§A5), and the one Cross-cutting candidate that has looked
promising on past reads is settled roadmap debt, not a disclosed gap, per
four prior ceremonies' own explicit ruling that this ceremony found no new
fact to overturn (§A3).

## Waves

**None.** No ticket exists to wave-assign.

## Recorded disagreements (Ceremony 2 rule 3 — not smoothed over)

**None recorded this sprint, and that absence is itself worth stating
rather than leaving silent.** T18, T19, and T20's plans each carried a live
PM/PdE-PE disagreement about sprint size and the shrinking-backlog signal.
That disagreement is not restated here as still-open: T20 retro sharpened
it into a specific, checkable question (D1's silence, not sprint size), and
§A2/§A7 record that the question was put to the user directly and answered
in favor of continuing the current operating mode. PM does not record a
fresh disagreement with that resolution this ceremony — the concern PM
raised at T18–T20 was about a signal reaching the user un-examined, not
about wanting a specific different outcome, and the signal has now reached
the user and been answered. If a future ceremony's own fresh read finds a
new reason to disagree with continuing the re-deferral pattern, that is
recorded there, on its own terms, rather than inherited from this entry.

## Sprint-level Definition of Done

All of `sprint-process.md`'s standing DoD, adapted for a sprint with no
tickets, stated now so it is not improvised at the retro:

1. **No ticket to merge; sprint goal met as stated** — correct the record,
   confirm-and-report, close the escalation loop — not build-and-ship.
2. **The merged-fix issue sweep run and reported with its count** — by the
   retro (reporting, not blocking) and again by T22's Ceremony 1
   (authoritative). Expected to be trivially clean, since no PR merges this
   sprint to have closed anything.
3. **Scoring owed at the retro:**
   - **(a)** Did this ceremony's own claim — that all 8 open issues remain
     genuinely blocked, re-verified live rather than assumed — hold for the
     whole sprint (a live re-check at retro time, not a re-read of this
     document)?
   - **(b)** Is the `golang-migrate`/`goose` roadmap-debt classification
     (§A3) still correctly unticketed at retro time, or did anything
     surface mid-sprint that would change that read?
   - **(c)** Did D1 or D2 get answered mid-sprint, as a formal ADR
     decision — distinct from the escalation-question disposition this
     ceremony already recorded, which is not itself an answer to either?
4. **Not scoreable by T21 and deliberately not pre-empted:** D1 and D2
   remain the user's own decisions to make, on whatever timeline the user
   chooses — this ceremony's disposition of the escalation *recommendation*
   does not shorten or extend that timeline.
5. Retro in `docs/process/t21-retro.md`, indexed by a `## T21 sprint retro`
   stub in `docs/LESSONS.md`. `HANDOFF.md`/`CLAUDE.md` state updated —
   noting that **T22's Ceremony 1**, not the retro, corrects T21's
   Docs-index row (the ordinary convention).
