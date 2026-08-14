package postgres

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/nhuthuynh/white-label/internal/booking/domain"
)

// TestDiscountColumnRoundTrip proves the T11.2 adapter's column mapping is
// lossless for every shape a DiscountRule can take, without a database:
// build the column values the way Create does, feed them back through
// fromDiscountFields the way the read path does, and require the original
// rule back.
//
// This is the layer where a discount silently becomes the wrong discount —
// a percent rule read back as a fixed_amount one, or an EndAfterOccurrences
// rule read back as NoEnd — and none of it is covered by `make test-domain`
// (this package needs internal/gen). Mirrors
// internal/facilities/adapter/postgres/translate_test.go's DB-free approach.
func TestDiscountColumnRoundTrip(t *testing.T) {
	t.Parallel()

	starts := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	ends := time.Date(2026, 4, 1, 9, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		rule domain.DiscountRule
	}{
		{
			name: "percent rule with no end",
			rule: domain.DiscountRule{
				DiscountType: domain.DiscountTypePercent,
				Amount:       domain.DiscountAmount{Percent: 12.5},
				AppliesTo:    []domain.Source{domain.SourceIndividual, domain.SourceGame},
				StartsAt:     starts,
				EndCondition: domain.NoEnd(),
			},
		},
		{
			name: "fixed-amount rule ending on a date",
			rule: domain.DiscountRule{
				DiscountType: domain.DiscountTypeFixedAmount,
				Amount:       domain.DiscountAmount{Fixed: domain.Money{Cents: 750, Currency: "USD"}},
				AppliesTo:    []domain.Source{domain.SourceRecurringHire},
				StartsAt:     starts,
				EndCondition: domain.EndAfterDate(ends),
			},
		},
		{
			name: "percent rule ending after occurrences",
			rule: domain.DiscountRule{
				DiscountType: domain.DiscountTypePercent,
				Amount:       domain.DiscountAmount{Percent: 100},
				AppliesTo:    []domain.Source{domain.SourceCompetition},
				StartsAt:     starts,
				EndCondition: domain.EndAfterOccurrences(5),
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Exactly what Create writes into the columns.
			var percent pgtype.Float8
			var fixed pgtype.Int8
			currency := tt.rule.Amount.Fixed.Currency
			switch tt.rule.DiscountType {
			case domain.DiscountTypePercent:
				percent = pgtype.Float8{Float64: tt.rule.Amount.Percent, Valid: true}
				currency = ""
			case domain.DiscountTypeFixedAmount:
				fixed = pgtype.Int8{Int64: tt.rule.Amount.Fixed.Cents, Valid: true}
			}

			got := fromDiscountFields(
				mustUUID("00000000-0000-4000-a000-000000000001"),
				mustUUID("00000000-0000-4000-c000-000000000001"),
				string(tt.rule.DiscountType),
				percent,
				fixed,
				currency,
				toSourceStrings(tt.rule.AppliesTo),
				toTimestamptz(tt.rule.StartsAt),
				string(tt.rule.EndCondition.Kind),
				endDateColumn(tt.rule.EndCondition),
				endOccurrencesColumn(tt.rule.EndCondition),
			)

			if got.DiscountType != tt.rule.DiscountType {
				t.Errorf("DiscountType = %q, want %q", got.DiscountType, tt.rule.DiscountType)
			}
			if !got.StartsAt.Equal(tt.rule.StartsAt) {
				t.Errorf("StartsAt = %v, want %v", got.StartsAt, tt.rule.StartsAt)
			}
			if len(got.AppliesTo) != len(tt.rule.AppliesTo) {
				t.Fatalf("AppliesTo = %v, want %v", got.AppliesTo, tt.rule.AppliesTo)
			}
			for i := range got.AppliesTo {
				if got.AppliesTo[i] != tt.rule.AppliesTo[i] {
					t.Errorf("AppliesTo[%d] = %q, want %q", i, got.AppliesTo[i], tt.rule.AppliesTo[i])
				}
			}
			if got.EndCondition.Kind != tt.rule.EndCondition.Kind {
				t.Errorf("EndCondition.Kind = %q, want %q", got.EndCondition.Kind, tt.rule.EndCondition.Kind)
			}
			if !got.EndCondition.Date.Equal(tt.rule.EndCondition.Date) {
				t.Errorf("EndCondition.Date = %v, want %v", got.EndCondition.Date, tt.rule.EndCondition.Date)
			}
			if got.EndCondition.Occurrences != tt.rule.EndCondition.Occurrences {
				t.Errorf("EndCondition.Occurrences = %d, want %d", got.EndCondition.Occurrences, tt.rule.EndCondition.Occurrences)
			}

			// The price a rule produces is the only thing that ultimately
			// matters, so assert on that too rather than on the amount
			// fields alone: a percent rule read back with its cents field
			// populated (or vice versa) would still pass field equality on
			// the half being checked.
			if got.ApplyToCents(2000) != tt.rule.ApplyToCents(2000) {
				t.Errorf("round-tripped rule prices 2000 as %d, want %d",
					got.ApplyToCents(2000), tt.rule.ApplyToCents(2000))
			}
		})
	}
}
