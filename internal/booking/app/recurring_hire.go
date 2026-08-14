package app

import (
	"context"
	"errors"
	"time"

	"github.com/nhuthuynh/white-label/internal/booking/domain"
)

// recurringHireHorizon bounds how far ahead an approval generates
// occurrences, and is the `upTo` safety cap T11.4's GenerateOccurrences
// requires its caller to supply — without one, a NoEnd (open-ended) template
// would ask for an unbounded slice.
//
// It is measured from the template's own StartsAt rather than from
// time.Now(), so approving the same template twice in different weeks would
// generate the same weeks: the app layer stays free of a clock dependency,
// and the test fixtures below assert real dates rather than offsets from
// whenever the suite happens to run.
//
// One year is a decision this ticket had to make — T11.5's text specifies the
// per-occurrence conflict behaviour but not the horizon. It means an
// open-ended "permanent club rate" materialises as a year of Bookings on
// approval, and extending it (a renewal job, or generating lazily) is a
// follow-up, deliberately not an unbounded write here.
const recurringHireHorizonYears = 1

// RequestRecurringHireInput is RequestRecurringHire's use-case input.
//
// Creation-RPC checklist (T10 retro finding 3, sprint plan A4), both items
// checked here rather than assumed:
//  1. There is deliberately no ID field — the ID is minted server-side from
//     the IDGenerator port, so no caller can choose or squat a template's
//     permanent identifier.
//  2. Role self-assignment — **this is the item that fires for this ticket**.
//     There is deliberately no IsClub/Role field. Whether the actor may
//     request a recurring hire as a Club is resolved server-side through
//     port.IdentityLookup against the User's real Roles; ActorUserID is the
//     only actor-shaped field and it is a subject to look up, never a
//     privilege to believe.
type RequestRecurringHireInput struct {
	// ActorUserID is a caller-supplied claim, not a verified identity — see
	// HANDOFF.md's Auth cross-cutting item. The role check it feeds is real
	// regardless: it is resolved against the User's actual Roles.
	ActorUserID  string
	CourtID      string
	Weekday      time.Weekday
	StartTime    domain.ClockTime
	EndTime      domain.ClockTime
	StartsAt     time.Time
	EndCondition domain.RecurringHireEndCondition
}

// OccurrenceOutcome is what happened to one generated occurrence when a
// template was approved.
type OccurrenceOutcome string

const (
	// OccurrenceBooked means a real SourceRecurringHire Booking now exists
	// for this slot.
	OccurrenceBooked OccurrenceOutcome = "booked"
	// OccurrenceSkippedConflict means the court was already booked for this
	// slot (domain.ErrCourtDoubleBooked). This is an ordinary, expected
	// outcome — not a failure of the approval.
	OccurrenceSkippedConflict OccurrenceOutcome = "skipped_conflict"
	// OccurrenceSkippedError means this occurrence could not be booked for
	// any other reason. Kept distinct from OccurrenceSkippedConflict on
	// purpose: an already-booked Tuesday and a backend fault are different
	// facts, and a Facility Owner reading the result must be able to tell
	// which one they are looking at.
	OccurrenceSkippedError OccurrenceOutcome = "skipped_error"
)

// OccurrenceResult reports one occurrence's fate. Reason is empty for a
// booked occurrence and carries the underlying error's message otherwise.
type OccurrenceResult struct {
	Range     domain.TimeRange
	Outcome   OccurrenceOutcome
	BookingID string
	Reason    string
}

// ApproveRecurringHireResult is what ApproveRecurringHire answers with: the
// approved template plus one entry per generated occurrence, in chronological
// order, whether or not it became a Booking.
type ApproveRecurringHireResult struct {
	Template    domain.RecurringHireTemplate
	Occurrences []OccurrenceResult
}

