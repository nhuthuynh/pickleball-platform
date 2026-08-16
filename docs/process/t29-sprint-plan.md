# T29 Sprint Plan — Ceremonies 1 + 2

Backlog refinement (Ceremony 1) and sprint planning (Ceremony 2) per
`docs/process/sprint-process.md`, six-role team (briefs:
`docs/agent-operating-handbook.md` Part B). Held against
`docs/process/t28-retro.md` (PR #236, merged `2026-08-16T14:12:15Z`),
`HANDOFF.md` including its Docs index, `CLAUDE.md`, and the live PR/issue
state of `nhuthuynh/white-label` (GitHub-side name `pickleball-platform`), at
branch tip `c975f21` (T28.1's own merge — T28's retro doc, PR #236, is a
docs-only commit on top of it), re-fetched and confirmed matching
`origin/claude/go-backend-pickleball-7up34j` before this ceremony's branch
was cut.

**This ceremony departs from the routine confirm-and-report shape a second
time running.** T28 broke the eight-sprint 0-ticket run by reclassifying
#164 and shipping Payments' conformance as a reference implementation. This
ceremony's own re-verification of that reference implementation — tracing
PR #235 end to end rather than trusting its own "why one PR" account, per
this project's standing rule that every factual claim is checked against the
live repository — found a live, previously-undetected regression the ticket
and its review did not catch (§B1). That finding, not backlog hygiene alone,
is what this ceremony's scope decision turns on.

## §A0 — Correcting T28's Docs-index row and Task-backlog narrative, and a
correction one sprint further back than usual

Per `sprint-process.md`'s standing convention (a retro PR never updates the
Docs-index row that points at it) and per `docs/process/t28-retro.md`'s own
closing line, **performed directly in `HANDOFF.md` by this same PR**:

