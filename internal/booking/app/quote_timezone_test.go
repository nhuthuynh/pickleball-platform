package app_test

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nhuthuynh/white-label/internal/booking/app"
	"github.com/nhuthuynh/white-label/internal/booking/domain"
)

// fakeZones is this package's port.LocationLoader: the real IANA database
// behind a seam the tests can break on purpose.
//
// It delegates to time.LoadLocation because the arithmetic under test IS the
// zone data — a hand-rolled fixed-offset stub would pass the Ho Chi Minh
// cases while proving nothing about a DST one. The empty-name guard mirrors
// internal/platform/tz.Loader's, for the same reason it exists there:
// time.LoadLocation("") answers UTC with no error, which is how a facility
// with no recorded zone would silently get priced.
//
// It is a local fake rather than an import of internal/platform/tz so this
// app-layer package keeps no dependency on platform code, and so failErr can
// exercise the loader-failed path at all.
type fakeZones struct {
	// failErr, when set, makes every load fail — the "a zone name is
	// recorded and does not load" case (a typo in the data, or a runtime
	// with no tz database).
	failErr error

	// loads counts calls, so a test can witness that GetQuote asks for the
	// zone at all rather than inferring it from a band that happens to be
	// right in both zones.
	loads atomic.Int64
}

func (z *fakeZones) LoadLocation(name string) (*time.Location, error) {
	z.loads.Add(1)
	if z.failErr != nil {
		return nil, z.failErr
	}
	if name == "" {
		return nil, fmt.Errorf("fakeZones: empty name would resolve to UTC")
	}
	return time.LoadLocation(name)
}

// quoteFixtureInZone wires a Service whose one court is priced in LOCAL clock
// time (weekday 06:00-17:00 = 2000, peak 17:00-22:00 = 3500, which is the
// shape of 0004_seed_pricing.sql) and whose Facility sits in
// Asia/Ho_Chi_Minh — +07:00, no DST.
func quoteFixtureInZone(t *testing.T, zone string) (*app.Service, *fakeZones) {
	t.Helper()

	open, err := domain.NewClockTime(6, 0)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	peakStart, err := domain.NewClockTime(17, 0)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	closeAt, err := domain.NewClockTime(22, 0)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	weekdays := []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday}

	pricingRepo := &fakePricingRepo{rulesByCourt: map[string][]domain.PricingRule{
		courtID(1): {
			{ID: "weekday-rule", CourtID: courtID(1), Band: domain.BandWeekday, Weekdays: weekdays, Start: open, End: peakStart, PriceCents: 2000},
			{ID: "peak-rule", CourtID: courtID(1), Band: domain.BandPeak, Weekdays: weekdays, Start: peakStart, End: closeAt, PriceCents: 3500},
		},
	}}
	zones := &fakeZones{}
	svc := app.NewService(app.ServiceOptions{
		Bookings:       newInMemoryRepo(),
		PricingRules:   pricingRepo,
		DiscountRules:  newFakeDiscountRepo(),
		RecurringHires: newFakeRecurringHireRepo(),
		Facilities:     &fakeFacilityLookup{zoneByCourt: map[string]string{courtID(1): zone}},
		Identity:       &fakeIdentityLookup{},
		Zones:          zones,
		IDs:            &sequentialIDs{},
	})
	return svc, zones
}

