package port

import "context"

// IdentityLookup is Facilities' outbound port into the Identity/Users bounded
// context — the first one Facilities has ever had (T13.3). Before this ticket
// internal/facilities/port contained exactly idgenerator.go and
// repository.go, and this context called no other context at all, so this is
// a new arrow in the context map rather than a reuse of an existing one.
//
// It follows the convention internal/booking/port/identity_lookup.go and
// internal/booking/port/facility_lookup.go established, which is this
// codebase's settled answer for a cross-context lookup: a primitive-typed
// interface, declared by the *consuming* context and implemented once per
// context rather than shared, so internal/facilities/domain and
// internal/facilities/app never import internal/identity/domain or
// internal/identity/app. Only the adapter that implements it
// (internal/facilities/adapter/identity) sees Identity's real types.
//
// Like FacilityLookup and Booking's IdentityLookup, this interface
// deliberately has no method that could return an identitydomain.User. A port
// that handed back the whole aggregate would invite app-layer code to start
// reading its fields — Roles, SelfReportedStartingLevel — and quietly acquire
// a dependency on another context's model. Facilities asks only for the one
// answer it needs, and gets nothing else.
//
// # The two identifier spaces (ADR-0014)
//
// Every method here names, in its own doc comment, WHICH identifier space its
// parameter belongs to, because there are two and confusing them is issue
// #154:
//
//   - a **subject** is the identity provider's verified `sub` claim
//     (auth.Principal.Subject, e.g. `auth0|abc123`). It is an arbitrary
//     provider string and is never a uuid.
//   - a **User.ID** is the uuid this platform mints and owns. It is what
//     every actor column in every other context references.
//
// ADR-0014's ruling: a subject is translated to a User.ID at the grpcapi
// boundary, and NOTHING below that boundary ever holds a subject. So the one
// method here takes a subject — UserIDBySubject, which performs exactly that
// translation — and any method added later that takes an actor must take a
// User.ID and say so. A method whose doc comment does not name its space is a
// bug in this file.
//
// This port is deliberately smaller than Booking's, which also carries
// EnsureClubRole. Facilities has no role-gated RPC: its authorization rule is
// object-level (domain.Facility.EnsureOwner compares the actor against the
// Facility's own OwnerID), not role-based, so a role predicate here would be
// an unused method whose presence implied Facilities makes role decisions it
// does not make. The port grows when a use case demands it.
type IdentityLookup interface {
	// UserIDBySubject translates a verified IdP **subject** into the
	// corresponding **User.ID** (uuid). It is the only method on this port,
	// and the only place in the Facilities context where a subject is a
	// legitimate input (ADR-0014).
	//
	// It exists so that a subject never reaches a uuid-typed column or a
	// uuid-keyed comparison. Facilities persists the actor as
	// facilities.owner_id, declared `uuid NOT NULL`
	// (db/migrations/0010_facilities.sql) — writing a subject there panics
	// the Postgres adapter's mustUUID(), which is issue #154 exactly, and
	// the panic is pinned Docker-free by
	// internal/facilities/adapter/postgres/owner_subject_panic_test.go.
	//
	// It returns the id as a plain string, never the User. That keeps the
	// restriction this port was designed around — no other context's
	// aggregate crosses into Facilities — while letting the one value
	// Facilities genuinely needs through. A uuid is not a model.
	//
	// Implementations return an error satisfying
	// errors.Is(err, domain.ErrUserNotFound) when the subject is registered
	// to no User, including when it is empty. Callers must not depend on any
	// other error type crossing this boundary for that case. That sentinel
	// maps to codes.PermissionDenied, not NotFound — see ADR-0014 §6 for why
	// an authenticated-but-unregistered caller is a permission answer rather
	// than a user-enumeration oracle.
	UserIDBySubject(ctx context.Context, subject string) (string, error)
}