| Field | What the row now said | What it now says | Why |
|---|---|---|---|
| T28 Reviews cell | `PR #234 (Ceremony 1/2 doc) → PR #235 (T28.1, "partial fix for #164") → PR (retro doc), … "13:19:47Z" → "13:59:00Z" → retro doc's own merge` | `… → PR #236 (retro doc), … "13:19:47Z" → "13:59:00Z" → "14:12:15Z"` | T28's retro (PR #236, `docs/process/t28-retro.md`'s own closing paragraph) claimed "`HANDOFF.md`'s T28 row is corrected by this same PR" — but a PR structurally cannot cite its own merge PR number before it exists (the same reason `sprint-process.md`'s Ceremony-1 convention exists at all), and the row it actually shipped left the number blank ("PR (retro doc)"). Filled in now that the number is knowable, per `list_pull_requests`: PR #236, `merged_at: 2026-08-16T14:12:15Z` |

**A second, one-sprint-further-back correction, found only because this
ceremony re-read every row rather than only the immediately preceding one.**
T28's own Ceremony 1 (`docs/process/t28-sprint-plan.md` §A0) corrected T27's
row's Retro/Reviews cells, but its own table left the retro doc's PR number
blank too: `"PR #232 (Ceremony 1/2 doc) → PR (retro doc, filed as part of the
same T27 cycle)"`. The row that actually landed in `HANDOFF.md` carried the
identical gap — `"PR #232 (Ceremony 1/2 doc) → retro doc, …"`, no number —
and nothing since has fixed it, because `sprint-process.md`'s convention
only obligates the *immediately following* ceremony to correct the
*immediately preceding* row; T28 discharged that obligation for T27's row
in every other respect and simply under-filled this one field. Corrected
here: `list_pull_requests` gives T27's retro doc as **PR #233**,
`merged_at: 2026-08-16T11:30:25Z`. **Stated so this isn't read as license to
skip re-checking older rows going forward:** this was only found because
this ceremony traced #164/T28.1 back through the full PR history rather
than trusting either the T27 or T28 row's own account, not because Ceremony
1 now owns an open-ended audit of every historical row. It is reported here
as an incidental find, not a new standing obligation.

**A new T29 row is added** in the same before-the-sprint form, Retro and
Reviews honestly marked "not yet written" / "not yet opened".

**ADR-status citation form** (T24 retro's convention, extended by T28
retro's finding 6 to cover ADR-0017 — still in force):

- ADR-0015 `## Status`: **"Escalated — awaiting product decision. This ADR
  decides nothing."**
- ADR-0016 `## Status`: **"Escalated — awaiting the user's decision. This
  ADR decides nothing."**
- ADR-0017 has **no separate `## Status` section** (it is Accepted, not
  escalated — mirrors ADR-0013/ADR-0014's shape). Its status is cited from
  its frontmatter bullet only: **"Status: Accepted."** Re-verified this
  ceremony: `docs/adr/0017-*.md` read in full, headings run Context →
  Decision 1 → Decision 2 → Decision 3 → Consequences → "What this ADR does
  not decide," no `## Status` heading anywhere.

## §A1 — The merged-fix issue sweep

**Step 1.** `list_issues(state: OPEN)` at ceremony start → **`totalCount: 8`**:
#124, #126, #130, #134, #144, #145, #149, #164. Identical set to T28's own
sweep and T28's retro sweep before it — #164 included, narrowed but still
open by design (T28.1's PR title, "partial fix for #164").

**Step 2.** `open_at_end_of_T28's_retro (8) − closed_since (0) +
opened_since (0) = 8`. Matches the live `totalCount: 8` exactly.

**Step 3.** `list_pull_requests(state: closed, base:
claude/go-backend-pickleball-7up34j, sort: updated, direction: desc)` — most
recent entry before this ceremony's branch cut is PR #236 (T28's retro doc,
`merged_at: 2026-08-16T14:12:15Z`); nothing merged since. Its title/body
name no issue as closed (a retro doc, not a fix). **Zero new closures to
cross-reference.**

**Sweep result: clean, continuing the unbroken run since T15 — the third
sprint running with a genuinely non-static backlog composition (T28
reclassified #164; this ceremony files #237, discussed below) that is still
arithmetically clean.** T30's Ceremony 1 re-runs this sweep in full
regardless.

**Post-T28.1 backlog-composition counter (T28 retro finding 8, recommendation
3): increments to 2.** The 8-issue set, as it stood after T28.1 merged
(`13:59:11Z`), is unchanged at this ceremony's own live check — the count
that began at 1 (T28's own retro, the first check after that timestamp) is
now 2 (this ceremony, the second). It will not extend to 3 automatically:
this ceremony **files a new issue (#237, §B1)**, which changes the tracked
set's composition the moment it lands — the identical "a counter that means
'found the set unchanged' stops incrementing the moment the set changes"
reasoning T28 plan §A3.1 applied to the old backlog-static-check counter.
**This counter is therefore also retired at 2, not carried into T30 as if
nothing changed** — T30's Ceremony 1 starts a fresh count (or does not,
PE/PM's call at that time) against the **post-#237** composition, the same
way T28 retired the pre-#164-reclassification counter rather than pretend
the set it measured was still the set on the ground.

## §A2 — T28 retro's five recommendations, dispositioned

| # | Recommendation | Disposition |
|---|---|---|
| 1 | Re-run the merged-fix sweep as the authoritative moment | **Executed** (§A1) — re-verified from the live API, not trusted from the retro's table |
| 2 | Consider posting a short comment to #149 disclosing `booking_host_id`'s subject→`User.ID` shape change | **Executed** — posted this ceremony (`issue_read`-confirmed: comment id `5307890924`), and used as the occasion to also cross-reference the newly-filed #237, which is a different mechanism from #149's own caller-supplied-fact gap and does not narrow it |
| 3 | Continue the post-T28.1 backlog-composition counter, incrementing to 2 if unchanged | **Executed, then retired on the same finding that changed it** — see §A1. Incremented to 2 at this ceremony's own live check, then retired at 2 rather than carried forward, since this same ceremony's own #237 changes the set's composition before T30 would otherwise re-check it |
| 4 | Re-verify ADR-0017's Decision 2/3 rulings still hold rather than re-deriving them, when Social Play/Competitions are taken | **Executed, with one gap found and corrected** — §B4: Decisions 1 and 3 hold and extend cleanly; Decision 2's own column table is **incomplete** (omits `game_admins`/`competition_admins`), corrected in this plan and disclosed on #164 itself (§B7) rather than silently patched over |
| 5 | Score a reviewer-authored fix on a real, non-recovery PR as a genuinely new D2 instance if one occurs | **Not yet applicable** — no PR has been reviewed this ceremony; scored at T29's own retro against whatever T29.1/T29.2's reviews actually do |

## §A3 — Routine re-verification (all 8 issues, live)

Per the task's own instruction: every open issue re-read via `issue_read`,
not trusted from T28 retro's table without re-checking `updated_at`/comment
counts directly.

| Issue | `updated_at`/comments at T28 retro | `updated_at`/comments now (live, this ceremony) | Changed? |
|---|---|---|---|
| #164 | `2026-08-16T13:59:11Z`, 2 comments | `2026-08-16T13:59:11Z`, 2 comments | No |
| #149 | `2026-08-15T16:56:58Z`, 3 comments | `2026-08-16T…` (this ceremony's own comment, id `5307890924`), 4 comments | **Yes — by this ceremony's own action** (§A2 item 2), not an external change |
| #145 | `2026-08-15T05:01:29Z`, 1 comment | `2026-08-15T05:01:29Z`, 1 comment | No |
| #144 (D1) | `2026-08-15T07:01:03Z`, 1 comment | `2026-08-15T07:01:03Z`, 1 comment | No |
| #134 | `2026-08-14T16:37:49Z`, 0 comments | `2026-08-14T16:37:49Z`, 0 comments | No |
| #130 | `2026-08-14T16:30:25Z`, 0 comments | `2026-08-14T16:30:25Z`, 0 comments | No |
| #126 | `2026-08-14T16:12:26Z`, 0 comments | `2026-08-14T16:12:26Z`, 0 comments | No |
| #124 | `2026-08-14T16:25:34Z`, 1 comment | `2026-08-14T16:25:34Z`, 1 comment | No |

**Every issue's full body re-read this ceremony (`issue_read(get)`), not
just its `updated_at`/comment count — per the task's explicit instruction not
to trust T28 retro's table.** #144, #145, #149, #124 all re-read in full;
their blockers are unchanged from T28's own characterization:

- **#144 (D1):** still zero authorization on `CancelBooking`/`CreateBooking`,
  blocked on a product decision (who "owns" a public quote-and-book
  Booking) — ADR-0015, unanswered.
- **#149:** still `booking_host_id` as Payments' one remaining
  caller-supplied ownership fact, blocked on D1 exactly as before — this
  ceremony's own comment (§A2 item 2) adds record-completeness, not a
  change to the blocker itself.
- **#145:** still blocked on a real IdP tenant's non-uuid `sub` claim, which
  exists nowhere in this environment — re-confirmed structurally different
  from #164's now-two-thirds-closed backfill (§B6 of T28's own plan already
  drew this line; nothing this ceremony found disturbs it).
- **#124:** still needs Product Owner input on cascade semantics
  (court-release, refund automaticity, waitlist handling) before
  `Game.Cancel()` can cascade.
- **#134, #130, #126:** unchanged — real assistive-technology hardware this
  environment cannot provide; an open product question on `no_show_fee`
  refunds; Product Owner input needed on a real per-Game price field,
  respectively.

**Migration-tooling classification (`golang-migrate`/`goose`):** re-checked,
`HANDOFF.md`'s Cross-cutting section still carries exactly the one line
every prior ceremony has quoted. `ls docs/adr/` ends at `0017`; `ls
db/migrations/` ends at `0024` (before this ceremony's own tickets claim
`0025`/`0026`, §B7). **Still correctly unticketed, unchanged.**

**Toolchain, re-run directly against the unmodified tree before any planning
text was written:** `make test-domain` — 12/12 packages green. `make
gate-coverage` — `OK — all 43 package(s) with runnable tests are executed by
"ci-checks"`, matching T28 retro's own count exactly (no drift from an
unmodified tree).

## §A4 — Migration-header-ownership check

`db/migrations/` ends at `0024`. Two new migrations are pre-assigned, one
per ticket, no collision possible since neither touches the other's tables:

- `0025_competitions_identity_conformance.sql` — **T29.1 only**
  (`competitions.host_id`, `competition_entries.player_id`,
  `competition_admins.user_id`/`assigned_by`).
- `0026_socialplay_identity_conformance.sql` — **T29.2 only**
  (`games.host_id`, `registrations.player_id`, `waitlist_entries.player_id`,
  `game_admins.user_id`/`assigned_by`).

## §A5 — The whole open backlog, ranked, with a disposition for each

| Issue | Ranked | Disposition |
|---|---|---|
| **#164** ADR-0014 conformance (Social Play/Competitions remaining thirds) | **Taken — both remaining contexts, as two tickets** | See §B in full. Not deferred a further sprint: §B1's live-regression finding makes closing both thirds this sprint materially more urgent than backlog hygiene alone would argue |
| **#237** (new, filed this ceremony) T28.1's cross-context regression in `authorizeGameRecording`/`authorizeCompetitionEntryRecording` | **Taken — closed as a side effect of T29.1/T29.2, not by separate code** | §B1–§B3. No Payments-side code change is proposed; the fix is Social Play's/Competitions' own conformance work, already in scope for #164 |
| **#144** `CancelBooking`/`CreateBooking` have no authz | **Escalated** | D1 unanswered; §A7. Seventeenth sprint carrying this |
| **#149** Payments' remaining caller-supplied fact (`booking_host_id`) | **Untouched, correctly** | Blocked on D1 exactly as its own text says. Record-completeness comment posted this ceremony (§A2 item 2); does not narrow it |
| **#145** pre-existing UUID rows vs. `Principal.Subject` | **Re-verified, still genuinely blocked** | Confirmed by re-reading its own text this ceremony; structurally distinct from #164/#237 — needs a real, non-uuid IdP `sub` claim this environment has never produced |
| **#126** real per-Game price field | **Deferred** | Needs Product Owner input; unchanged since T8.10 |
| **#130** refunding a `no_show_fee` Payment | **Deferred** | Carries its own stated open product question; unchanged |
| **#134** WCAG manual screen-reader pass | **Deferred** | Needs real assistive-technology hardware this environment cannot provide; T29 ships no UI change |

**Dependency-completeness check.**

| Arrow | Upstream delivers | Downstream consumes | Complete? |
|---|---|---|---|
| ADR-0017 Decisions 1/3 → T29.1/T29.2 | The "translate, not widen" ruling and the orphan-NULL policy, both already ruled for all three contexts | Each ticket's own migration design | ✅ Complete — re-verified this ceremony (§B4), no re-derivation needed |
| ADR-0017 Decision 2 → T29.1/T29.2 | The target-shape ruling (real `uuid NOT NULL REFERENCES identity_users(id)` once backfilled) | Each ticket's own column list | ⚠️ **Complete only after this plan's own correction** — ADR-0017's own table omits `game_admins`/`competition_admins`; each ticket's instructions below include those two tables explicitly, and #164 carries a comment disclosing the gap (§B7) so a future reader of ADR-0017 alone is not misled |
| Social Play's own `actor()` funnel resolution → `authorizeGameRecording`'s `hostID`/admin-list comparisons | A `User.ID`-shaped `Game.HostID`/`game_admins.user_id` | Payments' existing `port.GameLookup`/`port.GameAdminReader` read paths, unchanged | ✅ Complete once T29.2 ships — no Payments-side code change needed, named explicitly in T29.2's own instructions so the implementer does not go looking for one |
| Competitions' own `actor()` funnel resolution → `authorizeCompetitionEntryRecording`'s `playerID`/admin-list comparisons | A `User.ID`-shaped `CompetitionEntry.PlayerID`/`competition_admins.user_id` | Payments' existing `port.EntryLookup`/`port.CompetitionAdminReader` read paths, unchanged | ✅ Complete once T29.1 ships — same reasoning |

**No same-wave shared-interface collision.** T29.1 (Competitions) and T29.2
(Social Play) touch disjoint bounded contexts — no shared file, no shared
interface either widens or implements. Both may be dispatched in the same
wave.

---

# §B — The live regression, and why it changes this sprint's scope

## §B1 — What was found, and how

Per the task's own instruction to read the merged code where useful, this
ceremony traced PR #235 (T28.1) end to end — every changed file, not just
the ones its own PR body and T28 retro's four mutation-check layers already
covered — rather than treating T28's clean retro as license to skip
re-tracing it. `internal/payments/adapter/grpcapi/handler.go`'s `actor()`
funnel change is not scoped to `ConfirmOnlinePayment` alone: it is the
**shared funnel** for every authenticated Payments RPC, confirmed by
reading `RecordOfflinePayment` (`handler.go:144-159`) and
`CreateOnlinePayment` (`handler.go:182-196`), both of which call `h.actor(ctx)`
and pass the resolved value on as `ActorUserID`.

`internal/payments/app/service.go`'s `authorizeGameRecording`
(`:1039-1067`) and `authorizeCompetitionEntryRecording` (`:1093-1116`) both
compare that now-resolved `ActorUserID` against facts read **out of** Social
Play/Competitions via `port.GameLookup.HostIDForGame`,
`port.GameAdminReader.ListGameAdmins`,
`port.EntryLookup.CompetitionIDAndPlayerIDForEntry`, and
`port.CompetitionAdminReader.ListCompetitionAdmins`. Those four ports are
wired non-nil in `cmd/server/main.go:286-308` against the real Social
Play/Competitions services — not a nil-port fail-closed no-op, a live read.

The values those reads return are Social Play's/Competitions' own stored
`games.host_id`/`registrations.player_id`(transitively)/`game_admins.user_id`
and `competitions.host_id`(transitively via `competition_entries`)/
`competition_admins.user_id` — still plain **subject** text, because neither
context's own `actor()` funnel has been touched
(`internal/socialplay/adapter/grpcapi/handler.go:56-58` and
`internal/competitions/adapter/grpcapi/handler.go:48-50` both still read
`return auth.RequireSubject(ctx)`, unchanged by T28.1).

**Conclusion: since PR #235 merged (`2026-08-16T13:59:00Z`), every genuinely
authorized Game Host, Game Admin, Competition entrant, and Competition Admin
is denied** (`domain.ErrNotPaymentRecorder`) when calling
`RecordOfflinePayment` for a `registration`/`no_show_fee` payable,
`RecordOfflinePayment`/`RefundPayment` for a `competition_entry` payable, or
`CreateOnlinePayment` for a `competition_entry` payable (which reuses
`authorizeCompetitionEntryRecording` per T17.1/#198) — because a resolved
`User.ID` never equals the raw subject these two contexts still store and
return. This fails **closed** (denies a legitimate caller; does not wrongly
permit an illegitimate one), so it is a functional/availability regression,
not a new access hole. It is nonetheless live, on the shared branch, right
now.

**Filed as issue #237** (full text in the issue; not reproduced verbatim
here to avoid the two drifting independently — read the issue for the
complete trace with exact line numbers).

## §B2 — Why T28.1's own review and T28's retro did not catch this

Not a criticism restated for its own sake — worth naming precisely so it
does not recur. T28.1's own ticket text (§B7 instruction 4) and its review
(T28 retro finding 2) both correctly verified `authorizeOnlineConfirmation`
— the **one** comparison inside Payments' own column — with real rigor
(four independent mutation-check layers, per T28 retro finding 3). Neither
the ticket text nor the review traced `authorizeGameRecording`/
`authorizeCompetitionEntryRecording`, because those two functions read facts
**from other contexts**, not from `payments.recorded_by_user_id` — outside
the blast radius T28.1's own instructions named. The dependency-completeness
check T28's own Ceremony 1 ran (`docs/process/t28-sprint-plan.md` §A5) asked
whether T28.1's *inputs* existed (`identityapp.Service.UserBySubject` — yes)
but did not ask the *reverse* question this project's own "Two questions,
not one" clause (`sprint-process.md`, adopted T16) exists for: does
widening what `actor()` returns change the correctness of anything that
reads a **different** context's still-narrow value against it? That
question was never posed, because T28's Ceremony 1 was scoped to #164's
*own* three per-context slices and had no reason to look at
`authorizeGameRecording`/`authorizeCompetitionEntryRecording`'s existing,
unrelated-looking code. This is disclosed rather than left for a future
ceremony to rediscover independently — worth a line in `docs/LESSONS.md`
this same PR (below), since it is exactly the "mistake caught... gets a line
immediately" shape `sprint-process.md`'s execution-loop-mechanics section
describes, even though it was caught at planning time rather than mid-loop.

## §B3 — Why the fix is #164's own remaining scope, not a Payments patch

Considered and rejected: resolving `hostID`/`playerID`/admin-list values
through Payments' own `port.IdentityLookup.UserIDBySubject` at the point of
comparison, as an interim patch that would not require Social Play/
Competitions to migrate anything. Rejected because it would have to be torn
out the moment Social Play/Competitions **do** migrate (§B4's Decision 2) —
at that point `GameLookup.HostIDForGame` would already return a `User.ID`,
and running it through `UserIDBySubject` (which expects a **subject**) a
second time would fail to resolve, reintroducing the identical hazard in
the opposite direction. The complete, permanent fix is the one already
planned: Social Play's and Competitions' own conformance work, which this
plan takes as T29.1/T29.2. **No Payments-side code change is proposed by
this plan** — named explicitly in both tickets' instructions so neither
implementer goes looking for one that does not belong there.

## §B4 — Re-verifying ADR-0017's rulings hold (T28 retro recommendation 4),
not re-deriving them — with one gap found

Per instruction, ADR-0017 read in full this ceremony rather than assumed.

- **Decision 1 (translate, not widen — extends to Social Play/Competitions
  unchanged).** Holds. Nothing in this ceremony's own findings bears on it;
  §B1's regression is a *consequence* of Decision 1 already being correct
  for Payments and not yet applied to the other two, not a challenge to the
  ruling itself.
- **Decision 2 (target shape: real `uuid NOT NULL REFERENCES
  identity_users(id)` once backfilled).** Holds for the five columns its own
  table names (`games.host_id`, `registrations.player_id`,
  `waitlist_entries.player_id`, `competitions.host_id`,
  `competitions.player_id` — the last one is actually
  `competition_entries.player_id`; ADR-0017's table is imprecise about which
  table carries it, corrected here). **Gap found: `game_admins.user_id`/
  `assigned_by` and `competition_admins.user_id`/`assigned_by` are not named
  in Decision 2's table at all**, even though `db/migrations/
  0020_socialplay_game_admins.sql:33-51` and
  `0021_competitions_competition_admins.sql`'s equivalent section both
  explicitly instruct — in their own header comments, written when those
  tables were created — that these columns "MUST BE BACKFILLED IN THE SAME
  PASS" as `games.host_id`/`competitions.host_id`. Decision 2's own
  reasoning (Booking's `recurring_hire_templates` precedent, `0016`'s stated
  anticipation) applies to these two tables identically; the omission looks
  like an enumeration gap in ADR-0017's own table, not a considered
  exclusion — ADR-0017 gives no reasoning anywhere for treating admin
  assignment differently from host/player ownership, and its own
  Consequences section doesn't mention them either. **Not re-litigated as a
  new ruling** — Decision 2's *reasoning* already covers these columns:
  this is a completeness correction to its table, not a disagreement with
  its decision. Disclosed on #164 itself (§B7) per this project's standing
  "post a correction to the issue itself" rule, since #164's own body is
  the other place a future reader would look for the full column list and
  it has the identical gap (inherited from the same oversight).
- **Decision 3 (orphan case — leave `NULL`, and the `NOT NULL`-starting-point
  restatement for Social Play/Competitions).** Holds, and its own text
  already anticipates exactly the two tickets this plan takes — quoted
  directly in each ticket's instructions below rather than re-derived.

## §B5 — Sizing: both contexts, as two tickets, not one combined ticket and
not deferred again

**Why not defer again** (the T28 plan §B9 disagreement, revisited). T28's
own PE-vs-PdE disagreement was about *unproven backfill risk* with no
countervailing urgency — PE's position won because nothing forced a
decision either way. §B1 changes that: leaving Social Play/Competitions
unmigrated for another sprint boundary is no longer neutral backlog
hygiene, it is **leaving a live, disclosed authorization regression
unfixed** for a sprint this project's own capacity has already shown it can
close. Re-deferring on the same "prove it once more before scaling" caution
that applied when the alternative was merely "the columns stay
non-conformant a while longer" does not apply once the alternative is
"legitimate Game Hosts and Competition entrants keep being denied a
functioning payment-recording path."

**Why two tickets, not one combined ticket.** Social Play and Competitions
are independent bounded contexts touching disjoint tables, disjoint
comparison call sites, and disjoint migration files (§A4) — there is no
shared interface either widens, and no ordering dependency between them
(fixing Competitions' comparison does not require Social Play's fix to
exist first, or vice versa). Splitting keeps each PR reviewable at roughly
T28.1's own size and keeps a defect in one from blocking the other's review
— the same reasoning that has governed every other pair of independent,
same-wave tickets on this project.

**Why not risk-neutral simply because the pattern is "proven twice" (T13.2/
T13.3, then T28.1).** Both remaining contexts present the **`NOT NULL`
starting point** ADR-0017 Decision 3's own restatement flags as genuinely
untested — T28.1's column was nullable, so its migration never had to
decide whether the new column could safely gain `NOT NULL` in the same pass
or had to ship nullable first. Each ticket below states, per Decision 3's
own instruction, that its migration adds the new column nullable, backfills
it, and adds `NOT NULL` **only if zero orphans are found** — and reports
which branch it actually took, since ADR-0017 predicts zero orphans are
plausible here (every Social Play/Competitions write already requires a
verified principal as of T12.7/T12.8) but does not know it.

## §B6 — Order: Competitions first, Social Play second

Not load-bearing for correctness (§A5's dependency-completeness table shows
neither ticket needs the other to exist first) but stated for the same
reason T28.1 was chosen as the smallest-first reference case: Competitions'
surface is smaller — 4 columns across 3 tables (`competitions.host_id`,
`competition_entries.player_id`, `competition_admins.user_id`/`assigned_by`)
against ~3 comparison call sites (`domain/competition.go:214,264`,
`domain/competition_admin.go:121`) — than Social Play's 5 columns across 4
tables (`games.host_id`, `registrations.player_id`,
`waitlist_entries.player_id`, `game_admins.user_id`/`assigned_by`) against
~5 comparison call sites (`domain/game.go:156,186`,
`domain/game_admin.go:107`, `domain/registration.go:195`,
`domain/waitlist.go:164`). T29.1 (Competitions) is numbered first for that
reason; both are dispatched in the same wave (§A5), so "first" names review
sequencing, not a gate.

## §B7 — The disclosure this plan posts to #164 and ADR-0017's own gap

A comment is posted on #164 (at merge time, alongside this PR) stating: (a)
T29 takes both remaining thirds, as two tickets, not deferred further; (b)
the reason includes a newly-found live regression (#237), not backlog
hygiene alone; (c) ADR-0017 Decision 2's own column table omits
`game_admins`/`competition_admins`, corrected in this plan's tickets rather
than in a fresh ADR, since Decision 2's *reasoning* already covers them and
nothing about the ruling itself needs to change.

---

# Ceremony 1 — Backlog refinement (remaining sections)

## §A6 — Why two tickets, not zero, not one, not three (the original
Payments+Social Play+Competitions-in-one-sprint shape T28 declined)

Zero would leave #237's live regression unfixed on a backlog this project's
own capacity has already shown can absorb the work (T28.1 shipped cleanly).
One combined ticket would recreate exactly the shared-file/shared-review
risk splitting avoids for two structurally independent contexts (§B5). Three
— the original T28 §B9 all-at-once shape, since Payments is already done —
does not apply: only two contexts remain. Two tickets, one per remaining
context, dispatched in the same wave, is the whole remaining scope of #164
and the whole fix for #237; nothing is left out for scope-discipline's own
sake this time, because leaving either out leaves that context's half of
the live regression standing.

## §A7 — DECISION D1 and DECISION D2 remain unanswered

Re-verified this ceremony (§A3): #144 carries exactly one comment, T14.3's
original escalation, unchanged. ADR-0015's and ADR-0016's `## Status`
headings unchanged.

> **DECISION D1 — seventeenth deferral.** Unchanged: who may cancel a court
> booking made through the public, unauthenticated quote-and-book flow, and
> whether that flow should remain unauthenticated at all. **No T29 ticket
> implements this** — T29.1/T29.2 touch Social Play's/Competitions' actor
> identifier space, not Booking's ownership model. D1 has now carried its
> single T14.3 comment for **sixteen consecutive sprints (T14 through T29)**.

> **DECISION D2 — unchanged.** May a session that reviews and merges a PR
> also author code on it. T29 has two real tickets this time (T29.1,
> T29.2), both real PRs for D2's interim rule to be exercised against again
> — T29's own retro scores which shape each review lands in, per T28 retro
> recommendation 5's instruction not to force a genuinely new instance into
> either of the two existing buckets by default.

**Neither is implemented, decided, or guessed at by this ceremony.**

## §A8 — Shared-file pre-assignment, and same-wave verification

| Artifact | Owner | Notes |
|---|---|---|
| `HANDOFF.md` | this ceremony only | An implementer that finds a stale line flags it for T30's Ceremony 1 |
| `docs/process/sprint-process.md` | not touched this ceremony | No amendment landed |
| `docs/adr/0017-*.md` | not touched by either ticket's own migration work — T29.1/T29.2 *implement* its existing Decisions, they do not amend the ADR file itself | — |
| `db/migrations/0025_*.sql` | T29.1 only | Pre-assigned, §A4 |
| `db/migrations/0026_*.sql` | T29.2 only | Pre-assigned, §A4 |
| `internal/competitions/port/identity_lookup.go`, `internal/competitions/adapter/identity/` | T29.1 only | New files, single owner |
| `internal/socialplay/port/identity_lookup.go`, `internal/socialplay/adapter/identity/` | T29.2 only | New files, single owner |
| `internal/payments/**` | **neither ticket** | §B3 — no Payments-side code change; stated here so neither implementer edits it by reflex |

**Same-wave shared-interface verification rule: does not apply.** T29.1 and
T29.2 touch disjoint bounded contexts — no shared file, no shared interface
either widens or a second implements (§A5's last row).

---

# Ceremony 2 — Sprint planning

## Sprint goal

> Close both remaining thirds of #164 — Social Play and Competitions — as
> two independent, same-wave tickets, following the reference pattern T28.1
> proved on Payments and re-verifying rather than re-deriving ADR-0017's
> Decisions 1–3 (finding and correcting one real gap in Decision 2's own
> column table along the way: `game_admins`/`competition_admins` belong in
> it and were never listed). Do this not only to finish backlog-hygiene
> conformance but because independently tracing T28.1's merged diff this
> ceremony found a live regression it introduced — `authorizeGameRecording`/
> `authorizeCompetitionEntryRecording` now compare Payments' correctly-resolved
> actor against Social Play's/Competitions' still-subject-shaped reads, denying
> every genuinely authorized Game Host, Game Admin, Competition entrant, and
> Competition Admin from recording or confirming a payment since PR #235
> merged (filed as **#237**). T29.1/T29.2 fix this as a side effect, with no
> Payments-side code change, by finishing the identifier-space migration
> those reads depend on. Re-verify the other 6 issues' blockers live and find
> them unchanged.

**What this sprint does not claim:**

- **This is not a claim that #237 is a security vulnerability.** It fails
  closed (denies a legitimate caller), not open. It is a functional/
  availability regression, disclosed and tracked (#237), not an access-control
  hole.
- **This does not touch D1 or D2.** Both remain exactly as escalated as
  T28 left them (§A7).
- **This is not a retroactive indictment of T28.1's implementer or
  reviewer.** §B2 states precisely why the gap fell outside what either
  party's own instructions or checks covered — a real, narrow, first
  instance of "widening what a shared funnel returns can silently break an
  unrelated comparison that reads a *different* context's still-narrow
  value," not a diligence failure repeating a known pattern.
- **This does not mean ADR-0017 needs to be reopened or re-voted.**
  Decision 2's *reasoning* already covers `game_admins`/`competition_admins`;
  only its enumerated table needed correcting, done in this plan's ticket
  text and disclosed on #164 (§B7) rather than via a fresh ADR revision.

## Tickets — 2 items, 34 points

### T29.1 — Competitions' Identity port, funnel wiring, and backfill (host_id, player_id, admin assignments)

**Story:** As a platform maintainer, I want `competitions.host_id`,
`competition_entries.player_id`, and `competition_admins.user_id`/
`assigned_by` to hold the same identifier space ADR-0014/ADR-0017 already
establish for Booking, Facilities, and Payments, so that every
authorization comparison inside Competitions — and every comparison
Payments makes against a value read out of Competitions — compares a
`User.ID` against a `User.ID` by design, closing Competitions' third of
#164 and Competitions' half of #237.

**Description:** Closes the Competitions third of **#164** (a partial fix —
Social Play remains, T29.2). Applies ADR-0017's already-ruled Decisions 1–3
to Competitions' four columns (including `competition_admins`, per this
plan's §B4/§B7 correction to ADR-0017's own table). Fixes
`authorizeCompetitionEntryRecording`'s half of **#237** as a side effect —
no code change to `internal/payments/**` is required or in scope for this
ticket (§B3).

**Instructions:**

1. **Do not re-litigate ADR-0017.** Read it in full first
   (`docs/adr/0017-*.md`), confirm Decisions 1–3 as this plan's §B4
   restates them, and proceed — this ticket implements the ruling, it does
   not re-derive it.
2. **Build `internal/competitions/port.IdentityLookup`**, mirroring
   `internal/payments/port/identity_lookup.go` exactly (T28.1, read in full
   this ticket) — one method, `UserIDBySubject(ctx, subject string)
   (string, error)`, same doc-comment discipline naming which identifier
   space its parameter takes. Implement it once in
   `internal/competitions/adapter/identity` against the real
   `identityapp.Service`, following `internal/payments/adapter/identity`'s
   shape (T28.1) — that is now the third instance of this exact pattern
   (T13.2 Booking, T13.3 Facilities, T28.1 Payments), so read all three
   before writing a fourth, not just the most recent.
3. **Wire the funnel.** `internal/competitions/adapter/grpcapi/handler.go`'s
   `actor(ctx)` (line 48, currently `return auth.RequireSubject(ctx)`)
   becomes a `*Handler` method resolving through the new port, exactly as
   T28.1's `internal/payments/adapter/grpcapi/handler.go` change does.
4. **Fix every comparison that reads `HostID`/`PlayerID`/admin `UserID` to
   compare the right space on both sides**, once the funnel resolves.
   Verified this ceremony — the full call-site list, so none is missed:
   - `internal/competitions/domain/competition.go:214` (`EnsureHost`),
     `:264` (host-or-admin check)
   - `internal/competitions/domain/competition_admin.go:121`
   - `internal/competitions/app/service.go:573` (`ListEntriesForCompetition`'s
     `EnsureHostOrCompetitionAdmin` call), `:660` (`CancelCompetition`'s
     `EnsureHost` call), plus `AssignCompetitionAdmin`/`RevokeCompetitionAdmin`
   None of these functions' **logic** needs to change (mirrors T28.1's own
   finding that `authorizeOnlineConfirmation`'s comparison itself was
   already correct) — only what feeds both sides of each comparison, via
   steps 2–3 and the migration in step 5.
5. **Migration `0025_competitions_identity_conformance.sql`** (pre-assigned,
   §A4). Converts, in one file:
   - `competitions.host_id` → `uuid NOT NULL REFERENCES identity_users(id)`
   - `competition_entries.player_id` → `uuid NOT NULL REFERENCES
     identity_users(id)`
   - `competition_admins.user_id` → `uuid NOT NULL REFERENCES
     identity_users(id)`, `competition_admins.assigned_by` → `uuid NOT NULL
     REFERENCES identity_users(id)`
   Per ADR-0017 Decision 3's own `NOT NULL`-starting-point restatement: add
   each new column **nullable**, backfill via the same `UPDATE … FROM
   identity_users` join keyed on `subject` T28.1's migration used, and only
   then evaluate whether `NOT NULL` can be added in this same migration —
   **only if the backfill leaves zero `NULL`s**. If any row is a genuine
   orphan (a subject with no matching `identity_users.subject`), the column
   ships nullable and a follow-up tightening migration is out of this
   ticket's scope (state which branch was actually taken in the PR body,
   per Decision 3's own instruction to T29). `competition_admins`'
   composite `PRIMARY KEY (competition_id, user_id)` and its
   `competition_admins_user_idx` index must be dropped and recreated
   against the new column, not left referencing a renamed-away `text`
   column.
6. **Update every read/write site** touching these four columns in
   `internal/competitions/adapter/postgres/repository.go` and
   `internal/competitions/adapter/postgres/competition_admin_repository.go`
   to the new `uuid`/`pgtype.UUID` shape, mirroring T28.1's
   `mustUUID`/`uuidString`-conversion pattern
   (`internal/payments/adapter/postgres/repository.go`). Domain-layer
   `HostID`/`PlayerID`/`UserID` fields stay Go `string` (a uuid string, not
   a Go UUID type) — this migration is a storage/identifier-space change,
   not a Go representation change.
7. **Payments-side verification, not modification (§B3).** Add or update a
   test in `internal/payments/app` — `TestRecordOfflinePayment_*`/
   `TestRefundPayment_*`/`TestCreateOnlinePayment_*` covering the
   `competition_entry` payable path — using **non-matching-by-default**
   fixture values for `ActorUserID` versus the fake `EntryLookup`/
   `CompetitionAdminReader`'s returned ids (e.g. distinct literal uuids,
   not the same string reused on both sides), so the test would fail if a
   future change reintroduced an identifier-space mismatch the way #237
   did. This is a **test-only** change inside `internal/payments/app`; no
   non-test file under `internal/payments/**` is touched by this ticket
   (§A8/§B3).
8. **Mutation-check the backfill** (CLAUDE.md rule 10): seed a
   `competitions`/`competition_entries`/`competition_admins` row set with
   one resolvable subject and one orphaned subject per table, confirm the
   migration resolves the first to the correct `identity_users.id` and
   leaves the second `NULL` (or reports why zero orphans were found and
   `NOT NULL` was added in the same pass, per step 5).
9. **State the constructor/dependency outcome explicitly in the PR body** —
   whether `competitionsapp.NewService`'s `ServiceOptions` gains a new
   field for the identity port.
10. **Standing rule:** a partial fix does not close #164. This PR's title
    is **"partial fix for #164"** (Social Play remains, T29.2), and its
    body states it closes Competitions' half of **#237** — post the same as
    a comment on #164 and on #237 at merge time, per this project's
    "closes #N"/"partial fix for #N" review-honesty convention generalized
    to two issues at once.
11. Non-functional: `make test-domain` stays green; domain stays pure
    (CLAUDE.md rule 2); land the funnel change and the backfill/column-type
    migration together, in one PR, with no window where any of step 4's
    comparisons could compare across identifier spaces (CLAUDE.md rule 9,
    and the same specific hazard T28.1's own migration header names).

**Cross-context dependency check:** calls `internal/identity` via the new
`port.IdentityLookup`; `identityapp.Service.UserBySubject` already exists
(same primitive T13.2/T13.3/T28.1 already consume) — re-verified present
this ceremony. Consumed by: `internal/payments`' existing `EntryLookup`/
`CompetitionAdminReader` read paths (§A5's dependency-completeness table) —
no change needed on that side, verified this ceremony that both ports'
existing signatures already return exactly what they need to once this
ticket's migration lands.

**Story points:** 13. **Role:** principal-engineer. **Labels:**
`role:principal-engineer`, `type:chore`, `points:13`.

### T29.2 — Social Play's Identity port, funnel wiring, and backfill (host_id, player_id ×2, admin assignments)

**Story:** As a platform maintainer, I want `games.host_id`,
`registrations.player_id`, `waitlist_entries.player_id`, and
`game_admins.user_id`/`assigned_by` to hold the same identifier space
ADR-0014/ADR-0017 already establish for Booking, Facilities, and Payments,
so that every authorization comparison inside Social Play — and every
comparison Payments makes against a value read out of Social Play —
compares a `User.ID` against a `User.ID` by design, closing Social Play's
third of #164 (the last of the three) and Social Play's half of #237.

**Description:** Closes the Social Play third of **#164** — the last
remaining context, following T29.1 (Competitions) and T28.1 (Payments).
Applies ADR-0017's already-ruled Decisions 1–3 to Social Play's five columns
across four tables (the largest of the three contexts' surfaces, per this
plan's §B6 sizing). Fixes `authorizeGameRecording`'s half of **#237** as a
side effect — no code change to `internal/payments/**` is required or in
scope for this ticket (§B3).

**Instructions:**

1. **Do not re-litigate ADR-0017 or T29.1's own PR** — read both first
   (`docs/adr/0017-*.md`; T29.1's merged diff if it has landed by the time
   this ticket starts, since both are same-wave but T29.1 is expected to
   review first per §B6's ordering) and proceed as an application of an
   already-proven pattern, now for its fourth instance
   (T13.2/T13.3/T28.1/T29.1).
2. **Build `internal/socialplay/port.IdentityLookup`**, mirroring
   `internal/payments/port/identity_lookup.go` (T28.1) and
   `internal/competitions/port/identity_lookup.go` (T29.1, if merged first)
   exactly. Implement it once in `internal/socialplay/adapter/identity`
   against the real `identityapp.Service`.
3. **Wire the funnel.** `internal/socialplay/adapter/grpcapi/handler.go`'s
   `actor(ctx)` (line 56, currently `return auth.RequireSubject(ctx)`)
   becomes a `*Handler` method resolving through the new port.
4. **Fix every comparison that reads `HostID`/`PlayerID`/admin `UserID` to
   compare the right space on both sides**, once the funnel resolves.
   Verified this ceremony — the full call-site list:
   - `internal/socialplay/domain/game.go:156` (host-or-admin check), `:186`
     (`EnsureHost`)
   - `internal/socialplay/domain/game_admin.go:107`
   - `internal/socialplay/domain/registration.go:195` (`Registration.Cancel`)
   - `internal/socialplay/domain/waitlist.go:164` (`WaitlistEntry.Cancel`)
   - `internal/socialplay/app/service.go:506` (`CancelRegistration`), `:694`
     (`ListRegistrationsForGame`'s host-or-admin check), `:978`
     (`CancelGame`'s `EnsureHost` call), plus `AssignGameAdmin`/
     `RevokeGameAdmin`
   As with T29.1, none of these functions' comparison **logic** changes —
   only what feeds both sides.
5. **Migration `0026_socialplay_identity_conformance.sql`** (pre-assigned,
   §A4). Converts, in one file:
   - `games.host_id` → `uuid NOT NULL REFERENCES identity_users(id)`
   - `registrations.player_id` → `uuid NOT NULL REFERENCES
     identity_users(id)`
   - `waitlist_entries.player_id` → `uuid NOT NULL REFERENCES
     identity_users(id)`
   - `game_admins.user_id` → `uuid NOT NULL REFERENCES identity_users(id)`,
     `game_admins.assigned_by` → `uuid NOT NULL REFERENCES
     identity_users(id)`
   Same nullable-first, `NOT NULL`-only-if-zero-orphans discipline as
   T29.1 step 5, applied independently per table (one table having zero
   orphans does not imply another does). `game_admins`' composite
   `PRIMARY KEY (game_id, user_id)` and `game_admins_user_idx` index must be
   dropped and recreated against the new column.
6. **Update every read/write site** touching these five columns in
   `internal/socialplay/adapter/postgres/repository.go` and
   `internal/socialplay/adapter/postgres/game_admin_repository.go` to the
   new `uuid`/`pgtype.UUID` shape, mirroring T28.1's/T29.1's conversion
   pattern. Domain-layer fields stay Go `string`.
7. **Payments-side verification, not modification (§B3).** Add or update a
   test in `internal/payments/app` covering the `registration`/
   `no_show_fee` payable paths (`authorizeGameRecording`), using
   non-matching-by-default fixture values for `ActorUserID` versus the fake
   `GameLookup`/`GameAdminReader`'s returned ids, mirroring T29.1
   instruction 7's reasoning exactly. Test-only change inside
   `internal/payments/app`; no non-test file under `internal/payments/**`
   touched.
8. **Mutation-check the backfill** (CLAUDE.md rule 10): seed
   `games`/`registrations`/`waitlist_entries`/`game_admins` rows with one
   resolvable and one orphaned subject per table, confirm correct
   resolution/`NULL` behavior (or report the zero-orphan/`NOT NULL` branch
   taken, per step 5).
9. **State the constructor/dependency outcome explicitly in the PR body** —
   whether `socialplayapp.NewService`'s `ServiceOptions` gains a new field
   for the identity port.
10. **Standing rule:** a partial fix does not close #164 unless T29.1 has
    already merged, in which case this PR's own merge is the one that
    **does** close #164 in full — state explicitly, at merge time, which
    of the two this PR is (check #164's live state before writing the
    closing comment, don't assume T29.1 already landed). Also closes
    Social Play's half of **#237**; if T29.1 has already merged, this PR's
    merge closes #237 in full — same live-state check before writing that
    comment too.
11. Non-functional: `make test-domain` stays green; domain stays pure
    (CLAUDE.md rule 2); land the funnel change and the backfill/column-type
    migration together, in one PR, with no window where any of step 4's
    comparisons could compare across identifier spaces.

**Cross-context dependency check:** calls `internal/identity` via the new
`port.IdentityLookup`; `identityapp.Service.UserBySubject` already exists —
re-verified present this ceremony. Consumed by: `internal/payments`'
existing `GameLookup`/`GameAdminReader` read paths — no change needed on
that side, re-verified this ceremony.

**Story points:** 21. **Role:** principal-engineer. **Labels:**
`role:principal-engineer`, `type:chore`, `points:21`.

*(Sized above T29.1's 13 rather than matching it: one more table
(`waitlist_entries`) and one more comparison call site than Competitions'
surface, per §B6's column/call-site count — not a difference in kind, a
difference in surface area.)*

## Waves

**Wave 1 (only wave): T29.1 and T29.2, dispatched together.** No same-wave
collision (§A8) — disjoint bounded contexts, disjoint files, no shared
interface. T29.1 is expected to reach review first per §B6's smaller-surface
ordering, but neither ticket's own correctness depends on the other's
merge order — whichever reviews second checks #164's and #237's live state
before writing its own closing comment (both tickets' instruction 10),
exactly the discipline `sprint-process.md`'s "closes #N" review-honesty
convention already requires, generalized here to a case where either PR
could plausibly be the one that actually closes each issue.

## Recorded disagreements (Ceremony 2 rule 3 — not smoothed over)

**PE's residual caution, stated even though the ceremony resolves in favor
of taking both.** The `NOT NULL`-starting-point risk §B5 names (proven zero
times before this sprint, unlike the nullable case T28.1 already proved) is
real and unretired by §B1's urgency argument — urgency changes *whether* to
take the work this sprint, not whether the specific technical risk exists.
PE's position, recorded rather than dropped: if either ticket's backfill
finds a non-trivial orphan population during implementation (contrary to
ADR-0017 Decision 3's own "plausible... but does not know it" hedge), the
right response is to ship that ticket nullable and open a follow-up
`NOT NULL`-tightening ticket rather than stretch the loop-cap (5 loops,
`sprint-process.md`) trying to force a same-migration `NOT NULL` that the
data does not actually support. This is not a prediction of failure; it is
naming, in advance, what "found a real problem" should lead to rather than
leaving that call to be improvised mid-loop.

## Sprint-level Definition of Done

1. **T29.1 and T29.2 merged per the per-ticket DoD**
   (`sprint-process.md`'s Execution section): acceptance criteria met, PR
   reviewed (findings addressed or explicitly deferred), `make test-domain`
   green at minimum, approved and merged by the user or an explicitly
   delegated gate — never self-merged.
2. **The merged-fix issue sweep run and reported** — by the retro
   (reporting) and again by T30's Ceremony 1 (authoritative).
3. **#164 closed in full** by whichever of T29.1/T29.2 merges second, with
   an explicit comment naming both merged PRs — scored at the retro: did
   the closing PR correctly check #164's live state before writing its
   comment (both tickets' instruction 10), and did the close actually use
   `state_reason: completed` per DoD step 5's mechanics.
4. **#237 closed in full** by the same PR, with the same live-state-check
   discipline — or, if only one of the two lands this sprint, left open
   with a written sentence naming which half remains, per the standing
   partial-fix convention.
5. **Scoring owed at the retro:**
   - **(a)** Did each ticket ship its funnel change and its
     backfill/column-type migration together, as one reviewed unit, with no
     window any of its named comparisons could break in?
   - **(b)** Was each backfill mutation-checked per CLAUDE.md rule 10 (an
     orphaned subject and a resolvable one both exercised per table, not
     merely asserted)?
   - **(c)** Did each ticket's migration ship `NOT NULL` or nullable, and
     was that the correct branch per Decision 3 (i.e., was it actually
     zero orphans if `NOT NULL` was chosen)?
   - **(d)** Do the Payments-side regression tests (both tickets'
     instruction 7) actually use non-matching-by-default fixture values,
     and would they have caught #237 had it still been present?
   - **(e)** Do the other 6 issues' blockers still hold, re-verified live
     at retro time?
   - **(f)** Did D1 or D2 get answered mid-sprint, as a formal ADR
     decision?
   - **(g)** Which of D2's two named shapes (or the genuinely-new third
     shape T28 retro recommendation 5 names) does each ticket's real review
     actually land in — scored per ticket, not pooled into one answer for
     both.
6. **Not scoreable by T29 and deliberately not pre-empted:** D1 and D2
   remain the user's own decisions.
7. Retro in `docs/process/t29-retro.md`, indexed by a `## T29 sprint retro`
   stub in `docs/LESSONS.md`. `HANDOFF.md`/`CLAUDE.md` state updated —
   noting that T30's Ceremony 1 corrects T29's Docs-index row.
