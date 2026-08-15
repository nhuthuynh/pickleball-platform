// Package grpcapi is the Facilities context's gRPC adapter. It translates
// between the wire contract generated from proto/pickleball/facilities/v1
// and the app/domain layers, and maps domain errors onto gRPC status codes
// so grpc-gateway's REST mapping produces the right HTTP status (e.g.
// ErrCameraConsentRequired -> codes.FailedPrecondition -> HTTP 400, not a
// 500). It only compiles after `make generate` has produced
// internal/gen/pickleball/facilities/v1 (see CLAUDE.md gotchas). Mirrors
// internal/booking/adapter/grpcapi/handler.go's shape.
package grpcapi

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/nhuthuynh/white-label/internal/facilities/app"
	"github.com/nhuthuynh/white-label/internal/facilities/domain"
	facilitiesv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/facilities/v1"
	"github.com/nhuthuynh/white-label/internal/platform/auth"
)

// actor resolves the acting user for an authenticated RPC (T12.7), and since
// T13.3 it is also ADR-0014's translation seam for this context.
//
// This one method is the whole of A11 Ruling 3 for this context: the
// Principal is translated into the plain actor string app.Service and
// domain.Facility.EnsureOwner already take, here at the grpcapi boundary, so
// internal/facilities/{domain,app} keep their existing signatures and never
// import internal/platform/auth.
//
// It takes only a context, deliberately. The request's actor_user_id field is
// not passed in and cannot be consulted, so there is no fallback to the
// caller's claim when no principal is present — the failure mode the T12.7
// ticket calls out as "a handler that falls back to the claimed value has
// changed nothing".
//
// **It returns a User.ID (uuid), not the subject.** That is ADR-0014's ruling
// and the fix for issue #154. Two steps, in this order:
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
// Every actor-taking RPC on this service calls it — four call sites:
// CreateFacility, AddCourt, AddCameraLink, AttestCameraConsent — so the
// translation happens once and cannot be half-applied. A handler has no other
// way to obtain an actor. Resolving inside each app method instead would have
// to be remembered once per method, which is the shape that produced #154 and
// #146 in the first place.
//
// **Why this fixes all four RPCs and not just CreateFacility.** The other
// three compare the actor against a stored facilities.owner_id via
// domain.Facility.EnsureOwner. Before this ticket both sides of that
// comparison were subjects, so the check was self-consistent and those RPCs
// were not independently broken — they were unreachable, because
// CreateFacility could not persist a Facility for them to act on without
// panicking the Postgres adapter. Moving the translation here keeps both
// sides in the same space, now the uuid one: the write path stores a
// resolved User.ID and the read path compares against a resolved User.ID.
// Resolving on one side only is what would have broken them, which is why
// this is a funnel and not four separate edits.
//
// It is a method rather than a package function only because it needs the
// service to reach port.IdentityLookup. NewHandler's signature is unchanged;
// app.NewService's is not — see its doc comment for why Facilities differs
// from Booking there.
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
	facilitiesv1.UnimplementedFacilitiesServiceServer
	svc *app.Service
}

func NewHandler(svc *app.Service) *Handler {
	return &Handler{svc: svc}
}

// CreateFacility mints the new Facility's owner from the verified caller
// (T12.7). req.OwnerId is deprecated and ignored.
//
// This goes one step further than the other three RPCs, and it has to. The
// others compare a caller against a stored owner_id; this one *writes* that
// owner_id. If it kept taking the owner from the wire, every Facility created
// after this ticket would be owned by whatever string the client sent, while
// AddCourt compared against the caller's verified subject — so either the two
// would never match (the owner locked out of their own Facility) or a caller
// could hand ownership to a subject that is not theirs. Minting it here is
// what makes the enforced check on the other three internally consistent.
func (h *Handler) CreateFacility(ctx context.Context, req *facilitiesv1.CreateFacilityRequest) (*facilitiesv1.CreateFacilityResponse, error) {
	ownerID, err := h.actor(ctx)
	if err != nil {
		return nil, err
	}

	f, err := h.svc.CreateFacility(ctx, app.CreateFacilityInput{
		OwnerID:     ownerID,
		Name:        req.GetName(),
		Description: req.GetDescription(),
		Address:     req.GetAddress(),
		PhotoURLs:   req.GetPhotoUrls(),
	})
	if err != nil {
		return nil, toStatus(err)
	}
	return &facilitiesv1.CreateFacilityResponse{Facility: toProto(f)}, nil
}

// GetFacility's response includes Courts (T8.2) alongside Facility — kept as
// a separate response field rather than on Facility itself, so
// CreateFacility/ListFacilities/AddCameraLink (which all build a Facility
// via toProto below) don't pay for an extra courts query none of them need.
// See GetFacilityResponse's doc comment in
// proto/pickleball/facilities/v1/facilities.proto for the full reasoning,
// including the ListFacilityCourts-RPC alternative considered and rejected.
func (h *Handler) GetFacility(ctx context.Context, req *facilitiesv1.GetFacilityRequest) (*facilitiesv1.GetFacilityResponse, error) {
	f, err := h.svc.GetFacility(ctx, req.GetFacilityId())
	if err != nil {
		return nil, toStatus(err)
	}
	courts := make([]*facilitiesv1.Court, 0, len(f.Courts))
	for _, c := range f.Courts {
		courts = append(courts, toProtoCourt(c))
	}
	return &facilitiesv1.GetFacilityResponse{Facility: toProto(f), Courts: courts}, nil
}

