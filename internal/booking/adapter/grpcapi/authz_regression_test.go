// T12.7 — Booking's verified-principal authorization regression suite.
//
// discount_regression_test.go (T11.2) and recurring_hire_regression_test.go
// (T11.5/T11.6) already prove the *authorization rules*: a non-owner cannot
// write a Facility's discount rules, a non-club cannot request a standing
// slot, a non-owner cannot approve one. This file proves the thing those rules
// were resting on is no longer a lie: **where the acting identity comes from**.
//
// Until this ticket the actor was a string in the request body. "Only the
// Facility owner may approve a recurring hire" therefore only ever meant "only
// a caller willing to type the owner's user ID may approve a recurring hire" —
// a check that rejects nobody. These tests assert the four cases the T12.7
// ticket requires, for every now-enforced RPC:
//
//	(a) a valid principal for the real owner/actor       -> succeeds
//	(b) a valid principal for a different user           -> PermissionDenied
//	(c) no principal at all                              -> Unauthenticated
//	(d) a wire actor_user_id claiming to be the owner,
//	    with a non-owner principal or none at all        -> rejected
//
// (b) and (c) must stay distinct codes: "I know who you are and you may not do
// this" versus "I do not know who you are" (ADR-0013 section 5). (d) is the
// case that would have been silently broken by a naive migration — a handler
// that read the principal but fell back to req.ActorUserId when there was none
// would pass (a), (b) and (c) and remain exactly as exploitable as before.
package grpcapi_test

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"

	"github.com/nhuthuynh/white-label/internal/platform/auth"

	bookingv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/booking/v1"
)

// ctxAs builds the context the auth interceptors produce for a caller whose
// bearer token verified as subject. It goes through the real exported
// auth.ContextWithPrincipal rather than stuffing a context key directly, so
// these tests break if that contract changes.
//
// The subjects here are the UUID-shaped fixture user IDs the rest of this
// package already uses, because the fakes compare the actor against
// userID(ownerUser) and friends. A real IdP subject is not a UUID
// (`auth0|abc123`) — the identifier-space question that raises is Identity's
// to answer (T12.9 adds identity_users.subject) and is disclosed in this
// ticket's PR rather than papered over here.
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

// ownerCtx is the Facility owner's verified context, the caller for the
// discount-rule fixtures in discount_regression_test.go — those tests are
// about pricing rules and error mapping, and the owner is simply who performs
// them now that the actor is not a request field.
func ownerCtx() context.Context { return ctxAs(userID(ownerUser)) }

// requestAs files a recurring-hire request as the club, returning the created
// template. It is the fixture the approve/reject/list cases need.
func requestAs(t *testing.T, h *recurringHarness, subject string) *bookingv1.RecurringHireTemplate {
	t.Helper()
	resp, err := h.handler.RequestRecurringHire(ctxAs(subject), validRequest(4))
	if err != nil {
		t.Fatalf("RequestRecurringHire as %s: %v", subject, err)
	}
	return resp.GetTemplate()
}

// --- (a) the real actor's principal succeeds -----------------------------

