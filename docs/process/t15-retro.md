# T15 Sprint Retro

Ceremony 3 per `docs/process/sprint-process.md` (read in its **T15.1-amended**
form — PR #188, merged 14:35:35Z, this same sprint — since T15.1 is the ticket
that named the merged-fix sweep's "third state," demoted the per-PR close to
an *optional* early close, and made the sprint-level sweep primary. This
ceremony is bound by that amendment and is also its first live test), six-role
team (briefs: `docs/agent-operating-handbook.md` Part B), held against
`docs/process/t15-sprint-plan.md` (§A0–§A15), `docs/process/t14-retro.md` as
the precedent and rigor bar, `HANDOFF.md`, `docs/adr/0016-*.md`, and the real
PR/issue history on `nhuthuynh/white-label` (GitHub-side name
`pickleball-platform`) — PRs #186–#193, issues #124–#185.

Every timestamp, merge order, label and issue-state claim below was pulled
from GitHub's own `created_at`/`merged_at`/`submitted_at`/`closed_at` fields
and from direct reads of the merged tree at `c18a74c`, not inferred from PR
titles or from the sprint plan's forward-looking text (CLAUDE.md rule 10).
Where this retro corrects the live issue tracker rather than merely reporting
on it, that action and its justification are stated in the finding that
prompted it — this ceremony performed three bookkeeping actions on GitHub
during its own run, and each is disclosed where it happened rather than
summarized only at the end.

