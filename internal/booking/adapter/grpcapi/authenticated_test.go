package grpcapi_test

import (
	"testing"

	"github.com/nhuthuynh/white-label/internal/booking/adapter/grpcapi"
	bookingv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/booking/v1"
)

// TestAuthenticatedAndPublicMethods_CoverEveryRPC is the guard that makes the
// authenticated/public split a decision rather than an omission (T12.7).
//
// A method in neither list is public at runtime, silently: adding an RPC and
// forgetting it needs a principal produces no compile error, no failing test,
// and a working endpoint. This reads the *generated* ServiceDesc — the same
// source of truth grpc dispatches from — and fails if any method is
// unclassified, double-classified, or classified but no longer real.
func TestAuthenticatedAndPublicMethods_CoverEveryRPC(t *testing.T) {
	classified := make(map[string]int)
	for _, m := range grpcapi.AuthenticatedMethods() {
		classified[m]++
	}
	for _, m := range grpcapi.PublicMethods() {
		classified[m]++
	}

	serviceName := bookingv1.BookingService_ServiceDesc.ServiceName
	for _, m := range bookingv1.BookingService_ServiceDesc.Methods {
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

// TestAuthenticatedMethods_KeepsPublicReadsPublic pins the specific regression
// the T12.7 ticket names: silently authenticating a currently-public read
// would break a shipped flow. GetQuote and ListCourtBookings are named in the
// ticket text itself; ListDiscountRulesForFacility is the same browse path.
func TestAuthenticatedMethods_KeepsPublicReadsPublic(t *testing.T) {
	authenticated := make(map[string]bool)
	for _, m := range grpcapi.AuthenticatedMethods() {
		authenticated[m] = true
	}

	for _, public := range []string{
		bookingv1.BookingService_GetQuote_FullMethodName,
		bookingv1.BookingService_ListCourtBookings_FullMethodName,
		bookingv1.BookingService_ListDiscountRulesForFacility_FullMethodName,
	} {
		if authenticated[public] {
			t.Errorf("%s is listed as authenticated — it is a public read and requiring a token breaks a shipped flow", public)
		}
	}
}
