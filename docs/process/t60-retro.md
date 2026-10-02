# T60 Sprint Retro

Ceremony 3 per `docs/process/sprint-process.md`, six-role team (briefs:
`docs/agent-operating-handbook.md` Part B), held against `HANDOFF.md`,
`docs/process/t58-retro.md` and `docs/process/t59-retro.md` as the immediate
precedents, PRs #307 and #309, issues #305 and #308, and the live issue/PR/commit
history.

> ## Written late, and with less to hold it against than its siblings
>
> **Written at T61, one sprint after T60 ran.** T60 held **no planning
> ceremony**, so unlike T59 there is no plan to hold this retro against, and —
> unlike T55 and T59 — **T60 has no `HANDOFF.md` Docs-index row at all** (§5).
> The record available is therefore two PRs, two issues and two commit messages.
>
> Those commit messages are unusually substantive, which is the only reason this
> retro has measured numbers rather than impressions. Every figure below was
> re-derived from them or from the tree at writing time, per `t61-retro.md`
> recommendation 4.
>
> The delay is short enough that little has decayed. One claim has: both T60 PRs
> record the integration suite as unrunnable here (§6).

**T60 held no planning ceremony.** Both of its tickets came from issues already
filed — #305 from T59's deliberate scope exclusion, #308 from T60's own work.

**Outcome: 2 tickets, 2 issues closed, 1 of them opened within the sprint.**
Merged as PR #307 (`940c444`, closing #305) → PR #309 (`52fcd66`, closing #308),
verified against `merged_at`. Issue count unchanged at 3 net; #308 opened and
closed inside the run.

---

## 1. What actually happened, in order

1. The session picked up **#305** — the Social Play identity-fixture pass, which
   T59.1 had deliberately excluded from its own scope and documented as a chosen
   gap.
2. Social Play's shared `fakeIdentityLookup` returned the verified subject
   unchanged, so `ctxAs("host-1")` produced `Game.HostID == "host-1"` — a shape
   `games.host_id` cannot hold, since migration 0026 made it
   `uuid NOT NULL REFERENCES identity_users (id)`.
3. T60 added a deterministic `sha256 → uuid` `resolvedUserID`, salted per
   package, made the shared fake return it, and retired T59.1's local
   `resolvingIdentityLookup` stopgap.
4. **It measured the fix by mutation rather than asserting it** (§2), and the
   measurement is this sprint's finding.
5. Writing that fix surfaced an **unstated property the security assertions rest
   on** — two fixture subjects must never resolve to the same `User.ID` — so a
   test was added for it, and **#308** was filed to mirror it into Payments and
   Competitions.
6. T60.2 then closed #308 **deliberately not in the shape #308 asked for** (§3),
   because the shape the issue asked for was itself the failure mode.

## 2. The finding this retro exists to record

**Social Play's own fixtures made it structurally unable to observe the thing it
spends most of its assertions on, and only mutation could reveal that.**

The measurement, from #307's commit message:

| | |
|---|---|
| mutation applied | make `actor()` skip resolution entirely — the exact defect the passthrough fake was blind to |
| tests catching it **before** the fix | **6** |
| tests catching it **after** | **36** |
| Social Play's own 30-plus authorization assertions, before | **zero** |
| what the 6 were | exactly the booking-backed tests — catching it only because *Booking* has a `uuidShape` guard, added by T59.1 |

Read that last row again. A package whose test suite is largely *about*
authorization — who may cancel, who may record, who may administer — could have
its entire actor-resolution step deleted and **not one of its own authorization
assertions would notice.** The six that noticed were borrowing a guard from
another bounded context.

### Why this is the sharpest instance of a pattern this project keeps hitting

