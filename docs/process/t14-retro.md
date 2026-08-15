# T14 Sprint Retro

Ceremony 3 per `docs/process/sprint-process.md` (read in its **T14-amended**
form, since T14.6's own PR #175 added the per-PR closure DoD step, the
merged-fix issue sweep and the decided label taxonomy that this ceremony is
governed by and must score), six-role team (briefs:
`docs/agent-operating-handbook.md` Part B), held against
`docs/process/t14-sprint-plan.md` (including its A0–A16 appendix),
`docs/process/t13-retro.md` as the precedent and rigor bar, `HANDOFF.md`, and
the real PR/issue history on `nhuthuynh/white-label` (GitHub-side name
`pickleball-platform`) — PRs #174–#183, issues #123–#168.

Every timestamp, merge order, label and file claim below was pulled from
GitHub's own `created_at`/`merged_at`/`submitted_at`/`closed_at` fields and
from `git`/`go`/`make` run against the merged branch at `14a2f0d` — not
inferred from titles, and **not taken from the coordinating session's own
account of the sprint**, which for two of the nine tickets is also the account
of its own authorship (finding 2). Claims checkable against the working tree
were checked here, and the toolchain in this environment is complete
(`go` 1.25.0, `buf`, `sqlc` all present), so unlike T13's ceremony this one
could run the codegen-dependent half itself:

- `make generate` run, then `make fmt-check`, `make test-domain`,
  `make test-platform`, `make test-tools`, `make test-adapters` (22 packages
  `ok`), `make test-cmd`, `make vet-integration` — **all green**.
- `make gate-coverage` run: `41 package(s) hold test functions … OK — all 41
  package(s) with runnable tests are executed by "ci-checks"`.
- **The gate-coverage check was mutation-checked by this ceremony
  independently of the PR that shipped it and independently of its own test
  suite** (CLAUDE.md rule 10): a synthetic `func Test` was added to
  `internal/booking/port`, a package outside all five gated patterns.
  `make gate-coverage` failed, named the package, exited non-zero; the file
  was removed and it returned to `OK — all 41`. The check is real.
- Side A of the check re-derived by hand with the same command T14's Ceremony 1
  used (`grep -rl "func Test" --include="*_test.go" internal tools cmd |
  xargs -n1 dirname | sort -u | wc -l` → **41**), so the tool's own count was
  not taken on trust.
- The issue list was read from the live API (`state: OPEN` → **`totalCount:
  13`**), every close's `closed_at` and comment read individually, and every
  one of the nine PRs' review objects fetched and read in full.

**This retro's assigned investigation points are not taken as given.**

- **Finding 1 declines to score the issue-closure sweep as either a pass or a
  repeat of T13.** It is neither. The act happened, correctly, with citations —
  and it happened at a moment the amendment does not name, performed by the one
  party the amendment's design was written to avoid relying on. The honest
  answer is that T14 discovered a **third state** T14.6's own text does not
  describe, and the recommendation is to name it rather than to declare
  victory or defeat.
- **Finding 2 is not on the list of things a normal retro checks, because it
  has never happened before.** Two implementer sessions were terminated by an
  account-level limit mid-run and the coordinating session authored the
  recovery for both. Two of nine PRs are therefore *self-authored and
  self-reviewed*, and the review-elapsed-time record shows it: 9 and 8 seconds
  from PR open to review submission, against a sprint median of 92 seconds.
- **Finding 3 reopens a question T13's retro closed**, on the trigger T13's
  retro itself specified — and the news is good, which is why it is worth
  recording precisely.
- **Finding 4 credits T14.1 and then names which of its three proofs is the
  weak one.** The sprint's own siblings exercised the property in its easy form
  only; the hard form was proven by mutation, twice, once by this ceremony.
- **Finding 5 is small, and it is the same shape as T13's finding 1 in
  miniature**: a review that states a follow-up will be done, and a PR that
  states an issue's state, with neither checked.

Findings are not a single voice. Recorded disagreements are left as
disagreements per the "do not manufacture consensus" rule, matching T9–T13.

**Sprint outcome:** all 9 tickets (39 points, T14.1–T14.9) merged across PRs
#175–#183, plus the Ceremony 1/2 doc (#174). Six issues closed (#131, #156,
#157, #158, #160, #165), zero opened, taking the open-issue count from **19 to
13**. T14 is the first sprint since the board-of-record split (T12's Ceremony
1) to open **zero** issues — T13 opened nine (one at Ceremony 1, eight in
execution), and T12's retro counted nineteen — and finding 5 records which
part of that zero is a *checked* zero and which
part is merely a silence. No ticket took a second review loop and no defect
reached the shared branch. The sprint ran from plan-merge 06:55:10Z to final
merge 13:48:47Z — **6h53m37s wall-clock**, which is not comparable to T13's
1h03m26s, because it contains a **5h40m41s window (07:35:27Z → 13:16:08Z) in
which no PR was opened or merged**, spanning the account-level session-limit
interruption that finding 2 is about. Discounting that window, the engineering
occupied roughly 1h08m, in line with T13.

The engineering is strong and the verification record is again the best
available: every one of the nine PRs carries a discoverable GitHub review
object; every review records an independent test-merge or fresh-worktree
run, a reviewer-performed full toolchain pass, and a **reviewer-performed
mutation check**. PR #180's review went further than any review in this
project's history — it stood up a real Postgres 16, applied all 19 (now 20)
migrations, ran the real server binary, minted a token from the new dev
fixture and drove a real RS256-verified request end-to-end to the database.

**What did not happen is again administrative, and again it is finding 1** —
but the failure is one notch smaller than T13's, and saying only "it happened
this time" would be the wrong record.

---

## 1. The six closes happened, correctly and with citations — in an 11-second batch, by the merging party, 26 minutes to 6h45m after the PRs that earned them. The per-PR half of T14.6's own new rule was followed 0/6, and the sweep's two sanctioned moments were both bypassed

**QA (the measurement, from `closed_at` on each issue and `merged_at` on each
PR).** All six closes carry `state: closed`, `state_reason: completed`, and a
comment naming the resolving PR — read individually, not inferred:

| Issue | Closed by PR | PR `merged_at` | Issue `closed_at` | Gap |
|---|---|---|---|---|
| #165 | #177 (T14.2) | 07:04:19Z | 13:49:44Z | **6h45m25s** |
| #156 | #178 (T14.8) | 07:11:48Z | 13:49:50Z | **6h38m02s** |
| #158 | #179 (T14.7) | 07:19:49Z | 13:49:47Z | **6h29m58s** |
| #131 | #179 (T14.7) | 07:19:49Z | 13:49:49Z | **6h30m00s** |
| #160 | #180 (T14.9) | 13:16:08Z | 13:49:53Z | 33m45s |
| #157 | #181 (T14.1) | 13:23:26Z | 13:49:42Z | 26m16s |

