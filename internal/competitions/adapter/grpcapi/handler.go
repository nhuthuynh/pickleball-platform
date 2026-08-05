// Package grpcapi is the Competitions context's gRPC adapter. It translates
// between the wire contract generated from proto/pickleball/competitions/v1
// and the app/domain layers, and maps domain errors onto gRPC status codes
// so grpc-gateway's REST mapping produces the right HTTP status — mirrors
// internal/socialplay/adapter/grpcapi's shape. It only compiles after `make
// generate` has produced internal/gen/pickleball/competitions/v1 (see
// CLAUDE.md gotchas).
package grpcapi

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/nhuthuynh/white-label/internal/competitions/app"
	"github.com/nhuthuynh/white-label/internal/competitions/domain"
	"github.com/nhuthuynh/white-label/internal/competitions/port"

	competitionsv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/competitions/v1"
)

// Handler serves CompetitionsService.
//
// It holds only the app.Service — unlike socialplay's Handler, which also
// carries CourtReservation and FacilityLookup because ScheduleGame takes
// them per-call. Competitions' app.Service stores every dependency on itself
// (see app.ServiceOptions), so there is nothing for this adapter to thread
// through, and the handler stays a pure translation layer.
type Handler struct {
	competitionsv1.UnimplementedCompetitionsServiceServer
	svc *app.Service
}

