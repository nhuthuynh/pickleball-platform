package port

import "context"

// IdentityLookup is Booking's outbound port into the Identity/Users bounded
// context — the first one Booking has ever had (T11.5; before this ticket no
// Booking code resolved a user id anywhere, see the sprint plan's A8
// dependency check). It mirrors port.FacilityLookup's shape, which is this
// codebase's established convention for a cross-context lookup: a
// primitive-typed interface, implemented once per context rather than shared,
// so internal/booking/domain and internal/booking/app never import
// internal/identity/domain or internal/identity/app. Only the adapter that
// implements it (internal/booking/adapter/identity) sees Identity's real
// types.
//
// Like FacilityLookup, this interface deliberately has no method that could
// return an identitydomain.User. A port that handed back the whole aggregate
// would invite app-layer code to start reading its fields — Roles,
// SelfReportedStartingLevel — and quietly acquire a dependency on another
// context's model. Booking needs exactly one answer today, and asks for
// exactly that one.
//
// (The T11.5 ticket text names a `GetUser`-shaped method, citing a
// socialplay/port.IdentityLookup precedent from T10.4. No such port exists in
// this repository — Booking is genuinely the first context to call into
// Identity — so the shape here follows the precedent that does exist,
// port.FacilityLookup's EnsureFacilityOwner, rather than a cited one that
// does not. The behaviour the ticket specifies is unchanged and lives in the
// adapter: it resolves the actor through the real
// identityapp.Service.GetUser and checks the User's real Roles for `club`.)
type IdentityLookup interface {
	// EnsureClubRole returns nil when actorUserID resolves to a real User
	// that holds the `club` role.
	//
	// This is the server-side half of the creation-RPC checklist's role
	// self-assignment item (T10 retro finding 3, sprint plan A4): whether a
	// caller may request a recurring hire as a Club is a fact read out of
	// Identity, never a field the caller sends. There is deliberately no
	// method here that accepts a claimed role.
	//
	// The role check itself is not a rule Booking gets to define — the
	// adapter reads identitydomain.User.Roles, so "which roles a User holds"
	// has one definition, in the context that owns the concept.
	//
	// Implementations return an error satisfying
	// errors.Is(err, domain.ErrNotClub) when the actor resolves but holds no
	// `club` role, and errors.Is(err, domain.ErrUserNotFound) when
	// actorUserID does not resolve to a User at all. Callers must not depend
	// on any other error type crossing this boundary for those two cases.
	EnsureClubRole(ctx context.Context, actorUserID string) error
}
