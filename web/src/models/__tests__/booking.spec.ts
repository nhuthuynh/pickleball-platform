import { describe, it, expect } from 'vitest'
import {
  mapToQuote,
  mapToBooking,
  formatPriceCents,
  computeNextAvailableSlots,
  type ConfirmedBooking,
} from '../booking'

describe('mapToQuote', () => {
  it('parses the int64-as-string priceCents into a number', () => {
    const quote = mapToQuote({ priceCents: '1800', band: 'peak' }, 1000)
    expect(quote).toEqual({ priceCents: 1800, band: 'peak', fetchedAt: 1000 })
  })

  it('defaults missing fields', () => {
    const quote = mapToQuote({}, 500)
    expect(quote).toEqual({ priceCents: 0, band: '', fetchedAt: 500 })
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