func TestEnforcedRPCs_ValidPrincipalForTheRealActorSucceeds(t *testing.T) {
	t.Run("CreateDiscountRule as the facility owner", func(t *testing.T) {
		h, _ := newTestHandler(t)
		if _, err := h.CreateDiscountRule(ctxAs(userID(ownerUser)), validCreateRequest()); err != nil {
			t.Fatalf("the owner's principal should succeed: %v", err)
		}
	})

	t.Run("RequestRecurringHire as a club", func(t *testing.T) {
		h := newRecurringHandler()
		if tpl := requestAs(t, h, userID(clubUser)); tpl.GetRequestedByUserId() != userID(clubUser) {
			t.Errorf("RequestedByUserId = %q, want the principal's subject %q",
				tpl.GetRequestedByUserId(), userID(clubUser))
		}
	})

	t.Run("ApproveRecurringHire as the facility owner", func(t *testing.T) {
		h := newRecurringHandler()
		tpl := requestAs(t, h, userID(clubUser))
		if _, err := h.handler.ApproveRecurringHire(ctxAs(userID(ownerUser)),
			&bookingv1.ApproveRecurringHireRequest{TemplateId: tpl.GetId()}); err != nil {
			t.Fatalf("the owner's principal should succeed: %v", err)
		}
	})

	t.Run("RejectRecurringHire as the facility owner", func(t *testing.T) {
		h := newRecurringHandler()
		tpl := requestAs(t, h, userID(clubUser))
		if _, err := h.handler.RejectRecurringHire(ctxAs(userID(ownerUser)),
			&bookingv1.RejectRecurringHireRequest{TemplateId: tpl.GetId()}); err != nil {
			t.Fatalf("the owner's principal should succeed: %v", err)
		}
	})

	t.Run("ListRecurringHireTemplatesForFacility as the facility owner", func(t *testing.T) {
		h := newRecurringHandler()
		tpl := requestAs(t, h, userID(clubUser))
		resp, err := h.handler.ListRecurringHireTemplatesForFacility(ctxAs(userID(ownerUser)),
			&bookingv1.ListRecurringHireTemplatesForFacilityRequest{FacilityId: facilityID(1)})
		if err != nil {
			t.Fatalf("the owner's principal should succeed: %v", err)
		}
		if len(resp.GetTemplates()) != 1 || resp.GetTemplates()[0].GetId() != tpl.GetId() {
			t.Errorf("got %v, want exactly template %q", resp.GetTemplates(), tpl.GetId())
		}
	})

	t.Run("ListRecurringHireTemplatesForActor scopes to the principal", func(t *testing.T) {
		h := newRecurringHandler()
		tpl := requestAs(t, h, userID(clubUser))

		resp, err := h.handler.ListRecurringHireTemplatesForActor(ctxAs(userID(clubUser)),
			&bookingv1.ListRecurringHireTemplatesForActorRequest{})
		if err != nil {
			t.Fatalf("the club's own principal should succeed: %v", err)
		}
		if len(resp.GetTemplates()) != 1 || resp.GetTemplates()[0].GetId() != tpl.GetId() {
			t.Fatalf("got %v, want exactly the club's own template %q", resp.GetTemplates(), tpl.GetId())
		}

		// The scope really is the principal, not the request: another user's
		// principal sees an empty list for the same (empty) request body.
		other, err := h.handler.ListRecurringHireTemplatesForActor(ctxAs(userID(playerUser)),
			&bookingv1.ListRecurringHireTemplatesForActorRequest{})
		if err != nil {
			t.Fatalf("another user's principal should still get a clean empty read: %v", err)
		}
		if len(other.GetTemplates()) != 0 {
			t.Errorf("another user's principal saw %v, want none — the list is not scoped by the principal", other.GetTemplates())
		}
	})
}

// --- (b) a different user's principal is PermissionDenied ----------------

func TestEnforcedRPCs_DifferentUsersPrincipalIsPermissionDenied(t *testing.T) {
	t.Run("CreateDiscountRule", func(t *testing.T) {
		h, discounts := newTestHandler(t)
		_, err := h.CreateDiscountRule(ctxAs(userID(attackerUser)), validCreateRequest())
		requireCode(t, err, codes.PermissionDenied)
		if len(discounts.byFacility) != 0 {
			t.Error("a rejected CreateDiscountRule wrote a rule anyway")
		}
	})

	t.Run("RequestRecurringHire as a non-club user", func(t *testing.T) {
		// A verified caller who is not a Club: known to Identity, no `club`
		// role. Still PermissionDenied, not Unauthenticated — we know exactly
		// who they are.
		h := newRecurringHandler()
		_, err := h.handler.RequestRecurringHire(ctxAs(userID(playerUser)), validRequest(4))
		requireCode(t, err, codes.PermissionDenied)
	})

	t.Run("ApproveRecurringHire", func(t *testing.T) {
		h := newRecurringHandler()
		tpl := requestAs(t, h, userID(clubUser))
		_, err := h.handler.ApproveRecurringHire(ctxAs(userID(attackerUser)),
			&bookingv1.ApproveRecurringHireRequest{TemplateId: tpl.GetId()})
		requireCode(t, err, codes.PermissionDenied)
	})

	t.Run("RejectRecurringHire", func(t *testing.T) {
		h := newRecurringHandler()
		tpl := requestAs(t, h, userID(clubUser))
		_, err := h.handler.RejectRecurringHire(ctxAs(userID(attackerUser)),
			&bookingv1.RejectRecurringHireRequest{TemplateId: tpl.GetId()})
		requireCode(t, err, codes.PermissionDenied)
	})

	t.Run("ListRecurringHireTemplatesForFacility", func(t *testing.T) {
		h := newRecurringHandler()
		requestAs(t, h, userID(clubUser))
		h.templates.listCalls = 0
		_, err := h.handler.ListRecurringHireTemplatesForFacility(ctxAs(userID(attackerUser)),
			&bookingv1.ListRecurringHireTemplatesForFacilityRequest{FacilityId: facilityID(1)})
		requireCode(t, err, codes.PermissionDenied)
		if h.templates.listCalls != 0 {
			t.Errorf("a non-owner's list reached the repository (%d calls)", h.templates.listCalls)
		}
	})
}

