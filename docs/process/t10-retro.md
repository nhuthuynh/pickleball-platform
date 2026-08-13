# T10 Sprint Retro

Ceremony 3 per `docs/process/sprint-process.md`, six-role team (briefs:
`docs/agent-operating-handbook.md` Part B), held against
`docs/process/t10-sprint-plan.md` (including its A7 dispatch-isolation
predictions and A8 recorded disagreements), `HANDOFF.md`'s T10-related
entries, and the real PR/issue history on `nhuthuynh/white-label`
(GitHub-side name `pickleball-platform`) — PRs #99–#109, issues #96–#98.

Every timestamp, merge order, and claim below was pulled from GitHub's own
`merged_at`/`submitted_at`/`created_at` fields and from re-reading each
PR's actual commits and review bodies — not assumed from titles. Two
claims in the record were independently re-verified by actually checking
out the commit in question and re-running the test suite in this
environment, rather than trusting either PR's own prose (see finding 1).
This retro's own five assigned investigation points are not taken as
given either: finding 4 corrects part of the premise it was asked to
investigate (the predicted `main.go` conflict did not, in fact, require
hand resolution — a different, unpredicted collision did).

Findings are not a single voice. Recorded disagreements are left as
disagreements per the "do not manufacture consensus" rule, matching T9's
retro.

