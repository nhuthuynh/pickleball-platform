# T9 Sprint Retro

Ceremony 3 per `docs/process/sprint-process.md`, six-role team (briefs:
`docs/agent-operating-handbook.md` Part B), held against
`docs/process/t9-sprint-plan.md`, `HANDOFF.md`'s T9 sections, the T9
entry in `docs/LESSONS.md`, and issues #70–#79 / PRs #81–#92 on
`nhuthuynh/white-label` (GitHub-side name `pickleball-platform`).

Merge order and every timestamp cited below were pulled from each PR's own
`merged_at` and each review's `submitted_at` rather than assumed — one
finding (6) exists specifically because the record contradicted what the
team expected to find.

Findings are not a single voice. Recorded disagreements are left as
disagreements per the "do not manufacture consensus" rule, matching T6's
and T8's sprint plans.

**Sprint outcome:** all 10 tickets (49 points) merged, sprint goal met,
plus one critical out-of-band fix (PR #89). First ticket PR opened
06:09:07, last feature PR merged 13:37:27 — the project's fastest sprint
by throughput. That is the context for finding 8, not a reason to skip
the rest.

---

## 1. The shared-checkout collision was self-corrected by five agents independently, and written down by none of them

**PdE/PE (the facts).** Batch 1 dispatched five implementers
simultaneously into one unisolated working directory: PRs #81 (T9.8,
created 06:09:07), #82 (T9.9, 06:14:28), #83 (T9.10, 06:16:14), #84
(T9.1, 06:17:46), #85 (T9.2, 06:28:23) — five branches opened inside a
19-minute window. They stepped on each other. Every implementer
independently diagnosed it and recovered by moving to an isolated git
worktree; no work was lost, and all five merged inside seven minutes of
each other (07:03:57–07:09:16). The process gap was real — agents were not
dispatched with worktree isolation initially — and it was fixed mid-sprint:
later dispatches used `isolation: "worktree"`. Traces of the recovery
survive in the record: PR #84's review notes it was "reproduced in an
isolated worktree at `04756dd`", PR #86's "reproduced independently in an
isolated worktree at `ff4a776`".

**QA (the finding).** `sprint-process.md`'s execution-loop-mechanics
section, item 2, says a mistake caught during a loop "gets a line in
`docs/LESSONS.md` **immediately** if it's the kind of mistake that could
recur on a different ticket in the *same* sprint — don't wait for Ceremony
3." This mistake did not merely *could* recur on a different ticket in the
same sprint; it recurred on **five tickets simultaneously**, which is the
most literal possible instance of the case that rule describes. It got
zero lines. Verified by grep across all of `docs/` and `HANDOFF.md` before
this retro: the collision is mentioned nowhere. The only reason it is
written down at all is this retro — held at the end of the sprint, which is
exactly what that rule exists to prevent.

**Recorded disagreement — PE vs. QA on whether this matters.**

- **PE:** no work was lost, five agents each diagnosed and fixed it without
  being told to, and the systemic fix (worktree isolation on subsequent
  dispatches) landed *within* the sprint. That is a self-correcting system
  behaving as designed. Writing it down after the fact is bookkeeping, not
  risk reduction — the risk was already retired by the dispatch change.
- **QA:** "no work was lost" is CLAUDE.md rule 10's own fallacy wearing a
  process costume. Five agents concurrently mutating one checkout is a
  data-loss race; a benign outcome on one run is not evidence about the
  next, and that is precisely the reasoning this project already codified
  after T4. The recovery was not one fix, it was **five independent
  rediscoveries of the same fix** — five times the cost of one written
  line, paid by five agents who each had to work it out from confusing
  symptoms. And the dispatch change was made by whoever happened to be
  dispatching, not recorded as a rule, so it binds nothing next sprint.
- **PdE** sides with QA on cost specifically: the rediscovery tax is real
  and measurable, whatever one concludes about the data-loss risk.
