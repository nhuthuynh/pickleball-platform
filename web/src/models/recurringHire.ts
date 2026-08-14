// View models + formatters for the Club rental request/approval flow (T11.6,
// docs/process/t11-sprint-plan.md), built from T11.5's real
// `RecurringHireTemplate`/`RecurringHireOccurrence` wire types
// (web/src/api/generated/booking.d.ts, produced from
// proto/pickleball/booking/v1/booking.proto).
//
// HONESTY RULES this module enforces by construction (T11.6 instruction #4,
// and instruction #2's "never a bare approved with no visibility into partial
// conflicts"), because a formatter is where fabrication actually happens:
//
//  1. **A missing field is never defaulted into a real-looking value.** Every
//     proto scalar is optional over protojson, and `weekday` in particular has
//     a zero value that MEANS something (0 = Sunday). `?? 0` there would turn
//     "the server didn't send a weekday" into "Sundays" — a fabricated
//     schedule. Absent fields map to `null` and format as "Not specified".
//
//  2. **A rejected template is terminal.** `isDecided`/`statusExplanation`
//     exist so no screen can render a decided request as though a further
//     decision were still coming. The copy says a new request is required, in
//     text, because that is what the backend actually requires: T11.4 models
//     approve/reject as one-way transitions, and
//     `domain.ErrInvalidRecurringHireStatusTransition` is what a second
//     decision on the same template would return.
//
//  3. **An approved template does NOT imply every week was booked.** T11.5's
//     approval books each occurrence independently and reports BOOKED /
//     SKIPPED_CONFLICT / SKIPPED_ERROR per week; the template still becomes
//     `approved` regardless. Those per-occurrence results exist ONLY in the
//     ApproveRecurringHire response — no read endpoint returns them later — so
//     `approvedWeeksAreUnknownNote` states that limitation instead of letting
//     a screen imply a full set of confirmed weeks it cannot know about.
import type { components } from '../api/generated/booking'

export type RawRecurringHireTemplate = components['schemas']['v1RecurringHireTemplate']
export type RawRecurringHireOccurrence = components['schemas']['v1RecurringHireOccurrence']
export type RecurringHireStatus = components['schemas']['v1RecurringHireStatus']
export type RecurringHireEndConditionKind = components['schemas']['v1RecurringHireEndConditionKind']
export type RawRecurringHireEndCondition = components['schemas']['v1RecurringHireEndCondition']
export type OccurrenceOutcome = components['schemas']['v1RecurringHireOccurrenceOutcome']

/** One row in either the Club's status view or the Owner's incoming queue. */
export interface RecurringHireTemplateView {
  id: string
  requestedByUserId: string
  courtId: string
  /** Go's `time.Weekday` numbering (0 = Sunday). `null` when absent — never
   * defaulted to 0, which would read as a real "Sundays" schedule. */
  weekday: number | null
  /** Minutes since midnight. `null` when absent; 0 (midnight) is a real,
   * distinct value and is preserved. */
  startMinute: number | null
  endMinute: number | null
  /** RFC3339 as received, or `''` when absent. */
  startsAt: string
  endCondition: RawRecurringHireEndCondition | null
  status: RecurringHireStatus
}

/** One generated week, as ApproveRecurringHire reported it. */
export interface OccurrenceView {
  startsAt: string
  endsAt: string
  outcome: OccurrenceOutcome
  bookingId: string
  reason: string
}

export function mapToRecurringHireTemplate(raw: RawRecurringHireTemplate): RecurringHireTemplateView {
  return {
    id: raw.id ?? '',
    requestedByUserId: raw.requestedByUserId ?? '',
    courtId: raw.courtId ?? '',
    weekday: typeof raw.weekday === 'number' ? raw.weekday : null,
    startMinute: typeof raw.startMinute === 'number' ? raw.startMinute : null,
    endMinute: typeof raw.endMinute === 'number' ? raw.endMinute : null,
    startsAt: raw.startsAt ?? '',
    endCondition: raw.endCondition ?? null,
    status: raw.status ?? 'RECURRING_HIRE_STATUS_UNSPECIFIED',
  }
}

export function mapToOccurrence(raw: RawRecurringHireOccurrence): OccurrenceView {
  return {
    startsAt: raw.startsAt ?? '',
    endsAt: raw.endsAt ?? '',
    outcome: raw.outcome ?? 'RECURRING_HIRE_OCCURRENCE_OUTCOME_UNSPECIFIED',
    bookingId: raw.bookingId ?? '',
    reason: raw.reason ?? '',
  }
}

