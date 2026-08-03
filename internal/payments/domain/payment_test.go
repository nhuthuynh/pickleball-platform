package domain_test

import (
	"errors"
	"testing"

	"github.com/nhuthuynh/white-label/internal/payments/domain"
)

// fixtureMoney returns a valid, positive Money value for use as a base
// fixture that individual test cases mutate to explore boundaries.
func fixtureMoney() domain.Money {
	return domain.Money{Cents: 1500, Currency: "USD"}
}

// TestNewPayment_Valid proves the happy path: a valid NewPayment call
// produces an "unpaid" Payment carrying exactly what was passed in.
func TestNewPayment_Valid(t *testing.T) {
	t.Parallel()

	p, err := domain.NewPayment("p1", domain.PayableTypeBooking, "booking-1", fixtureMoney(), domain.MethodOnline, "")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if p.ID != "p1" {
		t.Fatalf("ID = %q, want p1", p.ID)
	}
	if p.PayableType != domain.PayableTypeBooking {
		t.Fatalf("PayableType = %v, want %v", p.PayableType, domain.PayableTypeBooking)
	}
	if p.PayableID != "booking-1" {
		t.Fatalf("PayableID = %q, want booking-1", p.PayableID)
	}
	if p.Amount != fixtureMoney() {
		t.Fatalf("Amount = %+v, want %+v", p.Amount, fixtureMoney())
	}
	if p.Method != domain.MethodOnline {
		t.Fatalf("Method = %v, want %v", p.Method, domain.MethodOnline)
	}
	if p.Status != domain.StatusUnpaid {
		t.Fatalf("Status = %v, want %v", p.Status, domain.StatusUnpaid)
	}
	if p.StripeReference != "" {
		t.Fatalf("StripeReference = %q, want empty", p.StripeReference)
	}
	if p.RecordedByUserID != "" {
		t.Fatalf("RecordedByUserID = %q, want empty", p.RecordedByUserID)
	}
}

// TestNewPayment_OfflineCarriesRecorder proves the offline-recording shape:
// RecordedByUserID is carried through NewPayment for the offline path.
func TestNewPayment_OfflineCarriesRecorder(t *testing.T) {
	t.Parallel()

	p, err := domain.NewPayment("p1", domain.PayableTypeRegistration, "reg-1", fixtureMoney(), domain.MethodOffline, "game-admin-1")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if p.RecordedByUserID != "game-admin-1" {
		t.Fatalf("RecordedByUserID = %q, want game-admin-1", p.RecordedByUserID)
	}
	if p.Method != domain.MethodOffline {
		t.Fatalf("Method = %v, want %v", p.Method, domain.MethodOffline)
	}
}

