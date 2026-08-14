// Package grpcapi is the Identity/Users context's gRPC adapter. It
// translates between the wire contract generated from
// proto/pickleball/identity/v1 and the app/domain layers, and maps domain
// errors onto gRPC status codes so grpc-gateway's REST mapping produces the
// right HTTP status (e.g. domain.ErrNotSelf -> codes.PermissionDenied ->
// HTTP 403, not a 500). It only compiles after `make generate` has produced
// internal/gen/pickleball/identity/v1 (see CLAUDE.md gotchas). Mirrors
// internal/facilities/adapter/grpcapi/handler.go's shape.
package grpcapi

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	identityv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/identity/v1"
	"github.com/nhuthuynh/white-label/internal/identity/app"
	"github.com/nhuthuynh/white-label/internal/identity/domain"
	"github.com/nhuthuynh/white-label/internal/platform/auth"
)

// serviceName is the fully-qualified gRPC service name, used to build the
// method strings AuthenticatedMethods reports. Kept as a constant so the
// two entries below cannot drift from each other by a typo.
const serviceName = "/pickleball.identity.v1.IdentityService/"

// AuthenticatedMethods lists the full gRPC method names of this context's
// RPCs that require a verified auth.Principal (T12 sprint plan A11 Ruling
// 2: the policy lives with the handlers that break if it is wrong, and
// cmd/server merely composes each context's list).
//
// The list is deliberately short, and what is ABSENT is a decision too:
//
//   - CreateUser — authenticated (T12.9). This is the change that closes
//     the identity-squatting DoS. The row's identity now comes from the
//     token, so there is no anonymous path to User creation left at all;
//     the surface is removed, not narrowed the way T10.2's role
//     restriction narrowed it.
//   - UpdateSelfReportedLevel — authenticated (T12.9). It mutates a fact
//     about a specific person, and its actor is now the verified principal
//     rather than a wire claim.
//   - GetUser — deliberately NOT listed. It is a shipped, public,
//     unauthenticated read, and quietly authenticating it would break a
//     live flow for no security gain (the same mistake T12.7's instruction
//     2 warns about). A User's profile being publicly readable is an
//     existing product decision; this ticket does not reopen it.
//
// Handlers enforce this themselves as well (see CreateUser below). The list
// is the interceptor-level backstop, so that a future RPC added to this
// service without its own guard still cannot be reached anonymously.
func AuthenticatedMethods() []string {
	return []string{
		serviceName + "CreateUser",
		serviceName + "UpdateSelfReportedLevel",
	}
}

type Handler struct {
	identityv1.UnimplementedIdentityServiceServer
	svc *app.Service
}

func NewHandler(svc *app.Service) *Handler {
	return &Handler{svc: svc}
}

// CreateUser registers the verified caller as a new User.
//
// The request's actor_user_id field is READ NOWHERE in this method, and
// that is the entire security fix rather than an incidental tidy-up. Before
// T12.9 it became the new row's permanent primary key, so any anonymous
// caller could name any uuid — including one a real-auth integration would
// later mint for a real person — and permanently occupy that identity
// (HANDOFF.md's T10.2 bullet). The field is retained on the wire only
// because deleting a proto field breaks clients; it is deprecated and
// ignored. A handler that fell back to it when no principal was present
// would have changed nothing, which is why
// TestCreateUser_PrincipalOverridesWireActorClaim asserts specifically
// against that fallback.
//
// No principal is codes.Unauthenticated — "I do not know who you are" —
// never PermissionDenied, which would claim we knew the caller and were
// refusing them (T12.2's ADR-0013 §5).
func (h *Handler) CreateUser(ctx context.Context, req *identityv1.CreateUserRequest) (*identityv1.CreateUserResponse, error) {
	principal, ok := auth.PrincipalFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "identity: creating a user requires an authenticated caller")
	}

	u, err := h.svc.CreateUser(ctx, app.CreateUserInput{
		// The verified subject, not req.GetActorUserId(). Deliberate.
		Subject:                   principal.Subject,
		DisplayName:               req.GetDisplayName(),
		Roles:                     fromProtoRoles(req.GetRoles()),
		SelfReportedStartingLevel: domain.SelfReportedStartingLevel(req.GetSelfReportedStartingLevel()),
	})
	if err != nil {
		return nil, toStatus(err)
	}
	return &identityv1.CreateUserResponse{User: toProto(u)}, nil
}

func (h *Handler) GetUser(ctx context.Context, req *identityv1.GetUserRequest) (*identityv1.GetUserResponse, error) {
	u, err := h.svc.GetUser(ctx, req.GetUserId())
	if err != nil {
		return nil, toStatus(err)
	}
	return &identityv1.GetUserResponse{User: toProto(u)}, nil
}

