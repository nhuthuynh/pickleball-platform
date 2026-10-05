# T62 Sprint Retro

Ceremony 3 per `docs/process/sprint-process.md`, six-role team (briefs:
`docs/agent-operating-handbook.md` Part B), held against
`docs/process/t62-sprint-plan.md`, `docs/process/t61-retro.md` as the immediate
precedent, PRs #316 and #317, issues #311/#314/#149/#145, and the tree at
`d7dc605`.

> ## This retro is the first document subject to T62.2's rule
>
> T62.2 adopted, this sprint, on the Product Owner's answer to #314: **any
> quantity asserted in a process document carries the runnable command that
> produces it — not the technique, the command.** T62's own retro is the first
> document the rule applies to, so it complies, and §7 reports what complying
> actually cost.
>
> The short version: the first two commands I wrote for this retro returned
> **wrong numbers**. The rule caught them.

**Outcome: 5 tickets, 2 issues closed, 1 production defect found and fixed,
2 new gates.** Merged as PR #316 (`ed14f0e`, Ceremony 1) → PR #317 (`d7dc605`,
all five tickets). #311 and #314 closed. Issue count 5 → 3.

---

## 1. What actually happened, in order

1. **Ceremony 1** (PR #316) ran escalations first, per the rule, and found its
   own escalation sweep mechanically unsound in three ways — one of them live:
   **#314 was a decision awaiting an answer that the sweep could not see.**
2. #314 went to the Product Owner before any ticket was refined. Answer:
   **widen the quantity rule to all quantities.**
3. A derived check of the Docs index found **eight sprints** of accumulated
   row debt, against `t60-retro.md`'s predicted two.
4. The issue sweep's premise check fired on **two of five** issues.
5. The PE pass on #316 **declined its own plan's ticket sequencing**; the plan
   was amended rather than merged over.
6. **T62 execution** (PR #317) delivered all five tickets.
7. **T62.5 found a live production defect on its first real run** (§2).
8. Three of the sprint's own written claims were overturned by checking them
   (§4, §5, §6).

## 2. The finding this retro exists to record

**Every refund of a Social Play Registration has failed against a real database
since T6.5, and the sprint that was built to look for exactly this class of
defect found it in its first run.**

```
$ grep -n "PaymentStatusRefunded" internal/socialplay/domain/registration.go | head -2
33:// PaymentStatusRefunded (T6.5) mirrors internal/payments/domain.Status's
48:	PaymentStatusRefunded PaymentStatus = "refunded"

$ grep -n "reconcileRegistrationPaymentStatus(ctx, updated" internal/payments/app/service.go
506:	… socialplaydomain.PaymentStatusPaid) …
959:	… socialplaydomain.PaymentStatusRefunded) …
```

`registrations.payment_status`'s CHECK, written at T5 in `0005_socialplay.sql`,
accepted `('unpaid', 'paid')`. The domain gained `refunded` one sprint later and
`RefundPayment` writes it at `service.go:959`. Reproduced on the real path, not
inferred:

```
ERROR: new row for relation "registrations" violates check constraint
"registrations_payment_status_check" (SQLSTATE 23514)
```

### The part that makes it a finding rather than a bug report

**The Competitions twin, twelve lines further down the same function, works.**
`0014_competitions.sql` wrote `competition_entries.payment_status ... IN
('unpaid', 'paid', 'refunded')`. One context got it right and its sibling did
not, in adjacent lines of one file.

So reading either file alone tells you nothing: `0005` looks internally
consistent, `0014` looks internally consistent, and `service.go:959` looks
correct against the domain. **Only comparing the two halves of CLAUDE.md rule 4
against each other — per pair, mechanically — surfaces it.** That is the whole
argument for T62.5's existence, and the defect is the argument's proof.

### Why nothing found it for the whole life of the feature

Two reasons, both already named in `CLAUDE.md` and both newly instantiated:

1. **The in-memory fake is more permissive than Postgres.** The Social Play
   in-memory repository has no CHECK constraint to violate, so every unit test
   passed. Third instance of that gotcha after `0030` and `0031`.
2. **There was no coverage of the path at all.**
   ```
   $ grep -rn "RefundPayment" internal/payments/adapter/socialplay/
   (no output)
   ```
   Zero. The cross-context integration test recorded a payment and asserted
   `paid`; nothing refunded anything.

## 3. #311 asked for five pairs. There are twenty-two.

The issue asked for conformance tests on five enum/CHECK pairs it listed. The
authoritative count comes from the test itself, at run time:

```
$ go test -tags=integration -run EveryEnumCheck -v ./internal/payments/adapter/postgres/ | grep checked
checked 22 enum-shaped CHECK constraints (of 56 CHECKs) against 22 mapped domain types
```

Building #311 as filed would have shipped a hand-maintained list **already
incomplete by a factor of four** — the issue's own failure mode. That is the
third consecutive instance in this project:

| | listed | actual | derived how |
|---|---|---|---|
| #308 (T60) Payments subjects | 12 | 22 | re-derived at T60.2 |
| #308 (T60) Competitions subjects | 8 | 21 | re-derived at T60.2 |
| **#311 (T61) enum pairs** | **5** | **22** | the command above |

**I filed #311, and I filed it with a warning that its own list needed
re-verifying.** The warning was right and the list was still wrong — which is
worth separating out, because it shows that *knowing* a list is unreliable does
not make it reliable. The only thing that did was replacing it with a
derivation.

### The inversion, and that it earned itself immediately

`tools/enumconformance` derives side A from the **live database** — a static
parse of `db/migrations` reports `payable_type`'s T6.4 definition and misses
`0031`'s widening — and inverts side B: **an enum constraint absent from the
mapping fails the test.** The mapping may be incomplete; it cannot be incomplete
*and quiet*.

On the first real run it named `games.payment_method`, which was missing from
the mapping I had built by hand. The inversion paid for itself before the ticket
was finished.

## 4. A design constraint found three times in one ceremony, and a fourth time in execution

> **A correction that preserves the claim it corrects will match any checker
> that greps for the claim.**

Instances, in order of discovery:

1. **ADR-0015/0016** preserve *"Escalated — awaiting product decision"*
   verbatim beneath a supersession notice, deliberately. A grep for "Escalated"
   over their prose hits two **resolved** decisions.
2. **T61's sweep** left 17 stale "no Docker daemon" clauses in place beneath
   their refutations, so a grep for the clause still matches all 17.
3. **`docs-index-check`'s own first draft** keyed on the *absence* of "not yet
   written" and immediately false-positived on T54's and T42's corrections.
4. **In execution:** my verification script for the 22-row mapping required a
   single space (`\w+ Type = "…"`), so it reported `EndConditionKind` as having
   one constant where it has three — the declarations are space-aligned. The
   tool's AST parser was never fooled; only my regex was.

The remedy is structural and is now pinned by a test: **assert the positive**
("this cell names a file that exists"), never the negative ("this cell does not
say X"). `TestCorrectionQuotingTheStalePhrasePasses` fails if anyone
re-introduces the negative form.

Instance 4 is the one worth keeping for a different reason: **the AST-based tool
was immune to the mistake that caught the regex.** Parsing beat grepping, in a
sprint whose whole subject is mechanical checks.

## 5. The sprint corrected three of its own written claims

Each was caught by *checking*, and each would have produced a wrong artifact.

### 5a. The Ceremony 1 plan's §1 defect 2 was wrong

The plan said *"4 of 17 ADRs have no `## Status` section"* and prescribed adding
one to each. That was my grep's artifact, not the ADRs':

```
$ for f in docs/adr/*.md; do grep -qE "^- \*\*Status:\*\*" "$f" && echo front || echo heading; done | sort | uniq -c
      6 front
     11 heading
```

All 17 have a status; **two conventions coexist**, and the sweep read one,
silently skipping six files. So the fix became the *reader*, not eleven files —
one ADR edited (`0006`, whose status was prose with no leading token) and a tool
that reads both forms.

**Had the plan been executed as written, the sprint would have rewritten eleven
ADRs to fix a problem they did not have.**

### 5b. A grep nearly made me correct #149 backwards

Scoping T62.3, a grep for the five caller-supplied ownership facts reported the
handler still reading all five — contradicting #149's title and T59's finding.
Acting on it would have "corrected" the issue in the **wrong direction**.

It was counting **comment lines**: `handler.go:132-133`, `:172-173` and
`:269-270` say those fields are *"deliberately NOT read here anymore"*. The real
reads are `:160` and `:284`, both `booking_host_id`. T59's finding was right —
and the correction then refined it further than the title: `booking_host_id`
reaches **two** of the three RPCs the body names, not three.

### 5c. The parser was written against an invented fixture whose comment claimed it was real

`enumconformance`'s `ParseConstraintDef` was written against a
`pg_get_constraintdef` string I made up, under a comment that said *"Pinned
against a real string rather than an invented one."* It matched **0 of 56** live
CHECK constraints.

**The unit tests did not catch it, because they asserted against the same
invention.** What caught it was the integration test's vacuity guard:

```
found only 0 enum-shaped CHECK constraints among 56 total — expected at least
the 21 this test maps … which would make this test vacuous rather than green
```

> A fixture and the code under test, both built from one wrong assumption, agree
> perfectly. Only something that touches the real system breaks the tie.

That is a sharper statement than "test against reality", and it is the reason
the vacuity guard was worth writing before it was needed.

## 6. What went well

- **The ceremony order earned its keep.** Escalations before bookkeeping before
  the sweep before refinement is not ritual: §1 produced an answered decision
  that became T62.2, so running it last would have produced a different and
  smaller sprint.
- **The PE pass declined its own plan.** The Ceremony 1 sign-off refused
  T62.1/T62.4's sequencing — the ADR status lines must precede a tool that reads
  them — and the plan was amended with the change recorded in §7 rather than
  silently edited. Recorded with the caveat the review itself states: there is
  no second party to referee a PE objection here, so the author adjudicated
  their own dispute.
- **Two tests where two claims existed.** The conformance test proves the
  **sets** agree; it does not prove the **path** works. Those are different
  claims and both now have a test, each mutation-verified against `0032`'s
  absence. `CLAUDE.md`'s "a DB-level guard with no DB-level test is a comment"
  applies to the write as much as to the guard.
- **Both vacuity guards did real work.** `enumconformance`'s caught §5c;
  `docsindex` prints its `PhasesInIndex`/`DocsOnDisk` counters on every run so a
  check that examined nothing cannot read as a check that found nothing. Both
  exist because this project's recurring failure is a green gate that tested
  nothing.
- **`gate-coverage` picked up both new tool packages with no edit.**
  ```
  $ make gate-coverage 2>&1 | grep "hold test functions"
  gate-coverage: 47 package(s) hold test functions.
  ```
  45 before. The tool whose design principle is "no list" demonstrated the
  principle on the two tools built to the same principle.
- **The direction of `0032` was checked before widening.** The lazy reading is
  "domain has a value the schema lacks, so widen the schema", which is only
  right if `refunded` *should* be storable. `competition_entries.payment_status`
  already accepts it, `payments.status` accepts it, and
  `PaymentStatusRefunded`'s own comment says it mirrors `payments.Status`. The
  schema was the outlier.

## 7. What complying with T62.2 actually cost, measured on this document

The rule says every quantity carries the command that produces it. Writing this
retro is the first time it was applied, and it changed the document twice:

| figure | my first command | what it returned | the right command |
|---|---|---|---|
| enum CHECK constraints | `grep -ohE "CHECK \(\w+ IN \(" db/migrations/*.sql \| wc -l` | **25** | the test's own run-time log: **22** |
| mapped pairs | `grep -c "{Table:" …_test.go` | **23** | same log line: **22** |

Both were wrong for mundane reasons — the first counts one column twice when a
later migration re-adds its constraint (`payable_type` in `0005` *and* `0031`);
the second matched a `Constraint{Table:` literal on line 143, inside the
scanning loop, which is not a mapping row.

**Neither error would have been caught by the narrow (durations-only) rule, and
neither would have been caught by "cite your method" — the methods were cited
and were wrong.** What caught them was being required to produce a command and
then reading its output against expectation.

The useful generalisation, which is narrower than the rule and worth adding to
it: **a count derived from source text is almost always wrong; prefer a count
the running system reports.** Three of this sprint's wrong figures (§5a, §7
twice) and one of its near-misses (§5b) were greps over source. The figures that
held were produced by a program that had loaded the thing it was counting.

## 8. Recommendations for T63 and beyond

1. **Prefer a figure the running system reports over a figure grepped from
   source** (§7). This is a refinement of T62.2, not a replacement: still carry
   the command, but prefer the command that runs the system. Source greps
   miscounted three times this sprint and the test's own log line was right
   every time.
2. **T63's Ceremony 1 must check whether T62.2 is being followed**, as
   `sprint-process.md` now says. **If it is not, that is evidence for the narrow
   form, not a reason to restate the wide one more firmly.** This retro is
   evidence of compliance in one document written by the rule's author, which is
   the weakest possible evidence; the test is a sprint that did not write it.
3. **A hand-maintained list in a test gets the unmapped-is-a-failure
   inversion.** `enumPairs` is the worked example (§3). The inversion costs one
   extra branch and converts a silent omission into a named failure. Candidates:
   anything in this repo that pairs two independently-maintained sets.
4. **When a plan's ticket rests on a finding, re-verify the finding before
   executing the ticket** (§5a). T62.4 would have rewritten eleven ADRs to fix a
   problem they did not have. The plan is the board of record, not an
   instruction that outranks the tree.

## 9. Sweep and bookkeeping

- **Issues: 5 → 3.** #311 and #314 closed by PR #317. Open and live-verified:
  **#149**, **#145** (both now describing their real gaps, corrected in their
  bodies by T62.3), **#134** (still blocked on an operator and a real screen
  reader). No issue opened.
