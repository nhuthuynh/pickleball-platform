-- name: ListPricingRulesForCourt :many
-- currency joins the column list in T68.2 (ADR-0005, #357): a band price the
-- domain cannot denominate is one domain.ResolvePrice refuses, so this read
-- must carry it. Per the sqlc gotcha in CLAUDE.md, adding a column here
-- changes the generated ...Row struct for this query alone — the adapter
-- converts from the shared columns, never from a shared table model.
SELECT id, court_id, band, weekdays, start_min, end_min, price_cents, currency
FROM pricing_rules
WHERE court_id = $1;
