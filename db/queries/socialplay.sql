-- name: CreateGame :one
-- venue_facility_id (T8.3) is nullable — sqlc infers CreateGameParams.
-- VenueFacilityID as pgtype.UUID, which the adapter must pass as an
-- explicitly-invalid (Valid: false) zero value for an empty domain string,
-- not attempt to parse "" as a uuid (see nullableUUID in repository.go).
-- payment_method and guest_allowance (T8.7, db/migrations/
-- 0012_socialplay_guest_capacity.sql) are always supplied explicitly by the
-- adapter (never relying on the column DEFAULT), mirroring how every other
-- column here is caller-supplied — the DEFAULT only exists to backfill rows
-- that predate the column. The same holds for entry_fee_cents/
-- entry_fee_currency (T9.2, db/migrations/0013_socialplay_entry_fee.sql):
-- the adapter always writes the Host's real EntryFee, including an explicit
-- 0 for a free Game -- the column DEFAULT of 0/'USD' is only ever the
-- backfill value for rows that predate the columns.
INSERT INTO games (id, host_id, facility_id, venue_facility_id, court_ids, starts_at, ends_at, capacity, status, payment_method, guest_allowance, entry_fee_cents, entry_fee_currency)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
RETURNING id, host_id, facility_id, venue_facility_id, court_ids, starts_at, ends_at, capacity, status, payment_method, guest_allowance, entry_fee_cents, entry_fee_currency;

-- name: GetGameByID :one
SELECT id, host_id, facility_id, venue_facility_id, court_ids, starts_at, ends_at, capacity, status, payment_method, guest_allowance, entry_fee_cents, entry_fee_currency
FROM games
WHERE id = $1;

-- name: ListGames :many
-- The Discover & Join Games browse/filter read path (T8.9). Only
-- 'scheduled' Games are ever returned — a cancelled Game isn't joinable, so
-- it has no place in a player-facing browse list (mirrors
-- ListActiveRegistrationsForGame's "status <> 'cancelled'" filtering
-- reasoning, just against Game.Status instead of Registration.Status).
--
-- venue_facility_id/starts_after/starts_before are all optional filters:
-- sqlc.narg generates nullable params (pgtype.UUID / pgtype.Timestamptz
-- with Valid: false for "unset"), mirroring ListFacilities' name_filter
-- empty-string convention but for columns where an empty string isn't a
-- meaningful sentinel.
--
-- spots_left is computed here (capacity minus the *weighted* sum of
-- (1 + guest_count) across this Game's active, non-cancelled
-- registrations) rather than requiring the caller to separately fetch and
-- sum registrations per Game (which would be an N+1 query per Game in the
-- list) — the exact same weighting domain.Register's own capacity check
-- uses (db/queries's ListActiveRegistrationsForGame is that check's own
-- read path; this aggregate mirrors its weighting rule, not a second,
-- independently-maintained one). GREATEST(...,0) is a defensive floor only
-- — the capacity guard trigger (0006/0012) should make a negative value
-- unreachable in practice, but a list/read path shouldn't ever surface a
-- negative "spots left" to a Player even if that invariant were ever
-- violated some other way.
SELECT g.id, g.host_id, g.facility_id, g.venue_facility_id, g.court_ids, g.starts_at, g.ends_at, g.capacity, g.status, g.payment_method, g.guest_allowance, g.entry_fee_cents, g.entry_fee_currency,
       GREATEST(g.capacity - COALESCE(SUM(CASE WHEN r.status <> 'cancelled' THEN 1 + r.guest_count ELSE 0 END), 0), 0)::int AS spots_left
FROM games g
LEFT JOIN registrations r ON r.game_id = g.id
WHERE g.status = 'scheduled'
  AND (sqlc.narg(venue_facility_id)::uuid IS NULL OR g.venue_facility_id = sqlc.narg(venue_facility_id))
  AND (sqlc.narg(starts_after)::timestamptz IS NULL OR g.starts_at >= sqlc.narg(starts_after))
  AND (sqlc.narg(starts_before)::timestamptz IS NULL OR g.starts_at <= sqlc.narg(starts_before))
GROUP BY g.id
ORDER BY g.starts_at;

-- name: UpdateGameStatus :one
-- Dedicated single-column update (T12.4's app.Service.CancelGame),
-- following the same one-query-per-updatable-field convention as
-- UpdateRegistrationStatus, UpdateBookingStatus and
-- UpdateCompetitionStatus: scoped to exactly the status column, so this
-- write path cannot accidentally clobber a Game's capacity, court_ids,
-- payment_method, guest_allowance or entry fee.
--
-- The RETURNING list is the same 13 columns CreateGame/GetGameByID select,
-- so gameFromFields converts this row exactly as it converts theirs (see
-- its doc comment: sqlc mints a distinct ...Row struct per query, so the
-- shared conversion takes the columns, not a struct).
UPDATE games
SET status = $2
WHERE id = $1
RETURNING id, host_id, facility_id, venue_facility_id, court_ids, starts_at, ends_at, capacity, status, payment_method, guest_allowance, entry_fee_cents, entry_fee_currency;

-- name: CreateRegistration :one
INSERT INTO registrations (id, game_id, player_id, source, status, payment_status, guest_count)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, game_id, player_id, source, status, payment_status, guest_count;

-- name: GetRegistrationByID :one
SELECT id, game_id, player_id, source, status, payment_status, guest_count
FROM registrations
WHERE id = $1;

-- name: ListActiveRegistrationsForGame :many
-- Non-cancelled registrations for game_id — the capacity-safe read path
-- app.Service.RegisterForGame uses to re-derive the active count/players
-- before calling domain.Register, mirroring ListActiveForCourt's role in
-- Booking's CreateBooking.
SELECT id, game_id, player_id, source, status, payment_status, guest_count
FROM registrations
WHERE game_id = $1
  AND status <> 'cancelled'
ORDER BY created_at;

-- name: UpdateRegistrationStatus :one
UPDATE registrations
SET status = $2
WHERE id = $1
RETURNING id, game_id, player_id, source, status, payment_status, guest_count;

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
RETURNING id, game_id, player_id, source, status, payment_status, guest_count;

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

-- name: CreateMatch :one
-- T10.4. score is jsonb (db/migrations/0015_socialplay_matches.sql) — the
-- adapter marshals domain.Match.Score (map[string]int) to []byte before
-- calling this, and unmarshals it back on every read (CreateMatch's own
-- RETURNING included), since sqlc's pgx/v5 driver maps a jsonb column to
-- []byte, not a Go map.
INSERT INTO matches (id, game_id, players, score, recorded_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, game_id, players, score, recorded_at;

-- name: ListMatchesForGame :many
-- Every Match recorded against game_id, oldest first — the read path
-- app.Service.ListMatchesForGame uses (T10.4). Existence of game_id itself
-- is checked by the app layer via GameRepository.GetByID before this runs
-- (see that method's doc comment for why this RPC's error-handling
-- requirements differ from ListActiveRegistrationsForGame's identical-
-- looking read).
SELECT id, game_id, players, score, recorded_at
FROM matches
WHERE game_id = $1
ORDER BY recorded_at;

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
