package socialplay

import (
	"context"
	"errors"
	"fmt"

	paymentsdomain "github.com/nhuthuynh/white-label/internal/payments/domain"
	socialplayapp "github.com/nhuthuynh/white-label/internal/socialplay/app"
	socialplaydomain "github.com/nhuthuynh/white-label/internal/socialplay/domain"
)

// RegistrationAmountLookup implements payments/port.RegistrationAmountLookup over
// Social Play's real app.Service (T56.2, issue #297).
//
// It is the read-in counterpart to RegistrationLookup in this same package,
// with the same relationship: Payments depends on Social Play through a
// port, implemented by an adapter living in the depending context's tree.
type RegistrationAmountLookup struct {
	socialplaySvc *socialplayapp.Service
}

func NewRegistrationAmountLookup(socialplaySvc *socialplayapp.Service) *RegistrationAmountLookup {
	return &RegistrationAmountLookup{socialplaySvc: socialplaySvc}
}

// ExpectedAmountForRegistration returns what the Registration owes.
//
// # It returns the STORED figure, not a recomputation
//
// This reads `Registration.AmountOwed`, frozen when the player registered
// (T56.1), rather than re-deriving `EntryFee × (1 + GuestCount)` from the
// Game. That is the whole point of freezing it: a Host who raises the entry
// fee afterwards must not retroactively make a player's correct payment
// wrong. Re-deriving here would reintroduce exactly the drift the frozen
// column exists to prevent — and would do it invisibly, since both paths
// return a plausible number.
//
// # Error translation
//
// socialplaydomain.ErrRegistrationNotFound becomes Payments' own
// ErrPayableNotFound: no Social Play error type crosses this boundary
// (CLAUDE.md rule 5). Anything else is wrapped with %s rather than %w, so
// no socialplaydomain sentinel leaks by accident either.
func (l *RegistrationAmountLookup) ExpectedAmountForRegistration(ctx context.Context, registrationID string) (paymentsdomain.Money, error) {
	reg, err := l.socialplaySvc.GetRegistrationByID(ctx, registrationID)
	if err != nil {
		if errors.Is(err, socialplaydomain.ErrRegistrationNotFound) {
			return paymentsdomain.Money{}, paymentsdomain.ErrPayableNotFound
		}
		return paymentsdomain.Money{}, fmt.Errorf("payments socialplay adapter: resolving amount owed for registration %s: %s", registrationID, err)
	}

	// Money is rebuilt field by field rather than converted wholesale: the
	// two types are the same shape today and are deliberately allowed to
	// diverge (see socialplay/domain.Money's own doc comment on why it is a
	// context-local duplicate), so a struct conversion here would be a
	// coupling that compiles until the day one of them changes.
	return paymentsdomain.Money{
		Cents:    reg.AmountOwed.Cents,
		Currency: reg.AmountOwed.Currency,
	}, nil
}
