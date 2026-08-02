package postgres

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/nhuthuynh/white-label/internal/socialplay/domain"
)

// TestTranslateRegistrationErr proves the DB-level half of CLAUDE.md rule 4
// (invariants enforced in Postgres AND expressed in the domain): a 23505
// unique_violation on registrations_active_player_per_game_idx (db/migrations/
// 0005_socialplay.sql) must translate to domain.ErrAlreadyRegistered, not
// leak as a raw pgconn.PgError — the adapter boundary CLAUDE.md rule 5
// requires. Runs without internal/gen or a real database.
func TestTranslateRegistrationErr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		err     error
		wantErr error
	}{
		{"unique violation (23505) becomes ErrAlreadyRegistered", &pgconn.PgError{Code: "23505"}, domain.ErrAlreadyRegistered},
		{"no rows becomes ErrRegistrationNotFound", pgx.ErrNoRows, domain.ErrRegistrationNotFound},
		{"unrelated pg error is wrapped, not silently mapped", &pgconn.PgError{Code: "42601"}, nil},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := translateRegistrationErr(tt.err)
			if tt.wantErr != nil {
				if !errors.Is(got, tt.wantErr) {
					t.Fatalf("translateRegistrationErr(%v) = %v, want errors.Is match for %v", tt.err, got, tt.wantErr)
				}
				return
			}
			if errors.Is(got, domain.ErrAlreadyRegistered) || errors.Is(got, domain.ErrRegistrationNotFound) {
				t.Fatalf("translateRegistrationErr(%v) = %v, an unrelated pg error must not be mismapped to a specific domain sentinel", tt.err, got)
			}
		})
	}
}

// TestTranslateGameErr mirrors TestTranslateRegistrationErr for the Game
// repository's only mapped case (not-found); games has no unique-constraint
// translation to prove since Create's only failure mode today is a
// malformed/duplicate ID collision, which is not a modelled domain error.
func TestTranslateGameErr(t *testing.T) {
	t.Parallel()

	if got := translateGameErr(pgx.ErrNoRows); !errors.Is(got, domain.ErrGameNotFound) {
		t.Fatalf("translateGameErr(pgx.ErrNoRows) = %v, want ErrGameNotFound", got)
	}

	unrelated := errors.New("boom")
	if got := translateGameErr(unrelated); errors.Is(got, domain.ErrGameNotFound) {
		t.Fatalf("translateGameErr(%v) = %v, an unrelated error must not be mismapped to ErrGameNotFound", unrelated, got)
	}
}

// TestMustUUID_PanicsOnMalformedInput proves mustUUID treats a malformed ID
// as the programmer error it documents (upstream invariant already
// violated), mirroring internal/booking/adapter/postgres's identical helper.
func TestMustUUID_PanicsOnMalformedInput(t *testing.T) {
	t.Parallel()

	defer func() {
		if recover() == nil {
			t.Fatal("expected mustUUID to panic on a malformed uuid")
		}
	}()
	mustUUID("not-a-uuid")
}
