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
// 0005_socialplay.sql) must translate to domain.ErrAlreadyRegistered, and a
// P0001 raised by the registrations_capacity_guard trigger (db/migrations/
// 0006_socialplay_capacity_guard.sql) must translate to domain.ErrGameFull
// — neither may leak as a raw pgconn.PgError, per the adapter boundary
// CLAUDE.md rule 5 requires. Runs without internal/gen or a real database.
//
// T17.2 (part of #195) adds the 23503 case: registrations.game_id uuid NOT
// NULL REFERENCES games (id) (db/migrations/0005_socialplay.sql — default
// Postgres naming, no CONSTRAINT clause given, gives
// registrations_game_id_fkey) firing when the Game named by a Create call no
// longer exists. app.Service.RegisterForGame already calls
// s.games.GetByID(in.GameID) before this insert, so in the non-racing case
// that read's own domain.ErrGameNotFound wins and this arm is unreached; #195's
// point is the narrower window where the Game is deleted *between* that read
// and this insert, which only the FK — not the app-level read — can still
// catch. Before this ticket that 23503 fell through to the wrapped default
// below and answered codes.Internal (see toStatus's default case) — a 500 for
// what is, at the moment it happens, a legitimate client-visible race.
// Deliberately the same sentinel the guarding read itself would have
// returned (ErrGameNotFound, not a new one) — the caller-visible fact is
// identical in the racing and non-racing case, per #195's own reasoning.
// Postgres actually raising 23503 for this insert is a separate claim, proved
// against a real database in foreign_key_race_integration_test.go; this test
// proves only the translation.
func TestTranslateRegistrationErr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		err     error
		wantErr error
	}{
		{"unique violation (23505) becomes ErrAlreadyRegistered", &pgconn.PgError{Code: "23505"}, domain.ErrAlreadyRegistered},
		{"capacity guard trigger (P0001) becomes ErrGameFull", &pgconn.PgError{Code: "P0001"}, domain.ErrGameFull},
		{"no rows becomes ErrRegistrationNotFound", pgx.ErrNoRows, domain.ErrRegistrationNotFound},
		{
			name:    "foreign key violation (23503) on registrations.game_id becomes ErrGameNotFound",
			err:     &pgconn.PgError{Code: "23503", ConstraintName: "registrations_game_id_fkey"},
			wantErr: domain.ErrGameNotFound,
		},
		{
			// Constraint name deliberately not part of the match — see
			// TestTranslateGameErr's identical case for why: Postgres names
			// FKs from the table/column by default, but a migration is free
			// to rename one, and matching on the name string would make the
			// mapping silently fall back to Internal the day that happens.
			name:    "23503 without a constraint name still translates",
			err:     &pgconn.PgError{Code: "23503"},
			wantErr: domain.ErrGameNotFound,
		},
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
			if errors.Is(got, domain.ErrAlreadyRegistered) || errors.Is(got, domain.ErrRegistrationNotFound) || errors.Is(got, domain.ErrGameFull) || errors.Is(got, domain.ErrGameNotFound) {
				t.Fatalf("translateRegistrationErr(%v) = %v, an unrelated pg error must not be mismapped to a specific domain sentinel", tt.err, got)
			}
		})
	}
}

// TestTranslateGameErr mirrors TestTranslateRegistrationErr for the Game
// repository. Before T17.2 its only mapped case was not-found (games had no
// unique-constraint translation to prove since Create's only failure mode
// was a malformed/duplicate ID collision, which is not a modelled domain
// error).
//
// T17.2 (part of #195) adds the 23503 case: games.venue_facility_id uuid
// REFERENCES facilities (id) (db/migrations/0011_socialplay_facility_fk.sql
// — an ALTER TABLE ADD COLUMN with no CONSTRAINT clause, so Postgres's
// default naming gives games_venue_facility_id_fkey) firing when the Facility
// named by a Create call no longer exists. app.Service.ScheduleGame already
// calls facilities.FacilityExists(ctx, game.VenueFacilityID) before this
// insert, so in the non-racing case that read's own
// domain.ErrFacilityNotFound wins; #195's point is the narrower window where
// the Facility is deleted *between* that read and this insert. games has
// exactly one FK column (VenueFacilityID; CourtIDs is a uuid[] with no
// database-level FK — see 0005_socialplay.sql's own doc comment on why), so a
// 23503 from this table's Create is unambiguous: it can only be this
// constraint. Reuses domain.ErrFacilityNotFound rather than minting a new
// sentinel — the same reasoning TestTranslateRegistrationErr's comment gives,
// and this context's own existing precedent: FacilityLookup.FacilityExists
// already returns this exact sentinel for the non-racing "unknown Facility"
// case (internal/socialplay/adapter/facilities/lookup.go).
func TestTranslateGameErr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		err     error
		wantErr error
	}{
		{"no rows becomes ErrGameNotFound", pgx.ErrNoRows, domain.ErrGameNotFound},
		{
			name:    "foreign key violation (23503) on games.venue_facility_id becomes ErrFacilityNotFound",
			err:     &pgconn.PgError{Code: "23503", ConstraintName: "games_venue_facility_id_fkey"},
			wantErr: domain.ErrFacilityNotFound,
		},
		{
			// Constraint name deliberately not part of the match, mirroring
			// booking's own T15.6 precedent (internal/booking/adapter/
			// postgres/foreign_key_test.go): a migration is free to rename
			// the constraint, and matching on the name string would make
			// this mapping silently fall back to Internal that day.
			name:    "23503 without a constraint name still translates",
			err:     &pgconn.PgError{Code: "23503"},
			wantErr: domain.ErrFacilityNotFound,
		},
		{"unrelated pg error is wrapped, not silently mapped", &pgconn.PgError{Code: "42601"}, nil},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := translateGameErr(tt.err)
			if tt.wantErr != nil {
				if !errors.Is(got, tt.wantErr) {
					t.Fatalf("translateGameErr(%v) = %v, want errors.Is match for %v", tt.err, got, tt.wantErr)
				}
				return
			}
			if errors.Is(got, domain.ErrGameNotFound) || errors.Is(got, domain.ErrFacilityNotFound) {
				t.Fatalf("translateGameErr(%v) = %v, an unrelated pg error must not be mismapped to a specific domain sentinel", tt.err, got)
			}
		})
	}
}

