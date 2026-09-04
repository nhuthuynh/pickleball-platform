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
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/nhuthuynh/white-label/internal/booking/app"
	"github.com/nhuthuynh/white-label/internal/booking/domain"
	bookingv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/booking/v1"
	"github.com/nhuthuynh/white-label/internal/platform/auth"
)

// actor resolves the acting user for an authenticated RPC (T12.7), and since
// T13.2 it is also ADR-0014's translation seam for this context.
//
// This one method is the whole of A11 Ruling 3 for Booking: the Principal is
// translated into the plain actor string app.Service already takes, here at
// the grpcapi boundary, so internal/booking/{domain,app} keep their existing
// signatures and never import internal/platform/auth.
//
// **It returns a User.ID (uuid), not the subject.** That is ADR-0014's ruling
// and the fix for issues #146 and #152. Two steps, in this order:
//
//  1. auth.RequireSubject — who is calling? A missing or unverified principal
//     is codes.Unauthenticated ("I do not know who you are"), never
//     PermissionDenied (ADR-0013 §5).
//  2. app.Service.ResolveActorUserID — which User is that? A subject
//     registered to no User is codes.PermissionDenied, never NotFound
//     (ADR-0014 §6: the caller is known and simply may not act, and NotFound
//     would make this an enumeration oracle).
//
// **Being the single funnel is the point, not an implementation detail.**
// Every actor-taking RPC on this service calls it — six call sites — so the
// translation happens once and cannot be half-applied. A handler has no other
// way to obtain an actor: the request's actor_user_id field is deprecated and
// never read, so nothing can fall back to the caller's claim, which is the
// failure T12.7 calls out as "a handler that falls back to the claimed value
// has changed nothing". Resolving inside each app method instead would have
// to be remembered once per method, which is the shape that produced #146.
//
// It is a method rather than a package function only because it needs the
// service to reach port.IdentityLookup. NewHandler's signature is unchanged:
// the service already held that port, so this adds no dependency to wire.
func (h *Handler) actor(ctx context.Context) (string, error) {
	subject, err := auth.RequireSubject(ctx)
	if err != nil {
		return "", err
	}

	actorUserID, err := h.svc.ResolveActorUserID(ctx, subject)
	if err != nil {
		return "", toStatus(err)
	}
	return actorUserID, nil
}

type Handler struct {
	bookingv1.UnimplementedBookingServiceServer
	svc *app.Service
}

func NewHandler(svc *app.Service) *Handler {
	return &Handler{svc: svc}
}

// CreateBooking mints an individual court Booking owned by the verified
// caller.
//
// DECISION D1 (ADR-0015 option (a); closes #144) made this an authenticated
// RPC. The owner is resolved from the token via h.actor and is NOT read off
// the request — CreateBookingRequest carries no owner field, deliberately,
// because a caller able to name the owner could mint bookings in somebody
// else's name, which is the same class of hole D1 was raised to close.
//
// This is the change that breaks the shipped T7.6 public quote-and-book
// flow: a player must now hold an account to book. That cost was stated in
// ADR-0015 option (a) and accepted by the Product Owner when D1 was
// answered; GetQuote and ListCourtBookings stay public, so the browse half
// of that flow (see a price, see availability) still works without a token
// and only the final confirm step now requires one.
func (h *Handler) CreateBooking(ctx context.Context, req *bookingv1.CreateBookingRequest) (*bookingv1.CreateBookingResponse, error) {
	actorUserID, err := h.actor(ctx)
	if err != nil {
		return nil, err
	}

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
		OwnerUserID: actorUserID,
	})
	if err != nil {
		return nil, toStatus(err)
	}

	return &bookingv1.CreateBookingResponse{Booking: toProto(b)}, nil
}

