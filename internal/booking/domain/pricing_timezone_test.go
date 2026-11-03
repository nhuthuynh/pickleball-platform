package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/nhuthuynh/white-label/internal/booking/domain"
)

// This file is T68.1's proof of ADR-0004 (closes #356): a price band resolves
// against the wall clock of the FACILITY's zone, not the caller's and not
// UTC.
//
// The defect it pins: ClockTimeOf is t.Hour()*60+t.Minute(), and
// adapter/grpcapi feeds ResolvePrice req.GetStartsAt().AsTime(), which
// protobuf normalises to UTC. An 18:00 Ho Chi Minh City slot therefore
// resolved clock 660 (11:00) and was quoted the weekday band instead of peak
// — the first of the two cases ADR-0004 names.
//
// Every zone here is loaded through time.LoadLocation, i.e. the real IANA
// database, because the arithmetic under test *is* the zone data: a fixed
// +07:00 offset would pass the Ho Chi Minh cases while proving nothing about
// the DST ones. (The loader a real quote goes through is
// port.LocationLoader — see internal/platform/tz, whose own test covers it.)
func mustLoadLocation(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatalf("LoadLocation(%q): %v — this test needs the IANA database (see internal/platform/tz)", name, err)
	}
	return loc
}

// hcmRules is a rule set whose windows are the facility's LOCAL clock, which
// is what a real pricing_rules row is: weekday 06:00-17:00, peak 17:00-22:00,
// weekend 06:00-22:00, plus a 22:00-24:00 late band so the local-midnight
// boundary has something to resolve against. Same shape as weekdayRules in
// pricing_test.go, kept separate so the UTC-fixture tests there and the zone
// tests here cannot drift into each other's expectations.
func hcmRules(t *testing.T) []domain.PricingRule {
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
	weekend := []time.Weekday{time.Saturday, time.Sunday}

	return []domain.PricingRule{
		{ID: "weekday-rule", CourtID: "court-1", Band: domain.BandWeekday, Weekdays: weekdays, Start: open, End: peakStart, PriceCents: 2000},
		{ID: "peak-rule", CourtID: "court-1", Band: domain.BandPeak, Weekdays: weekdays, Start: peakStart, End: closeAt, PriceCents: 3500},
		{ID: "weekend-rule", CourtID: "court-1", Band: domain.BandWeekend, Weekdays: weekend, Start: open, End: closeAt, PriceCents: 3000},
		// End 1440 is 24:00 — a value NewClockTime cannot build and only a
		// DB-sourced rule carries; see clockTimeOfEnd.
		{ID: "late-rule", CourtID: "court-1", Band: domain.BandPeak, Weekdays: weekdays, Start: closeAt, End: domain.ClockTime(1440), PriceCents: 1200},
	}
}

