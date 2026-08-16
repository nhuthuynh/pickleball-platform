# T28 Sprint Retro

Ceremony 3 per `docs/process/sprint-process.md`, six-role team (briefs:
`docs/agent-operating-handbook.md` Part B), held against
`docs/process/t28-sprint-plan.md` (§A0–§B9, the first non-zero sprint since
T19), `docs/process/t27-retro.md`/`docs/process/t26-retro.md` as the
structure/tone/rigor precedent, `HANDOFF.md`, and the real PR/issue history
on `nhuthuynh/white-label` (GitHub-side name `pickleball-platform`) — PR #234
(Ceremony 1/2 doc), PR #235 (T28.1, "partial fix for #164: Payments identity
conformance"), one new ADR (`docs/adr/0017-*.md`), zero new issues, one
narrowing comment on #164.

**This retro departs from T20–T27's routine confirm-and-report shape**,
because T28 shipped a real ticket touching a live authorization comparison.
Per the task brief, every factual claim below — PR merge SHAs, ADR section
contents, issue states, which mutation checks actually executed where — was
re-verified against the live repository and, where practical, independently
reproduced from scratch, not re-read from T28's plan or PR #235's own
account of itself (CLAUDE.md rule 10; this project's own standing
convention after several prior retros caught and corrected exactly this
class of self-reported claim).

