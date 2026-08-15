# Sprint Process — Pickleball Platform

Durable process rulebook for how work moves from backlog to merged code on
this project, from this point forward (T5 onward — T0-T4 predate this
process and are not retroactively reprocessed, see `docs/LESSONS.md`).

Board of record: **split by how long the item lives** — the sprint plan
document (`docs/process/t<N>-sprint-plan.md`) for that sprint's own tickets,
**GitHub Issues** on `nhuthuynh/white-label` for anything that outlives its
sprint. See "Board of record — split by lifetime" under Ceremony 1 for the
full rule and why it is split this way; it resolves a PM-vs-BA disagreement
T11's retro (finding 6) left open, decided at T12's Ceremony 1
(`docs/process/t12-sprint-plan.md` §A7).

(Jira was considered but isn't connected to this workspace; see
`docs/reviews/00-bootstrap.md`-adjacent decision log for why. If Jira is
connected later, this doc's ticket fields map directly onto Jira fields —
nothing here is GitHub-specific in spirit.)

## Roles

Six specialist roles staff every sprint. Each has a **role brief**
(`docs/agent-operating-handbook.md` Part B — mandate, what it's adversarial
toward) and, as of this process, a **deep knowledge dossier**
(`docs/roles/<role>.md` — sourced from public engineering blogs, leveling
guides, and job descriptions at Google, Meta, X, Uber, PayPal, WhatsApp,
Telegram, and similar) that a subagent playing the role should load before
acting in that capacity:

| Role | Brief | Dossier |
|---|---|---|
| Principal Engineer (PE) | handbook B1 | `docs/roles/principal-engineer.md` |
| Product Manager (PM) | handbook B2 | *(uses handbook brief only — see note below)* |
| Senior Product Engineer (PdE) | handbook B3 | `docs/roles/product-engineer.md` |
| QA | handbook B4 | `docs/roles/qa-engineer.md` |
| Product Owner (PO) | handbook B5 | `docs/roles/product-owner.md` |
| Business Analyst (BA) | handbook B6 | `docs/roles/business-analyst.md` |
| UX/UI Designer | *(new — no prior brief)* | `docs/roles/ux-ui-designer.md` |

*Note:* the user's process request named six roles for deep dossiers
(Principal Engineer, UX/UI Designer, QA, Senior Product Engineer, Product
Owner, Business Analyst) and separately still refers to "Product Manager"
as a backlog-splitting authority. PM keeps its existing handbook brief; flag
to the user if a PM dossier turns out to be wanted too.

## Sprint = one HANDOFF.md task

Per the agreed mapping, each remaining backlog task (T5, T6, ...) is one
sprint. A sprint is not re-scoped from the spec from scratch — it inherits
the task's "Why" and "AC" from `HANDOFF.md` as its starting scope, which
backlog refinement (below) turns into tickets.

## Ceremony 1 — Backlog refinement (before the sprint starts)

**Product Manager + Principal Engineer** split the sprint's HANDOFF.md task
into tickets. PM drives scope/value framing, PE drives technical
sequencing/feasibility; both sign off before the plan is dispatched. Each
ticket gets:

- **Title**: short, imperative (e.g. "Add Game aggregate with capacity invariant")
- **Story** (INVEST-style, per `docs/roles/product-owner.md`): *As a
  `<role>`, I want `<capability>`, so that `<benefit>`.*
- **Description**: 1-3 sentences of context — why this ticket exists, what
  it depends on.
- **Instructions**: numbered steps covering:
  - **Functional requirements** — what the code must do, phrased as
    testable behavior (Given/When/Then where it helps).
  - **Non-functional requirements** — anything cross-cutting that applies
    (invariant enforcement, concurrency, security, performance,
    accessibility if UI is involved) — never left implicit.
- **Story points**: Fibonacci-like scale (1, 2, 3, 5, 8, 13) per
  `docs/roles/product-owner.md`'s estimation section — relative sizing, not
  time-boxed.
- **Role** and **type**: `role:<primary-owner-role>` and
  `type:story|bug|chore|spike`.

All of the above are recorded **in the sprint plan document**
(`docs/process/t<N>-sprint-plan.md`), which is this sprint's board of
record — not as a GitHub issue with labels. Sprint and points live only
there; role and type additionally become GitHub labels if the item is ever
filed as an issue. See the rule below.

### Correct the previous sprint's Docs-index row

