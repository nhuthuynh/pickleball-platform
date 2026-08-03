# web

The Vue 3 web client for the Pickleball Platform. Bootstrapped by T7.1
(`docs/process/t7-sprint-plan.md`) — see that ticket and
`docs/design/handoff-2026-08/README.md` / `docs/design/v1-review-round-10-final.md`
for the design-token/breakpoint provenance. **No product screens ship in
this ticket** — this is scaffolding (design tokens, breakpoints, a typed
API client generator, and a placeholder role-indicator nav) for T7.4-T7.6
to build on.

## Stack

Vue 3 + TypeScript + Vite, scaffolded with `npm create vue@latest` (no
router/Pinia/ESLint added yet — deliberately minimal until a ticket
actually needs them). Vitest for unit/component tests.

## Node/npm version

Built and verified against **Node v22.22.2 / npm 10.9.7** (this repo's
`web/package.json` `engines` field: `^22.18.0 || >=24.12.0`). Per
`docs/process/t7-sprint-plan.md`'s Toolchain note, this is **not**
guaranteed present in every environment — check `node --version` /
`npm --version` first; install Node LTS if missing.

## Project setup

```sh
npm install
```

### Dev server

```sh
npm run dev
```

### Type-check, build for production

```sh
npm run build
```

Runs `vue-tsc --build` (type-check) and `vite build` in parallel. **Note:**
this only succeeds once `npm run generate:client` has produced
`src/api/generated/` (see below) — `src/api/bookingClient.ts` imports types
from there, and the whole point of a typed client is that it doesn't
type-check against a stale or missing schema. This mirrors this repo's own
`internal/gen/**` convention (CLAUDE.md rule 6: generated code isn't hand
edited, and code depending on it just doesn't compile until generation has
run) — not a bug specific to this ticket.

### Tests (Vitest)

```sh
npm run test       # single run (vitest run) — use this in CI
npm run test:unit  # watch mode, for local development
```

Unlike `npm run build`, the test suite **does not** require
`npm run generate:client` to have been run first — `src/api/client.ts`
(the thing under test) is generic over its `Paths` type and the test file
supplies its own small fixture `paths` shape rather than importing the real
generated one, specifically so tests pass on a fresh clone. See
`src/api/__tests__/client.spec.ts`'s file comment for why.

Current coverage from this ticket:
- `src/api/__tests__/client.spec.ts` — the typed-client wrapper (base URL
  default/override, error surfacing).
- `src/composables/__tests__/useBreakpoint.spec.ts` — breakpoint
  resolution at each boundary, live `matchMedia` change reactions, and
  listener cleanup on scope dispose.
- `src/components/__tests__/RoleIndicator.spec.ts` — the placeholder role
  nav's mock-data rendering and role-switching.

Added by T7.5 (Discover & browse facilities/courts, Player-facing —
`src/components/discover/DiscoverFacilities.vue`, mounted as `App.vue`'s
main content):
- `src/models/__tests__/facility.spec.ts` — the API-response-to-view-model
  mapping, including the regression test that camera-link data present on
  the raw API response never survives the mapping.
- `src/composables/__tests__/useFacilityList.spec.ts` /
  `useFacilityDetail.spec.ts` — loading/empty/error state transitions
  against an injectable fake client (network-unreachable and non-2xx
  cases both covered).
- `src/components/discover/__tests__/FacilityListPanel.spec.ts` /
  `FacilityDetailPanel.spec.ts` — the empty/loading/error UI for each
  panel (incl. the zero-courts empty state), plus a second,
  render-output-level camera-link assertion.
- `src/components/discover/__tests__/DiscoverFacilities.spec.ts` — the
  end-to-end integration test T7.5 specifically asks for: a mocked API
  response that includes camera-link data, asserted to never appear
  anywhere in this screen's rendered output; plus the iPhone
  (list-then-detail-on-selection) vs. iPad/web
  (always-two-column-with-placeholder) layout behavior.

**Known gap, not fixed by this ticket:** the merged Facilities API (T7.3)
has `AddCourt` to create a Court but no endpoint that returns a Facility's
Courts (`GetFacility`/`ListFacilities`'s `Facility` message has no
`courts` field — confirmed against the generated OpenAPI types, not
assumed). `models/facility.ts`'s `mapToFacilityDetail` therefore always
maps `courts: []`, so every facility shows the "zero courts" empty state
today regardless of what a Facility Owner has actually added via
`AddCourt`. The view model and components already have a real `courts`
list end to end (`FacilityCourt[]`, rendered by
`FacilityDetailPanel.vue`), so a follow-up ticket that adds a
courts-listing capability to the Facilities API only needs to change the
one line in `mapToFacilityDetail` that currently hardcodes `[]` — see that
function's doc comment.

