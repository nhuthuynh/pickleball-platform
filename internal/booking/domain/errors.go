package domain

import "errors"

// Domain errors. Adapters must translate infrastructure failures (e.g. a
// Postgres 23P01 exclusion-constraint violation) into these — upper layers
// only ever see errors from this file.
var (
	ErrInvalidTimeRange             = errors.New("booking: invalid time range")
	ErrInvalidClockTime             = errors.New("booking: invalid clock time")
	ErrInvalidSource                = errors.New("booking: invalid source")
	ErrEmptyCourtID                 = errors.New("booking: court id is required")
	ErrCourtDoubleBooked            = errors.New("booking: court already booked for an overlapping time range")
	ErrIllegalStatusTransition      = errors.New("booking: illegal status transition")
	ErrBookingNotFound              = errors.New("booking: not found")
	ErrNoPricingRule                = errors.New("booking: no pricing rule matches the requested slot")
	ErrAmbiguousPricingRule         = errors.New("booking: more than one pricing rule matches the requested slot")
	ErrPricingSlotSpansMultipleDays = errors.New("booking: pricing slot must fall within a single calendar day")

	// ErrInvalidCourtReference is the single answer to "this CourtID does not
	// name a court this context can use", raised from BOTH of the two places
	// that can discover it:
	//
	//   - app.Service.CreateBooking's uuidShape guard, for a malformed id
	//     (T10.7, issue #97). A value that is not a canonical UUID cannot
	//     name any court that exists or ever will.
	//   - adapter/postgres.translateErr, on a 23503 foreign_key_violation
	//     from bookings.court_id REFERENCES courts(id), for a well-formed id
	//     naming no row (T15.6, issue #185). A shape check cannot detect
	//     this; only the FK can.
	//
	// It maps to codes.NotFound (adapter/grpcapi's toStatus).
	//
	// T15.6 changed both halves of that, and the reasoning is worth keeping
	// because the previous comment here recorded the opposite decision.
	// T10.7 mapped this sentinel to Internal *deliberately*, to match the
	// unknown-id path: that path produced an unclassified 23503 and answered
	// Internal, and T10.7's scope was only to stop a malformed id panicking,
	// not to answer better than the unknown case already did. That made
	// Internal the correct choice for consistency at the time and the wrong
	// answer in absolute terms, as that comment said in as many words ("a
	// clean NotFound would be better"). T15.6 fixes the path it was matching,
	// so the reason to stay on Internal is gone with it.
	//
	// Why one sentinel for both halves rather than a second one for the
	// unknown case: this file's own precedent, twice. ErrFacilityNotFound
	// ("an unknown or malformed FacilityID ... the app layer answers a
	// malformed one exactly like an unknown one") and
	// ErrRecurringHireTemplateNotFound ("an unknown or malformed template
	// id") already collapse exactly this distinction, and CLAUDE.md rule 7
	// wants one name for one concept. Splitting them would also hand a
	// caller a distinction the codebase deliberately denies elsewhere,
	// for no gain: both halves mean the same thing to a client and warrant
	// the same fix on their side.
	//
	// Why NotFound rather than InvalidArgument: it is what this codebase
	// answers for "the request names something that does not exist" —
	// ErrFacilityNotFound and ErrRecurringHireTemplateNotFound here,
	// ErrGameNotFound in socialplay, ErrCompetitionNotFound in competitions.
	// InvalidArgument is reserved for request *shape* problems that need no
	// lookup to reject (ErrInvalidTimeRange, and socialplay/competitions'
	// ErrMalformedCourtID on their own repeated court_ids field). The
	// malformed half is genuinely shape-shaped and could argue for
	// InvalidArgument on its own, but it is subordinate here: it shares a
	// sentinel with the unknown half by the paragraph above, and a shared
	// sentinel gets one code. Client impact of the difference is nil over
	// REST for the malformed case only in the sense that both are 4xx — 404
	// vs 400 — and 404 is the honest answer to "no such court".
	//
	// No enumeration oracle is created by answering NotFound here, and this
	// was checked rather than assumed (T15.6, #185's note): court ids are
	// already public. facilities' ListFacilities and GetFacility are both
	// absent from that context's AuthenticatedMethods — deliberately, as the
	// browse path — and GetFacilityResponse carries `repeated Court courts`
	// (T8.2). Anyone who can call CreateBooking can already enumerate every
	// court id from the public facility detail page, so "no such court"
	// discloses nothing new.
	ErrInvalidCourtReference = errors.New("booking: court id does not reference a usable court")

	// DiscountRule errors (T11.1). ErrInvalidSource (above) is reused for an
	// AppliesTo entry outside the four locked Source values, rather than
	// adding a second sentinel for the same underlying condition — CLAUDE.md
	// rule 7's "one ubiquitous language": an invalid Source is an invalid
	// Source everywhere, whether it's Booking.Source or DiscountRule.AppliesTo.
	ErrEmptyFacilityID                = errors.New("booking: facility id is required")
	ErrInvalidDiscountType            = errors.New("booking: invalid discount type")
	ErrInvalidDiscountAmount          = errors.New("booking: discount amount out of range for its discount type")
	ErrEmptyAppliesTo                 = errors.New("booking: discount rule must apply to at least one source")
	ErrInvalidEndConditionOccurrences = errors.New("booking: end-after-occurrences count must be positive")
	ErrAmbiguousDiscountRule          = errors.New("booking: more than one discount rule matches the requested facility/source/time")

	// ErrFacilityNotFound and ErrNotFacilityOwner are Booking's own,
	// context-local sentinels for the two answers port.FacilityLookup can
	// give (T11.2, Booking's first call into Facilities). They are
	// deliberately NOT internal/facilities/domain's sentinels of the same
	// name: CLAUDE.md rule 5 requires the adapter to translate at the
	// boundary so no facilitiesdomain error type ever reaches Booking's app
	// or grpcapi layers — the identical reasoning
	// internal/socialplay/domain.ErrFacilityNotFound's own comment gives
	// for keeping a per-context copy rather than importing one.
	//
	// T17.5 (issue #195) adds a second raise site for ErrFacilityNotFound,
	// deliberately reusing rather than adding a parallel sentinel:
	// adapter/postgres.translateRecurringHireErr, on a 23503
	// foreign_key_violation from recurring_hire_templates.court_id
	// REFERENCES courts(id). RequestRecurringHire's app-level guard
	// (FacilityIDForCourt) already answers this sentinel for "no such
	// Court" in the non-racing case; the FK only fires when a concurrent
	// delete removes the Court between that guard's read and this INSERT —
	// the same narrow-window shape #185/T15.6 fixed for bookings.court_id,
	// on a call site guarded by a read instead of unguarded.
	ErrFacilityNotFound = errors.New("booking: facility not found")
	ErrNotFacilityOwner = errors.New("booking: actor is not authorized to modify this facility")

	// RecurringHireTemplate sentinels (T11.4). Named with a RecurringHire
	// prefix so they cannot collide as Go identifiers with T11.1's
	// DiscountRule sentinels landing in this same errors.go file (e.g. its
	// own "EndAfterOccurrences n <= 0" validation) — see A9's shared-kernel
	// note and A10's same-package watch item in
	// docs/process/t11-sprint-plan.md.
	ErrEmptyRequestedByUserID                  = errors.New("booking: requested by user id is required")
	ErrInvalidRecurringHireTimeRange           = errors.New("booking: recurring hire start time must be before end time")
	ErrInvalidRecurringHireEndAfterOccurrences = errors.New("booking: recurring hire end-after-occurrences count must be positive")
	ErrInvalidRecurringHireStatusTransition    = errors.New("booking: invalid recurring hire status transition")

	// ErrRecurringHireTemplateNotFound is the miss sentinel for
	// port.RecurringHireRepository's reads (T11.5). It is deliberately not
	// ErrBookingNotFound: a template and a Booking are different aggregates,
	// and reusing one not-found sentinel across both would make a future
	// `:one` template read indistinguishable from a missing Booking at the
	// gRPC boundary — the same reasoning translateDiscountErr's own comment
	// gives for staying separate from translateErr.
	ErrRecurringHireTemplateNotFound = errors.New("booking: recurring hire template not found")

	// ErrUserNotFound and ErrNotClub are Booking's own, context-local
	// sentinels for the two answers port.IdentityLookup can give (T11.5,
	// Booking's first call into Identity/Users). Like ErrFacilityNotFound and
	// ErrNotFacilityOwner above, they are deliberately NOT
	// internal/identity/domain's sentinels: CLAUDE.md rule 5 requires the
	// adapter to translate at the boundary so no identitydomain error type
	// ever reaches Booking's app or grpcapi layers.
	//
	// Both map to PermissionDenied at the gRPC boundary, not NotFound — see
	// adapter/grpcapi's toStatus. An actor who cannot be resolved to a real
	// User has not proven the `club` role any more than a resolved non-club
	// actor has, and answering NotFound for the unresolvable case would turn
	// RequestRecurringHire, an unauthenticated endpoint, into a
	// user-enumeration oracle that reports which user ids exist.
	//
	// T17.5 (issue #195) adds a second raise site for ErrUserNotFound:
	// adapter/postgres.translateRecurringHireErr, on a 23503
	// foreign_key_violation from
	// recurring_hire_templates.requested_by_user_id REFERENCES
	// identity_users(id). RequestRecurringHire's app-level guard
	// (EnsureClubRole, issue #164's call site) already answers this
	// sentinel for "no such User" in the non-racing case; the FK only fires
	// when a concurrent delete removes the User between that guard's read
	// and this INSERT. Reused rather than a parallel sentinel, per the same
	// reasoning ErrFacilityNotFound's comment above gives.
	ErrUserNotFound = errors.New("booking: user not found")
	ErrNotClub      = errors.New("booking: actor does not hold the club role")
)
