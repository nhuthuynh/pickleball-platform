-- Booking context schema (see docs/pickleball-platform-spec.md §6, §8 and
-- CLAUDE.md "Locked decisions"). D3b: bookings is polymorphic over `source`
-- so the EXCLUDE constraint below is the single no-double-booking invariant
-- covering all four reservation kinds.
--
-- Prototype-only migration tooling: applied via docker-compose initdb.d on a
-- FRESH volume only (see CLAUDE.md gotchas). Adopt golang-migrate/goose
-- before production.

CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS btree_gist;

-- Minimal stub for the not-yet-built Facilities context (HANDOFF.md), just
-- enough for `bookings.court_id` to have a real referential target and for
-- seed data / the smoke test in README.md to work.
CREATE TABLE courts (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name        text NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE bookings (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    court_id        uuid NOT NULL REFERENCES courts (id),
    source          text NOT NULL CHECK (source IN ('recurring_hire', 'individual', 'game', 'competition')),
    status          text NOT NULL DEFAULT 'confirmed' CHECK (status IN ('confirmed', 'cancelled')),
    -- Kept as plain timestamptz columns (not the generated range directly)
    -- so sqlc sees concrete types instead of typing `during` as interface{}
    -- — see the CLAUDE.md gotcha. `during` is derived and GiST-indexed for
    -- the EXCLUDE constraint; queries select starts_at/ends_at, never during.
    starts_at       timestamptz NOT NULL,
    ends_at         timestamptz NOT NULL,
    during          tstzrange GENERATED ALWAYS AS (tstzrange(starts_at, ends_at, '[)')) STORED,
    reference_id    text,
    created_at      timestamptz NOT NULL DEFAULT now(),
    CHECK (ends_at > starts_at),

    -- The no-double-booking invariant (spec §6): no two non-cancelled
    -- bookings on the same court may have overlapping `during` ranges,
    -- regardless of source. This is authoritative; domain.EnsureNoConflict
    -- is a same-rule pre-check, not a substitute (see HANDOFF.md T4).
    EXCLUDE USING gist (
        court_id WITH =,
        during WITH &&
    ) WHERE (status <> 'cancelled')
);

CREATE INDEX bookings_court_during_idx ON bookings USING gist (court_id, during);
