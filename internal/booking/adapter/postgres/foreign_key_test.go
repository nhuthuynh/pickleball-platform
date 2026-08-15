package postgres

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/nhuthuynh/white-label/internal/booking/domain"
)

// TestTranslateErr_ForeignKeyViolation is T15.6's unit-level half of the
// regression proof for issue #185: a well-formed but *unknown* court id — a
// syntactically valid UUID naming no row in `courts` — passes
// app.Service.CreateBooking's shape guard, reaches the INSERT, and fails
// `bookings.court_id uuid NOT NULL REFERENCES courts (id)`
// (db/migrations/0001_init.sql:24) as Postgres `23503 foreign_key_violation`.
//
// Before this ticket translateErr had no 23503 arm at all, so that error fell
// through to the `fmt.Errorf("booking postgres adapter: %w", err)` default,
// stayed unclassified all the way to adapter/grpcapi's toStatus default, and
// answered codes.Internal — a 500 for what is unambiguously the caller naming
// something that does not exist. CLAUDE.md rule 5 requires the adapter to
// translate infra errors into domain errors; the 23P01 -> ErrCourtDoubleBooked
// arm this file sits beside is the pattern being followed.
//
// This test is deliberately a pure unit test over translateErr with a
// synthetic *pgconn.PgError: it needs no Docker, so it runs in
// `make test-adapters` on every machine. That the *real* Postgres actually
// raises 23503 for this INSERT is a separate claim, proved separately against
// a real database in unknown_court_integration_test.go — issue #185's testing
// note is that no in-memory fixture can prove the FK half, and this file
// cannot either. The two halves together are the proof.
func TestTranslateErr_ForeignKeyViolation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want error
	}{
		{
			name: "23503 on the bookings.court_id FK is an unusable court reference",
			err:  &pgconn.PgError{Code: pgForeignKeyViolation, ConstraintName: "bookings_court_id_fkey"},
			want: domain.ErrInvalidCourtReference,
		},
		{
			// The constraint name is deliberately not part of the match.
			// Postgres names FKs from the table/column by default, but a
			// migration is free to name one differently, and every
			// INSERT/UPDATE this adapter issues writes bookings,
			// pricing_rules or recurring_hire_templates rows whose court FK
			// is the court_id -> courts(id) reference. Matching on a name
			// string would make the mapping silently fall back to Internal
			// the day someone renames the constraint — the exact
			// fail-quietly shape this ticket is removing.
			name: "23503 without a constraint name still translates",
			err:  &pgconn.PgError{Code: pgForeignKeyViolation},
			want: domain.ErrInvalidCourtReference,
		},
		{
			name: "23503 wrapped by an intermediate error still translates",
			err:  wrapPgErr(&pgconn.PgError{Code: pgForeignKeyViolation}),
			want: domain.ErrInvalidCourtReference,
		},
		{
			// Regression guard on the arms that already existed: adding the
			// 23503 arm must not change what 23P01 or ErrNoRows answer.
			name: "23P01 is still a double booking, not a bad reference",
			err:  &pgconn.PgError{Code: pgExclusionViolation},
			want: domain.ErrCourtDoubleBooked,
		},
		{
			name: "no rows is still not found",
			err:  pgx.ErrNoRows,
			want: domain.ErrBookingNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := translateErr(tt.err)
			if !errors.Is(got, tt.want) {
				t.Fatalf("translateErr(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

// TestTranslateErr_UnrelatedPgErrorStaysInternal pins the other half: only
// the codes this adapter classifies become domain sentinels. Everything else
// must keep wrapping through to the Internal default, so a genuine server
// fault is never mis-reported as a client error — the inverse of the defect
// #185 describes, and just as wrong.
func TestTranslateErr_UnrelatedPgErrorStaysInternal(t *testing.T) {
	t.Parallel()

	err := translateErr(&pgconn.PgError{Code: "42601"}) // syntax_error
	for _, sentinel := range []error{
		domain.ErrInvalidCourtReference,
		domain.ErrCourtDoubleBooked,
		domain.ErrBookingNotFound,
	} {
		if errors.Is(err, sentinel) {
			t.Fatalf("translateErr(42601) = %v, must not match %v", err, sentinel)
		}
	}
}

// TestForeignKeyViolationIsNotRetryable guards the interaction between this
// ticket's new arm and T4's retry loop: 23503 is a permanent, caller-caused
// failure, so Repository.Create must translate it on the first attempt rather
// than spending its deadlock budget re-issuing an INSERT that cannot ever
// succeed.
func TestForeignKeyViolationIsNotRetryable(t *testing.T) {
	t.Parallel()

	if isRetryableConflict(&pgconn.PgError{Code: pgForeignKeyViolation}) {
		t.Fatal("isRetryableConflict(23503) = true, want false — an unknown court never becomes known by retrying")
	}
}

// wrapPgErr hides a pg error one level down, the way a query helper might, so
// the test proves errors.As-based unwrapping rather than a type assertion.
func wrapPgErr(err error) error {
	return errors.Join(errors.New("query failed"), err)
}