func NewHandler(svc *app.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) CreateCompetition(ctx context.Context, req *competitionsv1.CreateCompetitionRequest) (*competitionsv1.CreateCompetitionResponse, error) {
	sessions, err := fromProtoSessions(req.GetSessions())
	if err != nil {
		// A malformed session range is a bad request, not a server fault.
		// Converted here rather than in the app layer because an invalid
		// wire value never becomes a domain type in the first place.
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	competition, err := h.svc.ScheduleCompetition(ctx, app.ScheduleCompetitionInput{
		HostID:          req.GetHostId(),
		Name:            req.GetName(),
		VenueFacilityID: req.GetVenueFacilityId(),
		Sessions:        sessions,
		Capacity:        int(req.GetCapacity()),
		GuestAllowance:  int(req.GetGuestAllowance()),
		PaymentMethod:   fromProtoPaymentMethod(req.GetPaymentMethod()),
		EntryFee:        fromProtoMoney(req.GetEntryFee()),
		Format:          fromProtoFormat(req.GetFormat()),
	})
	if err != nil {
		return nil, toStatus(err)
	}

	// share_token is returned on this response only — the one moment the
	// caller is provably the Host. It is deliberately absent from the
	// Competition message so it can't leak through the unauthenticated
	// Get/List reads; see the proto's own doc comments.
	return &competitionsv1.CreateCompetitionResponse{
		Competition: toProtoCompetition(competition),
		ShareToken:  competition.ShareToken,
	}, nil
}

func (h *Handler) GetCompetition(ctx context.Context, req *competitionsv1.GetCompetitionRequest) (*competitionsv1.GetCompetitionResponse, error) {
	competition, err := h.svc.GetCompetition(ctx, req.GetCompetitionId())
	if err != nil {
		return nil, toStatus(err)
	}
	return &competitionsv1.GetCompetitionResponse{Competition: toProtoCompetition(competition)}, nil
}

// ListCompetitions serves the browse/filter read. starts_after/starts_before
// are only converted to a real time.Time when actually present on the wire
// (protobuf Timestamp field presence — GetStartsAfter() returns nil for an
// unset field); otherwise the zero value flows through, which
// port.CompetitionListingFilter and the adapter's nullableTimestamptz
// already treat as "no bound on this side".
func (h *Handler) ListCompetitions(ctx context.Context, req *competitionsv1.ListCompetitionsRequest) (*competitionsv1.ListCompetitionsResponse, error) {
	filter := port.CompetitionListingFilter{VenueFacilityID: req.GetVenueFacilityId()}
	if req.GetStartsAfter() != nil {
		filter.StartsAfter = req.GetStartsAfter().AsTime()
	}
	if req.GetStartsBefore() != nil {
		filter.StartsBefore = req.GetStartsBefore().AsTime()
	}

	listings, err := h.svc.ListCompetitions(ctx, filter)
	if err != nil {
		return nil, toStatus(err)
	}

	out := make([]*competitionsv1.CompetitionListing, 0, len(listings))
	for _, l := range listings {
		out = append(out, toProtoCompetitionListing(l))
	}
	return &competitionsv1.ListCompetitionsResponse{Competitions: out}, nil
}

func (h *Handler) EnterCompetition(ctx context.Context, req *competitionsv1.EnterCompetitionRequest) (*competitionsv1.EnterCompetitionResponse, error) {
	entry, err := h.svc.EnterCompetition(ctx, app.EnterCompetitionInput{
		CompetitionID: req.GetCompetitionId(),
		PlayerID:      req.GetPlayerId(),
		GuestCount:    int(req.GetGuestCount()),
		Source:        fromProtoEntrySource(req.GetSource()),
	})
	if err != nil {
		return nil, toStatus(err)
	}
	return &competitionsv1.EnterCompetitionResponse{Entry: toProtoEntry(entry)}, nil
}

func (h *Handler) CancelCompetition(ctx context.Context, req *competitionsv1.CancelCompetitionRequest) (*competitionsv1.CancelCompetitionResponse, error) {
	competition, err := h.svc.CancelCompetition(ctx, req.GetCompetitionId(), req.GetActorUserId())
	if err != nil {
		return nil, toStatus(err)
	}
	return &competitionsv1.CancelCompetitionResponse{Competition: toProtoCompetition(competition)}, nil
}

func (h *Handler) ListEntriesForCompetition(ctx context.Context, req *competitionsv1.ListEntriesForCompetitionRequest) (*competitionsv1.ListEntriesForCompetitionResponse, error) {
	entries, err := h.svc.ListEntriesForCompetition(ctx, req.GetCompetitionId())
	if err != nil {
		return nil, toStatus(err)
	}

	out := make([]*competitionsv1.CompetitionEntry, 0, len(entries))
	for _, e := range entries {
		out = append(out, toProtoEntry(e))
	}
	return &competitionsv1.ListEntriesForCompetitionResponse{Entries: out}, nil
}

// toStatus maps domain errors onto gRPC status codes; grpc-gateway then maps
// those onto HTTP statuses. Mirrors internal/socialplay/adapter/grpcapi's
// toStatus, with the four mappings T9.4 requires by name — none of which may
// ever fall through to Internal:
//
//   - ErrCompetitionFull -> FailedPrecondition. This is Competitions' one
//     deliberate DIVERGENCE from Social Play, which maps its equivalent
//     ErrGameFull to AlreadyExists. T9.4 specifies FailedPrecondition by
//     name, and it is the better-fitting gRPC code on its merits:
//     AlreadyExists says "the thing you tried to create is already there",
//     which is true of a duplicate entry but not of a full Competition —
//     nothing about the caller's own entry already exists. The Competition
//     simply isn't in a state that permits the call, which is precisely what
//     FailedPrecondition means.
//
//     ⚠ HTTP SHAPE, VERIFIED RATHER THAN ASSUMED: T9.4's ticket text
//     describes this mapping as "409-shaped". It is not. grpc-gateway's
//     default HTTPStatusFromCode maps FailedPrecondition to **400 Bad
//     Request**, not 409 Conflict — confirmed empirically against the real
//     running gateway during this ticket (a full Competition returns
//     HTTP 400 with code 9). Only AlreadyExists yields 409, which is why
//     Social Play's ErrGameFull produces one and this does not.
//
//     The explicitly-named gRPC code is implemented as specified, since that
//     is the unambiguous instruction (and the same mapping
//     internal/competitions/domain/errors.go already documents). The
//     "409-shaped" parenthetical is recorded here as a disclosed
//     discrepancy for review rather than silently satisfied by substituting
//     AlreadyExists — see the T9.4 PR description. If the intended
//     requirement is genuinely the HTTP status rather than the gRPC code,
//     the fix is one line (move ErrCompetitionFull into the AlreadyExists
//     group) plus updating errors.go's comment, and should be a deliberate
//     decision rather than this adapter's own guess.
//   - ErrFacilityNotFound -> NotFound (404-shaped): an unknown
//     venue_facility_id is a bad reference, never a 500 and never a silent
//     accept.
//   - ErrNotCompetitionHost -> PermissionDenied (403-shaped): the
//     object-level (BOLA) rejection. Regression-tested end to end in
//     authz_regression_test.go, which asserts specifically that this is NOT
//     Internal and NOT a silent success.
//   - ErrGuestAllowanceExceeded -> InvalidArgument (400-shaped): asking for
//     more guests than the Host permits is a malformed request against a
//     fixed, knowable limit, not a conflict over contended capacity. Keeping
//     it distinct from ErrCompetitionFull is what lets a client tell "you
//     may bring at most 2 guests" apart from "this competition is full" —
//     the distinction domain.Enter deliberately checks in that order.
//
// The remaining groups follow the same reasoning the other contexts use:
// not-found sentinels to NotFound, request-shape violations to
// InvalidArgument, and ErrCourtUnavailable to AlreadyExists (matching
// Booking's own ErrCourtDoubleBooked, since that IS a genuine "already
// taken" conflict).
//
// ErrCompetitionCancelled maps to FailedPrecondition alongside
// ErrCompetitionFull: both mean "this Competition is not in a state that
// accepts entries", which is the same category of answer.
func toStatus(err error) error {
	switch {
	case errors.Is(err, domain.ErrCompetitionFull),
		errors.Is(err, domain.ErrCompetitionCancelled):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, domain.ErrNotCompetitionHost):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, domain.ErrCompetitionNotFound),
		errors.Is(err, domain.ErrFacilityNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrAlreadyEntered),
		errors.Is(err, domain.ErrCourtUnavailable):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, domain.ErrGuestAllowanceExceeded),
		errors.Is(err, domain.ErrInvalidTimeRange),
		errors.Is(err, domain.ErrInvalidCapacity),
		errors.Is(err, domain.ErrEmptySessions),
		errors.Is(err, domain.ErrEmptyCourtIDs),
		errors.Is(err, domain.ErrEmptyPlayerID),
		errors.Is(err, domain.ErrInvalidPaymentMethod),
		errors.Is(err, domain.ErrInvalidFormat),
		errors.Is(err, domain.ErrInvalidGuestAllowance),
		errors.Is(err, domain.ErrInvalidEntrySource),
		errors.Is(err, domain.ErrInvalidMoney),
		errors.Is(err, domain.ErrOverlappingSessions),
		errors.Is(err, domain.ErrIllegalStatusTransition):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}

