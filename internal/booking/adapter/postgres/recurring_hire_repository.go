package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nhuthuynh/white-label/internal/booking/domain"
	bookingdb "github.com/nhuthuynh/white-label/internal/gen/bookingdb"
)

// recurring_hire_templates carries TWO FK-backed references
// (0018_booking_recurring_hire_templates.sql), unlike bookings' single
// court_id FK (repository.go's translateErr): court_id -> courts(id) and
// requested_by_user_id -> identity_users(id). Both are guarded by an
// app-level read before the INSERT — app.Service.RequestRecurringHire calls
// port.FacilityLookup.FacilityIDForCourt then port.IdentityLookup.
// EnsureClubRole (T11.5) — so in the ordinary case a 23503 here can only
// fire on a race: the parent existed when the guard read it and is gone by
// the time this INSERT runs (T17.5, issue #195, mirroring #185/T15.6's
// identical shape for bookings.court_id on a narrower window).
//
// Because there are two FKs sharing one violation code, unlike translateErr's
// single-FK table, the constraint name IS load-bearing here: it is the only
// signal that tells the two apart, and pgconn.PgError does not set
// ColumnName for a foreign_key_violation (only ConstraintName and TableName
// are populated for that SQLSTATE). Both names below are Postgres' default
// for an unnamed inline REFERENCES clause (`<table>_<column>_fkey`) —
// 0018_booking_recurring_hire_templates.sql declares neither FK with an
// explicit CONSTRAINT name, so nothing here renames them out from under this
// mapping. An unrecognized (or absent) constraint name deliberately falls
// through to the wrapped Internal default rather than guessing which parent
// was missing — the same fail-closed-on-the-unknown-case discipline
// translateErr's own "23503 without a constraint name still translates"
// case explains is safe there only because that table has exactly one FK.
const (
	pgRecurringHireTemplateCourtIDFKey           = "recurring_hire_templates_court_id_fkey"
	pgRecurringHireTemplateRequestedByUserIDFKey = "recurring_hire_templates_requested_by_user_id_fkey"
)

// RecurringHireRepository implements port.RecurringHireRepository (T11.5)
// against sqlc-generated queries over the recurring_hire_templates table
// (0018_booking_recurring_hire_templates.sql). It only compiles after `make
// generate` — see the package doc in repository.go.
type RecurringHireRepository struct {
	q *bookingdb.Queries
}

func NewRecurringHireRepository(pool *pgxpool.Pool) *RecurringHireRepository {
	return &RecurringHireRepository{q: bookingdb.New(pool)}
}

func (r *RecurringHireRepository) Create(ctx context.Context, t domain.RecurringHireTemplate) (domain.RecurringHireTemplate, error) {
	row, err := r.q.CreateRecurringHireTemplate(ctx, bookingdb.CreateRecurringHireTemplateParams{
		ID:                      mustUUID(t.ID),
		RequestedByUserID:       mustUUID(t.RequestedByUserID),
		CourtID:                 mustUUID(t.CourtID),
		Weekday:                 int16(t.Weekday),
		StartMin:                int16(t.StartTime),
		EndMin:                  int16(t.EndTime),
		StartsAt:                toTimestamptz(t.StartsAt),
		EndConditionKind:        string(t.EndCondition.Kind),
		EndConditionDate:        recurringEndDateColumn(t.EndCondition),
		EndConditionOccurrences: recurringEndOccurrencesColumn(t.EndCondition),
		Status:                  string(t.Status),
	})
	if err != nil {
		return domain.RecurringHireTemplate{}, translateRecurringHireErr(err)
	}
	return fromRecurringHireFields(row.ID, row.RequestedByUserID, row.CourtID, row.Weekday, row.StartMin,
		row.EndMin, row.StartsAt, row.EndConditionKind, row.EndConditionDate,
		row.EndConditionOccurrences, row.Status), nil
}

func (r *RecurringHireRepository) GetByID(ctx context.Context, id string) (domain.RecurringHireTemplate, error) {
	row, err := r.q.GetRecurringHireTemplateByID(ctx, mustUUID(id))
	if err != nil {
		return domain.RecurringHireTemplate{}, translateRecurringHireErr(err)
	}
	return fromRecurringHireFields(row.ID, row.RequestedByUserID, row.CourtID, row.Weekday, row.StartMin,
		row.EndMin, row.StartsAt, row.EndConditionKind, row.EndConditionDate,
		row.EndConditionOccurrences, row.Status), nil
}

