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

**T4 — Concurrency/integration test for the invariant. DONE (see docs/reviews/04-t4-concurrency-invariant.md).**
Use testcontainers-go (real Postgres) to fire N simultaneous CreateBooking calls
on the same court/slot; assert exactly one succeeds and the rest get
ErrCourtDoubleBooked. This is the test that actually proves the EXCLUDE guard.
AC: race test passes reliably.

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