const WEEKDAY_NAMES = [
  'Sundays',
  'Mondays',
  'Tuesdays',
  'Wednesdays',
  'Thursdays',
  'Fridays',
  'Saturdays',
] as const

/** Weekday numbers as the form offers them, in the order they are shown. */
export const WEEKDAY_OPTIONS: { value: number; label: string }[] = WEEKDAY_NAMES.map(
  (label, value) => ({ value, label }),
)

export function formatWeekday(weekday: number | null): string {
  if (weekday === null || !Number.isInteger(weekday) || weekday < 0 || weekday > 6) {
    return 'Not specified'
  }
  // `?? 'Not specified'` is unreachable given the range check above, but the
  // compiler's `noUncheckedIndexedAccess` is right that an index access is not
  // a proof — and the fallback it forces is the honest value anyway, not a
  // fabricated weekday.
  return WEEKDAY_NAMES[weekday] ?? 'Not specified'
}

/** Minutes-since-midnight to a 24-hour clock time. Returns `''` for an absent
 * or out-of-range value rather than inventing "00:00". */
export function formatMinuteOfDay(minute: number | null): string {
  if (minute === null || !Number.isInteger(minute) || minute < 0 || minute > 24 * 60) return ''
  const hours = Math.floor(minute / 60)
  const minutes = minute % 60
  return `${String(hours).padStart(2, '0')}:${String(minutes).padStart(2, '0')}`
}

export function formatTimeRange(startMinute: number | null, endMinute: number | null): string {
  const start = formatMinuteOfDay(startMinute)
  const end = formatMinuteOfDay(endMinute)
  if (!start || !end) return 'Time not specified'
  return `${start}–${end}`
}

/** A date-only rendering of an RFC3339 timestamp, or `''` when absent or
 * unparseable — never today's date as a stand-in. */
export function formatDate(timestamp: string): string {
  if (!timestamp) return ''
  const parsed = new Date(timestamp)
  if (Number.isNaN(parsed.getTime())) return ''
  return parsed.toISOString().slice(0, 10)
}

export function formatDateTime(timestamp: string): string {
  if (!timestamp) return ''
  const parsed = new Date(timestamp)
  if (Number.isNaN(parsed.getTime())) return ''
  return `${parsed.toISOString().slice(0, 10)} ${parsed.toISOString().slice(11, 16)}`
}

/**
 * The end condition in words. NO_END says the request itself is open-ended
 * AND that approval still only books a bounded horizon — the server applies
 * one (`app.recurringHireHorizonYears`), so calling it "forever" would be a
 * promise the backend does not make.
 */
export function formatRecurringEndCondition(condition: RawRecurringHireEndCondition | null): string {
  if (!condition || !condition.kind) return 'Not specified'
  switch (condition.kind) {
    case 'RECURRING_HIRE_END_CONDITION_KIND_END_AFTER_DATE': {
      const date = formatDate(condition.endDate ?? '')
      return date ? `Until ${date}` : 'Until a set date (date not specified)'
    }
    case 'RECURRING_HIRE_END_CONDITION_KIND_END_AFTER_OCCURRENCES': {
      const count = condition.occurrences
      if (typeof count !== 'number' || count <= 0) return 'For a set number of weeks (count not specified)'
      return `For ${count} ${count === 1 ? 'week' : 'weeks'}`
    }
    case 'RECURRING_HIRE_END_CONDITION_KIND_NO_END':
      return 'Ongoing, with no end date'
    default:
      return 'Not specified'
  }
}

export function formatStatus(status: RecurringHireStatus): string {
  switch (status) {
    case 'RECURRING_HIRE_STATUS_REQUESTED':
      return 'Awaiting a decision'
    case 'RECURRING_HIRE_STATUS_APPROVED':
      return 'Approved'
    case 'RECURRING_HIRE_STATUS_REJECTED':
      return 'Rejected'
    case 'RECURRING_HIRE_STATUS_CANCELLED':
      return 'Cancelled'
    default:
      return 'Status not specified'
  }
}

/** True once the Facility Owner has answered — approved or rejected — or the
 * request was cancelled. A decided template accepts no further decision
 * (T11.4's one-way transitions), which is what every "can this still be
 * approved?" question on either screen resolves through. */
