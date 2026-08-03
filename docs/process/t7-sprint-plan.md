# T7 Sprint Plan — Web UI foundation + Facilities context

Produced by Ceremony 1 (Backlog refinement) + Ceremony 2 (Sprint planning)
per `docs/process/sprint-process.md`, played jointly by **Product Manager**
(handbook B2) and **Principal Engineer** (handbook B1 + `docs/roles/
principal-engineer.md`). Unlike T5/T6, this sprint is not inherited
directly from a single `HANDOFF.md` task — it is the first sprint of a new
phase (T7+) whose scope comes from the user's own request ("Improve UI/UX,
add mobile design, and integrate with social media platforms... to allow
people to register and join games") reconciled against the 10-round design
review (`docs/design/v1-system-design.md`, `v1-review-round-10-final.md`),
the external design handoff (`docs/design/handoff-2026-08/`), and its
reconciliation note (`docs/design/v1-external-reference-reconciliation.md`).
Part A below is the multi-sprint roadmap this ticket list is the first slice
of; Part B is T7 itself, fully ticketed.

---

# Part A — Roadmap (T7–T9), not fully ticketed beyond T7

## Starting facts that shaped every sequencing call below

Confirmed by reading the actual code, not assumed:

- **There is zero frontend code in this repo.** No `web/`, no `package.json`,
  no `node_modules` anywhere at the repo root. `node` v22.22.2 / `npm`
  10.9.7 **are** installed in this authoring environment, but that is not
  guaranteed for whatever environment a background implementer agent runs
  in next — see T7.1's kickoff note below, same caveat class as T4's
  buf/sqlc gotcha, just with the opposite finding (present here, unverified
  elsewhere).
- **Courts have no owning Facility today.** `db/migrations/0001_init.sql`
  defines `courts` with exactly `id` and `name` — no description, photos,
  address, owner, or camera-link field anywhere in the schema, and no
  `facilities` table exists at all. `booking.proto`'s `Booking.court_id` is
  a bare string. This confirms the task brief's suspicion directly: **a
  "browse and book a court" UI cannot be built as a real flow today** — it
  would either have to fake facility data client-side (throwaway, and
  actively misleading once real data exists) or the backend needs a
  Facilities context first. This is the single fact that most shapes T7's
  scope.
