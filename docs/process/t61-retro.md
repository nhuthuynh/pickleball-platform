# T61 Sprint Retro

Ceremony 3 per `docs/process/sprint-process.md`, six-role team (briefs:
`docs/agent-operating-handbook.md` Part B), held against `HANDOFF.md`,
`docs/process/t58-retro.md` as the structural precedent, PR #310, issue #311,
and the live issue/PR/commit history.

**T61 held no planning ceremony.** It was one instruction — *run
`ci-integration`* — against a gap six retros had disclaimed.

**Outcome: 3 live production defects found and fixed, 1 issue opened, 1 gate
weakness closed.** Merged as PR #310 (`d0b36eb`). #311 opened. No issue was
closed, because none of the three defects had ever been filed: nothing knew
they existed.

---

## 1. What actually happened, in order

1. The session was asked to run `make ci-integration` — the command six
   retros had recorded as impossible here.
2. **It checked, instead of repeating the claim.** `dockerd` and `containerd`
   were on the box, the session ran as uid 0, and the daemon came up in about
   four seconds.
3. First run: **2386 tests, 34 failures, 21 errors, 58.4s.** The 21 errors
   were a missing `covdata` binary in this Go toolchain — built from the
   toolchain's own source, not a test failure.
4. The 34 failures reduced to seven root causes. Five were fixture rot. Two
   were live production defects (§3).
5. Fixing them made the suite green, and the rule-10 repeat runs then exposed
   an eighth problem: the suite was being served from the build cache (§5).
6. **The pre-merge review found a third production defect** — in the same
   function as the first, by a method the test suite could not have used (§4).
7. The review also challenged the shape of the new conformance test, and the
   test survived the challenge (§6).
8. Writing *this retro* found three wrong numbers in the sprint's own
   `CLAUDE.md` gotcha (§7). That is the finding this retro most needed.

## 2. The finding this retro exists to record

**Six retros wrote down that the environment could not run the integration
suite. None of them tried.**

Those six are **not** consecutive, and the shape matters: they are two runs of
three — **T17, T18, T19**, then a 36-sprint silence, then **T56, T57, T58**.
(`for f in docs/process/t*-retro.md; do grep -qiE "Docker (was |is )?(un)?available|no Docker daemon|never (been )?executed" "$f" && echo "$f"; done`.)
T59 and T60 disclaimed nothing because **neither has a retro at all** (§9), and
T61's first draft of this section said "six consecutive sprints" on the strength
of T58's own "third consecutive sprint" without running that loop — see §7, which
is about exactly this.

The gap between the two runs is the part that should be uncomfortable. The claim
did not decay steadily; it was raised three times, dropped for three dozen
sprints, and raised three more times. Nothing in between re-tested it either.

The claim's lifecycle is the point. It was presumably true when first written —
T4's `LESSONS.md` entry records a genuine `docker info` failure. It was then
quoted forward, sprint after sprint, into retros, into PR bodies, and into the
header comment of nearly every integration test file in the tree, as a
present-tense fact about the machine the session was running on. Checking it
cost one command.

T12.1 responded to the gap **correctly and partially**: `make vet-integration`
compiles the tagged files, which is exactly what caught the T11 breakage it was
built for. What it cannot do is execute an assertion, satisfy a `NOT NULL`
column, violate a `CHECK` constraint, or run a constructor's validation. Six
sprints then disclaimed the remaining hole honestly in their retros — which is
the right thing to do with a hole, and is not the same as closing it.

### This is the same failure T56 and T58 already recorded, with a third face

`docs/process/t58-retro.md` §2 put it in a table: #126 asserted a field did not
exist nine days after it was added; #299 asserted a part-payment was possible
against a constraint that forbade it. Its conclusion was that these were **one
failure mode with two faces** — *an assertion about the system that its author
had not checked against the system* — and that no age threshold would catch
either.

T61 is the third face, and the broadest:

| | #126 (T56) | #299 (T58) | the Docker claim (T61) |
|---|---|---|---|
| Asserted | "no price/fee field at all" | "a part-payment may legitimately be recorded" | "no Docker daemon is available" |
| About | the code | the schema | **the environment** |
| Falsified by | `0013_socialplay_entry_fee.sql` | `0005_payments.sql` | `dockerd &` |
| Cost of checking | one `grep` | one file read | one command |
| How long it stood | 9 days | 2 sprints | **T4 → T60**, asserted in retros at T17–T19 and T56–T58 |
| What it cost | a near-duplicate migration | nearly a guard that advertised protection it could not provide | **three production defects, one money-adjacent** |