func (h *Handler) GetQuote(ctx context.Context, req *bookingv1.GetQuoteRequest) (*bookingv1.GetQuoteResponse, error) {
	rng, err := domain.NewTimeRange(req.GetStartsAt().AsTime(), req.GetEndsAt().AsTime())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	quote, err := h.svc.GetQuote(ctx, app.GetQuoteInput{
		CourtID: req.GetCourtId(),
		Source:  quoteSource(req.GetSource()),
		Range:   rng,
	})
	if err != nil {
		return nil, toStatus(err)
	}

	resp := &bookingv1.GetQuoteResponse{
		PriceCents:     quote.PriceCents,
		Band:           string(quote.Band),
		BandPriceCents: quote.BandPriceCents,
	}
	// Absence, not a zero-valued rule, is what "no discount" looks like on
	// the wire — a 0%-off discount and no discount are different statements,
	// and a client reading `discount != null` must be reading the first.
	if quote.Discount.Applies() {
		resp.Discount = toProtoDiscountRule(quote.Discount)
	}
	return resp, nil
}

// quoteSource applies GetQuoteRequest.source's documented default: an
// unspecified Source is SOURCE_INDIVIDUAL. See the field's own comment in
// booking.proto for why that default is part of the contract rather than a
// guess — briefly: GetQuote is the court-hire quote endpoint, every Booking
// created from a quote is created with SOURCE_INDIVIDUAL, and the endpoint
// has shipped clients that predate this field. Unlike CreateBooking's
// fromProtoSource, an unrecognized value is not rejected here: it simply
// matches no DiscountRule.AppliesTo entry, so the caller gets the correct
// undiscounted band price rather than an error on a read that has always
// succeeded.
func quoteSource(s bookingv1.Source) domain.Source {
	if s == bookingv1.Source_SOURCE_UNSPECIFIED {
		return domain.SourceIndividual
	}
	source, err := fromProtoSource(s)
	if err != nil {
		return domain.Source("")
	}
	return source
}

func (h *Handler) CreateDiscountRule(ctx context.Context, req *bookingv1.CreateDiscountRuleRequest) (*bookingv1.CreateDiscountRuleResponse, error) {
	discountType, err := fromProtoDiscountType(req.GetDiscountType())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	appliesTo, err := fromProtoSources(req.GetAppliesTo())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// No ID is read off the request — CreateDiscountRuleRequest has no id
	// field at all (T11.2's creation-RPC checklist item 1); app.Service mints
	// it from the IDGenerator port. Nor is the actor read off the request any
	// more (T12.7): req.ActorUserId is deprecated and ignored.
	actorUserID, err := h.actor(ctx)
	if err != nil {
		return nil, err
	}

	rule, err := h.svc.CreateDiscountRule(ctx, app.CreateDiscountRuleInput{
		FacilityID:   req.GetFacilityId(),
		ActorUserID:  actorUserID,
		DiscountType: discountType,
		Amount: domain.DiscountAmount{
			Percent: req.GetPercent(),
			Fixed:   domain.Money{Cents: req.GetFixedAmountCents(), Currency: req.GetCurrency()},
		},
		AppliesTo:    appliesTo,
		StartsAt:     req.GetStartsAt().AsTime(),
		EndCondition: fromProtoEndCondition(req.GetEndCondition()),
	})
	if err != nil {
		return nil, toStatus(err)
	}

	return &bookingv1.CreateDiscountRuleResponse{DiscountRule: toProtoDiscountRule(rule)}, nil
}

func (h *Handler) ListDiscountRulesForFacility(ctx context.Context, req *bookingv1.ListDiscountRulesForFacilityRequest) (*bookingv1.ListDiscountRulesForFacilityResponse, error) {
	rules, err := h.svc.ListDiscountRulesForFacility(ctx, req.GetFacilityId())
	if err != nil {
		return nil, toStatus(err)
	}

	out := make([]*bookingv1.DiscountRule, 0, len(rules))
	for _, r := range rules {
		out = append(out, toProtoDiscountRule(r))
	}
	return &bookingv1.ListDiscountRulesForFacilityResponse{DiscountRules: out}, nil
}

func (h *Handler) ListCourtBookings(ctx context.Context, req *bookingv1.ListCourtBookingsRequest) (*bookingv1.ListCourtBookingsResponse, error) {
	rng, err := domain.NewTimeRange(req.GetFrom().AsTime(), req.GetTo().AsTime())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	bookings, err := h.svc.ListCourtBookings(ctx, req.GetCourtId(), rng)
	if err != nil {
		return nil, toStatus(err)
	}

	out := make([]*bookingv1.Booking, 0, len(bookings))
	for _, b := range bookings {
		out = append(out, toProto(b))
	}
	return &bookingv1.ListCourtBookingsResponse{Bookings: out}, nil
}

