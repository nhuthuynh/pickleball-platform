//go:build integration

// T68.2 (ADR-0005, closes #357). The half of "a quote says what currency it
// is in" that no in-memory fixture can prove, because the facts it rests on
// are SQL and schema facts:
//
//  1. **0035 seeds a currency for 0004's three rules, and it is VND.**
//     Requirement 4 of this ticket. 0034 made the demo Facility
//     Asia/Ho_Chi_Minh; the one court in the schema with pricing_rules rows
//     belongs to it, so a dollar-priced demo facility in Ho Chi Minh City
//     would be a self-inconsistent fixture. The seed is the claim, so the
//     seed is what is read back.
//  2. **The real sqlc query carries the column through to the domain.**
//     db/queries/pricing.sql gained `currency`, which regenerates a new
//     ...Row struct for that query alone (the sqlc gotcha in CLAUDE.md). A
//     quote driven through the real Postgres adapter is the only thing that
//     proves the adapter reads the real column rather than a zero value.
//  3. **The DB half of rule 4 actually refuses bad data.** 0035's NOT NULL
//     and pricing_rules_currency_shape CHECK are the write-side statement of
//     the same predicate domain.isValidCurrencyCode makes in Go. CLAUDE.md:
//     "an in-memory fake is more permissive than Postgres, and that asymmetry
//     hides storage bugs" — 0031's defect was exactly a CHECK nobody tested —
//     and "a DB-level guard with no DB-level test is a comment".
//
// Excluded from `make test-domain`, `make test-adapters` and plain
// `go test ./...` by the integration build tag; compiled (not run) by `make
// vet-integration`; executed by `make ci-integration` or `make test`.
// Requires Docker.
package postgres_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	bookingfacilities "github.com/nhuthuynh/white-label/internal/booking/adapter/facilities"
	bookingpg "github.com/nhuthuynh/white-label/internal/booking/adapter/postgres"
	bookingapp "github.com/nhuthuynh/white-label/internal/booking/app"
	"github.com/nhuthuynh/white-label/internal/booking/domain"
	facilitiespg "github.com/nhuthuynh/white-label/internal/facilities/adapter/postgres"
	facilitiesapp "github.com/nhuthuynh/white-label/internal/facilities/app"
	"github.com/nhuthuynh/white-label/internal/platform/idgen"
	"github.com/nhuthuynh/white-label/internal/platform/tz"
)

// seededPricingCurrency is 0035's own value, read off the migration rather
// than guessed, and seededCourtRuleCount is 0004_seed_pricing.sql's row count
// for that court.
const (
	seededPricingCurrency = "VND"
	seededCourtRuleCount  = 3
)

// TestSeededPricingRulesAreDenominatedInTheFacilitysCurrency is claim 1: every
// one of 0004's three seeded rules carries 0035's currency, and nothing in the
// table is left unlabelled.
//
// The second query is the one that would catch a scoped UPDATE that missed a
// row. 0035's `SET NOT NULL` is supposed to make that impossible — it is the
// last statement in the file precisely so an unanticipated row fails the
// migration loudly — but a count asserted here is cheaper than trusting that
// the container would have failed to start for the right reason.
func TestSeededPricingRulesAreDenominatedInTheFacilitysCurrency(t *testing.T) {
	ctx := context.Background()
	pool := startPostgres(t, ctx)

	var total, inSeededCurrency int
	if err := pool.QueryRow(ctx,
		`SELECT count(*), count(*) FILTER (WHERE currency = $1) FROM pricing_rules`,
		seededPricingCurrency,
	).Scan(&total, &inSeededCurrency); err != nil {
		t.Fatalf("counting seeded pricing rules: %v", err)
	}

	if total != seededCourtRuleCount {
		t.Errorf("pricing_rules holds %d rows, want %d — 0004_seed_pricing.sql's three; if a seed was added, "+
			"0035's scoped UPDATE and this test both need to know about it", total, seededCourtRuleCount)
	}
	if inSeededCurrency != total {
		t.Errorf("%d of %d seeded rules are in %s — requirement 4 is that every seeded rule names the demo "+
			"facility's currency (0034 makes it Asia/Ho_Chi_Minh)", inSeededCurrency, total, seededPricingCurrency)
	}
}

