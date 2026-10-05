# T62 Sprint Plan — Ceremony 1 (backlog refinement)

Per `docs/process/sprint-process.md`. PM + PE, held against `HANDOFF.md`,
`docs/process/t61-retro.md` and the three late retros merged in PR #315
(`t55-retro.md`, `t59-retro.md`, `t60-retro.md`), the live issue list, and the
tree at `fcdee67`.

**This is the first planning ceremony since T59.** T60 and T61 each held none,
which is why §3 is the largest section in this document: Ceremony 1 owns the
Docs-index correction, and nobody has run it for eight sprints.

**Ceremony order is the one `sprint-process.md` mandates and is not
cosmetic:** escalations (§1) before bookkeeping (§3) before the issue sweep
(§4) before any ticket is refined (§5). §1 found a decision that needed the
user and got it answered, which changed what §5 contains.

---

## §1 — Escalations, raised before anything else

The rule: *"Ceremony 1 opens by listing every open escalation — every ADR whose
`## Status` is `Escalated`, and every issue labelled `role:product-owner`. For
each one that is still unanswered, the ceremony puts the question to the user
before planning anything else."*

### Result: no ADR is escalated, no issue carries the label — and the sweep is broken in three ways

Run, not read:

```
$ for f in docs/adr/*.md; do ... grep -m1 -A6 '^## Status' "$f" ... done
$ gh/issue_read: labels across all 5 open issues
```

| finding | detail |
|---|---|
| **genuinely escalated ADRs** | **none.** ADR-0015 and ADR-0016 were resolved at T55 (D1 = option (a), D2 = option (b)) |
| **`role:product-owner` issues** | **none** of the five open issues carries it |
| **defect 1 — false positives** | ADR-0015 and ADR-0016 **still contain the words "Escalated — awaiting product decision"** inside their `## Status` section, preserved verbatim beneath a "Superseded by `## Resolution` above" notice. A literal `grep Escalated` over the Status section **hits both.** The preservation is correct and deliberate (T55 kept the question as it stood); the sweep's stated mechanism cannot tell it from a live escalation |
| **defect 2 — false negatives** | **4 of 17 ADRs have no `## Status` section at all** — `0011`, `0013`, `0014`, `0017`. A status-keyed sweep cannot classify them and skips them *silently*. None is in fact escalated (no occurrence of the word), but the sweep cannot know that |
| **defect 3 — label-blindness** | **#311 and #314 carry no labels at all** — the two most recent issues this project filed. A sweep keyed on `role:product-owner` cannot see either |

### Defect 3 was live, not theoretical

**#314 is a decision awaiting an answer, filed by this project, and the
mechanism built to surface decisions awaiting answers could not see it.**

That is D1's failure mode exactly — the one the escalation rule was written to
prevent after D1 sat for 41 sprints — reintroduced through the label gap, by
the sprint that wrote the recommendation. Per the rule's *intent* rather than
its letter, #314 was put to the user before any ticket was refined.

### #314, answered

> **Question.** T61's retro adopted "state durations as ticket ranges, not
> sprint counts" after three wrong figures shipped in `CLAUDE.md`. Two reviews
> objected that the rule is narrower than its evidence: of the five wrong
> numbers, three were **counts**, which the duration rule does not catch. T59's
> retro then found a sixth case with a new twist — `t56-retro.md`'s "23 sweeps"
> figure **cites its method and still does not reproduce**.
>
> **Answer: widen to all quantities.** Any quantity asserted in a process
> document carries the command that produces it — not the technique, the
> runnable command.

The answer is implemented as **T62.2**. The user's choice overrides the
author's own hesitation, which was recorded in #314 and in two reviews: that a
broad rule risks being ignored, per `t56-retro.md`'s explicit rejection of a
general "mutation-test everything" rule. **Recorded here because the objection
was overruled by the person entitled to overrule it, not because it was
withdrawn.** T63's Ceremony 1 should check whether the widened rule is actually
being followed; if it is not, that is evidence for the narrow form and not for
trying harder.

**Defects 1–3 are T62.4.**

---

## §2 — The 0-ticket counter, and whether there is a sprint to plan

The counter is at **0** and stays there. Tickets shipped per sprint, counted
from each sprint's own ticket labels rather than from commits (T55.2+T55.3 and
T56.1+T56.2 each shared a single PR, so a commit count undercounts):

