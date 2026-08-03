-- name: CreateGame :one
INSERT INTO games (id, host_id, facility_id, court_ids, starts_at, ends_at, capacity, status)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, host_id, facility_id, court_ids, starts_at, ends_at, capacity, status;

-- name: GetGameByID :one
SELECT id, host_id, facility_id, court_ids, starts_at, ends_at, capacity, status
FROM games
WHERE id = $1;

-- name: CreateRegistration :one
INSERT INTO registrations (id, game_id, player_id, source, status, payment_status)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, game_id, player_id, source, status, payment_status;

-- name: GetRegistrationByID :one
SELECT id, game_id, player_id, source, status, payment_status
FROM registrations
WHERE id = $1;

-- name: ListActiveRegistrationsForGame :many
-- Non-cancelled registrations for game_id — the capacity-safe read path
-- app.Service.RegisterForGame uses to re-derive the active count/players
-- before calling domain.Register, mirroring ListActiveForCourt's role in
-- Booking's CreateBooking.
SELECT id, game_id, player_id, source, status, payment_status
FROM registrations
WHERE game_id = $1
  AND status <> 'cancelled'
ORDER BY created_at;

-- name: UpdateRegistrationStatus :one
UPDATE registrations
SET status = $2
WHERE id = $1
RETURNING id, game_id, player_id, source, status, payment_status;

-- name: UpdateRegistrationPaymentStatus :one
-- Dedicated single-column update for PaymentStatus (T6.5), mirroring
-- UpdateRegistrationStatus's shape: a separate query per updatable field
-- rather than one generic "update everything" statement, so this write
-- path can never accidentally touch status (or vice versa). The sole
-- caller is app.Service.MarkRegistrationPaymentStatus, itself only called
-- through port.RegistrationPaymentUpdater by
-- internal/payments/adapter/socialplay.
UPDATE registrations
SET payment_status = $2
WHERE id = $1
RETURNING id, game_id, player_id, source, status, payment_status;

-- name: JoinWaitlistEntry :one
-- The DB-level race-closing operation for queue-position assignment
-- (db/migrations/0009_socialplay_waitlist_join_position.sql, T6.6-loop-2) —
-- see that migration's doc comment for the full race analysis. Replaces a
-- plain INSERT (which trusted a caller-computed Position) with a call into
-- join_waitlist_entry, which locks the owning games row FOR UPDATE, derives
-- Position from fresh state, and inserts in one atomic step.
SELECT id, game_id, player_id, position, status, promoted_at
FROM join_waitlist_entry($1, $2, $3);

-- name: GetWaitlistEntryByID :one
SELECT id, game_id, player_id, position, status, promoted_at
FROM waitlist_entries
WHERE id = $1;

-- name: ListWaitlistEntriesForGame :many
-- Every entry for game_id, any status, oldest (by position) first -- the
-- read path app.Service.JoinWaitlist and RegisterForGame use to derive the
-- next Position, the duplicate-join check, and the
-- reserved-by-promotion check (domain.SlotReservedByPromotion).
SELECT id, game_id, player_id, position, status, promoted_at
FROM waitlist_entries
WHERE game_id = $1
ORDER BY position;

-- name: PromoteNextWaiting :one
-- The DB-level race-closing operation (db/migrations/
-- 0008_socialplay_waitlist_promotion.sql) -- see that migration's doc
-- comment for the full race analysis. Returns zero rows (sql.ErrNoRows via
-- :one) when the game has no waiting entry to promote; the adapter
-- translates that into domain.ErrNoWaitingEntries.
SELECT id, game_id, player_id, position, status, promoted_at
FROM promote_next_waiting($1, $2);

-- name: ExpireWaitlistPromotion :one
-- Compare-and-swap: only actually transitions a row that is still
-- 'promoted' at the moment this runs, so a concurrent confirm (the
-- promoted player registering) or a second expiry sweep can't double-act on
-- the same entry. Zero rows back (sql.ErrNoRows via :one) means either the
-- id doesn't exist or it's no longer promoted -- the adapter disambiguates
-- via a follow-up existence check (see translateWaitlistErr's caller).
UPDATE waitlist_entries
SET status = 'expired'
WHERE id = $1 AND status = 'promoted'
RETURNING id, game_id, player_id, position, status, promoted_at;
