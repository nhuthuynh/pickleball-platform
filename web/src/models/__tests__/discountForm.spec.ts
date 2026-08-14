// T11.3 instruction #4 — WCAG 2.2 AA 3.3.1 (Error Identification) and 3.3.3
// (Error Suggestion) for the Facility Owner's discount form, tested as pure
// functions so the *content* of every message is pinned, not just that "an
// error appeared".
//
// 3.3.3 is the demanding one: every message below must tell the Owner how to
// FIX the input, not merely that it is invalid. The last test in this file
// enforces that as a blanket rule over every message the validator can emit,
// so a future field can't quietly ship a bare "Invalid".
import { describe, it, expect } from 'vitest'
import {
  validateDiscountForm,
  toCreateDiscountRuleBody,
  emptyDiscountForm,
  type DiscountFormInput,
} from '../discountForm'

function form(overrides: Partial<DiscountFormInput> = {}): DiscountFormInput {
  return {
    ...emptyDiscountForm(),
    discountType: 'DISCOUNT_TYPE_PERCENT',
    percent: '15',
    appliesTo: ['SOURCE_INDIVIDUAL'],
    startsAt: '2026-09-01',
    endKind: 'END_CONDITION_KIND_NO_END',
    ...overrides,
  }
}

describe('validateDiscountForm', () => {
  it('accepts a well-formed percent discount', () => {
    expect(validateDiscountForm(form())).toEqual({})
  })

  it('accepts a well-formed fixed-amount discount', () => {
    const errors = validateDiscountForm(
      form({ discountType: 'DISCOUNT_TYPE_FIXED_AMOUNT', percent: '', amount: '5.00', currency: 'USD' }),
    )
    expect(errors).toEqual({})
  })

  const cases: { name: string; input: DiscountFormInput; field: string; expected: string }[] = [
    {
      name: 'empty percent',
      input: form({ percent: '' }),
      field: 'percent',
      expected: 'Enter a percentage greater than 0 and up to 100 — for example, 15 for 15% off.',
    },
    {
      name: 'percent of 0 (the proto requires (0, 100])',
      input: form({ percent: '0' }),
      field: 'percent',
      expected: 'Enter a percentage greater than 0 and up to 100 — for example, 15 for 15% off.',
    },
    {
      name: 'percent above 100',
      input: form({ percent: '150' }),
      field: 'percent',
      expected: 'Enter a percentage greater than 0 and up to 100 — for example, 15 for 15% off.',
    },
    {
      name: 'non-numeric percent',
      input: form({ percent: 'abc' }),
      field: 'percent',
      expected: 'Enter a percentage greater than 0 and up to 100 — for example, 15 for 15% off.',
    },
    {
      name: 'zero fixed amount',
      input: form({ discountType: 'DISCOUNT_TYPE_FIXED_AMOUNT', amount: '0', currency: 'USD' }),
      field: 'amount',
      expected: 'Enter an amount greater than 0 — for example, 5.00 for five dollars off.',
    },
    {
      name: 'missing currency on a fixed-amount discount',
      input: form({ discountType: 'DISCOUNT_TYPE_FIXED_AMOUNT', amount: '5.00', currency: '' }),
      field: 'currency',
      expected: 'Enter a 3-letter currency code — for example, USD.',
    },
    {
      name: 'malformed currency code',
      input: form({ discountType: 'DISCOUNT_TYPE_FIXED_AMOUNT', amount: '5.00', currency: 'dollars' }),
      field: 'currency',
      expected: 'Enter a 3-letter currency code — for example, USD.',
    },
    {
      name: 'no booking types selected',
      input: form({ appliesTo: [] }),
      field: 'appliesTo',
      expected: 'Select at least one booking type this discount applies to, for example Individual bookings.',
    },
    {
      name: 'missing start date',
      input: form({ startsAt: '' }),
      field: 'startsAt',
      expected: 'Choose the date this discount starts, in the format YYYY-MM-DD.',
    },
    {
      name: 'end-after-date with no date',
      input: form({ endKind: 'END_CONDITION_KIND_END_AFTER_DATE', endDate: '' }),
      field: 'endDate',
      expected: 'Choose the date this discount ends, or select "No end date" instead.',
    },
    {
      name: 'end date on or before the start date',
      input: form({ endKind: 'END_CONDITION_KIND_END_AFTER_DATE', endDate: '2026-09-01' }),
      field: 'endDate',
      expected: 'Choose an end date after the start date (2026-09-01).',
    },
    {
      name: 'end-after-occurrences with no count',
      input: form({ endKind: 'END_CONDITION_KIND_END_AFTER_OCCURRENCES', occurrences: '' }),
      field: 'occurrences',
      expected: 'Enter how many total uses this discount allows — a whole number of 1 or more, for example 10.',
    },
    {
      name: 'end-after-occurrences with a fractional count',
      input: form({ endKind: 'END_CONDITION_KIND_END_AFTER_OCCURRENCES', occurrences: '2.5' }),
      field: 'occurrences',
      expected: 'Enter how many total uses this discount allows — a whole number of 1 or more, for example 10.',
    },
    {
      name: 'no end condition chosen',
      input: form({ endKind: 'END_CONDITION_KIND_UNSPECIFIED' }),
      field: 'endKind',
      expected: 'Choose when this discount ends — pick "No end date" if it should run indefinitely.',
    },
  ]

  it.each(cases)('$name is identified on the right field with a fix suggestion', ({ input, field, expected }) => {
    const errors = validateDiscountForm(input) as Record<string, string | undefined>
    expect(errors[field]).toBe(expected)
  })

  // REGRESSION (found by FacilityDiscounts.spec.ts while building this
  // ticket): Vue's `v-model` on an `<input type="number">` hands back a
  // NUMBER, not a string, whenever the entered text parses. The validator
  // called `.trim()` on these fields unconditionally and threw
  // "input.percent.trim is not a function" — the form silently did nothing
  // on submit. Numeric fields are typed `string | number` and normalised
  // through `asText` for exactly this reason.
  it('accepts numeric-typed values from a number input, not just strings', () => {
    expect(validateDiscountForm(form({ percent: 15 }))).toEqual({})
    expect(
      validateDiscountForm(form({ discountType: 'DISCOUNT_TYPE_FIXED_AMOUNT', amount: 5, currency: 'USD' })),
    ).toEqual({})
    expect(
      validateDiscountForm(form({ endKind: 'END_CONDITION_KIND_END_AFTER_OCCURRENCES', occurrences: 10 })),
    ).toEqual({})
  })

  it('still rejects out-of-range NUMERIC values, not just out-of-range strings', () => {
    expect(validateDiscountForm(form({ percent: 0 })).percent).toBeTruthy()
    expect(validateDiscountForm(form({ percent: 150 })).percent).toBeTruthy()
    expect(
      validateDiscountForm(form({ endKind: 'END_CONDITION_KIND_END_AFTER_OCCURRENCES', occurrences: 0 }))
        .occurrences,
    ).toBeTruthy()
  })

  it('builds a correct request body from numeric-typed values too', () => {
    const body = toCreateDiscountRuleBody(
      form({ discountType: 'DISCOUNT_TYPE_FIXED_AMOUNT', amount: 5, currency: 'USD' }),
      'owner-1',
    )
    expect(body.fixedAmountCents).toBe('500')
  })

  it('does not validate percent when the discount is fixed-amount (and vice versa)', () => {
    // Only the field matching `discount_type` is meaningful (booking.proto),
    // so the irrelevant one must never block submission.
    expect(
      validateDiscountForm(form({ discountType: 'DISCOUNT_TYPE_FIXED_AMOUNT', percent: '', amount: '5', currency: 'USD' }))
        .percent,
    ).toBeUndefined()
    expect(validateDiscountForm(form({ percent: '15', amount: '', currency: '' })).amount).toBeUndefined()
  })

  it('every message it can emit suggests a fix rather than only saying something is wrong (3.3.3)', () => {
    for (const { input } of cases) {
      for (const message of Object.values(validateDiscountForm(input))) {
        expect(message).toBeTruthy()
        // A suggestion names an example value, a required format, or a
        // concrete alternative action ("...instead", "pick X"); a bare
        // "Invalid"/"Required" does none of those.
        expect(message!.toLowerCase()).toMatch(/for example|instead|format|after the start date|pick "/)
        expect(message!.toLowerCase()).not.toMatch(/^(invalid|required|error)\.?$/)
      }
    }
  })
})

describe('toCreateDiscountRuleBody', () => {
  it('sends only the amount field matching the discount type, with cents as an int64 STRING', () => {
    const body = toCreateDiscountRuleBody(
      form({ discountType: 'DISCOUNT_TYPE_FIXED_AMOUNT', percent: '', amount: '5.00', currency: 'usd' }),
      'owner-1',
    )

    expect(body.actorUserId).toBe('owner-1')
    expect(body.discountType).toBe('DISCOUNT_TYPE_FIXED_AMOUNT')
    // int64 over protojson is a string — see models/discount.ts.
    expect(body.fixedAmountCents).toBe('500')
    // Currency is normalised to the uppercase ISO 4217 form the proto asks for.
    expect(body.currency).toBe('USD')
    expect(body.percent).toBeUndefined()
  })

  it('sends percent (a real number, not a string) and no money fields for a percent discount', () => {
    const body = toCreateDiscountRuleBody(form({ percent: '15' }), 'owner-1')
    expect(body.percent).toBe(15)
    expect(body.fixedAmountCents).toBeUndefined()
    expect(body.currency).toBeUndefined()
  })

  it('sends startsAt as an RFC3339 timestamp, not the raw date-input value', () => {
    const body = toCreateDiscountRuleBody(form({ startsAt: '2026-09-01' }), 'owner-1')
    expect(body.startsAt).toBe(new Date('2026-09-01').toISOString())
  })

  it('sends only the EndCondition field matching its kind — NoEnd carries no date', () => {
    const body = toCreateDiscountRuleBody(form({ endKind: 'END_CONDITION_KIND_NO_END', endDate: '2027-01-01' }), 'o')
    expect(body.endCondition).toEqual({ kind: 'END_CONDITION_KIND_NO_END' })
  })

  it('sends occurrences as a number for an EndAfterOccurrences rule', () => {
    const body = toCreateDiscountRuleBody(
      form({ endKind: 'END_CONDITION_KIND_END_AFTER_OCCURRENCES', occurrences: '10' }),
      'o',
    )
    expect(body.endCondition).toEqual({ kind: 'END_CONDITION_KIND_END_AFTER_OCCURRENCES', occurrences: 10 })
  })

  it('sends endDate as an RFC3339 timestamp for an EndAfterDate rule', () => {
    const body = toCreateDiscountRuleBody(
      form({ endKind: 'END_CONDITION_KIND_END_AFTER_DATE', endDate: '2026-12-31' }),
      'o',
    )
    expect(body.endCondition).toEqual({
      kind: 'END_CONDITION_KIND_END_AFTER_DATE',
      endDate: new Date('2026-12-31').toISOString(),
    })
  })
})
