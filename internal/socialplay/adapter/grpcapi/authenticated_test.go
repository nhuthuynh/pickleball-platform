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

// TestAuthenticatedMethods_IncludesListRegistrationsForGame pins T13.6's half
// of the fix for #147 the same way the test above pins T12.8's.
//
// The handler defends itself (it calls actor(ctx) and refuses without a
// principal), so moving this RPC back to PublicMethods would NOT show up as a
// roster leak in any behavioural test — it would only silently stop the
// interceptor requiring a token, leaving the classification list and the
// handler disagreeing about what this endpoint is. That is exactly the kind of
// quiet drift the two lists exist to make visible, so it gets an explicit
// assertion rather than relying on the exhaustiveness test, which would happily
// accept either placement.
func TestAuthenticatedMethods_IncludesListRegistrationsForGame(t *testing.T) {
	for _, m := range grpcapi.AuthenticatedMethods() {
		if m == socialplayv1.SocialPlayService_ListRegistrationsForGame_FullMethodName {
			return
		}
	}
	t.Errorf("ListRegistrationsForGame is not in AuthenticatedMethods() — it returns per-person "+
		"roster data (player_id, payment_status, guest_count) and T13.6 made it Host-only "+
		"(partial fix for #147). Got: %v", grpcapi.AuthenticatedMethods())
}

// TestAuthenticatedMethods_IncludesListRegistrationsForActor pins T67.2's
// Player-facing read (part of #353) the same way the two tests above pin
// T12.8's and T13.6's. It is the THIRD RPC of this class, and it needs its own
// assertion for the identical reason the comment above gives.
//
// # Why the exhaustiveness test is not enough, measured rather than assumed
//
// TestAuthenticatedAndPublicMethods_CoverEveryRPC rejects only ABSENCE from
// both lists. T67.2's PR verified that deleting this entry fails that test —
// which is true, and was the wrong thing to check. The PE review found the gap:
// **MOVING** the entry into PublicMethods(), i.e. making the RPC public, leaves
// `internal/socialplay/adapter/grpcapi` and `internal/platform/...` entirely
// green, because the exhaustiveness test happily accepts either placement.
//
// Nothing behavioural would catch it either. The handler defends itself — it
// calls actor(ctx) and returns Unauthenticated without a principal — so a
// public classification would not leak one Player's registrations to another.
// It would instead leave the interceptor not requiring a token for an endpoint
// whose entire query predicate IS the verified caller, so the declared policy
// and the enforced one would silently disagree. That drift is exactly what the
// two lists exist to make visible, which is why this gets an explicit pin.
func TestAuthenticatedMethods_IncludesListRegistrationsForActor(t *testing.T) {
	for _, m := range grpcapi.AuthenticatedMethods() {
		if m == socialplayv1.SocialPlayService_ListRegistrationsForActor_FullMethodName {
			return
		}
	}
	t.Errorf("ListRegistrationsForActor is not in AuthenticatedMethods() — the verified "+
		"principal is this read's ENTIRE query predicate (WHERE r.player_id = the caller), "+
		"so without a required token there is no scope at all and T67.2 made it "+
		"authenticated. Got: %v", grpcapi.AuthenticatedMethods())
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
