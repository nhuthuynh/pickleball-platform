package domain_test

import (
	"testing"

	"github.com/nhuthuynh/white-label/internal/competitions/domain"
)

// TestSpotsLeft is the boundary proof T9.4 requires for the listing's
// server-computed spots_left: an UNWEIGHTED implementation (capacity minus
// the *number* of active entries) would visibly lie the moment any entrant
// brings a guest, and several of these cases are constructed so that the
// weighted and unweighted answers differ by a wide margin. If someone ever
// "simplifies" SpotsLeft to a COUNT, the wantIfUnweighted column names
// exactly the wrong answer they would get.
//
// This is the domain-side half of CLAUDE.md rule 4's dual enforcement for
// the same quantity: db/queries/competitions.sql's ListCompetitions computes
// spots_left in SQL with the identical weighted formula, and the
// -tags=integration test asserts the two agree against a real Postgres. The
// rule lives here in pure Go so it is provable in milliseconds; the SQL
// exists so the browse list doesn't need an N+1 query per Competition.
func TestSpotsLeft(t *testing.T) {
	comp := func(capacity int) domain.Competition {
		return domain.Competition{ID: "comp-1", Capacity: capacity, Status: domain.StatusScheduled}
	}
	entry := func(playerID string, guests int, status domain.EntryStatus) domain.CompetitionEntry {
		return domain.CompetitionEntry{
			CompetitionID: "comp-1",
			PlayerID:      playerID,
			GuestCount:    guests,
			Status:        status,
		}
	}

	tests := []struct {
		name    string
		compet  domain.Competition
		entries []domain.CompetitionEntry
		want    int
		// wantIfUnweighted documents what a (wrong) COUNT-based
		// implementation would return, so each case's purpose is legible.
		wantIfUnweighted int
	}{
		{
			name:             "empty competition has every place free",
			compet:           comp(8),
			entries:          nil,
			want:             8,
			wantIfUnweighted: 8,
		},
		{
			name:             "guestless entries weigh exactly one each",
			compet:           comp(8),
			entries:          []domain.CompetitionEntry{entry("p1", 0, domain.EntryStatusEntered), entry("p2", 0, domain.EntryStatusEntered)},
			want:             6,
			wantIfUnweighted: 6,
		},
		{
			// The headline case: one entry, three guests. A COUNT-based
			// spots_left would advertise 7 free places on a Competition that
			// really has 4 — the exact visible lie this test exists to
			// prevent.
			name:             "one entry with three guests occupies four places",
			compet:           comp(8),
			entries:          []domain.CompetitionEntry{entry("p1", 3, domain.EntryStatusEntered)},
			want:             4,
			wantIfUnweighted: 7,
		},
		{
			// Boundary: weighted occupancy lands exactly on capacity, so the
			// answer is precisely 0 — not 1, and not negative.
			name:             "exactly full reports zero, not a negative or an off-by-one",
			compet:           comp(8),
			entries:          []domain.CompetitionEntry{entry("p1", 3, domain.EntryStatusEntered), entry("p2", 3, domain.EntryStatusEntered)},
			want:             0,
			wantIfUnweighted: 6,
		},
		{
			// Defensive floor. The DB capacity guard should make this state
			// unreachable, but a browse list must never show a player a
			// negative number of free places even if it were reached some
			// other way — mirrors the GREATEST(...,0) floor in the SQL.
			name:             "over-occupied floors at zero rather than going negative",
			compet:           comp(3),
			entries:          []domain.CompetitionEntry{entry("p1", 3, domain.EntryStatusEntered)},
			want:             0,
			wantIfUnweighted: 2,
		},
		{
			// Cancelled entries release their guests' places too, not just
			// the entrant's own — the counting rule domain.Enter already
			// applies, restated for the read path.
			name:             "cancelled entries free all of their places, guests included",
			compet:           comp(8),
			entries:          []domain.CompetitionEntry{entry("p1", 3, domain.EntryStatusCancelled), entry("p2", 1, domain.EntryStatusEntered)},
			want:             6,
			wantIfUnweighted: 7,
		},
		{
			// Scope filtering, mirroring countActiveEntries': another
			// Competition's entries must never be counted against this one's
			// capacity, so an adapter handing over an unfiltered slice is
			// safe.
			name:   "entries belonging to another competition are not counted",
			compet: comp(8),
			entries: []domain.CompetitionEntry{
				{CompetitionID: "other-comp", PlayerID: "px", GuestCount: 5, Status: domain.EntryStatusEntered},
				entry("p1", 1, domain.EntryStatusEntered),
			},
			want:             6,
			wantIfUnweighted: 7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := domain.SpotsLeft(tt.compet, tt.entries)
			if got != tt.want {
				t.Errorf("SpotsLeft() = %d, want %d (an unweighted COUNT-based implementation would have returned %d — see this test's doc comment)", got, tt.want, tt.wantIfUnweighted)
			}
		})
	}
}

// TestSpotsLeft_AgreesWithEnter is the consistency proof that matters most:
// SpotsLeft and Enter must never disagree about whether one more entry of a
// given weight fits. A spots_left that says "2 places free" while Enter
// rejects a 2-place entry with ErrCompetitionFull is a bug a client would
// surface as an unexplained failure right at the point of payment.
//
// This walks every guest count from 0 up to the allowance against a
// part-filled Competition and asserts the two agree in both directions.
func TestSpotsLeft_AgreesWithEnter(t *testing.T) {
	const capacity = 10

	competition, err := domain.NewCompetition(
		"comp-1", "host-1", "Autumn Open", "",
		[]domain.Session{{
			Range:    mustRange(t, "2026-09-01T09:00:00Z", "2026-09-01T12:00:00Z"),
			CourtIDs: []string{"court-1"},
		}},
		capacity, 4,
		domain.PaymentMethodEither,
		domain.Money{AmountCents: 2500, CurrencyCode: "AUD"},
		domain.FormatDoubles,
		"token-1",
	)
	if err != nil {
		t.Fatalf("bad fixture competition: %v", err)
	}

	// Part-fill: one entry bringing 2 guests occupies 3 of 10 places, so 7
	// remain.
	existing := []domain.CompetitionEntry{{
		CompetitionID: competition.ID,
		PlayerID:      "seated-player",
		GuestCount:    2,
		Status:        domain.EntryStatusEntered,
	}}

	free := domain.SpotsLeft(competition, existing)
	if free != 7 {
		t.Fatalf("SpotsLeft() = %d, want 7 (1 entrant + 2 guests occupy 3 of %d)", free, capacity)
	}

	for guests := 0; guests <= 4; guests++ {
		weight := 1 + guests
		_, err := domain.Enter(competition, existing, "new-player", guests, domain.EntrySourceApp)
		fits := weight <= free

		if fits && err != nil {
			t.Errorf("guests=%d (weight %d): SpotsLeft reported %d free so this must fit, but Enter rejected it: %v", guests, weight, free, err)
		}
		if !fits && err == nil {
			t.Errorf("guests=%d (weight %d): SpotsLeft reported only %d free so this must NOT fit, but Enter accepted it", guests, weight, free)
		}
	}
}
