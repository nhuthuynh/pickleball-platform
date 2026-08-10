# ADR-0012: Identity/Users and Match are built in T10; PlayerRating, the matching algorithm, and gender-mix matching remain blocked on two escalated product/legal decisions

## Status

**Accepted (T10 Ceremony 1, 2026-08-10). Supersedes ADR-0010.** ADR-0010's
sequencing analysis (§(a)–(c): why `Level` is structurally an Identity
concept, why parking it in the wrong context is a bill this project has
already paid once, why T9 had no call site to justify building early) is
**not overturned and remains the authoritative record of that reasoning** —
this ADR does not relitigate it. What changes is the trigger's resolution:
ADR-0010 required T10's Ceremony 1 to either build auto-matching in full or
supersede it with a new ADR stating a new decision and its own trigger.
This is that new ADR.

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

## Open questions escalated to the user — unchanged, restated for continuity

Both remain exactly as ADR-0010 posed them; this ADR resolves neither.

**Q1 — How is the Player Level formula weighted?** Tenure + win rate,
per `docs/design/handoff-2026-08/README.md:83`; also unresolved since
`docs/design/v1-system-design.md` §5/§7 Q6 whether "Level" is a
player-facing tenure+wins score, a restatement of the internal
DUPR-style `PlayerRating`, or both.

**Q2 — Is gender-mix matching in scope at all?** Requirement #15 and Flows
3/4 of the design handoff show it; the question is whether collecting and
algorithmically acting on a protected attribute is something this
platform should do, in the jurisdictions it will launch in — a
product/legal call, not an engineering one.

Neither question blocks what this ADR decides to build (Identity/Users,
`Match`); both block the sprint after the one that receives the answers.

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
