# T24 Sprint Plan — Ceremonies 1 + 2

Backlog refinement (Ceremony 1) and sprint planning (Ceremony 2) per
`docs/process/sprint-process.md`, six-role team (briefs:
`docs/agent-operating-handbook.md` Part B). Held against
`docs/process/t23-retro.md` (PR #225, merged `2026-08-16T08:04:37Z`),
`HANDOFF.md` including its Docs index, `CLAUDE.md`, and the live PR/issue
state of `nhuthuynh/white-label` (GitHub-side name `pickleball-platform`).

**Every factual claim below was re-derived against the repository and the
GitHub API during this ceremony**, at the merged branch tip (`d9dd0cd`, T23
retro's own merge), rather than taken from the retro's or T23's plan's prose
— CLAUDE.md rule 10 applied to planning. Per T23 retro's recommendation 2,
this ceremony keeps the reopening-condition check and the "is this healthy"
question at the length they now warrant (a one-line check, not a fresh
essay) rather than re-deriving analysis three prior ceremonies already
settled.

## §A0 — Correcting T23's Docs-index row and Task-backlog narrative: this
ceremony's first job, per the retro's own instruction

Per `sprint-process.md`'s standing convention (a retro PR never updates the
Docs-index row that points at it, since that row must cite the retro's own
merge PR number, which does not exist until it merges) and per
`docs/process/t23-retro.md`'s own closing line — *"T24's Ceremony 1 corrects
T23's row, including the honest-form sentence above, as its first job — the
same standing convention every prior ceremony has followed"* — **performed
directly in `HANDOFF.md` by this same PR**, not summarized twice here. What
was corrected, and how each fact was independently re-verified rather than
copied from the retro's prose:

| Field | What the row now says | Independently re-verified this ceremony |
|---|---|---|
| Retro link | `docs/process/t23-retro.md` (no incident-grade finding; independently re-verified live that all 8 issues' blockers held, byte-for-byte against the plan's own table, and that the migration-tooling classification stayed unticketed; scored DoD (d) live for the third time running with an identical result; found the "is a 0-ticket sprint healthy" engagement question now genuinely exhausted at four consecutive 0-ticket sprints; 4 recommendations for T24) | File exists, read in full this ceremony; its own characterization matches its content |
| Reviews cell | PRs #224 (Ceremony 1/2 doc) → #225 (retro doc), in that merge order | Re-fetched both via `list_pull_requests`: `merged_at` **07:58:07Z → 08:04:37Z** — ascending, matching the retro's own dispatch-then-retro order |
| Task-backlog narrative | The retro's own agreed sentence, quoted verbatim from `docs/process/t23-retro.md`'s "The sprint goal, scored" section, per `sprint-process.md` Ceremony 1 item 3 ("the retro's form, not a stronger one") | Cross-checked word-for-word against the retro's own blockquote — copied, not paraphrased |

**A new T24 row is added** in the same before-the-sprint form, Retro and
Reviews honestly marked "not yet written" / "not yet opened", for T25's
Ceremony 1 to correct in turn — the same convention every prior ceremony has
followed.

## §A1 — The merged-fix issue sweep, run as this ceremony's first
substantive act (per DoD, T24's Ceremony 1 is the authoritative moment)

**Step 1 — list the open issues, live.** `list_issues(state: OPEN)` at
ceremony start → **`totalCount: 8`**: #124, #126, #130, #134, #144, #145,
#149, #164. Identical set to T23's own Ceremony 1 sweep and T23's retro
sweep before it.

**Step 2 — reconcile arithmetically, before reading any issue
individually.** `open_at_end_of_T23's_own_retro − closed_since_then +
opened_since_then`. T23's own retro (its sweep moment 1,
`docs/process/t23-retro.md` finding 1) left the count at **8**. Since that
retro's own PR (#225) merged, zero PRs have merged (confirmed by `git log`,
below) — `closed_since = 0`, `opened_since = 0`. `8 − 0 + 0 = 8`. **Matches
the live `totalCount: 8` read at this ceremony's start exactly.**

