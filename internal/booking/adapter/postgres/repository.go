// Package postgres is the Booking context's Postgres adapter. It implements
// port.Repository against sqlc-generated queries and translates Postgres
// errors into domain errors — CLAUDE.md rule 5. It only compiles after
// `make generate` has produced internal/gen/bookingdb (see CLAUDE.md
// gotchas); that package is gitignored and not committed.
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

// Postgres error codes this adapter cares about. 23P01 is the
// bookings.no_overlap-equivalent EXCLUDE constraint firing cleanly (a real
// conflict). 40P01/40001 are deadlock/serialization failures: under
// concurrent inserts contending for the same GiST index page, Postgres can
// abort a transaction with one of these instead of a clean 23P01 — that's
// not a logical conflict, just lock-ordering noise, and retrying resolves it
// (see docs/reviews/04-t4-concurrency-invariant.md's correction and
// docs/LESSONS.md for how this was found: an initial, unverified claim that
// concurrent CreateBooking calls always produce a clean 23P01 turned out to
// be false under real load).
// 23503 is a foreign-key violation: the row being written names a parent row
// that does not exist. On this adapter's writes that means one thing —
// bookings.court_id (db/migrations/0001_init.sql:24) referencing a courts(id)
// that isn't there, i.e. the caller named a court that does not exist. That
// is a client error, and until T15.6 (issue #185) it had no arm in
// translateErr at all, so it fell through to the wrapped default and answered
// codes.Internal: a 500 for a caller's own mistake. Like 23P01 and unlike
// 40P01/40001 it is permanent — retrying an INSERT against a court that does
// not exist cannot make it exist — so it is deliberately absent from
// isRetryableConflict.
const (
	pgExclusionViolation   = "23P01"
	pgForeignKeyViolation  = "23503"
	pgDeadlockDetected     = "40P01"
	pgSerializationFailure = "40001"
)

// createBookingMaxAttempts bounds the deadlock/serialization retry loop.
// Chosen empirically: a 20-way concurrent burst against a cold connection
// pool reproduced deadlocks that cleared within 2 attempts every time this
// was tested locally; 3 leaves one more attempt of margin without turning a
// genuine, persistent failure into a long stall.
const createBookingMaxAttempts = 3

type Repository struct {
	q *bookingdb.Queries
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{q: bookingdb.New(pool)}
}

func (r *Repository) Create(ctx context.Context, b domain.Booking) (domain.Booking, error) {
	params := bookingdb.CreateBookingParams{
		ID:          mustUUID(b.ID),
		CourtID:     mustUUID(b.CourtID),
		Source:      string(b.Source),
		Status:      string(b.Status),
		StartsAt:    toTimestamptz(b.Range.Start),
		EndsAt:      toTimestamptz(b.Range.End),
		ReferenceID: toText(b.ReferenceID),
	}

	var lastErr error
	for attempt := 1; attempt <= createBookingMaxAttempts; attempt++ {
		row, err := r.q.CreateBooking(ctx, params)
		if err == nil {
			return fromFields(row.ID, row.CourtID, row.Source, row.Status, row.StartsAt, row.EndsAt, row.ReferenceID), nil
		}
		lastErr = err
		if attempt == createBookingMaxAttempts || !isRetryableConflict(err) {
			break
		}
		select {
		case <-ctx.Done():
			return domain.Booking{}, ctx.Err()
		case <-time.After(retryBackoff(attempt)):
		}
	}
	return domain.Booking{}, translateErr(lastErr)
}

// isRetryableConflict reports whether err is a transient Postgres
// deadlock/serialization failure worth retrying, as opposed to a real
// exclusion-constraint conflict (23P01) or any other error.
func isRetryableConflict(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	return pgErr.Code == pgDeadlockDetected || pgErr.Code == pgSerializationFailure
}

func retryBackoff(attempt int) time.Duration {
	return time.Duration(attempt) * 5 * time.Millisecond
}

