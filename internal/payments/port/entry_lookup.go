package port

import "context"

// EntryLookup is Payments' inbound read port resolving a Competitions
// CompetitionEntry id to both the CompetitionID and PlayerID it carries
// (T16.2, closing #168) — the CompetitionEntry analogue of
// RegistrationLookup+GameLookup combined into one call, since
// competitionsapp.Service.GetEntryByID already returns both facts on a
// single domain.CompetitionEntry (unlike Social Play, which splits
// Registration -> Game and Game -> Host across two separate aggregates and
// therefore two separate lookups).
//
// internal/payments/adapter/competitions.EntryLookup implements this
// against that real method (T16.2) — the first exported single-entry read
// Competitions has offered outside its own package (GetEntryByID previously
// existed only as an unexported-from-Payments'-view internal call inside
// MarkCompetitionEntryPaymentStatus).
type EntryLookup interface {
	// CompetitionIDAndPlayerIDForEntry returns the CompetitionID and
	// PlayerID of the CompetitionEntry identified by entryID.
	//
	// An unknown or malformed entryID returns ("", "", nil), not an error —
	// the identical convention RegistrationLookup/GameLookup document, and
	// for the same reason: this port's only intended caller is an
	// authorization pre-check, where an empty CompetitionID/PlayerID can
	// never match a real CompetitionAdminReader lookup or a non-empty
	// ActorUserID, so "no such CompetitionEntry" safely resolves to not
	// authorized without a distinct error path.
	CompetitionIDAndPlayerIDForEntry(ctx context.Context, entryID string) (competitionID, playerID string, err error)
}
