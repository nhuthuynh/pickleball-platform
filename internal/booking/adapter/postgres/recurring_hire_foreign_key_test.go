package postgres

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/nhuthuynh/white-label/internal/booking/domain"
)

// TestTranslateRecurringHireErr_ForeignKeyViolation is T17.5's unit-level
// half of the regression proof for issue #195's two `recurring_hire_templates`
// rows: a concurrent delete of the referenced Court or User, landing between
// app.Service.RequestRecurringHire's guarding read (FacilityIDForCourt /
// EnsureClubRole) and this repository's INSERT, must surface as the same
// domain sentinel that guard already answers for the non-racing "no such
// parent" case — not as an unclassified 23503 that falls through to
// codes.Internal (the #185 shape, mirrored here on a narrower window per
// #195).
//
// This is deliberately a pure unit test over translateRecurringHireErr with a
// synthetic *pgconn.PgError: it needs no Docker, so it runs in `make
// test-adapters` on every machine. That the *real* Postgres actually raises
// 23503 for these two INSERT columns, with these two constraint names, is a
// separate claim, proved separately against a real database in
// recurring_hire_foreign_key_integration_test.go — #185's testing note
// (repeated by #195) is that no in-memory fixture can prove the FK half, and
// this file cannot either. The two halves together are the proof.
func TestTranslateRecurringHireErr_ForeignKeyViolation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want error
	}{
		{
			name: "23503 on recurring_hire_templates.court_id is the same not-found the app-level guard already answers",
			err:  &pgconn.PgError{Code: pgForeignKeyViolation, ConstraintName: pgRecurringHireTemplateCourtIDFKey},
			want: domain.ErrFacilityNotFound,
		},
		{
			name: "23503 on recurring_hire_templates.requested_by_user_id is the same not-found the app-level guard already answers",
			err:  &pgconn.PgError{Code: pgForeignKeyViolation, ConstraintName: pgRecurringHireTemplateRequestedByUserIDFKey},
			want: domain.ErrUserNotFound,
		},
		{
			name: "23503 wrapped by an intermediate error still translates",
			err:  errors.Join(errors.New("query failed"), &pgconn.PgError{Code: pgForeignKeyViolation, ConstraintName: pgRecurringHireTemplateCourtIDFKey}),
			want: domain.ErrFacilityNotFound,
		},
		{
			name: "no rows is still not found",
			err:  pgx.ErrNoRows,
			want: domain.ErrRecurringHireTemplateNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := translateRecurringHireErr(tt.err)
			if !errors.Is(got, tt.want) {
				t.Fatalf("translateRecurringHireErr(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

// TestTranslateRecurringHireErr_UnrecognizedConstraintNameStaysInternal pins
// the deliberate difference from repository.go's translateErr: that function
// translates a 23503 regardless of constraint name because bookings has only
// one FK, so any name (or none) can only mean one thing. This table has two,
// so a name this adapter does not recognize — including no name at all — must
// NOT guess which parent was missing; it falls through to the wrapped
// Internal default exactly as before this ticket, rather than risk mapping a
// court-reference failure onto ErrUserNotFound or vice versa.
func TestTranslateRecurringHireErr_UnrecognizedConstraintNameStaysInternal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
	}{
		{
			name: "23503 with no constraint name",
			err:  &pgconn.PgError{Code: pgForeignKeyViolation},
		},
		{
			name: "23503 with an unrecognized constraint name",
			err:  &pgconn.PgError{Code: pgForeignKeyViolation, ConstraintName: "some_other_fkey"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := translateRecurringHireErr(tt.err)
			for _, sentinel := range []error{domain.ErrFacilityNotFound, domain.ErrUserNotFound, domain.ErrRecurringHireTemplateNotFound} {
				if errors.Is(got, sentinel) {
					t.Fatalf("translateRecurringHireErr(%v) = %v, must not match %v — an unrecognized constraint name must not guess", tt.err, got, sentinel)
				}
			}
		})
	}
}

// TestTranslateRecurringHireErr_UnrelatedPgErrorStaysInternal pins the other
// half: only 23503 with a recognized constraint name, and pgx.ErrNoRows,
// become domain sentinels. Everything else must keep wrapping through to the
// Internal default, so a genuine server fault is never mis-reported as a
// client error — the inverse of the defect #185/#195 describe, and just as
// wrong.
func TestTranslateRecurringHireErr_UnrelatedPgErrorStaysInternal(t *testing.T) {
	t.Parallel()

	err := translateRecurringHireErr(&pgconn.PgError{Code: "42601"}) // syntax_error
	for _, sentinel := range []error{
		domain.ErrFacilityNotFound,
		domain.ErrUserNotFound,
		domain.ErrRecurringHireTemplateNotFound,
	} {
		if errors.Is(err, sentinel) {
			t.Fatalf("translateRecurringHireErr(42601) = %v, must not match %v", err, sentinel)
		}
	}
}
