package domain_test

import (
	"errors"
	"testing"

	"github.com/nhuthuynh/white-label/internal/socialplay/domain"
)

// T56.1 (issue #126, per-head pricing) — a Registration owes the Game's
// EntryFee once per HEAD, not once per registration.
//
// #126's original headline ask (add a price field, retire T8.10's
// PLACEHOLDER_REGISTRATION_FEE_CENTS) was already delivered by T9.2 —
// Game.EntryFee has existed since then. What was never built is the part
// the Product Owner answered on 2026-09-04: **per head, including guests**.
// Until this ticket a player bringing three guests paid for one, because
// nothing multiplied EntryFee by the party size at all.
//
// The rule lives in ExpectedAmount rather than inline in Register, because
// Payments has to validate a payment against the same number (#297) and
// CLAUDE.md rule 7 wants one definition of a concept, not two that drift.

func gameWithFee(t *testing.T, cents int64, currency string, guestAllowance int) domain.Game {
	t.Helper()

	g, err := domain.NewGame(
		"game-1", "host-1", "", "", []string{"court-1"},
		mustRange(t, "2026-10-01T09:00:00Z", "2026-10-01T10:00:00Z"),
		16, domain.PaymentMethodCash, guestAllowance,
		domain.Money{Cents: cents, Currency: currency},
	)
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	return g
}

// The arithmetic itself, including the boundary the whole ticket turns on:
// one player alone is 1 head, not 0.
func TestExpectedAmount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		feeCents   int64
		currency   string
		guestCount int
		wantCents  int64
	}{
		{name: "no guests pays for one head", feeCents: 2500, currency: "USD", guestCount: 0, wantCents: 2500},
		{name: "one guest pays for two heads", feeCents: 2500, currency: "USD", guestCount: 1, wantCents: 5000},
		{name: "three guests pays for four heads", feeCents: 2500, currency: "USD", guestCount: 3, wantCents: 10000},
		{
			// The regression this ticket exists to prevent: a free Game
			// stays free no matter how many people come.
			name: "a free game is free for any party size", feeCents: 0, currency: "", guestCount: 3, wantCents: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			g := gameWithFee(t, tc.feeCents, tc.currency, 5)
			got := domain.ExpectedAmount(g, tc.guestCount)

			if got.Cents != tc.wantCents {
				t.Fatalf("ExpectedAmount cents = %d, want %d", got.Cents, tc.wantCents)
			}
			// The currency must ride along with the amount — ADR-0005's
			// whole point is that a bare cents figure never exists in the
			// domain.
			if got.Currency != tc.currency {
				t.Fatalf("ExpectedAmount currency = %q, want %q", got.Currency, tc.currency)
			}
		})
	}
}

// Register freezes the owed amount onto the Registration rather than
// leaving it to be recomputed later.
//
// The Product Owner chose "freeze, and lock guests once paid" on
// 2026-09-04. Freezing is what makes the answer to "what does this player
// owe" stable: it is the figure agreed when they registered, not one
// re-derived from a Game whose EntryFee the Host may since have changed.
func TestRegisterFreezesAmountOwed(t *testing.T) {
	t.Parallel()

	g := gameWithFee(t, 2500, "USD", 3)

	reg, err := domain.Register(g, nil, "player-1", 2)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	if reg.AmountOwed.Cents != 7500 {
		t.Fatalf("AmountOwed cents = %d, want 7500 (2500 × 3 heads)", reg.AmountOwed.Cents)
	}
	if reg.AmountOwed.Currency != "USD" {
		t.Fatalf("AmountOwed currency = %q, want USD", reg.AmountOwed.Currency)
	}
}

// The frozen figure must not drift when the Game's fee later changes. This
// is the property that makes #297's server-side validation meaningful: it
// compares a payment against what was actually agreed, not against a
// number that moved underneath the player.
func TestRegisterAmountOwedDoesNotFollowALaterFeeChange(t *testing.T) {
	t.Parallel()

	g := gameWithFee(t, 2500, "USD", 3)

	reg, err := domain.Register(g, nil, "player-1", 1)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if reg.AmountOwed.Cents != 5000 {
		t.Fatalf("AmountOwed cents = %d, want 5000", reg.AmountOwed.Cents)
	}

	// The Host raises the price after this player registered.
	g.EntryFee = domain.Money{Cents: 9900, Currency: "USD"}

	if reg.AmountOwed.Cents != 5000 {
		t.Fatalf("AmountOwed changed to %d after the Game's fee changed — it must be frozen at registration", reg.AmountOwed.Cents)
	}
}

// A free Game produces a free Registration, and IsFree is what says so —
// not a bare `Cents == 0` comparison, per Money.IsFree's own doc comment.
func TestRegisterFreeGameOwesNothing(t *testing.T) {
	t.Parallel()

	g := gameWithFee(t, 0, "", 3)

	reg, err := domain.Register(g, nil, "player-1", 3)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if !reg.AmountOwed.IsFree() {
		t.Fatalf("AmountOwed = %+v, want free", reg.AmountOwed)
	}
}

// Guest-count validation still runs before the amount is computed, so an
// over-allowance registration is rejected rather than priced.
func TestRegisterRejectsOverAllowanceBeforePricing(t *testing.T) {
	t.Parallel()

	g := gameWithFee(t, 2500, "USD", 1)

	_, err := domain.Register(g, nil, "player-1", 2)
	if !errors.Is(err, domain.ErrGuestAllowanceExceeded) {
		t.Fatalf("Register with too many guests = %v, want %v", err, domain.ErrGuestAllowanceExceeded)
	}
}