// TestTranslateWaitlistErr mirrors TestTranslateRegistrationErr for the
// waitlist repository (T6.6): a 23505 unique_violation on
// waitlist_entries_active_player_per_game_idx (db/migrations/
// 0007_socialplay_waitlist.sql) becomes domain.ErrAlreadyOnWaitlist, and a
// P0001 raised by the (now waitlist-aware) enforce_game_capacity trigger
// becomes domain.ErrGameFull — neither may leak as a raw pgconn.PgError.
func TestTranslateWaitlistErr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		err     error
		wantErr error
	}{
		{"unique violation (23505) becomes ErrAlreadyOnWaitlist", &pgconn.PgError{Code: "23505"}, domain.ErrAlreadyOnWaitlist},
		{"capacity guard trigger (P0001) becomes ErrGameFull", &pgconn.PgError{Code: "P0001"}, domain.ErrGameFull},
		{"no rows becomes ErrWaitlistEntryNotFound", pgx.ErrNoRows, domain.ErrWaitlistEntryNotFound},
		{"unrelated pg error is wrapped, not silently mapped", &pgconn.PgError{Code: "42601"}, nil},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := translateWaitlistErr(tt.err)
			if tt.wantErr != nil {
				if !errors.Is(got, tt.wantErr) {
					t.Fatalf("translateWaitlistErr(%v) = %v, want errors.Is match for %v", tt.err, got, tt.wantErr)
				}
				return
			}
			if errors.Is(got, domain.ErrAlreadyOnWaitlist) || errors.Is(got, domain.ErrWaitlistEntryNotFound) || errors.Is(got, domain.ErrGameFull) {
				t.Fatalf("translateWaitlistErr(%v) = %v, an unrelated pg error must not be mismapped to a specific domain sentinel", tt.err, got)
			}
		})
	}
}

// TestTranslateMatchErr mirrors TestTranslateGameErr for the Match
// repository (T10.4): a 23503 foreign_key_violation on matches.game_id
// becomes domain.ErrGameNotFound (db/migrations/0015_socialplay_matches.sql
// has no unique-constraint/capacity-guard case to prove, unlike
// registrations/waitlist_entries — see that migration's own doc comment on
// why matches carries no such invariant).
func TestTranslateMatchErr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		err     error
		wantErr error
	}{
		{"foreign key violation (23503) becomes ErrGameNotFound", &pgconn.PgError{Code: "23503"}, domain.ErrGameNotFound},
		{"no rows becomes ErrGameNotFound", pgx.ErrNoRows, domain.ErrGameNotFound},
		{"unrelated pg error is wrapped, not silently mapped", &pgconn.PgError{Code: "42601"}, nil},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := translateMatchErr(tt.err)
			if tt.wantErr != nil {
				if !errors.Is(got, tt.wantErr) {
					t.Fatalf("translateMatchErr(%v) = %v, want errors.Is match for %v", tt.err, got, tt.wantErr)
				}
				return
			}
			if errors.Is(got, domain.ErrGameNotFound) {
				t.Fatalf("translateMatchErr(%v) = %v, an unrelated pg error must not be mismapped to ErrGameNotFound", tt.err, got)
			}
		})
	}
}

// TestMarshalUnmarshalScore_RoundTrips proves domain.Match.Score survives
// the jsonb marshal/unmarshal round trip this adapter's first jsonb column
// needs (CreateMatchParams.Score's doc comment) — every key and value comes
// back exactly as given.
func TestMarshalUnmarshalScore_RoundTrips(t *testing.T) {
	t.Parallel()

	want := map[string]int{"player-1": 11, "player-2": 7}

	b, err := marshalScore(want)
	if err != nil {
		t.Fatalf("marshalScore: %v", err)
	}

	got, err := unmarshalScore(b)
	if err != nil {
		t.Fatalf("unmarshalScore: %v", err)
	}

	if len(got) != len(want) {
		t.Fatalf("unmarshalScore round trip = %v, want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Fatalf("unmarshalScore round trip = %v, want %v", got, want)
		}
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
