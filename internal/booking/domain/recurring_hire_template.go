package domain

import "time"

// RecurringHireStatus is a RecurringHireTemplate's lifecycle state, a closed
// enum mirroring this project's other request/approval and paid/unpaid
// state machines (internal/payments/domain.Status's unpaid -> paid ->
// refunded, internal/competitions/domain.EntryStatus): a request starts as
// `requested`, and a Facility Owner (T11.5's job, not this file's) either
// approves or rejects it. Approving is what makes GenerateOccurrences'
// output become real Bookings — that wiring is T11.5's, this file only
// models the transition. `cancelled` exists for an approved template a Club
// later withdraws; no T11 ticket wires a transition into it yet, mirroring
// competitions.EntryStatusCancelled's own "modelled ahead of its writer"
// precedent, so there is deliberately no Cancel method here.
type RecurringHireStatus string

const (
	RecurringHireStatusRequested RecurringHireStatus = "requested"
	RecurringHireStatusApproved  RecurringHireStatus = "approved"
	RecurringHireStatusRejected  RecurringHireStatus = "rejected"
	RecurringHireStatusCancelled RecurringHireStatus = "cancelled"
)

// IsValid reports whether s is one of RecurringHireStatus's closed enum
// values.
func (s RecurringHireStatus) IsValid() bool {
	switch s {
	case RecurringHireStatusRequested, RecurringHireStatusApproved, RecurringHireStatusRejected, RecurringHireStatusCancelled:
		return true
	default:
		return false
	}
}

// RecurringHireEndKind is RecurringHireEndCondition's closed discriminator.
type RecurringHireEndKind string

const (
	RecurringHireEndNone             RecurringHireEndKind = "no_end"
	RecurringHireEndAfterDate        RecurringHireEndKind = "end_after_date"
	RecurringHireEndAfterOccurrences RecurringHireEndKind = "end_after_occurrences"
)

// RecurringHireEndCondition bounds how far into the future a
// RecurringHireTemplate implies occurrences: EndAfterDate, EndAfterOccurrences
// (e.g. "first 10 bookings"), or NoEnd (open-ended, e.g. a permanent club
// rate) — the same three-variant shape docs/design/v1-system-design.md §3.2
// specifies for T11.1's DiscountRule.
//
// This is deliberately kept as its own, independently-named type rather than
// shared with T11.1's identically-shaped variant, per A9's shared-kernel
// note in docs/process/t11-sprint-plan.md: the two land in the very same
// internal/booking/domain package (A9 Ruling 1 keeps DiscountRule there, A9
// Ruling 2 keeps RecurringHireTemplate there too), so this type's name and
// its constructor functions are namespaced ("RecurringHire...") specifically
// so the two additions cannot collide as Go identifiers, not just so they
// read as conceptually separate.
type RecurringHireEndCondition struct {
	Kind        RecurringHireEndKind
	Date        time.Time
	Occurrences int
}

// NoRecurringHireEnd is an open-ended template (e.g. a permanent club rate).
// GenerateOccurrences still bounds it by its upTo safety cap — NoEnd must
// never be asked to generate an unbounded slice.
func NoRecurringHireEnd() RecurringHireEndCondition {
	return RecurringHireEndCondition{Kind: RecurringHireEndNone}
}

// EndRecurringHireAfterDate stops occurrence generation after date,
// inclusive of the last occurrence starting on date's own calendar day.
func EndRecurringHireAfterDate(date time.Time) RecurringHireEndCondition {
	return RecurringHireEndCondition{Kind: RecurringHireEndAfterDate, Date: date}
}

// EndRecurringHireAfterOccurrences stops generation after n occurrences. n
// must be positive — "first 0 bookings" isn't a request, it's a no-op the
// constructor rejects rather than silently accepting.
func EndRecurringHireAfterOccurrences(n int) (RecurringHireEndCondition, error) {
	if n <= 0 {
		return RecurringHireEndCondition{}, ErrInvalidRecurringHireEndAfterOccurrences
	}
	return RecurringHireEndCondition{Kind: RecurringHireEndAfterOccurrences, Occurrences: n}, nil
}

// RecurringHireTemplate is a Club's request for a standing weekly slot on a
// court. It is the source of RecurringHireStatusApproved case
// SourceRecurringHire Bookings: GenerateOccurrences reads it to produce the
// concrete date/time slots it implies, but the template itself never
// becomes a Booking — turning an occurrence into a real, persisted Booking
// (once approved) is T11.5's job, not this domain's.
//
// Weekday reuses time.Weekday, the same convention pricing_rules.weekdays
// already established (a logged, accepted convention per HANDOFF.md's
// Cross-cutting note, not something to change here). StartTime/EndTime are
// time-of-day (ClockTime, the same type PricingRule's own Start/End window
// already uses), not calendar timestamps — StartsAt is what anchors the
// pattern to an actual first occurrence date.
type RecurringHireTemplate struct {
	ID                string
	RequestedByUserID string
	CourtID           string
	Weekday           time.Weekday
	StartTime         ClockTime
	EndTime           ClockTime
	StartsAt          time.Time
	EndCondition      RecurringHireEndCondition
	Status            RecurringHireStatus
}

