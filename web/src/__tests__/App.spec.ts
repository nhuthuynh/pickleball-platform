import { describe, it, expect, beforeAll, beforeEach, afterEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createRouter, createMemoryHistory } from 'vue-router'
import App from '../App.vue'
import { routes } from '../router'
import DiscoverFacilities from '../components/discover/DiscoverFacilities.vue'
import DiscoverGames from '../components/discover-games/DiscoverGames.vue'
import FacilityOnboarding from '../views/FacilityOnboarding.vue'
import GameCreation from '../views/GameCreation.vue'
import GameCheckout from '../views/GameCheckout.vue'
import HostPayments from '../views/HostPayments.vue'
import Profile from '../views/Profile.vue'
import MyBookings from '../views/MyBookings.vue'
import ComingSoonView from '../views/placeholders/ComingSoonView.vue'

// jsdom doesn't implement matchMedia — same minimal always-"no match" stub
// FacilityOnboarding.spec.ts and DiscoverFacilities.spec.ts already use.
beforeAll(() => {
  if (!window.matchMedia) {
    window.matchMedia = (query: string) =>
      ({
        matches: false,
        media: query,
        onchange: null,
        addListener: () => {},
        removeListener: () => {},
        addEventListener: () => {},
        removeEventListener: () => {},
        dispatchEvent: () => false,
      }) as unknown as MediaQueryList
  }
})

// DiscoverFacilities calls its (real, non-injected — App.vue doesn't wire a
// fake client through the router) facilitiesClient's ListFacilities on
// mount. Stub the ambient fetch so that's a fast, deterministic rejection
// (caught by useFacilityList's own error handling — see
// composables/useFacilityList.ts) rather than a real network call.
beforeEach(() => {
  vi.stubGlobal(
    'fetch',
    vi.fn(() => Promise.reject(new Error('network disabled in tests'))),
  )
})

afterEach(() => {
  vi.unstubAllGlobals()
})

async function mountAppAt(path: string) {
  const router = createRouter({ history: createMemoryHistory(), routes })
  router.push(path)
  await router.isReady()
  const wrapper = mount(App, { global: { plugins: [router] } })
  await flushPromises()
  return wrapper
}

