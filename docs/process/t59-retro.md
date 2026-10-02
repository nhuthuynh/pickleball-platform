# T59 Sprint Retro

Ceremony 3 per `docs/process/sprint-process.md`, six-role team (briefs:
`docs/agent-operating-handbook.md` Part B), held against
`docs/process/t59-sprint-plan.md`, `HANDOFF.md`,
`docs/process/t58-retro.md` as the immediate precedent, PRs #304 and #306, issue
#149, and the live issue/PR/commit history.

> ## Written late
>
> **Written at T61, two sprints after T59 ran.** `HANDOFF.md`'s T59 row said
> "not yet written". Unlike T55's retro, this one has a sprint plan to be held
> against — `t59-sprint-plan.md` is unusually complete, including a
> per-ticket status and a §305 "In: T59.1 only" scope statement — so the
> reconstruction is firmer here than for T55 or T60. Every figure was
> re-derived with a command at writing time, per `t61-retro.md`
> recommendation 4.
>
> The delay did buy one thing a contemporaneous retro could not have had:
> **T60 and T61 have since run**, and both acted on T59's output. That makes
> §4's finding observable, where at T59 it would have been a prediction.

**T59 held the first planning ceremony since T55** — T56, T57 and T58 each held
none.

**Outcome: 1 ticket shipped, 2 completed inside Ceremony 1, 1 issue closed.**
T59.1 merged as PR #306 (`b7608e3`), closing #296. Ceremony 1 merged as PR #304
(`9fed700`), carrying T59.2 and T59.3 as its own bookkeeping. Issue count
unchanged at 4 for the sprint, 4 → 3 counting #296's closure.

---

## 1. What actually happened, in order

1. Ceremony 1 ran the merged-fix sweep clean, reconciled the live issue count
   arithmetically against T55's, and confirmed the 0-ticket counter had reset.
2. **It applied T56's new premise check for the first time, and the check
   fired** (§2): #149's premise had substantially drifted.
3. Three tickets were taken: **T59.1** (guard `Booking.OwnerUserID`'s shape,
   closing #296), **T59.2** (re-scope #149), **T59.3** (adopt T56–T58's retro
   recommendations into `sprint-process.md`).
4. T59.2 and T59.3 were **completed during the ceremony itself** — explicitly,
   not by omission: the plan's §"In" says *"T59.1 only. T59.2 and T59.3
   completed during Ceremony 1 as its own bookkeeping rather than sprint
   execution."*
5. T59.1 shipped three days later as PR #306, guarding a malformed
   `OwnerUserID` with `domain.ErrInvalidOwnerReference` before it can reach the
   adapter's `mustUUID`.
6. **T59.1 fixed a Social Play fixture problem locally**, with a
   `resolvingIdentityLookup` used only in booking's tests — a stopgap T60 then
   had to retire (§4).

## 2. The premise check fired on its first use

T56's retro recommendation 1 asked that the Ceremony 1 issue sweep re-verify an
issue's **premise**, not only its blocker, for every issue regardless of age.
T59's Ceremony 1 was the first sweep to apply it, and it immediately found that
**#149 was describing a world four of its five facts had left.**

#149 named five caller-supplied ownership facts. Four — `game_host_id`,
`assigned_game_admin_user_ids`, `entrant_player_id`,
`assigned_competition_admin_user_ids` — were no longer read from the wire,
closed by T16.2 and T17.1. Only `booking_host_id` remained.

A process rule justifying itself on first contact is rare enough to record
plainly. T56's recommendation was written after #126 turned out to have been
false on the day it was filed; two sprints later the same check caught a
different issue drifting a different way. **The sweep had been asking "can anyone
act on this yet?" for many sprints and never "is this still describing
reality?", and the first time it asked, the answer was no.**

