# Sprint Process — Pickleball Platform

Durable process rulebook for how work moves from backlog to merged code on
this project, from this point forward (T5 onward — T0-T4 predate this
process and are not retroactively reprocessed, see `docs/LESSONS.md`).

Board of record: **GitHub Issues + labels** on `nhuthuynh/white-label`
(Jira was considered but isn't connected to this workspace; see
`docs/reviews/00-bootstrap.md`-adjacent decision log for why). If Jira is
connected later, this doc's ticket fields map directly onto Jira fields —
nothing here is GitHub-specific in spirit.

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
sequencing/feasibility; both sign off before tickets are opened. Each ticket
gets:

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
- **GitHub link**: the ticket *is* the GitHub issue; once work starts, the
  issue is linked to its branch and PR (GitHub does this automatically via
  "Closes #N" in the PR body, or manual linking if the PR doesn't close the
  issue outright).
- **Labels**: `sprint:t<N>`, `role:<primary-owner-role>`,
  `type:story|bug|chore|spike`, `points:<n>`.

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
   agent (CLAUDE.md rule 9).
5. The GitHub issue is linked to the merged PR and closed.

## Ceremony 3 — Sprint retro (end of sprint)

The six-role team reconvenes and discusses: what went well, what mistakes
were made, what should change next sprint. Findings are appended to
`docs/LESSONS.md` under a `## T<N> sprint retro` heading — same append-only,
don't-rewrite-history discipline the file already follows. A retro that
produces zero findings is treated as suspicious, not a clean bill of health
(mirrors QA's "an invariant with no test that could fail is untested, not
proven" heuristic, applied to process).

## Label taxonomy

- `sprint:t5`, `sprint:t6`, … — one per HANDOFF.md task/sprint.
- `role:principal-engineer`, `role:product-manager`, `role:product-engineer`,
  `role:qa`, `role:product-owner`, `role:business-analyst`,
  `role:ux-ui-designer` — primary-owner role for the ticket.
- `type:story`, `type:bug`, `type:chore`, `type:spike`.
- `points:1`, `points:2`, `points:3`, `points:5`, `points:8`, `points:13`.

Labels are created on first use (GitHub auto-creates a label the first time
it's applied to an issue if it doesn't already exist).

## Definition of Done (sprint-level)

All in-scope tickets merged per the per-ticket DoD above, sprint goal met or
explicitly descoped with reasoning recorded, retro held and appended to
`docs/LESSONS.md`, `HANDOFF.md`/`CLAUDE.md` state updated for the next
sprint to resume from.
