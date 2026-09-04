package grpcapi

import (
	bookingv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/booking/v1"
)

// AuthenticatedMethods lists the Booking RPCs that may only be called by a
// verified caller. cmd/server composes this with every other context's list
// into one auth.MethodSet (T12 sprint plan A11 Ruling 2).
//
// The names come from the generated *_FullMethodName constants, not
// hand-written strings, so a renamed RPC is a compile error rather than a
// policy entry that silently stops matching and silently stops enforcing.
//
// # Why each of these six requires a principal
//
//   - CreateDiscountRule: writes pricing for a Facility; guarded by an owner
//     check that compared the Facility's owner against a caller-supplied
//     string until this ticket.
//
//   - RequestRecurringHire: creates a standing claim on a Court in the actor's
//     own name. The template records requested_by_user_id, and every later
//     approve/reject/list decision keys off it — so letting the caller name
//     themselves let them file a request as somebody else.
//
//   - ApproveRecurringHire, RejectRecurringHire: Facility-owner decisions that
//     mint (or refuse) real Bookings.
//
//   - ListRecurringHireTemplatesForFacility: a read, but an owner-scoped one.
//     It returns other parties' pending requests to a Facility's owner, so it
//     is authenticated for the same reason the writes are — the actor is what
//     scopes the result set, not merely what is checked before it.
//
//   - ListRecurringHireTemplatesForActor: the Club's own "my requests" view.
//     The actor *is* the scope here: an unauthenticated caller passing someone
//     else's ID reads that Club's entire request history, including rejections.
//
//   - CreateBooking, CancelBooking: moved here from PublicMethods by
//     DECISION D1 (ADR-0015 option (a); closes #144). Until T55 these two
//     were public, and this comment said so in as many words — that being
//     public was "not because that is right, but because making them
//     otherwise is out of this ticket's reach", and that "anyone holding a
//     booking id can cancel that booking today". D1 answered the product
//     question that blocked the fix: every Booking is owned by a verified
//     user (domain.Booking.OwnerUserID), so there is now both a principal to
//     compare against and an owner to compare it to. The handler resolves
//     the owner from the token, never from the request.
//
//     Authenticating CreateBooking is the accepted cost of option (a): it
//     breaks the shipped T7.6 public quote-and-book flow, which must now
//     sign a player in before the confirm step. GetQuote and
//     ListCourtBookings deliberately stay public so the browse half of that
//     flow is unaffected.
func AuthenticatedMethods() []string {
	return []string{
		bookingv1.BookingService_CreateDiscountRule_FullMethodName,
		bookingv1.BookingService_RequestRecurringHire_FullMethodName,
		bookingv1.BookingService_ApproveRecurringHire_FullMethodName,
		bookingv1.BookingService_RejectRecurringHire_FullMethodName,
		bookingv1.BookingService_ListRecurringHireTemplatesForFacility_FullMethodName,
		bookingv1.BookingService_ListRecurringHireTemplatesForActor_FullMethodName,
		bookingv1.BookingService_CreateBooking_FullMethodName,
		bookingv1.BookingService_CancelBooking_FullMethodName,
	}
}

// PublicMethods lists the Booking RPCs that deliberately stay callable without
// a token, so the split is exhaustive and checkable rather than implied by
// omission — see authenticated_test.go, which fails if any RPC on the service
// is in neither list.
//
// # Why each of these five stays public
//
//   - GetQuote, ListCourtBookings: the price-and-availability reads a player
//     performs *before* deciding to sign up. The T12.7 ticket names both
//     explicitly; authenticating them would break a shipped flow.
//   - ListDiscountRulesForFacility: the public promotions a Facility advertises
//     to attract players. Same browse path, same reasoning.
func PublicMethods() []string {
	return []string{
		bookingv1.BookingService_GetQuote_FullMethodName,
		bookingv1.BookingService_ListCourtBookings_FullMethodName,
		bookingv1.BookingService_ListDiscountRulesForFacility_FullMethodName,
	}
}
