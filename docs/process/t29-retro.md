# T29 Sprint Retro

Ceremony 3 per `docs/process/sprint-process.md`, six-role team (briefs:
`docs/agent-operating-handbook.md` Part B), held against
`docs/process/t29-sprint-plan.md` (§A0–§B7 and the Ceremony 2 tickets/DoD),
`docs/process/t28-retro.md` as the structure/tone/rigor precedent, `HANDOFF.md`,
ADR-0016, ADR-0017, `docs/LESSONS.md`'s T9 entry, and the live PR/issue/commit
history on `nhuthuynh/white-label` (GitHub-side name `pickleball-platform`) —
PR #238 (Ceremony 1/2 doc), PR #239 (T29.1, "partial fix for #164:
Competitions identity conformance"), PR #240 (T29.2, "partial fix for #164:
Social Play identity conformance"), two closed issues (#164, #237), zero new
ADRs, three narrowing/closing comments on #164 and two on #237.

**Every factual claim below was re-derived against the live repository and
GitHub API, not read from the sprint plan's or either PR's own account of
itself** (CLAUDE.md rule 10; this project's standing convention, tightened
further after several prior retros caught a claim that had only ever been
restated). Concretely, this retro: re-ran the full toolchain against the
merged tip (`be9c8ca9`); independently reproduced both migrations' mutation
checks against a real local Postgres 16 instance with this retro's own seed
data, not the PR's or the review's fixtures; read every changed file named
below directly rather than trusting a PR body's description of it; fetched
every issue and PR referenced from the API; and found two real, previously
undetected corrections along the way (§10, §11) that neither T29's own plan
nor either PR's own account had caught.