// The regression this ticket (T8.1) exists to fix: before this ticket,
// App.vue rendered FacilityOnboarding and DiscoverFacilities as stacked
// block siblings regardless of the URL. These assertions fail against that
// old App.vue (both components would be found at both paths).
describe('App routing', () => {
  it('renders only DiscoverFacilities at /facilities', async () => {
    const wrapper = await mountAppAt('/facilities')

    expect(wrapper.findComponent(DiscoverFacilities).exists()).toBe(true)
    expect(wrapper.findComponent(FacilityOnboarding).exists()).toBe(false)
    expect(wrapper.findComponent(ComingSoonView).exists()).toBe(false)
  })

  it('renders only FacilityOnboarding at /facilities/onboard', async () => {
    const wrapper = await mountAppAt('/facilities/onboard')

    expect(wrapper.findComponent(FacilityOnboarding).exists()).toBe(true)
    expect(wrapper.findComponent(DiscoverFacilities).exists()).toBe(false)
    expect(wrapper.findComponent(ComingSoonView).exists()).toBe(false)
  })

  // T68.3 (part of #353) replaces the /bookings "Coming soon" placeholder
  // with the real Bookings screen, now that T67's two actor-scoped reads
  // (`ListBookingsForActor`, `ListRegistrationsForActor`) exist to back it —
  // same "only the matched route's screen renders" regression check /profile
  // and /games each got when their own placeholders were replaced.
  //
  // This test used to assert the OPPOSITE (that the placeholder rendered,
  // with the literal text "Coming soon."), which is the assertion that had
  // to be inverted for this ticket. With it inverted, NO route renders
  // ComingSoonView any more — every remaining `ComingSoonView` assertion in
  // this file is a negative one, guarding against a route regressing BACK
  // to a placeholder.
  it('renders only MyBookings at /bookings', async () => {
    const wrapper = await mountAppAt('/bookings')

    expect(wrapper.findComponent(MyBookings).exists()).toBe(true)
    expect(wrapper.findComponent(ComingSoonView).exists()).toBe(false)
    expect(wrapper.findComponent(DiscoverFacilities).exists()).toBe(false)
    expect(wrapper.findComponent(FacilityOnboarding).exists()).toBe(false)
    expect(wrapper.findComponent(GameCreation).exists()).toBe(false)
    // WCAG 2.4.2 — the route's own `meta.title`, still set by the real guard.
    expect(wrapper.text()).toContain('Bookings')
    expect(wrapper.text()).not.toContain('Coming soon.')
  })

  // T10.5 (docs/process/t10-sprint-plan.md) replaces the /profile
  // "Coming soon" placeholder with the real Profile screen, now that
  // Identity/Users (T10.1-T10.2) exists to back it — same "only the
  // matched route's screen renders" regression check every other real
  // screen above gets.
  it('renders only Profile at /profile', async () => {
    const wrapper = await mountAppAt('/profile')

    expect(wrapper.findComponent(Profile).exists()).toBe(true)
    expect(wrapper.findComponent(ComingSoonView).exists()).toBe(false)
    expect(wrapper.findComponent(DiscoverFacilities).exists()).toBe(false)
    expect(wrapper.findComponent(FacilityOnboarding).exists()).toBe(false)
    expect(wrapper.findComponent(GameCreation).exists()).toBe(false)
  })

  // T8.9 (Discover & Join Games) replaces the /games placeholder with a
  // real screen — this is the same "only the matched route's screen
  // renders" regression T8.1's own tests above already guard for
  // /facilities and /facilities/onboard.
  it('renders only DiscoverGames at /games', async () => {
    const wrapper = await mountAppAt('/games')

    expect(wrapper.findComponent(DiscoverGames).exists()).toBe(true)
    expect(wrapper.findComponent(ComingSoonView).exists()).toBe(false)
    expect(wrapper.findComponent(DiscoverFacilities).exists()).toBe(false)
    expect(wrapper.findComponent(FacilityOnboarding).exists()).toBe(false)
    expect(wrapper.findComponent(GameCreation).exists()).toBe(false)
  })

  // T8.8 (docs/process/t8-sprint-plan.md) replaces the /games/new
  // placeholder with the real GameCreation screen — this is the ticket's
  // own version of the same stacked-siblings regression check every other
  // real screen above gets.
  it('renders only GameCreation at /games/new', async () => {
    const wrapper = await mountAppAt('/games/new')

    expect(wrapper.findComponent(GameCreation).exists()).toBe(true)
    expect(wrapper.findComponent(ComingSoonView).exists()).toBe(false)
    expect(wrapper.findComponent(DiscoverFacilities).exists()).toBe(false)
    expect(wrapper.findComponent(FacilityOnboarding).exists()).toBe(false)
    expect(wrapper.findComponent(DiscoverGames).exists()).toBe(false)
  })

  // T8.10 (Payments UI) replaces the /games/:id/checkout and /host/payments
  // placeholders with real screens — same "only the matched route's screen
  // renders" regression check every other real screen above gets.
  it('renders only GameCheckout at /games/:id/checkout', async () => {
    const wrapper = await mountAppAt('/games/abc-123/checkout')

    expect(wrapper.findComponent(GameCheckout).exists()).toBe(true)
    expect(wrapper.findComponent(ComingSoonView).exists()).toBe(false)
    expect(wrapper.findComponent(DiscoverFacilities).exists()).toBe(false)
    expect(wrapper.findComponent(FacilityOnboarding).exists()).toBe(false)
    expect(wrapper.findComponent(DiscoverGames).exists()).toBe(false)
  })

  it('renders only HostPayments at /host/payments', async () => {
    const wrapper = await mountAppAt('/host/payments')

    expect(wrapper.findComponent(HostPayments).exists()).toBe(true)
    expect(wrapper.findComponent(ComingSoonView).exists()).toBe(false)
    expect(wrapper.findComponent(DiscoverFacilities).exists()).toBe(false)
    expect(wrapper.findComponent(FacilityOnboarding).exists()).toBe(false)
    expect(wrapper.findComponent(DiscoverGames).exists()).toBe(false)
  })

  it('redirects / to /facilities', async () => {
    const wrapper = await mountAppAt('/')

    expect(wrapper.findComponent(DiscoverFacilities).exists()).toBe(true)
  })

  it('always renders the persistent header chrome (brand + RoleIndicator) regardless of route', async () => {
    const wrapper = await mountAppAt('/games')

    expect(wrapper.text()).toContain('Court&Play')
    expect(wrapper.find('.role-indicator').exists()).toBe(true)
  })
})
