# T12 Sprint Retro

Ceremony 3 per `docs/process/sprint-process.md` (read in its **T12.6-amended**
form, since T12.6 changed the board-of-record section this ceremony is
governed by), six-role team (briefs: `docs/agent-operating-handbook.md`
Part B), held against `docs/process/t12-sprint-plan.md` (including its
A1–A14 appendix), `docs/process/t11-retro.md` and `docs/process/t10-retro.md`
as precedent, `HANDOFF.md`, and the real PR/issue history on
`nhuthuynh/white-label` (GitHub-side name `pickleball-platform`) — PRs
#127–#151, issues #123–#152.

Every timestamp, merge order, file list and diff below was pulled from
GitHub's own `created_at`/`merged_at`/`submitted_at` fields and from
`git show`/`git log` against the merged branch — not inferred from titles or
from the coordinating session's summary. Claims checkable against the
working tree were checked: `make test-domain` was re-run here (green, all
twelve domain+app packages), `go test ./internal/platform/... -race` was run
separately (green — and that separation is finding 4's evidence), the
`Makefile`'s `ci` and `vet-integration` targets were read rather than
trusted, `internal/platform/auth/` was listed directly, every
`internal/*/adapter/*/` package was enumerated and its test files counted,
and `internal/socialplay/adapter/grpcapi/handler.go`'s `toStatus` was read
line-by-line to adjudicate a plan claim.

**This retro's assigned investigation points are not taken as given.**

- **Finding 1 substantially reframes its premise.** It was handed to this
  ceremony as a "same-package new-file collision" and asked whether that is
  T11 finding 3's class in a new guise. It is neither. The root cause is not
  file overlap at all: **T12.2 shipped no enforcement primitive whatsoever**
  (verified — `git log --diff-filter=A` shows `require.go` was created by
  T12.7, not T12.2), so *both* Wave-2 consumers independently discovered a
  capability nobody had been assigned to build. That is a new category, and
  it makes A14's PdE the substantively correct party in a way neither role
  predicted.
- **Finding 2 corrects the "happened to" in its premise.** T12.8 did not
  stumble on the regression. The sprint plan's own T12.8 instruction item 2
  *ordered* the investigation that found it, in writing, including "record
  the finding either way." Planning found this bug, not luck — which changes
  the recommendation completely.
- **Finding 2 also narrows its own diagnosis.** The premise offered
  "every cross-context port gets faked in the calling context's tests" as
  the systemic gap. Faking the port is correct and is not the gap. The gap is
  that **the adapter implementing the port against the real other context had
  zero test files**, and that this is measurable: 5 of 8 cross-context
  adapters still have no behavioural test today.
- **Finding 5 adds a plan error nobody was asked to look for**, in the exact
  signature T11 finding 5 identified — an unmarked factual claim, and it was
  false.
- **Finding 6 partly corrects the sprint's self-assessment**: A5's rule was
  satisfied in outcome 19 times out of 19, but the mechanism that delivered
  it was not the mechanism A5 wrote.

Findings are not a single voice. Recorded disagreements are left as
disagreements per the "do not manufacture consensus" rule, matching T9's,
T10's and T11's retros.

**Sprint outcome:** all 9 tickets (46 points, T12.1–T12.9) merged across PRs
#128–#150, plus the Ceremony 1/2 doc (#127) and one unticketed hotfix (#151).
The sprint ran in a single unbroken work block, **2026-08-14T16:15:52Z to
18:03:33Z — 1 hour 47 minutes 41 seconds** from plan merge to final merge,
the fastest full sprint in this project's history and the first with no
overnight gap. All three waves executed in their planned order and every
dispatched ticket produced a PR; there was **no silent agent** (A2's
roll-call scored in finding 5). Real authentication exists: 24 RPCs across
all six bounded contexts now resolve their actor from a verified
`auth.Principal` instead of a caller-supplied string, the `CreateUser`
identity-squatting DoS `HANDOFF.md` has carried since T10.2 is closed and
proven closed by a test that replays the original attack, and `RefundPayment`
(open since T6.5) and `CancelGame` (open since T5.5) are finally built. The
review record continues T11's standard: every one of the nine PRs carries a
discoverable GitHub review object, and the reviewer performed independent
fresh-worktree re-verification and independent mutation-checking — not merely
reading the implementer's captured output — on the security-critical ones.

**One real defect reached the shared branch and is still there.** T12.7 and
T12.9's interaction broke `RequestRecurringHire` for every caller; PR #151
fixed one of its two causes; the second is tracked as #152 and the capability
remains non-functional at the time of writing. This is the first sprint since
T10 in which a defect survived to the shared branch, and it is the subject of
findings 2 and 3.

---

## 1. T12.2 shipped the auth platform without the one primitive all three of its consumers needed — so two of them built it independently, in the same package, 68 seconds apart

**PE (the mechanism, from primary sources, because the premise needs
replacing before it can be assessed).** This was handed to the ceremony as a
new-file collision inside a shared package. The collision is real, but its
cause is not adjacency. Checked directly:

```
$ git show --stat --format="" 8a7c8ed          # T12.2, PR #140
 internal/platform/auth/auth_test.go        | 369 +
 internal/platform/auth/chain_test.go       | 331 +
 internal/platform/auth/interceptor.go      | 183 +
 internal/platform/auth/principal.go        | 170 +
 internal/platform/auth/rs256/…             | (4 files)
 internal/platform/auth/verifier.go         | 153 +
 …
$ git log --oneline --diff-filter=A -- internal/platform/auth/require.go
5d185ac T12.7: Migrate Booking + Facilities to the verified principal (#142)
```

**T12.2 shipped no enforcement primitive at all.** There is no `require.go`,
no `enforce.go`, nothing that turns an `AuthenticatedMethods()` list into a
rejected call. That was not an oversight — it is A14's PE ruling, made
deliberately: T12.2's AC 4 required observe-only ("the interceptor populates
the context and **enforces nothing**"), and A11 Ruling 2 gave each context an
`AuthenticatedMethods()` list. What no ticket owned was the **consumer of
those lists**. Every migration ticket needed it; none was told to build it.

So both Wave-2 tickets built it:

| Ticket | Primitive built | File |
|---|---|---|
| T12.7 (#142) | `MethodSet` / `RequireUnaryInterceptor` / `RequireStreamInterceptor` / `RequireSubject` | `internal/platform/auth/require.go` |
| T12.9 (#143) | `RequireAuthentication` / `RequireAuthenticationStream` | `internal/platform/auth/enforce.go` |

**QA (the timings, computed from GitHub's own fields).** The two PRs opened
**68 seconds apart** — #142 at 17:09:09Z, #143 at 17:10:17Z — and merged
17:13:23Z and 17:19:28Z. Neither implementer could see the other's work:
T12.9's body states *"T12.7 had not pushed when I looked"*, and T12.7's states
*"T12.9 had not merged at push time."* Both were telling the truth.

**The part that is genuinely to this sprint's credit: both implementers
predicted the collision in their own PR bodies, independently, before it was
adjudicated.**

- T12.7, disclosure 4: *"`internal/platform/auth` grew an enforcement half,
  which A12 did not predict… which **T12.9 and T12.8 also need**. If T12.9
  built its own, that is a same-wave collision A12 asserted would not occur —
  worth a note for T12's retro either way."*
- T12.9, §5: *"this PR adds `internal/platform/auth/enforce.go`… **T12.7
  needs the same primitive**, which makes `internal/platform/auth` a
  same-wave new-file collision the plan's 'no two tickets in the same wave
  create the same new file' claim did not predict. Raising it for the retro."*

Two implementers, working blind, each reasoned to the same conclusion about
the other. That is the disclosure discipline this project has been building
since T9 working exactly as intended, and it is why the collision cost
minutes rather than a bad merge.

**PdE (the resolution, verified).** It was caught in the coordinating
session's review of T12.9 and resolved on T12.9's own source branch. The
merge commit `74b5632` records it verbatim:

> *"Resolved by keeping T12.7's already-merged primitives as canonical and
> deleting `enforce.go`/`enforce_test.go` entirely, rather than shipping two
> functionally-identical enforcement mechanisms in the same package."*

Verified against the tree: `internal/platform/auth/` today contains
`require.go` and no `enforce.go`, and `git log --all -- .../enforce.go`
returns exactly one commit (`a4a4b46`, T12.9's original, superseded by its own
merge). Post-merge verification is recorded in the same commit (build, vet,
`vet-integration`, lint 0 issues, race suite, and both load-bearing security
regression tests re-run through the consolidated wiring).

**Worth noting: A12's stated resolution rule generalised correctly even
though A12's table did not.** A12 named only `cmd/server/main.go` and stated
"whichever merges second resolves on its own branch, keeping both entries."
Applied to a file the table never named, the same rule produced the right
answer — with the correct adaptation that "keeping both" is wrong when the
two entries are functionally identical. The rule was portable; the *table*
was not.

**Is this T11 finding 3's class in a new guise, or a new category? PE and
PdE agree, unusually, and the answer is: new category.**

T11 finding 3 was a **visibility** failure. Two tickets both appended to
`internal/booking/domain/errors.go`; the file existed, both diffs would touch
it, and neither ticket's *expected file list* mentioned it. The adopted fix —
ask which **existing** files each pair will append to — is exactly right for
that failure and would have worked.

T12's collision is a **provenance** failure, and the T11 fix is structurally
incapable of catching it. At planning time, the honest answer to "which files
will T12.7 and T12.9 create in `internal/platform/auth`?" was **"none — T12.2
provides that package."** Both tickets' plans were correct as written. The
missing capability only became visible when two implementers independently
hit the same wall. No file-overlap question, asked of new files or existing
ones, reaches it.

**The question that would have caught it is about the dependency graph, not
the file system:** *for each arrow in the wave dependency graph, does the
upstream ticket's AC actually deliver everything the downstream tickets' ACs
require?* T12.2's AC delivered a `Principal`, a `TokenVerifier` and two
observe-only interceptors. T12.7/T12.8/T12.9's ACs each required "enforcement
turned on per-RPC." Those two statements do not meet, and they are two pages
apart in the same document. A capability-completeness check on the dependency
graph is a *reading* of the plan the plan can perform on itself.

**T12.8 did not repeat it — verified, and it is stronger than "avoided."**
T12.8 was dispatched with explicit knowledge of the collision, and its diff
*does* touch `internal/platform/auth` — two files, checked directly:

```
$ git show --stat --format="" 7c5fa80 | grep platform/auth
 internal/platform/auth/require.go       | 10 +
 internal/platform/auth/require_test.go  | 32 +
```

It added a single `MethodSet.Len()` method **to the consolidated canonical
file**, with a doc comment explaining why (`main.go`'s no-verifier startup
warning had been hand-summing each context's `len(AuthenticatedMethods())` —
a list that "had already gone stale for three contexts"). The reviewer
verified this specifically: *"Confirmed `MethodSet.Len()` is a small,
legitimate addition to the existing canonical `require.go`… not a second
enforcement primitive, correctly avoiding a repeat of the T12.9 collision."*
So T12.8 consumed, extended, and removed a staleness source in the primitive
its predecessors fought over. That is the best possible outcome and it is
recorded here as checked, not assumed.

**Change for T13, adopted:** the sprint plan's dispatch-isolation section
gains a **dependency-completeness check** alongside its two file-overlap
questions. For every dependency arrow in the wave graph, state in one line
what the upstream ticket's AC delivers and what each downstream ticket's AC
consumes, and name any capability that appears on a consumer's side and no
producer's side. Where such a capability is found, it is assigned to exactly
one ticket — normally the upstream one — before dispatch.

## 2. A real regression escaped two individually well-tested tickets because the adapter joining the two contexts had no test file at all — and the sprint plan, not luck, is what found it

**QA (the defect, traced end to end against merged code).** T12.7 migrated
Booking's handler so the actor is `auth.RequireSubject(ctx)` — an IdP subject
like `auth0|abc123`. T12.9 split `User.ID` (a server-minted uuid) from
`User.Subject` (the IdP string) and added `UserBySubject` for exactly that
translation, wiring it only into Identity's own handler. Booking's
cross-context adapter still called `GetUser`, which opens with
`if !uuidShape.MatchString(id) { return ErrUserNotFound }`. A subject can
never match a uuid, so **`RequestRecurringHire` returned `NotFound` for every
caller alive** — a whole shipped capability, non-functional on the shared
branch.

Both tickets were individually well-tested and both were right in isolation.
The break lives strictly in the seam.

**PdE (correcting the premise: this was not luck).** The ceremony was told
this surfaced because T12.8 "happened to do a cross-context identifier-space
investigation as part of its own scope." It did not happen to. The sprint
plan **ordered it**, in T12.8's Instructions item 2:

> *"**This is where `actor_player_id` and `actor_user_id` meet.** … Decide
> and state whether a Principal's subject maps to both… — **check the
> existing relationship in `internal/socialplay` and `internal/identity`
> before deciding, and record the finding either way.** If it turns out these
> are genuinely two identifier spaces, that is a real design finding:
> disclose it, and per A5 open an issue rather than inventing a mapping
> mid-ticket."*

T12.8's own commit message traces the causal chain from that instruction to
the discovery: it checked the relationship, concluded the two names were one
space, noted that this was "safe for these three contexts specifically
because none of them calls into Identity," and then — *"That last clause is
load-bearing, and checking it surfaced a REAL BUG in already-merged code."*

**This matters enormously for what to recommend.** A finding framed as "we
got lucky" produces a recommendation to buy more luck (test everything
against everything). A finding framed as "a written instruction to verify a
relationship rather than assume it paid off two tickets later" produces a
much cheaper recommendation: keep writing that instruction. This is the
second consecutive sprint where an explicit *verify-don't-assume* instruction
in ticket text found a real defect — T11.9's fixture-ID sweep found a vacuous
test the same way. The pattern is now established enough to name.

**PE (the actual coverage gap, measured rather than characterised).** The
premise offered was that every cross-context port is faked in the calling
context's tests, making this systemic. Half right, and the wrong half is
load-bearing. Faking `port.IdentityLookup` in `internal/booking/app`'s tests
is **correct** — that is what a port is for, and those tests are testing
Booking's logic, not Identity's. Mandating real implementations there would
be expensive and would couple every context's unit tests to every other
context's internals.

The real gap is one level down and is exactly measurable. Every
`internal/<ctx>/adapter/<other-ctx>/` package exists solely to implement one
context's port against another context's real service — it is the *only*
code in the repo where the two contexts actually meet. Enumerated directly:

| Cross-context adapter | Test files | Nature |
|---|---|---|
| `booking/adapter/identity` | `lookup_test.go` (216 L) + `recurring_hire_end_to_end_test.go` | **real behavioural — added by PR #151, the hotfix** |
| `payments/adapter/socialplay` | `registration_updater_test.go` (186 L) + Docker-gated | real behavioural (drives the real `socialplayapp.Service`) |
| `payments/adapter/competitions` | `entry_updater_test.go` (165 L) + Docker-gated | real behavioural |
| `booking/adapter/facilities` | `lookup_test.go` (**12 L**) | compile-time `var _ port.X = (*T)(nil)` only |
| `competitions/adapter/facilities` | `lookup_test.go` (**12 L**) | compile-time assertion only |
| `competitions/adapter/booking` | `reservation_test.go` (**25 L**) | compile-time assertion only (with reasoning) |
| `socialplay/adapter/booking` | **none** | none |
| `socialplay/adapter/facilities` | **none** | none |

Verified by listing every `internal/*/adapter/*/` directory and running `wc -l`
on each `_test.go`. **`internal/booking/adapter/identity/` contained exactly
one file — `lookup.go` — from its creation in T11.5 until PR #151.** The
adapter that joins Booking to Identity had no test of any kind for two
sprints. It is not that a fake hid a real bug; it is that **nothing at all
was pointed at the code where the bug had to live.**

**PE (why the mitigation is cheap, and why "test everything against the real
thing" is the wrong ask).** This project already has the pattern, twice,
Docker-free, and it predates T12.
`internal/payments/adapter/socialplay/registration_updater_test.go` wires the
**real** `socialplayapp.NewService` with in-memory repository fakes beneath it
and drives the adapter through it. PR #151's fix test uses the identical
shape with the real `identityapp.Service`. So the recommendation is not a new
discipline to invent — it is a T11.9-style generalisation of a pattern that
already exists in the repo and already works without Docker.

The cost is bounded and countable: **5 packages**. Three currently have a
compile-time assertion only, two have nothing. That is a small, well-specified
chore ticket, not an open-ended testing mandate.

**QA (how this compares to T10's and T11's related findings, since the
ceremony was asked to contrast rather than conflate).**

- **T10 retro finding 5** — "absence assertion checks syntax, not semantics":
  a test that ran, passed, and proved nothing about the thing it named.
- **T11 retro finding, T11.9's discovery** — `CompetitionId:
  "no-such-competition"` short-circuited on a `uuidShape` guard before
  reaching the repository, so a test exercised a different code path than its
  name claimed. A test that ran, passed, and proved the wrong thing.
- **T12's** — no test existed at all on the code in question. Not a vacuous
  test; an **empty set**.

These are three genuinely different failures, and only the third is fixed by
"point a test at this package." Worth noting the family resemblance though:
in all three, a *green* signal was read as evidence about code the signal
never touched. What changes each time is the reason it never touched it.

**Recorded disagreement — QA vs. PE on how far the mitigation goes.**
- **QA:** the five-package backfill is necessary but treats the symptom. The
  deeper property is that a *port contract change* (T12.9 splitting `User.ID`
  from `User.Subject`) had no mechanism forcing a re-check of every adapter
  implementing a port that touches the changed concept. Backfilling tests
  makes the next such change *detectable*; it does not make it *checked*. QA
  wants a plan-level rule: a ticket that changes an identifier's meaning or a
  port's contract must enumerate every adapter on that seam in its PR body.
- **PE:** that rule is real but it is finding 1's rule wearing different
  clothes, and this project has a bad habit of adopting the same fix twice in
  two shapes and then discovering a third shape (finding 1's own history:
  migration numbers T6, T10; `errors.go` T11; capability provenance T12).
  Backfill the five tests — they are cheap, mechanical, and would have caught
  this specific bug outright — and let finding 1's dependency-completeness
  check carry the general case, since a port-contract change *is* a
  capability the downstream consumers depend on.
- **Unresolved.** Both agree the five-package backfill is adopted. Whether a
  separate port-contract-change rule is needed on top of finding 1's check is
  not settled, and both agree it should be **re-asked at T13's retro with
  evidence** rather than decided now — specifically: did the
  dependency-completeness check, once adopted, catch anything of this shape?

**Change for T13, adopted:** a chore ticket backfilling a behavioural,
Docker-free test for each of the **five** cross-context adapter packages that
lack one (`booking/adapter/facilities`, `competitions/adapter/facilities`,
`competitions/adapter/booking`, `socialplay/adapter/booking`,
`socialplay/adapter/facilities`), following the shape already established by
`payments/adapter/socialplay/registration_updater_test.go` — the real
other-context `app.Service` over in-memory repository fakes. Each test must
exercise the adapter with a realistically-shaped identifier, not a
convenient one; the whole bug was that `auth0|abc123` never appeared in a
test. And per CLAUDE.md rule 10 each must be mutation-checked, since a
compile-time assertion dressed up as a behavioural test is exactly the
failure this ticket exists to end.

## 3. Two causes, one symptom: #146's diagnosis was complete-looking and wrong, and only a real end-to-end test proved it — the escalation that followed was the right call

**QA (what happened, from the PR and the issue).** Issue #146 diagnosed the
break accurately and completely as far as it went, with the exact call chain
and the exact guard. PR #151 applied the one-line fix it prescribed
(`GetUser` → `UserBySubject`), verified it non-vacuously by reverting against
the final tests, and then — building the end-to-end test that #146 itself had
recommended — discovered a **second, independent cause**:
`internal/booking/app/recurring_hire.go:114` guards `ActorUserID` with
`uuidShape.MatchString` and returns `ErrUserNotFound` **before
`port.IdentityLookup` is consulted at all**.

Verified in the current tree: the guard is still at line 114, and
`make test-domain` is green with it there — because the test that pins the
defect asserts the *current, broken* behaviour deliberately.

**The reason the first diagnosis looked complete is worth stating precisely,
because it generalises.** Both causes produce the **identical** outward
symptom: `ErrUserNotFound` → `NotFound`. Sequentially, the app-layer guard
fires *first*, so it is the one a real caller actually hits — yet #146
diagnosed the *second* one, correctly, by reading the code inward from the
adapter. Reading a call chain from either end produces a true, sufficient,
and incomplete explanation when two independent gates return the same error.
Nothing short of executing the whole path distinguishes them.

**PdE — is "write the real end-to-end test before believing a diagnosis is
complete" generalisable, or did it just happen once?** Generalisable, but the
sharp form is narrower than that sentence. The general version ("test before
you believe") is already CLAUDE.md rule 1 and rule 10 and adds nothing. The
version that has teeth is conditional and cheap to check:

> When two or more independent guards on the same call path can return the
> **same** error sentinel or status code, a fix that removes one of them is
> not proven by a unit test at that guard's level. It requires a test that
> traverses the whole path.

That is testable against this repo immediately: `uuidShape` guards and
`ErrUserNotFound` are used at several layers in several contexts, by
deliberate convention. The condition — same sentinel, multiple layers — is
mechanically greppable, which makes it a checklist item rather than an
aspiration. It is also, notably, the same structural trap T11.9 found in
`competitions` (a `uuidShape` guard short-circuiting before the repository,
with an identical outward `NotFound`), which means this codebase has now been
bitten by *the same guard idiom producing the same ambiguous symptom* twice
in two sprints. That is enough repetition to name it.

**PO (evaluating the escalation, against this project's own yardstick).** The
question posed was whether escalating via a well-specified issue rather than
forcing a fix under review pressure was correct, measured against T11 retro
finding 4 and T12's own A5. Assessed:

1. **The alternative was concretely worse, and this was demonstrated rather
   than asserted.** PR #151 traced what naively deleting the guard would do:
   the actor is persisted as `recurring_hire_templates.requested_by_user_id
   uuid NOT NULL REFERENCES identity_users(id)`, written through the Postgres
   adapter's `mustUUID()`, **which panics on a non-uuid**. So the "obvious"
   fix converts a clean `NotFound` into a server panic plus an FK violation.
   A fix that turns a wrong answer into a crash is not a fix.
2. **The real fix is a genuine design decision, and the issue says so
   specifically.** #152 spells out that `port.IdentityLookup` deliberately
   exposes no method returning a `User` or `User.ID` (its own doc comment
   explains why), so adding subject→uuid resolution is an interface change,
   not an addition; that **every** Booking handler `actor(ctx)` call site
   needs the same translation; and that the handler-boundary-vs-app-layer
   placement should be "confirmed rather than assumed." It even instructs the
   future implementer not to trust its own call-site list without re-checking.
   That is a well-specified escalation, not a shrug.
3. **The gap is pinned in executable form, not only in prose.**
   `TestRequestRecurringHire_SubjectActorStillBlockedByAppLayerUUIDGuard`
   asserts the current defective behaviour, with a doc comment stating exactly
   how to flip it. This is materially stronger than T11 finding 4's standard —
   there, the durable record owed was "an issue, or a `HANDOFF.md` bullet at
   minimum." Here there is an issue *and* a failing-by-design test that a
   future refactor cannot silently satisfy.
4. **The partial fix was not oversold.** #146 was **deliberately left open**
   with a comment explaining precisely why: *"Leaving this issue open until
   #152 lands and `RequestRecurringHire` actually works for a real caller
   end-to-end, rather than closing it on the strength of the partial fix."*
   Under `sprint-process.md` DoD step 5 closing is a manual judgement call on
   this branch topology, and the judgement exercised was the conservative one.

**Verdict: the escalation was correct on all four counts**, and it is the
strongest instance of A5's discipline this sprint produced. **One legibility
defect, recorded because it is cheap to fix and would mislead a future
reader:** PR #151's *title* is "Fix: Booking's `RequestRecurringHire` broken
for every caller (**closes #146**)" and its body opens "Closes #146." The
issue was then, correctly, not closed. Anyone scanning merge history — which
is exactly how `HANDOFF.md`'s Docs index rows get built — would conclude the
capability works. It does not.

**Change for T13, adopted:** a PR that fixes part of an issue says so in its
title and body — "partial fix for #N", never "closes #N" — and names what
remains. `sprint-process.md`'s DoD step 5 already makes closing a manual
judgement; this makes the PR's own headline consistent with that judgement
instead of pre-empting it.

## 4. The sprint goal, stated precisely: the mechanism is real and universal; "every authorization check" is not — 11 tracked exceptions, and the honest sentence is different from the one the goal used

**PM (what unambiguously shipped, verified).** The goal was:

> *"Every authorization check in this codebase stops trusting a
> caller-supplied `actor_user_id`/`actor_player_id` and starts trusting a
> verified principal derived from a signed token…"*

Checked, and a great deal of it is simply true:

- **All six bounded contexts are migrated.** Booking (6 RPCs) + Facilities
  (4) via T12.7; Social Play (6) + Payments (3) + Competitions (3) via T12.8;
  Identity (2) via T12.9. **24 RPCs** now resolve their actor from a verified
  principal.
- **The wire field is ignored, not merely deprecated, and this is
  mutation-proven in all three migration tickets** — each reintroduced the
  naive `if err != nil { actor = req.GetActorUserId() }` fallback and captured
  the resulting failures. The reviewer independently re-ran the mutation on
  T12.4 and T12.8 rather than trusting the capture.
- **The failure mode is structurally unexpressible, not merely tested
  against.** `auth.RequireSubject(ctx)` takes only a context and has no access
  to the request message, so a handler cannot consult the wire field by
  accident.
- **The public/authenticated split cannot silently rot.** Each context exports
  both `AuthenticatedMethods()` and `PublicMethods()`, and a test reads the
  real generated `ServiceDesc` and fails if any RPC is in neither list, in
  both, or names a method that no longer exists. A new RPC cannot become
  silently public.
- **The `CreateUser` squatting DoS is closed and proven closed** by a test
  replaying the original attack, shown failing against restored pre-T12.9
  behaviour with the attacker's chosen uuid visible in the captured output.

That is substantial and should not be undersold. It is the single largest
security improvement in this project's history.

**BA (why the goal sentence nonetheless overstates, and what the honest
sentence is).** Nineteen GitHub issues were opened during T12 — four at
Ceremony 1 (#123–#126) and **fifteen during execution**. All nineteen are
open. Eleven of them qualify the sprint goal directly:

| Issue | Residual gap | Class |
|---|---|---|
| #135 | Fail-open vs fail-closed when the `TokenVerifier` **panics** — undecided | mechanism |
| #136 | An unconfigured (nil) `TokenVerifier` should be a startup failure, isn't | mechanism |
| #137 | **No remote JWKS `KeySource`** — auth cannot verify a token from a live IdP | mechanism |
| #138 | `internal/platform/**` tests **do not run in `make ci`** — including the auth spine's | mechanism |
| #144 | `CancelBooking`/`CreateBooking` have **no authorization check at all** | never had one |
| #145 | Pre-existing `owner_id` rows are uuids; a real `Principal.Subject` is not — forward-migration gap | data |
| #146 | `RequestRecurringHire` broken for every caller (partially fixed) | live defect |
| #152 | Subject→`User.ID` translation needed; `RequestRecurringHire` still broken | live defect |
| #147 | Roster reads leak every registrant/entrant to anyone holding an id | never had one |
| #148 | `ConfirmOnlinePayment` has no owner check — anyone holding a `payment_id` can capture | never had one |
| #149 | Payments still accepts **caller-supplied ownership facts** | fact-fabrication |

Three of these are load-bearing against the goal's own wording:

1. **#149 is the sharpest, and its own text puts it best: "It closes
   impersonation; it does not close fact-fabrication."** T12.8 made the
   *actor* verified on Payments' three write RPCs. The **ownership facts**
   those checks compare the actor against — `game_host_id`,
   `booking_host_id`, `entrant_player_id`, and the assigned-admin lists — are
   still supplied by the same caller. A caller can no longer claim to *be* the
   Host, but can still assert *who the Host is* and name themselves. The
   authorization check now starts from a verified principal and compares it
   against an unverified fact. That is a real narrowing of the gap, and it is
   not what "starts trusting a verified principal" implies to a reader.
2. **#144, #147 and #148 are RPCs with no authorization check to migrate.**
   The sprint migrated every check that existed; it did not add checks where
   none existed, and A11 Ruling 3 correctly scoped it away from doing so
   (adding an owner to the Booking aggregate is a domain change). But the goal
   says "*every* authorization check," and a reader naturally hears "every RPC
   that needs one." `CancelBooking` — *anyone holding a booking id can cancel
   it* — is as true after this sprint as before.
3. **#137 means no token from a real identity provider can be verified
   today.** Verification is real, RS256, checks signature/`exp`/`nbf`/`iss`/
   `aud`, and is tested — against local key material and an in-process key
   source. There is no remote JWKS fetcher.

**The honest sentence, which this retro proposes as the one `HANDOFF.md` and
T13's plan should carry:**

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

**PE — was ADR-0013's framing about the IdP honest, and did A1's "buildable
now" hold?** Both hold, and A1's central distinction was vindicated by
events. A1 split "real auth" into the IdP-provisioning half (*"not doable from
a coding session"*) and the verification half (*"entirely doable here"*), and
the verification half shipped working and tested. ADR-0013 (read directly,
`docs/adr/0013-verified-caller-identity-as-a-platform-capability.md`, lines
205–225) states it in the same terms `HANDOFF.md` uses for Jenkins:
*"Provisioning an identity provider tenant, registering the application, and
publishing a JWKS endpoint is server-side infrastructure work that no coding
session can perform,"* concluding that "auth exists" means *verification and
enforcement exist and are tested*, not *a production IdP is wired up*.

**Where A1 turned out to be less complete than stated**: it framed the
IdP-shaped residue as purely a *hosting* gap. Three of the eleven residuals
(#137 remote JWKS, #145 the uuid-vs-subject data migration, and #152's
subject-vs-uuid boundary translation) are **repo-side code consequences of
not having an IdP**, not hosting work. The absence of a real subject value in
this environment is precisely why nothing exercised a non-uuid actor until
T12.8 reasoned about it abstractly. So A1's honesty is not in question — its
completeness is: "the server-side half is out of reach" was true, and
"therefore the remaining work is server-side" did not follow.

## 5. The plan's predictions, audited: A3's correction held, A10 held again, A2's roll-call could not be scored, A9 scored four ways — and T12.4's Instructions carried an unmarked factual claim that was false

**PE (A3 — confirmed, including the correction the ceremony made by running
the command).** A3 predicted the retro's own bare `vet-integration` snippet
would fail on missing generated code and must depend on `generate`. Verified
in the merged `Makefile`:

```make
vet-integration: generate
	go vet -tags=integration ./...

ci: generate tidy lint test-domain vet-integration test-tools generate-client lint-web test-web build-web
```

Both the dependency and the after-`test-domain` placement landed exactly as
specified. This is the second consecutive sprint in which a ceremony caught a
plan defect **by running a command rather than transcribing it** (T11's A3 was
the first). Credit where due: T11 retro recommendation 1 → T12.1 is now the
cleanest retro→plan→merged-gate pipeline this project has, and it worked
end to end in one sprint.

**QA (A9(b) — the disagreement A9 deliberately left open is now scoreable,
and the answer is mixed).** A9(b) recorded QA's premise (implementers skip
full `make ci`, so the gate needs prose backup) as *"directly checkable
against T12's own PR verification lists."* Checked, every Go-touching ticket:

| Ticket | Integration-tagged files in its contexts? | Ran the check? | Who |
|---|---|---|---|
| T12.1 (#128) | — | built it, proved it non-vacuous | implementer |
| T12.2 (#140) | — | yes | implementer |
| T12.3 (#133) | payments (3) | yes — as raw `go vet -tags=integration ./...` | implementer |
| T12.4 (#132) | socialplay (5) | **not in its own record** | **reviewer**, in a fresh worktree |
| T12.7 (#142) | booking (1) | yes | implementer |
| T12.9 (#143) | — | yes | implementer |
| T12.8 (#150) | socialplay, payments, competitions (10) | yes | implementer + reviewer |
| PR #151 | booking (1) | yes | implementer |

**QA's premise is partly borne out and PE's conclusion still holds.** One of
eight PRs — T12.4, touching the context with the *most* integration-tagged
files — has no record of the check in its own commit message or PR body; the
reviewer ran it independently and found nothing broken. And two implementers
who *did* run it invoked the raw command rather than the new `make
vet-integration` target, which suggests the target is not yet the reflex even
where the habit is. But **no break occurred, and the gate plus a reviewer who
runs it caught the one gap**. PE's original position — one executable gate
beats three prose reminders — survives the sprint's evidence; QA's — that
implementers skip it — is confirmed at a rate of 1 in 8.

- **QA (revised position):** drop the CLAUDE.md prose request; it was the
  weaker half. Ask instead for the one thing that is mechanical: since the
  reviewer already re-runs these gates, make "`make vet-integration` was run,
  by implementer or reviewer" an explicit line in the review, so the 1-in-8
  is visible rather than incidental.
- **PE:** agreed, and notes this is not the prose he objected to — it is a
  reviewer checklist item, which is enforcement at the point of the decision.
- **Resolved.** A9(b) is closed. Adopted as stated.

**PdE (A9(a) — the roll-call exit condition did not fire, and cannot be
scored).** A9(a) declined polling for T12 with a named exit condition: a
silent agent caught by the roll-call closes the question; a silent agent
missed by it reopens polling; **no silent agent means it carries forward with
one sprint of evidence.** Checked: all 9 dispatched tickets produced a PR, and
the sprint ran in one continuous 1h47m block with no overnight gap — the
condition that produced T11's failure never arose. So the third branch fires:
**carried forward, unresolved, with one sprint of non-evidence.** Worth
stating plainly that this is *weak* evidence: a sprint with no silent agent
does not test a mechanism for detecting silent agents. If T13 also runs in one
unbroken block, this question should be closed as unfalsifiable-in-practice
rather than re-recorded a fourth time.

**PE (A9(d) and A9(e) — both held).** A9(d) declined the fixture-ID lint rule
with a stated trigger ("if a sixth instance appears in a file the shared
constant blocks do not cover"). No sixth instance appeared this sprint;
correctly not reopened. A9(e) made fresh-worktree re-verification **required**
for PRs touching an integration-tagged file or a multi-ticket shared-append
file. Verified in the PR record: T12.7, T12.9 and T12.8 each performed and
recorded it, and T12.9's caught nothing while T12.7's confirmed a clean
rebase — but the reviewer's *own* fresh-worktree pass on T12.9 is what found
finding 1's collision. The rule paid, and it paid on the reviewer's side.

**PO (A9(c) and T12.6 — resolved and correctly implemented).** The
board-of-record question is closed. `sprint-process.md` now carries the
split-by-lifetime rule, its Ceremony 1 section states the sprint plan is the
board of record for in-sprint tickets, DoD step 5 is amended so most tickets
have no issue to close, T11's nine tickets are explicitly declared never to
have been owed issues, and the four Ceremony-1 issues are cross-linked as
worked examples. Read directly and confirmed. **This is the first
disagreement in this project's history to be carried by a retro, resolved by
the next ceremony, implemented as a ticket, and verified in the retro after —
a complete loop.**

**PdE (A10 — held again, second consecutive sprint).** Migration `0019` was
pre-assigned to T12.9 and `db/migrations/` now ends at
`0019_identity_subject.sql`. Exactly one ticket added a migration, as
predicted; A10's checked-and-does-not-fire note for T12.3 (the `refunded`
CHECK already existing at `0005_payments.sql:30`) was independently re-verified
by T12.3's own commit. T12.9 also *strengthened* the spec — `subject text NOT
NULL UNIQUE` rather than the ticket's literal `UNIQUE` — with the reason
stated (Postgres treats NULLs as distinct under UNIQUE, so a nullable column
silently permits unlimited subject-less rows) and the safety argument checked
(no seed migration inserts into `identity_users`). A deliberate, disclosed,
reasoned strengthening of a plan instruction is the behaviour A5 and finding 5
have been trying to produce for three sprints.

**PdE (A12's table — right about what it examined, structurally blind to what
mattered).** A12 predicted the `cmd/server/main.go` T12.7/T12.9 same-wave
overlap. Outcome: **the predicted collision did not occur** (T12.9's merge
resolved `main.go` cleanly because A11 Ruling 2 had reduced it to one
composable line, exactly as designed), while an unpredicted collision occurred
one directory over. A12's four "checked, does not fire" rows — the three
`errors.go` files and the two wave-separated proto files — all held: T12.8
branched from a base containing T12.3's and T12.4's proto additions and hit no
conflict. So A12 was accurate on every question it asked. Finding 1 is about
the question it did not know to ask.

**QA (a plan error nobody was assigned to find — and it is T11 finding 5's
exact signature, one level out).** T11 finding 5 established that in T11's
A8 table, *the single claim carrying no evidence marker was the single claim
that was false*, and T12's A6 adopted evidence-marking for every cell of the
cross-context dependency table. **A8 was duly evidence-marked and its claims
held.** But the discipline was applied to the *table only*, not to ticket
**Instructions** — and this sprint's false claim was in an Instruction.
T12.4's item 6:

> *"already-cancelled Game → `FailedPrecondition` (the domain's existing
> `ErrIllegalStatusTransition` already produces this — check the existing
> mapping before adding one)."*

Verified directly in
`internal/socialplay/adapter/grpcapi/handler.go`: `ErrGameCancelled` maps to
`FailedPrecondition` (line 366), while **`ErrIllegalStatusTransition` is
grouped with the `InvalidArgument` cases** (line 372), sharing that mapping
with three waitlist transition paths. The plan's factual claim is false.
T12.4's implementer caught it, routed the already-cancelled case through
`ErrGameCancelled` instead, and stated why re-mapping the sentinel globally
would have *"silently changed unrelated RPCs' response codes."* The reviewer
then independently re-derived it by reading `toStatus` directly, and noted
that the sibling T12.3 PR had flagged the same mapping inconsistency
independently (now issue #131) — *"good convergent evidence."*

**What saved this was the second half of the same sentence.** "Check the
existing mapping before adding one" is a verify-don't-assume instruction, and
it is the reason a false premise cost nothing. That is the same instruction
shape that found finding 2's regression. Two independent confirmations in one
sprint that *the cheapest thing a plan can do is tell an implementer to check
the plan's own claim.*

**Change for T13, adopted:** T12's A6 evidence-marking discipline extends from
the cross-context dependency table to **any factual claim about existing code
in a ticket's Instructions** — a named sentinel, an existing mapping, an
existing helper, a line number. Either mark it (what was checked, how) or
phrase it as an instruction to check. A plan may say "confirm which code
`ErrX` maps to before relying on it"; it may not assert the mapping it has not
read.

## 6. A5 produced a durable record for all 19 disclosed gaps — but the mechanism that delivered it was a reviewer-side backstop, not the PR-side rule A5 actually wrote

**PO (the outcome, which is a genuine first).** A5's standing rule was that a
disclosed-but-not-closed gap gets a GitHub issue rather than a paragraph.
Measured against T11, the improvement is dramatic: T11 opened **zero** issues
for disclosed gaps (finding 4's whole subject); T12 opened **19**, of which
15 came out of execution rather than the ceremony. Cross-checked every PR
body against the issue list: **every gap disclosed in a T12 PR has a tracked
issue.** No prose-only deferral survived the sprint. That is exactly what T11
finding 4 asked for, delivered in full.

**BA (the mechanism, which is not what A5 specified, and matters for whether
it survives).** A5's text is specific about *who* creates the record:

> *"the same PR or review that records the decision creates the durable
> record."*

Reconstructed from issue `created_at` against PR `created_at`/`merged_at`:

| Source | Issues | Created by |
|---|---|---|
| T12.8 (#150) | #146, #147, #148, #149 | **the implementer**, before its own PR was opened (17:45:42–17:46:33Z vs. PR at 17:47:42Z) |
| T12.1, T12.2, T12.3, T12.5, T12.7 | #129, #130, #131, #134, #135, #136, #137, #138, #144, #145 | **the reviewer**, after the PR opened, minutes before merge |
| PR #151 | #152 | **the reviewer**, 25 seconds *after* merge |

**Exactly one of the eight PRs followed A5's literal text.** T12.8 opened its
own four issues before its PR existed. Everywhere else the PR *recommended*
an issue and the reviewing session created it — which #145's own body records
in so many words: *"the PR named this explicitly and recommended an issue but
didn't open one."* PR #151 was explicit about declining: *"I have not filed
one, to avoid pre-empting how you want it scoped."*

**QA (why a perfect outcome is still a finding).** The reviewer-side backstop
is not a worse mechanism — arguably it is better, since a reviewer sees
disclosures across several PRs and can scope, label and cross-link them
consistently (all 19 carry `role:*`/`type:*` labels per T12.6's taxonomy, and
several cross-reference each other). What makes it a finding is that **it is
one person deep and it is not the rule anyone wrote down.** Every gap this
sprint depended on a single reviewing session reading every disclosure
paragraph and acting on it before or just after merge. Remove that session's
attention on any one PR and the gap reverts to prose in a merged PR body —
the precise T11 finding-4 failure mode, in a sprint that believes it has
solved it. And because the *outcome* was 19-for-19, nothing in the record
signals the dependency.

PR #151's declining reason is also worth taking seriously rather than
treating as non-compliance: an implementer mid-review genuinely may not know
how a follow-up should be scoped, and #152 as the reviewer eventually wrote it
is a better-specified issue than the implementer would likely have produced.
So the rule as written may be asking the wrong party.

**Recorded disagreement — PO vs. BA on which side to change.**
- **BA:** amend A5 to describe what actually works: the **PR must disclose the
  gap in a named section**, and the **reviewer must open the issue before
  merging**, with the review naming each issue it opened. That is descriptive
  of 18 of 19 cases, puts the obligation on the party with the best context,
  and — critically — makes the omission *visible*, because a review with a
  disclosure section and no issue list is obviously incomplete.
- **PO:** that formalises a single point of failure. The implementer knows
  the gap best at the moment of discovery, and T12.8 — the one ticket that
  followed the letter — produced the four best-specified issues of the sprint
  (#146 in particular is a complete diagnosis with call chain, root cause and
  suggested fix, written before anyone reviewed it). Keep A5's text and add
  the reviewer as an explicit *backstop* rather than replacing the primary.
- **Unresolved.** Both agree on the concrete mechanical half: **the review
  must enumerate the issues opened for that PR's disclosures, or state that
  there were none.** That single line makes the backstop auditable regardless
  of which party is nominally primary, and it is adopted either way.

---

## Recommendations for T13's Ceremony 1 and 2

Concrete and mechanical, in the spirit of T11 retro finding 2 → T12.1 and
T10 retro finding 4 → T11's A5:

1. **Add a dependency-completeness check to the dispatch-isolation
   section** (finding 1). For every arrow in the wave dependency graph, one
   line: what the upstream ticket's AC *delivers*, what each downstream
   ticket's AC *consumes*. Any capability appearing on a consumer's side and
   no producer's side is assigned to exactly one ticket before dispatch.
   This is the highest-value item here: it is the only one of these
   recommendations that addresses a collision class the existing two
   file-overlap questions structurally cannot see.
2. **Ticket a backfill of behavioural tests for the five cross-context
   adapter packages that lack one** (finding 2) —
   `booking/adapter/facilities`, `competitions/adapter/facilities`,
   `competitions/adapter/booking`, `socialplay/adapter/booking`,
   `socialplay/adapter/facilities`. Follow the existing Docker-free shape in
   `payments/adapter/socialplay/registration_updater_test.go` (real
   other-context `app.Service` over in-memory repo fakes). Each test must use
   a realistically-shaped identifier — a non-uuid subject where the seam
   carries one — and each must be mutation-checked, since three of the five
   currently hold a compile-time assertion that a reader could mistake for
   coverage.
3. **Extend evidence-marking from the dependency table to every factual claim
   about existing code in ticket Instructions** (finding 5). Mark it (what was
   checked, how) or phrase it as an instruction to check. T12.4's item 6
   asserted a sentinel mapping that is false; the same sentence's
   "check before adding one" is what made it harmless.
4. **A partial fix says "partial fix for #N", never "closes #N"** (finding 3),
   and names what remains. PR #151 closed half a bug under a title claiming
   the whole one, against an issue correctly left open.
5. **Every review enumerates the issues it opened for that PR's disclosures,
   or states there were none** (finding 6). Adopted regardless of how the
   PO/BA disagreement about who is primary resolves — it is what makes the
   backstop auditable.
6. **Add "`make vet-integration` run — by implementer or reviewer" as an
   explicit review line** (finding 5, A9(b), now resolved). One of eight T12
   PRs had no record of it; the reviewer covered it silently. Make the cover
   visible rather than incidental. The CLAUDE.md prose half of the T11
   disagreement is **dropped** — the sprint's evidence did not support it.
7. **Carry T12's residual auth work as a named, prioritised group, not 11
   loose issues** (finding 4). They are not equivalent: #146/#152 is a live
   broken capability, #138 means the auth spine's own tests run in no
   Docker-free gate, #144/#147/#148 are missing checks, #149 is
   fact-fabrication, #137/#145 are blocked on an IdP that does not exist.
   T13's Ceremony 1 should rank them and state which it takes, so "the
   mechanism exists with 11 tracked exceptions" does not quietly become the
   permanent state.
8. **State the sprint goal in the honest form** (finding 4). Finding 4's
   proposed sentence — mechanism universal, 11 named exceptions — should
   replace the "every authorization check" phrasing wherever `HANDOFF.md` and
   T13's plan describe T12's outcome. Overclaiming here is worse than
   elsewhere: a future ticket that believes authorization is finished will not
   look for #149.
9. **Fix `HANDOFF.md`'s T12 Docs-index row, and notice this is now
   structural.** The row reads *"not yet written"* for the retro and *"not yet
   opened"* for the reviews — both stale (nine reviewed PRs; this retro). T12's
   own ceremony had to make the identical correction to T11's row. The cause
   is mechanical and repeatable: by established precedent (verified — PRs #110
   and #122 each touched exactly `docs/LESSONS.md` and the retro doc, and this
   PR does the same) **a retro PR never updates the index row pointing at
   itself**, so every sprint inherits a stale row. Either the retro PR takes
   the row too, or `sprint-process.md`'s Ceremony 1 gains an explicit
   "correct the previous sprint's Docs-index row" step. Pick one; it has now
   silently recurred twice.
10. **Carried forward, unresolved:** A9(a)'s roll-call-vs-polling question,
    which T12 could not score because it had no silent agent and no
    multi-block work period. If T13 also runs unbroken, close it as
    unfalsifiable-in-practice rather than re-recording it a fourth time. Also
    open: QA's port-contract-change rule (finding 2), deliberately deferred to
    be re-asked at T13's retro with evidence on whether recommendation 1's
    dependency-completeness check catches that shape.

---

## No finding on

**No finding on T12.6 (board of record) beyond finding 5's credit.** The
ticket implemented A7's resolution faithfully, amended both sections of
`sprint-process.md`, corrected DoD step 5, preserved T11.8's manual-close
finding intact, cross-linked the four worked examples, and explicitly recorded
the decision *not* to backfill T11's nine tickets as a decision rather than an
omission. Verified by reading the merged document. A retro that manufactures a
finding here would be filling a slot.

**No finding on T12.1 (`vet-integration`) beyond the audit in finding 5.** The
target landed with the `generate` dependency A3 predicted, in the position the
T11 retro specified, was proven non-vacuous by breaking an integration-tagged
call site and watching it fail, and corrected CLAUDE.md's stale
one-file-eleven-files claim while explicitly stating that the correction was
**not** the reminder QA argued for — preserving A9(b)'s open disagreement for
this ceremony to score instead of settling it by ticket shape. That restraint
is unusual and worth naming.

**No finding on T12.5 (WCAG) or T12.3 (`RefundPayment`).** T12.5 fixed the
verified 3.3.4 defect, stated its checked negatives, and — per A14's one point
of QA/PM agreement — opened **#134** recording that a full manual
screen-reader pass is still owed, so QA's concern that "finding one defect
where the audit never looked is weak evidence there is exactly one" is
inherited as a named gap rather than a silent one. T12.3 shipped the refund
path, reused the existing authorization concept rather than inventing a
second, and disclosed two real scope gaps as #130 and #131 — one of which
independently corroborated T12.4's error-mapping correction. Neither produced
a process-level finding.

**No finding on dispatch isolation itself.** Every ticket ran worktree-isolated
per A12, and no shared-checkout collision recurred. Finding 1 is the
capability-provenance class, filed as its own finding accordingly.

**A14's PdE/PE disagreement, scored — and PdE was substantively right, in a
form neither role predicted.** A14 recorded PdE's concern that T12.2's blast
radius was wider than anything attempted before, with the scoring condition:
*"if T12.7/T12.8/T12.9 each independently hit the same T12.2 defect, PdE was
right."* Two of the three consumers independently hit the same **T12.2 gap** —
not a defect in what T12.2 built, but the absence of something none of them
could proceed without (finding 1). PdE's proposed mitigation was a Wave-1.5
checkpoint: *"T12.7 merges and is reviewed before T12.8 and T12.9 are
dispatched at all."* Applied, that is precisely what would have prevented the
collision — T12.9 would have found `require.go` already present, which is the
outcome T12.8 actually got by being one wave later. PE's counter-argument
(the observe-only mode is the proving run, and serialising Wave 2 puts the DoS
fix behind two merges for schedule reasons) also held on its own terms: T12.2
broke no shipped flow, and T12.9 closed the DoS in the same wave it was
planned for. **Both were right about their own concern; PdE was right about
the one that fired.** T13 should apply the Wave-1.5 shape specifically where a
new platform capability has three or more first-time consumers — not as a
general rule, which is the over-correction PE correctly resisted.

**PM had limited independent material this sprint**, as in T9, T10 and T11.
PM's substantive contribution is embedded in finding 4's goal assessment,
which is a product-framing judgement rather than an engineering one: the
difference between "authorization is done" and "the mechanism is done, with
11 tracked exceptions" is the difference between a roadmap that schedules
#144/#147/#148/#149 and one that never looks at them again.
