package app

import (
	"context"

	"github.com/nhuthuynh/white-label/internal/payments/domain"
	"github.com/nhuthuynh/white-label/internal/payments/port"
)

// Service is the Payments context's application layer: it orchestrates the
// domain and the outbound ports, but holds no business rules itself —
// those live in internal/payments/domain. T6.2 gives Service just enough
// to drive the online path against port.PaymentProcessor; persistence
// (port.Repository) and the Social Play payment-status projection land in
// T6.3/T6.4/T6.5 and will grow this into an options-struct constructor per
// the sprint plan's kickoff note, rather than growing positional args past
// three.
type Service struct {
	ids       port.IDGenerator
	processor port.PaymentProcessor
}

// NewService constructs a Service. ids and processor are both required
// ports — tests inject a fixed-sequence IDGenerator and the stripestub
// Processor (or a hand-written fake) so behaviour stays deterministic.
func NewService(ids port.IDGenerator, processor port.PaymentProcessor) *Service {
	return &Service{ids: ids, processor: processor}
}

// CreateOnlinePaymentInput is the use-case input for starting an online
// Payment.
type CreateOnlinePaymentInput struct {
	PayableType domain.PayableType
	PayableID   string
	Amount      domain.Money
}

// CreateOnlinePayment builds an unpaid, online Payment (domain.NewPayment)
// and authorizes it with the payment processor (port.PaymentProcessor.
// CreateIntent). On success, the returned Payment carries the processor's
// intent reference as StripeReference but is still unpaid — creating an
// intent authorizes funds, it does not capture them. Persisting the
// returned Payment is the caller's responsibility (T6.4's repository
// port); this method's job ends at "here is the Payment to persist,
// pending confirmation."
func (s *Service) CreateOnlinePayment(ctx context.Context, in CreateOnlinePaymentInput) (domain.Payment, error) {
	p, err := domain.NewPayment(s.ids.NewID(), in.PayableType, in.PayableID, in.Amount, domain.MethodOnline, "")
	if err != nil {
		return domain.Payment{}, err
	}

	intentRef, err := s.processor.CreateIntent(ctx, in.Amount, in.PayableID)
	if err != nil {
		return domain.Payment{}, err
	}
	p.StripeReference = intentRef

	return p, nil
}

// ConfirmOnlinePayment captures funds for p (an online Payment previously
// returned by CreateOnlinePayment, or reloaded from persistence in T6.4)
// via port.PaymentProcessor.CapturePayment, then transitions p to paid
// (domain.Payment.MarkPaid) on success.
//
// A declined card (domain.ErrPaymentDeclined) or any other processor
// failure (domain.ErrPaymentProcessorUnavailable) is not an illegal state
// transition — it's a capture that simply didn't happen — so p is
// returned unchanged (still whatever status it was passed in as,
// typically unpaid) alongside the error, rather than mutated into some
// half-applied state. Callers must not persist p on a non-nil error other
// than to leave the existing unpaid row as-is.
func (s *Service) ConfirmOnlinePayment(ctx context.Context, p domain.Payment) (domain.Payment, error) {
	// Any processor-side failure (declined or otherwise unavailable) means
	// no capture happened, so p is returned exactly as it was passed in —
	// there is nothing to roll back because MarkPaid was never called.
	if err := s.processor.CapturePayment(ctx, p.StripeReference); err != nil {
		return p, err
	}

	if err := p.MarkPaid(domain.MethodOnline, p.StripeReference); err != nil {
		return p, err
	}

	return p, nil
}
