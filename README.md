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

You need **Go >= 1.26.9**, **Node >= 22.18**, and **Docker** (for `make up`).
Everything else one command installs.

```bash
# 1. Tools — buf, sqlc, gotestsum AND the four protoc plugins buf shells out to
make tools-install
export PATH="$PATH:$(go env GOPATH)/bin"     # add this to your shell profile

make tools-check                             # every line should read "ok"

# 2. Fast, dependency-free gate — should be green before anything else
make test-domain

# 3. Generate gRPC/REST/OpenAPI code + typed SQL (writes to internal/gen/, gitignored)
make generate
make tidy

# 4. Run the whole stack: Postgres + the API + the Vue web client
cp .env.template .env
make up
```

> **Why `make tools-install` and not just `brew install buf sqlc …`** — which
> is what this section used to say. `buf generate` uses **local** protoc
> plugins rather than the Buf Schema Registry's remote ones, because the BSR
> is not reachable from every environment this repo is developed in. A
> package manager installs `buf` but not `protoc-gen-go`,
> `protoc-gen-go-grpc`, `protoc-gen-grpc-gateway` or `protoc-gen-openapiv2`,
> so step 3 failed on a fresh clone with a plugin-not-found error that did
> not say what to install. `make tools-check` now names all four.

`make up` serves:

| | |
|---|---|
| Web (Vue) | <http://localhost:5173> |
| REST API | <http://localhost:8080> |
| gRPC API | `localhost:8081` |
| Postgres | `localhost:5432` |

### Your first run, in the browser

Open <http://localhost:5173>. The API is reached at a **relative** `/v1`,
proxied by the same nginx that serves the page, so there is nothing to
configure and no cross-origin call to allow.

Everything that creates or joins something needs a signed-in player with a
profile, and local development has a committed key fixture for exactly that:

```bash
TOKEN=$(make -s dev-token)   # dev-only; the keypair is public, see dev/auth/README.md
echo "$TOKEN"
```

1. Go to **/sign-in**, paste the token, **Sign in**.
2. Go to **/welcome** — pick a display name and your starting level. That
   creates your player profile. (There is no redirect; the first API call
   that needs a profile puts a banner at the top of every page linking
   here, so you cannot get far without noticing.)
3. Now the whole loop works, all of it from the nav: **Discover → Add a
   facility** (name, address, then a court), **Games → Create a game**, then
   **Games**, open the card, and **Join**.

`node web/e2e/journey.mjs` drives exactly that sequence in a real browser if
you would rather watch it than click it; `make e2e` wraps it and names the
prerequisites.

It runs `make generate-client` first, so the generated Go *and* TypeScript
code is up to date before anything is built — the web image refuses to
build without it, exactly as the API image refuses to build without
`internal/gen`.

## CI/CD

