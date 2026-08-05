# Agent Operating Handbook — Pickleball Platform

This handbook is the shared reference for every agent (human or Claude subagent)
working on this codebase. **Part A** is the shared foundation everyone loads.
**Part B** is a set of role briefs — load only the brief(s) relevant to the task
at hand, in addition to Part A.

---

## Part A — Shared foundations

### A1. Bounded contexts

The platform is decomposed into the following bounded contexts. Each maps to
`internal/<context>/{domain,app,port,adapter}` and its own
`proto/pickleball/<context>/v1` package. A context owns its own aggregates and
never reaches into another context's tables directly — cross-context
collaboration happens through application services calling other contexts'
ports, or through domain events, never through shared row access.

| Context | Owns | Depends on (via ports, never directly) |
|---|---|---|
| **Booking** | The polymorphic `Booking` aggregate, the no-double-booking invariant, availability, cancellation | Pricing (for quotes) |
| **Pricing** | `PricingRule`s and price resolution for a (court, slot) | — |
| **Facilities** | `Facility`, `Court`, court attributes, geo search | — |
| **Social Play** | `Game`, `Registration`, matchmaking, `Match`, `PlayerRating` | Booking (a Game reserves courts by creating `game`-source Bookings) |
| **Competitions** | `Competition`, brackets/rounds, competition results | Booking (a Competition reserves courts the same way a Game does) |
| **Payments** | `Payment` aggregate, the payment-status state machine, Stripe anti-corruption layer, offline recording | Booking, Social Play, Competitions, Facilities/Clubs (whatever is being paid for) |
| **Statements** | Owner/host statement aggregation | Payments, Booking, Facilities |
| **Identity/Users** | `User`, roles (player, host/organiser, game admin, facility owner, club, platform admin), self-reported starting level | — |

**The spine.** Booking, Pricing, and Facilities are the shared spine every other
context sits on (spec §9). Build and hardened first; every later context reuses
them rather than reinventing reservation or pricing logic.

### A2. Ubiquitous language (glossary)

Use these terms identically in Go identifiers, SQL columns, proto fields, and
UI copy. If a new term is needed, add it here in the same PR that introduces it.

- **Booking** — the single aggregate representing *any* reservation of a
  Court for a time range. Polymorphic over `source`:
  `recurring_hire | individual | game | competition`. This is the one and
  only mechanism that occupies a court — see D3b. A Booking has a `during`
  range `[starts_at, ends_at)`, a `status` (`confirmed | cancelled`), and an
  optional link to the Game/Competition/Club it belongs to.
- **No-double-booking invariant** — no two non-cancelled Bookings on the same
  `court_id` may have overlapping `during` ranges. Enforced authoritatively by
  a Postgres `EXCLUDE` constraint and mirrored in the domain as
  `domain.EnsureNoConflict` for fast unit-level pre-checks. Back-to-back
  bookings (one ends exactly when another starts) are **not** a conflict —
  ranges are half-open `[start, end)`.
- **Facility** — a physical venue owned by an Owner, containing one or more
  Courts. (T7.2, `internal/facilities/domain`) Fields: `ID`, `OwnerID`,
  `Name`, `Description`, `Address`, `PhotoURLs` (`[]string`), `CameraLinks`
  (`[]CameraLink`), `CameraConsentAttested` (`bool`, always starts `false` —
  see Camera Link below). `OwnerID`/`Name`/`Address` are required
  non-empty; `Description`/`PhotoURLs`/`CameraLinks` may be empty.
- **Court** — a single playable surface within a Facility; the unit that
  Bookings reserve. (T7.2, `internal/facilities/domain`) Fields: `ID`,
  `FacilityID`, `Name` — deliberately minimal, with no pricing fields;
  pricing for a Court lives entirely in the existing `pricing_rules` table
  keyed by `court_id` (T1), untouched by T7.2. `FacilityID`/`Name` are
  required non-empty.
- **Camera Link** — a link to a Facility's live camera feed, either
  facility-wide (`CourtID` empty) or scoped to one Court (`CourtID` set).
  (T7.2, `internal/facilities/domain`) Fields: `URL`, `CourtID`. May only be
  added to a Facility once that Facility's `CameraConsentAttested` is
  `true` — `CameraConsentAttested` defaults to `false` and there is no
  constructor path that sets it `true` on creation, encoding the round-10
  design review's finding that the consent checkbox must default to
  unchecked (`docs/design/v1-review-round-10-final.md` §2b) as a domain
  invariant, not just a UI default.
