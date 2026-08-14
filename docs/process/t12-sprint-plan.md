# T12 Sprint Plan — Real authentication, the two aging authorization gaps, and the process fixes T11's retro made concrete

Ceremonies 1 and 2 per `docs/process/sprint-process.md`, six-role team
(briefs: `docs/agent-operating-handbook.md` Part B), against `HANDOFF.md`'s
T10/T11 entries and its Cross-cutting section, `docs/process/t11-sprint-plan.md`
(whose A2–A9 sections are this document's structural template), and
`docs/process/t11-retro.md`'s six findings and eight recommendations —
threaded into this plan below, one subsection per finding, adopted or
adapted or explicitly declined, never silently dropped.

**Every claim in this plan that could be checked against the working tree
was checked this ceremony**, per T11 retro finding 5. Where a cell or a
sentence cites a precedent, it names the real path and says how it was
verified; where a precedent was looked for and *not* found, that is stated
as a negative finding rather than dressed up as a match. The commands are
recorded inline so a reader can re-run them rather than trust them. Two
of this ceremony's checks changed the plan materially — see A3 (the
`vet-integration` command has a dependency the retro's own snippet does
not) and A6 (a real, unticketed WCAG 3.3.4 defect on a shipped screen).

---

# Part A — Ceremony 1 (backlog refinement)

**Product Manager + Principal Engineer**, per `sprint-process.md`. PM drove
scope/value framing; PE drove technical sequencing/feasibility. Both sign
off on every ticket below.

## A0. What this ceremony inherits

