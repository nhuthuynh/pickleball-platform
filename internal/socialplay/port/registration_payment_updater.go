package port

import (
	"context"

	"github.com/nhuthuynh/white-label/internal/socialplay/domain"
)

// RegistrationPaymentUpdater is Social Play's inbound port for the Payments
// context to push a Registration's PaymentStatus forward once its
// corresponding Payment transitions (T6.5,
// docs/process/t6-sprint-plan.md's architecture note). Defined here, in
// the context that is *depended on*: the context map
// (docs/agent-operating-handbook.md A1) states Payments depends on Social
// Play, never the reverse, so this interface — and the domain.PaymentStatus
// values it's expressed in terms of — lives in internal/socialplay/port,
// and only internal/payments/adapter/socialplay (the mirror image of
// internal/socialplay/adapter/booking) is allowed to implement it and see
// both contexts' real types. internal/socialplay/domain and
// internal/socialplay/app never import anything under internal/payments.
type RegistrationPaymentUpdater interface {
	// UpdatePaymentStatus sets the Registration identified by
	// registrationID to status. Implementations return Social Play's own
	// sentinel errors (domain.ErrRegistrationNotFound,
	// domain.ErrInvalidPaymentStatus) — never a Payments- or
	// infrastructure-specific error type; the caller
	// (internal/payments/app.Service) never needs to know this update
	// happened through Postgres, an in-memory fake, or anything else.
	UpdatePaymentStatus(ctx context.Context, registrationID string, status domain.PaymentStatus) error
}
