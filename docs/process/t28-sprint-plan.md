# T28 Sprint Plan — Ceremonies 1 + 2

Backlog refinement (Ceremony 1) and sprint planning (Ceremony 2) per
`docs/process/sprint-process.md`, six-role team (briefs:
`docs/agent-operating-handbook.md` Part B). Held against
`docs/process/t27-retro.md` (PR #232, merged `2026-08-16T11:25:54Z`),
`HANDOFF.md` including its Docs index, `CLAUDE.md`, and the live PR/issue
state of `nhuthuynh/white-label` (GitHub-side name `pickleball-platform`), at
branch tip `c245a98` (T27 retro's own merge), re-fetched and confirmed
matching `origin/claude/go-backend-pickleball-7up34j` before this ceremony's
branch was cut.

**This ceremony departs from T20–T27's established 0-ticket shape.** Those
eight ceremonies each independently re-verified that all 8 open issues were
genuinely blocked and correctly took no tickets. This ceremony re-examines
one of those eight — **#164** — because its long-standing classification
("blocked on a real IdP tenant this environment cannot provision") was
flagged by the user as likely wrong, and an independent re-verification
performed in this ceremony, from primary sources rather than from that
classification's own repeated restatement, **confirms the flag is correct**.
§B below is the substantial new content this produces; the routine sections
(§A0–§A3, D1/D2, the sweep) are kept at the length T23–T27's retros
established, since nothing about them changed.

## §A0 — Correcting T27's Docs-index row and Task-backlog narrative

Per `sprint-process.md`'s standing convention (a retro PR never updates the
Docs-index row that points at it) and per `docs/process/t27-retro.md`'s own
closing line, **performed directly in `HANDOFF.md` by this same PR**:

| Field | What the row now says | Independently re-verified this ceremony |
|---|---|---|
| Retro link | `docs/process/t27-retro.md` (no incident-grade finding; independently re-verified live, issue by issue, that all 8 open issues' blockers held for the whole sprint; migration-tooling classification unchanged; D1/D2 unanswered; neither T21 retro reopening condition fired; two running counters carried forward — backlog-static count to 14, D1 silence held at 14) | File exists, read in full this ceremony; its own characterization matches its content |
| Reviews cell | PR #232 (Ceremony 1/2 doc) → PR (retro doc, filed as part of the same T27 cycle) | Re-fetched via `list_pull_requests`; `merged_at` for #232 is `2026-08-16T11:25:54Z`; no PR merged after it before this ceremony's branch cut |
| Task-backlog narrative | The retro's own agreed sentence, quoted verbatim from `docs/process/t27-retro.md`'s "The sprint goal, scored" section | Copied word-for-word, not paraphrased (see block below) |

> T27 shipped zero tickets, the eighth 0-ticket sprint in this project's
> history, and this retro independently re-verified rather than trusted
> that the reason was real: all 8 tracked issues' blockers were re-checked
> live, issue by issue, and every field matches T27's plan's own
> live-fetched table exactly — none moved. The `golang-migrate`/`goose`
> migration-tooling classification is unchanged, re-checked against a fresh
> grep and the ADR/migration directory listings (still ending at `0016`
> and `0023` respectively). Neither D1 nor D2 was answered as a formal ADR
> decision this sprint — both ADRs' `## Status` sections and #144's comment
> body were read directly and are unchanged. Neither of T21 retro's two
> named reopening conditions fired, checked live for the eighth time by
> eight different ceremonies with an identical result each time. Two
> running counts were carried forward: the backlog's consecutive-static-check
> count increments to fourteen; D1's consecutive-sprint-silence count holds
> at fourteen (T14 through T27, and will only become fifteen if T28 opens
> with #144 still uncommented). Per the task's own instruction and T23
> retro's finding 7, the "is this healthy" question is not re-derived here —
> nothing fired, nothing changed, and this retro states that in one sentence
> rather than manufacturing a fresh analysis.

**A new T28 row is added** in the same before-the-sprint form, Retro and
Reviews honestly marked "not yet written" / "not yet opened".

**ADR-status citation form** (T24 retro's convention, still in force):

- ADR-0015 `## Status`: **"Escalated — awaiting product decision. This ADR
  decides nothing."**
- ADR-0016 `## Status`: **"Escalated — awaiting the user's decision. This
  ADR decides nothing."**

## §A1 — The merged-fix issue sweep

**Step 1.** `list_issues(state: OPEN)` at ceremony start → **`totalCount: 8`**:
#124, #126, #130, #134, #144, #145, #149, #164. Identical set to T27's own
sweep and T27's retro sweep before it.

**Step 2.** `open_at_end_of_T27's_retro (8) − closed_since (0) +
opened_since (0) = 8`. Matches the live `totalCount: 8` exactly.

**Step 3.** `list_pull_requests(state: closed, base:
claude/go-backend-pickleball-7up34j, sort: updated, direction: desc)` — most
recent entry is T27's own retro PR; nothing merged since. `git log
--oneline -5` shows `c245a98` as the tip with no descendants, matching
`origin` exactly. **Zero PRs to cross-reference.**

**Sweep result: clean, continuing the unbroken run since T15.** T29's
Ceremony 1 re-runs this sweep in full regardless.

## §A2 — T27 retro's four recommendations, dispositioned

| # | Recommendation | Disposition |
|---|---|---|
| 1 | Continue treating the merged-fix sweep as authoritative regardless of the prior retro's clean result | **Executed** (§A1) — re-verified from the live API, not trusted from the retro's table |
| 2 | Keep DoD (d) at the abbreviated length | **Superseded by this sprint's shape** — see §B; DoD (d) is still checked (§A3) but this ceremony is not a routine confirm-and-report sprint, so the abbreviated form governs only the routine 7-issue re-check, not #164's re-examination |
| 3 | Carry forward the two running counts | **Executed, with one count broken on purpose** — see §A3.1: the backlog's consecutive-static-check count does **not** extend to 15, because the backlog is no longer static this ceremony (#164 moves). D1's consecutive-sprint-silence count still increments normally (§A7) since D1 itself is untouched |
| 4 | If a ninth consecutive 0-ticket sprint arrives, do not open a new "is this healthy" engagement question by default | **Moot** — T28 is not a 0-ticket sprint (§B), so the condition this recommendation gates does not arise |

## §A3 — Routine re-verification (the 7 issues whose status is unchanged)

| Issue | `updated_at` at T27 retro | `updated_at` now (live) | Comments | Changed? |
|---|---|---|---|---|
| #144 (D1) | `2026-08-15T07:01:03Z` | `2026-08-15T07:01:03Z` | 1 | No |
| #149 | `2026-08-15T16:56:58Z` | `2026-08-15T16:56:58Z` | 3 | No |
| #124 | `2026-08-15T16:25:34Z` | `2026-08-15T16:25:34Z` | 1 | No |
| #145 | `2026-08-15T05:01:29Z` | `2026-08-15T05:01:29Z` | 1 | No |
| #126 | `2026-08-14T16:12:26Z` | `2026-08-14T16:12:26Z` | 0 | No |
| #130 | `2026-08-14T16:30:25Z` | `2026-08-14T16:30:25Z` | 0 | No |
| #134 | `2026-08-14T16:37:49Z` | `2026-08-14T16:37:49Z` | 0 | No |

**#164 is the eighth row, and it is the one this ceremony does not carry
forward unchanged — see §B.** Its own `updated_at`/comment count above
GitHub's API (`2026-08-15T14:16:28Z`, 1 comment) is unchanged as of ceremony
start; what changes is this ceremony's own classification of it, on
independently re-derived evidence, not anything that happened to the issue
between sprints.

**Migration-tooling classification (`golang-migrate`/`goose`):** re-checked,
`HANDOFF.md`'s Cross-cutting section (line 1373) still carries exactly the
one line every prior ceremony has quoted, no growth. `ls docs/adr/` ends at
`0016`; `ls db/migrations/` ends at `0023`. **Still correctly unticketed,
unchanged.**

**Cross-cutting section re-scan (the eleventh sprint running this exact
check, T18–T27, now T28):** read in full (lines 1344–2008). No new
#212/#213-shaped candidate found beyond what T18's through T27's ceremonies
already catalogued and either closed or tracked. (§B's re-examination of
#164 is not a Cross-cutting-section find — #164 was already a tracked
issue; this ceremony re-derives its blocker status from the issue's own
body and the codebase, not from a new sentence discovered in `HANDOFF.md`.)

**D1/D2 re-confirmed unchanged:** ADR-0015 and ADR-0016's `## Status`
sections read in full this ceremony, byte-for-byte identical to T27's
quotation. #144's single comment re-fetched via `issue_read(get_comments)`
— T14.3's original escalation, unchanged.

**Toolchain:** `make test-domain` — 12/12 packages green. `make
gate-coverage` — `OK — all 42 package(s) with runnable tests are executed by
"ci-checks"`. Both run directly against the unmodified tree.

### §A3.1 — The backlog-static-check counter breaks on purpose

T27's count stood at fourteen ceremonies finding the 8-issue set's blockers
unchanged. **This ceremony does not extend that count to fifteen**, because
extending it would misstate what happened: seven of the eight issues are
unchanged (table above), but the eighth (#164) is not being carried forward
with the same blocker classification it has held since roughly T14 — see
§B. A counter whose entire meaning is "found the set unchanged" must stop
incrementing the moment the set is found changed, on pain of becoming
exactly the kind of number nobody re-derives, which is the failure mode
this whole ceremony exists to catch elsewhere. **The counter is retired at
fourteen** (T21 Ceremony 1 through T27 retro) rather than carried forward
incorrectly; if a future sprint wants an analogous counter for "consecutive
checks finding the *current* backlog composition unchanged," it starts a
new count from this ceremony, not from fourteen.

---

# §B — Re-examining #164: is it genuinely blocked on a real IdP tenant?

## §B1 — What every ceremony since roughly T14 has said, and what the user
flagged

`docs/process/t14-sprint-plan.md` through `docs/process/t27-sprint-plan.md`
(fourteen sprints, T14–T27, ranks #164 alongside #145 identically) states
the blocker as: *"blocked on a real IdP tenant this environment cannot
provision."* No ceremony in that run re-read #164's own issue body against
current code before restating that classification — each cited the
classification's own prior restatement, which is the same "trusting a
prior ceremony's diligence rather than re-deriving it" failure mode this
project's own retros have repeatedly found and corrected elsewhere (T14
finding 5 / #97, T15 finding 5 / #149). The user flagged this directly, and
this ceremony re-derives the answer from primary sources rather than
accepting either the user's flag or the fourteen-sprint precedent on faith.

## §B2 — #164's own text, read in full this ceremony

`issue_read(get, #164)`, title: *"Bring Social Play, Competitions and
Payments actor columns into ADR-0014 conformance (needs a backfill, not just
code)."* It requires, per its own numbered list:

1. A new `port.IdentityLookup` + `adapter/identity` per context (Social
   Play, Competitions, Payments) — **none of the three has one today**,
   verified directly this ceremony (§B4).
2. Resolution wired into each context's `actor()` funnel.
3. **A backfill of every existing row** mapping the stored subject string to
   the corresponding `identity_users.id`.
4. A decision on whether those columns become real `uuid` FKs to
   `identity_users(id)` — which `0016_identity.sql`'s own header comment
   already anticipates.

**Nothing in this list names an external identity provider.** The issue's
own "Why this is not a quick fix" section is explicit about what actually
blocks conformance: doing the three contexts *piecemeal* (one at a time,
each independently) is unsafe, because a resolution step added to only one
side of an `actor(ctx)`-vs-stored-value comparison silently breaks
authorization in either direction. That is a **sequencing** hazard inside
this codebase, not an external dependency.

## §B3 — ADR-0014 §5/§5a, read in full this ceremony

`docs/adr/0014-actor-identifier-space-and-the-subject-resolution-seam.md`
§5's table rules Social Play, Competitions and Payments **"⚠️ non-conformant
but self-consistent"** — both sides of every comparison in these three
contexts read from `actor(ctx)` today, so they agree with each other by
coincidence, not because either side is the "right" identifier space. §5a
states the target state explicitly: *"Conformance for these three is
deliberately deferred, because it is not a code change: it needs a backfill
of existing rows plus a port and adapter per context. Tracked as a follow-up
issue rather than a paragraph here."* — that follow-up issue is #164. The
"What this ADR does not decide" section names the same two open items #164's
body names: how the three contexts get to the target state, and whether the
columns become real FKs. **Nowhere in ADR-0014 is a real external IdP named
as a precondition for this specific follow-up** — ADR-0014's *own* subject
translation ruling was proven inside this environment, against this
environment's dev-auth fixture, by T13.2 and T13.3.

## §B4 — Independent verification of #164's own citations (not trusted from
its prose)

| #164's claim | Verified this ceremony | Result |
|---|---|---|
| None of Social Play/Competitions/Payments has a port into Identity | `ls internal/{socialplay,competitions,payments}/port/` | **Confirmed.** Social Play: `court_reservation.go`, `facility_lookup.go`, `game_admin_repository.go`, `idgenerator.go`, `registration_payment_updater.go`, `repository.go`, `waitlist_repository.go` — no `identity_lookup.go`. Competitions: `competition_admin_repository.go`, `competition_entry_payment_updater.go`, `court_reservation.go`, `facility_lookup.go`, `idgenerator.go`, `repository.go`, `share_token_generator.go` — no `identity_lookup.go`. Payments: `competition_admin_reader.go`, `entry_lookup.go`, `game_admin_reader.go`, `game_lookup.go`, `idgenerator.go`, `payment_processor.go`, `registration_lookup.go`, `repository.go`, `webhook_event_store.go`, `webhook_verifier.go` — no `identity_lookup.go`. (`game_admin_repository.go`/`competition_admin_repository.go`/`*_admin_reader.go` are the T14.4/T15.3/T15.5 admin-assignment stores — a different capability, not an Identity port.) |
| `games.host_id`, `registrations.player_id`, `waitlist_entries.player_id` are `text NOT NULL` | Read `db/migrations/0005_socialplay.sql` and `0007_socialplay_waitlist.sql` directly | **Confirmed.** `host_id text NOT NULL` (`0005:22`), `player_id text NOT NULL` (`0005:36`), `player_id text NOT NULL` (`0007:19`) |
| `competitions.host_id`, `player_id` are `text NOT NULL` | Read `db/migrations/0014_competitions.sql` directly | **Confirmed.** `host_id text NOT NULL` (`:25`), `player_id text NOT NULL` (`:151`) |
| `payments.recorded_by_user_id` is `text` (nullable) | Read `db/migrations/0005_payments.sql` directly | **Confirmed.** `recorded_by_user_id text` (`:32`), no `NOT NULL` |
| `0016_identity.sql`'s header anticipates these becoming real FK targets | Read `db/migrations/0016_identity.sql` header in full | **Confirmed, quoted exactly.** *"A User is the aggregate other contexts' currently-opaque actor_user_id/host_id/player_id strings will eventually reference"* |
| The `identity_users.id`/`identity_users.subject` backfill targets already exist | Read `db/migrations/0016_identity.sql` and `0019_identity_subject.sql` | **Confirmed.** `identity_users.id uuid` (server-minted, `0016`) and `identity_users.subject text NOT NULL UNIQUE` (`0019`, T12.9) both already exist and are populated by every `CreateUser` call this environment can make — including through `make dev-token` + the committed `dev/auth/` fixture (CLAUDE.md's own documented dev flow) |
| T13.2/T13.3 already built this exact pattern, twice, with no real IdP | Read `internal/booking/port/identity_lookup.go`, `internal/facilities/port/identity_lookup.go`, `internal/booking/adapter/identity/`, `internal/facilities/adapter/identity/` in full | **Confirmed.** Both ports expose `UserIDBySubject(ctx, subject string) (string, error)`, both are implemented once by an `adapter/identity` package against the real `identityapp.Service`, both were reviewed and merged (`HANDOFF.md`'s T13 row: PR #166 for T13.2, PR #170 for T13.3) without any external identity provider — T13.2's own text is explicit that the translation is proven against `identity_users.subject`, a column this backend owns |

**Every citation in #164's body checks out exactly as written.** The issue
is accurate, current, and does not itself claim an external-IdP blocker
anywhere in its text — that classification was added by ceremony
restatement starting around T14, not by anything in the issue.

## §B5 — One genuinely new wrinkle #164 flags that T13.2/T13.3 did not have
to solve: live backfill data

T13.2 and T13.3 both fixed **previously-broken write paths** — before their
fix, `RequestRecurringHire` returned an error to every caller and
`CreateFacility` panicked, so neither had accumulated any rows in the wrong
identifier space to migrate. Social Play, Competitions and Payments are
different: since T12.8, their `actor(ctx)`-sourced writes have been
**succeeding** (both sides of the comparison are subjects, so the checks
are self-consistent, per ADR-0014 §5a) — meaning any `host_id`/`player_id`/
`recorded_by_user_id` value written since T12.8 is a live subject string
that a backfill migration must resolve, not an empty table. This is real,
additional engineering surface T13.2/T13.3's tickets did not have to design
for, and it is why #164's own body separates step 3 (backfill) from step 1
(port) as its own numbered item with its own warning ("steps 2 and 3 must
land together or the comparison breaks in between").

**Stated so the sizing below is not understated:** this makes each
per-context ticket **larger** than T13.3 was, not equivalent to it — a
migration that must `UPDATE ... SET col = (SELECT id FROM identity_users
WHERE subject = <old text value>)` and decide what happens to a row whose
subject matches no `identity_users.id` (an orphaned actor fact — a real
possibility if a `Game`/`Competition`/`Payment` was written before its
actor ever called `CreateUser` through Identity, which this environment's
own flow allows) is a materially different, harder problem than translating
a not-yet-written value at write time. It remains **fully solvable inside
this environment** — nothing about resolving an orphan requires an external
IdP, it requires only a decision (leave `NULL`/leave unconvertable rows
`text`, or reject the migration if any orphan exists) — but it is real
scope, not busywork, and the sizing in §B7 reflects it.

**Caveat, stated for honesty rather than hidden:** this repo's own
Postgres migrations apply via `docker compose`'s `initdb.d`, **only on a
fresh volume** (CLAUDE.md gotcha). There is no persistent production
database in this environment to actually contain accumulated orphan rows
today — every `make down && make up` cycle starts empty. The backfill
migration's *logic* must be correct for a real deployment where data has
accumulated, and this ceremony sizes the work on that basis, but this
environment cannot exercise it against a realistic volume of pre-existing
rows the way a real deployment eventually will. This is disclosed rather
than treated as reducing the work, the same way T4's concurrency proof was
disclosed as "manually proven" rather than claimed proven under production
load.

## §B6 — Conclusion: #164 is genuinely unblocked, mis-classified for
roughly fourteen sprints

**The coordinating session's analysis holds, independently re-verified from
primary sources rather than trusted:**

1. #164's own text names four concrete, scoped requirements, none of them
   an external IdP (§B2).
2. ADR-0014 §5/§5a name the same target state and the same two open items,
   also without naming an external IdP as a precondition (§B3).
3. Every one of #164's own citations — the missing ports, the column types,
   the migration comments, `0016`'s anticipation of the FK target — checks
   out exactly as written against the current tree (§B4).
4. The resolution pattern (`port.IdentityLookup.UserIDBySubject` +
   `adapter/identity`, translated at the grpcapi boundary) is not
   hypothetical — it shipped twice already, reviewed and merged, entirely
   inside this environment, against this environment's own `identity_users`
   table (§B4, last row).
5. The one genuinely new element — backfilling rows that already hold live
   subject values — is real added engineering work, but it is a **data
   migration problem with a decision to make**, not a dependency on
   anything this environment cannot provide (§B5). `identity_users.id` and
   `identity_users.subject` — the entire backfill target and source — are
   both this backend's own schema, already populated by this environment's
   own dev-auth flow.

**#164 is therefore reclassified: genuinely unblocked, scoped engineering
work, mis-filed under "needs a real IdP tenant" since approximately T14 by
ceremony restatement rather than by anything in the issue's own text or
ADR-0014's own ruling.** This is corrected on the issue itself, per this
project's own standing rule that a correction to an earlier claim about an
issue is posted to the issue, not left only in a plan document (see §B8).

**#145 is not reclassified, and remains genuinely blocked — confirmed by
the same re-derivation, not merely left alone by default.** #145 is about
mapping **pre-existing `identity_users` rows whose primary key is already a
uuid** to a **real, non-uuid IdP `sub` claim** that does not exist anywhere
in this environment (no real IdP has ever issued a token in this
environment's history, per CLAUDE.md's own dev-only fixture warning). #164's
backfill runs in the opposite direction, entirely within this environment's
own schema (`text` subject values this environment's own dev-auth already
produced → `identity_users.id`, which this environment's own `CreateUser`
already mints). The two issues share a superficial "backfill" shape and
nothing else; conflating them is the exact error the fourteen-sprint
misclassification made by grouping #164 with #145 rather than testing each
against its own text.

## §B7 — Sizing: too large for one sprint at all three contexts; Wave 1
scoped honestly

Per the task's own instruction, this is sized realistically rather than
forced. The full #164 scope is 3 contexts × (new port + new adapter +
funnel wiring + backfill migration with an orphan-handling decision + FK
conversion + updating every authorization comparison that reads the
column, e.g. `ConfirmOnlinePayment`'s owner check at
`internal/payments/app/service.go:1057`, which directly compares
`actorUserID != p.RecordedByUserID` and would silently break in either
direction if the two sides of that comparison were resolved
inconsistently). That is comparable in shape to T13.2+T13.3 combined, times
three, with the added backfill/orphan complexity §B5 names. Dispatching all
three in one sprint would be the largest sprint this project has run since
T13 (40 points), after fourteen sprints during which dispatched capacity
has trended toward zero-to-single-digit tickets — taking three first-time,
backfill-shaped tickets at once on no prior evidence of what that backfill
work actually costs in this codebase is exactly the kind of overcommitment
this project's own PdE-role has pushed back on before (T13's Wave-1.5
disagreement, T12's A14 disagreement).

**Wave 1 (this sprint): one ADR + the single simplest context, as the
reference implementation — mirroring how T13.2 built the pattern once
before T13.3 copied it, and going one step more conservative given the
added backfill risk.**

- **Why Payments, and not Social Play or Competitions, as the reference
  context:** Payments has exactly **one** actor column
  (`recorded_by_user_id`), and it is **nullable** — the lowest blast radius
  of the three (Social Play has three columns across two tables; Competitions
  has two columns in one table). Proving the backfill-plus-FK-conversion
  mechanics once, on the smallest surface, before repeating it twice more,
  is the same risk-reduction logic A11's Wave-1.5 checkpoint used for
  ADR-0014 itself.
- **Social Play and Competitions conformance are explicitly NOT taken this
  sprint.** They remain within #164, narrowed rather than closed (§B8),
  and are the natural first candidates for T29's Ceremony 1 — by which
  point T28.1's ADR and its backfill-migration pattern will have been
  reviewed and merged, the same de-risking T13.3 got from T13.2 landing
  first.

### T28.1 — ADR-0017 (Social Play/Competitions/Payments conformance ruling) + Payments' Identity port, funnel wiring, and backfill (reference implementation)

**Story:** As a platform maintainer, I want Payments' `recorded_by_user_id`
to hold the same identifier space ADR-0014 already establishes for Booking
and Facilities, so that `ConfirmOnlinePayment`'s ownership check compares a
`User.ID` against a `User.ID` by design rather than a subject against a
subject by coincidence.

**Description:** Closes the Payments third of **#164** (a partial fix —
Social Play and Competitions remain, see §B8). Extends ADR-0014's decisions
to the one remaining open question its own §7 reasoning already leans on
but never formally applied to these three contexts: whether the
now-conformant columns become real `uuid` FKs. Establishes the backfill
pattern the next two contexts will copy.

**Instructions:**

1. **Write ADR-0017 first**, numbered `0017` (verified this ceremony:
   `docs/adr/` ends at `0016`). It must rule, once, for all three
   remaining contexts (even though only Payments is implemented this
   sprint — the ruling should not need re-litigating three times):
   - Confirm ADR-0014's existing "translate, not widen" ruling extends
     unchanged to these three contexts — re-derive §7's four reasons
     (leaks the IdP identifier, destroys the FK, makes an IdP swap a
     whole-database migration, and is not actually cheaper) against
     `recorded_by_user_id`/`host_id`/`player_id` specifically, rather than
     asserting they still apply.
   - Rule on #164's fourth open question: **yes**, once backfilled, these
     columns become real `uuid NOT NULL REFERENCES identity_users(id)` (or
     `uuid REFERENCES identity_users(id)`, nullable, for
     `payments.recorded_by_user_id` specifically) — following `0016`'s own
     anticipation and Booking's `recurring_hire_templates` precedent. State
     this as the target shape for Social Play and Competitions too, so
     T29 does not have to re-decide it.
   - Rule on the orphan case §B5 names: a text value with no matching
     `identity_users.subject` (an actor who wrote a row before ever
     calling `CreateUser`). Recommended default, to confirm or override
     with reasoning: leave the column nullable at the Postgres level even
     where the domain would prefer `NOT NULL`, and leave an orphaned row's
     new uuid column `NULL` rather than fail the migration outright — an
     unresolvable historical actor fact should not block deploying the fix
     for every resolvable one. Payments' own column is already nullable,
     which makes it the right first case to prove this against; Social
     Play/Competitions' `NOT NULL` columns will need this ADR's ruling
     restated against a stricter starting point.
2. **Build `internal/payments/port.IdentityLookup`**, mirroring
   `internal/booking/port/identity_lookup.go` and
   `internal/facilities/port/identity_lookup.go` exactly (both read in full
   this ceremony) — one method, `UserIDBySubject(ctx, subject string)
   (string, error)`, doc-commented with which identifier space it takes,
   returning the id as a plain string, never a `User`. Implement it once in
   `internal/payments/adapter/identity` against the real
   `identityapp.Service`, following the same two adapters' shape.
3. **Wire the funnel.** `internal/payments/adapter/grpcapi/handler.go`'s
   `actor(ctx)` (line 68, currently `return auth.RequireSubject(ctx)`)
   becomes a `*Handler` method that resolves through the new port, exactly
   as ADR-0014 §3's pattern describes and as Booking's/Facilities'
   handlers already do.
4. **Fix `ConfirmOnlinePayment`'s owner check to compare the right space on
   both sides.** Verified this ceremony:
   `internal/payments/app/service.go:1057` —
   `if actorUserID == "" || p.RecordedByUserID == "" || actorUserID !=
   p.RecordedByUserID`. Once the funnel resolves `actorUserID` to a
   `User.ID`, this comparison is broken until `p.RecordedByUserID` is also
   a `User.ID` — **this is exactly the "silently denies everybody or
   silently permits" hazard #164's own body warns about**, and it is why
   this ticket must land the funnel change and the backfill/column-type
   change together, not as two separate PRs with a window in between.
5. **Migration `0024_payments_recorded_by_user_id_uuid.sql`** (verified
   this ceremony: `db/migrations/` ends at `0023`, so `0024` is next and
   free — pre-assigned to this ticket and no other). Adds a new
   `recorded_by_user_id_uuid` column, backfills it via a join against
   `identity_users.subject`, leaves unmatched rows `NULL` (per ADR-0017's
   ruling), then — in the same migration, since this prototype's
   migration tooling has no separate rollout/backfill-then-swap phases
   (CLAUDE.md's own migration-tooling gotcha) — drops the old `text` column
   and renames the new one into its place. State explicitly in the PR
   whether a two-phase (expand/contract) approach was considered and why
   a single migration is or is not acceptable given this is prototype-only
   tooling applied on a fresh volume (CLAUDE.md gotcha) rather than a live
   production database today.
6. **Update every read/write site touching `RecordedByUserID`** (verified
   this ceremony: `app/service.go:284,560,1033,1039,1043,1057`;
   `adapter/grpcapi/handler.go:440`; `adapter/postgres/repository.go:47,
   52,60,69,81,117`; `domain/payment.go:77,84,97,129`) to the new type.
   `domain.Payment.RecordedByUserID` stays a Go `string` (a uuid string,
   not a Go UUID type — matching every other actor field in this codebase);
   only the Postgres column's type and the value it holds change.
7. **Mutation-check the backfill** (CLAUDE.md rule 10): seed a Payment with
   a resolvable subject and one with an orphaned (unmatched) subject,
   confirm the migration resolves the first to the correct
   `identity_users.id` and leaves the second `NULL`, not silently dropped
   or defaulted to an empty string.
8. **State the constructor/dependency outcome explicitly in the PR body**
   — whether `payments.app.NewService`'s `ServiceOptions` gains a new
   field for the identity port.
9. **Standing rule:** a partial fix does not close #164. This PR's title
   is **"partial fix for #164"**, and its body states plainly that Social
   Play and Competitions remain, per T13.3's own precedent narrowing #145
   without closing it. Post the same narrowing as a comment on #164
   itself at merge time (or state explicitly why not, per the "closes #N"
   review rule generalized to "partial fix for #N" review honesty) — see
   §B8.
10. Non-functional: `make test-domain` stays green; domain stays pure
    (CLAUDE.md rule 2); the invariant (a column and the comparison against
    it are always in the same identifier space) is enforced by the
    combined port-plus-migration change landing as one reviewed unit, not
    two.

**Cross-context dependency check:** calls `internal/identity` via the new
`port.IdentityLookup`; `identityapp.Service.UserBySubject` already exists
(`internal/identity/app/service.go:153`, same primitive T13.2/T13.3 already
consume) — re-verified present this ceremony.

**Story points:** 8. **Role:** principal-engineer. **Labels:**
`role:principal-engineer`, `type:chore`, `points:8`.

## §B8 — What happens to #164 itself, and what does not

**#164 is not closed by T28.1.** It is a **partial fix**: Payments'
conformance ships; Social Play's and Competitions' do not. Per this
project's own PR-title convention (§A5 of `docs/process/t13-sprint-plan.md`,
still binding), T28.1's PR title reads "partial fix for #164," and its
review/merge comment narrows #164 explicitly — the same treatment T13.3 gave
#145 — rather than leaving the issue's own text describing three contexts
as equally undone once one of them ships.

**Social Play and Competitions conformance are the explicit first
candidates for T29's Ceremony 1**, not re-opened as a fresh backlog
question — this plan states that now so T29 does not have to re-derive
the reasoning in §B5–§B7 from scratch, only re-verify it still holds.

## §B9 — Recorded disagreement (Ceremony 2 rule 3 — not smoothed over)

**PdE vs. PE — should T28 take all three contexts at once, now that the
pattern is proven twice (T13.2, T13.3) and the "IdP tenant" blocker is
confirmed false?**

- **PdE:** the blocker was never real; taking only one context this sprint
  under-corrects fourteen sprints of a wrong classification, and Social
  Play's and Competitions' columns sit exactly as non-conformant for
  another sprint boundary for no reason connected to the actual work.
- **PE:** the *port-and-funnel* half of the pattern is proven twice; the
  *backfill-against-live-subject-data* half (§B5) is proven zero times,
  because neither T13.2 nor T13.3 had any pre-existing rows to migrate.
  Taking three first-time-shaped backfill migrations in one sprint, on a
  column-type change that a broken authorization comparison (§B7 instruction
  4) makes actively dangerous if done inconsistently, is the over-commitment
  T12's A14 and T13's Wave-1.5 disagreements were both about — and both of
  those were resolved in favor of proving a new capability once before
  scaling it.
- **Resolved in PE's favor for this ceremony, with PdE's objection recorded
  rather than smoothed over.** T28.1 proves the backfill pattern on the
  smallest, nullable, single-column context. T29's Ceremony 1 is instructed
  to take Social Play and Competitions **together or separately, PE/PM's
  call at that time**, informed by how T28.1's backfill migration actually
  went — not pre-committed here, since this ceremony has no evidence yet of
  how long a real backfill-plus-FK-conversion migration takes to review in
  this codebase.

---

# Ceremony 1 — Backlog refinement (remaining sections)

## §A4 — Migration-header-ownership check

`0024_payments_recorded_by_user_id_uuid.sql` is pre-assigned to T28.1 alone
(§B7 instruction 5). No other T28 ticket exists to collide with it.

## §A5 — The whole open backlog, ranked, with a disposition for each

| Issue | Ranked | Disposition |
|---|---|---|
| **#164** ADR-0014 conformance (Social Play/Competitions/Payments) | **Reclassified — genuinely unblocked; Payments third taken as T28.1** | See §B in full. Not a real-IdP-tenant blocker; Social Play/Competitions remain, explicitly deferred to T29 with reasoning (§B7/§B9), not silently |
| **#144** `CancelBooking`/`CreateBooking` have no authz | **Escalated** | D1 unanswered; §A7. Sixteenth sprint carrying this |
| **#149** Payments' remaining caller-supplied fact (`booking_host_id` etc.) | **Untouched, correctly** | Blocked on D1 exactly as its own text says. Note: unrelated to #164 — #149 is about *ownership facts* Payments is told rather than resolves; #164/T28.1 is about the *identifier space* of the actor doing the telling. T28.1 does not narrow #149 |
| **#124** court-Bookings half of the cascade | **Untouched, correctly** | Blocked on D1, same reason as #149 |
| **#145** pre-existing UUID rows vs. `Principal.Subject` | **Re-verified, still genuinely blocked** | Confirmed by the same re-derivation applied to #164 — needs a real, non-uuid IdP `sub` claim that exists nowhere in this environment; see §B6 for why #164 and #145 diverge despite the superficial "backfill" similarity |
| **#126** real per-Game price field | **Deferred** | Needs Product Owner input; unchanged since T8.10 |
| **#130** refunding a `no_show_fee` Payment | **Deferred** | Carries its own stated open product question; unchanged |
| **#134** WCAG manual screen-reader pass | **Deferred** | Needs real assistive-technology hardware this environment cannot provide; T28 ships no UI change |

**Dependency-completeness check (T28.1's arrows):**

| Arrow | Upstream delivers | Downstream consumes | Complete? |
|---|---|---|---|
| T28.1's own ADR-0017 → T28.1's own implementation | The ruling (translate, FK-convert, orphan handling) | The migration's own shape | ✅ Complete — same ticket, ADR authored first per its own instruction 1, mirroring T13.2's structure |
| T28.1 → (a future T29 ticket for Social Play/Competitions) | ADR-0017's ruling, stated for all three contexts; the reviewed backfill-migration pattern | Social Play/Competitions' own port+adapter+migration | Not a T28 arrow — explicitly out of this sprint's scope (§B7/§B9); named here so T29's Ceremony 1 does not have to re-discover the dependency |

No other T28 ticket exists, so no other arrow to check.

## §A6 — Why one ticket, not zero and not three

Per §B7/§B9 in full: zero tickets would re-carry a classification this
ceremony independently found to be wrong, which is worse than taking no
action — it would mean the ceremony discovered the mistake and repeated it
anyway. Three tickets (all of #164 at once) would be sizing a first-time,
backfill-shaped capability against zero evidence of what that class of
migration costs in this codebase, the exact overcommitment pattern this
project's own retros have flagged before. **One ticket, the smallest and
lowest-risk of the three contexts, as a reference implementation, is the
honest middle scoping** — stated as a judgement call, not a mechanical
result, per PE's recorded position in §B9.

## §A7 — DECISION D1 and DECISION D2 remain unanswered

Re-verified this ceremony (§A3): #144 carries exactly one comment, T14.3's
original escalation, unchanged across T14 through T27, now T28. Both ADRs'
`## Status` headings are unchanged.

> **DECISION D1 — sixteenth deferral.** Unchanged from T27's own statement
> of it: who may cancel a court booking made through the public,
> unauthenticated quote-and-book flow, and should that flow remain
> unauthenticated at all. `docs/adr/0015-*.md` lays out four options and
> recommends none. **No T28 ticket implements this** — T28.1 touches
> Payments' actor-identifier space, not Booking's ownership model, and does
> not narrow D1 in any way. D1 has now carried its single T14.3 comment for
> **fifteen consecutive sprints (T14 through T28)**.

> **DECISION D2 — unchanged from T27's statement.** May a session that
> reviews and merges a PR also author code on it. `docs/adr/0016-*.md`
> recommends none of its four options. **T28 has one real ticket this
> time** (T28.1), so — unlike T20 through T27's null-result sprints — this
> is the first sprint since T19 where D2's interim rule (reviewer performs
> or independently re-derives) will actually be exercised again, not merely
> carried as an unexercised deferral. T28's retro is instructed to score
> which of T15–T19's five "exercised, no fix needed" instances or T20–T27's
> nine "no PR existed" instances this sprint's own review most resembles,
> rather than defaulting to the weaker shape by habit.

**Neither is implemented, decided, or guessed at by this ceremony.**

## §A8 — Shared-file pre-assignment, and same-wave verification

| Artifact | Owner | Notes |
|---|---|---|
| `HANDOFF.md` | this ceremony only | An implementer that finds a stale line flags it for T29's Ceremony 1 |
| `docs/process/sprint-process.md` | not touched this ceremony | No amendment landed |
| `docs/adr/0017-*.md` | T28.1 only | New file, single ticket |
| `internal/payments/port/identity_lookup.go`, `internal/payments/adapter/identity/` | T28.1 only | New files, single owner |
| `db/migrations/0024_*.sql` | T28.1 only | Pre-assigned, §A4 |

**Same-wave shared-interface verification rule: does not apply.** T28 has
exactly one ticket — there is no second same-wave ticket to collide with.

---

# Ceremony 2 — Sprint planning

## Sprint goal

> Correct the record for T27. Independently re-verify, from #164's own
> issue text, ADR-0014's own ruling, and the current codebase — not from
> fourteen sprints of restated classification — whether #164 is genuinely
> blocked on an external identity provider. It is not: it is scoped,
> repeatable engineering work following a pattern this project has already
> shipped twice (T13.2, T13.3). Take the smallest, lowest-risk third of it
> — Payments' conformance — as this sprint's real scope, sized honestly
> rather than forcing all three contexts into one sprint on zero evidence
> of what the added backfill work costs. Re-verify the other 7 issues'
> blockers live and find them unchanged, with #145 specifically
> re-confirmed as genuinely IdP-blocked despite its superficial similarity
> to #164.

**What this sprint does not claim:**

- **This is not a claim that #164's remaining two-thirds (Social Play,
  Competitions) are now trivial.** They follow the same pattern but against
  `NOT NULL` columns and more of them; T29 re-verifies rather than assumes
  this.
- **This does not touch D1 or D2.** Both remain exactly as escalated as
  T27 left them (§A7). No T28 ticket implements `CancelBooking`/
  `CreateBooking` authorization or a reviewer-authorship carve-out.
- **This is not a retroactive indictment of T14 through T27's ceremonies**
  for not catching this sooner — each accepted a classification that had
  already been independently established and never had a specific reason
  to re-derive it from primary sources, the same way this project's own
  #97/#149 misattribution incidents were caught only when something
  specific prompted a re-read. What changes this sprint is that something
  specific did: the user's direct flag.
- **This does not mean #145 was wrongly classified too.** §B6 confirms
  #145 is different in kind, re-derived independently rather than assumed
  safe by association with #164.

## Tickets — 1 item, 8 points

**T28.1** — ADR-0017 + Payments' Identity port, funnel wiring, and backfill
(reference implementation, partial fix for #164). Full text: §B7. 8 points,
`role:principal-engineer`, `type:chore`.

## Waves

**Wave 1 (only wave): T28.1.** No same-wave collision — it is the only
ticket.

## Recorded disagreements (Ceremony 2 rule 3 — not smoothed over)

**PdE vs. PE, on sprint size** — recorded in full at §B9. Resolved in PE's
favor for this ceremony (one context, not three), with PdE's objection
that this under-corrects the fourteen-sprint misclassification's cost
carried forward rather than dismissed.

## Sprint-level Definition of Done

1. **T28.1 merged per the per-ticket DoD** (`sprint-process.md`'s
   Execution section): acceptance criteria met, PR reviewed (findings
   addressed or explicitly deferred), `make test-domain` green at minimum,
   approved and merged by the user or an explicitly delegated gate — never
   self-merged.
2. **The merged-fix issue sweep run and reported** — by the retro
   (reporting) and again by T29's Ceremony 1 (authoritative).
3. **#164 narrowed by an explicit comment at merge time**, per §B8 —
   scored at the retro: did the merge/review actually post the narrowing
   comment, or only state an intention to (the same "closes #N" vs.
   "partial fix for #N" review-honesty distinction `sprint-process.md`
   already draws, applied here for the first time to a "partial fix"
   PR rather than a full close).
4. **Scoring owed at the retro:**
   - **(a)** Did T28.1 ship the funnel change and the backfill/column-type
     change together, as one reviewed unit (§B7 instruction 4's hazard),
     or did any window exist where `ConfirmOnlinePayment`'s comparison was
     broken?
   - **(b)** Was the backfill mutation-checked per CLAUDE.md rule 10 (an
     orphaned subject and a resolvable one both exercised, not merely
     asserted)?
   - **(c)** Do the other 7 issues' blockers still hold, re-verified live
     at retro time rather than re-read from this document (the standing
     practice, unbroken since T12)?
   - **(d)** Did D1 or D2 get answered mid-sprint, as a formal ADR
     decision?
   - **(e)** Given T28 has a real PR to review, which of D2's two named
     shapes (§A7) does its review actually land in — scored explicitly
     rather than left to be inferred from the PR's own silence.
5. **Not scoreable by T28 and deliberately not pre-empted:** D1 and D2
   remain the user's own decisions. Whether T29 takes Social Play,
   Competitions, or both is PE/PM's call at that time, informed by how
   T28.1's migration actually went (§B9) — not pre-committed here.
6. Retro in `docs/process/t28-retro.md`, indexed by a `## T28 sprint retro`
   stub in `docs/LESSONS.md`. `HANDOFF.md`/`CLAUDE.md` state updated —
   noting that T29's Ceremony 1 corrects T28's Docs-index row.
