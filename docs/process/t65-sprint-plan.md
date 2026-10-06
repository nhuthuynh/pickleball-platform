# T65 Sprint Plan — Ceremony 1 (backlog refinement)

Per `docs/process/sprint-process.md`. PM + PE, held against `HANDOFF.md`,
`docs/process/t64-retro.md`, `docs/process/t64-sprint-plan.md`, the live issue
list, and the tree at `64ad1ce`.

Subject to **T62.2** (every quantity carries its command), **T63.1** (an
action-claim carries the command that re-reads live state), **T63.3** (a guard
is verified by removing it) and **T64.3** (a gate claim carries its date).

**Ceremony order:** security gate (§0, new) → escalations (§1) → bookkeeping
(§2) → issue sweep (§3) → tickets (§4).

---

## §0 — The security gate, run first

`docs/process/t64-retro.md` recommendation 1, applied on its first
opportunity. T64's ceremony found the gate red by running it before planning;
T61's and T62's recorded it as owed and planned anyway.

```
2026-10-06T11:10:54Z $ SKIP_GOVULNCHECK=1 make security | tail -1
PASS: no new gating findings (0 baselined, 2 below threshold).
```

**PASS, dated** — per T64.3's clause, which exists because this exact gate's
verdict changed in under a day. The two below-threshold findings are #320's
`vitest` pair and are T65.1.

**The Go half still has never run on this project.** `vuln.go.dev` is
`Forbidden` from here, and T64's review established the honest wording:
**"does not, as configured"** rather than "cannot" — whether the host can be
allowlisted needs the proxy's own status endpoint, which is denied to this
session and to its agents. Not re-tested here; carried in §5.

## §1 — Escalations

### ADRs: none escalated, from the listing rather than a scratch copy

**First real use of `cmd/docsindex -statuses`** (T64.5), which exists because
T64's own ceremony had to copy the package into a scratch directory to answer
this question without re-implementing the parser as a grep:

```
$ go run ./cmd/docsindex -statuses | tail -2
17 ADR(s): Accepted=17
```

Eleven `## Status` heading form, six front-matter. **Nothing escalated.** The
tool paid for itself on its first ceremony — one command instead of a
twelve-step workaround — which is the cheapest possible vindication of a
3-point ticket.

### Issues: every open one considered, and its **body** read

**First application of T64.5's prose step**, which exists because T62.4's
*corrected* sweep still missed a product question sitting in #145's body. The
step asks, per issue, whether an unanswered question is present — not whether
the issue's labels say so.

```
$ for n in 330 329 328 320 149 145 134; do gh api .../issues/$n --jq .body \
    | grep -inE "needs (product|a decision|input|sign-off)|open question|to be decided|awaiting|decide (whether|what|which)"; done
#328:40  "Then decide what to do about the blind spot, explicitly." Options, none free:
#145:68  ~~Needs product input on which.~~ **Answered at T64's Ceremony 1 …**
```

| Issue | Awaits a decision? |
|---|---|
| #330 | No. Carries three options and a stated preference; a ticket settles it. |
| #329 | No. Two options (fail the gate vs note it); a ticket settles it. |
| #328 | **An engineering decision, not a product one** — what `gate-coverage` should answer about packages with *zero* tests. T65.2 settles it. |
| #320 | No. The remedy is verified (§3); what remains is execution. |
| #149 | No. A design change (a read port into Booking), unbuilt rather than undecided. |
| #145 | **Answered at T64**, as a deferral with a trigger a future sprint can cause. The strike-through above is the step working. |
| #134 | No. Blocked on hardware. |

**No product decision awaits an answer.** Three issues carry *engineering*
options — which the step surfaces and the old label-and-state sweep would not
have — and each is assigned to the ticket that settles it rather than left as
an open question in a body.

**And the limit, which this ceremony can state first-hand.** All seven bodies
were written or corrected within the last 24 hours, by this session. The step
cost nothing here and proved nothing either: it was never going to surface a
question this session did not already know about. **Its first real test is a
ceremony reading an issue it did not write**, and that has not happened yet.

## §2 — Bookkeeping

