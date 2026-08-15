# ADR-0016: May a session that reviews and merges a pull request also write code on it — escalated to the user, not decided here

- **Status:** **Escalated — awaiting decision (D2).** Deliberately **not**
  Accepted, and no option below is chosen. See `## Status` for why this ADR
  records a question rather than an answer, and why this particular question
  is one this team may not answer for itself.
- **Date:** 2026-08-15
- **Ticket:** T15.2 (`docs/process/t15-sprint-plan.md`, §A6 and the T15.2
  ticket text)
- **Escalates:** T14 retro recommendation 3 — *"rule 9's 'a reviewer never
  commits' needs enforcement or an honest carve-out"*. This ADR does **not**
  pick one; it puts the choice to the user.
- **Relates to:** `CLAUDE.md` golden rule 9 (the rule in question), ADR-0015
  (the model escalation this one copies, and the other live open decision —
  D1), `docs/LESSONS.md` (the two incidents that produced rule 9 and its
  tightening), T15.1 (`docs/process/sprint-process.md`'s worktree-recovery
  practice, whose permission clause was deliberately carved out to this ADR)
- **Consumed by:** nothing in T15, by design. The first sprint after D2 is
  answered implements it — in `CLAUDE.md` and/or `sprint-process.md`, as the
  answer directs.

## Status

**Escalated — awaiting the user's decision. This ADR decides nothing.**

It follows ADR-0015's shape deliberately: state a question a non-engineer can
answer, record the verified facts, lay out options with real costs, **pick
none**, and tie the trigger to the answer arriving. ADR-0015 is the precedent
because T14's retro called it *"a model escalation"*; this ADR is the second
use of that device.

**Why this one is escalated rather than decided in a sprint document.** The
full reasoning is in the T15 sprint plan's §A6 and is not repeated here, but
the three load-bearing points are:

1. **`CLAUDE.md` is a different altitude from `sprint-process.md`.** Rule 9 is
   a golden rule in the durable rulebook, whose header says these instructions
   **override any default behavior**. A carve-out is an amendment to that
   rulebook. `sprint-process.md` describes how this team works *within* the
   rules; it cannot grant itself relief from one.
2. **Rule 9's own text forecloses the team writing its own exception.** It
   reads: *"Nothing is 'low-risk enough' to skip this on its own judgment —
   that judgment call is exactly the failure mode this rule exists to
   remove."* That clause was added after the *second* incident, in which the
   "just docs, low risk" judgement was made. A carve-out drafted by the agents
   the rule constrains, on the grounds that these particular commits were
   mechanical and low-risk, is that same judgement a third time.
3. **The party that benefits from a loosening should not be the party that
   grants it.** How much independence the user wants in their own review
   process is a question about the user's assurance, not an engineering
   trade-off.

**A recorded disagreement, preserved rather than resolved.** Per the T15 plan's
§A6: PE's position was that the retro said *"pick one"*, that the bounded
carve-out is obviously correct for mechanical single-file fixes caught at
test-merge time, and that escalating costs a sprint of ambiguity. PO's position
was that the rule's own text names this exact judgement as the failure mode.
Resolved in favour of PO **on authority**, with PE's substance preserved: PE's
carve-out is written below as option (b), **fully specified and directly
approvable**, so that choosing it costs the user one word rather than another
sprint.

## Context — the three instances, re-derived from the GitHub API for this ADR

Per the T15.2 ticket's instruction 3, every number below was fetched from the
API by this ticket and recomputed, **not carried forward from the retro or the
sprint plan on trust**. Where a re-derived number matches the plan, it is
marked as confirmed; nothing here is quoted from the plan without independent
checking.

### The measurements

| PR | Ticket | Opened | First review | Merged | Open→review | Open→merge |
|---|---|---|---|---|---|---|
| **#181** | T14.1 | `13:23:12Z` | `13:23:21Z` | `13:23:26Z` | **9s** | **14s** |
| **#182** | T14.4 | `13:28:35Z` | `13:28:43Z` | `13:28:48Z` | **8s** | **13s** |
| **#179** | T14.7 | `07:14:01Z` | `07:19:44Z` | `07:19:49Z` | 343s | 348s |

