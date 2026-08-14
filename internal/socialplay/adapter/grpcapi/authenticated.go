package grpcapi

import (
	socialplayv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/socialplay/v1"
)

// AuthenticatedMethods lists the Social Play RPCs that may only be called by a
// verified caller. cmd/server composes this with every other context's list
// into one auth.MethodSet (T12 sprint plan A11 Ruling 2: the knowledge of
// which of *this* context's RPCs are public belongs here, next to the handlers
// that break if it is wrong — not in a single list in main.go).
//
// The names come from the generated *_FullMethodName constants rather than
// hand-written strings, for the reason facilities' equivalent file spells out:
// a hand-written name that drifted from the proto would silently stop matching
// and silently stop enforcing. Using the constant makes that a compile error.
//
// # Why each of these six requires a principal
//
//   - CreateGame: establishes Host-ship. Its host_id is no longer read off the
//     wire — the Game is hosted by the verified caller. This is the same
//     ownership-writing RPC CreateFacility is in Facilities (T12.7), and it
//     moves for the same reason: CancelGame and RecordMatchResult both compare
//     the caller against Game.HostID, so if CreateGame kept taking that value
//     from the wire, every new Game would be hosted by an arbitrary
//     client-supplied string while the checks compared against a verified
//     subject. Either real Hosts are locked out of their own Games, or a
//     caller hands Host-ship to a subject that is not theirs.
//   - RegisterForGame: player_id is likewise no longer read off the wire. It
//     is the value CancelRegistration compares against
//     (domain.Registration.Cancel's actorPlayerID-vs-PlayerID check), so the
//     identical argument applies one aggregate down: a Registration created
//     for a claimed player_id could never be cancelled by the verified human
//     it belongs to.
//   - JoinWaitlist: same, one aggregate further. A WaitlistEntry's player_id
//     becomes a Registration's player_id on promotion, so a claimed value here
//     propagates into the aggregate CancelRegistration guards.
//   - CancelRegistration: the object-level check
//     (domain.Registration.Cancel) previously compared the Registration's
//     player against a string the caller supplied about themselves.
//   - RecordMatchResult: guarded by domain.Game.EnsureHostOrGameAdmin, same
//     claimed-actor shape.
//   - CancelGame: shipped with a claimed-actor check in T12.4 and is named
//     explicitly in the T12.8 ticket as the RPC not to miss — this is the
//     ticket that makes its domain.Game.EnsureHost check rest on a verified
//     identity rather than a typed one. Cancelling is destructive and
//     Host-only (deliberately not delegated to Game Admins — see
//     CancelGameRequest's proto doc comment), which makes a claimed actor here
//     the sharpest of the six.
func AuthenticatedMethods() []string {
	return []string{
		socialplayv1.SocialPlayService_CreateGame_FullMethodName,
		socialplayv1.SocialPlayService_RegisterForGame_FullMethodName,
		socialplayv1.SocialPlayService_JoinWaitlist_FullMethodName,
		socialplayv1.SocialPlayService_CancelRegistration_FullMethodName,
		socialplayv1.SocialPlayService_RecordMatchResult_FullMethodName,
		socialplayv1.SocialPlayService_CancelGame_FullMethodName,
	}
}

// PublicMethods lists the Social Play RPCs that deliberately stay callable
// without a token.
//
// It exists so the split is *exhaustive and checkable* rather than implied by
// omission: authenticated_test.go asserts every method on the generated
// ServiceDesc appears in exactly one of these two lists, which turns "a new
// RPC was added and nobody decided whether it needs auth" from a silent
// default-to-public into a failing test.
//
// # Why each of these three stays public
//
//   - ListGames: the Discover & Join Games browse path (T8.9), reached by
//     players who have not signed in. Authenticating it would break a shipped
//     flow — the same reasoning that keeps Facilities' ListFacilities public.
//   - ListMatchesForGame: recorded Match results are a public fact about a
//     Game that already happened, and the message carries no per-caller data
//     (ADR-0012 keeps ratings off it entirely).
//   - ListRegistrationsForGame: public **not because that is right, but
//     because making it otherwise is out of this ticket's reach**, and leaving
//     that unsaid would be worse than saying it. This is the Host's
//     pending-cash-payments dashboard read (T8.10): it returns the roster —
//     every registrant's player_id and payment status — to anyone holding a
//     game_id. It has no actor field and no ownership check anywhere in the
//     domain, so there is nothing to migrate here; giving it one means adding
//     a Host-only read rule to the domain, and A11 Ruling 3 scopes this ticket
//     to the handler boundary precisely so it does not make domain changes in
//     passing. Requiring a bare token instead would be worse than either
//     option: it would look like authorization while granting every
//     authenticated user on the platform the same roster access an anonymous
//     one has today. Disclosed and tracked as a GitHub issue per the sprint's
//     A5 standing rule, exactly as T12.7 did for CreateBooking/CancelBooking.
func PublicMethods() []string {
	return []string{
		socialplayv1.SocialPlayService_ListGames_FullMethodName,
		socialplayv1.SocialPlayService_ListMatchesForGame_FullMethodName,
		socialplayv1.SocialPlayService_ListRegistrationsForGame_FullMethodName,
	}
}
