# T57 Sprint Retro

Ceremony 3 per `docs/process/sprint-process.md`, six-role team (briefs:
`docs/agent-operating-handbook.md` Part B), held against `HANDOFF.md`,
`docs/process/t56-retro.md` as the immediate precedent, PR #300, and the
live issue/PR/commit history.

**T57 held no planning ceremony**, so there is no `t57-sprint-plan.md`. Its
scope came from one line in T56's own code: the paragraph in
`ensureAmountMatchesPayable` naming `competition_entry` as out of scope.

**Outcome: 2 tickets, 1 issue half-closed, 0 issues opened.** T57.1 and
T57.2 merged as PR #300 (`bc3668c`), closing #126's remaining Competitions
half.

---

## 1. What actually happened, in order

1. T56 shipped per-head pricing and amount validation for Social Play and
   named Competitions out of scope, in writing, in the code.
2. T57 took that exclusion as its whole scope: `ExpectedAmount` and a frozen
   `CompetitionEntry.AmountOwed` (migration 0029), plus `EntryAmountLookup`
   and the validation branch that uses it.
3. **Investigation found the defect was sharper in Competitions than in
   Social Play** — see §2.
4. `PayableAmountLookup` was renamed `RegistrationAmountLookup` and a
   sibling `EntryAmountLookup` added, one port per context.
5. The branch was stacked on #298's pre-squash branch; after #298
   squash-merged it was rebased `--onto` the shared branch and retargeted
   before merge.
6. The pre-merge review carried T56's wire-test finding across, and found
   that a straight copy of Social Play's test would have been insufficient
   here (§3).

## 2. The finding this retro exists to record

**A scope exclusion is a dated object. It is defensible on the day it is
written and becomes an undocumented inconsistency the moment the sprint
ends, and nothing in this project's process notices the transition.**

T56.2's exclusion was correct when written: Competitions had no frozen owed
amount, so there was nothing to validate against. That reason was stated in
the code and on the issue.

But for the span of one sprint the system was in this state:

| | Registration | CompetitionEntry |
|---|---|---|
| Underpay by 99% | refused | **accepted, marked paid in full** |

Two payable types, the same shape, opposite behaviour, with nothing but a
ticket boundary explaining the difference. A reader encountering that in
month six would not find "T56 ran out of scope" — they would find two rules
and assume one was deliberate.

**The sharper version, specific to Competitions.** `domain.Enter` sums
`(1 + GuestCount)` and `enforce_competition_capacity()` does the same in
Postgres. So a Competition was **correctly sold out by four heads while
being correctly billed for one** — the capacity rule and the pricing rule
sat in the same aggregate, derived from the same field, disagreeing. That is
not a gap anyone introduced; it is a gap nobody noticed because the two
rules were written by different tickets and never read side by side.