// CancelBooking cancels a Booking on behalf of its owner.
//
// This is the sharp half of #144: before DECISION D1 this RPC was public and
// took no actor, so anyone who knew or guessed a booking id could cancel it.
// The actor now comes from the verified token via h.actor, and
// app.Service.CancelBooking refuses any caller who is not the booking's
// owner (domain.ErrNotBookingOwner -> PermissionDenied).
//
// As with CreateBooking, no actor is read off the request.
func (h *Handler) CancelBooking(ctx context.Context, req *bookingv1.CancelBookingRequest) (*bookingv1.CancelBookingResponse, error) {
	actorUserID, err := h.actor(ctx)
	if err != nil {
		return nil, err
	}

	b, err := h.svc.CancelBooking(ctx, req.GetBookingId(), actorUserID)
	if err != nil {
		return nil, toStatus(err)
	}
	return &bookingv1.CancelBookingResponse{Booking: toProto(b)}, nil
}

// RequestRecurringHire submits a Club's request for a standing weekly slot
// (T11.5).
//
// No ID is read off the request — RequestRecurringHireRequest has no id field
// at all (creation-RPC checklist item 1); app.Service mints it from the
// IDGenerator port. Nor is any role read off the request (checklist item 2,
// the one that fires for this ticket): the handler passes only the claimed
// actor_user_id, and the app layer resolves the `club` role from Identity.
func (h *Handler) RequestRecurringHire(ctx context.Context, req *bookingv1.RequestRecurringHireRequest) (*bookingv1.RequestRecurringHireResponse, error) {
	startTime, err := fromProtoMinuteOfDay(req.GetStartMinute())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	endTime, err := fromProtoMinuteOfDay(req.GetEndMinute())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	weekday, err := fromProtoWeekday(req.GetWeekday())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	endCondition, err := fromProtoRecurringHireEndCondition(req.GetEndCondition())
	if err != nil {
		return nil, toStatus(err)
	}

	actorUserID, err := h.actor(ctx)
	if err != nil {
		return nil, err
	}

	template, err := h.svc.RequestRecurringHire(ctx, app.RequestRecurringHireInput{
		ActorUserID:  actorUserID,
		CourtID:      req.GetCourtId(),
		Weekday:      weekday,
		StartTime:    startTime,
		EndTime:      endTime,
		StartsAt:     req.GetStartsAt().AsTime(),
		EndCondition: endCondition,
	})
	if err != nil {
		return nil, toStatus(err)
	}

	return &bookingv1.RequestRecurringHireResponse{Template: toProtoRecurringHireTemplate(template)}, nil
}

func (h *Handler) ApproveRecurringHire(ctx context.Context, req *bookingv1.ApproveRecurringHireRequest) (*bookingv1.ApproveRecurringHireResponse, error) {
	actorUserID, err := h.actor(ctx)
	if err != nil {
		return nil, err
	}

	result, err := h.svc.ApproveRecurringHire(ctx, req.GetTemplateId(), actorUserID)
	if err != nil {
		return nil, toStatus(err)
	}

	occurrences := make([]*bookingv1.RecurringHireOccurrence, 0, len(result.Occurrences))
	for _, occ := range result.Occurrences {
		occurrences = append(occurrences, &bookingv1.RecurringHireOccurrence{
			StartsAt:  timestamppb.New(occ.Range.Start),
			EndsAt:    timestamppb.New(occ.Range.End),
			Outcome:   toProtoOccurrenceOutcome(occ.Outcome),
			BookingId: occ.BookingID,
			Reason:    occ.Reason,
		})
	}

	return &bookingv1.ApproveRecurringHireResponse{
		Template:    toProtoRecurringHireTemplate(result.Template),
		Occurrences: occurrences,
	}, nil
}