// TestGetQuote_ResolvesBandInTheFacilitysZone is the app-layer half of
// T68.1 (ADR-0004, closes #356): the zone comes from the FACILITY, through
// port.FacilityLookup, and the band is resolved against it — not against the
// UTC instant adapter/grpcapi hands this layer, and not against the caller's
// own clock.
//
// 18:00 in Ho Chi Minh City is 11:00Z, which is inside the weekday window.
// Before this ticket that is the band this endpoint quoted: 2000 instead of
// 3500, on the one booking a player is most likely to make.
func TestGetQuote_ResolvesBandInTheFacilitysZone(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	// 2026-08-03 is a Monday in both zones, so only the clock is in play.
	peakStart := time.Date(2026, 8, 3, 11, 0, 0, 0, time.UTC)
	peakEnd := time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC)
	callerMinus5 := time.FixedZone("caller-05", -5*3600)

	tests := []struct {
		name      string
		zone      string
		slot      domain.TimeRange
		wantBand  domain.Band
		wantPrice int64
	}{
		{
			name:     "an 18:00 local slot in a +07:00 facility is peak",
			zone:     "Asia/Ho_Chi_Minh",
			slot:     mustTimeRangeAt(t, peakStart, peakEnd),
			wantBand: domain.BandPeak, wantPrice: 3500,
		},
		{
			name:     "the same instant from a -05:00 caller is still peak",
			zone:     "Asia/Ho_Chi_Minh",
			slot:     mustTimeRangeAt(t, peakStart.In(callerMinus5), peakEnd.In(callerMinus5)),
			wantBand: domain.BandPeak, wantPrice: 3500,
		},
		{
			name:     "the same instant for a UTC facility is weekday",
			zone:     "UTC",
			slot:     mustTimeRangeAt(t, peakStart, peakEnd),
			wantBand: domain.BandWeekday, wantPrice: 2000,
		},
		{
			// The local-midnight-adjacent case ADR-0004 names: 06:30-07:30
			// Tuesday in Ho Chi Minh City is 23:30 Monday to 00:30 Tuesday
			// UTC. Two UTC calendar days, one local one — so the
			// single-calendar-day guard has to run after the conversion, or
			// this is ErrPricingSlotSpansMultipleDays instead of a price.
			name: "a local morning slot spanning two UTC days still quotes",
			zone: "Asia/Ho_Chi_Minh",
			slot: mustTimeRangeAt(t,
				time.Date(2026, 8, 3, 23, 30, 0, 0, time.UTC),
				time.Date(2026, 8, 4, 0, 30, 0, 0, time.UTC),
			),
			wantBand: domain.BandWeekday, wantPrice: 2000,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc, zones := quoteFixtureInZone(t, tt.zone)

			quote, err := svc.GetQuote(ctx, app.GetQuoteInput{
				CourtID: courtID(1), Source: domain.SourceIndividual, Range: tt.slot,
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
			// The zone was actually looked up, not inferred. Without this a
			// future refactor could drop the lookup and still pass every
			// UTC-facility row above.
			if got := zones.loads.Load(); got != 1 {
				t.Errorf("LoadLocation calls = %d, want 1", got)
			}
		})
	}
}

// TestGetQuote_DSTTransitionResolvesOnLocalWallClock carries requirement 3
// through the app layer. America/Los_Angeles springs forward on 2026-03-08 at
// 02:00 PST -> 03:00 PDT, so 14:00Z reads 06:00 PST on the 7th and 07:00 PDT
// on the 8th: the same UTC clock, two different local clocks, one on each side
// of the facility's 06:00 opening.
func TestGetQuote_DSTTransitionResolvesOnLocalWallClock(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svc, _ := quoteFixtureInZone(t, "America/Los_Angeles")

	// Saturday 2026-03-07 14:00Z = 06:00 PST, inside the 06:00-17:00 weekday
	// window... except Saturday is not a weekday in this rule set, so use the
	// Monday AFTER the transition for the band assertion and keep the
	// transition itself in the offsets.
	//
	// Monday 2026-03-09 14:00Z = 07:00 PDT (UTC-7). Had the offset stayed at
	// PST (UTC-8), it would read 06:00 — also inside the window, which is why
	// the discriminating assertion below is the 13:00Z one.
	monday0700 := mustTimeRangeAt(t,
		time.Date(2026, 3, 9, 14, 0, 0, 0, time.UTC),
		time.Date(2026, 3, 9, 15, 0, 0, 0, time.UTC),
	)
	if quote, err := svc.GetQuote(ctx, app.GetQuoteInput{
		CourtID: courtID(1), Source: domain.SourceIndividual, Range: monday0700,
	}); err != nil || quote.Band != domain.BandWeekday {
		t.Fatalf("07:00 local Monday: quote %+v, err %v — want the weekday band", quote, err)
	}

	// Monday 2026-03-09 13:00Z is 06:00 PDT — the first hour the court is
	// open. Under PST it would read 05:00, which is BEFORE the 06:00 window
	// opens and would be refused. So this case passes only if the
	// post-transition offset is the one in force.
	monday0600 := mustTimeRangeAt(t,
		time.Date(2026, 3, 9, 13, 0, 0, 0, time.UTC),
		time.Date(2026, 3, 9, 14, 0, 0, 0, time.UTC),
	)
	quote, err := svc.GetQuote(ctx, app.GetQuoteInput{
		CourtID: courtID(1), Source: domain.SourceIndividual, Range: monday0600,
	})
	if err != nil {
		t.Fatalf("06:00 local Monday (PDT): %v — a slot at the facility's opening hour must quote", err)
	}
	if quote.PriceCents != 2000 {
		t.Errorf("price = %d, want 2000", quote.PriceCents)
	}

	// And the hour before opening is still refused, so the test above is not
	// passing because everything passes.
	if _, err := svc.GetQuote(ctx, app.GetQuoteInput{
		CourtID: courtID(1), Source: domain.SourceIndividual,
		Range: mustTimeRangeAt(t,
			time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC), // 05:00 PDT
			time.Date(2026, 3, 9, 13, 0, 0, 0, time.UTC),
		),
	}); !errors.Is(err, domain.ErrNoPricingRule) {
		t.Fatalf("05:00 local: got err %v, want %v", err, domain.ErrNoPricingRule)
	}
}

