package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/nhuthuynh/white-label/internal/booking/domain"
)

func validPercentAmount() domain.DiscountAmount {
	return domain.DiscountAmount{Percent: 15}
}

func validFixedAmount(t *testing.T) domain.DiscountAmount {
	t.Helper()
	return domain.DiscountAmount{Fixed: domain.Money{Cents: 500, Currency: "USD"}}
}

// TestNewDiscountRule_Validation exercises every distinct sentinel error the
// ticket calls for, plus the happy path for each DiscountType. Facility-wide
// scope only (A9's narrowing) — FacilityID is just a string reference here,
// not a cross-context lookup (that's T11.2).
func TestNewDiscountRule_Validation(t *testing.T) {
	t.Parallel()

	starts := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name         string
		facilityID   string
		discountType domain.DiscountType
		amount       domain.DiscountAmount
		appliesTo    []domain.Source
		startsAt     time.Time
		endCondition domain.EndCondition
		wantErr      error
	}{
		{
			name:         "valid percent rule",
			facilityID:   "facility-1",
			discountType: domain.DiscountTypePercent,
			amount:       validPercentAmount(),
			appliesTo:    []domain.Source{domain.SourceIndividual},
			startsAt:     starts,
			endCondition: domain.NoEnd(),
			wantErr:      nil,
		},
		{
			name:         "valid fixed_amount rule",
			facilityID:   "facility-1",
			discountType: domain.DiscountTypeFixedAmount,
			amount:       validFixedAmount(t),
			appliesTo:    []domain.Source{domain.SourceRecurringHire},
			startsAt:     starts,
			endCondition: domain.NoEnd(),
			wantErr:      nil,
		},
		{
			name:         "empty facility id is rejected",
			facilityID:   "",
			discountType: domain.DiscountTypePercent,
			amount:       validPercentAmount(),
			appliesTo:    []domain.Source{domain.SourceIndividual},
			startsAt:     starts,
			endCondition: domain.NoEnd(),
			wantErr:      domain.ErrEmptyFacilityID,
		},
		{
			name:         "invalid discount type is rejected",
			facilityID:   "facility-1",
			discountType: domain.DiscountType("bogo"),
			amount:       validPercentAmount(),
			appliesTo:    []domain.Source{domain.SourceIndividual},
			startsAt:     starts,
			endCondition: domain.NoEnd(),
			wantErr:      domain.ErrInvalidDiscountType,
		},
		{
			name:         "percent amount of zero is rejected",
			facilityID:   "facility-1",
			discountType: domain.DiscountTypePercent,
			amount:       domain.DiscountAmount{Percent: 0},
			appliesTo:    []domain.Source{domain.SourceIndividual},
			startsAt:     starts,
			endCondition: domain.NoEnd(),
			wantErr:      domain.ErrInvalidDiscountAmount,
		},
		{
			name:         "percent amount above 100 is rejected",
			facilityID:   "facility-1",
			discountType: domain.DiscountTypePercent,
			amount:       domain.DiscountAmount{Percent: 100.01},
			appliesTo:    []domain.Source{domain.SourceIndividual},
			startsAt:     starts,
			endCondition: domain.NoEnd(),
			wantErr:      domain.ErrInvalidDiscountAmount,
		},
		{
			name:         "percent amount of exactly 100 is accepted",
			facilityID:   "facility-1",
			discountType: domain.DiscountTypePercent,
			amount:       domain.DiscountAmount{Percent: 100},
			appliesTo:    []domain.Source{domain.SourceIndividual},
			startsAt:     starts,
			endCondition: domain.NoEnd(),
			wantErr:      nil,
		},
		{
			name:         "negative percent amount is rejected",
			facilityID:   "facility-1",
			discountType: domain.DiscountTypePercent,
			amount:       domain.DiscountAmount{Percent: -5},
			appliesTo:    []domain.Source{domain.SourceIndividual},
			startsAt:     starts,
			endCondition: domain.NoEnd(),
			wantErr:      domain.ErrInvalidDiscountAmount,
		},
		{
			name:         "zero-value fixed amount is rejected",
			facilityID:   "facility-1",
			discountType: domain.DiscountTypeFixedAmount,
			amount:       domain.DiscountAmount{},
			appliesTo:    []domain.Source{domain.SourceIndividual},
			startsAt:     starts,
			endCondition: domain.NoEnd(),
			wantErr:      domain.ErrInvalidDiscountAmount,
		},
		{
			name:         "negative fixed amount is rejected",
			facilityID:   "facility-1",
			discountType: domain.DiscountTypeFixedAmount,
			amount:       domain.DiscountAmount{Fixed: domain.Money{Cents: -100, Currency: "USD"}},
			appliesTo:    []domain.Source{domain.SourceIndividual},
			startsAt:     starts,
			endCondition: domain.NoEnd(),
			wantErr:      domain.ErrInvalidDiscountAmount,
		},
		{
			name:         "empty appliesTo is rejected",
			facilityID:   "facility-1",
			discountType: domain.DiscountTypePercent,
			amount:       validPercentAmount(),
			appliesTo:    nil,
			startsAt:     starts,
			endCondition: domain.NoEnd(),
			wantErr:      domain.ErrEmptyAppliesTo,
		},
		{
			name:         "appliesTo value outside the four locked sources is rejected",
			facilityID:   "facility-1",
			discountType: domain.DiscountTypePercent,
			amount:       validPercentAmount(),
			appliesTo:    []domain.Source{domain.SourceIndividual, domain.Source("subscription")},
			startsAt:     starts,
			endCondition: domain.NoEnd(),
			wantErr:      domain.ErrInvalidSource,
		},
		{
			name:         "appliesTo with all four locked sources is accepted",
			facilityID:   "facility-1",
			discountType: domain.DiscountTypePercent,
			amount:       validPercentAmount(),
			appliesTo: []domain.Source{
				domain.SourceRecurringHire, domain.SourceIndividual,
				domain.SourceGame, domain.SourceCompetition,
			},
			startsAt:     starts,
			endCondition: domain.NoEnd(),
			wantErr:      nil,
		},
		{
			name:         "EndAfterOccurrences of zero is rejected",
			facilityID:   "facility-1",
			discountType: domain.DiscountTypePercent,
			amount:       validPercentAmount(),
			appliesTo:    []domain.Source{domain.SourceIndividual},
			startsAt:     starts,
			endCondition: domain.EndAfterOccurrences(0),
			wantErr:      domain.ErrInvalidEndConditionOccurrences,
		},
		{
			name:         "EndAfterOccurrences negative is rejected",
			facilityID:   "facility-1",
			discountType: domain.DiscountTypePercent,
			amount:       validPercentAmount(),
			appliesTo:    []domain.Source{domain.SourceIndividual},
			startsAt:     starts,
			endCondition: domain.EndAfterOccurrences(-1),
			wantErr:      domain.ErrInvalidEndConditionOccurrences,
		},
		{
			name:         "EndAfterOccurrences positive is accepted",
			facilityID:   "facility-1",
			discountType: domain.DiscountTypePercent,
			amount:       validPercentAmount(),
			appliesTo:    []domain.Source{domain.SourceIndividual},
			startsAt:     starts,
			endCondition: domain.EndAfterOccurrences(10),
			wantErr:      nil,
		},
		{
			name:         "EndAfterDate is accepted",
			facilityID:   "facility-1",
			discountType: domain.DiscountTypePercent,
			amount:       validPercentAmount(),
			appliesTo:    []domain.Source{domain.SourceIndividual},
			startsAt:     starts,
			endCondition: domain.EndAfterDate(starts.AddDate(0, 1, 0)),
			wantErr:      nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := domain.NewDiscountRule("d1", tt.facilityID, tt.discountType, tt.amount, tt.appliesTo, tt.startsAt, tt.endCondition)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got err %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && got.ID != "d1" {
				t.Errorf("got ID %q, want d1", got.ID)
			}
		})
	}
}

