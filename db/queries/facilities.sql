-- name: CreateFacility :one
INSERT INTO facilities (id, owner_id, name, description, address, photo_urls, camera_consent_attested)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, owner_id, name, description, address, photo_urls, camera_consent_attested;

-- name: GetFacilityByID :one
SELECT id, owner_id, name, description, address, photo_urls, camera_consent_attested
FROM facilities
WHERE id = $1;

-- name: ListFacilities :many
-- Simple ILIKE substring match on name (no real geo-search, per HANDOFF.md
-- T7.3) — an empty name_filter returns every facility.
SELECT id, owner_id, name, description, address, photo_urls, camera_consent_attested
FROM facilities
WHERE sqlc.arg(name_filter)::text = '' OR name ILIKE '%' || sqlc.arg(name_filter) || '%'
ORDER BY name;

-- name: AddCourt :one
-- Inserts into the *existing* courts table (0001_init.sql) with
-- facility_id set — not a second courts table (HANDOFF.md T7.3 AC 5), so
-- the id this returns is immediately a valid bookings.court_id for
-- Booking's CreateBooking/ListCourtBookings, unmodified.
INSERT INTO courts (id, name, facility_id)
VALUES ($1, $2, $3)
RETURNING id, name, facility_id;

-- name: GetCourtByID :one
-- T11.2: the reverse of ListCourtsForFacility — given a Court, which
-- Facility owns it. Booking's port.FacilityLookup uses this (through
-- facilities' app.Service.GetCourt) to resolve a facility-scoped
-- DiscountRule from the CourtID GetQuote already has, instead of Booking
-- reading the courts table itself behind the context boundary.
-- facility_id is nullable (0010_facilities.sql), so a pre-Facilities seeded
-- court comes back with a NULL facility_id — found, but belonging to no
-- Facility.
SELECT id, name, facility_id
FROM courts
WHERE id = $1;

-- name: GetCourtsByIDs :many
-- The batch form of GetCourtByID (T67.1, part of issue #353): resolve many
-- Courts to their owning Facilities in one round trip.
--
-- It exists because Booking's new "my bookings" read needs the facility of
-- every court in a player's list so a client can show each court's NAME
-- rather than its uuid (issue #342, closed). Resolving them one at a time
-- would be an N+1 whose N is set by how much the player has booked, and
-- Booking must not read the courts table itself — the court->facility fact
-- belongs to this context, so the batch read belongs here too, behind
-- port.FacilityLookup.
--
-- Keyed by an id array, following ListRecurringHireTemplatesForCourts in
-- db/queries/recurring_hire.sql. An id matching no Court simply contributes
-- no row: callers get back the Courts that exist and learn nothing about the
-- ones that do not, which is what lets a Booking on a deleted or unknown
-- court still be listed. facility_id is nullable, so a pre-Facilities seeded
-- court (0002_seed.sql) comes back with NULL — found, belonging to no
-- Facility. No ORDER BY: callers key the result by id, and
-- ListBookingsForOwner sets the order the user sees.
SELECT id, name, facility_id
FROM courts
WHERE id = ANY(@court_ids::uuid[]);

-- name: ListCourtsForFacility :many
-- The read path AddCourt (T7.3) never had (T8.2): every Court belonging to
-- facilityID, in creation order, from the *existing* courts table (same one
-- AddCourt inserts into, and Booking's CreateBooking/ListCourtBookings read
-- court_id from) — no second courts table, no Booking schema change.
SELECT id, name, facility_id
FROM courts
WHERE facility_id = $1
ORDER BY created_at;

-- name: AddCameraLink :one
INSERT INTO facility_camera_links (facility_id, court_id, url)
VALUES ($1, $2, $3)
RETURNING id, facility_id, court_id, url;

-- name: ListCameraLinksForFacility :many
SELECT id, facility_id, court_id, url
FROM facility_camera_links
WHERE facility_id = $1
ORDER BY created_at;

-- name: AttestCameraConsent :one
-- T8.4: the write path that sets camera_consent_attested = true. RETURNING
-- id (rather than :exec) so the adapter can distinguish "facility doesn't
-- exist" (pgx.ErrNoRows, translated to domain.ErrFacilityNotFound) from a
-- successful no-op update on an already-true row — idempotent per
-- domain.Facility.AttestCameraConsent's doc comment.
UPDATE facilities
SET camera_consent_attested = true, updated_at = now()
WHERE id = $1
RETURNING id;
