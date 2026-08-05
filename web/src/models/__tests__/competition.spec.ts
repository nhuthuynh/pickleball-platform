import { describe, it, expect } from 'vitest'
import {
  competitionFormatLabel,
  entrySourceLabel,
  entryPaymentStatusLabel,
  entryStatusLabel,
  competitionDatesLabel,
  findOverlappingSessionIndices,
  draftSessionToWire,
  shareUrlForToken,
  mapToCompetitionSummary,
  mapToCompetitionEntry,
  type DraftSession,
} from '../competition'

/** A DraftSession with sensible defaults, overridable per case. */
function draft(overrides: Partial<DraftSession> = {}): DraftSession {
  return {
    date: '2026-09-01',
    startTime: '10:00',
    endTime: '12:00',
    courtIds: ['court-1'],
    ...overrides,
  }
}

describe('competitionFormatLabel (WCAG 1.4.1 — every branch returns text)', () => {
  const cases: { name: string; format: string; want: string }[] = [
    { name: 'singles', format: 'COMPETITION_FORMAT_SINGLES', want: 'Singles' },
    { name: 'doubles', format: 'COMPETITION_FORMAT_DOUBLES', want: 'Doubles' },
    { name: 'unspecified', format: 'COMPETITION_FORMAT_UNSPECIFIED', want: 'Format not set' },
    { name: 'unrecognized value', format: 'COMPETITION_FORMAT_TEAMS', want: 'Format not set' },
  ]

  for (const c of cases) {
    it(`renders ${c.name} as text`, () => {
      expect(competitionFormatLabel(c.format)).toBe(c.want)
    })
  }
})

describe('entrySourceLabel', () => {
  const cases: { name: string; source: string; want: string }[] = [
    { name: 'in-app entry', source: 'ENTRY_SOURCE_APP', want: 'In app' },
    { name: 'shared-link entry', source: 'ENTRY_SOURCE_SOCIAL', want: 'Via shared link' },
    { name: 'unspecified', source: 'ENTRY_SOURCE_UNSPECIFIED', want: 'Entry source not set' },
    { name: 'unrecognized value', source: 'ENTRY_SOURCE_CARRIER_PIGEON', want: 'Entry source not set' },
  ]

  for (const c of cases) {
    it(`renders ${c.name} as text`, () => {
      expect(entrySourceLabel(c.source)).toBe(c.want)
    })
  }

  // docs/design/v1-external-reference-reconciliation.md's explicit
  // instruction: the design attachment's wireframe labels this "via
  // WhatsApp reply" / "via Facebook reply", depicting reply-scraping that
  // is NOT built and (per ADR-0009) is not going to be. The backend's
  // EntrySource enum has exactly two values and neither names a platform.
  it('never names a third-party platform — the wire has no such fact to carry', () => {
    const rendered = [
      'ENTRY_SOURCE_APP',
      'ENTRY_SOURCE_SOCIAL',
      'ENTRY_SOURCE_UNSPECIFIED',
    ].map(entrySourceLabel)

    for (const label of rendered) {
      const lower = label.toLowerCase()
      expect(lower).not.toContain('whatsapp')
      expect(lower).not.toContain('facebook')
      expect(lower).not.toContain('instagram')
      expect(lower).not.toContain('reply')
    }
  })
})

describe('entryPaymentStatusLabel (WCAG 1.4.1 — status is text, never colour alone)', () => {
  const cases: { status: string; want: string }[] = [
    { status: 'PAYMENT_STATUS_UNPAID', want: 'Unpaid' },
    { status: 'PAYMENT_STATUS_PAID', want: 'Paid' },
    // "never paid" and "paid, then refunded" are different facts the
    // domain deliberately keeps apart — the label must too.
    { status: 'PAYMENT_STATUS_REFUNDED', want: 'Refunded' },
    { status: 'PAYMENT_STATUS_UNSPECIFIED', want: 'Payment status not set' },
    { status: 'nonsense', want: 'Payment status not set' },
  ]

  for (const c of cases) {
    it(`renders ${c.status} as "${c.want}"`, () => {
      expect(entryPaymentStatusLabel(c.status)).toBe(c.want)
    })
  }
})

