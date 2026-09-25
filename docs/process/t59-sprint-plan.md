# T59 Sprint Plan — Ceremony 1 (Backlog refinement)

Held per `docs/process/sprint-process.md`, six-role team (briefs:
`docs/agent-operating-handbook.md` Part B), against `HANDOFF.md`,
`docs/process/t56-retro.md`, `docs/process/t57-retro.md`,
`docs/process/t58-retro.md`, and the live issue/PR/commit history.

**This is the first Ceremony 1 since T55**, because T56, T57 and T58 each
held none — deliberately, and correctly under the escalation rule: all three
had answered work in front of them and built rather than planned. Three
sprints of bookkeeping therefore land here at once.

**The headline result: the issue sweep's new premise check fired on its
first run.** #149's premise has substantially drifted — four of the five
fields it names are no longer read from the wire. See §3.

---

## 1. First act — the merged-fix issue sweep (authoritative run)

Per `sprint-process.md`, this runs before anything is ranked.

**Live open issues: 4** — #296, #149, #145, #134. Fetched live, not carried.

**No open issue has a fix that merged in T56–T58.** The three closed in that
run — #126, #297, #299 — were each closed manually at merge time, since
`Closes #N` structurally cannot auto-fire against a non-default base branch.

**Arithmetic reconciliation.** T55's retro recorded 5 open. #126 closed
(T56/T57). #297 and #299 were each opened *and* closed within the run. 5 − 1
= 4. ✅

## 2. Merge order, verified against `merged_at`

This project's standing convention is that PR numbers and merge order
routinely disagree, so the order is verified rather than inferred:

| PR | Sprint | `merged_at` |
|---|---|---|
| #298 | T56 | `2026-09-21T09:03:58Z` |
| #300 | T57 | `2026-09-21T09:06:59Z` |
| #301 | T58 | `2026-09-22T07:12:17Z` |
| #302 | T56–T58 bookkeeping | `2026-09-22T11:35:40Z` |
| #303 | T56–T58 retros | `2026-09-22T11:49:42Z` |

Numbering and merge order **agree** this time. Recorded explicitly because
the convention exists for the times they do not, and "they agreed" is
itself a checked fact rather than an assumption.

## 3. Issue sweep with premise verification — the new check, and what it found

**Adopted here from `docs/process/t56-retro.md` recommendation 1**, and
written into `sprint-process.md` by this ceremony's PR. The rule: the sweep
re-verifies each issue's **premise** against the tree, not only its
**blocker** — for every issue, not only old ones, because #126 was false
nine days after filing and an age threshold would not have caught it.

| Issue | Blocker | Premise | Verified by |
|---|---|---|---|
| #296 | unchanged | **holds exactly** | no `uuidShape` guard on `OwnerUserID` in `CreateBooking`; `repository.go:81` still `mustUUID(b.OwnerUserID)` |
| #149 | unchanged | **substantially drifted — see below** | `grep` for each named field's use in `app/service.go` and `grpcapi/handler.go` |
| #145 | unchanged | holds | `0019_identity_subject.sql` present; the pre-existing-row gap is structural |
| #134 | unchanged | holds | all three screens present; routes still in `ROUTES_UNDER_TEST` |

### #149's premise has drifted, and the issue now describes one field, not five

#149 says Payments "still accepts caller-supplied ownership facts" and names
five:

| Field | Still read from the wire? | Evidence |
|---|---|---|
| `game_host_id` | **no** | `handler.go:269` — *"deliberately NOT read here anymore"* (T16.2) |
| `assigned_game_admin_user_ids` | **no** | same, T16.2 |
| `entrant_player_id` | **no** | `handler.go:172` — *"deliberately NOT read here"* (T17.1) |
| `assigned_competition_admin_user_ids` | **no** | same, T17.1 |
| `booking_host_id` | **yes** | `handler.go:160` and `:284` read it; `service.go:1287` compares against it |

Four of five were closed by T16.2 and T17.1, which built the read-side
resolver ports #149 itself proposed — `RegistrationLookup`, `GameLookup`,
`GameAdminReader`, `EntryLookup`, `CompetitionAdminReader`. The issue was
never updated to say so.

**What remains is narrower and sharper than the issue states:**
`booking_host_id` is the last caller-asserted ownership fact, and it
survives for a structural reason the other four did not have — **Payments
has no port into Booking at all.** `internal/payments/port/` holds lookups
into Social Play, Competitions and Identity; there is no `booking_lookup.go`
and no `internal/payments/adapter/booking/`. Closing it means building that
seam, which is exactly the shape T16.2 and T17.1 already built twice.

