// T11.7 (docs/process/t11-sprint-plan.md) instruction #1: "an automated pass
// (axe-core or equivalent) across every route web/src/router/index.ts
// currently registers." This file is that pass — one test per registered
// route, mounting the real App shell (chrome + RouterView) at that route,
// exactly the way App.spec.ts's own `mountAppAt` helper already does for the
// routing-regression checks, and running axe-core against the whole
// rendered page.
//
// Every route resolves to its NETWORK-ERROR/EMPTY state here (fetch is
// stubbed to reject, same as App.spec.ts) rather than a populated one — that
// is deliberate: this sweep's job is catching structural a11y defects (bad
// roles, missing labels, invalid ARIA, broken heading order) that are
// present regardless of what data loaded, not data-shaped ones. The two
// flagship flows' POPULATED/interactive states (a fetched quote, a booking
// conflict, a game-full state, a payment confirmation) are covered
// separately in flagshipFlows.spec.ts against the real component tree with
// fixture data, which is where a per-state check actually belongs.
import { describe, it, expect, beforeAll, beforeEach, afterEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createRouter, createMemoryHistory } from 'vue-router'
import App from '../App.vue'
import { routes } from '../router'
import { expectNoA11yViolations } from '../test-support/axe'

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
  // axe-core requires its target node to be attached to `document` (it
  // validates the node is part of a live document context before running
  // any rule) — a detached `mount()` tree, which is fine for @vue/test-utils
  // assertions, isn't enough for axe. `attachTo: document.body` plus
  // `wrapper.unmount()` in the caller keeps each test's DOM isolated from
  // the next, same discipline as any other attached-mount test.
  const wrapper = mount(App, { global: { plugins: [router] }, attachTo: document.body })
  await flushPromises()
  return wrapper
}

// Every path currently in router/index.ts's route table, with concrete
// values filled in for dynamic segments (mirroring App.spec.ts's/
// competitionRoutes.spec.ts's own fixture ids) — one entry per registered
// route, not per URL App.vue could theoretically resolve, so this list stays
// in sync with router/index.ts by construction: adding a route here is the
// same discipline as adding it there.
const ROUTES_UNDER_TEST: { path: string; label: string }[] = [
  { path: '/facilities', label: 'facilities' },
  { path: '/facilities/onboard', label: 'facilities-onboard' },
  { path: '/games', label: 'games' },
  { path: '/games/new', label: 'games-new' },
  { path: '/games/game-1/checkout', label: 'game-checkout' },
  { path: '/host/payments', label: 'host-payments' },
  { path: '/competitions/new', label: 'competitions-new' },
  { path: '/competitions/comp-1/manage', label: 'competition-manage' },
  { path: '/competitions/comp-1/checkout', label: 'competition-checkout' },
  { path: '/competitions', label: 'competitions' },
  { path: '/competitions/comp-1', label: 'competition-detail' },
  { path: '/c/share-token-1', label: 'competition-share-link' },
  { path: '/bookings', label: 'bookings (placeholder)' },
  { path: '/profile', label: 'profile' },
]

describe('WCAG 2.2 AA automated pass (axe-core) — every registered route', () => {
  it('covers every route currently in router/index.ts', () => {
    // Guards ROUTES_UNDER_TEST itself against silently going stale — a new
    // route added to the app without a corresponding entry here would
    // otherwise never get swept, defeating this file's whole point.
    // `/` only redirects, so it isn't a screen of its own to check.
    const registeredCount = routes.filter((r) => r.path !== '/').length
    expect(ROUTES_UNDER_TEST).toHaveLength(registeredCount)
  })

  it.each(ROUTES_UNDER_TEST)('$path ($label) has no automated a11y violations', async ({ path }) => {
    const wrapper = await mountAppAt(path)
    try {
      await expectNoA11yViolations(wrapper.element)
    } finally {
      wrapper.unmount()
    }
  })
})
