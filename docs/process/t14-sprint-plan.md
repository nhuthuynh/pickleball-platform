# T14 Sprint Plan — Answer the gate question once and for all, give Game Admins a real store, and make issue closure structural

Ceremonies 1 and 2 per `docs/process/sprint-process.md` (read in its
**T13-amended** form — the "Correct the previous sprint's Docs-index row"
section T13's own Ceremony 1 added is the version this plan is governed by, and
this ceremony is its first live test), six-role team (briefs:
`docs/agent-operating-handbook.md` Part B), against `HANDOFF.md`,
`docs/process/t13-sprint-plan.md` (whose A0–A17 sections are this document's
structural template), and `docs/process/t13-retro.md`'s **six findings and ten
recommendations** — threaded into this plan below, one subsection per item,
adopted or adapted or explicitly declined, never silently dropped.

**Every claim in this plan that could be checked against the working tree was
checked this ceremony**, per the evidence-marking discipline T12 established and
T13 extended to any factual claim about existing code. Commands are recorded
inline so a reader can re-run them rather than trust them. Where a claim could
*not* be checked here, this plan says so rather than phrasing it confidently —
see A3's `internal/gen` limitation, which is load-bearing for T14.1.

**Four of this ceremony's checks changed the plan materially:**

1. Re-running #157's own enumeration found **22** ungated adapter packages, not
   the 20 its title claims — and #157's own body lists **21**. The delta is
   `internal/facilities/adapter/identity`, created by T13.3 *after* #157 was
   filed, plus an off-by-one between the issue's title and its own list. A gate
   built from #157's list would have shipped already-stale (A3).
2. The 22 split cleanly **11 codegen-free / 11 codegen-dependent**, and the
   codegen-free 11 were run green here with **no `internal/gen` present at
   all** — which decides the shape of T14.1's gate and proves half of it can
   land green in this environment (A3).
3. The retro's recommendation 4 dual question, applied to T14's own waves,
   produced a concrete AC that no capability-consumption question reaches:
   **T14.1's check must enumerate dynamically, never from a list**, because
   T14.4 creates new packages in the same sprint (A5).
4. `internal/socialplay/domain/game.go` — the file holding #165's gofmt
   violation — is also the file holding `EnsureHostOrGameAdmin` and
   `EnsureHost`, which T14.5 must change. That collision is what fixes T14.2's
   wave position, and it is the reason the retro's "combine recommendations 2
   and 3 into one ticket" suggestion is **declined with reasoning** (A4, A15).

---

# Part A — Ceremony 1 (backlog refinement)

**Product Manager + Principal Engineer**, per `sprint-process.md`. PM drove
scope/value framing; PE drove technical sequencing/feasibility. Both sign off on
every ticket below.

## A0. Ceremony 1's first act — the nine issues, closed for real (recommendation 1)

Recommendation 1 is explicit that this comes **before ranking anything**:
*"Ranking the backlog off the current issue list without doing this first will
re-rank six finished pieces of work."*

**Done, and verified rather than asserted.** All nine issues T13's code resolved
are closed with `state: closed`, `state_reason: completed`, and a comment naming
the merged PR that resolved each:

| Issue | Closed by | Verified |
|---|---|---|
| #123 | PR #172 (T13.8) | `state: closed`, `state_reason: completed`, `closed_at: 2026-08-15T06:36:27Z`, 1 comment |
| #129, #138 | PR #162 (T13.4) | #138's comment read back via the API this ceremony; cites PR #162 and the reviewer's own mutation check |
| #135, #136 | PR #163 (T13.5) | closed |
| #146, #152 | PR #166 (T13.2) | closed |
| #148 | PR #169 (T13.7) | closed |
| #154 | PR #170 (T13.3) | closed |

**Arithmetic cross-check, mirroring the retro's own method rather than trusting
the nine individual reads:** `list_issues(state: OPEN)` now returns
**`totalCount: 19`**. T13 ended at 28. 28 − 9 = 19, exactly. The count that
finding 1 used to prove nothing had been closed now proves the opposite.

**Re-checked as correctly *left* open, per recommendation 1's second half:**

- **#131** — T13.9 fixed the Payments half only and was titled *"partial fix for
  #131"*. The remaining cross-context half is **#158**, and this sprint takes it
  (T14.7). #131 stays open until #158 lands.
- **#147** — T13.6 shipped Host-only and was titled *"partial fix for #147"*.
  Correctly open: the entitled set #147 asks for is Host **+ assigned Game
  Admins**, and admins have no durable store. This sprint takes that store
  (T14.4) and the Social Play half of the widening (T14.5); #147 stays open for
  the Competitions half.

**The issue list no longer misdescribes the codebase.** That was
recommendation 1's stated cost, and it is discharged before any ranking below.

## A1. Threading recommendation 1's two durable fixes — and answering the question the retro left open

Recommendation 1 adopts two fixes *together*, because T13 proved either alone is
insufficient:

- **(i) Per-PR:** every review states the issues the PR closes, and the reviewer
  performs the close before moving to the next ticket — the symmetric half of
  A6, which has only ever had a slot for issues *opened*.
- **(ii) Sprint-level:** a DoD check that no issue remains open whose fix merged
  this sprint, *"verifiable in one API call by a party other than the merger."*

Both are adopted, and both land as amendments to `sprint-process.md` in
**T14.6**.

**The question the retro explicitly left open — *"whether the sprint-level gate
should block the retro or merely be reported in it is left open for T14's
Ceremony 1"* — is answered here, and the evidence is this ceremony itself.**

Neither, exactly. The sweep is placed in **two** moments, and the reasoning is
that this session just ran the experiment:

- **The retro reports it, and does not block on it.** A retro that cannot be
  held until a bookkeeping sweep is clean is a retro that will be held anyway
  with the sweep hand-waved — and the retro is the artifact that *records*
  failures, so making it refuse to exist when there is a failure to record
  inverts its purpose. T13's retro found the gap precisely *because* it was free
  to run and report.
- **The next sprint's Ceremony 1 runs it as its first act, before ranking**,
  and that is where the "party other than the merger" property actually lives.
  PO's requirement was never really about *which* moment — it was that the
  checker not be the person who just merged. Ceremony 1 satisfies that
  structurally: it is a different session, on a different day's work, already
  re-reading the issue list to rank it.

**This is not a prediction.** Recommendation 1 asked T14's Ceremony 1 to do
exactly this, and A0 above is the result: nine issues closed, the count
independently reconciled, before a single item was ranked. The mechanism worked
on its first run, which is the strongest available argument for putting it where
it just worked rather than somewhere it has never been tried.

**PE's dissent, recorded rather than smoothed over.** PE holds that placing the
authoritative sweep in the *next* sprint means the issue list is wrong for the
entire gap between sprints, and that this is a real cost — anyone reading the
repository in that window sees finished work as open. PO's answer is that fix
(i) is what closes that window, and the sprint-level sweep is the backstop for
when (i) is skipped. **Both are adopted; the disagreement is about which is
primary, and it is not resolved.** It becomes scoreable at T14's retro: if (i)
is followed for every T14 ticket that closes an issue, PE is right that the
backstop should rarely fire; if the backstop fires again, PO is right that the
per-PR moment is the one that gets skipped under dispatch pressure.

## A2. The remaining backlog, ranked — every open issue given a disposition