`t61-retro.md` §3 records that an in-memory fake is more permissive than
Postgres, and that the asymmetry hid a CHECK-constraint defect for 51 sprints.
T60 is the same lesson **one sprint earlier and in a purer form**, because here
the fake was not merely more permissive — it was *wrong in the dimension the
tests existed to check*. A permissive fake lets a bad value through; a
passthrough identity fake makes "was this value resolved?" unobservable, which
is the question every one of those 30-plus assertions is implicitly asking.

> **A fixture that is wrong in the same dimension a suite tests cannot be caught
> by that suite. It can only be caught by changing the code and watching what
> fails to fail.**

T56's retro recommendation 4 said *"prefer deleting a line and re-running the
gates over reading code to decide whether a test exists"*, deliberately scoped
narrow — to a single new wire field — on the grounds that a general
mutation-testing rule would be ignored. **T60 applied it unprompted at a much
larger scope and got a result no amount of reading would have produced**, which
is evidence the narrow scoping was too modest. T61 then applied it again, three
times. The practice has now justified itself in three consecutive sprints
without ever being made a rule.

### The second-order test, which is the better half of the ticket

The fix created a new unstated dependency: the security assertions are only
meaningful if two fixture subjects never resolve to the same `User.ID`. If they
collide, every principal-not-wire assertion still passes while proving nothing.
T60 pinned it (`TestResolvedUserIDIsInjectiveAcrossFixtureSubjects`), verified by
mutation against both a passthrough and a constant resolver.

**A test that protects the tests.** Worth naming as a category, because the
defect it guards against is invisible by construction: a colliding resolver
produces a green suite that asserts nothing, and no assertion in the suite can
detect that, for the same reason §2 gives.

## 3. The issue this project filed, and why the ticket refused its shape

#308 asked for Social Play's injectivity test to be copied into Payments and
Competitions, **with each package's subject list**.

T60.2 did not do that, and was right not to. Two reasons, both verifiable:

1. **The hand-maintained list is the known failure mode, by this project's own
   rulebook.** `CLAUDE.md` says of `make gate-coverage`: *"there is no package
   list in the tool, and adding one is the one change that would defeat it —
   three sprints running (T11, T12, T13/#157) shipped a hand-written glob that
   was stale before its sprint ended."* #308 asked for exactly that artifact, one
   layer over. T60 itself had shipped a hand-list of 14 subjects.
2. **#308's own lists were incomplete the day it was filed.** Re-deriving each
   package's subjects fresh gave **22 for Payments where the issue listed 12**,
   and **21 for Competitions where it listed 8**.

So T60.2 made each test parse its own package's `*_test.go` at run time and
assert the property over every distinct string literal found — **333 for
Payments, 415 for Competitions, 476 for Social Play.** No list, nothing to
forget. Over-inclusive on purpose: a degenerate resolver is caught by any two
distinct inputs, so a wider net dominates and a narrower one only buys the
ability to go stale. Social Play's own hand-list version was retrofitted to the
derived one, rather than leaving two approaches across three packages with the
weaker one in the package where the defect actually happened.

Verified by mutation **9 for 9** — three degeneracies × three packages — plus two
mutations of the derivation itself, to prove the guard cannot pass vacuously. And
a detail worth carrying forward: *"Each mutation written so imports stayed used —
a mutation that fails to compile proves nothing."*

### This is the fourth instance of one failure mode

| | asserted | falsified by |
|---|---|---|
| #126 (T56) | "no price/fee field at all" | a migration 9 days earlier |
| #299 (T58) | "a part-payment may legitimately be recorded" | `payments_payable_unique_idx` |
| **#308 (T60)** | **"these are the subjects: 12 / 8"** | **re-deriving them: 22 / 21** |
| the Docker claim (T61) | "no Docker daemon available" | one command |

T58's retro called #126 and #299 *"one failure mode with two faces"*. #308 is the
third face and T61's is the fourth, and #308 is the one that makes the shape
unmistakable: **it was filed by the same session that then implemented it, days
apart, and was still wrong** — so this is not knowledge decaying between authors
or across sprints. It is a list written by inference where a command was
available, which is the identical diagnosis `t61-retro.md` §7 reached about its
own numbers.

## 4. The scope estimate, recorded against itself

#305's scope estimate was **wrong in the pessimistic direction**, and T60's own
commit says so:

> it inferred "~24 assertions across ~12 files" from a correct 6->30 failure
> count, but 24 of those traced to two seed helpers. Real edit: 6 files, ~8
> assertions. **A failure count is not an edit count.**

That last sentence is the useful artifact. The failure count was *correct*; the
inference from it was not, because failures cluster behind shared fixtures. Worth
keeping because the error is in the safe direction and therefore the kind nobody
corrects — an over-estimate costs nothing visible, so it is never audited, and a
project that habitually over-estimates scope from failure counts will decline
work it could do. T61 hit the same clustering from the other side: its 34
integration failures reduced to seven root causes.

## 5. T60 left no record in `HANDOFF.md` at all

Not a stale row, not a "not yet written" cell — **no row.** Verified:
`grep -c "^| T60 " HANDOFF.md` returns 0, as does the same grep for T61.

Two tickets, two issues closed, a new test category introduced, and three
packages' fixtures restructured, with nothing in the project's index of record.
The mechanism is structural rather than careless:

- `sprint-process.md` assigns the previous sprint's row and narrative to the
  **next** sprint's Ceremony 1 (correctly — a sprint cannot cite its own merge
  numbers).
