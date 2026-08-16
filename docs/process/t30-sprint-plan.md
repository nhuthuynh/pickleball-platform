# T30 Sprint Plan — Ceremonies 1 + 2

Backlog refinement (Ceremony 1) and sprint planning (Ceremony 2) per
`docs/process/sprint-process.md`, six-role team (briefs:
`docs/agent-operating-handbook.md` Part B). Held against
`docs/process/t29-retro.md` (PR #241, merged `2026-08-16T15:43:19Z`),
`HANDOFF.md` including its Docs index, `CLAUDE.md`, and the live PR/issue
state of `nhuthuynh/white-label` (GitHub-side name `pickleball-platform`), at
branch tip `5756abe` (T29's own retro merge), re-fetched and confirmed
matching `origin/claude/go-backend-pickleball-7up34j` before this ceremony's
branch was cut.

**Every factual claim below was re-derived against the live repository and
GitHub API this ceremony, not read from T29's plan's or retro's own
account** (CLAUDE.md rule 10; this project's standing convention, most
recently the reason T29's own retro caught a real drafting gap in T29's own
plan — see §A1 below). This ceremony also carries two pieces of real,
actionable process work forward from T29's retro (recommendations 2 and 3)
rather than treating a 0-ticket sprint as reporting-only — see §A9.

## §A0 — Correcting T29's Docs-index row and Task-backlog narrative

Per `sprint-process.md`'s standing convention (a retro PR never updates the
Docs-index row that points at it, since that row must cite the retro's own
merge PR number, which does not exist until it merges) — **performed
directly in `HANDOFF.md` by this same PR**:

| Field | What the row said | What it now says | Why |
|---|---|---|---|
| T29 Reviews cell | `… → PR #240 (T29.2, …) → PR (retro doc), in that merge order (… "14:28:18Z" → "15:04:23Z" → "15:23:09Z")` | `… → PR #241 (retro doc), in that merge order (… "14:28:18Z" → "15:04:23Z" → "15:23:09Z" → "15:43:19Z")` | T29's own retro (`docs/process/t29-retro.md`, closing paragraph) claimed its own PR would correct this row — but a PR structurally cannot cite its own merge-PR number before it exists, the exact reason this convention exists. Filled in now that the number is knowable: `pull_request_read(get, #241)` → `merged_at: 2026-08-16T15:43:19Z` |

**A new T30 row is added** in the same before-the-sprint form, Retro and
Reviews honestly marked "not yet written" / "not yet opened", for T31's
Ceremony 1 to correct in turn.

**ADR-status citation form** (T24 retro's convention, extended by T28
retro's finding 6 to cover ADR-0017 — re-verified unchanged this ceremony,
`grep -n "^## Status" -A3` on all three files):

- ADR-0015 `## Status`: **"Escalated — awaiting product decision. This ADR
  decides nothing."**
- ADR-0016 `## Status`: **"Escalated — awaiting the user's decision. This
  ADR decides nothing."**
- ADR-0017 has **no separate `## Status` section** (Accepted, not
  escalated). Cited from its frontmatter bullet only: **"Status:
  Accepted."** `git log --oneline -- docs/adr/0017-*.md` still shows
  exactly one commit (its T28.1 authoring commit) — untouched by T29.

## §A1 — The merged-fix issue sweep

**Step 1 — list the open issues, live.** `list_issues(state: OPEN)` at
ceremony start → **`totalCount: 7`**: #124, #126, #130, #134, #144, #145,
#149. Identical set to T29 retro's own sweep.

**Step 2 — reconcile arithmetically, before reading any issue
individually.** `open_at_end_of_T29's_own_retro (7) − closed_since (0) +
opened_since (0) = 7`. `list_pull_requests(state: closed, base:
claude/go-backend-pickleball-7up34j, sort: updated, direction: desc)`
confirms the most recent merged PR is **#241 itself** (T29's own retro doc,
`merged_at: 2026-08-16T15:43:19Z`); nothing merged since. `git log
--oneline -5` at this ceremony's branch cut shows `5756abe` (PR #241's
squash merge) as the tip with no descendants, matching
`origin/claude/go-backend-pickleball-7up34j` exactly after `git fetch`, and
`git status` was clean before this ceremony's branch was cut. **Matches the
live `totalCount: 7` exactly.**

