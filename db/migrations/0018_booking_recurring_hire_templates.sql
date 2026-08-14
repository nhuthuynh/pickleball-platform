-- RecurringHireTemplates for the Booking context (T11.5, on top of T11.4's
-- internal/booking/domain.RecurringHireTemplate — see
-- docs/process/t11-sprint-plan.md T11.5). Migration number 0018 was
-- pre-assigned by that sprint plan's A5 namespace check, not derived from the
-- directory listing, so two tickets in the same sprint cannot claim it (the
-- collision shape that actually happened at T6 and again at T10).
--
-- A template is a Club's *request* for a standing weekly slot; it is not
-- itself a Booking and never becomes one. Approving it generates concrete
-- occurrences (domain.GenerateOccurrences) which are inserted into `bookings`
-- with source = 'recurring_hire', so a recurring hire is subject to exactly
-- the same no-double-booking EXCLUDE constraint as every other Booking source
-- (D3b). That is why there is deliberately no time-range/EXCLUDE constraint on
-- this table: two templates may legitimately overlap while both sit in
-- `requested` — the conflict is only real once an approval tries to
-- materialise the weeks, and it is resolved there, per occurrence, by the
-- constraint that already exists on `bookings`.
--
-- Prototype-only migration tooling: applied via docker-compose initdb.d on a
-- FRESH volume only (see CLAUDE.md gotchas). Adopt golang-migrate/goose
-- before production.

CREATE TABLE recurring_hire_templates (
    id                    uuid PRIMARY KEY DEFAULT gen_random_uuid(),

    -- The requesting Club's User. The FK into the Identity/Users context's
    -- table mirrors 0011_socialplay_facility_fk.sql's precedent for a
    -- cross-context reference: the app layer has already resolved this user
    -- through port.IdentityLookup and proven they hold the `club` role before
    -- any insert reaches here, so the FK is a backstop against an
    -- inconsistent write, not the primary check.
    requested_by_user_id  uuid NOT NULL REFERENCES identity_users (id),

    court_id              uuid NOT NULL REFERENCES courts (id),

    -- Go's time.Weekday numbering (Sunday = 0 .. Saturday = 6), the same
    -- convention pricing_rules.weekdays (0003) already established for this
    -- codebase, so the Postgres adapter round-trips
    -- domain.RecurringHireTemplate.Weekday without a lookup table. Singular
    -- here, not an array: a template is one weekly slot (T11.4's domain
    -- model), unlike a PricingRule which spans several weekdays at once.
    weekday               smallint NOT NULL CHECK (weekday BETWEEN 0 AND 6),

    -- Minutes since midnight, matching domain.ClockTime — the same
    -- representation and the same column naming pricing_rules.start_min/
    -- end_min use.
    start_min             smallint NOT NULL CHECK (start_min >= 0 AND start_min <= 1440),
    end_min               smallint NOT NULL CHECK (end_min >= 0 AND end_min <= 1440),

    -- The date the weekly pattern is anchored to; generation starts at the
    -- first calendar date on or after this that falls on `weekday`.
    starts_at             timestamptz NOT NULL,

    -- RecurringHireEndCondition is domain's three-shape variant: kind plus
    -- whichever payload that kind carries, with the pairing CHECK below.
    -- Deliberately its own set of columns rather than shared with
    -- discount_rules' identically-shaped end_condition_* columns (0017) —
    -- they are different aggregates in different tables, and the domain keeps
    -- the two variant types separately named for the same reason (see
    -- RecurringHireEndCondition's own doc comment).
    end_condition_kind         text NOT NULL CHECK (
        end_condition_kind IN ('end_after_date', 'end_after_occurrences', 'no_end')
    ),
    end_condition_date         timestamptz,
    end_condition_occurrences  integer,

    -- The DB half of T11.4's RecurringHireStatus closed enum (CLAUDE.md rule
    -- 4); its domain counterpart is RecurringHireStatus.IsValid plus the
    -- Approve/Reject transition guards. 'cancelled' is included because the
    -- domain models it, even though no T11 ticket writes a transition into it
    -- yet — the enum is the enum, and leaving a modelled value out of the
    -- CHECK would make the two halves disagree.
    status                text NOT NULL CHECK (
        status IN ('requested', 'approved', 'rejected', 'cancelled')
    ),

    created_at            timestamptz NOT NULL DEFAULT now(),

    -- The DB half of NewRecurringHireTemplate's
    -- ErrInvalidRecurringHireTimeRange guard.
    CONSTRAINT recurring_hire_templates_time_range CHECK (end_min > start_min),

    -- EndCondition pairing, the DB half of
    -- ErrInvalidRecurringHireEndAfterOccurrences plus the variant's own
    -- consistency: only the column belonging to the stated kind may be
    -- populated, so a row can never carry two contradictory end conditions.
    CONSTRAINT recurring_hire_templates_end_condition_matches_kind CHECK (
        (end_condition_kind = 'end_after_date'
            AND end_condition_date IS NOT NULL AND end_condition_occurrences IS NULL)
        OR
        (end_condition_kind = 'end_after_occurrences'
            AND end_condition_occurrences IS NOT NULL AND end_condition_occurrences > 0
            AND end_condition_date IS NULL)
        OR
        (end_condition_kind = 'no_end'
            AND end_condition_date IS NULL AND end_condition_occurrences IS NULL)
    )
);

-- ListRecurringHireTemplatesForCourts' access path (the owner-facing queue
-- resolves a Facility to its Courts first, then reads by court_id), mirroring
-- pricing_rules_court_idx and discount_rules_facility_idx.
CREATE INDEX recurring_hire_templates_court_idx ON recurring_hire_templates (court_id);

-- The Club-facing "my requests" access path. No RPC reads by this column yet
-- (T11.6's Club status view is the consumer), but the column is the natural
-- key for it and the index costs nothing on a table this size.
CREATE INDEX recurring_hire_templates_requested_by_idx
    ON recurring_hire_templates (requested_by_user_id);
