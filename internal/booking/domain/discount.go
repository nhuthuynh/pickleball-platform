package domain

import (
	"math"
	"time"
)

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

// Applies reports whether r is a real, resolved rule rather than the
// zero-value DiscountRule ResolveDiscount returns for a no-match. The check
// is DiscountType validity, not an ID or a bool flag: a DiscountRule that
// came out of NewDiscountRule always has a valid DiscountType, and the zero
// value never can, so this cannot drift out of sync with the constructor the
// way a separate "found" flag would.
func (r DiscountRule) Applies() bool {
	return r.DiscountType.IsValid()
}

// ApplyToCents returns priceCents reduced by this rule. T11.2 added it: T11.1
// resolved *which* rule applies but never applied one, since nothing consumed
// a resolved rule until GetQuote did.
//
// Two rules are enforced here rather than at the call site, so every future
// consumer inherits them:
//   - A non-applying rule (the ResolveDiscount no-match zero value) returns
//     priceCents unchanged — "no discount" is not a 0%-off discount.
//   - The result is floored at 0. A 100%-off promotion, or a fixed_amount
//     rule larger than the band price, makes a slot free; it never produces a
//     negative price the payment path would have to interpret.
//
// Percent rounding is half away from zero on the *discount*, computed in
// float64 only for the percentage multiplication (DiscountAmount.Percent is a
// float64, e.g. 12.5) and immediately returned to integer cents — money is
// never carried as a float across this boundary (ADR-0005's integer-minor-
// units convention, the same reason Money.Cents is an int64).
func (r DiscountRule) ApplyToCents(priceCents int64) int64 {
	if !r.Applies() {
		return priceCents
	}

	var off int64
	switch r.DiscountType {
	case DiscountTypePercent:
		off = int64(math.Round(float64(priceCents) * r.Amount.Percent / 100))
	case DiscountTypeFixedAmount:
		off = r.Amount.Fixed.Cents
	}

	if off >= priceCents {
		return 0
	}
	return priceCents - off
}

// EnsureCurrency reports whether this rule may be taken off a price
// denominated in currency (T68.2, ADR-0005, closes #357).
//
// # The bug it closes
//
// ApplyToCents computes `priceCents - r.Amount.Fixed.Cents` and has never
// looked at either currency, because until this ticket the band price had no
// currency to look at: a 500-cent USD rule applied to a 200000-dong band
// price quietly subtracted 500 dong. The response then carried two
// currencies that disagreed — its own, and the nested DiscountRule's — which
// is #357's complaint in its sharpest form.
//
// # Why it is a separate method rather than a parameter on ApplyToCents
//
// ApplyToCents answers "how much", and that question has one answer. This
// answers "may it be applied at all", and a caller must be able to refuse
// before any arithmetic happens — app.Service.GetQuote returns the sentinel
// instead of a wrong number. Folding the check into ApplyToCents would force
// it to return an error, changing a total function used by three call sites
// into a fallible one; and the alternative, silently returning priceCents
// unchanged on a mismatch, would answer a quote at the undiscounted price
// while the response still advertised the discount.
//
// # What it deliberately does not check
//
//   - Anything that is not a fixed-amount rule, which is two exemptions in
//     one line (see "One check, not two" below). A percent rule passes
//     whatever its stored code: a percentage is dimensionless, and 0017
//     documents discount_rules.currency as "Empty for a percent rule, which
//     has no currency", so requiring a match would refuse every percent rule
//     in the table. Reading a stray non-empty code on one would be worse
//     still — a field the schema calls meaningless would change the
//     arithmetic.
//   - The shape of currency itself. That is the pricing side's invariant
//     (isValidCurrencyCode, pricing.go) and ResolvePrice has already refused
//     an unusable one before any caller can get here, so re-checking it would
//     report the same fault under a second name.
//
// An unlabelled fixed-amount rule (currency "") is a mismatch, not an
// exemption: 0017 made that column `NOT NULL DEFAULT ""` with no CHECK, so
// such rows can exist today, and reading "" as "the same currency as whatever
// this is applied to" is precisely the implicit-currency assumption this
// ticket removes.
//
// # One check, not two — and how that was found
//
// ResolveDiscount's no-match zero value is exempt, and that matters: a quote
// with no discount is the common outcome, so refusing it would break every
// undiscounted quote. This function was first written with its own
// `if !r.Applies() { return nil }` for that case, sitting above the type
// check.
//
// **The mutation matrix found that guard was dead**, and it is worth
// recording why rather than quietly deleting it. Cutting it changed no test's
// result, and — this is the part that decided the outcome — no test COULD
// have. Applies() is DiscountType.IsValid(), so !Applies() is true exactly
// when DiscountType is neither percent nor fixed_amount, and every one of
// those values already takes the `!= DiscountTypeFixedAmount` branch below.
// The two guards agree on the entire input domain.
//
// That makes this guard UNPINNABLE, which is a different finding from
// UNPINNED and leads to the opposite action. T67.5's cmd.WaitDelay was
// unpinned — no test noticed its removal — but pinnable: a test using setsid
// to escape the process group reaches it, so the answer there was to write
// that test. Nothing can reach this one. A guard no test can distinguish is a
// guard a later reader will take to be load-bearing when it is not, so it is
// gone and this paragraph is what remains of it.
//
// The BEHAVIOUR is still pinned, and not thinly. Isolating just this half —
// making EnsureCurrency return ErrDiscountCurrencyMismatch when !Applies()
// while leaving the percent exemption intact — fails **ten top-level tests
// across three packages** (booking/domain, booking/app,
// booking/adapter/grpcapi), because every quote that resolves no discount
// goes through here. Measured with that mutation rather than estimated; a
// coarser cut of the whole type check fails thirteen, but conflates this with
// the percent exemption. CLAUDE.md's standing lesson, which this is one more
// instance of: a mechanism whose necessity you can argue is not the same as a
// mechanism whose removal something notices.
func (r DiscountRule) EnsureCurrency(currency string) error {
	if r.DiscountType != DiscountTypeFixedAmount {
		return nil
	}
	if r.Amount.Fixed.Currency != currency {
		return ErrDiscountCurrencyMismatch
	}
	return nil
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