// TestResolvePrice_ResolvesInFacilityZone is the table for requirements 1, 2,
// 4 and 6 of the ticket. Every slot is given as the INSTANTS a client would
// send (UTC, or an arbitrary caller offset), and the expectation is the band
// the facility's own wall clock makes correct.
func TestResolvePrice_ResolvesInFacilityZone(t *testing.T) {
	t.Parallel()

	rules := hcmRules(t)
	hcm := mustLoadLocation(t, "Asia/Ho_Chi_Minh") // +07:00 all year, no DST.
	callerPlus9 := time.FixedZone("caller+09", 9*3600)
	newYork := mustLoadLocation(t, "America/New_York")

	// The ticket's case 1 instant, named once: Monday 2026-08-03 18:00 in Ho
	// Chi Minh City is 11:00Z the same day. 2026-08-03 is a Monday in both
	// zones, so only the CLOCK is in question here, not the weekday.
	peakStartUTC := time.Date(2026, 8, 3, 11, 0, 0, 0, time.UTC)
	peakEndUTC := time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		slot       domain.TimeRange
		loc        *time.Location
		wantRuleID string
		wantPrice  int64
		wantErr    error
	}{
		{
			// ADR-0004's first named case, and the bug in one line: 660
			// minutes (11:00) is inside the weekday window, 1080 (18:00) is
			// inside peak. Before T68.1 this resolved weekday-rule/2000.
			name:       "18:00 local in a +07:00 zone resolves the peak band",
			slot:       mustRangeAt(t, peakStartUTC, peakEndUTC),
			loc:        hcm,
			wantRuleID: "peak-rule", wantPrice: 3500,
		},
		{
			// Requirement 2: the same instant, handed over with a different
			// Location attached. The answer must not move, because the
			// question is about the facility, not the caller's clock.
			name:       "the same instant sent from a +09:00 caller resolves the same band",
			slot:       mustRangeAt(t, peakStartUTC.In(callerPlus9), peakEndUTC.In(callerPlus9)),
			loc:        hcm,
			wantRuleID: "peak-rule", wantPrice: 3500,
		},
		{
			name:       "the same instant sent from New York resolves the same band",
			slot:       mustRangeAt(t, peakStartUTC.In(newYork), peakEndUTC.In(newYork)),
			loc:        hcm,
			wantRuleID: "peak-rule", wantPrice: 3500,
		},
		{
			// The control: that instant genuinely IS in the weekday window
			// for a facility whose zone is UTC. The fix is not "peak always
			// now" — the zone is what decides.
			name:       "the same instant for a UTC facility is still the weekday band",
			slot:       mustRangeAt(t, peakStartUTC, peakEndUTC),
			loc:        time.UTC,
			wantRuleID: "weekday-rule", wantPrice: 2000,
		},
		{
			// Requirement 4, and the local-midnight case ADR-0004 names as
			// already-latent: Monday 00:00-01:00 in Ho Chi Minh City is
			// Sunday 17:00-18:00 UTC. The weekday is wrong in UTC (Sunday ->
			// weekend-rule), and the clock is outside the 06:00 open anyway.
			// Half-open [start, end) means a slot STARTING exactly at local
			// midnight is clock 0, so a window opening at 00:00 contains it.
			name: "a slot starting at local midnight resolves the local day's band",
			slot: mustRangeAt(t,
				time.Date(2026, 8, 2, 17, 0, 0, 0, time.UTC),
				time.Date(2026, 8, 2, 18, 0, 0, 0, time.UTC),
			),
			loc: hcm,
			// Monday local, 00:00-01:00: inside no window of hcmRules (the
			// court opens at 06:00), so no band — but a *weekday* no-match,
			// not the weekend band a UTC reading would have found.
			wantErr: domain.ErrNoPricingRule,
		},
		{
			// Requirement 6: the single-calendar-day check must run on the
			// CONVERTED range. 06:30-07:30 Tuesday local is 23:30 Monday to
			// 00:30 Tuesday UTC — two UTC days, one local day — and before
			// T68.1 fitsSingleCalendarDay refused it outright.
			name: "an early-morning local slot that spans two UTC days still resolves",
			slot: mustRangeAt(t,
				time.Date(2026, 8, 3, 23, 30, 0, 0, time.UTC),
				time.Date(2026, 8, 4, 0, 30, 0, 0, time.UTC),
			),
			loc:        hcm,
			wantRuleID: "weekday-rule", wantPrice: 2000,
		},
		{
			// Requirement 4's other half: a slot ending exactly at local
			// midnight is still one local day (clockTimeOfEnd reads 1440,
			// not 0). 22:00-24:00 Monday local is 15:00-17:00Z — and 17:00Z is
			// already Tuesday 00:00 in Ho Chi Minh City, so the local day is
			// the one this has to be resolved against twice over.
			name: "a slot ending exactly at local midnight resolves the local day's band",
			slot: mustRangeAt(t,
				time.Date(2026, 8, 3, 15, 0, 0, 0, time.UTC),
				time.Date(2026, 8, 3, 17, 0, 0, 0, time.UTC),
			),
			loc:        hcm,
			wantRuleID: "late-rule", wantPrice: 1200,
		},
		{
			// Requirement 5's domain half: no zone is not UTC. The app layer
			// never passes nil (see courtLocation), which is exactly why the
			// domain refuses it rather than trusting that.
			name:    "a nil location is an explicit error, not UTC",
			slot:    mustRangeAt(t, peakStartUTC, peakEndUTC),
			loc:     nil,
			wantErr: domain.ErrFacilityTimezoneUnset,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := domain.ResolvePrice(rules, "court-1", tt.slot, tt.loc)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("got err %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if got.ID != tt.wantRuleID {
				t.Errorf("resolved rule ID = %q, want %q", got.ID, tt.wantRuleID)
			}
			if got.PriceCents != tt.wantPrice {
				t.Errorf("resolved price = %d, want %d", got.PriceCents, tt.wantPrice)
			}
		})
	}
}

