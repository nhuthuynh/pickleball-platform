package domain_test

import (
	"errors"
	"testing"

	"github.com/nhuthuynh/white-label/internal/socialplay/domain"
)

// fixtureGame builds a valid scheduled Game with the given capacity, for
// use as Register's game argument in these tests. GuestAllowance defaults
// to 0 (no guests) — tests that need guests use fixtureGameWithGuests.
func fixtureGame(t *testing.T, capacity int) domain.Game {
	t.Helper()
	return fixtureGameWithGuests(t, capacity, 0)
}

// fixtureGameWithGuests builds a valid scheduled Game with the given
// capacity and GuestAllowance (T8.6), for tests exercising the weighted
// capacity rule / guest-count boundary checks. PaymentMethod is fixed at
// "either" — no test in this file asserts on PaymentMethod, so any valid
// value would do; "either" is the least restrictive, least likely to be
// mistaken for a deliberate test input.
func fixtureGameWithGuests(t *testing.T, capacity, guestAllowance int) domain.Game {
	t.Helper()
	r := mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")
	g, err := domain.NewGame("g1", "host-1", "facility-1", "venue-1", []string{"court-1"}, r, capacity, domain.PaymentMethodEither, guestAllowance, domain.Money{Cents: 1500, Currency: "USD"})
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
	reg, err := domain.Register(g, nil, "player-1", 0)
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
	_, err := domain.Register(g, nil, "", 0)
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
			_, err := domain.Register(g, tt.existing, "player-2", 0)
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

	_, err := domain.Register(g, existing, "player-1", 0)
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

	_, err := domain.Register(g, existing, "player-1", 0)
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

// TestRegistration_MarkPaymentStatus_Valid proves MarkPaymentStatus (T6.5)
// accepts every recognised PaymentStatus value and actually updates the
// field — this is the only path, besides Register's initial "unpaid"
// default, allowed to change PaymentStatus, so
// internal/payments/adapter/socialplay's RegistrationUpdater routes through
// it rather than mutating the exported field directly.
func TestRegistration_MarkPaymentStatus_Valid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status domain.PaymentStatus
	}{
		{"unpaid", domain.PaymentStatusUnpaid},
		{"paid", domain.PaymentStatusPaid},
		{"refunded", domain.PaymentStatusRefunded},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			reg := domain.Registration{ID: "r1", GameID: "g1", PlayerID: "player-1", PaymentStatus: domain.PaymentStatusUnpaid}
			if err := reg.MarkPaymentStatus(tt.status); err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if reg.PaymentStatus != tt.status {
				t.Fatalf("PaymentStatus = %v, want %v", reg.PaymentStatus, tt.status)
			}
		})
	}
}

// TestRegistration_MarkPaymentStatus_InvalidRejected proves an unrecognised
// PaymentStatus value is rejected rather than silently written — the field
// is a closed enum, mirroring PayableType.IsValid's role in
// internal/payments/domain.
func TestRegistration_MarkPaymentStatus_InvalidRejected(t *testing.T) {
	t.Parallel()

	reg := domain.Registration{ID: "r1", GameID: "g1", PlayerID: "player-1", PaymentStatus: domain.PaymentStatusUnpaid}
	err := reg.MarkPaymentStatus(domain.PaymentStatus("not-a-real-status"))
	if !errors.Is(err, domain.ErrInvalidPaymentStatus) {
		t.Fatalf("got err %v, want %v", err, domain.ErrInvalidPaymentStatus)
	}
	if reg.PaymentStatus != domain.PaymentStatusUnpaid {
		t.Fatalf("PaymentStatus = %v, want unchanged (unpaid)", reg.PaymentStatus)
	}
}

