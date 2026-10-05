# T63 Sprint Retro

Ceremony 3 per `docs/process/sprint-process.md`, six-role team (briefs:
`docs/agent-operating-handbook.md` Part B), held against
`docs/process/t63-sprint-plan.md`, `docs/process/t62-retro.md` as the immediate
precedent, PRs #319 and #321, issues #311/#314/#320/#322, and the tree at
`2b118b3` (merged as `e7710af`).

> ## This retro is the first document subject to T63.1's rule
>
> T63.1 adopted, this sprint: **a claim that an action was taken carries the
> command that re-reads the resulting live state** — not the command that
> performed the action, and not a description of it. T63's own retro is the
> first document the rule applies to, so it complies, and §7 reports what
> complying caught.
>
> The short version: it caught **two** stale claims, one of them in the body of
> this sprint's own merged PR, and it is the reason §2 and §3 exist at all. Both
> were found by re-running a command whose answer I already believed I knew.

**Outcome: 4 tickets, 5 `high` advisories fixed, 1 ticket declined on its own
premise, 2 process rules, 2 issues filed.**

```
$ grep -c "^### T63\." docs/process/t63-sprint-plan.md                       # tickets
4
$ git show 9021b76 -- docs/process/sprint-process.md | grep -cE "^\+#{2,4} "   # new rule sections
7                                                                             # under 2 rules
```

The five advisories are named in §3's table; the declined ticket is §4. Merged
as PR #319 (`31000d8`, Ceremony 1) → PR #321 (`e7710af`, all four tickets), in
that order:

```
$ (pull_request_read #319).merged_at ; (pull_request_read #321).merged_at
2026-10-05T11:22:24Z
2026-10-05T13:29:52Z
```

Live issue count after the sprint, which is what T63.1 requires of the sentence
above rather than a list of what was opened and closed:

```
$ (list_issues state=OPEN) -> totalCount
5          # #322, #320, #149, #145, #134
```

---

## 1. What actually happened, in order

