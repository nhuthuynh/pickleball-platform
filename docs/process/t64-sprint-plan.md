# T64 Sprint Plan — Ceremony 1 (backlog refinement)

Per `docs/process/sprint-process.md`. PM + PE, held against `HANDOFF.md`,
`docs/process/t63-retro.md`, `docs/process/t63-sprint-plan.md`, the live issue
list, and the tree at `ae5c287`.

**This plan is subject to T62.2's rule** (every quantity carries the runnable
command that produces it) **and to T63.1's** (a claim that an action was taken
carries the command that re-reads the live state). §2 is what happened when the
ceremony applied the second one to the previous sprint's green gate.

**Ceremony order, as mandated:** escalations (§1) → bookkeeping (§3) → issue
sweep (§4) → tickets (§5). §1 and §2 both changed what §5 contains.

---

## §1 — Escalations

### ADRs: none escalated, read with the gate's own parser

```
$ go run . /home/user/pickleball-platform/docs/adr   # scratch copy of tools/docsindex, calling adrStatus()
Accepted     ## heading     0001-dual-invariant-enforcement.md
… 17 lines …
---
Accepted: 17
```

Eleven `## Status` heading form, six `- **Status:**` front matter, which is the
split `sprint-process.md` documents. **Nothing escalated.**

**Why that command is strange, and why it is a ticket.** T62.4 made ADR status
*classification* a gate (`make docs-index-check` fails on an ADR it cannot
read), but the gate reports only pass/fail — it has no way to **print** the
statuses, and `adrStatus` is unexported. So a ceremony that wants to list
escalations has two options: re-implement the parser as a shell grep, which is
the mistake this project has now made five times, or do what this one did —
copy the package to a scratch directory, rename the package clause, and call
the real function from a throwaway `main`.

That is an absurd amount of ceremony for "show me the statuses", and the
absurdity is the argument: **the gate can refuse a bad status and cannot show a
good one.** T64.5 adds the listing.

### Issues: every open one considered, and one of them was awaiting a decision

```
$ (list_issues state=OPEN) -> totalCount
5          # #322, #320, #149, #145, #134
```

| Issue | Awaits a decision? |
|---|---|
| #322 | No. Actionable now; it is T64.2. |
| #320 | No. Blocked on a newer npm or a separately-reviewed major bump. |
| #149 | No. Ordinary unbuilt work (a read port into Booking). |
| #145 | **YES — and two corrected sweeps missed it.** See below. |
| #134 | No. Blocked on hardware this environment does not have. |

**#145's body contains an unanswered product question** — *"either (a) a
one-time backfill … or (b) an explicit 'claim this pre-existing account' UX …
**Needs product input on which**"* — and it carries no `role:product-owner`
label.

T62.4 rewrote this sweep precisely because a label-keyed sweep could not see
#314. The corrected rule says to **consider every open issue, not every
labelled issue**, and both T62's and T63's ceremonies then ran it and reported
that none of the open issues awaited a product decision. **Both were wrong, and
the corrected mechanism is why:** it asks whether an *issue* awaits a decision —
a property of its labels and state — rather than whether the issue's *text*
contains a question nobody has answered. #145 has carried that sentence since
2026-08-14.

So the fix is one more turn of the same screw: **the thing to read is the prose,
because that is where the question is written.** T64.5 carries it.

**Put to the Product Owner before any ticket was refined, per the rule, and
answered: defer until the provider is chosen, with the trigger named.** The
reasoning in the answer is the reasoning the option carried — the provider's own
capabilities (verified emails? native account linking?) determine which options
exist, so an answer now might not survive contact with the real system. Recorded
**on the issue**, not only here, per `sprint-process.md`'s "when an issue's
analysis is corrected, correct the issue" — and **in the body, above the T62
correction block**, not as a comment, because `t59-retro.md` §3's finding is
that GitHub renders the body first and a comment is the thing nobody re-reads.
The "Needs product input on which" sentence is struck in place and points at the
block. Verified after writing, since a body edit retypes nothing:

```
$ gh api …/issues/145 --jq .body | tr -d '\r' | diff - /tmp/i145-new.md
81a82,85    # only the attribution footer the proxy appends
```

**This is a deferral with a trigger, which is what D1 lacked.** The trigger: the
ticket that provisions a real identity provider must answer this question
**before it merges**. Unlike ADR-0015's trigger, this one is conditioned on an
event a future sprint can cause.

