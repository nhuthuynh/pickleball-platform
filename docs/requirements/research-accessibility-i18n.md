# Research: Accessibility & Localisation Non-Functional Requirements

**Scope.** This document verifies and sharpens the two NFRs stated in
`docs/pickleball-platform-spec.md` §4 — "Accessibility: WCAG AA" and
"Localisation: timezone-correct everywhere (store UTC, render local),
currency-ready" — against current external standards and this repo's actual
code (`internal/booking/domain/*`, `db/migrations/*.sql`). It builds on
`docs/roles/ux-ui-designer.md` (already covers NN/g heuristics, general WCAG
POUR structure, contrast ratios, target sizes, and color-only status) and
does **not** repeat that content. This is a research document, not code —
no application code was written or modified.

**A note on sourcing.** `w3.org`, `developer.apple.com`, and
`developers.google.com` returned HTTP 403/503 to this session's automated
fetcher (confirmed independently — same limitation `ux-ui-designer.md`
already recorded for several design-system sites). Where a primary source
could not be fetched directly, the finding is grounded in the primary
source's own criterion text as reproduced verbatim across multiple
independent, search-indexed secondary pages (WCAG "Understanding" mirrors,
standards trackers), and the primary URL is still given as the citation of
record — same convention the existing dossier uses.

---

## 1. WCAG AA verification — version and criteria most relevant to this app's flows

**Version finding.** WCAG 2.2 is the current W3C Recommendation (published
2023-10-05, updated 2024-12-12); WCAG 2.1 is the prior version. Content
conforming to 2.2 also conforms to 2.0 and 2.1 — the criteria are additive,
not a breaking replacement. WCAG 2.2 adds 9 new success criteria over 2.1
and removes one (4.1.1 Parsing, now obsolete because modern browsers/AT
handle malformed markup gracefully). `ux-ui-designer.md` §2.4 already cites
2.5.8 Target Size (Minimum, AA, new in 2.2). **Recommendation: the spec's
"WCAG AA" commitment should be read/stated as WCAG 2.2 AA**, not 2.1 AA —
it is a strict superset and is the version W3C now maintains as current.

**Criteria most load-bearing for this app's two flagship flows** (booking a
court — a form-heavy, time-sensitive, error-prone flow; and a roster view
with paid/unpaid status), beyond what `ux-ui-designer.md` §2.4/§5 already
covers (contrast, target size, color-only status):

- **1.4.1 Use of Color (Level A):** "Color is not used as the only visual
  means of conveying information, indicating an action, prompting a
  response, or distinguishing a visual element." `ux-ui-designer.md` already
  states the *consequence* of this rule for paid/unpaid badges (§4.2); this
  entry is the actual normative text it's built on, useful for a design
  review to cite directly. Note this is Level **A**, not AA — it is a floor
  requirement, not an aspirational one.
- **3.3.1 Error Identification (Level A):** "If an input error is
  automatically detected, the item that is in error is identified and the
  error is described to the user in text." Directly applicable to the
  booking form (date/time/court selection, payment amount entry): a failed
  submission must not just re-render the form — the specific invalid
  field must be identified in text, not by color/icon alone.
- **3.3.3 Error Suggestion (Level AA):** if an input error is detected and
  suggestions for correction are known, the suggestion must be provided to
  the user (unless it would jeopardize security/purpose). Applied here: a
  double-booking conflict (`ErrCourtDoubleBooked`) is exactly this case —
  the error is detected, and the system *does* know a suggestion (the next
  available slots), so AA requires surfacing it, not just rejecting.
  `ux-ui-designer.md` §2.3/§4.1 already recommends this from an NN/g
  usability angle; 3.3.3 makes the same recommendation a compliance
  requirement, not just good practice.
