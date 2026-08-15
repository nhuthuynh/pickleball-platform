package webhookstub_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/nhuthuynh/white-label/internal/payments/adapter/webhookstub"
	"github.com/nhuthuynh/white-label/internal/payments/domain"
	"github.com/nhuthuynh/white-label/internal/payments/port"
)

// TestVerifier_SatisfiesPort proves the stub actually implements
// port.WebhookVerifier — a compile-time check as much as a runtime one,
// mirroring stripestub's TestProcessor_SatisfiesPort exactly.
func TestVerifier_SatisfiesPort(t *testing.T) {
	t.Parallel()
	var _ port.WebhookVerifier = webhookstub.NewVerifier("secret")
}

// validSignature computes the same HMAC-SHA256-over-raw-bytes-then-hex
// signature the stub is documented to check, independently of the
// implementation under test — this is what makes the "known-good" half of
// the required test actually prove something rather than tautologically
// pass.
func validSignature(secret string, raw []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(raw)
	return hex.EncodeToString(mac.Sum(nil))
}

// TestVerifier_ValidSignature_Succeeds proves a signature computed the same
// way the stub computes its own is accepted.
func TestVerifier_ValidSignature_Succeeds(t *testing.T) {
	t.Parallel()
	v := webhookstub.NewVerifier("test-secret")
	raw := []byte(`{"id":"evt_1","type":"payment_intent.succeeded"}`)

	if err := v.VerifySignature(raw, validSignature("test-secret", raw)); err != nil {
		t.Fatalf("VerifySignature() with a genuinely valid signature returned %v, want nil", err)
	}
}

// TestVerifier_InvalidSignature_Fails proves a signature that does not match
// is rejected with domain.ErrWebhookSignatureInvalid — the exact sentinel
// app.Service.HandleStripeWebhookEvent maps to PermissionDenied.
func TestVerifier_InvalidSignature_Fails(t *testing.T) {
	t.Parallel()
	v := webhookstub.NewVerifier("test-secret")
	raw := []byte(`{"id":"evt_1","type":"payment_intent.succeeded"}`)

	err := v.VerifySignature(raw, "deadbeef")
	if !errors.Is(err, domain.ErrWebhookSignatureInvalid) {
		t.Fatalf("VerifySignature() with a bogus signature = %v, want domain.ErrWebhookSignatureInvalid", err)
	}
}

// TestVerifier_WrongSecret_Fails proves a signature that is validly-shaped
// but computed with the wrong secret is rejected — distinct from the
// malformed-header case above, proving the check is a real HMAC comparison
// and not merely a shape/length check.
func TestVerifier_WrongSecret_Fails(t *testing.T) {
	t.Parallel()
	v := webhookstub.NewVerifier("test-secret")
	raw := []byte(`{"id":"evt_1","type":"payment_intent.succeeded"}`)

	err := v.VerifySignature(raw, validSignature("wrong-secret", raw))
	if !errors.Is(err, domain.ErrWebhookSignatureInvalid) {
		t.Fatalf("VerifySignature() with the wrong secret = %v, want domain.ErrWebhookSignatureInvalid", err)
	}
}

// TestVerifier_MutatedPayload_Fails proves the signature is byte-exact over
// raw: a signature valid for one payload is rejected against a mutated one,
// even a single-byte change — the property the proto's own doc comment on
// raw_payload relies on.
func TestVerifier_MutatedPayload_Fails(t *testing.T) {
	t.Parallel()
	v := webhookstub.NewVerifier("test-secret")
	raw := []byte(`{"id":"evt_1","type":"payment_intent.succeeded"}`)
	sig := validSignature("test-secret", raw)

	mutated := []byte(`{"id":"evt_2","type":"payment_intent.succeeded"}`)
	err := v.VerifySignature(mutated, sig)
	if !errors.Is(err, domain.ErrWebhookSignatureInvalid) {
		t.Fatalf("VerifySignature() over a mutated payload = %v, want domain.ErrWebhookSignatureInvalid", err)
	}
}
