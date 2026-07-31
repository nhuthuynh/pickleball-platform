-- Pricing rules for the Booking context's GetQuote use case (HANDOFF.md T1).
-- Weekdays are stored as smallint[] using Go's time.Weekday numbering
-- (Sunday = 0 .. Saturday = 6) so the Postgres adapter can round-trip
-- domain.PricingRule.Weekdays without a lookup table. Start/End are minutes
-- since midnight, matching domain.ClockTime.
CREATE TABLE pricing_rules (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    court_id    uuid NOT NULL REFERENCES courts (id),
    band        text NOT NULL CHECK (band IN ('weekday', 'peak', 'weekend')),
    weekdays    smallint[] NOT NULL CHECK (
        cardinality(weekdays) > 0
        AND weekdays <@ ARRAY[0,1,2,3,4,5,6]::smallint[]
    ),
    start_min   smallint NOT NULL CHECK (start_min >= 0 AND start_min <= 1440),
    end_min     smallint NOT NULL CHECK (end_min >= 0 AND end_min <= 1440),
    price_cents bigint NOT NULL CHECK (price_cents > 0),
    created_at  timestamptz NOT NULL DEFAULT now(),
    CHECK (end_min > start_min)
);

CREATE INDEX pricing_rules_court_idx ON pricing_rules (court_id);
