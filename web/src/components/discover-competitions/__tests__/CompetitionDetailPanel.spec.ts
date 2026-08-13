import { describe, it, expect, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import CompetitionDetailPanel from '../CompetitionDetailPanel.vue'
import { mapToCompetitionListing, type CompetitionSummary } from '../../../models/competition'
import type { CompetitionsClient } from '../../../api/competitionsClient'
import type { IdentityClient } from '../../../api/identityClient'
import type { FacilitiesClient } from '../../../api/facilitiesClient'

const RAW = {
  id: 'c1',
  hostId: 'host-1',
  name: 'Autumn Doubles Ladder',
  venueFacilityId: 'facility-1',
  sessions: [
    { startsAt: '2026-09-01T09:00:00Z', endsAt: '2026-09-01T12:00:00Z', courtIds: ['court-1', 'court-2'] },
    { startsAt: '2026-09-08T09:00:00Z', endsAt: '2026-09-08T12:00:00Z', courtIds: ['court-3'] },
  ],
  capacity: 16,
  guestAllowance: 2,
  paymentMethod: 'PAYMENT_METHOD_CASH',
  entryFee: { amountCents: '2500', currencyCode: 'USD' },
  format: 'COMPETITION_FORMAT_DOUBLES',
  status: 'COMPETITION_STATUS_SCHEDULED',
}

const COMPETITION: CompetitionSummary = mapToCompetitionListing({ competition: RAW, spotsLeft: 3 } as never)

const client = { GET: vi.fn(), POST: vi.fn() } as unknown as CompetitionsClient

/** T10.8: resolves any user id to a fixed DisplayName, so tests unrelated
 * to the Host join don't need to special-case the fetch. */
function fakeIdentityClient(): IdentityClient {
  return {
    GET: vi.fn(async () => ({ data: { user: { displayName: 'Ada Lovelace' } } })),
    POST: vi.fn(),
  } as unknown as IdentityClient
}

/** T10.8: resolves any facility id to a fixed Name, same reasoning as
 * fakeIdentityClient above. */
function fakeFacilitiesClient(): FacilitiesClient {
  return {
    GET: vi.fn(async () => ({ data: { facility: { name: 'Riverside Courts' } } })),
    POST: vi.fn(),
  } as unknown as FacilitiesClient
}

function props(overrides: Record<string, unknown> = {}) {
  return {
    competition: COMPETITION,
    loading: false,
    error: null,
    hasSelection: true,
    source: 'ENTRY_SOURCE_APP' as const,
    client,
    identityClient: fakeIdentityClient(),
    facilitiesClient: fakeFacilitiesClient(),
    ...overrides,
  }
}

describe('CompetitionDetailPanel', () => {
  it('shows the resolved host and venue names (T10.8, closes #98), format, spots left, entry fee and payment method', async () => {
    const wrapper = mount(CompetitionDetailPanel, { props: props() })
    await flushPromises()
    const text = wrapper.text()
    expect(text).toContain('Autumn Doubles Ladder')
    expect(text).toContain('Ada Lovelace')
    expect(text).toContain('Riverside Courts')
    expect(text).not.toContain('host-1')
    expect(text).not.toContain('facility-1')
    expect(text).toContain('Doubles')
    expect(text).toContain('3 spots left')
    expect(wrapper.get('[data-testid="competition-detail-entry-fee"]').text()).toContain('$25.00')
    // WCAG 1.4.1: the payment method is words, never a colour swatch.
    expect(text).toContain('Cash at facility')
  })

  it('degrades honestly to a named empty state when the Host lookup fails — never a fabricated name', async () => {
    const failingIdentityClient = {
      GET: vi.fn(async () => ({ data: undefined, error: { code: 5, message: 'not found' } })),
      POST: vi.fn(),
    } as unknown as IdentityClient
    const wrapper = mount(CompetitionDetailPanel, {
      props: props({ identityClient: failingIdentityClient }),
    })
    await flushPromises()

    expect(wrapper.text()).toContain('Unknown host')
  })

  it('shows an honest "No venue set" for a Competition with no venue, without a facility lookup', async () => {
    const facilitiesClient = fakeFacilitiesClient()
    const wrapper = mount(CompetitionDetailPanel, {
      props: props({ competition: { ...COMPETITION, venueFacilityId: '' }, facilitiesClient }),
    })
    await flushPromises()

    expect(wrapper.text()).toContain('No venue set')
    expect(facilitiesClient.GET).not.toHaveBeenCalled()
  })

  it('lists EVERY session with its own date/time and courts', () => {
    const wrapper = mount(CompetitionDetailPanel, { props: props() })
    const sessions = wrapper.findAll('.competition-detail__session')
    expect(sessions).toHaveLength(2)
    expect(sessions[0]?.text()).toContain('court-1, court-2')
    expect(sessions[1]?.text()).toContain('court-3')
  })

  it('renders "Free" — the word — for a zero entry fee', () => {
    const free = mapToCompetitionListing({
      competition: { ...RAW, entryFee: { amountCents: '0', currencyCode: 'USD' } },
      spotsLeft: 3,
    } as never)
    const wrapper = mount(CompetitionDetailPanel, { props: props({ competition: free }) })
    const fee = wrapper.get('[data-testid="competition-detail-entry-fee"]').text()
    expect(fee).toContain('Free')
    expect(fee).not.toContain('$0.00')
  })

  it('omits the spots-left row entirely when the API did not supply one, rather than inventing a number', () => {
    const noSpots = { ...COMPETITION, spotsLeft: null }
    const wrapper = mount(CompetitionDetailPanel, { props: props({ competition: noSpots }) })
    expect(wrapper.find('[data-testid="competition-detail-spots"]').exists()).toBe(false)
    // …and the rest of the detail still renders.
    expect(wrapper.text()).toContain('Autumn Doubles Ladder')
  })

  it('shows a placeholder before anything is selected', () => {
    const wrapper = mount(CompetitionDetailPanel, { props: props({ hasSelection: false, competition: null }) })
    expect(wrapper.text()).toContain('Select a competition')
  })

  it('shows a loading state and an error state with a retry action', async () => {
    const loading = mount(CompetitionDetailPanel, { props: props({ loading: true, competition: null }) })
    expect(loading.get('[role="status"]').text()).toContain('Loading competition details')

    const errored = mount(CompetitionDetailPanel, { props: props({ error: 'Boom', competition: null }) })
    expect(errored.get('[role="alert"]').text()).toContain('Boom')
    await errored.get('.competition-detail__retry').trigger('click')
    expect(errored.emitted('retry')).toHaveLength(1)
  })

  it('offers no entry form for a cancelled competition, and says why', () => {
    const cancelled = mapToCompetitionListing({
      competition: { ...RAW, status: 'COMPETITION_STATUS_CANCELLED' },
      spotsLeft: 3,
    } as never)
    const wrapper = mount(CompetitionDetailPanel, { props: props({ competition: cancelled }) })
    expect(wrapper.find('.competition-entry__form').exists()).toBe(false)
    expect(wrapper.text()).toContain('cancelled')
  })

  // There is deliberately NO waitlist for Competitions in T9 — Social Play's
  // waitlist is Game-scoped (T6.6 / ADR-0006), a different bounded context.
  it('never offers a waitlist', () => {
    const wrapper = mount(CompetitionDetailPanel, { props: props() })
    expect(wrapper.text().toLowerCase()).not.toContain('waitlist')
  })
})
