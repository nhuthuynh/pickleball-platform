<script setup lang="ts">
// Root layout, bootstrapped minimal by T7.1 (docs/process/t7-sprint-plan.md)
// to prove the design tokens, breakpoint composable, and role-indicator nav
// component work together end to end. T7.5 adds the first real product
// screen (DiscoverFacilities, Player-facing browse/search) as this shell's
// main content; T7.4/T7.6 build Facility onboarding/booking the same way.
import { useBreakpoint } from './composables/useBreakpoint'
import RoleIndicator from './components/RoleIndicator.vue'
import DiscoverFacilities from './components/discover/DiscoverFacilities.vue'

const { breakpoint } = useBreakpoint()
</script>

<template>
  <div class="app-shell" :data-breakpoint="breakpoint">
    <header class="app-shell__header">
      <span class="app-shell__brand">Court&amp;Play</span>
      <!-- Mock/hardcoded role data — see RoleIndicator.vue's own comment
           header for why, and what it must not be used for. -->
      <RoleIndicator />
    </header>

    <main class="app-shell__main">
      <!-- T7.5: Discover & browse facilities/courts (Player-facing,
           read-only). See docs/process/t7-sprint-plan.md's T7.5 section
           and src/components/discover/DiscoverFacilities.vue. -->
      <DiscoverFacilities />
    </main>
  </div>
</template>

<style scoped>
.app-shell {
  min-height: 100vh;
  background: var(--paper);
  color: var(--ink);
  font-family: var(--font-family-ui);
  display: flex;
  flex-direction: column;
}

.app-shell__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 1rem;
  background: var(--paper-raised);
  border-bottom: 1px solid var(--hs-border);
}

.app-shell__brand {
  font-weight: 700;
  font-size: var(--font-size-lg);
  color: var(--court);
}

.app-shell__main {
  flex: 1;
  padding: 1rem;
}

/* Single-column stacked layout on iPhone (<600px), per the external
   handoff's Platform Notes. */
@media (max-width: 599px) {
  .app-shell__header {
    flex-direction: column;
    align-items: flex-start;
  }
}

/* Persistent sidebar-shaped header on web (>=1280px) — a full sidebar nav
   is a future ticket's job; this just widens the header's breathing room
   per the reviewed multi-column web layout. */
@media (min-width: 1280px) {
  .app-shell__main {
    padding: 2rem;
    max-width: 1280px;
    margin: 0 auto;
    width: 100%;
  }
}
</style>
