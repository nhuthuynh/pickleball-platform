// T12.8 — the verified-principal half of this context's authorization
// regression suite. authz_regression_test.go (T6.4/T6.5),
// competition_entry_authz_regression_test.go (T10.6) and refund_test.go
// (T12.3) proved the *ownership rules*: a caller who is not the Booking's
// Host, the Game's Host, an assigned Game Admin, the entrant, or an assigned
// Competition Admin is rejected. This file proves the thing those rules were
// resting on is no longer a lie: **where the acting identity comes from**.
//
// Before this ticket the actor was a string the caller put in the request
// body, and money is where that mattered most: "only the Game's Host may mark
// this Registration paid" was only ever "only a caller willing to type the
// Host's user ID" — a value readable off Social Play's public ListGames
// response. These tests assert the four cases the T12.8 ticket requires:
//
//	(a) a valid principal for the real owner        -> succeeds
//	(b) a valid principal for a different user      -> PermissionDenied
//	(c) no principal at all                         -> Unauthenticated
//	(d) a wire actor_user_id claiming the real owner
//	    with a non-owner principal or none          -> rejected
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

	"github.com/nhuthuynh/white-label/internal/payments/adapter/grpcapi"
	"github.com/nhuthuynh/white-label/internal/payments/adapter/stripestub"
	"github.com/nhuthuynh/white-label/internal/payments/app"
	"github.com/nhuthuynh/white-label/internal/payments/domain"
	"github.com/nhuthuynh/white-label/internal/platform/auth"

	paymentsv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/payments/v1"
)

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
	attackerSubject = "auth0|attacker-9"

	// fixturePaymentID is UUID-shaped for the reason the
	// fixtureBookingID/fixtureRegistrationID block above states: every
	// caller-supplied id crossing this boundary meets a uuidShape guard.
	fixturePaymentID = "6ba7b810-0000-4000-8000-000000000009"
)

// requireCode asserts the exact gRPC code, and reports Internal separately — a
// 500-shaped answer to an authorization question is its own bug class (it
// means an error escaped toStatus), and the T6.4 tests this file joins already
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

// offlineReq is the Registration-payable RecordOfflinePayment fixture every
// case below varies only the context of. game_host_id is set to hostSubject
// too, on principle — this file's whole point is proving the field is
// IGNORED — but as of T16.2 it has no effect either way: the caller entitled
// to record this payment is whoever newPrincipalAuthzHandler below resolved
// as fixtureGameID's real Host, not whoever this field names.
func offlineReq() *paymentsv1.RecordOfflinePaymentRequest {
	return &paymentsv1.RecordOfflinePaymentRequest{
		PayableType: paymentsv1.PayableType_PAYABLE_TYPE_REGISTRATION,
		PayableId:   fixtureRegistrationID,
		Amount:      offlineFixtureAmount(),
		GameHostId:  hostSubject,
	}
}

// newPrincipalAuthzHandler builds a handler like newTestHandler
// (authz_regression_test.go), but with the resolver fakes configured so
// fixtureRegistrationID's Game is hosted by hostID — parameterized, unlike
// newTestHandler's fixed "host-1" default, because this file's tests need
// the *principal*, not the (now-ignored) wire game_host_id, to be what the
// resolvers report, and different tests below need different hosts:
// hostSubject for the "real owner" cases, attackerSubject for
// TestRecordOfflinePayment_RecordedByComesFromPrincipalNotWire's "the
// attacker legitimately hosts this Game" case (T16.2, closes #168).
func newPrincipalAuthzHandler(hostID string, seedIDs ...string) (*grpcapi.Handler, *fakeRepository) {
	repo := newFakeRepository()
	regs, games, admins := newAuthzResolverFixtures(hostID)
	svc := app.NewService(app.ServiceOptions{
		Payments:           repo,
		IDs:                &fixedIDs{ids: seedIDs},
		RegistrationLookup: regs,
		GameLookup:         games,
		GameAdminReader:    admins,
	})
	return grpcapi.NewHandler(svc), repo
}

// newPrincipalAuthzHandlerWithProcessor is newPrincipalAuthzHandler plus a
// stripestub.Processor, for the one case in this file
// (TestRefundPayment_WireActorClaimingOwnershipIsIgnored) that seeds a real
// Payment via RecordOfflinePayment and then refunds it.
func newPrincipalAuthzHandlerWithProcessor(hostID string, seedIDs ...string) (*grpcapi.Handler, *fakeRepository) {
	repo := newFakeRepository()
	regs, games, admins := newAuthzResolverFixtures(hostID)
	svc := app.NewService(app.ServiceOptions{
		Payments:           repo,
		IDs:                &fixedIDs{ids: seedIDs},
		Processor:          stripestub.NewProcessor(),
		RegistrationLookup: regs,
		GameLookup:         games,
		GameAdminReader:    admins,
	})
	return grpcapi.NewHandler(svc), repo
}

