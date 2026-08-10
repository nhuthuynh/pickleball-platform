package app

import (
	"context"
	"regexp"

	"github.com/nhuthuynh/white-label/internal/booking/domain"
	"github.com/nhuthuynh/white-label/internal/booking/port"
)

// uuidShape matches the canonical 8-4-4-4-12 hex form internal/platform/idgen
// mints for every Booking and Court ID.
//
// Boundary guard for caller-supplied IDs: the Postgres adapter's mustUUID
// panics on anything pgtype.UUID.Scan can't parse, and grpc installs no
// recover() of its own, so an unvalidated ID off the wire could take the whole
// process down. Deliberately narrower than github.com/google/uuid's Validate,
// which accepts braced and `urn:uuid:` forms that pgtype rejects. The canonical
// write-up lives on internal/competitions/app's copy.
var uuidShape = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// Service is the Booking context's application layer: it orchestrates the
// domain and the repository port, but holds no business rules itself — those
// live in internal/booking/domain.
type Service struct {
	repo        port.Repository
	pricingRepo port.PricingRuleRepository
	ids         port.IDGenerator
}

func NewService(repo port.Repository, pricingRepo port.PricingRuleRepository, ids port.IDGenerator) *Service {
	return &Service{repo: repo, pricingRepo: pricingRepo, ids: ids}
}

// CreateBookingInput is the use-case input for creating any of the four
// Booking sources (D3b) — the same use case serves a direct court booking, a
// Game reserving its courts, a Competition reserving its courts, and a Club
// recurring-hire occurrence, differing only in Source/ReferenceID.
type CreateBookingInput struct {
	CourtID     string
	Source      domain.Source
	Range       domain.TimeRange
	ReferenceID string
}

// CreateBooking validates the candidate booking, pre-checks it against the
// court's other active bookings (domain.EnsureNoConflict — regardless of
// their source, per D3b/F1), and persists it. The Postgres adapter's EXCLUDE
// constraint remains the authoritative guard under concurrent requests (see
// HANDOFF.md T4); this pre-check exists to fail fast and give a clear domain
// error without waiting on a round trip that's doomed to be rejected.
func (s *Service) CreateBooking(ctx context.Context, in CreateBookingInput) (domain.Booking, error) {
	candidate, err := domain.NewBooking(s.ids.NewID(), in.CourtID, in.Source, in.Range, in.ReferenceID)
	if err != nil {
		return domain.Booking{}, err
	}

	// A malformed (non-empty, wrong-shape) CourtID is rejected here, before
	// either repository call, with domain.ErrInvalidCourtReference (T10.7
	// follow-up, closing issue #97): ListActiveForCourt below is the same
	// mustUUID-backed adapter method ListCourtBookings' own already-guarded
	// read calls, and a malformed CourtID reached it unguarded — panicking
	// there, one step before Create's own FK-violation path, which is what
	// this sentinel's gRPC code (Internal, via adapter/grpcapi's default
	// toStatus case — see the sentinel's own doc comment) is chosen to
	// match. An empty CourtID is unaffected: domain.NewBooking's own
	// ErrEmptyCourtID check above already runs first and still fires for
	// that case, exactly as before this guard was added.
	if !uuidShape.MatchString(in.CourtID) {
		return domain.Booking{}, domain.ErrInvalidCourtReference
	}

	existing, err := s.repo.ListActiveForCourt(ctx, in.CourtID, in.Range)
	if err != nil {
		return domain.Booking{}, err
	}

	if err := domain.EnsureNoConflict(candidate, existing); err != nil {
		return domain.Booking{}, err
	}

	return s.repo.Create(ctx, candidate)
}

// ListCourtBookings returns the active (non-cancelled) bookings on courtID
// that intersect r, regardless of source (HANDOFF.md T2). All the actual
// filtering lives in the repository (mirroring the query the Postgres
// adapter runs); this method exists so the API layer depends on the app
// layer rather than the repository port directly.
func (s *Service) ListCourtBookings(ctx context.Context, courtID string, r domain.TimeRange) ([]domain.Booking, error) {
	// A malformed courtID is answered exactly like an unknown one. This read is
	// list-shaped — an unknown Court yields an empty schedule rather than an
	// error — so a malformed Court must yield an empty schedule too, rather
	// than an error this method never otherwise returns.
	if !uuidShape.MatchString(courtID) {
		return []domain.Booking{}, nil
	}
	return s.repo.ListActiveForCourt(ctx, courtID, r)
}

// CancelBooking transitions a booking to cancelled (HANDOFF.md T3). Once
// cancelled, the slot it held is free — domain.EnsureNoConflict already
// ignores cancelled bookings (T0), so no separate "free the slot" step is
// needed here beyond persisting the status change itself.
func (s *Service) CancelBooking(ctx context.Context, bookingID string) (domain.Booking, error) {
	// A malformed bookingID is answered exactly like an unknown one (T10.7,
	// closing issue #97): this method already calls GetByID first and
	// already returns the bare domain.ErrBookingNotFound for a miss —
	// found by this ticket's required inspection sweep, since CancelBooking
	// (unlike ListCourtBookings/GetQuote just above) had never had this
	// guard applied, though it reaches the identical mustUUID panic path.
	if !uuidShape.MatchString(bookingID) {
		return domain.Booking{}, domain.ErrBookingNotFound
	}

	b, err := s.repo.GetByID(ctx, bookingID)
	if err != nil {
		return domain.Booking{}, err
	}

	if err := b.Cancel(); err != nil {
		return domain.Booking{}, err
	}

	return s.repo.Update(ctx, b)
}

// GetQuote resolves the price a slot on courtID would cost, per the court's
// pricing rules (HANDOFF.md T1). It is thin by design — all the actual
// resolution logic (band matching, boundary handling, ambiguity detection)
// lives in domain.ResolvePrice; this method's only job is the repository
// round trip.
func (s *Service) GetQuote(ctx context.Context, courtID string, r domain.TimeRange) (domain.PricingRule, error) {
	// A malformed courtID is answered exactly like a well-formed one with no
	// rules configured (ErrNoPricingRule) — this get-shaped read already
	// returns that error for an unknown Court, so a malformed Court must
	// produce the same answer rather than reaching the Postgres adapter's
	// mustUUID, which panics on non-UUID input (PR #89's review found this
	// endpoint, GetQuote, was the one public unauthenticated read PR #89
	// itself missed; ListCourtBookings' identical guard just above is the
	// pattern this mirrors).
	if !uuidShape.MatchString(courtID) {
		return domain.PricingRule{}, domain.ErrNoPricingRule
	}
	rules, err := s.pricingRepo.ListForCourt(ctx, courtID)
	if err != nil {
		return domain.PricingRule{}, err
	}
	return domain.ResolvePrice(rules, courtID, r)
}
