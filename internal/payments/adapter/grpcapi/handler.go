// Package grpcapi is the Payments context's gRPC adapter. It translates
// between the wire contract generated from proto/pickleball/payments/v1
// and the app/domain layers, persists results via port.Repository, and
// maps domain errors onto gRPC status codes so grpc-gateway's REST mapping
// produces the right HTTP status (e.g. ErrPaymentAlreadyRecorded ->
// codes.AlreadyExists -> HTTP 409, ErrNotPaymentRecorder ->
// codes.PermissionDenied -> HTTP 403). It only compiles after `make
// generate` has produced internal/gen/pickleball/payments/v1 (see
// CLAUDE.md gotchas). Mirrors internal/booking/adapter/grpcapi's shape.
package grpcapi

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/nhuthuynh/white-label/internal/payments/app"
	"github.com/nhuthuynh/white-label/internal/payments/domain"

	paymentsv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/payments/v1"
)

type Handler struct {
	paymentsv1.UnimplementedPaymentsServiceServer
	svc *app.Service
}

func NewHandler(svc *app.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RecordOfflinePayment(ctx context.Context, req *paymentsv1.RecordOfflinePaymentRequest) (*paymentsv1.RecordOfflinePaymentResponse, error) {
	payableType, err := fromProtoPayableType(req.GetPayableType())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	p, err := h.svc.RecordOfflinePayment(ctx, app.RecordOfflinePaymentInput{
		PayableType:              payableType,
		PayableID:                req.GetPayableId(),
		Amount:                   fromProtoMoney(req.GetAmount()),
		ActorUserID:              req.GetActorUserId(),
		BookingHostID:            req.GetBookingHostId(),
		GameHostID:               req.GetGameHostId(),
		AssignedGameAdminUserIDs: req.GetAssignedGameAdminUserIds(),
	})
	if err != nil {
		return nil, toStatus(err)
	}

	return &paymentsv1.RecordOfflinePaymentResponse{Payment: toProto(p)}, nil
}

func (h *Handler) CreateOnlinePayment(ctx context.Context, req *paymentsv1.CreateOnlinePaymentRequest) (*paymentsv1.CreateOnlinePaymentResponse, error) {
	payableType, err := fromProtoPayableType(req.GetPayableType())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	p, err := h.svc.CreateOnlinePayment(ctx, app.CreateOnlinePaymentInput{
		PayableType: payableType,
		PayableID:   req.GetPayableId(),
		Amount:      fromProtoMoney(req.GetAmount()),
	})
	if err != nil {
		return nil, toStatus(err)
	}

	return &paymentsv1.CreateOnlinePaymentResponse{Payment: toProto(p)}, nil
}

// ConfirmOnlinePayment loads the Payment referred to by payment_id and
// hands it to app.Service.ConfirmOnlinePayment — the app layer's
// ConfirmOnlinePayment signature takes the Payment itself (T6.2's original
// shape, kept unchanged by the T6.4 merge, see internal/payments/app/
// service.go's doc comment), so this handler is the "caller" that does the
// id -> Payment lookup the wire contract's payment_id-only request
// implies.
func (h *Handler) ConfirmOnlinePayment(ctx context.Context, req *paymentsv1.ConfirmOnlinePaymentRequest) (*paymentsv1.ConfirmOnlinePaymentResponse, error) {
	p, err := h.svc.Payments().GetByID(ctx, req.GetPaymentId())
	if err != nil {
		return nil, toStatus(err)
	}

	confirmed, err := h.svc.ConfirmOnlinePayment(ctx, p)
	if err != nil {
		return nil, toStatus(err)
	}

	return &paymentsv1.ConfirmOnlinePaymentResponse{Payment: toProto(confirmed)}, nil
}

// toStatus maps domain errors to gRPC status codes. grpc-gateway then maps
// those codes onto HTTP statuses: AlreadyExists -> 409, PermissionDenied ->
// 403, NotFound -> 404, InvalidArgument -> 400 — this is what makes the PR
// description's smoke-test AC true end-to-end (duplicate offline recording
// -> 409, actor mismatch -> 403-shaped, not 500).
func toStatus(err error) error {
	switch {
	case errors.Is(err, domain.ErrPaymentAlreadyRecorded):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, domain.ErrNotPaymentRecorder):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, domain.ErrPaymentNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrPaymentDeclined):
		// A decline is a legitimate business outcome, not a client input
		// error or a server bug (see port.PaymentProcessor's doc
		// comment) — FailedPrecondition (-> HTTP 400 via grpc-gateway)
		// distinguishes it from both InvalidArgument-shaped bad input and
		// an actual Internal server error.
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, domain.ErrPaymentProcessorUnavailable):
		return status.Error(codes.Unavailable, err.Error())
	case errors.Is(err, domain.ErrEmptyPayableID),
		errors.Is(err, domain.ErrInvalidPayableType),
		errors.Is(err, domain.ErrInvalidAmount),
		errors.Is(err, domain.ErrInvalidCurrency),
		errors.Is(err, domain.ErrIllegalStatusTransition):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}