**Verification performed before writing a single finding.** `git fetch`
confirmed the local tip matches `origin/claude/go-backend-pickleball-7up34j`
exactly at `c975f219ea5571c863453021ae277961abf72218` (`c975f21`), the squash
merge of PR #235, with no descendant commits. A separate `git worktree`
(detached at `c975f21`, removed after use) was used for the mutation checks
below so nothing here touches the shared checkout's working tree. `make
generate` was re-run before any Go toolchain command, since `internal/gen`
is gitignored and this session's pre-existing copy predated T28.1's schema
change — `go vet -tags=integration ./...` failed against the stale
generated code and went clean immediately after regenerating, which is
disclosed here as a fact about this environment's setup, not a finding
about the PR.

**Sprint outcome, stated before the findings that qualify it.** T28 shipped
one ticket, T28.1, 8 points — PR #235, merged `2026-08-16T13:59:00Z`,
squash sha `c975f219ea5571c863453021ae277961abf72218`, 1 commit, 21 files,
+2067/−74. It ends the eight-sprint run of 0-ticket sprints (T20 through
T27) on the same #164 re-examination T28's own Ceremony 1 performed, and it
is the first sprint since T19 with a real PR for DECISION D2's interim rule
to actually be exercised against (T28 plan §A7).

**What this retro found, in one sentence, so the findings do not have to be
read before the headline is known:** T28.1 shipped the funnel change and
the backfill/column-type migration in the same single commit with no window
where the authorization comparison could silently break, the backfill was
mutation-checked against both the orphaned and resolvable cases by three
independent means (the committed test's design, the reviewing session's own
DB reproduction, and this retro's own separate third reproduction below —
all agreeing), the other 7 backlog issues' blockers were re-verified live
and hold unchanged, #164 was correctly narrowed (not closed) by an explicit
comment actually posted 11 seconds after merge, D1 and D2 both remain
formally unanswered, and this sprint's real PR review lands in D2's
"exercised, no fix needed" shape — the sixth such instance, the first since
T19. One process-hygiene observation is recorded (§5) and it is not scored
as an incident.

---

## 1. The merged-fix issue sweep

**PO.** Per `sprint-process.md`'s DoD, this retro is sweep moment 1; T29's
Ceremony 1 remains the authoritative moment regardless of this result.

**Step 1 — list the open issues, live, at this retro's start.**
`list_issues(state: OPEN)` → **`totalCount: 8`**: #124, #126, #130, #134,
#144, #145, #149, #164 — the identical eight numbers every sweep has found
since T15, #164 included (narrowed, not closed).

**Step 2 — reconcile arithmetically, before reading any issue
individually.** `open_at_end_of_T27's_retro (8) − closed_during_T28 (0) +
opened_during_T28 (0) = 8`. T28 closed nothing (#164 stays open by design —
a partial fix, per T28.1's own PR title and instruction 9) and opened
nothing (zero new issues filed this sprint). Matches the live `totalCount:
8` exactly.

**Step 3 — cross-reference merged PRs against the open list.** Two PRs
merged this sprint: **#234** (Ceremony 1/2 doc, `merged_at:
2026-08-16T13:19:47Z`, titled "…reclassifies #164 as unblocked, takes
Payments conformance") and **#235** (T28.1, `merged_at:
2026-08-16T13:59:00Z`, titled "partial fix for #164: …"). Both name #164 in
their titles; neither claims to close it — #234 states a reclassification,
#235 states "partial fix." Cross-referenced against the open list: #164 is
correctly still open. No other issue number appears in either PR's title or
body as a claimed closure.

**Sweep result: clean.** The one issue touched this sprint (#164) was
correctly left open with its narrowing written both in the PR and, per the
next finding, on the issue itself. **T29's Ceremony 1 still re-runs this
sweep in full**, per the standing rule that a prior ceremony's clean result
does not discharge the next one.

## 2. Point 1 — did T28.1 ship the funnel change and the backfill/migration
together as one reviewed unit, with no window where `authorizeOnlineConfirmation`
could silently break?

**PE, verified directly against the merged commit rather than PR #235's own
"why one PR" section.**

**Single commit, confirmed.** `git show --stat c975f219ea5571c863453021ae277961abf72218`
shows exactly one commit touching all 21 files — there is no sequence of
commits where the funnel change could have landed on the shared branch
ahead of the migration, or vice versa. The prior merge on this branch
(`1176829`, T28's own Ceremony 1/2 doc) touched no Payments code at all
(docs only), so there is no earlier partial state to have exposed either.

**The funnel change, read directly.** `internal/payments/adapter/grpcapi/handler.go`'s
diff: `actor(ctx)` becomes `func (h *Handler) actor(ctx context.Context) (string, error)`,
which calls `h.svc.ResolveActorUserID(ctx, subject)` — a `*Handler` method
resolving through the new port, exactly as claimed and exactly matching
Booking's/Facilities' T13.2/T13.3 shape.

**The migration, read in full.** `db/migrations/0024_payments_recorded_by_user_id_uuid.sql`
adds a nullable `recorded_by_user_id_uuid` column, backfills it via a single
`UPDATE … FROM identity_users` join keyed on `subject`, drops the old `text`
column, and renames the new one into place — one migration, no
expand/contract split, with the file's own header comment explaining why
(this prototype's migrations apply only against a fresh volume, so there is
no live deployment for a two-phase split to protect). The header comment
states the hazard in the same words as the PR body: *"Landing this
migration without the funnel change (or vice versa) would silently compare
a uuid against a stale subject string for every request in the gap."*

**The comparison itself, read directly at `internal/payments/app/service.go`.**
`authorizeOnlineConfirmation(p domain.Payment, actorUserID string)`'s body
(`actorUserID == "" || p.RecordedByUserID == "" || actorUserID !=
p.RecordedByUserID`) is **unchanged** by this diff — only its doc comment
grew. This is the correct shape for a fix of this kind: the defect was
never in the comparison's logic, it was in the two sides of the comparison
belonging to different identifier spaces (a resolved `User.ID` vs. a raw
subject) before this PR. Fixing what feeds both sides, in one commit,
rather than touching the comparison, is what actually closes the hazard.

**Wiring, read directly at `cmd/server/main.go`.** `paymentsIdentityLookup
:= paymentsidentity.NewLookup(identitySvc)` reuses the same `identitySvc`
instance Booking's and Facilities' own lookups are built from immediately
above it — not a second stack — and is passed into `paymentsapp.ServiceOptions{
…, Identity: paymentsIdentityLookup }` in the same commit.

**Conclusion: confirmed, independently, against the actual merged diff —
not merely restated from the PR body's own "why one PR" section.** The
funnel resolution and the column/backfill that its comparison depends on
landed in exactly one commit, on exactly one PR, with no intermediate state
ever reachable on the shared branch. No window existed.

## 3. Point 2 — was the backfill mutation-checked per CLAUDE.md rule 10?

**QA, with three independent layers distinguished rather than collapsed
into one restated claim.**

**Layer 1 — the committed test, read in full.**
`internal/payments/adapter/postgres/migration_backfill_integration_test.go`
(`-tags=integration`) seeds one resolvable payments row (subject matching a
seeded `identity_users` row) and one orphaned row (subject matching
nobody), applies migrations 0001–0023 then only 0024, and asserts by exact
value: the resolvable row's new column equals the seeded `identity_users.id`
precisely (not merely non-null); the orphan row's column is `NULL` via
`pgtype.UUID.Valid`, not an empty-string check; the row **count** is still
2 (guards against an orphan being silently `DELETE`d rather than nulled);
and `information_schema.columns.data_type = 'uuid'` (guards against a
regression that reverted the `RENAME COLUMN` step while leaving the
backfill logic in place). This is a real mutation-check design, not an
assertion that the migration merely ran without erroring.

**Layer 1's precise status, stated exactly rather than glossed over: this
committed test has never itself executed in this environment.** Docker
image pulls (`postgres:16-alpine`) are network-policy-blocked in the
sandbox both T28.1's implementer and this retro ran in — confirmed
independently: `docker info` succeeds (the daemon runs) but the same
registry-pull block PR #235's own body discloses applies here too. The test
was **compile-checked only**, via `make vet-integration` (re-run this retro
after regenerating `internal/gen` — clean, no errors). This is the same
"manually proven, CI-unexecuted" status `HANDOFF.md` already has named
vocabulary for (T19.2), and it is the accurate description here: the
committed test's *design* is sound and independently reproducible (next
paragraph), but its own execution record does not yet exist in this
environment.

**Layer 2 — the reviewing session's own DB-level reproduction, as described
in its PR #235 review** (`pull_request_review_write`, `id 4946345853`,
`submitted_at: 2026-08-16T13:58:54Z`, six seconds before merge): a real
local Postgres, throwaway database, migrations 0001–0023, one hand-seeded
resolvable row and one hand-seeded orphan row, migration 0024 applied
directly via `psql`. Reported result: resolvable → correct uuid, orphan →
`NULL`, row count preserved, column type `uuid`, live FK constraint. This
review is real (fetched from the API, not assumed) and its account is
internally consistent with what the merged diff actually contains.

**Layer 3 — this retro's own separate, third reproduction, run independently
rather than trusted from either of the first two.** A local Postgres 16
system service was already running in this environment (`pg_lsclusters`
confirms it, independent of and not started by this retro). A fresh
throwaway database (`t28_retro_check`) was created; migrations
`0001`–`0023` were applied in order via `psql -v ON_ERROR_STOP=1`, all
clean; one `identity_users` row and two `payments` rows were seeded with
**this retro's own distinct UUIDs**, not reused from the committed test's
fixtures or the reviewing session's account of its own seed — one
`payments` row's `recorded_by_user_id` matching the seeded subject, one
matching nobody; migration `0024` was applied directly. Result, read back
by raw SQL:

| Assertion | Result |
|---|---|
| Resolvable row's `recorded_by_user_id` | `99999999-9999-9999-9999-999999999999` — the exact seeded `identity_users.id`, not merely non-null |
| Orphan row's `recorded_by_user_id` | `NULL` |
| Row count for the two seeded rows | `2` (neither dropped) |
| `information_schema.columns.data_type` for `payments.recorded_by_user_id` | `uuid` |
| Foreign key | `payments_recorded_by_user_id_uuid_fkey FOREIGN KEY (recorded_by_user_id) REFERENCES identity_users(id)`, present and live |

The throwaway database was dropped immediately after. This is a genuinely
independent reproduction — different session, different seed values, same
migration file read directly off the merged commit — and it agrees exactly
with both Layer 1's test design and Layer 2's reported result.

**Layer 4 — the Go-level fail-closed mutation, independently re-performed
this retro rather than taken on the review's word.** In an isolated,
disposable `git worktree` detached at `c975f21` (removed after use):
`internal/payments/app/service.go`'s `ResolveActorUserID` was mutated so
the nil-`Identity` branch returns `subject, nil` (fail **open**) instead of
`"", domain.ErrUserNotFound` (fail **closed**). Re-running the two targeted
tests produced exactly the claimed result:

```
--- PASS: TestResolveActorUserID_DelegatesToTheIdentityPort (0.00s)
--- FAIL: TestResolveActorUserID_NilIdentityFailsClosed (0.00s)
    resolve_actor_test.go:73: ResolveActorUserID with a nil Identity = <nil>,
    want ErrUserNotFound (fail closed, mirroring RegistrationLookup/
    GameLookup/GameAdminReader/EntryLookup/CompetitionAdminReader's
    existing nil-safe convention)
