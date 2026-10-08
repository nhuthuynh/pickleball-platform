# T65 Sprint Retro — Ceremony 3

Per `docs/process/sprint-process.md` Ceremony 3. The six-role team, held
against `docs/process/t65-sprint-plan.md`, the merged work, and the three
adversarial review passes that ran against it.

Subject to **T62.2** (every quantity carries its command), **T63.1** (an
action-claim carries the command that re-reads live state), **T63.3** (a guard
is verified by removing it) and **T64.3** (a gate claim carries its date).

**A retro with zero findings is treated as suspicious, not as a clean bill of
health** — this document's own governing rule. This one has no such problem.

---

## 1. What happened, in order

Seven commits, squashed into one:

```
$ git log --format='%h %s' b635eab..51c8a10 | cut -c1-72
51c8a10 T65 second review pass: a committed 3 MB binary, a precedence f
53b0268 T65.2 review fixes: a Q2 guard that did not guard, a false just
b32437f T65.2 follow-up: map the four new domain sentinels, and file th
63757b3 T65.4: #328 — tests for cmd/docsindex, and a decided answer to
4c5fb3d T65.3: record both answers where the next reader meets them, an
3445f6c T65.2: Player Level, per ADR-0012 Q1's answer — the first new c
e269a06 T65.1: the Vue client can authenticate — every write path in t

$ git diff --shortstat b635eab 12942a8
 35 files changed, 4632 insertions(+), 54 deletions(-)

$ git diff b635eab 12942a8 -- '*_test.go' | grep -c "^+func Test"
53
```

Merged, `merged_at`-verified rather than assumed:

```
$ gh api .../pulls/{332,336} --jq '"#\(.number) \(.merged_at) \(.merge_commit_sha[0:7])"'
#332 2026-10-06T12:33:28Z b635eab    Ceremony 1
#336 2026-10-07T15:22:58Z 12942a8    all four tickets + two review passes
```

**Two PRs, not five.** T63 and T64 each took one PR per ticket; T65 took one
for the ceremony and one for the work. That was not a decision, it was drift,
and §9 records what it cost.

---

## 2. The finding this retro exists to record: a fix needs the same scrutiny as
the claim it fixes

The first review pass on T65.2 returned 17 findings. Its through-line was one
pattern: **correct arithmetic, a conclusion that overshoots it.** Two examples
it caught:

- `ConfidenceGames = 20` was justified as *"the point where one more win stops
  moving the number by a visible step"*, citing a standard error that is
  arithmetically right. The conclusion is wrong twice: the per-win step is
  **0.2 flat for every n from 1 to 20** and only shrinks above it, and the
  emitted value's confidence interval is **widest** exactly at n = 20.
- `SuggestMatchups` was documented as *"the optimum for 'make every match as
  even as possible'"*. It is the optimum for the **total** gap, which is not
  the same objective.

Both were fixed. **And both fixes repeated the pattern**, which the third pass
found:

| the fix | why it overshot |
|---|---|
| the error-precedence fix | hoisted pinned membership above pinned duplicates, which made the two *reported* cases agree — while validation stayed interleaved per element, so four more cases still contradicted the documented order. Meanwhile the doc comment had been **strengthened** to claim the class |
| the `ConfidenceGames` rewrite | replaced a false conclusion with another: *"a player cannot be moved a grade by one evening."* The per-result cap (0.2 of a level, exactly `4/ConfidenceGames`) is right and does not compose — at seed 1, five straight wins reach 2.0 and ten reach 3.0 |

**That is the finding.** A fix is a new claim, written under the belief that
the problem is now understood, and that belief is exactly what made the first
version wrong. Reviewing the original work and not the repair leaves the
sprint's most confident prose unexamined.

**It is not a new lesson so much as this project's oldest one, one level up.**
`CLAUDE.md` already records that *"a redefinition that reads as purely
additive is exactly the shape this failure takes"* about
`enforce_game_capacity()`, redefined five times, two live defects. The
analogue for prose is: a correction that reads as purely clarifying is exactly
the shape this failure takes.

---

## 3. A 3 MB compiled binary was committed, and no gate could see it

T65.4 committed `docsindex`, a 3,061,505-byte ELF executable, at the
repository root. It was found by the second review pass — **not** by
`fmt-check`, `lint`, `gate-coverage`, `docs-index-check`, `lock-check` or any
test.

