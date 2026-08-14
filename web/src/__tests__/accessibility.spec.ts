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
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
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

// ── T12.5: the KEYBOARD half of the WCAG pass, for T11's three new screens ──
//
// T11.7's automated sweep above ran in T11 Wave 1; T11.3 and T11.6 merged in
// Waves 3 and 4. The sweep covers their routes (the path-identity guard at the
// top of this file makes that structural, not a claim), but the MANUAL half —
// keyboard traversal, focus visibility — never ran against them, because they
// did not exist yet.
//
// This block is the reproducible part of that manual pass. It does not replace
// a real keyboard-and-screen-reader session (see the PR's checked-negatives
// list and the issue filed for the part still owed); what it does is pin shut
// the failure modes a later edit could silently reintroduce, stated as
// properties over the three screens rather than as one-off spot checks.
//
// flagshipFlows.spec.ts asserts the first of these for the two flagship flows
// by recording that a grep "found none" at review time. A grep at review time
// protects the revision it was run against and nothing after it, so here the
// same check is executable instead.
const T11_NEW_SCREENS = [
  { file: 'FacilityDiscounts.vue', route: '/facilities/:facilityId/discounts', ticket: 'T11.3' },
  { file: 'FacilityRentalRequests.vue', route: '/facilities/:facilityId/rental-requests', ticket: 'T11.6' },
  { file: 'ClubRentals.vue', route: '/clubs/rentals', ticket: 'T11.6' },
] as const

function sourceOf(file: string): string {
  // Resolved from the Vitest root (web/), not from `import.meta.url`: under
  // Vitest a spec's `import.meta.url` is the transformed module's
  // root-relative URL, so `new URL('../views/…')` yields `/src/views/…` and
  // misses the project directory entirely.
  return readFileSync(resolve(process.cwd(), 'src/views', file), 'utf8')
}

/** The template half of a single-file component — style/script blocks carry
 * neither click handlers nor tabindex, and prose comments in them would
 * otherwise produce false matches. */
function templateOf(file: string): string {
  const source = sourceOf(file)
  const start = source.indexOf('<template>')
  const end = source.lastIndexOf('</template>')
  expect(start, `${file} has no <template> block`).toBeGreaterThan(-1)
  return source.slice(start, end)
}

describe('WCAG 2.1.1 Keyboard / 2.4.7 Focus Visible — T11.3 + T11.6 screens', () => {
  // 2.1.1: a `<div @click>` is unreachable and unoperable by keyboard. Native
  // interactive elements (or, at minimum, an explicit role plus a tabindex)
  // are the only shapes that are not. Asserted over the whole template rather
  // than against a list of the controls that happen to exist today.
  it.each(T11_NEW_SCREENS)(
    '$file ($ticket): every click handler is on a natively keyboard-operable element',
    ({ file }) => {
      const template = templateOf(file)

      // Every element that opens a tag and carries a click handler before that
      // tag closes, captured together with its tag name.
      const clickable = [...template.matchAll(/<([a-zA-Z][\w-]*)\b[^>]*?@click[^>]*>/g)]
      expect(clickable.length, `${file}: expected at least one interactive control`).toBeGreaterThan(0)

      const NATIVE_INTERACTIVE = new Set(['button', 'a', 'input', 'select', 'textarea', 'summary'])
      for (const [tagMarkup, tag] of clickable) {
        const isNative = NATIVE_INTERACTIVE.has(tag!.toLowerCase())
        // The escape hatch a non-native control would need to be operable at
        // all: an explicit role AND a tabindex placing it in the tab order.
        const isPolyfilled = /\brole=/.test(tagMarkup) && /\btabindex=/.test(tagMarkup)
        expect(
          isNative || isPolyfilled,
          `${file}: <${tag}> has @click but is neither natively keyboard-operable nor given role+tabindex`,
        ).toBe(true)
      }
    },
  )

  // 2.4.7: none of these three screens declared a focus indicator of its own
  // before T12.5 — each inherited whatever ring the user agent draws, whose
  // contrast is not guaranteed against a `--court`-filled button or a tinted
  // panel. This asserts the indicator is DECLARED. It deliberately does NOT
  // claim the indicator's rendered contrast has been measured: jsdom applies
  // no scoped styles and implements no :focus-visible matching, so that stays
  // a checked negative in the PR rather than something this file proves.
  it.each(T11_NEW_SCREENS)('$file ($ticket): declares an explicit focus-visible indicator', ({ file }) => {
    const source = sourceOf(file)
    const styles = source.slice(source.indexOf('<style'))

    expect(styles, `${file}: no :focus-visible rule`).toMatch(/:focus-visible/)
    // An indicator that only changed colour would fail 1.4.1; the convention
    // this codebase already uses (CompetitionLanding.vue) is an outline.
    expect(styles, `${file}: :focus-visible rule draws no outline`).toMatch(
      /:focus-visible[^{]*\{[^}]*outline:[^};]*(solid|auto)/,
    )
  })

  // A positive tabindex re-orders the tab sequence away from DOM order and is
  // the classic way a "sensible tab order" quietly stops being one. `-1`
  // (programmatic focus — e.g. the confirm panel T12.5 adds) is fine, `0` is
  // fine, anything above 0 is not.
  it.each(T11_NEW_SCREENS)('$file ($ticket): uses no positive tabindex', ({ file }) => {
    const positive = [...templateOf(file).matchAll(/tabindex="(\d+)"/g)].filter(
      ([, value]) => Number(value) > 0,
    )
    expect(positive.map(([match]) => match), `${file}: positive tabindex re-orders the tab sequence`).toEqual([])
  })
})
