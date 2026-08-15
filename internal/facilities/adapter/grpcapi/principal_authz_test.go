// T12.7 — the verified-principal half of this context's authorization
// regression suite. authz_regression_test.go (T7.7/T8.4) proved the *ownership
// rule*: a caller who is not the owner is rejected. This file proves the thing
// that rule was resting on is no longer a lie: **where the acting identity
// comes from**.
//
// Before this ticket the actor was a string the caller put in the request
// body. "Only the owner may add a Court" was therefore only ever "only a
// caller willing to type the owner's user ID may add a Court" — a check that
// rejects nobody who has read the owner_id off the public GetFacility
// response. These tests assert the four cases the T12.7 ticket requires, for
// every now-enforced RPC:
//
//	(a) a valid principal for the real owner            -> succeeds
//	(b) a valid principal for a different user          -> PermissionDenied
//	(c) no principal at all                             -> Unauthenticated
//	(d) a wire actor_user_id claiming to be the owner,
//	    with a non-owner principal or none at all       -> rejected
//
// (c) and (b) must stay distinct codes: "I do not know who you are" versus "I
// know who you are and you may not do this" (ADR-0013 §5). (d) is the one that
// would have been silently broken by a naive migration — a handler that read
// the principal but fell back to req.ActorUserId when there wasn't one would
// pass (a), (b) and (c) and still be exactly as exploitable as before.
package grpcapi_test

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/nhuthuynh/white-label/internal/facilities/adapter/grpcapi"
	"github.com/nhuthuynh/white-label/internal/platform/auth"

	facilitiesv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/facilities/v1"
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
	ownerSubject    = "auth0|owner-1"
	attackerSubject = "auth0|attacker-9"
)

// seedOwnedFacility creates a Facility owned by ownerSubject through the real
// CreateFacility handler. Note there is no OwnerId on the request: the owner
// is minted from the principal now, which is itself part of what is under
// test — see TestCreateFacility_OwnerComesFromPrincipalNotWire.
func seedOwnedFacility(t *testing.T, h *grpcapi.Handler) *facilitiesv1.Facility {
	t.Helper()
	resp, err := h.CreateFacility(ctxAs(ownerSubject), &facilitiesv1.CreateFacilityRequest{
		Name:    "Riverside Courts",
		Address: "123 Main St",
	})
	if err != nil {
		t.Fatalf("seeding a facility as its owner should succeed, got: %v", err)
	}
	return resp.GetFacility()
}

// requireCode asserts the exact gRPC code, and reports Internal separately —
// a 500-shaped answer to an authorization question is its own bug class (it
// means an error escaped toStatus), and the T7.7 tests this file joins already
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
	h, repo := newTestHandler()
	facility := seedOwnedFacility(t, h)
	// Consent is a precondition for AddCameraLink only; attest it first so
	// this test measures ownership, not consent.
	stored := repo.facilities[facility.GetId()]
	stored.CameraConsentAttested = true
	repo.facilities[facility.GetId()] = stored

	ctx := ctxAs(ownerSubject)

	if _, err := h.AddCourt(ctx, &facilitiesv1.AddCourtRequest{
		FacilityId: facility.GetId(), Name: "Court 1",
	}); err != nil {
		t.Errorf("AddCourt as the owner's principal should succeed, got: %v", err)
	}
	if _, err := h.AddCameraLink(ctx, &facilitiesv1.AddCameraLinkRequest{
		FacilityId: facility.GetId(), Url: "https://example.com/cam1.m3u8",
	}); err != nil {
		t.Errorf("AddCameraLink as the owner's principal should succeed, got: %v", err)
	}
	if _, err := h.AttestCameraConsent(ctx, &facilitiesv1.AttestCameraConsentRequest{
		FacilityId: facility.GetId(),
	}); err != nil {
		t.Errorf("AttestCameraConsent as the owner's principal should succeed, got: %v", err)
	}
}

