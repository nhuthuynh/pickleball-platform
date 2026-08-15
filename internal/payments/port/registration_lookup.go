package port

import "context"

// RegistrationLookup is Payments' inbound read port resolving a Social Play
// Registration id to the Game id it belongs to (T16.2, closing #168). This
// is the exact resolution step port.GameAdminReader's doc comment named as
// missing when T15.5 built the admin readers and could not wire them in:
// RecordOfflinePaymentInput/RefundPaymentInput carry a Registration's own
// PayableID, never its parent Game's id, and no context exported a read from
// one to the other until socialplayapp.Service.GetRegistrationByID (T16.2).
//
// internal/payments/adapter/socialplay.RegistrationLookup implements this
// against that real method — the read-in counterpart to GameAdminReader,
// living in the same package for the identical "one context depends on
// another through a port, implemented by an adapter in the depending
// context's tree" relationship.
type RegistrationLookup interface {
	// GameIDForRegistration returns the Game id the Registration identified
	// by registrationID belongs to.
	//
	// An unknown or malformed registrationID returns ("", nil), not an
	// error — the same deliberate widening GameAdminReader.ListGameAdmins
	// documents for an unknown gameID: this port's only intended caller is
	// an authorization pre-check (authorizeOfflineRecording), where "no such
	// Registration" and "the Registration exists but the lookup found
	// nothing to authorize against" both resolve to the same safe answer,
	// not authorized. A returned empty GameID is never itself treated as a
	// valid Game id by any caller of this port — see
	// internal/payments/adapter/socialplay.GameLookup's identical
	// empty-input convention, which composes with this one without either
	// side needing a special case for the other.
	GameIDForRegistration(ctx context.Context, registrationID string) (string, error)
}