- **PM** notes for balance that the sprint still delivered 49 points on
  schedule, so whatever this cost, it did not cost delivery.

**Not resolved.** Recorded as a genuine split. Both sides agree on the
concrete change below, which is why the disagreement does not need
resolving to act.

**Change for T10:** dispatch isolation becomes an explicit Ceremony 2
checklist item — *how many implementers run in parallel, and is each one
isolated?* — rather than a per-dispatch judgment call made in the moment
by whoever is dispatching. This is the same remedy shape T5's retro
finding 1 chose for a different recurring gap: move the check from review
time to planning time.

## 2. The panic-recovery incident: found by accident, not by testing

**Cross-reference, not restated.** Full postmortem —
root cause, why nine phases missed it, the two-layer fix, and the
actionable lesson — is `docs/LESSONS.md`, `## T9 (2026-08-05) — grpc
installs no panic recovery; net/http intuition does not transfer`. That
entry was owed by PR #89's own description and is written as part of this
retro's output.

**What belongs here rather than there — two sprint-level observations.**

**QA:** the discovery route is the finding. One `curl` to a malformed ID
crashed the entire server process, across all five bounded contexts, from
an unauthenticated public read — and it was found because a PE+QA reviewer
working on PR #88 (T9.5) looked at the read path *next to* the one they
were reviewing. No ticket in T9 would have found it. No ticket in T0–T8
did. The sprint's most severe defect was found outside the loop the
process defines, by a reviewer exceeding their ticket's boundary. That is
a good behaviour to have and a bad thing to depend on.

