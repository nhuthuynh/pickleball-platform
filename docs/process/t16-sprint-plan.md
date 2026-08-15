# T16 Sprint Plan — Ceremonies 1 + 2

Backlog refinement (Ceremony 1) and sprint planning (Ceremony 2) per
`docs/process/sprint-process.md`, read in its **T16-amended** form (this
ceremony's own edits — the dependency-completeness check's second clause,
the mandatory close for "closes #N"-titled PRs, and the correction-not-just
-closure clause — all landed directly in this same PR, per the explicit
instruction that produced them, rather than spun into a ticket for later
execution; see §A3). Held against `docs/process/t15-retro.md` (PR #194,
merged 15:51:36Z), `HANDOFF.md` including its Docs index, `CLAUDE.md`, and
the live PR/issue state of `nhuthuynh/white-label` (GitHub-side name
`pickleball-platform`).

**Every factual claim below was re-derived against the repository during
this ceremony**, at the merged branch tip, rather than taken from the
retro's or T15's plan's prose — CLAUDE.md rule 10 applied to planning.
Where this ceremony repeats a claim it could not independently check, it
says so.

## §A0 — The merged-fix issue sweep, run as this ceremony's first act

Per `sprint-process.md`'s Definition of Done, T16's Ceremony 1 is the next
sweep's **authoritative** moment, and it does not defer to T15's retro
having already run one (*"a self-sweep does not discharge the next
Ceremony 1's run… it re-runs the arithmetic anyway, and should treat this
retro's closes the same way it would treat any other prior sweep's output:
verified, not trusted"*) — recommendation 5, executed here rather than
merely acknowledged.

**Step 1 — list the open issues.** `list_issues(state: OPEN)` at ceremony
start (before any action this ceremony takes) → **`totalCount: 11`**:
#124, #125, #126, #130, #134, #144, #145, #149, #164, #167, #168.

**Step 2 — reconcile arithmetically, before reading any issue
individually.** `open_at_end_of_T15's_own_Ceremony_1 − closed_during_T16 + opened_during_T16`,
tracing every issue-count change between the two ceremonies rather than
diffing only the two totals. T15's own Ceremony 1 (§A0) left the count at
**14** (13, then +1 for #185, with zero closes that sweep found). Three
things changed it during T15's execution and retro, none of them a T16
action: **#147** closed at review time (PR #192, T15.4) — `14 − 1 = 13`;
the retro then independently closed **#185** and **#137**, both fully
resolved in code but never closed on GitHub despite their own PRs' titles
— `13 − 2 = 11`. **This matches** the live `totalCount: 11` read at this
ceremony's start exactly, tracing every step rather than asserting the
endpoints agree.

**Step 3 — cross-reference merged T16 PRs against the open list.** As of
this ceremony's start, **no ticket PRs have merged this sprint** — T16 has
not begun execution. This step therefore has nothing to check yet and is
correctly empty, not skipped. **Sweep result: clean, by construction, and
stated as such rather than omitted.**

**This ceremony's own actions during Ceremony 1** (filing #195, correcting
and re-titling #125 — see §A4/§A5) change the count to **12** by the time
Ceremony 2 ranks the backlog: `11 − 0 + 1 (#195) = 12`. Recorded now so a
future reader checking this ceremony's own arithmetic has the number to
check it against: `list_issues(state: OPEN)` read again after §A4/§A5's
actions returns `totalCount: 12`, confirmed live.

## §A1 — Correcting T15's Docs-index row and Task-backlog narrative

Ceremony 1's designated first job. **Performed directly in `HANDOFF.md` by
this same PR** (not summarized here twice): the previous entry for T15
said "not yet written" / "not yet opened" for Retro/Reviews; both are now
filled in with PR merge order **verified against each PR's `merged_at`**
(fetched live, not assumed from numbering):

| PR | Ticket | `merged_at` |
|---|---|---|
| #186 | Ceremony 1/2 doc | 14:27:16Z |
| #187 | T15.2 | 14:34:10Z |
| #188 | T15.1 | 14:35:35Z |
| #189 | T15.7 | 14:50:51Z |
| #190 | T15.3 | 14:53:29Z |
| #191 | T15.6 | 14:58:39Z |
| #192 | T15.4 | 15:14:20Z |
| #193 | T15.5 | 15:36:26Z |
| #194 | retro doc | 15:51:36Z |

**For T15, merge order and numeric order agree** — verified rather than
assumed, per this project's standing convention.

**The Task-backlog narrative is replaced with the retro's own agreed
sentence**, quoted verbatim in substance from `docs/process/t15-retro.md`'s
"The sprint goal, scored" section, per `sprint-process.md` Ceremony 1 item 3
("the retro's form, not a stronger one"). It states plainly that **T15 did
not close #168** despite the sprint's headline goal naming it, names the
two silently-missed "closes #N" claims the retro's own sweep caught, and
records that D1/D2 remain unanswered. See `HANDOFF.md`'s T15 entry for the
full text — not reproduced a second time here.

**A new T16 row** is added in the same before-the-sprint form, Retro and
Reviews honestly marked "not yet written" / "not yet opened", for T17's
Ceremony 1 to correct in turn.

## §A2 — What this ceremony verified, and how

| Claim | How it was checked | Result |
|---|---|---|
| Open-issue count at ceremony start | live `list_issues(state: OPEN)` | `totalCount: 11` |
| #185 and #137 are genuinely closed, not just absent from a stale list | `issue_read(get)` on each | both `state: closed`, `state_reason: completed`, `closed_at: 2026-08-15T15:40:26Z` / `:34Z` |
| #147 is genuinely closed | `issue_read(get)` | `state: closed`, `closed_at: 2026-08-15T15:14:24Z` |
| #168 is genuinely still open | `issue_read(get)` | `state: open` — confirmed, not assumed from the retro's prose |
| The exact resolution-read gap T15.5 found still exists unfixed | `grep -n "^func (s \*Service)" internal/socialplay/app/service.go internal/competitions/app/service.go` | no `GetRegistrationByID`-shaped or `GetGame`-shaped method on Social Play's `app.Service`; no exported `GetEntryByID`-shaped method on Competitions' `app.Service` (only an *internal* call at `competitions/app/service.go:549`, inside `MarkCompetitionEntryPaymentStatus`) |
| The join-key fields those future methods would return already exist | read `internal/socialplay/domain/registration.go`, `internal/socialplay/domain/game.go`, `internal/competitions/domain/entry.go` | `Registration.GameID`, `Game.HostID`, `CompetitionEntry.CompetitionID`, `CompetitionEntry.PlayerID` all exist today as plain struct fields |
| The underlying repository reads those methods would wrap already exist (this is not a new capability, only a new export) | read `internal/socialplay/port/repository.go`, `internal/competitions/port/repository.go` | `RegistrationRepository.GetByID`, `GameRepository.GetByID`, `Repository.GetEntryByID` all already implemented (Postgres + fakes) |
| T15.5's admin-reader ports are real, tested, and still unwired | read `internal/payments/port/{game_admin_reader,competition_admin_reader}.go`, `internal/payments/adapter/{socialplay,competitions}/*_admin_reader.go`; `grep -n "GameAdminReader\|CompetitionAdminReader" internal/payments/app/service.go` | both ports and adapters exist and carry tests; zero references in `payments/app/service.go` — confirmed still not constructor-wired |
| `payments/app.ServiceOptions` already carries a live push-port into Competitions | read `internal/payments/app/service.go:61-98` | `CompetitionEntryUpdater competitionsport.CompetitionEntryPaymentUpdater` already a field, wired since T10.6 — the mechanism T16.4 needs to push `PaymentStatusRefunded` already exists |
| `RefundPayment`'s payable-type gate currently rejects `competition_entry` | read `internal/payments/app/service.go` around `authorizeOnlineCreation`/`RefundPayment`, `internal/payments/app/refund_test.go` | `TestRefundPayment_OutOfScopePayableTypesRejected` pins it, subtest literally named `"competition_entry is out of scope (issue #125)"` |
| **#125's own text is stale** — the Competitions `PayableType` it asks for already shipped | `grep -n "PayableTypeCompetitionEntry"` across `internal/payments`; read `internal/payments/domain/payment.go`'s doc comment | `PayableTypeCompetitionEntry` exists since **T10.6** (`payment.go:31-42`, "T10.6, closes #96"); #96 independently confirmed `closed`, `state_reason: completed`, `closed_at: 2026-08-13T14:45:52Z` |
| `Game.Cancel()` / `Competition.Cancel()` still do not cascade | read `internal/socialplay/domain/game.go:206-227`, `internal/competitions/app/service.go:587-609` | both confirmed status-flip-only, no registration/entry side effect in either context — the Competitions half is a **new finding this ceremony**, not previously tracked anywhere |
| `internal/socialplay/port.RegistrationRepository` / `internal/competitions/port.Repository` have no bulk-cancel method | read both files in full | confirmed — only single-row `Update`/`UpdatePaymentStatus`-shaped writers exist on either side |
| `make gate-coverage` still works on the unmodified tree | `make generate && make gate-coverage` | `OK — all 41 package(s) … executed by "ci-checks"` — unaffected by this ceremony's doc-only edits |
| D1 (#144) has received no answer | `issue_read(get_comments)` | exactly one comment, T14.3's original escalation |
| D2 (ADR-0016) has received no answer | read `docs/adr/0016-*.md`'s `## Status` | still "Escalated — awaiting decision (D2)"; no comment, no follow-up PR |
| Next free migration / ADR numbers | `ls db/migrations`, `ls docs/adr` | `0021` is the last migration, `0016` the last ADR → **`0022`** and **`0017`** are free (not consumed this ceremony — no new migration or ADR is dispatched) |

**Not re-verified by this ceremony, and named as such:** the full `make
test` / `ci-integration` Docker path (no Docker daemon reachable here); any
claim about what Jenkins would run (no Jenkins job exists); T15.6's own
Postgres-integration verification of the `court_id` fix (compiled here via
`make vet-integration`, not executed, matching every prior sprint's stated
gap).

---

# Ceremony 1 — Backlog refinement

## §A3 — T15 retro's seven recommendations, each given a disposition

None left unaddressed. "Executed here" means this ceremony performed it
directly (in `sprint-process.md`, in `HANDOFF.md`, or via a GitHub API
call); "ticketed" means a T16 ticket owns the code that implements it;
"escalated" means it leaves this team's authority.

| # | Recommendation | Disposition |
|---|---|---|
| 1 | Add the consumer-input clause to the dependency-completeness check | **Executed here** — landed directly in `sprint-process.md`'s new "The dependency-completeness check" subsection (§A4), and applied concretely to T16.2's own dependency check (§A9) |
| 2 | Distinguish "closes #N" from "partial fix for #N" in the closure DoD; make the close mandatory for the former | **Executed here** — landed directly in `sprint-process.md`'s DoD step 5 (§A4) |
| 3 | File T15.6's disclosed FK-race residual as an issue | **Executed here** — filed as **#195** (§A5). **Not folded into this sprint as an implementation ticket** — see §A5's reasoning and the recorded disagreement in Ceremony 2 |
| 4 | Add the correction-not-just-closure clause to the review checklist | **Executed here** — landed directly in `sprint-process.md`'s Label taxonomy section (§A4) |
| 5 | Continue treating the merged-fix sweep as authoritative regardless of prior ceremonies' results | **Executed here** (§A0) — independently re-derived the open count and re-verified #185/#137/#147's closes from the API rather than the retro's own table |
| 6 | Put D1 and D2 to the user again, as their own items, not buried in the plan | **Executed here** (§A8) |
| 7 | Any T16 ticket that reads across a context boundary must name the specific field or method supplying the join key | **Executed here (as a `sprint-process.md` amendment, folded into recommendation 1's same subsection) and applied** — T16.2's dependency check (§A9) names `Registration.GameID`, `Game.HostID`, `CompetitionEntry.CompetitionID`, `CompetitionEntry.PlayerID` explicitly, and the new methods that must expose them |

## §A4 — The three `sprint-process.md` amendments (recommendations 1, 2, 4, 7)

Landed directly in `docs/process/sprint-process.md` by this same PR, per
the explicit instruction that produced this ceremony rather than deferred
to a future ticket's execution (the one deliberate departure from T15's own
precedent, where T15.1 was a ticket executed later in the sprint — see the
header note). Summarized here; the amendments themselves are the source of
truth, not this summary:

1. **A new "The dependency-completeness check" subsection**, under
   Ceremony 1, formalizing a practice that has been run every sprint since
   T13 but never written into this document — each sprint plan re-explained
   it from scratch. Carries both the original question (does the producer
   capability exist?) and the new one T15.5 forced (does the consumer's own
   input actually contain the join key?), plus recommendation 7's
   generalization to any cross-context read, not only ID-shaped
   export/input pairs.
2. **DoD step 5 gains a conditional-optionality clause.** "Partial fix for
   #N" keeps the optional-early-close treatment T15.1 established
   unchanged. **"Closes #N" does not** — a review whose PR's title asserts
   a close either performs it or states in the review why it is not doing
   so at that moment. This is the direct fix for T15's own finding 2/3 (two
   "closes #N" PRs, zero closes at review time, caught only by the retro).
3. **The Label taxonomy section gains the correction clause.** When a
   review or PR body's own findings falsify an earlier claim on a
   still-open issue — including a claim an earlier ceremony itself wrote,
   as #149's mid-sprint prediction was — the correction is posted to the
   issue, not left only in the PR that found it. Direct fix for T15's
   finding 5.

**Non-functional, verified rather than assumed:** documentation only, no
Go/proto/SQL/`Makefile` change. `make test-domain` and `make gate-coverage`
re-run after the edit (§A2) and confirmed unaffected.

## §A5 — The FK-race residual: filed, not folded in (recommendation 3)

**Filed as #195**, per PR #191's (T15.6) own §7 sibling-sweep table —
verbatim, not re-derived, since PR #191 already did the enumeration work
(`grep -n REFERENCES db/migrations/*.sql` cross-checked against 15 `INSERT`
queries) and re-running it here would only reproduce the same nine rows.
Labelled `role:product-engineer`, `type:bug`, and all four `context:`
values it touches (`booking`, `socialplay`, `competitions`, `facilities`) —
conformant at creation, matching #185's own precedent.

**This ceremony's judgement: track it, do not take it this sprint.**
Argued, not asserted:

- **It is real, but not urgent.** Every one of the nine paths is guarded by
  an app-level existence read immediately before the insert; the failure
  mode requires a *concurrent delete of the parent row* landing in the
  narrow window between that read and the insert. None of the four
  contexts involved ships a bulk-delete or cascading-delete flow today
  (confirmed by inspection: `facilities`, `socialplay`, `competitions`, and
  `booking` have no delete RPC of any kind, only cancel/status
  transitions), so the practical exposure is low relative to #185's own
  window (a `CreateBooking` racing nothing, just a caller naming an id that
  was never real).
- **It is not small.** T15.6 (one FK, `bookings.court_id`, three contexts)
  landed as 25 files, +1255/−46. Nine FK paths across **four** contexts is
  a materially larger surface, even though every row is a mechanical
  repeat of the same template (translate the constraint, pick
  reuse-vs-new-sentinel, update each context's exhaustiveness table, prove
  it against real Postgres). Sizing it honestly puts it well past what
  fits alongside T16.2–T16.4 without displacing them.
- **It would compete for the same reviewer attention T16.2 already needs.**
  T16.2 (§A9) is this sprint's Principal-Engineer-scoped headline ticket,
  touching `internal/payments`, `internal/socialplay/app`, and
  `internal/competitions/app` — the identical context set three of the
  nine FK rows in #195 also touch. Running both in the same sprint against
  the same files raises exactly the shared-file collision risk
  `sprint-process.md`'s file pre-assignment practice exists to avoid,
  for a ticket whose own severity argument (above) does not demand it.

**Recorded disagreement — PE vs. PM, per Ceremony 2 rule 3.** PE's
position: #195 is the same defect class as #185, mechanical and
low-risk to fix once, and deferring a known security-adjacent gap two
sprints running (T15.6 disclosed it, T16 tracks-but-doesn't-fix it) risks
it going the way of #147/#149's "permanent furniture" pattern QA has
flagged twice already. PM's position: severity is genuinely low (no
bulk-delete flow exists to trigger it), the scope is real (nine paths,
four contexts, larger than T15.6's own delivered size), and this sprint's
actual unfinished business — #168, the sprint T15 promised and did not
deliver — deserves the full reviewer bandwidth rather than splitting it
with a defense-in-depth hardening pass. **Not resolved; ranked for T17
with a scoring condition**: if #195 is still untaken at T17's Ceremony 1,
that is the third sprint the same defect class has sat disclosed-but-open
(counting #185's own one-sprint gap before it was filed), and PE's
"permanent furniture" concern should be treated as confirmed rather than
re-argued from scratch.

## §A6 — #125 corrected: a stale issue found during backlog verification, not carried forward on trust

**Found while checking T16's candidate backlog against the code, not while
looking for it.** #125's title and body ask for a Competitions-shaped
`payments.PayableType` plus its port/adapter pair — work that shipped in
**T10.6** (PR #102), which closed **#96**, the original T9.6/T9.7-filed
issue for the identical gap. #125 was opened later, at **T12's** Ceremony
1, apparently without checking `internal/payments/domain/payment.go`
first. `PayableTypeCompetitionEntry`, the full port/adapter pair, and the
online-checkout wiring have all existed since T10 — Competitions has not
been cash-only since then.

**What "#125" now actually points at in the live codebase is narrower and
still real**: `RefundPayment`'s payable-type gate (T12.3) explicitly
excludes `competition_entry`, and cites this issue number three times
(the proto's own RPC doc comment, `service.go`'s exclusion comment, and
`refund_test.go`'s subtest name) — a case of the same number surviving
while its subject silently drifted, the identical shape as #97's
misattribution (T14 finding 5) and #149's stale prediction (T15 finding
5), just discovered by inspection this time rather than by a PR's own
review.

**Corrected, not closed**, per recommendation 4's own new clause (§A4):
a comment posted to #125 explaining the drift in full
([#125#issuecomment-5303043198](https://github.com/nhuthuynh/pickleball-platform/issues/125#issuecomment-5303043198)),
and the issue re-titled — per the #147 precedent — to what it actually
still tracks: *"RefundPayment rejects competition_entry payables (the
Competitions-PayableType half this issue originally asked for shipped in
T10.6/#96)"*. **Ranked and taken this sprint as T16.4** (§A9) — the
remaining gap is narrow, mechanical, and the push-mechanism it needs
(`CompetitionEntryUpdater`) already exists, wired since T10.6.

## §A7 — Confirming `Game.Cancel()`/`Competition.Cancel()` still do not cascade, and that Competitions has the identical gap

Re-verified against the tree (§A2): both `domain.Game.Cancel()` and
`app.Service.CancelCompetition` flip status and nothing else. #124 (open
since T5.1, carried seven sprints) only ever named the Social Play half.
**The Competitions half is a new finding, not previously tracked
anywhere** — `grep -rli "cascade" internal/competitions` returns nothing,
and no issue mentions `CancelCompetition`'s own version of the gap. Per the
board-of-record rule this would ordinarily need its own issue if left
untaken; instead it is folded into T16.3 (§A9) as a mirrored fix built the
same sprint as the Social Play half it duplicates, following this
project's own established precedent of shipping a Social Play mechanism
and its Competitions mirror together or in immediate succession (T14.4→T15.3,
T13.6→T15.4) rather than disclosing the mirror as a second gap and waiting.

## §A8 — DECISION D1 and DECISION D2 remain unanswered. Put to the user again, as their own items

**Re-verified this ceremony, not assumed** (§A2): #144 carries exactly one
comment, T14.3's original escalation. ADR-0016's own `## Status` field is
unchanged: "Escalated — awaiting decision (D2)." Neither has been touched
since T15 planned around them.

> **DECISION D1 (for the user / Product Owner) — third deferral.** When
> somebody books a court through the public flow *without an account* —
> which is how the flow works today — who should be allowed to cancel that
> booking later, and should booking without an account remain possible at
> all? `docs/adr/0015-booking-ownership-for-public-bookings.md` lays out
> four options with their costs and recommends none. No T16 ticket
> implements this, and per ADR-0015's restriction list no T16 PR may guess
> at it.

> **DECISION D2 (for the user) — first deferral, second sprint open.**
> When the same session both reviews and merges a pull request, may it
> also write the code on that pull request — and if so, under exactly what
> limits? `docs/adr/0016-reviewer-authored-code-on-a-reviewed-pull-request.md`
> lays out four options, including a fully-specified carve-out the user can
> approve directly by naming it, and recommends none. Until it is answered,
> the interim rule governs: a reviewer that finds a gap on a branch under
> review requests changes; it does not fix the branch itself. T16 obeys
> this without exception — see §A11's confirmation that no T16 execution
> has happened yet to check it against, and the standing instruction
> threaded into every T16 ticket regardless.

**Neither is implemented, decided, or guessed at by this ceremony.** Both
remain exactly as blocked as `sprint-process.md`'s own restriction lists
require.

## §A9 — The whole open backlog, ranked, with a disposition for each

All 12 open issues (11 carried from T15, plus #195 opened here). "Taken"
means a T16 ticket owns it; every deferral carries its reason, per the
board-of-record rule that a deferral without a written reason is a process
violation.

| Issue | Ranked | Disposition |
|---|---|---|
| **#168** Payments never actually reads either admin store | **Taken** | **T16.2** — the sprint's headline ticket, finishing what T15.5 built but could not wire |
| **#124** `Game.Cancel()`/`CancelCompetition` do not cascade | **Taken, in part** | **T16.3** — Registrations/Entries half only, resolving the carried T15-plan §A11 disagreement (below); court-Bookings half stays deferred, reason restated |
| **#125** `RefundPayment` rejects `competition_entry` | **Taken** | **T16.4** — corrected and re-titled this ceremony (§A6); narrow, mechanical, push-mechanism already exists |
| **#195** FK-race class, nine write paths | **Filed, not taken** | Ranked for T17; reasoning and recorded disagreement in §A5 |
| **#149** Payments' remaining caller-supplied fact (`booking_host_id`) | **Advanced, not taken as its own ticket** | T16.2 resolves `game_host_id`/`entrant_player_id`/both admin lists as a side effect of closing #168, leaving `booking_host_id` as #149's **only** remaining fact — blocked on D1 exactly as #149's own text already says (Booking has no owner concept to resolve it against). Comment posted to #149 recording the narrowing once T16.2 merges — not before, since nothing has shipped yet |
| **#144** `CancelBooking` has no authz | **Escalated** | D1 unanswered; §A8. Fourth sprint carrying this, recorded plainly rather than softened |
| **#164** ADR-0014 actor-column conformance | **Deferred, still blocked** | #137 (one of its two named blockers) closed in T15; **#145 (the other) does not** — a real IdP tenant still does not exist and cannot be provisioned from this environment. Re-checked, not assumed: `docs/adr/0013-*.md`'s "Known gaps" list is unchanged |
| **#145** pre-existing UUID rows vs. `Principal.Subject` | **Deferred** | Unreachable without a real IdP tenant, same as every prior sprint |
| **#126** real per-Game price field | **Deferred** | Needs Product Owner input on pricing shape before any code; unchanged |
| **#130** refunding a `no_show_fee` | **Deferred** | Its own text asks that the product behaviour be confirmed before shipping ("confirm first that reversing a no-show fee is the product behaviour actually wanted, rather than assuming it from symmetry") — unlike #125's mirror-image gap, this one carries a live product question, not just an engineering one, and is not folded into T16.4 for that reason |
| **#134** WCAG manual screen-reader pass | **Deferred** | Needs a `role:ux-ui-designer` pass with real assistive tech; T16 ships no UI change |
| **#167** Stripe webhook receiver | **Deferred** | A payments-architecture ticket competing for the same reviewer attention T16.2 already claims this sprint; unchanged reasoning from T15's plan |

**Recorded disagreement on #124, resolved rather than carried forward
again.** T15's plan (§A11) left a live PO/PE disagreement with a binding
scoring condition for T16: *"If D1 is answered before T16 plans, #124 is
taken whole… If D1 is still unanswered, PO is right that the blockage is
no longer the reason — T16 must either take the Registrations half
explicitly as a partial fix, or re-rank #124 and record that it is not a
priority. A fifth silent deferral is the finding."* D1 is still
unanswered (§A8), so the condition fires as its second branch. **PO's
position is adopted**: the Registrations half (and, found this ceremony,
its Competitions-Entries mirror, §A7) has no dependency on D1 at all and
ships as **T16.3**. **PE's original concern about the court-Bookings
half is preserved, not overridden**: it stays deferred, because it would
call `app.Service.CancelBooking`, whose signature ADR-0015's still-open D1
may yet change — building against a signature under active escalation
means either guessing D1 or shipping code this project already knows may
need rewriting. This is not a fifth silent deferral; it is the fourth,
now split into a taken half and an explicitly-reasoned deferred half,
which is the disagreement's own scoring condition being honoured rather
than dodged.

## §A10 — Dependency-completeness check, applied per `sprint-process.md`'s newly-amended shape

Both questions (§A4) answered for every T16 ticket, run against the code.

**T16.2 — closing #168.**

1. *Does the producer's capability exist?* **Partially.** T15.5's admin
   readers exist and are tested (`internal/payments/port.{GameAdminReader,
   CompetitionAdminReader}`, backed by `socialplayapp.Service.
   ListGameAdmins` and `competitionsapp.Service.ListCompetitionAdmins`).
   The Registration→Game and CompetitionEntry→Competition resolution reads
   **do not exist as exported methods** — confirmed by grep (§A2) — though
   the repository-level reads they would wrap already do.
2. *Does the consumer's own input actually contain the join key?* **Yes,
   already** — this is the clause that changes the outcome here versus
   T15.5. `RecordOfflinePaymentInput`/`RefundPaymentInput` both carry
   `PayableID` (confirmed at `payments/app/service.go:349-361`), which
   *is* a Registration's or CompetitionEntry's own id — the exact
   argument a new `GetRegistrationByID`/`GetEntryByID` method would take.
   **T15.5's gap was never "no id available"; it was "no read exists from
   that id to the id the admin-reader actually needs."** Naming this
   precisely is what recommendation 1/7's new clause is for: the specific
   field is `RecordOfflinePaymentInput.PayableID` /
   `RefundPaymentInput.PayableID`, and the specific new methods this
   ticket must add are `socialplayapp.Service.GetRegistrationByID`,
   `socialplayapp.Service.GetGame`, and `competitionsapp.Service.
   GetEntryByID` — named, not merely gestured at.

**Required, not optional, per this check**: T16.2's instructions must add
all three new app-layer read methods (mirroring `GetCompetition`'s
existing public-read shape) before anything else in the ticket can be
built.

**T16.3 — cascading cancel.**

1. *Producer capability?* **No** — confirmed by reading both port files in
   full (§A2): neither `RegistrationRepository` nor competitions'
   `Repository` has a bulk-transition write. `ListActiveForGame` /
   the competitions equivalent exist as reads, but nothing writes more than
   one row per call.
2. *Consumer's own input?* N/A in the ID-shaped sense — `CancelGame`/
   `CancelCompetition` already hold the `gameID`/`competitionID` they need
   (their own parameter). The gap is a **missing write capability**, not a
   missing join key — named explicitly so this check is not stretched to
   cover a shape it does not fit.

**Required**: T16.3 must add one new repository method per context
(`CancelAllActiveForGame`/equivalent), a single atomic `UPDATE …
WHERE game_id = $1 AND status != 'cancelled'`-shaped query — mirroring the
codebase's existing one-query-per-write-path convention
(`UpdateRegistrationStatus`, `promote_next_waiting`) rather than N
sequential `Update` calls.

**T16.4 — `RefundPayment` admits `competition_entry`.**

1. *Producer capability?* **Yes, already** — `payments/app.ServiceOptions.
   CompetitionEntryUpdater` (`competitionsport.CompetitionEntryPaymentUpdater`)
   has been wired since T10.6 (§A2), and `reconcileCompetitionEntryPaymentStatus`
   already knows how to push a paid status through it on the creation
   path.
2. *Consumer's own input?* **Yes** — `RefundPayment` already resolves the
   `domain.Payment` by id (`s.GetPayment`) before authorization runs, so
   `p.PayableID` is already in hand; no new read is needed to reach the
   updater, only to widen the gate that currently rejects the payable
   type outright.

**Required**: widen the payable-type gate; call the existing updater on
success, mirroring the registration-payable refund's existing call site
exactly rather than inventing a second pattern.

## §A11 — Shared-file pre-assignment, and the T16.2/T16.4 sequencing it produces

| Artifact | Owner | Notes |
|---|---|---|
| `internal/payments/app/service.go` | **T16.2 (Wave 1) → T16.4 (Wave 2)** | T16.2 restructures `RecordOfflinePayment`/`RefundPayment`/`authorizeOfflineRecording`/`ServiceOptions` wholesale (new resolver wiring, deleted input fields); T16.4 widens `RefundPayment`'s payable-type gate on top of that new shape rather than the old one, and should use T16.2's newly-resolved `EntrantPlayerID`/admin-reader wiring for the `competition_entry` refund path rather than the caller-supplied fields T16.2 will have already deleted. Sequenced, never concurrent |
| `proto/pickleball/payments/v1/payments.proto` | **T16.2 only** | Marks the now-fully-superseded admin-list/host/entrant fields `[deprecated = true]`; T16.4 touches no proto field, only the `RefundPayment` doc comment's out-of-scope list |
| `internal/socialplay/app/service.go` | **T16.2 (new methods) and T16.3 (CancelGame body), same wave** | Non-overlapping regions — T16.2 only *adds* `GetRegistrationByID`/`GetGame`; T16.3 only *edits* the existing `CancelGame` body. Low collision risk, cleared to run concurrently; whichever merges second resolves any line-proximity conflict per the established worktree-recovery-adjacent practice (`sprint-process.md`), not a redesign |
| `internal/competitions/app/service.go` | **T16.2 (new method) and T16.3 (CancelCompetition body), same wave** | Identical reasoning to the row above |
| `internal/socialplay/port/repository.go`, `internal/socialplay/adapter/postgres/*` | **T16.3 only** | New bulk-cancel query |
| `internal/competitions/port/repository.go`, `internal/competitions/adapter/postgres/*` | **T16.3 only** | New bulk-cancel query, mirrored |
| `internal/payments/port/{registration_lookup,game_lookup,entry_lookup}.go` (new files) | **T16.2 only** | |
| `internal/payments/adapter/{socialplay,competitions}/*_lookup.go` (new files) | **T16.2 only** | |
| **`HANDOFF.md`** | **this ceremony only** | An implementer that finds a stale line flags it for T17's Ceremony 1 and does not edit it — the standing rule, unchanged |
| **`docs/process/sprint-process.md`** | **this ceremony only** | No T16 execution ticket touches process |

## §A12 — Wave-1.5 checkpoint: condition checked, does not fire

The checkpoint applies when a new cross-cutting decision has three or more
first-time in-sprint consumers. **No T16 ticket introduces a shared
decision with three consumers** — T16.2's new resolver methods have
exactly one in-sprint consumer each (T16.2 itself); T16.3's new bulk-cancel
query has one consumer per context. **No Wave-1.5 checkpoint applies.**

---

# Ceremony 2 — Sprint planning

## Sprint goal

> **Payments actually reads the admin stores it built last sprint.**
> `RecordOfflinePayment`/`RefundPayment` resolve a Registration's owning
> Game and a CompetitionEntry's owning Competition for themselves — the
> exact join-key gap T15.5 discovered and could not close — and consume
> T15.5's already-built, already-tested admin readers for real. This closes
> **#168** and narrows **#149** to its one remaining, D1-blocked fact.
> Alongside it: cancelling a Game or a Competition actually cancels its
> Registrations/Entries with it, closing the reachable half of a
> seven-sprint-old gap (**#124**) the way its own recorded disagreement's
> scoring condition requires rather than deferring a fifth time; and a
> stale issue (**#125**) found during this ceremony's own backlog
> verification is corrected to what it actually still tracks and closed by
> a small, mechanical widening of `RefundPayment`. On process: the three
> `sprint-process.md` amendments T15's retro asked for land directly in
> this ceremony rather than waiting for a later ticket, a new FK-race
> residual (**#195**) is tracked rather than lost the way #185 almost was,
> and D1/D2 go back to the user unanswered — a fourth and second deferral
> respectively, said plainly.

**What this sprint does not claim** (the half PM insists on):

- **#168 closes; #149 does not fully close.** `booking_host_id` remains
  caller-supplied — it is the one fact none of T16's new reads can resolve,
  because Booking has no owner concept to resolve it against (D1).
- **D1 and D2 remain unanswered.** No T16 ticket implements `CancelBooking`
  authorization or a reviewer-authorship carve-out; none may guess either.
- **The court-Bookings half of #124 is still deferred**, and for a stated
  reason (§A9) — not a silent fifth pass.
- **#195 is filed, not fixed.** The FK-race class remains latent, at low
  practical severity (§A5), tracked for T17.
- **#164/#145/#134/#126/#130/#167 are untouched**, each with a reason
  recorded in §A9.
- **This sprint is smaller than recent precedent (3 execution tickets, 16
  points, versus T15's 7/34) by design, not by omission.** Most of the
  open backlog is genuinely blocked — on D1, on D2, on a real IdP tenant,
  or on Product Owner input that has not arrived across three to seven
  sprints of asking. T16 takes the three items that are actually
  unblocked and reachable this sprint, plus the process debt T15's retro
  named, rather than manufacturing scope to match a prior sprint's size.

## Tickets — 3 items, 16 points

### T16.2 — Close #168 for real: Payments resolves a Registration's Game and a CompetitionEntry's Competition, and reads the admin stores it already built

- **Story:** As a facility operator, I want offline payment recording and
  refunding to be authorized against the real Game/Competition Admin
  assignments the platform already stores, so that "per-game Game Admins
  can record offline payments" — CLAUDE.md's own locked decision — is a
  rule the system enforces rather than one T15 built the infrastructure for
  and then could not wire in.
- **Points:** 8 · **Role:** `role:principal-engineer` · **Type:** `type:story`
- **Description:** Finishes what T15.5 started and disclosed as blocked.
  T15.5 built `port.GameAdminReader`/`CompetitionAdminReader` and their
  real adapters, tested against the real `socialplay`/`competitions`
  `app.Service`s, and left both unwired because Payments had no read from
  a Registration/CompetitionEntry id to the Game/Competition id those
  readers need. This ticket builds exactly that missing read — §A10's
  dependency-completeness check names the specific methods required.
  **Closes #168.** Also resolves two of #149's remaining three facts
  (`game_host_id`, `entrant_player_id`) as a direct consequence of
  building the same resolution reads — leaving `booking_host_id` as
  #149's one remaining fact, blocked on D1.

**Instructions**

1. **Add `internal/socialplay/app.Service.GetRegistrationByID(ctx,
   registrationID string) (domain.Registration, error)`**, exposing the
   already-implemented `port.RegistrationRepository.GetByID` — a public
   read with no ownership check, mirroring `GetCompetition`'s existing
   shape on the Competitions side. Reading a Registration by its own
   opaque id reveals no more than `ListRegistrationsForGame` already does
   to a Host; confirm this against the actual read paths rather than
   asserting it (T15.6's own instruction 5 is the template for how to
   check an enumeration-oracle claim rather than assume it).
2. **Add `internal/socialplay/app.Service.GetGame(ctx, gameID string)
   (domain.Game, error)`**, exposing the already-implemented
   `port.GameRepository.GetByID` — needed to resolve `Game.HostID` for the
   `game_host_id` fact. Social Play has never exposed a single-Game read at
   the app layer before this; note that explicitly in the PR, since it is
   a real gap this ticket closes as a side effect, not an oversight being
   quietly worked around.
3. **Add `internal/competitions/app.Service.GetEntryByID(ctx, entryID
   string) (domain.CompetitionEntry, error)`**, exposing the
   already-implemented `port.Repository.GetEntryByID` (already called
   *internally* at `service.go:549` by `MarkCompetitionEntryPaymentStatus`
   — this ticket exports the same read as its own public method, it does
   not add a new repository capability). Resolves both `CompetitionID` and
   `PlayerID` in one call.
4. **Build the new Payments-side resolver ports** —
   `internal/payments/port.RegistrationLookup` (`GameIDForRegistration`),
   `internal/payments/port.GameLookup` (`HostIDForGame`), and
   `internal/payments/port.EntryLookup`
   (`CompetitionIDAndPlayerIDForEntry`) — implemented in
   `internal/payments/adapter/socialplay` and
   `internal/payments/adapter/competitions` against the three new methods
   above. Translate errors at the boundary per rule 5 (`%s`, not `%w`),
   exactly as T15.5's admin readers already do.
5. **Wire all five ports** (the three new lookups plus T15.5's already-built,
   already-tested `GameAdminReader`/`CompetitionAdminReader`) into
   `payments/app.ServiceOptions`/`Service`, and into both live call sites of
   `authorizeOfflineRecording` (`RecordOfflinePayment`, `RefundPayment`).
   For a `PayableTypeRegistration`/`PayableTypeNoShowFee` payable: resolve
   `GameID` from the Registration, then `HostID` from the Game and the
   admin set from `GameAdminReader`. For `PayableTypeCompetitionEntry`:
   resolve `CompetitionID` and `PlayerID` from the Entry in one call, then
   the admin set from `CompetitionAdminReader`. `PayableTypeBooking` is
   untouched — `BookingHostID` stays caller-supplied, per instruction 6.
6. **Delete the now-superseded app-input fields** (`GameHostID`,
   `AssignedGameAdminUserIDs`, `EntrantPlayerID`,
   `AssignedCompetitionAdminUserIDs`) from `RecordOfflinePaymentInput`/
   `RefundPaymentInput` rather than leaving them unread — T14.5/T15.5's
   established load-bearing pattern: a future re-plumb of the wire list
   must fail to *compile*, not silently restore a forgeable check.
   `BookingHostID` stays on both input structs, untouched.
7. **Mark the now-fully-superseded wire fields**
   (`assigned_game_admin_user_ids`, `assigned_competition_admin_user_ids`,
   `entrant_player_id`, `game_host_id`) `[deprecated = true]` in the proto —
   T15.5 deliberately left them non-deprecated because they were "still
   the only mechanism"; that is no longer true after this ticket, and the
   PR must say so explicitly rather than leave the deprecation
   unexplained. `booking_host_id` stays non-deprecated. Regenerate.
8. **Close #168 after merge**, per DoD step 5, with a comment naming this
   PR. **State plainly, in the PR and in the proto's own comments, what is
   still not fixed**: `booking_host_id` remains caller-supplied, and #149
   stays open for that one fact alone — do not let "#168 closed" read as
   "Payments now verifies every ownership fact." Post the narrowing
   comment to #149 itself, per the newly-amended review checklist
   (§A4) — this is a correction to #149's content, not a close, and the
   clause applies to it directly.
9. **Non-functional:** TDD-first. Headline mutation checks, reported by
   name: a caller naming themselves in the now-deleted-from-input,
   still-on-the-wire deprecated fields must be refused (the forgery case
   T15.5's own note on this exists to prevent); a genuine Game/Competition
   Admin succeeds and is refused again once revoked (positive control,
   T15.4's mutation table is the template); a Game's real Host succeeds via
   the newly-resolved `GameHostID`; a Competition's real entrant succeeds
   via the newly-resolved `EntrantPlayerID`. Watch the cross-context test
   shape (T14.8/T15.5's standing note): a fake that returns whatever it is
   told proves nothing about the real seam — use the real `socialplay`/
   `competitions` `app.Service`s in the boundary tests, as T15.5 already
   did for the admin readers.
10. **Sibling sweep, reported either way** (the standing instruction every
    T15 ticket carried, per T14's own recommendation 10, threaded
    forward): is there any other Payments authorization branch
    still comparing against a caller-supplied fact this ticket's new
    resolvers could also answer? (From inspection: no — `BookingHostID` is
    the only remaining one, and it is blocked on D1, not on a missing
    read; state this explicitly rather than leaving it implicit.)

### T16.3 — Cancelling a Game or Competition cancels its active Registrations/Entries (partial fix for #124; closes the mirrored Competitions gap found this ceremony)

- **Story:** As a Player registered in a Game (or entered in a Competition)
  that its Host cancels, I want my Registration/Entry cancelled with it, so
  that I am not left holding a "registered" seat in something that no
  longer exists, and so capacity/waitlist numbers stay honest.
- **Points:** 5 · **Role:** `role:product-engineer` · **Type:** `type:story`
- **Description:** Resolves the carried T15-plan §A11 PO/PE disagreement
  on #124 (§A9): D1 remains unanswered, so PO's position governs — the
  Registrations half has no dependency on D1 and ships now. **Partial fix
  for #124** — the court-Bookings half stays deferred, and this ticket's
  PR states why (§A9) rather than silently narrowing scope. Also fixes
  `CancelCompetition`'s identical, previously-undisclosed gap (§A7), built
  as a mirror in the same ticket rather than left as a second disclosed
  residual.

**Instructions**

1. **Add one new bulk-write repository method per context**:
   `internal/socialplay/port.RegistrationRepository.CancelAllActiveForGame(ctx,
   gameID string) (int, error)` and the Competitions equivalent —
   a single atomic `UPDATE … SET status = 'cancelled' WHERE game_id = $1
   AND status != 'cancelled'`-shaped query (mirroring the codebase's
   existing one-query-per-write-path convention:
   `UpdateRegistrationStatus`, `promote_next_waiting`), returning the
   number of rows actually transitioned. **Not** N sequential
   `Update` calls — that would be N round trips for a single Host action
   and loses the atomicity a concurrent join during the cancel window
   needs.
2. **Wire it into `app.Service.CancelGame`/`CancelCompetition`**, called
   after `game.Cancel()`/`competition.Cancel()` succeeds and the status
   write persists — cancelling the parent before cascading to its
   children, so a failure partway through never leaves an active Game with
   cancelled Registrations (the reverse order would).
3. **Do not touch court-Bookings in this ticket.** State explicitly, in
   the PR and as a comment on #124 itself, why: the cascade would call
   `app.Service.CancelBooking`, whose signature ADR-0015's still-open D1
   may yet change (adding an actor parameter is one of D1's four options).
   Building against a signature under active escalation means either
   guessing D1 or shipping code this project already knows may need
   rewriting — PE's argument from the recorded disagreement, preserved
   rather than overridden by PO's win on the Registrations half.
4. **Do not touch refunds.** #124's own "why this needs a decision" list
   names whether cancelled Registrations' Payments get refunded as its own
   open sub-question (item 2), now sharper given T12.3's `RefundPayment`
   exists. This ticket cancels the Registration/Entry only; it does not
   call `RefundPayment` and does not answer whether a future ticket
   should.
5. **Waitlist entries**, Social Play only (Competitions has no waitlist):
   confirm and test what a cancelled Game does to its waitlist — no
   promotion should fire for a cancelled Game. Non-functional, prove with
   a test rather than asserting the behaviour is already correct.
6. **Update #124's own text**, mirroring the #147 re-titling precedent:
   comment explaining which half is done, which remains (court-Bookings),
   and why (D1's signature risk) — do not close #124; this is a partial
   fix.
7. **Non-functional:** TDD-first. Mutation-check: revert the cascade call
   in `CancelGame`/`CancelCompetition`, confirm Registrations/Entries stay
   in their pre-cancel status after the parent cancels, confirm the new
   tests catch it and are restored once the fix returns.
8. **Sibling sweep, reported either way:** any other Cancel-shaped
   transition in this codebase with the same unshipped cascade? (Check
   `RecurringHireTemplate`, `CompetitionEntry.Status` transitions outside
   `CancelCompetition`, and report the categories searched even if the
   answer is none beyond what this ticket already found.)

### T16.4 — `RefundPayment` admits `competition_entry` payables (closes the corrected #125)

- **Story:** As a Host or Competition Admin who recorded a competition
  entry's payment in error, I want to refund it the same way I can already
  refund a Booking or Registration payment, so that reversing a mistaken
  charge does not require an out-of-band workaround.
- **Points:** 3 · **Role:** `role:product-engineer` · **Type:** `type:bug`
- **Description:** **Closes #125** (corrected and re-titled this ceremony,
  §A6 — the Competitions-`PayableType` half it originally asked for
  shipped in T10.6/#96; what remains is `RefundPayment`'s payable-type
  gate). **Depends on T16.2 — Wave 2** (§A11): builds on the resolved
  `EntrantPlayerID`/`CompetitionAdminReader` wiring T16.2 lands, rather
  than authorizing the new refund path against the caller-supplied fields
  T16.2 will have already deleted.

**Instructions**

1. **Widen `RefundPayment`'s payable-type gate** to admit
   `PayableTypeCompetitionEntry`, mirroring how it already admits
   `booking`/`registration`. Reuse `authorizeOfflineRecording`'s existing
   `PayableTypeCompetitionEntry` branch (already exists for the
   *recording* path, T10.6) rather than writing a second authorization
   rule for the refund path — T12.3's own precedent for `RefundPayment` is
   "reusing `authorizeOfflineRecording` unchanged… a refund is the same
   Host/Game-Admin action as recording one," and this ticket follows it
   for the Competitions side.
2. **Push `PaymentStatusRefunded` through the existing
   `CompetitionEntryUpdater`** (`payments/app.ServiceOptions`, wired since
   T10.6, §A10) on a successful refund — mirroring exactly how a
   `registration`-payable refund already pushes the equivalent status
   through `RegistrationPaymentUpdater`. Read that existing call site
   before writing this one; do not invent a second pattern for the
   identical concept.
3. **Unlike #130's `no_show_fee` gap, this one does not carry an open
   product question** — state this explicitly in the PR, since #130 is
   deliberately deferred (§A9) for exactly the reason this ticket is not:
   a competition entry's refund is symmetric with a registration's, which
   this codebase already ships, and #125's own history shows the
   Competitions-payment path was always meant to be full parity with
   Social Play's (T10.6's own stated goal), not a narrower one.
4. **Close #125 after merge**, per DoD step 5, with a comment naming this
   PR. State in the review, per this ceremony's own new correction clause
   (§A4), that #125's earlier misdescription (asking for infrastructure
   that already shipped in T10.6) was corrected at T16's Ceremony 1, not
   by this PR — do not re-describe the correction as this ticket's own
   finding.
5. **Non-functional:** TDD-first. Extend
   `TestRefundPayment_OutOfScopePayableTypesRejected` to prove
   `competition_entry` is **no longer** in the out-of-scope set (moving,
   not deleting, the assertion — a test that simply vanishes proves
   nothing was checked). Mutation-check the updater push: temporarily
   disable the `CompetitionEntryUpdater` call, confirm a test fails,
   restore it.
6. **Sibling sweep, reported either way:** does `CreateOnlinePayment`'s or
   `ConfirmOnlinePayment`'s own payable-type handling need the identical
   widening, or is refund the only online/offline asymmetry left for
   `competition_entry`? Check directly rather than assuming refund is the
   last gap.

## Waves

**Wave 1 — no in-sprint dependencies (2 tickets, 13 points)**
`T16.2` (close #168) · `T16.3` (cascading cancel)

**Wave 2 — depends on T16.2 (1 ticket, 3 points)**
`T16.4` (RefundPayment admits `competition_entry`) — needs T16.2's
resolved-facts shape on `payments/app/service.go` to land first (§A11);
technically could authorize against the old caller-supplied
`EntrantPlayerID`/admin-list fields instead, but doing so would ship a
refund path this same sprint already knows it is about to delete the
input for.

**No Wave-1.5 checkpoint** — condition checked (§A12), does not fire.

## Recorded disagreements (Ceremony 2 rule 3 — not smoothed over)

1. **PE vs. PM — fold #195 into this sprint or track it for T17?** Full
   text in §A5. Resolved in favour of PM's scope argument for this sprint,
   with a scoring condition binding T17's Ceremony 1: a third sprint of
   the same defect class sitting disclosed-but-open is treated as
   confirming PE's "permanent furniture" concern, not re-argued fresh.
2. **PO vs. PE — #124's split**, resolved rather than carried forward
   again. Full text in §A9. PO's position governs for the Registrations/
   Entries half (taken, T16.3); PE's signature-churn concern governs the
   court-Bookings half (deferred, reasoned).

## Sprint-level Definition of Done

All of `sprint-process.md`'s standing DoD, plus the scorings T16 owes,
stated now so they are not improvised at the retro:

1. All 3 tickets merged per the per-ticket DoD; sprint goal met or
   explicitly descoped with reasoning recorded.
2. **The merged-fix issue sweep run and reported with its count** — by the
   retro (reporting, not blocking) and again by T17's Ceremony 1
   (authoritative).
3. **Scoring owed at the retro:**
   - **(a)** Did the "closes #N" mandatory-close amendment (§A4,
     recommendation 2) actually change the outcome for T16.2's and
     T16.4's own "closes #168" / "closes #125" PR titles? Report whether
     each performed the close at review time or explicitly deferred it
     with a stated reason — the first live test of this amendment.
   - **(b)** Did T16.2 actually close #168, and did it actually narrow
     #149 to its one remaining fact? Score both directly against the
     merged code, not against the PR's own account.
   - **(c)** Did T16.3 actually ship both halves (Social Play and
     Competitions), and does #124 carry the required "which half, why the
     other stays deferred" comment?
   - **(d)** Was the new consumer-input clause of the
     dependency-completeness check (§A4, recommendation 1/7) exercised
     correctly for T16.2 — did the ticket's own dispatch verify
     `PayableID` was the join key before work began, the way §A10 states
     it should?
4. **Not scoreable by T16 and deliberately not pre-empted:** D1 and D2
   remain the user's. If either is answered mid-sprint, the answer's own
   trigger takes over and T16's plan does not constrain it.
5. Retro in `docs/process/t16-retro.md`, indexed by a `## T16 sprint
   retro` stub in `docs/LESSONS.md`. `HANDOFF.md`/`CLAUDE.md` state
   updated — noting that **T17's Ceremony 1**, not the retro, corrects
   T16's Docs-index row.