// TestResolvePrice_DSTTransitionUsesLocalWallClock is the ticket's
// requirement 3 — the one case it permitted cutting, kept instead.
//
// America/Los_Angeles springs forward on 2026-03-08 at 02:00 PST, which
// becomes 03:00 PDT. So on that Sunday:
//
//	09:00Z = 01:00 PST (UTC-8, before the jump)
//	11:00Z = 04:00 PDT (UTC-7, after it)
//
// A 2-hour slot of elapsed time therefore spans THREE hours of wall clock,
// and that wall clock is what a band window is written in.
//
// # Exactly what the second subtest catches, and what it does not
//
// It catches resolving the END with the START's offset — the natural shape of
// a hand-rolled conversion: at -08:00 the end reads 03:00, which the
// 00:00-03:00 window wrongly contains, so the subtest fails. It does NOT
// catch every fixed-offset stand-in, and the first version of this comment
// claimed it did (review of PR #370). Measured, for the 09:00Z-11:00Z slot
// against window A (00:00-06:00, must cover) and window B (00:00-03:00, must
// not):
//
//	fixed -08:00: local 01:00-03:00 | A covers: true | B covers: true   -> B FAILS
//	fixed -07:00: local 02:00-04:00 | A covers: true | B covers: false  -> both PASS
//
// A single wrong-but-constant offset of -07:00 therefore survives these two
// cases. What rules it out is elsewhere in this file: the Ho Chi Minh table
// (+07:00, same code path) and the fixture assertions below, which pin that
// the real zone reports -0800 at the start and -0700 at the end. Coverage is
// not weaker than it needs to be — the claim was just broader than the cases.
func TestResolvePrice_DSTTransitionUsesLocalWallClock(t *testing.T) {
	t.Parallel()

	la := mustLoadLocation(t, "America/Los_Angeles")
	nightOpen, _ := domain.NewClockTime(0, 0)
	sunday := []time.Weekday{time.Sunday}

	// 01:00 PST -> 04:00 PDT on the spring-forward Sunday.
	slot := mustRangeAt(t,
		time.Date(2026, 3, 8, 9, 0, 0, 0, time.UTC),
		time.Date(2026, 3, 8, 11, 0, 0, 0, time.UTC),
	)

	// Sanity on the fixture itself: if the tz database ever stopped putting
	// the transition here, the two subtests below would still agree with each
	// other and prove nothing.
	if off := slot.Start.In(la).Format("-0700"); off != "-0800" {
		t.Fatalf("fixture start offset = %s, want -0800 (pre-transition PST)", off)
	}
	if off := slot.End.In(la).Format("-0700"); off != "-0700" {
		t.Fatalf("fixture end offset = %s, want -0700 (post-transition PDT)", off)
	}

	t.Run("a window covering the whole local span resolves", func(t *testing.T) {
		t.Parallel()
		until6, _ := domain.NewClockTime(6, 0)
		rules := []domain.PricingRule{{
			ID: "night-rule", CourtID: "court-1", Band: domain.BandWeekend,
			Weekdays: sunday, Start: nightOpen, End: until6, PriceCents: 1500,
		}}
		got, err := domain.ResolvePrice(rules, "court-1", slot, la)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if got.ID != "night-rule" {
			t.Errorf("resolved rule ID = %q, want %q", got.ID, "night-rule")
		}
	})

	t.Run("a window ending at 03:00 local does not cover a slot ending at 04:00 local", func(t *testing.T) {
		t.Parallel()
		until3, _ := domain.NewClockTime(3, 0)
		rules := []domain.PricingRule{{
			ID: "early-rule", CourtID: "court-1", Band: domain.BandWeekend,
			Weekdays: sunday, Start: nightOpen, End: until3, PriceCents: 1500,
		}}
		_, err := domain.ResolvePrice(rules, "court-1", slot, la)
		if !errors.Is(err, domain.ErrNoPricingRule) {
			t.Fatalf("got err %v, want %v — the slot ends at 04:00 local, so a 00:00-03:00 window must not cover it", err, domain.ErrNoPricingRule)
		}
	})
}

