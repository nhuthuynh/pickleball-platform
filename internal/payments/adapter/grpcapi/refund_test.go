// T12.3 — handler-level tests for the RefundPayment RPC, proving the gRPC
// status-code mapping the ticket specifies survives the real
// grpcapi.Handler -> app.Service -> domain path (not just the app-layer
// unit tests in internal/payments/app/refund_test.go).
//
// Handler-level with an in-memory port.Repository fake rather than a
// `-tags=integration` testcontainers-go test, for the reason this package's
// authz_regression_test.go already documents at length: nothing under test
// here is influenced by port.Repository's implementation — a real Postgres
// round trip would add infrastructure, not proof — and this environment has
// no Docker daemon, so CLAUDE.md rule 10 means the test has to actually be
// runnable where it was written.
package grpcapi_test

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/nhuthuynh/white-label/internal/payments/adapter/grpcapi"
	"github.com/nhuthuynh/white-label/internal/payments/adapter/stripestub"
	"github.com/nhuthuynh/white-label/internal/payments/app"
	"github.com/nhuthuynh/white-label/internal/payments/domain"

	paymentsv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/payments/v1"
)

// UUID-shaped Payment id fixtures — RefundPayment resolves the Payment
// through app.Service.GetPayment's uuidShape guard, so a "pay-1"-shaped id
// would be rejected as malformed before the repository is consulted.
const (
	refundBookingPaymentID     = "6ba7b810-0000-4000-8000-0000000000b1"
	refundRegistrationPayentID = "6ba7b810-0000-4000-8000-0000000000b2"
	refundNoShowPaymentID      = "6ba7b810-0000-4000-8000-0000000000b3"
	refundUnknownPaymentID     = "6ba7b810-0000-4000-8000-0000000000bf"

	refundBookingHostID = "host-booking-9"
	refundGameHostID    = "host-game-9"
	refundOtherPlayerID = "player-outsider-9"
)

// newRefundTestHandler wires the real app.Service and real grpcapi.Handler
// against an in-memory repository and the deterministic stub processor —
// the same production code cmd/server wires, with only the persistence and
// processor boundaries faked.
func newRefundTestHandler() (*grpcapi.Handler, *fakeRepository, *stripestub.Processor) {
	repo := newFakeRepository()
	proc := stripestub.NewProcessor()
	svc := app.NewService(app.ServiceOptions{
		Payments:  repo,
		IDs:       &fixedIDs{},
		Processor: proc,
	})
	return grpcapi.NewHandler(svc), repo, proc
}

// seedPaidOnline puts a paid, online Payment in the repository carrying an
// intent reference the stub processor really issued and captured, so a
// refund against it succeeds at the processor for the right reason.
func seedPaidOnline(t *testing.T, repo *fakeRepository, proc *stripestub.Processor, paymentID string, payableType domain.PayableType, payableID string) {
	t.Helper()

	ctx := context.Background()
	amount := domain.Money{Cents: 3000, Currency: "USD"}
	ref, err := proc.CreateIntent(ctx, amount, payableID)
	if err != nil {
		t.Fatalf("seed: CreateIntent: %v", err)
	}
	if err := proc.CapturePayment(ctx, ref); err != nil {
		t.Fatalf("seed: CapturePayment: %v", err)
	}

	if _, err := repo.Create(ctx, domain.Payment{
		ID:              paymentID,
		PayableType:     payableType,
		PayableID:       payableID,
		Amount:          amount,
		Method:          domain.MethodOnline,
		Status:          domain.StatusPaid,
		StripeReference: ref,
	}); err != nil {
		t.Fatalf("seed: Create: %v", err)
	}
}

func wantCode(t *testing.T, err error, want codes.Code) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected an error with code %v, got nil", want)
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("error %v is not a gRPC status", err)
	}
	if st.Code() != want {
		t.Fatalf("status code = %v, want %v (err: %v)", st.Code(), want, err)
	}
}

