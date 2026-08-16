# ADR-0017: Identifier conformance and FK conversion for Social Play, Competitions and Payments' actor columns

- **Status:** Accepted
- **Date:** 2026-08-16
- **Ticket:** T28.1 (`docs/process/t28-sprint-plan.md` §B7)
- **Closes:** the ruling half of #164 (Payments' implementation is a partial
  fix — see the PR; Social Play and Competitions are explicitly deferred to
  T29, not silently dropped, per §B8)
- **Relates to:** ADR-0014 (actor identifier space and the subject
  resolution seam — this ADR is the ruling ADR-0014 §5a/§7 deferred for
  exactly these three contexts), `db/migrations/0016_identity.sql` (which
  typed `identity_users.id` as `uuid` specifically so it would "stay a valid
  foreign-key target for every other context's eventual real reference" —
  the anticipation this ADR now cashes in), `db/migrations/0018_booking_
  recurring_hire_templates.sql` (the `uuid NOT NULL REFERENCES
  identity_users (id)` precedent this ADR extends)
- **Consumed by:** T28.1 (Payments, implemented this sprint),
  T29 (Social Play, Competitions — ruling only, implementation deferred)

## Context

ADR-0014 ruled, once, on *where* a subject gets translated to a `User.ID`
(the grpcapi boundary) and *what invariant that creates* (below the
boundary, an actor value is always a `User.ID`). Its own §5a explicitly
carved out three contexts and refused to resolve them in the same ticket:

> Social Play, Competitions and Payments store actor facts as plain text...
> Conformance for these three is deliberately deferred, because it is not a
> code change: it needs a backfill of existing rows plus a port and adapter
> per context.

Two things ADR-0014 left genuinely open, because closing them needed to see
a real backfill happen at least once first (§B9 of this sprint's plan
records that disagreement and resolves it in favor of proving the pattern
once, on Payments, before scaling to the other two):

1. Do the four "translate, not widen" reasons in ADR-0014 §7 actually apply
   to `recorded_by_user_id`/`host_id`/`player_id`, or were they only ever
   argued for Booking's `requested_by_user_id`?
2. Once a column is backfilled, does it become a real `uuid` foreign key, and
   what happens to a row whose text value matches no `identity_users.subject`
   — an actor who wrote a row before ever calling `CreateUser`?

This ADR answers both, for all three contexts at once, so T29 does not have
to re-litigate either question — only re-verify the ruling still holds and
execute the backfill for its two contexts.

## Decision 1 — ADR-0014 §7's four reasons, re-derived against these three contexts specifically

ADR-0014 §7 rejected widening Booking's actor column to `text` and storing
subjects, for four reasons. Re-argued here against `recorded_by_user_id`,
`host_id`, and `player_id` by name, not asserted to "still apply":

1. **Leaks the IdP identifier into six other tables.** This reason has
   nothing Booking-specific in it — it is about what a column *is*, not which
   context owns it. `payments.recorded_by_user_id`, `games.host_id`,
   `registrations.player_id`, `waitlist_entries.player_id`, and
   `competitions.host_id`/`player_id` would each carry `auth0|...`-shaped
   provider strings if left as-is with subjects written into them, in the
   *same* six-plus tables `0019_identity_subject.sql` already argued against
   widening. Re-deriving it per column changes nothing about the shape of
   the argument, because the argument was never about Booking's schema in
   particular.
2. **Destroys the foreign key.** This is the sharpest of the four against
   these three specifically, because none of the three currently *has* a
   foreign key to destroy — `games.host_id`, `registrations.player_id`, and
   `payments.recorded_by_user_id` are all plain `text` today, predating
   Identity's existence (each column's own migration says so:
   `0005_socialplay.sql`, `0014_competitions.sql`, `0005_payments.sql`).
   Widening to `text`-holding-subjects would not merely fail to add a
   missing FK, it would foreclose ever adding one later on the same
   reasoning as ADR-0014 §7.2 — a `text` column holding an IdP-owned string
   can reference nothing this backend controls, permanently, not just today.
3. **Makes an IdP swap a whole-database migration.** Applies identically:
   Social Play/Competitions/Payments are three more tables that would need
   re-keying on every future IdP change if they held subjects, exactly the
   blast radius ADR-0014 §7.3 described for Booking, just with three more
   contexts added to the list this ADR is being asked to grow.