func (r *Repository) ListActiveForCourt(ctx context.Context, courtID string, rng domain.TimeRange) ([]domain.Booking, error) {
	rows, err := r.q.ListActiveForCourt(ctx, bookingdb.ListActiveForCourtParams{
		CourtID: mustUUID(courtID),
		FromTs:  toTimestamptz(rng.Start),
		ToTs:    toTimestamptz(rng.End),
	})
	if err != nil {
		return nil, translateErr(err)
	}
	out := make([]domain.Booking, 0, len(rows))
	for _, row := range rows {
		out = append(out, fromFields(row.ID, row.CourtID, row.Source, row.Status, row.StartsAt, row.EndsAt, row.ReferenceID))
	}
	return out, nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (domain.Booking, error) {
	row, err := r.q.GetBookingByID(ctx, mustUUID(id))
	if err != nil {
		return domain.Booking{}, translateErr(err)
	}
	return fromFields(row.ID, row.CourtID, row.Source, row.Status, row.StartsAt, row.EndsAt, row.ReferenceID), nil
}

func (r *Repository) Update(ctx context.Context, b domain.Booking) (domain.Booking, error) {
	row, err := r.q.UpdateBookingStatus(ctx, bookingdb.UpdateBookingStatusParams{
		ID:     mustUUID(b.ID),
		Status: string(b.Status),
	})
	if err != nil {
		return domain.Booking{}, translateErr(err)
	}
	return fromFields(row.ID, row.CourtID, row.Source, row.Status, row.StartsAt, row.EndsAt, row.ReferenceID), nil
}

// translateErr maps infrastructure failures onto domain errors — the only
// errors allowed to cross out of this package (CLAUDE.md rule 5).
func translateErr(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgExclusionViolation:
			return domain.ErrCourtDoubleBooked
		case pgForeignKeyViolation:
			// T15.6 (issue #185). The caller named a court that does not
			// exist: a well-formed UUID passes app.Service.CreateBooking's
			// uuidShape guard — which is a *shape* check and nothing more —
			// and only the FK on bookings.court_id can tell it from a real
			// court. Without this arm the 23503 fell through to the wrapped
			// default below and answered codes.Internal, reporting a
			// client's own mistake as a server fault.
			//
			// It shares ErrInvalidCourtReference with the app-layer shape
			// guard on purpose; see that sentinel's doc comment in
			// domain/errors.go for why one sentinel covers both halves.
			return domain.ErrInvalidCourtReference
		}
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrBookingNotFound
	}
	return fmt.Errorf("booking postgres adapter: %w", err)
}

// fromFields builds a domain.Booking from the 7 columns every booking query
// selects. sqlc generates a distinct Row struct per query (CreateBookingRow,
// ListActiveForCourtRow, ...) rather than reusing the bookingdb.Booking table
// model, because none of the queries select the generated `during` column —
// see the CLAUDE.md gotcha on why they can't. A shared field-level converter
// avoids duplicating this mapping four times.
func fromFields(id, courtID pgtype.UUID, source, status string, startsAt, endsAt pgtype.Timestamptz, referenceID pgtype.Text) domain.Booking {
	return domain.Booking{
		ID:      id.String(),
		CourtID: courtID.String(),
		Source:  domain.Source(source),
		Status:  domain.Status(status),
		Range: domain.TimeRange{
			Start: startsAt.Time,
			End:   endsAt.Time,
		},
		ReferenceID: referenceID.String,
	}
}

func mustUUID(s string) pgtype.UUID {
	var u pgtype.UUID
	if err := u.Scan(s); err != nil {
		// Booking IDs are generated by port.IDGenerator (adapter/idgen) as
		// UUIDs; a malformed ID here means an upstream invariant was
		// already violated, which is a programmer error, not a runtime one.
		panic(fmt.Sprintf("booking postgres adapter: invalid uuid %q: %v", s, err))
	}
	return u
}

func toTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func toText(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: s != ""}
}
