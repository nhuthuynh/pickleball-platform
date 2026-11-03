//go:build integration

// T68.1 (ADR-0004, closes #356). The half of "a price band resolves in the
// facility's own timezone" that no in-memory fixture can prove, because the
// facts it rests on are SQL and schema facts:
//
//  1. **0034 puts a timezone on facilities, and 0034's seed attaches both
//     seeded courts to a facility that has one.** The quote path is reachable
//     only for a court with pricing_rules rows, and 0004_seed_pricing.sql
//     gives those to exactly one court — the one the README's curl
//     walkthrough uses. Without the seed, shipping this fix turns the single
//     working demo quote into an error. A fake repository cannot notice that.
//  2. **The band that resolves for an 18:00 local slot is peak.** The band
//     rows are stored as start_min/end_min — clock minutes local to the
//     facility — and read back through sqlc; an 18:00 Ho Chi Minh City slot
//     arrives over the wire as 11:00Z. Before this ticket the real stack
//     answered the weekday band's 2000 for it. That is a claim about the
//     seeded data, the real query and the real conversion together.
//  3. **A court with no facility cannot be quoted at all**, where previously
//     it was quoted in UTC. The in-memory Facilities fake cannot produce a
//     NULL facility_id; this test inserts one.
//
// CLAUDE.md's own lesson for this shape: "an in-memory fake is more permissive
// than Postgres, and that asymmetry hides storage bugs" — and its sharper
// sibling, "a DB-level guard with no DB-level test is a comment".
//
// Excluded from `make test-domain`, `make test-adapters` and plain
// `go test ./...` by the integration build tag; compiled (not run) by `make
// vet-integration`; executed by `make ci-integration` or `make test`.
// Requires Docker.
package postgres_test

