// The Facility Owner's discount-creation form (T11.3): its input shape, its
// validation, and the translation from what the Owner typed into the real
// `CreateDiscountRule` request body.
//
// Kept as PURE functions, separate from the component and the composable,
// for two reasons: the WCAG 3.3.1/3.3.3 error messages are then pinned by
// table-driven unit tests (CLAUDE.md rule 1) rather than only observed
// through a mounted component, and the request-body translation — which has
// to respect several wire subtleties — is testable on its own.
//
// The wire subtleties, all read off the generated client
// (web/src/api/generated/booking.d.ts) rather than assumed:
//   - `fixedAmountCents` is int64 → a STRING over protojson.
//   - `percent` is a double → a real number.
//   - `startsAt`/`endDate` are `Format: date-time`, i.e. RFC3339 — an
//     <input type="date"> value ("2026-09-01") is NOT that.
//   - only the field matching `discount_type` is meaningful, and only the
//     `EndCondition` field matching `kind` is — so this module sends only
//     those, never a defaulted 0/"" alongside them.
import type { components } from '../api/generated/booking'
import type { DiscountType, Source } from './discount'

export type EndConditionKind = components['schemas']['v1EndConditionKind']
export type CreateDiscountRuleBody = components['schemas']['BookingServiceCreateDiscountRuleBody']

/**
 * Raw form state — parsing is validation's job, not the template's.
 *
 * The numeric fields are typed `string | number` rather than just `string`
 * because that is what the DOM actually produces: Vue's `v-model` on an
 * `<input type="number">` hands back a NUMBER when the entered text parses,
 * and the raw string only when it doesn't (e.g. mid-typing "1e"). Typing
 * these as `string` alone was a real bug — `validateDiscountForm` called
 * `.trim()` on them and threw a TypeError the moment a number arrived,
 * caught by FacilityDiscounts.spec.ts. `asText` below normalises both shapes
 * at the single point that reads them.
 */
export type NumericFormValue = string | number

export interface DiscountFormInput {
  discountType: DiscountType
  percent: NumericFormValue
  /** Major units, as typed (e.g. "5.00"), converted to minor units on submit. */
  amount: NumericFormValue
  currency: string
  appliesTo: Source[]
  /** `<input type="date">` value, i.e. YYYY-MM-DD. */
  startsAt: string
  endKind: EndConditionKind
  endDate: string
  occurrences: NumericFormValue
}

/** Normalises a `v-model` value that may arrive as either a number (numeric
 * input, parseable text) or a string (any other input, or unparseable text)
 * into the trimmed text form the validators reason about. */
function asText(value: NumericFormValue): string {
  return typeof value === 'number' ? String(value) : value.trim()
}

export function emptyDiscountForm(): DiscountFormInput {
  return {
    discountType: 'DISCOUNT_TYPE_PERCENT',
    percent: '',
    amount: '',
    currency: 'USD',
    appliesTo: [],
    startsAt: '',
    endKind: 'END_CONDITION_KIND_NO_END',
    endDate: '',
    occurrences: '',
  }
}

export interface DiscountFormErrors {
  percent?: string
  amount?: string
  currency?: string
  appliesTo?: string
  startsAt?: string
  endKind?: string
  endDate?: string
  occurrences?: string
}

/**
 * WCAG 3.3.1 (Error Identification) + 3.3.3 (Error Suggestion).
 *
 * Every message names the field's requirement AND how to satisfy it — an
 * example value, or the alternative control to use instead. None of them is
 * a bare "Invalid": identifying that something is wrong without saying what
 * would be right is precisely the 3.3.3 failure. The returned object is
 * keyed by field so the component can render each message next to its own
 * input (identification in TEXT, adjacent to the control — never colour
 * alone, and never a single lumped "check your input" banner).
 */
