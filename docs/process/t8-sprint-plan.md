# T8 Sprint Plan — Social Play + Payments UI spine, closing T7's carried gaps

Produced by Ceremony 1 (Backlog refinement) + Ceremony 2 (Sprint planning)
per `docs/process/sprint-process.md`, played jointly by **Product Manager**
(handbook B2) and **Principal Engineer** (handbook B1 + `docs/roles/
principal-engineer.md`).

> **Re-scope notice, read this first.** `docs/process/t7-sprint-plan.md`'s
> Part A roadmap named T8 as five sub-scopes in one sprint: (a) Social Game
> creation + Discover/Join Games UI, (b) Payments UI, (c) a new
> **Competitions** bounded context, (d) social-account-linking +
> shareable-registration-links, (e) a WhatsApp/Zalo bot spike. Re-verifying
> the actual code (not just the roadmap's assumptions) before ticketing
> found two things that make shipping all five in one sprint the wrong
> call, not just a lot of work:
>
> 1. **The roadmap's own reasoning already argued against this.** T7's
>    "Why not fewer or more sprints" section rejected folding T8 into T7
>    specifically because stacking a new bounded context and new vendor
>    integrations onto first-time infrastructure "repeats the exact 'too
>    much new, unproven surface in one sprint' pattern this project's own
>    process exists to avoid" (PE dossier §3, "innovation tokens"). Sub-scope
>    (c) is a brand-new bounded context (the same weight class as T5's
>    27-point, 5-ticket Social Play build), (d) is a brand-new external
>    integration surface (OAuth token storage per host per platform), and
>    (e) is *two* candidate brand-new vendor integrations condensed into "a
>    spike" — that is three separate categories of new, unproven surface
>    stacked on top of (a)/(b)'s UI work in the same sprint T7's own
>    reasoning says not to do. The roadmap named the risk and then queued
>    the exact pattern it warned about for the very next sprint.
> 2. **Verifying "Social Play... now pointing at real Facilities/Courts
>    instead of a fake" against the actual code found it's only half true,
>    plus a second, undocumented gap.** `CreateGame`'s `court_ids` are real
>    Booking court IDs (confirmed — `internal/socialplay/adapter/booking`
>    calls Booking's real `app.Service.CreateBooking`, live since T5.3/T5.4)
>    — but `Game.FacilityID` (`internal/socialplay/domain/game.go`) is a
>    bare, unvalidated string with no FK to `facilities.id` (confirmed —
>    `db/migrations/0005_socialplay.sql:23`, `games.facility_id text NOT
>    NULL`, already flagged in `HANDOFF.md`'s Cross-cutting section as
>    unreconciled). Separately, and **not previously flagged anywhere**:
>    `domain.Game` (`internal/socialplay/domain/game.go`) and
>    `domain.Registration` (`internal/socialplay/domain/registration.go`)
>    have *no* fields at all for `PaymentMethod`, `CancellationCutoff`,
>    `GuestAllowance`, or a per-registration guest count — confirmed by
>    grepping both files and `socialplay.proto` for `Level|Match|AutoMatch|
>    Gender|PaymentMethod|Guest|Cancellation` (zero hits beyond
>    `Registration.PaymentStatus`, which is a different concept — whether a
>    payment was made, not which method a Game accepts). Every one of these
>    is a named field in Flow 3 (Social Game Creation) and Flow 4/6
>    (Discover & Join, Payments) in
>    `docs/design/handoff-2026-08/README.md`'s "Screens / Flows" section.
>    Building a "Social Game creation" screen against the roadmap's assumed
>    real API today would mean a UI with a payment-method selector,
>    guest-allowance stepper, and auto-match/level/gender controls that
>    **do nothing on submit** — none of that state has anywhere to be
>    persisted. This is the same class of gap T7's own kickoff note caught
>    for Facilities before committing to build fixture-backed UI (PM
>    proposed it, PE caught that a fixture "would need its data-fetching
>    layer rewritten the moment the real API lands" and worse, "is exactly
>    the kind of detail... expensive to discover was wrong against a
>    fixture later") — the same reasoning applies here: don't build a
>    payment-method selector against a Game that structurally cannot
>    remember which method was picked.
>
> **Resolution: T8 is re-scoped down to sub-scopes (a) and (b) only**,
> expanded to include the real backend prerequisites (a) and (b) turned out
> to need (facility linkage, courts-listing, the missing Game/Registration
> fields), plus closing three small, already-flagged T7/T6 carry-over gaps
> that sit on the same code this sprint touches anyway (courts-listing,
> camera-consent-attest-path, and the still-outstanding T6.7 Payments authz
> regression ticket). **Competitions (c), social-account-linking +
> shareable-registration-links (d), and the WhatsApp/Zalo spike (e) move to
> T9** — a new roadmap slice for "growth & Competitions," described below.
> The former T9 (`discount_rules` + Club rentals + cross-sprint WCAG
> hardening) becomes **T10**. This is a plain re-numbering, not a scope
> cut — every item T7's roadmap named is still tracked, just one sprint
> later than originally sketched, for the same reason T7 itself was
> sequenced ahead of T8 in the first place (spine before growth features).
> Auto-matching (level/gender matching) turns out to have **no scheduled
> sprint at all** in any prior roadmap — flagged below as a genuinely open
> gap, not silently folded into T9's Competitions slice.

---

# Part A — Roadmap update (supersedes T7 plan's T8/T9 lines)

## Revised sprint boundaries

**T8 (this document, fully ticketed in Part B) — Social Play + Payments UI
spine, closing T7's carried gaps.** Builds the real backend surface Flow 3
(Social Game Creation) and Flow 4 (Discover & Join Games) need that doesn't
exist yet (facility linkage, a courts-listing read path, payment-method /
guest-allowance / guest-count fields), then the two screens themselves, then
Flow 6 (Payments) wired to the now-real Registration/Game data — plus three
small, already-flagged carry-over fixes (`ListFacilityCourts`,
`AttestCameraConsent`, the outstanding T6.7 Payments authz regression
ticket) that sit on the same Facilities/Payments code this sprint already
touches. Does **not** touch Competitions, social-account-linking,
shareable-registration-links, or any social-platform bot integration.

**T9 (not ticketed here) — Competitions context + growth/social
integration.** Everything T8's original roadmap named for sub-scopes (c),
(d), (e), unchanged in substance:
- A new **Competitions** bounded context
  (`internal/competitions/{domain,app,port,adapter}` +
  `proto/pickleball/competitions/v1`, mirroring Social Play's shape closely
  — a `Competition` reserves courts the same way a `Game` does, per the
  glossary) plus its Discover/Advertise UI (Flow 5).
- Social-account-linking (an OAuth token per host per platform) and
  shareable-registration-links (in-app RSVP — **still** not reply-scraping,
  this remains a locked decision, see `docs/design/v1-system-design.md` §4
  and `docs/design/v1-external-reference-reconciliation.md`'s "one direct,
  load-bearing conflict" section).
- A scoped WhatsApp-or-Zalo owned-channel-bot spike. T7's research
  (`docs/process/t7-sprint-plan.md`'s "Social platform integration:
  WhatsApp vs. Zalo, researched" section) is re-checked here for staleness
  and found **not stale** — the follow-based-vs-message-based opt-in
  difference, the Zalo OA tiered-verification cost, and the
  Vietnam-vs-global market-scope question are all still-open facts about
  the platforms themselves, not project-specific state that could have
  drifted in one sprint. T9 planning should cite it directly rather than
  re-research.
- **New, not in T7's roadmap at all: a decision on whether/when
  auto-matching (level/gender-based matchmaking) gets built.** T7's open
  question #2 ("auto-matching transparency") was deferred sprint over
  sprint on the reasoning that there was no screen for it to attach to yet
  — but this ceremony found the deeper reason: there is no `Match`/
  `PlayerRating` aggregate, no matching algorithm, and no `Level` field
  anywhere in the codebase (`HANDOFF.md`'s T5-done note already says
  "Matchmaking deferred past T5" and no sprint since has picked it up).
  This is not a T9 sub-scope by default — T9 Ceremony 1 needs to explicitly
  decide whether Competitions' bracket/seeding logic creates enough shared
  need with matchmaking to justify building a minimal `PlayerRating` this
  sprint, or whether it stays deferred again with a real reason recorded
  (not silently dropped a fourth time).

**T10 (renumbered from the T7 plan's "T9") — Pricing/discount UI + Club
rentals + hardening.** Unchanged in substance from T7's plan: the
`discount_rules` backend + its UI (Flow 2), the Club account type + club
recurring-rental flow (Flow 7), and the cross-sprint WCAG 2.2 AA audit.

## Why re-scope now rather than push through as roadmapped

**PE's position:** the roadmap's own "innovation tokens" argument is the
reason to split, and the newly-found Game/Registration field gap makes it
concrete rather than abstract — building Flow 3/4's UI against fields that
don't exist would either (a) silently ship dead controls, which is worse
than not building the screen, or (b) require inventing the backend fields
*inside* a UI ticket, which violates this project's own TDD-first,
domain-before-adapter-before-UI sequencing (CLAUDE.md rule 1, rule 3) and is
exactly the kind of "an approach looks elegant on paper but is a trap in
practice" PdE's brief (`agent-operating-handbook.md` B3) exists to catch.
Splitting cleanly — backend fields first (T8), a proper bounded context for
Competitions with its own spine (T9) — costs one sprint of sequencing, not
one sprint of lost work.

**PM's position:** agreed on the split, with one condition — T8 must still
ship something end-to-end demoable (not just backend plumbing), because a
sprint that's "all prerequisite, no user-visible slice" repeats the T7
kickoff note's own worry about a sprint whose entire output sits behind
another sprint's UI. Resolution: T8's ticket order guarantees Flow 3, Flow
4, and Flow 6 (create a game, discover and join it, pay for it — online or
cash) are all real and clickable by the end of this sprint; only
Competitions/growth is what actually moves to T9, not "Social Play/Payments
UI in general."

**Genuine disagreement recorded (not manufactured consensus):** PM initially
argued the T6.7 Payments-authz-regression ticket (open since T6, never
implemented) should be pulled into T8 on the grounds that a two-sprint-old
gap shouldn't age into a third sprint unaddressed (same reasoning T5's own
retro finding #6 already established for this project). PE's initial
position was that T8 is already carrying more prerequisite work than any
prior sprint and T6.7 is unrelated in theme (it's Payments-authz debt, not
Social-Play/Payments-*UI* work) — cutting it to keep T8's total point count
in line with T5–T7's growth trend (27 → 37 → 40). **Resolution: PM's
position prevailed.** T6.7 is small (3 points, a well-understood pattern by
its third live instance — T5.5, T7.7 both already did this exact shape of
work), touches code T8 doesn't otherwise modify, and "wait for a less-full
sprint" is exactly the reasoning that let it slip through T7 already. It is
ticketed below as T8.5.

---

# Part B — T8 Sprint Plan

## Sprint goal

> A Host can publish a Social Game at a real, linked Facility with a real
> payment method and guest allowance, a Player can discover it, join it
> with guests, and pay — online (Stripe-stub checkout) or cash (surfaced as
> pending to the Host until settled) — all through the real Vue app
> navigating between actual routed screens, against real gRPC-gateway REST
> APIs with no fabricated or dead-end fields anywhere in the flow.

## Kickoff note

### Scope decisions against the open UX questions (carried from T7's kickoff note)

1. **Role-switching UX.** Already decided in T7 (contextual indicator, UI-
   state only). T8 extends the existing `RoleIndicator` mechanism to
   recognize "Host" once the signed-in browser session has created at least
   one Game (T8.8) — no new mechanism, just a second role fed into the
   pattern T7.1 already built for exactly this. Same caveat as before: this
   is client-side session state, not a server-verified permission, until
   the JWT/Auth0 cross-cutting item lands.
2. **Auto-matching transparency.** **Still deferred, but for a newly
   verified, stronger reason than "no screen yet."** There is no
   `Match`/`PlayerRating` aggregate or matching algorithm in this codebase
   at all (verified this ceremony — see the re-scope notice above), so a
   transparency UI has no underlying feature to be transparent about. T8.8
   (Social Game Creation) ships the fields Flow 3 names that *do* have a
   real backend home (court & time, capacity, payment method, guest
   allowance) and explicitly omits the auto-match toggle / level-range
   slider / gender-mix selector, with an inline note in the ticket
   (T8.8's Instructions) rather than silently truncating the form. Raised
   as a T9-Ceremony-1 input above, not decided here.
3. **Cash-vs-online payment surfacing prominence.** **Decided now,
   implemented this sprint** (T8.10): a Host-facing dashboard shows a
   pending-cash-payments count badge plus a list (Payment status = unpaid,
   method = offline) they can mark paid from, reusing
   `RecordOfflinePayment` (live since T6.3/T6.4) — no push notification or
   email (no notification infrastructure exists; out of scope, not
   silently assumed). Player-facing: a Game card and the join/confirm
   screen show "Cash at facility" as a plain text label next to price
   (WCAG 1.4.1, same rule T7.5 already applied to booking-status
   indicators — never color-only).
4. **Pricing-conflict UI.** Unchanged from T7 — still deferred to T10 with
   T7's kickoff note already recording the design direction (validation
   error at entry time, not a runtime resolution surface). Not touched in
   T8.
5. **Club as a fourth account type.** Unchanged from T7 — adopted as domain
   direction, not built. Not touched in T8; still T10.

### Scope decisions against the newly-found Game/Registration field gap

- **`PaymentMethod` is a display/gating hint, not a new enforced
  invariant.** Flow 3's "payment method: online/cash/either" selector
  becomes `domain.Game.PaymentMethod` (T8.6), read by the UI to decide
  which join button(s) to show (T8.9) — it does **not** gate which of
  `CreateOnlinePayment`/`RecordOfflinePayment` the backend will actually
  accept. PE flagged this explicitly as a one-way-door risk avoided, not
  an oversight: those two RPCs are already independently callable
  (T6.2/T6.3) with their own actor/ownership checks, and adding
  cross-context enforcement ("reject `RecordOfflinePayment` if
  `Game.PaymentMethod == online`") would mean Payments' `app.Service`
  reaching back into Social Play's domain to check a field, which the
  context map (`agent-operating-handbook.md` A1) already says runs the
  other direction (Payments depends on Social Play via a port, never a
  live cross-context read of a domain field). If real enforcement is
  wanted later, it's a `port.GamePaymentPolicy`-shaped addition — not
  designed further here, logged as a deferred question, not a gap.
- **Guest count affects the capacity invariant, and that's real domain
  work, not just a new field.** `GuestAllowance` (max guests per
  registrant, on `Game`) and a per-`Registration` `GuestCount` mean the
  capacity check T5.2 built (`count of active registrations >=
  Capacity`) is no longer correct — it must become `sum of (1 +
  GuestCount) across active registrations >= Capacity`. T8.6's
  Instructions require this exact test (a capacity-3 Game where one
  registration books 2 guests should reject a second registration that
  would push the total over 3, even though only 2 *registrations* exist).
  This is the same shape of gap T5's retro finding #1 named ("does this
  ticket introduce a count-limited invariant? If so, does the ticket
  text itself require a DB-level guard, not only a unique-key guard?") —
  T8.7 (the Postgres/proto wiring ticket) must extend T5.4/T6.6's existing
  capacity-guard trigger (`db/migrations/0006_socialplay_capacity_guard.sql`)
  rather than leave the DB-level guard checking registration *count* while
  the domain checks registration *weight* — see T8.7's Instructions.
- **`CancellationCutoff` is explicitly dropped from this sprint's scope**,
  not silently renamed to something smaller. Unlike `PaymentMethod`/
  `GuestAllowance`, a cancellation cutoff is a real time-based enforcement
  rule (a `Registration.Cancel` call after the cutoff should behave
  differently than one before it) with no existing analog anywhere in this
  codebase (Booking's own `CancelBooking` has no time-based restriction
  either). Shipping a cosmetic text field that enforces nothing would be
  actively misleading (a Host would reasonably expect a cutoff they typed
  in to do something). Logged here as a real, not-yet-ticketed gap —
  raise at T9 or T10 backlog refinement once there's a concrete reason
  (e.g., a no-show-fee policy) to build real time-gated cancellation logic.

### Toolchain / environment note (same gotcha class as T4's and T7's)

This authoring environment was not verified for `node`/`npm`/Docker/buf/sqlc
availability as part of this ceremony (a planning-only session, per the
task brief — no `make test` run). Whoever implements T8's tickets should
run the same "first actions" check `HANDOFF.md` and T7's kickoff note
already establish (verify toolchain availability before assuming it) rather
than trust this note either way.

---

## Tickets

### T8.1 — Add client-side routing (Vue Router)

**Story:** As a Player or Host using the web app, I want to navigate
between screens (browse facilities, view a game, check out) via real URLs
and back/forward navigation, so that the app behaves like a real multi-
screen application instead of every screen being stacked on one page.

**Description:** `App.vue`'s own header comment already flags this as
overdue: T7.4 and T7.5 both landed their first product screen with no
router in between, so both got stacked as block siblings
(`FacilityOnboarding` + `DiscoverFacilities`, the latter itself owning
`CourtBookingFlow`). `HANDOFF.md`'s "Not yet built" section names this as
"first candidate for a T8 ticket." T8.8/T8.9/T8.10 (Game creation, Discover
& Join, Payments) each need a real screen of their own — building three
more stacked-sibling screens on top of the existing two would make the
problem this ticket exists to fix strictly worse, so it's sequenced first
and everything else in this sprint depends on it.

**Instructions:**
1. Functional requirements:
   - Add `vue-router` (pin an exact version, avoid repeating the
     `--legacy-peer-deps` friction `HANDOFF.md` already logged for
     `openapi-typescript` — check `vue-router`'s peer range against the
     scaffolded `vue@^3.5`/`typescript@~6.0` versions before adding).
   - Define routes for every existing and T8-planned top-level screen:
     `/facilities` (existing `DiscoverFacilities`), `/facilities/onboard`
     (existing `FacilityOnboarding`), `/games` (new, T8.9 Discover & Join),
     `/games/new` (new, T8.8 Social Game Creation), `/games/:id/checkout`
     (new, T8.10 Payments UI), `/host/payments` (new, T8.10's pending-cash
     dashboard). Routes for T8.8–T8.10's screens are added as empty
     placeholder components in *this* ticket (a route that renders "Coming
     soon") — the tickets that build each screen for real replace the
     placeholder, they don't invent the route.
   - `App.vue` becomes a shell with `<RouterView />` plus persistent chrome
     (the existing header/`RoleIndicator`) — replace the current
     stacked-siblings body, per the file's own comment inviting exactly
     this replacement.
   - A minimal nav (matching the external handoff's Platform Notes: a
     persistent sidebar on web ≥1280px, a bottom tab bar on iPhone <600px
     per its named tabs — Discover / Bookings / Games / Profile — reuse
     that exact set as the nav's structure even though "Bookings"/"Profile"
     have no screen yet; link them to a "Coming soon" placeholder route
     too rather than omit them, so the nav shape doesn't need rework when
     those screens eventually land).
2. Non-functional requirements:
   - Existing T7.4/T7.5 component tests must still pass unmodified in
     substance — if a test currently mounts `FacilityOnboarding`/
     `DiscoverFacilities` standalone (not through the router), it's fine to
     leave as-is; only add router-aware tests for new behavior, don't force
     a rewrite of passing tests as part of this ticket.
   - Add a component test asserting `/facilities` and `/facilities/onboard`
     each render their existing screen and no other screen renders
     alongside it (the specific regression this ticket exists to fix —
     write the test that would fail against today's stacked-siblings
     `App.vue`).
   - Responsive nav behavior (sidebar vs. bottom tab bar) verified across
     `useBreakpoint()`'s three breakpoints via a component test.

**Story points:** 5

**Labels:** `sprint:t8`, `role:product-engineer`, `type:chore`, `points:5`

---

### T8.2 — Add `ListFacilityCourts` (Facilities read endpoint)

**Story:** As a client (Vue this sprint), I want to fetch the list of
Courts belonging to a Facility, so that a Facility's courts list is real
data instead of the permanent empty state T7.5/T7.6 both had to design
around.

**Description:** Closes the gap `HANDOFF.md`'s Cross-cutting section
already names explicitly: "T7.3 shipped `AddCourt` as create-only — there
is no RPC that lists a Facility's Courts back... `FacilityCourt[]` type
[already exists], but `mapToFacilityDetail` has nothing to populate it
from... A `ListFacilityCourts` (or equivalent `GetFacility` response field)
RPC is the natural next Facilities ticket." T8.8's Game-creation court
picker and T8.9's Discover-games facility detail both need this to show
real courts rather than an empty list or T7.6's manual court-ID-entry
workaround.

**Instructions:**
1. Functional requirements:
   - Add either a `ListFacilityCourts` RPC or a `courts` field on
     `GetFacilityResponse` (implementer's call — check what
     `FacilityCourt[]`/`mapToFacilityDetail` in `web/src/models/
     facility.ts` already expects and pick whichever needs less client
     rework, document the choice in the PR) to
     `proto/pickleball/facilities/v1/facilities.proto`. Run `make
     generate` — never hand-edit `internal/gen/**` (CLAUDE.md rule 6).
   - Backing sqlc query + `internal/facilities/adapter/postgres` method
     reading `courts WHERE facility_id = $1`, following the existing
     `fromFields`-per-query pattern.
   - Wire `internal/facilities/adapter/grpcapi`; this is a read path, no
     new ownership check needed (any Player may view a Facility's public
     court list, same as `GetFacility`/`ListFacilities` today).
   - Wire the Vue client: `DiscoverFacilities`/`FacilityDetailPanel.vue`
     (T7.5) should now render real courts instead of the empty state —
     update its existing zero-courts-empty-state test to also cover the
     populated case, don't just add a new test file alongside an
     unmodified stale one.
2. Non-functional requirements:
   - Confirm this ticket does not touch Booking's schema/domain/adapter at
     all (reads the existing `courts` table only) — same "two-way door,
     don't touch Booking's tested invariant" discipline T7.3's kickoff note
     established.
   - `make test` green, including this context.

**Story points:** 3

**Labels:** `sprint:t8`, `role:principal-engineer`, `type:story`, `points:3`

---

### T8.3 — Reconcile `games.facility_id` with real Facilities

**Story:** As the platform, I want a Game to reference a real, existing
Facility (not an arbitrary string), so that "browse games at a real venue"
(Flow 4) shows venues that actually exist and Facility Owners can trust
that a Game claiming to be at their Facility actually is.

**Description:** Closes the gap `HANDOFF.md`'s Cross-cutting section
already names: "`games.facility_id text NOT NULL`... is deliberately left
unreconciled — it's an opaque string column with no FK to anything...
reconciling it... needs a decision on whether a Game's free-text facility
description and a real onboarded Facility are the same concept or two that
need a migration path between them — raise at the next backlog refinement,
don't decide unilaterally." This ceremony's decision, recorded here: they
are the **same concept** — a Game happens at a real, onboarded Facility,
not an independent free-text description of one, because Flow 4's whole
premise (discover games at browsable venues) only works if a Game's
Facility is the same entity Facilities' `ListFacilities`/`GetFacility`
already returns.

**Instructions:**
1. Functional requirements:
   - Add `db/migrations/0011_socialplay_facility_fk.sql`: since
     `games.facility_id` is currently `text NOT NULL` with no guaranteed
     relationship to `facilities.id uuid`, don't attempt an in-place type
     change against unknown existing data — add a new nullable
     `games.venue_facility_id uuid REFERENCES facilities (id)` column
     (nullable: pre-T8 seeded/test Games have no real Facility to point
     at) and leave the old `facility_id text` column in place,
     unreferenced by new code, with a migration-file comment stating it is
     deprecated and slated for removal once no reader depends on it
     (mirrors the `courts.facility_id` nullable-FK precedent T7.3 already
     set for the same "existing rows predate the concept" reason).
   - `domain.Game` gains `VenueFacilityID string` (T8.6 is the ticket that
     restructures `Game`'s other new fields — if T8.6 hasn't merged yet
     when this ticket starts, coordinate the struct change once rather
     than twice; PE's call on sequencing if both are in flight
     simultaneously).
   - `app.Service.ScheduleGame` validates `VenueFacilityID` against a new
     `port.FacilityLookup` interface (mirrors the existing
     `port.CourtReservation` shape from T5.3 — primitives only, no
     `facilitiesdomain.Facility` type crossing into Social Play's
     domain/app) implemented by `internal/socialplay/adapter/facilities`
     calling the real `facilitiesapp.Service.GetFacility`. An unknown
     `VenueFacilityID` returns a new `domain.ErrFacilityNotFound`, mapped
     to a 404-shaped gRPC status, not a 500 or a silent accept.
   - `CreateGameRequest`/`Game` proto messages gain `venue_facility_id`
     (keep the existing `facility_id` field on the wire too, marked
     deprecated in a doc comment, for the same "don't break existing
     readers mid-migration" reason as the DB column).
2. Non-functional requirements:
   - `internal/socialplay/domain` and `internal/socialplay/app` still must
     not import `internal/facilities/domain` or `internal/facilities/app`
     directly (same dependency-rule discipline T5.3's kickoff note
     established for Booking) — verify via import inspection.
   - Test: `ScheduleGame` with a `VenueFacilityID` that doesn't exist in
     Facilities is rejected before any Booking reservation is attempted
     (no partial-state Game/Booking left behind).
   - `make down && make up` after the schema change; confirm pre-existing
     seeded Games (if any reference the old `facility_id` only) don't
     break — decide and document whether they get a backfilled
     `venue_facility_id` or are left `NULL` (leaving `NULL` is acceptable
     given the column is nullable by design; don't invent backfill logic
     beyond what a straightforward migration note explains).

**Story points:** 5

**Labels:** `sprint:t8`, `role:principal-engineer`, `type:story`, `points:5`

---

### T8.4 — Add `AttestCameraConsent` (Facilities), closing the T7-flagged dead-end gap

**Story:** As a Facility Owner, I want to actually attest camera consent
for my Facility, so that adding a camera link (a feature the onboarding UI
already builds a full, working form for) is reachable by a real user
instead of unconditionally rejected.

**Description:** Closes the gap `HANDOFF.md`'s Cross-cutting section
already names as a real footgun, not a hypothetical: "T7.2/T7.3
deliberately shipped no API path that ever sets
`Facility.CameraConsentAttested` to `true` server-side... as of T7 there is
no way for a real user to ever get a camera link accepted end-to-end —
every correct client submission still gets `ErrCameraConsentRequired`...
with no ticket filed yet." Folds in the object-level ownership check in
the same ticket that adds the write path, per the T5-retro lesson (finding
1: bake an invariant's DB/domain guard into the ticket that introduces it,
not a follow-up).

**Instructions:**
1. Functional requirements:
   - Add an `AttestCameraConsent` RPC to
     `proto/pickleball/facilities/v1/facilities.proto`
     (`AttestCameraConsentRequest{facility_id, actor_user_id}` /
     `AttestCameraConsentResponse{facility}`) — a dedicated RPC, not folded
     into `AddCameraLinkRequest`, per the round-10 design review's
     requirement (cited in T7.2's own ticket text) "that consent not be
     settable at creation time" — attesting consent and adding a link stay
     two explicit steps, matching the onboarding flow's own "Cameras" step
     already being separate from "Courts."
   - `domain.Facility` gains an `AttestCameraConsent(actorUserID string)
     error` method: calls the existing `EnsureOwner` check first (T7.7's
     pattern — a non-owner gets `ErrNotFacilityOwner`, mapped to 403, never
     seeing whether consent was already attested), then sets
     `CameraConsentAttested = true`. Idempotent: attesting twice is not an
     error (re-attesting an already-attested Facility just confirms the
     same state).
   - `app.Service.AttestCameraConsent` fetches the Facility, calls the
     domain method, persists.
   - Regression test (handler-level, same reasoning T7.7 used): a non-owner
     calling `AttestCameraConsent` is rejected with a mapped 403, not a
     500 and not a silent success — verify by temporarily disabling the
     `EnsureOwner` call and confirming the new test fails (CLAUDE.md rule
     10 / T7.7's own verification pattern), then restore it.
2. Non-functional requirements:
   - `FacilityOnboarding.vue`'s existing "Cameras" step (T7.4) is wired to
     call this new RPC before its already-built `AddCameraLink` call —
     T7.4's consent checkbox already exists and defaults unchecked
     correctly; this ticket is what makes checking it and submitting
     actually work end-to-end for the first time. Update T7.4's existing
     component test suite to cover the real network call rather than
     leave it asserting only that the checkbox blocks submission.
   - `make test` green, including this context.

**Story points:** 3

**Labels:** `sprint:t8`, `role:principal-engineer`, `type:story`, `points:3`

---

### T8.5 — Finish T6.7: object-level authorization regression tests across Payments' endpoints

**Story:** As a platform operator, I want every Payments endpoint that acts
on a specific payable action to reject a mismatched actor, so that one user
can't record or view another user's payment data before real auth exists —
closing the gap `docs/process/t6-sprint-plan.md`'s T6.7 ticket already
specified in full but that never got implemented.

**Description:** `HANDOFF.md`'s "Not yet built" section lists this
explicitly: "QA object-level-auth regression tests for Payments (T6.7) —
no PR, not started." PM raised at this ceremony that a two-sprint-old
carried gap shouldn't age into a third sprint unaddressed — the same
"flagged in prose, not tracked as a real ticket" pattern T5's retro finding
6 already flagged as a process risk for this project specifically. PE
agreed to include it (see Part A's "genuine disagreement recorded" note)
given it's small, well-understood by its third live instance of this
pattern (T5.5, T7.7), and touches code no other T8 ticket modifies, so it
carries no sequencing risk. Ticket text below is T6.7's original text,
unchanged (the work was never done, the ticket was never wrong).

**Instructions:**
1. Functional requirements:
   - Add an integration-level test (mirroring T5.5's pattern, `-tags=
     integration` or handler-level if a full Postgres round trip isn't
     needed) that: a Player who is neither the Game's Host nor an
     assigned Game Admin attempts `RecordOfflinePayment` against another
     player's Registration, and asserts the request is rejected with the
     mapped status — not a 500, not a silent success.
   - Same proof for the Booking-payable path: a user who is not the
     Booking's owning Host attempts `RecordOfflinePayment` against it.
   - Document explicitly (PR description + a line added to `HANDOFF.md`'s
     existing Auth cross-cutting item, not a new one) that this proves the
     *object-level* check given a claimed `ActorUserID`, not real
     authentication — same caveat T5.5/T7.7 already established, must not
     be contradicted or re-litigated here.
2. Non-functional requirements:
   - No real authentication work in this ticket — same boundary T5.5/T7.7
     held.
   - Verify the test actually fails if the ownership check is removed
     (CLAUDE.md rule 10), same as T5.5/T7.7's own verification.

**Story points:** 3

**Labels:** `sprint:t8`, `role:qa`, `type:story`, `points:3`

---

### T8.6 — Extend Game/Registration domain: `PaymentMethod`, `GuestAllowance`, `GuestCount` (pure domain, TDD)

**Story:** As a Host, I want to set a payment method and a guest allowance
when I create a Game, and as a Player, I want to bring guests when I
register, so that Flow 3's and Flow 4's real, named fields have an actual
domain home instead of being UI controls with nothing behind them.

**Description:** The backend-gap ticket the re-scope notice above exists
because of. Pure domain, framework-free, TDD-first, no proto/adapter work
in this ticket (T8.7 wires it). The load-bearing rule this ticket must get
right: guest count changes what "capacity" means.

**Instructions:**
1. Functional requirements:
   - Write failing table-driven tests first (CLAUDE.md rule 1).
   - `domain.PaymentMethod` (new type, `internal/socialplay/domain`):
     `online | cash | either`. Add `PaymentMethod PaymentMethod` and
     `GuestAllowance int` (>= 0, 0 means no guests permitted) to
     `domain.Game`. `NewGame(...)` gains these as parameters, validates
     `PaymentMethod` is one of the three valid values
     (`domain.ErrInvalidPaymentMethod`) and `GuestAllowance >= 0`
     (`domain.ErrInvalidGuestAllowance`).
   - Add `GuestCount int` to `domain.Registration` (>= 0). `domain.Register`
     gains a `guestCount int` parameter, validates `0 <= guestCount <=
     game.GuestAllowance` (`domain.ErrGuestAllowanceExceeded` if over).
   - **The capacity invariant changes shape.** `domain.Register`'s existing
     capacity check (count of active registrations `>= game.Capacity`)
     must become a *weighted* count: `sum(1 + r.GuestCount for r in active
     registrations) + (1 + guestCount) > game.Capacity` is the rejection
     condition, replacing the old `count >= game.Capacity` check. Required
     test: a `Capacity: 3` Game where one existing registration already
     has `GuestCount: 2` (weight 3, already at capacity) rejects a second
     registration with `GuestCount: 0` — this is the test that would pass
     under the *old* (count-based) logic and must fail under the *new*
     (weight-based) logic, i.e., write it expecting `ErrGameFull` and
     confirm it would NOT have failed pre-change (PE dossier §4 — prove the
     test is real, not decorative).
   - Update the existing capacity test suite (T5.2's original tests) to
     confirm the `GuestCount: 0` case still behaves identically to before
     — this ticket must not silently change behavior for the common case
     where nobody brings guests.
2. Non-functional requirements:
   - Zero non-stdlib imports in `internal/socialplay/domain` (unchanged
     rule, re-verify).
   - Table-driven boundary tests: `GuestAllowance == 0` (no guests
     permitted, `GuestCount > 0` rejected), `GuestCount` exactly at the
     allowance boundary (allowed), one over (rejected), invalid
     `PaymentMethod` string.
   - Add `Payment Method` (Game-level field, not to be confused with the
     existing `Registration.PaymentStatus`) and `Guest` terms to
     `agent-operating-handbook.md` A2's glossary in this ticket's PR
     (CLAUDE.md rule 7), explicit about the distinction from
     `PaymentStatus` so the two don't get conflated by a future reader.

**Story points:** 3

**Labels:** `sprint:t8`, `role:principal-engineer`, `type:story`, `points:3`

---

### T8.7 — Wire T8.6's new fields through proto + Postgres + gRPC, extending the capacity guard

**Story:** As a client (Vue this sprint), I want `CreateGame` and
`RegisterForGame` to accept payment method, guest allowance, and guest
count, so that T8.6's domain rules are reachable outside Go tests — and as
the platform, I want the DB-level capacity guard to enforce the same
*weighted* rule the domain now does, not the old count-based one.

**Description:** Depends on T8.6. Mirrors T5.4's shape (adapter/infra
ticket following a domain-only ticket), with one addition T5.4 didn't need:
this ticket must update an *existing* DB-level guard
(`db/migrations/0006_socialplay_capacity_guard.sql`, added T5.4-loop-2 —
see `HANDOFF.md`'s T5.4 entry), not just add a new column, because T8.6
changed what "at capacity" means.

**Instructions:**
1. Functional requirements:
   - Add `db/migrations/0012_socialplay_guest_capacity.sql`:
     `ALTER TABLE games ADD COLUMN payment_method text NOT NULL DEFAULT
     'either'` (a safe default for pre-existing rows — document why
     `'either'` specifically: it's the least restrictive value, matching
     "unspecified" more closely than either single-method value would) and
     `ADD COLUMN guest_allowance integer NOT NULL DEFAULT 0`;
     `ALTER TABLE registrations ADD COLUMN guest_count integer NOT NULL
     DEFAULT 0 CHECK (guest_count >= 0)`.
   - **Rewrite the existing capacity-guard trigger function** (the one
     T5.4-loop-2 added) to sum `1 + guest_count` across active
     registrations for the Game being locked, rather than `COUNT(*)`,
     comparing against `games.capacity`. Required proof, per the same
     standard T5.4-loop-2 itself was held to: a concurrency test (N
     simultaneous `RegisterForGame` calls with varying `guest_count`
     values against a Game sized so exactly one combination fits) showing
     the DB-level guard rejects the same set the domain-level check would,
     run more than once including a cold start (CLAUDE.md rule 10) — don't
     assume the rewritten trigger is correct by inspection alone, the same
     lesson T5.4-loop-2's own PR description already states in full.
   - Update `proto/pickleball/socialplay/v1/socialplay.proto`:
     `CreateGameRequest`/`Game` gain `payment_method` (new
     `PaymentMethod` enum, mirroring `PayableType`'s style) and
     `guest_allowance`; `RegisterForGameRequest`/`Registration` gain
     `guest_count`. Run `make generate`.
   - Update sqlc queries + `internal/socialplay/adapter/postgres` for the
     new columns, following the existing `fromFields`-per-query pattern.
2. Non-functional requirements:
   - Any new Postgres constraint violation (e.g. the rewritten trigger's
     rejection) is translated to `domain.ErrGameFull` in the adapter — same
     mapping already established, no new error type introduced at the
     wire level (CLAUDE.md rule 5).
   - `make down && make up` after the schema change; confirm existing
     T5/T6 integration tests referencing `games`/`registrations` still pass
     against the rewritten trigger (regression check — this ticket changes
     a tested invariant's implementation, not just adds a column).
   - `make test` green, including the `-tags=integration` suite.

**Story points:** 5

**Labels:** `sprint:t8`, `role:principal-engineer`, `type:story`, `points:5`

---

### T8.8 — Vue: Social Game creation flow (Host/Owner)

**Story:** As a Host or Facility Owner, I want to publish a Social Game at
one of my (or any) real Facility's courts, with a real payment method and
guest allowance, so that "configure and publish a social game" (Flow 3,
design handoff README) is a real, working flow rather than a wireframe.

**Description:** Depends on T8.1 (routing), T8.2 (real courts to pick
from), T8.3 (real Facility linkage), and T8.7 (the fields this form
actually submits). Implements Flow 3's "Court & time → Capacity & format →
Rules → Guests → Matching & publish" step order **minus the Matching
step**, per the kickoff note's explicit, documented omission (no
Match/PlayerRating backend exists to attach it to).

**Instructions:**
1. Functional requirements:
   - A multi-step form at `/games/new`: Facility & court picker (calls
     `ListFacilities` then T8.2's courts endpoint — reuse T7.5's existing
     facility-browse components where the UI shape overlaps rather than
     duplicating a second facility picker from scratch) → date/time &
     capacity → payment method (3-way selector: Online / Cash / Either,
     per Flow 3's "Key fields") → guest allowance (stepper) → review &
     publish, calling `CreateGame` (T8.7's extended request) on submit.
   - **The "Matching & publish" step from Flow 3 becomes just "Publish"** —
     no auto-match toggle, level-range slider, or gender-mix selector
     anywhere in this form. Add a one-line note on the review step
     ("Automated matching isn't available yet — players join directly")
     rather than a dead control, mirroring how T7.4 handled the
     not-yet-built pricing status with an honest "coming soon" note instead
     of a non-functional button.
   - Error handling: a failed `CreateGame` (e.g. capacity <= 0, unknown
     `venue_facility_id`) surfaces the specific invalid field in text next
     to that field (WCAG 3.3.1, same rule T7.4 already applied).
   - Once a Game is successfully created, the `RoleIndicator` (T7.1)
     recognizes "Host" as an available role for this browser session per
     the kickoff note's decision #1.
2. Non-functional requirements:
   - Responsive across all three breakpoints, matching T7.4's established
     pattern (single-column stacked on iPhone, multi-column on web).
   - WCAG 4.1.3 Status Messages: a successful "Game published" confirmation
     via an ARIA live region (`role="status"`), same rule T7.4/T7.6 already
     applied.
   - Touch targets >= 44px on iPad/iPhone layouts.
   - Component tests (Vitest): the omitted-matching-step note renders (and
     no matching controls render at all — assert their absence, not just
     that other things are present), payment-method selection is required
     before submit, guest-allowance stepper respects a minimum of 0, and
     forward/back state retention across steps (same requirement class
     T7.4 already had for its own multi-step form).

**Story points:** 8

**Labels:** `sprint:t8`, `role:ux-ui-designer`, `type:story`, `points:8`

---

### T8.9 — Vue: Discover & Join Games flow (Player)

**Story:** As a Player, I want to browse and search real Social Games,
review one, and join it (with guests, paying online or marking cash), so
that "find and join social games" (Flow 4) is a real, working flow.

**Description:** Depends on T8.1, T8.2, T8.3, T8.7 (needs real games with
real facility/payment-method/guest data to browse). Implements Flow 4's
"Browse & filter → Review game → Join + guests" steps; the flow's final
step ("Play & level up... result updates Level") requires the
`Match`/`PlayerRating` aggregate that doesn't exist (same gap as T8.8's
omitted matching step) and is explicitly out of scope here too.

**Instructions:**
1. Functional requirements:
   - A games list/search view at `/games` (facility/date filter — a
     `facility_id`/date-range filter against a `ListGames`-shaped read is
     sufficient for T8; if no such list RPC exists yet, add a minimal one
     in this ticket following T7.3/T8.2's established
     migration-free-read-path pattern, since `socialplay.proto` currently
     has no list method at all — confirm by inspection before assuming one
     exists) and a game detail/review view (host, court, facility, spots
     left, price-or-"Cash at facility" per the kickoff note's decision #3,
     matching Flow 4's "Key fields").
   - Join flow: guest-count stepper (bounded by the Game's
     `GuestAllowance`, mirroring T8.6's domain rule client-side as a UX
     nicety — the server-side check is still authoritative, this is not a
     substitute for it), calls `RegisterForGame` (T8.7's extended
     request). A capacity-exceeded rejection (mapped `ErrGameFull`, 409)
     surfaces a specific, actionable message ("This game just filled up")
     and offers `JoinWaitlist` (live since T6.6) as the next action, per
     the same WCAG 3.3.3 Error Suggestion rule T7.6 already applied to
     Booking's double-booking conflict.
   - Empty/loading/error states for the list and detail views (no games
     yet; API unreachable; a facility with zero games) — real UI for each,
     not a blank screen.
2. Non-functional requirements:
   - Responsive across all three breakpoints.
   - WCAG 1.4.1 Use of Color: the "Cash at facility" vs. price label and
     any spots-left urgency indicator carry text, not color alone.
   - Component tests (Vitest): guest-count stepper respects
     `GuestAllowance`, the game-full/waitlist-offer path, and empty/
     loading/error states.

**Story points:** 8

**Labels:** `sprint:t8`, `role:ux-ui-designer`, `type:story`, `points:8`

---

### T8.10 — Vue: Payments UI — online checkout + cash "pending" surfacing

**Story:** As a Player, I want to pay for my Game registration online
through a real checkout step, and as a Host, I want to see which of my
Games' cash registrations are still unpaid, so that Flow 6 (Payments) is a
real, working flow end to end for both payment methods.

**Description:** Depends on T8.1 and T8.9 (needs a real Registration to
attach a Payment to). Wires the already-real `CreateOnlinePayment`/
`ConfirmOnlinePayment` (Stripe-stub, live since T6.2/T6.4) and
`RecordOfflinePayment` (live since T6.3/T6.4) to a UI for the first time —
mirrors T7.6's role as "first end-user-facing consumer of [a context's]
API," this time for Payments rather than Booking.

**Instructions:**
1. Functional requirements:
   - From T8.9's join flow, if `Game.PaymentMethod` is `online` or
     `either` and the Player chooses to pay now: a checkout step at
     `/games/:id/checkout` calls `CreateOnlinePayment`, then
     `ConfirmOnlinePayment` (Stripe-stub — no real card form; per the PCI
     guardrail (CLAUDE.md rule 11), this ticket does not add any
     card-shaped input field, consistent with `stripestub.Processor`
     taking no card data either — a "Confirm payment (stub)" button is
     sufficient, do not fabricate a fake card-entry UI that would
     misleadingly imply PCI scope this project explicitly avoids).
   - If `Game.PaymentMethod` is `cash` or `either` and the Player chooses
     cash: no Payment is created client-side at all (the existing
     `RecordOfflinePayment` flow is a Host/Game-Admin action, not a
     Player-initiated one, per T6.3's original scope) — the Registration
     simply shows "Payment: pending (cash at facility)" until a Host
     records it.
   - A new Host-facing `/host/payments` view: lists this Host's Games'
     Registrations where `PaymentStatus == unpaid` and `method == offline`
     (reads via the existing `Registration.PaymentStatus` projection,
     T6.5), with a "Mark paid" action calling `RecordOfflinePayment`. A
     count badge in the nav (T8.1's shell) shows the pending count — this
     is the kickoff note's decision #3 (dashboard badge + list, no push
     notification).
   - Booking confirmation-style review step before the online-payment
     `ConfirmOnlinePayment` call fires (WCAG 3.3.4 Error Prevention,
     same rule T7.6 already applied to `CreateBooking`).
2. Non-functional requirements:
   - No card data, PAN, CVV, or track-data field anywhere in this ticket's
     UI or any request it sends — review against
     `docs/checklists/proto-review.md` even though this ticket adds no new
     proto messages, since it's the first UI consumer of the existing
     Payments proto and the guardrail's spirit ("card data is tokenized
     client-side... and never reaches this backend") applies to the UI
     layer just as much as the wire format.
   - A payment-success confirmation exposed via an ARIA live region (WCAG
     4.1.3, same rule class as T8.8/T7.4/T7.6).
   - Responsive across all three breakpoints; the Host pending-payments
     list follows the same list/detail responsive pattern T7.5 already
     established.
   - Component/integration tests (Vitest, mocking the generated client):
     the online happy path (checkout -> confirm -> paid), the cash
     "pending" display path, the Host mark-paid action, and the confirm-
     step gate (asserts `ConfirmOnlinePayment` is never called before the
     review step is shown, mirroring T7.6's own confirm-gate test).

**Story points:** 8

**Labels:** `sprint:t8`, `role:ux-ui-designer`, `type:story`, `points:8`

---

## Sprint totals

- **Tickets:** 10 (T8.1–T8.10)
- **Total story points:** 51 (5 + 3 + 5 + 3 + 3 + 3 + 5 + 8 + 8 + 8)
- **Composition, for calibration against T5 (27/5 tickets), T6 (37/7), T7
  (40/7):** roughly 22 points (T8.1–T8.6) are prerequisite/gap-closure work
  (routing, two Facilities endpoints, a carried-over QA ticket, and the
  domain+adapter split for the newly-found Game/Registration field gap),
  and 29 points (T8.7 is counted above as prerequisite since it's
  wiring/infra, not a user-facing screen — recount: T8.8+T8.9+T8.10 = 24
  points) are new, user-facing screens — a comparable ratio to T7's own
  19-points-prerequisite / 21-points-UI split (T7.1–T7.3+T7.7 vs.
  T7.4–T7.6). The larger total than T5/T6/T7 is explained by T8 carrying
  more prerequisite debt into it than any prior sprint inherited (three
  separate already-flagged T6/T7 gaps plus one newly-found one), not by
  attempting more new *product* surface than T7 did.
- **Design-review / open-question items resolved this ceremony:** #3
  (cash-vs-online payment surfacing prominence) gets a real decision and
  implementation (T8.10); #2 (auto-matching transparency) is re-examined
  and found to have no backend to attach to at all (a stronger, more
  specific finding than T7's "no screen yet" deferral) — logged as a T9-
  Ceremony-1 input, not silently re-deferred with the same reasoning as
  before.
- **Genuine disagreements recorded (not manufactured consensus):** PE's
  "innovation tokens" framing (three categories of new/unproven surface —
  Competitions, OAuth linking, a bot integration — shouldn't stack in one
  sprint) drove the re-scope, argued from the roadmap's own prior
  reasoning rather than freshly invented; PM's push to include T6.7 (the
  outstanding Payments-authz ticket) despite PE's initial preference to
  keep T8's total down — PM's position prevailed (T8.5).
- **Explicitly deferred, not forgotten:** Competitions context + Discover/
  Advertise UI (Flow 5), social-account-linking, shareable-registration-
  links, WhatsApp/Zalo owned-channel-bot spike (→ T9, renumbered from the
  T7 plan's T8 sub-scopes c/d/e); a decision on whether/when to build real
  auto-matching (→ T9 Ceremony 1, newly flagged, not previously scheduled
  anywhere); `CancellationCutoff` enforcement (→ raise at T9/T10 backlog
  refinement once there's a concrete driver); `discount_rules` backend +
  pricing UI, Club account type + rental flow, cross-sprint WCAG audit (→
  T10, renumbered from the T7 plan's T9); native Swift/Kotlin mobile apps
  (→ T10+ per T7's own deferral, unchanged); the `npm install
  --legacy-peer-deps` chore and Jenkins CI wiring `HANDOFF.md`'s
  Cross-cutting section already names (→ not picked up this sprint either,
  still open — flag again at T9 refinement if a third sprint passes
  without them, per the same T5-retro-finding-6 discipline this sprint
  applied to T6.7).
