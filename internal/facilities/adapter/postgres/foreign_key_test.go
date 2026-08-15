package postgres

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/nhuthuynh/white-label/internal/facilities/domain"
)

// TestTranslateErr_ForeignKeyViolation is T17.4's unit-level half of the
// regression proof for issue #195: a facility_id that names no row in
// `facilities` reaches AddCourt or AddCameraLink's INSERT — because the
// Facility that app.Service's own GetFacilityByID guard confirmed a moment
// earlier was deleted in the narrow window before the write — and fails
// courts.facility_id or facility_camera_links.facility_id's
// "REFERENCES facilities (id)" constraint (db/migrations/0010_facilities.sql)
// as Postgres `23503 foreign_key_violation`.
//
// Before this ticket translateErr had no 23503 arm at all, so that error fell
// through to the `fmt.Errorf("facilities postgres adapter: %w", err)` default,
// stayed unclassified all the way to adapter/grpcapi's toStatus default, and
// answered codes.Internal — a 500 for what is, at the moment it happens, the
// caller-visible fact "no such Facility". CLAUDE.md rule 5 requires the
// adapter to translate infra errors into domain errors; this mirrors
// internal/booking/adapter/postgres/foreign_key_test.go's identical proof for
// bookings.court_id (T15.6, issue #185), one row of #195's own table over.
//
// This test is deliberately a pure unit test over translateErr with a
// synthetic *pgconn.PgError: it needs no Docker, so it runs in
// `make test-adapters` on every machine. That the *real* Postgres actually
// raises 23503 for these two INSERTs, and specifically under the guard-then-
// delete race rather than merely "the id was never valid", is a separate
// claim proved separately against a real database in
// facility_deleted_race_integration_test.go — #195's own body names the same
// fixture-infidelity trap #185's testing note did: no in-memory fixture can
// prove the FK half. The two halves together are the proof.
func TestTranslateErr_ForeignKeyViolation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
	}{
		{
			name: "23503 on the courts.facility_id FK is an unresolvable facility reference",
			err:  &pgconn.PgError{Code: pgForeignKeyViolation, ConstraintName: "courts_facility_id_fkey"},
		},
		{
			name: "23503 on the facility_camera_links.facility_id FK is an unresolvable facility reference",
			err:  &pgconn.PgError{Code: pgForeignKeyViolation, ConstraintName: "facility_camera_links_facility_id_fkey"},
		},
		{
			// The constraint name is deliberately not part of the match —
			// see pgForeignKeyViolation's doc comment in repository.go for
			// why: a migration renaming either constraint must not silently
			// fall this mapping back to Internal.
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
			got := translateErr(tt.err)
			if !errors.Is(got, domain.ErrFacilityNotFound) {
				t.Fatalf("translateErr(%v) = %v, want domain.ErrFacilityNotFound", tt.err, got)
			}
		})
	}
}

// TestTranslateErr_UnrelatedPgCodeStaysWrapped pins the other half: only
// 23503 (and pgx.ErrNoRows, already covered by TestTranslateErr in
// translate_test.go) becomes ErrFacilityNotFound. Everything else must keep
// wrapping through to the Internal default, so a genuine server fault is
// never mis-reported as a client error — the inverse of the defect #195
// describes, and just as wrong. Mirrors
// internal/booking/adapter/postgres/foreign_key_test.go's
// TestTranslateErr_UnrelatedPgErrorStaysInternal.
func TestTranslateErr_UnrelatedPgCodeStaysWrapped(t *testing.T) {
	t.Parallel()

	got := translateErr(&pgconn.PgError{Code: "42601"}) // syntax_error
	if errors.Is(got, domain.ErrFacilityNotFound) {
		t.Fatalf("translateErr(42601) = %v, must not match domain.ErrFacilityNotFound", got)
	}

	// Regression guard: adding the 23503 arm must not change what
	// pgx.ErrNoRows answers (it already maps to ErrFacilityNotFound too, via
	// the separate arm TestTranslateErr in translate_test.go pins — this
	// just proves the two arms don't interfere with each other).
	if got := translateErr(pgx.ErrNoRows); !errors.Is(got, domain.ErrFacilityNotFound) {
		t.Fatalf("translateErr(pgx.ErrNoRows) = %v, want domain.ErrFacilityNotFound (regression)", got)
	}
}

// wrapPgErr hides a pg error one level down, the way a query helper might, so
// the test proves errors.As-based unwrapping rather than a type assertion.
// Mirrors internal/booking/adapter/postgres/foreign_key_test.go's identical
// helper.
func wrapPgErr(err error) error {
	return errors.Join(errors.New("query failed"), err)
}