- **Pricing Rule** — a rule attached to a Court (or inherited from its
  Facility) that resolves to a price for a given slot based on day-of-week /
  time window / weekend band.
- **Quote** — the result of resolving Pricing Rules for a specific
  (court, slot): a price plus which rule produced it.
- **Game** — a social play session an Organiser (Host) creates at a Facility,
  on specific Courts and a time range, with a capacity. Reserves its courts by
  creating `game`-source Bookings, never a separate mechanism.
- **Registration** — a Player's signup for a Game; tracks `source`
  (`app | social`), payment status, and `GuestCount` (T8.6, see Guest below).
- **Payment Method** — a Game-level field (`domain.Game.PaymentMethod`,
  T8.6): `online | cash | either`, set once by the Host at scheduling time
  (`NewGame`), describing which payment method(s) the Host accepts for that
  Game's registrations. **Not to be confused with `Registration.PaymentStatus`**
  (`unpaid | paid | refunded`, see Payment above) — PaymentMethod is a
  Game-level policy statement set by the Host once; PaymentStatus is a
  per-Registration fact, reported by the Payments context, that changes over
  a Registration's lifetime. A Game's PaymentMethod does not itself enforce
  which Payments RPC (`CreateOnlinePayment`/`RecordOfflinePayment`) the
  backend will accept — it is UI guidance only until a real enforcement
  mechanism (a `port.GamePaymentPolicy`-shaped addition) is designed; see
  `docs/process/t8-sprint-plan.md`'s T8.6 kickoff note.
- **Guest** — a non-registered player a Registration's owner brings along.
  `domain.Game.GuestAllowance` (T8.6) is the Host-set maximum guests any
  single Registration against that Game may bring (0 = no guests permitted);
  `domain.Registration.GuestCount` (T8.6) is how many guests that specific
  Registration actually brings, validated against `GuestAllowance`
  (`0 <= GuestCount <= GuestAllowance`). Guests occupy capacity slots the
  same as the registering Player: the Game-full check is a *weighted* count
  — `sum(1 + GuestCount)` across active Registrations — not a plain
  registration headcount, since a single Registration with enough guests can
  fill a Game's capacity on its own.
- **Host / Organiser** — the user who owns and created a Game or Competition.
- **Game Admin** — a role assigned per-Game (or per-Competition) by the Host;
  manages/directs players on the day and may record offline Payments. Scoped
  to the specific Game/Competition they are assigned to — distinct from the
  Host and from the platform Admin.
- **Match** — a single pairing of players within a Game or Competition, with a
  recorded score, feeding `PlayerRating`.
- **PlayerRating** — a player's DUPR-style internal rating, derived from
  Match history; seeded by `self-reported starting level` before any history
  exists (cold-start).
- **Self-reported starting level** — a value a Player sets on their profile at
  signup, used as the matchmaking input until real rating history accrues.
- **Competition** — an Organiser-run tournament a Host creates at a Facility;
  reserves courts the same way a Game does (`competition`-source Booking),
  inheriting the no-double-booking invariant with no change to Booking.
  (T9.1, `internal/competitions/domain`) Fields: `ID`, `HostID`, `Name`,
  `VenueFacilityID` (optional, same semantics as `Game.VenueFacilityID`),
  `Sessions`, `Capacity`, `GuestAllowance`, `PaymentMethod`, `EntryFee`
  (`Money`; a zero amount is a real value meaning a free competition, not
  "unset"), `Format`, `Status` (`scheduled | cancelled`), `ShareToken`.
  **How it differs from a Game:** a Game is a single sitting (one time
  range, one set of courts); a Competition runs across **one or more
  Sessions**, potentially on different dates. Brackets, rounds, seeding,
  match scheduling and results are **deliberately absent in T9** — the
  reservation-and-entry spine ships first and still produces a complete
  loop (create → advertise → enter → roster); see
  `docs/process/t9-sprint-plan.md` §A4. Cancelling a Competition flips only
  its own status; the cascade to its Bookings and entries is a known gap,
  the same one `Game.Cancel` has.
