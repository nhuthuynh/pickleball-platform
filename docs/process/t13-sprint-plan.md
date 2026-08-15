# T13 Sprint Plan — Making the verified principal actually work end to end, and the process fixes T12's retro made concrete

Ceremonies 1 and 2 per `docs/process/sprint-process.md` (read in its
**T12.6-amended** form — the board-of-record split resolved at T12's Ceremony 1
is the version this plan is governed by), six-role team (briefs:
`docs/agent-operating-handbook.md` Part B), against `HANDOFF.md`,
`docs/process/t12-sprint-plan.md` (whose A2–A14 sections are this document's
structural template), and `docs/process/t12-retro.md`'s **six findings and ten
recommendations** — threaded into this plan below, one subsection per item,
adopted or adapted or explicitly declined, never silently dropped.

**Every claim in this plan that could be checked against the working tree was
checked this ceremony**, per T12 retro recommendation 3, which extends T12's
A6 evidence-marking discipline from the dependency table to *any factual claim
about existing code in a ticket's Instructions*. Where a ticket asserts
something about existing code, this plan either states how it was verified or
phrases it as an instruction to verify. Commands are recorded inline so a
reader can re-run them rather than trust them.

**Three of this ceremony's checks changed the plan materially**, and one of
them found a live defect nobody was looking for:

1. Running recommendation 1's new dependency-completeness check against the
   T12.7 → Facilities-persistence arrow surfaced an **untracked, live break**:
   `CreateFacility` writes a verified subject into a `uuid NOT NULL` column
   through a helper that panics. Opened as **#154** (A2).
2. The same check found a **capability with no producer** in this sprint's own
   graph — Facilities has no port into Identity — and assigned it to exactly
   one ticket before dispatch (A13).
3. Issue **#145 contains a false scope claim** that actively directed readers
   away from (1). Corrected by comment on the issue, not silently (A2).

---

# Part A — Ceremony 1 (backlog refinement)

**Product Manager + Principal Engineer**, per `sprint-process.md`. PM drove
scope/value framing; PE drove technical sequencing/feasibility. Both sign off
on every ticket below.

## A0. What this ceremony inherits — stated in the retro's own honest sentence, not the sprint goal's

Per **recommendation 8**, T12's outcome is recorded here in the precise form
finding 4 proposed, because overclaiming on this specific subject is worse than
elsewhere: *a future ticket that believes authorization is finished will not go
looking for #149.*