Every one of the six `closed_at` values falls inside **13:49:42Z–13:49:53Z —
an eleven-second window**, 65 seconds after the sprint's final ticket (#183)
merged at 13:48:47Z, and immediately before this ceremony was dispatched.

**Arithmetic, run the way the amendment specifies (step 2, before reading any
issue individually):** `19 − 6 + 0 = 13`. `list_issues(state: OPEN)` returns
`totalCount: 13`. Exact. **The sweep's own step 3 — cross-referencing every
`#N` in every merged PR against the open list — produces zero hits for a full
resolution, so as a sweep, this retro's run is clean.** That is stated with
its count per the amendment's "a sweep whose output is silence is
indistinguishable from a sweep never run" rule.

**PO (why "clean sweep" is not the finding).** The retro's sweep is clean
because the merging party ran it 65 seconds before dispatching the retro. That
is a real improvement over T13 — the API calls were made, the citations are
accurate, and nobody had to discover the failure a sprint later. It is also
**not** what any of the three written moments say:

- **DoD step 5, the per-PR half (T14.6's own item 1):** *"the reviewer performs
  the close before moving to the next ticket."* Followed **0 times out of 6**.
  Every close post-dates not just its own PR but *every other ticket in the
  sprint*. The two Wave-1 tickets whose closes waited 6h30m+ had five
  subsequent PRs opened, reviewed and merged in between.
- **Sweep moment 1 — the retro runs and reports it.** This retro found nothing
  to close, because there was nothing left.
- **Sweep moment 2 — the next sprint's Ceremony 1, "*where the 'party other
  than the merger' property actually lives*."** Not reached; nothing will be
  left for T15's Ceremony 1 to find either.

So the property PO argued for in T13's retro — *"checkable in one API call by
**someone other than the merger**"* — **was never obtained.** The merger
checked its own work, on the same day, in the same session. The bookkeeping is
correct; the independence is not.

**PE (the dissent, and it is strengthened rather than weakened by this).** PE's
recorded position (A1, and `sprint-process.md`'s "Recorded disagreement") was
that the per-PR half must be primary because otherwise *"the issue list
misdescribes the codebase for the entire gap between sprints."* Measured: the
issue list misdescribed the codebase for **6h45m** in #165's case — inside the
sprint, not between sprints. PE's cost is real and was paid, in the exact
currency PE named. But PE's *remedy* — make the per-PR half primary — is the
half that scored 0/6, for the second consecutive sprint. PE is right about the
cost and wrong about which mechanism will collect it.

**PM/BA (what actually made the difference, which neither role predicted).**
T13 scored 0/9 on the act. T14 scored 6/6 on the act and 0/6 on the moment.
The variable that changed is not the reviewer's attention at merge time — that
is exactly what did *not* change. What changed is that the sprint acquired an
**end-of-sprint checklist item with a name** (`the merged-fix issue sweep`)
that the merging session executed as a discrete task once the tickets stopped
arriving. The mechanism that worked is the sprint-level one, run early; the
mechanism that failed is the per-PR one, again.

**Score: PO is right, and the scoring condition the plan wrote is satisfied in
substance even though its literal wording is not.** `sprint-process.md`'s
scoring condition reads: *"If every ticket that closes an issue follows DoD
step 5 and the sweep therefore finds nothing, PE is right… If the sweep fires
again, PO is right."* Read literally, neither branch fires: DoD step 5 was not
followed, and the sweep found nothing. That is a **third state the condition
does not enumerate** — *the merger swept its own work at sprint end* — and
pretending it is branch one would be manufacturing a PE win out of a
bookkeeping outcome PE's mechanism did not produce.

**The honest classification, adopted:** T14's closure record is
**compliant-in-outcome, non-compliant-in-moment, and unverified-by-anyone-
else.** It is a genuine improvement on T13 and it is not the property the
amendment was written to deliver.

**The sweep's second disposition was skipped entirely, and that half is a
clean miss.** The amendment's three dispositions include: *"The PR was a
partial fix → leave it open and **write down why**, naming the successor issue
or ticket that will close it… **it is only correct once someone has written the
sentence**."* Checked against the live issue record:

| Issue | Status | Comments added this sprint |
|---|---|---|
| #144 | correctly open, escalated | **1** — a substantive, same-minute comment (07:01:03Z, at T14.3's merge) recording ADR-0015, the four options, the re-verified facts, a correction to the record, and "this is the second deferral" |
| #147 | correctly open, half-fixed by T14.5 | **0** |
| #168 | correctly open, half-fixed by T14.4/T14.5 | **0** |
| #149 | correctly open, materially advanced | **0** |

**#144 — the one issue nobody was required to comment on, because it was
escalated rather than partially fixed — is the only one that got the
sentence.** The three issues the disposition explicitly names got nothing.
Their "why" exists in `docs/process/t14-sprint-plan.md` §A7 and in PR #182's
and #183's bodies, which is better than nothing and is not where a reader of
the issue list looks. This is the same asymmetry as finding 1's main half:
the judgement-laden work was done well and recorded in the artifact its author
was already writing; the mechanical step onto the durable record was skipped.

**This finding does not warrant a new `docs/LESSONS.md` incident entry.** T13's
entry (`## T13 (2026-08-15) — nine PRs said "closes #N"…`) already carries the
mechanism, and this sprint is a partial recovery from it, not a fresh
incident. It is recorded here and indexed by the retro stub.

## 2. Two of nine tickets were authored by the coordinating session recovering interrupted agents' work, and reviewed by the same session — the two shortest open-to-merge windows in the sprint by an order of magnitude. The engineering survived independent re-derivation; the review's independence did not exist

**QA (what happened, from the PRs' own disclosures, which are honest and
unprompted).** PR #181 (T14.1) and PR #182 (T14.4) both open with a
**"Note on provenance"** paragraph stating that the implementer session hit an
account-level session limit mid-run:

> *#181:* "…after pushing its commit but before writing this PR body. I (the
> coordinating/reviewing session) am opening the PR on its behalf, after
> independently reviewing, merge-conflict-resolving, and re-verifying the work
> from scratch"

> *#182:* "…after committing substantial work locally in its worktree but
> before pushing or opening a PR. I … recovered the uncommitted work from the
> worktree, reviewed it, committed it, resolved a clean auto-merge …"

**Credit where it is due, and it is substantial.** The disclosure is
voluntary, specific, and appears in the first line of both PR bodies where it
cannot be missed. Nothing about this was concealed, and the alternative
outcomes — silently re-dispatching two 8-point tickets, or losing T14.4's work
entirely, since it existed **nowhere but an unpushed local worktree** — were
both worse. The recovery work itself is careful: #181's body records a real
`Makefile` merge conflict against the shared branch's newer tip (both T14.1
and T14.9 added `.PHONY` entries and targets in the same region, purely
additive on both sides), resolved by keeping both; #182's records a
`TODO`/`FIXME`/unimplemented-marker scan before deciding to proceed with the
recovered work rather than re-dispatch.

