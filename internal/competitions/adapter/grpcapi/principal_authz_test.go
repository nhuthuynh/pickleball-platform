// T12.8 — the verified-principal half of this context's authorization
// regression suite. authz_regression_test.go (T9.4/T9.6) proved the *ownership
// rule*: a caller who is not the Competition's Host may not cancel it. This
// file proves the thing that rule was resting on is no longer a lie: **where
// the acting identity comes from**.
//
// Before this ticket the actor was a string the caller put in the request
// body. "Only the Host may cancel a Competition" was therefore only ever "only
// a caller willing to type the Host's user ID" — a value readable off the
// public GetCompetition response. These tests assert the four cases the T12.8
// ticket requires:
//
//	(a) a valid principal for the real owner        -> succeeds
//	(b) a valid principal for a different user      -> PermissionDenied
//	(c) no principal at all                         -> Unauthenticated
//	(d) a wire actor_user_id claiming the real Host
//	    with a non-Host principal or none           -> rejected
//
// (c) and (b) must stay distinct codes: "I do not know who you are" versus "I
// know who you are and you may not do this" (ADR-0013 §5). (d) is the one a
// naive migration would silently break — a handler that read the principal but
// fell back to req.GetActorUserId() when there wasn't one would pass (a), (b)
// and (c) and still be exactly as exploitable as before. It was
// mutation-checked rather than assumed; see the PR body.
package grpcapi_test

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/nhuthuynh/white-label/internal/platform/auth"

	"github.com/nhuthuynh/white-label/internal/competitions/domain"
	competitionsv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/competitions/v1"
)

// scheduledStatus is the "nothing happened" state every rejected-call
// assertion below compares against.
func scheduledStatus() domain.Status { return domain.StatusScheduled }

// createCompetitionReq is the CreateCompetition fixture the principal cases
// vary only the context of. host_id is left populated with a value that is not
// the acting principal, so each use doubles as a check that it is ignored.
func createCompetitionReq() *competitionsv1.CreateCompetitionRequest {
	return &competitionsv1.CreateCompetitionRequest{
		HostId:         "auth0|not-the-caller",
		Name:           "Spring Doubles Open",
		Sessions:       []*competitionsv1.CompetitionSession{protoSession("2026-09-01T09:00:00Z", "2026-09-01T12:00:00Z", courtID(1))},
		Capacity:       8,
		GuestAllowance: 0,
		PaymentMethod:  competitionsv1.PaymentMethod_PAYMENT_METHOD_EITHER,
		EntryFee:       &competitionsv1.Money{AmountCents: 2500, CurrencyCode: "AUD"},
		Format:         competitionsv1.CompetitionFormat_COMPETITION_FORMAT_DOUBLES,
	}
}

// ctxAs builds the context the auth interceptors produce for a caller whose
// bearer token verified as subject. It goes through the real exported
// auth.ContextWithPrincipal rather than stuffing a context key directly, so
// these tests break if that contract changes.
func ctxAs(subject string) context.Context {
	return auth.ContextWithPrincipal(context.Background(), auth.Principal{
		Subject: subject,
		Issuer:  "https://issuer.test/",
	})
}

// anonymous is the context of a caller who presented no token, or one the
// verifier rejected — the observe-only interceptor leaves the context
// untouched in both cases, so they are indistinguishable here by design.
func anonymous() context.Context { return context.Background() }

const (
	hostSubject     = "auth0|host-1"
	playerSubject   = "auth0|player-1"
	attackerSubject = "auth0|attacker-9"
)

// requireCode asserts the exact gRPC code, and reports Internal separately — a
// 500-shaped answer to an authorization question is its own bug class (it
// means an error escaped toStatus), and the T9.4 tests this file joins already
// call it out that way.
func requireCode(t *testing.T, what string, err error, want codes.Code) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s succeeded, want %v", what, want)
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("%s returned a non-gRPC-status error: %v", what, err)
	}
	if st.Code() == codes.Internal && want != codes.Internal {
		t.Fatalf("%s mapped to Internal (500-shaped), want %v: %v", what, want, err)
	}
	if st.Code() != want {
		t.Fatalf("%s code = %v, want %v (%v)", what, st.Code(), want, st.Message())
	}
}

// --- (a) and (b): the principal decides ---------------------------------

