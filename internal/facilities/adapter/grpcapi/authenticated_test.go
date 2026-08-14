package grpcapi_test

import (
	"testing"

	"github.com/nhuthuynh/white-label/internal/facilities/adapter/grpcapi"
	facilitiesv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/facilities/v1"
)

// TestAuthenticatedAndPublicMethods_CoverEveryRPC is the guard that makes the
// authenticated/public split a decision rather than an omission (T12.7).
//
// A method that is in neither list is public at runtime, silently. That is the
// dangerous direction of the default: adding an RPC and forgetting it needs a
// principal produces no compile error, no failing test, and a working
// endpoint. This test reads the *generated* ServiceDesc — the same source of
// truth grpc dispatches from — and fails if any of its methods is unclassified
// or classified twice.
func TestAuthenticatedAndPublicMethods_CoverEveryRPC(t *testing.T) {
	classified := make(map[string]int)
	for _, m := range grpcapi.AuthenticatedMethods() {
		classified[m]++
	}
	for _, m := range grpcapi.PublicMethods() {
		classified[m]++
	}

	serviceName := facilitiesv1.FacilitiesService_ServiceDesc.ServiceName
	for _, m := range facilitiesv1.FacilitiesService_ServiceDesc.Methods {
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

// TestAuthenticatedMethods_KeepsTheBrowsePathPublic pins the specific
// regression the T12.7 ticket names: silently authenticating a currently-public
// read would break a shipped flow. ListFacilities and GetFacility are the
// Discover/Browse path (T7.5), reached by players who have not signed in.
func TestAuthenticatedMethods_KeepsTheBrowsePathPublic(t *testing.T) {
	authenticated := make(map[string]bool)
	for _, m := range grpcapi.AuthenticatedMethods() {
		authenticated[m] = true
	}

	for _, public := range []string{
		facilitiesv1.FacilitiesService_ListFacilities_FullMethodName,
		facilitiesv1.FacilitiesService_GetFacility_FullMethodName,
	} {
		if authenticated[public] {
			t.Errorf("%s is listed as authenticated — it is a public browse read and requiring a token breaks a shipped flow", public)
		}
	}
}