// --- wire -> domain --------------------------------------------------------

// fromProtoSessions converts the wire's session list into domain.Sessions,
// validating each range through domain.NewTimeRange so a zero-duration or
// inverted range is rejected at the boundary with a clean InvalidArgument
// rather than reaching the domain as a malformed value.
//
// An empty list is passed through rather than rejected here, so that
// domain.NewCompetition's own ErrEmptySessions is the single place that rule
// lives (and the single error message clients see for it).
func fromProtoSessions(in []*competitionsv1.CompetitionSession) ([]domain.Session, error) {
	out := make([]domain.Session, 0, len(in))
	for _, s := range in {
		r, err := domain.NewTimeRange(s.GetStartsAt().AsTime(), s.GetEndsAt().AsTime())
		if err != nil {
			return nil, err
		}
		out = append(out, domain.Session{Range: r, CourtIDs: s.GetCourtIds()})
	}
	return out, nil
}

// fromProtoPaymentMethod resolves the wire zero value
// (PAYMENT_METHOD_UNSPECIFIED — sent by any client that never sets the
// field) to domain.PaymentMethodEither, the least restrictive value, so the
// field stays genuinely optional.
//
// Any OTHER unrecognized value (e.g. from a future proto version this build
// doesn't know) is deliberately NOT folded into the same default: it is
// passed through as an invalid domain.PaymentMethod so
// domain.NewCompetition's IsValid() check rejects it, rather than silently
// accepting an unknown value as if it meant "either". UNSPECIFIED and
// "unrecognized" are not the same thing and must not share a fallback — a PE
// review finding on T8.7's original PR, inherited here rather than
// rediscovered.
func fromProtoPaymentMethod(m competitionsv1.PaymentMethod) domain.PaymentMethod {
	switch m {
	case competitionsv1.PaymentMethod_PAYMENT_METHOD_UNSPECIFIED:
		return domain.PaymentMethodEither
	case competitionsv1.PaymentMethod_PAYMENT_METHOD_ONLINE:
		return domain.PaymentMethodOnline
	case competitionsv1.PaymentMethod_PAYMENT_METHOD_CASH:
		return domain.PaymentMethodCash
	case competitionsv1.PaymentMethod_PAYMENT_METHOD_EITHER:
		return domain.PaymentMethodEither
	default:
		return domain.PaymentMethod("")
	}
}