```
$ git show --name-status 63757b3 | grep -x "A.docsindex"
A	docsindex
$ file docsindex
docsindex: ELF 64-bit LSB executable, x86-64, … with debug_info, not stripped
```

**The mechanism is three steps and every one is ordinary.** `go build
./cmd/docsindex` with no `-o` writes the executable into the working
directory, named after the package. `git add -A` tracks it. And then — the
part that makes it durable rather than noticeable — **once tracked, `git
status` is clean**, so the next `go build` silently stages a 3 MB binary diff
into whatever commit comes next.

The second-order harm is worse than the size. `CLAUDE.md` now tells Ceremony 1
to run `go run ./cmd/docsindex -statuses`. A stale executable sitting where
`./docsindex -statuses` also works is a ceremony answering its question from
code that no longer exists.

**`make binary-check` now gates it**, on the four-byte ELF magic number rather
than on a list of names, so a sixth `cmd` package or a binary committed from
anywhere else fails it too. `.gitignore` also lists the five current `cmd/`
names, and the gate exists precisely not to depend on that list.

```
$ make binary-check
binary-check: OK — no tracked file carries the ELF magic number.
```

Verified by committing one, which is the only verification that counts
(T63.3).

**Why this is a finding and not a slip.** `make ci-checks` runs eighteen
prerequisites —

```
$ grep -E "^ci-checks:" Makefile | sed 's/^ci-checks: //' | tr ' ' '\n' | wc -l
18
```

— and the project spent three sprints building the one that answers *"which
tests does no gate run?"*. Nothing answered *"is anything in this repository
not source?"*, and the answer had been no for 64 sprints, which is why nobody
asked.

---

## 4. `git checkout <file>` destroyed uncommitted work. Twice.

Self-inflicted, and the costliest process failure of the sprint in wall-clock
terms.

Mutation verification works by breaking the code, running the suite, and
restoring. The restore step was `git checkout <file>` — which restores the
file to its **committed** state, not to its pre-mutation state. On a file
holding an hour of uncommitted work, that is a silent, total loss.

It happened twice:

1. `tools/gatecoverage/gatecoverage.go` — lost the CRLF fix, the
   `withoutSkipped` filter, the rejected-options record and a figure
   correction. Noticed immediately, because the suite went red on a missing
   function.
2. `internal/identity/domain/player_level_test.go` — lost the **entire**
   repo-wide Gender guard rewrite: the extracted scan, 16 synthetic cases, the
   derived vacuity floor, the comment-stripping fixes. Noticed only because
   the next mutation produced no output at all, and then confirmed by grepping
   for a function that was no longer there.

Both were reconstructed. The second took longer to rebuild than to write the
first time, because the reasoning had to be recovered from the review report
rather than from the file.

**The rule this wants.** A mutation is a temporary edit to a file that may
already hold uncommitted work, so the restore must come from a copy taken
immediately before the mutation, never from git:

```
cp <file> "$SCRATCH/<file>.orig"      # before the first mutation
...mutate, run, observe...
cp "$SCRATCH/<file>.orig" <file>      # restore
```

The second half of the sprint did exactly this and had no further losses.
`git checkout`, `git restore` and `git stash` are all the same trap here: they
are defined against the index, and the index is not where the work is.

**And a second-order note worth more than the rule.** The loss was detected
*by a test failing*, both times. A mutation campaign that restores wrongly and
restores to something that still compiles would be silent — and the project
has no gate that could tell.

---

## 5. A hand-written list inside a test is the same defect as one inside a tool

