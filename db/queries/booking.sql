-- name: CreateBooking :one
INSERT INTO bookings (id, court_id, source, status, starts_at, ends_at, reference_id, owner_user_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, court_id, source, status, starts_at, ends_at, reference_id, owner_user_id;

-- name: GetBookingByID :one
SELECT id, court_id, source, status, starts_at, ends_at, reference_id, owner_user_id
FROM bookings
WHERE id = $1;

-- name: ListActiveForCourt :many
-- Active (non-cancelled) bookings on court_id overlapping [from_ts, to_ts).
-- Mirrors domain.TimeRange.Overlaps's half-open semantics.
SELECT id, court_id, source, status, starts_at, ends_at, reference_id, owner_user_id
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
RETURNING id, court_id, source, status, starts_at, ends_at, reference_id, owner_user_id;

-- name: ListActiveForReference :many
-- Active (non-cancelled) bookings made against reference_id — a Game's,
-- Competition's, or RecurringHireTemplate's id. Serves the cancellation
-- cascade (#124): when a Game is cancelled, the courts its `game`-source
-- Bookings hold must be released, and this is how those Bookings are found.
--
-- An empty reference_id deliberately matches nothing rather than matching
-- every unreferenced booking: reference_id is nullable and empty for plain
-- individual bookings, so `WHERE reference_id = ''` without this guard would
-- be a cascade that cancels the whole table. The app layer also refuses an
-- empty reference before reaching here; both halves are deliberate.
SELECT id, court_id, source, status, starts_at, ends_at, reference_id, owner_user_id
FROM bookings
WHERE reference_id = sqlc.arg(reference_id)
  AND sqlc.arg(reference_id) <> ''
  AND status <> 'cancelled'
ORDER BY starts_at;
