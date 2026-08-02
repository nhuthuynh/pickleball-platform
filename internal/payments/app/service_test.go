package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nhuthuynh/white-label/internal/payments/adapter/stripestub"
	"github.com/nhuthuynh/white-label/internal/payments/app"
	"github.com/nhuthuynh/white-label/internal/payments/domain"
)

func fixtureAmount() domain.Money {
	return domain.Money{Cents: 3000, Currency: "USD"}
}

// TestCreateOnlinePayment_Succeeds proves the happy path: an online
// Payment is built (unpaid), an intent is created against the stub
// processor, and the returned intent reference is stored as
// StripeReference — while the Payment itself stays unpaid, because
// creating an intent is not the same as capturing funds.
func TestCreateOnlinePayment_Succeeds(t *testing.T) {
	t.Parallel()

	proc := stripestub.NewProcessor()
	svc := app.NewService(&fixedIDs{ids: []string{"pay-1"}}, proc)

	p, err := svc.CreateOnlinePayment(context.Background(), app.CreateOnlinePaymentInput{
		PayableType: domain.PayableTypeBooking,
		PayableID:   "booking-1",
		Amount:      fixtureAmount(),
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if p.ID != "pay-1" {
		t.Fatalf("ID = %q, want pay-1", p.ID)
	}
	if p.Status != domain.StatusUnpaid {
		t.Fatalf("Status = %v, want unpaid", p.Status)
	}
	if p.Method != domain.MethodOnline {
		t.Fatalf("Method = %v, want online", p.Method)
	}
	if p.StripeReference == "" {
		t.Fatal("StripeReference must be populated from CreateIntent")
	}
}

// TestCreateOnlinePayment_InvalidInputRejectedByDomain proves
// CreateOnlinePayment delegates validation to domain.NewPayment rather
// than duplicating it — an invalid amount is rejected before the
// processor is ever called.
func TestCreateOnlinePayment_InvalidInputRejectedByDomain(t *testing.T) {
	t.Parallel()

	proc := stripestub.NewProcessor()
	svc := app.NewService(&fixedIDs{ids: []string{"pay-1"}}, proc)

	_, err := svc.CreateOnlinePayment(context.Background(), app.CreateOnlinePaymentInput{
		PayableType: domain.PayableTypeBooking,
		PayableID:   "booking-1",
		Amount:      domain.Money{Cents: 0, Currency: "USD"},
	})
	if !errors.Is(err, domain.ErrInvalidAmount) {
		t.Fatalf("got err %v, want %v", err, domain.ErrInvalidAmount)
	}
}

// TestCreateOnlinePayment_ProcessorUnavailable proves a processor failure
// at intent-creation time surfaces as the port's sentinel, not a panic or
// a partially-built Payment being returned as if it succeeded.
func TestCreateOnlinePayment_ProcessorUnavailable(t *testing.T) {
	t.Parallel()

	proc := stripestub.NewProcessor()
	proc.SeedUnavailable("booking-1")
	svc := app.NewService(&fixedIDs{ids: []string{"pay-1"}}, proc)

	_, err := svc.CreateOnlinePayment(context.Background(), app.CreateOnlinePaymentInput{
		PayableType: domain.PayableTypeBooking,
		PayableID:   "booking-1",
		Amount:      fixtureAmount(),
	})
	if !errors.Is(err, domain.ErrPaymentProcessorUnavailable) {
		t.Fatalf("got err %v, want %v", err, domain.ErrPaymentProcessorUnavailable)
	}
}

// TestConfirmOnlinePayment_Succeeds proves the happy path: capturing a
// created intent transitions the Payment to paid via MarkPaid.
func TestConfirmOnlinePayment_Succeeds(t *testing.T) {
	t.Parallel()

	proc := stripestub.NewProcessor()
	svc := app.NewService(&fixedIDs{ids: []string{"pay-1"}}, proc)

	p, err := svc.CreateOnlinePayment(context.Background(), app.CreateOnlinePaymentInput{
		PayableType: domain.PayableTypeBooking,
		PayableID:   "booking-1",
		Amount:      fixtureAmount(),
	})
	if err != nil {
		t.Fatalf("unexpected err creating: %v", err)
	}

	confirmed, err := svc.ConfirmOnlinePayment(context.Background(), p)
	if err != nil {
		t.Fatalf("unexpected err confirming: %v", err)
	}
	if confirmed.Status != domain.StatusPaid {
		t.Fatalf("Status = %v, want paid", confirmed.Status)
	}
	if confirmed.StripeReference != p.StripeReference {
		t.Fatalf("StripeReference = %q, want unchanged %q", confirmed.StripeReference, p.StripeReference)
	}
}

// TestConfirmOnlinePayment_Declined is the ticket's required test: a
// declined card must leave the Payment unchanged (still unpaid), not
// error out as an illegal state transition. This is the case most likely
// to be implemented wrong by reaching for MarkPaid's error path instead
// of just not calling MarkPaid at all.
func TestConfirmOnlinePayment_Declined(t *testing.T) {
	t.Parallel()

	proc := stripestub.NewProcessor()
	proc.SeedDecline("booking-1")
	svc := app.NewService(&fixedIDs{ids: []string{"pay-1"}}, proc)

	p, err := svc.CreateOnlinePayment(context.Background(), app.CreateOnlinePaymentInput{
		PayableType: domain.PayableTypeBooking,
		PayableID:   "booking-1",
		Amount:      fixtureAmount(),
	})
	if err != nil {
		t.Fatalf("unexpected err creating: %v", err)
	}

	before := p

	result, err := svc.ConfirmOnlinePayment(context.Background(), p)
	if !errors.Is(err, domain.ErrPaymentDeclined) {
		t.Fatalf("got err %v, want %v", err, domain.ErrPaymentDeclined)
	}
	if result.Status != domain.StatusUnpaid {
		t.Fatalf("Status = %v, want unchanged (unpaid)", result.Status)
	}
	if result != before {
		t.Fatalf("Payment mutated on decline: got %+v, want unchanged %+v", result, before)
	}
}

// TestConfirmOnlinePayment_ProcessorUnavailable proves a processor-level
// failure (distinct from a decline) also leaves the Payment unchanged,
// not just the decline path.
func TestConfirmOnlinePayment_ProcessorUnavailable(t *testing.T) {
	t.Parallel()

	proc := stripestub.NewProcessor()
	svc := app.NewService(&fixedIDs{ids: []string{"pay-1"}}, proc)

	p, err := svc.CreateOnlinePayment(context.Background(), app.CreateOnlinePaymentInput{
		PayableType: domain.PayableTypeBooking,
		PayableID:   "booking-1",
		Amount:      fixtureAmount(),
	})
	if err != nil {
		t.Fatalf("unexpected err creating: %v", err)
	}
	// Corrupt the reference so the stub treats it as unknown, simulating
	// a processor-side failure distinct from a decline.
	p.StripeReference = "pi_does_not_exist"

	result, err := svc.ConfirmOnlinePayment(context.Background(), p)
	if !errors.Is(err, domain.ErrPaymentProcessorUnavailable) {
		t.Fatalf("got err %v, want %v", err, domain.ErrPaymentProcessorUnavailable)
	}
	if result.Status != domain.StatusUnpaid {
		t.Fatalf("Status = %v, want unchanged (unpaid)", result.Status)
	}
}

// TestConfirmOnlinePayment_AlreadyPaidIsIllegal proves ConfirmOnlinePayment
// still routes through Payment.MarkPaid's own transition guard for a
// Payment that is not unpaid — confirming an already-paid Payment again is
// an illegal transition, not a silent no-op or a second successful
// capture.
func TestConfirmOnlinePayment_AlreadyPaidIsIllegal(t *testing.T) {
	t.Parallel()

	proc := stripestub.NewProcessor()
	svc := app.NewService(&fixedIDs{ids: []string{"pay-1"}}, proc)

	p, err := svc.CreateOnlinePayment(context.Background(), app.CreateOnlinePaymentInput{
		PayableType: domain.PayableTypeBooking,
		PayableID:   "booking-1",
		Amount:      fixtureAmount(),
	})
	if err != nil {
		t.Fatalf("unexpected err creating: %v", err)
	}
	paid, err := svc.ConfirmOnlinePayment(context.Background(), p)
	if err != nil {
		t.Fatalf("unexpected err confirming: %v", err)
	}

	_, err = svc.ConfirmOnlinePayment(context.Background(), paid)
	if !errors.Is(err, domain.ErrIllegalStatusTransition) {
		t.Fatalf("got err %v, want %v", err, domain.ErrIllegalStatusTransition)
	}
}
