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
