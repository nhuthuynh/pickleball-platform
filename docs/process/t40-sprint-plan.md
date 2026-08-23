# T40 Sprint Plan — Ceremonies 1 + 2

Backlog refinement (Ceremony 1) and sprint planning (Ceremony 2) per
`docs/process/sprint-process.md`, six-role team (briefs:
`docs/agent-operating-handbook.md` Part B). Held against
`docs/process/t39-retro.md` (PR #261, merged `2026-08-23T05:34:51Z`),
`HANDOFF.md` including its Docs index, `CLAUDE.md`, and the live PR/issue
state of `nhuthuynh/white-label` (GitHub-side name `pickleball-platform`), at
branch tip `0e15bda` (T39's own retro merge), re-fetched via `git log`/`git
status` and confirmed matching `origin/claude/go-backend-pickleball-7up34j`
exactly (clean working tree) before this ceremony's branch was cut.

**Every factual claim below was re-derived against the live repository and
GitHub API this ceremony, not read from T39's retro's own account**
(CLAUDE.md rule 10; this project's standing convention). This ceremony's
first act is the correction T39's retro deliberately left undone: its own
PR could not cite its own merge number, so it supplied an agreed sentence
for this ceremony to carry forward instead (§A0).

## §A0 — Correcting T39's Docs-index row and Task-backlog narrative

Per `sprint-process.md`'s standing convention (a retro PR never updates the
Docs-index row that points at it, since that row must cite the retro's own
merge PR number, which does not exist until it merges) — **performed
directly in `HANDOFF.md` by this same PR**:

| Field | What the row said | What it now says | Why |
|---|---|---|---|
| T39 Retro cell | `not yet written` | `docs/process/t39-retro.md` (no incident-grade finding; independently re-verified live down to full bodies that all 7 open issues' blockers held; carried the backlog-composition counter to twenty-one and confirmed D1's silence counter at twenty-six; 7 recommendations for T40) | `docs/process/t39-retro.md` exists and is complete |
| T39 Reviews cell | `not yet opened` | `PR #260 (Ceremony 1/2 doc) → PR #261 (retro doc), in that merge order (merged_at 05:29:12Z → 05:34:51Z)` — both merged, both reviewed via GitHub review comments, see naming convention | `pull_request_read(get, #260)` → `merged_at: 2026-08-23T05:29:12Z`; `pull_request_read(get, #261)` → `merged_at: 2026-08-23T05:34:51Z` (both re-fetched live this ceremony, not assumed from numbering) |

**A new T40 row is added** in the same before-the-sprint form, Retro and
Reviews honestly marked "not yet written" / "not yet opened", for T41's
Ceremony 1 to correct in turn.

**Task-backlog narrative for T39**, corrected per `sprint-process.md`
Ceremony 1 item 3 ("the previous sprint's narrative entry in the Task
backlog, stating its outcome in the form its own retro agreed, not a
stronger one"): T39's own Ceremony 1/2 paragraph (written before the retro
existed, ending "Retro not yet written") is left as the historical record
of what was planned, and this PR appends an **Outcome** paragraph plus
**T39 retro's own agreed honest-form sentence, carried verbatim** — the
exact text `docs/process/t39-retro.md`'s closing section supplies for this
purpose, not a paraphrase or a strengthened version of it. See the "Task
backlog" edit in `HANDOFF.md` for the landed text; T28's through T39's own
rows are the structural precedent for this two-part (prose Outcome +
verbatim blockquote) shape.

**ADR-status citation form** (T24 retro's convention, extended by T28
retro's finding 6 to cover ADR-0017 — re-verified unchanged this ceremony,
`grep -n "^## Status" -A3` on ADR-0015/0016, frontmatter bullet on
ADR-0017):

- ADR-0015 `## Status`: **"Escalated — awaiting product decision. This ADR
  decides nothing."** `git log --oneline -- docs/adr/0015-*.md` still shows
  exactly one commit (`5828550`, its T14.3 authoring commit).
- ADR-0016 `## Status`: **"Escalated — awaiting the user's decision. This
  ADR decides nothing."** `git log --oneline -- docs/adr/0016-*.md` still
  shows exactly one commit (`0aaa912`, its T15.2 authoring commit).
- ADR-0017 has **no separate `## Status` section** (Accepted, not
  escalated). Cited from its frontmatter bullet only: **"Status:
  Accepted."** Untouched since T28.1's authoring commit.

## §A1 — The merged-fix issue sweep

**Step 1 — list the open issues, live.** `list_issues(state: OPEN)` at
ceremony start → **`totalCount: 7`**: #124, #126, #130, #134, #144, #145,
#149. Identical set to T39 retro's own closing sweep.

**Step 2 — reconcile arithmetically, before reading any issue
individually.** `open_at_end_of_T39's_own_retro (7) − closed_since (0) +
opened_since (0) = 7`. `list_pull_requests(state: closed, base:
claude/go-backend-pickleball-7up34j)` confirms the most recent merged PR is
**#261 itself** (T39's own retro doc, `merged_at: 2026-08-23T05:34:51Z`);
nothing merged since. `git log` at this ceremony's branch cut shows
`0e15bda` (PR #261's squash merge) as the tip with no descendants, matching
`origin/claude/go-backend-pickleball-7up34j` exactly, and `git status` was
clean before this ceremony's branch was cut. **Matches the live
`totalCount: 7` exactly.**

**Step 3 — cross-reference merged PRs against the open list.** Zero PRs
merged since T39's retro (confirmed above), so there is nothing to
cross-reference — the correct, trivial shape for a sprint whose predecessor
shipped no application code, only planning/process docs.

**Sweep result: clean — still 7.** **T41's Ceremony 1 still re-runs this
sweep in full**, per the standing rule that a prior ceremony's clean result
does not discharge the next one.

## §A2 — T39 retro's seven recommendations, dispositioned

| # | Recommendation | Disposition |
|---|---|---|
| 1 | Re-run the merged-fix sweep as the authoritative moment, re-verifying the open count (7) from the live API rather than trusting the retro's table | **Executed** (§A1) — re-derived independently from `list_issues`, not read off the retro's own table |
| 2 | Correct `HANDOFF.md`'s T39 Docs-index row and insert the Task-backlog Outcome/blockquote using this retro's agreed sentence, fetching this retro's own PR number live rather than treating it as "already known" | **Executed** (§A0) — `pull_request_read(get, #260)`/`pull_request_read(get, #261)` re-fetched live this ceremony, not assumed from the task prompt's own mention of "#261" |
| 3 | Continue the post-T29 backlog-composition counter, incrementing it to twenty-two if T40 Ceremony 1's own live check again finds the identical 7-issue set unchanged | **Executed** (§A1/§A3) — this ceremony's own live sweep finds the identical 7-issue set (#124, #126, #130, #134, #144, #145, #149) unchanged. **Counter increments to twenty-two** |
| 4 | If a real ticket PR is dispatched in T40, D2's interim rule finally has something to be exercised against again | **Not applicable this sprint** — §A6/§A8 below: zero tickets, zero PRs, so D2 is not exercised at all (§A7) |
| 5 | The soft observation from T30 retro §3a (who audits that a dispatched wave's text actually named its isolation mechanism) remains open and untested; resolve it by example the first time a real multi-implementer wave is dispatched, not pre-emptively | **Not applicable this sprint** — T40 has zero tickets and zero waves, same as T30–T39. Carried forward again, not resolved on no example, per the recommendation's own instruction |
| 6 | A nineteenth 0-ticket sprint arriving with no new fact is not, by itself, grounds to reopen the "is a 0-ticket sprint healthy" question | **Applies** — see §A6's closing note. This *is* the project's nineteenth 0-ticket sprint by total count and the eleventh sprint of the unbroken consecutive run (T30, T31, T32, T33, T34, T35, T36, T37, T38, T39, T40) since T28 broke the T20–T27 streak. The underlying instruction — don't reopen the healthiness question absent a new fact — is followed regardless, and neither of T21 retro's two named reopening conditions fired (checked live, §A6) |
| 7 | The stale repo-metadata artifact does not need its own investigation unless a future ceremony's git operations actually fail or the "repository moved" push message actually surfaces | **Applies, re-checked (§A9)** — still present, still functionally inert; not chased further |

## §A3 — Routine re-verification (all 7 issues, live, full bodies)

Per the task's own instruction: every open issue re-read via `issue_read`
(method `get`, full body), not trusted from T39 retro's table without
re-checking directly, plus a direct `get_comments` re-fetch on #144.

| Issue | `updated_at`/comments at T39 retro | `updated_at`/comments now (live, this ceremony) | Changed? |
|---|---|---|---|
| #149 | `2026-08-16T14:20:31Z`, 4 comments | `2026-08-16T14:20:31Z`, 4 comments | No |
| #145 | `2026-08-15T05:01:29Z`, 1 comment | `2026-08-15T05:01:29Z`, 1 comment | No |
| #144 (D1) | `2026-08-15T07:01:03Z`, 1 comment | `2026-08-15T07:01:03Z`, 1 comment | No |
| #134 | `2026-08-14T16:37:49Z`, 0 comments | `2026-08-14T16:37:49Z`, 0 comments | No |
| #130 | `2026-08-14T16:30:25Z`, 0 comments | `2026-08-14T16:30:25Z`, 0 comments | No |
| #126 | `2026-08-14T16:12:26Z`, 0 comments | `2026-08-14T16:12:26Z`, 0 comments | No |
| #124 | `2026-08-15T16:25:34Z`, 1 comment | `2026-08-15T16:25:34Z`, 1 comment | No |

**Every field matches T39 retro's own live-fetched table exactly,
byte-for-byte — zero blockers changed state, zero new comments on any of
the 7, and every timestamp predates PR #261's own merge (`05:34:51Z`).**

**Every issue's full body re-read this ceremony (`issue_read(method: get)`
on all 7, individually, not just `list_issues`'s batch return, plus a
direct `get_comments` on #144), genuinely reconsidering rather than
assuming last sprint's disposition still holds:**

- **#144 (D1):** still zero authorization on `CancelBooking`/
  `CreateBooking` — `domain.Booking` records no owner field, so there is
  nothing to check either RPC's caller against. Blocked on a product
  decision (what "owns" a public quote-and-book Booking, and whether the
  unauthenticated flow should remain possible at all) — ADR-0015,
  unanswered. Nothing in the issue's own text has changed; no code change
  landed this sprint that could touch `internal/booking/domain`.
- **#149:** Payments still resolves the *actor* to a verified `User.ID` but
  accepts caller-supplied *ownership facts* (`booking_host_id`,
  `game_host_id`, admin-assignment lists, `entrant_player_id`) with no port
  to resolve them against — blocked on D1 (the Booking-side fact) plus the
  still-unbuilt Game-Admin/Competition-Admin durable store the issue's own
  text names as a sub-problem ("that sub-gap should probably be closed
  first"). Both halves unchanged.
- **#124:** `internal/socialplay/domain/game.go`'s `Game.Cancel()` still
  flips status only — the issue's own three named product questions (court
  release, refund automaticity via T12.3's `RefundPayment`, waitlist
  handling) still need Product Owner input this team cannot supply
  unilaterally. No new fact this sprint.
- **#126:** `domain.Game`/`domain.Registration` still carry no price/fee
  field; T8.10's placeholder ($10.00, visibly labeled) is still what ships.
  Still needs Product Owner input on whether the price is per-Game,
  per-Registration, or per-head-including-guests — unchanged since T8.10.
- **#130:** `RefundPayment` still rejects `PAYABLE_TYPE_NO_SHOW_FEE` with
  `domain.ErrInvalidPayableType`, pinned by two tests in two layers. The
  issue's own stated open question — whether reversing a `no_show_fee` is
  the product behavior actually wanted, and what (if anything) it should
  project onto `Registration.PaymentStatus` — is not answerable from
  symmetry alone. Unchanged.
- **#145:** still needs a real IdP tenant issuing a non-uuid `sub` claim,
  which exists nowhere in this environment; both named failure modes
  (`AddCourt`/`AddCameraLink`/`AttestCameraConsent` for pre-existing
  Facilities, `RequestRecurringHire`'s `EnsureClubRole`) still fail closed,
  so this remains a forward-migration gap, not an active vulnerability.
  Structurally distinct from the now-fully-closed #164/#237 backfill, that
  distinction re-verified again this ceremony.
- **#134:** still needs a real screen-reader session (NVDA/JAWS/VoiceOver)
  and rendered-in-browser measurements this environment's lack of
  assistive-technology hardware cannot supply. No ticket since T12.5 ships
  a UI change to any of the three named screens, so there is nothing new
  for even a targeted pass to cover.

**Genuine reconsideration, stated plainly rather than assumed.** None of the
seven issues' blocking conditions is something this team could resolve
unilaterally without either guessing at a product decision (#124, #126,
#130), guessing at DECISION D1 (#144, #149), or fabricating environment
capability that does not exist here (#145's real IdP tenant, #134's
assistive-technology hardware). This was checked fresh against each issue's
current full body this ceremony, not inherited from T39's disposition.

**Migration-tooling classification (`golang-migrate`/`goose`), re-checked:**
`HANDOFF.md`'s Cross-cutting section still carries exactly the line every
prior ceremony has quoted (`grep -n "golang-migrate\|goose" HANDOFF.md`,
unchanged content — line numbers shift only because §A0's own row/narrative
additions pushed later text down, not because the Cross-cutting section's
own content changed). `ls docs/adr/` ends at `0017`; `ls db/migrations/`
ends at `0026`. **Still correctly unticketed, unchanged — twenty-three
separate prior ceremonies (T11–T14, T21–T39) have already ruled or
re-confirmed this settled roadmap debt, and zero PRs merged this sprint to
supply a new fact that could overturn it.**

**`HANDOFF.md`'s Cross-cutting section re-scanned in full for anything
newly actionable** (the same check T18 through T39 ran): a targeted grep
for the phrases prior disclosed-but-unticketed gaps have carried (`raise at
the next backlog refinement`, `follow-up ticket`, `not yet ticketed`,
`deferred with no ticket`, `logged not fixed`, `still open, still not
ticketed`) returns the same candidate set as T39's own scan, at line
offsets shifted down by exactly the number of lines §A0's corrections added
earlier in the file; each hit's surrounding text was re-read to confirm it
is the same candidate at a new offset, not a new one. **No new candidate
found** — every hit resolves to one of: already closed (the write-handler
malformed-ID guard → #97/T10.7; the `RefundPayment` wiring gap → T12.3; the
Competitions `PayableType` gap → filed and then closed), a decision this
project has already made and is not reopening (ADR-0006's waitlist
deferral, ADR-0009's OAuth/messaging deferral), tracked by an issue already
in this ceremony's 7-issue swept set (T5.2's `Game.Status`-on-`Register`
finding → #124's cascade; T8.10's placeholder price → #126), or a
long-settled, never-actionable observation (T7.3's count-based waitlist
`Position` formula note, unticketed since T7 and re-confirmed
non-actionable by every prior re-scan; T6.4's uncommitted-throwaway-program
note, superseded by T4/T28's own committed concurrency tests). This is the
twenty-first re-scan (T18–T27, T30–T39) to confirm no second instance of
T19's one genuine find exists — T28/T29 are excluded from this count the
same way T31's through T39's own citations excluded them, since both were
non-zero-ticket sprints whose Ceremony 1 did not run this same re-scan
framing.

**Toolchain, re-run directly against the unmodified tree before any
planning text was written:**

```
$ make test-domain
go test ./internal/.../domain/... ./internal/.../app/... -race -count=1
ok      github.com/nhuthuynh/white-label/internal/booking/domain       1.022s
ok      github.com/nhuthuynh/white-label/internal/competitions/domain  1.029s
ok      github.com/nhuthuynh/white-label/internal/facilities/domain    1.016s
ok      github.com/nhuthuynh/white-label/internal/identity/domain      1.018s
ok      github.com/nhuthuynh/white-label/internal/payments/domain      1.020s
ok      github.com/nhuthuynh/white-label/internal/socialplay/domain    1.026s
ok      github.com/nhuthuynh/white-label/internal/booking/app           1.026s
ok      github.com/nhuthuynh/white-label/internal/competitions/app      1.025s
ok      github.com/nhuthuynh/white-label/internal/facilities/app        1.027s
ok      github.com/nhuthuynh/white-label/internal/identity/app          1.016s
ok      github.com/nhuthuynh/white-label/internal/payments/app          1.023s
ok      github.com/nhuthuynh/white-label/internal/socialplay/app        1.028s
```

12/12 packages green — matching T39 retro's own count exactly on an
unmodified tree (no code change has landed since).

## §A4 — Migration-header-ownership check: not applicable this sprint

No T40 ticket exists (§A6), so no ticket names a database table or a
migration file.

## §A5 — The whole open backlog, ranked, with a disposition for each

All 7 open GitHub issues, generated directly from §A1's swept set:

| Issue | Ranked | Disposition |
|---|---|---|
| **#144** `CancelBooking`/`CreateBooking` have no authz | **Escalated** | D1 unanswered; §A7. Twenty-seventh sprint carrying this |
| **#149** Payments' remaining caller-supplied facts | **Untouched, correctly** | Blocked on D1 (the Booking-side fact) plus an unbuilt Game-Admin/Competition-Admin store (the issue's own named sub-problem); unchanged |
| **#124** Game-cancellation cascade | **Untouched, correctly** | Needs Product Owner input on cascade semantics; unchanged |
| **#145** pre-existing UUID rows vs. `Principal.Subject` | **Deferred** | Needs a real, non-uuid IdP `sub` claim this environment cannot produce; unchanged |
| **#126** real per-Game price field | **Deferred** | Needs Product Owner input on price shape; unchanged since T8.10 |
| **#130** refunding a `no_show_fee` Payment | **Deferred** | Carries its own stated open product question; unchanged |
| **#134** WCAG manual screen-reader pass | **Deferred** | Needs real assistive-technology hardware this environment cannot provide; no UI change has shipped since T12.5 for a targeted pass to cover |

**Every one of the 7 tracked issues remains genuinely blocked, independently
re-verified rather than assumed (§A3).** None of the seven is actionable
this sprint without guessing at D1, a real IdP tenant, a Product Owner
decision this team cannot make unilaterally, or assistive-technology
hardware this environment does not have.

**Considered and rejected: manufacturing partial progress under a stated
assumption, per the task's own instruction to check first.** For each of
the three PO-blocked issues (#124, #126, #130), the blocking question is not
"what should the code look like" but "which of several materially different
product behaviors is wanted" — #124's court-release/refund/waitlist
semantics, #126's per-Game-vs-per-Registration-vs-per-head pricing unit,
#130's whether a no-show-fee reversal is wanted at all. Shipping code
against a guessed answer to any of these is not a smaller version of the
same ticket, it is a different ticket that may need to be reverted the
moment the real answer arrives — the same reasoning this project has
applied to D1/D2 throughout, extended here to product questions that are
escalated in substance even though no ADR names them. No defensible
assumption exists for any of the three that would not risk exactly that.

## §A6 — Why zero tickets, and why that is the honest answer

**The alternatives this project has repeatedly named and rejected are still
rejected, for the same reasons:**

1. **Manufacture a ticket against one of the 7 blocked issues by guessing
   at its blocker.** Not done — every blocker was independently
   re-verified live this ceremony (§A3/§A5), and none has moved. Per the
   task's own explicit instruction, no T40 ticket implements
   `CancelBooking`/`CreateBooking` authorization or a reviewer-authorship
   carve-out — D1 and D2 both remain formally open, unresolved, and this
   ceremony does not guess at either.
2. **Take the migration-tooling item as new scope.** Not done — settled
   roadmap debt, re-confirmed unchanged (§A3), with no new fact this sprint
   to overturn twenty-three prior ceremonies' own ruling.

**The third alternative — the one this ceremony takes — is a 0-ticket
sprint: the nineteenth in this project's history by total count, and the
eleventh sprint of an unbroken consecutive run (T30, T31, T32, T33, T34,
T35, T36, T37, T38, T39, T40) since T28 ended the earlier T20–T27
eight-sprint streak.**

**Per T23 retro's finding 7 (as extended by T30 retro recommendation 6, T31
retro recommendation 6, T32 retro recommendation 6, T33 retro
recommendation 6, T34 retro recommendation 6, T35 retro recommendation 6,
T36 retro recommendation 6, T37 retro recommendation 6, T38 retro
recommendation 6, and T39 retro recommendation 6), the "is a 0-ticket
sprint healthy" question is not reopened by default** — it was closed as
genuinely settled once the user directly chose continued deferral (T21),
and stays closed absent one of T21 retro's two named reopening conditions,
checked live this ceremony rather than assumed:

1. **A materially different blocker profile emerging** — e.g., an eighth
   issue joining D1's cluster, or one of the environmentally-blocked issues
   (#145, #134) turning out to be answerable from inside this environment
   after all and still not acted on. **Did not fire** — the 7-issue set and
   every blocker's substance are unchanged (§A3).
2. **The backlog running dry in the other direction** — every issue closing
   without replacement. **Did not fire** — `totalCount: 7`, unchanged
   (§A1).

Neither condition held at T39 retro's own check; neither holds now. This
ceremony accordingly does not re-run the healthiness analysis from scratch,
consistent with T23/T30/T31/T32/T33/T34/T35/T36/T37/T38/T39 retro's own
disposition of the question.

**Dependency-completeness check: not applicable.** With no ticket, there is
no producer/consumer pair to check either question of
(`sprint-process.md`'s two-question form).

## §A7 — DECISION D1 and DECISION D2 remain unanswered

Re-verified this ceremony (§A3): #144 carries exactly one comment, T14.3's
original escalation, unchanged. Both ADR-0015's and ADR-0016's own `##
Status` headings are unchanged (§A0).

> **DECISION D1 (for the user / Product Owner) — twenty-seventh deferral.**
> When somebody books a court through the public flow *without an
> account*, who should be allowed to cancel that booking later, and should
> booking without an account remain possible at all?
> `docs/adr/0015-booking-ownership-for-public-bookings.md` lays out four
> options with their costs and recommends none. No T40 ticket exists to
> implement this, and per ADR-0015's restriction list no T40 PR may guess
> at it regardless. D1 has now carried its single T14.3 comment for
> **twenty-seven consecutive sprints (T14 through T40)** with no second
> escalation attempt beyond the original ADR text — the increment this
> ceremony takes per the per-sprint counting convention T29/T30/T31/T32/
> T33/T34/T35/T36/T37/T38/T39's own retros established (a new sprint
> boundary opening with #144 still uncommented is the trigger, and T40's
> own Ceremony 1 is that boundary, verified live via
> `issue_read(get_comments)` above).

> **DECISION D2 (for the user) — no PR to exercise the interim rule
> against this sprint.** May a session that reviews and merges a PR also
> author code on it? `docs/adr/0016-reviewer-authored-code-on-a-reviewed-
> pull-request.md` lays out four options and recommends none. T40 ships no
> tickets and therefore no PRs, so this sprint lands in the structurally
> weaker "no PR existed" shape (per T21 retro's naming) — the same shape
> T20, T21–T27, and T30–T39 all landed in.

**Neither is implemented, decided, or guessed at by this ceremony.** Both
remain exactly as escalated as `sprint-process.md`'s own restriction lists
require. Per the task's own standing instruction, the sprint loop continues
regardless — re-deferring both as needed, not escalating further absent a
materially different blocker profile or the backlog running dry, and
neither condition fired this ceremony (§A1, §A5, §A6).

## §A8 — Shared-file pre-assignment, dispatch isolation, and same-wave
verification: not applicable this sprint

| Artifact | Owner | Notes |
|---|---|---|
| `HANDOFF.md` | this ceremony only | No execution ticket exists this sprint to collide with it |

**Dispatch isolation: does not apply, trivially.** T40 has zero tickets and
therefore zero waves — no implementer to isolate, and no wave whose text
could omit naming an isolation mechanism. The "Dispatch isolation" section
of `sprint-process.md` is read and understood, but has nothing to be
exercised against this sprint; per T30 retro's own recommendation 3 (and
T31/T32/T33/T34/T35/T36/T37/T38/T39 retro's own recommendation 5/4), its
one soft observation stays open for the first sprint that actually
dispatches a multi-implementer wave, not resolved pre-emptively here.

**Same-wave shared-interface verification rule: does not apply, trivially.**
T40 has zero tickets. There is no wave to have a collision within.

## §A9 — Stale repo-metadata artifact: re-checked, still present, still inert

Per T39 retro's recommendation 7, not chased further absent a functional
symptom, but re-checked as part of this ceremony's own live API calls:
`pull_request_read(get, #260)`, `pull_request_read(get, #261)`, and
`list_pull_requests` all still return `head.repo.full_name`/
`base.repo.full_name` as `nhuthuynh/pickleball-platform` with the stale "A
Vinyl-Trading enterprise app..." `description`. The local checkout's `git
remote -v`, `git log`, and `git status` all ran clean against
`nhuthuynh/white-label` throughout this ceremony's own verification work,
with no "repository moved" push message surfaced (no push has been made yet
at this point in the ceremony). **Confirmed still present, confirmed still
functionally inert.** Not pursued further, per the task's own instruction
and T39 retro's recommendation 7.

---

# Ceremony 2 — Sprint planning

## Sprint goal

> **Confirm, rather than assume, that the backlog remains exactly as
> blocked as T39's retro left it, correct the two structural bookkeeping
> items T39's retro deliberately deferred (its own `HANDOFF.md` row and
> Task-backlog narrative), and continue the two running counters on their
> established per-ceremony/per-sprint schedules.** All 7 open issues were
> independently re-verified live and found unchanged: two on DECISION D1,
> one on a real IdP tenant this environment cannot provision, three on
> genuine Product Owner input this team cannot supply unilaterally, one on
> real assistive-technology hardware. `HANDOFF.md`'s Cross-cutting section
> was re-scanned in full and produced no new candidate. The
> `golang-migrate`/`goose` roadmap-debt classification remains settled.

**What this sprint does not claim:**

- **This is not evidence the project has run out of real work.** A
  nineteenth 0-ticket sprint by total count — and the eleventh sprint of
  the current unbroken consecutive run, since T28 broke the earlier
  T20–T27 streak — is the honest output of a discipline that takes only
  what is genuinely unblocked and files what is genuinely disclosed, not a
  target this team engineered.
- **This does not mean the 7 tracked issues have become less real or less
  worth doing eventually.** Every one is exactly as real and exactly as
  blocked as it was at T39's retro.
- **D1 and D2 remain formally open and unanswered as ADR decisions.**
  Nothing this ceremony did or found bears on either decision itself. No
  ticket implements `CancelBooking`/`CreateBooking` authorization or a
  reviewer-authorship carve-out.
- **The Docs-index/Task-backlog corrections in §A0 are bookkeeping, not
  scope.** Landing them in this PR does not make this a non-zero-ticket
  sprint in the sense `sprint-process.md`'s Tickets/Waves sections track —
  there is still no implementer ticket, no wave, no PR against application
  code.

## Tickets — 0 items, 0 points

**None.** §A6 states the reasoning in full: every tracked issue is
genuinely blocked (§A5), and the one Cross-cutting candidate that has
looked promising on past reads (migration tooling) remains settled roadmap
debt with no new fact to overturn that classification.

## Waves

**None.** No ticket exists to wave-assign, so the dispatch-isolation
naming obligation §A8 describes has nothing to attach to this sprint.

## Recorded disagreements (Ceremony 2 rule 3 — not smoothed over)

**None recorded this sprint.** No party found grounds to disagree with
continuing the re-deferral pattern on D1/D2, with the zero-ticket
conclusion, or with the nineteenth-by-total-count / eleventh-consecutive
framing in §A6 — every blocker was independently re-verified live rather
than assumed (§A3), and none has moved.

## Sprint-level Definition of Done

1. **No ticket to merge; sprint goal met as stated** — confirm-and-report
   on the backlog, plus land the two named §A0 bookkeeping corrections —
   not build-and-ship.
2. **The merged-fix issue sweep run and reported with its count (7,
   reconciled arithmetically — §A1)** — by the retro (reporting, not
   blocking) and again by T41's Ceremony 1 (authoritative).
3. **Scoring owed at the retro:**
   - **(a)** Did this ceremony's own claim — that all 7 open issues remain
     genuinely blocked, re-verified live — hold for the whole sprint (a
     live re-check at retro time, not a re-read of this document)?
   - **(b)** Is the `golang-migrate`/`goose` roadmap-debt classification
     still correctly unticketed at retro time?
   - **(c)** Did D1 or D2 get answered mid-sprint, as a formal ADR
     decision?
   - **(d)** Did either of T21 retro's two named reopening conditions fire
     mid-sprint (checked live, not re-read from this document)?
4. **Not scoreable by T40 and deliberately not pre-empted:** D1 and D2
   remain the user's own decisions to make, on whatever timeline the user
   chooses.
5. Retro in `docs/process/t40-retro.md`, indexed by a `## T40 sprint
   retro` stub in `docs/LESSONS.md`. `HANDOFF.md`/`CLAUDE.md` state
   updated — noting that **T41's Ceremony 1**, not the retro, corrects
   T40's Docs-index row (the ordinary convention).
