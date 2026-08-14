// Package app is the Facilities context's application layer: it
// orchestrates the domain and the repository port, but holds no business
// rules itself — those live in internal/facilities/domain. Mirrors
// internal/booking/app's shape (CLAUDE.md: "mirror the booking context
// exactly").
package app

import (
	"context"
	"regexp"

	"github.com/nhuthuynh/white-label/internal/facilities/domain"
	"github.com/nhuthuynh/white-label/internal/facilities/port"
)

// uuidShape matches the canonical 8-4-4-4-12 hex form internal/platform/idgen
// mints for every Facility and Court ID.
//
// Boundary guard for caller-supplied IDs: the Postgres adapter's mustUUID
// panics on anything pgtype.UUID.Scan can't parse, and grpc installs no
// recover() of its own, so an unvalidated ID off the wire could take the whole
// process down. Deliberately narrower than github.com/google/uuid's Validate,
// which accepts braced and `urn:uuid:` forms that pgtype rejects — a guard
// wider than the thing it protects is not a guard. Kept context-local rather
// than shared, matching how this repo keeps each context's own not-found
// sentinel local (see socialplay/domain's ErrFacilityNotFound comment). The
// canonical write-up lives on internal/competitions/app's copy.
var uuidShape = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// Service is the Facilities context's application layer.
type Service struct {
	repo port.Repository
	ids  port.IDGenerator
}

func NewService(repo port.Repository, ids port.IDGenerator) *Service {
	return &Service{repo: repo, ids: ids}
}

// CreateFacilityInput is CreateFacility's use-case input.
type CreateFacilityInput struct {
	OwnerID     string
	Name        string
	Description string
	Address     string
	PhotoURLs   []string
}

// CreateFacility validates and persists a new Facility. CameraConsentAttested
// always starts false — domain.NewFacility deliberately has no parameter
// that could set it true on creation (T7.2's round-10 design-review
// finding), so there is nothing for this method to pass through either.
func (s *Service) CreateFacility(ctx context.Context, in CreateFacilityInput) (domain.Facility, error) {
	f, err := domain.NewFacility(s.ids.NewID(), in.OwnerID, in.Name, in.Description, in.Address, in.PhotoURLs)
	if err != nil {
		return domain.Facility{}, err
	}
	return s.repo.CreateFacility(ctx, f)
}

// GetFacility returns a single Facility by id, or domain.ErrFacilityNotFound.
func (s *Service) GetFacility(ctx context.Context, id string) (domain.Facility, error) {
	// A malformed ID is answered exactly like an unknown one. Besides keeping
	// the adapter's mustUUID from panicking on wire input, this preserves the
	// project convention that an unresolvable Facility reference is a 404 and
	// never a 500 — see domain.ErrFacilityNotFound.
	if !uuidShape.MatchString(id) {
		return domain.Facility{}, domain.ErrFacilityNotFound
	}
	return s.repo.GetFacilityByID(ctx, id)
}

// GetCourt returns a single Court by id, or domain.ErrCourtNotFound (T11.2).
// It exists so another bounded context can resolve a Court to its owning
// Facility through a port instead of reading the courts table directly:
// internal/booking/adapter/facilities calls this to satisfy Booking's
// port.FacilityLookup.FacilityIDForCourt, which GetQuote needs to resolve a
// facility-scoped DiscountRule from the CourtID it already has.
//
// A Court whose facility_id is NULL (the seeded, pre-Facilities courts of
// 0001_init.sql/0010_facilities.sql) is returned normally with an empty
// FacilityID — "this Court belongs to no Facility" is a real, expected answer
// in this schema, not an error.
func (s *Service) GetCourt(ctx context.Context, id string) (domain.Court, error) {
	// Same T10.7 guard GetFacility/AddCourt apply to their own
	// caller-supplied IDs: a malformed ID is answered exactly like an unknown
	// one, so it never reaches the adapter's mustUUID (which panics on
	// anything pgtype.UUID.Scan rejects).
	if !uuidShape.MatchString(id) {
		return domain.Court{}, domain.ErrCourtNotFound
	}
	return s.repo.GetCourtByID(ctx, id)
}

// ListFacilities returns Facilities matching nameFilter (a case-insensitive
// substring match on Name — no real geo-search, per HANDOFF.md T7.3), or
// all Facilities when nameFilter is empty. All the actual filtering lives
// in the repository (mirroring the query the Postgres adapter runs).
func (s *Service) ListFacilities(ctx context.Context, nameFilter string) ([]domain.Facility, error) {
	return s.repo.ListFacilities(ctx, nameFilter)
}

