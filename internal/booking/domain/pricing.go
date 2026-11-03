package domain

import "time"

const oneDay = 24 * time.Hour

// Band names the kind of pricing window a PricingRule represents. Purely
// descriptive — matching is driven by Weekdays/Start/End, not by Band itself,
// but the name is part of the ubiquitous language (see the operating
// handbook glossary) and is surfaced to hosts/owners when a Quote is shown.
type Band string

const (
	BandWeekday Band = "weekday"
	BandPeak    Band = "peak"
	BandWeekend Band = "weekend"
)

// PricingRule resolves to a price for any slot on CourtID that falls fully
// within [Start, End) on one of Weekdays. Rules are per-court (facility-level
// inheritance, if any, is an app-layer concern of copying/looking up rules —
// the domain only resolves against whatever rule set it's given).
type PricingRule struct {
	ID         string
	CourtID    string
	Band       Band
	Weekdays   []time.Weekday
	Start      ClockTime
	End        ClockTime
	PriceCents int64
	// Currency is the ISO 4217 alphabetic code PriceCents is denominated in
	// (ADR-0005, T68.2, closes #357). Two fields rather than a Money: this
	// is booking's own convention, set by DiscountRule's
	// fixed_amount_cents/currency pair in booking.proto and by every
	// `*_cents` + `*_currency` column pair in db/migrations. See money.go
	// for why PricingRule does not use this context's Money type.
	//
	// It is not optional and it has no default. ResolvePrice refuses a rule
	// whose code is missing or malformed (ErrInvalidPricingCurrency) and
	// refuses a court whose rules disagree (ErrMixedPricingCurrency) —
	// ADR-0009's market scope was answered on 2026-10-07 as *both* Vietnam
	// and global, so there is no single value a missing code could honestly
	// mean.
	Currency string
}

func (r PricingRule) appliesToWeekday(w time.Weekday) bool {
	for _, d := range r.Weekdays {
		if d == w {
			return true
		}
	}
	return false
}

// covers reports whether the rule's window fully contains the slot. The
// slot must not cross midnight for clock-time comparison to be meaningful;
// callers resolving multi-day slots must split them per day first.
func (r PricingRule) covers(slot TimeRange) bool {
	if !r.appliesToWeekday(slot.Start.Weekday()) {
		return false
	}
	slotStart := ClockTimeOf(slot.Start)
	slotEnd := clockTimeOfEnd(slot)
	return r.Start <= slotStart && slotEnd <= r.End
}

// clockTimeOfEnd handles the common case of a slot ending exactly at
// midnight (e.g. 22:00-24:00), which time.Time represents as 00:00 the next
// calendar day — that must read as end-of-day (1440), not start-of-day (0),
// for boundary comparisons against a rule's End to work.
func clockTimeOfEnd(slot TimeRange) ClockTime {
	end := ClockTimeOf(slot.End)
	if end == 0 && slot.End.After(slot.Start) {
		return ClockTime(minutesPerDay)
	}
	return end
}

// fitsSingleCalendarDay reports whether slot lies within one calendar day —
// clock-time comparison (ClockTimeOf) is only meaningful within a single
// day, so this is the guard that makes PricingRule.covers's documented
// precondition actually hold rather than just assumed. A slot ending
// exactly at midnight the next day still counts (that's the same case
// clockTimeOfEnd exists to handle); anything else spanning into a second
// calendar day does not.
//
// **"One calendar day" means one day in slot's own Location**, which is why
// ResolvePrice converts to the facility's zone before calling this (T68.1,
// ADR-0004) rather than after. Applied to UTC instants it answers a different
// question than the one pricing asks: a 06:30-07:30 slot in a +07:00 zone is
// 23:30-00:30 UTC the day before — one local day, two UTC ones — and would be
// refused as multi-day although no clock-time comparison on it is ambiguous
// at all.
//
// The next local midnight is computed as day+1 at 00:00 (time.Date normalises
// an out-of-range day), NOT as local midnight plus 24 hours. On **either** DST
// transition those are different instants, in opposite directions —
// America/Los_Angeles, measured:
//
//	spring forward  midnight+24h = 2026-03-09 01:00 PDT | true next midnight = 2026-03-09 00:00 PDT
//	fall back       midnight+24h = 2026-11-01 23:00 PST | true next midnight = 2026-11-02 00:00 PST
//	ordinary day    midnight+24h = 2026-06-11 00:00 PDT | true next midnight = 2026-06-11 00:00 PDT
//
// So a 22:00-24:00 local slot on the night of a transition compared unequal
// against that boundary and was rejected as spanning two days — twice a year,
// not once (the review of PR #370 caught the first version of this comment
// claiming only the spring-forward half). A test in pricing_timezone_test.go
// pins the spring-forward case; the fall-back row above is why the fix is
// `day+1` rather than a one-hour fudge in either direction.
func fitsSingleCalendarDay(slot TimeRange) bool {
	// A 24-hour cap on top of the date comparison below. It is redundant for
	// every slot whose local calendar day is 24 hours long, and on a
	// fall-back day (a 25-hour local day) it is deliberately conservative:
	// a full-day slot there is refused rather than priced by a band whose
	// ClockTime window cannot express the repeated hour.
	if slot.Duration() > oneDay {
		return false
	}
	y1, m1, d1 := slot.Start.Date()
	y2, m2, d2 := slot.End.Date()
	if y1 == y2 && m1 == m2 && d1 == d2 {
		return true
	}
	startOfNextDay := time.Date(y1, m1, d1+1, 0, 0, 0, 0, slot.Start.Location())
	return slot.End.Equal(startOfNextDay)
}

