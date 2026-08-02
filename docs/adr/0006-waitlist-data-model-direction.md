# ADR-0006: Waitlists are a first-class entity with an ordered queue and a response timeout

## Status
Accepted in direction; full schema/ticket detail deferred to T5/T6 backlog refinement

## Context
Spec §3.4 lists "waitlists" as a feature in the same breath as
cancellation windows and no-show handling, but `docs/requirements/
research-functional.md` §1.3 found the data model (§8) has no
`waitlist_entries` table or anything resembling one — the feature is
named but has no design. Real comparable platforms treat this as core
plumbing: CourtReserve auto-promotes a waitlisted player the instant a
slot frees up, combined with cancellation-window/no-show enforcement.
Left undesigned, "waitlists" would either not ship, or ship as an
afterthought bolted onto `CancelBooking` without the queue-ordering and
promotion-timeout semantics real users expect from the word "waitlist."

## Decision
Model waitlists as their own entity (`waitlist_entries` or equivalent),
keyed to the thing being waited for (a court/slot, or a game — both need
this per §3.4 and §3.6), with:
1. An **ordered queue** per waited-for resource (FIFO as the v1 default;
   skill-matched ordering for games is a documented future refinement,
   not a v1 requirement).
2. An **auto-promotion trigger** on cancellation/no-show freeing the slot
   — the same event `CancelBooking` (T3) already produces.
3. A **response timeout** on a promoted entry: a promoted player who
   doesn't confirm within a bounded window (exact duration is a PM/PO
   call during T5/T6 backlog refinement, not fixed here) is skipped and
   the next entry is promoted instead, so one unresponsive player can't
   block a slot indefinitely — this is the concrete gap CourtReserve's
   real-time-backfill design implies exists in practice.

This ADR fixes the *shape* of the design (entity + queue + promotion +
timeout); the exact ticket(s), table columns, and timeout duration are
left to normal backlog refinement (Ceremony 1) once T5/T6 scope reaches
this feature, per `docs/process/sprint-process.md`.

## Consequences
**Pros:** the waitlist claim in §3.4 becomes actually buildable rather
than aspirational; the promotion-timeout requirement is decided now,
before a v1 implementation ships without one and has to retrofit it after
a real slot gets stuck on an unresponsive player.
**Cons:** adds a genuinely new entity + background/event-driven promotion
logic that wasn't previously scoped as its own piece of work — likely its
own ticket(s) rather than folded silently into `CancelBooking`.
**Alternative considered and rejected:** treat "waitlist" as just
reopening the slot for direct re-request (Playtomic's Open Match
approach, per the same research) rather than a queued auto-fill.
Rejected because the spec's existing wording ("waitlists," used alongside
cancellation windows and no-show handling as club-facing booking
mechanics, §3.4) and the recurring-hire/club context (§3.5) both imply an
ordered, fairness-preserving queue is the expected behavior, not a free-
for-all re-request — worth confirming with PM/PO at T5/T6 refinement, but
this is the default assumed here.
