# ADR-0012: Identity/Users and Match are built in T10; PlayerRating, the matching algorithm, and gender-mix matching remain blocked on two escalated product/legal decisions

## Status

**Accepted (T10 Ceremony 1, 2026-08-10). Supersedes ADR-0010. Q1 answered
2026-10-07 and built in T65.2; Q2 still escalated — awaiting the user's
decision. See "Q1 answered" below; the two questions are separate and must
not be carried as one row again.** ADR-0010's
sequencing analysis (§(a)–(c): why `Level` is structurally an Identity
concept, why parking it in the wrong context is a bill this project has
already paid once, why T9 had no call site to justify building early) is
**not overturned and remains the authoritative record of that reasoning** —
this ADR does not relitigate it. What changes is the trigger's resolution:
ADR-0010 required T10's Ceremony 1 to either build auto-matching in full or
supersede it with a new ADR stating a new decision and its own trigger.
This is that new ADR.

## Q1 answered (2026-10-07), built in T65.2 — and Q2 deliberately not asked

**Amendment, not a rewrite.** Everything below the next heading is the
original ADR as accepted on 2026-08-10 and is left standing; this section
records the trigger firing.

### The answer

> **Balance win rate with experience.** Win rate is the signal, but a
> player needs a reasonable number of games before the rating is trusted,
> and early results move it less.

Put to the Product Owner at T65's Ceremony 1 and answered the same day. The
options offered alongside it were win-rate-dominant (rejected as easier to
game and harsher on beginners) and tenure-dominant (rejected as rewarding
attendance rather than standard).

**A second, smaller product answer arrived with it**, because T65.2 could
not count a win without it: *highest points wins, and a tie counts for
everyone tied.* That answer needed no change to what is stored — partners
in doubles share one side's point total, so both appear in a Match's `Score`
with the same value and either both win or neither does.

### What that answers, and what it does not

The answer fixes the formula's **character**. It does not fix its
**constants**. `ConfidenceGames = 20` and the linear shape of the ramp were
chosen by the engineer who wrote `internal/identity/domain/player_level.go`,
with the reasoning stated on each declaration, and either may be retuned
without going back to the Product Owner. What *would* need sign-off again is
changing the character — making win rate not the signal, or removing the
ramp. The code says so in as many words, so a future reader cannot mistake a
tuned constant for a decided one.

### What was built (T65.2)

Per the trigger's own clause — *"If only one of Q1/Q2 is answered, build the
part that answer unblocks (e.g., an answered Q1 with an unanswered Q2 ships
level-only automated matching, still with no `Gender` field)"*:

| Piece | Where |
|---|---|
| The Level formula — a pure function, `seed + c*(observed − seed)` with `c = min(games, 20)/20` and `observed = 1 + winRate*4` | `internal/identity/domain/player_level.go` |
| `Provisional`, so a value still partly made of a player's own claim is never presented as a measurement | same file, `PlayerLevel` |
| Manual override that **survives a recompute** (`RecomputeLevel`), which is what makes `CLAUDE.md`'s "always manually overridable" true rather than vacuous | same file |
| The win rule — `Match.Winners()` / `Match.Won()`, ties counting for everyone tied | `internal/socialplay/domain/match.go` |
| Level-only automated match suggestion, with **pinning**: an organiser's own pairings are carried through untouched and the algorithm arranges only the rest | `internal/socialplay/domain/matchmaking.go` |

**No `PlayerRating` field or table was added.** The decision's §4 list
forbids one — and §4's "compute a Level from `Match` history" clause is the
one Q1's answer releases, so §4 has been amended below rather than left to
contradict this section. (That contradiction was a review finding: §4 as
written forbade this ticket outright while the trigger required it.)
Nothing here needs a stored rating: a `PlayerRecord` is a count of games and
wins, and a `PlayerLevel` is computed from it on demand. Rule 4's Postgres
half is therefore not owed by this ticket — there is no persisted derived
value for a constraint to protect. The ticket that *stores* a level owes the
schema half, and owes it in the same ticket.