// --- (c) no principal at all is Unauthenticated, NOT PermissionDenied ----

// TestEnforcedRPCs_NoPrincipalIsUnauthenticated pins the distinction ADR-0013
// section 5 establishes. Asserting merely "an error" would let a handler
// answering PermissionDenied pass — and PermissionDenied tells an anonymous
// caller "you are known and refused", which is both untrue and unactionable:
// the remedy is to authenticate, and that code does not say so.
//
// ListRecurringHireTemplatesForActor is the sharpest one here. Its actor is
// its entire scope, so before this ticket an anonymous caller passing a Club's
// ID read that Club's whole request history, rejections included.
func TestEnforcedRPCs_NoPrincipalIsUnauthenticated(t *testing.T) {
	ctx := anonymous()

	t.Run("CreateDiscountRule", func(t *testing.T) {
		h, _ := newTestHandler(t)
		_, err := h.CreateDiscountRule(ctx, validCreateRequest())
		requireCode(t, err, codes.Unauthenticated)
	})

	t.Run("RequestRecurringHire", func(t *testing.T) {
		h := newRecurringHandler()
		_, err := h.handler.RequestRecurringHire(ctx, validRequest(4))
		requireCode(t, err, codes.Unauthenticated)
		if h.templates.createCalls != 0 {
			t.Error("an unauthenticated RequestRecurringHire reached the repository")
		}
	})

	t.Run("ApproveRecurringHire", func(t *testing.T) {
		h := newRecurringHandler()
		tpl := requestAs(t, h, userID(clubUser))
		_, err := h.handler.ApproveRecurringHire(ctx,
			&bookingv1.ApproveRecurringHireRequest{TemplateId: tpl.GetId()})
		requireCode(t, err, codes.Unauthenticated)
	})

	t.Run("RejectRecurringHire", func(t *testing.T) {
		h := newRecurringHandler()
		tpl := requestAs(t, h, userID(clubUser))
		_, err := h.handler.RejectRecurringHire(ctx,
			&bookingv1.RejectRecurringHireRequest{TemplateId: tpl.GetId()})
		requireCode(t, err, codes.Unauthenticated)
	})

	t.Run("ListRecurringHireTemplatesForFacility", func(t *testing.T) {
		h := newRecurringHandler()
		_, err := h.handler.ListRecurringHireTemplatesForFacility(ctx,
			&bookingv1.ListRecurringHireTemplatesForFacilityRequest{FacilityId: facilityID(1)})
		requireCode(t, err, codes.Unauthenticated)
	})

	t.Run("ListRecurringHireTemplatesForActor", func(t *testing.T) {
		h := newRecurringHandler()
		_, err := h.handler.ListRecurringHireTemplatesForActor(ctx,
			&bookingv1.ListRecurringHireTemplatesForActorRequest{})
		requireCode(t, err, codes.Unauthenticated)
	})
}

// --- (d) the load-bearing case: the wire field is ignored ----------------

