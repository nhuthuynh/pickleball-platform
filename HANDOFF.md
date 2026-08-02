# HANDOFF — resume notes for Claude Code

Read this once when picking the project up, then follow `CLAUDE.md` for the
durable rules. Companion planning docs are in `docs/` (spec, operating handbook
with the ubiquitous language + role briefs, design review, technology options).

## Current state

**Done and runnable now**
- DDD layering established for the **Booking** context (`internal/booking/{domain,
  app,port,adapter}`).
- Pure domain rules + table-driven tests: time-range validity, overlap logic
  (incl. back-to-back edges), booking-source validation, the no-conflict
  invariant, and pricing resolution across weekday/peak/weekend bands + boundaries.
- Application service with an in-memory test proving cross-source overlaps are
  rejected (game vs competition on the same court/time).
- Schema with the working `EXCLUDE` invariant; sqlc queries; seed data.
- One `booking.proto` → gRPC + grpc-gateway REST + OpenAPI.
- Postgres + gRPC adapters, server wiring, Dockerfile, docker-compose, Makefile,
  Jenkinsfile, and Swift/Kotlin client-generation config.

**Generated on the developer's machine (gitignored)**
- `internal/gen/**` from `make generate` (needs `buf` + `sqlc` installed).

**Not yet built**
- Pricing is modelled + tested but not wired into a use case or the API.
- ListCourtBookings/CancelBooking handlers.
- Social Play, Payments, Facilities, Competitions, Statements contexts.
- Auth, real migration tooling, integration/concurrency tests, observability.

## First actions on resume (T0 — do this before anything else)
1. Install tools: `buf`, `sqlc`, `gotestsum`, `golangci-lint`, Docker — all
   four of the Go-based ones (`buf`, `sqlc`, `gotestsum`, `golangci-lint`)
   install via plain `go install .../cmd/...@latest` even without BSR
   (`buf.build`) access; see the buf.build gotcha below if `make generate`
   fails with "the server hosted at that remote is unavailable."
2. `make test-domain` → must be **green**. This confirms the pure core compiles
   and the TDD baseline holds with zero external deps.
3. `make generate && make tidy` → resolve any dependency-version drift in
   `go.mod` until it builds.