**Verification performed before writing a single finding.** `git log`
confirms the local checkout's tip matches `origin/claude/go-backend-pickleball-7up34j`
exactly at `be9c8ca9ecc001cefddaed05f9ed9239c2edef91`, three commits ahead of
T28's own tip (`c975f21`): `8394e97` (PR #238, Ceremony 1/2 doc), `580dd18`
(PR #239, T29.1), `be9c8ca` (PR #240, T29.2) — squash merges, one commit each
on the shared branch, confirmed via `git log --oneline`. `make generate` was
re-run first (gitignored `internal/gen` is not checked in); every gate below
ran directly against that regenerated tree.

**What this retro found, in one sentence, so the findings do not have to be
read before the headline is known:** both tickets shipped their funnel change
and backfill migration together as one reviewed unit with no window either
comparison could break in, both backfills were mutation-checked per CLAUDE.md
rule 10 by verification shapes that legitimately differ rather than one being
weaker, both tickets' `NOT NULL`-vs-nullable migration branches were the
correct one for their own structural/empirical reasons, both Payments-side
regression tests use genuinely non-matching fixture values that would have
caught #237, the open backlog is **7** routine issues at this retro's live
check (not 6 — the sprint plan's own DoD undercounted by omitting #124,
corrected in §5), neither D1 nor D2 was answered as a formal decision, T29.1's
review lands in D2's already-named "exercised, no fix needed" shape while
T29.2's review lands in a **fourth, previously-unnamed shape** — a real gap
found, changes formally requested, and the fix authored by neither the
implementer nor the reviewer but a third, independently-dispatched party —
distinct from both of D2's existing shapes and from the "reviewer authors the
fix itself" shape T28 retro recommendation 5 anticipated (§7); the
shared-checkout hazard is scored as a near-miss with one real, verified
process-institutionalization gap underneath it worth its own `docs/LESSONS.md`
line (§8); and the empty-PR-body incident is scored as adequately caught by
existing process, with one concrete, cheap safeguard recommended for T30 (§9).

---

## 1. The merged-fix issue sweep

**PO.** Per `sprint-process.md`'s DoD, this retro is sweep moment 1; T30's
Ceremony 1 remains the authoritative moment regardless of this result.

**Step 1 — list the open issues, live, at this retro's start.**
`list_issues(state: OPEN)` → **`totalCount: 7`**: #124, #126, #130, #134,
#144, #145, #149.

**Step 2 — reconcile arithmetically, before reading any issue individually.**
`open_at_end_of_T28's_retro (8) − closed_during_T29 (2: #164, #237) +
opened_during_T29 (1: #237) = 7`. Matches the live `totalCount: 7` exactly.
**This is 7, not the 6 the task brief assumed and not the 6 T29's own plan's
DoD item 5(e) named** — see §5 for the independently-traced cause (the plan's
own §A5 ranked-backlog table silently dropped #124, which is why its
downstream DoD line inherited the same miscount). The live sweep arithmetic
is unaffected by that drafting gap; it was never derived from the §A5 table,
only from `list_issues` and the plan's own step-1/step-2 counts, both of
which correctly carried #124 throughout (§A1, §A3).

**Step 3 — cross-reference merged PRs against the open list.** Three PRs
merged this sprint: **#238** (Ceremony 1/2 doc, `merged_at: 2026-08-16T14:28:18Z`),
**#239** (T29.1, `merged_at: 15:04:23Z`, titled "partial fix for #164:
Competitions identity conformance…"), **#240** (T29.2, `merged_at:
15:23:09Z`, titled "partial fix for #164: Social Play identity
conformance…"). #164 is named in both #239's and #240's titles; #237 is named
in both PRs' bodies (not their titles). Cross-referenced against the closed
issues: `issue_read` on both #164 and #237 confirms `state: closed,
state_reason: completed`, `closed_by: nhuthuynh`, `closed_at: 2026-08-16T15:23:14Z`
(#164) and `15:23:16Z` (#237) — both closed by the comment PR #240's own
review posted immediately after merge, per instruction 10's live-state-check
discipline (verified: the comment text on #164, id `5308164898`, correctly
names all three merged PRs — #235/T28.1, #239/T29.1, #240/T29.2 — and states
"Closing"; the comment on #237, id `5308165079`, does the same for the
regression). No other issue number appears as a claimed closure in any of the
three PRs' titles or bodies.

**Sweep result: clean.** Both issues taken this sprint (#164, #237) were
correctly closed, with comments that checked live state before writing
(exactly the discipline both tickets' instruction 10 required — see §7 for
why this discipline still applies even though this sprint's PR titles never
say "closes #N" outright). **T30's Ceremony 1 still re-runs this sweep in
full**, per the standing rule that a prior ceremony's clean result does not
discharge the next one.

## 2. Point (a) — did each ticket ship its funnel change and
backfill/column-type migration together as one reviewed unit, with no window
any comparison could break in?

**PE, verified directly against each merged commit, not either PR body's own
"why one PR" section.**

**T29.1 (PR #239).** `git log --oneline 8394e97..580dd18` shows exactly one
commit (`6b4e9ab1`) touching all 24 changed files. `internal/competitions/adapter/grpcapi/handler.go`'s
`actor(ctx)` free function becomes `h.actor`, resolving through the new
`internal/competitions/port.IdentityLookup` — read directly, confirmed
present in the same commit as `db/migrations/0025_competitions_identity_conformance.sql`
and every comparison site the ticket named (`competition.go:214,264`,
`competition_admin.go:121`, `service.go`'s `ListEntriesForCompetition`/
`CancelCompetition`/`AssignCompetitionAdmin`/`RevokeCompetitionAdmin`). One
commit, one PR, no intermediate state ever reachable on the shared branch.

**T29.2 (PR #240).** Two commits on the branch (`e15c2cc6`, `6067fef1`), but
the funnel change and the migration are both in the **first** commit
(`e15c2cc6`) — confirmed via `pull_request_read(get_commits)` and by reading
the commit's own message ("Converts games.host_id… wires the grpcapi actor()
funnel through it, and backfills existing rows"). The second commit
(`6067fef1`) adds only `internal/socialplay/adapter/identity/lookup_test.go`
— a test file, touching neither the funnel nor the migration nor any
comparison site. Squash-merged as one commit (`be9c8ca9`) onto the shared
branch (confirmed: `git log --oneline` shows exactly one T29.2 commit on
`claude/go-backend-pickleball-7up34j`), so **no window existed on the shared
branch either way** — the PR's own two-commit internal history never
partially landed.

**The comparisons themselves, read directly.** `internal/competitions/domain/competition.go`'s
`EnsureHost`/`EnsureHostOrCompetitionAdmin` and `internal/socialplay/domain/game.go`'s
`EnsureHost`/`EnsureHostOrGameAdmin` (plus `registration.go`'s `Cancel`,
`waitlist.go`'s `Cancel`, both `game_admin.go`/`competition_admin.go`'s
`HasGameAdmin`/`HasCompetitionAdmin`) are unchanged in comparison **logic** by
either diff — only what feeds both sides changed, the identical shape T28.1
established and both tickets' own PR bodies claim. Confirmed by reading each
file directly rather than trusting the claim.

**Conclusion, both tickets: confirmed, independently, against the actual
merged diffs.** The funnel resolution and the backfill/column-type migration
it depends on landed in exactly one commit each, on exactly one PR each, with
no intermediate state ever reachable on the shared branch. No window existed
for either ticket.

## 3. Point (b) — was each backfill mutation-checked per CLAUDE.md rule 10,
and why the two verification shapes legitimately differ

**QA, with this retro's own third, independent reproduction of both
migrations — not merely re-reading the PRs' or reviews' claims.**

**Why the shapes differ, stated first since it is the substantive part of
this finding.** T29.1's migration ships the four Competitions columns
**nullable**; T29.2's ships all five Social Play columns **`NOT NULL`**
(§4 scores whether each branch was the *correct* one — this finding scores
only whether rule 10's orphan-and-resolvable exercise happened). A nullable
column can hold and display the orphan case directly in the shipped
migration text; a column that goes `NOT NULL` in the same migration
**cannot** — a real orphan makes that exact migration abort at the `SET NOT
NULL` step, by design (ADR-0017 Decision 3's own "fails loudly, not
silently" requirement). This is not a gap in T29.2's proof, it is a
structural fact about what a `NOT NULL` migration can and cannot
demonstrate about itself, and it is why T29.2's own mutation-check
necessarily has two parts where T29.1's needs one.

**T29.1 (Competitions, nullable) — one reproduction, direct.** This retro
built a scratch database (`t29_comp_orphan`), applied migrations `0001`–`0024`,
seeded one `identity_users` row plus one resolvable and one orphaned
`competitions` row (this retro's own literal UUIDs, not the PR's or the
review's), and applied `0025` directly. Result, read back by raw SQL:

| Assertion | Result |
|---|---|
| Resolvable row's `host_id` | `11111111-1111-1111-1111-111111111111` — the exact seeded `identity_users.id`, not merely non-null |
| Orphan row's `host_id` | `NULL` |
| Row count | 2 (neither dropped) |
| Column type | `uuid`, with a live `competitions_host_id_uuid_fkey` FK |

This directly reproduces the shipped migration text's own orphan-handling —
Layer-1-equivalent to T28.1's committed-test shape, except run here for real
against the actual `0025` file, not merely read.

**T29.2 (Social Play, `NOT NULL`) — two reproductions, matching the PR's own
two-part proof exactly, independently re-derived rather than trusted.**

1. **The backfill mechanism, in isolation from the `NOT NULL` determination.**
   A fresh scratch database, migrations `0001`–`0025`, one resolvable and one
   orphaned `games` row seeded with this retro's own literals, then **only**
   the `ADD COLUMN`/`UPDATE … FROM identity_users` portion of `0026` applied
   by hand (mirroring the migration's own first two statements, not the
   file's `SET NOT NULL`/`DROP`/`RENAME` steps). Result: the resolvable row's
   `host_id_uuid` resolved to the exact seeded `identity_users.id`; the
   orphan row's is `NULL`. This proves the join resolves-or-nulls correctly,
   independent of whether `NOT NULL` can ultimately be added.
2. **The shipped migration's actual behavior on a real orphan.** A second
   scratch database, migrations `0001`–`0025`, one seeded orphaned `games`
   row, then the **real, verbatim, unmodified `0026`** applied. Result:
   `psql` exits non-zero with `ERROR: column "host_id_uuid" of relation
   "games" contains null values` at the `ALTER TABLE games ALTER COLUMN
   host_id_uuid SET NOT NULL` step — a clean, comprehensible Postgres error,
   not a silent pass and not data loss, exactly matching ADR-0017 Decision
   3's "fails loudly, not silently" requirement and both PR #240's own claim
   and its review's independent reproduction.

Both scratch databases were dropped after use. This is a genuinely
independent, third reproduction (after the PR's own account and the
reviewing session's own DB-level check, per PR #240's first review) — this
retro's own seed values, this retro's own session, same migration files read
directly off the merged commit.

**Point (b), scored: yes for both tickets — mutation-checked per CLAUDE.md
rule 10, with an orphaned subject and a resolvable one both exercised per
table.** The verification *shape* legitimately differs (one direct
reproduction for the nullable case, two complementary reproductions for the
`NOT NULL` case) because the two migrations make structurally different
claims about themselves, not because either ticket's own diligence was
weaker. Scoring T29.2's two-part proof as "weaker" would be scoring the wrong
axis — it is *more* than T29.1's proof covers, not less, precisely because a
`NOT NULL` migration has an extra thing (loud, not silent, failure) that
needs proving which a nullable migration does not.

## 4. Points (c) and (d) — the migration's `NOT NULL`/nullable branch, and
whether the Payments-side regression tests would have caught #237

### Point (c) — did each migration ship `NOT NULL` or nullable, and was
that the correct branch per ADR-0017 Decision 3?

**PE, scored independently per ticket, against Decision 3's own restated text
for the `NOT NULL`-starting-point case (`docs/adr/0017-*.md`, read in full
this retro).**

**T29.1 shipped nullable — correct, for a structural reason, not a
preference.** `competition_admins.user_id` was part of a composite `PRIMARY
KEY (competition_id, user_id)`; Postgres forces `NOT NULL` on every `PRIMARY
KEY` column, with no exception. Confirmed directly against the pre-migration
schema (`db/migrations/0021_competitions_competition_admins.sql`) and the
shipped `0025`'s own `DROP COLUMN user_id` / re-`ADD CONSTRAINT
competition_admins_competition_id_user_id_key UNIQUE (competition_id,
user_id)` sequence (read in full — the composite `PRIMARY KEY` is genuinely
gone, replaced by a `UNIQUE` constraint on the same two columns, confirmed by
`\d competition_admins` in this retro's own scratch database above). Had
`0025` shipped `NOT NULL` and tried to preserve the `PRIMARY KEY`, the
ticket's own required orphan mutation-check (§3) would have had no orphan
case to observe — the migration would have aborted at the same `SET NOT
NULL`-equivalent step T29.2's does, for the same reason, just one column
sooner. Nullable was the only choice that let this ticket satisfy both
ADR-0017 Decision 3 and CLAUDE.md rule 10 at once for this specific column,
and the other three columns shipped nullable alongside it for consistency
(stated in the migration's own header) rather than independently — a
reasonable choice given the four columns' text sits in one migration file
and one column's structural constraint would otherwise leave three others in
a different nullability posture than the fourth for no principled reason.

**T29.2 shipped `NOT NULL` — correct, for an empirical reason, independently
re-verified rather than trusted from the migration header's own narrative.**
This retro re-ran the same checks the migration header and PR #240's body
claim: `grep -l "games\|registrations\|waitlist_entries\|game_admins"
db/migrations/000{2,4}_seed*.sql` returns no matches (confirmed: neither seed
migration references any of the four tables), and `ls db/migrations/` +
CLAUDE.md's own documented gotcha (`db/migrations/*.sql` applies via
docker-compose `initdb.d` only on a **fresh** volume) together mean every
environment that can run this migration starts these four tables at zero
rows. §3's own two-part mutation-check reproduced both consequences directly:
a genuinely empty backfill succeeds cleanly with `NOT NULL` (the full
`0001`–`0026` chain applied clean in this retro's first scratch database,
§3's context, before any of the targeted orphan tests), and a real orphan —
which cannot occur in this deployment model, but which the migration is
still required to handle safely if the deployment-model assumption ever
turned out to be wrong — is refused loudly rather than silently accepted.
ADR-0017 Decision 3's own restated permission ("the `NOT NULL` constraint
may be added in the same migration once the backfill is observed clean")
is satisfied on the stronger of its two grounds: not merely "plausible," as
the ADR's own hedge puts it, but **structurally guaranteed** by this
project's fresh-volume-only deployment model, independently confirmed by
this retro rather than taken on the migration header's word.

**Point (c), scored: yes for both tickets, each independently correct for
its own reason.** Neither is the "safe default" copied from the other —
T29.1's nullable choice is forced by a PK/rule-10 conflict specific to
`competition_admins`; T29.2's `NOT NULL` choice is earned by an empirical,
independently-reproduced zero-orphan guarantee specific to Social Play's four
tables. Scoring either as "the more careful" or "the less careful" choice
would miss that they are answering the same ADR clause correctly from two
different starting facts.

### Point (d) — do the Payments-side regression tests use non-matching
fixture values, and would they have caught #237?

**QA, verified by reading both test files in full — not trusting either PR
body's own description of its fixtures.**

**#237's own root-cause diagnosis, quoted directly from the issue rather
than paraphrased, since it is the exact defect both tests exist to make
impossible to reintroduce silently:** *"`internal/payments/app/service_test.go`'s
fixtures… construct `ActorUserID` and the fake `GameLookup`'s `HostID` from
the SAME LITERAL STRING (e.g. `"host-1"`), which does not model the real
system's two distinct identifier spaces post-T28.1 and so cannot detect a
space mismatch either way."*

**T29.1's test (`internal/payments/app/competition_entry_identifier_space_regression_test.go`,
331 lines, read in full).** Declares four named, real, uuid-shaped, mutually
distinct constants — one per role (entrant, admin, and the rejected
stranger) — each with its own doc comment naming the real-world identifier-
space fact it stands in for, explicitly *not* a single string retyped in two
places. 8 tests exercise `RecordOfflinePayment`/`RefundPayment`/
`CreateOnlinePayment`'s `competition_entry` path with these constants.

**T29.2's tests (`internal/payments/app/service_test.go`, two new tests read
in full: `TestRecordOfflinePayment_RegistrationPayable_ResolvedIdentifierSpaces_GameHostSucceeds`
and its `_MismatchedActorRejected` sibling).** Two independently-declared
uuid-shaped constants (`gameHostUserID`, `mismatchedActorUserID`), neither
derived from the other. The success test asserts `RecordOfflinePayment`
succeeds and `p.RecordedByUserID` equals the exact constant; the rejection
test asserts the mismatched constant is rejected with
`domain.ErrNotPaymentRecorder` specifically (`errors.Is`, not a bare
non-nil-error check). Both tests' own doc comments name the pre-existing
tests' short-mnemonic-string shape (`"host-1"`, `"admin-1"`) as the exact
pattern being deliberately avoided.

**Would either actually have caught #237?** Yes, by construction, and this
is the load-bearing property rather than an incidental one: #237's actual
defect was a resolved `User.ID` (Payments' side) being compared against a
still-subject-shaped value (Social Play's/Competitions' side) — two values
that, before either T29 ticket's own conformance work, could never
coincidentally be equal, meaning the *success* half of each pair (the "Host
succeeds"/"Entrant succeeds" tests) would fail against pre-conformance code
even with genuinely-matching-in-principle actor and host values, because one
side would still be a subject string and the other a resolved uuid. A test
built from `"host-1"`/`"host-1"` on both sides cannot expose this, because a
short mnemonic string is not distinguishable, before or after resolution, in
a way a passing assertion would notice; a test built from two independently-
declared, realistic uuid literals is exactly the shape that fails the moment
one side of the comparison silently stops being the same value as the other
— which is precisely what #237 did.

**Point (d), scored: yes for both tickets** — both regression suites use
genuinely non-matching-by-default (independently declared, mutually
distinct, realistic) fixture values rather than a short string reused on
both sides, and both would have caught #237's specific defect shape by
construction, confirmed by reading the files directly rather than trusting
either PR body's own claim.

## 5. Point (e) — do the other issues' blockers still hold, re-verified
live? (And: 6 or 7?)

**QA and BA, against freshly fetched fields — and a correction to the task's
own and the sprint plan's own count, traced to its source.**

**The correction, traced.** T29's own sprint plan §A5 ("The whole open
backlog, ranked, with a disposition for each") lists eight rows — #164,
#237, #144, #149, #145, #126, #130, #134 — and its own Sprint-level DoD item
5(e) then asks whether "the other **6** issues'" blockers hold, i.e. the
eight-row set minus #164 and #237. **#124 is not one of the eight rows**,
despite being present, open, and re-verified with an unchanged blocker in
that same plan's own §A3 routine-re-verification table two sections
earlier ("#124: still needs Product Owner input on cascade semantics…").
This is an internal inconsistency in the plan document itself — §A3 correctly
carries all 8 pre-sprint issues including #124; §A5's ranking table silently
drops it with no stated reason; the DoD's "6" is downstream of §A5, not §A3,
and inherits the gap. Nothing in T29.1's or T29.2's own scope touches #124 or
its blocker, so this is a drafting omission in the plan, not a missed
disposition — but it is the reason both the task brief's "should be down to
6" assumption and the plan's own DoD wording are off by one, and it is
reported here rather than silently corrected, per this project's standing
practice of naming a drafting gap rather than quietly working around it.

**The live count is 7**, confirmed in §1: #124, #126, #130, #134, #144,
#145, #149.

| Issue | `updated_at`/comments at T29 Ceremony 1 (plan §A3) | `updated_at`/comments now (live, this retro) | Changed? |
|---|---|---|---|
| #144 (D1) | `2026-08-15T07:01:03Z`, 1 comment | `2026-08-15T07:01:03Z`, 1 comment | No |
| #149 | 4 comments (this ceremony's own §A2 item 2 comment already counted) | `2026-08-16T14:20:31Z`, 4 comments | No — unchanged since the plan's own action |
| #145 | `2026-08-15T05:01:29Z`, 1 comment | `2026-08-15T05:01:29Z`, 1 comment | No |
| #134 | `2026-08-14T16:37:49Z`, 0 comments | `2026-08-14T16:37:49Z`, 0 comments | No |
| #130 | `2026-08-14T16:30:25Z`, 0 comments | `2026-08-14T16:30:25Z`, 0 comments | No |
| #126 | `2026-08-14T16:12:26Z`, 0 comments | `2026-08-14T16:12:26Z`, 0 comments | No |
| #124 | plan's own table says `2026-08-14T16:25:34Z`, 1 comment | `2026-08-15T16:25:34Z`, 1 comment (`issue_read`, this retro) | No — see note below |

**One more transcription slip found while re-deriving this table, not a live
change.** T29 plan's own §A3 table gives #124's "at T28 retro" timestamp as
`2026-08-14T16:25:34Z` — but T28 retro's own table (`docs/process/t28-retro.md`
§4) gives the same field as `2026-08-15T16:25:34Z`, one day later, and this
retro's own live fetch confirms `2026-08-15T16:25:34Z` still. So T29 plan's
own copy of the figure has a one-day transcription error against T28 retro's
correctly-fetched value; the underlying fact (#124 unchanged since T28 retro)
was never wrong, only one document's re-transcription of it. Reported here
per this project's standing practice of naming a drafting slip rather than
silently repeating it a third time.

Every other field matches, byte-for-byte.
Every issue's full body was also re-read this retro (`issue_read(get)`), not
just `updated_at`/comment counts: #144 still has zero authz on `CancelBooking`/
`CreateBooking`, blocked on D1; #149 still names `booking_host_id` as its one
caller-supplied fact, blocked on D1; #145 still needs a real IdP tenant's
non-uuid `sub` claim this environment cannot produce; #124 still needs
Product Owner input on cascade semantics; #126/#130/#134 unchanged (per-Game
price field, `no_show_fee` refund policy, WCAG hardware access, respectively).

**Point (e), scored: yes, all 7 (not 6) other issues' blockers hold,
re-verified live — with the plan's own undercount corrected here rather than
silently propagated.**

## 6. Point (f) — did D1 or D2 get answered mid-sprint, as a formal ADR
decision?

**PE and PO, jointly, checked directly against both files' git history and
their own `## Status`/frontmatter sections — not assumed unchanged.**

`git log --oneline -- docs/adr/0015-*.md` and `git log --oneline --
docs/adr/0016-*.md` each show exactly one commit — their original T14.3/T15.2
authoring commits — with **no** commit since, including none from T29's three
merged PRs (#238, #239, #240 all confirmed via `pull_request_read(get_files)`
to touch neither file). Read in full this retro regardless of the git-log
result, per this project's standing "don't infer from a diff, read the
section" discipline:

- **ADR-0015 `## Status`:** unchanged — **"Escalated — awaiting product
  decision. This ADR decides nothing."**
- **ADR-0016 `## Status`:** unchanged — **"Escalated — awaiting the user's
  decision. This ADR decides nothing."**
- **#144, re-fetched via `issue_read(get_comments)` this retro — the comment
  body itself, not just the count.** Exactly one comment, `id 5301056598`,
  T14.3's original escalation, word-for-word identical to every prior
  retro's quotation.

**Point (f), scored: no — neither D1 nor D2 was answered mid-sprint, as a
formal ADR decision.** Nothing in ADR-0017 (unchanged this sprint; T29.1/
T29.2 *implement* its already-ruled Decisions 1–3, they do not amend the file
— confirmed via `pull_request_read(get_files)` on both PRs, neither touches
`docs/adr/0017-*.md`) bears on either question either.

## 7. Point (g) — which of D2's shapes does each ticket's real review land
in, scored per ticket

**PE and PO, jointly — the substantive question this sprint's DoD asked,
argued from ADR-0016's own definitions rather than asserted.**

**The shapes as ADR-0016 and T28 retro actually define them, restated
precisely before scoring against them.** "Exercised, no fix needed": a PR
existed, was reviewed, and the review found nothing requiring a
reviewer-authored fix — D2's underlying tension (may the *reviewing* session
also *author code* on the PR it reviews) is never actually engaged, because
there is no fix to attribute to anyone. "No PR existed": no ticket, no
review, nothing to score. T28 retro recommendation 5 named a third,
not-yet-observed possibility: a reviewer finds a real gap and authors the fix
itself, directly, on the branch under review — which would engage D2's
actual tension for the first time, since that is precisely the practice D2
is escalated about.

**T29.1 (PR #239) — "exercised, no fix needed," the seventh such instance.**
`pull_request_read(get_reviews, #239)` returns exactly one review (`id
4946469700`, state `COMMENTED` — self-approval blocked, the same standing
workaround every prior review uses), closing **"No blocking findings…
Approving."** Read in full: a fresh-worktree toolchain re-run, an
independent DB-level mutation check (this retro's own §3 reproduces it
separately), and direct reads of the `resolveTargetUserID` seam and the
Payments-side regression test — no fix authored on
`feature/t29.1-competitions-identity-conformance` by the reviewing session;
the PR's single commit (`6b4e9ab1`) is the one the implementer wrote, and the
review's own `commit_id` field matches the PR's `head.sha` exactly (both
`6b4e9ab1b4b71cee7e94e816c9e4dd792fbbccc5`), confirming no push happened
between review and merge. This is the same shape T28.1's review landed in —
squarely "exercised, no fix needed," D2's tension not engaged because there
was nothing to fix.

**T29.2 (PR #240) — a genuinely new, fourth shape, argued from ADR-0016's own
text rather than forced into one of the three named possibilities.**
`pull_request_read(get_reviews, #240)` returns **two** reviews:

1. `id 4946479055` (`15:09:17Z`) — **requests changes**, with one real,
   substantiated gap: `internal/socialplay/adapter/identity/lookup.go`
   shipped with zero direct test coverage, breaking the pattern all three
   prior instances of this exact port/adapter shape carry (Booking T13.2,
   Facilities T13.3, Payments T28.1, and — this same sprint — Competitions
   T29.1). The review states explicitly: *"per this project's D2 interim
   rule, a reviewer who finds a gap requests changes rather than fixing it,
   so I'm not touching the code myself."* It also discloses the PR-body
   incident (§9).
2. Between the two reviews, commit `6067fef1` lands
   (`internal/socialplay/adapter/identity/lookup_test.go`, five non-vacuous
   tests, confirmed by reading the file directly — exact-value resolution,
   the unregistered-subject case with an explicit negative assertion that
   `identitydomain.ErrUserNotFound` does not leak across the CLAUDE.md rule
   5 boundary, the empty-subject case, an "other error" wrapping case with
   both a positive and negative assertion, and a compile-time structural
   check). Its own commit message frames it in the third person — *"closing
   a test-coverage gap **found in PR #240 review**"* — language that
   describes reacting to someone else's finding, not a first-person account
   of authoring one's own review's fix.
3. `id 4946512136` (`15:23:02Z`) — confirms the gap resolved, re-runs the
   full toolchain against both commits merged onto the current shared-branch
   tip, confirms `gate-coverage` picked up the new package (44 → 45,
   independently reproduced by this retro: `go run ./cmd/gatecoverage`
   reports exactly **45/45** against the current merged tree), and closes
   **"No further blocking findings. Approving."**

**What this is not.** It is not "exercised, no fix needed" — a real fix was
required and made. It is not "no PR existed." It is not T28 retro
recommendation 5's named third possibility either, on a precise reading of
that possibility's own definition: recommendation 5 describes **the
reviewing session itself** authoring the fix directly on the branch under
review, which is the specific practice D2 is escalated about (ADR-0016's
question, verbatim: *"may it also write the code on that pull request"* —
"it" being the same session that reviews and merges). Here, the review's own
first-person language ("I'm not touching the code myself") and the second
review's third-person framing of who found the gap, together with the
requesting-changes review explicitly declining to author anything, indicate
the fix was authored by a **separate, independently-dispatched party** — not
the original implementer (who had already opened the PR and moved on), and
not the reviewing/merging session (which explicitly declined, in writing,
per the interim rule). This is a caveat worth stating plainly rather than
glossing over: GitHub's own record cannot distinguish "session identity" by
commit author or committer (every commit and review in this project's
history carries the same account, `nhuthuynh`/the same bot committer
identity — ADR-0016's own Context section already flags this as a standing
limitation of this environment's single-authenticated-account setup). The
determination above rests on the reviews' own textual content — a reviewer
that writes "I'm not touching the code myself" and then, in a later review,
describes a fix as something that simply "now exists" rather than something
"I added" is the strongest evidence this record can produce, and it points
the same direction consistently across both reviews.

**Scored: this is a genuinely new, fourth shape — "PR existed, review found
a real gap, the reviewing session correctly declined to author the fix
itself (following the interim strict rule exactly as ADR-0016 currently
requires), a separate dispatched party authored and pushed the fix, and the
reviewing session re-verified and merged."** It does not test D2's actual
question (whether the *same* session may review and author) at all — it is,
on the read above, a clean instance of **option (a), strict enforcement,**
succeeding at bounded cost: one extra review cycle, roughly fourteen minutes
wall-clock (`15:09:17Z` to `15:23:09Z`), a single well-scoped commit. This
matters beyond bookkeeping: ADR-0016's own "What would change this" section
names *"a second reviewing party that can be dispatched reliably and
independently, which would make option (a) nearly costless and reduce D2 to
a formality"* as a genuine changed circumstance — not a resolution, but
something the ADR itself says should be *reported*. T29.2 is real evidence
of exactly that circumstance obtaining, at low cost, for the first time this
project has recorded it happening with a real (not hypothetical) gap. This
is reported here as evidence for the question, per ADR-0016's own explicit
instruction that a further instance "is more evidence for the question, not
an answer to it" — it does not resolve D2, and this retro does not treat it
as doing so.

**One further precision, mirroring T28 retro's own caution.** The depth and
correctness of both reviews (independent DB reproductions, direct file
reads, an honest disclosure of the PR-body gap) is a separate axis from
either the "exercised, no fix needed" scoring or the new fourth-shape
scoring — a thorough review is not itself evidence for or against any of
D2's options, only a fact about how well this sprint's reviews were
conducted.

## 8. The shared-checkout hazard — near-miss, plus one real
institutionalization gap found underneath it

**PdE/PE, scored against the T9 precedent (`docs/LESSONS.md`'s `## T9 sprint
retro` entry and `docs/process/t9-retro.md` finding 1, read in full this
retro) — a genuine judgment call, not defaulted to either extreme.**

**What happened, read from both PR bodies directly.** T29.1's PR body
("Toolchain — run in a fully isolated `git worktree`") discloses: *"T29.2
(Social Play) was being developed concurrently in the *same* shared
checkout during this session… leaving uncommitted `internal/socialplay/**`/
`cmd/server/main.go` edits in the working tree that don't belong to this
PR."* Its remedy: build a disposable `git worktree` off the branch's parent
commit, apply only its own diff there, verify via `git diff --stat` that
exactly 24 files under its own scope were touched. T29.2's PR body
("Isolation from the concurrent T29.1 work") discloses the same collision
from the other side and took a different remedy: branch directly from the
shared parent commit (`8394e97`) via raw git plumbing, never touching the
shared checkout's working tree at all, then independently re-confirmed via
`git diff --name-only 8394e97 origin/feature/t29.2-socialplay-identity-conformance`
that all 35 changed files fall under its own scope, zero under
`internal/competitions/**`. **Both PRs' claims are independently confirmed
by this retro**: `pull_request_read(get_files, #239)` lists 24 files, all
under `internal/competitions/**`/`internal/payments/app/…regression_test.go`/
`cmd/server/main.go`/one migration/one proto — zero under
`internal/socialplay/**`; `pull_request_read(get_files, #240)` lists 36
files (35 the PR body claims plus the later test-file commit), all under
`internal/socialplay/**`/`internal/payments/**`/`cmd/server/main.go`/one
migration/one proto — zero under `internal/competitions/**`. Neither
branch's pushed content was corrupted by the other; this retro confirms it
directly rather than trusting either PR's own claim.

**The T9 precedent, restated precisely rather than gestured at.** Five
implementers, one unisolated checkout, 19-minute dispatch window, real
collisions ("they stepped on each other"), self-corrected by all five
independently discovering the same worktree fix, **zero of the five wrote it
down** until the retro — scored by QA in `docs/process/t9-retro.md` as a
genuine data-loss race whose benign outcome on one run was not evidence
about the next, with PE recording the opposing view that a self-correcting
system behaving as designed does not need a written line. **Not resolved as
a disagreement** — but both sides agreed on the concrete remedy: *"dispatch
isolation becomes an explicit Ceremony 2 checklist item."*

**Two ways T29's instance differs from T9's, both favorably.** First, both
T29.1 and T29.2 detected and self-isolated **before** any actual collision
manifested as corrupted state — T29.1 built a worktree proactively upon
discovering concurrent edits already present, and T29.2 avoided the shared
checkout's working tree from the start by branching via raw git plumbing;
neither is a post-hoc recovery from an actual stepped-on-each-other
collision the way T9's five agents' fix was. Second, **both wrote it down**
— in their own PR bodies, at first-instance time, not only if a retro later
asked — which is precisely the gap QA's T9 finding scored as the real cost
("five independent rediscoveries… five times the cost of one written
line"). By that same yardstick, T29 paid the isolation cost twice (not
five times) and recorded both instances without being asked.

**The real gap, found by checking whether T9's own remedy was ever actually
durably adopted — it was not.** `grep -rli "isolat" docs/process/` returns
many sprint retros and plans from T10 through T28, but a direct content
`grep -ni "isolat"` against **`docs/process/sprint-process.md` itself
returns zero matches** — the "dispatch isolation becomes an explicit
Ceremony 2 checklist item" remedy T9's retro adopted was never written into
the durable process document as a standing rule, unlike "the same-wave
shared-interface verification rule" or "the dependency-completeness check,"
both of which earned their own named `sprint-process.md` sections from
similarly-scoped retro findings. The nearest thing to institutionalization
this retro could find is descriptive, not prescriptive: T13's sprint plan
(`docs/process/t13-sprint-plan.md`, its "Dependency order and dispatch
waves" section) explicitly labels its parallel waves *"(parallel, ≤5
implementers, worktree-isolated)"* — a real instance of the checklist item
being honored in a plan's own text. **T29's own plan carries no such
annotation anywhere**: `grep -ni "isolat" docs/process/t29-sprint-plan.md`
returns zero matches, and its "Waves" section reads only *"Wave 1 (only
wave): T29.1 and T29.2, dispatched together"* — no isolation note, despite
this being exactly the two-parallel-implementer shape the T9 remedy targets.
**This is the actual mechanism behind T29's near-collision**: not that the
checklist item was consciously skipped, but that it was never durably
written down anywhere Ceremony 2 would be prompted to apply it, so a sprint
plan four years — thirteen sprints — after T9 shows no trace of it, and the
same collision shape recurred, prevented only by both implementers'
individual diligence rather than by planning-time process.

**Scored: a near-miss, not a repeat T9-grade incident, but the near-miss
uncovers a real, separately-scoreable process-institutionalization gap that
does warrant its own `docs/LESSONS.md` line — not for the collision itself,
which both sides self-corrected and disclosed exactly as the T9 remedy would
want, but for the discovery that the remedy was never actually durably
adopted.** Writing a `docs/LESSONS.md` entry for "two agents collided in a
shared checkout, no harm done" alone would be exactly the "bookkeeping, not
risk reduction" PE argued against in T9's own recorded disagreement — the
data-loss risk was genuinely retired *this time* by both agents' own
diligence, and a third near-miss on the same shape without a systemic fix
would be the point at which "no work was lost" stops being reassuring, per
QA's original T9 argument. What tips this into "warrants an entry" rather
than "note and move on" is specifically the second half: this project has a
real, working pattern (a durable `sprint-process.md` section) for exactly
this kind of retro finding, used successfully for at least two other T9/T15/T16-era
findings, and it was not used here — that is a fixable, systemic gap
distinguishable from "an agent made a judgment call," and it is exactly the
shape of finding this project's own `docs/LESSONS.md` exists to make durable
rather than let re-erode a second time.

## 9. The empty-PR-body incident for #240 — adequately caught by existing
process, one cheap safeguard recommended

**QA and PdE.**

**What happened, confirmed from the review's own first-person account rather
than inferred.** PR #240's first review (`id 4946479055`) states directly:
*"At review time this PR's description was essentially empty (only the
attribution footer) despite your own final report to me describing
substantial content… I've already asked a follow-up session to restore the
full body."* The PR's **current** body (fetched this retro via
`pull_request_read(get)`) is long and substantive — meaning the
reconstruction did happen, consistent with the review's own account, though
GitHub's API does not expose body-edit history directly, so this retro
relies on the review's contemporaneous first-person statement (written
before the fix, when the reviewer had direct knowledge of the empty state)
rather than a diffable record. That statement is corroborated indirectly by
the **second** review's own observation: *"the PR body's 'Surprises /
deviations' item 2 still reads as if the test gap is unresolved (written
before commit `6067fef` landed)"* — i.e. the second reviewer independently
noticed the body's content predates the fix commit, which is only possible
if the body was substantively (re)written at some point between the two
reviews, consistent with the first review's own account of ordering a
reconstruction.

**Scored: adequately caught by existing process, not a new gap.** The
failure mode here — an implementer's own final chat-session report
describing content that never reached the actual PR description — sits
entirely upstream of anything `sprint-process.md`'s per-ticket DoD or this
project's review discipline is designed to catch *after* the fact, and it
was in fact caught, by the very next step the process already has: review.
No merge happened against the empty body; the gap was found, a targeted
follow-up session restored it, and the restoration was itself verified (the
second review's own before/after comparison). This is the review step doing
exactly its job, not a near-miss that slipped through — contrast with §8,
where the systemic remedy for the *underlying* hazard had quietly gone
missing from the process document; here, the relevant safeguard (review
before merge) was present, current, and worked on the first attempt.

**One concrete, cheap safeguard recommended for T30 regardless, because
catching a gap at review time is strictly worse than not creating it.**
Before an implementer session reports a ticket "done" and hands off to
review, it should perform one fresh `pull_request_read(get)` against its own
just-opened PR and confirm the returned `body` is non-empty and matches what
it intended to post — a single read call, not a new process stage, targeting
specifically the failure class observed here (a chat-session report and the
PR API call that was supposed to carry the same content silently
diverging). This is deliberately narrow: it does not ask every implementer
to re-verify every field of every PR, only the one field this incident shows
can silently go missing between an agent's own belief about what it did and
what the API actually recorded.

## 10. Issue sweep, reconciled arithmetically

Covered in full in §1 and §5. Summary: **8** open at T29 Ceremony 1's start
(carried from T28 retro) → **+1** (opens #237, mid-Ceremony-1) → **−2**
(#164, #237 both close this sprint) → **7** open now, matching the live
`totalCount` exactly. The task brief's assumption of "should be down to 6"
undercounts by the same drafting gap traced in §5 (T29 plan §A5's ranked
table silently drops #124); the correct, live, arithmetically-reconciled
figure is **7**.

## 11. Running counters

**PO.**

**D1's consecutive-sprint-silence counter: confirmed at sixteen, not
incremented a second time within this sprint.** T29's own Ceremony 1 already
computed this as **sixteen consecutive sprints (T14 through T29)** when the
sprint opened with #144 still carrying only its original T14.3 comment. This
retro re-checked #144 live (§6): still exactly one comment, unchanged. Per
T28 retro's own established convention (a per-*sprint*, not per-*ceremony*,
counter — the retro re-confirms the same count Ceremony 1 already set within
the same sprint, rather than incrementing a second time), this retro
confirms the count **holds at sixteen**. It becomes **seventeen** only if T30
opens with #144 still uncommented.

**The post-T28.1 backlog-composition counter (T28 retro finding 8): retired
at 2, per T29 plan §A1's own disposition, not resurrected here.** T29's own
Ceremony 1 incremented it to 2 and then correctly retired it in the same
breath, since filing #237 changed the tracked set's composition before this
retro could re-check it unchanged — re-confirmed here rather than silently
inherited: the set that counter tracked (`#124, #126, #130, #134, #144,
#145, #149, #164` — narrowed-but-open) no longer exists; #164 is now closed.
Continuing to increment it would be measuring a set that no longer describes
the codebase.

**A new counter is proposed, not silently assumed — mirroring the exact
precedent T28 retro's finding 8 set.** The backlog now has a stable-again
composition worth tracking for drift: **the post-T29 backlog-composition
count** — consecutive live checks finding the *current* 7-issue set
(#124, #126, #130, #134, #144, #145, #149 — all "other," #164 and #237 both
now closed, D1/D2 both still open) unchanged. This retro is the first such
check, since it is the first live check performed after `2026-08-16T15:23:16Z`
(the later of #164's/#237's two `closed_at` timestamps), and it found the set
unchanged in composition from the moment both closures landed (§5) — so this
counter **starts at one, established by this retro**, following the identical
per-ceremony-increment shape the T28.1-era counter used rather than a fresh
invention. It becomes two if T30 Ceremony 1's own sweep again finds the same
7-issue set unchanged. **Deciding this is not premature**: the alternative —
waiting until the set has "settled for a while" before starting a counter —
is exactly backwards, since T28 retro's own precedent starts the counter at
the retro immediately following the set's stabilization, not after some
further waiting period, and that is the position this retro is in now.

## 12. Label-taxonomy conformance — one real gap found and corrected live

**QA, per `sprint-process.md`'s Label taxonomy section (role:/type: mandatory
on every issue), checked directly rather than assumed from the plan's own
account.**

`issue_read(get_labels, #164)` returns `role:principal-engineer`,
`type:chore` — conformant, unchanged from before this sprint.
`issue_read(get_labels, #237)` returned **zero labels** at the start of this
retro — a real conformance gap in the same family T13/T14 previously caught
on #165/#167/#168, missed at filing time (T29's own Ceremony 1 filed #237
mid-ceremony but never labelled it) and not caught by either PR's own review,
since neither review's scope was issue-tracker bookkeeping. **Corrected live
by this retro**: `issue_write` applied `role:principal-engineer`,
`type:chore` (matching #164's own classification, since #237 is the cross-
context regression #164's own conformance work fixed as a side effect, and
shares its owning role and kind), and a comment (`id 5308199250`) was posted
to #237 disclosing the gap and the correction, per this project's standing
practice that retros perform live issue-tracker bookkeeping directly (the
same practice T15's retro used to close #185/#137 and correct #149) — this
is not a code change and does not implicate CLAUDE.md rule 9's PR-only
requirement, which governs commits to the repository, not GitHub issue
metadata.

---

## The sprint goal, scored

> *"Close both remaining thirds of #164 — Social Play and Competitions — as
> two independent, same-wave tickets, following the reference pattern T28.1
> proved on Payments and re-verifying rather than re-deriving ADR-0017's
> Decisions 1–3… Do this not only to finish backlog-hygiene conformance but
> because independently tracing T28.1's merged diff this ceremony found a
> live regression it introduced… T29.1/T29.2 fix this as a side effect, with
> no Payments-side code change, by finishing the identifier-space migration
> those reads depend on. Re-verify the other 6 issues' blockers live and find
> them unchanged."*

**Every substantive clause holds, independently re-verified rather than
taken from the plan's or either PR's own account — with one clause (the "6")
corrected rather than restated.** Both tickets shipped their funnel change
and migration together with no window either comparison could break in
(§2). Both backfills were mutation-checked per CLAUDE.md rule 10, by
verification shapes that legitimately and correctly differ (§3). Both
migrations' `NOT NULL`/nullable branches were the correct one, for two
different and independently-verified reasons (§4). #164 and #237 both closed
in full, correctly, with live-state-checked comments (§1). The other
issues' blockers hold — but there are **7** of them, not 6, a real drafting
gap in the plan's own §A5/DoD traced to its source rather than silently
inherited (§5). Neither D1 nor D2 was answered as a formal ADR decision
(§6). T29.1's review scores as D2's "exercised, no fix needed" shape;
T29.2's scores as a genuinely new, fourth shape distinct from both of D2's
named possibilities and from T28 retro recommendation 5's anticipated third
one — real evidence for ADR-0016's own "changed circumstance" clause, not a
resolution of it (§7). The shared-checkout hazard is a near-miss with one
real, separately-scoreable institutionalization gap underneath it, now
recommended for a `docs/LESSONS.md` entry (§8). The empty-PR-body incident
was caught cleanly by existing review process, with one cheap safeguard
recommended regardless (§9).

**The agreed honest sentence, which T30's Ceremony 1 should carry into
`HANDOFF.md`'s Task-backlog entry** (`sprint-process.md` Ceremony 1 item 3
requires the retro's form, not a stronger one):

> T29 shipped two tickets, T29.1 (Competitions, 13 points, PR #239) and
> T29.2 (Social Play, 21 points, PR #240), 34 points total, closing both
> remaining thirds of issue #164 (all three contexts — Payments/T28.1,
> Competitions/T29.1, Social Play/T29.2 — now ADR-0014/ADR-0017 conformant)
> and, as a side effect with no Payments-side code change in either ticket,
> closing issue #237 (the live authorization regression T28.1 introduced in
> `authorizeGameRecording`/`authorizeCompetitionEntryRecording`, found by
> this same sprint's own Ceremony 1 tracing PR #235 end to end). This retro
> independently re-verified — not trusted — every load-bearing claim: both
> tickets' funnel changes and backfill/column-type migrations landed
> together as one reviewed unit with no window either comparison could break
> in; both backfills were mutation-checked per CLAUDE.md rule 10 by two
> legitimately different verification shapes (T29.1's nullable migration
> demonstrated directly, T29.2's `NOT NULL` migration demonstrated via a
> two-part proof — the backfill mechanism in isolation, and the shipped
> migration's loud, not silent, failure on a real orphan — both independently
> reproduced a third time by this retro against a real local Postgres with
> this retro's own seed data); both migrations' `NOT NULL`-vs-nullable
> branches were independently confirmed correct for two different reasons
> (T29.1's forced by a `PRIMARY KEY`/rule-10 structural conflict, T29.2's
> earned by an independently-reproduced, deployment-model-guaranteed
> zero-orphan empirical case); both Payments-side regression tests use
> genuinely non-matching, mutually distinct fixture values that would have
> caught #237, confirmed by reading both test files directly; the backlog's
> other issues' blockers hold, re-verified live — but the correct count is
> **7**, not the 6 both the task's own framing and the sprint plan's own DoD
> item assumed, a drafting gap in the plan's own §A5 ranking table (silently
> dropping #124) traced to its source and corrected here rather than
> propagated; neither D1 nor D2 was answered as a formal ADR decision this
> sprint, both files' git history and `## Status` sections read directly;
> T29.1's review scores as D2's "exercised, no fix needed" shape (the
> seventh such instance), while T29.2's review — a real gap found, changes
> formally requested, the fix authored by neither the implementer nor the
> reviewer but a separately-dispatched party, then re-verified and merged —
> scores as a genuinely new, fourth shape, argued from ADR-0016's own
> definitions rather than forced into an existing bucket, and reported as
> real evidence for ADR-0016's own "a second reviewing party that can be
> dispatched reliably… would make option (a) nearly costless" clause, not a
> resolution of D2. The shared-checkout collision both tickets independently
> self-detected and disclosed is scored as a near-miss, not a T9-grade
> incident (no collision-caused corruption occurred, both sides wrote it
> down at first-instance time rather than only if asked) — but a real,
> separately-scoreable gap was found underneath it: T9's own "dispatch
> isolation becomes an explicit Ceremony 2 checklist item" remedy was never
> durably written into `sprint-process.md` itself and has silently eroded
> over thirteen sprints, recommended here for a `docs/LESSONS.md` entry and
> a durable process-document fix, distinct from the collision incident
> itself. The empty-PR-body incident for #240 is scored as caught cleanly by
> existing review process (not a new gap), with one cheap safeguard
> recommended for T30 regardless (an implementer verifying its own
> just-opened PR's body is non-empty via a fresh read before reporting
> done). D1's consecutive-sprint-silence counter holds at sixteen (T14
> through T29, confirmed rather than incremented a second time within this
> sprint); the post-T28.1 backlog-composition counter is retired at 2, per
> T29 plan's own disposition; a new post-T29 backlog-composition counter is
> proposed, starting at one, established by this retro. One label-taxonomy
> conformance gap was found (#237 filed with zero labels) and corrected live
> by this retro, per this project's standing practice of retros performing
> issue-tracker bookkeeping directly.

---

## Recommendations for T30's Ceremony 1 and 2

1. **Re-run the merged-fix sweep as the authoritative moment**, per the
   standing rule — re-verify the open count (7) from the live API rather
   than trusting this retro's table (§1, §10).
2. **Add a `docs/LESSONS.md` entry for the shared-checkout
   institutionalization gap found in §8** — not for the T29.1/T29.2
   collision itself (a near-miss, both sides self-corrected and disclosed),
   but for the discovery that T9's "dispatch isolation becomes an explicit
   Ceremony 2 checklist item" remedy was never durably written into
   `sprint-process.md` and has silently eroded for thirteen sprints. Pair it
   with a concrete fix: give dispatch isolation its own named
   `sprint-process.md` section, the same way "the same-wave shared-interface
   verification rule" and "the dependency-completeness check" earned theirs
   from similarly-scoped prior findings — not another one-off mention in a
   sprint plan's own prose, which is exactly the shape that already failed
   to persist once.
3. **Adopt the cheap PR-body-verification safeguard from §9** for T30
   onward: an implementer confirms its own just-opened PR's `body` is
   non-empty via a fresh `pull_request_read(get)` before reporting a ticket
   done.
4. **Continue the post-T29 backlog-composition counter proposed in §11**,
   incrementing it to two if T30 Ceremony 1's own live check again finds the
   same 7-issue set unchanged — do not reintroduce or extend the retired
   post-T28.1 counter.
5. **When drafting a Sprint-level DoD's issue count, derive it from the
   ceremony's own §A1/§A3 sweep tables, not from a separately-maintained
   ranking table** — §5's finding traces the "6 vs 7" discrepancy
   specifically to T29 plan's §A5 table silently dropping an issue its own
   §A3 table correctly carried two sections earlier. A DoD line that counts
   issues should cite the sweep's own arithmetic directly rather than
   re-deriving a second count from a document written for a different
   purpose (ranking work, not enumerating the backlog).
6. **If D2's fourth shape (§7) recurs, score it against this retro's
   definition rather than re-deriving one from scratch** — and if the
   *same*-session-authors-its-own-review's-fix shape (T28 retro
   recommendation 5's original third possibility) is ever actually observed,
   score that as a fifth, still-distinct shape rather than conflating it
   with this sprint's fourth.

## Sprint-level Definition of Done — scored against what T29's own plan asked

Per `docs/process/t29-sprint-plan.md`'s "Sprint-level Definition of Done,"
the items owed at this retro, restated here with their answers:

1. **T29.1 and T29.2 merged per the per-ticket DoD?** **Yes** — acceptance
   criteria met, both reviewed (§7), `make test-domain`/`make
   test-adapters`/`make fmt-check`/`make vet-integration`/`make
   gate-coverage` all re-run clean directly against the merged tree this
   retro (12/12; 26 of 27 report `ok`, the sole exception a pre-existing
   `[no test files]` package unrelated to this sprint, including the new
   `internal/socialplay/adapter/identity` and `internal/competitions/adapter/identity`
   both reporting `ok`; clean; clean; 45/45), approved and merged by the
   delegated gate, never self-merged by an implementing session distinct
   from the reviewer.
2. **The merged-fix issue sweep run and reported?** **Yes** — §1; T30's
   Ceremony 1 remains authoritative.
3. **#164 closed in full, with an explicit comment naming both merged PRs,
   correctly checking live state first?** **Yes** — §1; PR #240's own merge
   comment (`id 5308164898`) names all three PRs (#235, #239, #240) and was
   posted after confirming #164's live state per instruction 10.
4. **#237 closed in full, same discipline?** **Yes** — §1; comment `id
   5308165079`, same live-state-check discipline, posted 3 seconds after
   #164's closing comment.
5. **(a) Funnel + backfill/column-type change shipped together, no
   window, both tickets?** **Yes, both** — §2.
   **(b) Backfill mutation-checked per CLAUDE.md rule 10, both tickets?**
   **Yes, both — by legitimately different verification shapes** — §3.
   **(c) `NOT NULL`/nullable the correct branch per Decision 3, both
   tickets?** **Yes, both — for two independently-verified, different
   reasons** — §4.
   **(d) Payments-side regression tests use non-matching fixtures and would
   have caught #237?** **Yes, confirmed by reading both test files
   directly** — verified in this retro's independent file reads
   (`internal/payments/app/competition_entry_identifier_space_regression_test.go`,
   `internal/payments/app/service_test.go`'s two new
   `ResolvedIdentifierSpaces` tests).
   **(e) The other issues' blockers still hold, re-verified live?** **Yes —
   but there are 7, not 6, corrected in §5.**
   **(f) D1 or D2 answered mid-sprint?** **No, neither** — §6.
   **(g) Which of D2's shapes does each ticket's real review land in, per
   ticket?** **T29.1: "exercised, no fix needed" (the seventh instance).
   T29.2: a genuinely new, fourth shape, distinct from both D2-named
   possibilities and from T28 retro recommendation 5's anticipated third —
   real evidence for ADR-0016's own "changed circumstance" clause, not a
   resolution** — §7.
6. **Not scoreable by T29 and deliberately not pre-empted:** D1 and D2
   remain the user's own decisions.
7. **Retro in `docs/process/t29-retro.md`** (this document), indexed by a
   `## T29 sprint retro` stub in `docs/LESSONS.md`. `HANDOFF.md`'s Docs
   index and Task-backlog entry updated by this same PR.

Retro complete. Issue-tracker actions this ceremony: **three** — #237's
missing labels corrected plus a disclosure comment (§12), all performed live
by this retro; #164's and #237's closing comments were already posted by the
merging session before this retro began (§1), not repeated here. Open count
at ceremony start: **7**. Open count now: **7** (unchanged — nothing this
retro did opens or closes a tracked issue; the label correction and its
comment do not change any issue's open/closed state). No incident rises to
the T9-precedent severity this sprint, but **one `docs/LESSONS.md` entry is
recommended** (§8, recommendation 2) — for a process-institutionalization gap
this retro found, not for the near-miss collision itself, which both
implementing sessions self-corrected and disclosed without prompting.

Toolchain re-verified directly against the merged tree this retro (after
regenerating `internal/gen`, gitignored and not checked in): `make
test-domain` — 12/12 packages green; `make test-adapters` — 26 of 27
packages report `ok` (the one exception, `internal/identity/adapter/postgres`,
reports `[no test files]`, pre-existing and unrelated to this sprint); `make
fmt-check` —
clean; `make vet-integration` — clean; `go run ./cmd/gatecoverage` — `OK —
all 45 package(s) with runnable tests are executed by "ci-checks"`, matching
PR #240's own second review's reported count exactly. Both migrations'
mutation checks independently reproduced a third time against a real local
Postgres 16 instance with this retro's own seed data (§3), agreeing exactly
with both PRs' own accounts and both reviews' independent reproductions.

Per `sprint-process.md`'s established convention: **`HANDOFF.md`'s T29 row is
corrected by this same PR**, since its Retro and Reviews cells were the only
fields still reading "not yet written"/"not yet opened."
