-- name: CreatePayment :one
INSERT INTO payments (id, payable_type, payable_id, amount_cents, currency_code, method, status, stripe_reference, recorded_by_user_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, payable_type, payable_id, amount_cents, currency_code, method, status, stripe_reference, recorded_by_user_id;

-- name: GetPaymentByID :one
SELECT id, payable_type, payable_id, amount_cents, currency_code, method, status, stripe_reference, recorded_by_user_id
FROM payments
WHERE id = $1;

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
