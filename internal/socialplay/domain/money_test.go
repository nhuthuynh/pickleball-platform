package domain_test

import (
	"errors"
	"testing"

	"github.com/nhuthuynh/white-label/internal/socialplay/domain"
)

// TestNewGame_EntryFee is T9.2's core table: a Game carries a real
// EntryFee Money, and NewGame validates it. The cases that matter and why:
//
//   - A ZERO amount is legal and means a free Game. This is the single most
//     important assertion in this file: zero is a real, host-chosen value —
//     a free game — not a sentinel for "no price set" and not an error. The
//     placeholder it replaces (T8.10's PLACEHOLDER_REGISTRATION_FEE_CENTS,
//     $10.00) had no way to express "free" at all.
//   - A negative amount is rejected. There is no such thing as a Game that
//     pays the player to attend; this is the domain-side twin of the
//     migration's CHECK (entry_fee_cents >= 0) (CLAUDE.md rule 4 — the
//     invariant is expressed in both places and they must stay in sync).
//   - A non-empty amount with an empty currency code is rejected: ADR-0005's
//     whole point is that amount and currency are coupled in the domain type
//     from day one, so a bare cents figure with no currency must never be
//     constructible.
//
// Deliberately distinct from internal/payments/domain's Money validation in
// one respect: payments.NewPayment rejects amount.Cents <= 0 (a Payment of
// nothing is meaningless), whereas a Game's EntryFee of zero is a real
// product state. Same shape, different rule — which is exactly why each
// context owns its own Money (ADR-0005, CLAUDE.md rule 2/3) rather than
// sharing one package.
func TestNewGame_EntryFee(t *testing.T) {
	t.Parallel()

	validRange := mustRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")

	tests := []struct {
		name     string
		entryFee domain.Money
		wantErr  error
	}{
		{
			name:     "zero amount with a currency is a free game, not an error",
			entryFee: domain.Money{Cents: 0, Currency: "USD"},
			wantErr:  nil,
		},
		{
			name:     "zero amount with an empty currency is accepted",
			entryFee: domain.Money{Cents: 0, Currency: ""},
			wantErr:  nil,
		},
		{
			name:     "positive amount with a currency is accepted",
			entryFee: domain.Money{Cents: 1500, Currency: "USD"},
			wantErr:  nil,
		},
		{
			name:     "non-empty amount with an empty currency is rejected",
			entryFee: domain.Money{Cents: 1500, Currency: ""},
			wantErr:  domain.ErrInvalidMoney,
		},
		{
			name:     "negative amount is rejected",
			entryFee: domain.Money{Cents: -1, Currency: "USD"},
			wantErr:  domain.ErrInvalidMoney,
		},
		{
			name:     "negative amount with an empty currency is rejected",
			entryFee: domain.Money{Cents: -500, Currency: ""},
			wantErr:  domain.ErrInvalidMoney,
		},
		{
			name:     "malformed currency code is rejected",
			entryFee: domain.Money{Cents: 1500, Currency: "dollars"},
			wantErr:  domain.ErrInvalidMoney,
		},
		{
			name:     "lowercase currency code is rejected",
			entryFee: domain.Money{Cents: 1500, Currency: "usd"},
			wantErr:  domain.ErrInvalidMoney,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g, err := domain.NewGame("g1", "host-1", "facility-1", "venue-1", validCourtIDs(), validRange, 4, domain.PaymentMethodEither, 0, tt.entryFee)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got err %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && g.EntryFee != tt.entryFee {
				t.Fatalf("EntryFee = %+v, want %+v (NewGame must store the fee it was given verbatim)", g.EntryFee, tt.entryFee)
			}
		})
	}
}

// TestMoney_IsFree proves the "free game" predicate reads as a real product
// state rather than a magic comparison scattered across call sites — the UI
// renders the word "Free" off exactly this condition (T9.2 non-functional
// requirement: a zero price is a real state, never blank or a bare "$0.00").
func TestMoney_IsFree(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		money domain.Money
		want  bool
	}{
		{"zero cents is free", domain.Money{Cents: 0, Currency: "USD"}, true},
		{"zero cents without a currency is free", domain.Money{}, true},
		{"positive cents is not free", domain.Money{Cents: 1, Currency: "USD"}, false},
		{"large positive cents is not free", domain.Money{Cents: 25000, Currency: "USD"}, false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.money.IsFree(); got != tt.want {
				t.Fatalf("IsFree() = %v, want %v", got, tt.want)
			}
		})
	}
}
