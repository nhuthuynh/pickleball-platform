# ADR-0011: CI pipeline shape and security gating

- **Status:** Accepted
- **Date:** 2026-08-05
- **Ticket:** SCRUM-6
- **Supersedes:** the T0 bootstrap `Jenkinsfile` placeholder (never executed)

## Context

`Jenkinsfile` had been the T0 placeholder since the repo was bootstrapped.
HANDOFF.md flagged "no CI is configured" as an open gap across T7, T8 and
T9 — three sprints — and noted that every "tests pass" claim in PRs #41
through #92 was an agent reporting the result of a command it ran in its
own sandbox, never an independently checkable signal.

Two constraints shape everything below:

1. **Docker is frequently absent.** Not hypothetically: it is a
   repeatedly documented gotcha in CLAUDE.md, it is why the T4
   concurrency test is behind a build tag, and it was absent again in the
   environment where this ticket was implemented. A pipeline that assumes
   a Docker daemon does not run here.
2. **The pipeline must not lie.** A CI system whose green tick sometimes
   means "checked nothing" is worse than no CI, because people act on it.

## Decisions

### 1. `agent any`, not `agent { docker { ... } }`

The placeholder ran the whole build inside a `golang:1.24-bookworm`
container, which (a) requires a Docker daemon merely to start, and (b)
has no Node, so it could never have built the Vue app that landed in T7.

Instead the pipeline runs on the agent directly and bootstraps its own Go
tooling into a workspace-local `GOPATH`. The agent must supply only Go,
Node, npm, git, make and curl.

**Trade-off:** less hermetic than a container. Accepted because a
hermetic pipeline that cannot start is worth less than a slightly leaky
one that runs. Pinned tool versions recover most of the reproducibility.

### 2. Generate before Lint

`internal/gen/` and `web/src/api/generated/` are both gitignored. The
Postgres/grpcapi adapters, `cmd/server`, and most of the Vue app do not
compile until they are generated, so neither `golangci-lint` nor
`vue-tsc` can run on a clean checkout before codegen.

The stage order requested in the ticket (Lint, then Unit tests, then
Generate) therefore cannot work in this repository, and the pipeline
deviates from it deliberately: **Checkout → Toolchain → Generate → Lint →
Unit tests → Build → Integration → Security → Load.**

### 3. Skipped stages mark the build UNSTABLE, never green

When no Docker daemon is present the integration stage does not run, and
that is exactly the stage proving the no-double-booking invariant holds
under concurrency. Silently passing would be the single most misleading
thing this pipeline could do.

So: skip marks the build `UNSTABLE` with a message saying the invariant
is unverified, and `FAIL_ON_SKIPPED_INTEGRATION` upgrades that to a hard
failure on the agent guarding the shared branch. Same principle applies
to the vulnerability scan — an unreachable vuln database reads as
"findings UNKNOWN", never "no findings".

This is CLAUDE.md rule 10 expressed in a pipeline: a run that did not
check something must not look like a run that checked it and found
nothing.

### 4. Severity model for the security gate

Two scanners, one gate (`cmd/vulngate`, logic in `tools/vulngate`):

- **npm audit** reports a real per-advisory severity, so gating is
  literal: `high` and `critical` fail the build.
- **govulncheck** does *not*. The Go vulnerability database carries no
  CVSS score, so there is no HIGH/CRITICAL field to read. What it reports
  instead is **reachability**. We gate on `called` — a vulnerability
  whose affected symbol this code actually invokes — and report
  `imported`/`required` without failing.

Reachability is arguably the better signal: CVSS scores a vulnerability
in the abstract, while `called` answers whether *this* codebase executes
the affected path.

### 5. Baselines must carry a written reason

Pre-existing findings that cannot be fixed in the ticket that discovers
them are recorded in `security/vuln-baseline.json`. `vulngate` **rejects
a baseline entry with an empty reason**, so a silent suppression is
impossible by construction rather than by convention. Entries are
scanner-scoped, so an npm entry cannot accidentally silence a same-named
Go finding.

The gate is real code with real tests rather than a shell one-liner
because "fail on new but not pre-existing" has genuine edge cases —
malformed reports that must not read as clean, scanner-scoped matching —
and CLAUDE.md rule 1 wants logic like that under test.

### 6. Load tests are opt-in, not a per-PR gate

k6, chosen over JMeter/Gatling/Locust/Vegeta because scenarios are plain
JavaScript (a language this repo already has), it is a single static
binary with no runtime to provision, and its `thresholds` make a run
objectively pass/fail. Full comparison in `loadtest/README.md`.

It does not gate PRs: load numbers are only meaningful against a stable
baseline, a per-PR measurement on a noisy shared agent is not that, and
gating merges on a timing threshold teaches people to hit "retry" until
it passes — which destroys the value of the one signal that should mean
something.

## Consequences

- CI can, for the first time, contradict an agent's claim that a change
  is green. It already did: wiring the checks up surfaced three
  pre-existing `vue-tsc` failures on the shared branch (T9.2 added entry-fee
  fields to `GameSummary`; T8.9's fixtures were never updated, and were
  rendering `$NaN`) and one `staticcheck` finding — none of which any
  amount of local `make test-domain` would have caught.
- `make ci` runs the same sequence locally, so the parity this project
  values is enforced by construction rather than by discipline.
- The pipeline is only half of "CI is wired". Job creation, the GitHub
  webhook, plugins and branch protection are all server-side and cannot
  be done from this repository — they are listed in the `Jenkinsfile`
  header and must be done by a Jenkins admin before any of this runs.
- Deploy remains unimplemented: it needs a registry credential that does
  not exist in this environment, and SCRUM-6 explicitly forbids adding
  stages that require absent secrets. The stage is a disabled stub naming
  what it would need.

## Alternatives considered

- **GitHub Actions instead of Jenkins.** Would have given a working
  pipeline faster and needs no server-side setup. Rejected: Jenkins is a
  locked stack decision in CLAUDE.md ("Locked decisions — do NOT
  reopen"), and this ticket is not the place to reopen it. Worth raising
  separately, since the server-side setup burden is the main thing
  standing between this file and a running pipeline.
- **Vendoring `internal/gen` into git** to drop the Generate stage.
  Rejected: contradicts CLAUDE.md rule 6 and ADR-0003.
- **`golangci-lint` with a wider linter set** (revive, gosec, errorlint).
  Deferred: turning on new linters across five bounded contexts would
  bury a pipeline change under unrelated code churn. `.golangci.yml`
  pins today's default set explicitly so widening it later is a visible,
  reviewable diff.
