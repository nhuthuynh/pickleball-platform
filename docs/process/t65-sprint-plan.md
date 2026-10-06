# T65 Sprint Plan — Ceremony 1 (backlog refinement)

Per `docs/process/sprint-process.md`. PM + PE, held against `HANDOFF.md`,
`docs/process/t64-retro.md`, `docs/process/t64-sprint-plan.md`, the live issue
list, and the tree at `64ad1ce`.

Subject to **T62.2** (every quantity carries its command), **T63.1** (an
action-claim carries the command that re-reads live state), **T63.3** (a guard
is verified by removing it) and **T64.3** (a gate claim carries its date).

**Ceremony order:** security gate (§0, new) → escalations (§1) → bookkeeping
(§2) → issue sweep (§3) → tickets (§4).

---

## §0 — The security gate, run first

`docs/process/t64-retro.md` recommendation 1, applied on its first
opportunity. T64's ceremony found the gate red by running it before planning;
T61's and T62's recorded it as owed and planned anyway.

```
2026-10-06T11:10:54Z $ SKIP_GOVULNCHECK=1 make security | tail -1
PASS: no new gating findings (0 baselined, 2 below threshold).
```

**PASS, dated** — per T64.3's clause, which exists because this exact gate's
verdict changed in under a day. The two below-threshold findings are #320's
`vitest` pair and are T65.1.

**The Go half still has never run on this project.** `vuln.go.dev` is
`Forbidden` from here, and T64's review established the honest wording:
**"does not, as configured"** rather than "cannot" — whether the host can be
allowlisted needs the proxy's own status endpoint, which is denied to this
session and to its agents. Not re-tested here; carried in §5.

## §1 — Escalations

### ADRs: none escalated, from the listing rather than a scratch copy

**First real use of `cmd/docsindex -statuses`** (T64.5), which exists because
T64's own ceremony had to copy the package into a scratch directory to answer
this question without re-implementing the parser as a grep:

```
$ go run ./cmd/docsindex -statuses | tail -2
17 ADR(s): Accepted=17
```

Eleven `## Status` heading form, six front-matter. **Nothing escalated.** The
tool paid for itself on its first ceremony — one command instead of a
twelve-step workaround — which is the cheapest possible vindication of a
3-point ticket.

### Issues: every open one considered, and its **body** read

**First application of T64.5's prose step**, which exists because T62.4's
*corrected* sweep still missed a product question sitting in #145's body. The
step asks, per issue, whether an unanswered question is present — not whether
the issue's labels say so.

```
$ for n in 330 329 328 320 149 145 134; do gh api .../issues/$n --jq .body \
    | grep -inE "needs (product|a decision|input|sign-off)|open question|to be decided|awaiting|decide (whether|what|which)"; done
#328:40  "Then decide what to do about the blind spot, explicitly." Options, none free:
#145:68  ~~Needs product input on which.~~ **Answered at T64's Ceremony 1 …**
```

| Issue | Awaits a decision? |
|---|---|
| #330 | No. Carries three options and a stated preference; a ticket settles it. |
| #329 | No. Two options (fail the gate vs note it); a ticket settles it. |
| #328 | **An engineering decision, not a product one** — what `gate-coverage` should answer about packages with *zero* tests. T65.2 settles it. |
| #320 | No. The remedy is verified (§3); what remains is execution. |
| #149 | No. A design change (a read port into Booking), unbuilt rather than undecided. |
| #145 | **Answered at T64**, as a deferral with a trigger a future sprint can cause. The strike-through above is the step working. |
| #134 | No. Blocked on hardware. |

Three issues carry *engineering* options — which the step surfaces and the old
label-and-state sweep would not have — and each is assigned to the ticket that
settles it.

### ⚠️ And the sentence that was going to go here was false

**The draft of this section read "No product decision awaits an answer."** The
premise-challenge pass (§6) found **two** awaiting one, and both were inside
ADRs this ceremony's own listing reports as `Accepted`:

