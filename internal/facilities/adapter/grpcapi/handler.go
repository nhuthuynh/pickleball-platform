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
)

type Handler struct {
	facilitiesv1.UnimplementedFacilitiesServiceServer
	svc *app.Service
}

func NewHandler(svc *app.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) CreateFacility(ctx context.Context, req *facilitiesv1.CreateFacilityRequest) (*facilitiesv1.CreateFacilityResponse, error) {
	f, err := h.svc.CreateFacility(ctx, app.CreateFacilityInput{
		OwnerID:     req.GetOwnerId(),
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

func (h *Handler) GetFacility(ctx context.Context, req *facilitiesv1.GetFacilityRequest) (*facilitiesv1.GetFacilityResponse, error) {
	f, err := h.svc.GetFacility(ctx, req.GetFacilityId())
	if err != nil {
		return nil, toStatus(err)
	}
	return &facilitiesv1.GetFacilityResponse{Facility: toProto(f)}, nil
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

func (h *Handler) AddCourt(ctx context.Context, req *facilitiesv1.AddCourtRequest) (*facilitiesv1.AddCourtResponse, error) {
	c, err := h.svc.AddCourt(ctx, req.GetFacilityId(), req.GetActorUserId(), req.GetName())
	if err != nil {
		return nil, toStatus(err)
	}
	return &facilitiesv1.AddCourtResponse{Court: toProtoCourt(c)}, nil
}

func (h *Handler) AddCameraLink(ctx context.Context, req *facilitiesv1.AddCameraLinkRequest) (*facilitiesv1.AddCameraLinkResponse, error) {
	f, err := h.svc.AddCameraLink(ctx, req.GetFacilityId(), req.GetActorUserId(), req.GetUrl())
	if err != nil {
		return nil, toStatus(err)
	}
	return &facilitiesv1.AddCameraLinkResponse{Facility: toProto(f)}, nil
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
