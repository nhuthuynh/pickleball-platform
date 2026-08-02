package domain_test

import (
	"errors"
	"testing"

	"github.com/nhuthuynh/white-label/internal/socialplay/domain"
)

// fixtureGame builds a valid scheduled Game with the given capacity, for
// use as Register's game argument in these tests.
func fixtureGame(t *testing.T, capacity int) domain.Game {
	t.Helper()
	r := mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")
	g, err := domain.NewGame("g1", "host-1", "facility-1", []string{"court-1"}, r, capacity)
	if err != nil {
		t.Fatalf("bad fixture game: %v", err)
	}
	return g
}

// TestRegister_Valid proves the happy path: registering into a Game with
// room produces a "registered"/"unpaid" Registration scoped to that Game
// and player.
func TestRegister_Valid(t *testing.T) {
	t.Parallel()

	g := fixtureGame(t, 4)
	reg, err := domain.Register(g, nil, "player-1")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if reg.GameID != g.ID {
		t.Fatalf("GameID = %q, want %q", reg.GameID, g.ID)
	}
	if reg.PlayerID != "player-1" {
		t.Fatalf("PlayerID = %q, want player-1", reg.PlayerID)
	}
	if reg.Status != domain.RegistrationStatusRegistered {
		t.Fatalf("Status = %v, want %v", reg.Status, domain.RegistrationStatusRegistered)
	}
	if reg.PaymentStatus != domain.PaymentStatusUnpaid {
		t.Fatalf("PaymentStatus = %v, want %v", reg.PaymentStatus, domain.PaymentStatusUnpaid)
	}
}

// TestRegister_EmptyPlayerID proves an empty playerID is rejected rather
// than silently producing a Registration nothing owns.
func TestRegister_EmptyPlayerID(t *testing.T) {
	t.Parallel()

	g := fixtureGame(t, 4)
	_, err := domain.Register(g, nil, "")
	if !errors.Is(err, domain.ErrEmptyPlayerID) {
		t.Fatalf("got err %v, want %v", err, domain.ErrEmptyPlayerID)
	}
}