```
$ for f in docs/adr/*.md; do n=$(grep -ciE "escalated to (the user|pm/po)|awaiting (the user|product)|unanswered|needs (product|the user)" "$f");     [ "$n" -gt 0 ] && echo "$n $f"; done
2  docs/adr/0009-social-channel-integration-deferred.md      # LIVE
4  docs/adr/0012-identity-users-...-escalated-decisions.md    # Q1 LIVE, Q2 deliberately not asked
4  docs/adr/0010-...-deferred-to-identity-context.md          # superseded by 0012
5  docs/adr/0015-booking-ownership-for-public-bookings.md     # answered T55
4  docs/adr/0016-reviewer-authored-code-...-pull-request.md   # answered T55
```

**This is the same one-level-short failure, for the third time, in the sweep
rebuilt to prevent it.** T62.4 moved from *labels* to *the open-issue list*;
T64.5 moved from *an issue's state* to *an issue's prose*; and the ADR half
still reads **only the status line**. T64.5 fixed prose-reading for issues
only, and this ceremony celebrated the ADR half as *"one command"* — which is
exactly what made it shallow.

**ADR-0012 Q1 is not Q2, and `HANDOFF.md` applied Q2's description to both.**

| | question | character |
|---|---|---|
| **Q1** | *"How is the Player Level formula weighted?"* — tenure + win rate | **an ordinary product-weighting question** |
| **Q2** | whether to collect and algorithmically act on a **protected attribute** (gender-mix matching) | genuinely may never be this project's to answer |

`HANDOFF.md`'s "Indefinitely blocked" table carried them as one row —
*"ADR-0012 Q1/Q2 — Legal/ethical dimension … May never be this project's to
answer"* — and `t59-sprint-plan.md:110-113` is where the collapse was written
down, as *"a legal/ethical question"*, singular. T62, T63, T64 and the draft of
this plan all inherited it. ADR-0012's own trigger says the opposite in as many
words: *"If only one of Q1/Q2 is answered, build the part that answer
unblocks."*

**Open since 2026-08-10 — longer than DECISION D1's 41 sprints**, by the same
mechanism, through four consecutive ceremonies with hardened sweeps. It gates
**matchmaking**, a locked v1 scope item.

### Both were put to the Product Owner before any ticket was finalised, and both were answered

| decision | answer |
|---|---|
| **ADR-0012 Q1** — Player Level weighting | **Balance win rate with experience**: win rate is the signal, but a player needs a reasonable number of games before the rating is trusted, and early results move it less. The conventional recreational-ladder choice, chosen over win-rate-dominant for being harder to game and kinder to beginners |
| **ADR-0009** — market scope | **Both, with one built first**: the channel is a pluggable port and one implementation ships first |
| **ADR-0012 Q2** | **Deliberately not asked.** It is the protected-attribute question, and nothing in this sprint needs it. Recorded as separate, which is the thing four ceremonies failed to do |

**One sub-decision stays open and is named rather than guessed: which channel
is built first.** The option the Product Owner chose asks for it, and the
answer was not given. It blocks nothing in T65 — no messaging ticket is in this
slate — so it is put when the messaging ticket is refined, not resolved here by
inference. **That is a deferral with a trigger a future ceremony causes**, the
distinction ADR-0015 lacked.

**Q1's answer has a standing consequence this sprint must honour.** ADR-0012:
*"The sprint immediately following the user's answers … must implement that
answer."* T65 is that sprint. T65.2 below is it.

**And the limit, which this ceremony can state first-hand.** All seven bodies
were written or corrected within the last 24 hours, by this session. The step
cost nothing here and proved nothing either: it was never going to surface a
question this session did not already know about. **Its first real test is a
ceremony reading an issue it did not write**, and that has not happened yet.

## §2 — Bookkeeping

