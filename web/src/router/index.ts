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
//   - /games/new: T8.8 (Social Game Creation) real screen, wired below.
//   - /games/:id/checkout, /host/payments: the remaining T8.10 routes.
//     Still placeholder components until that ticket swaps its own
//     `component:` entry in.
//   - /bookings, /profile: not named in T8.1's Instructions #1 route list,
//     but required by Instructions #1's nav requirement ("Bookings"/
//     "Profile" tabs must link to a "Coming soon" placeholder route too,
//     not be omitted) — added here so the nav has somewhere real to link.
import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import DiscoverFacilities from '../components/discover/DiscoverFacilities.vue'
import DiscoverGames from '../components/discover-games/DiscoverGames.vue'
import FacilityOnboarding from '../views/FacilityOnboarding.vue'
import GameCreation from '../views/GameCreation.vue'
import GameCheckout from '../views/GameCheckout.vue'
import HostPayments from '../views/HostPayments.vue'
import CompetitionCreation from '../views/CompetitionCreation.vue'
import CompetitionManage from '../views/CompetitionManage.vue'
import ComingSoonView from '../views/placeholders/ComingSoonView.vue'

export const routes: RouteRecordRaw[] = [
  { path: '/', redirect: '/facilities' },
  { path: '/facilities', name: 'facilities', component: DiscoverFacilities },
  { path: '/facilities/onboard', name: 'facilities-onboard', component: FacilityOnboarding },
  // T8.9 (Discover & Join Games) — real screen, replacing T8.1's placeholder.
  { path: '/games', name: 'games', component: DiscoverGames, meta: { title: 'Games' } },
  // T8.8 (Social Game Creation, Host/Owner) — real screen, replacing T8.1's placeholder.
  { path: '/games/new', name: 'games-new', component: GameCreation, meta: { title: 'Create a game' } },
  // T8.10 (Payments UI): the online checkout step reached from
  // GameJoinPanel.vue's "Pay online now" button (via DiscoverGames.vue's
  // router push). `:id` is the Game id; the Registration id (the actual
  // Payment payableId) travels as `?registrationId=` — see
  // GameCheckout.vue's header comment for why a Registration id doesn't
  // fit this route's own path shape.
  {
    path: '/games/:id/checkout',
    name: 'game-checkout',
    component: GameCheckout,
    meta: { title: 'Checkout' },
  },
  // T8.10 (Payments UI, Host pending-cash dashboard).
  { path: '/host/payments', name: 'host-payments', component: HostPayments, meta: { title: 'Payments' } },
  // T9.6 (Competitions, Host): create/advertise, and the roster.
  //
  // These belong to the GAMES area of the nav rather than a sixth
  // top-level tab — see AppNav.vue's own note. `/competitions` and
  // `/competitions/:id` (the Player-facing browse + detail) and
  // `/c/:shareToken` (the shared-link landing) are T9.7's, not registered
  // here. `/competitions/new` and `/competitions/:id` can coexist safely:
  // vue-router ranks a static segment above a dynamic one, so `new` is
  // never captured as an id regardless of registration order.
  {
    path: '/competitions/new',
    name: 'competitions-new',
    component: CompetitionCreation,
    meta: { title: 'Create a competition' },
  },
  {
    path: '/competitions/:id/manage',
    name: 'competition-manage',
    component: CompetitionManage,
    // `props: true` so the view takes the Competition id as a plain prop —
    // which is also what lets its spec mount it without a router.
    props: true,
    meta: { title: 'Manage competition' },
  },
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