4. `make up` (or run Postgres another way + `go run ./cmd/server` if Docker
   isn't available — both were exercised during T4), then smoke-test with the
   `curl`s in `README.md`: creating a booking returns 200, an overlapping one
   returns 409. If both hold, the slice is live.
5. Only then start the backlog below.

## Task backlog (ordered, TDD-first)

Each task: write the failing test(s) first. "Done" = tests green + `make test`
green + adapters/handlers wired + (if an architectural choice was made) an ADR
added under `docs/adr/`.

**T1 — Wire Pricing into a Quote use case. DONE (see docs/reviews/01-t1-pricing-quote.md).**
Why: prove a second slice of domain logic reaches the API.
Do: add `pricing_rules` table + migration + sqlc queries; a
`port.PricingRuleRepository` + Postgres adapter; an app method that resolves a
price for a (court, slot); expose via a `GetQuote` rpc (new proto method) and/or
attach the price to CreateBooking.
AC: table-driven quote tests pass; REST `GetQuote` returns the correct band price;
no-rule case returns a clear error.
Known gap, deliberately deferred: `pricing_rules` has no DB-level guard against
overlapping rule windows (no EXCLUDE-style constraint), so CLAUDE.md rule 4's
"invariant in Postgres AND the domain" only holds on the domain side today
(`domain.ErrAmbiguousPricingRule`, detected at read time in `ResolvePrice`).
Accepted for T1 because there is no write path yet — `pricing_rules` is
seeded via migration only; Pricing/Facilities CRUD doesn't exist. Add the
write-time guard (an app-level pre-check at minimum, ideally a DB constraint)
when a `CreatePricingRule` use case is built.

**T2 — ListCourtBookings. DONE (see docs/reviews/02-t2-list-court-bookings.md).**
The proto method already exists; `repo.ListActiveForCourt` exists. Implement the
app method + handler + tests. AC: REST GET returns bookings intersecting the range.

**T3 — CancelBooking. DONE (see docs/reviews/03-t3-cancel-booking.md).**
Add status→cancelled transition; cancelled bookings free the slot (the invariant
already ignores them). AC: test that cancelling then re-booking the same slot
succeeds; REST endpoint added.

**T4 — Concurrency/integration test for the invariant. DONE, with a
follow-up fix (see docs/reviews/04-t4-concurrency-invariant.md and its
"Correction" section, and docs/LESSONS.md).**
Use testcontainers-go (real Postgres) to fire N simultaneous CreateBooking calls
on the same court/slot; assert exactly one succeeds and the rest get
ErrCourtDoubleBooked. This is the test that actually proves the EXCLUDE guard.
AC: race test passes reliably — this required a bounded deadlock/serialization
retry in `Repository.Create` (Postgres can raise `40P01`/`40001` under
concurrent EXCLUDE-index contention instead of a clean `23P01`); verified
clean across 7 runs including 2 cold starts after the fix.

**T5 — Social Play context (skeleton first).**
New `internal/socialplay/{domain,app,port,adapter}` + `proto/pickleball/socialplay/v1`.
Start with the Game aggregate (capacity invariant) and Registration. Crucially:
scheduling a game reserves courts by creating **`game`-source Bookings**, so it
inherits the no-overlap invariant — add a test proving a game cannot be scheduled
onto a court already booked. Defer matchmaking to a follow-up task.
AC: capacity invariant tested; game scheduling creates bookings and respects
overlap; registration paid/unpaid status modelled.

**T6 — Payments context.**
`Payment` aggregate with a `PaymentStatus` state machine (unpaid→paid→refunded);
online path behind a Stripe **anti-corruption layer** (interface + stub adapter
first, real Stripe later); offline path where Host/Game Admin records the amount.
AC: state-transition tests (incl. illegal transitions rejected); offline
mark-paid path; one `payments` row per payable action.

## Cross-cutting / later
- `app.Service.NewService`'s constructor has grown to 3 positional args
  (repo, pricingRepo, ids) after T1; Principal Engineer review flagged this
  as fine for now but worth revisiting (options struct or split services)
  if a 4th dependency lands — likely in T5/T6.
- `GetQuote` currently lives on Booking's `app.Service` rather than a
  standalone Pricing bounded context, since Pricing has no aggregate/CRUD of
  its own yet. Reasonable for T1 (trivially extractable — it's a thin
  ListForCourt + domain.ResolvePrice pass-through); revisit if/when Pricing
  grows real CRUD and its own lifecycle.
- Pricing rule weekday encoding uses Go's `time.Weekday` numbering
  (Sunday=0..Saturday=6) directly in the `pricing_rules.weekdays` column —
  fine for a solo Go shop, but leaks a language convention into the schema.
  Consider ISO-8601 numbering (Mon=1..Sun=7) if/when non-Go tooling reads
  this table directly.
- Swap docker initdb.d for **golang-migrate** or **goose** before production.
- Auth (JWT) + per-context authorization; wire into gRPC interceptors.
- T5.2 PR review finding (non-blocking, logged not fixed): `domain.Register`
  never checks `Game.Status`, so nothing currently stops registering into a
  cancelled Game. Not in T5.2's AC; close this when Game-cancellation
  cascading (also flagged in T5.1, see PR #11) is built.
- T5.5's actor-scoped authorization checks (Registration/Game ownership) use
  a request-supplied `actor_player_id` field, not a verified identity — this
  is *not* a real authorization boundary (anyone can claim to be anyone)
  until the JWT/Auth0 item above lands. Don't mistake it for one.
- T6.4 PR review finding (non-blocking, logged not fixed): the claimed
  20-way concurrent-duplicate-recording burst against the `payments`
  `UNIQUE(payable_type, payable_id)` guard (1 success, 19 clean conflicts,
  3 runs incl. a cold start) was only run via an uncommitted throwaway
  program, not a committed test — unlike T4's committed concurrency test,
  nothing guards this invariant against regression. Port it into a
  committed `-tags=integration` test (lower risk than T4's exclusion
  constraint, since a unique index doesn't have the same deadlock-prone
  failure mode, but still worth a permanent regression proof).
- T6.4's `ServiceOptions` deliberately omits `RegistrationUpdater`
  (`socialplayport.RegistrationPaymentUpdater`) because `internal/socialplay`
  isn't merged into the T6 lineage yet — T6.5 is the ticket that first
  merges Social Play (T5) into the same branch as Payments (T6), and needs
  to add `RegistrationUpdater` to `ServiceOptions` at that point, not
  invent a second constructor path.
- T5.5 (see PR stacked on #11-#14, closes issue #10) added a full-stack
  regression test — `internal/socialplay/adapter/grpcapi/authz_regression_test.go`
  — proving `Registration.Cancel`'s object-level ownership check (Player A
  cannot cancel Player B's registration) survives the real path
  (`grpcapi.Handler.CancelRegistration` -> `app.Service.CancelRegistration`
  -> `domain.Registration.Cancel`), not just the domain-level unit test
  T5.2 already had. It is a handler-level test against in-memory
  `port.GameRepository`/`port.RegistrationRepository` fakes rather than a
  `-tags=integration` Postgres round trip: the ownership check has no SQL
  involved, so a real DB adds infrastructure, not proof (ticket text
  explicitly allows this), and this environment had no Docker daemon
  available to actually execute a testcontainers-based version (only the
  `docker` CLI, `docker ps` fails to dial the socket — the same gap
  `internal/socialplay/adapter/postgres/concurrency_integration_test.go`'s
  package comment already documents for this context). Verified as a real
  regression test, not a decorative one, by temporarily commenting out the
  ownership check in `domain.Registration.Cancel` and confirming the new
  test fails, then restoring it and confirming green again (CLAUDE.md rule
  10). **Still reiterating the caveat above**: this proves the object-level
  check given a claimed `actor_player_id`; it does not and cannot prove
  that identity itself without real auth.
  **Split to a follow-up, not silently skipped**: the ticket also asked for
  the equivalent on `CreateGame`/`Game.Cancel()` (only a Game's `HostID`
  may cancel it, T5.1). This did not fit T5.5's scope because it doesn't
  exist yet to test — there is no `CancelGame` RPC in
  `proto/pickleball/socialplay/v1/socialplay.proto`, no
  `app.Service.CancelGame` method, and `domain.Game.Cancel()`
  (`internal/socialplay/domain/game.go`) takes no actor parameter at all
  (unlike `Registration.Cancel`, it isn't even ownership-checked at the
  domain level yet). Building one is proto + app + handler + regression-test
  work, not an extension of an existing pattern — a new ticket (proposed:
  "Add `CancelGame` with HostID-scoped authorization + regression test",
  same shape as T5.1/T5.4/T5.5 combined) should cover it; raise at the next
  backlog refinement.
- Observability: Sentry + slog + uptime.
- Generate the **Vue** typed REST client from the OpenAPI output; generate Swift +
  Kotlin gRPC clients (`buf generate --template buf.gen.mobile.yaml`).

## Definition of Done (per task)
Acceptance criteria met · new/updated tests green · `make test` green · invariants
have explicit tests · domain stayed framework-free · infra errors mapped to domain
errors · ADR written if an architectural decision was made.

## Bootstrap log (T0)
- 2026-07-31: repo bootstrapped from scratch on `claude/go-backend-pickleball-7up34j`.
  Old unrelated TypeScript sample removed. Docs (`CLAUDE.md`, this file,
  `docs/agent-operating-handbook.md`, `docs/pickleball-platform-spec.md`,
  `docs/spec-design-review.md`, `docs/technology-options.md`) written from the
  uploaded planning material. Booking context (domain/app/port/adapter),
  `booking.proto`, DB migrations + sqlc queries, docker-compose, Dockerfile,
  Makefile, Jenkinsfile built to match the state described above.
  `make test-domain` green (see `docs/reviews/00-bootstrap.md`).
