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
9. **No direct commits/pushes to the shared branch — PR only. No exceptions
   for docs.** Every change — application code *and* process/planning
   artifacts (sprint plans, retros, ADRs, `HANDOFF.md`/`LESSONS.md`
   entries, design docs, review docs) — lands via a branch + PR that is
   reviewed, tested, and explicitly approved before merge. A reviewer/QA/PE
   agent's job is to *report* findings, never to commit or push itself.
   Nothing is "low-risk enough" to skip this on its own judgment — that
   judgment call is exactly the failure mode this rule exists to remove.
   (Added after an incident where a review-only subagent pushed unreviewed
   work directly — see `docs/LESSONS.md`. Tightened to explicitly cover
   docs after a stretch of sprint plans/retros/design docs landing via
   direct push on the reasoning that they were "just docs" — see
   `docs/LESSONS.md`'s "Direct-push-for-docs" entry.)

   **Reviewer-authored fixes — the narrow exception.** (DECISION D2,
   ADR-0016 option (b), answered 2026-09-04.) A session that reviews a
   pull request may commit a fix to that pull request's source branch
   **only when every one of the following five conditions holds.** If any
   one fails, the reviewer requests changes and does not commit.

   1. **Mechanical.** The fix applies an existing, already-decided
      convention that is visible in the file being edited — e.g. adding a
      row to an exhaustive table, or a missing case to a switch whose
      other cases settle the shape. It introduces **no new decision**: no
      new domain rule, no new error mapping not already established
      elsewhere, no schema change, no API or proto change, and no change
      to a test's assertions.
   2. **Compiler- or test-caught.** The gap was surfaced by a failing
      build or a failing test — not by the reviewer's own reading, taste,
      or judgement. The reviewer must quote the actual failure in the
      review.
   3. **Single-file.** The fix touches one file. (A change that must be
      repeated across bounded contexts is **not** single-file and does not
      qualify, even when each edit is individually trivial.)
   4. **Disclosed.** The review states, in its own text: that the reviewer
      authored the fix, what the fix was, the failure that prompted it,
      and the branch it was pushed to. A reviewer-authored fix that is not
      disclosed in the review is a rule-9 violation regardless of its
      content.
   5. **Re-verified.** After the fix, the reviewer re-runs the full gate
      on the fixed tree and reports the result. A fix that is not itself
      re-verified does not qualify.

   This exception covers **fixes to a branch under review**. It does not
   permit a reviewing session to author a feature, and it does not by
   itself settle the separate question of recovering an interrupted
   session's work, which is governed by `sprint-process.md`'s
   worktree-recovery clauses.
10. **A single successful run is not proof of reliability**, especially for
    concurrency claims. Re-run non-deterministic tests (cold start + several
    repeats) before writing "proven" or "reliable" anywhere.
11. **PCI guardrail: never accept a raw PAN/card-number/CVV/track-data field
    on any request DTO, in proto or REST.** Card data is tokenized
    client-side (Stripe.js/Elements/Checkout or the mobile SDKs) and never
    reaches this backend (SAQ A scope). Review every proto change touching
    payment flows against `docs/checklists/proto-review.md` before merging.
    (Added T6.4, `docs/process/t6-sprint-plan.md` kickoff note P1 #13.)

## Commands
- `make test-domain` — run the dependency-free domain + app tests (no DB/codegen).
- `make test-platform` — run `internal/platform/...`'s own tests (the auth
  spine above all). Same Docker-free, codegen-free class as `test-domain`,
  but deliberately a separate target: `test-domain`'s pattern is
  domain+app-only and matches nothing under `internal/platform/`. Part of
  `make ci`. Added T13.4 — before it, those tests ran in no gate at all (#138).
- `make test-adapters` — run every `internal/*/adapter/*` package's tests
  (depends on `generate`; some of them import `internal/gen`). Part of
  `make ci-checks`. Added T14.1 — before it these 22 packages ran in no gate
  (#157), including T13.1's five new behavioural tests and #146's regression
  test.
- `make test-cmd` — run `./cmd/...`'s tests (depends on `generate`). Part of
  `make ci-checks`. Added T14.1: `cmd/server/main_test.go` proves the server
  refuses to start without an auth verifier (#136) and ran in no gate either.
  It was missed by #157 **and** by the T14 plan's own re-enumeration, both of
  which scanned `internal`/`tools` and not `cmd` — the gate-coverage check
  below is what found it.
- `make gate-coverage` — **the standing answer to "which packages hold tests
  that no gate executes?"** Fails, naming the packages, when some package
  holds a `func Test` and nothing reachable from `ci-checks` runs it. Both
  sides are computed at run time: side A from a scan of the tree, side B by
  parsing this repo's own `Makefile` and expanding its `go test` patterns
  with `go list`. **There is no package list in the tool, and adding one is
  the one change that would defeat it** — three sprints running (T11, T12,
  T13/#157) shipped a hand-written glob that was stale before its sprint
  ended. Fix a failure by widening a `go test` pattern in a target reachable
  from `ci-checks`, never by adding an exclusion. Tool in
  `tools/gatecoverage` (covered by `make test-tools`), entry point
  `cmd/gatecoverage`. Added T14.1.

  **It also reports, without failing, every package holding no test function
  at all** — T65.4's answer to #328's blind spot, which is real: the tool's
  question is "which TESTS does no gate run?", and a package with none has
  none to run, so `gate-coverage: OK` was silent about every `cmd` package
  holding no test — **four of five when #328 was filed**, three now that
  T65.4 tested one — and about `internal/identity/adapter/postgres`, an
  adapter. Run `make gate-coverage | grep "^    cmd/"` for the live list
  rather than trusting that count. The
  decision, recorded rather than left as a convention: **report, never fail**
  (#328's option (b); its (a) and (c), and the "fail on it" option nobody
  offered, are rejected with reasons in `tools/gatecoverage`'s `Untested` doc
  comment). A gate that failed here is satisfied by a stub `func TestNothing`,
  and it would assert a policy — every package must have a test — that nobody
  decided and that is wrong for a thin `main` wrapper. Generated packages are
  counted rather than listed, and are recognised by **Go's own `DO NOT EDIT`
  header**, not by a path prefix: `internal/gen` as a hardcoded skip would be
  exactly the package list this tool must not grow. Current state, from the
  running command rather than from this paragraph — **and the figure depends
  on which of this project's two documented tree states you are in**, because
  `gate-coverage` has no `generate` prerequisite and 13 of the 27 are the
  gitignored `internal/gen/**`:

  ```
  $ make generate >/dev/null && make gate-coverage | grep "no test function"
  gate-coverage: NOTE — 27 package(s) hold no test function at all.

  $ rm -rf internal/gen && go run ./cmd/gatecoverage | grep "no test function"
  gate-coverage: NOTE — 14 package(s) hold no test function at all.
  ```

  The second is a fresh clone before any codegen, and in that state the
  generated/hand-written split is a no-op — which is also why
  `tools/gatecoverage`'s real-repository test **skips** rather than fails
  there (`test-tools` is deliberately codegen-free). Both figures measured
  2026-10-07.
- `go run ./cmd/docsindex -statuses` — **list every ADR's status** (file, token,
  which form carried it) from the **same parser `docs-index-check` uses**. This
  is what Ceremony 1's escalation sweep should use; `sprint-process.md` requires
  reading the status line rather than grepping the body: **ADR-0015** preserves
  *"Escalated — awaiting product decision"* and **ADR-0016** preserves
  *"Escalated — awaiting the user's decision"*, both beneath a supersession
  notice, so a grep for either string hits a resolved decision. (This bullet
  attributed one quote to both files until T64's review read them; the quote
  was copied from `sprint-process.md` rather than verified.) Added T64.5 — before it the gate could *refuse* an
  unreadable status and could not *print* a readable one, so T64's own ceremony
  copied the package into a scratch directory rather than re-implement the
  parser as a grep — a bet this project has lost **repeatedly**, most recently
  on ADR-0012, whose `## Status` is followed by a blank line so `tail -1` took
  the blank. (*"Five times"* stood here until T64's review pointed out that
  `sprint-process.md` says in as many words: if a count cannot be enumerated,
  write *"repeatedly"* rather than *"five times"*.) Unclassifiable
  statuses fail this listing too, not just the gate. **Tested as of T65.4**
  (#328): `cmd/docsindex/main_test.go` drives the listing, the tally and — in
  a subprocess, because an exit code is the only part of this program a
  Makefile can see — the **exit 2** on an unclassifiable status. It shipped at
  T64.5 with no test at all, which rules 1 and 8 do not allow.
- `make binary-check` — fails when a tracked file carries the **ELF magic
  number**, i.e. a compiled binary has been committed. Part of `make
  ci-checks`. Added T65.4's review pass, which found a 3 MB `docsindex`
  executable committed at the repository root **by that same sprint**:
  `go build ./cmd/<x>` with no `-o` drops the binary there, named after the
  package, and `git add -A` tracks it. Nothing noticed, and once TRACKED
  `git status` stays clean, so the next build lands a 3 MB diff in an
  unrelated commit — and a stale executable sits where `./docsindex
  -statuses` works, answering a ceremony's question from code that no longer
  exists. Build to a scratch path (`go build -o /tmp/<x> ./cmd/<x>`).
  `.gitignore` also lists the five `cmd/` names, and this check exists not
  to depend on that list: it tests the four-byte property, so a sixth
  command or a binary committed from anywhere else fails it too. Images,
  fonts and PDFs are binary but not ELF, so there is nothing to exempt — and
  an exemption list here would be the same mistake the `.gitignore` list
  already is.
- `make lock-check` — fails when `web/package-lock.json` is missing, or when it
  and `web/package.json` disagree. Wires `npm ci --dry-run --offline` (npm's own
  check, no network, ~0.6s) rather than re-implementing semver comparison. Part
  of `make ci-checks`. Added T64.4: the lockfile is what actually holds an
  advisory fix in place (see the npm gotcha below), a fresh resolve cannot
  regenerate it on this graph, and nothing noticed it going missing.
- `make generate` — buf + sqlc → `internal/gen` and `openapi/`.
- `make tidy` — `go mod tidy` (run after first generate).
- `make vet-integration` — `go vet -tags=integration ./...` (depends on
  `generate`). Compiles the `//go:build integration` files without Docker;
  does not run them. Part of `make ci`.
- `make test` — full suite: race + JUnit + coverage.
- `make up` / `make down` — run / tear down via docker compose. `up` supplies
  `AUTH_ISSUER`/`AUTH_AUDIENCE`/`AUTH_JWKS_FILE` from the committed local-dev
  key fixture in `dev/auth/` (T14.9, #160) — without them the server refuses
  to start, by design (T13.5).
- `make dev-token` — mint a local-dev bearer token against that fixture, so an
  authenticated RPC can be exercised locally: `TOKEN=$(make -s dev-token)`,
  then `curl -H "Authorization: Bearer $TOKEN" …`. **Dev-only and public** —
  the keypair is committed, therefore worthless; never point a deployment at
  `dev/auth/`. See `dev/auth/README.md`.
- `make lint` — golangci-lint.
- `make fmt-check` — fails if `gofmt -l ./internal ./cmd ./tools` names any
  file. Part of `make ci-checks` (after `generate`). Added T14.2; before it
  there was no formatting gate anywhere, and `.golangci.yml` enables no
  formatter, so `make lint` does not cover this — the two checks are
  additive. Never use `gofmt -w` as the gate: it would rewrite CI's checkout
  and report green.

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

## Docs index & naming convention
`HANDOFF.md` has a **Docs index** section, organized by phase (T0..T6+),
linking every sprint plan, retro, review, ADR, and design doc relevant to
that phase. **Read it before starting any task** — it's the map; this file
is the rulebook. Naming, so a doc's phase/task/workstream is legible from
its filename alone and nothing collides or goes stale silently:
- Sprint plans: `docs/process/t{N}-sprint-plan.md`
- Sprint retros: `docs/process/t{N}-retro.md` (one per sprint; retro
  ceremony output is a distinct artifact from `docs/LESSONS.md`'s
  incident postmortems — don't fold one into the other)
- Ticket/PR reviews: `docs/reviews/{NN}-t{N}-{slug}.md` when committed as a
  file (T0–T4); from T5 onward, ticket reviews are posted as GitHub PR
  reviews directly (`pull_request_review_write`) rather than committed
  files — the PR *is* the review record, no separate file needed.
- ADRs: `docs/adr/{NNNN}-{slug}.md`, globally numbered (not phase-prefixed
  — an ADR is a cross-cutting decision, not tied to one phase).
- Design workstreams: `docs/design/{workstream}-{artifact}.md`, e.g.
  `docs/design/v1-system-design.md` and `docs/design/v1-review-round-{N}.md`
  — the workstream tag (`v1`, `v2`, ...) is what prevents a second design
  pass from colliding with or shadowing the first.
- `docs/LESSONS.md` stays one running, append-only file (per its own
  header) — entries are dated/phase-tagged via `##` headers, not split
  into per-phase files, since a single chronological postmortem log is the
  point.

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
- **Integration-tagged test files are counted by this command, not by this
  paragraph** (T64 review fix — the paragraph said 26 across 5 contexts with
  payments at 6, and the tree said 27 with payments at 7, because T62.5 added
  one *after* T61 re-derived the figure):

  ```
  $ for f in $(find . -name '*_test.go'); do \
      head -5 "$f" | grep -q '^//go:build integration' && echo "$f"; done \
      | sed 's|^\./internal/\([^/]*\)/.*|\1|' | sort | uniq -c
  ```

  **`grep -rl "go:build integration"` — which this paragraph used to prescribe
  — cannot produce that number**: it returns 45 paths, because the string also
  appears in docs, the `Makefile`, the `Jenkinsfile`, a migration and
  `tools/gatecoverage`'s own fixtures. A prescribed verification command that
  does not reproduce the figure is worse than none. They
  need Docker (testcontainers-go) to *run*, so they are excluded from
  `make test-domain` and from plain `go test ./...`/`go vet ./...` — which
  means a break in one of them is invisible to every gate a machine without a
  Docker daemon can run. `make vet-integration` (`go vet -tags=integration
  ./...`, depends on `generate`) closes that hole by **compiling** them
  without Docker, and `make ci` runs it after `test-domain`. *Executing* them
  still needs Docker: `make ci-integration`, or `make test`. Added T12.1 after
  booking's `concurrency_integration_test.go` broke twice in T11 with every
  runnable command reporting green — see `docs/process/t11-retro.md` finding 2.
  **`make gate-coverage` does NOT report these packages, and the sentence that
  used to say it did was false** (T64 review fix). The tool classifies a
  package as compiled-but-never-executed only when it holds **no** runnable
  tests — `tools/gatecoverage/gatecoverage.go`'s `CompiledOnly` branch — and
  every integration-bearing package here also holds untagged tests
  (socialplay/adapter/postgres 10 tagged + 1 untagged, booking 4 + 5,
  facilities 2 + 3, and so on). So each one passes on its untagged siblings and
  **every integration file is invisible to `gate-coverage`**:

  ```
  $ make gate-coverage | grep -iE "ONLY build-tagged|compiled"     # no output
  ```

  **The grep above was `-iE "NOTE|compiled|never"` until T65.4**, which added a
  second, unrelated NOTE to this tool's output (packages holding no test
  function at all, #328) and so made the old command print a line that has
  nothing to do with integration tests. Narrowed to the `CompiledOnly`
  section's own wording, which is what the claim is about.

  That is not a defect in the tool — a package with runnable tests that run is
  correctly covered — but do not read `gate-coverage: OK` as saying anything
  about the tagged files. Only `make ci-integration` does. **The counts above are narrative, not a gate** — do not
  hand-edit them into a checklist, and never add an exclusion list to
  `tools/gatecoverage` to keep them true.
- **Docker works here. Start it and run `make ci-integration`.** Every sprint
  from T4 to T60 recorded "no Docker daemon available" and shipped
  integration tests nobody had executed. The `dockerd`/`containerd` binaries
  were present the whole time; the daemon starts in about four seconds
  (`dockerd` detached, then poll `docker ps`), and the full suite takes
  a couple of minutes (measured 100–106s on a quiet container; the run prints
  its own count, so no number is carried here). T61 ran it for the first time
  and it reported 34 failures — including **two live production defects**. A
  **third** was found in the same sprint by the *review* of the fix, which
  diffed every historical body of `enforce_game_capacity()`, and
  `t61-retro.md` §4 is titled *"The third defect — found by the review, and not
  findable by the suite"* — so crediting the suite with three, as this
  paragraph used to, overstates what running it buys (T64 review fix).
  `db/migrations/0030` fixes two of the three, `0031` the third.
  Each had been reported green by every Docker-free gate from the sprint that
  broke it — **T8.7, T19.1 and T10.6 respectively** — until T61. (Those are
  ticket numbers, not a count of elapsed sprints: T61's own first draft of this
  paragraph said "18 and 22 sprints", which was two migration-distances
  mislabelled as sprints and one invented figure. See
  `docs/process/t61-retro.md` §4.) So: **compiling an integration test is not a
  substitute for running it, and "the environment can't" is a claim to test
  before it is written down.** A session that changes a migration, an actor
  column, or anything a `*_integration_test.go` touches is expected to run
  `make ci-integration`, not `make vet-integration` alone.

  **Two per-container setup steps, since a reclaimed container loses both**
  (both cost minutes and both have now bitten twice):
  1. `$(go env GOPATH)/bin` is empty on a fresh clone, so `make generate` has
     no `buf` — `go install` the four protoc plugins plus `buf`, `sqlc` and
     `gotestsum` first (the list is in the `buf generate` gotcha above).
  2. `make test` reports *errors* rather than failures — `go: no such tool
     "covdata"` — because this toolchain ships `cmd/covdata`'s source and no
     built binary. Build it and place it in the active `GOROOT`:
     `go build -o /tmp/covdata cmd/covdata`, then `chmod -R u+w
     "$(go env GOROOT)/pkg"` and copy it into
     `$(go env GOROOT)/pkg/tool/linux_amd64/`. **Verify the copy landed** —
     the module-cache `GOROOT` is mode `555`, so a `cp` without that `chmod`
     can report success and leave nothing behind.
- **`make security`: the Go half cannot run here, and that is not a reason to
  skip the npm half.** Tested at T63.2, not quoted: `govulncheck` gets
  `Forbidden` fetching `https://vuln.go.dev/index/modules.json.gz` from this
  environment. The Makefile's designed response is `SKIP_GOVULNCHECK=1`, which
  warns loudly and removes the report so `tools/vulngate` gates on npm findings
  only. **Never set it in CI**, and never set it inside a `ci-*` target — it is
  for a human in an environment that cannot reach the database.

  Why this is a gotcha and not a footnote: `security-go` runs **before**
  `security-npm`, so for the life of this project the Go half's failure meant
  **the npm half had never run at all**. T61's and T62's retros both recorded
  "govulncheck is owed" without testing it. Running it once, with the documented
  flag, surfaced **7 vulnerabilities — 5 of them `high`** — in shipped web
  dependencies. **A broken half of a gate is not permission to skip the whole
  gate.**

- **Some of npm's remediation commands crash on this project's dependency
  graph.** npm 10.9.7, reproducibly: `npm audit fix` and a **fresh**
  `npm install` (no lockfile) die with
  `npm error Cannot read properties of null (reading 'edgesOut')`. `npm install`
  **with the lockfile present** works, and `npm ci` works.

  **`npm update <pkg>` is package-dependent, not a blanket crash** (T64 review
  fix): it crashes on `vitest` and **succeeds** on `nanoid`, `vue`, `js-yaml`
  and `undici`. Stated as universal — as it was here until T64's review tested
  each one — it tells a reader not to try the command that would have worked.
  **And npm ≥ 11 resolves the case 10.9.7 crashes on**: npm 12.2.0 in a scratch
  prefix completes the `vitest` bump with `found 0 vulnerabilities` (see #320,
  where the remaining half — running the web suite on that version — is still
  owed).

  Consequences, learned the hard way at T63.2:
  1. **Fix transitive advisories with an `overrides` block in
     `web/package.json`**, pinned to the first version *above* the vulnerable
     range in the *same major line* — not to `latest`. All five of T63.2's high
     findings had a same-major patch fix, so no breaking bump was needed; read
     the advisory's `range` rather than reaching for the newest version.
  2. **Do not delete `web/package-lock.json`.** It is what makes installs work
     here; without it npm cannot resolve this graph at all. T63.2 deleted it
     while investigating and had to restore it from a copy.
  3. **Read `metadata.dependencies.total` beside every `npm audit` tally.**
     The habit is sound and is what caught T63.2's real problem; **the
     mechanism this bullet used to assert is not reproducible** (T64 review
     fix). Tested both configurations on npm 10.9.7:

     ```
     lockfile present, node_modules absent -> {moderate: 2, high: 0}, deps 368   # correct
     no lockfile, no node_modules          -> npm error code ENOLOCK, exit 1     # refuses
     ```

     npm audits **from the lockfile**, so it is right without `node_modules`,
     and without a lockfile it refuses rather than lying. So this was a
     misattribution, not a third instance of vacuous green — keep the habit,
     drop the claim.

- **An in-memory fake is more permissive than Postgres, and that asymmetry
  hides storage bugs.** `0031`'s defect — `payments.payable_type`'s CHECK
  never widened for `competition_entry` — passed every unit-level test for 22
  sprints, because the in-memory Payments repository has no CHECK constraint
  to violate. The fixtures proved the routing and hid the storage. When a
  domain enum, status set, or payable type gains a value, the schema half is
  part of the same ticket (rule 4), and the test that pins it belongs where a
  real database can refuse it. See
  `internal/payments/adapter/postgres/payable_type_conformance_integration_test.go`
  for the shape: derive the set from the source, never list it.
- **`CREATE OR REPLACE FUNCTION` replaces the whole body, and this project has
  got that wrong twice in the same function.** `enforce_game_capacity()` has
  been redefined five times. `0012` added guest weighting and silently dropped
  `0007`'s waitlist-promotion reservation; `0023` added a cancelled-Game check
  and silently dropped `0012`'s weighted sum. Both rebuilt their body from
  `0006`'s version, and `0023`'s header even states the function is "unchanged
  by this migration (that function is 0006's)" — recording which migration
  *created* the function rather than which last *defined* it. `0030` is the
  union of all four. **Before redefining a function, diff your new body
  against the migration that last defined it**, found by grepping every
  `CREATE OR REPLACE FUNCTION <name>` across `db/migrations`, not against the
  one that created it. A redefinition that reads as purely additive is exactly
  the shape this failure takes.
- **A DB-level guard with no DB-level test is a comment.** `0007`'s reservation
  shipped with no test on either side of the boundary, so when `0012` (T8.7)
  dropped it nothing failed until T61 — seventeen migrations landed in between.
  Its test
  (`internal/socialplay/adapter/postgres/waitlist_reservation_integration_test.go`,
  T61) has to drive the **repository**, not `app.Service` — the app layer's own
  pre-check refuses the call before Postgres sees it, so a test through
  `app.Service` passes against a trigger with no such logic at all. That is
  how the regression stayed invisible, and it is the general shape: to test
  the Postgres half of rule 4, the test must bypass the Go half.
- **The lockfile, not the `overrides` block, is what holds an npm fix in
  place** — so **to mutation-verify a dependency pin, revert
  `web/package-lock.json`, not the declaration.** Measured at T63
  (`docs/process/t63-retro.md` §3): reverting the lockfile to its pre-fix state
  returns all five advisories by name, while **deleting the `overrides` block
  changes nothing at all**, because `npm install --package-lock-only` will not
  downgrade a package that already satisfies its range. A pin's removal is
  therefore a silent no-op today that bites at the next fresh resolve — which
  crashes on this graph, per the npm gotcha above. Never treat the `overrides`
  block alone as the record of what is pinned, and never read "removing it
  changed nothing" as evidence that it was unnecessary.

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
