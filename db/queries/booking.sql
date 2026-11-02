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

-- name: ListBookingsForOwner :many
-- Every booking owned by owner_user_id — the player-facing "my bookings"
-- read (T67.1, part of issue #353). Served by bookings_owner_user_id_idx,
-- the index 0027_booking_owner_user_id.sql added alongside the column for
-- exactly this query; its own header names this read ("which bookings does
-- this user own ... the read a 'my bookings' view performs"), so no new index
-- is needed and this ticket adds no migration.
--
-- NOTE THE MISSING `status <> 'cancelled'`, AND DO NOT ADD IT. Every other
-- list query in this file excludes cancelled bookings because each answers an
-- availability question, where a cancelled booking is noise. This one answers
-- "what did I book?", where a cancelled booking is information: a player who
-- cannot see a cancelled booking cannot tell "I cancelled that" from "my
-- booking was lost", and the second is the bug #353 reports. A filter here
-- would reintroduce it in the fix.
--
-- ORDER BY starts_at, id: ascending start time is the contract three
-- generated clients read (see the RPC's comment in booking.proto), and id
-- breaks ties so the order is total rather than merely sorted — two bookings
-- on different courts can share a start time, and an unstable order between
-- them would be a Hyrum's-Law surface that changed with the plan.
SELECT id, court_id, source, status, starts_at, ends_at, reference_id, owner_user_id
FROM bookings
WHERE owner_user_id = $1
ORDER BY starts_at, id;

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
