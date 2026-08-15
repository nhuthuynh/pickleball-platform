package socialplay

import (
	"context"
	"errors"
	"fmt"

	socialplayapp "github.com/nhuthuynh/white-label/internal/socialplay/app"
	socialplaydomain "github.com/nhuthuynh/white-label/internal/socialplay/domain"
)

// RegistrationLookup implements internal/payments/port.RegistrationLookup by
// calling the Social Play context's real app.Service.GetRegistrationByID
// (T16.2, closing #168) — the resolution-read counterpart to
// GameAdminReader that T15.5 disclosed as missing: this is what lets
// authorizeOfflineRecording turn a Registration's own PayableID into the
// GameID GameAdminReader/GameLookup need.
type RegistrationLookup struct {
	socialplaySvc *socialplayapp.Service
}

// NewRegistrationLookup builds a RegistrationLookup against the real, shared
// Social Play app.Service instance — the same one cmd/server wires for
// Social Play's own gRPC handler, mirroring NewGameAdminReader's identical
// constructor shape.
func NewRegistrationLookup(socialplaySvc *socialplayapp.Service) *RegistrationLookup {
	return &RegistrationLookup{socialplaySvc: socialplaySvc}
}

// GameIDForRegistration calls the real socialplayapp.Service.
// GetRegistrationByID and returns its GameID, translating errors at the
// boundary per CLAUDE.md rule 5 — %s, not %w — exactly as
// GameAdminReader.ListGameAdmins does.
//
// socialplaydomain.ErrRegistrationNotFound is deliberately swallowed into
// ("", nil), not propagated: see port.RegistrationLookup's doc comment for
// why an unresolved Registration and a genuinely absent one share one safe
// answer here (an empty GameID, which can never authorize anyone) rather
// than the two genuinely different facts they are on a real response body.
func (r *RegistrationLookup) GameIDForRegistration(ctx context.Context, registrationID string) (string, error) {
	reg, err := r.socialplaySvc.GetRegistrationByID(ctx, registrationID)
	if err != nil {
		if errors.Is(err, socialplaydomain.ErrRegistrationNotFound) {
			return "", nil
		}
		return "", fmt.Errorf("payments socialplay adapter: get game id for registration %s: %s", registrationID, err)
	}
	return reg.GameID, nil
}