- **Session** — one sitting of a Competition: a time range plus the Courts
  it reserves for that range (T9.1). A value type inside the Competition
  aggregate — no identity and no lifecycle of its own. A Competition
  requires at least one; two Sessions of the same Competition may not
  reserve the same Court at overlapping times (rejected at construction,
  since the Booking-level invariant would otherwise only catch it part-way
  through reserving them). Back-to-back Sessions on one Court are **not** an
  overlap — ranges are half-open `[start, end)`, as everywhere else.
- **Competition Entry** — a Player's claim on a Competition's places; the
  Competitions analogue of a Registration (T9.1). Fields: `ID`,
  `CompetitionID`, `PlayerID`, `GuestCount`, `Source` (**Entry Source**,
  below), `PaymentStatus` (`unpaid | paid | refunded`, modelled in T9 with
  Payments integration deferred), `Status` (`entered | cancelled`). Guests
  occupy places exactly as the entrant does: the competition-full check is a
  *weighted* count — `sum(1 + GuestCount)` across active Entries — never a
  plain entry headcount, and it is weighted from the first commit rather
  than retrofitted (contrast Guest above, where T8.6 had to reweight Social
  Play's rule after the fact).
- **Entry Source** — how a Competition Entry came to exist: `app | social`
  (T9.1). Deliberately the **same two values** `Registration.source` already
  uses, not a new `shared_link` vocabulary — one ubiquitous language
  (`docs/process/t9-sprint-plan.md` §A3).
  **`social`** means, precisely: *the entering client declared, on its
  `EnterCompetition` request, that this entrant arrived via a Shareable
  Registration Link (below) rather than through in-app browsing.* Three
  things that definition deliberately excludes, each of which someone will
  otherwise assume:
  1. It is **a client's declaration the server validates, never a server
     inference.** The backend checks the value against the closed enum and
     stores it; it does not derive it from any other signal. In particular
     resolving a Competition by its share token does **not** make a
     subsequent entry `social` — how a client reached a Competition is not
     something the backend can observe, and a guess rendered as a fact on a
     Host's roster is worse than no attribution (T9.5).
  2. It says nothing about **which** channel. There is no Facebook/WhatsApp/
     Instagram distinction anywhere in the model, because the platform never
     touches those channels (see Shareable Registration Link).
  3. It is not evidence the link was *used* — an entrant who follows a link
     and then enters declaring `app` is recorded as `app`. `social` is a
     self-reported attribution, useful in aggregate, not an audit trail.
  `app` means the entrant found the Competition inside the platform. An
  unset/unspecified value on the wire resolves to `app` (an unaware client
  is by definition an in-app one); an *unrecognised* value is rejected 400,
  never defaulted. **Producer:** T9.1 modelled both values, but until T9.5
  nothing produced `social` — Social Play's identically-named
  `Registration.source` value has still had no producer since T5.2. T9.5's
  share-link flow is the first real producer of `social` in this codebase.
- **Shareable Registration Link** — the link a Host posts anywhere they like
  (a group chat, a club page, a poster QR code) that takes a Player straight
  to entering their Competition (T9.5). Concretely it is a URL carrying that
  Competition's **share token**: an opaque, URL-safe, `crypto/rand`-generated
  value (256 bits, `base64.RawURLEncoding` — `internal/competitions/adapter/
  sharetoken`, T9.4) minted for every Competition at creation and stored in
  `competitions.share_token` (`NOT NULL UNIQUE`). It is disclosed exactly
  once, to the Host, on `CreateCompetitionResponse`; it is deliberately not a
  field on the `Competition` message, so it cannot leak through the
  unauthenticated read paths.
  The token is a **capability, not an identifier**: possession of it is what
  grants the read (`GetCompetitionByShareToken`), which returns exactly the
  same public projection `GetCompetition` returns — a link never discloses
  more than the app already shows. Treat it like a secret: never log it,
  never put it in an analytics event.
  **Outbound only.** The platform publishes a link and nothing more. It does
  not read, scrape, poll, or ingest anything from the channel the link was
  posted to — no reply parsing, no third-party platform API reads, no
  webhooks. Every entry arrives through this platform's own
  `EnterCompetition` API, exactly as an in-app entry does; inbound social
  integration is deferred (`docs/adr/0009-social-channel-integration-deferred.md`).
  Two properties a reader must not assume otherwise:
  * A **cancelled** Competition's link still resolves, returning the
    Competition with `status: cancelled` rather than a not-found — a link
    already published outlives the Competition's scheduled state, and "this
    competition was cancelled" and "this link is broken" are different facts.
    Entering it is separately rejected.
  * **There is no revocation.** T9 ships no rotate-token or revoke-token
    path: once published, a link stays valid for the Competition's lifetime,
    and deleting the post does not invalidate it. This is blocked on real
    authentication (rotation is an act only a Host may perform, and
    `actor_user_id` is still an unverified claim), not merely unfinished —
    build it alongside auth.
- **Format** — the play format a Host advertises for a Competition:
  `singles | doubles` (T9.1). **Descriptive, not enforcing** — nothing
  validates entry counts or pairings against it, and partner pairing is not
  modelled in T9 at all. This does not contradict T8's removal of
  `CancellationCutoff` ("a cosmetic field that enforces nothing is actively
  misleading"): the test is whether a field *implies enforcement*. A
  cancellation cutoff is a rule a Host asserts over other people's
  behaviour, so a silent no-op is a broken promise; a format label is
  information for players, complete and honest the moment it is displayed.
- **Waitlist Entry** — a Player's queued position (`waiting | promoted |
  expired | cancelled`) for a Game that was full when they tried to join
  (T6.6, fulfilling ADR-0006). Ordered FIFO per Game. A cancelled
  Registration auto-promotes the oldest `waiting` entry; a promoted entry
  reserves its freed slot for a bounded response window
  (`domain.PromotionResponseWindow`) before expiring and cascading the
  promotion to the next entry. Scoped to Games only in T6 — a standalone
  court/slot waitlist is a distinct, deferred concept (see ADR-0006's
  Status section).
- **Club** — an entity that can hire specific Courts on a recurring schedule
  (`recurring_hire`-source Booking generated from a recurring template).
- **Payment** — the single aggregate recording money for any payable action
  (a Booking, a Registration, a recurring hire, a subscription). Has a
  `PaymentStatus` state machine: `unpaid → paid → refunded` (illegal
  transitions rejected). One `payments` row per payable action, regardless of
  online or offline mode — this is the single source of truth referenced by
  rosters, owner views, and statements.
- **Owner** — the user who owns a Facility and receives payouts for its
  Bookings/recurring hire.
- **Statement** — an aggregation over Payments for a period, scoped to an
  Owner (per-facility, monthly) or a Host (per-game).
- **Anti-corruption layer (ACL)** — the boundary translating an external
  system's model (e.g. Stripe's) into this platform's domain model, so the
  domain never depends on a third party's types or error shapes.

### A3. Architectural rules (see also root `CLAUDE.md`)

1. TDD: failing table-driven test first, minimum code to pass, then refactor.
2. Domain packages (`internal/<context>/domain`) are framework-free — no pgx,
   grpc, or third-party imports beyond the standard library.
3. Dependency rule points inward: `adapter → app → domain`. Never the reverse.
4. Every invariant is enforced in Postgres **and** expressed in the domain.
5. Adapters translate infrastructure errors into domain errors; upper layers
   never see a raw pgx/Stripe/grpc error.
6. Generated code (`internal/gen/**`, from `make generate`) is never hand
   edited — change the `.proto`/`.sql` source and regenerate.
7. One ubiquitous language (§A2) across DB, Go, proto, and clients.
8. `make test` green before any task is called done; every bug becomes a
   regression test.

### A4. Industry standards checklist (apply every phase)

DDD bounded contexts/aggregates/invariants with a pure domain · TDD red-green-
refactor with table-driven tests for every invariant · Pragmatic Programmer
(DRY, orthogonality, tracer bullets, no broken windows, reversible decisions,
good-enough software) · Fowler refactoring under green tests · Google-style
mandatory readable-code review and a small/medium/large automated test
culture, with a short design doc before non-trivial work · Meta-style strong
tooling, metrics-informed choices, small flagged increments · Uber-style clear
service/domain boundaries, reliability/observability as first-class, RFCs for
cross-cutting decisions · Microsoft-style security-by-design on money/auth
paths, backward compatibility, engineering rigour · Canva-style high UX/DX bar
and safe incremental delivery · ThoughtWorks-style evolutionary architecture
with fitness functions, continuous delivery, trunk-based development, and a
Tech-Radar mindset when picking tools.

---

## Part B — Role briefs

Each brief states the role's mandate, what it is adversarial toward, and what
"good" looks like from that seat. When phases are debated, **do not manufacture
consensus** — a role that disagrees should say so and the disagreement should
be recorded in the phase's review doc.

### B1. Principal Engineer (PE)

**Mandate:** own technical soundness and the cost of decisions over time.
**Adversarial toward:** unnecessary complexity, one-way doors taken too early,
leaky abstractions, and anything that violates the dependency rule or mixes
infrastructure into the domain.
**Checks every phase against:** Is this the simplest design that satisfies the
invariant? Is it reversible? Does it respect bounded-context boundaries? Would
this survive a real code review at a company with a strong engineering bar?
**Escalates when:** a decision is genuinely hard to reverse (e.g. a schema
choice, a public API shape, taking real payments) — flag it as a one-way door
explicitly rather than let it pass quietly.

### B2. Product Manager (PM)

**Mandate:** protect product value and market timing.
**Adversarial toward:** scope creep that delays validating the core loop(s),
and technical purity pursued past the point of user value.
**Checks every phase against:** does this move the platform closer to a
loop real users can complete end-to-end? Is effort going to differentiating
features (matchmaking, paid games reducing no-shows) over incidental polish?
**Escalates when:** a phase's scope silently expands beyond its stated AC, or
a technically "correct" choice meaningfully delays shipping value.

### B3. Product Engineer (PdE)

**Mandate:** the builder's hat — turn the plan into working, shipped
increments; keep velocity honest.
**Adversarial toward:** half-finished implementations, gold-plating, and
plans that sound clean but are expensive to actually build solo.
**Checks every phase against:** can this actually be built and shipped in the
time implied? Is there a cheaper path to the same acceptance criteria?
**Escalates when:** an approach looks elegant on paper but is a trap in
practice (e.g. an abstraction that needs three contexts to prove itself but
only one exists yet).

### B4. QA

**Mandate:** actively try to break the result before it ships.
**Adversarial toward:** untested edge cases, invariants asserted but not
proven, and "happy path only" implementations.
**Checks every phase against:** concurrency (can two conflicting operations
race?), boundary conditions (back-to-back ranges, zero-capacity, empty
inputs), illegal state transitions, and whether every invariant in the spec
has an explicit failing-then-passing test.
**Escalates when:** a claimed invariant has no test that would fail if the
invariant were removed — that's not tested, it's assumed.

### B5. Product Owner (PO)

**Mandate:** own backlog priority and acceptance criteria; the tie-breaker
between PM's value push and PE's technical caution.
**Adversarial toward:** ambiguous "done," and phases that reopen already-locked
decisions instead of working within them.
**Checks every phase against:** does the delivered work meet its literal
Definition of Done in `HANDOFF.md`? Is the next phase correctly informed by
what actually happened in this one?
**Escalates when:** a phase cannot reach alignment within the loop budget, or
requires a product/legal decision (launch market, commission %, refund
policy) that is explicitly out of scope for an engineering session.

### B6. Business Analyst (BA)

**Mandate:** hunt rule and edge-case gaps between the spec's prose and what
the code actually enforces.
**Adversarial toward:** modelling gaps where two spec sections *sound*
consistent but don't actually compose (the Booking/Game gap in
`spec-design-review.md` Topic 2 is the canonical example this role exists to
catch).
**Checks every phase against:** re-read the relevant spec section line by
line against the implementation — does every stated rule have a home in the
code? Are there silent scope narrowings (e.g. "peak" band boundaries,
same-instant edge cases) that need an explicit test or an explicit deferral?
**Escalates when:** a spec rule cannot be implemented as stated without a
product decision — surface the specific ambiguous sentence, don't guess.
