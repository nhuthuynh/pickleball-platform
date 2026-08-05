package domain_test

import (
	"errors"
	"testing"

	"github.com/nhuthuynh/white-label/internal/competitions/domain"
)

// entry is a fixture helper for an already-persisted CompetitionEntry.
func entry(competitionID, playerID string, guestCount int, status domain.EntryStatus) domain.CompetitionEntry {
	return domain.CompetitionEntry{
		CompetitionID: competitionID,
		PlayerID:      playerID,
		GuestCount:    guestCount,
		Source:        domain.EntrySourceApp,
		Status:        status,
		PaymentStatus: domain.PaymentStatusUnpaid,
	}
}

// TestEnter_Valid proves the happy path and the initial field values a new
// entry starts with: entered, unpaid, and the caller-supplied source.
func TestEnter_Valid(t *testing.T) {
	t.Parallel()

	c := newValidCompetition(t, 8, 2)

	e, err := domain.Enter(c, nil, "player-1", 2, domain.EntrySourceSocial)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if e.CompetitionID != "comp-1" || e.PlayerID != "player-1" {
		t.Fatalf("unexpected identity fields: %+v", e)
	}
	if e.GuestCount != 2 {
		t.Fatalf("guest count = %d, want 2", e.GuestCount)
	}
	if e.Source != domain.EntrySourceSocial {
		t.Fatalf("source = %v, want %v", e.Source, domain.EntrySourceSocial)
	}
	if e.Status != domain.EntryStatusEntered {
		t.Fatalf("status = %v, want %v", e.Status, domain.EntryStatusEntered)
	}
	if e.PaymentStatus != domain.PaymentStatusUnpaid {
		t.Fatalf("payment status = %v, want %v", e.PaymentStatus, domain.PaymentStatusUnpaid)
	}
	// Enter is a pure domain constructor: assigning a durable ID is the
	// app/adapter layer's job at persistence time (mirrors
	// socialplay.Register).
	if e.ID != "" {
		t.Fatalf("ID = %q, want empty (assigned at persistence time)", e.ID)
	}
}

// TestEnter_WeightedCapacity is the ticket's required proof that the
// capacity invariant is *weighted* from day one, not a headcount. The first
// case is the one that would pass a count-based check and must fail it: a
// Capacity 3 Competition holding exactly one entry that brought 2 guests
// (weight 3 — already full) must reject a brand-new entry bringing no
// guests at all, even though only one entry exists against a capacity of 3.
func TestEnter_WeightedCapacity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		capacity       int
		guestAllowance int
		existing       []domain.CompetitionEntry
		guestCount     int
		wantErr        error
	}{
		{
			name:     "one entry with two guests fills a capacity of three and rejects a guestless entry",
			capacity: 3, guestAllowance: 2,
			existing:   []domain.CompetitionEntry{entry("comp-1", "player-1", 2, domain.EntryStatusEntered)},
			guestCount: 0,
			wantErr:    domain.ErrCompetitionFull,
		},
		{
			name:     "exactly at capacity still succeeds",
			capacity: 4, guestAllowance: 2,
			existing:   []domain.CompetitionEntry{entry("comp-1", "player-1", 2, domain.EntryStatusEntered)},
			guestCount: 0,
			wantErr:    nil, // weight 3 + 1 == 4 == capacity
		},
		{
			name:     "exactly at capacity via guests still succeeds",
			capacity: 4, guestAllowance: 2,
			existing:   []domain.CompetitionEntry{entry("comp-1", "player-1", 0, domain.EntryStatusEntered)},
			guestCount: 2,
			wantErr:    nil, // weight 1 + 3 == 4 == capacity
		},
		{
			name:     "one over capacity via guests is rejected",
			capacity: 4, guestAllowance: 3,
			existing:   []domain.CompetitionEntry{entry("comp-1", "player-1", 0, domain.EntryStatusEntered)},
			guestCount: 3,
			wantErr:    domain.ErrCompetitionFull, // weight 1 + 4 == 5 > capacity
		},
		{
			name:     "empty competition at capacity one accepts a guestless entry",
			capacity: 1, guestAllowance: 0,
			existing:   nil,
			guestCount: 0,
			wantErr:    nil,
		},
		{
			name:     "a single entry fills a capacity of one",
			capacity: 1, guestAllowance: 0,
			existing:   []domain.CompetitionEntry{entry("comp-1", "player-1", 0, domain.EntryStatusEntered)},
			guestCount: 0,
			wantErr:    domain.ErrCompetitionFull,
		},
		{
			name:     "cancelled entries free their weighted slots",
			capacity: 3, guestAllowance: 2,
			existing:   []domain.CompetitionEntry{entry("comp-1", "player-1", 2, domain.EntryStatusCancelled)},
			guestCount: 0,
			wantErr:    nil,
		},
		{
			name:     "entries belonging to another competition do not consume capacity",
			capacity: 3, guestAllowance: 2,
			existing:   []domain.CompetitionEntry{entry("comp-other", "player-1", 2, domain.EntryStatusEntered)},
			guestCount: 0,
			wantErr:    nil,
		},
		{
			name:     "several entries accumulate their guest weights",
			capacity: 5, guestAllowance: 2,
			existing: []domain.CompetitionEntry{
				entry("comp-1", "player-1", 1, domain.EntryStatusEntered),
				entry("comp-1", "player-2", 1, domain.EntryStatusEntered),
			},
			guestCount: 1,
			wantErr:    domain.ErrCompetitionFull, // (1+1) + (1+1) + (1+1) == 6 > 5
		},
		{
			name:     "several entries accumulating to exactly capacity still succeed",
			capacity: 6, guestAllowance: 2,
			existing: []domain.CompetitionEntry{
				entry("comp-1", "player-1", 1, domain.EntryStatusEntered),
				entry("comp-1", "player-2", 1, domain.EntryStatusEntered),
			},
			guestCount: 1,
			wantErr:    nil, // (1+1) + (1+1) + (1+1) == 6 == capacity
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := newValidCompetition(t, tt.capacity, tt.guestAllowance)
			_, err := domain.Enter(c, tt.existing, "new-player", tt.guestCount, domain.EntrySourceApp)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("got err %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
		})
	}
}