// ResolvePrice finds the single PricingRule (scoped to courtID) whose window
// fully contains slot, resolved against the wall clock of loc — the IANA zone
// of the Facility that owns courtID. Returns ErrNoPricingRule if none match (a
// slot straddling two rule windows, e.g. spanning a weekday/peak boundary, is
// a no-match rather than a partial price) and ErrAmbiguousPricingRule if more
// than one rule matches, since that means the rule set itself is
// misconfigured (overlapping windows) — the caller must fix the data, the
// domain won't silently pick one.
//
// # Why loc is a parameter and not an assumption (T68.1, ADR-0004, #356)
//
// A PricingRule's Start/End are ClockTimes: minutes since midnight, local to
// the court's facility. Nothing about a time.Time pins it to that zone — a
// TimeRange built by adapter/grpcapi from a protobuf Timestamp is normalised
// to UTC, so before this parameter existed an 18:00 Ho Chi Minh City slot
// arrived as 11:00 and was quoted the weekday band instead of peak. The
// conversion happens here, once, rather than at each caller, which is
// ADR-0004's "convert before computing, never assume the caller already
// converted" in code: every clock-time, weekday and calendar-day question
// below is asked of the converted range.
//
// loc is a *time.Location rather than an IANA name on purpose. Resolving a
// name means reading the system's tz database, which is I/O this package must
// not do (CLAUDE.md rule 2) — the app layer loads it through
// port.LocationLoader and hands the result down.
//
// A nil loc is ErrFacilityTimezoneUnset, never UTC. The domain will not price
// a slot it cannot place on a local clock; see that sentinel's doc comment.
//
// # The currency the resolved price is in (T68.2, ADR-0005, #357)
//
// The returned rule carries its own Currency, and two further refusals exist
// to make that field trustworthy rather than decorative:
// ErrInvalidPricingCurrency when any of the court's rules names no usable ISO
// 4217 code, and ErrMixedPricingCurrency when they do not agree on one. Both
// are decided before matching, by ensureOneUsableCurrency — see its doc
// comment for why the check spans the court's whole rule set, why a court with
// no rules is unaffected, and why this is a read-time refusal rather than a
// schema constraint.
func ResolvePrice(rules []PricingRule, courtID string, slot TimeRange, loc *time.Location) (PricingRule, error) {
	if loc == nil {
		return PricingRule{}, ErrFacilityTimezoneUnset
	}
	local := slot.In(loc)

	if !fitsSingleCalendarDay(local) {
		return PricingRule{}, ErrPricingSlotSpansMultipleDays
	}

	forCourt := make([]PricingRule, 0, len(rules))
	for _, r := range rules {
		if r.CourtID == courtID {
			forCourt = append(forCourt, r)
		}
	}

	if err := ensureOneUsableCurrency(forCourt); err != nil {
		return PricingRule{}, err
	}

	var matches []PricingRule
	for _, r := range forCourt {
		if r.covers(local) {
			matches = append(matches, r)
		}
	}

	switch len(matches) {
	case 0:
		return PricingRule{}, ErrNoPricingRule
	case 1:
		return matches[0], nil
	default:
		return PricingRule{}, ErrAmbiguousPricingRule
	}
}

