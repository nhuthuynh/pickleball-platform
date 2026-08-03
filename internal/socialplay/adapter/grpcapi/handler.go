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

// Handler serves SocialPlayService. It holds the CourtReservation and
// FacilityLookup ports alongside the app.Service because ScheduleGame takes
// both per-call (T5.3's original design for CourtReservation — see
// app.Service.ScheduleGame's doc comment; T8.3 gives FacilityLookup the
// same treatment) rather than storing them on Service itself.
type Handler struct {
	socialplayv1.UnimplementedSocialPlayServiceServer
	svc         *app.Service
	reservation port.CourtReservation
	facilities  port.FacilityLookup
}

func NewHandler(svc *app.Service, reservation port.CourtReservation, facilities port.FacilityLookup) *Handler {
	return &Handler{svc: svc, reservation: reservation, facilities: facilities}
}

func (h *Handler) CreateGame(ctx context.Context, req *socialplayv1.CreateGameRequest) (*socialplayv1.CreateGameResponse, error) {
	rng, err := domain.NewTimeRange(req.GetStartsAt().AsTime(), req.GetEndsAt().AsTime())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	g, err := h.svc.ScheduleGame(ctx, app.ScheduleGameInput{
		HostID:          req.GetHostId(),
		FacilityID:      req.GetFacilityId(),
		VenueFacilityID: req.GetVenueFacilityId(),
		CourtIDs:        req.GetCourtIds(),
		Range:           rng,
		Capacity:        int(req.GetCapacity()),
		PaymentMethod:   fromProtoPaymentMethod(req.GetPaymentMethod()),
		GuestAllowance:  int(req.GetGuestAllowance()),
	}, h.reservation, h.facilities)
	if err != nil {
		return nil, toStatus(err)
	}

	return &socialplayv1.CreateGameResponse{Game: toProtoGame(g)}, nil
}

