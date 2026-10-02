# T55 Sprint Retro

Ceremony 3 per `docs/process/sprint-process.md`, six-role team (briefs:
`docs/agent-operating-handbook.md` Part B), held against `HANDOFF.md`,
`docs/process/t54-retro.md` as the immediate precedent, PRs #291–#295, and the
live issue/PR/commit history.

> ## Written late, and what that costs
>
> **This retro was written at T61, not at T55.** `HANDOFF.md`'s T55 row said
> "not yet written" for six sprints. It is reconstructed from the record — PR
> bodies, commit messages, `merged_at` timestamps, the issue timeline and the
> diffs — and not from the sprint.
>
> What that loses is specific and worth naming rather than hedging around:
> **a retro's most valuable material is the disagreement that did not make it
> into an artifact.** A contemporaneous Ceremony 3 can record that the PE and
> the QA role disagreed about a scope call, or that a decision felt forced at
> the time. Six sprints later, only what was written down survives, so every
> finding below is one the record can support — which systematically biases
> this document toward process facts and away from judgement calls.
>
> What it does *not* excuse is vagueness. Every figure here was re-derived with
> a command at writing time, per `t61-retro.md` recommendation 4.
>
> One consequence of the delay is unavoidable and flagged in §6: several things
> T55 recorded as true have since been falsified, most importantly its Docker
> disclaimer. Where that happens this retro says so rather than reproducing the
> sprint's own belief as if it still held.

**T55 held no planning ceremony, deliberately.** It resumed an interrupted T54
session, and its first act was to put the two standing escalations (D1, D2) to
the user rather than open a thirty-fourth consecutive 0-ticket plan.

**Outcome: 4 tickets, 5 decisions answered, 3 issues closed, 1 opened, 2 ADRs
resolved.** Issue count 7 → 5. Merged as #291 (`442bc68`) → #292 (`797e6b3`) →
#293 (`9e039ab`) → #294 (`725ac72`) → #295, in that order, verified against each
PR's `merged_at`.

---

## 1. What actually happened, in order

1. The session inherited an interrupted T54 and, instead of planning, **asked**:
   the two open escalations (D1, D2) plus three product questions went to the
   user in one exchange.
2. **All five were answered the same day** (2026-09-04). D1 = option (a),
   authenticate the flow; D2 = option (b), the bounded reviewer carve-out,
   adopted verbatim and unrelaxed.
3. Four tickets were built over the following ten days: T55.1 (every Booking has
   an owner; only the owner may cancel), T55.2+T55.3 (cancelling a Game or
   Competition releases its courts and refunds who paid), T55.4 (`RefundPayment`
   admits `no_show_fee`, projecting nothing), and the process change applying
   T54's retro recommendations.
4. **Nothing merged until 2026-09-17**, when all of it merged inside nine
   minutes (§2).
5. `CLAUDE.md` rule 9 gained D2's five-condition carve-out; ADR-0015 and
   ADR-0016 both moved **Escalated → Accepted** in place.
