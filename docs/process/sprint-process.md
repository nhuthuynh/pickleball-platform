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
follow-ups, disclosed-but-deferred gaps, and escalations:

- `role:principal-engineer`, `role:product-manager`, `role:product-engineer`,
  `role:qa`, `role:product-owner`, `role:business-analyst`,
  `role:ux-ui-designer` — primary-owner role.
- `type:story`, `type:bug`, `type:chore`, `type:spike`.

Labels are created on first use (GitHub auto-creates a label the first time
it's applied to an issue if it doesn't already exist).

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

## Definition of Done (sprint-level)

All in-scope tickets merged per the per-ticket DoD above, sprint goal met or
explicitly descoped with reasoning recorded, retro held with findings in
`docs/process/t<N>-retro.md` and indexed via a `## T<N> sprint retro` stub
in `docs/LESSONS.md`, `HANDOFF.md`/`CLAUDE.md` state updated for the next
sprint to resume from.
