# CLAUDE.md — Pickleball Platform (backend)

Project memory for Claude Code. Read `HANDOFF.md` once at the start of a resume
session for current state + the task backlog. This file is the durable rulebook.

## What this is
A Go backend for a pickleball court-management + community platform. Currently a
runnable vertical slice through the **Booking** bounded context; other contexts
follow the same pattern. Web client = Vue, mobile = Swift (iOS) + Kotlin
(Android), all generated from `proto/`. Domain logic lives only in this backend.

## Golden rules (do not violate)
1. **TDD.** Write a failing table-driven test first, then the minimum code to
   pass, then refactor. No production code without a test that demanded it.
2. **Keep the domain pure.** `internal/<context>/domain` imports nothing outside
   the standard library — no pgx, grpc, or framework imports. Business rules live
   here as pure functions/types.
3. **Dependency rule points inward:** `adapter → app → domain`. Never import an
   adapter from the domain or app.
4. **Invariants are enforced in Postgres AND expressed in the domain.** The
   no-double-booking rule is an `EXCLUDE` constraint (authoritative) and
   `domain.EnsureNoConflict` (for unit tests / pre-checks). Keep both in sync.
5. **Adapters translate infra errors into domain errors** (e.g. Postgres `23P01`
   → `domain.ErrCourtDoubleBooked`). Upper layers only ever see domain errors.
6. **Never hand-edit generated code.** `internal/gen/**` comes from `make
   generate` (buf + sqlc). Change the `.proto` / `.sql`, then regenerate.
7. **One ubiquitous language** across DB, Go, proto, and clients. A `Booking` is
   a `Booking` everywhere. See glossary in `docs/agent-operating-handbook.md`.
8. **Run `make test` green before calling any task done.** Add/adjust tests for
   every change; turn every bug into a regression test.
9. **No direct commits/pushes to the shared branch — PR only.** Every change,
   including subagent/review work, lands via a branch + PR that is reviewed,
   tested, and explicitly approved before merge. A reviewer/QA/PE agent's job
   is to *report* findings, never to commit or push itself. (Added after an
   incident where a review-only subagent pushed unreviewed work directly —
   see `docs/LESSONS.md`.)
10. **A single successful run is not proof of reliability**, especially for
    concurrency claims. Re-run non-deterministic tests (cold start + several
    repeats) before writing "proven" or "reliable" anywhere.

## Commands
- `make test-domain` — run the dependency-free domain + app tests (no DB/codegen).
- `make generate` — buf + sqlc → `internal/gen` and `openapi/`.
- `make tidy` — `go mod tidy` (run after first generate).
- `make test` — full suite: race + JUnit + coverage.
- `make up` / `make down` — run / tear down via docker compose.
- `make lint` — golangci-lint.

## Architecture
```
proto/                     API contract (gRPC + REST + OpenAPI source of truth)
db/migrations, db/queries  schema (EXCLUDE constraint) + sqlc queries
internal/<context>/
  domain/  app/  port/  adapter/{postgres,grpcapi}
internal/platform/pg       db pool
cmd/server                 wires gRPC + grpc-gateway REST
```
Add a new bounded context as `internal/<context>/{domain,app,port,adapter}` with
its own `proto/pickleball/<context>/v1` — mirror the `booking` context exactly.

## Locked decisions — do NOT reopen
- Stack: **Go** backend, **Vue** web, **Swift + Kotlin** native, **gRPC +
  OpenAPI** from one proto, **Docker**, **Jenkins**. See `docs/technology-options.md`.
- Full scope in v1 (court mgmt + hosting + joining); build order is spine-first.
- **Polymorphic Booking:** recurring-hire, individual, game, competition are ONE
  Booking aggregate so the invariant covers all four.
- Payments: Stripe (online) **and** offline amount entry; paid/unpaid tracking is
  one source of truth. Per-game **Game Admins** can record offline payments.
- Matchmaking: automated from history, always manually overridable; new players
  seeded by a self-reported starting level.

## Gotchas
- Nothing was compiled in the authoring environment — run `make tidy` first;
  `go.mod` versions are indicative and may need a nudge.
- `internal/gen/**` is gitignored; the postgres/grpc adapters + `cmd/server` only
  compile after `make generate`.
