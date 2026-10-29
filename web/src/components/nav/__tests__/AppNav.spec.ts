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

    // Retargeted from '/facilities' to '/bookings' when Discover gained a
    // sub-link of its own (see the describe block below): /facilities is no
    // longer a childless area, so mounting there stopped proving that
    // sub-links are area-scoped. /bookings has no children.
    it('is not shown while a different area is active', async () => {
      const wrapper = await mountAtWidth(1440, emptyGamesClient(), '/bookings')
      expect(wrapper.findAll('.app-nav__subitem')).toHaveLength(0)
    })

    it('does not leak the Games sub-links into the Discover area', async () => {
      const wrapper = await mountAtWidth(1440, emptyGamesClient(), '/facilities')
      const labels = wrapper.findAll('.app-nav__subitem').map((s) => s.text())
      expect(labels).not.toContain('Create a competition')
      expect(labels).not.toContain('Create a game')
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
                // T56.1: the frozen owed figure a real server returns.
                // Without it, T58's filter drops the row and this badge
                // would count 0 for a reason unrelated to the badge.
                { id: 'r1', gameId: 'g1', playerId: 'player-1', status: 'REGISTRATION_STATUS_REGISTERED', paymentStatus: 'PAYMENT_STATUS_UNPAID', guestCount: 0, amountOwed: { amountCents: '2500', currencyCode: 'USD' } },
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

    // T11.7 (WCAG 4.1.3 Status Messages): this badge changes with no direct
    // user action on it — marking a payment paid elsewhere in the app
    // updates the count while AppNav stays mounted, unchanged, in the
    // persistent chrome (App.vue). Before this fix there was no role and no
    // aria-live anywhere on it. The live region must exist REGARDLESS of
    // whether the badge is currently showing a count — an aria-live element
    // that only gets inserted into the DOM once the count becomes nonzero
    // is exactly the unreliable shape DisplayName.vue/VenueName.vue's own
    // T10.8 fix avoided (a persistent element AT is already watching, not
    // one that pops into existence with the change already applied).
    it('carries a persistent aria-live region for the Payments count, even with nothing pending', async () => {
      __resetPendingCashPaymentsCountForTests()
      const wrapper = await mountAtWidth(1440, emptyGamesClient())
      await flushPromises()

      const liveRegion = wrapper.find('.app-nav__badge-live')
      expect(liveRegion.exists()).toBe(true)
      expect(liveRegion.attributes('aria-live')).toBe('polite')
      // No visible badge yet, but the live region itself is already there.
      expect(liveRegion.find('.app-nav__badge').exists()).toBe(false)
    })

    it('keeps the count inside the same persistent aria-live region once pending payments exist', async () => {
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
                // T56.1: the frozen owed figure a real server returns.
                // Without it, T58's filter drops the row and this badge
                // would count 0 for a reason unrelated to the badge.
                { id: 'r1', gameId: 'g1', playerId: 'player-1', status: 'REGISTRATION_STATUS_REGISTERED', paymentStatus: 'PAYMENT_STATUS_UNPAID', guestCount: 0, amountOwed: { amountCents: '2500', currencyCode: 'USD' } },
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

      const liveRegion = wrapper.find('.app-nav__badge-live')
      expect(liveRegion.exists()).toBe(true)
      expect(liveRegion.attributes('aria-live')).toBe('polite')
      expect(liveRegion.find('.app-nav__badge').exists()).toBe(true)
      expect(liveRegion.text()).toContain('1')
    })
  })

  // The two screens the MVP journey cannot do without — facility onboarding
  // and game creation — were reachable only by typing their URL. Nothing in
  // the UI linked to /facilities/onboard or /games/new:
  //
  //   $ grep -rn "facilities/onboard\\|games/new" web/src --include=*.vue
  //   (nothing outside the route table and the specs)
  //
  // Found 2026-10-17 while writing the README's "your first run" walkthrough,
  // which had to describe a nav path that did not exist. Note what this
  // means about the sub-level mechanism T9.6 built: it was used to reach
  // *competition* creation and never for the game creation underneath it.
  //
  // These are sub-links, not tabs, for T9.6's reason: T8.1's design-handoff
  // mapping fixed the top-level tab set at five, and "create one" is an
  // action within an area rather than a destination of equal rank.
  describe('the create screens are reachable by clicking, not only by URL', () => {
    it('adds no new top-level tab', async () => {
      // The Payments tab's badge text is part of that tab's own text(), and
      // `pendingCashPaymentsCount` is module-level shared state the badge
      // tests above leave set — so reset it, exactly as they do.
      __resetPendingCashPaymentsCountForTests()
      const wrapper = await mountAtWidth(1440, emptyGamesClient(), '/facilities')
      const labels = wrapper.findAll('.app-nav__item').map((item) => item.text())
      expect(labels).toEqual(['Discover', 'Bookings', 'Games', 'Payments', 'Profile'])
    })

    it('links facility onboarding under the Discover area', async () => {
      const wrapper = await mountAtWidth(1440, emptyGamesClient(), '/facilities')
      const sub = wrapper.findAll('.app-nav__subitem')
      const link = sub.find((s) => s.text() === 'Add a facility')
      expect(link, 'no "Add a facility" sub-link under Discover').toBeTruthy()
      expect(link!.attributes('href')).toBe('/facilities/onboard')
    })

    it('keeps that link visible once you are on the onboarding screen', async () => {
      const wrapper = await mountAtWidth(1440, emptyGamesClient(), '/facilities/onboard')
      const labels = wrapper.findAll('.app-nav__subitem').map((s) => s.text())
      expect(labels).toContain('Add a facility')
    })

    // The MVP gap audit's finding 4: /competitions — the screen a player
    // browses and enters competitions from — had NO link anywhere in the
    // app. It was in the Games tab's `areaPrefixes` (so the sub-menu shows
    // once you are already there) but was never a destination, so the only
    // way to find it was a host's shared /c/:token link or typing the URL.
    // #346 made the CREATE screens clickable and left discovery behind.
    it('links competition discovery under the Games area, not just creation', async () => {
      const wrapper = await mountAtWidth(1440, emptyGamesClient(), '/games')
      const sub = wrapper.findAll('.app-nav__subitem')
      const link = sub.find((s) => s.text() === 'Browse competitions')
      expect(link, 'no "Browse competitions" sub-link under Games').toBeTruthy()
      expect(link!.attributes('href')).toBe('/competitions')
    })

    it('keeps that link visible once you are browsing competitions', async () => {
      const wrapper = await mountAtWidth(1440, emptyGamesClient(), '/competitions')
      const labels = wrapper.findAll('.app-nav__subitem').map((s) => s.text())
      expect(labels).toContain('Browse competitions')
    })

    it('links game creation under the Games area', async () => {
      const wrapper = await mountAtWidth(1440, emptyGamesClient(), '/games')
      const sub = wrapper.findAll('.app-nav__subitem')
      const link = sub.find((s) => s.text() === 'Create a game')
      expect(link, 'no "Create a game" sub-link under Games').toBeTruthy()
      expect(link!.attributes('href')).toBe('/games/new')
    })

    it('keeps that link visible once you are on the creation screen', async () => {
      const wrapper = await mountAtWidth(1440, emptyGamesClient(), '/games/new')
      const labels = wrapper.findAll('.app-nav__subitem').map((s) => s.text())
      expect(labels).toContain('Create a game')
    })

    // Sidebar-only would leave both screens unreachable on iPhone, where the
    // tab bar is the whole navigation — T9.6's own reason for rendering
    // sub-links in both variants.
    it('reaches both on iPhone, not sidebar-only', async () => {
      const discover = await mountAtWidth(375, emptyGamesClient(), '/facilities')
      expect(discover.findAll('.app-nav__subitem').map((s) => s.text())).toContain(
        'Add a facility',
      )
      const games = await mountAtWidth(375, emptyGamesClient(), '/games')
      expect(games.findAll('.app-nav__subitem').map((s) => s.text())).toContain('Create a game')
    })
  })

})
