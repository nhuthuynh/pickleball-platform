package port

import (
	"context"

	"github.com/nhuthuynh/white-label/internal/payments/domain"
)

// PaymentProcessor is Payments' outbound anti-corruption layer for the
// online path: the processor-agnostic lifecycle a card payment actually
// has (create an intent/authorization, capture it, refund it), expressed
// entirely in this context's own primitives (domain.Money, plain strings)
// rather than any Stripe SDK type. This mirrors the reasoning
// docs/requirements/research-security-compliance.md §3 gives for the Auth0
// TokenVerifier port: shape the port around the vendor-agnostic concept
// (RFC 9068 claims there, the intent/capture/refund lifecycle here), not
// around the vendor's own types, so that swapping the stub adapter
// (internal/payments/adapter/stripestub, T6.2) for a real
// internal/payments/adapter/stripe package later is an adapter-only
// change — app and domain never change. Treat this interface as a promise
// from day one (PE dossier §2, Hyrum's Law): a future real adapter has to
// honor this exact signature and error contract even though nothing here
// is real Stripe yet.
//
// PCI guardrail: no method on this interface, and no type in
// internal/payments/port or internal/payments/domain, may ever carry a
// card number, CVV/CVC, or raw track-data field. There is nothing to
// tokenize server-side — Stripe.js/Elements/Checkout (or the mobile SDKs)
// submit card data directly to Stripe from the client, never through this
// backend (SAQ A scope, docs/requirements/research-security-compliance.md
// §2). A future real Stripe adapter must be reviewed against this
// constraint before it's merged.
//
// Implementations return exactly one of the two domain sentinel errors
// below on failure — never a processor-specific error type — so app/domain
// stay processor-agnostic (CLAUDE.md rule 5):
//   - domain.ErrPaymentProcessorUnavailable: the processor itself couldn't
//     be reached, or rejected the request as malformed/unknown (e.g. an
//     unrecognised intent reference) — an infra-shaped failure.
//   - domain.ErrPaymentDeclined: the processor was reachable and
//     affirmatively declined the payment (e.g. a declined card) — a
//     legitimate business outcome, not an error condition to alarm on.
type PaymentProcessor interface {
	// CreateIntent authorizes amount against payableID and returns a
	// processor reference (a Stripe PaymentIntent id, in the real
	// adapter) that later CapturePayment/RefundPayment calls use to refer
	// back to it. Creating an intent does not move money — the Payment
	// stays unpaid until a corresponding CapturePayment succeeds.
	CreateIntent(ctx context.Context, amount domain.Money, payableID string) (intentRef string, err error)

	// CapturePayment captures funds for a previously created intent. On
	// success, the caller (app.Service.ConfirmOnlinePayment) transitions
	// the Payment to paid. On domain.ErrPaymentDeclined, the caller must
	// leave the Payment unchanged (still unpaid) — a decline is not an
	// illegal state transition, it's a capture that simply didn't happen.
	CapturePayment(ctx context.Context, intentRef string) error

	// RefundPayment refunds a previously captured intent.
	RefundPayment(ctx context.Context, intentRef string) error
}
