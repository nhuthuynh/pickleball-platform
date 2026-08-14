package grpcapi

import (
	paymentsv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/payments/v1"
)

// AuthenticatedMethods lists the Payments RPCs that may only be called by a
// verified caller. cmd/server composes this with every other context's list
// into one auth.MethodSet (T12 sprint plan A11 Ruling 2: the knowledge of
// which of *this* context's RPCs are public belongs here, next to the handlers
// that break if it is wrong — not in a single list in main.go).
//
// The names come from the generated *_FullMethodName constants rather than
// hand-written strings, so a renamed RPC is a compile error instead of a
// policy entry that silently stops matching and silently stops enforcing.
//
// # Why each of these three requires a principal
//
// All three carry an actor_user_id that an authorization branch reads. Money
// is the sharpest case of the claimed-actor problem in this codebase: before
// this ticket, "only the Game's Host or an assigned Game Admin may record this
// payment" meant "only a caller willing to type the Host's user ID", and the
// Host's ID is readable off Social Play's public ListGames response.
//
//   - RecordOfflinePayment: marks someone else's Registration or Competition
//     entry paid without money changing hands. A claimed actor here is a free
//     seat at any Game on the platform.
//   - CreateOnlinePayment: creates a payment intent against a payable the
//     caller must be entitled to pay for.
//   - RefundPayment: moves real money back out (T12.3). Its authorization
//     branch is chosen from the *stored* payable type precisely so a caller
//     cannot pick which check judges them — that care is wasted if the actor
//     the chosen branch compares is itself caller-supplied.
func AuthenticatedMethods() []string {
	return []string{
		paymentsv1.PaymentsService_RecordOfflinePayment_FullMethodName,
		paymentsv1.PaymentsService_CreateOnlinePayment_FullMethodName,
		paymentsv1.PaymentsService_RefundPayment_FullMethodName,
	}
}

// PublicMethods lists the Payments RPCs that deliberately stay callable
// without a token.
//
// It exists so the split is *exhaustive and checkable* rather than implied by
// omission: authenticated_test.go asserts every method on the generated
// ServiceDesc appears in exactly one of these two lists, which turns "a new
// RPC was added and nobody decided whether it needs auth" from a silent
// default-to-public into a failing test.
//
// # Why this one stays public
//
//   - ConfirmOnlinePayment: public **not because that is right, but because
//     making it otherwise is out of this ticket's reach**, and leaving that
//     unsaid would be worse than saying it. It is the only RPC on this service
//     with no actor field at all: it takes a payment_id and captures the
//     intent. There is no ownership fact recorded anywhere for it to check —
//     a Payment stores its payable and its processor reference, not the human
//     who is entitled to confirm it — so there is no claimed actor to migrate
//     and no verified one to compare against. Giving it a check means adding
//     an owner to the Payment aggregate, which is a domain change; A11 Ruling
//     3 scopes this ticket to the handler boundary precisely so it does not
//     make domain changes in passing.
//
//     Adding it to AuthenticatedMethods() instead was considered and
//     rejected: requiring a bare token would look like authorization while
//     granting every authenticated user on the platform exactly the capability
//     an anonymous one has today — anyone holding a payment_id can capture
//     that intent. That is worse than an honest public listing, because it
//     reads as solved. Disclosed and tracked as a GitHub issue per the
//     sprint's A5 standing rule, the same way T12.7 handled Booking's
//     CreateBooking/CancelBooking.
func PublicMethods() []string {
	return []string{
		paymentsv1.PaymentsService_ConfirmOnlinePayment_FullMethodName,
	}
}
