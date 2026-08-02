package stripestub_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nhuthuynh/white-label/internal/payments/adapter/stripestub"
	"github.com/nhuthuynh/white-label/internal/payments/domain"
	"github.com/nhuthuynh/white-label/internal/payments/port"
)

// fixtureAmount returns a valid, positive Money value for tests.
func fixtureAmount() domain.Money {
	return domain.Money{Cents: 2000, Currency: "USD"}
}

// TestProcessor_SatisfiesPort proves the stub actually implements
// port.PaymentProcessor — a compile-time check as much as a runtime one.
func TestProcessor_SatisfiesPort(t *testing.T) {
	t.Parallel()
	var _ port.PaymentProcessor = stripestub.NewProcessor()
}

// TestProcessor_CreateIntent_Succeeds proves the default, unseeded
// behaviour: CreateIntent succeeds deterministically with no network call
// and no STRIPE_* env var involved.
func TestProcessor_CreateIntent_Succeeds(t *testing.T) {
	t.Parallel()

	p := stripestub.NewProcessor()
	ref, err := p.CreateIntent(context.Background(), fixtureAmount(), "booking-1")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if ref == "" {
		t.Fatal("intentRef must not be empty")
	}
}

// TestProcessor_CreateIntent_DistinctRefs proves two intents created for
// different payables get distinct references.
func TestProcessor_CreateIntent_DistinctRefs(t *testing.T) {
	t.Parallel()

	p := stripestub.NewProcessor()
	ref1, err := p.CreateIntent(context.Background(), fixtureAmount(), "booking-1")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	ref2, err := p.CreateIntent(context.Background(), fixtureAmount(), "booking-2")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if ref1 == ref2 {
		t.Fatalf("expected distinct refs, got %q twice", ref1)
	}
}

// TestProcessor_CapturePayment_Succeeds proves the default, unseeded
// behaviour: capturing a previously created intent succeeds.
func TestProcessor_CapturePayment_Succeeds(t *testing.T) {
	t.Parallel()

	p := stripestub.NewProcessor()
	ref, err := p.CreateIntent(context.Background(), fixtureAmount(), "booking-1")
	if err != nil {
		t.Fatalf("unexpected err creating intent: %v", err)
	}
	if err := p.CapturePayment(context.Background(), ref); err != nil {
		t.Fatalf("unexpected err capturing: %v", err)
	}
}

// TestProcessor_CapturePayment_UnknownRef proves capturing an intent
// reference the stub never created is a processor-unavailable failure, not
// a panic or a silent success.
func TestProcessor_CapturePayment_UnknownRef(t *testing.T) {
	t.Parallel()

	p := stripestub.NewProcessor()
	err := p.CapturePayment(context.Background(), "pi_never_created")
	if !errors.Is(err, domain.ErrPaymentProcessorUnavailable) {
		t.Fatalf("got err %v, want %v", err, domain.ErrPaymentProcessorUnavailable)
	}
}

// TestProcessor_SeedDecline_DeclinesCapture is the deterministic-failure
// seeding this ticket requires: every call succeeds unless a test seeds a
// specific failure, so this proves the seed actually takes effect.
func TestProcessor_SeedDecline_DeclinesCapture(t *testing.T) {
	t.Parallel()

	p := stripestub.NewProcessor()
	p.SeedDecline("booking-1")

	ref, err := p.CreateIntent(context.Background(), fixtureAmount(), "booking-1")
	if err != nil {
		t.Fatalf("unexpected err creating intent: %v", err)
	}

	err = p.CapturePayment(context.Background(), ref)
	if !errors.Is(err, domain.ErrPaymentDeclined) {
		t.Fatalf("got err %v, want %v", err, domain.ErrPaymentDeclined)
	}
}

// TestProcessor_SeedDecline_OnlyAffectsSeededPayable proves the seed is
// scoped to the payable it was seeded for, not global.
func TestProcessor_SeedDecline_OnlyAffectsSeededPayable(t *testing.T) {
	t.Parallel()

	p := stripestub.NewProcessor()
	p.SeedDecline("booking-1")

	ref, err := p.CreateIntent(context.Background(), fixtureAmount(), "booking-2")
	if err != nil {
		t.Fatalf("unexpected err creating intent: %v", err)
	}
	if err := p.CapturePayment(context.Background(), ref); err != nil {
		t.Fatalf("unexpected err capturing unseeded payable: %v", err)
	}
}

// TestProcessor_SeedUnavailable_FailsCreateIntent proves the stub can also
// simulate the processor itself being unreachable at intent-creation time.
func TestProcessor_SeedUnavailable_FailsCreateIntent(t *testing.T) {
	t.Parallel()

	p := stripestub.NewProcessor()
	p.SeedUnavailable("booking-1")

	_, err := p.CreateIntent(context.Background(), fixtureAmount(), "booking-1")
	if !errors.Is(err, domain.ErrPaymentProcessorUnavailable) {
		t.Fatalf("got err %v, want %v", err, domain.ErrPaymentProcessorUnavailable)
	}
}

// TestProcessor_RefundPayment_Succeeds proves refunding a captured intent
// succeeds by default.
func TestProcessor_RefundPayment_Succeeds(t *testing.T) {
	t.Parallel()

	p := stripestub.NewProcessor()
	ref, err := p.CreateIntent(context.Background(), fixtureAmount(), "booking-1")
	if err != nil {
		t.Fatalf("unexpected err creating intent: %v", err)
	}
	if err := p.CapturePayment(context.Background(), ref); err != nil {
		t.Fatalf("unexpected err capturing: %v", err)
	}
	if err := p.RefundPayment(context.Background(), ref); err != nil {
		t.Fatalf("unexpected err refunding: %v", err)
	}
}

// TestProcessor_RefundPayment_UncapturedRef proves refunding an intent
// that was never captured is rejected rather than silently accepted.
func TestProcessor_RefundPayment_UncapturedRef(t *testing.T) {
	t.Parallel()

	p := stripestub.NewProcessor()
	ref, err := p.CreateIntent(context.Background(), fixtureAmount(), "booking-1")
	if err != nil {
		t.Fatalf("unexpected err creating intent: %v", err)
	}
	err = p.RefundPayment(context.Background(), ref)
	if !errors.Is(err, domain.ErrPaymentProcessorUnavailable) {
		t.Fatalf("got err %v, want %v", err, domain.ErrPaymentProcessorUnavailable)
	}
}
