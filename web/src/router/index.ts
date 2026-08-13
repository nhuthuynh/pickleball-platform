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
//     /profile gained a real screen in T10.5 once Identity/Users existed
//     to back it (see that route entry below); /bookings remains a
//     placeholder — no Bookings-history ticket has landed yet.
import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import DiscoverFacilities from '../components/discover/DiscoverFacilities.vue'
import DiscoverGames from '../components/discover-games/DiscoverGames.vue'
import FacilityOnboarding from '../views/FacilityOnboarding.vue'
import GameCreation from '../views/GameCreation.vue'
import GameCheckout from '../views/GameCheckout.vue'
import HostPayments from '../views/HostPayments.vue'
import CompetitionCreation from '../views/CompetitionCreation.vue'
import CompetitionCheckout from '../views/CompetitionCheckout.vue'
import CompetitionManage from '../views/CompetitionManage.vue'
import DiscoverCompetitions from '../components/discover-competitions/DiscoverCompetitions.vue'
import CompetitionLanding from '../views/CompetitionLanding.vue'
import Profile from '../views/Profile.vue'
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
  // `/competitions/:id` (the Player-facing browse + detail, T9.7 below) and
  // `/c/:shareToken` (the shared-link landing, also T9.7) are registered
  // separately. `/competitions/new` and `/competitions/:id` can coexist
  // safely: vue-router ranks a static segment above a dynamic one, so `new`
  // is never captured as an id regardless of registration order.
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
  // T10.6 (closes #96, Payments UI): the online checkout step reached from
  // CompetitionEntryPanel.vue's "Pay online now" button (via
  // DiscoverCompetitions.vue's/CompetitionLanding.vue's router push).
  // `:id` is the Competition id; the CompetitionEntry id (the actual
  // Payment payableId) travels as `?entryId=` — mirrors `/games/:id/checkout`
  // exactly, see CompetitionCheckout.vue's header comment for why a
  // CompetitionEntry id doesn't fit this route's own path shape.
  {
    path: '/competitions/:id/checkout',
    name: 'competition-checkout',
    component: CompetitionCheckout,
    meta: { title: 'Checkout' },
  },
  // T9.7 (Discover & Enter Competitions, Player). `/competitions/:id` is the
  // SAME screen as `/competitions` with an initial selection — the detail is
  // derived from the ListCompetitions response the list already fetched
  // (which is also the only read carrying the server-computed spots_left),
  // so a second route component would only duplicate that fetch. `:id` is
  // mapped onto the component's `competitionId` prop, keeping the screen
  // mountable without a router in component tests.
  {
    path: '/competitions',
    name: 'competitions',
    component: DiscoverCompetitions,
    meta: { title: 'Competitions' },
  },
  {
    path: '/competitions/:id',
    name: 'competition-detail',
    component: DiscoverCompetitions,
    props: (route) => ({ competitionId: String(route.params.id) }),
    meta: { title: 'Competition' },
  },
  // T9.7: the deep-link a Host's shared registration link lands on.
  // Deliberately short and top-level (`/c/…`, not `/competitions/share/…`) —
  // it is pasted into social posts and messages, where length costs. The
  // token is a CAPABILITY, not an identifier: never log it, never send it to
  // analytics, never put it in a page title (see
  // GetCompetitionByShareTokenRequest's doc comment).
  {
    path: '/c/:shareToken',
    name: 'competition-share-link',
    component: CompetitionLanding,
    props: (route) => ({ shareToken: String(route.params.shareToken) }),
    meta: { title: 'Competition invitation' },
  },
  // No ticket owns Bookings-history yet — see file header comment.
  { path: '/bookings', name: 'bookings', component: ComingSoonView, meta: { title: 'Bookings' } },
  // T10.5 (docs/process/t10-sprint-plan.md): real Profile screen (display
  // name read-only, self-reported starting level editable via
  // `UpdateSelfReportedLevel`), replacing the `ComingSoonView` placeholder
  // T8.1 originally wired here before Identity/Users existed.
  { path: '/profile', name: 'profile', component: Profile, meta: { title: 'Profile' } },
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