// fromProtoFormat converts the wire format enum.
//
// Unlike payment method, COMPETITION_FORMAT_UNSPECIFIED is NOT resolved to a
// default: there is no "least restrictive" format the way "either" is the
// least restrictive payment method, and silently picking singles or doubles
// on the Host's behalf would put a label on their Competition that they
// never chose — one players would read as fact. It is passed through as an
// invalid domain.Format so NewCompetition rejects it with
// ErrInvalidFormat/400, making the client state its choice.
func fromProtoFormat(f competitionsv1.CompetitionFormat) domain.Format {
	switch f {
	case competitionsv1.CompetitionFormat_COMPETITION_FORMAT_SINGLES:
		return domain.FormatSingles
	case competitionsv1.CompetitionFormat_COMPETITION_FORMAT_DOUBLES:
		return domain.FormatDoubles
	default:
		return domain.Format("")
	}
}

// fromProtoEntrySource resolves the wire zero value
// (ENTRY_SOURCE_UNSPECIFIED) to domain.EntrySourceApp — the in-app default,
// which is what a client that doesn't set the field is by definition doing —
// while passing any other unrecognized value through as invalid so
// domain.Enter rejects it (ErrInvalidEntrySource/400). Same
// UNSPECIFIED-is-not-unrecognized discipline as fromProtoPaymentMethod.
func fromProtoEntrySource(s competitionsv1.EntrySource) domain.EntrySource {
	switch s {
	case competitionsv1.EntrySource_ENTRY_SOURCE_UNSPECIFIED:
		return domain.EntrySourceApp
	case competitionsv1.EntrySource_ENTRY_SOURCE_APP:
		return domain.EntrySourceApp
	case competitionsv1.EntrySource_ENTRY_SOURCE_SOCIAL:
		return domain.EntrySourceSocial
	default:
		return domain.EntrySource("")
	}
}

// fromProtoMoney maps an ABSENT entry_fee (a nil *Money — an older client,
// or a field left unset) onto the zero domain.Money, which means a FREE
// Competition. That is not an "unspecified" resolution step: zero is a real,
// well-formed value here, so there is nothing to resolve.
func fromProtoMoney(m *competitionsv1.Money) domain.Money {
	if m == nil {
		return domain.Money{}
	}
	return domain.Money{
		AmountCents:  m.GetAmountCents(),
		CurrencyCode: m.GetCurrencyCode(),
	}
}

// --- domain -> wire --------------------------------------------------------

func toProtoCompetitionStatus(s domain.Status) competitionsv1.CompetitionStatus {
	switch s {
	case domain.StatusScheduled:
		return competitionsv1.CompetitionStatus_COMPETITION_STATUS_SCHEDULED
	case domain.StatusCancelled:
		return competitionsv1.CompetitionStatus_COMPETITION_STATUS_CANCELLED
	default:
		return competitionsv1.CompetitionStatus_COMPETITION_STATUS_UNSPECIFIED
	}
}

func toProtoFormat(f domain.Format) competitionsv1.CompetitionFormat {
	switch f {
	case domain.FormatSingles:
		return competitionsv1.CompetitionFormat_COMPETITION_FORMAT_SINGLES
	case domain.FormatDoubles:
		return competitionsv1.CompetitionFormat_COMPETITION_FORMAT_DOUBLES
	default:
		return competitionsv1.CompetitionFormat_COMPETITION_FORMAT_UNSPECIFIED
	}
}

func toProtoPaymentMethod(m domain.PaymentMethod) competitionsv1.PaymentMethod {
	switch m {
	case domain.PaymentMethodOnline:
		return competitionsv1.PaymentMethod_PAYMENT_METHOD_ONLINE
	case domain.PaymentMethodCash:
		return competitionsv1.PaymentMethod_PAYMENT_METHOD_CASH
	case domain.PaymentMethodEither:
		return competitionsv1.PaymentMethod_PAYMENT_METHOD_EITHER
	default:
		return competitionsv1.PaymentMethod_PAYMENT_METHOD_UNSPECIFIED
	}
}

