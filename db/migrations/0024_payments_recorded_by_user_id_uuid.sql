-- T28.1 (docs/process/t28-sprint-plan.md §B7; closes the Payments third of
-- #164) — payments.recorded_by_user_id becomes a real `uuid` foreign key
-- into identity_users(id), per ADR-0017's ruling.
--
-- 0024 is the next free number: db/migrations ended at 0023
-- (0023_socialplay_registration_status_guard.sql), confirmed by listing the
-- directory before naming this file rather than assuming the sprint plan's
-- number was still free. Pre-assigned to this ticket alone
-- (sprint plan §A4) — no other T28 ticket claims a migration.
--
-- WHY THIS MUST LAND IN THE SAME PR AS THE grpcapi funnel change
--
-- internal/payments/adapter/grpcapi/handler.go's actor() funnel starts
-- resolving a verified IdP subject to a User.ID (uuid) in this same PR. Its
-- own doc comment and internal/payments/app/service.go's
-- authorizeOnlineConfirmation doc comment both walk through the hazard in
-- full: authorizeOnlineConfirmation compares that resolved actor against
-- the *stored* payments.recorded_by_user_id, so the column and the value
-- compared against it must change identifier space together. Landing this
-- migration without the funnel change (or vice versa) would silently
-- compare a uuid against a stale subject string for every request in the
-- gap. CLAUDE.md rule 9 already requires one PR; this is the concrete
-- reason it is also one *reviewed unit* for this specific change, not a
-- process rule applied for its own sake.
--
-- WHY ONE MIGRATION, NOT AN EXPAND/CONTRACT PAIR (PR body instruction 5)
--
-- A real production migration for a column-type conversion this shape would
-- normally split into an expand phase (add the new column, dual-write,
-- backfill, verify) and a contract phase (drop the old column) as two
-- separate deploys, so a mid-rollout process reading the old shape never
-- observes a half-migrated row. That two-phase discipline does not apply
-- here, and applying it anyway would be performing a ritual this tooling
-- cannot back up: CLAUDE.md's own migration-tooling gotcha states plainly
-- that db/migrations/*.sql apply via docker-compose initdb.d **only on a
-- fresh volume** — there is no live, already-running deployment with
-- pre-T28.1 application code and pre-T28.1 schema state that this migration
-- rolls forward underneath. Every environment that runs this migration also
-- runs `make down` (which drops the volume) then `make up` first, and gets
-- the new application code and the new schema atomically, in the same
-- container rebuild. A two-phase split would add real complexity (two
-- migration files, a dual-write window in application code that would then
-- need its own follow-up removal) to protect against a race this prototype
-- genuinely cannot have: there is no window in which old code and new
-- schema, or new code and old schema, coexist against the same running
-- database. Prototype-only migration tooling: applied via docker-compose
-- initdb.d on a FRESH volume only (see CLAUDE.md gotchas). Adopt
-- golang-migrate/goose with a real expand/contract split before production,
-- at which point this reasoning stops applying and a real two-phase
-- migration is the right call for this exact kind of change.
--
-- THE BACKFILL, AND THE ORPHAN CASE (ADR-0017 Decision 3)
--
-- payments.recorded_by_user_id currently holds either "" /nothing (no
-- actor recorded — legitimate, unrelated to this migration) or an IdP
-- subject string written by the pre-T28.1 actor(ctx) (auth.RequireSubject
-- unresolved). The backfill below joins each non-null value against
-- identity_users.subject (the UNIQUE mapping column
-- db/migrations/0019_identity_subject.sql added) and takes that row's id.
--
-- A row whose subject matches no identity_users row — an actor who
-- recorded or created an online Payment before ever calling CreateUser —
-- is an orphan. ADR-0017 Decision 3 rules explicitly: leave such a row's
-- new column NULL rather than fail this migration outright or invent a
-- default. The application code already treats a NULL/empty
-- RecordedByUserID as "nobody may confirm this" (app.
-- authorizeOnlineConfirmation, T13.7/#148), so an orphaned row does not
-- become newly confirmable or newly unconfirmable by this migration — it
-- is exactly as unconfirmable as it already was, for the identical reason,
-- now expressed as a real NULL instead of an opaque, un-joinable string.
--
-- Adds a new nullable column (NULL for every row by default — no DEFAULT
-- clause, deliberately), then backfills it with a single UPDATE ... FROM
-- join against identity_users.subject. A row whose subject matches no
-- identity_users row simply is not touched by the UPDATE, so it keeps the
-- NULL the ADD COLUMN step already gave it — orphan handling requires no
-- special-case branch, because "leave it NULL" is the join's natural
-- behaviour for a non-match, not a code path that has to be written. Then
-- the old text column is dropped and the new one renamed into its place —
-- in one migration, per the reasoning above.
--
-- NULLABLE, unlike Booking's requested_by_user_id (`uuid NOT NULL
-- REFERENCES identity_users (id)`, 0018): payments.recorded_by_user_id was
-- already nullable before this ticket (0005_payments.sql), and ADR-0017
-- Decision 2 rules this column's nullability stays unchanged — a Payment
-- genuinely can have no recorded actor (T13.7's own doc comment), and an
-- orphaned row must be representable without violating a NOT NULL
-- constraint the pre-migration schema never had.

ALTER TABLE payments
    ADD COLUMN recorded_by_user_id_uuid uuid REFERENCES identity_users (id);

UPDATE payments p
SET recorded_by_user_id_uuid = u.id
FROM identity_users u
WHERE p.recorded_by_user_id IS NOT NULL
  AND p.recorded_by_user_id = u.subject;

ALTER TABLE payments DROP COLUMN recorded_by_user_id;
ALTER TABLE payments RENAME COLUMN recorded_by_user_id_uuid TO recorded_by_user_id;