- **T61 held no planning ceremony**, so T60's row had no owner.
- T61 then also produced no row for itself, for the same reason one sprint on.

So the rule that correctly prevents a sprint from writing its own row has no
fallback for *"the next sprint did not hold the ceremony that would write it"*,
and two sprints' worth of bookkeeping fell through the resulting gap. This is the
same class of defect as `make gate-coverage` exists to catch in the test gates:
**a responsibility assigned to a step that does not always run.**

## 6. What T60 believed that has since been falsified

- **Both PRs record "Not run: ci-integration — no Docker daemon."** False; one
  sprint later T61 started the daemon in about four seconds and found three
  production defects. T60's changes were test-only, so nothing it shipped was at
  risk — but T60 is the last sprint to repeat the claim, and it repeated it
  twice.

## 7. Recommendations

1. **Make the mutation check a rule, at the scope T60 and T61 actually used
   it.** T56 scoped it narrowly to a single new wire field on the explicit
   grounds that a broad rule would be ignored. Three consecutive sprints have
   since applied it voluntarily at much larger scope and found something each
   time (§2). The evidence now contradicts the caution. Suggested form: **a
   change that adds or relies on a guard is verified by removing the guard and
   recording what fails** — which is what all three sprints did anyway.
2. **A test fixture that participates in the property under test needs its own
   test.** §2's injectivity guard is the worked example: the suite cannot detect
   a colliding resolver, because a collision makes every assertion vacuously
   pass. Candidates to audit: any fake that *derives* a value the assertions then
   compare.
