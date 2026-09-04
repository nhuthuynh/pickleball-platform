# T54 Sprint Retro

Ceremony 3 per `docs/process/sprint-process.md`, six-role team (briefs:
`docs/agent-operating-handbook.md` Part B), held against
`docs/process/t54-sprint-plan.md`, `docs/process/t53-retro.md` as the
structural precedent, `HANDOFF.md`, ADR-0015, ADR-0016, and the live
issue/PR/commit history.

**This retro breaks the pattern of the previous twenty-four, and says so in
its first sentence rather than burying it:** T54 planned as the
thirty-third consecutive 0-ticket sprint, and would have been one, except
that **DECISION D1 and DECISION D2 were both answered mid-sprint** — the
first answers either has received since they were escalated in T14 and T15
respectively. T54 therefore closes as a **2-decision, 1-ticket sprint**:
PR #290 (the Ceremony 1/2 doc, planned) merged, and T55.1 implemented D1's
answer and closed #144.

---

## 1. What actually happened, in order

1. A session resumed the project, read `HANDOFF.md`, and found the
   backlog in the state T53's retro left it: 7 open issues, all blocked,
   the whole of the blocking traceable to D1 (#144, #149) and to Product
   Owner input the team had repeatedly declined to invent on the user's
   behalf (#124, #126, #130).
2. **Rather than open the thirty-fourth planning ceremony against the same
   unanswered questions, the session put D1 and D2 to the user directly**,
   with the ADRs' own option tables and costs as the question.
3. Both were answered in the same exchange:
   - **D1 → option (a), "authenticate the flow."**
   - **D2 → option (b), "a bounded carve-out," as specified, unrelaxed.**
4. PR #290 was reviewed and merged (docs-only: the T54 Ceremony 1/2 doc
   plus `HANDOFF.md`'s T53 row correction).
5. T55.1 implemented D1's answer end to end and closed #144.

## 2. The finding this retro exists to record

**A correctly-formed escalation sat unanswered for 41 sprints, and no
ceremony in that span was empowered to do anything about it.**

The numbers, so the size is not left to impression:

| | |
|---|---|
| D1 escalated | T14 (ADR-0015) |
| D1 answered | T54 |
| Sprints in between | **41**, T14 → T54 |
| Consecutive 0-ticket sprints ended by the answer | **24** (T30–T53) |
| 0-ticket sprints by total count | 32 |
| Last sprint to merge application code before T55 | **T29.2**, 2026-08-17 |
| Ceremony documents produced in that span | ~50 (a plan and a retro per sprint) |

**What the process got right.** Every one of those ceremonies did what
`sprint-process.md` told it to do, and did it honestly. Not one
manufactured a ticket against a blocked issue. Not one guessed at D1 or D2
on the user's behalf — and for #124/#126/#130, which turn on real product
judgement, that restraint was correct and remains correct. The
bookkeeping was accurate: this retro re-derived T53's counters and found
them right. ADR-0015 in particular is a genuinely excellent artifact — it
states a question a non-engineer can answer in one sentence, lays out four
options with honest costs, corrects a cost the sprint plan had got wrong,
adds a fifth option that correction made available, and explicitly refuses
to express a preference. It is exactly what an escalation should look like.

**What the process got wrong.** It had no mechanism for *delivering* an
escalation to the person who could answer it, and no trigger that fired on
an escalation ageing. ADR-0015 anticipated the failure precisely — *"a
third deferral without an answer is a finding, not a decision"* — and T21's
retro even defined two reopening conditions. But both conditions were
written to detect a *change* (the blocker profile moving, the backlog
running dry), and this situation's defining feature was that **nothing
changed, for 41 sprints**. Every ceremony correctly evaluated the reopening
conditions, correctly found neither had fired, correctly recorded another
0-ticket sprint, and correctly incremented a counter measuring how long it
had been doing so. The counters were not a warning system; they were a
record of one going unheeded. D1's "consecutive-sprint-silence counter"
reached **forty-one** without anything being able to act on that number.

**The one-sentence version:** the team built an excellent instrument for
asking the Product Owner a question and no instrument at all for noticing
that the Product Owner had never been handed it.

**What actually broke the deadlock** was not a ceremony. It was a session
choosing to ask the user the two questions directly instead of opening the
next planning document. That took one exchange. Both answers were given
immediately, and neither required information the user did not already
have — D1's answer had been one question away for 41 sprints.

## 3. Recommendations for T55 and beyond

1. **An escalation must name a delivery mechanism, not just a trigger.**
   ADR-0015's trigger — *"the sprint immediately following the user's
   answer"* — is conditioned on an event no ceremony can cause. Every
   future escalating ADR must additionally record **how and when the
   question reaches the person who can answer it**, and a ceremony that
   opens with an unanswered escalation on the books must put it to the
   user before planning anything else. **Adopt.**
2. **Cap consecutive 0-ticket sprints at two.** A third consecutive
   0-ticket sprint is not a sprint; it is a signal that the team is
   blocked on something outside its authority, and the correct response is
   to escalate to the user and stop holding ceremonies until answered.
   Holding a ceremony that can only conclude "still blocked" consumes real
   effort to produce a document whose content is knowable in advance.
   **Adopt.**
3. **Counters must have thresholds or they are not instrumentation.**
   D1's silence counter reaching forty-one is the clearest possible
   evidence: a number nothing is empowered to act on is bookkeeping, not
   monitoring. Every counter this project maintains should either carry a
   threshold with a defined action, or be dropped. **Adopt.**
4. **Distinguish the two kinds of blocked issue in the backlog.**
   ADR-0015 went out of its way to warn that D1 was *"waiting on nothing
   but the Product Owner's attention"* and must not be filed alongside
   ADR-0012's Q1/Q2, which carry a legal/ethical dimension and may be
   blocked indefinitely. That warning was correct and was nonetheless
   effectively ignored — D1 sat in the same undifferentiated "blocked"
   bucket for 41 sprints. `HANDOFF.md` should track *answerable-now*
   blockers separately from *indefinitely-blocked* ones. **Adopt.**
5. **The remaining Product-Owner-blocked issues (#124, #126, #130) should
   be put to the user as explicit questions, in ADR-0015's format**, rather
   than waiting for another ceremony to observe they are still blocked.
   Recommendation 1 makes this mandatory going forward; these three are the
   existing backlog it should be applied to first. **Adopt — carried into
   `HANDOFF.md`'s T55 backlog.**
6. **Do not add a retro finding about the ceremony documents' length.**
   Considered and rejected: the documents are long because they
   re-derive their claims rather than trusting prior ones, which is
   CLAUDE.md rule 10 working as intended and caught real errors in the
   past. The defect was never that the ceremonies were verbose; it was
   that they could not escalate. Fixing the wrong thing here would cost the
   project its verification discipline and address nothing.

## 4. Sweep and bookkeeping

- **Open issues at T54's start:** 7 (#124, #126, #130, #134, #144, #145,
  #149) — matching T54's plan §A1 and T53's retro.
- **Closed during T54/T55.1:** **#144**, by the D1 implementation. First
  issue closed since T29.
- **Open at close:** 6.
- **PRs merged:** #290 (T54 Ceremony 1/2 doc), plus T55.1's implementation
  PR.
- **D1 silence counter:** ended at **forty-one**, and is now **retired** —
  D1 is answered, and per recommendation 3 a counter with nothing left to
  count is dropped rather than carried.
- **Post-T29 backlog-composition counter:** **fifty**, and likewise
  retired — the backlog composition it tracked has changed for the first
  time since T29.
- **Stale repo-metadata artifact** (GitHub returning the old
  Vinyl-Trading description for this repository): re-observed, still
  present, still functionally inert — pushes and merges both succeeded.
  Not chased, per the standing instruction.

## 5. Honest-form outcome sentence

For `HANDOFF.md`'s T54 row, to be carried verbatim rather than strengthened:

> T54 was planned as the thirty-third 0-ticket sprint and did not become
> one. DECISION D1 (ADR-0015) and DECISION D2 (ADR-0016) were both put to
> the user directly and both answered — D1 as option (a), "authenticate the
> flow"; D2 as option (b), the bounded carve-out, adopted verbatim and
> unrelaxed. PR #290's planning document merged as written, and T55.1
> implemented D1's answer end to end: `domain.Booking` gained a required
> `OwnerUserID`, `bookings.owner_user_id` became
> `uuid NOT NULL REFERENCES identity_users (id)` (migration 0027),
> `CreateBooking` and `CancelBooking` moved from `PublicMethods()` to
> `AuthenticatedMethods()`, and #144 — open since T13, the sharpest
> object-level authorization hole in the codebase — was closed. The
> accepted cost is the one ADR-0015 option (a) states: the shipped T7.6
> public quote-and-book flow now requires an account at its confirm step,
> a conversion call the Product Owner made with the cost in front of them.
> This retro's substantive finding is not about the ticket but about the
> process that preceded it: a correctly-formed escalation sat unanswered
> for 41 sprints and 24 consecutive 0-ticket ceremonies, because the team
> had an excellent mechanism for *asking* the Product Owner a question and
> none at all for *delivering* it or for noticing it had gone unanswered.
> Five process recommendations are adopted in response; the sixth
> candidate — shortening the ceremony documents — was considered and
> deliberately rejected as fixing the wrong thing.
