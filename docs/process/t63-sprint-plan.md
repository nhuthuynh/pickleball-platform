# T63 Sprint Plan — Ceremony 1 (backlog refinement)

Per `docs/process/sprint-process.md`. PM + PE, held against `HANDOFF.md`,
`docs/process/t62-retro.md`, `docs/process/t62-sprint-plan.md`, the live issue
list, and the tree at `4621975`.

**This plan is subject to T62.2's rule** — every quantity below carries the
runnable command that produces it. §2 is what happened when the ceremony applied
that rule to the previous sprint's own claims.

**Ceremony order, as mandated:** escalations (§1) → bookkeeping (§3) → issue
sweep (§4) → tickets (§5). §1 and §2 both changed what §5 contains.

---

## §1 — Escalations, using T62.4's corrected mechanism

T62.4 rewrote this sweep after T62's ceremony found it unsound three ways. First
run of the corrected version.

### ADRs: none escalated, and the gate now says so

```
$ go run ./cmd/docsindex
docs-index-check: 64 phase row(s) in HANDOFF.md, 107 process doc(s) on disk, 17 ADR(s).
docs-index-check: OK — the Docs index, the narratives and every ADR status agree with the tree.
```

`docs-index-check` **fails on any ADR it cannot classify**, reading both status
conventions, so "OK across 17 ADRs" is now a positive statement that every ADR
has a classifiable status — not the silence that let T62's sweep skip six files.

**A fifth instance of this sprint's recurring lesson, found immediately.** The
ad-hoc loop written to double-check the gate reported **16** of 17:

```
$ for f in docs/adr/*.md; do v=$(grep -m1 -oE "^- \*\*Status:\*\* \**[A-Za-z]+" "$f" \
    || grep -m1 -A1 "^## Status" "$f" | tail -1); echo "$v"; done \
  | grep -oE "Accepted|Escalated|Superseded|Proposed|Rejected" | sort | uniq -c
      16 Accepted
```

It missed ADR-0012, whose `## Status` is followed by a blank line, so `tail -1`
took the blank. The tool's regex matches `\n+` and was never fooled. **The
parser was right and the grep was wrong** — the fourth time this sprint and the
fifth across T62–T63, which is `t62-retro.md` recommendation 1 earning itself
again.

### Issues: every open one considered, not every labelled one