func (h *Handler) RejectRecurringHire(ctx context.Context, req *bookingv1.RejectRecurringHireRequest) (*bookingv1.RejectRecurringHireResponse, error) {
	actorUserID, err := h.actor(ctx)
	if err != nil {
		return nil, err
	}

	template, err := h.svc.RejectRecurringHire(ctx, req.GetTemplateId(), actorUserID)
	if err != nil {
		return nil, toStatus(err)
	}
	return &bookingv1.RejectRecurringHireResponse{Template: toProtoRecurringHireTemplate(template)}, nil
}

func (h *Handler) ListRecurringHireTemplatesForFacility(ctx context.Context, req *bookingv1.ListRecurringHireTemplatesForFacilityRequest) (*bookingv1.ListRecurringHireTemplatesForFacilityResponse, error) {
	actorUserID, err := h.actor(ctx)
	if err != nil {
		return nil, err
	}

	templates, err := h.svc.ListRecurringHireTemplatesForFacility(ctx, req.GetFacilityId(), actorUserID)
	if err != nil {
		return nil, toStatus(err)
	}

	out := make([]*bookingv1.RecurringHireTemplate, 0, len(templates))
	for _, t := range templates {
		out = append(out, toProtoRecurringHireTemplate(t))
	}
	return &bookingv1.ListRecurringHireTemplatesForFacilityResponse{Templates: out}, nil
}

// ListRecurringHireTemplatesForActor is the Club's own status view (T11.6).
// Every status is returned — a rejected template is reported as rejected, not
// omitted — because omission is what would let a client imply a decided
// request is still open. See the app method for the authorization reasoning
// (the actor's identity is the scope; deliberately no role check).
func (h *Handler) ListRecurringHireTemplatesForActor(ctx context.Context, req *bookingv1.ListRecurringHireTemplatesForActorRequest) (*bookingv1.ListRecurringHireTemplatesForActorResponse, error) {
	actorUserID, err := h.actor(ctx)
	if err != nil {
		return nil, err
	}

	templates, err := h.svc.ListRecurringHireTemplatesForActor(ctx, actorUserID)
	if err != nil {
		return nil, toStatus(err)
	}

	out := make([]*bookingv1.RecurringHireTemplate, 0, len(templates))
	for _, t := range templates {
		out = append(out, toProtoRecurringHireTemplate(t))
	}
	return &bookingv1.ListRecurringHireTemplatesForActorResponse{Templates: out}, nil
}

