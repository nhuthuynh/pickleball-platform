# ADR-0015: What owns a Booking made through the public quote-and-book flow — escalated to the Product Owner, not decided here

- **Status:** **Accepted — D1 answered on 2026-09-04: option (a),
  "Authenticate the flow".** Implemented in T55.1; #144 closed. The
  question, the options and the reasoning below are preserved unedited as
  the record of what was decided and against what alternatives — read
  `## Resolution` (immediately after `## Status`) first, then treat the
  rest of this ADR as history rather than as an open question.
- **Date:** 2026-08-15
- **Ticket:** T14.3 (`docs/process/t14-sprint-plan.md`, §A6 and the T14.3
  ticket text)
- **Escalates:** #144 (*"CancelBooking (and CreateBooking) have no
  authorization check at all — anyone holding a booking id can cancel it"*).
  **#144 stays open**; this ADR does not close it and no ticket in T14
  implements it.
- **Relates to:** ADR-0012 (the other live escalation — see the explicit
  distinction below), ADR-0013 (verified caller identity as a platform
  capability), ADR-0014 (actor identifier space — the seam any owner column
  added here would have to route through), T7.6
  (`docs/process/t7-sprint-plan.md`, the shipped public quote-and-book flow)
- **Consumed by:** nothing in T14, by design. The first ticket that
  implements #144 consumes it — which cannot be scheduled until D1 is
  answered.

## Resolution — D1, answered 2026-09-04

**The Product Owner chose option (a): authenticate the flow.**

Concretely, and as implemented in T55.1:

- `domain.Booking` gains `OwnerUserID`, a **User.ID** uuid (never an IdP
  subject — ADR-0014), required by `domain.NewBooking`.
- `bookings.owner_user_id` is `uuid NOT NULL REFERENCES identity_users (id)`
  (migration `0027_booking_owner_user_id.sql`), following
  `recurring_hire_templates.requested_by_user_id`'s shape exactly, as the
  "A precedent already in this context" section below anticipated.
- `domain.Booking.EnsureOwner` is the rule; `ErrNotBookingOwner` maps to
  `PermissionDenied`, `ErrEmptyOwnerUserID` to `InvalidArgument`.
- `CreateBooking` and `CancelBooking` move from `PublicMethods()` to
  `AuthenticatedMethods()`. The owner is resolved from the verified token
  through `Handler.actor`, and is **not** a request field — `Booking` gained
  a read-only `owner_user_id` in the proto, but `CreateBookingRequest`
  deliberately did not.
- Every other Booking source supplies the owner from a fact it already
  holds: Social Play from `Game.HostID`, Competitions from
  `Competition.HostID`, `ApproveRecurringHire` from the template's
  `RequestedByUserID`. The two `port.CourtReservation` interfaces gained an
  `ownerUserID` parameter on both `ReserveCourt` and `ReleaseCourt` — the
  rollback path needs it, since a compensating cancel must now be performed
  by the booking's owner.

**The accepted cost, stated as plainly as the option table states it.**
Option (a)'s cost is that it *"breaks the shipped T7.6 public booking flow —
a player must register before booking"*, and that this is *"a
conversion/product call, not an engineering one"*. The Product Owner made
that call with the cost in front of them. `GetQuote` and
`ListCourtBookings` deliberately stay public, so the browse half of T7.6
(see a price, see availability) is unaffected and only the final confirm
step now requires an account. The Vue client at
`web/src/components/booking/CourtBookingFlow.vue` needs a sign-in step
before its confirm call; that is follow-up client work, not backend work,
and is recorded in `HANDOFF.md`'s Cross-cutting section rather than being
silently absorbed here.

**What this ADR's own trigger condition demanded, and whether it was met.**
The trigger below says *"the sprint immediately following the user's answer
to D1 must implement that answer ... and close #144"*. The answer arrived
during T55 and was implemented in that same sprint, so the trigger is
satisfied rather than deferred.

**Why the deferral history below still matters.** #144 was deferred twice
(T13 in prose, T14 as this ADR) and then sat unanswered for 41 consecutive
sprints — T14 through T54 — during which the backlog produced zero tickets
because this and D2 blocked everything in it. That is the finding this ADR
warned about in its own words: *"a third deferral without an answer is a
finding, not a decision."* The finding is recorded in
`docs/process/t54-retro.md`; the mechanism that let an escalation sit
unanswered for 41 sprints is a process defect, not a product one, and is
where the retro directs attention.

## Status

**Superseded by `## Resolution` above. Preserved verbatim as the record of
the question as it stood before D1 was answered.**

