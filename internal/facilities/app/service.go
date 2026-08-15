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
	repo     port.Repository
	identity port.IdentityLookup
	ids      port.IDGenerator
}

// NewService gained its identity parameter in T13.3, when Facilities acquired
// its first outbound port into another context (ADR-0014, issue #154).
//
// The parameter order mirrors internal/booking/app.NewService's — every
// cross-context lookup port first, port.IDGenerator last — rather than
// appending the new one on the end, so the two services stay readable against
// each other (CLAUDE.md: "mirror the booking context exactly").
//
// Note that ADR-0014's "constructor signatures are unchanged" consequence is
// specific to Booking, and does not apply here: Booking already held
// port.IdentityLookup as a constructor parameter before that ADR, so its seam
// added a method to an existing dependency. Facilities held no such port at
// all, so the dependency is genuinely new and has to be wired. cmd/server
// passes the real internal/facilities/adapter/identity.Lookup over the same
// identitySvc instance every other context resolves against.
func NewService(repo port.Repository, identity port.IdentityLookup, ids port.IDGenerator) *Service {
	return &Service{repo: repo, identity: identity, ids: ids}
}

// ResolveActorUserID translates a verified IdP subject into the caller's
// User.ID (uuid) — ADR-0014's resolution seam applied to Facilities, and the
// fix for issue #154.
//
// **This is the only method in internal/facilities/app whose parameter is a
// subject, and the name says so deliberately.** ADR-0014's invariant is that
// below the grpcapi boundary an actor value is always a User.ID; this method
// is the one place that invariant is established rather than assumed. Every
// other actor-taking method here — CreateFacility, AddCourt, AddCameraLink,
// AttestCameraConsent — takes the resolved uuid.
//
// It is called from the grpcapi handler's actor() funnel, once per
// authenticated RPC, so a subject never reaches the two places that would
// mishandle it: a uuid column (the Postgres adapter's mustUUID panics — see
// adapter/postgres/owner_subject_panic_test.go) and a uuid-keyed comparison
// (domain.Facility.EnsureOwner against a stored owner_id).
//
// It takes a plain string rather than an auth.Principal on purpose: this
// package still imports nothing from internal/platform/auth (A11 Ruling 3),
// so the app layer keeps no opinion about how the caller was authenticated —
// only that somebody upstream did.
//
// An unregistered subject is domain.ErrUserNotFound, which grpcapi maps to
// PermissionDenied rather than NotFound (ADR-0014 §6): the caller is known,
// they simply may not act, and answering NotFound would turn every
// actor-taking endpoint into a user-enumeration oracle.
func (s *Service) ResolveActorUserID(ctx context.Context, subject string) (string, error) {
	return s.identity.UserIDBySubject(ctx, subject)
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
	// T13.3 / ADR-0014: OwnerID is a resolved User.ID by the time it gets
	// here, because the grpcapi handler's actor() funnel resolved it. This
	// guard is what makes that an enforced invariant rather than a hope.
	//
	// It is the same shape as the FacilityID guards on AddCourt/
	// AddCameraLink/AttestCameraConsent below (T10.7, issue #97), applied to
	// the actor field for the same reason: the Postgres adapter converts
	// owner_id with mustUUID, which PANICS on a non-uuid rather than
	// returning an error, and grpc installs no recover() beyond
	// internal/platform/grpcrecovery. That panic is issue #154, and it is
	// pinned Docker-free in
	// internal/facilities/adapter/postgres/owner_subject_panic_test.go.
	//
	// So a future RPC wired past the funnel — or a caller of this package
	// from outside grpcapi — fails closed with a clean PermissionDenied
	// instead of taking the process down. domain.NewFacility cannot do this
	// job: it validates OwnerID for non-emptiness only, and must, since
	// internal/facilities/domain is not allowed to know that a User.ID is a
	// uuid in Postgres (CLAUDE.md rule 2).
	//
	// ErrUserNotFound rather than a malformed-input error, matching how
	// internal/booking/app answers the same case: an actor value that is not
	// a User.ID resolves to no User, and ADR-0014 §6 rules that answer is
	// PermissionDenied.
	if !uuidShape.MatchString(in.OwnerID) {
		return domain.Facility{}, domain.ErrUserNotFound
	}

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
