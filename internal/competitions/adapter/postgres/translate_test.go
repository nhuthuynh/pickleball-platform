package postgres

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/nhuthuynh/white-label/internal/competitions/domain"
)

// TestTranslateEntryErr proves the DB-level half of CLAUDE.md rule 4
// (invariants enforced in Postgres AND expressed in the domain): a 23505
// unique_violation on competition_entries_active_player_idx (db/migrations/
// 0014_competitions.sql) must translate to domain.ErrAlreadyEntered, and a
// P0001 raised by the competition_entries_capacity_guard trigger must
// translate to domain.ErrCompetitionFull — neither may leak as a raw
// pgconn.PgError, per the adapter boundary CLAUDE.md rule 5 requires.
//
// T10.6 (closes #96) is what gives the pgx.ErrNoRows case a real caller for
// the first time (GetEntryByID/UpdateEntryPaymentStatus): every entry-level
// query before it was either an insert or a :many list, neither of which
// can return pgx.ErrNoRows, so this exact mapping had never been exercised.
// It must be domain.ErrCompetitionEntryNotFound, not
// domain.ErrCompetitionNotFound (the Competition-level sentinel a careless
// copy-paste from translateErr would have produced) — mirrors
// internal/socialplay/adapter/postgres/translate_test.go's
// TestTranslateRegistrationErr exactly. Runs without internal/gen or a real
// database.
func TestTranslateEntryErr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		err     error
		wantErr error
	}{
		{"unique violation (23505) becomes ErrAlreadyEntered", &pgconn.PgError{Code: "23505"}, domain.ErrAlreadyEntered},
		{"capacity guard trigger (P0001) becomes ErrCompetitionFull", &pgconn.PgError{Code: "P0001"}, domain.ErrCompetitionFull},
		{"no rows becomes ErrCompetitionEntryNotFound, not ErrCompetitionNotFound", pgx.ErrNoRows, domain.ErrCompetitionEntryNotFound},
		{"unrelated pg error is wrapped, not silently mapped", &pgconn.PgError{Code: "42601"}, nil},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := translateEntryErr(tt.err)
			if tt.wantErr != nil {
				if !errors.Is(got, tt.wantErr) {
					t.Fatalf("translateEntryErr(%v) = %v, want errors.Is match for %v", tt.err, got, tt.wantErr)
				}
				return
			}
			if errors.Is(got, domain.ErrAlreadyEntered) || errors.Is(got, domain.ErrCompetitionEntryNotFound) || errors.Is(got, domain.ErrCompetitionFull) {
				t.Fatalf("translateEntryErr(%v) = %v, an unrelated pg error must not be mismapped to a specific domain sentinel", tt.err, got)
			}
		})
	}
}

// TestTranslateEntryErr_NoRowsNeverBecomesCompetitionNotFound pins the
// specific regression this ticket's own review flagged as easy to
// introduce by copy-paste from translateErr (the Competition-level
// sibling): a miss on GetEntryByID/UpdateEntryPaymentStatus must never
// answer with the wrong aggregate's not-found sentinel.
func TestTranslateEntryErr_NoRowsNeverBecomesCompetitionNotFound(t *testing.T) {
	t.Parallel()

	got := translateEntryErr(pgx.ErrNoRows)
	if errors.Is(got, domain.ErrCompetitionNotFound) {
		t.Fatalf("translateEntryErr(pgx.ErrNoRows) = %v, must not be domain.ErrCompetitionNotFound (wrong aggregate)", got)
	}
}

// TestTranslateEntryErr_ForeignKeyViolation pins T17.3's fix (part of issue
// #195, the Competitions mirror of #185/T15.6): a 23503 foreign_key_violation
// on competition_entries.competition_id — REFERENCES competitions (id)
// (db/migrations/0014_competitions.sql:150) — must translate to
// domain.ErrCompetitionNotFound, not fall through to the wrapped default and
// answer codes.Internal for what is, at the moment it happens, a legitimate
// (if racy) client-visible condition: app.Service.EnterCompetition's own
// GetByID guard passed, and the Competition was deleted before this INSERT
// ran.
//
// Matched on Code alone, with no ConstraintName check — deliberately,
// because the actual raise site in this table's case is NOT always the named
// constraint. enforce_competition_capacity()'s BEFORE INSERT trigger
// (same migration) locks the parent row with `SELECT ... FOR UPDATE` and
// raises its OWN 23503 (`USING ERRCODE = 'foreign_key_violation'`) when the
// Competition is missing — and a BEFORE trigger always fires before
// Postgres's own implicit FK-check trigger, which runs AFTER the row would
// be inserted. That manually-raised error carries no ConstraintName at all
// (it isn't a real constraint firing), so a mapping keyed on ConstraintName
// would miss it; Code "23503" is the only fact both possible origins share.
func TestTranslateEntryErr_ForeignKeyViolation(t *testing.T) {
	t.Parallel()

	got := translateEntryErr(&pgconn.PgError{Code: "23503"})
	if !errors.Is(got, domain.ErrCompetitionNotFound) {
		t.Fatalf("translateEntryErr(23503) = %v, want errors.Is match for domain.ErrCompetitionNotFound", got)
	}
	// The inverse guard: a foreign-key violation must not be mistaken for
	// the capacity trigger's own P0001 or the unique index's 23505 — three
	// different constraints firing on the same table must not collapse onto
	// one sentinel.
	if errors.Is(got, domain.ErrCompetitionFull) || errors.Is(got, domain.ErrAlreadyEntered) {
		t.Fatalf("translateEntryErr(23503) = %v, must not be mistaken for the capacity guard or the unique index", got)
	}
}

// TestTranslateErr_ForeignKeyViolation pins T17.3's other half: a 23503 on
// competitions.venue_facility_id — REFERENCES facilities (id)
// (db/migrations/0014_competitions.sql:36) — must translate to
// domain.ErrFacilityNotFound, reusing the SAME sentinel
// app.Service.ScheduleCompetition's own FacilityExists guard already returns
// for the non-racing "no such Facility" case — the caller-visible fact is
// identical whichever path caught it (#195's own reasoning).
//
// This is the only FK translateErr's callers can hit: Create's other insert
// in the same transaction (CreateCompetitionSession, competition_sessions.
// competition_id) references the Competition row this same transaction just
// inserted, uncommitted — no concurrent session can see or delete it before
// commit, so that FK cannot race. Matching on Code alone is therefore safe
// here too, with only one real candidate FK reachable through this
// function's callers.
func TestTranslateErr_ForeignKeyViolation(t *testing.T) {
	t.Parallel()

	got := translateErr(&pgconn.PgError{Code: "23503"})
	if !errors.Is(got, domain.ErrFacilityNotFound) {
		t.Fatalf("translateErr(23503) = %v, want errors.Is match for domain.ErrFacilityNotFound", got)
	}
	if errors.Is(got, domain.ErrCompetitionNotFound) {
		t.Fatalf("translateErr(23503) = %v, must not be mistaken for the pgx.ErrNoRows arm (wrong condition)", got)
	}
}

// TestMustUUID_PanicsOnMalformedInput proves mustUUID treats a malformed ID
// as the programmer error it documents (upstream invariant already
// violated), mirroring internal/socialplay/adapter/postgres's identical
// helper.
func TestMustUUID_PanicsOnMalformedInput(t *testing.T) {
	t.Parallel()

	defer func() {
		if recover() == nil {
			t.Fatal("expected mustUUID to panic on a malformed uuid")
		}
	}()
	mustUUID("not-a-uuid")
}