4. **Not actually cheaper.** ADR-0014 §7.4's argument was that the request
   already touches Identity for a role check, so translation adds no new
   round-trip *class*. That specific argument does not carry over
   unchanged — Social Play/Competitions/Payments' authorization checks
   (`Registration.Cancel`, `authorizeOfflineRecording`, etc.) are
   object-level ownership comparisons, not role checks, so there is no
   existing Identity round trip to piggyback on. But the underlying claim —
   "one keyed read on a `UNIQUE` column is not the round-trip class this
   trade should be decided on" — holds independently: `UserIDBySubject`
   resolves against `identity_users_subject_key` (the same unique index
   ADR-0014's table cites), a single indexed lookup, and it happens once per
   authenticated RPC at the funnel, not once per authorization branch inside
   it. The absolute cost is a few hundred microseconds against a b-tree
   lookup; the alternative is permanently forfeiting reason 2. This is not a
   close call.

**Ruling: ADR-0014's "translate, not widen" ruling extends unchanged to
Social Play, Competitions, and Payments.** No column in any of the three
widens to hold a subject. This was ADR-0014 §5a's explicit deferral, not a
live question — restated here, with its own reasoning, so this ADR is
self-contained and T29 does not have to open ADR-0014 to find it.

## Decision 2 — the target shape: real `uuid` foreign keys, once backfilled

`0016_identity.sql` gave `identity_users.id` no `DEFAULT` specifically so it
would stay caller-independent, but it typed the column `uuid`, "not `text`",
with this stated reason: it "stays a valid foreign-key target for every
other context's eventual real reference (mirrors `facilities.owner_id`'s
uuid typing ahead of a real FK target existing yet)". That sentence has been
sitting unclaimed since T10.2. Booking's own
`recurring_hire_templates.requested_by_user_id uuid NOT NULL REFERENCES
identity_users (id)` (`0018_booking_recurring_hire_templates.sql`) is the
one column in this schema that already cashed it in.

> **Ruling: yes.** Once backfilled, every one of these columns becomes a
> real `uuid` foreign key to `identity_users(id)`:
>
> - `payments.recorded_by_user_id` → `uuid REFERENCES identity_users (id)`,
>   **nullable** (unchanged nullability — see Decision 3; the column is
>   nullable today and this ADR does not tighten it).
> - `games.host_id`, `registrations.player_id`, `waitlist_entries.player_id`,
>   `competitions.host_id`, `competitions.player_id` → `uuid NOT NULL
>   REFERENCES identity_users (id)` (unchanged nullability — every one of
>   these is `NOT NULL` today; T29's backfill must confront the orphan case
>   against that stricter starting point, per Decision 3 below).
>
> This is the target shape for **all three contexts**, stated once so T29's
> Ceremony 1 does not re-decide it — only re-verify it still holds and do
> the column-by-column execution for its two.

This is not a new design — it is Booking's already-shipped, already-reviewed
precedent, generalized from one column to the remaining six.

## Decision 3 — the orphan case

A row written before its actor ever called `CreateUser` has a `text` value
in the old column matching no `identity_users.subject`. The backfill must
decide what happens to that row's new `uuid` column.

Two options were on the table:

- **(a) Fail the migration** if any row is unresolvable, forcing every
  historical actor fact to be reconciled (or the row deleted/hand-fixed)
  before the fix for every other, resolvable row can ship.
- **(b) Leave the new column `NULL` for an orphaned row**, and let the
  migration otherwise succeed.