1. **T64's Reviews cell filled**, `merged_at`-verified rather than assumed:

   ```
   $ gh api .../pulls/{325,326,327,331} --jq '"#\(.number) \(.merged_at) \(.merge_commit_sha[0:7])"'
   #325 2026-10-06T07:14:07Z 1febf8e    Ceremony 1
   #326 2026-10-06T07:44:55Z 2e9b59f    all five tickets
   #327 2026-10-06T08:20:45Z bdab076    retro
   #331 2026-10-06T08:32:09Z 64ad1ce    review fixes
   ```

   Its **Retro cell** was already set by the retro PR, per the path/number
   distinction — nothing to correct, the second sprint running.

2. **A T65 row created.**

3. **T64's narrative added**, with **53 blockquote lines lifted
   programmatically** from `t64-retro.md` §13 (3949 chars), not retyped.

4. **The "Open issues" section was corrected at T64's review and is left
   alone here.** It drifted twice in one day — re-derived at T64's Ceremony 1,
   wrong by that afternoon — and the honest reading is that its failure mode is
   structural: **it caches a derived result in prose.** T65.3 covers it.

## §3 — Issue sweep: both questions, every issue

`sprint-process.md`'s premise check. **Fired on zero of seven**, and every
premise was verified against the tree rather than read off the issue:

| Issue | Blocker | Premise, verified |
|---|---|---|
| #330 | none | **true.** Differencing `Makefile`'s `ci-checks` prerequisites against the Jenkinsfile comment: 17 prereqs, 11 named, **6 missing** (`fmt-check`, `test-adapters`, `test-cmd`, `gate-coverage`, `docs-index-check`, `lock-check`) |
| #329 | none | **true.** `0015` and `0016` still carry both status forms with disagreeing tokens |
| #328 | none | **true, and worse than filed.** `cmd/server` has 4 tests; `cmd/devtoken`, `cmd/docsindex`, `cmd/gatecoverage` and `cmd/vulngate` have **zero** — four of five |
| #320 | npm 10.9.7 locally (`npm -v`) | **true**, and its remedy is now verified: npm 12.2.0 resolves the bump in a scratch prefix with `found 0 vulnerabilities` |
| #149 | `internal/payments/port/` has no Booking reader | **true.** `grep -c "BookingHostID: req.GetBookingHostId()"` → 2 |
| #145 | real IdP | **true**, and its product question is answered |
| #134 | AT hardware | **true.** All three routes in `ROUTES_UNDER_TEST` at `:92`, `:98`, `:99` |

**Zero of seven is the expected outcome here and should not be read as the
check working**: every one of these issues was written or corrected inside the
last 24 hours, six of them by T64. A clean sweep over issues this fresh is
arithmetic, not evidence. The check's rate over its four real runs — T59, T62,
T63, T64 — stands at three of four.

### Compliance check: are T62.2/T63.1/T63.3/T64.3 being followed?

`t62-retro.md` recommendation 2 asks each Ceremony 1 to check this, and says
the only evidence that counts is **a sprint that did not write the rule.**

**That sprint still has not arrived**, and saying so is the honest answer.
Every sprint from T62 to T65 was written by one session; the rules' author has
audited the rules' application every time. What *can* be said is narrower and
real: **T64's review audited T64's artifacts against the four rules derived
rather than hand-listed, and every figure that failed was a figure carrying no
command** — eight of them, including two in documents that cite the rules. That
is evidence the rules **discriminate**: they are not satisfied by prose that
merely sounds careful.

**The open question this ceremony will not pretend to answer:** whether the
rules are followable by anyone other than their author. T65.5 does not attempt
it either — a rule's portability cannot be established by the party that wrote
it.

## §4 — Tickets

**The slate was rebuilt after the premise challenge (§6) and the two answers in
§1.** The draft's five tickets were two documents, an internal tool's tests, a
dev-only dependency bump, two internal lists and two process clauses — **zero
of five touching the product**, in a project whose purpose is a pickleball
platform. Four tickets now, 15 points, **two of them product**.