**All 19 open issues were re-read from the live API this ceremony**, not carried
from `HANDOFF.md` or from the retro's prose. Recommendation 5 sets the top of
this ranking (#144 and #168 first); everything below rank 2 is this ceremony's
own judgement.

| Rank | Issue(s) | Why here | T14? |
|---|---|---|---|
| **1** | **#157** (+ #138's shape) | Third consecutive sprint of the same class. 22 packages holding real tests are executed by **no** gate any machine here can run — including the regression test for T13's own headline bug. Every prior fix closed the named glob and left the next one. | **Taken — T14.1** |
| **2** | **#168** | The single missing store constraining three separate things (#147's residue, #149's admin branch, `RecordMatchResult`), and a **locked** CLAUDE.md product decision ("Per-game Game Admins can record offline payments") that cannot be implemented trustworthily without it. | **Taken — T14.4 + T14.5 (Social Play half; partial fix)** |
| **3** | **#144** | The sharpest remaining BOLA hole, and T13's plan already named it "should be first". **But its blocker is a product decision only the user/PO can make** — see A6. Implementing it means guessing that decision. | **Escalated, not guessed — T14.3 writes ADR-0015 and escalates; implementation is not in T14** |
| **4** | **#165** | Two lines of struct alignment that survived nine PRs and three reviews, because `gofmt -l` is a review convention and not a gate. Re-verified violated at the tip this ceremony. | **Taken — T14.2, with the gate** |
| **5** | **#131 / #158** | One concept, two gRPC codes, across three contexts — and `booking`'s single `toStatus` contradicts *itself*. T13.9 built the reusable template and deliberately declined to widen. | **Taken — T14.7** |
| **6** | **#160** | T13.5 made a nil verifier fail startup, which is correct — and it broke `make up`, which is **step 4 of `HANDOFF.md`'s own "First actions on resume"**. A documented onboarding path is currently non-functional. | **Taken — T14.9** |
| **7** | **#156** | `ScheduleGame`/`ScheduleCompetition` never shape-check caller-supplied `court_ids`, so a malformed id is a 500 instead of `InvalidArgument`. Verified this ceremony (A8). Cheap, real, public write path. | **Taken — T14.8** |
| **8** | **#149** | Fact-fabrication. **Its own named sub-gap is #168**, which T14 takes — so T14 moves this materially without closing it. Still needs read-side ports Payments does not have. | **Not taken; materially advanced by T14.4/T14.5** |
| **9** | **#164** | ADR-0014 conformance for three contexts' actor columns. Needs a **data backfill**, and the ADR itself records that today's comparisons are correct (by coincidence). | **Deferred — reasoned below** |
| **10** | **#167** | A Stripe webhook receiver would *remove* the `ConfirmOnlinePayment` authorization question rather than answer it. Needs a real Stripe account and a reachable endpoint. | **Deferred — blocked, same class as the IdP** |
| **11** | **#124, #125, #126, #130, #134, #137, #145** | Re-checked open; reasoning unchanged from T13's A16. | **Deferred — see "Explicitly deferred"** |

**PM/PE scoping decision: T14 takes ranks 1, 2 (partial), 4, 5, 6, 7, plus the
escalation at rank 3 and the process work recommendations 1 and 6 require.**
Nine tickets, 39 points.

## A3. Threading finding 2 / recommendation 2 — the mechanical gate-coverage check, and what re-measuring changed

**Adopted in full, as this sprint's marquee ticket.** This is the third
consecutive sprint of one class, and the retro's table is the argument:

| Sprint | The ungated code | The fix | What it left |
|---|---|---|---|
| T11 (finding 2) | `//go:build integration` files | T12.1's `vet-integration` (compiles them) | never *executed* without Docker |
| T12 (#138) | `internal/platform/**`, incl. the auth spine | T13.4's `test-platform` | the 22 adapter packages |
| T13 (#157) | 22 `internal/*/adapter/*` packages | **T14.1** | — |

PE's position in the retro's recorded disagreement is the one adopted: build
**the mechanical check**, so the *general* question is answered once instead of
the specific one being answered a fourth time.

### Re-measured this ceremony, and the numbers moved

Recommendation 2 is worded from #157, and #157's numbers are now stale. Verified
by re-running the enumeration against the tip (`b71fa36`):

```
$ grep -rl "func Test" --include="*_test.go" internal tools \
    | xargs -n1 dirname | sort -u | wc -l
38
```

Broken down against what each gate actually executes:

| Set | Count | Gate that runs it |
|---|---|---|
| `internal/*/{domain,app}` | 12 | `make test-domain` ✅ (re-run green here, 12 packages) |
| `internal/platform/**` | 3 | `make test-platform` ✅ (T13.4) |
| `tools/**` | 1 | `make test-tools` ✅ |
| **`internal/*/adapter/*`** | **22** | **none** ❌ |

**22, not the 20 in #157's title — and #157's own body lists 21.** The delta:

- `internal/facilities/adapter/identity` did not exist when #157 was filed at
  05:29:17Z; T13.3 created it and merged as PR #170 at 06:01:01Z. (+1)
- #157's title and its own enumerated list disagree by one. (+1)

**This is exactly why the check must be mechanical.** A ticket written from
#157's list would have shipped a gate that was already stale on the day it
merged, having missed a package created by its own sprint sibling.

### The split that decides the gate's shape

```
$ grep -rl "internal/gen" --include="*.go" internal/*/adapter | xargs -n1 dirname | sort -u
```

returns exactly the `adapter/postgres` and `adapter/grpcapi` packages — 12
packages, of which 11 hold tests. So of the 22:

- **11 are codegen-free** (`adapter/{facilities,identity,booking,sharetoken,socialplay,competitions,stripestub}`)
- **11 are codegen-dependent** (`adapter/{postgres,grpcapi}` across the contexts)

**Verified here, and this is the load-bearing measurement:** `internal/gen` does
**not** exist in this worktree (`ls internal/gen` → no such directory), and all
11 codegen-free packages were run green anyway:

```
$ go test ./internal/booking/adapter/facilities/... ./internal/booking/adapter/identity/... \
    ./internal/socialplay/adapter/{booking,facilities}/... \
    ./internal/competitions/adapter/{booking,facilities,sharetoken}/... \
    ./internal/payments/adapter/{socialplay,competitions,stripestub}/... \
    ./internal/facilities/adapter/identity/... -count=1
ok  (all 11)
```

**What this ceremony could NOT check, stated rather than glossed:** the 11
codegen-dependent packages were not run, because `make generate` needs
`buf`/`sqlc` and `internal/gen` is absent here. T14.1's implementer must
establish that half itself, and the ticket says so rather than asserting it
passes.

This settles #157's own open question between its two proposed shapes: **shape 1
(a post-`generate` target) is correct for the codegen-dependent half, and the
codegen-free half can join a Docker-free, codegen-free gate immediately** —
which is the half that can be proven green in this environment, and the half
holding T13.1's five new tests and #146's regression test.

## A4. Threading finding 5 / recommendation 3 — the formatting gate, and why it is *not* combined with recommendation 2

**Adopted: a formatting gate lands in `ci-checks`, together with the one
violation it fires on, so it goes green immediately.** Re-verified at the tip:

```
$ gofmt -l ./internal ./cmd
internal/socialplay/domain/game.go
$ grep -n "gofmt" Makefile Jenkinsfile
(no matches)
```

Both halves of the retro's finding confirmed independently: the violation
survives, and `make ci` has no formatting gate at all.

**Recommendation 3's closing suggestion — *"Same `Makefile` and same class as
recommendation 2 — reasonable to combine into one ticket"* — is declined, with
reasoning, and the reasoning is a fact the retro did not have.**

`internal/socialplay/domain/game.go`, the file holding #165's two-line
violation, is also the file holding `EnsureHostOrGameAdmin` (line 139) and
`EnsureHost` (line 174) — **the exact functions T14.5 must change** to consume
T14.4's Game-Admin store. Verified this ceremony by `grep`. So the gofmt fix is
not scope-free filler that can ride along with a `Makefile` ticket; it is an
edit to a file two other tickets in this sprint touch, and its *position in the
wave order* is what keeps it from colliding.

Combining it into T14.1 would put an 8-point CI ticket on the critical path of a
2-point formatting fix that every later wave wants as an ancestor. **T14.2
therefore ships first, in Wave 1, and T14.1 branches from a base already
containing it** — which also gives T14.1 a clean `gofmt` baseline to build its
own gate against. See A9 for the wave assignment and A10 for the shared-`Makefile`
pre-assignment.

## A5. Threading recommendation 4 — the dual question, adopted for one sprint only, and it earned its keep immediately

Recommendation 4 adds one question to the dependency-completeness check, **for
one sprint only**, and is explicit about the exit: *"Drop this question once
recommendation 2's mechanical check exists — do not accumulate both, which is
this project's recurring failure mode and one PE named in advance."*

> **The dual question:** for every gate, glob, or shared coverage artifact a
> ticket produces, which other in-flight tickets' outputs must it cover?

**Adopted, asked, and it changed a ticket's AC.** Asked of T14.1:

- **T14.4** creates a new migration, new domain files, a new port, and a new
  Postgres repository — plausibly a new package, certainly new test files.
- **T14.5** adds tests to `internal/socialplay/{domain,app,adapter/grpcapi}`.
- **T14.7** adds `error_mapping_test.go` to three `adapter/grpcapi` packages.

Every one of those lands **after** T14.1 is written and, for T14.4/T14.5,
possibly after it merges. A gate built from any enumeration performed *at
authoring time* — including one derived from #157's list, or from this plan's
own table in A3 — is stale before the sprint ends.

**Resulting AC, which no consumption-shaped question reaches:** T14.1's check
must compute both sides **at run time**, from `go list` and a source scan, and
must **fail** when a package holding a `func Test` is executed by no gate. It
must not contain a hand-maintained package list. This explicitly rejects #157's
own shape 2, whose stated cost was *"a hand-maintained list that will go
stale"* — a cost this ceremony can now demonstrate rather than predict, since
#157's list went stale inside one sprint (A3).

**Scheduled for removal.** If T14.1 merges, this question is **dropped** at
T15's Ceremony 1 and does not become a fourth standing planning question. Stated
here so the removal is a plan the next ceremony executes, not a judgement it has
to re-make.

## A6. Threading recommendation 5 — #144's product decision, surfaced here and escalated, not guessed

Recommendation 5 ranks #144 first among the authorization backlog and is precise
about the obligation: *"it needs a product decision on what owns a booking made
through the public quote-and-book flow, so **surface that decision at Ceremony 1
rather than discovering it mid-ticket**."*

**Surfaced here. It is a decision only the user/PO can make, and this ceremony
does not make it.**

### The gap, re-verified this ceremony rather than taken from #144

- `bookings` has **no owner column of any kind**. Read from
  `db/migrations/0001_init.sql:22-42`: `id, court_id, source, status,
  starts_at, ends_at, during, reference_id, created_at`. Nothing else.
- `app.Service.CancelBooking(ctx, bookingID string)`
  (`internal/booking/app/service.go:287`) takes **no actor parameter at all**.
  It shape-checks the id, fetches, cancels, updates. There is nothing to check.
- `CreateBooking` and `CancelBooking` are both in Booking's `PublicMethods()`
  (`internal/booking/adapter/grpcapi/authenticated.go:56-63`), whose own doc
  comment says they are public *"not because that is right, but because making
  them otherwise is out of this ticket's reach."*

So: anyone who knows or guesses a booking id can cancel that booking today.

### Why an engineering ticket cannot proceed

Adding `created_by_user_id` and requiring the principal to match is a
five-minute change *once you know the answer to this*:

> **DECISION D1 (for the user / PO):** What owns a Booking created through the
> public quote-and-book flow (T7.6), which today requires no account and no
> token?

The options are genuinely different products, and each has a cost this ceremony
can state but not choose between:

| Option | What it means | Cost |
|---|---|---|
| **(a) Authenticate the flow** | `CreateBooking` requires a principal; the booking is owned by that user; `CancelBooking` requires a match. | Breaks a **shipped** public UI flow — a player must now register before booking. A product/conversion decision. |
| **(b) Guest bookings with a capability token** | The flow stays public; creating a booking returns a one-time management token the client keeps; cancelling requires the token or an owning principal. | New concept in the domain and the API; a lost token means a booking nobody can cancel except an operator. |
| **(c) Owner-when-known, open-when-not** | Record the owner when a principal exists; leave it null otherwise; enforce only when non-null. | **Leaves the hole open for exactly the bookings that have it today.** Honest but does not close #144. |

**This ceremony declines to pick.** Option (a) changes a shipped user-facing
flow, which `sprint-process.md` puts on PM/PO, not PE — and unlike ADR-0012's
Q1/Q2 there is no legal dimension, so it is not permanently blocked, merely
*unanswered*.

**What T14 does instead of guessing:** **T14.3** writes **ADR-0015** recording
the three options, the verified facts above, and the escalation — following the
established ADR-0009/ADR-0010/ADR-0012 precedent, where an escalated question is
recorded as a decision-with-a-named-trigger rather than deferred a second time
in prose. Its trigger is the user's answer, not a sprint boundary. **No schema,
domain, or handler change is in T14.3's scope.** #144 stays open.

**Recorded so the retro can score it:** this is #144's *second* deferral (T13
was the first). It is deferred in a materially different form — an ADR with
named options and an escalation, rather than a paragraph in a plan — which is
the same progression ADR-0010 made after three prose deferrals. **If T15 defers
it a third time without an answer, that is a finding, not a decision.**

## A7. Threading recommendation 5's second half — #168, and what T14 actually takes

Recommendation 5 ranks #168 alongside #144: *"the blocker under both #147's
residue and #149, and #149's own text says it should probably be closed first."*

**Taken, and it is the sprint's largest engineering item.** Verified from #168
and the code this ceremony:

- Game-Admin and Competition-Admin assignments are **persisted nowhere**. No
  table, no domain type, no port.
- The concept exists only as a caller-supplied repeated string on the requests
  that need it — which means a caller can name themselves an admin and satisfy
  the check.
- `domain.Game.EnsureHostOrGameAdmin(actorUserID string, assignedGameAdminUserIDs []string)`
  (`internal/socialplay/domain/game.go:139`) takes the list **as an argument**,
  for exactly that reason: it has nothing to look it up in.

**Scope decision: T14 takes the Social Play half and says so.** Building both
the Game-Admin and Competition-Admin stores, both sets of assign/revoke RPCs,
and both consumption sites is two migrations, two domain types, two port/adapter
pairs and two proto changes — more than this sprint's remaining budget after
T14.1. Per A5 of T13 (the rule that survived and is still standing), the PRs are
titled **"partial fix for #168"**, never "closes", and **#168 stays open** for
the Competitions half. So does **#147**, whose Competitions roster read is
unchanged by this sprint.

**What T14 does close, honestly stated:** after T14.5, Social Play's roster read
is Host-**or-assigned-admin** with the assignment resolved from a durable store
rather than from the caller, and `RecordMatchResult`'s admin branch stops
trusting a caller-supplied list. That is the first authorization rule in this
codebase that can include an admin without being defeatable by naming yourself
one.

## A8. The rest of the backlog, re-checked rather than inherited

- **#156** — **Verified real this ceremony, and taken (T14.8).**
  `internal/socialplay/app/service.go` applies `uuidShape` to `GameID` at lines
  322, 442, 566, 668, 709, 765 and to `registrationID` at 381 — but
  `ScheduleGame` (line 244) passes `in.CourtIDs` straight into `domain.NewGame`
  and then into the reservation loop (lines 255-256) with **no shape check**.
  The guard convention is established and this path was missed by it.
- **#160** — **Taken (T14.9).** T13.5 correctly made a nil verifier fail
  startup and correctly disclosed the cost. The cost is that `make up` —
  `HANDOFF.md`'s "First actions on resume" step 4 — no longer works without
  `AUTH_*` env vars. A documented onboarding path being broken is worth three
  points, and T13.5's own reasoning (an opt-out flag re-creates the fail-open
  path) constrains the fix's shape: a **real dev key fixture**, not a bypass.
- **#164** — **Deferred, reasoned.** ADR-0014 §5a ruled that Social Play,
  Competitions and Payments hold subjects in `text` columns, are
  non-conformant-but-self-consistent, and that *"the checks are correct today by
  that coincidence."* Closing it needs a data backfill against identities a real
  IdP has not issued. **Directly relevant to T14.4**, which must not quietly
  pick a third convention — handled as a dependency, not by taking the issue
  (A12).
- **#167** — **Deferred, blocked.** A Stripe webhook receiver needs a real
  Stripe account and a publicly reachable endpoint. Same class as the IdP tenant
  and the Jenkins server-side wiring: the repo-side design is possible here, the
  thing that would make it real is not. Named so it is not mistaken for
  unprioritised.
- **#149** — **Not taken, but materially advanced.** Its own text names #168 as
  the sub-gap to close first; T14 closes that sub-gap's Social Play half.
  Payments still has no cross-context read ports (`internal/payments/port/` =
  `idgenerator.go`, `payment_processor.go`, `repository.go` — re-verified), so
  the issue itself stays open.
- **#124, #125, #126, #130** — Re-checked open. Reasoning unchanged from T13's
  A16. #126 still needs PO sign-off before any code.
- **#134** — Still open, still **declined as unticketable in this environment**:
  a manual screen-reader pass means driving NVDA/JAWS/VoiceOver and listening to
  it. No session here has assistive technology. Flagged for the user.
- **#137, #145** — Blocked on a real IdP. #145 narrowed by T13.3, not closed.

## A9. Threading recommendation 6 — the label taxonomy, decided rather than left to drift

Recommendation 6: *"Either adopt `context:*` deliberately … and add
`type:tech-debt` or map it to `type:chore`, or scrub them."*

**Re-verified this ceremony** against the live issue list: #167 carries **no
labels**; #168 carries `type:tech-debt`, `context:socialplay`,
`context:payments` — three labels, none in `sprint-process.md`'s stated
taxonomy; #165 carries `type:chore` but no `role:`.

**Decision (PM + PE), applied by T14.6:**

1. **`context:*` is adopted as a sanctioned, optional axis.** It is orthogonal
   to `role:` and `type:` — it says *where* the work lands, not who owns it or
   what kind it is — and this repository genuinely lacks that axis with six
   bounded contexts. Adopting it costs nothing and it is already in use.
   Permitted values are the six context names plus `platform`, so the set is
   closed rather than free-form.
2. **`type:tech-debt` is mapped to `type:chore` and retired.** The `type:` axis
   stays closed at `story|bug|chore|spike`. A second word for one concept is
   precisely what CLAUDE.md rule 7 exists to prevent, and "tech debt" and
   "chore" are not distinguishable here in any way that would change a decision.
3. **`role:` becomes mandatory on every issue**, `context:*` optional. #165
   (missing `role:`) and #167 (missing everything) are relabelled.
4. **Label conformance joins the review's issue enumeration**, which A6 of T13
   already asked for and which no T13 review performed.

**The alternative, stated because it was genuinely arguable:** scrub `context:*`
and keep the taxonomy minimal. Rejected because the labels are already applied,
already useful, and scrubbing them costs the same effort as sanctioning them
while destroying information. **BA's note, recorded:** an auto-created label is
indistinguishable from a sanctioned one after the fact, so this decision is only
durable if the taxonomy is stated somewhere a reviewer reads — which is why it
lands in `sprint-process.md` (T14.6) and not only here.

## A10. Threading recommendation 7 — the Wave-1.5 checkpoint, and the check that it does not fire

Recommendation 7: *"Keep the Wave-1.5 checkpoint on its narrow condition — a new
cross-cutting decision with three or more first-time consumers — and do not
generalise it."*

**The condition was checked against T14's shape. It does not fire, and the count
is stated rather than asserted:**

| Candidate | New cross-cutting capability? | First-time in-sprint consumers |
|---|---|---|
| **T14.4** (Game-Admin store) | Yes — a durable admin assignment, plus a ruling on which identifier space its `user_id` holds | **1** — T14.5 only |
| **T14.1** (gate-coverage check) | It is a gate, not a capability | 0 consumers — **and per finding 2 that sentence is exactly what missed last time**, so A5's dual question was asked of it instead, and it changed the AC |
| **T14.3** (ADR-0015) | No — it records options and escalates; it decides nothing downstream can consume | 0 |

**One consumer is not three. No checkpoint this sprint.** T14.5 simply sits in
Wave 3 behind T14.4's merge, which is ordinary dependency sequencing and needs
no special ceremony.

**Worth naming, since it is the honest qualification:** T14.4's ruling *does*
have three or more consumers **outside** this sprint (#149's admin branch,
#147's Competitions half, `RecordMatchResult`). The condition as the retro
worded it is about *first-time consumers in the sprint*, because the risk it
prices is four implementers inferring one unwritten decision in parallel. That
risk is absent here. Generalising the checkpoint to "3+ consumers ever" would be
the over-correction PE resisted in T12 and recommendation 7 explicitly forbids.

**PdE's cost objection stays carried, unresolved, with its scoring condition
intact** (retro finding 3): the next sprint that applies a Wave-1.5 checkpoint
*and* has its checkpoint ticket take a second review loop scores it. T14 applies
no checkpoint, so T14 does not score it, and this is **not** a sprint that
counts toward "several sprints passing with every checkpoint merging
first-loop" — no checkpoint ran.

## A11. Threading recommendation 8 — QA's port-contract-change rule is NOT adopted, and this plan does not reintroduce it

Recommendation 8: *"Do not adopt QA's port-contract-change rule. A13's check
caught the semantic half before dispatch and the Go compiler caught the
mechanical half at test-merge."*

**Not adopted. Deliberately checked that nothing below smuggles it back in**,
since this project's named failure mode is adopting the same fix in a second
shape.

**T14 contains a real port-contract change**, which makes this a live test
rather than a formality: **T14.5 changes
`domain.Game.EnsureHostOrGameAdmin`'s signature**, dropping the caller-supplied
`assignedGameAdminUserIDs []string` parameter in favour of a resolved lookup.
Every call site must change.

**No ticket in this plan requires enumerating those call sites in a PR body**,
and that is intentional:

- The **mechanical half** is a compiler error. A Go method losing a parameter
  breaks every caller at build time; `go build ./...` is in `ci-checks`.
  Enumerating them by hand produces the same list the build produces, later.
- The **semantic half** — whether the resolved admin set means the same thing
  as the caller-supplied one — is exactly what A12's dependency-completeness
  check asks, and it is asked there (arrow 1), before dispatch.

**QA's genuine residue is still tracked as #164 and is still a ticket, not a
rule** — a semantic-only contract change (same signature, different meaning) is
invisible to the compiler, and that is the one place the instinct has force.
T14.4/T14.5's change is *not* that class: it changes the signature, so the
compiler sees it.

## A12. Cross-context dependency check (every cell evidence-marked)

| Ticket | Calls into | Member/port exists? | How this was checked |
|---|---|---|---|
| T14.1 (gate-coverage check) | nothing — `Makefile` + a new `tools/` package | `tools/vulngate` is the in-repo precedent for a tested build tool, and `make test-tools` already runs `./tools/...` | Read `Makefile` `ci-checks`/`test-tools`/`test-domain`/`test-platform` targets directly (lines 6-7, 30-31, 36-37, 193-194) |
| T14.2 (gofmt gate + #165) | nothing — `Makefile` + one domain file | Violation confirmed present; no `gofmt` string in `Makefile` or `Jenkinsfile` | `gofmt -l ./internal ./cmd`; `grep -n "gofmt" Makefile Jenkinsfile` (no matches) |
| T14.3 (ADR-0015) | nothing — one new ADR file | ADR number verified: `docs/adr/` ends at **`0014`**, so `0015` is free | `ls docs/adr \| tail -4` |
| T14.4 (Game-Admin store) | `internal/socialplay` only; **no cross-context call** | Social Play has `port/` + `adapter/postgres` already; the new repository mirrors the existing shape. Identifier space is **already ruled** by ADR-0014 §5a (text/subject), so no new decision is needed — see A12 arrow 3 | Read `internal/socialplay/domain/game.go:139,174`; #168's own "what closing it looks like" section; ADR-0014 §5a |
| T14.5 (consume the store) | `internal/socialplay` only | `EnsureHostOrGameAdmin` (game.go:139) and `EnsureHost` (game.go:174) both exist and both live in `game.go` | `grep -rn "func .*EnsureHost" internal/socialplay/domain/` |
| T14.6 (process amendments) | nothing — `docs/process/sprint-process.md` | The two sections to amend (Ceremony 1, Execution DoD step 5, Label taxonomy) all exist | Read `sprint-process.md` in full this ceremony |
| T14.7 (#158 error mapping) | `booking`, `socialplay`, `competitions` `adapter/grpcapi` — **no cross-context call**, three parallel edits | `booking`'s self-contradiction confirmed: `ErrInvalidRecurringHireStatusTransition` at handler.go:402, `ErrIllegalStatusTransition` at :423, different codes in one `toStatus` | `grep -n "ErrIllegalStatusTransition\|ErrInvalidRecurringHireStatusTransition" internal/booking/adapter/grpcapi/handler.go` |
| T14.8 (#156 court-id shape) | `socialplay` + `competitions` `app` — no cross-context call | `uuidShape` exists and is used at 8 sites in socialplay's service; `ScheduleGame` (line 244) is not one of them | `grep -n "uuidShape\|CourtIDs" internal/socialplay/app/service.go` |
| T14.9 (#160 dev auth fixture) | `internal/platform/auth` config surface + `docker-compose.yml` | `auth.EnsureVerifierConfigured` exists (T13.5); `AUTH_*` env vars are the configured path | #160's body; T13.5's disclosed cost in the retro's "No finding on" section |

**No gap of the T9.6/T9.7 `PayableType` class was found hiding in these calls.**
Notably, **T14 introduces no new cross-context dependency direction at all** —
the first sprint since T12 for which that is true, and a direct consequence of
the sprint being weighted toward gates, process, and one context's own store.

## A13. Dependency-completeness check

For every arrow in the wave dependency graph: what the upstream ticket's AC
**delivers**, what the downstream ticket's AC **consumes**, and whether any
capability appears on a consumer's side with no producer.

| # | Arrow | Upstream AC delivers | Downstream AC consumes | Complete? |
|---|---|---|---|---|
| 1 | **T14.4 → T14.5** | A durable Game-Admin assignment table, domain type, port, Postgres repository, and Host-only assign/revoke RPCs | (a) A **read** path answering "is user U an admin of game G" from the store; (b) the guarantee that an admin cannot appoint admins | ❌ **GAP A — see below** |
| 2 | **T14.2 → T14.1** | A `ci-checks` line already containing the formatting gate, and a `gofmt`-clean tree | A `ci-checks` line to append `gate-coverage` to, and a clean baseline so the new gate's first run is not polluted by a pre-existing violation | ✅ Complete — wave-separated (A9/A14) |
| 3 | **ADR-0014 §5a → T14.4** | The ruling that Social Play's stored actor facts are `text` holding **subjects**, non-conformant-but-self-consistent, conformance deferred to #164 | Which identifier space the new `game_admins.user_id` column holds | ✅ Complete — **producer already exists and is merged.** #168's own text demands this be explicit ("should not quietly pick a third convention"); folded into T14.4's AC as a stated, checked decision rather than a new ticket |
| 4 | **T14.7 → T14.4** | A table-driven sentinel→code mapping test per context, with a source-level guard that fails when a sentinel in `domain/errors.go` has no mapping row | A `toStatus` mapping for any new sentinel T14.4 introduces (e.g. an admin-assignment authorization error) | ✅ Complete — and the guard **actively enforces** it: if T14.4 adds a sentinel and no mapping, T14.7's test fails. Wave-separated |
| 5 | **T14.1 → (all)** | A gate that executes every package holding a `func Test` | Nothing consumes it — **but A5's dual question applies**: its coverage set must include packages T14.4/T14.5/T14.7 create later | ✅ Complete **via the dynamic-enumeration AC** (A5), not via consumption |
| 6 | **T14.6 → (all)** | The closure DoD, the label taxonomy, and the review's closes-enumeration line | Every ticket's review and this sprint's own DoD | ✅ Complete — applies to T14 itself, including T14.6's own PR |
| 7 | **T14.3 → (none)** | ADR-0015: three options for D1, the verified facts, the escalation, a trigger tied to the user's answer | Nothing in T14 — deliberately. #144's implementation is out of scope until D1 is answered | ✅ Complete — no arrow |
| 8 | **T14.8 → (none)**, **T14.9 → (none)** | A boundary shape-check; a working local-dev auth path | Nothing in T14 | ✅ Complete — deliberately independent |

### The one gap, assigned to exactly one ticket before dispatch

**GAP A — "a read path answering *is user U an admin of game G*" has a consumer
(T14.5) and, as naively scoped, no producer.** #168's "what closing it looks
like" section lists a migration, domain types, a port/repository pair, and
**assign/revoke RPCs** — all *write*-side. T14.5's entire purpose is to make
`EnsureHostOrGameAdmin` resolve the set itself, which is a **read**.

Scoped naively, T14.4 would ship a store nothing can query and T14.5 would
discover mid-ticket that it must add the read method itself — the precise shape
of T12's finding 1, one sprint after the check that exists to prevent it.

**Assigned to T14.4**, whose AC explicitly includes a read method on the port
(`ListGameAdmins(ctx, gameID)` or `IsGameAdmin(ctx, gameID, userID)` — the
implementer picks, and states why) **plus a test proving the read observes what
the write wrote**. T14.5 consumes it and does not build it.

**GAP A's second half, also assigned to T14.4:** the guarantee that an admin
cannot appoint another admin. #168 names it (*"an admin must not be able to
appoint admins, or the Host-only distinction `ErrNotGameHost` exists to protect
is worthless"*), and it is a property of the *write* path, so it belongs to
T14.4, not to the consumer.

**Every capability on a consumer's side now has exactly one producer.** That is
the check's exit condition, and it is met.

## A14. Migration-number and shared-file pre-assignment

**Migration tip verified by listing the directory this ceremony**, not inferred:
`db/migrations/` ends at **`0019_identity_subject.sql`**. `0020` is **unclaimed**
— T13.3 chose translation over widening, so A14 of T13's conditional
pre-assignment correctly did not fire. That prediction resolving cleanly is
worth noting: the hedge was honest and the number is still free.

| Migration | Ticket | Condition |
|---|---|---|
| `0020_socialplay_game_admins.sql` | **T14.4** | **Unconditional.** The Game-Admin assignment table. |

**No other T14 ticket may claim `0020`, and no other T14 ticket adds a
migration.** Checked-and-does-not-fire, stated rather than left to inference:
T14.1/T14.2/T14.3/T14.6/T14.7/T14.8/T14.9 touch no schema; **T14.5 adds no
migration** — it consumes T14.4's table and changes only Go.

**ADR number verified:** `docs/adr/` ends at `0014`, so **`0015` belongs to
T14.3 only.**

### Shared *existing* files, and who appends to each

| Existing file | Tickets | Same wave? | Rule stated up front |
|---|---|---|---|
| `Makefile` (`ci-checks` line) | T14.2 (W1, `fmt-check`), T14.1 (W2, `gate-coverage`) | **No** — wave-separated | T14.1 branches from a base already containing T14.2's `ci-checks` edit and appends to it. Same mechanism as T13.9→T13.7 on `payments` `handler.go`, which held. |
| `internal/socialplay/domain/game.go` | T14.2 (W1, gofmt alignment), T14.5 (W3, `EnsureHostOrGameAdmin` signature) | **No** | **This is the collision that fixed the wave order (A4).** T14.2 lands first so T14.5 edits an already-clean file. **T14.4 must not touch `game.go`** — its new domain type goes in a new file (`game_admin.go`); if it believes it must edit `game.go`, that is a finding to disclose, not a licence. |
| `internal/socialplay/app/service.go` | T14.8 (W1, `ScheduleGame` shape check), T14.4 (W2, admin service methods), T14.5 (W3, roster read) | **No — all three in different waves** | Each appends independently; the wave gaps are the mitigation. Checked explicitly because three tickets in one file is this sprint's highest-contention spot. |
| `internal/socialplay/adapter/grpcapi/handler.go` | T14.7 (W1, `toStatus`), T14.4 (W2, assign/revoke handlers), T14.5 (W3, roster read authz) | **No — all three in different waves** | T14.4 adds its new sentinel's mapping into T14.7's **corrected** `toStatus`, rather than reverting or duplicating it — exactly what T13.7 did with T13.9's, which the retro scored as a clean handoff. |
| `internal/competitions/{app/service.go, adapter/grpcapi/handler.go}` | T14.8 (W1, app), T14.7 (W1, handler) | **Yes — same wave, but different files** | Checked: T14.8 touches `app/service.go` only; T14.7 touches `adapter/grpcapi/handler.go` only. **No overlap.** Named because "same context, same wave" is where a reader would expect one. |
| `internal/booking/adapter/grpcapi/handler.go` | **T14.7 only** | n/a | |
| `docs/process/sprint-process.md` | **T14.6 only** | n/a | Both recommendation 1's DoD amendments and recommendation 6's taxonomy land in one ticket **specifically to avoid two tickets appending to this file**. |
| `docker-compose.yml` | **T14.9 only** | n/a | |
| `cmd/server/main.go` | T14.4 (W2, wires the new repository) | n/a — single writer | **Checked, does not fire for T14.9.** T13.5 made the verifier configurable through `AUTH_*` env vars, so a dev fixture is compose + key material + docs, **not** a `cmd/server` change. T14.9 is instructed that if it believes it needs one, that is a finding to disclose. Named explicitly because this is the file T12's finding-1 collision lived in. |
| `tools/gatecoverage/**`, `docs/adr/0015-*.md`, `db/migrations/0020_*.sql` | **T14.1 / T14.3 / T14.4 respectively** | n/a | New files, single owner each. |

**No two tickets in the same wave create the same new file, and no two tickets
in the same wave append to the same existing file.** Both file-overlap
questions asked, plus A13's capability question, plus A5's dual coverage
question — which is the full set, one question wider than T13's.

## A15. ADR-0012's Q1/Q2 remain blocked — named, not touched, not dropped

`docs/adr/0012-*.md` blocks `PlayerRating`, the matching algorithm, and
gender-mix matching on two questions escalated to the user: **Q1** (the Player
Level formula's weighting) and **Q2** (whether gender-mix matching is in scope
at all, given it means collecting and algorithmically acting on a protected
attribute). Its trigger is explicit: *the sprint immediately following the
user's answers to both* — the user answering, not a sprint boundary.

**Checked this ceremony: no answer to either question exists, and nothing in the
codebase has drifted toward one.** Verified by grep across `internal`, `proto`
and `db`: every occurrence of `PlayerRating` or `Gender` is a **doc comment
stating the field deliberately does not exist** —
`internal/identity/domain/user.go:11,18`,
`internal/socialplay/domain/match.go:7,13`,
`internal/socialplay/domain/errors.go:135`,
`proto/pickleball/identity/v1/identity.proto:28`,
`db/migrations/0016_identity.sql:5-6`. **No `Gender` field, no `PlayerRating`
type, no Level-scoring formula anywhere in the tree.**

T14 therefore builds none of it, and this ceremony makes no product call on
either — they are not this team's to make. Recorded exactly as T10's, T11's,
T12's and T13's plans recorded it, so it is neither silently dropped nor
silently decided, for the **fifth** consecutive sprint. ADR-0012's constraint
that *"no PR may add a `Gender` field, a `PlayerRating` type, or any
Level-scoring formula"* binds every T14 ticket unchanged.

**Interaction check, stated because it would otherwise be discovered
mid-ticket:** T14.4 adds a per-Game *role* assignment (Game Admin). A Game Admin
is an **authorization** fact, not a player attribute — it is not a rating, not a
level, and not a protected attribute. It does not touch ADR-0012's blocked
surface, and T14.4 may not take the opportunity to add profile fields.
`SelfReportedStartingLevel` keeps its existing shape and meaning.

**Note the distinction from D1 (A6), which is also escalated to the user but is
NOT in this class.** ADR-0012's Q1/Q2 carry a legal/ethical dimension (a
protected attribute) and are blocked indefinitely. D1 is an ordinary product
decision that is merely *unanswered*, and it should be answered.

## A16. Recorded disagreements (not manufactured consensus)

**PE vs. PO — where the authoritative issue-closure sweep belongs.** Recorded in
full in A1. Both fixes adopted; the disagreement is about which is primary, and
it is scoreable at T14's retro. Not resolved here.

**QA vs. PdE — is T14.4/T14.5 a real close of anything, or #147's overclaim
again in a new coat?**

- **QA:** T13 shipped Host-only and honestly called it a partial fix. T14 now
  ships Host-or-admin for **Social Play only**, leaving Competitions' roster
  read exactly as it was. Two sprints of partial fixes on one issue is how an
  issue becomes permanent furniture. Either take both contexts or say plainly
  that #147 is a two-sprint item that has now taken three.
- **PdE:** the difference between T13's partial fix and this one is categorical,
  not incremental. T13's was narrower than the ask because the mechanism did not
  exist. T14 **builds the mechanism** — after T14.4/T14.5 the remaining work for
  Competitions is a second instance of a proven pattern, not another blocked
  question. That is exactly the shape this project ships well (T7.7 → T8.5 →
  T12.8 on the object-level authorization pattern).
- **Unresolved.** T14.5 as ticketed takes PdE's scope, **plus** an explicit
  instruction to state in its PR, as a checked negative, that Competitions'
  roster read is unchanged and why — and both PRs are titled **"partial fix for
  #168"** / **"partial fix for #147"** per A5's surviving rule. Both roles agree
  on that instruction; they disagree on whether taking one context is right.
  **Scoring condition for T14's retro:** if T15 does not close the Competitions
  half, QA's "permanent furniture" prediction is confirmed and #147 should be
  re-ranked accordingly.

**PE vs. PM — is 8 points on T14.1 defensible when it closes no user-facing
gap?**

- **PM:** T14.1 is the sprint's single largest ticket and ships nothing a user
  can see. #144 — an actual BOLA hole — gets 3 points and an ADR. That ordering
  needs a stated defence, not an inherited one from the retro's enthusiasm.
- **PE:** the defence is that #144 is *blocked on the user* (A6) and T14.1 is
  not, so this is not a choice between them. And T14.1 is the only ticket here
  that makes future sprints cheaper: three sprints have now each paid to close
  one glob, and the fourth would too.
- **Resolved in PE's favour, with PM's framing adopted for the sprint goal**:
  the goal below leads with what changes for a *reader of this repository*, and
  explicitly names what the sprint does **not** claim — including that #144 is
  still open.

**BA note, not a disagreement.** Ran the standing contradiction check against
every locked decision T14 touches. **One is directly relevant and is being
*honoured*, not reopened:** CLAUDE.md's locked *"Per-game **Game Admins** can
record offline payments."* T14.4/T14.5 are the first work that makes that
decision implementable trustworthily — today the admin list is caller-supplied,
so the locked decision is technically shipped and practically defeatable.
Nothing here reopens it. One naming note carried into T14.4's text: the
ubiquitous language already has **Game Admin** (CLAUDE.md, both protos,
`EnsureHostOrGameAdmin`); the new table, domain type, port and RPCs must use
that exact term and must not introduce "moderator", "organiser", or "manager" as
a synonym (CLAUDE.md rule 7).

---

# Part B — Ceremony 2 (sprint planning)

## Sprint goal

> Every package in this repository that holds a test is executed by a gate a
> session can actually run, enforced **mechanically** rather than by a list
> someone remembers to update — so the third consecutive sprint of the same
> class is the last one; `gofmt` becomes a gate instead of a review convention;
> a Game Admin becomes a fact the system stores rather than a claim the caller
> makes, so Social Play's roster read can widen to the people actually entitled
> to it; and the nine issues T13 fixed but never closed are closed, with the
> mechanism that let that happen amended into the process. **What this sprint
> does not claim:** `CancelBooking` still has no authorization check (#144) — it
> is escalated as a product decision, not built; Competitions' roster read and
> Competition-Admin store are untouched (#147, #168 stay open); Payments still
> compares a verified actor against caller-supplied ownership facts (#149,
> materially advanced but open); and no token from a real identity provider can
> be verified until a remote JWKS source exists (#137).

## In-scope tickets

### T14.1 — A mechanical gate-coverage check: every package holding a test is executed by some gate

**Story:** As a maintainer, I want a check that mechanically proves no package
holding tests is invisible to every gate, so that the next ungated glob is
caught by a command rather than by whoever happens to trip over it three sprints
from now.

**Description:** T13 retro finding 2 / recommendation 2, and the sprint's
highest-value item. Three consecutive sprints have each closed the *named* glob
and left the next one (T11 build tags → T12 `internal/platform` → T13's 22
adapter packages). This ticket answers the **general** question instead. Closes
**#157** and generalises **#138**'s shape so a fourth instance cannot form.
Depends on T14.2 (Wave 1) for a `gofmt`-clean baseline and the `ci-checks` line.

**Instructions:**

1. **Build the check as a tested Go program under `tools/`**, following
   `tools/vulngate`'s in-repo precedent (a build tool that carries real logic and
   therefore real tests). `make test-tools` already runs `./tools/...`, so the
   check gets test coverage for free. TDD-first per CLAUDE.md rule 1.
2. **Both sides of the diff must be computed at run time. A hand-maintained
   package list is explicitly rejected** (A5, and #157's own shape 2 names
   "a list that will go stale" as its cost):
   - **Packages holding tests**: derive from the module — a package containing
     at least one `func Test` in a `_test.go` file. Include build-tagged files
     in the scan, and classify them (see instruction 4).
   - **Packages some gate executes**: derive from the `Makefile`'s own test
     targets rather than from a copy of their patterns, so a target's pattern
     changing cannot silently desynchronise the check from the thing it checks.
     If deriving from the `Makefile` proves impractical, a single declared
     mapping inside the tool is acceptable **provided a test fails when the
     `Makefile` and the mapping disagree** — state which approach was taken and
     why.
3. **The check fails (non-zero exit) when a package holds a `func Test` and no
   gate executes it**, printing the offending packages. Wire it into
   `ci-checks` **after** the existing test targets.
4. **Classify, do not conflate, the three states** — this is what distinguishes
   this check from a fourth glob widening:
   - executed by a Docker-free gate ✅
   - **compiled but never executed** (the `//go:build integration` files —
     `make vet-integration` typechecks them; this is T11's finding, already
     known and deliberately accepted) ⚠️
   - executed by nothing at all ❌ — the failing state
5. **Close the 22, using measurements you re-derive rather than this plan's.**
   Verified this ceremony (A3), and re-verify before relying on it:
   - **22** adapter packages hold tests and are executed by no gate. **Note
     #157's title says 20 and its own body lists 21** — both are stale; the
     delta is `internal/facilities/adapter/identity` (created by T13.3 after
     #157 was filed) plus an off-by-one. **Do not build from #157's list.**
   - **11 are codegen-free** and were run green here **with no `internal/gen`
     present at all**. These can join a Docker-free, codegen-free gate
     immediately.
   - **11 are codegen-dependent** (`adapter/postgres`, `adapter/grpcapi` — they
     import `internal/gen`). Per #157, these **cannot** fold into `test-domain`
     without breaking its documented dependency-free contract, which is the T0
     resume gate. Use #157's shape 1: a target depending on `generate`, mirroring
     how `vet-integration` already does.
   - **This ceremony could not run the codegen-dependent 11** (`internal/gen`
     absent, no `buf`/`sqlc` here). **Establish that they pass before wiring
     them into a gate**, and if any does not, that is a finding to disclose —
     not a reason to exclude it.
6. **The gate must land green**, like T13.4's did. If a package fails, fix it or
   disclose it; do not silence the check.
7. **Mutation-check the check itself (CLAUDE.md rule 10)**: temporarily add a
   package holding a `func Test` that no gate executes, confirm
   `make gate-coverage` fails and names it, remove it, confirm green. A
   coverage check that cannot fail is exactly the vacuous artifact this ticket
   exists to prevent — record the output in the PR.
8. **Sprint-sibling coverage (A5's dual question, and the reason this AC
   exists):** T14.4, T14.5 and T14.7 all add test files, and T14.4 plausibly a
   new package, **after** this ticket is written. Confirm in the PR that the
   check's enumeration is dynamic enough that a package created later is covered
   with no edit to this ticket's output.
9. Non-functional: no new Docker dependency; `make test-domain` and
   `make test-platform` stay green; the tool itself is covered by
   `make test-tools`.

**Cross-context dependency check:** none — `Makefile` plus a new `tools/`
package (A12).

**Story points:** 8. **Role:** principal-engineer. **Type:** chore.
**Labels:** `role:principal-engineer`, `type:chore`, `context:platform`.

---

### T14.2 — A formatting gate in `ci-checks`, landing with the one violation it fires on

**Story:** As a reviewer, I want `gofmt` enforced by a gate rather than by
convention, so that a red `gofmt -l` line means something new is wrong instead
of being noise every reviewer learns to ignore.

**Description:** T13 retro finding 5. Two lines of struct-field alignment in
`internal/socialplay/domain/game.go` survived nine PRs and **three reviews that
each independently re-observed and correctly declined to fix them** — every
refusal was correct scope discipline, and the outcome was still that nothing
happened, because no ticket owns the repository and `make ci` has no formatting
gate at all. Closes **#165**. **First in the wave order** (A4): it lands the
gate every later PR is judged by, and it cleans the exact file T14.5 must edit.

**Instructions:**

1. Add a formatting check to `ci-checks`. `gofmt -l` printing any path must be a
   **failure**, not a warning — verified this ceremony that `grep -n "gofmt"
   Makefile Jenkinsfile` returns **no matches**, so there is nothing to extend
   and this is a genuinely new gate.
2. **Fix #165 in this same PR**, so the gate goes green on its first run rather
   than landing red and training people to ignore it. Verified this ceremony:
   `gofmt -l ./internal ./cmd` returns exactly one file,
   `internal/socialplay/domain/game.go`, and the whole diff is struct-field
   alignment.
3. **Change nothing else in that file.** It also holds `EnsureHostOrGameAdmin`
   (line 139) and `EnsureHost` (line 174), which **T14.5 must change in Wave 3**
   (A14). A whitespace-only diff keeps that handoff clean; a behavioural change
   here would collide.
4. **Prove the gate is not vacuous (CLAUDE.md rule 10):** after fixing #165,
   temporarily mis-format a file, confirm `make ci-checks` fails at the new
   step, restore, confirm green. Record the output. A formatting gate that
   passes on a mis-formatted tree is worse than none.
5. Scope: the check covers the Go tree (`./internal`, `./cmd`, `./tools`).
   `golangci-lint run ./...` reports 0 issues today, so **no configured linter
   catches this** — state in the PR whether the new check subsumes anything the
   linter already does, or is genuinely additive.
6. Non-functional: `make test-domain` and `make test-platform` stay green; no
   behavioural change to any package.

**Cross-context dependency check:** none — `Makefile` plus one domain file
(A12).

**Story points:** 2. **Role:** principal-engineer. **Type:** chore.
**Labels:** `role:principal-engineer`, `type:chore`, `context:platform`.

---

### T14.3 — ADR-0015: what owns a Booking made through the public flow (#144's escalation)

**Story:** As a Product Owner, I want the one product decision blocking the
sharpest remaining authorization hole written down with its options and costs,
so that I can answer it — rather than an engineer guessing it inside a ticket
and shipping a flow change nobody chose.

**Description:** Recommendation 5 asks that #144's product decision be *surfaced
at Ceremony 1 rather than discovered mid-ticket*. A6 surfaces it as **D1** and
declines to answer it. This ticket makes that escalation durable, following the
ADR-0009/ADR-0010/ADR-0012 precedent where an escalated question becomes an ADR
with a named trigger rather than a second paragraph of deferral. **#144 is not
implemented by this ticket and stays open.**

**Instructions:**

1. Write `docs/adr/0015-booking-ownership-for-public-bookings.md`. **ADR number
   verified this ceremony:** `docs/adr/` ends at `0014`.
2. **Re-verify and record the three facts that define the gap** — do not copy
   them from this plan:
   - `bookings` has **no owner column** (`db/migrations/0001_init.sql:22-42`).
   - `app.Service.CancelBooking` takes **no actor parameter**
     (`internal/booking/app/service.go:287`).
   - `CreateBooking`/`CancelBooking` are in `PublicMethods()`
     (`internal/booking/adapter/grpcapi/authenticated.go:56-63`), whose doc
     comment already discloses this.
3. **State the question as one sentence a non-engineer can answer**, and lay out
   the three options with their costs (A6 gives the starting shapes: authenticate
   the flow / a capability token for guest bookings / owner-when-known). Add or
   correct options if the code says something this ceremony got wrong.
4. **Do not recommend one.** This is the ADR's whole point. Recording a
   preference is fine and useful; recording it as *the decision* is not — option
   (a) changes a shipped public UI flow (T7.6), which is PM/PO's call.
5. **State the trigger explicitly: the user's answer, not a sprint boundary** —
   mirroring ADR-0012's wording exactly, so the two escalations read the same
   way to a future ceremony.
6. **Distinguish this from ADR-0012's Q1/Q2 in the ADR itself.** Those carry a
   legal/ethical dimension and are blocked indefinitely; D1 is an ordinary
   product decision that is merely unanswered and *should* be answered. Naming
   the difference stops a future reader filing D1 in the permanently-blocked
   drawer.
7. **Record that this is #144's second deferral, and in what form**, so T15 can
   score it: T13 deferred it in a plan paragraph; T14 escalates it as an ADR
   with named options. A third deferral without an answer is a finding.
8. Non-functional: **no code, schema, proto, or handler change.** If writing the
   ADR surfaces a fact that changes the options, update the ADR — not the
   codebase.

**Cross-context dependency check:** none — one new ADR file (A12).

**Story points:** 3. **Role:** product-owner. **Type:** spike.
**Labels:** `role:product-owner`, `type:spike`, `context:booking`.

---

### T14.4 — A durable Game-Admin store for Social Play, with a Host-only assignment path

**Story:** As a Game Host, I want the Game Admins I assign to be recorded by the
system, so that an authorization rule can include them without any caller being
able to name themselves one.

**Description:** #168, ranked second by recommendation 5 and the blocker under
#147's residue, #149's admin branch, and `RecordMatchResult`. Verified this
ceremony: Game-Admin assignment is **persisted nowhere** — no table, no domain
type, no port — and `domain.Game.EnsureHostOrGameAdmin`
(`internal/socialplay/domain/game.go:139`) takes the admin list **as an
argument** because it has nothing to look it up in. **Partial fix for #168**:
Social Play only; the Competition-Admin half is deferred (A7, A16). Wave 2 —
needs T14.2's clean `game.go` and T14.7's corrected `toStatus` as ancestors.

**Instructions:**

1. **Migration `0020_socialplay_game_admins.sql`** — number pre-assigned in A14
   and **verified free** this ceremony (`db/migrations/` ends at `0019`). A
   Game-Admin assignment table roughly `(game_id, user_id, assigned_by,
   assigned_at)` with a uniqueness guard so the same user is not assigned twice
   to one Game.
2. **State the identifier space explicitly, and follow ADR-0014 §5a** — do not
   pick a third convention. #168 names this requirement itself. Social Play
   stores actor facts as `text` holding **subjects** today, non-conformant but
   self-consistent, with conformance deferred to **#164**. The new column must
   match its neighbours (`games.host_id`), and the migration must **say so in a
   comment**, citing ADR-0014 §5a and #164, so a future backfill finds it. Going
   uuid here would make one side of an ownership comparison a uuid and the other
   a subject — the exact failure ADR-0014 §5a legislated against.
3. **Domain type in a NEW file** — `internal/socialplay/domain/game_admin.go`.
   **Do not edit `game.go`** (A14): T14.2 owns its formatting and T14.5 owns its
   `EnsureHostOrGameAdmin` change. If you believe you must edit `game.go`, that
   is a finding to disclose, not a licence.
4. **Port + Postgres repository**, mirroring Social Play's existing shape
   (`internal/socialplay/port/`, `internal/socialplay/adapter/postgres/`), plus
   sqlc queries. Follow the repo's existing patterns rather than inventing one.
5. **A READ method is in scope and is not optional (A13 GAP A).** #168's own
   sketch lists only write-side work; **T14.5 consumes a read.** Ship
   `IsGameAdmin(ctx, gameID, userID)` or `ListGameAdmins(ctx, gameID)` — pick
   one, state why — **with a test proving the read observes what the write
   wrote.** Shipping a store nothing can query would reproduce T12's finding-1
   shape one sprint after the check that exists to prevent it.
6. **`AssignGameAdmin` / `RevokeGameAdmin` RPCs, Host-only** — both in
   `AuthenticatedMethods()`, both resolving the actor from `auth.Principal` per
   ADR-0014, both enforcing `EnsureHost`. **An admin must not be able to appoint
   an admin** (#168: *"or the Host-only distinction `ErrNotGameHost` exists to
   protect is worthless"*). This is a write-path property and it belongs here,
   not to the consumer. Prove it with a test where an assigned admin attempts an
   assignment and gets `PermissionDenied`.
7. **Any new domain sentinel needs a `toStatus` mapping row** in
   `internal/socialplay/adapter/grpcapi/handler.go`, added into **T14.7's
   corrected `toStatus`** — not reverted, not duplicated (A14; this is exactly
   what T13.7 did with T13.9's and the retro scored it as clean). T14.7's
   source-level guard will fail if a sentinel has no mapping row, so this is
   enforced rather than remembered.
8. **Do NOT retire the caller-supplied proto fields in this ticket.**
   `RecordMatchResultRequest.assigned_game_admin_user_ids` and the Payments
   equivalents stay on the wire; **T14.5** stops trusting Social Play's.
   Deprecating rather than deleting is needed for client compatibility (#168
   says so). Naming this as out of scope so it is a decision, not an omission.
9. **Ubiquitous language (CLAUDE.md rule 7, A16's BA note):** the term is **Game
   Admin**, already used in CLAUDE.md's locked decision, both protos, and
   `EnsureHostOrGameAdmin`. Do not introduce "moderator", "organiser" or
   "manager".
10. **PR title: "partial fix for #168", never "closes"** (T13's A5, which the
    retro scored as the sprint's cleanest adopted rule). The Competition-Admin
    half is not built.
11. Non-functional: TDD-first; `make test-domain`, `make test-platform` and
    (once T14.1 merges) the new gate stay green; mutation-check the Host-only
    enforcement (CLAUDE.md rule 10) and record it.

**Cross-context dependency check:** none — `internal/socialplay` only. No new
cross-context dependency direction (A12).

**Story points:** 8. **Role:** product-engineer. **Type:** story.
**Labels:** `role:product-engineer`, `type:story`, `context:socialplay`.

---

### T14.5 — Resolve Game Admins from the store, and widen Social Play's roster read

**Story:** As a Game Admin, I want to read my Game's roster, so that I can do the
job I was assigned — without the check being satisfiable by any stranger who
lists their own id.

**Description:** The consumption half of #168, and the reason it was worth
building. T13.6 shipped Host-only precisely because the alternative — "Host **or**
anyone in the caller-supplied list" — would have let any caller read any roster.
With T14.4's durable store the entitled set becomes expressible. **Partial fix
for #147** (Social Play half; Competitions' roster read is untouched). Wave 3 —
needs T14.4 merged.

**Instructions:**

1. **Change `domain.Game.EnsureHostOrGameAdmin` to stop taking a caller-supplied
   list.** Today it is
   `EnsureHostOrGameAdmin(actorUserID string, assignedGameAdminUserIDs []string)`
   (`internal/socialplay/domain/game.go:139`, verified this ceremony). The
   resolved set comes from T14.4's read method instead. Keep the domain pure
   (CLAUDE.md rule 2): the resolution happens in `app`, and the domain receives
   the resolved set or a resolved boolean — it must not gain a port import.
2. **Every call site must change**, and the compiler will name them for you —
   `go build ./...` is in `ci-checks`. **Per A11 this ticket does NOT owe a
   hand-enumerated adapter list in its PR body**: the retro closed that question
   in PE's favour, and hand-enumerating would produce the same list the build
   produces, later.
3. **Widen `ListRegistrationsForGame` from Host-only to Host-or-assigned-admin**,
   with the admin set resolved from the store. Prove with a test that an
   assigned admin succeeds and a non-admin, non-Host principal still gets
   `PermissionDenied`.
4. **`RecordMatchResult`'s admin branch stops trusting the caller.** Its
   `assigned_game_admin_user_ids` field stays on the wire (T14.4 instruction 8)
   but must be **ignored** — the same shape T12.8 used for the deprecated
   `actor_*` fields. **Prove it is ignored by mutation**, exactly as T12 proved
   the `actor_*` fields were: a request naming the caller as an admin, with no
   corresponding store row, must be denied.
5. **State as a checked negative, in the PR: Competitions' roster read
   (`ListEntriesForCompetition`) is unchanged and still Host-only**, and why —
   no Competition-Admin store exists (T14.4 built the Social Play half only).
   A16 records a live QA/PdE disagreement about exactly this scope; the PR must
   not paper over it.
6. **PR title: "partial fix for #147"**, never "closes". #147 and #168 both stay
   open.
7. **`game.go` is clean when you get it** — T14.2 fixed #165 in Wave 1 and T14.4
   was forbidden from touching the file (A14). Your diff to it should be
   behavioural only.
8. Non-functional: TDD-first; mutation-check both the widened read and the
   ignored wire field and record both (CLAUDE.md rule 10); all gates green,
   including T14.1's new one.

**Cross-context dependency check:** none — `internal/socialplay` only; consumes
T14.4's port within the same context (A12, A13 arrow 1).

**Story points:** 5. **Role:** product-engineer. **Type:** story.
**Labels:** `role:product-engineer`, `type:story`, `context:socialplay`.

---

### T14.6 — Amend `sprint-process.md`: the closure DoD, and a decided label taxonomy

**Story:** As a future ceremony, I want the issue-closure step and the label
taxonomy to be written rules rather than habits, so that the failure T13 hit nine
times out of nine has something structural standing in its way.

**Description:** Recommendations 1 and 6. T13 got the hard, judgement-laden half
of issue hygiene right (every PR title honestly said "partial fix" or "closes")
and dropped the trivial mechanical half entirely — because **A5 governed the
sentence and nothing governed the API call**. Both amendments land in one ticket
specifically so two tickets do not append to `sprint-process.md` (A14).

**Instructions:**

1. **Amend the Execution section's DoD step 5 with the per-PR half
   (recommendation 1(i)):** every review **states the issues the PR closes**, and
   **the reviewer performs the close before moving to the next ticket**. This is
   the symmetric half of the rule T12 added for issues *opened* — the review
   template has had a slot for new gaps and none for resolved ones, so a
   reviewer scanning their own checklist finds nothing missing.
2. **Amend the sprint-level Definition of Done with the sprint-level half
   (recommendation 1(ii)):** the retro **runs and reports** a sweep for any issue
   still open whose fix merged this sprint, and **the next sprint's Ceremony 1
   runs the same sweep as its first act, before ranking anything.** Record A1's
   reasoning for splitting it this way — including that the retro **reports and
   does not block**, and that Ceremony 1 is where the "party other than the
   merger" property actually lives.
3. **Record that this is not a prediction.** T14's own Ceremony 1 ran the sweep
   and closed nine issues before ranking (A0). Cite it, so the amendment carries
   its own evidence rather than a hope.
4. **Record PE's dissent** (A1): PE holds the issue list is wrong for the whole
   gap between sprints and that the per-PR half must be primary. Per
   `sprint-process.md`'s own "do not manufacture consensus" rule, this stays a
   recorded disagreement with a scoring condition, not a smoothed-over sentence.
5. **Amend the Label taxonomy section with A9's decision:**
   - `context:*` **adopted** as an optional axis, values closed to the six
     bounded contexts plus `platform`.
   - `type:tech-debt` **mapped to `type:chore`** and retired; the `type:` axis
     stays closed at `story|bug|chore|spike`.
   - `role:` **mandatory** on every issue; `context:*` optional.
   - **Label conformance joins the review's issue enumeration** (T13's A6 asked
     for the enumeration; no review checked conformance).
6. **Apply the taxonomy to the three non-conformant issues**, so the amendment
   is not merely aspirational: **#168** (`type:tech-debt` → `type:chore`, keep
   both `context:*`, add `role:`), **#165** (add `role:`), **#167** (add `role:`
   and `type:`, plus `context:payments`).
7. **Do not add a fourth planning question.** A5 adopts recommendation 4's dual
   coverage question **for this sprint only**, and it is **dropped at T15** if
   T14.1 merges. Record that scheduled removal here, so T15 executes a plan
   rather than re-making a judgement — this project's named failure mode is
   accumulating two shapes of one fix.
8. Non-functional: documentation only, no code. Per CLAUDE.md rule 9 this lands
   via PR like everything else — **no exception for process docs.**

**Cross-context dependency check:** none — `docs/process/sprint-process.md` only
(A12).

**Story points:** 3. **Role:** product-manager. **Type:** chore.
**Labels:** `role:product-manager`, `type:chore`.

---

### T14.7 — One domain sentinel, one gRPC code across booking, socialplay and competitions

**Story:** As a gRPC client, I want a status-machine violation to return the same
code whichever context raised it, so that error handling is a property of the
API rather than a per-context accident.

**Description:** #158, the cross-context half of #131 that T13.9 deliberately
declined to widen into. Verified this ceremony: `booking`'s **single** `toStatus`
maps `ErrInvalidRecurringHireStatusTransition` → `FailedPrecondition` (line 402)
and `ErrIllegalStatusTransition` → `InvalidArgument` (line 423) — the same
one-concept-two-codes defect as #131, **inside one function**, with no RPC
boundary to explain it. Closes **#158** and, with it, **#131**.

**Instructions:**

1. **Re-derive the current mapping from each context's `toStatus`** rather than
   trusting #158's table or this plan's. #158 was written at T13.9 and the
   handlers have changed since.
2. **Decide once, then migrate all three contexts in one PR** — #158 is explicit
   that piecemeal is how #131 arose. The reasoning is settled and should be
   restated, not re-litigated: gRPC defines `INVALID_ARGUMENT` as a problem with
   the argument *regardless of system state*, and `FAILED_PRECONDITION` as *"the
   system is not in a state required for the operation's execution."* A
   status-machine violation is state-dependent by definition, and **each
   context's own `FailedPrecondition` sibling already concedes the point**
   (`ErrGameCancelled`, `ErrCompetitionCancelled`,
   `ErrInvalidRecurringHireStatusTransition`).
3. **Use T13.9's template**, which exists for this:
   `internal/payments/adapter/grpcapi/error_mapping_test.go` pins every sentinel,
   asserts cross-RPC equality, and adds **two source-level guards** — one that
   fails on a second `func(error) error` mapper, one that fails when a sentinel
   in `domain/errors.go` has no mapping row. **Port both guards to all three
   contexts.** The second guard is what makes T14.4's new sentinel enforced
   rather than remembered (A13 arrow 4).
4. **Record the blast radius rather than assuming it** — #158 measured it and it
   should be re-checked: grpc-gateway maps **both** `InvalidArgument` and
   `FailedPrecondition` to **HTTP 400**, so REST/`web/` clients are unaffected;
   a survey of `web/src` at T13.9 found it branches on HTTP status only, never
   on gRPC codes. Re-verify both claims and state them in the PR.
5. **Wave 1, and it precedes T14.4 deliberately** (A14): T14.4 adds its new
   sentinel's mapping into your corrected `toStatus`, so the corrected version
   must exist first.
6. Non-functional: TDD-first; `make test-domain` and `make test-platform` green;
   mutation-check at least one mapping per context (break the row, confirm the
   test fails, restore) and record it.

**Cross-context dependency check:** three contexts' `adapter/grpcapi` edited in
parallel, but **no context calls another** — these are three independent
`toStatus` functions (A12).

**Story points:** 5. **Role:** principal-engineer. **Type:** story.
**Labels:** `role:principal-engineer`, `type:story`, `context:booking`,
`context:socialplay`, `context:competitions`.

---

### T14.8 — Shape-check caller-supplied `court_ids` on ScheduleGame and ScheduleCompetition

**Story:** As an API client, I want a malformed court id to be answered as a bad
argument, so that a typo returns a 400 instead of a 500 and the server stops
telling me it broke when I did.

**Description:** #156, disclosed by T13.1 and not closed by it. Verified this
ceremony: `internal/socialplay/app/service.go` applies `uuidShape` at eight
sites (lines 322, 381, 442, 566, 668, 709, 765) but **`ScheduleGame` (line 244)
passes `in.CourtIDs` straight into `domain.NewGame` and the reservation loop
(255-256) unchecked.** The T10.7 boundary-guard convention is established; this
public write path was missed by it. Closes **#156**.

**Instructions:**

1. **Re-verify the gap before fixing it**, including the Competitions side —
   this plan checked Social Play directly and took Competitions from #156.
2. **Follow the T10.7 convention exactly** rather than inventing a response:
   each handler answers a malformed id the same way that handler's existing
   not-found/precondition semantics answer an unknown-but-well-formed one, so no
   enumeration oracle is created. `uuidShape` already exists in both services —
   reuse it, do not add a second regex.
3. **Check the whole input, not just the first element.** `CourtIDs` is a slice
   (`internal/socialplay/app/service.go:168`); a malformed id anywhere in it must
   be rejected **before any court is reserved**, so no partial-state Game or
   Booking is left behind. This mirrors
   `TestScheduleGame_UnknownVenueFacilityRejectedBeforeReservingCourts` (T8.3),
   which is the pattern to copy.
4. **Assert no partial state**, not merely the error code: after a rejected
   call, no Game and no Booking exists.
5. **Sweep for siblings while you are here**, and report the result either way:
   are there other caller-supplied id *collections* on write paths with no shape
   check? If yes and they are out of scope, **open an issue** rather than
   widening (T13's A6, which the retro scored as half-taken — make it a full
   instance).
6. **Wave 1, and it precedes T14.4** on `internal/socialplay/app/service.go`
   (A14) — three tickets touch that file this sprint and the wave gaps are the
   mitigation.
7. Non-functional: TDD-first (a failing test per path first); mutation-check
   (remove the guard, confirm the new tests fail, restore) and record it.

**Cross-context dependency check:** `socialplay` and `competitions` `app`
services; no cross-context call (A12).

**Story points:** 2. **Role:** qa. **Type:** bug.
**Labels:** `role:qa`, `type:bug`, `context:socialplay`,
`context:competitions`.

---

### T14.9 — A local-dev auth fixture, so `make up` works again

**Story:** As a developer resuming this project, I want `make up` to start a
working server, so that step 4 of the documented resume path does what it says.

**Description:** #160, opened by T13.5 as a disclosed cost of a correct security
fix. Making a nil verifier fail startup was right, and T13.5 rightly **rejected
an opt-out flag** because it re-creates the fail-open path the ticket existed to
close. The consequence is that `make up` no longer starts without `AUTH_*` env
vars — and `make up` is **step 4 of `HANDOFF.md`'s own "First actions on
resume"**. Closes **#160**.

**Instructions:**

1. **Provide a real local-dev key fixture, not a bypass.** T13.5's reasoning
   binds this ticket: anything that lets the server start with no verifier
   re-opens the fail-open path. The fixture must supply **real key material** a
   real verifier really verifies, so the local path exercises the same code the
   deployed path does.
2. **Wire it through `docker-compose.yml`'s existing `AUTH_*` env surface**
   (T13.5 made the verifier configurable that way). **`cmd/server/main.go` is
   not in scope** — T14.4 owns it in the same wave (A14). If you believe a
   `cmd/server` change is required, that is a finding to disclose, not a licence.
3. **It must be unmistakably dev-only.** Name the key material so nobody can
   mistake it for a secret; document that it is committed on purpose and must
   never be used anywhere real; put it where a reader trips over that fact.
   Check it does not trip the security gate (`make security` / `vulngate`) or
   any secret scanner, and say which you checked.
4. **Prove the resume path end to end**: `make up` starts, and a `curl` from
   `README.md` succeeds against the running server with a fixture-minted token.
   Record the commands and output in the PR. **This is the ticket's actual
   acceptance test** — a fixture that exists but does not make `make up` work
   has not closed #160.
5. **Update the docs that are currently wrong**: `HANDOFF.md`'s "First actions
   on resume" step 4, and `README.md`'s `curl` examples if they now need a token.
   The gap this closes is documentation-shaped as much as code-shaped.
6. **State as a checked negative** whether the fixture changes anything about
   the deployed path. It must not. Fail-closed behaviour with no `AUTH_*`
   configured stays exactly as T13.5 shipped it — **prove that by mutation**:
   with the fixture absent, startup still fails.
7. Non-functional: no change to `internal/platform/auth`'s behaviour; all gates
   green.

**Cross-context dependency check:** `internal/platform/auth`'s existing config
surface plus `docker-compose.yml`; no context code (A12).

**Story points:** 3. **Role:** principal-engineer. **Type:** chore.
**Labels:** `role:principal-engineer`, `type:chore`, `context:platform`.

---

## Dependency order and dispatch waves (see A13 for the completeness check, A14 for isolation)

```
Wave 1 (parallel, ≤5 implementers, worktree-isolated):
  T14.2  gofmt gate + #165 fix            (Makefile, socialplay/domain/game.go) ─┐
  T14.3  ADR-0015 — #144's escalation     (docs/adr/0015 only)                   ─┤
  T14.6  process amendments               (sprint-process.md only)               ─┼─ independent
  T14.7  #158 error mapping, 3 contexts   ({b,sp,c}/adapter/grpcapi/handler.go)  ─┤
  T14.8  #156 court-id shape check        ({sp,c}/app/service.go)                ─┘

  No Wave-1.5 checkpoint this sprint — the condition was checked and does
  not fire (A10): T14.4 is the only new cross-cutting capability and it has
  ONE first-time in-sprint consumer, not three. Not generalising the
  checkpoint is recommendation 7's explicit ask.

Wave 2 (parallel, ≤3 implementers, worktree-isolated):
  T14.1  mechanical gate-coverage check   (needs T14.2's ci-checks line +
                                           gofmt-clean baseline)
  T14.4  Game-Admin durable store         (needs T14.7's corrected toStatus;
                                           needs T14.8's service.go edit;
                                           MUST NOT touch game.go)
  T14.9  local-dev auth fixture           (docker-compose only; NOT cmd/server)

Wave 3 (1 implementer):
  T14.5  resolve admins from the store, widen the roster read
  — needs T14.4 merged (A13 GAP A: consumes its read method)
```

Total: **39 points across 9 tickets** (T13: 40/9, T12: 46/9, T11: 47/9,
T10: 37/8).

**Deliberately at T13's size with a different weight distribution**, for stated
reasons rather than by accident: (i) the single largest item (T14.1, 8 points)
ships no user-visible behaviour, which A16 records as a live PM/PE disagreement
resolved in PE's favour; (ii) the sharpest authorization hole (#144) is
**blocked on the user**, so its 3 points buy an escalation rather than an
implementation — a deliberate refusal to guess, not a light ticket; (iii) one
ticket (T14.5) sits alone in Wave 3 because it consumes a store built in Wave 2,
which is ordinary sequencing and not a checkpoint.

**Wave roll-call is mandatory before ending any work block** (T12's A2, standing
practice, retained by T13's retro finding 6a on the explicit reasoning that *"the
thing the roll-call detects stopped happening, for a reason that is probably
sprint duration rather than the roll-call itself"*). The dispatch list above is
the list to roll-call against.

**Reopening trigger, carried from the retro rather than re-decided:** A9(a) is
**closed as unfalsifiable in practice** and struck from the carried-forward
list. It becomes live again on its own terms **if T14's work spans more than one
work block** — not as an agenda item someone must remember, but as a condition
anyone can recognise when it occurs.

## Explicitly deferred, not silently dropped

Per the board-of-record rule, **every item below that outlives this sprint has a
GitHub issue** — verified open this ceremony against the live issue list
(`totalCount: 19`).

- **#144 — `CancelBooking`/`CreateBooking` have no authorization check.**
  **Escalated, not implemented.** T14.3 writes ADR-0015 with three options and
  the question addressed to the user (A6). Verified this ceremony that the gap
  is exactly as described: `bookings` has no owner column, `CancelBooking` takes
  no actor, both RPCs are in `PublicMethods()`. **This is #144's second
  deferral, in a materially different form** — the first was a plan paragraph,
  this is an ADR with named options and a trigger. **A third deferral without an
  answer is a finding, not a decision.**
- **#168's Competitions half — Competition-Admin store.** T14.4/T14.5 take
  Social Play only; #168 stays open and both PRs say "partial fix". A16 records
  QA's dissent that two sprints of partial fixes is how an issue becomes
  permanent furniture, with a scoring condition for T14's retro.
- **#147's Competitions half — `ListEntriesForCompetition` stays Host-only.**
  Blocked on the same missing store. T14.5 must state this as a checked
  negative rather than let a reader infer the whole issue was closed.
- **#149 — Payments accepts caller-supplied ownership facts.** Not taken;
  **materially advanced** — its own named sub-gap (#168) has its Social Play half
  closed. Still needs read-side ports Payments does not have (re-verified:
  `internal/payments/port/` = `idgenerator.go`, `payment_processor.go`,
  `repository.go`).
- **#164 — ADR-0014 conformance backfill.** Deferred: needs a data backfill
  against identities a real IdP has not issued. **Actively honoured by T14.4**,
  which must match its neighbours' identifier space and cite §5a and #164 in the
  migration comment rather than quietly picking a third convention.
- **#167 — Stripe webhook receiver.** Blocked: needs a real Stripe account and a
  publicly reachable endpoint. Same class as the IdP tenant and the Jenkins
  server-side wiring.
- **#137 — no remote JWKS `KeySource`; #145 — uuid `owner_id` rows vs. a
  non-uuid subject.** Both blocked on a real identity provider. #145 was narrowed
  by T13.3, not closed.
- **#124 (Game-cancellation cascade), #125 (Competitions `PayableType`), #126
  (real per-Game price), #130 (refunding a `no_show_fee`).** Re-checked open;
  reasoning unchanged from T13's A16. #126 still needs PO sign-off before code.
- **#134 — the WCAG manual screen-reader pass.** Still **declined as
  unticketable in this environment**: a manual pass means driving NVDA/JAWS/
  VoiceOver and listening. No session here has assistive technology. Flagged for
  the user, since only a human can discharge it.
- **ADR-0012's Q1/Q2 work** (`PlayerRating`, matching algorithm, gender-mix
  matching). **Still blocked on the user**, checked this ceremony (A15) — and
  verified untouched in code, not merely un-discussed. Not deferred by this
  team's choice and not this team's to decide.
- **CI server-side wiring** (Jenkins job/webhook/branch protection). **Not
  ticketable**, unchanged since SCRUM-6. T13.4 closed the repo-side half (#129);
  T14.1 widens what that pipeline *would* run, and must say so honestly rather
  than implying CI now gates anything.
- **Observability (Sentry + slog + uptime); `golang-migrate`/`goose` swap;
  ISO-8601 weekday numbering; the T6.4 uncommitted payments concurrency proof.**
  Unchanged low-urgency infra debt. No issues opened: roadmap items, not
  disclosed gaps.

## `HANDOFF.md` updates this ceremony's PR carries

Per **recommendation 9** and `sprint-process.md`'s Ceremony 1 step, this PR
carries:

- **The T13 Docs-index row, corrected** — it currently reads **"not yet
  written"** for the Retro and lists only T13.2 under Reviews, both written
  before T13 ran. Replaced with `docs/process/t13-retro.md` and the real PR
  list **in `merged_at`-verified order** (below).
- **A new T14 row**, matching T10–T13's format.
- **T13's Task-backlog narrative entry**, written in **the retro's own agreed
  sentence** (`sprint-process.md` Ceremony 1 item 3 — *"in the form its own
  retro agreed, not a stronger one"*), with the closure sequence stated
  accurately: **shipped correctly, closed late, corrected same-day.**

**Merge order verified against each PR's `merged_at`, not assumed from
numbering** — this project's standing convention, because PR numbers and merge
order have disagreed before (T8, T10):

| # | PR | Ticket | `merged_at` (2026-08-15) |
|---|---|---|---|
| 1 | #155 | Ceremony 1/2 doc | 05:16:29Z |
| 2 | #159 | T13.9 | 05:33:24Z |
| 3 | #161 | T13.1 | 05:34:52Z |
| 4 | #162 | T13.4 | 05:37:55Z |
| 5 | #163 | T13.5 | 05:40:24Z |
| 6 | #166 | T13.2 — Wave-1.5 checkpoint | 05:43:02Z |
| 7 | #169 | T13.7 | 05:59:14Z |
| 8 | #170 | T13.3 | 06:01:01Z |
| 9 | #171 | T13.6 | 06:02:54Z |
| 10 | #172 | T13.8 | 06:19:55Z |
| 11 | #173 | retro doc | 06:35:58Z |

**Worth recording, since the convention exists because of the opposite
result:** this sprint's merge order and numeric order **agree**. That is the
finding, not an excuse to skip the check — it was only knowable by running it.

Ticket-level `HANDOFF.md` edits (T14.4's #168 partial-fix note, T14.9's resume-
step correction, T14.1's gate documentation) belong to their own tickets' PRs,
not this one.

## Definition of Done (sprint-level)

All 9 tickets merged per `sprint-process.md`'s per-ticket DoD (PR-only,
CLAUDE.md rule 9 — **no exception for this plan**, which lands the same way);
sprint goal met (every test-holding package executed by a gate, enforced
mechanically; a formatting gate in `ci-checks` with #165 fixed; a durable
Game-Admin store with Social Play's roster read widened to resolved admins;
#158/#131, #156, #160 closed; #144 escalated as ADR-0015 rather than guessed);
**no Wave-1.5 checkpoint required, having been checked and found not to fire**
(A10 — recorded so the retro scores the check, not its absence); **wave roll-call
performed before every work-block end**, with any "neither PR nor branch" ticket
named; **every review enumerating both the issues it opened and the issues it
closes, performing the closes before moving on** (T14.6's new rule, applied to
T14 itself); **every review checking label conformance** against T14.6's decided
taxonomy; every partial fix titled **"partial fix for #N"** (T13's A5); retro
held (`docs/process/t14-retro.md`) with findings indexed from
`docs/LESSONS.md`'s `## T14 sprint retro`; `HANDOFF.md` updated for T15 to
resume from.

**Three scorings this sprint's retro owes, stated now so they are not
improvised:**

1. **Recommendation 4's dual question must be DROPPED, not renewed**, if T14.1
   merged (A5). Carrying both the mechanical check and the planning question is
   this project's named failure mode, and PE called it in advance.
2. **A1's PE-vs-PO disagreement** on where the authoritative closure sweep
   belongs: if every T14 ticket that closes an issue closed it at review time,
   PE is right that the backstop should rarely fire; if the backstop fires
   again, PO is right.
3. **A16's QA-vs-PdE disagreement** on partial fixes: whether taking Social
   Play's half of #168/#147 built a mechanism the Competitions half can follow,
   or started a third sprint of an issue becoming permanent furniture.
