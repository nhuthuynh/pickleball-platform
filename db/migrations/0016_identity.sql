-- Identity/Users context schema (T10.2, on top of T10.1's internal/identity/
-- domain — see docs/process/t10-sprint-plan.md T10.2). A User is the
-- aggregate other contexts' currently-opaque actor_user_id/host_id/
-- player_id strings will eventually reference — see ADR-0012 for what is
-- and is not built this sprint (no PlayerRating, no derived Level, no
-- Gender, no matching-mode flag).
--
-- id is NOT server-generated the way every other aggregate's id in this
-- schema is (facilities.id, courts.id, payments.id, ... all default
-- gen_random_uuid()) — deliberately no DEFAULT here. Identity/Users is the
-- first context that represents the concept of a caller's own claimed
-- identity, and there is no separate context to mint one on the caller's
-- behalf: the id is the caller-claimed actor_user_id itself, supplied by
-- app.Service.CreateUser (see that method's doc comment). It is still typed
-- `uuid`, not `text`, so it stays a valid foreign-key target for every other
-- context's eventual real reference (mirrors facilities.owner_id's uuid
-- typing ahead of a real FK target existing yet).
--
-- roles is a text[] rather than a join table: Role is a small, closed enum
-- (domain.Role, six values). CLAUDE.md rule 4 requires the closed-enum
-- invariant to be enforced in BOTH the domain (NewUser's per-element
-- IsValid loop) AND Postgres — mirrored here with the identical
-- cardinality-plus-containment CHECK shape pricing_rules.weekdays already
-- uses for its own smallint[] closed set (db/migrations/
-- 0003_pricing_rules.sql), the array-typed equivalent of the scalar
-- `text ... CHECK (col IN (...))` pattern every other closed-enum column in
-- this schema uses (bookings.status/source, payments.payable_type/method/
-- status, competitions.status/format/payment_method). An earlier version of
-- this comment incorrectly claimed payments.status/payments.method were
-- unchecked plain text columns — they are not, see db/migrations/
-- 0005_payments.sql. self_reported_starting_level gets the scalar-range
-- equivalent via a BETWEEN CHECK, mirroring domain.
-- SelfReportedStartingLevel's bounded 1..5 range (internal/identity/domain/
-- level.go).
--
-- Prototype-only migration tooling: applied via docker-compose initdb.d on a
-- FRESH volume only (see CLAUDE.md gotchas). Adopt golang-migrate/goose
-- before production.

CREATE TABLE identity_users (
    id                            uuid PRIMARY KEY,
    display_name                  text NOT NULL,
    roles                         text[] NOT NULL CHECK (
        cardinality(roles) > 0
        AND roles <@ ARRAY['player', 'host_organiser', 'game_admin', 'facility_owner', 'club', 'platform_admin']::text[]
    ),
    self_reported_starting_level  smallint NOT NULL CHECK (self_reported_starting_level BETWEEN 1 AND 5),
    created_at                    timestamptz NOT NULL DEFAULT now(),
    updated_at                    timestamptz NOT NULL DEFAULT now()
);
