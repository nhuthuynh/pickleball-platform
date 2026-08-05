# Pickleball Platform — Backend

> A Go backend for a pickleball court-management and community platform —
> booking, social games, and payments on one polymorphic invariant that
> guarantees a court is never double-booked, no matter which feature reserved it.

## What this is

Court owners list facilities and courts. Players book a court directly, a
host runs a social game on booked courts, a club hires courts on a recurring
schedule, or an organiser runs a competition — **four different reasons to
reserve a court, one aggregate underneath.** Every reservation, regardless of
source, is the same `Booking`, so the no-double-booking rule only has to be
correct once and it covers all four (see `docs/adr/0001-dual-invariant-enforcement.md`).

Domain logic — booking rules, pricing resolution, matchmaking, the payment
state machine — lives entirely in this Go backend. The web app (Vue), iOS app
(Swift), and Android app (Kotlin) are planned as thin clients generated from
the same `proto/` contract; no business rule is duplicated across them.

**Goal:** a solo-buildable spine (facilities → courts → pricing → booking)
that a community/social layer, clubs, competitions, and payments all sit on
top of, without becoming three different apps that happen to share a
database. See `docs/pickleball-platform-spec.md` for the full product spec
and `docs/technology-options.md` for why this stack.

### How it's built

| | |
|---|---|
| **Domain-Driven Design** | one bounded context per package (`internal/booking`, more to follow), a pure domain layer with zero framework imports, ubiquitous language shared across DB/Go/proto — see `docs/agent-operating-handbook.md` |
| **Test-Driven Development** | every invariant starts as a failing table-driven test; `make test-domain` is the fast, dependency-free gate that must stay green |
| **API contract-first** | one `.proto` per context generates gRPC, a REST mapping (grpc-gateway), and OpenAPI — the single source of truth for the Go server *and* every client |
| **Invariants enforced twice** | a Postgres `EXCLUDE` constraint is authoritative under concurrency; a pure-Go domain check gives the same answer instantly in unit tests |

### Stack

Go (stdlib `net/http`/Chi + pgx + sqlc) · Postgres + PostGIS · gRPC + grpc-gateway + OpenAPI · Vue 3 (planned) · Swift/SwiftUI (planned) · Kotlin (planned) · Docker · Jenkins

### Status

This is an actively developed vertical slice, not a finished product. `CLAUDE.md`
("Current state") and `HANDOFF.md` (task backlog) are the living source of
truth for what's built versus planned — check those rather than this README
for the current phase, since they're updated every increment and this section
isn't.

## Quick start

```bash
# 1. Tools
brew install bufbuild/buf/buf sqlc gotestsum golangci-lint   # or your platform's equivalent

# ...plus Go >= 1.25, Node >= 22.18, and (optionally) Docker.
# Check what's actually present before you start:
make tools-check

# 2. Fast, dependency-free gate — should be green before anything else
make test-domain

# 3. Generate gRPC/REST/OpenAPI code + typed SQL (writes to internal/gen/, gitignored)
make generate
make tidy

# 4. Run the whole stack: Postgres + the API + the Vue web client
cp .env.template .env
make up
```

`make up` serves:

| | |
|---|---|
| Web (Vue) | <http://localhost:5173> |
| REST API | <http://localhost:8080> |
| gRPC API | `localhost:8081` |
| Postgres | `localhost:5432` |

It runs `make generate-client` first, so the generated Go *and* TypeScript
code is up to date before anything is built — the web image refuses to
build without it, exactly as the API image refuses to build without
`internal/gen`.

## Running the CI checks locally

`make ci` runs the same lint / unit-test / codegen / build / security-scan
sequence the Jenkins pipeline runs, in the same order, so "green locally"
and "green in CI" mean the same thing:

```bash
make ci
```

The Docker-dependent half is separate, because a large share of this
project's development environments have no Docker daemon (see CLAUDE.md's
gotchas) and the pipeline skips the same stage on the same condition:

```bash
make ci-integration    # testcontainers concurrency + integration suite
```

Individual pieces, if you want them one at a time: `make lint`,
`make lint-web`, `make test-domain`, `make test-tools`, `make test-web`,
`make build-web`, `make security`.

`make security` runs `govulncheck` (Go) and `npm audit` (web) and gates on
**new** high-severity findings via `cmd/vulngate`; anything pre-existing
must be listed, with a written reason, in `security/vuln-baseline.json`.
On a machine that cannot reach `vuln.go.dev`, `SKIP_GOVULNCHECK=1` skips
the Go half with a loud warning — never set it in CI.

Load tests are opt-in and need a running stack: see `loadtest/README.md`.

## Smoke test

With `make up` running (Postgres seeded with two courts, see
`db/migrations/0002_seed.sql`):

```bash
# Create a booking — expect 200 (grpc-gateway's default status for a
# successful POST; wiring a ForwardResponseOption for a semantic 201 is a
# cross-cutting follow-up, not done yet — see HANDOFF.md).
curl -i -X POST localhost:8080/v1/bookings \
  -H 'Content-Type: application/json' \
  -d '{
    "court_id": "11111111-1111-1111-1111-111111111111",
    "source": "SOURCE_INDIVIDUAL",
    "starts_at": "2026-08-10T09:00:00Z",
    "ends_at": "2026-08-10T10:00:00Z"
  }'

# Create an overlapping booking on the same court — expect 409.
curl -i -X POST localhost:8080/v1/bookings \
  -H 'Content-Type: application/json' \
  -d '{
    "court_id": "11111111-1111-1111-1111-111111111111",
    "source": "SOURCE_GAME",
    "starts_at": "2026-08-10T09:30:00Z",
    "ends_at": "2026-08-10T10:30:00Z",
    "reference_id": "game-1"
  }'
```

If both hold, the vertical slice is live end to end.

```bash
# List court-1's bookings for the day — expect both bookings above.
curl -s "localhost:8080/v1/courts/11111111-1111-1111-1111-111111111111/bookings?from=2026-08-10T00:00:00Z&to=2026-08-11T00:00:00Z"

# Get a quote for the same slot — expect 200 with a priceCents/band body.
curl -i -X POST localhost:8080/v1/quotes \
  -H 'Content-Type: application/json' \
  -d '{
    "court_id": "11111111-1111-1111-1111-111111111111",
    "starts_at": "2026-08-10T09:00:00Z",
    "ends_at": "2026-08-10T10:00:00Z"
  }'

# Cancel the first booking (use its id from the create response above), then
# re-create the same slot — expect success (200), not 409, now that it's freed.
curl -i -X POST localhost:8080/v1/bookings/<booking-id>:cancel
```

All of the above was verified live against a real Postgres 16 instance
during development (not just unit-tested) — see
`docs/reviews/04-t4-concurrency-invariant.md`.

## Repository layout

```
proto/                     API contract (gRPC + REST + OpenAPI source of truth)
db/migrations, db/queries  schema (EXCLUDE constraint) + sqlc queries
internal/<context>/
  domain/  app/  port/  adapter/{postgres,grpcapi}
internal/platform/         shared adapters (db pool, id generation)
cmd/server                 wires gRPC + grpc-gateway REST
cmd/vulngate               CI security gate (CLI over tools/vulngate)
tools/vulngate             scanner-report parsing + baseline logic (tested)
security/                  vuln-baseline.json — accepted findings, with reasons
loadtest/                  k6 scenarios (opt-in, not a per-PR gate)
web/                       Vue client (+ its Dockerfile / nginx config)
docs/                       spec, operating handbook, design review, ADRs, reviews
Jenkinsfile                CI/CD pipeline; `make ci` runs the same checks locally
```

See `CLAUDE.md` "Architecture" and "Locked decisions" before adding a new
bounded context — mirror the `booking` context's shape exactly.