**PE:** a sequencing fact worth recording without overstating it. The
crash was known from 08:37:34 — PR #88's review raised it as finding F2,
correctly scoped *out* of that ticket since `mustUUID`/`GetByID` were
unchanged T9.4 code — and #88 merged 5 minutes later at 08:42:58. The fix
merged at 13:29:10. So the shared branch knowingly carried an
unauthenticated total-outage crash for **4h52m**, during which two further
feature PRs (#90, #91) were built on top of it. Nothing was deployed and
no environment was exposed, so the practical severity was nil; the
decision to merge #88 with the defect disclosed rather than block an
unrelated ticket on someone else's bug was also correct. It is recorded
because "a known critical defect sat on the shared branch for five hours"
is a sentence the team should have to look at, not because anyone acted
wrongly. **PO** agrees and declines to make it a rule: a blocking gate here
would have penalised the ticket that *found* the bug.

## 3. The regression test for T9.5's NUL-byte oracle was vacuous on its first attempt — and the reason generalises

**QA (the bug).** PR #88's PE+QA review broke the PR's own headline
security claim. The endpoint promised — in the proto doc comment, the
`port.Repository` contract, the `app.Service` doc comment, and the PR body
— that unknown, malformed, and empty share tokens all produce a
**byte-identical** `NotFound`. A token containing a NUL byte does not.
`\x00` is valid UTF-8 and therefore a legal proto3 `string`, so it passed
grpc-gateway, passed the `== ""` guard, and reached Postgres, where a
`text` column cannot hold it: SQLSTATE 22021, surfaced as a 500 carrying
raw driver text, adapter package name and all, to an anonymous caller on a
public endpoint whose entire stated job is to disclose nothing. Reproduced
3/3 against a real server on real Postgres 16. Twelve adversarial shapes
were probed; ten returned the bare sentinel correctly — SQL metacharacters,
LIKE wildcards, case-flips, Unicode homoglyphs, a 100k-character token —
and exactly two leaked, both encoding-shaped.

**PE (the methodology finding, which is the more valuable half).** The
obvious regression test — add a NUL-byte row to the existing fake-backed
probe table — **is a no-op**. The in-memory `fakeRepository` is a Go map,
and a Go map indexes a NUL-byte string exactly like any other, so the
assertion passes identically with and without the fix. This is not
hypothetical: the merged test's own doc comment records the confirmation
that "reverting the fix in `internal/competitions/app/service.go` still
passes every other test in this file."

The author resolved it by asserting the security property directly rather
than its symptom — `TestGetCompetitionByShareToken_MalformedShapeNeverReachesRepository`
asserts that a malformed-shape token **never reaches the repository call at
all** — and documented it as a deliberate exception to the QA dossier's
"don't write change-detector tests against a call log" rule, on the
grounds that here the call not happening *is* the property, not an
implementation detail a refactor might reasonably change. The team agrees
that is the right resolution and that the explicit justification in the
test comment is what makes it safe from a future "cleanup."

**QA (the generalisation, and why it earns a rule).** A fake that cannot
produce the failure mode under test makes the regression test a no-op, and
a **no-op regression test is worse than no test, because it reads as
coverage**. This project has now hit the fake-fidelity boundary three
times and named it as a general rule zero times:

- **T4:** a single successful concurrency run against a real DB was
  generalised to "proven reliable"; the intermittent `40P01` deadlock was
  exactly what one run misses. → rule 10.
- **T5.4:** a domain-level count-then-insert check passed every unit test
  with a fake repository, and had a TOCTOU window the moment two real
  requests interleaved.
- **T9.5 (this one):** and, in the same sprint, **PR #89's root cause was
  the same family** — fixtures minting `"id-1"`, `"court-1"`, `"g-1"`,
  shapes the real generator never produces and Postgres cannot store,
  which is precisely why no existing test could see a process-killing
  crash. Notably, CI would not have caught that either; the fixture
  infidelity, not the missing CI, is what hid it.

**Recommendation, adopted:** when a regression test is written for a bug
whose failure *originates in infrastructure* — Postgres, the wire, the
driver, the encoder — the test must either run against that
infrastructure, or assert a property observable without it, and the PR
must state which of the two it did. Never assert on a return value that a
fake produces identically in the fixed and the broken version. **BA** adds
the diagnostic question that makes this checkable in review: *what did you
change to confirm this test fails?* — the T9.5 author could answer it, which
is why the vacuity was caught rather than shipped.

## 4. T9.4's status-code defect was in the ticket text, not the code

**BA.** The sprint plan's own T9.4 instruction (line 807) reads: `toStatus`
maps `ErrCompetitionFull` → `FailedPrecondition`/409-shaped. **Those are
two different things.** `FailedPrecondition` does not map to 409 through
grpc-gateway; `AlreadyExists` does. The sentence names a gRPC code and an
HTTP shape that contradict each other, and reads as consistent because
each half is individually plausible. The implementer followed the named
code. The joint PE+QA review (submitted 08:07:35) returned
REQUEST_CHANGES scoped to exactly that one mapping, recommending
`AlreadyExists`/409 — everything else in an 8-point ticket was
approve-quality, with the capacity-guard and authz proofs independently
reproduced against real Postgres. Fixed before merge at 08:10:06 (the
merged head differs from the reviewed commit).

This is the BA role's canonical shape — two clauses that sound consistent
and do not compose — and it is the **second sprint running** that the
canonical shape appeared in *ticket text written at Ceremony 1* rather
than in a spec: T5's retro finding 1 was the same structure (a ticket
naming the DB-mirror requirement for uniqueness but not for capacity,
caught by review after a full implement pass).

**PE** notes the same error class recurred downstream in the same sprint,
which is the argument for fixing it at the source: T9.7's review found the
`ErrCompetitionFull` **proto comment** now stale, because
`ErrCompetitionFull` and `ErrAlreadyEntered` both map to
`AlreadyExists`/409 and the status code alone therefore cannot
disambiguate them — the client has to string-match. A contradiction
introduced at planning time propagated into a doc comment and a client
workaround before anyone reconciled it.

