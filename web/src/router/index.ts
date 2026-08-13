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
import { createRouter, createWebHistory, type Router, type RouteRecordRaw } from 'vue-router'
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
  // meta.title on these two (T11.7 fix): every other route already carried
  // one; these were the only two gaps, silently relying on index.html's
  // static <title> never actually updating on navigation — see this file's
  // new afterEach guard below (WCAG 2.4.2 Page Titled).
  { path: '/facilities', name: 'facilities', component: DiscoverFacilities, meta: { title: 'Facilities' } },
  {
    path: '/facilities/onboard',
    name: 'facilities-onboard',
    component: FacilityOnboarding,
    meta: { title: 'Add a facility' },
  },
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

/** Shown as the fallback/suffix half of every page title — matches the
 * brand text App.vue's own header already renders literally
 * (`.app-shell__brand`), not invented separately here. */
const APP_NAME = 'Court&Play'

/**
 * Sets `document.title` on every navigation from the matched route's
 * `meta.title` (T11.7 fix — WCAG 2.4.2 Page Titled, Level A: an SPA that
 * never updates the browser tab title away from index.html's static
 * fallback fails this criterion on every route after the first, even though
 * `meta.title` has existed on almost every route since T8.1 — it was never
 * actually wired to anything). Exported separately from `createAppRouter`
 * so a caller building its own router with `createMemoryHistory` (this is
 * exactly what src/__tests__/accessibility.spec.ts's route sweep and
 * src/__tests__/App.spec.ts's routing tests both do, rather than going
 * through the factory below) can install the SAME guard the real app runs —
 * a second, parallel title-setting mechanism just for tests would be able to
 * drift from what production actually does.
 */
export function installTitleGuard(router: Router): void {
  router.afterEach((to) => {
    const screenTitle = to.meta.title as string | undefined
    document.title = screenTitle ? `${screenTitle} — ${APP_NAME}` : APP_NAME
  })
}

/**
 * Factory (rather than a single module-level instance) so tests can build
 * their own router with an isolated history (`createMemoryHistory`) against
 * the same route table — see router/__tests__/routes.spec.ts and
 * src/__tests__/App.spec.ts.
 */
export function createAppRouter() {
  const router = createRouter({
    history: createWebHistory(),
    routes,
  })
  installTitleGuard(router)
  return router
}
