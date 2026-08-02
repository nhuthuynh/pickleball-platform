-- Social Play capacity guard (T5.4 loop-2 fix, PM+PE review on PR #14):
-- registrations_active_player_per_game_idx (0005_socialplay.sql) closes the
-- no-double-registration race, but nothing in Postgres previously enforced
-- "active registrations for a game <= games.capacity" — that lived only in
-- app.Service.RegisterForGame's get -> list -> insert round trip and
-- domain.Register's in-process count check, with no row lock. Two
-- concurrent RegisterForGame calls for the last open slot could both pass
-- the in-process check and both insert successfully, since they target
-- different (game_id, player_id) keys the unique index doesn't cover. This
-- violates CLAUDE.md rule 4 (invariants enforced in Postgres AND the
-- domain) for the one invariant the T5 sprint goal names explicitly.
--
-- Fix, mirroring ADR-0001's dual-invariant pattern (docs/adr/
-- 0001-dual-invariant-enforcement.md) the same way the booking context's
-- EXCLUDE constraint backstops domain.EnsureNoConflict: a BEFORE
-- INSERT/UPDATE trigger that locks the owning games row (SELECT ... FOR
-- UPDATE), counts non-cancelled registrations for that game_id, and raises
-- an exception if accepting this row would exceed games.capacity. The
-- FOR UPDATE lock is what actually closes the race: it serializes
-- concurrent transactions touching the same game_id on that row lock, so
-- the second transaction's count only proceeds after the first one's
-- INSERT has committed (and is therefore visible to its own count).
-- domain.Register's own check remains the fast-path pre-check, exercised by
-- fast dependency-free unit tests, exactly as EnsureNoConflict is for
-- bookings.
--
-- Only INSERT and the "un-cancel" UPDATE transition need guarding.
-- Registration.Cancel (internal/socialplay/domain/registration.go) only
-- allows registered -> cancelled, never the reverse, so a row going FROM
-- cancelled TO active is not a transition anything in this codebase
-- produces today -- but the trigger still guards it defensively, the same
-- way the unique index doesn't assume application discipline either.
--
-- Prototype-only migration tooling: applied via docker-compose initdb.d on
-- a FRESH volume only (see CLAUDE.md gotchas), so this must be a new file,
-- not an edit to the already-applied/reviewed 0005_socialplay.sql.

CREATE OR REPLACE FUNCTION enforce_game_capacity() RETURNS trigger AS $$
DECLARE
    game_capacity integer;
    active_count  integer;
BEGIN
    -- A row landing in (or staying in) 'cancelled' never consumes a slot.
    IF NEW.status = 'cancelled' THEN
        RETURN NEW;
    END IF;

    -- A row that was already active before this UPDATE and stays active
    -- (e.g. a future payment_status-only update) isn't claiming a new
    -- slot -- skip the check. Only INSERT and cancelled->active UPDATEs
    -- reach the count below.
    IF TG_OP = 'UPDATE' AND OLD.status <> 'cancelled' THEN
        RETURN NEW;
    END IF;

    -- Lock the owning game row so concurrent inserts/updates for the same
    -- game_id serialize here -- this is the actual race fix, not just the
    -- count that follows it.
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

    IF active_count >= game_capacity THEN
        RAISE EXCEPTION 'socialplay: game % is at capacity (%/%)',
            NEW.game_id, active_count, game_capacity
            USING ERRCODE = 'P0001';
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER registrations_capacity_guard
    BEFORE INSERT OR UPDATE ON registrations
    FOR EACH ROW
    EXECUTE FUNCTION enforce_game_capacity();
