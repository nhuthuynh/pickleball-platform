package app_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/nhuthuynh/white-label/internal/payments/adapter/stripestub"
	"github.com/nhuthuynh/white-label/internal/payments/adapter/webhookstub"
	"github.com/nhuthuynh/white-label/internal/payments/app"
	"github.com/nhuthuynh/white-label/internal/payments/domain"
)

// hmacHex computes the same HMAC-SHA256-hex signature webhookstub.Verifier
// checks, independently of the implementation under test (mirroring
// webhookstub's own verifier_test.go validSignature helper) — this is what
// makes signedEvent's fixtures genuinely valid rather than tautologically
// accepted by whatever webhookstub happens to compute.
func hmacHex(secret string, raw []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(raw)
	return hex.EncodeToString(mac.Sum(nil))
}

// T18.1 (closes #167): app.Service.HandleStripeWebhookEvent lets a
// successful Stripe charge complete its own Payment even if the client's
// own ConfirmOnlinePayment call never lands. These are the ticket's own
// required headline tests (instruction 10).

const webhookSecret = "whsec_test_secret"

// webhookFixtureEventID/webhookFixtureStripeRef are deterministic fixtures
// for the tests below — UUID-shaped is not required here (event_id and
// stripe_reference are opaque strings, never parsed as uuids by this
// package), so plain descriptive literals are used instead.
const (
	webhookFixtureEventID       = "evt_test_1"
	webhookFixtureStripeRef     = "pi_stub_1"
	webhookFixturePayableID     = "6ba7b810-0000-4000-8000-0000000000e1"
	webhookFixturePaymentID     = "pay-webhook-1"
	stripeEventPaymentSucceeded = "payment_intent.succeeded"
)

// newWebhookTestService wires the real app.Service the way cmd/server does
// for the webhook path: a real webhookstub.Verifier (not a fake) and a real
// stripestub.Processor (not a fake), so the headline "valid signature
// captures the Payment" test is driven through real production code end to
// end (T14.8/T15.5's cross-context-fake warning applied to this same-context
// boundary, per T18.1 instruction 10's own text). Returns the pieces a test
// needs to seed/assert against directly.
func newWebhookTestService(ids ...string) (*app.Service, *fakeRepository, *stripestub.Processor, *fakeWebhookEventStore) {
	repo := newFakeRepository()
	proc := stripestub.NewProcessor()
	events := newFakeWebhookEventStore()
	svc := app.NewService(app.ServiceOptions{
		Payments:        repo,
		IDs:             &fixedIDs{ids: ids},
		Processor:       proc,
		WebhookVerifier: webhookstub.NewVerifier(webhookSecret),
		WebhookEvents:   events,
	})
	return svc, repo, proc, events
}

// signedEvent builds a valid, webhookstub-verifiable
// app.HandleStripeWebhookEventInput for the given event/stripe reference.
// webhookstub.Verifier does not expose a Sign method (a real Stripe
// signature is computed by Stripe itself, never by this backend), so this
// helper recomputes the identical HMAC-SHA256-hex the stub checks via
// hmacHex, exactly as webhookstub's own verifier_test.go validSignature
// helper does — independent of app.Service's own call to VerifySignature,
// so these fixtures are genuinely valid rather than trivially accepted.
func signedEvent(eventID, stripeRef string) app.HandleStripeWebhookEventInput {
	raw := []byte(`{"id":"` + eventID + `","type":"` + stripeEventPaymentSucceeded + `"}`)
	return app.HandleStripeWebhookEventInput{
		RawPayload:      raw,
		SignatureHeader: hmacHex(webhookSecret, raw),
		EventID:         eventID,
		EventType:       stripeEventPaymentSucceeded,
		StripeReference: stripeRef,
	}
}

