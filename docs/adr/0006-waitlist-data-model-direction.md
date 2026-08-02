# ADR-0006: Waitlists are a first-class entity with an ordered queue and a response timeout

## Status
**Game waitlists shipped in T6.6.** `internal/socialplay/domain.WaitlistEntry`
(`waiting | promoted | expired | cancelled`), `domain.JoinWaitlist`, the
app-layer auto-promotion orchestration (`app.Service.CancelRegistration` /
`ExpireWaitlistPromotion`), and the `waitlist_entries` table +
`promote_next_waiting` Postgres function (`db/migrations/
0007_socialplay_waitlist.sql`, `0008_socialplay_waitlist_promotion.sql`)
implement all three elements this ADR fixed as the design's shape: an
ordered (FIFO) queue per Game, an auto-promotion trigger firing on
`Registration.Cancel`, and a response timeout
(`domain.PromotionResponseWindow`, defaulted to 30 minutes, product-tunable)
after which an unresponsive promoted entry expires and cascades the
promotion to the next waiting entry. The promotion-ordering race (two
concurrent cancellations/expiries racing to promote off the same freed
slot) is closed at the DB level by a `FOR UPDATE`-locking Postgres function
mirroring ADR-0001/T5.4's dual-invariant pattern — see T6.6's PR description
for the full race analysis and concurrency-test evidence.

**Standalone court/slot waitlists remain deferred, with no ticket yet.**
This ADR originally described waitlists as "keyed to the thing being waited
for (a court/slot, or a game — both need this per §3.4 and §3.6)." T6
sprint planning (`docs/process/t6-sprint-plan.md`) scoped T6.6 to the Game
case only: there is a concrete, already-built hook for it
(`domain.ErrGameFull`, per T5.2's own design), whereas a standalone
court/slot waitlist has no existing "request a specific court/slot
directly" feature to hook onto (`CreateBooking` either succeeds or returns
`ErrCourtDoubleBooked` today — there's no "ask to be notified" concept at
all). Revisit if a direct court-request feature is ever built; until then
this is a real, logged gap, not a silently dropped one.

--- History (T5) ---
T5 sprint planning (`docs/process/t5-sprint-plan.md`) resolved a genuine
PM/PE disagreement on timing: T5 shipped `Game`/`Registration` with
capacity enforcement that rejects overflow via a stable `domain.ErrGameFull`
error (a deliberate hook), but the actual `waitlist_entries` entity, queue
ordering, promotion trigger, and response timeout described below were
deferred to T6 at that time, not built in T5.

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
