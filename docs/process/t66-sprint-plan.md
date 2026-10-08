# T66 Sprint Plan — Ceremony 1 (backlog refinement)

Per `docs/process/sprint-process.md`. PM + PE, held against `HANDOFF.md`,
`docs/process/t65-retro.md`, the live issue list, and the tree at `986c904`.

Subject to **T62.2** (every quantity carries its command), **T63.1** (an
action-claim carries the command that re-reads live state), **T63.3** (a guard
is verified by removing it) and **T64.3** (a gate claim carries its date).

**Ceremony order:** security gate (§0) → **the previous retro's
recommendations** (§1, new) → escalations (§2) → bookkeeping (§3) → issue
sweep (§4) → tickets (§5).

---

## §0 — The security gate, run first, and it caught a critical advisory

T64's recommendation 1, applied for the second sprint running. It paid
immediately.

```
2026-10-08T07:40:16Z $ SKIP_GOVULNCHECK=1 make security | tail -4
NEW gating findings (1):
  - [npm-audit] shell-quote (critical): shell-quote: `quote()` command
    injection via a line terminator in a token after a `{ comment }` token
FAIL: 1 new gating finding(s).
```

**PASS at 2026-10-06, `critical` at 2026-10-08, with no code change in
between.** This is T64.3's lesson for the second time in six days and the
first at `critical`: the advisory database moved, not the tree. The rule
exists because T63.2's green gate went red in twenty hours, and *"run the
gate at the start of a ceremony"* is the consequence it recorded.