6. The bookkeeping PR (#295) **had to be rewritten rather than rebased**,
   because the text written while the PRs were open no longer described
   anything by the time they merged (§3).

## 2. The finding this retro exists to record

**T55's five PRs were open for up to thirteen days and merged inside nine
minutes.** The figures, re-derived from the API rather than recalled:

| PR | created | merged | open for |
|---|---|---|---|
| #291 (T55.1) | 2026-09-04T04:01:52Z | 2026-09-17T12:23:52Z | **13d 8h** |
| #292 (T55.2+T55.3) | 2026-09-08T08:10:30Z | 2026-09-17T12:27:06Z | 9d 4h |
| #293 (T55.4) | 2026-09-10T06:33:29Z | 2026-09-17T12:29:11Z | 7d 6h |
| #294 (process) | 2026-09-13T07:28:16Z | 2026-09-17T12:30:21Z | 4d 5h |
| #295 (bookkeeping) | 2026-09-14T00:21:53Z | 2026-09-17T12:32:55Z | 3d 12h |

First merge to last: **9 minutes 3 seconds**, for thirteen days of work.

This is not a scheduling complaint. It is the mechanical cause of two things the
sprint recorded as incidental:

1. **The rebase churn.** `HANDOFF.md`'s T55 row notes that "#292 and #293 were
   each rebased onto the shared branch before merge, because each was stacked on
   its predecessor's pre-squash branch." That is a direct consequence: T55.2 had
   nowhere to branch from but T55.1's unmerged head, because T55.1 had not
   merged and would not for nine more days. Squash-merging #291 then orphaned
   #292's base. The row presents this as a fact about branching; it is a fact
   about **sequencing**.
2. **The stale bookkeeping PR** (§3). A document written against four open PRs
   describes a world that the merge burst destroys in nine minutes.

### Why it happened, as far as the record can say

Each PR waited for review, and the reviewer was the author. With no second party
to merge, nothing forced a merge at any particular moment, so merges happened
when someone thought to do them — which turned out to be all at once, at the
end. **Self-review removes not just the second opinion but the scheduling
pressure**, and the second loss is the one nobody writes down.

### Why this is a finding and not a nitpick

The project's convention is "one ticket, one PR, reviewed and merged before the
next". T55 followed the letter and inverted the intent: four PRs existed
simultaneously, each depending on the last, with the integration risk
accumulating silently until it was discharged in a burst. The rebases were
handled correctly. **That they were needed at all was avoidable, and the thing
that would have avoided it is merging #291 before starting T55.2.**

## 3. The bookkeeping PR that had to be rewritten

#295's own commit message is the primary source, and it is unusually candid:

> Rewritten against the merged reality rather than rebased. The original version
> of this change was written while all five PRs were open, and said "resolved
> pending merge" for three issues that are now genuinely closed; landing that
> text would have been recording a snapshot that no longer described anything.

Three observations:

1. **The sprint caught this itself, before merge.** That is the right outcome
   and is recorded here as such.
2. **It is §2 restated as a cost.** "Resolved pending merge" is only ever
   written by someone documenting a sprint whose PRs have not merged. Shorten
   the window and the phrase becomes unwriteable.
3. **The fix was a rewrite, not a rebase**, and the distinction matters: a
   rebase would have carried the stale sentences forward silently, because git
   does not notice that prose has gone false. **A document is the one artifact
   where a clean merge is no evidence of correctness.**

## 4. The decisions, and the thing the delay revealed

Five decisions were answered **within a single exchange of being asked**. One of
them, D1, had been open for 41 sprints.

Forty-one sprints of a decision being described as blocked, resolved in one
message. The blocker was never the decision's difficulty, the product's
complexity, or anything about D1 at all. **It was that nobody asked.**

T54's retro had recommended exactly this — raise escalations before planning —
and T55 was its first application. It worked immediately and at a scale that
makes the preceding thirty-three 0-ticket sprints look less like a blocked
backlog and more like an unasked question. That reading is harsh and the record
supports it: nothing about D1 changed between T13 and T55 except that someone
put it in front of the person who could answer it.

**This is the same shape T61 later found in "no Docker daemon" (see
`t61-retro.md` §2), one layer up.** A claim that licenses inaction — "blocked on
a product decision" — sits unexamined precisely because nothing is waiting on
it. T55 is the earliest instance in the record and was not recognised as a
pattern at the time; it took T56, T58, T61 and this retro to see it.

## 5. What went well

- **The escalations were raised before any ticket was refined**, which is now
  what `sprint-process.md` requires of every Ceremony 1. T55 is where that
  stopped being a recommendation and became behaviour.
- **Both ADRs were resolved in place rather than superseded.** ADR-0015 and
  ADR-0016 moved Escalated → Accepted with their original questions and option
  tables preserved. A new ADR would have been easier and would have hidden what
  was decided against.
- **D2 was adopted verbatim and unrelaxed.** The carve-out that reached
  `CLAUDE.md` rule 9 carries **five** conditions, every one of which must hold,
  rather than a general "a reviewer may fix small things". That precision is
  what let T61 state plainly that its own reviewer-authored fix *failed*
  conditions 1 and 2 — a vague rule would have been satisfiable by argument.
- **Four tickets, real scope.** After thirty-three 0-ticket sprints, T55
  delivered an owner on every Booking, cascade-cancel with refunds across two
  contexts, and a refund path for `no_show_fee`. The sprint's value is not only
  in its process findings.

## 6. What T55 believed that has since been falsified

A late retro can do one thing a contemporaneous one cannot: say which of the
sprint's own claims did not survive.

- **"The Docker-backed integration tests were never executed, and
  `make ci-integration` is still owed."** T55 recorded this honestly in
  `HANDOFF.md` and in all four PR reviews. **It was false** — Docker was
  available the whole time, and T61 ran the suite in about four seconds of
  daemon start. More pointedly: T55.1 added `bookings.owner_user_id` as a uuid
  FK (migration 0027) and T55.1–T55.3 altered the table the no-double-booking
  `EXCLUDE` invariant lives on, and the sprint said plainly that the concurrency
  proof had not been re-run across that change. It could have been. See
  `t61-retro.md` §2 and `docs/LESSONS.md`.
- **T55.1's fixtures went stale on merge.** `bookings.owner_user_id` became a
  uuid FK into `identity_users`, and the integration fixtures that seeded
  bookings were not updated — which is one of the five fixture-rot root causes
  T61 had to fix. Nothing could have told T55 this, because no gate it could run
  executed those files. That is the gap, not the oversight.

## 7. Recommendations

Written from T61, so these are stated as what the record supports rather than as
what T56 should have done.

1. **Merge each ticket before starting the next one that depends on it.** T55's
   rebase churn and its rewritten bookkeeping PR both trace to a 13-day window
   where four dependent PRs were open at once (§2, §3). This is the sprint's one
   concrete process lesson and it was never written down.
2. **A docs PR written while the PRs it describes are open must be re-read
   against merged reality before merging, not rebased.** #295 did this correctly
   by instinct. Git cannot tell you that prose went false (§3).
3. **When a decision has been "blocked" for more than a few sprints, the thing
   to check is whether anyone asked.** D1 waited 41 sprints for a message (§4).
   `sprint-process.md`'s escalation rule now covers this; the reason to restate
   it is that T55 is the evidence for how large the effect is.
4. **Self-review costs scheduling pressure as well as a second opinion.** The
   nine-minute merge burst is what "nobody is waiting for this" looks like in
   the timestamps (§2). Worth knowing for as long as this project reviews its
   own work, which as of T61 is seven consecutive sprints.

## 8. Sweep and bookkeeping

- **Issues: 7 → 5.** #144, #124 and #130 closed; #296 opened from T55.1's merge
  review. All closures manual — `Closes #N` cannot auto-fire against a
  non-default base branch, which has been true for every sprint in this project.
- **Merge order #291 → #292 → #293 → #294 → #295**, verified against each PR's
  `merged_at` (§2's table), not inferred from numbering. They agree here, which
  is itself a checked fact rather than an assumption.
- **ADR-0015 and ADR-0016: Escalated → Accepted.** D1 = (a) authenticate the
  flow. D2 = (b) the bounded carve-out. Neither is a new ADR.
- **`CLAUDE.md` rule 9 gained D2's five-condition carve-out**, which is the
  single most-cited process artifact T55 produced — invoked by name at T61.
- **#295 is not in the T55 Docs-index row's PR chain**, because #295 *wrote*
  that row and a PR cannot cite its own merge number. Same structural rule that
  keeps a retro PR from writing its own row. Noted so the apparent omission is
  not read as an error.
- **Self-review for the whole sprint.** All four code PRs carry review comments
  rather than approvals, each disclosing that the reviewing session authored the
  code.
- **This retro does not update `HANDOFF.md`'s T55 Docs-index row**, which still
  reads "not yet written" for the retro cell. Per `sprint-process.md` a retro PR
  cannot write the row that points at it. T62's Ceremony 1 owns it.

## 9. Honest-form outcome sentence

For `HANDOFF.md`'s T55 row, to be carried verbatim rather than strengthened:

> T55 held no planning ceremony, deliberately: it resumed an interrupted T54 and
> its first act was to put the two standing escalations to the user rather than
> open a thirty-fourth consecutive 0-ticket plan. **Five decisions were answered
> within a single exchange of being asked, one of which (D1) had been open for
> 41 sprints** — so the blocker was never the decision's difficulty but that
> nobody had asked, which is the same shape T61 later found in "no Docker
> daemon", one layer up. Four tickets shipped (an owner on every Booking,
> cascade-cancel with refunds across two contexts, `no_show_fee` refunds) and
> ADR-0015/ADR-0016 moved Escalated → Accepted in place, with D2 adopted
> verbatim as `CLAUDE.md` rule 9's five-condition reviewer carve-out — precision
> that later let T61 state plainly that its own reviewer-authored fix failed two
> of those conditions. **Its five PRs were open for up to thirteen days and
> merged inside nine minutes**, which is the mechanical cause of both the rebase
> churn the Docs-index row records as a branching fact and the bookkeeping PR
> that had to be **rewritten rather than rebased**, because text saying
> "resolved pending merge" described nothing once the burst landed. Issue count
> 7 → 5. **This retro was written at T61, six sprints late**, from the record
> rather than the sprint, which biases it toward process facts and away from the
> judgement calls a contemporaneous Ceremony 3 would have captured. Two of T55's
> own claims have since been falsified: its Docker disclaimer (false — the suite
> was runnable, and T55.1 altered the very table the no-double-booking invariant
> lives on without re-proving it) and the durability of T55.1's integration
> fixtures, which went stale on merge and were among the five root causes T61
> had to fix.
