# T5 Sprint Retro

Ceremony 3 per `docs/process/sprint-process.md`, six-role team, held against
`docs/process/t5-sprint-plan.md`, the T5.4 capacity-invariant entry in
`docs/LESSONS.md`, and PRs #11–#15 (`nhuthuynh/white-label`, GitHub-side name
`pickleball-platform`). Findings below, not a single voice — recorded
disagreements are left as disagreements per the "do not manufacture
consensus" rule.

**1. The capacity-invariant gap was a ticket-writing gap first, a review-loop
catch second.** PE/QA: T5.4's ticket text (`t5-sprint-plan.md`) names the
DB-mirror requirement explicitly for *uniqueness* ("a unique constraint on
`(game_id, player_id)`... per CLAUDE.md rule 4") but never names it for
*capacity*, even though the sprint goal's headline clause is "capacity...
enforced" and ADR-0001 already established the precedent (invariant in
Postgres, not just the domain) two sprints ago. The PM+PE review loop caught
it, which is the mechanism working as designed — but it caught it after a
full implement pass, not before one started. BA: this is the same shape of
gap the BA role exists to catch (spec clauses that sound consistent but
don't compose) applied to ticket-writing rather than spec-reading — a
"capacity is enforced" AC and a "translate any new Postgres constraint"
non-functional note read as compatible without anyone checking whether the
second actually covers the first. **Process change (the one this retro
recommends for T6):** Ceremony 1 (backlog refinement) adds an explicit
checklist question PM+PE must answer per ticket, not just per review: *does
this ticket introduce a count-limited invariant ("N of these, capped at
M")? If so, does the ticket text itself require a DB-level guard (lock,
trigger, or equivalent), not only a unique-key guard?* Put in the ticket at
refinement time, this class of bug is caught before loop 1 opens a PR, not
during it.

**2. Every one of PRs #11–#15 hit the identical self-approval error, and
none had merged by retro time.** PdE: all five review passes independently
discovered "GitHub rejected APPROVE — Can not approve your own pull
request" and fell back to COMMENT-with-recommended-verdict, because the
MCP session's credentials and the PR-author account are the same
`nhuthuynh` identity. That's a correct, rule-9-compliant workaround each
time, but it repeated five times in one sprint with no process adjustment
— and as of this retro, `pull_request_read` shows **all five PRs still
open, zero merged**, `mergeable_state` clean on four and "dirty" on #15
(needs a rebase once #11 actually lands). PO: by the sprint-level
Definition of Done ("all in-scope tickets merged... `HANDOFF.md` state
updated for the next sprint to resume from"), T5 is not actually done —
every ticket is review-approved but human-merge-blocked, and the stack
means #12–#15 are all sequentially gated behind #11. PM: this isn't a
quality problem — the content of every review was substantive and
adversarial, not rubber-stamped — but a sprint whose entire output sits
unmerged is a real process risk if it repeats: work piles up faster than a
human reviewer can clear it, and later sprints will keep stacking on an
unmerged base (T6 tickets would stack on top of an already-5-deep unmerged
queue). **Recommended for T6:** either get a second, distinct reviewing
identity so an agent review can actually land as a real GitHub APPROVE, or
treat "COMMENT with recommended-event: APPROVE" as the documented, expected
terminal state in `sprint-process.md` itself (not a workaround) and add an
explicit sprint-level checkpoint for clearing the human-merge queue before
the next sprint's stack grows on top of it.

**3. The 5-loop cap was exercised meaningfully exactly once.** QA/PE: T5.1,
T5.2, T5.3, and T5.5 each converged in loop 1 with a straight recommended
APPROVE; only T5.4 used a second loop (2 of 5) to close the capacity-guard
finding. Notably, ticket size didn't predict loop count — T5.3 (8 points, a
real architecture judgment call on rollback-vs-single-court) also converged
in loop 1, while T5.4 (also 8 points) was the one that needed rework, and
the discriminator was "touches a new DB invariant," not story points. One
sprint and one instance of loop 2 is too little data to conclude 5 is
correctly calibrated — it's neither confirmed too generous nor too tight
yet. **Not changing the cap for T6**; watching whether any T6 ticket needs
3+ loops (which would suggest either the cap is fine and doing its job, or
that ticket was mis-scoped per the loop-mechanics rule) before revisiting
the number.

**4. Ticket sizing held up on the two small tickets; the stacked-diff
`additions` count is not a usable estimation signal.** PO: T5.1 (3 pts, 339
additions, 1 commit) and T5.5 (3 pts, small own-commit diff) both landed
clean in one loop, consistent with their points. T5.2 (5 pts) also
converged cleanly. The two 8-point tickets diverged in outcome (T5.3 clean,
T5.4 needed loop 2) for a reason unrelated to points, per finding 3.
PdE: raw PR `additions`/`changed_files` are misleading under this stacking
model — T5.4 shows 2,751 additions and T5.5 shows 3,058, but both totals
are inflated by carrying every earlier unmerged ticket's diff along; the
reviews themselves correctly scoped to "this ticket's own commit" rather
than the visible diff size, but anyone estimating T6 tickets from these PR
stats alone would badly overestimate T5.4/T5.5's actual size.
**Recommendation:** when using T5 as a reference class for T6 estimation,
use each ticket's own-commit diff (stated in each PR body's "Stacking"
section), not the PR's total `additions`, and treat "does this ticket touch
a new DB invariant or cross a context boundary for the first time" as a
better risk signal than point count alone.

**5. The PM/PE waitlist disagreement's resolution held up, and was tested
harder than expected.** PE: the kickoff note's promise — "a future
waitlist ticket can hook a promotion trigger onto that boundary without an
API-shape change" — got an unplanned early test when loop 2's DB-level
capacity trigger was added: the fix mapped its new `P0001` case onto the
*same* `domain.ErrGameFull` sentinel already wired through to a 409 in
T5.4's gRPC handler, with no proto change, no new error type, and no
handler change. That's about as direct a validation of the "stable hook"
design bet as this sprint could have produced. `docs/adr/
0006-waitlist-data-model-direction.md`'s status line was in fact updated to
point at T6, honoring PM's stated condition for accepting the compromise.
PM: agreed the technical hook is validated — but flags that the underlying
product gap PM raised (a real player hitting a hard, un-queued rejection
when a Game is full) is still fully live in what shipped; nothing about
the trigger fix changes the user-facing behavior, it only made the
rejection *reliable* instead of racy. This is not a reason to revisit the
T5/T6 split now — the scoped compromise did exactly what it said it would
— but it is a reason for T6 planning to treat the waitlist ticket as a
real commitment already validated as buildable, not a nice-to-have that
can slip again; PM's original condition ("not allowed to silently slip
again") should be checked explicitly at T6 Ceremony 1, not assumed.

**6. Several findings are flagged in PR bodies/`HANDOFF.md` prose but have
no tracked ticket yet, across two sprints now.** BA/PO: by count — (a)
Game-cancellation not cascading to Bookings/Registrations (flagged T5.1,
PR #11), (b) `domain.Register` not checking `Game.Status` (flagged T5.2
review, PR #12, now in `HANDOFF.md`), (c) `CancelBooking`'s missing
actor-ownership check, carried since T3 and re-flagged in T5.2 without a
fix, and (d) `CreateGame`/`Game.Cancel()` having no actor parameter at all,
found while scoping T5.5 (PR #15) and explicitly split out rather than
built. All four are real, all four are documented somewhere, and none of
the four is a GitHub issue with a `sprint:t*` label — they exist only as
prose. This is exactly the "flagged, not silently skipped" discipline the
tickets ask for working correctly at the PR level, but it puts the burden
on someone re-reading old PR bodies to not lose them. **Recommended for
T6:** Ceremony 1 for T6 should open these four as real tickets (in or out
of T6 scope, PM/PE's call) rather than let a third sprint pass with them
still living only in prose.

**No finding on:** UX/UI Designer and Product Owner had limited independent
material to contribute this sprint specifically — T5 shipped no
client-facing surface (no Vue/Swift/Kotlin work touched) and the backlog
mechanics PO owns (DoD, ticket ambiguity) were largely clean aside from
finding 6. Noted rather than manufactured, per the same rule this retro
follows for disagreements.
