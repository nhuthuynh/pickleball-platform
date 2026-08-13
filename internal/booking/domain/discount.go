package domain

import "time"

// DiscountType is a closed enum for how a DiscountRule computes its
// reduction: a percentage off the resolved price, or a flat amount off.
type DiscountType string

const (
	DiscountTypePercent     DiscountType = "percent"
	DiscountTypeFixedAmount DiscountType = "fixed_amount"
)

func (t DiscountType) IsValid() bool {
	switch t {
	case DiscountTypePercent, DiscountTypeFixedAmount:
		return true
	default:
		return false
	}
}

// DiscountAmount holds both possible representations of a DiscountRule's
// Amount field; which one is meaningful is selected by the sibling
// DiscountType field on DiscountRule. NewDiscountRule enforces that pairing
// and bounds whichever one applies: Percent in (0,100], Fixed a positive
// Money.
type DiscountAmount struct {
	Percent float64
	Fixed   Money
}

// EndConditionKind names which of EndCondition's three shapes is in play.
type EndConditionKind string

const (
	EndConditionKindDate        EndConditionKind = "end_after_date"
	EndConditionKindOccurrences EndConditionKind = "end_after_occurrences"
	EndConditionKindNoEnd       EndConditionKind = "no_end"
)

// EndCondition is a small variant (design doc v1-system-design.md §3.2): a
// DiscountRule ends on a fixed date, after a fixed number of occurrences, or
// never. Build one via EndAfterDate/EndAfterOccurrences/NoEnd rather than
// the struct literal, so the Kind/payload pairing stays consistent.
type EndCondition struct {
	Kind        EndConditionKind
	Date        time.Time
	Occurrences int
}

func EndAfterDate(d time.Time) EndCondition {
	return EndCondition{Kind: EndConditionKindDate, Date: d}
}

func EndAfterOccurrences(n int) EndCondition {
	return EndCondition{Kind: EndConditionKindOccurrences, Occurrences: n}
}

func NoEnd() EndCondition {
	return EndCondition{Kind: EndConditionKindNoEnd}
}

// DiscountRule is a facility-wide promotion layered on top of pricing
// (T1's PricingRule/ResolvePrice) — court-level discounts are out of scope
// this sprint (A9's scope-narrowing note). FacilityID is just a string
// reference here, not a cross-context lookup; resolving it against the real
// Facilities context is T11.2's job.
type DiscountRule struct {
	ID           string
	FacilityID   string
	DiscountType DiscountType
	Amount       DiscountAmount
	AppliesTo    []Source
	StartsAt     time.Time
	EndCondition EndCondition
}

// NewDiscountRule validates and constructs a DiscountRule. Each invariant
// below has its own distinct sentinel error so callers/tests can tell them
// apart with errors.Is, mirroring NewBooking's convention in booking.go.
func NewDiscountRule(id, facilityID string, discountType DiscountType, amount DiscountAmount, appliesTo []Source, startsAt time.Time, endCondition EndCondition) (DiscountRule, error) {
	if facilityID == "" {
		return DiscountRule{}, ErrEmptyFacilityID
	}
	if !discountType.IsValid() {
		return DiscountRule{}, ErrInvalidDiscountType
	}
	switch discountType {
	case DiscountTypePercent:
		if amount.Percent <= 0 || amount.Percent > 100 {
			return DiscountRule{}, ErrInvalidDiscountAmount
		}
	case DiscountTypeFixedAmount:
		if amount.Fixed.Cents <= 0 {
			return DiscountRule{}, ErrInvalidDiscountAmount
		}
	}
	if len(appliesTo) == 0 {
		return DiscountRule{}, ErrEmptyAppliesTo
	}
	for _, s := range appliesTo {
		if !s.IsValid() {
			return DiscountRule{}, ErrInvalidSource
		}
	}
	if endCondition.Kind == EndConditionKindOccurrences && endCondition.Occurrences <= 0 {
		return DiscountRule{}, ErrInvalidEndConditionOccurrences
	}

	return DiscountRule{
		ID:           id,
		FacilityID:   facilityID,
		DiscountType: discountType,
		Amount:       amount,
		AppliesTo:    appliesTo,
		StartsAt:     startsAt,
		EndCondition: endCondition,
	}, nil
}

func (r DiscountRule) appliesToSource(source Source) bool {
	for _, s := range r.AppliesTo {
		if s == source {
			return true
		}
	}
	return false
}

// isActiveAt reports whether the rule's time window covers at, using only
// what the domain can know from time alone. EndAfterOccurrences expiry
// depends on how many times the rule has already been used — that requires
// booking/usage history this pure function doesn't have (the same reason
// EnsureNoConflict takes `existing []Booking` as a parameter rather than
// holding it itself). Out of scope for T11.1; an EndAfterOccurrences rule is
// therefore not time-limited here, left for the app layer (or a later
// ticket) to pre-filter by usage count before calling ResolveDiscount.
func (r DiscountRule) isActiveAt(at time.Time) bool {
	if at.Before(r.StartsAt) {
		return false
	}
	if r.EndCondition.Kind == EndConditionKindDate && !at.Before(r.EndCondition.Date) {
		return false
	}
	return true
}

// ResolveDiscount finds the single DiscountRule that applies to a booking
// for facilityID/source at the given instant. Mirrors ResolvePrice's shape
// (pricing.go, T1):
//   - zero matches is NOT an error — a quote with no discount is a valid,
//     common outcome, so the zero-value DiscountRule is returned with a nil
//     error.
//   - exactly one match resolves normally.
//   - more than one match is domain.ErrAmbiguousDiscountRule, reusing
//     ADR-0002's precedent explicitly: an ambiguous match is a
//     data-configuration error, never silently resolved by priority or
//     insertion order.
func ResolveDiscount(rules []DiscountRule, facilityID string, source Source, at time.Time) (DiscountRule, error) {
	var matches []DiscountRule
	for _, r := range rules {
		if r.FacilityID != facilityID {
			continue
		}
		if !r.appliesToSource(source) {
			continue
		}
		if !r.isActiveAt(at) {
			continue
		}
		matches = append(matches, r)
	}

	switch len(matches) {
	case 0:
		return DiscountRule{}, nil
	case 1:
		return matches[0], nil
	default:
		return DiscountRule{}, ErrAmbiguousDiscountRule
	}
}
