# T30 Sprint Retro

Ceremony 3 per `docs/process/sprint-process.md`, six-role team (briefs:
`docs/agent-operating-handbook.md` Part B), held against
`docs/process/t30-sprint-plan.md` (§A0–§A9 and the Ceremony 2 goal/DoD),
`docs/process/t27-retro.md` as the closest 0-ticket-sprint structure/tone/rigor
precedent, `docs/process/t28-retro.md`/`docs/process/t29-retro.md` as the
precedent for scoring real (non-ticket) work product within a retro,
`HANDOFF.md`, ADR-0015, ADR-0016, `docs/process/sprint-process.md` itself (the
two sections this ceremony's own Ceremony 1 added), and the live
issue/PR/commit history on `nhuthuynh/white-label` (GitHub-side name
`pickleball-platform`) — one PR this sprint (#242, the Ceremony 1/2 doc),
zero new issues, zero comments posted.

**Every factual claim below was re-derived against the live repository and
GitHub API this ceremony, not read from T30's plan's own account of itself**
(CLAUDE.md rule 10; this project's standing convention). Concretely, this
retro: re-ran `list_issues(state: OPEN)` fresh rather than trusting the
plan's own count; re-fetched all 7 open issues' full bodies via `issue_read`,
not just their `updated_at`/comment-count fields; re-fetched #144's comment
body directly; re-read both ADR-0015's and ADR-0016's `## Status` sections
and their git history directly; read both new `sprint-process.md` sections in
full and checked them against `docs/process/t29-retro.md` §8/§9's own text
rather than trusting the plan's paraphrase of either; and re-ran
`make test-domain` and `go run ./cmd/gatecoverage` against the unmodified
tip in an isolated worktree.

**What this retro found, in one sentence, so the findings do not have to be
read before the headline is known:** the live sweep matches the plan's own
count exactly (7 open issues, unchanged), all 7 issues' blockers are
independently re-confirmed unchanged down to their full bodies, both new
`sprint-process.md` sections hold up under real editorial scrutiny — well
scoped, faithful to the incidents they cite, and non-duplicative of existing
sections — with one soft observation recorded for a future sprint rather than
a defect, D2 was correctly not exercised this sprint (zero PRs beyond the
planning doc), D1 and D2 both remain formally unanswered, the post-T29
backlog-composition counter increments to **three** and D1's
consecutive-sprint-silence counter holds at **seventeen** (confirmed, not
incremented a second time within this sprint), and — per a repeated mistake
this retro found while checking its own footing (T28's and T29's retros both
mistakenly claimed to correct their own `HANDOFF.md` Docs-index row, which is
structurally impossible before their own PR merges, and both had to be
corrected by the following sprint's Ceremony 1) — this retro deliberately
does **not** repeat that mistake a third time.

---

## 1. The merged-fix issue sweep — clean, re-derived live, matching the
plan's own count exactly

**PO.** Per `sprint-process.md`'s DoD, this retro is sweep moment 1; T31's
Ceremony 1 remains the authoritative moment regardless of this result.

**Step 1 — list the open issues, live, at this retro's start.**
`list_issues(state: OPEN)` → **`totalCount: 7`**: #124, #126, #130, #134,
#144, #145, #149 — identical to T30 plan's own §A1 sweep and to T29 retro's
own closing count.

**Step 2 — reconcile arithmetically, before reading any issue
individually.** `open_at_end_of_T30's_own_Ceremony_1 (7) − closed_during_T30
(0) + opened_during_T30 (0) = 7`. `list_pull_requests(state: closed, base:
claude/go-backend-pickleball-7up34j, sort: updated, direction: desc)` shows
the most recent entries as **#242** (this sprint's own Ceremony 1/2 doc,
`merged_at: 2026-08-16T15:53:47Z`) → #241 (T29's retro) → #240 (T29.2) → #239
(T29.1) → #238 (T29's Ceremony 1/2 doc) — nothing merged after #242. `git
log -1` at this retro's branch cut shows `ec3b986` as the tip with no
descendants, matching `origin/claude/go-backend-pickleball-7up34j` exactly
after `git fetch`, and `git status` was clean before this ceremony's branch
was cut. **Matches the live `totalCount: 7` exactly.**

**Step 3 — cross-reference merged PRs against the open list.** Exactly one
PR merged this sprint (#242, the plan document itself), naming no issue
number as a claimed closure in its title. **Zero PRs to cross-reference
against the open list** — the correct, trivial shape for a 0-ticket sprint.

**Sweep result: clean.** **T31's Ceremony 1 still re-runs this sweep in
full**, per the standing rule that a prior ceremony's clean result does not
discharge the next one.

## 2. All 7 issues' blockers, re-verified live — full bodies re-read, not
just `updated_at`/comment counts

**QA and BA**, per the task's own explicit instruction not to trust the
plan's table without re-checking `updated_at`/comment counts directly, and
to go further than that and re-read each issue's full body.

| Issue | `updated_at`/comments at T30 plan §A3 | `updated_at`/comments now (live, this retro) | Changed? |
|---|---|---|---|
| #149 | `2026-08-16T14:20:31Z`, 4 comments | `2026-08-16T14:20:31Z`, 4 comments | No |
| #145 | `2026-08-15T05:01:29Z`, 1 comment | `2026-08-15T05:01:29Z`, 1 comment | No |
| #144 (D1) | `2026-08-15T07:01:03Z`, 1 comment | `2026-08-15T07:01:03Z`, 1 comment | No |
| #134 | `2026-08-14T16:37:49Z`, 0 comments | `2026-08-14T16:37:49Z`, 0 comments | No |
| #130 | `2026-08-14T16:30:25Z`, 0 comments | `2026-08-14T16:30:25Z`, 0 comments | No |
| #126 | `2026-08-14T16:12:26Z`, 0 comments | `2026-08-14T16:12:26Z`, 0 comments | No |
| #124 | `2026-08-15T16:25:34Z`, 1 comment | `2026-08-15T16:25:34Z`, 1 comment | No |

**Every field matches T30 plan's own live-fetched table exactly,
byte-for-byte — zero blockers changed state, zero new comments on any of the
7, and every timestamp predates PR #242's own merge (`15:53:47Z`).**

**Every issue's full body independently re-read this retro
(`issue_read(get)`), not just its cached table row — the check the task
brief specifically asked for beyond what T30's own plan performed:**

- **#149** (`role:principal-engineer`, `type:bug`, `context:payments`, 4
  comments): body unchanged — Payments still resolves the *actor* to a
  verified `User.ID` but still accepts caller-supplied *ownership facts*
  (`booking_host_id`, `game_host_id`, admin-assignment lists,
  `entrant_player_id`) with no port to resolve them against; blocked on D1
  (the Booking-side fact) plus the still-unbuilt Game-Admin/
  Competition-Admin durable store the issue's own text names as a
  sub-problem.
- **#145** (`role:principal-engineer`, `type:chore`, 1 comment): body
  unchanged — pre-existing UUID rows vs. a real, non-uuid IdP `sub` claim;
  needs a real IdP tenant this environment cannot provision; both failure
  modes (`AddCourt`/`AddCameraLink`/`AttestCameraConsent` for
  pre-existing Facilities, `RequestRecurringHire`'s `EnsureClubRole`) still
  fail closed, so this remains a forward-migration gap, not an active
  vulnerability.
- **#144** (`D1`, 1 comment): body of the T14.3 escalation comment re-read
  in full — still zero authorization on `CancelBooking`/`CreateBooking`,
  blocked on the Product Owner's answer to D1 (what a public quote-and-book
  Booking is owned by, and whether the unauthenticated flow should remain
  possible at all).
- **#134** (`role:ux-ui-designer`, `type:chore`, 0 comments): body
  unchanged — a real screen-reader session (NVDA/JAWS/VoiceOver) and
  rendered-in-browser measurements (focus-indicator contrast, 200–400% zoom
  reflow, literal keyboard traversal, target-size measurement) this
  environment's lack of assistive-technology hardware cannot supply; T30
  shipped no UI change that would give a targeted pass anything new to
  cover.
- **#130** (`role:principal-engineer`, `type:story`, 0 comments): body
  unchanged — still carries its own stated open product question, whether
  reversing a `no_show_fee` Payment is the product behavior actually wanted
  and what (if anything) the refund should project onto
  `Registration.PaymentStatus`.
- **#126** (`role:product-owner`, `type:story`, 0 comments): body unchanged
  — still needs Product Owner input on whether the price is per-Game,
  per-Registration, or per-head-including-guests before any code, unchanged
  since T8.10.
- **#124** (`role:product-owner`, `type:story`, 1 comment): body unchanged
  — still needs Product Owner input on cascade semantics (court release,
  refund automaticity via T12.3's `RefundPayment`, waitlist handling)
  before `Game.Cancel()` can cascade to its Bookings/Registrations.

**Migration-tooling classification (`golang-migrate`/`goose`), re-checked
directly rather than inherited.** `grep -n "golang-migrate\|goose"
HANDOFF.md` still returns exactly the single Cross-cutting line every prior
ceremony has quoted; `ls docs/adr/` still ends at `0017`; `ls
db/migrations/` still ends at `0026`. Nothing changed, because nothing
merged this sprint capable of changing it (§1). Still correctly unticketed —
the fourteenth consecutive prior ceremony's own ruling standing unchallenged
by a new fact.

**Point, scored: yes, all 7 issues' blockers held for the whole sprint,
independently re-verified live down to their full bodies, not assumed from
silence or trusted from the plan's own account.**

## 3. Scoring the two new `sprint-process.md` sections — real editorial
scrutiny, not a rubric check

**PE and PO, jointly** — both sections read in full this retro
(`docs/process/sprint-process.md`, "### Dispatch isolation" under Ceremony
2, and "### PR-body self-verification" under Execution), cross-checked
against `docs/process/t29-retro.md` §8 and §9 respectively rather than
trusted from T30 plan §A9's own paraphrase.

### 3a. "Dispatch isolation"

**Fidelity to the incident it cites.** Re-reading T29 retro §8 side by side
with the new section confirms the section does not overstate or
misrepresent what §8 found: T9's own retro (`docs/process/t9-retro.md`
finding 1) is quoted correctly (*"dispatch isolation becomes an explicit
Ceremony 2 checklist item"*), the thirteen-sprint erosion claim is backed by
the same `grep -ni "isolat" docs/process/sprint-process.md` (zero matches
before this section existed) and `grep -ni "isolat"
docs/process/t29-sprint-plan.md` (zero matches) that §8 itself ran, and the
T29.1/T29.2 near-miss is scored identically — *"near-miss, not a repeat
T9-grade incident"* — in both documents, word for word. Re-run this retro:
`grep -ni "isolat" docs/process/sprint-process.md` now returns multiple
matches, confirming the section actually landed rather than being described
as landed.

**Scoping.** Well-scoped on three axes: it states the rule (every
implementer in a multi-implementer wave works in its own isolated
`git worktree`/equivalent), it states what Ceremony 2 must concretely do
(name the isolation mechanism in the wave's own text), and it states what it
does **not** require (a solo-implementer or genuinely-sequential wave has no
hazard to isolate against) — the same three-part shape "the
dependency-completeness check" and "the same-wave shared-interface
verification rule" use, so it is structurally consistent with this
document's existing conventions rather than a one-off.

**Overlap/conflict check with "Same-wave shared-interface verification" —
the specific concern the task asked to check directly.** No conflict found,
and the section is explicit about why: it states in its own text (point 2
under "What Ceremony 2 must do") that *"isolation and the same-wave
shared-interface verification rule… are independent checks and both apply
when both conditions hold… they guard different hazards — a live collision
in a shared working tree during implementation, versus a semantic conflict
in the merged tree discovered at review time."* This is correct as stated:
dispatch isolation prevents two implementers' *uncommitted, in-progress*
work from corrupting each other's working directories; same-wave
shared-interface verification catches a *textually clean but semantically
broken* merged tree after both branches are pushed. A wave could violate
either without violating the other (two implementers in one unisolated
checkout touching disjoint files never collide in a working tree sense but
could still hit the interface-widening hazard; two implementers correctly
worktree-isolated could still both touch one interface's blast radius from
different files). The two rules are genuinely orthogonal, not two names for
the same check.

**One soft observation, not scored as a defect.** The section names what
Ceremony 2 must do (name the isolation mechanism in the wave's own text) and
states that a wave failing to do so is *"incomplete per this section,"* but
— unlike "Same-wave shared-interface verification," which ties its rule
directly to a review-time obligation (a specific party's PR review must
reconstruct the merged tree) — it does not yet name *who* checks that a
dispatched wave's text actually named an isolation mechanism, or what
happens if it did not. This is not a defect requiring a fix: T30 has zero
tickets and no wave to test the section against, so there is nothing yet to
observe the section either catching or missing. It is recorded here as a
question for **the first sprint that actually dispatches a multi-implementer
wave** to resolve by example (does the retro check it, per T30 plan's own
"scoring owed at the retro" convention for other sections? does Ceremony 2
itself self-certify?) rather than something this retro can score one way or
the other on a sprint with no wave.

**Scored: sound.** Faithful to the incident it cites, correctly scoped,
genuinely non-duplicative of "Same-wave shared-interface verification" —
confirmed by reading the section's own stated reasoning rather than taking
"no conflict" on faith.

### 3b. "PR-body self-verification"

**Fidelity to the incident it cites.** Re-reading T29 retro §9 side by side
with the new section confirms the same fit: the incident (PR #240's first
review finding an essentially empty body despite the implementer's own
final chat report describing substantial content), the "already caught
cleanly by existing review process, not a new gap" framing, and the exact
recommended mechanic (*"one fresh `pull_request_read(get)`… confirming
`body` is non-empty and matches what it intended to post… a single read
call, not a new process stage"*) all carry over essentially verbatim from
§9's own text into the new section. No overstatement found — the new
section does not claim this closes a gap that existing process failed to
catch; it explicitly states the opposite (*"this is not a new failure class
the project had no guard against"*), matching §9's own scoring precisely.

**Placement, verified rather than assumed from the plan's own account.**
T30 plan §A9 claims the section is placed *"immediately after the per-ticket
DoD numbered list and before 'Same-wave shared-interface verification.'"*
Confirmed directly against the file's actual section order: `## Execution`
→ its five-item numbered DoD list → `### PR-body self-verification` → `###
Same-wave shared-interface verification` → `### Recovering an interrupted
session's work`. The claim is accurate.

**Scoping.** Deliberately narrow, and states its own narrowness explicitly
(*"not a general instruction to re-verify every field of every PR"*) —
targeted at exactly the one failure class observed (an agent's own belief
about what it wrote and the API's actual record of it silently diverging),
which is the correct size for a safeguard adopted from a single incident
rather than a pattern.

**Overlap/conflict check.** No conflict with the per-ticket DoD's existing
review step (DoD item 2, "reviewed… findings addressed") — this is a
pre-review self-check that runs *before* handoff to review, not a
substitute for it. No conflict with "Recovering an interrupted session's
work"'s own verification clause (b), which verifies a *recovered* session's
work is correct — a different concern (correctness of recovered work) from
this section's concern (the PR description accurately reflecting what was
written).

**Scored: sound.** Faithful to the incident it cites, correctly and
deliberately narrow, placed where the plan says it is placed, and creates
no overlap with any existing Execution-section safeguard.

### 3c. Overall assessment of §A9's process work

Both sections pass real editorial scrutiny: neither misrepresents the
incident it answers, neither duplicates or conflicts with an existing
section, and both are scoped to the narrow failure class each was written
for rather than over-generalized. The one soft observation (3a) is a
question for the next multi-implementer wave to answer by example, not a
finding against this ceremony's work.

## 4. DECISION D2 — confirmed not exercised this sprint, stated plainly
rather than skipped

**PE and PO, jointly**, per the task's explicit instruction to state this
point rather than pass over it because there is "nothing to score."

T30 shipped zero tickets and therefore zero PRs beyond the Ceremony 1/2
planning document itself (§1). D2 asks whether a session that reviews and
merges a pull request may also author code on it — a question that can only
be engaged by a real PR under real review. With no ticket PR opened this
sprint, there was no review to score against any of D2's named shapes
("exercised, no fix needed"; T29.2's genuinely-new fourth shape; or a
reviewer directly authoring a fix, which remains unobserved on any real,
non-recovery PR). **This sprint lands in the structurally weaker "no PR
existed" shape** (T21 retro's own naming for this case) — the same shape
T20 through T27 all landed in, and distinct from T28's and T29's own real
exercises of the interim rule. Confirmed directly rather than inferred: zero
tickets (§A6 of T30's own plan, independently re-confirmed here by the
same empty ticket/wave sections), zero PRs beyond #242 (§1's own
cross-reference).

## 5. D1 and D2 remain formally unanswered — checked directly against both
ADR files and #144's comment, not assumed unchanged

**PE and PO, jointly.**

- **ADR-0015 (`docs/adr/0015-booking-ownership-for-public-bookings.md`),
  `## Status` heading (line 23), read in full this retro.** Unchanged:
  **"Escalated — awaiting product decision. This ADR decides nothing."** No
  option (a)–(d) is marked chosen. `git log --oneline -- docs/adr/0015-*.md`
  shows exactly one commit (`5828550`, its T14.3 authoring commit) — no
  commit since, including none from PR #242.
- **ADR-0016 (`docs/adr/0016-reviewer-authored-code-on-a-reviewed-pull-
  request.md`), `## Status` heading (line 22), read in full this retro.**
  Unchanged: **"Escalated — awaiting the user's decision. This ADR decides
  nothing."** Same restriction list, same wording, word for word.
  `git log --oneline -- docs/adr/0016-*.md` shows exactly one commit
  (`0aaa912`, its T15.2 authoring commit) — no commit since.
- **#144, re-fetched via `issue_read(get_comments)` this retro — the
  comment body itself, not just its count.** Exactly one comment (`id
  5301056598`, `created_at: 2026-08-15T07:01:03Z`) — T14.3's original
  escalation, full text re-read and matching every prior retro's quotation
  verbatim, including the note that this is #144's *"second deferral"* and
  that *"a third deferral without an answer is a finding, not a
  decision."* No second comment exists; `state: OPEN` confirmed by §1's own
  live `list_issues` call.

**Point, scored: neither D1 nor D2 was answered mid-sprint, as a formal ADR
decision.** Both files' git history proves nothing could have changed
either — zero commits touched either file since their original authoring
commits, and this sprint shipped zero commits capable of touching them
(only PR #242, a docs-only planning PR that does not touch either ADR file
— confirmed via `pull_request_read`-equivalent reasoning: the plan §A0's
own change list names only `HANDOFF.md` and `sprint-process.md`).

## 6. Running counters

**PO.**

**The post-T29 backlog-composition counter: increments to three.** T29
retro established this at one; T30 plan's own Ceremony 1 incremented it to
two. This retro's own live sweep (§1/§2) independently finds the identical
7-issue set (#124, #126, #130, #134, #144, #145, #149) unchanged in both
membership and blocker content — the third consecutive live check to find
it so. **Counter becomes three**, following the same per-ceremony-increment
shape the prior (retired) counter used.

**D1's consecutive-sprint-silence counter: confirmed at seventeen, not
incremented a second time within this sprint.** T30's own Ceremony 1 already
computed this as **seventeen consecutive sprints (T14 through T30)** when
the sprint opened with #144 still carrying only its original T14.3 comment.
This retro re-checked #144 live (§5): still exactly one comment, unchanged.
Per the established per-*sprint*, not per-*ceremony*, counting convention —
verified by checking how T28's and T29's own retros handled the identical
continuation question (T28 retro: *"This retro is Ceremony 3 of the *same*
sprint T28, not a new sprint boundary… confirms the count at fifteen,
unchanged"*; T29 retro: *"confirmed at sixteen, not incremented a second
time within this sprint"*) — this retro **confirms the count holds at
seventeen**, rather than incrementing it a second time within the same
sprint. It becomes **eighteen** only if T31 opens with #144 still
uncommented — that is T31's Ceremony 1's own count to take, not this
retro's to pre-empt.

**Stated so a future ceremony does not have to re-derive which counter
moves when, per the same distinction T27 retro's finding 6 and every
retro since has kept separate:** the backlog-composition counter counts
*live checks that found the set unchanged* (a per-ceremony shape, now
three: T29 retro, T30 Ceremony 1, this retro); D1's silence counter counts
*sprints elapsed with no comment* (a per-sprint shape, held at seventeen
since T30's Ceremony 1 already took this sprint's increment).

## 7. `HANDOFF.md`'s Docs-index row and Task-backlog entry — deliberately
NOT touched by this retro, and why

**PO**, per the task's own instruction to update these — reconciled below
against a real, repeated mistake this retro found while checking its own
footing, rather than followed mechanically.

**The mistake, found and traced.** Both T28's and T29's own retros closed
with a paragraph claiming *"`HANDOFF.md`'s T2X row is corrected by this same
PR."* Both claims were wrong in the same way, caught by the *next* sprint's
own Ceremony 1 in both cases: `docs/process/t29-sprint-plan.md` §A0 found
that T28 retro's PR (#236) *"claimed… but a PR structurally cannot cite its
own merge PR number before it exists… and the row it actually shipped left
the number blank"*; `docs/process/t30-sprint-plan.md` §A0 found the
identical mistake in T29 retro's own claim about PR #241. This is not one
mistake, corrected — it is the **same** mistake, made twice in a row by two
different retros, each one caught only because the following sprint's
Ceremony 1 happened to re-verify rather than trust it. `sprint-process.md`'s
own Ceremony 1 section states the reason plainly: *"the row must cite the
retro's own merge PR number, which does not exist until that PR merges…
a retro PR is structurally incapable of writing its own row correctly."*
T27's own retro got this right the first time (*"`HANDOFF.md`'s T27 row is
not touched by this PR. T28's Ceremony 1 corrects it… as its first job"*),
and T30's own plan (§B, Sprint-level DoD item 5) already restates the
correct convention for this exact sprint: *"noting that T31's Ceremony 1,
not the retro, corrects T30's Docs-index row (the ordinary convention)."*

**What this retro does instead.** Per the correct convention — restated by
T27's own retro, by `sprint-process.md`'s Ceremony 1 section, and by T30's
own plan — this retro does **not** edit `HANDOFF.md`'s T30 Docs-index row
or insert the Task-backlog Outcome/blockquote paragraph itself. Both edits
require this PR's own merge PR number and `merged_at` timestamp, neither of
which is knowable before this PR merges — attempting either here would be a
third instance of the identical, now twice-demonstrated mistake. Instead,
per `sprint-process.md` Ceremony 1 item 3 (*"the previous sprint's narrative
entry in the Task backlog, stat[es] its outcome in the form its own retro
agreed"*), this retro supplies **the agreed honest-form sentence** below,
for **T31's Ceremony 1** to carry into `HANDOFF.md`'s Task-backlog entry and
Docs-index row as its first job — exactly the mechanism that correctly
produced T28's and T29's own narrative entries (`docs/process/
t28-sprint-plan.md:35` and `docs/process/t29-sprint-plan.md` §A0 both show
the *next* sprint's Ceremony 1, not the retro, performing this transcription).

---

## The sprint goal, scored

> *"Confirm, rather than assume, that the backlog #164/#237's closure left
> behind is exactly as blocked as it was before T28/T29 took real scope —
> and land the two pieces of real process work T29's retro identified as
> genuinely owed, rather than treat a 0-ticket sprint as a pure
> confirm-and-report exercise."*

**Every clause holds, independently re-verified rather than taken from the
plan's own account.** All 7 open issues were re-verified live, full bodies
included, and found unchanged (§2). `HANDOFF.md`'s Cross-cutting section and
the `golang-migrate`/`goose` classification remain settled (§2). Both new
`sprint-process.md` sections hold up under real editorial scrutiny — sound,
faithful to their incidents, non-duplicative (§3). D2 was correctly not
exercised this sprint, stated plainly (§4). D1 and D2 remain formally open,
checked directly against both ADR files and #144's comment (§5). The
post-T29 backlog-composition counter increments to three; D1's silence
counter holds at seventeen (§6). `HANDOFF.md`'s own row/narrative correction
is deliberately deferred to T31's Ceremony 1, avoiding a mistake this
project's own last two retros made (§7).

**The agreed honest sentence, which T31's Ceremony 1 should carry into
`HANDOFF.md`'s Task-backlog entry and Docs-index row** (`sprint-process.md`
Ceremony 1 item 3 requires the retro's form, not a stronger one):

> T30 shipped zero tickets, the ninth 0-ticket sprint in this project's
> history (the first since T28 broke the T20–T27 run), plus real process
> work executing two of T29 retro's recommendations. This retro
> independently re-verified — not trusted — every load-bearing claim: the
> merged-fix sweep's live `totalCount: 7` matches T30 plan's own count
> exactly, arithmetically reconciled (`7 − 0 + 0 = 7`); all 7 open issues'
> blockers were re-checked live down to their full bodies, not just cached
> `updated_at`/comment-count fields, and every one is unchanged — #124,
> #126, #130 need Product Owner input the team cannot supply unilaterally;
> #144 and #149 are blocked on D1; #145 needs a real, non-uuid IdP `sub`
> claim this environment cannot produce; #134 needs real assistive-
> technology hardware this environment does not have. The two new
> `sprint-process.md` sections — "Dispatch isolation" (executing T29 retro
> recommendation 2) and "PR-body self-verification" (executing recommendation
> 3) — were read in full and checked against T29 retro §8/§9's own text
> rather than trusted from the plan's paraphrase: both are faithful to the
> incidents they cite, correctly and narrowly scoped, and non-duplicative of
> existing sections (in particular, "Dispatch isolation" is confirmed
> genuinely orthogonal to "Same-wave shared-interface verification" rather
> than a restatement of it), with one soft observation recorded for the next
> multi-implementer wave to resolve by example rather than a defect found
> in either. DECISION D2 was correctly not exercised this sprint — zero
> tickets means zero PRs beyond the planning doc, landing in the
> structurally weaker "no PR existed" shape rather than either of D2's
> exercised shapes. Neither D1 nor D2 was answered mid-sprint as a formal
> ADR decision, both ADR files' `## Status` sections and git history read
> directly, and #144's single T14.3 comment re-fetched and confirmed
> unchanged. The post-T29 backlog-composition counter increments to
> **three** (T29 retro, T30 Ceremony 1, this retro — three consecutive live
> checks finding the identical 7-issue set unchanged); D1's
> consecutive-sprint-silence counter holds at **seventeen** (T14 through
> T30, confirmed rather than incremented a second time within this sprint —
> it becomes eighteen only if T31 opens with #144 still uncommented). This
> retro also found and named a real, twice-repeated process mistake: T28's
> and T29's own retros both incorrectly claimed to correct their own
> `HANDOFF.md` Docs-index row in their own PR, which is structurally
> impossible before that PR's own merge PR number and `merged_at` are
> known — both mistakes were caught only by the *following* sprint's
> Ceremony 1. This retro does not repeat that mistake: `HANDOFF.md`'s T30
> row and Task-backlog narrative are left for **T31's Ceremony 1** to
> correct, per the actually-correct convention T27's own retro and T30's own
> plan both already state.

---

## Recommendations for T31's Ceremony 1 and 2

1. **Re-run the merged-fix sweep as the authoritative moment**, per the
   standing rule — re-verify the open count (7) from the live API rather
   than trusting this retro's table (§1).
2. **Correct `HANDOFF.md`'s T30 Docs-index row and insert the Task-backlog
   Outcome/blockquote using this retro's agreed sentence above**, per the
   ordinary convention (§7) — and, in doing so, do not repeat the mistake
   named in §7: do not let this retro's own PR number get treated as
   "already known" from context; fetch it live via `list_pull_requests`/
   `pull_request_read` the same way T29's and T30's own Ceremony 1s did
   when correcting the prior sprint's row.
3. **Resolve the soft observation in §3a** (who checks that a
   multi-implementer wave's text actually named an isolation mechanism, and
   what happens if it did not) the first time a real multi-implementer wave
   is dispatched — not by amending the section pre-emptively on no live
   example to test it against.
4. **Continue the post-T29 backlog-composition counter**, incrementing it
   to four if T31 Ceremony 1's own live check again finds the identical
   7-issue set unchanged (§6).
5. **If a real ticket PR is dispatched in T31, D2's interim rule finally has
   something to be exercised against again** — score whichever of D2's
   named shapes (or a new one) it lands in, per T29 retro recommendation 6's
   standing instruction to score a recurrence against retro-established
   definitions rather than re-deriving one from scratch.
6. **If a tenth consecutive 0-ticket sprint arrives with no new fact, do not
   reopen the "is a 0-ticket sprint healthy" question by default** — T23
   retro's finding 7 already closed this question, its two named reopening
   conditions have fired zero times across every sprint since, and this
   retro found nothing that would reopen it either.

## Sprint-level Definition of Done — scored against what T30's own plan asked

Per `docs/process/t30-sprint-plan.md`'s "Sprint-level Definition of Done,"
the items owed at this retro, restated here with their answers:

1. **No ticket to merge; sprint goal met as stated?** **Yes** —
   confirm-and-report on the backlog plus the two named process fixes, not
   build-and-ship, and both halves independently re-verified above (§2,
   §3).
2. **The merged-fix issue sweep run and reported with its count (7,
   reconciled arithmetically)?** **Yes** — §1; T31's Ceremony 1 remains
   authoritative.
3. **Scoring owed at the retro:**
   - **(a)** Did the "all 7 open issues remain genuinely blocked" claim
     hold for the whole sprint, checked live at retro time rather than
     re-read from the plan? **Yes** — §2, full bodies re-read, not just
     cached fields.
   - **(b)** Is the `golang-migrate`/`goose` roadmap-debt classification
     still correctly unticketed at retro time? **Yes** — §2.
   - **(c)** Did D1 or D2 get answered mid-sprint, as a formal ADR
     decision? **No, neither** — §5.
   - **(d)** Do the two new `sprint-process.md` sections read as the plan
     describes them, independently re-read rather than assumed? **Yes,
     both** — §3, with one soft observation recorded, not a defect.
4. **Not scoreable by T30 and deliberately not pre-empted:** D1 and D2
   remain the user's own decisions to make, on whatever timeline the user
   chooses.
5. **Retro in `docs/process/t30-retro.md`** (this document), indexed by a
   `## T30 sprint retro` stub in `docs/LESSONS.md`. Per §7 above,
   `HANDOFF.md`'s Docs-index row and Task-backlog entry are **deliberately
   not corrected by this PR** — left to T31's Ceremony 1, the convention
   T30's own plan already named and which this retro found good reason
   (§7's repeated-mistake trace) not to override.

Retro complete. Issue-tracker actions this ceremony: **none** — the sweep
was clean (§1), no issue needed a label correction, and no correction to an
earlier ceremony's claim on any issue was found necessary. Open count at
ceremony start: **7**. Open count now: **7** (unchanged). No incident rises
to a `docs/LESSONS.md` entry this sprint — every finding above either
confirms a claim held under independent re-verification or names a process
observation already scored as non-blocking (§3a) or already self-corrected
by this retro's own footing check (§7) — so only the standard index stub is
added.

Toolchain re-verified directly against the unmodified tree this retro, in an
isolated `git worktree` detached at `ec3b986` (removed after use): `make
test-domain` — 12/12 packages green, matching T30 plan's own count exactly;
`go run ./cmd/gatecoverage` — `OK — all 45 package(s) with runnable tests are
executed by "ci-checks"`, matching T29 retro's own reported count exactly
(no code has landed since T29.2 to change either number).

Per §7 above: **`HANDOFF.md`'s T30 row and Task-backlog entry are not
touched by this PR.** T31's Ceremony 1 corrects them, including the honest-
form sentence above, as its first job — the correct convention, restated
here after tracing why the two most recent retros' own attempts to do this
themselves were both wrong.