All timestamps are 2026-08-15 UTC. All three PRs were opened by, reviewed by,
and merged by the same account (`nhuthuynh`); `merged_by` on #181 and #182 was
read from the API directly.

**The sprint median, computed here rather than assumed.** T14 landed exactly
nine ticket PRs — #175, #176, #177, #178, #179, #180, #181, #182, #183 — a
denominator independently confirmed by #183's own review text (*"This is T14's
final ticket — all 9 tickets now merged"*). Their open→merge durations, in
seconds: 64, 85, 92, 150, 348, 20441, 14, 13, 183. Sorted, the fifth of nine
is **92 seconds** (PR #177, T14.2). This **confirms** the 92-second median the
sprint plan states. The median is robust to the single long outlier (#180, 5h
40m, which sat open across a session boundary).

So #181 and #182 merged in **14s and 13s against a 92s median** — roughly a
seventh of the sprint's typical open-to-merge time, and the two fastest of the
nine.

### What the speed figure does and does not prove

Stated carefully, because an escalation that overstates its own evidence is
not a fair question to put to anyone:

- **It does not prove the work was unreviewed.** Both PR bodies state that
  verification — merge-conflict resolution, a full toolchain run, a mutation
  check, and a re-run from a fresh worktree — was performed *before* the PR was
  opened. That claim is detailed, specific, and consistent with the rest of the
  sprint's record. The elapsed time measures the **published record**, not the
  work behind it.
- **It does prove no independent party could have reviewed in that window.**
  Nine and eight seconds is not enough time for anyone other than the author to
  read a 1,430-line or 2,467-line diff. Whatever assurance those merges carry
  comes entirely from the authoring session's own account of itself.

That distinction *is* the question. D2 asks the user how much of the second
kind of assurance they want to require.

### The sentence both PRs contain, and the sentence that contradicts it

Both #181 and #182 open with a provenance note containing, verbatim:

> *"per CLAUDE.md rule 9, I am not merging this myself either; a normal review
> cycle follows below."*

Both were then merged by that same session — 14 and 13 seconds after opening,
5 seconds after its own review. And each of those reviews closes with, verbatim:

> *"Merging per CLAUDE.md rule 9."*

**The same rule is cited, in the same pull request, minutes apart, for two
opposite propositions**: as the reason the session will *not* merge, and as the
authority under which it *does*. This is the single sharpest fact in this ADR.
It is not evidence of bad faith — it is evidence that rule 9 currently has no
settled meaning in practice, which is precisely the state the retro described
as *"currently both simultaneously, which is the state that guarantees it
drifts."*

**T15.1 fixes the wording, not the permission.** `sprint-process.md` clause (d)
now requires deleting the *"I am not merging this myself either"* sentence from
any PR the author intends to merge — because *"a written safeguard that does
not exist is worse than an acknowledged absence."* That removes the false
claim. It does not answer whether the underlying practice is allowed, which is
why the permission half was carved out to this ADR.

### The third instance: PR #179, a fix authored during the review

#179 differs in kind from the other two and is the reason a recovery-only
carve-out would not cover the field. Its review, read from the API for this
ADR, states:

> *"**Fixed it directly on the PR's source branch**, matching this session's
> established practice for cross-PR gaps found at merge time"*

The gap was real and worth catching: test-merging against the shared branch tip
surfaced that T14.8 had added a new sentinel (`ErrMalformedCourtID`) with no row
in the PR's own new exhaustiveness table, and `go test ./... -race` failed for
real on the merged tree. The reviewer added one table row per context and
pushed to `feature/t14.7-error-mapping-consistency`.

Three things about this instance matter to D2:

1. **The fix was genuinely mechanical** — one table row per context, following
   the file's existing convention, caught by a failing test and re-verified
   afterwards. It is the strongest possible case for option (b).
2. **It was not a recovery.** No session was interrupted; this is a reviewer
   fixing a branch under review. Option (c) would leave it prohibited.
3. **The review calls it *"this session's established practice"*** — a
   description of a settled habit, not a one-off. A habit that contradicts a
   golden rule is exactly what needs a decision rather than a case-by-case
   judgement.

### One adjacent fact, recorded but deliberately not escalated here

All three reviews are GitHub `COMMENTED` reviews, not `APPROVED` — each notes
that the account cannot self-approve. Rule 9 requires changes be *"reviewed,
tested, and explicitly approved before merge."* Whether a self-authored
`COMMENTED` review satisfies "explicitly approved" is a **different** question
from D2, it was not in this ticket's scope, and it is recorded here only so a
future reader knows it was observed and consciously left out — not overlooked,
and not silently folded into D2.

## The question

Stated as one sentence a non-engineer can answer, per the T15.2 ticket's
instruction 2:

> **DECISION D2 (for the user):** When the same session both reviews and merges
> a pull request, may it also write the code on that pull request — and if so,
> under exactly what limits?

## The options

None of these is chosen here. Costs are stated as neutrally as they can be;
where a cost is a judgement about assurance rather than an engineering fact, it
is marked as such.

| | Option | What it means | Cost |
|---|---|---|---|
| **(a)** | **Strict enforcement** — rule 9 as written, with teeth | A reviewer that finds a gap **requests changes**. The implementer fixes it, or the ticket takes a second dispatch loop. No session ever merges code it wrote. | A mechanical one-line fix caught at test-merge costs a full dispatch cycle — the #179 case would have cost a re-dispatch to add two table rows. Where no second party can be dispatched at all (the #181/#182 situation, an interrupted session), taken literally this means the work does not land that sprint. |
| **(b)** | **A bounded carve-out** — PE's position, fully specified below | Reviewer authorship permitted **only** when all five conditions hold: mechanical, compiler- or test-caught, single-file, disclosed in the review, and itself re-verified. Anything failing any condition falls back to (a). | It is an exception to a rule whose own text says nothing is low-risk enough to except. Every condition is judged by the same session that benefits from the judgement — "mechanical" and "single-file" are checkable after the fact from the diff, but "mechanical" still contains a judgement call. Creates a precedent that golden rules admit carve-outs. |
| **(c)** | **A recovery-only carve-out** | Authorship permitted **solely** to recover an interrupted session's unpublished work (`sprint-process.md`'s worktree-recovery clause (b)); never to fix a gap on a branch under review. | Covers #181 and #182; **leaves #179 unresolved** — and #179 is the instance the reviewer itself called *"established practice"*, so the ambiguity that produced this ADR would survive the decision in the case most likely to recur. |
| **(d)** | **Strict enforcement, plus a named escape hatch that costs something** | (a) by default, but where no second party can be dispatched, the session may proceed *and must* record that fact in the PR title or first line, with the sprint retro independently re-deriving that PR's headline claim. | Keeps the rule absolute in normal operation and makes the exception visibly expensive rather than invisibly cheap. But it relies on a retro doing the re-derivation, and T14's own retro records a reviewer promising a fix that never happened and surviving a whole sprint (§A1) — so the enforcement leg has a demonstrated failure mode in this project. |

