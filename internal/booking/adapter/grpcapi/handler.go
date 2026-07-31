// Package grpcapi is the Booking context's gRPC adapter. It translates
// between the wire contract generated from proto/pickleball/booking/v1 and
// the app/domain layers, and maps domain errors onto gRPC status codes so
// grpc-gateway's REST mapping produces the right HTTP status (e.g.
// ErrCourtDoubleBooked -> codes.AlreadyExists -> HTTP 409). It only compiles
// after `make generate` has produced internal/gen/pickleball/booking/v1 (see
// CLAUDE.md gotchas).
package grpcapi

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/nhuthuynh/white-label/internal/booking/app"
	"github.com/nhuthuynh/white-label/internal/booking/domain"
	bookingv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/booking/v1"
)

type Handler struct {
	bookingv1.UnimplementedBookingServiceServer
	svc *app.Service
}

func NewHandler(svc *app.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) CreateBooking(ctx context.Context, req *bookingv1.CreateBookingRequest) (*bookingv1.CreateBookingResponse, error) {
	source, err := fromProtoSource(req.GetSource())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	rng, err := domain.NewTimeRange(req.GetStartsAt().AsTime(), req.GetEndsAt().AsTime())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	b, err := h.svc.CreateBooking(ctx, app.CreateBookingInput{
		CourtID:     req.GetCourtId(),
		Source:      source,
		Range:       rng,
		ReferenceID: req.GetReferenceId(),
	})
	if err != nil {
		return nil, toStatus(err)
	}

	return &bookingv1.CreateBookingResponse{Booking: toProto(b)}, nil
}

// toStatus maps domain errors to gRPC status codes. grpc-gateway then maps
// those codes onto HTTP statuses: AlreadyExists -> 409, InvalidArgument ->
// 400, NotFound -> 404 — this is what makes README.md's "overlapping booking
// returns 409" smoke test true end-to-end.
func toStatus(err error) error {
	switch {
	case errors.Is(err, domain.ErrCourtDoubleBooked):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, domain.ErrBookingNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrInvalidTimeRange),
		errors.Is(err, domain.ErrInvalidSource),
		errors.Is(err, domain.ErrEmptyCourtID),
		errors.Is(err, domain.ErrIllegalStatusTransition):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}

func fromProtoSource(s bookingv1.Source) (domain.Source, error) {
	switch s {
	case bookingv1.Source_SOURCE_RECURRING_HIRE:
		return domain.SourceRecurringHire, nil
	case bookingv1.Source_SOURCE_INDIVIDUAL:
		return domain.SourceIndividual, nil
	case bookingv1.Source_SOURCE_GAME:
		return domain.SourceGame, nil
	case bookingv1.Source_SOURCE_COMPETITION:
		return domain.SourceCompetition, nil
	default:
		return "", domain.ErrInvalidSource
	}
}

func toProtoSource(s domain.Source) bookingv1.Source {
	switch s {
	case domain.SourceRecurringHire:
		return bookingv1.Source_SOURCE_RECURRING_HIRE
	case domain.SourceIndividual:
		return bookingv1.Source_SOURCE_INDIVIDUAL
	case domain.SourceGame:
		return bookingv1.Source_SOURCE_GAME
	case domain.SourceCompetition:
		return bookingv1.Source_SOURCE_COMPETITION
	default:
		return bookingv1.Source_SOURCE_UNSPECIFIED
	}
}

func toProtoStatus(s domain.Status) bookingv1.Status {
	switch s {
	case domain.StatusConfirmed:
		return bookingv1.Status_STATUS_CONFIRMED
	case domain.StatusCancelled:
		return bookingv1.Status_STATUS_CANCELLED
	default:
		return bookingv1.Status_STATUS_UNSPECIFIED
	}
}

func toProto(b domain.Booking) *bookingv1.Booking {
	return &bookingv1.Booking{
		Id:          b.ID,
		CourtId:     b.CourtID,
		Source:      toProtoSource(b.Source),
		Status:      toProtoStatus(b.Status),
		StartsAt:    timestamppb.New(b.Range.Start),
		EndsAt:      timestamppb.New(b.Range.End),
		ReferenceId: b.ReferenceID,
	}
}