**Step 3 — cross-reference merged PRs against the open list.** Zero PRs
merged since T29's retro (confirmed above), so there is nothing to
cross-reference — the correct, trivial shape for a sprint whose predecessor
already closed everything it had open (#164, #237 both closed by PR #240,
per T29 retro §1).

**Sweep result: clean.** **T31's Ceremony 1 still re-runs this sweep in
full**, per the standing rule that a prior ceremony's clean result does not
discharge the next one.

**Note on this document's own DoD issue count, per T29 retro's
recommendation 5.** The Sprint-level DoD below cites **7** — this same
number, drawn directly from this section's own sweep arithmetic, not from
a separately-maintained ranking table. §A5's ranking table below lists the
identical 7 issues for the same reason: it is generated from this section's
own set, not compiled independently. This is the discipline T29's own plan
did not apply (its §A5 ranking table silently dropped #124, an issue its
own §A3 table two sections earlier correctly carried — T29 retro §5)
undercounting its DoD line by one; applying it here means the two tables
cannot silently diverge, because one is not re-derived independently of the
other.

## §A2 — T29 retro's six recommendations, dispositioned

| # | Recommendation | Disposition |
|---|---|---|
| 1 | Re-run the merged-fix sweep as the authoritative moment, re-verifying the open count (7) from the live API rather than trusting the retro's table | **Executed** (§A1) — re-derived independently from `list_issues`, not read off the retro's own table |
| 2 | Add a `docs/LESSONS.md` entry for the shared-checkout institutionalization gap, paired with a concrete fix: give dispatch isolation its own named `sprint-process.md` section | **Half already executed, half executed here.** The `docs/LESSONS.md` entry was already written by T29's own retro (`## T29 (2026-08-16)` — a retro is free to perform this kind of live documentation, per this project's standing practice; it is not a code change and does not implicate CLAUDE.md rule 9). This ceremony supplies the durable fix that entry called for and did not itself land: a new **"Dispatch isolation"** section in `docs/process/sprint-process.md`, under Ceremony 2 — §A9 |
| 3 | Adopt the cheap PR-body-verification safeguard for T30 onward: an implementer confirms its own just-opened PR's `body` is non-empty via a fresh `pull_request_read(get)` before reporting a ticket done | **Executed** — a new **"PR-body self-verification"** section added to `docs/process/sprint-process.md`'s Execution section — §A9 |
| 4 | Continue the post-T29 backlog-composition counter, incrementing to two if T30 Ceremony 1's own live check again finds the same 7-issue set unchanged | **Executed** (§A1/§A3) — this ceremony's own live sweep finds the identical 7-issue set (#124, #126, #130, #134, #144, #145, #149) unchanged. **Counter increments to two** |
| 5 | Derive a Sprint-level DoD's issue count from the ceremony's own sweep tables, not a separately-maintained ranking table | **Executed** (§A1's closing note, §A5) — this plan's DoD line and §A5's ranking table are both generated from §A1's sweep set, not independently compiled |
| 6 | If D2's fourth shape (T29 retro §7) recurs, score it against that retro's definition rather than re-deriving one from scratch | **Not applicable this sprint** — zero tickets means zero PRs to review, so D2's interim rule is not exercised at all this sprint (§A7) |

## §A3 — Routine re-verification (all 7 issues, live)

Per the task's own instruction: every open issue re-read via `issue_read`,
not trusted from T29 retro's table without re-checking `updated_at`/comment
counts directly.

| Issue | `updated_at`/comments at T29 retro | `updated_at`/comments now (live, this ceremony) | Changed? |
|---|---|---|---|
| #149 | `2026-08-16T14:20:31Z`, 4 comments | `2026-08-16T14:20:31Z`, 4 comments | No |
| #145 | `2026-08-15T05:01:29Z`, 1 comment | `2026-08-15T05:01:29Z`, 1 comment | No |
| #144 (D1) | `2026-08-15T07:01:03Z`, 1 comment | `2026-08-15T07:01:03Z`, 1 comment | No |
| #134 | `2026-08-14T16:37:49Z`, 0 comments | `2026-08-14T16:37:49Z`, 0 comments | No |
| #130 | `2026-08-14T16:30:25Z`, 0 comments | `2026-08-14T16:30:25Z`, 0 comments | No |
| #126 | `2026-08-14T16:12:26Z`, 0 comments | `2026-08-14T16:12:26Z`, 0 comments | No |
| #124 | `2026-08-15T16:25:34Z`, 1 comment | `2026-08-15T16:25:34Z`, 1 comment | No |

**Every field matches T29 retro's own live-fetched table exactly,
byte-for-byte — zero blockers changed state, zero new comments on any of
the 7.**

**Every issue's full body also re-read this ceremony (`issue_read(get)`),
not just its `updated_at`/comment count**, per the task's explicit
instruction not to trust the retro's table alone:

- **#144 (D1):** still zero authorization on `CancelBooking`/
  `CreateBooking`, blocked on a product decision (who "owns" a public
  quote-and-book Booking, and whether that flow should stay
  unauthenticated) — ADR-0015, unanswered.
- **#149:** still names `booking_host_id`/`game_host_id`/
  `entrant_player_id`/admin-assignment lists as Payments' remaining
  caller-supplied ownership facts, blocked on D1 (Booking's half) plus the
  yet-unbuilt Game-Admin/Competition-Admin durable store the issue's own
  text names as a sub-problem — both unchanged.
- **#145:** still needs a real IdP tenant's non-uuid `sub` claim, which
  exists nowhere in this environment; structurally distinct from the
  now-fully-closed #164/#237 backfill (that distinction, drawn at T28's
  Ceremony 1, re-verified again this ceremony against #145's own text).
- **#124:** still needs Product Owner input on cascade semantics
  (court-release, refund automaticity via T12.3's now-existing
  `RefundPayment`, waitlist handling) before `Game.Cancel()` can cascade.
- **#126:** still needs Product Owner input on whether the price is
  per-Game, per-Registration, or per-head-including-guests, before any
  code — unchanged since T8.10.
- **#130:** still carries its own stated open product question — whether
  reversing a `no_show_fee` is the product behavior actually wanted, and
  what (if anything) the refund projects onto `Registration.PaymentStatus`
  — not implementable from symmetry alone without guessing at that answer.
- **#134:** still needs a real screen-reader session (NVDA/JAWS/VoiceOver)
  and rendered-in-browser measurements this environment's lack of
  assistive-technology hardware cannot supply; T30 ships no UI change that
  would give a targeted pass anything new to cover.

**Migration-tooling classification (`golang-migrate`/`goose`), re-checked:**
`HANDOFF.md`'s Cross-cutting section still carries exactly the line every
prior ceremony has quoted (`grep -n "golang-migrate\|goose" HANDOFF.md`,
unchanged line count and content). `ls docs/adr/` ends at `0017`; `ls
db/migrations/` ends at `0026`. **Still correctly unticketed, unchanged —
thirteen separate prior ceremonies (T11–T14, T21–T29) have already ruled or
re-confirmed this settled roadmap debt (T28's and T29's own plans re-check
the identical line even though both took real tickets elsewhere — verified
by `grep -l "golang-migrate" docs/process/t28-sprint-plan.md
docs/process/t29-sprint-plan.md`), and zero PRs merged this sprint to
supply a new fact that could overturn it.**

**`HANDOFF.md`'s Cross-cutting section re-scanned in full for anything
newly actionable** (the same check T18 through T27 ran, paused during
T28/T29's real tickets, resumed here): read end to end
(`HANDOFF.md:1631-2295`, the Cross-cutting section in the current tree,
after this same PR's own T29-row/T30-row edits shifted every line number
below them — re-located live via `grep -n "^## Cross-cutting\|^## Definition
of Done"` rather than cited from a stale offset), plus a targeted grep for
the phrases prior disclosed-but-unticketed gaps have carried (`raise at the
next backlog refinement`, `follow-up ticket`, `not yet ticketed`, `deferred
with no ticket`, `logged not fixed`, `still open, still not ticketed`).
**No new candidate found.** Every hit resolves to one of: already closed
(the write-handler malformed-ID guard → #97/T10.7, confirmed by the
section's own correction note around `HANDOFF.md:2213`; the `RefundPayment` wiring
gap → T12.3; the Competitions `PayableType` gap → filed and then closed;
T8.10's placeholder price → tracked as the still-open #126, already in
§A1's swept set), or a decision this project has already made and is not
reopening (ADR-0009's OAuth/messaging deferral, ADR-0010/0012's
auto-matching sequencing). T19's Ceremony 1 remains the only ceremony that
ever found a genuine new gap here; this is the eleventh re-scan (T18–T27,
paused T28–T29, resumed T30) to confirm no second instance exists.

**Toolchain, re-run directly against the unmodified tree before any
planning text was written:**

```
$ make test-domain
go test ./internal/.../domain/... ./internal/.../app/... -race -count=1
ok      github.com/nhuthuynh/white-label/internal/booking/domain       1.026s
ok      github.com/nhuthuynh/white-label/internal/competitions/domain  1.023s
ok      github.com/nhuthuynh/white-label/internal/facilities/domain    1.016s
ok      github.com/nhuthuynh/white-label/internal/identity/domain      1.015s
ok      github.com/nhuthuynh/white-label/internal/payments/domain      1.026s
ok      github.com/nhuthuynh/white-label/internal/socialplay/domain    1.018s
ok      github.com/nhuthuynh/white-label/internal/booking/app          1.030s
ok      github.com/nhuthuynh/white-label/internal/competitions/app     1.024s
ok      github.com/nhuthuynh/white-label/internal/facilities/app       1.020s
ok      github.com/nhuthuynh/white-label/internal/identity/app         1.015s
ok      github.com/nhuthuynh/white-label/internal/payments/app         1.020s
ok      github.com/nhuthuynh/white-label/internal/socialplay/app       1.024s
```

12/12 packages green — matching T29 retro's own count exactly on an
unmodified tree (no code change has landed since).

## §A4 — Migration-header-ownership check: not applicable this sprint

No T30 ticket exists (§A6), so no ticket names a database table or a
migration file.

## §A5 — The whole open backlog, ranked, with a disposition for each

All 7 open GitHub issues, generated directly from §A1's swept set (per T29
retro recommendation 5 — not a separately-compiled ranking):

| Issue | Ranked | Disposition |
|---|---|---|
| **#144** `CancelBooking`/`CreateBooking` have no authz | **Escalated** | D1 unanswered; §A7. Seventeenth sprint carrying this |
| **#149** Payments' remaining caller-supplied facts | **Untouched, correctly** | Blocked on D1 (the Booking-side fact) plus an unbuilt Game-Admin/Competition-Admin store (the issue's own named sub-problem); unchanged |
| **#124** Game-cancellation cascade | **Untouched, correctly** | Needs Product Owner input on cascade semantics; unchanged |
| **#145** pre-existing UUID rows vs. `Principal.Subject` | **Deferred** | Needs a real, non-uuid IdP `sub` claim this environment cannot produce; unchanged |
| **#126** real per-Game price field | **Deferred** | Needs Product Owner input on price shape; unchanged since T8.10 |
| **#130** refunding a `no_show_fee` Payment | **Deferred** | Carries its own stated open product question; unchanged |
| **#134** WCAG manual screen-reader pass | **Deferred** | Needs real assistive-technology hardware this environment cannot provide; T30 ships no UI change |

**Every one of the 7 tracked issues remains genuinely blocked, independently
re-verified rather than assumed (§A3).** None of the seven is actionable
this sprint without guessing at D1, a real IdP tenant, a Product Owner
decision this team cannot make unilaterally, or assistive-technology
hardware this environment does not have.

**Considered and rejected: manufacturing partial progress under a stated
assumption, per the task's own instruction to check first.** For each of
the three PO-blocked issues (#124, #126, #130), the blocking question is
not "what should the code look like" but "which of several materially
different product behaviors is wanted" — #124's court-release/refund/
waitlist semantics, #126's per-Game-vs-per-Registration-vs-per-head pricing
unit, #130's whether a no-show-fee reversal is wanted at all. Shipping code
against a guessed answer to any of these is not a smaller version of the
same ticket, it is a different ticket that may need to be reverted the
moment the real answer arrives — the same reasoning this project has
applied to D1/D2 throughout (`sprint-process.md`'s standing rule against
guessing at an escalated decision), extended here to product questions that
are escalated in substance even though no ADR names them. No defensible
assumption exists for any of the three that would not risk exactly that.

## §A6 — Why zero tickets, and why that is the honest answer

**The alternatives this project has repeatedly named and rejected are still
rejected, for the same reasons:**

1. **Manufacture a ticket against one of the 7 blocked issues by guessing
   at its blocker.** Not done — every blocker was independently
   re-verified live this ceremony (§A3/§A5), and none has moved. Per the
   task's own explicit instruction, no T30 ticket implements
   `CancelBooking`/`CreateBooking` authorization or a reviewer-authorship
   carve-out — D1 and D2 both remain formally open, unresolved, and this
   ceremony does not guess at either.
2. **Take the migration-tooling item as new scope.** Not done — settled
   roadmap debt, re-confirmed unchanged (§A3), with no new fact this sprint
   to overturn thirteen prior ceremonies' own ruling.

**The third alternative — the one this ceremony takes — is a 0-ticket
sprint, the ninth in this project's history and the first since T28 broke
the T20–T27 run.** This is not a reversion to "nothing changed" bookkeeping
padding: T28 and T29 shipped two real sprints (8 points, then 34 points)
closing #164/#237 in full, and this ceremony independently confirmed that
work is done and durable (§A1's clean sweep) before concluding the
*remaining* backlog is exactly as blocked as it was before either of those
sprints started. The honest reading is that #164/#237 were the genuinely
actionable items this backlog held, they are now closed, and what remains
is — as it was at T20–T27 — blocked on decisions and resources outside
this team's authority to supply.

**Dependency-completeness check: not applicable.** With no ticket, there is
no producer/consumer pair to check either question of
(`sprint-process.md`'s two-question form).

## §A7 — DECISION D1 and DECISION D2 remain unanswered

Re-verified this ceremony (§A3): #144 carries exactly one comment, T14.3's
original escalation, unchanged. Both ADR-0015's and ADR-0016's own `##
Status` headings are unchanged (§A0).

> **DECISION D1 (for the user / Product Owner) — seventeenth deferral.**
> When somebody books a court through the public flow *without an
> account*, who should be allowed to cancel that booking later, and should
> booking without an account remain possible at all?
> `docs/adr/0015-booking-ownership-for-public-bookings.md` lays out four
> options with their costs and recommends none. No T30 ticket exists to
> implement this, and per ADR-0015's restriction list no T30 PR may guess
> at it regardless. D1 has now carried its single T14.3 comment for
> **seventeen consecutive sprints (T14 through T30)** with no second
> escalation attempt beyond the original ADR text.

> **DECISION D2 (for the user) — no PR to exercise the interim rule
> against this sprint.** May a session that reviews and merges a PR also
> author code on it? `docs/adr/0016-reviewer-authored-code-on-a-reviewed-
> pull-request.md` lays out four options and recommends none. T30 ships no
> tickets and therefore no PRs, so this sprint lands in the structurally
> weaker "no PR existed" shape (per T21 retro's naming), not the "PR
> existed, no fix needed" shape or T29's own genuinely-new fourth shape —
> named precisely rather than folded into either count.

**Neither is implemented, decided, or guessed at by this ceremony.** Both
remain exactly as escalated as `sprint-process.md`'s own restriction lists
require. Per the task's own standing instruction, the sprint loop continues
regardless — re-deferring both as needed, not escalating further absent a
materially different blocker profile or the backlog running dry, and
neither condition fired this ceremony (§A1, §A5).

## §A8 — Shared-file pre-assignment, and same-wave verification: not
applicable this sprint

| Artifact | Owner | Notes |
|---|---|---|
| `HANDOFF.md` | this ceremony only | No execution ticket exists this sprint to collide with it |
| `docs/process/sprint-process.md` | this ceremony only | Two new named sections landed directly by this ceremony (§A9) — no implementer ticket touches it |

**Same-wave shared-interface verification rule: does not apply, trivially.**
T30 has zero tickets. There is no wave to have a collision within.

---

# §A9 — Process-institutionalization work (T29 retro recommendations 2 and 3)

This section is real work product for this ceremony, not backlog triage —
per the task's own instruction, and per this project's own standing rule
that a 0-ticket sprint is honest about scope, not idle.

## Recommendation 2 — Dispatch isolation

**What was already done, verified rather than assumed.** T29's own retro
(`docs/process/t29-retro.md` §8) scored the T29.1/T29.2 shared-checkout
near-miss and, in the same breath, wrote a `docs/LESSONS.md` entry (`## T29
(2026-08-16) — the T9 dispatch-isolation remedy was never durably written
down…`) documenting the underlying institutionalization gap: T9's own
retro (`docs/process/t9-retro.md` finding 1) adopted "dispatch isolation
becomes an explicit Ceremony 2 checklist item" as its remedy, and a direct
`grep -ni "isolat" docs/process/sprint-process.md` — re-run this ceremony,
confirmed still zero matches **before** this ceremony's own edit — showed
that remedy had never actually been written into this project's durable
process document. Re-read `docs/LESSONS.md`'s entry in full this ceremony:
it is substantive, correctly scoped (a near-miss, not a T9-grade incident;
the institutionalization gap, not the collision itself, is what earns the
entry), and needed no correction or restatement.

**What this ceremony did.** Reviewed the retro's own reasoning (§8's full
argument, not just its recommendation line) rather than accepting the
recommendation at face value, and agrees with it: a remedy that survives
only as a habit some sprint plans repeat in their own prose (T13 did; T29
did not) is exactly as durable as the next ceremony's memory of it, and
this project has now measured that duration directly at thirteen sprints.
Added a new **"Dispatch isolation"** section to `docs/process/
sprint-process.md`, placed under **Ceremony 2** (mirroring T9's own framing
of the remedy as "an explicit Ceremony 2 checklist item," and structurally
parallel to how "the dependency-completeness check" and "the same-wave
shared-interface verification rule" each earned a named section from a
similarly-scoped prior finding). The section states the rule (every
implementer in a multi-implementer wave works in its own isolated
`git worktree` or equivalent, never sharing a local checkout's working
tree with a concurrently-dispatched ticket), states what Ceremony 2 must
do concretely (name the isolation mechanism in the wave's own text; treat
isolation and the same-wave shared-interface rule as independent checks
that both apply when both conditions hold), and states what it does not
require (a solo-implementer or genuinely-sequential wave has no hazard to
isolate against). Full text: `docs/process/sprint-process.md`, "###
Dispatch isolation" (new section, this PR).

**Why a `docs/LESSONS.md` addendum was not also written here.** The entry
already exists, is accurate, and this ceremony found nothing in it that
needed correcting — writing a second entry for the same incident would be
the "restate the same instruction a third time" anti-pattern
`sprint-process.md`'s own "Scheduled removals" section warns against for a
different case. The fix this ceremony owed was the durable section itself,
not a second retrospective note about needing one.

## Recommendation 3 — PR-body self-verification

Added a new **"PR-body self-verification"** section to `docs/process/
sprint-process.md`'s Execution section, placed immediately after the
per-ticket DoD numbered list and before "Same-wave shared-interface
verification." Considered both candidate homes the task named
(`sprint-process.md` and `docs/agent-operating-handbook.md`) — read both in
full this ceremony before deciding. `docs/agent-operating-handbook.md` is
Part A (shared foundations: bounded contexts, ubiquitous language,
architectural rules) plus Part B (six role briefs, each a mandate/
adversarial-toward/checks/escalates quartet); it carries no per-ticket
procedural mechanics anywhere, and grepping it for `PR body`/
`pull_request_read`/`empty` returns nothing. `sprint-process.md`'s
Execution section is where every other per-ticket mechanical safeguard of
this shape already lives (the "closes #N" review-time discipline, the
same-wave shared-interface verification rule, the interrupted-session
recovery practice) — so this is the section's natural continuation, not a
new category of document. The new subsection states the incident it
answers (PR #240's own first review finding an essentially-empty body
despite the implementer's own final chat report describing substantial
content), states plainly that this was already caught cleanly by existing
review process (T29 retro §9's own scoring, not restated as a new gap
here), and states the rule: one fresh `pull_request_read(get)` against the
implementer's own just-opened PR, confirming `body` is non-empty and
matches intent, before reporting a ticket done.

---

# Ceremony 2 — Sprint planning

## Sprint goal

> **Confirm, rather than assume, that the backlog #164/#237's closure left
> behind is exactly as blocked as it was before T28/T29 took real scope —
> and land the two pieces of real process work T29's retro identified as
> genuinely owed, rather than treat a 0-ticket sprint as a pure
> confirm-and-report exercise.** All 7 open issues were independently
> re-verified live and found unchanged: two on DECISION D1, one on a real
> IdP tenant this environment cannot provision, three on genuine Product
> Owner input this team cannot supply unilaterally, one on real
> assistive-technology hardware. `HANDOFF.md`'s Cross-cutting section was
> re-scanned in full and produced no new candidate. The
> `golang-migrate`/`goose` roadmap-debt classification remains settled.
> Dispatch isolation now has its own named `sprint-process.md` section, and
> PR-body self-verification is now a standing Execution-section practice.

**What this sprint does not claim:**

- **This is not evidence the project has run out of real work.** A ninth
  0-ticket sprint, the first since T28 broke the T20–T27 run, is the
  honest output of a discipline that takes only what is genuinely
  unblocked and files what is genuinely disclosed — not a target this team
  engineered, and not in tension with T28/T29 having just shipped 42 real
  points of work two sprints running (8 + 34).
- **This does not mean the 7 tracked issues have become less real or less
  worth doing eventually.** Every one is exactly as real and exactly as
  blocked as it was at T29's retro.
- **D1 and D2 remain formally open and unanswered as ADR decisions.**
  Nothing this ceremony did or found bears on either decision itself. No
  ticket implements `CancelBooking`/`CreateBooking` authorization or a
  reviewer-authorship carve-out.
- **The two `sprint-process.md` additions are process fixes, not scope.**
  Landing them in this PR does not make this a non-zero-ticket sprint in
  the sense `sprint-process.md`'s Tickets/Waves sections track — there is
  still no implementer ticket, no wave, no PR against application code.

## Tickets — 0 items, 0 points

**None.** §A6 states the reasoning in full: every tracked issue is
genuinely blocked (§A5), and the one Cross-cutting candidate that has
looked promising on past reads (migration tooling) remains settled roadmap
debt with no new fact to overturn that classification.

## Waves

**None.** No ticket exists to wave-assign.

## Recorded disagreements (Ceremony 2 rule 3 — not smoothed over)

**None recorded this sprint.** No party found grounds to disagree with
continuing the re-deferral pattern on D1/D2, or with the zero-ticket
conclusion — every blocker was independently re-verified live rather than
assumed (§A3), and none has moved.

## Sprint-level Definition of Done

1. **No ticket to merge; sprint goal met as stated** — confirm-and-report
   on the backlog, plus land the two named process fixes — not
   build-and-ship.
2. **The merged-fix issue sweep run and reported with its count (7,
   reconciled arithmetically — §A1)** — by the retro (reporting, not
   blocking) and again by T31's Ceremony 1 (authoritative).
3. **Scoring owed at the retro:**
   - **(a)** Did this ceremony's own claim — that all 7 open issues remain
     genuinely blocked, re-verified live — hold for the whole sprint (a
     live re-check at retro time, not a re-read of this document)?
   - **(b)** Is the `golang-migrate`/`goose` roadmap-debt classification
     still correctly unticketed at retro time?
   - **(c)** Did D1 or D2 get answered mid-sprint, as a formal ADR
     decision?
   - **(d)** Do the two new `sprint-process.md` sections (Dispatch
     isolation, PR-body self-verification) read as this plan describes
     them, independently re-read rather than assumed from this document's
     own account?
4. **Not scoreable by T30 and deliberately not pre-empted:** D1 and D2
   remain the user's own decisions to make, on whatever timeline the user
   chooses.
5. Retro in `docs/process/t30-retro.md`, indexed by a `## T30 sprint retro`
   stub in `docs/LESSONS.md`. `HANDOFF.md`/`CLAUDE.md` state updated —
   noting that **T31's Ceremony 1**, not the retro, corrects T30's
   Docs-index row (the ordinary convention).
