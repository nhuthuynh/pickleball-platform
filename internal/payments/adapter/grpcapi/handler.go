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
// Note what is NOT migrated here, deliberately, and what changed since
// (T12.8 vs. T16.2 vs. T17.1, kept all three dated so this comment stays
// accurate rather than rewritten into a single blended claim): as of T12.8,
// booking_host_id, game_host_id, entrant_player_id and the two
// assigned-admin lists all remained caller-supplied ownership *facts*,
// because Payments had no port into Booking, Social Play or Competitions to
// resolve them against — a real, pre-existing gap that ticket narrowed but
// did not close. T16.2 (closes #168) closes it for
// RecordOfflinePayment/RefundPayment's four fields: game_host_id,
// entrant_player_id, assigned_game_admin_user_ids and
// assigned_competition_admin_user_ids are no longer read off the wire at
// all for those two RPCs (see RecordOfflinePayment/RefundPayment below) —
// app.Service now RESOLVES the equivalent facts itself via
// port.RegistrationLookup/GameLookup/GameAdminReader/EntryLookup/
// CompetitionAdminReader against the real Social Play/Competitions stores,
// so a caller can no longer assert who a Game's Host or a Competition's
// admins are. T17.1 (closes #198) closes the identical gap on
// CreateOnlinePayment's own separate entrant_player_id/
// assigned_competition_admin_user_ids fields (see that method below) — T16.2
// built the resolver ports but scoped their wiring to
// RecordOfflinePayment/RefundPayment only, leaving CreateOnlinePayment's own
// authorization check trusting the caller one RPC longer. booking_host_id is
// the one fact that STAYS caller-supplied, on every RPC that accepts it:
// Payments still has no live join to Booking's database (ADR-0015's
// still-open D1 blocks the identical resolution built here for the other
// two contexts), so a caller can still assert who a Booking's Host is.
// Tracked as issue #149.
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
//
// T16.2 (closes #168): req.GetGameHostId(), req.GetAssignedGameAdminUserIds(),
// req.GetEntrantPlayerId() and req.GetAssignedCompetitionAdminUserIds() are
// deliberately NOT read here anymore — those four wire fields are now
// [deprecated = true] on the proto (see payments.proto) and app.
// RecordOfflinePaymentInput no longer even has fields to receive them into
// (deleted, not left unread — see that struct's doc comment). A caller
// still sending them (an un-migrated client) has those values silently
// ignored, exactly as actor_user_id already was as of T12.8: app.Service
// resolves the equivalent facts itself from PayableID via the real
// Social Play/Competitions stores. booking_host_id is the one field this
// handler still reads off the wire — see the actor func's doc comment for
// why.
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
		PayableType:   payableType,
		PayableID:     req.GetPayableId(),
		Amount:        fromProtoMoney(req.GetAmount()),
		ActorUserID:   actorUserID,
		BookingHostID: req.GetBookingHostId(),
	})
	if err != nil {
		return nil, toStatus(err)
	}

	return &paymentsv1.RecordOfflinePaymentResponse{Payment: toProto(p)}, nil
}

