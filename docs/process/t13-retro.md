# T13 Sprint Retro

Ceremony 3 per `docs/process/sprint-process.md` (read in its **T13-amended**
form, since T13's own Ceremony 1 added the "Correct the previous sprint's
Docs-index row" section this retro is governed by), six-role team (briefs:
`docs/agent-operating-handbook.md` Part B), held against
`docs/process/t13-sprint-plan.md` (including its A0–A17 appendix),
`docs/process/t12-retro.md` as the precedent and rigor bar, `HANDOFF.md`, and
the real PR/issue history on `nhuthuynh/white-label` (GitHub-side name
`pickleball-platform`) — PRs #155–#172, issues #123–#168.

Every timestamp, merge order, branch ancestry and file claim below was pulled
from GitHub's own `created_at`/`merged_at`/`submitted_at` fields and from
`git`/`go`/`make` run against the merged branch at `066b237` — not inferred
from titles, and not taken from the coordinating session's own summary of the
sprint. Claims checkable against the working tree were checked here:
`make test-domain` was re-run (green, 12 packages), `make test-platform` was
re-run (green — `auth`, `auth/rs256`, `grpcrecovery`; `idgen` and `pg` carry no
test files), the `Makefile`'s `ci`/`ci-checks`/`test-platform` targets were read
rather than trusted, `gofmt -l` was re-run against the tip, `go list` was used
to derive which packages each gate actually reaches, `git branch -r --contains`
was used to establish wave ancestry structurally rather than by timestamp, and
ADR-0014 §5/§5a and `internal/booking/port/identity_lookup.go` were read
line-by-line to adjudicate the plan's own dependency-completeness claims.

**This retro's assigned investigation points are not taken as given.**

- **Finding 1 was not on the list this ceremony was handed, and it inverts the
  sprint's headline claim.** The brief given to this ceremony stated that T13.2
  "closed #146/#152", T13.3 "closed #154", T13.4 "closed #138/#129", T13.5
  "closed #136/#135", T13.7 "closed #148" and T13.8 "closed #123". Checked
  against the live issue list: **not one of those nine issues is closed.** All
  28 issues in this repository are open. The code shipped and is verified; the
  bookkeeping step that `sprint-process.md` DoD step 5 exists to compel did not
  happen once in nine opportunities.
- **Finding 2 reframes #157 from "a gap T13.1 found" into "a gap in T13.4's own
  fix, produced by the dependency-completeness check's blind spot."** The two
  tickets ran in the same wave. One wrote five new test packages; the other
  built the Docker-free gate. The gate does not run the tests. A13 arrow 7
  dismissed T13.4 with the sentence *"Nothing consumes it; it is a gate, not a
  capability"* — which is the exact sentence that missed it.
- **Finding 3 declines to score the Wave-1.5 checkpoint the way its own
  evidence invites.** The checkpoint is vindicated on outcome, and the
  ancestry evidence is stronger than the plan asked for. But PdE's objection
  was about *cost under a second review loop*, and T13.2 took one loop and
  merged in 5m14s. The cost was never tested. Recording "the checkpoint is
  cheap" from this sprint would be CLAUDE.md rule 10 violated at the process
  level.
- **Finding 4 resolves a disagreement T12 left open, in the direction T12's
  own evidence pointed away from** — and finds BA's half of it recurring
  anyway, in the precise place BA predicted.
- **Finding 5 is a two-line diff that survived nine PRs**, and the interesting
  part is that every ticket's scope discipline was *correct* and the outcome
  was still stuck.

Findings are not a single voice. Recorded disagreements are left as
disagreements per the "do not manufacture consensus" rule, matching T9's, T10's,
T11's and T12's retros.

**Sprint outcome:** all 9 tickets (40 points, T13.1–T13.9) merged across PRs
#159–#172, plus the Ceremony 1/2 doc (#155). The sprint ran in a single unbroken
work block, **2026-08-15T05:16:29Z to 06:19:55Z — 1 hour 3 minutes 26 seconds**
from plan merge to final merge, the fastest full sprint in this project's
history (T12: 1h47m41s). All three waves executed in their planned order, the
Wave-1.5 checkpoint was observed in its strongest available form, and every
dispatched ticket produced a PR — **no silent agent** (A9(a) scored in
finding 6). No implementer PR needed a review-time fix, no defect reached the
shared branch, and no ticket consumed a second loop. **This is the first sprint
since T10 with no defect escaping to the shared branch, and the first ever in
which every ticket merged on its first loop.**

