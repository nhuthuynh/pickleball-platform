# Research: Performance, Scalability & Availability NFRs

**Scope:** sourced research to calibrate the spec's §4 non-functional
requirements ("Correctness under load," "Availability & observability,"
"Performance") to a realistic **solo-dev, pre-launch** stage — not enterprise
numbers. Grounded in this project's own data point: the T4 concurrency work
(`docs/reviews/04-t4-concurrency-invariant.md`, `docs/LESSONS.md`) found a
real intermittent Postgres deadlock (`40P01`) on cold-start concurrent
booking bursts that a single successful test run missed. That is direct
evidence that "prove it once" is not a real NFR strategy for this system —
the research below is chosen to make that repeatable instead of anecdotal.

---

## 1. SLO/error-budget framing (Google SRE book)

**Findings**

- The SRE book's core advice for choosing a target: **"don't pick a target
  based on current performance"** and instead **"start with a loose target
  that you can tighten"** over time. Locking in whatever the system happens
  to do today either overcommits you to fragile heroics (if current
  performance is actually bad) or wastes engineering effort chasing headroom
  nobody asked for (if it's actually very good). Current performance is only
  an acceptable *starting point* "if you don't have any other information,
  and if you have a good process for iterating in place." (sre.google,
  *Service Level Objectives*, ch. 4.)
- The book explicitly frames over-delivering reliability as a cost, not a
  virtue: a 100%-reliable target "can reduce the rate of innovation and
  deployment," requires "expensive" and "overly conservative" engineering,
  and can "spoil users" into expecting it permanently. The right question
  when reliability is already "good enough" is whether that engineering time
  is better spent on features or technical debt instead.
- The error-budget mechanic: SLO defines an acceptable failure rate (e.g.
  99.9% = ~43 minutes of allowed badness per month); the *remaining* budget
  after real incidents is a shared, spendable resource for both reliability
  work and release velocity, not a scorecard to feel bad about.
- The companion Workbook (*Implementing SLOs*) reinforces the same iterate-
  don't-perfect posture: the first SLI/SLO attempt "doesn't have to be
  correct" — the goal is to get *something* measured and a feedback loop in
  place, then revise from real data.

**Recommendation for this project's actual stage (pre-launch, zero real
users)**

Do not write 99.9%/99.99% "enterprise" numbers into the spec now — there is
no traffic to calibrate them against and no on-call organization to hold
them. Instead:

- Adopt **two SLIs** that map directly to spec §4's own language:
  1. **Booking-write correctness** — fraction of `CreateBooking` calls that
     resolve to either a clean success or a clean, correctly-translated
     `domain.ErrCourtDoubleBooked`/conflict response (i.e., *not* a raw,
     untranslated DB error reaching the caller). Target while pre-launch:
     **no numeric SLO yet — track it, alert on any non-zero rate of
     untranslated errors**, since T4 already proved that failure mode is
     real. This is "start with a loose target you tighten" applied
     literally: the target today is "visibility," not a percentage.
  2. **API availability** (uptime of `cmd/server` responding to health
     checks) — a deliberately loose starting SLO, e.g. **99% monthly**
     (~7.3 hours of allowed downtime), reviewed and tightened only after a
     few months of real traffic data, per the SRE book's explicit guidance
     against picking a target you have no data to justify.
- Treat any SLO written into the spec today as provisional and dated —
  revisit once there's a first cohort of real bookings, not before.

**Sources**
- Google SRE Book, ch. 4, *Service Level Objectives* — https://sre.google/sre-book/service-level-objectives/
- Google SRE Workbook, *Implementing SLOs* — https://sre.google/workbook/implementing-slos/
- Google SRE Workbook, *Error Budget Policy* — https://sre.google/workbook/error-budget-policy/
- Outline of SRE book ch. 4 (secondary, corroborating) — https://gist.github.com/jonbrouse/921658823a094eb7888d45b676bfb173

---

## 2. Postgres `EXCLUDE` + GiST under write concurrency

**Findings**

- Mechanically, a GiST-backed `EXCLUDE` constraint is the right tool here:
  it uses the index to find only *potentially* conflicting rows instead of
  scanning the table, and non-overlapping inserts proceed concurrently
  without blocking each other — this is why the spec (§6) and
  `technology-options.md` prefer it over `SERIALIZABLE` isolation, which
  would take much coarser predicate locks (one credible write-up notes
  serializable's predicate-lock granularity can effectively lock an entire
  "showing"/court-day, not just the contended slot).
