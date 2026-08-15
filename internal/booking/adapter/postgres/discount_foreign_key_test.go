package postgres

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/nhuthuynh/white-label/internal/booking/domain"
)

// TestTranslateDiscountErr_ForeignKeyViolation is T17.4's unit-level half of
// the regression proof for issue #195: a facility_id that names no row in
// `facilities` reaches CreateDiscountRule's INSERT — because the Facility
// app.Service.CreateDiscountRule's own EnsureFacilityOwner guard confirmed a
// moment earlier was deleted in the narrow window before the write — and
// fails discount_rules.facility_id's "REFERENCES facilities (id)"
// (db/migrations/0017_booking_discount_rules.sql) as Postgres
// `23503 foreign_key_violation`.
//
// Before this ticket translateDiscountErr had no 23503 arm at all (see its
// own doc comment for why, and for why this file — not
// internal/facilities/adapter/postgres — is where #195's table's
// discount_rules.facility_id row actually lands), so that error fell through
// to the wrapped default and would have answered codes.Internal — a 500 for
// what is, at the moment it happens, the caller-visible fact "no such
// Facility". Mirrors foreign_key_test.go's identical proof for
// bookings.court_id (T15.6, issue #185) one row of #195's own table over,
// and internal/facilities/adapter/postgres/foreign_key_test.go's twin proof
// for courts.facility_id/facility_camera_links.facility_id.
//
// Deliberately a pure unit test over translateDiscountErr with a synthetic
// *pgconn.PgError: no Docker needed, runs in `make test-adapters` on every
// machine. That the *real* Postgres actually raises 23503 for this INSERT,
// specifically under the guard-then-delete race, is proved separately
// against a real database in
// discount_rule_facility_race_integration_test.go — no in-memory fixture can
// prove that half (the same trap #185/#195's own testing notes name twice).
func TestTranslateDiscountErr_ForeignKeyViolation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
	}{
		{
			name: "23503 on the discount_rules.facility_id FK is an unresolvable facility reference",
			err:  &pgconn.PgError{Code: pgForeignKeyViolation, ConstraintName: "discount_rules_facility_id_fkey"},
		},
		{
			// The constraint name is deliberately not part of the match —
			// mirrors foreign_key_test.go's identical case and reasoning: a
			// migration renaming the constraint must not silently fall this
			// mapping back to Internal.
			name: "23503 without a constraint name still translates",
			err:  &pgconn.PgError{Code: pgForeignKeyViolation},
		},
		{
			name: "23503 wrapped by an intermediate error still translates",
			err:  wrapPgErr(&pgconn.PgError{Code: pgForeignKeyViolation}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := translateDiscountErr(tt.err)
			if !errors.Is(got, domain.ErrFacilityNotFound) {
				t.Fatalf("translateDiscountErr(%v) = %v, want domain.ErrFacilityNotFound", tt.err, got)
			}
		})
	}
}

// TestTranslateDiscountErr_UnrelatedPgCodeStaysWrapped pins the other half:
// only 23503 becomes ErrFacilityNotFound. Everything else must keep wrapping
// through to the Internal default, so a genuine server fault is never
// mis-reported as a client error — the inverse of the defect #195 describes,
// and just as wrong.
func TestTranslateDiscountErr_UnrelatedPgCodeStaysWrapped(t *testing.T) {
	t.Parallel()

	got := translateDiscountErr(&pgconn.PgError{Code: "42601"}) // syntax_error
	if errors.Is(got, domain.ErrFacilityNotFound) {
		t.Fatalf("translateDiscountErr(42601) = %v, must not match domain.ErrFacilityNotFound", got)
	}

	nonPg := errors.New("boom")
	if got := translateDiscountErr(nonPg); errors.Is(got, domain.ErrFacilityNotFound) {
		t.Fatalf("translateDiscountErr(%v) = %v, an unrelated error must not be mismapped", nonPg, got)
	}
}
