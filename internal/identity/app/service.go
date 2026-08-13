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
// internal/platform/idgen mints for every other aggregate's ID in this
// codebase (Facility, Court, Booking, Payment, Competition, ...) — User's
// own ID is caller-claimed rather than server-generated (see
// CreateUserInput's doc comment), but it must still be well-formed enough
// for the `uuid` column identity_users.id is stored as.
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
}

func NewService(repo port.Repository) *Service {
	return &Service{repo: repo}
}

// CreateUserInput is CreateUser's use-case input.
//
// ActorUserID becomes the new User's own ID (domain.NewUser's id
// parameter), not a separate server-generated value the way every other
// aggregate in this codebase is minted (via a port.IDGenerator — see
// internal/facilities/app.Service's ids field for the pattern this
// deliberately departs from). Identity/Users is the first entity in this
// codebase that represents the concept of a caller's own identity, so
// there is no pre-existing owner/host/actor to check a claimed
// actor_user_id against the way AddCourt checks it against a Facility's
// OwnerID (T7.7) or RecordOfflinePayment checks it against a Booking's
// HostID (T6.3): the only sensible "same claimed-actor pattern" reading
// for a *creation* RPC with no prior object is that the actor is claiming
// to create their own identity, so their claim IS the created User's ID.
// A second CreateUser call for the same ActorUserID is a real, reachable
// case (unlike every server-generated-ID aggregate) — see
// domain.ErrUserAlreadyExists.
type CreateUserInput struct {
	ActorUserID               string
	DisplayName               string
	Roles                     []domain.Role
	SelfReportedStartingLevel domain.SelfReportedStartingLevel
}

// CreateUser validates and persists a new User. ActorUserID becomes the
// User's own ID — see CreateUserInput's doc comment for why. An empty
// ActorUserID is rejected the same way an empty DisplayName is
// (domain.NewUser's ErrEmptyID -> codes.InvalidArgument via
// grpcapi.toStatus), before anything is persisted.
func (s *Service) CreateUser(ctx context.Context, in CreateUserInput) (domain.User, error) {
	// Guards the same Postgres adapter panic on a malformed (non-empty,
	// non-UUID-shaped) id that GetUser/UpdateSelfReportedLevel guard below
	// — Create's id is caller-claimed via ActorUserID, not server-generated
	// (see CreateUserInput's doc comment), so it reaches mustUUID
	// unvalidated unless checked here. Reuses ErrEmptyID (rather than a new
	// sentinel) for a malformed-but-nonempty id too, mirroring
	// internal/payments/app's CreateOnlinePayment/RecordOfflinePayment,
	// which reuse ErrEmptyPayableID the identical way for their own
	// caller-supplied PayableID — both cases mean the same thing to a
	// caller (InvalidArgument), and this single check also subsumes the
	// plain-empty case domain.NewUser's own ErrEmptyID guard below would
	// otherwise catch (kept regardless, as defense in depth for any future
	// caller that constructs a User directly against the domain package).
	if !uuidShape.MatchString(in.ActorUserID) {
		return domain.User{}, domain.ErrEmptyID
	}

	u, err := domain.NewUser(in.ActorUserID, in.DisplayName, in.Roles, in.SelfReportedStartingLevel)
	if err != nil {
		return domain.User{}, err
	}
	return s.repo.Create(ctx, u)
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
