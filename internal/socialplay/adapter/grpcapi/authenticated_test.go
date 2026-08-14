package grpcapi_test

import (
	"testing"

	socialplayv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/socialplay/v1"
	"github.com/nhuthuynh/white-label/internal/socialplay/adapter/grpcapi"
)

// TestAuthenticatedAndPublicMethods_CoverEveryRPC is the guard that makes the
// authenticated/public split a decision rather than an omission (T12.8,
// copying T12.7's shape exactly).
//
// A method in neither list is public at runtime, silently. That is the
// dangerous direction of the default: adding an RPC and forgetting it needs a
// principal produces no compile error, no failing test, and a working
// endpoint. This test reads the *generated* ServiceDesc — the same source of
// truth grpc dispatches from, not a hand-maintained list that can drift from
// it — and fails if any of its methods is unclassified or classified twice.
func TestAuthenticatedAndPublicMethods_CoverEveryRPC(t *testing.T) {
	classified := make(map[string]int)
	for _, m := range grpcapi.AuthenticatedMethods() {
		classified[m]++
	}
	for _, m := range grpcapi.PublicMethods() {
		classified[m]++
	}

	serviceName := socialplayv1.SocialPlayService_ServiceDesc.ServiceName
	for _, m := range socialplayv1.SocialPlayService_ServiceDesc.Methods {
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

// TestAuthenticatedMethods_IncludesCancelGame pins the T12.8 ticket's item 4
// by name.
//
// CancelGame shipped in T12.4 with a claimed-actor check, one wave before this
// migration, which makes it the RPC most likely to be missed — it is not the
// one an implementer thinking about "the actor_player_id fields" has in mind,
// and the exhaustiveness test above would happily accept it in PublicMethods().
// This test rejects that specific outcome rather than trusting it not to
// happen.
func TestAuthenticatedMethods_IncludesCancelGame(t *testing.T) {
	for _, m := range grpcapi.AuthenticatedMethods() {
		if m == socialplayv1.SocialPlayService_CancelGame_FullMethodName {
			return
		}
	}
	t.Errorf("CancelGame is not in AuthenticatedMethods() — cancelling a Game is a destructive, "+
		"Host-only action and T12.8 is the ticket that makes its EnsureHost check rest on a "+
		"verified identity (ticket item 4). Got: %v", grpcapi.AuthenticatedMethods())
}

// TestAuthenticatedMethods_KeepsTheBrowsePathPublic pins the regression the
// migration tickets name: silently authenticating a currently-public read
// breaks a shipped flow. ListGames is Discover & Join Games (T8.9), reached by
// players who have not signed in.
func TestAuthenticatedMethods_KeepsTheBrowsePathPublic(t *testing.T) {
	authenticated := make(map[string]bool)
	for _, m := range grpcapi.AuthenticatedMethods() {
		authenticated[m] = true
	}

	for _, public := range []string{
		socialplayv1.SocialPlayService_ListGames_FullMethodName,
		socialplayv1.SocialPlayService_ListMatchesForGame_FullMethodName,
	} {
		if authenticated[public] {
			t.Errorf("%s is listed as authenticated — it is a public read and requiring a token breaks a shipped flow", public)
		}
	}
}
