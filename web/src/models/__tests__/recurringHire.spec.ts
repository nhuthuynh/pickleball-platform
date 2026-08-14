// T11.6 — the recurring-hire view models and formatters.
//
// These are the honesty rules in test form. Each one exists because the
// obvious implementation of the same formatter would have fabricated
// something: a defaulted weekday of 0 reads as a real "Sundays" schedule, a
// bare "Approved" hides skipped weeks, and a rejected request that stops
// saying it is final invites the reader to wait for a decision that will never
// come (T11.6 instruction #4).
import { describe, it, expect } from 'vitest'
import {
  mapToRecurringHireTemplate,
  mapToOccurrence,
  formatWeekday,
  formatMinuteOfDay,
  formatTimeRange,
  formatDate,
  formatRecurringEndCondition,
  formatStatus,
  statusExplanation,
  isDecided,
  formatOccurrenceOutcome,
  summariseOccurrences,
  summariseApproval,
  type OccurrenceView,
  type RecurringHireStatus,
  type RawRecurringHireEndCondition,
} from '../recurringHire'

describe('mapToRecurringHireTemplate', () => {
  it('keeps every field the server actually sent', () => {
    const view = mapToRecurringHireTemplate({
      id: 'template-1',
      requestedByUserId: 'club-1',
      courtId: 'court-1',
      weekday: 1,
      startMinute: 540,
      endMinute: 600,
      startsAt: '2026-09-07T00:00:00Z',
      endCondition: { kind: 'RECURRING_HIRE_END_CONDITION_KIND_END_AFTER_OCCURRENCES', occurrences: 12 },
      status: 'RECURRING_HIRE_STATUS_REQUESTED',
    })

    expect(view).toEqual({
      id: 'template-1',
      requestedByUserId: 'club-1',
      courtId: 'court-1',
      weekday: 1,
      startMinute: 540,
      endMinute: 600,
      startsAt: '2026-09-07T00:00:00Z',
      endCondition: { kind: 'RECURRING_HIRE_END_CONDITION_KIND_END_AFTER_OCCURRENCES', occurrences: 12 },
      status: 'RECURRING_HIRE_STATUS_REQUESTED',
    })
  })

  // The one that matters: every proto scalar is optional over protojson, and
  // `weekday: 0` is a REAL value (Sunday). `?? 0` would make an absent weekday
  // indistinguishable from a Sunday booking.
  it('maps an absent weekday/time to null rather than defaulting it to 0', () => {
    const view = mapToRecurringHireTemplate({ id: 'template-1' })

    expect(view.weekday).toBeNull()
    expect(view.startMinute).toBeNull()
    expect(view.endMinute).toBeNull()
    expect(formatWeekday(view.weekday)).toBe('Not specified')
    expect(formatTimeRange(view.startMinute, view.endMinute)).toBe('Time not specified')
  })

  it('preserves a real zero (Sunday, midnight) rather than treating it as absent', () => {
    const view = mapToRecurringHireTemplate({ weekday: 0, startMinute: 0, endMinute: 60 })

    expect(view.weekday).toBe(0)
    expect(formatWeekday(view.weekday)).toBe('Sundays')
    expect(formatTimeRange(view.startMinute, view.endMinute)).toBe('00:00–01:00')
  })
})

describe('formatMinuteOfDay', () => {
  it.each([
    [0, '00:00'],
    [540, '09:00'],
    [615, '10:15'],
    [1439, '23:59'],
  ])('formats %i as %s', (minute, expected) => {
    expect(formatMinuteOfDay(minute)).toBe(expected)
  })

  it.each([null, -1, 2000, 1.5])('returns an empty string for the unusable value %s', (minute) => {
    expect(formatMinuteOfDay(minute as number | null)).toBe('')
  })
})

describe('formatDate', () => {
  it('renders the date part of an RFC3339 timestamp', () => {
    expect(formatDate('2026-09-07T09:00:00Z')).toBe('2026-09-07')
  })

  // Never today's date as a stand-in — an empty string is what lets a caller
  // render "Not specified" instead of a plausible-looking wrong date.
  it.each(['', 'not-a-date'])('returns an empty string for %s', (value) => {
    expect(formatDate(value)).toBe('')
  })
})

describe('formatRecurringEndCondition', () => {
  it.each<[RawRecurringHireEndCondition, string]>([
    [
      { kind: 'RECURRING_HIRE_END_CONDITION_KIND_END_AFTER_DATE', endDate: '2026-12-01T00:00:00Z' },
      'Until 2026-12-01',
    ],
    [
      { kind: 'RECURRING_HIRE_END_CONDITION_KIND_END_AFTER_OCCURRENCES', occurrences: 12 },
      'For 12 weeks',
    ],
    [
      { kind: 'RECURRING_HIRE_END_CONDITION_KIND_END_AFTER_OCCURRENCES', occurrences: 1 },
      'For 1 week',
    ],
    [{ kind: 'RECURRING_HIRE_END_CONDITION_KIND_NO_END' }, 'Ongoing, with no end date'],
  ])('formats %o', (condition, expected) => {
    expect(formatRecurringEndCondition(condition)).toBe(expected)
  })

  it('does not invent a date or a count when the payload is missing', () => {
    expect(
      formatRecurringEndCondition({ kind: 'RECURRING_HIRE_END_CONDITION_KIND_END_AFTER_DATE' }),
    ).toBe('Until a set date (date not specified)')
    expect(
      formatRecurringEndCondition({
        kind: 'RECURRING_HIRE_END_CONDITION_KIND_END_AFTER_OCCURRENCES',
      }),
    ).toBe('For a set number of weeks (count not specified)')
    expect(formatRecurringEndCondition(null)).toBe('Not specified')
  })

  // "Ongoing", not "forever": the server applies a generation horizon at
  // approval time (app.recurringHireHorizonYears), so an unbounded promise
  // would be one the backend does not make.
  it('does not promise an unbounded booking for an open-ended request', () => {
    expect(formatRecurringEndCondition({ kind: 'RECURRING_HIRE_END_CONDITION_KIND_NO_END' })).not.toMatch(
      /forever|always|permanent/i,
    )
  })
})