**PE (the property that is missing, measured rather than asserted).** Every
T14 PR was reviewed and merged by the same account, which is this project's
standing supported mode (`sprint-process.md` DoD step 4). That mode's stated
safety property is *"the review was correct"* — which presupposes the reviewer
is reading **someone else's** work. For #181 and #182 that presupposition does
not hold, and the elapsed-time record shows what that costs:

| PR | Created | Review submitted | Merged | Open→review | Open→merge |
|---|---|---|---|---|---|
| #175 T14.6 | 07:00:08Z | 07:01:07Z | 07:01:12Z | 59s | 1m04s |
| #176 T14.3 | 07:00:49Z | 07:02:09Z | 07:02:14Z | 80s | 1m25s |
| #177 T14.2 | 07:02:47Z | 07:04:14Z | 07:04:19Z | 87s | 1m32s |
| #178 T14.8 | 07:09:18Z | 07:11:41Z | 07:11:48Z | 2m23s | 2m30s |
| #179 T14.7 | 07:14:01Z | 07:19:44Z | 07:19:49Z | 5m43s | 5m48s |
| #180 T14.9 | 07:35:27Z | 13:16:03Z | 13:16:08Z | 5h40m36s | 5h40m41s |
| **#181 T14.1** | 13:23:12Z | 13:23:21Z | 13:23:26Z | **9s** | **14s** |
| **#182 T14.4** | 13:28:35Z | 13:28:43Z | 13:28:48Z | **8s** | **13s** |
| #183 T14.5 | 13:45:44Z | 13:48:42Z | 13:48:47Z | 2m58s | 3m03s |

**The two recovery PRs hold the two shortest windows in the sprint, by roughly
an order of magnitude against the next shortest (59s).** The charitable and
almost certainly correct reading is that the verification genuinely happened
*before* the PR was opened — both bodies describe a full toolchain run, a
fresh-worktree re-run and a mutation check performed in the worktree, and the
review text says exactly that: *"all verification described in the PR body was
performed by me independently before opening it."* But that is the point: the
review had **nothing left to do**, because the reviewer had already done it as
the author. Nine seconds is not a review; it is a signature on work already
signed.

**Both PR bodies also contain a sentence the merge record contradicts.**
#181 and #182 each state: *"per CLAUDE.md rule 9, I am not merging this myself
either; a normal review cycle follows below."* Both were merged by the same
account 14 and 13 seconds later, with no other party involved. The intent was
right and the sentence should not have been written, because it describes a
safeguard that did not exist.

**PdE (the third instance, which turns two data points into a pattern).**
Reviewer-as-author appeared a third time this sprint, in a ticket with no
session-limit involved. PR #179's review records:

> *"**Fixed it directly on the PR's source branch**, matching this session's
> established practice for cross-PR gaps found at merge time: added one table
> row per context (`ErrMalformedCourtID` → `InvalidArgument`…). Pushed to
> `feature/t14.7-error-mapping-consistency`."*

The finding it fixed was excellent — T14.8 (merged after T14.7 forked) added a
sentinel with no mapping row, and T14.7's own new exhaustiveness guard caught
it at test-merge, which is the guard doing precisely its job. But the fix was
**authored by the reviewer, on the branch under review, and then merged by
that reviewer**, and CLAUDE.md rule 9 says in terms: *"A reviewer/QA/PE agent's
job is to report findings, never to commit or push itself."* Three of nine T14
PRs therefore contain code the reviewing party wrote. In T13, per that retro's
"No finding on" section, **no PR needed a review-time fix at all.**

**Recorded disagreement — PE vs. QA on how much of this to formalise.**

- **PE:** the recovery was correct, disclosed, and better than every
  alternative; the fix on #179 was correct and cheap. Formalising a "recovery
  protocol" risks the over-correction this project keeps making — a second
  shape of a fix — and the honest mitigation is one line: *disclose it*, which
  already happened unprompted.
