<script setup lang="ts">
// Primary navigation (T8.1, docs/process/t8-sprint-plan.md's T8.1
// Instructions #1: "a persistent sidebar on web >=1280px, a bottom tab bar
// on iPhone <600px" per the external handoff's Platform Notes, tabs
// Discover / Bookings / Games / Profile). iPad and the two unnamed
// between-breakpoint gaps (see useBreakpoint.ts) get the tab-bar layout —
// only "web" gets the sidebar — since the handoff only specifies the two
// named layouts and a bottom tab bar degrades reasonably at narrower/
// in-between widths.
//
// "Bookings" and "Profile" have no real screen yet (no Bookings-history or
// Identity/Profile context exists in this repo) — per the ticket's own
// instruction they still appear here, linked to a "Coming soon" placeholder
// route (see src/router/index.ts), so the nav's shape doesn't need rework
// once those screens land.
//
// T8.10 addition: a fifth "Payments" tab (-> /host/payments, the Host
// pending-cash-payments dashboard) with a count badge, per that ticket's
// kickoff note decision #3. Shown unconditionally (not gated on
// `hasHostEvidence`), mirroring Bookings/Profile's own "always show, even
// before every user has the underlying evidence/screen" precedent above —
// same reasoning, so the nav's shape doesn't need rework later. AppNav is
// mounted once, in App.vue's persistent chrome, so it independently
// refreshes the shared `pendingCashPaymentsCount` (state/
// hostPendingPayments.ts) on mount rather than depending on the Host
// having actually visited /host/payments first.
//
// T9.6 addition: Competitions. Deliberately NOT a sixth top-level tab —
// T8.1's design-handoff mapping fixed this nav's tab set, and a Competition
// is a thing a Host organises within the Games area, not a separate
// destination of the same rank as Discover or Profile. So the nav gains a
// SUB-LEVEL under Games instead: a `children` list on the tab, rendered
// while the Games area is the active one (that includes `/competitions/*`,
// which is part of the Games area even though its path differs). Shown in
// both nav variants — sidebar-only would leave the screen unreachable on
// iPhone, where the tab bar is the whole navigation.
import { computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useBreakpoint } from '../../composables/useBreakpoint'
import { useHostPayments, MOCK_HOST_ID } from '../../composables/useHostPayments'
import { pendingCashPaymentsCount, setPendingCashPaymentsCount } from '../../state/hostPendingPayments'
import type { SocialPlayClient } from '../../api/socialplayClient'
import type { PaymentsClient } from '../../api/paymentsClient'

interface NavSubLink {
  id: string
  label: string
  to: string
}

interface NavTab {
  id: string
  label: string
  to: string
  /** Screens that belong to this tab's area but aren't the tab itself.
   * Rendered as a sub-level while the area is active — never promoted to a
   * top-level tab. */
  children?: NavSubLink[]
  /** Path prefixes that count as "inside this area" for the purpose of
   * showing `children`. Needed because Competitions lives in the Games area
   * under a different path root. */
  areaPrefixes?: string[]
}

const NAV_TABS: NavTab[] = [
  { id: 'discover', label: 'Discover', to: '/facilities' },
  { id: 'bookings', label: 'Bookings', to: '/bookings' },
  {
    id: 'games',
    label: 'Games',
    to: '/games',
    areaPrefixes: ['/games', '/competitions'],
    children: [{ id: 'competitions-new', label: 'Create a competition', to: '/competitions/new' }],
  },
  { id: 'payments', label: 'Payments', to: '/host/payments' },
  { id: 'profile', label: 'Profile', to: '/profile' },
]

const props = defineProps<{
  /** Injectable for tests; defaults to the ambient `window` (matchMedia) —
   * same pattern as DiscoverFacilities.vue's `win` prop. */
  win?: Pick<Window, 'matchMedia'>
  /** Injectable for tests; defaults to the real socialplayClient. */
  client?: SocialPlayClient
  /** Injectable for tests; defaults to the real paymentsClient. */
  paymentsClient?: PaymentsClient
}>()

const { isWeb } = useBreakpoint(props.win ?? window)

const route = useRoute()

/** Whether `tab`'s area is the one currently being viewed — which is what
 * decides if its sub-links are shown. */
function isAreaActive(tab: NavTab): boolean {
  const prefixes = tab.areaPrefixes ?? [tab.to]
  return prefixes.some((prefix) => route.path === prefix || route.path.startsWith(`${prefix}/`))
}

