# T64 Sprint Retro

Ceremony 3 per `docs/process/sprint-process.md`, six-role team (briefs:
`docs/agent-operating-handbook.md` Part B), held against
`docs/process/t64-sprint-plan.md`, `docs/process/t63-retro.md` as the immediate
precedent, PRs #325 and #326, issues #322/#320/#145, and the tree at `2e9b59f`.

> ## This sprint had a second reader, for the first time in twelve sprints
>
> Every sprint from T53 onward has been self-reviewed: the author posts a
> comment-review on their own PR, because GitHub refuses an author's own
> approval. Each retro has recorded that as the standing weakness.
>
> T64's retro was written while **three report-only reviewer agents** audited
> the sprint in parallel — one adversarial QA pass over the merged diff, one
> compliance audit of T64's own documents against this project's four evidence
> rules, and one derived stale-claim sweep over `CLAUDE.md` and `HANDOFF.md`.
> §8 records what they found and §9 is honest about what that is worth: a
> second pass is not a second person, and an agent given the author's framing
> inherits the author's blind spots. It is still the first time this project's
> work has been read by something that was not the thing that wrote it.

**Outcome: 5 tickets, 3 `high` advisories fixed, 1 issue closed, 2 new rules,
1 new gate, 5 mutations.** Merged as PR #325 (`1febf8e`, Ceremony 1) → PR #326
(`2e9b59f`, all five tickets):

```
$ gh api .../pulls/{325,326} --jq '"#\(.number) \(.merged_at) \(.merge_commit_sha[0:7])"'
#325 2026-10-06T07:14:07Z 1febf8e
#326 2026-10-06T07:44:55Z 2e9b59f
```

Live issue count after the sprint, per T63.1's rule rather than by arithmetic:

```
$ gh api "repos/.../issues?state=open" --jq '[.[] | select(.pull_request == null)] | length'
4          # #320, #149, #145, #134
```

---

## 1. What actually happened, in order

