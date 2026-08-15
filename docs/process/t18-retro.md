# T18 Sprint Retro

Ceremony 3 per `docs/process/sprint-process.md`, six-role team (briefs:
`docs/agent-operating-handbook.md` Part B), held against
`docs/process/t18-sprint-plan.md` (§A0–§A8, Ceremony 2), `docs/process/t17-retro.md`
and `docs/process/t16-retro.md` as the precedent and rigor bar, `HANDOFF.md`,
and the real PR/issue history on `nhuthuynh/white-label` (GitHub-side name
`pickleball-platform`) — PRs #209–#210, issue #167.

Every timestamp, closure, label, and code claim below was pulled from GitHub's
own API fields and from direct reads/builds/**mutation tests of the merged
tree at `34e9275`** — never inferred from the PR's title, its own account of
what it did, or the sprint plan's forward-looking text (CLAUDE.md rule 10).

**Verification performed before writing a single finding.** `git status`
showed a clean worktree at the shared branch's tip (`34e9275`, T18.1's own
merge) before this retro's branch was cut. `make generate && go build ./...
&& go vet ./... && make fmt-check && make vet-integration && make test-domain
&& make test-adapters && make test-cmd && make test-platform && make
gate-coverage` were all run directly, not assumed from the PR's own account:

```
go build ./...                 # clean
go vet ./...                   # clean
make fmt-check                  # OK — gofmt clean
make vet-integration            # clean
make test-domain                # ok, all 12 packages
make test-adapters              # ok, all 22 packages, incl. the new
                                 #   internal/payments/adapter/webhookstub
make test-cmd                   # ok
make test-platform              # ok
make gate-coverage               # OK — all 42 package(s) executed by
                                 #   ci-checks (41 → 42, the new webhookstub
                                 #   package), matching the PR's own claim
```

**Beyond re-running the toolchain, this retro independently reproduced three
mutation checks against the actual merged tree** — not re-read the PR's or
the review's table, but re-performed the mutations itself, on a scratch copy,
restoring cleanly each time (`git status --short` empty after each restore):

1. Removed the already-paid guard (`if p.Status != domain.StatusUnpaid {
   return nil }`) from `HandleStripeWebhookEvent` → exactly
   `TestHandleStripeWebhookEvent_AlreadyPaidPayment_CapturesNothingTwice`
   failed, nothing else, confirmed with `go test -run
   TestHandleStripeWebhookEvent -v`.
2. Removed the idempotency claim short-circuit (`if !claimed { return nil
   }`) → exactly `TestHandleStripeWebhookEvent_RedeliveredEvent_CapturesNothingTwice`
   failed, nothing else.
3. Reverted `PublicMethods()` to `return nil` → exactly
   `TestAuthenticatedAndPublicMethods_CoverEveryRPC` failed, nothing else.

All three reproduce, exactly, both the PR's own claimed table and the
reviewing session's own three independent checks recorded in its PR #210
review comment. Not re-verified here (no Docker daemon reachable, same
standing gap as every prior sprint): the Docker-backed `make
test`/`ci-integration` path; this retro also did not independently re-run the
signature-bypass or event-type-filter/not-found mutations the PR's own table
claims — named honestly as not re-checked, per this project's own "silence is
indistinguishable from not having checked" standard, rather than folded
silently into "all mutation checks verified."

**Sprint outcome, stated before the findings that qualify it:** the sprint's
one ticket (T18.1, 8 points) merged as a single PR (#210), opened at
18:53:41Z and merged at 18:56:41Z (3 minutes wall-clock — a normal, not a
suspiciously fast, review window; contrast the 9s/8s/14s/13s windows
`docs/adr/0016-*.md` scrutinizes for D2). Issue #167 was closed 5 seconds
after merge (`18:56:46Z`), citing PR #210 by number, with the closing comment
independently restating the toolchain result and the three mutation checks —
matching DoD step 5's mandatory-close-for-"closes #N" requirement, held for
what is now several consecutive sprints running.

**What this retro found, in one sentence, so the findings do not have to be
read before the headline is known:** T18.1 genuinely closes #167 with
idempotency guards that are mutation-proven, not merely asserted, and the
verification chain behind it — implementer self-check, independent reviewer
re-check, and now this retro's independent re-check — converges on every
point checked; the one real, narrow miss is that the PR's and the review's
own text both assert `ConfirmOnlinePayment` is "unchanged, byte-for-byte,"
and the merged diff shows that claim is not literally true — the method's
*behavior*, *authorization gate*, and *doc comment* are unchanged, but its
*body* was refactored (extracted into a shared `captureAndMarkPaid`) exactly
as the ticket's own instruction 6 required. Finding 2 is that finding,
argued in full rather than folded into a "no gaps found" summary.

---

## 1. The merged-fix issue sweep — clean, reconciled exactly, fourth sprint
running, #167's close independently re-verified against the merged code

**PO.** Per `sprint-process.md`'s DoD, this retro is sweep moment 1; T19's
Ceremony 1 remains the authoritative moment regardless of this result.

**Step 1 — list the open issues**, live, at this retro's start:
`list_issues(state: OPEN)` → **`totalCount: 8`**: #124, #126, #130, #134,
#144, #145, #149, #164.

**Step 2 — reconcile arithmetically, before reading any issue individually.**
`open_at_end_of_T18's_own_Ceremony_1 − closed_during_T18 + opened_during_T18`.
T18's own Ceremony 1 (§A1 of `docs/process/t18-sprint-plan.md`) left the count
at **9**. During execution: **#167** closed (T18.1, PR #210) — `9 − 1 = 8`.
**Matches the live `totalCount: 8` read at this retro's start exactly.** Zero
issues opened this sprint — confirmed by re-reading PR #210's own body and
its instruction-11 sibling sweep, which found no new gap requiring a fresh
issue.

**Step 3 — cross-reference the merged PR against the open list, checking
attribution against the merged code, not the closing comment's prose.**

| Issue | `state` (re-fetched now) | `state_reason` | `closed_at` | Closed by / cites |
|---|---|---|---|---|
| #167 | `closed` | `completed` | 2026-08-15T18:56:46Z | Comment naming PR #210, independently restating the toolchain result and three mutation checks |

Cross-checked against PR #210's own `merged_at`: `18:56:41Z` — 5 seconds
before #167's close, performed by the same account that merged, matching the
per-PR optional-early-close DoD step 5's mandatory form for "closes #N"
titles.

**#167's own gap is genuinely closed, verified against the code, not the
comment's word.** `internal/payments/adapter/grpcapi/authenticated.go`'s
`PublicMethods()` now returns `[]string{
paymentsv1.PaymentsService_ReceiveStripeWebhookEvent_FullMethodName}`
(previously `nil`, per T18's own Ceremony 1 §A3 read of the pre-sprint tree);
`app.Service.HandleStripeWebhookEvent` exists, is wired into
`cmd/server`, and its full signature-verify → idempotency-claim →
event-type-filter → resolve → already-paid-guard → capture sequence is
implemented and mutation-tested (finding 2's DoD-(c) scoring, below).

**Sweep result: clean, fourth sprint running** (after T15, T16, T17).
**T19's Ceremony 1 still re-runs this sweep in full**, per the standing rule
that a prior ceremony's clean result does not discharge the next one.

## 2. DoD scoring (a)–(d), each argued against the merged code — and the one
real finding: "unchanged, byte-for-byte" is not literally true, though every
functional guarantee that phrase was meant to protect does hold

**PE (the claim, checked before accepting it).** PR #210's own body states:
*"`ConfirmOnlinePayment` is unchanged, byte-for-byte, and stays the only path
a Player's own device drives directly."* The reviewing session's own review
comment repeats the same framing implicitly by confirming the ticket's
instructions were followed "in the specified order." **Checked directly
against `git diff 76de029 34e9275 -- internal/payments/app/service.go`**,
not taken from either account:

```go
 func (s *Service) ConfirmOnlinePayment(ctx context.Context, p domain.Payment, actorUserID string) (domain.Payment, error) {
 	if err := authorizeOnlineConfirmation(p, actorUserID); err != nil {
 		return domain.Payment{}, err
 	}
-
-	// Any processor-side failure (declined or otherwise unavailable) means
-	// no capture happened, so p is returned exactly as it was passed in —
-	// there is nothing to roll back because MarkPaid was never called, and
-	// nothing is persisted.
-	if err := s.processor.CapturePayment(ctx, p.StripeReference); err != nil {
-		return p, err
-	}
-	if err := p.MarkPaid(domain.MethodOnline, p.StripeReference); err != nil {
-		return p, err
-	}
-	updated, err := s.payments.Update(ctx, p)
-	if err != nil {
-		return updated, err
-	}
-	if err := s.reconcileRegistrationPaymentStatus(ctx, updated, socialplaydomain.PaymentStatusPaid); err != nil {
-		return updated, err
-	}
-	if err := s.reconcileCompetitionEntryPaymentStatus(ctx, updated, competitionsdomain.PaymentStatusPaid); err != nil {
-		return updated, err
-	}
-	return updated, nil
+	return s.captureAndMarkPaid(ctx, p)
 }
```

**This is not "unchanged, byte-for-byte."** The method's body was reduced
from a ~20-line inline sequence to a single delegating call. **Three things
are genuinely unchanged, and this is where the claim's spirit does hold up:**

1. `authorizeOnlineConfirmation` itself — the actual authorization
   gate — has **zero lines touched** by this diff (confirmed: `git diff
   76de029 34e9275 -- internal/payments/app/service.go` contains no hunk
   inside that function's body; `grep -n "^func authorizeOnlineConfirmation"`
   at both commits shows the function exists at both, untouched).
2. `ConfirmOnlinePayment`'s own doc comment — every line of prose above the
   `func` signature, the exact wording covering the declined-card behaviour,
   the actor parameter, and why it's separate from `p` — is byte-identical
   at both commits (diffed directly; zero changed lines in the comment
   block).
3. `internal/payments/adapter/grpcapi/handler.go`'s `ConfirmOnlinePayment`
   RPC handler has **zero lines changed** — `git diff 76de029 34e9275 --
   internal/payments/adapter/grpcapi/handler.go` shows only additions (the
   new `ReceiveStripeWebhookEvent` handler and a new `toStatus` case), no
   modification to the existing handler.
4. The extracted `captureAndMarkPaid`'s body is the identical sequence of
   statements — `CapturePayment` → `MarkPaid` → `Update` → the two
   `reconcile*PaymentStatus` calls, in the same order, with the same error
   handling at each step — moved verbatim into the new function, not
   rewritten. The PR's own commit message states this precisely: *"This is
   the exact body `ConfirmOnlinePayment` had inline before this ticket;
   behaviour is unchanged for that caller"* — a narrower, accurate claim
   that the wider "byte-for-byte" sentence in the PR's summary paragraph
   overstates.

**Why this is not a defect, and why it is still worth naming.** T18.1's own
instruction 6 explicitly *required* this extraction — *"extract
`ConfirmOnlinePayment`'s capture-then-mark-paid-then-reconcile sequence into
a shared private method"* — precisely so `HandleStripeWebhookEvent` could
call the identical logic rather than duplicating it (CLAUDE.md's DRY
expectation, and the single-source-of-truth property the ticket's own text
says "is what makes instruction 7's already-paid guard correct in exactly
one place instead of two"). A literal byte-for-byte-identical
`ConfirmOnlinePayment` was never achievable once instruction 6 was followed
— the ticket that asked for "byte-for-byte unchanged" in the sprint plan's
DoD item (b) and the ticket that mandated the refactor are the same ticket.
**The honest scoring is that DoD (b) is satisfied in every sense that
matters — behaviour, authorization, and the handler boundary are all
provably unchanged — but not in the literal sense its own wording states**,
and the PR's summary paragraph repeats the stronger, inaccurate version of
the claim rather than the narrower, accurate one its own commit message
uses two paragraphs later. Worth a recommendation (below) precisely because
this project has now seen several instances of a claim's strong form being
asserted where a weaker, true form was available and would have cost nothing
extra to state.

**DoD (a) — did T18.1 actually close #167, scored against the merged code?**
**Yes.** `PublicMethods()` carries the new entry (finding 1); the RPC is
wired end-to-end through `cmd/server` (`webhookstub.NewVerifier(...)` and
`paymentspg.NewWebhookEventRepository(pool)` both passed into
`payments/app.ServiceOptions`, confirmed by reading `cmd/server/main.go`
directly); `TestHandleStripeWebhookEvent_ValidSignatureCapturesPayment`
exercises the real `webhookstub`/`stripestub`/repository stack end to end
(not a fake standing in for a cross-context boundary, per T14.8/T15.5's
standing warning applied here) and passes. A valid, signed webhook event
genuinely captures a Payment with no client call involved.

**DoD (b) — is `ConfirmOnlinePayment` still functionally unchanged, proving
the webhook path was additive?** **Yes, functionally — scored above with
the literal-claim caveat named rather than smoothed over.**

**DoD (c) — did the redelivery/already-paid idempotency guards actually get
exercised by a test that would fail without them?** **Yes, and this retro's
own mutation checks (above) independently confirm it, not just the PR's or
the review's account.** Worth naming specifically:
`TestHandleStripeWebhookEvent_RedeliveredEvent_CapturesNothingTwice`'s own
doc comment states *why* it asserts a repository call-count
(`getByStripeReferenceCalls.Load()`) rather than only the final `Status` —
because the already-paid guard downstream would independently produce the
same end state even if the idempotency-claim guard were entirely missing, so
an end-state-only assertion would pass regardless of whether guard 7(b)
exists at all. This is exactly the QA discipline
`docs/agent-operating-handbook.md`'s B4 brief asks for ("an invariant with
no test that could fail is untested, not proven") applied correctly, and the
PR's own account (*"a weaker end-state-only assertion would NOT have caught
this... strengthened the test after discovering this during the mutation
check itself"*) is corroborated, not just repeated: this retro's own
mutation 2 (above) reproduces exactly the described failure.

**DoD (d) — was the migration-header-ownership check applied correctly?**
**Yes.** `db/migrations/0022_payments_webhook_events.sql`'s header states
the owning context (Payments), the ticket (T18.1, closes #167), and — going
beyond the check's own minimum bar — the full reasoning for why a bare
`PRIMARY KEY` is sufficient invariant enforcement with no domain-side
pre-check, addressed explicitly to "a future ceremony's migration-header-
ownership check" by name. A future ceremony reading this header gets the
ownership answer and the design rationale in the same read, which is what
T17 retro recommendation 1 asked this check to produce.

## 3. The three-layer verification chain — implementer, independent
reviewer, and this retro — converges everywhere checked, honestly scoped to
what was actually re-checked rather than claimed in full

**QA.** The task that dispatched this retro asked whether the PR's own
claimed mutation checks, the reviewing session's three independent
mutation checks (recorded in its PR #210 review comment), and this retro's
own re-checks form a genuinely strong convergent chain or reveal a gap.
**Scored precisely, not rounded up:**

| Mutation | PR's own claim | Reviewer's independent re-check | This retro's independent re-check |
|---|---|---|---|
| Already-paid guard removed | claimed, table row | performed, reproduced | **performed, reproduced** |
| Idempotency claim short-circuit removed | claimed, table row | performed, reproduced | **performed, reproduced** |
| `PublicMethods()` reverted to `nil` | claimed, table row | performed, reproduced | **performed, reproduced** |
| Signature bypass (`VerifySignature` returns `nil`) | claimed, table row | not independently re-run (review's own list has 3 items, not this one) | not independently re-run |
| Event-type filter removed | claimed, table row | not independently re-run | not independently re-run |
| Not-found error swallowed | claimed, table row | not independently re-run | not independently re-run |

**Three of the six mutations the PR itself claims were independently
re-performed by both the reviewer and, now, this retro — and all three
reproduced exactly, with no discrepancy at any layer.** This is a genuinely
strong result and worth naming as a positive finding on its own terms,
matching T17 retro's own practice of naming a positive result rather than
only listing what broke. **It is not, however, evidence that all six
mutations were independently verified** — the reviewer's own review comment
is honest about scope (*"three of my own independent mutation checks"*), and
this retro is equally honest that it re-checked the same three, not the
other three. Stated so a future reader does not read "the chain held" as "all
six checks were triple-verified" when only half were.

## 4. D1 and D2 — both re-verified unchanged; D2's own prediction from T18's
Ceremony 1 confirmed correct

**Re-verified this retro, not assumed.** `issue_read(get_comments)` on #144
(D1): still exactly **one** comment, T14.3's original escalation, unchanged
across T14 through T18 — six sprints now. `docs/adr/0016-*.md`'s (D2) own
`## Status` field: unchanged, still **"Escalated — awaiting the user's
decision. This ADR decides nothing."**

**D1's footprint.** No T18 ticket touches `CancelBooking`, `CreateBooking`,
or the court-Bookings half of #124 — confirmed by reading the diff (finding
2's `git diff --stat`, 22 files, all under `internal/payments/**`,
`cmd/server`, `db/migrations`, and `proto/pickleball/payments/v1/**`; nothing
under `internal/booking/**`). D1's footprint neither grew nor shrank this
sprint, the same result T17 retro reported.

**D2.** T18's own Ceremony 1 (§A7) predicted, as a stated guess rather than
a guarantee, that this sprint would produce a fourth consecutive sprint with
nothing to score, since T18.1 was implemented and reviewed by the ordinary
two-role loop with no reviewer-authored code. **Checked directly against
PR #210's own commit list** (`get_commits`): one commit, authored entirely
by the implementing session; the review's own text explicitly states *"No
gaps found. Merging, and will close #167 naming this PR"* — no fix pushed
by the reviewing pass, no gap found that would have required one. **The
prediction holds: a fourth consecutive sprint (after T15, T16, T17) with no
reviewer-authored gap-fix to score.** Per this project's own stated caution
against over-reading a small, consistent sample, this is recorded as a
fourth null result, not evidence the interim rule can be retired.

**Neither D1 nor D2 is implemented, decided, or guessed at by this retro.**
Both remain exactly as blocked as `sprint-process.md`'s own restriction
lists require.

## No finding on

**No finding on the sibling-sweep instruction (11).** T18.1's own instruction
11 asked whether #167's "only the payer can capture" consequence-1 statement
still holds now that a webhook-authenticated path exists alongside the
principal-authenticated one. Checked directly: `ConfirmOnlinePayment`'s own
RPC-level authorization (`authorizeOnlineConfirmation`, still comparing the
verified principal against `RecordedByUserID`) is untouched by this PR
(finding 2, point 1) — the webhook path is a structurally separate
completion mechanism, not a widening of who may call `ConfirmOnlinePayment`.
The ticket's own planning-time inspection is confirmed correct at
implementation time, as instruction 11 itself required rather than assumed.

**No finding on the label taxonomy.** #167 already carried a conformant
three-label set (`type:story`, `role:principal-engineer`,
`context:payments`) from its original filing at T13.7; unchanged at close.
No new issue was opened this sprint to check.

**No finding on the wave structure.** T18 had exactly one ticket; no
same-wave shared-interface question could arise, and the sprint plan (§A8)
correctly scored this as "not exercised, zero opportunities" rather than
silently skipping the section.

**No finding on `port.Repository`'s blast radius.** `GetByStripeReference`
was added to `port.Repository` and every implementer that satisfies it
(`fakeRepository` in `internal/payments/app`, the Postgres adapter, and every
cross-context boundary fake in `internal/{competitions,socialplay}` that
implements the same interface) was checked directly — a grep for structs
implementing this repository's `GetByID` shape found none missing
`GetByStripeReference`. Not the T16.2/T16.3 shared-interface hazard, since
this sprint had exactly one ticket touching this interface with no
concurrent implementer being added elsewhere, but checked rather than
assumed.

**No finding on PCI conformance.** The new proto message
(`ReceiveStripeWebhookEventRequest`) carries only `raw_payload`,
`signature_header`, `event_id`, `event_type`, and `stripe_reference` — no
PAN, card number, CVV/CVC, or track-data field, confirmed by reading the
proto diff directly rather than trusting the PR's own PCI-guardrail
paragraph. CLAUDE.md rule 11 holds.

---

## The sprint goal, scored: what was proven, what shipped exactly as claimed,
and the one place a claim needed a narrower restatement

> *"The one genuinely unblocked issue on the backlog gets built: online
> payment capture stops depending entirely on a client call... while the
> existing client-driven `ConfirmOnlinePayment` path stays exactly as it is,
> deliberately not removed, so this ticket adds a capability rather than
> taking one away."*

**Every clause of the stated goal is met, verified independently.** #167 is
closed and the merged code backs the claim exactly (findings 1, 2). The
webhook path is additive: `ConfirmOnlinePayment`'s authorization, doc
comment, and RPC handler are all untouched; only its internal body was
refactored, in the way its own ticket instructions required, not as an
undisclosed side effect (finding 2). The idempotency and already-paid guards
are mutation-proven, independently reproduced by this retro, not merely
asserted (finding 2, DoD (c)). D1 and D2 remain open, confirmed via the API
(finding 4).

**What the plan's own DoD wording did not anticipate, and this retro's
central finding**: DoD item (b) asked whether `ConfirmOnlinePayment` is
"byte-for-byte unchanged," using the same wording the PR's own summary later
asserted as fact — and the ticket's own instruction 6, in the same document,
required the one change (extraction into `captureAndMarkPaid`) that makes a
literal reading of that phrase false. This produced no shipped defect and no
functional gap; it is a case of a plan's own DoD wording and a plan's own
ticket instructions pulling in slightly different directions, both correctly
followed, with a resulting PR claim that overstates in the direction the DoD
wording invited it to.

**The agreed honest sentence, which T19's Ceremony 1 should carry into
`HANDOFF.md`'s Task-backlog entry** (`sprint-process.md` Ceremony 1 item 3
requires the retro's form, not a stronger one):

> T18 closed #167 (a new, public, signature-authenticated
> `ReceiveStripeWebhookEvent` RPC lets a successful Stripe charge complete
> its own Payment even when the client's own `ConfirmOnlinePayment` call
> never lands) via the mandatory "closes #N" mechanism, the fourth
> consecutive clean sweep. The idempotency and already-paid guards are
> mutation-tested with tests specifically designed to isolate each guard
> from the other (the redelivery test asserts a call count, not just an end
> state, precisely because the already-paid guard would otherwise mask a
> missing idempotency guard) — independently reproduced by this retro, not
> merely re-read. `ConfirmOnlinePayment`'s authorization, doc comment, and
> RPC handler are genuinely untouched, so the webhook path is additive as
> claimed; its internal body was refactored, exactly as the ticket's own
> instruction 6 required, which makes the PR's "unchanged, byte-for-byte"
> phrasing a narrow overclaim of an otherwise-true functional guarantee —
> caught here, zero shipped consequence. D1's footprint held steady; D2 had
> its fourth consecutive sprint with nothing to score, confirming the
> prediction T18's own Ceremony 1 made rather than merely repeating "still
> unanswered." Both remain unanswered by the user.

---

## Recommendations for T19's Ceremony 1 and 2

1. **When a sprint plan's own DoD item and a sprint plan's own ticket
   instruction make claims about the same code that cannot both be true in
   their strongest form — here, "byte-for-byte unchanged" (DoD (b)) versus
   "extract this into a shared method" (instruction 6) — the ticket text
   should state the narrower, achievable form of the DoD claim up front
   (e.g. "behaviourally unchanged; its body will be refactored per
   instruction N") rather than leaving the PR to discover the tension and
   resolve it by asserting the stronger, technically inaccurate version.
   This is a one-sentence addition to ticket-writing discipline, not a new
   ceremony step (finding 2).
2. **Continue treating the merged-fix sweep as authoritative regardless of
   this retro's clean result** — T19's Ceremony 1 re-runs the sweep and
   re-verifies #167's close and attribution from the API rather than
   trusting this retro's table (finding 1).
3. **When a review claims N independent mutation checks, a later retro that
   re-performs any of them should state exactly which ones it re-checked
   and which it did not**, rather than letting "the chain held" read as "all
   claims were re-verified" — the three-of-six scoping in finding 3 is the
   worked example to follow.
4. **D1 and D2 stay with the user.** No T19 ticket should implement
   `CancelBooking` authorization or a reviewer-authorship carve-out; neither
   should be guessed at. If either answer arrives mid-sprint, each
   escalation's own trigger takes over.
5. **Now that T18 has produced a case where a single-ticket sprint's own PR
   summary paragraph and its own commit-message body state the same fact at
   two different strengths (the summary's "byte-for-byte," the commit
   message's narrower "the exact body... moved"), a future review should
   flag the stronger phrasing specifically when a weaker, equally-cheap-to-
   state, and actually-true phrasing was available in the same PR** — not a
   new check, just a specific instance of "verify the PR's own claim against
   its own diff" this project's reviews already do, applied to the PR's
   summary prose and not only its code.

## Sprint-level Definition of Done — scored against what T18's own plan asked

Per `docs/process/t18-sprint-plan.md`'s "Sprint-level Definition of Done,"
four scorings were owed at this retro, stated there so they would not be
improvised — restated here with their answers:

- **(a) Did T18.1 actually close #167, scored directly against the merged
  code?** **Yes** — finding 2, DoD (a).
- **(b) Is `ConfirmOnlinePayment` still byte-for-byte unchanged, proving the
  webhook path was additive not a silent replacement?** **Functionally yes;
  literally no** — its authorization gate, doc comment, and RPC handler are
  untouched, but its body was refactored per the ticket's own instruction 6.
  The additive property the question is actually protecting holds; the
  literal wording of the question does not. Argued in full at finding 2.
- **(c) Did the redelivery/already-paid idempotency guards actually get
  exercised by a test that would fail without them, not merely asserted?**
  **Yes**, independently re-verified by this retro's own mutation checks,
  not taken from either the PR's or the review's account — finding 2, DoD
  (c), and the top-of-document mutation log.
- **(d) Did the migration-header-ownership check get applied correctly?**
  **Yes** — finding 2, DoD (d); the header states both the ownership answer
  and the design rationale for a future reader.

**Not scoreable by T18 and deliberately not pre-empted:** D1 and D2 remain
the user's (finding 4).

Retro complete. Issue-tracker actions this ceremony: none — #167's close and
its correct attribution were already performed correctly during the sprint
itself, the fourth sprint running with nothing left for the retro to clean
up on the closure axis. Open count at ceremony start: **8**. Open count now:
**8** (unchanged — nothing found here needed a live tracker action; finding
2's DoD-(b) caveat caused no shipped defect and needs no issue, the same
reasoning T17 retro's finding 4 used for its own caught-before-merge miss).

Per `sprint-process.md`'s established convention (a retro PR never updates
the Docs-index row that points at it, since that row must cite this PR's own
merge number, which does not exist until it merges): **`HANDOFF.md`'s T18
row is not touched by this PR.** T19's Ceremony 1 corrects it, including its
real PR merge order and the honest-form sentence above, as its first job —
the same standing convention T17's retro followed (and the same convention
T16's retro's own self-correction was an explicitly-argued one-off deviation
from, not a new default).
