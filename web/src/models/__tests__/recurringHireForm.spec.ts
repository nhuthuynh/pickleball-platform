// T11.6 — the Club rental request form's pure validation and wire
// translation. Table-driven (CLAUDE.md rule 1), and kept away from the
// component so the WCAG 3.3.1/3.3.3 messages are pinned as data rather than
// only observed through a mount.
import { describe, it, expect } from 'vitest'
import {
  emptyRecurringHireForm,
  validateRecurringHireForm,
  toRequestRecurringHireBody,
  toMinuteOfDay,
  type RecurringHireFormInput,
} from '../recurringHireForm'

function validForm(overrides: Partial<RecurringHireFormInput> = {}): RecurringHireFormInput {
  return {
    ...emptyRecurringHireForm(),
    courtId: '0f3c0c9e-0000-4000-8000-000000000001',
    weekday: 1,
    startTime: '09:00',
    endTime: '10:00',
    startsAt: '2026-09-07',
    ...overrides,
  }
}

describe('toMinuteOfDay', () => {
  it.each([
    ['00:00', 0],
    ['09:00', 540],
    ['10:15', 615],
    ['23:59', 1439],
  ])('converts %s to %i', (clock, expected) => {
    expect(toMinuteOfDay(clock)).toBe(expected)
  })

  it.each(['', '9', '25:00', '09:60', 'nine', '09:0'])('rejects %s', (clock) => {
    expect(toMinuteOfDay(clock)).toBeNull()
  })
})

describe('validateRecurringHireForm', () => {
  it('accepts a complete request', () => {
    expect(validateRecurringHireForm(validForm())).toEqual({})
  })

  // WCAG 3.3.1 (identified in text) + 3.3.3 (with a suggestion). Every message
  // has to say what a correct value looks like, not just that this one is
  // wrong.
  it.each([
    ['courtId', validForm({ courtId: '' }), /pick a facility first/i],
    ['weekday', validForm({ weekday: '' }), /for example, Mondays/i],
    ['startTime', validForm({ startTime: '' }), /for example, 09:00/i],
    ['endTime', validForm({ endTime: 'half nine' }), /for example, 10:30/i],
    ['startsAt', validForm({ startsAt: '' }), /YYYY-MM-DD/],
  ])('identifies a missing %s with a suggested fix', (field, input, pattern) => {
    const errors = validateRecurringHireForm(input) as Record<string, string | undefined>

    expect(errors[field]).toBeDefined()
    expect(errors[field]).toMatch(pattern)
    expect(errors[field]).not.toMatch(/^invalid$/i)
  })

  // Mirrors domain.ErrInvalidRecurringHireTimeRange rather than adding a
  // client-only rule, and names the value it is comparing against.
  it.each([
    ['equal', '09:00'],
    ['earlier', '08:00'],
  ])('rejects an end time %s than the start, quoting the start', (_label, endTime) => {
    const errors = validateRecurringHireForm(validForm({ endTime }))

    expect(errors.endTime).toMatch(/later than the start time \(09:00\)/)
  })

  it('requires an end date only when the request ends on a date, and it must be after the start', () => {
    const missing = validateRecurringHireForm(
      validForm({ endKind: 'RECURRING_HIRE_END_CONDITION_KIND_END_AFTER_DATE' }),
    )
    expect(missing.endDate).toMatch(/no end date/i)

    const tooEarly = validateRecurringHireForm(
      validForm({
        endKind: 'RECURRING_HIRE_END_CONDITION_KIND_END_AFTER_DATE',
        endDate: '2026-09-01',
      }),
    )
    expect(tooEarly.endDate).toMatch(/after the start date \(2026-09-07\)/)

    const fine = validateRecurringHireForm(
      validForm({
        endKind: 'RECURRING_HIRE_END_CONDITION_KIND_END_AFTER_DATE',
        endDate: '2026-12-01',
      }),
    )
    expect(fine.endDate).toBeUndefined()
  })

  // Mirrors domain.ErrInvalidRecurringHireEndAfterOccurrences.
  it.each(['', 0, -1, 1.5])('rejects a week count of %s', (occurrences) => {
    const errors = validateRecurringHireForm(
      validForm({
        endKind: 'RECURRING_HIRE_END_CONDITION_KIND_END_AFTER_OCCURRENCES',
        occurrences,
      }),
    )

    expect(errors.occurrences).toMatch(/whole number of 1 or more/i)
  })

  // `v-model` on <input type="number"> hands back a NUMBER when the text
  // parses and the raw string otherwise — the exact shape that made an earlier
  // form's validator throw a TypeError (see discountForm.ts's NumericFormValue
  // note). Both shapes must work.
  it('handles numeric fields arriving as either a string or a number', () => {
    for (const occurrences of ['12', 12]) {
      const errors = validateRecurringHireForm(
        validForm({
          endKind: 'RECURRING_HIRE_END_CONDITION_KIND_END_AFTER_OCCURRENCES',
          occurrences,
        }),
      )
      expect(errors.occurrences).toBeUndefined()
    }
    for (const weekday of ['0', 0]) {
      expect(validateRecurringHireForm(validForm({ weekday })).weekday).toBeUndefined()
    }
  })
})

describe('toRequestRecurringHireBody', () => {
  it('converts clock times to minutes and the date to RFC3339', () => {
    const body = toRequestRecurringHireBody(validForm(), 'actor-1')

    expect(body).toEqual({
      actorUserId: 'actor-1',
      courtId: '0f3c0c9e-0000-4000-8000-000000000001',
      weekday: 1,
      startMinute: 540,
      endMinute: 600,
      startsAt: '2026-09-07T00:00:00.000Z',
      endCondition: { kind: 'RECURRING_HIRE_END_CONDITION_KIND_NO_END' },
    })
  })

  // booking.proto: "only the field matching `kind` is meaningful". Sending a
  // defaulted zero alongside it would be shipping a value the server has to
  // ignore, and would read as a real end condition to anyone inspecting it.
  it('sends only the end-condition field matching the chosen kind', () => {
    const byDate = toRequestRecurringHireBody(
      validForm({
        endKind: 'RECURRING_HIRE_END_CONDITION_KIND_END_AFTER_DATE',
        endDate: '2026-12-01',
        occurrences: 12,
      }),
      'actor-1',
    )
    expect(byDate.endCondition).toEqual({
      kind: 'RECURRING_HIRE_END_CONDITION_KIND_END_AFTER_DATE',
      endDate: '2026-12-01T00:00:00.000Z',
    })

    const byCount = toRequestRecurringHireBody(
      validForm({
        endKind: 'RECURRING_HIRE_END_CONDITION_KIND_END_AFTER_OCCURRENCES',
        occurrences: '12',
        endDate: '2026-12-01',
      }),
      'actor-1',
    )
    expect(byCount.endCondition).toEqual({
      kind: 'RECURRING_HIRE_END_CONDITION_KIND_END_AFTER_OCCURRENCES',
      occurrences: 12,
    })
  })

  // Sprint plan A4 checklist item 2 (the one that fires for this flow), in
  // client form: the request carries no role/isClub claim for the server to
  // believe. Whether this actor may request as a Club is resolved server-side
  // from their real Roles.
  it('never sends a self-declared role or club flag', () => {
    const body = toRequestRecurringHireBody(validForm(), 'actor-1') as Record<string, unknown>

    for (const key of Object.keys(body)) {
      expect(key).not.toMatch(/role|club|isClub/i)
    }
  })
})