1. **Ceremony 1** (PR #325) ran the security gate **before** planning rather
   than quoting the previous sprint's result, and found it **red on a tree
   nobody had touched** (§2).
2. The same ceremony found that T62.4's *corrected* escalation sweep had still
   missed a product question, because it reads an issue's labels and state and
   the question was a sentence in its body (§3). Put to the Product Owner and
   answered: defer, **with a trigger a future sprint can cause**.
3. The ceremony's own PR **could not merge** — a squash-ancestry conflict on
   `HANDOFF.md` against byte-identical content — and resolving it required a
   permission this session had been denied earlier (§6).
4. **T64 execution** (PR #326) delivered all five tickets, with five mutation
   checks.
5. **T64.2's derived sweep found two live instances the issue that prompted it
   had not predicted** (§4).
6. **T63.3's rule, whose first application had failed, was repaired and then
   immediately applied** — in the same sprint, to the same class of change
   (§5).
7. Three self-corrections, two of them figures this sprint asserted before
   computing (§7).

## 2. The finding this retro exists to record

**`make security` went red on a tree nobody had touched, twenty hours after
T63.2 left it green.**

```
2026-10-05 $ SKIP_GOVULNCHECK=1 make security | tail -1
PASS: no new gating findings (0 baselined, 2 below threshold).

2026-10-06 $ SKIP_GOVULNCHECK=1 make security | tail -5
NEW gating findings (3):
  - [npm-audit] @vue/server-renderer (high): XSS via missing CR in attribute-name blacklist
  - [npm-audit] source-map-js (high): event-loop DoS via indexed source-map section offsets
  - [npm-audit] vue (high): via @vue/server-renderer
FAIL: 3 new gating finding(s).
```

`vue` is a **direct** dependency and the framework the client ships; the XSS
advisory carries CVSS 7.2 and the DoS 7.5. All three had same-major fixes, so
T64.1 closed them in one ticket.

### Why this is a finding and not just a ticket

Nothing in T63 was wrong. The code did not change, the pins did not slip, the
gate was not misconfigured, and T63's verification was honest and carried its
command. **The advisory database moved.** So:

> **A green gate is a statement about a moment, not about a tree.**

That is a sharper claim than T63.1's rule, and the difference is the useful
part. T63.1 asks for a live-state command on the reasoning that *the actor's
own account of what they did* is the unreliable part — T62's retro asserted two
issues closed when neither was. **Here the account was accurate and the world
moved underneath it.** A re-read is therefore owed even when nobody doubts the
claim, and the signal that it is owed is a **date**, not a suspicion. T64.3
wrote that in as the cheapest rule this project has adopted: one word beside a
claim that already carries a command.

### What it re-prices

T61's and T62's retros both recorded "govulncheck is owed" without running the
gate, and T63's ceremony then found **five** `high` advisories behind that
deferral. **The same deferral would have cost three more in a single day.** The
number that matters is not how many findings a run produces — it is how fast
the answer goes stale. T63.2's green lasted under a day.

Hence the operational half, which is what T64's ceremony actually did:
**run the security gate at the start of a ceremony, not at the end of a
sprint.**

## 3. The corrected sweep still missed a question, and the reason is structural

T62.4 rewrote the escalation sweep after T62's ceremony found it unsound three
ways, the live one being that a **label**-keyed sweep could not see #314. The
corrected rule says: consider **every open issue**, not every labelled one.

T62's and T63's ceremonies both ran the corrected version. Both reported that
no open issue awaited a product decision. **Both were wrong:**

> **#145 has carried the sentence *"Needs product input on which"* since
> 2026-08-14.**

The issue is unlabelled, its state is `open`, and nothing about it *as an
object* says a decision is pending. The question is a sentence in its body.

**The pattern is one level of derivation at a time, and each fix stops one
level short:**

| sweep keyed on | missed | because |
|---|---|---|
| a `role:product-owner` label (pre-T62.4) | #314 | a label is a thing someone must remember to add |
| the open-issue list (T62.4) | #145 | an issue's labels and state do not say what its prose says |
| the issue's prose (T64.5) | a question with no recognisable marker | no wording list is complete |

Each step is a genuine improvement and none of them is the last one. T64.5's
step is honest about being the third rather than the final: what it changes is
that the reading is **obligatory and written down per issue**, so a ceremony
must say "no question in this body" in so many words rather than reporting
"none awaiting a decision" as a property of the issue set. T62's and T63's
ceremonies could each have written that sentence about #145 only by reading it
and deciding wrongly; **neither had to read it at all.**

### The answer, and why its shape matters more than its content

Put to the Product Owner before any ticket was refined, per the rule, and
answered: **defer until the identity provider is chosen, with the trigger that
the IdP-provisioning ticket must answer it before it merges.**

DECISION D1 sat escalated for 41 sprints with a trigger conditioned on an event
*no ceremony could cause* — "the sprint immediately following the user's
answer". A deferral whose trigger a future sprint can cause is a decision. One
whose trigger nobody can cause is the 41-sprint failure wearing a decision's
clothes. Recorded in #145's body, above the T62 correction block, because
`t59-retro.md` §3 established that a comment is the thing nobody re-reads.

## 4. The derived sweep found what the issue that prompted it did not

#322 named the `Makefile` and, in passing, `make ci`'s closing line. It also
asserted that its grep "confirms this is the only surviving instance in
non-doc, non-test files."

**That was true of the three phrases it searched.** Widening the search to the
claim's other forms — *"Docker daemon available"*, *"on a machine with a Docker
daemon"* — found two more live instances:

| file | what it said | disposition |
|---|---|---|
| `README.md` | *"a large share of this project's development environments have no Docker daemon"* | rewritten — the first thing a new reader meets |
| `Jenkinsfile` | `agent any` justified by *"sandboxes that repeatedly turn out to have no Docker daemon"* | **premise corrected, decision kept** |

So the rule earned itself on its first application, and in the most useful way:
**the issue that argued for deriving the scope had itself listed the scope, and
its list was incomplete.** A sweep is only as wide as its search terms, which
is now part of the rule's text.

The `Jenkinsfile` case is the one worth keeping. `agent any` is still the right
choice — a Jenkins agent may genuinely lack Docker — but the *reason* recorded
beside it was this project's false belief about its own sandboxes. **A decision
can survive its own rationale**, and the honest repair is to correct the
rationale rather than reopen the decision or leave a false premise in place as
load-bearing commentary.

### Classifying is half the rule

Of 103 matching lines in 64 files, **three** were edited. The other hundred are
retros recording what was believed at the time, the 17 test headers T61
rewrote to quote the claim beneath its refutation (the assert-the-positive
pattern), two migration comments that date themselves by their own file number,
and `CLAUDE.md`'s refutation, which necessarily contains the words.

> **A sweep that edited every match would destroy this project's own record of
> having been wrong**, which is what these documents are for.

That is why the rule's output is a classification table rather than a diff
count.

## 5. A rule whose first application failed, repaired and re-applied in one sprint

T63.3 — "a guard is verified by removing it" — was applied to T63.2's npm pins
and **the application failed.** PR #321's review recorded that a dependency pin
had "no meaningful mutation", flagging it as a limit of the rule. T63's retro
ran one anyway and found the limit was imaginary and the finding was not:

| mutation | `high` findings |
|---|---|
| revert `web/package-lock.json` | all five, by name |
| delete the `overrides` block, keep the lockfile | **none** |

T64.3 turned that into a clause — **mutate the lockfile, not the
declaration** — and T64.1 then applied it the same day:

```
HEAD package.json + HEAD lockfile  -> high: [@vue/server-renderer, source-map-js, vue]
NEW  package.json + HEAD lockfile  -> high: [@vue/server-renderer, source-map-js, vue]
the fixed tree                     -> high: []     (2 moderate, 368 deps audited)
```

The middle row is the clause earning itself: **the new declaration against the
old lockfile fixes nothing at all.**

**The loop closed inside one sprint** — rule applied, application failed,
failure found by a retro, clause written, clause applied.

**The first draft of that paragraph then claimed "every previous process rule
took two or three sprints between the finding and the fix", which is false.**
Derived from the rulebook's own adoption lines rather than remembered:

```
$ grep -n "Adopted at T\|Adopted T" docs/process/sprint-process.md
```

| rule | finding | adopted | lag |
|---|---|---|---|
| escalations raised before planning | `t54-retro.md` rec 1 | T55 | 1 sprint |
| the 0-ticket cap | `t54-retro.md` rec 2 | T55 | 1 |
| premise check | `t56-retro.md` recs 1–2, `t58-retro.md` | T59 | 3 |
| quantity rule | `t61-retro.md` rec 4 | T62 | 1 |
| action-claim rule | T63's own ceremony §2 | T63.1 | **0** |
| mutation rule | `t60-retro.md` rec 1, deferred at T62 | T63.3 | 3 |
| sweep-scope rule | `t63-retro.md` §8 rec 1 | T64.2 | 1 |

So the lag has ranged 0 to 3 and a same-sprint adoption already happened at
T63.1. **What is new at T64 is narrower and still worth keeping: the first time
a rule's *failed application* was found and the rule repaired, rather than a
finding being turned into a rule.** `t56-retro.md`'s complaint about rules
becoming wallpaper is about the second kind of lag, not the first.

## 6. The mechanic that stopped a correct PR from merging

PR #325 was complete, reviewed and verified, and **GitHub refused to merge
it**: one conflict hunk in `HANDOFF.md`.

The cause is structural and will recur. Every PR here squash-merges into
`claude/go-backend-pickleball-7up34j`, so a branch cut from the previous PR's
tip shares **no commit** with the squashed result. Git then 3-way merges from
the grandparent, and a file both sides touched conflicts **even when the two
trees are byte-identical**. Proved rather than assumed, with a stand-in commit
carrying the merged tree and the same parent:

```
$ git merge-tree --write-tree --name-only HEAD "$E"
HANDOFF.md
CONFLICT (content): Merge conflict in HANDOFF.md
$ git diff --stat ae5c287 67cdf21        # the base's tree vs the branch's pre-edit tree
                                         # (no output: identical)
```

And the resolution was **provably safe before it was applied** — the two T63
rows differed in exactly one cell, the Reviews cell this ceremony exists to
fill:

```
Retro cell identical: True
Only differing cell is index 3 (Reviews): [3]
```

### What this cost, and the part worth recording

The safe routes were tried and exhausted first: GitHub's own branch update
(conflict), then writing the two files onto a fresh branch cut from the base
via the contents API (refused — *"Write access to this GitHub API path is not
permitted through this proxy"*), then the placement trick that had saved PR
#324 a day earlier, which cannot work when the file you must edit *is* the
conflicted one.

What unblocked it was **a permission this session had been denied earlier** —
a single `git fetch` of the shared branch, denied once with the reason "Merge
Without Review", then granted when asked with the reason stated. Two things
follow:

1. **A denial of a mechanism is not a denial of a purpose.** The same fetch
   that looks like "merge without review" is also the only way to rebase a
   reviewed PR onto a reviewed base. The right response was to exhaust the
   alternatives, state precisely what was needed and why, and ask — not to
   route around it.
2. **PR #324's placement workaround was luck, not technique.** It worked
   because the file was `CLAUDE.md` and the addition could sit twelve lines
   away from the previous PR's hunk. It is worth recording as a trick with a
   narrow precondition rather than as the answer.

## 7. Three self-corrections, two of them figures asserted before being computed

### 7a. "66 lines in 63 files" was invented

T64.2's own section — the one adopting *derive the scope, don't list it* —
first asserted a scope figure nobody had computed, written while looking at a
per-file listing and never totalled. Deriving it:

```
$ cd <pristine checkout of 1febf8e> && grep -rIn --exclude-dir=node_modules <7 patterns> . | wc -l
103        # across 64 files
```

**A sweep-scope rule whose own scope figure was invented would have been a poor
advertisement for itself.** Corrected in the committed text, not only here.

Note the second-order detail, which is now in the section: the count is taken
against a **pristine checkout**, because the write-up matches its own search
terms and the number rises as it is written. Same shape as `docs-index-check`
counting the retro that cites it (`t63-retro.md` §7).

### 7b. The gate was right and the measurement was wrong

T64.4's mutation table first read **exit 141** for a failing `lock-check`,
which looks like a gate failing in some third way. 141 is SIGPIPE: the
measurement piped `make` through `head`. Without the pipe the codes are 2, 2,
0.

> A check whose failure code you have mis-measured is indistinguishable from
> one that does not fail.

Third instance this project has recorded of **the measurement being the broken
thing** — after T62's `cp` into a mode-555 `GOROOT` that reported success and
copied nothing, and T63's `DONE` line that proves a suite *completed* rather
than that it was *clean*.

### 7c. A live-state read, taken too soon, was stale

Immediately after closing #322, the live issue count read **5**. A second read
moments later read **4**. Nothing was wrong except the timing: GitHub's list
endpoint had not caught up with the write.

That is a genuine qualification on T63.1, which this retro is the first
document to hit: **a live-state command run in the same breath as the action it
verifies can return the pre-action state.** The remedy is not a rule change but
a habit — read it twice, or read the object rather than the list (`gh api
issues/322 --jq .state` was correct immediately). Worth one sentence in the
rule, and T65 can decide whether to add it.

## 8. What the reviewer agents found

**Three report-only reviewer agents** (rule 9: they report, they never commit)
audited this sprint in parallel with the retro being written: an adversarial QA
pass over the merged diff, a compliance audit of T64's own documents against
the four evidence rules, and a derived stale-claim sweep over `CLAUDE.md` and
`HANDOFF.md`.

<!-- T64-RETRO-AGENT-FINDINGS -->

## 9. What a reviewer agent is worth, stated honestly

This is the twelfth consecutive sprint with no human reviewer, and the first
with any second reader at all. Both halves of that deserve to be said.

**What it is worth.** An agent that was not the thing that wrote the code reads
without the author's memory of intending the code to work. It re-runs commands
rather than trusting them, and it has no stake in the sprint looking good.
Three of this project's most expensive defects — the invented
`pg_get_constraintdef` fixture, the `0012` migration that silently dropped
`0007`'s reservation, the `payable_type` CHECK — were all things a sceptical
reader with a terminal would plausibly have caught.

**What it is not worth.** It is one model reading another instance of itself,
given a framing written by the author, pointed at the files the author named.
Every structural blind spot the author has, it has. It cannot ask "why is this
sprint's premise the right premise?", and a question nobody thought to ask is
exactly the class of failure that cost this project 41 sprints on D1 and 57 on
Docker. **A second pass is not a second person**, and this retro does not
record one as if it were.

**The honest reading** is that this closes the cheapest half of the gap — the
half about commands not being re-run and figures not being re-derived — and
leaves the expensive half, which is a reader who disagrees about what the
sprint should have been.

## 10. What went well

- **Running the gate before planning instead of quoting it** is the whole of
  §2. It cost one command at the start of a ceremony and it caught a direct
  dependency shipping a known XSS.
- **The minimal fix was taken over the newest one**, again. Every advisory had
  a same-major patch fix, found by reading each one's `range`: `^3.5.42` and
  `^1.2.2`, not 5.x or 2.x. The declaration is minimal; npm's resolution
  (3.5.43) is latest-compatible, and the review said so rather than letting a
  reader discover the difference.
- **A planning check named its own unresolved question instead of asserting
  past it.** T64.4's dependency-completeness row said that whether `npm ci
  --dry-run` detects a lockfile disagreement offline was *to be verified by the
  ticket*. It did, and the answer changed the implementation from a parser to
  three lines of Makefile. T15.5's lesson — a planning check that asserts a
  capability it has not read — is what that row exists to avoid.
- **Five mutations, all quoted**, and one of them (T64.5's) deliberately
  reintroduced T62's exact defect in the new tool to prove the new test sees
  it.
- **`gate-coverage` stayed at 47 with two new tests**, because they went into
  an already-covered package. The tool whose principle is "no list" keeps
  demonstrating the principle.
- **The ceremony widened its own mandated bookkeeping and said so.** The "Open
  issues" section was six sprints stale — four open, dated T58, listing a
  closed issue. Fixing it was discretionary; recording that the mandated rule
  enumerates three artifacts and therefore protects three artifacts is the part
  that outlives this sprint.

## 11. Recommendations for T65 and beyond

1. **Run `SKIP_GOVULNCHECK=1 make security` as the first command of every
   Ceremony 1**, before escalations. §2 is the argument and T64.3 is the rule;
   what is missing is the habit being written into the ceremony's own ordered
   steps rather than into a clause about dates.
2. **`make lock-check` does not catch the failure that actually happened**
   (#322's sibling gap). T63.2's incident was a lockfile **deleted and then
   regenerated**; the gate catches *absent* and *out of sync*. A regenerated
   lockfile that still satisfies `package.json` but has silently dropped the
   `overrides`-driven pins would pass. Fixing that needs a policy about which
   resolved versions are load-bearing — i.e. a list — so **only take this
   ticket if someone can design it without one.**
3. **#320 is no longer blocked — the remedy was tested while this retro was
   being written, and it works.** npm **12.2.0** exists (this container has
   **10.9.7**), and #320's preferred fix was "a newer npm; verify on npm ≥ 11
   before any other approach". Verified in a scratch prefix, against a copy of
   `web/`, doing exactly what npm 10.9.7 crashes on:

   ```
   $ "$SCRATCH/npmtool/node_modules/.bin/npm" -v
   12.2.0
   $ npm install --package-lock-only     # vitest ^4.1.11 + @vitest/mocker override
   found 0 vulnerabilities
   exit=0
   ```

   No `edgesOut` crash. **The hypothesis on the issue was right: the crash is
   npm 10.9.7's resolver, not this dependency graph.**

   What is *not* verified is the thing that actually matters — `--package-lock-only`
   installs nothing and runs nothing, so **the 717 web tests on `vitest` 4.1.11
   remain unrun**, and that is #320's own acceptance criterion. So the ticket is
   now bounded rather than blocked, and T64 deliberately did not take it:
   changing the toolchain mid-sprint, under a security PR and under a brand-new
   `lock-check` gate's first appearance, is how a clean result becomes
   unattributable to any one change. Recorded on the issue with both halves,
   including a caution the new gate creates: `lock-check` runs
   `npm ci --dry-run --offline`, so a lockfile regenerated by a *different* npm
   than CI runs is a new way to make it disagree. **Decide which npm the project
   targets** as part of that ticket.
4. **Add the stale-read qualification to T63.1** (§7c), or decide against it
   explicitly. One sentence: a live-state command run in the same breath as the
   action can return the pre-action state; read the object, not the list, or
   read twice.
5. **The squash-ancestry mechanic deserves a documented procedure, not a
   rediscovery each time** (§6). Three sprints in a row have now hit it. The
   procedure is short — verify the base's file is byte-identical to the
   branch's pre-edit version, then rebuild the change on the base rather than
   resolving markers by hand — and it currently exists only in two retros and a
   PR body.

## 12. Sweep and bookkeeping

- **Issues: 5 → 4.** #322 closed **by hand** (`closed_at
  2026-10-06T07:45:26Z`), with its rationale and the three-row disposition
  table on the issue. `Closes #N` still cannot auto-fire against a non-default
  base branch, which is the mechanism behind T62's false claim, so the closure
  was performed at merge time rather than recorded as done.
- **Merge order #325 → #326**, verified against each PR's `merged_at` (see the
  header), squash-merged as `1febf8e` and `2e9b59f`.
- **One new gate**, `make lock-check`, reachable from `ci-checks`. **Two new
  rules**, both in `sprint-process.md`. **One new listing mode**,
  `cmd/docsindex -statuses`, sharing the gate's parser.
- **Verification.** `make ci-checks` green: `gate-coverage` 47/47,
  `docs-index-check` OK across 65 rows / **111** process docs / 17 ADRs — 110
  before this retro existed, the self-counting effect `t63-retro.md` §7
  recorded —
  `lock-check` OK, `fmt-check` and `golangci-lint` clean, web 61 files / 717
  tests unchanged either side of `vue` 3.5.40 → 3.5.43, `build-web` green.
  `make ci-integration` green on **four** runs; two quote `DONE 2502 tests`
  with zero `FAIL` lines in the full captured output, and **two are exit-0 only
  because the capture took the wrong line** — stated that way rather than
  rounded up to four quoted totals. `SKIP_GOVULNCHECK=1 make security` **PASS
  as of 2026-10-06T07:28:59Z**, carrying its date per this sprint's own new
  clause.
- **Five mutations**, each with its output quoted: T64.1's lockfile revert
  (both directions), T64.2's guard made to fire with a stub `docker`, T64.4's
  absent-lockfile and drifted-`package.json` cases, and T64.5's listing made to
  skip front-matter ADRs.
- **The two `vitest` moderates are untouched** and remain #320. Not perturbing
  that subtree is precisely what makes every other advisory in this sprint and
  the last one fixable.
- **This retro sets its own row's Retro cell to the retro's path and leaves
  Reviews alone**, per the path/number distinction `sprint-process.md` records.
  T65's Ceremony 1 owes the **Reviews** cell (#325 → #326, `merged_at`-verified)
  and the **narrative**, and should check whether T64.3's two clauses are being
  followed by a sprint that did not write them — which, as `t62-retro.md`
  recommendation 2 says, is the only evidence that counts.

## 13. Honest-form outcome sentence

For `HANDOFF.md`'s T64 row, to be carried verbatim rather than strengthened:

> T64 was the sprint that found **its predecessor's green security gate had
> gone red on a tree nobody had touched** — three new `high` advisories within
> twenty hours of T63.2 leaving the gate passing, `vue` among them and direct,
> with an XSS at CVSS 7.2 and a DoS at 7.5. Nothing in T63 was wrong: the code
> did not change and its verification was honest and commanded; **the advisory
> database moved.** So **a green gate is a statement about a moment, not about
> a tree** — a sharper claim than T63.1's rule, because there the actor's
> account was the unreliable part and here the account was accurate and the
> world moved under it, which means a re-read is owed **even when nobody doubts
> the claim**. T64.3 wrote that in as a date beside the claim, and T64.1 fixed
> all three advisories with **minimal same-major versions** read off each
> advisory's own `range`. Its ceremony also found that T62.4's **corrected**
> escalation sweep had still missed a product question — #145 has carried
> *"Needs product input on which"* since 2026-08-14, and the corrected sweep
> reads an issue's labels and state, one level short of where the question
> lives — which makes three sweeps in a row each fixing one level of derivation
> and stopping one short. Put to the Product Owner and answered as a deferral
> **with a trigger a future sprint can cause**, which is the specific thing
> DECISION D1 lacked for 41 sprints. T64.2's derived sweep then **found two
> live instances the issue that prompted it had not predicted** — the `README`
> and the `Jenkinsfile`, whose `agent any` rationale rested on this project's
> false belief about its own sandboxes, so the premise was corrected and the
> decision kept: **a decision can survive its own rationale**. Of 103 matching
> lines across 64 files, three were edited and a hundred deliberately left,
> because a sweep that edited every match would destroy the project's own
> record of having been wrong. **T63.3's rule, whose first application had
> failed, was repaired and re-applied inside one sprint**: the mutation for a
> dependency pin is reverting the lockfile, and T64.1's middle row proves it —
> the new declaration against the old lockfile fixes nothing at all. The sprint
> corrected three of its own claims: a scope figure of "66 lines in 63 files"
> that nobody had computed (it is 103/64); an exit code of **141** that was
> SIGPIPE from the measurement's own `head`, not a third failure mode of the
> gate; and a **live issue count read one second after the write, which
> returned the pre-action state** — the first qualification this project has
> found on T63.1's own rule. PR #325 could not merge at all until a
> squash-ancestry conflict on byte-identical content was resolved by a `git
> fetch` this session had earlier been denied, which is worth recording as **a
> denial of a mechanism not being a denial of a purpose**. Verified by
> `make ci-checks` green (`gate-coverage` 47/47, `docs-index-check` 65/110/17,
> `lock-check` OK), `make ci-integration` green on four runs with **2502 tests
> and zero failure lines in the two where the total was captured**, and
> `SKIP_GOVULNCHECK=1 make security` **PASS as of 2026-10-06T07:28:59Z**. Five
> mutations, each quoted. While the retro was being written, **#320's own
> preferred remedy was tested and works** — npm 12.2.0 resolves the `vitest`
> bump that npm 10.9.7 crashes on, in a scratch prefix, `found 0
> vulnerabilities`; the 717 web tests on that version remain unrun, which is
> #320's actual acceptance criterion, so the issue is now bounded rather than
> blocked and was deliberately left to its own ticket. **The first sprint in
> twelve with any second reader**
> — three report-only agent reviewers — and §9 says what that is and is not
> worth: one model reading another instance of itself, given the author's
> framing, which closes the half of the gap about commands not being re-run and
> leaves the half about a reader who disagrees with the premise.
