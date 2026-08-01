-- name: CreateBooking :one
INSERT INTO bookings (id, court_id, source, status, starts_at, ends_at, reference_id)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, court_id, source, status, starts_at, ends_at, reference_id;

-- name: GetBookingByID :one
SELECT id, court_id, source, status, starts_at, ends_at, reference_id
FROM bookings
WHERE id = $1;

-- name: ListActiveForCourt :many
-- Active (non-cancelled) bookings on court_id overlapping [from_ts, to_ts).
-- Mirrors domain.TimeRange.Overlaps's half-open semantics.
SELECT id, court_id, source, status, starts_at, ends_at, reference_id
FROM bookings
WHERE court_id = $1
  AND status <> 'cancelled'
  AND starts_at < sqlc.arg(to_ts)
  AND sqlc.arg(from_ts) < ends_at
ORDER BY starts_at;

-- name: UpdateBookingStatus :one
UPDATE bookings
SET status = $2
WHERE id = $1
RETURNING id, court_id, source, status, starts_at, ends_at, reference_id;