// isZeroDiscountRule reports whether r is the "no discount" zero value
// ResolveDiscount returns on a no-match. DiscountRule embeds a slice
// (AppliesTo), so it isn't comparable with == directly; every rule built via
// NewDiscountRule has a non-empty ID, so ID emptiness is a reliable proxy.
func isZeroDiscountRule(r domain.DiscountRule) bool {
	return r.ID == "" && r.FacilityID == ""
}

func mustDiscountRule(t *testing.T, facilityID string, discountType domain.DiscountType, amount domain.DiscountAmount, appliesTo []domain.Source, startsAt time.Time, endCondition domain.EndCondition) domain.DiscountRule {
	t.Helper()
	r, err := domain.NewDiscountRule("rule-"+facilityID+"-"+string(discountType), facilityID, discountType, amount, appliesTo, startsAt, endCondition)
	if err != nil {
		t.Fatalf("bad fixture discount rule: %v", err)
	}
	return r
}

// TestResolveDiscount_NoMatchIsNotAnError proves the "no discount" case is a
// valid, non-error outcome (the ticket's own framing: a quote with no
// discount is common) distinct from the ambiguous-match error.
func TestResolveDiscount_NoMatchIsNotAnError(t *testing.T) {
	t.Parallel()

	starts := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	queryAt := starts.AddDate(0, 0, 5)

	tests := []struct {
		name  string
		rules []domain.DiscountRule
	}{
		{"no rules at all", nil},
		{
			"rule exists for a different facility",
			[]domain.DiscountRule{
				mustDiscountRule(t, "facility-other", domain.DiscountTypePercent, validPercentAmount(), []domain.Source{domain.SourceIndividual}, starts, domain.NoEnd()),
			},
		},
		{
			"rule exists but source not in appliesTo",
			[]domain.DiscountRule{
				mustDiscountRule(t, "facility-1", domain.DiscountTypePercent, validPercentAmount(), []domain.Source{domain.SourceGame}, starts, domain.NoEnd()),
			},
		},
		{
			"rule has not started yet",
			[]domain.DiscountRule{
				mustDiscountRule(t, "facility-1", domain.DiscountTypePercent, validPercentAmount(), []domain.Source{domain.SourceIndividual}, queryAt.AddDate(0, 0, 1), domain.NoEnd()),
			},
		},
		{
			"rule already ended (EndAfterDate before queryAt)",
			[]domain.DiscountRule{
				mustDiscountRule(t, "facility-1", domain.DiscountTypePercent, validPercentAmount(), []domain.Source{domain.SourceIndividual}, starts, domain.EndAfterDate(starts.AddDate(0, 0, 1))),
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := domain.ResolveDiscount(tt.rules, "facility-1", domain.SourceIndividual, queryAt)
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if !isZeroDiscountRule(got) {
				t.Errorf("got %+v, want zero value", got)
			}
		})
	}
}