### Q2 is unchanged, and is now visibly separate

**Q2 — is gender-mix matching in scope at all? — was deliberately not put to
the Product Owner**, and nothing in T65 needs it. It is the
protected-attribute question: whether this platform should collect and
algorithmically act on `Gender`, in the jurisdictions it launches in. It
remains escalated, and the §4 prohibition on a `Gender` field anywhere
remains in force. T65.2 asserts it in two packages —
`TestNoGenderFieldAnywhereInThisRepository`, which parses every Go file's
field and type declarations and scans every migration and `.proto`, and
`TestNoGenderAnywhereInThisPackage` for Social Play's own source — rather
than leaving it to a reviewer's eye. Both parse the tree.

**The first versions listed types by hand, and two review passes were
needed to make them worth having.** The first demonstrated two bypasses by
running them (a new `Gender`-bearing type in the same package, and a
`Gender` field on `socialplay.Registration`) and observed that the tests
asserted nothing at all about the schema or any proto. The second, against
the rewritten guard, demonstrated five more — a Go package in a directory
the walk skipped by basename; a column hidden behind a `DEFAULT
'https://…'`; `Sex` and `BiologicalSex`, the same attribute under another
name; a `const`/`var`/`func`; and an embedded field — plus one false
positive, where a migration documenting its own compliance *inside a block
comment* turned the gate red. All seven are now cases in the suite, and the
scan covers `.go` declarations, `.sql`, `.proto`, `.ts` and `.vue`.
*(Earlier wording here credited the first pass with four demonstrations; it
ran two and reasoned about the other two. The distinction is the kind this
project's rules exist to keep.)*

### Why this sat unanswered for 55 sprint-labels, which is the part worth keeping

Q1 is an **ordinary product-weighting question**. Q2 is a legal/ethical one
that may never be this project's to answer. `HANDOFF.md` carried them as a
single "Indefinitely blocked" row described in Q2's terms —
*"Legal/ethical dimension … May never be this project's to answer"* — and
`docs/process/t59-sprint-plan.md:110-113` is where the collapse was written
down, as *"a legal/ethical question"*, singular. Four consecutive
ceremonies (T62, T63, T64 and T65's own first draft) inherited it and
reported no product decision awaiting an answer. The question was answerable
the whole time; its **record** said it was not.

The ADR's own text said the opposite in as many words — *"If only one of
Q1/Q2 is answered, build the part that answer unblocks"* — and no sweep read
it, because the escalation sweep read only each ADR's **status line**. T65.3
is the fix: the sweep reads an ADR's body, the same step T64.5 added for
issues.

### The input gap, named rather than invented

The formula is pure and tested, and **nothing in this repository can yet
build a real player's `PlayerRecord`.** Three gaps, each verified against
the tree rather than assumed:

1. No query reads one player's match history — `db/queries/socialplay.sql`
   has `CreateMatch` and `ListMatchesForGame` only.
2. Social Play's player ids are opaque `registrations.player_id` strings,
   not `identity_users.id`; `match.go`'s own comment records that "there is
   no Users bounded context wiring these ids to real UUIDs yet".
3. There is nowhere to put a computed level, by design — see the
   no-`PlayerRating` note above.

Filed as #333 rather than built here: T65.2's scope is what Q1's answer
unblocks, and a cross-context identifier bridge is a design change that
wants its own ticket. Shipping the formula now is what the trigger requires;
pretending it has an input would not be.

## Why this isn't exit (a) as originally framed, and isn't a fourth deferral either

ADR-0010's trigger text anticipated this exact situation and pre-empted the
easy way out of it:

> If the open questions below are still unanswered when T10's Ceremony 1
> runs, that does not license a fourth roll-forward: it licenses exit (b).

Q1 (Player Level formula weighting) and Q2 (whether gender-mix matching is
in scope) are still unanswered — they are product/legal decisions this
ceremony has no authority to make on the user's behalf, per
`docs/agent-operating-handbook.md` B5 (PO) and B6 (BA), and per this
sprint's own explicit instruction not to have any role invent an answer.
So exit (a), "build auto-matching in full," is not honestly available:
ADR-0010 itself states Q1 "blocks... the shape of `PlayerRating` and
`Level` themselves — whether they are one field or two," and that
"building storage shaped by a guessed formula is how the wrong shape
becomes load-bearing." Guessing here would be exactly the mistake ADR-0010
was written to prevent, just moved one level down — from "which context
owns this" to "what does this field actually mean."

Exit (b) is therefore the honest choice. But **"supersede" does not mean
"defer again with a new number."** The difference this ADR is required to
make concrete, per the same trigger text: **build everything that does
not require Q1 or Q2's answer, name precisely what remains blocked and
why, and set a trigger tied to an external event (the user answering),
not to another ceremony's judgment call** — which is the failure mode
that let this roll forward three times before ADR-0010 existed.

## What "auto-matching" decomposes into, and which pieces are blocked

Re-reading the locked decision precisely (`CLAUDE.md`): *"Matchmaking:
automated from history, always manually overridable; new players seeded
by a self-reported starting level."* And the glossary
(`docs/agent-operating-handbook.md` A2): `PlayerRating` is "a player's
DUPR-style internal rating, derived from `Match` history; seeded by
self-reported starting level before any history exists." Four distinct
pieces are bundled inside "auto-matching," and they are not equally
blocked:

| Piece | Blocked by Q1/Q2? | Decision |
|---|---|---|
| A user profile that can hold a self-reported starting level, and roles (player/host/game-admin/facility-owner/club/platform-admin per A1) | No — the *raw self-reported value* is the locked decision's own cold-start mechanism, not the tenure+win-rate formula Q1 asks about | **Build (T10.1–T10.2)** |
| Recording `Match` results (players, score, timestamp) against a Game | No — a match result is a fact, independent of any formula that might later consume it | **Build (T10.3–T10.4)** |
| `PlayerRating`'s derived value and update algorithm, and any "Level" score computed from it | **Yes — Q1 explicitly** ("the shape of `PlayerRating` and `Level` themselves... whether they are one field or two") | **Not built. Named, not silently dropped.** |
| Automated match suggestion using that rating, manually overridable | Yes, transitively — there is no rating to match on yet | **Not built.** |
| Gender-mix matching (collecting `Gender`, a matching-mode flag) | **Yes — Q2 explicitly** (collecting and algorithmically acting on a protected attribute) | **Not built. No `Gender` field anywhere in this sprint's schema, domain, or proto.** |

This is not "build the easy 80% and call it done" — it is the literal
boundary ADR-0010 already drew between "storage" (buildable) and "the
formula" (not). The two pieces on the blocked side are genuinely the
entire remaining scope of "matching," which is why this ADR does not
claim auto-matching ships in T10. It claims Identity/Users — the context
ADR-0010's whole argument turned on not existing — now exists, with the
one field the locked decision specifies (self-reported level) real and
seeded correctly, and that this happened without repeating the
`games.facility_id` mistake a second time.

## Decision

1. **Build Identity/Users** (`internal/identity/{domain,app,port,adapter}`
   + `proto/pickleball/identity/v1`, mirroring `booking` exactly per
   `CLAUDE.md`): a `User` aggregate with `ID`, `DisplayName`, `Roles`
   (`player | host_organiser | game_admin | facility_owner | club |
   platform_admin`, per A1), and `SelfReportedStartingLevel` (the raw
   value a player sets at signup — no formula, no weighting, nothing Q1
   touches). T10.1–T10.2.
2. **Build `Match`** in Social Play (per A1's existing context ownership —
   `Match`/`PlayerRating`/matchmaking are Social Play concepts; only the
   self-reported level is Identity's, connected via a new
   `port.IdentityLookup` mirroring T8.3's `port.FacilityLookup` pattern,
   not a shared-kernel shortcut): a `Match` records a result against an
   existing Game. No `PlayerRating` field, no rating computation. T10.3–T10.4.
3. **Do not build** `PlayerRating`'s derived value, any rating-update
   algorithm, automated match suggestion, a `Gender` field, or a
   matching-mode flag, in any context, this sprint. Existing UI
   disclosures ("matching isn't available yet," T8.8/T8.9) are updated to
   state precisely why — Identity/Users now exists, the two blocking
   product questions do not — rather than repeating a generic "coming
   soon." T10.5.
4. **Concretely, until Q1 and Q2 are answered, no PR may:** add a
   `PlayerRating` field/table anywhere, add a `Gender` field/table
   anywhere, add a matchmaking RPC/request field/UI control that implies
   matching happens, or compute a "Level" from `Match` history. This
   restates ADR-0010's own "concretely, no PR may" list, narrowed now
   that `Level` (self-reported) and `Match` (raw results) have moved from
   "blocked" to "built."

   **AMENDED 2026-10-07 — Q1 is answered, so this list is narrowed again.**
   T65.2's review found that §4 as written forbids the very thing that
   sprint was required to build, and that the amendment above then cited §4
   *in the other direction* to justify not persisting anything. A reader
   could not tell whether §4 was in force. It is, minus what Q1's answer
   released. Each clause, and where it stands now:

   | Clause | Status |
   |---|---|
   | compute a "Level" from `Match` history | **released by Q1's answer.** This is the piece the decomposition table attributes to "Q1 explicitly", and the trigger requires the next sprint to build it. T65.2 did |
   | add a `PlayerRating` field/table anywhere | **still in force, and not merely tolerated — honoured.** There is one value, computed on demand, and nothing persisted. Rule 4's Postgres half is therefore genuinely not owed by T65.2. A later ticket that caches a level must revisit this clause explicitly rather than quietly outgrow it |
   | add a matchmaking RPC / request field / UI control that implies matching happens | **still in force.** T65.2 ships pure domain functions and no RPC, no proto field and no UI control. Not as caution: there is no readable input yet (#333), so a control would imply a capability that does not exist |
   | add a `Gender` field/table anywhere | **still in force, unchanged, because Q2 is unanswered.** Asserted by tests in two packages, one of which parses every Go file's field declarations plus every migration and `.proto` — the listed-type version of that guard let a new `Gender`-bearing type through, which is why it was replaced |

## Trigger condition

**The sprint immediately following the user's answers to both Q1 and Q2
must build `PlayerRating`, the rating-update algorithm, automated
match-suggestion (manually overridable), and — if and only if Q2's answer
is yes — the `Gender` field and matching-mode flag.** Unlike ADR-0010's
trigger ("the next Ceremony 1," which had already rolled forward three
times before ADR-0010 gave it teeth), this trigger is tied to an event
outside any ceremony's own judgment: the user's answer arriving. A future
Ceremony 1 that has the answers in hand and still doesn't build is subject
to the same rule ADR-0010 established — defer again in prose and a
reviewer may block on that basis alone.

If only one of Q1/Q2 is answered, build the part that answer unblocks
(e.g., an answered Q1 with an unanswered Q2 ships level-only automated
matching, still with no `Gender` field) rather than waiting for both.

## Open questions escalated to the user — Q1 ANSWERED 2026-10-07, Q2 still open

As posed by ADR-0010 and restated here. This ADR resolved neither when it
was accepted; **Q1 has since been answered** (see "Q1 answered" above) and
Q2 has not. They are listed separately and described in their own terms,
because carrying them as one row is what hid Q1 for 55 sprint-labels.

**Q1 — How is the Player Level formula weighted?** Tenure + win rate,
per `docs/design/handoff-2026-08/README.md:83`; also unresolved since
`docs/design/v1-system-design.md` §5/§7 Q6 whether "Level" is a
player-facing tenure+wins score, a restatement of the internal
DUPR-style `PlayerRating`, or both.

> **ANSWERED 2026-10-07: balance win rate with experience** — win rate is
> the signal, but a player needs a reasonable number of games before the
> rating is trusted, and early results move it less. Built in T65.2. The
> §5/§7 Q6 half is answered by the same ticket's shape: there is **one**
> value, a Level on the self-reported 1..5 scale, computed on demand from
> match history and seeded by the self-reported level — not two fields, and
> no separately stored `PlayerRating`.

**Q2 — Is gender-mix matching in scope at all?** Requirement #15 and Flows
3/4 of the design handoff show it; the question is whether collecting and
algorithmically acting on a protected attribute is something this
platform should do, in the jurisdictions it will launch in — a
product/legal call, not an engineering one.

> **STILL ESCALATED — awaiting the user's decision.** Deliberately not
> asked at T65's Ceremony 1: nothing in that sprint needed it, and asking a
> legal/ethical question to clear a backlog row is how a decision gets made
> badly. The §4 prohibition on a `Gender` field anywhere remains in force
> and is asserted by tests in two packages. **This is the live escalation in
> this file**; Q1's text above is history.

Neither question blocked what this ADR decided to build (Identity/Users,
`Match`). Q1's answer unblocked T65.2; Q2 still blocks gender-mix matching
and nothing else.

## Not a scope reversal

Same statement ADR-0010 made, still true: this does not reopen, weaken, or
cancel `CLAUDE.md`'s locked decision that matchmaking — automated from
history, always manually overridable, cold-start seeded by self-reported
level — is in v1 scope. It remains a v1 requirement. This ADR is about
what is buildable *this sprint* without guessing at two answers only the
user can give.

## What would change this decision

Same three conditions ADR-0010 named, restated because they still apply
verbatim: a real ticketed call site pulling `PlayerRating` in earlier than
its own trigger; Identity/Users itself slipping past this sprint (it
doesn't — see Consequences); or the user answering Q1/Q2 with a request to
build sooner than the trigger implies (a direct instruction is new
information). Would **not** change it: elapsed time, a "minimal" rating
offered as low-risk (rejected on ADR-0010's own `games.facility_id`
evidence, unchanged), or a mockup/UI showing matching controls.

## Consequences

**Pros.** Identity/Users exists as a real bounded context for the first
time, closing the structural blocker ADR-0010's entire argument rested on
— and it closes it without guessing at either open question. `Match`
recording is real and immediately useful (a fact worth capturing
regardless of how it's later scored). The three-times-repeated
`actor_user_id`-is-a-claim-not-authentication caveat (Social Play,
Payments, Facilities) now has a real context to eventually anchor
authentication to, unblocking that work's prerequisite even though this
ADR does not build authentication itself. The host/venue display-name
gap (a T10 follow-up, see the sprint plan) gets a real field to join
against for the first time.

**Cons.** "Auto-matching," as a user-visible feature, is still not
shipped after two full sprints (T9, T10) that both touched the question —
disclosed honestly in-product rather than silently, per the existing
T8.8/T8.9 pattern, but a real product gap nonetheless. `PlayerRating` and
`Gender` stay open, meaning a third sprint could plausibly pass with
"auto-matching" still not user-visible if the user's answers don't arrive
before T11 planning — which is precisely why this ADR's trigger is tied
to the answers landing, not to another sprint boundary, so that risk sits
on external input, not on this project quietly re-deferring.

**Alternative considered and rejected: build a placeholder weighting for
Q1 (e.g., a simple win-rate-only score) and revise later.** Rejected for
the same reason ADR-0010 rejected a minimal `PlayerRating` in Social
Play: a "temporary" formula that real matches start accumulating history
against becomes load-bearing the moment a second sprint builds on it, and
this project has direct, recent, receipted evidence (`games.facility_id`)
of what silently-wrong-shape cleanup actually costs.

**Alternative considered and rejected: build gender-mix matching behind a
feature flag, defaulted off.** Rejected — a flag defaulted off still
requires designing the data collection (a `Gender` field on `User`, a
consent/optionality surface) before there is a legal answer on whether to
collect it at all; the flag protects the matching logic, not the
collection decision, which is the part actually in question.