## §2 — The finding this ceremony exists to record

**`make security` is red today on a tree nobody has touched.**

T63.2 fixed five `high` npm advisories and left the gate passing. Its PR body,
its review and T63's retro all record that, correctly, with the command:

```
2026-10-05 $ SKIP_GOVULNCHECK=1 make security | tail -1
PASS: no new gating findings (0 baselined, 2 below threshold).
```

Under a day later — **≈17.7 h by the only timestamps that exist**, PR #321's
`merged_at` to PR #325's; "twenty hours" stood here until T64's review pointed
out that T63 never recorded the clock time of its own `make security` run,
which is exactly the gap T64.3's clause exists to close — same tree, same
command:

```
2026-10-06 $ SKIP_GOVULNCHECK=1 make security | tail -6
NEW gating findings (3):
  - [npm-audit] @vue/server-renderer (high): @vue/server-renderer: XSS via missing CR in attribute-name blacklist
  - [npm-audit] source-map-js (high): source-map-js allows event-loop denial of service through indexed source-map section offsets
  - [npm-audit] vue (high): via @vue/server-renderer
FAIL: 3 new gating finding(s).
```

```
$ cd web && npm audit --json   # tally + what it actually loaded
tally: {'moderate': 2, 'high': 3, 'total': 5}
deps audited: 368
```

| package | severity | direct | vulnerable range | advisory |
|---|---|---|---|---|
| `vue` | **high** | **yes** | `3.2.13 - 3.5.41` | via `@vue/server-renderer` |
| `@vue/server-renderer` | **high** | no | `<3.5.42` | XSS via missing CR in attribute-name blacklist, CVSS **7.2** |
| `source-map-js` | **high** | no | `1.0.0 - 1.2.1` | event-loop DoS via indexed source-map section offsets, CVSS **7.5** |

All three have `fixAvailable: true`. `vue` is a **direct** dependency and the
framework the client ships, so this is not a dev-only finding like #320's.

### Why this is a finding and not just a ticket

Nothing in T63 was wrong. The code did not change, the pins did not slip, the
gate was not misconfigured, and the verification was honest and commanded.
**The advisory database moved.** So:

> **A green security gate is a statement about a moment, not about a tree.**

That is a sharper version of T63.1's rule than T63.1 states. T63.1 says a claim
about an action carries the command that re-reads the live state — on the
reasoning that *the actor's account of what they did* is the unreliable part.
Here the actor's account was accurate and the **world** moved underneath it.
Both failures are repaired by the same discipline, but the second one means a
re-read is owed even when nobody doubts the claim, and the useful form is a
**date**: "the security gate passed" is not a fact about the repository unless
it says when.

T64.3 writes that in, and it is the cheapest rule this project has adopted: one
extra word next to a claim that already carries a command.

### What this says about the two sprints that recorded "govulncheck is owed"

T61's and T62's retros both deferred running the gate, and T63's ceremony found
five `high` advisories behind that deferral. **The same deferral would have
cost three more highs in a single day.** The number that matters is not how many
findings a run produces, it is how fast the answer goes stale: T63.2's green
lasted under a day, which means the gate is worth running at the start of every
ceremony rather than at the end of a sprint.

## §3 — Bookkeeping

Per `sprint-process.md`'s "Correct the previous sprint's Docs-index row", all
three items, plus one this ceremony widened on purpose.

1. **T63's Reviews cell filled**, verified against each PR's `merged_at` rather
   than assumed from numbering:

   ```
   $ (pull_request_read #319/#321/#323/#324).merged_at
   2026-10-05T11:22:24Z   #319  31000d8   Ceremony 1
   2026-10-05T13:29:52Z   #321  e7710af   all four tickets
   2026-10-05T14:37:22Z   #323  a0c497a   retro
   2026-10-05T14:39:36Z   #324  ae5c287   the retro's one rulebook line
   ```

   Its **Retro cell was already set by the retro PR itself**, per the
   path/number distinction T62's retro forced into the rule — so this ceremony
   had nothing to correct there, which is the rule working.

2. **A T64 row created.**

