-- Social Play waitlist schema (T6.6 / ADR-0006): waitlist_entries, the
-- queue Players join when a Game is full, plus the extension to
-- enforce_game_capacity() (db/migrations/0006_socialplay_capacity_guard.sql)
-- that makes an unexpired promoted entry reserve its freed slot at the DB
-- level, not just in app.Service (see that method's doc comment,
-- internal/socialplay/app/service.go).
--
-- game_id/player_id follow 0005_socialplay.sql's existing conventions:
-- game_id is a real FK (games already exists), player_id stays plain text
-- (no Users bounded context yet, same as registrations.player_id).
--
-- Prototype-only migration tooling: applied via docker-compose initdb.d on a
-- FRESH volume only (see CLAUDE.md gotchas). This is a NEW file, not an edit
-- to an already-applied migration.

CREATE TABLE waitlist_entries (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    game_id      uuid NOT NULL REFERENCES games (id),
    player_id    text NOT NULL,
    position     integer NOT NULL CHECK (position > 0),
    status       text NOT NULL DEFAULT 'waiting'
                     CHECK (status IN ('waiting', 'promoted', 'expired', 'cancelled')),
    promoted_at  timestamptz NULL,
    created_at   timestamptz NOT NULL DEFAULT now()
);

-- The no-double-waitlisting invariant (CLAUDE.md rule 4 — mirrors
-- registrations_active_player_per_game_idx exactly): a player may hold at
-- most one active (waiting/promoted) entry per game. A partial unique index
-- so a player can rejoin after their entry expires or they cancel it.
-- internal/socialplay/adapter/postgres translates this constraint's 23505
-- violation into domain.ErrAlreadyOnWaitlist.
CREATE UNIQUE INDEX waitlist_entries_active_player_per_game_idx
    ON waitlist_entries (game_id, player_id)
    WHERE status IN ('waiting', 'promoted');

-- The read/ordering path: "every entry for this game, oldest first" and
-- "the oldest still-waiting entry for this game" both filter/sort on these
-- two columns together.
CREATE INDEX waitlist_entries_game_position_idx ON waitlist_entries (game_id, position);

-- Extend enforce_game_capacity() (0006) so an unexpired promoted waitlist
-- entry for a DIFFERENT player also counts as claiming a slot -- otherwise
-- a direct INSERT into registrations for some other player could still
-- succeed at the DB level during a promotion's response window even though
-- app.Service.RegisterForGame's own pre-check (domain.SlotReservedByPromotion)
-- already rejects it; this closes the same gap in Postgres, not just Go
-- (CLAUDE.md rule 4). The promoted player themself is exempted (their own
-- INSERT is the "confirm" and must succeed), mirroring
-- domain.SlotReservedByPromotion's own exemption exactly -- the response
-- window duration is duplicated here as a literal interval because Postgres
-- can't import domain.PromotionResponseWindow; if that constant ever
-- changes, this literal must be updated to match (see this function's
-- comment as the pointer back).
CREATE OR REPLACE FUNCTION enforce_game_capacity() RETURNS trigger AS $$
DECLARE
    game_capacity      integer;
    active_count       integer;
    reserved_by_others integer;
BEGIN
    IF NEW.status = 'cancelled' THEN
        RETURN NEW;
    END IF;

    IF TG_OP = 'UPDATE' AND OLD.status <> 'cancelled' THEN
        RETURN NEW;
    END IF;

    SELECT capacity INTO game_capacity
    FROM games
    WHERE id = NEW.game_id
    FOR UPDATE;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'socialplay: game % not found', NEW.game_id
            USING ERRCODE = 'foreign_key_violation';
    END IF;

    SELECT count(*) INTO active_count
    FROM registrations
    WHERE game_id = NEW.game_id
      AND status <> 'cancelled'
      AND id <> NEW.id;

    -- T6.6: unexpired promotions belonging to a DIFFERENT player than this
    -- row's own player_id also occupy a slot -- 30 minutes must match
    -- domain.PromotionResponseWindow (internal/socialplay/domain/waitlist.go).
    SELECT count(*) INTO reserved_by_others
    FROM waitlist_entries
    WHERE game_id = NEW.game_id
      AND status = 'promoted'
      AND player_id <> NEW.player_id
      AND promoted_at > now() - interval '30 minutes';

    IF active_count + reserved_by_others >= game_capacity THEN
        RAISE EXCEPTION 'socialplay: game % is at capacity (%/%, % reserved by pending waitlist promotions)',
            NEW.game_id, active_count, game_capacity, reserved_by_others
            USING ERRCODE = 'P0001';
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
