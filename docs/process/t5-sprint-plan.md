# T5 Sprint Plan — Social Play context (skeleton first)

Produced by Ceremony 1 (Backlog refinement) + Ceremony 2 (Sprint planning)
per `docs/process/sprint-process.md`, played jointly by **Product Manager**
(handbook B2) and **Principal Engineer** (handbook B1 + `docs/roles/
principal-engineer.md`). Scope inherited from `HANDOFF.md` T5 and cross-
checked against `docs/requirements/README.md` and `docs/adr/
0006-waitlist-data-model-direction.md` per the task brief.

## Sprint goal

> A Host can create a Game that reserves its courts as `game`-source
> Bookings — inheriting the no-double-booking invariant end to end through
> Postgres and gRPC/REST — and a Player can register for it with capacity
> and paid/unpaid status enforced, including a basic actor-scoped
> authorization check on who may cancel whose registration.

## Kickoff note

**Scope decisions against `docs/requirements/README.md`:**

- **P1 #6 (BOLA/BFLA regression tests)** — in scope. T5 adds Social Play's
  first externally-callable endpoints (`CreateGame`, `RegisterForGame`,
  `CancelRegistration`). There is no JWT/auth system yet (`HANDOFF.md`
  cross-cutting backlog), so a full authz layer is out of reach this
  sprint — but the *object-level* check ("does the acting player own this
  registration?") is a domain-level concern that doesn't need real auth to
  test, exactly like `CancelBooking`'s status-transition guard didn't need
  one. PE and PM agreed this is cheap now and expensive to retrofit later
  (an endpoint shipped without an explicit actor parameter tends to grow
  callers that assume trust), so it's folded into ticket T5.2 and given its
  own regression-test ticket (T5.5) across every new endpoint, mirroring
  the T3 review's "prove it, don't assume it" standard.
- **P0 #3 / ADR-0006 (waitlist data model)** — this is the one **genuine
  PM/PE disagreement** this pass surfaced, recorded per sprint-process.md
  §"do not manufacture consensus":
  - **PM's position:** rejecting a signup outright once a Game is full is a
    real product gap — comparable platforms (per `research-functional.md`
    §1.3) auto-queue overflow signups, and shipping Social Play without
    *any* answer to "I want to play but the game's full" undercuts the
    core loop this sprint is meant to prove.
  - **PE's position:** ADR-0006 already scopes a real waitlist as its own
    entity with an ordered queue, an auto-promotion trigger on
    cancellation/no-show, and a response timeout — that is a second
    aggregate plus event-driven promotion logic, not a field on
    `Registration`. Building it inside a "skeleton first" sprint whose own
    AC (`HANDOFF.md`) is "capacity invariant tested; game scheduling
    creates bookings and respects overlap; registration paid/unpaid status
    modelled" is scope creep that risks *all* of T5 landing in a half-done
    state across two aggregates instead of one aggregate landing solid.
  - **Resolution (not full consensus, but a scoped compromise both signed
    off on):** T5 ships a capacity invariant that rejects overflow
    registration with a distinct, stable domain error
    (`domain.ErrGameFull`, not a generic validation error) specifically so
    a future waitlist ticket can hook a promotion trigger onto that
    boundary without an API-shape change. The actual `waitlist_entries`
    entity, queue ordering, promotion trigger, and response timeout are
    **deferred to T6**, tracked by updating ADR-0006's status line
    (currently "full schema/ticket detail deferred to T5/T6 backlog
    refinement") to point at T6 explicitly. PM accepted this on the
    condition the T6 ticket exists and isn't allowed to silently slip
    again — see ticket T5.5 sub-note.
- **P1 #10/#11 (rating margin weighting, verified-vs-self-reported match
  weighting)** — out of scope, confirmed. `HANDOFF.md` explicitly defers
  matchmaking to a follow-up task after T5, and T5 has no `Match`/
  `PlayerRating` work at all (only `Game`/`Registration`); these findings
  belong with that follow-up, not here.
- **Other P0 items (#1 timezone, #2 currency, #4 GDPR/retention, #5
  split-court modelling)** — considered and left out. None are specific to
  Game/Registration; they affect Booking/Facilities/Users, which T5 doesn't
  touch. Logged here so they aren't silently dropped, not force-fit into
  this sprint.

**Architecture note PE flagged for the tickets below (two-way door, not an
ADR):** Social Play must not import `internal/booking/domain` or call
`internal/booking/app.Service` directly from its own `app` package — the
context map (`agent-operating-handbook.md` A1) requires the dependency to
run through a port. T5.3 defines a small `port.CourtReservation` interface
in `internal/socialplay/port` expressed in primitive types (court ID,
start/end `time.Time`, source string, reference ID) so `internal/socialplay/
domain` and `internal/socialplay/app` never see a `bookingdomain.Booking`
type; an adapter in `internal/socialplay/adapter/booking` implements that
port by calling Booking's real `app.Service.CreateBooking` and translating
`domain.ErrCourtDoubleBooked` into Social Play's own conflict error. This
keeps the boundary honest without inventing an event bus this sprint
doesn't need (Golden Rule 3 / PE dossier §5 "speculative generality").

---

## Tickets

### T5.1 — Add the Game aggregate with a capacity invariant

**Story:** As a Host, I want to define a Game with a fixed player capacity,
so that the platform can guarantee a Game never accepts more registrations
than it has room for.

**Description:** Bootstraps `internal/socialplay/domain` with the `Game`
aggregate — pure, framework-free, TDD-first, mirroring how
`internal/booking/domain/booking.go` shapes the `Booking` aggregate. This is
the first Social Play code; no adapter/proto work happens in this ticket.

**Instructions:**
1. Functional requirements:
   - Write failing table-driven tests first (CLAUDE.md rule 1), then add
     `domain.Game{ID, HostID, FacilityID, CourtIDs []string, Range
     domain.TimeRange, Capacity int, Status}` (reuse the existing
     `TimeRange`/half-open-range semantics shape from booking's domain —
     either via a small shared package or a locally duplicated equivalent
     type; do not import `internal/booking/domain` — see kickoff note).
   - `NewGame(...)` validates: `Capacity` must be `> 0`
     (`domain.ErrInvalidCapacity`); `CourtIDs` must be non-empty
     (`domain.ErrEmptyCourtIDs`); the time range must be valid (reuses the
     same "end after start" rule booking's `NewTimeRange` enforces).
   - `Status` starts `scheduled`; add a `Cancel()` transition
     (`scheduled -> cancelled`) with the same "illegal transition rejected"
     pattern as `booking.Booking.Cancel()` — a Game cancellation is a
     precondition later tickets need (cancelling a Game should eventually
     cancel its Bookings/Registrations, but that cascade is explicitly
     **out of scope for T5** — note it as a follow-up in the PR
     description, don't build it here).
   - Given/When/Then coverage required: capacity of 1 accepts exactly one
     active registration and rejects a second (this test belongs with
     T5.2's `Registration` work since capacity is enforced at registration
     time, not on `Game` alone — `Game` just holds and validates the
     `Capacity` value in this ticket).
2. Non-functional requirements:
   - Zero non-stdlib imports in `internal/socialplay/domain` (CLAUDE.md
     rule 2) — verify with the same import-boundary discipline as the
     Booking context.
   - Table-driven boundary tests: `Capacity == 0`, negative capacity,
     empty `CourtIDs`, zero-duration range.

**Story points:** 3

**Labels:** `sprint:t5`, `role:principal-engineer`, `type:story`, `points:3`

---

### T5.2 — Add the Registration aggregate with capacity enforcement, paid/unpaid status, and actor-scoped cancellation

**Story:** As a Player, I want to register for a Game and see whether my
registration is paid or unpaid, so that I know my spot is confirmed and
whether I still owe money; and as a Player, I want confidence that only I
(or an authorized Game Admin) can cancel my own registration.

**Description:** Depends on T5.1's `Game`. Adds `domain.Registration` and
the capacity-checking function that ties it to `Game.Capacity`. This ticket
is where P0 #3 (waitlist) and P1 #6 (BOLA) land, per the kickoff note.

**Instructions:**
1. Functional requirements:
   - Failing tests first. `domain.Registration{ID, GameID, PlayerID,
     Source (app|social), Status (registered|cancelled), PaymentStatus
     (unpaid|paid)}` — reuse the `unpaid`/`paid` vocabulary from the
     glossary (`agent-operating-handbook.md` A2 `Payment`); T5 does not
     wire real Payments (T6), so `PaymentStatus` here is just a field a
     Game Admin can flip via a later app-layer method — modelling only,
     no Stripe/ACL work.
   - `RegisterForGame`-shaped domain function (name it what reads best,
     e.g. `domain.Register(game Game, existing []Registration, playerID
     string) (Registration, error)`) enforces the capacity invariant:
     counts non-cancelled existing registrations for the Game; if count
     `>= game.Capacity`, return `domain.ErrGameFull` — **this exact error
     type**, not a generic validation error (kickoff note: T6's waitlist
     promotion trigger needs a stable hook).
   - A player already actively registered for the same Game cannot
     register twice — `domain.ErrAlreadyRegistered`.
   - `Registration.Cancel(actorPlayerID string) error`: only legal when
     `actorPlayerID == r.PlayerID`; a mismatched actor returns a new
     `domain.ErrNotRegistrationOwner` rather than silently succeeding or
     falling through to `ErrIllegalStatusTransition`. This is the P1 #6
     BOLA-shaped test: **write a test that asserts Player A cannot cancel
     Player B's registration** — same spirit as the object-level check
     this repo will eventually need on `CancelBooking` too, but that
     retrofit is out of scope here (T3 already shipped without it; flag,
     don't fix, in this PR).
   - Cancelling a `Registration` frees a capacity slot for the next
     `Register` call — test this the way T3 tested cancel-frees-the-slot
     for Bookings (prove it via re-registration succeeding after a cancel,
     not just via the status field).
2. Non-functional requirements:
   - `domain.ErrGameFull` must be distinguishable from
     `domain.ErrAlreadyRegistered` and `domain.ErrNotRegistrationOwner` by
     type (sentinel errors, `errors.Is`-compatible) so callers/adapters can
     map each to a distinct gRPC/REST status in T5.4.
   - No full waitlist queue, promotion, or timeout logic in this ticket
     (ADR-0006 scope deferred to T6 per kickoff note) — resist the urge to
     partially build it; a half-built queue with no promotion trigger is
     worse than no queue (PE dossier §5, "silent technical-debt accrual
     with no trigger to pay it down" — the trigger here is explicitly "T6").

**Story points:** 5

**Labels:** `sprint:t5`, `role:principal-engineer`, `type:story`, `points:5`

---

### T5.3 — Game scheduling reserves courts as game-source Bookings (cross-context integration)

**Story:** As a Host, I want scheduling a Game to actually reserve its
courts, so that a Game can never be double-booked against an existing
individual booking, recurring hire, competition, or another Game.

**Description:** The load-bearing ticket for T5 — this is what makes Social
Play "inherit the no-overlap invariant" rather than merely modelling a
`Game` that has no real effect on court availability. Depends on T5.1.
Deliberately app-layer-only (in-memory fakes), no Postgres/proto in this
ticket — mirrors how Booking's own app-layer cross-source overlap test was
proven before any adapter existed (`HANDOFF.md` "Application service with
an in-memory test proving cross-source overlaps are rejected").

**Instructions:**
1. Functional requirements:
   - Define `port.CourtReservation` in `internal/socialplay/port`, shaped
     around primitives only (per kickoff note's architecture decision):
     e.g. `ReserveCourt(ctx, courtID string, start, end time.Time,
     referenceID string) (bookingID string, err error)`, returning a
     Social-Play-local `ErrCourtUnavailable` (translated from whatever the
     adapter gets back — see T5.4) rather than leaking
     `bookingdomain.ErrCourtDoubleBooked` across the context boundary.
   - `internal/socialplay/app.Service.ScheduleGame(...)`: validates/builds
     the `Game` (T5.1), then calls `port.CourtReservation.ReserveCourt` once
     per court in `Game.CourtIDs` with `Source = "game"` (the string value
     Booking's proto/DB expects — confirm against
     `internal/booking/domain/booking.go`'s `SourceGame` constant's
     underlying value, `"game"`) and `ReferenceID = Game.ID`.
   - **Required test:** using an in-memory fake `port.CourtReservation`
     seeded with an existing overlapping reservation on one of the target
     courts, prove `ScheduleGame` fails with the translated conflict error
     and — critically — that no `Game` is left in a half-scheduled state
     (either all courts reserve or none do; if court 2 of 2 conflicts,
     court 1's reservation must not be left dangling — decide and test
     rollback-or-single-court-games-only as the resolution, whichever is
     cheaper to build correctly; do not ship an unspecified partial-failure
     state).
   - **Required test (the literal HANDOFF AC):** a Game cannot be scheduled
     onto a court that already has a confirmed Booking of *any* source
     (individual, recurring hire, competition, another game) — reuse the
     same cross-source-overlap style proof Booking's own app layer already
     has.
2. Non-functional requirements:
   - `internal/socialplay/domain` and `internal/socialplay/app` must not
     import `internal/booking/domain` or `internal/booking/app` (verify
     with `go vet`/import check) — the only place `internal/booking/*`
     types are visible is the adapter built in T5.4.
   - If the multi-court-reservation-atomicity question above turns out to
     need more than a single ticket's worth of design (e.g. a real
     saga/compensation pattern), stop and flag it as a scope split rather
     than improvising a distributed-transaction pattern mid-ticket — per
     the loop-mechanics rule on mis-scoped tickets.

**Story points:** 8

**Labels:** `sprint:t5`, `role:principal-engineer`, `type:story`, `points:8`

---

### T5.4 — Wire Social Play to Postgres + proto + gRPC/REST

**Story:** As a client (Vue/Swift/Kotlin, generated from proto), I want
`CreateGame`, `RegisterForGame`, and `CancelRegistration` as real API
endpoints, so that Social Play is actually reachable outside of Go tests.

**Description:** Depends on T5.1–T5.3. The adapter/infra ticket — mirrors
the combined effort of T0's Booking scaffolding + T1–T3's handler wiring,
scoped to Social Play's first three use cases. This is also where the
`port.CourtReservation` adapter (T5.3) gets its real implementation against
Booking's actual `app.Service`.

**Instructions:**
1. Functional requirements:
   - Add `proto/pickleball/socialplay/v1/socialplay.proto` with
     `CreateGame`, `RegisterForGame`, `CancelRegistration` RPCs +
     grpc-gateway REST annotations, mirroring `booking.proto`'s style. Run
     `make generate` — never hand-edit `internal/gen/**` (CLAUDE.md rule 6).
   - Add `games` and `registrations` tables via a new migration under
     `db/migrations`, plus sqlc queries. `registrations` needs a
     capacity-safe read path for the app-layer count in T5.2 (list active
     registrations for a game) — follow the existing `fromFields`-per-query
     pattern documented in CLAUDE.md's Gotchas (sqlc emits a distinct
     `...Row` type per query; don't assume a shared struct).
   - Implement `internal/socialplay/adapter/postgres` (repository for
     Game/Registration) and `internal/socialplay/adapter/booking`
     (implements T5.3's `port.CourtReservation` by calling the real
     `bookingapp.Service.CreateBooking` and translating
     `domain.ErrCourtDoubleBooked` → Social Play's `ErrCourtUnavailable`,
     per Golden Rule 5 — adapters translate infra/other-context errors,
     upper layers never see the foreign error type).
   - Wire `internal/socialplay/adapter/grpcapi` and register it alongside
     Booking's in `cmd/server`.
   - Smoke-test AC (add to README-style curl list or PR description):
     creating a Game reserves its courts (subsequent
     `ListCourtBookings` on that court shows a `game`-source booking);
     registering up to capacity succeeds; the next registration returns
     the `ErrGameFull`-mapped REST status (409, matching how
     `ErrCourtDoubleBooked` maps today); Player A cancelling Player B's
     registration returns a clear rejection (403-shaped, not 500).
2. Non-functional requirements:
   - Postgres 23xxx-class errors (if any new constraint is added — e.g. a
     unique constraint on `(game_id, player_id)` for active registrations
     as the DB-level mirror of `ErrAlreadyRegistered`, per CLAUDE.md rule 4
     "invariants enforced in Postgres AND the domain") must be translated
     to the matching domain error in the adapter, not leaked.
   - `make down && make up` after the schema change (Gotchas: initdb.d only
     applies on a fresh volume).
   - `make test` green, including this context in whatever the top-level
     race/coverage run covers.

**Story points:** 8

**Labels:** `sprint:t5`, `role:principal-engineer`, `type:story`, `points:8`

---

### T5.5 — Object-level authorization regression tests across Social Play's new endpoints

**Story:** As a platform operator, I want every new Social Play endpoint
that acts on a specific Game or Registration to reject a mismatched actor,
so that one player can't cancel or manipulate another player's data before
real auth exists (P1 #6, `docs/requirements/README.md`).

**Description:** Depends on T5.4 (needs real endpoints to test through, not
just domain functions). T5.2 already unit-tests the domain-level ownership
check (`ErrNotRegistrationOwner`); this ticket proves the same guarantee
survives the full stack (gRPC/REST handler → app → domain), the way a BOLA
regression suite actually needs to run, and closes the loop the kickoff
note opened.

**Instructions:**
1. Functional requirements:
   - Add an integration-level test (alongside the existing
     `-tags=integration` pattern from T4, or a lighter handler-level test
     if a full Postgres round trip isn't needed to prove the point) that:
     registers Player A for a Game, then attempts
     `CancelRegistration` as Player B, and asserts the request is rejected
     with the mapped status — not a 500, not a silent success.
   - Since there's no JWT/session layer yet, the "acting player" is
     necessarily just a request field today (e.g. `actor_player_id` in the
     `CancelRegistration` request) rather than derived from a verified
     token — **document this explicitly as a known gap** (a request can
     currently claim to be anyone) in the PR description and as a new line
     in `HANDOFF.md`'s cross-cutting Auth item, so it isn't mistaken for a
     real authorization boundary once JWT auth lands. This ticket proves
     the *object-level* check works given a claimed identity; it does not
     and cannot prove the identity itself, and should not be reported as
     doing so.
   - Extend the same regression pattern to `CreateGame`
     (only the `HostID` who created a Game may `Cancel()` it, per T5.1) if
     time allows within this ticket's points — if it doesn't fit, split it
     into a follow-up rather than silently skip it (loop-mechanics rule).
2. Non-functional requirements:
   - This ticket must not attempt to build real authentication — that's
     explicitly a cross-cutting backlog item (`HANDOFF.md` "Auth (JWT) +
     per-context authorization"), not T5 scope. Scope creep here is exactly
     the kind of thing PM should push back on if a PR starts growing a
     token-verification layer.

**Story points:** 3

**Labels:** `sprint:t5`, `role:qa`, `type:story`, `points:3`

---

## Sprint totals

- **Tickets:** 5 (T5.1–T5.5)
- **Total story points:** 27 (3 + 5 + 8 + 8 + 3)
- **In-scope P0/P1 findings addressed:** P1 #6 (BOLA hygiene — T5.2 + T5.5),
  P0 #3 (waitlist — scoped decision recorded, full build deferred to T6 per
  ADR-0006 update)
- **Explicitly deferred, not forgotten:** full waitlist entity/queue/
  promotion/timeout (→ T6, ADR-0006), P1 #10/#11 rating-margin and
  verified-vs-self-reported weighting (→ matchmaking follow-up per
  `HANDOFF.md`), Game-cancellation cascading to its Bookings/Registrations
  (flagged in T5.1, no ticket yet — raise at T6 refinement if still
  needed).