// newPrincipalAuthzOnlineHandler mirrors newPrincipalAuthzHandler, but for
// CreateOnlinePayment's own PayableTypeCompetitionEntry authorization branch
// (T17.1, closes #198): the resolver fakes are configured so
// fixtureCompetitionEntryID's entrant is playerID — parameterized, like
// newPrincipalAuthzHandler's hostID, because this file's tests need the
// *principal*, not the (now-ignored) wire entrant_player_id, to be what
// port.EntryLookup reports. Needs its own Processor, unlike
// newPrincipalAuthzHandler's offline-only callers, since CreateOnlinePayment
// reaches port.PaymentProcessor.CreateIntent on the authorized path.
func newPrincipalAuthzOnlineHandler(playerID string, seedIDs ...string) (*grpcapi.Handler, *fakeRepository) {
	repo := newFakeRepository()
	entries, compAdmins := newEntryAuthzResolverFixtures(playerID)
	svc := app.NewService(app.ServiceOptions{
		Payments:               repo,
		IDs:                    &fixedIDs{ids: seedIDs},
		Processor:              stripestub.NewProcessor(),
		EntryLookup:            entries,
		CompetitionAdminReader: compAdmins,
	})
	return grpcapi.NewHandler(svc), repo
}

// onlineReq sets entrant_player_id, but T17.1 (closes #198) means that field
// no longer decides anything: the entrant paying for their own entry is the
// allowed case, and it is a caller's principal — resolved server-side via
// port.EntryLookup against whichever handler built this request's Service —
// that decides whether the caller *is* that entrant, not this wire value.
// Retained set, deprecated, purely so tests below can show it has no effect
// (mirroring offlineReq's identical game_host_id comment above).
func onlineReq() *paymentsv1.CreateOnlinePaymentRequest {
	return &paymentsv1.CreateOnlinePaymentRequest{
		PayableType: paymentsv1.PayableType_PAYABLE_TYPE_COMPETITION_ENTRY,
		PayableId:   fixtureCompetitionEntryID,
		Amount:      offlineFixtureAmount(),
		// nolint:staticcheck // SA1019: setting the deprecated field IS the point — see the doc comment above.
		EntrantPlayerId: hostSubject,
	}
}

// --- (a) and (b): the principal decides ---------------------------------

func TestEnforcedRPCs_OwnerPrincipalSucceeds(t *testing.T) {
	h, repo := newPrincipalAuthzHandler(hostSubject, "pay-1")

	if _, err := h.RecordOfflinePayment(ctxAs(hostSubject), offlineReq()); err != nil {
		t.Fatalf("RecordOfflinePayment as the Game Host's principal should succeed, got: %v", err)
	}
	if len(repo.byID) != 1 {
		t.Errorf("repo has %d Payments, want 1 after the Host's successful call", len(repo.byID))
	}

	h2, _ := newPrincipalAuthzOnlineHandler(hostSubject, "pay-2")
	if _, err := h2.CreateOnlinePayment(ctxAs(hostSubject), onlineReq()); err != nil {
		t.Errorf("CreateOnlinePayment as the entrant's own principal should succeed, got: %v", err)
	}
}

func TestEnforcedRPCs_NonOwnerPrincipalIsPermissionDenied(t *testing.T) {
	h, repo := newTestHandler("pay-1")

	_, err := h.RecordOfflinePayment(ctxAs(attackerSubject), offlineReq())
	requireCode(t, "RecordOfflinePayment with a non-Host principal", err, codes.PermissionDenied)

	_, err = h.CreateOnlinePayment(ctxAs(attackerSubject), onlineReq())
	requireCode(t, "CreateOnlinePayment with a non-entrant principal", err, codes.PermissionDenied)

	if len(repo.byID) != 0 {
		t.Errorf("repo has %d Payments after rejected calls, want 0 — a rejected call must have no side effect", len(repo.byID))
	}
}

// --- (c): no principal is Unauthenticated, NOT PermissionDenied ---------

// TestEnforcedRPCs_NoPrincipalIsUnauthenticated pins the distinction ADR-0013
// §5 establishes. Asserting merely "an error" here would let a handler that
// answered PermissionDenied pass — and PermissionDenied tells an anonymous
// caller "you are known and refused", which is both wrong and unactionable
// (the fix is to authenticate, and that answer does not say so).
func TestEnforcedRPCs_NoPrincipalIsUnauthenticated(t *testing.T) {
	h, repo := newTestHandler("pay-1")

	ctx := anonymous()

	_, err := h.RecordOfflinePayment(ctx, offlineReq())
	requireCode(t, "RecordOfflinePayment with no principal", err, codes.Unauthenticated)

	_, err = h.CreateOnlinePayment(ctx, onlineReq())
	requireCode(t, "CreateOnlinePayment with no principal", err, codes.Unauthenticated)

	_, err = h.RefundPayment(ctx, &paymentsv1.RefundPaymentRequest{
		PaymentId:  fixtureRegistrationID,
		GameHostId: hostSubject,
	})
	requireCode(t, "RefundPayment with no principal", err, codes.Unauthenticated)

	if len(repo.byID) != 0 {
		t.Errorf("repo has %d Payments after anonymous calls, want 0", len(repo.byID))
	}
}

// --- (d) the load-bearing case: the wire field is ignored ----------------