// ensureOneUsableCurrency is T68.2's answer to "what happens when one court's
// pricing rules disagree about currency?" (ADR-0005, closes #357). The answer
// is that the quote is refused, and this is where that is decided.
//
// # Why the whole court's rule set, and not just the rule that matched
//
// The decisive reason is what the CONTRACT CAN EXPRESS, not who notices.
// GetQuoteResponse carries ONE currency field covering both price_cents and
// band_price_cents, so the wire format has no way to say "this court is
// priced in two currencies" — and the client renders exactly that way:
// CourtBookingFlow.vue puts the band price, the discounted price and a
// client-computed saving side by side, all through one formatter
// (web/src/models/booking.ts's formatPriceCents). A matched-rule check would
// let one court answer 200000 VND for a Tuesday slot and 35 USD for a
// Saturday one, both well-formed, both rendered identically, with nothing
// in-band to tell a user or a client that the unit changed between two views
// of the same court. Agreement is therefore a property of the COURT, because
// that is the granularity at which a client can represent it at all.
//
// Operator visibility is the second reason and the weaker one: every
// individual quote against a mixed set looks fine, so checking only the match
// would make the misconfiguration visible on none of the requests anyone
// makes — the shape of #356 (a band that resolved cleanly in the wrong zone)
// and of 0031 (a CHECK never widened, that no unit test could fail on).
//
// The scope is per-court because pricing_rules is per-court. Two facilities
// in two markets pricing their courts in two currencies is the world ADR-0009
// described on 2026-10-07 ("both markets"), so a platform-wide single
// currency is not the invariant — a single currency *per court* is.
//
// # What it costs, stated plainly
//
// The blast radius is the real price of this choice: ONE bad row makes a
// whole court unquotable, including every slot whose own band is perfectly
// well-formed. That is deliberate — a quote that fails is a bug report, a
// quote in an unknown currency is a wrong price that looks right — but it is
// not free, and it is the reason this should be read as an INTERIM
// READ-TIME CHECK rather than the final design. When #352 builds a write path
// for pricing_rules, the right place for this invariant is at the write, so a
// bad row cannot be stored and no reader has to refuse a court over it; the
// read-time check then becomes a backstop rather than the only line.
//
// # Why this is a read-time refusal and not a schema constraint
//
// Asked and answered rather than assumed, because the ticket named it as a
// real question:
//
//   - "all rows for one court name the same currency" is a cross-row
//     predicate. A Postgres CHECK sees one row, so it cannot express it. The
//     mechanisms that could are a trigger or a denormalised court-level copy
//     of the code.
//   - A trigger is this repository's most-broken mechanism:
//     CREATE OR REPLACE FUNCTION silently dropped prior logic from
//     enforce_game_capacity() twice (0012 over 0007, 0023 over 0012, union
//     restored by 0030), and CLAUDE.md records that "a DB-level guard with no
//     DB-level test is a comment" after 0007's reservation went seventeen
//     migrations unnoticed.
//   - Decisively: pricing_rules has NO write path. It is migration-seeded
//     only; there is no CreatePricingRule use case, no Pricing CRUD, and no
//     sqlc INSERT. HANDOFF.md's T1 "known gap, deliberately deferred" already
//     made exactly this call for exactly this table, for the overlapping-
//     window invariant — read-time detection in the domain
//     (ErrAmbiguousPricingRule), write-time guard deferred to the ticket that
//     builds the write path — and 0017 restates it as ADR-0002's standing
//     decision for discount_rules. A write-time guard on a table nothing
//     writes to would be a guard nothing can fire.
//
// Where that leaves CLAUDE.md rule 4, stated as what is true rather than as
// the rule restated in a form this satisfies:
//
//   - The PER-ROW half is in both places, as rule 4 requires. 0035's NOT NULL
//     and pricing_rules_currency_shape CHECK refuse a missing or
//     non-ISO-4217-shaped code; isValidCurrencyCode below is the same
//     predicate in Go. Its test drives pool.Exec directly rather than
//     app.Service, which is what CLAUDE.md demands of a Postgres-side
//     invariant — "to test the Postgres half of rule 4, the test must bypass
//     the Go half" — so a weakened CHECK fails a test even though the Go
//     guard would still refuse the value.
//   - The CROSS-ROW half is a DOCUMENTED EXCEPTION to rule 4, not a
//     satisfaction of it. It exists only in the domain. It inherits the
//     precedent HANDOFF.md's T1 entry set for this same table and the same
//     kind of invariant (overlapping rule windows, lines 391-398): accepted
//     as domain-only "because there is no write path yet", with the DB half
//     owed to the ticket that builds one. #352 is that ticket.
//
// # Ordering
//
// A malformed code is reported before a disagreement: it is the more basic
// fault, it may be the whole cause of the disagreement, and "mixed" would
// name a symptom. Both are reported before matching, so a priced court with
// broken currency data does not answer ErrNoPricingRule and send an operator
// hunting for band rows that are already there. An empty rule set has nothing
// to check, so an unpriced court keeps answering ErrNoPricingRule exactly as
// it has since T1 — pinned by a row in pricing_currency_test.go's precedence
// table.
func ensureOneUsableCurrency(forCourt []PricingRule) error {
	for _, r := range forCourt {
		if !isValidCurrencyCode(r.Currency) {
			return ErrInvalidPricingCurrency
		}
	}
	for _, r := range forCourt {
		if r.Currency != forCourt[0].Currency {
			return ErrMixedPricingCurrency
		}
	}
	return nil
}

// isValidCurrencyCode is a cheap ISO 4217 *format* check — exactly three
// uppercase letters — not a lookup against the real currency registry. It is
// a deliberate copy of internal/payments/domain's function of the same name,
// for the same reason this package keeps its own Money type rather than
// importing payments' (see money.go): a bounded context owns its own copy of
// this shape (T9's ruling against a shared-kernel Money, restated by T11.1).
//
// It is the Go half of 0035's pricing_rules_currency_shape CHECK, and the two
// are deliberately the same predicate stated twice — CLAUDE.md rule 4. The
// regexp the CHECK uses, '^[A-Z]{3}$', is this loop.
//
// Validating against the real ISO 4217 list is out of scope and stays out:
// ADR-0005 accepted a format check explicitly, and a registry would be a
// table this project would then have to keep current.
func isValidCurrencyCode(code string) bool {
	if len(code) != 3 {
		return false
	}
	for _, r := range code {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return true
}
