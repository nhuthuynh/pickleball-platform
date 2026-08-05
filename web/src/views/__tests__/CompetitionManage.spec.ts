import { describe, it, expect, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import CompetitionManage from '../CompetitionManage.vue'
import type { CompetitionsClient } from '../../api/competitionsClient'

function matchMediaForWidth(width: number): Pick<Window, 'matchMedia'> {
  return {
    matchMedia: (query: string) => {
      const min = Number(query.match(/min-width:\s*(\d+)px/)?.[1] ?? -Infinity)
      const max = Number(query.match(/max-width:\s*(\d+)px/)?.[1] ?? Infinity)
      return {
        matches: width >= min && width <= max,
        media: query,
        addEventListener: () => {},
        removeEventListener: () => {},
      } as unknown as MediaQueryList
    },
  }
}

const COMPETITION = {
  id: 'comp-1',
  hostId: 'host-mock-1',
  name: 'Autumn Open',
  venueFacilityId: 'fac-1',
  sessions: [
    { startsAt: '2026-09-01T10:00:00Z', endsAt: '2026-09-01T12:00:00Z', courtIds: ['court-1'] },
    { startsAt: '2026-09-02T10:00:00Z', endsAt: '2026-09-02T12:00:00Z', courtIds: ['court-1'] },
  ],
  capacity: 16,
  guestAllowance: 2,
  paymentMethod: 'PAYMENT_METHOD_CASH',
  entryFee: { amountCents: '2500', currencyCode: 'USD' },
  format: 'COMPETITION_FORMAT_DOUBLES',
  status: 'COMPETITION_STATUS_SCHEDULED',
}

const ENTRIES = [
  {
    id: 'entry-1',
    competitionId: 'comp-1',
    playerId: 'player-1',
    guestCount: 2,
    source: 'ENTRY_SOURCE_APP',
    paymentStatus: 'PAYMENT_STATUS_PAID',
    status: 'ENTRY_STATUS_ENTERED',
  },
  {
    id: 'entry-2',
    competitionId: 'comp-1',
    playerId: 'player-2',
    guestCount: 0,
    source: 'ENTRY_SOURCE_SOCIAL',
    paymentStatus: 'PAYMENT_STATUS_UNPAID',
    status: 'ENTRY_STATUS_ENTERED',
  },
]

function makeClient(
  overrides: Partial<{
    competition: () => { data?: unknown; error?: unknown }
    entries: () => { data?: unknown; error?: unknown }
  }> = {},
) {
  const competition = overrides.competition ?? (() => ({ data: { competition: COMPETITION } }))
  const entries = overrides.entries ?? (() => ({ data: { entries: ENTRIES } }))

  const GET = vi.fn(async (path: string) => {
    if (path === '/v1/competitions/{competitionId}') return competition()
    if (path === '/v1/competitions/{competitionId}/entries') return entries()
    throw new Error(`unexpected path in test fake: ${path}`)
  })

  return { GET, POST: vi.fn() } as unknown as CompetitionsClient
}

function mountManage(client: CompetitionsClient = makeClient()) {
  return mount(CompetitionManage, {
    props: { id: 'comp-1', client, win: matchMediaForWidth(1440) },
    global: { stubs: { RouterLink: { template: '<a><slot /></a>' } } },
  })
}

describe('CompetitionManage — roster', () => {
  it('lists one row per entry with guest count, payment status, and source, all as text', async () => {
    const wrapper = mountManage()
    await flushPromises()

    const rows = wrapper.findAll('[data-testid="roster-row"]')
    expect(rows).toHaveLength(2)

    // WCAG 1.4.1: payment status and entry source are words in the row, not
    // a colour-coded dot or a tinted background.
    expect(rows[0]!.text()).toContain('player-1')
    expect(rows[0]!.text()).toContain('2 guests')
    expect(rows[0]!.text()).toContain('Paid')
    expect(rows[0]!.text()).toContain('In app')

    expect(rows[1]!.text()).toContain('player-2')
    expect(rows[1]!.text()).toContain('Unpaid')
    expect(rows[1]!.text()).toContain('Via shared link')
  })

  it('renders no guest phrase at all for an entry that brought none', async () => {
    const wrapper = mountManage()
    await flushPromises()

    const rows = wrapper.findAll('[data-testid="roster-row"]')
    expect(rows[1]!.text()).not.toContain('guest')
  })

  // docs/design/v1-external-reference-reconciliation.md's explicit
  // instruction — the design attachment's wireframe says "via WhatsApp
  // reply"; nothing on the wire records a platform, so nothing here may
  // claim one. An absence assertion, because this is precisely the label a
  // future well-meaning change would "restore" from the wireframe.
  it('NEVER labels an entry with a third-party platform', async () => {
    const wrapper = mountManage()
    await flushPromises()

    const text = wrapper.text().toLowerCase()
    expect(text).not.toContain('whatsapp')
    expect(text).not.toContain('facebook')
    expect(text).not.toContain('instagram')
    expect(text).not.toContain('telegram')
    expect(text).not.toContain('reply')
  })

  it('shows a designed empty state when nobody has entered yet', async () => {
    const wrapper = mountManage(makeClient({ entries: () => ({ data: { entries: [] } }) }))
    await flushPromises()

    expect(wrapper.findAll('[data-testid="roster-row"]')).toHaveLength(0)
    expect(wrapper.get('[data-testid="roster-empty"]').text()).toContain('No entries yet')
  })

  it('shows an error state, not a blank screen, when the roster read fails', async () => {
    const wrapper = mountManage(
      makeClient({ entries: () => ({ error: { code: 13, message: 'boom' } }) }),
    )
    await flushPromises()

    const error = wrapper.get('[data-testid="manage-error"]')
    expect(error.attributes('role')).toBe('alert')
    expect(error.text().length).toBeGreaterThan(0)
  })

  it('distinguishes a withdrawn entry from an active one rather than hiding it', async () => {
    const wrapper = mountManage(
      makeClient({
        entries: () => ({
          data: {
            entries: [
              { ...ENTRIES[0], id: 'entry-3', status: 'ENTRY_STATUS_CANCELLED' },
            ],
          },
        }),
      }),
    )
    await flushPromises()

    expect(wrapper.findAll('[data-testid="roster-row"]')).toHaveLength(1)
    expect(wrapper.get('[data-testid="roster-row"]').text()).toContain('Withdrawn')
  })
})

describe('CompetitionManage — unpaid cash entries (T8.10 pattern, reused)', () => {
  it('surfaces the amount due on an unpaid entry using the shared UnpaidCashAmount component', async () => {
    const wrapper = mountManage()
    await flushPromises()

    const amounts = wrapper.findAll('[data-testid="unpaid-cash-amount"]')
    // Only the unpaid one — the paid entry owes nothing.
    expect(amounts).toHaveLength(1)
    expect(amounts[0]!.text()).toContain('$25.00 due (cash at facility)')
  })

  it('does not surface a cash amount for an ONLINE-only competition', async () => {
    const wrapper = mountManage(
      makeClient({
        competition: () => ({
          data: { competition: { ...COMPETITION, paymentMethod: 'PAYMENT_METHOD_ONLINE' } },
        }),
      }),
    )
    await flushPromises()

    expect(wrapper.findAll('[data-testid="unpaid-cash-amount"]')).toHaveLength(0)
  })

  it('does not surface a cash amount for a FREE competition — there is nothing to collect', async () => {
    const wrapper = mountManage(
      makeClient({
        competition: () => ({
          data: {
            competition: { ...COMPETITION, entryFee: { amountCents: '0', currencyCode: 'USD' } },
          },
        }),
      }),
    )
    await flushPromises()

    expect(wrapper.findAll('[data-testid="unpaid-cash-amount"]')).toHaveLength(0)
  })

  // Payments has no PayableType for a Competition entry
  // (PAYABLE_TYPE_BOOKING / REGISTRATION / NO_SHOW_FEE only), so
  // RecordOfflinePayment cannot be called for one. A "Mark paid" button
  // here could only ever fail — so there isn't one, and the UI says why.
  it('offers NO "Mark paid" action, and says in text why not', async () => {
    const wrapper = mountManage()
    await flushPromises()

    expect(wrapper.find('[data-testid="mark-paid"]').exists()).toBe(false)
    expect(wrapper.text().toLowerCase()).not.toContain('mark paid')
    expect(wrapper.get('[data-testid="cash-recording-note"]').text().toLowerCase()).toContain(
      "isn't available yet",
    )
  })
})

describe('CompetitionManage — competition summary (real fields only)', () => {
  it('shows the name, format, dates, capacity, and entry fee from the API response', async () => {
    const wrapper = mountManage()
    await flushPromises()

    const summary = wrapper.get('[data-testid="competition-summary"]').text()
    expect(summary).toContain('Autumn Open')
    expect(summary).toContain('Doubles')
    expect(summary).toContain('$25.00')
    expect(summary).toContain('16')
  })

  it('lists every session, because a Competition spans dates', async () => {
    const wrapper = mountManage()
    await flushPromises()

    expect(wrapper.findAll('[data-testid="session-item"]')).toHaveLength(2)
  })

  it('renders a free competition as the word "Free", never "$0.00"', async () => {
    const wrapper = mountManage(
      makeClient({
        competition: () => ({
          data: {
            competition: { ...COMPETITION, entryFee: { amountCents: '0', currencyCode: 'USD' } },
          },
        }),
      }),
    )
    await flushPromises()

    const summary = wrapper.get('[data-testid="competition-summary"]').text()
    expect(summary).toContain('Free')
    expect(summary).not.toContain('$0.00')
  })

  it('says so when the competition has been cancelled, rather than showing it as live', async () => {
    const wrapper = mountManage(
      makeClient({
        competition: () => ({
          data: { competition: { ...COMPETITION, status: 'COMPETITION_STATUS_CANCELLED' } },
        }),
      }),
    )
    await flushPromises()

    expect(wrapper.get('[data-testid="cancelled-notice"]').text().toLowerCase()).toContain(
      'cancelled',
    )
  })
})

describe('CompetitionManage — honest omissions', () => {
  it('has NO matching control, and carries the same note T8.8 established', async () => {
    const wrapper = mountManage()
    await flushPromises()

    expect(wrapper.find('input[type="range"]').exists()).toBe(false)
    const text = wrapper.text().toLowerCase()
    expect(text).not.toContain('auto-match')
    expect(text).not.toContain('automatch')
    expect(text).not.toContain('gender')
    expect(text).not.toContain('skill level')
    expect(text).not.toContain('seeding')
    expect(text).not.toContain('bracket')

    expect(wrapper.get('[data-testid="matching-note"]').text()).toContain(
      "Automated matching isn't available yet",
    )
  })

  it('has NO "Connect account" affordance', async () => {
    const wrapper = mountManage()
    await flushPromises()

    const text = wrapper.text().toLowerCase()
    expect(text).not.toContain('connect account')
    expect(text).not.toContain('connected')
    expect(wrapper.find('[data-testid="connect-account"]').exists()).toBe(false)
  })

  // The share token is returned ONCE, on CreateCompetitionResponse, and is
  // deliberately absent from Competition itself so it can't leak through
  // the unauthenticated reads. There is no authenticated re-read and no
  // rotation path in T9 — so this screen cannot show the link, and says so
  // instead of rendering a fabricated one.
  it('does not fabricate a share link it has no way to obtain', async () => {
    const wrapper = mountManage()
    await flushPromises()

    expect(wrapper.find('[data-testid="promo-text"]').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('/c/')
    expect(wrapper.get('[data-testid="share-link-note"]').text().toLowerCase()).toContain(
      'registration link',
    )
  })
})