**Sprint outcome:** all 8 tickets (37 points, T10.1–T10.8) merged, plus
three out-of-band fix PRs (#104, #107, and a same-PR follow-up commit on
#101) and the Ceremony 1/2 doc PR (#99, carrying ADR-0012). Sprint goal
met: Identity/Users is a real bounded context, Social Play records real
`Match` results, and all three T9 follow-up issues (#96–#98) are
implemented — though, per finding 6 below, not one of them is actually
*closed* on GitHub. Work happened in two real blocks with a roughly
three-day gap between them: 2026-08-10T10:15:17Z–11:02:39Z (PRs #99–#102,
46 minutes) and 2026-08-13T08:09:32Z–14:06:47Z (PRs #103–#109, including
both fix PRs, ~6 hours). This is the first sprint where `go test ./...`
across the *entire* repository is reported green with no red anywhere —
that framing is the backdrop for finding 1, not a contradiction of it: the
branch was genuinely red for part of this sprint, self-corrected, and is
clean now.

---

## 1. The verification-vs-shipped mismatch: fully investigated, and worse on inspection than the one-file version already known

**PE (the mechanism).** During PR #101's (T10.7) merge-conflict
resolution against the already-merged PR #102 (T10.6) — both PRs added a
guard clause to the same line of `CreateOnlinePayment` — the coordinating
session also fixed a fixture-infidelity problem T10.6's new tests had
introduced: `internal/payments/app/service_test.go` used non-UUID
literals (`"entry-1"`, plus stray `"booking-1"`/`"reg-1"`) that T10.7's
new `uuidShape` guard correctly rejects. The merge commit
(`80ae9ae4d`, message: *"Also fixed fixture-infidelity fallout from the
merge... Verified: make test-domain green"*) edited the file, ran
`gofmt -w`, ran `make test-domain` (green — the command reads from disk,
so the edit was live in the working tree), and reported it as verified in
the PR review. What didn't happen: `git add` on the edited file. It had
already been staged with its **pre-fix** content by git's own automatic
merge resolution, and the commit went through without re-staging —
so the pushed, merged commit (`306df838`) silently kept the broken
content. The working-tree verification was real; it just verified a
state that was never the one that got committed.

**QA (exactly how long, and who was exposed).** Verified directly against
`merged_at`, not estimated:

| Event | Timestamp | PR |
|---|---|---|
| Red begins (PR #101 merges, contains the un-staged fixture fix) | 2026-08-10T11:02:38Z | #101 |
| Red ends (fix actually lands) | 2026-08-13T08:15:14Z | #104 |

**Duration: 2 days, 21 hours, 12 minutes, 36 seconds** — not "about three
days," computed to the second from the two `merged_at` values. Session
work was paused for most of that window (the two work blocks above), so
this was not 69 hours of active development running against a red branch
— but it was 69 hours during which a fresh clone, a fresh worktree, or a
new implementer picking up the branch would have hit red immediately.

**Did anyone build on it? Yes — checked, not assumed, and it's worse than
"no work was lost."** PR #103 (T10.3)'s base commit is `306df838` —
**the exact commit PR #101 merged**, confirmed by directly diffing the
PR's recorded `base.sha` against the commit PR #104's own body names as
the point where redness began. This is not a coincidence of git history:
T10.3 was dispatched as part of the same Wave 1 batch as T10.7 (per A7),
and when its session resumed on 2026-08-13, it branched from whatever the
shared branch's tip was — which had been red since two days and 21 hours
earlier. **PR #103's own body claims "`make test-domain` green across the
entire repo (booking/competitions/facilities/payments/socialplay
domain+app), not just this package."** That claim was checked against
reality in this retro, not taken on faith: `306df838` was checked out in
an isolated worktree in this environment and `go test
./internal/.../domain/... ./internal/.../app/...` was run directly against
it. Result: **9 failures in `internal/payments/app`**, all `"payments:
payable id is required"` — the identical count and identical failure mode
PR #104's own verification log independently reports for that same
commit. PR #103's green claim does not hold at the commit it says it was
made against. T10.3's own diff (three files under
`internal/socialplay/domain`) never touched the broken code and shipped
no defect — the mismatch had no product consequence — but it is a second,
independent instance of the exact failure class finding 1 already names:
a verification claim that describes a different state than the one that
was actually checked. The most likely mechanism, consistent with T10.3's
own "rebased before starting any work" note, is that the implementer's
local verification ran against an earlier tip captured before the branch
moved again underneath it, and the claim in the PR body was never
re-checked against the actual base GitHub recorded. (T10.3's separate
in-review finding — that its "rebased onto tip" claim also
mis-described its true merge-base — is a related but distinct
inaccuracy, already caught and corrected by the reviewer at the time; see
PR #103's review. That correction did not extend to auditing the
domain-suite-wide green claim, which is the one this retro caught.)

**What the second verification (PR #104) did differently — the
generalizable half.** PR #104's own log describes three checks: (1)
`git diff --cached` — the **staged** diff, not the working tree; (2) a
full local `make test-domain` run; (3) a completely fresh `git worktree
add` off the **pushed remote branch**, independent of any local state at
all. Step 3 is what actually proves the fix; steps 1–2 are necessary but
not sufficient, since a working-tree-only check is exactly the failure
mode that shipped the bug in the first place. **The lesson, stated
generally: after any merge-conflict resolution that edits files, "tests
pass" is not evidence about what shipped unless it was run against the
staged/committed snapshot or a fresh checkout of the pushed commit — never
against a working tree that might contain uncommitted or unstaged edits.**
This is not a new category of mistake for this project (LESSONS.md's T4
entry is "a single successful run generalized to 'proven'"; this one is
"a correct run generalized to the wrong artifact") but it is a new
*mechanism* within that family, worth naming on its own.

**Recorded disagreement — PE vs. QA on whether the three-way verification
becomes a mandatory step.**
- **QA:** PR #104's three-step verification (staged diff, local suite,
  fresh worktree-off-remote) should be a **required** step for any PR
  whose commits include a merge-conflict resolution, not a one-off
  reaction to this specific incident. The failure mode — edit, verify
  working tree, forget to stage — is mechanically easy to repeat and had
  no guardrail before this.
- **PE:** mandating a fresh-worktree check on every conflict-touching PR
  is real overhead (a second `go`/`npm` toolchain cold-start) for a
  mistake that, on the evidence of this sprint, self-corrected within one
  cycle and had zero product impact both times it occurred (T10.3's
  instance shipped no defect either). A cheaper version — `git diff
  --cached` before any commit that follows a manual edit post-merge — de-
  risks the actual failure (unstaged content) at near-zero cost, without
  requiring the heavier fresh-worktree step every time.
- **Unresolved**, recorded rather than smoothed. Both agree `git diff
  --cached` (or equivalent) after any post-merge manual edit is the
  cheap, adopted floor; the fresh-worktree step's mandatory-vs-optional
  status is not settled.

**Adopted for T11 (uncontested part):** any commit that follows a manual
edit made during merge-conflict resolution must be checked with `git diff
--cached` (or `git show --stat`/`git log -p -1` on the resulting commit)
before pushing — verifying the working tree alone is not sufficient
evidence of what will actually ship.

## 2. A second, independent fixture-infidelity instance this sprint, invisible to `make test-domain` by construction — and a third+ instance of a bug class LESSONS.md already named

**QA (the facts, verified against the actual commit history).**
`internal/payments/adapter/grpcapi`'s own test fixtures
(`authz_regression_test.go`, `competition_entry_authz_regression_test.go`)
had the identical non-UUID-literal problem as finding 1's file — but in a
different file, in a different package layer (`adapter/grpcapi`, not
`app`). Both PR #105 (T10.4, merged 08:48:14) and PR #106 (T10.2, merged
13:22:10) ran their full local suites, found these failures, correctly
identified them as pre-existing and out of their own ticket's scope
(neither ticket touches `internal/payments`), and **disclosed them by
name in their own PR bodies** — PR #105: *"Disclosed, not fixed...
`internal/payments/adapter/grpcapi` has an identical pre-existing
`uuidShape`/fixture-ID break... Left for whoever owns that
context/ticket to fix."* PR #106's review (submitted 13:22:04) found the
same 7 failures independently, by running the **full** `go test ./...`
rather than `make test-domain` — which does not build or run
`internal/*/adapter/**` at all (`Makefile`: `test-domain` is scoped to
`./internal/.../domain/... ./internal/.../app/...`). The coordinator
opened PR #107 three minutes after that review (created 13:25:06, merged
13:25:25) and fixed both files with one shared UUID-shaped fixture
constant block.

**PE (the gate, checked against what it actually covers).**
`make test-domain` is deliberately scoped to the framework-free
domain/app layers (CLAUDE.md's own gotcha section says so, and for good
reason — it's the fast, dependency-free bar every ticket must clear).
That scoping is also, mechanically, why this exact bug class can survive
inside it indefinitely: a non-UUID literal in an `adapter/grpcapi` test
fixture will never fail `make test-domain`, no matter how long it sits
there, because that command never touches the package it lives in.
`sprint-process.md`'s per-ticket DoD already says `go test`/`make test`
should run "where the ticket touches adapter/infra code" — and T10.2/
T10.4/T10.7 *did* touch adapter code and *did* run the full suite, which
is exactly how both implementers found this. The gap isn't in what ran;
it's that a regression **found and correctly disclosed as out-of-scope**
had no owner until a reviewer happened to hit it again during an
unrelated review. Two implementers doing the right thing (disclose,
don't silently expand scope) still left the bug live for hours.

**QA (the count, corrected).** The sprint's own framing describes this as
"the third time" this project has hit non-idgen-shaped fixtures, after
PR #89's root cause and T9.5's vacuous test. Checked against the actual
record: that undercounts specifically for T10. This sprint alone produced
**two** separate instances of the identical bug class in two different
files of the same package (`internal/payments/app/service_test.go` via
finding 1, and `internal/payments/adapter/grpcapi`'s two files via this
finding) — both are non-idgen-shaped `PayableID` literals rejected by the
same `uuidShape` guard, both in the same context, introduced within the
same 46-hour window. Counting PR #89 and T9.5 as the first two, this
sprint's two instances make it the third *and* fourth. The project has
now hit this bug class at least four times across three sprints (T9,
T10×2) without a structural fix landing — every instance so far has been
caught by a human/reviewer noticing failing tests, not by anything that
prevents the fixture from being written in the first place.

**Recommendation, adopted:** two structural changes, not just another
disclosure norm:
1. A shared, UUID-shaped fixture-ID helper per context (PR #107 already
   did this for `payments`' two `grpcapi` test files sharing one package —
   generalize the pattern: one `fixture*ID` constant block per package,
   reused by every test file in it, rather than each file minting its own
   literals).
2. When a ticket's own review disclosures name a pre-existing,
   out-of-scope regression (as both PR #105 and PR #106 correctly did),
   that disclosure should produce a tracked follow-up (an issue, at
   minimum a `HANDOFF.md` bullet) at the moment it's found — not rely on
   a future reviewer re-discovering it by chance while doing something
   else. This mirrors finding 5 of T9's own retro (untracked follow-ups
   living only in prose) applied to test regressions specifically instead
   of product gaps.

## 3. T10.2's `CreateUser` shipped two real security-shaped gaps the implementer's own thorough verification didn't surface — because they're a new shape of problem for this project

**PE.** The implementer's own first-round verification of PR #106 was
genuinely strong by this project's normal bar: real Postgres migration
testing (not just sqlc's DDL parsing — the PR explicitly demonstrates
sqlc does *not* catch a broken function name inside a CHECK expression,
then verifies past that gap by hand via `psql`), adversarial CHECK-
constraint testing (duplicate PK, out-of-range level, unrecognized role,
empty roles array, all exercised against a real Postgres 16 instance).
None of that surfaced either finding. Both — permanent, unauthenticated
`User.ID` squatting (a caller can permanently claim a UUID a real identity
will later be minted with, denying that person their own account forever)
and unrestricted role self-assignment on `CreateUser` (any caller could
originally request `RolePlatformAdmin`) — were found in the PR review
round, not self-disclosed. Confirmed via the actual commit history:
commit `18930b1` ("T10.2 PR review fixes...") is the fix for both,
titled as addressing "PR #106 review findings," and no round-1 review
object is visible via GitHub's API for this PR (only the round-2 APPROVE
review is — round 1's findings survive only in the fix commit's own
message and the PR body's "Update" section, which is itself a minor
traceability gap worth naming: this project's convention of leading a
review with a bolded verdict, per T9 retro finding 6, depends on the
review actually being a discoverable GitHub review object).

**QA.** The gap is structural, not a lapse in effort. Every prior
`actor_user_id`-shaped check in this codebase (T5.5 Social Play, T6.3/
T6.7→T8.5 Payments, T7.7 Facilities — `HANDOFF.md`'s own Cross-cutting
section calls this "a three-times-repeated pattern") gates a **mutation
on an object that already exists**: a false claim is rejected and leaves
no trace. `CreateUser` is this project's first public *creation* RPC
where the caller-supplied value becomes the row's own permanent primary
key. The established adversarial habit — "does the claimed actor match
the resource's real owner?" — has no owner to check against on a create
path, so it doesn't fire, and nothing in this project's existing review
muscle memory asks the different question a creation endpoint needs:
*what can an anonymous caller permanently claim or elevate by calling
this at all, with no prior state to be wrong about?*

**BA (the checklist gap this warrants).** This project has strong,
repeated-until-automatic coverage for the mutation-on-existing-object
shape (four contexts now, same caveat restated and cross-referenced each
time). It has zero prior instances of a creation-RPC-specific check,
because `CreateUser` is the first creation RPC in this codebase whose
caller-supplied identifier persists as a permanent key rather than being
generated server-side. **Recommended new checklist item, adopted, for
any future creation RPC specifically** (distinct from the existing
mutation-on-existing-object item):
1. Can an anonymous/unauthenticated caller choose the resource's own
   permanent identifier? If so, can they permanently occupy an
   identifier a legitimate future caller will need?
2. Can an anonymous caller self-assign any field that gates authorization
   or privilege elsewhere in the system (a role, an admin flag, an
   elevated tier) at creation time, given there's no pre-existing owner
   fact to check the request against?

Both gaps are already disclosed in detail in `HANDOFF.md`'s Cross-cutting
section (the ID-squatting bullet in particular is a thorough, named
disclosure with a concrete closure condition) — not restated here. This
finding's contribution is the process angle: the checklist item above,
not the disclosure itself.

## 4. The sprint plan's predictions: one confirmed exactly, one that predicted the wrong file, and a genuinely new gap in the dispatch-isolation checklist

**PdE (T10.5/T10.8 — confirmed accurate).** A7 flagged T10.5 and T10.8 as
a likely Vue-component collision. It materialized exactly as predicted:
PR #109's own review describes "a real add/add on `identityClient.ts`"
(both sides independently wrote near-identical code) "plus two
independent-and-additive import conflicts in `CompetitionManage.vue`/its
spec file" — resolved on the source branch (T10.5, merging second, per
the plan's own stated rule), both sides' additions kept. Correctly
predicted, correctly priced, correctly handled — credit due exactly as
T9's retro gave it for the T9.6/T9.7 conflict.

**QA (T10.2/T10.4 — checked, and the prediction was wrong about *which*
file needed hands-on resolution).** A7 named `cmd/server/main.go` as the
conflict risk between T10.2 and T10.4. Checked against what actually
happened: PR #106's review states the merge was a *"clean auto-merge,
including the `cmd/server/main.go` conflict both tickets were pre-flagged
to touch (both additions were in disjoint regions)"* — git's line-based
merge resolved it with **zero hand intervention**. The file both tickets
really did collide on, requiring a real, human-authored fix, was **never
named in A7 at all**: both T10.2 and T10.4 independently claimed
migration number `0015` (T10.4's PR #105 says so explicitly — *"at
authoring time T10.2... had not pushed a branch with a migrations change
to check against, so no collision could be confirmed either way"* — and
T10.2's own first commit used `0015_identity.sql` too, per its commit
message, before being renamed to `0016` in the review-fix round). A7's
dispatch table checks file-level overlap (does ticket X touch a file
ticket Y also touches?) but has no concept of a **shared numeric-sequence
namespace** — two tickets can touch zero of the same files and still
collide, because a migration's sequence number is a project-wide resource
neither ticket's own diff makes visible to the other. This is not the
first time: `HANDOFF.md`'s T6.5 entry already records the identical
collision shape (`0005_payments.sql` vs. `0005_socialplay.sql`, T6.1 vs.
T6.6) and left it "worth renumbering... not urgent" rather than turning
it into a checklist item. This is the **second** occurrence of the same
collision class, four sprints later, still uncaught by planning.

**Also worth recording, minor but real:** A7's Wave 1 named four tickets
(T10.1, T10.3, T10.6, T10.7) intended to run "up to 4 implementers in
parallel." In practice, three (T10.1, T10.6, T10.7) landed together
within the first 46-minute block (2026-08-10T10:20–11:02); T10.3 didn't
land until the second work block, nearly three days later
(2026-08-13T08:16). Not a process failure — session timing, not
dispatch — but worth noting so "Wave 1 ran as planned" isn't assumed from
the sprint plan's intent without checking the actual merge timestamps,
which is exactly this retro's own standing rule.

**Change for T11, adopted:** extend A7's dispatch-isolation checklist
question beyond file-level overlap to explicitly ask, for any two tickets
dispatched in the same or adjacent waves: *does either ticket add a
migration file, and if so, have they coordinated on the next sequence
number* — the same "shared namespace, not shared file" gap this finding
names for migrations. **PdE vs. QA disagree on how far to generalize
this**, recorded below rather than resolved.

**Recorded disagreement — PdE vs. QA on scope of the fix.**
- **QA:** the checklist item should name migration numbers *and* any
  other shared, project-wide sequence a ticket might independently claim
  — proto field numbers being the obvious next candidate, since two
  parallel tickets editing the same `.proto` file's message could
  plausibly pick the same next field number the same way two tickets
  picked the same next migration number, twice now.
- **PdE:** proto field-number collisions haven't happened yet in five
  sprints of parallel dispatch, and speculatively covering every
  numeric-sequence resource this project *might* have is exactly the
  "abstract on a guess" pattern this project's own convention (cited in
  PR #108's review) argues against elsewhere. Fix the demonstrated
  problem — migrations, now twice — and revisit if a third shared-
  namespace class actually collides.
- **Unresolved.** The migration-number item is adopted either way; only
  its generalization to other namespaces is contested.

## 5. Two Wave-3 tickets, two real findings each, one round of fixes each — genuine misses, and they point at two different, nameable patterns

**Designer/QA (T10.8 — verified against the actual fix commits, not just
PR #108's own description).** A request-sequencing race: `DisplayName.vue`/
`VenueName.vue` are never remounted when the id they resolve changes,
so a slower, earlier lookup could resolve after a faster, later one and
silently overwrite a correct name with a stale one. Fixed with a
generation-counter guard in `useDisplayName`/`useFacilityName`, verified
non-vacuous per the PR #108 review (guard disabled, new tests confirmed
failing, restored, confirmed green). Separately, both components render
async-changing text with no `aria-live` announcement, breaking this
project's own established convention (`GameJoinPanel.vue`,
`GameCheckout.vue`, `CompetitionCheckout.vue`, `CourtBookingFlow.vue` all
carry `aria-live="polite"`) — an accessibility regression against a
pattern this codebase had already normalized elsewhere, missed on first
pass of a new pair of components implementing that same pattern.

**Designer/QA (T10.5 — same verification standard).** The gender-mix-
absence check (required by ADR-0012's "no matching UI control of any
kind" instruction, tested across four screens) initially checked only
native `<input>`/`<select>` `id`/`name` attributes matching `/gender/i`.
Real, but narrow: it would silently miss a control identified only by
`aria-label` (a shape this codebase already ships elsewhere —
`RoleIndicator.vue`'s `<span>`), a `<label>`-associated control, or a
non-native ARIA `role="radiogroup"`/`"radio"` widget. Fixed by
centralizing all four screens' checks into one shared
`findGenderControls()` helper checking all four signal types, verified
non-vacuous (a throwaway `aria-label`-only `<select>` was injected,
confirmed to fail the strengthened check, then removed).

**QA (the pattern, stated once instead of twice).** These are not two
unrelated one-off misses; they're two instances of the same underlying
shape this project has now named once for backend regression tests
(T9 retro finding 3: "a fake that cannot produce the failure mode makes
the test a no-op") and is seeing for the first time on the frontend in
two different guises this sprint:
- **T10.8's race** is "async state without request-sequencing" — a new
  failure shape for this project (every prior race this codebase has
  found and fixed was a database-concurrency race — T4, T5.4, T6.6 — this
  is the first one found in client-side async state). Not yet claimed as
  a recurring pattern (one instance so far), but worth watching for on
  any future component that resolves a network call keyed to a
  changeable prop.
- **T10.5's narrow check** is a frontend-specific instance of the same
  "absence assertion checks syntax, not semantics" trap the backend
  fixture-fidelity findings (1, 2 above) already represent for ID shapes:
  checking `id`/`name` alone proves a control isn't present *in the one
  shape checked for*, not that the control is absent under every way
  a future implementation could satisfy the same UI requirement.

**Change for T11, adopted:** when a ticket's AC requires asserting a UI
control's *absence* (this project's established "no matching UI control"
discipline, per T8.8/T9.6/T9.7/T10.5), the check must cover semantic
signals — `aria-label`, `<label>` association, ARIA `role` — not just
native `id`/`name`, using `findGenderControls()`'s shape as the reusable
reference implementation rather than each screen re-deriving its own
narrower version.

## 6. GitHub issues are never actually closing on this branch topology — not new to T10, verified project-wide, and directly contradicts the sprint-process DoD

**PO (found while checking finding 4's "closes #96" claims against
reality, not part of the assigned investigation list).** All three T9
follow-up issues opened at this sprint's Ceremony 1 — #96 (T10.6), #97
(T10.7), #98 (T10.8) — remain **OPEN** on GitHub, despite each one's
implementing PR (`#102`, `#101`, `#108` respectively) carrying an explicit
"closes #N"/"Closes #98" in its title or body and being merged. Checked
directly against each issue's own `closed_by_pull_requests` field:
`total_count: 0` on all three — GitHub never linked them at all, not
"linked but still shows open." **This is not new to T10 and not specific
to these three issues**: spot-checked issue #70 (T9.1, "Add the
Competition and CompetitionEntry domain model"), whose implementing PR
(#84) merged in T9 and is recorded as done in `HANDOFF.md` — also still
OPEN, also `closed_by_pull_requests: {"total_count": 0}`. The root cause
is structural: GitHub's `Closes #N` auto-close only fires for a PR merged
into the repository's **default branch**; every PR in this project's
history merges into `claude/go-backend-pickleball-7up34j`, which is not
the default branch (`main` is). `sprint-process.md`'s per-ticket
Definition of Done explicitly requires, as its own step 5, "the GitHub
issue is linked to the merged PR and closed" — and this has silently not
been true for any ticket, in any sprint, since the process was adopted at
T5.

**BA.** Not a correctness bug in any shipped code, and not a new
discovery about T10 specifically — but it is a real, verified gap in the
one thing `sprint-process.md` names as the board of record ("Board of
record: GitHub Issues + labels"), and it means every past retro's and
sprint plan's claim that a ticket's issue "closes" on merge has been
false for the entire life of this process, unnoticed because nobody
checked the issue list against the merge list until this retro did.

**Change for T11, adopted:** since GitHub's automatic mechanism
structurally cannot fire on this branch topology, closing the issue must
become an explicit manual step in the per-ticket merge flow (a follow-up
`issue_write` close call, or equivalent), not an assumption carried by
writing "closes #N" in the PR body. Cheap to add; worth doing retroactively
for #70–#98 as a small chore, not just going forward — flagged for T11
Ceremony 1 to open as its own ticket rather than fixed unilaterally here
per CLAUDE.md rule 9.

---

## No finding on

**PM had limited independent material this sprint**, structurally similar
to T9's retro note on the same point: T10's scope and the ADR-0012
exit-(b) decision were both settled in Ceremony 1 (A0–A2), and the
sprint's only live product tension (A8, does T10 ship enough
user-visible value) was already recorded as a disagreement in the sprint
plan itself rather than surfacing fresh in execution. PM's material
contribution to this retro is embedded in finding 6's framing (a process
gap affecting every sprint's own claimed Definition of Done, which is
PM's document to care about as much as QA's).

**No new finding on dispatch isolation itself** (T9 retro finding 1's
subject) — every Wave 1–3 ticket this sprint ran in an isolated worktree
per A7, and no shared-unisolated-checkout collision recurred. The
migration-number and `main.go` questions in finding 4 are a different
failure class (a shared *namespace*, not a shared *working directory*)
and are treated as their own finding rather than folded into "isolation
worked/didn't."