`CLAUDE.md` states the rule for `tools/gatecoverage`: *"There is no package
list in the tool, and adding one is the one change that would defeat it."*
Three sprints running (T11, T12, T13/#157) shipped a hand-written glob that
was stale before its sprint ended.

T65 produced **three** fresh instances of the same defect, none of them in a
tool:

1. **The ADR-0012 Q2 Gender guard** listed three types and checked them by
   reflection. The first review walked a new `Gender`-bearing type past it and
   a `Gender` field on `socialplay.Registration` — the natural home for a
   gender-mix feature, since a Registration is how a player joins a Game — and
   observed that the test said nothing at all about the schema or any proto,
   which three documents claimed it did.
2. **Its replacement** parsed the tree and had **five** more bypasses plus a
   false positive, found by the third pass: a Go package in a directory named
   `coverage` or `dist` (`fs.SkipDir` matches a basename *anywhere*); a column
   behind `DEFAULT 'https://…'` and another behind a `'--'` literal (comment
   stripping ran before quote stripping); `Sex` and `BiologicalSex`, the same
   protected attribute under another name, while the message said "a protected
   attribute"; a `const`/`var`/`func`, although ADR-0012 §4 bans a
   matching-mode flag separately; and an embedded field, which has no name of
   its own. The false positive: a migration documenting its own compliance
   **inside a `/* */` block comment** turned the gate red.
3. **`gatecoverage_test.go` asserted that every generated package starts with
   `internal/gen/`** — the forbidden list, relocated from the tool into its
   gate. A correctly-marked `wire` or `mockgen` package anywhere else would
   have turned `ci-checks` red *for doing its job*, and the repair a session
   under pressure reaches for is editing the hardcoded prefix.

**The generalisation.** The anti-list rule has always been stated about
`tools/gatecoverage`, so it reads as a fact about that tool. It is a fact about
**guards**: a guard whose subject is enumerated by hand is stale the moment
the tree moves, and it does not matter whether the enumeration lives in the
tool, in the tool's test, or in a reflection loop over three struct literals.

All three now derive their subject. The Gender scan parses `.go` declarations
and scans `.sql`, `.proto`, `.ts` and `.vue` with comments stripped; its
vacuity floor is derived too (every bounded context under `internal/` must
have contributed a parsed file, read from disk), because a flat `goFiles < 100`
floor would still have reported green after losing `socialplay` (77 files) and
`identity` (18).

---

## 6. T64's recommendation 8, applied, was the highest-value step of the sprint

T64's retro asked for one thing this sprint had not done before:

> **Brief a reviewer agent on the premise, not just the diff.** … The one
> thing none of them asked was whether T64's tickets were the right five.

T65's Ceremony 1 ran that pass, and it found:

- **the draft slate had zero of five tickets touching the product**, in a
  project whose purpose is a pickleball platform — two documents, an internal
  tool's tests, a dev-only dependency bump, two internal lists and two process
  clauses;
- **two live product decisions the escalation sweep had declared absent**:
  ADR-0012 **Q1** (Player Level weighting, open since 2026-08-10) and
  ADR-0009's market scope (open since T7). Both were put to the Product Owner
  and both were answered the same day.

The slate was rebuilt to four tickets, two of them product, and the sprint
shipped the **first new user-facing capability since T58** plus a fix for a UI
whose entire write surface was unreachable.

**The cost was one agent brief. The return was the sprint.** That ratio is
worth recording precisely because the step is cheap enough to skip.

---

## 7. The sprint plan's own stated property was false, and a test found it

T65.2's instruction 2 named three properties the formula had to have. The
second — *"a 100%-win-rate newcomer must sit below a proven regular"* — is
false as written, and the test that encoded it failed on its first run:

```
player_level_test.go:212: a 100%-win-rate player with 14 games (4.4)
    must sit below an 85%-over-100 regular (4.4)
```

At seed 3 a flawless run reaches the regular's level at 14 games and passes it
at 15. A 14-0 run is not a newcomer's record, and the Product Owner's answer
("early results move it less") does not say a flawless mid-length run must
stay below a merely-strong long one.

**The claim was narrowed and the crossover pinned by search** rather than by
restating 15, so retuning the constant moves the test's answer instead of
breaking it. The plan was not edited to pretend it had said this.

**Worth noting about the process:** the plan's three properties came from the
same ceremony that produced the premise challenge, and this one was asserted
without being computed. Writing a property is not the same as checking one,
even in a document whose own rules say so.

---

## 8. What the reviewer agents found

Three passes, all report-only under rule 9 — none committed, pushed, or edited
a tracked file. **Findings are attributed to the agents rather than re-derived
here**, per T62.2's quotation exemption:

| Pass | Scope | Findings | Of which real defects |
|---|---|---|---|
| 1 | T65.2 code | 17 | 4 |
| 2 | T65.3 + T65.4 | 18 | the committed binary, the relocated path list, a clean-checkout failure, an untested error path, CRLF, `./...` bypassing the skip rule |
| 3 | re-review of pass 1's fixes | 8 | **2 introduced by the fixes** (§2), 5 Gender bypasses + 1 false positive |

**Every one was addressed**; none was deferred as a follow-up ticket, which is
itself a signal — see §11 recommendation 5.

**What the agents were worth, stated as narrowly as the evidence allows.**
Pass 2's single finding — a 3 MB tracked binary — is worth more than the rest
of the sprint's review effort combined, because no gate existed and none would
have been written. Pass 3's two findings are worth more than they look,
because they are the only evidence in this project that a *fix* carries the
same risk as the work.

**And the limit.** All three passes were pointed at named files with named
questions. The one thing none asked was whether the *code* should exist at all
— the Ceremony 1 premise pass asked that about the tickets, and nobody asked
it about the implementations. Pass 3 came closest, by attacking the fixes'
claims rather than their code.

---

## 9. T64's nine recommendations: three were silently dropped

The retro's own follow-through, checked one by one against the tree rather
than from memory:

| # | T64's recommendation | T65 |
|---|---|---|
| 1 | run the security gate first in every Ceremony 1 | **done** — §0 of the plan, PASS and dated |
| 2 | `lock-check` superset detection, only if designable without a list | **deferred, recorded** — plan §5 says the honest answer may be no |
| 3 | #320's `vitest` bump, now bounded rather than blocked | **deferred, recorded** — plan's deferral list |
| 4 | add the stale-read qualification to T63.1, or decide against it | **silently dropped** |
| 5 | correct `CLAUDE.md`/`HANDOFF.md`; treat the structural half as a ticket | mechanical half **done** at #331; structural half **deferred, recorded** |
| 6 | carry `vuln.go.dev` as *"does not, as configured"* | **partially** — the plan says it; `CLAUDE.md:382` still says *"the Go half cannot run here"* |
| 7 | fix what the review found in its own PR, ticket the rest | **done** — #331, and #328 became T65.4 |
| 8 | brief a reviewer agent on the premise | **done**, and §6 is what it bought |
| 9 | document the squash-ancestry procedure | **silently dropped** |

Verified:

```
$ grep -c "read the object, not the list\|stale-read" docs/process/sprint-process.md
0
$ grep -n "cannot run here\|does not, as configured" CLAUDE.md
382:- **`make security`: the Go half cannot run here, and that is not a reason to
$ grep -rc "squash-ancestry\|merge-tree --write-tree" docs/process/sprint-process.md CLAUDE.md
docs/process/sprint-process.md:0
CLAUDE.md:0
```

**Six of nine honoured; three not — and the distinction that matters is
"deferred and recorded" versus "silently dropped".** Three recommendations
were deferred *with the decision written down in the sprint plan*, which is
exactly right. Three others simply did not happen, and nothing anywhere says
why.

**This is the same failure mode the escalation sweep was rebuilt three times
to fix**, one artifact over: a question that has nowhere to be read is a
question nobody answers. T65's Ceremony 1 checked compliance with the four
*rules* (T62.2/T63.1/T63.3/T64.3) and never looked at the previous retro's
*recommendations*. Recommendation 6 is the sharpest instance — the plan
adopted the honest wording in its own §0 and left the rulebook sentence it was
about untouched, which means the recommendation was read and still not done.

---

## 9b. Eleven retros have no `LESSONS.md` stub, and this ceremony's own
instruction is where that was found

Ceremony 3 requires two artifacts, not one: the findings in
`docs/process/t<N>-retro.md`, **and** a short `## T<N> sprint retro` stub
appended to `docs/LESSONS.md` pointing at that file. Writing this retro meant
reading that instruction, which meant looking for the stub to copy.

```
$ ls docs/process/t*-retro.md | sed 's/.*\/t\([0-9]*\)-retro.md/\1/' | sort -n | tr '\n' ' '
5 9 10 11 … 52 53 54 55 56 57 58 59 60 61 62 63 64 65

$ grep -o "^## T[0-9]* sprint retro" docs/LESSONS.md | grep -o "T[0-9]*" | sed 's/T//' | sort -n | tr '\n' ' '
5 9 10 11 … 52 53
```

**The stubs stop at T53. Eleven retros — T54 through T64 — have none**, and
T65's would have been the twelfth had this section not been written.

**Why this is the same finding as §9 and worth stating separately anyway.**
Both are a required artifact that nothing reads: §9's recommendations list has
no reader, and this index has no checker. But this one is *gated-adjacent* in a
way the other is not — `make docs-index-check` verifies that `HANDOFF.md`'s
Docs index agrees with the tree, and it does not look at `LESSONS.md` at all.
Eleven sprints produced the artifact the convention is about and skipped the
pointer to it, which means the convention's purpose — a single chronological
log from which any retro is reachable — has been broken since T54 without one
gate, ceremony or review noticing.

**T65's stub is appended, because this ceremony owes it.** The backfill of
T54–T64 is **not** done here and is not a correction: writing eleven summaries
of other sprints' retros is real work with real room to misrepresent them, and
it wants its own ticket and its own review. Recommendation 9 carries it.

**And the cheaper half, which is the actual fix.** `docs-index-check` already
parses the tree and `HANDOFF.md`; adding "every `docs/process/t<N>-retro.md`
has a `## T<N> sprint retro` stub in `docs/LESSONS.md`" is a third derived
check in a tool built for exactly that, with no list. Had it existed at T54 it
would have failed eleven times.

---

## 10. What went well

1. **The security gate ran first, and passed** (T64's recommendation 1, on its
   first opportunity):

   ```
   2026-10-06T11:10:54Z $ SKIP_GOVULNCHECK=1 make security | tail -1
   PASS: no new gating findings (0 baselined, 2 below threshold).
   ```

2. **T65.1 was proven end-to-end against a real server**, which is the first
   time this project's client auth path has been exercised outside a fixture.
   A real Postgres, 33 migrations, `cmd/server` on `dev/auth/`'s fixture, and
   the sequence 401 → 403 → 200 → 200 → 401. The server's own log line
   (`authenticated_methods: 33`) also corrected a figure that a source grep
   had got wrong in both directions — a live instance of T63.1's "prefer a
   count the running system reports".

3. **Mutation verification earned its rule, repeatedly.** Roughly thirty
   mutations across the sprint; the ones that paid were the ones that
   *survived* first: a manual override would have lasted exactly one
   recompute, `WithManualOverride` could clear `Provisional` unnoticed,
   `ComputeLevel`'s documented precedence had no case where both inputs were
   invalid, `TotalLevelGap` had no test at all, and a failing `./...` listing
   would have emptied `gate-coverage`'s new report and still printed `OK`.

4. **ADR-0012 §4's contradiction was found and resolved rather than papered
   over.** §4 forbade computing a Level from `Match` history while the
   trigger required building exactly that; the amendment had even cited §4 in
   the other direction to justify not persisting anything. It now carries a
   clause-by-clause carve-out.

5. **`make ci-checks` green on the merged tree, dated, and captured with
   `make`'s own exit status** — which matters because an earlier run this
   sprint reported exit 0 through a `| tail` while `make` had failed. The
   pipeline's status is `tail`'s, not `make`'s, and that nearly shipped a red
   gate as green.

6. **Four issues were filed rather than absorbed**: #333 (the formula has no
   readable input), #334 (`-statuses` overcounts live decisions), #335 (two of
   six contexts have no sentinel-conformance test), and #328 closed by T65.4.

   ```
   $ gh api .../issues/328 --jq '"\(.state) \(.closed_at)"'
   closed 2026-10-07T15:23:30Z
   ```