1. **T64's Reviews cell filled**, `merged_at`-verified rather than assumed:

   ```
   $ gh api .../pulls/{325,326,327,331} --jq '"#\(.number) \(.merged_at) \(.merge_commit_sha[0:7])"'
   #325 2026-10-06T07:14:07Z 1febf8e    Ceremony 1
   #326 2026-10-06T07:44:55Z 2e9b59f    all five tickets
   #327 2026-10-06T08:20:45Z bdab076    retro
   #331 2026-10-06T08:32:09Z 64ad1ce    review fixes
   ```

   Its **Retro cell** was already set by the retro PR, per the path/number
   distinction — nothing to correct, the second sprint running.

2. **A T65 row created.**

3. **T64's narrative added**, with **53 blockquote lines lifted
   programmatically** from `t64-retro.md` §13 (3949 chars), not retyped.

4. **The "Open issues" section was corrected at T64's review and is left
   alone here.** It drifted twice in one day — re-derived at T64's Ceremony 1,
   wrong by that afternoon — and the honest reading is that its failure mode is
   structural: **it caches a derived result in prose.** T65.3 covers it.

## §3 — Issue sweep: both questions, every issue

`sprint-process.md`'s premise check. **Fired on zero of seven**, and every
premise was verified against the tree rather than read off the issue:

| Issue | Blocker | Premise, verified |
|---|---|---|
| #330 | none | **true.** Differencing `Makefile`'s `ci-checks` prerequisites against the Jenkinsfile comment: 17 prereqs, 11 named, **6 missing** (`fmt-check`, `test-adapters`, `test-cmd`, `gate-coverage`, `docs-index-check`, `lock-check`) |
| #329 | none | **true.** `0015` and `0016` still carry both status forms with disagreeing tokens |
| #328 | none | **true, and worse than filed.** `cmd/server` has 4 tests; `cmd/devtoken`, `cmd/docsindex`, `cmd/gatecoverage` and `cmd/vulngate` have **zero** — four of five |
| #320 | npm 10.9.7 locally (`npm -v`) | **true**, and its remedy is now verified: npm 12.2.0 resolves the bump in a scratch prefix with `found 0 vulnerabilities` |
| #149 | `internal/payments/port/` has no Booking reader | **true.** `grep -c "BookingHostID: req.GetBookingHostId()"` → 2 |
| #145 | real IdP | **true**, and its product question is answered |
| #134 | AT hardware | **true.** All three routes in `ROUTES_UNDER_TEST` at `:92`, `:98`, `:99` |

**Zero of seven is the expected outcome here and should not be read as the
check working**: every one of these issues was written or corrected inside the
last 24 hours, six of them by T64. A clean sweep over issues this fresh is
arithmetic, not evidence. The check's rate over its four real runs — T59, T62,
T63, T64 — stands at three of four.

### Compliance check: are T62.2/T63.1/T63.3/T64.3 being followed?

`t62-retro.md` recommendation 2 asks each Ceremony 1 to check this, and says
the only evidence that counts is **a sprint that did not write the rule.**

**That sprint still has not arrived**, and saying so is the honest answer.
Every sprint from T62 to T65 was written by one session; the rules' author has
audited the rules' application every time. What *can* be said is narrower and
real: **T64's review audited T64's artifacts against the four rules derived
rather than hand-listed, and every figure that failed was a figure carrying no
command** — eight of them, including two in documents that cite the rules. That
is evidence the rules **discriminate**: they are not satisfied by prose that
merely sounds careful.

**The open question this ceremony will not pretend to answer:** whether the
rules are followable by anyone other than their author. T65.5 does not attempt
it either — a rule's portability cannot be established by the party that wrote
it.

## §4 — Tickets

Five. Ordered by value below; the numbering follows discovery.

### T65.3 — The two documents every session reads first describe a T4-era prototype

**Story.** As a session starting work on this repository, I want the first two
documents I am told to read to describe the system that exists, so that I do
not begin from a four-sprint-old prototype's premises.

**Description.** `t64-retro.md` §8 and recommendation 5. `CLAUDE.md`'s "What
this is" says *"a runnable vertical slice through the **Booking** bounded
context"*; its "Current state" enumerates T0–T4 and ends *"Next phase: see
`HANDOFF.md` task backlog (T5 onward)"*. `HANDOFF.md`'s "Current state" stops
at T9. The tree:

```
$ ls internal/ | tr '\n' ' '
booking competitions facilities gen identity payments platform socialplay
$ ls db/migrations/*.sql | wc -l
33
$ grep -rh '^func Test' --include=*_test.go internal cmd tools | wc -l
1039
```

Six bounded contexts, 33 migrations, 1039 test functions, a working auth spine
and a green integration suite. **This is not a correction, which is why T64's
sweep did not do it**: the content is stale because of *where it lives*, and a
patch would be stale again in two sprints.

**Instructions.**
1. **`CLAUDE.md`'s "What this is"**: one sentence, derived — the contexts that
   exist, named. Keep it short enough that it stays true.
2. **Delete `CLAUDE.md`'s "Current state" section** and point at
   `HANDOFF.md`'s Docs index. The argument to state in the commit: a per-sprint
   status list inside a document that calls itself *the durable rulebook* is
   guaranteed to go stale, and the index next door is **derived, gated by
   `make docs-index-check`, and current to T65**. Removing the section removes
   the content *and the mechanism*.
3. **`HANDOFF.md`'s "Current state"**: cut the per-sprint prose blocks (each
   duplicates that sprint's retro, which the Docs index already links) down to
   a current statement of what exists and what does not, **re-derived against
   the tree** rather than carried. The "Not yet built" list must be re-verified
   per item — T64's review found "Auth" on it.
4. **State what is lost.** The prose blocks contain judgements that are not in
   the retros. Anything worth keeping moves to the retro it belongs to, or is
   quoted in the new section with its sprint named. **Do not delete a judgement
   silently to make a section shorter.**
5. **Do not add a gate for this.** `docs-index-check` covers the index and the
   narratives; extending it to prose about "current state" would need it to
   know what is true of the tree, which is the whole repository. Record the
   decision.

**NFRs.** `make docs-index-check` must stay green. No claim in either rewritten
section without a command or a path that supports it; where a claim is about
the environment rather than the tree, it carries the date it was tested.
**Points: 5.** `role:principal-engineer`, `type:chore`.

### T65.2 — #328: tests for `cmd/docsindex`, and decide what `gate-coverage` owes

**Story.** As a maintainer, I want the 52 lines T64.5 shipped untested to be
tested, and I want to know whether the project's standing answer to "what is
untested?" is supposed to cover `main` packages at all.

**Description.** #328, premise re-verified in §3 and **worse than filed**: four
of five `cmd/*` packages hold zero tests, so `gate-coverage: OK` is silent
about all of them by construction. Rules 1 and 8 were not met by T64.5.

**Instructions.**
1. Test `-statuses`: the listing's lines, the tally, and — most important —
   the **non-zero exit** on an unclassifiable status. A listing that exits 0
   while skipping an unreadable ADR is the defect T62.4 was written to stop.
2. **Mutation-verify each** (T63.3), quoting the failure.
3. **Then decide the blind spot explicitly, and record the decision**: leave it
   and say so in `CLAUDE.md`'s `gate-coverage` entry; extend the tool with a
   separate "zero-test package" report; or make `main`-package exemption an
   explicit convention. #328 names the trade-offs.
4. **Do not add an exclusion list to `tools/gatecoverage`.** Its doc comment
   forbids it and three sprints of history back that up.

**NFRs.** `make gate-coverage` must still pass; the new tests must be reachable
from `ci-checks` without widening any pattern by hand (they will be — `cmd/...`
is already a `test-cmd` pattern, which is itself worth confirming rather than
assuming).
**Points: 3.** `role:principal-engineer`, `type:chore`.

### T65.1 — #320: take the `vitest` bump, now that the remedy is verified

**Story.** As an operator, I want the last two advisories in the web client
closed, so that `make security` reports nothing rather than two
below-threshold findings.

**Description.** #320's preferred remedy — *"a newer npm; verify on npm ≥ 11"*
— was tested while T64's retro was being written and **works**: npm 12.2.0 in
a scratch prefix resolves the bump with `found 0 vulnerabilities`. What remains
unverified is the half that matters: **the 717 web tests on `vitest` 4.1.11.**

**Instructions.**
1. **Decide and record how a newer npm enters this environment** before
   touching the lockfile — a pinned `npx npm@12`, a scratch prefix on `PATH`,
   or a global install. `make lock-check` runs `npm ci --dry-run` with whatever
   npm is on `PATH`, so **two npm versions can produce two lockfiles**; the
   project must target one. Verify, do not assume.