```
$ git log --oneline origin/claude/go-backend-pickleball-7up34j \
    | grep -oE '\bT(5[5-9]|6[0-9])(\.[0-9]+)?:' | sort -u
```

| T55 | T56 | T57 | T58 | T59 | T60 | T61 |
|---|---|---|---|---|---|---|
| 4 | 2 | 2 | 1 | 1 | 2 | — |

**T61 used no ticket numbering at all** — it was one instruction, and it shipped
four PRs fixing three production defects. It is therefore not a 0-ticket sprint
and also not comparable to the others on this axis; recorded as "—" rather than
folded into the series, because an earlier draft of this section wrote
"4, 2, 2, 1, 1, 2, 3" and the 3 was a count of *defects* sitting in a row of
*tickets*. That is the error class T62.2 exists to make harder.

There is a sprint to plan, and §5 plans it.

---

## §3 — Docs-index correction: eight sprints of debt, found by a derived check

`sprint-process.md` requires Ceremony 1 to correct the previous sprint's row
*"before refining any tickets"*, and explains why it cannot be the retro PR's
job: the row must cite the retro's own merge number, which does not exist until
it merges.

**What the rule does not survive is a successor that holds no Ceremony 1.**
T55, T56, T57, T58, T60 and T61 each held none. Only T59 did. So:

### The check, written as a command rather than an inspection

Rather than re-reading the table, the ceremony derived the defects: for every
`| T<N> |` row, does the Retro cell **name a file that exists**, and does the
row have the header's **six** cells?

