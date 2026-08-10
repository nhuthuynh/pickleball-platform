package port

import (
	"context"

	"github.com/nhuthuynh/white-label/internal/competitions/domain"
)

// CompetitionEntryPaymentUpdater is Competitions' inbound port for the
// Payments context to push a CompetitionEntry's PaymentStatus forward once
// its corresponding Payment transitions (T10.6, closes #96). Mirrors
// internal/socialplay/port.RegistrationPaymentUpdater exactly, including its
// placement: defined here, in the context that is *depended on* — the
// context map (docs/agent-operating-handbook.md A1) states Payments depends
// on Competitions, never the reverse, so this interface — and the
// domain.PaymentStatus values it's expressed in terms of — lives in
// internal/competitions/port, and only internal/payments/adapter/competitions
// (the mirror image of internal/payments/adapter/socialplay) is allowed to
// implement it and see both contexts' real types. internal/competitions/domain
// and internal/competitions/app never import anything under internal/payments.
type CompetitionEntryPaymentUpdater interface {
	// UpdatePaymentStatus sets the CompetitionEntry identified by entryID to
	// status. Implementations return Competitions' own sentinel errors
	// (domain.ErrCompetitionEntryNotFound, domain.ErrInvalidPaymentStatus) —
	// never a Payments- or infrastructure-specific error type; the caller
	// (internal/payments/app.Service) never needs to know this update
	// happened through Postgres, an in-memory fake, or anything else.
	UpdatePaymentStatus(ctx context.Context, entryID string, status domain.PaymentStatus) error
}