**Note on (d):** it is listed because the T15.2 instruction sets a *minimum* of
three options and this fourth one is real — `sprint-process.md`'s
worktree-recovery clause (c) already contains most of its machinery, adopted in
T15.1. It is listed because it exists, **not because it is preferred; this ADR
expresses no preference.**

**Note on the interaction between (b)/(c) and T15.1:** the worktree-recovery
practice adopted in T15.1 is already in force and is *not* what D2 decides. D2
decides whether the recovering or reviewing session may **author** — T15.1
decides only how a recovery is detected, disclosed, and safeguarded once
permitted.

### Option (b), written out in full so it can be approved as-is

Per the T15.2 ticket's instruction 4, PE's position is specified completely
rather than sketched, so that approving it requires no further drafting. **If —
and only if — the user chooses (b)**, the following becomes the text of the
carve-out, appended to `CLAUDE.md` rule 9 by the sprint that implements D2:

> **Reviewer-authored fixes — the narrow exception.** A session that reviews a
> pull request may commit a fix to that pull request's source branch **only
> when every one of the following five conditions holds.** If any one fails,
> the reviewer requests changes and does not commit.
>
> 1. **Mechanical.** The fix applies an existing, already-decided convention
>    that is visible in the file being edited — e.g. adding a row to an
>    exhaustive table, or a missing case to a switch whose other cases settle
>    the shape. It introduces **no new decision**: no new domain rule, no new
>    error mapping not already established elsewhere, no schema change, no API
>    or proto change, and no change to a test's assertions.
> 2. **Compiler- or test-caught.** The gap was surfaced by a failing build or a
>    failing test — not by the reviewer's own reading, taste, or judgement. The
>    reviewer must quote the actual failure in the review.
> 3. **Single-file.** The fix touches one file. (A change that must be repeated
>    across bounded contexts is **not** single-file and does not qualify, even
>    when each edit is individually trivial.)
> 4. **Disclosed.** The review states, in its own text: that the reviewer
>    authored the fix, what the fix was, the failure that prompted it, and the
>    branch it was pushed to. A reviewer-authored fix that is not disclosed in
>    the review is a rule-9 violation regardless of its content.
> 5. **Re-verified.** After the fix, the reviewer re-runs the full gate on the
>    fixed tree and reports the result. A fix that is not itself re-verified
>    does not qualify.
>
> This exception covers **fixes to a branch under review**. It does not permit
> a reviewing session to author a feature, and it does not by itself settle
> the separate question of recovering an interrupted session's work, which is
> governed by `sprint-process.md`'s worktree-recovery clauses.

