import { describe, it, expect, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import CompetitionEntryPanel from '../CompetitionEntryPanel.vue'
import { mapToCompetitionListing, type CompetitionSummary } from '../../../models/competition'
import type { CompetitionsClient } from '../../../api/competitionsClient'

const RAW = {
  id: 'c1',
  hostId: 'host-1',
  name: 'Autumn Doubles Ladder',
  venueFacilityId: 'facility-1',
  sessions: [{ startsAt: '2026-09-01T09:00:00Z', endsAt: '2026-09-01T12:00:00Z', courtIds: ['court-1'] }],
  capacity: 16,
  guestAllowance: 2,
  paymentMethod: 'PAYMENT_METHOD_EITHER',
  entryFee: { amountCents: '2500', currencyCode: 'USD' },
  format: 'COMPETITION_FORMAT_DOUBLES',
  status: 'COMPETITION_STATUS_SCHEDULED',
}

function competition(overrides: Record<string, unknown> = {}, spotsLeft: number | null = 3): CompetitionSummary {
  return mapToCompetitionListing({ competition: { ...RAW, ...overrides }, spotsLeft } as never)
}

function fakeClient(post: (body: unknown) => unknown): CompetitionsClient {
  const POST = vi.fn(async (path: string, options: { body: unknown }) => {
    if (path === '/v1/competitions/{competitionId}/entries') return post(options.body)
    throw new Error(`unexpected POST ${path}`)
  })
  return { POST, GET: vi.fn() } as unknown as CompetitionsClient
}

function okEntry(guestCount = 0) {
  return {
    data: {
      entry: {
        id: 'e1',
        competitionId: 'c1',
        playerId: 'player-mock-1',
        guestCount,
        source: 'ENTRY_SOURCE_APP',
        paymentStatus: 'PAYMENT_STATUS_UNPAID',
        status: 'ENTRY_STATUS_ENTERED',
      },
    },
    error: undefined,
    response: { status: 200 },
  }
}

function mountPanel(overrides: Record<string, unknown> = {}) {
  return mount(CompetitionEntryPanel, {
    props: {
      competition: competition(),
      source: 'ENTRY_SOURCE_APP' as const,
      client: fakeClient(() => okEntry()),
      ...overrides,
    },
  })
}

describe('CompetitionEntryPanel — guest stepper', () => {
  it('is bounded by the competition\'s GuestAllowance', async () => {
    const wrapper = mountPanel()
    const add = wrapper.get('[aria-label="Add a guest"]')
    await add.trigger('click')
    await add.trigger('click')
    expect(wrapper.get('.competition-entry__stepper-count').text()).toBe('2')

    await add.trigger('click')
    expect(wrapper.get('.competition-entry__stepper-count').text()).toBe('2')
    expect(add.attributes('disabled')).toBeDefined()
  })

  it('disables the add control outright when no guests are permitted', () => {
    const wrapper = mountPanel({ competition: competition({ guestAllowance: 0 }) })
    expect(wrapper.get('[aria-label="Add a guest"]').attributes('disabled')).toBeDefined()
    expect(wrapper.text()).toContain('Up to 0 guests allowed')
  })

  it('never goes below zero', () => {
    const wrapper = mountPanel()
    expect(wrapper.get('[aria-label="Remove a guest"]').attributes('disabled')).toBeDefined()
  })
})

describe('CompetitionEntryPanel — entering', () => {
  it('sends the current guest count and shows a confirmation', async () => {
    const client = fakeClient(() => okEntry(1))
    const wrapper = mountPanel({ client })

    await wrapper.get('[aria-label="Add a guest"]').trigger('click')
    await wrapper.get('.competition-entry__form').trigger('submit')
    await flushPromises()

    expect(client.POST).toHaveBeenCalledWith('/v1/competitions/{competitionId}/entries', {
      params: { path: { competitionId: 'c1' } },
      body: { playerId: 'player-mock-1', guestCount: 1, source: 'ENTRY_SOURCE_APP' },
    })
    expect(wrapper.get('[role="status"]').text()).toContain("You're in")
  })

  it('shows the real entry fee before entering, and "Free" as a word', () => {
    expect(mountPanel().get('[data-testid="entry-fee"]').text()).toContain('$25.00')

    const free = mountPanel({ competition: competition({ entryFee: { amountCents: '0', currencyCode: 'USD' } }) })
    const label = free.get('[data-testid="entry-fee"]').text()
    expect(label).toContain('Free')
    expect(label).not.toContain('$0.00')
  })
})

describe('CompetitionEntryPanel — the competition-full rejection', () => {
  it('surfaces a specific, actionable message (WCAG 3.3.3) on a capacity 409', async () => {
    const wrapper = mountPanel({
      client: fakeClient(() => ({
        data: undefined,
        error: { message: 'competitions: competition is at capacity' },
        response: { status: 409 },
      })),
    })
    await wrapper.get('.competition-entry__form').trigger('submit')
    await flushPromises()

    expect(wrapper.get('[role="alert"]').text()).toContain('This competition just filled up')
  })

  // T9 has no Competitions waitlist — Social Play's is Game-scoped
  // (T6.6 / ADR-0006). The absence is deliberate, not a missed reuse.
  it('offers NO waitlist on the full path — it suggests browsing instead', async () => {
    const wrapper = mountPanel({
      client: fakeClient(() => ({
        data: undefined,
        error: { message: 'competitions: competition is at capacity' },
        response: { status: 409 },
      })),
    })
    await wrapper.get('.competition-entry__form').trigger('submit')
    await flushPromises()

    expect(wrapper.text().toLowerCase()).not.toContain('waitlist')
  })

  it('a pre-known full competition (spotsLeft 0) says so instead of offering a form that must fail', () => {
    const wrapper = mountPanel({ competition: competition({}, 0) })
    expect(wrapper.find('.competition-entry__form').exists()).toBe(false)
    expect(wrapper.text()).toContain('full')
  })

  it('a duplicate-entry 409 does NOT claim the competition filled up', async () => {
    const wrapper = mountPanel({
      client: fakeClient(() => ({
        data: undefined,
        error: { message: 'competitions: player has already entered this competition' },
        response: { status: 409 },
      })),
    })
    await wrapper.get('.competition-entry__form').trigger('submit')
    await flushPromises()

    const alert = wrapper.get('[role="alert"]').text()
    expect(alert).toContain('already entered')
    expect(alert).not.toContain('filled up')
  })
})

describe('CompetitionEntryPanel — payment (T8.10 paths, unchanged)', () => {
  it('a cash competition shows the pending-cash text with the real amount, and makes no payment call', async () => {
    const client = fakeClient(() => okEntry())
    const wrapper = mountPanel({ competition: competition({ paymentMethod: 'PAYMENT_METHOD_CASH' }), client })
    await wrapper.get('.competition-entry__form').trigger('submit')
    await flushPromises()

    expect(wrapper.text()).toContain('pending (cash at facility)')
    expect(wrapper.text()).toContain('$25.00')
    expect((client.POST as ReturnType<typeof vi.fn>).mock.calls).toHaveLength(1)
  })

  it('an "either" competition lets the entrant choose cash, with no extra network call', async () => {
    const client = fakeClient(() => okEntry())
    const wrapper = mountPanel({ client })
    await wrapper.get('.competition-entry__form').trigger('submit')
    await flushPromises()

    const before = (client.POST as ReturnType<typeof vi.fn>).mock.calls.length
    await wrapper.get('.competition-entry__pay-cash').trigger('click')

    expect(wrapper.text()).toContain('pending (cash at facility)')
    expect((client.POST as ReturnType<typeof vi.fn>).mock.calls.length).toBe(before)
  })

  it('a free competition offers no payment step at all', async () => {
    const wrapper = mountPanel({
      competition: competition({ entryFee: { amountCents: '0', currencyCode: 'USD' } }),
    })
    await wrapper.get('.competition-entry__form').trigger('submit')
    await flushPromises()

    expect(wrapper.get('[data-testid="entry-free-notice"]').text()).toContain('free')
    expect(wrapper.find('.competition-entry__pay-cash').exists()).toBe(false)
  })

  // T10.6 (closes #96): Competitions is now wired to the Payments context
  // via PAYABLE_TYPE_COMPETITION_ENTRY + internal/payments/adapter/
  // competitions, so the former "not available yet" disclosure is replaced
  // by a real "Pay online now" button — mirrors GameJoinPanel.vue's
  // identical T8.10 tests exactly.
  it('an online-only Competition shows a "Pay online now" button and no cash option', async () => {
    const wrapper = mountPanel({ competition: competition({ paymentMethod: 'PAYMENT_METHOD_ONLINE' }) })
    await wrapper.get('.competition-entry__form').trigger('submit')
    await flushPromises()

    const buttons = wrapper.findAll('.competition-entry__payment-choice button').map((b) => b.text())
    expect(buttons).toEqual(['Pay online now'])
  })

  it('clicking "Pay online now" emits payOnline with the confirmed entry id, and makes no Payments network call itself', async () => {
    const wrapper = mountPanel({ competition: competition({ paymentMethod: 'PAYMENT_METHOD_ONLINE' }) })
    await wrapper.get('.competition-entry__form').trigger('submit')
    await flushPromises()

    await wrapper.get('[data-testid="entry-pay-online"]').trigger('click')

    expect(wrapper.emitted('payOnline')).toEqual([['e1']])
  })

  it('an "either" Competition offers both options; choosing cash switches to the pending text without any network call', async () => {
    const client = fakeClient(() => okEntry())
    const wrapper = mountPanel({ client })
    await wrapper.get('.competition-entry__form').trigger('submit')
    await flushPromises()

    const buttons = wrapper.findAll('.competition-entry__payment-choice button').map((b) => b.text())
    expect(buttons).toEqual(['Pay online now', 'Pay cash at facility'])

    const callsBeforeCashChoice = (client.POST as ReturnType<typeof vi.fn>).mock.calls.length
    await wrapper.find('.competition-entry__pay-cash').trigger('click')

    expect(wrapper.text()).toContain('pending (cash at facility)')
    expect((client.POST as ReturnType<typeof vi.fn>).mock.calls.length).toBe(callsBeforeCashChoice)
    expect(wrapper.emitted('payOnline')).toBeUndefined()
  })
})