func (r *RecurringHireRepository) UpdateStatus(ctx context.Context, t domain.RecurringHireTemplate) (domain.RecurringHireTemplate, error) {
	row, err := r.q.UpdateRecurringHireTemplateStatus(ctx, bookingdb.UpdateRecurringHireTemplateStatusParams{
		ID:     mustUUID(t.ID),
		Status: string(t.Status),
	})
	if err != nil {
		return domain.RecurringHireTemplate{}, translateRecurringHireErr(err)
	}
	return fromRecurringHireFields(row.ID, row.RequestedByUserID, row.CourtID, row.Weekday, row.StartMin,
		row.EndMin, row.StartsAt, row.EndConditionKind, row.EndConditionDate,
		row.EndConditionOccurrences, row.Status), nil
}

func (r *RecurringHireRepository) ListForCourts(ctx context.Context, courtIDs []string) ([]domain.RecurringHireTemplate, error) {
	// An empty id list short-circuits without a query: `= ANY('{}')` matches
	// nothing anyway, so the round trip would be pure waste, and the port
	// documents this case as an empty result rather than an error.
	if len(courtIDs) == 0 {
		return []domain.RecurringHireTemplate{}, nil
	}

	ids := make([]pgtype.UUID, 0, len(courtIDs))
	for _, id := range courtIDs {
		ids = append(ids, mustUUID(id))
	}

	rows, err := r.q.ListRecurringHireTemplatesForCourts(ctx, ids)
	if err != nil {
		return nil, translateRecurringHireErr(err)
	}

	out := make([]domain.RecurringHireTemplate, 0, len(rows))
	for _, row := range rows {
		out = append(out, fromRecurringHireFields(row.ID, row.RequestedByUserID, row.CourtID, row.Weekday,
			row.StartMin, row.EndMin, row.StartsAt, row.EndConditionKind, row.EndConditionDate,
			row.EndConditionOccurrences, row.Status))
	}
	return out, nil
}

// ListForRequester is the Club-facing "my requests" read (T11.6). It needs no
// empty-input short-circuit the way ListForCourts does — a single id is either
// present or not — but it does need the app layer's uuidShape guard upstream,
// because mustUUID below panics on anything pgtype.UUID.Scan cannot parse.
func (r *RecurringHireRepository) ListForRequester(ctx context.Context, requestedByUserID string) ([]domain.RecurringHireTemplate, error) {
	rows, err := r.q.ListRecurringHireTemplatesForRequester(ctx, mustUUID(requestedByUserID))
	if err != nil {
		return nil, translateRecurringHireErr(err)
	}

	out := make([]domain.RecurringHireTemplate, 0, len(rows))
	for _, row := range rows {
		out = append(out, fromRecurringHireFields(row.ID, row.RequestedByUserID, row.CourtID, row.Weekday,
			row.StartMin, row.EndMin, row.StartsAt, row.EndConditionKind, row.EndConditionDate,
			row.EndConditionOccurrences, row.Status))
	}
	return out, nil
}

// fromRecurringHireFields converts the eleven columns every
// recurring_hire_templates query selects into a domain.RecurringHireTemplate.
// It takes fields rather than a row struct on purpose: sqlc generates a
// distinct Row type per query (CreateRecurringHireTemplateRow,
// GetRecurringHireTemplateByIDRow, ...) instead of one shared table model, so
// a converter written against any one of them would not compile against the
// others — the same CLAUDE.md gotcha fromFields and fromDiscountFields exist
// for.
func fromRecurringHireFields(
	id, requestedByUserID, courtID pgtype.UUID,
	weekday, startMin, endMin int16,
	startsAt pgtype.Timestamptz,
	endConditionKind string,
	endConditionDate pgtype.Timestamptz,
	endConditionOccurrences pgtype.Int4,
	status string,
) domain.RecurringHireTemplate {
	return domain.RecurringHireTemplate{
		ID:                id.String(),
		RequestedByUserID: requestedByUserID.String(),
		CourtID:           courtID.String(),
		Weekday:           time.Weekday(weekday),
		StartTime:         domain.ClockTime(startMin),
		EndTime:           domain.ClockTime(endMin),
		StartsAt:          startsAt.Time,
		EndCondition:      fromRecurringEndConditionColumns(endConditionKind, endConditionDate, endConditionOccurrences),
		Status:            domain.RecurringHireStatus(status),
	}
}

