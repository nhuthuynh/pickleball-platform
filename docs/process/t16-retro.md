# T16 Sprint Retro

Ceremony 3 per `docs/process/sprint-process.md`, six-role team (briefs:
`docs/agent-operating-handbook.md` Part B), held against
`docs/process/t16-sprint-plan.md` (§A0–§A12), `docs/process/t15-retro.md` as
the precedent and rigor bar, `HANDOFF.md`, and the real PR/issue history on
`nhuthuynh/white-label` (GitHub-side name `pickleball-platform`) — PRs
#196–#200, issues #124–#198.

Every timestamp, merge order, label, issue-state and interface-satisfaction
claim below was pulled from GitHub's own `created_at`/`merged_at`/
`submitted_at`/`closed_at` fields, from `git log --format="%H %P"` on the
actual merge commits, and from direct reads/builds of the merged tree —
never inferred from PR titles, review prose, or the sprint plan's
forward-looking text (CLAUDE.md rule 10).

**Verification performed before writing a single finding, per this
ceremony's own instruction to check the branch it is about to build on.**
`git fetch` + `git status` confirmed a clean worktree at the shared branch's
tip (`c0e9c7d`, T16.4's merge). `make generate && go build ./... && go vet
./... && make fmt-check && make vet-integration && make test-domain && make
test-adapters && make test-cmd && make test-platform && make gate-coverage`
were all run directly, not assumed from any PR's own account:

```
go build ./...                # clean
go vet ./...                  # clean
make fmt-check                 # OK — gofmt clean
make vet-integration           # clean
make test-domain               # ok, all 12 packages
make test-adapters             # ok, all 22 packages, including
                                #   internal/payments/adapter/competitions
make test-cmd                  # ok
make test-platform             # ok
make gate-coverage              # OK — all 41 package(s) executed by ci-checks
```

**The shared branch is not broken from this worktree's perspective — but it
was broken for real, on the shared branch's own tip, for 15m21s inside this
sprint, and finding 1 below is the full investigation of that window rather
than a note that it resolved itself.** Not re-verified here (no Docker
daemon reachable, same standing gap as every prior sprint): `make test` /
`make ci-integration`'s Docker-backed suite.

**Sprint outcome, stated before the findings that qualify it:** all 3
tickets (16 points, T16.2–T16.4) merged in a single unbroken work block —
plan-doc merge (PR #196) at 16:12:46Z to final ticket merge (T16.4, PR #200)
at 17:12:00Z, **59m14s wall-clock**, no session-limit interruption, no
worktree recovery needed. T16.2 and T16.3 were implemented concurrently in
isolated worktrees from the same base commit (665957a, the plan-doc merge)
per Wave 1's "no in-sprint dependency" design, then merged sequentially;
T16.4 (Wave 2) branched only after both had landed. Both "closes #N"-titled
PRs (T16.2 → #168, T16.4 → #125) actually performed the mandated close
within seconds of merging — the first live test of T16's own
Ceremony-1-amended DoD step 5, and it held both times. `#124` correctly
stayed open with the required "which half, why the other stays deferred"
comment. The merged-fix sweep reconciles exactly: `12 − 2 + 1 = 11`,
matching the live `totalCount` read at this ceremony's start.

**What did not go as smoothly as the closed-issue count implies, in one
sentence, so seven findings do not have to be read before the headline is
known:** the shared branch's own tip — the thing every later PR builds on
and every future reader `git clone`s — was genuinely uncompilable for over
fifteen minutes this sprint, and the PR whose review should have caught it
before merge instead asserted, in writing, that it had already checked
against the merged state. It had not. This is finding 1, and it is argued
in full below rather than accepted as "the shared branch briefly needed a
follow-up fix, which is normal."

---

## 1. The shared branch's tip was genuinely broken for 15m21s — not a case where nothing could have caught it, but one specific review's own verification claim that was false and checkably so

**QA (what actually happened, reconstructed from git, not from any PR's
prose).**

**T16.2 and T16.3 were both Wave 1 — dispatched with no in-sprint dependency
on each other, per §A11/§A12's design — and both branched from the same
commit, `665957a` (the T16 plan-doc merge), before either had merged.**
Verified directly:

```
$ git show --format="%H %P" -s 9aa78c7c...   # T16.3 PR head
9aa78c7c... 665957a89c8...
$ git show --format="%H %P" -s e1216d39...   # T16.2 PR head
e1216d39... 665957a89c8...
$ git merge-base e1216d39... d2ac53bea4...   # T16.3's merged tip
665957a89c8...
```

T16.2's own branch tree **never, at any point before its own merge,
contained T16.3's commits** — its merge-base with T16.3's merged tip is the
same common ancestor both started from, not T16.3's tip itself. This is a
plain fact about the commit graph, not an inference.

**T16.3 landed first** (PR #197, merged 16:28:35Z onto `665957a`, producing
`d2ac53b`). It widened `internal/competitions/port.Repository` with a new
method, `CancelAllActiveForCompetition`, and — per its own PR body's file
list — patched the two existing test fakes in
`internal/payments/adapter/competitions/` that implemented that interface at
the time (`competition_admin_reader_test.go`, `entry_updater_test.go`),
plus one more in `internal/competitions/adapter/grpcapi/`. This was a real,
correctly-executed blast-radius search: T16.3 found and fixed every
implementer of the widened interface **that existed in the tree it could
see.**

**T16.2, working concurrently and unaware of T16.3's widening, was
simultaneously authoring a brand-new file** —
`internal/payments/adapter/competitions/entry_lookup_test.go` — containing a
**third** implementer of the same interface, `entryLookupFakeRepository`,
assigned to a `competitionsport.Repository`-typed field
(`competitionsapp.ServiceOptions.Competitions`) inside that same file's
`newEntryLookupTestService` helper. This file did not exist when T16.3 ran
its sweep, so T16.3 could not have found it — **not a gap in T16.3's own
diligence.** As authored, against the interface as it stood at T16.2's own
branch point (`665957a`, pre-T16.3), `entryLookupFakeRepository` was
complete and compiled cleanly.

**T16.2 merged second** (PR #199, merged 16:56:39Z onto `d2ac53b`, producing
`cd31f86`). GitHub's merge combined T16.3's already-landed interface
widening with T16.2's new file — and the new file's fake still lacked the
method the interface now required. **Confirmed directly against the actual
committed tree:**

```
$ git show cd31f86:internal/payments/adapter/competitions/entry_lookup_test.go \
  | grep CancelAllActiveForCompetition
(no output — the method is absent)
```

`go vet ./internal/payments/adapter/competitions/...` at `cd31f86` fails:
`*entryLookupFakeRepository does not implement competitionsport.Repository
(missing method CancelAllActiveForCompetition)`. **This is exactly the
failure T16.4's review reported finding** — independently re-verified here
by this retro, not taken on T16.4's own account.

**T16.2's own review — submitted at 16:56:33Z, six seconds before the
PR it reviewed merged — explicitly claimed to have checked this:**

> *"Toolchain, fresh worktree, base already included T16.3 (mergeable_state
> clean, no test-merge needed): make generate, go build ./..., go vet ./...,
> ... make test-adapters ... — all green."*

