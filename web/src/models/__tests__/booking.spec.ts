import { describe, it, expect } from 'vitest'
import {
  mapToQuote,
  mapToBooking,
  formatPriceCents,
  computeNextAvailableSlots,
  quoteSavingCents,
  type ConfirmedBooking,
} from '../booking'

describe('mapToQuote', () => {
  it('parses the int64-as-string priceCents AND bandPriceCents into numbers', () => {
    // T11.2 added bandPriceCents (the pre-discount band price) and discount
    // to GetQuoteResponse — both int64-as-string / message respectively.
    const quote = mapToQuote({ priceCents: '1530', band: 'peak', bandPriceCents: '1800' }, 1000)
    expect(quote).toEqual({
      priceCents: 1530,
      bandPriceCents: 1800,
      band: 'peak',
      discount: null,
      fetchedAt: 1000,
    })
  })

  it('defaults missing fields', () => {
    const quote = mapToQuote({}, 500)
    expect(quote).toEqual({ priceCents: 0, bandPriceCents: 0, band: '', discount: null, fetchedAt: 500 })
  })

  it('falls back bandPriceCents to priceCents when the server did not send one', () => {
    // A response with no bandPriceCents (e.g. a server predating T11.2) must
    // not be read as "the band price was $0.00" — booking.proto states the
    // two are equal when no discount applied, so mirror that rather than
    // rendering a fabricated zero as a struck-through original price.
    const quote = mapToQuote({ priceCents: '1800', band: 'peak' }, 1000)
    expect(quote.bandPriceCents).toBe(1800)
  })

  it('maps an applied discount, and reports ABSENCE as null (not a zero-valued rule)', () => {
    // booking.proto: "Absence, not a zero-valued rule, is what 'no discount'
    // looks like on the wire — a 0%-off discount and no discount are
    // different statements."
    const withDiscount = mapToQuote(
      {
        priceCents: '1530',
        band: 'peak',
        bandPriceCents: '1800',
        discount: { id: 'd-1', discountType: 'DISCOUNT_TYPE_PERCENT', percent: 15 },
      },
      1000,
    )
    expect(withDiscount.discount?.id).toBe('d-1')
    expect(withDiscount.discount?.percent).toBe(15)

    expect(mapToQuote({ priceCents: '1800', band: 'peak' }, 1000).discount).toBeNull()
  })
})

describe('quoteSavingCents', () => {
  it('reports the real saving when a discount actually reduced the price', () => {
    const quote = mapToQuote({
      priceCents: '1530',
      bandPriceCents: '1800',
      discount: { id: 'd-1', discountType: 'DISCOUNT_TYPE_PERCENT', percent: 15 },
    })
    expect(quoteSavingCents(quote)).toBe(270)
  })

  it('is 0 when no discount applied, so nothing can render a phantom saving', () => {
    expect(quoteSavingCents(mapToQuote({ priceCents: '1800', bandPriceCents: '1800' }))).toBe(0)
  })

  it('is 0 for a discount that reduced nothing, rather than a negative "saving"', () => {
    const quote = mapToQuote({
      priceCents: '1800',
      bandPriceCents: '1800',
      discount: { id: 'd-0', discountType: 'DISCOUNT_TYPE_PERCENT', percent: 0 },
    })
    expect(quoteSavingCents(quote)).toBe(0)
  })

  it('defaults fetchedAt to Date.now() when not given', () => {
    const before = Date.now()
    const quote = mapToQuote({ priceCents: '100', band: 'off-peak' })
    const after = Date.now()
    expect(quote.fetchedAt).toBeGreaterThanOrEqual(before)
    expect(quote.fetchedAt).toBeLessThanOrEqual(after)
  })
})

describe('mapToBooking', () => {
  it('maps a full raw booking', () => {
    const booking = mapToBooking({
      id: 'b1',
      courtId: 'c1',
      startsAt: '2026-08-10T09:00:00Z',
      endsAt: '2026-08-10T10:00:00Z',
      status: 'STATUS_CONFIRMED',
      referenceId: 'ref-1',
    })
    expect(booking).toEqual({
      id: 'b1',
      courtId: 'c1',
      startsAt: '2026-08-10T09:00:00Z',
      endsAt: '2026-08-10T10:00:00Z',
      status: 'STATUS_CONFIRMED',
      referenceId: 'ref-1',
    })
  })

  it('defaults missing fields', () => {
    const booking = mapToBooking({})
    expect(booking).toEqual({
      id: '',
      courtId: '',
      startsAt: '',
      endsAt: '',
      status: 'STATUS_UNSPECIFIED',
      referenceId: '',
    })
  })
})

