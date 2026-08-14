// Package app is the Identity/Users context's application layer: it
// orchestrates the domain and the repository port, but holds no business
// rules itself — those live in internal/identity/domain. Mirrors
// internal/facilities/app's shape (CLAUDE.md: "mirror the booking context
// exactly").
package app

import (
	"context"
	"regexp"

	"github.com/nhuthuynh/white-label/internal/identity/domain"
	"github.com/nhuthuynh/white-label/internal/identity/port"
)

// uuidShape matches the canonical 8-4-4-4-12 hex form
// internal/platform/idgen mints for every aggregate's ID in this codebase
// (Facility, Court, Booking, Payment, Competition, ... and, since T12.9,
// User too). It must be well-formed enough for the `uuid` column
// identity_users.id is stored as.
//
// Since T12.9 this guard is no longer needed on the CREATE path — that ID
// is server-minted now, so it is well-formed by construction — but it is
// still required on the read/update paths, whose `id` still arrives off the
// wire as a lookup key.
//
// Boundary guard for caller-supplied IDs: the Postgres adapter's mustUUID
// panics on anything pgtype.UUID.Scan can't parse, and grpc installs no
// recover() of its own, so an unvalidated ID off the wire could take the
// whole process down. Deliberately narrower than github.com/google/uuid's
// Validate, which accepts braced and `urn:uuid:` forms that pgtype rejects
// — a guard wider than the thing it protects is not a guard. Kept
// context-local rather than shared, matching how this repo keeps each
// context's own not-found sentinel local. The canonical write-up lives on
// internal/competitions/app's copy (reused here per T10.2's own instruction
// not to re-derive it — PR #89's postmortem, docs/LESSONS.md).
var uuidShape = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// Service is the Identity/Users context's application layer.
type Service struct {
	repo port.Repository
	ids  port.IDGenerator
}

// NewService wires the repository and the ID generator. The ids parameter
// is T12.9's addition — Identity was the only context in this codebase
// without one, and that absence was the identity-squatting bug (see
// port.IDGenerator's doc comment).
func NewService(repo port.Repository, ids port.IDGenerator) *Service {
	return &Service{repo: repo, ids: ids}
}

// CreateUserInput is CreateUser's use-case input.
//
// T12.9 REMOVED THIS STRUCT'S ActorUserID FIELD, and its removal is the
// security fix. It used to become the new User's own permanent primary key,
// taken verbatim from the request: an anonymous caller could name any uuid
// — including one a real-auth integration would later mint for a real
// person — and permanently occupy that identity, so the real owner's later
// registration failed with domain.ErrUserAlreadyExists forever. HANDOFF.md's
// T10.2 bullet recorded that as "a persistent, targeted denial-of-service,
// not a rejected mutation", with the closure condition "must close the
// moment real auth exists". It exists (T12.2), so the field is gone rather
// than validated more carefully — there is no longer any caller-supplied
// value that can become a User's identity.
//
// Subject replaces it, and the two are not the same shape of thing. Subject
// is the VERIFIED IdP `sub` claim, resolved from auth.Principal at the
// grpcapi boundary and never read from a request message; the ID is minted
// here (s.ids.NewID()) exactly like every other aggregate in this codebase.
// Note this package does not import internal/platform/auth to obtain it —
// the handler translates the Principal into this plain string, which is
// what keeps the app and domain layers auth-free (A11 Ruling 3).
//
// A second CreateUser for an already-registered Subject is rejected with
// domain.ErrUserAlreadyExists rather than replayed idempotently; see that
// sentinel's doc comment for the reasoning. Because the subject is verified,
// such a collision is always a self-collision.
type CreateUserInput struct {
	Subject                   string
	DisplayName               string
	Roles                     []domain.Role
	SelfReportedStartingLevel domain.SelfReportedStartingLevel
}