// TestGetQuote_CourtWithNoZoneIsAnExplicitError is requirement 5: a court
// whose facility has no zone — including a court attached to no facility at
// all — fails with a distinct sentinel rather than being quoted in UTC.
//
// The court in each case HAS pricing rules. That matters: a court nothing
// prices still answers ErrNoPricingRule (the behaviour this endpoint has had
// since T1, pinned by the last case here), and conflating the two would hide
// whichever of them regressed.
func TestGetQuote_CourtWithNoZoneIsAnExplicitError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	slot := mustTimeRangeAt(t,
		time.Date(2026, 8, 3, 11, 0, 0, 0, time.UTC),
		time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC),
	)

	t.Run("a facility with no zone recorded", func(t *testing.T) {
		t.Parallel()
		// The fake reports ErrFacilityTimezoneUnset for an empty entry,
		// exactly as the real adapter does for facilities.timezone = ''.
		svc, zones := quoteFixtureInZone(t, "")
		_, err := svc.GetQuote(ctx, app.GetQuoteInput{
			CourtID: courtID(1), Source: domain.SourceIndividual, Range: slot,
		})
		if !errors.Is(err, domain.ErrFacilityTimezoneUnset) {
			t.Fatalf("got err %v, want %v", err, domain.ErrFacilityTimezoneUnset)
		}
		// Nothing was handed to the loader — in particular not "", which
		// time.LoadLocation would have answered UTC for.
		if got := zones.loads.Load(); got != 0 {
			t.Errorf("LoadLocation calls = %d, want 0 — an unset zone must not reach the loader", got)
		}
	})

	t.Run("a zone that is recorded but does not load", func(t *testing.T) {
		t.Parallel()
		svc, zones := quoteFixtureInZone(t, "Mars/Olympus_Mons")
		_, err := svc.GetQuote(ctx, app.GetQuoteInput{
			CourtID: courtID(1), Source: domain.SourceIndividual, Range: slot,
		})
		if !errors.Is(err, domain.ErrInvalidFacilityTimezone) {
			t.Fatalf("got err %v, want %v", err, domain.ErrInvalidFacilityTimezone)
		}
		if got := zones.loads.Load(); got != 1 {
			t.Errorf("LoadLocation calls = %d, want 1", got)
		}
	})

	t.Run("a loader that fails outright", func(t *testing.T) {
		t.Parallel()
		svc, zones := quoteFixtureInZone(t, "Asia/Ho_Chi_Minh")
		zones.failErr = errors.New("no tz database in this runtime")
		_, err := svc.GetQuote(ctx, app.GetQuoteInput{
			CourtID: courtID(1), Source: domain.SourceIndividual, Range: slot,
		})
		if !errors.Is(err, domain.ErrInvalidFacilityTimezone) {
			t.Fatalf("got err %v, want %v", err, domain.ErrInvalidFacilityTimezone)
		}
		// The platform error is reported but NOT wrapped with %w: nothing on
		// the Booking side may errors.Is against it (CLAUDE.md rule 5).
		if errors.Is(err, zones.failErr) {
			t.Error("the loader's own error type crossed the boundary")
		}
	})

	t.Run("a court with no pricing rules still answers ErrNoPricingRule", func(t *testing.T) {
		t.Parallel()
		// courtID(9) has no rules AND no zone — the shape of every court in
		// this schema that belongs to no Facility. The pre-T68.1 answer must
		// survive: a caller asking about an unpriced court gets told about
		// pricing, not about timezones.
		svc, _ := quoteFixtureInZone(t, "Asia/Ho_Chi_Minh")
		_, err := svc.GetQuote(ctx, app.GetQuoteInput{
			CourtID: courtID(9), Source: domain.SourceIndividual, Range: slot,
		})
		if !errors.Is(err, domain.ErrNoPricingRule) {
			t.Fatalf("got err %v, want %v", err, domain.ErrNoPricingRule)
		}
	})
}

// mustTimeRangeAt is mustTimeRange (quote_test.go) for time.Time fixtures
// rather than RFC3339 strings — needed here because several of these cases
// are about which Location a fixture carries, which a string parse would
// flatten.
func mustTimeRangeAt(t *testing.T, start, end time.Time) domain.TimeRange {
	t.Helper()
	r, err := domain.NewTimeRange(start, end)
	if err != nil {
		t.Fatalf("bad fixture range: %v", err)
	}
	return r
}
