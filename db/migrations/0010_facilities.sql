-- Facilities context schema (T7.3, on top of T7.2's internal/facilities/
-- domain — see docs/process/t7-sprint-plan.md T7.3). A Facility is the
-- aggregate a Facility Owner creates: a physical venue containing one or
-- more Courts, with photos and optional camera links gated by
-- camera_consent_attested (round-10 design review §2b, encoded as a
-- domain invariant in Facility.AddCameraLink).
--
-- owner_id is a uuid column even though there is no Users bounded context
-- yet (the same gap socialplay's host_id/player_id note calls out) —
-- unlike those, this ticket's spec (HANDOFF.md T7.3 AC 1) calls for owner_id
-- as uuid explicitly, so it's typed that way now; there is deliberately no
-- FK to a users table since one doesn't exist yet.
--
-- Prototype-only migration tooling: applied via docker-compose initdb.d on a
-- FRESH volume only (see CLAUDE.md gotchas). Adopt golang-migrate/goose
-- before production.

CREATE TABLE facilities (
    id                        uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id                  uuid NOT NULL,
    name                      text NOT NULL,
    description               text NOT NULL DEFAULT '',
    address                   text NOT NULL,
    photo_urls                text[] NOT NULL DEFAULT '{}',
    camera_consent_attested   boolean NOT NULL DEFAULT false,
    created_at                timestamptz NOT NULL DEFAULT now(),
    updated_at                timestamptz NOT NULL DEFAULT now()
);

-- courts already exists (0001_init.sql) as Booking's minimal stub for the
-- not-yet-built Facilities context. This ticket is that context arriving,
-- so courts gains a nullable facility_id FK rather than a second table —
-- nullable because the existing seeded courts (0002_seed.sql) predate any
-- Facility and must keep working, unmodified, against the same courts.id
-- values Booking already references (HANDOFF.md T7.3 AC 5/7).
ALTER TABLE courts ADD COLUMN facility_id uuid REFERENCES facilities (id);

-- facility_camera_links is a table, not a text[] array column on
-- facilities, specifically so a link can be scoped to one Court
-- (court_id set) or to the whole Facility (court_id null) — matching
-- internal/facilities/domain.CameraLink's CourtID field (T7.2). The
-- currently-merged Facility.AddCameraLink domain method only ever appends
-- facility-wide links (CourtID empty) — see its doc comment — so
-- court_id is populated by this migration's schema but not yet
-- exercised end-to-end by any RPC in this ticket; that's for a future
-- ticket to wire once the domain grows a per-court variant. Logged as a
-- forward-looking note here rather than silently left unused.
CREATE TABLE facility_camera_links (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    facility_id   uuid NOT NULL REFERENCES facilities (id),
    court_id      uuid REFERENCES courts (id),
    url           text NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX facility_camera_links_facility_idx ON facility_camera_links (facility_id);
CREATE INDEX courts_facility_idx ON courts (facility_id);
