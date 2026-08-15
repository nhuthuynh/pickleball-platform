// Package webhookstub implements port.WebhookVerifier as a deterministic,
// network-free stand-in for real Stripe webhook signature verification. The
// package is deliberately named webhookstub, not stripe, mirroring
// internal/payments/adapter/stripestub's own naming for the identical
// reason: it is obvious at a glance that this is not the real Stripe
// integration — no Stripe SDK dependency is added by this package, it makes
// no network calls, and it reads no STRIPE_* environment variables. A future
// internal/payments/adapter/stripe package implements the same
// port.WebhookVerifier interface against Stripe's real signature scheme
// (https://stripe.com/docs/webhooks/signatures — a timestamped, versioned
// HMAC over "timestamp.payload", not the plain HMAC-over-raw-bytes this stub
// checks); swapping one for the other is wiring-only (cmd/server), never an
// app/domain change.
package webhookstub

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"

	"github.com/nhuthuynh/white-label/internal/payments/domain"
)

// Verifier is a deterministic port.WebhookVerifier: it checks
// signatureHeader is the lowercase-hex HMAC-SHA256 of raw, keyed by a
// caller-supplied shared secret. Real Stripe signatures are considerably
// more involved (timestamped, versioned, multi-scheme) — this stub checks
// only the one property the app layer's ordering (T18.1 instruction 7)
// actually depends on: that the signature is a function of both the secret
// and the exact bytes, so a forged or mutated payload is rejected.
type Verifier struct {
	secret string
}

// NewVerifier returns a Verifier keyed by secret, so tests can seed
// known-good and known-bad signatures deterministically.
func NewVerifier(secret string) *Verifier {
	return &Verifier{secret: secret}
}

// VerifySignature implements port.WebhookVerifier.
func (v *Verifier) VerifySignature(raw []byte, signatureHeader string) error {
	mac := hmac.New(sha256.New, []byte(v.secret))
	mac.Write(raw)
	want := hex.EncodeToString(mac.Sum(nil))

	got, err := hex.DecodeString(signatureHeader)
	if err != nil {
		return domain.ErrWebhookSignatureInvalid
	}
	wantBytes, err := hex.DecodeString(want)
	if err != nil {
		// want is our own hex.EncodeToString output — this branch is
		// unreachable in practice, but returning the same sentinel keeps
		// this function's contract total rather than panicking.
		return domain.ErrWebhookSignatureInvalid
	}
	if !hmac.Equal(got, wantBytes) {
		return domain.ErrWebhookSignatureInvalid
	}
	return nil
}
