# T17 Sprint Plan — Ceremonies 1 + 2

Backlog refinement (Ceremony 1) and sprint planning (Ceremony 2) per
`docs/process/sprint-process.md`, read in its **T17-amended** form (this
ceremony's own two edits — the "Same-wave shared-interface verification"
subsection under Execution, and the transcription clause added to "The
dependency-completeness check" — both land directly in this same PR, per
this project's established precedent that a process amendment produced by
the same instruction that requested it lands in that ceremony's own PR
rather than waiting for a later ticket; see §A4). Held against
`docs/process/t16-retro.md` (PR #201, merged 17:26:33Z), `HANDOFF.md`
including its Docs index, `CLAUDE.md`, and the live PR/issue state of
`nhuthuynh/white-label` (GitHub-side name `pickleball-platform`).

**Every factual claim below was re-derived against the repository and the
GitHub API during this ceremony**, at the merged branch tip (`c35b6ae`,
T16 retro's own merge), rather than taken from the retro's or T16's plan's
prose — CLAUDE.md rule 10 applied to planning. Where this ceremony repeats
a claim it could not independently re-check, it says so.

## §A0 — The merged-fix issue sweep, run as this ceremony's first act

Per `sprint-process.md`'s Definition of Done, T17's Ceremony 1 is the next
sweep's **authoritative** moment, and it does not defer to T16's retro
having already reported one clean — recommendation 5, executed here rather
than merely acknowledged, exactly as T16's own Ceremony 1 executed T15's
identical recommendation.

**Step 1 — list the open issues.** `list_issues(state: OPEN)` at ceremony
start → **`totalCount: 11`**: #124, #126, #130, #134, #144, #145, #149,
#164, #167, #195, #198.

**Step 2 — reconcile arithmetically, before reading any issue
individually.** `open_at_end_of_T16's_own_Ceremony_1 − closed_during_T16 +
opened_during_T16`. T16's own Ceremony 1 (§A0 of `docs/process/
t16-sprint-plan.md`) left the count at **12** (11 carried from T15, +1 for
#195, filed that ceremony). During T16's execution: **#168** closed (T16.2,
PR #199) and **#125** closed (T16.4, PR #200) — `12 − 2 = 10`; **#198**
opened (T16.2's own sibling sweep, instruction 10) — `10 + 1 = 11`. **This
matches the live `totalCount: 11` read at this ceremony's start exactly**,
re-derived independently rather than taken from T16 retro's own table (§3
there, which reported the identical arithmetic and the identical number —
re-verified here, not trusted).

**Step 3 — cross-reference merged PRs against the open list.** No PR has
merged since T16 retro's own PR (#201, 17:26:33Z, doc-only, closes no
issues per its own body: *"Issue-tracker actions this ceremony: none"*).
This ceremony's own start is therefore the same tree T16's retro already
swept. **Re-verified independently rather than assumed identical**, via
direct `issue_read` rather than reading the retro's table:

| Issue | `state` (re-fetched now) | `state_reason` | `closed_at` | Matches T16 retro's table? |
|---|---|---|---|---|
| #168 | `closed` | `completed` | `2026-08-15T16:56:44Z` | Yes |
| #125 | `closed` | `completed` | `2026-08-15T17:12:06Z` | Yes |

Both closes were also cross-checked against their resolving PRs' own
`merged_at` timestamps, fetched directly via `pull_request_read` rather
than assumed: PR #199 (T16.2) `merged_at: 2026-08-15T16:56:39Z` — 5 seconds
before #168's close; PR #200 (T16.4) `merged_at: 2026-08-15T17:12:00Z` — 6
seconds before #125's close. Both closes are genuine, both are correctly
attributed, and both were performed by the party that merged (per-PR
optional-early-close, not the sweep) — consistent with T16 retro's own
account, now independently re-derived rather than trusted.

**Sweep result: clean, second sprint running.** Zero unclosed hits, zero
mis-attributed closes, arithmetic reconciles exactly. Unlike T16's own
sweep (which had to re-derive results from a live sprint retro's table),
this sweep has the added property that literally nothing has changed
between the retro's report and this ceremony's re-check — the cleanest
possible instance of "verified, not trusted" the standing rule can produce.

## §A1 — Correcting T16's Docs-index row: checked, found already correct, not re-touched

Per this ceremony's instruction: `HANDOFF.md`'s T16 row was already
corrected by T16's own retro (PR #201) — an intentional, explicitly-argued
departure from the usual "next Ceremony 1 corrects it" convention, because
T16's retro cited only already-merged PR numbers (its own future PR number
was the one thing a retro still cannot cite, and T16's retro did not need
to). **This ceremony's job is to verify that correction was performed
accurately, not to re-perform it.**

Checked directly against `HANDOFF.md` as it stands now:

| Field | What the row says | Independently re-verified this ceremony |
|---|---|---|
| Sprint plan link | `docs/process/t16-sprint-plan.md` | File exists, read in full this ceremony |
| Retro link | `docs/process/t16-retro.md` (7 findings, "a real defect that reached the shared branch's own tip for 15m21s — traced to a false review-time claim, not to anything genuinely uncatchable") | File exists, read in full this ceremony; finding count and characterization both match its own content |
| Reviews cell | PRs #197 (T16.3) → #199 (T16.2) → #200 (T16.4), in that merge order, verified against `merged_at` | Re-fetched all three via `pull_request_read`: `merged_at` **16:28:35Z → 16:56:39Z → 17:12:00Z** — ascending, matches the stated order exactly |
| Task-backlog narrative | The retro's own agreed sentence (quoted in full in `HANDOFF.md`'s T16 entry), stating plainly that a real defect reached the shared branch's tip, naming the cause (a false review-time verification claim, not an uncatchable timing accident), and restating D1/D2 as unanswered | Cross-checked against `docs/process/t16-retro.md`'s "The sprint goal, scored" section — the two are the same sentence, not a paraphrase drifting from it |

**Row is accurate. Not re-touched.** A new T17 row is added below in the
same before-the-sprint form (§A1a), for T18's Ceremony 1 to correct in
turn — the same convention every prior ceremony has followed.

### §A1a — New T17 Docs-index row (placeholder, correct as written)

Added to `HANDOFF.md`'s Docs index:

> | T17 | `docs/process/t17-sprint-plan.md` (Ceremony 1 re-runs the merged-fix
> sweep clean for the second sprint running, lands two `sprint-process.md`
> amendments — the same-wave shared-interface verification rule and the
> dependency-completeness check's transcription clause — takes #195's four
> per-context tickets now that its T16-plan scoring condition has fired,
> and takes the corrected #198; Ceremony 2 tickets 5 items) | not yet
> written | not yet opened | none new | — |

## §A2 — What this ceremony verified, and how

| Claim | How it was checked | Result |
|---|---|---|
| Open-issue count at ceremony start | live `list_issues(state: OPEN)` | `totalCount: 11` |
| #168, #125 genuinely closed, correctly attributed | `issue_read(get)` on each, cross-checked against `pull_request_read` on #199/#200 | both `state: closed`, `state_reason: completed`; closed within seconds of their resolving PR's `merged_at` |
| #198's gap still exists unfixed | `grep -n "EntrantPlayerID\|AssignedCompetitionAdminUserIDs" internal/payments/app/service.go` | both fields still present; `authorizeOnlineCreation` (read in full) still compares them directly, unchanged since T16.2 |
| #198 carried no labels | `issue_read(get)` before this ceremony's action | confirmed — empty label set. **Corrected this ceremony**: `role:product-engineer` (matching #168's own established role for this shape of work — Payments authorization resolved via existing resolver ports), `type:bug`, `context:payments` applied via `issue_write`; re-fetched after to confirm |
| #195's nine FK paths are unchanged since it was filed | `git log --oneline` shows no commit touching `db/migrations/*.sql` or any of the nine listed files' Postgres adapters since T16.3/T16.4 (the only T16 PRs touching adjacent adapter code); spot-checked two of the nine directly | `internal/facilities/adapter/postgres/repository.go` and `internal/socialplay/adapter/postgres/repository.go` (`translate_test.go` in both) confirmed present with no FK-violation mapping for the rows #195 names; unchanged |
| Next free migration / ADR numbers | `ls db/migrations`, `ls docs/adr` | `0021` last migration, `0016` last ADR → **`0022`**/**`0017`** free (not consumed — no new migration or ADR dispatched this ceremony; #195's fix is translation-only, no new constraint) |
| D1 (#144) still has exactly one comment | `issue_read(get_comments)` | confirmed — T14.3's original escalation only, unchanged across T14→T16, now T17 |
| D2 (ADR-0016) still unanswered | read `docs/adr/0016-*.md`'s `## Status` | unchanged: "Escalated — awaiting decision (D2)" |
| #124's court-Bookings half still deferred, comment present | `issue_read(get_comments)` | confirmed — T16.3's comment (`#124#issuecomment-5303145228`) states which half shipped (Registrations/Entries) and which remains (court-Bookings, D1-blocked), issue left `state: open` |
| `make generate && make test-domain && make gate-coverage` still green on the unmodified tree | ran directly | `make test-domain`: all 12 packages green; `make gate-coverage`: `OK — all 41 package(s) … executed by "ci-checks"` |
| Local worktree matches the shared branch's real tip | `git status`, `git log --oneline -5` | clean, up to date with `origin/claude/go-backend-pickleball-7up34j`, HEAD at `c35b6ae` (T16 retro's own merge) |

**Not re-verified by this ceremony, and named as such:** the full `make
test`/`ci-integration` Docker path (no Docker daemon reachable here, same
standing gap as every prior sprint); any claim about what Jenkins would
run (no Jenkins job exists).

---

# Ceremony 1 — Backlog refinement

## §A3 — T16 retro's five recommendations, each given a disposition

| # | Recommendation | Disposition |
|---|---|---|
| 1 | Add a same-wave shared-interface verification rule to `sprint-process.md` | **Executed here** — a new "Same-wave shared-interface verification" subsection lands directly in `sprint-process.md`'s Execution section (§A4), naming the `git fetch`-and-merge-locally / `refs/pull/<n>/merge` verification method as mandatory for the second-to-merge PR in any flagged same-wave shared-interface pair, and placing an advance-flagging obligation on Ceremony 1 going forward |
| 2 | Add #198 to T17's ranked backlog, labelled | **Executed here** — read via `issue_read` first (§A2), confirmed accurate and still unfixed, labelled `role:product-engineer`/`type:bug`/`context:payments` (§A2), ranked and **taken** as **T17.1** (§A6, §A8) |
| 3 | Escalate D1 (#144) a fifth time, stating explicitly that its footprint has grown, not just that it remains unanswered | **Executed here** (§A7) — two named instances of scope now shaped by D1's absence, stated in this doc's own "does not claim" section per the task's instruction, not implemented or guessed at |
| 4 | Transcribe already-known unblocking facts from a prior sprint's own artifacts into a new ticket's text, rather than asking the implementer to re-derive them, when drafting T17's tickets | **Applied directly in both T17.1 and T17.2–T17.5's instructions** (§A6, §A9's tickets) — #198's own body and #195's own body already enumerate the exact methods/fields/table rows needed; transcribed verbatim rather than re-derived. **Also formalized into `sprint-process.md`** this ceremony — see §A4's second amendment and its own justification for why this is now written in rather than left a practice |
| 5 | Continue treating the merged-fix sweep as authoritative regardless of the prior retro's clean result | **Executed here** (§A0) — independently re-derived the open count and re-verified #168's and #125's closes from the API rather than the retro's own table |

## §A4 — Two `sprint-process.md` amendments

Landed directly in `docs/process/sprint-process.md` by this same PR, per
this ceremony's own instruction and this project's established precedent
(T16's own three amendments landed the same way, in its own Ceremony 1
PR, rather than being spun into a ticket for later execution).

**1. "Same-wave shared-interface verification"** (recommendation 1) — a new
subsection under Execution, immediately before "Recovering an interrupted
session's work" (the same altitude: both govern how a specific, recurring
failure shape is verified against, not what gets built). Summarized here;
the amendment itself is the source of truth, not this summary:

- **When it applies**: two same-wave tickets, correctly dispatched with no
  functional dependency between them, both touch one shared interface's
  blast radius — one widening it, one authoring a new implementer — in
  disjoint files that produce no textual conflict.
- **Why `mergeable_state: clean` is not enough**: it certifies absence of a
  textual conflict, not that the combined tree type-checks. A new file
  implementing a widened interface and the widening itself never share a
  line, so they merge cleanly and can still fail `go vet`/`go build`.
- **The rule**: whichever of the pair merges **second** must verify its own
  PR against an actually reconstructed post-merge tree (`git fetch` + local
  merge, or GitHub's `refs/pull/<n>/merge` ref) before its review may claim
  a green run against "the merged state" — not infer that state from a
  GitHub flag.
- **Ceremony 1's own new obligation**: when the dependency-completeness
  check finds this shape in a same-wave pair, the ticket text for whichever
  is expected to merge second names the shared interface and the sibling
  ticket explicitly, so the reviewer knows the rule applies.

This is the direct, narrow fix for T16 retro's finding 1: T16.2's own
review claimed "base already included T16.3 (`mergeable_state` clean, no
test-merge needed)" — a claim the commit graph proved false, since the
reviewed commit's only parent was the pre-widening base. T16.4's review is
the positive example this rule generalizes into policy: it checked out the
bare shared-branch tip in an independent worktree and ran the toolchain
directly against it.

**2. The dependency-completeness check gains a transcription clause**
(recommendation 4, applied and now formalized). Added to "The
dependency-completeness check" subsection under Ceremony 1:

> When a ticket's own unblocking facts — the specific methods, fields, or
> join keys it needs — are already fully enumerated in a prior sprint's own
> artifact (an issue body, a PR body, a review), the ticket's Instructions
> transcribe them directly rather than asking the implementer to re-derive
> them. Naming the read method to call without naming the join key it needs
> is the shape of gap the check's second question already exists to close
> (T15.5); asking an implementer to re-discover a fact someone already
> wrote down, in the same repository, is the same waste in a different
> direction.

**Why formalized now, not left a practice** (T16 retro's own explicit
caution against over-mechanizing on one data point, addressed directly):
this is the **third** occurrence of the same shape in this project's
history — the practice was first named as a lesson (an earlier sprint's
retro, per T16 retro's own framing of "an earlier one"), then scored as a
real, cheaply-avoidable miss at T15→T16 (T15.5's review had already
enumerated T16.2's exact methods and join key twenty minutes before T15's
own retro wrote a generic "flag for T16 planning" instead of transcribing
them), and this ceremony is the **third**: #198's and #195's own bodies
already fully specify what T17.1–T17.5 need, down to the exact method
names, table rows, and reuse targets (§A6, §A9). **This matches the exact
maturation arc the dependency-completeness check itself followed** — run
informally for three sprints (T13–T15) before being written into
`sprint-process.md` at T16, rather than mechanized after one instance.
Writing this in now follows that same precedent rather than pre-empting
it. If a future retro finds this transcription silently skipped despite an
artifact clearly containing the facts, that is the signal the rule needs
strengthening (e.g. into an explicit Ceremony-1 checklist line naming the
source artifact per ticket) — not evidence it was formalized too early.

**Non-functional, verified rather than assumed:** documentation only, no
Go/proto/SQL/`Makefile` change. `make test-domain` and `make gate-coverage`
re-run after the edit (§A2) and confirmed unaffected.

## §A5 — #195: the T16-plan scoring condition has fired, and PE's concern is adopted

**Re-verified, not assumed** (§A2, §A0): #195 is still open, still
carrying zero PR references, unchanged since it was filed at T16's
Ceremony 1.

**The condition, quoted from `docs/process/t16-sprint-plan.md` §A5, in
full rather than paraphrased**: *"if #195 is still untaken at T17's
Ceremony 1, that is the third sprint the same defect class has sat
disclosed-but-open (counting #185's own one-sprint gap before it was
filed), and PE's 'permanent furniture' concern should be treated as
confirmed rather than re-argued from scratch."*

**Counting the sprints, precisely**: T15 (T15.6 disclosed the class,
unfiled — sprint 1); T16 (filed as #195 at Ceremony 1, deliberately not
taken — sprint 2); T17 (this ceremony) is the third. **The condition
fires.** Per its own text, PE's concern — that deferring a known
security-adjacent defect class two sprints running risks it going the way
of #147/#149's "permanent furniture" pattern — is **adopted as confirmed,
not re-argued.** #195 is taken this sprint.

**Sizing it honestly, the way T16's plan already argued it would need to
be** (§A5 there): nine FK-backed write paths across four bounded contexts
is materially larger than T15.6's own single-FK, three-context fix (which
alone landed as 25 files, +1255/−46). Splitting into one ticket per
**writing** context — mirroring how each context owns its own Postgres
adapter and its own `toStatus` exhaustiveness table (the T14.7 guard) —
keeps each ticket mechanical, single-context, and file-disjoint from every
other T17 ticket, rather than one large cross-context PR repeating the
review-attention risk §A5 of T16's plan already flagged. Grouped from
#195's own table, transcribed directly (recommendation 4, §A4):

| Ticket | Context (write side) | FK paths | Guarding read (already exists) |
|---|---|---|---|
| **T17.2** | Social Play | `registrations.game_id`; `games.venue_facility_id` | Game fetched before insert; `FacilityExists` |
| **T17.3** | Competitions | `competition_entries.competition_id`; `competitions.venue_facility_id` | Competition fetched before insert; `FacilityExists` |
| **T17.4** | Facilities | `discount_rules.facility_id`; `courts.facility_id`; `facility_camera_links.facility_id` | `EnsureFacilityOwner` / `GetFacilityByID` (all three) |
| **T17.5** | Booking | `recurring_hire_templates.court_id`; `recurring_hire_templates.requested_by_user_id` | `FacilityIDForCourt`; the Identity lookup |

**No cross-ticket file overlap**: each ticket only edits its own context's
`adapter/postgres/repository.go` (or equivalent), its own `domain/errors.go`
sentinels (reusing an existing not-found sentinel per #195's own suggested
shape — `ErrFacilityNotFound`-shaped is the natural candidate for every row
whose parent is a Facility, per context), and its own `adapter/grpcapi`
`toStatus` exhaustiveness table. No new migration in any of the four — this
is Postgres-error-translation only, mapping an existing `23503` to an
existing domain sentinel; the FK constraints themselves already exist.
Confirmed: `0022` stays the next free migration number (§A2), not consumed
by this sprint.

**Recorded disagreement — PE vs. PM, resolved by the fired condition
rather than re-argued** (Ceremony 2 rule 3). PM's residual position: even
with the condition fired, #195's four tickets (§A9 estimates 14 points
combined) plus #198 (3 points) is a materially bigger sprint than T16's 16
points, and every one of the nine paths still requires a genuine
concurrent-delete race to trigger — none of the four affected contexts
ships a bulk-delete flow today, so real-world exposure has not changed
since T16 assessed it as low. PE's position, which governs per the
condition's own text: severity assessment is not what the condition was
testing — the condition was specifically about whether *deferring a known
class a second time* becomes its own risk (silent permanence), independent
of any single sprint's severity read, and that is the question T16 already
decided to bind itself on rather than re-litigate here. **PE's position
governs; #195 is taken in full this sprint, across T17.2–T17.5.**

## §A6 — #198: taken, labelled, transcribed directly from its own fully-specified body

**Re-verified this ceremony** (§A2): the gap is real, unchanged, and #198's
own body — written by T16.2's own sibling sweep — already names the exact
fix shape: reuse `authorizeCompetitionEntryRecording` (built by T16.2,
already wired into `payments/app.Service`) for `authorizeOnlineCreation`'s
`PayableTypeCompetitionEntry` branch, delete
`CreateOnlinePaymentInput.EntrantPlayerID`/`AssignedCompetitionAdminUserIDs`,
mark the matching proto fields `[deprecated = true]`, and add the same
mutation-checked test shape T16.2 already used one RPC over.

**Labelled this ceremony** (§A2): `role:product-engineer` (matching #168's
own established role for this exact shape of work — resolving Payments
authorization against an already-built resolver port rather than a
caller-supplied field), `type:bug`, `context:payments`. Confirmed via a
fresh `issue_read` that the labels are applied.

**Taken as T17.1** — small, unblocked, mechanically shaped (closer to
T16.4 than to any D1/D2-blocked item), and the one ticket in this sprint
with genuinely zero new capability to build (every port and method it
needs already exists and is already wired).

## §A7 — DECISION D1 and DECISION D2 remain unanswered. D1 escalated a fifth time, with its growth stated explicitly

**Re-verified this ceremony, not assumed** (§A2): #144 carries exactly one
comment, T14.3's original escalation, unchanged across T14, T15, and T16.
ADR-0016's own `## Status` field is unchanged: "Escalated — awaiting
decision (D2)."

> **DECISION D1 (for the user / Product Owner) — fifth deferral, and its
> footprint has grown, not merely persisted.** When somebody books a court
> through the public flow *without an account* — which is how the flow
> works today — who should be allowed to cancel that booking later, and
> should booking without an account remain possible at all?
> `docs/adr/0015-booking-ownership-for-public-bookings.md` lays out four
> options with their costs and recommends none. **Stated plainly, because
> this is now the second sprint in a row it is true**: D1's absence no
> longer only blocks one named ticket — it now visibly shapes what a
> *shipped* feature is allowed to build. Two named instances, not one:
>
> 1. `CancelBooking` itself has no authorization check at all (#144's
>    original scope, five sprints old as of this ceremony).
> 2. A cancelled Game's or Competition's court-Bookings stay reserved
>    forever — T16.3 shipped the Registrations/Entries half of that
>    cascade and *deliberately stopped short* of the court-Bookings half
>    specifically because releasing them would call
>    `app.Service.CancelBooking`, whose signature D1 may still change
>    (adding an actor parameter is one of D1's four options). This is not
>    a ticket waiting on D1; it is a real, shipped capability with a
>    permanent gap in it, and the gap exists *because of* D1, not merely
>    alongside it.
>
> No T17 ticket implements this, and per ADR-0015's restriction list no
> T17 PR may guess at it.

> **DECISION D2 (for the user) — second deferral, third sprint open.**
> When the same session both reviews and merges a pull request, may it
> also write the code on that pull request — and if so, under exactly
> what limits? `docs/adr/0016-reviewer-authored-code-on-a-reviewed-pull-
> request.md` lays out four options, including a fully-specified carve-out
> the user can approve directly by naming it, and recommends none. T16
> shipped no reviewer-authored gap-fix to test the interim rule against
> (T16 retro finding 7) — its second consecutive sprint with nothing to
> score either way, not a third deferral of the same shape as D1's.

**Neither is implemented, decided, or guessed at by this ceremony.** Both
remain exactly as blocked as `sprint-process.md`'s own restriction lists
require.

## §A8 — The whole open backlog, ranked, with a disposition for each

All 11 open issues. "Taken" means a T17 ticket owns it; every deferral
carries its reason, per the board-of-record rule that a deferral without a
written reason is a process violation.

| Issue | Ranked | Disposition |
|---|---|---|
| **#195** FK-race class, nine write paths, four contexts | **Taken, in full** | **T17.2–T17.5** — one ticket per writing context, per the fired scoring condition (§A5) |
| **#198** `CreateOnlinePayment` still trusts caller-supplied entrant/admin facts | **Taken** | **T17.1** — small, mechanical, fully specified in its own body (§A6) |
| **#144** `CancelBooking` has no authz | **Escalated** | D1 unanswered; §A7. Fifth sprint carrying this, footprint growth stated explicitly rather than only re-asserted |
| **#149** Payments' remaining caller-supplied fact (`booking_host_id`) | **Untouched, correctly** | Its one remaining fact is blocked on D1 exactly as its own text says; no T17 ticket can resolve it without D1 |
| **#124** court-Bookings half of the cascade | **Untouched, correctly** | Deferred per its own recorded comment (T16.3, §A2) — blocked on D1 the same way #149 is |
| **#164** ADR-0014 actor-column conformance | **Deferred, still blocked** | Needs a real IdP tenant, unreachable from this environment; unchanged since T14 |
| **#145** pre-existing UUID rows vs. `Principal.Subject` | **Deferred** | Same real-IdP-tenant blocker as #164; unchanged |
| **#126** real per-Game price field | **Deferred** | Needs Product Owner input on pricing shape before any code; unchanged since T8.10 |
| **#130** refunding a `no_show_fee` | **Deferred** | Carries a live product question (whether reversing a no-show fee is the wanted behaviour), not just an engineering one; unchanged |
| **#134** WCAG manual screen-reader pass | **Deferred** | Needs a `role:ux-ui-designer` pass with real assistive tech, which this environment cannot provide; T17 ships no UI change |
| **#167** Stripe webhook receiver | **Deferred** | A payments-architecture ticket, not a bug fix; would compete for the same Payments-context reviewer attention T17.1 already claims this sprint, and carries no urgency signal (#198 does, since it is a live authorization gap; #167 is a design improvement) |

**No new issue opened this ceremony.** Unlike T16's Ceremony 1 (which
found and filed #195, and corrected #125), this ceremony's own backlog
re-verification (§A2) found no new stale issue and no new undisclosed gap
— stated explicitly rather than left to be inferred from silence, per
`sprint-process.md`'s "silence is indistinguishable from not having
checked" standard.

## §A9 — Dependency-completeness check, applied per `sprint-process.md`'s T16-amended shape (both questions, every ticket)

**T17.1 — closing #198.**

1. *Does the producer's capability exist?* **Yes, already** — T16.2 built
   and wired `internal/payments/port.EntryLookup`
   (`CompetitionIDAndPlayerIDForEntry`) and
   `internal/payments/port.CompetitionAdminReader`
   (`ListCompetitionAdmins`) into `payments/app.Service`, consumed today by
   `authorizeCompetitionEntryRecording`. Confirmed by reading
   `internal/payments/app/service.go` directly this ceremony (§A2).
2. *Does the consumer's own input actually contain the join key?* **Yes,
   already** — `CreateOnlinePaymentInput.PayableID` is the same
   CompetitionEntry id `RecordOfflinePaymentInput.PayableID` is (both
   RPCs' inputs share the same `PayableID`/`PayableType` shape); confirmed
   by reading `internal/payments/app/service.go`'s `CreateOnlinePaymentInput`
   struct directly.

**Required, transcribed directly from #198's own body (recommendation 4,
§A4), not re-derived**: extend or add an `authorizeOnlineCreation` branch
for `PayableTypeCompetitionEntry` that calls
`s.entryLookup.CompetitionIDAndPlayerIDForEntry` +
`s.competitionAdminReader.ListCompetitionAdmins` — reuse
`authorizeCompetitionEntryRecording` outright if its signature fits, rather
than writing a second authorization rule for the identical concept, T16.4's
own precedent for `RefundPayment` reusing `authorizeOfflineRecording`.

**T17.2–T17.5 — #195's four per-context tickets.**

1. *Does the producer's capability exist?* **N/A in the read-capability
   sense — this is not a missing read, it is a missing error
   translation.** Each of the nine FK paths already has an app-level read
   that guards it in the non-racing case (transcribed into the table in
   §A5, directly from #195's own body); what does not exist is a mapping
   from the Postgres `23503` code the insert would raise on a race to a
   domain sentinel at each context's own Postgres adapter boundary.
   Confirmed absent by reading each context's `adapter/postgres/
   repository.go` and `translate_test.go` this ceremony (§A2) — none maps
   any of the nine constraint names named in #195's own table.
2. *Does the consumer's own input contain the join key?* **N/A in the
   ID-shaped sense, named explicitly so this check is not stretched to
   cover a shape it does not fit** — same reasoning T16.3's own dependency
   check used for its bulk-cancel write capability (`t16-sprint-plan.md`
   §A10): the gap here is a missing *translation*, not a missing *join
   key*. Every one of the nine inserts already has the foreign id it needs
   (it is the insert's own parameter); the question this check exists to
   ask does not apply to a translation-only fix the same way it applies to
   a new cross-context read.

**Required, transcribed directly from #195's own body (recommendation 4,
§A4) and from #185/T15.6's own template**, per ticket:

- **T17.2 (Social Play)**: map the `registrations_game_id_fkey` and
  `games_venue_facility_id_fkey` constraint names (or whatever `\d`
  reports their actual names as — confirm against the live schema, do not
  assume the naming convention) to `socialplay.domain.ErrGameNotFound` and
  a reused-or-new `ErrFacilityNotFound`-shaped sentinel respectively, in
  `internal/socialplay/adapter/postgres/repository.go`'s `translateErr`.
  Update `internal/socialplay/adapter/grpcapi`'s `toStatus` exhaustiveness
  table (the T14.7 guard fires if this is missed). Prove each with a
  `//go:build integration` test forcing the race (insert while the parent
  row is concurrently deleted) — compiled via `make vet-integration`, not
  executed, per this repo's standing no-Docker-daemon gap.
- **T17.3 (Competitions)**: identical shape, for
  `competition_entries_competition_id_fkey` and
  `competitions_venue_facility_id_fkey`, in
  `internal/competitions/adapter/postgres/repository.go`.
- **T17.4 (Facilities)**: identical shape, for
  `discount_rules_facility_id_fkey`, `courts_facility_id_fkey`, and
  `facility_camera_links_facility_id_fkey` — all three reuse
  `internal/facilities/domain.ErrFacilityNotFound` (already exists, per
  §A2's grep), in `internal/facilities/adapter/postgres/repository.go`.
- **T17.5 (Booking)**: identical shape, for
  `recurring_hire_templates_court_id_fkey` (reuse a Court-not-found
  sentinel) and `recurring_hire_templates_requested_by_user_id_fkey`
  (reuse an Identity-lookup-not-found sentinel — check what
  `internal/booking/adapter/facilities/lookup.go`'s Identity-side
  equivalent already returns before inventing a new one), in
  `internal/booking/adapter/postgres/repository.go`.

**Explicitly not required, stated so it is not assumed by omission**: no
new migration in any of the four tickets (§A5) — the FK constraints
already exist; this is translation-only.

## §A10 — Shared-file pre-assignment, and same-wave verification per §A4's new rule

| Artifact | Owner | Notes |
|---|---|---|
| `internal/payments/app/service.go`, `internal/payments/app/refund_test.go`, `proto/pickleball/payments/v1/payments.proto` | **T17.1 only** | No other T17 ticket touches Payments |
| `internal/socialplay/adapter/postgres/repository.go`, `internal/socialplay/domain/errors.go`, `internal/socialplay/adapter/grpcapi/*` (toStatus table only) | **T17.2 only** | |
| `internal/competitions/adapter/postgres/repository.go`, `internal/competitions/domain/errors.go`, `internal/competitions/adapter/grpcapi/*` (toStatus table only) | **T17.3 only** | |
| `internal/facilities/adapter/postgres/repository.go`, `internal/facilities/domain/errors.go`, `internal/facilities/adapter/grpcapi/*` (toStatus table only) | **T17.4 only** | |
| `internal/booking/adapter/postgres/repository.go`, `internal/booking/domain/errors.go`, `internal/booking/adapter/grpcapi/*` (toStatus table only) | **T17.5 only** | |
| **`HANDOFF.md`** | **this ceremony only** | An implementer that finds a stale line flags it for T18's Ceremony 1 and does not edit it — the standing rule, unchanged |
| **`docs/process/sprint-process.md`** | **this ceremony only** | No T17 execution ticket touches process |

**§A4's new same-wave rule, applied to this sprint's own wave design and
found not to fire**: all five T17 tickets touch **disjoint** files in
**disjoint** contexts, and none widens a shared interface another T17
ticket implements — unlike T16.2/T16.3's shared
`internal/competitions/port.Repository`, no T17 ticket adds a method to an
interface any other T17 ticket implements a new instance of. **No ticket
text needs the advance-flagging §A4 requires**, and this is stated
explicitly rather than left to be inferred from the absence of a flag
(same "silence is indistinguishable from not having checked" standard
§A8's closing line applies). All five tickets are therefore cleared for
concurrent Wave 1 dispatch with no interface-blast-radius risk between
them.

---

# Ceremony 2 — Sprint planning

## Sprint goal

> **Two disclosed, unblocked gaps get closed for real, rather than a third
> and fourth sprint of tracking them without acting.** `CreateOnlinePayment`
> stops trusting a caller's own claim about who a competition entrant or
> admin is, reusing the exact resolver ports T16.2 already built for the
> sibling RPCs (**#198**) — the online/offline authorization asymmetry
> Payments has carried since T15.5 closes completely. And the FK-race
> defect class T15.6 first disclosed and T16 tracked-but-did-not-fix
> (**#195**) gets its full nine-path, four-context translation, split one
> ticket per writing context so each stays mechanical and file-disjoint —
> because the sprint-plan condition written specifically to prevent a third
> silent deferral has now fired. On process: this ceremony's own two
> `sprint-process.md` amendments — the same-wave shared-interface
> verification rule (the direct fix for the one real defect that has ever
> reached this project's shared branch tip) and the dependency-completeness
> check's transcription clause — land directly rather than waiting for a
> ticket. D1 and D2 go back to the user unanswered, D1's growing footprint
> stated plainly rather than only re-asserted.

**What this sprint does not claim** (the half PM insists on):

- **#149's `booking_host_id` fact and #124's court-Bookings half remain
  exactly as blocked as before.** Neither is touched by any T17 ticket;
  both are blocked on D1, and D1's absence now shapes two named, separate
  pieces of shipped-or-shippable scope, not one — stated in full in §A7,
  not softened here.
- **D1 and D2 remain unanswered.** No T17 ticket implements `CancelBooking`
  authorization or a reviewer-authorship carve-out; none may guess either.
- **#195's fix is translation-only, not a new invariant.** T17.2–T17.5 map
  an existing Postgres constraint violation to a clean domain error; they
  do not add a new constraint, and the narrow race window itself (a
  concurrent delete of the parent row) is unchanged by this sprint — what
  changes is what the caller sees when it happens (a clean 4xx instead of
  an unclassified 500), the identical shape #185 already established for
  `bookings.court_id`.
- **#164/#145/#134/#126/#130/#167 are untouched**, each with a reason
  recorded in §A8.
- **This sprint (5 tickets, 17 points) is close in size to T16's (3
  tickets, 16 points), not a return to T15-era scope.** The increase from
  T16 is #195's fired scoring condition, not a general loosening of the
  "take only what's genuinely unblocked" discipline T16 established —
  every other item in the backlog remains blocked exactly as §A8 states.

## Tickets — 5 items, 17 points

### T17.1 — `CreateOnlinePayment` resolves its competition-entry facts instead of trusting the caller (closes #198)

- **Story:** As a Player entering a Competition online, I want the payment
  I create to be authorized against who I actually am and who is actually
  the entrant/admin on record, so that the online path can't be satisfied
  by a caller simply naming themselves the entrant — the same guarantee
  T16.2 already built for the offline path.
- **Points:** 3 · **Role:** `role:product-engineer` · **Type:** `type:bug`
- **Description:** Closes **#198** (filed by T16.2's own sibling sweep,
  §A6). `authorizeOnlineCreation`'s `PayableTypeCompetitionEntry` branch
  still compares the verified actor against caller-supplied
  `EntrantPlayerID`/`AssignedCompetitionAdminUserIDs`; T16.2 already built
  and wired the exact resolver ports (`EntryLookup`,
  `CompetitionAdminReader`) that would answer this identically for
  `CreateOnlinePayment`, one RPC over from where they're already used.

**Instructions**

1. **Extend or add an `authorizeOnlineCreation` branch for
   `PayableTypeCompetitionEntry`** that calls
   `s.entryLookup.CompetitionIDAndPlayerIDForEntry` and
   `s.competitionAdminReader.ListCompetitionAdmins` — both already
   constructed on `payments/app.Service` since T16.2. **Prefer reusing
   `authorizeCompetitionEntryRecording` outright** (T16.2's own method) if
   its signature already fits the online-creation call site; only write a
   second method if a real signature mismatch forces it, and state which
   happened in the PR.
2. **Delete `CreateOnlinePaymentInput.EntrantPlayerID` and
   `.AssignedCompetitionAdminUserIDs`** from the Go struct — the
   T14.5/T15.5/T16.2 pattern: deleted, not left unread, so a future
   re-plumb of the wire list fails to compile rather than silently
   restoring a forgeable check.
3. **Mark `CreateOnlinePaymentRequest.entrant_player_id` and
   `.assigned_competition_admin_user_ids` `[deprecated = true]`** on the
   proto, mirroring T16.2's identical treatment of the four
   `RecordOfflinePaymentRequest`/`RefundPaymentRequest` fields it
   deprecated. Regenerate.
4. **Close #198 after merge**, per DoD step 5, with a comment naming this
   PR.
5. **Non-functional:** TDD-first. Mutation-checked headline tests, named,
   one RPC over from T16.2's own shape: a caller naming themselves via the
   now-deleted-from-input, still-on-the-wire deprecated fields is refused;
   a genuine entrant succeeds; a genuine Competition Admin succeeds and is
   refused again once revoked (positive/negative pair, same admin-assign-
   then-revoke shape T16.2's `TestRecordOfflinePayment_
   RealCompetitionsSeam_EntrantSucceeds_AdminAssignThenRevoke` already
   established — reuse its test infrastructure rather than inventing a
   parallel harness). Drive the boundary test through the real
   `competitions.app.Service`, not a fake that returns whatever it's told
   (T14.8/T15.5's standing cross-context-fake-trap warning).
6. **Sibling sweep, reported either way:** any other Payments
   authorization branch still comparing against a caller-supplied fact
   that this ticket's resolvers (or T16.2's) could also answer? (From
   inspection at planning time: no — `booking_host_id` is the only
   remaining caller-supplied fact anywhere in Payments, and it is blocked
   on D1, not on a missing resolver; confirm this is still true at
   implementation time rather than trusting the plan's own inspection —
   T16.2's identical plan-time inspection missed #198 itself.)

### T17.2 — Social Play: translate FK violations on `registrations.game_id` and `games.venue_facility_id` (part of #195)

- **Story:** As a caller of Social Play's write RPCs, I want a
  concurrent-delete race against a Game or a Facility to surface as a
  clean, typed error rather than an unclassified `Internal` (500), so that
  a legitimate (if racy) client-visible condition doesn't read as a server
  bug.
- **Points:** 3 · **Role:** `role:product-engineer` · **Type:** `type:bug`
- **Description:** One of four context-scoped tickets closing **#195**
  together (§A5) — the FK-race class T15.6 (PR #191) disclosed and T16
  tracked-but-did-not-fix, mirroring #185's own already-shipped fix
  template exactly.

**Instructions**

1. **Map `registrations.game_id`'s and `games.venue_facility_id`'s FK
   violation (`23503`) to domain sentinels** in
   `internal/socialplay/adapter/postgres/repository.go`'s error
   translation, mirroring `translate_test.go`'s existing pattern for
   whatever FK this context already translates (if any — check first).
   Confirm the actual Postgres constraint names against the live schema
   (`\d registrations`, `\d games`) rather than assuming a naming
   convention. Reuse an existing not-found sentinel per parent type
   (`ErrGameNotFound`-shaped for `registrations.game_id`; a
   `ErrFacilityNotFound`-shaped sentinel, defined in this context or
   reused via its existing Facilities-lookup port, for
   `games.venue_facility_id`) rather than inventing a new one — the
   caller-visible fact in the race case is identical to the non-racing
   "parent doesn't exist" case, per #195's own reasoning.
2. **Update `internal/socialplay/adapter/grpcapi`'s `toStatus`
   exhaustiveness table** to map the (re-used or new) sentinel — the T14.7
   guard fails the build if this is missed.
3. **Prove it with a `//go:build integration` test per FK path** (2
   total): insert a row referencing a parent, delete the parent
   concurrently in the narrow window between the guarding read and the
   insert, assert the insert now returns the mapped domain error rather
   than an unclassified `23503`. An in-memory fake cannot reproduce this
   — it does not model the race (the same fixture-infidelity trap #185 was
   caught in twice, per #195's own body). **Not run** in this environment
   (no Docker daemon) — compiled via `make vet-integration`, matching
   every other integration test in this repo.
4. **Do not add a new migration.** The FK constraints already exist; this
   is translation-only.
5. **Non-functional:** TDD-first. Mutation-check: temporarily remove the
   new mapping, confirm the integration test (or, if compile-only, a
   direct-against-a-local-Postgres manual run per T4/T6.5's own documented
   fallback methodology) now fails with the wrong status code, restore it.

### T17.3 — Competitions: translate FK violations on `competition_entries.competition_id` and `competitions.venue_facility_id` (part of #195)

- **Story:** As a caller of Competitions' write RPCs, I want the identical
  guarantee T17.2 gives Social Play — a race against a Competition or a
  Facility surfaces as a clean, typed error, not a 500.
- **Points:** 3 · **Role:** `role:product-engineer` · **Type:** `type:bug`
- **Description:** Second of four context-scoped tickets closing **#195**
  together (§A5). Identical shape to T17.2, mirrored into Competitions —
  this project's established precedent of shipping a Social Play mechanism
  and its Competitions mirror in the same sprint (T14.4→T15.3, T13.6→T15.4,
  and T16.3's own Game/Competition cascade mirror) applied to a fix rather
  than a feature this time.

**Instructions**

1. **Map `competition_entries.competition_id`'s and
   `competitions.venue_facility_id`'s FK violation** to domain sentinels in
   `internal/competitions/adapter/postgres/repository.go`, identical
   method to T17.2 instruction 1 — confirm actual constraint names against
   the live schema first.
2. **Update `internal/competitions/adapter/grpcapi`'s `toStatus`
   exhaustiveness table.**
3. **Prove it with a `//go:build integration` test per FK path** (2
   total), identical method to T17.2 instruction 3.
4. **Do not add a new migration.**
5. **Non-functional:** TDD-first, mutation-checked, identical method to
   T17.2 instruction 5.
6. **Sibling note, not a sweep** (this ticket's scope is fixed by #195's
   own table, not open-ended): confirm no other FK-backed write path
   exists in Competitions beyond the two #195 already names — a single
   `grep -n REFERENCES db/migrations/000{9,14}*.sql` cross-checked against
   `db/queries/competitions.sql`'s own `INSERT`s is sufficient; report the
   result either way.

### T17.4 — Facilities: translate FK violations on three `facility_id`-referencing tables (part of #195)

- **Story:** As a caller of Facilities' write RPCs, I want the identical
  guarantee T17.2/T17.3 give Social Play and Competitions — a race against
  a deleted Facility surfaces as a clean, typed error on any of the three
  child tables that reference it.
- **Points:** 5 · **Role:** `role:product-engineer` · **Type:** `type:bug`
- **Description:** Third of four context-scoped tickets closing **#195**
  together (§A5) — the largest of the four (3 FK paths, all sharing one
  parent table) but still single-context and mechanical.

**Instructions**

1. **Map `discount_rules.facility_id`'s, `courts.facility_id`'s, and
   `facility_camera_links.facility_id`'s FK violation** to
   `internal/facilities/domain.ErrFacilityNotFound` (already exists —
   confirmed via `grep` this ceremony, §A2) in
   `internal/facilities/adapter/postgres/repository.go`. All three reuse
   the **same** sentinel — do not invent three distinct ones for the same
   parent type; confirm actual constraint names against the live schema
   first, same method as T17.2/T17.3.
2. **Update `internal/facilities/adapter/grpcapi`'s `toStatus`
   exhaustiveness table** — a single sentinel already mapped for the
   non-racing `EnsureFacilityOwner`/`GetFacilityByID` case likely already
   has an entry; confirm the mapping is genuinely exercised for the FK
   case too rather than assuming the existing entry already covers it (a
   sentinel can be mapped once and still not be the value this new
   translation path actually returns if the translation function returns
   something else on the FK branch — check, don't assume).
3. **Prove it with a `//go:build integration` test per FK path** (3
   total), identical method to T17.2 instruction 3.
4. **Do not add a new migration.**
5. **Non-functional:** TDD-first, mutation-checked, identical method to
   T17.2 instruction 5, run once per FK path.

### T17.5 — Booking: translate FK violations on `recurring_hire_templates.court_id` and `.requested_by_user_id` (part of #195)

- **Story:** As a Club requesting a recurring hire, I want a race against a
  deleted Court or a deleted Identity user to surface as a clean, typed
  error rather than a 500.
- **Points:** 3 · **Role:** `role:product-engineer` · **Type:** `type:bug`
- **Description:** Fourth and last of four context-scoped tickets closing
  **#195** together (§A5). The one ticket of the four with a genuinely
  cross-context parent on one of its two rows
  (`requested_by_user_id` → `identity_users.id`) — translation still
  happens entirely inside Booking's own Postgres adapter, mapping to a
  sentinel Booking's own domain package defines, per the same
  no-cross-context-domain-import boundary every other translation in this
  codebase already respects (CLAUDE.md rule 3).

**Instructions**

1. **Map `recurring_hire_templates.court_id`'s FK violation** to a
   Court-not-found sentinel in
   `internal/booking/adapter/postgres/repository.go` — check whether
   Booking's own `domain/errors.go` already has one (it resolves Courts
   via `internal/booking/adapter/facilities/lookup.go`, so check that
   adapter's own not-found handling before inventing a second sentinel for
   the same concept) before adding a new one.
2. **Map `recurring_hire_templates.requested_by_user_id`'s FK violation**
   to an Identity-lookup-not-found sentinel — same check: Booking already
   resolves a `Club` role via an Identity lookup (per #164's own
   description of this exact call site), so read that existing call site's
   error handling first and reuse its sentinel rather than adding a
   parallel one for the same concept.
3. **Update `internal/booking/adapter/grpcapi`'s `toStatus` exhaustiveness
   table** for whichever sentinel(s) are new.
4. **Prove it with a `//go:build integration` test per FK path** (2
   total), identical method to T17.2 instruction 3.
5. **Do not add a new migration.**
6. **Non-functional:** TDD-first, mutation-checked, identical method to
   T17.2 instruction 5.
7. **After all four of T17.2–T17.5 have merged** (this ticket's own merge
   may not be the sprint's last — check the actual merge order before
   acting): **close #195**, per DoD step 5, with a comment naming all four
   resolving PRs. **Do not close #195 from any single one of the four
   tickets individually** — each closes only its own context's rows, and
   #195 names all nine; closing it prematurely from one PR would misdescribe
   the codebase for however long the remaining tickets take to land. If
   this ticket's own PR happens to merge before all three siblings have,
   its review states that explicitly and defers the close to whichever
   PR merges last (mirroring PR #192's own "will close #147 per instruction
   5" pattern from T15) — or to the merged-fix sweep at sprint end if no
   single PR review ends up positioned to do it.

## Waves

**Wave 1 — no in-sprint dependencies, no shared-interface blast radius (5
tickets, 17 points)**
`T17.1` (#198) · `T17.2` (Social Play) · `T17.3` (Competitions) ·
`T17.4` (Facilities) · `T17.5` (Booking)

All five touch disjoint files in disjoint contexts (§A10); none widens an
interface any other T17 ticket implements. **No Wave-1.5 checkpoint** —
the condition (a new cross-cutting decision with three or more first-time
in-sprint consumers) does not fire; every ticket's own fix is local to its
own context.

## Recorded disagreements (Ceremony 2 rule 3 — not smoothed over)

1. **PE vs. PM — #195's scope, resolved by the fired condition rather than
   re-argued.** Full text in §A5. PM's residual concern about sprint size
   and unchanged real-world severity is recorded, not smoothed over; PE's
   position governs because the scoring condition T16's own plan wrote for
   exactly this situation has fired, and re-litigating severity from
   scratch would be the "re-argued from scratch" the condition's own text
   was written to foreclose.

## Sprint-level Definition of Done

All of `sprint-process.md`'s standing DoD, plus the scorings T17 owes,
stated now so they are not improvised at the retro:

1. All 5 tickets merged per the per-ticket DoD; sprint goal met or
   explicitly descoped with reasoning recorded.
2. **The merged-fix issue sweep run and reported with its count** — by the
   retro (reporting, not blocking) and again by T18's Ceremony 1
   (authoritative).
3. **Scoring owed at the retro:**
   - **(a)** Did T17.1 actually close #198, scored directly against the
     merged code (does `authorizeOnlineCreation` now resolve rather than
     trust the entrant/admin facts), not against the PR's own account?
   - **(b)** Did all four of T17.2–T17.5 actually ship (nine FK paths, all
     mapped, all with an exhaustiveness-table entry, all integration-test
     compiled), and was #195 closed by exactly one of them citing all four
     PRs — not zero, and not more than one claiming it?
   - **(c)** Did the new same-wave shared-interface verification rule
     (§A4) get exercised this sprint at all? (Scoreable only if a same-wave
     pair with a genuine blast-radius overlap actually occurred — §A10
     found none by design, so the honest answer may be "not exercised,
     zero opportunities," which is not a finding against the rule any more
     than ADR-0016's interim rule not being exercised was a finding against
     it in a sprint with no reviewer-authored gap-fix.)
   - **(d)** Was the transcription clause (§A4) actually followed for
     T17.1 and T17.2–T17.5 — did their Instructions match what #198's and
     #195's own bodies already specified, rather than silently re-deriving
     it?
4. **Not scoreable by T17 and deliberately not pre-empted:** D1 and D2
   remain the user's. If either is answered mid-sprint, the answer's own
   trigger takes over and T17's plan does not constrain it.
5. Retro in `docs/process/t17-retro.md`, indexed by a `## T17 sprint
   retro` stub in `docs/LESSONS.md`. `HANDOFF.md`/`CLAUDE.md` state
   updated — noting that **T18's Ceremony 1**, not the retro, corrects
   T17's Docs-index row (the ordinary convention; T16's retro's own
   self-correction was an explicitly-argued one-off per that ceremony's
   own instruction, not a new standing practice).
