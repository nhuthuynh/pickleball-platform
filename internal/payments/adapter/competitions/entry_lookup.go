package competitions

import (
	"context"
	"errors"
	"fmt"

	competitionsapp "github.com/nhuthuynh/white-label/internal/competitions/app"
	competitionsdomain "github.com/nhuthuynh/white-label/internal/competitions/domain"
)

// EntryLookup implements internal/payments/port.EntryLookup by calling the
// Competitions context's real app.Service.GetEntryByID (T16.2, closing
// #168) — the resolution-read counterpart to CompetitionAdminReader that
// T15.5 disclosed as missing: this is what lets authorizeOfflineRecording
// turn a CompetitionEntry's own PayableID into the CompetitionID
// CompetitionAdminReader needs, and the PlayerID the entrant-is-authorized
// branch already compared against a caller-supplied EntrantPlayerID.
type EntryLookup struct {
	competitionsSvc *competitionsapp.Service
}

// NewEntryLookup builds an EntryLookup against the real, shared Competitions
// app.Service instance, mirroring NewEntryUpdater's identical constructor
// shape.
func NewEntryLookup(competitionsSvc *competitionsapp.Service) *EntryLookup {
	return &EntryLookup{competitionsSvc: competitionsSvc}
}

// CompetitionIDAndPlayerIDForEntry calls the real competitionsapp.Service.
// GetEntryByID and returns its CompetitionID and PlayerID, translating
// errors at the boundary per CLAUDE.md rule 5 — %s, not %w — exactly as
// internal/payments/adapter/socialplay.RegistrationLookup.
// GameIDForRegistration does.
//
// competitionsdomain.ErrCompetitionEntryNotFound is deliberately swallowed
// into ("", "", nil), for the identical reason
// RegistrationLookup.GameIDForRegistration swallows
// socialplaydomain.ErrRegistrationNotFound: see port.EntryLookup's doc
// comment.
func (e *EntryLookup) CompetitionIDAndPlayerIDForEntry(ctx context.Context, entryID string) (competitionID, playerID string, err error) {
	entry, err := e.competitionsSvc.GetEntryByID(ctx, entryID)
	if err != nil {
		if errors.Is(err, competitionsdomain.ErrCompetitionEntryNotFound) {
			return "", "", nil
		}
		return "", "", fmt.Errorf("payments competitions adapter: get competition id and player id for entry %s: %s", entryID, err)
	}
	return entry.CompetitionID, entry.PlayerID, nil
}