// toStatus maps domain errors to gRPC status codes. grpc-gateway then maps
// those codes onto HTTP statuses: AlreadyExists -> 409, InvalidArgument ->
// 400, NotFound -> 404 — this is what makes README.md's "overlapping booking
// returns 409" smoke test true end-to-end.
//
// domain.ErrInvalidCourtReference used to be deliberately left unmatched
// here, falling through to the default Internal branch to match the
// untranslated Postgres FK violation an unknown-but-well-formed CourtID
// produced. T15.6 (closes #185) translated that FK violation — 23503 ->
// ErrInvalidCourtReference in adapter/postgres, per CLAUDE.md rule 5 — so
// both halves now arrive as one sentinel and get one explicit NotFound case
// below. See that sentinel's doc comment in domain/errors.go for why one
// sentinel covers both, why NotFound, and why it is not an enumeration
// oracle.
func toStatus(err error) error {
	switch {
	case errors.Is(err, domain.ErrCourtDoubleBooked):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, domain.ErrBookingNotFound),
		errors.Is(err, domain.ErrNoPricingRule),
		// An unknown or malformed FacilityID on CreateDiscountRule (T11.2).
		// The app layer answers a malformed one exactly like an unknown one,
		// so both arrive here as the same sentinel and get the same code.
		// T11.5 also routes an unresolvable CourtID on RequestRecurringHire
		// here: a Court naming no Facility is a request that could never be
		// approved, and NotFound is the honest answer for it. T17.5 (#195)
		// adds a third raise site with the same meaning: a 23503 on
		// recurring_hire_templates.court_id, when a concurrent delete races
		// RequestRecurringHire's FacilityIDForCourt guard — see
		// domain/errors.go's ErrFacilityNotFound comment.
		errors.Is(err, domain.ErrFacilityNotFound),
		// An unknown or malformed template id on approve/reject (T11.5).
		errors.Is(err, domain.ErrRecurringHireTemplateNotFound),
		// T15.6 (closes #185): a CourtID naming no usable court, whether
		// because it is malformed (app-layer uuidShape guard) or because it
		// is well-formed but names no courts row (the Postgres FK, 23503,
		// now translated in adapter/postgres). Joins the two arms directly
		// above it — the same "an unknown or malformed id for a thing this
		// request references" concept they already collapse — rather than
		// the InvalidArgument group: this is a missing referent, not a
		// malformed request envelope.
		errors.Is(err, domain.ErrInvalidCourtReference):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrNotFacilityOwner),
		// DECISION D1 (ADR-0015 option (a); closes #144). A caller who holds
		// a real booking id but does not own it. PermissionDenied, not
		// NotFound, for exactly the reason the arm below states for the
		// Identity sentinels, and matching ErrNotFacilityOwner's own
		// treatment: answering NotFound here would make CancelBooking an
		// oracle for which booking ids exist.
		errors.Is(err, domain.ErrNotBookingOwner),
		// T11.5's two Identity answers. Both are PermissionDenied rather than
		// NotFound, deliberately: an actor who cannot be resolved to a real
		// User has not proven the `club` role any more than a resolved
		// non-club actor has, and answering NotFound for the unresolvable
		// case would turn RequestRecurringHire — an unauthenticated endpoint
		// — into a user-enumeration oracle. See the sentinels' own doc
		// comments in domain/errors.go. T17.5 (#195) adds a third raise site
		// for ErrUserNotFound with the same meaning: a 23503 on
		// recurring_hire_templates.requested_by_user_id, when a concurrent
		// delete races RequestRecurringHire's EnsureClubRole guard.
		errors.Is(err, domain.ErrUserNotFound),
		errors.Is(err, domain.ErrNotClub):
		// PermissionDenied, mirroring facilities' own grpcapi mapping for
		// its ErrNotFacilityOwner (T7.7): a known actor who may not touch
		// this object, not an unknown object and not a server fault.
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, domain.ErrInvalidRecurringHireStatusTransition),
		// T14.7 (closes #158): ErrIllegalStatusTransition joins its own
		// sibling here rather than staying in the InvalidArgument group
		// below. Until this ticket, this single function sent two
		// status-transition sentinels to two different codes with no RPC
		// boundary to explain it — #131's one-concept-two-codes defect,
		// entirely inside one switch. gRPC defines INVALID_ARGUMENT as a
		// problem with the argument *regardless of the state of the system*;
		// domain.Booking.Cancel's guard is `if b.Status != StatusConfirmed`,
		// which is state-dependent by construction. The request is
		// well-formed and would succeed against the same aggregate in a
		// different state — precisely what FAILED_PRECONDITION ("the system
		// is not in a state required for the operation's execution") means
		// and what INVALID_ARGUMENT denies. Matches payments' own
		// ErrIllegalStatusTransition since T13.9.
		//
		// Client impact is nil over REST: grpc-gateway maps both
		// InvalidArgument and FailedPrecondition to HTTP 400.
		errors.Is(err, domain.ErrIllegalStatusTransition):
		// FailedPrecondition: approving or rejecting a template that is
		// already approved/rejected/cancelled, or cancelling an
		// already-cancelled Booking, is a legal request against an object in
		// the wrong state, not a malformed one (InvalidArgument) and not a
		// missing one (NotFound). Mirrors the code this file already gives
		// ErrAmbiguousPricingRule for the same "the request is fine, the
		// world isn't" reason.
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, domain.ErrAmbiguousPricingRule),
		// An ambiguous DiscountRule match surfaced through GetQuote (T11.2)
		// takes the same code as its pricing twin for the same ADR-0002
		// reason — same class of data misconfiguration, so the existing
		// mapping is reused rather than a new one invented.
		errors.Is(err, domain.ErrAmbiguousDiscountRule):
		// FailedPrecondition (-> HTTP 400 via grpc-gateway), not Internal:
		// this is a pricing-data misconfiguration (ADR-0002), distinguishable
		// from an actual server bug so ops/clients can tell them apart.
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, domain.ErrInvalidTimeRange),
		errors.Is(err, domain.ErrInvalidSource),
		errors.Is(err, domain.ErrEmptyCourtID),
		errors.Is(err, domain.ErrPricingSlotSpansMultipleDays),
		// T11.1's DiscountRule constructor sentinels (T11.2 surfaces them):
		// every one is a malformed request the caller must fix, not a
		// server fault. ErrInvalidSource above already covers an AppliesTo
		// entry outside the four locked Source values — see its own comment
		// in domain/errors.go for why that sentinel is reused rather than
		// duplicated.
		errors.Is(err, domain.ErrEmptyFacilityID),
		errors.Is(err, domain.ErrInvalidDiscountType),
		errors.Is(err, domain.ErrInvalidDiscountAmount),
		errors.Is(err, domain.ErrEmptyAppliesTo),
		errors.Is(err, domain.ErrInvalidEndConditionOccurrences),
		// T11.4's RecurringHireTemplate constructor sentinels (T11.5 surfaces
		// them): every one is a malformed request the caller must fix, not a
		// server fault. ErrEmptyCourtID above already covers a template with
		// no court.
		errors.Is(err, domain.ErrEmptyRequestedByUserID),
		// DECISION D1: a Booking built with no owner. InvalidArgument, not
		// PermissionDenied — this is a malformed construction rather than a
		// refused actor, and the two are deliberately separate sentinels;
		// see domain/errors.go. Unreachable through the gRPC handlers, which
		// always pass a resolved actor, but reachable through the app layer
		// by any other caller, so it is mapped rather than left to Internal.
		errors.Is(err, domain.ErrEmptyOwnerUserID),
		errors.Is(err, domain.ErrInvalidClockTime),
		errors.Is(err, domain.ErrInvalidRecurringHireTimeRange),
		errors.Is(err, domain.ErrInvalidRecurringHireEndAfterOccurrences):
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

