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
import { routes, installTitleGuard } from '../router'
import { expectNoA11yViolations, pageStructureCounts } from '../test-support/axe'

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

  // jsdom's test document is never actually loaded from index.html — vitest
  // hands each test file a bare, empty document, so `<html lang="en">`
  // (WCAG 3.1.1) never gets here the way a real browser navigation would
  // provide it automatically. Set it once, by hand, to the SAME value
  // index.html itself carries (kept in sync deliberately, not duplicated by
  // coincidence) so this sweep models what a real page load actually looks
  // like instead of either (a) failing on something no source change here
  // could ever fix, or (b) silently disabling the axe rule that would catch
  // a real regression. `document.title` needs no equivalent hand-set-up:
  // installTitleGuard (below, per mountAppAt) sets it for real, from the
  // same router guard production actually runs.
  document.documentElement.lang = 'en'
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
  installTitleGuard(router)
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
  { path: '/facilities/facility-1/discounts', label: 'facility-discounts' },
  // T11.6's two screens. Both build to this bar from the start (per the
  // sprint plan's A6 note that T11.7 does not re-audit them), and both are
  // swept here for the same reason every other route is: this list is
  // path-identity-checked against router/index.ts by the test just below, so
  // a new route cannot land without landing here too.
  { path: '/facilities/facility-1/rental-requests', label: 'facility-rental-requests' },
  { path: '/clubs/rentals', label: 'club-rentals' },
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
  it('covers every route currently in router/index.ts, by path identity, not just count', () => {
    // Guards ROUTES_UNDER_TEST itself against silently going stale. A
    // COUNT-only check (`toHaveLength`) is not enough: a 1-for-1 path
    // rename (e.g. /games/new -> /games/create) leaves the count unchanged
    // — the guard would still pass while ROUTES_UNDER_TEST keeps sweeping
    // the old, now-dead URL (which resolves to nothing real and reports
    // zero violations) and the real new route is never swept at all. This
    // resolves each ROUTES_UNDER_TEST entry through the real router and
    // compares the resulting route PATTERNS (not raw counts) against
    // router/index.ts's own registered patterns — a rename, a removal, or
    // an addition all change that comparison, not just an addition.
    // `/` only redirects, so it isn't a screen of its own to check.
    const registeredPaths = routes
      .filter((r) => r.path !== '/')
      .map((r) => r.path)
      .sort()

    const router = createRouter({ history: createMemoryHistory(), routes })
    const coveredPaths = ROUTES_UNDER_TEST.map(({ path }) => {
      const [firstMatch] = router.resolve(path).matched
      // A path that no longer resolves to any registered route (e.g. after
      // a rename this file wasn't updated for) gets a specific, readable
      // failure here rather than a confusing `undefined` further down —
      // and satisfies TypeScript's narrowing on the destructured element
      // too (a separate `matched.length` assertion doesn't narrow
      // `matched[0]`'s possibly-undefined type the way this does).
      expect(firstMatch, `'${path}' did not resolve to any registered route`).toBeDefined()
      return firstMatch!.path
    }).sort()

    expect(coveredPaths).toEqual(registeredPaths)
  })

  it.each(ROUTES_UNDER_TEST)('$path ($label) has no automated a11y violations', async ({ path }) => {
    const wrapper = await mountAppAt(path)
    try {
      // Scoped to `document` (the WHOLE page), not `wrapper.element` (just
      // the mounted subtree) — deliberately, not incidentally. Several
      // page-level axe rules only meaningfully evaluate when the run
      // context includes the actual `<html>` root (e.g. `region`'s `body *`
      // selector) — scoping to `wrapper.element` leaves them inapplicable
      // rather than actually checked.
      await expectNoA11yViolations(document)

      // page-has-heading-one / landmark-one-main direct equivalents (NOT
      // delegated to axe — see test-support/axe.ts's header comment for why
      // those two specific axe rules resolve `incomplete` under jsdom
      // regardless of actual page content, confirmed empirically, and so
      // are disabled there rather than silently providing zero real
      // coverage). This is what actually caught several routes rendering
      // with no `<h1>` at all — fixed in this same PR revision in
      // DiscoverFacilities.vue/DiscoverGames.vue/DiscoverCompetitions.vue/
      // FacilityOnboarding.vue/GameCreation.vue/CompetitionCreation.vue/
      // CompetitionManage.vue.
      const { headingCount, mainCount } = pageStructureCounts()
      expect(headingCount, `'${path}': expected exactly one top-level heading, found ${headingCount}`).toBe(1)
      expect(mainCount, `'${path}': expected exactly one <main> landmark, found ${mainCount}`).toBe(1)
    } finally {
      wrapper.unmount()
    }
  })
})