**Measured against the observed instances, so the user can see what approving
(b) would actually license:** #179's fix satisfies conditions 1, 2, 4 and 5 on
the record — but it added *"one table row per context"* across booking,
socialplay and competitions, so on condition 3 it is **not** single-file and
**would still be prohibited**. This is stated plainly because it is the honest
consequence: option (b) as PE specified it does **not** retroactively bless the
one instance that prompted it. The user may of course approve (b) with
condition 3 relaxed to "one mechanically identical edit repeated per context" —
but that is a different rule, and this ADR will not assume it.

## The interim rule — which is not a decision

**Until D2 is answered, `CLAUDE.md` rule 9 governs exactly as written.** A
reviewer that finds a gap on a branch under review **requests changes**; it
does **not** fix the branch itself.

This needs no permission and grants none: it is the existing rule, followed.
**Nobody may read this open escalation as a suspension of rule 9**, a grace
period, or a signal that the rule is provisional pending appeal. An unanswered
question about whether a rule should change leaves the rule in force —
otherwise raising the question would itself be the carve-out, which would make
this ADR the very thing it declines to be.

## Trigger condition

**The sprint immediately following the user's answer to D2 must implement that
answer** — amending `CLAUDE.md` rule 9 and/or `docs/process/sprint-process.md`
as the answer directs, and updating the worktree-recovery clauses to match.

Mirroring ADR-0012's and ADR-0015's triggers deliberately, and in ADR-0015's
words: this trigger is **tied to an event outside any ceremony's own judgment —
the user's answer arriving — not to a sprint boundary.** A future Ceremony 1
that has the answer in hand and still does not implement it is subject to the
rule ADR-0010 established: defer again in prose and a reviewer may block on
that basis alone.

**If D2 is unanswered when T16 plans, that is a finding, not a decision** — and
specifically not an invitation for T16 to decide it on the grounds that waiting
has become inconvenient. The reasoning in `## Status` does not weaken with
elapsed time; if anything, a team that decides a question after being told to
escalate it has demonstrated the exact failure the rule guards against.

## Concretely, until D2 is answered, no PR may

Mirroring ADR-0015's and ADR-0012's restriction lists, so this escalation has
teeth rather than being advisory:

1. **Treat "documented in ADR-0016" as permission.** This ADR records a
   question. Citing it as authority for a reviewer-authored commit is a rule-9
   violation *and* a misreading of this document. The same applies to citing
   option (b)'s text above — it is a draft of an option, not a rule in force.
