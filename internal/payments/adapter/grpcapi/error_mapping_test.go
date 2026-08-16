// T13.9 (closes issue #131) — the PaymentsService's sentinel -> gRPC code
// mapping, asserted service-wide rather than per-RPC.
//
// The bug this file exists to prevent: T12.3 added a second, RefundPayment-only
// mapper (`toRefundStatus`) alongside the shared `toStatus`, so the *same*
// domain sentinel produced a *different* status code depending on which RPC
// surfaced it. Error handling stopped being a contract and became a
// per-endpoint guess.
//
// Three layers of guard, because the table alone would not have caught #131:
//
//  1. TestPaymentsService_SentinelToCode — one row per sentinel in
//     internal/payments/domain/errors.go, driven through a real RPC. A mapping
//     cannot be changed without a test changing with it.
//  2. TestPaymentsService_SameSentinelSameCodeAcrossRPCs — the actual #131
//     claim: the two sentinels that diverged are driven through two *different*
//     RPCs and the codes are asserted equal to each other. A future re-divergence
//     fails here even if someone updates the table above to match it.
//  3. TestPaymentsService_HasExactlyOneErrorMapper /
//     TestSentinelToCodeTableCoversEveryDomainSentinel — source-level guards:
//     a second `func(error) error` mapper in handler.go, or a new sentinel added
//     to domain/errors.go with no mapping row, fails the build's tests rather
//     than silently shipping. #131 was created by exactly the former.
//
// Handler-level with the in-memory port.Repository fake and the deterministic
// stub processor, for the reason refund_test.go and authz_regression_test.go
// already document: the mapping under test is unaffected by which
// port.Repository implementation is behind it, and this environment has no
// Docker daemon, so the test has to be runnable where it was written
// (CLAUDE.md rule 10).
package grpcapi_test

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
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

// UUID-shaped fixtures local to this file — every caller-supplied id crossing
// this boundary meets app.Service's uuidShape guard.
const (
	mapPaymentID      = "6ba7b810-0000-4000-8000-0000000000c1"
	mapOtherPaymentID = "6ba7b810-0000-4000-8000-0000000000c2"

	// An intent reference the stub processor never issued — what a real
	// adapter reports for an unknown or stale intent, and the only way either
	// direction (capture or refund) reaches ErrPaymentProcessorUnavailable.
	mapUnknownIntentRef = "pi_stub_never_issued"
)

// mappingWebhookSecret is the shared secret newMappingHandler wires its real
// webhookstub.Verifier with, so the "invalid webhook signature" row below
// exercises a genuine HMAC mismatch rather than the nil-verifier
// fail-closed path — those are two different code paths that happen to
// share a sentinel (see app.Service.HandleStripeWebhookEvent's own doc
// comment), and this row's whole point is testing the former.
const mappingWebhookSecret = "whsec_mapping_test"

// newMappingHandler wires the real app.Service and real grpcapi.Handler, as
// cmd/server does, with only persistence, the processor, and the webhook
// idempotency ledger faked. ids seeds the deterministic id generator for
// the RPCs that create a Payment.
func newMappingHandler(ids ...string) (*grpcapi.Handler, *fakeRepository, *stripestub.Processor) {
	repo := newFakeRepository()
	proc := stripestub.NewProcessor()
	svc := app.NewService(app.ServiceOptions{
		Payments:        repo,
		IDs:             &fixedIDs{ids: ids},
		Processor:       proc,
		WebhookVerifier: webhookstub.NewVerifier(mappingWebhookSecret),
		WebhookEvents:   newFakeWebhookEventStore(),
		// T28.1: every RPC below is authenticated, and actor(ctx) now
		// resolves through this port before any of the sentinels in the
		// table are ever reached — a nil Identity would fail every case
		// closed with ErrUserNotFound before its real sentinel had a chance
		// to fire.
		Identity: newFakeIdentityLookup(),
	})
	return grpcapi.NewHandler(svc), repo, proc
}

// newMappingHandlerWithUnregisteredActor is newMappingHandler's variant for
// the one row (ErrUserNotFound) that needs a subject Identity has never
// heard of — every other row in this table authenticates as a subject
// newFakeIdentityLookup resolves successfully, by design.
func newMappingHandlerWithUnregisteredActor(unregisteredSubject string, ids ...string) (*grpcapi.Handler, *fakeRepository, *stripestub.Processor) {
	repo := newFakeRepository()
	proc := stripestub.NewProcessor()
	svc := app.NewService(app.ServiceOptions{
		Payments:        repo,
		IDs:             &fixedIDs{ids: ids},
		Processor:       proc,
		WebhookVerifier: webhookstub.NewVerifier(mappingWebhookSecret),
		WebhookEvents:   newFakeWebhookEventStore(),
		Identity:        newFakeIdentityLookup(unregisteredSubject),
	})
	return grpcapi.NewHandler(svc), repo, proc
}