// TestNewPayment_Validation is the required boundary coverage: empty
// PayableID, invalid PayableType, zero/negative amount, and a malformed
// currency code must all be rejected with their own distinct sentinel
// rather than a generic validation error.
func TestNewPayment_Validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		payableType domain.PayableType
		payableID   string
		amount      domain.Money
		wantErr     error
	}{
		{
			name:        "valid booking payable",
			payableType: domain.PayableTypeBooking,
			payableID:   "booking-1",
			amount:      domain.Money{Cents: 1500, Currency: "USD"},
			wantErr:     nil,
		},
		{
			name:        "valid registration payable",
			payableType: domain.PayableTypeRegistration,
			payableID:   "reg-1",
			amount:      domain.Money{Cents: 1500, Currency: "USD"},
			wantErr:     nil,
		},
		{
			name:        "valid no_show_fee payable",
			payableType: domain.PayableTypeNoShowFee,
			payableID:   "reg-1",
			amount:      domain.Money{Cents: 1500, Currency: "USD"},
			wantErr:     nil,
		},
		{
			name:        "empty payable id is rejected",
			payableType: domain.PayableTypeBooking,
			payableID:   "",
			amount:      domain.Money{Cents: 1500, Currency: "USD"},
			wantErr:     domain.ErrEmptyPayableID,
		},
		{
			name:        "unknown payable type is rejected",
			payableType: domain.PayableType("subscription"),
			payableID:   "booking-1",
			amount:      domain.Money{Cents: 1500, Currency: "USD"},
			wantErr:     domain.ErrInvalidPayableType,
		},
		{
			name:        "empty payable type is rejected",
			payableType: domain.PayableType(""),
			payableID:   "booking-1",
			amount:      domain.Money{Cents: 1500, Currency: "USD"},
			wantErr:     domain.ErrInvalidPayableType,
		},
		{
			name:        "zero amount is rejected",
			payableType: domain.PayableTypeBooking,
			payableID:   "booking-1",
			amount:      domain.Money{Cents: 0, Currency: "USD"},
			wantErr:     domain.ErrInvalidAmount,
		},
		{
			name:        "negative amount is rejected",
			payableType: domain.PayableTypeBooking,
			payableID:   "booking-1",
			amount:      domain.Money{Cents: -100, Currency: "USD"},
			wantErr:     domain.ErrInvalidAmount,
		},
		{
			name:        "empty currency is rejected",
			payableType: domain.PayableTypeBooking,
			payableID:   "booking-1",
			amount:      domain.Money{Cents: 1500, Currency: ""},
			wantErr:     domain.ErrInvalidCurrency,
		},
		{
			name:        "too-short currency code is rejected",
			payableType: domain.PayableTypeBooking,
			payableID:   "booking-1",
			amount:      domain.Money{Cents: 1500, Currency: "US"},
			wantErr:     domain.ErrInvalidCurrency,
		},
		{
			name:        "too-long currency code is rejected",
			payableType: domain.PayableTypeBooking,
			payableID:   "booking-1",
			amount:      domain.Money{Cents: 1500, Currency: "USDD"},
			wantErr:     domain.ErrInvalidCurrency,
		},
		{
			name:        "lowercase currency code is rejected",
			payableType: domain.PayableTypeBooking,
			payableID:   "booking-1",
			amount:      domain.Money{Cents: 1500, Currency: "usd"},
			wantErr:     domain.ErrInvalidCurrency,
		},
		{
			name:        "numeric currency code is rejected",
			payableType: domain.PayableTypeBooking,
			payableID:   "booking-1",
			amount:      domain.Money{Cents: 1500, Currency: "12A"},
			wantErr:     domain.ErrInvalidCurrency,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := domain.NewPayment("p1", tt.payableType, tt.payableID, tt.amount, domain.MethodOnline, "")
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got err %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// TestPayableType_IsValid proves the extensible enum accepts exactly the
// three T6 values and nothing else.
func TestPayableType_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		pt   domain.PayableType
		want bool
	}{
		{"booking is valid", domain.PayableTypeBooking, true},
		{"registration is valid", domain.PayableTypeRegistration, true},
		{"no_show_fee is valid", domain.PayableTypeNoShowFee, true},
		{"recurring_hire is not yet built, invalid", domain.PayableType("recurring_hire"), false},
		{"subscription is not yet built, invalid", domain.PayableType("subscription"), false},
		{"empty is invalid", domain.PayableType(""), false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.pt.IsValid(); got != tt.want {
				t.Fatalf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestPayment_StatusTransitions is the required exhaustive
// (fromStatus, transition) matrix: all three statuses crossed with both
// transition methods, so the illegal-transition surface is fully covered
// rather than spot-checked.
func TestPayment_StatusTransitions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		from       domain.Status
		transition string // "MarkPaid" or "Refund"
		wantErr    error
		wantStatus domain.Status
	}{
		{
			name: "unpaid MarkPaid succeeds", from: domain.StatusUnpaid, transition: "MarkPaid",
			wantErr: nil, wantStatus: domain.StatusPaid,
		},
		{
			name: "unpaid Refund is illegal", from: domain.StatusUnpaid, transition: "Refund",
			wantErr: domain.ErrIllegalStatusTransition, wantStatus: domain.StatusUnpaid,
		},
		{
			name: "paid MarkPaid is illegal", from: domain.StatusPaid, transition: "MarkPaid",
			wantErr: domain.ErrIllegalStatusTransition, wantStatus: domain.StatusPaid,
		},
		{
			name: "paid Refund succeeds", from: domain.StatusPaid, transition: "Refund",
			wantErr: nil, wantStatus: domain.StatusRefunded,
		},
		{
			name: "refunded MarkPaid is illegal", from: domain.StatusRefunded, transition: "MarkPaid",
			wantErr: domain.ErrIllegalStatusTransition, wantStatus: domain.StatusRefunded,
		},
		{
			name: "refunded Refund is illegal", from: domain.StatusRefunded, transition: "Refund",
			wantErr: domain.ErrIllegalStatusTransition, wantStatus: domain.StatusRefunded,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p, err := domain.NewPayment("p1", domain.PayableTypeBooking, "booking-1", fixtureMoney(), domain.MethodOnline, "")
			if err != nil {
				t.Fatalf("unexpected err building fixture: %v", err)
			}
			p.Status = tt.from

			switch tt.transition {
			case "MarkPaid":
				err = p.MarkPaid(domain.MethodOnline, "pi_ref_123")
			case "Refund":
				err = p.Refund()
			default:
				t.Fatalf("unknown transition %q", tt.transition)
			}

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got err %v, want %v", err, tt.wantErr)
			}
			if p.Status != tt.wantStatus {
				t.Fatalf("status = %v, want %v", p.Status, tt.wantStatus)
			}
		})
	}
}

// TestPayment_MarkPaid_RecordsMethodAndReference proves MarkPaid actually
// stores the method/reference it's given, not just flipping the status.
func TestPayment_MarkPaid_RecordsMethodAndReference(t *testing.T) {
	t.Parallel()

	p, err := domain.NewPayment("p1", domain.PayableTypeBooking, "booking-1", fixtureMoney(), domain.MethodOnline, "")
	if err != nil {
		t.Fatalf("unexpected err building fixture: %v", err)
	}

	if err := p.MarkPaid(domain.MethodOnline, "pi_abc123"); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if p.StripeReference != "pi_abc123" {
		t.Fatalf("StripeReference = %q, want pi_abc123", p.StripeReference)
	}
	if p.Status != domain.StatusPaid {
		t.Fatalf("Status = %v, want %v", p.Status, domain.StatusPaid)
	}
}