---

## 11. Recommendations for T66 and beyond

1. **Add a Ceremony 1 step: walk the previous retro's recommendations one by
   one and record, per recommendation, done / deferred-with-reason / dropped.**
   §9 is the argument and it is not a close call: three of nine vanished, one
   of them after being read and partially applied. The step is cheap — nine
   rows — and it is the only mechanism that distinguishes a deliberate
   deferral from an omission. Note the precedent: the escalation sweep needed
   three rebuilds because each one read one level too shallow, and the
   recommendations list currently has no reader at all.

2. **Write the mutation-verification procedure into `sprint-process.md`,
   including the restore step.** §4 cost this sprint two reconstructions, the
   second expensive. The procedure is four lines and the load-bearing part is
   negative: **never restore a mutation with `git checkout`, `git restore` or
   `git stash`** — they are defined against the index, and the uncommitted
   work is not there. T63.3 made mutation verification a rule without saying
   how to run one safely.

3. **Review the fixes, not only the work.** §2's two findings exist because a
   third pass was pointed at the second pass's repairs. Make that the default
   shape for a review-fix PR: the brief is "attack the fixes' claims", and the
   reviewer is told which claims were strengthened.

4. **Generalise the anti-list rule from `tools/gatecoverage` to guards.** §5
   produced three fresh instances in one sprint, one of them inside the test of
   the very tool the rule is written about. `CLAUDE.md` states it as a fact
   about one tool; it is a fact about any check whose subject is enumerated by
   hand. One sentence in the rulebook, plus the three instances as the
   evidence.