// TestRefundPayment_Handler_BookingHostSucceeds is the happy path through
// the real handler: the response carries the Payment with the refunded
// status, mapped onto the proto enum.
func TestRefundPayment_Handler_BookingHostSucceeds(t *testing.T) {
	t.Parallel()

	h, repo, proc := newRefundTestHandler()
	seedPaidOnline(t, repo, proc, refundBookingPaymentID, domain.PayableTypeBooking, fixtureBookingID)

	resp, err := h.RefundPayment(ctxAs(refundBookingHostID), &paymentsv1.RefundPaymentRequest{
		PaymentId:     refundBookingPaymentID,
		BookingHostId: refundBookingHostID,
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got := resp.GetPayment().GetStatus(); got != paymentsv1.PaymentStatus_PAYMENT_STATUS_REFUNDED {
		t.Fatalf("status = %v, want PAYMENT_STATUS_REFUNDED", got)
	}
}

// TestRefundPayment_Handler_NonRecorderActorPermissionDenied is the
// object-level authorization regression test at the handler boundary: a
// player who is neither Host nor assigned Game Admin gets PermissionDenied
// — not Internal, and certainly not a silent success.
func TestRefundPayment_Handler_NonRecorderActorPermissionDenied(t *testing.T) {
	t.Parallel()

	h, repo, proc := newRefundTestHandler()
	seedPaidOnline(t, repo, proc, refundRegistrationPayentID, domain.PayableTypeRegistration, fixtureRegistrationID)

	_, err := h.RefundPayment(ctxAs(refundOtherPlayerID), &paymentsv1.RefundPaymentRequest{
		PaymentId:  refundRegistrationPayentID,
		GameHostId: refundGameHostID,
	})
	wantCode(t, err, codes.PermissionDenied)

	stored, getErr := repo.GetByID(context.Background(), refundRegistrationPayentID)
	if getErr != nil {
		t.Fatalf("unexpected err: %v", getErr)
	}
	if stored.Status != domain.StatusPaid {
		t.Fatalf("persisted status = %v, want paid — a denied refund must have no side effect", stored.Status)
	}
}

// TestRefundPayment_Handler_UnknownAndMalformedIDsNotFound proves both an
// unknown and a malformed payment_id answer NotFound — the malformed one
// without ever reaching the repository (and so without reaching the
// Postgres adapter's mustUUID, which panics on non-UUID input).
func TestRefundPayment_Handler_UnknownAndMalformedIDsNotFound(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		paymentID string
	}{
		{name: "well-formed but unknown", paymentID: refundUnknownPaymentID},
		{name: "malformed", paymentID: "not-a-uuid"},
		{name: "empty", paymentID: ""},
		{name: "sql-injection-shaped", paymentID: "' OR 1=1 --"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h, _, _ := newRefundTestHandler()

			_, err := h.RefundPayment(ctxAs(refundBookingHostID), &paymentsv1.RefundPaymentRequest{
				PaymentId:     tc.paymentID,
				BookingHostId: refundBookingHostID,
			})
			wantCode(t, err, codes.NotFound)
		})
	}
}

// TestRefundPayment_Handler_IllegalTransitionFailedPrecondition covers both
// illegal transitions the ticket names — already refunded, and never paid —
// and pins them to FailedPrecondition rather than InvalidArgument.
//
// FailedPrecondition is the semantically right code here: the request is
// perfectly well-formed, it's the *system state* that makes the operation
// illegal, and the caller shouldn't retry until that state changes.
//
// T13.9 (closing #131) resolved the divergence this comment used to
// describe: ErrIllegalStatusTransition now maps to FailedPrecondition for
// *every* RPC in this service, not just RefundPayment, so this test no
// longer pins a per-RPC exception. The service-wide version of this
// assertion lives in error_mapping_test.go.
func TestRefundPayment_Handler_IllegalTransitionFailedPrecondition(t *testing.T) {
	t.Parallel()

	t.Run("already refunded", func(t *testing.T) {
		t.Parallel()

		h, repo, proc := newRefundTestHandler()
		seedPaidOnline(t, repo, proc, refundBookingPaymentID, domain.PayableTypeBooking, fixtureBookingID)

		req := &paymentsv1.RefundPaymentRequest{
			PaymentId:     refundBookingPaymentID,
			BookingHostId: refundBookingHostID,
		}
		if _, err := h.RefundPayment(ctxAs(refundBookingHostID), req); err != nil {
			t.Fatalf("first refund should succeed: %v", err)
		}

		_, err := h.RefundPayment(ctxAs(refundBookingHostID), req)
		wantCode(t, err, codes.FailedPrecondition)
	})

	t.Run("never paid", func(t *testing.T) {
		t.Parallel()

		h, repo, _ := newRefundTestHandler()
		if _, err := repo.Create(context.Background(), domain.Payment{
			ID:          refundBookingPaymentID,
			PayableType: domain.PayableTypeBooking,
			PayableID:   fixtureBookingID,
			Amount:      domain.Money{Cents: 3000, Currency: "USD"},
			Method:      domain.MethodOffline,
			Status:      domain.StatusUnpaid,
		}); err != nil {
			t.Fatalf("seed: %v", err)
		}

		_, err := h.RefundPayment(ctxAs(refundBookingHostID), &paymentsv1.RefundPaymentRequest{
			PaymentId:     refundBookingPaymentID,
			BookingHostId: refundBookingHostID,
		})
		wantCode(t, err, codes.FailedPrecondition)
	})
}