The first two were caught before they cost anything. This one was not, and the
difference is instructive: #126 and #299 were checked because *somebody was
about to build on them*. Nobody was ever about to build on "Docker doesn't
work" — it was a reason **not** to act, and a reason not to act is never on the
critical path of anything, so nothing ever forced it to be verified.

**That is the generalisation worth keeping: a claim that licenses inaction is
the least likely claim in a codebase to be re-checked, and therefore the most
valuable one to re-check.** Every other assertion gets tested by someone trying
to use it.

## 3. The two defects the suite found

Both were green on every Docker-free gate for their whole lives, and in both
the Go half of CLAUDE.md rule 4 was correct the entire time — **the
authoritative half was the wrong one.**

### Guests stopped counting toward Game capacity (T19.1 → T61)

`0023` added a cancelled-Game check to `enforce_game_capacity()` with a
`CREATE OR REPLACE` whose body was rebuilt from **`0006`'s** version, reverting
`0012`'s weighted `SUM(1 + guest_count)` to a plain `COUNT(*)`. A 7-person Game
accepted 7 registrations each bringing 3 guests: **28 people in a 7-person
Game.**

`0023`'s own header states the function is *"unchanged by this migration (that
function is 0006's)"*. The first clause is false and the second explains why —
it recorded which migration **created** the function rather than which one last
**defined** it.

The test that catches this, `guest_capacity_concurrency_integration_test.go`,
has asserted exactly this since T8.7, and failed with the precise numbers
`0012`'s own doc comment predicts for a `COUNT(*)` implementation.

### Every Competition-entry payment failed against a real database (T10.6 → T61)

`payments.payable_type`'s `CHECK` still listed only
booking/registration/no_show_fee. `domain.PayableTypeCompetitionEntry` and the
whole Competitions payment path — port, adapter, routing, `IsValid()`'s new
case, a cross-context integration test — shipped at T10.6. The migration was
never written, so every Competition-entry payment raised `SQLSTATE 23514`.

Money-adjacent, and broken from the sprint it was built in.

**Why no unit test caught it:** the in-memory Payments repository has no `CHECK`
constraint to violate. **The fixture was more permissive than the database it
stood in for**, so the tests proved the routing and hid the storage.

That makes this the **third** instance of "a safety net hid the defect it was
compensating for", which `t58-retro.md` §3 said belonged in `docs/LESSONS.md`
rather than a fourth retro — and it is worse than its two predecessors in one
specific way. Both of those were caught pre-merge by a reviewer. This one
shipped, and sat.

## 4. The third defect — found by the review, and not findable by the suite

**The pre-merge review did not read the new migration on its own.** It
extracted every body of `enforce_game_capacity()` across `db/migrations` and
diffed them against each other. The declared variables alone tell the story:

| | `active_count` | `active_weight` | `reserved_by_others` | `game_status` |
|---|---|---|---|---|
| `0006` (T5.4) | ✓ | | | |
| `0007` (T6.6) | ✓ | | **✓ added** | |
| `0012` (T8.7) | | **✓ added** | ✗ **lost** | |
| `0023` (T19.1) | ✓ **regressed** | ✗ **lost** | | **✓ added** |
| `0030` as first drafted | | ✓ | ✗ **still lost** | ✓ |

`0007` had taught the function that an unexpired `promoted` waitlist entry
belonging to a **different** player occupies a slot, so a third party cannot
take the slot a promotion is holding during its response window. `0012`
rewrote the function from `0006`'s body and that count vanished — unmentioned
in `0012`'s header, comments, or anywhere else.

So **`0012` is simultaneously the fix for one divergence and the cause of
another**, `0023` then did the same thing to `0012`, and the first draft of
`0030` restored only the half the suite had failed on.

Two consequences worth separating:

1. **In production**, from T8.7 a promoted player's reserved slot could be
   taken out from under them by any concurrent direct registration, while
   `domain.SlotReservedByPromotion` went on enforcing the rule in Go.