- **This project's own T4 finding is itself the most important documented
  data point**: under concurrent `CreateBooking` bursts against a cold
  connection pool, Postgres can abort a *losing* transaction with
  `deadlock_detected` (40P01) or `serialization_failure` (40001) instead of
  the clean `23P01` exclusion-violation the code originally only handled.
  This is a known, general characteristic of exclusion constraints, not a
  bug specific to this schema: exclusion-constraint checking does its
  conflict detection in a way that can make two concurrent inserters wait on
  each other rather than one cleanly losing immediately, which is exactly
  the shape of a deadlock under contention. A public PostgreSQL bug report
  ("BUG #15026: Deadlock using GIST index," postgrespro.com mailing list
  archive) documents the same class of issue independently, corroborating
  that this is a real, recurring pattern for GiST exclusion constraints
  under concurrent writes — not something unique to this codebase.
- Separately, there are documented reports of GiST range/exclusion queries
  becoming *slow* (not just deadlock-prone) at larger data volumes — one
  PostgreSQL mailing-list thread ("Extremely slow when query uses GIST
  exclusion index") describes pathological slowness on timestamp-range GiST
  queries that improved once the query was restructured to avoid forcing a
  GiST scan. `btree_gist`/GiST index build and lookup performance has also
  materially improved in recent Postgres releases (PostgreSQL 18 cites
  faster `CREATE INDEX` timings on multi-million-row `btree_gist` indexes
  versus PostgreSQL 17), which matters for this project mainly as a "don't
  worry about it prematurely" signal, not an action item yet.
- No credible source gives a single hard "row count where this breaks" —
  the deadlock failure mode is about **concurrent write contention on the
  same court/slot**, not table size. A booking table can hold millions of
  rows fine; the risk is dozens of simultaneous writers targeting the
  *same* narrow (court_id, time-range) key, which for this product means
  popular-slot booking bursts (peak evening/weekend court release), not
  raw traffic volume.

**Recommendation**

- Current mitigation (bounded retry on 40P01/40001 in `Repository.Create`,
  already shipped per T4) is the *correct* first-line fix per the pattern
  documented in the public bug thread, and is consistent with general
  Postgres deadlock-handling advice (retry the loser transaction) — no
  architecture change needed at pre-launch traffic.
- Set an explicit, honest revisit trigger instead of a vague "if it gets
  slow" note: **re-evaluate this pattern (e.g., a Redis/DB-row advisory
  lock per court+slot ahead of the INSERT, or sharding hot slots) once
  sustained concurrent write attempts on a *single* court/time-slot
  regularly exceed ~20–30 simultaneous transactions** — i.e., the same
  order of magnitude the T4 test already exercises, since that's the
  regime where the deadlock rate was empirically non-trivial (17/20 on a
  cold pool). Below that, connection-pool warm-up + bounded retry is
  sufficient and cheaper than adding a locking layer prematurely.
- Keep exercising the **cold-start** case specifically in any future load
  test — T4's own finding was that cold pools are where the deadlock rate
  spikes, so "warm" load tests alone would hide the exact bug this project
  already found once.

**Sources**
- PostgreSQL exclusion-constraint concurrency behavior (dev.to, Franck
  Pachot) — https://dev.to/franckpachot/postgresql-exclude-constraints-for-better-concurrency-than-serializable-pob
- "BUG #15026: Deadlock using GIST index" — https://postgrespro.com/list/thread-id/2368259
- "PostgreSQL: Extremely slow when query uses GIST exclusion index" (pgsql mailing list) — https://www.postgresql.org/message-id/CA+NEr9zpge7KTH1dfOa90YyNUjyQoz8PRSfqB6NpS2HuOrg0Wg@mail.gmail.com
- CYBERTEC, *btree_gist improvements in PostgreSQL 18* — https://www.cybertec-postgresql.com/en/btree_gist-improvements-in-postgresql-18/
- This project's own primary evidence: `docs/reviews/04-t4-concurrency-invariant.md` (Correction section) and `docs/LESSONS.md` ("T4 (follow-up)")

---

## 3. k6 load-testing methodology for a booking API