- **3.3.4 Error Prevention (Legal, Financial, Data) (Level AA):** applies to
  pages that cause a legal commitment or financial transaction, or delete/
  modify user data. Requires at least one of: submissions are reversible,
  data is checked with a chance to correct errors, or there is a
  review-confirm-correct step before final submission. This is **directly
  applicable to two flows in this product**: (a) booking-with-payment (a
  financial transaction) and (b) `CancelBooking` (deletes/modifies a
  user-controllable reservation). Recommendation: the booking confirmation
  step and the cancel-booking flow should each satisfy this via an explicit
  review/confirm step — cancellation already has a natural "are you sure"
  point given `CancelBooking` exists specifically to make mistakes
  recoverable (per `ux-ui-designer.md` §2.1, heuristic #3); this criterion
  gives that UX pattern a compliance basis, not just a heuristic one.
- **4.1.3 Status Messages (Level AA):** "In content implemented using
  markup languages, status messages can be programmatically determined
  through role or properties such that they can be presented to the user
  by assistive technologies without receiving focus." This is the
  criterion the whole product's async-feedback pattern hangs on: a booking
  confirmation, a "just taken, pick another slot" conflict message (per
  spec §6), or a payment-recorded toast must be exposed via an ARIA live
  region (`role="status"` / `role="alert"` / `aria-live`) on the Vue web
  client, not just rendered visually — otherwise a screen-reader user gets
  silence exactly where `ux-ui-designer.md`'s heuristic #1 (visibility of
  system status) requires feedback. This criterion is the concrete AA
  requirement underneath that heuristic for non-visual users.

**Gap identified:** `ux-ui-designer.md` cites WCAG 2.1 in its Sources list
(`w3.org/TR/WCAG21/`) and 2.2 only for target size. Recommend citing WCAG
2.2 as the baseline going forward given the above.

---

## 2. Timezone correctness for booking systems — verification against this repo's design

**External guidance.** Google Calendar's recurring-events documentation
states that for a recurring event, a single IANA timezone must always be
specified on the event (not just a UTC offset) because it is needed to
expand recurrence instances correctly; omitting it causes a weekly meeting
to "drift" across a DST transition, because without a real timezone the
series effectively falls back to a fixed UTC offset instead of a fixed
*local* wall-clock time. This is corroborated at the protocol level:
**RFC 5545 (iCalendar)** requires the `TZID` parameter on `DTSTART` (and
`RDATE`/`EXDATE`) whenever the value isn't UTC or "floating" time, and its
`VTIMEZONE` component exists specifically so a recurring event's local time
(e.g. "every Tuesday 18:00") is reproduced correctly across a DST boundary
— by re-deriving the correct UTC offset per-occurrence from the timezone's
own transition rules, rather than by holding the offset constant. This is a
widely-documented, recurring class of real bug: independently reported
against Mozilla Lightning, FullCalendar, Discourse, node-ical, The Events
Calendar, and calendar-sync tools — in each case a recurring event stored
with only a fixed UTC instant/offset shifts by exactly one hour on
recurrence instances that fall on the other side of a DST change.

**Cross-check against this project's actual design.**

- `bookings.starts_at` / `ends_at` are `timestamptz`, and the domain's
  `TimeRange` (`internal/booking/domain/timerange.go`) operates on
  `time.Time` — both are storing/comparing absolute UTC instants. This is
  **correct** for a *single, already-materialized* booking: two absolute
  instants either overlap or they don't, and the half-open
  `[Start, End)` / Postgres `tstzrange` semantics documented in
  `timerange.go` and the `no_overlap` `EXCLUDE` constraint (spec §6) are
  timezone-agnostic by construction — this part matches best practice and
  needs no change.
