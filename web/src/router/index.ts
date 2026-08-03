// Client-side routing (T8.1, docs/process/t8-sprint-plan.md). Replaces the
// stacked-siblings state App.vue's own former header comment flagged as
// overdue: T7.4 (FacilityOnboarding) and T7.5 (DiscoverFacilities) each
// landed their first product screen with no router in between.
//
// Route table covers every existing and T8-planned top-level screen per
// T8.1's Instructions #1:
//   - /facilities, /facilities/onboard: existing T7.4/T7.5 screens, wired
//     to real components unchanged.
//   - /games: T8.9 (Discover & Join Games) real screen, wired below.
//   - /games/new, /games/:id/checkout, /host/payments: the remaining
//     T8.8/T8.10 routes. Still placeholder components until each of those
//     tickets swaps its own `component:` entry in — this ticket only
//     claims the one route it owns.
//   - /bookings, /profile: not named in T8.1's Instructions #1 route list,
//     but required by Instructions #1's nav requirement ("Bookings"/
//     "Profile" tabs must link to a "Coming soon" placeholder route too,
//     not be omitted) — added here so the nav has somewhere real to link.
import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import DiscoverFacilities from '../components/discover/DiscoverFacilities.vue'
import DiscoverGames from '../components/discover-games/DiscoverGames.vue'
import FacilityOnboarding from '../views/FacilityOnboarding.vue'
import ComingSoonView from '../views/placeholders/ComingSoonView.vue'

export const routes: RouteRecordRaw[] = [
  { path: '/', redirect: '/facilities' },
  { path: '/facilities', name: 'facilities', component: DiscoverFacilities },
  { path: '/facilities/onboard', name: 'facilities-onboard', component: FacilityOnboarding },
  // T8.9 (Discover & Join Games) — real screen, replacing T8.1's placeholder.
  { path: '/games', name: 'games', component: DiscoverGames, meta: { title: 'Games' } },
  // T8.8 (Social Game Creation) replaces this placeholder.
  { path: '/games/new', name: 'games-new', component: ComingSoonView, meta: { title: 'Create a game' } },
  // T8.10 (Payments UI) replaces this placeholder.
  {
    path: '/games/:id/checkout',
    name: 'game-checkout',
    component: ComingSoonView,
    meta: { title: 'Checkout' },
  },
  // T8.10 (Payments UI, Host pending-cash dashboard) replaces this placeholder.
  { path: '/host/payments', name: 'host-payments', component: ComingSoonView, meta: { title: 'Payments' } },
  // No ticket owns these yet (no Bookings-history or Profile/Identity
  // screen exists) — see file header comment.
  { path: '/bookings', name: 'bookings', component: ComingSoonView, meta: { title: 'Bookings' } },
  { path: '/profile', name: 'profile', component: ComingSoonView, meta: { title: 'Profile' } },
]

/**
 * Factory (rather than a single module-level instance) so tests can build
 * their own router with an isolated history (`createMemoryHistory`) against
 * the same route table — see router/__tests__/routes.spec.ts and
 * src/__tests__/App.spec.ts.
 */
export function createAppRouter() {
  return createRouter({
    history: createWebHistory(),
    routes,
  })
}