**Findings**

- k6's own documentation (Grafana k6 docs, *API load testing*) and the
  companion k6-learn course distinguish several test *shapes*, each suited
  to a different question — not one generic "load test":
  - **Smoke/shakeout** — minimal load, catches script and gross
    performance bugs before anything bigger runs.
  - **Average-load test** — gradual ramp-up → steady state → ramp-down,
    modeling ordinary usage.
  - **Stress test** — pushes toward peak expected traffic ("rush hours or
    sale periods").
  - **Spike test** — a **sudden, short, massive** jump from a low baseline,
    explicitly the pattern k6's own docs associate with "sale of concert
    tickets" and comparable release-moment scenarios — the closest
    documented analogue to "a popular court's Saturday-evening slot opens
    for booking and many users hit Create at once," which is this
    project's actual concurrency risk (see §2).
  - **Soak test** — long duration at moderate load, for slow leaks/resource
    exhaustion, not a first-priority test pre-launch.
  - **Breakpoint test** — deliberately push past capacity to find where it
    breaks, for capacity planning once there's a real user base to plan
    capacity around.
- k6 supports modeling load either by **virtual users** (concurrency-shaped)
  or by **arrival rate** (throughput-shaped, e.g. `ramping-arrival-rate`),
  and its own guidance is to **model the traffic shape you actually
  expect**, not just an arbitrary VU count — for a spike/burst scenario
  specifically, the recommended executor is a ramping-VUs or
  ramping-arrival-rate profile with a fast ramp and short steady peak, not
  a slow linear ramp.

**Recommendation — a concrete first scenario, not an invented number**

Given this project's own already-proven concurrency point (T4: 20
simultaneous `CreateBooking` calls against one court/slot), the natural
first k6 scenario is to make *that exact proof* a repeatable, scriptable
load test rather than a one-off manual/testcontainers run, then widen it
using k6's documented spike pattern:

1. **Spike scenario, contended-slot focus** (mirrors T4, now scriptable and
   re-runnable against a cold-started stack in CI): baseline ~1 VU, spike to
   **20–50 concurrent VUs** issuing `CreateBooking` for the *same*
   court+time-slot within a 1–2 second ramp, hold a few seconds, ramp down.
   Assert (via k6 `checks`/`thresholds`): exactly one 2xx, the rest a clean
   `409`/conflict response — **zero** 5xx or unrecognized errors. This is
   the automated form of the manual proof already in T4, and per §2, run it
   from a cold container start at least once per suite (not only against a
   warm pool) since that's where T4's real bug lived.
