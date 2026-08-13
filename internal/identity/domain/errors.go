package domain

import "errors"

// Domain errors. Adapters must translate infrastructure failures into these
// — upper layers only ever see errors from this file. Mirrors
// internal/booking/domain/errors.go's sentinel-error pattern; deliberately
// not shared with that package (CLAUDE.md rule 2/3: internal/identity/domain
// must not import internal/booking/domain, or any other context's domain
// package).
var (
	ErrEmptyID                          = errors.New("identity: id is required")
	ErrEmptyDisplayName                 = errors.New("identity: display name is required")
	ErrEmptyRoles                       = errors.New("identity: at least one role is required")
	ErrInvalidRole                      = errors.New("identity: invalid role")
	ErrInvalidSelfReportedStartingLevel = errors.New("identity: self-reported starting level must be between 1 and 5")

	// ErrUserNotFound is returned when a User lookup by ID has no match.
	// Added in T10.2 (postgres/grpc wiring) alongside the persistence
	// adapter, mirroring facilities.ErrFacilityNotFound — CLAUDE.md rule 5
	// requires adapters to translate infra errors (e.g. a Postgres
	// pgx.ErrNoRows) into a domain sentinel like this one, not let the raw
	// infra error cross into app/grpcapi. Also the answer the app layer's
	// malformed-ID boundary guard gives for a caller-supplied ID that never
	// reaches the repository at all (T10.2 item 4, mirrors
	// facilities.app's uuidShape guard) — a malformed ID and an
	// unknown-but-well-formed one must be indistinguishable to the caller.
	ErrUserNotFound = errors.New("identity: user not found")

	// ErrNotSelf is T10.2's object-level (BOLA) authorization sentinel:
	// returned by User.EnsureSelf when a caller-supplied actor_user_id does
	// not match the target User's own ID. Mirrors
	// internal/facilities/domain.ErrNotFacilityOwner (T7.7) and
	// internal/socialplay/domain.ErrNotRegistrationOwner (T5.2/T5.5): a
	// distinct sentinel (not a generic "unauthorized" string) so
	// grpcapi.toStatus can map it to codes.PermissionDenied (-> HTTP 403)
	// rather than a 500, and so a mismatched actor is distinguishable by
	// type from any other rejection. As with those two precedents, this
	// only proves the *object-level* check given a claimed actor_user_id —
	// it is not itself authentication; see HANDOFF.md's Auth cross-cutting
	// item.
	ErrNotSelf = errors.New("identity: actor is not authorized to modify this user")

	// ErrUserAlreadyExists is returned when Create is called with an ID
	// that already belongs to another User. Unlike every other aggregate in
	// this codebase (Facility, Court, Booking, Payment — all server-generate
	// their ID via a port.IDGenerator, so an ID collision on Create is
	// unreachable in practice), a User's ID is the caller-claimed
	// actor_user_id itself (see CreateUserRequest's doc comment in
	// proto/pickleball/identity/v1/identity.proto for why): Identity is the
	// bounded context that represents the concept of a caller's own claimed
	// identity, so there is no separate context to generate one on the
	// caller's behalf, and a second CreateUser call for the same
	// actor_user_id is a real, reachable case this sentinel exists to
	// answer honestly (-> codes.AlreadyExists via grpcapi.toStatus) rather
	// than surface as a raw Postgres unique-violation.
	ErrUserAlreadyExists = errors.New("identity: user already exists")

	// ErrRoleNotSelfAssignable is T10.2's PR-review fix (PR #106): returned
	// when a CreateUser request names any Role other than RolePlayer.
	// CreateUser is Identity's only unauthenticated, self-service entry
	// point, so unlike a mismatched actor_user_id on an *existing* object
	// (which is simply rejected, leaving no trace — the caveat every other
	// actor_user_id field in this codebase carries, see
	// HANDOFF.md's Cross-cutting section), an unchecked Roles field here
	// would let any anonymous caller mint a brand-new,
	// permanently-persisted RolePlatformAdmin (or any other privileged
	// role) for themselves out of nothing. This sentinel is the app layer's
	// (internal/identity/app.Service.CreateUser) enforcement that the
	// public path only ever accepts RolePlayer — every self-registering
	// caller needs exactly that role. A real mechanism for creating a User
	// with an elevated role is a different, auth-gated capability this
	// ticket does not build.
	ErrRoleNotSelfAssignable = errors.New("identity: role is not self-assignable via public registration")
)