5. **Zero findings deferred across 43 is a signal, not a triumph.** Every
   finding from three passes was fixed in-sprint. That reads as diligence and
   is also how a sprint's scope doubles: T65 took four tickets and shipped
   four tickets plus two review-fix commits larger than two of the tickets. A
   ceremony should be able to say "this one is a T66 ticket" without it
   feeling like a failure — and the way to make that possible is to decide, at
   the point of triage, which findings block the merge.

6. **Teach `docs-index-check` the `LESSONS.md` stub, and ticket the T54–T64
   backfill separately** (§9b). The gate already derives both sides it
   compares; a third derived check — every retro file has a stub pointing at
   it — needs no list and would have failed eleven times since T54. The
   backfill is eleven summaries of other sprints' work and belongs in its own
   reviewed ticket, not in a gate's PR.

7. **Finish T64's 4, 6 and 9.** They are small, they are already argued, and
   two of them are one sentence each. Recommendation 6 in particular:
   `CLAUDE.md:382` still asserts *"cannot run here"* about a claim T64
   established is only *"does not, as configured"*, and that is the wording
   this project has watched survive 57 sprints before.

8. **`make ci-integration` was not run this sprint, and that is now a
   standing debt rather than an incident.** T65 touched no migration and no
   file a `*_integration_test.go` reads, so the gate was not *owed* by
   `CLAUDE.md`'s own rule — but the suite is 27 files across five contexts:

   ```
   $ for f in $(find . -name '*_test.go'); do \
       head -5 "$f" | grep -q '^//go:build integration' && echo "$f"; done \
       | sed 's|^\./internal/\([^/]*\)/.*|\1|' | sort | uniq -c
         4 booking
         4 competitions
         2 facilities
         7 payments
        10 socialplay
   ```

   T61 ran them for the first time in 57 sprints and found two live production
   defects. The interval since is now worth tracking on purpose rather than
   per-sprint-by-exception.