### T65.1 — The Vue client cannot authenticate, so every write path in the shipped UI is `Unauthenticated`

**Story.** As a player, I want to sign in, so that booking a court, joining a
game, paying, entering a competition or editing my profile does something other
than fail.

**Description.** Found by the premise challenge and verified here. The server
enforces authentication on **31 RPCs** across all six contexts; the client
cannot send a token at all.

```
$ grep -rn "Authorization" web/src --include=*.ts --include=*.vue | grep -v api/generated
web/src/composables/useGamePayment.ts:15:/** Authorization facts for a … checkout
                                        # a comment. That is the only match.
$ grep -rln "\.POST(" web/src --include=*.ts --include=*.vue | grep -v __tests__ | wc -l
12
$ for c in booking socialplay competitions payments facilities identity; do \
    awk '/func AuthenticatedMethods\(\) \[\]string \{/,/^\}/' internal/$c/adapter/grpcapi/*.go \
    | grep -c "_FullMethodName,"; done        # 8 9 6 4 4 0
31
$ grep -rn "MOCK_OWNER_ID\|MOCK_PLAYER_ID" web/src --include=*.ts | grep -v __tests__ | wc -l
10
```

Twelve write call sites — book a court, join a game, pay for a game, enter a
competition, create a facility, add a court, create a game, create a
competition, set a discount, request a club rental, approve one, edit a profile
— and ten of them still send `actorUserId: MOCK_*`, a wire field the handlers
have ignored since T12.8. **Owed since T55 (2026-09-04)**, recorded in
`HANDOFF.md`'s "Cross-cutting / later", which **no ceremony step reads**:

```
$ grep -n "Cross-cutting" docs/process/sprint-process.md | wc -l
0
```

**Why no gate noticed, and this is the sharper half.** All 717 web tests mount
with a fixture client, and `web/` has no e2e tooling at all. That is
`CLAUDE.md`'s own best lesson one layer up — *"an in-memory fake is more
permissive than Postgres, and that asymmetry hides storage bugs"* — except the
fake here is the **client**, and what it hides is that the product's entire
write surface is unreachable. The project learned this about Postgres at T61
after 57 sprints and has not applied it to the browser.

**Instructions.**
1. `web/src/state/authSession.ts` — a token `ref` plus `localStorage`,
   mirroring `web/src/state/roleEvidence.ts`, which already exists and is the
   convention to copy rather than invent.
2. Attach `Authorization: Bearer <token>` in `web/src/api/client.ts` via
   `openapi-fetch`'s `.use()` middleware. **Confirm 0.17's middleware API
   first** — `^0.17.0` is declared and 0.17.0 is installed; verify, do not
   assume.
3. A **401 → "sign in required"** branch in the shared error path, not per
   composable.
4. A **dev-only token-entry route** fed by `make dev-token`, which already
   exists and whose `Makefile` comment documents the exact
   `Authorization: Bearer` form.
5. **Delete the `actorUserId: MOCK_*` arguments**, which the handlers ignore.
   Deleting a field the server ignores is safe; leaving it is a lie on the
   wire.
6. **One real end-to-end check, or an honest statement that there is none.**
   717 green tests against a fixture client is precisely what hid this. Either
   drive one authenticated write against a running server (`make up` supplies
   the auth env from `dev/auth/`, `make dev-token` mints the token) and quote
   it, **or** say in the PR that the change is unproven against a real
   gateway — and if so, that gap gets an issue.

