-- T68.2 — a quote says what currency it is in (ADR-0005, closes #357).
--
-- WHY: ADR-0005 is **Accepted** and decided that an ISO 4217 code sits
-- beside every `price_cents`-style column — in its own words, "starting with
-- `pricing_rules`". The column reached `payments` (0005), games (0013),
-- competitions (0014) and `discount_rules` (0017) and never reached the table
-- the ADR named first. The sharpest consequence, and the one #357 reports:
-- GetQuoteResponse carries a bare `price_cents`/`band_price_cents` AND nests
-- a `DiscountRule` that does declare `string currency`. One message, two
-- conventions — a client reading the discount's currency knows what it is
-- denominated in and a client reading the price does not.
--
-- ADR-0009's market-scope question, open since T7, was answered on
-- 2026-10-07: **both markets, Vietnam and global.** That is what turns an
-- implicit single currency from a reasonable prototype shortcut into a
-- defect: there is no longer one value the whole platform can be assumed to
-- mean.
--
-- WHY `currency` AND NOT `currency_code`: ADR-0005 writes "`currency_code`
-- (e.g. Postgres `char(3)`)", and this column is deliberately named after
-- its sibling in the same bounded context instead. `discount_rules.currency`
-- (0017) is `text`, the proto field it serialises to is
-- `DiscountRule.currency`, and `domain.Money.Currency` is what both map to —
-- so `currency text` is the name and type that already hold in Booking.
-- CLAUDE.md rule 7 wants one word for one concept within a context; matching
-- the ADR's illustrative spelling here would mean Booking had two. (The
-- `char(3)` vs `text` half matters too: `char(3)` blank-pads, so a
-- three-letter code read back from `char(3)` compares equal to itself but a
-- shorter stored value silently gains spaces. `payments` took `char(3)`;
-- Booking's own column did not, and this follows Booking.)
--
-- NO DEFAULT, and that is the decision rather than an oversight. ADR-0009
-- says both markets, so there is no value a missing currency could honestly
-- mean. A `DEFAULT 'USD'` — which 0013 and 0028/0029 chose for their own
-- columns, under the single-launch-market assumption that ADR-0009 has since
-- answered — would make every future un-populated pricing rule silently
-- claim dollars: a wrong currency that looks exactly like a right one, which
-- is precisely the failure shape 0034 refused for `facilities.timezone` and
-- for the same reason. `NOT NULL` with no default means a future
-- CreatePricingRule writer must say.
--
-- NO BACKFILL, in the sense 0034 uses the phrase — nothing is being inferred
-- for rows whose real value is unknown:
--   * There are no deployed pricing rules to backfill. This project is
--     pre-launch, and docker compose applies these files via initdb.d **only
--     on a fresh volume** (CLAUDE.md gotchas), so every environment that has
--     ever run them starts at 0001 and arrives here holding exactly the rows
--     the seeds created.
--   * Those rows are 0004_seed_pricing.sql's three, for the demo court, and
--     they are the only `INSERT INTO pricing_rules` in the repository
--     (`grep -rn "INSERT INTO pricing_rules" db/` returns that one
--     statement). Their currency is therefore not unknown — it is a fact
--     about a demo fixture, chosen below.
--   * `pricing_rules` has no write path at all: no CreatePricingRule use
--     case, no sqlc INSERT, no Pricing CRUD. So there is no row anywhere
--     whose currency this migration has to guess.
--
-- Because the three seeded rows already exist when this file runs, the column
-- is added nullable, populated explicitly, and only then made NOT NULL. The
-- `SET NOT NULL` is deliberately the last step and is load-bearing: it is
-- what fails the migration, loudly and on a fresh volume, if a row this file
-- did not anticipate ever appears. An `ADD COLUMN ... NOT NULL DEFAULT 'XXX'`
-- would have been one statement and would have silently labelled that row
-- instead.
--
-- Prototype-only migration tooling, as every file here: adopt
-- golang-migrate/goose before production.

ALTER TABLE pricing_rules
    ADD COLUMN currency text;

-- The demo court's three bands, in Vietnamese dong.
--
-- WHY VND: 0034 seeds the demo Facility as `Asia/Ho_Chi_Minh` and attaches
-- both seeded courts to it, and 0004_seed_pricing.sql gives the only
-- pricing_rules rows in the schema to one of those courts. A demo facility in
-- Ho Chi Minh City priced in dollars would be the same kind of
-- self-inconsistent fixture 0034's own header argues against — and it would
-- quietly re-assert the single-market default this ticket removes, in the one
-- place a human poking `make up` would see it.
--
-- Scoped by court_id rather than a bare `UPDATE pricing_rules`: this
-- statement's job is to state 0004's rows' currency, not to label whatever
-- happens to be in the table. Anything else is caught by the SET NOT NULL
-- below rather than absorbed here.
--
-- KNOWN COSMETIC ODDITY, recorded rather than fixed: VND has an ISO 4217
-- exponent of 0, so "minor units" and units are the same thing and these
-- amounts read as 2000/3500/3000 dong (a few US cents) rather than the
-- plausible court fees they were written as. Re-pricing them is out of this
-- ticket's scope — the seeded numbers are asserted by name in
-- internal/booking/adapter/postgres/facility_timezone_quote_integration_test.go
-- and by the README walkthrough — and teaching the stack about per-currency
-- exponents is a decision nobody has made (ADR-0005 assumes integer minor
-- units and says nothing about zero-decimal currencies). Flagged here so the
-- next reader knows it was seen and not missed.
UPDATE pricing_rules
SET currency = 'VND'
WHERE court_id = '11111111-1111-1111-1111-111111111111';

ALTER TABLE pricing_rules
    ALTER COLUMN currency SET NOT NULL;

-- The DB half of rule 4. The Go half is
-- internal/booking/domain.isValidCurrencyCode, which is this same predicate —
-- exactly three uppercase letters — written as a loop; the two are
-- deliberately one invariant stated twice.
--
-- A shape check, not a validity check, exactly as 0034's
-- facilities_timezone_shape is. Whether 'XQZ' is a real currency is a
-- question for the ISO 4217 register, which is not in this database and which
-- ADR-0005 explicitly declined to carry ("v1 only ever populates it with one
-- constant value"; a registry table would be one more thing to keep current).
-- What the schema CAN refuse is a value that could never be an ISO 4217
-- alphabetic code at all: '', a lowercase code, a currency symbol, the
-- 3-digit numeric code, or a padded one. Note in particular that '' is
-- refused: a column that merely exists and holds the empty string would
-- relocate #357 rather than close it, which is why this constraint is here
-- and not left off as `discount_rules.currency`'s deliberately was.
--
-- Unlike 0017's reasoning for leaving its own currency column unconstrained,
-- there is no asymmetry to create here: that column is meaningfully empty for
-- a percent rule, and `domain.NewDiscountRule` validates no currency at all,
-- so a CHECK there would have been a DB-only invariant. This one has its
-- domain counterpart in the same ticket.
ALTER TABLE pricing_rules
    ADD CONSTRAINT pricing_rules_currency_shape
    CHECK (currency ~ '^[A-Z]{3}$');
