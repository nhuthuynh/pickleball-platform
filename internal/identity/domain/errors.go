package domain

import "errors"

// Domain errors. Adapters must translate infrastructure failures into these
// — upper layers only ever see errors from this file. Mirrors
// internal/booking/domain/errors.go's sentinel-error pattern; deliberately
// not shared with that package (CLAUDE.md rule 2/3: internal/identity/domain
// must not import internal/booking/domain, or any other context's domain
// package).
var (
	ErrEmptyID = errors.New("identity: id is required")

	// ErrEmptySubject is returned by NewUser when the verified IdP subject
	// is empty (T12.9). Distinct from ErrEmptyID because the two identifiers
	// mean different things and fail for different reasons: an empty ID is
	// a server-side minting bug (port.IDGenerator returned nothing), while
	// an empty Subject means the request reached the domain without a
	// verified caller at all — which after T12.9 should be impossible,
	// since the handler rejects a principal-less CreateUser with
	// codes.Unauthenticated long before this point. Kept as a real check
	// rather than trusted, because a User with no subject is a row nobody
	// can authenticate as and a "" that could later match another "" in an
	// ownership comparison. Maps to codes.InvalidArgument (grpcapi.
	// toStatus) — the handler's own Unauthenticated guard is what a
	// token-less caller actually sees.
	ErrEmptySubject = errors.New("identity: verified subject is required")

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
	// type from any other rejection.
	//
	// T12.9: the "given a claimed actor_user_id, not authentication" caveat
	// that used to end this comment no longer applies to Identity. The
	// actor is now resolved from the verified auth.Principal at the grpcapi
	// boundary — the handler looks up the User owning the principal's
	// subject and passes that User's ID — and the wire actor_user_id field
	// is ignored. So this sentinel now means what it always claimed to
	// mean: a verified caller who is not this User. It is also what an
	// authenticated caller with no registered User of their own receives,
	// since they are provably not the target User (authenticated but not
	// authorized -> PermissionDenied, not Unauthenticated).
	ErrNotSelf = errors.New("identity: actor is not authorized to modify this user")

	// ErrUserAlreadyExists is returned when Create is called for a Subject
	// that is already registered (-> codes.AlreadyExists via
	// grpcapi.toStatus, rather than surfacing as a raw Postgres
	// unique-violation — CLAUDE.md rule 5).
	//
	// T12.9 CHANGED WHAT THIS MEANS, and the change is the security fix.
	// It used to mean "the caller-claimed actor_user_id you sent is already
	// taken", which was reachable by ANY caller against ANY uuid: that is
	// the identity-squatting denial-of-service HANDOFF.md's T10.2 bullet
	// disclosed, where a stranger's registration could permanently and
	// deliberately consume a real person's future identity. It now means
	// "you, the verified caller, are already registered", because the
	// uniqueness key is the IdP subject from the verified principal rather
	// than a value off the wire. A collision is therefore always a
	// SELF-collision, and the targeted-DoS reading of this error is gone —
	// not narrowed, gone: there is no longer any way to make this error
	// happen to someone else.
	//
	// A second CreateUser for an already-registered subject is REJECTED
	// with this sentinel rather than answered idempotently with the stored
	// User. The two calls are not necessarily the same request — the second
	// carries its own display name and level — so replaying the first would
	// answer "create me as X" with a user named Y and silently drop the
	// difference, which is a wrong answer wearing a success code.
	//
	// Backed by identity_users.subject's UNIQUE constraint
	// (db/migrations/0019_identity_subject.sql), which is the authoritative
	// half of CLAUDE.md rule 4's dual enforcement for this invariant: cross-
	// row uniqueness is not visible from inside a single aggregate, so the
	// domain expresses "a User always has a subject" and Postgres enforces
	// "at most one User per subject".
	ErrUserAlreadyExists = errors.New("identity: user already exists")

	// ErrRoleNotSelfAssignable is T10.2's PR-review fix (PR #106): returned
	// when a CreateUser request names any Role other than RolePlayer. An
	// unchecked Roles field would let a caller mint a brand-new,
	// permanently-persisted RolePlatformAdmin (or any other privileged
	// role) for themselves out of nothing.
	//
	// T12.9 made CreateUser authenticated, which does NOT retire this
	// check and the check was deliberately kept: self-registration is still
	// self-service, and holding a valid token proves who you are, not that
	// you may appoint yourself platform admin. Authentication narrowed who
	// can reach this path; it says nothing about what they may grant
	// themselves once here. This sentinel is the app layer's
	// (internal/identity/app.Service.CreateUser) enforcement that the
	// public path only ever accepts RolePlayer — every self-registering
	// caller needs exactly that role. A real mechanism for creating a User
	// with an elevated role is a different, auth-gated capability this
	// ticket does not build.
	ErrRoleNotSelfAssignable = errors.New("identity: role is not self-assignable via public registration")
)
