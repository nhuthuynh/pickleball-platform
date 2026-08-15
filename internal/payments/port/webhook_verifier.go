package port

// WebhookVerifier is Payments' outbound anti-corruption layer for
// authenticating an inbound Stripe webhook delivery (T18.1, closes #167) —
// mirroring PaymentProcessor's own doc comment: the port is shaped around
// the vendor-agnostic concept (verify a signature over a byte payload)
// rather than any Stripe SDK type, so a future real
// internal/payments/adapter/stripe package (already named as a placeholder
// in stripestub's own doc comment) can implement both PaymentProcessor and
// WebhookVerifier for real, and swapping the stub adapter
// (internal/payments/adapter/webhookstub) out for it is wiring-only
// (cmd/server), never an app/domain change.
//
// A signature-verified webhook is Payments' one deliberate exception to
// "every RPC requires a verified principal" (see
// internal/payments/adapter/grpcapi.PublicMethods()): Stripe itself has no
// login on this platform to hold a token, so the HMAC signature over the raw
// payload IS the authentication for this one caller, not a stand-in for it.
type WebhookVerifier interface {
	// VerifySignature checks that signatureHeader is a valid signature over
	// raw, returning domain.ErrWebhookSignatureInvalid if it is not.
	// raw must be the exact bytes the signature covers — HMAC verification
	// is byte-exact, so any mutation (including a well-meaning
	// re-serialization) fails verification.
	VerifySignature(raw []byte, signatureHeader string) error
}
