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
	"github.com/nhuthuynh/white-label/internal/platform/auth"

	paymentsv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/payments/v1"
)

// actor resolves the acting user for an authenticated RPC (T12.8).
//
// This one function is the whole of A11 Ruling 3 for this context: the
// Principal is translated into the plain actor string app.Service and the
// domain's authorization branches already take, here at the grpcapi boundary,
// so internal/payments/{domain,app} keep their existing signatures and never
// import internal/platform/auth.
//
// It takes only a context, deliberately. The request's actor_user_id is not
// passed in and cannot be consulted, so there is no fallback to the caller's
// claim when no principal is present — the failure mode the ticket calls out
// as "a handler that falls back to the claimed value has changed nothing".
// Missing principal is codes.Unauthenticated ("I do not know who you are"),
// never PermissionDenied (ADR-0013 §5).
//
// Note what is NOT migrated here, deliberately: booking_host_id, game_host_id,
// entrant_player_id and the two assigned-admin lists remain caller-supplied
// ownership *facts*, because Payments has no port into Booking, Social Play or
// Competitions to resolve them against. That is a real, pre-existing gap this
// ticket narrows but does not close — verifying the actor stops a caller
// impersonating the Host, but a caller can still assert that some other Game's
// Host is whoever they like. Closing it means new cross-context ports, which
// is a structural change well outside a handler-boundary ticket. Disclosed in
// the PR body with a tracked issue, per the sprint's A5 rule.
func actor(ctx context.Context) (string, error) {
	return auth.RequireSubject(ctx)
}

type Handler struct {
	paymentsv1.UnimplementedPaymentsServiceServer
	svc *app.Service
}

func NewHandler(svc *app.Service) *Handler {
	return &Handler{svc: svc}
}

// RecordOfflinePayment marks a payable paid on behalf of the verified caller
// (T12.8). The actor is the principal, never req.GetActorUserId() — and since
// this RPC also *writes* that actor into Payment.RecordedByUserId, honoring
// the wire field would additionally let a caller record a payment under
// someone else's name in the audit trail.
func (h *Handler) RecordOfflinePayment(ctx context.Context, req *paymentsv1.RecordOfflinePaymentRequest) (*paymentsv1.RecordOfflinePaymentResponse, error) {
	actorUserID, err := actor(ctx)
	if err != nil {
		return nil, err
	}

	payableType, err := fromProtoPayableType(req.GetPayableType())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	p, err := h.svc.RecordOfflinePayment(ctx, app.RecordOfflinePaymentInput{
		PayableType:                     payableType,
		PayableID:                       req.GetPayableId(),
		Amount:                          fromProtoMoney(req.GetAmount()),
		ActorUserID:                     actorUserID,
		BookingHostID:                   req.GetBookingHostId(),
		GameHostID:                      req.GetGameHostId(),
		AssignedGameAdminUserIDs:        req.GetAssignedGameAdminUserIds(),
		EntrantPlayerID:                 req.GetEntrantPlayerId(),
		AssignedCompetitionAdminUserIDs: req.GetAssignedCompetitionAdminUserIds(),
	})
	if err != nil {
		return nil, toStatus(err)
	}

	return &paymentsv1.RecordOfflinePaymentResponse{Payment: toProto(p)}, nil
}