// TestEnter_GuestAllowance covers the guest-allowance boundary in both
// directions, including the GuestAllowance == 0 case.
func TestEnter_GuestAllowance(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		guestAllowance int
		guestCount     int
		wantErr        error
	}{
		{"zero allowance with zero guests is allowed", 0, 0, nil},
		{"zero allowance with one guest is rejected", 0, 1, domain.ErrGuestAllowanceExceeded},
		{"guest count exactly at the allowance is allowed", 2, 2, nil},
		{"guest count one under the allowance is allowed", 2, 1, nil},
		{"guest count one over the allowance is rejected", 2, 3, domain.ErrGuestAllowanceExceeded},
		{"negative guest count is rejected", 2, -1, domain.ErrGuestAllowanceExceeded},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Capacity is deliberately generous so a guest-allowance
			// rejection can never be confused with a capacity rejection.
			c := newValidCompetition(t, 100, tt.guestAllowance)
			_, err := domain.Enter(c, nil, "player-1", tt.guestCount, domain.EntrySourceApp)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("got err %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
		})
	}
}

// TestEnter_GuestAllowanceCheckedBeforeCapacity proves precedence: a caller
// asking for more guests than the Competition permits gets that specific,
// actionable error rather than a generic "full", even on a Competition that
// is also out of room (mirrors socialplay.Register's documented ordering).
func TestEnter_GuestAllowanceCheckedBeforeCapacity(t *testing.T) {
	t.Parallel()

	c := newValidCompetition(t, 1, 0)
	existing := []domain.CompetitionEntry{entry("comp-1", "player-1", 0, domain.EntryStatusEntered)}

	_, err := domain.Enter(c, existing, "player-2", 3, domain.EntrySourceApp)
	if !errors.Is(err, domain.ErrGuestAllowanceExceeded) {
		t.Fatalf("got err %v, want %v", err, domain.ErrGuestAllowanceExceeded)
	}
}

// TestEnter_AlreadyEntered proves a player holding an active entry can't
// enter twice, and that the more specific ErrAlreadyEntered wins over
// ErrCompetitionFull — a player who is already in must never be told the
// competition is full because of their own entry.
func TestEnter_AlreadyEntered(t *testing.T) {
	t.Parallel()

	t.Run("an active entry blocks a second one", func(t *testing.T) {
		t.Parallel()

		c := newValidCompetition(t, 8, 0)
		existing := []domain.CompetitionEntry{entry("comp-1", "player-1", 0, domain.EntryStatusEntered)}
		if _, err := domain.Enter(c, existing, "player-1", 0, domain.EntrySourceApp); !errors.Is(err, domain.ErrAlreadyEntered) {
			t.Fatalf("got err %v, want %v", err, domain.ErrAlreadyEntered)
		}
	})

	t.Run("already entered wins over full", func(t *testing.T) {
		t.Parallel()

		c := newValidCompetition(t, 1, 0)
		existing := []domain.CompetitionEntry{entry("comp-1", "player-1", 0, domain.EntryStatusEntered)}
		if _, err := domain.Enter(c, existing, "player-1", 0, domain.EntrySourceApp); !errors.Is(err, domain.ErrAlreadyEntered) {
			t.Fatalf("got err %v, want %v", err, domain.ErrAlreadyEntered)
		}
	})

	t.Run("a cancelled entry does not block re-entering", func(t *testing.T) {
		t.Parallel()

		c := newValidCompetition(t, 8, 0)
		existing := []domain.CompetitionEntry{entry("comp-1", "player-1", 0, domain.EntryStatusCancelled)}
		if _, err := domain.Enter(c, existing, "player-1", 0, domain.EntrySourceApp); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
	})

	t.Run("an entry in another competition does not block", func(t *testing.T) {
		t.Parallel()

		c := newValidCompetition(t, 8, 0)
		existing := []domain.CompetitionEntry{entry("comp-other", "player-1", 0, domain.EntryStatusEntered)}
		if _, err := domain.Enter(c, existing, "player-1", 0, domain.EntrySourceApp); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
	})
}