2. Bump `vitest` to the first fixed version in its current major line
   (`^4.1.11`), with `@vitest/mocker` overridden if it does not follow.
3. **Mutation-verify by reverting the lockfile** (T64.3's clause), showing both
   advisories returning by name.
4. Run `make test-web` and `make build-web` and report both. **If a web test
   breaks, stop and report — do not fix the test in this PR.** Standing
   instruction from T63.2; this project has one recorded instance of a suite
   asserting a defect as correct (`t58-retro.md` §3).
5. Re-run `SKIP_GOVULNCHECK=1 make security` **with its date**.

**NFRs.** 61 files / 717 tests stay green. No baseline entry. The lockfile is
regenerated by the npm the project decided on in step 1, not by whichever was
handy.
**Points: 3.** `role:principal-engineer`, `type:bug`.

### T65.4 — #329 and #330: two lists that are right by luck

**Story.** As a ceremony, I want the ADR status parser and the Jenkinsfile's
account of `ci-checks` to be right by construction rather than by convention.

**Description.** Both are "a hand-maintained or convention-dependent answer
that happens to be correct today". #329: two of seventeen ADRs carry **both**
status forms with disagreeing tokens, and `adrStatus` takes front-matter
unconditionally and silently — right because of which form this project
happens to use for the live value. #330: the Jenkinsfile's comment omits **6 of
17** prerequisites (§3), in the comment whose own argument is that such lists
drift.

**Instructions.**
1. **#329:** when both forms are present, surface it. Prefer the **note** form
   over failing the gate, since both current instances are benign and a red
   gate on the shared branch blocks every PR — but say which you chose and why.
   Add the both-forms fixture T64.5's tests lack. The fenced-code-block variant
   is the same fix region; cover it or record why not.
2. **#330:** **delete the prose list** rather than re-typing it, and have the
   pipeline print the real one (`make -n ci-checks`, or a small `print-ci-checks`
   target). A list that cannot go stale beats a list that is currently correct.
3. Record the finding that `lock-check` **can never fire in the real pipeline**
   (`npm ci` runs two stages earlier), and either reorder the stages or say
   plainly in `CLAUDE.md` that it is a local-developer gate. It currently reads
   as though CI is where it earns its keep.

**NFRs.** `docs-index-check` stays green — if #329 is implemented as a failure
rather than a note, the two existing ADRs must be normalised **in the same PR**,
because a rule that ships the gate red is worse than the gap.
**Points: 3.** `role:principal-engineer`, `type:chore`.

### T65.5 — The two process debts T64's retro left open

**Story.** As a future ceremony, I want the two mechanics T64 learned the hard
way written down, so the next session does not rediscover them.

**Description.** `t64-retro.md` recommendations 4 and 9, both deliberately left
to a sprint that did not discover them.

**Instructions.**
1. **The stale-read qualification on T63.1.** A live-state command run in the
   same breath as the action it verifies can return the **pre-action** state —
   T64 closed #322 and read the issue count as 5, then 4 moments later. One
   sentence: read the object rather than the list, or read twice. **Or decide
   against it and say why**: it is a habit, and this document already warns
   against rules that cannot be followed mechanically.
2. **The squash-ancestry procedure.** Three sprints running have hit it: every
   PR squash-merges, so a branch cut from the previous PR's tip shares no
   commit with the squashed result and conflicts on byte-identical content.
   The procedure is short — verify the base's file is byte-identical to the
   branch's pre-edit version (`git diff --stat <base> <branch-tip>`), then
   rebuild the change on the base rather than resolving markers by hand — and
   it currently lives only in two retros and a PR body.
3. Keep both short. `t56-retro.md`'s complaint about rules becoming wallpaper
   is the constraint; two paragraphs each is the budget.

**NFRs.** Prose only. Each clause cites the instance that produced it.
**Points: 2.** `role:principal-engineer`, `type:chore`.

### PM value ordering, if the sprint has to be cut

1. **T65.3** — every session pays for this one on arrival, and it is the only
   ticket whose cost is borne by *all future work* rather than by this sprint.
2. **T65.2** — rules 1 and 8 were not met last sprint; the longer that stands,
   the weaker both rules are.
3. **T65.1** — real advisories, but dev-only and below the gate's own
   threshold, with the remedy already verified.
4. **T65.4** — two latent correctness gaps in tooling the ceremony now depends
   on.
5. **T65.5** — valuable, cheap, and nothing degrades while it waits.

**PE sign-off, with two sequencing constraints.** T65.3 and T65.2 both touch
`CLAUDE.md`'s `gate-coverage` entry — T65.3 first, so T65.2's decision is
recorded in a section that has already been rewritten. T65.4 and T65.1 are
independent of everything.

**Dependency-completeness check** (both questions):

| Ticket | Producer exists? | Consumer can reach it? |
|---|---|---|
| T65.3 | `HANDOFF.md`'s Docs index is derived and gated | yes — the index is already the designated map, and `docs-index-check` covers it |
| T65.2 | `tools/docsindex`'s fixture helpers; `ADRStatuses` is exported | **to be confirmed by the ticket**: that `test-cmd`'s `./cmd/...` pattern picks up a new `cmd/docsindex` test without a Makefile edit. Stated as a step, not assumed |
| T65.1 | npm 12.2.0 verified resolvable in a scratch prefix | **the open question is the consumer**: which npm `make lock-check` and CI will run. Instruction 1, before any lockfile change |
| T65.4 | `adrStatus` is the single parser; `gatecoverage` already parses the Makefile | yes for #329; for #330 the pipeline needs a target to call — naming it is instruction 2 |
| T65.5 | both `sprint-process.md` sections exist | yes — prose |

The two honest gaps are T65.2's and T65.1's, and both are written as the
ticket's **first instruction** rather than resolved here, per T15.5's lesson
that a planning check asserting an unread capability is how a ticket gets
cleared and then hits a wall.

## §5 — Deliberately out of scope, with the decision recorded

- **#149's read port into Booking.** A genuine design change and the last
  caller-supplied ownership fact; it wants its own sprint.
- **#134's WCAG pass.** Premise re-verified; needs an operator and a real
  screen reader.
- **Whether `lock-check` should detect a superset lockfile.** Needs a policy
  about which resolved versions are load-bearing — i.e. a list — and the honest
  answer may be no. T64's review recorded it in the gate's own OK message
  rather than as an issue.
- **`govulncheck`.** `Forbidden` from here, and the one question that would
  settle whether that is permanent (reading the proxy's status endpoint) is
  **denied to this session and to its agents**. Carried as *"does not, as
  configured"*. **This is the project's longest-standing unchecked
  inaction-licensing claim and the only one it cannot check itself.**
- **A gate for T65.3's rewritten sections.** Rejected in T65.3 instruction 5.

## §6 — The premise challenge (T64 retro recommendation 8)

T64's retro recorded that its three reviewer passes raised 42 findings and
**none of them asked whether the sprint's tickets were the right work.** Every
pass checked whether the work was done correctly. Recommendation 8 was to brief
an agent on the **premise** instead.

**This plan is the first subject to that**, and the brief is deliberately
hostile: argue the slate is self-referential, argue the ordering is wrong,
argue T65.3 is the wrong fix, name the product work that should displace
something, and argue the sprint is too big. The agent was given the plan before
it was proposed for merge, so its objections can change the tickets rather than
annotate them.

<!-- T65-PREMISE-CHALLENGE -->

## §7 — What this ceremony produced

1. **The security gate run first, and dated** — PASS, where T64's ceremony
   found it red by doing the same thing one sprint earlier.
2. **The ADR half of the escalation sweep answered by one command**, the first
   use of the listing T64.5 built after the previous ceremony had to copy a
   package to a scratch directory.
3. **T64.5's prose step applied**, surfacing three engineering decisions in
   issue bodies and confirming #145's product question answered — **and stating
   that it proved nothing here**, because this session wrote all seven bodies
   within a day.
4. **The premise check fired on zero of seven, recorded as arithmetic rather
   than evidence** for the same reason.
5. **T64's row completed**, including that **#326 and #331 carry the first
   non-author reviews in this project's history**.
6. **The compliance question answered honestly: the sprint that did not write
   these rules has not happened yet**, so the only real evidence remains that
   T64's derived audit found every uncommanded figure and nothing else.
7. **Five tickets**, 16 points, with two dependency gaps named as first
   instructions, and five documented exclusions.
