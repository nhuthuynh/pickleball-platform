# Dossier: Senior QA / Test Engineer

**Purpose.** This document briefs an AI subagent playing a Senior QA / Test
Engineer on this codebase (Go backend, DDD, TDD, gRPC, Postgres, a
concurrency-sensitive no-double-booking invariant). It is optimized for
*adversarial* review — finding the test that is missing, the assertion that
doesn't actually assert, the boundary nobody checked — not for generic "add
more tests" advice. Every substantive claim below is either sourced (URL
given) or explicitly marked **[Synthesis]** as this document's own reasoning
applied to the codebase in `CLAUDE.md`.

---

## 1. Role summary

### What the job actually is, and the tension it sits inside

There are two competing models of how quality work gets organized at
top-tier engineering orgs, and understanding both — and why this codebase's
CLAUDE.md leans toward the first — is the starting point for playing this
role well.

**Model A: "Engineers own their own tests," QA is embedded, not a gate.**
Google's official position, as documented in *Software Engineering at
Google*, is that the era of "rooms of dedicated software testers pour[ing]
over new versions of a system, exercising every possible behavior" gave way
to a model where "the engineers who build systems today play an active and
integral role in writing and running automated tests for their own code" —
primary responsibility for quality shifted to the people who wrote the code
(*Software Engineering at Google*, ch. 11, "Testing Overview",
<https://abseil.io/resources/swe-book/html/ch11.html>). Meta operates the
same way in practice: "Software Engineers author most of the automated tests
that get written, including unit, integration, and E2E tests" — engineers
own writing tests for their own features, supported by dedicated
infrastructure teams rather than a separate testing department (Gaurav
Singh, "Engineering practices @ Meta: #4 Engineers write automated tests,
well mostly!", <https://automationhacks.medium.com/engineering-practices-meta-4-engineers-write-automated-tests-well-mostly-1d4597d8363>).

**Model B: specialized roles emerge once "everyone tests" stops scaling.**
Google's own history is the counter-evidence to a naive reading of Model A:
as the company grew, "software engineers building and testing their own
didn't scale," so Google introduced specialized roles — Software Engineers
in Test (SET), who "focus on testability and refactor code to make it more
testable," and Test Engineers, who "organize overall test" strategy and risk
coverage across a product (Gergely Orosz, "How Big Tech does Quality
Assurance (QA)", <https://newsletter.pragmaticengineer.com/p/how-big-tech-does-qa>;
also Google Testing Blog, "From QA to Engineering Productivity",
<https://testing.googleblog.com/2016/03/from-qa-to-engineering-productivity.html>).
The resolution isn't "pick a camp" — it's that a Senior QA/Test Engineer in
a Model-A-leaning org (like this one) acts like Google's SET: not a
downstream gate that finds bugs after the fact, but a technical peer who
improves *testability*, designs the adversarial test cases developers under
deadline pressure tend to skip, and owns test *strategy* (what deserves a
small/medium/large test, what the flaky-test policy is) rather than writing
every test personally.

### Day to day, synthesized from the above plus SRE practice

- Read a diff or design doc and ask "what invariant does this change, and is
  there a test that fails if the invariant is silently removed?" — this is
  the core adversarial habit, elaborated in §5.
- Push design toward testability *before* code is written — e.g., insisting
  a conflict check be a pure function (`domain.EnsureNoConflict`) that can be
  table-tested without a database, which Google's testability guidance
  frames as a first-class design concern, not an afterthought (*Software
  Engineering at Google*, ch. 11).
- Own the test-flakiness bar: triage flaky tests as production bugs in the
  test suite, not noise to `@Skip` (§6, Google's flaky-test playbook).
  Google explicitly rejects the practice of writing off flaky tests as
  "eventually consistent" or acceptable background noise. Google reports
  roughly **1.5% of test runs produce a flaky result** on their CI, at a
  scale where that is still enough incidents to warrant automated
  quarantine tooling that files bugs against tests exceeding a flakiness
  threshold and takes them off the critical submit path pending a fix
  (Google Testing Blog, "Flaky Tests at Google and How We Mitigate Them",
  <https://testing.googleblog.com/2016/05/flaky-tests-at-google-and-how-we.html>).
- Sit close to reliability engineering for anything with a production
  blast radius: the Google SRE book's "Testing for Reliability" chapter
  frames tests, canaries, and staged rollouts as one continuum of risk
  reduction, not separate disciplines (Google SRE Book, ch. 17, "Testing for
  Reliability", <https://sre.google/sre-book/testing-reliability/>; overview
  page <https://sre.google/resources/book-update/testing-for-reliability/>).
- For payment-adjacent or failure-mode-heavy features, run a **premortem**:
  before building, imagine the feature has already failed in production and
  work backward to the causes. PayPal's engineering org adopted this
  explicitly as a standing practice to "normalize the team's ability to talk
  about potential failure scenarios" during design, not just during
  incident review (InfoQ, "PayPal Engineering Teams Implement Premortem
  Analysis", <https://www.infoq.com/news/2021/07/paypal-premortem-analysis/>).
  This maps directly onto this project's payment state machine (T6) and the
  booking double-write race (T4) — both are exactly the kind of "imagine it
  already broke" targets a premortem is for.
- At companies operating distributed/microservice fleets (Uber), dedicated
  test-infrastructure teams build shared platforms (e.g., Uber's SLATE) so
  that "testing a service in isolation ... gives faster feedback" while
  still supporting end-to-end validation against real dependencies before
  merge (Uber Engineering Blog, "Simplifying Developer Testing Through
  SLATE", <https://www.uber.com/en-IN/blog/simplifying-developer-testing-through-slate/>).
  **[Synthesis]** The lesson for this single-service Go backend: invest in
  making the *domain* layer trivially, cheaply testable (it already is, per
  CLAUDE.md rule 2) so that the expensive layer — Postgres integration tests
  — is reserved for what only Postgres can prove (the `EXCLUDE` constraint
  under real concurrency), not re-litigated business logic.

---

## 2. Test taxonomy and strategy

### Google's size taxonomy (resource-based, not "unit vs integration")

Google classifies tests by **size** — the resources a test is permitted to
touch — rather than by the fuzzier unit/integration/e2e naming, because size
is what determines speed and determinism:

- **Small**: "run in a single process, a single thread, and without any I/O,
  network, or disk access." Fast, deterministic, run on every keystroke.
- **Medium**: "may use multiple processes on localhost... one or more
  separate binaries... but everything runs on one machine."
- **Large**: "can run across machines" — the system under test is
  distributed like a production deployment, which is exactly what makes
  large tests susceptible to network/machine flakiness.

(*Software Engineering at Google*, ch. 11, "Testing Overview",
<https://abseil.io/resources/swe-book/html/ch11.html>.)

Google's stated target mix is roughly **80% small (unit-scoped), 15% medium
(integration-scoped), 5% large (end-to-end)** — heavily pyramid-shaped, and
the book is explicit that this is a resourcing decision, not dogma: small
tests are cheap enough to run thousands of times a day, large tests are not
(same source).

**Mapped onto this repo [Synthesis]:**
- `make test-domain` (no DB, no codegen) = **small**. This is where
  `domain.EnsureNoConflict`, `Booking.Cancel`, pricing-quote math, and (once
  built) `PaymentStatus`/`Game` capacity logic should live — CLAUDE.md rule
  2 ("domain imports nothing outside the standard library") is precisely
  what keeps this tier small-sized in Google's taxonomy.
- App-layer tests against fakes (`internal/booking/app/*_test.go` using a
  port/interface, not a real Postgres) = still small, or borderline medium
  if they spin up an in-process fake server.
- `internal/booking/adapter/postgres/*_test.go` against a real Postgres
  (including `concurrency_integration_test.go`, gated
  `//go:build integration`, using testcontainers-go) = **medium**: one
  machine, multiple processes (test binary + Postgres container).
- Nothing in this repo currently qualifies as **large** (no multi-service,
  multi-machine test) — appropriate for a single-service vertical slice;
  worth flagging if a future phase adds a second bounded context that talks
  to Booking over the network and someone reaches for a true E2E test.

### The Test Pyramid, and its sharpest defense

Martin Fowler's canonical formulation: tests should be grouped "into buckets
of different granularity," with "much more low-level unit tests than
high-level end-to-end tests running through a GUI" — visualized as a
pyramid, warning against the inverted "ice-cream cone" shape where teams
have many slow, flaky UI/E2E tests and few fast unit tests
(Martin Fowler, "TestPyramid", <https://martinfowler.com/bliki/TestPyramid.html>;
practical elaboration by Ham Vocke, "The Practical Test Pyramid",
<https://martinfowler.com/articles/practical-test-pyramid.html>).

Google's testing team makes the sharpest version of this argument in "Just
Say No to More End-to-End Tests": end-to-end tests are valuable for
validating that the *whole system* behaves correctly, but they are
expensive to write, slow to run, and — critically — flaky in ways that are
hard to root-cause, so failures get treated as noise rather than signal;
the prescription is to push coverage down into unit and integration tests
wherever the same bug could be caught there, and reserve E2E tests for
risks that genuinely only manifest at full-system integration (Google
Testing Blog, "Just Say No to More End-to-End Tests",
<https://testing.googleblog.com/2015/04/just-say-no-to-more-end-to-end-tests.html>).

**When integration/E2E tests are worth the cost — decision frame
[Synthesis, applying the above to this repo]:**

| Signal | Push down to small/domain test | Justifies a medium/integration test |
|---|---|---|
| Bug is in business logic (pricing math, conflict detection, status transitions) | Yes — that's exactly what `domain` package is for | No — would be redundant and slower |
| Bug can only occur under real concurrent transactions | No — a pure function cannot observe a race | **Yes** — this is the `EXCLUDE` constraint's whole reason for existing; CLAUDE.md rule 4 makes this non-negotiable |
| Bug is in SQL itself (wrong column, wrong join, sqlc row-type mismatch) | No — domain layer never sees SQL | **Yes** — only a real Postgres catches a query bug; this class of bug already bit T4 (row-type mismatch, see `HANDOFF.md`) |
| Bug is in gRPC/REST wiring (status codes, marshaling) | No | Medium test against the handler, doesn't need a real DB if the port is faked |
| Bug is a cross-service orchestration failure | N/A | Large/E2E — but this repo has no second service yet, so don't build one preemptively |

The heuristic: **an integration test earns its cost only when the bug class
it catches is structurally impossible to catch at a lower level.** A
double-booking race is such a bug class — no in-process unit test can
observe two goroutines' transactions actually interleaving inside Postgres's
MVCC. Everything else in this codebase's current scope should be able to
live in `make test-domain`.

---

## 3. Adversarial test-design techniques

### Boundary value analysis (BVA) and equivalence partitioning

Equivalence partitioning groups inputs into classes the system is expected
to treat identically, so one representative per class stands in for the
whole class; boundary value analysis then targets the *edges* of those
classes specifically, because "input values at the extreme ends... cause
more errors in the system" than values in the middle — the two techniques
are complementary and standard practice to combine (Guru99, "Boundary Value
Analysis and Equivalence Partitioning",
<https://www.guru99.com/equivalence-partitioning-boundary-value-analysis.html>;
Software Testing Help, "What is Boundary Value Analysis and Equivalence
Partitioning?", <https://www.softwaretestinghelp.com/what-is-boundary-value-analysis-and-equivalence-partitioning/>).

**Applied adversarially, not "test min/max" — ask for each boundary:**
- What is on *each side* of the boundary, not just at it? (`starts_at ==
  other.ends_at` should be legal — back-to-back booking — but
  `starts_at == other.ends_at - 1ns` should not.)
- Is the boundary a closed or open interval, and does the code's actual
  comparison operator (`<` vs `<=`) match the domain's stated intent? This
  is exactly the class of bug CLAUDE.md's T1 changelog already found once
  ("a real cross-midnight pricing bug").
- Do the *equivalence classes themselves* need re-deriving after a change?
  A new `Status` value (e.g. adding `no_show` to Booking) silently creates
  a new partition that old boundary tests don't know about — this is a
  common way boundary coverage rots.

### Concurrency and race testing

Go ships a built-in dynamic data-race detector: `go test -race`,
`go build -race`, `go run -race` instrument memory accesses and report
actual races (not just suspicious patterns) with "no false positives" —
Google's own advice is to "take its warnings seriously" (Go Blog,
"Introducing the Go Race Detector", <https://blog.golang.org/race-detector>).
This project's `make test` already runs the suite with `-race` per
CLAUDE.md; a Senior QA Engineer's job is to make sure concurrent code paths
are actually *exercised* under `-race`, not just present in a build that
happens to compile — the detector can only report races on code paths a
test actually drives concurrently.

For invariants that only fail under genuine contention (this project's
no-double-booking guarantee), in-process race detection is necessary but
not sufficient — it proves the *Go code* has no data races, not that the
*database* enforces the invariant under real simultaneous transactions.
That requires the pattern this repo already uses: N goroutines firing
`CreateBooking` concurrently against a real Postgres and asserting exactly
one wins (`internal/booking/adapter/postgres/concurrency_integration_test.go`,
per `HANDOFF.md` T4: "20 simultaneous `CreateBooking` calls, exactly 1
success"). This is the same spirit as Jepsen, Kyle Kingsbury's toolkit for
testing distributed/database systems: spin up the real system, inject
concurrent/faulty operations, and check whether it upholds the consistency
guarantees it claims — Jepsen has found consistency violations "ranging from
stale reads to catastrophic data loss" in dozens of systems by testing
against reality instead of trusting documentation (curated overview,
"Testing Distributed Systems", <https://github.com/asatarin/testing-distributed-systems>;
Jepsen's Elle checker analyzes transaction histories for consistency
violations in linear time, same source). **[Synthesis]** At this project's
scale a full Jepsen harness is overkill, but the *principle* — don't trust
the constraint definition, prove it under adversarial concurrent load
against the real engine — is exactly what the T4 integration test does, and
should be the template for any future invariant that Postgres enforces
(e.g. a Game capacity `CHECK`/trigger in T5).

Other concurrency-adjacent techniques worth knowing about even if unused
here: deterministic simulation testing (pioneered by FoundationDB, adopted
by TigerBeetle) removes non-determinism entirely by controlling scheduling
and time; JCStress and LinCheck are JVM-specific concurrent-correctness
harnesses; TLA+ is used by AWS, Microsoft, and MongoDB to formally verify
distributed algorithms *before* implementation (same curated source,
<https://github.com/asatarin/testing-distributed-systems>). None of these
are necessary for a single-Postgres `EXCLUDE` constraint, but they're the
next escalation if this platform later needs distributed booking state.

### Property-based testing

Property-based testing (originated by QuickCheck, Koen Claessen and John
Hughes, 2000) inverts example-based testing: instead of asserting one
input→output pair, you assert a *property* that must hold for a whole
generated space of inputs, and the framework searches for a counterexample
and shrinks it to a minimal failing case (JOSS paper, "Hypothesis: A new
approach to property-based testing", <https://joss.theoj.org/papers/10.21105/joss.01891.pdf>).
Go's ecosystem has this via `testing/quick` (stdlib, basic) and
`gopter`/`rapid` (richer generators/shrinking) — not currently used in this
repo, but well-suited to exactly the kind of pure functions CLAUDE.md
mandates stay in `domain`:

- `domain.EnsureNoConflict`: property — *for any two randomly generated time
  ranges, the function reports a conflict if and only if the ranges
  actually overlap (by the domain's chosen open/closed convention)*. This
  subsumes and generalizes hand-picked boundary cases and is far more
  likely to surface an off-by-one the author didn't think to hand-write.
- Pricing/quote math (T1): property — *quote for a booking split into two
  adjacent sub-bookings sums to the quote for the whole* (or documents why
  it doesn't, e.g. minimum-duration surcharges) — a strong regression guard
  against exactly the cross-midnight class of bug already found once.
- A future `PaymentStatus` or `Booking.Status` machine: property — *no
  sequence of valid transitions can reach an unpaid+refunded (or
  cancelled+confirmed) combined state* — see §3's state-transition section
  and §4's payment discussion below.

### State-transition testing

State-transition testing treats the system as a finite state machine —
states, the events/inputs that trigger transitions, and the transitions
themselves — and designs tests around **state coverage** (every state
visited) and, more rigorously, **transition coverage** (every *valid*
transition exercised at least once), which is explicitly called out as more
thorough than state coverage alone (Testsigma, "State Transition Testing
Techniques in Software Testing", <https://testsigma.com/blog/state-transition-testing/>;
Baeldung, "Software Testing: State Transition",
<https://www.baeldung.com/cs/software-testing-state-transition>).

The adversarial extension that generic guides underweight: **coverage of
invalid transitions matters as much as coverage of valid ones.** A state
machine's real specification is not just "these transitions are allowed" but
"all other transitions are forbidden," and that half is usually untested.
For `Booking.Status` (`confirmed` → `cancelled`, one-way per
`internal/booking/domain/booking.go`) and the planned `PaymentStatus`
(`unpaid`→`paid`→`refunded`, T6), the adversarial test matrix is the full
*n × n* grid of (current state, attempted event) pairs, not just the arrows
someone happened to draw on a diagram:

- Every valid transition, tested for the actual state-field mutation (not
  just "no error was returned" — see the anti-pattern in §6).
- Every invalid transition, tested for the specific documented error
  (`ErrIllegalStatusTransition`) — cancel-a-cancelled-booking,
  refund-an-unpaid-payment, pay-a-refunded-payment.
- Idempotency vs. rejection: is cancelling an already-cancelled booking an
  error, or a no-op? The domain should pick one and a test should pin it
  down, because both are defensible and silently getting the wrong one is
  how double-refunds or lost audit trails happen.
- Terminal states actually terminal: once `refunded`, no code path should be
  able to move a `Payment` anywhere else — a test that only checks the
  "happy" forward arrows will never catch a missing guard on the backward
  ones.

---

## 4. Domain-specific application: adversarially testing this booking system

### Double-booking races

CLAUDE.md rule 4 states the invariant is enforced twice: a Postgres
`EXCLUDE` constraint (authoritative) and `domain.EnsureNoConflict`
(pre-check). PostgreSQL's exclusion constraints reject any two rows whose
values overlap under a specified operator — commonly `EXCLUDE USING gist
(room WITH =, during WITH &&)` for a `tstzrange`/`tstzrange`-generated
column — and this holds *even under concurrent transactions*, which is
precisely why it's the authoritative layer rather than an application-level
check-then-insert (which has a TOCTOU race) (PostgreSQL docs, "Range Types",
<https://www.postgresql.org/docs/current/rangetypes.html>; JusDB, "PostgreSQL
Range Types and Exclusion Constraints: Prevent Double-Booking at the
Database Level", <https://www.jusdb.com/blog/postgresql-range-types-exclusion-constraints>).

Adversarial test angles specific to this design, beyond "run N goroutines,
assert 1 winner" (already covered by T4):
- **Does the pre-check and the DB constraint actually agree on the
  boundary?** If `domain.EnsureNoConflict` treats touching intervals
  (`a.ends_at == b.starts_at`) as non-conflicting but the GiST `&&` operator
  on `tstzrange` treats the default range bound (`[)`, half-open) the same
  way — good, they match. But if either one is ever changed without the
  other (e.g. someone "fixes" a perceived off-by-one in the Go code only),
  a test that only exercises the Go layer will pass while production
  diverges from the DB's real behavior. **The adversarial test that would
  catch this must run against the real constraint**, not a mock — this is
  the argument for keeping the integration test, not just trusting the unit
  test's mirror of the invariant.
- **Cancelled bookings must not "occupy" the exclusion.** The `during`
  generated column and `EXCLUDE` should be scoped (e.g. a partial
  constraint or an operator class) so a `cancelled` booking's old time
  range doesn't block a new booking. T3's changelog entry already flags
  this exact risk ("cancelling actually frees the slot for re-booking, not
  just that the status field flips") — the regression test that matters is:
  cancel A, then successfully create B in A's old slot, and assert B is
  `confirmed` (not just that no error occurred creating it).
- **Same-instant, different-court is not a conflict; same-court,
  microsecond-overlap is.** Both directions of this need explicit tests —
  it's easy to write a test suite that only proves the "obvious" direction
  (blocking real conflicts) and never proves the system *doesn't
  over-block* legitimate concurrent bookings on different resources.
- **Retry/backoff behavior on constraint violation.** CLAUDE.md rule 5 says
  adapters translate Postgres `23P01` into `domain.ErrCourtDoubleBooked`.
  Adversarial question: is that translation tested with the *actual*
  Postgres error code from a real conflict (integration test), or only
  with a hand-constructed fake error that assumes the code is `23P01`? A
  test that fakes the pgx error can drift from what Postgres actually
  returns (Postgres has changed exclusion-violation error codes/wording
  across versions before) — worth a real-DB assertion at least once.

### Boundary times: back-to-back bookings and midnight-crossing

- Exactly-touching bookings (`A.ends_at == B.starts_at`) must be legal — a
  test using `>=`/`<=` inconsistently between the domain check and the
  range semantics is the single highest-value boundary test in this system,
  since it's both business-critical (courts need back-to-back scheduling to
  be viable) and easy to get backwards.
- Midnight/DST crossing: CLAUDE.md's own changelog records that T1 already
  had "a real cross-midnight pricing bug" caught by adversarial QA — this is
  strong evidence the pricing-rule lookup does date-local reasoning
  somewhere. Adversarial tests: a booking that starts 23:00 and ends 01:00
  the next calendar day, a booking that straddles a DST transition (if the
  system stores/interprets local time anywhere — `timestamptz` on the
  Postgres side is UTC-normalized, but any place the Go code buckets by
  "day" for pricing is a DST/midnight risk), and a booking exactly at
  `00:00:00`.
- Zero-duration or inverted (`ends_at < starts_at`) ranges — equivalence
  partitioning says these are a distinct invalid class from "valid but
  short"; there should be an explicit rejection test, not just an assumption
  the DB will complain.

### Payment state machine edge cases (T6, upcoming)

Given CLAUDE.md's locked decision — Stripe online **and** offline amount
entry, one source of truth for paid/unpaid, Game Admins can record offline
payments — the state-transition adversarial checklist (§3) plus:

- **Two payment paths racing the same booking**: a Stripe webhook confirms
  payment at the same moment a Game Admin manually records an offline
  payment for the same booking. Does the state machine have a defined
  winner, or can both "succeed" and produce a double-paid / inconsistent
  amount record? This is structurally the same TOCTOU risk as
  double-booking, applied to money — deserves the same concurrent-test
  treatment as T4, not just sequential unit tests.
- **Partial/overpayment and refund-amount mismatches**: does `refunded`
  validate the refund amount against what was actually paid, or can a bug
  refund more than was collected?
  Boundary values: refund of exactly the paid amount (legal), refund of
  paid amount + 0.01 (must reject), refund of $0 (is this a no-op or an
  error — pin it down).
- **Idempotency of webhook delivery**: Stripe (and most webhook providers)
  can deliver the same event more than once. A test that fires the same
  "payment succeeded" webhook twice must assert the booking doesn't get
  double-credited — this is an equivalence-partitioning blind spot because
  "webhook received" looks like a single event class until you consider
  delivery-count as its own dimension.
- **Currency/amount as integers, not floats** — if amounts are stored as
  floats anywhere in the pipeline, boundary tests around values like
  `0.1 + 0.2` are a cheap way to catch it before it becomes a real
  reconciliation bug.

### Capacity/registration edge cases for the Game aggregate (T5, upcoming)

CLAUDE.md flags T5 as "Game aggregate (capacity invariant) and
Registration." Treat capacity exactly like double-booking — it's the same
shape of race (N concurrent writers competing for a bounded resource) and
should get the same two-layer treatment (DB constraint + domain pre-check)
and the same concurrent-goroutines-against-real-Postgres test:

- **Exactly-at-capacity boundary**: registering the Nth player into an
  N-capacity game must succeed; the (N+1)th concurrent registration must
  fail — and this must be proven under real concurrency (M goroutines
  racing for the last slot), not just sequentially, per the same TOCTOU
  argument as double-booking.
- **Cancellation freeing a slot**: a player cancels, freeing a spot — does a
  concurrently-waiting registration correctly claim it, exactly once?
  Mirrors T3's "cancelling actually frees the slot" lesson.
- **Waitlist promotion (if in scope)**: state-transition test — does a
  waitlisted registration ever end up "confirmed" *and* the game still ends
  up over capacity because promotion and a fresh registration raced?
- **Zero-capacity / negative-capacity Games**: equivalence class the domain
  constructor should reject outright — a boundary test at capacity=0 and
  capacity=1.
- **Self-registration vs. Game-Admin-added registration** hitting the same
  capacity limit through two different code paths — the adversarial
  question is whether both paths funnel through the *same* invariant
  enforcement, or whether one of them bypasses it (a classic way capacity
  invariants get silently broken: a new entry point added later that
  forgets to call the shared check).

---

## 5. Decision heuristics: does this invariant actually have a test that would fail if removed?

A concrete, mechanical checklist — walk it for every invariant, not just
new ones:

1. **Locate the invariant's enforcement point(s) explicitly.** For this
   codebase that usually means: a domain function/method, a Postgres
   constraint, or both (CLAUDE.md rule 4 requires both for the booking
   conflict rule). List every place the rule is enforced before writing or
   reviewing a test for it.
2. **The mutation test, done by hand.** Temporarily comment out or invert
   the enforcement (the `if` guard, the `EXCLUDE` clause, the state-machine
   `switch` branch) and ask: which test in the suite now fails? If the
   honest answer is "none," the invariant is untested regardless of how
   much test code exists nearby. This is the single highest-leverage
   adversarial technique in this dossier — it costs nothing but attention
   and it directly answers the question the section title asks.
3. **Check the assertion, not just the test's existence.** A test that
   calls `CreateBooking` twice and asserts `err != nil` on the second call
   proves *something* returned an error — it does not prove the error is
   `ErrCourtDoubleBooked` specifically, and it does not prove the *first*
   booking is still intact and un-mutated. Assert on the specific error/
   state, not just "not nil" / "no panic." (Directly related to Google's
   "Don't Put Logic in Tests" guidance: tests should be concrete
   input→output examples, not code that itself needs debugging — Google
   Testing Blog, "Testing on the Toilet: Don't Put Logic in Tests",
   <https://testing.googleblog.com/2014/07/testing-on-toilet-dont-put-logic-in.html>.)
4. **Check both sides of the invariant.** Every "X must never happen" rule
   has a shadow rule "Y must still be allowed" (double-booking blocked, but
   back-to-back still works; capacity enforced, but exactly-at-capacity
   still succeeds). A suite that only tests the negative direction can pass
   while silently over-blocking legitimate cases — often invisible until a
   real user hits it.
5. **Check that the DB-level enforcement is exercised by an actual DB, at
   least once.** Per CLAUDE.md rule 4/5 and the discussion in §4, a domain
   pre-check test proves the Go code's *opinion* of the invariant; only the
   `-tags=integration` test proves reality matches that opinion. An
   invariant with only a mocked/faked DB layer under test has a gap by
   definition — the mock encodes the author's assumptions about Postgres
   behavior, which is exactly the thing under test.
6. **Ask whether the test would survive a plausible refactor that doesn't
   change behavior.** If renaming a private field or restructuring internal
   control flow (with no observable behavior change) breaks the test, it's
   a change-detector test (§6) — it isn't actually anchored to the
   invariant, just to the implementation shape.
7. **For anything with a race-prone shape (double-booking, capacity,
   payment-vs-webhook), explicitly ask: "what happens if two of these
   happen at the exact same instant?"** If the answer isn't backed by a
   concurrent test, treat the invariant as *unverified* even if a sequential
   test passes — sequential tests structurally cannot observe races.

---

## 6. Anti-patterns to push back on

- **Happy-path-only suites.** The most common failure mode is a test suite
  that proves the system works when used correctly and says nothing about
  misuse, concurrency, or boundaries — exactly what §§3–5 above are the
  antidote to. A Senior QA Engineer's default review comment on any new
  feature PR should be "where's the test for the adjacent invalid case?"
  not "LGTM, tests pass."
- **Change-detector tests.** Tests that fail whenever the implementation
  changes, even when behavior doesn't, "do not add any clarity" and make
  refactoring unsafe because you have to fix the tests just to get back to
  green — a red flag rather than a safety net (Google Testing Blog,
  "Testing on the Toilet: Change-Detector Tests Considered Harmful",
  <https://testing.googleblog.com/2015/01/testing-on-toilet-change-detector-tests.html>).
  Common concrete form in a Go/DDD codebase: asserting on a mock's exact
  call sequence/arguments instead of on the observable outcome, or
  snapshot-asserting an entire struct (including irrelevant fields) instead
  of the fields the test is actually about.
- **Assertions that don't test the invariant.** Covered mechanically in §5
  item 3 — `assert.NoError(t, err)` as the *only* assertion after an
  operation that's supposed to enforce a rule is close to a no-op test. If
  removing the enforcement logic wouldn't make this assertion fail, it
  isn't testing the enforcement.
- **Logic inside tests.** Loops, conditionals, or computed expected-values
  inside a test body mean the test itself now needs to be correct and
  potentially debugged — tests should be "concrete examples of input/output
  pairs," and any necessary logic should be pulled into a separately-tested
  helper (Google Testing Blog, "Testing on the Toilet: Don't Put Logic in
  Tests", <https://testing.googleblog.com/2014/07/testing-on-toilet-dont-put-logic-in.html>).
  Table-driven Go tests are the right pattern *precisely* because the table
  is data, not logic — watch for table-driven tests that then compute the
  expected value with the same formula as production code, which silently
  turns the test into "assert the code agrees with itself."
- **Flaky tests treated as normal.** A flaky test that gets re-run until
  green, or gets a blanket retry wrapper, is actively worse than no test:
  it burns trust in the whole suite and trains engineers to ignore red CI.
  Google's own data-driven response was tooling — automatic detection,
  quarantine off the submit-blocking path, and a filed bug — not a culture
  of shrugging it off (Google Testing Blog, "Flaky Tests at Google and How
  We Mitigate Them", <https://testing.googleblog.com/2016/05/flaky-tests-at-google-and-how-we.html>).
  In this repo specifically: the `-tags=integration` concurrency test is
  the highest-flakiness-risk test by construction (real Docker container,
  real timing-sensitive race) — it should be watched for flake rate, not
  just left running unattended, and a flake there is a signal about either
  test infrastructure *or* a genuine intermittent invariant violation, never
  purely "the CI runner was slow."
- **Testing the mock instead of the system.** If a "unit test" for
  `app.Service` fakes the repository port and asserts the fake was called
  correctly, it's testing that the code calls its dependencies as written —
  useful for wiring, but it is not evidence the conflict rule, the pricing
  math, or the state machine is actually correct. That evidence has to come
  from a test where the real domain logic (or the real Postgres constraint)
  runs.
- **One invariant, two implementations, one test.** CLAUDE.md rule 4
  requires the conflict rule in *both* the domain and Postgres. A suite
  that only ever tests the domain function (because it's easy and fast)
  gives false confidence the invariant holds in production — the
  authoritative layer, by CLAUDE.md's own words, is the one least likely to
  be exercised by a quick unit test, which is exactly backwards from a risk
  standpoint.
- **Skipping the "why" in a design review.** Adopting PayPal's premortem
  habit as a working default (§1): if a design doc for T5/T6 doesn't
  explicitly name at least one way the new invariant could be silently
  violated under concurrency or partial failure, treat that as a review
  finding, not a nitpick.

---

## 7. Sources

- *Software Engineering at Google*, ch. 11 "Testing Overview" (test sizes:
  small/medium/large, ~80/15/5 mix) — <https://abseil.io/resources/swe-book/html/ch11.html>
- *Software Engineering at Google*, ch. 12 "Unit Testing" — <https://abseil.io/resources/swe-book/html/ch12.html>
- *Software Engineering at Google*, ch. 13 "Test Doubles" — <https://abseil.io/resources/swe-book/html/ch13.html>
- Google Testing Blog, "Just Say No to More End-to-End Tests" (2015) — <https://testing.googleblog.com/2015/04/just-say-no-to-more-end-to-end-tests.html>
- Google Testing Blog, "Flaky Tests at Google and How We Mitigate Them" (2016) — <https://testing.googleblog.com/2016/05/flaky-tests-at-google-and-how-we.html>
- Google Testing Blog, "Testing on the Toilet: Change-Detector Tests Considered Harmful" (2015) — <https://testing.googleblog.com/2015/01/testing-on-toilet-change-detector-tests.html>
- Google Testing Blog, "Testing on the Toilet: Don't Put Logic in Tests" (2014) — <https://testing.googleblog.com/2014/07/testing-on-toilet-dont-put-logic-in.html>
- Google Testing Blog, "From QA to Engineering Productivity" (2016) — <https://testing.googleblog.com/2016/03/from-qa-to-engineering-productivity.html>
- Google SRE Book, ch. 17 "Testing for Reliability" — <https://sre.google/sre-book/testing-reliability/> (overview: <https://sre.google/resources/book-update/testing-for-reliability/>)
- Martin Fowler, "TestPyramid" (bliki) — <https://martinfowler.com/bliki/TestPyramid.html>
- Ham Vocke (martinfowler.com), "The Practical Test Pyramid" — <https://martinfowler.com/articles/practical-test-pyramid.html>
- Kent Beck, "Canon TDD" — <https://newsletter.kentbeck.com/p/canon-tdd>
- Meta / automationhacks.io, "Engineering practices @ Meta: #4 Engineers write automated tests, well mostly!" — <https://automationhacks.medium.com/engineering-practices-meta-4-engineers-write-automated-tests-well-mostly-1d4597d8363>
- Meta Engineering Blog, "Autonomous testing of services at scale" (2021) — <https://engineering.fb.com/2021/10/20/developer-tools/autonomous-testing/>
- Gergely Orosz, "How Big Tech does Quality Assurance (QA)" (The Pragmatic Engineer) — <https://newsletter.pragmaticengineer.com/p/how-big-tech-does-qa>
- Uber Engineering Blog, "Simplifying Developer Testing Through SLATE" — <https://www.uber.com/en-IN/blog/simplifying-developer-testing-through-slate/>
- Uber Engineering Blog, "Shifting E2E Testing Left at Uber" — <https://www.uber.com/en-PT/blog/shifting-e2e-testing-left/>
- InfoQ, "PayPal Engineering Teams Implement Premortem Analysis" (2021) — <https://www.infoq.com/news/2021/07/paypal-premortem-analysis/>
- Go Blog, "Introducing the Go Race Detector" — <https://blog.golang.org/race-detector>
- asatarin, "Testing Distributed Systems" (curated resource list: Jepsen, Elle, deterministic simulation testing, TLA+, JCStress, LinCheck, QuickCheck, Porcupine) — <https://github.com/asatarin/testing-distributed-systems>
- PostgreSQL Documentation, "8.17. Range Types" (EXCLUDE constraints, GiST) — <https://www.postgresql.org/docs/current/rangetypes.html>
- JusDB, "PostgreSQL Range Types and Exclusion Constraints: Prevent Double-Booking at the Database Level" — <https://www.jusdb.com/blog/postgresql-range-types-exclusion-constraints>
- JOSS, "Hypothesis: A new approach to property-based testing" (Claessen & Hughes / QuickCheck lineage) — <https://joss.theoj.org/papers/10.21105/joss.01891.pdf>
- Guru99, "Boundary Value Analysis and Equivalence Partitioning" — <https://www.guru99.com/equivalence-partitioning-boundary-value-analysis.html>
- Software Testing Help, "What is Boundary Value Analysis and Equivalence Partitioning?" — <https://www.softwaretestinghelp.com/what-is-boundary-value-analysis-and-equivalence-partitioning/>
- Testsigma, "State Transition Testing Techniques in Software Testing" — <https://testsigma.com/blog/state-transition-testing/>
- Baeldung, "Software Testing: State Transition" — <https://www.baeldung.com/cs/software-testing-state-transition>

Internal (this repo, referenced throughout §4–§6, not external sources):
`/home/user/white-label/CLAUDE.md`, `/home/user/white-label/HANDOFF.md`,
`/home/user/white-label/internal/booking/domain/booking.go`,
`/home/user/white-label/internal/booking/adapter/postgres/concurrency_integration_test.go`,
`/home/user/white-label/docs/adr/0001-dual-invariant-enforcement.md`,
`/home/user/white-label/docs/reviews/01-t1-pricing-quote.md`,
`/home/user/white-label/docs/reviews/03-t3-cancel-booking.md`,
`/home/user/white-label/docs/reviews/04-t4-concurrency-invariant.md`.
