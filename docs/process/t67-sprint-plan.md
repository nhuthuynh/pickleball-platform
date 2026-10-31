# T67 sprint plan

Ceremony 1 ran as two independent report-only passes (Product Manager and
Principal Engineer), dispatched separately and deliberately not shown each
other's output. They disagreed on the sprint's main axis. That disagreement is
recorded in §4 rather than smoothed into a consensus neither reached.

## Sprint goal

> A real person can see what they booked and joined, so the loop they can
> already complete end to end stops forgetting them the moment they reload.

Deliberately not "#353 closed": the cancel half of #353 does not ship this
sprint, for the reason in §3.

## 1. Product decisions taken this sprint

Three questions were put to the product owner and answered. They are recorded
here because each one changes what gets built, and because a decision that
lives only in a chat log is not a decision the next sprint can find.

| # | Question | Answer | Consequence |
|---|---|---|---|
| D3 | A player cancels a registration they paid for. What happens to the money? (#358) | **The Host sets the refund policy per Game.** | The most capable of four options and the most expensive: a policy field on `Game`, a migration, a Host control, and a dialog that reads it. Turns the cancel ticket from ~5 points into 8–13, which is why cancel leaves this sprint. |
| D4 | Two Accepted ADRs were never implemented — facility timezone (#356, a live mispricing) and currency on `pricing_rules` (#357). | **Fix both before any pricing form.** | #356 and #357 are hard prerequisites gating #352. They become T68's first two tickets, ahead of the write path. |
| D5 | The sprint can carry one of two blockers. #353 (player visibility) or #352 (pricing end to end)? | **#353.** | #352 stays open another sprint. Tolerable only because nothing can write a pricing rule yet, so the unguarded overlap gap cannot be reached. |

D4 carries a tension worth stating: #356 is wrong **today** for any non-UTC
client, independent of the pricing form. Deferring it to T68 is a judgement
that one more sprint of mis-banded prices on a prototype is cheaper than
displacing a read. If that reads wrong later, this paragraph is why.

## 2. Tickets

| # | Ticket | Issue | Role |
|---|---|---|---|
| T67.1 | `ListBookingsForActor` — a player reads back their own court bookings | #353 | PdE |
| T67.2 | `ListRegistrationsForActor` + Game projection + `registrations(player_id)` index | #353 | PdE |
| T67.3 | The Bookings screen replaces `ComingSoonView` | #353 | PdE + UX |
| T67.4 | Quote failure tells the truth about an unpriced court | #352 | PdE |
| T67.5 | `make docs-doctest` — a documented command that does not reproduce its output fails a gate | — | PE |

**Naming, decided here so no implementer chooses it.** The RPCs are
`…ForActor`, not `ListMy…` as #353 suggests. Rule 7 (one ubiquitous language):
the codebase's existing actor-scoped read is
`ListRecurringHireTemplatesForActor`, and an RPC name is a permanent wire
surface.

**Rejected here so no implementer reopens it.** #353's alternative suggestion —
an actor filter on the existing list RPCs — is refused. `ListCourtBookings` is
a *public* availability read; an optional actor field would make one RPC mean
two different authorizations depending on whether a field is set. New,
authenticated, player-scoped RPCs instead.

## 3. What is deliberately not in this sprint

- **The cancel half of #353.** Blocked on D3 until this sprint, then made
  larger by D3's answer. Four Cancel RPCs still have zero client call sites.
  Its own slate, immediately after.
- **`CancelGame` / `CancelCompetition` controls.** Host actions whose cascade
  to Bookings and Registrations is #124, still open. Recorded as a comment on
  #124 rather than left as an undocumented gap.
- **Player-initiated competition-entry withdrawal.** #353's table implies this
  is one of four existing RPCs. It is not: `CancelCompetition` is the Host
  cancelling the whole competition, and no entry withdrawal exists at any
  layer — no domain transition (`internal/competitions/domain/entry.go` says so
  in as many words), no app method, no port method, no RPC. A full vertical
  slice, recorded in #358.
- **`ListCompetitionEntriesForActor`.** The third actor-scoped read. Cut to
  keep the sprint inside its velocity; the Bookings screen ships two sections,
  not three.
- **The role-doc and process-doc rewrite.** See §5.

## 4. The disagreement, recorded

PM proposed #353 and recommended #352's main half slip. PE proposed #352
end-to-end and said, in its own words, that it expected PM to argue for #353
and that the argument was strong. Both independently refused to split the
difference: "one of the two must be cut whole; splitting both produces two
half-features and a Bookings screen listing one of three things."

PE's estimates for the same work ran consistently higher than PM's — the
Bookings screen at 8 against 3, cancel at 5 against 3 — and PE's were the
better-grounded ones, because PE measured things PM guessed at. The clearest
case: PM flagged as an untested guess that the pricing overlap invariant could
not be an `EXCLUDE` constraint and would need a trigger. PE ran five designs
against a throwaway Postgres with all 32 real migrations applied and found a
purely additive `EXCLUDE` that works, with no column-type change and therefore
no sqlc row-type change. PM's guess was wrong, PM labelled it as a guess, and
the labelling is what let PE's measurement replace it rather than contradict a
confident assertion.

Both passes independently found the same falsehood in #352 — that the owner
screens' unreachability was "filed separately" when no such issue exists. Two
independent passes converging on one error is the strongest signal either
produced.

## 5. Why this sprint does not rewrite the role and process docs

A rewrite was planned and then rejected on evidence, by a report-only red-team
pass whose only brief was to argue against it. The measurements that decided it:

- Of the 23 incident entries in `docs/LESSONS.md`, classified by **each
  entry's own stated remedy**, 4 name better process prose. 13 name a gate, or
  an artifact that existed and nobody ran.
- The six role dossiers total 2,901 lines, were all last edited on
  `2026-07-31` in one commit, and are cited in **0 of 23 incidents and 0 of 58
  retros**. The last sprint plan to cite one was T9. Two still say
  "(T6, upcoming)".
- Across those 2,901 lines, **2 lines mention any gate this project runs**,
  against 386 external citations.
- `sprint-process.md` has grown +1999/−120 and `CLAUDE.md` +570/−55. A **6.4%
  deletion rate.** In that history exactly one practice was ever retired, and
  what retired it was a gate — `make gate-coverage` computing at run time what
  a planning question had been asking by hand.
- Nothing loads the dossiers. There is no `.claude/` directory, no settings
  file, no `@path` import in `CLAUDE.md`, and no reference to `docs/roles/` in
  either `CLAUDE.md` or `HANDOFF.md`. The only pointer is a table in
  `sprint-process.md`, four voluntary hops from the auto-injected context.

A further provenance finding, from the research passes themselves: external
primary sources are largely unreachable from this environment (`w3.org`,
`nngroup.com`, `istqb.org`, `testing-library.com`, `martinfowler.com`, `go.dev`
all refuse). One pass worked around it by cloning publishers' own GitHub
repositories and read primary text; another correctly declined to manufacture
quotations it could not see and marked every claim as search-snippet
provenance. `product-engineer.md:474-480` already admits the same limitation
for its own 30 citations. **So the existing dossiers' authority was never
verifiable here either**, which is a reason to prefer short repo-specific
mandates over imported industry prose regardless of who writes them.

T67 therefore takes **one** documentation ticket — T67.5, the doctest gate —
chosen because it converts a prose rule this project already has ("every
quantity carries the command that produced it") into a check, and because it
catches two live falsehoods in `CLAUDE.md` on its first run. The research is
not discarded: it becomes the input to short, enforced agent definitions in a
later sprint, with the industry rationale archived rather than loaded.

## 6. Execution

Four implementers, each in its own disposable `git worktree` cut from the same
shared-branch commit; no two write the project checkout's working tree. One
PR per ticket, reviewed by a report-only pass before merge. Rule 9 holds for
every artifact in this plan, this document included.

**Isolation the worktree does not provide — migration numbers.** T67.2 adds a
migration. `0032` is current; T67.2 takes `0033`. Any ticket that later adds one
re-lists the directory rather than trusting this line.

**Same-wave shared interface.** T67.1 and T67.2 run concurrently in different
bounded contexts, but both edit `proto/`, regenerate `internal/gen`, and
regenerate the web client; T67.1 widens `booking/port.Repository` (8 fake
implementers across 7 packages and 3 contexts) and T67.2 widens
`socialplay/port.RegistrationRepository` (implementers include a file under
`internal/payments/`). GitHub will report `mergeable_state: clean` for both
regardless. Whichever merges second verifies against a reconstructed
post-merge tree and runs `make test-adapters` on it — not on
`mergeable_state`.

**Every ticket carries a real-browser pass.** Neither issue's defect class is
visible to any gate: both were found by a person opening the app, the web
suite was green throughout, and `web/e2e/journey.mjs` names booking as a flow
it does not cover. A PR reporting only green unit tests is unverified for this
sprint's purposes.