// TestEnforcedRPCs_WireActorClaimingTheOwnerIsIgnored is the whole point of
// the sprint. Every request below carries actor_user_id = the legitimate actor
// — the exact bytes that succeeded before this ticket — while the verified
// principal is either a different user or absent entirely.
//
// If any of these succeeds, the migration is cosmetic: the wire field is still
// what authorizes the call, and anyone who can learn a facility owner's or a
// club's user ID still acts as them.
func TestEnforcedRPCs_WireActorClaimingTheOwnerIsIgnored(t *testing.T) {
	tests := []struct {
		name string
		// ctx is the caller's verified identity (or lack of one); claimed is
		// what the request body asserts about them.
		ctx  func() context.Context
		want codes.Code
	}{
		{
			// The principal says attacker, the body says owner. The body must
			// lose, and the answer is "you may not", not "who are you".
			name: "non-owner principal, wire field claims the legitimate actor",
			ctx:  func() context.Context { return ctxAs(userID(attackerUser)) },
			want: codes.PermissionDenied,
		},
		{
			// No principal at all, body claims the legitimate actor. This is
			// precisely the fallback the ticket forbids: a handler that
			// trusted the wire field when there was no principal returns OK
			// here.
			name: "no principal, wire field claims the legitimate actor",
			ctx:  anonymous,
			want: codes.Unauthenticated,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("CreateDiscountRule", func(t *testing.T) {
				h, discounts := newTestHandler(t)
				req := validCreateRequest()
				//nolint:staticcheck // SA1019: setting the deprecated field is
				// exactly what this test exists to prove the server ignores.
				req.ActorUserId = userID(ownerUser) // the lie
				_, err := h.CreateDiscountRule(tc.ctx(), req)
				requireCode(t, err, tc.want)
				if len(discounts.byFacility) != 0 {
					t.Error("the claimed actor_user_id was honored — a discount rule was written")
				}
			})

			t.Run("RequestRecurringHire", func(t *testing.T) {
				h := newRecurringHandler()
				req := validRequest(4)
				//nolint:staticcheck // SA1019: setting the deprecated field is
				// exactly what this test exists to prove the server ignores.
				req.ActorUserId = userID(clubUser) // the lie
				_, err := h.handler.RequestRecurringHire(tc.ctx(), req)
				requireCode(t, err, tc.want)
				if h.templates.createCalls != 0 {
					t.Error("the claimed actor_user_id was honored — a template was created")
				}
			})

			t.Run("ApproveRecurringHire", func(t *testing.T) {
				h := newRecurringHandler()
				tpl := requestAs(t, h, userID(clubUser))
				h.templates.updateCalls = 0
				_, err := h.handler.ApproveRecurringHire(tc.ctx(), &bookingv1.ApproveRecurringHireRequest{
					TemplateId:  tpl.GetId(),
					ActorUserId: userID(ownerUser), // the lie
				})
				requireCode(t, err, tc.want)
				if h.templates.updateCalls != 0 {
					t.Error("the claimed actor_user_id was honored — the template was approved")
				}
			})

			t.Run("RejectRecurringHire", func(t *testing.T) {
				h := newRecurringHandler()
				tpl := requestAs(t, h, userID(clubUser))
				h.templates.updateCalls = 0
				_, err := h.handler.RejectRecurringHire(tc.ctx(), &bookingv1.RejectRecurringHireRequest{
					TemplateId:  tpl.GetId(),
					ActorUserId: userID(ownerUser), // the lie
				})
				requireCode(t, err, tc.want)
				if h.templates.updateCalls != 0 {
					t.Error("the claimed actor_user_id was honored — the template was rejected")
				}
			})

			t.Run("ListRecurringHireTemplatesForFacility", func(t *testing.T) {
				h := newRecurringHandler()
				requestAs(t, h, userID(clubUser))
				h.templates.listCalls = 0
				_, err := h.handler.ListRecurringHireTemplatesForFacility(tc.ctx(),
					&bookingv1.ListRecurringHireTemplatesForFacilityRequest{
						FacilityId:  facilityID(1),
						ActorUserId: userID(ownerUser), // the lie
					})
				requireCode(t, err, tc.want)
				if h.templates.listCalls != 0 {
					t.Error("the claimed actor_user_id was honored — the facility queue was read")
				}
			})

			t.Run("ListRecurringHireTemplatesForActor", func(t *testing.T) {
				// The sharpest read of the six: the claimed actor IS the whole
				// scope, so honoring the wire field hands over another Club's
				// entire request history.
				h := newRecurringHandler()
				requestAs(t, h, userID(clubUser))
				h.templates.listCalls = 0

				resp, err := h.handler.ListRecurringHireTemplatesForActor(tc.ctx(),
					&bookingv1.ListRecurringHireTemplatesForActorRequest{
						ActorUserId: userID(clubUser), // the lie
					})

				if tc.want == codes.Unauthenticated {
					requireCode(t, err, codes.Unauthenticated)
					if h.templates.listCalls != 0 {
						t.Error("the claimed actor_user_id was honored — another actor's templates were read")
					}
					return
				}

				// With a *valid* principal for a different user this read is
				// legitimately allowed — it is scoped, not owner-checked — so
				// the proof here is that the attacker sees their own (empty)
				// list rather than the Club's, however loudly the body claims
				// to be the Club.
				if err != nil {
					t.Fatalf("a scoped self-read should succeed for any verified caller: %v", err)
				}
				if len(resp.GetTemplates()) != 0 {
					t.Errorf("got %v, want none — the claimed actor_user_id leaked another actor's templates",
						resp.GetTemplates())
				}
			})
		})
	}
}
