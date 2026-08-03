import { describe, it, expect, vi } from 'vitest'
import { nextTick } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import DiscoverFacilities from '../DiscoverFacilities.vue'
import type { FacilitiesClient } from '../../../api/facilitiesClient'
import type { BookingClient } from '../../../api/bookingClient'

/**
 * Minimal `window.matchMedia` stand-in for a fixed viewport width — see
 * composables/__tests__/useBreakpoint.spec.ts for the fuller, live-update
 * version this borrows the query-matching logic from. This screen only
 * needs the breakpoint resolved once at mount for these tests.
 */
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

function fakeClient(handlers: {
  list?: (...args: unknown[]) => unknown
  detail?: (...args: unknown[]) => unknown
}): FacilitiesClient {
  const get = vi.fn((path: string, ...rest: unknown[]) => {
    if (path === '/v1/facilities') return handlers.list?.(...rest) ?? { data: { facilities: [] }, error: undefined }
    if (path === '/v1/facilities/{facilityId}')
      return handlers.detail?.(...rest) ?? { data: { facility: undefined }, error: undefined }
    throw new Error(`unexpected path in test fake: ${path}`)
  })
  return { GET: get } as unknown as FacilitiesClient
}

const RIVERSIDE_WITH_CAMERA_LINKS = {
  id: 'f1',
  name: 'Riverside Courts',
  description: 'Six outdoor courts by the river.',
  address: '100 River Rd, Springfield',
  photoUrls: ['https://cdn.example.test/riverside-1.jpg'],
  cameraConsentAttested: true,
  cameraLinks: [{ url: 'https://cameras.example.test/riverside/secret-feed', courtId: '' }],
}

describe('DiscoverFacilities', () => {
  it('shows the loading state, then the list, on mount', async () => {
    const client = fakeClient({
      list: async () => ({ data: { facilities: [{ ...RIVERSIDE_WITH_CAMERA_LINKS }] }, error: undefined }),
    })
    const wrapper = mount(DiscoverFacilities, { props: { client, win: matchMediaForWidth(1024) } })

    await nextTick()
    expect(wrapper.text()).toContain('Loading facilities')
    await flushPromises()

    expect(wrapper.text()).toContain('Riverside Courts')
    expect(wrapper.text()).not.toContain('Loading facilities')
  })

  it('shows the empty state when there are no facilities yet', async () => {
    const client = fakeClient({ list: async () => ({ data: { facilities: [] }, error: undefined }) })
    const wrapper = mount(DiscoverFacilities, { props: { client, win: matchMediaForWidth(1024) } })
    await flushPromises()

    expect(wrapper.text()).toContain('No facilities yet')
  })

  it('shows the error state when the facilities API is unreachable', async () => {
    const client = fakeClient({
      list: async () => {
        throw new TypeError('Failed to fetch')
      },
    })
    const wrapper = mount(DiscoverFacilities, { props: { client, win: matchMediaForWidth(1024) } })
    await flushPromises()

    expect(wrapper.find('[role="alert"]').exists()).toBe(true)
  })

  it('loads and shows facility details, including the zero-courts empty state, when a facility is selected', async () => {
    const client = fakeClient({
      list: async () => ({ data: { facilities: [{ ...RIVERSIDE_WITH_CAMERA_LINKS }] }, error: undefined }),
      detail: async () => ({ data: { facility: { ...RIVERSIDE_WITH_CAMERA_LINKS } }, error: undefined }),
    })
    const wrapper = mount(DiscoverFacilities, { props: { client, win: matchMediaForWidth(1024) } })
    await flushPromises()

    await wrapper.find('.facility-list__item').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('Six outdoor courts by the river.')
    expect(wrapper.text()).toContain("hasn't listed any courts yet")
  })

  it('never renders camera-link data anywhere on screen, even though the mocked API response includes it', async () => {
    const client = fakeClient({
      list: async () => ({ data: { facilities: [{ ...RIVERSIDE_WITH_CAMERA_LINKS }] }, error: undefined }),
      detail: async () => ({ data: { facility: { ...RIVERSIDE_WITH_CAMERA_LINKS } }, error: undefined }),
    })
    const wrapper = mount(DiscoverFacilities, { props: { client, win: matchMediaForWidth(1024) } })
    await flushPromises()

    // Camera link data must not leak into the list view either.
    expect(wrapper.html()).not.toContain('cameras.example.test')

    await wrapper.find('.facility-list__item').trigger('click')
    await flushPromises()

    expect(wrapper.html()).not.toContain('cameras.example.test')
    expect(wrapper.html().toLowerCase()).not.toContain('camera')
  })

  it('exposes the resolved breakpoint on the root element for responsive styling', async () => {
    const client = fakeClient({})
    const wrapper = mount(DiscoverFacilities, { props: { client, win: matchMediaForWidth(1024) } })
    await flushPromises()

    expect(wrapper.get('.discover').attributes('data-breakpoint')).toBe('ipad')
  })

  it('on iPhone, does not render the detail panel (with its "select a facility" placeholder) until a facility is selected', async () => {
    const client = fakeClient({
      list: async () => ({ data: { facilities: [{ ...RIVERSIDE_WITH_CAMERA_LINKS }] }, error: undefined }),
      detail: async () => ({ data: { facility: { ...RIVERSIDE_WITH_CAMERA_LINKS } }, error: undefined }),
    })
    const wrapper = mount(DiscoverFacilities, { props: { client, win: matchMediaForWidth(375) } })
    await flushPromises()

    expect(wrapper.find('.facility-detail').exists()).toBe(false)

    await wrapper.find('.facility-list__item').trigger('click')
    await flushPromises()

    expect(wrapper.find('.facility-detail').exists()).toBe(true)
    expect(wrapper.text()).toContain('Riverside Courts')
  })

  it('on iPad/web, renders the detail panel with a placeholder before any selection (two-column list+detail)', async () => {
    const client = fakeClient({ list: async () => ({ data: { facilities: [] }, error: undefined }) })
    const wrapper = mount(DiscoverFacilities, { props: { client, win: matchMediaForWidth(1024) } })
    await flushPromises()

    expect(wrapper.find('.facility-detail').exists()).toBe(true)
    expect(wrapper.text()).toContain('Select a facility')
  })

  describe('booking entry point (T7.6)', () => {
    function fakeBookingClient(): BookingClient {
      return { POST: vi.fn(), GET: vi.fn() } as unknown as BookingClient
    }

    it('shows the quote/book flow for a manually-entered court id once "Book this court" is submitted', async () => {
      const client = fakeClient({
        list: async () => ({ data: { facilities: [{ ...RIVERSIDE_WITH_CAMERA_LINKS }] }, error: undefined }),
        detail: async () => ({ data: { facility: { ...RIVERSIDE_WITH_CAMERA_LINKS } }, error: undefined }),
      })
      const bookingClient = fakeBookingClient()
      const wrapper = mount(DiscoverFacilities, { props: { client, bookingClient, win: matchMediaForWidth(1024) } })
      await flushPromises()

      expect(wrapper.text()).not.toContain('Book Court court-123')

      await wrapper.find('.facility-list__item').trigger('click')
      await flushPromises()

      await wrapper.find('.facility-detail__manual-booking-input').setValue('court-123')
      await wrapper.find('.facility-detail__manual-booking').trigger('submit')
      await flushPromises()

      expect(wrapper.text()).toContain('Book Court court-123')

      await wrapper.find('.court-booking__close').trigger('click')
      await flushPromises()

      expect(wrapper.text()).not.toContain('Book Court court-123')
    })
  })
})
