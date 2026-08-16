# T31 Sprint Retro

Ceremony 3 per `docs/process/sprint-process.md`, six-role team (briefs:
`docs/agent-operating-handbook.md` Part B), held against
`docs/process/t31-sprint-plan.md` (§A0–§A8 and the Ceremony 2 goal/DoD),
`docs/process/t30-retro.md` as the closest 0-ticket-sprint precedent,
`HANDOFF.md`, ADR-0015, ADR-0016, `docs/process/sprint-process.md`'s
"Dispatch isolation" and "PR-body self-verification" sections (both adopted
at T30, exercised for the first time for real this sprint), and the live
issue/PR/commit history on `nhuthuynh/white-label` (GitHub-side name
`pickleball-platform`) — one PR this sprint (#244, the Ceremony 1/2 doc),
zero new issues, zero comments posted.

**Every factual claim below was re-derived against the live repository and
GitHub API this ceremony, not read from T31's plan's own account of itself**
(CLAUDE.md rule 10). Concretely, this retro: re-ran `list_issues(state:
OPEN)` fresh; re-read all 7 open issues' full bodies via the same call, not
just cached `updated_at`/comment-count fields; re-fetched #144's comment
body directly; re-read both ADR-0015's and ADR-0016's `## Status` sections
and their git history directly; re-fetched PR #242, #243, and #244 live
rather than trusting the plan's citation of them; and re-ran
`make test-domain` against the unmodified tip.

**What this retro found, in one sentence, so the findings do not have to be
read before the headline is known:** this is a routine, clean retro with
nothing incident-grade — the live sweep matches the plan's own count
exactly (7 open issues, unchanged), all 7 issues' blockers are independently
re-confirmed unchanged down to their full bodies, D1 and D2 both remain
formally unanswered and D2 was correctly not exercised (zero PRs beyond the
planning doc), both standing process safeguards ("PR-body self-verification"
and the HANDOFF-row-correction convention) were exercised for the first time
for real this sprint and both check out clean, the post-T29
backlog-composition counter increments to **five** and D1's
consecutive-sprint-silence counter holds at **eighteen** (confirmed, not
incremented a second time within this sprint), and — per the now-settled
convention T27's and T30's own retros established, and which T28's and
T29's own retros got wrong — this retro deliberately does **not** touch
`HANDOFF.md`'s own T31 row or Task-backlog narrative.

---

## 1. The merged-fix issue sweep — re-run live, clean, matches the plan exactly

**PO.** Per `sprint-process.md`'s DoD, this retro is sweep moment 1; T32's
Ceremony 1 remains the authoritative moment regardless of this result.

**Step 1 — list the open issues, live, at this retro's start.**
`list_issues(state: OPEN)` → **`totalCount: 7`**: #124, #126, #130, #134,
#144, #145, #149 — identical to T31 plan's own §A1 sweep and to T30 retro's
own closing count.

**Step 2 — reconcile arithmetically.** `open_at_end_of_T31's_own_Ceremony_1
(7) − closed_during_T31 (0) + opened_during_T31 (0) = 7`.
`list_pull_requests(state: closed, base: claude/go-backend-pickleball-7up34j,
sort: updated, direction: desc)` shows the most recent merges as **#244**
(this sprint's own Ceremony 1/2 doc, `merged_at: 2026-08-16T16:11:47Z`) →
#243 (T30's retro) → #242 (T30's Ceremony 1/2 doc) → #241 (T29's retro) —
nothing merged after #244. `git fetch` + `git log --oneline -5
origin/claude/go-backend-pickleball-7up34j` at this retro's branch cut shows
`55bf7e9` (PR #244's squash merge) as the tip with no descendants, matching
the local checkout exactly, with a clean `git status` before this ceremony's
branch was cut. **Matches the live `totalCount: 7` exactly.**

**Step 3 — cross-reference merged PRs against the open list.** Exactly one
PR merged this sprint (#244, the plan document itself), naming no issue
number as a claimed closure in its title. **Zero PRs to cross-reference** —
the correct, trivial shape for a 0-ticket sprint.

**Sweep result: clean.** **T32's Ceremony 1 still re-runs this sweep in
full**, per the standing rule that a prior ceremony's clean result does not
discharge the next one.

## 2. All 7 issues' blockers, re-verified live — full bodies re-read

**QA and BA**, per the task's explicit instruction not to trust the plan's
table without re-checking `updated_at`/comment counts directly.

| Issue | `updated_at`/comments at T31 plan §A3 | `updated_at`/comments now (live, this retro) | Changed? |
|---|---|---|---|
| #149 | `2026-08-16T14:20:31Z`, 4 comments | `2026-08-16T14:20:31Z`, 4 comments | No |
| #145 | `2026-08-15T05:01:29Z`, 1 comment | `2026-08-15T05:01:29Z`, 1 comment | No |
| #144 (D1) | `2026-08-15T07:01:03Z`, 1 comment | `2026-08-15T07:01:03Z`, 1 comment | No |
| #134 | `2026-08-14T16:37:49Z`, 0 comments | `2026-08-14T16:37:49Z`, 0 comments | No |
| #130 | `2026-08-14T16:30:25Z`, 0 comments | `2026-08-14T16:30:25Z`, 0 comments | No |
| #126 | `2026-08-14T16:12:26Z`, 0 comments | `2026-08-14T16:12:26Z`, 0 comments | No |
| #124 | `2026-08-15T16:25:34Z`, 1 comment | `2026-08-15T16:25:34Z`, 1 comment | No |

**Every field matches T31 plan's own live-fetched table exactly,
byte-for-byte — zero blockers changed state, zero new comments on any of
the 7, and every timestamp predates PR #244's own merge (`16:11:47Z`).**

**Every issue's full body re-read this retro (`list_issues` returning each
issue's complete `body`), genuinely reconsidering rather than assuming last
sprint's disposition still holds:**

- **#144 (D1):** still zero authorization on `CancelBooking`/
  `CreateBooking` — `domain.Booking` records no owner field. Blocked on the
  product decision ADR-0015 escalates and does not answer.
- **#149:** still resolves the *actor* to a verified `User.ID` but accepts
  caller-supplied *ownership facts* with no port to resolve them against —
  blocked on D1 plus the still-unbuilt Game-Admin/Competition-Admin durable
  store the issue's own text names as a sub-problem.
- **#124:** `Game.Cancel()` still flips status only; still needs Product
  Owner input on cascade semantics (court release, refund automaticity,
  waitlist handling).
- **#126:** `domain.Game`/`domain.Registration` still carry no price/fee
  field; still needs Product Owner input on whether the price is per-Game,
  per-Registration, or per-head-including-guests.
- **#130:** `RefundPayment` still rejects `PAYABLE_TYPE_NO_SHOW_FEE`; still
  carries its own open product question about whether reversing a
  no-show-fee is wanted at all.
- **#145:** still needs a real IdP tenant issuing a non-uuid `sub` claim,
  which exists nowhere in this environment; both named failure modes still
  fail closed.
- **#134:** still needs a real screen-reader session and rendered-in-browser
  measurements this environment's lack of assistive-technology hardware
  cannot supply; no T31 ticket shipped a UI change for a targeted pass to
  cover.

**Migration-tooling classification (`golang-migrate`/`goose`), re-checked
directly.** `grep -n "golang-migrate\|goose" HANDOFF.md` still returns
exactly the single Cross-cutting line every prior ceremony has quoted; `ls
docs/adr/` still ends at `0017`; `ls db/migrations/` still ends at `0026`.
Nothing changed, because nothing merged this sprint capable of changing it
(§1). Still correctly unticketed — the fifteenth consecutive prior
ceremony's own ruling standing unchallenged by a new fact.

**Point, scored: yes, all 7 issues' blockers held for the whole sprint,
independently re-verified live down to their full bodies, not assumed from
silence or trusted from the plan's own account.**

## 3. D1 and D2 remain formally unanswered — checked directly, not assumed

**PE and PO, jointly.**

- **ADR-0015, `## Status` heading (line 23), read in full this retro.**
  Unchanged: **"Escalated — awaiting product decision. This ADR decides
  nothing."** `git log --oneline -- docs/adr/0015-*.md` shows exactly one
  commit (`5828550`, its T14.3 authoring commit) — no commit since.
- **ADR-0016, `## Status` heading (line 22), read in full this retro.**
  Unchanged: **"Escalated — awaiting the user's decision. This ADR decides
  nothing."** `git log --oneline -- docs/adr/0016-*.md` shows exactly one
  commit (`0aaa912`, its T15.2 authoring commit) — no commit since.
- **#144, re-fetched via `issue_read(get_comments)` this retro.** Exactly
  one comment (`id 5301056598`, `created_at: 2026-08-15T07:01:03Z`) —
  T14.3's original escalation, full text re-read, matching every prior
  retro's quotation verbatim. No second comment exists; `state: OPEN`
  confirmed by §1's own live `list_issues` call.

**Point, scored: neither D1 nor D2 was answered mid-sprint, as a formal ADR
decision.** Both files' git history proves nothing could have changed
either — zero commits touched either file since their original authoring
commits, and this sprint shipped one commit total (#244, a docs-only
planning PR touching only `docs/process/t31-sprint-plan.md` and
`HANDOFF.md` per its own "Files changed" section).

## 4. DECISION D2 — confirmed not exercised this sprint, stated plainly

**PE and PO, jointly**, per the task's instruction to state this point
rather than pass over it because there is "nothing to score."

T31 shipped zero tickets and therefore zero PRs beyond the Ceremony 1/2
planning document itself (§1). D2 asks whether a session that reviews and
merges a pull request may also author code on it — a question that can only
be engaged by a real PR under real review. With no ticket PR opened this
sprint, there was no review to score against any of D2's named shapes.
**This sprint lands in the structurally weaker "no PR existed" shape**
(T21 retro's own naming for this case) — the same shape T20, T21–T27, and
T30 all landed in. Confirmed directly: zero tickets, zero PRs beyond #244
(§1's own cross-reference).

## 5. Standing process safeguards, checked for real for the first time

**PE and PO, jointly.** T30 adopted two safeguards into `sprint-process.md`
and scored them sound on editorial review; T31's own Ceremony 1 is the
first ceremony to actually *apply* both in a live situation. This retro
checks whether they held up in practice, not just on paper.

### 5a. PR-body self-verification

PR #244's body was fetched fresh via `pull_request_read(get)` this retro
(§1). It is substantive and coherent: a full Ceremony 1/2 summary, a "Files
changed" section, and a nine-item test-plan checklist that includes the line
*"This PR's own body verified non-empty via a fresh `pull_request_read(get)`
before reporting done"* — consistent with the claim that the safeguard was
actually followed, though (as the safeguard's own adopting text
acknowledges) a retrospective read cannot prove the *implementing* session
ran that check at the moment it claims to have. What this retro **can**
confirm directly, and does: the PR that resulted is not the empty-body
failure mode "PR-body self-verification" exists to catch. No incident.

### 5b. HANDOFF.md row-correction under the new convention

T31's own Ceremony 1 was also the first ceremony to correct a prior
sprint's Docs-index row and Task-backlog narrative under the convention
T27's retro and T30's retro/plan settled (rather than the twice-mistaken
T28/T29 convention of a retro attempting to correct its own row). Re-verified
directly against live PR data, not trusted from T31 plan's own citation:

- `pull_request_read(get, #242)` → `merged_at: 2026-08-16T15:53:47Z`.
- `pull_request_read(get, #243)` → `merged_at: 2026-08-16T16:03:40Z`.

Both match exactly what `HANDOFF.md`'s T30 Docs-index row now cites
(`PR #242 (Ceremony 1/2 doc) → PR #243 (retro doc) … 15:53:47Z →
16:03:40Z`). The Task-backlog T30 entry's blockquote was also compared,
sentence by sentence, against `docs/process/t30-retro.md`'s own closing
"agreed honest sentence" — it is carried **verbatim**, not paraphrased or
strengthened. **The convention was applied correctly on its first live
use.** No incident.

**Scored: both safeguards held.** Neither is a hypothetical win — both were
checked against live, independently re-fetched data rather than accepted on
the plan's word.

## 6. Running counters

**PO.**

**The post-T29 backlog-composition counter: increments to five.** History:
T29 retro (1) → T30 Ceremony 1 (2) → T30 retro (3) → T31 Ceremony 1 (4) →
this retro (5) — five consecutive live checks finding the identical 7-issue
set unchanged.

**D1's consecutive-sprint-silence counter: confirmed at eighteen, not
incremented a second time within this sprint.** T31's own Ceremony 1 already
computed this as **eighteen consecutive sprints (T14 through T31)** when the
sprint opened with #144 still carrying only its original T14.3 comment. This
retro re-checked #144 live (§3): still exactly one comment, unchanged. Per
the established per-*sprint*, not per-*ceremony*, counting convention —
verified by checking how T29's and T30's own retros handled the identical
continuation question — this retro **confirms the count holds at
eighteen**, rather than incrementing it a second time within the same
sprint. It becomes **nineteen** only if T32 opens with #144 still
uncommented — that is T32's Ceremony 1's own count to take, not this
retro's to pre-empt.

**Stated so a future ceremony does not have to re-derive which counter moves
when:** the backlog-composition counter counts *live checks that found the
set unchanged* (a per-ceremony shape, now five); D1's silence counter counts
*sprints elapsed with no comment* (a per-sprint shape, held at eighteen
since T31's own Ceremony 1 already took this sprint's increment).

## 7. `HANDOFF.md`'s Docs-index row and Task-backlog entry — deliberately NOT touched by this retro

**PO**, per the now-settled convention T27's own retro stated first and
T30's retro/plan both restated: a retro PR cannot cite its own merge PR
number or `merged_at` before it exists, so it is structurally incapable of
correcting its own Docs-index row or writing its Task-backlog Outcome
paragraph correctly. T28's and T29's own retros each got this wrong —
claiming, in their own closing paragraph, to have corrected their own row —
and both mistakes were caught only by the *following* sprint's Ceremony 1.
This retro does not repeat that mistake a third time (the mistake this
task's own framing flags explicitly).

**What this retro does instead:** leaves `HANDOFF.md`'s T31 Docs-index row
and Task-backlog entry untouched, and supplies **the agreed honest-form
sentence** below for **T32's Ceremony 1** to carry into `HANDOFF.md` as its
first job — the same mechanism that correctly produced every prior sprint's
narrative entry from T27 onward.

---

## The sprint goal, scored

> *"Confirm, rather than assume, that the backlog remains exactly as
> blocked as T30's retro left it, correct the two structural bookkeeping
> items T30's retro deliberately deferred, and continue the two running
> counters on their established per-ceremony/per-sprint schedules."*

**Every clause holds, independently re-verified rather than taken from the
plan's own account.** All 7 open issues were re-verified live, full bodies
included, and found unchanged (§2). The `golang-migrate`/`goose`
classification remains settled (§2). D1 and D2 remain formally open,
checked directly against both ADR files and #144's comment (§3). D2 was
correctly not exercised this sprint, stated plainly (§4). Both standing
process safeguards were exercised for real for the first time and both held
(§5). The post-T29 backlog-composition counter increments to five; D1's
silence counter holds at eighteen (§6). `HANDOFF.md`'s own row/narrative
correction is deliberately deferred to T32's Ceremony 1, avoiding the
mistake T28's and T29's own retros made (§7).

**The agreed honest sentence, which T32's Ceremony 1 should carry into
`HANDOFF.md`'s Task-backlog entry and Docs-index row** (`sprint-process.md`
Ceremony 1 item 3 requires the retro's form, not a stronger one):

> T31 shipped zero tickets, the tenth 0-ticket sprint in this project's
> history by total count and the second sprint of the fresh consecutive run
> (T30, T31) since T28 broke the T20–T27 streak, plus the two §A0
> bookkeeping corrections T30's retro deliberately left undone. This retro
> independently re-verified — not trusted — every load-bearing claim: the
> merged-fix sweep's live `totalCount: 7` matches T31 plan's own count
> exactly, arithmetically reconciled (`7 − 0 + 0 = 7`); all 7 open issues'
> blockers were re-checked live down to their full bodies, not just cached
> `updated_at`/comment-count fields, and every one is unchanged — #124,
> #126, #130 need Product Owner input the team cannot supply unilaterally;
> #144 and #149 are blocked on D1; #145 needs a real, non-uuid IdP `sub`
> claim this environment cannot produce; #134 needs real assistive-
> technology hardware this environment does not have. DECISION D2 was
> correctly not exercised this sprint — zero tickets means zero PRs beyond
> the planning doc, landing in the structurally weaker "no PR existed"
> shape. Neither D1 nor D2 was answered mid-sprint as a formal ADR decision,
> both ADR files' `## Status` sections and git history read directly, and
> #144's single T14.3 comment re-fetched and confirmed unchanged. Both
> standing process safeguards adopted at T30 — "PR-body self-verification"
> and the HANDOFF-row-correction convention — were exercised for real for
> the first time this sprint by T31's own Ceremony 1, and this retro
> independently re-checked both against live PR data (PR #244's body;
> `pull_request_read` on #242/#243 against `HANDOFF.md`'s new T30 row) and
> found both held cleanly, with no incident. The post-T29
> backlog-composition counter increments to **five** (T29 retro, T30
> Ceremony 1, T30 retro, T31 Ceremony 1, this retro — five consecutive live
> checks finding the identical 7-issue set unchanged); D1's
> consecutive-sprint-silence counter holds at **eighteen** (T14 through
> T31, confirmed rather than incremented a second time within this sprint —
> it becomes nineteen only if T32 opens with #144 still uncommented). No
> incident-grade finding this sprint. `HANDOFF.md`'s T31 row and
> Task-backlog narrative are left for **T32's Ceremony 1** to correct, per
> the convention T27's own retro and T30's own retro/plan both already
> state.

---

## Recommendations for T32's Ceremony 1 and 2

1. **Re-run the merged-fix sweep as the authoritative moment**, per the
   standing rule — re-verify the open count (7) from the live API rather
   than trusting this retro's table (§1).
2. **Correct `HANDOFF.md`'s T31 Docs-index row and insert the Task-backlog
   Outcome/blockquote using this retro's agreed sentence above**, per the
   ordinary convention (§7) — fetching this retro's own PR number live via
   `list_pull_requests`/`pull_request_read`, not treating it as "already
   known" from context.
3. **Continue the post-T29 backlog-composition counter**, incrementing it
   to six if T32 Ceremony 1's own live check again finds the identical
   7-issue set unchanged (§6).
4. **If a real ticket PR is dispatched in T32, D2's interim rule finally has
   something to be exercised against again** — score whichever of D2's
   named shapes (or a new one) it lands in.
5. **The soft observation from T30 retro §3a (who audits that a dispatched
   wave's text actually named its isolation mechanism) remains open and
   untested** — T31 had zero tickets and zero waves, same as T30. Resolve
   it by example the first time a real multi-implementer wave is
   dispatched, not pre-emptively.
6. **If an eleventh 0-ticket sprint arrives with no new fact, do not reopen
   the "is a 0-ticket sprint healthy" question by default** — T23 retro's
   finding 7 closed this question, and neither of T21 retro's two named
   reopening conditions fired this sprint either (checked live: the 7-issue
   set and every blocker's substance are unchanged, §2; `totalCount: 7`,
   unchanged, §1).

## Sprint-level Definition of Done — scored against what T31's own plan asked

1. **No ticket to merge; sprint goal met as stated?** **Yes** —
   confirm-and-report on the backlog plus the two named bookkeeping
   corrections, independently re-verified above (§2, §3).
2. **The merged-fix issue sweep run and reported with its count (7,
   reconciled arithmetically)?** **Yes** — §1; T32's Ceremony 1 remains
   authoritative.
3. **Scoring owed at the retro:**
   - **(a)** Did the "all 7 open issues remain genuinely blocked" claim
     hold for the whole sprint, checked live at retro time? **Yes** — §2,
     full bodies re-read.
   - **(b)** Is the `golang-migrate`/`goose` roadmap-debt classification
     still correctly unticketed at retro time? **Yes** — §2.
   - **(c)** Did D1 or D2 get answered mid-sprint, as a formal ADR
     decision? **No, neither** — §3.
   - **(d)** Did either of T21 retro's two named reopening conditions fire
     mid-sprint? **No, neither** — §1, §2.
4. **Not scoreable by T31 and deliberately not pre-empted:** D1 and D2
   remain the user's own decisions to make, on whatever timeline the user
   chooses.
5. **Retro in `docs/process/t31-retro.md`** (this document), indexed by a
   `## T31 sprint retro` stub in `docs/LESSONS.md`. Per §7 above,
   `HANDOFF.md`'s Docs-index row and Task-backlog entry are **deliberately
   not corrected by this PR** — left to T32's Ceremony 1.

Retro complete. Issue-tracker actions this ceremony: **none** — the sweep
was clean (§1), no issue needed a label correction, and no correction to an
earlier ceremony's claim on any issue was found necessary. Open count at
ceremony start: **7**. Open count now: **7** (unchanged). No incident rises
to a `docs/LESSONS.md` entry this sprint — every finding above either
confirms a claim held under independent re-verification or confirms a
safeguard held on its first live use — so only the standard index stub is
added.

Toolchain re-verified directly against the unmodified tree this retro:
`make test-domain` — 12/12 packages green, matching T31 plan's own count
exactly (no code has landed since T31's own Ceremony 1 to change either
number).

Per §7 above: **`HANDOFF.md`'s T31 row and Task-backlog entry are not
touched by this PR.** T32's Ceremony 1 corrects them, including the
honest-form sentence above, as its first job.