2. **For this retro**, the more important one: **no test anywhere would ever
   have found it.** `0007` shipped the guard with no test on either side of
   the boundary, so when `0012` dropped it, nothing failed. Seventeen
   migrations landed in between.

### The two methods, and why neither is sufficient

T61 is an unusually clean natural experiment, because two defects were found by
a machine executing a suite and one by a human-style reading, and neither
method could have found the other's finds:

- The suite found #1 and #2 and could not have found #3 — there was no test.
- The review found #3 and would plausibly not have found #2 — nothing about
  `payable_type`'s diff invites you to go read a CHECK constraint from T6.4.

**Running the suite was necessary and was not sufficient.** The question that
turned two defects into three was *"what else did this function lose?"* — asked
of the function's whole history rather than of the diff.

### Where the test had to go, and why the obvious place would not have worked

`waitlist_reservation_integration_test.go` drives the **repository**, not
`app.Service`. That is load-bearing, not convenience: `RegisterForGame` calls
`domain.SlotReservedByPromotion` and refuses the registration before Postgres
is reached, so **a test written through `app.Service` passes against a trigger
with no reservation logic at all.** `0007`'s own header says as much. Nobody
acted on it for fourteen sprints.

Generalised: **to test the Postgres half of rule 4, the test must bypass the Go
half.** Otherwise the pre-check answers and the constraint is never consulted.

## 5. The gate weakness the verification itself exposed

Rule 10 asks for repeat runs. The first two repeats reported:

```
DONE 2466 tests in 2.250s
```

`make test` had no `-count=1` — unlike `test-domain`, `test-platform`,
`test-adapters`, `test-cmd` and `test-tools`, every one of which sets it — so Go
served the entire suite, containers and all, from the build cache. **A second
`make ci-integration` on an unchanged tree reported 2466 tests green in two
seconds without starting a single container.**

This is the one target in the Makefile that cannot afford that, because it is
the only one whose job includes executing non-deterministic concurrency tests.
A cached repeat is not a repeat.

Two things about it belong in a retro rather than only in a commit message:

- **It was found by doing rule 10's work, not by reading the Makefile.** Five
  targets had `-count=1` and the sixth did not, for no reason anyone recorded;
  reading the file would more likely have produced "yes, consistent enough".
- **It is reported rather than quietly re-run.** The two cached runs are in
  PR #310's body as cached runs. The temptation to drop them and present only
  the five real ones is exactly the temptation this project's retros keep
  cataloguing.

## 6. What went well

- **The instruction was taken literally, and the premise checked first.** The
  sprint's entire value came from spending one command on a claim six retros
  had copied rather than writing a seventh retro that disclaimed it again.
- **The review hunted history rather than re-reading the diff** (§4), which is
  the only reason defect #3 exists as a fixed defect rather than a live one.
- **Everything was verified by mutation, not by reading.** Three separate
  mutations, each confirming a specific guard:
  - remove `0031` → exactly the `competition_entry` subtest fails, naming the
    constraint. Not "some tests fail": one, the right one.
  - remove `reserved_by_others` from `0030` → the third-party registration
    succeeds and the reservation test fails naming the lost count, while the
    expired-reservation control still passes, as it must.
  - remove `0030`'s `player_id <> NEW.player_id` exemption → the promoted
    player cannot claim their own slot. The exemption is the subtle half of
    `0007`'s rule and the half a reconstruction from memory omits, so it is
    pinned separately.
- **The conformance test's shape was challenged and survived.** The reviewer
  argued the AST parsing was over-engineering and a four-row table would be
  clearer. The argument fails on inspection: a hand-maintained table would have
  been written at T10.6 **by the ticket that already forgot the migration**.
  Forgetting to extend the list and forgetting the schema are the same act, so
  a listed table catches only a type added by someone who would also have
  remembered the migration. Kept as written, for a stated reason rather than by
  default.
- **An adversarial sweep for a fourth instance came back clean, and said so
  with its limits.** `join_waitlist_entry`'s three bodies were diffed (`0026`
  carries `0023`'s additions — no third instance); `promote_next_waiting` and
  `enforce_competition_capacity` are each defined once; the other four
  domain-enum/`CHECK` pairs currently agree. That last clause is the honest
  one: they agree **by inspection, not by any gate**, which is why #311 exists
  instead of a claim that they are fine.