The engineering is genuinely strong and the review record is the best this
project has produced: every one of the nine PRs carries a discoverable GitHub
review object; every review records an independent fresh-worktree or test-merge
verification, a full toolchain run performed by the reviewer, and **a mutation
check the reviewer performed itself** rather than reading the implementer's
captured table. Two reviews caught more failures than the PR claimed (#166's
seam mutation produced 6 failures against the PR's claimed 4).

**What did not happen is administrative, and it is finding 1.** Nine issues
this sprint's code resolves remain open, so the "eleven tracked exceptions"
number T12's retro coined has not gone down as a matter of record — it has gone
**up**, from 19 open issues to 28.

---

## 1. Nine PR titles said "closes #N". Zero issues were closed. The sprint goal's one countable clause is unmet as a matter of record, and the rule T13 adopted governs the wording rather than the act

**QA (the measurement, from the live issue list rather than any PR body).**
`list_issues(state: OPEN)` returns **28** issues. T12 ended with 19 open. T13's
Ceremony 1 opened #154 and execution opened eight more (#156, #157, #158, #160,
#164, #165, #167, #168). 19 + 9 = 28, exactly — which is arithmetic proof that
**nothing was closed**, before checking a single issue individually.

Checked individually anyway, because a headline finding should not rest on a
count:

| Issue | Claimed closed by | Actual state | Evidence |
|---|---|---|---|
| #123 | T13.8 / PR #172 title "(closes #123)" | **open** | `closed_by_pull_requests: {total_count: 0}`; `updated_at` == `created_at` == `2026-08-14T16:11:46Z` — untouched since T12's Ceremony 1 opened it |
| #146, #152 | T13.2 / PR #166 | **open** | #146's only comment is from T12 (`2026-08-14T18:04:08Z`); merge commit `4b12aa4`'s body opens *"Closes #146 and #152."* |
| #154 | T13.3 / PR #170 | **open** | PR #170's review states *"This closes issue #154, the second and last instance of the subject-vs-uuid bug class"* |
| #138, #129 | T13.4 / PR #162 | **open** | — |
| #136, #135 | T13.5 / PR #163 | **open** | — |
| #148 | T13.7 / PR #169 | **open** | — |

**PO (why this is a process failure and not a clerical one).**
`sprint-process.md` does not merely omit a reminder — it documents this exact
mechanism at length, twice, in bold, because it has bitten this project before:

> *"Closing an issue is always a manual step on this project, never
> automatic."* … *"Writing `Closes #N` in the PR body is still good practice
> for human/PR-review legibility, but **it is not sufficient by itself and must
> not be treated as satisfying this step**."*

That text exists because the auto-close mechanism structurally cannot fire on
this branch topology (every PR merges into `claude/go-backend-pickleball-7up34j`,
not the default branch), and because it silently never fired for any ticket from
T5 through T10 until T11.8 fixed the backlog retroactively (issue #111,
`docs/process/t10-retro.md` finding 6). **T13 is the same failure, two sprints
after it was diagnosed, documented, and retroactively repaired.**

**BA (the sharpest part, and it is genuinely uncomfortable).** T13 adopted A5
specifically to make PR titles honest about closure — *"a PR that fixes part of
an issue says 'partial fix for #N', never 'closes #N'"* — and **A5 was followed
perfectly.** PR #171 is titled "partial fix for #147"; PR #159 is titled
"partial fix for #131"; PR #170 correctly declined to claim #145. Every
judgement about *what to claim* was made correctly.

So the sprint got the hard, judgement-laden half right and dropped the trivial,
mechanical half entirely. That is diagnostic: **A5 governs the sentence; nothing
governs the API call.** The rule that was adopted operates on PR titles, which a
human writes while thinking about scope. Closing an issue is a separate action
after merge, at the moment attention has already moved to dispatching the next
wave — and every one of the nine reviews ends with the same phrase, *"Merging
per CLAUDE.md rule 9,"* and then moves on.

**PE (why the reviewer-side backstop did not catch it, when it caught
everything else).** T12's A6 added *"every review enumerates the issues it
**opened** for that PR's disclosures."* That rule was carried into T13 and
partly followed (finding 4). **Nobody added the symmetric half — the issues the
PR *closes*.** The review template this project has evolved has a slot for new
gaps and no slot for resolved ones, so a reviewer scanning their own checklist
finds nothing missing. The one review that did think about closure — PR #170's,
which states *"This closes issue #154"* — states it as an accomplished fact in
prose while the API call was never made. That is the failure mode in one line:
the record says the thing was done because someone wrote that it was done.

**PM (what it costs, concretely, and it is not bookkeeping hygiene).** T12's
retro's central product framing was that the difference between "authorization
is done" and "the mechanism is done, with 11 tracked exceptions" is the
difference between a roadmap that schedules #144/#147/#148/#149 and one that
never looks at them again. T13's plan (A1) then executed on that framing by
*ranking the eleven and taking eight*. The whole point was to make the number go
down.

Anyone opening this repository's issue list today sees **28 open issues,
including six the sprint fixed**, with #138 still claiming the auth spine runs
in no gate (it now runs in `make test-platform`), #123 still claiming booking
takes 7 positional parameters (it now takes a `ServiceOptions`), and #148 still
claiming `ConfirmOnlinePayment` has no owner check (it has one, with a
mutation-proven regression test). **The issue list now actively misdescribes the
codebase in six places.** A future ceremony that ranks the backlog off this list
— which is exactly what A1 did, and did well — will re-rank work that is
already done.

**Recorded disagreement — PE vs. PO on where the fix belongs.**
- **PE:** put it in the review, symmetric with A6: every review states the
  issues the PR closes and the reviewer performs the close before moving on. It
  is one line in a template that already exists and already works, and the
  reviewer is the only party with merge authority, so it is the only party that
  can act at the right moment.
- **PO:** that is the same single-point-of-attention dependency finding 6 of
  T12 flagged, and this sprint just demonstrated it failing 9 times out of 9
  while the *same* session performed nine flawless mutation checks. Attention is
  not the scarce resource — *sequence* is. Make it a Definition-of-Done gate at
  the sprint level: the sprint is not done until the issue list contains no
  issue whose fix merged this sprint. That is checkable in one API call by
  someone other than the merger, and it fails loudly rather than silently.
- **Partially resolved.** Both agree on PE's review line, adopted (T14
  recommendation 1). Both agree it is insufficient on its own, precisely
  because it failed silently here. PO's sprint-level gate is adopted **as
  well**, not instead — the two operate at different moments and this sprint is
  evidence that the per-PR moment is the one that gets skipped under dispatch
  pressure. Whether the sprint-level gate should block the retro or merely be
  reported in it is left open for T14's Ceremony 1.

**This finding warrants a `docs/LESSONS.md` entry** and gets one. It is a
documented rule, with a documented mechanism, and a documented prior incident,
re-violated in full — which is the definition of a lesson that has not landed.

## 2. T13.1 wrote five new test packages and T13.4 built the Docker-free gate, in the same wave — and the gate does not run them. The dependency-completeness check has a structural blind spot, one level out from the one it was built to close

