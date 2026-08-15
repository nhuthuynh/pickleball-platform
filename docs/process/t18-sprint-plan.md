# T18 Sprint Plan — Ceremonies 1 + 2

Backlog refinement (Ceremony 1) and sprint planning (Ceremony 2) per
`docs/process/sprint-process.md`, six-role team (briefs:
`docs/agent-operating-handbook.md` Part B). Held against
`docs/process/t17-retro.md` (PR #208, merged 18:21:38Z), `HANDOFF.md`
including its Docs index, `CLAUDE.md`, and the live PR/issue state of
`nhuthuynh/white-label` (GitHub-side name `pickleball-platform`).

**Every factual claim below was re-derived against the repository and the
GitHub API during this ceremony**, at the merged branch tip (`5f50ff8`, T17
retro's own merge), rather than taken from the retro's or T17's plan's
prose — CLAUDE.md rule 10 applied to planning. Where this ceremony repeats a
claim it could not independently re-check, it says so.

## §A0 — Correcting T17's Docs-index row and Task-backlog narrative: this
ceremony's first job, per the retro's own instruction

Per `sprint-process.md`'s standing convention (a retro PR never updates the
Docs-index row that points at it, since that row must cite the retro's own
merge PR number, which does not exist until it merges) and per
`docs/process/t17-retro.md`'s own closing line — *"T18's Ceremony 1 corrects
T17's row, including its real PR merge order and the honest-form sentence
above, as its first job"* — **performed directly in `HANDOFF.md` by this
same PR**, not summarized twice here. What was corrected, and how each fact
was independently re-verified rather than copied from the retro's prose:

| Field | What the row now says | Independently re-verified this ceremony |
|---|---|---|
| Retro link | `docs/process/t17-retro.md` (5 findings; the discount_rules context-ownership miss) | File exists, read in full this ceremony; finding count (5, not folded into a generic "no gaps") matches its own content |
| Reviews cell | PRs #202 (Ceremony 1/2 doc) → #203 (T17.2) → #204 (T17.5) → #205 (T17.3) → #206 (T17.1) → #207 (T17.4), in that merge order | Re-fetched all six via `list_pull_requests`: `merged_at` **17:38:05Z → 17:54:25Z → 17:57:57Z → 18:00:49Z → 18:03:32Z → 18:09:51Z** — ascending, matches the retro's stated order exactly, not assumed from PR numbering (which happens to agree here, but was checked, not presumed) |
| Task-backlog narrative | The retro's own agreed sentence, quoted verbatim from `docs/process/t17-retro.md`'s "The sprint goal, scored" section, per `sprint-process.md` Ceremony 1 item 3 ("the retro's form, not a stronger one") | Cross-checked word-for-word against the retro's own blockquote — copied, not paraphrased |
| Retro's own PR | #208, `merged_at` **18:21:38Z** | Fetched via `list_pull_requests`; its head commit (`3b8ad07`) is the parent of the current tip (`5f50ff8`) per `git log` |

**A new T18 row is added** in the same before-the-sprint form, Retro and
Reviews honestly marked "not yet written" / "not yet opened", for T19's
Ceremony 1 to correct in turn — the same convention every prior ceremony has
followed.

## §A1 — The merged-fix issue sweep, run as this ceremony's first substantive
act (per DoD, T18's Ceremony 1 is the authoritative moment)

**Step 1 — list the open issues, live.** `list_issues(state: OPEN)` at
ceremony start → **`totalCount: 9`**: #124, #126, #130, #134, #144, #145,
#149, #164, #167. This is the exact list the CLAUDE-level task handed this
ceremony predicted from `docs/process/t17-retro.md`'s own sweep — re-fetched
live rather than trusted, per the standing rule that a prior ceremony's clean
result does not discharge the next one.

**Step 2 — reconcile arithmetically, before reading any issue
individually.** `open_at_end_of_T17's_own_Ceremony_1 − closed_during_T17 +
opened_during_T17`. T17's own Ceremony 1 (§A2 there) left the count at
**11**, with zero new issues opened at that ceremony. During T17's
execution: **#198** closed (T17.1, PR #206) and **#195** closed (T17.4, PR
#207, on behalf of all four T17.2–T17.5 tickets) — `11 − 2 = 9`. **Matches
the live `totalCount: 9` read at this ceremony's start exactly.**

**Step 3 — cross-reference merged PRs against the open list, checking
attribution against the merged code, not the closing comments' prose.**

| Issue | `state` (re-fetched now) | `state_reason` | `closed_at` | Closed by / cites |
|---|---|---|---|---|
| #198 | `closed` | `completed` | 2026-08-15T18:03:39Z | Comment on the issue naming PR #206 |
| #195 | `closed` | `completed` | 2026-08-15T18:09:56Z | Comment on the issue naming all four resolving PRs (#203, #204, #205, #207) |

Both closes were also cross-checked against their resolving PRs' own
`merged_at` timestamps: PR #206 (T17.1) `merged_at` 18:03:32Z — 7 seconds
before #198's close; PR #207 (T17.4) `merged_at` 18:09:51Z — 5 seconds before
#195's close. Both closes are genuine, both are correctly attributed, and
both were performed by the party that merged (per-PR optional-early-close,
mandatory for "closes #N" titles per DoD step 5) — consistent with T17
retro's own account, now independently re-derived rather than trusted.

**No new issue has been opened or closed since T17's retro** — confirmed:
this ceremony's own start is the same tree the retro already swept, and
nothing has merged in between (`git log` shows `5f50ff8`, the retro's own
commit, as the current tip with no descendants).

**Sweep result: clean, third sprint running.** Zero unclosed hits, zero
mis-attributed closes, arithmetic reconciles exactly. **T19's Ceremony 1
still re-runs this sweep in full**, per the standing rule that a prior
ceremony's clean result does not discharge the next one.

## §A2 — T17 retro's four recommendations, each given a disposition

| # | Recommendation | Disposition |
|---|---|---|
| 1 | When a ticket's scope names a specific database table, verify that table's owning bounded context against its migration file's own header comment before finalizing the ticket's target file path | **Applied, for real, to this sprint's only ticket** — §A4 below performs the check live against `db/migrations/0005_payments.sql`'s header comment for the `payments` table T18.1 touches, and the check's absence-of-bite is stated explicitly (T18.1 introduces one brand-new table it does not need to attribute to a *prior* migration) rather than silently skipped |
| 2 | Continue treating the merged-fix sweep as authoritative regardless of the prior retro's clean result | **Executed here** (§A1) — independently re-derived the open count and re-verified #198's and #195's closes from the API rather than the retro's own table |
| 3 | When a fix spans multiple independent PRs closing one issue together, require every non-last PR's review to state it is deferring the close and why, and the last to state its intent before merging | **Not exercised this sprint, by design** — T18 has exactly one ticket resolving exactly one issue via exactly one PR; there is no multi-PR close to require the pattern of. Stated explicitly rather than left to be inferred from silence, per this project's own "silence is indistinguishable from not having checked" standard. If a future sprint's ticket split produces a multi-PR close, that ticket's own text must apply the pattern by name |
| 4 | D1 and D2 stay with the user; no T18 ticket implements `CancelBooking` authorization or a reviewer-authorship carve-out | **Executed here** (§A7) — neither implemented nor guessed at |

## §A3 — What this ceremony verified, and how

| Claim | How it was checked | Result |
|---|---|---|
| Open-issue count at ceremony start | live `list_issues(state: OPEN)` | `totalCount: 9`, matching #124/#126/#130/#134/#144/#145/#149/#164/#167 exactly |
| #198, #195 genuinely closed, correctly attributed | `issue_read(get)` on each, cross-checked against `list_pull_requests` on #206/#207 | both `state: closed`, `state_reason: completed`; closed within seconds of their resolving PR's `merged_at` |
| #167's own gap still exists unfixed | read `internal/payments/adapter/grpcapi/authenticated.go` in full | `PublicMethods()` still returns `nil`; its own doc comment (lines 82–87) still names "a future public RPC — a Stripe webhook receiver, say" as the shape that would close this, unbuilt |
| `ConfirmOnlinePayment`'s capture path is still client-driven only | read `internal/payments/app/service.go`'s `ConfirmOnlinePayment` (lines 305–333) | confirmed: no code path calls `s.processor.CapturePayment` except from this one client-invoked method |
| `port.Repository` has no lookup by `StripeReference` | read `internal/payments/port/repository.go` in full | confirmed: only `Create`, `GetByID`, `Update` — no `GetByStripeReference`-shaped method exists |
| No existing idempotency/webhook-event port or table exists anywhere in Payments | `grep -rn "idempot\|webhook\|Webhook" internal/payments`, `ls db/migrations` | zero hits for idempotency/webhook in `internal/payments`; no migration past `0021` |
| `domain.Payment.MarkPaid` rejects a second call once already paid | read `internal/payments/domain/payment.go`'s `MarkPaid` (line 140) and `payment_test.go`'s illegal-transition cases | confirmed: `MarkPaid` returns `ErrIllegalStatusTransition` off a non-`unpaid` status — a webhook-driven capture racing an already-completed client-driven capture must be guarded before calling it a second time, not left to error out uncontrolled |
| Next free migration / ADR numbers | `ls db/migrations`, `ls docs/adr` | `0021` last migration, `0016` last ADR → **`0022`**/**`0017`** free (0022 consumed by this sprint's one ticket, §A6; no ADR needed) |
| `payments` table's owning context, per its own migration header | `head -6 db/migrations/0005_payments.sql` (T17 retro recommendation 1, applied for real, §A4) | "Payments context schema (T6.4)" — confirmed Payments-owned, no cross-context misattribution risk on this ticket's only touched pre-existing table |
| D1 (#144) still has exactly one comment | `issue_read(get_comments)` | confirmed — T14.3's original escalation only, unchanged across T14 through T17, now T18 |
| D2 (ADR-0016) still unanswered | read `docs/adr/0016-*.md`'s `## Status` | unchanged: "Escalated — awaiting decision (D2)" |
| #124's court-Bookings half still deferred, comment present | `issue_read(get_comments)` on #124 | confirmed — T16.3's comment states which half shipped and which remains (court-Bookings, D1-blocked); issue left `state: open` |
| `make generate && make test-domain && make gate-coverage` still green on the unmodified tree | ran directly | `make test-domain`: all 12 packages green; `make gate-coverage`: `OK — all 41 package(s) … executed by "ci-checks"` |
| Local worktree matches the shared branch's real tip | `git status`, `git log --oneline -3` | clean, up to date with `origin/claude/go-backend-pickleball-7up34j`, HEAD at `5f50ff8` (T17 retro's own merge) before this ceremony's branch was cut |

**Not re-verified by this ceremony, and named as such:** the full `make
test`/`ci-integration` Docker path (no Docker daemon reachable here, same
standing gap as every prior sprint); any claim about what Jenkins would run
(no Jenkins job exists).

---

# Ceremony 1 — Backlog refinement

## §A4 — Migration-header-ownership check, applied for real (T17 retro
recommendation 1)

T18.1 (below) is the only T18 ticket that names a database table, and it
names two: the pre-existing `payments` table (whose `stripe_reference`
column it reads via a new lookup method) and a brand-new
`payments_webhook_events` table it creates.

**Pre-existing table, checked against its own migration header — not
assumed Payments-owned from the ticket's own framing:**

```
-- Payments context schema (T6.4): a single `payments` table records money
-- for any payable action (booking, registration, or no_show_fee today —
```

`db/migrations/0005_payments.sql`'s first line settles it: `payments` is
Payments-owned, unambiguously, and the ticket's target file path
(`internal/payments/port/repository.go` + `internal/payments/adapter/
postgres/repository.go`) is correctly scoped. **This is the check
performing its job on the easy case** (a table already inside the same
context the ticket's issue names) rather than skipped because it seemed
obvious — the T17 finding this check exists to prevent (`discount_rules`
misattributed to `facilities` instead of `booking`) looked just as obvious
at the time it was made.

**Brand-new table**: `payments_webhook_events` does not exist yet, so there
is no prior migration header to check against — the ownership question this
check exists to catch (a fact *carried forward* from an earlier artifact and
never re-verified) does not arise for a table this same ticket is creating.
Stated explicitly rather than silently omitted, per this project's own
"silence is indistinguishable from not having checked" standard. The new
migration's own header comment (§A6, instruction 3) states its owning
context in the same form `0017_booking_discount_rules.sql`'s header already
does, so a *future* ticket that reads it gets the correct answer on the
first read — closing the loop this check exists to protect, not just
passing it once.

## §A5 — The whole open backlog, ranked, with a disposition for each

All 9 open issues. "Taken" means a T18 ticket owns it; every deferral
carries its reason, per the board-of-record rule that a deferral without a
written reason is a process violation.

| Issue | Ranked | Disposition |
|---|---|---|
| **#167** Stripe webhook receiver would remove the `ConfirmOnlinePayment` authorization question rather than answer it | **Taken** | **T18.1** — the one issue on the backlog with no product/data/infrastructure blocker, only the reviewer-bandwidth reason T16 and T17 each gave for deferring it, and that reason no longer applies (§A6) |
| **#144** `CancelBooking` (and `CreateBooking`) have no authz | **Escalated** | D1 unanswered; §A7. Sixth sprint carrying this |
| **#149** Payments' remaining caller-supplied fact (`booking_host_id`) | **Untouched, correctly** | Its one remaining fact is blocked on D1 exactly as its own text says; no T18 ticket can resolve it without D1 |
| **#124** court-Bookings half of the cascade | **Untouched, correctly** | Deferred per its own recorded comment (T16.3) — blocked on D1 the same way #149 is; re-verified still open, still carrying that comment (§A3) |
| **#164** ADR-0014 actor-column conformance | **Deferred, still blocked** | Needs a real IdP tenant, unreachable from this environment; unchanged since T14 |
| **#145** pre-existing UUID rows vs. `Principal.Subject` | **Deferred** | Same real-IdP-tenant blocker as #164; unchanged |
| **#126** real per-Game price field | **Deferred** | Needs Product Owner input on whether the price is per-Game, per-Registration, or per-head before any code; unchanged since T8.10 |
| **#130** refunding a `no_show_fee` Payment | **Deferred** | Carries its own stated open product question ("confirm first that reversing a no-show fee is the product behaviour actually wanted"), not just an engineering one; unchanged |
| **#134** WCAG manual screen-reader pass | **Deferred** | Needs a `role:ux-ui-designer` pass with real assistive-technology hardware, which this environment cannot provide; T18 ships no UI change |

**No new issue opened this ceremony.** This ceremony's own backlog
re-verification (§A3) found no new stale issue and no new undisclosed gap —
stated explicitly rather than left to be inferred from silence.

**Why #167 and not a bigger sprint.** Every other open issue is blocked on
one of: DECISION D1 (3 issues — #144, #149, #124's remainder), a real IdP
tenant this environment structurally cannot provision (2 issues — #164,
#145), genuine Product Owner input on a live product question (2 issues —
#126, #130), or real assistive-technology hardware (1 issue — #134). None
of those eight blockers changed state between T17's retro and this
ceremony (§A3 re-verifies each). #167 is the one issue whose only stated
blocker across T16 and T17 was competing for the same Payments-context
reviewer attention T16.2 and T17.1 each already claimed those sprints
(`docs/process/t16-sprint-plan.md` §A9, `docs/process/t17-sprint-plan.md`
§A8) — an attention constraint, not a product/data/infra one, and it does
not recur this sprint since no other T18 ticket touches Payments.

## §A6 — T18.1: dependency-completeness check, both questions, run against
the code

**1. Does the producer's capability exist?** **No, on three separate axes —
named individually rather than bundled into one generic "build it":**

- **No lookup from a Stripe payment-intent reference to the owning
  `domain.Payment`.** `internal/payments/port.Repository` has exactly three
  methods (`Create`, `GetByID`, `Update`, §A3) — none keyed on
  `StripeReference`. A webhook event carries Stripe's own intent reference,
  not this backend's internal Payment id, so this is a **required new
  method**: `GetByStripeReference(ctx context.Context, ref string)
  (domain.Payment, error)`, mirroring `GetByID`'s existing shape exactly,
  implemented in both the Postgres adapter (`SELECT … WHERE
  stripe_reference = $1`) and any in-memory test fakes.
- **No signature-verification capability anywhere in this context.**
  `grep -rn "Verif" internal/payments/port` returns nothing. A new port,
  `port.WebhookVerifier` (one method,
  `VerifySignature(rawPayload []byte, signatureHeader string) error`),
  is required — mirroring `port.PaymentProcessor`'s own doc-comment pattern
  (an interface shaped so a real Stripe SDK type never crosses it, per
  CLAUDE.md rule 2) — plus a deterministic stub implementation in a new
  `internal/payments/adapter/webhookstub` package, naming-mirroring
  `stripestub` for the identical reason: obvious at a glance that this is
  not a real Stripe integration, no Stripe SDK dependency, no network call,
  no `STRIPE_*` environment variable read (§A3 confirms none exists today).
- **No idempotency ledger.** Confirmed absent (§A3). A webhook is
  redelivered by design on any ambiguous response, so a second delivery of
  the same event must be a safe no-op, not a second capture attempt. A new
  narrow port (e.g. `port.WebhookEventStore` with one atomic
  `ClaimEvent(ctx, eventID string) (claimed bool, err error)` — an `INSERT
  … ON CONFLICT (event_id) DO NOTHING`-shaped write, returning whether *this*
  call was the one that claimed it) plus a new migration (§A4, §A6
  instruction 3) are required.

**2. Does the consumer's own input contain what it needs?** **N/A in the
ID-shaped cross-context sense, named explicitly so this check is not
stretched to cover a shape it does not fit** — the same reasoning T17's own
plan used for its translation-only tickets. This is not a cross-context read
(Payments is resolving its own `Payment` from its own `StripeReference`
column); the "join key" is the Stripe event payload itself, which this
ticket's own request message must be designed to carry (instruction 1
below) rather than a fact that already exists somewhere else in the system
and merely needs plumbing.

**Required, transcribed from #167's own body and `authenticated.go`'s own
prediction (T17 retro recommendation 4's transcription discipline, kept in
force even though this is a different shape of source — a comment in the
code itself rather than a prior issue/PR):** `authenticated.go` lines 82–87
name the exact fix shape before this ticket exists: *"a future public RPC —
a Stripe webhook receiver, say, which issue #148's closing note raises as
the shape that would make the client-driven capture question moot — needs
somewhere to be declared deliberately rather than by omission."* T18.1's
instructions below transcribe that shape directly: a new RPC, declared
public, added to `PublicMethods()` by name.

## §A7 — DECISION D1 and DECISION D2 remain unanswered

**Re-verified this ceremony, not assumed** (§A3): #144 carries exactly one
comment, T14.3's original escalation, unchanged across T14 through T17, now
T18. ADR-0016's own `## Status` field is unchanged: "Escalated — awaiting
decision (D2)."

> **DECISION D1 (for the user / Product Owner) — sixth deferral.** When
> somebody books a court through the public flow *without an account* —
> which is how the flow works today — who should be allowed to cancel that
> booking later, and should booking without an account remain possible at
> all? `docs/adr/0015-booking-ownership-for-public-bookings.md` lays out
> four options with their costs and recommends none. No T18 ticket
> implements this, and per ADR-0015's restriction list no T18 PR may guess
> at it. Its footprint is unchanged from T17 retro's own re-check (finding
> 5 there): still exactly the same two named instances of shaped scope
> (`CancelBooking`'s own missing check; the court-Bookings half of #124),
> neither grown nor shrunk, because no T18 ticket touches either surface.

> **DECISION D2 (for the user) — third deferral, fourth sprint open.** When
> the same session both reviews and merges a pull request, may it also
> write the code on that pull request — and if so, under exactly what
> limits? `docs/adr/0016-reviewer-authored-code-on-a-reviewed-pull-request.md`
> lays out four options and recommends none. T17 shipped no
> reviewer-authored gap-fix to test the interim rule against — its third
> consecutive sprint with nothing to score either way (T17 retro finding
> 5). T18's own single ticket is implemented and reviewed by the ordinary
> two-role loop (§ below); nothing in this sprint's design calls for a
> reviewer to author code on a PR under its own review, so this sprint is
> not expected to produce a fourth data point either — stated as a
> prediction, not a guarantee, and the retro is asked to score it
> regardless of which way it lands.

**Neither is implemented, decided, or guessed at by this ceremony.** Both
remain exactly as blocked as `sprint-process.md`'s own restriction lists
require.

## §A8 — Shared-file pre-assignment, and same-wave verification (finds
nothing to do, by construction)

| Artifact | Owner | Notes |
|---|---|---|
| `proto/pickleball/payments/v1/payments.proto` | **T18.1 only** | No other T18 ticket touches proto |
| `internal/payments/**` (all of it — `port`, `adapter/postgres`,
  `adapter/webhookstub` (new), `adapter/grpcapi`, `app`, `domain`) | **T18.1 only** | Single-ticket sprint; no concurrent Payments work exists to collide with |
| `db/migrations/0022_*.sql` | **T18.1 only** | Next free number, confirmed (§A3); pre-assigned so no other future-dispatched ticket can claim it out from under this one |
| **`HANDOFF.md`** | **this ceremony only** | An implementer that finds a stale line flags it for T19's Ceremony 1 and does not edit it — the standing rule, unchanged |
| **`docs/process/sprint-process.md`** | **this ceremony only** | No T18 execution ticket touches process; no amendment is landed this ceremony |

**Same-wave shared-interface verification rule (§ sprint-process.md,
Execution): does not apply, trivially and statedly rather than silently.**
The rule's own precondition is *two* same-wave tickets touching one shared
interface's blast radius. T18 has exactly one ticket. There is no sibling to
collide with, so the rule has zero opportunity to fire this sprint — the
honest answer, not a finding against the rule, matching how T17's own plan
scored the identical "not exercised, zero opportunities" outcome for a
different reason (file-disjointness rather than ticket-count) the sprint
before.

---

# Ceremony 2 — Sprint planning

## Sprint goal

> **The one genuinely unblocked issue on the backlog gets built: online
> payment capture stops depending entirely on a client call.** A Stripe
> webhook receiver (**#167**) lets a successful charge complete itself —
> authenticated by a signature check rather than a verified principal, since
> Stripe itself has no login on this platform — while the existing
> client-driven `ConfirmOnlinePayment` path stays exactly as it is,
> deliberately not removed, so this ticket adds a capability rather than
> taking one away. Every other open issue stays exactly where T16 and T17
> left it, each for a restated, re-verified reason rather than a silent
> re-deferral: three are D1-blocked, two need a real IdP tenant, two need
> Product Owner input on a live product question, and one needs
> assistive-technology hardware this environment does not have. D1 and D2
> go back to the user unanswered.

**What this sprint does not claim** (the half PM insists on):

- **This does not remove the client-driven capture path.** #167's own body
  frames the webhook receiver as what *would* make the authorization
  question on `ConfirmOnlinePayment` moot — not as removing that RPC. T18.1
  adds a second, independently-authenticated completion path; it does not
  delete the first one. Removing the client-driven path is a one-way door
  (PE's mandate: flag it explicitly rather than let it pass quietly) this
  ticket does not open — if a future sprint wants to make the webhook the
  *sole* capture mechanism, that is a new, separate decision, not an
  implication of this one.
- **This is not a real Stripe integration.** `port.WebhookVerifier`'s stub
  implementation is deterministic and network-free, mirroring
  `stripestub`'s own stated boundary (no Stripe SDK, no `STRIPE_*` env var)
  — a future `internal/payments/adapter/stripe` package (already named as
  a placeholder in `stripestub`'s own doc comment) would implement both
  `port.PaymentProcessor` and `port.WebhookVerifier` for real; swapping is
  wiring-only, never an app/domain change, per that package's own stated
  design.
- **D1 and D2 remain unanswered.** No T18 ticket implements `CancelBooking`
  authorization or a reviewer-authorship carve-out; neither may be guessed
  at.
- **#144/#149/#124's remainder/#164/#145/#126/#130/#134 are untouched**,
  each with a reason recorded in §A5.
- **This sprint (1 ticket, 8 points) is the smallest yet, by design.** Not a
  loosening of scope discipline — the opposite: eight of the nine open
  issues are genuinely blocked on something this session cannot supply
  (a product decision, a real IdP tenant, assistive-tech hardware, or D1),
  and manufacturing scope to hit a bigger number would mean either guessing
  at one of those blockers or padding the sprint with busywork neither PE
  nor PdE could defend against QA's "an invariant with no test that could
  fail is untested, not proven" standard applied to scope itself.

## Tickets — 1 item, 8 points

### T18.1 — A Stripe webhook receiver lets a successful online payment complete itself, without removing the client-driven path (closes #167)

- **Story:** As a Player whose card is successfully charged by Stripe, I
  want my Payment to complete even if my own device never gets to call
  `ConfirmOnlinePayment` — a dropped connection, a closed tab, a crashed
  app — so that a real charge doesn't sit forever as an "unpaid" Payment the
  platform has no way to reconcile against what actually happened.
- **Points:** 8 · **Role:** `role:principal-engineer` · **Type:** `type:story`
- **Description:** Closes **#167** (filed by T13.7's own closing note,
  re-deferred at T16 and T17 for reviewer-bandwidth reasons that no longer
  apply this sprint, §A5). `authenticated.go`'s own doc comment has named
  this exact shape since T13.7 — a public RPC, signature-authenticated
  rather than principal-authenticated, declared deliberately in
  `PublicMethods()` (§A6).

**Instructions**

1. **Add a new RPC to `PaymentsService`**:
   `ReceiveStripeWebhookEvent(ReceiveStripeWebhookEventRequest) returns
   (ReceiveStripeWebhookEventResponse)`. Request carries, as a deliberate,
   disclosed simplification of Stripe's actual arbitrary-JSON webhook
   envelope (state this explicitly in the PR, not silently): `raw_payload
   bytes` (the exact bytes the signature covers — must survive
   proto/JSON marshalling unmutated, since HMAC verification is
   byte-exact), `signature_header string`, `event_id string`, `event_type
   string`, and `stripe_reference string` (Stripe's own intent id). A real
   Stripe integration will need to parse `event_id`/`event_type`/
   `stripe_reference` out of `raw_payload` itself rather than receive them
   as separate structured fields — name this as a known, bounded gap for
   the future real-Stripe ticket to close, the same way `stripestub`'s own
   doc comment discloses its boundary rather than hiding it. Response is
   empty (an ack). Regenerate.
2. **Add the RPC to `internal/payments/adapter/grpcapi.PublicMethods()`**,
   replacing its `return nil` with this one entry, and update that
   function's own doc comment (lines 62–87) to say the prediction it made
   has been fulfilled by this ticket, naming it. This is required, not
   optional: `authenticated_test.go`'s exhaustiveness check (§A3) will fail
   the build if the new RPC is left in neither list, which is the point —
   confirm this test actually fails before instruction 2 is applied, and
   passes after, as the mutation check for this step specifically.
3. **Add migration `0022_payments_webhook_events.sql`** (confirmed the next
   free number, §A3/§A4), with a header comment in the same form
   `0017_booking_discount_rules.sql`'s own header uses — naming the owning
   context (Payments) and this ticket (T18.1, closes #167) explicitly, so a
   future ceremony's migration-header-ownership check (T17 retro
   recommendation 1) gets the right answer in one read. Table:
   `payments_webhook_events (event_id text PRIMARY KEY, received_at
   timestamptz NOT NULL DEFAULT now())` — the primary key is the
   idempotency guard itself; no separate `EXCLUDE` or unique-index dance is
   needed since `event_id` uniqueness *is* the invariant (CLAUDE.md rule 4:
   this is a case where the Postgres primary key alone already fully
   enforces the invariant, so `domain`-side pre-checking is redundant, not
   omitted — state this reasoning in the PR rather than adding a pre-check
   that duplicates the constraint for no reason).
4. **Add `internal/payments/port.WebhookVerifier`** (`VerifySignature(raw
   []byte, signatureHeader string) error`) and
   **`internal/payments/port.WebhookEventStore`** (`ClaimEvent(ctx,
   eventID string) (claimed bool, err error)`, backed by an `INSERT INTO
   payments_webhook_events (event_id) VALUES ($1) ON CONFLICT (event_id) DO
   NOTHING`-shaped write in the Postgres adapter, returning whether *this*
   call's insert was the one that succeeded). Add
   **`internal/payments/adapter/webhookstub`** (mirrors `stripestub`'s
   naming and its doc-comment disclosure exactly — no Stripe SDK, no
   network, no `STRIPE_*` env var this ticket) implementing
   `port.WebhookVerifier` via `crypto/hmac`/`crypto/sha256` over a
   caller-supplied shared secret, constructed as
   `webhookstub.NewVerifier(secret string)` so tests can seed known-good
   and known-bad signatures deterministically.
5. **Add `internal/payments/port.Repository.GetByStripeReference(ctx,
   ref string) (domain.Payment, error)`**, mirroring `GetByID`'s existing
   shape exactly (§A6) — implement in the Postgres adapter (`SELECT … WHERE
   stripe_reference = $1`, returning `domain.ErrPaymentNotFound` on no
   rows, same sentinel `GetByID` already uses for the identical concept)
   and in any in-memory test fakes.
6. **Extract `ConfirmOnlinePayment`'s capture-then-mark-paid-then-reconcile
   sequence into a shared private method** (e.g.
   `s.captureAndMarkPaid(ctx context.Context, p domain.Payment)
   (domain.Payment, error)`, covering exactly the body currently between
   `s.processor.CapturePayment` and the two `reconcile*PaymentStatus` calls
   in `ConfirmOnlinePayment`, `internal/payments/app/service.go` lines
   ~314–332) — called by both the existing `ConfirmOnlinePayment` (after
   `authorizeOnlineConfirmation` passes, unchanged) and this ticket's new
   webhook handler (after signature verification and idempotency-claim
   pass, no principal involved). **Do not duplicate the capture/MarkPaid/
   reconcile sequence a second time** — CLAUDE.md's own DRY expectation,
   and the single source of truth this refactor buys is what makes
   instruction 7's already-paid guard correct in exactly one place instead
   of two.
7. **Add `app.Service.HandleStripeWebhookEvent(ctx,
   in HandleStripeWebhookEventInput) error`**: (a) verify the signature via
   `s.webhookVerifier.VerifySignature(in.RawPayload, in.SignatureHeader)` —
   on failure, return a new sentinel (e.g. `domain.
   ErrWebhookSignatureInvalid`), mapped by `grpcapi`'s `toStatus` to
   `PermissionDenied`, never `Internal` — a forged or malformed signature is
   a rejected caller, not a server bug, the same discipline #195's four
   tickets just established project-wide for a different failure shape; (b)
   claim the event via `s.webhookEvents.ClaimEvent(ctx, in.EventID)` — if
   not claimed (already seen), **return nil without reprocessing**: this is
   the redelivery-safe no-op Stripe's own retry behaviour requires, proven
   with a test that delivers the same `event_id` twice and asserts the
   second delivery captures nothing a second time; (c) if `in.EventType` is
   not the one type this ticket handles (`"payment_intent.succeeded"`),
   **return nil without further action** — an explicit, tested, disclosed
   no-op for every other event type, not a silent crash or an
   `Unimplemented`; (d) resolve the Payment via
   `s.payments.GetByStripeReference(ctx, in.StripeReference)` — a not-found
   result maps to a clean not-found status, never `Internal` (same
   discipline as (a)); (e) **if the resolved Payment is already in a paid
   status, return nil without calling `captureAndMarkPaid` again** — proven
   confirmed necessary by §A3's read of `MarkPaid`'s illegal-transition
   behaviour: this guard is what makes a webhook event racing an
   already-completed client-driven `ConfirmOnlinePayment` call (or a
   second, differently-`event_id`'d redelivery of conceptually the same
   underlying event) a safe no-op rather than a surfaced error; otherwise
   (f) call the shared `s.captureAndMarkPaid(ctx, p)` from instruction 6.
8. **Wire `webhookVerifier`/`webhookEvents` into `payments/app.ServiceOptions`
   /`Service`** and into `cmd/server`'s construction of the Payments
   service, using `webhookstub.NewVerifier` with a placeholder shared
   secret exactly as `stripestub.NewProcessor()` is wired today (no new
   env var this ticket, matching instruction 4's stated boundary).
9. **Close #167 after merge**, per DoD step 5, with a comment naming this
   PR and stating plainly what remains true and unclosed by it: the
   client-driven `ConfirmOnlinePayment` path still exists and is still the
   only mechanism a Player's own device drives directly; this ticket adds a
   second, independent completion path, it does not retire the first one
   (per the sprint goal's own "does not claim" list).
10. **Non-functional:** TDD-first. Mutation-checked headline tests, named:
    a valid signature over a known event captures the Payment (asserted via
    the real `s.payments`/`webhookstub`/`s.processor` — not a fake that
    returns whatever it's told, T14.8/T15.5's standing cross-context-fake-
    trap warning applied here to a same-context boundary); an invalid
    signature is refused with `PermissionDenied` and captures nothing;
    redelivering the same `event_id` a second time captures nothing a
    second time; an event for an already-paid Payment (simulating a race
    with a prior client-driven confirm) captures nothing a second time and
    returns success; an unrecognized `event_type` is acked and does
    nothing; an unrecognized `stripe_reference` returns a clean not-found
    status, not `Internal`. Also required: run `authenticated_test.go`'s
    exhaustiveness check both before instruction 2 (to confirm it fails,
    proving the new RPC really is unclassified until this ticket classifies
    it) and after (to confirm it passes).
11. **Sibling sweep, reported either way:** does this ticket change anything
    about #167's own "Two consequences of the current shape" section — in
    particular, is "only the payer can capture" (consequence 1) still
    accurate once a webhook-authenticated path exists alongside the
    principal-authenticated one? (From inspection at planning time: the
    RPC-level authorization on `ConfirmOnlinePayment` itself is unchanged —
    still exactly the Payment's own `RecordedByUserID` — the webhook path
    is a structurally separate completion mechanism authenticated by
    Stripe's own signature, not a widening of who may call
    `ConfirmOnlinePayment`; confirm this reading is still correct at
    implementation time rather than trusting the plan's own inspection,
    per this project's own recurring lesson about planning-time inspection
    missing things — T16.2's identical planning-time inspection missed
    #198 itself.)

## Waves

**Wave 1 — the sprint's only ticket (1 ticket, 8 points)**
`T18.1` (#167)

No same-wave pair exists (§A8); no Wave-1.5 checkpoint condition (a new
cross-cutting decision with three or more first-time in-sprint consumers)
can fire with a single ticket.

## Recorded disagreements (Ceremony 2 rule 3 — not smoothed over)

**PM's residual concern, recorded rather than smoothed over.** PM's mandate
is protecting product value and market timing, and a 1-ticket, 8-point
sprint is the smallest this project has run. PM's position: a
webhook-receiver ticket is real engineering value but is not, on its own, a
loop a real user completes differently than they could before — the online
payment flow already works end to end via the client-driven path; this
ticket hardens reliability, it does not unlock new user-facing capability.
**PdE and PE's position, which governs:** the alternative to taking #167 is
not a bigger sprint, it is guessing at one of D1, a real IdP tenant, or a
Product Owner decision this session cannot supply (§A5) — none of which PM
disputes are genuinely blocked. Manufacturing sprint size against a
genuinely thin, mostly-blocked backlog is exactly the failure mode
`sprint-process.md`'s "take only what's genuinely unblocked" discipline
(established at T16, reaffirmed at T17) exists to prevent, and PM does not
propose a specific alternative ticket that would be both unblocked and
larger. **Not overridden by force of the size difference — PdE/PE's
position that no larger unblocked alternative exists is what governs, and
that claim is checkable** (§A5's per-issue blocker table), not asserted.

## Sprint-level Definition of Done

All of `sprint-process.md`'s standing DoD, plus the scorings T18 owes,
stated now so they are not improvised at the retro:

1. T18.1 merged per the per-ticket DoD; sprint goal met or explicitly
   descoped with reasoning recorded.
2. **The merged-fix issue sweep run and reported with its count** — by the
   retro (reporting, not blocking) and again by T19's Ceremony 1
   (authoritative).
3. **Scoring owed at the retro:**
   - **(a)** Did T18.1 actually close #167, scored directly against the
     merged code (does a valid, signed webhook event actually capture a
     Payment without any client call), not against the PR's own account?
   - **(b)** Is the client-driven `ConfirmOnlinePayment` path still
     byte-for-byte functional and unchanged in its own authorization
     behaviour, proving the sprint goal's "does not claim" section held —
     the webhook path was additive, not a silent replacement?
   - **(c)** Did the redelivery/already-paid idempotency guards (instruction
     7(b), 7(e)) actually get exercised by a test that would fail without
     them, not merely asserted?
   - **(d)** Did the migration-header-ownership check (§A4, T17 retro
     recommendation 1) get applied correctly — was `payments_webhook_events`'s
     own new migration header written in a form a future ceremony can read
     the ownership answer from in one line, the way this ceremony asked
     `0017_booking_discount_rules.sql`'s header be used as the template?
4. **Not scoreable by T18 and deliberately not pre-empted:** D1 and D2
   remain the user's. If either is answered mid-sprint, the answer's own
   trigger takes over and T18's plan does not constrain it.
5. Retro in `docs/process/t18-retro.md`, indexed by a `## T18 sprint retro`
   stub in `docs/LESSONS.md`. `HANDOFF.md`/`CLAUDE.md` state updated —
   noting that **T19's Ceremony 1**, not the retro, corrects T18's
   Docs-index row (the ordinary convention).
