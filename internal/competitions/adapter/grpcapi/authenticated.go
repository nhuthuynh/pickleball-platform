package grpcapi

import (
	competitionsv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/competitions/v1"
)

// AuthenticatedMethods lists the Competitions RPCs that may only be called by
// a verified caller. cmd/server composes this with every other context's list
// into one auth.MethodSet (T12 sprint plan A11 Ruling 2: the knowledge of
// which of *this* context's RPCs are public belongs here, next to the handlers
// that break if it is wrong — not in a single list in main.go).
//
// The names come from the generated *_FullMethodName constants rather than
// hand-written strings, so a renamed RPC is a compile error instead of a
// policy entry that silently stops matching and silently stops enforcing.
//
// # Why each of these four requires a principal
//
//   - CreateCompetition: establishes Host-ship. Its host_id is no longer read
//     off the wire — the Competition is hosted by the verified caller. Same
//     ownership-writing RPC CreateFacility is in Facilities (T12.7) and
//     CreateGame is in Social Play, and it moves for the same reason:
//     CancelCompetition compares the caller against Competition.HostID, so a
//     Host-ship written from a caller-supplied string would either lock real
//     Hosts out of their own Competitions or hand Host-ship to a subject that
//     is not theirs. It also mints and returns the share token, which is the
//     one moment the caller is provably the Host — a claim that means nothing
//     if the "Host" was whoever the request named.
//   - EnterCompetition: player_id is the entrant. It is no longer read off the
//     wire because Payments' competition-entry authorization compares its
//     actor against exactly this stored value (entrant_player_id), so an entry
//     created for a claimed player_id would decide who may pay for it.
//     Note this does NOT add an ownership check to entering — any authenticated
//     Player may still enter a published Competition, which is the deliberate
//     asymmetry EnterCompetitionRequest's proto comment describes (entering is
//     the invitation being accepted, not an act *on* the Competition). What
//     changed is only that the entrant is now who they say they are.
//   - CancelCompetition: guarded by the Host-only check its proto comment
//     describes, which until now compared the Host against a string the caller
//     supplied about themselves.
//   - ListEntriesForCompetition: moved here from PublicMethods by T13.6
//     (partial fix for #147). The Host-facing roster read returns every
//     entrant's player_id and payment status, and as a public method it handed
//     that to anyone holding a competition_id — a value the public
//     ListCompetitions and GetCompetition responses supply, which is what made
//     the leak enumerable. T12.8 could not fix it because there was no check
//     to re-plumb; T13.6 adds the check itself (Host-only,
//     domain.Competition.EnsureHost, applied in app.Service). It is the one
//     entry here whose reason is a *read*: the other three need a principal
//     because they write or act on an aggregate, this one because the response
//     body is private data.
func AuthenticatedMethods() []string {
	return []string{
		competitionsv1.CompetitionsService_CreateCompetition_FullMethodName,
		competitionsv1.CompetitionsService_EnterCompetition_FullMethodName,
		competitionsv1.CompetitionsService_CancelCompetition_FullMethodName,
		competitionsv1.CompetitionsService_ListEntriesForCompetition_FullMethodName,
	}
}

// PublicMethods lists the Competitions RPCs that deliberately stay callable
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
//   - ListCompetitions, GetCompetition: the browse path and the public detail
//     page behind it, reached by players who have not signed in. Requiring a
//     token would break a shipped flow, which the migration tickets name
//     explicitly as the thing not to do.
//   - GetCompetitionByShareToken: public *by construction* (T9.5). The whole
//     point of a shareable registration link is that the recipient can open it
//     before having an account; the token is the credential. Authenticating it
//     would defeat the feature rather than secure it — and note the token is
//     deliberately absent from the Competition message so it cannot leak
//     through this or any other read.
//
// ListEntriesForCompetition used to be the fourth entry here, carrying a long
// comment that said in so many words that it was public *not because that was
// right* but because T12.8 could not reach it. T13.6 reached it: it is now in
// AuthenticatedMethods above, Host-only. Same note as Social Play's twin — an
// honest comment about a known hole is not a substitute for closing it, and
// what closed it was the tracked issue (#147).
func PublicMethods() []string {
	return []string{
		competitionsv1.CompetitionsService_ListCompetitions_FullMethodName,
		competitionsv1.CompetitionsService_GetCompetition_FullMethodName,
		competitionsv1.CompetitionsService_GetCompetitionByShareToken_FullMethodName,
	}
}
