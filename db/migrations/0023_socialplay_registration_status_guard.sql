-- Social Play registration/waitlist cancelled-Game guard (T19.1, closes
-- #212): domain.Register and domain.JoinWaitlist gained an in-process check
-- (internal/socialplay/domain/registration.go, waitlist.go) rejecting an
-- attempt to register for, or join the waitlist of, a Game whose Status is
-- already 'cancelled' — a gap disclosed at T5.2, whose own stated closing
-- trigger ("when Game-cancellation cascading is built") fired at T16.3 with
-- nobody wiring the check until now. This migration is that invariant's
-- Postgres-side half (CLAUDE.md rule 4: invariants enforced in Postgres AND
-- expressed in the domain), applying ADR-0001's dual-invariant pattern
-- (docs/adr/0001-dual-invariant-enforcement.md) to a THIRD Social Play
-- invariant.
--
-- This is a CREATE OR REPLACE FUNCTION **redefinition** of two functions
-- already applied by earlier, already-reviewed migrations — NOT an edit to
-- those files, per CLAUDE.md's migration-append-only gotcha (prototype-only
-- tooling applies db/migrations/*.sql via docker-compose initdb.d, on a
-- FRESH volume only):
--
--   - enforce_game_capacity() — owned by, and originally defined in,
--     0006_socialplay_capacity_guard.sql (unchanged by this migration; that
--     file's own header and body remain exactly as reviewed on PR #14).
--   - join_waitlist_entry() — owned by, and originally defined in,
--     0009_socialplay_waitlist_join_position.sql (unchanged by this
--     migration; that file's own header and body remain exactly as reviewed
--     on PR #25).
--
-- Both functions already lock the owning games row FOR UPDATE before their
-- existing capacity/position logic runs — enforce_game_capacity's own
-- `SELECT capacity INTO game_capacity FROM games WHERE id = NEW.game_id FOR
-- UPDATE`, join_waitlist_entry's own `PERFORM 1 FROM games WHERE id =
-- p_game_id FOR UPDATE`. This migration adds, at the top of each function's
-- already-locked section, a check that raises (mirroring
-- enforce_game_capacity's own `RAISE EXCEPTION ... USING ERRCODE = 'P0001'`
-- pattern, the same code the domain-level ErrGameCancelled sentinel's gRPC
-- mapping — adapter/grpcapi.toStatus — already expects for this invariant)
-- when the locked row's status = 'cancelled'. No new lock is taken and no
-- new race window is opened: both functions serialize on the identical
-- games-row FOR UPDATE lock they already held before this migration, so a
-- concurrent CancelGame and a concurrent RegisterForGame/JoinWaitlist for
-- the same Game still fully serialize against each other exactly as they
-- did before this migration — only the outcome once serialized changes (the
-- late arrival now sees the committed 'cancelled' status and is rejected,
-- instead of racing to insert against stale, pre-cancellation state).
--
-- Registration.Cancel (a player cancelling their OWN registration) is
-- deliberately unaffected: enforce_game_capacity's existing early return
-- (`IF NEW.status = 'cancelled' THEN RETURN NEW; END IF;`) already exits
-- before this migration's new check runs, so a Registration transitioning
-- TO cancelled is never blocked by its own Game also being cancelled — this
-- migration only guards a row trying to become (or stay) ACTIVE against an
-- already-cancelled Game, matching domain.Register/domain.JoinWaitlist's own
-- scope exactly (T19.1's sprint-goal "does not claim": an already-active
-- Registration on a Game cancelled after the fact is unaffected — that is
-- T16.3's cascade, already shipped).
--
-- NOT INDEPENDENTLY EXECUTED (CLAUDE.md rule 10, stated plainly rather than
-- claimed): this environment has no Docker daemon reachable, the standing
-- gap every prior sprint's committed `-tags=integration` tests have carried.
-- Manual-verification fallback, per this project's own LESSONS.md
-- methodology (T4/T6.4): apply 0001-0023 in order against a local Postgres
-- instance, then attempt a RegisterForGame/JoinWaitlist call against a
-- cancelled Game's row directly via the repository (or call
-- enforce_game_capacity/join_waitlist_entry directly with a games.id whose
-- status is 'cancelled') and confirm the trigger/function raises with
-- ERRCODE 'P0001'.

CREATE OR REPLACE FUNCTION enforce_game_capacity() RETURNS trigger AS $$
DECLARE
    game_capacity integer;
    game_status   text;
    active_count  integer;
BEGIN
    -- A row landing in (or staying in) 'cancelled' never consumes a slot,
    -- and is never blocked by its own Game's status (see this migration's
    -- header note on Registration.Cancel).
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
    -- count that follows it. T19.1 adds game_status to this same locked
    -- read rather than a second SELECT, so the cancelled-Game check below
    -- observes the identical locked snapshot the capacity check already
    -- relies on.
    SELECT capacity, status INTO game_capacity, game_status
    FROM games
    WHERE id = NEW.game_id
    FOR UPDATE;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'socialplay: game % not found', NEW.game_id
            USING ERRCODE = 'foreign_key_violation';
    END IF;

    -- T19.1 (closes #212): a Registration cannot become (or stay) active
    -- against an already-cancelled Game. Checked before the capacity count
    -- below -- the game not being in a bookable state at all is the more
    -- fundamental fact, mirroring domain.Register's own ordering.
    IF game_status = 'cancelled' THEN
        RAISE EXCEPTION 'socialplay: game % is cancelled', NEW.game_id
            USING ERRCODE = 'P0001';
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

CREATE OR REPLACE FUNCTION join_waitlist_entry(p_id uuid, p_game_id uuid, p_player_id text)
RETURNS SETOF waitlist_entries AS $$
DECLARE
    v_position    integer;
    v_entry       waitlist_entries%ROWTYPE;
    v_game_status text;
BEGIN
    -- The actual race-closing lock: every concurrent join for this
    -- game_id (and every promote_next_waiting / enforce_game_capacity call
    -- for it too, since they lock the same row) serializes here, so the
    -- count below always runs against a state that already reflects every
    -- earlier-committed join/promotion for this game, never a stale one.
    -- T19.1 reads status into this same locked row rather than a bare
    -- PERFORM, so the cancelled-Game check below observes the identical
    -- locked snapshot the position count already relies on.
    SELECT status INTO v_game_status
    FROM games
    WHERE id = p_game_id
    FOR UPDATE;

    -- T19.1 (closes #212): a waitlist entry cannot be created against an
    -- already-cancelled Game. Checked before the position count below --
    -- same ordering rationale as enforce_game_capacity's identical check
    -- above, mirroring domain.JoinWaitlist's own ordering. A game_id that
    -- matches no row leaves v_game_status NULL (`NULL = 'cancelled'` is
    -- NULL, not true), so this check does not change this function's
    -- pre-existing behaviour for an unknown game_id -- unchanged from
    -- before this migration, and out of this ticket's scope.
    IF v_game_status = 'cancelled' THEN
        RAISE EXCEPTION 'socialplay: game % is cancelled', p_game_id
            USING ERRCODE = 'P0001';
    END IF;

    -- Mirrors domain.JoinWaitlist's Position formula exactly: every
    -- non-cancelled entry counts (waiting, promoted, AND expired — only a
    -- voluntary cancel is excluded from the tally), the new entry lands one
    -- past that count.
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
