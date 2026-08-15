# T17 Sprint Retro

Ceremony 3 per `docs/process/sprint-process.md`, six-role team (briefs:
`docs/agent-operating-handbook.md` Part B), held against
`docs/process/t17-sprint-plan.md` (§A0–§A10), `docs/process/t16-retro.md` as
the precedent and rigor bar, `HANDOFF.md`, and the real PR/issue history on
`nhuthuynh/white-label` (GitHub-side name `pickleball-platform`) — PRs
#202–#207, issues #124–#198.

Every timestamp, merge order, label, issue-state and file-overlap claim below
was pulled from GitHub's own `created_at`/`merged_at`/`submitted_at`/
`closed_at` fields, from direct `get_files` diffs on all five execution PRs,
and from direct reads/runs of the merged tree at `cc5fb54` — never inferred
from PR titles, review prose, or the sprint plan's forward-looking text
(CLAUDE.md rule 10).

**Verification performed before writing a single finding.** `git fetch` +
`git status` confirmed a clean worktree at the shared branch's tip (`cc5fb54`,
T17.4's own merge — matching `origin/claude/go-backend-pickleball-7up34j`
exactly, no fast-forward needed). `make generate && go build ./... && go vet
./... && make fmt-check && make vet-integration && make test-domain && make
test-adapters && make test-cmd && make test-platform && make gate-coverage`
were all run directly, not assumed from any PR's own account:

```
go build ./...                 # clean
go vet ./...                   # clean
make fmt-check                 # OK — gofmt clean
make vet-integration            # clean
make test-domain                # ok, all 12 packages
make test-adapters              # ok, all 22 packages
make test-cmd                   # ok
make test-platform              # ok
make gate-coverage              # OK — all 41 package(s) executed by ci-checks
```

Package count (41) is unchanged from T16's retro — expected, since all five
T17 tickets add test files to existing packages rather than new ones. Not
re-verified here (no Docker daemon reachable, same standing gap as every
prior sprint): the Docker-backed `make test`/`make ci-integration` path.

