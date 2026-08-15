# T15 Sprint Plan — Ceremonies 1 + 2

Backlog refinement (Ceremony 1) and sprint planning (Ceremony 2) per
`docs/process/sprint-process.md`, read in its **T14-amended** form (PR #175 —
the per-PR closure DoD step, the merged-fix issue sweep, the decided label
taxonomy, and the scheduled-removals table, all of which govern this ceremony
and one of which this ceremony must *execute*). Held against
`docs/process/t14-retro.md` (PR #184, merged 14:07:35Z), `HANDOFF.md` including
its Docs index, `CLAUDE.md`, and the live PR/issue state of
`nhuthuynh/white-label` (GitHub-side name `pickleball-platform`).

**Every factual claim below was re-derived against the repository during this
ceremony**, at the merged branch tip `a5aaee3`, rather than taken from the
retro's or T14's plan's prose — CLAUDE.md rule 10 applied to planning. Where
this ceremony repeats a claim it could not independently check, it says so.

## §A0 — The merged-fix issue sweep, run as this ceremony's first act

`sprint-process.md`'s Definition of Done designates the next sprint's Ceremony 1
as the sweep's **authoritative** moment — *"where the 'party other than the
merger' property actually lives"*. T14's retro (finding 1) records that T14's
own closes bypassed both sanctioned moments, so this run is the first time that
property has actually been obtained on this project. It is reported with its
counts, per the rule that *"a sweep whose output is silence is
indistinguishable from a sweep that was never run"*.

**Step 1 — list the open issues.** `list_issues(owner: "nhuthuynh", repo:
"white-label", state: OPEN)` → **`totalCount: 13`**, numbers #124, #125, #126,
#130, #134, #137, #144, #145, #147, #149, #164, #167, #168.

**Step 2 — reconcile arithmetically, before reading any issue individually.**
`open_at_end_of_T13 − closed_in_T14 + opened_in_T14` = `19 − 6 + 0` = **13**.
Matches `totalCount` exactly. The six closes (#131, #156, #157, #158, #160,
#165) are all absent from the open list, verified by their absence rather than
by trusting the retro's table.

**Step 3 — cross-reference every merged T14 PR against the open list.** PRs
merged into `claude/go-backend-pickleball-7up34j` with `merged_at` inside the
sprint window were listed and their titles and bodies scanned for every `#N`.
Five open issues are referenced. **Every one of them is correctly open, and
none is a close this sweep must catch up:**

| Issue | Referenced by | Disposition | Sentence written? |
|---|---|---|---|
| #147 | #183 (title: *partial fix for #147*) | Partial fix — Social Play half only | **Now yes** — written by this ceremony |
| #168 | #182, #183 (both *partial fix*) | Partial fix — Social Play store only | **Now yes** — written by this ceremony |
| #149 | #183 (body, as the untouched twin) | Untouched; materially unblocked | **Now yes** — written by this ceremony |
| #164 | #182 (body, as the deferral target) | Correctly deferred; scope grew | **Now yes** — written by this ceremony |
| #144 | #176 (title: *escalated not decided*) | Escalated, awaiting the user | Already written, in-sprint, at 07:01:03Z |

**Sweep result: clean — zero issues require closing, on a list of 13.** That is
stated as a number rather than as silence.

**But the sweep's *second* disposition was not clean, and this ceremony
discharged it rather than re-reporting it.** The partial-fix disposition
requires that a correctly-open issue carry a written reason naming its
successor — *"it is only correct once someone has written the sentence"* — and
T14 ran it 1 of 4 times (retro finding 1). The three missing sentences are now
on the issues themselves, not only in a plan document a reader of the issue
list never opens:

- **#147** — comment posted; **re-titled** (recommendation 6) and **labelled**
  (recommendation 5).
- **#168** — comment posted, with a three-row table mapping its own three named
  constraints to their status and successor tickets.
- **#149** — comment posted, with the five ownership facts split into the two
  T15 takes and the three it does not; **labelled** (recommendation 5).
- **#164** — comment posted recording that T15.3 grows its backfill scope by one
  table, and why T15 is nevertheless not taking it.

**One issue opened by this ceremony**, mandatory rather than discretionary
under the board-of-record rule (recommendation 4):

- **#185** — *A well-formed but unknown `court_id` answers `Internal` (500), not
  a client error.* This is the residual T14.8 disclosed and misattributed to
  #97 (which is closed, and was never about this gap). It had been tracked
  nowhere. **The claim was verified for the filing rather than copied from PR
  #178's prose** — see §A2.

**Post-sweep open count: 14** (13 + #185). Recorded now so T16's Ceremony 1 has
the starting number its own arithmetic needs.

## §A1 — Correcting T14's Docs-index row and Task-backlog narrative

Ceremony 1's designated first job (`sprint-process.md`, *"Correct the previous
sprint's Docs-index row"*). T14's row was written before T14 ran, so its Retro
and Reviews cells still said "not yet written" / "not yet opened"; a retro PR
structurally cannot write its own row, because the row must cite the retro's own
merge number. PR #184's body confirms it deliberately left `HANDOFF.md`
untouched for this reason.

**PR order verified against each PR's `merged_at`, not assumed from numbering** —
this project's standing convention, because numbers and merge order routinely
disagree. They were fetched individually for this ceremony:

| PR | Ticket | `merged_at` |
|---|---|---|
| #174 | Ceremony 1/2 doc | 06:55:10Z |
| #175 | T14.6 | 07:01:12Z |
| #176 | T14.3 | 07:02:14Z |
| #177 | T14.2 | 07:04:19Z |
| #178 | T14.8 | 07:11:48Z |
| #179 | T14.7 | 07:19:49Z |
| #180 | T14.9 | 13:16:08Z |
| #181 | T14.1 | 13:23:26Z |
| #182 | T14.4 | 13:28:48Z |
| #183 | T14.5 | 13:48:47Z |
| #184 | retro doc | 14:07:35Z |

**For T14, merge order and numeric order agree** — which, as T13's row records
of its own sprint, was only knowable by checking. The row states it as verified
rather than implying the numbers were trusted.

**Three further corrections this ceremony makes, beyond the standing three:**

1. **`HANDOFF.md:36` and `:609` say ADR-0015 records "three options". It
   records four.** Re-verified directly: `docs/adr/0015-*.md`'s "The options"
   table carries rows **(a)** authenticate the flow, **(b)** guest capability
   token, **(c)** owner-when-known, and **(d)** authenticate cancellation only —
   with a note that "(d) is new in this ADR". PR #176's implementer correctly
   declined to touch an unassigned shared file and flagged it; PR #176's
   reviewer wrote *"I'll fix that stale reference directly"* and did not. Fixed
   in both places here (recommendation 9).
2. **T14's Task-backlog narrative** is added in the **retro's own agreed form**,
   quoted from `docs/process/t14-retro.md`'s "The sprint goal, scored" section —
   `sprint-process.md` Ceremony 1 item 3 requires the retro's form, **not a
   stronger one**. That paragraph's bolded middle sentence (two of nine tickets
   authored, reviewed and merged by one session, 14 and 13 seconds open to
   merge) and its closing sentence (all six closes correct, all six in an
   eleven-second batch by the merging party, per-PR step 0/6, no independent
   check) are carried verbatim in substance. **This ceremony does not soften
   them, and does not add a stronger claim of its own.**
3. **A new T15 row**, written in the same before-the-sprint form, with its Retro
   and Reviews cells honestly marked "not yet written" / "not yet opened" — which
   T16's Ceremony 1 will correct in turn.

## §A2 — What this ceremony verified, and how

Every item below was run or read during this ceremony. Items marked **not
re-verified** are stated so rather than glossed.

| Claim | How it was checked | Result |
|---|---|---|
| The gate-coverage check still works | `make generate` (exit 0) then `make gate-coverage` | `41 package(s) hold test functions … OK — all 41 … executed by "ci-checks"`; side B printed as 5 `go test` invocations parsed from the `Makefile` |
| It is wired into the gate | read `Makefile:326` | `ci-checks: generate tidy fmt-check lint test-domain test-platform vet-integration test-tools test-adapters test-cmd gate-coverage generate-client lint-web test-web build-web` |
| Toolchain present | `go version`, `which buf sqlc` | `go1.25.0`, both present |
| Open-issue count | live `list_issues(state: OPEN)` | `totalCount: 13` |
| #97's real state and subject | `issue_read` | `closed`, `state_reason: completed`, `closed_at: 2026-08-13T14:45:54Z`; subject is **malformed**-ID guards on write handlers — not the gap PR #178 attributed to it |
| No Competition-Admin anything exists | `grep -rli "competition_admin\|CompetitionAdmin" db/migrations internal/competitions proto` | matches **only** `proto/pickleball/payments/v1/payments.proto` (the caller-supplied list, i.e. #149) |
| Competitions' roster read is unchanged | read `internal/competitions/app/service.go:467` | `ListEntriesForCompetition(ctx, competitionID, actorUserID string)`, bare Host check |
| Social Play's pattern is complete and *readable from outside the context* | read `internal/socialplay/app/service.go`, `internal/socialplay/port/game_admin_repository.go` | `app.Service.ListGameAdmins` exported at `service.go:1009`; `port.GameAdminRepository` carries `ListGameAdmins` |
| Payments has no read-side ports | `ls internal/payments/port` | only `idgenerator.go`, `payment_processor.go`, `repository.go` |
| **#185's defect is real** | read `internal/booking/app/service.go:252` (shape-only guard), `db/migrations/0001_init.sql:23` (`court_id uuid NOT NULL REFERENCES courts (id)`), `grep -n "23503\|foreign" internal/booking/adapter/postgres/*.go` → **no matches**, `internal/socialplay/adapter/booking/reservation.go:63` (`%s`, not `%w`) | An unknown-but-well-formed court id fails the FK, is never translated to a domain error, is stripped at the context boundary, and lands in `default: codes.Internal` |
| ADR-0015 records four options | read `docs/adr/0015-*.md` "The options" | four rows, (a)–(d), with (d) marked as new |
| ADR-0015's D1 is untouched | `grep` `db/migrations/0001_init.sql` for an owner column (**none**); read `internal/booking/app/service.go:287`; read `internal/booking/adapter/grpcapi/authenticated.go:68-75` | `bookings` has no owner column; `CancelBooking(ctx, bookingID string)` takes no actor; **both** `CreateBooking` and `CancelBooking` are still in `PublicMethods()` |
| ADR-0012's Q1/Q2 are untouched | `grep -rn "PlayerRating\|Gender" internal proto db` | every occurrence is a doc comment stating the field deliberately does not exist (`identity/domain/user.go:11,18`, `socialplay/domain/match.go:7,13`, `socialplay/domain/errors.go:158`, `db/migrations/0016_identity.sql:5-6`). No `Gender` field, no `PlayerRating` type, no Level formula |
| #144 has received no answer | `issue_read(get_comments)` on #144 | exactly **one** comment, T14.3's escalation at 07:01:03Z. No user reply |
| Next free migration / ADR numbers | `ls db/migrations`, `ls docs/adr` | `0020` is the last migration; `0015` the last ADR → **`0021`** and **`0016`** are free |

**Not re-verified by this ceremony, and named as such:** T14.4's Host-only
mutation check (the retro states it did not re-perform it either, so it remains
verified only by its author); the full `make test` / `ci-integration` Docker
path, since no Docker daemon is reachable here; and any claim about what Jenkins
would run, since no Jenkins job exists.

---

# Ceremony 1 — Backlog refinement

## §A3 — T14 retro's ten recommendations, each given a disposition

None is left unaddressed. "Ticketed" means a T15 ticket owns it; "executed
here" means this ceremony performed it; "escalated" means it leaves this team's
authority.

| # | Recommendation | Disposition |
|---|---|---|
| 1 | Name the closure sweep's **third state** and score it honestly; do not re-exhort the per-PR half a third time | **Ticketed — T15.1**, and the per-PR half's *shape* is changed rather than repeated (§A4) |
| 2 | Adopt **worktree-recovery-after-session-limit** as a named practice with one safeguard | **Ticketed — T15.1**, with the authorship half carved out to T15.2 for a reason given in §A5 |
| 3 | Rule 9's "a reviewer never commits" needs **enforcement or an honest carve-out** | **Ticketed as an escalation — T15.2** writes ADR-0016 and puts it to the **user**; the team does not decide it (§A6) |
| 4 | Add "state each named issue's current state, read from the API" to the review enumeration; **file T14.8's residual** | **Both**: clause ticketed in T15.1; residual filed as **#185** by this ceremony, and ticketed for fixing as **T15.6** |
| 5 | **Sweep the label taxonomy** across the whole open list once, then stop hand-listing | **Executed here** (§A8). No checker built — the recommendation explicitly says not to build one pre-emptively |
| 6 | Take the **Competitions half** of #168/#147; **re-title #147** | **Ticketed — T15.3 + T15.4**; #147 re-titled, relabelled and commented by this ceremony |
| 7 | **Drop A5's dual coverage question.** Do not renew, adapt, or fold it | **Executed here** (§A7). Condition verified met; the question is not asked anywhere in this plan |
| 8 | **#144 is on its third sprint** — put D1 to the user as its own item | **Executed here as an escalation** (§A9). Not ticketed for implementation, because implementing it would mean guessing the answer |
| 9 | Correct `HANDOFF.md`, including the stale **"three options"** reference | **Executed here** (§A1) |
| 10 | Do not treat **"zero issues opened"** as a quality signal without checking it | **Executed here** (§A15), and threaded into every ticket's instructions as a standing requirement |

## §A4 — The closure sweep's third state (recommendation 1)

T14's retro declines to score T14's closure record as either branch of the
scoring condition `sprint-process.md` wrote. The measured facts, re-checked
against the retro's own table and this ceremony's arithmetic: six closes, all
correct, all cited, all landing inside an **eleven-second window** 65 seconds
after the sprint's last ticket merged, performed by the merging party, between
26 minutes and 6h45m after the PRs that earned them. The per-PR half scored
**0/6**, after **0/9** in T13.

**PE's cost argument is confirmed by measurement and PE's remedy is not.** The
issue list misdescribed the codebase for up to 6h45m — *inside* the sprint,
which is worse than the between-sprints window PE predicted. And the mechanism
that produced the correct outcome was the named sprint-level task, not the
per-PR step PE wants made primary.

**This ceremony does not re-exhort the per-PR step, because the retro
explicitly asks it not to** (*"it has now scored 0/9 and 0/6 in consecutive
sprints while every other quality signal was green"*). Instead T15.1 changes its
shape: the per-PR step stops being described as *"the moment the close
happens"* and becomes an **optional early close**, with the named sprint-level
sweep promoted to the primary mechanism — which is what the evidence supports
and what resolves the recorded PE/PO disagreement in the direction two sprints
of measurement point.

The third state itself is named and classified rather than declared a pass:
**acceptable-but-not-sufficient**, with the consequence stated in text — when
the merging session sweeps its own work, the *"party other than the merger"*
property is **not** obtained, and the next Ceremony 1's run is **not** thereby
discharged. This ceremony's §A0 is the first run of that re-run rule, and it
found the three unwritten partial-fix sentences that T14's self-sweep did not.

## §A5 — Worktree recovery after a session limit (recommendation 2)

The retro's verdict is **adopt-as-named-practice with a stated safeguard**, and
it earned that verdict: the practice saved an 8-point ticket (T14.4) whose only
copy was an unpushed local commit — T11.4's exact shape, the one case
`docs/LESSONS.md` says branch-listing cannot catch.

**This ceremony's judgement: yes, it needs a `sprint-process.md` amendment, and
it should land as a ticket rather than inside this planning PR.** The precedent
is direct and recent — T14.6 was a *ticket* that amended `sprint-process.md`
via its own reviewed PR (#175), not a section folded into T14's Ceremony 1/2
document. A rulebook change that arrives inside a planning PR is reviewed as
part of a plan; one that arrives as a ticket is reviewed as a rule. So T15.1
drafts it.

**But the practice as the retro words it cannot be adopted whole, and this is a
real seam rather than a technicality.** Its clause (b) says *"the recovering
session may commit, push and open the PR"* — and in both observed cases the
recovering session **was** the reviewing/merging session, which is precisely
what CLAUDE.md rule 9 forbids. Writing (b) into `sprint-process.md` as-is would
be a process document granting an exception to a golden rule in the project's
own rulebook. `sprint-process.md` is not the right altitude for that, and the
team is not the right author for it (§A6).

**So the practice splits, and T15.1 takes only the half that needs no
exception:**

- **(a)** check the interrupted session's worktree for unpushed commits *and
  uncommitted work* before re-dispatching — branch-listing is not sufficient.
  **Adopted in T15.1.** Pure detection; no rule-9 interaction.
- **(b)** a mandatory first-line provenance note, with #181/#182's wording as
  the template. **Adopted in T15.1.** T14 did this correctly and unprompted.
- **(c)** the safeguard: a recovered PR is reviewed by a **different** party
  than the one that recovered it; where none can be dispatched, the recovering
  session **says so plainly** and the sprint's retro independently re-derives
  that PR's headline claim and records the result. **Adopted in T15.1.**
- **(d)** delete the sentence *"I am not merging this myself either"* from any
  PR the author intends to merge — *"a written safeguard that does not exist is
  worse than an acknowledged absence."* **Adopted in T15.1.**
- **The permission question inside (b)** — may the reviewing/merging session
  author the code at all? — **is not settled by T15.1.** It goes to ADR-0016
  (T15.2) with the reviewer-authored-fix case, because it is the same question.

## §A6 — The reviewer-never-commits question: escalated to the user, not decided here (recommendation 3)

This is the disposition this ceremony thought hardest about, so its reasoning
is recorded in full rather than asserted.

**The facts, from the retro and re-read here.** Three of nine T14 PRs contain
code the reviewing party wrote: #181 and #182 (recovered from interrupted
sessions) and #179, where the review says it *"**Fixed it directly on the PR's
source branch**, matching this session's established practice for cross-PR gaps
found at merge time."* CLAUDE.md rule 9 says, in terms: *"A reviewer/QA/PE
agent's job is to report findings, never to commit or push itself."*

**The retro asks for "enforcement or an honest carve-out — pick one and write it
down"**, and sketches what a bounded carve-out might say (*mechanical, compiler-
or test-caught, single-file, disclosed in the review, and the fix itself
re-verified*). Read narrowly, that is an invitation for this ceremony to pick.
**This ceremony declines to pick, for three reasons that are about authority
rather than difficulty:**

1. **CLAUDE.md is a different altitude from `sprint-process.md`.** Rule 9 is a
   golden rule in the project's durable rulebook, whose header says these
   instructions **override any default behavior**. A carve-out is an amendment
   to that rulebook. `sprint-process.md` describes how this team works *within*
   the rules; it cannot grant itself relief from one.
2. **Rule 9's own text forecloses the team writing its own exception.** It
   reads: *"Nothing is 'low-risk enough' to skip this on its own judgment —
   that judgment call is exactly the failure mode this rule exists to remove."*
   That clause was added after a *second* incident, when the "just docs, low
   risk" judgement was made. A carve-out drafted by the agents the rule
   constrains, on the grounds that these particular commits were mechanical and
   low-risk, is that same judgement for a third time. The retro's own framing
   supports this reading: it observes the practice is *"currently both
   simultaneously, which is the state that guarantees it drifts"* — and an
   in-team carve-out is drift with a signature on it.
3. **The project already has a device for exactly this, and T14 used it well.**
   ADR-0015 states a question a non-engineer can answer, lists options with
   costs, **picks none**, ties its trigger to the answer arriving rather than a
   sprint boundary, and carries a restriction list forbidding future PRs from
   guessing. The retro calls it *"a model escalation."* The reviewer-authorship
   question has the same shape: it is a question about how much independence the
   user wants in their own review process, and the party that benefits from a
   loosening should not be the party that grants it.

**Decision: T15.2 writes ADR-0016 and escalates to the user as DECISION D2.**
The ADR states the question, records the three observed instances with their
measurements, lays out the options (strict enforcement, where a reviewer finding
a gap requests changes instead of fixing it; a bounded carve-out with the
retro's five conditions; and a recovery-only carve-out narrower than either),
states the cost of each, and **picks none**.

**The interim rule, which is not a decision.** Until D2 is answered, T15 obeys
rule 9 as written: a reviewer that finds a gap **requests changes**; it does not
fix the branch under review. This needs no permission — it is the existing rule,
followed. T15.2's instructions require that this be stated in the ADR so nobody
reads the escalation as a suspension.

**Recorded disagreement — PE vs. PO, per Ceremony 2 rule 3.** PE's position:
the retro said "pick one", the bounded carve-out is obviously correct for
mechanical single-file fixes caught at test-merge time, and escalating a
question the team can answer costs a sprint of ambiguity. PO's position: the
rule's own text names this exact judgement as the failure mode, two prior
tightenings both followed an agent deciding a case was low-risk, and a rulebook
the constrained party edits is not a rulebook. **Resolved in favour of PO on
authority, with PE's substance preserved** — PE's proposed carve-out text is
written into ADR-0016 as a fully-specified option the user can simply approve,
so choosing it costs one word rather than another sprint.

## §A7 — The dual coverage question is dropped (recommendation 7)

`sprint-process.md`'s "Scheduled removals" table names this ceremony as the
remover and states the condition as **"T14.1 merges"**, with the removal
described as executing a plan rather than re-making a judgement.

**Condition verified met, three ways, by this ceremony:**

1. T14.1 merged — PR #181, `merged_at` 13:23:26Z (fetched, not assumed).
2. `make gate-coverage` runs green here: `41 package(s) … OK`, with side B
   printed as five `go test` invocations parsed out of the `Makefile` itself.
3. It is reachable from the gate: `Makefile:326` lists `gate-coverage` in
   `ci-checks`.

**The question is therefore not asked in §A12's dependency-completeness check,
and does not become a fourth standing planning question.** Recorded emphatically
because this project's named failure mode is accumulating two shapes of one fix,
and the table exists so the removing ceremony does not re-litigate it: keeping
the question would be the failure the table was built to prevent.

**Where its job now lives.** T15 creates at least one new adapter package
(T15.5's read-side adapters) and new test files in existing packages (T15.3,
T15.4, T15.6, T15.7). Under the old practice a human would have been asked which
in-flight tickets' outputs each gate must cover. Instead, `make gate-coverage`
answers it at run time in `ci-checks`, and **T15.1's amendment records the
removal in the table with today's evidence**, so the row becomes history rather
than disappearing silently.

## §A8 — Label taxonomy swept across the whole open list (recommendation 5)

The taxonomy (`role:` mandatory, `type:` mandatory and closed-set, `context:`
optional and closed-set) landed in T14.6 applied to a hand-written list of
three — in the sprint whose marquee ticket exists to prove hand-written lists go
stale. Two of the thirteen open issues carried **no labels at all**, and they
were #147 and #149, the two T14's own plan discussed at length.

**Swept here across all 13, not a hand-picked subset:**

| Issue | Before | Action |
|---|---|---|
| #147 | **no labels** | `role:product-engineer`, `type:bug`, `context:competitions`, `context:socialplay` |
| #149 | **no labels** | `role:principal-engineer`, `type:bug`, `context:payments` |
| #185 (new) | — | `role:product-engineer`, `type:bug`, `context:booking`, `context:socialplay`, `context:competitions` — conformant at creation |
| #124, #125, #126, #130, #134, #137, #144, #145, #164, #167, #168 | `role:` + `type:` present, all in the closed sets | none needed — checked individually against the live label list, not assumed |

**No `make label-conformance` check is built.** Recommendation 5 is explicit:
one sweep first, and only if a **third** instance of label drift appears is a
mechanical check the answer. Building it now would be the two-shapes failure
mode again, one sprint after the table that exists to prevent it was honoured.
Recorded here so T16 can recognise the third instance if it comes.

## §A9 — DECISION D1 (#144) is on its third sprint. Put to the user as its own item, not implemented

**Status verified this ceremony, from the code and from the issue:** `bookings`
has no owner column of any kind; `app.Service.CancelBooking(ctx, bookingID
string)` takes no actor parameter; `CreateBooking` **and** `CancelBooking` are
both still in `PublicMethods()`. #144 carries exactly one comment — T14.3's
escalation, posted 07:01:03Z. **No answer has arrived.**

**ADR-0015 wrote this moment's rule in advance, and it binds this ceremony:**

> *"**If D1 is unanswered when T15 plans, that is a finding, not a decision.**"*

**This ceremony records it as a finding.** It is #144's third deferral. The hole
is unchanged and is stated plainly rather than softened: **anyone who knows a
booking id can cancel that booking, with no authentication of any kind.**

**Raised to the user as its own item, per recommendation 8** — which asks
specifically that it not be a line inside a plan document:

> **DECISION D1 (for the user / Product Owner).** When somebody books a court
> through the public flow *without an account* — which is how the flow works
> today — who should be allowed to cancel that booking later, and should
> booking without an account remain possible at all?
>
> `docs/adr/0015-booking-ownership-for-public-bookings.md` lays out **four**
> options with their costs and **recommends none**. Option (d) — authenticate
> cancellation only, leave creation public — is the one that closes the sharp
> half without touching the shipped T7.6 booking UI, and it exists because
> T14.3 verified that nothing in `web/src` calls `CancelBooking` today. **This
> is a product decision, not an engineering one, which is why three sprints of
> engineers have declined to guess it.**

**No T15 ticket implements D1**, and per ADR-0015's restriction list no T15 PR
may guess it. If the answer arrives mid-sprint the implementation is small and
well-understood (an owner column, an actor parameter on `CancelBooking`, the
RPC leaving `PublicMethods()`), and ADR-0015's trigger — *the sprint
immediately following the answer* — takes over.

## §A10 — ADR-0012's Q1/Q2 remain blocked, and remain untouched

Q1 (how a Player Level is computed and weighted) and Q2 (whether gender-mix
matching is in scope at all) are blocked on the user and are a different class
from D1: ADR-0015 distinguishes them as *legally/ethically blocked* rather than
*ordinary-but-unanswered*.

**Re-verified this ceremony by grep across `internal`, `proto` and `db`**: every
occurrence of `PlayerRating` or `Gender` is a doc comment stating the field
deliberately does not exist (`internal/identity/domain/user.go:11,18`;
`internal/socialplay/domain/match.go:7,13`;
`internal/socialplay/domain/errors.go:158`;
`db/migrations/0016_identity.sql:5-6`). **No `Gender` field, no `PlayerRating`
type, no Level-scoring formula anywhere in the tree.** Seventh consecutive
sprint recorded this way. **No T15 ticket touches them, and no T15 PR may guess
at them.**

## §A11 — The whole open backlog, ranked, with a disposition for each

All 14 open issues (13 from the sweep plus #185, opened here). "Taken" means a
T15 ticket owns it; every deferral carries its reason, per the board-of-record
rule that a deferral without a written reason is a process violation.

| Issue | Ranked | Disposition |
|---|---|---|
| **#168** Game/Competition-Admin store | **Taken** | T15.3 + T15.4 + T15.5 together close it — see the three-row table on the issue |
| **#147** Competitions' roster read | **Taken** | T15.4 closes it. Re-titled + labelled here |
| **#149** Payments' caller-supplied facts | **Partly taken** | T15.5 resolves the two **admin-list** facts; the three host/entrant facts stay open with the sentence written on the issue |
| **#185** unknown court id → 500 | **Taken** | T15.6. Opened by this ceremony |
| **#137** No remote JWKS `KeySource` | **Taken** | T15.7. Buildable here in full — `httptest` plus locally-minted keys, no external tenant needed — and it removes one of #164's two blockers |
| **#144** `CancelBooking` has no authz | **Escalated** | D1 unanswered; §A9. Recorded as a **finding**, third deferral |
| **#164** ADR-0014 actor-column conformance | **Deferred, scope grew** | Needs a backfill against real IdP subjects; #137 and #145 sit in front of it. T15.3 adds a third table to its scope — recorded on the issue |
| **#145** pre-existing UUID rows vs. `Principal.Subject` | **Deferred** | Its own title says *"Once a real IdP exists"*. Not reachable this sprint even with T15.7 |
| **#124** `Game.Cancel()` does not cascade | **Deferred — and this is a judgement, see below** | Its court-Bookings half must call `app.Service.CancelBooking`, whose signature and authorization model are the subject of the **unanswered** D1. Building against a signature slated to change means either guessing D1 or knowingly writing throwaway code |
| **#125** Competitions-shaped `PayableType` | **Deferred** | Untouched by T15's contexts; no dependency created or removed this sprint |
| **#126** real per-Game price, retiring the placeholder | **Deferred** | Product input needed on pricing shape; no engineering blocker, but no T15 ticket needs it |
| **#130** refunding a `no_show_fee` | **Deferred** | Out of T12.3's stated scope and out of T15's; Payments' T15.5 work does not touch the refund path |
| **#134** WCAG manual screen-reader pass | **Deferred** | Needs a `role:ux-ui-designer` pass with real assistive tech; T15 ships no UI change |
| **#167** Stripe webhook receiver | **Deferred** | Would *remove* the `ConfirmOnlinePayment` authorization question rather than answer it — a genuine improvement, but it is a payments-architecture ticket competing directly with T15.5 for the same reviewer attention in the same context |

**Recorded disagreement on #124 — PO vs. PE.** PO's position: #124 was opened at
T12's Ceremony 1 and has now been passed over by T13, T14 and T15. It is a real
user-visible correctness bug (a cancelled Game silently keeps its courts
reserved and its registrations live), and "blocked on D1" is doing more work
than it should — the **Registrations** half has no D1 dependency at all and
could ship alone. PE's position: splitting an issue mid-ceremony to harvest its
easy half is how partial fixes become permanent furniture (which is QA's
standing objection on #147/#149), and a cascade that cancels half of what it
promises is worse than one that has not shipped. **Not resolved.** Recorded with
a scoring condition so it does not simply recur:

> **Scoring condition for T16's Ceremony 1.** If D1 is answered before T16 plans,
> #124 is taken whole and this disagreement is moot. If D1 is **still**
> unanswered, PO is right that the blockage is no longer the reason — and T16
> must either take the Registrations half explicitly as a partial fix, or
> re-rank #124 and record that it is not a priority. A fifth silent deferral is
> the finding.

## §A12 — Dependency-completeness check

The check T13's retro added and T13/T14 ran: for every ticket, does every
context it touches already have the members, ports and read paths it needs — or
does some *other* ticket have to supply them? Run against the code, not against
the tickets' own descriptions.

**The dual coverage question is deliberately absent** (§A7). `make
gate-coverage` answers it mechanically in `ci-checks`.

**GAP A — found, and it is T14's §A13 GAP A recurring exactly.** T15.4 and
T15.5 both need to *read* the Competition-Admin set, but T15.3 is a write-side
store ticket with no consumer of its own read. T14 hit this precisely: T14.4 had
to ship `ListGameAdmins` for T14.5's benefit, and its port doc comment records
the reasoning at length. **Resolution: T15.3's instructions require both the
port-level `ListCompetitionAdmins` and an exported `app.Service.ListCompetitionAdmins`,
even though T15.3 itself consumes neither.** Ranked as a requirement, not a
suggestion, because a missing read is the one gap that forces a second ticket to
widen a merged interface.

**GAP B — checked and *not* a gap, which is worth recording.** T15.5 must read
Social Play's admin set from inside Payments. `internal/socialplay/app/service.go:1009`
already exports `ListGameAdmins(ctx, gameID)`. **T15.5 therefore needs no edit
to `internal/socialplay` at all** — it consumes an existing exported method
through a new Payments-side port, exactly as `internal/socialplay/adapter/booking`
consumes Booking. This is T14.4's port-design decision paying off one sprint
later in a context nobody was thinking about, and it is why T15.5 is 8 points
rather than 13.

**GAP C — structural, owned by T15.5 alone.** `internal/payments/port/` holds
only `repository.go`, `payment_processor.go` and `idgenerator.go`. The existing
`internal/payments/adapter/{socialplay,competitions}` push **out**; T15.5 needs
adapters that read **in**. No other ticket supplies them, and T15.5's estimate
includes building them.

**GAP D — a verification gap, not a code gap, and it has hidden #185 twice.**
T15.6's defect lives in a Postgres foreign key. An in-memory fake accepts the
unknown court id and passes whether or not the fix exists — the fixture
infidelity `docs/LESSONS.md`' T9 entry names. **T15.6's instructions require the
ticket to state which of the two verification shapes it used** (real infra via
the `//go:build integration` testcontainers path, or a fake that models the FK
and says so), because a green suite over a fake would prove nothing here.

**Checked and clear:** T15.7 touches `internal/platform/auth/rs256` and
`cmd/server`'s configuration only — no bounded context, no proto, no migration,
and the `KeySource` seam it drops behind already exists (ADR-0013 §3). T15.1 and
T15.2 touch documentation only.

## §A13 — Wave-1.5 checkpoint: the condition is checked, and does not fire

The checkpoint (T13's A14, re-checked in T14's A10) applies when a new
cross-cutting decision has **three or more** first-time in-sprint consumers.

**T15.3's port and store have exactly two in-sprint consumers — T15.4 and
T15.5.** The condition does not fire, so **no Wave-1.5 checkpoint is applied**,
and Waves 2 and 3 dispatch on ordinary wave sequencing.

Recorded so the absence is legible as a *checked* absence. And, exactly as T14's
retro says of T14: **T15 therefore does not count toward "several sprints
passing with every checkpoint merging first-loop"**, so PdE's cost objection
(T13 finding 3) is carried forward untouched and unscored rather than quietly
eroded by two sprints in which the mechanism was never exercised.

## §A14 — Shared-file and migration pre-assignment

Pre-assigned so no two tickets collide, per the practice T13 introduced and T14
confirmed held with no conflict across three tickets on one file.

| Artifact | Owner | Notes |
|---|---|---|
| `db/migrations/0021_competitions_competition_admins.sql` | **T15.3 only** | `0021` is the next free number (verified: `0020` is the last). Must carry the ADR-0014 §5a citation and a #164 pointer, as `0020` does |
| `docs/adr/0016-*.md` | **T15.2 only** | `0016` is the next free ADR number (verified) |
| `docs/process/sprint-process.md` | **T15.1 only** | No other ticket amends process |
| `proto/pickleball/competitions/v1/competitions.proto` | **T15.3 only** | T15.4 changes authorization, not the contract |
| `proto/pickleball/payments/v1/payments.proto` | **T15.5 only** | Deprecating the two admin-list fields |
| `internal/competitions/app/service.go` | **T15.3 (Wave 1) → T15.4 (Wave 2)** | Sequenced by wave, never concurrent. T15.5 must **not** touch it — GAP B is why it does not need to |
| `internal/competitions/adapter/grpcapi/{handler,authenticated}.go` | **T15.3 → T15.4** | Same sequencing |
| `internal/payments/**` | **T15.5 only** | |
| `internal/booking/adapter/postgres/repository.go`, `internal/booking/domain/errors.go` | **T15.6 only** | |
| `internal/platform/auth/rs256/**`, `cmd/server` config | **T15.7 only** | |
| `Makefile` | **unassigned; no ticket is expected to need it** | If T15.7 adds a config knob it changes `docker-compose`/env, not a gate. A ticket that finds it needs a `Makefile` edit flags it in its PR rather than assuming |
| **`HANDOFF.md`** | **this ceremony only** | The A14 rule stands: an implementer that finds a stale line in it **flags it for Ceremony 1 and does not edit it**. T14 proves the other half of that rule also needs care — the reviewer who promised to fix the "three options" line did not, and it survived a whole sprint (§A1) |

## §A15 — "Zero issues opened" is not read as a quality signal (recommendation 10)

T14 opened zero issues and the open count fell 19 → 13. The retro's warning is
that **exactly one ticket (T14.8) demonstrated a *checked* zero**, with an
exhaustive documented sibling sweep; for the rest, "nothing disclosed" and
"nothing found" are indistinguishable from outside.

**This ceremony's reading of that number, stated before it is used:** T14's
falling count is partly a real improvement and partly unaudited. This ceremony's
own §A0 is evidence for the cautious reading — the sweep was clean on closures
and found **three missing partial-fix sentences and one entirely untracked
residual (#185)** on the same list. A count that falls while a residual goes
unrecorded is not straightforwardly progress.

**Threaded into every T15 ticket as a standing instruction**, in T14.8's proven
form rather than as an exhortation: *every ticket performs the sibling sweep its
subject implies and reports the result **either way**, with the enumeration
shown — "no other instance, so no issue opened" is an acceptable result only
when the categories searched are listed.* T14.8's PR body is the template.

**Consequence accepted:** T15 may well open more issues than T14 did, and that
is not a regression.

---

# Ceremony 2 — Sprint planning

## Sprint goal

> **An admin is a fact the system stores, everywhere it is used — not just where
> T14 got to.** Competitions gets the Competition-Admin store Social Play got in
> T14, its roster read widens from Host-only to Host-or-assigned-admin, and
> Payments stops taking the word of the caller about who the admins are, reading
> both stores instead — so the authorization rule CLAUDE.md locks in ("per-game
> **Game Admins** can record offline payments") is implementable
> trustworthily for the first time. Alongside it: a caller who names a court
> that does not exist gets a client error instead of a 500; this backend can
> verify a token from a live identity provider for the first time; and the two
> process contradictions T14 left standing — *when* an issue actually gets
> closed, and *whether a reviewer may write code* — are resolved into text, the
> first by amendment and the second by **escalation to the user**, because a
> rulebook the constrained party rewrites for itself is not a rulebook.

**What this sprint does not claim** (the half PM insists on, and the half T14's
retro found checkable):

- **`CancelBooking` still has no authorization check.** D1 is unanswered for a
  third sprint (§A9); anyone holding a booking id can still cancel it. No T15
  ticket touches it and none may guess the answer.
- **ADR-0012's Q1/Q2 remain blocked** — no `PlayerRating`, no `Gender`, no Level
  formula, and T15 adds none.
- **#149 is not closed.** Payments' `booking_host_id`, `game_host_id` and
  `entrant_player_id` remain caller-supplied; only the two admin lists are
  resolved from stores.
- **#164's backfill is not done, and its scope grows by one table** (T15.3's).
- **"Reaches CI" is still unproven.** `Jenkinsfile` calls `make ci-checks`; there
  is still no Jenkins job, webhook or branch protection, and no session here can
  create one. Unchanged since SCRUM-6.
- **A registrant/entrant still cannot read the roster**, deliberately — that is a
  product question T14.5 declined to smuggle into an authorization change and
  T15.4 inherits the boundary unchanged.
- **#124, #125, #126, #130, #134, #145, #167 are untouched**, each with a reason
  recorded in §A11.

## Tickets — 7 items, 34 points

### T15.1 — Amend `sprint-process.md`: the closure sweep's third state, and worktree recovery as a named practice

- **Story:** As a session picking this project up mid-stream, I want the process
  document to describe what sprints actually do — including the failure modes
  measured twice — so that I follow a rule that has survived contact rather than
  one that has scored zero for two sprints running.
- **Points:** 5 · **Role:** `role:principal-engineer` · **Type:** `type:chore`
- **Description:** T14's retro produced four process changes that belong in
  text. This ticket is their single owner, following T14.6's precedent of
  amending `sprint-process.md` via its own reviewed ticket rather than inside a
  planning PR. It deliberately does **not** touch `CLAUDE.md` — see instruction 3.

**Instructions**

1. **Name the third state, and change the per-PR step's shape rather than
   repeating it** (retro recommendation 1, §A4). Add to "The merged-fix issue
   sweep" a third, explicitly-named moment: *the merging session sweeps its own
   work at sprint end*. Classify it **acceptable-but-not-sufficient**, and state
   the consequence in text: when the merger sweeps its own work, the *"party
   other than the merger"* property is **not** obtained, and the next Ceremony
   1's run is **not** discharged — it re-runs the arithmetic anyway. Cite T15's
   §A0 as the first run of that rule and what it found (three unwritten
   partial-fix sentences, one untracked residual).
2. **Resolve the recorded PE/PO disagreement, in the direction the measurements
   point.** Rewrite DoD step 5 so the per-PR close is an **optional early
   close**, not *"the moment the close happens"*, and the named sprint-level
   sweep is primary. Update the "Recorded disagreement" block and its scoring
   condition to match — the old condition enumerated two branches and T14 landed
   in neither, which is the defect being fixed. **Do not add a third exhortation
   to the per-PR step**; the retro forbids it in terms.
3. **Adopt worktree-recovery-after-session-limit, minus its permission
   clause** (recommendation 2, §A5). Write clauses (a) check the worktree for
   unpushed commits *and* uncommitted work before re-dispatching, (b) a
   mandatory first-line provenance note (use #181/#182's wording as the
   template), (c) the safeguard — a recovered PR is reviewed by a **different**
   party; where none can be dispatched, the recovering session **says so
   plainly** and the retro independently re-derives that PR's headline claim,
   and (d) delete the sentence *"I am not merging this myself either"* from any
   PR the author intends to merge. **The question of whether the
   reviewing/merging session may author the code at all is out of scope here and
   belongs to ADR-0016 (T15.2)** — say so explicitly in the text, with a
   cross-reference, so a reader does not mistake this section for a rule-9
   carve-out. If you believe this split is wrong, **report it as a review
   finding; do not resolve it by widening this ticket.**
4. **Add the issue-state clause to the review enumeration** (recommendation 4).
   The review already enumerates issues opened, issues closed, and label
   conformance. Add: **each named issue's current state, read from the API, not
   from memory.** State the cost honestly — one field on a call already being
   made — and cite the failure it would have caught (PR #178 and its review both
   calling #97 "still-open" when it had been closed two days earlier and was
   never about the cited gap).
5. **Execute the scheduled removal** (recommendation 7, §A7). Mark the "dual
   coverage question" row **Removed at T15's Ceremony 1**, with the date and the
   evidence (T14.1 merged at 13:23:26Z; `make gate-coverage` green at 41
   packages; wired at `Makefile:326`). **Leave the row in the table as history
   rather than deleting it** — the table's value is that a reader can see a
   temporary practice was retired on schedule, which is unreadable if retired
   rows vanish.
6. **Non-functional:** documentation only. No Go, proto, SQL or `Makefile`
   change. Verify `make test-domain` and `make gate-coverage` are still green
   (they should be untouched — say so). State in the PR that `HANDOFF.md` was
   **not** touched and why (§A14).
7. **Sibling sweep, reported either way** (§A15): check whether any *other*
   process rule in `sprint-process.md` is stated in a form that scored zero
   across T13 and T14, and either name it or state the enumeration you performed
   and found nothing.

### T15.2 — ADR-0016: may a reviewing session author code? Escalate to the user as DECISION D2

- **Story:** As the owner of this project, I want to decide myself whether the
  agent that reviews and merges a PR may also write code on it, so that a
  safeguard I put in `CLAUDE.md` after two incidents is not relaxed by the
  agents it constrains.
- **Points:** 3 · **Role:** `role:principal-engineer` · **Type:** `type:chore`
- **Description:** Three of nine T14 PRs contain code the reviewing party wrote,
  and two PR bodies asserted a safeguard the merge record contradicts. CLAUDE.md
  rule 9 forbids it; T14's practice did it and called it *"this session's
  established practice."* It is currently both at once. §A6 records why this
  ceremony escalates rather than decides.

**Instructions**

1. **Write `docs/adr/0016-*.md`** following ADR-0015's structure exactly — it is
   the model escalation this project has, and the retro says so. Status:
   **Escalated — awaiting decision**, explicitly *not* Accepted.
2. **State the question in one sentence a non-engineer can answer**, as
   **DECISION D2**. Suggested form, to be sharpened not weakened: *"When the
   same session both reviews and merges a pull request, may it also write code
   on that pull request — and if so, under exactly what limits?"*
3. **Record the three observed instances with their measurements**, re-derived
   from the API rather than copied from the retro: PR #181 (9s open→review, 14s
   open→merge), PR #182 (8s, 13s) against a sprint median of 92 seconds, and PR
   #179's review-authored fix on the branch under review. Include the fact that
   #181 and #182 both state *"per CLAUDE.md rule 9, I am not merging this myself
   either"* and both were merged by that session seconds later. **Verify each
   number yourself; do not carry them forward on trust.**
4. **Lay out at least three options with costs, and pick none:**
   (a) **strict enforcement** — a reviewer that finds a gap requests changes; the
   implementer fixes it, or the ticket takes a second loop. Cost: a mechanical
   one-line fix caught at test-merge costs a full dispatch cycle.
   (b) **a bounded carve-out** — permitted when the fix is *mechanical,
   compiler- or test-caught, single-file, disclosed in the review, and itself
   re-verified*. This is PE's position from §A6 and must be written as a
   **fully-specified, directly-approvable** option, not a sketch. Cost: it is an
   exception to a rule whose own text says nothing is low-risk enough to except.
   (c) **a recovery-only carve-out** — authorship permitted solely to recover an
   interrupted session's unpublished work (T15.1 clause (b)), never for a fix on
   a branch under review. Cost: leaves PR #179's case unresolved.
5. **State the interim rule, and make clear it is not a decision.** Until D2 is
   answered, rule 9 as written governs: a reviewer that finds a gap **requests
   changes**. Nobody may read the open escalation as a suspension of the rule.
6. **Carry a restriction list**, as ADR-0015 does: no future PR may treat
   "documented in ADR-0016" as permission, and no PR may amend `CLAUDE.md`
   rule 9 until D2 is answered.
7. **Tie the trigger to the answer arriving, not to a sprint boundary** — the
   sprint immediately following the user's answer implements it, in
   `CLAUDE.md` and/or `sprint-process.md` as the answer directs.
8. **Non-functional:** documentation only. **Do not edit `CLAUDE.md`.** If the
   review disagrees with escalating, that is a review finding on this plan's
   §A6, not a licence to decide in the ADR.

### T15.3 — A durable Competition-Admin store for Competitions

- **Story:** As a Competition Host, I want to appoint Competition Admins that
  the system records, so that authorization rules can include them without any
  caller being able to name themselves one.
- **Points:** 5 · **Role:** `role:product-engineer` · **Type:** `type:story`
- **Description:** The mirror of T14.4, in the context T14 did not reach. Verified
  this ceremony: no store, no domain type, no port, no RPC, and **nothing
  blocking** — `grep -rli "competition_admin\|CompetitionAdmin" db/migrations
  internal/competitions proto` matches only the payments proto's caller-supplied
  list. **Partial fix for #168.** It costs materially less than T14.4's 8 points
  because T14.4 is a complete worked pattern to copy, which is exactly the
  premise T14's retro verified and which A16's scoring condition tests.

**Instructions**

1. **Copy T14.4's shape deliberately; do not redesign it.** `db/migrations/0021_competitions_competition_admins.sql`
   mirroring `0020_socialplay_game_admins.sql`: composite `PRIMARY KEY
   (competition_id, user_id)` as the authoritative uniqueness guard (rule 4),
   actor columns (`user_id`, `assigned_by`) as **`text`** per ADR-0014 §5a — **do
   not invent a third convention**, which is what #168 explicitly warns against —
   and a migration comment citing §5a and **#164** so a future backfill finds
   this table. Add the reverse `user_id` index, as `0020` does.
2. **Domain rules, Host-only, in `internal/competitions/domain`:** assign and
   revoke, with **an admin unable to appoint an admin** (#168's explicit demand —
   if this rule is absent the Host-only distinction is worthless), a Host unable
   to be their own admin, and reject-don't-guess on duplicates. Pure per rule 2:
   no port import.
3. **Port + Postgres adapter**, translating the composite-PK unique violation
   into the duplicate sentinel per rule 5, mirroring `game_admins`.
4. **Ship the read side, even though this ticket consumes it** — **§A12's GAP A,
   and this is a requirement rather than a nicety.** The port must carry
   `ListCompetitionAdmins(ctx, competitionID)`, and `app.Service` must export
   `ListCompetitionAdmins`. T15.4 needs the first and **T15.5 needs the second
   from outside the context**; Social Play's equivalent
   (`internal/socialplay/app/service.go:1009`) is what makes T15.5 cheap, and
   omitting it here would force a merged interface to be widened by a later
   ticket. Read `internal/socialplay/port/game_admin_repository.go`'s doc comment
   before choosing list-vs-boolean: it argues the list case in full and that
   reasoning applies unchanged.
5. **Assign/Revoke RPCs, in `AuthenticatedMethods()`** — confirm by reading
   `internal/competitions/adapter/grpcapi/authenticated.go` directly rather than
   assuming, as T14.4's PR did.
6. **Do not** wire the store into `ListEntriesForCompetition` (T15.4) or into
   Payments (T15.5). Say so in the PR, and say that #147 and #168 stay open.
7. **Non-functional:** TDD-first per rule 1; mutation-check the "an admin cannot
   appoint an admin" rule (T14.4's `TestAssignGameAdmin_AnAdminCannotAppointAnAdmin`
   is the template) and report the named failures. `make gate-coverage` must stay
   green with no new exclusion and no widened glob — if it fails, widen a `go
   test` pattern in a target reachable from `ci-checks`, never add an exclusion.
8. **Sibling sweep, reported either way** (§A15): enumerate any other
   entitlement in Competitions currently expressed as a caller-supplied list, and
   state the categories searched even if the answer is none.

### T15.4 — Resolve Competition Admins from the store; widen `ListEntriesForCompetition`

- **Story:** As an assigned Competition Admin, I want to read my competition's
  entry roster, so that I can do the job I was appointed for without the Host
  having to do it for me — and without anyone who is not an admin being able to.
- **Points:** 5 · **Role:** `role:product-engineer` · **Type:** `type:story`
- **Description:** The mirror of T14.5. T13.6 shipped this read Host-only and
  disclosed exactly why: the only expression of "assigned admin" was a
  caller-supplied list, so honouring it would have admitted anyone willing to
  name themselves. T15.3 makes the entitled set a server fact. **Closes #147.**
  **Depends on T15.3 — Wave 2.**

**Instructions**

1. **Resolve in `app`, enforce in `domain`, and make the parameter type the
   enforcement.** Follow T14.5's deviation, which its reviewer scored on its
   merits: the domain rule takes the **resolved `[]CompetitionAdmin`**, not a
   `[]string` — a request DTO cannot construct that type, so no future call site
   can hand it forged input however carelessly written. Record in the doc comment
   that the parameter type is load-bearing, as T14.5 did.
2. **Widen `ListEntriesForCompetition` to Host-or-assigned-admin**, resolving
   **per call and never cached** (a revoked admin loses the read on their next
   call) and **scoped to this competition** (an admin of a different competition
   is refused — enforced by the query, not by a droppable filter).
3. **No fast path for the Host.** T14.5 refused one and the reasoning holds:
   splitting the rule across the app layer to save one indexed read is the wrong
   trade against rule 2.
4. **Preserve two behaviours deliberately, with tests:** an unknown or malformed
   `competition_id` keeps its current answer (do not let this ticket change it
   silently), and an **entrant** of the competition is still refused — that is a
   product question, and answering it here would smuggle a product decision into
   an authorization change. T14.5 drew the same line for registrants.
5. **Close #147 explicitly after merge**, per DoD step 5, with a comment naming
   this PR — and state in the review that #168 stays open pending T15.5, with
   **#168's current state read from the API** (T15.1's new clause; use it even if
   T15.1 has not merged yet).
6. **Non-functional:** TDD-first. Mutation-check by reverting to Host-only and
   reporting the count and names of the failures (T14.5's mutation table is the
   template). Prove the forgery case directly: a caller with **no stored row**
   who names themselves must be refused.
7. **Sibling sweep, reported either way** (§A15).

### T15.5 — Payments resolves Game and Competition Admins from the stores, not from the caller

- **Story:** As a facility operator, I want offline payments to be recordable
  only by the people actually entitled to record them, so that "per-game Game
  Admins can record offline payments" is a rule the system enforces rather than
  a sentence the caller can satisfy by naming themselves.
- **Points:** 8 · **Role:** `role:principal-engineer` · **Type:** `type:story`
- **Description:** The last constraint #168 names. `RecordOfflinePayment` still
  honours caller-supplied `assigned_game_admin_user_ids` and
  `assigned_competition_admin_user_ids` (`payments.proto:115,118`). Both stores
  now exist. **Closes #168; partial fix for #149** — the three host/entrant
  ownership facts stay open, with the reason written on the issue. **Depends on
  T15.3 — Wave 3.**

**Instructions**

1. **Build read-side ports in `internal/payments/port`** — §A12's GAP C.
   `internal/payments/port/` currently holds only `repository.go`,
   `payment_processor.go` and `idgenerator.go`; the existing
   `internal/payments/adapter/{socialplay,competitions}` push **out**. Add the
   read direction, mirroring their shape.
2. **Consume the *app-level* reads that already exist. Do not edit
   `internal/socialplay` or `internal/competitions`** — §A12's GAP B, verified:
   `socialplay/app.Service.ListGameAdmins` is exported at `service.go:1009`, and
   T15.3 exports the Competitions equivalent. If you find you need to change
   either context, **stop and report it as a finding** — it means GAP B's
   verification was wrong, and that is worth more than a workaround.
3. **Translate errors at the boundary per rule 5**, `%s` not `%w`, exactly as
   `internal/socialplay/adapter/booking/reservation.go:63` does, so no Payments
   caller can `errors.Is` against another context's sentinel.
4. **Delete the app-level input fields rather than leaving them unread.**
   `assigned_game_admin_user_ids` / `assigned_competition_admin_user_ids` stay on
   the wire marked `[deprecated = true]` and ignored (client compatibility;
   #168's own text anticipates this), but the corresponding fields on the app
   input struct are **deleted**, so a future edit that re-plumbed the wire list
   would **not compile** instead of silently restoring a forgeable check. T14.5
   established exactly this and it is the load-bearing half.
5. **State plainly, in the PR and in the proto comment, what is *not* fixed:**
   `booking_host_id`, `game_host_id` and `entrant_player_id` remain
   caller-supplied. #149 stays open for them. Do not let "#168 closed" read as
   "Payments now verifies ownership."
6. **Close #168 after merge** per DoD step 5, with a comment naming all three
   PRs (T15.3, T15.4, T15.5) that together resolved its three constraints. State
   in the review that **#149 and #147's** current states were read from the API.
7. **Non-functional:** TDD-first. The headline mutation check: a caller naming
   **themselves** in the deprecated wire field must be refused, and the *same*
   caller must succeed once genuinely assigned and be refused again once revoked
   — the positive control matters as much as the negative. Report the failures by
   name. Watch the cross-context test shape: a fake that returns whatever it was
   told proves nothing about the real seam (T14.8's note on this is the
   reference).
8. **Sibling sweep, reported either way** (§A15).

### T15.6 — A well-formed but unknown `court_id` is a client error, not a 500

- **Story:** As a client developer, I want naming a court that does not exist to
  return a client error, so that I can tell my own bug from a server fault —
  which is the same thing #156 asked for and got only half of.
- **Points:** 3 · **Role:** `role:product-engineer` · **Type:** `type:bug`
- **Description:** **Closes #185**, opened by this ceremony. T14.8 fixed the
  *malformed* half and disclosed this residual, misattributing it to #97 (closed,
  and about a different gap). Verified this ceremony: `CreateBooking`'s guard is
  shape-only (`service.go:252`), `bookings.court_id` has an FK to `courts(id)`
  (`0001_init.sql:23`), and the Postgres adapter has **no `23503` mapping** — so
  the FK violation surfaces raw, is stripped at the context boundary, and lands
  in `default: codes.Internal`.

**Instructions**

1. **Re-verify the defect before fixing it**, both directly on
   `booking.CreateBooking` and through `ScheduleGame`/`ScheduleCompetition`. Do
   not carry the paragraph above forward on trust.
2. **Translate in the adapter, per rule 5.** Map `23503` on `bookings.court_id`
   to a domain sentinel, exactly as `23P01` → `domain.ErrCourtDoubleBooked` is
   mapped today. An app-level existence pre-check is **not** an equivalent
   substitute — it adds a read and still races the FK — though it may be added
   *in addition* if you argue for it.
3. **Answer #185's second question explicitly, in writing:** reuse
   `bookingdomain.ErrInvalidCourtReference` or add a new sentinel? Its doc
   comment (`internal/booking/domain/errors.go:20`) records that its `Internal`
   mapping was chosen deliberately to match this FK path, so either that comment
   and mapping change or a new sentinel is added. Answer it the way T14.8
   answered #156's two open questions — with the reasoning, not just the choice.
4. **Update every affected `toStatus`** across booking, socialplay and
   competitions. **T14.7's exhaustiveness guard exists for exactly this** and
   caught a missing mapping row at test-merge last sprint — expect it to fire if
   you miss one, and say in the PR whether it did.
5. **Confirm the enumeration-oracle question rather than assuming it.** Unlike
   T10.7's malformed-Game-ID convention, "no such court" plausibly reveals
   nothing a caller cannot already learn — but check it against the real read
   paths and record the check.
6. **Non-functional, and this is the ticket's hardest part — §A12's GAP D.**
   An in-memory fake accepts the unknown court id and passes whether or not the
   fix exists. **State which verification shape you used**: the `//go:build
   integration` testcontainers path against real Postgres, or a fake that models
   the FK and says so in its own comment. A green suite over a naive fake proves
   nothing here, and this fixture infidelity is what hid the defect twice.
   `make vet-integration` must stay green if you add integration-tagged files.
7. **Sibling sweep, reported either way** (§A15): are there other FK-backed
   references (`facility_id`, `game_id`, `competition_id`) with the same
   unmapped-`23503` shape? Enumerate them and report, even if the answer is none.

### T15.7 — A remote JWKS `KeySource`, so a live identity provider's tokens can be verified

- **Story:** As an operator deploying this backend against a real identity
  provider, I want it to fetch and refresh the provider's signing keys, so that
  a key rotation does not silently invalidate every token with no warning and no
  redeploy.
- **Points:** 5 · **Role:** `role:principal-engineer` · **Type:** `type:story`
- **Description:** **Closes #137.** `internal/platform/auth/rs256` ships exactly
  one `KeySource` — `StaticKeys`, buildable from a JWKS document or a file. T14.9
  made that usable for local dev with a committed fixture; it is still unusable
  against any real provider. The seam was split out for this purpose in ADR-0013
  §3, so this drops in behind an existing interface without touching the
  verification logic the platform's security rests on.

**Instructions**

1. **`rs256.RemoteKeys` implementing `KeySource`** over an HTTP JWKS endpoint.
2. **Cache with a TTL, plus a bounded, rate-limited refresh on unknown `kid`** —
   the standard rotation path. Both bounds are security requirements, not
   niceties: an unbounded unknown-`kid` refresh lets an unauthenticated caller
   drive requests to the provider at will.
3. **Reuse `NewStaticKeysFromJWKS` for parsing.** It already treats the document
   as untrusted input (rejects non-RSA `kty`, non-`sig` `use`, undersized moduli,
   duplicate `kid`s). **Do not add a second, laxer parser** — that is the same
   two-shapes failure this project keeps naming.
4. **Config: `AUTH_JWKS_URL` as an alternative to `AUTH_JWKS_FILE`.** Preserve
   T13.5's property exactly — **the server still refuses to start when neither is
   set** — and prove it by mutation rather than asserting it;
   `cmd/server/main_test.go` holds the existing startup-refusal tests and is now
   gated by `make test-cmd`. Say explicitly what happens when **both** are set:
   decide, document, and test it.
5. **Test against `httptest` with locally-minted keys** — no external tenant
   required, which is what makes this ticket buildable here at all. Cover: happy
   path; rotation (unknown `kid` triggers exactly one refresh, not one per
   request); a provider returning garbage; a provider that is unreachable; and
   TTL expiry.
6. **Non-functional:** TDD-first. Do not weaken `StaticKeys` or the dev fixture
   path — `make up` and `make dev-token` must still work unchanged, and the PR
   should say they were checked. `make gate-coverage` green with no new
   exclusion.
7. **Sibling sweep, reported either way** (§A15): does any other part of the
   auth spine assume a static, never-rotating key set? Enumerate and report.

## Waves

Sequenced by real dependency, with §A14's file assignments making concurrency
within a wave safe.

**Wave 1 — no in-sprint dependencies (5 tickets, 21 points)**
`T15.1` (process) · `T15.2` (ADR) · `T15.3` (Competition-Admin store) ·
`T15.6` (unknown court id) · `T15.7` (remote JWKS)

**Wave 2 — depends on T15.3 (1 ticket, 5 points)**
`T15.4` (widen `ListEntriesForCompetition`) — needs the store's read side.

**Wave 3 — depends on T15.3 (1 ticket, 8 points)**
`T15.5` (Payments reads both stores). Technically it needs only T15.3, but it is
placed after Wave 2 rather than beside it so that `internal/competitions/app/service.go`
is never held by two tickets at once (§A14). If Wave 2 finishes early, T15.5 may
dispatch as soon as **T15.4 has merged**, not merely when T15.3 has.

**No Wave-1.5 checkpoint** — the condition was checked and does not fire
(T15.3 has two first-time in-sprint consumers, not three). §A13 records why the
absence is a checked absence, and why T15 does not count toward the
"several sprints" bar PdE's cost objection is waiting on.

## Recorded disagreements (Ceremony 2 rule 3 — not smoothed over)

1. **PE vs. PO — should the reviewer-authorship question be decided here or
   escalated?** Full text in §A6. Resolved in favour of PO **on authority**,
   with PE's substance preserved as a directly-approvable option (b) in
   ADR-0016. PE's cost objection — a sprint of ambiguity over a question the
   team could answer in a paragraph — stands unrebutted and is recorded.
2. **QA vs. PdE — do repeated partial fixes become permanent furniture?**
   Carried forward live from `docs/process/t14-sprint-plan.md` §A16, and now
   applying to **#149** as well as #147. QA: #147 has been open three sprints
   with two partial fixes, #149 three sprints with none, and T15 adds a *third*
   partial fix to #149 rather than closing it. PdE: T15 closes #147 and #168
   outright, which is what A16's scoring condition asked of this sprint, and the
   #149 split is drawn at a real structural seam (two admin tables vs. three
   cross-context ownership resolutions) rather than at a convenient one.
   **A16's condition is met on its own terms** — T15 takes the Competitions half
   — so PdE's premise is upheld for #147/#168. **QA's objection transfers intact
   to #149** and is carried forward with a fresh scoring condition: *if T16 does
   not close #149's remaining three facts, QA's prediction is confirmed for that
   issue and it should be re-ranked or re-titled to the residual.*
3. **PO vs. PE — is #124's deferral still justified?** Full text in §A11, with a
   scoring condition binding T16's Ceremony 1.

## Sprint-level Definition of Done

All of `sprint-process.md`'s standing DoD, plus the three scorings T15 owes,
stated now so they are not improvised at the retro:

1. All 7 tickets merged per the per-ticket DoD; sprint goal met or explicitly
   descoped with reasoning recorded.
2. **The merged-fix issue sweep run and reported with its count** — by the retro
   (reporting, not blocking) and again by T16's Ceremony 1 (authoritative), and
   **T15.1's new third-state clause means a sprint-end self-sweep by the merging
   party does not discharge either.**
3. **Scoring owed at the retro:**
   - **(a)** Did T15.1's reshaped DoD step 5 change the closure *outcome*, and
     at which moment did the closes actually happen? The per-PR step has scored
     0/9 and 0/6; T15 is the first sprint where it is no longer described as
     primary. **Report the moment, not just the count.**
   - **(b)** A16's carried condition: T15 takes the Competitions half, so score
     whether #147 and #168 actually closed — and score disagreement 2's transfer
     to #149.
   - **(c)** Did any T15 PR contain code written by its reviewer? With ADR-0016
     open and the interim rule being rule 9 as written, the answer should be
     **none** — and the retro should verify it from the commit record rather
     than from the reviews' own accounts.
4. **Not scoreable by T15 and deliberately not pre-empted:** D1 and ADR-0012's
   Q1/Q2 remain the user's. If either is answered mid-sprint, the answer's own
   trigger takes over and T15's plan does not constrain it.
5. Retro in `docs/process/t15-retro.md`, indexed by a `## T15 sprint retro` stub
   in `docs/LESSONS.md`. `HANDOFF.md`/`CLAUDE.md` state updated — noting that
   **T16's Ceremony 1**, not the retro, corrects T15's Docs-index row.
