package socialplay

import (
	"context"
	"errors"
	"fmt"

	socialplayapp "github.com/nhuthuynh/white-label/internal/socialplay/app"
	socialplaydomain "github.com/nhuthuynh/white-label/internal/socialplay/domain"
)

// GameLookup implements internal/payments/port.GameLookup by calling the
// Social Play context's real app.Service.GetGame (T16.2, closing #168) —
// the second half of the Registration -> Game -> Host chain, paired with
// RegistrationLookup.
type GameLookup struct {
	socialplaySvc *socialplayapp.Service
}

// NewGameLookup builds a GameLookup against the real, shared Social Play
// app.Service instance, mirroring NewRegistrationLookup's identical
// constructor shape.
func NewGameLookup(socialplaySvc *socialplayapp.Service) *GameLookup {
	return &GameLookup{socialplaySvc: socialplaySvc}
}

// HostIDForGame calls the real socialplayapp.Service.GetGame and returns its
// HostID, translating errors at the boundary per CLAUDE.md rule 5 — %s, not
// %w — exactly as RegistrationLookup.GameIDForRegistration does.
//
// socialplaydomain.ErrGameNotFound is deliberately swallowed into ("", nil),
// for the identical reason RegistrationLookup.GameIDForRegistration swallows
// socialplaydomain.ErrRegistrationNotFound: see port.GameLookup's doc
// comment. An empty gameID (RegistrationLookup's own safe answer for an
// unresolved Registration) reaches this method as a plain unknown-id lookup
// and takes the same path — no special-casing required.
func (g *GameLookup) HostIDForGame(ctx context.Context, gameID string) (string, error) {
	game, err := g.socialplaySvc.GetGame(ctx, gameID)
	if err != nil {
		if errors.Is(err, socialplaydomain.ErrGameNotFound) {
			return "", nil
		}
		return "", fmt.Errorf("payments socialplay adapter: get host id for game %s: %s", gameID, err)
	}
	return game.HostID, nil
}