**Sprint outcome, stated before the findings that qualify it:** all 5 tickets
(17 points, T17.1–T17.5) merged in a single unbroken work block — plan-doc
merge (PR #202) at 17:38:05Z to final ticket merge (T17.4, PR #207) at
18:09:51Z, **31m46s wall-clock**, no session-limit interruption evidenced (one
PR, T17.1/#206, notes it was built in an isolated worktree specifically to
avoid touching four other sessions' concurrent uncommitted T17.x work in the
shared checkout — a precaution, not a recovery; the shared checkout is clean
now and none of that concern is live at retro time). Merge order: T17.2
(17:54:25Z) → T17.5 (17:57:57Z) → T17.3 (18:00:49Z) → T17.1 (18:03:32Z) →
T17.4 (18:09:51Z).

**What this retro found, in one sentence, so five findings do not have to be
read before the headline is known:** this was a clean execution sprint on
every axis this project has previously broken on — the merged-fix sweep
reconciles exactly, the harder of the two "closes #N" claims (a close that
depended on four separate PRs landing in the right order) was coordinated
correctly on its first live test, and the new same-wave shared-interface rule
correctly found nothing to do — but Ceremony 1's own ticket-writing carried
forward a wrong fact (which bounded context owns `discount_rules`) from an
earlier ceremony's artifact without checking it against the one file that
would have settled it in one read, and only implementation-time diligence
caught it before anything shipped wrong. Finding 4 is that finding, argued in
full rather than folded into a "no gaps found" summary.

---

## 1. The merged-fix issue sweep — clean, reconciled exactly, both closes
independently re-verified against the merged code, not the PRs' own account

**PO.** Per `sprint-process.md`'s DoD, this retro is sweep moment 1; T18's
Ceremony 1 remains the authoritative moment regardless of this result.

**Step 1 — list the open issues**, live, at this ceremony's start:
`list_issues(state: OPEN)` → **`totalCount: 9`**: #124, #126, #130, #134,
#144, #145, #149, #164, #167.

**Step 2 — reconcile arithmetically, before reading any issue individually.**
`open_at_end_of_T17's_own_Ceremony_1 − closed_during_T17 + opened_during_T17`.
T17's own Ceremony 1 (§A0 of `docs/process/t17-sprint-plan.md`) left the count
at **11**. During execution: **#198** closed (T17.1, PR #206) and **#195**
closed (T17.4, PR #207, on behalf of all four T17.2–T17.5 tickets) —
`11 − 2 = 9`. **Matches the live `totalCount: 9` read at this ceremony's
start exactly.** Zero issues opened this sprint — confirmed by re-reading
every PR body for a new `#N` reference; none of the five ticket PRs disclosed
a new gap requiring a fresh issue (T17.1's own instruction-6 sibling sweep
explicitly re-checked and found none; T17.2/T17.3's own sibling notes found
none either).

**Step 3 — cross-reference merged PRs against the open list, checking
attribution, not just the count.**

| Issue | `state` (re-fetched now) | `state_reason` | `closed_at` | Closed by / cites |
|---|---|---|---|---|
| #198 | `closed` | `completed` | 2026-08-15T18:03:43Z | Comment: *"Closed by PR #206 (T17.1)"* |
| #195 | `closed` | `completed` | 2026-08-15T18:10:00Z | Comment: cites all four resolving PRs — #203 (T17.2), #205 (T17.3), #204 (T17.5), #207 (T17.4) |

Both closes verified as genuine full resolutions, not partial fixes wrongly
titled, against the **merged code directly**, not the closing comments'
prose:

- **#198**: `internal/payments/app/service.go`'s `authorizeOnlineCreation`
  (read at the current tip) delegates to `s.authorizeCompetitionEntryRecording`
  for `PayableTypeCompetitionEntry`, exactly as the closing comment claims.
  `CreateOnlinePaymentInput`'s struct (also read at the current tip) carries
  only `PayableType`/`PayableID`/`Amount`/`ActorUserID` — `EntrantPlayerID`
  and `AssignedCompetitionAdminUserIDs` are gone, not merely unread.
- **#195**: all nine FK paths #195's own table named are mapped, confirmed by
  reading each context's `translateErr`-shaped function directly in the
  merged diffs (finding 2 below lists all nine with their sentinel and test
  coverage). No row is missing, no row is double-counted.

**Both closes happened at the per-PR moment (DoD step 5's optional early
close, made mandatory this project's `sprint-process.md` for "closes #N"
titles), not via a later sweep** — continuing T16's clean result for a third
consecutive sprint on this specific axis (T15: 0/2; T16: 2/2; T17: 2/2). The
#195 case is a materially harder instance of the same mechanism than
anything scored before it — see finding 2.

**Sweep result: clean, third sprint running.** Zero unclosed hits, zero
mis-attributed closes, arithmetic reconciles exactly. **T18's Ceremony 1
still re-runs this sweep in full**, per the standing rule that a prior
ceremony's clean result does not discharge the next one.

## 2. The coordinated, cross-PR "closes #N" for #195 — a harder version of
the mechanism T16 first proved, correctly executed on its own first live test

**QA.** T16's clean 2/2 result (finding 2b of `docs/process/t16-retro.md`)
tested the mechanism in its simplest shape: one PR, one ticket, one close, at
the moment that PR merged. **#195 is a structurally different case**: its fix
is spread across four independent PRs (T17.2, T17.3, T17.4, T17.5), no two of
which functionally depend on each other, and the issue can only be honestly
closed once **all four** have landed — closing it from any single PR would
misdescribe the codebase for however long the remaining PRs take to merge,
exactly as T17.5's own ticket instruction 7 warned.

**Checked directly against every review's own text, not assumed from the
final closing comment:**

| PR | Merge order | Review's own statement about #195 |
|---|---|---|
| T17.2 (#203) | 1st | *"does not close #195 — correctly deferred to the coordinated close across all four FK tickets"* |
| T17.5 (#204) | 2nd | *"does not close #195 — correctly deferred to the coordinated close across all four FK tickets"* |
| T17.3 (#205) | 3rd | *"does not close #195 — correctly deferred to the coordinated close across all four FK tickets"* |
| T17.4 (#207) | 4th (last) | *"Merging, and will close #195 with a comment naming all four resolving PRs"* — **then did**, at 18:10:00Z, 9 seconds after PR #207 merged (18:09:51Z) |

**Every one of the first three correctly recognized it was not the last to
land and said so explicitly; the fourth correctly recognized it was and
executed the close exactly as it promised**, naming all four PR numbers in
the closing comment — verified directly against the comment body, not
inferred from its existence. This is precisely the discipline T15's own PR
#192 established as the working pattern (*"will close #147 per instruction
5," then did*) and T16 retro's own recommendation 2 asked future "closes #N"
reviews to follow — now proven to generalize from a single-PR close to a
four-PR coordinated one, which is a harder case because three separate
reviews each had to correctly decline the same action for the same reason,
not just one review correctly performing it.

**Nothing about this was structurally forced to work.** Any one of the first
three reviews could have closed #195 early (mis-describing the remaining
three FK paths as fixed), or the fourth could have merged without noticing it
was last and left #195 open for the sweep to catch. Neither happened. **Score:
held, on the first sprint this exact multi-PR shape existed to test it
against.**

## 3. The same-wave shared-interface verification rule (§A4, T17's own
amendment) — zero opportunities, confirmed independently by diffing every
PR's actual file set, not by trusting the plan's own §A10 claim

**PE.** T17's own Ceremony 1 (§A10) predicted no same-wave shared-interface
overlap existed across the five tickets and that the rule would therefore not
fire. **Independently re-checked against each PR's actual `get_files` diff**,
not the plan's prose:

| Ticket | Files touched |
|---|---|
| T17.1 (#206) | `internal/payments/**`, `proto/pickleball/payments/v1/payments.proto` |
| T17.2 (#203) | `internal/socialplay/adapter/postgres/**` |
| T17.3 (#205) | `internal/competitions/adapter/{postgres,grpcapi}/**` |
| T17.4 (#207) | `internal/facilities/adapter/{postgres,grpcapi}/**`, `internal/booking/adapter/postgres/discount_*.go` |
| T17.5 (#204) | `internal/booking/adapter/{postgres,grpcapi}/recurring_hire_*.go`, `internal/booking/domain/errors.go` |

**Genuinely disjoint, confirmed at the file level, with one pair worth naming
explicitly rather than waved through:** T17.4 and T17.5 both touch
`internal/booking/adapter/postgres/`, but on different files
(`discount_repository.go`/`discount_foreign_key_test.go` vs.
`recurring_hire_repository.go`/`recurring_hire_foreign_key_test.go`/
`recurring_hire_foreign_key_test.go`) — both reviews (PR #204's and PR #207's)
independently noticed this and independently test-merged the shared tip to
confirm no conflict, textual or semantic, existed. No ticket widens an
interface any other ticket implements; every FK translation is a local
`switch`/`if` addition inside that context's own existing `translateErr`-
shaped function, never a new interface method. **The rule correctly found
nothing to do, verified rather than assumed** — the same "silence is
indistinguishable from not having checked" standard `t17-sprint-plan.md` §A8
holds itself to.

**Worth naming as a positive, not just a null result: every one of the five
reviews performed a reconstructed-merge-tree verification (`git fetch` + a
real local merge into a fresh worktree) anyway**, even for the pairs with no
possible interface overlap (T17.2 explicitly noted its own base already
matched the shared tip with nothing to reconstruct; T17.1, T17.3, T17.4, and
T17.5 each found their own base stale by the time they ran and test-merged
before verifying). The discipline §A4 mandated for the narrow shared-interface
case has generalized into this sprint's standing review practice regardless
of whether that narrow case applies — which is a healthier outcome than a
rule that only fires exactly where it is required and is silently skipped
everywhere else.

## 4. Ceremony 1's own ticket text carried forward a wrong bounded-context
assignment for `discount_rules` — caught by implementation-time diligence,
not by planning-time verification, and not the repeat it was described as

**BA (the claim, checked before accepting it).** T17.4's PR (#207) and its
review both describe the correction as *"mirroring PR #191/T15.6 §1's
identical re-verification."* **Checked directly against PR #191's own §1**
(fetched in full for this retro, not taken from the analogy's own framing):
T15.6's §1 is titled *"The defect, re-verified rather than carried forward"*
and re-confirms four **facts about the bug itself** (the guard's shape, the
FK's existence, the missing translation arm, the `%s`-vs-`%w` stripping) —
it says nothing about which bounded context owns the table in question,
because `bookings.court_id` was never in dispute; `bookings` has always been
Booking's own table. **The phrase "identical re-verification" describes the
right general discipline (re-check the tree, don't trust inherited text —
CLAUDE.md rule 10) but the specific shape of this sprint's mistake — a
ticket naming the wrong bounded context for a table entirely — has no prior
instance in this project's history that this retro could find** (checked:
`docs/process/t9-retro.md`'s and `t9-sprint-plan.md`'s two other "wrong
context" hits are a payables-routing decision and a Player-level-field design
debate, neither about which context's adapter directory a table's write code
belongs in). **This is the first occurrence of this specific shape, not a
repeat** — worth stating plainly rather than letting a PR's own analogy stand
unchecked, since a genuinely novel gap and a recurring one call for different
recommendations.

**QA (tracing where the wrong fact actually entered the record).** #195's
own table — filed at **T16's** Ceremony 1 — lists a `Context` column and
assigns `discount_rules.facility_id` to **facilities**. T15.6's own PR #191,
the artifact #195 drew its rows from, never asserted a context for this row
at all (its §7 sibling sweep grouped rows only by "guarding read," no context
column existed yet). **The wrong assignment was introduced at #195's own
filing, one sprint before this one**, then correctly transcribed —
faithfully, but faithfully wrong — into T17.4's own ticket text by this
sprint's Ceremony 1 (§A5's table), per the transcription clause §A4 itself
just formalized this same ceremony. The fact was carried through two
ceremonies' worth of "trust the prior artifact" before anyone checked it
against the one document that actually settles bounded-context ownership for
a Postgres table: the migration file itself.

**The migration file settles it in its first line, unambiguously, since
T11 — five sprints before #195 was ever filed:**

```
-- DiscountRules for the Booking context (T11.2, on top of T11.1's
-- internal/booking/domain.DiscountRule — see docs/process/t11-sprint-plan.md
-- T11.2). ...
```

`db/migrations/0017_booking_discount_rules.sql`'s own filename even encodes
it (`booking_discount_rules`, not `facilities_discount_rules`). T17.4's own
implementer found this by checking (per instruction 2's "check, don't
assume") and its reviewer independently re-confirmed with `find
internal/facilities -iname "*discount*"` returning empty — **caught before
merge, zero shipped harm**: the fix landed correctly in
`internal/booking/adapter/postgres/discount_repository.go`, not
`internal/facilities/adapter/postgres/repository.go` as the ticket's own
literal instruction 1 said.

**PdE (was this avoidable at planning time, and how cheaply).** Yes, and
cheaply — this is the same shape as T15's own finding 1 (a check that
verifies the wrong half of a dependency) but one level earlier: the
dependency-completeness check's own standing rule is *"run against the code
… never against a ticket's or an issue's own prose description of what
exists"* (`sprint-process.md`, "The dependency-completeness check"). That
rule already exists and already covers exactly this case in spirit — Ceremony
1's §A5 table transcribed #195's own prose (`Context: facilities`) rather
than checking the code (or, more cheaply still, the migration file's own
header comment — one line, no ambiguity) before assigning T17.4's target file
path. This is not a new class of gap; it is the existing dependency-
completeness discipline not being applied to a fact (bounded-context
ownership) that the check's own text does not currently name as something to
verify.

**Score, argued rather than asserted: a real, narrow, cheaply-avoidable
planning-time miss — caught by execution-time diligence rather than by the
planning-time check that should have caught it, with zero shipped
consequence.** Not a repeat of an identical prior incident (checked, per
above); a genuinely new instance of the general "trust the artifact, not the
code" failure mode this project has now seen in several different shapes
(T15.5's producer/consumer gap, #97's misattribution, #149's stale
prediction, and now this). Recommendation 1 below is the concrete, narrow
fix — not a new ceremony step, one added checkpoint to a check that already
exists.

## 5. D1 and D2 — D1's footprint neither grew nor shrank this sprint (a
contrast worth naming against T16, where it grew); D2 has a third
consecutive sprint with nothing to score

**Re-verified this ceremony, not assumed.** `issue_read(get_comments)` on
#144 (D1): still exactly **one** comment, T14.3's original escalation,
unchanged across T14 through T17. `docs/adr/0016-*.md`'s (D2) own `## Status`
field: still **"Escalated — awaiting the user's decision. This ADR decides
nothing."**

**PE.** T16's retro (finding 5) documented D1's footprint *growing* — a
second, independent piece of shipped scope (the court-Bookings half of
T16.3's cascade) built around D1's absence, not merely a repeated mention of
it. **Checked whether the same happened again this sprint**: it did not.
`internal/payments/app/service.go`'s `BookingHostID` field and its direct
comparison (`if in.BookingHostID == "" || in.ActorUserID != in.BookingHostID`)
are present and unchanged at the current tip — confirmed by reading the file
directly, not assumed from #149's own unmoved state. T17.1 touched
`authorizeOnlineCreation`'s `PayableTypeCompetitionEntry` branch only; its own
instruction 6 sibling sweep (re-verified at implementation time, not just
planning time — PR #206's own body states it explicitly) confirmed
`BookingHostID` is still the only caller-supplied fact left anywhere in
Payments, unchanged, still blocked on D1. **D1's footprint is exactly the
same size it was at T16 retro's end — neither grown (unlike T16) nor shrunk
(nothing in this sprint could touch it, correctly, since no T17 ticket
implements `CancelBooking` authorization).** Worth naming as a genuine, if
modest, contrast to T16's trajectory rather than silently repeating "still
unanswered" without checking whether the cost kept compounding.

**D2** has no PR this sprint carrying reviewer-authored code to score against
the interim rule — every one of the five T17 PRs' commits were authored
entirely by their own implementer (checked directly against each PR's commit
list, not assumed): a third consecutive sprint (after T15 and T16) with
nothing to report either way. Per this project's own stated caution against
over-reading a small sample, this is recorded as a null result, not evidence
the interim rule can be retired.

**Neither D1 nor D2 is implemented, decided, or guessed at by this ceremony.**
Both remain exactly as blocked as `sprint-process.md`'s own restriction lists
require. Per the task's own instruction, neither is re-escalated with new
text here — T17's Ceremony 1 already carried D1's fifth deferral forward
(§A7 of `docs/process/t17-sprint-plan.md`); this retro's job is to check
whether the sprint that followed changed the picture, not to re-escalate on
its own initiative.

## No finding on

**No finding on wave structure or dispatch.** All five tickets were
correctly dispatched as one disjoint Wave 1 with no Wave-1.5 checkpoint,
exactly as `t17-sprint-plan.md`'s own waves section stated, and finding 3
confirms the disjointness was real, not merely claimed.

**No finding on the label taxonomy.** #198's labels (`role:product-engineer`,
`type:bug`, `context:payments`) were applied at T17's own Ceremony 1 (§A2) and
verified unchanged now; #195 already carried a conformant six-label set
(`role:product-engineer`, `type:bug`, four `context:*` values) from T16's
Ceremony 1. No new issue was opened this sprint to check.

**No finding on the transcription clause's own first live test, beyond
finding 4's narrower point.** §A4's transcription clause (adopted this same
sprint, per T16 retro recommendation 4) was applied to both T17.1 (transcribed
#198's own fully-specified body verbatim — method names, deletion targets,
proto fields — and every one of those was implemented exactly as
transcribed, confirmed against the merged code) and T17.2–T17.5 (transcribed
#195's own table of nine rows, eight of which were correct). The clause did
its job — it saved re-derivation of already-known facts — and finding 4 is
the honest caveat on the one row where the artifact being transcribed was
itself wrong, not a reason to say the clause failed generally.

**No finding on any T17 ticket's own engineering correctness beyond what
findings 1–3 already establish.** Every headline mutation check claimed by a
PR was independently re-performed by that PR's own review (self-performed,
matching every prior sprint's pattern) and this retro's own spot-checks
(finding 1's direct reads of `service.go` and each context's
`translateErr`) corroborate the claims rather than merely repeat them.

---

## The sprint goal, scored: what was proven, what shipped exactly as claimed

> *"Two disclosed, unblocked gaps get closed for real … `CreateOnlinePayment`
> stops trusting a caller's own claim about who a competition entrant or
> admin is … (#198). And the FK-race defect class … gets its full nine-path,
> four-context translation … (#195) … D1 and D2 go back to the user
> unanswered, D1's growing footprint stated plainly rather than only
> re-asserted."*

**Every clause of the stated goal is met, verified independently, not taken
from any PR's own account.** #198 is closed and the merged code backs the
claim exactly (finding 1). #195 is closed, all nine FK paths mapped with
integration-test coverage compiled (`make vet-integration` green), and the
close correctly waited for all four resolving PRs (finding 2). D1 and D2
remain open, confirmed via the API (finding 5) — and, worth stating since the
goal text specifically flagged D1's footprint rather than just its status,
this sprint's own #149 check confirms that footprint did not grow further
this time, a genuine (if modest) improvement in trajectory over T16.

**What the plan did not anticipate, and this retro's own central finding**:
the plan's own §A5 table, in ranking and scoping #195's four tickets, silently
carried forward a wrong bounded-context assignment for one of its nine rows
— not from this sprint's own error, but inherited from #195's own filing a
sprint earlier and never checked against the migration file that would have
settled it in one read. It caused no shipped defect (finding 4), but it is
the same shape of gap this project has now named several times in different
forms: an artifact's prose trusted instead of the code (or, here, the
migration) it describes.

**The agreed honest sentence, which T18's Ceremony 1 should carry into
`HANDOFF.md`'s Task-backlog entry** (`sprint-process.md` Ceremony 1 item 3
requires the retro's form, not a stronger one):

> T17 closed both #198 (`CreateOnlinePayment` now resolves its
> competition-entry facts through the same resolver ports T16.2 built,
> rather than trusting the caller) and #195 (all nine FK-race write paths
> across four bounded contexts now translate their `23503` into a clean
> domain error instead of an unclassified `Internal`), both via the
> mandatory "closes #N" mechanism — the harder of the two, #195, required
> three PRs to correctly decline the close and a fourth to perform it exactly
> as promised, and all four did. The merged-fix sweep reconciles exactly
> (`11 − 2 + 0 = 9`). The new same-wave shared-interface verification rule
> found nothing to do — all five tickets were genuinely file-disjoint,
> confirmed by diffing every PR rather than trusting the plan's own claim —
> and every review performed the reconstructed-merge-tree check anyway as
> standing practice. **One real, narrow planning-time gap was found**:
> Ceremony 1's own ticket text, for T17.4, named the wrong bounded context
> (`facilities` instead of `booking`) for `discount_rules.facility_id`,
> carried forward unchecked from #195's own filing a sprint earlier; the
> migration file's own header comment has said "for the Booking context"
> since T11, and a single read of it at planning time would have caught
> this before the ticket was ever dispatched. The implementer and reviewer
> both caught and correctly fixed it before merge — zero shipped harm — but
> the planning-time check that should have caught it first did not.
> D1's footprint held steady rather than growing further this sprint
> (contrast T16); D2 had a third consecutive sprint with no reviewer-authored
> gap-fix to score. Both remain unanswered by the user.

---

## Recommendations for T18's Ceremony 1 and 2

1. **When a ticket's scope names a specific database table, Ceremony 1
   verifies that table's owning bounded context against its migration
   file's own header comment (or, absent one, its filename/directory)
   before finalizing the ticket's target file path — not only before the
   ticket merges** (finding 4). This is a narrow addition to the existing
   dependency-completeness check's standing rule ("run against the code …
   never against a ticket's or an issue's own prose description of what
   exists"), not a new ceremony step: the same discipline already applies to
   read/write capability, extended here to apply to context ownership too,
   since this sprint showed that the same "trust the prior artifact" failure
   mode can attach to a fact the check's current wording doesn't explicitly
   name.
2. **Continue treating the merged-fix sweep as authoritative regardless of
   this retro's clean result** — T18's Ceremony 1 re-runs the sweep and
   re-verifies both #198's and #195's closes and attributions from the API
   rather than trusting this retro's table (finding 1).
3. **When a fix spans multiple independent PRs that together close one
   issue, require every non-last PR's review to state explicitly that it is
   deferring the close and why, and the intended-last PR's review to state
   its intent to close before merging** — this sprint is the first time this
   exact shape was tested (finding 2) and it held on every one of the four
   PRs involved; naming it as the expected pattern going forward (rather
   than leaving it to be independently reasoned out again next time a
   multi-PR issue closure comes up) is the cheap way to keep it holding.
4. **D1 and D2 stay with the user.** No T18 ticket should implement
   `CancelBooking` authorization or a reviewer-authorship carve-out; neither
   should be guessed at. If either answer arrives mid-sprint, each
   escalation's own trigger takes over.

## Sprint-level Definition of Done — scored against what T17's own plan asked

Per `docs/process/t17-sprint-plan.md`'s "Sprint-level Definition of Done,"
four scorings were owed at this retro, stated there so they would not be
improvised:

- **(a) Did T17.1 actually close #198, scored directly against the merged
  code?** **Yes.** `authorizeOnlineCreation` delegates to
  `authorizeCompetitionEntryRecording` for competition-entry payables, and
  `CreateOnlinePaymentInput` no longer carries the caller-supplied
  entrant/admin fields — both confirmed by reading the current tip directly
  (finding 1), not the PR's own account.
- **(b) Did all four of T17.2–T17.5 actually ship, and was #195 closed by
  exactly one of them citing all four PRs?** **Yes to both.** All nine FK
  paths are mapped with an exhaustiveness-table entry apiece and an
  integration test compiled per path (`make vet-integration` green); #195
  was closed by exactly one PR (T17.4/#207), citing all four resolving PRs
  by number, with the other three correctly declining (finding 2).
- **(c) Did the new same-wave shared-interface verification rule get
  exercised this sprint at all?** **No — zero opportunities, confirmed by
  diffing every PR's actual file set rather than trusting the plan's own
  §A10 claim** (finding 3). Not a finding against the rule, per the plan's
  own scoring instruction: a sprint with no genuine shared-interface overlap
  has nothing for the rule to do.
- **(d) Was the transcription clause actually followed for T17.1 and
  T17.2–T17.5?** **Yes, and followed faithfully enough to also faithfully
  transcribe one wrong fact.** T17.1's instructions matched #198's own body
  exactly, and every element was implemented as transcribed. T17.2–T17.5's
  instructions matched #195's own table for eight of nine rows correctly;
  the ninth (`discount_rules.facility_id`'s context) was transcribed
  accurately from #195's own text but that text was itself wrong — caught at
  implementation time, not planning time (finding 4).

**Not scoreable by T17 and deliberately not pre-empted:** D1 and D2 remain
the user's (finding 5).

Retro complete. Issue-tracker actions this ceremony: none — both closes
(#198, #195) and their correct attribution were already performed correctly
during the sprint itself, the second sprint running (after T16) with nothing
left for the retro to clean up on the closure axis. Open count at ceremony
start: **9**. Open count now: **9** (unchanged — the one real finding this
ceremony made, finding 4, caused no live-tracker action since it was already
caught and fixed before merge; no new issue is warranted for a
planning-time process gap that shipped no defect, distinguishing it from
#185's/#195's own origin as a *shipped, disclosed* residual).

Per `sprint-process.md`'s established convention (a retro PR never updates
the Docs-index row that points at it, since that row must cite this PR's own
merge number, which does not exist until it merges): **`HANDOFF.md`'s T17
row is not touched by this PR.** Unlike T16's retro (which corrected its own
sprint's row under an explicitly-argued one-off instruction), this ceremony
follows the ordinary convention — T18's Ceremony 1 corrects T17's row,
including its real PR merge order and the honest-form sentence above, as its
first job, per the standing rule and per `t17-sprint-plan.md`'s own DoD item
5, which already states this explicitly rather than leaving it to be
inferred.
