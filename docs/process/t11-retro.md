# T11 Sprint Retro

Ceremony 3 per `docs/process/sprint-process.md`, six-role team (briefs:
`docs/agent-operating-handbook.md` Part B), held against
`docs/process/t11-sprint-plan.md` (including its A5 migration
pre-assignment, A6/A8/A9/A10 predictions and A11 recorded
disagreements), `HANDOFF.md`'s T11 entries, and the real PR/issue history
on `nhuthuynh/white-label` (GitHub-side name `pickleball-platform`) —
PRs #112–#121, issue #111.

Every timestamp, merge order, and claim below was pulled from GitHub's
own `created_at`/`merged_at`/`submitted_at` fields and from re-reading
each PR's actual commits, bodies, and review objects — not assumed from
titles. Claims that could be checked against the working tree were
checked: `make test-domain` was re-run in this environment (green, all
twelve domain+app packages), the migration directory was listed rather
than trusted, `internal/booking/port/` and `internal/socialplay/port/`
were listed directly to test two separate plan claims, and
`web/src/test-support/semanticControlAssertions.ts` was read in full
rather than summarized from its PR body.

This retro's own six assigned investigation points are not taken as
given either. Finding 1 **refines** the premise it was handed (the
failure was not only that two agents became unreachable — the
coordinating session stayed active for another 43–48 minutes afterward
and ended the work block without noticing two of five dispatched Wave-1
tickets had produced no PR). Finding 2 **partly corrects** its premise
(A10 did flag T11.1/T11.4 as a same-package watch item, so the collision
was not wholly unanticipated — but it reasoned about the wrong files).
Finding 5 **adds** an error in the sprint plan that nobody was asked to
look for and that the ticket's own reviewer saw and then mis-filed as a
match.

Findings are not a single voice. Recorded disagreements are left as
disagreements per the "do not manufacture consensus" rule, matching T9's
and T10's retros.

