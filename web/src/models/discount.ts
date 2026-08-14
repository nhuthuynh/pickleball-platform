// View models + honest-rendering helpers for `DiscountRule` (T11.3,
// docs/process/t11-sprint-plan.md), built from the raw Booking API response
// (web/src/api/generated/booking.d.ts's `components['schemas']`, produced
// from proto/pickleball/booking/v1/booking.proto by T11.2).
//
// TWO WIRE FACTS THIS FILE EXISTS TO HANDLE, both read off the real
// generated types rather than assumed:
//
// 1. `fixed_amount_cents` is an `int64`, which protojson (and therefore the
//    generated OpenAPI/TS types) represents as a *string*, not a number —
//    see `v1DiscountRule.fixedAmountCents?: string`. Same trap
//    `models/booking.ts` already documents for `priceCents`. This module is
//    the one place that parses it.
//
// 2. `v1EndCondition` is exactly `{ kind?, endDate?, occurrences? }`. There
//    is NO remaining-count, uses-so-far, or occurrences-consumed field on
//    it, on `v1DiscountRule`, or on `v1ListDiscountRulesForFacilityResponse`.
//    `formatEndCondition` therefore renders the honest TOTAL for
//    END_AFTER_OCCURRENCES ("ends after N total uses") — T11.3 instruction
//    #3's explicit requirement not to invent a live counter this ticket
//    builds no backend support for.
import type { components } from '../api/generated/booking'

export type RawDiscountRule = components['schemas']['v1DiscountRule']
export type RawEndCondition = components['schemas']['v1EndCondition']
export type DiscountType = components['schemas']['v1DiscountType']
export type Source = components['schemas']['v1Source']

/** A `DiscountRule` with its int64 money field parsed. `endCondition` stays
 * OPTIONAL on purpose — see `formatEndCondition` for why an absent condition
 * must not be defaulted to `NO_END`. */
export interface DiscountRule {
  id: string
  facilityId: string
  discountType: DiscountType
  percent: number
  fixedAmountCents: number
  currency: string
  appliesTo: Source[]
  startsAt: string
  endCondition?: RawEndCondition
}

export function mapToDiscountRule(raw: RawDiscountRule): DiscountRule {
  return {
    id: raw.id ?? '',
    facilityId: raw.facilityId ?? '',
    discountType: raw.discountType ?? 'DISCOUNT_TYPE_UNSPECIFIED',
    percent: raw.percent ?? 0,
    fixedAmountCents: Number(raw.fixedAmountCents ?? 0),
    currency: raw.currency ?? '',
    appliesTo: raw.appliesTo ?? [],
    startsAt: raw.startsAt ?? '',
    // Deliberately NOT `?? { kind: 'END_CONDITION_KIND_NO_END' }` — see
    // formatEndCondition.
    endCondition: raw.endCondition,
  }
}

/**
 * Renders how much a rule takes off, in words, without ever asserting a
 * number the rule doesn't actually carry.
 *
 * `percent` is meaningful ONLY for DISCOUNT_TYPE_PERCENT and
 * `fixedAmountCents`/`currency` ONLY for DISCOUNT_TYPE_FIXED_AMOUNT
 * (v1DiscountRule's own doc comments), so an unspecified type gets a bare
 * "Discount applied" rather than a confidently-rendered 0.
 *
 * A non-USD currency is never printed with a "$": `formatPriceCents` in
 * models/booking.ts hardcodes a dollar sign (correct for the quote path,
 * which is USD-cents by construction), but a DiscountRule carries its own
 * ISO 4217 code (ADR-0005) and may not be USD.
 */
export function formatDiscountAmount(rule: DiscountRule): string {
  if (rule.discountType === 'DISCOUNT_TYPE_PERCENT') {
    return `${rule.percent}% off`
  }
  if (rule.discountType === 'DISCOUNT_TYPE_FIXED_AMOUNT') {
    const amount = (rule.fixedAmountCents / 100).toFixed(2)
    return rule.currency && rule.currency !== 'USD' ? `${amount} ${rule.currency} off` : `$${amount} off`
  }
  return 'Discount applied'
}

const SOURCE_LABELS: Record<Source, string> = {
  SOURCE_UNSPECIFIED: 'Unspecified',
  SOURCE_RECURRING_HIRE: 'Recurring hire',
  SOURCE_INDIVIDUAL: 'Individual bookings',
  SOURCE_GAME: 'Games',
  SOURCE_COMPETITION: 'Competitions',
}

/** The four locked `Source` values in readable words. An empty list reads as
 * "Not specified" rather than "All bookings" — `applies_to` must be non-empty
 * server-side, so an empty one on the read path is missing information, not a
 * wildcard, and must not be rendered as one. */
export function formatAppliesTo(sources: Source[]): string {
  if (sources.length === 0) return 'Not specified'
  return sources.map((source) => SOURCE_LABELS[source] ?? source).join(', ')
}

/**
 * Renders an `EndCondition` honestly — T11.3 instruction #3.
 *
 * Four rules, each of which exists because the dishonest alternative is
 * tempting:
 *   - NO_END says "No end date". It never renders a date, including the
 *     meaningless `endDate` a NoEnd rule might still carry (only the field
 *     matching `kind` is meaningful, per v1EndCondition's doc comment).
 *   - END_AFTER_DATE renders the date it was actually given, and says the
 *     date is missing when it isn't there — it never substitutes "today",
 *     `startsAt`, or any other stand-in.
 *   - END_AFTER_OCCURRENCES renders the TOTAL, explicitly labelled "total
 *     uses". No read path in this API returns how many uses are left (there
 *     is no such field — see this file's header), so no remaining count is
 *     shown or implied.
 *   - An absent or UNSPECIFIED condition reads "End condition not set",
 *     NOT "No end date". Those are different statements: NO_END is a choice
 *     the Owner made; absence is the lack of one. booking.proto draws the
 *     same absent-vs-zero-valued distinction for `GetQuoteResponse.discount`.
 */
export function formatEndCondition(endCondition: RawEndCondition | undefined): string {
  const kind = endCondition?.kind

  if (kind === 'END_CONDITION_KIND_NO_END') {
    return 'No end date'
  }

  if (kind === 'END_CONDITION_KIND_END_AFTER_DATE') {
    if (!endCondition?.endDate) return 'End date not provided'
    return `Ends ${new Intl.DateTimeFormat(undefined, { dateStyle: 'medium' }).format(new Date(endCondition.endDate))}`
  }

  if (kind === 'END_CONDITION_KIND_END_AFTER_OCCURRENCES') {
    const occurrences = endCondition?.occurrences
    if (!occurrences) return 'Ends after a set number of total uses (count not provided)'
    return `Ends after ${occurrences} total ${occurrences === 1 ? 'use' : 'uses'}`
  }

  return 'End condition not set'
}