func TestEnforcedRPCs_NonOwnerPrincipalIsPermissionDenied(t *testing.T) {
	h, repo := newTestHandler()
	facility := seedOwnedFacility(t, h)
	stored := repo.facilities[facility.GetId()]
	stored.CameraConsentAttested = true
	repo.facilities[facility.GetId()] = stored

	ctx := ctxAs(attackerSubject)

	_, err := h.AddCourt(ctx, &facilitiesv1.AddCourtRequest{FacilityId: facility.GetId(), Name: "Court 1"})
	requireCode(t, "AddCourt with a non-owner principal", err, codes.PermissionDenied)

	_, err = h.AddCameraLink(ctx, &facilitiesv1.AddCameraLinkRequest{
		FacilityId: facility.GetId(), Url: "https://example.com/evil.m3u8",
	})
	requireCode(t, "AddCameraLink with a non-owner principal", err, codes.PermissionDenied)

	_, err = h.AttestCameraConsent(ctx, &facilitiesv1.AttestCameraConsentRequest{FacilityId: facility.GetId()})
	requireCode(t, "AttestCameraConsent with a non-owner principal", err, codes.PermissionDenied)

	if len(repo.courts) != 0 {
		t.Errorf("repo.courts = %d after rejected calls, want 0 — a rejected call must have no side effect", len(repo.courts))
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

	// A facility must exist for the ownership check to be reachable at all,
	// so that this test proves the principal check runs *first* rather than
	// the call failing for lack of a fixture.
	facility := seedOwnedFacility(t, h)

	ctx := anonymous()

	_, err := h.CreateFacility(ctx, &facilitiesv1.CreateFacilityRequest{Name: "Anon Courts", Address: "1 Nowhere"})
	requireCode(t, "CreateFacility with no principal", err, codes.Unauthenticated)

	_, err = h.AddCourt(ctx, &facilitiesv1.AddCourtRequest{FacilityId: facility.GetId(), Name: "Court 1"})
	requireCode(t, "AddCourt with no principal", err, codes.Unauthenticated)

	_, err = h.AddCameraLink(ctx, &facilitiesv1.AddCameraLinkRequest{
		FacilityId: facility.GetId(), Url: "https://example.com/cam.m3u8",
	})
	requireCode(t, "AddCameraLink with no principal", err, codes.Unauthenticated)

	_, err = h.AttestCameraConsent(ctx, &facilitiesv1.AttestCameraConsentRequest{FacilityId: facility.GetId()})
	requireCode(t, "AttestCameraConsent with no principal", err, codes.Unauthenticated)
}

// --- (d) the load-bearing case: the wire field is ignored ----------------

// TestEnforcedRPCs_WireActorClaimingOwnershipIsIgnored is the whole point of
// the sprint. Each request below carries actor_user_id = the real owner — the
// exact bytes that succeeded before this ticket — while the verified principal
// is either a different user or absent entirely.
//
// If any of these succeeds, the migration is cosmetic: the wire field is still
// what authorizes the call, and an attacker who can read owner_id off the
// public GetFacility response still owns every Facility on the platform.
func TestEnforcedRPCs_WireActorClaimingOwnershipIsIgnored(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
		want codes.Code
	}{
		{
			// The principal says attacker, the body says owner. The body must
			// lose, and the answer must be "you may not", not "who are you".
			name: "non-owner principal, wire field claims the owner",
			ctx:  ctxAs(attackerSubject),
			want: codes.PermissionDenied,
		},
		{
			// No principal at all, body claims the owner. This is the precise
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
			facility := seedOwnedFacility(t, h)
			stored := repo.facilities[facility.GetId()]
			stored.CameraConsentAttested = true
			repo.facilities[facility.GetId()] = stored

			_, err := h.AddCourt(tc.ctx, &facilitiesv1.AddCourtRequest{
				FacilityId:  facility.GetId(),
				Name:        "Court 1",
				ActorUserId: ownerSubject, // the lie
			})
			requireCode(t, "AddCourt", err, tc.want)

			_, err = h.AddCameraLink(tc.ctx, &facilitiesv1.AddCameraLinkRequest{
				FacilityId:  facility.GetId(),
				Url:         "https://example.com/evil.m3u8",
				ActorUserId: ownerSubject, // the lie
			})
			requireCode(t, "AddCameraLink", err, tc.want)

			_, err = h.AttestCameraConsent(tc.ctx, &facilitiesv1.AttestCameraConsentRequest{
				FacilityId:  facility.GetId(),
				ActorUserId: ownerSubject, // the lie
			})
			requireCode(t, "AttestCameraConsent", err, tc.want)

			if len(repo.courts) != 0 {
				t.Errorf("repo.courts = %d — a rejected call wrote anyway", len(repo.courts))
			}
			if links := repo.facilities[facility.GetId()].CameraLinks; len(links) != 0 {
				t.Errorf("CameraLinks = %v — a rejected call wrote anyway", links)
			}
		})
	}
}

// TestCreateFacility_OwnerComesFromPrincipalNotWire is case (d) for the RPC
// that *writes* ownership rather than checking it.
//
// A caller sends owner_id = the victim. If that field were still honored, the
// attacker would have minted a Facility owned by someone else — the
// identity-squatting shape T12.9 closes for identity_users, one table over.
// The Facility must come back owned by the verified caller.
//
// T13.3 changed what "owned by the verified caller" is spelled as, and the
// change is the point rather than incidental. This assertion used to expect
// attackerSubject — the raw `sub` claim — which is a value
// facilities.owner_id (`uuid NOT NULL`) cannot hold; that it passed is
// precisely how #154 stayed invisible. It now expects attackerUserID, the
// resolved identity_users.id, because ADR-0014 translates the subject at the
// actor() funnel. The property under test is unchanged: the *caller* owns the
// Facility, and the wire field is ignored.
func TestCreateFacility_OwnerComesFromPrincipalNotWire(t *testing.T) {
	h, _ := newTestHandler()

	resp, err := h.CreateFacility(ctxAs(attackerSubject), &facilitiesv1.CreateFacilityRequest{
		OwnerId: ownerSubject, // the lie
		Name:    "Squatted Courts",
		Address: "1 Main St",
	})
	if err != nil {
		t.Fatalf("CreateFacility with a valid principal should succeed: %v", err)
	}

	got := resp.GetFacility().GetOwnerId()
	if got == ownerSubject || got == ownerUserID {
		t.Fatalf("Facility.OwnerId = %q — owner_id was taken from the wire, so a caller can create a Facility owned by someone else", got)
	}
	if got != attackerUserID {
		t.Errorf("Facility.OwnerId = %q, want %q (the verified caller's resolved User.ID)", got, attackerUserID)
	}
}
