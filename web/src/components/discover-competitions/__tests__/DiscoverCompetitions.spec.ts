import { describe, it, expect, vi } from 'vitest'
import { nextTick } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import DiscoverCompetitions from '../DiscoverCompetitions.vue'
import type { CompetitionsClient } from '../../../api/competitionsClient'
import type { IdentityClient } from '../../../api/identityClient'
import type { FacilitiesClient } from '../../../api/facilitiesClient'

/** Mirrors DiscoverGames.spec.ts's identical fixed-viewport matchMedia stand-in. */
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

const LISTING = {
  competition: {
    id: 'c1',
    hostId: 'host-1',
    name: 'Autumn Doubles Ladder',
    venueFacilityId: 'facility-1',
    sessions: [{ startsAt: '2026-09-01T09:00:00Z', endsAt: '2026-09-01T12:00:00Z', courtIds: ['court-1'] }],
    capacity: 16,
    guestAllowance: 2,
    paymentMethod: 'PAYMENT_METHOD_CASH',
    entryFee: { amountCents: '2500', currencyCode: 'USD' },
    format: 'COMPETITION_FORMAT_DOUBLES',
    status: 'COMPETITION_STATUS_SCHEDULED',
  },
  spotsLeft: 5,
}

function fakeClient(list?: (...args: unknown[]) => unknown): CompetitionsClient {
  const GET = vi.fn((path: string, ...rest: unknown[]) => {
    if (path === '/v1/competitions') {
      return list?.(...rest) ?? { data: { competitions: [] }, error: undefined, response: { status: 200 } }
    }
    throw new Error(`unexpected GET in test fake: ${path}`)
  })
  return { GET, POST: vi.fn() } as unknown as CompetitionsClient
}

/** T10.8 (closes #98): DiscoverCompetitions forwards these to
 * CompetitionDetailPanel so its Host/Venue name joins resolve without a
 * real network call. */
function fakeIdentityClient(): IdentityClient {
  return {
    GET: vi.fn(async () => ({ data: { user: { displayName: 'Ada Lovelace' } } })),
    POST: vi.fn(),
  } as unknown as IdentityClient
}

function fakeFacilitiesClient(): FacilitiesClient {
  return {
    GET: vi.fn(async () => ({ data: { facility: { name: 'Riverside Courts' } } })),
    POST: vi.fn(),
  } as unknown as FacilitiesClient
}

function mountScreen(client: CompetitionsClient, width = 1024) {
  return mount(DiscoverCompetitions, {
    props: {
      client,
      identityClient: fakeIdentityClient(),
      facilitiesClient: fakeFacilitiesClient(),
      win: matchMediaForWidth(width),
    },
  })
}