// TestResolvePrice_NextLocalMidnightBoundary is the regression guard for the
// arithmetic fitsSingleCalendarDay used to use: local midnight PLUS 24 HOURS.
// A 22:00-24:00 local slot — the exact case clockTimeOfEnd exists to price —
// ends on the next local midnight, and that comparison was wrong at BOTH
// America/Los_Angeles transitions, in opposite directions:
//
//	spring forward (2026-03-08, a 23-hour day): midnight+24h = 01:00 on the 9th,
//	    an hour PAST the boundary the slot ends on
//	fall back      (2026-11-01, a 25-hour day): midnight+24h = 23:00 the SAME day,
//	    an hour SHORT of it
//
// Both rows are cases, not prose: the review of PR #370 found the first
// version of this test pinning only the spring-forward half while the
// production comment claimed a general DST fix. An ordinary day is included
// too, so a "fix" that simply stopped checking the boundary fails here.
func TestResolvePrice_NextLocalMidnightBoundary(t *testing.T) {
	t.Parallel()

	la := mustLoadLocation(t, "America/Los_Angeles")
	lateStart, _ := domain.NewClockTime(22, 0)
	rules := []domain.PricingRule{{
		ID: "late-rule", CourtID: "court-1", Band: domain.BandWeekend,
		Weekdays: []time.Weekday{time.Sunday}, Start: lateStart, End: domain.ClockTime(1440), PriceCents: 4000,
	}}

	tests := []struct {
		name string
		// from/to are the INSTANTS, and wantLocalEnd is the local wall clock
		// they must land on — asserted, so a tz-database change that moved a
		// transition fails loudly here instead of quietly turning the case
		// into an ordinary day.
		from, to     time.Time
		wantLocalEnd string
	}{
		{
			// 2026-03-08 22:00 PDT -> 2026-03-09 00:00 PDT = 05:00Z -> 07:00Z.
			name:         "the night of the spring-forward Sunday",
			from:         time.Date(2026, 3, 9, 5, 0, 0, 0, time.UTC),
			to:           time.Date(2026, 3, 9, 7, 0, 0, 0, time.UTC),
			wantLocalEnd: "2026-03-09 00:00",
		},
		{
			// 2026-11-01 22:00 PST -> 2026-11-02 00:00 PST = 06:00Z -> 08:00Z.
			// PST is UTC-8, in force by 22:00 because the fall back happened
			// that morning at 02:00 PDT.
			name:         "the night of the fall-back Sunday",
			from:         time.Date(2026, 11, 2, 6, 0, 0, 0, time.UTC),
			to:           time.Date(2026, 11, 2, 8, 0, 0, 0, time.UTC),
			wantLocalEnd: "2026-11-02 00:00",
		},
		{
			// 2026-06-07 22:00 PDT -> 2026-06-08 00:00 PDT = 05:00Z -> 07:00Z.
			name:         "an ordinary Sunday night, where the old arithmetic was right",
			from:         time.Date(2026, 6, 8, 5, 0, 0, 0, time.UTC),
			to:           time.Date(2026, 6, 8, 7, 0, 0, 0, time.UTC),
			wantLocalEnd: "2026-06-08 00:00",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			slot := mustRangeAt(t, tt.from, tt.to)
			if got := slot.End.In(la).Format("2006-01-02 15:04"); got != tt.wantLocalEnd {
				t.Fatalf("fixture end = %s local, want %s", got, tt.wantLocalEnd)
			}
			if got := slot.Start.In(la).Hour(); got != 22 {
				t.Fatalf("fixture start = %02d:00 local, want 22:00", got)
			}

			got, err := domain.ResolvePrice(rules, "court-1", slot, la)
			if err != nil {
				t.Fatalf("unexpected err: %v — a slot ending exactly on the next local midnight is one local day", err)
			}
			if got.PriceCents != 4000 {
				t.Errorf("resolved price = %d, want 4000", got.PriceCents)
			}
		})
	}
}

// TestTimeRangeIn_PreservesInstants pins what TimeRange.In is allowed to
// change: the wall clock, and nothing else. A conversion that moved an
// instant would make every band answer right for the wrong slot, and the
// bookings table (and its EXCLUDE constraint) stores instants.
func TestTimeRangeIn_PreservesInstants(t *testing.T) {
	t.Parallel()

	hcm := mustLoadLocation(t, "Asia/Ho_Chi_Minh")
	r := mustRangeAt(t,
		time.Date(2026, 8, 3, 11, 0, 0, 0, time.UTC),
		time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC),
	)
	local := r.In(hcm)

	if !local.Start.Equal(r.Start) || !local.End.Equal(r.End) {
		t.Fatalf("In(hcm) moved an instant: %v -> %v", r, local)
	}
	if local.Duration() != r.Duration() {
		t.Errorf("duration changed: %v -> %v", r.Duration(), local.Duration())
	}
	if h := local.Start.Hour(); h != 18 {
		t.Errorf("local start hour = %d, want 18", h)
	}
}
