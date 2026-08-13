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
-- (domain.Role, six values) checked in the domain layer (NewUser's
-- per-element IsValid loop), not by a CHECK constraint here — mirrors this
-- schema's existing practice of enforcing closed-enum values in the domain,
-- not duplicated as a Postgres CHECK (e.g. payments.status/payments.method
-- are plain text columns too).
--
-- Prototype-only migration tooling: applied via docker-compose initdb.d on a
-- FRESH volume only (see CLAUDE.md gotchas). Adopt golang-migrate/goose
-- before production.

CREATE TABLE identity_users (
    id                            uuid PRIMARY KEY,
    display_name                  text NOT NULL,
    roles                         text[] NOT NULL,
    self_reported_starting_level  smallint NOT NULL,
    created_at                    timestamptz NOT NULL DEFAULT now(),
    updated_at                    timestamptz NOT NULL DEFAULT now()
);