// CreateOnlinePayment creates a payment intent on behalf of the verified
// caller (T12.8). The actor is the principal, never req.GetActorUserId().
func (h *Handler) CreateOnlinePayment(ctx context.Context, req *paymentsv1.CreateOnlinePaymentRequest) (*paymentsv1.CreateOnlinePaymentResponse, error) {
	actorUserID, err := actor(ctx)
	if err != nil {
		return nil, err
	}

	payableType, err := fromProtoPayableType(req.GetPayableType())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	p, err := h.svc.CreateOnlinePayment(ctx, app.CreateOnlinePaymentInput{
		PayableType:                     payableType,
		PayableID:                       req.GetPayableId(),
		Amount:                          fromProtoMoney(req.GetAmount()),
		ActorUserID:                     actorUserID,
		EntrantPlayerID:                 req.GetEntrantPlayerId(),
		AssignedCompetitionAdminUserIDs: req.GetAssignedCompetitionAdminUserIds(),
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
//
// T10.7 (closing issue #97): this used to call h.svc.Payments().GetByID
// directly, bypassing app.Service's boundary validation entirely — the one
// caller-supplied-id lookup in this codebase that reached a repository
// straight from a gRPC adapter. Now goes through app.Service.GetPayment,
// which applies the same uuidShape guard every other context's get-shaped
// read already has, so a malformed payment_id answers with the same
// domain.ErrPaymentNotFound an unknown one gets, instead of reaching the
// Postgres adapter's mustUUID and panicking.
func (h *Handler) ConfirmOnlinePayment(ctx context.Context, req *paymentsv1.ConfirmOnlinePaymentRequest) (*paymentsv1.ConfirmOnlinePaymentResponse, error) {
	p, err := h.svc.GetPayment(ctx, req.GetPaymentId())
	if err != nil {
		return nil, toStatus(err)
	}

	confirmed, err := h.svc.ConfirmOnlinePayment(ctx, p)
	if err != nil {
		return nil, toStatus(err)
	}

	return &paymentsv1.ConfirmOnlinePaymentResponse{Payment: toProto(confirmed)}, nil
}

// RefundPayment refunds an already-paid Payment (T12.3, closing the gap
// open since T6.5). The app layer does the work — this handler only
// translates the wire request into app.RefundPaymentInput and the resulting
// domain error into a gRPC status.
//
// Unlike ConfirmOnlinePayment above, this handler does not do its own
// id -> Payment lookup: RefundPayment takes the id and resolves the Payment
// itself, because it needs the *stored* payable type to pick an
// authorization branch and must not accept that fact from the caller (see
// app.RefundPaymentInput's doc comment).
// T12.8: the actor is the verified principal, not req.GetActorUserId(). This
// is the RPC that moves real money back out, so the care T12.3 took to read
// the payable type from the *stored* Payment — denying the caller any say in
// which authorization branch judges them — was only as good as the actor that
// branch compares, which until now the caller also supplied.
func (h *Handler) RefundPayment(ctx context.Context, req *paymentsv1.RefundPaymentRequest) (*paymentsv1.RefundPaymentResponse, error) {
	actorUserID, err := actor(ctx)
	if err != nil {
		return nil, err
	}

	p, err := h.svc.RefundPayment(ctx, app.RefundPaymentInput{
		PaymentID:                req.GetPaymentId(),
		ActorUserID:              actorUserID,
		BookingHostID:            req.GetBookingHostId(),
		GameHostID:               req.GetGameHostId(),
		AssignedGameAdminUserIDs: req.GetAssignedGameAdminUserIds(),
	})
	if err != nil {
		return nil, toRefundStatus(err)
	}

	return &paymentsv1.RefundPaymentResponse{Payment: toProto(p)}, nil
}

// toRefundStatus is RefundPayment's error mapping. It handles the two codes
// T12.3 specifies differently from the shared toStatus below, then delegates
// everything else to it so there is exactly one place each remaining
// sentinel is mapped.
//
// The two deliberate divergences, and why they are scoped to this RPC:
//
//   - ErrIllegalStatusTransition -> FailedPrecondition (toStatus sends it to
//     InvalidArgument). FailedPrecondition is the semantically correct code:
//     an already-refunded or never-paid Payment is a well-formed request
//     against a system state that forbids it, not a malformed argument.
//   - ErrPaymentProcessorUnavailable -> Internal (toStatus sends it to
//     Unavailable). Specified by the ticket: a refund that failed at the
//     processor left real money unreturned, and the ticket treats that as a
//     server-side failure the caller should escalate rather than a
//     retry-and-it-might-work condition.
//
// Both are scoped to RefundPayment rather than changed in toStatus because
// changing toStatus would silently alter the wire contract of
// RecordOfflinePayment/CreateOnlinePayment/ConfirmOnlinePayment, which no
// ticket asked for and existing clients may depend on (PE dossier §2,
// Hyrum's Law). The resulting inconsistency — one sentinel, two codes,
// depending on the RPC — is real and is tracked as issue #131 rather than
// left undisclosed.
func toRefundStatus(err error) error {
	switch {
	case errors.Is(err, domain.ErrIllegalStatusTransition):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, domain.ErrPaymentProcessorUnavailable),
		errors.Is(err, domain.ErrPaymentDeclined):
		return status.Error(codes.Internal, err.Error())
	default:
		return toStatus(err)
	}
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
	case paymentsv1.PayableType_PAYABLE_TYPE_COMPETITION_ENTRY:
		return domain.PayableTypeCompetitionEntry, nil
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
	case domain.PayableTypeCompetitionEntry:
		return paymentsv1.PayableType_PAYABLE_TYPE_COMPETITION_ENTRY
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
