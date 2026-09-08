-- name: CreatePayment :one
INSERT INTO payments (id, payable_type, payable_id, amount_cents, currency_code, method, status, stripe_reference, recorded_by_user_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, payable_type, payable_id, amount_cents, currency_code, method, status, stripe_reference, recorded_by_user_id;

-- name: GetPaymentByID :one
SELECT id, payable_type, payable_id, amount_cents, currency_code, method, status, stripe_reference, recorded_by_user_id
FROM payments
WHERE id = $1;

-- name: GetPaymentByPayable :one
-- T55.3 (issue #124's refund half): the lookup a cancellation cascade needs
-- — it knows which Registration/CompetitionEntry it just cancelled, not
-- which Payment id paid for it.
--
-- Returns at most one row by construction: payments_payable_unique_idx
-- (0005_payments.sql:51) is a UNIQUE index on exactly this pair, which is
-- the authoritative half of domain.EnsureOnePaymentPerPayable's invariant.
-- So :one is correct here rather than optimistic.
SELECT id, payable_type, payable_id, amount_cents, currency_code, method, status, stripe_reference, recorded_by_user_id
FROM payments
WHERE payable_type = $1
  AND payable_id = $2;

-- name: GetPaymentByStripeReference :one
-- T18.1 (closes #167): the lookup a Stripe webhook delivery needs — it
-- carries Stripe's own intent reference (stripe_reference), not this
-- backend's internal Payment id, mirroring GetPaymentByID's shape exactly.
SELECT id, payable_type, payable_id, amount_cents, currency_code, method, status, stripe_reference, recorded_by_user_id
FROM payments
WHERE stripe_reference = $1;

-- name: UpdatePayment :one
-- Persists a status transition (MarkPaid/Refund) plus whatever
-- stripe_reference the transition carried. updated_at is bumped
-- explicitly rather than via a trigger, mirroring how UpdateBookingStatus
-- keeps its write path simple (single-table, no other side effects).
UPDATE payments
SET status = $2,
    stripe_reference = $3,
    updated_at = now()
WHERE id = $1
RETURNING id, payable_type, payable_id, amount_cents, currency_code, method, status, stripe_reference, recorded_by_user_id;

-- name: ClaimWebhookEvent :execrows
-- T18.1 (closes #167): port.WebhookEventStore.ClaimEvent's backing write.
-- ON CONFLICT DO NOTHING means a redelivered event_id inserts zero rows
-- rather than erroring; :execrows lets the adapter tell "this call claimed
-- it" (1 row) apart from "already claimed" (0 rows) without a second
-- round trip.
INSERT INTO payments_webhook_events (event_id)
VALUES ($1)
ON CONFLICT (event_id) DO NOTHING;
