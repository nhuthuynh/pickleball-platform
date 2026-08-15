package port

import "context"

// GameLookup is Payments' inbound read port resolving a Social Play Game id
// to its HostID (T16.2, closing #168) — the second half of the
// Registration -> Game -> Host chain authorizeOfflineRecording's
// PayableTypeRegistration/PayableTypeNoShowFee branch needs.
// RegistrationLookup resolves a Registration's own id to its GameID; this
// port resolves that GameID the rest of the way to the Host subject the
// authorization check actually compares against.
//
// internal/payments/adapter/socialplay.GameLookup implements this against
// socialplayapp.Service.GetGame (T16.2) — the first single-Game read Social
// Play has ever exposed at the app layer, added by this same ticket.
type GameLookup interface {
	// HostIDForGame returns the HostID of the Game identified by gameID.
	//
	// An unknown, malformed, or empty gameID returns ("", nil), not an
	// error — the identical convention RegistrationLookup.
	// GameIDForRegistration documents, and for the same reason: this port's
	// only intended caller is an authorization pre-check, and an empty
	// HostID can never match a non-empty ActorUserID, so "no such Game"
	// safely resolves to not authorized without a distinct error path. In
	// particular, RegistrationLookup returning "" for an unresolved
	// Registration composes directly into this method's empty-gameID case
	// with no special-casing required on either side.
	HostIDForGame(ctx context.Context, gameID string) (string, error)
}