9. **The web suite still has no e2e tooling, and T65.1 is the proof it
   matters.** 62 files and 724 tests pass against a fixture client, and the
   product's entire write surface was unreachable from a browser for ten
   sprints:

   ```
   $ cd web && npx vitest run --reporter=dot | grep -E "Test Files|Tests "
    Test Files  62 passed (62)
         Tests  724 passed (724)
   ```

   T65.1's end-to-end check was run by hand and is not repeatable by a gate.
   That is `CLAUDE.md`'s own best lesson — *"an in-memory fake is more
   permissive than Postgres"* — with the client as the fake. Worth a ticket
   with its own review, because the cost is real and so is the gap.

---

## 12. Sweep and bookkeeping

1. **T65's row completed** in `HANDOFF.md`'s Docs index: Reviews cell →
   #336 (with its three review passes), Retro cell → this file.
2. **A `## T65 sprint retro` stub appended to `docs/LESSONS.md`**, pointing
   here — per Ceremony 3's own instruction that the retro and `LESSONS.md`'s
   incident postmortems are distinct artifacts. It is the first such stub
   since T53; §9b records why the eleven in between are missing and why their
   backfill is not done here.
3. **The open-issue list re-derived rather than copied**:

   ```
   $ gh api ".../issues?state=open" --jq '.[] | select(.pull_request == null) | .number'
   335 334 333 330 329 320 149 145 134
   ```

   Nine open, three of them filed this sprint. `HANDOFF.md`'s "Open issues"
   prose is a cached derived result and T65.3 recorded that its failure mode is
   structural; this retro does not re-cache it.
4. **No ADR is escalated**, and the ADR half of the sweep now reads bodies:
   ADR-0009 carries no live question, ADR-0012 carries Q2 and only Q2.

---

## 13. Honest-form outcome sentence

T65 shipped four tickets, two of them product, including the first new
user-facing capability since T58 and a fix for a UI whose entire write surface
had been unreachable for ten sprints — and the sprint's three most useful
findings were a 3 MB binary no gate could see, two of its own repairs
repeating the mistake they were repairing, and two required process artifacts
— the previous retro's recommendations and eleven sprints of `LESSONS.md`
stubs — going unread and unwritten because nothing checks either one.
