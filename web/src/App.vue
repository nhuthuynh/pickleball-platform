<script setup lang="ts">
// Root layout, bootstrapped by T7.1 (docs/process/t7-sprint-plan.md) as a
// screen-free shell. T7.4 (this change) is the first product screen to
// build on it: Facility onboarding (Owner/Host). No routing library is
// added yet — there is exactly one screen and no auth to gate it behind,
// so this mounts FacilityOnboarding directly rather than introducing
// vue-router ahead of an actual second-screen need (T7.5/T7.6 build the
// next ones; add real routing then, not speculatively here).
import { useBreakpoint } from './composables/useBreakpoint'
import RoleIndicator from './components/RoleIndicator.vue'
import FacilityOnboarding from './views/FacilityOnboarding.vue'

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
      <FacilityOnboarding />
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
