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
import { useBreakpoint } from '../../composables/useBreakpoint'

interface NavTab {
  id: string
  label: string
  to: string
}

const NAV_TABS: NavTab[] = [
  { id: 'discover', label: 'Discover', to: '/facilities' },
  { id: 'bookings', label: 'Bookings', to: '/bookings' },
  { id: 'games', label: 'Games', to: '/games' },
  { id: 'profile', label: 'Profile', to: '/profile' },
]

const props = defineProps<{
  /** Injectable for tests; defaults to the ambient `window` (matchMedia) —
   * same pattern as DiscoverFacilities.vue's `win` prop. */
  win?: Pick<Window, 'matchMedia'>
}>()

const { isWeb } = useBreakpoint(props.win ?? window)
</script>

<template>
  <nav
    class="app-nav"
    :class="isWeb ? 'app-nav--sidebar' : 'app-nav--tabbar'"
    :data-nav-variant="isWeb ? 'sidebar' : 'tabbar'"
    aria-label="Primary"
  >
    <RouterLink
      v-for="tab in NAV_TABS"
      :key="tab.id"
      :to="tab.to"
      class="app-nav__item"
      active-class="app-nav__item--active"
    >
      {{ tab.label }}
    </RouterLink>
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

/* Persistent sidebar (web >=1280px). */
.app-nav--sidebar {
  flex-direction: column;
  width: 200px;
  flex-shrink: 0;
  padding: 1.5rem 0.5rem;
  border-right: 1px solid var(--hs-border);
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
</style>