T62.4's step 3. Per-issue disposition is in §4; none of the five awaited a
*product* decision, and the two that awaited an *engineering* one (#311, #314)
were already answered — but **not closed**, which is §2.

## §2 — The finding this ceremony exists to record

**T62's retro claimed two issues were closed and "live-verified". Neither was
closed. The live count was 5, not 3.**

```
$ (list_issues state=OPEN) -> totalCount
5
```

`Closes #N` cannot auto-fire on this project — PRs merge into
`claude/go-backend-pickleball-7up34j`, not the default branch — a fact
`t62-retro.md` §9 **states in the same section as the false claim**. Nobody
closed them by hand. Both are closed now, by this ceremony, with their rationale
on the issues themselves.

### Why this is the sharpest evidence available for T62.2's own rule

Of the quantities in T62's retro, this is one of the few that carried **no
command**. It is the one that was wrong. The word used was *"live-verified"*,
for something not verified at all.

The rule adopted at T62.2 says a quantity carries the command that produces it.
This figure's command would have been one line, would have taken a second, and
would have returned `5`.

### It also shows the hole is worse than T62's retro recorded

`t62-retro.md` §8 recommendation 5 — added by that PR's own review — says the
rule has a hole: **a quantity describing an action the sprint took cannot be
re-derived by a command later.** It proposed exempting such claims, carrying
instead the command that *was* run plus where its output is recorded.

**That proposal would not have caught this.** "Issues: 5 → 3" was not a
description of an action the sprint took; it was a description of an action the
sprint **believed** it had taken and had not. An exemption for action-claims
exempts exactly the claim that was false.

So the fix is the opposite of an exemption:

> **A claim about an action carries the command that re-reads the live state** —
> precisely because the action's own description is the thing that cannot be
> trusted.

"I closed two issues" is checked by listing open issues, not by citing the
closing call. That is **T63.1**.

### And the compliance audit was itself incomplete

PR #318's review audited ten quantities in T62's retro against T62.2 and
**did not include this one**. A hand-enumerated audit of a rule about
hand-enumerated figures, which missed a figure. Same failure, one level up —
which is why T63.1 asks for the audit to be derived rather than listed.

## §3 — Docs-index correction

Previous sprint's row, per the rule, and T62.1's gate now enforces it.

| what | done |
|---|---|
| T62 **Reviews** cell | PR #316 → #317 → #318, verified against each PR's `merged_at` |
| T62 **narrative** | added, carrying the retro's honest-form sentence verbatim, lifted programmatically rather than retyped |
| T62 retro's false closure claim | **corrected in place**, in `t62-retro.md` §9, where the claim sits (`t58-retro.md` recommendation 3) rather than only here |
| new **T63** row | added, with Retro/Reviews honestly "not yet written" / "not yet opened" |

```
$ python3 -c "
src=open('docs/process/t62-retro.md').read().split(chr(10))
i=next(k for k,l in enumerate(src) if 'Honest-form outcome sentence' in l)
out=[]
for l in src[i:]:
    if l.startswith('> '): out.append(l)
    elif out: break
print(len(out), 'blockquote lines')"
52 blockquote lines
```

The Retro cell was already set by T62's own retro PR — the path/number
distinction T62's retro forced into `sprint-process.md`. This ceremony adds only
what a retro PR structurally cannot know: the merge numbers.

```
$ go run ./cmd/docsindex | tail -1
docs-index-check: OK — the Docs index, the narratives and every ADR status agree with the tree.
```

**Note what did not happen this time:** no eight-sprint debt, no malformed rows,
no missing rows. T62.1's gate has been in `ci-checks` for one sprint and the
Docs index is clean. That is one data point, not a trend.

## §4 — Issue sweep: both questions, every issue

| issue | blocker still in place? | premise still true? |
|---|---|---|
| **#311** | n/a | ✅ delivered by T62.5, **not as filed** (it listed 5 enum pairs; there are 22). **Closed by this ceremony** |
| **#314** | n/a | ✅ answered (widen) and implemented by T62.2. **Closed by this ceremony** |
| **#149** | **yes** — Payments has no read port into Booking | ✅ **premise now accurate.** T62.3 corrected the body: four facts closed by T16.2/T17.1, `booking_host_id` the only one still read, at `handler.go:160` and `:284` |
| **#145** | **yes** — needs a real IdP tenant (ADR-0013) | ✅ **premise now accurate.** T62.3 corrected it: the subject-vs-uuid comparison was replaced by the resolution seam; what remains is backfilling `identity_users.subject` |
| **#134** | **yes** — needs an operator and a real screen reader | ✅ re-verified at T62's ceremony, unchanged since |

**The premise check fired on zero of five this sprint**, for the first time since
it was adopted — because T62.3 corrected the two that had drifted. Worth
recording as the rule working rather than as nothing happening.

**Issues after this ceremony: 3** (#149, #145, #134), all three blocked on
things no coding session can supply: a cross-context port that is a real design
change, a real IdP tenant, and a human with a screen reader.

## §5 — Tickets

Four. One is the highest-value item this project has had in several sprints and
was invisible until this ceremony ran a command it had twice only quoted.

### T63.2 — Fix the 7 npm vulnerabilities `make security` has never reported

**Listed first because it is first by value** (see the PM ordering below); the
numbering follows the ceremony's discovery order.

**Story.** As an operator of this platform, I want the dependency security gate
to run and pass, so that known-vulnerable packages are not shipped in the web
client.

**Description.** `docs/process/t61-retro.md` §9 and `t62-retro.md` §9 both
recorded that `make security`'s `govulncheck` cannot reach `vuln.go.dev` from
this environment, and both recorded it as *owed* without testing it. **T61's own
recommendation 1 is that a claim licensing inaction is the least likely to be
re-checked and therefore the most valuable to re-check.** This ceremony tested
it.

The claim is **true**:

```
$ make security-go
govulncheck: fetching vulnerabilities: Get "https://vuln.go.dev/index/modules.json.gz": Forbidden
```

**And it had masked something.** `security-go` runs before `security-npm`, so its
failure meant **the npm half of the gate had never run at all**. The Makefile
already carries the designed response — `SKIP_GOVULNCHECK=1`, which warns loudly
and removes the report — and with it:

```
$ SKIP_GOVULNCHECK=1 make security
*** WARNING: SKIPPING the Go vulnerability scan (SKIP_GOVULNCHECK=1). ***
NOTE: no Go scan report present — gating on npm findings ONLY.
FAIL: 5 new gating finding(s).
```

```
$ cd web && npm audit --json   # severity tally
{'moderate': 2, 'high': 5, 'total': 7}
```

Seven, five of them **high**, **every one with `fixAvailable: true`**:
`@redocly/openapi-core`, `brace-expansion`, `js-yaml`, `nanoid`, `undici` (high);
`@vitest/mocker`, `vitest` (moderate). Only `vitest` is a direct dependency.

**So two sprints of "govulncheck is owed" concealed five high-severity
findings in shipped web dependencies.** The unreachable Go database licensed not
running the gate, and not running the gate hid the half that works.

**Instructions.**
1. Resolve all seven. `fixAvailable: true` on each means npm believes no
   breaking change is needed — **verify that claim rather than trusting it**:
   run the fix, then `make test-web` and `make build-web`, and report both.
2. Where a fix forces a major bump, **do not take it silently**: say so, and
   either justify the bump or add an explicit, justified baseline entry, which is
   what `tools/vulngate` asks for in its own failure message.
3. Re-run `SKIP_GOVULNCHECK=1 make security` and report the result. A green npm
   half is the acceptance criterion; the Go half stays unrun here.
4. **Record the Go gap honestly and stop re-deriving it.** Add to `CLAUDE.md`'s
   gotchas: `vuln.go.dev` is `Forbidden` from this environment (tested at T63,
   not quoted), `SKIP_GOVULNCHECK=1` is the designed response, it must never be
   set in CI, and **running the rest of the gate is not optional just because
   one half cannot run.**
5. Do **not** add `SKIP_GOVULNCHECK=1` to any `ci-*` target. The flag is for a
   human in an environment that cannot reach the database; CI is not that.

**NFRs.** `make test-web` and `make build-web` must stay green — a security fix
that breaks the client is not a fix. No baseline entry without a written reason.
**Points: 5.** `role:principal-engineer`, `type:bug`.

### T63.1 — Extend T62.2 so a claim about an action carries a live-state command

**Story.** As a reader of a process document, I want a claim that something was
*done* to carry the command that re-reads the world, so that a sprint cannot
assert an outcome it did not achieve.

**Description.** §2's finding. T62's retro asserted two issues closed and
"live-verified" when neither was; the figure carried no command; and the review's
own compliance audit missed it. `t62-retro.md` recommendation 5 proposed
exempting action-claims, which would have exempted the false one.

**Instructions.**
1. Amend `sprint-process.md`'s quantity rule: **a claim that an action was taken
   carries the command that re-reads the resulting live state**, not the command
   that performed it and not a description of it. Worked examples to include:
   "issues closed" → list open issues; "merged in this order" → each PR's
   `merged_at`; "N tests pass" → the run's own summary line.
2. Fold in `t62-retro.md` recommendation 1 as part of the same rule rather than
   a separate one: **prefer a count the running system reports over a count
   grepped from source.** Five instances across T62–T63 now, the most recent in
   §1 of this plan.
3. **Reject the exemption** recommendation 5 proposed, and say why in the text:
   an action-claim is the *most* in need of a command, not the least, because the
   actor's own account of what they did is the unreliable part.
4. State the limit honestly: this rule cannot make a session run the command. It
   can only make the omission visible on the page. §2 is the worked example of
   what the omission costs.
5. **Do not build a gate for this.** Considered and rejected: a checker for
   "every number has an adjacent command" would need to tell a quantity from any
   other number in prose, and a gate that mis-fires on prose would be disabled
   within a sprint. Record the decision so T64 does not re-litigate it.

**NFRs.** The amendment is prose; `t62-retro.md` §9's correction is the evidence
it rests on and must be cited from it.
**Points: 2.** `role:principal-engineer`, `type:chore`.

### T63.3 — Make the mutation check a rule (twice deferred)

**Story.** As a reviewer, I want "verified by mutation" to be an expectation
rather than a habit, so that a guard shipped without one is visibly incomplete.

**Description.** `t60-retro.md` recommendation 1, deferred by T62 §6 with a
stated reason (two process rules in one sprint is how rules become wallpaper).
**This is its second deferral**, and the evidence has only grown: T61, T62 and
T63 all applied mutation verification voluntarily and it found something every
time — including `enumconformance`'s invented fixture, which no unit test caught.

**Instructions.**
1. Amend `sprint-process.md`: **a change that adds or relies on a guard is
   verified by removing the guard and recording what fails.** Scope it to guards
   — constraints, triggers, validation, gates — not to all code, which is what
   `t56-retro.md` rejected as unenforceable.
2. Require the *output*, not the claim: the review quotes the failure the
   mutation produced. T62's five mutation checks are the worked examples.
3. **Add the one case that is easy to get wrong:** a mutation that fails to
   *compile* proves nothing. T60.2 recorded this; it belongs in the rule.
4. Cross-reference the vacuity-guard pattern — `enumconformance`'s
   "found only 0 … which would make this test vacuous rather than green" — as
   the complement: mutation proves a test *can* fail, a vacuity guard proves it
   *is looking at something*.

**NFRs.** If this is deferred a third time, the deferral must say what evidence
would change the answer, since "more instances" has now been satisfied twice.
**Points: 2.** `role:principal-engineer`, `type:chore`.

### T63.4 — Give `enumPairs` a guard against a plausible-but-wrong row

**Story.** As a maintainer, I want the enum mapping's rows to be verified against
the tree, so that a row pointing at the wrong type cannot silently compare the
wrong sets.

**Description.** `t62-retro.md` §9 names this the weakest link in an otherwise
derived check: `enumPairs` is 22 hand-written rows, and **five of the first 21
were wrong** (wrong file or wrong type name). The unmapped-is-a-failure
inversion stops *omissions*; nothing stops a row whose file and type both exist
but are the wrong ones for that column.

**Instructions.**
1. `DeclaredConstants` already errors when it finds no constants, so a wrong
   *type name* fails loudly. Confirm that with a test if one does not exist.
2. The uncovered case is a row naming a real type whose values happen to match
   the column. Add the cheapest available check: assert each mapped type's
   **declaring file is in the bounded context that owns the table** — a
   `payments.*` column must map into `internal/payments/domain`. That catches
   cross-context mis-mappings, which is the shape four of the five errors had.
3. **State what this does not catch** — two same-context enums with identical
   value sets — and whether that is worth more machinery. A one-line answer in
   the PR is enough; do not build for it speculatively.

**NFRs.** Docker-free: this is a check over the mapping and the source tree, not
the database, so it belongs in `tools/enumconformance`'s unit tests.
**Points: 3.** `role:principal-engineer`, `type:chore`.

### PM value ordering, if the sprint has to be cut

1. **T63.2** — five high-severity findings in shipped dependencies, invisible
   for the whole life of the project. Nothing else here has a cost that accrues
   while it waits.
2. **T63.1** — the rule that would have caught §2, and §2 happened last sprint.
3. **T63.4** — narrows a known weak link in a gate that is now load-bearing.
4. **T63.3** — valuable and twice-deferred, but it codifies a practice already
   being followed voluntarily, so delay costs least.

**PE sign-off:** no sequencing objection this time. T63.2 is independent;
T63.1 and T63.3 both touch `sprint-process.md` and should land in that order to
avoid a conflict; T63.4 is independent of all three.

## §6 — Deliberately out of scope, with the decision recorded

- **Closing #149 or #145.** Both now describe their real gaps (T62.3) and both
  need something no coding session can supply: a read port into Booking that is
  a genuine design change, and a real IdP tenant.
- **#134's WCAG pass.** Still needs an operator and a real screen reader.
  Premise re-verified at T62's ceremony and unchanged.
- **Running `govulncheck` here.** Tested, `Forbidden`, not a solvable problem
  from inside this environment. T63.2 covers everything that *is* solvable and
  records the rest so it stops being re-derived every sprint.
- **A gate for T63.1's rule.** Rejected with reasons in T63.1 instruction 5.

## §7 — What this ceremony produced

1. **T62.4's corrected escalation sweep, run for the first time.** No ADR
   escalated; every open issue considered rather than every labelled one.
2. **A false claim in the previous sprint's retro, found and corrected** — and
   with it the strongest available evidence for T62.2's rule, since the one
   figure carrying no command is the one that was wrong.
3. **#311 and #314 actually closed**, with rationale on each.
4. **The `govulncheck` claim tested rather than quoted** for the first time in
   three sprints — and the discovery that its failure had hidden **7
   vulnerabilities, 5 of them high**, in the npm half of the same gate.
5. **The premise check fired on zero of five issues**, for the first time, because
   T62.3 corrected the two that had drifted.
6. **A fifth instance of "the parser was right and the grep was wrong"** (§1),
   this time in a command written to double-check the parser.
7. **Four tickets**, T63.1–T63.4, and four documented exclusions.
