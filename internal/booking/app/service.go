package app

import (
	"context"
	"errors"
	"regexp"
	"time"

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
	repo         port.Repository
	pricingRepo  port.PricingRuleRepository
	discountRepo port.DiscountRuleRepository
	facilities   port.FacilityLookup
	ids          port.IDGenerator
}

func NewService(
	repo port.Repository,
	pricingRepo port.PricingRuleRepository,
	discountRepo port.DiscountRuleRepository,
	facilities port.FacilityLookup,
	ids port.IDGenerator,
) *Service {
	return &Service{
		repo:         repo,
		pricingRepo:  pricingRepo,
		discountRepo: discountRepo,
		facilities:   facilities,
		ids:          ids,
	}
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

// GetQuoteInput is GetQuote's use-case input. It became a struct in T11.2
// when Source joined CourtID/Range — the same threshold CreateBookingInput
// crossed, and the same reason: a 4-argument positional call is where the
// wrong two strings start swapping places silently.
type GetQuoteInput struct {
	CourtID string
	// Source is the reservation kind this quote is for. It selects which
	// DiscountRules apply (domain.DiscountRule.AppliesTo), and is new in
	// T11.2 — the quote path had no notion of Source before discounts
	// existed. The gRPC handler defaults an unspecified Source to
	// SourceIndividual; see its doc comment for why that default is part of
	// the contract rather than a guess.
	Source domain.Source
	Range  domain.TimeRange
}

// Quote is what GetQuote answers with. Before T11.2 this method returned the
// resolved domain.PricingRule directly; a discount means the price a Player
// pays is no longer simply the band's price, and the caller needs both
// numbers — the UI has to be able to show what was taken off rather than
// swapping one number for another with no indication a discount applied
// (T11.3's honest-UI requirement).
type Quote struct {
	// Band is the pricing band that resolved, unchanged by any discount.
	Band domain.Band
	// BandPriceCents is the pre-discount price from that band.
	BandPriceCents int64
	// PriceCents is what the Player actually pays: BandPriceCents with any
	// resolved discount applied. Equal to BandPriceCents when no discount
	// applies.
	PriceCents int64
	// Discount is the rule that was applied, or the zero value when none
	// was. Use Discount.Applies() to tell them apart — a zero-value rule is
	// "no discount", not a 0%-off discount.
	Discount domain.DiscountRule
}

// GetQuote resolves the price a slot on CourtID would cost: the court's
// pricing band (HANDOFF.md T1), with any active facility-wide DiscountRule
// applied on top (T11.2). It stays thin by design — band matching, boundary
// handling, ambiguity detection and the discount arithmetic all live in the
// domain (ResolvePrice / ResolveDiscount / DiscountRule.ApplyToCents); this
// method's job is the repository round trips and the order they happen in.
//
// Pricing is resolved first and its errors are returned unchanged, so every
// pre-T11.2 GetQuote behaviour is preserved exactly: an unknown or malformed
// Court, or a slot outside every band, still fails the same way it always did
// and never reaches the discount step.
func (s *Service) GetQuote(ctx context.Context, in GetQuoteInput) (Quote, error) {
	// A malformed courtID is answered exactly like a well-formed one with no
	// rules configured (ErrNoPricingRule) — this get-shaped read already
	// returns that error for an unknown Court, so a malformed Court must
	// produce the same answer rather than reaching the Postgres adapter's
	// mustUUID, which panics on non-UUID input (PR #89's review found this
	// endpoint, GetQuote, was the one public unauthenticated read PR #89
	// itself missed; ListCourtBookings' identical guard just above is the
	// pattern this mirrors).
	if !uuidShape.MatchString(in.CourtID) {
		return Quote{}, domain.ErrNoPricingRule
	}
	rules, err := s.pricingRepo.ListForCourt(ctx, in.CourtID)
	if err != nil {
		return Quote{}, err
	}
	band, err := domain.ResolvePrice(rules, in.CourtID, in.Range)
	if err != nil {
		return Quote{}, err
	}

	quote := Quote{Band: band.Band, BandPriceCents: band.PriceCents, PriceCents: band.PriceCents}

	discount, err := s.resolveDiscount(ctx, in)
	if err != nil {
		return Quote{}, err
	}
	quote.Discount = discount
	quote.PriceCents = discount.ApplyToCents(band.PriceCents)
	return quote, nil
}

// resolveDiscount finds the DiscountRule (if any) that applies to this quote.
// DiscountRules are facility-scoped (A9's narrowing) while the quote path
// keys off CourtID, so this is where Booking resolves a Court to its Facility
// — through port.FacilityLookup, the same one CreateDiscountRule authorizes
// against, never a second mechanism.
//
// A Court that resolves to no Facility (unknown Court, or the nullable
// courts.facility_id case) yields no discount and no error: the quote is
// still a real, correct band price, and refusing to quote a Court that
// predates the Facilities context would break a read that has worked since
// T1. Every other error — including domain.ErrAmbiguousDiscountRule, which
// ADR-0002 requires be surfaced rather than silently resolved — propagates.
func (s *Service) resolveDiscount(ctx context.Context, in GetQuoteInput) (domain.DiscountRule, error) {
	facilityID, err := s.facilities.FacilityIDForCourt(ctx, in.CourtID)
	if err != nil {
		if errors.Is(err, domain.ErrFacilityNotFound) {
			return domain.DiscountRule{}, nil
		}
		return domain.DiscountRule{}, err
	}

	discountRules, err := s.discountRepo.ListForFacility(ctx, facilityID)
	if err != nil {
		return domain.DiscountRule{}, err
	}
	return domain.ResolveDiscount(discountRules, facilityID, in.Source, in.Range.Start)
}

// CreateDiscountRuleInput is CreateDiscountRule's use-case input.
//
// Creation-RPC checklist (T10 retro finding 3, sprint plan A4), both items
// checked here rather than assumed:
//  1. There is deliberately no ID field — the ID is minted server-side from
//     the IDGenerator port, so no caller can choose or squat a DiscountRule's
//     permanent identifier.
//  2. No field here gates authorization or privilege on the actor.
//     DiscountType/Amount/AppliesTo/StartsAt/EndCondition all describe the
//     discount; the only actor-shaped field, ActorUserID, is checked against
//     the Facility's real OwnerID via port.FacilityLookup rather than being
//     believed. A caller cannot self-declare anything that makes them an
//     owner. Recorded as a checked negative finding.
type CreateDiscountRuleInput struct {
	FacilityID string
	// ActorUserID is a caller-supplied claim, not a verified identity — see
	// domain.ErrNotFacilityOwner and HANDOFF.md's Auth cross-cutting item.
	// The object-level (BOLA) check it feeds is real regardless: it is
	// resolved against the Facility's actual OwnerID.
	ActorUserID  string
	DiscountType domain.DiscountType
	Amount       domain.DiscountAmount
	AppliesTo    []domain.Source
	StartsAt     time.Time
	EndCondition domain.EndCondition
}

// CreateDiscountRule authorizes the actor against the target Facility, then
// validates and persists a new DiscountRule (T11.2).
//
// The ownership check runs *before* the rule is constructed or persisted,
// mirroring facilities' AddCourt (T7.7) exactly: a rejected actor has no code
// path to the repository at all. That ordering is the AC, not an
// implementation detail — see the app-layer test that asserts the repository
// call counter is untouched, since asserting only the returned error would
// pass even if the write had happened first.
func (s *Service) CreateDiscountRule(ctx context.Context, in CreateDiscountRuleInput) (domain.DiscountRule, error) {
	// A malformed FacilityID is answered exactly like an unknown one, the
	// same T10.7/PR #89 Layer 2 guard facilities' own AddCourt applies to
	// this field — and it fires before the lookup, so a non-UUID never
	// reaches the Facilities adapter's mustUUID.
	if !uuidShape.MatchString(in.FacilityID) {
		return domain.DiscountRule{}, domain.ErrFacilityNotFound
	}

	if err := s.facilities.EnsureFacilityOwner(ctx, in.FacilityID, in.ActorUserID); err != nil {
		return domain.DiscountRule{}, err
	}

	rule, err := domain.NewDiscountRule(
		s.ids.NewID(),
		in.FacilityID,
		in.DiscountType,
		in.Amount,
		in.AppliesTo,
		in.StartsAt,
		in.EndCondition,
	)
	if err != nil {
		return domain.DiscountRule{}, err
	}

	return s.discountRepo.Create(ctx, rule)
}

// ListDiscountRulesForFacility returns every DiscountRule configured for
// facilityID (T11.2). It is an unauthenticated, list-shaped read, matching
// ListCourtBookings: the discounted price is something a Player is quoted
// anyway, so the rules behind it are not owner-only information, and no
// ownership check is applied here. Creating one is the privileged operation.
func (s *Service) ListDiscountRulesForFacility(ctx context.Context, facilityID string) ([]domain.DiscountRule, error) {
	// A malformed facilityID is answered exactly like an unknown one. This
	// read is list-shaped — an unknown Facility has no discount rules rather
	// than being an error — so a malformed Facility must yield an empty list
	// too, matching ListCourtBookings' identical convention.
	if !uuidShape.MatchString(facilityID) {
		return []domain.DiscountRule{}, nil
	}
	return s.discountRepo.ListForFacility(ctx, facilityID)
}