1. **Ceremony 1** (PR #319) ran T62.4's corrected escalation sweep for the first
   time, found that **T62's retro had claimed two issues closed when neither
   was**, and closed them.
2. The same ceremony **tested the `govulncheck` claim two retros had only
   quoted**. The claim was true — and its failure had masked the npm half of the
   same gate entirely, hiding **7 advisories, 5 of them `high`**.
3. **T63.2** fixed all five with same-major patch pins, after `npm audit fix`,
   `npm update` and a fresh `npm install` all crashed on this dependency graph.
4. **T63.1 and T63.3** landed the two process rules, in the order the PE pass
   asked for, both touching `sprint-process.md`.
5. **T63.4's premise turned out to be wrong**, and re-verifying it before
   building is what changed the ticket's scope (§4).
6. **This PR's own review found an untested branch in T63.4's code**, which was
   fixed before merge rather than filed (§5a).
7. **Writing this retro found two more things**: a gate whose own failure message
   licenses inaction (§2, now #322), and the fact that the mutation the review
   said did not exist does exist and says something different from what anyone
   expected (§3).

## 2. The finding this retro exists to record

**`make ci-integration`'s own failure message tells the operator that the Docker
gap is documented and expected. It is the eighteenth instance of the claim T61
retired, it sits in the one place no sweep looked, and it fires at the exact
moment someone is deciding whether to run the suite.**

The daemon stopped between integration runs while this retro was being written.
The gate said:

```
$ make ci-integration
No Docker daemon reachable — the integration tests (testcontainers) cannot run here.
This is the documented gap in CLAUDE.md's gotchas, not a new problem.
make: *** [Makefile:375: ci-integration] Error 1
```

`CLAUDE.md`'s gotchas say the opposite, and have since T61: *"**Docker works
here. Start it and run `make ci-integration`.**"* The message sends its reader
to a document that refutes it. What it actually cost:

```
$ (dockerd >/tmp/dockerd.log 2>&1 &) ; for i in $(seq 1 20); do docker ps >/dev/null 2>&1 && { echo "daemon up after ${i}s"; break; }; sleep 1; done
daemon up after 2s
$ make ci-integration | tail -1
DONE 2500 tests in 101.922s
```

**Two seconds and one command, behind a message saying it cannot be done here.**

### Why seventeen copies were retired and this one survived

T61's sweep (PR #313, `bb2bcda`) retired the claim across 17 test files. Its
scope is visible in its own stat:

```
$ git show --stat --oneline bb2bcda | grep -c "_test.go"
17
$ git show --stat --oneline bb2bcda | sed -n '2,4p'
 HANDOFF.md                                              |  6 +++++-
 docs/process/t61-retro.md                               | 15 ++++++++++-----
 .../booking/adapter/grpcapi/discount_regression_test.go | 17 +++++++++++++++++
```

Documents and test files. **No `Makefile`.** The surviving instances, derived
rather than recalled:

```
$ grep -rn "cannot run here\|not a new problem\|documented gap" --include=Makefile --include="*.sh" --include="*.yml" --include=Jenkinsfile . | grep -v node_modules | wc -l
2
```

Both in `Makefile:376-377`, plus a softer third in `make ci`'s success message
(`Makefile:370`) implying this machine has no daemon.

### The generalisation, which is sharper than "sweep more files"

> **A sweep scoped to documents and tests will not reach the claims embedded in
> tooling — and those are the copies a reader meets while deciding whether to
> act.**

A stale sentence in a retro misleads someone reading history. A stale sentence
in a gate's failure path misleads someone who is *blocked right now*, which is
the worst possible moment to be told the blockage is expected. T61 retired
seventeen of the former and left the one instance of the latter.

This is also T61's own recommendation 1 — *a claim that licenses inaction is the
least likely to be re-checked and therefore the most valuable to re-check* —
instantiated in a place that recommendation did not think to look. Filed as
**#322** rather than fixed here, following T61's precedent of a separate sweep PR
(#313) after its retro (#312), so the retro stays the record and the fix stays
reviewable on its own.

## 3. The mutation PR #321's review said did not exist

That review recorded, about T63.3's rule on its first application:

> The five npm `overrides` have **no meaningful mutation** — the rule as written
> doesn't cleanly cover a dependency pin, and I'm flagging that as a limit of
> the rule on its first application rather than claiming compliance I didn't
> earn.

**That was half right and the useful half was wrong.** A mutation exists; it is
just not the obvious one. Run while writing this retro, in a scratch copy so the
real tree was never perturbed:

| mutation | command | `high` findings |
|---|---|---|
| none (the merged tree) | `cd web && npm audit --json` | **0** (2 moderate, 368 deps) |
| revert the **lockfile** to `31000d8` | audit a scratch dir holding `git show 31000d8:web/package-lock.json` | **5** |
| delete the **`overrides` block**, keep the fixed lockfile | `npm install --package-lock-only` then audit | **0** |

```
$ cd "$D" && npm audit --json   # D holds 31000d8's package.json + package-lock.json
PRE-FIX tree (31000d8) tally: {'moderate': 2, 'high': 5, 'total': 7}
pkgs: ['@redocly/openapi-core', '@vitest/mocker', 'brace-expansion', 'js-yaml', 'nanoid', 'undici', 'vitest']
deps audited: 368
```

So the fix is verified: remove it and all five advisories return, by name.

### The third row is the finding

**Deleting the `overrides` block changes nothing today.** `npm install
--package-lock-only` does not downgrade a package that already satisfies its
range, so the already-resolved `nanoid 3.3.18` and friends stay put. The thing
actually holding the five fixes is **the resolved versions in the lockfile**; the
`overrides` block is what *put* them there and is now only a statement of intent.

That is the most dangerous shape a guard can have:

> A declaration whose removal breaks nothing now, and breaks silently at the
> next fresh resolve.

And it compounds with `CLAUDE.md`'s new npm gotcha — a fresh resolve crashes on
this graph, so the day someone regenerates the lockfile is the day both the
pins and the ability to re-derive them are gone at once.

**What this gives T63.3, on its first application, for free:** the mutation for
a dependency pin is **reverting the lockfile, not the declaration**. The review
was right that the rule did not say how; it was wrong that there was nothing to
run. Had the rule been applied properly in the first place, §3's third row would
have been found in the security PR rather than in its retro.

## 4. T63.4: the ticket whose premise was wrong

The plan's instruction 2 asked for a context-ownership check and justified it:
such a check *"catches cross-context mis-mappings, **which is the shape four of
the five errors had**."*

Re-verified before building, per `t62-retro.md` recommendation 4. **Both halves
of that sentence were wrong:**

1. The five errors were **three nonexistent files and two nonexistent types** —
   and `DeclaredConstants` already errors rather than returning an empty set, so
   all five were already loud. None was a cross-context mis-mapping.
2. The count of rows that cross a context boundary today is **zero**:

```
$ python3 - <<'PY'   # parse enumPairs, group each row's declaring context by table
rows parsed: 22
 booking: ['bookings', 'discount_rules', 'pricing_rules', 'recurring_hire_templates']
 competitions: ['competition_entries', 'competitions']
 payments: ['payments']
 socialplay: ['games', 'registrations', 'waitlist_entries']
PY
```

Every table maps into the context that owns it. So the check would have been
machinery for a failure that has never occurred, and it would have needed a
hand-maintained `table → context` list — **the exact artifact
`tools/enumconformance` exists to avoid**, and the one `CLAUDE.md` says of
`gate-coverage` is "the one change that would defeat it".

**Declined, with the reasoning written into the code** rather than the ticket
closed quietly. What was built instead is narrower and answers a real gap:
`Status` is declared in four bounded contexts, so `ValidateMapping` **reports**
every type name used by more than one row, naming each row's file, and fails only
on the two machine-decidable errors. Seven tests cover it:

```
$ grep -c "func TestValidateMapping\|func TestVerify" tools/enumconformance/enumconformance_test.go
7
```

**The lesson is not "the plan was sloppy".** I wrote that plan, two days ago,
with the evidence in front of me. The failure is that *"the shape four of the
five errors had"* is a quantity, it carried no command, and T62.2 had been in
force for a sprint. A figure with a command attached would have been checked
when it was written instead of when it was built.

## 5. The sprint corrected three of its own claims

### 5a. The review of this sprint's PR found an untested branch in it

`ValidateMapping` was wired into the Docker-only conformance test with a bare
`t.FailNow()`, and nothing asserted that the comparison is skipped when the
mapping does not hold. By **T63.3's own rule, adopted in the same PR**, that is
an untested branch — and it sat where no Docker-free gate could reach it, with
`ci-integration` only ever exercising the branch where the mapping was already
sound.

Fixed before merge (`2b118b3`): the ordering moved into `enumconformance.Verify`,
whose `Result.Compared` field separates *"nothing was wrong"* from *"nothing was
examined"*, with `OK()` requiring both — the vacuous-green failure this project
has now shipped three sprints running, written into a type instead of a comment.

**Two tests, not one.** Without the control
(`TestVerifyComparesWhenTheMappingHolds`), a `Verify` that short-circuited
unconditionally would satisfy the test meant to prove the short-circuit.

### 5b. The guard-removal check refuted my prediction of what it would show

T63.3 says to remove the guard and record what fails. The first draft of that
test's comment predicted the removal would fail three named assertions. It does
not:

```
$ # with the early return in Verify deleted
--- FAIL: TestVerifyDoesNotCompareAgainstABrokenMapping
    enumconformance_test.go:360: a broken mapping is a finding, not an error: found no Nope constants in nope.go
```

The assertions are never reached: `Compare` propagates the unresolvable row as a
Go **error**, so the caller learns about one row instead of receiving a finding
list naming every broken one — which is a better argument for the guard than the
one I invented. The comment now quotes the real failure and says the removal is
what corrected it.

**This is the fourth time this sprint that the reasoned answer was wrong and the
run was right**, and it happened inside the ticket that adopted "a guard is
verified by removing it". That is a point in the rule's favour: a rule whose
first application corrects its own author is doing work.

### 5c. A "clean" npm audit over zero dependencies, nearly accepted

While investigating the resolver crash, `npm audit` reported no vulnerabilities
on a tree with no `node_modules`. It was auditing nothing. Caught by reading the
tally alongside what it had loaded:

```
$ cd web && npm audit --json | python3 -c "...; print(d['metadata']['dependencies']['total'])"
368
```

**Third instance of vacuous-green in three sprints** — after #308's hand-written
subject lists and T62.5's parser that matched 0 of 56 constraints
(`t62-retro.md` §5c, where that run's output is recorded). The pattern is
stable enough now to state as a rule: *a check's result is meaningless until you
know how much it examined*, which is why both of this project's new gates print
their own denominators.

In the same investigation I **deleted `web/package-lock.json`**, after which
nothing could resolve at all. Restored from a copy taken before the first change
and verified with `npm ci` and a clean `git status`. Recovery by luck rather than
by process — see §8 recommendation 2.

## 6. What went well

- **Testing a claim that licensed inaction paid for the whole sprint.** T61's
  recommendation 1 said to re-check exactly this class of claim; T61's and T62's
  own retros then recorded `govulncheck` as "owed" without testing it. T63's
  Ceremony 1 tested it, and the five `high` advisories behind it are the most
  valuable thing this sprint shipped. **The claim was true** — and being true was
  not the point, because its failure had hidden the half of the gate that works.
- **The PM value ordering was followed and was right.** T63.2 was listed first
  because its cost accrues while it waits; it was also the only ticket whose
  subject was a live exposure.
- **The minimal fix was taken over the newest one.** Every one of the five
  advisories had a patch-level fix inside its current major line, found by
  reading each advisory's `range` rather than reaching for `latest` — which would
  have meant 5.x/6.x/8.x/2.x for four of them. No breaking bump, so the plan's
  standing instruction about stopping if a bump breaks a web test never had to
  fire. Web tests were identical either side:

  ```
  $ npm run test:unit -- --run   # before and after
  Test Files  61 passed (61)
       Tests  717 passed (717)
  ```
- **The blocked half was filed, not baselined.** #320 records the two `vitest`
  moderates with the full reproduction table, why `--force` is the wrong lever,
  and three things that would close it. `tools/vulngate` already treats them as
  below threshold, so a baseline entry would have added nothing and suppressed
  the signal if they were ever re-rated.
- **Re-verifying a premise before building to it saved a hand-maintained list**
  (§4). Second consecutive sprint in which `t62-retro.md` recommendation 4 has
  prevented a wrong artifact: at T62 it would have rewritten eleven ADRs, here it
  would have added the one kind of list this tool exists to avoid.
- **`gate-coverage` and `docs-index-check` both stayed green with no edit**, on a
  sprint that added a package-level API and three documents:

  ```
  $ make gate-coverage docs-index-check | grep -E "OK|hold test"
  gate-coverage: 47 package(s) hold test functions.
  gate-coverage: OK — all 47 package(s) with runnable tests are executed by "ci-checks".
  docs-index-check: OK — the Docs index, the narratives and every ADR status agree with the tree.
  ```
- **Two rules landed in the order the PE pass asked for.** T63.1 then T63.3, both
  in `sprint-process.md`, no conflict. Seven new subsections between them:

  ```
  $ git show 9021b76 -- docs/process/sprint-process.md | grep -cE "^\+#{2,4} "
  7
  ```

## 7. What T63.1's own rule caught, measured on this document

T62.2's retro measured what the quantity rule *cost*. T63.1's extension is
cheaper to satisfy, because the claims it governs are the ones a sprint is most
confident about — and confidence is cheap to re-check and expensive to leave
unchecked.

**Four of the five rows below were found by writing this document under the
rule. The second predates the rule** — Ceremony 1 found it, which is what
produced T63.1 — and is listed so the table reads as the rule's case rather
than only its yield.

| claim | the command it now carries | what re-reading found |
|---|---|---|
| "`ci-integration` green, 2498 tests" (PR #321's body) | re-run it | **stale on the merged head** — 2498 describes `9021b76`; the head is **2500** |
| "issues 5 → 3" (T62's retro, corrected at T63's Ceremony 1) | list open issues | the original claim was false; the live count was 5 |
| "Docker cannot run here" (`Makefile:376`) | start the daemon | **false**, 2 seconds to refute (§2, #322) |
| "no meaningful mutation exists" (PR #321's review) | run one anyway | a mutation exists and says something else (§3) |
| "108 process docs" (this retro's own first draft) | `make docs-index-check` | **109** — the document making the claim is counted by the gate verifying it |

The first row is the one worth keeping. **PR #321's body was true when written
and false when merged**, because the follow-up commit that closed its own
review's finding added two tests to the same PR. Nobody lied and nothing was
sloppy; the number simply described a tree that stopped existing inside its own
pull request. An action-claim is stale the moment anything downstream of it
moves, which is why the rule asks for the command that re-reads rather than the
figure.

The fifth row is the cheapest possible demonstration and arrived unprompted:
the retro's first draft quoted the gate's own output, correctly, and adding the
retro to the tree invalidated it before the file was committed. **A figure can
go stale between writing a sentence and saving the file**, which is an argument
for carrying the command and against carrying the number at all where the
command's output would do.

**The limit, stated as T63.1 itself states it:** this rule cannot make a session
run the command. It can only make the omission visible on the page. Three of the
four rows above were found because *writing a retro under the rule* forced the
re-run — not because any gate caught them. That is the rule working exactly as
advertised and no better.

## 8. Recommendations for T64 and beyond

1. **When a claim is retired, derive the sweep's scope rather than listing it**
   (§2, #322). T61 swept documents and tests and left the copy that fires inside
   a gate. The question "which files can utter this claim?" has a mechanical
   answer — a repo-wide grep, filtered — and this project already knows what a
   hand-written list is worth.
2. **Give `web/package-lock.json` a mechanical guard.** It is load-bearing
   (§3: it, not `package.json`, is what carries the five fixes), a fresh resolve
   cannot regenerate it on this graph, and this sprint's recovery from deleting
   it depended on a backup copy I happened to take. `Makefile:153` already runs
   `npm ci` when `node_modules` is absent, which means CI would notice a
   lockfile that no longer satisfies `package.json` — but **nothing notices the
   lockfile going missing or being regenerated**, which is the failure that
   actually happened.
3. **T63.3 needs the dependency-pin clause §3 supplies**: the mutation for a
   version pin is reverting the **lockfile**, not the declaration, because
   removing the declaration is a no-op against an already-resolved tree. The
   *finding* belongs in `CLAUDE.md`'s npm gotcha — a future session needs it
   whether or not the rule is amended — and goes in its own PR, for the reason in
   §9's scope bullet. **The rule clause itself is deliberately left to T64**,
   because a rule amended to fit the first document judged against it should have
   a second reader, which is `t62-retro.md` recommendation 5's own reasoning: the
   one T63.1 overruled on its merits.
4. **A quantity in a *plan* gets the same scrutiny as one in a retro** (§4).
   T63.4's premise — "the shape four of the five errors had" — was a quantity
   with no command, in a document written one sprint after the rule requiring
   one, and it survived the plan's own PE pass. The plan is where a wrong figure
   is cheapest to catch and most expensive to miss, because tickets get built
   from it.
5. **The eleventh consecutive self-reviewed sprint.** Stated again rather than
   dropped: both PRs carry comment-reviews, GitHub refuses an author's own
   approval, and this sprint merged a dependency change on the shipped web client
   and two process rules with no second reader. §3 is the concrete cost — the
   review's "no meaningful mutation" claim went unchallenged until the retro,
   because there was nobody to challenge it.

## 9. Sweep and bookkeeping

- **Issues: 3 → 5.** #311 and #314 closed at Ceremony 1 (the correction T62's
  retro needed); #320 filed by T63.2 for the blocked `vitest` moderates; #322
  filed by this retro for §2. Live, per T63.1's own rule rather than by arithmetic:

  ```
  $ (list_issues state=OPEN) -> totalCount
  5          # #322, #320, #149, #145, #134
  ```

  **#149, #145 and #134 remain blocked on things no coding session can supply** —
  a read port into Booking that is a genuine design change, a real IdP tenant,
  and a human with a screen reader. The premise check fired on zero of five at
  Ceremony 1, for the first time, because T62.3 corrected the two that had
  drifted.
- **Merge order #319 → #321**, verified against each PR's `merged_at` (see the
  header), squash-merged as `31000d8` and `e7710af`.
- **`Closes #N` still cannot fire** against a non-default base branch. Every
  closure this sprint was manual, as every closure on this project has been —
  which is the mechanism behind T62's false claim and is now carried in T63.1's
  worked examples.
- **Verification.** `make ci-checks` green: `gate-coverage` 47/47,
  `docs-index-check` OK across 64 rows / **109** process docs / 17 ADRs — 108
  before this document existed, which is §7's fifth row — `fmt-check`
  and `golangci-lint` clean, web 61 files / 717 tests, `build-web` green.
  `make ci-integration` green on the final tree **four times — 2500 tests each**
  (93.2s, 101.9s, 96.6s, 96.3s). One was cold in both senses — a just-started
  daemon and a cleared `go test` cache — and the two attempts between were not
  runs at all, because the daemon had stopped: that is §2's evidence, met rather
  than read.

  ```
  $ go clean -testcache && for i in 1 2; do make ci-integration | tail -1; done
  DONE 2500 tests in 101.922s
  DONE 2500 tests in 96.586s

  $ make ci-integration | tee /tmp/ci-int-full.log | tail -1
  DONE 2500 tests in 96.283s
  $ grep -cE "^(FAIL|--- FAIL)|failures" /tmp/ci-int-full.log
  0

  $ SKIP_GOVULNCHECK=1 make security | tail -1
  PASS: no new gating findings (0 baselined, 2 below threshold).
  ```

  **The fourth run exists because "0 failures" was an inference.** `gotestsum`'s
  `DONE` line omits the failure count when it is zero, so three runs' tails were
  evidence of a completed suite and not, strictly, of a clean one. The fourth
  captured the full output and was grepped for failure lines. That is a small
  instance of the same discipline as the rest of this document: the claim was
  almost certainly true, and checking it cost 96 seconds.
 Per rule 10 that is evidence, not proof
  for all time.
- **Mutation verification, per T63.3, on the sprint that adopted it.** Two, both
  with their output quoted: `Verify`'s short-circuit (§5b, which refuted the
  prediction made for it) and the npm pins (§3, which found that the declaration
  is not the guard). The second was run **for this retro, not for the PR**, which
  is the rule's first application being late rather than absent — recorded that
  way deliberately.
- **`govulncheck` is now tested rather than quoted, and stays unrun here.**
  `vuln.go.dev` is `Forbidden` from this environment; `SKIP_GOVULNCHECK=1` is the
  Makefile's designed response, it warns loudly, and it must never appear in a
  `ci-*` target. That is in `CLAUDE.md`'s gotchas now, so the third sprint in a
  row does not re-derive it. **The Go half of the security gate has still never
  run on this project** — that is a real gap, it is now a documented one, and it
  is not solvable from inside this container.
- **Two new `CLAUDE.md` gotchas**, both earned the hard way: the security gate's
  broken half is not permission to skip the whole gate, and npm's own remediation
  commands crash on this dependency graph (with the lockfile warning and the
  vacuous-audit warning folded in).
- **This retro sets its own row's Retro cell to the retro's path and leaves
  Reviews alone**, per the path/number distinction `sprint-process.md` records —
  T62's retro being the worked example that forced it. T64's Ceremony 1 owes the
  **Reviews** cell (#319 → #321, `merged_at`-verified) and the **narrative**, and
  should also check whether T63.1's and T63.3's rules are being followed by a
  sprint that did not write them, which is the only evidence that counts.
- **What this retro's own PR changes, stated so the diff is not larger than the
  record implies:** the retro itself; `HANDOFF.md`'s T63 **Retro cell only**; and
  a `docs/LESSONS.md` entry for §2 and §3, since both are postmortem-shaped and
  that file is where this project keeps recurring-mistake generalisations. No
  code, no gate, no rulebook — the three changes §8 asks for (#322's Makefile
  fix, `CLAUDE.md`'s npm sub-bullet, and T63.3's dependency-pin clause) are each
  left to a reviewable change of their own.

  **The `CLAUDE.md` sub-bullet was in this PR and was removed, for a mechanical
  reason worth recording** — it is the one genuinely new thing this project has
  learned about its own branch topology. Every PR here merges into
  `claude/go-backend-pickleball-7up34j` **squashed**, so a branch cut from the
  previous PR's tip shares no commit with the squashed result. Git then 3-way
  merges from the grandparent, and a file both sides touched in the same region
  conflicts even though **the two trees are byte-identical**. Reproduced rather
  than guessed, using a stand-in commit with the merged tree and the same parent:

  ```
  $ T=$(git rev-parse 2b118b3^{tree}); E=$(git commit-tree "$T" -p 31000d8 -m stand-in)
  $ git merge-tree --write-tree --name-only HEAD "$E"
  CLAUDE.md
  CONFLICT (content): Merge conflict in CLAUDE.md
  ```

  Exactly one file, and the reasoning predicted it: PR #321 touched `CLAUDE.md`
  and nothing else this PR touches. So the narrower scope is not a judgement
  call dressed up as one — the alternative was a conflict resolution on a
  rulebook file, which is the last file to resolve by hand.
- **One thing this retro deliberately did not do:** fix `Makefile:376-377`. It is
  a two-line change and the temptation to fold it in was real. It went to #322
  instead, following T61's precedent (retro #312, sweep #313), so that the retro
  stays the record of the sprint and the fix gets reviewed as a change to a gate.

## 10. Honest-form outcome sentence

For `HANDOFF.md`'s T63 row, to be carried verbatim rather than strengthened:

> T63 was the sprint in which **testing a claim that licensed inaction paid for
> the whole sprint**. Two retros had recorded that `make security`'s
> `govulncheck` cannot reach `vuln.go.dev` and had recorded it as *owed without
> testing it*; T63's Ceremony 1 tested it, found the claim **true**, and found
> that its truth had masked something: `security-go` runs before `security-npm`,
> so **the npm half of the gate had never run at all**, concealing **7
> advisories, 5 of them `high`**, in shipped web dependencies. T63.2 fixed all
> five with **same-major patch pins** read off each advisory's own `range` —
> `latest` would have meant a major bump for four of them — after `npm audit
> fix`, `npm update` and a fresh `npm install` all crashed reproducibly with
> `Cannot read properties of null (reading 'edgesOut')`; the two remaining
> `vitest` moderates are **below the gate's own threshold**, dev-only, and filed
> as #320 rather than baselined. **T63.4's premise turned out to be wrong**: its
> plan asked for a context-ownership check on the grounds that cross-context
> mis-mapping was *"the shape four of the five errors had"*, and re-verifying
> first showed all five were already loud (three nonexistent files, two
> nonexistent types) and that **zero** rows cross a context boundary — so the
> check would have been machinery for a failure that has never occurred, needing
> the one hand-maintained list that package exists to avoid. Declined with the
> reasoning in the code, and what was built instead reports the genuinely
> ambiguous case (`Status` is declared in four bounded contexts) rather than
> failing on it. **This PR's own review found an untested branch in that same
> code** — the mapping short-circuit was a `t.FailNow()` behind a build tag only
> Docker could reach — and it was fixed before merge, with `Result.Compared`
> separating *"nothing was wrong"* from *"nothing was examined"*. **The
> guard-removal check then refuted the prediction written for it**: the removal
> fails earlier and differently than its own comment claimed, which is the fourth
> time this sprint the reasoned answer was wrong and the run was right, inside
> the ticket that adopted "a guard is verified by removing it". Writing the retro
> under T63.1's new rule caught two further stale claims: **PR #321's body was
> true when written and false when merged** (2498 tests against the merged head's
> 2500, because the commit closing its own review finding added two), and
> **`make ci-integration`'s own failure message tells the operator the Docker gap
> is "the documented gap in CLAUDE.md's gotchas, not a new problem"** when
> `CLAUDE.md` has said the opposite since T61 — met live, refuted in **two
> seconds** by one command, and the eighteenth instance of a claim T61's sweep
> retired seventeen times without ever looking at the `Makefile`; filed as #322,
> deliberately not fixed in the retro. The mutation that PR #321's review called
> non-existent for a dependency pin **does exist and says something else**:
> reverting the lockfile returns all five advisories by name, while deleting the
> `overrides` block changes nothing at all — so **the lockfile, not the
> declaration, is the guard**, which is the most dangerous shape a guard can
> have. Verified by `make ci-checks` green (`gate-coverage` 47/47,
> `docs-index-check` across 64 rows / 109 docs / 17 ADRs), `make ci-integration`
> green four times on the final tree (**2500 tests each**, one cold in both
> senses, the fourth's full output grepped for failure lines because "0
> failures" had until then been an inference from `gotestsum`'s summary line),
> and
> `SKIP_GOVULNCHECK=1 make security` **PASS**. Issues 3 → 5 (two closed, two
> filed), live-counted rather than derived. Eleventh consecutive self-reviewed
> sprint, and §3 is the concrete cost of that: the review's "no meaningful
> mutation" went unchallenged until the retro, because there was nobody to
> challenge it.
