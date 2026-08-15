# T20 Sprint Retro

Ceremony 3 per `docs/process/sprint-process.md`, six-role team (briefs:
`docs/agent-operating-handbook.md` Part B), held against
`docs/process/t20-sprint-plan.md` (§A0–§A8, a 0-ticket sprint — the first in
this project's history), `docs/process/t19-retro.md` and
`docs/process/t18-retro.md` as the precedent and rigor bar, `HANDOFF.md`, and
the real PR/issue history on `nhuthuynh/white-label` (GitHub-side name
`pickleball-platform`) — PR #218, no tickets, no new issues.

Every count, timestamp, and blocker claim below was pulled from GitHub's own
API fields and from direct toolchain runs against the live tree at `bbc9c90`
— never inferred from the sprint plan's own account of itself, and never
assumed to still hold because the plan already checked it once (CLAUDE.md
rule 10).

**Verification performed before writing a single finding.** `git status`
showed a clean worktree at the shared branch's tip (`bbc9c90`, T20's plan's
own merge, PR #218) before this retro's branch was cut. `make generate &&
go build ./... && go vet ./... && make fmt-check && make vet-integration &&
make test-domain && make test-adapters && make test-cmd && make test-platform
&& make gate-coverage` were all run directly, not assumed from the plan's
own account:

```
go build ./...                 # clean
go vet ./...                   # clean
make fmt-check                 # OK — gofmt clean
make vet-integration            # clean
make test-domain                # ok, all 12 packages
make test-adapters              # ok, all 22 packages
make test-cmd                   # ok
make test-platform              # ok
make gate-coverage               # OK — all 42 package(s) executed by
                                 #   "ci-checks" (unchanged — no ticket
                                 #   landed a new package)
```

**No mutation checks are owed this retro, stated rather than silently
omitted.** T20 shipped zero tickets, so there is no new production code path
to mutation-test — a different, honest shape from T18's and T19's retros,
both of which had a merged fix to independently re-verify. This retro's own
verification burden is instead the live re-check DoD (a)/(b)/(c) below ask
for, which is where the actual work in this retro's task lives.

**Sprint outcome, stated before the findings that qualify it:** T20 shipped
zero tickets, zero points, one PR (#218, the Ceremony 1+2 plan document
itself), merged `19:46:10Z`. No PR has merged since. The plan's own claim —
that all 8 open issues remain genuinely blocked and no new #212/#213-shaped
gap exists — is re-verified live in this retro, not re-read from the plan's
prose.

**What this retro found, in one sentence, so the findings do not have to be
read before the headline is known:** every one of the plan's own claims
holds unchanged at retro time — all 8 issues are still blocked on exactly
the blockers named, zero of them received a comment or a state change since
the plan merged, the migration-tooling roadmap-debt classification is
untouched, D1 and D2 are both still open with D2 scoring its cleanest null
result yet — and this retro's own fresh read of the backlog and PM's
carried-forward concern does not surface a different conclusion than the
planning ceremony reached, though it reaches that agreement by independently
re-examining the question rather than by deferring to the plan's authority.
No incident-grade finding this sprint; the one substantive finding is an
engaged, non-mechanical answer to PM's own question (finding 4).

---

## 1. The merged-fix issue sweep — clean, trivially reconciled, the correct
result for a sprint with zero PRs to sweep, seventh sprint running

**PO.** Per `sprint-process.md`'s DoD, this retro is sweep moment 1; T21's
Ceremony 1 remains the authoritative moment regardless of this result.

**Step 1 — list the open issues, live, at this retro's start.**
`list_issues(state: OPEN)` → **`totalCount: 8`**: #124, #126, #130, #134,
#144, #145, #149, #164. Identical set to T20's own Ceremony 1 sweep and to
T19's retro sweep before it.

**Step 2 — reconcile arithmetically, before reading any issue
individually.** `open_at_end_of_T20's_own_Ceremony_1 − closed_during_T20 +
opened_during_T20`. T20's own Ceremony 1 (§A1) left the count at **8**.
During execution: **zero PRs merged** (T20 dispatched no tickets to merge —
confirmed directly, below), so `closed_during_T20 = 0` and
`opened_during_T20 = 0`. `8 − 0 + 0 = 8`. **Matches the live `totalCount: 8`
read at this retro's start exactly.**

**Step 3 — cross-reference merged PRs against the open list.**
`list_pull_requests(state: closed, base: claude/go-backend-pickleball-7up34j,
sort: updated, direction: desc)` → the most recent entry is **#218 itself**
(T20's own Ceremony 1+2 plan, `merged_at: 2026-08-15T19:46:10Z`); nothing
merged after it. `git log --oneline -3` at this retro's branch cut shows
`bbc9c90` (PR #218's merge) as the tip with no descendants, and `git status`
was clean. **Zero PRs to cross-reference against the open list** — the
correct, trivial shape for a 0-ticket sprint, checked rather than assumed
from the plan's own "expected to be trivially clean" prediction (§A0 DoD
item 2).

**Sweep result: clean, seventh consecutive sprint** (after T15, T16, T17,
T18, T19, T20's own Ceremony-1 sweep). **T21's Ceremony 1 still re-runs this
sweep in full**, per the standing rule that a prior ceremony's clean result
does not discharge the next one.

## 2. DoD (a) — did the "all 8 issues remain genuinely blocked" claim hold
for the whole sprint? Re-checked live at retro time, issue by issue, not
re-read from the plan's table

**QA and BA.** This is the scoring the sprint's own DoD flagged as "the one
that actually requires work," so it is done issue by issue against freshly
fetched fields, not against T20's plan's own per-issue table.

| Issue | Comments at T20 Ceremony 1 | Comments now (live) | `updated_at` now | Changed since plan merged (`19:46:10Z`)? |
|---|---|---|---|---|
| #144 (D1) | 1 | 1 | `2026-08-15T07:01:03Z` | No — predates the plan's own merge |
| #149 | 3 | 3 | `2026-08-15T16:56:58Z` | No — predates the plan's own merge |
| #145 | 1 | 1 | `2026-08-15T05:01:29Z` | No |
| #164 | 1 | 1 | `2026-08-15T14:16:28Z` | No |
| #124 | 1 | 1 | `2026-08-15T16:25:34Z` | No |
| #126 | 0 | 0 | `2026-08-14T16:12:26Z` | No |
| #130 | 0 | 0 | `2026-08-14T16:30:25Z` | No |
| #134 | 0 | 0 | `2026-08-14T16:37:49Z` | No |

**Every one of the 8 issues' `updated_at` timestamp is earlier than PR
#218's own merge (`19:46:10Z`) — none has been touched since T20's plan was
written, let alone since it merged.** This is the live check the task
specifically asked for, not a re-read of the plan's document: an issue whose
blocker resolved mid-sprint would show a new comment or a state change with
a timestamp after `19:46:10Z`, and none does.

**Named individually, per the task's own instructions, rather than folded
into the table above:**

- **D1 (ADR-0015) / D2 (ADR-0016).** `issue_read(get_comments)` on #144
  returns exactly the same single comment as every prior sprint — T14.3's
  original escalation, `created_at: 2026-08-15T07:01:03Z`, nothing after
  it. `docs/adr/0016-*.md`'s own `## Status` field, read in full this
  retro, is unchanged: **"Escalated — awaiting the user's decision. This
  ADR decides nothing."** Neither has been answered.
- **A real IdP tenant for #164/#145.** No new comment on either issue (both
  still exactly 1 comment each, both predating the plan's merge); nothing
  in this environment provisions an IdP tenant, and nothing in the git
  history since `bbc9c90` touches `internal/platform/auth` or
  `dev/auth/**`.
- **Product Owner response on #126/#130.** Both still carry **zero**
  comments — not "unchanged since the plan," literally never commented on
  by anyone at any point in this project's history. No PO response
  exists to have arrived.
- **Assistive-tech capability for #134.** Still zero comments; no UI
  change landed this sprint (zero tickets) that could have altered its
  blocker in either direction.

**DoD (a), scored: yes, the claim held for the whole sprint.** All 8 issues
remain exactly as blocked as T20's Ceremony 1 found them, independently
re-verified live rather than assumed from silence — and the record shows
*why* it held rather than merely asserting that it did: this sprint shipped
zero commits capable of changing any of these facts, and the API confirms
no external party changed them either.

## 3. DoD (b) — the golang-migrate/goose roadmap-debt classification, still
correctly unticketed

**PE.** T20's Ceremony 1 (§A3) classified the `golang-migrate`/`goose`
migration-tooling swap as settled roadmap debt, not a disclosed gap, citing
four prior ceremonies' (T11–T14) explicit rulings. Re-checked at retro time
rather than trusted:

- `HANDOFF.md`'s Cross-cutting section, read in full again this retro
  (lines 990–1668), still contains exactly the single line T20's plan
  quoted: *"Swap docker initdb.d for **golang-migrate** or **goose** before
  production."* No new paragraph, no new cross-reference, no promotion to
  a numbered concern.
- `grep -rn "golang-migrate\|goose" HANDOFF.md docs/process/*.md` returns
  the same hit set as T20's plan found — the single `HANDOFF.md` line plus
  the T11/T12/T13/T14 sprint-plan dispositions already on record. **No new
  file references either tool.**
- No new ADR, issue, or `HANDOFF.md` edit touching migration tooling exists
  between `bbc9c90` and this retro's branch cut — there is nothing to find,
  because zero commits landed on the shared branch in that window at all
  (§1).

**DoD (b), scored: yes, still correctly unticketed, unchanged.** Nothing
surfaced mid-sprint that would change the read — trivially true in the same
sense as the merged-fix sweep, since a sprint with zero commits cannot
generate a new fact about a `HANDOFF.md` line nobody touched. Stated as a
live finding rather than an assumption because the task asked for the check
to be performed, not skipped on the grounds that it was obviously going to
come out this way.

## 4. PM's carried-forward stalled-backlog concern — engaged directly, not
just restated, and the honest answer is that PdE/PE's position holds for a
reason this retro can name precisely rather than merely repeat

**The whole team, PM and PO leading.** This is the retro's most interesting
question and it is treated as one, not mechanically answered "the check was
performed."

**Restating PM's concern precisely, so it isn't answered as a weaker
version of itself.** PM's position, carried from T20's plan: two
consecutive small sprints (T18: 1 ticket/8 points, T19: 2 tickets/8 points)
followed by a zero-ticket sprint is a *shape* that, viewed from outside
this process's own reasoning, reads the same whether the backlog is
genuinely exhausted of unblocked work or the process has quietly stopped
finding it. PM's ask was that this concern be carried forward explicitly,
and escalated to the user directly, if T21 also lands at zero or
near-zero with the same blockers unchanged.

**This retro's own fresh look, not a restatement of the plan's defense.**
Three questions actually test PM's concern, and each is worth asking
independently rather than assuming the planning ceremony already asked it
correctly:

1. **Is the shrinking-sprint-size trend itself evidence of a stalling
   process, independent of whether each individual sprint's zero is
   locally justified?** Looking at the raw shape — T17: 5 tickets/17
   points, T18: 1/8, T19: 2/8, T20: 0/0 — this is a real, monotonic-ish
   decline in ticket count across four sprints. **That is not nothing,
   and PdE/PE's framing in T20's plan (which measures each sprint's zero
   individually and finds each one locally correct) does not by itself
   address the trend line, only each point on it.** A process that is
   locally correct at every step can still be structurally converging on
   zero — this is a legitimate distinct claim from "was T20 specifically
   justified," and it is the sharper version of PM's concern this retro
   thinks is worth naming rather than letting "each sprint checked out"
   stand in for it.
2. **Is the decline explained by something other than the process
   stalling?** Checked directly rather than asserted: T17's 5 tickets were
   a **backlog cleanup burst** — four of its five tickets (T17.2/.3/.4/.5)
   were mechanical FK-error-translation work split per bounded context
   from one filed issue (#195), a one-time shape of work this project
   is unlikely to have another instance of. T18 and T19's shrinking
   ticket counts are not "less work was found" so much as "the specific
   kind of work this backlog contains — disclosed, unblocked,
   correctness/coverage gaps sized 1-2 tickets — was itself running out":
   T19's own two tickets (#212, #213) were gaps that had sat in prose for
   13-14 sprints, i.e. genuinely close to the bottom of that particular
   barrel. **This is a real explanation, not a hand-wave**: it predicts
   exactly the shape observed (a burst, then diminishing small finds, then
   zero) without needing to assume anything about process health.
3. **Does the 8-issue blocked list itself look like a healthy backlog or a
   stuck one?** Read fresh, not through the plan's framing: of the 8, three
   (#144, #149, #124) share one root blocker (D1) that has had a **single,
   unchanging comment for eight sprints running** — this is the part of
   the picture that most resembles "stuck" rather than "correctly
   sequenced," because D1 is not blocked on anything this team cannot
   answer *for the user* — it is blocked on the user not yet answering it.
   The other five are blocked on things genuinely outside this
   environment's control (a real IdP tenant, Product Owner sign-off,
   assistive-tech hardware) — those are not evidence of a stalled process
   at all, they are evidence of a correctly-identified external
   dependency. **So the honest, disaggregated answer is: 5 of 8 blocked
   issues are unremarkable and expected; 3 of 8 (all sharing D1) are the
   one place where "genuinely blocked" and "the process itself carrying an
   unresolved escalation for eight sprints" start to look similar from the
   outside** — not because the team could resolve them, but because eight
   sprints of an unchanging one-line ADR status is exactly the kind of
   signal that, unlabelled, is indistinguishable from neglect.

**This retro's conclusion, stated plainly rather than split the
difference.** PdE/PE's position — that a 0-ticket sprint is the correct
output of a discipline that only takes genuinely unblocked work — is
**not overridden by this retro's fresh look; it holds**, but for a reason
this retro can now state more precisely than the plan did: the trend is
real and explicable (finding-3 above) rather than mysterious, and the part
of the backlog that most resembles "stuck" is not the small/shrinking
sprints, it is **D1's eight-sprint silence specifically**, which is a
different problem than "the process stopped finding work" — it is "the
process correctly stopped guessing at a question that has now sat with the
user, unescalated beyond a single ADR comment, for eight sprints." **That
is this retro's one genuine finding on this question, and it is sharper
than "carry the concern forward unchanged": the thing actually worth
surfacing to the user is not "sprints are getting smaller" (which has an
ordinary, checkable explanation) but "D1 has been silently waiting eight
sprints with no escalation stronger than the original ADR comment,"** which
is the harder, more concrete version of PM's concern and the one this
retro recommends actually be raised (see Recommendations).

**What this retro did not find.** No evidence in the 8 issues or in
`HANDOFF.md`'s Cross-cutting section that a genuinely unblocked ticket was
overlooked, mis-scoped as blocked, or quietly dropped — the live re-check
in finding 2 accounts for every issue's actual state, not a assumed one. If
this retro had found a ninth issue's worth of guessable scope, it would say
so; it did not.

## 5. DoD (c) — did D1 or D2 get answered mid-sprint? No, and D2's null
result this sprint is the precise "no PR existed" shape, kept distinct from
"a PR existed and needed no fix" per T19 retro's own precedent

**PE.** Checked directly, not assumed from "not expected":

- **D1.** #144 carries exactly one comment (finding 2). Unanswered.
- **D2.** `docs/adr/0016-*.md`'s `## Status` unchanged: "Escalated —
  awaiting the user's decision. This ADR decides nothing." Unanswered.

**D2's null-result shape, named precisely per the task's own instruction
and T19 retro's precedent-setting distinction.** T20 shipped **zero
tickets**, so there was no PR under review this sprint at all — not one PR
that happened to need no reviewer-authored fix (T15 through T19's shape),
but the **structurally stronger** null result of no PR existing in the
first place for the interim rule to have been tested against. Kept
explicitly distinct, as the task instructs: "T20 had no PR to test the
interim rule against" is a different and weaker claim than "T20 had a PR
and it needed no reviewer-authored fix" — the former says nothing about
whether the interim rule would hold under pressure, while the latter (T15
through T19's five instances) is at least some evidence the ordinary
two-role loop keeps working under real dispatch. This retro records T20 as
a **sixth data point of a different kind**, not a sixth instance of the
same claim T15–T19 made.

**DoD (c), scored: no, neither D1 nor D2 was answered mid-sprint** — the
expected result, confirmed rather than assumed, with D2's specific shape
named precisely rather than folded into the prior five.

## No finding on

**No finding on the migration-header-ownership check.** Not applicable —
zero tickets, zero migrations, the same "no opportunity to fire" answer
T20's own plan gave (§A4), re-confirmed rather than silently repeated.

**No finding on the same-wave shared-interface verification rule.** Not
applicable — zero tickets, no wave.

**No finding on the label taxonomy.** No issue was opened or touched this
sprint to check conformance against.

**No finding on PCI conformance.** No `.proto` file or payment-DTO field
changed this sprint — CLAUDE.md rule 11 has nothing to check, the same
honest "nothing to check" T19 retro recorded for a different reason.

**No finding on a new #212/#213-shaped gap.** This retro independently
re-read `HANDOFF.md`'s Cross-cutting section in full (not merely trusted
T20's plan's own re-scan) and found nothing T20's own Ceremony 1 missed —
the same conclusion, independently re-derived rather than copied.

---

## The sprint goal, scored: confirm-and-report was the actual deliverable,
and it holds up under this retro's own independent re-check

> *"Confirm, rather than assume, that the tracked backlog remains genuinely
> blocked and that no new disclosed-but-unfiled gap has surfaced since
> T19 — and take no ticket this sprint rather than manufacture one."*

**Every clause holds, re-verified independently rather than taken from the
plan's own account.** All 8 issues are confirmed still blocked, issue by
issue, with each one's `updated_at` shown to predate the plan's own merge
(finding 2). The migration-tooling classification is confirmed unchanged,
checked against a fresh grep and a fresh read of the Cross-cutting section,
not assumed (finding 3). No ticket was manufactured. D1 and D2 remain
unanswered, with D2's null result named at the precise strength this
sprint's shape actually supports (finding 5).

**What this retro adds beyond re-confirming the plan's own claims**: a
direct, non-mechanical engagement with PM's stalled-backlog concern
(finding 4) that reaches the same operational conclusion the plan did —
proceed, don't manufacture scope — but for a reason the plan itself did not
state: the shrinking-sprint-size trend has an ordinary explanation (a
one-time cleanup burst followed by a genuinely finite pool of small,
disclosed gaps running low), and the part of the backlog that actually
resembles a stalled process is not sprint size at all, it is **D1's eight
sprints of silence with no escalation beyond one ADR comment** — a sharper,
more actionable form of PM's concern than "sprints are shrinking."

**The agreed honest sentence, which T21's Ceremony 1 should carry into
`HANDOFF.md`'s Task-backlog entry** (`sprint-process.md` Ceremony 1 item 3
requires the retro's form, not a stronger one):

> T20 shipped zero tickets, the first 0-ticket sprint in this project's
> history, and this retro independently re-verified rather than trusted
> that the reason was real: all 8 tracked issues' blockers were re-checked
> live, issue by issue, and every one's `updated_at` timestamp predates the
> sprint plan's own merge — none moved. The `golang-migrate`/`goose`
> migration-tooling classification (settled roadmap debt per four prior
> ceremonies, T11–T14) is unchanged, re-checked against a fresh grep and a
> fresh full read of `HANDOFF.md`'s Cross-cutting section rather than
> assumed. Neither D1 nor D2 was answered; D2 recorded its sixth sprint
> with nothing to score, but of a structurally different and weaker kind
> than T15–T19's five instances — no PR existed this sprint to test the
> interim rule against, not a PR that needed no fix. On PM's carried-forward
> stalled-backlog concern, this retro independently re-examined rather than
> restated the plan's defense and reaches the same operational conclusion
> (proceed, don't manufacture scope) for a sharper reason: the
> shrinking-sprint-size trend (T17: 5 tickets, T18: 1, T19: 2, T20: 0) has
> an ordinary explanation — a one-time FK-translation cleanup burst at T17
> followed by a genuinely finite pool of small disclosed-but-unfiled gaps
> running low by T19 — and the part of the backlog that actually resembles
> a stalled process is not sprint size, it is D1 sitting with a single
> unchanging ADR comment for eight sprints running with no escalation
> beyond that comment. This retro recommends T21 raise D1's silence to the
> user directly, on its own terms, rather than folding it into a general
> "sprints are getting smaller" framing that has an ordinary explanation
> and would misdirect the actual signal.

---

## Recommendations for T21's Ceremony 1 and 2

1. **Continue treating the merged-fix sweep as authoritative regardless of
   this retro's clean result** — T21's Ceremony 1 re-runs the sweep and
   re-verifies the open count from the API rather than trusting this
   retro's table (finding 1).
2. **Carry PM's stalled-backlog concern forward, but in the sharper form
   this retro derived rather than the form it arrived in.** Per PM's own
   recorded ask, T20 landing at zero triggers exactly the carry-forward PM
   requested — but the concern worth escalating to the user is not "three
   shrinking sprints in a row," which finding 4 shows has an ordinary
   explanation, it is **D1 specifically**: eight sprints with the ADR's own
   `## Status` field and #144's comment count both frozen since T14.3, no
   second escalation attempt, no reminder, nothing beyond the original
   ADR text. If T21 also lands at zero or near-zero, raise D1's silence by
   name, not the sprint-size trend by itself.
3. **T21's Ceremony 1 should explicitly re-examine whether D1's eight-sprint
   silence changes the cost-benefit of escalating harder** (e.g. a direct,
   short message to the user rather than another ADR comment nobody reads
   until the next ceremony) — this retro does not do that escalation
   itself (out of scope for a retro, and D1 stays unimplemented/unguessed
   per the standing restriction), but names it as the concrete next step
   finding 4 points to, rather than leaving "carry the concern forward" as
   an abstract instruction with no proposed action.
4. **D1 and D2 stay with the user; no T21 ticket implements
   `CancelBooking`/`CreateBooking` authorization or a reviewer-authorship
   carve-out, and neither is guessed at.** If either answer arrives
   mid-sprint, each escalation's own trigger takes over.

## Sprint-level Definition of Done — scored against what T20's own plan asked

Per `docs/process/t20-sprint-plan.md`'s "Sprint-level Definition of Done,"
three scorings were owed at this retro, stated there so they would not be
improvised — restated here with their answers:

- **(a) Did the "all 8 issues remain genuinely blocked" claim hold for the
  whole sprint — a live re-check at retro time, not a re-read of the
  plan's document?** **Yes** — every issue's `updated_at` predates the
  plan's own merge, checked individually, not assumed — finding 2.
- **(b) Is the `golang-migrate`/`goose` roadmap-debt classification still
  correctly unticketed at retro time, or did anything surface mid-sprint
  that would change that read?** **Yes, still correctly unticketed** —
  re-checked against a fresh grep and a fresh full read of the
  Cross-cutting section, nothing new found — finding 3.
- **(c) Did D1 or D2 get answered mid-sprint?** **No, neither** — finding
  5, with D2's specific null-result shape ("no PR existed" vs. "a PR
  existed and needed no fix") kept distinct per T19 retro's own
  precedent-setting distinction.

**Not scoreable by T20 and deliberately not pre-empted:** D1 and D2 remain
the user's (finding 5).

Retro complete. Issue-tracker actions this ceremony: none — zero PRs merged
this sprint, so there was nothing to close and nothing to file. Open count
at ceremony start: **8**. Open count now: **8** (unchanged).

Per `sprint-process.md`'s established convention (a retro PR never updates
the Docs-index row that points at it, since that row must cite this PR's own
merge number, which does not exist until it merges): **`HANDOFF.md`'s T20
row is not touched by this PR.** T21's Ceremony 1 corrects it, including the
honest-form sentence above, as its first job — the same standing convention
every prior ceremony has followed.
