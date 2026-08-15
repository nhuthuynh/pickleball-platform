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
// context's model. Booking asks only for the specific answers it needs, and
// gets nothing else.
//
// # The two identifier spaces (ADR-0014)
//
// Every method here names, in its own doc comment, WHICH identifier space its
// parameter belongs to, because there are two and confusing them is issue
// #146:
//
//   - a **subject** is the identity provider's verified `sub` claim
//     (auth.Principal.Subject, e.g. `auth0|abc123`). It is an arbitrary
//     provider string and is never a uuid.
//   - a **User.ID** is the uuid this platform mints and owns. It is what
//     every actor column in every other context references.
//
// ADR-0014's ruling: a subject is translated to a User.ID at the grpcapi
// boundary, and NOTHING below that boundary ever holds a subject. So exactly
// one method here takes a subject — UserIDBySubject, which performs that
// translation — and every other method takes a User.ID. A method whose doc
// comment does not say which space it takes is a bug in this file.
//
// UserIDBySubject was added in T13.2. Adding it was a real change to this
// port's contract rather than an addition to it: the paragraph above used to
// say this interface exposes no method returning a User *or its ID*, and
// #152 correctly flagged that removing that restriction was a design
// decision, not a patch. It is relaxed exactly far enough — a uuid string
// crosses the boundary, the aggregate still does not.
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
	// UserIDBySubject translates a verified IdP **subject** into the
	// corresponding **User.ID** (uuid), and is the one method on this port
	// that takes a subject (ADR-0014).
	//
	// It exists so that a subject never reaches a uuid-typed column or a
	// uuid-keyed comparison. Booking persists the actor as
	// recurring_hire_templates.requested_by_user_id, declared
	// `uuid NOT NULL REFERENCES identity_users (id)` — writing a subject
	// there panics the Postgres adapter's mustUUID() and violates the FK,
	// which is why issue #152 insists the fix is a translation and not the
	// deletion of the app layer's uuid guard.
	//
	// It returns the id as a plain string, never the User. That keeps the
	// restriction this port was designed around — no other context's
	// aggregate crosses into Booking — while letting the one value Booking
	// genuinely needs through. A uuid is not a model.
	//
	// Implementations return an error satisfying
	// errors.Is(err, domain.ErrUserNotFound) when the subject is registered
	// to no User, including when it is empty. Callers must not depend on any
	// other error type crossing this boundary for that case. That sentinel
	// maps to codes.PermissionDenied, not NotFound — see ADR-0014 §6 for why
	// an authenticated-but-unregistered caller is a permission answer rather
	// than a user-enumeration oracle.
	UserIDBySubject(ctx context.Context, subject string) (string, error)

	// EnsureClubRole returns nil when actorUserID — a **User.ID** uuid, NOT
	// a subject (ADR-0014); callers get one from UserIDBySubject above —
	// resolves to a real User that holds the `club` role.
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
