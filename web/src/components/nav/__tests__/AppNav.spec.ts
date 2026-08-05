import { describe, it, expect, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createRouter, createMemoryHistory } from 'vue-router'
import AppNav from '../AppNav.vue'
import { routes } from '../../../router'
import type { SocialPlayClient } from '../../../api/socialplayClient'
import { __resetPendingCashPaymentsCountForTests } from '../../../state/hostPendingPayments'

/** A SocialPlayClient stand-in that returns no Games at all (T8.10's
 * pending-cash-payments badge fetch, added to AppNav — see that file's
 * header comment) — the tests in this file that don't care about the
 * badge just need this fetch to resolve harmlessly rather than hit the
 * ambient ("ambient" = real, unmocked) ~fetch~. */
function emptyGamesClient(): SocialPlayClient {
  return {
    GET: vi.fn(async () => ({ data: { games: [] }, error: undefined, response: { status: 200 } })),
    POST: vi.fn(),
  } as unknown as SocialPlayClient
}

/**
 * Static `window.matchMedia` stand-in for a fixed viewport width — same
 * pattern as DiscoverFacilities.spec.ts's `matchMediaForWidth` (this
 * component only needs the breakpoint resolved once at mount for these
 * tests, not live resize updates — those are useBreakpoint's own unit
 * tests, see composables/__tests__/useBreakpoint.spec.ts).
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

async function mountAtWidth(
  width: number,
  client: SocialPlayClient = emptyGamesClient(),
  startAt = '/facilities',
) {
  const router = createRouter({ history: createMemoryHistory(), routes })
  router.push(startAt)
  await router.isReady()
  return mount(AppNav, {
    props: { win: matchMediaForWidth(width), client },
    global: { plugins: [router] },
  })
}

describe('AppNav', () => {
  it('renders the persistent sidebar on web (>=1280px)', async () => {
    const wrapper = await mountAtWidth(1440)
    expect(wrapper.get('.app-nav').attributes('data-nav-variant')).toBe('sidebar')
    expect(wrapper.find('.app-nav--sidebar').exists()).toBe(true)
    expect(wrapper.find('.app-nav--tabbar').exists()).toBe(false)
  })

  it('renders a bottom tab bar on iPhone (<600px)', async () => {
    const wrapper = await mountAtWidth(375)
    expect(wrapper.get('.app-nav').attributes('data-nav-variant')).toBe('tabbar')
    expect(wrapper.find('.app-nav--tabbar').exists()).toBe(true)
    expect(wrapper.find('.app-nav--sidebar').exists()).toBe(false)
  })

  it('renders a bottom tab bar on iPad (768-1180px)', async () => {
    const wrapper = await mountAtWidth(1024)
    expect(wrapper.get('.app-nav').attributes('data-nav-variant')).toBe('tabbar')
    expect(wrapper.find('.app-nav--tabbar').exists()).toBe(true)
    expect(wrapper.find('.app-nav--sidebar').exists()).toBe(false)
  })

  it('exposes exactly the Discover / Bookings / Games / Payments / Profile tabs, in that order', async () => {
    const wrapper = await mountAtWidth(1440)
    const labels = wrapper.findAll('.app-nav__item').map((item) => item.text())
    expect(labels).toEqual(['Discover', 'Bookings', 'Games', 'Payments', 'Profile'])
  })

  it("links Bookings, Payments, and Profile to their routes rather than omitting them", async () => {
    const wrapper = await mountAtWidth(1440)
    const links = wrapper.findAll('.app-nav__item')
    const byLabel = Object.fromEntries(links.map((l) => [l.text(), l.attributes('href')]))
    expect(byLabel['Discover']).toBe('/facilities')
    expect(byLabel['Bookings']).toBe('/bookings')
    expect(byLabel['Games']).toBe('/games')
    expect(byLabel['Payments']).toBe('/host/payments')
    expect(byLabel['Profile']).toBe('/profile')
  })

  // T9.6: Competitions lives UNDER the existing Games area, not as a sixth
  // top-level tab — T8.1's design-handoff mapping fixed the tab set, so the
  // nav gains a sub-level rather than a new destination in the tab bar.
  describe('Competitions (T9.6) sits under the Games area', () => {
    it('adds NO new top-level tab — the tab set is still exactly the five', async () => {
      const wrapper = await mountAtWidth(1440, emptyGamesClient(), '/competitions/new')
      const labels = wrapper.findAll('.app-nav__item').map((item) => item.text())
      expect(labels).toEqual(['Discover', 'Bookings', 'Games', 'Payments', 'Profile'])
    })

    it('shows the Competitions sub-link while the Games area is active', async () => {
      const wrapper = await mountAtWidth(1440, emptyGamesClient(), '/games')
      const sub = wrapper.findAll('.app-nav__subitem')
      expect(sub.map((s) => s.text())).toContain('Create a competition')
      expect(sub.find((s) => s.text() === 'Create a competition')!.attributes('href')).toBe(
        '/competitions/new',
      )
    })

    it('stays visible on a /competitions route, so the Host can see where they are', async () => {
      const wrapper = await mountAtWidth(1440, emptyGamesClient(), '/competitions/new')
      expect(wrapper.findAll('.app-nav__subitem').length).toBeGreaterThan(0)
    })

    it('is not shown while a different area is active', async () => {
      const wrapper = await mountAtWidth(1440, emptyGamesClient(), '/facilities')
      expect(wrapper.findAll('.app-nav__subitem')).toHaveLength(0)
    })

    it('is reachable on iPhone too, not sidebar-only', async () => {
      const wrapper = await mountAtWidth(375, emptyGamesClient(), '/games')
      expect(wrapper.findAll('.app-nav__subitem').length).toBeGreaterThan(0)
    })
  })

  // T8.10: the Payments tab's pending-cash-payments count badge.
  describe('Payments tab pending-count badge', () => {
    it('shows no badge when there are no pending cash payments', async () => {
      __resetPendingCashPaymentsCountForTests()
      const wrapper = await mountAtWidth(1440, emptyGamesClient())
      await flushPromises()
      expect(wrapper.find('.app-nav__badge').exists()).toBe(false)
    })

    it('shows a count badge on the Payments tab once pending cash registrations are found', async () => {
      __resetPendingCashPaymentsCountForTests()
      const client: SocialPlayClient = {
        GET: vi.fn(async (path: string) => {
          if (path === '/v1/games') {
            return {
              data: {
                games: [
                  {
                    game: {
                      id: 'g1',
                      hostId: 'host-mock-1',
                      venueFacilityId: 'facility-1',
                      courtIds: ['court-1'],
                      startsAt: '2026-09-01T10:00:00Z',
                      endsAt: '2026-09-01T11:00:00Z',
                      capacity: 8,
                      status: 'GAME_STATUS_SCHEDULED',
                      paymentMethod: 'PAYMENT_METHOD_CASH',
                      guestAllowance: 2,
                      entryFee: { amountCents: '1000', currencyCode: 'USD' },
                    },
                    spotsLeft: 5,
                  },
                ],
              },
              error: undefined,
              response: { status: 200 },
            }
          }
          return {
            data: {
              registrations: [
                { id: 'r1', gameId: 'g1', playerId: 'player-1', status: 'REGISTRATION_STATUS_REGISTERED', paymentStatus: 'PAYMENT_STATUS_UNPAID', guestCount: 0 },
              ],
            },
            error: undefined,
            response: { status: 200 },
          }
        }),
        POST: vi.fn(),
      } as unknown as SocialPlayClient

      const wrapper = await mountAtWidth(1440, client)
      await flushPromises()
      await wrapper.vm.$nextTick()

      expect(wrapper.find('.app-nav__badge').text()).toContain('1')
    })
  })
})
