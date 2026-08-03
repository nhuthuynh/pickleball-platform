-- Social Play context schema (T5.4): games and registrations tables, for
-- CreateGame/RegisterForGame/CancelRegistration.
--
-- host_id/player_id are stored as plain text, not uuid FKs: there is no
-- Users bounded context yet (HANDOFF.md), so these are opaque identifiers
-- for now, the same way bookings.reference_id is a plain text reference
-- rather than a real FK.
--
-- court_ids is a uuid[] rather than a join table (mirroring pricing_rules'
-- own use of a native array column for weekdays) — Postgres can't express
-- an array-element FK directly, so referential integrity against courts is
-- enforced at the application boundary instead: internal/socialplay/adapter/
-- booking reserves each court via the real Booking context, which validates
-- courtID as it creates the underlying Booking.
--
-- Prototype-only migration tooling: applied via docker-compose initdb.d on a
-- FRESH volume only (see CLAUDE.md gotchas). Adopt golang-migrate/goose
-- before production.

CREATE TABLE games (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    host_id       text NOT NULL,
    facility_id   text NOT NULL,
    court_ids     uuid[] NOT NULL CHECK (cardinality(court_ids) > 0),
    starts_at     timestamptz NOT NULL,
    ends_at       timestamptz NOT NULL,
    capacity      integer NOT NULL CHECK (capacity > 0),
    status        text NOT NULL DEFAULT 'scheduled' CHECK (status IN ('scheduled', 'cancelled')),
    created_at    timestamptz NOT NULL DEFAULT now(),
    CHECK (ends_at > starts_at)
);

CREATE TABLE registrations (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    game_id         uuid NOT NULL REFERENCES games (id),
    player_id       text NOT NULL,
    source          text NOT NULL DEFAULT 'app' CHECK (source IN ('app', 'social')),
    status          text NOT NULL DEFAULT 'registered' CHECK (status IN ('registered', 'cancelled')),
    payment_status  text NOT NULL DEFAULT 'unpaid' CHECK (payment_status IN ('unpaid', 'paid')),
    created_at      timestamptz NOT NULL DEFAULT now()
);

-- The no-double-registration invariant (CLAUDE.md rule 4: invariants are
-- enforced in Postgres AND expressed in the domain — this is the DB-level
-- half, mirroring domain.Register's own-scope check the same way bookings'
-- EXCLUDE constraint mirrors domain.EnsureNoConflict). A partial unique
-- index (not a plain UNIQUE constraint) so a player can register again
-- after cancelling — a cancelled row must not collide with a new active
-- one. internal/socialplay/adapter/postgres translates this constraint's
-- 23505 violation into domain.ErrAlreadyRegistered rather than leaking it.
CREATE UNIQUE INDEX registrations_active_player_per_game_idx
    ON registrations (game_id, player_id)
    WHERE status <> 'cancelled';

CREATE INDEX registrations_game_idx ON registrations (game_id);