T56–T58 narrowed it further in a way worth naming: those sprints added
read-side ports for **price** (`RegistrationAmountLookup`,
`EntryAmountLookup`) — the same family of fix (Payments resolving a fact
rather than being told it) applied to a different fact. The pattern is now
established three times over.

**Action:** #149's body is corrected as part of this ceremony, per
`docs/process/t58-retro.md` recommendation 3 ("when an issue's analysis is
corrected, correct the issue"). It is not closed — one real hole remains.

## 4. Unanswered escalations

Per the rule adopted at T55, raised **before** the sprint is planned.

**None.** D1 (ADR-0015) and D2 (ADR-0016) are both **Accepted**. Three
further product questions were put and answered during T56–T58 (#126's
pricing, #126's guest-lock, #299's validation rule), all within the same
exchange as being asked. There is no question waiting on the user.

`ADR-0012 Q1/Q2` remains in the indefinitely-blocked column — a
legal/ethical question that may never be this project's to answer — and is
deliberately not escalated, per ADR-0015's own warning against bundling it
with actionable decisions.

## 5. 0-ticket cap status

**Not engaged.** T56, T57 and T58 each took tickets, so the consecutive
0-ticket count stands at **0**. The cap (two consecutive, then stop) is
nowhere near firing. Recorded because the counter that produced the T30–T53
run is the one this section exists to watch.

## 6. Cross-cutting re-scan

One item is newly actionable and one is not:

- **`make ci-integration` has never been run** — now across three
  consecutive sprints of payment-path changes. This is not a GitHub issue
  and never has been; it is a standing environmental gap recorded in
  `HANDOFF.md` and in every PR body. **It cannot be closed by this project**
  (no Docker daemon in any session so far), which places it with #134 and
  #145 in character if not in form. Named here so it is not invisible for a
  fourth sprint.
- **The D1 client follow-up** (a sign-in step before the Vue booking
  client's confirm call) remains open and unblocked — ordinary unbuilt work,
  carried since T55.

## 7. Tickets for T59

Ceremony 1's job is to split the sprint's work into tickets. Three are
defined; dispatch is Ceremony 2's.

---

### T59.1 — Guard `Booking.OwnerUserID`'s shape (closes #296)

**Story.** As an operator, I want a malformed owner id to produce a domain
error rather than a server panic, so that a future caller supplying one
cannot take the process down.

**Description.** `domain.NewBooking` rejects an *empty* owner
(`ErrEmptyOwnerUserID`), but a malformed non-empty one reaches
`repository.go:81`'s `mustUUID`, which panics by design. `CourtID` has a
`uuidShape` guard for exactly this reason (#97/T10.7); `OwnerUserID` does
not. Unreachable today — every supplier is structurally a uuid — so this is
a consistency and blast-radius fix, not an active defect.

**Instructions.**
1. Write a failing test first: `CreateBooking` with a well-formed `CourtID`
   and a malformed `OwnerUserID` must return a domain error, not panic.
2. Add a `uuidShape` guard alongside the existing `CourtID` one, returning a
   new `domain.ErrInvalidOwnerReference` (mirroring
   `ErrInvalidCourtReference`'s naming). Map it at the gRPC boundary —
   `InvalidArgument`, since a malformed **server-resolved** value is a
   programming error and not a permission answer.
3. Add the sentinel's row to booking's error-mapping table, which is
   source-checked for completeness and will fail without it.
4. **Decide and record** whether `CancelBookingsForReference`'s
   `actorUserID` (T55.2) wants the same treatment. It is *compared*, never
   written, so it cannot panic — the symmetry question is real but the
   answer may be "no, and here is why."

**NFRs.** No behaviour change for any currently-reachable input; the new
test must fail before the guard exists (verify by deletion).
**Points:** 2. **`role:principal-engineer`, `type:chore`.**

---

### T59.2 — Re-scope #149 to the one hole that remains

**Story.** As a future maintainer, I want #149 to describe the gap that
actually exists, so that whoever picks it up does not re-litigate four
fields that were closed two sprints apart.

**Description.** §3's finding. This is the first application of
`t58-retro.md` recommendation 3, and it is deliberately a *ticket* rather
than a silent edit, because correcting an issue's analysis is work with a
reviewable output.

**Instructions.**
1. Post a correction comment on #149 stating, per field, which are no longer
   caller-read and which PR/sprint closed them (T16.2, T17.1), with the
   handler line references as evidence.
2. State plainly what remains: `booking_host_id`, and that it survives
   because Payments has no Booking port — with the observation that T16.2,
   T17.1 and T56–T58 have now built that same seam three times, so the shape
   is known.
3. Do **not** close it. One real hole remains.
4. Re-title if the current title is now misleading (it names five facts).

**NFRs.** Evidence, not assertion: cite file and line for each claim.
**Points:** 1. **`role:principal-engineer`, `type:chore`.**

**Status: DONE during this ceremony.** Correcting an issue is the ceremony's
own bookkeeping rather than sprint execution, and the rule adopted as T59.3
requires the correction to land *on the issue*. The comment is posted and
the title retitled from five facts to one. A further finding surfaced while
writing it: this issue's stated dependency ordering — "the Game-Admin /
Competition-Admin durable store should probably be closed first" — is
**also stale**, since `GameAdminReader`/`CompetitionAdminReader` have
resolved admin sets against real data since T16.2/T17.1. And
`bookings.owner_user_id` has existed as a real FK since T55.1/migration
0027, so the fact a `BookingLookup` would resolve is now durably stored —
which was not true when the issue was filed.

---

### T59.3 — Adopt T56–T58's retro recommendations into `sprint-process.md`

**Story.** As a future ceremony, I want the three sprints' findings to be
rules I am held to, so that they do not depend on one session's memory.

**Description.** T54's recommendations were adopted at T55 and are the
reason T56–T58 ran the way they did. Same treatment.

**Instructions.** Adopt, each with its originating retro cited:
1. **The sweep re-verifies premises, not only blockers** (t56 rec 1) — for
   every issue, no age threshold, with §3 as the worked example of why.
2. **An issue quoting an earlier sprint's finding must re-verify it at
   filing time and date it** (t56 rec 2).
3. **A deliberate scope exclusion in shipped code gets a tracked issue**,
   extending the board-of-record rule from "a gap you found" to "a gap you
   chose to leave" (t57 rec 1).
4. **When an issue's analysis is corrected, correct the issue** (t58 rec 3).
5. **A third instance of "a safety net hid a defect" goes to
   `docs/LESSONS.md`, not a retro** (t58 rec 4) — with the two existing
   instances named so the counter is legible.

Recommendations *not* adopted, with reasons stated in the PR: t56 rec 3/4
and t57 rec 2/3 are ticket-level craft, already practised, and writing them
into the process document would add rules nobody consults for behaviour
that is already habitual.

**NFRs.** Each adopted rule carries its threshold and its action, per the
counter rule adopted at T55 — a rule without an action is a counter that
gets ignored.

**Status: DONE during this ceremony** (the adoption of process rules is
Ceremony 1's own output, as T55 did with T54's). Four sections added to
`sprint-process.md`: the premise check (with the no-age-threshold argument
and both worked examples), the correct-the-issue rule, the
safety-net-hid-a-defect counter at threshold two, and the scope-exclusion
extension to the board-of-record rule.
**Points:** 2. **`role:product-manager`, `type:chore`.**

---

## 8. Dependency-completeness check

Run against the code, both questions (does the producer exist; can the
consumer reach it):

| Ticket | Needs | Exists? | Reachable? |
|---|---|---|---|
| T59.1 | `uuidShape` in booking's app layer | yes — `service.go:25` | yes, same package, same function |
| T59.1 | an error-mapping table to extend | yes — booking's `error_mapping_test.go`, source-checked | yes |
| T59.2 | nothing in code; GitHub issue write | n/a | n/a |
| T59.3 | nothing in code | n/a | n/a |

**No inter-ticket dependency.** All three are independent and may be
dispatched in any order or in parallel.

## 9. What this ceremony deliberately did not do

- **It did not take #145 or #134.** Both remain blocked on things this
  environment cannot produce (a real IdP `sub` claim; real
  assistive-technology hardware). Neither premise has drifted.
- **It did not manufacture a payments ticket.** Amount validation is
  complete and checkable (`grep -n "s.payments.Create("` → two sites, both
  validated). The remaining unchecked payable types are argued, not
  inherited. Building further there would be work looking for a
  justification.
- **It did not open an issue for `make ci-integration`.** It is an
  environmental gap, not a code defect, and filing it would produce an issue
  no session can close — the failure mode `sprint-process.md`'s
  indefinitely-blocked split exists to keep visible rather than to
  accumulate.

---

# T59 Ceremony 2 — Sprint planning (kickoff note)

Full six-role team against §7's refined tickets.

## Sprint goal

> A malformed owner id produces a domain error rather than a server panic,
> closing the last of #97's unfinished business — and whatever that guard
> surfaces is reported rather than absorbed.

The second clause was added during Ceremony 2 at QA's insistence, and it
earned its place within the hour: see §12.

## Scope

**In:** T59.1 only. T59.2 and T59.3 completed during Ceremony 1 as its own
output (correcting an issue and adopting process rules are ceremony
bookkeeping, not sprint execution) and are marked DONE in §7 rather than
carried here as if pending.

**Not taken, with reasons:** #145 and #134 remain blocked on things this
environment cannot produce. #149, freshly re-scoped by Ceremony 1, is now
*ready* — one field, one seam, a shape built three times before — but it is
a new port plus adapter across a context boundary, which is not the same
sprint as a two-line guard. It is the obvious T60 candidate.

## Dispatch isolation

**One ticket, one implementer — no isolation mechanism required**, per
`sprint-process.md`'s own scope note ("a solo-implementer wave... does not
require this"). Stated explicitly rather than omitted, because that section
exists precisely because T29's plan omitted it and a near-collision
followed. A wave of one is the case where the rule does not apply; saying so
is how a reader knows it was considered.

Same-wave shared-interface verification: not applicable, same reason.

## Recorded disagreement — QA vs PE, resolved in QA's favour

**PE** held that T59.1 is a two-line guard against an unreachable input, and
that the ticket's instruction 4 (decide whether
`CancelBookingsForReference`'s `actorUserID` wants the same treatment) was
scope creep on a 2-point ticket — a question to note, not to answer.

**QA** held that "unreachable" is the claim #97 disproved about `CourtID`,
and that a guard whose justification is *"no supplier is malformed today"*
must state what happens when one is, or the next ticket re-opens the same
question from scratch.

**Resolved in QA's favour, and the resolution paid off immediately.**
Instruction 4 is answered as a *test* rather than as prose
(`TestCancelBookingsForReference_MalformedActorIsRefusedNotErrored`), and
the answer is **no guard there** — that value is only ever compared, never
written, so it cannot reach `mustUUID`; a malformed actor correctly fails
the ownership check. Adding a guard would convert a correct
`PermissionDenied` into an `InvalidArgument` and tell an unauthorized caller
their id was *shaped* wrong, on the endpoint #144 was filed about.

PE's cost was real and is recorded: the ticket came in at 4 points of work
against a 2-point estimate, and the overrun is entirely instruction 4 plus
§12.

## Execution outcome

Delivered as specified. `domain.ErrInvalidOwnerReference`, a guard in
`CreateBooking` placed **after** `CourtID`'s (so a doubly-malformed request
keeps answering the code it always has — pinned by a test), and the gRPC
mapping to `InvalidArgument` rather than the `NotFound` its `CourtID`
counterpart gets, because the owner is server-resolved and no request names
it.

The source-checked error-mapping table failed on the new sentinel before the
mapping was added, exactly as instruction 3 predicted. **Verified by
deletion:** removing the guard fails two tests.

## §12 — What the guard surfaced, and the scope call it forced

A guard for an "unreachable" condition immediately broke five Social Play
tests. Not a regression — **those tests were modelling a state the schema
forbids.**

`internal/socialplay/adapter/grpcapi`'s shared `fakeIdentityLookup` returns
the subject *unchanged*, so `ctxAs("host-1")` produced
`Game.HostID == "host-1"`. Social Play passes `Game.HostID` as a Booking's
owner, and `games.host_id` has been `uuid NOT NULL REFERENCES identity_users
(id)` since migration 0026. They passed only because that harness's booking
repository is in-memory and never reaches `mustUUID`.

**The scope call, measured rather than guessed.** Fixing the shared fake
properly — mirroring Payments' `resolvedUserID` from T28.1 — takes the
package from 6 failures to **30**: ~24 assertions across ~12 files compare a
resolved actor against a raw-subject fixture. That is Social Play's own
T28.1-equivalent fixture pass, not a 2-point guard ticket.

T59.1 therefore fixed it **locally** — a `resolvingIdentityLookup` used only
by `newBookingBackedHandler`, the one harness driving the real
`bookingapp.Service` — and **filed #305** for the rest.

That filing is itself the first application of the board-of-record extension
adopted hours earlier at T59.3 (*"a deliberate scope exclusion gets a
tracked issue"*), applied by its own author to their own exclusion. The rule
would have been easy to adopt and then quietly not follow on its first
occasion.

**A pattern worth watching, named in #305.** This is the third time in four
sprints that a fixture modelling something the real system cannot produce
has hidden or distorted a result (T56's wire-test gap, T58's fallback, this).
`docs/LESSONS.md`'s T9 entry already names fixture infidelity; what is new is
that it keeps recurring in fakes of the **identity resolution seam**, where
the subject-vs-`User.ID` distinction is exactly what is being faked away.
This does not meet T59.3's safety-net counter (a different pattern), and it
is deliberately not forced into it.