// RequestRecurringHire records a Club's request for a standing weekly slot
// (T11.5).
//
// Order of checks is the AC, not an implementation detail:
//  1. the actor's `club` role, resolved from Identity — a caller who is not a
//     Club never reaches the Court lookup or the repository at all;
//  2. the Court, resolved to its Facility;
//  3. the template's own invariants (T11.4's constructor);
//  4. the write.
//
// A Court that resolves to no Facility is rejected rather than accepted. That
// is deliberate: approval requires a Facility Owner (see
// ApproveRecurringHire), so a template on an ownerless Court — an unknown
// Court, or one of 0001_init.sql's pre-Facilities seeded Courts whose
// courts.facility_id is still NULL — could never be approved by anyone.
// Accepting it would be recording a request that is guaranteed to sit
// unanswered forever.
func (s *Service) RequestRecurringHire(ctx context.Context, in RequestRecurringHireInput) (domain.RecurringHireTemplate, error) {
	// A malformed ActorUserID is answered exactly like an unknown one, the
	// same T10.7/PR #89 Layer 2 guard CreateDiscountRule applies to its own
	// FacilityID — and it fires before the lookup, so a non-UUID never
	// reaches identity_users.id, a `uuid` column.
	if !uuidShape.MatchString(in.ActorUserID) {
		return domain.RecurringHireTemplate{}, domain.ErrUserNotFound
	}
	if err := s.identity.EnsureClubRole(ctx, in.ActorUserID); err != nil {
		return domain.RecurringHireTemplate{}, err
	}

	if !uuidShape.MatchString(in.CourtID) {
		return domain.RecurringHireTemplate{}, domain.ErrFacilityNotFound
	}
	if _, err := s.facilities.FacilityIDForCourt(ctx, in.CourtID); err != nil {
		return domain.RecurringHireTemplate{}, err
	}

	template, err := domain.NewRecurringHireTemplate(
		s.ids.NewID(),
		in.ActorUserID,
		in.CourtID,
		in.Weekday,
		in.StartTime,
		in.EndTime,
		in.StartsAt,
		in.EndCondition,
	)
	if err != nil {
		return domain.RecurringHireTemplate{}, err
	}

	return s.recurringRepo.Create(ctx, template)
}

// ApproveRecurringHire approves a requested template and turns the
// occurrences it implies into real SourceRecurringHire Bookings (T11.5).
//
// **A per-occurrence conflict does not fail the approval.** Each occurrence is
// attempted independently and reported independently: a single already-booked
// Tuesday must not block an entire multi-week request, which is what an
// abort-on-first-conflict implementation would do. The template transitions to
// approved regardless of how many occurrences were skipped, and the caller
// gets one OccurrenceResult per occurrence saying which weeks were booked,
// which were skipped, and why.
//
// A non-conflict failure on one occurrence is likewise recorded rather than
// returned, as OccurrenceSkippedError. That is a deliberate trade: returning
// it would discard the report for every other occurrence *and* strand the
// caller, since the template is already approved by then and a retry would
// only answer FailedPrecondition. The distinct outcome value is what keeps a
// backend fault from being silently indistinguishable from an ordinary
// double-booking.
//
// Ownership is checked first, before the status transitions and before any
// Booking is created — the same ordering T11.2's CreateDiscountRule
// established, asserted by a call-counter test rather than by the returned
// error alone.
func (s *Service) ApproveRecurringHire(ctx context.Context, templateID, actorUserID string) (ApproveRecurringHireResult, error) {
	template, err := s.authorizeRecurringHireDecision(ctx, templateID, actorUserID)
	if err != nil {
		return ApproveRecurringHireResult{}, err
	}

	if err := template.Approve(); err != nil {
		return ApproveRecurringHireResult{}, err
	}

	// The approval is persisted before any occurrence is attempted, so the
	// Facility Owner's decision is durable independently of how the
	// individual weeks turn out — the ticket's "the template still
	// transitions to approved regardless" is a property of the stored
	// template, not just of the returned value.
	approved, err := s.recurringRepo.UpdateStatus(ctx, template)
	if err != nil {
		return ApproveRecurringHireResult{}, err
	}

	horizon := template.StartsAt.AddDate(recurringHireHorizonYears, 0, 0)
	occurrences := domain.GenerateOccurrences(template, horizon)

	results := make([]OccurrenceResult, 0, len(occurrences))
	for _, occ := range occurrences {
		results = append(results, s.bookOccurrence(ctx, template, occ))
	}

	return ApproveRecurringHireResult{Template: approved, Occurrences: results}, nil
}

// bookOccurrence attempts one occurrence and classifies the outcome. It goes
// through Service.CreateBooking rather than the repository directly, so a
// recurring-hire occurrence is subject to exactly the same no-double-booking
// invariant (domain.EnsureNoConflict plus the Postgres EXCLUDE constraint) as
// every other Booking source — D3b's whole point.
func (s *Service) bookOccurrence(ctx context.Context, template domain.RecurringHireTemplate, occ domain.TimeRange) OccurrenceResult {
	booking, err := s.CreateBooking(ctx, CreateBookingInput{
		CourtID: template.CourtID,
		Source:  domain.SourceRecurringHire,
		Range:   occ,
		// The template is the Booking's reference, which is what makes the
		// generated Bookings traceable back to the request that created them.
		ReferenceID: template.ID,
	})
	switch {
	case err == nil:
		return OccurrenceResult{Range: occ, Outcome: OccurrenceBooked, BookingID: booking.ID}
	case errors.Is(err, domain.ErrCourtDoubleBooked):
		return OccurrenceResult{Range: occ, Outcome: OccurrenceSkippedConflict, Reason: err.Error()}
	default:
		return OccurrenceResult{Range: occ, Outcome: OccurrenceSkippedError, Reason: err.Error()}
	}
}