// TestRegister_CancelFreesCapacitySlot proves cancelling a Registration
// actually frees a capacity slot for a subsequent Register call — the same
// "prove it via the real re-booking path, not just the status field"
// standard T3 applied to Booking.Cancel.
func TestRegister_CancelFreesCapacitySlot(t *testing.T) {
	t.Parallel()

	g := fixtureGame(t, 1)

	regA, err := domain.Register(g, nil, "player-a", 0)
	if err != nil {
		t.Fatalf("player-a's registration should succeed: %v", err)
	}

	// The game is now full: player-b cannot register.
	_, err = domain.Register(g, []domain.Registration{regA}, "player-b", 0)
	if !errors.Is(err, domain.ErrGameFull) {
		t.Fatalf("got err %v, want %v", err, domain.ErrGameFull)
	}

	// Player-a cancels, freeing their slot.
	if err := regA.Cancel("player-a"); err != nil {
		t.Fatalf("player-a's cancel should succeed: %v", err)
	}

	// Player-b can now register into the freed slot.
	regB, err := domain.Register(g, []domain.Registration{regA}, "player-b", 0)
	if err != nil {
		t.Fatalf("player-b's registration should succeed after the cancel freed a slot: %v", err)
	}
	if regB.Status != domain.RegistrationStatusRegistered {
		t.Fatalf("regB.Status = %v, want registered", regB.Status)
	}
}

// TestRegister_GuestCount_SetOnRegistration proves a successful Register
// call actually stores guestCount on the returned Registration (T8.6) — not
// just validates it and discards it.
func TestRegister_GuestCount_SetOnRegistration(t *testing.T) {
	t.Parallel()

	g := fixtureGameWithGuests(t, 4, 2)
	reg, err := domain.Register(g, nil, "player-1", 2)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if reg.GuestCount != 2 {
		t.Fatalf("GuestCount = %d, want 2", reg.GuestCount)
	}
}

// TestRegister_WeightedCapacity_GuestsFillCapacityAloneRejectsNextComer is
// T8.6's required, load-bearing test (docs/process/t8-sprint-plan.md T8.6
// kickoff note; PE dossier §4 "prove the test is real"): a Capacity-3 Game
// where a single existing registration already brought 2 guests (weight
// 1 + 2 = 3, exactly at capacity) must reject a second registration, even
// with zero guests of its own — even though only ONE Registration row
// currently exists, well under Capacity by a plain headcount.
//
// This is deliberately the test that would incorrectly PASS (report no
// error) under the pre-T8.6 count-based check (1 active registration < 3
// capacity) and must FAIL correctly (report ErrGameFull) under the new
// weight-based check (3 existing weight + 1 new weight = 4 > 3 capacity).
// Verified manually during development by temporarily reverting Register's
// capacity check to the old `activeCount >= game.Capacity` shape and
// confirming this exact test then fails (see the PR description for that
// verification's output) — this comment records the process, the revert
// itself was never committed.
func TestRegister_WeightedCapacity_GuestsFillCapacityAloneRejectsNextComer(t *testing.T) {
	t.Parallel()

	g := fixtureGameWithGuests(t, 3, 2)
	existing := []domain.Registration{
		{ID: "r1", GameID: "g1", PlayerID: "player-1", Status: domain.RegistrationStatusRegistered, GuestCount: 2},
	}

	// Sanity check on the scenario itself: only one registration exists,
	// which is well under Capacity (3) by a plain headcount — the old
	// count-based check would have let this through.
	if len(existing) >= g.Capacity {
		t.Fatalf("test setup invalid: registration count %d already >= capacity %d, this scenario doesn't exercise the weighted-vs-count distinction", len(existing), g.Capacity)
	}

	_, err := domain.Register(g, existing, "player-2", 0)
	if !errors.Is(err, domain.ErrGameFull) {
		t.Fatalf("got err %v, want %v (existing weight 3 + new weight 1 > capacity 3)", err, domain.ErrGameFull)
	}
}

// TestRegister_WeightedCapacity_ExactFitAccepted is the companion happy
// path to the rejection test above: a registration whose guest weight
// exactly fills the remaining capacity is accepted, not off-by-one
// rejected.
func TestRegister_WeightedCapacity_ExactFitAccepted(t *testing.T) {
	t.Parallel()

	g := fixtureGameWithGuests(t, 3, 2)
	existing := []domain.Registration{
		{ID: "r1", GameID: "g1", PlayerID: "player-1", Status: domain.RegistrationStatusRegistered, GuestCount: 1},
	}

	// Existing weight = 1 + 1 = 2. New registration with 0 guests has
	// weight 1. Total = 3, exactly at capacity 3: must be accepted.
	_, err := domain.Register(g, existing, "player-2", 0)
	if err != nil {
		t.Fatalf("unexpected err: %v, want nil (weight exactly fills capacity)", err)
	}
}

