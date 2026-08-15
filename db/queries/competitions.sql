-- name: CreateCompetition :one
-- venue_facility_id (db/migrations/0014_competitions.sql) is nullable — sqlc
-- infers CreateCompetitionParams.VenueFacilityID as pgtype.UUID, which the
-- adapter must pass as an explicitly-invalid (Valid: false) zero value for
-- an empty domain string, never attempt to parse "" as a uuid (see
-- nullableUUID in repository.go). Mirrors CreateGame's identical handling.
--
-- Every column is supplied explicitly by the adapter, never relying on a
-- column DEFAULT — including entry_fee_cents/entry_fee_currency, where the
-- adapter always writes the Host's real EntryFee (an explicit 0 for a free
-- Competition, which is a real value and not "unset"; see domain.Money).
INSERT INTO competitions (id, host_id, name, venue_facility_id, capacity, guest_allowance, payment_method, format, status, entry_fee_cents, entry_fee_currency, share_token)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING id, host_id, name, venue_facility_id, capacity, guest_allowance, payment_method, format, status, entry_fee_cents, entry_fee_currency, share_token;

-- name: CreateCompetitionSession :one
-- One row per domain.Session, written inside the same transaction as its
-- owning Competition (see Repository.Create) so a Competition can never be
-- persisted with a partial session list. position preserves the Host's
-- supplied ordering — see the migration's doc comment for why ordering by
-- starts_at alone would not round-trip deterministically.
--
-- id is deliberately NOT a parameter: it falls to the column's
-- gen_random_uuid() DEFAULT. A domain.Session is a VALUE TYPE inside the
-- Competition aggregate with no identity of its own (see the migration and
-- domain.Session's doc comments), so the primary key here is a storage key
-- that nothing in the domain, the app layer, or the wire contract ever
-- refers to. Threading it through port.IDGenerator would manufacture the
-- appearance of a domain identity this type is explicitly documented not to
-- have.
INSERT INTO competition_sessions (competition_id, position, starts_at, ends_at, court_ids)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, competition_id, position, starts_at, ends_at, court_ids;

-- name: GetCompetitionByID :one
SELECT id, host_id, name, venue_facility_id, capacity, guest_allowance, payment_method, format, status, entry_fee_cents, entry_fee_currency, share_token
FROM competitions
WHERE id = $1;

-- name: GetCompetitionByShareToken :one
-- The token-addressed read behind a Competition's shareable registration
-- link (T9.5). Selects the identical column list GetCompetitionByID selects,
-- so the two read paths cannot drift into disclosing different things.
--
-- NOTE what this query deliberately does NOT do:
--
--   * No status filter. A CANCELLED Competition's token still resolves —
--     unlike ListCompetitions, which hides cancelled Competitions from the
--     browse list. A link already posted to a social channel outlives the
--     Competition's scheduled state, and a Player who follows it must get an
--     honest "this competition was cancelled" rather than a not-found that
--     reads as a broken link. Entering it is separately rejected
--     (domain.Enter -> ErrCompetitionCancelled).
--   * No shape validation of the token, here or in the adapter above it. Any
--     string is a legal parameter; a token that matches nothing returns no
--     rows, which the adapter turns into the same domain.ErrCompetitionNotFound
--     an unknown ID produces. That sameness is the point: this read is
--     unauthenticated and keyed by a secret, so distinguishing "malformed
--     token" from "unknown token" would tell an enumerator which guesses had
--     the right shape.
--
-- share_token is `text NOT NULL UNIQUE` (db/migrations/0014_competitions.sql),
-- so the unique constraint's own index already serves this lookup and at most
-- one row can ever match — no extra index and no LIMIT are needed.
SELECT id, host_id, name, venue_facility_id, capacity, guest_allowance, payment_method, format, status, entry_fee_cents, entry_fee_currency, share_token
FROM competitions
WHERE share_token = $1;

-- name: ListSessionsForCompetition :many
-- A single Competition's sessions, in the Host's original order.
SELECT id, competition_id, position, starts_at, ends_at, court_ids
FROM competition_sessions
WHERE competition_id = $1
ORDER BY position;

-- name: ListSessionsForCompetitions :many
-- The batched form used by ListCompetitions: every session belonging to any
-- of the listed Competitions, fetched in ONE round trip rather than one per
-- Competition. This is what keeps the browse list from being an N+1 query —
-- the adapter groups the returned rows by competition_id in memory (see
-- Repository.ListCompetitions).
SELECT id, competition_id, position, starts_at, ends_at, court_ids
FROM competition_sessions
WHERE competition_id = ANY(@competition_ids::uuid[])
ORDER BY competition_id, position;

-- name: ListCompetitions :many
-- The Competitions browse/list read path (T9.4), mirroring socialplay's
-- ListGames. Only 'scheduled' Competitions are ever returned — a cancelled
-- Competition can't be entered (domain.Enter rejects it), so it has no place
-- in a player-facing browse list.
--
-- spots_left is computed HERE, server-side, using the WEIGHTED formula:
-- capacity minus SUM(1 + guest_count) across this Competition's active,
-- non-cancelled entries. This is the SQL half of CLAUDE.md rule 4's dual
-- enforcement for the same derived quantity — domain.SpotsLeft
-- (internal/competitions/domain/entry.go) expresses the identical rule in
-- pure Go, and the -tags=integration test asserts the two agree against a
-- real Postgres. An UNWEIGHTED (COUNT-based) version would advertise 7 free
-- places on a capacity-8 Competition holding one entry that brought 3
-- guests, when the truth is 4: a visible lie to every player reading the
-- listing.
--
-- It is a CORRELATED SUBQUERY, deliberately, NOT a LEFT JOIN + GROUP BY the
-- way ListGames does it. That difference is load-bearing rather than
-- stylistic: a Competition also has a one-to-many competition_sessions
-- table, and joining BOTH children in one query would fan out the rows and
-- multiply each entry's weight by the number of sessions — silently
-- inflating occupancy and under-reporting spots_left on every multi-session
-- Competition. (games has no such second child table, which is why the join
-- form is safe there and not here.) The subquery cannot fan out, so
-- sessions are fetched separately by ListSessionsForCompetitions above.
--
-- GREATEST(..., 0) is a defensive floor mirroring domain.SpotsLeft's own:
-- the capacity guard trigger should make a negative value unreachable, but a
-- read path must never surface a negative "spots left" to a player even if
-- that invariant were somehow violated.
--
-- venue_facility_id/starts_after/starts_before are optional filters via
-- sqlc.narg (pgtype.UUID / pgtype.Timestamptz with Valid: false meaning
-- "unset"). The two time bounds filter on the Competition's EARLIEST session
-- start — the same value the result is ordered by, so the filter and the
-- ordering agree on one definition of when a Competition "starts" rather
-- than two (see port.CompetitionListingFilter's doc comment).
--
-- A Competition always has at least one session (domain.NewCompetition
-- rejects an empty list), so the MIN(starts_at) subquery is never NULL for
-- any row this query can legitimately return.
SELECT c.id, c.host_id, c.name, c.venue_facility_id, c.capacity, c.guest_allowance, c.payment_method, c.format, c.status, c.entry_fee_cents, c.entry_fee_currency, c.share_token,
       GREATEST(
           c.capacity - COALESCE((
               SELECT SUM(1 + e.guest_count)
               FROM competition_entries e
               WHERE e.competition_id = c.id
                 AND e.status <> 'cancelled'
           ), 0),
           0
       )::int AS spots_left
FROM competitions c
WHERE c.status = 'scheduled'
  AND (sqlc.narg(venue_facility_id)::uuid IS NULL OR c.venue_facility_id = sqlc.narg(venue_facility_id))
  AND (sqlc.narg(starts_after)::timestamptz IS NULL OR (
        SELECT MIN(s.starts_at) FROM competition_sessions s WHERE s.competition_id = c.id
      ) >= sqlc.narg(starts_after))
  AND (sqlc.narg(starts_before)::timestamptz IS NULL OR (
        SELECT MIN(s.starts_at) FROM competition_sessions s WHERE s.competition_id = c.id
      ) <= sqlc.narg(starts_before))
ORDER BY (SELECT MIN(s.starts_at) FROM competition_sessions s WHERE s.competition_id = c.id), c.id;

-- name: UpdateCompetitionStatus :one
-- Dedicated single-column update (app.Service.CancelCompetition), following
-- the same one-query-per-updatable-field convention as
-- UpdateRegistrationStatus and UpdateBookingStatus: scoped to exactly the
-- status column, so this write path cannot accidentally clobber a
-- Competition's capacity, sessions, or share token.
UPDATE competitions
SET status = $2
WHERE id = $1
RETURNING id, host_id, name, venue_facility_id, capacity, guest_allowance, payment_method, format, status, entry_fee_cents, entry_fee_currency, share_token;

-- name: CreateCompetitionEntry :one
-- The write the weighted capacity guard fires on. A P0001 raised by
-- enforce_competition_capacity() (db/migrations/0014_competitions.sql)
-- becomes domain.ErrCompetitionFull in the adapter, and a 23505 on
-- competition_entries_active_player_idx becomes domain.ErrAlreadyEntered —
-- neither leaks as a raw pgconn.PgError (CLAUDE.md rule 5).
INSERT INTO competition_entries (id, competition_id, player_id, guest_count, source, status, payment_status)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, competition_id, player_id, guest_count, source, status, payment_status;

-- name: ListActiveEntriesForCompetition :many
-- Non-cancelled entries only — the capacity-safe read path
-- app.Service.EnterCompetition uses to re-derive the weighted occupancy and
-- already-entered players before calling domain.Enter. Deliberately distinct
-- from ListEntriesForCompetition below; see port.Repository's doc comments
-- for why the two are separate methods rather than one with a flag.
SELECT id, competition_id, player_id, guest_count, source, status, payment_status
FROM competition_entries
WHERE competition_id = $1
  AND status <> 'cancelled'
ORDER BY created_at;

-- name: ListEntriesForCompetition :many
-- EVERY entry, cancelled ones included — the Host-facing roster read behind
-- the ListEntriesForCompetition RPC. A cancelled entry is real information a
-- Host needs ("who withdrew" and "who never entered" are different answers,
-- and a roster that hides the former can't be reconciled against payments),
-- which is exactly why this must not reuse the filtered query above.
SELECT id, competition_id, player_id, guest_count, source, status, payment_status
FROM competition_entries
WHERE competition_id = $1
ORDER BY created_at;

-- name: GetCompetitionEntryByID :one
-- T10.6 (closes #96): the read app.Service.MarkCompetitionEntryPaymentStatus
-- needs before validating and writing a PaymentStatus transition — mirrors
-- socialplay.sql's GetRegistrationByID exactly.
SELECT id, competition_id, player_id, guest_count, source, status, payment_status
FROM competition_entries
WHERE id = $1;

-- name: UpdateCompetitionEntryPaymentStatus :one
-- Dedicated single-column update for PaymentStatus (T10.6, closes #96),
-- mirroring UpdateCompetitionStatus's/socialplay's UpdateRegistrationPaymentStatus's
-- shape: a separate query per updatable field rather than one generic
-- "update everything" statement, so this write path can never accidentally
-- touch status (or guest_count, or anything else) — or vice versa. The sole
-- caller is app.Service.MarkCompetitionEntryPaymentStatus, itself only
-- called through port.CompetitionEntryPaymentUpdater by
-- internal/payments/adapter/competitions.
UPDATE competition_entries
SET payment_status = $2
WHERE id = $1
RETURNING id, competition_id, player_id, guest_count, source, status, payment_status;

-- name: CancelAllActiveEntriesForCompetition :execrows
-- T16.3 (closes the mirrored Competitions gap found this ceremony; see
-- socialplay.sql's CancelAllActiveRegistrationsForGame, which this mirrors
-- field-for-field apart from the owning column). Fired by
-- app.Service.CancelCompetition immediately after a Competition's own
-- status write persists, so a Host's single cancel action leaves no active
-- CompetitionEntry behind. ONE atomic statement, not N sequential
-- UpdateEntryPaymentStatus/status-style calls — a round trip per active
-- entry would multiply a single Host action into N, and the set-based
-- UPDATE is what closes the race a loop of individual updates would leave
-- open: an entry that CreateEntry lands concurrently, between this
-- Competition's own status write and this statement running, still
-- matches "status <> 'cancelled' AND competition_id = $1" and is swept up
-- in the same statement as every entry that existed before the cancel
-- began.
--
-- Scoped to status <> 'cancelled' so an already-cancelled entry (a player
-- who withdrew before the Host cancelled the Competition) is left alone
-- rather than rewritten, and :execrows returns exactly the count of rows
-- this call actually transitioned.
--
-- Deliberately does NOT touch payment_status (T16.3 instruction 4: this
-- ticket cancels the CompetitionEntry only, and does not call
-- RefundPayment or decide whether a future ticket should). Competitions
-- has no waitlist (T16.3 instruction 5 scopes that half to Social Play
-- only), so there is no sibling waitlist concern here.
UPDATE competition_entries
SET status = 'cancelled'
WHERE competition_id = $1
  AND status <> 'cancelled';

-- name: AssignCompetitionAdmin :one
-- T15.3 (partial fix for #168), mirroring socialplay.sql's AssignGameAdmin.
-- assigned_at is always supplied by the adapter (app.Service passes
-- time.Now()) rather than defaulted in the column, so the timestamp the domain
-- value carries and the timestamp the row carries are the same value rather
-- than two clocks that agree by luck — created_at keeps its own DEFAULT now()
-- as the row's insertion record.
--
-- A duplicate (competition_id, user_id) violates competition_admins' composite
-- primary key (23505); the adapter translates that into
-- domain.ErrAlreadyCompetitionAdmin, which is the authoritative guard behind
-- domain.AssignCompetitionAdmin's own fail-fast pre-check (CLAUDE.md rule 4).
-- A competition_id naming no Competition violates the FK (23503) and
-- translates to domain.ErrCompetitionNotFound.
INSERT INTO competition_admins (competition_id, user_id, assigned_by, assigned_at)
VALUES ($1, $2, $3, $4)
RETURNING competition_id, user_id, assigned_by, assigned_at;

-- name: RevokeCompetitionAdmin :one
-- T15.3. RETURNING (with :one) rather than a bare DELETE, so the adapter can
-- tell "removed an assignment" from "there was nothing to remove": zero rows
-- back arrives as pgx.ErrNoRows and becomes
-- domain.ErrCompetitionAdminNotFound. A revoke that silently removed nothing
-- would confirm a Host's belief about who holds authority that this table does
-- not support — see port.CompetitionAdminRepository.Revoke's doc comment.
DELETE FROM competition_admins
WHERE competition_id = $1 AND user_id = $2
RETURNING competition_id, user_id, assigned_by, assigned_at;

-- name: ListCompetitionAdmins :many
-- T15.3 — the read half the sprint plan requires this ticket to ship even
-- though it does not consume it (§A12 GAP A): T15.4 resolves the roster read's
-- entitled set from this query, and T15.5 reaches app.Service.
-- ListCompetitionAdmins from the Payments context. Resolution goes through
-- domain.HasCompetitionAdmin, so the membership rule including its blank-entry
-- guard stays in the domain rather than in SQL (CLAUDE.md rule 2).
--
-- Ordered oldest assignment first, with user_id as the tiebreaker so two
-- assignments sharing an assigned_at (possible: the column is caller-supplied)
-- still come back in a stable order rather than whatever the heap returns.
-- Existence of competition_id itself is checked by the app layer via
-- Repository.GetByID before this runs — this query answers an empty list for
-- an unknown Competition, which is a real and common state (most Competitions
-- have no admins) and must not be confused with "no such Competition".
SELECT competition_id, user_id, assigned_by, assigned_at
FROM competition_admins
WHERE competition_id = $1
ORDER BY assigned_at, user_id;
