-- T56.1 (issue #126) — a Registration records what it owes, frozen at
-- registration time: the Game's per-player entry fee once per HEAD.
--
-- 0028 is the next free number: db/migrations ended at 0027
-- (0027_booking_owner_user_id.sql), confirmed by listing the directory
-- rather than assuming a plan's number was still free.
--
-- WHY THIS COLUMN EXISTS
--
-- #126's headline ask — "add a real per-Game price field and retire T8.10's
-- PLACEHOLDER_REGISTRATION_FEE_CENTS" — was already delivered by T9.2,
-- three sprints before that issue was even opened: games.entry_fee_cents
-- has existed since 0013_socialplay_entry_fee.sql.
--
-- What was never built is the part the Product Owner answered on
-- 2026-09-04: **per head, including guests**. Nothing multiplied the entry
-- fee by the party size anywhere in the stack, so a player bringing three
-- guests paid for one. The Game's GuestAllowance and the Registration's
-- guest_count were enforced for *capacity* and never reached the price.
--
-- WHY FROZEN ON THE REGISTRATION RATHER THAN DERIVED FROM THE GAME
--
-- The owed amount could have been computed on every read as
-- games.entry_fee_cents * (1 + registrations.guest_count). It is stored
-- instead, which was the Product Owner's explicit choice ("freeze, and lock
-- guests once paid"), for two reasons that outlive the convenience:
--
--   1. A Host who raises the entry fee after a player registered must not
--      retroactively change what that player agreed to pay. A derived
--      figure would do exactly that, silently.
--   2. Issue #297 makes Payments validate a payment against this figure.
--      That check only means something if the figure is the one actually
--      agreed at registration, not one that can move underneath it.
--
-- The cost is the usual cost of denormalising: this column can drift from
-- games.entry_fee_cents * (1 + guest_count), and *that is the point* — the
-- drift is the historical record. It is not a bug to be reconciled away.
--
-- WHY guest_count IS SAFE TO FREEZE AGAINST TODAY
--
-- guest_count is written once, by RegisterForGame, and has no mutation
-- path: socialplay.proto exposes only RegisterForGame and
-- CancelRegistration, and nothing calls the repository's Update to change
-- it. So "lock guests once paid" currently holds by construction rather
-- than by a guard, and no guard is added here for an operation that does
-- not exist. A future ticket adding a change-your-guests path must either
-- recompute this column or refuse the change once a Payment exists — see
-- domain.Registration.GuestCount's doc comment, which carries the same
-- warning where a Go reader will hit it.
--
-- BACKFILL, AND WHY THE DEFAULT IS 0 RATHER THAN THE COMPUTED AMOUNT
--
-- DEFAULT 0 mirrors 0013's own choice for games.entry_fee_cents, and 0 is a
-- real value in this domain (a free registration) rather than a sentinel —
-- see domain.Money.IsFree.
--
-- A cleverer backfill (UPDATE ... SET amount_owed_cents = entry_fee_cents *
-- (1 + guest_count) FROM games ...) is deliberately NOT done. Existing rows
-- were created under the flat-per-registration rule, and their players were
-- charged accordingly; rewriting history to say they owed more would
-- manufacture arrears nobody agreed to. On the only path this migration
-- ever runs — docker-compose initdb.d on a FRESH volume, per CLAUDE.md's
-- migration gotcha — the table is empty and the question is moot anyway.
-- Adopt golang-migrate/goose before production, at which point a real
-- backfill decision is owed for real rows.

ALTER TABLE registrations
    ADD COLUMN amount_owed_cents bigint NOT NULL DEFAULT 0,
    ADD COLUMN amount_owed_currency text NOT NULL DEFAULT 'USD';

-- The DB-side half of domain.Money.Validate's "no negative amount" rule
-- (CLAUDE.md rule 4: invariants live in Postgres AND the domain, and the
-- two must be kept in sync if either changes). Mirrors
-- games_entry_fee_cents_non_negative from 0013 exactly — there is no
-- Registration that pays a player to attend.
ALTER TABLE registrations
    ADD CONSTRAINT registrations_amount_owed_cents_non_negative
        CHECK (amount_owed_cents >= 0);
