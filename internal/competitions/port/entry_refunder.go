package port

import "context"

// EntryRefunder is Competitions' outbound port into the Payments bounded
// context, for reversing what entrants paid when a Competition they entered
// is cancelled.
//
// It is the exact mirror of socialplay/port.PaymentRefunder — see that
// interface's doc comment for the full reasoning, which applies here
// unchanged with CompetitionEntry substituted for Registration. The two
// exist separately rather than as one shared interface for the same reason
// each context has its own CourtReservation and FacilityLookup: a port
// belongs to the context that calls it, expressed in that context's own
// terms, so neither app layer imports the other's domain.
//
// Note the direction, as with Social Play's: Payments already reaches INTO
// Competitions through port.CompetitionEntryPaymentUpdater, which projects a
// payment fact onto an entry. This port is the other direction, and asks
// Payments to actually reverse money.
type EntryRefunder interface {
	// RefundForEntry reverses the payment made for entryID, if there is one
	// and it is refundable, and reports whether a refund actually happened.
	//
	// An entry with no payment, or one already refunded, is a (false, nil)
	// no-op rather than an error — refundability is Payments' concept and
	// implementations here must not reimplement it. See
	// socialplay/port.PaymentRefunder.RefundForRegistration for why a
	// cascade that errored on those cases would fail on almost every real
	// cancellation.
	//
	// actorUserID is the **User.ID** performing the cancellation, checked by
	// Payments' own authorization rules, which this port does not relax.
	RefundForEntry(ctx context.Context, entryID, actorUserID string) (refunded bool, err error)
}