// TestRegister_CapacityEnforced is the required Given/When/Then coverage
// from T5.1/T5.2: a Game of capacity 1 accepts exactly one active
// registration and rejects a second with the exact, stable ErrGameFull
// sentinel (kickoff note: T6's waitlist promotion trigger needs it).
// Cancelled registrations must not count against capacity.
func TestRegister_CapacityEnforced(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		capacity int
		existing []domain.Registration
		wantErr  error
	}{
		{
			name:     "capacity 1, no existing registrations accepts the first",
			capacity: 1,
			existing: nil,
			wantErr:  nil,
		},
		{
			name:     "capacity 1, one active existing registration rejects a second",
			capacity: 1,
			existing: []domain.Registration{
				{ID: "r1", GameID: "g1", PlayerID: "player-1", Status: domain.RegistrationStatusRegistered},
			},
			wantErr: domain.ErrGameFull,
		},
		{
			name:     "capacity 1, one cancelled existing registration does not count",
			capacity: 1,
			existing: []domain.Registration{
				{ID: "r1", GameID: "g1", PlayerID: "player-1", Status: domain.RegistrationStatusCancelled},
			},
			wantErr: nil,
		},
		{
			name:     "capacity 2, one active existing registration accepts a second",
			capacity: 2,
			existing: []domain.Registration{
				{ID: "r1", GameID: "g1", PlayerID: "player-1", Status: domain.RegistrationStatusRegistered},
			},
			wantErr: nil,
		},
		{
			name:     "registrations scoped to another game do not count against this game's capacity",
			capacity: 1,
			existing: []domain.Registration{
				{ID: "r1", GameID: "some-other-game", PlayerID: "player-1", Status: domain.RegistrationStatusRegistered},
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := fixtureGame(t, tt.capacity)
			_, err := domain.Register(g, tt.existing, "player-2")
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got err %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// TestRegister_AlreadyRegistered proves a player already actively
// registered for a Game cannot register a second time, and that this
// check fires even when it would otherwise also trip the capacity check
// (double-registration is the more specific, more useful error).
func TestRegister_AlreadyRegistered(t *testing.T) {
	t.Parallel()

	g := fixtureGame(t, 1)
	existing := []domain.Registration{
		{ID: "r1", GameID: "g1", PlayerID: "player-1", Status: domain.RegistrationStatusRegistered},
	}

	_, err := domain.Register(g, existing, "player-1")
	if !errors.Is(err, domain.ErrAlreadyRegistered) {
		t.Fatalf("got err %v, want %v", err, domain.ErrAlreadyRegistered)
	}
}

// TestRegister_AlreadyRegistered_AllowedAfterCancel proves a player whose
// prior registration for the Game was cancelled is free to register again.
func TestRegister_AlreadyRegistered_AllowedAfterCancel(t *testing.T) {
	t.Parallel()

	g := fixtureGame(t, 4)
	existing := []domain.Registration{
		{ID: "r1", GameID: "g1", PlayerID: "player-1", Status: domain.RegistrationStatusCancelled},
	}

	_, err := domain.Register(g, existing, "player-1")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
}

// TestRegistration_Cancel proves the registered -> cancelled transition by
// its owner, and that a double-cancel is rejected rather than silently
// accepted, mirroring booking.Booking.Cancel()/Game.Cancel()'s pattern.
func TestRegistration_Cancel(t *testing.T) {
	t.Parallel()

	reg := domain.Registration{
		ID: "r1", GameID: "g1", PlayerID: "player-1",
		Status: domain.RegistrationStatusRegistered,
	}

	if err := reg.Cancel("player-1"); err != nil {
		t.Fatalf("first cancel by owner should succeed, got %v", err)
	}
	if reg.Status != domain.RegistrationStatusCancelled {
		t.Fatalf("status = %v, want cancelled", reg.Status)
	}

	if err := reg.Cancel("player-1"); !errors.Is(err, domain.ErrIllegalStatusTransition) {
		t.Fatalf("double-cancel should be rejected, got %v", err)
	}
}

// TestRegistration_Cancel_WrongActor is the required BOLA-shaped test
// (P1 #6 / T5.5 groundwork): Player A must not be able to cancel Player
// B's registration. A mismatched actor gets a distinct
// ErrNotRegistrationOwner, not a silent success and not
// ErrIllegalStatusTransition.
func TestRegistration_Cancel_WrongActor(t *testing.T) {
	t.Parallel()

	reg := domain.Registration{
		ID: "r1", GameID: "g1", PlayerID: "player-b",
		Status: domain.RegistrationStatusRegistered,
	}

	err := reg.Cancel("player-a")
	if !errors.Is(err, domain.ErrNotRegistrationOwner) {
		t.Fatalf("got err %v, want %v", err, domain.ErrNotRegistrationOwner)
	}
	// The registration must be untouched: rejecting the wrong actor must
	// not have any side effect on Player B's registration state.
	if reg.Status != domain.RegistrationStatusRegistered {
		t.Fatalf("status = %v, want unchanged (registered)", reg.Status)
	}
}

// TestRegister_CancelFreesCapacitySlot proves cancelling a Registration
// actually frees a capacity slot for a subsequent Register call — the same
// "prove it via the real re-booking path, not just the status field"
// standard T3 applied to Booking.Cancel.
func TestRegister_CancelFreesCapacitySlot(t *testing.T) {
	t.Parallel()

	g := fixtureGame(t, 1)

	regA, err := domain.Register(g, nil, "player-a")
	if err != nil {
		t.Fatalf("player-a's registration should succeed: %v", err)
	}

	// The game is now full: player-b cannot register.
	_, err = domain.Register(g, []domain.Registration{regA}, "player-b")
	if !errors.Is(err, domain.ErrGameFull) {
		t.Fatalf("got err %v, want %v", err, domain.ErrGameFull)
	}

	// Player-a cancels, freeing their slot.
	if err := regA.Cancel("player-a"); err != nil {
		t.Fatalf("player-a's cancel should succeed: %v", err)
	}

	// Player-b can now register into the freed slot.
	regB, err := domain.Register(g, []domain.Registration{regA}, "player-b")
	if err != nil {
		t.Fatalf("player-b's registration should succeed after the cancel freed a slot: %v", err)
	}
	if regB.Status != domain.RegistrationStatusRegistered {
		t.Fatalf("regB.Status = %v, want registered", regB.Status)
	}
}