- **Merge order #316 → #317**, verified against each PR's `merged_at`.
- **Two new gates**, both reachable from `ci-checks`: `make docs-index-check`
  (T62.1) and the enum conformance integration test (T62.5).
- **`Closes #N` still cannot fire** against a non-default base branch. Both
  closures manual, as every closure on this project has been.
- **Verification.** `make ci-checks` green. `make ci-integration` green
  **three** times — **2493 tests, 0 failures** — one of them cold after
  restarting `dockerd` and `go clean -testcache`. Five mutation checks across
  the two new gates (three on `docsindex`, two on `0032`), each naming the right
  defect. Per rule 10 that is evidence, not proof for all time.
- **Two per-container setup steps** are now in `CLAUDE.md`'s gotchas, both
  rediscovered this sprint: an empty `GOPATH/bin` after a re-clone, and
  `go: no such tool "covdata"`. The second cost two full integration runs,
  because **the module-cache `GOROOT` is mode 555 and a `cp` into it reported
  success and left nothing behind.** That is itself an instance of §5c's lesson
  — a command that claims to have worked is not evidence that it did.
- **Self-review for the tenth consecutive sprint.** Both PRs carry review
  comments rather than approvals; GitHub refuses an author's own. This sprint
  merged **a migration on the money path** and **two new gates** with no second
  reader. The #317 review names `0032` and `enumPairs` as the two things most
  worth another pair of eyes, and that remains true.