// seedOnline puts an online Payment in the repository in the given status,
// carrying the given intent reference verbatim and recorded as belonging to
// recordedBy.
//
// recordedBy is an explicit parameter rather than a constant baked in (T13.7):
// it is the fact ConfirmOnlinePayment authorizes against, so which value a
// seeded Payment carries decides whether a capture is allowed at all, and an
// empty one models a row written before that ticket. Passing it at the call
// site keeps that visible where it matters instead of hiding it in a helper.
func seedOnline(t *testing.T, repo *fakeRepository, paymentID string, payableType domain.PayableType, payableID, intentRef string, st domain.Status, recordedBy string) {
	t.Helper()

	if _, err := repo.Create(context.Background(), domain.Payment{
		ID:               paymentID,
		PayableType:      payableType,
		PayableID:        payableID,
		Amount:           domain.Money{Cents: 3000, Currency: "USD"},
		Method:           domain.MethodOnline,
		Status:           st,
		StripeReference:  intentRef,
		RecordedByUserID: recordedBy,
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
}

// errorMappingCase is one sentinel, the RPC call that provokes it through real
// production code, and the status code the whole service must answer with.
type errorMappingCase struct {
	name string
	// sentinel is the domain.Err* identifier this row pins, matched against
	// internal/payments/domain/errors.go by
	// TestSentinelToCodeTableCoversEveryDomainSentinel.
	sentinel string
	invoke   func(t *testing.T) error
	wantCode codes.Code
	// why records the reasoning for a code that is not self-evident, so a
	// future change has to argue with a stated position rather than a bare
	// constant.
	why string
}

func errorMappingCases() []errorMappingCase {
	return []errorMappingCase{
		{
			name:     "unknown payment id",
			sentinel: "ErrPaymentNotFound",
			wantCode: codes.NotFound,
			invoke: func(t *testing.T) error {
				h, _, _ := newMappingHandler()
				_, err := h.RefundPayment(ctxAs(refundBookingHostID), &paymentsv1.RefundPaymentRequest{
					PaymentId:     mapPaymentID,
					BookingHostId: refundBookingHostID,
				})
				return err
			},
		},
		{
			name:     "actor is not the payable's recorder",
			sentinel: "ErrNotPaymentRecorder",
			wantCode: codes.PermissionDenied,
			why:      "a verified principal failing an object-level ownership check is 403-shaped, never 500-shaped (T5.2/T5.5's BOLA rule)",
			invoke: func(t *testing.T) error {
				h, repo, proc := newMappingHandler()
				seedPaidOnline(t, repo, proc, mapPaymentID, domain.PayableTypeBooking, fixtureBookingID)
				_, err := h.RefundPayment(ctxAs(refundOtherPlayerID), &paymentsv1.RefundPaymentRequest{
					PaymentId:     mapPaymentID,
					BookingHostId: refundBookingHostID,
				})
				return err
			},
		},
		{
			name:     "actor is not the payment's own recorded owner",
			sentinel: "ErrNotPaymentOwner",
			wantCode: codes.PermissionDenied,
			why: "T13.7 (#148): a verified principal who is not the actor the Payment records as its own. " +
				"Same code as ErrNotPaymentRecorder and for the same reason — the token verified, so this is " +
				"not Unauthenticated, and NotFound would make the RPC a payment-id enumeration oracle",
			invoke: func(t *testing.T) error {
				h, repo, proc := newMappingHandler()
				ref, err := proc.CreateIntent(context.Background(), domain.Money{Cents: 3000, Currency: "USD"}, fixtureBookingID)
				if err != nil {
					t.Fatalf("seed: CreateIntent: %v", err)
				}
				seedOnline(t, repo, mapPaymentID, domain.PayableTypeBooking, fixtureBookingID, ref, domain.StatusUnpaid, seededPaymentOwnerID)
				_, err = h.ConfirmOnlinePayment(ctxAs(refundOtherPlayerID), &paymentsv1.ConfirmOnlinePaymentRequest{
					PaymentId: mapPaymentID,
				})
				return err
			},
		},
		{
			name:     "second payment for the same payable",
			sentinel: "ErrPaymentAlreadyRecorded",
			wantCode: codes.AlreadyExists,
			invoke: func(t *testing.T) error {
				h, _, _ := newMappingHandler(mapPaymentID, mapOtherPaymentID)
				req := &paymentsv1.RecordOfflinePaymentRequest{
					PayableType:   paymentsv1.PayableType_PAYABLE_TYPE_BOOKING,
					PayableId:     fixtureBookingID,
					Amount:        &paymentsv1.Money{AmountCents: 3000, CurrencyCode: "USD"},
					BookingHostId: resolvedUserID(refundBookingHostID), // T28.1: resolved, not raw
				}
				if _, err := h.RecordOfflinePayment(ctxAs(refundBookingHostID), req); err != nil {
					t.Fatalf("first RecordOfflinePayment: %v", err)
				}
				_, err := h.RecordOfflinePayment(ctxAs(refundBookingHostID), req)
				return err
			},
		},
		{
			name:     "empty payable id",
			sentinel: "ErrEmptyPayableID",
			wantCode: codes.InvalidArgument,
			why:      "malformed request regardless of system state — the textbook InvalidArgument",
			invoke: func(t *testing.T) error {
				h, _, _ := newMappingHandler(mapPaymentID)
				_, err := h.RecordOfflinePayment(ctxAs(refundBookingHostID), &paymentsv1.RecordOfflinePaymentRequest{
					PayableType:   paymentsv1.PayableType_PAYABLE_TYPE_BOOKING,
					PayableId:     "",
					Amount:        &paymentsv1.Money{AmountCents: 3000, CurrencyCode: "USD"},
					BookingHostId: resolvedUserID(refundBookingHostID), // T28.1: resolved, not raw — authorization runs before this check
				})
				return err
			},
		},
		{
			name:     "payable type out of scope for refund",
			sentinel: "ErrInvalidPayableType",
			wantCode: codes.InvalidArgument,
			why: "no_show_fee (issue #130) is the one remaining out-of-scope payable type for RefundPayment " +
				"as of T16.4 — competition_entry moved into scope (closes the corrected #125), so this row now " +
				"exercises no_show_fee instead, mirroring refund_test.go's own OutOfScopePayableTypesRejected move",
			invoke: func(t *testing.T) error {
				h, repo, proc := newMappingHandler()
				seedPaidOnline(t, repo, proc, mapPaymentID, domain.PayableTypeNoShowFee, fixtureRegistrationID)
				_, err := h.RefundPayment(ctxAs(refundGameHostID), &paymentsv1.RefundPaymentRequest{
					PaymentId:  mapPaymentID,
					GameHostId: refundGameHostID,
				})
				return err
			},
		},
		{
			name:     "zero amount",
			sentinel: "ErrInvalidAmount",
			wantCode: codes.InvalidArgument,
			invoke: func(t *testing.T) error {
				h, _, _ := newMappingHandler(mapPaymentID)
				_, err := h.RecordOfflinePayment(ctxAs(refundBookingHostID), &paymentsv1.RecordOfflinePaymentRequest{
					PayableType:   paymentsv1.PayableType_PAYABLE_TYPE_BOOKING,
					PayableId:     fixtureBookingID,
					Amount:        &paymentsv1.Money{AmountCents: 0, CurrencyCode: "USD"},
					BookingHostId: resolvedUserID(refundBookingHostID), // T28.1: resolved, not raw — authorization runs before this check
				})
				return err
			},
		},
		{
			name:     "non-ISO-4217 currency",
			sentinel: "ErrInvalidCurrency",
			wantCode: codes.InvalidArgument,
			invoke: func(t *testing.T) error {
				h, _, _ := newMappingHandler(mapPaymentID)
				_, err := h.RecordOfflinePayment(ctxAs(refundBookingHostID), &paymentsv1.RecordOfflinePaymentRequest{
					PayableType:   paymentsv1.PayableType_PAYABLE_TYPE_BOOKING,
					PayableId:     fixtureBookingID,
					Amount:        &paymentsv1.Money{AmountCents: 3000, CurrencyCode: "US"},
					BookingHostId: resolvedUserID(refundBookingHostID), // T28.1: resolved, not raw — authorization runs before this check
				})
				return err
			},
		},
		{
			name:     "card declined on capture",
			sentinel: "ErrPaymentDeclined",
			wantCode: codes.FailedPrecondition,
			why:      "a decline is a legitimate business outcome, not a client input error and not a server bug (port.PaymentProcessor's doc comment)",
			invoke: func(t *testing.T) error {
				h, repo, proc := newMappingHandler()
				proc.SeedDecline(fixtureRegistrationID)
				ref, err := proc.CreateIntent(context.Background(), domain.Money{Cents: 3000, Currency: "USD"}, fixtureRegistrationID)
				if err != nil {
					t.Fatalf("seed: CreateIntent: %v", err)
				}
				// T28.1: recordedBy must be the RESOLVED User.ID so
				// ConfirmOnlinePayment's authorizeOnlineConfirmation passes
				// and the flow reaches the (declining) processor.
				seedOnline(t, repo, mapPaymentID, domain.PayableTypeRegistration, fixtureRegistrationID, ref, domain.StatusUnpaid, resolvedUserID(seededPaymentOwnerID))

				_, err = h.ConfirmOnlinePayment(ctxAs(seededPaymentOwnerID), &paymentsv1.ConfirmOnlinePaymentRequest{
					PaymentId: mapPaymentID,
				})
				return err
			},
		},
		{
			name:     "confirming an already-paid payment",
			sentinel: "ErrIllegalStatusTransition",
			wantCode: codes.FailedPrecondition,
			why: "gRPC defines InvalidArgument as a problem with the argument *regardless of the state of the system*; " +
				"a status-machine violation is state-dependent by definition, which is exactly FailedPrecondition " +
				"(\"the system is not in a state required for the operation's execution\")",
			invoke: func(t *testing.T) error {
				h, repo, proc := newMappingHandler()
				seedPaidOnline(t, repo, proc, mapPaymentID, domain.PayableTypeBooking, fixtureBookingID)
				_, err := h.ConfirmOnlinePayment(ctxAs(seededPaymentOwnerID), &paymentsv1.ConfirmOnlinePaymentRequest{
					PaymentId: mapPaymentID,
				})
				return err
			},
		},
		{
			name:     "invalid webhook signature",
			sentinel: "ErrWebhookSignatureInvalid",
			wantCode: codes.PermissionDenied,
			why: "T18.1 (#167): a forged/malformed Stripe webhook signature is a rejected caller, not a server bug — " +
				"the same discipline this package already applies to ErrNotPaymentRecorder/ErrNotPaymentOwner, " +
				"just with no principal involved at all on this RPC",
			invoke: func(t *testing.T) error {
				h, repo, proc := newMappingHandler()
				ref, err := proc.CreateIntent(context.Background(), domain.Money{Cents: 3000, Currency: "USD"}, fixtureBookingID)
				if err != nil {
					t.Fatalf("seed: CreateIntent: %v", err)
				}
				seedOnline(t, repo, mapPaymentID, domain.PayableTypeBooking, fixtureBookingID, ref, domain.StatusUnpaid, seededPaymentOwnerID)

				raw := []byte(`{"id":"evt_mapping_1","type":"payment_intent.succeeded"}`)
				_, err = h.ReceiveStripeWebhookEvent(context.Background(), &paymentsv1.ReceiveStripeWebhookEventRequest{
					RawPayload:      raw,
					SignatureHeader: "not-a-valid-signature",
					EventId:         "evt_mapping_1",
					EventType:       "payment_intent.succeeded",
					StripeReference: ref,
				})
				return err
			},
		},
		{
			name:     "processor unreachable on refund",
			sentinel: "ErrPaymentProcessorUnavailable",
			wantCode: codes.Unavailable,
			why: "the sentinel means one thing in both directions — the processor could not complete the request, so nothing moved. " +
				"Unavailable is gRPC's \"transient, may be corrected by retrying with a backoff\"; Internal describes broken invariants " +
				"in *our* system, which a third party's outage is not",
			invoke: func(t *testing.T) error {
				h, repo, _ := newMappingHandler()
				seedOnline(t, repo, mapPaymentID, domain.PayableTypeBooking, fixtureBookingID, mapUnknownIntentRef, domain.StatusPaid, seededPaymentOwnerID)
				_, err := h.RefundPayment(ctxAs(refundBookingHostID), &paymentsv1.RefundPaymentRequest{
					PaymentId:     mapPaymentID,
					BookingHostId: resolvedUserID(refundBookingHostID), // T28.1: resolved, not raw
				})
				return err
			},
		},
		{
			name:     "actor subject resolves to no User",
			sentinel: "ErrUserNotFound",
			wantCode: codes.PermissionDenied,
			why: "T28.1 (closes the Payments third of #164): a verified caller whose subject " +
				"matches no identity_users row. PermissionDenied, not NotFound (ADR-0014 §6, " +
				"restated by ADR-0017) and not Unauthenticated (the token verified) — mirrors " +
				"booking/facilities' identical mapping for their own ErrUserNotFound",
			invoke: func(t *testing.T) error {
				const unregistered = "auth0|never-called-createuser"
				h, _, _ := newMappingHandlerWithUnregisteredActor(unregistered, mapPaymentID)
				_, err := h.RecordOfflinePayment(ctxAs(unregistered), &paymentsv1.RecordOfflinePaymentRequest{
					PayableType:   paymentsv1.PayableType_PAYABLE_TYPE_BOOKING,
					PayableId:     fixtureBookingID,
					Amount:        &paymentsv1.Money{AmountCents: 3000, CurrencyCode: "USD"},
					BookingHostId: resolvedUserID(unregistered),
				})
				return err
			},
		},
	}
}

// TestPaymentsService_SentinelToCode pins every domain sentinel to exactly one
// status code, across the whole service.
func TestPaymentsService_SentinelToCode(t *testing.T) {
	t.Parallel()

	for _, tc := range errorMappingCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.invoke(t)
			if err == nil {
				t.Fatalf("%s: call succeeded, want %v (%s)", tc.sentinel, tc.wantCode, tc.why)
			}
			st, ok := status.FromError(err)
			if !ok {
				t.Fatalf("%s: error is not a gRPC status: %v", tc.sentinel, err)
			}
			if st.Code() != tc.wantCode {
				t.Fatalf("%s mapped to %v, want %v%s (err: %v)", tc.sentinel, st.Code(), tc.wantCode, reason(tc.why), err)
			}
		})
	}
}

