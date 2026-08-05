import { describe, it, expect } from 'vitest'
import {
  mapToCompetitionListing,
  mapToCompetition,
  mapToCompetitionEntry,
  competitionFormatLabel,
  formatSessionRange,
  earliestSessionStart,
  isCancelled,
  type RawCompetition,
} from '../competition'

const RAW_COMPETITION = {
  id: 'c1',
  hostId: 'host-1',
  name: 'Autumn Doubles Ladder',
  venueFacilityId: 'facility-1',
  sessions: [
    { startsAt: '2026-09-05T09:00:00Z', endsAt: '2026-09-05T12:00:00Z', courtIds: ['court-1', 'court-2'] },
    { startsAt: '2026-09-01T09:00:00Z', endsAt: '2026-09-01T12:00:00Z', courtIds: ['court-1'] },
  ],
  capacity: 16,
  guestAllowance: 2,
  paymentMethod: 'PAYMENT_METHOD_EITHER',
  entryFee: { amountCents: '2500', currencyCode: 'USD' },
  format: 'COMPETITION_FORMAT_DOUBLES',
  status: 'COMPETITION_STATUS_SCHEDULED',
} as unknown as RawCompetition

describe('mapToCompetitionListing', () => {
  it('carries spots_left from the listing wrapper', () => {
    const summary = mapToCompetitionListing({ competition: RAW_COMPETITION, spotsLeft: 6 } as never)
    expect(summary.id).toBe('c1')
    expect(summary.name).toBe('Autumn Doubles Ladder')
    expect(summary.spotsLeft).toBe(6)
    expect(summary.sessions).toHaveLength(2)
    expect(summary.sessions[0]?.courtIds).toEqual(['court-1', 'court-2'])
  })

  it('reads the int64 entry fee, which protojson sends as a string', () => {
    const summary = mapToCompetitionListing({ competition: RAW_COMPETITION, spotsLeft: 0 } as never)
    expect(summary.entryFeeCents).toBe(2500)
    expect(summary.entryFeeCurrency).toBe('USD')
  })

  it('treats a missing entry fee as free (0), matching the wire default', () => {
    const summary = mapToCompetitionListing({
      competition: { ...RAW_COMPETITION, entryFee: undefined },
      spotsLeft: 1,
    } as never)
    expect(summary.entryFeeCents).toBe(0)
  })
})

describe('mapToCompetition', () => {
  // GetCompetition and GetCompetitionByShareToken both return a bare
  // Competition with NO spots_left field — see the proto. `null` is how the
  // UI knows it has no real number to show, rather than inventing one.
  it('leaves spotsLeft null, because the endpoint does not return it', () => {
    const summary = mapToCompetition(RAW_COMPETITION)
    expect(summary.spotsLeft).toBeNull()
    expect(summary.id).toBe('c1')
  })
})

describe('mapToCompetitionEntry', () => {
  it('maps a confirmed entry', () => {
    const entry = mapToCompetitionEntry({
      id: 'e1',
      competitionId: 'c1',
      playerId: 'player-mock-1',
      guestCount: 2,
      source: 'ENTRY_SOURCE_SOCIAL',
      paymentStatus: 'PAYMENT_STATUS_UNPAID',
      status: 'ENTRY_STATUS_ENTERED',
    } as never)
    expect(entry).toEqual({
      id: 'e1',
      competitionId: 'c1',
      playerId: 'player-mock-1',
      guestCount: 2,
      source: 'ENTRY_SOURCE_SOCIAL',
      paymentStatus: 'PAYMENT_STATUS_UNPAID',
      status: 'ENTRY_STATUS_ENTERED',
    })
  })
})

describe('competitionFormatLabel', () => {
  // WCAG 1.4.1: every branch returns text.
  it.each([
    ['COMPETITION_FORMAT_SINGLES', 'Singles'],
    ['COMPETITION_FORMAT_DOUBLES', 'Doubles'],
    ['COMPETITION_FORMAT_UNSPECIFIED', 'Format not set'],
    ['SOMETHING_NEWER', 'Format not set'],
  ])('%s -> %s', (input, expected) => {
    expect(competitionFormatLabel(input)).toBe(expected)
  })
})

describe('earliestSessionStart', () => {
  // The backend orders and filters on the EARLIEST session start (see
  // ListCompetitionsRequest's doc comment); the list row must agree.
  it('returns the earliest start, not the first in array order', () => {
    const summary = mapToCompetitionListing({ competition: RAW_COMPETITION, spotsLeft: 1 } as never)
    expect(earliestSessionStart(summary.sessions)).toBe('2026-09-01T09:00:00Z')
  })

  it('returns an empty string when there are no sessions', () => {
    expect(earliestSessionStart([])).toBe('')
  })
})

describe('formatSessionRange', () => {
  it('renders a date and a time range', () => {
    const text = formatSessionRange({
      startsAt: '2026-09-01T09:00:00Z',
      endsAt: '2026-09-01T12:00:00Z',
      courtIds: [],
    })
    expect(text).toMatch(/2026/)
    expect(text).toContain('–')
  })

  it('returns a stated placeholder rather than "Invalid Date" for a missing range', () => {
    expect(formatSessionRange({ startsAt: '', endsAt: '', courtIds: [] })).toBe('Time not set')
  })
})

describe('isCancelled', () => {
  it('is true only for the cancelled status', () => {
    expect(isCancelled('COMPETITION_STATUS_CANCELLED')).toBe(true)
    expect(isCancelled('COMPETITION_STATUS_SCHEDULED')).toBe(false)
  })
})