**NFRs.** Real-IdP work is **excluded and named**: no redirect flow, no remote
JWKS `KeySource` (#145, #137). This is the dev-fixture token path only, which
is what `dev/auth/` exists for. 61 files / 717 tests stay green.
**Points: 5.** `role:senior-product-engineer`, `type:story`.

### T65.2 — Player Level, per ADR-0012 Q1's answer

**Story.** As a player, I want my level to reflect how I actually do, so that
matchmaking suggests opponents near my standard.

**Description.** §1. Q1 was answered **"balance win rate with experience"**,
and ADR-0012's trigger says the sprint immediately following the answer
implements what it unblocks. **Q2 remains unasked**, so this ships
**level-only** matching and touches nothing gender-related.

**Instructions.**
1. The formula lives in `internal/identity/domain` as a **pure function** —
   rule 2. Win rate is the signal; a confidence ramp makes early results move
   the value less, so a player below the threshold is weighted toward their
   self-reported starting level (the locked cold-start mechanism) rather than
   toward their first result.
2. **Table-driven tests first** (rule 1), and the cases that encode the
   answer: a 1-win player must not outrank a long-run strong player; a
   100%-win-rate newcomer must sit below a proven regular; the value must be
   monotonic in wins at fixed games played.
3. **Pick the threshold and the ramp shape in the PR, with the reasoning**, and
   state that the Product Owner chose the *character* of the formula, not its
   constants. Do not present a constant as though it were decided for you.
4. Automated suggestion must stay **manually overridable** — a locked v1
   decision.
5. **No `Gender` field anywhere** — schema, domain or proto. ADR-0012
   instruction: *"until Q1 and Q2 are answered, no PR may…"*, and Q2 is
   unanswered.
6. Update ADR-0012: Q1 **answered**, with the answer, the date and what it
   unblocks; Q2 **still escalated**, now visibly separate.

**NFRs.** `internal/identity/domain` imports nothing outside the standard
library. Rule 4 applies if any value is persisted: the schema half ships in the
same ticket.
**Points: 5.** `role:principal-engineer`, `type:story`.

### T65.3 — Record both answers where the next reader meets them

**Story.** As a future ceremony, I want these two decisions to be findable
without re-deriving them, so the next sweep does not report them absent for a
fifth time.

**Description.** The bookkeeping half of §1, and it is not optional: the reason
Q1 went unasked for 55 sprint-labels is that **its record said it was
unanswerable.**

**Instructions.**
1. **ADR-0009**: status and resolution updated with the market-scope answer
   (both markets; pluggable channel; one implementation first) and the named
   open sub-decision (**which one first**), with its trigger.
2. **`HANDOFF.md`'s "Indefinitely blocked" table**: split the `ADR-0012 Q1/Q2`
   row. Q1 → answered, with its answer. Q2 → stays, with Q2's own description
   and **not Q1's**. Say that the collapse is what hid an answerable question.
3. **`sprint-process.md`**: the escalation sweep must read an **ADR's body**,
   not only its status line — the same step T64.5 added for issues, which the
   ADR half never got. Carry the derived grep from §1 as the worked example and
   the three-step history (labels → issue list → prose) as the reason.
4. **Also fix what the listing got wrong**: `cmd/docsindex -statuses` reports
   `Accepted=17`, and **ADR-0010 is `Accepted … Superseded by ADR-0012`** — one
   status line carrying two tokens, so the tally overcounts live decisions. Not
   what #329 covers (that is two *forms* disagreeing); add it there.

**NFRs.** `make docs-index-check` stays green. No new claim without its command.
**Points: 2.** `role:principal-engineer`, `type:chore`.

### T65.4 — #328: tests for `cmd/docsindex`, and decide what `gate-coverage` owes

**Story.** As a maintainer, I want the 52 lines T64.5 shipped untested to be
tested, and to know whether the project's standing answer to "what is
untested?" covers `main` packages at all.

**Description.** #328, premise re-verified in §3 and **worse than filed**: four
of five `cmd/*` packages hold zero tests (`cmd/server` has 4), so
`gate-coverage: OK` is silent about all of them by construction. Rules 1 and 8
were not met by T64.5 — which makes this the one piece of self-maintenance that
survives the premise challenge, because it is a rule violation rather than a
tidiness.

**Instructions.**
1. Test `-statuses`: the listing, the tally, and above all the **non-zero exit**
   on an unclassifiable status.
2. **Mutation-verify each** (T63.3), quoting the failure.
3. **Decide the blind spot explicitly and record it**: say so in `CLAUDE.md`, or
   add a separate zero-test-package report, or make `main`-package exemption an
   explicit convention.
4. **No exclusion list in `tools/gatecoverage`.**

**NFRs.** `make gate-coverage` must still pass. Confirm `test-cmd`'s `./cmd/...`
pattern picks the new test up without a Makefile edit — a step, not an
assumption.
**Points: 3.** `role:principal-engineer`, `type:chore`.

### PM value ordering, if the sprint has to be cut

1. **T65.1** — twelve write call sites, behind 31 enforced RPCs, are
   unreachable from the browser and have been for ten sprints. Nothing else
   here is a user-visible defect.
2. **T65.2** — a standing ADR commitment triggered by an answer given today,
   and the first new capability since T58.
3. **T65.3** — cheap, and it is what stops the fifth repetition of §1's
   failure.
4. **T65.4** — a rule-1/rule-8 violation from last sprint.

**Deferred to T66, deliberately**: the `CLAUDE.md`/`HANDOFF.md` structural
rewrite (the draft's T65.3 — see §6 for why it was cut rather than reordered),
#320's `vitest` bump (dev-only, below the gate's threshold, and it changes which
npm the project targets — the challenge made the case that 3 points is
optimistic), #329/#330, and T64's two process debts. **Four of those five are
process or tooling work, which is the point.**

**PE sign-off, with one sequencing constraint.** T65.2 and T65.3 both touch
ADR-0012; T65.2 first, so T65.3 records an answer the code already honours.
T65.1 and T65.4 are independent.

**Dependency-completeness check** (both questions):

| Ticket | Producer exists? | Consumer can reach it? |
|---|---|---|
| T65.1 | `dev/auth/` fixture, `make dev-token`, `cmd/devtoken`, and a server that enforces 31 methods | **the open question is the client**: whether `openapi-fetch@0.17.0` exposes `.use()` middleware. Instruction 2's own first step, not asserted here |
| T65.2 | `internal/identity/domain` exists; the self-reported starting level is the locked cold-start input | **to be confirmed by the ticket**: that match history is actually *stored* in a form the formula can read. If it is not, the ticket's first finding is that Q1's answer unblocks a formula with no input, and that is worth knowing on day one |
| T65.3 | ADR-0009, ADR-0012, `HANDOFF.md`, `sprint-process.md` all exist | yes — prose |
| T65.4 | `tools/docsindex` fixtures; `ADRStatuses` exported | **to be confirmed**: `test-cmd`'s pattern picking up a new test |

**Three gaps, all named as first instructions** rather than resolved here. T65.2's
is the one that could change the ticket: per T15.5, a planning check that asserts
an unread capability is how a ticket gets cleared and then hits a wall.

## §5 — Deliberately out of scope, with the decision recorded

- **#149's read port into Booking.** A genuine design change and the last
  caller-supplied ownership fact; it wants its own sprint.
- **#134's WCAG pass.** Premise re-verified; needs an operator and a real
  screen reader.
- **Whether `lock-check` should detect a superset lockfile.** Needs a policy
  about which resolved versions are load-bearing — i.e. a list — and the honest
  answer may be no. T64's review recorded it in the gate's own OK message
  rather than as an issue.
- **`govulncheck`.** `Forbidden` from here, and the one question that would
  settle whether that is permanent (reading the proxy's status endpoint) is
  **denied to this session and to its agents**. Carried as *"does not, as
  configured"*. **This is the project's longest-standing unchecked
  inaction-licensing claim and the only one it cannot check itself.**
- **A gate for T65.3's rewritten sections.** Rejected in T65.3 instruction 5.

## §6 — The premise challenge (T64 retro recommendation 8)

T64's retro recorded that its three reviewer passes raised 42 findings and
**none of them asked whether the sprint's tickets were the right work.** Every
pass checked whether the work was done correctly. Recommendation 8 was to brief
an agent on the **premise** instead.

**This plan is the first subject to that**, and the brief is deliberately
hostile: argue the slate is self-referential, argue the ordering is wrong,
argue T65.3 is the wrong fix, name the product work that should displace
something, and argue the sprint is too big. The agent was given the plan before
it was proposed for merge, so its objections can change the tickets rather than
annotate them.

### Verdict: the slate was wrong, and the ceremony's own steps are why

The pass's one-paragraph verdict, which this ceremony accepts:

> The five tickets are individually well-formed, correctly premise-checked, and
> correctly sized — and collectively they are the wrong sprint.

**Not a judgement failure. A sourcing failure, and a structural one.** Ceremony
1 sources tickets from open issues, the previous retro's recommendations, and
escalated ADR statuses — **all three artifacts the process generates about
itself.** The one place this project parks product work it has chosen not to do
is `HANDOFF.md`'s "Cross-cutting / later", and no ceremony step reads it:

```
$ grep -n "Cross-cutting" docs/process/sprint-process.md | wc -l
0
```

**And the 0-ticket cap does not fire**, because it counts tickets rather than
product tickets, and *"a sprint that takes even one ticket resets the count"*.
T65's draft took five. The cap — adopted precisely because T30–T53 were 24
consecutive sprints of ceremony documents — was satisfied while the failure mode
it was written against was fully intact. The cap's own text is the argument:
*"Every one of those ceremonies did what this document told it to do, and did it
honestly … That is exactly why a cap is needed rather than better judgement."*
Replace "0-ticket" with "0-product-ticket" and it describes this plan's draft.

**T65 would have been the seventh consecutive sprint with no new user-facing
capability** — the last was T58's per-head entry fees — and the first of the
seven with no product surface to stumble over: T61 and T62 at least found live
defects while doing infrastructure work.

### What changed, and what was defended

| objection | outcome |
|---|---|
| the slate is self-referential, zero of five touch the product | **accepted.** Rebuilt: 4 tickets, two of them product |
| the client cannot authenticate; owed since T55; invisible to every sweep | **accepted and taken** as T65.1 — verified independently here (31 enforced RPCs, 12 write call sites, 10 `MOCK_*` arguments, zero `Authorization` headers) |
| T65.3's "every session pays on arrival" is true of 34 lines and false of 239 | **accepted.** The `CLAUDE.md` section is auto-injected; the 239-line `HANDOFF.md` section is opt-in, and most of the 5 points were in the opt-in half |
| T65.3's NFR is vacuous | **accepted, and it is the sharper point.** `docsindex`'s only narrative content check is `stalePhraseRe = (?i)retro not yet written`, which occurs **zero** times in the section being rewritten. The ticket's only stated verification could not fail — in a sprint whose predecessor's retro is largely about vacuous green |
| deleting a section of the instruction file is a product-owner call | **accepted.** Deferred rather than taken; if it returns, it returns as a question, not a ticket |
| §1's "no product decision awaits an answer" is false, twice | **accepted** — see §1. This is the pass's most valuable finding and the reason the brief existed |
| 16 points is too big | **partly.** The arithmetic is within precedent (T62 took 17 and delivered five); the composition was the problem — 5 of 16 points were work no gate in this project can check |
| promote T65.1 (`vitest`) instead | **declined, with its own evidence**: both findings are in the test runner, 29 production dependencies, zero affected, nothing in the shipped bundle. The pass argued this against itself and was right to |
| #330's instruction, T65.4, T65.5, §0, and §1/§3's self-discipline | **defended by the pass and kept** — #329/#330 and the process debts move to T66 on priority, not on merit |

### Two things the pass got wrong, checked before they were repeated

1. **It said `internal/payments/port/` holds no Booking reader and 13
   interfaces** — correct. But it also said *"none of `db/queries/booking.sql`
   is owner-scoped"*, and my first check (`grep -c owner_user_id` → 6) appeared
   to refute it. Reading the file settles it in the pass's favour: all six hits
   are column lists in `INSERT`/`SELECT`/`RETURNING`, and
   `grep -c "WHERE.*owner_user_id"` → **0**. The index at `0027:79` serves no
   query. **A count was nearly used to refute a claim it did not address** —
   the exact failure this project keeps finding, one level in.
2. **It said 33 authenticated RPCs; the count is 31** (8+9+6+4+4+0). Its
   decomposition put 2 in booking and 2 in identity; the real split is 8 and 0.
   Immaterial to the argument, and recorded because an uncorrected figure is how
   the next document inherits it.

### What this says about the review arrangement

T64's retro said a reviewer agent *"cannot ask why is this sprint's premise the
right premise"*, and listed the question nobody asked. **Briefed to ask exactly
that, it did** — and found a false sentence, a 55-sprint escalation failure, a
structurally self-referential sourcing step, and a vacuous NFR, in a plan that
had already been written to this project's own rules.

So the limit in T64 §9 was **narrower than stated**: the constraint was not
that an agent cannot challenge a premise, it was that nobody had asked one to.
What remains true is that this one was pointed at a plan by its author and told
which five objections to attack — it did not choose its own target. That is the
next thing to test.



## §7 — What this ceremony produced

1. **The security gate run first, and dated** — PASS, where T64's ceremony
   found it red by doing the same thing one sprint earlier.
2. **Two live product decisions found, raised and answered** (§1) — ADR-0012
   Q1, open since 2026-08-10 and therefore longer than D1's 41 sprints, and
   ADR-0009's market scope, escalated at T7. Both sat inside ADRs this
   ceremony's own listing reports as `Accepted`, and **four consecutive
   ceremonies with progressively hardened sweeps declared no such decision
   existed.**
3. **The reason, stated structurally rather than as an oversight:** the
   escalation sweep has now been fixed three times and each fix stopped one
   level short — labels → the open-issue list → an issue's prose — and the ADR
   half still reads only a status line. T65.3 carries it.
4. **Q1 and Q2 separated.** `HANDOFF.md` carried them as one
   "legal/ethical, may never be ours to answer" row; Q1 is an ordinary
   weighting question and ADR-0012 says in as many words that answering one
   unblocks its half.
5. **The slate rebuilt after a premise challenge** (§6): from five tickets with
   **zero** product work to four with two, including a client that cannot
   authenticate against 31 enforced RPCs and has not been able to since T55.
6. **The 0-ticket cap shown to be satisfiable while its failure mode is
   intact**, because it counts tickets rather than product tickets — and the
   place this project parks deferred product work is read by no ceremony step.
7. **T64's row completed**, recording that #326 and #331 carry the first
   non-author reviews in this project's history.
8. **Four tickets**, 15 points, three dependency gaps named as first
   instructions, and the compliance question answered honestly: the sprint that
   did not write these rules has still not happened.

## §8 — Recommendations for T66's Ceremony 1

1. **Add a sourcing step: name the highest-value unbuilt *product* capability
   and say why it is not this sprint's work.** §6's finding is that the
   ceremony's three ticket sources are all self-generated. One sentence per
   ceremony makes the omission visible, which is all any rule in this document
   does.
2. **Make the 0-ticket cap count product tickets**, or add a second counter
   that does, with a threshold and an action — per this document's own rule
   that a counter without those is bookkeeping.
3. **Read ADR bodies in the escalation sweep** (T65.3 instruction 3 does this;
   T66 should check it was actually followed by a ceremony that did not write
   it).
4. **Put the deferred structural doc rewrite to the Product Owner as a
   question, not a ticket** — deleting a section of the instruction file is
   their call, and §6 is the argument for asking rather than doing.
5. **The one open sub-decision: which messaging channel is built first.** Put
   it when the messaging ticket is refined. It blocks nothing now, and the
   trigger is a ceremony's own act rather than an event nobody can cause.