func reason(why string) string {
	if why == "" {
		return ""
	}
	return " — " + why
}

// TestPaymentsService_SameSentinelSameCodeAcrossRPCs is issue #131's claim
// stated directly: it asserts the codes two *different* RPCs answer with are
// equal to each other for the same sentinel, rather than each matching a
// constant. Re-introducing a per-RPC mapper fails here even if its author also
// updates the table above to describe the divergence.
func TestPaymentsService_SameSentinelSameCodeAcrossRPCs(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		sentinel string
		// Two RPCs that can each surface the same sentinel.
		viaConfirm func(t *testing.T) error
		viaRefund  func(t *testing.T) error
	}{
		{
			name:     "illegal status transition",
			sentinel: "ErrIllegalStatusTransition",
			viaConfirm: func(t *testing.T) error {
				h, repo, proc := newMappingHandler()
				seedPaidOnline(t, repo, proc, mapPaymentID, domain.PayableTypeBooking, fixtureBookingID)
				_, err := h.ConfirmOnlinePayment(ctxAs(seededPaymentOwnerID), &paymentsv1.ConfirmOnlinePaymentRequest{
					PaymentId: mapPaymentID,
				})
				return err
			},
			viaRefund: func(t *testing.T) error {
				h, repo, _ := newMappingHandler()
				// Already refunded — refunding again is the same
				// state-machine violation from the other direction.
				seedOnline(t, repo, mapPaymentID, domain.PayableTypeBooking, fixtureBookingID, mapUnknownIntentRef, domain.StatusRefunded, seededPaymentOwnerID)
				_, err := h.RefundPayment(ctxAs(refundBookingHostID), &paymentsv1.RefundPaymentRequest{
					PaymentId:     mapPaymentID,
					BookingHostId: resolvedUserID(refundBookingHostID), // T28.1: resolved, not raw
				})
				return err
			},
		},
		{
			name:     "processor unreachable",
			sentinel: "ErrPaymentProcessorUnavailable",
			viaConfirm: func(t *testing.T) error {
				h, repo, _ := newMappingHandler()
				// T28.1: recordedBy must be resolved so the ownership check
				// passes and the flow reaches the (unreachable) processor.
				seedOnline(t, repo, mapPaymentID, domain.PayableTypeBooking, fixtureBookingID, mapUnknownIntentRef, domain.StatusUnpaid, resolvedUserID(seededPaymentOwnerID))
				_, err := h.ConfirmOnlinePayment(ctxAs(seededPaymentOwnerID), &paymentsv1.ConfirmOnlinePaymentRequest{
					PaymentId: mapPaymentID,
				})
				return err
			},
			viaRefund: func(t *testing.T) error {
				h, repo, _ := newMappingHandler()
				seedOnline(t, repo, mapPaymentID, domain.PayableTypeBooking, fixtureBookingID, mapUnknownIntentRef, domain.StatusPaid, seededPaymentOwnerID)
				_, err := h.RefundPayment(ctxAs(refundBookingHostID), &paymentsv1.RefundPaymentRequest{
					PaymentId:     mapPaymentID,
					BookingHostId: resolvedUserID(refundBookingHostID), // T28.1: resolved, not raw
				})
				return err
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			confirmCode := codeOf(t, "ConfirmOnlinePayment", tc.viaConfirm(t))
			refundCode := codeOf(t, "RefundPayment", tc.viaRefund(t))

			if confirmCode != refundCode {
				t.Fatalf("%s produced %v from ConfirmOnlinePayment but %v from RefundPayment — "+
					"one domain sentinel must mean one gRPC code regardless of which RPC surfaced it (issue #131)",
					tc.sentinel, confirmCode, refundCode)
			}
		})
	}
}

