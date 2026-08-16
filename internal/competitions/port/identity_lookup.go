package port

import "context"

// IdentityLookup is Competitions' outbound port into the Identity/Users
// bounded context — the first one Competitions has ever had (T29.1, closing
// the Competitions third of #164, and Competitions' half of #237). Before
// this ticket internal/competitions/port held FacilityLookup,
// CourtReservation, ShareTokenGenerator, IDGenerator, and
// CompetitionAdminRepository, but nothing into Identity, so this context's
// own actor(ctx) funnel returned a raw IdP subject unresolved (ADR-0014
// §5a's explicit, deliberate deferral for this context).
//
// It follows internal/payments/port/identity_lookup.go exactly (T28.1, the
// closest and most recent of the three existing templates — read in full
// this ticket alongside internal/booking/port/identity_lookup.go, T13.2,
// and internal/facilities/port/identity_lookup.go, T13.3): a
// primitive-typed interface, declared by the *consuming* context and
// implemented once per context rather than shared, so
// internal/competitions/domain and internal/competitions/app never import
// internal/identity/domain or internal/identity/app. Only the adapter that
// implements it (internal/competitions/adapter/identity) sees Identity's
// real types.
//
// Like Booking's/Facilities'/Payments' IdentityLookup, this interface
// deliberately has no method that could return an identitydomain.User. A
// port that handed back the whole aggregate would invite app-layer code to
// start reading its fields — Roles, SelfReportedStartingLevel — and quietly
// acquire a dependency on another context's model. Competitions asks only
// for the one answer it needs, and gets nothing else.
//
// # The two identifier spaces (ADR-0014, extended to this context by ADR-0017)
//
// Every method here names, in its own doc comment, WHICH identifier space
// its parameter belongs to, because there are two and confusing them is
// issue #164's Competitions third:
//
//   - a **subject** is the identity provider's verified `sub` claim
//     (auth.Principal.Subject, e.g. `auth0|abc123`). It is an arbitrary
//     provider string and is never a uuid.
//   - a **User.ID** is the uuid this platform mints and owns. It is what
//     every actor column in every other conformant context references, and
//     — as of ADR-0017 and the migration this ticket ships
//     (`db/migrations/0025_competitions_identity_conformance.sql`) — what
//     competitions.host_id, competition_entries.player_id, and
//     competition_admins.user_id/assigned_by now hold too.
//
// ADR-0014's ruling: a subject is translated to a User.ID at the grpcapi
// boundary, and NOTHING below that boundary ever holds a subject. So the one
// method here takes a subject — UserIDBySubject, which performs exactly that
// translation — and any method added later that takes an actor must take a
// User.ID and say so. A method whose doc comment does not name its space is
// a bug in this file.
//
// This port is deliberately the same size as Facilities'/Payments', for the
// same reason: Competitions has no role-gated RPC that this port would need
// to answer. Every authorization decision in this context is object-level
// (Competition.EnsureHost/EnsureHostOrCompetitionAdmin compare the actor
// against the Competition's own HostID or its resolved admin set), so a
// role predicate here would be an unused method whose presence implied
// Competitions makes role decisions it does not make.
type IdentityLookup interface {
	// UserIDBySubject translates a verified IdP **subject** into the
	// corresponding **User.ID** (uuid). It is the only method on this port,
	// and the only place in the Competitions context where a subject is a
	// legitimate input (ADR-0014, ADR-0017).
	//
	// It exists so that a subject never reaches a uuid-typed column or a
	// uuid-keyed comparison. As of this ticket, Competitions persists the
	// actor as competitions.host_id / competition_entries.player_id /
	// competition_admins.user_id / competition_admins.assigned_by, all
	// declared `uuid REFERENCES identity_users (id)` (nullable — ADR-0017
	// Decision 3, restated for the NOT NULL starting point these four
	// columns had; see the migration's own header for why this ticket ships
	// them nullable rather than tightening to NOT NULL in the same pass) —
	// writing a subject into any of them would panic the Postgres adapter's
	// mustUUID() helper, the same failure mode ADR-0014 §4 walks through for
	// Booking's identical column-shape hazard.
	//
	// It returns the id as a plain string, never the User. That keeps the
	// restriction this port was designed around — no other context's
	// aggregate crosses into Competitions — while letting the one value
	// Competitions genuinely needs through. A uuid is not a model.
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
