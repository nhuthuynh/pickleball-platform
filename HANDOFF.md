# HANDOFF — resume notes for Claude Code

Read this once when picking the project up, then follow `CLAUDE.md` for the
durable rules. Companion planning docs are in `docs/` (spec, operating handbook
with the ubiquitous language + role briefs, design review, technology options).

## Docs index

Read the row for whatever phase you're touching *before* starting a task —
this is the map CLAUDE.md's "Docs index & naming convention" section points
to. Don't duplicate a decision, ADR, or finding that already lives in one of
these; add to or supersede it explicitly instead (see `docs/LESSONS.md`'s
own append-only convention). File-naming rules are in CLAUDE.md.

| Phase | Sprint plan | Retro | Reviews | Key ADRs | Design |
|---|---|---|---|---|---|
| T0 | — | — | `docs/reviews/00-bootstrap.md` | — | — |
| T1 | — | — | `docs/reviews/01-t1-pricing-quote.md` | `adr/0002` (pricing ambiguity) | — |
| T2 | — | — | `docs/reviews/02-t2-list-court-bookings.md` | — | — |
| T3 | — | — | `docs/reviews/03-t3-cancel-booking.md` | — | — |
| T4 | — | — | `docs/reviews/04-t4-concurrency-invariant.md` | `adr/0001` (dual invariant), `adr/0003` (local codegen) | — |
| T5 | `docs/process/t5-sprint-plan.md` | `docs/process/t5-retro.md` | PRs #11–#15 (GitHub review comments, not files — see naming convention) | `adr/0006` (waitlist direction), `adr/0007`, `adr/0008` | — |
| T6 | `docs/process/t6-sprint-plan.md` | not yet written | PRs #23, #24, #26, #27, #28 (GitHub review comments) | `adr/0004`, `adr/0005` (currency column, referenced by T6.1) | `docs/design/v1-system-design.md` + `docs/design/v1-review-round-{1..10-final}.md` (10-round Designer+PM+PE+PO review of the requirements list gathered mid-T6; two open items need the user's product/legal sign-off — see the design doc's top blockquote) |

Requirements research (not phase-tied, referenced across T5/T6 planning):
`docs/requirements/README.md` (synthesis) +
`research-{functional,performance-availability,security-compliance,accessibility-i18n}.md`.

Process mechanics (ceremonies, loop caps, this doc-naming convention's origin
incident): `docs/process/sprint-process.md`, `docs/LESSONS.md`.

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

**T5 (Social Play) and T6 (Payments): all tickets implemented and PM+PE-review-approved,
none merged yet** (T5: PRs #11–#15; T6: PRs #23, #24, #26, #27, #28 — #27 and #28 are
sequential merges that fold T6.1–T6.4 and T5.1–T5.5+T6.1–T6.4 together respectively, so
merging #28 first, or #27 then #28, brings in the whole stack). T6.6 (Game waitlist) and
T6.7 (QA auth regression tests) are the two T6 tickets not yet landed as of this entry —
check `docs/process/t6-sprint-plan.md` and GitHub issues #21/#22 for their current state
before assuming they're done. **Merging this backlog is a real, undone task** — see the
`docs/process/t5-retro.md` finding #2 self-approval gap this repeats if left unaddressed
again.

**Not yet built**
- Facilities, Competitions, Statements contexts.
- Game waitlist (T6.6), QA object-level-auth regression tests for Payments (T6.7).
- Auth, real migration tooling, observability.

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

**T5 — Social Play context. All 5 tickets (T5.1–T5.5) implemented and
review-approved (PRs #11–#15); NOT MERGED — see the note above.** Full
ticket breakdown, kickoff note, and PM/PE disagreements:
`docs/process/t5-sprint-plan.md`. Retro: `docs/process/t5-retro.md`.
Original scope (for reference): `internal/socialplay/{domain,app,port,adapter}` +
`proto/pickleball/socialplay/v1`, Game aggregate (capacity invariant) +
Registration, game scheduling reserves courts as `game`-source Bookings
(inherits the no-overlap invariant). Matchmaking deferred past T5.

**T6 — Payments context. T6.1–T6.5 implemented and review-approved
(PRs #23, #24, #26, #27, #28); T6.6 (Game waitlist) and T6.7 (QA auth
tests) not yet landed — check issues #21/#22. NOT MERGED — see the note
above.** Full ticket breakdown, kickoff note, and PM/PE disagreements:
`docs/process/t6-sprint-plan.md`. Original scope (for reference):
`Payment` aggregate with a `PaymentStatus` state machine
(unpaid→paid→refunded); online path behind a Stripe **anti-corruption
layer** (interface + stub adapter first, real Stripe later); offline
path where Host/Game Admin records the amount. See "Cross-cutting /
later" below for follow-ups T6's own reviews surfaced (uncommitted
concurrency proof, missing `RefundPayment` wiring, migration-number
collision).

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
- T6.5 found `payments/app.Service` has no `RefundPayment` method at all —
  `domain.Payment.Refund()` and the `PaymentProcessor.RefundPayment` port
  method both exist, but nothing calls them, so there's no live path that
  would ever push Social Play's new `refunded` `PaymentStatus` (T6.5) end
  to end today. Needs its own ticket: wire `RefundPayment` on
  `payments/app.Service`, call `RegistrationUpdater.UpdatePaymentStatus`
  with `refunded` from it, and prove it with a test — don't assume T6.5's
  plumbing makes this automatic.
- Both the T6.4 and T5 migration lineages independently used migration
  number `0005` for different files (payments table vs. an earlier T5
  migration) — harmless today under the initdb.d alphabetical-apply
  approach once merged, but a landmine for the eventual golang-migrate/
  goose swap (Gotchas already flags initdb.d as prototype-only). Renumber
  when that swap happens, not before.
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
