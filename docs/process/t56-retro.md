# T56 Sprint Retro

Ceremony 3 per `docs/process/sprint-process.md`, six-role team (briefs:
`docs/agent-operating-handbook.md` Part B), held against `HANDOFF.md`,
`docs/process/t54-retro.md` as the structural precedent, PR #298, and the
live issue/PR/commit history.

**T56 held no planning ceremony, deliberately**, so there is no
`t56-sprint-plan.md` for this retro to be held against. It continued T55's
session directly: the backlog's product questions had been answered the day
before, so the sprint built rather than planned. That is the posture
`sprint-process.md`'s escalation rule now requires — ask, or build; do not
open a ceremony to restate a backlog you already know.

**Outcome: 2 tickets, 2 issues closed, 1 issue opened.** T56.1 and T56.2
merged as PR #298 (`62e6af3`), closing #126's Social Play half and #297.
#299 was opened.

---

## 1. What actually happened, in order

1. The session picked up #126 ("add per-Game pricing") from T55's answered
   backlog and began scoping it against the tree.
2. **Scoping found half the ticket was already built — and had been
   before the ticket was written.** Its headline ask — a real per-Game
   price field, retiring T8.10's `PLACEHOLDER_REGISTRATION_FEE_CENTS` —
   shipped in **T9.2** (`0013_socialplay_entry_fee.sql`, 2026-08-05), nine
   days before #126 was opened (2026-08-14). See §2: this is not staleness,
   it is an observation quoted across sprints without re-checking.
3. What was genuinely unbuilt was the Product Owner's 2026-09-04 answer:
   **per head, including guests**. Nothing multiplied the fee by party size
   anywhere in the stack, so a player bringing three guests paid for one.
4. The session put one further question to the Product Owner — what happens
   when a registration's guest count changes after payment — and was told
   **freeze, and lock guests once paid**.
5. While scoping, a second defect was found and filed as **#297**: payment
   amounts were caller-supplied and compared to nothing, so a $25.00 Game
   could be paid for with one cent. The Product Owner chose to fold its fix
   into the same change.
6. T56.1 (frozen `Registration.AmountOwed`, migration 0028) and T56.2
   (`CreateOnlinePayment` validates against it) were built TDD-first and
   merged as PR #298.
7. **The pre-merge review found a defect**, described in §3.

## 2. The finding this retro exists to record

**An observation was carried across three sprints without re-verification
and then filed as current. It was already false on the day the issue was
opened.**

The dates, checked rather than assumed:

| | |
|---|---|
| `0013_socialplay_entry_fee.sql` added (T9.2) | **2026-08-05** |
| #126 opened, at T12's Ceremony 1 | **2026-08-14** |

#126's body asserts that `domain.Game` and `domain.Registration` have *"no
price/fee field **at all** — confirmed by inspection at T8.10."* That
inspection was genuine and correct **at T8.10**. T9.2 then added the field.
Nine days and three sprints later the observation was written into a new
issue as a present-tense fact, without being re-checked.

So this is not a ticket that went stale in the backlog. **It was wrong when
it was filed**, and the cause is narrower and more fixable than "backlogs
rot": a finding made in one sprint was quoted in another as though findings
do not expire.

Every Ceremony 1 sweep from T30 onward then re-verified it as "blocked,
unchanged" — **23 of them state, verbatim in `HANDOFF.md`, that they
"re-verified all 7 open issues' blockers live down to their full bodies."**
(That is a `grep -c` of narratives carrying that exact sentence, not a count
of sprints inferred from a range; T30–T54 is 25 sprints and not all phrase
the sweep identically.) Each confirmed the issue's *blocker*. None
confirmed its *premise*.

That distinction is the second half of the finding. The sweep asks "can
anyone act on this yet?" and never asks "is this still describing reality?"
For a blocked issue those look like the same question, and they are not: an
issue can remain perfectly blocked while the thing it describes is quietly
built by someone else — or, as here, while it was never true to begin
with.

What saved the sprint was ordinary practice, not process: the session read
the tree before writing code. Had it trusted the ticket — which is exactly
what a well-written ticket invites — it would have delivered a migration and
a domain field that already existed, and the redundancy would have been
discovered by whoever next opened `0013_socialplay_entry_fee.sql`.

**Why this is not a LESSONS.md entry.** No incident occurred; the waste was
avoided. `docs/LESSONS.md` is for postmortems of things that went wrong (its
own header says so). This is a retro finding about a sweep's blind spot.

## 3. The review finding, and why it is recorded here too

The pre-merge review of #298 deleted the single line putting `amount_owed`
on the wire and re-ran the gates:

```
test-domain:   PASS
test-adapters: PASS
test-cmd:      PASS
```

Every Go gate green with the feature entirely broken. A client would have
read `0`, sent `0`, and **this sprint's own validation would have refused
every payment it existed to permit**.

The part worth recording is not the gap but its neighbourhood. The guard
already existed one field earlier: `TestCreateGame_EntryFeeRoundTrip` was
written for precisely this failure mode — *"survives the full wire → app →
domain → wire path, rather than being silently dropped by a translation
layer"* — on the **previous** field added to that same proto message. The
next field added to it had no such test.

