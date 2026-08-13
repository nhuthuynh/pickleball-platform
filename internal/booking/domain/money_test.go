package domain_test

import (
	"errors"
	"testing"

	"github.com/nhuthuynh/white-label/internal/booking/domain"
)

// TestNewMoney_Validation proves Money's constructor rejects the two ways a
// money value can be meaningless: a non-positive amount, or an amount with
// no currency attached. Money is context-local to booking (T9's ruling
// against a shared-kernel Money type, restated for T11.1) — this file's
// tests are booking's own, independent of any other context.
func TestNewMoney_Validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		amountCents  int64
		currencyCode string
		wantErr      error
	}{
		{"positive amount with currency is valid", 1500, "USD", nil},
		{"zero amount is rejected", 0, "USD", domain.ErrInvalidMoney},
		{"negative amount is rejected", -100, "USD", domain.ErrInvalidMoney},
		{"empty currency code is rejected", 1500, "", domain.ErrInvalidMoney},
		{"zero amount and empty currency is rejected", 0, "", domain.ErrInvalidMoney},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := domain.NewMoney(tt.amountCents, tt.currencyCode)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got err %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil {
				if got.AmountCents != tt.amountCents || got.CurrencyCode != tt.currencyCode {
					t.Errorf("got %+v, want AmountCents=%d CurrencyCode=%q", got, tt.amountCents, tt.currencyCode)
				}
			}
		})
	}
}

func TestMoney_IsPositive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		m    domain.Money
		want bool
	}{
		{"positive amount with currency", domain.Money{AmountCents: 500, CurrencyCode: "USD"}, true},
		{"zero amount", domain.Money{AmountCents: 0, CurrencyCode: "USD"}, false},
		{"negative amount", domain.Money{AmountCents: -1, CurrencyCode: "USD"}, false},
		{"positive amount but no currency", domain.Money{AmountCents: 500, CurrencyCode: ""}, false},
		{"zero value Money", domain.Money{}, false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.m.IsPositive(); got != tt.want {
				t.Errorf("IsPositive() = %v, want %v", got, tt.want)
			}
		})
	}
}