// RejectRecurringHire rejects a requested template (T11.5). Same ownership
// check and same ordering as ApproveRecurringHire; no Bookings are ever
// created on this path.
func (s *Service) RejectRecurringHire(ctx context.Context, templateID, actorUserID string) (domain.RecurringHireTemplate, error) {
	template, err := s.authorizeRecurringHireDecision(ctx, templateID, actorUserID)
	if err != nil {
		return domain.RecurringHireTemplate{}, err
	}

	if err := template.Reject(); err != nil {
		return domain.RecurringHireTemplate{}, err
	}

	return s.recurringRepo.UpdateStatus(ctx, template)
}

// authorizeRecurringHireDecision resolves templateID and proves actorUserID
// owns the Facility that owns the template's Court, returning the template
// only if both hold. It is the shared front half of Approve and Reject, so
// the two cannot drift apart on the check that matters.
//
// The template's Court is resolved through the *reused* T11.2
// port.FacilityLookup (sprint plan A10: T11.5 must not build a second one),
// and the ownership decision itself is delegated to
// facilitiesdomain.Facility.EnsureOwner inside that adapter — so "who owns a
// Facility" keeps exactly one definition, in the context that owns the
// concept.
func (s *Service) authorizeRecurringHireDecision(ctx context.Context, templateID, actorUserID string) (domain.RecurringHireTemplate, error) {
	// A malformed templateID is answered exactly like an unknown one (the
	// T10.7 guard, applied here because GetByID below reaches the Postgres
	// adapter's mustUUID).
	if !uuidShape.MatchString(templateID) {
		return domain.RecurringHireTemplate{}, domain.ErrRecurringHireTemplateNotFound
	}

	template, err := s.recurringRepo.GetByID(ctx, templateID)
	if err != nil {
		return domain.RecurringHireTemplate{}, err
	}

	facilityID, err := s.facilities.FacilityIDForCourt(ctx, template.CourtID)
	if err != nil {
		return domain.RecurringHireTemplate{}, err
	}
	if err := s.facilities.EnsureFacilityOwner(ctx, facilityID, actorUserID); err != nil {
		return domain.RecurringHireTemplate{}, err
	}

	return template, nil
}

// ListRecurringHireTemplatesForFacility returns every RecurringHireTemplate
// requested on any of facilityID's Courts — the Facility Owner's incoming
// requests queue (T11.5, consumed by T11.6's owner console).
//
// Unlike T11.2's ListDiscountRulesForFacility, which is deliberately
// unauthenticated because a discounted price is something a Player is quoted
// anyway, this read is owner-only: it exposes other parties' pending business
// requests — who wants which court, when, and whether the owner has answered
// yet — which is not information a Player is entitled to by virtue of being
// quoted a price. An unresolvable Facility is domain.ErrFacilityNotFound
// rather than an empty list, because there is an actor check here that must
// not silently pass for a Facility that does not exist.
//
// The ticket does not specify this endpoint's authorization; making it
// owner-only is this implementation's call, recorded here and in the PR
// rather than left implicit. Note the consequence: a Club has no RPC to read
// back its own templates yet, which T11.6's Club-facing status view will need.
func (s *Service) ListRecurringHireTemplatesForFacility(ctx context.Context, facilityID, actorUserID string) ([]domain.RecurringHireTemplate, error) {
	if !uuidShape.MatchString(facilityID) {
		return nil, domain.ErrFacilityNotFound
	}

	if err := s.facilities.EnsureFacilityOwner(ctx, facilityID, actorUserID); err != nil {
		return nil, err
	}

	courtIDs, err := s.facilities.CourtIDsForFacility(ctx, facilityID)
	if err != nil {
		return nil, err
	}
	if len(courtIDs) == 0 {
		return []domain.RecurringHireTemplate{}, nil
	}

	return s.recurringRepo.ListForCourts(ctx, courtIDs)
}
