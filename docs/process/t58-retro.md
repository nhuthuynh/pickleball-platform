# T58 Sprint Retro

Ceremony 3 per `docs/process/sprint-process.md`, six-role team (briefs:
`docs/agent-operating-handbook.md` Part B), held against `HANDOFF.md`,
`docs/process/t56-retro.md` and `docs/process/t57-retro.md` as the immediate
precedents, PR #301, PR #302, and the live issue/PR/commit history.

**T58 held no planning ceremony.** Its first act was to re-put a question to
the Product Owner, because the issue it was asked to build turned out to
rest on a false premise.

**Outcome: 1 ticket, 1 issue closed, 1 decision answered, 1 issue's analysis
publicly corrected.** T58 merged as PR #301 (`619a974`), closing #299; PR
#302 (`6cd7ade`) recorded T56–T58 in `HANDOFF.md`.

---

## 1. What actually happened, in order

1. The session was asked to do #299 — a question this project had filed
   itself, two sprints earlier, asking whether `RecordOfflinePayment` should
   validate the amount.
2. **It read the schema before building**, and found `payments_payable_
   unique_idx` (migration 0005): one Payment per `(payable_type,
   payable_id)`, enforced in Postgres since T5.
3. That single constraint falsified #299's central argument (§2), so the
   options were **re-put to the Product Owner with the corrected framing**
   rather than built as filed. Answer: **exact match**.
4. T58 was built TDD-first — the defect reproduced as a failing test before
   the fix — and merged.
5. Removing a client fallback that validation had made unsafe exposed a
   **live assertion error** in the existing suite (§3).
6. The pre-merge review ran a deliberate bypass hunt (§4).
7. #299 was closed with the correction written onto the issue itself, not
   only into the commit message.

## 2. The finding this retro exists to record

**This project filed an issue arguing to preserve a capability the schema
forbids, and came one investigation away from building to it.**

#299's central argument, in its own words:

> a Game Admin recording cash may legitimately be recording a part-payment.
> Someone hands over $20 of a $30 obligation at the court; refusing to
> record it would mean the system has no way to represent money that
> genuinely changed hands.

Every clause of that is reasonable. None of it was true here.
`payments_payable_unique_idx` permits exactly one Payment per payable, so
the part-payment path was not *supported-but-unvalidated*. It was
impossible, and its failure mode was worse than a refusal:

1. the Host records $20.00 of the $30.00 debt;
2. `reconcileRegistrationPaymentStatus` marks the Registration **paid in
   full** — it keys off a Payment's *existence* and never reads its amount;
3. the remaining $10.00 can never be recorded: `ErrPaymentAlreadyRecorded`.

Mis-recorded, then locked. **The case the exemption existed to protect was
already broken, in precisely the way the exemption claimed to prevent.**

### Root cause, stated plainly

The options in #299 were produced by **reasoning about the domain rather
than reading the constraint**. "A part-payment is a legitimate business
event" is a true statement about pickleball. It is not a statement about
this system, and the file that settles it is 51 lines long and was written
by this same project.

