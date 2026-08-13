import { describe, it, expect, vi } from 'vitest'
import { nextTick } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import DiscoverGames from '../DiscoverGames.vue'
import type { SocialPlayClient } from '../../../api/socialplayClient'
import type { IdentityClient } from '../../../api/identityClient'
import type { FacilitiesClient } from '../../../api/facilitiesClient'

/** Mirrors DiscoverFacilities.spec.ts's identical fixed-viewport-width
 * matchMedia stand-in. */
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

const GAME_LISTING = {
  game: {
    id: 'g1',
    hostId: 'host-1',
    venueFacilityId: 'facility-1',
    courtIds: ['court-1'],
    startsAt: '2026-09-01T10:00:00Z',
    endsAt: '2026-09-01T11:00:00Z',
    capacity: 8,
    status: 'GAME_STATUS_SCHEDULED',
    paymentMethod: 'PAYMENT_METHOD_EITHER',
    guestAllowance: 2,
  },
  spotsLeft: 5,
}

function fakeClient(list?: (...args: unknown[]) => unknown): SocialPlayClient {
  const GET = vi.fn((path: string, ...rest: unknown[]) => {
    if (path === '/v1/games') return list?.(...rest) ?? { data: { games: [] }, error: undefined }
    throw new Error(`unexpected GET in test fake: ${path}`)
  })
  return { GET, POST: vi.fn() } as unknown as SocialPlayClient
}

/** T10.8 (closes #98): DiscoverGames forwards these to GameDetailPanel so
 * its Host/Facility name joins resolve without a real network call. */
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

describe('DiscoverGames', () => {
  it('shows the loading state, then the list, on mount', async () => {
    const client = fakeClient(async () => ({ data: { games: [GAME_LISTING] }, error: undefined }))
    const wrapper = mount(DiscoverGames, { props: {
        client,
        identityClient: fakeIdentityClient(),
        facilitiesClient: fakeFacilitiesClient(),
        win: matchMediaForWidth(1024),
      } })

    await nextTick()
    expect(wrapper.text()).toContain('Loading games')
    await flushPromises()

    expect(wrapper.text()).toContain('5 spots left')
    expect(wrapper.text()).not.toContain('Loading games')
  })

  it('shows the empty state when there are no games yet', async () => {
    const client = fakeClient(async () => ({ data: { games: [] }, error: undefined }))
    const wrapper = mount(DiscoverGames, { props: {
        client,
        identityClient: fakeIdentityClient(),
        facilitiesClient: fakeFacilitiesClient(),
        win: matchMediaForWidth(1024),
      } })
    await flushPromises()

    expect(wrapper.text()).toContain('No games yet')
  })

  it('shows the error state when the games API is unreachable', async () => {
    const client = fakeClient(async () => {
      throw new TypeError('Failed to fetch')
    })
    const wrapper = mount(DiscoverGames, { props: {
        client,
        identityClient: fakeIdentityClient(),
        facilitiesClient: fakeFacilitiesClient(),
        win: matchMediaForWidth(1024),
      } })
    await flushPromises()

    expect(wrapper.find('[role="alert"]').exists()).toBe(true)
  })

  it('shows game details (derived from the already-fetched list, no second fetch) when a game is selected', async () => {
    const client = fakeClient(async () => ({ data: { games: [GAME_LISTING] }, error: undefined }))
    const wrapper = mount(DiscoverGames, { props: {
        client,
        identityClient: fakeIdentityClient(),
        facilitiesClient: fakeFacilitiesClient(),
        win: matchMediaForWidth(1024),
      } })
    await flushPromises()

    await wrapper.find('.game-list__item').trigger('click')
    await flushPromises()

    // T10.8 (closes #98): the resolved Host/venue names, not the raw ids.
    expect(wrapper.text()).toContain('Ada Lovelace')
    expect(wrapper.text()).toContain('Riverside Courts')
    expect(client.GET).toHaveBeenCalledTimes(1)
  })

  it('on iPhone, does not render the detail panel until a game is selected', async () => {
    const client = fakeClient(async () => ({ data: { games: [GAME_LISTING] }, error: undefined }))
    const wrapper = mount(DiscoverGames, { props: {
        client,
        identityClient: fakeIdentityClient(),
        facilitiesClient: fakeFacilitiesClient(),
        win: matchMediaForWidth(375),
      } })
    await flushPromises()

    expect(wrapper.find('.game-detail').exists()).toBe(false)

    await wrapper.find('.game-list__item').trigger('click')
    await flushPromises()

    expect(wrapper.find('.game-detail').exists()).toBe(true)
  })

  it('on iPad/web, renders the detail panel with a placeholder before any selection', async () => {
    const client = fakeClient(async () => ({ data: { games: [] }, error: undefined }))
    const wrapper = mount(DiscoverGames, { props: {
        client,
        identityClient: fakeIdentityClient(),
        facilitiesClient: fakeFacilitiesClient(),
        win: matchMediaForWidth(1024),
      } })
    await flushPromises()

    expect(wrapper.find('.game-detail').exists()).toBe(true)
    expect(wrapper.text()).toContain('Select a game')
  })

  it('exposes the resolved breakpoint on the root element for responsive styling', async () => {
    const client = fakeClient()
    const wrapper = mount(DiscoverGames, { props: {
        client,
        identityClient: fakeIdentityClient(),
        facilitiesClient: fakeFacilitiesClient(),
        win: matchMediaForWidth(1024),
      } })
    await flushPromises()

    expect(wrapper.get('.discover-games').attributes('data-breakpoint')).toBe('ipad')
  })
})