func codeOf(t *testing.T, what string, err error) codes.Code {
	t.Helper()

	if err == nil {
		t.Fatalf("%s succeeded, want an error to map", what)
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("%s returned a non-gRPC-status error: %v", what, err)
	}
	return st.Code()
}

// TestPaymentsService_HasExactlyOneErrorMapper is the structural guard against
// #131's actual shape. The inconsistency was not a typo in a mapping — it was a
// *second mapper* (`toRefundStatus`) added beside the shared one, which no
// assertion about codes would have flagged as long as both mappers agreed with
// their own tests. Adding another `func(error) error` to this handler now fails
// a test.
func TestPaymentsService_HasExactlyOneErrorMapper(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "handler.go", nil, 0)
	if err != nil {
		t.Fatalf("parse handler.go: %v", err)
	}

	var mappers []string
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv != nil {
			continue
		}
		if isErrorToErrorFunc(fn.Type) {
			mappers = append(mappers, fn.Name.Name)
		}
	}

	if len(mappers) != 1 || mappers[0] != "toStatus" {
		t.Fatalf("handler.go declares error mappers %v, want exactly [toStatus] — "+
			"a per-RPC error mapper is how issue #131's one-sentinel-two-codes split arose; "+
			"map the sentinel once in toStatus, or introduce a distinct sentinel if two RPCs "+
			"genuinely mean different things", mappers)
	}
}

