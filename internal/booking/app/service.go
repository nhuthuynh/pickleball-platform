package app

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
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
	repo          port.Repository
	pricingRepo   port.PricingRuleRepository
	discountRepo  port.DiscountRuleRepository
	recurringRepo port.RecurringHireRepository
	facilities    port.FacilityLookup
	identity      port.IdentityLookup
	ids           port.IDGenerator
}

// ServiceOptions is the dependency bundle for NewService (T13.8, closing
// issue #123).
//
// The constructor was positional and reached seven parameters in T11.5, which
// is where its own doc comment flagged the migration to the shape
// internal/competitions/app and internal/payments/app already use. This is
// that migration. The shape follows paymentsapp.ServiceOptions, the older of
// the two, because it is the one that has already absorbed a cross-context
// dependency (T6.5's RegistrationUpdater) without touching a single existing
// call site — which is the whole property being bought here.
//
// **Unlike Payments' ServiceOptions, no field here is optional.** Payments has
// genuinely optional reconciliation hooks with nil guards at their use sites;
// Booking has none — every field below is dereferenced unconditionally by at
// least one use case, and no method in this package guards any of them. That
// is why NewService validates rather than documenting which fields a caller
// may skip: there are none.
type ServiceOptions struct {
	// Bookings is the Booking aggregate's persistence port, used by every
	// write use case and both list reads.
	Bookings port.Repository
	// PricingRules backs GetQuote.
	PricingRules port.PricingRuleRepository
	// DiscountRules backs CreateDiscountRule and GetQuote's discount pass.
	DiscountRules port.DiscountRuleRepository
	// RecurringHires backs the RequestRecurringHire / ApproveRecurringHire /
	// RejectRecurringHire / ListRecurringHireTemplatesForFacility set (T11.5).
	RecurringHires port.RecurringHireRepository
	// Facilities resolves court→facility and answers facility-ownership
	// questions for the authorization checks.
	Facilities port.FacilityLookup
	// Identity resolves a verified IdP subject to a User.ID and answers club
	// role questions — ADR-0014's seam (T13.2).
	Identity port.IdentityLookup
	// IDs mints every Booking and RecurringHireTemplate ID.
	IDs port.IDGenerator
}

// ErrMissingDependency reports that a ServiceOptions left at least one
// required dependency unusable (unset, or holding a nil pointer).
//
// A sentinel so a caller that wants to react to this specific misconfiguration
// — rather than to any error Validate might grow later — can do so without
// matching on message text. Same reasoning as
// internal/platform/auth.ErrVerifierRequired.
var ErrMissingDependency = errors.New("booking/app: ServiceOptions is missing a required dependency")

