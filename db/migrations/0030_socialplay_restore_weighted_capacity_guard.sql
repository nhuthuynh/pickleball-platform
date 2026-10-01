-- T61 — restore the WEIGHTED capacity guard that
-- 0023_socialplay_registration_status_guard.sql silently reverted.
--
-- THE DEFECT
--
-- 0012_socialplay_guest_capacity.sql (T8.7) rewrote enforce_game_capacity()
-- to compare a weighted SUM(1 + guest_count) against games.capacity, because
-- the original COUNT(*) version (0006_socialplay_capacity_guard.sql, T5.4)
-- counted a registration bringing three guests as one person. 0023 (T19.1)
-- then needed to add a cancelled-Game check to the same function, and did so
-- with `CREATE OR REPLACE FUNCTION enforce_game_capacity()` carrying a body
-- built from **0006's** version plus the new status check — so it shipped
-- 0012's COUNT(*) bug again, eleven migrations after it was fixed.
--
-- 0023's own header says enforce_game_capacity() is "unchanged by this
-- migration (that function is 0006's)". The first half is false and the second
-- half is why: by 0023 the function was 0012's, not 0006's, and the comment
-- records which file *created* the function rather than which file last
-- defined its body. A CREATE OR REPLACE is a whole-body replacement, so
-- "unchanged" was never a thing that migration could be.
--
-- WHAT WAS ACTUALLY BROKEN IN A DEPLOYED SCHEMA
--
-- Guests stopped counting toward Game capacity in Postgres from 0023 onward.
-- A 7-person Game accepted 7 registrations each bringing 3 guests — 28 people
-- in a 7-person Game. internal/socialplay/domain.Register's own check stayed
-- weighted the whole time (`activeWeight+(1+guestCount) > game.Capacity`), so
-- the two halves CLAUDE.md rule 4 requires to agree had disagreed since T19.1,
-- with the authoritative half being the wrong one. The domain pre-check hid it
-- for every sequential request; only concurrency, where the pre-check cannot
-- hold, exposed it.
--
-- HOW IT WENT UNNOTICED FOR 42 MIGRATIONS
--
-- guest_capacity_concurrency_integration_test.go has asserted exactly this
-- since T8.7 and fails loudly against 0023's body — but it is
-- `//go:build integration`, so no gate on a Docker-free machine ever ran it.
-- `make vet-integration` COMPILES it, which cannot execute an assertion. The
-- first full `make ci-integration` run on this project (T61) is what failed,
-- with the numbers 0012's own doc comment predicts for the COUNT(*) version:
-- 7 successes where 1 is correct, 28 occupied against capacity 7.
--
-- WHY THIS IS A NEW MIGRATION AND NOT AN EDIT TO 0023
--
-- Editing 0023 in place would make this invisible: the point on the record is
-- that a guard regressed and which change did it. A separate migration also
-- keeps the prototype's apply-from-0001-on-a-fresh-volume model honest for
-- anyone who has already applied 0023 (CLAUDE.md gotchas).
--
-- 0030 is the next free number: db/migrations ended at 0029
-- (0029_competitions_entry_amount_owed.sql), confirmed by listing the
-- directory.
--
-- THE BODY BELOW IS THE UNION OF BOTH, NOT A REVERT
--
-- 0012's weighted occupancy AND 0023's cancelled-Game check, in 0023's
-- ordering (game-not-bookable is the more fundamental fact and is raised
-- first, mirroring domain.Register). The lock, the two early-return branches
-- and both ERRCODEs are unchanged from what each migration established — see
-- 0006, 0012 and 0023's own headers for why each is shaped as it is. Nothing
-- here is new behaviour; every line is something one of those three migrations
-- already decided.
CREATE OR REPLACE FUNCTION enforce_game_capacity() RETURNS trigger AS $$
DECLARE
    game_capacity integer;
    game_status   text;
    active_weight integer;
    new_weight    integer;
BEGIN
    -- A row landing in (or staying in) 'cancelled' never consumes a slot,
    -- and is never blocked by its own Game's status (0023's header note on
    -- Registration.Cancel).
    IF NEW.status = 'cancelled' THEN
        RETURN NEW;
    END IF;

    -- A row that was already active before this UPDATE and stays active
    -- (e.g. a payment_status-only update) isn't claiming a new slot -- skip
    -- the check. Only INSERT and cancelled->active UPDATEs reach the
    -- occupancy calculation below.
    IF TG_OP = 'UPDATE' AND OLD.status <> 'cancelled' THEN
        RETURN NEW;
    END IF;

    -- Lock the owning game row so concurrent inserts/updates for the same
    -- game_id serialize here -- this is the actual race fix, not the weighted
    -- sum that follows it. capacity and status come from the one locked read,
    -- so the status check below observes the identical snapshot (0023).
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

    -- 0012 (T8.7): weighted occupancy of every OTHER active registration for
    -- this game -- each row occupies (1 + its own guest_count) slots,
    -- mirroring domain.Register exactly. This SUM is the line 0023 dropped.
    SELECT COALESCE(SUM(1 + guest_count), 0) INTO active_weight
    FROM registrations
    WHERE game_id = NEW.game_id
      AND status <> 'cancelled'
      AND id <> NEW.id;

    new_weight := 1 + NEW.guest_count;

    IF active_weight + new_weight > game_capacity THEN
        RAISE EXCEPTION 'socialplay: game % is at capacity (%/%, incl. guests)',
            NEW.game_id, active_weight + new_weight, game_capacity
            USING ERRCODE = 'P0001';
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