func toProtoEntrySource(s domain.EntrySource) competitionsv1.EntrySource {
	switch s {
	case domain.EntrySourceApp:
		return competitionsv1.EntrySource_ENTRY_SOURCE_APP
	case domain.EntrySourceSocial:
		return competitionsv1.EntrySource_ENTRY_SOURCE_SOCIAL
	default:
		return competitionsv1.EntrySource_ENTRY_SOURCE_UNSPECIFIED
	}
}

func toProtoEntryStatus(s domain.EntryStatus) competitionsv1.EntryStatus {
	switch s {
	case domain.EntryStatusEntered:
		return competitionsv1.EntryStatus_ENTRY_STATUS_ENTERED
	case domain.EntryStatusCancelled:
		return competitionsv1.EntryStatus_ENTRY_STATUS_CANCELLED
	default:
		return competitionsv1.EntryStatus_ENTRY_STATUS_UNSPECIFIED
	}
}

func toProtoPaymentStatus(s domain.PaymentStatus) competitionsv1.PaymentStatus {
	switch s {
	case domain.PaymentStatusUnpaid:
		return competitionsv1.PaymentStatus_PAYMENT_STATUS_UNPAID
	case domain.PaymentStatusPaid:
		return competitionsv1.PaymentStatus_PAYMENT_STATUS_PAID
	case domain.PaymentStatusRefunded:
		return competitionsv1.PaymentStatus_PAYMENT_STATUS_REFUNDED
	default:
		return competitionsv1.PaymentStatus_PAYMENT_STATUS_UNSPECIFIED
	}
}

// toProtoMoney always emits the message, including for a free Competition,
// so a client can distinguish "this Competition is free" (amount_cents 0)
// from a field the server never populated — clients render the former as the
// word "Free", never as blank or a bare "$0.00".
func toProtoMoney(m domain.Money) *competitionsv1.Money {
	return &competitionsv1.Money{
		AmountCents:  m.AmountCents,
		CurrencyCode: m.CurrencyCode,
	}
}

func toProtoSession(s domain.Session) *competitionsv1.CompetitionSession {
	return &competitionsv1.CompetitionSession{
		StartsAt: timestamppb.New(s.Range.Start),
		EndsAt:   timestamppb.New(s.Range.End),
		CourtIds: s.CourtIDs,
	}
}

// toProtoCompetition deliberately does NOT emit ShareToken — the wire
// Competition message has no such field, precisely so this function cannot
// leak it through GetCompetition or ListCompetitions. See the proto's
// Competition doc comment.
func toProtoCompetition(c domain.Competition) *competitionsv1.Competition {
	sessions := make([]*competitionsv1.CompetitionSession, 0, len(c.Sessions))
	for _, s := range c.Sessions {
		sessions = append(sessions, toProtoSession(s))
	}
	return &competitionsv1.Competition{
		Id:              c.ID,
		HostId:          c.HostID,
		Name:            c.Name,
		VenueFacilityId: c.VenueFacilityID,
		Sessions:        sessions,
		Capacity:        int32(c.Capacity),
		GuestAllowance:  int32(c.GuestAllowance),
		PaymentMethod:   toProtoPaymentMethod(c.PaymentMethod),
		EntryFee:        toProtoMoney(c.EntryFee),
		Format:          toProtoFormat(c.Format),
		Status:          toProtoCompetitionStatus(c.Status),
	}
}

// toProtoCompetitionListing converts a port.CompetitionListing to its wire
// message. SpotsLeft is already the weighted value computed server-side —
// see the proto's CompetitionListing doc comment for why the weighting
// matters and internal/competitions/domain's TestSpotsLeft for the boundary
// proof.
func toProtoCompetitionListing(l port.CompetitionListing) *competitionsv1.CompetitionListing {
	return &competitionsv1.CompetitionListing{
		Competition: toProtoCompetition(l.Competition),
		SpotsLeft:   int32(l.SpotsLeft),
	}
}

func toProtoEntry(e domain.CompetitionEntry) *competitionsv1.CompetitionEntry {
	return &competitionsv1.CompetitionEntry{
		Id:            e.ID,
		CompetitionId: e.CompetitionID,
		PlayerId:      e.PlayerID,
		GuestCount:    int32(e.GuestCount),
		Source:        toProtoEntrySource(e.Source),
		PaymentStatus: toProtoPaymentStatus(e.PaymentStatus),
		Status:        toProtoEntryStatus(e.Status),
	}
}
