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

## What this ticket does NOT do

Per `docs/process/t7-sprint-plan.md`'s T7.1 scope (and its own "no product
screens" instruction): no Facility onboarding/browse/booking screens (T7.4
-T7.6), no Facilities backend (T7.2/T7.3), no routing/state-management
library, no real authentication. (T7.5, added later, is the first
exception to "no product screens" — see the section above.)
