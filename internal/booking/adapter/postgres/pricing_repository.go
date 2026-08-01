package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nhuthuynh/white-label/internal/booking/domain"
	bookingdb "github.com/nhuthuynh/white-label/internal/gen/bookingdb"
)

// PricingRuleRepository implements port.PricingRuleRepository (HANDOFF.md
// T1) against sqlc-generated queries. It only compiles after `make
// generate` — see the package doc in repository.go.
type PricingRuleRepository struct {
	q *bookingdb.Queries
}

func NewPricingRuleRepository(pool *pgxpool.Pool) *PricingRuleRepository {
	return &PricingRuleRepository{q: bookingdb.New(pool)}
}

func (r *PricingRuleRepository) ListForCourt(ctx context.Context, courtID string) ([]domain.PricingRule, error) {
	rows, err := r.q.ListPricingRulesForCourt(ctx, mustUUID(courtID))
	if err != nil {
		return nil, translatePricingErr(err)
	}

	out := make([]domain.PricingRule, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.PricingRule{
			ID:         row.ID.String(),
			CourtID:    row.CourtID.String(),
			Band:       domain.Band(row.Band),
			Weekdays:   toWeekdays(row.Weekdays),
			Start:      domain.ClockTime(row.StartMin),
			End:        domain.ClockTime(row.EndMin),
			PriceCents: row.PriceCents,
		})
	}
	return out, nil
}

// translatePricingErr is deliberately separate from translateErr in
// repository.go: that function maps pgx.ErrNoRows to
// domain.ErrBookingNotFound, which is Booking-specific and wrong for
// Pricing. ListForCourt is a sqlc `:many` query so it never returns
// pgx.ErrNoRows today (a no-match is an empty slice), but reusing the
// Booking-flavored translator would silently mismap errors the moment a
// `:one` pricing query (e.g. a future GetRuleByID) is added — this keeps
// that failure mode from being possible by construction.
func translatePricingErr(err error) error {
	return fmt.Errorf("pricing postgres adapter: %w", err)
}

func toWeekdays(days []int16) []time.Weekday {
	out := make([]time.Weekday, 0, len(days))
	for _, d := range days {
		out = append(out, time.Weekday(d))
	}
	return out
}