```

The mutation was then reverted (`git checkout --`), confirmed an empty
diff, and `go test ./internal/payments/app/... -race -count=1` passed
clean again.

**Point 2, scored: yes — mutation-checked, by four layers, not one
restated claim.** Both the orphaned and the resolvable cases were
exercised at the database level (Layers 2 and 3, agreeing) and the
fail-closed nil-identity guard was exercised at the Go level (Layer 4, this
retro's own independent re-derivation, not merely a re-read of the
committed test's existence). Layer 1's committed test is sound by design
and compile-clean but has not itself executed in either the implementing
or this retro's environment — stated precisely rather than folded into a
blanket "the tests ran" claim.

## 4. Point 3 — do the other 7 backlog issues' blockers still hold,
re-verified live?

**QA and BA, against freshly fetched fields, not re-read from the plan's
own table.**

| Issue | `updated_at`/comments at T27 retro & T28 plan §A3 | `updated_at`/comments now (live, this retro) | Changed? |
|---|---|---|---|
| #144 (D1) | `2026-08-15T07:01:03Z`, 1 comment | `2026-08-15T07:01:03Z`, 1 comment | No |
| #149 | `2026-08-15T16:56:58Z`, 3 comments | `2026-08-15T16:56:58Z`, 3 comments | No |
| #124 | `2026-08-15T16:25:34Z`, 1 comment | `2026-08-15T16:25:34Z`, 1 comment | No |
| #145 | `2026-08-15T05:01:29Z`, 1 comment | `2026-08-15T05:01:29Z`, 1 comment | No |
| #126 | `2026-08-14T16:12:26Z`, 0 comments | `2026-08-14T16:12:26Z`, 0 comments | No |
| #130 | `2026-08-14T16:30:25Z`, 0 comments | `2026-08-14T16:30:25Z`, 0 comments | No |
| #134 | `2026-08-14T16:37:49Z`, 0 comments | `2026-08-14T16:37:49Z`, 0 comments | No |

Every field matches T27 retro's and T28 plan §A3's own live-fetched table
exactly, and every timestamp predates PR #234's merge (`13:19:47Z`) — none
of the 7 routine issues was touched by T28's own work, which is the correct
shape: T28.1 only ever wrote to Payments' own column and to #164.

**#164 itself, live at this retro's start:** `updated_at:
2026-08-16T13:59:11Z`, **2** comments (T15's original classification
comment, unchanged, plus the new narrowing comment — see finding 5). Open,
correctly.

**D1 (#144) and D2, named individually per the task's instructions.** #144's
comment re-fetched via `issue_read(get_comments)` — exactly one comment,
T14.3's original escalation (`id 5301056598`, `created_at
2026-08-15T07:01:03Z`), text identical word-for-word to every prior
retro's quotation. ADR-0015's and ADR-0016's `## Status` sections re-read in
full this retro — see finding 6.