func (h *Handler) RegisterForGame(ctx context.Context, req *socialplayv1.RegisterForGameRequest) (*socialplayv1.RegisterForGameResponse, error) {
	reg, err := h.svc.RegisterForGame(ctx, app.RegisterForGameInput{
		GameID:     req.GetGameId(),
		PlayerID:   req.GetPlayerId(),
		GuestCount: int(req.GetGuestCount()),
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

func (h *Handler) JoinWaitlist(ctx context.Context, req *socialplayv1.JoinWaitlistRequest) (*socialplayv1.JoinWaitlistResponse, error) {
	entry, err := h.svc.JoinWaitlist(ctx, app.JoinWaitlistInput{
		GameID:   req.GetGameId(),
		PlayerID: req.GetPlayerId(),
	})
	if err != nil {
		return nil, toStatus(err)
	}
	return &socialplayv1.JoinWaitlistResponse{Entry: toProtoWaitlistEntry(entry)}, nil
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
//
// T6.6 additions: ErrAlreadyOnWaitlist and ErrGameNotFull join the
// AlreadyExists/InvalidArgument groups respectively (mirroring
// ErrAlreadyRegistered's and the validation-error group's own reasoning —
// "not full" is a precondition violation on the request, not a conflict).
// ErrNotWaitlistEntryOwner joins ErrNotRegistrationOwner's PermissionDenied
// group. ErrWaitlistEntryNotFound joins the NotFound group.
// ErrWaitlistPromotionNotExpired and ErrNoWaitingEntries are deliberately
// NOT mapped here: neither is reachable from a client-facing RPC in this
// ticket (JoinWaitlist is the only new RPC; promotion/expiry are
// app-layer-internal, triggered by CancelRegistration and a future
// sweep/admin path, not exposed directly) — falling through to Internal for
// either would be a signal something upstream called an app method it
// shouldn't have, not a case this handler needs to translate for a client.
//
// T8.3 addition: ErrFacilityNotFound joins the NotFound group — an unknown
// CreateGameRequest.venue_facility_id is a 404, not a 500 or a silent
// accept (the ticket's explicit requirement).
//
// T8.7 additions: ErrInvalidPaymentMethod, ErrInvalidGuestAllowance, and
// ErrGuestAllowanceExceeded join the validation-error/InvalidArgument
// group — all three are precondition violations on the request itself
// (a bad enum value, a negative allowance, or a guest count outside the
// Game's own allowance), the same category ErrInvalidCapacity/
// ErrEmptyCourtIDs already occupy.
func toStatus(err error) error {
	switch {
	case errors.Is(err, domain.ErrGameFull),
		errors.Is(err, domain.ErrAlreadyRegistered),
		errors.Is(err, domain.ErrCourtUnavailable),
		errors.Is(err, domain.ErrAlreadyOnWaitlist):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, domain.ErrNotRegistrationOwner),
		errors.Is(err, domain.ErrNotWaitlistEntryOwner):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, domain.ErrGameNotFound),
		errors.Is(err, domain.ErrRegistrationNotFound),
		errors.Is(err, domain.ErrWaitlistEntryNotFound),
		errors.Is(err, domain.ErrFacilityNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrInvalidTimeRange),
		errors.Is(err, domain.ErrInvalidCapacity),
		errors.Is(err, domain.ErrEmptyCourtIDs),
		errors.Is(err, domain.ErrEmptyPlayerID),
		errors.Is(err, domain.ErrIllegalStatusTransition),
		errors.Is(err, domain.ErrGameNotFull),
		errors.Is(err, domain.ErrInvalidPaymentMethod),
		errors.Is(err, domain.ErrInvalidGuestAllowance),
		errors.Is(err, domain.ErrGuestAllowanceExceeded):
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
	case domain.PaymentStatusRefunded:
		return socialplayv1.PaymentStatus_PAYMENT_STATUS_REFUNDED
	default:
		return socialplayv1.PaymentStatus_PAYMENT_STATUS_UNSPECIFIED
	}
}

// toProtoPaymentMethod converts a domain.PaymentMethod to its wire enum
// (T8.7) — mirrors toProtoGameStatus/toProtoRegistrationStatus's shape. Every
// domain.PaymentMethod value is one of its own closed enum's non-zero
// values, so there is no "unspecified" case to fall into on this direction
// (unlike fromProtoPaymentMethod below, which does have to handle the wire
// zero value).
func toProtoPaymentMethod(m domain.PaymentMethod) socialplayv1.PaymentMethod {
	switch m {
	case domain.PaymentMethodOnline:
		return socialplayv1.PaymentMethod_PAYMENT_METHOD_ONLINE
	case domain.PaymentMethodCash:
		return socialplayv1.PaymentMethod_PAYMENT_METHOD_CASH
	case domain.PaymentMethodEither:
		return socialplayv1.PaymentMethod_PAYMENT_METHOD_EITHER
	default:
		return socialplayv1.PaymentMethod_PAYMENT_METHOD_UNSPECIFIED
	}
}

// fromProtoPaymentMethod converts a wire PaymentMethod into a
// domain.PaymentMethod (T8.7). PAYMENT_METHOD_UNSPECIFIED — the wire zero
// value, sent by any client that never sets this field, including every
// pre-T8.7 client — resolves to domain.PaymentMethodEither, the least
// restrictive value: see db/migrations/0012_socialplay_guest_capacity.sql's
// payment_method column default for the identical "closest thing to
// unspecified" reasoning applied at the DB layer. This keeps CreateGame
// backward compatible: a request that predates this field behaves exactly
// as it did before T8.6/T8.7 introduced PaymentMethod at all. Any OTHER,
// unrecognized wire value (not one of the enum's named non-zero constants —
// e.g. a value from a future proto version this build doesn't know about)
// is deliberately NOT folded into the same "unspecified" default: it's
// passed through as an invalid domain.PaymentMethod so domain.NewGame's own
// IsValid() check rejects it via ErrInvalidPaymentMethod, rather than
// silently accepting an unrecognized value as if it meant "either" (a
// PE-review finding on T8.7's original PR — UNSPECIFIED and "unrecognized"
// are not the same thing and must not share a fallback).
func fromProtoPaymentMethod(m socialplayv1.PaymentMethod) domain.PaymentMethod {
	switch m {
	case socialplayv1.PaymentMethod_PAYMENT_METHOD_UNSPECIFIED:
		return domain.PaymentMethodEither
	case socialplayv1.PaymentMethod_PAYMENT_METHOD_ONLINE:
		return domain.PaymentMethodOnline
	case socialplayv1.PaymentMethod_PAYMENT_METHOD_CASH:
		return domain.PaymentMethodCash
	case socialplayv1.PaymentMethod_PAYMENT_METHOD_EITHER:
		return domain.PaymentMethodEither
	default:
		return domain.PaymentMethod("")
	}
}

func toProtoGame(g domain.Game) *socialplayv1.Game {
	return &socialplayv1.Game{
		Id:              g.ID,
		HostId:          g.HostID,
		FacilityId:      g.FacilityID,
		VenueFacilityId: g.VenueFacilityID,
		CourtIds:        g.CourtIDs,
		StartsAt:        timestamppb.New(g.Range.Start),
		EndsAt:          timestamppb.New(g.Range.End),
		Capacity:        int32(g.Capacity),
		Status:          toProtoGameStatus(g.Status),
		PaymentMethod:   toProtoPaymentMethod(g.PaymentMethod),
		GuestAllowance:  int32(g.GuestAllowance),
	}
}

func toProtoRegistration(r domain.Registration) *socialplayv1.Registration {
	return &socialplayv1.Registration{
		Id:            r.ID,
		GameId:        r.GameID,
		PlayerId:      r.PlayerID,
		Status:        toProtoRegistrationStatus(r.Status),
		PaymentStatus: toProtoPaymentStatus(r.PaymentStatus),
		GuestCount:    int32(r.GuestCount),
	}
}

func toProtoWaitlistStatus(s domain.WaitlistStatus) socialplayv1.WaitlistStatus {
	switch s {
	case domain.WaitlistStatusWaiting:
		return socialplayv1.WaitlistStatus_WAITLIST_STATUS_WAITING
	case domain.WaitlistStatusPromoted:
		return socialplayv1.WaitlistStatus_WAITLIST_STATUS_PROMOTED
	case domain.WaitlistStatusExpired:
		return socialplayv1.WaitlistStatus_WAITLIST_STATUS_EXPIRED
	case domain.WaitlistStatusCancelled:
		return socialplayv1.WaitlistStatus_WAITLIST_STATUS_CANCELLED
	default:
		return socialplayv1.WaitlistStatus_WAITLIST_STATUS_UNSPECIFIED
	}
}

func toProtoWaitlistEntry(e domain.WaitlistEntry) *socialplayv1.WaitlistEntry {
	out := &socialplayv1.WaitlistEntry{
		Id:       e.ID,
		GameId:   e.GameID,
		PlayerId: e.PlayerID,
		Position: int32(e.Position),
		Status:   toProtoWaitlistStatus(e.Status),
	}
	if !e.PromotedAt.IsZero() {
		out.PromotedAt = timestamppb.New(e.PromotedAt)
	}
	return out
}
