# Load tests

`booking-quote.js` exercises the two endpoints that carry the platform's
core value — `CreateBooking` (the write path guarded by the
no-double-booking invariant) and `GetQuote` (the pricing read path) —
plus `ListCourtBookings` for read coverage.

## Verification status (read this first)

**This script has never been executed against a running stack.** It was
written against the real REST contract (`proto/`, the README smoke-test
`curl`s, and the seed data in `db/migrations/0002_seed.sql` +
`0004_seed_pricing.sql`) and syntax-checked as an ES module, but the
environment it was authored in has neither a Docker daemon nor a k6
binary, so no run has produced real numbers.

Concretely, per CLAUDE.md rule 10:

| Claim | Status |
|---|---|
| Script parses as valid ESM | **Verified** (`node --check`) |
| Endpoint paths/payloads match the API | **Verified by reading** `proto/` + README, not by a request |
| Slot-allocation scheme avoids 409 collisions | **Reasoned, not observed** — a `booking_conflicts` counter exists precisely to catch it being wrong |
| Thresholds (p95 < 500ms, <1% failures) are correct | **Unvalidated guesses.** Tune from the first real runs |

Treat the first real run as calibration, not as a gate.

## Why k6

| Option | Why not |
|---|---|
| **JMeter** | XML test plans, JVM runtime, awkward to review in a PR diff |
| **Gatling** | Scala DSL — a third language in a Go + TypeScript repo |
| **Locust** | Python — same objection, plus a runtime to install |
| **Vegeta** | Go and lovely for flat request floods, but no real scenario scripting (multi-step flows, per-VU state) |

**k6** wins on three project-specific counts: scenarios are plain
JavaScript, which is a language this repo already has (the Vue app) rather
than a new one; it ships as a single static binary with no runtime to
provision on a Jenkins agent; and `thresholds` make a run objectively
pass/fail instead of producing a wall of numbers a human has to judge.

Requires **k6 >= 0.44** (the script uses `??` and optional catch binding).

## Running it locally

```bash
make up                        # Postgres + API + web
k6 run loadtest/booking-quote.js
```

Tunables (all optional):

```bash
BASE_URL=http://localhost:8080 \
VUS=20 DURATION=30s \
COURT_ID=11111111-1111-1111-1111-111111111111 \
k6 run loadtest/booking-quote.js
```

Or via the Makefile, which fails with a clear message if k6 is missing:

```bash
make loadtest
```

## It writes real data

`CreateBooking` is a write. Every iteration inserts a booking, so a 30s /
20 VU run leaves a few thousand rows behind. Two consequences:

1. **Run it against a local stack only**, never a shared environment.
2. Reset between runs with `make down && make up` (which drops the volume —
   see CLAUDE.md's initdb gotcha) if you want comparable numbers.

Bookings are placed from 2027-01-01 onward, one hour per (VU, iteration),
so they never collide with each other or with the seeded smoke-test data.

## Why it is not a per-PR gate

The Jenkins `Load test` stage is **opt-in**, and deliberately so:

- it needs a fully running stack (Postgres + API), so it is the slowest and
  most fragile stage in the pipeline;
- load results are only meaningful compared against a *stable baseline*, and
  a per-PR number measured on a noisy shared Jenkins agent is not that;
- gating every PR on a timing threshold is how teams learn to click "retry"
  until it passes, which trains people to ignore the one signal that is
  supposed to mean something.

So it runs on an explicit trigger or a schedule instead — see the
Jenkinsfile's `RUN_LOAD_TEST` parameter and the `Load test` stage.