**Point 3, scored: yes, all 7 issues' blockers hold, re-verified live.**

## 5. #164 — narrowed correctly, and the comment was actually posted, not
merely stated as intent

**PO, per the "closes #N" vs. "partial fix for #N" review-honesty
distinction generalized in `sprint-process.md`, applied here for the first
time to a partial-fix PR rather than a full close (T28 plan's DoD item 3).**

PR #235's title reads exactly `partial fix for #164: Payments identity
conformance (recorded_by_user_id → uuid, ADR-0017)` — confirmed via the API,
not assumed. Instruction 9 required posting the narrowing as a comment on
#164 "at merge time (or state explicitly why not)." `issue_read(get_comments,
#164)` returns exactly that comment (`id 5307800989`, `created_at:
2026-08-16T13:59:11Z`), posted **11 seconds after** PR #235's own
`merged_at` (`13:59:00Z`). It correctly: names PR #235 by number and title,
states which column is now conformant, states which two contexts remain
non-conformant and that they are deferred to T29 (not silently dropped),
and states the issue stays open. This is the comment actually landing, not
an intention recorded and left undischarged — the same gap this project's
own DoD step 5 measured at 0/9 and 0/6 in T13/T14 before being demoted to
an optional-early-close with the sweep as backstop. Here it was performed.

**One process-hygiene observation, not scored as an incident.** T28.1's PR
body discloses a real, out-of-scope interaction with **#149**: once the
funnel resolves `actorUserID` to a `User.ID`, a legitimate
`booking`-payable caller must now send their own resolved `User.ID` as
`booking_host_id` to match — previously a subject would have sufficed,
since both sides were subjects before this PR. This is new, accurate
information about the shape of the one caller-supplied fact #149 already
names as its residual scope (confirmed: #149's own most recent comment,
`16:56:58Z`, T16.2, already states `booking_host_id` is "the one fact that
remains caller-supplied and forgeable," blocked on D1). **Checked against
the standing "post a correction to the issue itself" rule
(`sprint-process.md`, Label taxonomy section) and found not to require
action**: that rule triggers when a PR's findings *falsify* an earlier
claim an issue carries, and #149's own claim (caller-supplied, forgeable,
blocked on D1) remains true after T28.1 — nothing on #149 became false.
But a future reader of #149 who does not also read PR #235's body will not
learn that the *shape* of the caller-supplied value #149 is about just
changed (subject-shaped before T28.1, `User.ID`-shaped after). This is
disclosed here for the record and left as a **recommendation** for T29's
Ceremony 1 (§7 below) rather than escalated as a finding against T28.1,
since nothing in `sprint-process.md`'s existing rules obligates the comment
in this exact case.

## 6. Point 4 — did D1 or D2 get answered mid-sprint, as a formal ADR
decision?

**PE and PO, jointly, checked directly against both ADRs' own sections —
including establishing ADR-0017's correct citation form for the first time,
since it did not exist at any prior retro.**

- **ADR-0015 (`docs/adr/0015-*.md`), `## Status` heading, read in full this
  retro.** Unchanged: **"Escalated — awaiting product decision. This ADR
  decides nothing."** No option is marked chosen. The file's only ancestor
  since T27 retro is `1176829`/`c975f21`, neither of which touches
  `docs/adr/0015-*.md` (confirmed: not in PR #234's or PR #235's changed
  files).
- **ADR-0016 (`docs/adr/0016-*.md`), `## Status` heading, read in full this
  retro.** Unchanged: **"Escalated — awaiting the user's decision. This ADR
  decides nothing."** Same restriction list, same wording, word for word.
- **ADR-0017 (`docs/adr/0017-*.md`), read in full this retro — a genuinely
  new citation, established correctly the first time.** Its frontmatter
  carries `- **Status:** Accepted`, and — unlike ADR-0015/ADR-0016 — **it
  has no separate `## Status`-headed section at all**: the document's
  headings run Context → Decision 1 → Decision 2 → Decision 3 →
  Consequences → "What this ADR does not decide," with no `## Status`
  heading anywhere in between. This is the structurally correct shape for
  an ordinarily-Accepted ADR (mirroring ADR-0013/ADR-0014's own shape, not
  ADR-0015/ADR-0016's escalation pattern, which repeats the status in a
  dedicated section specifically because those two are *not* Accepted and
  the redundancy is deliberate emphasis). **The citation-precision lesson
  T24 retro established (quote the actual `## Status`-headed section, not
  a frontmatter bullet a few lines above it) does not mechanically apply
  to ADR-0017, because ADR-0017 is Accepted and has no such section to
  conflate with its frontmatter** — a future ceremony citing ADR-0017's
  status correctly cites the frontmatter bullet, since that is the only
  place its status is stated. Recorded here so no future retro re-derives
  this or, worse, goes looking for a `## Status` section in ADR-0017 that
  does not exist.
- **#144, re-fetched via `issue_read(get_comments)` this retro — the
  comment body itself.** Exactly one comment, unchanged, matching every
  prior retro's quotation verbatim (finding 4).

**Point 4, scored: no — neither D1 nor D2 was answered mid-sprint, as a
formal ADR decision.** ADR-0017 answers questions ADR-0014 §5a explicitly
deferred to a *follow-up* ADR (the FK-conversion and orphan-handling
questions) — it is not, and does not purport to be, an answer to D1 or D2,
which remain the user's own decisions per `sprint-process.md`'s standing
restriction.

## 7. Point 5 — which of D2's two named shapes did T28's real PR review
land in?

**PE and PO, jointly — the substantive question T28 plan §A7 asked this
retro to score, since T28.1 is the first real PR to review since T19.**

**The two shapes, restated from ADR-0016 and T28 plan §A7 exactly.**
T15–T19 produced **five** instances the plan calls "exercised, no fix
needed" — a PR existed, was reviewed, and the review found nothing
requiring the reviewer to author a fix on the branch under review, so D2's
underlying tension (may the reviewing session also *write code* on the PR
it is reviewing) was never actually tested by that PR. T20–T27 produced
**nine** "no PR existed" sprints — the tension could not be exercised at
all, because there was no real ticket to review.

**Which shape T28.1's review lands in, checked against the actual review
record, not inferred from silence.** `pull_request_read(get_reviews, #235)`
returns exactly one review (`id 4946345853`, state `COMMENTED` — GitHub
blocks self-approval since implementer and reviewer share this
environment's one authenticated account, the same standing workaround every
prior sprint's reviews use). Its content, read in full: a re-run of the
whole toolchain, direct reads of every changed file listed by name (not a
restatement of the PR's own summary), and the two independent mutation
checks this retro reproduced in finding 3 (Layers 2 and 4 above). It closes
**"No blocking findings. Approving."** — no fix was authored on
`feature/t28.1-payments-identity-conformance` by the reviewing session; the
PR's single commit (`c975f21`'s pre-merge form, `136e8d5`) is the one the
implementer wrote, and the review's own commit reference
(`commit_id: 136e8d57aa486d2352fc148e40e2f5df9eb41e84`) is that same,
unmodified commit — confirmed by checking that PR #235's `head.sha` and the
review's `commit_id` are identical, i.e. no push happened between review
and merge.

**Scored: this lands in the "exercised, no fix needed" shape — the sixth
such instance, and the first since T19.** A real PR existed, a real review
happened (with independent verification substantially deeper than T15–T19's
five instances, per finding 3's four layers), and it found nothing
requiring a reviewer-authored fix, so the practice D2 is escalated about
(a reviewing session authoring code on the PR it is reviewing) was not
exercised — not because no PR existed this time (T20–T27's shape), but
because the review that did happen needed nothing fixed. This is a genuine
judgement call, stated as one rather than a mechanical classification:
had the review found even a single-file, test-caught, mechanical gap and
fixed it directly on the branch, this would instead be a **new**,
unprecedented instance — neither of D2's two named shapes, since none of
T15–T19's five nor T20–T27's nine involved a reviewer-authored fix on a
real, non-recovery PR. That did not happen here, and the record supports
saying so plainly rather than hedging.

**One further precision, since D2's interim rule is about actual
authorship, not review depth:** the *depth* of this review (four
independent mutation-check layers, full toolchain re-run, direct file
reads) is a separate axis from the D2 question, and a deep review that
finds nothing wrong is not evidence for or against D2's options — it is
simply what "exercised, no fix needed" looks like when the reviewer does
real, independent work rather than restating the PR body. Recorded so a
future ceremony does not conflate "this review was thorough" with
"therefore this instance is stronger evidence for option (b)" — ADR-0016's
own text is explicit that a further instance of the underlying practice
would be "more evidence for the question, not an answer to it," and that
applies here to review depth as much as to authorship.

## 8. Running counters — one retired correctly, one incremented correctly,
one proposed

**PO.**

**The backlog-static-check counter stays retired at fourteen, not
extended.** T28 plan §A3.1 retired this counter at T27 retro's own count
(fourteen: T21 Ceremony 1 through T27 retro) because #164's reclassification
broke the counter's premise — "found the 8-issue set unchanged" stopped
being true the moment T28 Ceremony 1 changed #164's classification. This
retro does **not** resurrect it or pretend a fifteenth check of "the same
unchanged set" happened; that would misdescribe what this ceremony actually
found (7 issues static, #164 in a genuinely new, narrower state).

**D1's consecutive-sprint-silence count — a per-*sprint* counter — continues
incrementing, since D1 itself received no comment this sprint.** T28 plan
§A7 computed this as **fifteen consecutive sprints (T14 through T28)** when
the sprint opened with #144 still uncommented. This retro is Ceremony 3 of
the *same* sprint T28, not a new sprint boundary — re-checking #144 live
(finding 4: still exactly one comment, T14.3's original) **confirms the
count at fifteen, unchanged within this sprint**, rather than incrementing
it a second time. It becomes **sixteen** only if T29 opens with #144 still
uncommented.

**A new counter is proposed, not silently assumed:** T28 plan §A3.1
correctly declined to extend the old counter but also correctly left open
whether a successor counter should start. This retro proposes one, since
the backlog now has a stable-again composition worth tracking for drift:
**the post-T28.1 backlog-composition count** — consecutive live checks
finding the *current* 8-issue set (7 routine issues at their T27-retro
state, #164 narrowed-but-open per PR #235's merge and its 13:59:11Z
comment, D1/D2 both still open) unchanged since T28.1 merged. This retro is
the **first** such check, since it is the first live check performed after
`13:59:11Z`, and it found the set unchanged (finding 4) — so this counter
**starts at one, established by this retro**. It becomes two if T29
Ceremony 1's own sweep finds the same set unchanged again, following the
same per-ceremony-increment shape the old counter used, not a fresh
invention.

## No finding on

**No finding on the migration-header-ownership check.** `0024` was
pre-assigned to T28.1 alone (T28 plan §A4); `ls db/migrations/` confirms
the sequence now ends at `0024` with exactly one new file, and no other
T28 ticket existed to collide with it.

**No finding on the same-wave shared-interface verification rule.** Not
applicable — T28 dispatched exactly one ticket (T28.1); there is no second
same-wave ticket to have collided with it.

**No finding on the label taxonomy.** #164 already carried conformant
labels (`role:principal-engineer`, `type:chore`) before this sprint,
re-verified unchanged via `issue_read(get_labels)`-equivalent fields in
this retro's `issue_read(get)` call. Zero new issues were opened this
sprint, so there is nothing new to check conformance against.

**No finding on PCI conformance (CLAUDE.md rule 11).** PR #235's 21 changed
files were checked directly: zero `.proto` files, zero payment-DTO fields
added or changed. T28.1 touches the *actor identifier space* of an
existing field (`recorded_by_user_id`'s storage type and the value that
populates it), not any card-data-shaped field, and adds no new request DTO
field at all. Nothing for rule 11 to check this sprint.

**No finding on a new #212/#213-shaped gap.** `HANDOFF.md`'s Cross-cutting
section was not touched by PR #235 (confirmed: not among its 21 changed
files) and was already re-scanned in full at T28's own Ceremony 1 (the
eleventh such scan, T18–T28, per T28 plan §A3) — re-reading an unmodified
section a second time within the same sprint would not be independent
verification of anything new, the same reasoning T26/T27 retro used for an
unmodified file within one sprint.

**No finding on the migration-tooling (`golang-migrate`/`goose`)
roadmap-debt classification.** Unchanged — still the single `HANDOFF.md`
line every prior ceremony has quoted, re-confirmed this retro by the same
grep every prior retro has run, with no new file referencing either tool
beyond this retro's own prose.

---

## The sprint goal, scored

> *"Correct the record for T27. Independently re-verify, from #164's own
> issue text, ADR-0014's own ruling, and the current codebase — not from
> fourteen sprints of restated classification — whether #164 is genuinely
> blocked on an external identity provider. It is not… Take the smallest,
> lowest-risk third of it — Payments' conformance — as this sprint's real
> scope… Re-verify the other 7 issues' blockers live and find them
> unchanged, with #145 specifically re-confirmed as genuinely IdP-blocked
> despite its superficial similarity to #164."*

**Every clause holds, independently re-verified rather than taken from the
plan's or the PR's own account.** #164 was correctly reclassified and its
Payments third shipped as a genuine reference implementation, verified
directly against the merged diff (finding 2) with the backfill
mutation-checked four independent ways (finding 3). The other 7 issues'
blockers are confirmed unchanged, byte-for-byte against the live API
(finding 4). #164 was narrowed, not closed, with the comment actually
posted (finding 5). Neither D1 nor D2 was answered as a formal decision
(finding 6), and this sprint's real review is scored against D2's two named
shapes as the task instructed (finding 7). The running counters are handled
correctly — one retired, one incremented, one newly proposed (finding 8).

**The agreed honest sentence, which T29's Ceremony 1 should carry into
`HANDOFF.md`'s Task-backlog entry** (`sprint-process.md` Ceremony 1 item 3
requires the retro's form, not a stronger one):

> T28 shipped one ticket, T28.1, 8 points — the first non-zero sprint since
> T19, ending the eight-sprint 0-ticket run (T20 through T27). It closed the
> Payments third of issue #164 (partial fix, PR #235, merged
> `2026-08-16T13:59:00Z`, squash sha `c975f219ea5571c863453021ae277961abf72218`)
> after independently re-deriving, at Ceremony 1, that #164's fourteen-sprint
> "blocked on a real IdP tenant" classification was never supported by the
> issue's own text or by ADR-0014's own ruling. This retro independently
> re-verified — not trusted — every load-bearing claim: the funnel change
> and the backfill/column-type migration landed in exactly one commit with
> no window where `authorizeOnlineConfirmation`'s comparison could silently
> break; the backfill was mutation-checked against both the orphaned and
> resolvable cases by four independent layers (a committed but
> CI-unexecuted integration test, the reviewing session's own DB
> reproduction, this retro's own separate third DB reproduction with its
> own seed data, and this retro's own independent Go-level fail-closed
> mutation); the other 7 backlog issues' blockers are unchanged,
> byte-for-byte against the live API; #164 was correctly narrowed rather
> than closed, with the narrowing comment actually posted 11 seconds after
> merge rather than only promised; neither D1 nor D2 was answered as a
> formal ADR decision this sprint, both ADRs' status sections read directly
> (and ADR-0017's own correct citation form — frontmatter-only, no separate
> `## Status` section, since it is Accepted rather than escalated —
> established for the first time); and this sprint's real PR review scores
> as D2's "exercised, no fix needed" shape, the sixth such instance and the
> first since T19, with review depth noted as a separate axis from the D2
> question itself. The backlog's old consecutive-static-check counter was
> correctly retired at fourteen rather than extended (T28 plan §A3.1); D1's
> consecutive-sprint-silence counter holds at fifteen (T14 through T28,
> unchanged within this same sprint); a new post-T28.1 backlog-composition
> counter is proposed, starting at one, established by this retro. One
> process-hygiene observation (not an incident) was recorded: PR #235's
> disclosed interaction with issue #149's caller-supplied `booking_host_id`
> field was not also posted as a comment on #149 itself, though nothing
> #149 already states was falsified by it.

---

## Recommendations for T29's Ceremony 1 and 2

1. **Re-run the merged-fix sweep as the authoritative moment**, per the
   standing rule — re-verify the open count from the live API rather than
   trusting this retro's table (finding 1).
2. **Consider posting a short comment to #149** disclosing that
   `booking_host_id`'s expected shape changed from subject to `User.ID` as
   a side effect of T28.1, purely for record completeness — not required by
   any existing rule (finding 5), but cheap and in keeping with this
   project's stated preference for the issue tracker carrying facts a PR
   body alone would otherwise hide from a future reader.
3. **Continue the post-T28.1 backlog-composition counter proposed in
   finding 8**, incrementing it to two if T29 Ceremony 1's own live check
   again finds the 8-issue set (as it stands after T28.1) unchanged — do
   not reintroduce or extend the old, retired counter.
4. **When Social Play and/or Competitions are taken (T29 or later),
   re-verify ADR-0017's rulings still hold rather than re-deriving them from
   scratch** — Decision 2 and Decision 3 were written explicitly so T29
   would not have to re-litigate either, per ADR-0017's own "Consequences"
   section.
5. **If a review ever does perform a reviewer-authored fix on a real,
   non-recovery PR under D2's still-unanswered interim rule, score it as a
   genuinely new instance** — neither "exercised, no fix needed" nor "no PR
   existed" — rather than forcing it into one of the two existing buckets
   (finding 7).

## Sprint-level Definition of Done — scored against what T28's own plan
asked

Per `docs/process/t28-sprint-plan.md`'s "Sprint-level Definition of Done,"
the items owed at this retro, restated here with their answers:

1. **T28.1 merged per the per-ticket DoD?** **Yes** — acceptance criteria
   met, reviewed (finding 7), `make test-domain`/`make test-adapters`/`make
   fmt-check`/`make gate-coverage` all re-run clean directly against the
   merged tree this retro, approved and merged by the delegated gate, never
   self-merged by an implementing session distinct from the reviewer.
2. **The merged-fix issue sweep run and reported?** **Yes** — finding 1;
   T29's Ceremony 1 remains authoritative.
3. **#164 narrowed by an explicit comment at merge time?** **Yes, actually
   posted** — finding 5, 11 seconds after merge, not merely stated as
   intent.
4. **(a) Funnel + backfill/column-type change shipped together, no
   window?** **Yes** — finding 2, verified against the actual merged
   commit.
   **(b) Backfill mutation-checked per CLAUDE.md rule 10?** **Yes, four
   independent layers** — finding 3.
   **(c) Other 7 issues' blockers still hold, re-verified live?** **Yes** —
   finding 4.
   **(d) D1 or D2 answered mid-sprint?** **No, neither** — finding 6.
   **(e) Which of D2's two named shapes does this sprint's review land
   in?** **"Exercised, no fix needed"** — the sixth instance, first since
   T19 — finding 7.
5. **Not scoreable by T28 and deliberately not pre-empted:** D1 and D2
   remain the user's own decisions. Whether T29 takes Social Play,
   Competitions, or both is PE/PM's call at that time (T28 plan §B9),
   informed by how T28.1's own backfill migration went — not pre-committed
   here.
6. **Retro in `docs/process/t28-retro.md`** (this document), indexed by a
   `## T28 sprint retro` stub in `docs/LESSONS.md`. `HANDOFF.md`'s Docs
   index and Task-backlog entry updated by this same PR, per this project's
   standing convention that a retro PR *may* correct the row that points at
   it (its own merge number is knowable once written, unlike the row
   pointing at the sprint plan) — carried out here rather than deferred to
   T29's Ceremony 1, since the T28 row's Retro/Reviews cells were the only
   ones still reading "not yet written"/"not yet opened."

Retro complete. Issue-tracker actions this ceremony: **one** — #164's
narrowing comment (finding 5), already posted by the merging session before
this retro began; nothing new posted by this retro itself beyond the
recommendation in finding 5 item 2. Open count at ceremony start: **8**.
Open count now: **8** (unchanged — #164 stays open by design). No incident
qualifies for a `docs/LESSONS.md` entry this sprint — every finding above
either confirms a claim held under independent re-verification or records a
non-blocking process-hygiene observation, so only the standard index stub
is added.

Toolchain re-verified directly against the merged tree this retro (after
regenerating `internal/gen`, gitignored and stale in this session before
that step): `make test-domain` — 12/12 packages green; `make test-adapters`
— 24/24 packages green, including the new `internal/payments/adapter/identity`;
`make fmt-check` — clean; `make gate-coverage` — `OK — all 43 package(s)
with runnable tests are executed by "ci-checks"`, matching PR #235's own
reported count exactly; `go build ./...` and `go vet -tags=integration
./...` both clean.

Per `sprint-process.md`'s established convention: **`HANDOFF.md`'s T28 row
is corrected by this same PR**, since its Retro and Reviews cells were the
only fields still reading "not yet written"/"not yet opened" and no later
ceremony's own first act would otherwise fix them before T29 plans off a
stale row.