**`facilitiesClient.ts`:** no `facilitiesClient.ts`-style wrapper existed
on the base branch (`claude/go-backend-pickleball-7up34j`) when this
ticket was implemented, so `src/api/facilitiesClient.ts` was written fresh
here, mirroring `bookingClient.ts`'s pattern exactly. T7.4 (Facility
onboarding UI), developed concurrently on a separate branch, may add its
own — if so, reconciling the two is a later ticket/PR-review concern, not
blocking for this one (per this ticket's own instructions).

## Typed client generation

```sh
npm run generate:client
```

Two steps, both run by `scripts/generate-client.mjs`:

1. **`make generate`** at the repo root (buf + sqlc → `internal/gen/` and
   `openapi/`, per `CLAUDE.md`). Needs buf, sqlc, and the four
   `protoc-gen-*` plugins CLAUDE.md's Gotchas section names, all on
   `$(go env GOPATH)/bin` — see that section (and
   `docs/process/t7-sprint-plan.md`'s Toolchain note) if this fails in a
   fresh environment. **Verified working in this ticket's authoring
   environment**: buf/sqlc were not preinstalled, but `go install`-ing
   them (per CLAUDE.md's own documented gotcha) and running `make
   generate` succeeded cleanly, producing
   `openapi/pickleball/{booking,payments,socialplay}/v1/*.swagger.json`
   (filenames confirmed by inspection, not guessed) plus `internal/gen/**`.
2. For each bounded context's `openapi/pickleball/<context>/v1/*.swagger.json`,
   convert it to OpenAPI 3.0 and generate TypeScript types into
   `src/api/generated/<context>.d.ts`.

### Why `openapi-typescript` + a thin fetch wrapper (not `openapi-typescript-codegen`)

`openapi-typescript` was chosen over `openapi-typescript-codegen` because
it only emits **types** (`paths`/`components`), not a generated runtime
client class per operation — pairing it with `openapi-fetch` (a small,
already-typed fetch wrapper library) plus this repo's own hand-written
`src/api/client.ts` keeps the actual HTTP call sites thin, inspectable,
and under our own error-handling/retry conventions instead of inside
generated code, while still getting full request/response typing from the
schema. This also keeps the *only* generated artifact to
`src/api/generated/*.d.ts` (pure type declarations, trivially safe to
regenerate and gitignore) rather than generated runtime `.ts` service
classes that would need their own regeneration-vs-hand-edit discipline.

### Real-world wrinkle: Swagger 2.0 vs. OpenAPI 3.x

`protoc-gen-openapiv2` (the plugin `buf.gen.yaml` already uses, per T4)
emits **Swagger 2.0**, not OpenAPI 3.x — confirmed by actually running
`openapi-typescript` against the raw output, which fails fast with
`Unsupported Swagger version: 2.x. Use OpenAPI 3.x instead.` (openapi-typescript
7.x dropped 2.0 support). `scripts/generate-client.mjs` pipes each
swagger.json through `swagger2openapi` (an established, actively
maintained converter) to upconvert to OpenAPI 3.0 in memory before handing
it to `openapi-typescript` — the intermediate `.{context}.openapi3.json`
file is written under `src/api/generated/` and deleted immediately after
use. This is a real toolchain gap this ticket found and fixed, not a
hypothetical one flagged for later.

### Output: `src/api/generated/` is gitignored

Mirrors `/internal/gen/`'s own convention (`.gitignore` — see the "Node.js"
section added by this ticket, and CLAUDE.md rule 6): generated code is
never hand-edited and never committed, only produced locally (or in CI) by
running the generator. `src/api/bookingClient.ts` is the hand-written,
committed wrapper that consumes `src/api/generated/booking.d.ts`'s types —
add `paymentsClient.ts`/`socialplayClient.ts` the same way once a ticket
actually needs them (the swagger/types for both already exist after
`make generate`; only the thin wrapper file is missing, deliberately, to
avoid building client surface nothing calls yet).

### API base URL

`src/api/client.ts`'s `DEFAULT_API_BASE_URL` reads `VITE_API_BASE_URL` (a
Vite env var — see [Vite's env docs](https://vite.dev/guide/env-and-mode)),
falling back to `http://localhost:8080` for local dev against
`cmd/server`.

## Design tokens

`src/styles/tokens.css` — CSS custom properties for color, radius, and
type scale, ported (not re-derived) from:

- `docs/design/v1-review-round-10-final.md` §2a/§2b (and the rounds it
  cites) — the semantic token names/hex values this project's own 10-round
  design review actually WCAG-contrast-verified in both themes
  (`--court`, `--paper`, `--paper-raised`, `--ink-*`, `--pill-*`).
- `docs/design/handoff-2026-08/README.md`'s "Design Tokens" section — the
  external handoff's `oklch(...)` palette, kept as a secondary
  generically-named alias layer (`--hs-*`).

Both light (default) and dark variants are defined, switched either via an
explicit `data-theme="light"|"dark"` attribute on `<html>`/`<body>` or
`prefers-color-scheme` when no explicit override is set. See the file's own
header comment for exactly which values are review-verified vs.
reasonably-extrapolated placeholders (dark-theme parity wasn't fully
spelled out by either source doc) — **per the handoff's own Fidelity
note, this is not a pixel-match of the `.dc.html` artifact.**

## Breakpoints

Three breakpoints, verbatim from the external handoff's Platform Notes
(`docs/design/handoff-2026-08/README.md`, adopted by
`docs/design/v1-external-reference-reconciliation.md`):

| Name   | Range          |
|--------|----------------|
| iPhone | `< 600px`      |
| iPad   | `768–1180px`   |
| Web    | `>= 1280px`    |

Two ways to consume them, kept in sync (see each file's header comment):

- **`src/composables/useBreakpoint.ts`** — a `matchMedia`-backed composable
  (`const { breakpoint, isIphone, isIpad, isWeb } = useBreakpoint()`),
  reactive to resize/orientation-change, unit-tested by mocking
  `window.matchMedia`.
- **`src/styles/breakpoints.css`** — the same values as plain `@media`
  queries, for CSS-only consumers.

## Role indicator (placeholder — read before wiring real data to it)

`src/components/RoleIndicator.vue` implements the T7 kickoff note's
decision #1 (contextual role indicator in the nav, e.g. "Viewing as:
Player ▾"). **Its role list and current-role state are hardcoded mock
data** — there is no Identity/Users/Auth backend in this repo yet, so this
must not be treated as an authorization boundary (same caveat class as
`ActorUserID` elsewhere in this codebase). See the component's own header
comment before wiring it to a real account/session store.

## Discover & browse facilities/courts (T7.5, Player-facing, read-only)

`src/components/discover/DiscoverFacilities.vue` — the first real product
screen mounted into `App.vue`. A read-only consumer of the Facilities
API's `ListFacilities`/`GetFacility` (T7.3): a facility list/search view
(name filter) plus a facility detail view (description, photos, address,
courts). No routing library yet (per "What this ticket does NOT do"
below) — this is the app's sole screen for now, T7.6 adds booking against
it.

- `src/api/facilitiesClient.ts` — thin typed client, mirrors
  `bookingClient.ts`.
- `src/models/facility.ts` — API-response-to-view-model mapping.
  **Camera links are deliberately never mapped onto the view models here**
  (Facility-owned camera links must not appear on this player-facing
  screen, T7.5's requirement #3) — see the file's header comment for why
  that's structural (no field to render), not just a template omission.
- `src/composables/useFacilityList.ts` / `useFacilityDetail.ts` — fetch +
  loading/error state, with an injectable client for tests.
- `src/components/discover/FacilityListPanel.vue` /
  `FacilityDetailPanel.vue` — presentational panels with real
  empty/loading/error UI (not a blank screen) for: no facilities yet, API
  unreachable, and a facility with zero courts.

See "Tests (Vitest)" above for this ticket's test coverage and the
Facilities-API courts-listing gap this screen currently works around.

## Get a quote and book a court (T7.6, Player-facing)

`src/components/booking/CourtBookingFlow.vue` — the first write path in
this app: date/time picker -> `GetQuote` -> review/confirm -> `CreateBooking`
(always `source: SOURCE_INDIVIDUAL`, the only source a bare Player-initiated
booking can legitimately use — `booking.proto`'s `Source` doc comment).

- `src/api/bookingClient.ts` already existed (T7.1); this ticket only added
  a `BookingClient` type export, mirroring `facilitiesClient.ts`'s
  `FacilitiesClient`, so composables/components can declare an
  injectable-for-tests `client?: BookingClient` prop.
- `src/models/booking.ts` — view models + mapping (`Quote`,
  `ConfirmedBooking`), `formatPriceCents` (display formatting only — see
  the file header for why this is not the "client-side price computation"
  the ticket forbids), and `computeNextAvailableSlots`, the client-side
  next-available-slot search used by the double-booking conflict path
  below (`ListCourtBookings`, live since T2, doesn't itself suggest a
  slot).
- `src/composables/useCourtBooking.ts` — owns the quote -> review ->
  book/success state machine and all three network calls (`GetQuote`,
  `CreateBooking`, and — only on a 409 conflict — `ListCourtBookings`). The
  **confirm-step gate** (WCAG 3.3.4 Error Prevention) is enforced here, not
  just in the UI: `confirmBooking()` is a no-op unless a quote has already
  been fetched and reviewed, so `CreateBooking` cannot fire out of order
  regardless of what a calling component does.
  - **Double-booking conflict handling** (WCAG 3.3.3 Error Suggestion): a
    409 (`ErrCourtDoubleBooked`) surfaces the specific message "This slot
    was just booked — pick another time" plus up to 3 computed
    next-available slots, rendered as one-click "try this instead" buttons
    — not just a rejection.
  - **Stale-quote protection:** if more than `QUOTE_STALE_MS` (60s) has
    elapsed since the displayed quote was fetched, confirming re-fetches
    the quote instead of booking, and requires an explicit second confirm
    against the fresh price — never submits a price the server hasn't just
    confirmed.
- `src/components/booking/CourtBookingFlow.vue` — the presentational flow:
  a date/start-time/duration form, a review/confirm screen (court, time,
  price — WCAG 3.3.4), and a success step whose confirmation text is in a
  `role="status"` live region (WCAG 4.1.3). Responsive across all three
  breakpoints (single-column form on iPhone, a widened 3-column field row
  from iPad up).

**How court selection works given the Facilities courts-listing gap:**
`FacilityDetailPanel.vue` (T7.5) always renders `facility.courts` as `[]`
today (see "Known gap" above), so `CourtBookingFlow` is designed to accept
a bare `courtId` prop rather than requiring a real court object.
`FacilityDetailPanel.vue` now emits a `book-court` event two ways: a
"Book this court" button per court in `facility.courts` (works once that
gap is closed and the list is ever non-empty — untested against real data
today, but the wiring is real) and a manual "Court ID" entry field, which
is the only way to reach this flow today and is what this ticket's own
tests exercise end to end. `DiscoverFacilities.vue` owns whether/which
booking flow is currently shown (same split as the rest of this screen:
panels are presentational, `DiscoverFacilities` owns the state), rendering
`CourtBookingFlow` below the browse layout — there's still no routing
library (T7.1's scope note), so this is the simplest way to keep browse and
book both reachable at once.

Test coverage added by this ticket:
- `src/models/__tests__/booking.spec.ts` — quote/booking mapping
  (including the `priceCents` int64-as-string parsing), price formatting,
  and `computeNextAvailableSlots` (overlap skipping, chronological
  ordering, cancelled-booking exclusion, a fully-booked window returning
  no suggestions).
- `src/composables/__tests__/useCourtBooking.spec.ts` — the full state
  machine: quote success/failure, the confirm-step gate (`CreateBooking`
  never called without a reviewed quote), the happy path booking with
  `SOURCE_INDIVIDUAL`, the stale-quote re-fetch-instead-of-book behavior
  (via `vi.useFakeTimers`), the double-booking conflict path (specific
  message + real suggestions, and that the suggestion lookup failing still
  surfaces the conflict message), and a generic non-409 booking failure.
- `src/components/booking/__tests__/CourtBookingFlow.spec.ts` —
  component-level happy path (quote -> confirm -> booked, asserting
  `SOURCE_INDIVIDUAL` and the `role="status"` success region), the
  confirm-step gate at the UI level (no "Confirm booking" control exists
  before a quote is fetched), and the conflict path's rendered message +
  suggestion buttons.
- `src/components/discover/__tests__/FacilityDetailPanel.spec.ts` — new
  cases for both `book-court` emission paths (a listed court's button, and
  the manual entry field, including that a blank manual entry emits
  nothing).
- `src/components/discover/__tests__/DiscoverFacilities.spec.ts` — one
  integration case proving the manual-entry -> `CourtBookingFlow` wiring
  end to end (submit a court id, see the flow appear; close it, see it
  disappear).

## What this ticket does NOT do

Per `docs/process/t7-sprint-plan.md`'s T7.1 scope (and its own "no product
screens" instruction): no Facility onboarding/browse/booking screens (T7.4
-T7.6), no Facilities backend (T7.2/T7.3), no routing/state-management
library, no real authentication. (T7.5, added later, is the first
exception to "no product screens"; T7.6, above, is the second.)

## Client-side routing (T8.1)

`vue-router@4.6.4` (exact-pinned, not `^4.6.4` — see below), added per
`docs/process/t8-sprint-plan.md`'s T8.1: `App.vue`'s former header comment
flagged the T7.4/T7.5 stacked-siblings state (`FacilityOnboarding` +
`DiscoverFacilities` both mounted unconditionally, no routing between them)
as overdue for exactly this fix.

- **Version choice:** `vue-router@5.x` was evaluated first but rejected —
  its `peerDependencies` require `pinia` and `@pinia/colada`
  (`^3.0.4 || ^4.0.2` / `>=0.21.2`), which would silently pull in a state
  library this project has deliberately deferred (see "Stack" above: "no
  router/Pinia/ESLint added yet — deliberately minimal until a ticket
  actually needs them"). `vue-router@4.6.4`'s only peer dependency is
  `vue: ^3.5.0`, matching the scaffolded `vue@^3.5.40` cleanly — installing
  it did **not** need `--legacy-peer-deps` on its own (only the
  pre-existing, unrelated `openapi-typescript`-vs-`typescript` conflict
  `HANDOFF.md` already logged needs that flag for a full `npm install`).
- **`src/router/index.ts`** — the route table:
  - `/facilities` -> `DiscoverFacilities` (T7.5), `/facilities/onboard` ->
    `FacilityOnboarding` (T7.4): existing screens, wired unchanged.
  - `/games`, `/games/new`, `/games/:id/checkout`, `/host/payments`: the
    four new T8.8-T8.10 routes. Each renders
    `src/views/placeholders/ComingSoonView.vue` (one shared component,
    parameterized by the matched route's `meta.title`) in this ticket —
    T8.8/T8.9/T8.10 replace the `component:` entry to build the real
    screen; they don't invent the route.
  - `/bookings`, `/profile`: not named in T8.1's route list, but required
    by its nav requirement (the Bookings/Profile tabs must link somewhere
    real, not be omitted) — same `ComingSoonView` placeholder.
  - `/` redirects to `/facilities`.
- **`App.vue`** is now a shell: persistent header chrome (brand,
  `RoleIndicator`) + `AppNav` + `<RouterView />`. It no longer imports or
  mounts any screen component directly.
- **`src/components/nav/AppNav.vue`** — the minimal nav T8.1 asks for:
  tabs Discover (`/facilities`) / Bookings (`/bookings`) / Games (`/games`)
  / Profile (`/profile`), reusing the external handoff's exact tab set
  even though Bookings/Profile have no real screen yet. Renders as a
  persistent sidebar when `useBreakpoint()` resolves `web` (>=1280px), and
  a fixed bottom tab bar otherwise (iPhone <600px, iPad, and the two named
  in-between gaps — the handoff's Platform Notes only specify the two named
  layouts, so the tab bar is the reasonable default at every width that
  isn't `web`). Takes the same injectable `win` prop pattern as
  `DiscoverFacilities.vue` for testability.

### Tests added by this ticket

- `src/__tests__/App.spec.ts` — the regression this ticket exists to fix:
  mounts `App.vue` behind a real (memory-history) router and asserts
  `/facilities` renders `DiscoverFacilities` with neither
  `FacilityOnboarding` nor a placeholder alongside it, and
  `/facilities/onboard` renders `FacilityOnboarding` alone — written so it
  fails against the old stacked-siblings `App.vue`. Also covers every
  placeholder route rendering `ComingSoonView` with the right title, `/`
  redirecting to `/facilities`, and the header chrome persisting across
  routes. Stubs the ambient `fetch` (rather than injecting a fake client,
  since `App.vue`/the router don't have a test-only seam to thread one
  through) so `DiscoverFacilities`'s mount-time `ListFacilities` call fails
  fast and deterministically instead of hitting the network.
- `src/components/nav/__tests__/AppNav.spec.ts` — the responsive nav
  behavior across `useBreakpoint()`'s three named breakpoints (sidebar at
  web, tab bar at iPhone and iPad), the exact tab set/order, and that
  Bookings/Profile link to their placeholder routes rather than being
  omitted.

Existing T7.4/T7.5 component tests
(`src/views/__tests__/FacilityOnboarding.spec.ts`,
`src/components/discover/__tests__/DiscoverFacilities.spec.ts` and its
siblings) were left unmodified — they already mount their component
standalone (not through `App.vue`/the router), which this ticket's
Instructions explicitly say not to force a rewrite of.