// TestQuoteReportsTheSeededCurrencyAgainstRealPostgres is claim 2: the column
// travels sqlc's generated row struct, the Postgres adapter and the app layer
// to land on the Quote. Nothing on this path is faked.
//
// It asserts the price alongside the currency on purpose: a regression that
// dropped the column from db/queries/pricing.sql would leave Currency empty
// while every amount stayed right, and ErrInvalidPricingCurrency would then be
// the symptom — so this test pins the successful quote rather than only the
// refusal.
func TestQuoteReportsTheSeededCurrencyAgainstRealPostgres(t *testing.T) {
	ctx := context.Background()
	pool := startPostgres(t, ctx)

	facilitiesSvc := facilitiesapp.NewService(facilitiespg.NewRepository(pool), unusedFacilitiesIdentity{}, idgen.UUID{})
	svc := bookingapp.NewService(bookingapp.ServiceOptions{
		Bookings:       bookingpg.NewRepository(pool),
		PricingRules:   bookingpg.NewPricingRuleRepository(pool),
		DiscountRules:  bookingpg.NewDiscountRuleRepository(pool),
		RecurringHires: unusedRecurringHireRepository{},
		Facilities:     bookingfacilities.NewLookup(facilitiesSvc),
		Identity:       unusedIdentityLookup{},
		Zones:          tz.Loader{},
		IDs:            idgen.UUID{},
	})

	// 11:00Z is 18:00 Asia/Ho_Chi_Minh on a Monday, which 0004 prices in the
	// peak band at 3500 — the same instant and expectation
	// facility_timezone_quote_integration_test.go uses, so the two tests
	// cannot drift apart about what the seeds say.
	quote, err := svc.GetQuote(ctx, bookingapp.GetQuoteInput{
		CourtID: seedCourtID,
		Source:  domain.SourceIndividual,
		Range:   mustRange(t, "2026-08-03T11:00:00Z", "2026-08-03T12:00:00Z"),
	})
	if err != nil {
		t.Fatalf("GetQuote: %v", err)
	}
	if quote.Currency != seededPricingCurrency {
		t.Errorf("Currency = %q, want %q — the column must reach the domain through the real sqlc query",
			quote.Currency, seededPricingCurrency)
	}
	if quote.Band != domain.BandPeak || quote.PriceCents != 3500 {
		t.Errorf("band/price = %q/%d, want peak/3500 — the seeded band must be unchanged by this ticket",
			quote.Band, quote.PriceCents)
	}
}

// TestPricingRulesCurrencyColumnRefusesUnusableValues is claim 3, and it is
// the test 0035's CHECK would otherwise be a comment without.
//
// The bad values are chosen as the ones a careless constraint would let
// through, and "" is the one that matters most: a column that merely exists
// and holds the empty string would relocate #357 rather than close it, which
// is why this CHECK exists where discount_rules.currency's deliberately does
// not.
func TestPricingRulesCurrencyColumnRefusesUnusableValues(t *testing.T) {
	ctx := context.Background()
	pool := startPostgres(t, ctx)

	// A court and a rule of this test's own, so nothing here mutates the
	// seeded rows the two tests above read.
	court := uuid.NewString()
	if _, err := pool.Exec(ctx, `INSERT INTO courts (id, name) VALUES ($1, 'Currency CHECK Court')`, court); err != nil {
		t.Fatalf("seeding a court: %v", err)
	}
	insert := func(currency string) error {
		_, err := pool.Exec(ctx,
			`INSERT INTO pricing_rules (court_id, band, weekdays, start_min, end_min, price_cents, currency)
			 VALUES ($1, 'weekday', ARRAY[1,2,3,4,5]::smallint[], 360, 1020, 2000, $2)`, court, currency)
		return err
	}

	t.Run("the shape CHECK refuses what could never be an ISO 4217 code", func(t *testing.T) {
		for _, bad := range []string{"", "vnd", "VN", "VNDX", "$", "704", " VND", "VN ", "V-D"} {
			if err := insert(bad); err == nil {
				t.Errorf("stored currency %q, want pricing_rules_currency_shape to refuse it", bad)
				// Undo the row the CHECK should have refused, so one failure
				// does not cascade into the NOT NULL subtest below.
				if _, derr := pool.Exec(ctx, `DELETE FROM pricing_rules WHERE court_id = $1`, court); derr != nil {
					t.Fatalf("cleaning up an unexpectedly accepted row: %v", derr)
				}
			}
		}
	})

	t.Run("NOT NULL refuses an INSERT that omits the column", func(t *testing.T) {
		// 0035 deliberately added no DEFAULT: ADR-0009 answered the market
		// scope as both Vietnam and global on 2026-10-07, so there is no
		// value a missing currency could honestly mean. A writer must say.
		_, err := pool.Exec(ctx,
			`INSERT INTO pricing_rules (court_id, band, weekdays, start_min, end_min, price_cents)
			 VALUES ($1, 'weekday', ARRAY[1,2,3,4,5]::smallint[], 360, 1020, 2000)`, court)
		if err == nil {
			t.Error("an INSERT omitting currency succeeded; 0035 must leave no default for it to fall back on")
		}
	})

	t.Run("a well-shaped code is storable", func(t *testing.T) {
		for _, ok := range []string{"VND", "USD", "GBP", "XQZ"} {
			if err := insert(ok); err != nil {
				t.Errorf("currency %q was refused: %v — the CHECK is a SHAPE check, not a registry lookup "+
					"(XQZ is not a real currency and the schema is deliberately not the place that knows)", ok, err)
			}
			if _, err := pool.Exec(ctx, `DELETE FROM pricing_rules WHERE court_id = $1`, court); err != nil {
				t.Fatalf("cleaning up: %v", err)
			}
		}
	})
}