// UpdateSelfReportedLevel updates the calling User's own self-reported
// level.
//
// Like CreateUser, the request's actor_user_id field is ignored: the actor
// is resolved from the verified principal. The translation takes one extra
// step here because domain.User.EnsureSelf compares User IDs, not subjects
// — so the handler looks up the User owning the principal's subject and
// passes THAT user's ID down. Keeping the translation at this boundary is
// what lets app and domain keep their existing signatures and stay free of
// any internal/platform/auth import (A11 Ruling 3), and it is why every
// pre-existing EnsureSelf test remains valid unchanged.
//
// A verified caller with no registered User of their own is
// PermissionDenied, not Unauthenticated: we know exactly who they are, and
// they are provably not the target User. Reported as domain.ErrNotSelf so
// it is indistinguishable from any other non-owner rejection — telling an
// authenticated stranger "that user exists but you have no account" would
// leak more than refusing them does.
func (h *Handler) UpdateSelfReportedLevel(ctx context.Context, req *identityv1.UpdateSelfReportedLevelRequest) (*identityv1.UpdateSelfReportedLevelResponse, error) {
	principal, ok := auth.PrincipalFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "identity: updating a self-reported level requires an authenticated caller")
	}

	actor, err := h.svc.UserBySubject(ctx, principal.Subject)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, toStatus(domain.ErrNotSelf)
		}
		return nil, toStatus(err)
	}

	u, err := h.svc.UpdateSelfReportedLevel(ctx, req.GetUserId(), actor.ID, domain.SelfReportedStartingLevel(req.GetSelfReportedStartingLevel()))
	if err != nil {
		return nil, toStatus(err)
	}
	return &identityv1.UpdateSelfReportedLevelResponse{User: toProto(u)}, nil
}

// toStatus maps domain errors to gRPC status codes only — never an HTTP
// status alongside them (T9 retro finding 4 / this sprint's A6, adopted
// after T9.4's shipped FailedPrecondition/409 contradiction). grpc-gateway
// then maps those codes onto HTTP statuses on its own: NotFound -> 404,
// InvalidArgument -> 400, PermissionDenied -> 403, AlreadyExists -> 409.
func toStatus(err error) error {
	switch {
	case errors.Is(err, domain.ErrUserNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrNotSelf):
		// PermissionDenied (-> HTTP 403 via grpc-gateway), not Internal:
		// T10.2's object-level (BOLA) authorization rejection — mirrors
		// internal/facilities/adapter/grpcapi's ErrNotFacilityOwner
		// handling (T7.7) and internal/socialplay/adapter/grpcapi's
		// ErrNotRegistrationOwner handling (T5.5).
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, domain.ErrUserAlreadyExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, domain.ErrEmptyID),
		errors.Is(err, domain.ErrEmptySubject),
		errors.Is(err, domain.ErrEmptyDisplayName),
		errors.Is(err, domain.ErrEmptyRoles),
		errors.Is(err, domain.ErrInvalidRole),
		errors.Is(err, domain.ErrRoleNotSelfAssignable),
		errors.Is(err, domain.ErrInvalidSelfReportedStartingLevel):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}

// toProto converts a domain.User to its wire form.
//
// domain.User.Subject is deliberately NOT mapped onto any wire field, and
// no such field was added to the User message. GetUser is a public,
// unauthenticated read, so exposing the subject would publish every user's
// identity-provider identifier to anonymous callers — an external
// identifier that is useful for correlation and belongs to the IdP, not to
// this platform's public profile surface. The subject stays server-side; the
// wire keeps showing the platform's own `id`.
func toProto(u domain.User) *identityv1.User {
	return &identityv1.User{
		Id:                        u.ID,
		DisplayName:               u.DisplayName,
		Roles:                     toProtoRoles(u.Roles),
		SelfReportedStartingLevel: int32(u.SelfReportedStartingLevel),
	}
}

func toProtoRoles(roles []domain.Role) []identityv1.Role {
	out := make([]identityv1.Role, 0, len(roles))
	for _, r := range roles {
		out = append(out, toProtoRole(r))
	}
	return out
}

func toProtoRole(r domain.Role) identityv1.Role {
	switch r {
	case domain.RolePlayer:
		return identityv1.Role_ROLE_PLAYER
	case domain.RoleHostOrganiser:
		return identityv1.Role_ROLE_HOST_ORGANISER
	case domain.RoleGameAdmin:
		return identityv1.Role_ROLE_GAME_ADMIN
	case domain.RoleFacilityOwner:
		return identityv1.Role_ROLE_FACILITY_OWNER
	case domain.RoleClub:
		return identityv1.Role_ROLE_CLUB
	case domain.RolePlatformAdmin:
		return identityv1.Role_ROLE_PLATFORM_ADMIN
	default:
		return identityv1.Role_ROLE_UNSPECIFIED
	}
}

// fromProtoRoles converts every wire Role to a domain.Role, including
// ROLE_UNSPECIFIED and any future enum value this handler doesn't
// recognize yet — both map to domain.Role(""), which domain.NewUser's own
// per-element IsValid loop already rejects with domain.ErrInvalidRole
// (-> codes.InvalidArgument via toStatus above). No separate error path is
// needed here: unrecognized-role rejection is handled once, in the domain,
// not duplicated at this boundary too.
func fromProtoRoles(roles []identityv1.Role) []domain.Role {
	out := make([]domain.Role, 0, len(roles))
	for _, r := range roles {
		out = append(out, fromProtoRole(r))
	}
	return out
}

func fromProtoRole(r identityv1.Role) domain.Role {
	switch r {
	case identityv1.Role_ROLE_PLAYER:
		return domain.RolePlayer
	case identityv1.Role_ROLE_HOST_ORGANISER:
		return domain.RoleHostOrganiser
	case identityv1.Role_ROLE_GAME_ADMIN:
		return domain.RoleGameAdmin
	case identityv1.Role_ROLE_FACILITY_OWNER:
		return domain.RoleFacilityOwner
	case identityv1.Role_ROLE_CLUB:
		return domain.RoleClub
	case identityv1.Role_ROLE_PLATFORM_ADMIN:
		return domain.RolePlatformAdmin
	default:
		return domain.Role("")
	}
}