3. **T63's narrative added**, with the retro's own honest-form sentence — **53
   blockquote lines lifted programmatically** from `t63-retro.md` §10, not
   retyped:

   ```
   $ python3 -c "...index('## 10. Honest-form outcome sentence'); [l for l in ... if l.startswith('>')]"
   lifted 53 blockquote lines; 3918 chars
   ```

4. **Widened on purpose: the "Open issues, split by whether anyone can act"
   section was six sprints stale.** It said **four open**, dated itself **T58**,
   listed **#296** as open — closed on 2026-09-25 — and knew nothing of #320 or
   #322. The mandated bookkeeping list does not include this section, and that
   is exactly how it drifted: a rule that enumerates three artifacts protects
   three artifacts. Re-derived live and dated, with every row's premise
   re-verified against the tree (§4).

   **Recorded rather than quietly fixed, because the gap is structural.**
   `make docs-index-check` covers the Docs-index table and the narratives; it
   does not know this section exists. A second list that nobody is required to
   re-read is the same defect in a different file.

## §4 — Issue sweep: both questions, every issue

`sprint-process.md`'s premise check — (1) is the blocker still in place, (2) is
what the issue describes still true of the tree, verified against the code.

| Issue | Blocker | Premise |
|---|---|---|
| #322 | none | **Verified true.** `sed -n 376,377p Makefile` unchanged; `grep -c "Docker works here" CLAUDE.md` → 1. **Corrected at T64's review:** this row also claimed *"the derived grep still returns exactly 2 lines outside docs and tests"* — the grep was named and never written down, and the sprint's own published 7-term version returns **9** such lines (Makefile 3, Jenkinsfile 4, README 1, `tools/docsindex` 1). The claim was unreproducible as stated. |
| #320 | npm 10.9.7 confirmed again (`npm -v`) | **Surrounding tally drifted; the issue's own claims did not.** Its figures are labelled *"Verification state when filed"* — the dated form `sprint-process.md` asks of a filing session — so they became **historical, not false**, and nothing needed striking. Corrected by [comment](https://github.com/nhuthuynh/pickleball-platform/issues/320#issuecomment-6011228605) rather than a body edit, with that reasoning stated. The three new highs are not its subject (T64.1 owns them); its own two moderates are unchanged in severity and still below threshold. |
| #149 | `internal/payments/port/` holds 11 ports, none reading Booking (`ls`) | **Verified true.** `grep -rn "BookingHostID: req.GetBookingHostId()"` → `handler.go:160` and `:284`, exactly the two the T62 correction names. |
| #145 | real IdP, unobtainable here | **Verified true.** All five `adapter/identity` seams exist (`ls -d internal/*/adapter/identity \| wc -l` → 5); `db/migrations/0019_identity_subject.sql` present. Its product question is now answered as a deferral with a trigger (§1). |
| #134 | assistive-technology hardware | **Verified true.** All three routes still in `ROUTES_UNDER_TEST` (`accessibility.spec.ts:98-99`, `:197-199`), so the automated half still covers them and the manual half is still owed. |

