-- T29.1 (docs/process/t29-sprint-plan.md; partial fix for #164 — the
-- Competitions third, following T28.1's Payments reference implementation
-- and ADR-0017's ruling) — competitions.host_id, competition_entries.
-- player_id, and competition_admins.user_id/assigned_by become real `uuid`
-- foreign keys into identity_users(id). Also closes Competitions' half of
-- issue #237 (internal/payments/app.authorizeCompetitionEntryRecording's
-- resolved-actor-vs-still-subject-shaped-read regression T28.1 introduced as
-- a side effect) as a consequence of finishing this backfill, not by any
-- change under internal/payments/** (see this ticket's PR body §B3).
--
-- 0025 is the next free number: db/migrations ended at 0024
-- (0024_payments_recorded_by_user_id_uuid.sql, T28.1), confirmed by listing
-- the directory before naming this file rather than assuming the sprint
-- plan's number was still free. Pre-assigned to this ticket alone (sprint
-- plan §A4) — T29.2 (Social Play) claims 0026, a disjoint table set, so no
-- collision is possible either way.
--
-- game_admins/competition_admins were not named in ADR-0017 Decision 2's own
-- column table, but issue #237's scope note (and this sprint plan's §B4/§B7
-- correction to that table) establishes they must convert in the SAME pass
-- as host_id/player_id — 0021_competitions_competition_admins.sql's own
-- header comment already said so when that table was created ("THIS TABLE'S
-- user_id AND assigned_by MUST BE BACKFILLED IN THE SAME PASS"). This
-- migration is that pass, for all four columns across the three tables.
--
-- WHY THIS MUST LAND IN THE SAME PR AS THE grpcapi funnel CHANGE
--
-- internal/competitions/adapter/grpcapi/handler.go's actor() becomes a
-- *Handler method resolving a verified subject to a User.ID (uuid) in this
-- same PR, exactly as T28.1's Payments funnel change did.
-- domain.Competition.EnsureHost/EnsureHostOrCompetitionAdmin and
-- domain.CompetitionAdmin's own AssignCompetitionAdmin/
-- EnsureMayRevokeCompetitionAdmin compare that resolved actor against the
-- *stored* host_id/player_id/user_id — so the columns and the values
-- compared against them must change identifier space together. Landing this
-- migration without the funnel change (or vice versa) would silently compare
-- a uuid against a stale subject string for every request in the gap.
-- CLAUDE.md rule 9 already requires one PR; this is the concrete reason it
-- is also one *reviewed unit* for this specific change.
--
-- WHY ONE MIGRATION, NOT AN EXPAND/CONTRACT PAIR
--
-- Identical reasoning to 0024's own header, restated rather than merely
-- cross-referenced so a future reader of this file alone has the full
-- argument: db/migrations/*.sql apply via docker-compose initdb.d only on a
-- FRESH volume (CLAUDE.md gotchas) — every environment that runs this
-- migration also runs `make down` (drops the volume) then `make up` first,
-- applying every migration from 0001 through this one in a single shot
-- BEFORE the application ever starts serving traffic. There is no live,
-- already-running deployment with pre-T29.1 application code and pre-T29.1
-- schema state that this migration rolls forward underneath, so the
-- two-phase expand/contract discipline a real production migration of this
-- shape would need has nothing to protect against here. Prototype-only
-- migration tooling; adopt golang-migrate/goose with a real expand/contract
-- split before production.
--
-- THE BACKFILL, AND THE ORPHAN CASE (ADR-0017 Decision 3)
--
-- Each of the four columns currently holds either an IdP subject string
-- written by the pre-conformance actor(ctx) (auth.RequireSubject
-- unresolved), for competitions.host_id/competition_entries.player_id, or
-- likewise for competition_admins.user_id/assigned_by (0021's own header
-- comment). Each backfill below joins the old text value against
-- identity_users.subject (the UNIQUE mapping column
-- db/migrations/0019_identity_subject.sql added) and takes that row's id,
-- following the exact join shape 0024's migration used.
--
-- NULLABLE, not NOT NULL, for all four columns in this migration —
-- DELIBERATELY DIFFERENT from Booking's recurring_hire_templates precedent
-- and from this ticket's own instructions' stated target shape, even though
-- ADR-0017 Decision 3's own restatement for Social Play/Competitions permits
-- NOT NULL "if the backfill finds zero orphans in practice". Two reasons,
-- both stated so a future reader does not read this as an oversight:
--
--   1. This ticket's own required mutation-check (CLAUDE.md rule 10,
--      T29.1 instruction 8 — see the PR body for the real-Postgres run)
--      seeds one genuinely orphaned subject per table and must observe this
--      exact migration file leave that row's new column NULL, not abort.
--      Shipping NOT NULL in the same migration would make that observation
--      impossible to make against the REAL migration text: a NOT NULL
--      constraint (or, for competition_admins.user_id specifically, a
--      PRIMARY KEY, which Postgres also forces NOT NULL) would reject the
--      whole statement the instant it met a genuine orphan, resurrecting
--      exactly the "one unresolvable row blocks the entire migration"
--      failure mode (option (a)) ADR-0017 Decision 3 rules against — by a
--      different route than an explicit `NOT NULL` on a text-vs-subject
--      mismatch, but the identical outcome.
--   2. This repo's fresh-volume-only deployment model (see above) means
--      these four columns are backfilled against zero pre-existing rows in
--      EVERY environment this migration is actually deployed to today, so
--      there is no real historical orphan population to observe zero of —
--      only a structural argument that there could never be one *under the
--      current deployment model*. That argument is real (0024's migration
--      relies on the identical one to skip expand/contract) but it is a
--      property of how this prototype is deployed, not a fact this
--      migration's own SQL can verify about data it will actually see.
--      Tightening to NOT NULL once a real deployment confirms zero orphans
--      against its own live data is a separate, later migration — out of
--      this ticket's scope, per ADR-0017 Decision 3's own "if any orphan is
--      found, the column ships nullable and a follow-up tightening is a
--      separate, later migration" instruction, applied here pre-emptively
--      rather than reactively because reason 1 above makes "wait and see"
--      unverifiable in-migration anyway.
--
-- Every consumer of these four columns already treats an empty/NULL actor as
-- "unknown, never authorized" — domain.Competition.EnsureHost/
-- EnsureHostOrCompetitionAdmin and domain.HasCompetitionAdmin's blank-actor
-- guards (see those functions' own doc comments) — so a NULL after this
-- migration is exactly as unauthorizable as the un-joinable string it
-- replaces, not a new kind of hole (mirrors ADR-0017 Decision 3's identical
-- point about payments.recorded_by_user_id).
--
-- competition_admins' composite PRIMARY KEY (competition_id, user_id) is
-- DROPPED and NOT recreated as a PRIMARY KEY, because a PRIMARY KEY column
-- can never hold NULL and user_id is now nullable (see above) — recreating
-- it verbatim would silently force NOT NULL back onto user_id via a
-- different route than an explicit column constraint, defeating the
-- nullable decision this migration just made. It is recreated as a plain
-- UNIQUE constraint instead: PostgreSQL's default UNIQUE semantics treat
-- NULL as distinct from NULL, so this still gives every RESOLVED
-- Competition-Admin assignment the identical no-duplicate-assignment
-- guarantee the PRIMARY KEY gave, while tolerating (not deduplicating)
-- orphaned rows — which is fine, since an orphaned admin row is exactly as
-- unauthorizable as any other orphaned actor fact per ADR-0017 Decision 3,
-- and nothing about authorization correctness depends on two orphaned rows
-- being detected as "the same" orphan. competition_admins_user_idx (the
-- reverse "which Competitions is this user an admin of" lookup, added by
-- 0021 for Payments' own T15.5 read) is dropped and recreated against the
-- new column unchanged in shape.

-- competitions.host_id -------------------------------------------------------

ALTER TABLE competitions
    ADD COLUMN host_id_uuid uuid REFERENCES identity_users (id);

UPDATE competitions c
SET host_id_uuid = u.id
FROM identity_users u
WHERE c.host_id = u.subject;

ALTER TABLE competitions DROP COLUMN host_id;
ALTER TABLE competitions RENAME COLUMN host_id_uuid TO host_id;

-- competition_entries.player_id ----------------------------------------------

ALTER TABLE competition_entries
    ADD COLUMN player_id_uuid uuid REFERENCES identity_users (id);

UPDATE competition_entries e
SET player_id_uuid = u.id
FROM identity_users u
WHERE e.player_id = u.subject;

ALTER TABLE competition_entries DROP COLUMN player_id;
ALTER TABLE competition_entries RENAME COLUMN player_id_uuid TO player_id;

-- competition_admins.user_id / assigned_by -----------------------------------

ALTER TABLE competition_admins
    ADD COLUMN user_id_uuid uuid REFERENCES identity_users (id),
    ADD COLUMN assigned_by_uuid uuid REFERENCES identity_users (id);

UPDATE competition_admins a
SET user_id_uuid = u.id
FROM identity_users u
WHERE a.user_id = u.subject;

UPDATE competition_admins a
SET assigned_by_uuid = u.id
FROM identity_users u
WHERE a.assigned_by = u.subject;

-- The composite PRIMARY KEY and the reverse-lookup index both reference the
-- OLD text user_id column and must be dropped before that column can be —
-- DROP COLUMN alone would fail ("cannot drop column user_id because other
-- objects depend on it") otherwise.
ALTER TABLE competition_admins DROP CONSTRAINT competition_admins_pkey;
DROP INDEX competition_admins_user_idx;

ALTER TABLE competition_admins DROP COLUMN user_id;
ALTER TABLE competition_admins RENAME COLUMN user_id_uuid TO user_id;
ALTER TABLE competition_admins DROP COLUMN assigned_by;
ALTER TABLE competition_admins RENAME COLUMN assigned_by_uuid TO assigned_by;

-- Recreated against the new column, per this migration's header note above:
-- a UNIQUE constraint (not a PRIMARY KEY — user_id is nullable now) plus the
-- reverse-lookup index, unchanged in shape from 0021.
ALTER TABLE competition_admins
    ADD CONSTRAINT competition_admins_competition_id_user_id_key UNIQUE (competition_id, user_id);
CREATE INDEX competition_admins_user_idx ON competition_admins (user_id);
