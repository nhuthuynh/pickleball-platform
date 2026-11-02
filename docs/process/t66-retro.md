# T66 retro

Written late, at T67's Ceremony 1, which is itself finding 1. Every quantity
below carries the command that produced it; a figure with no command is a
figure nobody measured.

## 1. The board of record did not describe what shipped

T66 merged **nine** PRs, verified against each `merged_at` rather than inferred
from numbering: **#338 → #341 → #343 → #346 → #347 → #349 → #350 → #351 →
#354.** T66's security gate caught a `critical` and the fix was the first
thing the sprint merged.

Mapping them to the plan's five tickets, with the criterion stated so the
count is checkable rather than assertable — a PR maps to a ticket when it
delivers that ticket's story, whether or not it followed its instructions:

- **#343 → T66.2.** Unambiguous.
- **#347 → T66.1, substantially.** T66.1's story is *"I want CI to run every
  gate `ci-checks` runs, so that a green pipeline means what the local gate
  means"*, and #347 delivers exactly that. It did not follow T66.1's
  instructions — the Jenkinsfile was not made to invoke `ci-checks`, and #330
  is still open — but calling it work no ticket covers is a stretch, and the
  first draft of this retro did.
- **#338 and #341 are ceremony, not uncovered work.** #341 is T66's own
  Ceremony 1 artifact, which the process mandates; #338 is the security fix
  that ceremony surfaced.
- **T66.3, T66.4 and T66.5 did not ship at all.** Verified for T66.3 and
  T66.4: no `LESSONS` reference anywhere in `tools/docsindex/` or
  `cmd/docsindex/`, and no mutation-restore rule in
  `docs/process/sprint-process.md`. T66.5 — run `ci-integration` — is
  evidenced differently and more weakly: no dated run or recorded interval
  exists anywhere after T61's, so the absence is the evidence. Said separately
  because an earlier draft put all three under one "Verified:" that covered
  two of them, which is this project's own recurring shape.

That leaves **five** PRs of real product work no ticket covers — #346, #349,
#350, #351 and #354 — and three planned tickets that never happened. The
finding stands. Note #351 carries *two* unticketed changes — court names and
the vulngate grouping — so a count of PRs and a count of changes differ here,
which is the trap an earlier draft of this paragraph fell into.

This is not a bookkeeping nit. The sprint plan is the board of record, and for
T66 it records a different sprint than the one that happened. The cause is
legible: T66's plan was written, then a browser session produced defects that
were obviously worth more than the planned slate, and the plan was never
amended to say so. The fix is not "plan harder" — it is that a slate replaced
mid-sprint gets one line saying what replaced it.

## 2. Three of four defects in #354 were green tests agreeing with a defect

The four defects #354 fixed were all introduced during T66 itself, and three
share one shape: **a test that passed because it asserted the defect.**

- The CI image step asserted only `status != 0`. The server connects to
  Postgres long before it checks auth, so a DB-less `docker run` exited 1 on
  `pg: ping: failed to connect` and the step printed "as designed". It would
  have passed with the auth refusal deleted.
- `GetCurrentUser` was missing from `AuthenticatedMethods()`. The test that
  should have caught it compared against a hardcoded two-entry list, so it was
  green by agreeing with the omission.
- A web spec asserted `toContain('court-1, court-2')` — the raw-uuid output
  that #342 existed to remove.

The replacements derive their expectations instead of restating them: the
authenticated-methods test now parses which handlers call
`auth.PrincipalFromContext`. **An expectation written by hand beside the code
it checks is not a test of that code.** This is the generalisation of T56's
wire-test lesson and it now has three instances in one sprint.

## 3. The vacuously-green gate shipped inside the PR about vacuously-green gates

#347's own commit message and PR body are about gates that report green while
checking nothing. The image-smoke step added in that PR was one. Writing about
a failure mode in the same change that commits it is worth recording plainly,
because it rules out the comfortable explanation — this was not inattention to
the lesson; the lesson was the subject of the paragraph above the defect.

Second-order cause, also recorded at the time: a shell pipeline whose last
command was `tail`, so the runner reported `tail`'s exit code and the log said
`exit=2` while the summary said green. **The command whose exit code is the
evidence must be the last thing the shell runs.**