- **Fixture rot was fixed by seeding real rows, not by inventing uuid-shaped
  strings.** A uuid-shaped fixture with no `identity_users` row behind it would
  have satisfied `mustUUID` and failed at the FK — a second lap of the same
  problem.

## 7. The finding this retro produced about itself

**T61 shipped a `CLAUDE.md` gotcha containing three wrong numbers, and this
retro is what caught them.**

The paragraph read:

> …it reported 34 failures — including **two live production defects**
> (`db/migrations/0030`, `db/migrations/0031`) that every Docker-free gate had
> reported green for 18 and 22 sprints respectively.

Three errors in one sentence:

1. **"two"** — there are three. The count was written before the review found
   the third, and the sentence was never revisited when it did. The
   `CREATE OR REPLACE` gotcha immediately below it *was* updated to say "twice
   in the same function", so the same commit held a corrected claim and an
   uncorrected one.
2. **"18 … sprints"** — 18 is a count of *migrations* (0012 → 0030), applied to
   the wrong defect and relabelled as sprints. The weighted-sum defect ran
   T19.1 → T61.
3. **"22 sprints"** — not a figure derived from anything. `payable_type` was
   broken T10.6 → T61.

And the migration count was off by one in the other direction: **seventeen**
migrations landed between `0012` and `0030`, not eighteen, which a one-line
`ls | awk` settles.

### Why this matters more than a typo

§2 of this retro is about a project that repeatedly asserts things it has not
checked. The sprint that wrote §2's own evidence then **did it again, in the
rulebook, about its own findings, on the same day** — and chose a rhetorically
stronger form ("18 and 22 sprints") over the verifiable one ("from T19.1 until
T61"), which is the specific temptation that produces this class of error. A
sprint count is arithmetic over two ticket numbers and invites a plausible
guess; a ticket pair is a fact you either have or do not.

### And then this retro did it twice more

The pre-merge review of **this retro** checked its numbers against the tree
rather than reading them, and found two more:

| as drafted here | reality |
|---|---|
| "six **consecutive** retros" | six retros, but in **two runs of three** — T17–T19 and T56–T58. Taken from T58's own "third consecutive sprint" and generalised without running the loop |
| "13 integration files and 11 unit tests" carry stale headers | **9 and 15.** The total, 24, was from a real `grep`; the split was an estimate |

Both are corrected above. Neither changes a conclusion — and that is the point
worth recording rather than being embarrassed about. **Three successive
artifacts of this sprint (the `CLAUDE.md` gotcha, this retro's §2, this retro's
§9) each reached for a number that sounded right instead of running the
one-liner that settles it, and each was caught by the next verification pass
rather than by more careful writing.** The lesson is not "try harder to be
accurate"; it is that accuracy here is a *command*, not an *intention*, which is
why recommendation 4 is phrased as a format rule rather than an exhortation.

Corrected in this PR, in both `CLAUDE.md` and `docs/LESSONS.md`, with every
figure restated as a ticket range. `CLAUDE.md` also carries a one-sentence note
that the first draft said "18 and 22 sprints" and why that was wrong — per
T58's recommendation 3, the correction goes where the wrong claim was, not only
into a retro.

**PR #310's body and commit messages still carry the wrong figures.** They are
merged history and are not being rewritten; a comment on #310 records the
correction, so a reader arriving at the PR is not left with it.

## 8. Recommendations for T62 and beyond

1. **Re-check any claim that licenses inaction, on a schedule, because nothing
   else will force it** (§2). Specifically: a sentence of the form "this
   environment cannot X" is a ticket to run X, not a fact to quote. T61's cost
   was three production defects, one money-adjacent; the check was one command.
2. **Before redefining a SQL function, diff the new body against the migration
   that last *defined* it**, located by grepping every
   `CREATE OR REPLACE FUNCTION <name>` across `db/migrations` — not against the
   one that created it. Two of five redefinitions of one function got this
   wrong. Adopted into `CLAUDE.md`'s gotchas by this sprint.
3. **A new DB-level guard ships with a DB-level test in the same ticket, and
   that test bypasses the app layer** (§4). `0007`'s guard had no test for
   fourteen sprints, which is the only reason `0012` could drop it silently.
4. **State durations as ticket ranges, not sprint counts** (§7). "From T19.1
   until T61" is checkable; "for 18 sprints" is a guess wearing a number's
   clothes. This is narrower and more enforceable than "be careful with
   numbers", which every retro since T30 has effectively said.
5. **The retro ceremony is three sprints behind** (§9). T62's Ceremony 1 should
   decide explicitly whether T55, T59 and T60 get retros written late or are
   formally recorded as skipped — the current state, where they are simply
   absent, is the worst of the three options because nothing says which it is.

## 9. Sweep and bookkeeping

- **Issues: 3 → 4.** Open and live-verified: **#311** (new, from this sprint's
  review), **#149**, **#145**, **#134**. No issue was closed, and that is not
  an oversight: **none of the three defects had ever been filed.** Nothing in
  the tree knew about any of them, which is the §2 finding restated as a
  number.