> The verified-principal **mechanism** exists, is real, is tested, and is
> consumed by all six bounded contexts: 24 RPCs resolve their actor from a
> verified token subject, the wire `actor_*` fields are ignored (proven by
> mutation), a new RPC cannot become silently public, and the `CreateUser`
> squatting DoS is closed. What does **not** yet hold is the stronger claim:
> several RPCs still have no authorization check at all (#144, #147, #148),
> Payments still compares a verified actor against caller-supplied ownership
> facts (#149), one migrated capability is non-functional (#146/#152), and no
> token from a real identity provider can be verified until a remote JWKS
> source exists (#137). Eleven tracked exceptions, every one of them with an
> issue.

**T13 is the sprint that stops treating those eleven as a permanent state.**
That framing is recommendation 7's actual ask, and A1 executes it.

T12 also left one defect on the shared branch — the first since T10.
`RequestRecurringHire` is non-functional for every caller. Verified still
broken this ceremony (A2). **T13 does not inherit a clean branch**, and the
first ticket in the dispatch order reflects that.

## A1. The eleven residual auth issues, ranked — recommendation 7 executed

Recommendation 7 asks this ceremony to carry T12's residual auth work as *a
named, prioritised group, not 11 loose issues*, and to state which T13 takes.

**All 19 issues opened during T12 were re-checked against the GitHub API this
ceremony** (`list_issues(state: OPEN)` → `totalCount: 19`, newest #152).
**None has been closed since T12 ended.** Per T12 retro's warning that stale
ticket-claims are a recurring risk here, this was re-derived rather than read
off `HANDOFF.md`.

**Correction to the retro, recorded rather than smoothed over.** Finding 6
states that all 19 issues "carry `role:*`/`type:*` labels per T12.6's
taxonomy." Checked: **#146, #147, #148 and #149 carry no labels at all** —
the four T12.8 opened, i.e. precisely the ones the *implementer* filed rather
than the reviewer. That does not weaken finding 6's conclusion; it sharpens it,
since the labelling was part of what the reviewer-side backstop was supplying.
Labelling them is folded into A6's review practice rather than ticketed.

### Ranking

| Rank | Issue(s) | Why here | T13? |
|---|---|---|---|
| **1** | **#146 / #152** | A shipped capability is **non-functional on the shared branch right now**. Not a missing check — a live break. #152 is the real remaining fix. | **Taken — T13.2** |
| **2** | **#154** *(new, this ceremony)* | Same class as #152, second instance, **and it panics** rather than returning cleanly. Found by A13's check; see A2. | **Taken — T13.3** |
| **3** | **#138** | The auth spine's own tests run in **no** Docker-free gate. The single most load-bearing package in the codebase is ungated. Cheap to fix. | **Taken — T13.4** |
| **4** | **#148** | `ConfirmOnlinePayment` has no owner check — anyone holding a `payment_id` can capture. Money-adjacent, and needs no domain change. | **Taken — T13.7** |
| **5** | **#147** | Roster reads leak every registrant/entrant to anyone holding an id. Privacy, not money; no domain change needed. | **Taken — T13.6** |
| **6** | **#136 / #135** | Auth mechanism hardening: a nil verifier should fail startup; a verifier panic should fail **closed**. Small, and they are the difference between "auth is on" and "auth is on unless something went wrong." | **Taken — T13.5** |
| **7** | **#144** | `CancelBooking` has no authorization check. Real and sharp — but needs an owner concept on the Booking aggregate, a migration, **and a product decision** the issue itself names (what "owns" a booking made through T7.6's public quote-and-book flow). | **Deferred — reasoned in "Explicitly deferred"** |
| **8** | **#149** | Fact-fabrication: verified actor compared against caller-supplied ownership facts. Structural — needs read-side ports from Payments into three contexts, and its own text says the Game-Admin persistence sub-gap "should probably be closed first." Payments has **no** cross-context read ports today (verified: `internal/payments/port/` = `idgenerator.go`, `payment_processor.go`, `repository.go`). | **Deferred — reasoned** |
| **9** | **#145** | Pre-existing uuid `owner_id` rows vs. a non-uuid subject. Genuinely blocked on a real IdP existing. **Narrowed** by T13.3, not closed. | **Partially addressed; stays open** |
| **10** | **#137** | No remote JWKS `KeySource`. Blocked on an IdP nothing here can provision — the same class as the Jenkins server-side gap. | **Deferred — blocked, not chosen** |

**PM/PE scoping decision: T13 takes ranks 1–6 (eight of the eleven issues),
defers 7–8 with stated reasoning, and records 9–10 as genuinely blocked.**
That is six of the eleven closed outright, two narrowed, and the "eleven
tracked exceptions" number reduced rather than carried forward untouched —
which is the outcome recommendation 7 exists to force.

## A2. Threading finding 1 / recommendation 1 — the dependency-completeness check, and the defect it found before dispatch

**Adopted in full, and it is this plan's marquee change.** Finding 1
established that T12.7 and T12.9 independently built competing implementations
of the same missing capability, 68 seconds apart, because **no ticket owned it**
— and that no file-overlap question, asked of new files or existing ones, can
reach that. It is a *provenance* failure, not a *visibility* one.

The check, as the retro specifies it: **for every arrow in the wave dependency
graph, state in one line what the upstream ticket's AC delivers and what each
downstream ticket's AC consumes, and name any capability that appears on a
consumer's side and no producer's side.** Any such capability is assigned to
exactly one ticket before dispatch.

**The table is A13.** It is a full section rather than a note because it is the
retro's highest-value recommendation and this ceremony wants it to be
re-runnable, not admired.

### It paid before the sprint started

Running the check against the *inherited* arrow — T12.7's "the actor is now a
verified subject" → each context's persistence layer — produced a real find.

**Verified this ceremony, by reading the files:**

- `internal/facilities/adapter/grpcapi/handler.go` — `CreateFacility` sets
  `OwnerID: ownerID` where `ownerID, err := actor(ctx)`, and `actor` is
  `return auth.RequireSubject(ctx)`. The owner is an **IdP subject**.
- `db/migrations/0010_facilities.sql:20` — `owner_id uuid NOT NULL`.
- `internal/facilities/adapter/postgres/repository.go:33` —
  `OwnerID: mustUUID(f.OwnerID)`.
- `repository.go:220-230` — `mustUUID` **panics** on a value that does not
  scan as a uuid.

So `CreateFacility` cannot write a row for a real IdP caller; it panics, and
the recovery interceptor turns that into `Internal`. **Opened as #154.**

**This was not covered by any of the eleven.** #145 explicitly excludes it:
*"`CreateFacility` was changed in T12.7 to mint `owner_id` directly from the
principal, so newly-created Facilities after a real IdP exists are
self-consistent."* That sentence is false, and it is the reason the gap went
unnoticed — a reader who trusts it stops looking. A correcting comment was
posted on #145 this ceremony pointing at #154.

**Noted for T13's retro, not scored here:** this is T12 finding 5's exact
signature (an unmarked factual claim about existing code, which turned out to
be false) occurring one level further out — in a GitHub issue body rather than
a sprint plan. Recommendation 3 currently binds *plan Instructions*. Whether it
should also bind issue bodies is a real question, and A16 carries it forward
rather than deciding it here.

### Blast radius, measured

Every actor-carrying column was enumerated and filtered to `uuid`. There are
exactly **two** seams where a verified subject can reach a uuid column:

| Column | Writer | Status |
|---|---|---|
| `recurring_hire_templates.requested_by_user_id uuid NOT NULL REFERENCES identity_users (id)` (`0018:32`) | `RequestRecurringHire` | #152 → **T13.2** |
| `facilities.owner_id uuid NOT NULL` (`0010:20`) | `CreateFacility` | #154 → **T13.3** |

**Checked-and-does-not-fire, stated rather than left to inference:** Social
Play (`0005_socialplay.sql:4`) and Competitions (`0014_competitions.sql:18`)
store `host_id`/`player_id` as **plain text** by deliberate design — their own
migration comments say so. They cannot hit this. That is why T13.6 and T13.7
are authorization tickets rather than translation tickets.

## A3. Threading finding 2 / recommendation 2 — the five-package behavioural-test backfill

**Adopted in full as T13.1.** Finding 2's measurement was re-verified against
the working tree this ceremony rather than carried over, by listing every
`internal/*/adapter/*/` directory and running `wc -l` on each `_test.go`:

| Cross-context adapter | Test files today | Verified |
|---|---|---|
| `booking/adapter/facilities` | `lookup_test.go` — **12 lines** | compile-time `var _ port.FacilityLookup = (*Lookup)(nil)` only; file read in full |
| `competitions/adapter/facilities` | `lookup_test.go` — **12 lines** | compile-time assertion only |
| `competitions/adapter/booking` | `reservation_test.go` — **25 lines** | compile-time assertion only |
| `socialplay/adapter/booking` | **none** — directory contains only `reservation.go` | `ls` |
| `socialplay/adapter/facilities` | **none** — directory contains only `lookup.go` | `ls` |

The retro's count of five holds exactly. The pattern to follow also holds:
`payments/adapter/socialplay/registration_updater_test.go` is **186 lines** and
drives the real `socialplayapp.Service` over in-memory repository fakes,
Docker-free.

**One refinement to the retro's instruction, made on evidence rather than
preference.** Recommendation 2 says each test *"must use a realistically-shaped
identifier — a non-uuid subject where the seam carries one."* Checked: of the
five packages, the only one whose surface carries an actor identifier at all is
`booking/adapter/facilities` (`EnsureFacilityOwner(ctx, facilityID,
actorUserID string)`). The other four carry court/facility/game ids only. So
the instruction is preserved with its condition made explicit — *where the seam
carries an actor, use a subject; where it does not, use uuid-shaped ids,
because the production code paths guard on `uuidShape`, and state that as a
checked negative.* Writing "use a non-uuid subject" unconditionally would
produce four tests exercising a path production never takes.

**There is real behaviour to test here, not just wiring** — which is what makes
this a 5-point ticket rather than busywork. Every one of these adapters exists
to translate another context's errors into its own, deliberately using `%s`
rather than `%w` so a caller cannot `errors.Is` across the boundary
(CLAUDE.md rule 5). Read directly in
`socialplay/adapter/booking/reservation.go` and
`socialplay/adapter/facilities/lookup.go`. That translation is exactly what a
mutation check can falsify, and exactly what a compile-time assertion cannot
see.

## A4. Threading finding 5 / recommendation 3 — evidence-marking extends to ticket Instructions

**Adopted in full, and applied throughout Part B.** T12.4's Instructions
asserted that `ErrIllegalStatusTransition` already mapped to
`FailedPrecondition`; it does not (it is grouped with the `InvalidArgument`
cases). The claim was false, and only the *second half* of the same sentence —
"check the existing mapping before adding one" — made it harmless.

**The rule this plan binds itself to:** every ticket below either

- marks a factual claim about existing code with **what was checked and how**
  (a path, a line number, a command run this ceremony), or
- phrases it as an **instruction to verify** rather than a stated fact.

A plan may say *"confirm which code `ErrX` maps to before relying on it"*; it
may not assert a mapping it has not read. Where this ceremony did read
something, the ticket says so and names the file, so an implementer knows the
difference between a checked premise and a hint.

**Deliberate application:** T13.9 (`#131`, error mapping) is the ticket most
exposed to this failure mode, since its whole subject *is* a sentinel-to-code
mapping. Its Instructions therefore assert nothing about the current mapping
and instruct the implementer to derive it from `toStatus` directly.

## A5. Threading finding 3 / recommendation 4 — "partial fix for #N", never "closes #N"

**Adopted in full as a standing rule for this sprint, binding on every PR.**

PR #151's title claimed *"closes #146"* for a fix that resolved one of two
independent causes; #146 was then correctly left open. Anyone reconstructing
history from merge titles — which is exactly how `HANDOFF.md`'s Docs-index rows
get built — would conclude the capability works. It does not.

**The rule:** a PR that fixes part of an issue says **"partial fix for #N"** in
its title *and* body, and names what remains. This is consistent with
`sprint-process.md` DoD step 5, which already makes closing a manual judgement
on this branch topology; the rule stops the PR headline from pre-empting that
judgement.

**T13.3 is the ticket most likely to need it**: it closes #154 and *narrows*
#145 without closing it (the pre-existing-rows half stays blocked on a real
IdP). Its Instructions say so explicitly.

## A6. Threading finding 6 / recommendation 5 — every review enumerates the issues it opened

**Adopted in full as a coordinating-session review practice, not a ticket** —
correctly, since no implementer performs it.

Finding 6 recorded a genuine first: 19 disclosed gaps, 19 tracked issues, no
prose-only deferral. But the mechanism was a **reviewer-side backstop one
session deep**, not the PR-side rule A5 wrote — 18 of 19 issues were created by
the reviewer, not the disclosing PR. Because the outcome was perfect, nothing
in the record signalled the dependency.

**The rule, which both PO and BA agreed on regardless of how the
who-is-primary disagreement resolves:**

> Every review of a T13 PR **enumerates the issues it opened for that PR's
> disclosures, or states explicitly that there were none.**

That single line makes the backstop auditable rather than incidental. The
PO/BA disagreement about whether the *implementer* or the *reviewer* should be
primary is **not** resolved here — it is left open and carried to T13's retro
(A16), because this ceremony has no new evidence and manufacturing a decision
would be exactly the smoothing-over `sprint-process.md` forbids.

**Small addition, from A1's finding:** the enumeration must include labels.
Four T12 issues (#146–#149) carry none, and they are the four an implementer
opened — evidence that the labelling has been riding on the same single
session's attention as the issue-opening.

## A7. Threading finding 5 / recommendation 6 — `make vet-integration` becomes an explicit review line, and the CLAUDE.md prose half is dropped

**Adopted exactly as the retro scoped it, including the part that drops
something.**

A9(b) is **resolved and closed**. The evidence: one of eight T12 PRs (T12.4 —
touching the context with the *most* integration-tagged files) had no record of
running the check; the reviewer ran it independently and found nothing broken.
QA's premise was confirmed at 1-in-8; PE's conclusion (one executable gate
beats three prose reminders) survived.

**Adopted:** *"`make vet-integration` run — by implementer or reviewer"* is an
explicit line in every T13 review. It makes the 1-in-8 cover visible rather
than incidental, and it is enforcement at the point of decision rather than
prose.

**Explicitly dropped, and named here so it is not re-carried a fourth time:**
the older CLAUDE.md-prose half of the T11 disagreement — a Gotchas reminder
plus a per-ticket DoD item. The retro drops it because the sprint's evidence
did not support it. **This plan does not re-open it, and T13's retro should not
re-record it.** QA's revised position (drop the prose, take the review line) is
the position adopted.

Related, and taken as a ticket rather than a note: **#129** records that the
`Jenkinsfile` does not call `make ci`, so `vet-integration` never actually
reaches CI even now. A review line covers the human path; #129 is the machine
path. Folded into **T13.4**.

## A8. Threading finding 4 / recommendation 8 — the honest sentence, everywhere

**Adopted in full.** A0 carries the retro's sentence verbatim. This plan's
sprint goal (Part B) is written in the same register — it claims a *specific,
countable* outcome and names what remains, rather than a sweeping one.

**The `HANDOFF.md` edits this ceremony's PR carries use that sentence too**,
replacing any "every authorization check" phrasing. Language matters here
because the failure mode is a future ticket assuming authorization is finished.

**PE note, from finding 4's own correction of A1:** T12's A1 framed the
IdP-shaped residue as purely a *hosting* gap, and that did not follow. Three of
the eleven residuals (#137, #145, #152) are **repo-side code consequences** of
not having an IdP, not server-side work. T13 proves the point by building two
of them (#152 → T13.2, and #154, its sibling → T13.3) with no IdP in sight.
"The server-side half is out of reach" was true; "therefore the remaining work
is server-side" was not.

## A9. Threading recommendation 9 — the stale Docs-index row, and the structural fix

Recommendation 9 asks for two things and is explicit that fixing the row alone
is not one of them: *"Either the retro PR takes the row too, or
`sprint-process.md`'s Ceremony 1 gains an explicit 'correct the previous
sprint's Docs-index row' step. Pick one; it has now silently recurred twice."*

**Verified this ceremony:** `HANDOFF.md`'s T12 row (line 32) still reads
**"not yet written"** for the retro and **"not yet opened"** for the reviews.
Both are stale — the retro merged as PR #153 and nine tickets merged as PRs
#128–#150.

**This PR does both halves, deliberately:**

1. **The immediate correction** — the T12 row is rewritten in this PR, with the
   real PR numbers and merge order.
2. **The structural fix — option (b): `sprint-process.md`'s Ceremony 1 gains an
   explicit step.** Amended in this PR.

**Why option (b) over option (a) (retro PRs take the row), stated because the
retro left the choice open.** Three reasons, and the third is the one that
decides it:

- (a) fights the precedent rather than using it. Verified: PRs #110, #122 and
  #153 each touched exactly `docs/LESSONS.md` and the retro doc. That is a
  *stable, deliberate* convention — a retro PR is the retro's record, and
  widening its diff to carry unrelated index maintenance is how conventions
  erode.
- Ceremony 1 already re-reads the Docs index. CLAUDE.md's "Docs index & naming
  convention" section says to read it *before starting any task*, and every
  sprint since T11 has in practice discovered the stale row there. Option (b)
  puts the correction where someone is already looking, rather than asking a
  different ceremony to remember.
- **The decisive one: a retro PR structurally cannot write the row correctly.**
  The row must cite the retro's own merge PR number, which does not exist until
  that PR is merged. #153's row would have had to say "PR #153" while being
  PR #153, before GitHub assigned it. Ceremony 1, by contrast, runs after
  everything from the previous sprint is merged and every number is knowable.
  Option (a) is not merely less convenient — it asks a document to cite a fact
  that has not happened yet.

The amendment is small and lands in Ceremony 1's own section, phrased so it
covers the general case rather than the T12 row specifically.

## A10. Threading recommendation 10 — the two carried-forward open questions

**(a) A9(a)'s roll-call-vs-polling disagreement. Carried into T13 with a
closing condition, not re-recorded as an open question.**

T12 could not score it: all 9 dispatched tickets produced a PR, and the sprint
ran in one unbroken 1h47m block, so the condition that produced T11's failure
never arose. The retro is blunt that this is *weak* evidence — a sprint with no
silent agent does not test a mechanism for detecting silent agents — and
recommends that if T13 also runs unbroken, the question be closed as
**unfalsifiable in practice** rather than recorded a fourth time.

**This ceremony adopts that, and states it as a binding instruction to T13's
own retro rather than a judgement made now** (the evidence does not exist at
ceremony time, obviously):

> **T13's Ceremony 3 must close A9(a) one of three ways and may not defer it a
> fourth time.** If T13 had a silent agent the roll-call caught → closed,
> roll-call sufficient. If T13 had a silent agent the roll-call missed →
> polling is adopted, with a concrete failure to design against. If T13 ran
> unbroken with no silent agent → **closed as unfalsifiable in practice**, per
> this recommendation, and struck from the carried-forward list.

The wave roll-call itself (T12's A2) remains standing practice for T13 and is
restated in the Definition of Done.

**(b) QA's port-contract-change rule. Carried, deliberately, to be scored
against evidence — and T13 generates exactly that evidence.**

Finding 2 left open whether a plan-level rule is needed on top of
recommendation 1's dependency-completeness check: *a ticket that changes an
identifier's meaning or a port's contract must enumerate every adapter on that
seam in its PR body.* PE's counter was that this is finding 1's rule wearing
different clothes, and that this project has a bad habit of adopting the same
fix twice in two shapes.

**T13 is an unusually good test of it**, and that is not a coincidence — T13.2
*is* a port-contract change (`port.IdentityLookup` gains subject resolution) of
precisely the shape that caused #146. So:

> **Scoring instruction for T13's retro:** did A13's dependency-completeness
> check, applied to T13.2's arrows, catch the port-contract-change shape on its
> own — or did something only a dedicated adapter-enumeration rule would have
> caught slip through? Score it against T13.2 and T13.3 specifically.

**Recorded now, since it is already partial evidence:** A13's check *did* catch
one capability of exactly this shape before dispatch (Facilities has no
Identity port — A13 Gap 1). That is one data point in PE's favour, and it is
logged rather than treated as settling the question.

## A11. A14's scored lesson — the Wave-1.5 checkpoint applies this sprint, and here is why

T12's retro scored A14's PdE/PE disagreement with real evidence and concluded
**PdE was substantively right**, in a specific and narrow form:

> T13 should apply the Wave-1.5 shape specifically where **a new platform
> capability has three or more first-time consumers** — not as a general rule,
> which is the over-correction PE correctly resisted.

**Checked against T13's shape: the condition fires.** T13.2 produces a new
cross-cutting capability — ADR-0014's ruling on *which identifier space
persisted actor columns hold*, plus the resolution seam that implements it —
and it has **four** first-time consumers:

| Consumer | What it consumes from T13.2 |
|---|---|
| T13.3 | The ruling, applied to `facilities.owner_id`; plus the port+adapter shape to copy |
| T13.6 | Whether `games.host_id` / `competitions.host_id` hold subjects or uuids — its ownership comparison is wrong if it assumes the other one |
| T13.7 | Same question for the Payments ownership comparison |
| T13.8 | Booking's final constructor shape, after T13.2 may add a dependency |

Three of those are *semantic* consumers, not merely file-adjacent ones. That is
the condition the retro named, met without forcing.

**Adopted: a Wave-1.5 checkpoint. T13.2 must be merged and reviewed before
Wave 2 is dispatched at all.** Not merely "started first" — merged. The cost is
a serialised sprint shape; the benefit is that the four consumers read a
decision that has survived review rather than four implementers each inferring
one, which is the precise mechanism that produced finding 1.

**PE's original counter-argument is not discarded, and does not apply here.**
PE resisted serialisation in T12 partly because it would have put the DoS fix
behind two merges *for schedule reasons rather than technical ones*. T13 has no
equivalent: nothing in Wave 2 is more urgent than T13.2, because T13.2 *is* the
live break. The checkpoint costs this sprint nothing it wants.

## A12. Cross-context dependency check (every cell evidence-marked, per recommendation 3)

| Ticket | Calls into | Member/port exists? | How this was checked |
|---|---|---|---|
| T13.1 (adapter test backfill) | drives the **real** other-context `app.Service` in five adapters | Yes — all five adapters already import and call them; the tests are what is missing | `ls` on all 22 `internal/*/adapter/*/` dirs; `wc -l` on each `_test.go`; read `socialplay/adapter/{booking,facilities}` source in full |
| T13.2 (subject↔User.ID seam) | `internal/identity` via `internal/booking/port.IdentityLookup` | `identityapp.Service.UserBySubject` **exists** (`internal/identity/app/service.go:153`, added T12.9). `port.IdentityLookup` has **only** `EnsureClubRole` (`identity_lookup.go:50`) — no method returns a `User` or `User.ID`, by documented design. **The port must change.** | Read both files; `grep -n "UserBySubject\|func (s \*Service)" internal/identity/app/service.go` |
| T13.3 (Facilities owner seam) | `internal/identity` — **no port exists** | **Negative finding, load-bearing:** `internal/facilities/port/` contains exactly `idgenerator.go` and `repository.go`. Facilities has never called another context. A new port **and** adapter are required. See A13 Gap 1. | `ls internal/facilities/port/` |
| T13.4 (CI gate) | nothing — `Makefile` + `Jenkinsfile` | n/a | Read `Makefile` `test-domain`/`ci` targets directly (lines 1–20, 145–155) |
| T13.5 (auth mechanism hardening) | nothing — `internal/platform/auth` + `cmd/server` | Package exists: `auth_test.go, chain_test.go, interceptor.go, principal.go, require.go, require_test.go, rs256/, verifier.go` | `ls internal/platform/auth/` |
| T13.6 (roster read authz) | none — `internal/socialplay` and `internal/competitions` only | Host identity is stored **as text** in both, so no cross-context resolution is needed | `grep` for actor columns across `db/migrations/*.sql`; `0005_socialplay.sql:4` and `0014_competitions.sql:18` both state it in comments |
| T13.7 (`ConfirmOnlinePayment` owner check) | none **that exists** — and that is the constraint | **Checked negative:** `internal/payments/port/` = `idgenerator.go`, `payment_processor.go`, `repository.go`. Payments has no read path into other contexts, which is exactly why #149 is deferred and why this ticket must scope to facts Payments already holds | `ls internal/payments/port/` |
| T13.8 (`ServiceOptions`) | nothing new — mechanical constructor refactor | `booking` and `socialplay` `app.NewService` are positional; `competitions`/`payments` already use an options struct (the shape to copy) | `HANDOFF.md` Cross-cutting + #123; to be re-verified by the implementer against current signatures, since T13.2 may add a dependency (A13 Gap 3) |
| T13.9 (`#131` error mapping) | none — `internal/payments/adapter/grpcapi` only | **This ticket deliberately asserts nothing about the current mapping** — see A4. The implementer derives it from `toStatus` | Existence of the inconsistency is taken from #131 and T12.4's independent corroboration, not re-derived here |

**No gap of the T9.6/T9.7 `PayableType` class was found hiding in these
calls.** The one genuinely new dependency direction — **Facilities →
Identity** — is new to the context map and is called out as such in A13 rather
than absorbed silently.

## A13. Dependency-completeness check (recommendation 1's new artifact)

For every arrow in the wave dependency graph: what the upstream ticket's AC
**delivers**, what the downstream ticket's AC **consumes**, and whether any
capability appears on a consumer's side with no producer.

| # | Arrow | Upstream AC delivers | Downstream AC consumes | Complete? |
|---|---|---|---|---|
| 1 | **T13.2 → T13.3** | ADR-0014's ruling on the identifier space of persisted actor columns; `port.IdentityLookup` extended with subject→`User.ID` resolution in **Booking**; the handler-boundary translation precedent | The same ruling, applied to `facilities.owner_id`; a way to resolve a subject to a `User.ID` **from inside Facilities** | ❌ **GAP 1 — see below** |
| 2 | **T13.2 → T13.6** | ADR-0014 stating, for **every** context, whether stored actor facts hold subjects or uuids | Which identifier space `games.host_id` / `competitions.host_id` hold, so the ownership comparison is against the right thing | ❌ **GAP 2 — see below** |
| 3 | **T13.2 → T13.7** | Same as arrow 2 | Which identifier space `payments` ownership facts hold | ❌ **GAP 2** (same gap) |
| 4 | **T13.2 → T13.8** | Booking's `app.Service` dependency list **after** T13.2 (it may gain an identity-resolution dependency) | The final constructor parameter set to fold into `ServiceOptions` | ❌ **GAP 3 — see below** |
| 5 | **T13.6 → T13.8** | Social Play's `app.Service` dependency list after the roster-authz change | Same | ✅ Complete — T13.6 adds no constructor dependency (authorization uses facts the service already holds; stated as a constraint in T13.6's Instructions) |
| 6 | **T13.9 → T13.7** | A consistent `toStatus` mapping in `payments/adapter/grpcapi/handler.go` | A `PermissionDenied` mapping for its new owner-check sentinel | ✅ Complete — but a **file-level** overlap, handled by wave separation (A14) |
| 7 | **T13.4 → (all)** | `internal/platform/**` covered by a Docker-free gate | Nothing consumes it; it is a gate, not a capability | ✅ Complete — no arrow, listed for completeness |
| 8 | **T13.5 → (none)** | Startup-time verifier validation + fail-closed panic behaviour | Nothing in T13 consumes it | ✅ Complete — deliberately independent |

### The three gaps, each assigned to exactly one ticket before dispatch

**GAP 1 — "resolve a subject to a `User.ID` from inside Facilities" has a
consumer and no producer.** T13.3 needs it; T13.2 delivers it **only for
Booking**, because `port.IdentityLookup` is Booking's port and Facilities has
none (verified: `internal/facilities/port/` = `idgenerator.go`,
`repository.go`). This is finding 1's exact shape — a capability every
consumer needs and no ticket owns. **Assigned to T13.3**, not T13.2: the port
belongs to the consuming context by this project's own convention (each context
declares its own primitive-typed port and implements it once — see
`internal/booking/port/facility_lookup.go`'s doc comment on precisely this).
T13.3's AC therefore explicitly includes building
`internal/facilities/port.IdentityLookup` + `internal/facilities/adapter/identity`.
**Stated up front so two tickets do not build it 68 seconds apart.**

**GAP 2 — "which identifier space do stored actor facts hold, per context" has
three consumers (T13.3, T13.6, T13.7) and no producer.** T13.2's natural scope
is Booking only. **Assigned to T13.2**, whose AC is widened accordingly:
ADR-0014 must rule for **all six contexts**, including the ones that need no
code change, and must state the checked negative for Social Play and
Competitions (text columns — verified in A2). Without this, T13.6 and T13.7
each guess, and a wrong guess is a silently-broken authorization check —
the worst possible failure for this sprint's subject matter.

**GAP 3 — "Booking's final constructor shape" has a consumer (T13.8) and a
producer whose output is not yet known.** T13.2 may or may not add a dependency
to `booking.app.Service`. **Assigned to T13.2**: its AC requires it to state,
in its PR body, whether it changed the constructor signature and what the final
parameter list is. T13.8 is placed in Wave 3 and instructed to re-derive the
signature from the merged code rather than from this plan (per A4 — the plan
cannot know it yet, so it must not assert it).

**Every capability on a consumer's side now has exactly one producer.** That is
the check's exit condition, and it is met.

## A14. Migration-number and shared-file pre-assignment

**Migration tip verified by listing the directory this ceremony**, not inferred
from T12's plan: `db/migrations/` ends at **`0019_identity_subject.sql`**.
(The directory still carries T6's duplicate-`0005` scar — this is what
pre-assignment prevents.)

**At most one T13 ticket adds a migration, and it is conditional:**

| Migration | Ticket | Condition |
|---|---|---|
| `0020_facilities_owner_subject.sql` | **T13.3** | **Fires only if** ADR-0014 (T13.2) chooses to *widen* actor columns to text rather than *translate* subjects to `User.ID`. If it chooses translation, T13.3 adds **no** migration and `0020` stays unclaimed for T14. |

**No other T13 ticket may claim `0020`.** Stated as a hedged pre-assignment
rather than a prediction, following T11's A10 precedent — the honest position
is that the number's owner is known even though whether it is used is not.

**Checked-and-does-not-fire, stated rather than left to inference:**

- **T13.2 needs no migration.** `identity_users.subject text NOT NULL UNIQUE`
  already exists (`0019_identity_subject.sql`, T12.9), which is the entire
  storage requirement for subject→`User.ID` resolution. Verified by reading the
  migration list, not by assuming T12.9 built it.
- **T13.6 / T13.7 need no migration.** They add authorization checks over facts
  already stored.
- **T13.8 / T13.9 / T13.4 / T13.5 / T13.1** touch no schema.

### Shared *existing* files, and who appends to each

| Existing file | Tickets | Same wave? | Rule stated up front |
|---|---|---|---|
| `internal/payments/adapter/grpcapi/handler.go` | T13.9 (W1, `toStatus`), T13.7 (W2, owner check) | **No** — wave-separated | T13.7 branches from a base already containing T13.9's mapping changes, and adds its new sentinel's mapping into the corrected `toStatus`. Flagged rather than assumed. |
| `internal/booking/app/**` | T13.2 (W1), T13.8 (W3) | **No** | T13.8 re-derives the constructor from merged code (A13 Gap 3). |
| `internal/socialplay/app/service.go` | T13.6 (W2), T13.8 (W3) | **No** | Same. |
| `cmd/server/main.go` | T13.5 (W1, verifier startup check), T13.3 (W2, wires the new Facilities identity adapter), T13.8 (W3, constructor call sites) | **No — all three in different waves** | Each appends independently; the wave gaps are the mitigation. Checked explicitly because this is the file finding 1's collision lived next to. |
| `internal/platform/auth/**` | **T13.5 only** | n/a | **Checked, does not fire.** T13.2 performs its translation at Booking's grpcapi boundary and does **not** touch the auth package — the constraint A11 Ruling 3 established in T12 and this plan preserves. Named explicitly because this is the exact package T12.7/T12.9 collided in. |
| `Makefile`, `Jenkinsfile` | **T13.4 only** | n/a | |
| `docs/adr/0014-*.md` | **T13.2 only** | n/a | ADR number verified: `docs/adr/` ends at `0013`. |
| `internal/facilities/port/`, `internal/facilities/adapter/identity/` | **T13.3 only** | n/a | New files, single owner, assigned in A13 Gap 1. |

**No two tickets in the same wave create the same new file, and no two tickets
in the same wave append to the same existing file.** Both file-overlap
questions asked, plus A13's capability question — which is the full set finding
1 asked for.

## A15. ADR-0012's Q1/Q2 remain blocked — named, not touched, not dropped

`docs/adr/0012-*.md` blocks `PlayerRating`, the matching algorithm, and
gender-mix matching on two questions escalated to the user: **Q1** (the Player
Level formula's weighting) and **Q2** (whether gender-mix matching is in scope
at all, given it means collecting and algorithmically acting on a protected
attribute). Its trigger is explicit: *the sprint immediately following the
user's answers to both* — the user answering, not a sprint boundary.

**Checked this ceremony: no answer to either question exists.** T13 therefore
builds none of it, and this ceremony makes no product call on either — they are
not this team's to make. Recorded exactly as T10's, T11's and T12's plans
recorded it, so it is neither silently dropped nor silently decided.
ADR-0012's constraint that *"no PR may add a `Gender` field, a `PlayerRating`
type, or any Level-scoring formula"* binds every T13 ticket unchanged.

**One genuine interaction, stated because it would otherwise be discovered
mid-ticket:** T13.2 changes how a `User` is looked up (by subject rather than
by id) and T13.8 refactors Social Play's service constructor. Neither may take
the opportunity to add profile fields or a rating type.
`SelfReportedStartingLevel` keeps its existing shape and meaning.

## A16. The rest of the backlog, re-checked rather than inherited

Per the standing risk that stale ticket-claims survive in `HANDOFF.md`, each
known thread was re-verified against current code and the live issue list.

- **#123 (`ServiceOptions` migration).** Still open. **Now more buildable, and
  taken as T13.8** — T12's own plan deferred it with a stated timing
  condition: *"Best taken in the sprint after the auth migration settles, not
  before,"* because bundling it with an auth diff would hurt reviewability.
  That condition is now met. The same reasoning is re-applied *within* T13,
  which is why T13.8 sits in Wave 3 behind T13.2 and T13.6 rather than running
  in parallel with them.
- **#124 (Game-cancellation cascade).** Still open. **Still deferred**, and the
  reason is unchanged and still correct: it is a product-semantics question
  (are courts freed, are players refunded), not an extension of an existing
  pattern. T12.3 shipped the refund path, which makes the refund half *more*
  answerable than before but does not answer it.
- **#125 (Competitions `PayableType`).** Still open. **Still deferred.** It is
  a genuine Payments feature, and T13's Payments budget is spent on two
  authorization gaps (#148, #131). Nothing in T12 made it more urgent.
- **#126 (real per-Game price).** Still open. **Still deferred** — unchanged
  reason: it needs Product Owner sign-off on pricing semantics before any code,
  per T8.10's own note. That sign-off has not happened.
- **#134 (WCAG manual screen-reader pass).** Still open. **Declined as
  unticketable in this environment, with reasoning rather than silence.** T12.5
  delivered a targeted 3.3.4 + keyboard pass and honestly recorded that a full
  manual pass was still owed. A *manual screen-reader* pass means driving
  NVDA/JAWS/VoiceOver and listening to it — it is not automatable, and no
  coding session in this project's history has had assistive technology
  available. **This is the same class as the Jenkins server-side wiring and the
  IdP tenant**: the repo-side work is doable here, this specific work is not.
  It stays open as #134 rather than being converted into a ticket that would
  quietly deliver another automated sweep and call it a manual pass. Flagged
  for the user, since only a human can discharge it.

## A17. Recorded disagreements (not manufactured consensus)

**QA vs. PE — is T13.6's roster-authorization scope right, or does it need the
Game-Admin store #149 names?**

- **QA:** #147 wants roster reads restricted to people entitled to see them.
  The entitled set is *Host + assigned Game Admins*. But
  `assigned_game_admin_user_ids` is **caller-supplied and persisted nowhere** —
  #149 says so explicitly and calls it a sub-gap that "should probably be
  closed first." A check that restricts rosters to the Host only is a different,
  narrower product behaviour than the issue asks for, and shipping it as if it
  closed #147 would overclaim.
- **PE:** true, and the narrower behaviour is still strictly better than the
  status quo (today *anyone holding an id* reads the roster). Building a
  durable Game-Admin store is its own domain + migration + proto cycle and
  would consume most of the sprint. Ship the Host check; disclose that Game
  Admins are not yet expressible.
- **Unresolved.** T13.6 as ticketed takes PE's narrower scope **plus** an
  explicit instruction to state, as a checked negative, that Game-Admin
  entitlement is not implemented and to open an issue for the durable
  Game-Admin store if one does not already exist. Per A5, its PR says
  **"partial fix for #147"**, not "closes". Both roles agree on that
  instruction; they disagree on whether the narrow version should ship at all.

**PdE vs. PO — is the Wave-1.5 checkpoint worth serialising a 40-point
sprint?**

- **PdE:** A11's reasoning is sound but the cost is real. T13.2 is 8 points and
  the largest single ticket; putting four tickets behind its *merge* (not its
  push) means Wave 2 cannot start until review completes. If T13.2 takes two
  review loops, half the sprint idles.
- **PO:** that is the trade the retro's evidence explicitly bought. The
  alternative is four implementers inferring the same unwritten decision, which
  is exactly what produced finding 1 — and the cost of *that* was a defect that
  reached the shared branch and is still there. A serialised sprint is
  recoverable; a wrong ownership comparison shipped four times is not.
- **Resolved in PO's favour, with PdE's mitigation adopted:** T13.2's AC puts
  ADR-0014 **first** in its instruction order, so the ruling the four consumers
  need is reviewable early even if the code half takes another loop. If the ADR
  is merged and the implementation is not, PE may release Wave 2 on the ADR
  alone — recorded here so it is a planned option rather than an improvised
  one.

**BA note, not a disagreement.** Ran the standing contradiction check against
every locked decision T13 touches. No contradiction found. Nothing here reopens
a locked decision: T13.2/T13.3 change how an actor identifier is *resolved*,
not what an Actor *is* (CLAUDE.md rule 7 is about the ubiquitous language, and
a Principal remains a Principal, a User remains a User). One naming note carried
into T13.2's text: whatever ADR-0014 decides, it must not introduce a third
word for "the thing identifying the caller" — `Principal.Subject` and `User.ID`
are the two existing concepts, and the ADR's job is to state how they relate,
not to add a synonym.

---

# Part B — Ceremony 2 (sprint planning)

## Sprint goal

> The verified principal T12 built stops being a mechanism with eleven tracked
> exceptions and starts being one that works: the two seams where a verified
> subject reaches a `uuid` column are fixed under a single recorded decision,
> so `RequestRecurringHire` and `CreateFacility` work for a real caller for the
> first time; the three RPCs that never had an authorization check
> (`ConfirmOnlinePayment`, and the two roster reads) get one; the auth spine's
> own tests run in a gate a session can actually execute, and that gate reaches
> CI; and six of T12's eleven residual auth issues are closed rather than
> carried. **What this sprint does not claim:** `CancelBooking` still has no
> authorization check (#144), Payments still compares a verified actor against
> caller-supplied ownership facts (#149), and no token from a real identity
> provider can be verified until a remote JWKS source exists (#137).

## In-scope tickets

### T13.1 — Backfill behavioural tests for the five cross-context adapter packages that have none

**Story:** As a maintainer, I want the code where two bounded contexts actually
meet to be covered by a test that can fail, so that a change to one context's
contract cannot silently break another context's adapter the way T12.9's did.

**Description:** T12 retro finding 2 / recommendation 2. A real regression
(`RequestRecurringHire`, #146) escaped two individually well-tested tickets
because the adapter joining the two contexts had **no test file at all** — not
a vacuous test, an empty set. Five packages are still in that state. No
dependency on any other T13 ticket; can start immediately.

**Instructions:**

1. Add a behavioural, **Docker-free** test to each of these five packages.
   Counts verified this ceremony by `ls` + `wc -l`, not carried from the retro:
   - `internal/booking/adapter/facilities` — currently `lookup_test.go`, **12
     lines**, a compile-time assertion only
   - `internal/competitions/adapter/facilities` — **12 lines**, same
   - `internal/competitions/adapter/booking` — `reservation_test.go`, **25
     lines**, same
   - `internal/socialplay/adapter/booking` — **no test file**
   - `internal/socialplay/adapter/facilities` — **no test file**
2. **Follow the pattern that already exists in this repo**, rather than
   inventing one: `internal/payments/adapter/socialplay/registration_updater_test.go`
   (**186 lines**, read this ceremony) wires the **real** other-context
   `app.Service` with in-memory repository fakes beneath it and drives the
   adapter through it. `internal/booking/adapter/identity/lookup_test.go`
   (216 lines, added by PR #151) uses the same shape. No Docker, no
   testcontainers.
3. **Test the behaviour these adapters actually exist for: error translation
   across the context boundary.** Verified by reading
   `socialplay/adapter/booking/reservation.go` and
   `socialplay/adapter/facilities/lookup.go` — each translates the other
   context's sentinel into its own (`bookingdomain.ErrCourtDoubleBooked` →
   `domain.ErrCourtUnavailable`; `facilitiesdomain.ErrFacilityNotFound` →
   `domain.ErrFacilityNotFound`) and deliberately wraps every *other* error
   with `%s` rather than `%w` so a caller cannot `errors.Is` across the
   boundary (CLAUDE.md rule 5). **Both properties must be asserted**: the
   sentinel is translated, and a non-sentinel error does **not** remain
   `errors.Is`-matchable against the source context's type.
4. **Identifier shape — read this carefully, it is a condition, not a blanket
   rule.** Recommendation 2 asks for "a realistically-shaped identifier — a
   non-uuid subject where the seam carries one." Checked this ceremony: of the
   five, only `booking/adapter/facilities` carries an actor at all
   (`EnsureFacilityOwner(ctx, facilityID, actorUserID string)`). For that one,
   use a **subject-shaped** actor (`auth0|…`). For the other four — which carry
   court/facility/game ids — use **uuid-shaped** ids, because production guards
   on `uuidShape`, and **state that as a checked negative in the PR** so a
   reader knows the condition was evaluated rather than skipped.
5. **Each test must be mutation-checked (CLAUDE.md rule 10).** Break the
   translation (return the raw upstream error), confirm the new test fails,
   restore, confirm green. Record the output in the PR. Three of these packages
   currently hold a compile-time assertion a reader could mistake for coverage;
   shipping five more tests nobody watched fail would reproduce that exact
   problem at five times the scale.
6. Keep the existing `var _ port.X = (*T)(nil)` assertions. They are cheap and
   correct — they are simply not tests.
7. Non-functional: `make test-domain` must stay green; no new dependency on
   Docker; no change to any adapter's production code (this is a test-only
   ticket — if a test cannot be written without changing production code, that
   is a finding to disclose, not a licence to refactor).

**Cross-context dependency check:** drives the real other-context `app.Service`
in all five; every one already imports it (A12).

**Story points:** 5. **Role:** qa. **Labels:** `role:qa`, `type:chore`,
`points:5`.

---

### T13.2 — ADR-0014 + the subject↔`User.ID` resolution seam; un-break `RequestRecurringHire`

**Story:** As a Club user, I want to request a recurring hire and have it
actually work, so that a capability this platform advertises stops returning
`NotFound` to every caller alive.

**Description:** The spine of this sprint, and the ticket that fixes the one
defect T12 left on the shared branch. Closes **#152** and completes **#146**
(PR #151 fixed the first of its two causes). It also produces the ruling three
other tickets depend on (A13 Gap 2), which is why it carries an ADR and why
Wave 2 does not dispatch until it merges (A11).

**Instructions:**

1. **Write ADR-0014 first, and get it reviewable before the implementation.**
   Number verified this ceremony: `docs/adr/` ends at `0013`. Per A17's
   resolution, the ADR being merged is what releases Wave 2, so it leads.
   It must decide, once, for the whole codebase:
   > When a verified `auth.Principal.Subject` must be compared against or
   > persisted alongside a stored actor fact, is the subject **translated** to
   > the corresponding `User.ID` (uuid) at the handler boundary, or do actor
   > columns **widen to text** and store subjects directly?
2. **The ADR must rule for all six contexts, including the ones needing no
   code change** (A13 Gap 2 — three tickets consume this). State, per context,
   which identifier space its persisted actor facts hold. **Two checked facts
   to build on, verified this ceremony:** Social Play (`0005_socialplay.sql:4`)
   and Competitions (`0014_competitions.sql:18`) store `host_id`/`player_id`
   as **plain text** — their migration comments say so explicitly. Facilities
   (`0010_facilities.sql:20`) and Booking's recurring-hire templates
   (`0018_…:32`) use **uuid**. Re-verify these before relying on them.
3. **Record why "just delete the guard" is wrong**, since it is the obvious
   move and it is worse than the bug. Verified this ceremony: the actor is
   persisted as
   `recurring_hire_templates.requested_by_user_id uuid NOT NULL REFERENCES identity_users (id)`
   and written through the Postgres adapter's `mustUUID()`, which **panics** on
   a non-uuid. Deleting the guard converts a clean `NotFound` into a server
   panic plus an FK violation.
4. **Fix the app-layer guard.** `internal/booking/app/recurring_hire.go`'s
   `RequestRecurringHire` opens with
   `if !uuidShape.MatchString(in.ActorUserID) { return domain.ErrUserNotFound }`
   — verified present in the current tree this ceremony — and it fires
   *before* `port.IdentityLookup` is consulted. Since T12.7 the actor here is a
   verified subject, so this rejects every real caller.
5. **Change `port.IdentityLookup`'s contract.** Verified by reading
   `internal/booking/port/identity_lookup.go`: it exposes **only**
   `EnsureClubRole(ctx, actorUserID string) error` (line 50), and its doc
   comment explains deliberately why it returns no `User` or `User.ID`. Adding
   subject resolution is a real interface change, not an addition — say so in
   the PR. The Identity side already exists:
   `identityapp.Service.UserBySubject` (`internal/identity/app/service.go:153`,
   T12.9), verified this ceremony.
6. **Audit every Booking handler `actor(ctx)` call site, do not assume.**
   `internal/booking/adapter/grpcapi/handler.go` defines
   `func actor(ctx) (string, error) { return auth.RequireSubject(ctx) }`
   (line ~40) and there are **six** call sites in the current tree. **Re-derive
   the list yourself** — #152 explicitly instructs a future implementer not to
   trust its own call-site list without re-checking, and new RPCs may have
   landed. For each, determine whether the actor reaches a uuid column.
7. **Confirm, do not assume, where the translation belongs.** #152 notes that
   the handler boundary is more consistent with A11 Ruling 3's
   handler-boundary-only precedent (which keeps `domain`/`app` free of
   `internal/platform/auth`), but says it "should be confirmed rather than
   assumed." Confirm it, and record the reasoning.
8. **Flip the pinned test.**
   `internal/booking/adapter/identity/recurring_hire_end_to_end_test.go`'s
   `TestRequestRecurringHire_SubjectActorStillBlockedByAppLayerUUIDGuard`
   asserts the **current, defective** behaviour deliberately and documents in
   its own comment how to flip it. Flip it to assert the fixed behaviour;
   do not delete it.
9. **State the constructor outcome explicitly in the PR body** (A13 Gap 3):
   whether `booking.app.NewService`'s signature changed, and its final
   parameter list. T13.8 consumes this.
10. **Do not touch `internal/platform/auth`** (A14). The translation is a
    Booking-side concern; the auth package is T13.5's this sprint. Named
    because this is the package T12.7/T12.9 collided in.
11. Non-functional: the domain stays pure (CLAUDE.md rule 2); gRPC codes only —
    a missing/invalid token stays `Unauthenticated`, a valid principal failing
    an object-level check stays `PermissionDenied`, and an actor whose subject
    resolves to no `User` is `NotFound` or `PermissionDenied` — **decide which
    and justify it**, since this is a new case ADR-0013 did not cover.
12. **Standing rule (A6):** any gap disclosed and not closed gets a GitHub
    issue in the same PR, not a paragraph.

**Cross-context dependency check:** calls `internal/identity` via
`port.IdentityLookup`; `UserBySubject` exists, the port method does not and
must be added (A12).

**Story points:** 8. **Role:** principal-engineer. **Labels:**
`role:principal-engineer`, `type:bug`, `points:8`.

---

### T13.3 — Apply ADR-0014 to Facilities' owner seam, and give Facilities its first Identity port

**Story:** As a Facility Owner, I want to create a Facility as a real
authenticated user, so that onboarding does not crash the moment a real
identity provider is connected.

**Description:** Closes **#154**, found during this ceremony by A13's
dependency-completeness check. `CreateFacility` mints `owner_id` from the
verified subject, but `facilities.owner_id` is `uuid NOT NULL` and the adapter
converts with a **panicking** helper. **Needs T13.2 merged** — it applies
ADR-0014's ruling rather than making its own.

**Instructions:**

1. **Verify the defect yourself before fixing it** (A4 — this ceremony read
   these, but re-derive rather than trust): `CreateFacility` in
   `internal/facilities/adapter/grpcapi/handler.go` sets `OwnerID` from
   `actor(ctx)` (i.e. `auth.RequireSubject`);
   `db/migrations/0010_facilities.sql:20` declares `owner_id uuid NOT NULL`;
   `internal/facilities/adapter/postgres/repository.go:33` converts via
   `mustUUID`, which panics (`repository.go:220-230`). Reproduce the panic in a
   test before fixing it — a bug with no failing test is an assumption
   (CLAUDE.md rule 1).
2. **Apply ADR-0014's ruling. Do not invent a second answer.** If it chose
   translation, resolve the subject to a `User.ID` before persistence. If it
   chose widening, add migration **`0020_facilities_owner_subject.sql`** —
   **pre-assigned to this ticket and no other** (A14), and only if that branch
   was chosen.
3. **Build Facilities' Identity port and adapter — this ticket owns it
   (A13 Gap 1).** Verified this ceremony: `internal/facilities/port/` contains
   exactly `idgenerator.go` and `repository.go`. **Facilities has never called
   another context**, so this is a new arrow in the context map, not a reuse.
   Follow the established convention: a primitive-typed port declared by the
   *consuming* context, implemented once in
   `internal/facilities/adapter/identity` against the real
   `identityapp.Service` — the reasoning is written out in
   `internal/booking/port/facility_lookup.go`'s doc comment. **No
   `identitydomain.User` may cross the boundary.**
4. **Do not let this become a shared kernel.** `internal/facilities/domain` and
   `app` must not import `internal/platform/auth` (A11 Ruling 3, carried from
   T12). The translation happens at the grpcapi boundary or in the adapter, not
   in the domain.
5. Wire the new adapter in `cmd/server/main.go`. T13.5 (Wave 1) and T13.8
   (Wave 3) also touch this file; all three are in different waves (A14), so
   append independently.
6. **Also fix `AddCourt` / `AddCameraLink` / `AttestCameraConsent` if the same
   seam breaks them** — verified this ceremony that
   `EnsureFacilityOwner(ctx, facilityID, actorUserID)` compares the actor
   against `Facility.OwnerID` via `facilitiesdomain.Facility.EnsureOwner`.
   Whether these break depends on ADR-0014's branch; **check each and record
   the answer either way**, per the verify-don't-assume instruction shape that
   found two real defects in T12.
7. **Per A5, this PR says "partial fix for #145" or does not mention #145 as
   closed at all.** It closes **#154** (new-row writes) but **not** #145
   (pre-existing uuid rows, genuinely blocked on a real IdP to map them
   against). Name precisely what remains. Do not write "closes #145".
8. Add the behavioural test for the new adapter in the shape T13.1
   establishes — real `identityapp.Service` over in-memory fakes, Docker-free,
   mutation-checked.
9. Non-functional: `make test-domain` green; `make vet-integration` run and
   recorded (A7).

**Cross-context dependency check:** **new arrow — Facilities → Identity.** No
port exists today (verified `ls internal/facilities/port/`); this ticket builds
it (A12, A13 Gap 1).

**Story points:** 5. **Role:** principal-engineer. **Labels:**
`role:principal-engineer`, `type:bug`, `points:5`.

---

### T13.4 — Make the CI gate cover what it claims: `internal/platform/**` in `make ci`, and a `Jenkinsfile` that calls it

**Story:** As an implementer, I want the auth spine's own tests to run in a
gate I can execute, so that the most security-critical package in this codebase
stops being the least verified.

**Description:** Closes **#138** and **#129**. Ranked 3rd in A1: cheap, and it
covers the package everything else now depends on. No dependency on any other
T13 ticket.

**Instructions:**

1. **The gap, verified this ceremony by reading the `Makefile` directly:**
   `test-domain` runs
   `go test ./internal/.../domain/... ./internal/.../app/... -race -count=1`
   — which matches **no** package under `internal/platform/`. And
   `ci: generate tidy lint test-domain vet-integration test-tools generate-client lint-web test-web build-web`
   contains no other step that would reach it. So
   `internal/platform/auth`'s tests (`auth_test.go`, `chain_test.go`,
   `require_test.go`, `rs256/…` — `ls` run this ceremony) run in **no** gate.
2. Add a gate covering `internal/platform/...` and wire it into `make ci`.
   **Do not simply widen `test-domain`'s glob** — its doc comment defines it as
   the dependency-free domain+app resume gate, and `internal/platform/pg`
   likely needs more than that. Check what each `internal/platform/*` package
   actually needs before choosing between widening, a new target, or a
   narrower pattern; state the reasoning in the PR.
3. **Close #129 in the same ticket:** the `Jenkinsfile` does not call
   `make ci`, so `vet-integration` (and any future `make ci` step, including
   this one) never reaches CI. Verify the current `Jenkinsfile` stage
   structure before changing it — it was written stage-by-stage deliberately
   (ADR-0011), so the fix may be "call `make ci`" or may be "add the missing
   stage"; decide and justify rather than assuming.
4. **Prove the gate is not vacuous (CLAUDE.md rule 10).** Break a test in
   `internal/platform/auth` (e.g. invert an assertion), confirm the new gate
   **fails**, restore, confirm green. Record the exact output. A gate nobody
   watched fail is an assumption, not a proof — this is the same standard T12.1
   was held to and met.
5. **Honest scope statement in the PR.** Closing #129 makes the pipeline
   *definition* correct; it does **not** mean CI runs. There is still no
   Jenkins job, webhook, or branch protection, and no session can create them —
   the same server-side gap `HANDOFF.md` has recorded since SCRUM-6. Say so in
   the same terms rather than implying CI now gates anything.
6. Non-functional: no Docker. If a step needs a daemon, the step is wrong —
   that is the whole point of the Docker-free gate class this joins.

**Cross-context dependency check:** none — `Makefile` + `Jenkinsfile` (A12).

**Story points:** 3. **Role:** qa. **Labels:** `role:qa`, `type:chore`,
`points:3`.

---

### T13.5 — Auth mechanism hardening: a nil verifier fails startup, a verifier panic fails closed

**Story:** As an operator, I want a misconfigured or malfunctioning token
verifier to stop the server rather than quietly admit everyone, so that "auth
is enabled" is a property I can rely on rather than hope for.

**Description:** Closes **#136** and **#135**. Grouped because they are the
same question asked at two moments — startup and runtime — and answering one
without the other leaves the guarantee half-built. Independent of every other
T13 ticket.

**Instructions:**

1. **#136 — an unconfigured (nil) `TokenVerifier` must be a startup failure.**
   Today it is not. Decide and implement where the check belongs (server
   construction vs. interceptor registration) and make the failure loud and
   immediate. A server that starts with no verifier and RPCs listed in
   `AuthenticatedMethods()` is claiming enforcement it cannot perform.
2. **#135 — decide fail-open vs fail-closed when the verifier panics, and
   implement it.** **The decision is not free**: the global recovery
   interceptor (PR #89) sits *in front of* the auth interceptor by design, so a
   verifier panic is currently caught there. Verify the current ordering and
   behaviour in `cmd/server/main.go` and `internal/platform/auth/interceptor.go`
   before choosing — do not assume the panic reaches the caller as
   `Unauthenticated` or as `Internal`; derive it.
3. **Fail-closed is the expected answer; justify it rather than asserting it,
   and say what it costs.** A panicking verifier that admits the request is a
   security failure; one that rejects every request is an availability failure.
   Record which this project prefers and why, in the PR (and in ADR-0013's
   record if the ADR's stated position needs amending — check whether it
   already covers this).
4. **Both must be proven by test, mutation-checked** (CLAUDE.md rule 10):
   a nil verifier fails startup (restore the old behaviour, watch the test
   fail); a panicking verifier rejects rather than admits.
5. **Do not widen scope into #137 (remote JWKS).** That is genuinely blocked on
   an IdP nothing here can provision (A1 rank 10). This ticket hardens the
   mechanism that exists; it does not add a key source.
6. Non-functional: `internal/platform/auth` imports no bounded context
   (CLAUDE.md rule 3). This ticket is the **only** T13 ticket touching that
   package (A14) — if you find yourself needing a change another ticket also
   needs, stop and raise it rather than building it, per finding 1.

**Cross-context dependency check:** none — `internal/platform/auth` +
`cmd/server` (A12).

**Story points:** 3. **Role:** principal-engineer. **Labels:**
`role:principal-engineer`, `type:chore`, `points:3`.

---

### T13.6 — Roster reads require an authorized actor

**Story:** As a Player, I want the list of people registered for a Game not to
be readable by anyone who guesses its id, so that my participation is not
public data by accident.

**Description:** **Partial fix for #147.** `ListRegistrationsForGame` and
`ListEntriesForCompetition` have no owner check — anyone holding an id reads
every registrant/entrant. **Needs T13.2 merged** for ADR-0014's ruling on which
identifier space `games.host_id` / `competitions.host_id` hold (A13 Gap 2).
Scope is contested — see A17.

**Instructions:**

1. Add an authorization check to both roster reads. The actor is already a
   verified principal (T12.8); what is missing is the comparison.
2. **Apply ADR-0014's per-context ruling rather than assuming.** This
   ceremony's checked finding is that Social Play and Competitions store
   `host_id`/`player_id` as **plain text** (`0005_socialplay.sql:4`,
   `0014_competitions.sql:18` — their own migration comments state it), so
   these are likely comparable directly against a subject with no translation.
   **Confirm against ADR-0014 and the current schema before relying on it** —
   if T13.2's ruling contradicts this, the ruling wins.
3. **Scope: Host-only, and disclose the limit rather than implying more**
   (A17's unresolved QA/PE disagreement). Game Admins *should* be entitled to
   read a roster, but `assigned_game_admin_user_ids` is caller-supplied and
   **persisted nowhere** — #149 records this and calls it a sub-gap to close
   first. Implementing "Host or caller-supplied admin list" would let any
   caller name themselves an admin, which is worse than Host-only.
4. **Per A5, the PR title and body say "partial fix for #147", never
   "closes #147"**, and name exactly what remains (Game-Admin entitlement).
5. **Open an issue for the durable Game-Admin store if one does not already
   exist** — check first; #149 discusses it but may not have a dedicated
   issue. Per A6 the review must enumerate what this PR opened.
6. **Constraint, load-bearing for T13.8 (A13 arrow 5):** do not add a
   constructor dependency to `socialplay`/`competitions` `app.Service`. The
   authorization uses facts the service already holds. If that turns out to be
   impossible, **say so loudly in the PR** — T13.8 is planned on the assumption
   that it holds.
7. **Mutation-check both checks** (CLAUDE.md rule 10): disable each, confirm
   the regression tests fail, restore. Assert the mapped gRPC status is
   `PermissionDenied`, not `Internal`.
8. Non-functional: `make test-domain` green; `make vet-integration` run and
   recorded (A7) — Social Play has 5 integration-tagged files and Competitions
   2, the most exposed context pair in this sprint.

**Cross-context dependency check:** none — `internal/socialplay` and
`internal/competitions` only; host identity is text in both (A12).

**Story points:** 5. **Role:** principal-engineer. **Labels:**
`role:principal-engineer`, `type:story`, `points:5`.

---

### T13.7 — `ConfirmOnlinePayment` requires an owner check

**Story:** As a payer, I want only the person who created a payment to be able
to capture it, so that holding a `payment_id` is not the same as holding the
money.

**Description:** Closes **#148**. `ConfirmOnlinePayment` has no owner check at
all — anyone holding a `payment_id` can capture the intent. Money-adjacent and
needs no domain change. **Needs T13.2 merged** (A13 Gap 2) and branches from a
base containing **T13.9** (same file — A14).

**Instructions:**

1. Add an authorization check to `ConfirmOnlinePayment`. The actor is already a
   verified principal (T12.8); the comparison is what is missing.
2. **Scope it to facts Payments already holds — this is the binding
   constraint.** Verified this ceremony: `internal/payments/port/` contains
   exactly `idgenerator.go`, `payment_processor.go` and `repository.go`.
   **Payments has no read path into Booking, Social Play or Competitions**,
   which is precisely why #149 (fact-fabrication) is deferred. Do **not** build
   cross-context read ports in this ticket — that is a structural change and
   its own sprint's work.
3. **Determine what Payments can legitimately check** before writing code: the
   `Payment` row records who created it. Compare the verified actor against
   that, rather than against a caller-supplied ownership fact. **Derive the
   available fields from `internal/payments/domain` and the schema; this plan
   deliberately asserts nothing about them** (A4).
4. **Disclose the residual honestly.** Closing #148 does **not** close #149 —
   the actor becomes verified against a fact Payments itself recorded, but the
   *other* ownership facts (`game_host_id`, `booking_host_id`,
   `entrant_player_id`, admin lists) remain caller-supplied. Say so; per A5 do
   not write "closes #149".
5. **Error mapping:** the new sentinel maps to `PermissionDenied`, never
   `Internal`. **T13.9 will have already corrected `toStatus` in this file** —
   add the mapping into the corrected structure rather than reverting or
   duplicating it, and re-read `toStatus` before editing (A4: this plan does
   not assert what it will look like after T13.9).
6. **Mutation-check** (CLAUDE.md rule 10): disable the check, confirm the
   handler-level regression test fails, restore, confirm green — mirroring
   T8.5's `authz_regression_test.go` shape, which is the established precedent
   for this context.
7. Non-functional: `make test-domain` green; `make vet-integration` run and
   recorded (A7) — payments has 3 integration-tagged files.

**Cross-context dependency check:** none, **and that is the constraint** —
Payments has no cross-context read ports (A12).

**Story points:** 3. **Role:** principal-engineer. **Labels:**
`role:principal-engineer`, `type:story`, `points:3`.

---

### T13.8 — Migrate `booking` and `socialplay` `app.NewService` to a `ServiceOptions` struct

**Story:** As a maintainer, I want every context's service constructed the same
way, so that adding a dependency stops being a breaking change to every call
site and every test.

**Description:** Closes **#123**, open since T1 and re-flagged at T6.6 and
T11.5. T12's plan deferred it with an explicit timing condition — *"best taken
in the sprint after the auth migration settles"* — which is now met (A16).
**Wave 3: needs T13.2 and T13.6 merged**, since both touch the same service
files.

**Instructions:**

1. Migrate `booking.app.NewService` and `socialplay.app.NewService` to a
   `ServiceOptions` struct, matching the shape `competitions` and `payments`
   already use. **Read those two as the pattern rather than inventing one**;
   `internal/payments/app`'s `ServiceOptions` is the older of the two and has
   already absorbed a cross-context dependency (T6.5's
   `RegistrationUpdater`), which is the exact scenario this refactor exists to
   make cheap.
2. **Re-derive both current signatures from the merged code, do not trust this
   plan or #123's counts** (A4, A13 Gap 3). #123 records booking at 7
   positional params and socialplay at 5, but **T13.2 may have added a
   dependency to booking** and its PR body is required to state the final
   list. `HANDOFF.md` also records socialplay growing from 4 to 5 across T6.6.
   Check the code.
3. **This is a mechanical refactor and must stay one.** No behaviour change, no
   signature change beyond the constructor, no reordering of validation. If a
   real bug surfaces while moving the parameters, **disclose it and open an
   issue** (A6) rather than fixing it inside a refactor diff — a behaviour
   change hidden in a mechanical diff is the least reviewable shape there is.
4. Update every call site, including `cmd/server/main.go` and all tests. The
   compiler finds them; the risk is a *silently wrong* argument, which
   positional constructors permit and this refactor exists to end — so
   **verify by reading each converted call site**, not only by a green build.
5. **Mutation-check the conversion where it matters most:** confirm that
   swapping two same-typed dependencies in a converted call site now fails
   loudly (or is impossible), which is the property being bought.
6. Non-functional: `make test-domain` green; `make vet-integration` run and
   recorded (A7) — this touches constructors called from integration-tagged
   files in socialplay (5) and booking (1), which is exactly the class that
   broke twice in T11.

**Cross-context dependency check:** none — mechanical constructor refactor
(A12).

**Story points:** 5. **Role:** principal-engineer. **Labels:**
`role:principal-engineer`, `type:chore`, `points:5`.

---

### T13.9 — One domain sentinel, one gRPC code: fix `RefundPayment`'s error mapping inconsistency

**Story:** As an API consumer, I want the same domain error to produce the same
status code regardless of which RPC surfaced it, so that error handling is a
contract rather than a per-endpoint guess.

**Description:** Closes **#131**. T12.3 disclosed that `RefundPayment` maps
`ErrIllegalStatusTransition` / `ErrPaymentProcessorUnavailable` differently
from the rest of `PaymentsService`; T12.4 independently hit the same
inconsistency from the Social Play side (finding 5's "good convergent
evidence"). Independent of every other ticket, but **shares
`payments/adapter/grpcapi/handler.go` with T13.7** — this ticket is Wave 1 and
T13.7 is Wave 2 (A14).

**Instructions:**

1. **This plan deliberately asserts nothing about the current mapping**
   (A4 — recommendation 3 applied at its sharpest, because this ticket's whole
   subject is a mapping, and T12.4's plan text got exactly this wrong).
   **Derive the current behaviour by reading `toStatus` in
   `internal/payments/adapter/grpcapi/handler.go` directly**, and write down
   what you found before changing anything.
2. Decide the correct mapping per sentinel and make it consistent across every
   RPC in the service. **A sentinel that means two different things to two RPCs
   is the actual bug** — if that turns out to be the case, the fix may be a new
   sentinel rather than a re-mapping.
3. **Do not re-map a sentinel globally without checking every call site.**
   T12.4's implementer specifically avoided this because it *"would have
   silently changed unrelated RPCs' response codes."* Enumerate the call sites
   and state the blast radius in the PR.
4. **Check whether Social Play has the mirror-image problem** — #131's
   corroboration came from T12.4 working in `socialplay`'s `toStatus`. If the
   same inconsistency exists there, **disclose it and open an issue** (A6);
   do not silently widen this ticket into a second context.
5. Table-driven tests over sentinel → code for the whole service, so a future
   mapping change cannot be made silently. Mutation-check at least one
   (CLAUDE.md rule 10).
6. Non-functional: no domain change — this is an adapter-layer mapping fix
   (CLAUDE.md rule 5: adapters translate; upper layers see domain errors).
   `make vet-integration` run and recorded (A7).

**Cross-context dependency check:** none — `internal/payments/adapter/grpcapi`
only (A12).

**Story points:** 3. **Role:** principal-engineer. **Labels:**
`role:principal-engineer`, `type:story`, `points:3`.

---

## Dependency order and dispatch waves (see A13 for the completeness check, A14 for isolation)

```
Wave 1 (parallel, ≤5 implementers, worktree-isolated):
  T13.1  cross-context adapter test backfill   (5 adapter pkgs, test-only)  ─┐
  T13.2  ADR-0014 + subject↔User.ID seam       (booking + ADR)              ─┤
  T13.4  CI gate covers platform + Jenkinsfile (Makefile/Jenkinsfile)       ─┼─ independent
  T13.5  auth mechanism hardening              (platform/auth + main.go)    ─┤
  T13.9  RefundPayment error mapping           (payments handler toStatus)  ─┘

════ WAVE 1.5 CHECKPOINT (A11 — A14's scored lesson, 4 first-time consumers) ════
  T13.2 MUST BE MERGED AND REVIEWED before Wave 2 is dispatched at all.
  Fallback agreed in A17: if ADR-0014 is merged but T13.2's code needs
  another loop, PE may release Wave 2 on the ADR alone — a planned
  option, not an improvisation.

Wave 2 (parallel, ≤3 implementers, worktree-isolated):
  T13.3  Facilities owner seam + new Identity port  (needs T13.2; maybe 0020)
  T13.6  roster read authorization                  (needs T13.2's ruling)
  T13.7  ConfirmOnlinePayment owner check           (needs T13.2's ruling;
                                                     base contains T13.9)

Wave 3 (1 implementer):
  T13.8  booking + socialplay → ServiceOptions
  — needs T13.2 AND T13.6 merged (same service files, A14)
```

Total: **40 points across 9 tickets** (T12: 46/9, T11: 47/9, T10: 37/8).

**Deliberately below T12's points at the same ticket count**, for two stated
reasons rather than by accident: (i) five of nine tickets are defect or
missing-check work on security- and money-adjacent paths, where the expensive
part is proving the fix rather than writing it; (ii) the Wave-1.5 checkpoint
serialises the sprint by design (A11), so a T12-sized backlog would not fit the
shape. PdE's objection to that trade is recorded in A17.

**Wave roll-call is mandatory before ending any work block** (T12's A2,
unchanged and still standing). The dispatch list above is the list to roll-call
against. **A9(a)'s closing condition (A10) is scored from what the roll-call
observes this sprint** — whoever runs the retro needs that record.

## Explicitly deferred, not silently dropped

Per the board-of-record rule, **every item below that outlives this sprint has
a GitHub issue** — verified open this ceremony against the live issue list.

- **#144 — `CancelBooking`/`CreateBooking` have no authorization check.**
  Ranked 7th (A1). **Not taken**, for three compounding reasons: it needs an
  owner concept on the `Booking` aggregate (a domain change), a migration
  (verified: `bookings` has no owner column — the table has
  `id, court_id, source, status, starts_at, ends_at, during, reference_id,
  created_at` and nothing else), **and a product decision the issue itself
  names** — what "owns" a booking made through T7.6's public quote-and-book
  flow. Making `CreateBooking` authenticated would change a shipped public UI
  flow. That is a PO/product call, not an engineering one, and T13 is already
  serialised behind a checkpoint. **Strong T14 candidate, and it should be
  first** — it is the sharpest remaining BOLA hole.
- **#149 — Payments accepts caller-supplied ownership facts.** Ranked 8th.
  **Not taken:** structural. Verified this ceremony that
  `internal/payments/port/` has no cross-context read ports at all, so closing
  it means building read-side ports into three contexts. Its own text says the
  Game-Admin persistence sub-gap "should probably be closed first" — and T13.6
  will open an issue for that store if one does not exist, which is the right
  first step.
- **#145 — pre-existing uuid `owner_id` rows vs. a non-uuid subject.**
  **Narrowed by T13.3, not closed**, and genuinely blocked: mapping existing
  uuid owners to real subjects requires a real IdP to map them against. Its
  false scope claim was corrected by comment this ceremony (A2).
- **#137 — no remote JWKS `KeySource`.** **Blocked**, same class as the
  Jenkins server-side wiring: it needs a reachable identity provider no session
  has. Named so it is not mistaken for unprioritised.
- **#124 (Game-cancellation cascade), #125 (Competitions `PayableType`), #126
  (real per-Game price), #130 (refunding a `no_show_fee`).** Re-checked open;
  reasoning unchanged (A16). #126 additionally still needs PO sign-off before
  any code.
- **#134 — the WCAG manual screen-reader pass.** **Declined as unticketable in
  this environment**, with reasoning in A16: a manual screen-reader pass is not
  automatable and no session here has assistive technology. Stays open;
  flagged for the user, since only a human can discharge it. Converting it into
  another automated sweep would be exactly the overclaim T12.5 avoided.
- **ADR-0012's Q1/Q2 work** (`PlayerRating`, matching algorithm, gender-mix
  matching). **Still blocked on the user**, checked this ceremony (A15). Not
  deferred by this team's choice and not this team's to decide.
- **Observability (Sentry + slog + uptime); `golang-migrate`/`goose` swap;
  ISO-8601 weekday numbering; the T6.4 uncommitted payments concurrency
  proof.** Unchanged low-urgency infra debt, no new pressure this sprint. No
  issues opened: these are roadmap items, not disclosed gaps.
- **CI server-side wiring** (Jenkins job/webhook/branch protection). **Not
  ticketable**, unchanged since SCRUM-6. T13.4 closes the repo-side half
  (#129) and is instructed to say so honestly rather than implying CI now
  gates anything.

## `HANDOFF.md` and `sprint-process.md` updates this ceremony's PR carries

Per **recommendation 9** (A9), this PR carries both halves of the fix:

- **The immediate correction:** the **T12** Docs-index row, which currently
  reads "not yet written" for the retro and "not yet opened" for the reviews.
  Both stale — the retro merged as PR #153 and the nine tickets as PRs
  #128–#150. Verified against the GitHub API this ceremony.
- **The structural fix (option b):** `sprint-process.md`'s Ceremony 1 gains an
  explicit *"correct the previous sprint's Docs-index row"* step, so this stops
  recurring. Reasoning for choosing (b) over (a) is in A9 — decisively, a retro
  PR cannot cite its own merge PR number before that number exists.
- A new **T13** row in the Docs index, matching T10/T11/T12's format.
- A **T12 entry in the "Task backlog" narrative**, written in the retro's
  honest sentence (A0/A8) rather than the sprint goal's "every authorization
  check" phrasing, so `HANDOFF.md` resumes accurately and no future ticket
  reads authorization as finished.

Ticket-level `HANDOFF.md` edits (T13.2's #146/#152 closure, T13.3's #154
closure and the Cross-cutting caveat updates) belong to their own tickets' PRs,
not this one.

## Definition of Done (sprint-level)

All 9 tickets merged per `sprint-process.md`'s per-ticket DoD (PR-only,
CLAUDE.md rule 9 — **no exception for this plan**, which lands the same way);
sprint goal met (both subject→uuid seams fixed under one recorded decision,
`RequestRecurringHire` and `CreateFacility` working for a real caller, three
missing authorization checks added, `internal/platform/**` gated, six of T12's
eleven residual auth issues closed); **the Wave-1.5 checkpoint observed** (A11)
— T13.2 merged and reviewed before Wave 2 dispatch, or the A17 ADR-only
fallback consciously invoked and recorded; **wave roll-call performed before
every work-block end**, with any "neither PR nor branch" ticket named; **every
review enumerating the issues it opened, or stating there were none** (A6), and
recording that **`make vet-integration` was run** (A7); every partial fix
titled **"partial fix for #N"** (A5); retro held (`docs/process/t13-retro.md`)
with findings indexed from `docs/LESSONS.md`'s `## T13 sprint retro`;
`HANDOFF.md` updated for T14 to resume from — including **A10's two mandated
scorings**: A9(a) closed one of its three ways and not deferred a fourth time,
and QA's port-contract-change rule scored against whether A13's
dependency-completeness check caught that shape on T13.2/T13.3.
