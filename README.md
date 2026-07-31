# Pickleball Platform — Backend

A Go backend for a pickleball court-management + community platform. See
`CLAUDE.md` for the durable engineering rulebook and `HANDOFF.md` for current
state and the task backlog. Planning docs live in `docs/`.

## Quick start

```bash
# 1. Tools
brew install bufbuild/buf/buf sqlc gotestsum golangci-lint   # or your platform's equivalent

# 2. Fast, dependency-free gate — should be green before anything else
make test-domain

# 3. Generate gRPC/REST/OpenAPI code + typed SQL (writes to internal/gen/, gitignored)
make generate
make tidy

# 4. Run Postgres + the API
cp .env.template .env
make up
```

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
docs/                       spec, operating handbook, design review, ADRs, reviews
```

See `CLAUDE.md` "Architecture" and "Locked decisions" before adding a new
bounded context — mirror the `booking` context's shape exactly.
