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
)

type Handler struct {
	identityv1.UnimplementedIdentityServiceServer
	svc *app.Service
}

func NewHandler(svc *app.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) CreateUser(ctx context.Context, req *identityv1.CreateUserRequest) (*identityv1.CreateUserResponse, error) {
	u, err := h.svc.CreateUser(ctx, app.CreateUserInput{
		ActorUserID:               req.GetActorUserId(),
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

func (h *Handler) UpdateSelfReportedLevel(ctx context.Context, req *identityv1.UpdateSelfReportedLevelRequest) (*identityv1.UpdateSelfReportedLevelResponse, error) {
	u, err := h.svc.UpdateSelfReportedLevel(ctx, req.GetUserId(), req.GetActorUserId(), domain.SelfReportedStartingLevel(req.GetSelfReportedStartingLevel()))
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
		errors.Is(err, domain.ErrEmptyDisplayName),
		errors.Is(err, domain.ErrEmptyRoles),
		errors.Is(err, domain.ErrInvalidRole),
		errors.Is(err, domain.ErrInvalidSelfReportedStartingLevel):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}

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