**That claim is false, and checkably so from the commit graph alone.** The
review's `commit_id` is `e1216d3989dcd38a6fccf62cb05e5c1e9f98a654` — T16.2's
PR head — whose only parent is `665957a`, not `d2ac53b`. "Base already
included T16.3" did not describe the tree the reviewer actually tested; it
described a belief inferred from GitHub's `mergeable_state: clean` flag,
which only certifies the **absence of a textual merge conflict** — it says
nothing about whether the two branches' combined tree still *type-checks*.
A file that is entirely new on one branch and an interface that is widened
on the other never produce a textual conflict (they touch disjoint lines in
disjoint files), so `mergeable_state` was accurately `clean` — and
semantically broken anyway. The reviewer treated "no conflict" and "already
tested against the merged tree" as the same fact. They are not, and this is
the exact place they came apart.

**T16.4's review (PR #200, submitted 17:11:55Z) is what T16.2's review
described doing but did not do.** It explicitly checked out the *bare*
shared-branch tip (`cd31f86`) in a **separate worktree with none of T16.4's
own changes**, ran `go vet ./internal/payments/adapter/competitions/...`
directly against it, and only then attributed the failure — the same
discipline `sprint-process.md`'s worktree-recovery precedent already
sanctions for a different purpose (verify a claim from scratch, not by
reading it). This retro re-ran that exact check independently (above) and
confirms it: the break was real, on the shared branch's own tip, for the
full window between the two merges.