**Step 3 — cross-reference merged PRs against the open list.**
`list_pull_requests(state: closed, base: claude/go-backend-pickleball-7up34j,
sort: updated, direction: desc)` → the most recent entry is **#225 itself**
(T23's own retro doc, `merged_at: 2026-08-16T08:04:37Z`); nothing merged
after it. `git log --oneline -3` at this ceremony's branch cut shows
`d9dd0cd` (PR #225's merge, "docs: T23 sprint retro (Ceremony 3)") as the
tip with no descendants, and `git status`/`git fetch` were clean before this
ceremony's branch was cut. **Zero PRs to cross-reference against the open
list** — the correct, trivial shape for a sprint whose predecessor shipped no
tickets, checked rather than assumed from the retro's own prediction.

**Sweep result: clean, continuing the unbroken run since T15.** **T25's
Ceremony 1 still re-runs this sweep in full**, per the standing rule that a
prior ceremony's clean result does not discharge the next one.

## §A2 — T23 retro's four recommendations, each given a disposition

| # | Recommendation | Disposition |
|---|---|---|
| 1 | Continue treating the merged-fix sweep as authoritative regardless of the prior retro's clean result — re-verify from the API, not from the retro's table | **Executed here** (§A1) — the open count was independently re-derived from the live API and reconciled arithmetically, not read off the retro's own table |
| 2 | Keep scoring DoD (d) live every sprint, but stop treating it as needing its own dedicated finding-length write-up once it has run this many times cleanly — a one-line "checked, neither fired" plus the two sub-checks is sufficient going forward unless a condition actually fires | **Executed here, in the shortened form the recommendation asks for** (§A5) — both conditions re-checked against freshly-fetched facts this ceremony, reported in one line plus its two sub-checks rather than a dedicated finding. **Neither fired** |
| 3 | Carry forward the two running counts using the distinct shapes finding 6 establishes — the backlog's consecutive-static-check count increments every ceremony (was 6); D1's consecutive-sprint-silence count increments only when the sprint number advances (was 10, becomes 11 only if T24 opens with #144 still unanswered) | **Executed here** (§A3/§A7) — both counts independently re-verified as unchanged this ceremony and incremented correctly: the backlog is now static across the **seven** most recent live checks; D1's single T14.3 comment has now stood for **eleven** consecutive sprints (T14 through T24), since this ceremony re-confirmed #144 still carries exactly one comment |
| 4 | Do not write a fresh multi-paragraph "is a 0-ticket sprint healthy" analysis at T24 absent one of T21 retro's two named conditions actually firing | **Executed here** (§A6) — neither condition fired, so this ceremony states that result plainly rather than re-litigating the question a fifth time |

## §A3 — What this ceremony verified, and how

| Claim | How it was checked | Result |
|---|---|---|
| Open-issue count at ceremony start | live `list_issues(state: OPEN)` | `totalCount: 8`, matching #124/#126/#130/#134/#144/#145/#149/#164 exactly |
| Every one of the 8 open issues' blockers are unchanged since T23's retro | `list_issues` with comment counts and `updated_at`, live; cross-checked against `docs/process/t23-retro.md`'s own per-issue table | #144 (D1) still exactly 1 comment, `updated_at 2026-08-15T07:01:03Z`; #149 still 3 comments, `2026-08-15T16:56:58Z`; #164 still 1 comment, `2026-08-15T14:16:28Z`; #124 still 1 comment, `2026-08-15T16:25:34Z`; #145 still 1 comment, `2026-08-15T05:01:29Z`; #126/#130/#134 still 0 comments, `2026-08-14T16:12:26Z`/`2026-08-14T16:30:25Z`/`2026-08-14T16:37:49Z`. **Every field matches the retro's own live-fetched table exactly, byte-for-byte — zero blockers changed state, zero new comments on any of the 8** |
| D1 (#144) still has exactly one comment | `issue_read(get_comments)`, re-fetched the comment body, not just the count | confirmed — T14.3's original escalation only, unchanged across T14 through T23, now T24 (twelfth sprint) |
| D2 (ADR-0016) still unanswered | read `docs/adr/0016-*.md`'s `## Status` in full | unchanged: **"Escalated — awaiting the user's decision. This ADR decides nothing."** No option marked chosen |
| D1 (ADR-0015) still unanswered | read `docs/adr/0015-*.md`'s `## Status` in full | unchanged: **"Escalated — awaiting product decision (D1). Deliberately not Accepted, and no option below is chosen."** No option (a)–(d) marked chosen anywhere in the document |
| No PR merged since T23's retro's own PR | `git fetch`, `git log --oneline -3`, `git status`, `list_pull_requests` most-recent entry | clean, up to date with `origin/claude/go-backend-pickleball-7up34j`, HEAD at `d9dd0cd` (T23 retro's own merge) before this ceremony's branch was cut; most recent merged PR against the shared branch is #225 itself |
| `HANDOFF.md`'s Cross-cutting section re-read in full for any new disclosed-but-never-filed gap — the same check run for the seventh sprint running (T18, T19, T20, T21, T22, T23, T24) | Read the entire Cross-cutting section (`HANDOFF.md`, the full block between "## Cross-cutting / later" and "## Definition of Done", all ~665 lines) line by line | **No new candidate of #212/#213's shape found.** Every open thread is one of: already closed with a closing note in place (ServiceOptions/#123, CreateUser squatting/T12.9, panic-recovery/#89+T10.7, Competitions PayableType/#96, `facility_id` reconciliation/T8.3, courts-list/T8.2, camera consent/T8.4, npm peer-deps/#79, TypeScript pin/#79, Registration/CancelGame authz split/T5.5, `RefundPayment` wiring/T12.3, waitlist position race/T6.6-loop-2); already tracked as one of the 8 open issues (#126 price field, #130 no-show refund, #134 WCAG); explicitly and structurally blocked on something outside this environment (Jenkins server-side wiring — no reachable instance/admin credentials); or a deliberate, named, low-urgency roadmap item (Observability, ISO-8601 weekday encoding, the `golang-migrate`/`goose` swap, native mobile Swift/Kotlin clients — checked separately below). T19's Ceremony 1 remains the only one that ever found a genuine #212/#213-shaped gap here; this is the sixth re-scan since (T20, T21, T22, T23, and now T24) to confirm the section holds no third instance |
| The `golang-migrate`/`goose` migration-tooling swap, checked against this project's own settled precedent, not treated as new | `grep -rn "golang-migrate\|goose" HANDOFF.md docs/process/*.md docs/adr/*.md`, then re-read the T11–T14/T21/T22/T23 sprint plans' own dispositions | **Same result as T20's, T21's, T22's and T23's own checks**: seven separate prior ceremonies (T11–T14, T21, T22, T23) already and explicitly ruled this out of the mandatory-issue treatment — "roadmap items, not disclosed gaps." No new fact surfaced this sprint (zero PRs merged) that could overturn that classification. **Disposition: still correctly unticketed, unchanged** |
| `go test ./internal/.../domain/... ./internal/.../app/... -race` (`make test-domain`) still green on the unmodified tree | ran directly | all 12 packages green |
| `make gate-coverage` still green | ran directly | `OK — all 42 package(s) with runnable tests are executed by "ci-checks"` — unchanged from T23, no new package landed since no PR merged |
| Next free migration / ADR numbers, checked in case a ticket needed one | `ls db/migrations`, `ls docs/adr` | `0023` last migration, `0016` last ADR → `0024`/`0017` would be free — **not consumed this sprint**, no ticket needs one |
| Local worktree matches the shared branch's real tip | `git fetch`, `git status`, `git log --oneline -3` | clean, up to date with `origin/claude/go-backend-pickleball-7up34j`, HEAD at `d9dd0cd` before this ceremony's branch was cut |

**Not re-verified by this ceremony, and named as such:** the full `make
test`/`ci-integration` Docker path (no Docker daemon reachable here, same
standing gap as every prior sprint); `make generate`/`go build ./...`/`go
vet ./...` (no `buf`/`sqlc`-generated `internal/gen` in this session's
environment, same standing gap named by every prior planning-only
ceremony); any claim about what Jenkins would run (no Jenkins job exists).

---

# Ceremony 1 — Backlog refinement

## §A4 — Migration-header-ownership check: not applicable this sprint

No T24 ticket exists (§A6), so no ticket names a database table or a
migration file. Stated explicitly rather than silently omitted, per this
project's own "silence is indistinguishable from not having checked"
standard.

## §A5 — The whole open backlog, ranked, with a disposition for each

All 8 open GitHub issues, re-verified live (§A3):

| Issue | Ranked | Disposition |
|---|---|---|
| **#144** `CancelBooking`/`CreateBooking` have no authz | **Escalated** | D1 unanswered; §A7. Twelfth sprint carrying this |
| **#149** Payments' remaining caller-supplied fact (`booking_host_id`) | **Untouched, correctly** | Blocked on D1 exactly as its own text says |
| **#124** court-Bookings half of the cascade | **Untouched, correctly** | Blocked on D1, same reason as #149; re-verified still open with its T16.3 comment intact |
| **#164** ADR-0014 actor-column conformance | **Deferred, still blocked** | Needs a real IdP tenant, unreachable from this environment; unchanged since T14 |
| **#145** pre-existing UUID rows vs. `Principal.Subject` | **Deferred** | Same real-IdP-tenant blocker as #164; unchanged |
| **#126** real per-Game price field | **Deferred** | Needs Product Owner input on whether the price is per-Game, per-Registration, or per-head; unchanged since T8.10 |
| **#130** refunding a `no_show_fee` Payment | **Deferred** | Carries its own stated open product question; unchanged |
| **#134** WCAG manual screen-reader pass | **Deferred** | Needs a `role:ux-ui-designer` pass with real assistive-technology hardware this environment cannot provide; T24 ships no UI change |

**Every one of the 8 tracked issues remains genuinely blocked, independently
re-verified rather than assumed** (§A3) — the thirteenth consecutive time
this exact set of blockers has been re-checked and found unchanged (T12
through T24). Zero of the 8 open issues are actionable this sprint without
guessing at D1, a real IdP tenant, a Product Owner decision, or
assistive-tech hardware.

**This ceremony's own re-scan of `HANDOFF.md`'s Cross-cutting section (§A3)
found nothing new to promote into an issue — the seventh sprint running this
exact check has come up empty (T18, T19, T20, T21, T22, T23, T24)** — T19's
Ceremony 1 is still the only one that found a genuine #212/#213-shaped gap in
this section.

**DoD (d) live check, reported at the length T23 retro's recommendation 2
asks for (a one-line result plus its two sub-checks, not a dedicated
finding) — neither of T21 retro's two named reopening conditions fired:**

1. **A materially different blocker profile — no.** No ninth issue joined
   D1's cluster (still exactly #144/#149/#124), and none of the five
   externally-blocked issues (#164/#145/#126/#130/#134) turned out
   answerable-from-inside-this-environment and left unacted-on — re-checked
   against each issue's own blocker text this ceremony (§A3).
2. **The backlog running dry entirely — no.** `totalCount: 8`, the identical
   eight issues, none closed, none replaced. It has not moved at all, let
   alone run dry.

## §A6 — Why zero tickets, and why that is the honest answer rather than a
gap in this ceremony's diligence

**The two alternatives this project has repeatedly named and rejected are
still rejected, for the same reasons:**

1. **Manufacture a ticket against one of the 8 blocked issues by guessing
   at its blocker.** Not done — every blocker was independently
   re-verified live this ceremony (§A3/§A5), and none has moved.
2. **Take the migration-tooling item (or Observability, or the ISO-8601
   weekday-encoding note, or native mobile clients) as new scope.** Not
   done, for the reason §A3's table states in full: this is settled,
   on-the-record roadmap debt, not a disclosed gap, per seven separate prior
   ceremonies' own explicit ruling, with no new fact this sprint to
   overturn it.

**The third alternative — the one this ceremony takes — is a 0-ticket
sprint, the fifth in this project's history, stated with its reasoning in
full rather than left implicit.** This ceremony independently re-ran every
check that produced that same conclusion at T20, T21, T22, and T23
(§A3/§A5) and reached the identical result on freshly-fetched facts, not by
assuming any prior ceremony's diligence still holds. **No new
#212/#213-shaped gap surfaced this time either.**

**On whether a fifth consecutive 0-ticket sprint needs its own fresh "is
this healthy" pass: no, and per the task's own explicit instruction this is
not re-derived from scratch here.** `docs/process/t23-retro.md` finding 7
scored this question genuinely exhausted at four consecutive 0-ticket
sprints — engaged at depth twice (T20, T21), closed by the user's own direct
answer, and its reopening mechanism independently exercised three times
(T22 retro, T23 Ceremony 1, T23 retro) with an identical "neither condition
fired" result each time. This ceremony's own live check (§A5) is a fourth
independent run of that same mechanism, on freshly-fetched facts rather than
inherited from T23's account, and it too found neither condition fired.
Per T23 retro's own recommendation 4 — *"do not write a fresh
multi-paragraph 'is a 0-ticket sprint healthy' analysis at T24 absent one of
T21 retro's two named conditions actually firing"* — this section states the
check and its result rather than re-arguing the question a fifth time.

**Dependency-completeness check: not applicable.** With no ticket, there is
no producer/consumer pair to check either question of
(`sprint-process.md`'s two-question form).

## §A7 — DECISION D1 and DECISION D2 remain unanswered

**Re-verified this ceremony, not assumed** (§A3): #144 carries exactly one
comment, T14.3's original escalation, unchanged across T14 through T23, now
T24. Both ADR-0015's and ADR-0016's own `## Status` fields are unchanged,
read in full this ceremony.

> **DECISION D1 (for the user / Product Owner) — twelfth deferral.** When
> somebody books a court through the public flow *without an account* —
> which is how the flow works today — who should be allowed to cancel that
> booking later, and should booking without an account remain possible at
> all? `docs/adr/0015-booking-ownership-for-public-bookings.md` lays out
> four options with their costs and recommends none. No T24 ticket exists to
> implement this (there are no T24 tickets at all), and per ADR-0015's
> restriction list no T24 PR may guess at it regardless. Its footprint is
> unchanged from T23's own re-check: still the same two named instances of
> shaped scope (`CancelBooking`'s own missing check; the court-Bookings half
> of #124), neither grown nor shrunk. D1 has now carried its single T14.3
> comment for **eleven consecutive sprints (T14 through T24)** with no
> second escalation attempt beyond the original ADR text — the precise,
> re-checked count carried forward per T22 retro's recommendation 3, not
> re-derived from scratch. Per T21 retro's recommendation 2, the
> escalation-mechanism question this deferral's length once raised has
> already been put to the user and answered (keep re-deferring); this
> ceremony does not re-raise it, only names the count honestly.

> **DECISION D2 (for the user) — ninth deferral, tenth sprint open.** When
> the same session both reviews and merges a pull request, may it also
> write the code on that pull request — and if so, under exactly what
> limits? `docs/adr/0016-reviewer-authored-code-on-a-reviewed-pull-request.md`
> lays out four options and recommends none. T23 shipped no reviewer-authored
> gap-fix (there were no PRs at all this sprint to test the interim rule
> against) — its fourth consecutive sprint of that structurally weaker
> "no PR existed" shape, distinct from T15–T19's five "a PR existed and
> needed no fix" instances (T21 retro named this distinction explicitly).
> **T24 also has no tickets**, so this sprint is a **fifth** instance of
> that same weaker null result (T20, T21, T22, T23, T24), bringing the
> combined total to ten sprints (T15 through T24) of D2 sitting open — not
> a tenth instance of T15–T19's stronger one, named precisely rather than
> folded into the same count.

**Neither is implemented, decided, or guessed at by this ceremony.** Both
remain exactly as blocked as `sprint-process.md`'s own restriction lists
require.

## §A8 — Shared-file pre-assignment, and same-wave verification: not
applicable this sprint

| Artifact | Owner | Notes |
|---|---|---|
| **`HANDOFF.md`** | **this ceremony only** | An implementer that finds a stale line flags it for T25's Ceremony 1 and does not edit it — the standing rule, unchanged. No execution ticket exists this sprint to collide with it |
| **`docs/process/sprint-process.md`** | **not touched this ceremony** | No amendment is landed this ceremony |

**Same-wave shared-interface verification rule: does not apply — trivially,
by construction.** T24 has zero tickets. There is no wave to have a
collision within.

---

# Ceremony 2 — Sprint planning

## Sprint goal

> **Correct the record for T23, confirm rather than assume that the tracked
> backlog remains genuinely blocked, and continue treating the "is a
> 0-ticket sprint healthy" question as the settled matter T21 retro's
> recommendation established it to be — checking its two named reopening
> conditions live, every sprint, in the abbreviated form T23 retro's
> recommendation 2 asks for, rather than re-litigating the underlying
> question at length.** All 8 open issues were independently re-verified
> live and found exactly as blocked as T23 left them: three on DECISION D1,
> two on a real IdP tenant this environment cannot provision, two on genuine
> Product Owner input, one on real assistive-technology hardware.
> `HANDOFF.md`'s Cross-cutting section was re-read in full for a seventh
> sprint running (T18–T24) and produced no third #212/#213-shaped find. The
> `golang-migrate`/`goose` roadmap-debt classification remains settled on no
> new fact. Neither of T21 retro's two reopening conditions fired, checked
> live rather than assumed — the fourth consecutive time this specific check
> has been performed and scored.

**What this sprint does not claim** (the half PM insists on):

- **This is not evidence the project has run out of real work**, and this
  plan does not claim that. A fifth zero-ticket sprint running is the
  honest output of a discipline that takes only what is genuinely unblocked
  and files what is genuinely disclosed — not a target this team engineered.
- **This does not mean the 8 tracked issues, or the roadmap-debt items,
  have become less real or less worth doing eventually.** Every one is
  exactly as real and exactly as blocked (or exactly as
  correctly-unticketed) as it was at T23.
- **D1 and D2 remain formally open and unanswered as ADR decisions.**
  Nothing this ceremony did or found bears on either decision itself —
  only on whether the backlog and the two named reopening conditions are
  unchanged, which they are.
- **This is not a precedent for skipping the Cross-cutting re-scan, the
  live blocker re-check, the merged-fix sweep, or the DoD (d) reopening-
  condition check in future sprints.** All four ran in full this ceremony
  (§A1/§A2/§A3/§A5/§A6) and are exactly what produced this sprint's honest
  zero.
- **This is not a precedent for skipping the "is this healthy" question
  forever, either — only for not re-deriving it from scratch when nothing
  new has happened.** If either of T21 retro's two named conditions fires
  in a future sprint, that ceremony engages it directly rather than citing
  this entry as a reason not to.

## Tickets — 0 items, 0 points

**None.** §A6 states the reasoning in full: every tracked issue is
genuinely blocked (§A5), and the one Cross-cutting candidate that has looked
promising on past reads is settled roadmap debt, not a disclosed gap, per
seven prior ceremonies' own explicit ruling that this ceremony found no new
fact to overturn (§A3).

## Waves

**None.** No ticket exists to wave-assign.

## Recorded disagreements (Ceremony 2 rule 3 — not smoothed over)

**None recorded this sprint.** The PM/PdE-PE disagreement about sprint size
and the shrinking-backlog signal that ran through T18–T20's plans was
resolved at T20 retro (sharpened into D1's silence specifically) and closed
at T21 (the user answered directly). No party records a fresh disagreement
with continuing that resolution this ceremony. If a future ceremony's own
live re-check finds a new reason to disagree with continuing the
re-deferral pattern, that is recorded there, on its own terms.

## Sprint-level Definition of Done

All of `sprint-process.md`'s standing DoD, adapted for a sprint with no
tickets, stated now so it is not improvised at the retro:

1. **No ticket to merge; sprint goal met as stated** — correct the record,
   confirm-and-report, hold the settled "is this healthy" question closed
   absent a named trigger — not build-and-ship.
2. **The merged-fix issue sweep run and reported with its count** — by the
   retro (reporting, not blocking) and again by T25's Ceremony 1
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
     decision?
   - **(d)** Did either of T21 retro's two named reopening conditions fire
     mid-sprint (a materially different blocker profile, or the backlog
     running dry entirely)? Scored live again, in the abbreviated one-line
     form T23 retro's recommendation 2 established — not a dedicated
     finding unless a condition actually fires.
4. **Not scoreable by T24 and deliberately not pre-empted:** D1 and D2
   remain the user's own decisions to make, on whatever timeline the user
   chooses.
5. Retro in `docs/process/t24-retro.md`, indexed by a `## T24 sprint retro`
   stub in `docs/LESSONS.md`. `HANDOFF.md`/`CLAUDE.md` state updated —
   noting that **T25's Ceremony 1**, not the retro, corrects T24's
   Docs-index row (the ordinary convention).