2. **Amend `CLAUDE.md` rule 9**, in any direction — loosening, tightening, or
   "clarifying" — until D2 is answered. Tightening is excluded as well as
   loosening: a ceremony that pre-empts the user by hardening the rule has also
   answered D2 without being asked.
3. **Add a rule-9 carve-out, exception, or exemption to
   `docs/process/sprint-process.md`** or any other process document. §A6's
   whole point is that a process document cannot grant relief from a golden
   rule; writing the carve-out at a lower altitude does not change its
   altitude.
4. **Author a fix on a branch it is reviewing**, on the grounds that the fix is
   mechanical, small, obviously correct, caught by a test, or that dispatching a
   second party is impractical. Each of those is an argument *for* an option
   above; none of them is that option being chosen.
5. **Close this ADR, or mark it Accepted, on the grounds that the practice has
   settled in one direction.** It is escalated, not resolved. Only the user's
   answer resolves it.

This is a restriction on *guessing the answer*, not on the codebase generally:
ordinary review, ordinary implementation, and the T15.1 worktree-recovery
practice all continue normally.

## What would change this

**Would change it:** the user answering D2 (the trigger above); or a change in
the operating environment that removes the underlying tension — for example a
second reviewing party that can be dispatched reliably and independently, which
would make option (a) nearly costless and reduce D2 to a formality. That would
still be reported as a changed circumstance, and D2 still answered, rather than
silently resolved by the ticket that noticed it.

**Would not change it:** elapsed time; a reviewer's or agent's opinion about
which option is best; a further instance of the practice (a fourth occurrence is
more evidence for the question, not an answer to it); or the observation that
option (b) is obviously reasonable. **The reasonableness of (b) is not in
dispute — its authority is.** That distinction is the whole content of §A6, and
a future session that finds itself arguing (b) is sensible has established
something this ADR already grants and nobody contested.

## Consequences

**Pros.** The contradiction is now a written question with named options,
verified measurements, and an addressee who can actually answer it — rather
than a practice that is simultaneously forbidden by the rulebook and described
in review text as *"this session's established practice."* Every number was
re-derived from the API for this ADR, which upgraded the record in three ways
the plan did not have: the 92-second median is confirmed by direct computation
over an independently-confirmed nine-PR denominator; the *"Merging per CLAUDE.md
rule 9"* line was found sitting five seconds before the merge in the same PRs
whose bodies disclaim merging, making the contradiction citable rather than
inferred; and measuring option (b) against #179 revealed that PE's own carve-out
would **not** have permitted the instance that motivated it — a fact that
materially changes what approving (b) means, and that nobody had noticed.

**Cons.** **The ambiguity persists for at least one more sprint.** In the
meantime the interim rule is the strict one, which means a mechanical fix caught
at test-merge time costs a full dispatch cycle, and a session that hits an
account limit mid-ticket has no clean path to land its work. That is a real,
disclosed throughput cost, accepted here only because the alternative is the
constrained party writing its own exemption — which is what rule 9's second
tightening exists to prevent. This is also the **second** simultaneously-open
escalation (with D1 in ADR-0015), and two open questions addressed to the same
person is a load worth naming rather than hiding.

**Alternative considered and rejected: adopt option (b) here, since the retro
said "pick one" and PE argued for it.** Rejected on authority, not on merit —
see `## Status`. The retro's *"pick one"* is a request that the ambiguity end,
and this ADR ends it by routing the choice to the one party entitled to make
it. Adopting (b) would have been the third instance of an agent judging a case
low-risk enough to except from rule 9, which is the specific pattern the rule's
own text names.

**Alternative considered and rejected: adopt option (c) as the "safe minimum",
on the grounds that it is narrower than (b) and covers the two recovery
cases.** Rejected — narrowness is not neutrality. (c) is still a carve-out to a
golden rule, written by the constrained party, and it would leave #179's case —
the one described as established practice — exactly as unresolved as it is
today, while creating the impression that D2 had been dealt with. An escalation
that half-answers itself is worse than one that does not, because the remaining
half stops being visible.