// NewRecurringHireTemplate constructs a requested RecurringHireTemplate,
// validating the invariants that need only the template's own fields — no
// conflict checking against other templates or Bookings here, mirroring how
// NewBooking/EnsureNoConflict and NewPayment/EnsureOnePaymentPerPayable
// split constructor validation from cross-aggregate checks.
//
// id is supplied by the caller (the app layer's IDGenerator port), the same
// pattern NewBooking and NewPayment use — RecurringHireTemplate does not
// generate its own identity.
func NewRecurringHireTemplate(
	id, requestedByUserID, courtID string,
	weekday time.Weekday,
	startTime, endTime ClockTime,
	startsAt time.Time,
	endCondition RecurringHireEndCondition,
) (RecurringHireTemplate, error) {
	if requestedByUserID == "" {
		return RecurringHireTemplate{}, ErrEmptyRequestedByUserID
	}
	if courtID == "" {
		return RecurringHireTemplate{}, ErrEmptyCourtID
	}
	if startTime >= endTime {
		return RecurringHireTemplate{}, ErrInvalidRecurringHireTimeRange
	}

	return RecurringHireTemplate{
		ID:                id,
		RequestedByUserID: requestedByUserID,
		CourtID:           courtID,
		Weekday:           weekday,
		StartTime:         startTime,
		EndTime:           endTime,
		StartsAt:          startsAt,
		EndCondition:      endCondition,
		Status:            RecurringHireStatusRequested,
	}, nil
}

// Approve transitions a requested template to approved. Legal only from
// requested — re-approving an already-approved template, or approving a
// rejected/cancelled one, is rejected as
// ErrInvalidRecurringHireStatusTransition rather than silently accepted,
// the same discipline Booking.Cancel and Payment.MarkPaid/Refund apply to
// their own transitions.
func (t *RecurringHireTemplate) Approve() error {
	if t.Status != RecurringHireStatusRequested {
		return ErrInvalidRecurringHireStatusTransition
	}
	t.Status = RecurringHireStatusApproved
	return nil
}

// Reject transitions a requested template to rejected. Legal only from
// requested, mirroring Approve.
func (t *RecurringHireTemplate) Reject() error {
	if t.Status != RecurringHireStatusRequested {
		return ErrInvalidRecurringHireStatusTransition
	}
	t.Status = RecurringHireStatusRejected
	return nil
}

// GenerateOccurrences computes the concrete weekly TimeRange slots t
// implies, starting from the first date on or after t.StartsAt that falls
// on t.Weekday, then weekly thereafter. It is pure — no I/O, and it never
// constructs a Booking; turning an occurrence into a real, persisted
// SourceRecurringHire Booking is T11.5's job once a template is approved.
//
// Generation stops at the first of:
//   - t.EndCondition, if it is EndAfterDate or EndAfterOccurrences;
//   - upTo, always applied regardless of EndCondition. This is what keeps a
//     NoEnd (open-ended) template from ever producing an unbounded slice:
//     upTo is a required safety cap the caller supplies (e.g. "generate the
//     next 90 days"), not an optional bound layered on top of EndCondition.
func GenerateOccurrences(t RecurringHireTemplate, upTo time.Time) []TimeRange {
	var occurrences []TimeRange

	date := firstOccurrenceDate(t.StartsAt, t.Weekday)
	count := 0
	for {
		start := atClockTime(date, t.StartTime)
		if !start.Before(upTo) {
			break
		}
		if t.EndCondition.Kind == RecurringHireEndAfterDate && dateAfter(date, t.EndCondition.Date) {
			break
		}

		end := atClockTime(date, t.EndTime)
		occurrences = append(occurrences, TimeRange{Start: start, End: end})
		count++

		if t.EndCondition.Kind == RecurringHireEndAfterOccurrences && count >= t.EndCondition.Occurrences {
			break
		}

		date = date.AddDate(0, 0, 7)
	}

	return occurrences
}

// firstOccurrenceDate finds the first calendar date on or after startsAt
// (truncated to midnight in startsAt's own location) whose weekday is
// weekday.
func firstOccurrenceDate(startsAt time.Time, weekday time.Weekday) time.Time {
	y, m, d := startsAt.Date()
	date := time.Date(y, m, d, 0, 0, 0, 0, startsAt.Location())
	for date.Weekday() != weekday {
		date = date.AddDate(0, 0, 1)
	}
	return date
}

// atClockTime combines date's calendar day with ct's time-of-day, in date's
// location.
func atClockTime(date time.Time, ct ClockTime) time.Time {
	y, m, d := date.Date()
	return time.Date(y, m, d, int(ct)/60, int(ct)%60, 0, 0, date.Location())
}

// dateAfter reports whether date's calendar day is strictly after cutoff's
// calendar day (cutoff truncated to midnight in its own location) — this is
// what makes EndAfterDate inclusive of the cutoff day itself, rather than
// excluding an occurrence that starts on that day.
func dateAfter(date, cutoff time.Time) bool {
	cy, cm, cd := cutoff.Date()
	c := time.Date(cy, cm, cd, 0, 0, 0, 0, cutoff.Location())
	return date.After(c)
}