**Change for T10, adopted:** a Ceremony 1 ticket-writing rule — when a
ticket specifies error handling, name the **gRPC code only**. The
gateway's code→HTTP mapping is the authority; restating an HTTP status
beside it adds no information and creates exactly one opportunity for the
two to disagree. Where an HTTP shape genuinely is the requirement (a
public REST contract), name it *instead of*, not alongside, the gRPC code.

## 5. T9.6/T9.7 converged independently on the PayableType gap — a strong positive, and the sprint's second late cross-context surprise

**PdE (the positive, and it is a real one).** Two UI implementers working
in parallel from separate tickets independently traced the same path and
reached the same conclusion: the online-checkout hop for Competition
entries could not be safely wired. `payments.PayableType` has exactly
three members (`booking`, `registration`, `no_show_fee`), none for a
competition entry; `IsValid()` closes the set and `NewPayment` rejects
anything else. Of the three available choices, **all three are wrong** —
`UNSPECIFIED` is rejected outright, `BOOKING` keys authorization to
`BookingHostID` in the wrong context, and `REGISTRATION` is the dangerous
one: `reconcileRegistrationPaymentStatus` passes `p.PayableID` through
**unvalidated** to Social Play's registration updater, *after* the Payment
has already been transitioned and persisted. Routing a Competition entry
through the existing checkout would have marked a Payment paid and then
written a Competition entry ID into another context's table at confirm
time. Money-adjacent data corruption, not extra work skipped.

Both PRs' reviews independently re-derived the trace from source rather
than trusting the implementer's claim. **Two independent implementations
plus two independent reviews reaching the same conclusion is the strongest
corroboration available without a third party** — and it corroborated a
*refusal to build*, which is the harder call to make and the easier one to
quietly skip. PdE and PE both want this recorded as a positive without
qualification.

**PM (the flag).** This is the **second unplanned cross-context gap this
sprint surfaced late**, after the panic-recovery one. Both were found in
the last third of the sprint, by review, not by planning. T9.3 and T9.4
both built the Competitions app service, its ports, and its adapters —
they were the tickets in the best position to ask what Competitions would
need from Payments — and neither did. A cross-context dependency check at
Ceremony 1 (*which existing contexts must this feature call, and does the
enum / port / type it needs actually have a member for this context?*)
would have surfaced `PayableType` while it was still a ticket to write,
rather than at T9.6/T9.7, where it became a scope cut decided mid-sprint.

**Recorded disagreement — PE vs. PM on whether "late" is the right word.**

- **PE:** the absence is not a bug and not an oversight. `PayableType`'s
  own doc comment states that values are added "in the same PR that first
  produces one" — a deliberate design stance against speculative enum
  members. Discovering the gap at the UI layer is discovering it exactly
  when a producer first exists, which is the policy working, not failing.
  A planning-time check would have found a gap that policy says should not
  be closed until that moment anyway.
- **PM:** that defends the *policy*, not the *timing*. The sprint goal
  committed to a Player entering a Competition "at a real entry fee," and
  cash-only was settled by two implementers mid-sprint rather than by the
  team that owns scope. Knowing at Ceremony 1 that Competitions would ship
  cash-only would not have changed the code — it would have changed the
  sprint goal's wording and the Advertise/Entry UI copy, both of which are
  PM's to own.
- **PO's tiebreak, recorded rather than imposed:** the outcome was right
  and the route to it was not visible to the people who own scope. The
  adopted change is PM's cross-context check at Ceremony 1, which costs
  little and does not require reopening PE's enum policy.

**PO (the recurring one).** HANDOFF.md records the recommended follow-up —
a Competitions-shaped `PayableType` plus the port/adapter pair already
established for Social Play in T6.5. It is **still not a GitHub issue**.
Same for the write-handler malformed-ID validation from PR #89, and the
host/venue display-name join both UI reviews flagged. That is three
untracked follow-ups from T9 living only in prose — the identical shape as
T5's retro finding 6, two sprints later, unimproved. **Recommendation
(second time of asking):** T10's Ceremony 1 opens these three as real
issues before scoping anything new.