> **A figure deliberately not quoted here.** `t56-retro.md` states that 23
> Ceremony 1 sweeps re-verified #126's blocker, and gives its method: a `grep -c`
> of `HANDOFF.md` narratives carrying one exact sentence. **That method does not
> reproduce at T61** — the sentence it names returns 0, and the nearest patterns
> return 5, 33, 47 and 12 depending on how much of it you keep. Either
> `HANDOFF.md`'s wording has drifted since, or the figure came from a pattern the
> retro did not record precisely enough to re-run. So this retro attributes "23"
> to T56 rather than repeating it as a fact, and says "many" where it cannot
> derive a number. This is `t61-retro.md` recommendation 4 applied to someone
> else's figure, and the first evidence that **a quoted method is only as good as
> its exactness** — T56 did the right thing by stating its method and still left
> a figure nobody can re-check.

## 3. The finding this retro exists to record

**T59.2's job was to stop #149 misleading its next reader. #149 still misleads
its next reader.**

T59.2 posted a correction comment and retitled the issue from five facts to one.
Both were done, correctly, and the retitle is genuinely useful. **The body was
not touched.** As of this retro, #149's body still states:

- all five ownership facts, presented as open: *"The **ownership facts** those
  authorization branches compare that actor against are still supplied by the
  same caller"* — followed by the list of five;
- *"The assigned-admin lists are a harder sub-problem: this codebase has never
  persisted Game-Admin or Competition-Admin assignments at all"* — **false since
  T14.4**. `db/migrations/0020_socialplay_game_admins.sql` (T14.4) and
  `0021_competitions_competition_admins.sql` (T15.3) both exist;
- *"That sub-gap should probably be closed first, since the rest of this depends
  on it"* — a dependency ordering built on the false claim above.

The plan **knew about the third one**. T59.2's status note records it:

> A further finding surfaced while writing it: this issue's stated dependency
> ordering … is **also stale** … And `bookings.owner_user_id` has existed as a
> real FK since T55.1/migration 0027 …

So the sprint discovered a false claim in the issue body and wrote the discovery
into **the sprint plan** — a document nobody reads when picking up an issue —
while leaving the false claim in the issue.

### Why this is not simply "they should have edited the body"

Because the choice T59.2 made is defensible, and the reason it fails is more
interesting than carelessness.

Appending a correction rather than editing the body preserves the record: a
reader can see what was believed and what replaced it. That is the same
reasoning **T61 used two sprints later** when it decided *not* to delete the
stale "no Docker daemon" clauses from 17 test files. Both sprints faced the
identical question — *how do you correct a false claim in a document without
erasing the evidence that it was believed?* — and both answered "supersede,
don't delete."

**They then implemented it oppositely.**