describe('entryStatusLabel', () => {
  it('tells a withdrawn entry apart from an active one', () => {
    expect(entryStatusLabel('ENTRY_STATUS_ENTERED')).toBe('Entered')
    expect(entryStatusLabel('ENTRY_STATUS_CANCELLED')).toBe('Withdrawn')
    expect(entryStatusLabel('ENTRY_STATUS_UNSPECIFIED')).toBe('Entry status not set')
  })
})

describe('competitionDatesLabel', () => {
  // Formatted with the same Intl call the label itself uses, so these
  // assertions hold in any CI timezone/locale rather than pinning the
  // runner's.
  const mediumDate = (iso: string) =>
    new Intl.DateTimeFormat(undefined, { dateStyle: 'medium' }).format(new Date(iso))

  it('spans the earliest session start to the latest session end', () => {
    const label = competitionDatesLabel([
      { startsAt: '2026-09-05T09:00:00Z', endsAt: '2026-09-05T12:00:00Z', courtIds: ['c1'] },
      { startsAt: '2026-09-01T09:00:00Z', endsAt: '2026-09-01T12:00:00Z', courtIds: ['c1'] },
    ])

    // Order-independent: the label reflects the range, not array order.
    expect(label).toContain(mediumDate('2026-09-01T09:00:00Z'))
    expect(label).toContain(mediumDate('2026-09-05T12:00:00Z'))
  })

  it('renders a single-day competition as one date, not a range of one day to itself', () => {
    const label = competitionDatesLabel([
      { startsAt: '2026-09-01T09:00:00Z', endsAt: '2026-09-01T12:00:00Z', courtIds: ['c1'] },
    ])
    expect(label).toBe(mediumDate('2026-09-01T09:00:00Z'))
  })

  it('returns an empty string for no sessions rather than a fabricated date', () => {
    expect(competitionDatesLabel([])).toBe('')
  })
})

// Client-side mirror of internal/competitions/domain's
// ensureNoSessionOverlap (T9.1). The server check stays authoritative —
// these cases exist so the same rule is pinned down on the client too.
describe('findOverlappingSessionIndices — mirrors domain.ensureNoSessionOverlap', () => {
  it('flags two sessions sharing a court at overlapping times', () => {
    const rows = [
      draft({ startTime: '10:00', endTime: '12:00', courtIds: ['court-1'] }),
      draft({ startTime: '11:00', endTime: '13:00', courtIds: ['court-1'] }),
    ]
    expect([...findOverlappingSessionIndices(rows)].sort()).toEqual([0, 1])
  })

  it('does NOT flag back-to-back sessions on one court — ranges are half-open [start, end)', () => {
    const rows = [
      draft({ startTime: '10:00', endTime: '12:00', courtIds: ['court-1'] }),
      draft({ startTime: '12:00', endTime: '14:00', courtIds: ['court-1'] }),
    ]
    expect(findOverlappingSessionIndices(rows).size).toBe(0)
  })

  it('does NOT flag overlapping times on DIFFERENT courts', () => {
    const rows = [
      draft({ startTime: '10:00', endTime: '12:00', courtIds: ['court-1'] }),
      draft({ startTime: '10:00', endTime: '12:00', courtIds: ['court-2'] }),
    ]
    expect(findOverlappingSessionIndices(rows).size).toBe(0)
  })

  it('does NOT flag the same court and times on DIFFERENT dates', () => {
    const rows = [
      draft({ date: '2026-09-01', courtIds: ['court-1'] }),
      draft({ date: '2026-09-02', courtIds: ['court-1'] }),
    ]
    expect(findOverlappingSessionIndices(rows).size).toBe(0)
  })

  it('flags a partial court overlap between multi-court sessions', () => {
    const rows = [
      draft({ startTime: '10:00', endTime: '12:00', courtIds: ['court-1', 'court-2'] }),
      draft({ startTime: '11:00', endTime: '13:00', courtIds: ['court-2', 'court-3'] }),
    ]
    expect([...findOverlappingSessionIndices(rows)].sort()).toEqual([0, 1])
  })

  it('flags only the offending pair, leaving an unrelated third row unflagged', () => {
    const rows = [
      draft({ startTime: '10:00', endTime: '12:00', courtIds: ['court-1'] }),
      draft({ startTime: '14:00', endTime: '15:00', courtIds: ['court-9'] }),
      draft({ startTime: '11:00', endTime: '13:00', courtIds: ['court-1'] }),
    ]
    expect([...findOverlappingSessionIndices(rows)].sort()).toEqual([0, 2])
  })

  // The domain's own walk catches this ("a range always overlaps itself"),
  // so the client mirror must too.
  it('flags a court listed twice within a single session row', () => {
    const rows = [draft({ courtIds: ['court-1', 'court-1'] })]
    expect([...findOverlappingSessionIndices(rows)]).toEqual([0])
  })

  it('ignores incomplete rows rather than guessing at a range', () => {
    const rows = [draft({ startTime: '', endTime: '' }), draft()]
    expect(findOverlappingSessionIndices(rows).size).toBe(0)
  })
})

