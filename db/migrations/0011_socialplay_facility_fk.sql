-- T8.3: reconcile games.facility_id (an opaque free-text column with no
-- relationship to anything, db/migrations/0005_socialplay.sql) with the
-- real Facilities context (facilities.id uuid, db/migrations/0010_facilities
-- .sql) — Facilities didn't exist when Social Play was originally built.
--
-- Mirrors T7.3's courts.facility_id precedent exactly (db/migrations/
-- 0010_facilities.sql's "ALTER TABLE courts ADD COLUMN facility_id uuid
-- REFERENCES facilities (id)" comment): add a NEW nullable FK column
-- alongside the existing one rather than attempting an in-place type change
-- on games.facility_id text. An in-place change isn't attempted because (a)
-- the existing column is free text, not guaranteed to parse as uuid, and
-- (b) changing an existing column's type/semantics out from under any
-- current reader is exactly the kind of breaking change the nullable-FK
-- precedent exists to avoid.
--
-- games.facility_id text is left in place, completely unreferenced by any
-- new code path from this ticket onward (internal/socialplay/domain,
-- internal/socialplay/app, and internal/socialplay/adapter/postgres's
-- CreateGame/GetGameByID queries all read/write venue_facility_id instead —
-- see db/queries/socialplay.sql). It is DEPRECATED and slated for removal
-- in a future migration once no reader still depends on it.
--
-- Nullable, not NOT NULL: existing/seeded Games (if any) predate any real
-- Facility relationship and must keep working unmodified — they get NULL
-- here, not a backfilled guess (inventing backfill logic from the opaque
-- facility_id text value to a real facilities.id uuid is explicitly out of
-- scope; there is no reliable mapping between the two).
--
-- Prototype-only migration tooling: applied via docker-compose initdb.d on a
-- FRESH volume only (see CLAUDE.md gotchas). Adopt golang-migrate/goose
-- before production.

ALTER TABLE games ADD COLUMN venue_facility_id uuid REFERENCES facilities (id);

CREATE INDEX games_venue_facility_idx ON games (venue_facility_id);