**The process question this raises, unresolved.** `sprint-process.md` has a
mechanism for a *disclosed gap* (file an issue — the board-of-record rule)
and for an *unanswered escalation* (T55's new rule). It has none for "a
deliberate exclusion that is correct today and rots tomorrow." T56's
exclusion did produce an issue (#297 named it), which is why T57 happened
at all — but that was the author's diligence, not a rule. See
recommendation 1.

## 3. Mirroring is not copying — the review's contribution

The pre-merge review carried T56's wire-test finding over: nothing pinned
`amount_owed` to the wire, and deleting the line would leave every gate
green.

**A straight copy of Social Play's test would have been insufficient.**
`toProtoEntry` has a *second* call site, reached by
`ListEntriesForCompetition` — the Host's roster read, which is what
`CompetitionManage.vue` uses to tell a Host what an unpaid entry owes in
cash. Social Play's equivalent has no such second path. A mirrored test
would have covered the write and left the Host-facing read unproven.

Recorded because "mirror the other context" is this project's standard
instruction (CLAUDE.md: *"mirror the `booking` context exactly"*), and it
is good advice that quietly assumes the two contexts have the same shape.
Here they did not.

The same review also found the roster **display** carried the identical
per-entrant bug: a Host chasing cash from a party of three was told to
collect $25.00 when $75.00 was owed. Display-only on that screen — there is
no "Mark paid" button for competition entries — which is exactly why
nothing downstream contradicted it.

## 4. What went well

- **Scope came from a written exclusion, not from a planning guess.** T56
  said what it was not doing, in the code, and T57 did exactly that.
- **The stacked-branch mechanic worked and is now routine.** Rebase `--onto`
  after the base squash-merges, retarget, verify `mergeable_state: clean`.
  This is the second sprint running it (T55 did it twice); it no longer
  costs thought.
- **The rename was taken rather than deferred.** `PayableAmountLookup`
  promised to resolve any *payable* while answering only for Registrations.
  Naming debt created in T56 and paid in T57 — one sprint of exposure, and
  the alternative was a name that misleads permanently.
- **Two ports, not one, with a test pinning the reason.** The tempting
  simplification (one shared nil guard) would have made an unwired
  *Competitions* lookup silently stop validating **Registrations**. That is
  a security regression wearing a refactor's clothes, and
  `TestCreateOnlinePayment_TheTwoAmountLookupsAreIndependent` exists so
  nobody makes it later in good faith.

## 5. Recommendations for T58 and beyond

1. **A deliberate scope exclusion in shipped code should carry its own
   tracked issue, the way a disclosed gap already does.** T56's exclusion
   happened to be covered because #297 mentioned it; that was luck. The
   board-of-record rule (`t12-sprint-plan.md` §A5/§A7) should extend from
   "a gap you found" to "a gap you chose to leave" — they have identical
   half-lives.
2. **When mirroring a change into another context, enumerate that
   context's call sites rather than the source context's.** §3's roster read
   is the concrete case. One `grep` for the translator function's callers
   would have found it.
3. **State a cross-context asymmetry in both places or neither.** T56.2's
   exclusion was documented in Payments. Competitions' own code said
   nothing about why its entries were unvalidated. A reader starting from
   Competitions had no thread to pull.

## 6. Sweep and bookkeeping

- **Issues: 4 open at sprint end.** #126 closed on this PR (its Competitions
  half; the Social Play half closed in T56). Nothing opened. Live-verified:
  #296, #149, #145, #134.
- **Merge order #298 → #300**, verified by merging in that sequence rather
  than inferred from numbering.
- **Self-review, again.** GitHub refuses an author's own approval; PR #300
  carries a review comment. The review's contribution here was real (§3) but
  it remains one party checking their own work.
- **Docker unavailable for the whole sprint.** Integration tests compile,
  never executed. Two consecutive sprints now. Nothing in T57 is described
  as proven under concurrency.
- **This retro does not update `HANDOFF.md`'s T57 Docs-index row** — per
  `sprint-process.md` a retro PR structurally cannot write a row that must
  cite the retro's own merge number. T58's Ceremony 1 owns it.

## 7. Honest-form outcome sentence

For `HANDOFF.md`'s T57 row, to be carried verbatim rather than strengthened:

> T57 held no planning ceremony; its scope was one paragraph of T56's code
> — the exclusion naming `competition_entry` out of scope. It mirrored
> per-head pricing and amount validation into Competitions, merged as PR
> #300 (`bc3668c`), closing #126's remaining half. The mirror found the
> defect sharper here than in Social Play: `domain.Enter` and
> `enforce_competition_capacity()` both weight guests, so a Competition was
> **correctly sold out by four heads while being billed for one** — the
> capacity rule and the pricing rule lived in the same aggregate, read the
> same field, and disagreed. This retro's substantive finding is about the
> exclusion rather than the code: a scope exclusion is correct on the day
> it is written and becomes an undocumented inconsistency when the sprint
> ends, and for one sprint an underpayment was refused for a Registration
> and accepted for an entry with nothing but a ticket boundary explaining
> it. `sprint-process.md` has a rule for a disclosed gap and none for a
> chosen one; three recommendations address that. The review found that a
> straight copy of Social Play's wire test would have been **insufficient**
> — `toProtoEntry` has a second call site behind the Host's roster read,
> which Social Play has no equivalent of — and that the roster display
> carried the same per-entrant bug, telling a Host to collect $25.00 where
> $75.00 was owed. `PayableAmountLookup` was renamed
> `RegistrationAmountLookup` alongside a new `EntryAmountLookup`: one port
> per context, naming debt created in T56 and paid within one sprint.
> Docker was unavailable throughout — two consecutive sprints now — so the
> integration tests compile and have never been executed, and nothing in
> T57 is described as proven under concurrency.