2. **Realistic average-load scenario** (secondary, once #1 is solid):
   ramping-arrival-rate across many *different* courts/slots at a modest
   rate (e.g. tens of requests/second) to characterize normal-day latency,
   separate from the contended-slot correctness question — these are two
   different things k6's own test-type taxonomy deliberately keeps
   distinct, and conflating them would hide the deadlock-class bug inside
   generic throughput numbers.

No invented "1000 RPS" style target is proposed — there is no traffic data
yet to justify one; the correctness-under-contention scenario (#1) is the
one this project actually needs *now*, sourced directly from k6's own
documented spike-test pattern and this project's existing T4 evidence.

**Sources**
- Grafana k6 docs, *API load testing* — https://grafana.com/docs/k6/latest/testing-guides/api-load-testing/
- k6-learn, *Load Testing* module (Performance testing principles) — https://github.com/grafana/k6-learn/blob/main/Modules/I-Performance-testing-principles/03-Load-Testing.md
- This project's own primary evidence: `docs/reviews/04-t4-concurrency-invariant.md`

---

## 4. Free/low-cost observability options for a solo Go project

Compared three concrete, currently-documented options (verified against
2026 pricing/docs, since these pages change often — see caveat below each).

| Option | What it gives you free, right now | Fit for this project |
|---|---|---|
| **Grafana Cloud (hosted) free tier** | Per Grafana's own pricing page and secondary pricing trackers current as of 2026: 10,000 active metric series, 50 GB/month each of logs (Loki), traces (Tempo), and profiles (Pyroscope), 3 Grafana viewer/editor users, 14-day retention on all telemetry, 500 k6 cloud VU-hours/month, 100,000 synthetic API check executions + 10,000 browser-check executions/month, ~2,232 host-hours of "Application Observability," 50,000 frontend-observability sessions/month. No credit card required; doesn't expire. | Best turnkey fit: one signup gets metrics+logs+traces+k6-cloud-runs+uptime checks in one place, with headroom well above what a pre-launch solo project generates. Direct Grafana pricing page returned HTTP 403 to automated fetch during this research (likely bot-blocking) — figures above are corroborated across three independent 2026 pricing-tracker pages, not the single primary source; **re-verify the numbers on grafana.com/pricing directly before relying on them**, since third-party trackers can lag. |
| **Self-hosted Prometheus + Grafana (OSS)** | Fully free forever (Apache-2.0/AGPL OSS), no usage caps, retention is whatever your disk holds. Runs comfortably in Docker Compose on a $5–10/mo VPS (roughly 1 vCPU/1GB RAM minimum reported by several self-hosting write-ups) alongside the existing Postgres. | Cheapest and most control (matches CLAUDE.md's "single Go binary + managed Postgres" minimalism), but *you* own the ops burden — backups, upgrades, exposing dashboards securely — which is real cost for a solo dev even if it's not a dollar cost. Reasonable as a *later* option once the hosted free tier's caps start to bind, not the first move pre-launch. |
| **Sentry (Developer/free plan)** | Per 2026 pricing trackers: free forever, 1 user, 5,000 errors/month, ~10,000 performance-monitoring units/month, 50 session replays/month, email alerts. `technology-options.md` already names Sentry explicitly for Go+Vue+iOS error tracking. | Good complementary fit specifically for *error tracking* (crashes, unhandled exceptions across all three clients) rather than metrics/dashboards — pairs with Grafana Cloud rather than replacing it. The 1-user/limited-event caps are a non-issue at solo pre-launch scale but will need a plan decision the moment a co-founder or second developer joins. |

**OpenTelemetry** is not itself a "free tier" (it's a vendor-neutral CNCF
instrumentation standard, not a hosted product) but is the right
*instrumentation* choice regardless of which backend above is picked: the
Go SDK is stable, and instrumenting once with OTel APIs means the export
target (Grafana Cloud, self-hosted Prometheus/Tempo, or a future paid
vendor) can change later without touching application code — avoids
locking the choice above in permanently.

**Recommendation**

Start with **Grafana Cloud's free tier for metrics/logs/traces/uptime
checks + Sentry's free tier for error tracking**, instrumented via
**OpenTelemetry** in the Go backend from day one so the backend choice
stays swappable. This directly satisfies spec §4's "error tracking
(Sentry), structured logs, uptime monitoring" line with zero incremental
cost at pre-launch scale, and defers the self-hosted-Prometheus option to
a later stage where either (a) the free-tier caps start binding, or (b)
cost optimization matters more than solo-dev ops time — neither of which
is true yet.

**Sources**
- Grafana Cloud free-tier figures (secondary, cross-corroborated; primary
  pricing page blocked automated fetch — re-verify directly):
  - https://monitoringcost.com/grafana-cloud-pricing
  - https://cubeapm.com/blog/grafana-cloud-pricing-and-review/
  - https://costbench.com/software/observability/grafana-cloud/
- Self-hosted Prometheus+Grafana on a solo VPS — https://ossalt.com/guides/how-to-self-host-grafana-prometheus-metrics-stack-2026
- Sentry free/Developer plan figures (secondary, cross-corroborated;
  re-verify at sentry.io/pricing directly):
  - https://last9.io/blog/sentry-pricing/
  - https://costbench.com/software/developer-tools/sentry/
- OpenTelemetry project (vendor-neutral instrumentation, stable Go SDK) — https://opentelemetry.io/

---

## Caveat on source freshness

Several primary vendor pricing pages (sre.google, postgresql.org,
grafana.com, dev.to, cybertec-postgresql.com) returned HTTP 403 to this
session's automated fetch tool during research (likely anti-bot
protection, not a content issue) — findings from those domains above are
sourced from search-result summaries and, for the two pricing sections,
cross-corroborated across multiple independent 2026 pricing-tracker sites
rather than a single primary fetch. Anyone acting on the specific
numeric free-tier limits in §4 should confirm them directly on
grafana.com/pricing and sentry.io/pricing before depending on them, since
these change without notice and this research could not load the primary
page directly.
