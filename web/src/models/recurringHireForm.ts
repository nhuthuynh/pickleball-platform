// The Club's recurring-rental request form (T11.6): its input shape, its
// validation, and the translation into T11.5's real `RequestRecurringHire`
// request body.
//
// Kept as PURE functions, separate from the component and the composable, for
// the same two reasons `discountForm.ts` (T11.3) is: the WCAG 3.3.1/3.3.3
// error messages get pinned by table-driven unit tests (CLAUDE.md rule 1)
// rather than only observed through a mounted component, and the request-body
// translation — which has real wire subtleties — is testable on its own.
//
// The wire subtleties, read off the generated client
// (web/src/api/generated/booking.d.ts) rather than assumed:
//   - `weekday` is Go's `time.Weekday` numbering (0 = Sunday), an int32.
//   - `startMinute`/`endMinute` are MINUTES SINCE MIDNIGHT, not clock strings —
//     an `<input type="time">` gives "09:00", which is not that.
//   - `startsAt` is `Format: date-time` (RFC3339); an `<input type="date">`
//     gives "2026-09-07", which is not that either.
//   - only the `endCondition` field matching `kind` is meaningful, so only
//     that one is sent — never a defaulted 0/"" alongside it.
import type { components } from '../api/generated/booking'
import type { RecurringHireEndConditionKind } from './recurringHire'

export type RequestRecurringHireBody = components['schemas']['v1RequestRecurringHireRequest']

/** Same `string | number` reasoning as `discountForm.ts`'s own: `v-model` on
 * an `<input type="number">` hands back a NUMBER when the text parses and the
 * raw string when it doesn't. */
export type NumericFormValue = string | number

export interface RecurringHireFormInput {
  /** Chosen from the selected facility's court list — never typed by hand. */
  courtId: string
  /** `''` until chosen; otherwise 0..6 (Sunday..Saturday), as a string or
   * number depending on which control set it. */
  weekday: NumericFormValue
  /** `<input type="time">` value, i.e. HH:MM. */
  startTime: string
  endTime: string
  /** `<input type="date">` value, i.e. YYYY-MM-DD. */
  startsAt: string
  endKind: RecurringHireEndConditionKind
  endDate: string
  occurrences: NumericFormValue
}

function asText(value: NumericFormValue): string {
  return typeof value === 'number' ? String(value) : value.trim()
}

export function emptyRecurringHireForm(): RecurringHireFormInput {
  return {
    courtId: '',
    weekday: '',
    startTime: '',
    endTime: '',
    startsAt: '',
    endKind: 'RECURRING_HIRE_END_CONDITION_KIND_NO_END',
    endDate: '',
    occurrences: '',
  }
}

export interface RecurringHireFormErrors {
  courtId?: string
  weekday?: string
  startTime?: string
  endTime?: string
  startsAt?: string
  endDate?: string
  occurrences?: string
}

/** "HH:MM" to minutes since midnight, or `null` when it isn't a clock time. */
export function toMinuteOfDay(clock: string): number | null {
  const match = /^(\d{1,2}):(\d{2})$/.exec(clock.trim())
  if (!match) return null
  const hours = Number(match[1])
  const minutes = Number(match[2])
  if (hours > 23 || minutes > 59) return null
  return hours * 60 + minutes
}

/**
 * WCAG 3.3.1 (Error Identification) + 3.3.3 (Error Suggestion).
 *
 * Every message names the requirement AND how to satisfy it — an example
 * value, or the control to use instead. None is a bare "Invalid": identifying
 * that something is wrong without saying what would be right is exactly the
 * 3.3.3 failure. Keyed by field so the component renders each message next to
 * its own input, in text, never colour alone.
 *
 * These mirror the server's own guards rather than adding client-only rules:
 * `end > start` is `domain.ErrInvalidRecurringHireTimeRange`, a positive
 * occurrence count is `domain.ErrInvalidRecurringHireEndAfterOccurrences`, and
 * a well-formed court id is what keeps `NotFound` from being the first thing a
 * Club sees. The server remains authoritative — this is a fast, kind failure,
 * not a replacement for it.
 */