// AddCourt validates and persists a new Court scoped to facilityID, via
// Repository.AddCourt — which the Postgres adapter wires against the
// *existing* courts table (HANDOFF.md T7.3 AC 5), so a Court created here
// is immediately usable by Booking's CreateBooking/ListCourtBookings
// against the same courts.id value, unmodified.
//
// T7.7 adds the object-level (BOLA) ownership check here: unlike
// AddCameraLink (which already loaded the Facility to check
// CameraConsentAttested), AddCourt previously never looked the Facility up
// at all, so this method now fetches it first and calls
// domain.Facility.EnsureOwner(actorUserID) before constructing or
// persisting the Court — a mismatched actorUserID returns
// domain.ErrNotFacilityOwner (-> codes.PermissionDenied via grpcapi.
// toStatus) and never reaches Repository.AddCourt. As with T5.5/T6.7,
// actorUserID is a caller-supplied claim, not a verified identity — see
// domain.ErrNotFacilityOwner's doc comment.
func (s *Service) AddCourt(ctx context.Context, facilityID, actorUserID, name string) (domain.Court, error) {
	// A malformed FacilityID is answered exactly like an unknown one (T10.7,
	// closing issue #97): this method already calls GetFacilityByID first,
	// before EnsureOwner, and already returns the bare
	// domain.ErrFacilityNotFound for a miss — matching GetFacility's own
	// guard on the same field, applied here because AddCourt is a write path
	// PR #89's original Layer 2 pass didn't cover.
	if !uuidShape.MatchString(facilityID) {
		return domain.Court{}, domain.ErrFacilityNotFound
	}

	f, err := s.repo.GetFacilityByID(ctx, facilityID)
	if err != nil {
		return domain.Court{}, err
	}
	if err := f.EnsureOwner(actorUserID); err != nil {
		return domain.Court{}, err
	}

	c, err := domain.NewCourt(s.ids.NewID(), facilityID, name)
	if err != nil {
		return domain.Court{}, err
	}
	return s.repo.AddCourt(ctx, c)
}

// AddCameraLink adds a facility-wide camera link to facilityID, but only
// once the caller owns the Facility (T7.7: domain.Facility.EnsureOwner,
// checked first, returns domain.ErrNotFacilityOwner for a mismatched
// actorUserID -> codes.PermissionDenied via grpcapi.toStatus) and once
// that Facility's CameraConsentAttested is true. Both checks live entirely
// in domain.Facility.AddCameraLink (T7.2 for consent, T7.7 for ownership)
// — this method's only job is to fetch the Facility, delegate to that
// domain method (which leaves the Facility untouched and never calls the
// repository when either check fails), and persist the newly appended
// link when it succeeds. grpcapi maps both ErrCameraConsentRequired and
// ErrNotFacilityOwner to non-500 statuses — see HANDOFF.md T7.3's
// smoke-test AC and T7.7's authz regression test.
func (s *Service) AddCameraLink(ctx context.Context, facilityID, actorUserID, url string) (domain.Facility, error) {
	// Same T10.7 guard AddCourt applies above (closing issue #97), found by
	// this ticket's required inspection sweep — this method also calls
	// GetFacilityByID(facilityID) first and already returns the bare
	// domain.ErrFacilityNotFound for an unknown-but-well-formed id.
	if !uuidShape.MatchString(facilityID) {
		return domain.Facility{}, domain.ErrFacilityNotFound
	}

	f, err := s.repo.GetFacilityByID(ctx, facilityID)
	if err != nil {
		return domain.Facility{}, err
	}

	if err := f.AddCameraLink(actorUserID, url); err != nil {
		return domain.Facility{}, err
	}

	newLink := f.CameraLinks[len(f.CameraLinks)-1]
	if _, err := s.repo.AddCameraLink(ctx, facilityID, newLink); err != nil {
		return domain.Facility{}, err
	}

	return f, nil
}

// AttestCameraConsent sets facilityID's CameraConsentAttested to true, but
// only once the caller owns the Facility (T8.4: domain.Facility.
// EnsureOwner, checked first inside domain.Facility.AttestCameraConsent —
// returns domain.ErrNotFacilityOwner for a mismatched actorUserID ->
// codes.PermissionDenied via grpcapi.toStatus, mirroring AddCourt/
// AddCameraLink's T7.7 check). This is the RPC that closes the gap
// AddCameraLink's doc comment describes: before this method existed,
// nothing could ever set CameraConsentAttested to true server-side, so
// every correct client submission to AddCameraLink was rejected with
// ErrCameraConsentRequired. Idempotent, per domain.Facility.
// AttestCameraConsent's doc comment.
func (s *Service) AttestCameraConsent(ctx context.Context, facilityID, actorUserID string) (domain.Facility, error) {
	// Same T10.7 guard as AddCourt/AddCameraLink above (closing issue #97).
	if !uuidShape.MatchString(facilityID) {
		return domain.Facility{}, domain.ErrFacilityNotFound
	}

	f, err := s.repo.GetFacilityByID(ctx, facilityID)
	if err != nil {
		return domain.Facility{}, err
	}

	if err := f.AttestCameraConsent(actorUserID); err != nil {
		return domain.Facility{}, err
	}

	if err := s.repo.AttestCameraConsent(ctx, facilityID); err != nil {
		return domain.Facility{}, err
	}

	return f, nil
}