**One of five drifted**, and it drifted in under a day — the same mechanism as
§2. Worth separating the two shapes, because the remedy differs: #320's *own*
assertions are intact and only the world around them moved, which a dated claim
survives; #149's and #145's were assertions about the tree that had quietly
become false, which is what a body correction is for. The premise check has now fired in three of the **four** ceremonies that have
run it (T59 #149, T62 #149 + #145, T64 #320) — **corrected at T64's review**:
this read *"three of the five"*, and only T59, T62, T63 and T64 have plan
documents, since `HANDOFF.md` records that T60 and T61 held no ceremony. The
denominator was invented, the real rate is **higher** than the one claimed, and
the figure carried no command. Either way it is a strong enough rate to stop
treating a clean sweep as the expected outcome.

## §5 — Tickets

Five. One is a live exposure in the shipped client and did not exist yesterday.

### T64.1 — Fix the three new `high` advisories, `vue` among them

**Story.** As an operator of this platform, I want the dependency security gate
to pass again, so that a known-XSS version of the web framework is not what we
ship.

**Description.** §2. Three new `high` findings against unchanged code, all with
`fixAvailable: true`, one of them `vue` itself at `3.2.13 - 3.5.41`. T63.2's
shape applies directly: read each advisory's `range`, take the **first version
above the vulnerable range in the same major line**, and do not reach for
`latest`.

**Instructions.**
1. Pin `vue` to `^3.5.42` (its direct `devDependencies`/`dependencies` entry,
   not an override — it is a direct dependency) and `@vue/server-renderer` and
   `source-map-js` to their first fixed versions, via `overrides` where they are
   transitive. Check each against the advisory's `range` rather than assuming
   the next patch is fixed.
2. **`npm install` with the lockfile present** — per `CLAUDE.md`'s npm gotcha,
   `npm audit fix`, `npm update` and a fresh resolve all crash on this graph.
   **Do not delete `web/package-lock.json`.**
3. **Mutation-verify per T63.3 and `CLAUDE.md`'s new clause: revert the
   lockfile, not the declaration**, and show the three findings returning by
   name. Removing an `overrides` entry proves nothing against an
   already-resolved tree.
4. Re-run `make test-web` and `make build-web` and report both. `vue` is the
   framework; a patch bump inside 3.5.x should be invisible to the 717 tests,
   and "should be" is why it gets run.
5. Re-run `SKIP_GOVULNCHECK=1 make security` and report the tally.

**NFRs.** 61 web test files / 717 tests must stay green — a security fix that
breaks the client is not a fix. No baseline entry without a written reason.

**Standing instruction, carried from T63.2 because it applies again.** If a bump
breaks a web test, **stop and report; do not fix the test in the same PR.** A
test adjusted under pressure inside a security PR is how a real assertion gets
weakened, and this project has one recorded instance of a suite asserting a
defect as correct (`t58-retro.md` §3).
**Points: 3.** `role:principal-engineer`, `type:bug`.

### T64.2 — #322: the gate message that licenses inaction, and a derived sweep scope

**Story.** As a session blocked by a failing gate, I want the gate's own message
to carry the remedy, so that the project stops recording an unrunnable suite
that runs in two seconds.

**Description.** #322, premise re-verified in §4. `Makefile:376-377` tells the
reader the Docker gap is *"the documented gap in CLAUDE.md's gotchas, not a new
problem"*; `CLAUDE.md` has said the opposite since T61. T63's retro met it live
and refuted it in two seconds.

**Instructions.**
1. Rewrite the `ci-integration` guard's message to carry the remedy: the command
   that starts the daemon, roughly how long it takes, and what the suite then
   costs. **Do not state that the gap is documented or expected** — that framing
   is the defect, independent of wording.
2. Soften `make ci`'s closing line (`Makefile:370`), which implies this machine
   has no daemon.
3. **Then do the general version, which is the actual ticket**
   (`t63-retro.md` §8 recommendation 1): when a claim is retired, the sweep's
   scope is **every file that can utter it**, derived by a repo-wide grep rather
   than hand-listed. Re-run that derivation for the "no Docker daemon" claim
   across the whole tree — docs, tests, tooling, CI config, error strings — and
   report the count, including whatever the first two steps did not cover.
4. Write the principle into `sprint-process.md` in one short subsection: a
   documentation sweep reaches documents, and the copies that matter most are
   the ones a reader meets while blocked.

**NFRs.** No behaviour change to any gate — this is messages and a derivation.
`make ci-integration` must still fail when there is genuinely no daemon.
**Points: 2.** `role:principal-engineer`, `type:bug`.

### T64.3 — Two clauses T63 earned and deliberately left for a second reader

**Story.** As a reviewer, I want the two rules T63's retro recommended to be in
the rulebook, so that the next sprint inherits them rather than rediscovering
them.

**Description.** Both were left to this ceremony on purpose, with the reason
stated: a rule amended to fit the first document judged against it should have a
second reader.

**Instructions.**
1. **The dependency-pin clause** for "a guard is verified by removing it"
   (`t63-retro.md` §8 rec 3): for a version pin, the mutation is reverting the
   **lockfile**, not the declaration, because removing the declaration is a
   no-op against an already-resolved tree. The measurement is in `CLAUDE.md`'s
   npm gotcha as of `ae5c287`; this adds the *requirement*. Cite that the rule's
   first application (T63.2) **failed** — the review accepted an argument that
   no mutation existed — so the clause is repairing a known miss, not a
   hypothetical one.
2. **The date clause** (§2): a claim that a gate passed carries **when** it was
   run, because a security gate's result is a statement about a moment. One
   worked example suffices — T63.2's PASS and T64's FAIL, same tree, 20 hours
   apart.
3. **State the limit**, as T63.1 does for itself: neither clause can make a
   session re-run anything. They make the omission visible on the page.
4. **Do not build a gate for either.** Record the decision so T65 does not
   re-litigate it: a checker for "every gate claim has a date" has the same
   prose-detection problem T63.1 instruction 5 rejected.

**NFRs.** Prose only. Both clauses must cite the instance that produced them.
**Points: 2.** `role:principal-engineer`, `type:chore`.

### T64.4 — A mechanical guard for `web/package-lock.json`

**Story.** As a maintainer, I want the lockfile's disappearance to fail a gate,
because it is the only thing that makes installs work on this dependency graph.

**Description.** `t63-retro.md` §8 recommendation 2. The lockfile is
load-bearing — §3 of that retro measured that it, not `package.json`, carries
the advisory fixes — a fresh resolve cannot regenerate it here, and T63.2's
recovery from deleting it depended on a backup copy that session happened to
take. `Makefile:153` runs `npm ci` when `node_modules` is absent, which would
notice a lockfile that **disagrees** with `package.json`; nothing notices one
that is **missing or regenerated**.

**Instructions.**
1. Add a check, reachable from `ci-checks`, that fails when
   `web/package-lock.json` is absent, and that fails when `package.json` and the
   lockfile disagree about a pinned version. `npm ci --dry-run` may already do
   the second half — **verify that rather than assuming it**, and if it does,
   wire it rather than writing a new parser.
2. **Mutation-verify it both ways** (T63.3): delete the lockfile, show the
   failure; change one pin in `package.json` without the lockfile, show the
   failure. Quote both.
3. **Do not add a package list or a version list to the check.** The failure
   mode this project keeps shipping is a hand-written list that was stale on the
   day it was written — see `make gate-coverage`'s doc comment.

**NFRs.** Must not slow `ci-checks` materially and must not require network
access — `npm ci` would; a presence-and-consistency check should not.
**Points: 3.** `role:principal-engineer`, `type:chore`.

### T64.5 — Make the escalation sweep readable, and make it read prose

**Story.** As a ceremony, I want to list ADR statuses and surface unanswered
questions inside issue bodies, so that an escalation cannot hide in a place the
sweep structurally cannot look.

**Description.** §1 found both halves. The gate can **refuse** an unclassifiable
ADR status and cannot **print** a classifiable one, so this ceremony had to copy
`tools/docsindex` into a scratch directory to avoid re-implementing the parser.
And #145's product question sat unanswered through **two** runs of the
*corrected* sweep, because the corrected sweep asks whether an issue awaits a
decision — a property of labels and state — not whether its text contains a
question.

**Instructions.**
1. Add a listing mode to `cmd/docsindex` (e.g. `-statuses`) that prints each
   ADR's file, status token and which form carried it, using the **same**
   `adrStatus` the gate uses. Export it if needed. One flag, no new tool.
2. Add a test that the listing and the gate agree on every ADR in the tree —
   the point is that there is exactly one parser.
3. For the prose half, **add a ceremony step, not a gate**: Ceremony 1's issue
   sweep reads each open issue's body for an unanswered question and states,
   per issue, whether one is present. Record *why not a gate*: phrases like
   "needs product input", "needs a decision" and "open question" are prose, a
   grep over prose mis-fires, and T63.1 instruction 5 already rejected a
   prose-detecting gate for the same reason.
4. **Say what step 3 does not fix.** A question phrased without any of the
   obvious markers is still invisible, and the only real remedy is that the
   person reading the issue is looking for one. The step makes the reading
   obligatory and visible; it does not make it mechanical.

**NFRs.** The listing must not become a second source of truth for statuses —
one parser, called twice. `make gate-coverage` must still pass with the new
test (it will pick the package up with no edit; that is the design).
**Points: 3.** `role:principal-engineer`, `type:chore`.

### PM value ordering, if the sprint has to be cut

1. **T64.1** — a `high`-severity XSS advisory against the shipped web
   framework, and a red gate on the shared branch. Nothing else here has a cost
   that accrues while it waits.
2. **T64.2** — the message that tells a blocked session to stay blocked. Cheap,
   and the disclaimer it embodies went unchallenged from T4 to T60 — **a span,
   not a count** (corrected at T64's review; `HANDOFF.md` records the gap as
   disclaimed by *six* retros, and `t61-retro.md` §4 corrected this project for
   the identical mislabelling once already).
3. **T64.3** — two clauses already paid for by T63; delay costs only that T65
   rediscovers them.
4. **T64.4** — guards a known-load-bearing file whose loss this project has
   already survived once by luck.
5. **T64.5** — the most interesting and the least urgent: it repairs a sweep
   that has already produced its finding for this sprint.

**PE sign-off, with one sequencing constraint.** T64.3 and T64.2 both touch
`sprint-process.md` and should land in that order (T64.2's subsection is
narrower and sits beside T64.3's clauses). T64.1 is independent and should go
first regardless of ordering, because it is the only ticket whose subject is
live. T64.4 and T64.5 are independent of everything.

**Dependency-completeness check** (`sprint-process.md`, both questions):

| Ticket | Producer exists? | Consumer can reach it? |
|---|---|---|
| T64.1 | `web/package.json` + lockfile present; `tools/vulngate` already gates npm findings | yes — `SKIP_GOVULNCHECK=1 make security` is the existing path, and `build/npm-audit.json` is the file it reads |
| T64.2 | `Makefile:370`, `:376-377` are the only sites (derived grep, §4) | yes — no code reads those strings |
| T64.3 | `sprint-process.md` has both host sections ("A guard is verified by removing it", "Every quantity carries the command") | yes — prose |
| T64.4 | `Makefile:153` already runs `npm ci` conditionally | **to be verified by the ticket, not assumed**: whether `npm ci --dry-run` reports a `package.json`/lockfile disagreement without network access is instruction 1's own first step |
| T64.5 | `tools/docsindex.adrStatus` exists and is unexported; `cmd/docsindex` has a `flag` set already | yes — the flag parser is already there, and the listing needs the same `docs/adr` read the gate performs |

The one honest gap is T64.4's, and it is named as a verification step inside the
ticket rather than resolved here — per T15.5's lesson, a planning check that
asserts a capability it has not read is how a ticket gets cleared for dispatch
and then hits a wall.

## §6 — Deliberately out of scope, with the decision recorded

- **#320's two `vitest` moderates.** Below the gate's own threshold, dev-only,
  and blocked on npm 10.9.7's resolver (re-confirmed in §4). T64.1 deliberately
  does **not** touch the vitest subtree — that is precisely why T63.2's five
  fixes were possible.
- **#149's read port into Booking.** Ordinary unbuilt work and a genuine design
  change; it wants its own sprint, not a corner of this one.
- **#134's WCAG pass.** Premise re-verified, still needs an operator and a real
  screen reader.
- **#145's linking mechanism.** Answered as a deferral with a trigger (§1); the
  trigger fires on the IdP ticket, not here.
- **Running `govulncheck`.** Still `Forbidden` from this environment, now a
  `CLAUDE.md` gotcha rather than a per-sprint rediscovery. **The Go half of the
  security gate has never run on this project** — stated plainly because §2's
  finding makes it worse, not better: if the npm database moved three highs in a
  day, the Go database has been moving unobserved for the project's whole life.

## §7 — What this ceremony produced

1. **A red security gate on an untouched tree** (§2), with three new `high`
   advisories including the shipped web framework — found because this ceremony
   ran the gate before planning instead of quoting the previous sprint's result.
2. **The rule that follows from it**: a green gate is a statement about a
   moment, so a claim that a gate passed carries its date. T64.3.
3. **An escalation that two corrected sweeps missed** (§1), put to the Product
   Owner and answered as a deferral **with a trigger a future sprint can
   cause** — which is the specific thing DECISION D1 lacked for 41 sprints.
4. **The premise check fired on one of five** (#320), inside a day of that
   issue being filed. Third firing in five ceremonies.
5. **Six sprints of drift in a second index** (§3 item 4) — a section listing a
   closed issue as open and dating itself T58, caught only because this ceremony
   re-derived it. The mandated bookkeeping list protects three artifacts; this
   was the fourth.
6. **Five tickets**, 13 points, and five documented exclusions.
7. **An absurdity worth keeping**: to list seventeen ADR statuses honestly, this
   ceremony copied a package to a scratch directory and called an unexported
   function, because the alternative was a grep and this project has lost that
   bet five times. T64.5 makes the honest path the easy one.