> **Ruling: (b), confirmed.** An unresolvable historical actor fact must not
> block deploying the fix for every resolvable one. The two failure modes
> this is weighed against are not symmetric:
>
> - Failing the migration outright means the conformance fix — which closes
>   a real defect class (ADR-0014 §7.1/§7.2, restated in Decision 1 above)
>   — cannot ship at all until every historical row is individually
>   reconciled, on a table this backend has no tooling to reconcile
>   automatically (there is no oracle that can invent the `User.ID` a
>   pre-Identity actor string should have mapped to; if there were, it would
>   not be an orphan). That is an indefinite block on a security-relevant
>   fix, held hostage by data this backend cannot regenerate.
> - Leaving the column `NULL` is not silent data loss. The **old** value is
>   not discarded into `NULL` and forgotten — see the migration's own
>   backfill step, which reads `payments.recorded_by_user_id`'s pre-migration
>   text value from the same row it is converting and only overwrites the
>   column after computing the join. Nothing about this ADR asks a future
>   migration to also drop an audit trail of what the unresolved value
>   *was*; T28.1's migration is scoped to the column-type conversion this
>   ticket exists for, and preserving the discarded raw value as a separate
>   audit column, if ever wanted, is real, additional scope this ADR
>   deliberately does not fold in here (see "What this ADR does not decide"
>   below).
>
> **`NULL` is the correct domain answer, not a default.** Every consumer of
> these columns already has to handle "no recorded actor" as a real state,
> independent of this migration: `payments.recorded_by_user_id` was already
> nullable before this ticket, and `app.authorizeOnlineConfirmation`
> already treats an empty `RecordedByUserID` as "nobody may confirm this",
> never "anybody may" (T13.7, closing #148) — see that function's own doc
> comment. `NULL` after backfill is the same fact this codebase already
> represents as an empty string before it: "this Payment's actor is
> unknown", which the authorization code already fails closed on. Defaulting
> to an empty string instead of `NULL` was explicitly rejected — a Postgres
  `uuid` column holding `''` is not valid input at all (it would not even
  `Scan`), so the only two honest choices were `NULL` or inventing a
  sentinel uuid, and a sentinel uuid would be indistinguishable from a real
  `identity_users.id` to any FK-unaware code path that later starts trusting
  the column type. `NULL` is the one representation that cannot be confused
  with a resolved answer.

**Restated for the `NOT NULL` starting point Social Play/Competitions
present (per this ADR's own instruction to name this explicitly for T29):**
`games.host_id`/`registrations.player_id`/`competitions.host_id`/
`competitions.player_id` are `NOT NULL` today. This ADR's ruling (b) still
applies to the *new* `uuid` column during the backfill window — the new
column must itself be added nullable, take whatever `NULL`s the join
produces, and only then be evaluated for a `NOT NULL` constraint. T29 must
not add `NOT NULL` to the new column in the same migration that backfills
it if any row is genuinely orphaned: doing so would resurrect option (a) — a
single unresolvable row blocking the entire migration — by a different
route, exactly the outcome this ADR rules against. If T29's backfill finds
zero orphans in practice (plausible for Social Play/Competitions, since
every write path already requires a verified principal as of T12.7/T12.8,
unlike Payments' older, pre-auth offline-recording rows), the `NOT NULL`
constraint may be added in the same migration once the backfill is
observed clean; if any orphan is found, the column must ship nullable and a
`NOT NULL` tightening is a separate, later migration once every orphan is
individually resolved. This is an instruction to T29, not a ruling this ADR
enforces itself.

## Consequences

- **Payments ships this sprint (T28.1).** Its column is the smallest,
  already-nullable, single-column case — the right first proof of this
  ADR's ruling, per this sprint's own §B9 resolution.
- **Social Play and Competitions are explicitly deferred to T29**, not
  silently dropped (§B8). This ADR is written so T29's Ceremony 1 opens with
  a ruling already made, not a re-derivation.
- **`recorded_by_user_id`/`host_id`/`player_id` all remain a Go `string` at
  the domain layer** (a uuid string, not a Go UUID type) — this ADR rules on
  storage and identifier space, not on Go representation, and does not
  disturb every other actor field's existing convention.
- **A read of one of these columns can now come back `NULL`/empty for a row
  that is genuinely known to have an actor** (the pre-Identity orphan case).
  Code reading these columns must keep treating empty as "unknown", not
  retrofit an assumption that a non-nil `Payment`/`Game`/`Registration`
  always has a resolvable actor. This was already true before this ADR for
  every reason a value could be empty; the orphan case is one more entry in
  an already-open set, not a new kind of hole.

## What this ADR does not decide

- **Preserving the raw pre-migration text value as a separate audit
  column.** Real, disclosed, out of scope — see Decision 3.
- **Whether T29 takes Social Play and Competitions together or separately.**
  Left to T29's own Ceremony 1, per this sprint's §B9.
- **Anything about `booking_host_id`'s identifier space (issue #149).** That
  is a caller-supplied *ownership fact* Payments is told, not a resolvable
  identifier-space question this ADR's port-and-backfill pattern addresses —
  see the T28.1 PR description for why this ticket does not narrow #149,
  and its disclosed note on the one place the two interact once the actor
  funnel resolves.
- **Role or permission modelling.** Unchanged, identical to ADR-0014's own
  scope boundary.
