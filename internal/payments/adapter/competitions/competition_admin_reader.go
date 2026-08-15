package competitions

import (
	"context"
	"errors"
	"fmt"

	competitionsapp "github.com/nhuthuynh/white-label/internal/competitions/app"
	competitionsdomain "github.com/nhuthuynh/white-label/internal/competitions/domain"
)

// CompetitionAdminReader implements internal/payments/port.
// CompetitionAdminReader by calling the Competitions context's real
// app.Service.ListCompetitionAdmins (T15.5, §A12 GAP C) — the Competitions
// mirror of internal/payments/adapter/socialplay.GameAdminReader, and the
// read-in counterpart to EntryUpdater's push-out, living in this same
// package for the identical reason EntryUpdater does.
//
// See internal/payments/port.CompetitionAdminReader's and GameAdminReader's
// doc comments before wiring this into internal/payments/app.Service: it is
// deliberately unwired as of T15.5, because no caller in Payments has a
// competitionID to pass it for the payable type that would need it.
type CompetitionAdminReader struct {
	competitionsSvc *competitionsapp.Service
}

// NewCompetitionAdminReader builds a CompetitionAdminReader against the
// real, shared Competitions app.Service instance, mirroring
// NewEntryUpdater's identical constructor shape.
func NewCompetitionAdminReader(competitionsSvc *competitionsapp.Service) *CompetitionAdminReader {
	return &CompetitionAdminReader{competitionsSvc: competitionsSvc}
}

// ListCompetitionAdmins calls the real
// competitionsapp.Service.ListCompetitionAdmins and flattens its result to
// bare subject strings, translating errors at the boundary per CLAUDE.md
// rule 5 — %s, not %w — so no Payments caller can errors.Is() against a
// competitionsdomain sentinel, mirroring GameAdminReader.ListGameAdmins
// exactly.
//
// competitionsdomain.ErrCompetitionNotFound is deliberately swallowed into
// an empty slice with a nil error, for the identical reason
// GameAdminReader.ListGameAdmins swallows socialplaydomain.ErrGameNotFound:
// see port.CompetitionAdminReader's doc comment.
func (r *CompetitionAdminReader) ListCompetitionAdmins(ctx context.Context, competitionID string) ([]string, error) {
	admins, err := r.competitionsSvc.ListCompetitionAdmins(ctx, competitionID)
	if err != nil {
		if errors.Is(err, competitionsdomain.ErrCompetitionNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("payments competitions adapter: list competition admins for competition %s: %s", competitionID, err)
	}

	ids := make([]string, 0, len(admins))
	for _, a := range admins {
		ids = append(ids, a.UserID)
	}
	return ids, nil
}