// fromProtoSources converts CreateDiscountRuleRequest.applies_to. An
// unrecognized (or unspecified) entry is rejected with domain.ErrInvalidSource
// rather than dropped: silently discarding it would persist a DiscountRule
// that applies to fewer sources than the caller asked for, which is exactly
// the "decorative promotion" failure this ticket exists to prevent. An empty
// list is left to domain.NewDiscountRule's own ErrEmptyAppliesTo check, so
// there is one definition of that rule.
func fromProtoSources(sources []bookingv1.Source) ([]domain.Source, error) {
	out := make([]domain.Source, 0, len(sources))
	for _, s := range sources {
		source, err := fromProtoSource(s)
		if err != nil {
			return nil, err
		}
		out = append(out, source)
	}
	return out, nil
}

func fromProtoDiscountType(t bookingv1.DiscountType) (domain.DiscountType, error) {
	switch t {
	case bookingv1.DiscountType_DISCOUNT_TYPE_PERCENT:
		return domain.DiscountTypePercent, nil
	case bookingv1.DiscountType_DISCOUNT_TYPE_FIXED_AMOUNT:
		return domain.DiscountTypeFixedAmount, nil
	default:
		return "", domain.ErrInvalidDiscountType
	}
}

func toProtoDiscountType(t domain.DiscountType) bookingv1.DiscountType {
	switch t {
	case domain.DiscountTypePercent:
		return bookingv1.DiscountType_DISCOUNT_TYPE_PERCENT
	case domain.DiscountTypeFixedAmount:
		return bookingv1.DiscountType_DISCOUNT_TYPE_FIXED_AMOUNT
	default:
		return bookingv1.DiscountType_DISCOUNT_TYPE_UNSPECIFIED
	}
}

// fromProtoEndCondition rebuilds the EndCondition variant through the
// domain's own constructors, so the Kind/payload pairing can only be produced
// one way. An unspecified or unrecognized kind becomes NoEnd — the same
// reading the Postgres adapter's fromEndConditionColumns gives an unreadable
// kind, and the only one that cannot invent an end date the caller never
// sent. An EndAfterOccurrences with a non-positive count is deliberately left
// intact for domain.NewDiscountRule to reject with
// ErrInvalidEndConditionOccurrences rather than being silently normalised
// here.
func fromProtoEndCondition(c *bookingv1.EndCondition) domain.EndCondition {
	switch c.GetKind() {
	case bookingv1.EndConditionKind_END_CONDITION_KIND_END_AFTER_DATE:
		return domain.EndAfterDate(c.GetEndDate().AsTime())
	case bookingv1.EndConditionKind_END_CONDITION_KIND_END_AFTER_OCCURRENCES:
		return domain.EndAfterOccurrences(int(c.GetOccurrences()))
	default:
		return domain.NoEnd()
	}
}