export function validateDiscountForm(input: DiscountFormInput): DiscountFormErrors {
  const errors: DiscountFormErrors = {}

  // Only the amount field matching the chosen type is meaningful, so only
  // that one is validated — the other is simply unused, not "empty".
  if (input.discountType === 'DISCOUNT_TYPE_PERCENT') {
    const percentText = asText(input.percent)
    const percent = Number(percentText)
    if (percentText === '' || Number.isNaN(percent) || percent <= 0 || percent > 100) {
      errors.percent = 'Enter a percentage greater than 0 and up to 100 — for example, 15 for 15% off.'
    }
  } else if (input.discountType === 'DISCOUNT_TYPE_FIXED_AMOUNT') {
    const amountText = asText(input.amount)
    const amount = Number(amountText)
    if (amountText === '' || Number.isNaN(amount) || amount <= 0) {
      errors.amount = 'Enter an amount greater than 0 — for example, 5.00 for five dollars off.'
    }
    if (!/^[A-Za-z]{3}$/.test(input.currency.trim())) {
      errors.currency = 'Enter a 3-letter currency code — for example, USD.'
    }
  }

  if (input.appliesTo.length === 0) {
    errors.appliesTo = 'Select at least one booking type this discount applies to, for example Individual bookings.'
  }

  if (!input.startsAt) {
    errors.startsAt = 'Choose the date this discount starts, in the format YYYY-MM-DD.'
  }

  if (input.endKind === 'END_CONDITION_KIND_END_AFTER_DATE') {
    if (!input.endDate) {
      errors.endDate = 'Choose the date this discount ends, or select "No end date" instead.'
    } else if (input.startsAt && new Date(input.endDate) <= new Date(input.startsAt)) {
      errors.endDate = `Choose an end date after the start date (${input.startsAt}).`
    }
  } else if (input.endKind === 'END_CONDITION_KIND_END_AFTER_OCCURRENCES') {
    const occurrencesText = asText(input.occurrences)
    const occurrences = Number(occurrencesText)
    if (occurrencesText === '' || !Number.isInteger(occurrences) || occurrences < 1) {
      errors.occurrences =
        'Enter how many total uses this discount allows — a whole number of 1 or more, for example 10.'
    }
  } else if (input.endKind !== 'END_CONDITION_KIND_NO_END') {
    errors.endKind = 'Choose when this discount ends — pick "No end date" if it should run indefinitely.'
  }

  return errors
}

/** Converts a major-unit value ("5.00" or 5) to integer minor units (500).
 * Rounded, not truncated, so "5.005" doesn't silently become 500. */
function toMinorUnits(amount: NumericFormValue): number {
  return Math.round(Number(asText(amount)) * 100)
}

/** `<input type="date">` gives YYYY-MM-DD; the API wants RFC3339. */
function toTimestamp(dateValue: string): string {
  return new Date(dateValue).toISOString()
}

/**
 * Builds the real `CreateDiscountRule` request body. Call only on input that
 * `validateDiscountForm` accepted.
 *
 * Sends ONLY the fields that are meaningful for the chosen discount type and
 * end-condition kind. A fixed-amount rule carries no `percent`, a percent
 * rule carries no `fixedAmountCents`/`currency`, and a NoEnd condition
 * carries neither a date nor a count — matching booking.proto's "only the
 * field matching `kind` is meaningful" contract instead of shipping
 * defaulted zeros the server would have to ignore.
 */
export function toCreateDiscountRuleBody(input: DiscountFormInput, actorUserId: string): CreateDiscountRuleBody {
  const endCondition: NonNullable<CreateDiscountRuleBody['endCondition']> = { kind: input.endKind }
  if (input.endKind === 'END_CONDITION_KIND_END_AFTER_DATE') {
    endCondition.endDate = toTimestamp(input.endDate)
  } else if (input.endKind === 'END_CONDITION_KIND_END_AFTER_OCCURRENCES') {
    endCondition.occurrences = Number(asText(input.occurrences))
  }

  const body: CreateDiscountRuleBody = {
    actorUserId,
    discountType: input.discountType,
    appliesTo: input.appliesTo,
    startsAt: toTimestamp(input.startsAt),
    endCondition,
  }

  if (input.discountType === 'DISCOUNT_TYPE_PERCENT') {
    body.percent = Number(asText(input.percent))
  } else if (input.discountType === 'DISCOUNT_TYPE_FIXED_AMOUNT') {
    body.fixedAmountCents = String(toMinorUnits(input.amount))
    body.currency = input.currency.trim().toUpperCase()
  }

  return body
}