T11 shipped its full 47 points across 9 tickets (PRs #113–#121, retro doc
#122): discounts end to end, Club recurring hires end to end, a WCAG 2.2 AA
audit of everything shipped through T10, and three T10-retro process fixes
threaded into execution. Its retro found no defect that reached the shared
branch. So T12 does not inherit a rescue job; it inherits a **queue of
aging, well-specified items that every sprint since T6 has deferred with a
reason**, and one process question two sprints have now let lapse.

T11's own plan closed with the list, and named the reason each item was not
taken: `RefundPayment` ("real candidate for T12's Ceremony 1 to pick up
first"), `CancelGame` (same reasoning), observability ("genuinely T12+"),
and real auth — **"large enough to deserve its own sprint's full Ceremony
1/2 scoping... Candidate for its own dedicated near-term sprint; named here
so it isn't mistaken for forgotten."**

T12 is that sprint. The scoping decision below is therefore not a fresh
judgement call so much as the execution of a commitment three sprints have
now made in writing, and the honest reading of T11's own words is that
declining it a fourth time would need a *new blocking fact* — this ceremony
looked for one and did not find it (A1).

## A1. Is real auth actually buildable now? — checked, not assumed

This is the load-bearing question of the sprint, so it gets a real
verification pass rather than an inherited conclusion.

**What exists today (verified this ceremony):**

- `grep -rin "jwt\|auth0" --include=*.go cmd/ internal/` returns **no
  implementation** — only two comments in `internal/payments` referring to
  the absence (`app/service.go:262`, "there is no JWT yet"). There is no
  auth package, no token verification, no principal type.
- `cmd/server/main.go:238-240` constructs the gRPC server with
  `grpc.ChainUnaryInterceptor(grpcrecovery.UnaryInterceptor(logger))` and
  the stream equivalent. Its own comment (line 235) states the chain form
  was chosen so that **"adding auth/metrics/..."** later does not disturb
  the recovery interceptor's position. The seam this sprint needs was
  deliberately left open by PR #89 and is still open.
- `internal/platform/` contains exactly `grpcrecovery`, `idgen`, `pg`
  (`ls internal/platform/`). `internal/platform/grpcrecovery/recovery.go`
  is a complete, tested, documented unary+stream interceptor pair — a real,
  path-checkable precedent for an `internal/platform/auth` sibling, not an
  invented one.
- The claimed-actor surface is **54 occurrences of `actor_user_id` /
  `actor_player_id` across all six protos** (`grep -rc` per file: booking
  13, identity 11, socialplay 9, facilities 8, payments 7, competitions 6).
  That is the migration's real size, counted rather than estimated.

**What is genuinely blocked, and what is not.** The distinction matters,
because "real auth" has historically been treated as one indivisible thing:

- **Provisioning an Auth0 tenant, registering the application, configuring
  the JWKS endpoint** — *not doable from a coding session*, exactly like
  the Jenkins server-side work `HANDOFF.md` has honestly recorded as
  unticketable since SCRUM-6. No session in this project's history has had
  a reachable external identity provider.
- **Verifying a signed JWT and turning it into a verified principal the
  handlers can trust** — *entirely doable here*, and is the part every
  authorization gap in this codebase is actually waiting on. RS256
  verification against a JWKS is testable with a locally-minted RSA keypair
  and an in-process key source: no network, no Docker, no external tenant.
  The `port`/adapter split this project already applies to Stripe
  (`port.PaymentProcessor` + `adapter/stripestub`, verified present) is the
  same shape — a `TokenVerifier` port with a real verifying implementation
  and test key material, with the IdP's *hosting* left as the disclosed
  server-side gap.

**PM/PE's scoping decision:** T12 takes real auth as its spine — the
platform capability (T12.2) and the three context-migration tickets that
consume it (T12.7, T12.8, T12.9) — plus the two aging authorization
follow-ups that are independently buildable (T12.3 `RefundPayment`, T12.4
`CancelGame`), the retro's own highest-value mechanical fix (T12.1
`vet-integration`), one verified WCAG defect (T12.5), and the board-of-record
resolution (T12.6). **Total: 46 points across 9 tickets** — deliberately
sized at T11's shape (47/9) rather than above it.

**The gap this closes is not hypothetical.** `HANDOFF.md`'s T10.2 bullet
records a real, disclosed security defect: `CreateUser` takes the caller's
own chosen UUID as the row's permanent primary key, so an anonymous caller
can permanently occupy an identity a real person will later need — "a
persistent, targeted denial-of-service, not a rejected mutation." That
bullet names its own closure condition in those words: *"Must close the
moment real auth exists"* — mint `User.ID` from the authenticated
principal's verified subject claim. T12.9 is that closure, and it is the
single most consequential ticket in this sprint.

## A2. Threading T11 retro finding 1 — the wave roll-call (a coordinating-session practice, not a ticket)

**Adopted in full, and stated here rather than ticketed, because it is not
work any implementer performs.** Finding 1 established that two Wave-1
implementer sessions in T11 finished correct work and never opened a PR
(T11.4's commit existed only locally; T11.9's branch was pushed but no PR
opened), and — the part the retro correctly identified as the actual
process failure — the coordinating session continued working for another
43–48 minutes and then ended the work block without ever comparing the
dispatch list against the set of PRs that existed.

**Standing practice for T12's coordinating session, binding on every work
block:**

> Before ending any work block, perform an explicit **wave roll-call**.
> For every ticket dispatched in any open wave and not yet merged, state
> which of three states it is in: **a PR exists**, **a remote branch exists
> but no PR**, or **neither**. A ticket in the "neither" state is a
> finding to be named and acted on, not an absence of news.

Two details the retro's evidence makes non-optional, recorded so the
practice is not weakened into something cheaper:

1. **A branch listing is not sufficient.** T11.4's commit never left its
   worktree, so `list_branches` would have shown nothing and reported it as
   clean. The roll-call must be driven from the **dispatch list** (the wave
   table in A12 below), asking about each ticket by name, not from whatever
   artifacts happen to exist.
2. **The check is cheap and the cost of skipping it is asymmetric.** The
   roll-call is one comparison against a list this plan already contains.
   The cost of skipping it was, in T11, one ticket's finished work sitting
   one cleanup away from being lost entirely.

See A9 for how this ceremony resolves the PE/PdE disagreement about
whether *polling* should additionally be adopted.

## A3. Threading T11 retro finding 2 — `vet-integration` becomes a ticket, and the retro's own snippet needed a correction

**Adopted as a real Wave-1 ticket (T12.1), not deferred prose** — this was
the retro's own #1 recommendation and its reasoning is the strongest in the
document: the same file broke twice in one sprint because
`//go:build integration` makes it invisible to every gate a session can run.

**Verified this ceremony, against the working tree rather than the retro's
summary:**

- `make ci`'s target list is `generate tidy lint test-domain test-tools
  generate-client lint-web test-web build-web` (Makefile line 125) —
  **confirmed, no `-tags=integration` step anywhere.**
- `make test` (line 36) is the only target passing `-tags=integration`, and
  it *runs* the suite, which is why `ci-integration` (line 135) hard-guards
  on Docker.
- The blind spot is **larger than CLAUDE.md currently says.** CLAUDE.md's
  Gotchas section names one integration-tagged file (booking's
  `concurrency_integration_test.go`). `grep -rl "go:build integration"`
  finds **11 files across four contexts** — socialplay (5), booking (1),
  competitions (2), payments (3). Eleven files, not one, are invisible to
  every runnable gate.

**The correction that matters, found by running the command rather than
citing it.** The retro proposes a bare target:

```make
vet-integration:
	go vet -tags=integration ./...
```

Run in this environment on a clean checkout, that **fails** — not on a
tagged file, but on missing generated code:

```
internal/booking/adapter/grpcapi/handler.go:22:2: no required module provides
package .../internal/gen/pickleball/booking/v1
```

`internal/gen/**` is gitignored and only exists after `make generate`
(CLAUDE.md's own Gotchas section says so). So the target must **depend on
`generate`**, or an implementer who adds it verbatim will meet a failure
that has nothing to do with build tags and may "fix" it in the wrong
direction. Inside `make ci` the ordering is already correct (`generate`
runs first), so the retro's placement recommendation — after `test-domain`
— stands unchanged; it is the *standalone* invocation that needs the
dependency. This is exactly the class of plan error finding 5 asks a
ceremony to catch before an implementer does, and it was caught by running
the command rather than transcribing it.

T12.1's Instructions carry this. It also carries a **factual correction**
to CLAUDE.md's Gotchas line (one file → eleven, four contexts). That is a
correctness fix to a statement now demonstrably false, and it is
deliberately **not** the CLAUDE.md/DoD *reminder* QA argued for in the
retro's recorded disagreement — see A9, where that disagreement is left
open rather than quietly decided by this ticket's shape.

## A4. Threading T11 retro finding 3 — the dispatch-isolation check now asks about existing shared files, and one collision class is designed out entirely

**Adopted, extended, and — for the sprint's worst instance — removed rather
than merely flagged.**

Finding 3 established that A10's isolation check compared the files each
ticket would *create* and therefore missed `internal/booking/domain/errors.go`,
which neither ticket created and both appended to. The retro's adopted fix
is a second question in the isolation table: *for each pair of tickets in
the same or adjacent waves, which **existing** files will both append to?*
A12's table asks it, with `internal/<context>/domain/errors.go` named
explicitly as the retro requires.

**T12's worst shared-append target is not `errors.go` — it is
`cmd/server/main.go`**, and it is worse in kind: four tickets (T12.2,
T12.7, T12.8, T12.9) would naturally append to the same wiring file, versus
T11's two. Checked against the tree: `cmd/server/main.go` registers gRPC
*services*, not individual RPCs (`grep -n "Register.*Server" cmd/server/main.go`),
so adding an RPC to an existing service — T12.3's `RefundPayment`, T12.4's
`CancelGame` — does **not** touch it. That halves the exposure before any
sequencing is applied.

**PE ruling, taken at planning time specifically to remove the rest of it
(see A11, Ruling 2):** the set of methods that require an authenticated
principal is **not** a single literal in `main.go`. Each context's
`grpcapi` package declares its own `AuthenticatedMethods()`, and `main.go`
composes them. The consequence is that T12.7, T12.8 and T12.9 each append
to **their own context's file**, not a shared one. This is the difference
between predicting a collision and designing it out, and it is the direct
product of finding 3 being written down.

A residual one-line composition edit in `main.go` remains for each
migration ticket. A12 states the merge-order rule for it up front, per the
retro's requirement, rather than leaving it to surface as
`mergeable_state: dirty`.

## A5. Threading T11 retro finding 4 — a disclosed gap must produce a durable record, and this ceremony creates the ones it owes

**Adopted, and — the part that makes it more than a restatement — applied
to this ceremony's own output before any ticket is dispatched.**

Finding 4 established that T11.5's disclosed owner-only-read gap was closed
by T11.6 only because the coordinating session's memory carried it across a
33-minute gap; no issue and no `HANDOFF.md` bullet was ever created, which
the plan's own A3 rule required. The retro's judgement is precise and
adopted verbatim: **in-flight briefing is a dispatch mechanism, not a
tracking mechanism, and A3 requires both.**

**Standing rule for T12, restated in every ticket's Instructions that could
produce one:**

> When a ticket discloses a real gap it is not closing, or a reviewer
> decides not to block a merge on a disclosed gap, **the same PR or review
> that records the decision creates the durable record** — a GitHub issue
> (per A8's resolved board-of-record rule, cross-sprint items are exactly
> what issues are now *for*), naming the PR that disclosed it and the
> ticket or sprint expected to absorb it. Briefing the next implementer is
> still worth doing; it points at the tracked item, it does not replace it.

**This ceremony owed four such records and created them.** Backlog
refinement surfaced four real, verified, cross-sprint items that this plan
explicitly declines to take (reasons in "Explicitly deferred" in Part B).
Under the pre-T12 practice they would have been deferred in prose for the
seventh consecutive sprint — which is the precise failure mode finding 4
describes. They are now GitHub issues, opened during this ceremony:

| Issue | Item | Open since | Verified this ceremony |
|---|---|---|---|
| #123 | `app.NewService` → `ServiceOptions` for `booking` (7 positional params) and `socialplay` (5) | T1 note, re-flagged T6.6, T11.5 | `grep -n "func NewService"` — booking takes 7 named params (`service.go:47-55`), socialplay 5 (`service.go:59`); `competitions`/`payments` already use `ServiceOptions` (`service.go:87`/`:76`) |
| #124 | Game cancellation does not cascade to its Bookings/Registrations | T5.1 | `internal/socialplay/domain/game.go:183` — `Cancel()` flips `Status` only; its own doc comment says cascading is "explicitly out of scope for T5.1" |
| #125 | Competitions-shaped `PayableType` + payments port/adapter pair | T9.6/T9.7 | `internal/payments/adapter/competitions/` exists but Competitions remains cash-only per `HANDOFF.md`'s own bullet |
| #126 | Real per-Game price field (retires T8.10's `PLACEHOLDER_REGISTRATION_FEE_CENTS`) | T8.10 | `HANDOFF.md` T8.10 bullet; no price field on `domain.Game`/`domain.Registration` |

T12.6 cross-links these from `sprint-process.md`'s amended board-of-record
section, so the rule and its first four instances land together.

## A6. Threading T11 retro finding 5 — evidence markers, no unverifiable citations, and what a real check turned up

**Adopted in both halves.**

**Half one — the cross-context dependency table (A10) carries an evidence
marker in every cell**, naming what was checked and how. Finding 5's
sharpest observation is that in T11's A8 table, *the single claim carrying
no evidence marker was the single claim that was false*
(`socialplay/port.IdentityLookup`, which `git log --all -S` proves never
existed). The discipline is therefore not decoration: an unmarked cell is
the signal. Any precedent this plan could not point at a real path was
**dropped rather than asserted** — and where a precedent was sought and not
found, the table says so.

Applied concretely, this ceremony verified before citing:

- `internal/platform/grpcrecovery` — real, and the model for
  `internal/platform/auth` (`ls internal/platform/`, file read).
- `domain.Payment.Refund()` — real, `internal/payments/domain/payment.go:145`,
  with `StatusRefunded` at `:70`.
- `port.PaymentProcessor.RefundPayment` — real,
  `internal/payments/port/payment_processor.go:59`, implemented by
  `adapter/stripestub/processor.go:107`.
- `ErrNotGameHostOrAdmin` — real, `internal/socialplay/domain/errors.go:179`,
  the sentinel T12.4's new `ErrNotGameHost` mirrors.
- **A negative finding, stated rather than skipped:** there is no existing
  JWT/OIDC verification precedent anywhere in this repo to mirror.
  `grep -rin "jwt\|auth0" --include=*.go` returns only comments about its
  absence. T12.2 is genuinely first; its shape follows the precedent that
  *does* exist (`grpcrecovery`'s interceptor pair and the
  `port`+`stub-adapter` split used for Stripe), and the ticket says so
  rather than sending an implementer looking for an auth pattern to copy.

**Half two — recommendation 7, specify the property, not the reference
implementation.** Finding 5's Designer half showed A6's four named signals
would all have missed the very control T11.6 was written to guard. T12.5 is
the only ticket with an assertion of this class, and its AC is phrased as
the retro recommends: *the check must find the control under every
identification shape a reasonable implementation could use, and must be
mutation-checked against the actual control this ticket ships.* No signal
list is enumerated in the ticket text, deliberately.

**And half two turned up a real defect, which is why this subsection is not
just process.** The instruction to this ceremony was to verify — not assume
— T11's A6 claim that T11.3/T11.6's new screens were "built to the bar from
the start." Verified, in both directions:

- **The automated half is better than claimed, and structurally so.**
  `web/src/__tests__/accessibility.spec.ts` sweeps all three new routes
  (`/facilities/:facilityId/discounts`, `/facilities/:facilityId/rental-requests`,
  `/clubs/rentals` — read directly in `ROUTES_UNDER_TEST`), and its first
  test asserts route coverage **by path identity against the real router**,
  so a new route cannot land without landing in the sweep. The screens also
  carry real error identification (`role="alert"` + `aria-describedby` on
  every field in `FacilityDiscounts.vue`) and live regions. The claim
  holds — and it is enforced, not merely honoured.
- **The manual half never reached them, and there is a concrete defect.**
  T11.7 ran in Wave 1; T11.3 and T11.6 merged in Waves 3 and 4. Its
  keyboard/screen-reader spot check and its 3.3.4 (Error Prevention —
  Legal/Financial/Data) pass therefore could not have covered screens that
  did not exist. Checked directly:
  `web/src/views/FacilityRentalRequests.vue:159-172` wires **"Approve
  request" and "Reject request" as single-click, one-way transitions with
  no confirmation step and no undo window** — and the file's own header
  (lines 16–31) confirms approval generates real Bookings across every
  generated occurrence and that both transitions are terminal. A one-click,
  irreversible, Booking-generating, data-consequential action is precisely
  what 3.3.4 governs, and it is the criterion T11.7's own scope named as
  load-bearing.

That is a verified, unticketed WCAG 2.2 AA gap on a shipped screen, found
because the instruction said to check rather than assume. It is T12.5.

## A7. Threading T11 retro finding 6 — the board-of-record question, resolved

**Resolved. This ceremony has the standing the retro correctly said it
lacked**, and the question is not left to lapse a third time.

The retro's own framing: `sprint-process.md` names GitHub Issues + labels
as the board of record and says *"the ticket **is** the GitHub issue"*,
while T11 opened zero issues for its nine tickets and closed all 42
historical ones — verified again this ceremony,
`list_issues(state: OPEN)` returns `totalCount: 0`, and the newest issue in
the repository is #111 from T11's own Ceremony 1. Two consecutive sprints
have run with the sprint-plan document as the real ticket record.

**PM's position** (issues added nothing the plan doesn't carry; the honest
fix is to amend the document) and **BA's position** (without issues the
label/points/role taxonomy is fiction and no queryable record exists) were
both re-argued. Neither is wrong, and the reason they deadlocked is that
each is right about a *different lifetime of record*:

- **Within-sprint tickets.** PM's evidence is decisive here. A T12 ticket's
  Story/Instructions/cross-context check/points/labels are richer than any
  issue body, version-controlled, reviewed, and linked from `HANDOFF.md`'s
  Docs index. Two sprints shipped 17 tickets this way with nothing lost.
  A parallel record nobody reads is process theater.
- **Items that outlive the sprint.** BA's evidence is decisive here, and
  this ceremony produced four fresh instances of the cost (A5's table:
  items open since T1, T5.1, T8.10 and T9.6 that have survived six sprints
  purely as prose in `HANDOFF.md` and PR bodies). Empirically, issues
  worked exactly here — #96/#97/#98/#111 were all cross-sprint follow-ups
  and all did their job. Finding 4's A3 bypass is the same gap from the
  other direction: what was missing was a *durable* record, which is what
  an issue is and what a sprint plan (scoped to one sprint) structurally
  is not.

**The resolution, split by lifetime rather than by preference:**

> 1. **The sprint plan document is the board of record for that sprint's
>    tickets.** `docs/process/t{N}-sprint-plan.md` carries the ticket's
>    Story, Description, Instructions, cross-context check, points, role and
>    labels. No GitHub issue is opened per in-sprint ticket.
> 2. **GitHub issues are the board of record for anything that outlives its
>    sprint** — cross-sprint follow-ups, disclosed-but-deferred gaps
>    (finding 4's A3 records), and escalations. These are **mandatory**:
>    an item deferred out of a sprint without an issue is a process
>    violation, not a judgement call.
> 3. **Label taxonomy follows the split.** `role:*`/`type:*` apply to
>    issues; `sprint:t<N>`/`points:*` become sprint-plan fields, since
>    they describe an in-sprint ticket and there is no longer an issue to
>    carry them.

This is genuinely both-and, but it is not a smoothing: it tells each side
it was wrong about the half it was arguing against, and it makes one thing
mandatory that was previously discretionary. T12.6 is the ticket that
amends `sprint-process.md` accordingly and cross-links A5's four issues as
the rule's first instances.

**PM and BA both sign off.** This disagreement is recorded as **resolved**,
not carried forward.

## A8. Cross-context dependency check (every cell evidence-marked, per finding 5)

| Ticket | Calls into | Member/port exists? | How this was checked |
|---|---|---|---|
| T12.1 (`vet-integration` gate) | nothing — `Makefile` + one CLAUDE.md line | n/a | Read `Makefile` lines 6–135 this ceremony; ran `go vet -tags=integration ./...` and recorded its real failure mode (A3) |
| T12.2 (`internal/platform/auth`) | nothing — new platform package; `cmd/server` registers its interceptor in observe-only mode | **No auth precedent exists in this repo** — stated as a negative finding, not papered over. Shape follows `internal/platform/grpcrecovery` (unary+stream pair) and the `port`+stub-adapter split used for Stripe | `ls internal/platform/` → `grpcrecovery, idgen, pg`; read `recovery.go` in full; `grep -rin "jwt\|auth0" --include=*.go cmd/ internal/` → comments only, no implementation |
| T12.3 (`RefundPayment` wiring) | `internal/socialplay` via the **existing** `port.RegistrationPaymentUpdater` (T6.5) — reused, not re-derived | `domain.Payment.Refund()` ✓ `payments/domain/payment.go:145`; `port.PaymentProcessor.RefundPayment` ✓ `port/payment_processor.go:59`; stub impl ✓ `adapter/stripestub/processor.go:107`; `socialplay.PaymentStatusRefunded` ✓ (T6.5 added it) | `grep -rn "RefundPayment"` across `internal/` and `proto/` — **no `app` method and no proto RPC**, confirming the gap is still open exactly as `HANDOFF.md`'s T6.5 bullet describes |
| T12.4 (`CancelGame` + HostID authz) | none — `internal/socialplay` only | `domain.Game.Cancel()` ✓ `game.go:183` but **takes no actor parameter**, so the authorization does not exist and must be added (not merely regression-tested) | Read `game.go:175-190`; `grep -rn "CancelGame"` → only `authz_regression_test.go:361-365`, a comment recording the RPC's own absence |
| T12.5 (WCAG 3.3.4 pass on T11's screens) | no new backend calls — audits three shipped Vue screens | The defect is real, not hypothetical | Read `ROUTES_UNDER_TEST` in `accessibility.spec.ts` (all three routes swept, path-identity-guarded) and `FacilityRentalRequests.vue:159-172` (one-click irreversible approve/reject, no confirm step) |
| T12.6 (board-of-record resolution) | GitHub API + `sprint-process.md` — no application code | n/a | `list_issues(state: OPEN)` → `totalCount: 0`, newest issue is #111; read `sprint-process.md`'s Ceremony 1 + label-taxonomy sections |
| T12.7 (Booking + Facilities → verified principal) | `internal/platform/auth` (T12.2) | Exists by dependency order (Wave 2 follows Wave 1) | `grep -rc actor_user_id` → booking.proto 13, facilities.proto 8 — the real surface, counted |
| T12.8 (Social Play + Payments + Competitions → verified principal) | `internal/platform/auth` (T12.2); merges after T12.3's and T12.4's proto additions | Exists by dependency order (Wave 3) | `grep -rc` → socialplay.proto 9, payments.proto 7, competitions.proto 6; proto-file overlap with T12.3/T12.4 identified and wave-separated (A12) |
| T12.9 (`CreateUser` ID from verified subject) | `internal/platform/auth` (T12.2) | Exists by dependency order (Wave 2). `identity_users.id` is a uuid PK today; the IdP subject is **not** a UUID, so a mapping column is required — migration `0019` pre-assigned (A10) | Read `HANDOFF.md`'s T10.2 bullet (states its own closure condition verbatim); `db/migrations/0016_identity.sql` for the current schema; `grep -rc actor_user_id` → identity.proto 11 |

No gap of the T9.6/T9.7 `PayableType` or T10.2 `CreateUser` class was found
hiding in this sprint's cross-context calls. The one genuinely new
dependency direction (**every context → `internal/platform/auth`**) is a
platform dependency, not a context-to-context one, so it does not create a
new arrow in the context map — checked explicitly rather than assumed,
because a shared platform package that every context imports is exactly the
shape that can quietly become a shared kernel. A11 Ruling 3 states the
constraint that keeps it from becoming one.

## A9. The four carried-forward disagreements — two resolved, one resolved-as-declined, one left open

The retro's recommendation 8 asks this ceremony to revisit rather than
"re-record the same open question a third time." Each is answered.

**(a) PE vs. PdE — polling long-running agents, in addition to the
roll-call (finding 1). Resolved as: roll-call adopted, polling declined for
T12, with a named exit condition rather than an open question.**
PdE's argument is evidence-backed and PE's is not refuted, which is why it
deadlocked. But the disagreement is *decidable by observation*, and T12 is
the observation: the roll-call (A2) is now standing practice, so this
sprint produces exactly the evidence needed. **Exit condition, recorded so
the next retro can score it without re-litigating:** if T12 has a silent
agent and the roll-call catches it, roll-call is sufficient and the
question closes. If T12 has a silent agent the roll-call *misses*, polling
is re-opened with a concrete failure to design against. If T12 has no
silent agent, it carries forward unresolved — but with one sprint of
evidence rather than none. Converting an aesthetic disagreement into a
falsifiable one is the resolution available here; picking a winner on the
current evidence is not.

**(b) QA vs. PE — does `vet-integration` need prose (a CLAUDE.md gotcha
line + a per-ticket DoD item) alongside the Makefile target (finding 2)?
Left open, deliberately, and this ceremony declines to decide it by
stealth.** T12.1 as scoped adds the gate and **corrects a factually stale
CLAUDE.md line** (it names one integration-tagged file; there are eleven,
across four contexts — verified in A3). That correction is a fix to a false
statement, and it is explicitly **not** the reminder QA argued for nor the
prose PE argued against. The honest position is that a ticket's shape
should not be used to quietly settle a disagreement the retro left open, so
the ticket says so in its own text. **Note for T12's retro:** once the gate
exists, QA's premise — that implementers skip full `make ci` and so need
prose — becomes directly checkable against T12's own PR verification lists,
which is better evidence than either side currently has.

**(c) PM vs. BA — the board of record (finding 6). Resolved.** See A7. Both
sign off; T12.6 implements it.

**(d) QA vs. PE — a lint rule or runtime guard for non-UUID fixture IDs
(T11's A11, which recommendation 8 asks to re-ask now that
`vet-integration` gives "a cheap template for a gate, not a convention").
Resolved as declined, with the reasoning the previous rounds lacked.**
Re-asked properly, and the analogy does not hold: `vet-integration` is
cheap **because the analyzer already exists** — it is one existing command
with one existing flag, and building it took a Makefile stanza. A
non-UUID-fixture-literal check has no existing analyzer; it means writing
and maintaining a custom `go/analysis` pass, deciding what a "fixture" is
syntactically, and handling legitimately non-UUID test data. That is not
the same order of cost, and T11.9 has since generalized the shared constant
block to all six contexts, so the demonstrated instances are covered.
**Declined for T12 with a stated trigger:** if a sixth instance appears in a
file the shared constant blocks do not cover, the enforcement half is taken
without further debate. This is a decision, not a fifth deferral.

**(e) Carried forward from T10's retro, now scored — is fresh-worktree
re-verification mandatory? Resolved, narrowly.** Recommendation 8 supplies
the evidence: the reviewer applied it voluntarily on four of nine T11 PRs
(#116, #119, #120, #121) and it is what caught finding 2's break. A blanket
mandate remains disproportionate for, say, a docs-only PR. **T12's rule:**
fresh-worktree re-verification off the pushed branch is **required** for any
PR whose diff touches (i) a `//go:build integration` file, or (ii) a file
A12's shared-append table names as claimed by more than one ticket.
Everywhere else it stays a judgement call. That is evidence-scoped to
exactly the two classes where it has demonstrably paid, and it closes a
question that has been re-recorded twice.

## A10. Migration-number and shared-file pre-assignment (finding 3 + T11's A5, which worked)

T11's A5 was, per its own retro, "this project's first process fix to fully
prevent its own incident class." It is applied again, unchanged in spirit.

**Migration tip verified by listing the directory this ceremony, not
inferred from the last plan's count:** `db/migrations/` ends at
`0018_booking_recurring_hire_templates.sql`. (The directory still carries
T6's `0005_payments.sql` **and** `0005_socialplay.sql` scar, and T10's
duplicate-`0015` near-miss is in the retro record — this is what the
pre-assignment prevents.)

**Exactly one T12 ticket adds a migration:**

| Migration | Ticket | Purpose |
|---|---|---|
| `0019_identity_subject.sql` | **T12.9** | `identity_users.subject text UNIQUE` — maps the IdP's verified subject claim to the user row, so `User.ID` stops being caller-supplied |

**Checked-and-does-not-fire, stated rather than left to a reader's
inference:** T12.3 (`RefundPayment`) needs **no** migration —
`db/migrations/0005_payments.sql:30` already declares
`status text NOT NULL DEFAULT 'unpaid' CHECK (status IN ('unpaid','paid','refunded'))`,
so the refunded state is already schema-legal. Verified by reading the
constraint, not by assuming T6 built it. T12.2/T12.7/T12.8 add no tables:
the principal comes from the token, not from storage.

## A11. Three PE rulings on where new code lives (stated once, not re-litigated per ticket)

**Ruling 1 — auth lives in `internal/platform/auth`, not in a seventh
bounded context and not inside `internal/identity`.** Authentication is a
cross-cutting platform concern every context consumes; Identity/Users is a
*domain* context that owns the `User` aggregate. Folding token verification
into `internal/identity` would make five contexts import a domain context
to authenticate, inverting the dependency rule (CLAUDE.md rule 3) and
recreating the `games.facility_id` mistake in a new place — a
cross-cutting concern parked in the wrong context. `internal/platform`
already holds exactly this class (`grpcrecovery`, `idgen`, `pg`), verified
by listing it. **T12.9 is where auth and Identity meet**, and it meets them
at the seam: Identity maps a verified subject to a `User`; it does not
verify tokens.

**Ruling 2 — the authenticated-method policy is per-context, not one list
in `main.go`.** Each context's `grpcapi` package exports its own
`AuthenticatedMethods()`; `cmd/server` composes them. Two reasons, and the
second is the one that decides it: (i) the knowledge of which of *this
context's* RPCs are public belongs with that context's handlers, next to
the code that would break if it were wrong; (ii) it converts A4's
four-ticket shared append target into three per-context files plus one
one-line composition edit — finding 3's collision class designed out at
planning time rather than predicted and hand-resolved at
`mergeable_state: dirty`.

**Ruling 3 — `auth.Principal` is a platform type, and contexts must not
grow a shared-kernel dependency on it.** Handlers translate a `Principal`
into whatever their own domain already understands (an owner ID string, an
actor ID) **at the grpcapi boundary**; `domain` and `app` packages keep
their existing signatures and must not import `internal/platform/auth`.
This preserves CLAUDE.md rule 2 (a pure domain), keeps every existing
domain-level authorization test valid unchanged, and is the same
shared-kernel discipline T9's A4 ruling applied to `Money` and T11's A9
applied to `EndCondition`. **Concretely: the migration tickets change
handlers, not domain rules** — which is also why they are 8 points and not
13.

## A12. Dispatch isolation, waves, and shared *existing* append targets (finding 3 applied)

**Wave 1 — up to 5 implementers in parallel, each on its own isolated git
worktree (`isolation: "worktree"`).**
T12.1 (`Makefile` + one CLAUDE.md line), T12.2 (new `internal/platform/auth`
package + `cmd/server/main.go` interceptor registration), T12.3
(`internal/payments/**` + `payments.proto`), T12.4 (`internal/socialplay/**`
+ `socialplay.proto`), T12.5 (`web/` only).

**Wave 2 — up to 3 implementers in parallel, worktree-isolated.** T12.6
(docs + GitHub API only), T12.7 (needs T12.2), T12.9 (needs T12.2;
migration `0019`).

**Wave 3 — 1 implementer.** T12.8 (needs T12.2 for the platform, **and**
T12.3 + T12.4 merged, because it edits the same two proto files those
tickets add RPCs to).

**Shared *existing* files, and who appends to each** — the second question
finding 3 requires, asked of existing files rather than new ones:

| Existing file | Tickets appending | Same wave? | Rule stated up front |
|---|---|---|---|
| `cmd/server/main.go` | T12.2 (W1), T12.7 + T12.9 (W2), T12.8 (W3) | **Yes — T12.7/T12.9** | T12.2 lands first and creates the seam. T12.7 and T12.9 each add **one line** composing their context's `AuthenticatedMethods()` (Ruling 2 keeps it to one line). Whichever merges second resolves on its **own source branch**, keeping both entries, with a one-line reason in that PR body. Per A9(e), that PR also requires fresh-worktree re-verification. |
| `proto/pickleball/socialplay/v1/socialplay.proto` | T12.4 (W1), T12.8 (W3) | No — two waves apart | Wave gap is the mitigation; T12.8 branches from a base that already contains `CancelGame`. Flagged rather than assumed, per T11's A10 hedged-prediction precedent. |
| `proto/pickleball/payments/v1/payments.proto` | T12.3 (W1), T12.8 (W3) | No — two waves apart | Same as above. |
| `internal/socialplay/domain/errors.go` | **T12.4 only** | n/a | Checked, does not fire. T12.8 re-plumbs actor identity at the handler boundary (Ruling 3) and adds no domain sentinel. |
| `internal/payments/domain/errors.go` | **T12.3 only** | n/a | Checked, does not fire — same reason. |
| `internal/booking/domain/errors.go` | **nobody** | n/a | The file finding 3 was written about. Checked explicitly, per the retro's requirement that it be named: no T12 ticket adds a Booking domain sentinel, because T12 adds no Booking domain rule. |
| `Makefile` | T12.1 only | n/a | |
| `docs/process/sprint-process.md` | T12.6 only | n/a | |
| `web/src/__tests__/accessibility.spec.ts` | T12.5 only | n/a | |

**No two tickets in the same wave create the same new file**, and the one
same-wave existing-file overlap (T12.7/T12.9 on `main.go`) is named above
with its resolution rule. That is both questions asked, which is what
finding 3 asked for.

## A13. ADR-0012's Q1/Q2 remain blocked — named, not touched, not dropped

`docs/adr/0012-*.md` blocks `PlayerRating`, the matching algorithm, and
gender-mix matching on two questions escalated to the user: **Q1** (Player
Level formula weighting) and **Q2** (whether gender-mix matching is in
scope at all, given it means collecting and algorithmically acting on a
protected attribute). Its trigger is explicit: *"the sprint immediately
following the user's answers to both Q1 and Q2"* — the user answering, not
a sprint boundary.

**Checked this ceremony: no answer to either question exists.** T12
therefore builds none of it, and this ceremony makes no product call on
either — they are not this team's to make. Recorded exactly as T10's and
T11's plans recorded it, so it is neither silently dropped nor silently
decided. ADR-0012's constraint that *"no PR may add a `Gender` field, a
`PlayerRating` type, or any Level-scoring formula"* binds every T12 ticket
unchanged.

One genuine interaction, stated because it would otherwise be discovered
mid-ticket: T12.9 changes how `User` rows are created. It must **not** take
the opportunity to add profile fields — `subject` is the only column added
(A10), and `SelfReportedStartingLevel` keeps its existing shape and meaning.

## A14. Recorded disagreements (not manufactured consensus)

**PdE vs. PE — is a 46-point sprint whose spine is a security-critical
platform change the right shape, when three of its nine tickets cannot
start until that platform lands?**

- **PdE:** the dependency shape is narrower than T11's 4-deep chain (this
  one is 2 deep: T12.2 → {T12.7, T12.9} → T12.8), but the *blast radius* is
  wider than anything this project has attempted. T12.2 gets it wrong and
  three tickets inherit the error across all six contexts simultaneously,
  versus T11 where a bad `FacilityLookup` would have affected two tickets
  in one context. The mitigation PdE would prefer is a Wave-1.5 checkpoint:
  T12.7 (two contexts, the smaller surface) merges and is reviewed
  *before* T12.8 and T12.9 are dispatched at all, converting the parallel
  Wave 2 into a deliberate proving run.
- **PE:** that is real risk, but the proposed mitigation buys less than it
  costs. T12.2's own AC requires the interceptor to ship in **observe-only
  mode** (it populates the principal and enforces nothing) precisely so
  that the platform is exercised end to end before any handler depends on
  it for correctness — the proving run is inside T12.2, not after T12.7.
  Serializing Wave 2 also puts T12.9 — the ticket closing a live,
  disclosed DoS vector — behind two other merges for schedule reasons
  rather than technical ones.
- **Unresolved**, recorded rather than smoothed. Both agree the wave
  structure in A12 is correct *if* T12.2's observe-only requirement holds;
  the disagreement is about whether an observe-only mode is as strong a
  proving mechanism as a merged, reviewed consumer. **PdE's concern is
  scoreable at T12's retro** — if T12.7/T12.8/T12.9 each independently hit
  the same T12.2 defect, PdE was right.

**QA vs. PM — is T12.5 (3 points) enough for the WCAG gap, or does T11.7's
whole manual half need re-running against the three new screens?**

- **QA:** A6 found the defect by checking *one* criterion on *one* screen.
  The manual half of T11.7 — keyboard-only traversal, screen-reader spot
  check, 3.3.1/3.3.3 confirmation — never ran against any of the three, so
  the honest scope is a full manual pass, and 3 points does not buy one.
  Finding one defect where the audit never looked is weak evidence that
  there is exactly one.
- **PM:** the automated sweep *did* cover all three screens and is
  path-identity-guarded (A6), and the screens demonstrably carry the error
  identification and live-region patterns the audit checks for — so the
  unchecked surface is the manual-only criteria, not everything. A targeted
  3.3.4 + keyboard pass is proportionate; a full re-audit of three screens
  is most of T11.7's 8 points again for a bar those screens were built to.
- **Unresolved.** T12.5 as ticketed takes PM's narrower scope **plus** an
  explicit instruction to state, as a checked negative, which criteria it
  confirmed clean and which it did not reach — so if QA is right, the next
  ceremony inherits a named gap rather than a silent one. That instruction
  is the part both roles agree on.

**BA note, not a disagreement.** Ran the standing contradiction check
against every locked decision T12 touches. No contradiction found. Real
auth is listed in `HANDOFF.md`'s Cross-cutting section as long-planned, not
as a locked-decision reversal; retiring `actor_user_id` from request DTOs
does not alter the ubiquitous language (an *Actor* remains an Actor — only
its provenance changes, from claimed to verified), which is CLAUDE.md rule
7's actual requirement. One naming note carried into T12.2's text: the
verified thing is a **Principal**, and it is not a synonym for `User` —
`User` stays the Identity context's aggregate, `Principal` is the platform's
verified-caller value. Introducing two words for one concept would be the
rule-7 violation; these are two concepts.

---

# Part B — Ceremony 2 (sprint planning)

## Sprint goal

> Every authorization check in this codebase stops trusting a
> caller-supplied `actor_user_id`/`actor_player_id` and starts trusting a
> **verified principal** derived from a signed token — closing, in
> particular, the `CreateUser` identity-squatting denial-of-service
> `HANDOFF.md` has carried as a disclosed security gap since T10; the two
> oldest well-specified authorization follow-ups (`RefundPayment` since T6,
> `CancelGame` since T5.5) are finally built rather than deferred a seventh
> time; the build-tag blind spot that broke the same file twice in T11
> becomes a gate `make ci` actually runs; and the board-of-record question
> two sprints let lapse is decided rather than re-recorded.

## In-scope tickets

### T12.1 — Chore: add a Docker-free `vet-integration` gate and wire it into `make ci`

**Story:** As an implementer, I want the `//go:build integration` files to
be compiled by a gate I can actually run, so that I stop shipping a break
that every command available to me reports as green.

**Description:** Per T11 retro finding 2, the retro's own #1 recommendation.
`internal/booking/adapter/postgres/concurrency_integration_test.go` broke
twice in T11 for exactly this reason. No dependency on any other T12
ticket — can start immediately.

**Instructions:**
1. Add a `vet-integration` target running `go vet -tags=integration ./...`.
   **It must depend on `generate`** — verified this ceremony: on a clean
   checkout the bare command fails with `no required module provides package
   .../internal/gen/...` because `internal/gen/**` is gitignored (CLAUDE.md
   Gotchas). Do not "fix" that by dropping the tag or narrowing the package
   pattern; the dependency is the fix.
2. Add it to `make ci`'s target list, **after `test-domain`** (the retro's
   placement; `generate` already runs first there, so ordering is
   consistent).
3. Update `make ci`'s closing message, which currently implies
   integration-tagged code is entirely uncovered locally. After this ticket
   it is *compiled* by `make ci` and only *executed* by `make ci-integration`
   — say exactly that.
4. **Factual correction to CLAUDE.md's Gotchas section:** it names a single
   integration-tagged file. `grep -rl "go:build integration"` finds **11
   files across 4 contexts** (socialplay 5, payments 3, competitions 2,
   booking 1). Correct the count and name the new target as the gate that
   compiles them. **State in the PR that this is a correction to a false
   statement and explicitly NOT the CLAUDE.md/DoD reminder QA argued for in
   the retro's recorded disagreement** — that disagreement stays open per
   A9(b), and this ticket must not be read as settling it.
5. **Prove the gate is not vacuous** (CLAUDE.md rule 10): temporarily break
   an integration-tagged call site (e.g. add an argument to a constructor
   one of the 11 files calls), confirm `make vet-integration` fails, restore,
   confirm green. Record the exact command output in the PR — a gate nobody
   watched fail is an assumption, not a proof.
6. Non-functional: no Docker. If any step needs a daemon, the step is wrong.

**Cross-context dependency check:** none — `Makefile` + one CLAUDE.md line
(A8).

**Story points:** 2. **Role:** qa. **Labels:** `role:qa`, `type:chore`,
`points:2`.

---

### T12.2 — `internal/platform/auth`: verified principal, token verification, and an observe-only gRPC interceptor (+ ADR-0013)

**Story:** As the platform, I want a caller's identity to be *verified*
rather than *claimed*, so that every object-level authorization check this
codebase already has stops resting on a string the caller chose.

**Description:** The spine of this sprint. Builds the platform capability
only — **no context is migrated in this ticket** (T12.7/T12.8/T12.9 do
that). Per A11 Ruling 1 this lives in `internal/platform/auth`, mirroring
`internal/platform/grpcrecovery`'s shape (verified present; read this
ceremony). **There is no JWT/OIDC precedent anywhere in this repo** —
verified by grep, stated as a negative finding: do not go looking for an
auth pattern to mirror, there isn't one. No dependency on any other T12
ticket.

**Instructions:**
1. `auth.Principal` — the verified-caller value: `Subject` (the IdP's `sub`
   claim, the only field that is authoritative), plus whatever claims the
   verifier is configured to surface. Per A14's naming note, `Principal` is
   **not** a synonym for Identity's `User` aggregate; its doc comment must
   say so, since the two will sit next to each other in T12.9.
2. `port`-style `TokenVerifier` interface + a real RS256/JWKS-backed
   implementation. Follow the split this project already uses for Stripe
   (`port.PaymentProcessor` + `adapter/stripestub`, verified paths): the
   interface is the seam, the implementation is replaceable. Verification
   must check signature, `exp`, `nbf`, `iss` and `aud` — a verifier that
   only checks the signature is a worse lie than no verifier, because it
   looks like one.
3. `auth.UnaryInterceptor` / `auth.StreamInterceptor`, mirroring
   `grpcrecovery`'s pair exactly (including registering **both** — see
   `recovery.go:96`'s own comment on why registering only the unary form is
   a trap). Put the `Principal` in the context; provide
   `auth.PrincipalFromContext`.
4. **Observe-only by default, and this is an AC, not a preference.** In this
   ticket the interceptor populates the context and **enforces nothing**: a
   request with no token, or an invalid one, proceeds exactly as it does
   today. Enforcement is turned on per-RPC by T12.7/T12.8/T12.9 via each
   context's `AuthenticatedMethods()` (A11 Ruling 2). Rationale, to be
   restated in the PR: this makes the platform exercisable end to end
   before any handler's correctness depends on it, and it means this
   ticket cannot break a single shipped flow.
5. Register the interceptor in `cmd/server/main.go`, into the **existing**
   `ChainUnaryInterceptor`/`ChainStreamInterceptor` calls (lines 238–240),
   after the recovery interceptor — whose own comment says it "stays first
   in the chain" so that it covers the others. Do not restructure the chain.
6. Tests: table-driven, with a **locally-minted RSA keypair and an
   in-process key source** — no network, no Docker, no external tenant.
   Cover at minimum: valid token → principal present; expired, wrong
   issuer, wrong audience, bad signature, malformed, absent → no principal
   and (in observe-only mode) no error; and that a panic in the verifier is
   still caught by the recovery interceptor sitting in front of it.
7. **ADR-0013** (next number — `docs/adr/` verified to end at `0012` this
   ceremony). Record: why auth is platform and not a context (A11 Ruling 1);
   the observe-only → per-context-enforcement migration path; that
   `Principal` is not `User`; and — with the same honesty `HANDOFF.md`
   applies to Jenkins — that **provisioning a real IdP tenant is
   server-side work no coding session in this project can perform**, so
   "auth exists" means "verification and enforcement exist and are tested
   against local key material," not "a production IdP is wired up."
8. Non-functional: `internal/platform/auth` imports no context (dependency
   rule, CLAUDE.md rule 3). Error handling — **gRPC codes only**: once
   enforcement is on (later tickets), a missing/invalid token is
   `Unauthenticated`; a valid token whose principal fails an object-level
   check stays `PermissionDenied`. Getting these two confused is the
   classic auth bug; state the distinction in the ADR.
9. **Standing rule (A5):** any gap this ticket discloses and does not close
   gets a GitHub issue in the same PR, not a paragraph in the PR body.

**Cross-context dependency check:** none — new platform package; no auth
precedent exists to mirror, stated as a checked negative (A8).

**Story points:** 8. **Role:** principal-engineer. **Labels:**
`role:principal-engineer`, `type:story`, `points:8`.

---

### T12.3 — Wire `Service.RefundPayment` and push `PaymentStatusRefunded` through

**Story:** As a Host, I want to refund a payment I took, so that a
cancellation is a real financial outcome rather than a status nobody can
reach.

**Description:** Open since T6.5 — `HANDOFF.md`'s own bullet already
specifies it in full and T11's plan named it "a real candidate for T12's
Ceremony 1 to pick up first." Verified this ceremony: `domain.Payment.Refund()`
(`payment.go:145`) and `port.PaymentProcessor.RefundPayment`
(`payment_processor.go:59`, stub impl at `stripestub/processor.go:107`)
both exist; **`grep -rn "RefundPayment"` finds no `app` method and no proto
RPC**, so the gap is exactly as described. **No migration needed** —
verified, `0005_payments.sql:30`'s CHECK already allows `'refunded'` (A10).
No dependency on any other T12 ticket.

**Instructions:**
1. `app.Service.RefundPayment`: online path calls
   `PaymentProcessor.RefundPayment` with the stored intent reference; offline
   path is a Host/Game-Admin action. Reuse the **existing**
   `authorizeOfflineRecording`/`ErrNotPaymentRecorder` authorization
   (`internal/payments/domain/errors.go:54`, verified) — do not invent a
   second authorization concept for refunds.
2. Drive `domain.Payment.Refund()`'s existing state machine; do not
   re-implement the transition. An already-refunded or never-paid Payment is
   rejected by the domain, not by a new app-level check.
3. On success, push `PaymentStatusRefunded` through the **existing**
   `socialplayport.RegistrationPaymentUpdater` (T6.5) — the port and Social
   Play's `refunded` value were both built for this call site and have had
   no caller since T6. Do not add a second updater path.
4. `payments.proto` gains `RefundPayment`. **CLAUDE.md rule 11 (PCI) applies
   directly to this ticket** — the request carries a payment/payable
   identifier and an actor, and **must never carry a PAN, card number, CVV
   or track data**. Review the proto change against
   `docs/checklists/proto-review.md` (verified present) before opening the
   PR, and say in the PR that you did.
5. **Scope, stated rather than left implicit:** `registration`- and
   `booking`-payable refunds only. Competitions remains cash-only (its
   `PayableType` gap is issue #125, A5) — state that as a checked negative,
   not a silent omission.
6. Error handling — **gRPC codes only:** non-recorder actor →
   `PermissionDenied`; unknown/malformed payment ID → `NotFound` (reuse the
   existing UUID-shape helper from PR #89's Layer 2, do not re-derive it);
   illegal state transition → `FailedPrecondition`; processor failure →
   `Internal` (and the Payment must **not** be marked refunded — assert
   this).
7. Non-functional: TDD-first; a refund that succeeds at the processor but
   fails to persist must not silently report success — test that path
   explicitly.
8. **Standing rule (A5):** disclosed-but-not-closed gaps get a GitHub issue
   in the same PR.

**Cross-context dependency check:** reuses `socialplayport.RegistrationPaymentUpdater`
(existing since T6.5, verified) — no new cross-context port (A8).

**Story points:** 5. **Role:** principal-engineer. **Labels:**
`role:principal-engineer`, `type:story`, `points:5`.

---

### T12.4 — `CancelGame` with HostID-scoped authorization + handler-level regression test

**Story:** As a Game's Host, I want to cancel my Game through the API, and
as any other player, I want to be unable to cancel someone else's, so that
Game cancellation is a real, authorized operation instead of a domain
method with no way to reach it.

**Description:** Open since T5.5, which split it out with reasoning and
`HANDOFF.md` has carried ever since. Verified this ceremony:
`domain.Game.Cancel()` exists (`game.go:183`) but **takes no actor
parameter at all** — so unlike T5.5/T6.7, this ticket must *add* the
authorization, not merely regression-test an existing one. `grep -rn
"CancelGame"` finds only `authz_regression_test.go:361-365`, a comment
recording the RPC's own absence. No dependency on any other T12 ticket.

**Instructions:**
1. Add the ownership check at the domain level: `Game.EnsureHost(actorPlayerID)`
   (or `Cancel(actorPlayerID)`), returning a new `ErrNotGameHost` sentinel
   appended to `internal/socialplay/domain/errors.go`. Mirror the existing
   `ErrNotGameHostOrAdmin` (`errors.go:179`, verified real) — **note it is
   deliberately a different sentinel**: recording a match result allows
   Host *or* Game Admin, cancelling the Game is Host-only. State that
   distinction in the sentinel's doc comment so a future reader doesn't
   "unify" them.
2. `app.Service.CancelGame` + `socialplay.proto` `CancelGame` RPC + handler.
3. **The required proof:** a handler-level authorization regression test in
   the existing `internal/socialplay/adapter/grpcapi/authz_regression_test.go`
   — the file that currently documents this RPC's absence at lines 361–365.
   Delete that comment and replace it with the test it was waiting for.
   Follow T5.5/T7.7's shape: real `grpcapi.Handler` → `app.Service` →
   `domain.Game` against in-memory fakes (no Docker; the check has no SQL in
   it, so a real DB would add infrastructure, not proof).
4. **Verify non-vacuously** (CLAUDE.md rule 10): temporarily disable the
   ownership check, confirm the regression test fails, restore, confirm
   green. Record it in the PR.
5. **Disclosed gap that must be tracked, not just mentioned (A5):**
   cancelling a Game does **not** cascade to its court Bookings or its
   Registrations — `Game.Cancel()`'s own doc comment says so and T5.1
   deferred it. This ticket does not build the cascade (it is a real
   product-semantics question: are reserved courts freed, are players
   refunded — which now touches T12.3). **Issue #124 already tracks it
   (opened this ceremony, A5); link it from this PR** rather than opening a
   duplicate or re-disclosing it in prose.
6. Error handling — **gRPC codes only:** non-Host actor → `PermissionDenied`;
   unknown/malformed game ID → `NotFound`; already-cancelled Game →
   `FailedPrecondition` (the domain's existing `ErrIllegalStatusTransition`
   already produces this — check the existing mapping before adding one).
7. **Caveat to restate, not re-litigate:** until T12.8 migrates this
   context, the actor is still a *claimed* `actor_player_id`. This ticket
   builds the object-level check; T12.8 makes the actor verified. Say so in
   the PR — the same caveat `HANDOFF.md` has carried for four contexts,
   with the difference that this sprint actually closes it.

**Cross-context dependency check:** none — `internal/socialplay` only (A8).

**Story points:** 5. **Role:** principal-engineer. **Labels:**
`role:principal-engineer`, `type:story`, `points:5`.

---

### T12.5 — WCAG 2.2 AA: error prevention (3.3.4) and keyboard pass on T11's three new screens

**Story:** As a Facility Owner using assistive technology or a keyboard, I
want an irreversible approval that creates a season of Bookings to ask me
first, so that the most consequential action on the newest screen isn't
also the easiest one to trigger by accident.

**Description:** T11.7's audit ran in Wave 1; T11.3 and T11.6 merged in
Waves 3 and 4, so its **manual** half — keyboard-only traversal,
screen-reader spot check, and the 3.3.1/3.3.3/3.3.4 confirmations — never
covered the three screens they shipped. The automated half **did** (verified
this ceremony: all three routes are in `ROUTES_UNDER_TEST` and the sweep is
path-identity-guarded against the real router, so a new route cannot escape
it — T11's A6 claim holds and is structurally enforced). **A concrete defect
was found in the unchecked half**, which is why this is a ticket and not a
spike. No dependency on any other T12 ticket — `web/` only.

**Instructions:**
1. **The verified defect, fix it first.**
   `web/src/views/FacilityRentalRequests.vue:159-172` renders "Approve
   request" and "Reject request" as single-click, one-way transitions with
   no confirmation step and no undo window. Approval generates real
   Bookings across every generated occurrence (the file's own header, lines
   16–31, confirms both the Booking generation and that both transitions are
   terminal). That is WCAG 2.2 AA **3.3.4 Error Prevention
   (Legal/Financial/Data)** — the criterion T11.7's own scope named as
   load-bearing. Add a confirm-before-commit step (or an undo window),
   following the pattern the flagship flows already use for this criterion
   rather than inventing a new one — `web/src/__tests__/flagshipFlows.spec.ts`
   has a "review/confirm step (WCAG 3.3.4)" case for `CourtBookingFlow`;
   read it and extend the convention.
2. **Keyboard-only pass** over all three screens
   (`/facilities/:facilityId/discounts`, `/facilities/:facilityId/rental-requests`,
   `/clubs/rentals`): every control reachable and operable, visible focus,
   no traps, sensible order. Fix what's found.
3. **3.3.1 / 3.3.3 confirmation** on the two forms (discount create, rental
   request): invalid input identified in text and not by colour alone, and
   a correction *suggestion*, not just "invalid." Spot-checked this
   ceremony as likely-clean (`FacilityDiscounts.vue` carries `role="alert"`
   + `aria-describedby` on every field) — **confirm it rather than inherit
   the spot check**, and say which you confirmed.
4. **Every fix gets a regression test, and the assertion is specified as a
   property, not a signal list** (T11 retro finding 5, recommendation 7):
   the check must find the control under **every identification shape a
   reasonable implementation could use**, and must be **mutation-checked
   against the actual control this ticket ships** — force the gate open and
   confirm the test fails. Reuse
   `web/src/test-support/semanticControlAssertions.ts`'s
   `findControlsMatching(wrapper, pattern)` (verified present — T11.6
   extracted it precisely so a second caller wouldn't re-implement the
   scan); do not add a screen-local reimplementation.
5. **State the checked negatives explicitly** (this is the part QA and PM
   both agreed on, A14): list which WCAG criteria this ticket confirmed
   clean on these three screens and **which it did not reach**. If the
   honest answer is that a full manual pass is still owed, say so and open
   a GitHub issue for it per A5 — do not let a targeted pass be mistaken
   for a complete one.

**Cross-context dependency check:** none — audits three shipped Vue screens,
no new backend calls (A8).

**Story points:** 3. **Role:** ux-ui-designer. **Labels:**
`role:ux-ui-designer`, `type:bug`, `points:3`.

---

### T12.6 — Chore: make the board of record match reality (amend `sprint-process.md`)

**Story:** As the team, I want the process document to describe the board of
record we actually use, so that "the ticket is the GitHub issue" stops
being a sentence two sprints have quietly contradicted.

**Description:** Implements A7's resolution of T11 retro finding 6's
PM-vs-BA disagreement. Verified this ceremony: `list_issues(state: OPEN)`
returns `totalCount: 0` and the newest issue in the repository is #111,
from T11's *Ceremony 1* — there is no issue for any of T11's nine tickets.
Depends on nothing technically; scheduled in Wave 2 so it can also record
whatever Wave 1 discloses.

**Instructions:**
1. Amend `docs/process/sprint-process.md` to state the resolved split
   (A7), in its Ceremony 1 section and its label-taxonomy section:
   - the **sprint plan document** is the board of record for that sprint's
     tickets (replacing "the ticket **is** the GitHub issue");
   - **GitHub issues are mandatory** for anything outliving its sprint —
     cross-sprint follow-ups, disclosed-but-deferred gaps, escalations. An
     item deferred out of a sprint without an issue is a process violation,
     not a judgement call;
   - `role:*`/`type:*` label issues; `sprint:t<N>`/`points:*` become
     sprint-plan fields.
2. Update the per-ticket DoD's step 5, which currently requires closing a
   GitHub issue after every merge — under the resolved rule most tickets
   have no issue to close. Keep the manual-close requirement **for tickets
   that do close an issue** (T11.8's finding that auto-close structurally
   cannot fire on this branch topology is unaffected and must not be lost).
3. Cross-link the four issues opened at this ceremony (#123 `ServiceOptions`,
   #124 Game-cancellation cascade, #125 Competitions `PayableType`, #126
   real per-Game price) as the rule's first instances, so the rule and its
   worked examples land together.
4. Record in the PR that this **resolves** a disagreement the T11 retro
   left open, naming both positions and why the split is not a smoothing
   (A7) — so a future retro can score the decision rather than rediscover
   the debate.
5. **Do not retroactively backfill issues for T11.1–T11.9.** Under the
   resolved rule they were never owed: the sprint plan is their board of
   record. State this explicitly so it is a decision, not an omission.

**Cross-context dependency check:** none — one doc edit + GitHub API (A8).

**Story points:** 2. **Role:** business-analyst. **Labels:**
`role:business-analyst`, `type:chore`, `points:2`.

---

### T12.7 — Migrate Booking + Facilities to the verified principal

**Story:** As a Facility Owner, I want the platform to know it's me because
I authenticated, not because I typed my own user ID into the request, so
that "only the owner may do this" is an actual boundary.

**Description:** Depends on T12.2. The first two contexts to move, chosen as
the smaller, best-tested surface (booking.proto 13 + facilities.proto 8
claimed-actor occurrences, counted this ceremony). Per A11 Ruling 3 this is
a **handler-boundary change**: `domain` and `app` signatures do not change
and must not import `internal/platform/auth`, which is why every existing
domain-level authorization test stays valid unchanged.

**Instructions:**
1. Handlers resolve the actor via `auth.PrincipalFromContext` instead of
   reading it off the request message, and translate the `Principal` into
   the actor value the `app`/`domain` layer already expects (Ruling 3).
2. Export `AuthenticatedMethods()` from each context's `grpcapi` package
   (A11 Ruling 2) listing the RPCs that require a principal, and compose it
   in `cmd/server/main.go` — **one line**. Public reads
   (`ListFacilities`, `GetQuote`, `ListCourtBookings`, …) stay public;
   **decide this per RPC and state the list in the PR**, because silently
   authenticating a public browse path would break a shipped flow.
3. Deprecate the `actor_user_id` fields on the affected request messages;
   do not delete them in this ticket (a removed proto field is a client
   break, and `web/` still sends them until its own follow-up). The handler
   must **ignore** the wire field entirely once enforcement is on — a
   handler that falls back to the claimed value when no principal is
   present has changed nothing. Test that specific fallback is absent.
4. Tests: extend the existing `authz_regression_test.go` files in both
   contexts (verified present for `facilities`) to prove (a) a valid
   principal for the owner succeeds, (b) a valid principal for a
   *different* user is `PermissionDenied`, (c) **no principal at all is
   `Unauthenticated`, not `PermissionDenied`**, and (d) a request whose
   wire `actor_user_id` claims the owner while the principal says otherwise
   is rejected — that last one is the whole point of the sprint and must be
   asserted, not assumed.
5. **Reminder (A12):** this PR and T12.9's both append one line to
   `cmd/server/main.go`. Whichever merges second resolves on its **own
   source branch**, keeping both entries, with a one-line reason in the PR
   body — and per A9(e) that PR requires **fresh-worktree re-verification**
   off the pushed branch, not just a working-tree run.
6. Error handling — **gRPC codes only**, per T12.2's ADR: missing/invalid
   token → `Unauthenticated`; authenticated but not the owner →
   `PermissionDenied`.

**Cross-context dependency check:** `internal/platform/auth` (T12.2, exists
by dependency order). No new context-to-context call (A8).

**Story points:** 8. **Role:** principal-engineer. **Labels:**
`role:principal-engineer`, `type:story`, `points:8`.

---

### T12.8 — Migrate Social Play + Payments + Competitions to the verified principal

**Story:** As a Player, I want to be unable to cancel someone else's
registration, refund someone else's payment, or manage someone else's
competition even if I know their user ID, so that the three-times-repeated
"claimed actor, not verified identity" caveat is finally false.

**Description:** Depends on T12.2 for the platform, **and on T12.3 and T12.4
being merged**, because it edits the same two proto files those tickets add
RPCs to (`payments.proto`, `socialplay.proto` — A12's shared-file table).
Scheduled alone in Wave 3 for that reason. Same Ruling 3 constraint as
T12.7: handler-boundary only.

**Instructions:**
1. Same migration as T12.7, applied to `socialplay` (9 occurrences),
   `payments` (7) and `competitions` (6): principal from context, wire
   field deprecated and ignored, `AuthenticatedMethods()` per context, one
   composition line in `main.go`.
2. **This is where `actor_player_id` and `actor_user_id` meet.** Social Play
   uses `actor_player_id`; the others use `actor_user_id`. Decide and state
   whether a Principal's subject maps to both, or whether Social Play needs
   an explicit Player↔User resolution step — **check the existing
   relationship in `internal/socialplay` and `internal/identity` before
   deciding, and record the finding either way.** If it turns out these are
   genuinely two identifier spaces, that is a real design finding: disclose
   it, and per A5 open an issue rather than inventing a mapping mid-ticket.
3. Extend the three contexts' existing `authz_regression_test.go` files
   with the same four assertions T12.7 item 4 requires, including the
   principal-overrides-wire-claim case.
4. **Include T12.4's brand-new `CancelGame`** in Social Play's
   `AuthenticatedMethods()` — it lands claimed-actor in Wave 1 and this is
   the ticket that makes it verified. Named explicitly so the newest RPC
   isn't the one that gets missed.
5. **Reminder (A12):** T12.3's and T12.4's proto additions are already
   merged when this branches, so no conflict is expected — but this PR
   touches two files another ticket edited this sprint, so per A9(e)
   fresh-worktree re-verification off the pushed branch is **required**.
6. Error handling — **gRPC codes only**, same mapping as T12.7.

**Cross-context dependency check:** `internal/platform/auth` (T12.2). Proto
overlap with T12.3/T12.4 resolved by wave separation, not by hope (A12).

**Story points:** 8. **Role:** principal-engineer. **Labels:**
`role:principal-engineer`, `type:story`, `points:8`.

---

### T12.9 — Close the `CreateUser` identity-squatting DoS: mint `User.ID` from the verified subject

**Story:** As a real person who has not signed up yet, I want it to be
impossible for a stranger to permanently occupy the account my identity
will map to, so that I can actually claim my own account when I arrive.

**Description:** Depends on T12.2. **This closes a real, disclosed security
gap**, described in `HANDOFF.md`'s T10.2 bullet as "a persistent, targeted
denial-of-service, not a rejected mutation," with a closure condition that
bullet states in its own words: *"Must close the moment real auth exists"*
— mint `User.ID` from the authenticated principal's verified subject claim
rather than accepting it as a bare client-supplied field. Migration number
pre-assigned: **`0019_identity_subject.sql`** (A10; tip verified at `0018`
by listing the directory this ceremony).

**Instructions:**
1. Migration `0019_identity_subject.sql`: add `identity_users.subject text
   UNIQUE`. The IdP subject is **not** a UUID (e.g. `auth0|abc123`), so it
   cannot be `identity_users.id` — the row keeps its uuid PK and gains a
   unique mapping column. State this reasoning in the migration file; a
   future reader will otherwise ask why there are two identifiers.
2. `CreateUser` no longer accepts a caller-supplied ID. The `User.ID` is
   **server-minted** (`ids.NewID()`, this codebase's universal pattern) and
   the row is keyed to the principal's verified `subject`. A second
   `CreateUser` for an already-registered subject is idempotent-or-rejected
   — pick one, state which and why.
3. `CreateUser` requires a principal (add it to `identity`'s
   `AuthenticatedMethods()`). An anonymous caller can no longer create a
   user at all, which is what removes the squatting surface entirely rather
   than narrowing it the way T10.2's role restriction did.
4. **Prove the gap is closed, don't assert it** (CLAUDE.md rule 10): a
   regression test that reproduces the original attack — a caller supplying
   a chosen UUID (or a subject that isn't theirs) — and asserts it now
   fails, plus a test that the legitimate owner of that subject can still
   register. A security fix with no test that could have caught the
   original bug is unproven.
5. Migrate `identity.proto`'s remaining claimed-actor fields (11
   occurrences) the same way T12.7 does, including the
   principal-overrides-wire-claim assertion.
6. **Update `HANDOFF.md`'s T10.2 bullet** to record the closure and how,
   rather than leaving a security gap described as open after it is fixed —
   the same staleness T11's Ceremony 1 had to correct for the
   write-handler-guard bullet.
7. **Scope guard (A13):** this ticket changes how `User` rows are *created*.
   It must **not** add profile fields. `subject` is the only new column;
   `SelfReportedStartingLevel` keeps its existing shape and meaning, and
   ADR-0012's prohibition on `Gender`/`PlayerRating`/Level-scoring binds
   this ticket unchanged.
8. **Reminder (A12):** this PR and T12.7's both append one line to
   `cmd/server/main.go` in the same wave — later merger resolves on its own
   source branch, keeping both, with fresh-worktree re-verification per
   A9(e).
9. Error handling — **gRPC codes only:** no principal → `Unauthenticated`;
   subject already registered → `AlreadyExists` (check the existing
   `ErrUserAlreadyExists` mapping before inventing one); malformed input →
   `InvalidArgument`.

**Cross-context dependency check:** `internal/platform/auth` (T12.2, exists
by dependency order). Migration `0019` pre-assigned; no other T12 ticket
adds a migration (A10).

**Story points:** 5. **Role:** principal-engineer. **Labels:**
`role:principal-engineer`, `type:story`, `points:5`.

---

## Dependency order and dispatch waves (see A12 for isolation detail)

```
Wave 1 (parallel, ≤5 implementers, worktree-isolated):
  T12.1  vet-integration gate           (Makefile + CLAUDE.md line)  ─┐
  T12.2  internal/platform/auth + ADR-0013                           ─┤
  T12.3  RefundPayment wiring           (payments.proto)             ─┼─ independent
  T12.4  CancelGame + HostID authz      (socialplay.proto)           ─┤
  T12.5  WCAG 3.3.4 + keyboard pass     (web/ only)                  ─┘

Wave 2 (parallel, ≤3 implementers, worktree-isolated):
  T12.6  board-of-record resolution     (docs + GitHub API only)
  T12.7  Booking + Facilities → verified principal      (needs T12.2)
  T12.9  CreateUser closure             (needs T12.2; migration 0019)
  ⚠ T12.7 and T12.9 each append ONE line to cmd/server/main.go.
    Later merger resolves on its own source branch, keeps both,
    one-line reason in the PR body, fresh-worktree re-verify (A9e).

Wave 3 (1 implementer):
  T12.8  Social Play + Payments + Competitions → verified principal
  — needs T12.2, and needs T12.3 + T12.4 MERGED (same two proto files)
```

Total: **46 points across 9 tickets** (T11: 47/9, T10: 37/8) —
deliberately at T11's shape rather than above it, with a shallower
dependency graph (2 deep vs. T11's 4) but a wider blast radius, which is
A14's recorded PdE/PE disagreement.

**Wave roll-call is mandatory before ending any work block** (A2). The
dispatch list above is the list to roll-call against.

## Explicitly deferred, not silently dropped

Per A5's standing rule, **every item below that outlives this sprint now
has a GitHub issue** — the first sprint in this project's history for which
that is true.

- **`app.NewService` → `ServiceOptions`** for `booking` (7 positional
  params, verified `service.go:47-55`) and `socialplay` (5, `service.go:59`).
  **Not taken:** it is a mechanical refactor of the exact files T12.7/T12.8
  are already editing, and bundling it would make an auth diff hard to
  review — the one class of diff where reviewability is the safety property.
  **Issue #123.** Best taken in the sprint *after* the auth migration
  settles, not before.
- **Game-cancellation cascade** to Bookings/Registrations (open since T5.1;
  `Game.Cancel()` flips status only, verified). **Not taken:** it is a
  product-semantics question (are courts freed, are players refunded — the
  latter now touching T12.3's brand-new refund path), not an extension of
  an existing pattern. **Issue #124**, linked from T12.4's PR.
- **Competitions-shaped `PayableType`** + the port/adapter pair (open since
  T9.6/T9.7). **Not taken:** it is a genuine Payments feature, and T12's
  Payments budget is spent on `RefundPayment` plus the auth migration.
  **Issue #125.**
- **Real per-Game price field**, retiring T8.10's
  `PLACEHOLDER_REGISTRATION_FEE_CENTS`. **Not taken:** needs Product Owner
  sign-off on pricing semantics before any code, per T8.10's own note.
  **Issue #126.**
- **Observability (Sentry + slog + uptime).** Not taken; unchanged
  reasoning from T11's plan — no incident or aging-issue pressure names it.
  Genuinely T13+. No issue opened: it is a roadmap item, not a disclosed
  gap, and A7's rule covers deferred *gaps*, not unstarted roadmap work.
- **`web/` migration off the deprecated `actor_user_id` fields.** T12.7/
  T12.8 deprecate but deliberately do not delete the wire fields, so the
  Vue client keeps working untouched this sprint. Removing them is a real
  follow-up — flagged here, and **the ticket that deprecates a field must
  open the issue** per A5 rather than this plan pre-opening one for work
  whose shape T12.7 will define.
- **CI server-side wiring** (Jenkins job/webhook/branch protection).
  **Not ticketed**, same structural reason as every sprint since SCRUM-6:
  it needs a reachable Jenkins instance and admin credentials no session
  has. **Exactly the same class as T12.2's disclosed IdP-tenant gap** —
  noted because this sprint adds a second instance of "the repo-side work
  is doable here, the server-side work is not," and ADR-0013 should say so
  in the same honest terms `HANDOFF.md` uses for Jenkins.
- **ADR-0012's Q1/Q2 work** (`PlayerRating`, matching algorithm, gender-mix
  matching). **Still blocked on the user**, checked this ceremony (A13).
  Not deferred by this team's choice and not this team's to decide.
- **`golang-migrate`/`goose` swap; ISO-8601 weekday numbering; the T6.4
  uncommitted payments concurrency proof.** Unchanged low-urgency infra
  debt, no new pressure this sprint.

## `HANDOFF.md` updates this ceremony's PR carries

- A new **T12** row in the Docs index, matching T10/T11's format.
- A correction to the **T11** row, which still reads "not yet written" for
  its retro and "not yet opened" for its reviews — both stale: the retro
  merged as PR #122 and all nine tickets merged as PRs #113–#121. Verified
  against the GitHub API this ceremony, not assumed. This is the same class
  of staleness T11's own Ceremony 1 had to fix for the write-handler-guard
  bullet, caught the same way — by checking a row rather than reading past
  it.
- A **T11 entry in the "Task backlog" narrative** recording the sprint's
  outcome, so `HANDOFF.md` resumes accurately from T11 before T12 work
  starts.

Ticket-level `HANDOFF.md` edits (T12.9's T10.2-bullet closure, the
Cross-cutting caveat that four contexts' claimed-actor checks become
verified) belong to their own tickets' PRs, not this one.

## Definition of Done (sprint-level)

All 9 tickets merged per `sprint-process.md`'s per-ticket DoD (PR-only,
CLAUDE.md rule 9 — **no exception for this plan**, which lands the same
way); sprint goal met (a verified principal enforced across all six
contexts, the `CreateUser` squatting DoS closed and proven closed,
`RefundPayment` and `CancelGame` shipped, `vet-integration` gating
`make ci`, the board-of-record question resolved in the process doc);
**wave roll-call performed before every work-block end** (A2), with any
"neither PR nor branch" ticket named; retro held
(`docs/process/t12-retro.md`) with findings indexed from `docs/LESSONS.md`'s
`## T12 sprint retro`; `HANDOFF.md` updated for T13 to resume from,
including whether A14's PdE/PE blast-radius concern and A9(a)'s
roll-call-vs-polling exit condition actually fired.
