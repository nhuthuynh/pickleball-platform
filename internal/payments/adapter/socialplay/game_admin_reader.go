package socialplay

import (
	"context"
	"errors"
	"fmt"

	socialplayapp "github.com/nhuthuynh/white-label/internal/socialplay/app"
	socialplaydomain "github.com/nhuthuynh/white-label/internal/socialplay/domain"
)

// GameAdminReader implements internal/payments/port.GameAdminReader by
// calling the Social Play context's real app.Service.ListGameAdmins (T15.5,
// §A12 GAP C) — the read-in counterpart to RegistrationUpdater's push-out,
// living in this same package because it is the identical "one context
// depends on another through a port, implemented by an adapter in the
// depending context's tree" relationship, reading instead of writing.
//
// See internal/payments/port.GameAdminReader's doc comment before wiring
// this into internal/payments/app.Service: it is deliberately unwired as of
// T15.5, because no caller in Payments has a gameID to pass it for the
// payable types that would need it.
type GameAdminReader struct {
	socialplaySvc *socialplayapp.Service
}

// NewGameAdminReader builds a GameAdminReader against the real, shared
// Social Play app.Service instance — the same one cmd/server wires for
// Social Play's own gRPC handler, mirroring NewRegistrationUpdater's
// identical constructor shape.
func NewGameAdminReader(socialplaySvc *socialplayapp.Service) *GameAdminReader {
	return &GameAdminReader{socialplaySvc: socialplaySvc}
}

// ListGameAdmins calls the real socialplayapp.Service.ListGameAdmins and
// flattens its result to bare subject strings, translating errors at the
// boundary per CLAUDE.md rule 5 — %s, not %w, exactly as
// internal/socialplay/adapter/booking/reservation.go's ReserveCourt does —
// so no Payments caller can errors.Is() against a socialplaydomain sentinel.
//
// socialplaydomain.ErrGameNotFound is deliberately swallowed into an empty
// slice with a nil error, not propagated: see port.GameAdminReader's doc
// comment for why "no such Game" and "this Game has no admins" share one
// safe answer here (not authorized) rather than the two genuinely different
// facts they are on a real roster-read response body.
func (r *GameAdminReader) ListGameAdmins(ctx context.Context, gameID string) ([]string, error) {
	admins, err := r.socialplaySvc.ListGameAdmins(ctx, gameID)
	if err != nil {
		if errors.Is(err, socialplaydomain.ErrGameNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("payments socialplay adapter: list game admins for game %s: %s", gameID, err)
	}

	ids := make([]string, 0, len(admins))
	for _, a := range admins {
		ids = append(ids, a.UserID)
	}
	return ids, nil
}