- **The weakest link, stated plainly:** `enumPairs` is 22 hand-written rows, of
  which **five of my first 21 were wrong** (wrong file or wrong type name),
  caught by verifying each against the tree before the first run. The inversion
  stops *omissions*; it does not stop a plausible-but-wrong row whose values
  happen to match. `DeclaredConstants` errors rather than returning an empty set,
  so a wrong type name fails loudly — but that is a mitigation, not a guarantee.
- **`make security`'s `govulncheck` still cannot reach `vuln.go.dev`.** T61's
  retro flagged that this is itself a claim licensing inaction and should be
  tested rather than quoted. **It was not tested this sprint either.** Recorded
  as owed for the second consecutive sprint, which is the point at which it
  stops being a footnote.
- **This retro sets its own row's Retro cell, and T62.1's gate is why.** The
  draft of this bullet said the opposite — that the retro would leave the row
  alone and `docs-index-check` would therefore go red until T63's Ceremony 1,
  and called that "the gate working". **Running the gate against the retro
  before merging it showed that was the wrong call**: a red gate on the shared
  branch blocks every PR in between, which is how red gets normalised.

  The contradiction turned out to be only apparent. `sprint-process.md`'s rule
  exists because a row must cite the retro's **merge PR number**, which a retro
  PR cannot know. It says nothing about the **path**, which is knowable while
  the PR is open. So this PR sets the Retro cell to
  `docs/process/t62-retro.md` and leaves **Reviews** as "not yet opened" for
  T63's Ceremony 1, along with the narrative. `sprint-process.md` now states
  that distinction explicitly.

  **This is the gate's first real encounter with the rule that created it, and
  it found a latent contradiction in that rule within one sprint.** Worth more
  than the eight rows it was built to catch: a prose rule and a gate that
  disagree will be resolved in favour of whichever one someone notices, and
  noticing is what the gate removes.

