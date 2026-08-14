package port

// IDGenerator produces new identifiers for aggregates. Mirrors
// internal/booking/port.IDGenerator and internal/facilities/port.IDGenerator
// — kept as its own per-context interface rather than shared, so no context
// depends on another's port package (CLAUDE.md rule 3). The single
// production implementation is internal/platform/idgen.UUID.
//
// Identity was the ONE context in this codebase without this port, and its
// absence was the security bug. Every other aggregate (Facility, Court,
// Booking, Payment, Game, Competition) server-mints its ID, which makes an
// ID collision on create unreachable; Identity instead took the ID from the
// caller, which let an anonymous caller permanently occupy any uuid they
// chose and lock the real owner out forever (HANDOFF.md's T10.2 bullet).
// T12.9 added this file to bring Identity back in line with the rest of the
// codebase: a User's ID is now minted here, and the caller's claim on their
// own identity is carried by the verified IdP subject instead.
type IDGenerator interface {
	NewID() string
}