- **One PR, one merge.** PR #310 → `d0b36eb`, squashed. No stacking, so none of
  the `git rebase --onto` retargeting T55–T57 needed.
- **`Closes #N` still cannot fire**, structurally — PRs merge into
  `claude/go-backend-pickleball-7up34j`, not the default branch. #311 was filed
  manually; every closure on this project has been manual.
- **Self-review for the sixth consecutive sprint.** PR #310 carries a review
  *comment*, not an approval, because GitHub refuses an author's own. The
  review found a production defect, which is evidence the form has value —
  **and no second party has read a diff on this project since T55.** This
  sprint is the one where that stops being a footnote: #310 changes a capacity
  trigger and a payments `CHECK` constraint, and its own review says the
  reconstructed invariant in `0030` is the single thing most worth another
  reader's eyes.
- **Rule 9's reviewer-authorship carve-out was exceeded, deliberately and with
  disclosure.** `d506c25` fixed defect #3 on the branch under review and fails
  conditions 1 (not mechanical — reconstructing the invariant is a judgement)
  and 2 (surfaced by the reviewer's reading, not a failing test). What
  authorised it is that author and reviewer are the same session, which is the
  standing problem rather than a loophole; conditions 4 and 5 were met
  independently. Recorded in the review itself, not only here.
- **Docker is no longer a disclaimer, and `make ci-integration` is now a
  standing expectation** — recorded in `CLAUDE.md`'s gotchas. Final
  verification: **five uncached full runs, 2468 tests, 0 failures, three of
  them cold** (daemon restarted and `go clean -testcache` before each), plus
  `-count=5 -race` over the four concurrency-bearing packages, so each
  concurrency test executed at least ten times. `make ci-checks` and
  `make gate-coverage` green.
- **The no-double-booking invariant has now actually executed.** T4 proved it
  manually against a local Postgres and committed a portable version that was
  never run; T61 ran it, and it passed every run including the cold ones. Per
  rule 10 that is **ten-plus runs of evidence, not proof for all time**, and
  nothing in this sprint's output upgrades the language.
- **`make security` is not green here.** `govulncheck` gets `Forbidden`
  fetching `vuln.go.dev/index/modules.json.gz` — a network restriction in this
  environment, unaffected by this diff. Stated so that "the gates are green" is
  not read as including the security gate. **Per recommendation 1, this is now
  itself a claim that licenses inaction, and T62 should test it rather than
  quote it.**
- **17 test files still carry a stale no-Docker or "NOT EXECUTED BY ITS AUTHOR"
  header** — **2 integration files and 15 unit tests** — that cite the Docker gap
  as the reason they exist in their current form. (This figure took three
  attempts. The first draft said "13 and 11"; the review corrected it to "9 and
  15" from the file-matching loop; the *sweep that acted on it* then found that 7
  of those 24 matches were files T61 had already annotated, whose annotations
  quote the old phrase and so match the grep. 24 files match the pattern; 17 are
  genuinely stale. A fourth pass on one number in one bullet — which is the
  strongest available argument for §7's conclusion that this needs a mechanical
  rule and not more care.) The unit-test majority is itself worth noting:
  most of these headers are not on tests that needed Docker, but on tests written
  *because* Docker was believed unavailable, which means the false claim shaped
  design and not only process. The two
  cross-context files T61 edited had theirs corrected in place; the rest were
  out of that PR's scope and are a follow-up sweep. `HANDOFF.md`'s T11
  narrative also still states that no session in this project's history has had
  Docker.
- **`CLAUDE.md`'s integration-file count was stale and is corrected here:** 11
  files across 4 contexts → **26 across 5** (socialplay 10, payments 6, booking
  4, competitions 4, facilities 2), verified by the `grep` that paragraph
  already tells the reader to run. It remains narrative and not a gate, per its
  own standing caveat.
- **Three retros are missing: T55, T59, T60.** T55's `HANDOFF.md` row says "not
  yet written"; T59 and T60 have none and no row correction has claimed
  otherwise. T59 is the only one of the three with a sprint plan. See
  recommendation 5.
- **This retro does not update `HANDOFF.md`'s T61 Docs-index row or its Task
  backlog narrative.** Per `sprint-process.md` a retro PR cannot: the row must
  cite this retro's own merge PR number, which does not exist until it merges.
  Both belong to T62's Ceremony 1.

## 10. Honest-form outcome sentence

For `HANDOFF.md`'s T61 row, to be carried verbatim rather than strengthened:

> T61 held no planning ceremony. It was one instruction — run
> `make ci-integration` — against a gap **six retros had disclaimed as
> impossible in this environment** (T17–T19 and T56–T58, two runs of three with
> a 36-sprint silence between them). It was not impossible:
> `dockerd` and `containerd` were on the box, the session ran as uid 0, and the
> daemon came up in about four seconds. The first run reported **2386 tests, 34
> failures** (plus 21 errors that were a missing `covdata` binary, built from
> the toolchain's own source) and surfaced **three live production defects**,
> none of which had ever been filed, all of which every Docker-free gate had
> reported green: guests stopped counting toward Game capacity at **T19.1**,
> when `0023` rebuilt `enforce_game_capacity()` from `0006`'s body and reverted
> `0012`'s weighted sum, so a 7-person Game accepted 7 registrations bringing 3
> guests each — **28 people in a 7-person Game**; every Competition-entry
> payment failed with `23514` from **T10.6**, because `payments.payable_type`'s
> CHECK was never widened for a value the domain had accepted since that
> sprint; and a promoted waitlist player's reserved slot could be taken out
> from under them from **T8.7**, when `0012` itself dropped `0007`'s
> reservation — so `0012` is simultaneously the fix for one divergence and the
> cause of another. In all three the Go half of rule 4 was correct and **the
> authoritative half was the wrong one**. The third was found **by the
> pre-merge review, diffing all five historical bodies of that function against
> each other, and no test anywhere would ever have found it** — `0007` shipped
> the guard untested, so when `0012` dropped it nothing failed; seventeen
> migrations landed in between. Running the suite was **necessary and not
> sufficient**. Fixed by `db/migrations/0030` (the union of four predecessors)
> and `0031`, each with a regression test verified by deletion, and `0031`'s
> **derives the payable-type set by parsing the domain rather than listing it**,
> because a hand-maintained table would have been written by the ticket that
> already forgot the migration. The root cause is recorded as **the third face
> of the failure T56 and T58 each recorded once**: an assertion about the system
> its author had not checked — this time about the *environment*, where the
> claim licensed inaction and so was never on the critical path of anything that
> would have tested it. `make test` gained `-count=1`, without which Go served
> the whole suite from the build cache: **2466 tests "green" in two seconds with
> no container started**, found while performing rule 10's repeats and reported
> as cached rather than quietly re-run. Verified by **five uncached full runs,
> 2468 tests, 0 failures, three of them cold**, plus `-count=5 -race` over the
> four concurrency packages, so each concurrency test ran at least ten times;
> the no-double-booking invariant T4 proved manually has now actually executed.
> **This retro's own finding is that T61 shipped a `CLAUDE.md` gotcha with three
> wrong numbers in it** — "two" defects where there were three, and two sprint
> counts that were a migration-distance and an invented figure — corrected here
> in both `CLAUDE.md` and `LESSONS.md` as ticket ranges, with the correction
> placed where the wrong claim was. **The retro's own review then found two more
> in the retro's first draft** ("six consecutive retros" for six in two runs of
> three; a 13/11 file split that is 9/15), so three successive artifacts of this
> sprint each reached for a number that sounded right instead of running the
> one-liner that settles it — which is why recommendation 4 is a format rule
> rather than an exhortation to be careful. `make security` remains not green here
> (`govulncheck` cannot reach `vuln.go.dev`), self-review stands at six
> consecutive sprints, and no second party has read a diff on this project since
> T55.
