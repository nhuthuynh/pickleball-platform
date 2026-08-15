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
// # Why each of these requires a principal
//
// Each carries an actor an authorization branch reads. Money is the sharpest
// case of the claimed-actor problem in this codebase: before T12.8, "only the
// Game's Host or an assigned Game Admin may record this payment" meant "only a
// caller willing to type the Host's user ID", and the Host's ID is readable off
// Social Play's public ListGames response.
//
//   - RecordOfflinePayment: marks someone else's Registration or Competition
//     entry paid without money changing hands. A claimed actor here is a free
//     seat at any Game on the platform.
//   - CreateOnlinePayment: creates a payment intent against a payable the
//     caller must be entitled to pay for. Its principal is also what the
//     Payment stores as RecordedByUserID (T13.7), which is what makes
//     ConfirmOnlinePayment's check below possible at all.
//   - RefundPayment: moves real money back out (T12.3). Its authorization
//     branch is chosen from the *stored* payable type precisely so a caller
//     cannot pick which check judges them — that care is wasted if the actor
//     the chosen branch compares is itself caller-supplied.
//   - ConfirmOnlinePayment: captures the funds (T13.7, closes issue #148).
//     Added here once there was something to compare a principal *against* —
//     see the note this replaced in PublicMethods(), which recorded the gap
//     honestly while it was open. Requiring a token is not on its own the fix
//     (that was the reason T12.8 rejected adding it then, and the reasoning
//     still stands): what makes it enforcement rather than theatre is
//     app.authorizeOnlineConfirmation comparing the principal against the
//     Payment's own RecordedByUserID.
func AuthenticatedMethods() []string {
	return []string{
		paymentsv1.PaymentsService_RecordOfflinePayment_FullMethodName,
		paymentsv1.PaymentsService_CreateOnlinePayment_FullMethodName,
		paymentsv1.PaymentsService_RefundPayment_FullMethodName,
		paymentsv1.PaymentsService_ConfirmOnlinePayment_FullMethodName,
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
// # Why this list is empty
//
// It is empty as of T13.7 (closes issue #148), and empty is a decision here,
// not an omission — that is exactly what authenticated_test.go's exhaustiveness
// check exists to keep true. Every RPC on PaymentsService moves or records
// money, and every one of them now has a verified actor and something to
// compare it against.
//
// The entry that used to live here was ConfirmOnlinePayment, listed public with
// a long note saying so was wrong but out of T12.8's reach: it took a
// payment_id, captured the intent, and had no ownership fact recorded anywhere
// to check, because CreateOnlinePayment stored an empty RecordedByUserID. T12.8
// also considered and rejected simply moving it into AuthenticatedMethods(),
// on the grounds that requiring a bare token would grant every authenticated
// user precisely the capability an anonymous one already had, while reading as
// solved. T13.7 closes it the way that note said it would have to be closed —
// by recording an owner at intent-creation time and comparing it at capture —
// so the RPC moves into AuthenticatedMethods() with a real object-level check
// behind it rather than a token requirement standing in for one.
//
// Keep this function rather than deleting it: authenticated_test.go's
// exhaustiveness check reads it (deleting it would make every future RPC
// unclassifiable except by adding it back), and a future public RPC — a Stripe
// webhook receiver, say, which issue #148's closing note raises as the shape
// that would make the client-driven capture question moot — needs somewhere to
// be declared deliberately rather than by omission.
func PublicMethods() []string {
	return nil
}