// CreateOnlinePayment creates a payment intent on behalf of the verified
// caller (T12.8). The actor is the principal, never req.GetActorUserId().
//
// T17.1 (closes #198): req.GetEntrantPlayerId() and
// req.GetAssignedCompetitionAdminUserIds() are deliberately NOT read here
// anymore — those two wire fields are now [deprecated = true] on the proto
// (see payments.proto) and app.CreateOnlinePaymentInput no longer even has
// fields to receive them into (deleted, not left unread — see that struct's
// doc comment). A caller still sending them (an un-migrated client) has
// those values silently ignored, exactly as RecordOfflinePayment's own four
// deprecated fields already are as of T16.2: app.Service resolves the
// equivalent facts itself from PayableID via the real Competitions store.
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
		PayableType: payableType,
		PayableID:   req.GetPayableId(),
		Amount:      fromProtoMoney(req.GetAmount()),
		ActorUserID: actorUserID,
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
//
// T13.7 (closes issue #148): this RPC now requires a verified principal and
// captures only on behalf of the actor the Payment records as its own. Until
// this ticket it was the single entry in PublicMethods() — a payment_id was the
// whole of the authorization, and a payment_id is not a secret (it is returned
// by CreateOnlinePayment and travels through client logs and URLs).
//
// Note the ordering: the actor is resolved first, so a caller with no principal
// is answered Unauthenticated before any lookup happens and cannot use this RPC
// to probe which payment ids exist.
func (h *Handler) ConfirmOnlinePayment(ctx context.Context, req *paymentsv1.ConfirmOnlinePaymentRequest) (*paymentsv1.ConfirmOnlinePaymentResponse, error) {
	actorUserID, err := actor(ctx)
	if err != nil {
		return nil, err
	}

	p, err := h.svc.GetPayment(ctx, req.GetPaymentId())
	if err != nil {
		return nil, toStatus(err)
	}

	// p is the *stored* Payment, and the actor is the *verified* principal —
	// neither side of the ownership comparison comes from the request body,
	// which carries only the payment_id.
	confirmed, err := h.svc.ConfirmOnlinePayment(ctx, p, actorUserID)
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
//
// T16.2 (closes #168): req.GetGameHostId() and
// req.GetAssignedGameAdminUserIds() are deliberately NOT read here anymore,
// for the identical reason RecordOfflinePayment's doc comment gives —
// app.RefundPaymentInput no longer has fields to receive them into, and
// app.Service resolves the equivalent facts itself. booking_host_id is
// still read off the wire; see the actor func's doc comment.
func (h *Handler) RefundPayment(ctx context.Context, req *paymentsv1.RefundPaymentRequest) (*paymentsv1.RefundPaymentResponse, error) {
	actorUserID, err := actor(ctx)
	if err != nil {
		return nil, err
	}

	p, err := h.svc.RefundPayment(ctx, app.RefundPaymentInput{
		PaymentID:     req.GetPaymentId(),
		ActorUserID:   actorUserID,
		BookingHostID: req.GetBookingHostId(),
	})
	if err != nil {
		return nil, toStatus(err)
	}

	return &paymentsv1.RefundPaymentResponse{Payment: toProto(p)}, nil
}

// ReceiveStripeWebhookEvent lets a successful Stripe charge complete its own
// Payment even if the client's own ConfirmOnlinePayment call never lands
// (T18.1, closes #167). Deliberately calls no actor(ctx): this RPC is
// PublicMethods()-listed (see that function's doc comment), authenticated
// by app.Service.HandleStripeWebhookEvent's own signature check rather than
// a verified principal — Stripe itself has no login on this platform to
// hold a token. See that method's own doc comment for the full ordering
// (signature verify -> idempotency claim -> event-type filter -> Payment
// resolution -> already-paid guard -> capture).
func (h *Handler) ReceiveStripeWebhookEvent(ctx context.Context, req *paymentsv1.ReceiveStripeWebhookEventRequest) (*paymentsv1.ReceiveStripeWebhookEventResponse, error) {
	err := h.svc.HandleStripeWebhookEvent(ctx, app.HandleStripeWebhookEventInput{
		RawPayload:      req.GetRawPayload(),
		SignatureHeader: req.GetSignatureHeader(),
		EventID:         req.GetEventId(),
		EventType:       req.GetEventType(),
		StripeReference: req.GetStripeReference(),
	})
	if err != nil {
		return nil, toStatus(err)
	}
	return &paymentsv1.ReceiveStripeWebhookEventResponse{}, nil
}

// toStatus maps domain errors to gRPC status codes. grpc-gateway then maps
// those codes onto HTTP statuses: AlreadyExists -> 409, PermissionDenied ->
// 403, NotFound -> 404, InvalidArgument -> 400, FailedPrecondition -> 400,
// Unavailable -> 503 — this is what makes the PR description's smoke-test AC
// true end-to-end (duplicate offline recording -> 409, actor mismatch ->
// 403-shaped, not 500).
//
// T13.9 (closes #131): this is the *only* error mapper in this package, and
// deliberately so. T12.3 added a second, RefundPayment-only mapper
// (toRefundStatus) that sent ErrIllegalStatusTransition to FailedPrecondition
// and ErrPaymentProcessorUnavailable/ErrPaymentDeclined to Internal, so the
// same domain sentinel produced a different code depending on which RPC
// surfaced it. That split is resolved here rather than perpetuated, because a
// sentinel's meaning does not depend on its caller:
//
//   - ErrIllegalStatusTransition now maps to FailedPrecondition service-wide
//     (RefundPayment's code won, not the majority's). gRPC defines
//     InvalidArgument as a problem with the argument *regardless of the state
//     of the system*; a status-machine violation is state-dependent by
//     definition, which is FailedPrecondition's own wording ("the system is
//     not in a state required for the operation's execution"). The one other
//     RPC that can surface it is ConfirmOnlinePayment; RecordOfflinePayment's
//     MarkPaid cannot fail, since domain.NewPayment always returns a Payment
//     in StatusUnpaid, and CreateOnlinePayment performs no transition at all.
//     Both codes map to HTTP 400, so no REST client sees a change.
//   - ErrPaymentProcessorUnavailable stays Unavailable service-wide
//     (toStatus's code won). The sentinel means one thing in both directions —
//     domain/errors.go defines it as the processor being unable to complete
//     the request — so in both the capture and refund directions nothing
//     moved and a retry is safe, which is precisely Unavailable's contract.
//     Internal describes broken invariants in *our* system; a third party's
//     outage is not that. T12.3's "escalate rather than retry" reasoning is
//     a client-side policy about consequences, not a statement about what
//     happened, and it does not survive the sentinel meaning the same thing
//     on both paths.
//   - ErrPaymentDeclined stays FailedPrecondition. toRefundStatus also sent
//     this one to Internal, undocumented in its own doc comment, and it was
//     unreachable there: port.PaymentProcessor documents declines on
//     CapturePayment only, and RefundPayment's contract (and its sole
//     implementation) never returns it. Removing the branch changes no
//     observable behaviour.
func toStatus(err error) error {
	switch {
	case errors.Is(err, domain.ErrPaymentAlreadyRecorded):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, domain.ErrNotPaymentRecorder),
		// T13.7 (#148): a verified principal who is not the Payment's own
		// recorded actor. PermissionDenied for the same reason
		// ErrNotPaymentRecorder gets it and ADR-0014 §6 restates: the token
		// verified, so this is not Unauthenticated, and answering NotFound
		// would turn the RPC into a payment-id enumeration oracle. Two
		// sentinels, one code — deliberately, per the doc comment above:
		// they mean different things to us and the same thing to a client.
		errors.Is(err, domain.ErrNotPaymentOwner):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, domain.ErrWebhookSignatureInvalid):
		// T18.1 (closes #167): a forged or malformed Stripe webhook
		// signature. Also PermissionDenied, but for a different reason
		// than the two sentinels above — there is no principal at all on
		// this RPC (it is PublicMethods()-listed), so this is not "a
		// verified caller failed an object-level check" but "the one
		// authentication mechanism this RPC has (the signature) did not
		// check out". Both answer PermissionDenied on the wire because
		// both mean "this caller is refused" — the same "different
		// sentinels, same code" reasoning the doc comment above states for
		// ErrNotPaymentRecorder/ErrNotPaymentOwner.
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, domain.ErrPaymentNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrPaymentDeclined),
		// A decline is a legitimate business outcome, not a client input
		// error or a server bug (see port.PaymentProcessor's doc
		// comment) — FailedPrecondition (-> HTTP 400 via grpc-gateway)
		// distinguishes it from both InvalidArgument-shaped bad input and
		// an actual Internal server error.
		//
		// ErrIllegalStatusTransition joins it here in T13.9 (#131): a
		// well-formed request against a system state that forbids it is
		// the definition of FailedPrecondition, and InvalidArgument is
		// explicitly reserved for arguments that are wrong regardless of
		// system state. See the doc comment above for the full reasoning
		// and the blast radius.
		errors.Is(err, domain.ErrIllegalStatusTransition):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, domain.ErrPaymentProcessorUnavailable):
		return status.Error(codes.Unavailable, err.Error())
	case errors.Is(err, domain.ErrEmptyPayableID),
		errors.Is(err, domain.ErrInvalidPayableType),
		errors.Is(err, domain.ErrInvalidAmount),
		errors.Is(err, domain.ErrInvalidCurrency):
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
