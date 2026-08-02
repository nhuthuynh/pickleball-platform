-- Payments context schema (T6.4): a single `payments` table records money
-- for any payable action (booking, registration, or no_show_fee today —
-- internal/payments/domain.PayableType is extensible), per T6.1's Payment
-- aggregate and ADR-0005's currency_code decision.
--
-- payable_type/payable_id are not a single FK: a payable action can be a
-- Booking or a Social Play Registration (polymorphic reference), mirroring
-- how bookings.reference_id is already a plain, un-FK'd reference for the
-- same "the referenced aggregate lives in another/no table yet" reason.
--
-- recorded_by_user_id/method/status/stripe_reference all stay plain text,
-- not enum/FK types — there is no Users bounded context yet (HANDOFF.md),
-- the same scope cut games.host_id/registrations.player_id already made
-- (db/migrations/0005_socialplay.sql on the T5 branches).
--
-- Prototype-only migration tooling: applied via docker-compose initdb.d on a
-- FRESH volume only (see CLAUDE.md gotchas). Adopt golang-migrate/goose
-- before production.

CREATE TABLE payments (
    id                   uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    payable_type         text NOT NULL CHECK (payable_type IN ('booking', 'registration', 'no_show_fee')),
    payable_id           uuid NOT NULL,
    amount_cents         bigint NOT NULL CHECK (amount_cents > 0),
    -- ISO 4217, per ADR-0005 (docs/adr/0005-currency-code-column.md):
    -- amount and currency are coupled from day one, even though v1 only
    -- ever populates one constant value.
    currency_code        char(3) NOT NULL,
    method               text NOT NULL CHECK (method IN ('online', 'offline')),
    status               text NOT NULL DEFAULT 'unpaid' CHECK (status IN ('unpaid', 'paid', 'refunded')),
    stripe_reference     text,
    recorded_by_user_id  text,
    created_at           timestamptz NOT NULL DEFAULT now(),
    updated_at           timestamptz NOT NULL DEFAULT now()
);

-- The "one Payment per payable action" invariant (T6.1's
-- EnsureOnePaymentPerPayable, CLAUDE.md rule 4's DB-level half): a plain
-- UNIQUE constraint, not a locking trigger — this is a *distinctness*
-- race, not a *counting* one, per the T5 LESSONS.md finding that a unique
-- index alone closes a distinctness race (contrast with Social Play's
-- registrations_capacity_guard trigger, db/migrations/
-- 0006_socialplay_capacity_guard.sql on the T5 branches, which needed a
-- row lock because it counts). internal/payments/adapter/postgres
-- translates this constraint's 23505 violation into
-- domain.ErrPaymentAlreadyRecorded rather than leaking it (CLAUDE.md
-- rule 5).
-- This single index both closes the race above and serves as the lookup
-- index for the payable-action read path — no separate plain index needed
-- on the same two columns.
CREATE UNIQUE INDEX payments_payable_unique_idx ON payments (payable_type, payable_id);
