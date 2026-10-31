# T66 retro

Written late, at T67's Ceremony 1, which is itself finding 1. Every quantity
below carries the command that produced it; a figure with no command is a
figure nobody measured.

## 1. The board of record did not describe what shipped

T66 merged eight PRs, verified against each `merged_at` rather than inferred
from numbering: **#341 → #343 → #346 → #347 → #349 → #350 → #351 → #354.**

Exactly **one** of them — #343 — maps to a ticket in `t66-sprint-plan.md`
(T66.2). The other seven are real product and infrastructure work that no
ticket covers: the first-run fixes, the CI/CD workflow, `GetCurrentUser`, the
409 split, court names, the vulngate grouping, and the four-defect audit fix.

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

## 4. CLAUDE.md drifted into two measurable falsehoods in 24 days

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

The lesson is not that the authors were careless — they were unusually careful,
and the figures were right when written. It is that **nothing re-checks the one
document that is always loaded.** No gate reads `CLAUDE.md`: `docsindex` parses
`HANDOFF.md`, `docs/process/` and `docs/adr/` only. T67 takes a ticket for this.

## 5. What found the product defects was a person in a browser

Every user-facing defect T66 fixed — the raw court uuids, the permanently-empty
host payments dashboard, the unreachable competitions route, the registered
player still offered a Join button — was found by opening the app, not by a
gate. The web suite was green throughout.

That is the same finding as T65's recommendation 9, unchanged a sprint later.
It is why every T67 ticket carries a real-browser pass as a non-functional
requirement rather than a suggestion.

## Recommendations for T67+

1. A slate replaced mid-sprint gets one line in the plan saying what replaced
   it. Applied: T67's plan records its three product decisions inline.
2. A test whose expectation is a hand-written list beside the code it checks is
   a review finding, not a style note. Derive it or justify it.
3. Something must re-check the always-loaded document. T67 takes it.
4. The retro is written in the sprint it concerns. This one was not, and the
   cost was that finding 1 was discovered by the next sprint's planning rather
   than by the sprint that caused it.
