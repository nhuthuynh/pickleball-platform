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
