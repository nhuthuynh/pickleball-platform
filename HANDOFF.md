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
| T6 | `docs/process/t6-sprint-plan.md` | not yet written | PRs #23, #27, #28 merged (T6.1–T6.5, in that dependency order — #24/#26 closed without their own merge, since their commits already landed as ancestors of #27's own hand-resolved 3-way merge); #25 merged (T6.6) — all reviewed via GitHub review comments, see naming convention. T6.7 not yet implemented, no PR | `adr/0005` (currency column, referenced by T6.1), `adr/0006`'s Status section (rewritten by #25 to say what actually shipped) | `docs/design/v1-system-design.md` + `docs/design/v1-review-round-{1..10-final}.md` (10-round Designer+PM+PE+PO review of the requirements list gathered mid-T6; two open items need the user's product/legal sign-off — see the design doc's top blockquote) |
| T7 | `docs/process/t7-sprint-plan.md` | not yet written | PRs #40 (T7.2) → #41 (T7.1) → #42 (T7.3) → #43 (T7.7) → #45 (T7.5) → #44 (T7.4, loop 2) → #46 (T7.6), in that merge order — all merged, all reviewed via GitHub review comments, see naming convention | none new this phase | `docs/design/v1-external-reference-reconciliation.md` (reconciles the external design handoff against the v1 review, resolves T7's five open UX questions) + `docs/design/handoff-2026-08/` (the external handoff itself) |
| T8 | `docs/process/t8-sprint-plan.md` (re-scopes T7's roadmapped T8/T9 — see its own re-scope notice at the top) | not yet written | PRs #59 (T8.1) → #60 (T8.5) → #62 (T8.6) → #61 (T8.2) → #63 (T8.4) → #64 (T8.3) → #65 (T8.7) → #66 (T8.8) → #67 (T8.9) → #68 (T8.10), in that merge order (verified against each PR's `merged_at` timestamp) — all merged, all reviewed via GitHub review comments, see naming convention | none new this phase | `docs/process/t8-sprint-plan.md`'s own re-scope notice (supersedes T7 plan's T8/T9 lines) |
| T9 | `docs/process/t9-sprint-plan.md` (supersedes T8 plan's T9/T10 lines — see its §A5 roadmap update) | `docs/process/t9-retro.md` (8 findings, 3 recorded as unresolved disagreements; indexed from `docs/LESSONS.md`'s `## T9 sprint retro`) | PRs #81 (T9.8) → #84 (T9.1) → #83 (T9.10) → #82 (T9.9) → #85 (T9.2) → #86 (T9.3) → #87 (T9.4) → #88 (T9.5) → #89 (critical fix, unticketed) → #90 (T9.7) → #91 (T9.6), in that merge order (verified against each PR's `merged_at` timestamp) — all merged, all reviewed via GitHub review comments, see naming convention | `adr/0009` (owned-channel messaging + social-account OAuth custody deferred until real auth, T9.8), `adr/0010` (auto-matching is built with the Identity/Users context and not before — a **sequencing** decision, not a scope reversal, with a binding T10 Ceremony 1 trigger and two product/legal questions escalated to the user; T9.9) | — |

| T10 | `docs/process/t10-sprint-plan.md` (Ceremony 1 resolves ADR-0010's binding trigger; Ceremony 2 tickets Identity/Users + `Match` plus three T9 follow-up issues #96–#98) | not yet written | not yet opened | `adr/0012` (supersedes `adr/0010`: Identity/Users + `Match` built this sprint; `PlayerRating`, the matching algorithm, and gender-mix matching remain named-blocked on Q1/Q2, escalated to the user, trigger tied to the user's answer rather than another sprint boundary) | — |

| SCRUM-6 (CI/CD, cross-cutting — not a phase) | — (Jira ticket, not a sprint) | — | PR for `SCRUM-6-cicd-pipeline` (GitHub review comments, see naming convention) | `adr/0011` (CI pipeline shape + security gating: `agent any` over a Docker agent, Generate-before-Lint, skipped stages mark UNSTABLE not green, reachability as the Go severity signal, baselines must carry a written reason, load tests opt-in) | `loadtest/README.md` (k6 choice + its verification-status table) |

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

**T5 (Social Play) and T6 (Payments, minus T6.7): all reviewed and MERGED**
into `claude/go-backend-pickleball-7up34j` as of this entry. Merge order:
#29 (docs governance) → #11 → #12 → #13 → #14 → #15 (T5.1–T5.5) → #25
(T6.6) → #23 (T6.1) → #27 (T6.4, folding in T6.1–T6.3) → #28 (T6.5, folding
in T6.1–T6.4 + T5.1–T5.5). #24 and #26 (T6.2, T6.3) were **closed without
their own merge**, not left unmerged in the sense of "not landed" — GitHub
shows them `merged: true` too, because their commits are ancestors of #27's
own hand-resolved 3-way merge; closing them (rather than merging separately
afterward, which would have duplicated/conflicted with content already in
#27) is what the GitHub UI calls a close, even though the commits did land.
Most, not all, merges past #29 hit a real git conflict from stacked
branches whose base had moved (#13, #14, #15, #25, #27, #28 did; #11, #12,
#23 merged clean) — every conflict that did occur was resolved on the
source branch (never a direct push to the shared branch) and re-verified
(`go build`/`go vet`/`go test -race` across
`internal/{booking,socialplay,payments}/{domain,app}`) before merging; see
each conflict-resolution commit's message for specifics (the
`internal/socialplay/app/service_test.go` one, on #28, is the one worth
reading if this class of stacked-PR conflict recurs — two different tests'
similar boilerplate confused the line-based diff badly enough that
reconstructing from each side's real content was safer than patching the
markers). Post-merge, the full domain+app suite is green across all three
contexts (`go test ./internal/.../domain/... ./internal/.../app/... -race
-count=1`) — this is a live, verified claim as of this entry, not carried
forward from a pre-merge PR description.

**T7 (Web client foundation + Facilities context): all 7 tickets (T7.1–T7.7)
implemented, reviewed, and MERGED** into `claude/go-backend-pickleball-7up34j`
as of this entry. Merge order: #40 (T7.2, Facility/Court domain) → #41 (T7.1,
Vue 3 scaffold) → #42 (T7.3, Facilities Postgres/proto/gRPC) → #43 (T7.7,
Facilities object-level authorization — found and closed a real gap, no
ownership check existed before this ticket) → #45 (T7.5, Discover/browse UI)
→ #44 (T7.4, Facility onboarding UI, merged in 2 loops — loop 1 found a real
blocking cross-PR gap: #44 never sent the `actor_user_id` field #43 made
required, which would have 403'd every onboarding submission; fixed and
re-verified in loop 2) → #46 (T7.6, quote + book UI). There is now a real
Vue 3 web app (`web/`) and a real Facilities backend context
(`internal/facilities/{domain,app,port,adapter}` +
`proto/pickleball/facilities/v1`), reachable end to end: create a facility →
add a court → get a live quote → book it, all against the real gRPC-gateway
REST API. Known gaps carried forward, not silently dropped (see "Not yet
built" and Cross-cutting below for detail): Facilities has no
courts-listing endpoint yet (a facility's courts list always renders empty
in the UI); there is no client-side router yet (`App.vue` mounts every T7
screen as stacked siblings); `games.facility_id text` (Social Play,
pre-T7) is still unreconciled with the new `facilities.id uuid`.

**T8 (Social Play + Payments UI spine, closing T7's carried gaps): all 10
tickets (T8.1–T8.10, 51 points) implemented, reviewed, and MERGED** into
`claude/go-backend-pickleball-7up34j` as of this entry. Merge order
(verified against each PR's `merged_at` timestamp, not just the plan's
intended dependency order): #59 (T8.1, Vue Router) → #60 (T8.5, Payments
authz — closed the long-open T6.7 gap) → #62 (T8.6, Game/Registration
domain fields) → #61 (T8.2, Facilities courts-list) → #63 (T8.4,
AttestCameraConsent) → #64 (T8.3, `games.facility_id` reconciliation) →
#65 (T8.7, guest-capacity proto/DB wiring, including a rewritten
weighted-sum capacity trigger) → #66 (T8.8, Social Game creation UI) → #67
(T8.9, Discover & Join Games UI, added a new `ListGames` RPC) → #68 (T8.10,
Payments UI, added a new `ListRegistrationsForGame` RPC). All of
T7.1–T7.6's carried gaps this
sprint was scoped to close are closed: real client-side routing exists
(`vue-router`), Facilities has a real courts-list read path, camera consent
has a real server-side attest path, and `games.facility_id` is reconciled
with `facilities.id` via a real FK + `port.FacilityLookup` boundary. There
is now a real, clickable Flow 3/4/6: a Host publishes a Social Game at a
real Facility with a real payment method and guest allowance, a Player
discovers it, joins it with guests, and pays — online (Stripe-stub
checkout, PCI guardrail verified clean) or cash (surfaced as pending to the
Host until settled). Three merge conflicts were resolved by hand across the
sprint (T8.2↔T8.4 on `internal/facilities/port`/`adapter/postgres`;
T8.3↔T8.7 on `socialplay.proto`'s field numbering plus a real
`fromProtoPaymentMethod` validation bug the conflict-resolution pass fixed;
T8.8↔T8.9 on `web/src/router/index.ts` and a duplicate `socialplayClient.ts`)
— every conflict was resolved on the source branch, never a direct push to
the shared branch, and re-verified (`go build`/`go vet`/`go test -race`
plus `npm run build`/`npm run test`) before merging. Known gaps carried
forward, not silently dropped (see Cross-cutting below): Social Play has no
price/fee field at all (T8.10 used a disclosed, visibly-labeled placeholder
amount), no CI is configured on this repo, and the `npm install
--legacy-peer-deps` friction from T7 was still open at the end of T8
(closed since, by T9.10 — see Cross-cutting).