## 6. Review-then-merge sequencing: the caveat `sprint-process.md` needs is real, but it is not the one we expected

The team went looking for evidence that tickets were merged ahead of
in-flight reviews. **The record does not support that.** Verified against
`merged_at` and review `submitted_at` on six PRs — including all three
that drew a blocking verdict:

| PR | Review submitted | Merged | Gap | Verdict |
|---|---|---|---|---|
| #84 (T9.1) | 07:05:37 | 07:06:17 | 40s | APPROVE |
| #86 (T9.3) | 07:28:04 | 07:28:40 | 36s | APPROVE |
| #87 (T9.4) | 08:07:35 | 08:10:06 | 2m31s | REQUEST_CHANGES |
| #88 (T9.5) | 08:37:34 | 08:42:58 | 5m24s | REQUEST_CHANGES |
| #90 (T9.7) | 13:17:57 | 13:29:29 | 11m32s | APPROVE |
| #91 (T9.6) | 13:21:52 | 13:37:27 | 15m35s | REQUEST_CHANGES |

In every case a review was submitted before the merge, and in all three
REQUEST_CHANGES cases a fix commit landed between the reviewed commit and
the merged head — the blocking finding was addressed, not overridden. Even
the docs PR held the line: #92 (HANDOFF update) drew a PO review that
found four inaccuracies, fixed in `6967ae6` before merge. **The process
worked as written, and the expected finding is withdrawn rather than
massaged into a smaller version of itself.**

**PE (what the record does show).** The merge followed the review by
between **36 seconds and 15 minutes**. That is fast enough that the review
functioned as a gate on the *implementer* — it caught real defects and
they were really fixed — but not as an input to a *human* decision. There
was no window in which anyone could read a 4,000-word adversarial review
before merging on its verdict. So the sprint's actual safety property was
"the agent review was correct," not "a human checked the agent review,"
while `sprint-process.md`'s Execution item 4 ("approved and merged by the
user or an explicitly delegated gate") reads as though independent human
judgment intervenes at that step. The doc describes a mode the sprint did
not run in.

**PO** is explicit that this is **not a rule-9 violation and not a
criticism**: the user owns the repo, merged deliberately, and every merge
was preceded by a substantive review recommending it. The gap is in the
documentation, not the conduct.

**Adopted:** write the real operating mode into `sprint-process.md`'s
Execution section — merging on an agent review's recommended verdict
without independently re-deriving it is a **supported mode, not an edge
case**. The obligation this creates falls on the *reviewing* agent: lead
with the verdict line, and make blocking findings unmissable at the top,
because the verdict line may be the only part read before the merge
button. T9's reviews already did this well — every one opened with
"**Recommended verdict:**" — which is very likely why fast merges
consistently landed on correct decisions.

**Recorded disagreement — PE vs. PM on whether to add a soak time.**

- **PE:** pair the documented mode with a minimum interval (or an explicit
  "read the verdict line" acknowledgement) between a REQUEST_CHANGES
  review and a merge. Three of six merges in this sample resolved a
  blocking finding in under six minutes; that is fast enough that a
  *wrong* fix would also have gone in unexamined.
- **PM:** a mandated delay solves a problem that did not occur. Nothing
  bad merged in T9, the throughput was the best in the project's history,
  and a throttle would cost real velocity against zero measured benefit.
  Document the mode; do not tax it.
- **Unresolved.** Recorded rather than smoothed. Both agree on documenting
  the mode; only the throttle is contested.

## 7. The largest merge conflict in five sprints was predicted, priced, and correctly handled — and the decision that caused it was never written down