| | T59.2 (#149) | T61's sweep (17 test files) |
|---|---|---|
| false claim | left in the body | left in the comment |
| correction placed | in a **separate comment**, below the body | **immediately adjacent**, in the same comment block |
| what a reader hits first | the false claim | the false claim, then the refutation, before anything else |
| can the reader miss it? | **yes** — GitHub renders the body first and a reader may stop there | no |

T58's recommendation 3 says *"When an issue's analysis is corrected, correct the
issue."* T59.2 satisfied the letter — the correction is on the issue — and not
the intent, because **"on the issue" and "where the reader will see it" are not
the same place.** T61's version is the better pattern and T61 arrived at it
without knowing T59 had faced the same choice, which is the part worth fixing:
the two sprints could not learn from each other because neither wrote the
principle down.

### The concrete cost, stated as a prediction this retro can already check

Whoever picks up #149 next reads a body saying four closed facts are open and
that a table created at T14.4 does not exist. The retitle protects them only if
they read the title as authoritative over the body, which is not how issue
bodies are normally read. **#149 has been open since 2026-08-14 and is one of
only three real open issues**, so this is not hypothetical scope — it is one of
the next things anyone picks up.

## 4. The local fix the next sprint had to undo

T59.1 hit a Social Play fixture problem: `fakeIdentityLookup` returned the
subject unchanged, so a booking created through Social Play's test fixtures
carried a non-uuid owner. T59.1 fixed it **locally**, adding a
`resolvingIdentityLookup` used only in booking's own tests.

T60 then retired that stopgap, replacing it with a real per-package
`resolvedUserID` across Social Play's grpcapi fixtures (#307's commit message
says so explicitly: *"retires the local `resolvingIdentityLookup` T59.1 added as
a stopgap"*).

Two readings, and the record supports the second:

- **Uncharitable:** T59.1 patched a symptom and created work.
- **What actually happened:** T59.1 was a two-line shape guard; fixing Social
  Play's entire fixture seam inside it would have been a different, much larger
  ticket in another context. The plan recorded the exclusion **as a deliberate
  scope decision, citing the rule adopted hours earlier at T59.3** — *"a
  deliberate scope exclusion gets a documented decision"*. T60 then closed it as
  its own ticket.

So this is the T57-retro rule working: a chosen gap was documented at the moment
it was chosen, and the next sprint closed it. **Recorded as a success with a
caveat**: the caveat is that the stopgap lived in `booking`'s tests, a context
that had nothing to do with the defect, so for one sprint booking's test suite
carried a fixture compensating for Social Play's. T60's commit notes the payoff —
after the real fix, deleting T59.1's guard breaks only booking's own two tests,
where before it broke five Social Play tests as collateral.

## 5. What went well

- **The premise check was applied on its first available opportunity and
  found something** (§2). Recommendations adopted from a retro frequently sit
  unused; this one was used two sprints after being written.
- **Ceremony 1 verified rather than assumed**, throughout: the live issue count
  (`totalCount: 4`) was reconciled arithmetically against T55's five minus
  #126, merge order was verified against `merged_at` rather than numbering, and
  the 0-ticket counter reset was confirmed rather than stated.
- **T59.1's asymmetry question was answered with a test, not prose.** #296 asked
  whether `CancelBookingsForReference`'s `actorUserID` wanted the same shape
  guard. The answer — it does not, because that value is only ever *compared*,
  never written, so it cannot reach `mustUUID`, and guarding it would convert a
  correct `PermissionDenied` into an `InvalidArgument` that tells an
  unauthorized caller their id was shaped wrong — is pinned as
  `TestCancelBookingsForReference_MalformedActorIsRefusedNotErrored`. A
  deliberate asymmetry recorded as an executable assertion rather than a comment
  is the strongest available form.
- **T59.1 did not pretend to be more than it was.** The plan records the PE
  position verbatim — that this is "a two-line guard against an unreachable
  input" — and the ticket's own test file says no test here can demonstrate a
  production panic, only that the guard exists. A ticket arguing for its own
  small value, in writing, is unusual and correct.

## 6. What T59 believed that has since been falsified

- **T59.1's PR recorded the integration tests as not executed, "no Docker
  daemon".** False; see `t61-retro.md` §2.
- **T59's Ceremony 1 verified `#149`'s premise and corrected its title, and
  recorded the issue as re-scoped.** It is re-scoped in the title only (§3).
  This retro is the correction.

## 7. Recommendations

1. **A correction to a document must sit where the false claim sits, not
   below it.** §3's table is the argument. T59.2 and T61 both chose "supersede,
   don't delete" and implemented it oppositely; T61's — adjacent, unmissable —
   is the one to keep. **Concretely: #149's body should be edited to mark the
   four closed facts and the false "never persisted" claim in place, with the
   original text struck rather than removed.**
2. **A correction found while writing a plan belongs in the artifact it
   corrects, not in the plan.** T59.2 found #149's dependency ordering stale and
   recorded it in `t59-sprint-plan.md` (§3). Nobody picking up an issue reads
   the sprint plan of the sprint that last touched it.
3. **When a ceremony absorbs a ticket, say so in the Docs-index row, not only
   in the plan.** T59.2 and T59.3 were done inside Ceremony 1, correctly and
   explicitly — but the `HANDOFF.md` row says T59 "takes three tickets", which
   reads as three shipped tickets. It took this retro reading the plan's own §In
   to establish otherwise, and the first inference drawn from the commit log
   alone was that T59.2 and T59.3 had never happened. If a row can be misread
   that way by a session with full repository access, it will be misread.

## 8. Sweep and bookkeeping

- **Issues: 4 → 3.** #296 closed by PR #306. No issue opened. Open and
  live-verified at the time: #149, #145, #134.
- **Merge order #304 → #306**, verified against `merged_at`
  (`2026-09-22T12:38:02Z`, then `2026-09-25T11:48:01Z`). Three days apart, with
  no other PR in between — a marked contrast to T55's nine-minute burst
  (`t55-retro.md` §2) and the correct shape.
- **`sprint-process.md` gained five rules at T59.3**, inside PR #304 — verified
  by that commit's diffstat (`docs/process/sprint-process.md | 101 +++++-`).
- **`Closes #N` still cannot fire** against a non-default base branch; #296 was
  closed manually, as every closure on this project has been.
- **Self-review.** Both PRs carry review comments, not approvals.
- **The retro ceremony was skipped**, which is why this document exists at T61.
  T59 is the fourth consecutive sprint without one at the time (T55, T56–T58 had
  theirs written late or in a batch), and the gap ran to T60 as well.
- **This retro does not update `HANDOFF.md`'s T59 Docs-index row.** Per
  `sprint-process.md` a retro PR cannot write the row that points at it. T62's
  Ceremony 1 owns it, along with the §7 recommendation 3 correction to the
  "takes three tickets" wording.

## 9. Honest-form outcome sentence

For `HANDOFF.md`'s T59 row, to be carried verbatim rather than strengthened:

> T59 held the first planning ceremony since T55 and **applied T56's new premise
> check for the first time, where it immediately fired**: #149 was describing a
> world four of its five ownership facts had left, closed by T16.2 and T17.1,
> leaving only `booking_host_id`. A sweep that had been asking "can anyone act on
> this yet?" for many sprints asked "is this still describing reality?" once and
> got no. (Not "23 sweeps": `t56-retro.md` gives that figure with a stated
> `grep -c` method that **does not reproduce at T61**, so this retro attributes
> it rather than repeating it.) One ticket shipped — T59.1, PR #306 (`b7608e3`), closing
> #296 — guarding `Booking.OwnerUserID`'s shape, with #296's symmetry question
> answered as an executable assertion rather than prose: `actorUserID` needs no
> guard because it is only ever compared, and guarding it would turn a correct
> `PermissionDenied` into an `InvalidArgument` disclosing that an unauthorized
> caller's id was shaped wrong. T59.2 and T59.3 were completed inside Ceremony 1
> by design, not omission. **The finding this retro exists for is that T59.2's
> job was to stop #149 misleading its next reader, and #149 still misleads its
> next reader**: the correction was posted as a comment and the title re-scoped
> from five facts to one, but the body still lists all five as open and still
> claims the codebase "has never persisted Game-Admin or Competition-Admin
> assignments at all" — false since T14.4's `0020_socialplay_game_admins.sql`.
> T59.2 found a third stale claim in that body and recorded it in the **sprint
> plan**, which nobody reads when picking up an issue. T61 faced the identical
> question two sprints later for 17 stale test headers, reached the same
> principle — supersede, don't delete — and implemented it the other way, with
> the correction adjacent and unmissable rather than below the body a reader may
> stop at. Neither sprint could learn from the other because neither wrote the
> principle down. **Written at T61, two sprints late**, which is also what makes
> observable the one thing a contemporaneous retro would only have predicted:
> T59.1's deliberately local `resolvingIdentityLookup` stopgap, documented as a
> chosen scope exclusion under a rule adopted hours earlier at T59.3, was
> retired by T60 exactly as intended.
