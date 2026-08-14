// T11.3 (docs/process/t11-sprint-plan.md) — the honest-rendering rules for a
// DiscountRule, tested table-driven per CLAUDE.md rule 1.
//
// Instruction #3 ("no fabricated fields") is what most of this file exists to
// pin down. The shapes asserted here were read off the REAL generated client
// (web/src/api/generated/booking.d.ts, produced by `make generate` +
// `npm run generate:client` from proto/pickleball/booking/v1/booking.proto),
// not assumed:
//
//   v1EndCondition: { kind?, endDate?, occurrences? }
//
// There is NO remaining-count/uses-so-far field anywhere on that message, on
// v1DiscountRule, or on v1ListDiscountRulesForFacilityResponse. So
// END_AFTER_OCCURRENCES renders the honest total ("ends after N total uses")
// and this ticket deliberately does NOT invent a live remaining counter it
// has no backend support for.
import { describe, it, expect } from 'vitest'
import {
  mapToDiscountRule,
  formatDiscountAmount,
  formatEndCondition,
  formatAppliesTo,
  type RawDiscountRule,
} from '../discount'

describe('formatEndCondition — never fabricates a date or a count', () => {
  const mediumDate = (iso: string) =>
    new Intl.DateTimeFormat(undefined, { dateStyle: 'medium' }).format(new Date(iso))

  const cases: { name: string; input: RawDiscountRule['endCondition']; expected: string }[] = [
    {
      name: 'NO_END renders "no end date", never a fabricated date',
      input: { kind: 'END_CONDITION_KIND_NO_END' },
      expected: 'No end date',
    },
    {
      name: 'NO_END ignores a stray endDate rather than rendering it',
      // Only the field matching `kind` is meaningful (v1EndCondition's own
      // doc comment) — a NoEnd rule that somehow carries an endDate must
      // still say "no end date", not surface the meaningless value.
      input: { kind: 'END_CONDITION_KIND_NO_END', endDate: '2027-01-01T00:00:00Z' },
      expected: 'No end date',
    },
    {
      name: 'END_AFTER_DATE renders the real date it was given',
      input: { kind: 'END_CONDITION_KIND_END_AFTER_DATE', endDate: '2026-12-31T00:00:00Z' },
      expected: `Ends ${mediumDate('2026-12-31T00:00:00Z')}`,
    },
    {
      name: 'END_AFTER_DATE with no date says so instead of inventing one',
      input: { kind: 'END_CONDITION_KIND_END_AFTER_DATE' },
      expected: 'End date not provided',
    },
    {
      name: 'END_AFTER_OCCURRENCES renders the TOTAL, labelled as a total — not a remaining count',
      input: { kind: 'END_CONDITION_KIND_END_AFTER_OCCURRENCES', occurrences: 10 },
      expected: 'Ends after 10 total uses',
    },
    {
      name: 'END_AFTER_OCCURRENCES singular reads correctly',
      input: { kind: 'END_CONDITION_KIND_END_AFTER_OCCURRENCES', occurrences: 1 },
      expected: 'Ends after 1 total use',
    },
    {
      name: 'END_AFTER_OCCURRENCES with no count says so instead of inventing one',
      input: { kind: 'END_CONDITION_KIND_END_AFTER_OCCURRENCES' },
      expected: 'Ends after a set number of total uses (count not provided)',
    },
    {
      name: 'an absent end condition is reported as unset, NOT silently as "no end date"',
      // "No end date" is a real, specific statement a Facility Owner chose
      // (END_CONDITION_KIND_NO_END). An absent/unspecified condition is the
      // ABSENCE of that statement — collapsing the two would put words in
      // the backend's mouth, the same distinction booking.proto draws for an
      // absent vs. zero-valued discount.
      input: undefined,
      expected: 'End condition not set',
    },
    {
      name: 'an explicitly UNSPECIFIED kind is likewise reported as unset',
      input: { kind: 'END_CONDITION_KIND_UNSPECIFIED' },
      expected: 'End condition not set',
    },
  ]

  it.each(cases)('$name', ({ input, expected }) => {
    expect(formatEndCondition(input)).toBe(expected)
  })

  it('never renders the words "remaining" or "left" — no read path returns a remaining count', () => {
    for (const { input } of cases) {
      const rendered = formatEndCondition(input).toLowerCase()
      expect(rendered).not.toContain('remaining')
      expect(rendered).not.toContain('left')
    }
  })
})

describe('formatDiscountAmount', () => {
  const cases: { name: string; input: RawDiscountRule; expected: string }[] = [
    {
      name: 'percent discount',
      input: { discountType: 'DISCOUNT_TYPE_PERCENT', percent: 15 },
      expected: '15% off',
    },
    {
      name: 'fractional percent keeps its precision',
      input: { discountType: 'DISCOUNT_TYPE_PERCENT', percent: 12.5 },
      expected: '12.5% off',
    },
    {
      name: 'fixed USD amount formats as dollars (int64 arrives as a STRING over protojson)',
      input: { discountType: 'DISCOUNT_TYPE_FIXED_AMOUNT', fixedAmountCents: '500', currency: 'USD' },
      expected: '$5.00 off',
    },
    {
      name: 'a non-USD currency is never rendered with a "$" sign',
      input: { discountType: 'DISCOUNT_TYPE_FIXED_AMOUNT', fixedAmountCents: '500', currency: 'EUR' },
      expected: '5.00 EUR off',
    },
    {
      name: 'an unspecified discount type claims no number at all',
      input: { discountType: 'DISCOUNT_TYPE_UNSPECIFIED' },
      expected: 'Discount applied',
    },
    {
      name: 'a rule with no discountType at all claims no number either',
      input: {},
      expected: 'Discount applied',
    },
  ]

  it.each(cases)('$name', ({ input, expected }) => {
    expect(formatDiscountAmount(mapToDiscountRule(input))).toBe(expected)
  })
})

describe('formatAppliesTo', () => {
  it('lists the booking types a rule applies to in readable words', () => {
    expect(formatAppliesTo(['SOURCE_INDIVIDUAL', 'SOURCE_GAME'])).toBe('Individual bookings, Games')
  })

  it('reports an empty list honestly rather than implying "everything"', () => {
    expect(formatAppliesTo([])).toBe('Not specified')
  })
})

describe('mapToDiscountRule', () => {
  it('parses the int64 fixedAmountCents string into a number and defaults everything absent', () => {
    const rule = mapToDiscountRule({
      id: 'discount-1',
      facilityId: 'facility-1',
      discountType: 'DISCOUNT_TYPE_FIXED_AMOUNT',
      fixedAmountCents: '750',
      currency: 'USD',
      appliesTo: ['SOURCE_INDIVIDUAL'],
      startsAt: '2026-09-01T00:00:00Z',
      endCondition: { kind: 'END_CONDITION_KIND_NO_END' },
    })

    expect(rule.id).toBe('discount-1')
    expect(rule.fixedAmountCents).toBe(750)
    expect(rule.percent).toBe(0)
    expect(rule.appliesTo).toEqual(['SOURCE_INDIVIDUAL'])
    expect(rule.endCondition).toEqual({ kind: 'END_CONDITION_KIND_NO_END' })
  })

  it('keeps an absent endCondition absent rather than substituting a NoEnd default', () => {
    expect(mapToDiscountRule({ id: 'd' }).endCondition).toBeUndefined()
  })
})