// TestEnter_Validation covers the remaining Enter rejections: an empty
// player id, an out-of-enum source, and a cancelled Competition.
func TestEnter_Validation(t *testing.T) {
	t.Parallel()

	t.Run("empty player id is rejected", func(t *testing.T) {
		t.Parallel()

		c := newValidCompetition(t, 8, 0)
		if _, err := domain.Enter(c, nil, "", 0, domain.EntrySourceApp); !errors.Is(err, domain.ErrEmptyPlayerID) {
			t.Fatalf("got err %v, want %v", err, domain.ErrEmptyPlayerID)
		}
	})

	t.Run("source outside the closed enum is rejected", func(t *testing.T) {
		t.Parallel()

		c := newValidCompetition(t, 8, 0)
		for _, src := range []domain.EntrySource{domain.EntrySource("shared_link"), domain.EntrySource(""), domain.EntrySource("App")} {
			if _, err := domain.Enter(c, nil, "player-1", 0, src); !errors.Is(err, domain.ErrInvalidEntrySource) {
				t.Fatalf("source %q: got err %v, want %v", src, err, domain.ErrInvalidEntrySource)
			}
		}
	})

	t.Run("entering a cancelled competition is rejected", func(t *testing.T) {
		t.Parallel()

		c := newValidCompetition(t, 8, 0)
		if err := c.Cancel(); err != nil {
			t.Fatalf("unexpected err cancelling: %v", err)
		}
		if _, err := domain.Enter(c, nil, "player-1", 0, domain.EntrySourceApp); !errors.Is(err, domain.ErrCompetitionCancelled) {
			t.Fatalf("got err %v, want %v", err, domain.ErrCompetitionCancelled)
		}
	})
}

// TestEntrySourceIsValid keeps EntrySource a closed enum over exactly the
// two values Social Play's RegistrationSource already uses (§A3: one
// ubiquitous language, not a third vocabulary for the same fact).
func TestEntrySourceIsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		value domain.EntrySource
		want  bool
	}{
		{domain.EntrySourceApp, true},
		{domain.EntrySourceSocial, true},
		{domain.EntrySource("shared_link"), false},
		{domain.EntrySource("Social"), false},
		{domain.EntrySource(""), false},
	}
	for _, tt := range tests {
		if got := tt.value.IsValid(); got != tt.want {
			t.Fatalf("EntrySource(%q).IsValid() = %v, want %v", tt.value, got, tt.want)
		}
	}
}

// TestEntrySourceValuesMatchSocialPlay pins the wire values themselves, not
// just the constant names: §A3's ruling is that Competitions reuses Social
// Play's exact `app | social` vocabulary. A rename on either side should
// break a test, not silently fork the ubiquitous language.
func TestEntrySourceValuesMatchSocialPlay(t *testing.T) {
	t.Parallel()

	if string(domain.EntrySourceApp) != "app" {
		t.Fatalf("EntrySourceApp = %q, want %q", domain.EntrySourceApp, "app")
	}
	if string(domain.EntrySourceSocial) != "social" {
		t.Fatalf("EntrySourceSocial = %q, want %q", domain.EntrySourceSocial, "social")
	}
}

// TestPaymentStatusIsValid keeps PaymentStatus a closed enum mirroring
// internal/payments/domain.Status's unpaid -> paid -> refunded values.
func TestPaymentStatusIsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		value domain.PaymentStatus
		want  bool
	}{
		{domain.PaymentStatusUnpaid, true},
		{domain.PaymentStatusPaid, true},
		{domain.PaymentStatusRefunded, true},
		{domain.PaymentStatus("part_paid"), false},
		{domain.PaymentStatus(""), false},
	}
	for _, tt := range tests {
		if got := tt.value.IsValid(); got != tt.want {
			t.Fatalf("PaymentStatus(%q).IsValid() = %v, want %v", tt.value, got, tt.want)
		}
	}
}