// Validate reports every required dependency opts leaves unusable, or nil if
// the bundle is complete.
//
// # Why this exists at all
//
// The positional constructor this replaced made omitting a dependency a
// compile error: seven parameters meant seven arguments. A struct gives that
// up — a forgotten field is silently the zero value, and an interface's zero
// value is nil. Left unguarded, this refactor would have traded a build
// failure for a nil-pointer dereference on whichever request first reached the
// missing dependency, with nothing in the panic naming which of the seven was
// never wired. That is a strictly worse failure than the one being removed.
// This function moves the property from the compiler to construction time
// rather than dropping it, which is the same argument
// internal/platform/auth.EnsureVerifierConfigured makes for a nil
// TokenVerifier (T13.5, issue #136): the misconfiguration is made at wiring
// time, so wiring time is where it should be reported.
//
// # Why it reports all of them
//
// Returning the first missing field would turn a seven-field mis-wiring into
// seven edit-build-run cycles. The set is cheap to compute and the whole set
// is what the caller needs.
func (o ServiceOptions) Validate() error {
	var missing []string
	for _, dep := range []struct {
		name  string
		value any
	}{
		{"Bookings", o.Bookings},
		{"PricingRules", o.PricingRules},
		{"DiscountRules", o.DiscountRules},
		{"RecurringHires", o.RecurringHires},
		{"Facilities", o.Facilities},
		{"Identity", o.Identity},
		{"IDs", o.IDs},
	} {
		if isUnusable(dep.value) {
			missing = append(missing, dep.name)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("%w: %s", ErrMissingDependency, strings.Join(missing, ", "))
}

// isUnusable reports whether a dependency is unset or holds a nil pointer.
//
// The second case is the one a plain `== nil` misses and the reason this is
// not a one-liner: an interface holding a typed nil is itself non-nil, so it
// walks past the easy check and panics on its first method call anyway — the
// exact fail-late shape Validate exists to stop. Lifted from
// internal/platform/auth.verifierIsNil (T13.5), which makes the same argument
// for the same reason; duplicated rather than shared because no app package
// imports internal/platform (the per-context duplication convention uuidShape
// already follows in this file).
func isUnusable(dep any) bool {
	if dep == nil {
		return true
	}
	switch v := reflect.ValueOf(dep); v.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan:
		return v.IsNil()
	default:
		// A non-pointer implementation (several of this package's test fakes
		// are structs, and internal/platform/idgen.UUID is a value type)
		// cannot be nil.
		return false
	}
}

// NewService wires the Booking context's application layer.
//
// It panics if opts is incomplete. A missing dependency is a wiring bug in
// cmd/server or in a test, never a runtime condition a caller could recover
// from, and returning an error would put an `if err != nil` at ~40 call sites
// to report something none of them can do anything about but crash. Panicking
// at construction keeps the failure at the moment and place the mistake was
// made: cmd/server builds every Service before it serves its first request, so
// this stops the process at boot rather than on the one endpoint that needed
// the dependency. Callers that want the error rather than the panic can call
// opts.Validate() themselves.
func NewService(opts ServiceOptions) *Service {
	if err := opts.Validate(); err != nil {
		panic(err.Error())
	}
	return &Service{
		repo:          opts.Bookings,
		pricingRepo:   opts.PricingRules,
		discountRepo:  opts.DiscountRules,
		recurringRepo: opts.RecurringHires,
		facilities:    opts.Facilities,
		identity:      opts.Identity,
		ids:           opts.IDs,
	}
}

// ResolveActorUserID translates a verified IdP subject into the caller's
// User.ID (uuid) — ADR-0014's resolution seam, and the fix for issues #146
// and #152.
//
// **This is the only method in internal/booking/app whose parameter is a
// subject, and the name says so deliberately.** ADR-0014's invariant is that
// below the grpcapi boundary an actor value is always a User.ID; this method
// is the one place that invariant is established rather than assumed. Every
// other actor-taking method here — RequestRecurringHire, CreateDiscountRule,
// ApproveRecurringHire, RejectRecurringHire, both list reads — takes the
// resolved uuid, which is what makes their uuidShape guards honest
// malformed-input checks again rather than the every-caller-rejecting checks
// #152 found.
//
// It is called from the grpcapi handler's actor() funnel, once per
// authenticated RPC, so a subject never reaches the two places that would
// mishandle it: a uuid column (the Postgres adapter's mustUUID panics) and a
// uuid-keyed comparison (an owner check that can never match).
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

// CreateBookingInput is the use-case input for creating any of the four
// Booking sources (D3b) — the same use case serves a direct court booking, a
// Game reserving its courts, a Competition reserving its courts, and a Club
// recurring-hire occurrence, differing only in Source/ReferenceID.
type CreateBookingInput struct {
	CourtID string
	Source  domain.Source
	Range   domain.TimeRange
	// OwnerUserID is the User.ID (uuid, never a subject — ADR-0014) the new
	// Booking belongs to. Required since DECISION D1 (ADR-0015 option (a));
	// domain.NewBooking rejects an empty one.
	//
	// Every caller supplies it from a fact it already holds rather than from
	// anything the requester sent: the grpcapi handler from the verified
	// principal via Handler.actor, Social Play from the Game's HostID,
	// Competitions from the Competition's HostID, and ApproveRecurringHire
	// from the template's RequestedByUserID. There is deliberately no path
	// by which a caller names the owner of a booking it is not making — that
	// would re-open #144 through the front door.
	OwnerUserID string
	ReferenceID string
}

// CreateBooking validates the candidate booking, pre-checks it against the
// court's other active bookings (domain.EnsureNoConflict — regardless of
// their source, per D3b/F1), and persists it. The Postgres adapter's EXCLUDE
// constraint remains the authoritative guard under concurrent requests (see
// HANDOFF.md T4); this pre-check exists to fail fast and give a clear domain
// error without waiting on a round trip that's doomed to be rejected.
func (s *Service) CreateBooking(ctx context.Context, in CreateBookingInput) (domain.Booking, error) {
	candidate, err := domain.NewBooking(s.ids.NewID(), in.CourtID, in.Source, in.Range, in.ReferenceID, in.OwnerUserID)
	if err != nil {
		return domain.Booking{}, err
	}

	// A malformed (non-empty, wrong-shape) CourtID is rejected here, before
	// either repository call, with domain.ErrInvalidCourtReference (T10.7
	// follow-up, closing issue #97): ListActiveForCourt below is the same
	// mustUUID-backed adapter method ListCourtBookings' own already-guarded
	// read calls, and a malformed CourtID reached it unguarded — panicking
	// there, one step before Create's own FK-violation path.
	//
	// This guard is a *shape* check and only a shape check: a well-formed
	// UUID naming no courts row passes it, reaches Create, and is caught by
	// the FK on bookings.court_id, which adapter/postgres translates to this
	// same sentinel (T15.6, closes #185). Both halves answer NotFound. That
	// is a deliberate change from T10.7, which matched the FK path's
	// then-unclassified Internal; see the sentinel's doc comment in
	// domain/errors.go. Note in particular that this guard is NOT an
	// existence check and must not be mistaken for one — the FK is the
	// authoritative half (CLAUDE.md rule 4's shape), because any app-level
	// existence pre-check would still race the INSERT.
	//
	// An empty CourtID is unaffected: domain.NewBooking's own
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
func (s *Service) CancelBooking(ctx context.Context, bookingID, actorUserID string) (domain.Booking, error) {
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

	// DECISION D1 (ADR-0015 option (a); closes #144). The existence check
	// above runs FIRST and the ownership check second, deliberately: an
	// unknown id answers NotFound for everybody, and a known id answers
	// PermissionDenied only to a caller who is not its owner.
	//
	// That ordering does leak one bit — whether a given booking id exists —
	// to an authenticated caller who guesses one. It is the same bit
	// ErrNotFacilityOwner already leaks for Facilities, and it is the lesser
	// of the two evils available: the alternative (answer NotFound to
	// non-owners) would mean a legitimate owner hitting a transient
	// repository miss cannot tell "your booking is gone" from "you may not
	// touch it". ADR-0014 §6 settled the same trade the same way.
	if err := b.EnsureOwner(actorUserID); err != nil {
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
	// ActorUserID is the VERIFIED caller's User.ID (uuid) — resolved from
	// auth.Principal.Subject by the grpcapi handler's actor() funnel, per
	// ADR-0014. It is neither a caller-supplied claim (T12.7 removed the
	// request field) nor a subject (ADR-0014 keeps subjects above the
	// boundary).
	//
	// The object-level (BOLA) check it feeds compares it against the
	// Facility's actual OwnerID — a uuid column — which is why it must be
	// the resolved id and not the subject: the comparison could otherwise
	// never match, silently denying every real owner.
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