3. **An issue that enumerates anything must derive the enumeration, or say it
   was not derived.** #308 listed 12 and 8 where the real figures were 22 and 21
   (§3). This is `t61-retro.md` recommendation 4 generalised from durations to
   lists, and it is the same question [#314](https://github.com/nhuthuynh/pickleball-platform/issues/314)
   asks — T60's #308 is a fourth data point for it.
4. **`sprint-process.md` needs a fallback for the sprint whose successor holds
   no Ceremony 1** (§5). T60 and T61 both have no Docs-index row because the
   step that writes one is assigned to a ceremony that did not happen. Minimum
   viable fix: a sprint that holds no planning ceremony still writes the
   *previous* sprint's row as its first act, since that is bookkeeping and not
   planning.

## 8. Sweep and bookkeeping

- **Issues: 3 net, unchanged.** #305 closed by PR #307. #308 opened **and**
  closed within the sprint, by PR #309 — the same opened-and-closed-in-run
  pattern T56 had with #297 and T58 with #299.
- **Merge order #307 → #309**, verified against `merged_at`
  (`2026-09-30T14:20:40Z`, then `2026-10-01T13:29:19Z`). Clean sequence, one day
  apart, no stacking — the shape `t55-retro.md` §2 wishes T55 had had.
- **A numbering break: there is no T60.1.** The tickets are labelled **T60** and
  **T60.2**, confirmed from the commit subjects. Harmless in itself, and worth
  recording because this project's conventions are load-bearing elsewhere: a
  reader looking for T60.1 will not find it, and nothing in the record explains
  the gap.
- **`Closes #N` still cannot fire** against a non-default base branch. Both
  closures manual.
- **Self-review.** Both PRs carry review comments, not approvals. Fifth
  consecutive sprint by T60's count.
- **No `HANDOFF.md` row** (§5) and **no retro at the time**, which is why this
  document exists at T61.
- **This retro does not create `HANDOFF.md`'s T60 Docs-index row.** Per
  `sprint-process.md` a retro PR cannot write the row that points at it. T62's
  Ceremony 1 owns it — and per §7 recommendation 4, owns deciding how that
  assignment survives a sprint that holds no ceremony.

## 9. Honest-form outcome sentence

For `HANDOFF.md`'s T60 row, to be carried verbatim rather than strengthened:

> T60 held no planning ceremony; both tickets came from already-filed issues.
> T60 (PR #307, `940c444`, closing #305) gave Social Play the identity-fixture
> pass Payments had at T28.1 and Competitions at T29.1 — its shared
> `fakeIdentityLookup` returned the verified subject unchanged, so
> `ctxAs("host-1")` produced a `Game.HostID` that `games.host_id` cannot hold,
> since migration 0026 made it a uuid FK. **The finding this retro exists for is
> what the fix measured**: deleting actor resolution entirely went from **6 tests
> catching it to 36**, and the 6 were exactly the booking-backed ones, catching
> it only because *Booking* had a `uuidShape` guard from T59.1 — **Social Play's
> own 30-plus authorization assertions caught it zero times.** A package whose
> suite is largely about authorization could have its actor resolution deleted
> and not one of its own assertions would notice, because a fixture wrong in the
> same dimension a suite tests cannot be caught by that suite; only mutation
> finds it. T60 also added a **test that protects the tests** —
> fixture subjects must never resolve to the same `User.ID`, or every
> principal-not-wire assertion passes while proving nothing — and filed #308 to
> mirror it. **T60.2 (PR #309, `52fcd66`) then closed #308 deliberately not in
> the shape #308 asked for**: the issue asked for a hand-maintained subject list,
> which is the exact artifact `CLAUDE.md` names as having defeated
> `gate-coverage` three sprints running, and **#308's own lists were incomplete
> the day it was filed — 12 subjects where 22 existed for Payments, 8 where 21
> existed for Competitions.** Each test now derives its inputs by parsing its own
> package at run time (333 / 415 / 476 literals), verified by mutation 9 for 9.
> That makes #308 the **third** face of the failure T58 called "one failure mode
> with two faces", and the clearest, because it was filed and implemented by the
> same session days apart — a list written by inference where a command was
> available. T60 also recorded its own scope estimate as wrong in the
> pessimistic direction (*"a failure count is not an edit count"* — ~24
> assertions inferred, 8 actual, because 24 clustered behind two seed helpers).
> **Written at T61, one sprint late, and T60 has no `HANDOFF.md` Docs-index row
> at all** — not a stale one, none — because `sprint-process.md` assigns a
> sprint's row to the next sprint's Ceremony 1 and T61 held no planning ceremony,
> a rule with no fallback for the successor that does not run. Both PRs record
> the integration suite as unrunnable for want of a Docker daemon, which T61
> falsified one sprint later.