- **QA:** disclosure is not a control. The measurable fact is that the two
  PRs whose author and reviewer were the same party received 9 and 8 seconds
  of post-hoc review, and the PR bodies asserted a safeguard ("I am not
  merging this myself") that did not exist. A practice that produces a written
  claim contradicted by the merge record needs more than a norm.
- **Resolved in favour of QA's diagnosis and PE's dosage.** **Adopt
  worktree-recovery-after-session-limit as a named practice with exactly one
  stated safeguard**, not a ceremony — see recommendation 2. The practice is
  worth keeping precisely because it *worked*: it saved an 8-point ticket
  whose only copy was an unpushed local commit.

**What this ceremony did about it, rather than only recommending.** The
safeguard recommendation 2 proposes — *if no independent reviewer is
available, the retro independently re-derives the recovered PR's headline
claim* — was executed here for T14.1, which is the more consequential of the
two: `make gate-coverage` re-run (41 packages, OK), Side A re-derived by hand
(41), and the check mutation-tested from scratch against
`internal/booking/port`, producing a real named failure and a clean restore.
**T14.1's headline claim holds under scrutiny that did not come from its
author.** T14.4's is verified only to the level of "all gates green and the
`AnAdminCannotAppointAnAdmin` test exists and the suite passes" — its
Host-only mutation check was not independently re-performed here, and that is
stated rather than glossed.

## 3. T14.4's work existed only as an unpushed commit in an interrupted agent's worktree — T11.4's exact shape, the one case `LESSONS.md` says polling and branch-listing cannot catch. It was caught. A9(a) reopens on its own stated trigger, and the answer this time is positive

**PdE (the trigger, quoted from the ceremony that set it).** T13's retro
closed A9(a) — the wave roll-call vs. agent polling — as *"unfalsifiable in
practice"*, with an explicit reopening condition rather than a fourth
deferral:

> *"**if a future sprint's work spans more than one work block, the question is
> live again on its own terms** — not as a carried-forward item someone must
> remember, but as a condition anyone can recognise when it occurs."*

T14's plan carried it verbatim into its dispatch section. **The condition
occurred.** Repository evidence: a 5h40m41s window (07:35:27Z → 13:16:08Z) in
which no PR was opened, reviewed or merged, bracketing an account-level
session limit that terminated two implementer sessions mid-run. Whether that
window is literally "more than one work block" is not determinable from
repository evidence alone — session exhaustion and an interruption leave
identical traces, exactly as `LESSONS.md`'s T11 entry says — but the substance
of the condition (agents becoming unreachable with work unpublished, for
hours) is unambiguously met.

**QA (why this is the *strong* test, not a re-run).** `docs/LESSONS.md`'s T11
entry ends on a specific, falsifiable claim about which detector catches which
case:

> *"polling agent-liveness or listing remote branches would **not** have caught
> T11.4, whose branch was never pushed — only comparing against the dispatch
> list catches the case where the work exists solely as an unpushed local
> commit, which is also the case where it can be lost outright."*

T14 produced **one of each case**, which is why this is the cleanest possible
test:

| Ticket | State when its session died | Would branch-listing have found it? | Outcome |
|---|---|---|---|
| **T14.1** (#181) | committed **and pushed**; no PR | Yes | recovered |
| **T14.4** (#182) | committed **only**, in the agent's worktree, never pushed | **No** | recovered |

T14.4 is T11.4's shape exactly. It was found, in-block, and published as a
32-file / +2467-line PR. The detector `LESSONS.md` prescribed is the only one
that could have found it, and this is the first time since T11 that anything
has tested it.

**Honest qualification, per CLAUDE.md rule 10 applied to process.** This
ceremony cannot prove *from repository evidence* that a formal roll-call
against the dispatch table is what found T14.4, as opposed to the coordinating
session simply noticing. What is provable is that both interrupted tickets
were recovered inside the same sprint, and that one of them was recoverable by
no other means. That is one successful run, and it is being recorded as one
successful run.

**Score: A9(a) is reopened by its own trigger and resolved positively, on
better evidence than the question has ever had.** The roll-call stays standing
practice — no longer as "a cheap safety net that has never been needed", which
is how T13's retro had to phrase it, but as **a detector with one confirmed
catch of the specific case that no alternative detector covers.** Polling and
remote-branch listing remain **not adopted**, now for a demonstrated reason
rather than a predicted one: branch-listing would have found T14.1 and missed
T14.4.

## 4. T14.1's gate is real, and it found a gap two independent human enumerations missed in the same sprint. Of its three proofs, the one the sprint itself supplied is the weak one — and this retro says which

**PE (the strongest evidence, which is not the one the brief highlighted).**
The plan's A3 re-enumerated #157's gap and found 22 ungated packages against
#157's claimed 20 and listed 21 — already a demonstration that hand
enumeration goes stale. **T14.1 then found a category that A3's own
re-enumeration also missed.** From PR #181's body, verified against the
merged `Makefile`:

> *"`test-cmd` (`cmd/server/main_test.go`'s T13.5 startup-refusal tests, which
> #157 itself and T14's own planning both missed — **neither scanned `cmd`**)"*

Confirmed: A3's enumeration command was `grep -rl "func Test" … internal tools`
— `cmd` is absent from it, twice (`docs/process/t14-sprint-plan.md` A3 and
instruction 5). `cmd/server/main_test.go` holds the tests proving the server
refuses to start without an auth verifier — **#136's regression test, the auth
spine's own startup guarantee** — and it ran in no gate. `make test-cmd` now
runs it (verified green here). This is the mechanical check doing the thing
three sprints of hand-written globs could not: catching a category nobody
framed, including the category the ceremony that commissioned it did not
frame.

**QA (the three proofs, ranked, because they are not equal).**

1. **Weakest — the sprint's own siblings.** T14.4 and T14.5 both merged after
   T14.1 and both added test files; `make gate-coverage` reported 41 packages
   covered with no intervention in both reviews, and this ceremony re-derived
   41 independently. **But the count did not change**: every T14.4/T14.5 test
   file landed in a package that already held tests and was already gated
   (`internal/socialplay/{domain,app,adapter/grpcapi,adapter/postgres}`). The
   dynamic-enumeration property was exercised at *file* granularity, which is
   the easy case.
2. **Middling, and genuinely a sibling instance — `tools/devtoken`.** T14.9
   (#180, merged 13:16:08Z) created `tools/devtoken`, a **new package holding
   tests**, seven minutes before T14.1 merged at 13:23:26Z. It was captured by
   the existing `./tools/...` pattern with zero edits to T14.1's output. That
   *is* A5's dual-coverage scenario at package granularity, and it passed —
   though it passed easily, because the sibling landed inside an existing
   pattern.
3. **Strongest — mutation, performed twice by two parties.** PR #181's author
   added a synthetic `func Test` to `internal/booking/port` and confirmed a
   real named failure. **This ceremony repeated that from scratch**, on the
   merged tip, and reproduced it exactly:

   ```
   gate-coverage: FAIL — 1 package(s) hold tests that NO gate executes:
       internal/booking/port  (1 test func(s), e.g. TestRetroSyntheticProbe)
   ```

   Removed, re-run, `OK — all 41`. The tool's own suite additionally carries
   `TestPackageAddedAfterTheGateWasWritten` and `TestRepoMakefileIsStillParseable`.

**PdE (the design property worth naming, since it is what makes this different
from the three globs before it).** Side B is derived by **parsing this repo's
own `Makefile`** and expanding each pattern with `go list` — verified in the
run output, which prints the five invocations it found and which target each
came from. There is no package list in the tool. The three-state
classification the ticket demanded is implemented and disjoint (`Covered`,
`Ungated`, `CompiledOnly`) — `CompiledOnly` reports build-tagged-only packages
as a NOTE rather than a failure, which is T11's known-and-accepted state
correctly preserved rather than silently folded into "covered". No
`CompiledOnly` packages exist today, because every package holding integration
tests also holds Docker-free ones.

**No finding against T14.1.** It is the strongest artifact this sprint
produced and it closed a three-sprint class. The qualification in (1) is
recorded so that nobody later writes "the gate was proven against a
newly-created package by its own sprint siblings" — it was proven against a
newly-created package by *mutation*, which is a better proof, and the sibling
evidence is real but softer than the reviews' phrasing implies.

## 5. T14.6's label taxonomy was applied to the three issues it was handed and not swept across the list it now governs; no review performed the conformance check the same ticket added; and one PR and its review both described a closed issue as open

**BA (the taxonomy hole, from the live label state).** T14.6 made `role:`
**mandatory on every issue** and relabelled the three non-conformant issues
A9 named — verified correct, all three:

| Issue | Labels now | Conformant? |
|---|---|---|
| #168 | `role:product-engineer`, `type:chore`, `context:socialplay`, `context:payments` | ✅ (`type:tech-debt` retired as specified) |
| #167 | `role:principal-engineer`, `type:story`, `context:payments` | ✅ (was unlabelled) |
| #165 | `role:principal-engineer`, `type:chore` | ✅ |

Then checked the **other ten** open issues, which nobody was handed:

| Issue | Labels | Conformant? |
|---|---|---|
| #147 | **none** | ❌ — no `role:`, no `type:` |
| #149 | **none** | ❌ — no `role:`, no `type:` |
| #124, #125, #126, #130, #134, #137, #144, #145, #164 | `role:` + `type:` present | ✅ |

**Two of the thirteen open issues carry no labels at all, and they are #147 and
#149 — the two the sprint plan's own §A7 and §A16 discuss at length.** The
amendment landed with a hole in exactly the region of the backlog it was
written about. This is not a new mistake: it is BA's T13 point recurring one
level out. The taxonomy was *decided* durably and *applied* to a hand-written
list of three, and a hand-written list is the thing this sprint's marquee
ticket exists to argue against.

**PE (the check that was added and never run).** T14.6's amendment states:
*"**Label conformance is checked in review.** … That same enumeration now
checks each named issue against this taxonomy."* Checked all nine T14 reviews:
**none performs a label-conformance check.** PR #175's review verifies the
three relabellings the PR itself claimed — which is verifying the diff, not
performing the standing check on the issues its own PR names.

**The closure-enumeration line, scored the same way.** T14.6's amendment
requires every **review** to state the issues its PR closes, by number or
explicitly "none". Checked all nine reviews and all nine PR bodies:

- **Reviews carrying the enumeration line: 0 of 9.**
- **PR bodies carrying it: 1 of 9** — PR #175's, T14.6's own, which opens
  *"Issues this PR closes: none."* Its review noticed and credited it: *"a
  small but correct demonstration of the amendment it's proposing."*

The one instance is in the wrong artifact (the PR body, written by the
implementer) and the artifact the rule actually binds (the review, written by
the reviewer) scored zero. Several reviews *discuss* closure in prose — #180's
*"#160 was a real quality-of-life gap and this closes it properly"*, #183's
*"correctly leaving #147 open (not 'closes') is the right call"* — which is the
judgement, again performed well, without the enumeration.

**QA (the citation error, small but exactly the shape T13's finding 1
named).** PR #178's body and PR #178's review both describe **#97** as an open,
disclosed gap:

> *body:* "the disclosed #97 gap on `bookingdomain.ErrInvalidCourtReference` is
> untouched by this PR"
> *review:* "doesn't touch the separate, **still-open** #97 gap on
> well-formed-but-unknown court ids"

**#97 is closed** — `state: closed`, `state_reason: completed`, `closed_at:
2026-08-13T14:45:54Z`, closed by T11.8's retroactive sweep. Worse, #97 was
never about that gap: its subject is *malformed*-ID boundary guards on write
handlers, which is what T14.8 just finished. **The residual behaviour T14.8
disclosed — what a well-formed but unknown `court_id` answers — is therefore
tracked nowhere at all.** Two parties stated an issue's state confidently and
neither read it. That is the same failure mode as T13's *"the record says the
thing was done because someone wrote that it was done"*, applied to an issue's
state instead of a close.

**The natural fix is one clause on a rule that already exists**, not a new
rule — see recommendation 4.

**Worth crediting in the same breath, because it is the counter-example.**
T14.8's instruction 5 asked for a sibling sweep with the result reported
either way, and PR #178 delivers a genuinely exhaustive one: every `repeated
string` id field across all six protos and every `[]string` in each context's
app inputs and domain aggregates, four categories enumerated with reasons,
concluding *"no other instance, so no issue opened."* **T14 opened zero
issues, and at least this instance of zero is a checked zero, not a silence.**

## 6. The three scorings the plan owed, plus the two escalated decisions — none deferred

`docs/process/t14-sprint-plan.md`'s sprint-level DoD names three scorings
"stated now so they are not improvised". All three are scored; two resolve,
one is correctly not yet scoreable and this retro says so rather than
inventing a verdict.

### (a) Recommendation 4's dual coverage question — **DROP it. T14.1 merged.**

`sprint-process.md`'s "Scheduled removals" table states the condition
precisely: the question is removed *"by **T15's Ceremony 1** — drop it; it does
**not** become a fourth standing planning question"*, on the trigger *"**T14.1
merges**"*. T14.1 merged (PR #181, 13:23:26Z), `make gate-coverage` is wired
into `ci-checks` (verified, `Makefile:326`) and runs green.

**The condition is met and the removal is not a judgement T15 has to re-make —
it executes the plan.** Recorded emphatically because this project's named
failure mode is accumulating two shapes of one fix, PE called it in advance,
and the scheduled-removals table exists specifically so that the removing
ceremony does not have to re-argue it. **Keeping the question would be the
failure the table was built to prevent.**

### (b) A1's PE-vs-PO disagreement on the closure sweep — **PO, with a third state named.** Scored in full in finding 1.

In brief: the per-PR half (PE's primary) scored 0/6 for the second consecutive
sprint; the sprint-level half is what produced the 6/6 outcome; and neither of
the sweep's two sanctioned moments ran, because the merger swept its own work
at sprint end. PE's *cost* argument is confirmed by measurement (the list
misdescribed the codebase for up to 6h45m, inside the sprint). PE's *remedy*
is not. The disagreement does not close — it acquires a third position that
recommendation 1 turns into text.

### (c) A16's QA-vs-PdE disagreement on partial fixes — **not scoreable yet, by its own terms; what *is* checkable is checked here.**

A16's scoring condition is explicit and belongs to the next sprint: *"if T15
does not close the Competitions half, QA's 'permanent furniture' prediction is
confirmed and #147 should be re-ranked accordingly."* This retro does not
pre-empt it. What this ceremony can establish is whether PdE's claim — *"T14
builds the mechanism; the remaining work for Competitions is a second instance
of a proven pattern, not another blocked question"* — is now true, since that
is the premise T15's score depends on. Verified against the tree:

- **The mechanism exists and is complete on the Social Play side.**
  `db/migrations/0020_socialplay_game_admins.sql`,
  `internal/socialplay/domain/game_admin.go`,
  `internal/socialplay/port/game_admin_repository.go`, the Postgres adapter
  with `ListGameAdmins`, `AssignGameAdmin`/`RevokeGameAdmin` RPCs in
  `AuthenticatedMethods()`, and the read consumed by
  `EnsureHostOrGameAdmin(actorUserID string, assigned []GameAdmin)`
  (`internal/socialplay/domain/game.go:152`).
- **Nothing blocks the Competitions instance.** `grep -rli
  "competition_admin\|CompetitionAdmin" db/migrations internal/competitions`
  returns **nothing** — no store, no domain type, no port, and no blocked
  question either. `internal/competitions/app/service.go:490`
  (`ListEntriesForCompetition`) still calls bare `EnsureHost`, unchanged.
- **T14.5 stated the negative as instructed**, and its review independently
  re-verified it by grep rather than accepting it.

**So PdE's premise holds: the Competitions half is now a second instance of a
shipped pattern.** QA's concern also holds unchanged: #147 has now been open
across three sprints with two partial fixes. **The score is T15's, and T15
should be able to take the Competitions half at a fraction of T14.4's 8
points** — which is what makes the prediction falsifiable.

### (d) Wave-1.5 checkpoint — **nothing to score, as A10 predicted, and that is the recorded result.**

A10 checked the condition (a new cross-cutting decision with three or more
first-time in-sprint consumers) and found it did not fire: T14.4 had exactly
one in-sprint consumer, T14.5. No checkpoint was applied, so **T14 is not a
sprint that counts toward "several sprints passing with every checkpoint
merging first-loop"**, and PdE's cost objection (T13 finding 3) is carried
forward untouched and unscored, exactly as T13's retro specified. Recorded so
that the *absence* is legible as a checked absence rather than an omission.

### (e) ADR-0012's Q1/Q2 and ADR-0015's D1 — **both still blocked on the user, neither guessed at nor silently dropped.**

- **ADR-0012 Q1/Q2** (Player Level formula weighting; whether gender-mix
  matching is in scope at all). Re-verified by grep across `internal`, `proto`
  and `db`: **every occurrence of `PlayerRating` or `Gender` is a doc comment
  stating the field deliberately does not exist** —
  `internal/identity/domain/user.go:11,18`,
  `internal/socialplay/domain/match.go:7,13`,
  `internal/socialplay/domain/errors.go:158`,
  `db/migrations/0016_identity.sql:5-6`. **No `Gender` field, no
  `PlayerRating` type, no Level-scoring formula anywhere in the tree.** Sixth
  consecutive sprint recorded this way.
- **ADR-0015's D1** (what owns a Booking made through the public
  quote-and-book flow). Re-verified untouched: `bookings` still has no owner
  column of any kind (`db/migrations/0001_init.sql:22-42` — `id, court_id,
  source, status, starts_at, ends_at, during, reference_id, created_at`);
  `app.Service.CancelBooking(ctx, bookingID string)`
  (`internal/booking/app/service.go:287`) still takes **no actor parameter**;
  `CreateBooking` and `CancelBooking` are both still in `PublicMethods()`
  (`internal/booking/adapter/grpcapi/authenticated.go:73-74`). **The hole is
  exactly as wide as it was.**

  **T14.3 is a model escalation and is worth crediting specifically.** ADR-0015
  picks no option, states the question in a sentence a non-engineer can answer,
  distinguishes D1 from ADR-0012's Q1/Q2 (ordinary-but-unanswered vs.
  legally/ethically blocked), carries a restriction list forbidding any future
  PR from guessing the answer, ties its trigger to the user's answer rather
  than a sprint boundary, and **corrected the ceremony that commissioned it**:
  verifying that nothing in `web/src` calls `CancelBooking` and that no
  `.swift`/`.kt` source exists, it added a **fourth** option (authenticate
  cancellation only, leave creation public) that A6's three-option table did
  not contain. That correction is recorded on #144 as a comment, on the day,
  with the facts re-derived.

  **One stale reference this created, and it is finding 5's shape again:**
  `HANDOFF.md` still says *"ADR-0015 records **three** options"* (line 609,
  and the T14 Docs-index row at line 36) against an ADR that records four. PR
  #176's implementer correctly declined to touch `HANDOFF.md` as an unassigned
  shared file (A14), flagged it for the reviewer, and **the reviewer wrote
  *"I'll fix that stale reference directly since it's a one-line correction to
  already-merged text"* — and did not.** `git log -- HANDOFF.md` shows its last
  change was PR #180 (T14.9's resume-step correction); no PR corrected the
  option count. Per `sprint-process.md`, `HANDOFF.md` corrections belong to
  T15's Ceremony 1, and this retro deliberately does not touch it — but the
  correction is now on record so Ceremony 1 does not have to rediscover it.

---

## Recommendations for T15's Ceremony 1 and 2

Concrete and mechanical, in the spirit of T11 finding 2 → T12.1, T12
recommendation 1 → T13's A13, and T13 recommendation 2 → T14.1.

1. **Amend `sprint-process.md`'s merged-fix sweep to name the third state T14
   produced, and score it honestly rather than as compliance** (finding 1).
   Today the amendment describes two sanctioned moments — the retro, and the
   next Ceremony 1 — and a per-PR step. T14 used **none of them**: the merging
   session swept its own work at sprint end, 65 seconds before dispatching the
   retro, achieving a correct outcome with no independent check. Add that as an
   explicitly-named third moment, classified as **acceptable-but-not-
   sufficient**, with the consequence stated: *when the merging session sweeps
   its own work, the sweep's "party other than the merger" property is not
   obtained, and the next Ceremony 1's run is not thereby discharged — it
   re-runs the arithmetic anyway.* Do **not** re-exhort the per-PR half a third
   time without changing its shape; it has now scored 0/9 and 0/6 in
   consecutive sprints while every other quality signal was green.
2. **Adopt worktree-recovery-after-session-limit as a named practice with
   exactly one safeguard, and write it into `sprint-process.md`'s Execution
   section** (finding 2). Verdict: **adopt-as-named-practice, with a stated
   safeguard** — not one-off, because it saved an 8-point ticket whose only
   copy was an unpushed local commit and the same interruption class will
   recur; not a ceremony, because the recovery itself was performed well and
   disclosed unprompted. The practice:
   - **(a)** When an implementer session ends without a PR, check its worktree
     for unpushed commits *and uncommitted work* before re-dispatching. Listing
     remote branches is not sufficient — T14.4 had no remote branch (finding 3).
   - **(b)** The recovering session may commit, push and open the PR, and
     **must** carry a first-line provenance note (T14 did this correctly, and
     the wording in #181/#182 is the template).
   - **(c)** **The safeguard:** a recovered PR is reviewed by a *different*
     party than the one that recovered it. Where no independent reviewer can be
     dispatched — which was the case here — the recovering session says so
     plainly **instead of** writing "I am not merging this myself", and the
     sprint's retro independently re-derives that PR's headline claim and
     records the result. This retro discharged that for T14.1 (gate re-run,
     Side A re-derived, mutation reproduced) and states explicitly that it did
     **not** re-perform T14.4's Host-only mutation check.
   - **(d)** Delete the sentence "I am not merging this myself either" from any
     PR the author intends to merge. A written safeguard that does not exist is
     worse than an acknowledged absence.
3. **Rule 9's "a reviewer never commits" needs either enforcement or an honest
   carve-out — pick one and write it down** (finding 2, third instance). T14.7's
   review found a real cross-PR gap at test-merge and fixed it on the branch
   under review, calling it *"this session's established practice."* It is
   either an established practice, in which case `sprint-process.md` should
   describe it and bound it (e.g. *mechanical, compiler- or test-caught,
   single-file, disclosed in the review, and the fix itself re-verified*), or it
   is a rule-9 violation, in which case the reviewer should have requested
   changes. It is currently both simultaneously, which is the state that
   guarantees it drifts.
4. **Add one clause to the review's issue enumeration: state each named
   issue's current state, read from the API, not from memory** (finding 5). The
   enumeration already exists (issues opened, issues closed, label
   conformance); this costs one field on a call already being made. It would
   have caught PR #178's *"still-open #97"* — an issue closed two days earlier,
   about a different gap — and with it the fact that T14.8's genuine residual
   (what a well-formed but unknown `court_id` answers) is tracked nowhere.
   **Also file that residual as an issue at Ceremony 1**, per the board-of-
   record rule: it outlives its sprint, so it is mandatory, not discretionary.
5. **Sweep the label taxonomy across the whole open list once, then stop
   hand-listing** (finding 5). #147 and #149 carry **no labels at all** and
   `role:`/`type:` are now mandatory. Fix both at Ceremony 1 as bookkeeping.
   Then note the pattern: T14.6 applied its taxonomy to a hand-written list of
   three in a sprint whose marquee ticket exists to prove hand-written lists go
   stale. If a third instance of label drift appears, the answer is a
   `make label-conformance` check against the live API, not a fourth
   relabelling pass — but **do not build it pre-emptively**; one sweep first.
6. **Take the Competitions half of #168/#147 in T15, and re-title #147**
   (finding 6c). PdE's premise is verified — no Competition-Admin store exists,
   nothing blocks building one, and T14.4 is a complete worked pattern to copy,
   so the ticket should cost materially less than T14.4's 8 points. A16's
   scoring condition makes this the sprint that decides QA's "permanent
   furniture" prediction. Separately, **#147's title is now factually wrong for
   both halves** — it reads *"ListRegistrationsForGame and
   ListEntriesForCompetition have no owner check"*, and both have had one since
   T13.6. Re-title it to the gap that actually remains (Competitions' roster
   read is Host-only, not Host-or-Competition-Admin) and add the "why it stays
   open" comment the sweep's second disposition required and nobody wrote
   (finding 1).
7. **Drop A5's dual coverage question. Do not renew it, do not adapt it, do not
   fold it into another question** (finding 6a). The scheduled-removals table
   states the condition and it is met. This is an execution step, not a
   decision — and the table exists precisely so that the removing ceremony does
   not re-litigate it.
8. **#144 is now on its third sprint. A third deferral without an answer is a
   finding, and T14.3 wrote that down in advance.** ADR-0015 is a model
   escalation and it changes nothing about the hole: anyone who knows a booking
   id can still cancel that booking. Ceremony 1 should put D1 to the user **as
   its own item**, not as a line inside a plan document, and if the answer
   arrives the implementation is small (`bookings` gains an owner column,
   `CancelBooking` gains an actor parameter, the RPC leaves `PublicMethods()`).
9. **Correct `HANDOFF.md` at Ceremony 1, including one item this retro found
   that the standing checklist does not cover** (finding 6e). Beyond the
   standing three (T14's Docs-index row with its real retro path and
   `merged_at`-verified PR order; a new T15 row; T14's Task-backlog narrative
   in the form this retro agreed, below), fix the **stale "three options"**
   reference to ADR-0015 at `HANDOFF.md:36` and `:609` — the ADR records
   **four**. PR #176's reviewer said in writing it would fix this and did not.
10. **Do not treat "zero issues opened" as a quality signal without checking
    it.** T14 is the first sprint since issues became the board of record to
    open none, and the open count fell 19 → 13 — a genuinely good outcome. But
    exactly one ticket (T14.8) *demonstrated* a checked zero, with an
    exhaustive documented sweep. For the rest, zero-opened and
    nothing-was-disclosed are indistinguishable from the outside, which is the
    same epistemics as finding 1's "a sweep whose output is silence." Ceremony 1
    should note which is which before it reads the falling number as progress.

---

## The sprint goal, scored: what was proven, what shipped unproven, and what is still open

T14's goal is long and has five positive clauses plus an explicit
"what this sprint does not claim" section, which makes it unusually scoreable.
Taken clause by clause against evidence gathered here — including a full
Docker-free gate run and an independent mutation check — rather than from PR
bodies.

> *"Every package in this repository that holds a test is executed by a gate a
> session can actually run, enforced **mechanically** rather than by a list
> someone remembers to update — so the third consecutive sprint of the same
> class is the last one"*

**Met, and independently proven rather than reported.** `make gate-coverage`
returns `41 package(s) … OK`, Side B derived at run time from five `go test`
invocations parsed out of the `Makefile` itself. Side A independently
re-derived here by hand: 41. The check was mutation-tested by this ceremony
against `internal/booking/port` and produced a real, correctly-exiting,
package-naming failure. Two new gates (`test-adapters`, 22 packages green;
`test-cmd`, `cmd/server` green) fill the gaps it found — including
`cmd/server/main_test.go`, which **#157 and T14's own Ceremony 1 both missed
because neither scanned `cmd`**, and which holds the auth spine's
startup-refusal tests.

**Honest qualification (CLAUDE.md rule 10):** "executed by a gate a session can
actually run" is proven. **"Reaches CI" is still not**, unchanged since
SCRUM-6: `Jenkinsfile` calls `make ci-checks`, and there is still no Jenkins
job, webhook or branch protection, and no session here can create one. The
gate widens what that pipeline *would* run.

> *"`gofmt` becomes a gate instead of a review convention"*

**Met.** `make fmt-check` exists (`Makefile:204`), is wired into `ci-checks`
after `generate` (`Makefile:326`), and runs green here (`gofmt clean across
./internal ./cmd ./tools`). #165's two-line violation is fixed and the diff was
whitespace-only, keeping T14.5's Wave-3 handoff on `game.go` clean as A4
designed. The gate was mutation-checked by its reviewer (a real violation
injected into `internal/payments/domain/payment.go`, `ci-checks` halted at
`fmt-check` before reaching `lint`). Confirmed additive, not a duplicate:
`golangci-lint run ./...` reports 0 issues.

> *"a Game Admin becomes a fact the system stores rather than a claim the
> caller makes, so Social Play's roster read can widen to the people actually
> entitled to it"*

**Met for Social Play.** `game_admins` exists (`0020`), with `text` actor
columns per ADR-0014 §5a and a migration comment citing §5a and #164 so a
future backfill finds it. `EnsureHostOrGameAdmin` now takes
`assigned []GameAdmin` — **a deviation from the ticket's literal instruction**
(which said drop the parameter), adopted deliberately: a request DTO cannot
construct a `[]GameAdmin`, so the type *is* the enforcement, and the domain
stays pure with resolution happening in `app`. The reviewer scored the
deviation on its merits rather than against the letter, correctly.
`ListRegistrationsForGame` is Host-or-assigned-admin, mutation-checked by the
reviewer (reverting to `EnsureHost`-only produced 4 real named failures across
`app` and `adapter/grpcapi`). "An admin cannot appoint an admin" is enforced
and pinned by `TestAssignGameAdmin_AnAdminCannotAppointAnAdmin`.

**Two qualifications, both disclosed in-sprint and neither retracted here.**
**(i)** Social Play only. Competitions' `ListEntriesForCompetition` is
unchanged Host-only and no Competition-Admin store exists — re-verified by
grep here, not taken from the PR. #147 and #168 correctly stay open. **(ii)**
Payments' `RecordOfflinePayment` still trusts caller-supplied
`assigned_game_admin_user_ids` / `assigned_competition_admin_user_ids` (both
still on the wire, `payments.proto:115,118`); #149 is untouched, materially
advanced only in the sense that its named sub-gap now has a Social Play
implementation to copy.

> *"and the nine issues T13 fixed but never closed are closed, with the
> mechanism that let that happen amended into the process"*

**Met — and the mechanism's own first live test is finding 1.** T14's Ceremony
1 closed the nine (verified: 28 − 9 = 19), T14.6 amended
`sprint-process.md` with the per-PR closure step, the merged-fix sweep, the
decided label taxonomy and a scheduled-removals table. Then **the amendment's
per-PR half scored 0 out of 6 in the sprint that wrote it**, all six closes
landing in an eleven-second batch at sprint end, run by the merging party.
The outcome is right; the mechanism that produced it is the backstop, not the
step; and no party other than the merger ever checked. The label taxonomy
landed with #147 and #149 unlabelled, and no review performed the conformance
check the same ticket added.

> *"**What this sprint does not claim:** `CancelBooking` still has no
> authorization check (#144)… Competitions' roster read and Competition-Admin
> store are untouched (#147, #168 stay open); Payments still compares a
> verified actor against caller-supplied ownership facts (#149…); and no token
> from a real identity provider can be verified until a remote JWKS source
> exists (#137)."*

**All four negative claims verified true, individually, against the tree** —
`bookings` has no owner column, `CancelBooking` takes no actor and stays in
`PublicMethods()`; no `competition_admin` anything exists; the payments protos
still carry the caller-supplied admin lists; #137 remains open and blocked.
**A sprint goal whose negative half is checkable and checks out is worth as
much as the positive half**, and this is the third consecutive sprint to write
one.

**The agreed honest sentence, which T15's Ceremony 1 should carry into
`HANDOFF.md`'s Task-backlog entry** (`sprint-process.md` Ceremony 1 item 3
requires the retro's form, not a stronger one):

> T14 answered the gate question mechanically rather than for the fourth time:
> `make gate-coverage` derives both sides at run time — packages holding tests
> from a source scan, packages some gate executes by parsing the `Makefile`
> itself — and fails naming any package no gate runs. It found a category
> #157 and T14's own Ceremony 1 had both missed (`cmd/server`'s
> startup-refusal tests), and all 41 test-holding packages are now executed by
> `ci-checks`, verified green and mutation-checked independently at the retro.
> `gofmt` became a gate. A durable `game_admins` store shipped for Social Play,
> so `ListRegistrationsForGame` is Host-or-assigned-admin with the assignment
> resolved from the database rather than from the caller — the first
> authorization rule here that can include an admin without being defeatable by
> naming yourself one. #144 was escalated as ADR-0015 with four options and no
> recommendation, not guessed. Six issues closed and none opened: the open
> count fell 19 → 13, and T14 is the first sprint since the board-of-record
> split to open none. **Two of the nine tickets were finished by the coordinating session
> recovering interrupted agents' work — one of them existed nowhere but an
> unpushed local worktree — and those two PRs were authored, reviewed and
> merged by the same session, 14 and 13 seconds from open to merge.** All six
> issue closes were correct and cited, and all six happened in an
> eleven-second batch at sprint end run by the merging party, so the per-PR
> closure step T14 itself adopted scored 0/6 and no independent party ever
> checked the sweep. Competitions' roster read, the Competition-Admin store
> (#147, #168) and Payments' caller-supplied ownership facts (#149) are
> untouched; ADR-0012's Q1/Q2 and ADR-0015's D1 remain blocked on the user.

---

## No finding on

**No finding on T14.9 beyond the credit it is owed, and its review is the
strongest verification act in this project's history.** #160 was closed with a
real key fixture rather than a bypass, exactly as T13.5's reasoning required:
a committed RSA keypair plus JWKS under `dev/auth/` with an unmissable warning
banner and `kid: "dev-only-insecure-do-not-trust"`, bind-mounted read-only into
compose so no built image carries key material, plus a `make dev-token` CLI.
The reviewer did not stop at reading it — it stood up a real Postgres 16,
applied every migration, ran the real server binary, confirmed it still
**refuses to start with no `AUTH_*` set** (T13.5's property preserved, proven
by mutation rather than asserted), minted a token, and drove an authenticated
request end to end until the token's `sub` claim appeared in
`identity_users.subject`. That is the "work for a real caller" standard T13's
retro had to qualify away, met for the first time.

**No finding on T14.3.** Scored in 6(e): an ADR that picks no option, states
the question for a non-engineer, carries a restriction list forbidding future
PRs from guessing, ties its trigger to the answer rather than a sprint
boundary, distinguishes itself from ADR-0012's permanently-blocked class, and
**corrected the ceremony that commissioned it** by verifying the client-side
facts and adding a fourth option. It also correctly declined to touch an
unassigned shared file and flagged it instead — the failure that followed was
the reviewer's, not the ticket's.

**No finding on T14.7's collision handling, which is the third clean instance
of a pattern this project has now established.** T14.4 added its new sentinel's
mapping into T14.7's *corrected* `toStatus` rather than reverting or
duplicating it — the same clean handoff T13.7 made with T13.9's, and A14's
wave-separation of three tickets on `internal/socialplay/app/service.go` and
three on its `handler.go` held with no conflict. The one real cross-PR gap
(T14.8's `ErrMalformedCourtID` with no mapping row) was caught by T14.7's own
new exhaustiveness guard at test-merge — the guard doing exactly the job A13
arrow 4 predicted for it, one wave earlier than predicted. Only *who fixed it*
is a finding (finding 2), not that it was found.

**No finding on the wave structure.** All three waves executed in planned
order; no Wave-1.5 checkpoint was required and A10's check that it does not
fire was correct; `0020` was claimed by exactly one ticket as pre-assigned;
ADR `0015` by exactly one. The 5h40m gap is an interruption, not a sequencing
failure.

**PM had limited independent material this sprint, as in T9–T13.** PM's
substantive contribution is inside finding 1: the open-issue count fell 19 →
13 and that is a real roadmap input, but the number was produced by the
merging session grading its own homework, and a backlog number nobody
independent has checked is a number that will be trusted exactly as much as
one that has been. The product framing PM contributed at Ceremony 1 — leading
the sprint goal with what changes for a *reader of this repository* and naming
what the sprint does not claim — is why the "does not claim" half of the goal
was checkable at all, and it is worth keeping.