// isErrorToErrorFunc reports whether sig is `func(error) error` — the shape of
// this package's status mappers, and narrow enough to exclude helpers like
// toProtoStatus(domain.Status).
func isErrorToErrorFunc(sig *ast.FuncType) bool {
	if sig.Params == nil || len(sig.Params.List) != 1 || len(sig.Params.List[0].Names) != 1 {
		return false
	}
	if sig.Results == nil || len(sig.Results.List) != 1 {
		return false
	}
	return isIdent(sig.Params.List[0].Type, "error") && isIdent(sig.Results.List[0].Type, "error")
}

func isIdent(expr ast.Expr, name string) bool {
	id, ok := expr.(*ast.Ident)
	return ok && id.Name == name
}

// TestSentinelToCodeTableCoversEveryDomainSentinel keeps the table above
// honest as the domain grows: a sentinel added to
// internal/payments/domain/errors.go with no mapping row falls through
// toStatus's default to Internal, which is a 500 for a condition the domain
// anticipated. That is a silent bug today; here it is a failing test.
func TestSentinelToCodeTableCoversEveryDomainSentinel(t *testing.T) {
	t.Parallel()

	covered := map[string]bool{}
	for _, tc := range errorMappingCases() {
		covered[tc.sentinel] = true
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "../../domain/errors.go", nil, 0)
	if err != nil {
		t.Fatalf("parse domain/errors.go: %v", err)
	}

	var missing []string
	ast.Inspect(file, func(n ast.Node) bool {
		spec, ok := n.(*ast.ValueSpec)
		if !ok {
			return true
		}
		for _, name := range spec.Names {
			if len(name.Name) > 3 && name.Name[:3] == "Err" && !covered[name.Name] {
				missing = append(missing, name.Name)
			}
		}
		return true
	})

	if len(missing) > 0 {
		t.Fatalf("domain sentinels with no row in errorMappingCases: %v — "+
			"an unmapped sentinel falls through toStatus's default to Internal (HTTP 500) "+
			"for a condition the domain explicitly anticipated", missing)
	}
}