func toProtoEndCondition(c domain.EndCondition) *bookingv1.EndCondition {
	out := &bookingv1.EndCondition{}
	switch c.Kind {
	case domain.EndConditionKindDate:
		out.Kind = bookingv1.EndConditionKind_END_CONDITION_KIND_END_AFTER_DATE
		out.EndDate = timestamppb.New(c.Date)
	case domain.EndConditionKindOccurrences:
		out.Kind = bookingv1.EndConditionKind_END_CONDITION_KIND_END_AFTER_OCCURRENCES
		out.Occurrences = int32(c.Occurrences)
	default:
		out.Kind = bookingv1.EndConditionKind_END_CONDITION_KIND_NO_END
	}
	return out
}

func toProtoDiscountRule(r domain.DiscountRule) *bookingv1.DiscountRule {
	appliesTo := make([]bookingv1.Source, 0, len(r.AppliesTo))
	for _, s := range r.AppliesTo {
		appliesTo = append(appliesTo, toProtoSource(s))
	}
	return &bookingv1.DiscountRule{
		Id:               r.ID,
		FacilityId:       r.FacilityID,
		DiscountType:     toProtoDiscountType(r.DiscountType),
		Percent:          r.Amount.Percent,
		FixedAmountCents: r.Amount.Fixed.Cents,
		Currency:         r.Amount.Fixed.Currency,
		AppliesTo:        appliesTo,
		StartsAt:         timestamppb.New(r.StartsAt),
		EndCondition:     toProtoEndCondition(r.EndCondition),
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

// --- T11.5: RecurringHireTemplate conversions --------------------------

// fromProtoMinuteOfDay converts a wire minute-of-day into domain.ClockTime
// through the domain's own NewClockTime constructor, so the 0..1439 bound has
// one definition. A value outside it becomes ErrInvalidClockTime ->
// InvalidArgument rather than a ClockTime the domain would never construct.
func fromProtoMinuteOfDay(minute int32) (domain.ClockTime, error) {
	return domain.NewClockTime(int(minute)/60, int(minute)%60)
}

// fromProtoWeekday converts a wire weekday using Go's time.Weekday numbering
// (Sunday = 0 .. Saturday = 6), the convention this codebase already uses in
// pricing_rules and recurring_hire_templates alike. Out-of-range values are
// rejected rather than wrapped: time.Weekday(9) is not a day, and silently
// folding it to Tuesday would book a Club a slot it never asked for.
func fromProtoWeekday(weekday int32) (time.Weekday, error) {
	if weekday < 0 || weekday > 6 {
		return 0, fmt.Errorf("%w: weekday %d is outside 0..6", domain.ErrInvalidClockTime, weekday)
	}
	return time.Weekday(weekday), nil
}

// fromProtoRecurringHireEndCondition rebuilds the end-condition variant
// through the domain's own constructors, so the Kind/payload pairing can only
// be produced one way — the same approach fromProtoEndCondition takes for
// DiscountRule's variant.
//
// It differs from that function in one respect, and deliberately: a
// non-positive END_AFTER_OCCURRENCES count is rejected here rather than passed
// through, because T11.4's EndRecurringHireAfterOccurrences validates at
// construction (returning an error) while T11.1's EndAfterOccurrences defers
// to NewDiscountRule. The resulting gRPC code is the same InvalidArgument
// either way; only the layer that notices differs.
//
// An unspecified or unrecognized kind becomes NoRecurringHireEnd — the only
// reading that cannot invent an end date the caller never sent. Note that an
// open-ended template is still bounded on approval by the app layer's
// generation horizon, so this is not a path to an unbounded write.
func fromProtoRecurringHireEndCondition(c *bookingv1.RecurringHireEndCondition) (domain.RecurringHireEndCondition, error) {
	switch c.GetKind() {
	case bookingv1.RecurringHireEndConditionKind_RECURRING_HIRE_END_CONDITION_KIND_END_AFTER_DATE:
		return domain.EndRecurringHireAfterDate(c.GetEndDate().AsTime()), nil
	case bookingv1.RecurringHireEndConditionKind_RECURRING_HIRE_END_CONDITION_KIND_END_AFTER_OCCURRENCES:
		return domain.EndRecurringHireAfterOccurrences(int(c.GetOccurrences()))
	default:
		return domain.NoRecurringHireEnd(), nil
	}
}

func toProtoRecurringHireEndCondition(c domain.RecurringHireEndCondition) *bookingv1.RecurringHireEndCondition {
	out := &bookingv1.RecurringHireEndCondition{}
	switch c.Kind {
	case domain.RecurringHireEndAfterDate:
		out.Kind = bookingv1.RecurringHireEndConditionKind_RECURRING_HIRE_END_CONDITION_KIND_END_AFTER_DATE
		out.EndDate = timestamppb.New(c.Date)
	case domain.RecurringHireEndAfterOccurrences:
		out.Kind = bookingv1.RecurringHireEndConditionKind_RECURRING_HIRE_END_CONDITION_KIND_END_AFTER_OCCURRENCES
		out.Occurrences = int32(c.Occurrences)
	default:
		out.Kind = bookingv1.RecurringHireEndConditionKind_RECURRING_HIRE_END_CONDITION_KIND_NO_END
	}
	return out
}

func toProtoRecurringHireStatus(s domain.RecurringHireStatus) bookingv1.RecurringHireStatus {
	switch s {
	case domain.RecurringHireStatusRequested:
		return bookingv1.RecurringHireStatus_RECURRING_HIRE_STATUS_REQUESTED
	case domain.RecurringHireStatusApproved:
		return bookingv1.RecurringHireStatus_RECURRING_HIRE_STATUS_APPROVED
	case domain.RecurringHireStatusRejected:
		return bookingv1.RecurringHireStatus_RECURRING_HIRE_STATUS_REJECTED
	case domain.RecurringHireStatusCancelled:
		return bookingv1.RecurringHireStatus_RECURRING_HIRE_STATUS_CANCELLED
	default:
		return bookingv1.RecurringHireStatus_RECURRING_HIRE_STATUS_UNSPECIFIED
	}
}

func toProtoOccurrenceOutcome(o app.OccurrenceOutcome) bookingv1.RecurringHireOccurrenceOutcome {
	switch o {
	case app.OccurrenceBooked:
		return bookingv1.RecurringHireOccurrenceOutcome_RECURRING_HIRE_OCCURRENCE_OUTCOME_BOOKED
	case app.OccurrenceSkippedConflict:
		return bookingv1.RecurringHireOccurrenceOutcome_RECURRING_HIRE_OCCURRENCE_OUTCOME_SKIPPED_CONFLICT
	case app.OccurrenceSkippedError:
		return bookingv1.RecurringHireOccurrenceOutcome_RECURRING_HIRE_OCCURRENCE_OUTCOME_SKIPPED_ERROR
	default:
		return bookingv1.RecurringHireOccurrenceOutcome_RECURRING_HIRE_OCCURRENCE_OUTCOME_UNSPECIFIED
	}
}

func toProtoRecurringHireTemplate(t domain.RecurringHireTemplate) *bookingv1.RecurringHireTemplate {
	return &bookingv1.RecurringHireTemplate{
		Id:                t.ID,
		RequestedByUserId: t.RequestedByUserID,
		CourtId:           t.CourtID,
		Weekday:           int32(t.Weekday),
		StartMinute:       int32(t.StartTime),
		EndMinute:         int32(t.EndTime),
		StartsAt:          timestamppb.New(t.StartsAt),
		EndCondition:      toProtoRecurringHireEndCondition(t.EndCondition),
		Status:            toProtoRecurringHireStatus(t.Status),
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
		OwnerUserId: b.OwnerUserID,
	}
}