**Escalated — awaiting product decision. This ADR decides nothing.**

Every prior ADR in this repository that carried an escalation
(ADR-0009, ADR-0010, ADR-0012) was still `Accepted`, because each one
*also* decided something buildable in the same sprint and named the
blocked remainder alongside it. ADR-0012 is the closest precedent and the
clearest illustration: it is `Accepted` because it decides to **build**
Identity/Users and `Match`, while recording that `PlayerRating`, the
matching algorithm, and gender-mix matching remain blocked on Q1/Q2.

**This ADR has no buildable half.** The entire content of #144 —
what field to add, what to compare it against, whether the flow that
creates these rows should require an account at all — is downstream of
the one unanswered question. There is no equivalent of "build the
storage, defer the formula" here: the storage *shape* is what the
question is about. Marking this `Accepted` would therefore assert a
decision that was not made, so the status is `Escalated` instead, and
the "Decision" section that a normal ADR carries is replaced by
"The question" and "The options."

### How this differs from ADR-0012's Q1/Q2 — read this before filing D1 in the same drawer

ADR-0012's Q1 (Player Level formula weighting) and Q2 (whether gender-mix
matching is in scope at all) carry a **legal/ethical dimension** — Q2
turns on whether this platform should collect and algorithmically act on a
protected attribute, in the jurisdictions it launches in. They are blocked
**indefinitely**, and ADR-0012's rule that no PR may add a `PlayerRating`
or `Gender` field stands until they are answered.

**D1 is not that.** It is an ordinary product decision — a
conversion-versus-security trade on a booking flow — with **no legal
dimension**, and it *should* be answered, soon, in the ordinary course of
product work. A future reader who files D1 next to Q1/Q2 in the
permanently-blocked drawer has mis-filed it: Q1/Q2 are waiting on a
judgement this project may never be entitled to make on the user's behalf,
whereas D1 is waiting on nothing but the Product Owner's attention.

This distinction is drawn here because both escalations otherwise read
identically — same "Status: not decided," same "trigger is the user's
answer" — and the two are one sprint apart in the ADR sequence.

## Context — the gap, re-verified against the code for this ADR

The T14 sprint plan (§A6) states these facts. Per the T14.3 ticket's
instruction 2, they were re-read from the source rather than copied, and
all three hold as of this commit:

**1. `bookings` has no owner column of any kind.**
`db/migrations/0001_init.sql:22-42` defines the table as exactly: `id`,
`court_id`, `source`, `status`, `starts_at`, `ends_at`, `during`
(generated), `reference_id`, `created_at` — plus a `CHECK` and the
no-double-booking `EXCLUDE` constraint. Nothing identifies a person. No
later migration adds one (`db/migrations/` contains no `ALTER TABLE
bookings` adding an owner). The domain agrees:
`domain.Booking` (`internal/booking/domain/booking.go:37-44`) is
`ID, CourtID, Source, Status, Range, ReferenceID` — six fields, no actor.

**2. `CancelBooking` takes no actor parameter.**
`func (s *Service) CancelBooking(ctx context.Context, bookingID string)
(domain.Booking, error)` —
`internal/booking/app/service.go:287`. It shape-checks the id, fetches,
calls `b.Cancel()`, and persists. There is no actor argument, so there is
nothing to compare an owner against even if one existed.
`CreateBooking` (`service.go:234`) likewise takes a `CreateBookingInput`
carrying no actor.

**3. Both RPCs are in `PublicMethods()`.**
`internal/booking/adapter/grpcapi/authenticated.go:68`, and the doc
comment immediately above it already discloses precisely this, in these
words: public *"not because that is right, but because making them
otherwise is out of this ticket's reach"*, and *"anyone holding a booking
id can cancel that booking today, and that is as true after this ticket as
before it."*

**Consequence, stated plainly:** anyone who knows or guesses a booking id
can cancel that booking. This is the sharpest remaining object-level
authorization hole in the codebase.

### One fact the sprint plan did not have, which changes an option's cost

§A6's cost for option (a) is *"Breaks a **shipped** public UI flow."*
Checked against the client this ADR asked which shipped flow, and the
answer is narrower than the plan states:

- **`CreateBooking` does have a shipped public consumer.** T7.6
  (`docs/process/t7-sprint-plan.md:671`) shipped the Vue
  quote → review/confirm → book flow; it lives at
  `web/src/components/booking/CourtBookingFlow.vue` with
  `web/src/composables/useCourtBooking.ts` and
  `web/src/api/bookingClient.ts`. Authenticating creation does break a
  real, shipped, user-facing path.