describe('DiscoverCompetitions', () => {
  it('shows the loading state, then the list, on mount', async () => {
    const wrapper = mountScreen(
      fakeClient(async () => ({ data: { competitions: [LISTING] }, error: undefined, response: { status: 200 } })),
    )

    await nextTick()
    expect(wrapper.text()).toContain('Loading competitions')
    await flushPromises()

    expect(wrapper.text()).toContain('Autumn Doubles Ladder')
    expect(wrapper.text()).toContain('5 spots left')
    expect(wrapper.text()).not.toContain('Loading competitions')
  })

  it('shows the empty state when nothing is scheduled', async () => {
    const wrapper = mountScreen(
      fakeClient(async () => ({ data: { competitions: [] }, error: undefined, response: { status: 200 } })),
    )
    await flushPromises()
    expect(wrapper.text()).toContain('No competitions scheduled yet')
  })

  it('shows the error state when the API is unreachable', async () => {
    const wrapper = mountScreen(
      fakeClient(async () => {
        throw new TypeError('Failed to fetch')
      }),
    )
    await flushPromises()
    expect(wrapper.find('[role="alert"]').exists()).toBe(true)
  })

  it('shows the detail, derived from the already-fetched list, when a competition is selected', async () => {
    const client = fakeClient(async () => ({
      data: { competitions: [LISTING] },
      error: undefined,
      response: { status: 200 },
    }))
    const wrapper = mountScreen(client)
    await flushPromises()

    await wrapper.get('.competition-list__item').trigger('click')
    await flushPromises()

    // T10.8 (closes #98): the resolved Host name, not the raw hostId.
    expect(wrapper.text()).toContain('Ada Lovelace')
    expect(wrapper.text()).toContain('Doubles')
    // No second round trip: ListCompetitions already returned everything the
    // detail view needs, including the server-computed spots_left.
    expect(client.GET).toHaveBeenCalledTimes(1)
  })

  it('sends the facility and date filters to ListCompetitions', async () => {
    const client = fakeClient(async () => ({
      data: { competitions: [] },
      error: undefined,
      response: { status: 200 },
    }))
    const wrapper = mountScreen(client)
    await flushPromises()

    await wrapper.get('#competition-facility-filter').setValue('facility-2')
    await wrapper.get('#competition-date-filter').setValue('2026-09-01')
    await wrapper.get('.competition-list__search').trigger('submit')
    await flushPromises()

    const calls = (client.GET as ReturnType<typeof vi.fn>).mock.calls
    const lastCall = calls[calls.length - 1]
    const query = (lastCall?.[1] as { params: { query: Record<string, string> } }).params.query
    expect(query.venueFacilityId).toBe('facility-2')
    expect(query.startsAfter).toBeTruthy()
    expect(query.startsBefore).toBeTruthy()
  })

  it('on iPhone, does not render the detail panel until a competition is selected', async () => {
    const wrapper = mountScreen(
      fakeClient(async () => ({ data: { competitions: [LISTING] }, error: undefined, response: { status: 200 } })),
      375,
    )
    await flushPromises()
    expect(wrapper.find('.competition-detail').exists()).toBe(false)

    await wrapper.get('.competition-list__item').trigger('click')
    await flushPromises()
    expect(wrapper.find('.competition-detail').exists()).toBe(true)
  })

  it('on iPad/web, renders the detail panel with a placeholder before any selection', async () => {
    const wrapper = mountScreen(
      fakeClient(async () => ({ data: { competitions: [] }, error: undefined, response: { status: 200 } })),
    )
    await flushPromises()
    expect(wrapper.get('.competition-detail').text()).toContain('Select a competition')
  })

  it('exposes the resolved breakpoint for responsive styling', async () => {
    const wrapper = mountScreen(fakeClient())
    await flushPromises()
    expect(wrapper.get('.discover-competitions').attributes('data-breakpoint')).toBe('ipad')
  })

  it('opens straight onto a competition when the route already names one', async () => {
    const client = fakeClient(async () => ({
      data: { competitions: [LISTING] },
      error: undefined,
      response: { status: 200 },
    }))
    const wrapper = mount(DiscoverCompetitions, {
      props: {
        client,
        identityClient: fakeIdentityClient(),
        facilitiesClient: fakeFacilitiesClient(),
        win: matchMediaForWidth(1024),
        competitionId: 'c1',
      },
    })
    await flushPromises()

    // T10.8 (closes #98): the resolved Host name, not the raw hostId.
    expect(wrapper.get('.competition-detail').text()).toContain('Ada Lovelace')
  })

  it('explains a selection that no longer resolves, rather than rendering nothing', async () => {
    const client = fakeClient(async () => ({
      data: { competitions: [] },
      error: undefined,
      response: { status: 200 },
    }))
    const wrapper = mount(DiscoverCompetitions, {
      props: {
        client,
        identityClient: fakeIdentityClient(),
        facilitiesClient: fakeFacilitiesClient(),
        win: matchMediaForWidth(1024),
        competitionId: 'gone',
      },
    })
    await flushPromises()

    expect(wrapper.get('[role="alert"]').text()).toContain('Could not find this competition')
  })
})
