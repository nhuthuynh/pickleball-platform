# ADR-0008: Split/shared courts are a known modeling gap, deferred but logged

## Status
Accepted as a deferred, tracked limitation (not implemented)

## Context
`docs/requirements/research-functional.md` §1.5/§4.4 found that a common
real facility pattern — one physical court bookable either as a whole
(e.g. a tennis court) or as independent halves (e.g. two pickleball
courts painted over it), as documented by AllBooked/SmartHealthClubs —
isn't modeled anywhere in this project. `courts` (§3.2/§8) are flat,
independent units with no parent/child or overlap relationship. The
`EXCLUDE (court_id WITH =, during WITH &&)` constraint (§6, this
project's authoritative no-double-booking guard) is keyed purely on
`court_id` equality, so it cannot detect a conflict between a
whole-court booking and a half-court booking that physically occupies
part of the same space but has a different `court_id`. This is the same
*category* of invariant hole as design-review finding F1 (Booking and
Game not originally sharing an invariant), which was found and fixed at
real cost (see `docs/reviews/00-bootstrap.md` and ADR-0001) — cheaper to
name and design for now than to retrofit after a pilot facility with
split courts onboards and silently double-books.

## Decision
**Defer** building split/shared-court support — no pilot facility
requiring it is currently lined up (spec §12 still lists this as an open
question) and the current flat-court model is sufficient for T0–T4 and
the near-term T5/T6 backlog. **Do not defer acknowledging it.** This ADR
is the record: before any facility with physically split/shared courts
is onboarded, the schema needs a resource-hierarchy or overlap-group
concept (e.g. a `courts.parent_court_id` or a `court_overlap_groups`
join table) that the `EXCLUDE` constraint's conflict check can consult —
matching the general interval-scheduling-literature pattern of a resource
hierarchy/graph rather than a flat resource ID feeding an exclusion
check.

## Consequences
**Pros:** avoids speculative schema complexity for a scenario with no
committed pilot facility yet (YAGNI, correctly applied — this is not the
same as ignoring the gap, per CLAUDE.md's guidance against building for
hypothetical requirements); the gap is documented and searchable
(`docs/LESSONS.md`-adjacent, this ADR, and HANDOFF.md cross-cutting notes)
so it can't be silently forgotten the way F1 briefly was.
**Cons:** if a pilot facility with split courts appears before this is
designed, onboarding it without the fix would silently reintroduce a
double-booking hole identical in shape to F1 — the mitigation is process
(check this ADR before onboarding any facility, add it explicitly to
facility-onboarding acceptance criteria once that flow is built), not
code, until the feature is actually needed.
**Alternative considered and rejected:** build the resource-hierarchy
model now, speculatively. Rejected per CLAUDE.md's explicit rule against
designing for hypothetical future requirements — there is no committed
facility needing this today, and the cost of adding it later (one new
join concept feeding the exclusion check) is bounded and well understood,
unlike, say, a storage-format change.