export function validateRecurringHireForm(input: RecurringHireFormInput): RecurringHireFormErrors {
  const errors: RecurringHireFormErrors = {}

  if (!input.courtId.trim()) {
    errors.courtId = 'Choose the court you want to book — pick a facility first, then one of its courts.'
  }

  const weekdayText = asText(input.weekday)
  const weekday = Number(weekdayText)
  if (weekdayText === '' || !Number.isInteger(weekday) || weekday < 0 || weekday > 6) {
    errors.weekday = 'Choose the day of the week this slot repeats on — for example, Mondays.'
  }

  const start = toMinuteOfDay(input.startTime)
  if (start === null) {
    errors.startTime = 'Enter the start time as a 24-hour clock time — for example, 09:00.'
  }

  const end = toMinuteOfDay(input.endTime)
  if (end === null) {
    errors.endTime = 'Enter the end time as a 24-hour clock time — for example, 10:30.'
  } else if (start !== null && end <= start) {
    errors.endTime = `Enter an end time later than the start time (${input.startTime}) — for example, 10:00.`
  }

  if (!input.startsAt) {
    errors.startsAt = 'Choose the first date this booking should run from, in the format YYYY-MM-DD.'
  }

  if (input.endKind === 'RECURRING_HIRE_END_CONDITION_KIND_END_AFTER_DATE') {
    if (!input.endDate) {
      errors.endDate = 'Choose the date this recurring booking should stop, or select “Keep going with no end date” instead.'
    } else if (input.startsAt && new Date(input.endDate) <= new Date(input.startsAt)) {
      errors.endDate = `Choose an end date after the start date (${input.startsAt}).`
    }
  } else if (input.endKind === 'RECURRING_HIRE_END_CONDITION_KIND_END_AFTER_OCCURRENCES') {
    const occurrencesText = asText(input.occurrences)
    const occurrences = Number(occurrencesText)
    if (occurrencesText === '' || !Number.isInteger(occurrences) || occurrences < 1) {
      errors.occurrences = 'Enter how many weeks this booking should run for — a whole number of 1 or more, for example 12.'
    }
  }

  return errors
}

/** `<input type="date">` gives YYYY-MM-DD; the API wants RFC3339. */
function toTimestamp(dateValue: string): string {
  return new Date(dateValue).toISOString()
}

/**
 * Builds the real `RequestRecurringHire` body. Call only on input that
 * `validateRecurringHireForm` accepted.
 *
 * Sends only the end-condition field matching the chosen kind, per
 * booking.proto's "only the field matching `kind` is meaningful" contract,
 * rather than shipping defaulted zeros the server would have to ignore.
 *
 * Note what is NOT here: no `isClub`/`role` field. Whether this actor may
 * request a recurring hire as a Club is resolved server-side against their
 * real Roles (T11.5, sprint plan A4 checklist item 2) — the client's own role
 * check decides what to SHOW, never what is permitted.
 */
export function toRequestRecurringHireBody(
  input: RecurringHireFormInput,
  actorUserId: string,
): RequestRecurringHireBody {
  const endCondition: NonNullable<RequestRecurringHireBody['endCondition']> = { kind: input.endKind }
  if (input.endKind === 'RECURRING_HIRE_END_CONDITION_KIND_END_AFTER_DATE') {
    endCondition.endDate = toTimestamp(input.endDate)
  } else if (input.endKind === 'RECURRING_HIRE_END_CONDITION_KIND_END_AFTER_OCCURRENCES') {
    endCondition.occurrences = Number(asText(input.occurrences))
  }

  return {
    actorUserId,
    courtId: input.courtId.trim(),
    weekday: Number(asText(input.weekday)),
    startMinute: toMinuteOfDay(input.startTime) ?? 0,
    endMinute: toMinuteOfDay(input.endTime) ?? 0,
    startsAt: toTimestamp(input.startsAt),
    endCondition,
  }
}