func TestEnforcedRPCs_OwnerPrincipalSucceeds(t *testing.T) {
	h, _ := newTestHandler()
	competition := seedCompetition(t, h, hostSubject, 8, 0)

	if _, err := h.EnterCompetition(ctxAs(playerSubject), &competitionsv1.EnterCompetitionRequest{
		CompetitionId: competition.GetId(),
	}); err != nil {
		t.Fatalf("EnterCompetition under the entrant's own principal should succeed, got: %v", err)
	}

	if _, err := h.CancelCompetition(ctxAs(hostSubject), &competitionsv1.CancelCompetitionRequest{
		CompetitionId: competition.GetId(),
	}); err != nil {
		t.Errorf("CancelCompetition as the Host's principal should succeed, got: %v", err)
	}
}

func TestEnforcedRPCs_NonOwnerPrincipalIsPermissionDenied(t *testing.T) {
	h, repo := newTestHandler()
	competition := seedCompetition(t, h, hostSubject, 8, 0)

	_, err := h.CancelCompetition(ctxAs(attackerSubject), &competitionsv1.CancelCompetitionRequest{
		CompetitionId: competition.GetId(),
	})
	requireCode(t, "CancelCompetition with a non-Host principal", err, codes.PermissionDenied)

	if got := repo.competitions[competition.GetId()].Status; got != scheduledStatus() {
		t.Errorf("Competition.Status = %v after a rejected cancel — a rejected call wrote anyway", got)
	}
}

// --- (c): no principal is Unauthenticated, NOT PermissionDenied ---------

// TestEnforcedRPCs_NoPrincipalIsUnauthenticated pins the distinction ADR-0013
// §5 establishes. Asserting merely "an error" here would let a handler that
// answered PermissionDenied pass — and PermissionDenied tells an anonymous
// caller "you are known and refused", which is both wrong and unactionable
// (the fix is to authenticate, and that answer does not say so).
func TestEnforcedRPCs_NoPrincipalIsUnauthenticated(t *testing.T) {
	h, _ := newTestHandler()

	// A Competition must exist for the ownership check to be reachable at all,
	// so this test proves the principal check runs *first* rather than the
	// call failing for lack of a fixture.
	competition := seedCompetition(t, h, hostSubject, 8, 0)

	ctx := anonymous()

	_, err := h.CreateCompetition(ctx, createCompetitionReq())
	requireCode(t, "CreateCompetition with no principal", err, codes.Unauthenticated)

	_, err = h.EnterCompetition(ctx, &competitionsv1.EnterCompetitionRequest{
		CompetitionId: competition.GetId(),
	})
	requireCode(t, "EnterCompetition with no principal", err, codes.Unauthenticated)

	_, err = h.CancelCompetition(ctx, &competitionsv1.CancelCompetitionRequest{
		CompetitionId: competition.GetId(),
	})
	requireCode(t, "CancelCompetition with no principal", err, codes.Unauthenticated)
}

// TestPublicRPCs_StayReachableAnonymously is the other half of (c), and the
// one that would catch an over-broad migration. The share link especially:
// T9.5 built it so a recipient without an account can open it, so a token
// requirement here would delete the feature rather than secure it.
func TestPublicRPCs_StayReachableAnonymously(t *testing.T) {
	h, _ := newTestHandler()
	competition := seedCompetition(t, h, hostSubject, 8, 0)

	ctx := anonymous()

	if _, err := h.GetCompetition(ctx, &competitionsv1.GetCompetitionRequest{
		CompetitionId: competition.GetId(),
	}); err != nil {
		t.Errorf("GetCompetition anonymously should succeed, got: %v", err)
	}
	if _, err := h.ListCompetitions(ctx, &competitionsv1.ListCompetitionsRequest{}); err != nil {
		t.Errorf("ListCompetitions anonymously should succeed, got: %v", err)
	}

	// ListEntriesForCompetition used to be asserted here, as an RPC that
	// "should succeed" anonymously. T13.6 inverted that (partial fix for
	// #147): the roster is per-person data, not a browse path, and the
	// assertion that it stayed anonymously readable was pinning the leak in
	// place. Its replacement is immediately below, plus the full four-case
	// proof in roster_authz_test.go.
	if _, err := h.ListEntriesForCompetition(ctx, &competitionsv1.ListEntriesForCompetitionRequest{
		CompetitionId: competition.GetId(),
	}); err == nil {
		t.Error("ListEntriesForCompetition anonymously should be refused — the roster is not a public browse path (#147)")
	} else {
		requireCode(t, "ListEntriesForCompetition with no principal", err, codes.Unauthenticated)
	}
}