**T9 (Competitions context + growth/social decisions): all 10 tickets
(T9.1–T9.10, 49 points) implemented, reviewed, and MERGED**, plus one
critical out-of-band fix (PR #89, unticketed — see below). Full re-scope
reasoning and all 10 tickets: `docs/process/t9-sprint-plan.md`. Merge
order (verified against each PR's `merged_at` timestamp): #81 (T9.8,
ADR-0009) → #84 (T9.1, Competition/CompetitionEntry domain) → #83 (T9.10,
npm chore) → #82 (T9.9, ADR-0010) → #85 (T9.2, Game `EntryFee`, retiring
T8.10's placeholder) → #86 (T9.3, Competitions app service) → #87 (T9.4,
Postgres/proto/gRPC + weighted DB capacity guard) → #88 (T9.5, shareable
registration links) → #89 (critical fix — see below) → #90 (T9.7,
Player-facing Competition UI) → #91 (T9.6, Host-facing Competition UI).
There is now a real, fifth bounded context
(`internal/competitions/{domain,app,port,adapter}` +
`proto/pickleball/competitions/v1`) reusing Booking's existing
`competition`-source invariant with zero changes to `internal/booking/`,
and a real, clickable Flow 5: a Host creates a multi-session Competition at
a real Facility, gets an honest share link (no fake "Connect account"
buttons — ADR-0009 at the UI layer), a Player discovers it or follows the
link, and enters with guests under a DB-level weighted capacity guard
(`FOR UPDATE`-locked trigger, proven with 10 concurrent-entry runs
including 3 cold starts against real Postgres). Two real decisions
replace what would otherwise have been further deferrals: ADR-0009 defers
OAuth/inbound-messaging custody until real auth exists (a credential-custody
argument, not budget), and ADR-0010 commits auto-matching to the sprint
that builds Identity/Users with a binding T10 trigger — the fourth time
this project touched the question (after three prior deferrals) and the
first time it produced an actual decision rather than a fourth deferral
(verified by an adversarial PE+PO review that went looking for
dressed-up-deferral #4 and didn't find it).

**Critical fix, found mid-sprint, not part of any ticket (PR #89):** a
PE+QA review of PR #88 (T9.5) noticed the Competitions ID-keyed read next
to the share-token read had no input guard, and pulled the thread —
`cmd/server` registered **zero gRPC interceptors**, so any unauthenticated
request with a malformed ID (e.g. `GET /v1/competitions/not-a-uuid`)
**crashed the entire server process**, taking every bounded context and
every in-flight request down with it (`net/http`'s per-connection panic
recovery does not carry over to grpc, which installs none of its own — the
reviewer's own words: "review intuition carried over from HTTP handlers is
wrong here"). Independently reproduced live on both branches (real
Postgres, real `cmd/server` binary): 6 distinct crash vectors on the base
branch, all surviving as clean errors on the fix branch, with a normal
request immediately after each attack still succeeding — the part that
actually proves recovery rather than just error-code mapping. Fixed with
two layers: a global gRPC panic-recovery interceptor in `cmd/server`
(protects all five contexts, present and future, logs a stack trace,
never echoes panic detail to the public caller) plus boundary-level UUID
shape validation on the five most obviously-public read handlers across
all five contexts (malformed IDs return the same not-found answer as an
unknown-but-well-formed one — no enumeration oracle). A second, independent
instance of the identical bug was found and fixed in
`booking.ListCourtBookings` while investigating. Deliberately NOT covered
by the boundary layer (still relies on the interceptor alone, disclosed
explicitly, not silently assumed fixed): every write handler taking a
caller-supplied ID (`CancelCompetition`, `EnterCompetition`, `AddCourt`,
`RecordOfflinePayment`, `CreateOnlinePayment`, `ConfirmOnlinePayment`) —
lower severity since they're intended to require real auth once it lands,
but not yet validated. Follow-up recommended, not yet ticketed: extend the
same boundary guard to those write paths (still open, still not ticketed).
The `docs/LESSONS.md` entry PR #89 flagged as owed **has since been
written** — see `## T9 (2026-08-05) — grpc installs no panic recovery;
net/http intuition does not transfer`, since corrected there to record that
PR #89's own Layer 2 pass had also missed `booking.GetQuote` (a second
public unauthenticated read reaching the same panic, closed by PR #94)
and had shipped a vacuous regression test for `ListCourtBookings` (also
fixed by #94). SCRUM-6 (PR #95) has since landed a real CI/CD pipeline
definition (repo-side only — no Jenkins job/webhook/branch-protection
configured yet, see the Cross-cutting CI entry below), which is the
structural direction the T9 retro's CI-gate candidate finding points at.

Two merge conflicts were resolved by hand this sprint (T9.8↔T9.9 on
`HANDOFF.md`'s Docs-index T9 row, resolved into one row citing both ADRs;
T9.6↔T9.7 — the largest conflict of the five T5–T9 sprints — on four files
both tickets independently created: `web/src/api/competitionsClient.ts`,
`web/src/models/competition.ts` + its test file, and `web/src/router/index.ts`.
Reconciled by hand rather than picking a side: `CompetitionSummary` gained
T9.7's nullable `spotsLeft` field, the one genuine signature collision
(`formatSessionRange`, two-string-args on T9.6's side vs. one-session-object
on T9.7's) resolved by keeping T9.6's name/signature for its two call sites
and renaming T9.7's variant to `formatSessionRangeFromSession`, and
`mapToCompetitionSummary` kept as an alias of T9.7's `mapToCompetition`,
and T9.7's `ConfirmedEntry` kept as an alias of T9.6's
`CompetitionEntrySummary` (the interface's declared name), so neither PR's
naming had to change) — every conflict was resolved on the source branch,
never a direct
push to the shared branch, and re-verified (`go build`/`go vet`/`go test
-race`/`make test-domain` plus `npm run build`/`npm run test`, 383/383 web
tests passing post-merge) before merging.

Known gaps carried forward, not silently dropped (see Cross-cutting
below): Competitions has no online-payment path (`payments.PayableType`
has no Competition-entry value — verified by two independent
implementations, T9.6 and T9.7, that separately reached the same
conclusion and traced the same would-be data-corruption bug; cash-only for
now), no share-token revocation/rotation (explicitly blocked on real auth,
not an oversight), and the write-handler malformed-ID validation gap noted
above.

**Not yet built**
- Statements context.
- Auth, real migration tooling, observability.
- CI: the pipeline definition now exists (SCRUM-6 — `Jenkinsfile`,
  `make ci`, `docs/adr/0011-*`), but no Jenkins job/webhook/branch
  protection has been configured, so nothing runs automatically yet. See
  the Cross-cutting entry below for exactly what remains.
- Competitions↔Payments online-payment wiring (needs a new `PayableType`
  value + port/adapter — see T9's Cross-cutting entry).
- `internal/gen/**` still needs `make generate` run locally/in CI before
  `go build ./...` (not just the domain/app packages) will succeed — the
  postgres/grpcapi adapters and `cmd/server` are unverified beyond
  `gofmt`/manual reading in this environment (no `buf`/`sqlc` toolchain
  available here). Run `make generate && go build ./...` as the first
  real verification step next session, before assuming the full binary
  compiles. (Note: T7's implementer agents did have buf/sqlc/node/npm
  available in their sandboxes and ran real builds — see PRs #41–#46 for
  what was actually verified there; the caveat above is about *this*
  environment, not a claim the toolchain is universally unavailable.)

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
(inherits the no-overlap invariant). Matchmaking was deferred past T5; it
now has a named owning context and a binding trigger rather than an
open-ended deferral — **ADR-0010** schedules it with Identity/Users (see
Cross-cutting below).

**T6 — Payments context (+ Game waitlist, T6.6, technically a Social Play
ticket scheduled in this sprint). T6.1–T6.5 implemented and review-approved
(PRs #23, #24, #26, #27, #28). T6.6 implemented but NOT YET REVIEWED (PR
#25) — dispatch a PM+PE review before treating it as done. T6.7 (QA auth
tests) not yet implemented — check issue #22. NOT MERGED — see the note
above.** Full ticket breakdown, kickoff note, and PM/PE disagreements:
`docs/process/t6-sprint-plan.md`. Original scope (for reference):
`Payment` aggregate with a `PaymentStatus` state machine
(unpaid→paid→refunded); online path behind a Stripe **anti-corruption
layer** (interface + stub adapter first, real Stripe later); offline
path where Host/Game Admin records the amount. See "Cross-cutting /
later" below for follow-ups T6's own reviews surfaced (uncommitted
concurrency proof, missing `RefundPayment` wiring, migration-number
collision).

**T7 — Web client foundation + Facilities context. All 7 tickets (T7.1–T7.7)
implemented, reviewed, and MERGED (see the note above).** Full roadmap
(T7–T9), kickoff note, and all 7 tickets: `docs/process/t7-sprint-plan.md`.
Sprint goal met: a Facility and its Courts are real persisted entities, an
Owner/Host can onboard a facility with a consent-gated camera link through
a real Vue app, and a Player can browse, quote, and book a real court
end to end. Retro not yet written.

**T8 — Social Play + Payments UI spine, closing T7's carried gaps. All 10
tickets (T8.1–T8.10) implemented, reviewed, and MERGED (see the note
above).** Full re-scope reasoning, kickoff note, and all 10 tickets:
`docs/process/t8-sprint-plan.md`. Sprint goal met: a Host can publish a
Social Game at a real, linked Facility with a real payment method and
guest allowance, a Player can discover it, join it with guests, and pay —
online or cash — all through the real Vue app navigating between actual
routed screens, against real gRPC-gateway REST APIs with no fabricated or
dead-end fields anywhere in the flow (the one disclosed exception being
T8.10's placeholder registration fee, since Social Play has no price field
— see Cross-cutting below). Retro not yet written. Competitions,
social-account-linking, shareable-registration-links, and the WhatsApp/Zalo
spike — originally roadmapped as part of T8 in `docs/process/
t7-sprint-plan.md` — move to a new T9 per T8's own re-scope decision; the
former T9 (pricing/discount UI, Club rentals, WCAG hardening) becomes T10.

**T9 — Competitions bounded context + growth/social decisions. All 10
tickets (T9.1–T9.10) implemented, reviewed, and MERGED, plus one critical
out-of-band fix (see the note above).** Full ticket breakdown, kickoff
note, and the auto-matching/OAuth-custody reasoning: `docs/process/
t9-sprint-plan.md`. Sprint goal met: a Host can create a multi-session
Competition at a real Facility, publish an honest share link, and manage
its roster; a Player can discover a Competition or follow a shared link
and enter it with guests, under a DB-authoritative weighted capacity
guard — reusing Booking's existing `competition`-source invariant with
zero changes to `internal/booking/`. Two real decisions replace what would
otherwise have been further deferrals (ADR-0009, ADR-0010 — see the note
above); social-account-linking's OAuth half and the WhatsApp/Zalo
messaging bot stay deferred per ADR-0009, in-app RSVP via the shareable
link being the shipped mechanism in the meantime. Retro:
`docs/process/t9-retro.md` — 8 findings, 3 left as recorded unresolved
disagreements; its adopted changes bind T10's Ceremony 1 and 2 (dispatch
isolation as a planning checklist item, a cross-context dependency check
when a ticket calls into another context, gRPC-code-only error specs in
ticket text, and three untracked T9 follow-ups to open as real issues).
Competitions/social-account-linking's remaining half (real inbound
messaging), plus the former T9 roadmap items now renumbered T10 (pricing/
discount UI, Club rentals, WCAG hardening) — both gated on real auth per
ADR-0009/ADR-0010's triggers — are next.

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
  until the JWT/Auth0 item above lands. Don't mistake it for one. T6.7/T6.3's
  Payments equivalent (`actor_user_id` vs. Booking/Game-Host/Game-Admin
  ownership facts) and T7.7's Facilities equivalent (`actor_user_id` vs.
  `Facility.OwnerID`, closing issue #39 — see the dedicated T7.7 bullet
  below) carry the exact same caveat: object-level check given a claimed
  actor, not authentication. This is now a three-times-repeated pattern
  (Social Play, Payments, Facilities) and the caveat is the same each time
  — don't re-litigate it per context, just extend real auth to all three
  call sites together when the JWT/Auth0 item above finally lands.
  T8.5 (closes issue #53, `internal/payments/adapter/grpcapi/
  authz_regression_test.go`) finally closed the long-open T6.7 gap: the
  scope-check found `app.authorizeOfflineRecording`/
  `domain.ErrNotPaymentRecorder` (T6.3) already existed and was already
  unit-tested at the app layer, so T8.5 only needed to add the missing
  handler-level regression proof (mirroring T5.5/T7.7) — real
  `grpcapi.Handler.RecordOfflinePayment` -> `app.Service` ->
  `authorizeOfflineRecording`, on both the Registration-payable (Player
  who is neither Host nor an assigned Game Admin) and Booking-payable
  (non-Host actor) paths, asserting the mapped gRPC status is
  `PermissionDenied`, not `Internal`, and that no Payment is persisted.
  Verified non-vacuous per CLAUDE.md rule 10 by temporarily disabling
  `authorizeOfflineRecording`'s call site, confirming both regression
  tests fail, then restoring it. Same caveat as above, not re-litigated:
  object-level check given a claimed `actor_user_id`, not real
  authentication.
- **T10.2 (`internal/identity`) is a second place the caveat above is NOT
  "the same caveat again," and a materially worse one — flagged by PR #106
  review, not discovered later.** Every other `actor_user_id`/
  `actor_player_id` check in this codebase (T5.5 Social Play, T6.3/T6.7→T8.5
  Payments, T7.7 Facilities, above) only gates a *mutation on an object that
  already exists*: a false claim is rejected and leaves no trace. Identity/
  Users' `CreateUser` is structurally different — the caller-supplied
  `actor_user_id` becomes the row's own **permanent primary key**
  (`identity_users.id`; see `internal/identity/app.CreateUserInput`'s doc
  comment for the full reasoning this bullet does not restate). An anonymous
  caller can call `CreateUser` with any UUID they choose — including one a
  future real-auth integration will eventually mint deterministically for a
  real person — and permanently occupy that identity; the real owner's later
  registration attempt then fails with `domain.ErrUserAlreadyExists` and can
  never claim their own account. That is a **persistent, targeted
  denial-of-service**, not a rejected mutation, and has no equivalent
  anywhere else in this codebase: nothing else lets an unauthenticated
  caller-supplied claim become a permanent artifact another real identity
  will later collide with. Not mitigated by real auth in this ticket (out of
  scope per ADR-0012) — only *narrowed*, not closed, by T10.2's other fix
  (public `CreateUser` accepts only `RolePlayer`, so a squatted ID can't also
  carry an elevated role). **Must close the moment real auth exists**: at
  that point `CreateUser` should mint `User.ID` from the authenticated
  principal's own verified subject claim (e.g. a JWT `sub`), never accept it
  as a bare, unverified client-supplied field the way it does today. Track
  this alongside the JWT/Auth0 item above; do not let it get silently folded
  into the generic "claimed actor" caveat again — it is a different and
  worse failure mode (permanent artifact vs. a rejected mutation).
- **The one place the caveat above is NOT "the same caveat again":
  third-party OAuth tokens — see `docs/adr/0009-social-channel-integration-deferred.md`.**
  T9's ceremony (§A1 of `docs/process/t9-sprint-plan.md`) found that
  storing a social-platform OAuth access/refresh token keyed to a claimed,
  unverified `actor_user_id` differs *in kind*, not degree, from the three
  instances above (T5.5 Social Play, T6.3/T6.7→T8.5 Payments, T7.7
  Facilities): each of those bounds the blast radius to this platform's own
  data, whereas a token guards **a third party's account** outside this
  system. ADR-0009 therefore defers all OAuth token storage and all inbound
  messaging integration until real authentication exists — shareable
  registration links (T9.5) are the shipped mechanism for social-driven
  registration meanwhile, and the `port.MessagingChannel` anti-corruption
  layer is designed on paper in that ADR only (no package created). Its
  trigger condition is the sprint that lands real auth (recommended T10,
  §A5) and requires verified identity + an encrypted token-at-rest story +
  a revocation path to all exist first. The locked "channel you control,
  not public reply scraping" position
  (`docs/design/v1-system-design.md` §4,
  `docs/design/v1-external-reference-reconciliation.md`) is unchanged;
  ADR-0009 adds only the timing decision. Also still open and addressed to
  the user there: the Vietnam-vs-global market-scope question T7 escalated
  (it decides WhatsApp vs. Zalo, and only one platform gets prototyped).
- T5.5 (see PR stacked on #11-#14, closes issue #10) added a full-stack
  regression test — `internal/socialplay/adapter/grpcapi/authz_regression_test.go`
  — proving `Registration.Cancel`'s object-level ownership check (Player A
  cannot cancel Player B's registration) survives the real path
  (`grpcapi.Handler.CancelRegistration` -> `app.Service.CancelRegistration`
  -> `domain.Registration.Cancel`), not just the domain-level unit test
  T5.2 already had. It is a handler-level test against in-memory
  `port.GameRepository`/`port.RegistrationRepository` fakes rather than a
  `-tags=integration` Postgres round trip: the ownership check has no SQL
  involved, so a real DB adds infrastructure, not proof (ticket text
  explicitly allows this), and this environment had no Docker daemon
  available to actually execute a testcontainers-based version (only the
  `docker` CLI, `docker ps` fails to dial the socket — the same gap
  `internal/socialplay/adapter/postgres/concurrency_integration_test.go`'s
  package comment already documents for this context). Verified as a real
  regression test, not a decorative one, by temporarily commenting out the
  ownership check in `domain.Registration.Cancel` and confirming the new
  test fails, then restoring it and confirming green again (CLAUDE.md rule
  10). **Still reiterating the caveat above**: this proves the object-level
  check given a claimed `actor_player_id`; it does not and cannot prove
  that identity itself without real auth.
  **Split to a follow-up, not silently skipped**: the ticket also asked for
  the equivalent on `CreateGame`/`Game.Cancel()` (only a Game's `HostID`
  may cancel it, T5.1). This did not fit T5.5's scope because it doesn't
  exist yet to test — there is no `CancelGame` RPC in
  `proto/pickleball/socialplay/v1/socialplay.proto`, no
  `app.Service.CancelGame` method, and `domain.Game.Cancel()`
  (`internal/socialplay/domain/game.go`) takes no actor parameter at all
  (unlike `Registration.Cancel`, it isn't even ownership-checked at the
  domain level yet). Building one is proto + app + handler + regression-test
  work, not an extension of an existing pattern — a new ticket (proposed:
  "Add `CancelGame` with HostID-scoped authorization + regression test",
  same shape as T5.1/T5.4/T5.5 combined) should cover it; raise at the next
  backlog refinement.
- T6.5 (branch `sprint/t6.5-registration-payment-reconciliation`, closes
  #16-#20, depends on #11-#15 merging) did the two-way merge of
  `sprint/t6.4-postgres-proto-grpc` and `sprint/t5.5-authz-regression-tests`
  T6.4's own PR description predicted ("T6.5 is the ticket that first
  merges Social Play (T5) into the same branch as Payments (T6)"), added
  `RegistrationUpdater` to the *existing* `payments/app.ServiceOptions`
  (not a second constructor), and wired
  `ConfirmOnlinePayment`/`RecordOfflinePayment` to push a registration's
  `PaymentStatus` to `paid` via the new
  `internal/socialplay/port.RegistrationPaymentUpdater` port ->
  `internal/payments/adapter/socialplay` adapter (mirror image of
  `internal/socialplay/adapter/booking`, dependency arrow pointed the
  direction the context map requires). `no_show_fee`-payable Payments
  deliberately do NOT trigger this update (only `registration`, per the
  ticket's literal wording) — a no-show fee is a separate charge, not the
  seat's own payment status.
  **Refund modelling decision:** extended `socialplay/domain.PaymentStatus`
  with a third value, `refunded`, mirroring `payments/domain.Status`'s own
  `unpaid -> paid -> refunded` machine exactly, rather than collapsing a
  refund back to `unpaid` — Registration.PaymentStatus is meant to be a
  faithful projection of the real Payment, and "never paid" vs. "paid, then
  refunded" are different facts a Game Admin needs to tell apart.
  **Known gap, not built here**: `payments/app.Service` has no
  `RefundPayment` method at all yet — `domain.Payment.Refund()` (T6.1) and
  `port.PaymentProcessor.RefundPayment` (T6.2) exist, but nothing in `app`
  or the proto/gRPC layer calls either one. There is therefore no real call
  site today to push `PaymentStatusRefunded` through the new port; Social
  Play's `refunded` value and `MarkPaymentStatus`/`UpdatePaymentStatus`
  already accept it and are ready for when that method is built, but wiring
  it now would mean inventing a new Payments feature outside T6.5's stated
  scope (mirrors T5.5's own CancelGame split-to-follow-up reasoning).
  Proposed follow-up ticket: "Wire `Service.RefundPayment` (online via
  `PaymentProcessor.RefundPayment`, offline as a Host/Game-Admin action) and
  push `PaymentStatusRefunded` through `RegistrationPaymentUpdater` on
  success" — raise at the next backlog refinement.
  **No other Social Play writer of `PaymentStatus`**: confirmed by
  inspection — `proto/pickleball/socialplay/v1/socialplay.proto` only has
  `CreateGame`/`RegisterForGame`/`CancelRegistration` RPCs (none accept a
  `PaymentStatus` field), and the only pre-T6.5 writer was
  `domain.Register`'s hardcoded `unpaid` default at construction. T6.5 adds
  exactly one more writer (`Registration.MarkPaymentStatus` ->
  `Service.MarkRegistrationPaymentStatus`, called only by
  `internal/payments/adapter/socialplay`). Caveat: `PaymentStatus` remains
  an exported Go struct field, so nothing at the language level stops
  future Social Play code from assigning it directly in-process — full
  encapsulation would mean unexporting the field and adding
  constructor/getter methods repo-wide, a larger refactor judged out of
  scope for a 3-point reconciliation ticket. Logged here so it isn't
  mistaken for an enforced invariant.
  **Merge**: `sprint/t6.4-postgres-proto-grpc` and
  `sprint/t5.5-authz-regression-tests` touch disjoint packages
  (`internal/payments/**` vs `internal/socialplay/**`); merge conflicts
  were confined to shared wiring/doc files (`cmd/server/main.go`,
  `sqlc.yaml`, `HANDOFF.md`) and were resolved by keeping both sides'
  additions. Noted, not fixed: both lineages independently numbered a
  migration `0005` (`0005_payments.sql`, `0005_socialplay.sql`) — harmless
  today (`docker-compose`'s initdb.d and this repo's own
  `applyMigrations` test helpers apply files in filename-sorted order,
  and "payments" sorts before "socialplay" alphabetically, and the two
  migrations touch disjoint tables with no ordering dependency between
  them), but a real migration tool (the `golang-migrate`/`goose` swap
  already tracked above) would need distinct sequence numbers — worth
  renumbering whenever that swap happens, not urgent before then.
  **Cross-context integration test**: this environment has no Docker
  daemon (same T4/T5.4/T6.4 gap), so the committed
  `-tags=integration` testcontainers-based test
  (`internal/payments/adapter/socialplay/cross_context_integration_test.go`)
  could not itself be executed here. The identical scenario (create a live
  Game + Registration via the real Social Play stack, record an offline
  Payment via the real Payments stack, `GetRegistrationByID` observes
  `paid`, plus a negative control proving a booking-payable Payment leaves
  an unrelated Registration untouched) was verified manually against a
  real local Postgres 16 instance (system package, already running in this
  environment; the missing `0005_socialplay.sql`/
  `0006_socialplay_capacity_guard.sql` migrations were applied to it) via a
  throwaway `cmd/t65verify` program — same T4 LESSONS.md fallback
  methodology T6.4 used, run twice for consistency, output confirmed with a
  direct `psql` read against `registrations.payment_status` too, then
  deleted before committing (not part of this PR). See the T6.5 PR
  description for the exact commands/output.
- T6.6 (closes issue #21, fulfils ADR-0006) added the Game waitlist:
  `domain.WaitlistEntry`/`JoinWaitlist` (`internal/socialplay/domain/
  waitlist.go`), app-layer auto-promotion on `CancelRegistration` +
  `ExpireWaitlistPromotion` (`internal/socialplay/app/service.go`), and the
  DB-level promotion-ordering guard (`db/migrations/
  0007_socialplay_waitlist.sql`, `0008_socialplay_waitlist_promotion.sql`,
  `promote_next_waiting`). See the PR description for the full race
  analysis (ordering-shaped, not distinctness-shaped) and repeated-run
  concurrency evidence (CLAUDE.md rule 10). `socialplayapp.NewService` is
  now at 4 positional args (`ids, games, registrations, waitlist`) — the
  exact threshold the T1/T5 cross-cutting note above already flagged as
  "worth revisiting... if a 4th dependency lands"; left positional in T6.6
  (out of this ticket's scope to refactor), but the next dependency added
  to Social Play's `Service` should switch it to an options struct instead
  of growing further, same reasoning T6's own sprint plan applied to
  Payments' `Service` from the start. Standalone (non-Game) court/slot
  waitlists remain deferred with no ticket — see ADR-0006's Status section.
  **T6.6-loop-2 correction (PR #25):** the PM+PE review found the T6.6
  sprint plan's DB-level race-analysis requirement had only actually been
  closed for the promotion trigger — `JoinWaitlist`'s `Position` field was
  still computed from an unlocked app-layer read (`len(existing
  non-cancelled entries) + 1` in `domain.JoinWaitlist`), with a plain
  unconditional insert (`CreateWaitlistEntry`) and no DB-level guard at all.
  Reproduced directly (30 concurrent `JoinWaitlist` calls for the same full
  Game produced 27 entries all at `Position` 1). Same conclusion as the
  promotion trigger's own analysis: ordering-shaped, not
  distinctness-shaped — a bare `UNIQUE(game_id, position)` would only reject
  one of two equally-legitimate concurrent joiners rather than compute the
  correct next value for them. Fixed with `join_waitlist_entry`
  (`db/migrations/0009_socialplay_waitlist_join_position.sql`), a
  `FOR UPDATE`-locked Postgres function mirroring `promote_next_waiting`'s
  pattern exactly (locks the owning `games` row, counts non-cancelled
  entries, inserts atomically); the Postgres adapter's `Create` now
  discards the caller-computed `Position` and returns whatever the DB
  authoritatively assigns. Verified against a real local Postgres 16 (no
  Docker here either): 30 concurrent `JoinWaitlist` calls against one Game
  produced positions `1..30` exactly, no collisions, no gaps, across 6 runs
  including a true process cold start (cluster stop/start) — see the PR
  description for the full run log. A related but separate, non-concurrency
  bug was found and deliberately NOT fixed in this loop (flagged for a
  follow-up ticket instead): the count-based `Position` formula can still
  collide with an already-active entry's `Position` if a lower-`Position`
  entry is cancelled before a later join recomputes the count — this is a
  product-semantics question (whether `Position` should stay
  count-based/history-reflecting or switch to a monotonic
  `MAX(position)+1`), not a concurrency one, and changing it would silently
  redefine behavior `TestJoinWaitlist_PositionCountsNonCancelledEntries`
  currently locks in as intentional.
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
  invent a second constructor path. (T6.5's own entry above covers both
  the `RefundPayment` gap this omission led to and the migration-`0005`
  collision in more detail — not repeated here.)
- T7.7 (closes issue #39, third instance of the T5.5/T6.7 object-level
  authorization pattern, this time for Facilities) checked the actual
  shipped T7.3 Facilities handler first, per the ticket's own scope-check
  instruction: `AddCourt` and `AddCameraLink` had **no** ownership check at
  all — `AddCourt` didn't even fetch the target `Facility` before
  constructing/persisting a `Court`, and `AddCameraLink`'s only existing
  gate was `CameraConsentAttested` (T7.2), unrelated to *who* was calling.
  So, unlike T5.5/T6.7 (which only needed a regression test proving an
  existing check), T7.7 had to add the check itself:
  `domain.Facility.EnsureOwner(actorUserID)` (new,
  `internal/facilities/domain/facility.go`) compares a caller-supplied
  `actor_user_id` against `Facility.OwnerID`, returning the new
  `domain.ErrNotFacilityOwner` sentinel on a mismatch (mirrors
  `socialplay.ErrNotRegistrationOwner`/`payments.ErrNotPaymentRecorder`).
  `AddCameraLink` now calls it first, before its existing consent check,
  so a non-owner never gets to observe the Facility's consent state.
  `AddCourt` (`internal/facilities/app/service.go`) now fetches the
  Facility via `Repository.GetFacilityByID` and calls `EnsureOwner` before
  ever calling `domain.NewCourt`/`Repository.AddCourt` — no code path to
  the repository exists for a rejected actor. Both `AddCourtRequest` and
  `AddCameraLinkRequest` (`proto/pickleball/facilities/v1/facilities.proto`)
  gained an `actor_user_id` field (regenerated via `make generate`);
  `grpcapi.toStatus` maps `ErrNotFacilityOwner` to `codes.PermissionDenied`
  (-> HTTP 403), never `codes.Internal`. `CreateFacility` was out of scope
  by inspection — there's no pre-existing Facility to be scoped against on
  create, the caller sets `owner_id` themselves; there is no
  `UpdateFacility` RPC in the shipped proto at all, so per the ticket's own
  instruction this is scoped to `AddCourt`/`AddCameraLink` only, not a
  hypothetical third RPC.
  **Test shape**: domain-level unit tests
  (`internal/facilities/domain/facility_test.go`:
  `TestFacility_EnsureOwner`, `TestFacility_AddCameraLink_RejectsNonOwner`),
  app-level tests (`internal/facilities/app/service_test.go`:
  `TestAddCourt_RejectsNonOwner`, `TestAddCameraLink_RejectsNonOwner`), and
  — the ticket's required proof — a full-stack handler-level regression
  test, `internal/facilities/adapter/grpcapi/authz_regression_test.go`
  (`TestAddCourt_RejectsMismatchedActor`,
  `TestAddCameraLink_RejectsMismatchedActor`, plus the symmetric
  `_AllowsOwningActor` positive-path cases for both), run through the real
  `grpcapi.Handler` -> `app.Service` -> `domain.Facility` path against an
  in-memory `port.Repository` fake, same reasoning T5.5 used for
  handler-level over a `-tags=integration` Postgres round trip (the check
  has no SQL involved, and this environment has no Docker daemon — same
  gap T5.5/T6.4/T6.5/T6.6's own entries above already document). Verified
  as a real regression test, not a decorative one, by temporarily
  disabling `domain.Facility.EnsureOwner` (short-circuited to always
  return `nil`) and re-running the full `internal/facilities/...` suite:
  both handler-level tests
  (`TestAddCourt_RejectsMismatchedActor`/`TestAddCameraLink_RejectsMismatchedActor`),
  both app-level tests, and both domain-level tests failed exactly as
  expected, then the check was restored and the full suite (`go build
  ./... && go vet ./internal/facilities/... && go test
  ./internal/facilities/... -race`) confirmed green again (CLAUDE.md rule
  10). **Reiterating the caveat** (see the T5.5 bullet above, now updated
  to cover all three contexts): this proves the *object-level* check given
  a claimed `actor_user_id`, not real authentication, and does not and
  cannot prove that identity itself until the JWT/Auth0 item above lands.
  No authentication work was done in this ticket.
  **Pre-existing, unrelated gap noted, not fixed**: `go vet ./...` at the
  repo root fails on `internal/payments/adapter/socialplay/
  registration_updater_test.go` (a stale call to `socialplayapp.NewService`
  with 3 args against its current 4-arg signature) — confirmed pre-existing
  on this branch before any T7.7 change (T7.7 never touches
  `internal/payments` or `internal/socialplay`); `go vet
  ./internal/facilities/...` and `go build ./...` are both clean. Worth a
  follow-up to fix that stale test call, out of scope here.
- Observability: Sentry + slog + uptime.
- **Vue typed REST client: DONE (T7.1)** — `web/src/api/` generates from the
  OpenAPI output via `npm run generate:client` (openapi-typescript +
  openapi-fetch + a `swagger2openapi` conversion step, since
  `protoc-gen-openapiv2` emits Swagger 2.0 and `openapi-typescript` 7.x
  requires 3.0). Still open: generate Swift + Kotlin gRPC clients (`buf
  generate --template buf.gen.mobile.yaml`) — no native mobile client exists,
  deliberately deferred past T9 per `docs/process/t7-sprint-plan.md`'s
  roadmap section on native mobile.
- ~~T7.3 shipped `facilities`/`facility_camera_links` tables + a nullable
  `courts.facility_id` FK, but `games.facility_id text NOT NULL`
  (`db/migrations/0005_socialplay.sql`, pre-T7) was deliberately left
  unreconciled — it's an opaque string column with no FK to anything,
  unconnected to the new `facilities.id uuid`. Flagged by both T7's own
  sprint plan and its PM+PE review as a real T8 input, not an oversight:
  reconciling it (likely: add a nullable `games.venue_facility_id uuid`
  FK, or migrate the existing text column) needs a decision on whether a
  Game's free-text facility description and a real onboarded Facility are
  the same concept or two that need a migration path between them — raise
  at the next backlog refinement, don't decide unilaterally.~~ Closed by
  T8.3 (issue #51): `db/migrations/0011_socialplay_facility_fk.sql` adds
  the nullable `games.venue_facility_id uuid REFERENCES facilities (id)`
  FK exactly as sketched above, mirroring T7.3's `courts.facility_id`
  precedent — the old `facility_id text` column stays in place,
  unreferenced by any new code, marked deprecated in the migration file
  and in `domain.Game`'s doc comment. `domain.Game` gained
  `VenueFacilityID string`; `app.Service.ScheduleGame` validates it
  against a new `port.FacilityLookup` port (mirrors `port.CourtReservation`
  T5.3's shape) implemented by `internal/socialplay/adapter/facilities`
  against the real `facilitiesapp.Service.GetFacility` — an unknown
  `VenueFacilityID` returns `domain.ErrFacilityNotFound`, mapped to a 404,
  *before* any court is reserved (proven by
  `TestScheduleGame_UnknownVenueFacilityRejectedBeforeReservingCourts`, no
  partial-state Game/Booking left behind). `CreateGameRequest`/`Game`
  proto messages gained `venue_facility_id`; `facility_id` stays on the
  wire, marked deprecated. This repo's dev environment has no Docker
  daemon (same known gap `concurrency_integration_test.go`'s package
  comment already documents), so the schema change was verified by
  applying every migration 0001-0011 in order against a local Postgres
  instead of `make down && make up`: the new FK enforces correctly (an
  unknown facility ID is rejected at the DB level too), the old
  `facility_id` column keeps working unmodified, and — since `games` has
  no seed data — there were no pre-existing rows to check get `NULL`
  either way.
- ~~T7.3 shipped `AddCourt` as create-only — there is no RPC that lists a
  Facility's Courts back...~~ Closed by T8.2 (issue #50): `GetFacilityResponse`
  gained a `courts` field (chosen over a dedicated `ListFacilityCourts` RPC
  or putting it on the shared `Facility` message, so `ListFacilities`/
  `CreateFacility`/`AddCameraLink` don't pay for an extra courts query per
  facility they don't need), wired through a new sqlc query
  (`ListCourtsForFacility`) and the Postgres/gRPC adapters. T7.5's
  `FacilityDetailPanel.vue` and T7.6's booking flow both now render real
  courts with no further frontend change, exactly as T7.5's original gap
  note predicted.
- ~~T7.2/T7.3 deliberately shipped no API path that ever sets
  `Facility.CameraConsentAttested` to `true` server-side...~~ Closed by
  T8.4 (issue #52): a dedicated `AttestCameraConsent` RPC (deliberately not
  folded into `AddCameraLinkRequest`, per the round-10 review's "two
  explicit steps" requirement) — `domain.Facility.AttestCameraConsent`
  checks `EnsureOwner` before touching consent state, idempotent on
  re-attestation. `FacilityOnboarding.vue`'s existing "Cameras" step now
  calls it before its already-built `AddCameraLink` call, making the
  default-unchecked consent checkbox actually work end-to-end for the
  first time.
- ~~T7.1 through T7.6 all needed `npm install --legacy-peer-deps`... Still
  open as of T8 — every T8 implementer sandbox hit the same
  `typescript@~6.0.0` vs. `openapi-typescript`'s `^5.x` peer range friction.
  Confirmed working across 16 tickets' worth of builds now (T7+T8), so
  still not a real incompatibility, but a third sprint carrying this
  unfixed is exactly the "flagged in prose, not tracked as a real ticket"
  pattern T5's retro finding 6 warned about — raise as a real, small T9/T10
  ticket rather than logging it a fourth time.~~ Closed by T9.10 (issue
  #79): `web/package.json` now pins `typescript` to `~5.9.0` instead of the
  scaffold's `~6.0.0`. Both directions were checked before picking one —
  there is no newer `openapi-typescript` major that widens the peer range
  (7.13.0 is the latest published version and still declares
  `typescript: ^5.x`), so moving TypeScript was the only fix that didn't
  require changing the generator. `~5.9.0` satisfies every other TypeScript
  consumer at the same time (`@vue/tsconfig@0.9.1` needs `>= 5.8`,
  `vue-tsc@3.3.7` needs `>= 5.0.0`). `npm install` **and** `npm ci` now both
  succeed with no flag from a clean checkout (`node_modules` deleted), and
  `npm run build`, `npm run test` (27 files / 211 tests), and
  `npm run generate:client` all pass afterward. Critically, the generated
  client under `web/src/api/generated/` is **byte-identical** under
  TypeScript 6.0.3 and 5.9.3 — verified by regenerating under each and
  diffing, so this closed the chore without changing any API surface. The
  only other lockfile movement is `yaml`'s hoisting (top-level `1.10.3` ->
  `2.9.0`, with `1.10.3` now nested under the four `oas-*`/`swagger2openapi`
  consumers that actually pin it): that is npm resolving optional peer deps
  properly once `--legacy-peer-deps` is no longer suppressing them, not a
  bundled upgrade — re-running the install *with* the flag still on produces
  a typescript-only diff, which is how it was confirmed. Three sprints old
  at closure; see `web/README.md`'s "TypeScript version pin" section for
  when to revisit the pin.
- No CI is configured on this repo yet (still 0 GitHub check runs as of
  T9's reviews) — every "tests pass"/"build green" claim in T5–T9's PRs
  was verified by an agent running the commands locally in its own sandbox
  and reporting the result, not by an independently-checkable CI signal.
  Same status as T7/T8: flagged consistently as "plausible but
  unverifiable," not blocking, but now a three-sprint-old gap (Jenkinsfile
  already exists from T0 — wiring it for real is a real T10 candidate, not
  a hypothetical one). Note this would not by itself have caught the
  unticketed panic bug below (PR #89): its own root-cause writeup found
  the *existing* test fixtures minted non-UUID IDs like `"id-1"`, so no
  test — CI-run or not — could have seen the crash until the fixtures
  themselves were fixed to mint real UUID-shaped IDs. CI is still worth
  doing on its own merits; it just isn't a substitute for that class of
  fixture-fidelity gap.
  **Partially closed by SCRUM-6** (branch `SCRUM-6-cicd-pipeline`): the
  T0 `Jenkinsfile` placeholder is replaced with a real pipeline (checkout,
  toolchain, generate, lint Go+web, unit tests, build, integration,
  security scan, opt-in k6 load test), plus `make ci` for local parity, a
  tested security gate (`cmd/vulngate` + `tools/vulngate`), a `web`
  service in `docker-compose.yml`, and `docs/adr/0011-*`. Read the
  remaining gap precisely, because it is the part that matters: **the
  repo-side work is done; the SERVER-side work is not, and cannot be done
  from this repo.** Nobody has created the Jenkins job, installed the
  plugins, registered the GitHub webhook, or added branch protection, and
  there is no reachable Jenkins instance to do it against — so as of this
  entry there are still **zero automated pipeline runs**, and no PR has
  yet been gated by one. The `Jenkinsfile` header lists the five
  server-side steps required. Until an admin performs them, treat
  "CI exists" as "the pipeline definition exists and its commands were
  verified by hand", not as "changes are automatically verified".
  Worth noting SCRUM-6 immediately paid for itself even so: running the
  checks it wires up surfaced four pre-existing defects on the shared
  branch — three `vue-tsc` errors from T9.2's `GameSummary` entry-fee
  fields never reaching T8.9's fixtures, and one `staticcheck` finding —
  none of which `make test-domain` or `npm run test` would ever have
  caught, because neither type-checks. **Corrected by PR #95's own
  review**, which reproduced the `vue-tsc` defect directly rather than
  taking the fix's severity on faith: all three errors are confined to
  `__tests__/*.spec.ts` fixtures, and production's real
  `mapToGameSummary` always coerces the entry-fee fields
  (`Number(game.entryFee?.amountCents ?? 0)`), so the `$NaN` this defect
  produced reached the vitest-rendered DOM in a test, never a real
  browser. A genuine test-fidelity defect (and a legitimate reason to add
  type-checking to the pipeline that plain unit tests would have missed
  indefinitely) — not the shipped user-facing bug an earlier draft of
  this entry implied.
  **Also found by that same review, and fixed the same day**: the
  Security scan stage's govulncheck-unreachable path had a real gate
  bug, not just a disclosed limitation — when `vuln.go.dev` can't be
  reached, `govulncheck` still writes a well-formed `config`/progress
  preamble to stdout before exiting non-zero, and the stage's original
  `[ -s build/govulncheck.json ]` check only asked "is this file
  non-empty", not "did the scan actually finish" — so a failed scan
  still fed its (findings-empty) partial output to `vulngate`, which
  printed **`PASS: no new gating findings`** for a Go dependency scan
  that never ran. Reproduced end-to-end with a stub `govulncheck` that
  mimics exactly this failure shape, confirming the false PASS under the
  old logic and its absence under the fix. Fixed by capturing the real
  exit code to `build/govulncheck.exit` and gating on that instead of
  file size — `warnError`'s UNSTABLE marking was never the problem (it
  fired correctly); the gate's own verdict was the thing silently wrong,
  which is precisely the property ADR-0011 and the baseline file both
  claim is impossible by construction. `tools/vulngate/gate_test.go`
  gained `TestParseGovulncheckTruncatedRunLooksLikeZeroFindings`,
  documenting why this is a Jenkinsfile-level fix rather than something
  `ParseGovulncheck` itself should try to detect (the byte stream alone
  cannot distinguish a truncated run from a real zero-findings one; only
  the process's exit code carries that information).
- **New this sprint (critical, out-of-band, not a ticket): `cmd/server`
  had zero gRPC interceptors, so any unauthenticated request with a
  malformed ID crashed the entire server process** — found as a byproduct
  of PR #88's review, fixed same-day in PR #89. Two-layer fix: a global
  panic-recovery interceptor (protects all five contexts, present and
  future) plus boundary-level UUID shape validation on the five most
  obviously-public read handlers. Independently reproduced live on a real
  server against real Postgres, both the original crash (6 vectors,
  including a second independent instance in `booking.ListCourtBookings`
  the review found while investigating) and the fix surviving all of them
  with a normal request succeeding immediately after each attack. **Not
  yet closed**: every write handler taking a caller-supplied ID
  (`CancelCompetition`, `EnterCompetition`, `AddCourt`,
  `RecordOfflinePayment`, `CreateOnlinePayment`, `ConfirmOnlinePayment`)
  relies on the interceptor alone (a panic there is now a clean 500, not a
  crash, but still not the specific not-found/empty-list answer the read
  paths got) — lower severity since these are intended to require real
  auth once it lands, but explicitly disclosed rather than silently
  assumed fixed. The PR #89 review (not just the PR itself) adds a
  concrete reason not to leave this indefinitely: each unguarded panic
  still logs a full goroutine stack trace, so an unauthenticated caller
  can drive attacker-controlled log volume even though the process no
  longer crashes — not an outage, but real enough to move this from
  "someday" toward "worth scheduling." Recommend a small follow-up ticket
  extending the same boundary guard to the write paths, and a
  `docs/LESSONS.md` entry ("grpc installs no default panic recovery;
  `net/http`'s per-connection recovery intuition does not transfer") —
  both flagged in PR #89, neither written yet.
- **New this sprint**: T9.6 and T9.7 (issues #75, #76) independently
  reached the same finding while building the Competitions UI —
  `payments.PayableType` has exactly three values (`BOOKING`,
  `REGISTRATION`, `NO_SHOW_FEE`), none for a Competition entry, and
  `internal/payments/app.Service`'s `reconcileRegistrationPaymentStatus`
  writes any `PAYABLE_TYPE_REGISTRATION` confirmation into **Social
  Play's** registrations table specifically. Routing a Competition entry
  through T8.10's existing checkout as-is would have persisted a Payment
  as paid, then written a Competition entry ID into the wrong context's
  table at confirm time — a real money-adjacent correctness bug, not
  extra work skipped. Both PRs' reviews independently verified this by
  tracing the exact code path rather than trusting the claim. Competitions
  is cash-only in T9 as a result (online-payment checkout deliberately not
  built). Recommend a follow-up ticket: a Competitions-shaped
  `PayableType` value plus the equivalent port/adapter pair
  `internal/payments/adapter/socialplay` already established for Social
  Play (T6.5).
- **New this sprint**: T8.10 (closes issue #58) found that `domain.Game`/
  `domain.Registration` have no price/fee field at all — confirmed by
  inspection, unchanged by T8.6/T8.7's own field additions this same
  sprint (`PaymentMethod`, `GuestAllowance`, `GuestCount`, none of them a
  price). Since `CreateOnlinePaymentRequest`/`RecordOfflinePaymentRequest`
  both require a `Money` amount on the wire regardless, T8.10 used a flat
  `PLACEHOLDER_REGISTRATION_FEE_CENTS` ($10.00), visibly labeled
  "placeholder" everywhere it's shown in the UI rather than presented as a
  real charge — reviewed and judged an acceptable disclosed workaround
  (silently sending $0 would misleadingly imply a free registration; a
  real price field is its own domain+migration+proto cycle, out of scope
  for a Payments-*UI* ticket). Recommend a follow-up ticket (T9/T10
  backlog) to add a real per-Game price field, the same way T8.6/T8.7 did
  for `PaymentMethod` — and get explicit Product Owner sign-off on any
  placeholder/default figure before this UI is ever pointed at a real
  (non-stub) Stripe integration.
- **New this sprint**: T8.9 (closes issue #57) found `socialplay.proto` had
  no list/browse RPC for Games at all — added a minimal `ListGames` RPC
  (server-computed `spots_left`, matching `domain.Register`'s weighted
  capacity formula exactly, verified by a boundary test). T8.10 (closes
  issue #58) found the same absence for a Game's Registrations — added
  `ListRegistrationsForGame`. Both follow the migration-free-read-path
  pattern T8.2/T7.3 already established (no schema change, no new domain
  type, public read, no ownership check).
- **Auto-matching (level/gender matchmaking, `PlayerRating`, `Match`, the
  self-reported starting `Level`) is no longer "deferred with no home."**
  `docs/adr/0010-auto-matching-deferred-to-identity-context.md` (T9.9,
  transcribing `docs/process/t9-sprint-plan.md` §A2) decides it: it is built
  in the sprint that stands up the **Identity/Users** context — named as T10
  in §A5 — and **not before**. Because `Level` belongs to Identity/Users per
  `docs/agent-operating-handbook.md` A1 and that context does not exist
  (verified by grep this ceremony; the commands and their empty result are
  recorded in the ADR); because parking a cross-context field in the wrong
  context is a bill this project already paid once at real expense
  (`games.facility_id` → T8.3: a migration, a new port, a new adapter, and
  deprecated artifacts still sitting in the schema, the domain struct, and
  the proto today), so knowingly repeating it is a repeat, not a tradeoff;
  and because T9 builds no brackets (§A4), so the seeding call site that
  would have justified a minimal rating this sprint does not exist.
  This is a **sequencing** decision, **not** a reversal of CLAUDE.md's
  locked "matchmaking is in v1" decision — do not read it as one.
  **Trigger, binding on the next sprint:** T10's Ceremony 1 must either
  build auto-matching or supersede ADR-0010 with a new ADR; it may not
  defer it again in prose. **Two product/legal questions are escalated to
  the user in that ADR, not resolved by it:** the Player Level formula's
  weighting (the design handoff's own open question,
  `docs/design/handoff-2026-08/README.md`), and whether gender-mix matching
  is in scope at all, given it means collecting and algorithmically acting
  on a protected attribute — the same class as the two items already
  awaiting sign-off in `docs/design/v1-system-design.md`'s top blockquote.

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