describe('formatPriceCents', () => {
  it('formats whole dollars', () => {
    expect(formatPriceCents(1800)).toBe('$18.00')
  })

  it('formats fractional cents', () => {
    expect(formatPriceCents(12345)).toBe('$123.45')
  })

  it('formats zero', () => {
    expect(formatPriceCents(0)).toBe('$0.00')
  })
})

function booking(startsAt: string, endsAt: string, status = 'STATUS_CONFIRMED'): ConfirmedBooking {
  return { id: 'x', courtId: 'c1', startsAt, endsAt, status, referenceId: '' }
}

describe('computeNextAvailableSlots', () => {
  it('returns the very first candidate slot when nothing conflicts', () => {
    const slots = computeNextAvailableSlots({
      bookings: [],
      durationMs: 60 * 60 * 1000,
      searchFrom: new Date('2026-08-10T09:00:00Z'),
      searchUntil: new Date('2026-08-10T18:00:00Z'),
      limit: 1,
    })
    expect(slots).toEqual([{ startsAt: '2026-08-10T09:00:00.000Z', endsAt: '2026-08-10T10:00:00.000Z' }])
  })

  it('skips over an overlapping existing booking and returns the next free slot', () => {
    const slots = computeNextAvailableSlots({
      bookings: [booking('2026-08-10T09:00:00Z', '2026-08-10T10:00:00Z')],
      durationMs: 60 * 60 * 1000,
      searchFrom: new Date('2026-08-10T09:00:00Z'),
      searchUntil: new Date('2026-08-10T18:00:00Z'),
      limit: 1,
      stepMs: 30 * 60 * 1000,
    })
    // The 09:00-10:00 slot conflicts; 09:30-10:30 still conflicts (overlaps
    // 09:00-10:00); 10:00-11:00 is the first fully free slot.
    expect(slots).toEqual([{ startsAt: '2026-08-10T10:00:00.000Z', endsAt: '2026-08-10T11:00:00.000Z' }])
  })

  it('returns up to `limit` suggestions in chronological order', () => {
    const slots = computeNextAvailableSlots({
      bookings: [],
      durationMs: 60 * 60 * 1000,
      searchFrom: new Date('2026-08-10T09:00:00Z'),
      searchUntil: new Date('2026-08-10T18:00:00Z'),
      limit: 3,
      stepMs: 60 * 60 * 1000,
    })
    expect(slots).toEqual([
      { startsAt: '2026-08-10T09:00:00.000Z', endsAt: '2026-08-10T10:00:00.000Z' },
      { startsAt: '2026-08-10T10:00:00.000Z', endsAt: '2026-08-10T11:00:00.000Z' },
      { startsAt: '2026-08-10T11:00:00.000Z', endsAt: '2026-08-10T12:00:00.000Z' },
    ])
  })

  it('ignores cancelled bookings when checking for overlap', () => {
    const slots = computeNextAvailableSlots({
      bookings: [booking('2026-08-10T09:00:00Z', '2026-08-10T10:00:00Z', 'STATUS_CANCELLED')],
      durationMs: 60 * 60 * 1000,
      searchFrom: new Date('2026-08-10T09:00:00Z'),
      searchUntil: new Date('2026-08-10T18:00:00Z'),
      limit: 1,
    })
    expect(slots).toEqual([{ startsAt: '2026-08-10T09:00:00.000Z', endsAt: '2026-08-10T10:00:00.000Z' }])
  })

  it('returns an empty array when the whole search window is booked solid', () => {
    const slots = computeNextAvailableSlots({
      bookings: [booking('2026-08-10T09:00:00Z', '2026-08-10T18:00:00Z')],
      durationMs: 60 * 60 * 1000,
      searchFrom: new Date('2026-08-10T09:00:00Z'),
      searchUntil: new Date('2026-08-10T18:00:00Z'),
    })
    expect(slots).toEqual([])
  })
})
