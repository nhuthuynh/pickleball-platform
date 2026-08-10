// Package competitions is the one place Payments code is allowed to import
// internal/competitions/* (CLAUDE.md rule 3 / T10.6, closes #96): it's the
// adapter that implements internal/competitions/port.
// CompetitionEntryPaymentUpdater against the real Competitions context's
// app.Service, mirroring internal/payments/adapter/socialplay's shape
// exactly (same "one context depends on another through a port, implemented
// by an adapter living in the depending context's tree" pattern), with the
// dependency arrow pointed the direction the context map requires here:
// Payments depends on Competitions, never the reverse.
// internal/competitions/domain and internal/competitions/app never import
// anything under internal/payments — this package, and internal/payments/app
// (which references internal/competitions/port/domain directly for the port
// type and its enum), are the only places in the Payments tree allowed to
// see Competitions' real types.
package competitions

import (
	"context"
	"errors"
	"fmt"

	competitionsapp "github.com/nhuthuynh/white-label/internal/competitions/app"
	competitionsdomain "github.com/nhuthuynh/white-label/internal/competitions/domain"
)

// EntryUpdater implements internal/competitions/port.
// CompetitionEntryPaymentUpdater by calling the Competitions context's real
// app.Service.MarkCompetitionEntryPaymentStatus. internal/payments/app.Service
// calls this (via the port interface, not this concrete type) after a
// competition_entry-payable Payment successfully transitions to paid
// (ConfirmOnlinePayment/RecordOfflinePayment, T10.6). Mirrors
// internal/payments/adapter/socialplay.RegistrationUpdater exactly.
type EntryUpdater struct {
	competitionsSvc *competitionsapp.Service
}

// NewEntryUpdater builds an EntryUpdater against the real, shared
// Competitions context app.Service instance (the same one cmd/server wires
// for Competitions' own gRPC handler) — there is no separate persistence
// handle here, this adapter only ever talks to Competitions through its own
// application layer, never straight to competitionsdb/Postgres, so
// Competitions' own invariants (e.g. MarkPaymentStatus's IsValid guard) are
// never bypassed.
func NewEntryUpdater(competitionsSvc *competitionsapp.Service) *EntryUpdater {
	return &EntryUpdater{competitionsSvc: competitionsSvc}
}

// UpdatePaymentStatus updates the CompetitionEntry identified by entryID to
// status, translating competitionsdomain.ErrCompetitionEntryNotFound and
// competitionsdomain.ErrInvalidPaymentStatus through unchanged (they are
// already Competitions' own stable sentinels — the exact contract
// port.CompetitionEntryPaymentUpdater's doc comment promises callers) and
// stripping the type off anything else (CLAUDE.md rule 5) so a caller on
// the Payments side can never accidentally errors.Is() against some other,
// unexpected competitionsdomain sentinel that leaked across the boundary by
// accident.
func (u *EntryUpdater) UpdatePaymentStatus(ctx context.Context, entryID string, status competitionsdomain.PaymentStatus) error {
	err := u.competitionsSvc.MarkCompetitionEntryPaymentStatus(ctx, entryID, status)
	if err == nil {
		return nil
	}
	if errors.Is(err, competitionsdomain.ErrCompetitionEntryNotFound) {
		return competitionsdomain.ErrCompetitionEntryNotFound
	}
	if errors.Is(err, competitionsdomain.ErrInvalidPaymentStatus) {
		return competitionsdomain.ErrInvalidPaymentStatus
	}
	return fmt.Errorf("payments competitions adapter: update payment status for entry %s: %s", entryID, err)
}
