// Package payments adapts Competitions' port.EntryRefunder onto the real
// Payments bounded context (T55.3, issue #124's refund half, mirrored onto
// Competitions).
//
// Mirror of internal/socialplay/adapter/payments, and the reverse direction
// of internal/payments/adapter/competitions — one adapter per direction,
// each owned by the calling context, so neither app layer imports the
// other's domain.
package payments

import (
	"context"
	"fmt"

	paymentsapp "github.com/nhuthuynh/white-label/internal/payments/app"
	paymentsdomain "github.com/nhuthuynh/white-label/internal/payments/domain"
)

// Refunder implements competitions/port.EntryRefunder over the real Payments
// app.Service. No Payments error type crosses this boundary (%s, not %w —
// CLAUDE.md rule 5).
type Refunder struct {
	paymentsSvc *paymentsapp.Service
}

func NewRefunder(paymentsSvc *paymentsapp.Service) *Refunder {
	return &Refunder{paymentsSvc: paymentsSvc}
}

// RefundForEntry reverses the payment for one CompetitionEntry. The payable
// type is fixed here rather than accepted as a parameter, for the same
// reason port.CourtReservation fixes the Booking source: a caller able to
// choose its own payable type could reverse a payment for something that is
// not a CompetitionEntry at all.
func (r *Refunder) RefundForEntry(ctx context.Context, entryID, actorUserID string) (bool, error) {
	refunded, err := r.paymentsSvc.RefundForPayable(ctx, paymentsapp.RefundForPayableInput{
		PayableType: paymentsdomain.PayableTypeCompetitionEntry,
		PayableID:   entryID,
		ActorUserID: actorUserID,
	})
	if err != nil {
		return false, fmt.Errorf("competitions payments adapter: refunding entry %s: %s", entryID, err)
	}
	return refunded, nil
}