- `bookings` uses separate `starts_at`/`ends_at` columns plus a generated
  `during tstzrange` so sqlc sees plain timestamptz. Don't change queries to
  select the range directly (sqlc will type it as `interface{}`).
- docker compose applies `db/migrations/*.sql` via initdb.d **only on a fresh
  volume**. After a schema change run `make down` (drops the volume) then `make up`.
  Prototype-only; adopt golang-migrate/goose for production migrations.
- `buf generate` uses **local plugins**, not `buf.build/...` remotes — the BSR
  isn't reachable from every environment this repo is developed in. `go
  install` these four once (`protoc-gen-go`, `protoc-gen-go-grpc` from
  `google.golang.org/grpc/cmd/...`, `protoc-gen-grpc-gateway` and
  `protoc-gen-openapiv2` from `github.com/grpc-ecosystem/grpc-gateway/v2/...`)
  and make sure `$(go env GOPATH)/bin` is on `PATH`. `google/api/*.proto` are
  vendored under `proto/google/api/` for the same reason — don't replace them
  with a `buf.build/googleapis/googleapis` dependency without confirming BSR
  access first.
- sqlc generates a **distinct `...Row` struct per query**, not a shared table
  model, whenever a query's column list doesn't exactly match `SELECT *` on
  that table (e.g. the booking queries omit the generated `during` column).
  Don't write adapter code assuming one row type across queries — see
  `fromFields` in `internal/booking/adapter/postgres/repository.go` for the
  pattern (convert from the shared columns, not a shared struct).
- The `internal/booking/adapter/postgres` concurrency test
  (`concurrency_integration_test.go`) is gated behind `//go:build integration`
  and needs Docker (testcontainers-go). Run it with `go test
  -tags=integration ./...` or `make test`; it's intentionally excluded from
  `make test-domain` and plain `go test ./...`.

## Current state (updated by each phase, see HANDOFF.md for detail)
- T0 bootstrap complete: Booking domain + app + Postgres/gRPC adapters +
  proto + docker/Jenkins scaffolding in place, `make test-domain` green.
- T1 complete: `Service.GetQuote` wired to a `pricing_rules` table, sqlc
  query, Postgres adapter, and the `GetQuote` gRPC/REST endpoint. Reviewed
  by adversarial QA + Principal Engineer passes (see
  `docs/reviews/01-t1-pricing-quote.md`); fixed a real cross-midnight
  pricing bug the QA pass found. `make test-domain` green.
- T2 complete: `ListCourtBookings` app method + gRPC/REST handler wired, with
  tests proving court scoping, range intersection, and cancelled-booking
  exclusion. See `docs/reviews/02-t2-list-court-bookings.md`. `make
  test-domain` green.
- T3 complete: `CancelBooking` app method + gRPC/REST handler wired, with a
  test proving cancelling actually frees the slot for re-booking (not just
  that the status field flips). See `docs/reviews/03-t3-cancel-booking.md`.
  `make test-domain` green.
- T4 complete: unblocked the full toolchain (buf/sqlc installed, local
  codegen plugins since the BSR isn't reachable here, vendored
  `google/api/*.proto`), fixed the Postgres adapter's row-type mismatch
  against real generated code, and proved the no-double-booking invariant
  under real concurrency (20 simultaneous `CreateBooking` calls, exactly 1
  success) both manually against a local Postgres and via a committed
  `testcontainers-go` test gated behind `-tags=integration`. `go build ./...`
  and `go vet ./...` now succeed for the **entire** repository. **Follow-up
  correction (same day):** the initial single-run verification missed an
  intermittent Postgres deadlock (`40P01`) on cold-start concurrent bursts —
  reproduced independently, then fixed with bounded retry in
  `Repository.Create`, and re-verified clean across 7 runs incl. 2 cold
  starts. See `docs/reviews/04-t4-concurrency-invariant.md` (with its
  correction section) and `docs/LESSONS.md`.
- Process change (same day): background/reviewer subagents are report-only
  — they must never commit or push. All changes land via a PR reviewed and
  approved before merging into this branch. See `docs/LESSONS.md`.
- Next phase: see `HANDOFF.md` task backlog (T5 onward).
