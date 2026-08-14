-- Discount rules for the Booking context's CreateDiscountRule /
-- ListDiscountRulesForFacility RPCs and its extended GetQuote (T11.2).
-- Kept in its own file rather than appended to pricing.sql: they are
-- separate aggregates with separate lifecycles (pricing is read-only from
-- Booking, discounts have a real write path), and sqlc.yaml lists query
-- files explicitly so a new file is a deliberate registration, not a
-- silent sweep.

-- name: CreateDiscountRule :one
-- The id is always supplied by the caller (the app layer, via its
-- IDGenerator port) rather than defaulted by gen_random_uuid(), matching
-- CreateBooking/CreateFacility — the server mints it, never the API caller
-- (T11.2's creation-RPC checklist item 1).
INSERT INTO discount_rules (
    id, facility_id, discount_type, percent, fixed_amount_cents, currency,
    applies_to, starts_at, end_condition_kind, end_condition_date,
    end_condition_occurrences
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING id, facility_id, discount_type, percent, fixed_amount_cents, currency,
          applies_to, starts_at, end_condition_kind, end_condition_date,
          end_condition_occurrences;

-- name: ListDiscountRulesForFacility :many
-- Unfiltered by time or source on purpose: domain.ResolveDiscount does its
-- own window/Source matching (and its own ambiguity detection) over the
-- whole rule set, exactly as ResolvePrice does for pricing_rules. Pushing
-- that filtering into SQL would split one invariant across two places.
SELECT id, facility_id, discount_type, percent, fixed_amount_cents, currency,
       applies_to, starts_at, end_condition_kind, end_condition_date,
       end_condition_occurrences
FROM discount_rules
WHERE facility_id = $1
ORDER BY created_at;