## 10. Honest-form outcome sentence

For `HANDOFF.md`'s T62 row, to be carried verbatim rather than strengthened:

> T62 held the first planning ceremony since T59 and delivered all five of its
> tickets. **Its Ceremony 1 found its own escalation sweep mechanically unsound
> three ways, one of them live**: #314 was a decision awaiting an answer that a
> label-keyed sweep could not see, because #311 and #314 carry no labels — D1's
> failure mode reintroduced through a label gap by the sprint that wrote the
> rule against it. Put to the Product Owner before any ticket was refined and
> answered: **widen the quantity rule to all quantities**. A derived check of
> the Docs index then found **eight** sprints of row debt against
> `t60-retro.md`'s predicted two — T54's Retro cell stale since T54, T58's row
> carrying four cells against a six-column header, T59's carrying eight, T60's
> and T61's missing entirely — which argued for a gate rather than a firmer
> prose rule, since six consecutive sprints had ignored the existing one.
> **T62.5 then found a live production defect on its first real run**:
> `registrations.payment_status`'s CHECK accepted only `('unpaid','paid')` while
> the domain has declared `refunded` since T6.5 and `RefundPayment` writes it at
> `service.go:959`, so **every refund of a Social Play Registration had failed
> against a real database with SQLSTATE 23514 for the whole life of the
> feature** — while the Competitions twin twelve lines later worked, because
> `0014` included the value. Fixed by `db/migrations/0032`, with two tests
> because the sets agreeing and the path working are different claims, and
> `grep -rn RefundPayment` over that package had returned nothing at all.
> **#311 was closed deliberately not as filed**: it listed five enum pairs and
> the schema has twenty-two, so the check derives side A from the live database
> and **fails on any unmapped constraint** — an inversion that earned itself on
> its first run by naming a column missing from the mapping. The sprint also
> **corrected three of its own written claims**: the Ceremony 1 plan's "4 ADRs
> have no status" was a grep artifact (all 17 have one; two conventions coexist,
> 11 heading and 6 front-matter) and executing it as written would have
> rewritten eleven ADRs to fix a problem they did not have; a grep nearly had
> #149 corrected backwards, having counted comment lines; and
> `enumconformance`'s parser was written against an **invented** fixture whose
> comment claimed it was real, matched **0 of 56** live constraints, and was
> caught not by its unit tests — which asserted against the same invention — but
> by the integration test's vacuity guard. **A fixture and the code under test,
> both built from one wrong assumption, agree perfectly.** T62's retro is the
> first document subject to T62.2's own rule, and complying changed it twice:
> the first two commands written for it returned 25 and 23 where the running
> system reports 22 and 22, which yields the refinement that **a count derived
> from source text is almost always wrong — prefer a count the running system
> reports**. Verified by `make ci-checks` green and `make ci-integration` green
> three times (2493 tests, 0 failures, one cold), with five mutation checks.
> Self-review for the tenth consecutive sprint, on a sprint that merged a
> migration on the money path and two new gates; `enumPairs`' 22 hand-written
> rows, five of whose first 21 were wrong, remain the weakest link. **The new
> gate's first real encounter with the rule that created it found a latent
> contradiction in that rule**: `sprint-process.md` forbids a retro PR from
> writing the row pointing at it, while `docs-index-check` requires a row to
> name the retro that exists — resolved by distinguishing the **path** (knowable
> while the PR is open, so the retro PR sets it) from the **merge PR number**
> (not knowable, so Ceremony 1 still owns the Reviews cell), and found only
> because the gate was run against the retro before merging it rather than
> after.