func fromProtoPayableType(t paymentsv1.PayableType) (domain.PayableType, error) {
	switch t {
	case paymentsv1.PayableType_PAYABLE_TYPE_BOOKING:
		return domain.PayableTypeBooking, nil
	case paymentsv1.PayableType_PAYABLE_TYPE_REGISTRATION:
		return domain.PayableTypeRegistration, nil
	case paymentsv1.PayableType_PAYABLE_TYPE_NO_SHOW_FEE:
		return domain.PayableTypeNoShowFee, nil
	default:
		return "", domain.ErrInvalidPayableType
	}
}

func toProtoPayableType(t domain.PayableType) paymentsv1.PayableType {
	switch t {
	case domain.PayableTypeBooking:
		return paymentsv1.PayableType_PAYABLE_TYPE_BOOKING
	case domain.PayableTypeRegistration:
		return paymentsv1.PayableType_PAYABLE_TYPE_REGISTRATION
	case domain.PayableTypeNoShowFee:
		return paymentsv1.PayableType_PAYABLE_TYPE_NO_SHOW_FEE
	default:
		return paymentsv1.PayableType_PAYABLE_TYPE_UNSPECIFIED
	}
}

func toProtoMethod(m domain.Method) paymentsv1.PaymentMethod {
	switch m {
	case domain.MethodOnline:
		return paymentsv1.PaymentMethod_PAYMENT_METHOD_ONLINE
	case domain.MethodOffline:
		return paymentsv1.PaymentMethod_PAYMENT_METHOD_OFFLINE
	default:
		return paymentsv1.PaymentMethod_PAYMENT_METHOD_UNSPECIFIED
	}
}

func toProtoStatus(s domain.Status) paymentsv1.PaymentStatus {
	switch s {
	case domain.StatusUnpaid:
		return paymentsv1.PaymentStatus_PAYMENT_STATUS_UNPAID
	case domain.StatusPaid:
		return paymentsv1.PaymentStatus_PAYMENT_STATUS_PAID
	case domain.StatusRefunded:
		return paymentsv1.PaymentStatus_PAYMENT_STATUS_REFUNDED
	default:
		return paymentsv1.PaymentStatus_PAYMENT_STATUS_UNSPECIFIED
	}
}

func fromProtoMoney(m *paymentsv1.Money) domain.Money {
	return domain.Money{
		Cents:    m.GetAmountCents(),
		Currency: m.GetCurrencyCode(),
	}
}

func toProto(p domain.Payment) *paymentsv1.Payment {
	return &paymentsv1.Payment{
		Id:          p.ID,
		PayableType: toProtoPayableType(p.PayableType),
		PayableId:   p.PayableID,
		Amount: &paymentsv1.Money{
			AmountCents:  p.Amount.Cents,
			CurrencyCode: p.Amount.Currency,
		},
		Method:           toProtoMethod(p.Method),
		Status:           toProtoStatus(p.Status),
		StripeReference:  p.StripeReference,
		RecordedByUserId: p.RecordedByUserID,
	}
}
