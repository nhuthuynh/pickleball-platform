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
// # Why each of these nine requires a principal
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
//   - ListRegistrationsForGame: moved here from PublicMethods by T13.6
//     (partial fix for #147). It is the Host's pending-cash-payments dashboard
//     read (T8.10) and it returns per-person data — every registrant's
//     player_id, payment_status and guest_count. As a public method it handed
//     that to anyone holding a game_id, and game_id is readable off the public
//     ListGames response, so the leak was enumerable rather than theoretical.
//     T12.8 could not fix it because there was no check to re-plumb; T13.6
//     adds the check itself (Host-only, domain.Game.EnsureHost, applied in
//     app.Service). Note this is the one entry here whose reason is a *read*
//     rather than a write: the T12.8 six all needed a principal because they
//     wrote or acted on an aggregate, whereas this one needs it because the
//     response body is private data.
//   - CancelGame: shipped with a claimed-actor check in T12.4 and is named
//     explicitly in the T12.8 ticket as the RPC not to miss — this is the
//     ticket that makes its domain.Game.EnsureHost check rest on a verified
//     identity rather than a typed one. Cancelling is destructive and
//     Host-only (deliberately not delegated to Game Admins — see
//     CancelGameRequest's proto doc comment), which makes a claimed actor here
//     the sharpest of the six.
//   - AssignGameAdmin / RevokeGameAdmin: added by T14.4 (partial fix for
//     #168), and the pair whose presence here is load-bearing rather than
//     merely correct. They are Host-only writes that decide **who else is
//     entitled to act on this Game** — the store every other admin-aware rule
//     will read. If either were callable without a principal, the durable
//     store would be worse than the caller-supplied list it replaces: a
//     forgeable claim at least dies with its request, whereas an anonymous
//     write here would persist a forged entitlement for every later check to
//     honour. The Host-only rule itself lives in the domain
//     (domain.AssignGameAdmin / EnsureMayRevokeGameAdmin, via
//     Game.EnsureHost); this list is what guarantees the actor it compares
//     against is verified rather than typed.
func AuthenticatedMethods() []string {
	return []string{
		socialplayv1.SocialPlayService_CreateGame_FullMethodName,
		socialplayv1.SocialPlayService_RegisterForGame_FullMethodName,
		socialplayv1.SocialPlayService_JoinWaitlist_FullMethodName,
		socialplayv1.SocialPlayService_CancelRegistration_FullMethodName,
		socialplayv1.SocialPlayService_RecordMatchResult_FullMethodName,
		socialplayv1.SocialPlayService_ListRegistrationsForGame_FullMethodName,
		socialplayv1.SocialPlayService_CancelGame_FullMethodName,
		socialplayv1.SocialPlayService_AssignGameAdmin_FullMethodName,
		socialplayv1.SocialPlayService_RevokeGameAdmin_FullMethodName,
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
// # Why each of these two stays public
//
//   - ListGames: the Discover & Join Games browse path (T8.9), reached by
//     players who have not signed in. Authenticating it would break a shipped
//     flow — the same reasoning that keeps Facilities' ListFacilities public.
//   - ListMatchesForGame: recorded Match results are a public fact about a
//     Game that already happened, and the message carries no per-caller data
//     (ADR-0012 keeps ratings off it entirely).
//
// ListRegistrationsForGame used to be the third entry here, carrying a long
// comment that said in so many words that it was public *not because that was
// right* but because T12.8 could not reach it. T13.6 reached it: it is now in
// AuthenticatedMethods above, Host-only. The disclosure is kept in git history
// rather than restated, but the shape of it is worth remembering — a comment
// honestly recording a known hole is not a substitute for closing it, and the
// thing that eventually closed it was the tracked issue (#147), not the
// comment.
func PublicMethods() []string {
	return []string{
		socialplayv1.SocialPlayService_ListGames_FullMethodName,
		socialplayv1.SocialPlayService_ListMatchesForGame_FullMethodName,
	}
}