**The broken window, named precisely: `cd31f86` (16:56:39Z, T16.2's merge)
to `c0e9c7d` (17:12:00Z, T16.4's merge) — 15 minutes 21 seconds.** Shorter
than T14's 6h45m self-sweep gap, but the same *class* of failure T10's
retro named first: **"a green claim checked against the wrong commit."**
T10's instance was a fixture fix verified against a working tree that was
never `git add`'d; this one is a full toolchain run verified against a
branch tree that was never actually merged with its sibling. Different
mechanism, identical shape — a verification step that reports true of the
tree it was run against, misdescribed as true of a different tree it was
never run against.

**PdE (was this avoidable, argued not asserted).** Two questions, scored
separately because they have different answers:

1. **Could T16.3 have prevented this?** No. The third fake did not exist
   in any tree T16.3 could read. Its blast-radius search (patch every
   existing implementer) was complete for the world as it stood.
2. **Could T16.2's own review have caught this?** **Yes, straightforwardly,
   and the fix was already known-and-demonstrated elsewhere in the same
   sprint.** T16.3's own review (PR #197) verified against a base that
   "matched the shared branch tip exactly," because at that moment it did.
   T16.4's review verified against the *bare* shared-branch tip in an
   independent worktree, explicitly to avoid trusting an inference. T16.2's
   review had the same two options available and took neither — it
   inferred "already merged" from a flag that does not carry that meaning,
   at a moment (16:56:33Z) nearly 28 minutes after T16.3 had already landed
   on the shared branch, when a `git fetch` and a real test-merge would have
   caught the missing method in under a minute.

**Score, stated as the task asks: this is a real, catchable process gap, not
a case where nothing could have caught it before T16.4.** The candidate
theory this ceremony was asked to check — that no Go compile error existed
until T16.4's own new code first assigned the fake to an interface-typed
variable — **does not hold**: the interface-satisfaction-checked assignment
(`competitionsapp.NewService(competitionsapp.ServiceOptions{Competitions:
repo, ...})`, inside `newEntryLookupTestService`) was already present in
T16.2's own file, at T16.2's own merge. The break was live and
`go vet`-visible **at the moment `cd31f86` was created**, fifteen minutes
before T16.4's branch even existed. What actually happened is narrower and
more specific: one review's stated verification method was not the method
it actually used.

**Not a finding against ADR-0016's interim rule.** T16.4 did not "fix a gap
in a branch under review" — the break was already on the shared branch's own
tip when T16.4's session started; fixing a pre-existing break on the branch
you are building on, discovered by your own mandated toolchain run, is
exactly what the per-ticket DoD's testing step requires, not a review-time
gap-fix on someone else's open PR. T16.4's PR body says so explicitly and
correctly ("flagging here rather than folding it in silently, in case the
reviewer prefers it split into its own PR") and its review treated the
fix as in scope rather than out of bounds.

## 2. All three of T16 Ceremony 1's own `sprint-process.md` amendments held on their first live test — scored individually, not as a bundle

**BA.** These three amendments were written this same sprint (T16's own
Ceremony 1, from T15 retro recommendations 1/7, 2, and 4) and this is their
first live test, mirroring how T15.1's amendments were tested within T15
itself. Scored one at a time, per the task's own instruction not to treat
them as a bundle:

**(a) The dependency-completeness check's consumer-input clause.** T16.2's
own ticket text (§A10) named four specific things before dispatch:
`socialplayapp.Service.GetRegistrationByID`, `.GetGame`,
`competitionsapp.Service.GetEntryByID`, and the join key —
`RecordOfflinePaymentInput.PayableID`/`RefundPaymentInput.PayableID`.
**Checked directly against what was actually built, not assumed from the
ticket text matching the PR title:**

```
$ grep -n "func (s \*Service) GetRegistrationByID" internal/socialplay/app/service.go
461:func (s *Service) GetRegistrationByID(ctx context.Context, registrationID string) (domain.Registration, error) {
$ grep -n "func (s \*Service) GetGame\b" internal/socialplay/app/service.go
346:func (s *Service) GetGame(ctx context.Context, gameID string) (domain.Game, error) {
$ grep -n "func (s \*Service) GetEntryByID" internal/competitions/app/service.go
391:func (s *Service) GetEntryByID(ctx context.Context, entryID string) (domain.CompetitionEntry, error) {
$ grep -n "type RegistrationLookup\|type GameLookup\|type EntryLookup" internal/payments/port/*.go
internal/payments/port/entry_lookup.go:19:type EntryLookup interface {
internal/payments/port/game_lookup.go:16:type GameLookup interface {
internal/payments/port/registration_lookup.go:18:type RegistrationLookup interface {
```

Exact match, method for method, port for port. **Held.**

**(b) The mandatory-close-for-"closes #N" amendment.** T15 scored 0/2 on
this exact shape (PRs #191, #189). T16 scored **2/2**: #168 closed
`state_reason: completed` at 16:56:44Z, five seconds after PR #199 merged
(16:56:39Z); #125 closed at 17:12:06Z, six seconds after PR #200 merged
(17:12:00Z). Both were performed via the GitHub API with a comment naming
the resolving PR, both by the same account that merged, both essentially at
the merge moment rather than deferred to a later sweep. **Held, on its
first live test, after two consecutive sprint-level misses.**

**(c) The correction-not-just-closure clause.** T16.2's PR body committed to
posting a narrowing comment on #149 (not closing it — a correction to its
content). Verified via the API: comment
[#149#issuecomment-5303271723](https://github.com/nhuthuynh/pickleball-platform/issues/149#issuecomment-5303271723),
posted 16:56:58Z (22 seconds after #168's close), states plainly that four
of #149's five originally-named facts are now resolved
(`game_host_id`, `assigned_game_admin_user_ids`, `entrant_player_id`,
`assigned_competition_admin_user_ids`) and that `booking_host_id` is the one
that remains, blocked on D1. **Checked against the merged code, not taken on
the comment's own word**: `RecordOfflinePaymentInput`/`RefundPaymentInput`
retain only `BookingHostID` among the five original fields (the other four
are deleted from the Go structs, per grep of `internal/payments/app/
service.go`) — the comment's claim is accurate. **Held.**

**Score, stated together because the pattern is the point:** three
amendments, all written the same ceremony, all exercised for the first time
this sprint, all three held. This is a genuinely different outcome from
T15's Ceremony-1 amendments (the closure-sweep third state, worktree
recovery), which this retro's own predecessor found holding on some axes and
not others. Worth naming as a positive result rather than passing over it —
this retro is not only a list of what broke.

## 3. The merged-fix issue sweep — clean, reconciled exactly, run per the standing rule before ranking anything

**PO.** Per `sprint-process.md`'s DoD, this retro is sweep moment 1 ("the
retro runs it and reports it; the retro does not block on it"); T17's
Ceremony 1 remains the authoritative moment regardless of this result.

**Step 1 — list the open issues**, live, at this ceremony's start:
`list_issues(state: OPEN)` → **`totalCount: 11`**: #198, #195, #167, #164,
#149, #145, #144, #134, #130, #126, #124.

**Step 2 — reconcile arithmetically, before reading any issue
individually.** `open_at_end_of_T16_Ceremony_1 − closed_during_T16 +
opened_during_T16`. T16's own Ceremony 1 left the count at **12** (11 + 1
for #195, filed that ceremony). During execution: **#168** closed (T16.2,
PR #199) and **#125** closed (T16.4, PR #200) — `12 − 2 = 10`; **#198**
opened (T16.2's own sibling sweep) — `10 + 1 = 11`. **This matches the live
`totalCount: 11` exactly** — the same reconciliation method every prior
sweep has used, checked here rather than assumed clean because the two
closes both happened at review time.

**Step 3 — cross-reference merged T16 PRs against the open list.** Three PRs
merged this sprint (#197, #199, #200) plus the plan-doc PR (#196). Every
`#N` referenced in a title or body:

| PR | References | Open-list hit? |
|---|---|---|
| #197 (T16.3) | #124 (partial fix, correctly left open) | No — correctly not closed |
| #199 (T16.2) | #168 (closes, verified closed), #149 (correction, correctly not closed), #198 (opened) | No hits — #168's close already accounted for in step 2 |
| #200 (T16.4) | #125 (closes, verified closed), #130 (contrast, untouched), #198 (referenced, not touched) | No hits |

**Zero unclosed hits.** Both closes this sprint claimed at "closes #N" moment
were verified genuinely closed via `issue_read` (not taken from the PR
titles): #168 `state: closed`, `state_reason: completed`, `closed_at:
2026-08-15T16:56:44Z`; #125 `state: closed`, `state_reason: completed`,
`closed_at: 2026-08-15T17:12:06Z`. #124 verified genuinely still `state:
open`, carrying the required explanatory comment (§4 below).

**Sweep result: clean. First clean, fully-per-PR-closed sweep this
project's history has produced** — every prior sprint's sweep found at
least one bookkeeping gap (T13: 0/9 closed at review time; T14: self-swept
in an 11-second batch, acceptable-but-not-sufficient; T15: 2 of 3 "closes
#N" claims never performed, caught only by the retro). T16 is the first
sprint where the per-PR close (DoD step 5's optional-early-close, made
mandatory this same sprint for "closes #N" titles) actually discharged the
obligation at the moment it was supposed to, for both instances that
applied. **T17's Ceremony 1 still re-runs this sweep in full, per the
standing rule that a prior ceremony's clean result does not discharge the
next one — this retro's own closes should be re-verified from the API, not
trusted, the same as every prior sweep's output.**

## 4. #198 — a new, real, accurately-described carried-forward item, flagged for T17

**QA.** #198 was filed by T16.2's own sibling sweep (instruction 10),
correcting the T16 sprint plan's own §A10 dependency check, which had
concluded "no — `BookingHostID` is the only remaining one" and was wrong.
**Re-verified directly against the merged code, not taken from the issue's
own account:**

```
$ grep -n "EntrantPlayerID\|AssignedCompetitionAdminUserIDs" internal/payments/app/service.go | head -6
190:     EntrantPlayerID                 string
191:     AssignedCompetitionAdminUserIDs []string
```

`CreateOnlinePaymentInput` still carries both caller-supplied fields, and
`authorizeOnlineCreation(in)` (a separate check on a separate RPC from the
`authorizeOfflineRecording` T16.2 actually rewired) still authorizes against
them directly rather than through the `EntryLookup`/`CompetitionAdminReader`
ports T16.2 built. **#198's own description is accurate**: it correctly
identifies this as "mechanically trivial" to close (the exact ports already
exist and are already wired into `payments/app.Service`; the fix is
extending `authorizeOnlineCreation` to call them, mirroring
`authorizeCompetitionEntryRecording`), correctly distinguishes it from #149
(a different RPC, different check, not blocked on D1), and correctly notes
this asymmetry was named as far back as T15.5's own proto comment without
being fixed then either.

**State open, confirmed via the API** (`state: open`, no `closed_by_pull_
requests`). **Flagged for T17 planning consideration**, as the task
requires: this is a small, unblocked, well-specified ticket — closer in
shape to T16.4 (mechanical, push-mechanism already built, no open product
question) than to any of the D1/D2-blocked items, and should rank
accordingly rather than compete with genuinely larger or blocked work.

## 5. D1 and D2 remain unanswered — and D1's absence now shapes ticket *scoping*, a second time, which is a materially different and worse cost than being merely named

**Re-verified this ceremony, not assumed.** `issue_read(get_comments)` on
#144 (D1): **exactly one comment**, T14.3's original escalation
(2026-08-15T07:01:03Z) — unchanged since T14, unchanged by T15, unchanged by
T16. `docs/adr/0016-*.md`'s (D2) own `## Status` field: still **"Escalated
— awaiting the user's decision. This ADR decides nothing."** — unchanged.
Neither PR merged this sprint references #144 or ADR-0016.

**PE (the argument, not just the observation).** T14.3 first escalated D1
as a **blocker on a specific piece of work**: `CancelBooking` has no
authorization check, and building one means picking one of ADR-0015's four
options. That is a *named, contained* cost — one RPC, one ticket, waiting.
**T16.3 is a second, structurally different instance, and the difference is
the argument.** T16.3 did not merely *mention* D1 while deferring — it drew
the actual **shape of a shipped feature** around D1's absence: cancelling a
Game/Competition now cascades to Registrations/Entries (real, shipped,
tested) but deliberately **stops short of releasing the reserved courts**,
specifically because doing so would call `CancelBooking`, whose signature
ADR-0015's still-open D1 may change. That is not the same cost as "one
ticket is blocked" — it is evidence that **the blast radius of the
unanswered question has grown from the RPC it was originally about to a
second, independent capability (cascading cancellation) that merely calls
that RPC.** Two named instances now, not one:

1. `CancelBooking` itself has no authorization check (#144's original
   scope, four sprints old).
2. A cancelled Game/Competition's court-Bookings stay reserved forever,
   specifically because nobody will build against `CancelBooking`'s
   signature while it might still change (#124's court-Bookings half, new
   this sprint).

**This is compounding, not merely additive.** Every future ticket that would
otherwise call `CancelBooking` — a refund-on-cancel flow, a facility-side
bulk-cancel, anything — now inherits the same reason to stop short, for the
same unanswered question, without any of them individually re-escalating
it. The deferral tax is starting to spread from the RPC D1 is actually about
into everything adjacent to it, which is a materially worse trajectory than
"one escalation, ignored, repeated." **Stated plainly, not softened**: this
is D1's fourth deferral and the second sprint running in which its absence
has visibly shaped what a ticket is allowed to build, not merely what it is
allowed to decide.

**D2** has no comparable spread yet — T16 shipped no PR with a reviewer-
authored gap-fix to check (§7 below), so ADR-0016's interim rule was simply
never in a position to be tested this sprint, unlike T15's full-sprint
clean result. This is D2's **first** sprint with nothing to report either
way, not a second deferral of the same shape as D1's.

## 6. The T15→T16 arc: T15.5's own review already knew T16.2's exact shape at 15:36Z — the retro that reported the block, twenty minutes later, should have drafted the successor rather than merely recommending a check for it

**PdE (the position, argued).** The task asks whether T15's planning should
have anticipated a T16.2-shaped follow-up explicitly, rather than letting it
emerge as a natural-but-unplanned Ceremony-1 discovery. **The honest answer
splits in two, and the split is the argument:**

**T15's own *Ceremony 1* could not have anticipated T16.2's shape**, and
this is not a hindsight-privileged claim — T15.5's own discovery (that
`RecordOfflinePaymentInput` carries no join key from a Registration to its
Game) was a genuine, first-time enumeration of both `app.Service`s that
nobody had performed before T15.5 itself did it. Nothing available at T15's
planning time made that discoverable without doing the implementation work
that revealed it. This part of T15's retro (finding 1) already argued this
correctly, and this retro does not re-litigate it.

**But T15's *retro* (Ceremony 3, same day) is a different case, and it had
strictly more information than T15's Ceremony 1 did.** By the time T15's
retro ran (15:51:36Z), T15.5's own PR body and review (submitted 15:36:20Z)
had already, in full technical detail:

- named the exact missing methods (`GetRegistrationByID`-shaped,
  `GetGame`-shaped, `GetEntryByID`-shaped);
- named the exact existing join key each would need
  (`RecordOfflinePaymentInput.PayableID`);
- named the exact repository-level reads each would wrap
  (`RegistrationRepository.GetByID`, `GameRepository.GetByID`,
  `Repository.GetEntryByID` — all already implemented).

**This is not a forecast; this is the actual content of §A10 of T16's own
Ceremony 1, produced independently roughly twenty minutes later from the
same facts.** T15's retro (posted as comment
[#168#issuecomment-5302944092](https://github.com/nhuthuynh/pickleball-platform/issues/168#issuecomment-5302944092),
timestamp 15:36:39Z — one minute after T15.5's own review) wrote: *"Flagging
for T16 planning rather than resolving here"* — a **recommendation to look**,
not the **look already performed**, despite the review immediately above it
on the same page having just performed exactly that look in prose.

**Score, argued rather than asserted: T15's retro should have transcribed
T15.5's review into a drafted ticket shape (the specific methods, the
specific join key) rather than deferring that transcription to the next
ceremony**, because the marginal cost of doing so at retro time was
close to zero — the facts were already written, in the same session's own
context, minutes earlier. The actual cost of *not* doing it was not large in
this instance (T16's Ceremony 1 re-derived the same facts in about the same
amount of time, same day, same account) — so this is not a claim that T15's
retro caused a material delay. **It is a claim about a general practice this
project should adopt going forward**: when a blocked ticket's own PR or
review already enumerates the specific unblocking work in full technical
detail, the retro that reports the block should hand the next ceremony a
drafted ticket, not a pointer back to redo the enumeration. The Ceremony-1
dependency-completeness check exists precisely to prevent this class of
rediscovery when it *isn't* already known; when it *is* already known, from
the same sprint's own artifacts, re-deriving it a second time is exactly the
kind of redundant verification CLAUDE.md rule 10 asks for when a claim is
*un*verified — not when it has already been verified once, in writing, by
the same team, the same day.

## 7. ADR-0016's interim rule — nothing to score this sprint, and that is stated rather than left implicit

**PE.** T16 shipped three PRs (#197, #199, #200), each with exactly one
commit, each authored entirely by its own implementer — checked directly,
not assumed:

| PR | Commits | Reviewer-authored code? |
|---|---|---|
| #197 (T16.3) | 1, by the implementer | No |
| #199 (T16.2) | 1, by the implementer | No |
| #200 (T16.4) | 1, by the implementer (includes the pre-existing fake fix, found and fixed by the *implementer's own mandated toolchain run*, not by a reviewer editing someone else's branch — see finding 1) | No |

**Zero instances, second consecutive sprint.** Worth naming, per the
predecessor retro's own "score it explicitly" instruction for a mechanism
with only one prior data point — a second clean sprint is a slightly
stronger signal than the first, but two sprints is still not a large sample,
and D2 itself remains exactly as unanswered as before (§5). This finding
does not argue the interim rule can be retired; it argues the same thing
T15's retro argued about its own single data point — record it, don't
over-read it.

## No finding on

**No finding on the wave structure or dispatch order.** T16.2 and T16.3's
concurrent Wave-1 dispatch is exactly what produced finding 1's failure
mode, but the *dispatch decision itself* was correctly reasoned (no
in-sprint functional dependency existed between them, and none was
introduced) — the gap is in verification discipline at merge time, not in
the wave design. T16.4's Wave-2 sequencing on top of both correctly waited
for both Wave-1 tickets to land before branching.

**No finding on the label taxonomy.** #198 was filed pre-labelled... **not
quite** — checked directly: #198 carries **no labels at all** as of this
ceremony. This is worth naming rather than silently passing, though it does
not rise to a numbered finding on its own (a single missing label,
mechanical to fix, not a pattern): T17's Ceremony 1 should apply
`role:principal-engineer` (or `role:product-engineer`, matching #168's
established role for the same authorization-resolution work),
`type:bug`, and `context:payments` when it ranks the backlog, the same
sweep-time labelling correction #147/#149 both received in prior sprints.

**No finding on T16.2's or T16.4's own engineering correctness.** Both
tickets' headline mutation checks were independently re-verified by their
reviews (self-performed, matching the pattern every prior sprint has used)
and by this retro's own spot-checks above (exact method names, exact field
deletions, exact comment content) — no gap found beyond finding 1's
narrower, specific claim about one review's verification method.

---

## The sprint goal, scored: what was proven, what shipped exactly as claimed, and the one place a claim needed correcting

> *"Payments actually reads the admin stores it built last sprint...
> Alongside it: cancelling a Game or a Competition actually cancels its
> Registrations/Entries with it, closing the reachable half of a
> seven-sprint-old gap (#124)... a stale issue (#125) found during this
> ceremony's own backlog verification is corrected... D1/D2 go back to the
> user unanswered."*

**Every clause of the stated goal is met, verified independently, not taken
from any PR's own account.** #168 is closed and the code backs the claim
(exact methods, exact wiring, exact deleted fields, confirmed above). #149
narrows to its one remaining fact, confirmed against the code, not just the
comment. #124 takes its Registrations/Entries half with the court-Bookings
half correctly deferred and explained. #125 is corrected and closed by a
narrow, mechanical widening, confirmed by test names and code. D1/D2 remain
open, confirmed via the API.

**What the sprint's own plan did not anticipate, and this retro's central
finding**: neither #168's closure nor #124's partial fix came with any
warning that the shared branch's own tip would sit uncompilable between
them. This is not a failure of the stated goal — the goal said nothing
about branch integrity — but it is exactly the shape of gap T10's retro
first named and this retro re-confirms recurs: **an artifact (here, the
shared branch's own tip) can be simultaneously "the sprint's goal was met"
and "genuinely broken for a real span of wall-clock time," and only a
verification step that actually reconstructs the artifact under test — not
one that infers its state from a flag — catches the difference.**

**The agreed honest sentence, which T17's Ceremony 1 should carry into
`HANDOFF.md`'s Task-backlog entry** (`sprint-process.md` Ceremony 1 item 3
requires the retro's form, not a stronger one):

> T16 closed #168 (Payments resolves a Registration's Game and a
> CompetitionEntry's Competition, reads the real admin stores T15.5 built)
> and the corrected #125 (`RefundPayment` admits `competition_entry`), both
> via the mandatory close T16's own Ceremony 1 amended into the DoD — the
> first sprint this amendment has actually fired, 2/2, after T15 scored
> 0/2 on the identical shape. #124 takes its Registrations/Entries half
> (T16.3), leaving court-Bookings deferred for a stated, D1-shaped reason.
> **A real defect reached the shared branch's own tip: T16.2 and T16.3, both
> Wave 1 and correctly dispatched with no functional dependency between
> them, concurrently touched the blast radius of the same widened interface
> in disjoint files neither could see the other editing — T16.3 patched
> every implementer it could find; T16.2 authored a third one it couldn't
> know existed. The resulting break was real and `go vet`-visible on the
> shared branch's tip for 15m21s. It was not caught by T16.2's own review,
> whose stated verification method ("base already included T16.3,
> mergeable_state clean, no test-merge needed") did not describe what it
> actually tested — verified false from the commit graph itself, not
> inferred. It was caught, correctly and independently, by T16.4's review,
> which did what T16.2's review claimed to have done.** T16.2's own sibling
> sweep found a second, real caller-supplied-fact gap `CreateOnlinePayment`
> still has, filed as **#198**, mechanically small and unblocked. D1's
> unanswered status now visibly shapes ticket scope a second time — #124's
> court-Bookings half is deferred specifically because of it, not merely
> alongside it. D1 and D2 both remain unanswered by the user.

---

## Recommendations for T17's Ceremony 1 and 2

1. **When two same-wave tickets both touch the blast radius of a shared
   interface — one widening it, one authoring a new implementer of it — in
   disjoint files that produce no textual conflict, the second one to merge
   must verify against an actually reconstructed post-merge tree, not
   against `mergeable_state: clean`.** Concretely: `git fetch` the shared
   branch, then either merge it locally into the feature branch before
   running the toolchain, or check out GitHub's `refs/pull/<n>/merge` ref —
   the same thing T16.4's review already did correctly and T15.6's review
   (per T15's own finding 7) already did for a textual conflict. This is
   the fix for finding 1: a specific, narrow verification-method gap, not a
   dispatch or wave-design gap.
2. **Add #198 to T17's ranked backlog** (finding 4) — small, unblocked,
   mechanically shaped like T16.4, and should be labelled at ranking time
   (currently carries no labels at all).
3. **Escalate D1 to the user a fifth time, and say explicitly that its
   footprint has grown**, not just that it remains unanswered (finding 5) —
   two named instances of scope now shaped by its absence, not one.
4. **When a review or retro finds that an earlier artifact (a PR body, a
   review, an issue comment) already contains the full technical shape of
   an unblocking follow-up, transcribe it into the next ceremony's ticket
   text directly rather than recommending that the next ceremony
   re-derive it** (finding 6) — the marginal cost of transcription at the
   moment of discovery is near zero, and re-deriving already-known facts a
   second time is exactly the redundant-verification CLAUDE.md rule 10 is
   not asking for.
5. **Continue treating the merged-fix sweep as authoritative regardless of
   this retro's clean result** — per the standing rule, T17's Ceremony 1
   re-runs the full sweep and re-verifies #168's and #125's closes from the
   API rather than trusting this retro's table.

## Sprint-level Definition of Done — scored against what T16's own plan asked

Per `docs/process/t16-sprint-plan.md`'s "Sprint-level Definition of Done,"
four scorings were owed at this retro, stated there so they would not be
improvised:

- **(a) Did the "closes #N" mandatory-close amendment actually change the
  outcome for T16.2's and T16.4's own PR titles?** **Yes, on both.**
  Reported in full in finding 2(b): both closes happened within seconds of
  merge, neither deferred to a later sweep. First live test, 2/2, after
  T15's 0/2 on the identical shape.
- **(b) Did T16.2 actually close #168, and did it actually narrow #149 to
  its one remaining fact — scored against the merged code, not the PR's own
  account?** **Yes to both**, verified in finding 2(c) and the sweep
  (§3) against the actual struct fields and the actual issue state, not
  taken from either PR's prose.
- **(c) Did T16.3 actually ship both halves, and does #124 carry the
  required comment?** **Yes.** `CancelGame` and `CancelCompetition` both
  call their new bulk-cancel methods (verified by reading the code
  directly, finding 1's own investigation); #124 carries the required
  comment (verified via the API, §4 of the plan's own scoring, cross-checked
  in the sweep above).
- **(d) Was the new consumer-input clause exercised correctly for T16.2 —
  did the ticket's own dispatch verify `PayableID` was the join key before
  work began?** **Yes**, scored in finding 2(a): the plan's own §A10 named
  the exact field and the exact methods, and what was built matches
  exactly.

**Not scoreable by T16 and deliberately not pre-empted:** D1 and D2 remain
the user's (finding 5). **Newly owed at this retro and answered in full**:
whether the shared branch sat broken, for how long, and whether anything
could have caught it — finding 1, answered with the commit graph rather
than guessed.

**Not scoreable by T16, but investigated per this ceremony's own explicit
instruction rather than left to guesswork**: whether T16.3's review ran the
full toolchain across the whole repository or only against files it
touched. **It ran the full toolchain across the whole repository, and
correctly, against the tree that was the shared branch's actual tip at that
moment** (`665957a` → `d2ac53b`, no other T16 PR having merged yet) — T16.3
is not implicated in finding 1 at all, a conclusion this retro reaches by
reading the commit graph rather than assuming symmetry between the two
Wave-1 tickets.

Retro complete. Issue-tracker actions this ceremony: none — both closes
(#168, #125) and the one correction (#149) were already performed correctly
at review time this sprint, the first sprint this retro has had nothing left
to clean up. Open count at ceremony start: **11**. Open count now: **11**
(unchanged — nothing found here needed a live tracker action; #198's missing
labels are flagged for T17's ranking pass, not corrected here, since this
retro's own convention is to fix labels at ranking time, not retro time,
unless the label gap itself caused a tracked failure, which it did not).

Per `sprint-process.md`'s established convention (a retro PR never updates
the Docs-index row that points at it, since that row must cite this PR's own
merge number, which does not exist until it merges): **`HANDOFF.md`'s T16
row is corrected by this same PR anyway, per this task's own explicit
instruction** — this retro is correcting the *prior* ceremony's (T16
Ceremony 1's) placeholder "not yet written"/"not yet opened" text with real,
now-knowable values (this retro's own path, and T16's real PR merge order
verified against `merged_at`), which is the same thing every T`N+1`
Ceremony 1 has always done for row `N` — the only difference is this PR
performs that correction directly rather than waiting for T17's Ceremony 1,
because the instruction that produced this retro explicitly asked for it and
distinguished it from the disallowed case (a retro citing its own future PR
number, which this is not — T16's PRs are all already merged and their
numbers already known).
