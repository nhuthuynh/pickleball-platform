# T36 Sprint Retro

Ceremony 3 per `docs/process/sprint-process.md`, six-role team (briefs:
`docs/agent-operating-handbook.md` Part B), held against
`docs/process/t36-sprint-plan.md` (§A0–§A9 and the Ceremony 2 goal/DoD),
`docs/process/t35-retro.md` as the closest 0-ticket-sprint precedent,
`HANDOFF.md`, ADR-0015, ADR-0016, and the live issue/PR/commit history on
`nhuthuynh/white-label` (GitHub-side name `pickleball-platform`) — one PR
this sprint (#254, the Ceremony 1/2 doc), zero new issues, zero comments
posted. "Dispatch isolation" and "PR-body self-verification" are both
already-validated `sprint-process.md` sections with nothing new to
exercise this sprint — zero tickets, zero waves, same as T30–T35 — so this
retro does not re-run their first-live-use scoring a second time on no new
fact.

**Every factual claim below was re-derived against the live repository and
GitHub API this ceremony, not read from T36's plan's own account of
itself** (CLAUDE.md rule 10). Concretely, this retro: re-ran `list_issues(state:
OPEN)` fresh; re-read all 7 open issues' full bodies individually via
`issue_read(method: get)`, not the batched summary, cross-checked
byte-for-byte against T36 plan's own table, and re-read #144's full
comment thread via `issue_read(method: get_comments)`; re-read both
ADR-0015's and ADR-0016's `## Status` sections and their git history
directly; re-fetched PR #252, #253, and #254 live rather than trusting the
plan's or the task prompt's own citation of them; and re-ran `make
generate` then `make test-domain` against the unmodified tip (`7fea6ad`).

**What this retro found, in one sentence, so the findings do not have to be
read before the headline is known:** this is a routine, clean retro with
nothing incident-grade — the live sweep matches the plan's own count
exactly (7 open issues, unchanged), all 7 issues' blockers are independently
re-confirmed unchanged, D1 and D2 both remain formally unanswered and D2 was
correctly not exercised (zero PRs beyond the planning doc), `HANDOFF.md`'s
T35 row/narrative correction landed by T36's own Ceremony 1 is verified
accurate against freshly re-fetched PR data, the post-T29 backlog-composition
counter increments to **fifteen** and D1's consecutive-sprint-silence
counter holds at **twenty-three** (confirmed, not incremented a second time
within this sprint), the stale-repo-metadata artifact remains present but
functionally inert, and — per the now-settled convention — this retro
deliberately does **not** touch `HANDOFF.md`'s own T36 row or Task-backlog
narrative.

---

## 1. The merged-fix issue sweep — re-run live, clean, matches the plan exactly

**PO.** Per `sprint-process.md`'s DoD, this retro is sweep moment 1; T37's
Ceremony 1 remains the authoritative moment regardless of this result.

**Step 1 — list the open issues, live, at this retro's start.**
`list_issues(state: OPEN)` → **`totalCount: 7`**: #124, #126, #130, #134,
#144, #145, #149 — identical to T36 plan's own §A1 sweep and to T35 retro's
own closing count.

**Step 2 — reconcile arithmetically.** `open_at_end_of_T36's_own_Ceremony_1
(7) − closed_during_T36 (0) + opened_during_T36 (0) = 7`.
`list_pull_requests(state: closed, base: claude/go-backend-pickleball-7up34j,
sort: updated, direction: desc)` shows the most recent merge as **#254**
(this sprint's own Ceremony 1/2 doc, `merged_at: 2026-08-23T05:00:09Z`) → #253
(T35's retro) → #252 (T35's Ceremony 1/2 doc) — nothing merged after #254.
`git log --oneline -5` at this retro's branch cut shows `7fea6ad` (PR #254's
squash merge) as the tip with no descendants, matching
`origin/claude/go-backend-pickleball-7up34j` exactly, with a clean `git
status` before this ceremony's branch was cut. **Matches the live
`totalCount: 7` exactly.**

**Step 3 — cross-reference merged PRs against the open list.** Exactly one
PR merged this sprint (#254, the plan document itself), naming no issue
number as a claimed closure in its title or body. **Zero PRs to
cross-reference** — the correct, trivial shape for a 0-ticket sprint.

**Sweep result: clean.** **T37's Ceremony 1 still re-runs this sweep in
full**, per the standing rule that a prior ceremony's clean result does not
discharge the next one.

## 2. All 7 issues' blockers, re-verified live

**QA and BA**, per the task's explicit instruction to re-verify blockers
live rather than assume T36 plan's disposition still holds — full bodies,
not cached fields.

| Issue | `updated_at`/comments at T36 plan §A3 | `updated_at`/comments now (live, this retro) | Changed? |
|---|---|---|---|
| #149 | `2026-08-16T14:20:31Z`, 4 comments | `2026-08-16T14:20:31Z`, 4 comments | No |
| #145 | `2026-08-15T05:01:29Z`, 1 comment | `2026-08-15T05:01:29Z`, 1 comment | No |
| #144 (D1) | `2026-08-15T07:01:03Z`, 1 comment | `2026-08-15T07:01:03Z`, 1 comment | No |
| #134 | `2026-08-14T16:37:49Z`, 0 comments | `2026-08-14T16:37:49Z`, 0 comments | No |
| #130 | `2026-08-14T16:30:25Z`, 0 comments | `2026-08-14T16:30:25Z`, 0 comments | No |
| #126 | `2026-08-14T16:12:26Z`, 0 comments | `2026-08-14T16:12:26Z`, 0 comments | No |
| #124 | `2026-08-15T16:25:34Z`, 1 comment | `2026-08-15T16:25:34Z`, 1 comment | No |

**Every field matches T36 plan's own live-fetched table exactly,
byte-for-byte — zero blockers changed state, zero new comments on any of
the 7, and every timestamp predates PR #254's own merge (`05:00:09Z` on
2026-08-23).**

**Every issue's full body re-read this retro (`issue_read(method: get)` on
all 7, individually, plus a direct `get_comments` re-fetch on #144),
genuinely reconsidering rather than assuming last ceremony's disposition
still holds:**

- **#144 (D1):** still zero authorization on `CancelBooking`/
  `CreateBooking` — `domain.Booking` records no owner field. Blocked on
  ADR-0015's unanswered product decision. No code has touched
  `internal/booking/domain` this sprint (§1 confirms zero implementation
  PRs merged).
- **#149:** still resolves the *actor* to a verified `User.ID` but accepts
  caller-supplied *ownership facts* with no port to resolve them against —
  blocked on D1 plus the still-unbuilt Game-Admin/Competition-Admin durable
  store. Both halves unchanged.
- **#124:** `Game.Cancel()` still flips status only; the issue's own three
  named product questions (court release, refund automaticity, waitlist
  handling) still need Product Owner input. No new fact this sprint.
- **#126:** `domain.Game`/`domain.Registration` still carry no price/fee
  field; T8.10's disclosed placeholder is still what ships. Unchanged since
  T8.10.
- **#130:** `RefundPayment` still rejects `PAYABLE_TYPE_NO_SHOW_FEE`, pinned
  by two tests in two layers. The issue's own open product question is not
  answerable from symmetry alone. Unchanged.
- **#145:** still needs a real IdP tenant issuing a non-uuid `sub` claim,
  which exists nowhere in this environment; both named failure modes still
  fail closed. Unchanged.
- **#134:** still needs a real screen-reader session and rendered-in-browser
  measurements this environment's lack of assistive-technology hardware
  cannot supply; no ticket has shipped a UI change to any of the three named
  screens for even a targeted pass to cover.

**Migration-tooling classification (`golang-migrate`/`goose`), re-checked
directly.** `grep -n "golang-migrate\|goose" HANDOFF.md` still returns
exactly the single Cross-cutting line every prior ceremony has quoted; `ls
docs/adr/` still ends at `0017`; `ls db/migrations/` still ends at `0026`.
Nothing changed, because nothing merged this sprint capable of changing it
(§1). Still correctly unticketed — nineteen consecutive prior ceremonies'
own ruling standing unchallenged by a new fact.

**Point, scored: yes, all 7 issues' blockers held for the whole sprint,
independently re-verified live, not assumed from silence or trusted from
the plan's own account.**

## 3. D1 and D2 remain formally unanswered — checked directly, not assumed

**PE and PO, jointly.**

- **ADR-0015, `## Status` heading, read in full this retro.** Unchanged:
  **"Escalated — awaiting product decision. This ADR decides nothing."**
  `git log --oneline -- docs/adr/0015-*.md` shows exactly one commit
  (`5828550`, its T14.3 authoring commit) — no commit since.
- **ADR-0016, `## Status` heading, read in full this retro.** Unchanged:
  **"Escalated — awaiting the user's decision. This ADR decides nothing."**
  `git log --oneline -- docs/adr/0016-*.md` shows exactly one commit
  (`0aaa912`, its T15.2 authoring commit) — no commit since.
- **#144, re-fetched via `issue_read(get_comments)` this retro.** Exactly
  one comment (`id 5301056598`, `created_at: 2026-08-15T07:01:03Z`) —
  T14.3's original escalation, full text re-read, matching every prior
  retro's quotation verbatim. No second comment exists; `state: OPEN`
  confirmed by §1's own live `list_issues` call.

**Point, scored: neither D1 nor D2 was answered mid-sprint, as a formal ADR
decision.** Both files' git history proves nothing could have changed
either — zero commits touched either file since their original authoring
commits, and this sprint shipped one commit total (#254, a docs-only
planning PR touching only `docs/process/t36-sprint-plan.md` and
`HANDOFF.md` per its own diff).

## 4. DECISION D2 — confirmed not exercised this sprint, stated plainly

**PE and PO, jointly**, per the task's instruction to state this point
rather than pass over it because there is "nothing to score."

T36 shipped zero tickets and therefore zero PRs beyond the Ceremony 1/2
planning document itself (§1). D2 asks whether a session that reviews and
merges a pull request may also author code on it — a question that can
only be engaged by a real PR under real review. With no ticket PR opened
this sprint, there was no review to score against any of D2's named
shapes. **This sprint lands in the structurally weaker "no PR existed"
shape** (T21 retro's own naming for this case) — the same shape T20,
T21–T27, and T30–T35 all landed in. Confirmed directly: zero tickets, zero
PRs beyond #254 (§1's own cross-reference).

## 5. `HANDOFF.md`'s T35 row correction — verified accurate against freshly re-fetched PR data

**PE**, per the task's own instruction to re-fetch #252/#253 live rather
than trust T36 plan's citation of them.

- `pull_request_read(get, #252)` → `merged_at: 2026-08-16T16:56:31Z`.
- `pull_request_read(get, #253)` → `merged_at: 2026-08-16T17:03:19Z`.

`HANDOFF.md`'s current T35 Docs-index row cites exactly these two values
(`PR #252 (Ceremony 1/2 doc) → PR #253 (retro doc) … 16:56:31Z →
17:03:19Z`), matching the freshly re-fetched data byte-for-byte. The T35
Task-backlog entry's narrative paragraph was also compared, sentence by
sentence, against `docs/process/t35-retro.md`'s own closing "agreed honest
sentence" — the landed text is a faithful prose rendering of that agreed
content (counters, per-issue blockers, and the D1/D2 disposition all
match), not a strengthened or diminished version of it. **T36 Ceremony 1's
correction was accurate on independent re-check.** No incident.

## 6. Running counters

**PO.**

**The post-T29 backlog-composition counter: increments to fifteen.**
History: T29 retro (1) → T30 Ceremony 1 (2) → T30 retro (3) → T31
Ceremony 1 (4) → T31 retro (5) → T32 Ceremony 1 (6) → T32 retro (7) → T33
Ceremony 1 (8) → T33 retro (9) → T34 Ceremony 1 (10) → T34 retro (11) → T35
Ceremony 1 (12) → T35 retro (13) → T36 Ceremony 1 (14) → this retro
(15) — fifteen consecutive live checks finding the identical 7-issue set
unchanged.

**D1's consecutive-sprint-silence counter: confirmed at twenty-three, not
incremented a second time within this sprint.** T36's own Ceremony 1
already computed this as **twenty-three consecutive sprints (T14 through
T36)** when the sprint opened with #144 still carrying only its original
T14.3 comment. This retro re-checked #144 live (§3): still exactly one
comment, unchanged. Per the established per-*sprint*, not per-*ceremony*,
counting convention — verified by re-reading how T31's through T35's own
retros handled this identical continuation question (all confirmed rather
than re-incremented within the same sprint) — this retro **confirms the
count holds at twenty-three**, rather than incrementing it a second time
within the same sprint. It becomes **twenty-four** only if T37 opens with
#144 still uncommented — that is T37's Ceremony 1's own count to take, not
this retro's to pre-empt.

**Stated so a future ceremony does not have to re-derive which counter moves
when:** the backlog-composition counter counts *live checks that found the
set unchanged* (a per-ceremony shape, now fifteen); D1's silence counter
counts *sprints elapsed with no comment* (a per-sprint shape, held at
twenty-three since T36's own Ceremony 1 already took this sprint's
increment).

## 7. The stale repo-metadata artifact — still present, still functionally inert

**PE**, re-checked because it has been part of every recent ceremony's own
live API calls, not chased further per the standing "don't chase it unless
it's breaking something" instruction (T35 retro recommendation 7).

Every `pull_request_read` call made this retro (#252, #253, #254) still
returns `head.repo.full_name`/`base.repo.full_name` as
`nhuthuynh/pickleball-platform`, with `description` still the stale "A
Vinyl-Trading enterprise app built with Node.js + TypeScript using Domain-
Driven Design" line — unchanged from what T36's own plan reported. The
local checkout's `git remote -v`, `git fetch`, `git log`, and `git status`
all ran clean against `nhuthuynh/white-label` during this retro's own
verification work (§1). **Confirmed still present, confirmed still
functionally inert.** Not pursued further.

## 8. `HANDOFF.md`'s Docs-index row and Task-backlog entry — deliberately NOT touched by this retro

**PO**, per the now-settled convention (T27's own retro, restated at
T30–T35): a retro PR cannot cite its own merge PR number or `merged_at`
before it exists, so it is structurally incapable of correcting its own
Docs-index row or writing its Task-backlog Outcome paragraph correctly.
This retro does not repeat the T28/T29 mistake.

**What this retro does instead:** leaves `HANDOFF.md`'s T36 Docs-index row
and Task-backlog entry untouched, and supplies **the agreed honest-form
sentence** below for **T37's Ceremony 1** to carry into `HANDOFF.md` as its
first job — the same mechanism that correctly produced every prior sprint's
narrative entry from T27 onward.

---

## The sprint goal, scored

> *"Confirm, rather than assume, that the backlog remains exactly as
> blocked as T35's retro left it, correct the two structural bookkeeping
> items T35's retro deliberately deferred (its own `HANDOFF.md` row and
> Task-backlog narrative), and continue the two running counters on their
> established per-ceremony/per-sprint schedules."*

**Every clause holds, independently re-verified rather than taken from the
plan's own account.** All 7 open issues were re-verified live and found
unchanged (§2). The `golang-migrate`/`goose` classification remains settled
(§2). D1 and D2 remain formally open, checked directly against both ADR
files and #144's comment (§3). D2 was correctly not exercised this sprint,
stated plainly (§4). `HANDOFF.md`'s T35 row correction, performed by T36's
own Ceremony 1, is verified accurate against freshly re-fetched PR data
(§5). The post-T29 backlog-composition counter increments to fifteen; D1's
silence counter holds at twenty-three (§6). `HANDOFF.md`'s own T36
row/narrative correction is deliberately deferred to T37's Ceremony 1, per
the settled convention (§8).

**The agreed honest sentence, which T37's Ceremony 1 should carry into
`HANDOFF.md`'s Task-backlog entry and Docs-index row** (`sprint-process.md`
Ceremony 1 item 3 requires the retro's form, not a stronger one):

> T36 shipped zero tickets, the fifteenth 0-ticket sprint in this
> project's history by total count and the seventh sprint of the fresh
> consecutive run (T30, T31, T32, T33, T34, T35, T36) since T28 broke the
> T20–T27 streak, plus the T35 §A0 bookkeeping corrections T35's own retro
> deliberately left undone. This retro independently re-verified — not
> trusted — every load-bearing claim: the merged-fix sweep's live
> `totalCount: 7` matches T36 plan's own count exactly, arithmetically
> reconciled (`7 − 0 + 0 = 7`); all 7 open issues' blockers were re-checked
> live, down to full bodies, and every one is unchanged — #124, #126, #130
> need Product Owner input the team cannot supply unilaterally; #144 and
> #149 are blocked on D1; #145 needs a real, non-uuid IdP `sub` claim this
> environment cannot produce; #134 needs real assistive-technology hardware
> this environment does not have. DECISION D2 was correctly not exercised
> this sprint — zero tickets means zero PRs beyond the planning doc,
> landing in the structurally weaker "no PR existed" shape. Neither D1 nor
> D2 was answered mid-sprint as a formal ADR decision, both ADR files' `##
> Status` sections and git history read directly, and #144's single T14.3
> comment re-fetched and confirmed unchanged. `HANDOFF.md`'s T35
> Docs-index row and Task-backlog narrative correction, performed by T36's
> own Ceremony 1, was independently re-verified against freshly re-fetched
> `pull_request_read` data on #252 and #253 and found accurate. The
> post-T29 backlog-composition counter increments to **fifteen** (T29
> retro, T30 Ceremony 1, T30 retro, T31 Ceremony 1, T31 retro, T32
> Ceremony 1, T32 retro, T33 Ceremony 1, T33 retro, T34 Ceremony 1, T34
> retro, T35 Ceremony 1, T35 retro, T36 Ceremony 1, this retro — fifteen
> consecutive live checks finding the identical 7-issue set unchanged);
> D1's consecutive-sprint-silence counter holds at **twenty-three** (T14
> through T36, confirmed rather than incremented a second time within this
> sprint — it becomes twenty-four only if T37 opens with #144 still
> uncommented). A stale GitHub repo-metadata artifact (API-reported
> `full_name`/`description` mismatch) was re-checked and is still present
> but still confirmed functionally inert — local git operations against
> `nhuthuynh/white-label` ran clean throughout. No incident-grade finding
> this sprint. `HANDOFF.md`'s T36 row and Task-backlog narrative are left
> for **T37's Ceremony 1** to correct, per the convention T27's own retro
> and T30's–T35's own retros/plans all already state.

---

## Recommendations for T37's Ceremony 1 and 2

1. **Re-run the merged-fix sweep as the authoritative moment**, per the
   standing rule — re-verify the open count (7) from the live API rather
   than trusting this retro's table (§1).
2. **Correct `HANDOFF.md`'s T36 Docs-index row and insert the Task-backlog
   Outcome/blockquote using this retro's agreed sentence above**, per the
   ordinary convention (§8) — fetching this retro's own PR number live via
   `list_pull_requests`/`pull_request_read`, not treating it as "already
   known" from context.
3. **Continue the post-T29 backlog-composition counter**, incrementing it
   to sixteen if T37 Ceremony 1's own live check again finds the identical
   7-issue set unchanged (§6).
4. **If a real ticket PR is dispatched in T37, D2's interim rule finally
   has something to be exercised against again** — score whichever of D2's
   named shapes (or a new one) it lands in.
5. **The soft observation from T30 retro §3a (who audits that a dispatched
   wave's text actually named its isolation mechanism) remains open and
   untested** — T36 had zero tickets and zero waves, same as T30–T35.
   Resolve it by example the first time a real multi-implementer wave is
   dispatched, not pre-emptively.
6. **A sixteenth 0-ticket sprint arriving with no new fact is not, by
   itself, grounds to reopen the "is a 0-ticket sprint healthy" question**
   — T23 retro's finding 7 closed this question, and neither of T21
   retro's two named reopening conditions fired this sprint either
   (checked live: the 7-issue set and every blocker's substance are
   unchanged, §2; `totalCount: 7`, unchanged, §1). Per the user's own
   standing instruction (carried in this ceremony's task), the sprint loop
   continues re-deferring D1/D2 without re-raising the escalation question
   absent a materially different blocker profile or the backlog running
   dry — neither of which has happened.
7. **The stale repo-metadata artifact (§7) does not need its own
   investigation** unless a future ceremony's git operations actually fail
   or the "repository moved" push message actually surfaces a functional
   symptom — it has been confirmed present-but-inert across multiple
   ceremonies now.

## Sprint-level Definition of Done — scored against what T36's own plan asked

1. **No ticket to merge; sprint goal met as stated?** **Yes** —
   confirm-and-report on the backlog plus the named §A0 bookkeeping
   correction, independently re-verified above (§2, §5).
2. **The merged-fix issue sweep run and reported with its count (7,
   reconciled arithmetically)?** **Yes** — §1; T37's Ceremony 1 remains
   authoritative.
3. **Scoring owed at the retro:**
   - **(a)** Did this ceremony's own claim — that all 7 open issues remain
     genuinely blocked, re-verified live — hold for the whole sprint (a
     live re-check at retro time, not a re-read of the plan)? **Yes** — §2.
   - **(b)** Is the `golang-migrate`/`goose` roadmap-debt classification
     still correctly unticketed at retro time? **Yes** — §2.
   - **(c)** Did D1 or D2 get answered mid-sprint, as a formal ADR
     decision? **No, neither** — §3.
   - **(d)** Did either of T21 retro's two named reopening conditions fire
     mid-sprint (checked live)? **No, neither** — §1, §2.
4. **Not scoreable by T36 and deliberately not pre-empted:** D1 and D2
   remain the user's own decisions to make, on whatever timeline the user
   chooses.
5. **Retro in `docs/process/t36-retro.md`** (this document), indexed by a
   `## T36 sprint retro` stub in `docs/LESSONS.md`. Per §8 above,
   `HANDOFF.md`'s Docs-index row and Task-backlog entry are **deliberately
   not corrected by this PR** — left to T37's Ceremony 1.

Retro complete. Issue-tracker actions this ceremony: **none** — the sweep
was clean (§1), no issue needed a label correction, and no correction to an
earlier ceremony's claim on any issue was found necessary. Open count at
ceremony start: **7**. Open count now: **7** (unchanged). No incident rises
to a `docs/LESSONS.md` entry this sprint — every finding above either
confirms a claim held under independent re-verification or confirms a
bookkeeping correction was accurate on re-check — so only the standard
index stub is added.

Toolchain re-verified directly against the unmodified tree this retro:
`make generate` (buf + sqlc, clean, no working-tree diff since
`internal/gen/**` is gitignored) then `make test-domain` — 12/12 packages
green, matching T36 plan's own count exactly (no code has landed since
T36's own Ceremony 1 to change either number).

Per §8 above: **`HANDOFF.md`'s T36 row and Task-backlog entry are not
touched by this PR.** T37's Ceremony 1 corrects them, including the
honest-form sentence above, as its first job.