**PdE.** The sprint plan's dependency-order section predicted this
precisely, in advance: "T9.6 and T9.7 both touch `web/src/router/index.ts`
and will conflict there if run in parallel — T8.8↔T8.9 hit exactly this...
Either sequence them, or have whichever lands second resolve on its own
source branch and re-verify (`npm run build` + `npm run test`) before
merge. Never resolve on the shared branch." The conflict duly arrived and
was the largest of the five T5–T9 sprints, across four files both tickets
independently created. It was resolved on the source branch, by hand,
without picking a side — `CompetitionSummary` gained T9.7's nullable
`spotsLeft`; the one genuine signature collision (`formatSessionRange`)
kept T9.6's name for its two call sites with T9.7's variant renamed
`formatSessionRangeFromSession`; aliases preserved both PRs' naming — and
re-verified (383/383 web tests) before merge. **PE:** a predicted, priced,
correctly-handled cost is not a mistake, and the plan deserves credit for
naming the prior instance rather than rediscovering it.

**QA (the finding).** The plan offered two options. The team took option B
(parallel, resolve on the source branch) and **no record exists of why**.
The decision that produced the sprint's largest conflict left no trace —
not in a PR body, not in HANDOFF, not in a ticket comment. Structurally
identical to finding 1: a real in-sprint decision, made in the moment,
recoverable only by inference afterward. Small, and worth naming precisely
because it is small — this is the low-stakes version of the pattern that
CLAUDE.md rule 9's history shows becoming expensive when it scales.

**Change for T10, adopted:** where a sprint plan offers an explicit either/or,
the choice gets one line in the PR body of the first ticket affected. One
sentence, not a document.

## 8. Every ticket converged in one loop — and the team does not agree on what that means

**PdE.** Ten tickets, 49 points, all merged; no ticket used a second
implement→review loop. The three REQUEST_CHANGES verdicts were resolved
inside the same PR rather than by reopening a loop. The 5-loop cap was
never approached — four sprints in, no ticket has ever needed more than 2
(T5.4's was the only loop 2 in project history). A cap that has never
bound is a cap that has never been validated. Not recommending a change (a
non-binding cap costs nothing), but flagging that "5" should stop being
cited as a calibrated number; it is an untested one.

**Recorded disagreement — PE/QA vs. PdE on whether one-loop convergence is
a quality signal at all.**

- **PdE:** ten-for-ten in a single loop, on a 0→1 bounded context plus two
  UI screens, is the strongest delivery result this project has produced.
- **PE:** loop count measures how fast the implement-review cycle closed,
  not how much was left uncaught. Three of these one-loop tickets carried
  blocking defects, and the sprint's *most severe* defect (finding 2) was
  found outside the loop entirely, by a reviewer exceeding their ticket's
  scope. A loop-count of 1 would look identical in a sprint where nobody
  looked hard.
- **QA** agrees and points at finding 3 as the proof: T9.5's blocking bug
  was found only because the reviewer ran the real server against real
  Postgres instead of reading the fake-backed tests. Same ticket, same
  loop count, entirely different outcome depending on reviewer effort —
  which is the definition of a metric that is not measuring the thing.
- **Unresolved**, and deliberately so. The team declines to adopt loop
  count as a quality indicator in either direction until there is a sprint
  where it varies for a legible reason.

---

## No finding on

**PM had less independent material this sprint than the role usually
does**, and the reason is structural rather than a gap: T9's scope was set
by T8's own re-scope decision, and the two live product questions
(social-account-linking's OAuth half, auto-matching) were settled by
ADR-0009 and ADR-0010 *inside* the sprint rather than left open for PM to
push against. PM's substantive contributions this retro are findings 5
(cross-context timing) and 6 (the throttle objection). Noted rather than
padded, per the same rule this retro follows for disagreements.

**The Designer role's T9 output was substantial but sits in the PR
reviews, not here** — the deceptive-pattern argument against a dead
"Connect" button (ADR-0009 at the UI layer), the live-region and
focus-management checks on both UI tickets, and the WCAG spot-checks are
recorded as GitHub PR reviews on #90 and #91 per the naming convention,
and produced no separate retro finding because none of them found a
process gap.