// TestEnforcedRPCs_WireActorClaimingOwnershipIsIgnored is the whole point of
// the sprint. Each request below carries actor_user_id = the real entitled
// actor — the exact bytes that succeeded before this ticket — while the
// verified principal is either a different user or absent entirely.
//
// If any of these succeeds, the migration is cosmetic: the wire field is still
// what authorizes the call, and an attacker who can read a Game's host_id off
// the public ListGames response can still mark any Registration on the
// platform paid without money changing hands.
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
			h, repo := newTestHandler("pay-1")

			offline := offlineReq()
			// nolint:staticcheck // SA1019: setting the deprecated field IS the test.
			offline.ActorUserId = hostSubject // the lie
			_, err := h.RecordOfflinePayment(tc.ctx, offline)
			requireCode(t, "RecordOfflinePayment", err, tc.want)

			online := onlineReq()
			// nolint:staticcheck // SA1019: setting the deprecated field IS the test.
			online.ActorUserId = hostSubject // the lie
			_, err = h.CreateOnlinePayment(tc.ctx, online)
			requireCode(t, "CreateOnlinePayment", err, tc.want)

			if len(repo.byID) != 0 {
				t.Errorf("repo has %d Payments — a rejected call wrote anyway", len(repo.byID))
			}
		})
	}
}

// TestRefundPayment_WireActorClaimingOwnershipIsIgnored is case (d) for the
// refund path, split into its own test because it is the one RPC that needs a
// real, already-paid Payment to reach its authorization check at all — a
// refund of a payment that does not exist answers NotFound long before anyone
// asks who the caller is, which would make a shared table case prove nothing.
func TestRefundPayment_WireActorClaimingOwnershipIsIgnored(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
		want codes.Code
	}{
		{"non-owner principal, wire field claims the owner", ctxAs(attackerSubject), codes.PermissionDenied},
		{"no principal, wire field claims the owner", anonymous(), codes.Unauthenticated},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// The seeded Payment id must be UUID-shaped: app.Service.GetPayment
			// applies a uuidShape boundary guard (T10.7, issue #97), so a
			// "pay-1"-style fixture id would answer NotFound before the
			// authorization check this test exists to reach. hostSubject
			// must resolve as fixtureGameID's real Host (T16.2) so the seed
			// call below still succeeds.
			h, repo := newPrincipalAuthzHandlerWithProcessor(hostSubject, fixturePaymentID)

			// Seed a real paid Payment, recorded by its rightful Host.
			paid, err := h.RecordOfflinePayment(ctxAs(hostSubject), offlineReq())
			if err != nil {
				t.Fatalf("seeding a paid Payment as the Host should succeed: %v", err)
			}
			paymentID := paid.GetPayment().GetId()

			_, err = h.RefundPayment(tc.ctx, &paymentsv1.RefundPaymentRequest{
				PaymentId:   paymentID,
				ActorUserId: hostSubject, // the lie
				GameHostId:  hostSubject,
			})
			requireCode(t, "RefundPayment", err, tc.want)

			if got := repo.byID[paymentID].Status; got != domain.StatusPaid {
				t.Errorf("Payment.Status = %v after a rejected refund, want still %v — a rejected call refunded anyway", got, domain.StatusPaid)
			}
		})
	}
}

// TestRecordOfflinePayment_RecordedByComesFromPrincipalNotWire is case (d) for
// the field this RPC *writes* rather than checks.
//
// Payment.RecordedByUserId is the audit trail for who marked money received.
// If it were taken from the wire, an attacker could record a payment under the
// Host's name — the authorization check would pass on the principal while the
// stored record blamed someone else, which is worse than either failing or
// succeeding honestly.
func TestRecordOfflinePayment_RecordedByComesFromPrincipalNotWire(t *testing.T) {
	// attackerSubject must resolve as fixtureGameID's real Host (T16.2) —
	// the ONLY way that fact can be established now that game_host_id below
	// is ignored — so this test's own "the attacker legitimately hosts this
	// Game" premise is expressed through newPrincipalAuthzHandler's
	// resolvers, not through the (still-set, still-forged-in-spirit,
	// now-inert) wire field.
	h, _ := newPrincipalAuthzHandler(attackerSubject, "pay-1")

	req := offlineReq()
	// DEPRECATED, IGNORED (T16.2): retained on the request only to prove it
	// really has no effect — the resolver above, not this field, is what
	// makes attackerSubject the real Host in this test.
	// nolint:staticcheck // SA1019: setting the deprecated field IS the test.
	req.GameHostId = attackerSubject // the attacker legitimately hosts this Game
	// nolint:staticcheck // SA1019: setting the deprecated field IS the test.
	req.ActorUserId = hostSubject // ...but claims to be someone else on the wire

	resp, err := h.RecordOfflinePayment(ctxAs(attackerSubject), req)
	if err != nil {
		t.Fatalf("RecordOfflinePayment by the real Host should succeed: %v", err)
	}

	if got := resp.GetPayment().GetRecordedByUserId(); got != attackerSubject {
		t.Errorf("Payment.RecordedByUserId = %q, want %q — the audit field was taken from the wire, so a caller can record a payment under someone else's name", got, attackerSubject)
	}
}