// TestRefundPayment_Handler_ProcessorFailureUnavailableAndNotRefunded is
// T12.3's required processor-failure assertion at the handler boundary, with
// T13.9's corrected code: the caller sees Unavailable, and — the part that
// actually matters, and the part T13.9 did not touch — the Payment is still
// paid, not silently advanced to refunded. A platform that marks a refund it
// never made is one that believes it returned money it still holds.
//
// T12.3 pinned Internal here. T13.9 (closing #131) changed it to Unavailable
// service-wide: ErrPaymentProcessorUnavailable means the processor could not
// complete the request — in both the capture and refund directions nothing
// moved, so a retry is safe, which is Unavailable's contract. Internal is for
// broken invariants in our own system. The persistence assertion below is
// what actually guarantees a failed refund is not mistaken for a completed
// one; the status code is how the caller is told, not what protects the money.
func TestRefundPayment_Handler_ProcessorFailureUnavailableAndNotRefunded(t *testing.T) {
	t.Parallel()

	h, repo, _ := newRefundTestHandler()

	// A paid online Payment whose intent reference the stub processor never
	// issued — exactly what a real adapter reports for an unknown or stale
	// intent.
	if _, err := repo.Create(context.Background(), domain.Payment{
		ID:              refundBookingPaymentID,
		PayableType:     domain.PayableTypeBooking,
		PayableID:       fixtureBookingID,
		Amount:          domain.Money{Cents: 3000, Currency: "USD"},
		Method:          domain.MethodOnline,
		Status:          domain.StatusPaid,
		StripeReference: "pi_stub_never_issued",
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	_, err := h.RefundPayment(ctxAs(refundBookingHostID), &paymentsv1.RefundPaymentRequest{
		PaymentId:     refundBookingPaymentID,
		BookingHostId: refundBookingHostID,
	})
	wantCode(t, err, codes.Unavailable)

	stored, getErr := repo.GetByID(context.Background(), refundBookingPaymentID)
	if getErr != nil {
		t.Fatalf("unexpected err: %v", getErr)
	}
	if stored.Status != domain.StatusPaid {
		t.Fatalf("persisted status = %v, want paid — a failed processor refund must not mark the Payment refunded", stored.Status)
	}
}

// TestRefundPayment_Handler_OutOfScopePayableTypeInvalidArgument pins
// T12.3's scope boundary at the wire: competition_entry (issue #125 —
// Competitions stays cash-only for refund purposes) and no_show_fee (issue
// #130) are rejected rather than silently refunded.
func TestRefundPayment_Handler_OutOfScopePayableTypeInvalidArgument(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		payableType domain.PayableType
		payableID   string
	}{
		{name: "competition_entry (#125)", payableType: domain.PayableTypeCompetitionEntry, payableID: fixtureCompetitionEntryID},
		{name: "no_show_fee (#130)", payableType: domain.PayableTypeNoShowFee, payableID: fixtureRegistrationID},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h, repo, proc := newRefundTestHandler()
			seedPaidOnline(t, repo, proc, refundNoShowPaymentID, tc.payableType, tc.payableID)

			_, err := h.RefundPayment(ctxAs(refundGameHostID), &paymentsv1.RefundPaymentRequest{
				PaymentId:  refundNoShowPaymentID,
				GameHostId: refundGameHostID,
			})
			wantCode(t, err, codes.InvalidArgument)

			stored, getErr := repo.GetByID(context.Background(), refundNoShowPaymentID)
			if getErr != nil {
				t.Fatalf("unexpected err: %v", getErr)
			}
			if stored.Status != domain.StatusPaid {
				t.Fatalf("persisted status = %v, want paid", stored.Status)
			}
		})
	}
}