| phase | defect | how long |
|---|---|---|
| **T54** | Retro cell read "not yet written"; `t54-retro.md` has existed since T54 | **8 sprints** |
| **T55** | same; `t55-retro.md` written at T61 (PR #315) | 7 sprints |
| **T58** | **4 cells** against a 6-column header — `Key ADRs` and `Design` simply absent | 4 sprints |
| **T59** | Retro cell stale **and 8 cells** — two cells beyond the header | 3 sprints |
| **T60** | **no row at all** | 2 sprints |
| **T61** | **no row at all** | 1 sprint |
| T42 (narrative) | "Retro not yet written" though `t42-retro.md` exists | 20 sprints |
| T55 (narrative) | same | 7 sprints |

A markdown table with inconsistent cell counts drops or empties cells when
rendered. **So the artifact CLAUDE.md instructs every session to read first —
"it's the map" — has been mis-rendering for its four most recent sprints**,
and two rows were missing entirely.

**All of it is corrected in this PR**, along with narratives for T59, T60 and
T61 (T56–T58 share a combined entry, which exists). Each narrative carries its
retro's **honest-form outcome sentence verbatim**, lifted programmatically from
the retro files rather than retyped — `sprint-process.md` item 3 requires the
retro's own agreed sentence, and retyping 117 lines of blockquote by hand is
how a stronger version gets introduced.

### The finding, and why it argues for a tool rather than a rule

`t60-retro.md` recommendation 4 proposed a prose fallback: *a sprint that holds
no planning ceremony still writes the previous sprint's row as its first act.*
**This ceremony's evidence says that is not enough.** The debt is not 2 sprints
from one missed ceremony; it is **8 sprints across six missed ones**, and the
T42 narrative note has been stale for twenty. A rule that six consecutive
sprints did not follow will not be followed by the seventh because it has been
written down more firmly.

What *did* find all of it, in one pass, was a derived check — the shape
`CLAUDE.md` already praises in `make gate-coverage`: **both sides computed at
run time, no list in the tool.** That is **T62.1**.

### A design constraint discovered while running it, which the ticket must honour

The first version of the check keyed on the **absence** of the phrase "not yet
written". It immediately produced two false positives — on T54's and T42's own
*corrections*, because each correction **quotes the phrase it is retiring**
("this cell read \"not yet written\" until T62's Ceremony 1").

That is the **third** instance of this exact pattern inside this one ceremony:

1. ADR-0015/0016 preserve "Escalated" verbatim, so the escalation sweep hits
   them (§1 defect 1);
2. T61's sweep left 17 stale test-header clauses in place beneath their
   refutations, so a grep for the clause still matches all of them;
3. this check, on its own corrections.

**The pattern: a correction that preserves the claim it is correcting will
match any checker that greps for the claim.** The fix is structural — assert
the **positive** ("the cell names a file that exists") rather than the negative
("the cell does not say X"). Re-run in the positive form, the check reports
clean. T62.1 must be built this way, and T62.4 must fix §1's sweep the same
way.

---

## §4 — Issue sweep: both questions, every issue

Per the rule, each open issue gets **two** questions: is the blocker still in
place, *and* is what it describes still true of the tree — verified against the
code, never the issue's own prose.

| issue | blocker still in place? | premise still true? |
|---|---|---|
| **#314** | no — answered in §1 | n/a (it *is* a decision). Closed by **T62.2** |
| **#311** | no — actionable now | **✓ verified.** All five enum/CHECK pairs exist and agree: `payments.method`, `payments.status`, `registrations.status`, `waitlist_entries.status`, `competition_entries.status`. Derived by parsing each `CREATE TABLE`'s column clause and each domain file's const block and comparing the sets — **the issue's own warning to re-verify its list was right to be there, and the list holds**. They agree *by this inspection, not by any gate*, which is the issue |
| **#149** | **yes** — Payments has no Booking port; closing it means new cross-context read ports | **✗ DRIFTED, for the second time** |
| **#145** | **yes** — needs a real IdP tenant, which no coding session can provision (ADR-0013) | **✗ DRIFTED** |
| **#134** | **yes** — needs one operator, a real browser and a real screen reader; no AT device here | **✓ verified.** All three screens exist; all three routes are still in `ROUTES_UNDER_TEST` with the path-identity check against the real router intact; `:focus-visible` still declared on all three. Item 2 (focus-indicator contrast, outline colour equal to the primary button's own fill) remains the plausible real defect the issue says it is |

**The premise check fires on two of five.** Its first firing ever was #149 at
T59; this is its second and third.

### #149 — drifted again, because T59.2 corrected the title and not the body

`t59-retro.md` §3 is the finding; this sweep confirms it is still true. #149's
**body** still states:

- all five caller-supplied ownership facts, presented as open — four were
  closed by T16.2 and T17.1;
- *"this codebase has never persisted Game-Admin or Competition-Admin
  assignments at all"* — **false since T14.4**
  (`db/migrations/0020_socialplay_game_admins.sql`, and
  `0021_competitions_competition_admins.sql` at T15.3);
- a dependency ordering built on that false claim — which **T59.2 knew was
  stale and recorded in `t59-sprint-plan.md`**, a document nobody reads when
  picking up an issue.

The correction exists, in a comment below the body. GitHub renders the body
first. **T62.3.**

### #145 — drifted, and nobody had checked

New this ceremony. #145's consequence 1 reads:

> Every Facility created before a real IdP existed becomes unmanageable. No
> principal's subject will ever match the UUID stored in `facilities.owner_id`,
> so `AddCourt`/`AddCameraLink`/`AttestCameraConsent` will reject the real
> owner forever.

**That mechanism no longer exists.** Verified:

- all five contexts have an identity adapter —
  `internal/{booking,competitions,facilities,payments,socialplay}/adapter/identity`;
- `internal/facilities/adapter/grpcapi/handler.go:80` calls
  `h.svc.ResolveActorUserID(ctx, subject)`, so the subject is resolved to a
  `User.ID` **before** any comparison against `facilities.owner_id` — the
  comparison is uuid-to-uuid;
- `identity_users.subject` exists (`db/migrations/0019_identity_subject.sql`),
  so consequence 2's *"until T12.9's column exists"* precondition is satisfied.

What genuinely remains is **narrower and still real**: `identity_users.subject`
must be backfilled for rows that predate a real IdP, which structurally needs
that IdP. The issue should say that and stop saying the rest. **T62.3.**

---

## §5 — Tickets

Five. One is Go code; four are process or issue-correction work that this
ceremony's own findings generated.

### T62.1 — `make docs-index-check`: derive the Docs index's correctness

**Story.** As a session starting work, I want the Docs index to be checkably
accurate, so that the map CLAUDE.md tells me to read first is not silently
wrong for the four most recent sprints.

**Description.** §3's finding. Eight sprints of row debt accumulated because
correction is assigned to a ceremony that did not run. `t60-retro.md`
recommendation 4 proposed a firmer prose rule; §3 argues a tool instead, on the
evidence that six consecutive sprints ignored the existing rule.

**Instructions.**
1. New tool under `tools/docsindex` with entry point `cmd/docsindex`, mirroring
   `tools/gatecoverage`'s structure, and a `make docs-index-check` target
   **reachable from `ci-checks`**.
2. It must fail, naming each offender, when any of these holds:
   - a `docs/process/t<N>-retro.md` or `t<N>-sprint-plan.md` exists and
     `HANDOFF.md` has no `| T<N> |` row;
   - a row's Retro cell does not **name** the retro file that exists for that
     phase (or names one that does not exist);
   - a row's cell count differs from the table header's;
   - a Task-backlog narrative section says "retro not yet written" for a phase
     whose retro file exists.
3. **Assert positively, never by absence of a phrase.** §3's design constraint:
   a correction quotes the claim it retires, so any negative grep false-positives
   on it. The check must ask "does this cell name an existing file?", not "does
   this cell avoid saying X". **Add a test that pins exactly this** — a row
   whose cell both names the real file *and* quotes "not yet written" inside a
   correction must PASS.
4. **No list of phases in the tool.** Both sides computed at run time: side A
   from `docs/process/*.md`, side B by parsing `HANDOFF.md`. Per `CLAUDE.md`,
   adding a list is the one change that would defeat it.
5. Covered by `make test-tools`. Verify by mutation: re-introduce one of §3's
   eight defects and confirm the tool names it.

6. **Also check ADR status**, per the PE sign-off below: the same tool reads
   `docs/adr/*.md`'s machine-readable status line (added by T62.4) and fails on
   an ADR it cannot classify. One tool over `docs/`, not two — the shapes are
   identical, and a second tool for one more file is the worse outcome.

**NFRs.** Docker-free, codegen-free — this must run in the cheapest gate.
**Points: 5.** `role:principal-engineer`, `type:chore`.
**Depends on T62.4 steps 1–2** (the ADR status lines must exist before a tool
can read them). **Sequenced second**, per the PE sign-off.

### T62.2 — Widen the quantity rule to all quantities (closes #314)

**Story.** As a future reader of a process document, I want every quantity in
it to carry the command that produces it, so that a figure can be re-checked
rather than trusted.

**Description.** §1's answered escalation. Six wrong figures in four days
across three artifacts; three were counts, which T61's duration-only rule does
not cover; the sixth (`t56-retro.md`'s "23 sweeps") **cited its method and
still does not reproduce**, which is why the rule is "carry the command", not
"cite your method".

**Instructions.**
1. Amend `sprint-process.md`'s retro/plan conventions: **any quantity asserted
   in a process document carries the runnable command that produces it**, inline
   or in an adjacent fenced block. A technique named in prose does not satisfy
   it — `t56-retro.md` is the worked counter-example.
2. Record the scope decision and the dissent: `t56-retro.md` recommendation 4
   explicitly rejected a broad rule as one that would be ignored, and the
   author of #314 shared that concern. The Product Owner chose the wide form.
   **Both the choice and the objection go in the text**, so T63 can evaluate it
   rather than relitigate it.
3. State the one exemption plainly: a quantity **quoted from another document**
   is attributed, not re-derived — and if its method does not reproduce, that is
   recorded rather than the figure repeated. `t59-retro.md`'s treatment of "23
   sweeps" is the worked example.
4. Close #314 with the decision and the reasoning on the issue.

**NFRs.** No new gate — this is a convention. T62.1 covers the Docs index
mechanically; this covers prose, where no tool is proposed.
**Points: 2.** `role:principal-engineer`, `type:chore`.

### T62.3 — Correct #149 and #145 in their bodies, adjacent and unmissable

**Story.** As whoever picks up #149 or #145 next, I want the issue's body to
describe the gap that exists, so that I do not re-litigate facts closed sprints
ago.

**Description.** §4's two drifted premises. #149 is the second attempt: T59.2
corrected the title and posted a comment, and the body still misleads.
`t59-retro.md` §3 and recommendation 1 establish the shape — **a correction
must sit where the false claim sits, not below it.**

**Instructions.**
1. **#149:** edit the body. Mark the four closed facts closed, naming T16.2 /
   T17.1 per field. Strike the *"never persisted Game-Admin or
   Competition-Admin assignments at all"* sentence in place, citing
   `0020_socialplay_game_admins.sql` (T14.4) and
   `0021_competitions_competition_admins.sql` (T15.3). Strike the dependency
   ordering that rests on it. **Strike, do not delete** — keep the original
   text visible as the record, following T61's convention and not T59.2's.
2. **#145:** edit the body. Consequence 1's mechanism is gone — cite
   `internal/facilities/adapter/grpcapi/handler.go`'s
   `ResolveActorUserID` call and the five per-context identity adapters.
   Consequence 2's *"until the column exists"* precondition is satisfied by
   `0019_identity_subject.sql`. State what remains: backfilling
   `identity_users.subject` for pre-existing rows, which needs a real IdP.
3. Neither is closed. Both still name a real gap.
4. Evidence, not assertion: file and line for every claim.
5. Re-title either if its title now overstates what remains.

**NFRs.** The correction must be legible **before** the stale text, not after
it — §3's design constraint restated for prose.
**Points: 2.** `role:principal-engineer`, `type:chore`.

### T62.4 — Make the escalation sweep able to see what it is looking for

**Story.** As a Ceremony 1, I want the escalation sweep to be mechanically
sound, so that a decision awaiting an answer cannot be invisible to the step
that exists to surface it.

**Description.** §1's three defects, one of which was live: **#314 was an
unanswered decision that the sweep could not see.** That is D1's failure mode,
which this rule was written to prevent.

**Instructions.**
1. **Defect 1 (false positives).** ADR-0015/0016 preserve "Escalated" verbatim
   under a supersession notice. Give every ADR a machine-readable status — a
   single `**Status:** <one word>` line at a fixed position — and have the sweep
   read *that*, not a grep over prose. Positive assertion, per §3.
2. **Defect 2 (false negatives).** `0011`, `0013`, `0014` and `0017` have no
   `## Status` section. Add one to each, reflecting reality (none is escalated).
   The sweep must **fail loudly** on an ADR it cannot classify rather than skip
   it silently — a silent skip is what made this invisible.
3. **Defect 3 (label-blindness).** The sweep keys on `role:product-owner`, and
   #311/#314 have no labels. Either label every open issue, or — better — have
   the sweep consider **every open issue** and require the ceremony to state,
   per issue, whether it awaits a decision. A label is a thing someone must
   remember; the issue list is not.
4. Amend `sprint-process.md`'s escalation rule to match, and record all three
   defects there with the §1 evidence, so the next ceremony inherits the finding
   and not just the fix.
5. **The ADR-status check is folded into T62.1's tool — decided, not deferred.**
   The PE pass settled this rather than leaving it as a "consider": the two
   checks are the same shape, and the mechanical half belongs in a gate rather
   than in a ceremony's discipline, for §3's reason. **This ticket owns adding
   the status lines (steps 1–2); T62.1 owns reading them.**

**NFRs.** Steps 1–2 are a prerequisite for T62.1, so this ticket is **sequenced
first**. Step 3 may land as prose; if it does, say so explicitly — this
ceremony's §3 finding is that prose rules here have a poor record.
**Points: 3.** `role:principal-engineer`, `type:chore`.

### T62.5 — Extend the enum/CHECK conformance test to the other four pairs (closes #311)

**Story.** As a maintainer adding a value to a domain enum, I want a gate to
refuse it if the schema cannot hold it, so that the next `payable_type` cannot
be broken for 51 sprints with every gate green.

**Description.** #311, premise verified in §4. `payable_type` was the only pair
that had diverged and is the only one with a conformance test; the other four
agree **by this ceremony's inspection, not by any gate** — exactly the state
`payable_type` was in before T61 ran the suite.

**Instructions.**
1. One conformance test per pair: `payments.method`, `payments.status`,
   `registrations.status`, `waitlist_entries.status`,
   `competition_entries.status`.
2. **Derive each set from the source**, as
   `payable_type_conformance_integration_test.go` does — parse the domain file's
   const block; never list the values. #311's own body explains why: a
   hand-maintained table would be written by the same ticket that forgets the
   migration.
3. Each test needs its negative control: a value the domain rejects must be
   refused by the constraint too. Without it a CHECK accepting anything passes.
4. Duplicate the AST helper per package rather than sharing it, following this
   repo's firm convention for integration-test helpers (`applyMigrations`,
   `waitForReady`, `newTestPool` are each duplicated across four packages, with
   the rationale in their comments). **State the choice rather than inheriting
   it**, per #311's open question 1.
5. #311's open question 3 asked whether a single Docker-free SQL-parsing tool
   would dominate five tests. **Answer it in the PR** rather than silently
   picking: it would catch a *new* pair automatically, which five tests cannot,
   but it must parse SQL rather than query Postgres. Recommend costing it as a
   follow-up and shipping the five tests now — a known-good pattern beats a
   speculative tool while four pairs sit unguarded.
6. **Run `make ci-integration`**, with repeats, per `CLAUDE.md`'s T61 gotcha.
   `make vet-integration` cannot execute a CHECK constraint.

**NFRs.** Rule 10: no "proven" language on one run. Rule 4: these tests *are*
the both-halves-in-sync check.
**Points: 5.** `role:principal-engineer`, `type:chore`.

---

## §6 — Deliberately out of scope, with the decision recorded

Per `t57-retro.md`: a chosen exclusion gets a documented decision, not silence.

- **`t60-retro.md` recommendation 1 — make the mutation check a rule.** Three
  consecutive sprints applied it voluntarily and found something every time, so
  the evidence is good. **Excluded anyway:** T62.2 and T62.4 each add a process
  rule, and a sprint that adds three at once is how rules become wallpaper —
  which is precisely `t56-retro.md`'s stated reason for keeping its own version
  narrow. Carried to T63, and this paragraph is the record that it was chosen,
  not forgotten.
- **#134's WCAG pass.** Still blocked on a real screen reader and an operator;
  no AT device here. Premise re-verified clean in §4, so it is not drifting
  while it waits.
- **#149's and #145's actual closure.** T62.3 corrects what they say; neither
  gap can be closed without, respectively, new cross-context read ports and a
  real IdP tenant.
- **`make security`'s `govulncheck`.** Cannot reach `vuln.go.dev` from this
  environment. `t61-retro.md` §9 flagged that this is now itself a claim that
  licenses inaction and should be tested rather than quoted. **Not tested this
  ceremony** — recorded as owed, because writing it down again without checking
  is the exact behaviour T61's finding is about.

## §7 — What this ceremony produced

1. **#314 answered** by the Product Owner (widen), after §1 found it invisible
   to the mechanism meant to surface it.
2. **Eight sprints of Docs-index debt cleared** — T54, T55, T58, T59 rows
   corrected (two of them malformed against the header), T60 and T61 rows
   created, T42 and T55 narrative notes corrected, and narratives written for
   T59, T60 and T61 carrying each retro's honest-form sentence verbatim.
3. **A new T62 row**, with its Retro and Reviews cells honestly "not yet
   written" / "not yet opened".
4. **Three defects found in the escalation sweep** by running it, one of them
   live.
5. **The premise check fired on two of five issues** — its second and third
   firings ever.
6. **A design constraint discovered three times in one ceremony**: a correction
   that preserves the claim it corrects will match any checker that greps for
   the claim. Assert positively. Written into T62.1 and T62.4 as a requirement
   with a test.
7. **Five tickets**, T62.1–T62.5, and one documented exclusion.
8. **A sequencing change made by the PE pass on this plan, after it was
   drafted:** T62.4's ADR status lines must precede T62.1's tool, and the ADR
   check folds into that tool rather than remaining a human step. The plan as
   first written implied the opposite order and left the fold as a "consider".
   Recorded here as a change rather than silently edited, because
   `sprint-process.md` makes this document the board of record — and recorded
   with the caveat the review states plainly: **there is no second party to
   referee a PE objection on this project, so the author adjudicated their own
   dispute.** That is the standing weakness, not a property of this call.

### Execution order

**T62.4 (steps 1–2) → T62.1 → T62.3 → T62.5 → T62.2 (and T62.4 step 3).**
T62.3 is first by *value* (per the PM pass: #149 and #145 actively mislead
their next reader, the only ticket whose delay has an ongoing cost) and can run
in parallel with the tool work, since it touches no code.