The consequence was not hypothetical. Option 2 as filed ("refuse more than
owed, allow less") would have shipped a guard that **advertised protection
it could not provide**: reconciliation settles the debt on any recorded
Payment regardless of amount, so a permitted $1.00 record would still have
marked a $30.00 debt paid. A plausible-sounding option, written by someone
with full repository access, that would have made things worse.

### Why the correction is on the issue and not only here

An issue is read by whoever picks it up next, usually without its retro.
Leaving a falsified premise in the issue body while correcting it in a
retro would reproduce the original failure one layer down. #299 carries the
correction in a closing comment that states what was wrong, why, and what
the options actually reduced to.

### Relationship to T56's finding — they are the same failure

T56's retro initially recorded #126 as a ticket that went *stale*. Checking
the dates during that retro showed otherwise: `0013_socialplay_entry_fee.sql`
landed 2026-08-05 and #126 was opened 2026-08-14, quoting a T8.10 inspection
in the present tense. **Both issues were false on the day they were filed.**

| | #126 (T56's finding) | #299 (this one) |
|---|---|---|
| Asserted | "no price/fee field at all" | "a part-payment may legitimately be recorded" |
| Falsified by | `0013_socialplay_entry_fee.sql`, 9 days earlier | `0005_payments.sql`, in place since T5 |
| Why the author believed it | quoted an older sprint's inspection | reasoned about the domain |
| Would an age filter catch it? | no — 9 days old | no — 2 sprints old |

Two issues, filed 5 weeks apart by this same project, each asserting
something about the system that its author had not checked against the
system. That is one failure mode with two faces, not two findings — and the
remedy is the same for both: **an issue that asserts what the code currently
does must cite where that was verified.**

Neither would have been caught by an age threshold, which is why T56's
recommendation 1 was written without one.

## 3. The fallback that hid the bug it was compensating for

Validation made T56.1's entry-fee fallback unsafe — the payable owes 0 for
a pre-T56.1 row, so the fallback figure would now be refused. Removing it
exposed a **live assertion error already in the suite**:

`HostPayments.spec` asserted that a registration **with a guest** records
`1000` against a 1000-per-player Game. That is #126's own defect, sitting in
the test suite asserted as *correct* — and it survived **T56.1, the ticket
that fixed #126**, because the fixture carried no `amount_owed` and the
composable fell back to the per-player fee.

Generalised:

> **A fallback that hides a missing fixture field will also hide the bug
> that field exists to fix.**

This is the second instance in three sprints of a safety net making a
broken thing look fine — T56's wire-test gap was the first, where a green
gate hid a feature that did not work at all. Both were caught pre-merge, so
neither is a `docs/LESSONS.md` incident. **A third instance should go to
LESSONS properly**, and this paragraph exists so whoever hits it knows there
were two before.

## 4. What went well

- **The question was re-put rather than answered unilaterally.** The
  corrected framing changed which options were even coherent. Building the
  issue as filed would have been defensible and wrong.
- **The review hunted for bypasses rather than re-reading the diff.** Three
  attempts, all recorded on the PR: payable-type confusion (send
  `no_show_fee` with a registration's id — fails, because reconciliation
  early-returns on type), the webhook path (cannot create a Payment, only
  capture one), and a completeness check reduced to a command anyone can
  run: `grep -n "s.payments.Create("` returns exactly two call sites, both
  validated. **A claim a reviewer can re-verify in one command beats a
  claim they have to trust.**
- **Verified by deletion, before claiming the tests worked.** Removing the
  new guard fails four tests. T56's review taught this; T58 applied it
  without being prompted.
- **`no_show_fee`'s exclusion got a test.** It shares a payable id with a
  Registration, so a check keyed on the *id* rather than the *type* would
  compare every no-show fee to the entry price and refuse all of them. That
  is a live trap, and it is now pinned rather than described.

## 5. Recommendations for T59 and beyond

1. **Before building an issue, verify its premise against the tree —
   regardless of the issue's age.** #299 was two sprints old and #126 nine
   days old; both were false at filing. Age is a proxy for wrongness and a
   bad one. The check is cheap enough to be unconditional for any issue
   whose body asserts what the system currently does.
2. **An issue that proposes options should cite the constraint each option
   is bounded by.** #299 listed three options and cited no schema. Had its
   author been required to name the constraint, they would have read
   `0005_payments.sql` and written a different issue — or none.
3. **When an issue's analysis is corrected, correct the issue.** Adopted in
   practice here; stated so it is a rule rather than one session's instinct.
4. **Treat a third instance of "a safety net hid a defect" as a LESSONS
   entry, not a retro finding.** Two have now occurred in three sprints
   (§3). The pattern is established; what is missing is evidence it recurs
   after being named.

## 6. Sweep and bookkeeping

- **Issues: 4 open, live-verified** — #296, #149, #145, #134. #299 closed
  by PR #301. Across T56–T58: #126, #297 and #299 closed; #297 and #299
  both opened *and* closed within the run.
- **Merge order #298 → #300 → #301 → #302**, verified by merging in that
  sequence.
- **Amount validation is now complete and checkable.** Exactly two call
  sites create a Payment and both validate; the webhook path cannot create
  one. What remains unchecked is argued rather than inherited: `booking`
  has no owed-amount concept, `no_show_fee` has no correct figure.
- **Self-review for the third consecutive sprint.** GitHub refuses an
  author's own approval, so all four PRs carry review comments rather than
  approvals. The reviews found real defects in T56 and T57 and a real
  premise error here — the form has demonstrable value — **but no second
  party has read any of these diffs.** Recorded plainly because three
  sprints of payment-path changes is the point at which that stops being a
  footnote.
- **Docker unavailable for the third consecutive sprint.** The integration
  tests compile and have never been executed. `make ci-integration` on a
  Docker-capable machine is owed and should be run before anything here
  goes near real money. Nothing in T56–T58 is described as proven under
  concurrency.
- **This retro does not update `HANDOFF.md`'s T58 Docs-index row** — per
  `sprint-process.md` a retro PR cannot cite its own merge number. T59's
  Ceremony 1 owns that row, and the T56 and T57 rows too.

## 7. Honest-form outcome sentence

For `HANDOFF.md`'s T58 row, to be carried verbatim rather than strengthened:

> T58 held no planning ceremony. Asked to build #299 — a question this
> project filed itself two sprints earlier — its first act was to read the
> schema, which falsified the issue's central argument: #299 defended the
> offline exemption on the grounds that a cash **part-payment** may be
> legitimate, but `payments_payable_unique_idx` (migration 0005) permits
> exactly one Payment per payable, so a part-payment was never
> supported-but-unvalidated. It was impossible, and its failure mode was
> worse than a refusal — the $20.00 record of a $30.00 debt takes the slot,
> reconciliation marks the payable paid in full, and the remaining $10.00
> can never be recorded. The options were **re-put to the Product Owner
> with the corrected framing** rather than built as filed, and answered:
> exact match, the only rule consistent with one-Payment-per-payable.
> Merged as PR #301 (`619a974`), closing #299, with the correction written
> onto the issue itself rather than only into a commit message. The root
> cause is recorded as a finding: #299's options were produced by
> **reasoning about the domain rather than reading the constraint**, and
> its option 2 would have shipped a guard that advertised protection it
> could not provide. Together with T56's #126 — opened nine days after the
> field it said did not exist — that is **two issues filed five weeks
> apart, each asserting something about the system its author had not
> checked against the system**: one failure mode with two faces, and
> neither catchable by an age threshold. Removing a client fallback that validation had made
> unsafe exposed a **live assertion error** — `HostPayments.spec` asserted
> a registration *with a guest* records one player's fee, which is #126's
> own defect asserted as correct, surviving the very ticket that fixed
> #126 because the fixture carried no `amount_owed`. That is the second
> instance in three sprints of a safety net hiding a defect; a third
> belongs in `docs/LESSONS.md`. Amount validation is now complete and
> checkable in one command: exactly two call sites create a Payment and
> both validate. Docker was unavailable for the third consecutive sprint,
> so the integration tests compile and have never been executed, and no
> second party has reviewed any of T56–T58's diffs.
