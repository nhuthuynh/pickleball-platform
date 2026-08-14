package grpcapi_test

import (
	"testing"

	"github.com/nhuthuynh/white-label/internal/competitions/adapter/grpcapi"
	competitionsv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/competitions/v1"
)

// TestAuthenticatedAndPublicMethods_CoverEveryRPC is the guard that makes the
// authenticated/public split a decision rather than an omission (T12.8,
// copying T12.7's shape exactly).
//
// A method in neither list is public at runtime, silently. That is the
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

	serviceName := competitionsv1.CompetitionsService_ServiceDesc.ServiceName
	for _, m := range competitionsv1.CompetitionsService_ServiceDesc.Methods {
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

// TestAuthenticatedMethods_KeepsTheShareLinkAndBrowsePathPublic pins the two
// regressions this service is most exposed to.
//
// The share link is the sharper of the two: T9.5 built it precisely so a
// recipient without an account can open it, and the token *is* the credential.
// Quietly authenticating it would not secure the feature, it would delete it.
func TestAuthenticatedMethods_KeepsTheShareLinkAndBrowsePathPublic(t *testing.T) {
	authenticated := make(map[string]bool)
	for _, m := range grpcapi.AuthenticatedMethods() {
		authenticated[m] = true
	}

	for _, public := range []string{
		competitionsv1.CompetitionsService_GetCompetitionByShareToken_FullMethodName,
		competitionsv1.CompetitionsService_ListCompetitions_FullMethodName,
		competitionsv1.CompetitionsService_GetCompetition_FullMethodName,
	} {
		if authenticated[public] {
			t.Errorf("%s is listed as authenticated — it is a public read and requiring a token breaks a shipped flow", public)
		}
	}
}