**A convention that lives only in one test's doc comment is not a
convention.** It is a note that happens to be adjacent to the next person's
work, and only if they read that file.

## 4. What went well, stated as specifically as the findings

- **The Product Owner question was asked in the right place.** "What happens
  when guests change after payment" was put *before* building, not
  discovered after. The answer ("freeze, lock once paid") then turned out to
  need **no machinery at all** — `GuestCount` has no mutation path, so the
  invariant holds by construction. The sprint built nothing and documented
  why, which is the correct response to a question whose answer is already
  structurally true.
- **The already-built half of #126 was reported, not quietly absorbed.** It
  would have been easy to describe the sprint as "delivered #126" and let
  the T9.2 overlap go unmentioned — nobody was checking, and the sprint
  would have looked larger.
- **Tests were written first and the defect reproduced before the fix**, on
  both tickets.

## 5. Recommendations for T57 and beyond

1. **The Ceremony 1 issue sweep should re-verify an issue's premise, not
   only its blocker** — for every issue, not only old ones. #126 was false
   on the day it was filed, so an age threshold would not have caught it.
   One question added to a sweep that already reads each issue's full body.
2. **An issue quoting a finding from an earlier sprint must re-verify it at
   filing time, and say when it was checked.** #126 quoted a T8.10
   inspection at T12 in the present tense. Had it said "as of T8.10", the
   staleness would have been visible on the page. This is the narrower,
   more actionable half of §2.
3. **When a proto message gains a field, the round-trip test is part of the
   ticket, not a reviewer's catch.** T9.2 wrote one and T56 did not, for no
   reason either author could have articulated. Adopted in practice by
   T56's own fix; worth stating so T57 does not need its reviewer to find
   it again.
4. **Prefer deleting a line and re-running the gates over reading code to
   decide whether a test exists.** The review found §3's gap by mutation,
   after reading had suggested the coverage was fine.

*(Recommendation 4 is deliberately narrow. A general "mutation-test
everything" rule was considered and rejected: it would be ignored. Applied
to a single new wire field, it is a ten-second check.)*

## 6. Sweep and bookkeeping

- **Issues: 5 → 4** across T56–T57 (#126 needed both halves). Within T56:
  #297 opened, #126's Social Play half and #297 closed by PR #298.
- **`Closes #N` did not fire and structurally cannot** — PRs merge into
  `claude/go-backend-pickleball-7up34j`, not the default branch. Both
  closures were manual, as every closure on this project has been.
- **The self-review posture is unchanged and still unsatisfying.** GitHub
  refuses an author's own approval, so PR #298 carries a review *comment*,
  not an approval. The review found a real defect, which is evidence the
  form has value — but no second party has looked at this diff.
- **Docker was unavailable for the whole sprint.** The integration tests
  compile and were never executed. Per CLAUDE.md rule 10, nothing in T56 is
  described as proven under concurrency.
- **This retro does not update `HANDOFF.md`'s T56 Docs-index row.** Per
  `sprint-process.md`, a retro PR structurally cannot: the row must cite the
  retro's own merge PR number, which does not exist until it merges. That
  row is T57's Ceremony 1's job.

## 7. Honest-form outcome sentence

For `HANDOFF.md`'s T56 row, to be carried verbatim rather than strengthened:

> T56 held no planning ceremony and built directly from T55's answered
> backlog. It delivered per-head pricing for Social Play (#126's half) and
> server-side amount validation (#297), merged as PR #298 (`62e6af3`).
> Scoping found #126's headline ask — a real per-Game price field — had
> shipped in **T9.2, nine days and three sprints before the issue was
> opened** (`0013_socialplay_entry_fee.sql`, 2026-08-05; #126, 2026-08-14).
> The issue was not a ticket that went stale: it quoted a T8.10 inspection
> in the present tense at T12 without re-checking it, so it was **already
> false when filed**. Every Ceremony 1 sweep from T30 onward then
> re-verified it as "blocked, unchanged" — 23 saying so in identical words
> — each confirming its blocker and none its premise. Only the Product Owner's "per head, including guests"
> was genuinely unbuilt. The owed amount is **frozen** onto the
> Registration rather than derived, so a Host raising the fee afterwards
> cannot retroactively change what a player agreed to pay — which is also
> what makes #297's validation mean anything. "Lock guests once paid" was
> asked before building and needed no machinery: `GuestCount` has no
> mutation path, so it holds by construction, and the sprint documented
> that rather than building a guard for an operation that does not exist.
> The pre-merge review found, **by deletion rather than by reading**, that
> nothing tested whether `amount_owed` reached the client: removing one
> line left every Go gate green while the feature was entirely broken, and
> the round-trip test that would have caught it already existed one field
> earlier and had not been followed. Fixed before merge and disclosed in
> the review. Docker was unavailable throughout, so the integration tests
> compile and were never executed; nothing in T56 is described as proven
> under concurrency.