export function isDecided(status: RecurringHireStatus): boolean {
  return (
    status === 'RECURRING_HIRE_STATUS_APPROVED' ||
    status === 'RECURRING_HIRE_STATUS_REJECTED' ||
    status === 'RECURRING_HIRE_STATUS_CANCELLED'
  )
}

/**
 * The honest sentence that goes next to a status. The REJECTED case is
 * T11.6 instruction #4 in one string: a rejection is final, and getting this
 * slot requires a NEW request — not waiting, not a follow-up on this one.
 */
export function statusExplanation(status: RecurringHireStatus): string {
  switch (status) {
    case 'RECURRING_HIRE_STATUS_REQUESTED':
      return 'Sent to the facility owner. No courts are booked until they approve it.'
    case 'RECURRING_HIRE_STATUS_APPROVED':
      return 'The facility owner approved this request and booked the weeks that were free.'
    case 'RECURRING_HIRE_STATUS_REJECTED':
      return 'The facility owner rejected this request. That decision is final — this request cannot be approved later. Send a new request if you still want the slot.'
    case 'RECURRING_HIRE_STATUS_CANCELLED':
      return 'This request was cancelled and is closed. Send a new request if you still want the slot.'
    default:
      return 'This request has no status the app recognises. Contact the facility owner before assuming any court is booked.'
  }
}

/**
 * What an APPROVED template can and cannot tell a Club after the fact.
 *
 * The per-week outcomes only exist in the ApproveRecurringHire response, at
 * the moment of approval — no read endpoint replays them. Saying "all weeks
 * booked" here would be a fabrication, and saying nothing would let "Approved"
 * imply it.
 */
export const approvedWeeksAreUnknownNote =
  'Weeks where the court was already booked were skipped. This screen cannot list which weeks those were — only the owner’s approval screen showed that, at the time. Check the court’s schedule to see the bookings that were made.'

export function formatOccurrenceOutcome(outcome: OccurrenceOutcome): string {
  switch (outcome) {
    case 'RECURRING_HIRE_OCCURRENCE_OUTCOME_BOOKED':
      return 'Booked'
    case 'RECURRING_HIRE_OCCURRENCE_OUTCOME_SKIPPED_CONFLICT':
      return 'Skipped — court already booked'
    case 'RECURRING_HIRE_OCCURRENCE_OUTCOME_SKIPPED_ERROR':
      return 'Skipped — could not be booked'
    default:
      return 'Outcome not specified'
  }
}

export interface OccurrenceSummary {
  total: number
  booked: number
  skippedConflict: number
  skippedError: number
  /** Anything the app doesn't recognise — counted rather than folded into one
   * of the known buckets, so an unfamiliar outcome can never be reported as a
   * booked week. */
  unknown: number
}

export function summariseOccurrences(occurrences: OccurrenceView[]): OccurrenceSummary {
  const summary: OccurrenceSummary = { total: occurrences.length, booked: 0, skippedConflict: 0, skippedError: 0, unknown: 0 }
  for (const occurrence of occurrences) {
    switch (occurrence.outcome) {
      case 'RECURRING_HIRE_OCCURRENCE_OUTCOME_BOOKED':
        summary.booked += 1
        break
      case 'RECURRING_HIRE_OCCURRENCE_OUTCOME_SKIPPED_CONFLICT':
        summary.skippedConflict += 1
        break
      case 'RECURRING_HIRE_OCCURRENCE_OUTCOME_SKIPPED_ERROR':
        summary.skippedError += 1
        break
      default:
        summary.unknown += 1
    }
  }
  return summary
}

/**
 * The one-line result of an approval, for the live region (WCAG 4.1.3).
 *
 * Always states the booked count out of the total — never a bare "Approved",
 * which is precisely what T11.6 instruction #2 forbids. Skipped weeks are
 * named with their reason class, and a partial result is not dressed up as a
 * complete one.
 */
export function summariseApproval(summary: OccurrenceSummary): string {
  if (summary.total === 0) {
    return 'Approved. This request implied no weeks to book — check its dates.'
  }
  const parts = [`Approved. ${summary.booked} of ${summary.total} weeks booked.`]
  if (summary.skippedConflict > 0) {
    parts.push(
      `${summary.skippedConflict} skipped because the court was already booked at that time.`,
    )
  }
  if (summary.skippedError > 0) {
    parts.push(`${summary.skippedError} could not be booked — see the list for the reason given.`)
  }
  if (summary.unknown > 0) {
    parts.push(`${summary.unknown} returned an outcome this app does not recognise.`)
  }
  return parts.join(' ')
}
