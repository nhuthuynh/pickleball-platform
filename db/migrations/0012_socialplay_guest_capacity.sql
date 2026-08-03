-- Guest capacity wiring (T8.7, docs/process/t8-sprint-plan.md T8.7 section).
-- T8.6 added domain.Game.PaymentMethod/GuestAllowance and
-- domain.Registration.GuestCount to the pure domain
-- (internal/socialplay/domain), reweighting domain.Register's capacity check
-- to sum (1 + GuestCount) per active registration rather than a plain
-- headcount (see registration.go's Register doc comment). This migration is
-- T8.7's DB-side half of that change: it adds the same three columns to
-- games/registrations, and -- critically -- REWRITES the existing
-- registrations_capacity_guard trigger function (introduced in
-- db/migrations/0006_socialplay_capacity_guard.sql) to match, replacing its
-- COUNT(*) with the same weighted sum.
--
-- Why the rewrite is not optional (CLAUDE.md rule 4: invariants enforced in
-- Postgres AND expressed in the domain -- the two must agree): without it,
-- the DB-level guard and the domain-level check would silently disagree.
-- domain.Register already rejects a registration whose guests would push a
-- Game's occupied capacity over Capacity, but the old trigger only compared
-- registration *row count* against capacity -- it had no idea some rows
-- represent more than one occupied slot. A single registration bringing
-- enough guests to fill (or overfill) a Game on its own would be correctly
-- rejected by domain.Register in the non-concurrent case, but under a race
-- (two requests both reading a pre-insert snapshot) only the DB-level guard
-- -- which recomputes under a row lock -- can actually close it, and the old
-- COUNT(*) version would have let it through: up to `capacity` ROWS could
-- land regardless of how many guests each one brought, silently overbooking
-- the venue well past its real capacity. See
-- guest_capacity_concurrency_integration_test.go (this PR) for the
-- concurrency proof this exact scenario demands, per CLAUDE.md rule 10 and
-- the T5.4-loop-2 standard (db/migrations/0006_socialplay_capacity_guard.sql
-- doc comment) this ticket is held to.
--
-- payment_method defaults to 'either' (not 'online' or 'cash'): it is the
-- least restrictive of the three closed enum values -- "accept either"
-- imposes no payment-method policy at all, which is the closest a
-- three-value closed enum can get to "unspecified". Every Game that already
-- exists predates the concept of a Host declaring a payment-method
-- preference at all (T5-T8.6), so defaulting new column to the option that
-- doesn't retroactively restrict any of them is the only non-breaking
-- choice -- mirrors internal/socialplay/adapter/grpcapi's mapping of the
-- proto PaymentMethod's zero value (PAYMENT_METHOD_UNSPECIFIED, what an
-- old/unaware client sends) onto domain.PaymentMethodEither for the exact
-- same reason (see handler.go's fromProtoPaymentMethod).
--
-- Prototype-only migration tooling: applied via docker-compose initdb.d on a
-- FRESH volume only (see CLAUDE.md gotchas) -- this is a new file, not an
-- edit to the already-applied/reviewed 0005_socialplay.sql or
-- 0006_socialplay_capacity_guard.sql.

ALTER TABLE games
    ADD COLUMN payment_method text NOT NULL DEFAULT 'either'
        CHECK (payment_method IN ('online', 'cash', 'either')),
    ADD COLUMN guest_allowance integer NOT NULL DEFAULT 0
        CHECK (guest_allowance >= 0);

ALTER TABLE registrations
    ADD COLUMN guest_count integer NOT NULL DEFAULT 0
        CHECK (guest_count >= 0);

-- Rewrite enforce_game_capacity(): the locking/skip logic is unchanged from
-- the T5.4 loop-2 version (see 0006's own doc comment for why the FOR UPDATE
-- lock and the "skip cancelled" / "skip active->active update" branches are
-- shaped the way they are) -- only the occupancy calculation changes, from
-- COUNT(*) to a weighted SUM(1 + guest_count) over the same row set, and the
-- comparison/exception now account for NEW's own guest_count too (NEW no
-- longer always occupies exactly 1 slot).
--
-- CREATE OR REPLACE is sufficient here -- the trigger
-- (registrations_capacity_guard, unchanged, still defined in
-- 0006_socialplay_capacity_guard.sql) already points at this function by
-- name, so no DROP/CREATE TRIGGER is needed to pick up the new body.
CREATE OR REPLACE FUNCTION enforce_game_capacity() RETURNS trigger AS $$
DECLARE
    game_capacity  integer;
    active_weight  integer;
    new_weight     integer;
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
    -- weighted sum that follows it.
    SELECT capacity INTO game_capacity
    FROM games
    WHERE id = NEW.game_id
    FOR UPDATE;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'socialplay: game % not found', NEW.game_id
            USING ERRCODE = 'foreign_key_violation';
    END IF;

    -- Weighted occupancy of every OTHER active registration for this game:
    -- each row occupies (1 + its own guest_count) slots, mirroring
    -- domain.Register's countActiveRegistrations exactly (T8.6). A plain
    -- COUNT(*) undercounts any row with guest_count > 0, which is the exact
    -- bug this migration fixes -- see this file's own doc comment.
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