const { pending, load } = useHostPayments(props.client, props.paymentsClient)

onMounted(() => {
  void load(MOCK_HOST_ID).then(() => setPendingCashPaymentsCount(pending.value.length))
})
</script>

<template>
  <nav
    class="app-nav"
    :class="isWeb ? 'app-nav--sidebar' : 'app-nav--tabbar'"
    :data-nav-variant="isWeb ? 'sidebar' : 'tabbar'"
    aria-label="Primary"
  >
    <template v-for="tab in NAV_TABS" :key="tab.id">
      <RouterLink :to="tab.to" class="app-nav__item" active-class="app-nav__item--active">
        {{ tab.label }}
        <!-- WCAG 1.4.1: the count itself is text (a visible number), never a
             color-only dot — plus visually-hidden context for screen readers. -->
        <span v-if="tab.id === 'payments' && pendingCashPaymentsCount > 0" class="app-nav__badge">
          {{ pendingCashPaymentsCount }}
          <span class="app-nav__badge-sr-text"> pending cash payments</span>
        </span>
      </RouterLink>

      <!-- Sub-level, shown only while this tab's area is active. A distinct
           class from `app-nav__item` on purpose: these are not tabs, and
           the spec's "exactly five top-level tabs" assertion counts
           `.app-nav__item` to prove it. -->
      <RouterLink
        v-for="child in isAreaActive(tab) ? (tab.children ?? []) : []"
        :key="child.id"
        :to="child.to"
        class="app-nav__subitem"
        active-class="app-nav__subitem--active"
      >
        {{ child.label }}
      </RouterLink>
    </template>
  </nav>
</template>

<style scoped>
.app-nav {
  display: flex;
  gap: 0.25rem;
}

.app-nav__item {
  color: var(--ink-soft);
  text-decoration: none;
  font-weight: 600;
  font-size: var(--font-size-sm);
  border-radius: var(--radius-sm);
  padding: 0.5rem 0.75rem;
  min-height: 44px;
  display: flex;
  align-items: center;
}

.app-nav__item--active {
  color: var(--court);
  background: var(--paper-raised);
}

/* Sub-level link (T9.6). Visually subordinate to a tab — indented and
   lighter in the sidebar — but still a >=44px touch target, because it is a
   real destination a finger has to hit on iPad/iPhone. */
.app-nav__subitem {
  color: var(--ink-soft);
  text-decoration: none;
  font-size: var(--font-size-xs);
  border-radius: var(--radius-sm);
  padding: 0.35rem 0.75rem;
  min-height: 44px;
  display: flex;
  align-items: center;
}

.app-nav__subitem--active {
  color: var(--court);
  background: var(--paper-raised);
}

.app-nav__badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 1.25rem;
  height: 1.25rem;
  margin-left: 0.35rem;
  padding: 0 0.35rem;
  border-radius: 999px;
  background: var(--pill-critical-bg);
  color: var(--ink);
  font-size: var(--font-size-xs);
  font-weight: 700;
}

.app-nav__badge-sr-text {
  position: absolute;
  width: 1px;
  height: 1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
}

/* Persistent sidebar (web >=1280px). */
.app-nav--sidebar {
  flex-direction: column;
  width: 200px;
  flex-shrink: 0;
  padding: 1.5rem 0.5rem;
  border-right: 1px solid var(--hs-border);
}

/* In the sidebar the sub-level reads as nested under its tab. */
.app-nav--sidebar .app-nav__subitem {
  margin-left: 0.75rem;
  border-left: 2px solid var(--hs-border);
}

/* Bottom tab bar (iPhone <600px, and the narrower/in-between widths — see
   file header comment). */
.app-nav--tabbar {
  position: fixed;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 10;
  flex-direction: row;
  justify-content: space-around;
  padding: 0.5rem;
  background: var(--paper-raised);
  border-top: 1px solid var(--hs-border);
}

.app-nav--tabbar .app-nav__item {
  flex-direction: column;
  flex: 1;
  justify-content: center;
  text-align: center;
}

/* In the tab bar the sub-level gets its own full-width row ABOVE the tabs
   (order:-1 + a 100% basis against the wrapping row), so it reads as a
   context strip for the active area rather than as a sixth tab crammed in
   beside the five. */
.app-nav--tabbar {
  flex-wrap: wrap;
}

.app-nav--tabbar .app-nav__subitem {
  order: -1;
  flex-basis: 100%;
  justify-content: center;
  border-bottom: 1px solid var(--hs-border);
}
</style>