**QA (the mechanism, measured directly rather than taken from #157).** Read the
merged `Makefile`:

```make
ci-checks: generate tidy lint test-domain test-platform vet-integration \
           test-tools generate-client lint-web test-web build-web
	go build ./...
```

Run here against the tip:

```
$ go list ./internal/.../domain/... ./internal/.../app/... | grep -c adapter
0
$ grep -rl "func Test" --include="*_test.go" internal/*/adapter/ | xargs -n1 dirname | sort -u | wc -l
22
```

`test-domain` matches domain+app only. `test-platform` matches
`internal/platform/...` only. `test-tools` matches `./tools/...`.
`vet-integration` **compiles** and does not run. There is no `go test ./...` in
`ci-checks`. **Twenty-two adapter packages' Docker-free tests are executed by no
gate any machine in this project can run.**

**PdE (why this is a finding about the plan and not about T13.4).** T13.4 did
exactly what it was ticketed to do, did it well, and was mutation-checked by
both implementer and reviewer — the reviewer short-circuited
`RequireUnaryInterceptor`'s enforcement check and confirmed `make test-domain`
stayed green (exit 0, 12 packages) while `make test-platform` genuinely failed.
The gate is real. Its *scope* was #138's scope, and #138 named
`internal/platform/**`.

The finding is that **T13.1 and T13.4 were dispatched into the same wave, and
nothing in the planning apparatus asked whether one's output was covered by the
other's gate.** T13.1's entire purpose (per T12 finding 2) was that five
cross-context adapter packages had no behavioural test, which is how #146
shipped. It delivered five test files, mutation-checked, into five of the 22
ungated packages. As #157 puts it:

> *"Without a gate that runs them, T13.1's deliverable is five more tests nobody
> watches — which is the failure mode its own instruction 5 was written to
> prevent, recurring one level out at the gate rather than at the test."*

The same applies to `internal/booking/adapter/identity/lookup_test.go` — the
216-line regression test PR #151 added for #146, and now the flipped end-to-end
test T13.2 built for #152. **The test that proves this sprint's headline bug
stays fixed runs in no gate a PR is judged by.**

**PE (the blind spot, stated precisely, because it generalises and this project
has a bad habit of adopting the same fix in three shapes).** A13 is a genuinely
good artifact and it did real work before dispatch (finding 6 scores it
positively twice). But read its own arrow 7:

> | 7 | **T13.4 → (all)** | `internal/platform/**` covered by a Docker-free gate | Nothing consumes it; it is a gate, not a capability | ✅ Complete — no arrow, listed for completeness |

That is the miss, and the reasoning inside it is what makes it structural. The
check is defined over **capabilities that a downstream ticket's AC consumes**.
It asks: *does every capability on a consumer's side have a producer?* T12's
collision was a capability with two consumers and no producer, and the check
catches that class perfectly.

T13's is the **dual**: a ticket that produces a *coverage set*, where the thing
that should have been checked is not "who consumes T13.4" but "what does
T13.4's output have to cover, including artifacts its own sprint siblings are
creating in parallel." A gate is not consumed by a ticket; it is applied to the
repository — and the repository is changing underneath it during the same wave.
No question phrased in terms of consumption reaches that, exactly as no question
phrased in terms of file overlap reached T12's.

**Worth stating that this is the third consecutive sprint of the same family,
each fix closing the instance named and leaving the next glob gap:**

| Sprint | The ungated code | The fix | What it left |
|---|---|---|---|
| T11 (finding 2) | `//go:build integration` files, invisible to every runnable gate | T12.1's `vet-integration` (compiles them) | still never *executed* without Docker |
| T12 (#138) | `internal/platform/**`, incl. the auth spine | T13.4's `test-platform` | the 22 adapter packages |
| T13 (#157) | 22 `internal/*/adapter/*` packages, incl. T13.1's five new ones | — | open |

Three sprints, three globs, one class. The pattern is that each ticket is
scoped to the gap *named in its issue*, and the issue names the gap someone
happened to trip over. Nothing has ever asked the general question: **which
packages holding tests are executed by no gate?** That is one `go list`
comparison, and it would have produced #138 and #157 simultaneously, two sprints
ago.

**Recorded disagreement — QA vs. PE on how to fix the check.**
- **QA:** add a third question to the dispatch-isolation section, symmetric with
  the existing two file-overlap questions and A13's capability question: *for
  every gate, glob, or shared coverage artifact a ticket produces, which other
  in-flight tickets' outputs must it cover?* It is cheap and it is exactly the
  question that was missing.
- **PE:** that adds a fourth question to a section that already has three, and
  the T13 evidence is that questions get answered well and *still* miss the
  category nobody framed. Prefer the mechanical version: a standing
  `make gate-coverage` check that lists every package containing a `func Test`
  and every package any gate executes, and diffs them. It cannot be
  forgotten, it is re-runnable, and it would have caught all three rows of the
  table above without anyone predicting the category.
- **Unresolved, and both are adopted for T14 as a pair** — PE's mechanical check
  is the durable fix and is folded into recommendation 2 (it is a natural part
  of taking #157); QA's question is adopted for the one sprint it takes to build
  PE's check, since the check does not exist yet and T14 will dispatch parallel
  tickets before it does. Both agree that if the mechanical check lands, the
  fourth planning question should be *dropped* rather than accumulated — this
  project's recurring failure is adopting the same fix in two shapes.

## 3. The Wave-1.5 checkpoint held, and the ancestry evidence is stronger than the plan asked for — but PdE's objection was about cost under a second review loop, and no second loop occurred, so the cost is untested

**PE (that it held, established structurally rather than by timestamp).** A11
required T13.2 **merged and reviewed** — not merely started — before Wave 2
dispatched at all. Timestamps are consistent with that (T13.2/#166 merged
05:43:02Z; the earliest Wave-2 PR, #169, was created 05:57:15Z), but PR-creation
time is an *upper* bound on dispatch time and therefore cannot prove it. Checked
the stronger property instead — whether the Wave-2 branches actually descend
from the checkpoint's merge commit:

```
$ git branch -r --contains 4b12aa4        # 4b12aa4 = T13.2's merge (PR #166)
  origin/feature/t13.3-facilities-owner-seam
  origin/feature/t13.6-roster-read-authz
  origin/feature/t13.7-confirm-payment-owner-check
  origin/feature/t13.8-service-options
$ git branch -r --contains c8ec931        # c8ec931 = T13.6's merge (PR #171)
  origin/feature/t13.8-service-options
```

All three Wave-2 branches, and the Wave-3 branch, are descendants of the merged
checkpoint; T13.8 additionally descends from T13.6's merge, as Wave 3 required.
The consumers did not merely *start later than* the decision — **they were built
on top of it.** That is the strongest form of the property A11 wanted and it is
verifiable by anyone, permanently, without reference to any session's account of
what it dispatched when.

**PdE (that the decision it delivered was the one four tickets needed —
verified by reading it, not by trusting the PR).** A13 GAP 2 assigned to T13.2 a
capability with three consumers and no producer: *"which identifier space do
stored actor facts hold, per context."* ADR-0014 §5 delivers a per-context
table for all six contexts, and §5a is the part that earns the checkpoint:

> **"Ruling for T13.6 and T13.7: compare the value `actor(ctx)` returns against
> the stored value, unchanged. Do NOT add a resolution step to these three
> contexts in T13."** … *"adding one would make the actor a uuid on one side of
> a comparison whose other side is a subject — turning a working authorization
> check into one that silently denies everybody, or (worse, on a check written
> the other way round) silently permits."*

That is a **negative** instruction, and negative instructions are precisely what
a checkpoint buys and parallel dispatch cannot. Four implementers inferring the
ruling independently would each have had to reason their way to "do nothing
here," from a sprint whose entire theme was "fix the subject-vs-uuid seam." The
plausible independent conclusion was the wrong one. Both reviews confirm the
ruling was applied: PR #171's records *"no Identity port added to either
context, `actorUserID` vs `HostID` compared unchanged as subjects, no resolution
step."*

**PO/PdE — scoring A17's second recorded disagreement, and declining to score
it the way the outcome invites.** A17 recorded PdE's objection in specific,
falsifiable terms:

> *"T13.2 is 8 points and the largest single ticket; putting four tickets behind
> its **merge** (not its push) means Wave 2 cannot start until review completes.
> **If T13.2 takes two review loops, half the sprint idles.**"*

Measured: T13.2's PR opened 05:37:48Z and merged 05:43:02Z — **5 minutes 14
seconds, one loop, no corrections requested** (PR #166's review: *"No
corrections needed. This is a well-built checkpoint."*). The gap between the
checkpoint merging and the first Wave-2 PR opening was 14 minutes 13 seconds.
The A17 fallback — release Wave 2 on the ADR alone if the code needs another
loop — was never invoked.

**So the honest score is split, and PO/PdE agree on the split even though it
denies PO a clean win:**

- **On outcome, PO was right and it is not close.** The checkpoint delivered a
  negative ruling three tickets needed, in a sprint that ran faster than any
  before it. The trade PO argued for — *"a serialised sprint is recoverable; a
  wrong ownership comparison shipped four times is not"* — was bought at a
  measured cost of about fourteen minutes.
- **On the objection PdE actually raised, there is no evidence at all.** PdE's
  claim was conditional on a second review loop, and no ticket in this sprint
  took a second loop. A sprint in which nothing went slowly does not test a
  prediction about what happens when something goes slowly, exactly as T12's
  sprint with no silent agent did not test the roll-call (finding 6a). Recording
  "the Wave-1.5 checkpoint is cheap" on this evidence would be **CLAUDE.md rule
  10 applied to process and then ignored** — one successful run, treated as
  proof of reliability.

**Adopted for T14: keep the checkpoint on its narrow condition — a new
cross-cutting decision with three or more first-time consumers — and do not
generalise it**, which remains the over-correction PE correctly resisted in T12
and which nothing here justifies. **PdE's cost objection is carried forward
unresolved, with a scoring condition rather than a re-litigation:** the next
sprint that applies a Wave-1.5 checkpoint *and* has its checkpoint ticket take a
second loop scores it. If several sprints pass with every checkpoint merging
first-loop, that is itself the answer — the risk PdE priced is not
materialising — and it should be closed then, on that reasoning, rather than
re-recorded indefinitely.

## 4. All eight execution-opened issues were opened by implementers before their own PRs — the exact inverse of T12, resolving the PO/BA disagreement in PO's favour. BA's half recurred anyway, in the precise place BA predicted

**BA (the measurement, reconstructed from `created_at` fields on both sides).**
T12 finding 6 recorded that 18 of 19 issues were opened by the *reviewing*
session rather than by the disclosing PR, and left a PO/BA disagreement about
which party should be primary. A6 carried it here explicitly. Measured:

| Issue | Opened | Ticket's PR opened | Gap | Opened by |
|---|---|---|---|---|
| #156 | 05:27:50 | T13.1 / #161 at 05:31:22 | −3m32s | implementer |
| #157 | 05:29:17 | T13.1 / #161 at 05:31:22 | −2m05s | implementer |
| #158 | 05:29:42 | T13.9 / #159 at 05:30:54 | −1m12s | implementer |
| #160 | 05:30:56 | T13.5 / #163 at 05:32:57 | −2m01s | implementer |
| #164 | 05:36:02 | T13.2 / #166 at 05:37:48 | −1m46s | implementer |
| #165 | 05:36:14 | T13.2 / #166 at 05:37:48 | −1m34s | implementer |
| #167 | 05:56:13 | T13.7 / #169 at 05:57:15 | −1m02s | implementer |
| #168 | 05:57:03 | T13.6 / #171 at 05:59:11 | −2m08s | implementer |

**Eight of eight, every one opened before its own PR existed.** T12 was one of
nineteen. That is a complete inversion in a single sprint, and it is the
strongest evidence either side of that disagreement has ever had.

**PO (scoring it, since this was PO's position).** A5's literal text — *"the
same PR or review that records the decision creates the durable record"* — is
now descriptive of 8 of 8 rather than 1 of 19. PO's argument was that *the
implementer knows the gap best at the moment of discovery*, and the quality of
what was produced supports it strongly: #157 enumerates all 20 packages, runs
and quotes the two `go list`/`grep` commands, explains why T13.1's own scope
forbade fixing it, and proposes two concrete `Makefile` shapes with the
trade-off between them stated. #165 verifies the violation is pre-existing by
`git stash push -u` and re-running `gofmt -l`. These are better-specified than
a reviewer working across nine PRs would plausibly produce.

**Score: resolved in PO's favour. A5 keeps its text; the reviewer stays a
backstop rather than the primary.** Recorded honestly: the mechanism that
produced this is not identified. Nothing in T13's plan instructed implementers
to open issues *before* opening their PRs — A6 addressed the reviewer's
obligation, not the implementer's. The inversion may be a consequence of A6
making disclosure legible, or of T12.8's example propagating, or of ticket
Instructions carrying "open an issue" as an explicit numbered step (T13.6's item
5, T13.9's item 4) where T12's mostly did not. **T14 should not assume this is
now self-sustaining** — it is one sprint, and the previous sprint went the other
way just as decisively.

**QA (BA's half, which did not survive — and it failed exactly where BA said it
would).** A6 added, on the strength of T12's four unlabelled implementer-opened
issues:

> *"the enumeration must include labels. Four T12 issues (#146–#149) carry none,
> and they are the four an implementer opened — evidence that the labelling has
> been riding on the same single session's attention."*

Checked against the eight:

| Issue | Labels | Conformant to `sprint-process.md`'s taxonomy? |
|---|---|---|
| #156, #157, #160, #164 | `role:principal-engineer`, `type:chore` | ✅ |
| #158 | `type:story`, `role:principal-engineer` | ✅ |
| #165 | `type:chore` | ⚠️ no `role:` label |
| #167 | **none** | ❌ |
| #168 | `type:tech-debt`, `context:socialplay`, `context:payments` | ❌ **three labels, none in the taxonomy** |

The taxonomy is closed and stated: `type:story|bug|chore|spike` and seven
`role:*` values. `type:tech-debt` and `context:*` are inventions. They are not
*bad* labels — `context:*` is arguably a useful axis this project lacks — but
they were created by a single ticket mid-sprint without a decision, and
`sprint-process.md`'s note that "GitHub auto-creates a label the first time it's
applied" means an invented label is indistinguishable from a sanctioned one
afterwards.

**So the split is clean and both halves are real: who opens the issue moved
decisively to the implementer and the issues got better; the labelling moved
with it and got worse.** Both #167 (unlabelled) and #168 (off-taxonomy) came
from the sprint's last pair of tickets, and no review enumerated them against
the taxonomy.

**A6's own rule, scored while we are here.** *"Every review enumerates the
issues it opened for that PR's disclosures, or states explicitly that there were
none."* Checked all nine reviews: #161 names #156/#157; #159 names #158; #163
names #160; #171 names #168; #170 states the position on #145/#154 explicitly.
**#166's review names neither #164 nor #165 as issues that PR opened** (it
mentions #165 only incidentally, as a pre-existing `gofmt` hit), and #162, #169
and #172 make no enumeration statement at all. **Five of nine.** The rule
half-took — better than nothing, not yet reflexive, and it is the same review
template that finding 1 shows has no slot for closures either.

## 5. A two-line `gofmt` fix survived nine PRs and three reviews that each re-observed it, because every ticket's scope discipline was correct and `make ci` has no formatting gate at all

**QA (the state, re-derived at the tip).**

```
$ gofmt -l ./internal ./cmd
internal/socialplay/domain/game.go
$ gofmt -d internal/socialplay/domain/game.go | wc -l
14
```

The entire diff is struct-field alignment:

```diff
 type Game struct {
-	ID         string
-	HostID     string
+	ID     string
+	HostID string
```

Two lines. Opened as **#165** at 05:36:14Z during T13.2, with the violation
proven pre-existing by `git stash push -u` and a re-run. Five PRs merged after
it (#166, #169, #170, #171, #172). The reviews of **#166, #171 and #172** each
independently re-observed the identical line and cited #165 by number. It is
still there.

**PE (why, and this is the part that makes it a finding rather than a chore).**
The obvious reading is that nobody could be bothered. That reading is wrong, and
the evidence contradicts it in each ticket's own text:

- **T13.1** was scoped test-only, in writing: *"no change to any adapter's
  production code — if a test cannot be written without changing production
  code, that is a finding to disclose, not a licence to refactor."*
- **T13.2** declined for a reason #165's own body states: *"touching another
  context's domain file inside a Booking PR is exactly the scope bleed reviews
  keep catching."*
- Every other ticket had an equally clean reason: a Payments authorization
  ticket has no business editing `socialplay/domain/game.go`.

**Every one of those refusals was correct.** This is scope discipline — a thing
this project has spent six sprints building and which finding 3 and finding 4
both credit — producing an outcome nobody wants. The rules that stop a
constructor refactor from hiding inside an auth diff also stop a two-line
formatting fix from happening anywhere, because **no ticket owns the repository
as a whole**, and there is no janitor lane.

**The second, larger half — and it connects directly to finding 2.** Checked:

```
$ grep -n "gofmt" Makefile Jenkinsfile
(no matches)
```

`make ci` has **no formatting gate**. `golangci-lint run ./...` reports 0 issues,
so whatever the deviation is, no configured linter flags it. #165 spotted this
itself: *"Worth checking at the same time whether the CI pipeline has a
formatting gate at all — if `make ci` had one, this would not have reached the
shared branch."*

So `gofmt -l` in this project is a **review-prose convention**: every implementer
and reviewer is asked to run it and report the result, and nine of them did. It
is not a gate. That is exactly finding 2's shape — *the review checks by hand
what the gate does not check at all* — and the two findings should be fixed
together, in the same `Makefile`, by the same ticket.

**QA (the cost, which #165 predicted before it was paid).**

> *"every implementer from here on will see a red line that is not theirs, has
> to re-derive that it is pre-existing, and then decide whether to mention it.
> That is a small tax paid repeatedly, and **the second time somebody silently
> ignores a `gofmt -l` line is the time a real one gets ignored too**."*

Three reviews have now consciously ignored a `gofmt -l` line, each correctly.
The habit being trained is "the `gofmt` line is noise," and the next violation
will be introduced by a PR rather than inherited by it. Nothing in the current
setup distinguishes those two cases for a reader.

**Change for T14, adopted:** add a `gofmt`/format check to `make ci-checks`, and
fix #165 in the same PR — the gate and the one violation it would fire on,
together, so the gate lands green. Folded into recommendation 3.

## 6. The four carried-forward questions, scored — none deferred

T12's retro carried two questions forward explicitly (recommendation 10), and
T13's own plan carried two disagreements forward in A17. All four are scored
here. A10 made one of them binding: *"T13's Ceremony 3 must close A9(a) one of
three ways and may not defer it a fourth time."*

### (a) A9(a) — the wave roll-call vs. polling. **Closed as unfalsifiable in practice.**

A10 specified three exits and bound this ceremony to one of them:

> *If T13 had a silent agent the roll-call caught → closed, roll-call
> sufficient. If T13 had a silent agent the roll-call missed → polling is
> adopted. If T13 ran unbroken with no silent agent → **closed as unfalsifiable
> in practice**, and struck from the carried-forward list.*

Measured: **9 tickets dispatched, 9 PRs opened**, one per ticket, every one
merged. The sprint ran in a single block, 05:16:29Z → 06:19:55Z, 1h03m26s, with
no overnight gap. The third condition fires exactly.

**Closed. Struck from the carried-forward list. Polling is not adopted, and the
roll-call remains standing practice** — it costs nothing and this ceremony is
not proposing to remove a cheap safety net on the grounds that it has not been
needed.

**PdE, recording the reasoning honestly rather than declaring victory.** This is
now the third consecutive sprint in which the detector was not tested: T11
produced the failure, T12 and T13 produced no instance. The condition that
caused T11's incident — a work block spanning ~23 hours, with agents unreachable
by the time anyone looked — has not recurred since sprints began running in a
single sitting. So the honest statement is not "the roll-call works" but **"the
thing the roll-call detects stopped happening, for a reason that is probably
sprint duration rather than the roll-call itself."**

That gives a natural reopening trigger, which is recorded here in place of a
fourth deferral: **if a future sprint's work spans more than one work block, the
question is live again on its own terms** — not as a carried-forward item
someone must remember, but as a condition anyone can recognise when it occurs.
`docs/LESSONS.md`'s T11 entry already carries the mechanism and the specific
observation that polling and remote-branch listing would *not* have caught
T11.4, whose branch was never pushed. That entry is the durable record; this
question does not need to be a standing agenda item alongside it.

### (b) QA's port-contract-change rule. **Closed in PE's favour — do not adopt it.**

Finding 2 of T12 left open whether a plan-level rule was needed on top of the
dependency-completeness check: *a ticket that changes an identifier's meaning or
a port's contract must enumerate every adapter on that seam in its PR body.* A10
made T13 the test, correctly noting T13.2 *is* a port-contract change of exactly
the shape that caused #146, and instructed this ceremony to score it against
T13.2 and T13.3 specifically.

**The change was real.** Verified in `internal/booking/port/identity_lookup.go`:
`UserIDBySubject` at line 81, with the port's own doc comment at line 41 stating
*"UserIDBySubject was added in T13.2. Adding it was a real change to this
port"* — the port had deliberately exposed no method returning a `User` or its
ID. This is the genuine article, not a near-miss.

**Both halves of what QA's rule would have produced were produced by mechanisms
already in place, and they are different mechanisms:**

1. **The semantic half — which identifier space each context's stored actor
   facts hold — was caught by A13's dependency-completeness check, before
   dispatch.** GAP 2 named it as a capability with three consumers (T13.3,
   T13.6, T13.7) and no producer, and assigned it to T13.2, whose AC was widened
   to rule for all six contexts. It was delivered as ADR-0014 §5/§5a. This is
   the check catching precisely the shape QA was worried about — *"a wrong guess
   is a silently-broken authorization check"* — one ceremony before any code
   was written.
2. **The mechanical half — every adapter on the seam — was caught by the
   compiler**, at the reviewer's test-merge. PR #166's review records it:

   > *"The two adapter test files T13.1 also touched (`socialplay`/
   > `competitions` `adapter/booking/reservation_test.go`) needed T13.2's new
   > `UserIDBySubject` stub method — confirmed this is a legitimate compile-time
   > consequence of the port contract change, not a content collision."*

   A Go interface gaining a method makes every implementation fail to compile.
   Enumerating them in a PR body would have produced the same list the build
   produced, later and by hand.

**Score: PE was right. Do not adopt a separate port-contract-change rule.** PE's
stated reason in T12 — *"this project has a bad habit of adopting the same fix
twice in two shapes and then discovering a third shape"* — is borne out: the
rule would have duplicated A13's check on the semantic axis and the compiler on
the mechanical one.

**QA's residue, recorded rather than dismissed, because it is real and is now
tracked.** The compiler covers seams that are *Go-visible*. A contract change
that is **semantic only** — same signature, different meaning — is invisible to
it, and that is precisely the class ADR-0014 §5a legislated against by hand for
the three text-column contexts. Those three are non-conformant today and their
comparisons are correct only *by coincidence* (both sides come from
`actor(ctx)`, so both hold subjects). The ADR says so plainly: *"the checks are
correct today by that coincidence."* That residual is tracked as **#164**, needs
a data backfill rather than a code change, and is the one place where QA's
instinct still has force. **It is a ticket, not a rule** — which is the right
shape for it.

### (c) A17's roster-authz scope (Host-only vs. entitled-set). **Resolved in PE's favour.**

QA's position was that #147's entitled set is *Host + assigned Game Admins*,
that `assigned_game_admin_user_ids` is caller-supplied and persisted nowhere,
and that shipping Host-only as if it closed #147 would overclaim. PE's was that
the narrower behaviour is strictly better than the status quo (today *anyone
holding an id* reads the roster) and that a durable Game-Admin store is its own
domain + migration + proto cycle.

**Scored on what shipped:**

- Host-only shipped, and the reviewer's own mutation check confirms it bites:
  replacing `EnsureHost` in `ListRegistrationsForGame` with a no-op produced 2
  real failures in `TestListRegistrationsForGame_NonHostPrincipalIsPermissionDenied`.
- **QA's overclaim concern was fully honoured.** PR #171 is titled *"partial fix
  for #147"*, and the review records the reasoning: *"widening would have been
  authorization theater… correctly disclosed as narrower than #147's full ask,
  hence 'partial fix' not 'closes'."* This is the cleanest instance in the sprint
  of an adopted rule (A5) doing exactly the job it was adopted for.
- **QA's substantive blocker was converted into a tracked item**, as A17's
  instruction required: **#168**, "Game-Admin (and Competition-Admin) assignment
  has no durable store, so no authorization rule can include admins."

**PdE adds a reason neither role anticipated, which strengthens PE's side.**
ADR-0014 §5a independently established that Social Play and Competitions must
*not* gain a resolution step this sprint, because both sides of their ownership
comparisons come from `actor(ctx)`. A wider entitled-set check reaching into a
persisted admin list would have collided with exactly that constraint. So the
narrow scope was correct for a second, independent reason that only became
visible once the checkpoint's ADR existed.

**The disagreement's substance survives its resolution, and should not be filed
as settled:** Host-only *is* a different product behaviour than #147 asks for.
QA was right about that and remains right. It is now #168's problem, and #168
should be ranked alongside #144 at T14's Ceremony 1 (recommendation 5).

### (d) A17's Wave-1.5 cost trade-off. **Split: resolved in PO's favour on outcome; PdE's actual objection untested and carried forward with a scoring condition.**

Scored in full in **finding 3**. In brief: the checkpoint held in its strongest
form (all Wave-2/3 branches descend from the checkpoint's merge commit), it
delivered a negative ruling three tickets needed and would plausibly have got
wrong independently, and it cost about fourteen minutes. But PdE's objection was
explicitly conditional on *"if T13.2 takes two review loops"*, and T13.2 took one
loop and merged in 5m14s. **No sprint in which nothing went slowly tests a
prediction about what happens when something goes slowly.** Carried forward with
a concrete scoring condition rather than re-litigated — see finding 3.

---

## Recommendations for T14's Ceremony 1 and 2

Concrete and mechanical, in the spirit of T11 retro finding 2 → T12.1 and T12
retro recommendation 1 → T13's A13.

1. **Close the nine issues T13's code resolved, as T14 Ceremony 1's first
   act, before ranking anything** (finding 1) — #123, #129, #135, #136, #138,
   #146, #148, #152, #154 — each with `state: closed`,
   `state_reason: completed`, and a comment naming the merged PR, per
   `sprint-process.md` DoD step 5. Then re-check #147 and #131 are correctly
   *left* open as partial fixes. **Ranking the backlog off the current issue
   list without doing this first will re-rank six finished pieces of work.**
   Two durable fixes, adopted together because this sprint proved either alone
   is insufficient: (i) **every review states the issues the PR closes, and the
   reviewer performs the close before moving to the next ticket** — the
   symmetric half of A6, which has only ever had a slot for issues *opened*;
   (ii) **a sprint-level DoD check that no issue remains open whose fix merged
   this sprint**, verifiable in one API call by a party other than the merger.
2. **Take #157, and fold #138's shape into it rather than closing another
   single glob** (finding 2). This is the highest-value item here: it is the
   third consecutive sprint of the same class, and the two prior fixes each
   closed the instance named and left the next gap. Build PE's mechanical
   version — a check that lists every package containing a `func Test` and every
   package the gates actually execute, and diffs them — so the *general*
   question is answered once instead of the specific one being answered a fourth
   time. Note #157's own warning that this is **not** a one-line glob widening:
   the `adapter/postgres` and `adapter/grpcapi` packages need `internal/gen`, so
   they cannot fold into `test-domain` without breaking its documented
   dependency-free contract.
3. **Add a formatting gate to `make ci-checks` and fix #165 in the same PR**
   (finding 5). Two lines of struct alignment have now survived nine PRs and
   three reviews that each re-observed them, because `gofmt -l` is a review
   convention and not a gate. Land the gate together with the one violation it
   fires on, so it goes green immediately. Same `Makefile` and same class as
   recommendation 2 — reasonable to combine into one ticket.
4. **Extend the dependency-completeness check with the dual question, for one
   sprint only** (finding 2): *for every gate, glob, or shared coverage artifact
   a ticket produces, which other in-flight tickets' outputs must it cover?*
   A13's check is defined over capabilities a downstream AC **consumes**, which
   structurally cannot see a gate whose coverage set its own sprint siblings are
   changing in parallel. **Drop this question once recommendation 2's mechanical
   check exists** — do not accumulate both, which is this project's recurring
   failure mode and one PE named in advance.
5. **Rank #144 and #168 first among the authorization backlog** (finding 6c and
   T13's own deferral note). #144 (`CancelBooking`/`CreateBooking` have no
   authorization check — *anyone holding a booking id can cancel it*) is the
   sharpest remaining BOLA hole and T13's plan already named it a strong T14
   candidate that "should be first"; it needs a product decision on what owns a
   booking made through the public quote-and-book flow, so **surface that
   decision at Ceremony 1 rather than discovering it mid-ticket**. #168 (no
   durable Game-Admin store) is the blocker under both #147's residue and #149,
   and #149's own text says it should probably be closed first.
6. **Decide the label taxonomy question rather than letting it drift**
   (finding 4). #167 carries no labels; #168 carries `type:tech-debt`,
   `context:socialplay`, `context:payments` — three labels, none in
   `sprint-process.md`'s stated taxonomy, auto-created by first use and now
   indistinguishable from sanctioned ones. Either adopt `context:*` deliberately
   (it is arguably a useful axis this repo lacks) and add `type:tech-debt` or
   map it to `type:chore`, or scrub them. Then make label conformance part of
   the review's issue enumeration, which A6 already asked for and which no
   review performed.
7. **Keep the Wave-1.5 checkpoint on its narrow condition — a new cross-cutting
   decision with three or more first-time consumers — and do not generalise it**
   (finding 3). Its cost is still untested: score PdE's objection at the next
   sprint whose checkpoint ticket takes a second review loop. If several sprints
   pass with every checkpoint merging first-loop, close it on that reasoning
   rather than carrying it indefinitely.
8. **Do not adopt QA's port-contract-change rule** (finding 6b). A13's check
   caught the semantic half before dispatch and the Go compiler caught the
   mechanical half at test-merge. Its one genuine residue — semantic-only
   contract drift, invisible to the compiler — is tracked as **#164** and is a
   ticket, not a rule.
9. **Correct `HANDOFF.md`'s T13 Docs-index row at Ceremony 1** (line 34 reads
   "not yet written" for the retro). This is the **first live test of the
   structural fix T13 itself adopted** — `sprint-process.md`'s new "Correct the
   previous sprint's Docs-index row" section, chosen over having retro PRs take
   the row because a retro PR cannot cite its own merge number before it exists.
   Report in T14's plan whether the amendment worked without prompting; if the
   row is still stale at T14's Ceremony 3, option (b) failed and option (a)
   should be reconsidered.
10. **State T13's outcome in the honest form when writing its Task-backlog
    narrative entry** (`sprint-process.md` Ceremony 1, item 3 — *"in the form
    its own retro agreed, not a stronger one"*). This retro's agreed form is in
    the sprint-goal register below. The engineering claim is strong and should
    not be undersold; the closure claim must not be made at all.

---

## The sprint goal, scored: what was proven, what shipped unproven, and what is still open

T13's goal was unusually specific and countable, which makes it scoreable rather
than a matter of framing. Taken clause by clause, against evidence gathered
here rather than from PR bodies.

> *"…the two seams where a verified subject reaches a `uuid` column are fixed
> under a single recorded decision, so `RequestRecurringHire` and
> `CreateFacility` work for a real caller for the first time"*

**Met, and independently verified.** ADR-0014 rules **translate, not widen**,
for all six contexts (§5), so both seams are fixed under one decision and
`0020` stays unclaimed (verified: `db/migrations/` ends at
`0019_identity_subject.sql`). Booking's `port.IdentityLookup` gained
`UserIDBySubject` (`identity_lookup.go:81`); Facilities gained its first-ever
cross-context port and adapter (`internal/facilities/port/identity_lookup.go`,
`internal/facilities/adapter/identity/`), which A13 GAP 1 assigned to exactly one
ticket before dispatch. PR #151's deliberately-inverted pinned test was flipped
rather than deleted, as its own comment specified — it is now
`TestRequestRecurringHire_SubjectActorResolvedAtTheSeamThenSucceeds`. Both fixes
were mutation-checked by the reviewer independently (bypassing each seam's
`actor()` funnel produced 21 and 17 real failures respectively).

**One honest qualification, stated because CLAUDE.md rule 10 asks for it:** "work
for a real caller" is proven against the real `identityapp.Service` over
in-memory fakes and the real gRPC handler stack, Docker-free. It is **not**
proven against Postgres or against a real IdP-issued token, because neither
exists in this environment. That is the same standard T12's auth work was held
to, and it should not be restated later as "verified in production."

> *"…the three RPCs that never had an authorization check
> (`ConfirmOnlinePayment`, and the two roster reads) get one"*

**Met.** `ConfirmOnlinePayment` compares the verified actor against a fact
Payments itself recorded, with `domain.ErrNotPaymentOwner`
(`internal/payments/domain/errors.go:73`) mapping to `PermissionDenied`; both
roster reads enforce Host-only. All three were mutation-checked by the reviewer,
not merely by the implementer. **Scope is narrower than the issues ask** and was
disclosed as such: roster reads are Host-only, not Host + Game Admins (#168),
and #149's fact-fabrication class is untouched — Payments still compares a
verified actor against caller-supplied `game_host_id`/`booking_host_id`/
`entrant_player_id`/admin lists.

> *"…the auth spine's own tests run in a gate a session can actually execute,
> and that gate reaches CI"*

**Met for the auth spine; the general claim it implies is false, and that is
finding 2.** `make test-platform` exists, runs green here (`auth`, `auth/rs256`,
`grpcrecovery`), and is wired into `ci-checks`; the `Jenkinsfile`'s four
hand-duplicated stages collapsed into one stage calling `make ci-checks`. The
gate was proven non-vacuous by the reviewer independently.

Two qualifications, both disclosed in-sprint and neither retracted here:
**(i)** "reaches CI" means the pipeline *definition* is correct. There is still
no Jenkins job, webhook, or branch protection, and no session can create them —
the same server-side gap standing since SCRUM-6. **(ii)** The gate covers
`internal/platform/**` and nothing else new: **22 adapter packages holding real
tests are executed by no gate**, including T13.1's five new ones and the
regression test for this sprint's own headline bug (#157).

> *"…and six of T12's eleven residual auth issues are closed rather than
> carried"*

**Not met as a matter of record — and this is the only clause that failed.** The
*engineering* for all six (#135, #136, #138, #146, #148, #152) is complete and
verified. The issues are all open, along with #123, #129 and #154. **Zero issues
were closed this sprint.** The open-issue count went from 19 to 28. See
finding 1.

**The agreed honest sentence, which T14's Ceremony 1 should carry into
`HANDOFF.md`'s Task-backlog entry** (`sprint-process.md` Ceremony 1 item 3
requires the retro's form, not a stronger one):

> T13 fixed both seams where a verified subject reached a `uuid` column, under
> one recorded decision (ADR-0014: translate at each context's grpcapi `actor()`
> funnel, never widen), so `RequestRecurringHire` and `CreateFacility` work for
> a real caller for the first time — proven by end-to-end tests against the real
> Identity service and by reviewer-performed mutation checks, though not against
> Postgres or a real IdP. Three RPCs that never had an authorization check now
> have one, deliberately narrower than their issues ask (#168, #149 remain).
> `internal/platform/**` is gated for the first time and the `Jenkinsfile` calls
> `make ci-checks` — though no Jenkins job exists to run it, and **22 adapter
> packages' tests are still executed by no gate (#157)**. All 9 tickets merged
> on their first loop with no defect reaching the shared branch, the first sprint
> in this project's history where that is true. **The six residual auth issues
> the sprint set out to close are fixed in code but were never closed on GitHub;
> the open-issue count rose from 19 to 28.** Nine issues await closure at T14's
> Ceremony 1.

**ADR-0012's Q1/Q2 remain blocked and correctly flagged.** Checked: no answer to
either escalated question exists, no `Gender` field, no `PlayerRating` type and
no Level-scoring formula was added by any T13 ticket, and A15 recorded the
constraint without making a product call. The two genuine interaction risks A15
named — T13.2 changing how a `User` is looked up, and T13.8 refactoring Social
Play's constructor — both landed without touching profile fields or rating
types. Neither silently dropped nor silently decided, for the fourth consecutive
sprint.

---

## No finding on

**No finding on the review record, beyond the credit it is due.** All nine PRs
carry a discoverable review; every review records an independent fresh-worktree
or test-merge verification, a reviewer-run full toolchain, and a mutation check
**the reviewer performed itself**. The coordinating session's own claim that this
was independent verification was spot-checked here rather than accepted, and it
holds: PR #166's review reports 6 failures where the PR claimed 4, PR #162's
review independently reproduced #138's exact claim by short-circuiting
`RequireUnaryInterceptor` and confirming `test-domain` stayed green while
`test-platform` failed, and PR #172's review hand-checked ~10 call-site
conversions beyond the PR's own table. **This ceremony was asked to check
whether "no PR needed a review-time fix" is a real quality signal or an artifact
of low-conflict-risk tickets. The answer is: partly both, and the honest split
matters.** The tickets were genuinely lower-collision by design — five of nine
were additive, the Wave-1.5 checkpoint existed specifically to prevent T12's
collision class, and A14 pre-assigned every shared file and the one conditional
migration number. But low conflict risk does not produce correct mutation
checks, correct negative rulings, or a reviewer catching more failures than the
implementer claimed. The design reduced the opportunity for defects; the
execution is why none appeared.

**No finding on T13.2 beyond findings 3 and 6.** ADR-0014 is the strongest
artifact this sprint produced: it rules for all six contexts including the three
needing no code change, states the checked negative that two downstream tickets
depended on, records why "just delete the guard" is wrong (§4), rejects the
widening alternative explicitly (§7), and has a "What this ADR does not decide"
section that names its own residue and files it as #164 rather than papering
over it. It also pins its own scope with a test —
`TestRequestRecurringHire_ResolvedActorStillFailsTheClubRoleCheck` — asserting
that resolving an actor does not grant it anything. That is an ADR that
anticipated being misread.

**No finding on T13.8, and its restraint is worth naming.** It re-derived both
constructor signatures from merged code rather than trusting the plan or #123's
counts (A13 GAP 3's instruction), and found that neither T13.2 nor T13.6 had
actually changed a signature — so GAP 3's hedge resolved to "no change," which
is the correct outcome for a hedge that was honest about not knowing. It reused
T13.5's `verifierIsNil` reflect-based typed-nil pattern from earlier in the same
sprint rather than inventing a second one, which is the *opposite* of T12
finding 1's collision: a later ticket consuming an earlier ticket's primitive
within one sprint.

**No finding on T13.9 or T13.7's file sharing.** A14 wave-separated them on
`payments/adapter/grpcapi/handler.go` and the separation held; T13.7 added its
`ErrNotPaymentOwner` mapping into T13.9's corrected `toStatus` rather than
reverting or duplicating it. T13.9 also declined to silently widen into Social
Play and opened #158 for the mirror-image inconsistency instead — the behaviour
A6 exists to produce, performed by the implementer without prompting.

**No finding on T13.5's disclosed cost, which was handled correctly.** Making a
nil verifier a startup failure genuinely breaks `make up` without `AUTH_*` env
vars. The ticket disclosed it, rejected an opt-out flag with a stated reason
(it re-creates the fail-open path the ticket exists to close), and opened
**#160** for a local-dev auth fixture rather than quietly degrading the
guarantee. A security fix that makes local development harder, disclosed with
its cost named and tracked, is the correct trade correctly recorded.

**PM had limited independent material this sprint**, as in T9–T12. PM's
substantive contribution is embedded in finding 1's cost analysis, which is a
product-framing judgement rather than an engineering one: an issue list that
misdescribes the codebase in six places is a roadmap input, and the ceremony
that ranks the backlog off it is three weeks of planning away from noticing.