[![CI](https://github.com/nhuthuynh/pickleball-platform/actions/workflows/ci.yml/badge.svg?branch=claude/go-backend-pickleball-7up34j)](https://github.com/nhuthuynh/pickleball-platform/actions/workflows/ci.yml)


The pipeline is **GitHub Actions** — `.github/workflows/ci.yml` — and it runs
on every pull request and every push to the shared branch. Four jobs:

| Job | What it runs | Why it exists |
|---|---|---|
| Gate | `make ci-checks` | codegen, tidy, format, lint, the five Go test targets, `vet-integration`, the meta-gates, the web client codegen, lint-web, test-web, build-web, `go build ./...` |
| Integration | `make ci-integration` | the testcontainers suite against a real Postgres — **no skip path**, since a runner has a Docker daemon |
| Security | `make security` | `govulncheck` + `npm audit`, gated by `cmd/vulngate` against the baseline |
| Images | `make build-images` | builds the API and web images on every PR, and publishes them to GHCR on a push to the shared branch |

Every job calls this repo's own `make` targets rather than re-listing their
steps, so a check added to `make ci-checks` is picked up with no change to
the workflow. That is the direct lesson of issue #129: the Jenkinsfile hand-
listed `make ci`'s steps and had drifted by a whole check before anyone
looked.

`Jenkinsfile` is still here, and `docs/technology-options.md` still names
Jenkins for the eventual deployment pipeline — but it needs a controller,
plugins, a webhook and a credential that do not exist, and it has never run.
Its header says so, and points here.

### Published images

A push to the shared branch publishes two images, tagged with the commit sha
and with `edge`:

```
ghcr.io/nhuthuynh/pickleball-platform/api:<sha>
ghcr.io/nhuthuynh/pickleball-platform/web:<sha>
```

No secret is involved — the workflow's built-in `GITHUB_TOKEN` can push to
this repository's own package namespace. **Deploying** those images to a
server is not wired: that needs a target and a credential this project does
not have yet, and a stage that cannot run is worth less than a documented
gap.

### Running the prototype without building anything

The published images are what `make prototype` runs, so a first look needs
no toolchain, no codegen and no Go or Node — only Docker and this checkout:

```bash
make prototype                  # pull + start
# -> http://localhost:5173
make prototype-down             # stop, and drop the database volume
```

`make up` remains the development path: it builds both images from your
working tree, which needs the full toolchain, because both Dockerfiles
refuse to build without generated code. That requirement is right for
development and wrong for "let me see the thing run", which is why these are
two targets.

`PICKLEBALL_TAG` selects the image tag and defaults to `edge`, the shared
branch's tip. Pin the commit sha for anything you intend to keep or compare
against — `edge` moves:

```bash
PICKLEBALL_TAG=<sha> make prototype
```

**No `docker login` is needed.** Measured against the first publish
(2026-10-28), not assumed: an anonymous registry token reads the manifest.

```
$ T=$(curl -s "https://ghcr.io/token?scope=repository:nhuthuynh/pickleball-platform/api:pull&service=ghcr.io" \
      | python3 -c "import json,sys;print(json.load(sys.stdin)['token'])")
$ curl -s -o /dev/null -w '%{http_code}\n' -H "Authorization: Bearer $T" \
      https://ghcr.io/v2/nhuthuynh/pickleball-platform/api/manifests/edge
200
```

An earlier draft of this section said the packages would probably be private
and would need their visibility changed, because a package published with
the workflow's `GITHUB_TOKEN` does not necessarily inherit a repository's
visibility. That was a guess, and it was wrong for this account; the command
above is what settles it, and it is here so the next reader can re-settle it
rather than trust the paragraph.

**It does not work inside this project's own cloud development containers.**
Their egress policy allows `ghcr.io` — the manifest fetch above succeeds —
but not `pkg-containers.githubusercontent.com`, which is where GHCR serves
image *blobs*, so the pull dies partway with `Forbidden` on the layers:

```
failed to copy: httpReadSeeker: failed open: failed to do request:
Get "https://pkg-containers.githubusercontent.com/ghcrblobs06/blobs/sha256:…": Forbidden
```

That is a property of those containers, not of the images or this file: on
an ordinary machine the pull completes. Allowing that one host in the
environment's network settings is what makes `make prototype` work there
too.

`compose.prototype.yml` is an **override** of `docker-compose.yml` that adds
two `image:` lines and nothing else, so every environment variable, mount,
port and healthcheck stays defined in one place. `make prototype` passes
`--no-build` deliberately: the base file still defines `build:`, and without
it compose would quietly build from your checkout and the published images
would never be exercised.

### Running the same checks locally

`make ci` runs the same sequence, in the same order, so "green locally" and
"green in CI" mean the same thing:

```bash
make ci
```

Build the images the way CI does (both Dockerfiles refuse to build without
generated code, so this regenerates first):

```bash
make build-images
```

The Docker-dependent half is a separate target because a CI agent may not
have a Docker daemon and the Jenkins pipeline skips that stage on the same
condition. The Actions pipeline does not: it fails instead.
**It is not separate because it cannot run here** — that claim was this
project's standing belief until T61 tested it, and the daemon starts in about
four seconds (see CLAUDE.md's "Docker works here"). Run it:

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

### Authenticated endpoints

Every `curl` above hits a **public** RPC, which is why none of them carries a
token. 24 RPCs across the six contexts are not public: they require a verified
bearer token, and since T13.5 the server refuses to start at all unless it has
been given something to verify tokens with (`AUTH_ISSUER`, `AUTH_AUDIENCE`, and
exactly one of `AUTH_JWKS_FILE` or `AUTH_JWKS_URL`).

`make up` supplies all three from a **development key fixture committed to
this repository** under `dev/auth/` — public key material, therefore worthless,
and deliberately so. Mint a token against it and call an authenticated
endpoint:

```bash
TOKEN=$(make -s dev-token)

# 1. Register the token's own subject as a User. CreateUser is itself an
#    authenticated RPC and takes the owner from the verified token, never
#    from the body (T12.9) — so this is the first call any new caller makes.
curl -s -X POST localhost:8080/v1/users \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"display_name":"Local Dev","roles":["ROLE_PLAYER"],"self_reported_starting_level":3}'
# → 200, {"user":{"id":"<server-minted uuid>", ...}}
#   Run it twice and the second returns "identity: user already exists" —
#   the row is keyed on the token's `sub`, which is the point.

# 2. Read an actor-scoped endpoint. The token's `sub` IS the scope here, so
#    this is where authentication is visibly doing real work.
curl -s -H "Authorization: Bearer $TOKEN" \
  localhost:8080/v1/recurring-hire-templates
# → 200, {"templates":[]}

# 3. The same call without the header.
curl -i localhost:8080/v1/recurring-hire-templates
# → HTTP/1.1 401 Unauthorized
#   {"code":16,"message":"this endpoint requires an authenticated caller"}
```

An expired token (`make dev-token` takes `-ttl`) and one minted for another
audience (`-audience`) are both 401 too, which is the fixture proving it is a
real credential rather than a doorstop.

That 200/401 contrast is the whole point of the fixture: enforcement is real,
runs the same verification code a deployment runs, and is now observable
locally. `make dev-token` takes flags for the subject, scope and lifetime —
`go run ./cmd/devtoken -h`.

**Never point a deployment at `dev/auth/`.** A real environment sets those
three variables at a real identity provider; the Docker image contains no copy
of the fixture (compose bind-mounts it at run time), and a server started
without them still fails closed exactly as it did before. See
`dev/auth/README.md` and issue #160.

### Pointing at a real identity provider

`AUTH_JWKS_URL` is the deployment counterpart of `AUTH_JWKS_FILE`: instead of
reading a key set once from disk, the server fetches the provider's published
JWKS over HTTPS, caches it, and refreshes it — so a key rotation is picked up
without a redeploy (T15.7, issue #137).

```bash
AUTH_ISSUER=https://your-tenant.auth0.com/ \
AUTH_AUDIENCE=https://api.your-domain.example/ \
AUTH_JWKS_URL=https://your-tenant.auth0.com/.well-known/jwks.json \
go run ./cmd/server
```

Four properties worth knowing before you deploy it:

- **The two sources are mutually exclusive.** Setting both `AUTH_JWKS_FILE` and
  `AUTH_JWKS_URL` is a startup error, not a precedence rule — a deployment that
  believes it follows its live provider while actually trusting a stale file in
  the image is exactly the failure this refuses to make quietly.
- **The URL must be `https`.** A JWKS fetched over plaintext can be swapped in
  flight, which would let an attacker choose the keys this process trusts.
- **Startup fetches once and fails if it cannot.** A wrong URL stops the
  server rather than producing one that boots, looks healthy, and rejects every
  authenticated request with something clients cannot distinguish from an
  expired credential.
- **It fails closed at run time too.** If the provider becomes unreachable,
  cached keys keep serving for a bounded window (15 min TTL + 1 h grace by
  default) and then every token is denied. Tokens naming a `kid` that no
  fetched key set contained are denied throughout, and an unknown `kid`
  triggers at most one refresh per minute so an anonymous caller cannot use
  this server to hammer your provider.

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
.github/workflows/ci.yml   the CI/CD pipeline that runs (Actions)
compose.prototype.yml      run the published images: `make prototype`
.github/actions/toolchain  Go + Node + pinned build tools, shared by its jobs
tools/versions.env         pinned versions of the go-installed build tools
Jenkinsfile                the Jenkins port; has never run — see its header
```

See `CLAUDE.md` "Architecture" and "Locked decisions" before adding a new
bounded context — mirror the `booking` context's shape exactly.
