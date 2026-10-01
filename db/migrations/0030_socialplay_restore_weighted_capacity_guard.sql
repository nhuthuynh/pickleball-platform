-- T61 — restore BOTH halves of enforce_game_capacity() that later
-- `CREATE OR REPLACE` redefinitions silently dropped.
--
-- THE DEFECTS — TWO OF THEM, IN ONE FUNCTION, ELEVEN MIGRATIONS APART
--
-- This function has been redefined five times (0006, 0007, 0012, 0023, and
-- this file). Twice, a redefinition built its new body from an older version
-- of the function instead of the then-current one, and so reverted a guard
-- that had already shipped:
--
--  1. **0012 dropped 0007's waitlist reservation.** 0007 (T6.6) added
--     `reserved_by_others`: an unexpired `promoted` waitlist entry belonging to
--     a DIFFERENT player also occupies a slot, so another player's direct
--     INSERT cannot take the slot a promotion is holding during its response
--     window. 0012 (T8.7) rewrote the function for guest weighting, from
--     **0006's** body, and the reservation vanished. It is not mentioned
--     anywhere in 0012. The Go half — `domain.SlotReservedByPromotion` and
--     `domain.PromotionResponseWindow` — has enforced it the whole time, so
--     from T8.7 a promoted player's reserved slot could be taken out from
--     under them by any concurrent direct registration.
--
--  2. **0023 dropped 0012's guest weighting.** 0023 (T19.1) added a
--     cancelled-Game check, building its body from **0006's** version too, and
--     reverted `SUM(1 + guest_count)` to a plain `COUNT(*)`. A 7-person Game
--     then accepted 7 registrations each bringing 3 guests — 28 people in a
--     7-person Game. `domain.Register` stayed weighted throughout. 0023's own
--     header asserts the function is "unchanged by this migration (that
--     function is 0006's)": the first half is false, and the second half is
--     why — it recorded which migration *created* the function rather than
--     which one last *defined* its body.
--
-- So 0012 is simultaneously the fix for one divergence and the cause of
-- another, and 0023 repeated the same mistake against 0012. Both left
-- CLAUDE.md rule 4's two halves disagreeing with the authoritative half wrong.
--
-- HOW BOTH WENT UNNOTICED
--
-- `guest_capacity_concurrency_integration_test.go` has asserted #2 since T8.7
-- and fails loudly on 0023's body. #1 had no test at all on either side of the
-- boundary — `waitlist_reservation_integration_test.go` (added by this ticket)
-- is its first. Neither could have been caught by a Docker-free gate: both
-- live behind `//go:build integration`, which `make vet-integration` compiles
-- and cannot execute, and no session in this project's history had run
-- `make ci-integration` until T61.
--
-- 0030 is the next free number: db/migrations ended at 0029
-- (0029_competitions_entry_amount_owed.sql), confirmed by listing the
-- directory.
--
-- WHY A NEW MIGRATION RATHER THAN EDITING 0012 AND 0023
--
-- Editing them in place would erase the part worth keeping on the record: that
-- a guard can regress through a redefinition that looks additive, twice, and
-- which changes did it. A separate migration also keeps the prototype's
-- apply-from-0001-on-a-fresh-volume model honest for anyone who already has
-- 0023 applied (CLAUDE.md gotchas).
--
-- THE BODY BELOW IS THE UNION OF ALL FOUR PREDECESSORS
--
-- 0006's lock and early returns, 0007's `reserved_by_others`, 0012's weighted
-- occupancy, and 0023's cancelled-Game check in 0023's ordering (a Game not
-- being bookable at all is the more fundamental fact, mirroring
-- domain.Register). Nothing here is new behaviour; every line is something one
-- of those four migrations already decided. See each of their headers for why
-- its own piece is shaped as it is.
--
-- The comparison is `active_weight + new_weight + reserved_by_others >
-- game_capacity`, which reduces **exactly** to each predecessor's own rule:
-- with every weight 1 it is 0007's `active_count + reserved_by_others >=
-- game_capacity`, and with no outstanding reservations it is 0012's
-- `active_weight + new_weight > game_capacity`. A reservation weighs 1 because
-- `waitlist_entries` has no `guest_count` column — a WaitlistEntry carries no
-- guests, in the schema or in the domain.
--
-- The 30-minute literal is still duplicated from
-- `domain.PromotionResponseWindow` for the reason 0007 gave: Postgres cannot
-- import a Go constant. If that constant changes, this literal must change
-- with it.
CREATE OR REPLACE FUNCTION enforce_game_capacity() RETURNS trigger AS $$
DECLARE
    game_capacity      integer;
    game_status        text;
    active_weight      integer;
    new_weight         integer;
    reserved_by_others integer;
BEGIN
    -- A row landing in (or staying in) 'cancelled' never consumes a slot, and
    -- is never blocked by its own Game's status (0023's header note on
    -- Registration.Cancel).
    IF NEW.status = 'cancelled' THEN
        RETURN NEW;
    END IF;

    -- A row that was already active before this UPDATE and stays active (e.g.
    -- a payment_status-only update) isn't claiming a new slot -- skip the
    -- check. Only INSERT and cancelled->active UPDATEs reach the occupancy
    -- calculation below.
    IF TG_OP = 'UPDATE' AND OLD.status <> 'cancelled' THEN
        RETURN NEW;
    END IF;

    -- Lock the owning game row so concurrent inserts/updates for the same
    -- game_id serialize here -- this is the actual race fix, not the sums that
    -- follow it. capacity and status come from the one locked read, so the
    -- status check below observes the identical snapshot (0023).
    SELECT capacity, status INTO game_capacity, game_status
    FROM games
    WHERE id = NEW.game_id
    FOR UPDATE;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'socialplay: game % not found', NEW.game_id
            USING ERRCODE = 'foreign_key_violation';
    END IF;

    -- 0023 (closes #212): a Registration cannot become (or stay) active
    -- against an already-cancelled Game.
    IF game_status = 'cancelled' THEN
        RAISE EXCEPTION 'socialplay: game % is cancelled', NEW.game_id
            USING ERRCODE = 'P0001';
    END IF;

    -- 0012 (T8.7): weighted occupancy of every OTHER active registration --
    -- each row occupies (1 + its own guest_count) slots, mirroring
    -- domain.Register exactly. This is the SUM 0023 dropped.
    SELECT COALESCE(SUM(1 + guest_count), 0) INTO active_weight
    FROM registrations
    WHERE game_id = NEW.game_id
      AND status <> 'cancelled'
      AND id <> NEW.id;

    new_weight := 1 + NEW.guest_count;

    -- 0007 (T6.6): an unexpired promotion belonging to a DIFFERENT player also
    -- occupies a slot, so this INSERT cannot take the slot that promotion is
    -- holding. The promoted player themself is exempted -- their own INSERT is
    -- the "confirm" and must succeed -- mirroring
    -- domain.SlotReservedByPromotion's identical exemption. This is the count
    -- 0012 dropped.
    SELECT count(*) INTO reserved_by_others
    FROM waitlist_entries
    WHERE game_id = NEW.game_id
      AND status = 'promoted'
      AND player_id <> NEW.player_id
      AND promoted_at > now() - interval '30 minutes';

    IF active_weight + new_weight + reserved_by_others > game_capacity THEN
        RAISE EXCEPTION 'socialplay: game % is at capacity (%/%, incl. guests; % reserved by pending waitlist promotions)',
            NEW.game_id, active_weight + new_weight, game_capacity, reserved_by_others
            USING ERRCODE = 'P0001';
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