**Sprint outcome:** all 9 tickets (47 points, T11.1–T11.9) merged across
PRs #113–#121, plus the Ceremony 1/2 doc PR (#112). Sprint goal met in
full: a Facility Owner can configure a real discount and a Player sees an
honestly-labeled discounted quote (T11.1/T11.2/T11.3); a Club can request
a recurring slot and an Owner can approve it, generating real
`recurring_hire`-source Bookings under the existing no-double-booking
invariant, with per-occurrence conflicts reported rather than aborting
the approval (T11.4/T11.5/T11.6); shipped screens got a real WCAG 2.2 AA
pass (T11.7); and the three T10-retro process fixes were threaded into
execution, with the fixture-ID generalization finding and fixing a real,
previously-undetected vacuous test in `competitions` (T11.9). Work
happened in two blocks: 2026-08-13T14:32:59Z–15:29:10Z (PRs #112–#115,
56 minutes) and 2026-08-14T14:04:40Z–15:42:50Z (PRs #116–#121, 1 hour
38 minutes). No PR merged in a broken state, and no defect reached the
shared branch this sprint — the one real break (finding 2) was caught in
review and fixed on the source branch before merge. The review record is
the strongest in this project's history: every one of the nine PRs
carries a discoverable GitHub review object, and four of them
(#116, #119, #120, #121) record independent re-verification from a
**fresh worktree off the pushed branch** — the heavier half of T10 retro
finding 1's disagreement, adopted in practice by the reviewer without
ever being made mandatory.

---

## 1. Two Wave-1 implementer sessions finished correct work and ended before opening a PR — and the work block ended anyway, with nothing checking the dispatch list for completeness

**PE (the mechanism, from primary sources).** Two of Wave 1's five
tickets produced no PR from their own session. Both PRs were opened the
next day by the coordinating session, whose own review bodies state the
cause plainly:

- PR #116 (T11.4): *"This PR's implementer session ended before pushing/
  opening the PR (the branch had a finished, correct commit sitting
  locally). I picked up the mechanical remainder: pushed the branch,
  opened this PR, and reviewed it."*
- PR #117 (T11.9): *"This PR's implementer session ended before opening
  the PR (the branch had a finished, correct commit already pushed to
  origin). I picked up the remainder: opened this PR and reviewed it."*

The two failed at **different points in the same sequence** — T11.9 got
as far as pushing its branch and stopped before opening the PR; T11.4
stopped one step earlier, with the commit only ever existing locally.
That difference matters for detection: a branch-listing check would have
found T11.9's work and still missed T11.4's.

**QA (the timings, computed from commit-author dates and PR
`created_at`, not estimated).**

| Ticket | Final implementer commit | PR opened | Gap |
|---|---|---|---|
| T11.4 (#116) | `af972ba` 2026-08-13T14:40:51Z | 2026-08-14T14:04:40Z | **23h 23m 49s** |
| T11.9 (#117) | `9a1ba84` 2026-08-13T14:45:36Z | 2026-08-14T14:06:09Z | **23h 20m 33s** |

Against the same wave's siblings, which opened their own PRs:

| Ticket | Final commit | PR opened | Gap |
|---|---|---|---|
| T11.8 (#113) | `337cc2a` 14:46:59Z | 14:47:34Z | 35s |
| T11.1 (#114) | `6c3e63f` 14:47:42Z | 14:48:38Z | 56s |

**The finding this table produces is not the one the sprint's framing
expected.** Both silent tickets finished their engineering work *before*
both tickets that succeeded — T11.4 at 14:40:51 and T11.9 at 14:45:36,
versus T11.8 at 14:46:59 and T11.1 at 14:47:42. So this was not slow work
running out of budget mid-task. Both agents completed correct,
independently-verifiable work early and then failed at, or never reached,
the purely mechanical publish step. Root cause beyond that is **not
determinable from repository evidence** — session/turn exhaustion and an
interruption leave identical traces here, and no agent-runtime log
survives in the repo. This retro declines to guess between them, per its
own standing "checked, not assumed" rule; what *is* determinable is
everything below, which is where the actionable part lives.

**PdE (the part that is genuinely a process failure, and is nobody's
agent-runtime problem).** The coordinating session did not stop when the
agents did. It merged PR #113 at 14:48:48Z, PR #114 at 14:50:08Z, opened
PR #115 at 14:54:18Z and merged it at 15:29:10Z — **48m 19s of continued
activity after T11.4's last commit, 43m 34s after T11.9's** — and then
ended the work block with two of five dispatched Wave-1 tickets having
produced no PR, no branch (in T11.4's case), and no notification. Nothing
in that 48 minutes compared the set of tickets dispatched into Wave 1
against the set of PRs that existed. The 23-hour gap is therefore not
23 hours of "waiting for a notification that never came" — it is
~45 minutes of a live session that could have caught it trivially,
followed by an overnight pause.

**QA (what it cost, stated honestly).** No product cost: both tickets'
work was correct, and the reviewer independently verified both
(domain-purity `go list` check on T11.4, `uuidShape` regex inspection and
full-suite run on T11.9) before merging. The real costs were (a) the
coordinating session absorbed unplanned verification and publish work it
had not scoped, and (b) had the session *not* resumed the next day, two
tickets' finished work would have been invisible — one of them existing
only as an unpushed local commit in a worktree, i.e. one cleanup away
from being lost entirely. That second cost is the one worth designing
against.

**Recorded disagreement — PE vs. PdE on the mitigation.**
- **PE:** the coordinating session should periodically poll long-running
  background agents rather than relying on completion notifications, since
  this sprint proves a notification may simply never arrive.
- **PdE:** polling is the wrong shape and more expensive than the problem.
  Polling a session that has already ended returns nothing useful (which
  is exactly what the coordinator observed the next day — `ListAgents`
  empty, `TaskOutput` "no task found"), so it detects the failure no
  earlier than a much cheaper check would. The cheap check is a
  **roll-call before ending a work block**: enumerate the tickets
  dispatched in each open wave, list the PRs and remote branches that
  exist, and name any ticket with neither. That is one comparison against
  a list the sprint plan already contains (A10's wave table), costs
  seconds, and — critically — would have caught T11.4, whose branch was
  never pushed and which no branch-listing or agent-polling check could
  see at all.
- **Unresolved.** Both agree the roll-call is worth adopting and is the
  cheaper of the two; whether periodic polling *additionally* earns its
  cost is not settled.

**Change for T12, adopted (the uncontested part):** before a work block
ends, the coordinating session performs an explicit **wave roll-call** —
for every ticket dispatched and not yet merged, state whether a PR
exists, a remote branch exists, or neither, and name any ticket in the
"neither" state as an open item rather than ending the block silently.
A dispatched ticket with no PR is a finding, not an absence of news.

## 2. `//go:build integration` makes a file invisible to every gate a session can actually run — and the same file broke twice in the same sprint

**QA (the facts, from the PRs' own verification lists).** T11.5's PR
(#120) added two parameters to `bookingapp.NewService` and did not update
`internal/booking/adapter/postgres/concurrency_integration_test.go`'s
call site. The file is gated behind `//go:build integration`, so
`go build ./...`, `go vet ./...`, `make test-domain` and
`go test ./... -race` all pass while it is broken. The break is visible
in the implementer's own commit message, which lists its verification as
*"make generate; go build ./...; go vet ./...", "make test-domain",
"go test ./... -race -count=1", golangci-lint, gofmt* — every one of
which is blind to the file, and **no `-tags=integration` variant**.

This is not a novel mistake. **T11.2's PR (#118), one PR earlier in the
same sprint, in the same file, explicitly ran
`go vet -tags=integration ./...`** (it is named in that commit's own
verification list) and proactively updated the same call site when it
added *its* two parameters. The reviewer caught T11.5's break in a fresh
worktree, fixed it on the source branch as commit `fdb0ff5`, and named
the recurrence precisely: *"This is the exact break class T11.2's own PR
explicitly caught and fixed in this same file; T11.5 reintroduced it."*
Broken state existed on the PR branch from 15:01:13Z to 15:08:39Z
(**7m 26s**) and **never reached the shared branch**.

**PE (the root cause, which is not "an implementer forgot").** The
question posed to this retro was whether `go vet -tags=integration` is
now reliably in this codebase's working memory. Within the sprint,
demonstrably yes: T11.6's PR (#121), authored after the fix, ran it
*"this last one specifically"* and cites T11.5's mistake by name as the
reason; the reviewer's own #121 review opens by singling it out *"given
T11.5 broke it in this same sprint."* Across sprints, there is no reason
to expect it to survive, because **nothing durable encodes it.** Checked
directly against the `Makefile`:

- `make test-domain` → `./internal/.../domain/... ./internal/.../app/...`
  only. Never compiles the file.
- `make ci` → `generate tidy lint test-domain test-tools generate-client
  lint-web test-web build-web` plus `go build ./...`. **No
  `-tags=integration` step anywhere.**
- `make test` → the only target that compiles integration-tagged code
  (`-tags=integration`), and it *runs* the testcontainers suite, so
  `make ci-integration` hard-guards it behind `docker info` and exits 1
  without a daemon.

No session in this project's history has had a Docker daemon — T11.2,
T11.5 and T11.6 each independently disclose this. So the only gate that
compiles this file cannot run here, and the file is structurally
unverifiable by every command an agent *can* run. T11.2 and T11.6 caught
it by each independently remembering an ad-hoc command that appears in no
target, no checklist, and no doc. That is a convention held in three PR
bodies, which is precisely the shape T10 retro finding 2 already
condemned for fixture IDs ("a shared constant block is still just
convention, not enforcement").

**The fix is cheap and Docker-free, which is what makes this worth
adopting rather than debating.** `go vet -tags=integration ./...`
type-checks the tagged files without executing anything and without a
Docker daemon — it would have failed on T11.5's commit in this very
environment. It belongs in the Makefile, on the Docker-free side of the
split.

**Change for T12, adopted:** add a `vet-integration` target —

```make
# Type-checks the //go:build integration files that every other gate is
# blind to. No Docker needed: vet compiles, it does not run.
vet-integration:
	go vet -tags=integration ./...
```

— and add it to `make ci`'s gating list (after `test-domain`), so the
check runs by default instead of depending on an implementer recalling an
undocumented command. `make ci`'s closing message should stop implying
integration-tagged code is entirely uncovered locally: it is now
*compiled* by `make ci`, just not *executed* until `make ci-integration`.

**Recorded disagreement — QA vs. PE on how much more to add.**
- **QA:** the Makefile target is necessary but not sufficient. This bug
  class has now appeared twice in one sprint, and a `make ci` that
  requires `make generate` (buf + sqlc) is not something every ticket
  runs. Add a CLAUDE.md gotcha line naming the trap and a line in the
  per-ticket DoD, so an implementer who never runs full `make ci` still
  sees it.
- **PE:** CLAUDE.md's gotcha section already documents the
  `-tags=integration` file's existence, and a second prose reminder is
  documentation that rots while pretending to be enforcement — the exact
  failure this finding is about. One executable gate is worth more than
  three prose reminders; if the concern is that implementers skip
  `make ci`, fix that, don't duplicate the rule into files nobody
  re-reads mid-ticket.
- **Unresolved.** The `vet-integration` target and its inclusion in
  `make ci` are adopted either way; the CLAUDE.md/DoD additions are not
  settled.

## 3. A10's dispatch-isolation check reasoned about the files each ticket would *add* and missed the existing file both would *append to*

**PdE (what A10 actually said, since the premise needs correcting
before it can be assessed).** A10 did not miss the T11.1/T11.4 pairing —
it named it: *"both new files in `internal/booking/domain`, disjoint
filenames — `discount_rule.go` vs. `recurring_hire_template.go` —
checked against each other's expected file list, no overlap expected but
flagged as the same package as a minor watch item."* So the tickets were
correctly identified as adjacent, and a watch item was raised. The
prediction still failed in a specific, diagnosable way: **the check
compared the files each ticket would create.** The collision was in
`internal/booking/domain/errors.go` — a file *neither* ticket's expected
file list mentions, because neither ticket creates it; both append a
sentinel block to its existing `var` block. PR #116 came back
`mergeable_state: dirty` on first open, and its review records the real,
hand-authored resolution: *"a real conflict in
`internal/booking/domain/errors.go` (both T11.1's `DiscountRule`
sentinels and this ticket's `RecurringHireTemplate` sentinels append to
the same var block). Resolved on this branch... kept both blocks, T11.1's
first, T11.4's second — no semantic change to either ticket's error set."*

**PE (the same class, third occurrence).** This is structurally identical
to T10 retro finding 4, which named it for migration numbers: two tickets
can pass a file-overlap check and still collide over a resource that
neither ticket's own diff makes visible to the other. The class now has
three demonstrated instances across three sprints — `0005_payments.sql`
vs. `0005_socialplay.sql` (T6), `0015` claimed twice (T10), and
`errors.go` (T11) — and the adopted fix each time has covered only the
instance in front of it. A5's migration pre-assignment worked perfectly
this sprint (finding 5), which is exactly why the gap is now visible in a
new shape: the fix was to the *instance*, not to the *question A10 asks*.

**Worth crediting, because it limited the damage.** T11.4's implementer
anticipated the risk that A10's watch item pointed at and defended
against the half it could see alone: its commit message records
*"errors.go: additive only — appended new sentinels after the existing
block, did not reformat/reorder T11.1's or any other ticket's entries"*
and it namespaced its variant type (`RecurringHireEndCondition`) so the
two tickets' identically-shaped `EndCondition` types could not collide as
Go identifiers in the same package. Both were correct calls that made the
conflict a clean two-block keep rather than a semantic merge. An
implementer cannot pre-coordinate a shared append target it cannot see;
that is planning's job.

**Change for T12, adopted:** A10's dispatch-isolation table gains a
second question alongside its new-file comparison — *for each pair of
tickets in the same or adjacent waves, which **existing** files will both
append to?* — with `internal/<context>/domain/errors.go` named
explicitly, since it is the shared append target of essentially every
domain ticket this project writes and has now caused one hand-resolved
conflict. Where both tickets will append to the same file, the plan states
the merge-order rule up front (later-merging ticket resolves on its own
source branch, both blocks kept) rather than leaving it to be discovered
at `mergeable_state: dirty`.

## 4. The disclosed-gap → next-ticket handoff worked cleanly, and bypassed the plan's own A3 rule requiring it to be tracked

**PO (what happened, verified in both PR bodies).** T11.5's PR (#120)
disclosed a real design gap in its own "decisions stated rather than left
implicit" section: *"`ListRecurringHireTemplatesForFacility` is
owner-only... Consequence: a Club cannot yet read back its own templates,
which T11.6's Club-facing status view will need."* The reviewer decided
not to block the merge and instead briefed T11.6's implementer, recording
that decision in the #120 review: *"I'll brief the T11.6 implementer to
build the actor-scoped read as part of that ticket, since T11.6's own
instructions already require a 'status view reading back the actor's own
templates'."* T11.6's PR (#121) opens by naming the pickup: *"closes the
backend gap T11.5's own PR flagged for this ticket... This PR is
therefore Go + Vue, unlike the sprint's other UI tickets."* It shipped
`ListRecurringHireTemplatesForActor` with a new port method, sqlc query,
proto RPC and REST route, reusing the `recurring_hire_templates_requested_by_idx`
index that migration `0018` had already added in anticipation (verified
directly — the index is at line 108 of `0018_booking_recurring_hire_templates.sql`,
and PR #121 adds no migration).

The engineering outcome is genuinely good. T11.6's version is also
better-reasoned than a mechanical gap-fill: it records a **checked
negative finding** for why it deliberately does *not* add a `club`-role
check on the new read (the actor id is itself the scope; a role check
would remove no exposure while locking a Club that lost the role out of
its own rejection history), and pins that absence with a test asserting
the identity port is never consulted.

**BA (the process gap, which is real despite the good outcome).** T11's
own plan, A3, states the standing rule: *"any disclosed-but-out-of-scope
regression must produce a tracked follow-up (an issue, at minimum a
`HANDOFF.md` bullet) at the moment it's found, not left for a future
reviewer to rediscover by chance."* Checked: **no GitHub issue was opened
for this gap** (no issue exists in the T11 range at all — finding 6), and
**no `HANDOFF.md` bullet records it** (`HANDOFF.md` was last modified by
the sprint-plan PR #112 on 2026-08-13, before T11.5 existed). The gap
lived in exactly two places — PR #120's body and the coordinating
session's working memory — for the 33 minutes between #120's merge
(15:09:35Z) and #121's opening (15:38:17Z).

**QA (why "it worked" is not sufficient evidence that it is a good
pattern).** It worked under conditions that made failure nearly
impossible: the dependent ticket was the *very next* thing dispatched, by
the *same* session that read the disclosure, within half an hour, and
that ticket's own written instructions already independently required the
capability the gap blocked. Change any one of those — the sprint ends
first, T11.6 slips to T12, a different session picks it up, or T11.6's
instructions had not happened to require a status view — and the only
surviving record is a paragraph in a merged PR body, which is precisely
the "rediscovered by chance" failure mode A3 exists to prevent. This is
the same lesson T9 retro finding 5 and T10 retro finding 2 already
reached from different directions.

**Recommendation for T12, adopted:** the in-flight briefing is worth
keeping — it is fast and it demonstrably produced a better result than a
cold ticket would have — but it is a **dispatch mechanism, not a tracking
mechanism**, and A3 requires both. When a reviewer decides not to block a
merge on a disclosed gap, the same review that records the decision must
also create the durable record: a GitHub issue where one is warranted, or
a `HANDOFF.md` Cross-cutting bullet at minimum, naming the PR that
disclosed it and the ticket expected to absorb it. The briefing then
points at the tracked item instead of substituting for it.

## 5. The plan's predictions, audited: A5 is this project's first process fix to fully prevent its own incident class; A9 held; A6 was exceeded in a way that exposed A6's own gap; and A8 carried one false citation

**PE (A5 — confirmed, and it is a genuine first).** A5 pre-assigned
`0017` to T11.2 and `0018` to T11.5 rather than leaving each session to
derive the next number from the directory. Verified by listing
`db/migrations/`: `0017_booking_discount_rules.sql` and
`0018_booking_recurring_hire_templates.sql` both exist, exactly as
assigned, and both implementers' commit messages independently confirm
they took the number from the plan rather than the directory (T11.2:
*"number pre-assigned by the sprint plan's A5 namespace check, not
derived from the directory"*; T11.5: the same phrasing). The directory
still carries the T6 scar (`0005_payments.sql` **and**
`0005_socialplay.sql`) as a permanent reminder of what this prevents.
Two tickets in the same sprint chain each added a migration and **no
collision occurred, for the first time** — T6 collided, T10 collided.
Credit where due: this is a T10-retro finding that became a planning
mechanism and then measurably worked, and it is the cleanest example this
project has of a retro feeding the next sprint's plan.

**PE (A9 — both rulings held, unreopened).** Verified structurally:
`internal/` contains `booking, competitions, facilities, identity,
payments, platform, socialplay` — **no `internal/pricing`**, so Ruling 1
held. `internal/booking/domain/` contains `discount.go` and
`recurring_hire_template.go` alongside `pricing.go` and `booking.go`, so
both new models landed where A9 placed them, and Ruling 2 held. Neither
ticket's PR reopens either question; T11.5 raises an adjacent one on its
own initiative (`app.NewService` reaching seven parameters, versus
`competitions/app`'s `ServiceOptions` struct) and explicitly declines to
bundle it — *"a mechanical, separable cleanup deliberately not bundled
here"* — which is the correct handling of a real observation that is not
this ticket's job. **A10's related instruction also held**: `internal/booking/port/`
contains exactly one `facility_lookup.go`, and T11.5 *extended* it with
`CourtIDsForFacility` rather than deriving a second port, which is what
A10 explicitly warned against.

**Designer (A6 — T11.6 did something meaningfully better than A6 asked,
and in doing so revealed that A6's own prescription was insufficient).**
A6 required T11.6 to check the rental-request control's absence across
semantic signals *"reusing `findGenderControls()`'s shape... as the
reference pattern, not a screen-local reimplementation that only checks
one signal type."* Read literally, that is satisfied by a second helper
that copies the same four signals — one implementation per concept, free
to drift. T11.6 instead extracted the body into a new pattern-parameterized
`findControlsMatching(wrapper, pattern)` in
`web/src/test-support/semanticControlAssertions.ts` and made
`findGenderControls` a one-line delegation to it (verified by reading
both files: `genderControlAssertions.ts` line 38 is
`return findControlsMatching(wrapper, GENDER_PATTERN)`). One scan, two
callers, drift structurally impossible. That is a better outcome than the
instruction's literal reading, and the reviewer mutation-checked it
(forcing the gate open fails exactly three tests).

**The sharper half:** A6 named four signals — `aria-label`, `<label>`
association, ARIA `role`, and native `id`/`name`. **All four would have
missed T11.6's own control.** The Club rental request control is a plain
`<button>Request a recurring rental</button>`, which carries no `id`, no
`name`, no `aria-label` and no `<label>` — the helper's own header
records this: *"a gender control is a *chooser* while a rental-request
control is an *action*."* T11.6 added a fifth signal (accessible text of
a command control — native `<button>`/`<summary>`/`<a href>` and the
ARIA `button`/`link`/`menuitem`/`tab` roles), a strict superset that
leaves all four existing gender specs green. Had T11.6 implemented A6
exactly as written, its absence assertion would have been **vacuous for
the very control it was written to guard** — the test would have passed
whether or not the gate worked. This is the same trap T10 retro finding 5
named ("absence assertion checks syntax, not semantics") recurring one
level up: the *remediation* for that finding was itself specified against
one control shape and generalized to a different one without rechecking.

**QA (A8 — accurate on the load-bearing claim, false on one citation,
and the error was seen and then mis-filed).** A8's table said T11.5's new
`internal/booking/port.IdentityLookup` *"mirrors T10.4's
`socialplay/port.IdentityLookup` pattern exactly."* **No such port has
ever existed.** Verified two ways: `internal/socialplay/port/` contains
`court_reservation.go`, `facility_lookup.go`, `idgenerator.go`,
`registration_payment_updater.go`, `repository.go`, `waitlist_repository.go`
— no identity lookup; and `git log --all -S "IdentityLookup" --
internal/socialplay/` returns **zero commits**, so the string has never
appeared in that context's history at all. T11.5's implementer caught
this independently and disclosed it in its commit: *"the ticket cites a
socialplay/port.IdentityLookup precedent from T10.4. No such port exists
in this repo — Booking is genuinely the first context to call into
Identity — so the shape follows the precedent that does exist
(FacilityLookup)."*

Two things make this worth a finding rather than a footnote. First,
**what was wrong is exactly what was unmarked.** A8's cells carry
explicit evidence markers where the ceremony actually checked —
*"confirmed by inspection this ceremony"*, *"confirmed by grep this
ceremony"*, *"confirmed `club` is a real enum value, not assumed"* — and
the load-bearing claims all hold (Identity's `GetUser`/`Roles` and the
`club` value do exist; Booking→Facilities and Booking→Identity really
were both new). The single claim with no marker is the single claim that
was false. Second, **the error was verified and then not recorded as an
error**: the #120 reviewer independently grepped for it and wrote
*"Confirmed the disclosed `IdentityLookup` precedent deviation is
accurate: grepped the repo myself, no `socialplay/port.IdentityLookup`
exists — Booking genuinely is first into Identity, **matching the sprint
plan's A8 table**."* The grep was right, but the conclusion conflates
A8's true half with its false half — A8 does not "match," it contains a
citation that this grep disproves. A8's table is the artifact T9 retro
finding 5 created to stop implementers discovering cross-context gaps
mid-ticket; a fabricated precedent inside it sends an implementer looking
for a pattern to mirror that does not exist, which is a cheaper version of
the same cost.

**Change for T12, adopted:** every cell of the cross-context dependency
table carries its evidence marker (what was checked, how) — and any
precedent citation that cannot be pointed at a real path in the repo is
**dropped rather than asserted**. A plan may say "no precedent exists,
follow X"; it may not name a file it has not confirmed. When a reviewer
verifies that an implementer's disclosed deviation from the plan is
correct, the plan is wrong and the review should say so in those words,
so the error is legible to the retro rather than absorbed as a match.

**PdE (A10's wave predictions — both correct, one by hedging).** A10
predicted T11.3 (Vue) and T11.5 (Go) were *"genuinely parallel"* with no
file overlap; confirmed by the #119 review, which verified *"the diff is
`web/`-only (`git diff` against base shows zero non-`web/` files)"*. A10
also pre-flagged a possible T11.3/T11.6 collision on `router/index.ts`
and checkout components *"if timing overlaps despite the wave gap."* It
did not fire — the wave gap held (T11.3 merged 14:58:43Z; T11.6 branched
from `e7c8d32`, which already contained it, and needed no merge commit).
A hedged prediction that correctly identifies both the risk and the
condition under which it would fire, and then does not fire because the
condition did not hold, is the system working; noted so that "Wave 4 ran
clean" is recorded as *checked* rather than assumed.

## 6. T11 fixed issue-closing hygiene for every past sprint and opened zero issues for its own nine tickets — the board of record now contains no evidence T11 happened

**PO (verified directly against the GitHub API).** T11.8 did its job
well. All 42 previously-open issues were closed with
`state_reason: completed` and a comment naming the implementing PR, each
cross-checked against `HANDOFF.md` and the actual PR before closing; the
two non-obvious attributions (#17/#18 landing via PR #27's three-way
merge, #22 completed by T8.5's PR #60) are documented in #113's body; and
`sprint-process.md`'s Ceremony 1 bullet and DoD step 5 were corrected to
state that closing is always manual on this branch topology. `#111` was
deliberately left open for the merging party and is now closed.
`list_issues(state: OPEN)` returns **`totalCount: 0`**.

And: `list_issues(state: CLOSED)` returns 44 issues, the newest of which
is **#111 itself** (created 2026-08-13T14:25:15Z). **There is no GitHub
issue for T11.1 through T11.9.** The board of record is, for the first
time, perfectly clean and perfectly uninformative — it contains no record
that a 47-point, 9-ticket sprint occurred.

**BA (why this is a finding and not pedantry).** `sprint-process.md`
names GitHub Issues + labels as the **board of record**, and its
Ceremony 1 spec says *"the ticket **is** the GitHub issue."* Its whole
label taxonomy (`sprint:t11`, `role:*`, `type:*`, `points:*`) is defined
against issues that, for T11, do not exist. T11's tickets exist only as
sections of `docs/process/t11-sprint-plan.md`. So the process document
and reality now disagree in the opposite direction from T10's finding 6:
that one was "issues exist but never close," this one is "there is
nothing to close."

**The sharp part is the timing.** T11.8's own PR body — written
2026-08-13, mid-sprint — correctly disclosed this exact gap **for T10**:
*"T10.1–T10.8 were never opened as individual GitHub issues at all... no
new GitHub issues were opened for the already-merged T10.1–T10.5 work,
since doing so retroactively... is out of this ticket's scope... Flagging
for a future ceremony to decide."* That disclosure is exemplary — named,
scoped, deferred with reasoning, not silently dropped. Nobody connected
it to the sprint in flight. At the moment it was written, T11's own nine
tickets were in exactly the same state, and remained so through both work
blocks. **The ticket auditing issue hygiene documented the historical
instance of the gap while standing inside a live one.** That is the
generalizable lesson: a disclosure about the past is not a check on the
present, and a retroactive-cleanup ticket is the single best-placed
moment to ask whether the thing being cleaned up is still happening.

**Recorded disagreement — PM vs. BA on which side to fix.**
- **BA:** open the nine T11 issues' worth of process going forward — at
  Ceremony 1, each T12 ticket gets a real GitHub issue with its labels
  before dispatch, restoring the board of record the process document
  actually specifies. Without it, the labels, points and role taxonomy are
  fiction, and there is no queryable record of what any sprint contained.
- **PM:** two consecutive sprints have now shipped successfully with the
  sprint-plan doc as the real ticket record, and nothing was lost — the
  plan is richer than an issue body, version-controlled, reviewed, and
  linked from `HANDOFF.md`'s docs index. If issues add no value that the
  plan doesn't already carry, the honest fix is to amend
  `sprint-process.md` to say the sprint plan is the board of record and
  issues are for cross-sprint follow-ups only (which is, empirically,
  exactly what #96/#97/#98/#111 were used for and where issues *did*
  work). Maintaining a parallel record nobody reads is process theater.
- **Unresolved.** Both agree the current state — a process document
  specifying a board of record that the last two sprints did not use — is
  not acceptable; whether T12 fixes the practice or fixes the document is
  not settled, and this retro does not have standing to pick.

---

## Recommendations for T12's Ceremony 1 and 2

Concrete and mechanical, in the spirit of T10 retro finding 2 → T11.9 and
T10 retro finding 4 → T11's A5:

1. **Add `vet-integration` to the Makefile and to `make ci`** (finding 2)
   — `go vet -tags=integration ./...`, Docker-free, positioned after
   `test-domain`. This is the highest-value item here: it converts a
   convention held in three PR bodies into a gate, and it is the second
   time this exact break has been introduced in one sprint.
2. **Wave roll-call before ending a work block** (finding 1) — for every
   dispatched, unmerged ticket, state whether a PR exists, a branch
   exists, or neither. A ticket in the "neither" state is an open item,
   not silence.
3. **Extend the dispatch-isolation table to shared *existing* append
   targets** (finding 3), naming `internal/<context>/domain/errors.go`
   explicitly, and state the merge-order resolution rule for any such pair
   up front. Migration numbers are already handled by A5; this is the same
   question asked of files rather than sequences.
4. **A disclosed gap that a reviewer decides not to block on must produce
   a durable record in the same review** (finding 4) — an issue, or a
   `HANDOFF.md` Cross-cutting bullet at minimum. Briefing the next ticket
   is a good dispatch mechanism and is not a substitute for tracking.
5. **Evidence-mark every cell of the cross-context dependency table, and
   drop precedent citations that cannot be pointed at a real path**
   (finding 5). When a reviewer confirms an implementer's disclosed
   deviation from the plan, record it as a plan error in those words.
6. **Decide the board-of-record question explicitly** (finding 6): either
   T12's Ceremony 1 opens a real GitHub issue per ticket before dispatch,
   or `sprint-process.md` is amended to name the sprint plan as the board
   of record with issues reserved for cross-sprint follow-ups. Either is
   defensible; the current mismatch is not, and it has now silently held
   for two sprints.
7. **When specifying a test-helper reuse, specify the *property*, not the
   reference implementation** (finding 5, Designer half). A6 named four
   signals and the control it was written for matched none of them. The
   durable phrasing is "the check must find the control under every
   identification shape a reasonable implementation could use, and must be
   mutation-checked against the actual control this ticket ships" — which
   is what T11.6 did, and what A6's literal text would not have produced.
8. **Carry forward, unresolved:** T10 retro finding 1's PE/QA
   disagreement on whether fresh-worktree verification is mandatory is
   now informed by real evidence — the reviewer applied it voluntarily on
   four of nine PRs this sprint (#116, #119, #120, #121) and it is what
   caught finding 2's break. T12's Ceremony 1 should revisit it with that
   evidence rather than re-recording the same open question a third time.
   Also still open: A11's QA/PE disagreement on a lint-rule enforcement
   for fixture IDs, which T11.9 deliberately left out of scope — and which
   finding 2's `vet-integration` precedent now gives a cheap template for
   (a gate, not a convention), so it is worth re-asking whether the
   enforcement half is still too expensive.

---

## No finding on

**No finding on the fixture-ID work itself (T11.9), which exceeded its
brief.** The ticket was scoped as a mechanical generalization of PR
#107's constant block and instead found a real, previously-undetected
vacuous test: `competitions`' `authz_regression_test.go` and
`sharelink_test.go` both used `CompetitionId: "no-such-competition"`,
which `competitions/app`'s `uuidShape` guard short-circuits *before*
reaching the repository — so both tests were exercising the guard rather
than the repository-miss path their own names claim to prove, with an
identical outward `NotFound` either way. The reviewer independently
confirmed the regex analysis rather than accepting it. Five of six
contexts were verified clean and each stated as a checked-clean negative
per the ticket's instruction. This is the T10-retro→T11-ticket pipeline
working exactly as intended, and the second such success this sprint
after A5.

**No finding on WCAG (T11.7) or the pricing UI (T11.3) beyond what their
reviews record.** Both merged with substantive independent verification
(#119's review re-ran `make generate` + `npm run generate:client`, read
the generated `v1EndCondition` type directly to confirm the
no-fabricated-fields claim, and read `CourtBookingFlow.vue`'s discount
markup rather than trusting the PR's quoted snippet). Neither produced a
process-level finding, and a retro that manufactures one to fill a slot
is the mirror image of the "zero findings is suspicious" rule.

**No finding on dispatch isolation itself** (T9 retro finding 1's
subject) — every ticket ran in its own worktree per A10, and no
shared-checkout collision recurred. Finding 3's `errors.go` conflict is
the shared-*namespace* class, not the shared-*working-directory* class,
and is filed as its own finding accordingly.

**PM had limited independent material this sprint**, as in T9 and T10.
A11's PM/PdE disagreement (whether a 4-deep wave chain was too much
sequencing risk) can now be scored, and PdE's concern did not materialize
as a scheduling failure: the chain T11.1→T11.2→T11.5→T11.6 executed in
order across two blocks, and the serialization cost nothing measurable.
The chain did, however, produce finding 4's handoff and finding 2's
recurrence — both artifacts of tickets landing close together in a
dependency line — so the risk was real but showed up as coordination
load rather than schedule slip. PM's substantive contribution to this
retro is embedded in finding 6's board-of-record disagreement.
