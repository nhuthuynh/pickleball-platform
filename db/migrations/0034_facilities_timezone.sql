-- T68.1 — a price band resolves in the facility's own timezone
-- (ADR-0004, closes #356).
--
-- WHY: internal/booking/domain.ClockTime has documented itself as "minutes
-- since midnight, local to the court's facility" since T0, while
-- ClockTimeOf is t.Hour()*60 + t.Minute() and the gRPC handler feeds it
-- req.GetStartsAt().AsTime() — which protobuf normalises to UTC. No column
-- anywhere in db/migrations held a timezone, so an 18:00 Ho Chi Minh City
-- slot resolved clock 660 and was quoted the weekday band instead of peak.
-- ADR-0004 (Accepted, pre-T5, never implemented until now) is explicit that
-- the column may live on facilities, with courts inheriting it, when courts
-- within one facility are never in different zones — the common case, and the
-- choice made here. A court-level override would be a superseding ADR, not a
-- judgement call inside this migration.
--
-- WHY ON facilities AND NOT ON courts OR pricing_rules: one fact, one place
-- (CLAUDE.md rule 7). Denormalising the zone onto pricing_rules would put a
-- copy of it on every band row, where the two can disagree and where Booking
-- would own a fact that belongs to Facilities.
--
-- NO BACKFILL, and why that is honest rather than lazy:
--   * There are no deployed facilities to backfill. This project is
--     pre-launch — ADR-0004's own Cons section says so ("needs a migration
--     if courts already exist in a deployed environment (none do yet)") —
--     and docker compose applies these files via initdb.d only on a FRESH
--     volume (CLAUDE.md gotchas), so every environment that has ever run
--     them starts from 0001 and reaches this file with whatever rows the
--     seeds created.
--   * The column is therefore NOT NULL DEFAULT '', and '' means "no zone
--     recorded". It is not DEFAULT 'UTC': a default zone would make the
--     defect above the *documented behaviour* of an unconfigured facility —
--     a wrong price that looks exactly like a right one — where '' makes
--     Booking refuse the quote with domain.ErrFacilityTimezoneUnset. An
--     explicit error is the decision this ticket was given.
--   * Nothing writes the column yet (CreateFacility's INSERT does not list
--     it), so a Facility created through the API today is quotable only
--     after its zone is set. Making it a required field on facility
--     creation — proto, handler, domain validation — is ADR-0004's
--     remaining half and a separate ticket; it is not needed to fix #356,
--     because the quote path is reachable only for courts with
--     pricing_rules rows, and those are seeded here (see below).
--
-- Prototype-only migration tooling, as every file here: adopt
-- golang-migrate/goose before production.

ALTER TABLE facilities
    ADD COLUMN timezone text NOT NULL DEFAULT '';

-- A shape check, not a validity check — the authoritative "is this a real
-- zone?" test is time.LoadLocation against the IANA database, which lives in
-- the Go process (internal/platform/tz) and cannot be expressed as a CHECK.
-- This is the DB half of rule 4 for what it CAN hold: a stored value is
-- either the empty "unset" marker or something IANA-shaped — one or more
-- slash-separated components of letters, digits, '_', '+', '-' — so a
-- whitespace string, a quoted name or a sentence cannot be stored at all.
-- pg_timezone_names is deliberately not consulted: it is Postgres's own
-- bundled copy of the database, so a CHECK against it would make this schema
-- refuse rows depending on the server's tzdata version rather than on the
-- data, and a CHECK may not query a table anyway.
ALTER TABLE facilities
    ADD CONSTRAINT facilities_timezone_shape
    CHECK (timezone = '' OR timezone ~ '^[A-Za-z][A-Za-z0-9_+-]*(/[A-Za-z0-9_+-]+)*$');

-- The demo Facility, and the two seeded courts attached to it.
--
-- WHY THIS IS PART OF THE SAME MIGRATION: 0002_seed.sql inserts Court 1 and
-- Court 2 as (id, name) only, so courts.facility_id is NULL for both, and
-- 0004_seed_pricing.sql gives Court 1 the only pricing_rules rows in the
-- schema (verified: `grep -rn "INSERT INTO pricing_rules" db/` returns that
-- one statement). The quote path is therefore reachable for exactly one court
-- in a fresh local stack — the one the README's curl walkthrough uses — and
-- a court with no Facility now has no zone, which is an explicit error by
-- design. Without this seed, shipping the fix would turn the single working
-- demo quote into a 400 for everyone running `make up`.
--
-- owner_id is a fixed uuid rather than gen_random_uuid() so the row is
-- reproducible across fresh volumes; there is no identity_users row for it
-- (facilities.owner_id has deliberately had no FK since 0010), so it names
-- nobody who can log in — it exists to make the demo facility's ownership
-- fields non-null, not to grant anything.
--
-- The zone is Asia/Ho_Chi_Minh on purpose: it is +07:00 with no DST, so a
-- local evening slot lands on the previous UTC day's clock, which is the
-- defect #356 reported and the one a human poking the demo stack is most
-- likely to reproduce by hand. A UTC demo facility would have hidden the bug
-- again.
INSERT INTO facilities (id, owner_id, name, description, address, timezone) VALUES
    ('33333333-3333-3333-3333-333333333333',
     '44444444-4444-4444-4444-444444444444',
     'Demo Facility',
     'Seed facility for the local smoke-test walkthrough (README.md).',
     '1 Demo Street',
     'Asia/Ho_Chi_Minh');

UPDATE courts
SET facility_id = '33333333-3333-3333-3333-333333333333'
WHERE id IN (
    '11111111-1111-1111-1111-111111111111',
    '22222222-2222-2222-2222-222222222222'
);