describe('status rendering', () => {
  it.each([
    ['RECURRING_HIRE_STATUS_REQUESTED', 'Awaiting a decision', false],
    ['RECURRING_HIRE_STATUS_APPROVED', 'Approved', true],
    ['RECURRING_HIRE_STATUS_REJECTED', 'Rejected', true],
    ['RECURRING_HIRE_STATUS_CANCELLED', 'Cancelled', true],
  ])('%s formats as %s and isDecided=%s', (status, label, decided) => {
    expect(formatStatus(status as RecurringHireStatus)).toBe(label)
    expect(isDecided(status as RecurringHireStatus)).toBe(decided)
  })

  // T11.6 instruction #4, at the level where the words are actually chosen: a
  // rejection is FINAL and getting the slot needs a NEW request. It must not
  // read as something still pending or re-openable.
  it('states that a rejection is final and needs a fresh request', () => {
    const explanation = statusExplanation('RECURRING_HIRE_STATUS_REJECTED')

    expect(explanation).toMatch(/final/i)
    expect(explanation).toMatch(/new request/i)
    expect(explanation).not.toMatch(/pending|awaiting|may still|could still be approved/i)
  })

  it('does not claim a pending request has booked anything', () => {
    expect(statusExplanation('RECURRING_HIRE_STATUS_REQUESTED')).toMatch(/no courts are booked/i)
  })

  it('refuses to guess at an unrecognised status', () => {
    expect(formatStatus('RECURRING_HIRE_STATUS_UNSPECIFIED')).toBe('Status not specified')
    expect(statusExplanation('RECURRING_HIRE_STATUS_UNSPECIFIED')).toMatch(/does not|no status/i)
  })
})

function occurrence(overrides: Partial<OccurrenceView> = {}): OccurrenceView {
  return {
    startsAt: '2026-09-07T09:00:00Z',
    endsAt: '2026-09-07T10:00:00Z',
    outcome: 'RECURRING_HIRE_OCCURRENCE_OUTCOME_BOOKED',
    bookingId: 'booking-1',
    reason: '',
    ...overrides,
  }
}

describe('occurrence reporting', () => {
  it('maps an occurrence without inventing a booking id for a skipped week', () => {
    const view = mapToOccurrence({
      startsAt: '2026-09-14T09:00:00Z',
      endsAt: '2026-09-14T10:00:00Z',
      outcome: 'RECURRING_HIRE_OCCURRENCE_OUTCOME_SKIPPED_CONFLICT',
      reason: 'court already booked',
    })

    expect(view.bookingId).toBe('')
    expect(view.reason).toBe('court already booked')
  })

  it('keeps a conflict and a backend fault distinguishable', () => {
    expect(formatOccurrenceOutcome('RECURRING_HIRE_OCCURRENCE_OUTCOME_SKIPPED_CONFLICT')).toMatch(
      /already booked/i,
    )
    expect(formatOccurrenceOutcome('RECURRING_HIRE_OCCURRENCE_OUTCOME_SKIPPED_ERROR')).not.toMatch(
      /already booked/i,
    )
  })

  it('counts each outcome separately and never folds an unknown one into booked', () => {
    const summary = summariseOccurrences([
      occurrence(),
      occurrence(),
      occurrence({ outcome: 'RECURRING_HIRE_OCCURRENCE_OUTCOME_SKIPPED_CONFLICT' }),
      occurrence({ outcome: 'RECURRING_HIRE_OCCURRENCE_OUTCOME_SKIPPED_ERROR' }),
      occurrence({ outcome: 'RECURRING_HIRE_OCCURRENCE_OUTCOME_UNSPECIFIED' }),
    ])

    expect(summary).toEqual({ total: 5, booked: 2, skippedConflict: 1, skippedError: 1, unknown: 1 })
  })
})

describe('summariseApproval', () => {
  // T11.6 instruction #2: never a bare "approved" with no visibility into
  // partial conflicts. The counts are in the sentence, always.
  it('always reports booked-out-of-total, even when everything succeeded', () => {
    const message = summariseApproval(
      summariseOccurrences([occurrence(), occurrence(), occurrence()]),
    )

    expect(message).toContain('3 of 3 weeks booked')
  })

  it('names the skipped weeks and why, when the approval was partial', () => {
    const message = summariseApproval(
      summariseOccurrences([
        occurrence(),
        occurrence({ outcome: 'RECURRING_HIRE_OCCURRENCE_OUTCOME_SKIPPED_CONFLICT' }),
        occurrence({ outcome: 'RECURRING_HIRE_OCCURRENCE_OUTCOME_SKIPPED_ERROR' }),
      ]),
    )

    expect(message).toContain('1 of 3 weeks booked')
    expect(message).toMatch(/1 skipped because the court was already booked/i)
    expect(message).toMatch(/1 could not be booked/i)
  })

  it('does not silently report an empty approval as a success', () => {
    expect(summariseApproval(summariseOccurrences([]))).toMatch(/no weeks to book/i)
  })
})
