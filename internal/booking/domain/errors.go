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

	// ErrInvalidCourtReference is CreateBooking's malformed-CourtID guard
	// sentinel (T10.7, closing issue #97). It deliberately maps to Internal
	// (the default, unclassified case in adapter/grpcapi's toStatus — see
	// that switch's comment), matching this codebase's own existing,
	// pre-T10.7 behavior for a well-formed-but-*unknown* CourtID: Postgres
	// adapter's Create INSERTs against bookings.court_id REFERENCES
	// courts(id), so an unknown court hits a 23503 FK violation that
	// translateErr does not specifically classify, and it falls through to
	// the same default Internal today. This is a real, disclosed gap (a
	// clean NotFound would be better — see the PR description's scope
	// note), but not this ticket's to fix: T10.7 only needs a malformed
	// *shape* (e.g. "not-a-uuid") to answer no worse than an unknown
	// well-formed one already does, without panicking to get there. A
	// malformed shape previously reached Repository.ListActiveForCourt's
	// mustUUID and panicked before Create was ever called; this sentinel is
	// what CreateBooking now returns instead, from the app-layer guard,
	// before either repository call happens.
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
	ErrUserNotFound = errors.New("booking: user not found")
	ErrNotClub      = errors.New("booking: actor does not hold the club role")
)