**Ceremony 1 must check, and correct, the previous sprint's row in
`HANDOFF.md`'s Docs index before refining any tickets.** The row's Retro and
Reviews cells are written *before* that sprint runs, so they say "not yet
written" / "not yet opened"; nothing in the sprint that follows ever goes back
to fix them. This is a mechanical, structural gap, not an oversight by any
particular sprint — it had silently recurred twice (T11's row, corrected at
T12's Ceremony 1; T12's row, corrected at T13's) before being fixed here.

**Why this ceremony and not the retro PR** (the alternative
`docs/process/t12-retro.md` recommendation 9 offered). By established
precedent — verified across PRs #110, #122 and #153, each of which touched
exactly `docs/LESSONS.md` and its own retro doc — **a retro PR never updates
the index row that points at it.** That convention is deliberate and worth
keeping. More decisively: the row must cite the retro's own merge PR number,
which does not exist until that PR merges, so a retro PR is structurally
incapable of writing its own row correctly. Ceremony 1 runs after everything
from the previous sprint is merged, when every number is knowable — and it
already re-reads the Docs index, since CLAUDE.md's "Docs index & naming
convention" section requires reading the relevant row before starting a task.

Concretely, the Ceremony 1 output PR carries:

1. The previous sprint's row, corrected: its real retro path, and its real
   PRs in merge order **verified against each PR's `merged_at`** rather than
   assumed from numbering (this project's standing convention — PR numbers and
   merge order routinely disagree).
2. A new row for the sprint being planned.
3. The previous sprint's narrative entry in the Task backlog, stating its
   outcome **in the form its own retro agreed**, not a stronger one.

### Board of record — split by lifetime

Resolved at T12's Ceremony 1 (`docs/process/t12-sprint-plan.md` §A7),
settling the PM-vs-BA disagreement T11's retro left open as finding 6. The
split is by **how long the record has to last**, not by preference:

1. **The sprint plan document is the board of record for that sprint's
   tickets.** `docs/process/t<N>-sprint-plan.md` carries each ticket's
   Story, Description, Instructions, cross-context dependency check, story
   points, role and type. **No GitHub issue is opened per in-sprint
   ticket.** The plan is version-controlled, reviewed, richer than any issue
   body, and linked from `HANDOFF.md`'s Docs index; a parallel issue nobody
   reads is process theater. This is descriptive of what T11 and T12 already
   did, not a new practice — and it applies **retroactively**: T11's nine
   tickets (T11.1–T11.9) were never owed an issue, and
   `docs/process/t11-sprint-plan.md` is permanently their board of record.

2. **GitHub issues are the board of record for anything that outlives its
   sprint** — cross-sprint follow-ups, disclosed-but-deferred gaps, and
   escalations. These are **mandatory, not discretionary: an item deferred
   out of a sprint without an issue is a process violation, not a judgement
   call.** A sprint plan is scoped to one sprint and structurally cannot
   carry an item past it; prose in `HANDOFF.md` or a PR body is not a
   durable record. This is the half BA was right about, and it is what
   T11's retro finding 4 identified from the other direction.

3. **Label taxonomy follows the split** — see "Label taxonomy" below.

**First worked examples.** T12's Ceremony 1 opened these four under rule 2,
each a real item that had survived multiple sprints as prose only:

| Issue | Item | Carried as prose since |
|---|---|---|
| [#123](https://github.com/nhuthuynh/white-label/issues/123) | Migrate `booking` (7 positional params) and `socialplay` (5) `app.NewService` to a `ServiceOptions` struct, matching `competitions`/`payments` | T1, re-flagged T6.6 and T11.5 |
| [#124](https://github.com/nhuthuynh/white-label/issues/124) | `Game.Cancel()` flips status only — it does not cascade to the Game's court Bookings or its Registrations | T5.1 |
| [#125](https://github.com/nhuthuynh/white-label/issues/125) | No Competitions-shaped `PayableType` or payments port/adapter pair; Competitions stays cash-only | T9.6/T9.7 |
| [#126](https://github.com/nhuthuynh/white-label/issues/126) | Add a real per-Game price field, retiring T8.10's `PLACEHOLDER_REGISTRATION_FEE_CENTS` | T8.10 |

**Closing an issue is always a manual step on this project, never
automatic.** GitHub's `Closes #N` auto-close only fires for a PR merged into
the repository's **default branch** (`main`), and every PR in this project's
history merges into `claude/go-backend-pickleball-7up34j` instead — a
different branch, not the default one. That means the auto-close mechanism
structurally cannot fire here, regardless of PR body wording, and it
silently never had for any ticket from T5 through T10 until this was caught
and fixed retroactively (see issue #111 / T11.8,
`docs/process/t11-sprint-plan.md`, and `docs/process/t10-retro.md`
finding 6). Under rule 1 most tickets have no issue to close at all; for the
tickets that do — a cross-sprint follow-up filed as an issue in an earlier
sprint and resolved now — the merging party (or a follow-up chore ticket)
must close it explicitly after merge. See the Execution section's DoD step 5
for exactly how.

## Ceremony 2 — Sprint planning (start of sprint)

The full six-role team (loading their briefs + dossiers) discusses the
refined tickets together and:
1. Picks which tickets are **in scope** for this sprint (not necessarily
   all of them — PdE/PE can push back on an over-full sprint; PM/PO defend
   value; QA/BA flag missing edge cases before work starts, not after).
2. Agrees a **sprint goal** — one sentence, e.g. "A Game can be scheduled
   without breaking the no-double-booking invariant, with capacity
   enforced."
3. Do **not** manufacture consensus — a genuine disagreement (e.g. PE
   thinks a ticket is a one-way door PM wants to ship fast) is recorded in
   the sprint's kickoff note, not smoothed over.

## Execution

Each ticket is implemented TDD-first (CLAUDE.md rule 1), on its own branch,
via its own PR (CLAUDE.md rule 9 — no direct commits to the project branch).
A ticket is only "done" when:

1. Acceptance criteria (the ticket's functional + non-functional
   instructions) are met.
2. Its PR is **reviewed** (by another agent role or the user) — findings
   addressed or explicitly deferred with reasoning.
3. Its PR is **tested** — `make test-domain` green at minimum; `make test`
   green where the ticket touches adapter/infra code and the toolchain is
   available.
4. Its PR is **approved and merged** into the project branch by the user
   or an explicitly delegated gate — never self-merged by the implementing
   agent (CLAUDE.md rule 9). **Merging on a reviewing agent's recommended
   verdict, without the merging party independently re-deriving every
   finding, is a supported mode, not a shortcut around this step** — T9's
   retro (finding 6, `docs/process/t9-retro.md`) found every T9 merge
   followed a real review within minutes, every REQUEST_CHANGES verdict
   was addressed by a fix commit before merge, and none was overridden;
   the practical safety property in that mode is "the review was correct,"
   not "a human independently re-checked it." This places the obligation
   on the *reviewing* agent: lead with a bolded recommended verdict and
   put any blocking finding where it can't be missed, since the verdict
   line may be the only part read before the merge decision.
5. **If — and only if — the ticket resolves a GitHub issue, that issue is
   closed via an explicit manual step after the PR merges.** Most tickets
   have **no issue to close**: under the board-of-record rule an in-sprint
   ticket is tracked by the sprint plan document and never had an issue, so
   this step is simply not applicable and its absence is not a gap. It
   applies to tickets that resolve an item filed as an issue in an earlier
   sprint — cross-sprint follow-ups, disclosed-but-deferred gaps,
   escalations.

   Where it does apply, closing **does not happen automatically** (see
   Ceremony 1's "Board of record — split by lifetime" for why the
   auto-close mechanism structurally cannot fire on this branch topology).
   The merging party (or a designated follow-up chore, as T11.8 was for the
   T5–T10 backlog) must call the GitHub issue-write API directly:
   `state: closed`, `state_reason: completed`, plus a comment naming the
   merged PR(s) that resolved it, so the close is traceable without relying
   on the (non-functional) auto-link. Writing "Closes #N" in the PR body is
   still good practice for human/PR-review legibility, but it is not
   sufficient by itself and must not be treated as satisfying this step.

   **The enumeration is mandatory in review; the close at that moment is an
   optional early close.** Every PR review **states the issues that PR closes —
   by number, or explicitly "none"**. That sentence is owed by every review
   without exception: it is the input the sprint-level sweep and the retro both
   read, and it is the *symmetric half* of the rule T12's A6 added for issues a
   PR **opens** — the review template this project evolved has always had a slot
   for newly disclosed gaps and never had one for resolved ones, so a reviewer
   scanning their own checklist found nothing missing. Opened, closed, each
   named issue's current state, and label conformance now belong to one
   enumeration (see "Label taxonomy" below).

   The reviewer **may** perform the close at that moment, and where it is
   cheap it should: an issue closed at review time never misdescribes the
   codebase, which is the cost PE's recorded disagreement (below) correctly
   identifies. But the close is **not owed at that moment**, and a review that
   enumerates correctly without closing is **not a finding against this step**.
   The obligation to have closed lands on **the merged-fix issue sweep** in the
   "Definition of Done (sprint-level)" section, which is the primary and
   authoritative mechanism.

   *Why the step's shape changed rather than its wording being repeated*
   (T15.1, from T14 retro finding 1 and recommendation 1;
   `docs/process/t15-sprint-plan.md` §A4). This step previously read *"the
   review is the moment the close happens"* and it has now been measured twice:
   the closing API call was made **0 times out of 9** at that moment in T13, and
   **0 times out of 6** in T14, in both sprints while every other quality signal
   was green. T13 followed the **wording** half flawlessly across the same nine
   PRs — every title correctly said "closes #N" or "partial fix for #N" — which
   is the whole lesson: a rule that governs the sentence does not govern the
   act, and restating the same instruction a third time would be the third
   attempt at a mechanism with a measured 0/15 record. So the timing half is
   demoted to optional and the named sprint-level task carries the obligation,
   because a named sprint-level task is the mechanism that has actually produced
   correct outcomes on this project.

   **The enumeration half is kept, but it is kept on notice, not on a record.**
   Being honest about its measurement rather than assuming the surviving half is
   the healthy one: the *wording* rule scored 9/9 in T13 on PR titles, but the
   *review-level* enumeration line this step requires scored **0 of 9 reviews in
   T14**, its first sprint (T14 retro finding 5; the single instance was in PR
   #175's body, the wrong artifact, since this rule binds the review). It is
   retained rather than reshaped because one sprint is not two, and because
   unlike the close it is genuinely free — it is a sentence in an artifact the
   reviewer is already writing. **If it scores zero again in T15, it has the
   per-PR close's record and is due the same treatment: change its shape, or
   mechanize it, rather than exhort it a third time.** The retro is asked to
   score it explicitly.

   A PR that resolves only *part* of an issue says so in its title ("partial
   fix for #N"), names the successor issue or ticket that finishes it, and
   **leaves the issue open**. That is an honest half-claim, not a close, and
   it is not a finding against this step.

### Recovering an interrupted session's work

Implementer sessions can end mid-ticket without opening a PR — most commonly on
an account-level session limit, which terminates the session wherever it
happens to be. **The work is usually not lost, but it is only findable if
someone looks in the right place**, and the place is the interrupted session's
worktree, not the remote branch list.

Adopted at T15.1 as a named practice from T14 retro finding 2 and
recommendation 2. It earned the name rather than staying a one-off: it saved
two 8-point tickets in one sprint (T14.1 → PR #181, T14.4 → PR #182), and
T14.4's only copy in existence was an **unpushed local commit** — the exact
shape `docs/LESSONS.md` records (T11.4) as the case that polling and
branch-listing structurally cannot catch. The same interruption class will
recur.

**(a) Detect — check the worktree, not just the branch list.** When an
implementer session ends without a PR, before re-dispatching the ticket,
inspect that session's worktree for **both** unpushed commits and uncommitted
working-tree changes. **Listing remote branches is not sufficient and its
failure is silent:** T14.1 had pushed a commit but written no PR body, so a
branch listing found it; T14.4 had committed locally and never pushed, so a
branch listing found **nothing at all** and reported the same "nothing there"
answer it would give for a session that had done no work. Re-dispatching on
that answer discards finished work without anyone noticing it existed.

**(b) Verify before trusting, then disclose it in a first-line provenance
note.** Recovered work has not been through the loop that normally precedes a
PR, so it is verified from scratch — not read and pronounced fine. At minimum:
run the toolchain gates the ticket's own DoD requires, re-run the ticket's
headline verification in a fresh worktree rather than the one being recovered,
and scan the recovered diff for `TODO`/`FIXME`/unimplemented markers to
distinguish finished work from work interrupted mid-thought (#182 did exactly
this before deciding to recover rather than re-dispatch). Any PR opened from
recovered work then **must** carry a provenance note as the **first** paragraph
of its body, where it cannot be missed, stating what interrupted the original
session, what state the work was recovered from, and what was independently
re-verified. #181 and #182 are the template:

> *#181:* "…after pushing its commit but before writing this PR body. I (the
> coordinating/reviewing session) am opening the PR on its behalf, after
> independently reviewing, merge-conflict-resolving, and re-verifying the work
> from scratch"

> *#182:* "…after committing substantial work locally in its worktree but
> before pushing or opening a PR. I … recovered the uncommitted work from the
> worktree, reviewed it, committed it, resolved a clean auto-merge …"

**(c) The safeguard: a recovered PR is reviewed by a party other than the one
that recovered it, wherever one can be dispatched.** Recovery makes the
recovering session the author, and this project's standing supported mode (DoD
step 4 — merging on a reviewing agent's recommended verdict) has *"the review
was correct"* as its safety property, which presupposes the reviewer is reading
**someone else's** work. That presupposition does not hold for a
self-recovered, self-reviewed PR, and the measurement shows it: #181 and #182
hold T14's two shortest open-to-review windows — **9s and 8s**, against 59s for
the next shortest — because the reviewer had nothing left to do, having already
done it as the author.

Where no independent reviewer can be dispatched, **the recovering session states
that plainly in the PR** — that no independent review was available and the
review is therefore not independent — and the sprint's retro **independently
re-derives that PR's headline claim** and records the result, including
recording it when it did not. (T14's retro discharged this for T14.1 by
re-running the gate, re-deriving Side A and reproducing the mutation, and stated
explicitly that it did **not** re-perform T14.4's Host-only mutation check.
Naming the undischarged half is the practice working, not a gap in it.)

**(d) Never write a safeguard sentence the merge record will contradict.**
Delete *"I am not merging this myself either"* — and any equivalent — from a PR
the author does in fact intend to merge. #181 and #182 both carried that
sentence and were both merged by the same account 14 and 13 seconds later, with
no other party involved. The intent behind it was right; the sentence was
false. **A written safeguard that does not exist is worse than an acknowledged
absence**, because it stops the reader looking for the thing that is missing.
State the absence instead, per (c).

**Out of scope here — deliberately, and not by oversight: this section does not
decide whether a reviewing or merging session may author code at all.** It
governs detection, verification, disclosure and review independence, and it
grants no exception to CLAUDE.md rule 9. The underlying permission question —
may the session that reviews and merges a PR also be the one that wrote it,
whether by recovering an interrupted agent's work or by fixing a gap it finds
in someone else's PR — is a **rulebook** question, above this document's
altitude, and is being put to the user as **DECISION D2** via ADR-0016 (T15.2,
`docs/process/t15-sprint-plan.md` §A6). Do not read this section as a rule-9
carve-out, and in particular **do not read it as licensing a reviewer to fix
gaps it finds in a PR under review** — that is the separate case ADR-0016
covers, and treating the two as one question would pre-empt the escalation.
Until D2 is answered, the practice above applies to the case it was written
for: work an interrupted session had already finished.

## Ceremony 3 — Sprint retro (end of sprint)

The six-role team reconvenes and discusses: what went well, what mistakes
were made, what should change next sprint. Findings are written to
`docs/process/t<N>-retro.md` (CLAUDE.md's Docs index & naming convention —
retro-ceremony output is a distinct artifact from `docs/LESSONS.md`'s
incident postmortems, filed separately rather than folded in), indexed
from a short `## T<N> sprint retro` stub appended to `docs/LESSONS.md`
pointing at that file — same append-only, don't-rewrite-history discipline
`LESSONS.md` already follows, just applied to the stub rather than the
full findings. A retro that produces zero findings is treated as
suspicious, not a clean bill of health (mirrors QA's "an invariant with no
test that could fail is untested, not proven" heuristic, applied to
process).

## Execution loop mechanics (within a sprint)

Per-ticket execution (the "Execution" section above) runs as a bounded,
self-correcting loop rather than a single implement-and-hope pass:

1. **Loop cap: 5.** Each ticket gets at most 5 implement→review loops
   within the sprint. A loop is: an implementer subagent does TDD-first
   work on the ticket's branch and opens/updates its PR, then **PM +
   Principal Engineer review it together** — PM checks the result against
   the ticket's story/AC and the sprint goal, PE checks technical
   correctness, architecture fit, and test quality. Findings go back to
   the implementer for the next loop; each loop must **improve** quality
   and alignment to the sprint goal, not just churn — a loop that makes no
   measurable improvement is itself a finding for the retro, not something
   to quietly repeat.
2. **Learn within the sprint, not just at the end.** A mistake caught
   during a loop (not just at sprint retro) gets a line in
   `docs/LESSONS.md` immediately if it's the kind of mistake that could
   recur on a different ticket in the *same* sprint — don't wait for
   Ceremony 3 to write down something the very next loop needs to avoid
   repeating.
3. **Exhausting 5 loops without a mergeable result is not a silent
   failure.** It gets flagged explicitly in the sprint's tracking (ticket
   comment + sprint kickoff/retro note) as either mis-scoped (split the
   ticket) or a genuine open technical question (escalate to the user) —
   never left to quietly stall.
4. **Sprints auto-proceed to the next sprint's planning.** Once a sprint's
   Definition of Done is met (all in-scope tickets merged, retro held,
   `docs/LESSONS.md` updated), Ceremony 1 (backlog refinement) and
   Ceremony 2 (sprint planning) for the *next* sprint start automatically
   — the team does not wait for a prompt to begin planning the next
   sprint. **PE and PM jointly own defining each sprint's goal and scope**,
   checked against the project's overall goal (the locked decisions in
   `CLAUDE.md`/spec, not just the immediately preceding sprint) before the
   full six-role team ceremony.
5. **What "auto-proceed" does NOT change: CLAUDE.md rule 9 still applies
   in full.** Auto-proceeding to the *next sprint's planning* is not the
   same thing as auto-*merging*. Every PR — regardless of how many loops
   it took or how confident the implementer/reviewer roles are — still
   requires human review and explicit approval before merge. No subagent,
   including a PM/PE review pass that approves a ticket's content, may
   merge a PR itself. The loop above governs how work gets *ready* for
   human review, not a way around it.

## Label taxonomy

The taxonomy follows the board-of-record split (Ceremony 1): a label only
means something where there is an issue to carry it, so the two dimensions
that describe an *in-sprint ticket* moved into the sprint plan document.

**GitHub issue labels** — applied to issues, i.e. to cross-sprint
follow-ups, disclosed-but-deferred gaps, and escalations. Three axes, each a
**closed** set; decided at T14's Ceremony 1 (§A9) from T13 retro finding 4:

- **`role:` — mandatory on every issue.** One of
  `role:principal-engineer`, `role:product-manager`, `role:product-engineer`,
  `role:qa`, `role:product-owner`, `role:business-analyst`,
  `role:ux-ui-designer` — the primary-owner role.
- **`type:` — mandatory on every issue.** One of `type:story`, `type:bug`,
  `type:chore`, `type:spike`. **`type:tech-debt` is retired and maps to
  `type:chore`.** A second word for one concept is precisely what CLAUDE.md
  rule 7 exists to prevent, and "tech debt" and "chore" are not
  distinguishable here in any way that would change a decision.
- **`context:` — sanctioned and optional.** One or more of
  `context:booking`, `context:competitions`, `context:facilities`,
  `context:identity`, `context:payments`, `context:socialplay`, plus
  `context:platform` for `internal/platform/**`. **The set is closed to those
  seven** — it is not free-form. This axis is orthogonal to the other two: it
  says *where* the work lands, not who owns it or what kind it is, and with
  six bounded contexts this repository genuinely lacked it.

*Why `context:*` was adopted rather than scrubbed*, since the alternative was
genuinely arguable: the labels were already applied and already useful, and
scrubbing costs the same effort as sanctioning while destroying information.
The recorded objection is BA's, and it is what makes this section load-bearing
rather than decorative: **GitHub auto-creates a label the first time it is
applied, so after the fact an invented label is indistinguishable from a
sanctioned one.** `type:tech-debt` and `context:*` were both invented
mid-sprint by single tickets with no decision behind them, and nothing could
tell them apart afterwards. That is why this list is the durable record and
why the set is closed — a taxonomy that is only stated in a sprint plan is not
a taxonomy, it is that sprint's habit.

**Label conformance is checked in review.** T12's A6 requires every review to
enumerate the issues its PR opened; per DoD step 5 it also enumerates the ones
the PR closes. That same enumeration now **checks each named issue against
this taxonomy** — `role:` present, `type:` present and in the closed set, any
`context:` values in the closed set — and fixes or reports what it finds. No
T13 review performed a conformance check, and three of eight execution-opened
issues were non-conformant as a result (#165 missing `role:`, #167 unlabelled
entirely, #168 carrying `type:tech-debt` plus two then-unsanctioned
`context:*` labels). All three were relabelled when this taxonomy landed.

**Each named issue's current state is read from the API, not from memory.**
Added at T15.1 from T14 retro recommendation 4. Whenever a review names an
issue — as opened, as closed, as partially fixed, or as a still-open gap it is
declining to fix — it reports that issue's `state` as fetched, in the same
`issue_read` call the label check is already making. **The cost is one field on
a call already being made**, which is the whole argument for it; nothing new is
fetched and no new step is added.

The failure it catches is not hypothetical. PR #178's body and PR #178's own
review **both** described **#97** as a still-open gap the PR was leaving alone
(*"the disclosed #97 gap"*, *"the separate, still-open #97 gap"*). #97 was
closed — `state: closed`, `state_reason: completed`, `closed_at:
2026-08-13T14:45:54Z`, closed two days earlier by T11.8's retroactive sweep —
**and was never about that gap in the first place**; its subject is
malformed-ID guards on write handlers. Two parties asserted an issue's state
and subject from memory, agreed with each other, and were both wrong, which is
exactly the shape a second reader cannot catch. The residual PR #178 actually
disclosed was consequently tracked nowhere until T15's Ceremony 1 filed it as
**#185**.

**Stated plainly, because this clause is being added to an enumeration that has
not yet been performed once.** The label-conformance check above scored **0 of
9 reviews in T14**, its first sprint, as did the closure-enumeration line in DoD
step 5 (T14 retro finding 5). Adding a third clause to an unperformed
enumeration is defensible only on the grounds that all of it is one sentence in
an artifact the reviewer already writes, and that one sprint is not a record.
**It is not defensible twice.** If T15's reviews again score zero on this
enumeration, the correct response is a mechanical check — the retro's own
suggestion was a `make label-conformance` target run against the live API — and
**not** a fourth clause or a fourth restatement. The T15 retro is asked to score
this enumeration explicitly and by count, in the same form: *reviews carrying
it, of reviews written.*

**Sprint-plan fields** — recorded in `docs/process/t<N>-sprint-plan.md`'s
ticket entry, **not** as GitHub labels, because in-sprint tickets no longer
have issues to label:

- **Sprint** (`sprint:t5`, `sprint:t6`, … in the old taxonomy) — now implied
  by which plan document the ticket is in, and stated in its entry.
- **Story points** (`points:1|2|3|5|8|13`) — now a field in the ticket entry.

Role and type are recorded in the plan entry too; they *additionally* become
GitHub labels if the item is ever filed as an issue. Historical issues
carrying `sprint:*`/`points:*` labels are left as they are — the labels are
not scrubbed retroactively, they simply stop being applied going forward.

## Scheduled removals — practices adopted with an expiry date

Some practices are adopted deliberately **for a bounded number of sprints**,
as a stopgap until a mechanical check replaces them. This project's named
recurring failure mode is **accumulating two shapes of one fix** — keeping the
human question *and* the automated gate, forever, because nobody wrote down
that the first was temporary. This table is that written record, so the
removing ceremony **executes a plan rather than re-making a judgement**.

| Practice | Adopted | Removal condition | Removed by | Status |
|---|---|---|---|---|
| **The dual coverage question** added to Ceremony 1's dependency-completeness check: *"for every gate, glob, or shared coverage artifact a ticket produces, which other in-flight tickets' outputs must it cover?"* | T14 Ceremony 1 (§A5), from T13 retro recommendation 4 — **explicitly for one sprint only** | **T14.1 merges** (the run-time gate-coverage check: computes both sides from `go list` + a source scan at run time, fails when a package holding a `func Test` is executed by no gate, and contains no hand-maintained package list). Once a gate computes this mechanically, asking a human the same question every sprint is the second shape. | **T15's Ceremony 1** — drop the question; it does **not** become a fourth standing planning question. If T14.1 did *not* merge, the question is carried one more sprint and this row's condition is re-checked at T16, unchanged. | **REMOVED at T15's Ceremony 1 (2026-08-15)**, on schedule, by executing the plan in this row rather than re-making the judgement. Condition verified three ways: T14.1 merged as **PR #181**, `merged_at` **13:23:26Z** (fetched, not assumed from numbering); **`make gate-coverage` green at 41 packages**, with side B printed as five `go test` invocations parsed out of the `Makefile` itself; and it is reachable from the gate — `Makefile:326` lists `gate-coverage` in `ci-checks`. The question was accordingly **not** asked in T15's dependency-completeness check (`docs/process/t15-sprint-plan.md` §A7, §A12) and did not become a fourth standing planning question. |

Adding a row here is the price of adopting a temporary practice: if a
ceremony cannot state the condition under which a new question goes away, it
is proposing a permanent one and should say so.

**Removed rows stay in this table as history; they are not deleted.** The
table's value is that a reader can see a temporary practice was actually
retired on schedule — which is exactly what becomes unreadable if retired rows
vanish, leaving an empty table indistinguishable from a project that never
adopted a stopgap or never removed one. A `Status` cell reading REMOVED is the
evidence that the mechanism works.

## Definition of Done (sprint-level)

All in-scope tickets merged per the per-ticket DoD above, sprint goal met or
explicitly descoped with reasoning recorded, **the merged-fix issue sweep run
and reported** (below), retro held with findings in
`docs/process/t<N>-retro.md` and indexed via a `## T<N> sprint retro` stub
in `docs/LESSONS.md`, `HANDOFF.md`/`CLAUDE.md` state updated for the next
sprint to resume from.

### The merged-fix issue sweep

**No issue may remain open whose fix merged this sprint.** This is the
**primary** mechanism for that property, and the obligation DoD step 5's
optional early close does not carry. Adopted at T14's Ceremony 1 (§A1) from T13
retro finding 1 and recommendation 1(ii) as a *backstop*; promoted to primary at
T15.1, after the per-PR moment scored 0/9 in T13 and 0/6 in T14 while every
other quality signal in both sprints was green (T14 retro finding 1;
`docs/process/t15-sprint-plan.md` §A4).

**Who runs it, and when — two sanctioned moments, plus a third that occurs in
practice and is named here rather than left unscored:**

1. **The retro (Ceremony 3) runs it and *reports* it. The retro does not
   block on it.** A retro that cannot be held until a bookkeeping sweep is
   clean is a retro that gets held anyway with the sweep hand-waved — and the
   retro is the artifact whose job is to *record* failures, so making it
   refuse to exist when there is a failure to record inverts its purpose.
   T13's retro found this gap precisely *because* it was free to run and
   report.
2. **The next sprint's Ceremony 1 runs the same sweep as its first act,
   before ranking anything**, and acts on what it finds. This is the
   authoritative run, and it is where the *"a party other than the merger"*
   property actually lives: a different session, on a different day's work,
   already re-reading the issue list in order to rank it. Ranking a backlog
   off an unswept list re-ranks work that is already finished.
3. **The third state, named: the merging session sweeps its own work at sprint
   end.** This is neither moment above, and it is what T14 actually did — six
   closes, every one correct, every one citing its resolving PR, all landing
   inside an **eleven-second window 65 seconds after the sprint's last ticket
   merged**, performed by the party that had just merged them, between 26
   minutes and 6h45m after the PRs that earned them (T14 retro finding 1,
   re-checked at `docs/process/t15-sprint-plan.md` §A4). It is named here
   because the scoring condition below originally enumerated only two branches
   — the per-PR step ran, or the sweep fired — and a sprint landing in this
   third state could be scored as neither, which is a defect in the rule, not
   in the sprint.

   **Classification: acceptable-but-not-sufficient.**

   - **Acceptable**, because the outcome is the one this rule exists to
     produce. The issue list was correct before the sprint closed, each close
     named its PR, and the arithmetic reconciled. This is not scored as a
     failure and a sprint that does it is not in violation.
   - **Not sufficient**, because *"a party other than the merger"* — the
     property moment 2 exists to supply — **is not obtained**. A session
     sweeping its own merges cannot catch the one error class that matters
     here: having misread its own merge. It re-reads its own conclusions and
     finds them consistent, which is what T14's six-for-six record measures and
     also what it cannot distinguish itself from. The eleven-second window is
     the same signature as the 9s/8s review windows in "Recovering an
     interrupted session's work" (c) above, and it means the same thing.
   - **Also not sufficient in time.** The issue list misdescribed the codebase
     for up to 6h45m *inside* the sprint — which is worse than the
     between-sprints window PE's recorded disagreement (below) predicted, and
     is why that disagreement is only half-resolved rather than dismissed.

   **Consequence, stated so it cannot be inferred away: a self-sweep does not
   discharge the next Ceremony 1's run.** Moment 2 re-runs the arithmetic
   anyway, in full, and does not shorten it on the strength of the previous
   sprint having swept itself. "The merger already checked" is not a reason to
   skip the check whose entire purpose is that the merger is not the one
   checking.

   **This rule has been run once, and it earned its keep on the first run.**
   T15's Ceremony 1 (`docs/process/t15-sprint-plan.md` §A0) re-ran the sweep
   over T14's self-swept list: the closes were confirmed correct and the
   arithmetic reconciled exactly (`19 − 6 + 0 = 13`, matching `totalCount`), so
   the self-sweep's *closes* were sound — and the independent run still found
   what the self-sweep had not, because it was looking at a different half of
   the rule. **Three partial-fix issues (#147, #168, #149) were correctly left
   open but carried no written sentence saying why or naming a successor** — the
   sweep's second disposition requires that sentence, and T14 wrote it 1 time in
   4 — **and one residual (#185) was tracked nowhere at all**, having been
   disclosed in PR #178 and misattributed to an issue that was already closed
   and had never been about it. All four are now written on the issues
   themselves rather than in a plan document no reader of the issue list opens.

**The sweep itself — three steps, the first of which is one API call:**

1. **List the open issues.**
   `list_issues(owner: "nhuthuynh", repo: "white-label", state: OPEN)` —
   equivalently `gh issue list --repo nhuthuynh/white-label --state open
   --json number,title,labels`. Keep both the `totalCount` and the numbers.
2. **Reconcile the count arithmetically, before reading any issue
   individually.**
   `open_at_end_of_previous_sprint − closed_this_sprint + opened_this_sprint`
   must equal `totalCount`. A mismatch is a hit even when no single issue
   looks wrong. This is the cheapest possible check and it is how the
   incident was both proved and repaired: T13's retro computed 19 + 9 = 28
   with nothing closed (arithmetic proof of total failure before a single
   issue was opened), and T14's Ceremony 1 computed 28 − 9 = 19 to prove the
   repair.
3. **Cross-reference merged PRs against the open list.** For every PR merged
   this sprint — `list_pull_requests(state: closed, base:
   claude/go-backend-pickleball-7up34j)`, filtered on `merged_at` within the
   sprint window (merge order, **not** PR number; this project's numbers and
   merge order routinely disagree) — extract every `#N` referenced in its
   title or body. **Any `#N` appearing in both that set and step 1's open
   list is a hit.**

**What to do with a hit** — three dispositions, and exactly one of them is a
judgement call:

- **The PR fully resolved it →** close it now, per DoD step 5's mechanics:
  `state: closed`, `state_reason: completed`, plus a comment naming the merged
  PR(s). This is bookkeeping being caught up, not a decision being re-made.
- **The PR was a partial fix →** leave it open and **write down why**, naming
  the successor issue or ticket that will close it. T14's Ceremony 1 (§A0)
  gives the two worked examples: **#131** stays open because T13.9 fixed the
  Payments half only and the cross-context half is **#158**; **#147** stays
  open because T13.6 shipped Host-only and the Game-Admin half needs the store
  T14.4 builds. A correctly-open issue is a sweep *result*, not a sweep
  failure — but it is only correct once someone has written the sentence.
- **Neither is clear →** it is a scope question, not bookkeeping. Escalate to
  the sprint's PM/PE rather than guessing at a close.

**Report the sweep's result either way, with the count.** A clean sweep is
stated as clean and numbered; **a sweep whose output is silence is
indistinguishable from a sweep that was never run**, which is the exact
failure mode this step exists to catch. Zero findings is a result, not a
reason to omit the section.

**This is not a prediction — it has been run once and it worked.** T14's own
Ceremony 1 executed this sweep as its first act: nine issues closed with a
comment naming each resolving PR, the count independently reconciled
(28 − 9 = 19), two issues re-checked as correctly left open — all before a
single backlog item was ranked (`docs/process/t14-sprint-plan.md` §A0). The
amendment carries its own evidence rather than a hope.

**Recorded disagreement — PE, per "do not manufacture consensus" (Ceremony 2
rule 3) — now resolved, with PE's cost confirmed.** PE held that placing the
*authoritative* sweep in the next sprint means the issue list misdescribes the
codebase for the entire gap between sprints, and that this is a real cost:
anyone reading the repository in that window sees finished work as open. PE's
position was that the per-PR half (DoD step 5) must be primary and the sweep
merely a rarely-firing backstop. PO's answer was that the per-PR half is exactly
the half that failed 9/9 under dispatch pressure.

**Resolved at T15.1 by two sprints of measurement, and the resolution splits the
two claims apart:**

- **PE's cost claim is confirmed, and was understated.** T14's issue list
  misdescribed the codebase for up to **6h45m**, and that window fell *inside*
  the sprint rather than between sprints. The cost is real and is now recorded
  in the third state's classification above rather than treated as
  hypothetical.
- **PE's proposed remedy is not adopted.** Making the per-PR step primary is
  the remedy that has now been tried twice and produced **0/9** and **0/6**.
  The mechanism that produced T14's correct outcome was a *named sprint-level
  task*, not the per-PR step. So the sprint-level sweep is **primary** (this
  section), and the per-PR close becomes an **optional early close** (DoD step
  5) which is where PE's cost argument is actually served — closing early is
  encouraged precisely because it shortens the misdescription window, but
  nothing depends on it happening.
- **The per-PR step is not exhorted a third time.** Changing a rule's shape is
  the response to a mechanism that scores zero; repeating it louder is not.

> **Scoring condition (replaces the two-branch version, which T14 landed
> outside of — that gap is the defect this rewrite fixes).** Score each sprint's
> closure record into exactly one of three states, and record which:
>
> 1. **Closed at review time, per-PR** (DoD step 5's optional early close) —
>    the misdescription window is near zero. Good; nothing to change.
> 2. **Closed by a sweep run by a party other than the merger** — the
>    designed path. Good; the property holds.
> 3. **Closed by the merging session sweeping its own work** —
>    acceptable-but-not-sufficient, per the third state above. The next
>    Ceremony 1 runs the full sweep regardless and reports what it found.
>
> **Only one outcome is a failure: an issue whose fix merged this sprint is
> still open when the next Ceremony 1 sweeps.** If that happens, the sweep's
> mechanics — not its exhortation — are what the next retro strengthens; a
> repeat of state 3 across two more sprints is a signal to automate the sweep,
> not to re-word it.
