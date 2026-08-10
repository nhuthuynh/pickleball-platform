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