func (h *Handler) ListFacilities(ctx context.Context, req *facilitiesv1.ListFacilitiesRequest) (*facilitiesv1.ListFacilitiesResponse, error) {
	facilities, err := h.svc.ListFacilities(ctx, req.GetNameFilter())
	if err != nil {
		return nil, toStatus(err)
	}
	out := make([]*facilitiesv1.Facility, 0, len(facilities))
	for _, f := range facilities {
		out = append(out, toProto(f))
	}
	return &facilitiesv1.ListFacilitiesResponse{Facilities: out}, nil
}

// AddCourt, AddCameraLink and AttestCameraConsent all take their actor from
// the verified principal (T12.7). Each request message still carries a
// deprecated actor_user_id — web/ sends it until its own follow-up, and
// removing a proto field is a client break — but no handler below reads it.
func (h *Handler) AddCourt(ctx context.Context, req *facilitiesv1.AddCourtRequest) (*facilitiesv1.AddCourtResponse, error) {
	actorUserID, err := h.actor(ctx)
	if err != nil {
		return nil, err
	}

	c, err := h.svc.AddCourt(ctx, req.GetFacilityId(), actorUserID, req.GetName())
	if err != nil {
		return nil, toStatus(err)
	}
	return &facilitiesv1.AddCourtResponse{Court: toProtoCourt(c)}, nil
}

func (h *Handler) AddCameraLink(ctx context.Context, req *facilitiesv1.AddCameraLinkRequest) (*facilitiesv1.AddCameraLinkResponse, error) {
	actorUserID, err := h.actor(ctx)
	if err != nil {
		return nil, err
	}

	f, err := h.svc.AddCameraLink(ctx, req.GetFacilityId(), actorUserID, req.GetUrl())
	if err != nil {
		return nil, toStatus(err)
	}
	return &facilitiesv1.AddCameraLinkResponse{Facility: toProto(f)}, nil
}

func (h *Handler) AttestCameraConsent(ctx context.Context, req *facilitiesv1.AttestCameraConsentRequest) (*facilitiesv1.AttestCameraConsentResponse, error) {
	actorUserID, err := h.actor(ctx)
	if err != nil {
		return nil, err
	}

	f, err := h.svc.AttestCameraConsent(ctx, req.GetFacilityId(), actorUserID)
	if err != nil {
		return nil, toStatus(err)
	}
	return &facilitiesv1.AttestCameraConsentResponse{Facility: toProto(f)}, nil
}

// toStatus maps domain errors to gRPC status codes. grpc-gateway then maps
// those codes onto HTTP statuses: NotFound -> 404, InvalidArgument -> 400,
// FailedPrecondition -> 400, PermissionDenied -> 403 — this is what makes
// the T7.3 smoke-test AC ("adding a camera link before consent is
// attested returns a mapped 4xx, not a 500") and T7.7's authz regression
// test ("AddCourt/AddCameraLink against a Facility the actor doesn't own
// returns a mapped 4xx, not a 500") true end-to-end.
func toStatus(err error) error {
	switch {
	case errors.Is(err, domain.ErrFacilityNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrNotFacilityOwner):
		// PermissionDenied (-> HTTP 403 via grpc-gateway), not Internal:
		// T7.7's object-level (BOLA) authorization rejection — mirrors
		// internal/socialplay/adapter/grpcapi's ErrNotRegistrationOwner
		// handling (T5.5) and internal/payments/adapter/grpcapi's
		// ErrNotPaymentRecorder handling (T6.3/T6.7).
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, domain.ErrCameraConsentRequired):
		// FailedPrecondition (-> HTTP 400 via grpc-gateway), not Internal:
		// this is a client-visible, expected precondition failure (the
		// caller hasn't attested camera consent yet), not a server bug —
		// mirrors booking's toStatus's handling of
		// ErrAmbiguousPricingRule.
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, domain.ErrUserNotFound):
		// PermissionDenied (-> HTTP 403 via grpc-gateway), not NotFound and
		// not Unauthenticated: ADR-0014 §6's ruling for a caller whose token
		// verified but who is registered to no User (T13.3).
		//
		// Unauthenticated is wrong because we know exactly who they are — the
		// token verified, and ADR-0013 §5 reserves that code for "I do not
		// know who you are". NotFound is wrong for two reasons: it would make
		// a 404 from AddCourt ambiguous between "no such Facility" and "no
		// such you", and it would turn every actor-taking RPC here into a
		// user-enumeration oracle. This reuses the mapping Booking's toStatus
		// already gives the same sentinel rather than inventing one.
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, domain.ErrEmptyFacilityField), errors.Is(err, domain.ErrEmptyCourtField):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}

func toProtoCameraLink(c domain.CameraLink) *facilitiesv1.CameraLink {
	return &facilitiesv1.CameraLink{Url: c.URL, CourtId: c.CourtID}
}

func toProto(f domain.Facility) *facilitiesv1.Facility {
	links := make([]*facilitiesv1.CameraLink, 0, len(f.CameraLinks))
	for _, l := range f.CameraLinks {
		links = append(links, toProtoCameraLink(l))
	}
	return &facilitiesv1.Facility{
		Id:                    f.ID,
		OwnerId:               f.OwnerID,
		Name:                  f.Name,
		Description:           f.Description,
		Address:               f.Address,
		PhotoUrls:             f.PhotoURLs,
		CameraConsentAttested: f.CameraConsentAttested,
		CameraLinks:           links,
	}
}

func toProtoCourt(c domain.Court) *facilitiesv1.Court {
	return &facilitiesv1.Court{Id: c.ID, FacilityId: c.FacilityID, Name: c.Name}
}