Fixed before planning continued, in its own PR (#338, merged `b8f240e`):
`shell-quote@1.10.0` is transitive via `npm-run-all2@9.0.3`, the advisory
range `>=1.8.4 <1.11.0` has a same-major fix, and the pin is an `overrides`
entry per `CLAUDE.md`'s npm gotcha. Mutation-verified both ways, which is a
third independent confirmation of the dependency-pin clause: reverting the
**lockfile** returns the advisory by name, deleting the **declaration**
changes nothing.

```
2026-10-08T12:03:55Z $ SKIP_GOVULNCHECK=1 make security | tail -1
PASS: no new gating findings (0 baselined, 2 below threshold).
```

**The Go half still has never run.** `vuln.go.dev` is `Forbidden` from here;
carried as *"does not, as configured"* per T64's recommendation 6, which
`CLAUDE.md:382` still contradicts — see §1.

## §1 — The previous retro's recommendations, walked row by row

**T65's retro recommendation 1, applied to T65's own retro on its first
opportunity.** It exists because T65 reported three of T64's nine
recommendations "silently dropped" and its review found the honest count was
zero — the deferrals were recorded, in a sentence of a sprint plan, which
nothing required and nothing checked.

| # | T65's recommendation | T66 |
|---|---|---|
| 1 | a Ceremony 1 step that walks these | **done — this section is it** |
| 2 | write the mutation-restore procedure into `sprint-process.md` | **ticketed: T66.4.** Earned a third instance this sprint (§4) |
| 3 | review the fixes, not only the work | **adopted as practice** — T65's own retro PR got a review pass of its own, which found five defects including two of its headline claims |
| 4 | generalise the anti-list rule to guards | **ticketed: T66.4**, and §4's #330 is a fresh instance |
| 5 | treat "zero of N findings deferred" as a scope signal | **adopted** — this slate defers four issues explicitly, in §6 |
| 6 | teach `docs-index-check` the `LESSONS.md` stub; ticket the backfill separately | **ticketed: T66.3** (the check) and **deferred with a reason** (the T54–T64 backfill, §6) |
| 7 | finish T64's 4, 6 and 9 | **ticketed: T66.4** |
| 8 | `ci-integration`'s interval is a standing debt | **ticketed: T66.5** |
| 9 | the web suite has no e2e tooling | **ticketed: T66.2**, and §2 is what running it by hand found |

**Nine of nine have a disposition, and every deferral names where it is
recorded.** That is the whole content of recommendation 1, and it took one
table.

## §2 — The app was run, in a browser, for the first time

T65's recommendation 9 said the web suite has no e2e tooling and T65.1 was
the proof it matters. This ceremony ran the app rather than arguing about it:
`cmd/server` on the `dev/auth/` fixture, Postgres with every migration, the
Vite dev server, and Chromium driving the UI.

**Two defects, neither of which any test or gate can see.**

1. **#339 — the dev client cannot reach the API from a browser at all.** No
   CORS headers on the server, no proxy in `web/vite.config.ts`; the client
   defaults to `:8080` and Vite serves `:5173`, so every call is blocked by
   preflight. The app loads and reaches nothing.
2. **#340 — a signed-in user with no account can do nothing and cannot make
   one.** Every write returns `403 facilities: user not found`, and
   `Profile.vue:22` says in as many words: *"NO SIGN-UP FLOW. This ticket
   builds no `CreateUser` UI"*. The capability exists on the server and has
   no door in the client.

**And what the run confirmed, which is the other half of its value.** With
the user bootstrapped by `curl`, the whole path works end to end: sign-in
stores the token, the UI's write lands in Postgres owned by the signed-in
subject, the created facility appears in the list, and a signed-out write
produces `401` plus T65.1's banner —

```
POST /v1/facilities -> 200       # signed in, account exists
POST /v1/facilities -> 403       # signed in, no account   (#340)
POST /v1/facilities -> 401       # signed out → "you are not signed in" banner
$ psql -tAc "select f.name, f.owner_id = u.id from facilities f
             join identity_users u on u.id = f.owner_id"
E2E Courts 1791460846|t
```

**One retraction, recorded because it is the sprint's own rule working.** The
run first reported a third defect — a signed-out write "advancing with no
request made". That was the test's error: `details-next` is pure navigation
and `submitFacility` is bound to the *Photos* step's button, so the script
had never clicked the submitting control. Reading `FacilityOnboarding.vue`
before reporting is what caught it.

## §3 — Escalations

**ADR statuses, from the parser the gate uses:**

```
$ go run ./cmd/docsindex -statuses | tail -2

17 ADR(s): Accepted=17
```

**ADR bodies, per T65.3's step** — the reading list, which is a list and not
a verdict:

```
$ for f in docs/adr/*.md; do n=$(grep -ciE "escalated to (the user|pm/po)|awaiting (the user|product)|unanswered|needs (product|the user)" "$f"); [ "$n" -gt 0 ] && echo "$n $f"; done
2 docs/adr/0009-…   4 docs/adr/0010-…   9 docs/adr/0012-…   5 docs/adr/0015-…   4 docs/adr/0016-…
```

Five files, four discarded by reading them: ADR-0009's question was answered
at T65.3 and the file says no escalation is live in it; ADR-0010 is
superseded; ADR-0015 and ADR-0016 were answered at T55 and preserve their
original wording beneath a supersession notice. **ADR-0012's Q2 is the one
live escalation**, and it stays unasked — it is the protected-attribute
question and nothing in this slate needs it.

**Issue bodies, per T64.5's step**, every open issue read:

| Issue | Awaits a decision? |
|---|---|
| #340 | **YES — a product call about onboarding.** Put to the Product Owner below |
| #339 | **YES, partly** — the dev fix needs no decision; production CORS depends on deployment topology. Put below |
| #335, #334, #333, #330, #329 | No. Engineering options with a stated preference; a ticket settles each |
| #320 | No. The remedy is verified; what remains is execution |
| #149 | No. A design change, unbuilt rather than undecided |
| #145 | No — answered at T64, struck through in the body |
| #134 | No. Blocked on hardware |

### Both were put to the Product Owner before any ticket was finalised

| decision | answer |
|---|---|
| **#340 — first-run onboarding** | **Ask for name and starting level**: an explicit "complete your profile" screen after sign-in, with recovery into it whenever a write returns "user not found". Chosen over silent creation because the self-reported starting level is a claim the player makes and T65.2's formula seeds from it — inventing it defeats the locked cold-start decision |
| **#339 — deployment topology** | **Dev proxy now, decide later**: add the Vite proxy so local development works, and leave production CORS until the topology is decided. A deferral with a trigger: the ticket that decides where the web client is served answers it |

## §4 — Issue sweep: both questions, every issue

Every premise re-verified against the tree. **One has changed, and this
sprint caused it.**

| Issue | Premise, verified 2026-10-08 |
|---|---|
| #340 | **true.** No caller of `POST('/v1/users')` anywhere in `web/src` outside the generated client |
| #339 | **true.** `grep -rniE "cors\|access-control" cmd/server/ internal/platform/` → 0; `grep -c proxy web/vite.config.ts` → 0 |
| #335 | **true.** `EveryDomainSentinel` present in booking/socialplay/competitions/payments, **absent in facilities and identity** |
| #334 | **true.** `-statuses` still reports `Accepted=17` with ADR-0010 superseded |
| #333 | **true.** Unchanged |
| #330 | **TRUE AND WORSE, BY THIS PROJECT'S OWN HAND** — see below |
| #329 | **true.** Unchanged |
| #320 | **true.** `npm -v` → 10.9.7 |
| #149, #145, #134 | **true.** Unchanged |

### #330 got worse because T65.4 made it worse

The issue says the Jenkinsfile hand-maintains a list of what `ci-checks`
covers and omits **6 of 17**. Re-derived:

```
$ comm -23 <(ci-checks prereqs, sorted) <(targets named in Jenkinsfile, sorted)
binary-check docs-index-check fmt-check gate-coverage lock-check test-adapters test-cmd
→ 7 of 18
$ grep -c "binary-check" Jenkinsfile
0
```

**T65.4 added `binary-check` to `ci-checks` and not to the Jenkinsfile.** The
sprint that generalised the anti-list rule to guards, and whose retro made
that recommendation 4, grew the hand-maintained list it is about by one. CI
does not run the gate that catches a committed binary.

That is not an argument for remembering harder. It is #330's own argument,
demonstrated: **the list is the wrong shape**, and T66.1 derives it.

### Compliance check: are T62.2/T63.1/T63.3/T64.3 being followed?

`t62-retro.md` recommendation 2 asks each Ceremony 1 to check, and says the
only evidence that counts is a sprint that did not write the rules. **That
sprint still has not arrived.** What can be said is narrower and new: T65's
retro was audited against the four rules **by a reviewer that did not write
them**, and five claims failed — two of them the retro's headline findings,
and one of them a figure the previous commit had just "corrected" with a
command measured on the wrong tree. The rules discriminate. Whether they are
followable by anyone other than their author is still untested.

## §5 — Tickets

Five tickets, 16 points. **Two are product, two are the gate debts this
sprint's own ceremony surfaced, one is process.**

### T66.1 — The Jenkinsfile's list of gates is hand-maintained, and CI does not run `binary-check`

**Story.** As a maintainer, I want CI to run every gate `ci-checks` runs, so
that a green pipeline means what the local gate means.

**Description.** #330, premise re-derived in §4 and **worse than filed**: 7
of 18 prerequisites are missing from the Jenkinsfile, and the newest omission
is `binary-check`, added by T65.4 four days ago. A hand-written list went
stale inside one sprint — the fourth time this project has watched that
happen (T11, T12, T13/#157, and now this).

**Instructions.**
1. Make the Jenkinsfile **invoke `make ci-checks`**, or derive its stage list
   from the Makefile. Do not hand-edit the comment to add seven names.
2. If a derived list is not achievable, **a gate that fails when the
   Jenkinsfile and `ci-checks` disagree** — `tools/gatecoverage` already
   parses the Makefile and is the precedent for how.
3. **No list in the tool.** Both sides computed at run time, per
   `CLAUDE.md`'s standing rule.
4. Mutation-verify: add a prerequisite to `ci-checks`, confirm the gate goes
   red (T63.3).

**NFRs.** `make ci-checks` stays green. **Points: 3.**
`role:principal-engineer`, `type:chore`.

### T66.2 — First-run onboarding, and a browser that can reach the API

**Story.** As a new player, I want signing in to give me an account I can
use, so that the first thing I try does not fail with "user not found".

**Description.** #340 and #339, both found by §2's browser run and both
invisible to 724 passing tests. The Product Owner answered both (§3): an
explicit "complete your profile" screen collecting display name and
self-reported starting level, with recovery into it on a "user not found"
write; and a Vite proxy now, production CORS deferred.

**Instructions.**
1. `web/vite.config.ts`: a `server.proxy` entry for `/v1`, and make the
   client's default base URL relative. **No CORS middleware on the Go
   server** — that is the deferred half.
2. A first-run profile screen: display name and the self-reported starting
   level, posting `CreateUser`. Only `ROLE_PLAYER` — `ErrRoleNotSelfAssignable`
   already enforces it server-side and the UI must not offer more.
3. A write that returns "user not found" routes the player into that screen
   rather than showing an aggregate's name in an alert.
4. **Do not invent the starting level.** It is the locked cold-start input
   and T65.2's formula seeds from it.
5. **Prove it in a browser, not only in tests** — the whole reason both
   defects survived is that the suite mounts a fixture client.

**NFRs.** The existing web suite stays green. No `Gender` field (ADR-0012
Q2). **Points: 5.** `role:senior-product-engineer`, `type:story`.

### T66.3 — Teach `docs-index-check` the `LESSONS.md` stub

**Story.** As a future ceremony, I want the retro index to be checked rather
than remembered.

**Description.** T65's retro §9b: eleven retros (T54–T64) have no
`## T<N> sprint retro` stub, which Ceremony 3 requires, and
`docs-index-check` never reads the file. Had the check existed at T54 it
would have failed eleven times.

**Instructions.**
1. A fourth derived check: every `docs/process/t<N>-retro.md` has a stub in
   `docs/LESSONS.md`. Both sides from the tree; no list.
2. It will go red immediately on T54–T64. **Land the check and the backfill
   in that order, in separate PRs** — see §6 for why the backfill is not
   bundled here.
3. Mutation-verify by removing T65's stub.

**NFRs.** `make docs-index-check` passes once the backfill lands, and the
interim red is expected and disclosed. **Points: 2.**
`role:principal-engineer`, `type:chore`.

### T66.4 — The three process debts, written down rather than carried

**Story.** As the next session, I want the procedures this project has learned
to be in the rulebook instead of in retros.

**Description.** T65's recommendations 2, 4 and 7, plus T64's 4, 6 and 9 —
six one-to-three-sentence clauses that have each been argued already and none
of which has landed.

**Instructions.**
1. **The mutation-restore procedure** into `sprint-process.md` beside T63.3.
   The load-bearing half is negative: never restore a mutation with `git
   checkout`, `git restore` or `git stash`. Three instances this sprint, the
   third while running the app (§2's teardown killed its own shell).
2. **Generalise the anti-list rule to guards** in `CLAUDE.md` — it is stated
   about `tools/gatecoverage` and is a fact about any check whose subject is
   enumerated by hand. T65 produced three instances; §4's #330 is a fourth.
3. **T64's 4** — the stale-read qualification to T63.1, or an explicit
   decision against it.
4. **T64's 6** — `CLAUDE.md:382` still says the Go half of `make security`
   *"cannot run here"* where the established claim is *"does not, as
   configured"*.
5. **T64's 9** — the squash-ancestry procedure, hit by three sprints running.

**NFRs.** No new claim without its command. **Points: 3.**
`role:principal-engineer`, `type:chore`.

### T66.5 — Run `make ci-integration`, and record the interval

**Story.** As a maintainer, I want to know whether the 27 integration tests
still pass, because the last time anyone checked it was T61 and the answer
was 34 failures.

**Description.** T65's recommendation 8. The suite needs Docker, and **Docker
works here** — this ceremony started the daemon in 2 seconds and ran a real
Postgres against it (§2). "The environment can't" is a claim this project has
already tested and found false.

**Instructions.**
1. Start the daemon, run `make ci-integration`, report the result **with its
   date** whatever it is.
2. If anything fails, that is the ticket — triage before fixing, because T61's
   34 failures contained two live production defects and a lot of noise.
3. Record the elapsed interval since T61 somewhere a ceremony reads, so the
   next gap is visible without an archaeology pass.

**NFRs.** Do not skip, disable or quarantine a failing integration test.
**Points: 3.** `role:principal-engineer`, `type:chore`.

### PM value ordering, if the sprint has to be cut

1. **T66.2** — the product is unusable from a browser in two independent
   ways, and both were invisible until someone opened one.
2. **T66.1** — CI does not run a gate that exists, including the one guarding
   against a defect this project committed four days ago.
3. **T66.5** — the longest-standing unknown in the repository.
4. **T66.3**, then **T66.4** — cheap, and they are what stop the repetitions.

## §6 — Deliberately out of scope, with the decision recorded

- **The T54–T64 `LESSONS.md` backfill.** Eleven summaries of other sprints'
  retros, with real room to misrepresent them. T66.3 lands the check; the
  backfill wants its own ticket and its own reviewer, and bundling them would
  put eleven unreviewed summaries inside a gate's PR.
- **Production CORS (#339's second half).** Deferred by the Product Owner with
  a trigger: the ticket that decides where the web client is served.
- **ADR-0012 Q2.** Deliberately unasked. Nothing here needs it.
- **#320's `vitest` bump, #329, #334, #335, #333, #149, #134.** Four of these
  are gate or tooling work and two are blocked on hardware or a real IdP;
  none competes with a product defect for this slate. **Naming them here is
  T65's recommendation 5 in practice** — a sprint that defers nothing is a
  sprint that is not choosing.

## §7 — The premise challenge

T64's recommendation 8, run again because T65's was the highest-value step of
that sprint.

**The challenge this time was answered by running the app rather than by
reading the slate**, which is a stronger form of the same question: *is this
sprint working on what is actually broken?* The draft slate before §2 was
T66.1, T66.3, T66.4 and T66.5 — **four tickets, zero product, all of them
gates and process**, which is precisely the shape T65's challenge rejected.
Thirty minutes in a browser produced two product defects and the slate now
leads with them.

**The honest limit.** The run found what a first run finds — the two things
standing between a user and any write at all. It did not exercise booking,
games, competitions, payments or the share-link flows, so "the rest works" is
not a claim this ceremony can make. T66.2's instruction 5 is deliberately
about proving one path in a browser rather than about building a suite;
whether the project wants e2e coverage as a standing gate is a bigger
question than one ticket, and it should be asked once there is a second
example.