## 4. CLAUDE.md drifted into three measurable falsehoods in 23 days

T64 put three adversarial passes over `CLAUDE.md` line by line. Measured at
`c9631dc`, with `internal/gen` present so the stated precondition holds:

```
$ make gate-coverage | grep "no test function"
gate-coverage: NOTE — 26 package(s) hold no test function at all.
$ make gate-coverage | grep -c "^    cmd/"
2
```

The file asserts **27** and **"three"**. Both wrong, and wrong according to the
command the file itself prints beside the figure.

**There is a third.** `CLAUDE.md:147` prints `14 package(s)` for a fresh clone
before any codegen. `make gate-coverage` reports 26 total "plus 13 generated",
so the hand-written remainder is 13 and that figure should be 13 too.

**And it is the one the new gate cannot catch.** The block's own command is
`rm -rf internal/gen`, which deletes generated code, so T67.5 (#362) leaves it
unmarked by design and no run of the gate produces this number. The gate's
documented blind spot and this project's third stale figure are the same
block — which is a stronger argument for the gate than crediting it with a
catch it did not make. #362 reports the drift as found while building the gate
but prints no run output for it, so the record does not show whether it was
read or measured by hand.

Three drafts of this paragraph were wrong about the third figure in three
different ways, which is on-topic here and nowhere else: this finding's whole
subject is a document asserting what nobody re-checked. See the commit history
for what each said.

**And the comfortable explanation is unavailable.** All three figures went
stale in one commit — `ce962d5` (#351), merged 2026-10-29 — which added
`cmd/vulngate/main_test.go`, taking that package off the untested list in
**both** documented tree states. That moved the total from 27 to 26, the `cmd`
count from three to two, and the fresh-clone figure from 14 to 13. **The
commit's own message says so:**

```
$ git log -1 --format=%B ce962d5 | grep -n "27"
29:27 -> 26.
```

So the author measured the change, wrote the new figure into the commit
message, and did not update the document that states the old one. This is not
inattention and not ignorance: the knowledge existed, in writing, in the same
change. The missing thing is a mechanism — **no gate reads `CLAUDE.md`**
(`docsindex` parses `HANDOFF.md`, `docs/process/` and `docs/adr/` only), so
there was nothing to fail. T67.5 takes it.

The second figure is worse than the first, because nothing announced it: the
`cmd` count fell to two as a side effect of the same test, and no commit
message mentions it. A figure that goes stale silently is the ordinary case;
this sprint happens to contain one of each.

## 5. What found the product defects was a person in a browser

Every user-facing defect T66 fixed — the raw court uuids, the permanently-empty
host payments dashboard, the unreachable competitions route, the registered
player still offered a Join button — was found by opening the app, not by a
gate. The web suite was green throughout.

That is the same finding as T65's recommendation 9, unchanged a sprint later.
It is why every T67 ticket carries a real-browser pass as a non-functional
requirement rather than a suggestion.

## 6. A row template is inherited, including its claims

`HANDOFF.md`'s T66 row asserted "all reviewed via GitHub PR reviews, see
naming convention" and **no review exists for any of the nine PRs** — measured,
with four live controls. Nobody decided to claim that. The sentence came with
the row shape, copied from sprints where it was true, and it invoked
`CLAUDE.md`'s own naming convention to vouch for a record that was never
created.

That is the same decay the red-team pass measured on `sprint-process.md`'s
dispatch-isolation rule: carried by template inheritance for ~24 sprints, then
silently zero, because nothing checked it. A template propagates its claims as
readily as its columns, and a claim in a table cell has no command beside it.

The cheap fix is not vigilance. It is that a cell asserting a GitHub artifact
exists should be derived from GitHub, the way `docsindex` derives the rest of
this file's agreement with the tree.

## Recommendations for T67+

1. A slate replaced mid-sprint gets one line in the plan saying what replaced
   it. Applied: T67's plan records its three product decisions inline.
2. A test whose expectation is a hand-written list beside the code it checks is
   a review finding, not a style note. Derive it or justify it.
3. Something must re-check the always-loaded document. T67 takes it.
4. The retro is written in the sprint it concerns. This one was not, and the
   cost was that finding 1 was discovered by the next sprint's planning rather
   than by the sprint that caused it.