- **Competitions doesn't exist** (`agent-operating-handbook.md` A1 lists it
  as a bounded context; no `internal/competitions/` directory exists).
  Advertising a competition via social media (requirement #9) has nothing
  to advertise yet.
- **Identity/Users doesn't exist as a real context either.** Every actor
  check shipped so far (T5.5, T6.3, T6.7) is scoped against a
  request-supplied `ActorUserID`/`actor_player_id` string, explicitly
  documented as *not* a real authorization boundary pending JWT/Auth0. This
  matters directly for T7: any "role-switching UX" the frontend builds this
  phase is UI-layer only — there is no backend account/role model yet to
  drive it authoritatively. Flagged, not silently assumed away.
- **Payments is wired to `registration`-payable actions, not plain
  `booking`-payable ones** (T6.3's kickoff note scoped offline recording to
  bookings that have an owning Host/Game — "a direct court hire...out of
  scope"). A bare `CreateBooking` call today has no payment step at all.
  This means a first "browse and book a court" UI slice is legitimately a
  free/unpaid reservation flow unless Payments' `booking`-payable path is
  extended — not done in this roadmap, logged as a gap for T8/T9.

## Sprint boundaries and reasoning

**T7 — Web client foundation + Facilities context (this document, fully
ticketed in Part B).** Bootstraps the Vue 3 app (scaffold, design tokens
from the reviewed mockup + the external handoff's breakpoints, typed OpenAPI
client generation) and builds the Facilities bounded context
(`internal/facilities/{domain,app,port,adapter}` +
`proto/pickleball/facilities/v1`) far enough to support **read** flows
(browse/search/view a facility and its courts) plus **facility onboarding**
(create a facility, add courts) — the two things the design review's Flow 1
and Flow 4 need to be real rather than mocked. Ends with an actual,
non-fake "discover a facility → view its courts → get a quote → book"
path running against the real Booking API, wired end to end in the browser.
Does **not** touch Social Play/Payments UI, Competitions, or any social
platform integration — those need Facilities to exist first (a Game is
created *at* a Facility; a Discover screen's game cards need a real court to
point at), so building them before Facilities would mean building against
fakes now and rewiring against real data next sprint. This is the PE-driven
sequencing call: **spine-first for the frontend, mirroring the same
"Booking, Pricing, Facilities are the shared spine" principle
`agent-operating-handbook.md` A1 already states for the backend** — it was
true for backend build order in T0–T6 and the task brief's own framing
("Facilities is almost certainly a hard prerequisite") independently
rediscovers the same conclusion for the frontend.

**T8 — Social Play + Payments UI, Competitions context, social/growth
integration.** Once Facilities is real, this sprint builds: (a) the
Social Game creation flow (Flow 3) and Discover & Join Games flow (Flow 4)
against real `internal/socialplay` APIs, now pointing at real Facilities/
Courts instead of a fake; (b) the Payments UI (online Stripe-stub checkout,
cash "pending" surfacing per open question #3 below); (c) a new
**Competitions** bounded context (`internal/competitions/{domain,app,port,
adapter}`, mirroring Social Play's shape closely — a `Competition`
reserves courts the same way a `Game` does, per the glossary) plus its
Discover/Advertise UI (Flow 5); (d) social-account-linking (store an OAuth
token per host per platform — a real, scoped piece of work) and
shareable-registration-links (the in-app-RSVP pattern already locked by
spec §7 and the design review's §4, not reply-scraping); (e) a scoped
evaluation/spike of the WhatsApp/Zalo owned-channel-bot pattern (see
"Social platform integration" analysis below) — likely a spike ticket that
produces a recommendation and a thin proof-of-concept for **one** platform,
not two production integrations in one sprint, given everything else T8
already carries.

**T9 — Pricing/discount UI + Club rentals + hardening.** Builds the
`discount_rules` backend concept design-round §3.2 already specified
(currently doesn't exist — `pricing_rules` has a write path only via
migration seed, per `HANDOFF.md` T1's own logged gap) plus its UI (Flow 2),
the Club account type + club recurring-rental request/approval flow
(Flow 7), and a hardening pass: the WCAG 2.2 AA criteria
`docs/requirements/research-accessibility-i18n.md` §1 named as load-bearing
for this app's two flagship flows (3.3.1/3.3.3/3.3.4/4.1.3) audited across
everything T7/T8 shipped, not just designed-in from the start per ticket.
Also where the round-10 design review's one concrete, no-judgment-call
finding (the camera-consent checkbox must not ship pre-checked) gets
verified against whatever T7 actually built, since T7 is the sprint that
first implements that field for real.

**Why not fewer or more sprints.** Two sprints (folding T8 into T7) was
considered and rejected: T7 alone is already a full sprint's worth of new
*infrastructure* (first frontend ever, first new bounded context since T6,
first typed-client-generation pipeline) before a single Social Play/Payments
screen exists — stacking Competitions and two social-platform integrations
on top in the same sprint repeats the exact "too much new, unproven surface
in one sprint" pattern this project's own process exists to avoid (PE
dossier §3, "innovation tokens" — a new frontend framework instance *and* a
new bounded context *and* new vendor integrations all at once spends more
than one sprint's worth of tokens). Four-plus sprints (splitting T8 further)
was considered and rejected the other direction: Social Play UI and
Payments UI share almost all their real estate (a Game card shows price and
payment method in the same view), so splitting them into separate sprints
would mean revisiting the same screens twice for no architectural reason —
three sprints is the boundary where each one ships a coherent, demoable
slice.

## Native mobile (Swift iOS + Kotlin Android): deliberate deferral, not silently dropped

**PM/PE joint call: web first, native apps are not in this roadmap's T7–T9
window.** Reasoning, and where PM and PE differ on *why*, recorded per
sprint-process.md's "don't manufacture consensus":

- **Shared reasoning (no disagreement here):** `CLAUDE.md`'s own locked
  decision already names Android as a previously-open sub-decision ("D4a...
  Android... is an open sub-decision"), and the design review's §6
  confirms Android is still not named in the user's latest requirements
  list either. iOS is locked (Swift/SwiftUI), Android is not — building
  either now, before a single real API-backed screen exists in *any*
  client, means designing the mobile IA against a UX that hasn't been
  proven against real users yet. `docs/roles/product-engineer.md`'s
  mandate ("can this actually be built and shipped in the time implied?")
  argues directly against three simultaneous client platforms (Vue, Swift,
  Kotlin) when zero exist today.
- **PM's framing:** the user's own request explicitly named "mobile
  design" as in-scope for this phase, and did so before naming social
  integration — treating it as an afterthought risks under-delivering
  against what was actually asked. PM's position: web-first is right for
  *build sequencing*, but this roadmap should say plainly that mobile is a
  T10+ commitment, not an open-ended "maybe," so it isn't quietly dropped
  the way Android sat as "open" for six sprints already.
- **PE's framing:** "add mobile design" is satisfied *this* phase by the
  Vue client's responsive breakpoints (web ≥1280px, iPad 768–1180px,
  iPhone <600px, per the external handoff's Platform Notes, adopted
  verbatim in the reconciliation note) — a fully responsive web app usable
  on an iPhone/iPad browser is a legitimate, shippable reading of "mobile
  design" for a phase that has zero native client infrastructure today, and
  is what the design review's own artifact actually depicts (iPhone/iPad
  wireframes of the *same* Vue-driven design system, not separate native
  screens). Native Swift/Kotlin apps should wait until the Vue app has
  proven the UX patterns and the OpenAPI/gRPC surface is stable enough that
  a native client isn't rebuilt every sprint alongside its web sibling —
  reusing both, per the task brief's own suggested framing.
- **Resolution (not a full disagreement, PM accepted PE's sequencing but
  not PE's exact framing):** native mobile is **explicitly out of scope for
  T7–T9**, revisit at the T9 retro once Facilities+Social Play+Payments+
  Competitions all have real, used Vue UI — at that point the Swift/Kotlin
  question (including whether Android is in scope at all) should be put to
  the user directly rather than re-deferred silently again, since it's now
  been open since before T0.

## Social platform integration: WhatsApp vs. Zalo, researched

The task brief asks specifically whether an **owned-channel-bot** pattern
(a number/account the platform itself controls, parsing a constrained reply
format like `IN 2`) is realistic for both WhatsApp and Zalo, or whether they
differ enough to matter for scoping. `docs/requirements/` has no existing
Zalo research (only WhatsApp/Facebook/X/Instagram were covered, in
`pickleball-platform-spec.md` §7 and the functional-research pass) — the
following is genuine new research for this ceremony, sourced from the
WhatsApp Business Platform's own documented model plus current external
documentation on the Zalo Official Account (OA) API, not from this repo's
prior docs.

**Where they're similar (the owned-channel-bot pattern works for both):**
both platforms let a business register an official presence it controls —
a WhatsApp Business phone number, a Zalo Official Account — receive inbound
messages via a webhook, and reply programmatically. Neither requires
scraping or monitoring public content, so both are consistent with the
spec's existing "channel you control" resolution to the reply-to-register
conflict (§4 of `v1-system-design.md`).

**Where they differ enough to matter for scoping:**
1. **Opt-in model.** WhatsApp Business Cloud API lets a business-initiated
   conversation start once a user has messaged the number (or an approved
   template is used); Zalo's OA API is explicitly **follow-based** — a user
   must *follow* the Official Account before it can message them at all,
   closer to a Facebook Page/subscriber model than to WhatsApp's
   number-to-number model. This changes the UX: a host advertising a game
   via Zalo needs players to already follow (or follow at the moment of
   registering), not just message in.
2. **Approval/verification weight.** Both require business verification,
   but Zalo OA has tiered account levels (trial vs. verified) with
   different rate limits and messaging permissions gating what an
   unverified OA can do — a materially heavier bring-up cost for a first
   integration than WhatsApp's more uniform Cloud API onboarding.
3. **Market scope.** Zalo is a Vietnam-specific platform; WhatsApp is
   global. Building the Zalo integration only pays off if the platform's
   actual user base is Vietnam-concentrated — a product/market question
   this ceremony cannot answer and is not trying to (flagged for PM/PO to
   confirm against real go-to-market plans before T8 commits engineering
   time to it).
4. **Both are genuinely buildable as thin adapters** behind the same
   port shape this project already uses for exactly this situation — see
   T6.2's `port.PaymentProcessor` ACL precedent. A `port.MessagingChannel`
   interface (`SendMessage`, `ParseInboundReply`) with
   `adapter/whatsappbusiness` and `adapter/zaloOA` implementations is the
   right shape *if and when* this gets built — not designed further here,
   this is a T8-scoping input, not a T7 deliverable.

**Recommendation carried into the T8 roadmap line above:** treat this as a
spike ticket first (confirm actual API access/approval feasibility, produce
a `port.MessagingChannel` shape, prototype against **one** platform, not
both) rather than committing to building both integrations as production
features in T8 sight-unseen — the follow-based vs. message-based opt-in
difference alone means the two aren't a single uniform feature.

---

# Part B — T7 Sprint Plan

## Sprint goal

> A Facility and its Courts are real, persisted entities (not bare IDs) —
> an Owner/Host can create a Facility with description, photos, address, and
> an opt-in-consent-gated camera link through a real Vue 3 web app, and a
> Player can browse facilities, view courts, get a live quote, and complete
> a real booking through the same app — reachable end to end in a browser
> against the actual gRPC-gateway REST API, on responsive layouts matching
> the reviewed design system's three breakpoints (web ≥1280px, iPad
> 768–1180px, iPhone <600px).

## Kickoff note

### Scope decisions against the five open UX questions (reconciliation note)

1. **Role-switching UX.** Decided **contextual, with a lightweight
   persistent indicator** — not a hard mode switch, not a per-action modal
   prompt. Concretely for T7: the app shell shows a role indicator in the
   nav ("Viewing as: Player ▾") that only lists roles the signed-in account
   actually holds evidence for (a Facility-owning account sees "Owner" as
   an option once it has created a facility); screens gain
   role-appropriate controls contextually (an Owner viewing their own
   Facility's detail page sees an "Edit" action a Player doesn't). T7 only
   has two roles to actually build this for (Player browsing, Owner/Host
   onboarding a facility) — the pattern is decided now so T8's Host/Game
   controls and Club account type slot into the same mechanism rather than
   inventing a second one. **Caveat, stated plainly per the roadmap's
   starting facts:** there is no real Identity/Users/Auth context yet, so
   this "role indicator" is UI-state only in T7 (which role the current
   browser session is *acting as*, not a server-verified permission) — it
   must not be built as if it's an authorization boundary, same caveat
   class as T5.5/T6.3/T6.7's `ActorUserID` pattern.
2. **Auto-matching transparency.** **Explicitly deferred to T8** — no
   matchmaking or Game-creation UI ships in T7 at all (Social Play has no
   UI yet), so there is nothing for this question to attach to this sprint.
   Not decided, not silently dropped — logged as a T8 planning input.
3. **Cash-payment surfacing prominence.** **Explicitly deferred to
   T8/T9** — T7's booking flow doesn't create a `Payment` at all (per the
   roadmap's starting facts: `booking`-payable Payments aren't wired, only
   `registration`-payable ones are, and Registration/Game UI is T8). Not
   decided this sprint.
4. **Pricing-conflict UI vs. validation error.** **Decided now, for future
   tickets to inherit, but not implemented in T7.** T7's Facility
   onboarding covers identity/description/photos/address/cameras only — it
   does not touch court pricing or discounts (no `discount_rules` backend
   exists yet, HANDOFF.md's T1 gap; building it is T9 scope). The
   reconciliation note's recommendation is formally adopted as the design
   direction for when that UI is built: **prevent the overlap at entry time
   with a validation error, not a runtime conflict-resolution surface** —
   consistent with ADR-0002's "ambiguity is a domain error, not silently
   prioritized" precedent for `pricing_rules`. Written down now so T9
   doesn't re-litigate it.
5. **Club as an explicit fourth account type.** **Adopted as the domain
   model direction**, not built in T7. `Facility`'s owner field and the
   role-indicator mechanism from decision #1 are both designed to
   accommodate a fourth role value without rework (the indicator is already
   "whichever roles this account has evidence for," not a hardcoded
   three-way enum) — but no Club-specific screen, backend field, or
   recurring-rental flow ships this sprint. T9 scope, per the roadmap.

### Scope decisions against the roadmap (Part A)

- **Facilities context is real backend work, not a mock.** PM raised
  whether T7 could ship faster by building the Vue UI against a hand-rolled
  JSON fixture standing in for a Facilities API, deferring the actual
  backend to T8, to get a demoable UI sooner. **PE pushed back and PM
  accepted the correction, recorded as a resolved (not manufactured)
  disagreement:** a fixture-backed UI would need its data-fetching layer
  rewritten the moment T7.3's real API lands, and the reviewed mockup's own
  fields (camera-consent checkbox default state, photo gallery, per-court
  "Priced"/"Needs pricing" status) are exactly the kind of detail that's
  cheap to get right against a real schema now and expensive to discover
  was wrong against a fixture later — this project's own `docs/LESSONS.md`
  already has a "direct-push-for-docs"-shaped lesson about work that felt
  low-risk enough to shortcut turning out not to be. Resolution: T7 builds
  the real Facilities context (T7.2/T7.3) before the UI tickets that
  consume it (T7.4/T7.5/T7.6) — sequenced, not shipped in parallel against
  a fake.
- **`courts` stays a Booking-owned table; Facilities gets a new FK onto
  it, not a competing entity.** PE flagged this as the one architecturally
  load-bearing call in this sprint (PE dossier §3, "classify before you
  deliberate: one-way door or two-way door"): duplicating `Court` as a
  second entity owned by Facilities, with Booking's `bookings.court_id`
  repointed at it, would touch the tested, invariant-bearing `EXCLUDE`
  constraint and `Repository.Create`'s bounded-retry logic from T4 — a
  one-way-door-shaped risk for zero behavioral gain. Instead: T7.3 adds a
  `facilities` table and a nullable `courts.facility_id` FK via a normal
  migration, plus `description`/`address`/`photo_urls`/`camera_links`
  columns either on `courts` or a new `facility_courts` metadata table
  (implementer's call, not prescribed here) — Booking's own domain, `during`
  EXCLUDE constraint, adapter, and proto are **untouched**. Facilities
  becomes a new read/write surface over largely the same physical table,
  not a rewrite of Booking's storage. This is a two-way door (the FK can be
  dropped/changed later) and should not consume more review time than that.
- **T7 does not add court pricing/discount UI or backend** (see open
  question #4 above) — logged here too so it isn't mistaken for an
  oversight when T7's tickets don't mention `discount_rules`.

### Toolchain note (same gotcha class as T4's buf/sqlc note)

This authoring environment has `node` v22.22.2 and `npm` 10.9.7 already
installed, confirmed by direct check — **unlike** T4's buf/sqlc gap, no
install step was needed here to scaffold. **This is not a guarantee for
whatever environment actually implements T7's tickets** — an implementer
agent should verify `node`/`npm` availability as its first step (same
spirit as HANDOFF.md's "First actions on resume") and install Node LTS if
missing before T7.1. `make generate`'s existing `openapi/` output (buf +
`protoc-gen-openapiv2`, already working per T4) is the typed-client
generation source T7.1 depends on — no new proto toolchain is needed, only
a new `openapi-typescript`-or-equivalent step consuming output that already
exists.

---

## Tickets

### T7.1 — Bootstrap the Vue 3 web client with a typed OpenAPI client and the reviewed design tokens

**Story:** As a developer building any future screen, I want a working Vue 3
project with a generated, typed REST client and the reviewed design
system's tokens in place, so that every subsequent UI ticket is additive
scaffolding, not repeated setup.

**Description:** No prior frontend code exists in this repo (confirmed —
no `web/`, no `package.json` anywhere). This ticket creates the
`web/` directory as the Vue app root, wires it to the already-working
`make generate` OpenAPI output, and ports the design tokens the 10-round
review validated (colors, radii, type scale — `docs/design/
v1-review-round-10-final.md` §2a names the exact CSS custom properties that
passed contrast verification, e.g. `--court`, `--paper`, `--ink`) rather
than re-deriving them from scratch. No product screens in this ticket.

**Instructions:**
1. Functional requirements:
   - Scaffold `web/` as a Vue 3 + TypeScript + Vite project (`npm create
     vue@latest` or equivalent), committed to this repo (not gitignored —
     unlike `internal/gen/**`, application frontend code is source, not
     generated output).
   - Add an `npm run generate:client` script that runs `make generate`
     (repo root) then feeds `openapi/pickleball/**/*.swagger.json` (or
     whatever `protoc-gen-openapiv2`'s actual output filenames are —
     confirm by inspection, don't guess) through a typed-client generator
     (`openapi-typescript` + a thin fetch wrapper, or `openapi-typescript-
     codegen` — implementer's choice, document which and why in the PR).
     Generated client code lives under `web/src/api/generated/` and is
     **gitignored**, mirroring `internal/gen/**`'s own "generated, never
     hand-edited, regenerate from source" convention (CLAUDE.md rule 6) —
     add the equivalent line to `.gitignore`.
   - Add a `web/src/styles/tokens.css` (or `.ts` design-token module)
     porting the color/radius/type-scale values from the round-10 artifact
     description (`v1-review-round-10-final.md` §2a's verified token names
     and the external handoff's `oklch(...)` values, `README.md`'s "Design
     Tokens" section) — both light and dark variants, since the artifact
     was verified theme-aware in both. Do not attempt to fetch or recreate
     the actual `.dc.html`/artifact file pixel-for-pixel (Fidelity note,
     handoff `README.md`: "not pixel-perfect final UI... apply the target
     codebase's own visual styling").
   - Add the three breakpoints as CSS custom media/container queries or a
     composable (`useBreakpoint()`), matching the external handoff's
     Platform Notes verbatim: web ≥1280px, iPad 768–1180px, iPhone <600px.
   - A minimal root layout (`App.vue`) with the role-indicator nav
     component described in the kickoff note's decision #1 — rendered with
     hardcoded/mock role data in this ticket (no real account backend
     exists), explicitly commented as a placeholder to be wired to real
     data once Identity/Users exists.
   - Add an npm script + CI-shaped step (a `web/README.md` note is
     sufficient for T7; no Jenkins wiring required this ticket) for
     `npm run build` and `npm run test` (component tests via Vitest) so
     future tickets have a place to add tests.
2. Non-functional requirements:
   - This ticket is TDD-adjacent, not TDD-first in the domain sense (there
     is no business logic yet) — but the generated-client wrapper and the
     `useBreakpoint()` composable each get at least one Vitest test
     (mocking `window.matchMedia` / a sample OpenAPI fixture) so the
     pattern is established for tickets that follow, not left untested
     "because it's just scaffolding."
   - Document in `web/README.md`: Node/npm version used, how to run
     `npm run generate:client` (depends on `make generate` at repo root
     first — cross-reference CLAUDE.md's own `make generate` gotchas),
     and that `web/src/api/generated/` is gitignored.

**Story points:** 5

**Labels:** `sprint:t7`, `role:product-engineer`, `type:chore`, `points:5`

---

### T7.2 — Add the Facility and Court domain model (pure domain, TDD)

**Story:** As the platform, I want `Facility` and its owned `Court`s modeled
as a real aggregate with validation rules, so that "a facility owner lists a
facility" (requirement #1/#2/#7) has an actual domain home instead of
courts existing as bare, ownerless IDs.

**Description:** Bootstraps `internal/facilities/domain`, mirroring how
`internal/booking/domain/booking.go` and `internal/socialplay/domain/
game.go` shape validated construction + explicit transition methods. Pure,
framework-free, TDD-first. No adapter/proto/migration work in this ticket —
see T7.3 for the architecture note on how this domain maps onto the
*existing* `courts` table without touching Booking's schema.

**Instructions:**
1. Functional requirements:
   - Write failing table-driven tests first (CLAUDE.md rule 1), then add
     `domain.Facility{ID, OwnerID, Name, Description, Address, PhotoURLs
     []string, CameraLinks []CameraLink, CameraConsentAttested bool}` and
     `domain.CameraLink{URL string, CourtID string (empty = facility-wide)}`
     to `internal/facilities/domain`.
   - `NewFacility(...)` validates: `OwnerID`/`Name`/`Address` non-empty
     (`domain.ErrEmptyFacilityField`, one sentinel naming which field via a
     wrapped error or a `Field` member — don't stringly-type three separate
     sentinels for three empty-string checks, mirror `domain.ErrInvalidAmount`
     -style single sentinel + context pattern from Payments T6.1, not a
     sprawling enum).
   - `AddCameraLink(url string) error`: **only legal when
     `CameraConsentAttested == true`** — this is the round-10 design
     review's one concrete, no-judgment-call finding (`v1-review-round-10-
     final.md` §2b: "the consent checkbox must default to unchecked...
     saving is blocked until the user actively checks it") translated into
     an actual domain rule, not just a UI default. Returns
     `domain.ErrCameraConsentRequired` if attestation is false. Write the
     test that would fail if this check were removed (PE dossier §4 — "what
     test would fail if this were subtly wrong").
   - `domain.Court{ID, FacilityID, Name}` — deliberately minimal in this
     ticket (no pricing fields; `courts` already has pricing via the
     existing `pricing_rules` table keyed by `court_id`, untouched by this
     ticket). `NewCourt(facilityID, name string) error` validates both
     non-empty.
   - Add `Facility`/`Court`/`Camera Link` terms to
     `agent-operating-handbook.md` A2's glossary in this ticket's PR
     (CLAUDE.md rule 7 — "add it here in the same PR that introduces it").
     Note: `Facility` and `Court` are already *named* in the glossary from
     T0 (context map table) — this ticket adds their actual field-level
     definitions, which don't exist yet; don't skip on the assumption the
     glossary already covers it.
2. Non-functional requirements:
   - Zero non-stdlib imports in `internal/facilities/domain` (CLAUDE.md
     rule 2).
   - Table-driven boundary tests: empty `OwnerID`/`Name`/`Address`,
     `AddCameraLink` before and after `CameraConsentAttested` is set,
     empty `Court.Name`/`FacilityID`.

**Story points:** 3

**Labels:** `sprint:t7`, `role:principal-engineer`, `type:story`, `points:3`

---

### T7.3 — Wire Facilities to Postgres + proto + gRPC/REST, reusing the existing `courts` table

**Story:** As a client (the Vue app this sprint, Swift/Kotlin later), I
want `CreateFacility`, `GetFacility`, `ListFacilities`, and `AddCourt` as
real API endpoints, so that Facilities is reachable outside of Go tests and
a Facility onboarding/browse UI has real data to read and write.

**Description:** Depends on T7.2. The adapter/infra ticket, mirroring
T5.4/T6.4's combined proto+DB+adapter+gRPC scope. **Architecturally
load-bearing, per the kickoff note:** this ticket adds a `facilities` table
and a `courts.facility_id` FK via migration, but does **not** modify
`bookings`, the `during` EXCLUDE constraint, or any Booking
domain/app/adapter code — Booking keeps referencing `courts.id` exactly as
it does today.

**Instructions:**
1. Functional requirements:
   - Add `db/migrations/0010_facilities.sql`: a `facilities` table
     (`id uuid`, `owner_id uuid`, `name text`, `description text`,
     `address text`, `photo_urls text[]`, `camera_consent_attested
     boolean not null default false`, timestamps) and `ALTER TABLE courts
     ADD COLUMN facility_id uuid REFERENCES facilities (id)` (nullable —
     existing seeded courts from `0002_seed.sql` have no facility until
     backfilled; a nullable FK is the correct, non-breaking choice, not an
     oversight). Add a `facility_camera_links` table (`id`, `facility_id`,
     `court_id nullable`, `url`) rather than an array column, so a single
     facility can have per-court camera links per the domain model's
     `CameraLink.CourtID` field.
   - Add `proto/pickleball/facilities/v1/facilities.proto`:
     `CreateFacility`, `GetFacility`, `ListFacilities` (supports a text/
     location filter — keep it simple, a `name ILIKE` filter is sufficient
     for T7, real geo-search is out of scope), `AddCourt` RPCs + grpc-
     gateway REST annotations, mirroring `booking.proto`/`payments.proto`'s
     style exactly. Run `make generate` — never hand-edit `internal/gen/**`
     (CLAUDE.md rule 6).
   - sqlc queries in `db/queries/facilities.sql`, following the existing
     `fromFields`-per-query pattern (CLAUDE.md Gotchas — sqlc emits a
     distinct `...Row` type per query whenever the column list isn't
     `SELECT *`).
   - Implement `internal/facilities/adapter/postgres` (repository) and
     `internal/facilities/adapter/grpcapi`; register alongside Booking/
     Social Play/Payments in `cmd/server`.
   - `AddCourt(facilityID, name)`: inserts into the **existing** `courts`
     table with `facility_id` set — do not create a second courts table or
     duplicate the row. Booking's `ListCourtBookings`/`CreateBooking`
     continue to read/write the same `courts.id` values unmodified.
   - Smoke-test AC (PR description, curl-list style): creating a facility
     with `camera_consent_attested: false` then calling `AddCourt` succeeds
     (courts don't require consent, only camera links do — T7.2's rule);
     attempting to add a camera link before consent is attested returns a
     mapped 4xx from `ErrCameraConsentRequired`, not a 500; `ListFacilities`
     returns the created facility.
2. Non-functional requirements:
   - Every new Postgres error this ticket can raise is translated to a
     domain error in the adapter — no raw `pgconn.PgError` crosses into
     `app`/`domain` (CLAUDE.md rule 5).
   - `make down && make up` after the schema change (Gotchas: initdb.d only
     applies on a fresh volume) — verify the pre-existing `0002_seed.sql`
     courts still exist and are bookable after the migration (a regression
     check that this ticket didn't silently break T0–T4's seed data).
   - `make test` green, including this context in the top-level
     race/coverage run.
   - Confirm via `go vet`/import inspection that `internal/booking/**`
     imports nothing under `internal/facilities/**` — the dependency runs
     the other way per the context map (Facilities has no listed
     dependency on Booking, and Booking doesn't need Facilities to keep
     working), and this ticket must not quietly introduce a
     cross-context coupling in either domain/app layer.

**Story points:** 8

**Labels:** `sprint:t7`, `role:principal-engineer`, `type:story`, `points:8`

---

### T7.4 — Vue: Facility onboarding flow (Owner/Host)

**Story:** As a Facility Owner or Host, I want to add a new facility with
its description, photos, address, and an optional, consent-gated camera
link, so that my courts become bookable on the platform (requirement #2/#7).

**Description:** Depends on T7.1 (scaffold) and T7.3 (real API). Implements
the design review's Flow 1 (`v1-system-design.md` §3.1, `handoff-2026-08/
README.md` "1. Facility Onboarding") against the real Facilities API, not a
mockup — including the round-10 review's specific, required correction.

**Instructions:**
1. Functional requirements:
   - A multi-step form (name/description/address → photos → cameras →
     courts, matching the reviewed flow's own step order) calling
     `CreateFacility` then `AddCourt` (one or more) via the generated
     client from T7.1.
   - **The camera-consent checkbox must default to unchecked** — this is
     not optional or a nice-to-have, it is the round-10 review's explicit,
     already-identified acceptance criterion (`v1-review-round-10-final.md`
     §2b/§4: "a required attestation that defaults to affirmed defeats its
     own stated purpose... recommendation: remove `checked`... make this an
     explicit acceptance-criterion line in whichever ticket builds this
     field"). Saving a camera link is blocked (button disabled or a
     validated submit-time error, implementer's choice) until the box is
     actively checked. Write a component test asserting the checkbox's
     initial DOM state has no `checked` attribute and that submitting a
     camera-link URL while unchecked does not call the API.
   - The checkbox's own click target should be at least the label's full
     width (round-10 §2b's secondary, non-blocking observation about the
     18×18 box vs. the `.toggle-hit` pattern elsewhere in the mockup) —
     implement with a generously-sized `<label>` wrapping the input, not a
     bare 18px box, so this ticket doesn't reintroduce the inconsistency
     the review flagged.
   - Per-court "Priced" / "Needs pricing" status badge on the courts list
     step: since T7 doesn't build pricing UI (kickoff note), this always
     reads "Needs pricing" for a freshly-created court — this is honest
     given `pricing_rules` has no write path yet (HANDOFF.md T1 gap), not
     a bug; add a one-line UI note ("Pricing setup coming soon") rather
     than a dead-end button that does nothing when clicked.
   - Error handling: a failed `CreateFacility` (e.g. empty name) surfaces
     the specific invalid field in text next to that field, not a generic
     toast — WCAG 3.3.1 Error Identification (`research-accessibility-
     i18n.md` §1), directly named as load-bearing for this app's form-heavy
     flows.
2. Non-functional requirements:
   - Responsive across all three breakpoints (T7.1's `useBreakpoint()`):
     single-column stacked steps on iPhone, two-column where space allows
     on iPad, full multi-column on web — per the external handoff's
     Platform Notes.
   - WCAG 4.1.3 Status Messages: a successful "Facility created" /
     "Court added" confirmation is exposed via an ARIA live region
     (`role="status"`), not just a visually-rendered toast (`research-
     accessibility-i18n.md` §1's concrete AA requirement).
   - Touch targets ≥44px on iPad/iPhone layouts (external handoff Platform
     Notes; Apple HIG minimum, `research-accessibility-i18n.md` §4).
   - Component tests (Vitest) for: consent-gate behavior (above), field-
     level error rendering, and the multi-step form's forward/back state
     retention (going back a step must not lose already-entered data).

**Story points:** 8

**Labels:** `sprint:t7`, `role:ux-ui-designer`, `type:story`, `points:8`

---

### T7.5 — Vue: Discover & browse facilities/courts (Player)

**Story:** As a Player, I want to browse and search real facilities and see
their courts, so that I can find a place to book (requirement #1),
replacing what would otherwise be fake/placeholder data in the UI.

**Description:** Depends on T7.1 and T7.3. A read-only consumer of
`ListFacilities`/`GetFacility` — no game/social data yet (that's T8), so
this screen shows facility/court browsing only, not the full Discover Games
card design from Flow 4 (which needs Social Play's Game data, out of scope
this sprint).

**Instructions:**
1. Functional requirements:
   - A facility list/search view (name filter, per T7.3's `ListFacilities`)
     and a facility detail view showing description, photos, address, and
     its courts list.
   - Empty/loading/error states for all three (no facilities yet; API
     unreachable; a facility with zero courts) — each with real UI, not a
     blank screen (NN/g heuristic #1, `docs/roles/ux-ui-designer.md`).
   - Facility-owned camera links are **not** shown on this player-facing
     screen — T7.2's domain note ("host-facing only... no player-facing
     surface") and the design review's own explicit statement of the same
     rule are both binding here; write a test asserting the camera-link
     data, even if present in the API response, never renders on this
     screen.
2. Non-functional requirements:
   - Responsive across all three breakpoints, matching the external
     handoff's per-breakpoint list/detail pattern (iPad: two-column
     list+detail; iPhone: single-column stacked; web: multi-column).
   - WCAG 1.4.1 Use of Color: any "open now" / "fully booked" style status
     indicator (if implemented) must carry a text label, not rely on color
     alone (`research-accessibility-i18n.md` §1).
   - Component tests (Vitest) for empty/loading/error states and the
     camera-link-never-renders-here assertion above.

**Story points:** 5

**Labels:** `sprint:t7`, `role:ux-ui-designer`, `type:story`, `points:5`

---

### T7.6 — Vue: Get a quote and book a court, end to end against the real Booking API

**Story:** As a Player, I want to pick a court and time, see a live price,
and complete a booking, so that "browse and book a court" (requirement #1)
is a real, working flow rather than a wireframe.

**Description:** Depends on T7.1, T7.3, and T7.5 (needs a facility/court to
book from). Wires the existing, already-working `GetQuote` and
`CreateBooking` RPCs (`booking.proto`, live since T1/T0) to a real UI for
the first time in this project's history — this is the first end-user-
facing consumer of Booking's API outside of curl/tests.

**Instructions:**
1. Functional requirements:
   - From a Court's detail view (T7.5), a date/time picker calls
     `GetQuote` and displays `price_cents`/`band`; confirming calls
     `CreateBooking` with `source: SOURCE_INDIVIDUAL` (the only source a
     bare Player-initiated booking can legitimately be — `game`/
     `competition`/`recurring_hire` sources are created by their owning
     context, not directly by this screen, per `booking.proto`'s own
     `Source` doc comment).
   - **Double-booking conflict handling:** a `409`-mapped
     `ErrCourtDoubleBooked` response surfaces as a specific, actionable
     message ("This slot was just booked — pick another time"), and per
     WCAG 3.3.3 Error Suggestion (`research-accessibility-i18n.md` §1,
     explicitly named as directly applicable to exactly this error), the
     UI re-queries and shows the next available slot(s) rather than just
     rejecting — reuse `ListCourtBookings` (already live since T2) to
     compute this client-side if the API doesn't already suggest one.
   - **Booking confirmation step:** per WCAG 3.3.4 Error Prevention
     (Legal, Financial, Data) — directly named in `research-accessibility-
     i18n.md` §1 as applicable to "booking-with-payment" — a review/confirm
     screen shows the court, time, and price before the final
     `CreateBooking` call fires; this is a real requirement, not
     boilerplate, since T7's booking flow has no payment step yet (per the
     roadmap's starting facts) and this confirm step is the *only*
     error-prevention mechanism this flow has this sprint.
   - A booking-success confirmation exposed via an ARIA live region (WCAG
     4.1.3, same requirement class as T7.4).
2. Non-functional requirements:
   - No client-side price computation — the displayed price is always
     whatever `GetQuote` returned, never recomputed/cached stale in the UI
     (a Quote can change between viewing and confirming; re-fetch on
     confirm if more than e.g. 60 seconds elapsed — implementer's exact
     threshold, but the principle — never submit a price the server hasn't
     just confirmed — is required, not optional).
   - Responsive across all three breakpoints.
   - Component/integration tests (Vitest, mocking the generated client):
     happy path (quote → confirm → booked), the double-booking conflict
     path (asserts the specific message + suggested-slots behavior, not
     just that an error appeared), and the confirm-step gate (asserts
     `CreateBooking` is never called before the review step is shown).

**Story points:** 8

**Labels:** `sprint:t7`, `role:ux-ui-designer`, `type:story`, `points:8`

---

### T7.7 — Object-level authorization regression tests for Facilities' write endpoints

**Story:** As a platform operator, I want `CreateFacility` and `AddCourt` to
reject an actor acting on a facility they don't own, so that one user can't
mutate another user's facility before real auth exists (mirrors T5.5/T6.7's
closure of the same class of gap for Social Play/Payments).

**Description:** Depends on T7.3 (needs real endpoints). QA-owned, same
role/shape as T5.5 and T6.7 — this project now has a established, repeated
pattern for this check and T7.7 is its third instance, not a novel design.

**Instructions:**
1. Functional requirements:
   - Add a handler-level or `-tags=integration` test (implementer's choice,
     same reasoning T5.5 used for picking handler-level over a full
     Postgres round trip when the check itself has no SQL involved) proving
     `AddCourt` against a Facility whose `owner_id` doesn't match the
     request's `actor_user_id` is rejected with a mapped, non-500 status —
     not a silent success.
   - Same proof for a hypothetical `UpdateFacility`/`AddCameraLink` RPC
     *if* T7.3 shipped one beyond `CreateFacility`/`AddCourt` (check the
     actual shipped proto — don't assume a method exists that T7.3 didn't
     build; if only `CreateFacility`/`ListFacilities`/`GetFacility`/
     `AddCourt` exist, scope this ticket to `AddCourt` only and say so in
     the PR, don't invent scope).
   - As with T5.5/T6.7, document explicitly (PR description + a line added
     to `HANDOFF.md`'s existing Auth cross-cutting item, not a new one)
     that this proves the *object-level* check given a claimed
     `actor_user_id`, not real authentication — same caveat already
     established twice, must not be contradicted or re-litigated here.
2. Non-functional requirements:
   - No real authentication work in this ticket — same boundary T5.5/T6.7
     held.
   - Verify the test actually fails if the ownership check is removed
     (CLAUDE.md rule 10 / PE dossier §4 — "tests do not test themselves"),
     the same way T5.5 verified its own regression test by temporarily
     commenting out the check.

**Story points:** 3

**Labels:** `sprint:t7`, `role:qa`, `type:story`, `points:3`

---

## Sprint totals

- **Tickets:** 7 (T7.1–T7.7)
- **Total story points:** 40 (5 + 3 + 8 + 8 + 5 + 8 + 3)
- **Design-review findings closed this sprint:** the round-10 review's one
  concrete, no-judgment-call finding (camera-consent checkbox must not
  default to checked) is implemented as a tested acceptance criterion in
  both T7.2 (domain rule) and T7.4 (UI default state) — the first sprint
  since the review closed that actually builds the field the finding was
  about.
- **Open UX questions resolved this ceremony:** #1 (role-switching UX —
  contextual indicator, decided) and #5 (Club as a 4th account type —
  adopted as domain direction, not built) both got real decisions; #4
  (pricing-conflict UI) got a decision for future tickets to inherit
  without building it now; #2 (auto-matching transparency) and #3 (cash
  payment surfacing) are explicitly deferred to T8/T9 because T7 has no
  screen for them to attach to.
- **Genuine disagreements recorded (not manufactured consensus):**
  PM's fixture-backed-UI proposal vs. PE's real-backend-first sequencing
  for T7.4–T7.6 (PE's position prevailed, PM accepted the reasoning);
  PM's vs. PE's framing of *why* native mobile is deferred (same
  conclusion, different emphasis — user-scope-completeness vs.
  build-order risk).
- **Explicitly deferred, not forgotten:** Social Play/Payments/Competitions
  UI, social-account-linking, WhatsApp/Zalo bot integration (→ T8);
  `discount_rules` backend + pricing UI, Club account type + rental flow,
  cross-sprint WCAG audit (→ T9); native Swift/Kotlin mobile apps (→ T10+,
  revisit explicitly with the user at the T9 retro rather than re-deferring
  silently again).
