package grpcapi_test

import (
	"testing"

	paymentsv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/payments/v1"
	"github.com/nhuthuynh/white-label/internal/payments/adapter/grpcapi"
)

// TestAuthenticatedAndPublicMethods_CoverEveryRPC is the guard that makes the
// authenticated/public split a decision rather than an omission (T12.8,
// copying T12.7's shape exactly).
//
// A method in neither list is public at runtime, silently. That is the
// dangerous direction of the default: adding an RPC and forgetting it needs a
// principal produces no compile error, no failing test, and a working
// endpoint — on a service that moves money. This test reads the *generated*
// ServiceDesc, the same source of truth grpc dispatches from, and fails if any
// of its methods is unclassified or classified twice.
func TestAuthenticatedAndPublicMethods_CoverEveryRPC(t *testing.T) {
	classified := make(map[string]int)
	for _, m := range grpcapi.AuthenticatedMethods() {
		classified[m]++
	}
	for _, m := range grpcapi.PublicMethods() {
		classified[m]++
	}

	serviceName := paymentsv1.PaymentsService_ServiceDesc.ServiceName
	for _, m := range paymentsv1.PaymentsService_ServiceDesc.Methods {
		full := "/" + serviceName + "/" + m.MethodName
		switch classified[full] {
		case 0:
			t.Errorf("RPC %s is in neither AuthenticatedMethods() nor PublicMethods() — "+
				"it is silently public. Decide which it is and add it to the right list.", full)
		case 1: // exactly one list, as required
		default:
			t.Errorf("RPC %s appears in both AuthenticatedMethods() and PublicMethods()", full)
		}
		delete(classified, full)
	}

	for stale := range classified {
		t.Errorf("%q is classified but is not an RPC on this service — a renamed or removed method leaves a policy entry that enforces nothing", stale)
	}
}

// TestAuthenticatedMethods_CoverEveryActorBearingRPC pins the specific way
// this service could regress: an RPC that reads an actor_user_id off the wire
// while not requiring a principal is the exact pre-T12.8 state, and it would
// look fine to the exhaustiveness test above as long as somebody had written
// it into PublicMethods().
func TestAuthenticatedMethods_CoverEveryActorBearingRPC(t *testing.T) {
	authenticated := make(map[string]bool)
	for _, m := range grpcapi.AuthenticatedMethods() {
		authenticated[m] = true
	}

	for _, m := range []string{
		paymentsv1.PaymentsService_RecordOfflinePayment_FullMethodName,
		paymentsv1.PaymentsService_CreateOnlinePayment_FullMethodName,
		paymentsv1.PaymentsService_RefundPayment_FullMethodName,
		// T13.7 (closes #148): ConfirmOnlinePayment reads an actor too, as of
		// this ticket. It is listed here rather than left implicit because
		// this list is what stops it drifting back out of
		// AuthenticatedMethods() — moving it would then pass the
		// exhaustiveness test above (PublicMethods() is a legal home for any
		// RPC) and silently restore issue #148.
		paymentsv1.PaymentsService_ConfirmOnlinePayment_FullMethodName,
	} {
		if !authenticated[m] {
			t.Errorf("%s reads an actor for an authorization branch but is not in AuthenticatedMethods() — "+
				"that is the claimed-actor state T12.8 exists to end", m)
		}
	}
}