describe('draftSessionToWire', () => {
  it('combines date + time into a single ISO instant per bound', () => {
    const wire = draftSessionToWire(
      draft({ date: '2026-09-01', startTime: '10:00', endTime: '12:30', courtIds: ['court-1'] }),
    )!

    expect(wire.courtIds).toEqual(['court-1'])
    expect(new Date(wire.startsAt).toISOString()).toBe(wire.startsAt)
    expect(new Date(wire.endsAt).getTime() - new Date(wire.startsAt).getTime()).toBe(
      2.5 * 60 * 60 * 1000,
    )
  })

  it('returns null for an incomplete row rather than an Invalid Date on the wire', () => {
    expect(draftSessionToWire(draft({ date: '' }))).toBeNull()
    expect(draftSessionToWire(draft({ startTime: '' }))).toBeNull()
  })
})

describe('shareUrlForToken', () => {
  // T9.7 owns the landing route `/c/:shareToken`; this is the URL a Host
  // actually posts, so it must point there and carry the real token.
  it('builds the T9.7 deep-link URL from the real token', () => {
    expect(shareUrlForToken('tok-abc123', 'https://play.example')).toBe(
      'https://play.example/c/tok-abc123',
    )
  })

  it('returns an empty string with no token rather than a link to nowhere', () => {
    expect(shareUrlForToken('', 'https://play.example')).toBe('')
  })
})

describe('mapToCompetitionSummary', () => {
  it('copies only declared fields, defaulting an absent entry fee to 0 (free)', () => {
    const summary = mapToCompetitionSummary({
      id: 'comp-1',
      hostId: 'host-1',
      name: 'Autumn Open',
      venueFacilityId: 'fac-1',
      sessions: [
        { startsAt: '2026-09-01T09:00:00Z', endsAt: '2026-09-01T12:00:00Z', courtIds: ['court-1'] },
      ],
      capacity: 16,
      guestAllowance: 1,
      paymentMethod: 'PAYMENT_METHOD_CASH',
      format: 'COMPETITION_FORMAT_DOUBLES',
      status: 'COMPETITION_STATUS_SCHEDULED',
    })

    expect(summary.name).toBe('Autumn Open')
    expect(summary.capacity).toBe(16)
    expect(summary.entryFeeCents).toBe(0)
    expect(summary.sessions).toHaveLength(1)
    expect(summary.sessions[0]!.courtIds).toEqual(['court-1'])
  })

  it('reads an int64 amount_cents delivered as a protojson string', () => {
    const summary = mapToCompetitionSummary({
      id: 'comp-1',
      entryFee: { amountCents: '2500', currencyCode: 'USD' },
    })
    expect(summary.entryFeeCents).toBe(2500)
    expect(summary.entryFeeCurrency).toBe('USD')
  })
})

describe('mapToCompetitionEntry', () => {
  it('carries guest count, source, payment status, and entry status', () => {
    const entry = mapToCompetitionEntry({
      id: 'entry-1',
      competitionId: 'comp-1',
      playerId: 'player-7',
      guestCount: 2,
      source: 'ENTRY_SOURCE_SOCIAL',
      paymentStatus: 'PAYMENT_STATUS_UNPAID',
      status: 'ENTRY_STATUS_ENTERED',
    })

    expect(entry).toEqual({
      id: 'entry-1',
      competitionId: 'comp-1',
      playerId: 'player-7',
      guestCount: 2,
      source: 'ENTRY_SOURCE_SOCIAL',
      paymentStatus: 'PAYMENT_STATUS_UNPAID',
      status: 'ENTRY_STATUS_ENTERED',
    })
  })
})