import (
	"context"
	"errors"
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

// seededDemoFacilityID and seededDemoFacilityZone are 0034's own values, read
// off the migration rather than guessed. The zone is deliberately +07:00 with
// no DST: an evening slot there lands on the previous UTC day's clock, which
// is the defect #356 reported.
const (
	seededDemoFacilityID   = "33333333-3333-3333-3333-333333333333"
	seededDemoFacilityZone = "Asia/Ho_Chi_Minh"
)

// TestQuoteResolvesInTheFacilityTimezoneAgainstRealPostgres drives the real
// Booking app.Service over the real Postgres repositories AND the real
// Facilities context (through internal/booking/adapter/facilities, the only
// place Booking may call it) with the real tz loader behind it. Nothing in
// this path is faked.
func TestQuoteResolvesInTheFacilityTimezoneAgainstRealPostgres(t *testing.T) {
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

	// Claim 1, asserted before anything else depends on it: the seeded court
	// resolves to the seeded facility, and that facility has 0034's zone.
	zone, err := bookingfacilities.NewLookup(facilitiesSvc).TimezoneForCourt(ctx, seedCourtID)
	if err != nil {
		t.Fatalf("TimezoneForCourt(seeded court): %v — 0034 must attach the seeded courts to a facility WITH a zone, "+
			"or the one demo court that has pricing_rules rows (0004_seed_pricing.sql) cannot be quoted at all", err)
	}
	if zone != seededDemoFacilityZone {
		t.Fatalf("seeded court's zone = %q, want %q (0034's own value)", zone, seededDemoFacilityZone)
	}

	// Claim 2. 0004_seed_pricing.sql prices court 1 at start_min 1020-1320
	// (17:00-22:00) for 3500 on weekdays 1-5, and 360-1020 (06:00-17:00) for
	// 2000. 2026-08-03 is a Monday in both zones.
	//
	//	11:00Z  ->  18:00 +07  ->  clock 1080  ->  peak,    3500
	//	04:00Z  ->  11:00 +07  ->  clock  660  ->  weekday, 2000
	//
	// The first row is the bug: 11:00Z read as a clock time is 660, so before
	// this ticket the real stack answered 2000 for an 18:00 booking.
	tests := []struct {
		name      string
		from, to  string
		wantBand  domain.Band
		wantPrice int64
	}{
		{
			name: "an 18:00 local slot resolves the peak band",
			from: "2026-08-03T11:00:00Z", to: "2026-08-03T12:00:00Z",
			wantBand: domain.BandPeak, wantPrice: 3500,
		},
		{
			name: "a late-morning local slot resolves the weekday band",
			from: "2026-08-03T04:00:00Z", to: "2026-08-03T05:00:00Z",
			wantBand: domain.BandWeekday, wantPrice: 2000,
		},
		{
			// 06:30-07:30 Tuesday local is 23:30 Monday to 00:30 Tuesday UTC:
			// two UTC calendar days, one local one. The single-calendar-day
			// guard runs on the converted range, so this is a price and not
			// ErrPricingSlotSpansMultipleDays.
			name: "an early local morning slot spanning two UTC days still quotes",
			from: "2026-08-03T23:30:00Z", to: "2026-08-04T00:30:00Z",
			wantBand: domain.BandWeekday, wantPrice: 2000,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			rng := mustRange(t, tt.from, tt.to)
			quote, err := svc.GetQuote(ctx, bookingapp.GetQuoteInput{
				CourtID: seedCourtID, Source: domain.SourceIndividual, Range: rng,
			})
			if err != nil {
				t.Fatalf("GetQuote: %v", err)
			}
			if quote.Band != tt.wantBand {
				t.Errorf("band = %q, want %q", quote.Band, tt.wantBand)
			}
			if quote.PriceCents != tt.wantPrice {
				t.Errorf("price = %d, want %d", quote.PriceCents, tt.wantPrice)
			}
		})
	}

	// Claim 3. A real courts row with facility_id NULL — which is what every
	// court outside a facility looks like, and what both seeded courts looked
	// like before 0034 — priced, and therefore quotable only if a zone can be
	// found. It cannot, so the quote is refused rather than answered in UTC.
	t.Run("a priced court with no facility is refused, not quoted in UTC", func(t *testing.T) {
		detached := uuid.NewString()
		if _, err := pool.Exec(ctx, `INSERT INTO courts (id, name) VALUES ($1, 'Detached Court')`, detached); err != nil {
			t.Fatalf("seeding a facility-less court: %v", err)
		}
		if _, err := pool.Exec(ctx,
			// currency is listed because 0035 made the column NOT NULL with
			// no default (T68.2, #357) — an INSERT omitting it is refused.
			// VND matches what 0035 seeds for the demo court, so this
			// detached court is priced the same way the real one is; the
			// value is irrelevant to the zone claim under test, but a row
			// that could not exist would be.
			`INSERT INTO pricing_rules (court_id, band, weekdays, start_min, end_min, price_cents, currency)
			 VALUES ($1, 'weekday', ARRAY[1,2,3,4,5]::smallint[], 360, 1020, 2000, 'VND')`, detached); err != nil {
			t.Fatalf("seeding pricing for the facility-less court: %v", err)
		}

		rng := mustRange(t, "2026-08-03T04:00:00Z", "2026-08-03T05:00:00Z")
		_, err := svc.GetQuote(ctx, bookingapp.GetQuoteInput{
			CourtID: detached, Source: domain.SourceIndividual, Range: rng,
		})
		if !errors.Is(err, domain.ErrFacilityTimezoneUnset) {
			t.Fatalf("got err %v, want %v — a court with no facility has no zone, and the decision for this "+
				"ticket was an explicit error rather than a silent UTC fallback", err, domain.ErrFacilityTimezoneUnset)
		}
	})

	// The DB half of rule 4 for the column 0034 adds: its CHECK refuses a
	// stored value that could never be an IANA name, so a typo like a quoted
	// or whitespace-padded zone cannot sit in the table waiting to fail at
	// quote time. (Whether a well-shaped name really exists is
	// time.LoadLocation's question, not the schema's — see the migration.)
	t.Run("the timezone column refuses a non-IANA-shaped value", func(t *testing.T) {
		for _, bad := range []string{" ", "Asia/Ho Chi Minh", "'Asia/Ho_Chi_Minh'", "not a zone at all"} {
			_, err := pool.Exec(ctx, `UPDATE facilities SET timezone = $1 WHERE id = $2`, bad, seededDemoFacilityID)
			if err == nil {
				t.Errorf("stored timezone %q, want the facilities_timezone_shape CHECK to refuse it", bad)
			}
		}
		// And the values that must remain storable: the seeded name, a
		// three-component name, one carrying a '+', and the empty "unset"
		// marker 0034's default uses. The last three are the ones a careless
		// CHECK would reject.
		for _, ok := range []string{seededDemoFacilityZone, "America/Argentina/Buenos_Aires", "Etc/GMT+7", ""} {
			if _, err := pool.Exec(ctx, `UPDATE facilities SET timezone = $1 WHERE id = $2`, ok, seededDemoFacilityID); err != nil {
				t.Errorf("timezone %q was refused: %v", ok, err)
			}
		}
		// Leave the row as 0034 seeded it, so a later subtest or a reader of
		// the container sees the migration's own state.
		if _, err := pool.Exec(ctx, `UPDATE facilities SET timezone = $1 WHERE id = $2`, seededDemoFacilityZone, seededDemoFacilityID); err != nil {
			t.Fatalf("restoring the seeded zone: %v", err)
		}
	})

	// A quote whose facility HAS a zone column set to the unset marker is the
	// other half of claim 3, and the one a freshly created Facility has today:
	// nothing writes facilities.timezone yet (ADR-0004's remaining half), so
	// '' is what CreateFacility leaves behind. It must be an error, not UTC.
	t.Run("a priced court whose facility recorded no zone is refused", func(t *testing.T) {
		if _, err := pool.Exec(ctx, `UPDATE facilities SET timezone = '' WHERE id = $1`, seededDemoFacilityID); err != nil {
			t.Fatalf("blanking the seeded zone: %v", err)
		}
		t.Cleanup(func() {
			if _, err := pool.Exec(context.Background(),
				`UPDATE facilities SET timezone = $1 WHERE id = $2`, seededDemoFacilityZone, seededDemoFacilityID); err != nil {
				t.Logf("restoring the seeded zone: %v", err)
			}
		})

		rng := mustRange(t, "2026-08-03T11:00:00Z", "2026-08-03T12:00:00Z")
		_, err := svc.GetQuote(ctx, bookingapp.GetQuoteInput{
			CourtID: seedCourtID, Source: domain.SourceIndividual, Range: rng,
		})
		if !errors.Is(err, domain.ErrFacilityTimezoneUnset) {
			t.Fatalf("got err %v, want %v", err, domain.ErrFacilityTimezoneUnset)
		}
	})
}

// unusedFacilitiesIdentity satisfies the Facilities context's own
// port.IdentityLookup without reaching Identity. This test never resolves a
// subject — it reads courts, facilities and pricing rules — so it fails closed
// rather than pretending a resolution succeeded, matching
// unusedIdentityLookup's reasoning in concurrency_integration_test.go.
type unusedFacilitiesIdentity struct{}

func (unusedFacilitiesIdentity) UserIDBySubject(_ context.Context, _ string) (string, error) {
	return "", errors.New("facilities identity lookup must not be reached by this test")
}
