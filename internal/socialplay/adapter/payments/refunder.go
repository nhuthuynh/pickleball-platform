// Package payments adapts Social Play's port.PaymentRefunder onto the real
// Payments bounded context (T55.3, issue #124's refund half).
//
// It is the mirror image of internal/payments/adapter/socialplay, which
// adapts Payments' registration-status push onto Social Play. Keeping one
// adapter per direction, each owned by the calling context, is what stops
// either app layer importing the other's domain — see
// port.PaymentRefunder's doc comment on why the two directions must not be
// confused.
package payments

import (
	"context"
	"fmt"

	paymentsapp "github.com/nhuthuynh/white-label/internal/payments/app"
	paymentsdomain "github.com/nhuthuynh/white-label/internal/payments/domain"
)

// Refunder implements socialplay/port.PaymentRefunder over the real
// Payments app.Service.
//
// As with every other cross-context adapter here, no Payments error type
// crosses this boundary (%s, not %w — CLAUDE.md rule 5): Social Play must
// never see a paymentsdomain sentinel.
type Refunder struct {
	paymentsSvc *paymentsapp.Service
}

func NewRefunder(paymentsSvc *paymentsapp.Service) *Refunder {
	return &Refunder{paymentsSvc: paymentsSvc}
}

// RefundForRegistration reverses the payment for one Registration.
//
// The payable type is fixed here rather than accepted as a parameter, for
// the same reason port.CourtReservation fixes the Booking source: this port
// exists to serve Social Play's own use case, and a caller able to choose
// its own payable type could reverse a payment for something that is not a
// Registration at all.
//
// PayableTypeNoShowFee is deliberately NOT reached by this path even though
// it also targets a Registration's Game. A no-show fee is a separate charge
// from the Registration's own seat, and cancelling a Game does not obviously
// mean forgiving the fees of players who failed to turn up to earlier ones.
//
// Note this is NOT the same question as #130, which asked whether a no-show
// fee is refundable at all and was answered yes (T55.4 — a Game Admin who
// charged one in error can now reverse it deliberately). What stays
// unanswered is whether a Game CANCELLATION should forgive them
// automatically, which nobody has asked for and which this adapter
// therefore does not assume.
func (r *Refunder) RefundForRegistration(ctx context.Context, registrationID, actorUserID string) (bool, error) {
	refunded, err := r.paymentsSvc.RefundForPayable(ctx, paymentsapp.RefundForPayableInput{
		PayableType: paymentsdomain.PayableTypeRegistration,
		PayableID:   registrationID,
		ActorUserID: actorUserID,
	})
	if err != nil {
		return false, fmt.Errorf("socialplay payments adapter: refunding registration %s: %s", registrationID, err)
	}
	return refunded, nil
}
