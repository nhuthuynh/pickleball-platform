// Package grpcapi is the Social Play context's gRPC adapter. It translates
// between the wire contract generated from proto/pickleball/socialplay/v1
// and the app/domain layers, and maps domain errors onto gRPC status codes
// so grpc-gateway's REST mapping produces the right HTTP status — mirrors
// internal/booking/adapter/grpcapi's shape. It only compiles after `make
// generate` has produced internal/gen/pickleball/socialplay/v1 (see
// CLAUDE.md gotchas).
package grpcapi

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/nhuthuynh/white-label/internal/socialplay/app"
	"github.com/nhuthuynh/white-label/internal/socialplay/domain"
	"github.com/nhuthuynh/white-label/internal/socialplay/port"

	socialplayv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/socialplay/v1"
)

// Handler serves SocialPlayService. It holds the CourtReservation port
// alongside the app.Service because ScheduleGame takes it per-call (T5.3's
// original design — see app.Service.ScheduleGame's doc comment) rather than
// storing it on Service itself.
type Handler struct {
	socialplayv1.UnimplementedSocialPlayServiceServer
	svc         *app.Service
	reservation port.CourtReservation
}

func NewHandler(svc *app.Service, reservation port.CourtReservation) *Handler {
	return &Handler{svc: svc, reservation: reservation}
}

func (h *Handler) CreateGame(ctx context.Context, req *socialplayv1.CreateGameRequest) (*socialplayv1.CreateGameResponse, error) {
	rng, err := domain.NewTimeRange(req.GetStartsAt().AsTime(), req.GetEndsAt().AsTime())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	g, err := h.svc.ScheduleGame(ctx, app.ScheduleGameInput{
		HostID:     req.GetHostId(),
		FacilityID: req.GetFacilityId(),
		CourtIDs:   req.GetCourtIds(),
		Range:      rng,
		Capacity:   int(req.GetCapacity()),
	}, h.reservation)
	if err != nil {
		return nil, toStatus(err)
	}

	return &socialplayv1.CreateGameResponse{Game: toProtoGame(g)}, nil
}

func (h *Handler) RegisterForGame(ctx context.Context, req *socialplayv1.RegisterForGameRequest) (*socialplayv1.RegisterForGameResponse, error) {
	reg, err := h.svc.RegisterForGame(ctx, app.RegisterForGameInput{
		GameID:   req.GetGameId(),
		PlayerID: req.GetPlayerId(),
	})
	if err != nil {
		return nil, toStatus(err)
	}
	return &socialplayv1.RegisterForGameResponse{Registration: toProtoRegistration(reg)}, nil
}

func (h *Handler) CancelRegistration(ctx context.Context, req *socialplayv1.CancelRegistrationRequest) (*socialplayv1.CancelRegistrationResponse, error) {
	reg, err := h.svc.CancelRegistration(ctx, req.GetRegistrationId(), req.GetActorPlayerId())
	if err != nil {
		return nil, toStatus(err)
	}
	return &socialplayv1.CancelRegistrationResponse{Registration: toProtoRegistration(reg)}, nil
}

// toStatus maps domain errors to gRPC status codes; grpc-gateway then maps
// those onto HTTP statuses (AlreadyExists -> 409, InvalidArgument -> 400,
// NotFound -> 404, PermissionDenied -> 403) — mirrors
// internal/booking/adapter/grpcapi.toStatus. ErrGameFull, ErrAlreadyRegistered,
// and ErrCourtUnavailable all map to the same conflict status
// (AlreadyExists/409) that ErrCourtDoubleBooked already uses for Booking,
// per the ticket's smoke-test AC. ErrNotRegistrationOwner maps to
// PermissionDenied/403 — the BOLA-shaped rejection T5.2/T5.5 require ("a
// clear rejection, not a 500").
func toStatus(err error) error {
	switch {
	case errors.Is(err, domain.ErrGameFull),
		errors.Is(err, domain.ErrAlreadyRegistered),
		errors.Is(err, domain.ErrCourtUnavailable):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, domain.ErrNotRegistrationOwner):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, domain.ErrGameNotFound), errors.Is(err, domain.ErrRegistrationNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrInvalidTimeRange),
		errors.Is(err, domain.ErrInvalidCapacity),
		errors.Is(err, domain.ErrEmptyCourtIDs),
		errors.Is(err, domain.ErrEmptyPlayerID),
		errors.Is(err, domain.ErrIllegalStatusTransition):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}

func toProtoGameStatus(s domain.Status) socialplayv1.GameStatus {
	switch s {
	case domain.StatusScheduled:
		return socialplayv1.GameStatus_GAME_STATUS_SCHEDULED
	case domain.StatusCancelled:
		return socialplayv1.GameStatus_GAME_STATUS_CANCELLED
	default:
		return socialplayv1.GameStatus_GAME_STATUS_UNSPECIFIED
	}
}

func toProtoRegistrationStatus(s domain.RegistrationStatus) socialplayv1.RegistrationStatus {
	switch s {
	case domain.RegistrationStatusRegistered:
		return socialplayv1.RegistrationStatus_REGISTRATION_STATUS_REGISTERED
	case domain.RegistrationStatusCancelled:
		return socialplayv1.RegistrationStatus_REGISTRATION_STATUS_CANCELLED
	default:
		return socialplayv1.RegistrationStatus_REGISTRATION_STATUS_UNSPECIFIED
	}
}

func toProtoPaymentStatus(s domain.PaymentStatus) socialplayv1.PaymentStatus {
	switch s {
	case domain.PaymentStatusUnpaid:
		return socialplayv1.PaymentStatus_PAYMENT_STATUS_UNPAID
	case domain.PaymentStatusPaid:
		return socialplayv1.PaymentStatus_PAYMENT_STATUS_PAID
	default:
		return socialplayv1.PaymentStatus_PAYMENT_STATUS_UNSPECIFIED
	}
}

func toProtoGame(g domain.Game) *socialplayv1.Game {
	return &socialplayv1.Game{
		Id:         g.ID,
		HostId:     g.HostID,
		FacilityId: g.FacilityID,
		CourtIds:   g.CourtIDs,
		StartsAt:   timestamppb.New(g.Range.Start),
		EndsAt:     timestamppb.New(g.Range.End),
		Capacity:   int32(g.Capacity),
		Status:     toProtoGameStatus(g.Status),
	}
}

func toProtoRegistration(r domain.Registration) *socialplayv1.Registration {
	return &socialplayv1.Registration{
		Id:            r.ID,
		GameId:        r.GameID,
		PlayerId:      r.PlayerID,
		Status:        toProtoRegistrationStatus(r.Status),
		PaymentStatus: toProtoPaymentStatus(r.PaymentStatus),
	}
}
