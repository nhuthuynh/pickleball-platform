package domain_test

import (
	"errors"
	"testing"

	"github.com/nhuthuynh/white-label/internal/competitions/domain"
)

// TestMoney_Validate covers the EntryFee boundary cases the T9.1 ticket
// requires: a zero amount is a real value (a free competition), not a
// placeholder, and a non-empty amount with an empty currency code is
// rejected — the ADR-0005 rule that amount and currency are coupled.
func TestMoney_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		money   domain.Money
		wantErr error
	}{
		{
			name:  "zero amount with no currency is a legal free competition",
			money: domain.Money{},
		},
		{
			name:  "zero amount with a currency is legal",
			money: domain.Money{AmountCents: 0, CurrencyCode: "GBP"},
		},
		{
			name:  "positive amount with a valid currency is legal",
			money: domain.Money{AmountCents: 1500, CurrencyCode: "GBP"},
		},
		{
			name:    "non-empty amount with an empty currency is rejected",
			money:   domain.Money{AmountCents: 1500, CurrencyCode: ""},
			wantErr: domain.ErrInvalidMoney,
		},
		{
			name:    "negative amount is rejected",
			money:   domain.Money{AmountCents: -1, CurrencyCode: "GBP"},
			wantErr: domain.ErrInvalidMoney,
		},
		{
			name:    "malformed currency code is rejected",
			money:   domain.Money{AmountCents: 1500, CurrencyCode: "pounds"},
			wantErr: domain.ErrInvalidMoney,
		},
		{
			name:    "lowercase currency code is rejected",
			money:   domain.Money{AmountCents: 1500, CurrencyCode: "gbp"},
			wantErr: domain.ErrInvalidMoney,
		},
		{
			name:    "malformed currency code is rejected even at a zero amount",
			money:   domain.Money{AmountCents: 0, CurrencyCode: "gbp"},
			wantErr: domain.ErrInvalidMoney,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.money.Validate()
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

// TestMoney_IsZeroFee proves a zero-amount EntryFee is distinguishable as a
// real "free competition" value rather than being conflated with "unset" by
// callers (T9.3's app layer and T9.4's proto mapping both need to render
// "Free" rather than an empty price).
func TestMoney_IsZeroFee(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		money domain.Money
		want  bool
	}{
		{"zero value is free", domain.Money{}, true},
		{"zero amount with a currency is free", domain.Money{AmountCents: 0, CurrencyCode: "GBP"}, true},
		{"positive amount is not free", domain.Money{AmountCents: 1, CurrencyCode: "GBP"}, false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.money.IsZero(); got != tt.want {
				t.Fatalf("IsZero() = %v, want %v", got, tt.want)
			}
		})
	}
}
