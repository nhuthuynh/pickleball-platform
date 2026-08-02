// Package socialplay is the one place Payments code is allowed to import
// internal/socialplay/* (CLAUDE.md rule 3 /
// docs/process/t6-sprint-plan.md's architecture note on the payments/
// socialplay context boundary): it's the adapter that implements
// internal/socialplay/port.RegistrationPaymentUpdater against the real
// Social Play context's app.Service, mirroring
// internal/socialplay/adapter/booking's shape exactly (same "one context
// depends on another through a port, implemented by an adapter living in
// the depending context's tree" pattern) but with the dependency arrow
// pointed the direction the context map requires here: Payments depends on
// Social Play, never the reverse. internal/socialplay/domain and
// internal/socialplay/app never import anything under internal/payments —
// this package, and internal/payments/app (which references
// internal/socialplay/port/domain directly for the port type and its enum),
// are the only places in the Payments tree allowed to see Social Play's
// real types.
package socialplay

import (
	"context"
	"errors"
	"fmt"

	socialplayapp "github.com/nhuthuynh/white-label/internal/socialplay/app"
	socialplaydomain "github.com/nhuthuynh/white-label/internal/socialplay/domain"
)

// RegistrationUpdater implements internal/socialplay/port.
// RegistrationPaymentUpdater by calling the Social Play context's real
// app.Service.MarkRegistrationPaymentStatus. internal/payments/app.Service
// calls this (via the port interface, not this concrete type) after a
// registration-payable Payment successfully transitions to paid
// (ConfirmOnlinePayment/RecordOfflinePayment, T6.5).
type RegistrationUpdater struct {
	socialplaySvc *socialplayapp.Service
}

// NewRegistrationUpdater builds a RegistrationUpdater against the real,
// shared Social Play app.Service instance (the same one cmd/server wires
// for Social Play's own gRPC handler) — there is no separate persistence
// handle here, this adapter only ever talks to Social Play through its own
// application layer, never straight to socialplaydb/Postgres, so Social
// Play's own invariants (e.g. MarkPaymentStatus's IsValid guard) are never
// bypassed.
func NewRegistrationUpdater(socialplaySvc *socialplayapp.Service) *RegistrationUpdater {
	return &RegistrationUpdater{socialplaySvc: socialplaySvc}
}

// UpdatePaymentStatus updates the Registration identified by registrationID
// to status, translating socialplaydomain.ErrRegistrationNotFound and
// socialplaydomain.ErrInvalidPaymentStatus through unchanged (they are
// already Social Play's own stable sentinels — the exact contract
// port.RegistrationPaymentUpdater's doc comment promises callers) and
// stripping the type off anything else (CLAUDE.md rule 5) so a caller on
// the Payments side can never accidentally errors.Is() against some other,
// unexpected socialplaydomain sentinel that leaked across the boundary by
// accident.
func (u *RegistrationUpdater) UpdatePaymentStatus(ctx context.Context, registrationID string, status socialplaydomain.PaymentStatus) error {
	err := u.socialplaySvc.MarkRegistrationPaymentStatus(ctx, registrationID, status)
	if err == nil {
		return nil
	}
	if errors.Is(err, socialplaydomain.ErrRegistrationNotFound) {
		return socialplaydomain.ErrRegistrationNotFound
	}
	if errors.Is(err, socialplaydomain.ErrInvalidPaymentStatus) {
		return socialplaydomain.ErrInvalidPaymentStatus
	}
	return fmt.Errorf("payments socialplay adapter: update payment status for registration %s: %s", registrationID, err)
}