// seedOnlinePayment creates an unpaid online Payment directly in repo,
// carrying stripeRef, so a webhook test can target it without going through
// CreateOnlinePayment's own authorization/actor plumbing (irrelevant to the
// webhook path, which has no principal at all).
func seedOnlinePayment(t *testing.T, repo *fakeRepository, id, payableID, stripeRef string, status domain.Status) domain.Payment {
	t.Helper()
	p, err := repo.Create(context.Background(), domain.Payment{
		ID:               id,
		PayableType:      domain.PayableTypeBooking,
		PayableID:        payableID,
		Amount:           domain.Money{Cents: 3000, Currency: "USD"},
		Method:           domain.MethodOnline,
		Status:           domain.StatusUnpaid,
		StripeReference:  stripeRef,
		RecordedByUserID: fixtureOnlinePayerID,
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	if status == domain.StatusPaid {
		p.Status = domain.StatusPaid
		if _, err := repo.Update(context.Background(), p); err != nil {
			t.Fatalf("seed: mark paid: %v", err)
		}
	}
	return p
}

// TestHandleStripeWebhookEvent_ValidSignatureCapturesPayment is the
// ticket's headline test: a valid signature over a known event captures the
// Payment, asserted via the real s.payments/webhookstub/s.processor.
func TestHandleStripeWebhookEvent_ValidSignatureCapturesPayment(t *testing.T) {
	t.Parallel()

	svc, repo, proc, _ := newWebhookTestService()
	ref, err := proc.CreateIntent(context.Background(), domain.Money{Cents: 3000, Currency: "USD"}, webhookFixturePayableID)
	if err != nil {
		t.Fatalf("seed: CreateIntent: %v", err)
	}
	seedOnlinePayment(t, repo, webhookFixturePaymentID, webhookFixturePayableID, ref, domain.StatusUnpaid)

	err = svc.HandleStripeWebhookEvent(context.Background(), signedEvent(webhookFixtureEventID, ref))
	if err != nil {
		t.Fatalf("HandleStripeWebhookEvent() = %v, want nil", err)
	}

	stored, err := repo.GetByID(context.Background(), webhookFixturePaymentID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if stored.Status != domain.StatusPaid {
		t.Fatalf("Status = %v, want paid", stored.Status)
	}
}

// TestHandleStripeWebhookEvent_InvalidSignature_Refused proves an invalid
// signature is refused with domain.ErrWebhookSignatureInvalid (mapped by
// grpcapi to PermissionDenied) and captures nothing.
func TestHandleStripeWebhookEvent_InvalidSignature_Refused(t *testing.T) {
	t.Parallel()

	svc, repo, proc, _ := newWebhookTestService()
	ref, err := proc.CreateIntent(context.Background(), domain.Money{Cents: 3000, Currency: "USD"}, webhookFixturePayableID)
	if err != nil {
		t.Fatalf("seed: CreateIntent: %v", err)
	}
	seedOnlinePayment(t, repo, webhookFixturePaymentID, webhookFixturePayableID, ref, domain.StatusUnpaid)

	in := signedEvent(webhookFixtureEventID, ref)
	in.SignatureHeader = "0000000000000000000000000000000000000000000000000000000000000000"

	err = svc.HandleStripeWebhookEvent(context.Background(), in)
	if !errors.Is(err, domain.ErrWebhookSignatureInvalid) {
		t.Fatalf("err = %v, want domain.ErrWebhookSignatureInvalid", err)
	}

	stored, err := repo.GetByID(context.Background(), webhookFixturePaymentID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if stored.Status != domain.StatusUnpaid {
		t.Fatalf("Status = %v, want unchanged (unpaid) — an invalid signature must capture nothing", stored.Status)
	}
}

// TestHandleStripeWebhookEvent_RedeliveredEvent_CapturesNothingTwice proves
// redelivering the same event_id a second time captures nothing a second
// time — the redelivery-safe no-op Stripe's own retry behaviour requires.
//
// This asserts repo.getByStripeReferenceCalls stays at 1, not merely that
// the end state looks right — because the end state alone does NOT isolate
// this guard: after a successful first delivery the Payment is paid, so
// instruction 7(e)'s already-paid guard would ALSO produce "no second
// capture, nil returned" even if the idempotency-claim guard (7(b)) were
// missing entirely. A mutation check that removed 7(b) and only asserted
// the final Status/error would have passed anyway, silently proving nothing
// — the call-count assertion below is what actually pins 7(b) as its own,
// independently-required guard, short-circuiting BEFORE Payment resolution
// is ever reached a second time.
func TestHandleStripeWebhookEvent_RedeliveredEvent_CapturesNothingTwice(t *testing.T) {
	t.Parallel()

	svc, repo, proc, _ := newWebhookTestService()
	ref, err := proc.CreateIntent(context.Background(), domain.Money{Cents: 3000, Currency: "USD"}, webhookFixturePayableID)
	if err != nil {
		t.Fatalf("seed: CreateIntent: %v", err)
	}
	seedOnlinePayment(t, repo, webhookFixturePaymentID, webhookFixturePayableID, ref, domain.StatusUnpaid)

	first := signedEvent(webhookFixtureEventID, ref)
	if err := svc.HandleStripeWebhookEvent(context.Background(), first); err != nil {
		t.Fatalf("first delivery: %v", err)
	}
	if got := repo.getByStripeReferenceCalls.Load(); got != 1 {
		t.Fatalf("after first delivery, GetByStripeReference calls = %d, want 1", got)
	}

	// A second, identical delivery of the same event_id.
	second := signedEvent(webhookFixtureEventID, ref)
	if err := svc.HandleStripeWebhookEvent(context.Background(), second); err != nil {
		t.Fatalf("redelivered event: err = %v, want nil (safe no-op)", err)
	}

	// The call count staying at 1 — not 2 — is the proof the idempotency
	// claim short-circuited the redelivery BEFORE Payment resolution, which
	// the already-paid guard downstream cannot substitute for (see this
	// test's own doc comment).
	if got := repo.getByStripeReferenceCalls.Load(); got != 1 {
		t.Fatalf("after redelivered event, GetByStripeReference calls = %d, want still 1 (unreached) — "+
			"the idempotency claim must short-circuit before Payment resolution", got)
	}

	stored, err := repo.GetByID(context.Background(), webhookFixturePaymentID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if stored.Status != domain.StatusPaid {
		t.Fatalf("Status = %v, want paid", stored.Status)
	}
}

// TestHandleStripeWebhookEvent_AlreadyPaidPayment_CapturesNothingTwice
// simulates a race with a prior client-driven ConfirmOnlinePayment call: a
// DIFFERENT event_id (so the idempotency-claim guard alone would not catch
// it) for a Payment that is already paid must capture nothing a second time
// and return success (nil), not an error.
func TestHandleStripeWebhookEvent_AlreadyPaidPayment_CapturesNothingTwice(t *testing.T) {
	t.Parallel()

	svc, repo, proc, _ := newWebhookTestService()
	ref, err := proc.CreateIntent(context.Background(), domain.Money{Cents: 3000, Currency: "USD"}, webhookFixturePayableID)
	if err != nil {
		t.Fatalf("seed: CreateIntent: %v", err)
	}
	// Already paid — as if ConfirmOnlinePayment (or an earlier webhook
	// delivery under a different event_id) already completed the capture.
	seedOnlinePayment(t, repo, webhookFixturePaymentID, webhookFixturePayableID, ref, domain.StatusPaid)

	err = svc.HandleStripeWebhookEvent(context.Background(), signedEvent("evt_different_delivery", ref))
	if err != nil {
		t.Fatalf("HandleStripeWebhookEvent() on an already-paid Payment = %v, want nil (safe no-op)", err)
	}

	stored, err := repo.GetByID(context.Background(), webhookFixturePaymentID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if stored.Status != domain.StatusPaid {
		t.Fatalf("Status = %v, want unchanged (paid)", stored.Status)
	}
}

// TestHandleStripeWebhookEvent_UnrecognizedEventType_NoOp proves an event
// type other than payment_intent.succeeded is acked and does nothing — an
// explicit, tested, disclosed no-op, not a silent crash or Unimplemented.
func TestHandleStripeWebhookEvent_UnrecognizedEventType_NoOp(t *testing.T) {
	t.Parallel()

	svc, repo, proc, _ := newWebhookTestService()
	ref, err := proc.CreateIntent(context.Background(), domain.Money{Cents: 3000, Currency: "USD"}, webhookFixturePayableID)
	if err != nil {
		t.Fatalf("seed: CreateIntent: %v", err)
	}
	seedOnlinePayment(t, repo, webhookFixturePaymentID, webhookFixturePayableID, ref, domain.StatusUnpaid)

	in := signedEvent(webhookFixtureEventID, ref)
	in.EventType = "charge.refunded"
	raw := []byte(`{"id":"` + webhookFixtureEventID + `","type":"charge.refunded"}`)
	in.RawPayload = raw
	in.SignatureHeader = hmacHex(webhookSecret, raw)

	if err := svc.HandleStripeWebhookEvent(context.Background(), in); err != nil {
		t.Fatalf("unrecognized event type: err = %v, want nil (no-op)", err)
	}

	stored, err := repo.GetByID(context.Background(), webhookFixturePaymentID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if stored.Status != domain.StatusUnpaid {
		t.Fatalf("Status = %v, want unchanged (unpaid) — an unrecognized event type must not capture", stored.Status)
	}
}

// TestHandleStripeWebhookEvent_UnknownStripeReference_NotFound proves an
// unrecognized stripe_reference returns a clean not-found status, not
// Internal.
func TestHandleStripeWebhookEvent_UnknownStripeReference_NotFound(t *testing.T) {
	t.Parallel()

	svc, _, _, _ := newWebhookTestService()

	err := svc.HandleStripeWebhookEvent(context.Background(), signedEvent(webhookFixtureEventID, "pi_stub_never_issued"))
	if !errors.Is(err, domain.ErrPaymentNotFound) {
		t.Fatalf("err = %v, want domain.ErrPaymentNotFound", err)
	}
}