**Sprint outcome, stated before the findings that qualify it:** all 7 tickets
(34 points, T15.1–T15.7) merged in a single unbroken work block — plan-doc
merge at 14:27:16Z to final ticket merge (T15.5, PR #193) at 15:36:26Z,
**1h09m10s wall-clock**, no session-limit interruption, no worktree recovery
needed, every open→merge window between 81s and 651s (none within an order of
magnitude of T14's 8–9s recovery-review signature). Wave sequencing held
exactly as planned: T15.3 (PR #190, 14:53:29Z) merged before both of its
Wave-2/3 dependents, T15.4 (PR #192, 15:14:20Z) and T15.5 (PR #193,
15:36:26Z). No PR took a second review loop. `docs/adr/0016-*.md`'s interim
rule (a reviewer that finds a gap requests changes, never fixes it) was
followed for every ticket after it took effect — checked directly against
commit history, not assumed (finding 4).

**What did not go as the sprint's own goal promised, in one sentence, so nine
findings do not have to be read before the headline is known:** T15's stated
goal was "Payments stops taking the word of the caller about who the admins
are, reading both stores instead"; **it does not.** `RecordOfflinePayment`'s
authorization is byte-for-byte identical to how T15 found it. That is finding
1, and it is a real, well-verified engineering discovery, not a shortcut —
but it means the sprint's headline claim was not achieved, and this retro
says so plainly rather than let "T15.5 merged" read as "T15.5's goal was met."

---

## 1. T15.5 did not close #168 — a genuine structural finding, independently re-verified by this retro, and a real (if narrow) planning-ceremony miss that a cheaper check would have caught before dispatch

**QA (what T15.5 actually shipped, verified against the merged tree at
`c18a74c`, not taken from the PR's own account):**

```
internal/socialplay/app/service.go:1009:func (s *Service) ListGameAdmins(ctx context.Context, gameID string) ([]domain.GameAdmin, error) {
internal/competitions/app/service.go:760:func (s *Service) ListCompetitionAdmins(ctx context.Context, competitionID string) ([]domain.CompetitionAdmin, error) {
```

Both reads exist, both are exported, both are exactly what §A12's GAP B said
they were. And:

```
internal/payments/app/service.go:375:func (s *Service) RecordOfflinePayment(ctx context.Context, in RecordOfflinePaymentInput) (domain.Payment, error) {
...
        if !uuidShape.MatchString(in.PayableID) {
```

`RecordOfflinePaymentInput` and `RefundPaymentInput` carry only `PayableID` —
a Registration's or CompetitionEntry's own id, never its parent Game's or
Competition's. `internal/socialplay/app/service.go` exports fifteen methods;
none of them takes a `registrationID` and returns a `gameID`. The closest is
`ListRegistrationsForGame`, which needs the Game id as *input* — the exact
thing missing. This retro independently re-grepped both `app.Service`s rather
than trusting either the PR body's or the review's claim that no such method
exists, and confirms it: **there is no resolution read, on either side.**

**So T15.5's finding is real.** `RecordOfflinePaymentInput.AssignedGameAdminUserIDs`
/ `AssignedCompetitionAdminUserIDs` are untouched; `authorizeOfflineRecording`
is untouched; the proto fields are deliberately **not** marked
`[deprecated = true]`, because — per T15.5's own commit message — *"they
remain the only mechanism authorizing an assigned admin."* PR #193 built the
read-side ports (`internal/payments/port/{game_admin_reader,competition_admin_reader}.go`)
and real adapters, tested against the real `socialplay`/`competitions`
`app.Service`s (not fakes — the assign→observe→revoke→not-observed control
instruction 7 asked for), and stopped there. Its review (submitted
15:36:20Z) independently re-derived the same negative before accepting it,
including reading #149's own "what closing it looks like" section via the API
to confirm the resolution-read gap was already named there. **This is careful,
honest, well-verified engineering that discovered a true fact about the
codebase — not an implementer giving up early.**

**PO (the sequencing question, scored as asked, not both-sided).** Was
splitting one conceptual piece of work (#168) across three tickets in three
waves — T15.3 (store) → T15.4 (in-context read, Wave 2) → T15.5 (cross-context
consumption, Wave 3) — sound, given T15.5 hit a wall that could have been
found at planning time?

**The sequencing *structure* is sound, and is proven sound by its own sibling
in the same sprint.** T15.3→T15.4 is the identical shape as T14.4→T14.5
(store, then an in-context consumer), and it worked cleanly: #147 closed,
9-failure and 13-failure mutation checks both reproduced exactly as
predicted, no rework. The structure is not the problem.

**What is a real, narrow planning-ceremony miss is §A12's GAP B check
itself — it verified the wrong half of the dependency, and the wrong half was
cheap to check correctly.** GAP B's text, quoted from the sprint plan: *"T15.5
must read Social Play's admin set from inside Payments.
`internal/socialplay/app/service.go:1009` already exports `ListGameAdmins(ctx,
gameID)`. **T15.5 therefore needs no edit to `internal/socialplay` at all**"*
— and concludes *"checked and *not* a gap."* Every word of that is true. But
the question it answers is **"does the producer's capability exist?"**, and
the question that actually gated T15.5 was **"does the consumer's own call
site already hold the argument that capability needs?"** Those are different
questions, and only the second one was load-bearing here — Payments never
lacked a *function to call*; it lacked a *value to call it with*. The second
question is answered by reading `RecordOfflinePaymentInput`'s own field list,
which is one grep, not an implementation attempt — T15.5's review performed
exactly that grep, after merge, in about one paragraph.

Crucially, **this was not an untested or unprecedented kind of check to have
asked.** GAP B exists in the plan specifically *because* the planners
recognized T15.5 as a new, cross-context hop that needed verification — the
Ceremony 1 team knew to look. They looked at the producer side (does
`ListGameAdmins` exist and is it exported?) and stopped one hop short of the
consumer side (does the ticket that will call it have the id?). That is a
different, and cheaper, mistake than T12's "no producer exists at all"
class (`docs/LESSONS.md`'s T12 entry) or T14's "capability exists, but the
plan never asked whether the *sprint's own goal statement* should hedge
against it" — this is its own third shape: **a capability-existence check
that stops at the producer's signature instead of walking the argument back
to the consumer's own inputs.** It is worth naming as a distinct check for
exactly that reason — recommendation 1 below.

**Also worth stating plainly: the sprint's own goal text did not hedge
against this outcome, unlike every other known gap.** T15's "what this sprint
does not claim" section lists #149 (partially), #164, the registrant-read
boundary, and seven untouched issues by number — and does **not** list #168
as an issue that might not close, because #168 was the sprint's headline
target. Every other disclosed risk in this sprint was pre-named; this one
was a genuine, undisclosed-at-planning-time surprise, which is exactly why it
belongs in this retro's headline rather than in a routine "partial fix"
paragraph.

**Score, stated as asked — argued, not both-sided:** this is a real planning
miss, narrow in scope, worth one added clause to the dependency-completeness
check (recommendation 1) — not a sequencing failure, not a story-points
failure (8 points bought real, tested, reusable infrastructure — the read-side
ports are not wasted; whoever resolves the missing read next consumes them
directly), and not a reason to re-litigate the T15.3→T15.4→T15.5 split, which
its own sibling proved works. It is a reason to say, without softening, that
**T15's sprint goal was not met on its own central clause**, and to fix the
one question that would have caught it before dispatch rather than after.

## 2. Two PRs said "closes #N," in the same sprint that just rewrote the rule around exactly that failure — and neither call was made. This retro's own sweep caught both, which is the mechanism working, not evidence nothing went wrong

**The merged-fix sweep, run as this ceremony's first act per
`sprint-process.md`'s DoD (as T15.1 itself just amended it):**

**Step 1 — list the open issues** (live, at ceremony start, before this
retro's own corrections): `list_issues(state: OPEN)` → **`totalCount: 13`**:
#124, #125, #126, #130, #134, #137, #144, #145, #149, #164, #167, #168, #185.

**Step 2 — reconcile arithmetically, before reading any issue
individually.** `open_at_end_of_T14 − closed_in_T15 + opened_in_T15`. At
T14's close, and re-verified by T15's own Ceremony 1 (§A0), the count was
**13**. T15's Ceremony 1 opened **#185** (+1). Before this retro ran, the
*only* issue actually closed via the API during T15 was **#147** (−1, by
PR #192/T15.4's review, at 15:14:24Z). `13 − 1 + 1 = 13` — matches the
`totalCount` this ceremony read at its start. **The arithmetic reconciles,
and reconciling is not the same as everything being correct** — it is the
cheap check that proves nothing is *missing*, not that nothing is *stale*,
which is exactly what step 3 exists to catch.

**Step 3 — cross-reference every merged T15 PR's title/body against the
open list.** Three PRs claim a close in their own title:

| PR | Ticket | Title's claim | Issue's live state at ceremony start |
|---|---|---|---|
| #192 | T15.4 | "closes #147" | **closed** (`closed_at` 15:14:24Z, cited PR, comment posted) — correct, no hit |
| #191 | T15.6 | "closes #185" | **open** — `updated_at` still equal to `created_at` (14:15:13Z), i.e. untouched since Ceremony 1 filed it — **HIT** |
| #189 | T15.7 | "closes #137" | **open** — `updated_at` still equal to `created_at` (2026-08-14T16:38:27Z) — **HIT** |

**Both hits verified as full resolutions, not partial fixes wrongly
titled**, before acting on them — re-derived independently rather than
trusted from either PR's own account:

- **#185.** `internal/booking/adapter/postgres/repository.go` now maps
  Postgres `23503` on `bookings.court_id` to `bookingdomain.ErrInvalidCourtReference`
  (re-mapped `Internal → NotFound`, with the sentinel-reuse and code-choice
  reasoning recorded in its own doc comment), `toStatus` updated across
  booking/socialplay/competitions, and covered by five FK-modelling fakes plus
  a real-Postgres integration test. PR #191's review (submitted 14:58:34Z)
  independently mutation-tested the central claim — disabling the sibling
  contexts' `errors.Is` arm reproduced the exact `Internal` regression #185
  describes — and closed "No gaps found. Merging."
- **#137.** `rs256.RemoteKeys` implements `KeySource` over a live JWKS
  endpoint (HTTPS-only, TTL cache, rate-limited unknown-`kid` refresh, a
  bounded stale-serving window, reusing `NewStaticKeysFromJWKS` rather than a
  second parser); `cmd/server` gains `AUTH_JWKS_URL`, with both sources set
  treated as a startup error. PR #189's review (submitted 14:50:46Z)
  independently mutation-tested the stale-ceiling claim and confirmed the
  `https`-only guard and both-sources startup error directly against the
  code, and closed "No gaps found. Merging."

**Neither review's own text mentions the issue-close API at all** — contrast
PR #192's review, which explicitly states *"Merging, and will close #147 per
instruction 5"* and then does. **#191 and #189's reviews never say they will
close, and they don't.** This retro closed both, per the sweep's first
disposition (*"the PR fully resolved it → close it now… bookkeeping being
caught up, not a decision being re-made"*), each with a comment naming the
resolving PR and stating explicitly that the close is the sweep catching a
miss rather than routine bookkeeping:
[#185#issuecomment-5302959223](https://github.com/nhuthuynh/pickleball-platform/issues/185#issuecomment-5302959223),
[#137#issuecomment-5302959784](https://github.com/nhuthuynh/pickleball-platform/issues/137#issuecomment-5302959784).
**Post-sweep open count: 11**, re-verified live
(`13 + 1 − 3 = 11`, matching `totalCount: 11` read after both closes).

**One same-sprint open-and-close, named so the arithmetic's framing does not
obscure it:** #185 was opened by T15's own Ceremony 1 (14:15:13Z) and closed
within the same sprint (by this retro, since T15.6's own close never
happened) — a real instance of the class `sprint-process.md`'s arithmetic
step does not explicitly separate out. It nets to zero in the `open_at_start
− closed + opened` formula, which is correct arithmetic, but a reader scanning
only the three counts would not see that one of "opened: 1" and one of
"closed: 3" are the *same issue*, filed and resolved inside one sprint. Worth
naming explicitly, as the task asked, rather than let the net-zero effect
read as "nothing happened to #185 this sprint."

**PO (why this is not a repeat of T13's 0/9, nor T14's self-swept 6/6, and
does not fit either of the amended rule's first two states cleanly).** T15.1's
own three-state classification, written earlier the same sprint, does not
have a clean slot for what actually happened:

- **Not state 1** (per-PR close) for #185/#137 — no close happened at review
  time for either.
- **Not state 2** (swept by a party other than the merger) — nobody but this
  retro swept them, and this retro *is* sweep moment 1, running exactly as
  designed (*"the retro runs it and reports it… T13's retro found this gap
  precisely because it was free to run and report"*) — so state 2, in the
  sense of *"the next Ceremony 1"*, has not run yet.
- **Not state 3** (the merging session self-sweeps at sprint end) — unlike
  T14, **nobody attempted a self-sweep this sprint at all.** T14 achieved
  6/6 correct closes via an eleven-second self-sweep batch; T15 achieved 1/3
  (#147, at review time) and made zero attempt to catch the other two before
  this retro asked.

**Read against the amended rule's own stated failure condition, this is
close to the worst case it names, caught one ceremony early.** The rule's
own text: *"Only one outcome is a failure: an issue whose fix merged this
sprint is still open when the next Ceremony 1 sweeps."* Had this retro not
run the sweep, **#185 and #137 would have been exactly that outcome** at
T16's Ceremony 1 — sitting open for a full sprint boundary, misdescribing the
codebase the entire time, with two PRs on record falsely claiming to have
closed them. This retro catching it one ceremony earlier than the failure
condition fires is the sweep working as intended (moment 1 exists to be a
free early check), but it is **not** evidence the underlying discipline
improved — if anything, T15 is a regression from T14 on the *effort* axis:
T14's self-sweep at least represents someone trying, however insufficiently;
T15 shows zero attempt on 2 of 3 "closes" claims until an external ceremony
asked. **Consequence, stated per the amended rule's own text: this does not
discharge T16's Ceremony 1 from running its own full sweep** — it re-runs
the arithmetic anyway, and should treat this retro's closes the same way it
would treat any other prior sweep's output: verified, not trusted.

## 3. The correlation worth acting on: every review that explicitly read an issue's state from the API also remembered to close it; every review that skipped that check also skipped the close

**BA (the mechanism, read across all five T15 reviews that named a live
issue).** T15.1's own instruction 4 (this sprint's own new clause) requires
*"each named issue's current state, read from the API, not from memory"* in
every review. Checked directly against each review's text:

| PR | Review states issue state, citing the API? | Did the PR's claimed close actually happen? |
|---|---|---|
| #192 (T15.4) | **Yes** — *"#147 and #168 both confirmed `state: open` via the API just now"* | **Yes**, #147 closed |
| #193 (T15.5) | **Yes** — *"Issue #149's own … section, read directly via the API"* | N/A — correctly claims no close |
| #190 (T15.3) | Partial — *"#147/#168 correctly left open rather than claimed closed,"* no explicit API citation | N/A — correctly claims no close |
| #191 (T15.6) | **No** — #185 is never mentioned in the review at all | **No**, missed (finding 2) |
| #189 (T15.7) | **No** — #137 is never mentioned in the review at all | **No**, missed (finding 2) |

**The two reviews that skipped the state-check clause are exactly the two
whose PRs' claimed closes were never performed.** This is not proof of
causation from two data points, but it is a clean, cheap, mechanically
checkable signal: a review that engages with an issue's live state — even
just to confirm it is still open, as #192's and #193's both did — is a review
that is looking at the issue tracker at all, which is the precondition for
remembering to update it. A review that names an issue only inside its own
prose summary, with no API-sourced state check, has nothing forcing that
look.

**Recommendation, not a new rule — see recommendation 2:** distinguish
**"closes #N"**-titled PRs from **"partial fix for #N"**-titled ones in the
DoD, rather than treating both under one "optional early close." A PR whose
own title already asserts a close is not offering an early close as a
convenience — it is making a claim that is false the moment the PR merges if
the API call is not also made. Optionality is the wrong frame for a title
that has already committed to the fact.

## 4. A second, unrelated instance of a disclosed-but-unfiled residual — in the very PR that fixed the last one

**QA.** PR #191's (T15.6) own body, under "Review notes": *"The latent race
class in §7 may deserve its own issue; I deliberately did not file one."*
Its review: *"The latent FK-race class … is correctly left unfiled as a new
finding, not this ticket's scope — noting it here so it isn't lost, but not
blocking."* The gap itself, quoted from the PR body's §7: nine other
FK-backed write paths (`registrations.game_id`, `competition_entries.competition_id`,
`games.venue_facility_id`, `competitions.venue_facility_id`,
`discount_rules.facility_id`, `courts.facility_id`,
`facility_camera_links.facility_id`, `recurring_hire_templates.court_id`,
`.requested_by_user_id`) are currently safe only because an app-level read
resolves the parent before the insert — **guarded by a read, not a
translation**, so a concurrent delete between that read and the insert would
each produce an unclassified `23503` and a 500, the identical shape #185 was
just fixed for, on a narrower window.

**This is a disclosed, cross-sprint-outlasting gap that neither the
implementer nor the reviewer filed.** Per `sprint-process.md`'s board-of-record
rule: *"GitHub issues are the board of record for anything that outlives its
sprint — cross-sprint follow-ups, disclosed-but-deferred gaps, and
escalations. These are mandatory, not discretionary: an item deferred out of
a sprint without an issue is a process violation, not a judgement call."* Two
parties looked directly at this gap, one after the other, and both correctly
declined to fix it in-ticket — and both then declined the one-line follow-up
that would have made the deferral compliant. **This is the same shape that
produced #185 in the first place** (T14.8 disclosed a residual, filed
nothing, T14's retro caught it, T15's Ceremony 1 filed it as #185) — recurring
inside the very ticket that closed #185, one PR later.

Per this project's own precedent (T14 retro recommendation 4 → T15's Ceremony
1 filing #185), **this retro does not file the issue itself** — that step
belongs to T16's Ceremony 1, the same as #185's own history. Recommendation 3
below carries it forward explicitly so it is not lost a second time.

## 5. A third instance of a stale claim sitting on the issue tracker, corrected by this retro rather than by anyone during the sprint

**QA.** #149's own T15 Ceremony 1 comment (14:15:59Z) predicted: *"`assigned_game_admin_user_ids`
| **T15.5 resolves it from the store**"* and the identical claim for
`assigned_competition_admin_user_ids`. **Finding 1 already establishes this
did not happen** — T15.5 built the read-side ports and wired none of them in,
and the app-layer input fields plus the proto's admin-list fields are all
untouched. Neither T15.5's PR nor its review went back to #149 to correct
the prediction its own outcome falsified; #149 carried exactly one comment,
the now-stale Ceremony 1 one, until this retro.

**This is the same failure shape as T14's finding 5** (PR #178 and its
review both calling #97 "still-open" — an issue closed two days earlier,
about a different gap) — an assertion about an issue's content, made once,
never re-checked against what actually shipped. This retro corrected #149
directly
([#149#issuecomment-5302979970](https://github.com/nhuthuynh/pickleball-platform/issues/149#issuecomment-5302979970)):
all five of #149's named facts remain open, not three of five as the
mid-sprint tracker claimed. **#149 stays open** — this correction changes
what the issue says, not its state.

**PdE (the common root across findings 2, 4 and 5, stated once rather than
three times).** All three are the same mechanism: a claim written to the
issue tracker — a title's "closes #N," an unfiled disclosure, a mid-sprint
prediction — that a later, truer fact superseded, with nobody obliged to go
back and reconcile the two. The project has a real, working detector for
this (the merged-fix sweep, and the "read state from the API" clause) but
both are scoped to **closing**, not to **correcting content that turns out
wrong**. Recommendation 4 names this explicitly rather than leaving it to be
independently rediscovered a fourth time.

## 6. DECISION D1 and DECISION D2 both remain open, unanswered by the user, as of T15's end — writing the escalation is not the same as answering it

**Re-verified, not assumed.** #144 (D1) carries exactly one comment —
T14.3's original escalation at 07:01:03Z on 2026-08-15 — unchanged this
sprint; no T15 PR references #144 at all. `docs/adr/0016-*.md`'s (D2) own
`## Status` field reads "Escalated — awaiting decision (D2)," not "Accepted,"
and nothing in the repository (no comment, no follow-up PR, no CLAUDE.md
edit) records the user having answered it. **Both stay stated plainly as
carried-forward items, third and first deferral respectively — this retro
does not attempt to resolve either, and does not let ADR-0016 having been
*authored and merged* this sprint read as progress on the question it asks.**
Writing a well-constructed escalation is real, credited work (finding 7
below), and it is a different thing from an answer arriving.

## 7. ADR-0016's interim rule — checked directly against the commit and review record, not assumed from good intentions

**PE.** The interim rule, stated in the ADR and effective the moment PR #187
merged (14:34:10Z): *"a reviewer that finds a gap on a branch under review
**requests changes**; it does **not** fix the branch itself."* Checked every
T15 PR merged after that moment — #188, #189, #190, #191, #192, #193 — for
any reviewer-authored commit:

| PR | Commits | Reviewer-authored code? |
|---|---|---|
| #188 (T15.1) | 1, by the implementer | No |
| #189 (T15.7) | 1, by the implementer | No |
| #190 (T15.3) | 1, by the implementer | No |
| #191 (T15.6) | 2 — implementer's, **plus one merge-conflict resolution pushed by the reviewing session** | See below |
| #192 (T15.4) | 1, by the implementer | No |
| #193 (T15.5) | 1, by the implementer | No |

**#191's second commit is a merge-conflict resolution, not a gap fix, and
this retro checked the distinction rather than assumed it.** Its review:
*"Test-merging the shared branch tip produced one genuine conflict in
`internal/competitions/adapter/grpcapi/handler.go`: this PR's new
`ErrCourtNotFound` case and #190/T15.3's new `ErrCompetitionAdminNotFound`
case both landed as additions to the same `NotFound`-group `case` list.
Purely additive on both sides — resolved by keeping both arms … pushed the
resolution to this PR's source branch, then re-verified the entire toolchain
from a completely fresh worktree."* This is the identical shape
`sprint-process.md`'s worktree-recovery precedent already sanctions (T14.1 and
T14.9's `.PHONY`/target collision, both additive, resolved by keeping both) —
a conflict against a **moving base branch**, not a gap in the implementer's
own work, and it is stated here explicitly so it is not conflated with the
class ADR-0016 is actually about. **Zero instances of a reviewer fixing a
gap in the branch under review this sprint** — a clean result, on the first
sprint the interim rule was actually in force for its full duration.

## No finding on

**No finding on the wave structure or dispatch.** All 7 tickets executed in
one continuous block with no interruption; the pre-assigned shared-file table
(§A14) held with the one predicted, resolved conflict (#191, finding 7); no
Wave-1.5 checkpoint fired, correctly, since T15.3 had exactly two first-time
consumers.

**No finding on the label taxonomy.** #185 was filed pre-labelled and
conformant at creation (`role:product-engineer`, `type:bug`, three `context:`
values, all in the closed sets); no other issue's labels drifted this
sprint. T15's own Ceremony 1 sweep (§A8) already cleared the two unlabelled
holes (#147, #149) T14 left; this retro found nothing new to sweep.

**No finding on T15.1's or T15.2's own content.** Both are documentation-only
process/ADR changes, both verified `make test-domain`/`make gate-coverage`
green and untouched by this sprint's engineering, and both reviews correctly
declined to self-approve, recording their reviews as `COMMENTED` rather than
`APPROVED` — consistent with the ADR's own "no self-approve" observation.

**No finding on T15.3's or T15.4's engineering.** Both mirror T14.4/T14.5's
proven pattern exactly, both were independently mutation-checked by their
reviewer (9-failure and 13-failure reverts, both reproduced), and #147's
closure is real, cited, and correctly performed at review time.

---

## The sprint goal, scored: what was proven, what shipped unproven, and what did not happen at all

> *"Competitions gets the Competition-Admin store Social Play got in T14, its
> roster read widens from Host-only to Host-or-assigned-admin, and **Payments
> stops taking the word of the caller about who the admins are, reading both
> stores instead** — so the authorization rule CLAUDE.md locks in … is
> implementable trustworthily for the first time."*

**The first two clauses are met, verified independently.** `0021_competitions_competition_admins.sql`
exists with the composite-PK invariant and `text` actor columns per
ADR-0014 §5a; `ListEntriesForCompetition` is Host-or-assigned-Competition-Admin,
mutation-proven (9 and 13 real failures on revert); #147 is closed, correctly
and with a citing comment.

**The third clause — the sentence carrying the sprint's actual authorization
promise — is not met, and the sprint's own "does not claim" section did not
warn that it might not be.** `RecordOfflinePayment` still trusts the caller's
`assigned_game_admin_user_ids`/`assigned_competition_admin_user_ids`
unconditionally. #168 remains open. The gap was found honestly, verified
independently by this retro (finding 1), and the read-side infrastructure
T15.5 shipped is real and reusable — but "implementable trustworthily for the
first time" is not true of Payments today, and was not true at any point in
T15.

> *"Alongside it: a caller who names a court that does not exist gets a
> client error instead of a 500; this backend can verify a token from a live
> identity provider for the first time."*

**Both true of the code, and both were true of the issue tracker only after
this retro corrected it.** #185 and #137 were both fully, correctly resolved
in code and both sat open on GitHub, contradicted by their own merged PRs'
titles, until this ceremony (finding 2).

> *"the two process contradictions T14 left standing … are resolved into
> text, the first by amendment and the second by escalation to the user."*

**True, and both are real, credited work** (finding 7) — **and neither is
the same thing as the underlying question being settled.** The closure
sweep's amendment produced a live instance of the exact problem it was
written to fix, one PR later, inside the same sprint (finding 2). The
reviewer-authorship escalation (D2) remains exactly as open as D1 (finding
6).

**The agreed honest sentence, which T16's Ceremony 1 should carry into
`HANDOFF.md`'s Task-backlog entry** (`sprint-process.md` Ceremony 1 item 3
requires the retro's form, not a stronger one):

> T15 closed #147 (Competitions' roster read, Host-or-Competition-Admin,
> mutation-proven) but **did not close #168** — Payments' offline-payment
> authorization is byte-for-byte unchanged from before the sprint. T15.5
> built real, tested read-side infrastructure onto both admin stores and then
> discovered, by independently enumerating every exported method on both
> `socialplay.app.Service` and `competitions.app.Service`, that Payments has
> no way to obtain the `gameID`/`competitionID` its new readers need for the
> payable types that require one — a genuine structural finding, verified
> twice (by the implementer and independently by the reviewer), not a
> shortcut. The dependency-completeness check that cleared this ticket for
> dispatch (§A12 GAP B) verified that the producer capability existed but
> never checked that the consumer's own inputs could reach it — a narrow,
> real planning miss, cheap to have caught, and now a standing check
> (recommendation 1). Two PRs (T15.6, T15.7) titled "closes #N" and neither
> call was made; **this retro's own sweep caught both**, closing #185 and
> #137 with the resolving PR cited on each — this is the merged-fix sweep
> (rewritten by T15.1 this same sprint to make it primary) catching, one
> ceremony early, exactly the failure its own text names as the one
> unacceptable outcome. A third stale tracker claim (#149's mid-sprint
> prediction that T15.5 would resolve two of its five facts) was also
> corrected. A disclosed FK-race residual from T15.6 went unfiled, the same
> shape that produced #185 in the first place, and is carried to T16 rather
> than lost twice. ADR-0016's interim rule held for its first full sprint —
> zero reviewer-authored gap-fixes, one correctly-distinguished merge-conflict
> resolution. D1 and D2 both remain unanswered by the user.

---

## Recommendations for T16's Ceremony 1 and 2

1. **Add one clause to the dependency-completeness check: when a downstream
   ticket's plan says it will call an upstream export with an ID-shaped
   parameter, verify the downstream ticket's own inputs actually contain
   that id — not just that the upstream function exists** (finding 1). This
   is the narrow, cheap fix for the exact way §A12's GAP B under-verified
   T15.5: it confirmed `ListGameAdmins(ctx, gameID)` exists and is exported,
   and stopped, when the load-bearing question was whether T15.5's own
   `RecordOfflinePaymentInput` held a `gameID` to pass it. One additional
   grep at planning time — reading the consumer's own input struct — would
   have surfaced this before dispatch. Do not generalize this into a new
   ceremony step; it is one sentence added to the existing
   dependency-completeness check, in the same spirit as recommendation 7 of
   T14's retro asking not to over-mechanize a single instance.
2. **Distinguish "closes #N"-titled PRs from "partial fix for #N"-titled ones
   in the closure DoD, and make the close mandatory (not optional) for the
   former** (findings 2 and 3). T15.1's own amendment, written this sprint,
   correctly demotes the per-PR close to *optional* for the general case —
   but a PR whose own title already asserts a close is not offering an early
   close as a convenience; it is making a claim that is false the moment the
   PR merges without the corresponding API call. "Optional" is the wrong
   frame for a title that has already committed to the fact. Concretely: a
   review whose PR's title contains "closes #N" either performs the close
   or explicitly states, in the review, why it is not doing so — matching
   the pattern that already worked this sprint (PR #192's review: *"will
   close #147 per instruction 5"*, then did).
3. **File T15.6's disclosed FK-race residual as an issue at T16's Ceremony
   1** (finding 4) — nine write paths guarded by a read rather than a
   translation, the same class #185 was just fixed for, on a narrower
   concurrent-delete window. Two parties (PR #191's implementer and its
   reviewer) both correctly declined to fix it in-ticket and both declined
   the one-line follow-up that would have made the deferral compliant with
   the board-of-record rule. This is mandatory, not discretionary, per that
   rule — the same rule that produced #185 from an identical T14.8 near-miss.
4. **When a review or a PR body corrects an earlier claim about an issue —
   including one written by an earlier Ceremony's own comment — post the
   correction to the issue itself, not only to the PR** (finding 5). #149's
   mid-sprint prediction (T15.5 resolves two of its five facts) turned out
   false and nobody went back to fix the record until this retro. The
   merged-fix sweep and the "read state from the API" clause are both
   scoped to *closing*; neither currently obliges anyone to *correct*
   content that a later, truer fact superseded. This is the same mechanism
   as T14's finding 5 (#97's misattribution), recurring a third time in a
   different shape — worth one added sentence to the review checklist
   rather than a fourth restatement after a fourth instance.
5. **Continue treating the merged-fix sweep as the authoritative,
   non-discharged check regardless of what this retro found.** Per
   T15.1's own amended text, a prior ceremony's sweep — even a clean one —
   does not excuse the next Ceremony 1 from re-running the full arithmetic.
   T16's Ceremony 1 should re-verify #185's and #137's closes independently
   rather than take this retro's word for them, exactly as this retro did
   not take T15.4's #147 close on faith before re-deriving it.
6. **Put D1 and D2 in front of the user again, as their own items, not as
   lines inside a plan document** — third deferral for D1, first for D2.
   Neither escalation's own restriction list permits guessing; neither is
   guessed here. If either answer arrives mid-sprint, each ADR's own trigger
   (tied to the answer, not a sprint boundary) takes over.
7. **When T16 scopes any ticket that reads across a context boundary
   (Payments reading Social Play/Competitions state is now the second
   instance, after T15.5, and won't be the last), require the ticket's own
   dependency check to name the specific field or method that supplies the
   join key** — not just the read method it will call. This generalizes
   recommendation 1 into standing planning practice rather than a one-off
   fix, without building a new mechanical check (per this project's own
   stated preference against pre-emptive tooling — one clause first).

## Sprint-level Definition of Done — scored against what T15's own plan asked

Per `docs/process/t15-sprint-plan.md`'s "Sprint-level Definition of Done,"
three scorings were owed at this retro, stated there so they would not be
improvised:

- **(a) "Did T15.1's reshaped DoD step 5 change the closure outcome, and at
  which moment did the closes actually happen? Report the moment, not just
  the count."** **Reported in full in finding 2.** One close (#147) happened
  at the per-PR moment, correctly, and is the first time this project's
  per-PR close has actually fired since the rule existed (0/9 in T13, 0/6 in
  T14). Two claimed closes (#185, #137) happened at **no** sanctioned moment
  during the sprint at all — not per-PR, not a self-sweep — and were caught
  only by this retro, one ceremony earlier than the amended rule's own
  stated failure condition would have caught them.
- **(b) "T15 takes the Competitions half, so score whether #147 and #168
  actually closed — and score disagreement 2's transfer to #149."** **#147:
  closed, verified.** **#168: did not close** — finding 1, scored in full,
  argued rather than asserted. QA's "permanent furniture" prediction
  (`t14-sprint-plan.md` §A16) is **confirmed for #168**, now four sprints
  open (T13 disclosed it, T14 half-fixed the store side, T15 fixed
  Competitions' half and built Payments' infrastructure without wiring it) —
  the third partial fix in a row. Disagreement 2's transfer to #149: **QA's
  objection is also confirmed there**, more sharply than PdE's premise
  allowed for — T15.5 was supposed to resolve two of #149's five facts and
  resolved zero (finding 5); #149 is now three sprints open with **zero**
  facts closed, not two.
- **(c) "Did any T15 PR contain code written by its reviewer? … the retro
  should verify it from the commit record rather than from the reviews' own
  accounts."** **None** — verified directly against every commit on every
  T15 PR (finding 7), with the one merge-conflict resolution (#191)
  explicitly distinguished from the class ADR-0016 is about, per established
  T14.1/T14.9 precedent.

**Not scoreable by T15 and not pre-empted here:** D1 and D2 remain the
user's (finding 6).

Retro complete. Issue-tracker corrections made during this ceremony:
[#185 closed](https://github.com/nhuthuynh/pickleball-platform/issues/185),
[#137 closed](https://github.com/nhuthuynh/pickleball-platform/issues/137),
[#149 corrected](https://github.com/nhuthuynh/pickleball-platform/issues/149#issuecomment-5302979970).
Open count at ceremony start: 13. Open count now: **11**.

Per `sprint-process.md`'s established convention (a retro PR never updates
the Docs-index row that points at it, since that row must cite this PR's own
merge number, which does not exist until it merges): **`HANDOFF.md` is
deliberately not touched by this PR.** T16's Ceremony 1 corrects T15's row —
including its real PR merge order and the honest-form sentence above — as
its first job, per the standing rule.
