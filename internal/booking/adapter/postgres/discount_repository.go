package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nhuthuynh/white-label/internal/booking/domain"
	bookingdb "github.com/nhuthuynh/white-label/internal/gen/bookingdb"
)

// DiscountRuleRepository implements port.DiscountRuleRepository (T11.2)
// against sqlc-generated queries over the discount_rules table
// (0017_booking_discount_rules.sql). It only compiles after `make generate` —
// see the package doc in repository.go.
type DiscountRuleRepository struct {
	q *bookingdb.Queries
}

func NewDiscountRuleRepository(pool *pgxpool.Pool) *DiscountRuleRepository {
	return &DiscountRuleRepository{q: bookingdb.New(pool)}
}

func (r *DiscountRuleRepository) Create(ctx context.Context, rule domain.DiscountRule) (domain.DiscountRule, error) {
	params := bookingdb.CreateDiscountRuleParams{
		ID:                      mustUUID(rule.ID),
		FacilityID:              mustUUID(rule.FacilityID),
		DiscountType:            string(rule.DiscountType),
		Currency:                rule.Amount.Fixed.Currency,
		AppliesTo:               toSourceStrings(rule.AppliesTo),
		StartsAt:                toTimestamptz(rule.StartsAt),
		EndConditionKind:        string(rule.EndCondition.Kind),
		EndConditionDate:        endDateColumn(rule.EndCondition),
		EndConditionOccurrences: endOccurrencesColumn(rule.EndCondition),
	}

	// Only the column belonging to this rule's DiscountType is populated; the
	// other stays NULL. That is not just tidiness — the table's
	// discount_rules_amount_matches_type CHECK rejects a row carrying both,
	// so writing both would fail the insert rather than silently persist a
	// contradictory rule.
	switch rule.DiscountType {
	case domain.DiscountTypePercent:
		params.Percent = pgtype.Float8{Float64: rule.Amount.Percent, Valid: true}
		params.Currency = ""
	case domain.DiscountTypeFixedAmount:
		params.FixedAmountCents = pgtype.Int8{Int64: rule.Amount.Fixed.Cents, Valid: true}
	}

	row, err := r.q.CreateDiscountRule(ctx, params)
	if err != nil {
		return domain.DiscountRule{}, translateDiscountErr(err)
	}
	return fromDiscountFields(row.ID, row.FacilityID, row.DiscountType, row.Percent, row.FixedAmountCents,
		row.Currency, row.AppliesTo, row.StartsAt, row.EndConditionKind, row.EndConditionDate,
		row.EndConditionOccurrences), nil
}

func (r *DiscountRuleRepository) ListForFacility(ctx context.Context, facilityID string) ([]domain.DiscountRule, error) {
	rows, err := r.q.ListDiscountRulesForFacility(ctx, mustUUID(facilityID))
	if err != nil {
		return nil, translateDiscountErr(err)
	}

	out := make([]domain.DiscountRule, 0, len(rows))
	for _, row := range rows {
		out = append(out, fromDiscountFields(row.ID, row.FacilityID, row.DiscountType, row.Percent,
			row.FixedAmountCents, row.Currency, row.AppliesTo, row.StartsAt, row.EndConditionKind,
			row.EndConditionDate, row.EndConditionOccurrences))
	}
	return out, nil
}

// fromDiscountFields converts the eleven columns every discount_rules query
// selects into a domain.DiscountRule. It takes fields rather than a row
// struct on purpose: sqlc generates a distinct Row type per query
// (CreateDiscountRuleRow, ListDiscountRulesForFacilityRow) instead of one
// shared table model, so a converter written against either would not compile
// against the other — the same CLAUDE.md gotcha fromFields exists for in
// repository.go.
func fromDiscountFields(
	id, facilityID pgtype.UUID,
	discountType string,
	percent pgtype.Float8,
	fixedAmountCents pgtype.Int8,
	currency string,
	appliesTo []string,
	startsAt pgtype.Timestamptz,
	endConditionKind string,
	endConditionDate pgtype.Timestamptz,
	endConditionOccurrences pgtype.Int4,
) domain.DiscountRule {
	return domain.DiscountRule{
		ID:           id.String(),
		FacilityID:   facilityID.String(),
		DiscountType: domain.DiscountType(discountType),
		Amount: domain.DiscountAmount{
			Percent: percent.Float64,
			Fixed:   domain.Money{Cents: fixedAmountCents.Int64, Currency: currency},
		},
		AppliesTo:    fromSourceStrings(appliesTo),
		StartsAt:     startsAt.Time,
		EndCondition: fromEndConditionColumns(endConditionKind, endConditionDate, endConditionOccurrences),
	}
}

// toSourceStrings/fromSourceStrings round-trip domain.Source values through
// the applies_to text[] column. No numeric mapping is involved — the stored
// strings ARE the domain values, the same ones bookings.source stores (see
// the migration's own note on why text[] rather than smallint[] or a
// Postgres ENUM).
func toSourceStrings(sources []domain.Source) []string {
	out := make([]string, 0, len(sources))
	for _, s := range sources {
		out = append(out, string(s))
	}
	return out
}

func fromSourceStrings(values []string) []domain.Source {
	out := make([]domain.Source, 0, len(values))
	for _, v := range values {
		out = append(out, domain.Source(v))
	}
	return out
}

func endDateColumn(c domain.EndCondition) pgtype.Timestamptz {
	if c.Kind != domain.EndConditionKindDate {
		return pgtype.Timestamptz{}
	}
	return toTimestamptz(c.Date)
}

func endOccurrencesColumn(c domain.EndCondition) pgtype.Int4 {
	if c.Kind != domain.EndConditionKindOccurrences {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: int32(c.Occurrences), Valid: true}
}

// fromEndConditionColumns rebuilds the EndCondition variant from its kind
// column plus whichever payload column that kind carries, going through the
// domain's own constructors so the Kind/payload pairing can only be produced
// one way. An unrecognized kind cannot occur (the table CHECKs the three
// values) but is mapped to NoEnd rather than a half-built variant, since a
// rule whose end is unreadable must not read as permanently active on a date
// it never had.
func fromEndConditionColumns(kind string, date pgtype.Timestamptz, occurrences pgtype.Int4) domain.EndCondition {
	switch domain.EndConditionKind(kind) {
	case domain.EndConditionKindDate:
		return domain.EndAfterDate(date.Time)
	case domain.EndConditionKindOccurrences:
		return domain.EndAfterOccurrences(int(occurrences.Int32))
	default:
		return domain.NoEnd()
	}
}

// translateDiscountErr is deliberately separate from translateErr in
// repository.go and from translatePricingErr, for the reason that function's
// own comment gives: translateErr maps pgx.ErrNoRows to
// domain.ErrBookingNotFound, which is Booking-specific and wrong for a
// discount lookup. Neither discount query can return pgx.ErrNoRows today
// (Create is an INSERT ... RETURNING that either inserts or fails, and
// ListForFacility is a :many whose no-match is an empty slice), so there is
// no sentinel to map yet — the point is that reusing a Booking-flavored
// translator would silently mismap the moment a `:one` discount read is
// added.
func translateDiscountErr(err error) error {
	return fmt.Errorf("discount postgres adapter: %w", err)
}
