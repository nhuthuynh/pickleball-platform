# T6 Sprint Plan — Payments context

Produced by Ceremony 1 (Backlog refinement) + Ceremony 2 (Sprint planning)
per `docs/process/sprint-process.md`, played jointly by **Product Manager**
(handbook B2) and **Principal Engineer** (handbook B1 + `docs/roles/
principal-engineer.md`). Scope inherited from `HANDOFF.md` T6 and
cross-checked against `docs/requirements/README.md`, `docs/adr/
0006-waitlist-data-model-direction.md`, and `docs/LESSONS.md`'s T5 entries
per the task brief.

**Note on inputs:** at the time this plan was written, `docs/LESSONS.md` had
no `## T5 sprint retro` heading yet — a parallel agent is understood to be
writing it concurrently. This plan proceeded without it per the task's
instruction not to wait; if the retro later surfaces a T5 finding that bears
on T6 scope, treat it as an input to the next backlog-refinement pass, not a
silent gap here. Everything else the task brief asked to be read (HANDOFF.md
T5 cross-cutting notes, CLAUDE.md, requirements/README.md, the mid-sprint
capacity-invariant LESSONS.md entry, booking.go/registration.go/socialplay
adapter code on the T5 sprint branches) was available and read in full.

## Sprint goal

> A `Payment` records money for any payable action — a Booking or a
> Registration today — through an explicit `unpaid → paid → refunded`
> state machine that rejects illegal transitions and is race-safe in
> Postgres, reachable by both an offline Host/Game-Admin-recorded path and
> an online path behind a swappable Stripe anti-corruption layer; Social
> Play's existing `Registration.PaymentStatus` becomes a projection of this
> one source of truth instead of a second, independently-writable field;
> and the Game waitlist promised by ADR-0006 actually ships this sprint.

## Kickoff note

**Scope decisions against `docs/requirements/README.md`:**