func TestResolveDiscount_SingleMatch(t *testing.T) {
	t.Parallel()

	starts := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	at := starts.AddDate(0, 0, 5)

	rule := mustDiscountRule(t, "facility-1", domain.DiscountTypePercent, validPercentAmount(), []domain.Source{domain.SourceIndividual, domain.SourceGame}, starts, domain.NoEnd())
	rules := []domain.DiscountRule{
		mustDiscountRule(t, "facility-other", domain.DiscountTypePercent, validPercentAmount(), []domain.Source{domain.SourceIndividual}, starts, domain.NoEnd()),
		rule,
	}

	got, err := domain.ResolveDiscount(rules, "facility-1", domain.SourceIndividual, at)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got.ID != rule.ID {
		t.Errorf("got rule %q, want %q", got.ID, rule.ID)
	}
}

// TestResolveDiscount_AmbiguousMatchIsRejected reuses ADR-0002's precedent
// (restated for discounts): two rules matching the same
// (FacilityID, Source, time) triple must error, never be silently resolved
// by priority or insertion order.
func TestResolveDiscount_AmbiguousMatchIsRejected(t *testing.T) {
	t.Parallel()

	starts := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	at := starts.AddDate(0, 0, 5)

	ruleA, err := domain.NewDiscountRule("rule-a", "facility-1", domain.DiscountTypePercent, domain.DiscountAmount{Percent: 10}, []domain.Source{domain.SourceIndividual}, starts, domain.NoEnd())
	if err != nil {
		t.Fatalf("bad fixture: %v", err)
	}
	ruleB, err := domain.NewDiscountRule("rule-b", "facility-1", domain.DiscountTypePercent, domain.DiscountAmount{Percent: 20}, []domain.Source{domain.SourceIndividual}, starts, domain.NoEnd())
	if err != nil {
		t.Fatalf("bad fixture: %v", err)
	}

	_, err = domain.ResolveDiscount([]domain.DiscountRule{ruleA, ruleB}, "facility-1", domain.SourceIndividual, at)
	if !errors.Is(err, domain.ErrAmbiguousDiscountRule) {
		t.Fatalf("got err %v, want %v", err, domain.ErrAmbiguousDiscountRule)
	}
}

// TestResolveDiscount_BoundaryTiming pins the inclusive-start / exclusive-end
// semantics for StartsAt and EndAfterDate.
func TestResolveDiscount_BoundaryTiming(t *testing.T) {
	t.Parallel()

	starts := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)

	rule := mustDiscountRule(t, "facility-1", domain.DiscountTypePercent, validPercentAmount(), []domain.Source{domain.SourceIndividual}, starts, domain.EndAfterDate(endDate))
	rules := []domain.DiscountRule{rule}

	tests := []struct {
		name      string
		at        time.Time
		wantMatch bool
	}{
		{"exactly at StartsAt matches (inclusive start)", starts, true},
		{"just before StartsAt does not match", starts.Add(-time.Second), false},
		{"exactly at end date does not match (exclusive end)", endDate, false},
		{"just before end date matches", endDate.Add(-time.Second), true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := domain.ResolveDiscount(rules, "facility-1", domain.SourceIndividual, tt.at)
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			matched := !isZeroDiscountRule(got)
			if matched != tt.wantMatch {
				t.Errorf("matched = %v, want %v", matched, tt.wantMatch)
			}
		})
	}
}

// TestResolveDiscount_EndAfterOccurrencesNotTimeLimited documents T11.1's
// scope boundary: occurrence-based expiry needs usage history this pure
// function isn't given (ResolveDiscount only takes rules/facilityID/source/
// at), so from time alone an EndAfterOccurrences rule behaves like NoEnd.
// Enforcing the occurrence cap itself is an app-layer concern for a later
// ticket, not silently done (or silently skipped) here.
func TestResolveDiscount_EndAfterOccurrencesNotTimeLimited(t *testing.T) {
	t.Parallel()

	starts := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	rule := mustDiscountRule(t, "facility-1", domain.DiscountTypePercent, validPercentAmount(), []domain.Source{domain.SourceIndividual}, starts, domain.EndAfterOccurrences(10))

	got, err := domain.ResolveDiscount([]domain.DiscountRule{rule}, "facility-1", domain.SourceIndividual, starts.AddDate(1, 0, 0))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got.ID != rule.ID {
		t.Errorf("got %+v, want rule %q to still match a year later", got, rule.ID)
	}
}