// CreateUser validates and persists a new User, minting its ID server-side
// and keying the row to the caller's verified Subject.
//
// The uuidShape guard the pre-T12.9 version opened with is gone because the
// value it guarded is gone: the ID no longer arrives off the wire, so it is
// well-formed by construction and the Postgres adapter's mustUUID cannot be
// reached with caller input on this path. The read/update paths keep their
// guards, whose `id` is still a caller-supplied lookup key.
func (s *Service) CreateUser(ctx context.Context, in CreateUserInput) (domain.User, error) {
	// PR #106 review fix (T10.2), deliberately KEPT now that this path is
	// authenticated: an unchecked Roles field would let a caller mint
	// themselves a brand-new, permanently-persisted RolePlatformAdmin (or
	// any other privileged role) out of nothing. Holding a valid token
	// proves who you are, not that you may appoint yourself an
	// administrator, so self-registration still only ever accepts
	// RolePlayer — every self-registering caller needs exactly that role. A
	// real mechanism for creating a User with an elevated role (Host/Game
	// Admin/Facility Owner/Club/Platform Admin) remains a different,
	// separately-authorized capability neither T10.2 nor T12.9 builds.
	//
	// Checked before domain.NewUser so an unrecognized role
	// (ErrInvalidRole) and a recognized-but-not-self-assignable one
	// (ErrRoleNotSelfAssignable) stay distinguishable — a caller sending
	// ROLE_HOST_ORGANISER should not be told their role value is invalid,
	// just that it isn't self-assignable here. A garbage/unrecognized role
	// is deliberately left for domain.NewUser's own IsValid loop to reject
	// as ErrInvalidRole, not caught here — this loop only ever rejects a
	// role that IS a recognized, valid Role value but isn't RolePlayer.
	for _, r := range in.Roles {
		if r != domain.RolePlayer && r.IsValid() {
			return domain.User{}, domain.ErrRoleNotSelfAssignable
		}
	}

	// Server-minted, exactly like Facility, Court, Booking, Payment, Game
	// and Competition. domain.NewUser rejects an empty Subject
	// (ErrEmptySubject), so a principal-less call cannot produce a row even
	// if it somehow got past the handler's Unauthenticated guard.
	u, err := domain.NewUser(s.ids.NewID(), in.Subject, in.DisplayName, in.Roles, in.SelfReportedStartingLevel)
	if err != nil {
		return domain.User{}, err
	}

	// Subject uniqueness is enforced by the repository, not by a
	// read-then-write here: two concurrent registrations for one subject
	// would both observe "not registered" and both insert. The Postgres
	// adapter relies on identity_users.subject's UNIQUE constraint and
	// translates the violation to domain.ErrUserAlreadyExists — the same
	// authoritative-constraint discipline Booking's EXCLUDE constraint gets
	// (CLAUDE.md rules 4 and 5).
	return s.repo.Create(ctx, u)
}

// UserBySubject returns the User registered to a verified IdP subject, or
// domain.ErrUserNotFound.
//
// This exists for exactly one caller: the grpcapi boundary, translating an
// auth.Principal into the actor value the domain already understands (a
// User ID) before calling UpdateSelfReportedLevel. Keeping the translation
// here rather than in the handler means the handler needs no repository of
// its own, and keeping it a plain string parameter rather than an
// auth.Principal means this package still imports nothing from
// internal/platform/auth (A11 Ruling 3).
//
// No uuidShape guard: a subject is not a uuid and must never be validated
// as one (see domain.User.Subject). It is passed to the repository as the
// opaque string it is.
func (s *Service) UserBySubject(ctx context.Context, subject string) (domain.User, error) {
	if subject == "" {
		return domain.User{}, domain.ErrUserNotFound
	}
	return s.repo.GetBySubject(ctx, subject)
}

// GetUser returns a single User by id, or domain.ErrUserNotFound.
func (s *Service) GetUser(ctx context.Context, id string) (domain.User, error) {
	// A malformed ID is answered exactly like an unknown one (T10.2 item
	// 4). Besides keeping the adapter's mustUUID from panicking on wire
	// input, this preserves the project convention that an unresolvable
	// User reference is a 404 and never a 500 — see domain.ErrUserNotFound.
	if !uuidShape.MatchString(id) {
		return domain.User{}, domain.ErrUserNotFound
	}
	return s.repo.GetByID(ctx, id)
}

// UpdateSelfReportedLevel fetches the User at id, checks actorUserID is
// that User themself (domain.User.EnsureSelf, T10.2's object-level (BOLA)
// check — a mismatch returns domain.ErrNotSelf -> codes.PermissionDenied
// via grpcapi.toStatus, and never reaches Repository.
// UpdateSelfReportedLevel), and, once level is validated as within its
// bounded range too (domain.ErrInvalidSelfReportedStartingLevel
// otherwise), persists the new level.
//
// The same malformed-id boundary guard GetUser has is applied to id here
// too: this method already calls Repository.GetByID(id) first, before
// EnsureSelf, with the identical unguarded shape GetUser had before T10.2 —
// leaving it unguarded here would be a second instance of exactly the
// issue #97 class of bug this ticket exists to close on the read path,
// just on a write path instead (mirrors internal/facilities/app.Service.
// AddCourt/AddCameraLink/AttestCameraConsent's identical T10.7 guard on
// their own facilityID parameter).
func (s *Service) UpdateSelfReportedLevel(ctx context.Context, id, actorUserID string, level domain.SelfReportedStartingLevel) (domain.User, error) {
	if !uuidShape.MatchString(id) {
		return domain.User{}, domain.ErrUserNotFound
	}

	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return domain.User{}, err
	}

	updated, err := u.UpdateSelfReportedLevel(actorUserID, level)
	if err != nil {
		return domain.User{}, err
	}

	return s.repo.UpdateSelfReportedLevel(ctx, id, updated.SelfReportedStartingLevel)
}
