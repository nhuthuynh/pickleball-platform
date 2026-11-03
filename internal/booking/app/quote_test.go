package app_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nhuthuynh/white-label/internal/booking/app"
	"github.com/nhuthuynh/white-label/internal/booking/domain"
)

// fakePricingRepo is an in-memory port.PricingRuleRepository, scoped by
// court like the real Postgres adapter will be.
type fakePricingRepo struct {
	rulesByCourt map[string][]domain.PricingRule

	// listForCourtCalls counts real invocations of ListForCourt — see
	// inMemoryRepo.listActiveForCourtCalls in service_test.go for why a
	// malformed-shape guard can only be proven this way against an in-memory
	// fake. See TestGetQuote_MalformedCourtIDNeverReachesRepository in
	// malformed_id_test.go. atomic.Int64, not a plain int: this fake is
	// shared across parallel subtests in TestGetQuote_ResolvesPriceForSlot
	// (one fakePricingRepo, many t.Parallel() subtests all calling
	// GetQuote concurrently), and a plain int++ raced under -race the
	// moment this field existed.
	listForCourtCalls atomic.Int64
}

func (r *fakePricingRepo) ListForCourt(_ context.Context, courtID string) ([]domain.PricingRule, error) {
	r.listForCourtCalls.Add(1)
	return r.rulesByCourt[courtID], nil
}

// weekdayPricingRules's Currency is "USD" (T68.2, ADR-0005, #357 — the field
// is new, every rule must name one, and ResolvePrice refuses a rule set that
// does not). USD rather than the seeded VND on purpose: it is what every other
// money fixture in this context already uses (discount_test.go,
// discount_apply_test.go, the postgres translate tests), and the one thing a
// fixture here must not do is disagree with the discount fixtures applied to
// it — a band in one currency and a fixed-amount discount in another is now a
// refused quote, which would fail these tests for a reason that has nothing
// to do with what they assert.
//
// The real seeded value is VND (0035, matching 0034's Asia/Ho_Chi_Minh demo
// facility) and it is asserted where it is real: the Postgres integration
// test. Tests that are ABOUT the currency parameterise it rather than
// inheriting a default — see quote_currency_test.go.
func weekdayPricingRules() []domain.PricingRule {
	weekdayStart, _ := domain.NewClockTime(6, 0)
	weekdayEnd, _ := domain.NewClockTime(17, 0)
	peakEnd, _ := domain.NewClockTime(22, 0)
	weekdays := []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday}

	return []domain.PricingRule{
		{ID: "weekday-rule", CourtID: courtID(1), Band: domain.BandWeekday, Weekdays: weekdays, Start: weekdayStart, End: weekdayEnd, PriceCents: 2000, Currency: "USD"},
		{ID: "peak-rule", CourtID: courtID(1), Band: domain.BandPeak, Weekdays: weekdays, Start: weekdayEnd, End: peakEnd, PriceCents: 3500, Currency: "USD"},
	}
}

// TestGetQuote_ResolvesPriceForSlot is the app-layer proof (HANDOFF.md T1 AC:
// "table-driven quote tests pass") that GetQuote wires domain.ResolvePrice
// to a real repository round trip.
//
// Court IDs here are UUID-shaped (via the courtID helper in
// malformed_id_test.go), not the old "court-1"/"court-9" fixture shape — the
// TestGetQuote_MalformedCourtIDIsNoPricingRuleNotAPanic guard added alongside
// this fix rejects non-UUID Court IDs before they reach ResolvePrice, so a
// non-UUID fixture here would fail closed exactly like the real crash this
// guard exists to prevent, for reasons that have nothing to do with what
// this test is actually proving.
func TestGetQuote_ResolvesPriceForSlot(t *testing.T) {
	t.Parallel()

	pricingRepo := &fakePricingRepo{rulesByCourt: map[string][]domain.PricingRule{
		courtID(1): weekdayPricingRules(),
	}}
	svc := app.NewService(app.ServiceOptions{
		Bookings:       newInMemoryRepo(),
		PricingRules:   pricingRepo,
		DiscountRules:  newFakeDiscountRepo(),
		RecurringHires: newFakeRecurringHireRepo(),
		// courtID(1) has a Facility, and that Facility has a zone — a world
		// that can exist. A bare &fakeFacilityLookup{} would give this court
		// no Facility while the fake's nil-map default still reported UTC for
		// it, which production refuses (review finding F4 of PR #370).
		// courtID(9), the unpriced court below, is deliberately NOT in this
		// lookup: it answers ErrNoPricingRule before any zone is needed.
		Facilities: zonedLookupFor(courtID(1)),
		Identity:   &fakeIdentityLookup{},
		Zones:      &fakeZones{},
		IDs:        &sequentialIDs{},
	})
	ctx := context.Background()

	tests := []struct {
		name      string
		courtID   string
		slot      domain.TimeRange
		wantPrice int64
		wantErr   error
	}{
		{
			name:      "weekday morning resolves the weekday band",
			courtID:   courtID(1),
			slot:      mustTimeRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z"), // 2026-08-03 is a Monday
			wantPrice: 2000,
		},
		{
			name:      "weekday evening resolves the peak band",
			courtID:   courtID(1),
			slot:      mustTimeRange(t, "2026-08-03T18:00:00Z", "2026-08-03T19:00:00Z"),
			wantPrice: 3500,
		},
		{
			name:    "court with no pricing rules returns a clear error",
			courtID: courtID(9),
			slot:    mustTimeRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z"),
			wantErr: domain.ErrNoPricingRule,
		},
		{
			name:    "slot outside every rule window returns a clear error",
			courtID: courtID(1),
			slot:    mustTimeRange(t, "2026-08-08T09:00:00Z", "2026-08-08T10:00:00Z"), // Saturday: no weekend rule configured
			wantErr: domain.ErrNoPricingRule,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			quote, err := svc.GetQuote(ctx, app.GetQuoteInput{
				CourtID: tt.courtID, Source: domain.SourceIndividual, Range: tt.slot,
			})
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("got err %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if quote.PriceCents != tt.wantPrice {
				t.Errorf("price = %d, want %d", quote.PriceCents, tt.wantPrice)
			}
		})
	}
}