func recurringEndDateColumn(c domain.RecurringHireEndCondition) pgtype.Timestamptz {
	if c.Kind != domain.RecurringHireEndAfterDate {
		return pgtype.Timestamptz{}
	}
	return toTimestamptz(c.Date)
}

func recurringEndOccurrencesColumn(c domain.RecurringHireEndCondition) pgtype.Int4 {
	if c.Kind != domain.RecurringHireEndAfterOccurrences {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: int32(c.Occurrences), Valid: true}
}

// fromRecurringEndConditionColumns rebuilds the RecurringHireEndCondition
// variant from its kind column plus whichever payload column that kind
// carries, going through the domain's own constructors so the Kind/payload
// pairing can only be produced one way.
//
// An unrecognized kind cannot occur (the table CHECKs the three values) but is
// mapped to NoRecurringHireEnd rather than a half-built variant. Note the
// consequence, which is the safe direction here and the reverse of
// fromEndConditionColumns' reasoning for DiscountRule: an unreadable end
// condition makes a template generate occurrences bounded only by the app
// layer's own horizon, never one that silently stops on a date it never had.
// EndRecurringHireAfterOccurrences returns an error for a non-positive count,
// which the CHECK already makes unreachable; a stored zero would degrade to
// NoRecurringHireEnd rather than panicking.
func fromRecurringEndConditionColumns(kind string, date pgtype.Timestamptz, occurrences pgtype.Int4) domain.RecurringHireEndCondition {
	switch domain.RecurringHireEndKind(kind) {
	case domain.RecurringHireEndAfterDate:
		return domain.EndRecurringHireAfterDate(date.Time)
	case domain.RecurringHireEndAfterOccurrences:
		c, err := domain.EndRecurringHireAfterOccurrences(int(occurrences.Int32))
		if err != nil {
			return domain.NoRecurringHireEnd()
		}
		return c
	default:
		return domain.NoRecurringHireEnd()
	}
}

// translateRecurringHireErr is deliberately separate from translateErr and
// translateDiscountErr, for the reason those functions' own comments give:
// translateErr maps pgx.ErrNoRows to domain.ErrBookingNotFound, which is
// Booking-specific and wrong for a template lookup. Here the mapping is to
// domain.ErrRecurringHireTemplateNotFound, which GetByID and UpdateStatus can
// both genuinely return (an UPDATE ... RETURNING that matches no row yields
// pgx.ErrNoRows).
//
// T17.5 (issue #195) adds the two 23503 arms below, on Create only (GetByID
// and UpdateStatus write no FK-backed columns). Both reuse a sentinel this
// context already has, rather than a new one — the caller-visible fact in
// the race case is identical to the one each column's own guarding read
// already answers in the non-racing case (#195's own reasoning, and the
// principle T15.6/#185 established first):
//
//   - court_id: adapter/facilities.Lookup.FacilityIDForCourt already answers
//     domain.ErrFacilityNotFound for "no such Court" — see that method's own
//     doc comment ("Both 'no such Court' and 'a Court belonging to no
//     Facility' ... come back as domain.ErrFacilityNotFound"). This is
//     NOT domain.ErrInvalidCourtReference, the sentinel repository.go's
//     translateErr uses for bookings.court_id's own FK: that sentinel
//     belongs to CreateBooking's call site, whose app-layer guard
//     (uuidShape only) and RPC-level meaning differ from
//     RequestRecurringHire's — a Court naming no Facility is specifically
//     rejected here because approval requires a Facility Owner (see
//     domain/errors.go's ErrFacilityNotFound comment and
//     app/recurring_hire.go's RequestRecurringHire doc comment). One
//     sentinel per concept per call site, not one sentinel per column.
//   - requested_by_user_id: adapter/identity.Lookup.EnsureClubRole (via its
//     shared translate helper) already answers domain.ErrUserNotFound for
//     "no such User" — issue #164's description of this exact call site.
//
// Both sentinels already have a codes.PermissionDenied/NotFound row apiece
// in adapter/grpcapi's toStatus exhaustiveness table (T14.7) from their
// existing raise sites, so this ticket adds no new table row — see that
// table's own comments for why each code was chosen.
func translateRecurringHireErr(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgForeignKeyViolation {
		switch pgErr.ConstraintName {
		case pgRecurringHireTemplateCourtIDFKey:
			return domain.ErrFacilityNotFound
		case pgRecurringHireTemplateRequestedByUserIDFKey:
			return domain.ErrUserNotFound
		}
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrRecurringHireTemplateNotFound
	}
	return fmt.Errorf("recurring hire postgres adapter: %w", err)
}
