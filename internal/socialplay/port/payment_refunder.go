package port

import "context"

// PaymentRefunder is Social Play's outbound port into the Payments bounded
// context, for reversing what players paid when a Game they registered for
// is cancelled.
//
// It mirrors port.CourtReservation's shape — primitive types only, so
// internal/socialplay/domain and internal/socialplay/app never import
// internal/payments/domain or internal/payments/app. Only the adapter that
// implements it (internal/socialplay/adapter/payments) sees Payments' real
// types.
//
// Note the direction: Payments already reaches INTO Social Play through
// port.RegistrationPaymentUpdater, which is how a recorded payment updates a
// Registration's payment status. This port is the other direction, and the
// two must not be confused — RegistrationPaymentUpdater projects a payment
// fact onto a Registration, whereas this one asks Payments to actually
// reverse money.
//
// # Why this exists (issue #124's refund half)
//
// T16.3 made a cancelled Game bulk-cancel its Registrations, but nothing
// refunded them: RefundPayment existed and was simply never called, so
// players who had paid for a Game its Host cancelled were left out of
// pocket unless they chased it. The Product Owner decided on 2026-09-04
// that a host-initiated cancellation refunds them automatically, on the
// reasoning that the players did nothing wrong.
type PaymentRefunder interface {
	// RefundForRegistration reverses the payment made for registrationID,
	// if there is one and it is refundable, and reports whether a refund
	// actually happened.
	//
	// A registration with no payment, or one already refunded, is a
	// (false, nil) no-op rather than an error: a cancelled Game's roster is
	// a mix of paid, unpaid and already-refunded registrations, and a
	// cascade that errored on the last two would fail on almost every real
	// cancellation. That judgement lives in Payments, which owns the
	// concept of refundability — implementations here must not reimplement
	// it.
	//
	// actorUserID is the **User.ID** performing the cancellation. It is
	// checked by Payments' own authorization rules, which this port does
	// not relax: a Game's Host passes them for that Game's registrations,
	// which is exactly who CancelGame lets through.
	RefundForRegistration(ctx context.Context, registrationID, actorUserID string) (refunded bool, err error)
}
