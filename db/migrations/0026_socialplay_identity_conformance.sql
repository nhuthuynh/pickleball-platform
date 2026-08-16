-- T29.2 (docs/process/t29-sprint-plan.md; closes the Social Play third of
-- #164, the last of the three ADR-0014 §5a deferred — Booking/Facilities
-- T13.2/T13.3, Payments T28.1, Social Play this ticket) — games.host_id,
-- registrations.player_id, waitlist_entries.player_id, and
-- game_admins.user_id/assigned_by all become real `uuid` foreign keys into
-- identity_users(id), per ADR-0017's ruling (Decisions 1-3, restated for
-- Social Play/Competitions specifically, and Decision 2's own column table
-- corrected by this sprint's plan to include game_admins/competition_admins
-- — see docs/process/t29-sprint-plan.md §B4/§B7 and issue #164's own
-- comment thread for the disclosure).
--
-- 0026 is the next free number: db/migrations ended at 0024
-- (0024_payments_recorded_by_user_id_uuid.sql) at the time this ticket's
-- branch was cut, confirmed by listing the directory before naming this
-- file rather than assuming the sprint plan's number was still free. 0025 is
-- pre-assigned to T29.1 (Competitions, same wave, disjoint tables) and does
-- not exist in this checkout yet — no collision is possible either way,
-- since neither ticket's migration touches the other's tables.
--
-- Fixes Social Play's half of issue #237 (a live regression T28.1
-- introduced as a side effect: internal/payments/app.authorizeGameRecording
-- compares its own now-resolved actor against games.host_id/
-- registrations.player_id (transitively)/game_admins.user_id reads, which
-- stayed subject-shaped until this migration backfills them) as a SIDE
-- EFFECT of finishing #164's own scope — no internal/payments/** code
-- changes anywhere in this ticket (sprint plan §B3).
--
-- WHY THIS MUST LAND IN THE SAME PR AS THE grpcapi funnel change
--
-- internal/socialplay/adapter/grpcapi/handler.go's actor() funnel starts
-- resolving a verified IdP subject to a User.ID (uuid) in this same PR, and
-- so does its second caller, resolveUserID, for AssignGameAdmin's/
-- RevokeGameAdmin's caller-supplied admin-target subject. Every comparison
-- this migration's five converted columns feed into (Game.EnsureHost/
-- EnsureHostOrGameAdmin, Registration.Cancel, WaitlistEntry.Cancel,
-- HasGameAdmin) is only correct once BOTH the funnel and this migration have
-- landed — landing one without the other would silently compare a uuid
-- against a stale subject string (or vice versa) for every request in the
-- gap, exactly the hazard T28.1's own migration header names for Payments,
-- now restated here for Social Play's five columns across four tables.
-- CLAUDE.md rule 9 already requires one PR; this is the concrete reason it
-- is also one *reviewed unit* for this specific change.
--
-- WHY ONE MIGRATION, NOT AN EXPAND/CONTRACT PAIR
--
-- Same reasoning as db/migrations/0024_payments_recorded_by_user_id_uuid.sql
-- (read in full before writing this one): this prototype's migration
-- tooling applies db/migrations/*.sql via docker-compose initdb.d on a FRESH
-- volume ONLY (CLAUDE.md gotchas) — there is no live, already-running
-- deployment with pre-T29.2 application code and pre-T29.2 schema state that
-- this migration rolls forward underneath. Every environment that runs this
-- migration also runs `make down` (drops the volume) then `make up` first,
-- getting the new application code and the new schema atomically in the
-- same container rebuild. Adopt golang-migrate/goose with a real
-- expand/contract split before production.
--
-- THE BACKFILL, AND THE ORPHAN CASE (ADR-0017 Decision 3, restated for the
-- NOT NULL starting point Social Play presents)
--
-- All five columns are `text NOT NULL` today (games.host_id,
-- registrations.player_id, waitlist_entries.player_id — 0005_socialplay.sql/
-- 0007_socialplay_waitlist.sql; game_admins.user_id/assigned_by —
-- 0020_socialplay_game_admins.sql, whose own header comment says "THIS
-- TABLE'S user_id AND assigned_by MUST BE BACKFILLED IN THE SAME PASS" —
-- this migration is that pass). Unlike payments.recorded_by_user_id (already
-- nullable before T28.1), ADR-0017 Decision 3's own restatement for Social
-- Play/Competitions applies here: each new column is added NULLABLE first,
-- backfilled via the identical `UPDATE ... FROM identity_users` join T28.1's
-- migration used (keyed on `subject`, the UNIQUE mapping column
-- db/migrations/0019_identity_subject.sql added), and ONLY THEN evaluated
-- for `NOT NULL` — added in this same migration ONLY IF that table's own
-- backfill leaves zero NULLs, independently per table (Decision 3's own
-- instruction: one table having zero orphans does not imply another does).
--
-- THE BRANCH ACTUALLY TAKEN, PER TABLE (verified against a real local
-- Postgres this ticket — see the PR body's mutation-check section for the
-- full run log; CLAUDE.md rule 10)
--
-- `games`, `registrations`, `waitlist_entries`, and `game_admins` are all
-- populated ONLY by application traffic through this context's own RPCs —
-- no db/migrations/*seed*.sql file inserts into any of the four (checked:
-- 0002_seed.sql and 0004_seed_pricing.sql are the only seed migrations in
-- this repo, and neither touches Social Play's tables). Every environment
-- this migration can actually run against (CLAUDE.md's fresh-volume-only
-- gotcha, restated above) therefore starts these four tables at zero rows,
-- which makes the backfill's join vacuously orphan-free — there is no row
-- for the join to fail to resolve. `NOT NULL` is added, in this same
-- migration, for all five columns across all four tables. This is the
-- SAME branch ADR-0017 Decision 3 itself flags as plausible for Social
-- Play/Competitions specifically ("every write path already requires a
-- verified principal as of T12.7/T12.8") but does not assert outright; this
-- migration's own environment-level reasoning (zero seeded rows, not merely
-- "every write is authenticated") is the fuller argument for why zero
-- orphans is actually guaranteed here, not just likely. A future run of this
-- migration against a database that already holds real, non-empty
-- games/registrations/waitlist_entries/game_admins rows with a genuinely
-- unresolvable subject would fail loudly at the `SET NOT NULL` step (a clean
-- Postgres error, "column contains null values"), never silently — which is
-- the correct, safe failure mode ADR-0017 Decision 3 describes, not a
-- degraded case this migration tries to route around.
--
-- game_admins' composite PRIMARY KEY (game_id, user_id) and its
-- game_admins_user_idx index are explicitly dropped and recreated against
-- the new uuid column (rather than relying on DROP COLUMN's implicit
-- constraint/index cascade), so the recreation step is auditable directly
-- from this file rather than inferred from Postgres's cascade behaviour.
--
-- join_waitlist_entry (db/migrations/0009_socialplay_waitlist_join_position.sql)
-- is recreated with its p_player_id parameter changed from `text` to `uuid`,
-- to match waitlist_entries.player_id's new type — a plain CREATE OR REPLACE
-- cannot change a function's parameter types, so this is a DROP + CREATE.
-- Recreated LAST, after every column conversion below, so its body is
-- authored against the final schema shape rather than an intermediate one.
-- enforce_game_capacity() (0006/0007/0012_socialplay_guest_capacity.sql)
-- needs NO change: its `waitlist_entries.player_id <> NEW.player_id`
-- comparison (0007) reads both sides dynamically off the row types at
-- execution time, so once both columns are uuid the comparison is simply
-- uuid <> uuid — PL/pgSQL function bodies are not schema objects Postgres
-- tracks column-level dependencies against, so this migration does not (and
-- structurally cannot) break by leaving that function's text unchanged.

-- ===========================================================================
-- games.host_id: text NOT NULL -> uuid NOT NULL REFERENCES identity_users (id)
-- ===========================================================================

ALTER TABLE games
    ADD COLUMN host_id_uuid uuid REFERENCES identity_users (id);

UPDATE games g
SET host_id_uuid = u.id
FROM identity_users u
WHERE g.host_id = u.subject;

ALTER TABLE games ALTER COLUMN host_id_uuid SET NOT NULL;
ALTER TABLE games DROP COLUMN host_id;
ALTER TABLE games RENAME COLUMN host_id_uuid TO host_id;

-- ===========================================================================
-- registrations.player_id: text NOT NULL -> uuid NOT NULL REFERENCES identity_users (id)
-- ===========================================================================
--
-- registrations_active_player_per_game_idx (0005_socialplay.sql) is dropped
-- explicitly before the old column goes away and recreated against the new
-- one afterward, rather than relying on DROP COLUMN's implicit index
-- cascade — auditable directly from this file. registrations_game_idx
-- (game_id only) is untouched by this section.

DROP INDEX registrations_active_player_per_game_idx;

ALTER TABLE registrations
    ADD COLUMN player_id_uuid uuid REFERENCES identity_users (id);

UPDATE registrations r
SET player_id_uuid = u.id
FROM identity_users u
WHERE r.player_id = u.subject;

ALTER TABLE registrations ALTER COLUMN player_id_uuid SET NOT NULL;
ALTER TABLE registrations DROP COLUMN player_id;
ALTER TABLE registrations RENAME COLUMN player_id_uuid TO player_id;

CREATE UNIQUE INDEX registrations_active_player_per_game_idx
    ON registrations (game_id, player_id)
    WHERE status <> 'cancelled';

-- ===========================================================================
-- waitlist_entries.player_id: text NOT NULL -> uuid NOT NULL REFERENCES identity_users (id)
-- ===========================================================================
--
-- waitlist_entries_active_player_per_game_idx (0007_socialplay_waitlist.sql)
-- gets the identical drop-and-recreate treatment as registrations' own
-- index above. waitlist_entries_game_position_idx (game_id, position) is
-- untouched by this section.

DROP INDEX waitlist_entries_active_player_per_game_idx;

ALTER TABLE waitlist_entries
    ADD COLUMN player_id_uuid uuid REFERENCES identity_users (id);

UPDATE waitlist_entries w
SET player_id_uuid = u.id
FROM identity_users u
WHERE w.player_id = u.subject;

ALTER TABLE waitlist_entries ALTER COLUMN player_id_uuid SET NOT NULL;
ALTER TABLE waitlist_entries DROP COLUMN player_id;
ALTER TABLE waitlist_entries RENAME COLUMN player_id_uuid TO player_id;

CREATE UNIQUE INDEX waitlist_entries_active_player_per_game_idx
    ON waitlist_entries (game_id, player_id)
    WHERE status IN ('waiting', 'promoted');

-- ===========================================================================
-- game_admins.user_id / game_admins.assigned_by:
--   text NOT NULL -> uuid NOT NULL REFERENCES identity_users (id)
-- ===========================================================================
--
-- Both columns convert in the same pass, per 0020_socialplay_game_admins.sql's
-- own header instruction ("THIS TABLE'S user_id AND assigned_by MUST BE
-- BACKFILLED IN THE SAME PASS") and issue #237's scope note (game_admins was
-- not named in ADR-0017 Decision 2's own column table, but Decision 2's
-- reasoning already covers it — see this file's own top-of-file comment).
--
-- The composite PRIMARY KEY (game_id, user_id) and game_admins_user_idx are
-- explicitly dropped before the old user_id column goes away and recreated
-- against the new one afterward — ticket instruction 5's own requirement,
-- done explicitly here rather than left to DROP COLUMN's implicit cascade.

ALTER TABLE game_admins DROP CONSTRAINT game_admins_pkey;
DROP INDEX game_admins_user_idx;

ALTER TABLE game_admins
    ADD COLUMN user_id_uuid uuid REFERENCES identity_users (id),
    ADD COLUMN assigned_by_uuid uuid REFERENCES identity_users (id);

UPDATE game_admins ga
SET user_id_uuid = u.id
FROM identity_users u
WHERE ga.user_id = u.subject;

UPDATE game_admins ga
SET assigned_by_uuid = u.id
FROM identity_users u
WHERE ga.assigned_by = u.subject;

ALTER TABLE game_admins ALTER COLUMN user_id_uuid SET NOT NULL;
ALTER TABLE game_admins ALTER COLUMN assigned_by_uuid SET NOT NULL;

ALTER TABLE game_admins DROP COLUMN user_id;
ALTER TABLE game_admins DROP COLUMN assigned_by;
ALTER TABLE game_admins RENAME COLUMN user_id_uuid TO user_id;
ALTER TABLE game_admins RENAME COLUMN assigned_by_uuid TO assigned_by;

ALTER TABLE game_admins ADD PRIMARY KEY (game_id, user_id);
CREATE INDEX game_admins_user_idx ON game_admins (user_id);

-- ===========================================================================
-- join_waitlist_entry: p_player_id text -> uuid, to match
-- waitlist_entries.player_id's new type. DROP + CREATE, not CREATE OR
-- REPLACE, because a parameter type change is a different function
-- signature (CREATE OR REPLACE only permits replacing a function's body
-- against its EXISTING signature).
-- ===========================================================================

DROP FUNCTION join_waitlist_entry(uuid, uuid, text);

CREATE FUNCTION join_waitlist_entry(p_id uuid, p_game_id uuid, p_player_id uuid)
RETURNS SETOF waitlist_entries AS $$
DECLARE
    v_position integer;
    v_entry    waitlist_entries%ROWTYPE;
BEGIN
    -- Unchanged from 0009_socialplay_waitlist_join_position.sql: the actual
    -- race-closing lock, every concurrent join for this game_id (and every
    -- promote_next_waiting / enforce_game_capacity call for it too, since
    -- they lock the same row) serializes here.
    PERFORM 1 FROM games WHERE id = p_game_id FOR UPDATE;

    -- Unchanged: mirrors domain.JoinWaitlist's Position formula exactly.
    SELECT count(*) INTO v_position
    FROM waitlist_entries
    WHERE game_id = p_game_id
      AND status <> 'cancelled';

    INSERT INTO waitlist_entries (id, game_id, player_id, position, status)
    VALUES (p_id, p_game_id, p_player_id, v_position + 1, 'waiting')
    RETURNING * INTO v_entry;

    RETURN NEXT v_entry;
END;
$$ LANGUAGE plpgsql;