- **P1 #13 (PCI guardrail — never accept a raw PAN/CVV field on any request
  DTO)** — in scope, as a process artifact rather than a feature. The
  finding's own recommendation is to add this "as a proto-review checklist
  item the moment Payments proto is drafted" (`research-security-
  compliance.md` §2) — that moment is T6.4 below. T6.4's instructions
  require the checklist to be written down (a short section added to
  `CLAUDE.md`'s golden rules or a new `docs/checklists/proto-review.md`,
  implementer's choice, but it must exist as a durable artifact, not just a
  PR-description mention) *before* `payments.proto` is merged, and the
  proto itself must be reviewed against it. This is cheap or nearly free
  this sprint (there's no card-form feature planned at all — Stripe
  Elements/Checkout tokenizes client-side per the research finding) but
  expensive to retrofit as a review habit after a `card_number` field has
  already shipped once. PM and PE both signed off with no disagreement —
  this is pure downside protection with no scope cost.
- **P1 #8 (no-show fee / account-credit concept)** — **partially** in
  scope; this is the sprint's first genuine PM/PE disagreement, recorded
  per sprint-process.md rather than smoothed over:
  - **PM's position:** empty booked courts from no-shows is called out in
    the research as "the single biggest complaint host/owners have"
    (`research-functional.md` §1.2) — shipping a Payments context this
    sprint without any answer to "the player didn't show and didn't pay"
    ships a payment model that's already behind what every comparable
    platform does on day one.
  - **PE's position:** a *real* no-show fee is a policy-triggered charge —
    it fires automatically when a per-facility cancellation window is
    violated. That per-facility configurable window is P1 #12
    (`research-functional.md` §1.1), which doesn't exist yet (booking
    policy still lives nowhere in the schema) and isn't in this sprint.
    Building automatic no-show detection/charging now means building it
    against a policy that isn't there, which is exactly the "half-built
    mechanism with no trigger to pay it down" anti-pattern the PE dossier
    (§5) and `docs/LESSONS.md`'s own T1 entry (`pricing_rules`'s deferred
    overlap guard, closed only once a real write path existed) both warn
    against.
  - **Resolution:** `Payment`'s payable-action reference (T6.1) is
    designed as an **extensible enum** (`PayableType`) from day one, and a
    `no_show_fee` value is added to that enum *now* — not left for a
    future breaking migration — specifically because a no-show fee, once
    someone decides to charge one, is structurally just another payable
    action with an amount and a payer, identical in shape to a Booking or
    Registration payment. T6.3 (offline recording) explicitly supports a
    Game Admin recording a `no_show_fee` payment through the *same*
    offline-recording mechanism already being built for ordinary
    payments — no new code path, no automation. What's explicitly
    **deferred, not built**: automatic detection of a no-show, automatic
    triggering off a cancellation-window violation, and the account-credit
    ledger (a running per-user balance, materially different from a single
    Payment row) the research flagged as a *lighter-weight alternative* to
    a Stripe refund. Logged here for a T7+ ticket once P1 #12 (facility
    cancellation policy) exists to drive it. PM accepted this as real
    progress on the finding, not a deferral in name only, because a Game
    Admin can manually charge a no-show fee starting this sprint even
    though the platform can't yet detect one automatically.
- **P0 #3 / ADR-0006 (full waitlist entity)** — **in scope**, per the task
  brief's explicit instruction not to defer it again without a real reason,
  and per ADR-0006's own status line naming T6 as the sprint PM extracted a
  commitment for. There is a second, narrower scope negotiation recorded
  here (not a full disagreement — PE proposed a cut, PM accepted it):
  - **PE's technical scoping concern:** ADR-0006 describes the waitlist as
    "keyed to the thing being waited for (a court/slot, or a game — both
    need this per §3.4 and §3.6)." Building both a standalone court/slot
    waitlist *and* a Game waitlist in one ticket is two designs, not one —
    and there is no existing "request a specific court/slot directly"
    feature for a standalone waitlist to hook onto (Booking's write path
    is `CreateBooking`, which either succeeds or returns
    `ErrCourtDoubleBooked` today; there's no "ask to be notified" concept
    at all). A Game waitlist, by contrast, has a concrete, already-built
    hook: T5.2 shipped `domain.ErrGameFull` as "this exact sentinel...
    specifically so a future waitlist ticket can hook a promotion trigger
    onto that boundary without an API-shape change." PE recommended
    scoping T6.6 to the Game waitlist only, with the entity shaped
    (per ADR-0006 and the `Payment`/`PayableType` precedent above) so a
    standalone court/slot waitlist is an additive extension later, not a
    rewrite.
  - **PM's response:** agreed, on the record — the Game case is where
    real, demonstrated demand already exists (it's the literal thing
    `ErrGameFull` rejects today); a court/slot waitlist has no shipped
    feature generating demand for it yet. Logged as deferred-with-reason
    rather than silently dropped, same as T5's own resolution pattern.
  - **What ships:** an ordered (FIFO v1) `waitlist_entries`-equivalent
    queue per Game, an auto-promotion trigger firing when a Registration
    cancels and the Game has open waitlist entries, and a response
    timeout after which an unresponsive promoted player is skipped in
    favor of the next entry — the three elements ADR-0006 names as the
    fixed *shape* of the design. Per `docs/LESSONS.md`'s T5 capacity
    lesson, T6.6's instructions explicitly require answering "what closes
    this race at the DB level?" for both the queue-position assignment and
    the promotion trigger — this is a second capacity-adjacent,
    concurrency-sensitive mechanism in as many sprints, and the T5.4
    loop-2 finding (a domain-only count-then-insert check is not an
    invariant under concurrency) applies to it directly.
- **P1 #7 (Auth0 port shaped around IdP-agnostic OAuth2/OIDC claims, not an
  Auth0 SDK type)** — considered, not folded in. This finding is about the
  *authentication* stub (a cross-cutting item already logged in
  `HANDOFF.md`), not the Payments-specific Stripe ACL T6 actually builds.
  The two are easy to conflate because both are "stub now, real vendor
  later" anti-corruption layers — PE flagged the *pattern* (shape the port
  around vendor-agnostic concepts, not the vendor's SDK types) as directly
  applicable to T6.2's Stripe ACL design and reused it there, but the
  Auth0 ticket itself stays out of scope for T6, unchanged from T5's own
  scoping call.
- **Other P0/P1 items (#1 timezone, #2 currency, #4 GDPR/retention, #5
  split-court, #6 BOLA general, #9 bracket formats, #10/#11 rating
  weighting, #12 cancellation window)** — considered and left out, except
  where already resolved by standing ADRs. #2 (currency) is **already
  decided, not re-litigated**: ADR-0005 named `payments` explicitly as the
  table that will carry `currency_code` "even while v1 only ever populates
  it with one constant value" — T6.1's `Payment` domain type reuses that
  decision directly (a `Money{Cents int64; Currency string}`-shaped value,
  per the ADR), so this shows up as an instruction inside T6.1, not as its
  own ticket. #12 (cancellation window) is the explicit blocker cited above
  for why full no-show automation isn't in T6 — logged, not forgotten. The
  rest are genuinely unrelated to Payments/Waitlist and are logged here so
  they aren't silently dropped, not force-fit into this sprint.

**Architecture note PE flagged for the tickets below (two-way door, not an
ADR):** the context map (`agent-operating-handbook.md` A1) states Payments
*depends on* Social Play (among others), never the reverse — Social Play
must not import `internal/payments/*`. This matters concretely for T6.5
(reconciling `Registration.PaymentStatus`): the update has to flow
Payments → Social Play, not the other way. T6.5 defines a
`port.RegistrationPaymentUpdater` (or similarly named) interface **inside
`internal/socialplay/port`**, implemented by an adapter living under
`internal/payments/adapter/socialplay` (mirroring T5.3/T5.4's
`internal/socialplay/adapter/booking` pattern exactly, just with the
dependency arrow pointed the way the context map already requires) — the
Payments app layer calls into Social Play through that port after a
transition affecting a `Registration`-payable `Payment` commits;
`internal/socialplay/domain` and `internal/socialplay/app` never import
anything under `internal/payments`.

**A second thing PE flagged, not a disagreement:** `app.Service.NewService`
constructors are already at 3 positional args for Booking and 3 for Social
Play; `HANDOFF.md`'s cross-cutting notes already logged this as "worth
revisiting... if a 4th dependency lands." Payments' own `app.Service` is
projected to need at least 4 (payments repo, ID generator, Stripe ACL port,
the new `RegistrationPaymentUpdater` port) from T6.4 onward. PE's
recommendation, accepted by PM as a non-blocking implementation note rather
than its own ticket: T6.4 should introduce an options-struct constructor
for Payments' `Service` from the start rather than growing positional args
past 3 and refactoring later — small enough to fold into T6.4's instructions
without its own ticket.

---

## Tickets

### T6.1 — Add the Payment aggregate with a PaymentStatus state machine and an extensible payable-action reference

**Story:** As the platform, I want a single `Payment` aggregate that records
money against any payable action with a strict `unpaid → paid → refunded`
state machine, so that every context needing "was this paid" has one source
of truth instead of each context inventing its own payment tracking.

**Description:** Bootstraps `internal/payments/domain`, mirroring how
`internal/booking/domain/booking.go` shapes `Booking`'s `Status`/`Cancel`
pattern (illegal transitions rejected, not silently accepted) and how
`internal/socialplay/domain/registration.go` shapes actor-scoped
transitions. Pure, framework-free, TDD-first. No adapter/proto work in this
ticket.

**Instructions:**
1. Functional requirements:
   - Write failing table-driven tests first (CLAUDE.md rule 1), then add
     `domain.Payment{ID, PayableType, PayableID, Amount Money, Method
     (online|offline), Status, StripeReference string (empty for offline),
     RecordedByUserID string (empty for online-Stripe-driven transitions)}`.
   - `Money{Cents int64; Currency string}` — reuses ADR-0005's decision
     directly ("carried into the future `payments` table (T6)"); do not
     reopen whether `payments` gets a currency column, it's already
     decided.
   - `PayableType` is a string-backed enum, **extensible by design**:
     `booking | registration | no_show_fee` for T6 (see kickoff note's P1
     #8 resolution — `no_show_fee` ships as a payable type now even though
     nothing auto-generates one yet). Document in a doc comment, the way
     `booking.Source`'s doc comment does, that `recurring_hire` and
     `subscription` are known future values per the glossary's Payment
     definition (`agent-operating-handbook.md` A2) but are not built this
     sprint — don't add unused enum values speculatively beyond what the
     glossary already names (PE dossier §5, over-engineering).
   - `NewPayment(...)` validates: `PayableID` non-empty
     (`domain.ErrEmptyPayableID`); `PayableType.IsValid()`
     (`domain.ErrInvalidPayableType`); `Amount.Cents > 0`
     (`domain.ErrInvalidAmount` — a zero/negative payment is not a
     meaningful payable action); `Amount.Currency` matches ISO 4217's
     3-letter shape at minimum (a cheap format check, not a full currency
     registry — mirror the "constant value in v1" scope ADR-0005 already
     accepted).
   - `Status` starts `unpaid`. Add explicit transition methods (not one
     generic `SetStatus`, so illegal calls are compile-time-visible, same
     spirit as `Booking.Cancel()` being its own method rather than a
     generic status setter):
     - `MarkPaid(method Method, ref string) error`: legal only from
       `unpaid`. Anything else (already `paid`, or `refunded`) returns
       `domain.ErrIllegalStatusTransition`.
     - `Refund() error`: legal only from `paid`. `unpaid → refunded`
       directly is explicitly illegal (nothing was paid to refund) —
       write a test asserting this exact case, since it is the one most
       likely to be waved through as "should probably be fine."
     - `refunded` is terminal: no method transitions out of it; a test
       must assert calling `MarkPaid` or `Refund` again on a refunded
       `Payment` returns `ErrIllegalStatusTransition`, not a silent no-op.
   - Given/When/Then coverage required (table-driven, one table covering
     all nine `(fromStatus, transition)` combinations across the three
     statuses × two transition methods, so the illegal-transition matrix
     is exhaustive rather than spot-checked):
     `unpaid→MarkPaid`(ok), `unpaid→Refund`(illegal), `paid→MarkPaid`
     (illegal), `paid→Refund`(ok), `refunded→MarkPaid`(illegal),
     `refunded→Refund`(illegal).
   - Add `domain.EnsureOnePaymentPerPayable(candidate Payment, existing
     []Payment) error`, the domain-side fast pre-check mirroring
     `EnsureNoConflict`'s role for Booking (CLAUDE.md rule 4 — expressed in
     the domain, backed by a DB constraint in T6.4): a `(PayableType,
     PayableID)` pair may have at most one `Payment` row (the sprint's
     "one payments row per payable action" AC, taken directly from
     `HANDOFF.md`). Note in the doc comment that this is a *distinctness*
     invariant, not a *capacity* one — closed by a DB unique constraint in
     T6.4, not a locking trigger (contrast with T5.4's capacity-guard
     trigger, which this ticket is not analogous to).
2. Non-functional requirements:
   - Zero non-stdlib imports in `internal/payments/domain` (CLAUDE.md
     rule 2).
   - Table-driven boundary tests: zero-amount, negative-amount, empty
     `PayableID`, malformed currency code, the full transition matrix
     above.

**Story points:** 5

**Labels:** `sprint:t6`, `role:principal-engineer`, `type:story`, `points:5`

---

### T6.2 — Stripe anti-corruption layer: port + stub adapter for the online path

**Story:** As the platform, I want the online payment path to run through
an interface shaped around payment-processor-agnostic concepts, so that
swapping the stub for real Stripe later is an adapter-only change, not a
rewrite of `app`/`domain`.

**Description:** Depends on T6.1. Builds the ACL `HANDOFF.md` T6 names
explicitly ("interface + stub adapter first, real Stripe later"). No real
Stripe SDK dependency is added this sprint — the stub is a deterministic
in-memory fake standing in for a future `internal/payments/adapter/stripe`
package.

**Instructions:**
1. Functional requirements:
   - Define `port.PaymentProcessor` in `internal/payments/port`, shaped
     around the processor-agnostic lifecycle a card payment actually has
     (authorize/intent → capture → refund), not around any Stripe SDK
     type — mirroring the reasoning `research-security-compliance.md` §3
     gives for the Auth0 `TokenVerifier` port (see kickoff note's P1 #7
     discussion): e.g. `CreateIntent(ctx, amount Money, payableID string)
     (intentRef string, err error)`, `CapturePayment(ctx, intentRef
     string) error`, `RefundPayment(ctx, intentRef string) error`. Return
     `internal/payments/domain` sentinel errors
     (`domain.ErrPaymentProcessorUnavailable`,
     `domain.ErrPaymentDeclined`) from the port's contract, never a
     Stripe-shaped error type — there is no Stripe type to leak yet, but
     the port's signature is what a future real adapter has to honor, so
     get this right now (Hyrum's Law, PE dossier §2 — treat the port as a
     promise from day one, not just when it's real).
   - Implement `internal/payments/adapter/stripestub` (name deliberately
     not `internal/payments/adapter/stripe`, so it's obvious at a glance
     this is not the real integration) — a deterministic fake: every
     `CreateIntent`/`CapturePayment` succeeds unless the test seeds a
     specific failure, no network calls, no `STRIPE_*` env vars read.
   - `app.Service.CreateOnlinePayment(...)`: builds a `Payment`
     (`domain.NewPayment`, `Method: online`) then calls
     `port.PaymentProcessor.CreateIntent`; on success, stores the
     `intentRef` as `Payment.StripeReference` (still `unpaid` — creating
     an intent is not the same as capturing funds).
   - `app.Service.ConfirmOnlinePayment(...)`: calls
     `port.PaymentProcessor.CapturePayment`; on success calls
     `Payment.MarkPaid(MethodOnline, intentRef)` (T6.1). On
     `ErrPaymentDeclined`, the `Payment` stays `unpaid` — a declined card
     is not an illegal state transition, it's a transition that simply
     didn't happen; write a test proving the `Payment` is unchanged, not
     erroring out with a status-machine error.
2. Non-functional requirements:
   - **PCI guardrail (P1 #13, kickoff note):** no message or type anywhere
     in `internal/payments/port` or `internal/payments/domain` may contain
     a card-number/CVV/track-data field, by design — there is nothing to
     tokenize server-side because Stripe.js/Elements/Checkout (or the
     mobile SDKs) submit card data directly to Stripe, never through this
     backend. Add a one-line doc comment on `port.PaymentProcessor`
     stating this explicitly, since it's the port a future real Stripe
     adapter will be reviewed against.
   - `internal/payments/domain` still imports nothing beyond stdlib —
     `port.PaymentProcessor` lives in `port`, not `domain`, same layering
     as Booking/Social Play's ports (CLAUDE.md rule 2/3).

**Story points:** 5

**Labels:** `sprint:t6`, `role:principal-engineer`, `type:story`, `points:5`

---

### T6.3 — Offline payment recording path with actor-scoped authorization

**Story:** As a Host or Game Admin, I want to record that a payment was
made offline (cash, bank transfer, or a manually-charged no-show fee), so
that the platform's paid/unpaid tracking stays accurate even for money that
never touches Stripe.

**Description:** Depends on T6.1. The offline half of `HANDOFF.md`'s T6 AC
("offline path where Host/Game Admin records the amount"). Also where P1
#8's manual no-show-fee recording lands (kickoff note) — no automation, a
Game Admin explicitly records it, exactly like any other offline payment.

**Instructions:**
1. Functional requirements:
   - `app.Service.RecordOfflinePayment(ctx, in RecordOfflinePaymentInput)`:
     builds a `Payment` (`Method: offline`, `RecordedByUserID:
     in.ActorUserID`) via `domain.NewPayment`, then `Payment.MarkPaid(...)`
     immediately (an offline recording *is* the payment event — there is
     no separate "intent" step the way Stripe has one).
   - Actor-scoped authorization, mirroring T5.2/T5.5's
     `ErrNotRegistrationOwner` pattern exactly (same known-gap caveat: no
     JWT yet, `ActorUserID` is a request-supplied field, not a verified
     identity — state this explicitly in the PR description and don't
     claim it's a real authorization boundary, per `HANDOFF.md`'s existing
     T5.5 caveat this ticket must not contradict):
     - Recording an offline payment for a `booking`-payable requires the
       actor be the Host who owns the Booking's Game/Competition
       (individual/recurring-hire bookings without a Host — e.g. a direct
       court hire — are out of scope for offline recording in T6; log this
       as a narrower-than-spec scope decision in the PR, don't silently
       widen the ticket to cover it).
     - Recording an offline payment for a `registration`-payable requires
       the actor be the Registration's Game's Host **or** a Game Admin
       assigned to that Game (the glossary's own definition of Game
       Admin's scope, `agent-operating-handbook.md` A2 — "may record
       offline Payments... scoped to the specific Game/Competition they
       are assigned to"). T5 did not build a Game-Admin-assignment
       mechanism; if it's still missing, this ticket must add the minimal
       version needed to test authorization (an assignment list/table),
       not stub the check out — write the failing test first per rule 1.
     - A mismatched actor returns `domain.ErrNotPaymentRecorder` (new
       sentinel, distinguishable from `ErrIllegalStatusTransition` by
       type, same reasoning as T5.2's `ErrNotRegistrationOwner`).
   - **Required test (P1 #8):** a Game Admin can record a
     `PayableType: no_show_fee` payment against a Registration exactly the
     same way they record an ordinary payment — same function, different
     `PayableType` value, proving the extensible-enum design from T6.1
     actually pays off with zero new code paths.
2. Non-functional requirements:
   - `domain.ErrNotPaymentRecorder` must be `errors.Is`-compatible and
     distinct from every other Payments sentinel error, same reasoning as
     T5.2's non-functional requirement on `ErrGameFull`/
     `ErrAlreadyRegistered`/`ErrNotRegistrationOwner`.
   - This ticket must not build the automatic no-show detection/trigger —
     resist scope growth here the same way T5.2 resisted partially
     building the waitlist (kickoff note's P1 #8 resolution; PE dossier
     §5's "silent technical-debt accrual with no trigger to pay it down" —
     here the explicit trigger condition is "P1 #12's facility
     cancellation-window policy exists").

**Story points:** 5

**Labels:** `sprint:t6`, `role:principal-engineer`, `type:story`, `points:5`

---

### T6.4 — Wire Payments to Postgres + proto + gRPC/REST

**Story:** As a client (Vue/Swift/Kotlin), I want `RecordOfflinePayment`,
`CreateOnlinePayment`, and `ConfirmOnlinePayment` as real API endpoints, so
that Payments is reachable outside of Go tests, with the money-handling
invariants this ticket adds actually enforced in Postgres, not just in Go.

**Description:** Depends on T6.1–T6.3. The adapter/infra ticket, mirroring
T5.4's combined proto+DB+adapter+gRPC scope for Social Play. Also where the
PCI proto-review checklist (kickoff note, P1 #13) gets written down and
applied for the first time.

**Instructions:**
1. Functional requirements:
   - **Before writing `payments.proto`:** add a short, durable PCI
     guardrail checklist — either a new `docs/checklists/proto-review.md`
     or a new bullet under `CLAUDE.md`'s golden rules, implementer's
     choice, but it must be a committed artifact, not just a PR-description
     mention — stating: "never accept a raw PAN/card-number/CVV/track-data
     field on any request DTO, in proto or REST; card data is tokenized
     client-side (Stripe.js/Elements/Checkout or the mobile SDKs) and never
     reaches this backend (SAQ A scope, `research-security-
     compliance.md` §2)." Then review the drafted `payments.proto` against
     it explicitly (a line in the PR description confirming this was
     checked, naming the checklist doc).
   - Add `proto/pickleball/payments/v1/payments.proto` with
     `RecordOfflinePayment`, `CreateOnlinePayment`, `ConfirmOnlinePayment`
     RPCs + grpc-gateway REST annotations, mirroring `booking.proto`/
     `socialplay.proto`'s style. Run `make generate` — never hand-edit
     `internal/gen/**` (CLAUDE.md rule 6).
   - Add a `payments` table via a new migration under `db/migrations`
     (`payable_type text`, `payable_id uuid`, `amount_cents bigint`,
     `currency_code char(3)` per ADR-0005, `method text`, `status text`,
     `stripe_reference text NULL`, `recorded_by_user_id uuid NULL`,
     timestamps), plus sqlc queries, following the existing
     `fromFields`-per-query pattern (CLAUDE.md Gotchas — sqlc emits a
     distinct `...Row` type per query).
   - **DB-level guard for T6.1's "one payment per payable action" AC:** a
     `UNIQUE (payable_type, payable_id)` constraint — this is a
     *distinctness* race (LESSONS.md's T5 lesson: "a unique index closes a
     distinctness race"), so a plain unique index/constraint is sufficient
     here, unlike T5.4's capacity-guard trigger which needed a row lock
     because it was a *counting* race. Do not add a locking trigger where
     a unique constraint already closes the race — that would be the
     inverse mistake (over-engineering a two-way-door problem, PE dossier
     §3/§5).
   - Translate the resulting Postgres `23505` unique-violation into a
     stable `domain.ErrPaymentAlreadyRecorded` sentinel in the adapter
     (CLAUDE.md rule 5), mirroring `translateRegistrationErr`'s pattern in
     `internal/socialplay/adapter/postgres/repository.go`.
   - Implement `internal/payments/adapter/postgres` (repository),
     `internal/payments/adapter/stripestub` is already built (T6.2), and
     `internal/payments/adapter/grpcapi`; register alongside Booking/Social
     Play in `cmd/server`.
   - **Constructor shape (kickoff note):** `app.Service`'s constructor
     takes an options struct (`ServiceOptions{Payments port.Repository,
     IDs port.IDGenerator, Processor port.PaymentProcessor,
     RegistrationUpdater socialplayport.RegistrationPaymentUpdater}` or
     equivalent) from the start, not four positional args — this is the
     "4th dependency lands" trigger `HANDOFF.md`'s cross-cutting note
     already named.
   - Smoke-test AC (PR description, curl-list style): recording an offline
     payment for a Registration returns 200 and a subsequent duplicate
     attempt for the same `(payable_type, payable_id)` returns 409, mapped
     from `ErrPaymentAlreadyRecorded`; an actor mismatch on
     `RecordOfflinePayment` returns a 403-shaped rejection, not a 500.
2. Non-functional requirements:
   - Every new Postgres error code this ticket introduces (`23505` at
     minimum) is translated to a domain error in the adapter — no raw
     `pgconn.PgError` crosses into `app`/`domain` (CLAUDE.md rule 5).
   - `make down && make up` after the schema change (Gotchas: initdb.d
     only applies on a fresh volume).
   - `make test` green, including this context in the top-level
     race/coverage run.

**Story points:** 8

**Labels:** `sprint:t6`, `role:principal-engineer`, `type:story`, `points:8`

---

### T6.5 — Reconcile Social Play's Registration.PaymentStatus with the Payment aggregate

**Story:** As a developer building on either context, I want
`Registration.PaymentStatus` to always reflect the real `Payment` for that
Registration, so that "is this player's spot paid for" has exactly one
source of truth, per the glossary's own definition of `Payment`.

**Description:** Depends on T6.4 (needs a real, persisted `Payment` whose
transitions can be observed) and on T5.2's existing
`Registration.PaymentStatus` field (`internal/socialplay/domain/
registration.go`, shipped as "modelling only... a later app-layer method
lets a Game Admin flip it once offline/Stripe payment recording exists" —
this ticket is that later method, built the right direction per the context
map). Without this ticket, T6 ships two independently-writable sources of
truth for the same fact, which is exactly the kind of drift CLAUDE.md rule
4 exists to prevent for invariants — this is the same failure shape applied
to a plain data-consistency fact rather than a capacity/uniqueness
invariant, and is worth closing in the same sprint the second field's
producer (`Payment`) is built, not later.

**Instructions:**
1. Functional requirements:
   - Define `port.RegistrationPaymentUpdater` in `internal/socialplay/port`
     (the context that is *depended on*, per the architecture note above —
     Social Play does not import Payments): `UpdatePaymentStatus(ctx,
     registrationID string, status domain.PaymentStatus) error`, returning
     Social Play's own sentinel errors (e.g.
     `domain.ErrRegistrationNotFound`) — Payments' adapter translates
     whatever it gets back, never leaks a raw error type across the
     boundary (CLAUDE.md rule 5, same pattern as
     `internal/socialplay/adapter/booking.Reservation`).
   - Implement `internal/payments/adapter/socialplay` — the adapter
     satisfying that port from the Payments side, calling Social Play's
     real `app.Service` (or a narrower method added for this purpose).
     This is the mirror image of T5.3/T5.4's
     `internal/socialplay/adapter/booking` package: same shape, dependency
     arrow pointed the direction the context map requires here.
   - `payments/app.Service.ConfirmOnlinePayment` and
     `RecordOfflinePayment` (T6.2/T6.3), on a successful `MarkPaid` where
     `PayableType == registration`, call
     `RegistrationPaymentUpdater.UpdatePaymentStatus(ctx, PayableID,
     paid)`. On `Refund()`, likewise push `unpaid` back (or a distinct
     Social-Play-side status if `Registration.PaymentStatus` needs a third
     state to represent "was paid, now refunded" — **decide this
     explicitly as part of the ticket, don't guess**: either extend
     `socialplay/domain.PaymentStatus` with a `refunded` value mirroring
     Payments' own state machine, or document that Social Play only ever
     needs the binary paid/unpaid distinction and a refund simply reverts
     to `unpaid`; write the decision and its reasoning into the PR
     description since it's a real modelling choice, not a mechanical
     step).
   - **Required test:** an in-memory fake `RegistrationPaymentUpdater`
     proves `ConfirmOnlinePayment` and `RecordOfflinePayment` call it
     exactly once with the right registration ID and status on success,
     and **not at all** when `PayableType == booking` (a Registration
     update path must never fire for a Booking payment).
   - **Required test, cross-context:** using T6.4's real Postgres adapters
     for both contexts (an integration test, `-tags=integration`, mirroring
     T4/T5.4's pattern), recording an offline payment for a live
     Registration is observable as `PaymentStatus: paid` on a subsequent
     `GetRegistrationByID` — proves the wiring end to end, not just that
     the port is called.
2. Non-functional requirements:
   - `internal/socialplay/domain` and `internal/socialplay/app` must not
     import anything under `internal/payments` (verify with `go vet`/import
     check, same discipline as T5.3's booking-boundary check) — only
     `internal/payments/adapter/socialplay` may see both.
   - Once this ticket merges, `Registration.PaymentStatus` must no longer
     be settable by any Social Play code path other than through this new
     port-driven update (if T5 left a direct setter for future
     Game-Admin use, either remove it or gate it so it can't silently
     diverge from `Payment`'s state) — call this out explicitly in the PR
     if such a path exists, don't leave two writers.

**Story points:** 3

**Labels:** `sprint:t6`, `role:principal-engineer`, `type:story`, `points:3`

---

### T6.6 — Game waitlist: ordered queue, auto-promotion on cancellation, and a response timeout

**Story:** As a Player, I want to join a waitlist when a Game is full, so
that I'm automatically offered the next open spot if someone cancels,
instead of having to keep checking manually; and as a Host, I want an
unresponsive waitlisted player to not block the slot forever.

**Description:** Fulfils ADR-0006, explicitly scoped to T6 per that ADR's
status line and the task brief's instruction that it must not slip again.
Scoped to **Game waitlists only** this sprint (kickoff note's PE scoping
recommendation, PM-accepted) — a standalone court/slot waitlist is
explicitly deferred, logged below, not built here. Depends on T5's `Game`/
`Registration` (existing, on the T5 sprint branches) and, for the
promotion trigger, on `Registration.Cancel` (T5.2) already existing —
**does not depend on any T6.1–T6.5 Payments ticket**; this ticket can run
in parallel with the Payments chain above if capacity allows, per PM/PE's
sprint-planning call on sequencing.

**Instructions:**
1. Functional requirements:
   - Failing tests first. Add `domain.WaitlistEntry{ID, GameID, PlayerID,
     Position int, Status (waiting|promoted|expired|cancelled)}` to
     `internal/socialplay/domain` (same package as `Game`/`Registration` —
     this is Social Play's entity, not a new bounded context; the
     glossary, `agent-operating-handbook.md` A2, should get a
     `Waitlist Entry` term added in this ticket's PR, per CLAUDE.md rule 7
     "add it here in the same PR that introduces it").
   - `domain.JoinWaitlist(game Game, existingRegistrations
     []Registration, existingEntries []WaitlistEntry, playerID string)
     (WaitlistEntry, error)`: only legal when the Game is actually full
     (i.e., `domain.Register` would currently return `ErrGameFull` for
     this player) — reuse that exact check rather than duplicating
     capacity-counting logic in a second place (call the same counting
     helper `Register` uses, refactored into a shared unexported function
     if needed). A player already actively registered, or already on the
     waitlist, gets `domain.ErrAlreadyOnWaitlist`/`ErrAlreadyRegistered`
     as appropriate. `Position` is assigned as `len(existing non-cancelled
     entries) + 1` (FIFO, per ADR-0006's v1 default).
   - **Auto-promotion:** when a `Registration.Cancel` succeeds and frees a
     capacity slot, the app layer (not the domain — this needs to
     orchestrate across `Game`/`Registration`/`WaitlistEntry`, same
     app-layer-orchestrates-domain-doesn't shape as `ScheduleGame`) looks
     up the Game's oldest `waiting` entry and transitions it to
     `promoted`, starting a response-timeout window. A promoted entry does
     **not** yet consume a capacity slot (it's an offer, not a
     confirmation) — write a test proving a promoted-but-unconfirmed entry
     doesn't block a *different* direct `Register` call for the same
     freed slot from a non-waitlisted player if the promoted player's
     window has already expired (see timeout below), and a test proving
     it correctly *does* reserve the slot while the window is still open
     (an explicit confirm-then-register-fails-full test).
   - **Response timeout:** a promoted entry not confirmed within the
     window (PM/PO call on exact duration per ADR-0006 — default to a
     named constant, e.g. 30 minutes, with a comment stating it's a
     product-tunable value, not load-bearing on any invariant) transitions
     to `expired`, and the *next* `waiting` entry is promoted in its
     place — write the test proving this cascades (player 1 expires,
     player 2 gets promoted next, not left stuck).
   - **What actually closes the race at the DB level (LESSONS.md T5
     lesson, explicitly required by the task brief for this ticket):**
     two concurrent events can both try to promote off the same freed
     slot (e.g. two registrations cancelling near-simultaneously, or a
     timeout-expiry sweep racing a fresh cancellation) — analyze this the
     same way T5.4 loop-2 analyzed the capacity race, and answer in the
     PR description, before writing the migration: is this a
     *distinctness* race (closed by a unique constraint, like T6.4's
     `payments` uniqueness) or a *counting/ordering* race (closed by a row
     lock/trigger, like T5.4's capacity guard)? Promotion is
     ordering-shaped (which entry is "next" must be computed
     consistently under concurrent cancellations), so the expected answer
     is a DB-level guard analogous to T5.4's `FOR UPDATE`-locking trigger
     on the `games`/`waitlist_entries` rows — but this ticket must verify
     that reasoning with a real concurrency test (two simultaneous
     cancellations on a Game with one waitlist entry and capacity 1;
     assert exactly one promotion happens, not zero, not two), not assume
     it by analogy. Per CLAUDE.md rule 10, this needs more than one run
     including a cold start before calling it proven.
2. Non-functional requirements:
   - This ticket is Postgres/proto/adapter-inclusive (unlike T5's split
     between T5.1–T5.3 domain-only and T5.4 wiring) — given the
     concurrency requirement above cannot be honestly verified without a
     real DB, don't defer persistence to a follow-up ticket the way T5
     split domain-first; if the ticket turns out too large to finish
     within the loop cap as a result, split it (loop-mechanics rule)
     rather than skip the concurrency proof.
   - Standalone court/slot waitlists (not tied to a Game) remain
     explicitly out of scope — log this in the PR and in ADR-0006's status
     line as "Game waitlists shipped T6; court/slot waitlists deferred,
     no ticket yet, revisit if a direct court-request feature is ever
     built."
   - Update ADR-0006's Status section to reflect what actually shipped
     once this ticket merges — don't leave it saying "Accepted in
     direction" with stale T5/T6 language after the real thing exists.

**Story points:** 8

**Labels:** `sprint:t6`, `role:principal-engineer`, `type:story`, `points:8`

---

### T6.7 — Object-level authorization regression tests across Payments' new endpoints

**Story:** As a platform operator, I want every new Payments endpoint that
acts on a specific payable action to reject a mismatched actor, so that one
user can't record or view another user's payment data before real auth
exists (mirrors T5.5's closure of the same class of gap for Social Play).

**Description:** Depends on T6.4 (needs real endpoints) and T6.3 (needs the
actor-scoping logic it's proving). QA-owned, same role/shape as T5.5.

**Instructions:**
1. Functional requirements:
   - Add an integration-level test (mirroring T5.5's pattern, `-tags=
     integration` or handler-level if a full Postgres round trip isn't
     needed) that: a Player who is neither the Game's Host nor an
     assigned Game Admin attempts `RecordOfflinePayment` against another
     player's Registration, and asserts the request is rejected with the
     mapped status — not a 500, not a silent success.
   - Same proof for the Booking-payable path: a user who is not the
     Booking's owning Host attempts `RecordOfflinePayment` against it.
   - As with T5.5, document explicitly (PR description + a line added to
     `HANDOFF.md`'s existing Auth cross-cutting item, not a new one) that
     this proves the *object-level* check given a claimed
     `ActorUserID`, not real authentication — same caveat T5.5 already
     established, must not be contradicted or re-litigated here.
2. Non-functional requirements:
   - No real authentication work in this ticket — same boundary T5.5 held.

**Story points:** 3

**Labels:** `sprint:t6`, `role:qa`, `type:story`, `points:3`

---

## Sprint totals

- **Tickets:** 7 (T6.1–T6.7)
- **Total story points:** 37 (5 + 5 + 5 + 8 + 3 + 8 + 3)
- **In-scope P0/P1 findings addressed:** P1 #13 (PCI guardrail — T6.4),
  P1 #8 (no-show fee, partial — extensible `PayableType` + manual
  Game-Admin recording in T6.1/T6.3; automatic detection/account-credit
  ledger explicitly deferred, see kickoff note), P0 #3 / ADR-0006 (Game
  waitlist — T6.6, scoped to Game only per PM/PE negotiation).
- **Genuine disagreements recorded (not manufactured consensus):** P1 #8's
  scope (PM wanted full no-show fee automation; PE scoped to manual
  recording + extensible schema pending P1 #12); ADR-0006's breadth (PM
  initially wanted both court/slot and Game waitlists; PE narrowed to Game
  only, PM accepted).
- **Explicitly deferred, not forgotten:** automatic no-show detection and
  an account-credit ledger (→ T7+, blocked on P1 #12's facility
  cancellation-window policy), standalone court/slot waitlists (→ no
  ticket yet, ADR-0006 status updated to reflect this), P1 #7's Auth0 port
  shaping (unchanged cross-cutting backlog item, not Payments-specific),
  real Stripe adapter behind `port.PaymentProcessor` (stub only this
  sprint, per `HANDOFF.md`'s own "stub first, real Stripe later" framing).
