<script setup lang="ts">
// Root layout. Bootstrapped by T7.1 (docs/process/t7-sprint-plan.md) as a
// screen-free shell; T7.4 and T7.5 each then landed their first product
// screen (FacilityOnboarding, DiscoverFacilities) directly here as stacked
// block siblings, with no routing library in between (see git history for
// the shape this file had before T8.1). T8.1
// (docs/process/t8-sprint-plan.md) replaces that with real routing: this
// file is now purely a shell — persistent chrome (header, RoleIndicator,
// AppNav) plus a <RouterView /> for whichever screen the current route
// matches. Screens themselves live under src/router/index.ts's route
// table, not here.
import { RouterLink, RouterView } from 'vue-router'
import RoleIndicator from './components/RoleIndicator.vue'
import { authRejected, clearAuthRejected, isSignedIn } from './state/authSession'
import AppNav from './components/nav/AppNav.vue'
</script>

<template>
  <div class="app-shell">
    <header class="app-shell__header">
      <span class="app-shell__brand">Court&amp;Play</span>
      <!-- Mock/hardcoded role data — see RoleIndicator.vue's own comment
           header for why, and what it must not be used for. -->
      <RoleIndicator />
      <!-- T65.1 — one place that says "sign in", rather than twelve write
           call sites each inventing a message. `authRejected` is set by the
           API client's middleware on a 401 and is deliberately distinct from
           "no token present": a rejected token is the case that needs
           explaining. -->
      <RouterLink v-if="!isSignedIn()" class="app-shell__sign-in" to="/sign-in">Sign in</RouterLink>
    </header>

    <p v-if="authRejected" class="app-shell__auth-alert" role="alert">
      That request was rejected because you are not signed in, or your token
      has expired. <RouterLink to="/sign-in" @click="clearAuthRejected()">Sign in again</RouterLink>.
    </p>

    <div class="app-shell__body">
      <AppNav />

      <main class="app-shell__main">
        <RouterView />
      </main>
    </div>
  </div>
</template>

<style scoped>
.app-shell__sign-in {
  margin-left: auto;
  min-height: 44px;
  display: inline-flex;
  align-items: center;
  padding-inline: 0.75rem;
}
.app-shell__sign-in:focus-visible,
.app-shell__auth-alert a:focus-visible {
  outline: 3px solid var(--court, #2f855a);
  outline-offset: 2px;
}
.app-shell__auth-alert {
  margin: 0;
  padding: 0.75rem 1rem;
  background: var(--danger-surface, #fef2f2);
  color: var(--danger, #b91c1c);
  font-weight: 600;
}
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

.app-shell__body {
  flex: 1;
  display: flex;
}

.app-shell__main {
  flex: 1;
  padding: 1rem;
  min-width: 0;
}

/* Single-column stacked layout on iPhone (<600px), per the external
   handoff's Platform Notes. AppNav renders as a fixed bottom tab bar at
   this width (see AppNav.vue), so leave room for it. */
@media (max-width: 599px) {
  .app-shell__header {
    flex-direction: column;
    align-items: flex-start;
  }

  .app-shell__main {
    padding-bottom: 4.5rem;
  }
}

/* Persistent sidebar-shaped web layout (>=1280px) — AppNav renders as a
   real sidebar at this width (see AppNav.vue); this just widens the main
   column's breathing room per the reviewed multi-column web layout. */
@media (min-width: 1280px) {
  .app-shell__main {
    padding: 2rem;
  }
}
</style>
