-- DiscountRules for the Booking context (T11.2, on top of T11.1's
-- internal/booking/domain.DiscountRule — see docs/process/t11-sprint-plan.md
-- T11.2). Migration number 0017 was pre-assigned by that sprint plan's A5
-- namespace check, not derived from the directory listing, so two tickets in
-- the same sprint cannot claim it (the collision shape that actually happened
-- at T6 and again at T10).
--
-- A DiscountRule sits on top of pricing_rules (0003), it does not replace it:
-- GetQuote resolves a band price first and then applies at most one discount
-- to it. Scope is facility-wide only this sprint (sprint plan A9's explicit
-- narrowing), hence facility_id and no court_id column — a court-level
-- discount would be an additive follow-up, not a schema rewrite.
--
-- Prototype-only migration tooling: applied via docker-compose initdb.d on a
-- FRESH volume only (see CLAUDE.md gotchas). Adopt golang-migrate/goose
-- before production.

CREATE TABLE discount_rules (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    facility_id   uuid NOT NULL REFERENCES facilities (id),

    discount_type text NOT NULL CHECK (discount_type IN ('percent', 'fixed_amount')),

    -- DiscountAmount is a two-shape value (domain.DiscountAmount) whose
    -- meaningful half is selected by discount_type, so it persists as two
    -- nullable columns plus the pairing CHECK below rather than one column
    -- that means different things on different rows. percent is double
    -- precision because domain.DiscountAmount.Percent is a float64 (12.5% is
    -- a legitimate rule); the *money* it produces never is — that stays
    -- integer cents everywhere (ADR-0005), computed in
    -- DiscountRule.ApplyToCents.
    percent             double precision,
    fixed_amount_cents  bigint,
    -- The ISO 4217 code paired with fixed_amount_cents (domain.Money.
    -- Currency). Deliberately not CHECK-constrained: domain.NewDiscountRule
    -- does not validate currency either, and a DB-only invariant with no
    -- domain counterpart is exactly what CLAUDE.md rule 4 asks us not to
    -- create. Empty for a percent rule, which has no currency.
    currency            text NOT NULL DEFAULT '',

    -- AppliesTo persists as a text[] of domain.Source values, mirroring how
    -- pricing_rules.weekdays (0003) already made this same array-vs-table
    -- call for the same kind of small, closed, always-read-whole set. text[]
    -- (not smallint[], not a Postgres ENUM type):
    --   * the values are the same strings bookings.source already stores as
    --     text with an identical CHECK (0001_init.sql), so one ubiquitous
    --     language holds across both tables and the Go domain (CLAUDE.md
    --     rule 7) with no numeric mapping to keep in sync — the very
    --     coupling HANDOFF.md's cross-cutting note flags about
    --     pricing_rules.weekdays leaning on time.Weekday's numbering;
    --   * a CREATE TYPE ... AS ENUM would give the same integrity via a
    --     type whose values can only be added by ALTER TYPE, migration-
    --     coupling a closed domain enum for no gain here;
    --   * the CHECK below is the authoritative half of the dual enforcement
    --     (CLAUDE.md rule 4) whose domain counterpart is
    --     NewDiscountRule's Source.IsValid()/non-empty loop.
    applies_to    text[] NOT NULL CHECK (
        cardinality(applies_to) > 0
        AND applies_to <@ ARRAY['recurring_hire', 'individual', 'game', 'competition']::text[]
    ),

    starts_at     timestamptz NOT NULL,

    -- EndCondition is domain.EndCondition's three-shape variant: kind plus
    -- whichever payload that kind carries, with the pairing CHECK below.
    end_condition_kind         text NOT NULL CHECK (
        end_condition_kind IN ('end_after_date', 'end_after_occurrences', 'no_end')
    ),
    end_condition_date         timestamptz,
    end_condition_occurrences  integer,

    created_at    timestamptz NOT NULL DEFAULT now(),

    -- Amount/type pairing, the DB half of NewDiscountRule's
    -- ErrInvalidDiscountAmount bounds: percent in (0,100], fixed a positive
    -- amount, and the column belonging to the *other* discount_type null so
    -- a row can never carry two contradictory amounts.
    CONSTRAINT discount_rules_amount_matches_type CHECK (
        (discount_type = 'percent'
            AND percent IS NOT NULL AND percent > 0 AND percent <= 100
            AND fixed_amount_cents IS NULL)
        OR
        (discount_type = 'fixed_amount'
            AND fixed_amount_cents IS NOT NULL AND fixed_amount_cents > 0
            AND percent IS NULL)
    ),

    -- EndCondition pairing, the DB half of
    -- ErrInvalidEndConditionOccurrences plus the variant's own consistency.
    CONSTRAINT discount_rules_end_condition_matches_kind CHECK (
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

-- ListDiscountRulesForFacility's access path, mirroring
-- pricing_rules_court_idx.
CREATE INDEX discount_rules_facility_idx ON discount_rules (facility_id);

-- Deliberately NO constraint preventing two rules from overlapping on
-- (facility_id, source, time window). That mirrors pricing_rules, which has
-- no such constraint either, and it is ADR-0002's decision rather than an
-- omission: an ambiguous match is a data-configuration error surfaced at
-- resolve time (domain.ResolveDiscount -> ErrAmbiguousDiscountRule ->
-- FailedPrecondition), never silently resolved by priority or insertion
-- order — and never silently *prevented* at write time either, which would
-- just relocate the same misconfiguration into a confusing INSERT failure.