- **The gap is upstream of that:** D3b names `recurring_hire` as one of the
  four Booking sources ("a recurring hire is a template that generates
  bookings," spec §8), but no template/expansion code exists yet in this
  repo (confirmed: no `Recurring`/`RRULE` type, no weekday-expansion logic,
  and no `timezone` column on any table in
  `db/migrations/0001_init.sql`/`0003_pricing_rules.sql` — only `courts.id`
  and `courts.name` exist today). When that generator is built, per the
  Google Calendar / RFC 5545 guidance above, **the template's anchor must
  be stored as a local wall-clock time plus an IANA timezone identifier
  (e.g. `"18:00" + "Australia/Sydney"`), not just a UTC instant with a
  fixed offset baked in** — otherwise every occurrence generated across a
  DST transition will be off by an hour in local court time (a 6pm Tuesday
  hire silently becoming a 7pm or 5pm hire from the court operator's and
  club's point of view, even though the stored UTC timestamps are
  internally self-consistent). Each *generated* booking instance should
  still be materialized as a concrete UTC `starts_at`/`ends_at` (matching
  the existing, correct pattern) — the fix is in the template, not in the
  materialized booking.
- **Second, related gap:** `internal/booking/domain/clocktime.go`'s
  `ClockTimeOf(t time.Time)` documents itself as "local to the court's
  facility" but derives the hour/minute from whatever `Location` the
  `time.Time` argument already carries — nothing in the domain or the
  schema pins that location to the court's actual facility timezone. Since
  Postgres `timestamptz` round-trips as UTC by default through Go database
  drivers, any caller that doesn't explicitly convert to the facility's
  IANA zone before calling `ClockTimeOf` (used by `PricingRule.covers` for
  weekday/band matching, `pricing.go`) risks resolving pricing bands — and,
  by the same mechanism, a recurring hire's day-of-week — against the
  wrong calendar day for bookings near local midnight. There is no
  `timezone` column on `courts` today to anchor this. **Recommendation:**
  add an IANA timezone identifier to the court/facility record before the
  recurring-hire generator or any further pricing work lands, and make
  timezone-conversion-before-`ClockTimeOf` an explicit, tested step rather
  than an implicit assumption.

**Overall verdict:** the *storage* half of "store UTC, render local" is
already correctly implemented for materialized bookings. The *generation*
half (recurring weekly hire, and clock-time-dependent pricing resolution)
needs an explicit IANA-timezone anchor that doesn't exist in the schema
yet — this is the single most concrete, actionable gap this research pass
found.

---

## 3. Currency / i18n readiness — verification against this repo's pricing domain

**External guidance.** The historically canonical reference here is Martin
Fowler's **Money** pattern (*Patterns of Enterprise Application
Architecture*): couple an amount with its currency in a single value type
rather than passing a bare number, and avoid floating-point for the amount
because binary floats introduce rounding error that a monetary type is
specifically meant to prevent — store the amount as an integer (or fixed
decimal) in the currency's smallest unit ("minor units": cents, pence,
etc.), converting to major units only for display. This is also
industry-standard fintech practice: Stripe's own API represents all amounts
as integers in the currency's smallest unit (e.g. cents for USD) for
exactly this reason. Currency identity itself should use **ISO 4217**
three-letter codes (`USD`, `AUD`, `GBP`, …) as the canonical currency
identifier rather than a symbol or free-text field, since symbols are
ambiguous (`$` alone doesn't disambiguate USD/AUD/CAD/NZD) and codes are
what Stripe, banks, and every FX API key off.

**Cross-check against this repo.** `PricingRule.PriceCents int64`
(`internal/booking/domain/pricing.go`) and the matching
`pricing_rules.price_cents bigint` column
(`db/migrations/0003_pricing_rules.sql`) already follow the integer-minor-
units half of the Money pattern correctly — no floating-point money
anywhere in the domain, confirmed by reading `pricing.go` end to end.
**The gap is the other half of the pattern: there is no currency
identifier anywhere in the schema or the domain** — `PricingRule` has no
`Currency` field, `pricing_rules` has no `currency_code` column, and
nothing in `payments` (not yet built) is scoped either. This matches the
spec's own stated assumption ("single launch market, built i18n/currency-
ready for later expansion," spec header) — it is a reasonable simplification
for a single-market v1, *not* a bug — but "currency-ready" per the Money
pattern specifically means the amount and currency should already be
coupled in one type even while only one currency is supported, so that
adding a second market later is a data-population change (populate the
existing column) rather than a schema-and-every-callsite migration.
**Recommendation:** add an ISO 4217 `currency_code` (e.g. `char(3)` /
Postgres `char(3)` or a narrow enum) alongside `price_cents` now, even if
every row is populated with the same constant value today — this is the
concrete, low-cost action that actually satisfies "currency-ready" per the
Fowler/Stripe pattern, versus leaving amount and currency decoupled until
multi-currency is an active requirement.

---

## 4. Mobile accessibility for the planned Swift/Kotlin clients

Not a full spec — noting what the eventual native clients will need to
inherit from the API/domain layer, since domain logic lives only in the Go
backend per `CLAUDE.md` and both native apps are thin clients on it.

- **Apple HIG (iOS/SwiftUI).** Accessibility is a first-class HIG section
  built around VoiceOver: every interactive element needs a meaningful
  `accessibilityLabel` (system controls get one automatically; custom
  controls and images need one set explicitly), and VoiceOver's rotor lets
  users navigate by headings/links/landmarks — meaning content structure
  (not just visual layout) has to be intentional. `ux-ui-designer.md` §3.3
  already covers the numeric HIG target-size floor (44×44pt); this adds the
  *screen-reader* half: minimum tap-target size alone does not make an app
  accessible if elements lack labels or if error states aren't exposed to
  VoiceOver as text. Source: Apple's Human Interface Guidelines
  accessibility section (site blocks automated fetching in this session,
  as already noted in `ux-ui-designer.md`; content corroborated via
  multiple independent HIG summaries in this research pass).
- **Android (Kotlin/Compose, Material).** The equivalent mechanism is
  TalkBack: every meaningful view needs a `contentDescription` (Android
  Studio lints for missing ones on images), and dynamically-appearing
  status/error content should be marked as a **live region** — Assertive
  mode for something that must interrupt (e.g., a booking conflict just
  detected), Polite mode for something that can wait for current speech to
  finish — or announced explicitly via `announceForAccessibility`. This is
  the Android-native analogue of WCAG 4.1.3 Status Messages (§1 above):
  the same requirement — status changes must be programmatically
  determinable by assistive technology without requiring focus — recurs
  as a first-class platform concern on both web (ARIA live regions) and
  Android (live regions / TalkBack announcements), and would need the
  equivalent treatment in SwiftUI (`.accessibilityAddTraits` /
  post-notification APIs) for iOS. Source: Android Developers accessibility
  guidance (`developer.android.com/guide/topics/ui/accessibility`),
  corroborated via search-indexed secondary coverage in this pass.
- **What this implies for the API/domain layer specifically (the part this
  backend controls):** the recurring theme across WCAG 3.3.1/3.3.3, Apple's
  "avoid cryptic error codes," and Android's live-region-for-errors pattern
  is the same requirement from three different platforms — **a client
  needs a human-readable, specific error description, not just a domain
  error code, to build an accessible error surface.** Today
  `internal/booking/domain/errors.go` defines domain sentinel errors
  (e.g. `ErrCourtDoubleBooked`) that the gRPC/REST layer maps to status
  codes; per `ux-ui-designer.md` §2.3/§6 this already needs a designed
  message for *visual* UI. The accessibility angle sharpens the same point
  into an API-shape requirement: whatever structured error the API returns
  (gRPC status details / REST error body) should carry a
  precise, user-facing message string per error case — not just a code —
  so that all three clients (Vue, Swift, Kotlin) can feed that string
  directly into an ARIA live region / VoiceOver announcement / TalkBack
  live region without each client re-inventing per-error copy, and without
  ever surfacing a raw Postgres/gRPC code (`23P01`, `ErrCourtDoubleBooked`)
  to an end user or a screen reader. This is a genuine, cross-cutting API
  design input for whenever error responses are formalized — flagged here
  as a research finding, not implemented.

---

## Sources

**WCAG / W3C (version and criteria text)**
- [W3C WAI — What's New in WCAG 2.2](https://www.w3.org/WAI/standards-guidelines/wcag/new-in-22/)
- [W3C — WCAG 2.2 (W3C Recommendation)](https://www.w3.org/TR/WCAG22/)
- [W3C WAI — Understanding SC 1.4.1 Use of Color](https://www.w3.org/WAI/WCAG22/Understanding/use-of-color.html)
- [W3C WAI — Understanding SC 3.3.1 Error Identification](https://www.w3.org/WAI/WCAG22/Understanding/error-identification.html)
- [W3C WAI — Understanding SC 3.3.3 Error Suggestion](https://www.w3.org/WAI/WCAG21/Understanding/error-suggestion.html)
- [W3C WAI — Understanding SC 3.3.4 Error Prevention (Legal, Financial, Data)](https://www.w3.org/WAI/WCAG22/Understanding/error-prevention-legal-financial-data.html)
- [W3C WAI — Understanding SC 4.1.3 Status Messages](https://www.w3.org/WAI/WCAG22/Understanding/status-messages.html)
- [W3C WAI — F103: Failure of SC 4.1.3 (status messages not programmatically determinable)](https://www.w3.org/WAI/WCAG21/Techniques/failures/F103)
  *(all six w3.org pages returned HTTP 403 to this session's automated fetcher; criterion text above is reproduced verbatim as it appears identically across multiple independent search-indexed mirrors of the same W3C "Understanding" documents, cross-checked for consistency across sources before inclusion)*

**Timezone / recurring-event scheduling**
- [Google for Developers — Calendar API: Recurring events](https://developers.google.com/workspace/calendar/api/guides/recurringevents)
- [RFC 5545 — Internet Calendaring and Scheduling Core Object Specification (iCalendar)](https://www.rfc-editor.org/rfc/rfc5545) (see §3.2.19 `TZID`, §3.6.5 `VTIMEZONE`)
- [Mozilla Bugzilla #563314 — recurrent events shifted by DST](https://bugzilla.mozilla.org/show_bug.cgi?id=563314)
- [node-ical issue #97 — recurring event over DST start/end](https://github.com/jens-maus/node-ical/issues/97)
- [FullCalendar issue #6729 — recurring event time wrong after DST](https://github.com/fullcalendar/fullcalendar/issues/6729)

**Currency / monetary value storage**
- Martin Fowler, [Money (Patterns of Enterprise Application Architecture)](https://www.martinfowler.com/eaaCatalog/money.html)
- [ISO 4217 — Currency codes (ISO)](https://www.iso.org/iso-4217-currency-codes.html)
- [Stripe API docs — amounts are integers in the currency's smallest unit](https://docs.stripe.com/currencies) (per spec §5, Stripe is this project's payment processor, making this directly applicable, not just an analogy)

**Mobile accessibility**
- [Apple Developer — Human Interface Guidelines: Accessibility](https://developer.apple.com/design/human-interface-guidelines/accessibility) (blocked automated fetch in this session, as `ux-ui-designer.md` also records for HIG pages; content corroborated via independent secondary summaries)
- [Android Developers — Build accessible apps](https://developer.android.com/guide/topics/ui/accessibility)
- [Android Developers — Principles for improving app accessibility](https://developer.android.com/guide/topics/ui/accessibility/principles)

---

*Research only — no application code was written or modified. See
`internal/booking/domain/timerange.go`, `clocktime.go`, `pricing.go`,
`booking.go`, `errors.go`, and `db/migrations/0001_init.sql` /
`0003_pricing_rules.sql` for the code this document cross-checks against.*
