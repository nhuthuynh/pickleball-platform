// Package stripestub implements port.PaymentProcessor as a deterministic
// in-memory fake. The package is deliberately named stripestub, not
// stripe, so it's obvious at a glance this is not the real Stripe
// integration — no Stripe SDK dependency is added by this package, it
// makes no network calls, and it reads no STRIPE_* environment variables.
// A future internal/payments/adapter/stripe package implements the same
// port.PaymentProcessor interface against the real Stripe API; swapping
// one for the other is wiring-only (cmd/server), never an app/domain
// change.
package stripestub

import (
	"context"
	"fmt"
	"sync"

	"github.com/nhuthuynh/white-label/internal/payments/domain"
)

// intentState tracks a single fake intent's lifecycle so CapturePayment/
// RefundPayment can validate the caller isn't skipping a step (capturing
// twice, refunding before capture, etc. are not this ticket's concern
// beyond the minimum needed to keep the fake honest — see the required
// "unknown/uncaptured ref" tests).
type intentState struct {
	payableID string
	captured  bool
}

// Processor is a deterministic, in-memory stand-in for a real Stripe
// client. Every CreateIntent/CapturePayment/RefundPayment call succeeds
// unless a test has seeded a specific failure for the payable in question
// via SeedDecline or SeedUnavailable — there is no other source of
// non-determinism (no clock, no randomness, no network).
type Processor struct {
	mu sync.Mutex

	counter int
	intents map[string]intentState

	declinePayables     map[string]bool
	unavailablePayables map[string]bool
}

// NewProcessor returns a ready-to-use Processor with no seeded failures.
func NewProcessor() *Processor {
	return &Processor{
		intents:             make(map[string]intentState),
		declinePayables:     make(map[string]bool),
		unavailablePayables: make(map[string]bool),
	}
}

// SeedDecline marks payableID so that the next CapturePayment against an
// intent created for it returns domain.ErrPaymentDeclined, simulating a
// declined card. Does not affect CreateIntent — an intent can still be
// created (authorized) for a payable whose eventual capture will decline,
// exactly like a real card that authorizes but then fails to settle.
func (p *Processor) SeedDecline(payableID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.declinePayables[payableID] = true
}

// SeedUnavailable marks payableID so that the next CreateIntent for it
// returns domain.ErrPaymentProcessorUnavailable, simulating the processor
// itself being unreachable.
func (p *Processor) SeedUnavailable(payableID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.unavailablePayables[payableID] = true
}

// CreateIntent implements port.PaymentProcessor.
func (p *Processor) CreateIntent(_ context.Context, _ domain.Money, payableID string) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.unavailablePayables[payableID] {
		return "", domain.ErrPaymentProcessorUnavailable
	}

	p.counter++
	ref := fmt.Sprintf("pi_stub_%d", p.counter)
	p.intents[ref] = intentState{payableID: payableID}
	return ref, nil
}

// CapturePayment implements port.PaymentProcessor.
func (p *Processor) CapturePayment(_ context.Context, intentRef string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	st, ok := p.intents[intentRef]
	if !ok {
		return domain.ErrPaymentProcessorUnavailable
	}
	if p.declinePayables[st.payableID] {
		return domain.ErrPaymentDeclined
	}
	st.captured = true
	p.intents[intentRef] = st
	return nil
}

// RefundPayment implements port.PaymentProcessor.
func (p *Processor) RefundPayment(_ context.Context, intentRef string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	st, ok := p.intents[intentRef]
	if !ok || !st.captured {
		return domain.ErrPaymentProcessorUnavailable
	}
	return nil
}