// TestPayment_UnpaidToRefundedDirectlyIsIllegal is the explicitly-required
// test (T6.1 instructions) for the transition most likely to be waved
// through as "should probably be fine": an unpaid Payment cannot be
// refunded directly, because nothing was paid to refund.
func TestPayment_UnpaidToRefundedDirectlyIsIllegal(t *testing.T) {
	t.Parallel()

	p, err := domain.NewPayment("p1", domain.PayableTypeBooking, "booking-1", fixtureMoney(), domain.MethodOnline, "")
	if err != nil {
		t.Fatalf("unexpected err building fixture: %v", err)
	}
	if p.Status != domain.StatusUnpaid {
		t.Fatalf("fixture Status = %v, want unpaid", p.Status)
	}

	err = p.Refund()
	if !errors.Is(err, domain.ErrIllegalStatusTransition) {
		t.Fatalf("got err %v, want %v", err, domain.ErrIllegalStatusTransition)
	}
	if p.Status != domain.StatusUnpaid {
		t.Fatalf("Status = %v, want unchanged (unpaid)", p.Status)
	}
}

// TestPayment_RefundedIsTerminal proves refunded has no way out: calling
// either transition method again returns ErrIllegalStatusTransition, not a
// silent no-op, and the Payment is left untouched.
func TestPayment_RefundedIsTerminal(t *testing.T) {
	t.Parallel()

	p, err := domain.NewPayment("p1", domain.PayableTypeBooking, "booking-1", fixtureMoney(), domain.MethodOnline, "")
	if err != nil {
		t.Fatalf("unexpected err building fixture: %v", err)
	}
	if err := p.MarkPaid(domain.MethodOnline, "pi_ref"); err != nil {
		t.Fatalf("unexpected err marking paid: %v", err)
	}
	if err := p.Refund(); err != nil {
		t.Fatalf("unexpected err refunding: %v", err)
	}

	if err := p.MarkPaid(domain.MethodOnline, "pi_other"); !errors.Is(err, domain.ErrIllegalStatusTransition) {
		t.Fatalf("MarkPaid after refund: got err %v, want %v", err, domain.ErrIllegalStatusTransition)
	}
	if err := p.Refund(); !errors.Is(err, domain.ErrIllegalStatusTransition) {
		t.Fatalf("Refund after refund: got err %v, want %v", err, domain.ErrIllegalStatusTransition)
	}
	if p.Status != domain.StatusRefunded {
		t.Fatalf("Status = %v, want refunded (unchanged)", p.Status)
	}
}

// TestEnsureOnePaymentPerPayable proves the distinctness pre-check: a
// (PayableType, PayableID) pair may have at most one Payment, mirroring
// EnsureNoConflict's role for Booking.
func TestEnsureOnePaymentPerPayable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		candidate domain.Payment
		existing  []domain.Payment
		wantErr   error
	}{
		{
			name:      "no existing payments never conflicts",
			candidate: domain.Payment{ID: "p1", PayableType: domain.PayableTypeBooking, PayableID: "booking-1"},
			existing:  nil,
			wantErr:   nil,
		},
		{
			name:      "same payable type and id conflicts",
			candidate: domain.Payment{ID: "p2", PayableType: domain.PayableTypeBooking, PayableID: "booking-1"},
			existing: []domain.Payment{
				{ID: "p1", PayableType: domain.PayableTypeBooking, PayableID: "booking-1"},
			},
			wantErr: domain.ErrPaymentAlreadyRecorded,
		},
		{
			name:      "same payable id but different payable type does not conflict",
			candidate: domain.Payment{ID: "p2", PayableType: domain.PayableTypeNoShowFee, PayableID: "reg-1"},
			existing: []domain.Payment{
				{ID: "p1", PayableType: domain.PayableTypeRegistration, PayableID: "reg-1"},
			},
			wantErr: nil,
		},
		{
			name:      "different payable id same type does not conflict",
			candidate: domain.Payment{ID: "p2", PayableType: domain.PayableTypeBooking, PayableID: "booking-2"},
			existing: []domain.Payment{
				{ID: "p1", PayableType: domain.PayableTypeBooking, PayableID: "booking-1"},
			},
			wantErr: nil,
		},
		{
			name:      "candidate does not conflict with itself",
			candidate: domain.Payment{ID: "p1", PayableType: domain.PayableTypeBooking, PayableID: "booking-1"},
			existing: []domain.Payment{
				{ID: "p1", PayableType: domain.PayableTypeBooking, PayableID: "booking-1"},
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := domain.EnsureOnePaymentPerPayable(tt.candidate, tt.existing)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got err %v, want %v", err, tt.wantErr)
			}
		})
	}
}
