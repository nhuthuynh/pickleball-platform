-- name: ListPricingRulesForCourt :many
SELECT id, court_id, band, weekdays, start_min, end_min, price_cents
FROM pricing_rules
WHERE court_id = $1;
