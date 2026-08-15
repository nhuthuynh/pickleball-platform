package grpcapi_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/nhuthuynh/white-label/internal/payments/adapter/grpcapi"
	"github.com/nhuthuynh/white-label/internal/payments/adapter/stripestub"
	"github.com/nhuthuynh/white-label/internal/payments/adapter/webhookstub"
	"github.com/nhuthuynh/white-label/internal/payments/app"
	"github.com/nhuthuynh/white-label/internal/payments/domain"

	paymentsv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/payments/v1"
)

// T18.1 (closes #167): Handler.ReceiveStripeWebhookEvent, driven end to end
// (real app.Service, real webhookstub, real stripestub) with only
// persistence and the idempotency ledger faked — mirroring
// newMappingHandler's own established pattern (error_mapping_test.go) for
// the identical reason: no Docker daemon in this environment (CLAUDE.md
// rule 10).

const webhookHandlerSecret = "whsec_handler_test"

// fakeWebhookEventStore is an in-memory port.WebhookEventStore test double
// local to this package, mirroring internal/payments/app/fixtures_test.go's
// identical fake.
type fakeWebhookEventStore struct {
	claimed map[string]bool
}

func newFakeWebhookEventStore() *fakeWebhookEventStore {
	return &fakeWebhookEventStore{claimed: map[string]bool{}}
}

func (f *fakeWebhookEventStore) ClaimEvent(_ context.Context, eventID string) (bool, error) {
	if f.claimed[eventID] {
		return false, nil
	}
	f.claimed[eventID] = true
	return true, nil
}

// newWebhookHandler wires the real app.Service + real grpcapi.Handler with
// a real webhookstub.Verifier and real stripestub.Processor, exactly as
// cmd/server wires them, with only persistence (fakeRepository) and the
// idempotency ledger (fakeWebhookEventStore) faked.
func newWebhookHandler(ids ...string) (*grpcapi.Handler, *fakeRepository, *stripestub.Processor) {
	repo := newFakeRepository()
	proc := stripestub.NewProcessor()
	svc := app.NewService(app.ServiceOptions{
		Payments:        repo,
		IDs:             &fixedIDs{ids: ids},
		Processor:       proc,
		WebhookVerifier: webhookstub.NewVerifier(webhookHandlerSecret),
		WebhookEvents:   newFakeWebhookEventStore(),
	})
	return grpcapi.NewHandler(svc), repo, proc
}

func webhookHMAC(raw []byte) string {
	mac := hmac.New(sha256.New, []byte(webhookHandlerSecret))
	mac.Write(raw)
	return hex.EncodeToString(mac.Sum(nil))
}

// TestReceiveStripeWebhookEvent_ValidSignature_Succeeds proves the RPC-level
// wiring: a validly-signed request for a known Payment captures it, with no
// principal in ctx at all (context.Background(), not ctxAs(...)) — this RPC
// is public (T18.1's own PublicMethods() entry), never requiring one.
func TestReceiveStripeWebhookEvent_ValidSignature_Succeeds(t *testing.T) {
	t.Parallel()

	h, repo, proc := newWebhookHandler()
	ref, err := proc.CreateIntent(context.Background(), domain.Money{Cents: 3000, Currency: "USD"}, fixtureBookingID)
	if err != nil {
		t.Fatalf("seed: CreateIntent: %v", err)
	}
	seedOnline(t, repo, mapPaymentID, domain.PayableTypeBooking, fixtureBookingID, ref, domain.StatusUnpaid, seededPaymentOwnerID)

	raw := []byte(`{"id":"evt_handler_1","type":"payment_intent.succeeded"}`)
	resp, err := h.ReceiveStripeWebhookEvent(context.Background(), &paymentsv1.ReceiveStripeWebhookEventRequest{
		RawPayload:      raw,
		SignatureHeader: webhookHMAC(raw),
		EventId:         "evt_handler_1",
		EventType:       "payment_intent.succeeded",
		StripeReference: ref,
	})
	if err != nil {
		t.Fatalf("ReceiveStripeWebhookEvent() = %v, want nil", err)
	}
	if resp == nil {
		t.Fatal("ReceiveStripeWebhookEvent() returned a nil response with a nil error")
	}

	stored, err := repo.GetByID(context.Background(), mapPaymentID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if stored.Status != domain.StatusPaid {
		t.Fatalf("Status = %v, want paid", stored.Status)
	}
}

// TestReceiveStripeWebhookEvent_InvalidSignature_PermissionDenied proves an
// invalid signature is refused with codes.PermissionDenied, not Internal —
// T18.1 instruction 7(a)'s explicit requirement.
func TestReceiveStripeWebhookEvent_InvalidSignature_PermissionDenied(t *testing.T) {
	t.Parallel()

	h, repo, proc := newWebhookHandler()
	ref, err := proc.CreateIntent(context.Background(), domain.Money{Cents: 3000, Currency: "USD"}, fixtureBookingID)
	if err != nil {
		t.Fatalf("seed: CreateIntent: %v", err)
	}
	seedOnline(t, repo, mapPaymentID, domain.PayableTypeBooking, fixtureBookingID, ref, domain.StatusUnpaid, seededPaymentOwnerID)

	raw := []byte(`{"id":"evt_handler_2","type":"payment_intent.succeeded"}`)
	_, err = h.ReceiveStripeWebhookEvent(context.Background(), &paymentsv1.ReceiveStripeWebhookEventRequest{
		RawPayload:      raw,
		SignatureHeader: "not-a-valid-signature",
		EventId:         "evt_handler_2",
		EventType:       "payment_intent.succeeded",
		StripeReference: ref,
	})
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.PermissionDenied {
		t.Fatalf("code = %v, want PermissionDenied (err: %v)", st.Code(), err)
	}

	stored, reloadErr := repo.GetByID(context.Background(), mapPaymentID)
	if reloadErr != nil {
		t.Fatalf("reload: %v", reloadErr)
	}
	if stored.Status != domain.StatusUnpaid {
		t.Fatalf("Status = %v, want unchanged (unpaid)", stored.Status)
	}
}
