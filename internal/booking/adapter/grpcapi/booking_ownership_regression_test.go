package grpcapi_test

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	bookingv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/booking/v1"
)

// T55.1 (DECISION D1, ADR-0015 option (a); closes #144) — the handler-boundary
// half of the ownership fix.
//
// internal/booking/app/booking_ownership_test.go proves the rule; this file
// proves the wiring: that CreateBooking and CancelBooking now demand a
// verified caller at all, that the owner is taken from the token rather than
// from the request, and that a stranger's token does not cancel somebody
// else's booking.

// bookerCtx is the verified context of an ordinary player booking a court.
//
// Before D1 these two RPCs were public and the error-mapping and
// court-resolution tests in this package drove them with anonymous(). They
// now need a principal, so they use this. playerUser is deliberately the
// fixture actor: it holds no `club` role and owns no Facility, which is
// exactly the point — booking a court requires only that you are somebody,
// not that you are somebody special.
func bookerCtx() context.Context { return ctxAs(subjectOf(playerUser)) }

// The two RPCs must now refuse an unauthenticated caller outright. This is
// the direct regression test for the sentence ADR-0015 quotes from the old
// PublicMethods comment — "anyone holding a booking id can cancel that
// booking today".
func TestBookingOwnership_AnonymousIsRejected(t *testing.T) {
	h := newRecurringHandler()

	t.Run("CreateBooking", func(t *testing.T) {
		_, err := h.handler.CreateBooking(anonymous(), mapCreateBookingRequest())
		if status.Code(err) != codes.Unauthenticated {
			t.Fatalf("CreateBooking(anonymous) code = %v, want %v (err: %v)",
				status.Code(err), codes.Unauthenticated, err)
		}
	})

	t.Run("CancelBooking", func(t *testing.T) {
		created, err := h.handler.CreateBooking(bookerCtx(), mapCreateBookingRequest())
		if err != nil {
			t.Fatalf("seed CreateBooking: %v", err)
		}

		_, err = h.handler.CancelBooking(anonymous(), &bookingv1.CancelBookingRequest{
			BookingId: created.GetBooking().GetId(),
		})
		if status.Code(err) != codes.Unauthenticated {
			t.Fatalf("CancelBooking(anonymous) code = %v, want %v (err: %v)",
				status.Code(err), codes.Unauthenticated, err)
		}
	})
}

// The booking is owned by the token's subject, translated through
// UserIDBySubject (ADR-0014) — not by anything the request carried.
func TestBookingOwnership_OwnerComesFromTheToken(t *testing.T) {
	h := newRecurringHandler()

	created, err := h.handler.CreateBooking(bookerCtx(), mapCreateBookingRequest())
	if err != nil {
		t.Fatalf("CreateBooking: %v", err)
	}

	// The owner recorded is the User.ID the subject resolves to, proving the
	// translation actually happened: a handler that stored the raw subject
	// would fail this, and that is issue #152's exact failure mode.
	if got := created.GetBooking().GetOwnerUserId(); got != userID(playerUser) {
		t.Fatalf("OwnerUserId = %q, want %q", got, userID(playerUser))
	}
}

// The #144 scenario, end to end at the RPC boundary: a different verified
// user, holding the real booking id, is refused.
func TestBookingOwnership_StrangerCannotCancel(t *testing.T) {
	h := newRecurringHandler()

	created, err := h.handler.CreateBooking(bookerCtx(), mapCreateBookingRequest())
	if err != nil {
		t.Fatalf("CreateBooking: %v", err)
	}
	id := created.GetBooking().GetId()

	_, err = h.handler.CancelBooking(ctxAs(subjectOf(attackerUser)), &bookingv1.CancelBookingRequest{
		BookingId: id,
	})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("CancelBooking(stranger) code = %v, want %v (err: %v)",
			status.Code(err), codes.PermissionDenied, err)
	}

	// And the owner still can — proving the refusal above was about the
	// actor, not about the booking having been left in an uncancellable
	// state by the failed attempt.
	if _, err := h.handler.CancelBooking(bookerCtx(), &bookingv1.CancelBookingRequest{
		BookingId: id,
	}); err != nil {
		t.Fatalf("CancelBooking(owner) after refused attempt: %v", err)
	}
}