- **`CancelBooking` has no shipped consumer at all.** No file under
  `web/src` references `CancelBooking` in any spelling, and there is no
  mobile client source in this repository (`buf.gen.mobile.yaml` exists;
  no `.swift`/`.kt` sources do). T7.6's ticket text scopes the flow to
  quote, conflict-handling, confirm, and success — it specifies no
  cancellation UI, and none was built.

So the "breaks a shipped flow" cost attaches to **creation**, not to
**cancellation** — even though cancellation is the sharper half of the
hole. This does not answer D1, but it means the option set is not a
single four-way choice; see option (d), which the sprint plan did not
name and which this fact makes available.

### A precedent already in this context

Booking already models an owner on a different aggregate:
`RecurringHireTemplate.RequestedByUserID`
(`internal/booking/domain/recurring_hire_template.go:103`, rejected empty
via `ErrEmptyRequestedByUserID` at `:130`), stored as
`requested_by_user_id uuid NOT NULL REFERENCES identity_users (id)`
(`db/migrations/0018_booking_recurring_hire_templates.sql:32`). #144's own
"suggested shape" points at it.

This matters for sizing, not for the decision: **the mechanics of adding
an owner are a solved, copyable pattern in this very context.** What is
unsolved is what value to put in the column when nobody is logged in —
which is D1, and is not an engineering question.

## The question

Stated as one sentence a non-engineer can answer, per the T14.3 ticket's
instruction 3:

> **DECISION D1 (for the user / Product Owner):** When somebody books a
> court through the public flow *without an account* — which is how the
> flow works today — who should be allowed to cancel that booking later,
> and should booking without an account remain possible at all?

## The options

None of these is chosen here. Costs are stated as neutrally as they can
be; where a cost is a product judgement rather than an engineering fact,
it is marked as such.

| | Option | What it means | Cost |
|---|---|---|---|
| **(a)** | **Authenticate the flow** | `CreateBooking` requires a verified principal; the booking is owned by that user; `CancelBooking` requires the principal to match the owner. | Breaks the **shipped** T7.6 public booking flow — a player must register before booking. A conversion/product call, not an engineering one. Fully closes #144. |
| **(b)** | **Guest bookings with a capability token** | The flow stays public. Creating a booking returns a one-time management token the client keeps; cancelling requires that token *or* an owning principal. | A new concept in the domain **and** the public API (token issuance, storage, presentation, expiry). A lost token means a booking nobody can cancel except an operator. Closes #144, at the price of a concept the platform does not otherwise have. |
| **(c)** | **Owner-when-known, open-when-not** | Record the owner when a principal exists; leave it null otherwise; enforce the match only when non-null. | Cheapest, and honest about what it is — but it **leaves the hole open for exactly the bookings that have it today** (the anonymous ones). Does **not** close #144. |
| **(d)** | **Authenticate cancellation only; leave creation public** | `CreateBooking` stays public and anonymous. `CancelBooking` leaves `PublicMethods()` and becomes an authenticated operator/facility action; a guest who needs to cancel contacts the facility. | Closes the sharp half of #144 — anonymous id-guessing cancellation — **without touching the shipped T7.6 flow** (see the verified fact above: nothing calls `CancelBooking` today). But guests then cannot self-serve a cancellation they could nominally perform today, which is an operational load on facilities and a product call; and bookings still have no owner, so #144's underlying modelling gap stays open. |

Notes on the option set, so a future reader can tell what is settled
from what is not:

- **(d) is new in this ADR**, added under the T14.3 ticket's instruction
  3 (*"Add or correct options if the code says something this ceremony got
  wrong"*). It is available only because `CancelBooking` turned out to have
  no shipped consumer — a fact §A6 did not have. It is listed because it is
  real, not because it is preferred; **this ADR expresses no preference.**
- **(a) and (d) are not mutually exclusive over time** — (d) is reachable
  immediately and does not foreclose (a), (b), or (c) later. Recording
  that is a sequencing fact, not a recommendation to take it.
- **(c) is the only option that does not close #144**, by its own
  description. If (c) is chosen, #144 should be closed as *won't fix* with
  that reasoning recorded, rather than left open against a decision that
  deliberately does not close it.

## Trigger condition

**The sprint immediately following the user's answer to D1 must implement
that answer** — the owner field (or token, or de-listing) it implies, the
migration, the domain change, and the authorization check — and close
#144.

Mirroring ADR-0012's trigger deliberately, and in its own words: this
trigger is **tied to an event outside any ceremony's own judgment — the
user's answer arriving — not to a sprint boundary.** A future Ceremony 1
that has the answer in hand and still does not implement it is subject to
the same rule ADR-0010 established: defer again in prose and a reviewer
may block on that basis alone.

**If D1 is unanswered when T15 plans, that is a finding, not a decision** —
see the next section for why that phrasing is load-bearing.

## This is #144's second deferral, and in what form

Recorded so T15's retro can score it without reconstructing the history:

| Sprint | Form of the deferral |
|---|---|
| **T13** | A paragraph in a sprint plan — ranked, named, not scheduled. |
| **T14** | **This ADR** — the question stated for a non-engineer, four options with costs, the facts re-verified against code, and a trigger tied to the user's answer. |

This is the same progression ADR-0010 made after three prose deferrals:
each deferral must change form and add information, or it is just
repetition. **A third deferral without an answer is a finding, not a
decision** (T14 sprint plan §A6).

## Concretely, until D1 is answered, no PR may

Mirroring ADR-0012's "concretely, no PR may" list, so the escalation has
teeth rather than being advisory:

1. Add an owner/actor column to `bookings`, or an owner/actor field to
   `domain.Booking`, in **any** shape — including a nullable one. The
   column's nullability *is* the choice between (a), (c), and (d); adding
   it presupposes an answer.
2. Add an actor parameter to `app.Service.CreateBooking` or
   `app.Service.CancelBooking`.
3. Remove `CreateBooking` or `CancelBooking` from
   `grpcapi.PublicMethods()`. (This is option (d), and it is an option,
   not a default.)
4. Introduce a booking-management token in the proto, domain, or API.
5. Close #144 on the grounds that it is documented here. **It is
   escalated, not resolved.** This ADR is a reference #144 should link to,
   not a closure reason.

This is a restriction on *guessing the answer*, not on the codebase
generally: unrelated Booking work continues normally.

## What would change this

**Would change it:** the user/PO answering D1 (the trigger above); or a
new, ticketed requirement that forces one option's hand — for example a
payments or refund flow that structurally cannot exist without a booking
owner, which would convert D1 from "product preference" to "blocked
dependency" and should be escalated with that framing rather than
silently resolved by the ticket that hit it.

**Would not change it:** elapsed time; a reviewer's or agent's opinion
about which option is best; the observation that (c) or (d) is "cheap";
or the security severity of the hole — the hole's sharpness is an argument
for answering D1 **urgently**, not for answering it **unilaterally**, and
those are different things. The whole reason this ADR exists is that the
cheapest-looking path (c) is precisely the one that does not close the
gap, and the fastest-looking path (d) still makes a product call about
what guests can do.

## Consequences

**Pros.** #144's blocker is now a written question with named options and
verified facts, addressed to the person who can actually answer it,
rather than a recurring paragraph rediscovered mid-ticket each sprint —
which is exactly what T13's retro recommendation 5 asked for. The
re-verification found one thing the plan had wrong (the shipped-flow cost
attaches to creation, not cancellation) and surfaced a fourth option as a
result, so the escalation the PO receives is better than the one Ceremony
1 drafted. The `RequestedByUserID` precedent means whichever option is
chosen, the implementation is a known pattern in this context.

**Cons.** **The sharpest object-level authorization hole in the codebase
remains open for at least one more sprint** — anyone with a booking id can
still cancel it, in production-shaped code, and this ADR does not change
that by a line. That is a real, disclosed security cost, accepted here
only because the alternative is an engineer choosing a public-flow product
change nobody asked for. The disclosure in `PublicMethods()`' doc comment
remains accurate and should stay until D1 is answered.

**Alternative considered and rejected: pick option (c) as a "safe
default" and revise later.** Rejected — (c) is the one option that by its
own description does not close the gap, so shipping it would let #144 be
closed on paper while every anonymous booking stays exactly as cancellable
as it is today. That is worse than the current state, which at least
carries an honest disclosure in code.

**Alternative considered and rejected: pick option (d) here, on the
grounds that it breaks nothing shipped.** Rejected — "breaks no shipped
client code" is not the same as "costs nothing." Removing self-service
cancellation decides that guests phone the facility instead, which is a
product and operations decision with a real support cost, and
`sprint-process.md` puts user-facing flow changes on PM/PO, not on PE.
The verified fact that it breaks no code is recorded above because it is
true and relevant to the PO's choice; it is not a licence for this ticket
to make that choice.
