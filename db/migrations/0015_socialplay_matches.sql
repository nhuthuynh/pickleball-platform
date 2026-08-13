-- Social Play `matches` table (T10.4, docs/process/t10-sprint-plan.md T10.4
-- section): the Postgres home for internal/socialplay/domain.Match (T10.3),
-- a recorded result against an existing Game.
--
-- 0015 is the next free number: 0014 was taken by T9.4
-- (0014_competitions.sql), confirmed by listing db/migrations before naming
-- this file, per that migration's own precedent (its doc comment records
-- 0013 was already taken by T9.2 when it was authored, and renumbers rather
-- than collide). T10.2 (Identity/Users' own Postgres wiring) is running in
-- an isolated worktree in parallel with this ticket and may also be
-- claiming a migration number around here; at the time this file was
-- authored, T10.2 had not yet pushed a branch with a migrations/ change to
-- check against (confirmed by inspection — see this PR's description). If
-- 0015 collides with a T10.2 migration by the time both land, the PR that
-- merges second renumbers on its own source branch rather than on the
-- shared branch, per CLAUDE.md rule 9 and this migration's own precedent
-- above.
--
-- Prototype-only migration tooling: applied via docker-compose initdb.d on a
-- FRESH volume only (see CLAUDE.md gotchas) — after this lands, `make down`
-- (which drops the volume) then `make up`. Adopt golang-migrate/goose before
-- production.
--
-- No EXCLUDE/UNIQUE concurrency guard here, unlike registrations/
-- competition_entries: a Match has no "only one may win" invariant for
-- Postgres to race-close (CLAUDE.md rule 4 applies where there's a real
-- invariant to enforce — recording a Match is a plain insert, not a
-- contested resource). The one real precondition this ticket adds
-- (recording a match against a cancelled Game -> FailedPrecondition) is
-- enforced in the domain/app layer (domain.Game.EnsureNotCancelled,
-- checked by app.Service.RecordMatchResult under the row Postgres already
-- returns for game_id's FK reference below) — there is no DB-level
-- analogue to add here the way the capacity guard trigger backstops
-- Register's own pre-check, because there is no cross-request race to
-- close: whether a specific Game is cancelled doesn't change between two
-- concurrent RecordMatchResult calls the way "how many slots are left"
-- does.

CREATE TABLE matches (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    -- game_id is a real FK into games — unlike games.host_id/
    -- registrations.player_id (opaque text, no Users context yet), Match
    -- always references an existing, same-context Game, so a real FK is
    -- both possible and correct here, mirroring registrations.game_id's
    -- own FK exactly.
    game_id      uuid NOT NULL REFERENCES games (id),
    -- players is text[], not uuid[], for the same reason
    -- registrations.player_id is plain text (db/migrations/
    -- 0005_socialplay.sql): there is no Users bounded context wiring these
    -- ids to real UUIDs yet, so they're opaque identifiers. CHECK mirrors
    -- domain.RecordMatch's own >= 2 players rule (CLAUDE.md rule 4: the
    -- invariant is expressed in both the domain and the database).
    players      text[] NOT NULL CHECK (cardinality(players) >= 2),
    -- score is a per-player point-total map (domain.Match.Score,
    -- map[string]int) stored as jsonb — the first jsonb column in this
    -- codebase (confirmed by inspection: no other migration uses it). A
    -- normalized child table (one row per (match_id, player_id, points))
    -- was considered and rejected: Score is a value, not an independently
    -- queryable/addressable entity (nothing in this ticket, or named as
    -- planned work, ever queries "find matches where player X scored N"),
    -- and T10.3's own doc comment already commits to keeping this shape
    -- "the simplest that records a real result." jsonb (not json) so
    -- Postgres normalizes/validates the structure on write. NOT NULL:
    -- domain.RecordMatch's own ErrEmptyScore (T10.4) already rejects an
    -- empty map before this table is ever reached, so there is no
    -- legitimate "no score" state to represent as NULL.
    score        jsonb NOT NULL,
    recorded_at  timestamptz NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX matches_game_idx ON matches (game_id);