// --- (d) the load-bearing case: the wire field is ignored ----------------

// TestEnforcedRPCs_WireActorClaimingOwnershipIsIgnored is the whole point of
// the sprint. The request below carries actor_user_id = the real Host — the
// exact bytes that succeeded before this ticket — while the verified principal
// is either a different user or absent entirely.
//
// If either case succeeds, the migration is cosmetic: the wire field is still
// what authorizes the call, and an attacker who can read host_id off the
// public GetCompetition response can still cancel every Competition on the
// platform.
func TestEnforcedRPCs_WireActorClaimingOwnershipIsIgnored(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
		want codes.Code
	}{
		{
			// The principal says attacker, the body says the Host. The body
			// must lose, and the answer must be "you may not", not "who are
			// you".
			name: "non-owner principal, wire field claims the owner",
			ctx:  ctxAs(attackerSubject),
			want: codes.PermissionDenied,
		},
		{
			// No principal at all, body claims the Host. This is the precise
			// fallback the ticket forbids: a handler that trusted the wire
			// field when there was no principal would return OK here.
			name: "no principal, wire field claims the owner",
			ctx:  anonymous(),
			want: codes.Unauthenticated,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, repo := newTestHandler()
			competition := seedCompetition(t, h, hostSubject, 8, 0)

			_, err := h.CancelCompetition(tc.ctx, &competitionsv1.CancelCompetitionRequest{
				CompetitionId: competition.GetId(),
				ActorUserId:   hostSubject, // the lie
			})
			requireCode(t, "CancelCompetition", err, tc.want)

			if got := repo.competitions[competition.GetId()].Status; got != scheduledStatus() {
				t.Errorf("Competition.Status = %v — a rejected call cancelled it anyway", got)
			}
		})
	}
}

// TestCreateCompetition_HostComesFromPrincipalNotWire is case (d) for the RPC
// that *writes* Host-ship rather than checking it.
//
// A caller sends host_id = the victim. If that field were still honored, the
// attacker would have minted a Competition hosted by someone else — and would
// hold its share token, which CreateCompetition returns on the reasoning that
// this response is "the one moment the caller is provably the Host".
func TestCreateCompetition_HostComesFromPrincipalNotWire(t *testing.T) {
	h, _ := newTestHandler()

	req := createCompetitionReq()
	// nolint:staticcheck // SA1019: setting the deprecated field IS the test.
	req.HostId = hostSubject // the lie

	resp, err := h.CreateCompetition(ctxAs(attackerSubject), req)
	if err != nil {
		t.Fatalf("CreateCompetition with a valid principal should succeed: %v", err)
	}

	if want := resolvedUserID(attackerSubject); resp.GetCompetition().GetHostId() != want {
		t.Errorf("Competition.HostId = %q, want %q — host_id was taken from the wire, so a caller can create a Competition hosted by someone else", resp.GetCompetition().GetHostId(), want)
	}
}

// TestEnterCompetition_EntrantComesFromPrincipalNotWire is the same assertion
// for the entry path.
//
// This one reaches beyond this context: Payments' competition-entry
// authorization compares its actor against the stored entrant_player_id, so an
// entry created for a claimed player_id would decide who may pay for it.
func TestEnterCompetition_EntrantComesFromPrincipalNotWire(t *testing.T) {
	h, _ := newTestHandler()
	competition := seedCompetition(t, h, hostSubject, 8, 0)

	resp, err := h.EnterCompetition(ctxAs(attackerSubject), &competitionsv1.EnterCompetitionRequest{
		CompetitionId: competition.GetId(),
		PlayerId:      playerSubject, // the lie
	})
	if err != nil {
		t.Fatalf("EnterCompetition with a valid principal should succeed: %v", err)
	}

	if want := resolvedUserID(attackerSubject); resp.GetEntry().GetPlayerId() != want {
		t.Errorf("CompetitionEntry.PlayerId = %q, want %q — player_id was taken from the wire", resp.GetEntry().GetPlayerId(), want)
	}
}
