-- RecurringHireTemplates for the Booking context's RequestRecurringHire /
-- ApproveRecurringHire / RejectRecurringHire /
-- ListRecurringHireTemplatesForFacility RPCs (T11.5). Kept in its own file
-- rather than appended to booking.sql or discounts.sql: it is a separate
-- aggregate with its own lifecycle, and sqlc.yaml lists query files
-- explicitly so a new file is a deliberate registration, not a silent sweep.

-- name: CreateRecurringHireTemplate :one
-- The id is always supplied by the caller (the app layer, via its
-- IDGenerator port) rather than defaulted by gen_random_uuid(), matching
-- CreateBooking/CreateDiscountRule — the server mints it, never the API
-- caller (T11.5's creation-RPC checklist item 1).
INSERT INTO recurring_hire_templates (
    id, requested_by_user_id, court_id, weekday, start_min, end_min,
    starts_at, end_condition_kind, end_condition_date,
    end_condition_occurrences, status
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING id, requested_by_user_id, court_id, weekday, start_min, end_min,
          starts_at, end_condition_kind, end_condition_date,
          end_condition_occurrences, status;

-- name: GetRecurringHireTemplateByID :one
SELECT id, requested_by_user_id, court_id, weekday, start_min, end_min,
       starts_at, end_condition_kind, end_condition_date,
       end_condition_occurrences, status
FROM recurring_hire_templates
WHERE id = $1;

-- name: UpdateRecurringHireTemplateStatus :one
-- Status is the only mutable column: T11.4 models approve/reject as status
-- transitions and nothing else, so a general-purpose UPDATE would offer a
-- write path the domain has no transition for.
UPDATE recurring_hire_templates
SET status = $2
WHERE id = $1
RETURNING id, requested_by_user_id, court_id, weekday, start_min, end_min,
          starts_at, end_condition_kind, end_condition_date,
          end_condition_occurrences, status;

-- name: ListRecurringHireTemplatesForCourts :many
-- Keyed by a court id array rather than by facility: a template references a
-- Court, and Booking does not read the courts table (the Facilities context
-- owns the court->facility fact, resolved through port.FacilityLookup before
-- this query runs). Ordered oldest-first so an owner's incoming-requests
-- queue reads in arrival order.
SELECT id, requested_by_user_id, court_id, weekday, start_min, end_min,
       starts_at, end_condition_kind, end_condition_date,
       end_condition_occurrences, status
FROM recurring_hire_templates
WHERE court_id = ANY(@court_ids::uuid[])
ORDER BY created_at;