// TestRegister_RejectsCancelledGame is T19.1's core proof (closes #212): a
// Player can no longer register for a Game that is already cancelled.
// domain.ErrGameCancelled is reused rather than a new sentinel — it already
// exists (RecordMatchResult's precondition) and is already mapped to
// codes.FailedPrecondition at the gRPC boundary (adapter/grpcapi.toStatus).
func TestRegister_RejectsCancelledGame(t *testing.T) {
	t.Parallel()

	g := fixtureGame(t, 4)
	g.Status = domain.StatusCancelled

	_, err := domain.Register(g, nil, "player-1", 0)
	if !errors.Is(err, domain.ErrGameCancelled) {
		t.Fatalf("got err %v, want %v", err, domain.ErrGameCancelled)
	}
}

// TestRegister_CancelledGameCheckedBeforeCapacityAndDoubleRegistration proves
// the cancelled-Game check runs first, ahead of the double-registration and
// capacity checks (ticket instruction 1's stated ordering rationale: "the
// game not being in a bookable state at all is the more fundamental fact").
// The scenario below would otherwise trip ErrAlreadyRegistered (this player
// already holds the game's one slot) — the cancelled-game rejection must win
// instead.
func TestRegister_CancelledGameCheckedBeforeCapacityAndDoubleRegistration(t *testing.T) {
	t.Parallel()

	g := fixtureGame(t, 1)
	g.Status = domain.StatusCancelled
	existing := []domain.Registration{
		{ID: "r1", GameID: "g1", PlayerID: "player-1", Status: domain.RegistrationStatusRegistered},
	}

	_, err := domain.Register(g, existing, "player-1", 0)
	if !errors.Is(err, domain.ErrGameCancelled) {
		t.Fatalf("got err %v, want %v (cancelled-game check must run before double-registration/capacity)", err, domain.ErrGameCancelled)
	}
}

// TestRegister_CancelledGameCheckedBeforeGuestAllowance is the same ordering
// proof against the guest-allowance check: guestCount here (5) exceeds the
// Game's GuestAllowance (0), which would otherwise trip
// ErrGuestAllowanceExceeded — the cancelled-game rejection must still win.
func TestRegister_CancelledGameCheckedBeforeGuestAllowance(t *testing.T) {
	t.Parallel()

	g := fixtureGameWithGuests(t, 4, 0)
	g.Status = domain.StatusCancelled

	_, err := domain.Register(g, nil, "player-1", 5)
	if !errors.Is(err, domain.ErrGameCancelled) {
		t.Fatalf("got err %v, want %v (cancelled-game check must run before guest-allowance)", err, domain.ErrGameCancelled)
	}
}

// TestRegister_GuestAllowance is the required table-driven boundary
// coverage (T8.6): GuestAllowance == 0 rejects any guest, GuestCount
// exactly at the allowance boundary is allowed, one over is rejected, and a
// negative guestCount is rejected regardless of allowance.
func TestRegister_GuestAllowance(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		guestAllowance int
		guestCount     int
		wantErr        error
	}{
		{"zero allowance, zero guests accepted", 0, 0, nil},
		{"zero allowance, one guest rejected", 0, 1, domain.ErrGuestAllowanceExceeded},
		{"guest count exactly at allowance boundary accepted", 2, 2, nil},
		{"guest count one over allowance rejected", 2, 3, domain.ErrGuestAllowanceExceeded},
		{"negative guest count rejected even with room in allowance", 2, -1, domain.ErrGuestAllowanceExceeded},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := fixtureGameWithGuests(t, 10, tt.guestAllowance)
			_, err := domain.Register(g, nil, "player-1", tt.guestCount)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got err %v, want %v", err, tt.wantErr)
			}
		})
	}
}
