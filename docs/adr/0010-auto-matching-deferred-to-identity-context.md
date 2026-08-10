# ADR-0010: Auto-matching is built with the Identity/Users context, and not before

## Status
**Accepted (T9 Ceremony 1, 2026-08-05) — a scheduling decision with an
explicit trigger condition, not an open-ended deferral.** Nothing is
implemented by this ADR. `PlayerRating`, `Match`, the self-reported starting
`Level`, and any level-/gender-based auto-matching logic are **not built in
T9**; they are built in the sprint that stands up the **Identity/Users**
bounded context, which `docs/process/t9-sprint-plan.md` §A5 names as T10.

This is a **sequencing** decision. It is **not** a reversal of `CLAUDE.md`'s
locked decision that matchmaking is in v1 scope — see "Not a scope reversal"
below, which is part of this decision, not a footnote to it.

Supersedes nothing. **Superseded by ADR-0012 (T10 Ceremony 1, 2026-08-10)**,
which resolves this ADR's own trigger: Q1 and Q2 below were still
unanswered when T10's Ceremony 1 ran, which this ADR's own trigger text
says does not license a fifth roll-forward — it licenses exit (b),
superseding with a new ADR. **This ADR's sequencing analysis (§(a)–(c) below
— why `Level` is structurally an Identity concept, why parking it in the
wrong context has already cost this project once, why T9 had no call site
to justify building it early) is not overturned and remains the
authoritative record of that reasoning; read ADR-0012 for what changed.**
ADR-0012's summary: Identity/Users and `Match` are built in T10 (the parts
of "auto-matching" that do not require Q1/Q2's answers); `PlayerRating`,
the rating/matching algorithm, and gender-mix matching remain blocked,
named precisely rather than silently dropped, with a trigger tied to the
user answering Q1/Q2 rather than to another sprint boundary.

## Context

T8's sprint plan flagged auto-matching as the one backlog item with **no
scheduled sprint at all** in any prior roadmap, and required T9's Ceremony 1
to "explicitly decide whether Competitions' bracket/seeding logic creates
enough shared need with matchmaking to justify building a minimal
`PlayerRating` this sprint, or whether it stays deferred again with a real
reason recorded (not silently dropped a fourth time)."
`docs/process/t9-sprint-plan.md` §A2 is that decision; this ADR is its
durable record, per `HANDOFF.md`'s per-task Definition of Done ("ADR written
if an architectural decision was made") and `docs/roles/principal-engineer.md`
§5's rule that a debt item recorded without a stated condition for paying it
down is, in effect, a decision never to pay it down.

Three things are now settled that were open the previous three times this
rolled forward.

### (a) The blocker is structural: `Level` belongs to a context that does not exist

`docs/agent-operating-handbook.md` A1 assigns the **self-reported starting
level** to the **Identity/Users** context (row: "`User`, roles (player,
host/organiser, game admin, facility owner, club, platform admin),
self-reported starting level"). A1 assigns `matchmaking`, `Match`, and
`PlayerRating` to **Social Play** — but the glossary (A2) defines
`PlayerRating` as "a player's DUPR-style internal rating, derived from `Match`
history; **seeded by `self-reported starting level` before any history
exists** (cold-start)." The cold-start input to the Social Play concept is
therefore an Identity-owned attribute, and no rating can be seeded honestly
without it.

Identity/Users does not exist in this codebase. Verified by inspection this
ceremony (2026-08-05, at `0b0b40b` on `claude/go-backend-pickleball-7up34j`),
not asserted:

```
$ grep -rnE "^type +(User|Player|PlayerRating|Level|Match|Rating)[A-Za-z]* +" \
      --include='*.go' internal/ --exclude-dir=gen
$ grep -rnE "PlayerRating|SelfReportedLevel|Matchmaking" \
      --include='*.go' internal/ web/ --exclude-dir=gen
$ grep -rnE "PlayerRating|player_rating|Matchmaking|matchmaking" proto/
$ grep -rniE "create table (users|players|player_ratings|matches)" db/migrations/
```

**All four return no matches.** There is no `User` type, no `Player` type, no
`PlayerRating`, no `Match`, no `Level`, no matchmaking identifier of any kind
in the non-generated Go, in any `.proto`, or as a table in any migration. The
only textual hits for these words anywhere in the tree are unrelated English
usage in comments ("object-level", "DB-level", "must match the returned
row"). At that commit `internal/` contains four bounded contexts plus shared
platform code — `booking`, `facilities`, `payments`, `socialplay`, and
`platform` — and none of them owns an identity, profile, or rating concept.
Neither does `competitions`, the fifth context T9.1 adds this sprint: §A4
scopes it to reservation + registration, with no rating, seeding, or player
attribute of any kind.

The schema says the same thing in its own comments, repeatedly and
deliberately: `db/migrations/0005_socialplay.sql:5` ("no Users bounded context
yet ... so these are opaque identifiers"), `0005_payments.sql:12`,
`0007_socialplay_waitlist.sql:10`, and `0010_facilities.sql:8-12` ("`owner_id`
is a uuid column even though there is no Users bounded context ... not a FK to
a users table since one doesn't exist yet"). Every actor identifier in this
system — `host_id`, `player_id`, `owner_id`, `actor_user_id` — is currently an
opaque string with nothing behind it.

Building `Level` in T9 therefore does not mean "adding a field." It means
putting an Identity concept, and the profile it hangs off, inside Social Play
or Competitions because Identity has no home yet.

### (b) This project has already paid the bill for parking a cross-context field in the wrong context — once, at real cost

`games.facility_id text` was a Facilities concept parked in Social Play
because Facilities did not exist when Social Play was built
(`db/migrations/0005_socialplay.sql`). Reconciling it in **T8.3** cost:

- a migration — `db/migrations/0011_socialplay_facility_fk.sql`, which adds a
  *new* nullable `venue_facility_id uuid REFERENCES facilities (id)` column
  rather than changing the old one in place, because "changing an existing
  column's type/semantics out from under any current reader is exactly the
  kind of breaking change the nullable-FK precedent exists to avoid";
- a new port — `port.FacilityLookup`, so the pure domain could stay free of
  the Facilities context (CLAUDE.md rules 2/3);
- a new adapter implementing it;
- and **deprecated artifacts that are still in the tree today**, not cleaned
  up by that ticket and not cleanable until nothing reads them:
  `games.facility_id text` still sits in the schema (0011's own comment:
  "DEPRECATED and slated for removal in a future migration once no reader
  still depends on it"); `Game.FacilityID` still sits in the domain struct
  (`internal/socialplay/domain/game.go`, carrying an explicit
  `DEPRECATED (T8.3)` doc comment) and is still a positional parameter of
  `NewGame`; and `proto/pickleball/socialplay/v1/socialplay.proto` still
  carries deprecated `facility_id` fields on both the `Game` message and
  `CreateGameRequest`, which will sit on the wire indefinitely because
  removing a proto field is a breaking change.

Putting `Level` or `PlayerRating` into Social Play or Competitions now would
be the same move, made with full knowledge of what it costs and a written
receipt for the last one. PE's position, unopposed on the technical merits:
**that is a repeat of a known mistake, not a fresh tradeoff.** A tradeoff
requires an upside on the other side of the ledger; see (c) for why there
isn't one this sprint.

### (c) The call site that would have justified building a minimal rating this sprint does not exist

T8's instruction was specific: check whether Competitions' bracket/seeding
logic creates enough shared need to justify a minimal `PlayerRating` now.
Seeding a bracket by rating is the canonical use, and it *would* justify it —
but **T9 does not build brackets.** `docs/process/t9-sprint-plan.md` §A4 cuts
them explicitly: "**Out, explicitly:** brackets, rounds, seeding, match
scheduling, and results," on PdE's accepted argument that those are the part
of Competitions with no analog anywhere in this codebase to mirror, that they
are "the part that most wants a `PlayerRating` that isn't being built," and
that shipping a 0→1 subsystem inside a 0→1 bounded context in the same sprint
is the "innovation tokens" pattern T8's own re-scope notice exists to stop.
T9 ships the reservation + registration spine, which is a complete user loop
(create → advertise → enter → roster) without them.

So there is no seeding call site to serve. Worse, the causality runs the wrong
way: brackets without ratings are a weaker feature, so building brackets first
and ratings later would ship a bracket that seeds arbitrarily and then
silently change how it seeds. Deferring both together, into the sprint that
owns Identity, keeps them consistent.

## Decision

**Auto-matching — `PlayerRating`, `Match`, the self-reported starting `Level`,
and any level- or gender-based matching logic — is not built in T9. It is
built in the sprint that stands up the Identity/Users bounded context, and not
before.** `docs/process/t9-sprint-plan.md` §A5 names that sprint as T10 and
recommends Identity/Users be sequenced first within it, because that one
context unblocks four separate open items (auto-matching, ADR-0009's deferred
OAuth work, real authentication, and three sprints' worth of `actor_user_id`
"claimed actor, not authenticated identity" caveats).

Concretely, until Identity/Users exists, no PR may:
- add a `PlayerRating`, `Match`, `Level`, or `Gender` field or table to Social
  Play, Competitions, or any other existing context;
- add a matchmaking RPC, request field, or UI control that implies matching
  happens.

What T9 *does* owe this decision, and is ticketed for: T8.8 and T8.9 both
shipped honest inline notes where the matching controls would have been, and
T9's Competitions UI (T9.6/T9.7) must do the same rather than silently
omitting them — a dead control is worse than an absent one, and the product
must tell the same story in both places.

## Not a scope reversal

**This ADR does not reopen, weaken, or cancel `CLAUDE.md`'s locked decision**
that matchmaking is in v1 scope: *"Matchmaking: automated from history, always
manually overridable; new players seeded by a self-reported starting level."*
That decision stands, in full, exactly as written. Every element of it —
automation from `Match` history, mandatory manual override, and cold-start
seeding from a self-reported level — remains a v1 requirement.

The only thing decided here is **when it is built and which context owns the
data it needs**, which is a sequencing question the locked decision does not
answer and was never intended to answer. `docs/agent-operating-handbook.md`
B5 makes the Product Owner adversarial toward "phases that reopen
already-locked decisions instead of working within them"; this ADR is an
instance of working within one. Any future reader who cites ADR-0010 as
authority for *dropping* matchmaking from v1 is misreading it.

## Trigger condition

Per `docs/roles/principal-engineer.md` §5 — an undocumented "we'll fix this
later" is a decision to never fix it, so a debt item without a stated
resolution condition should be pushed back on — this deferral is bounded by an
explicit trigger, not by intent:

> **Ceremony 1 of T10 (backlog refinement, per `docs/process/sprint-process.md`)
> must either (a) ticket and build auto-matching as part of standing up
> Identity/Users, or (b) supersede this ADR with a new, numbered ADR stating
> the new decision and its own trigger. It may not defer it again in prose —
> not in a sprint plan, not in a kickoff note, not in a HANDOFF.md bullet.**

"Build it" and "write ADR-00NN superseding ADR-0010" are the only two exits.
A T10 sprint plan that mentions auto-matching in passing and moves on has not
satisfied this trigger, and any reviewer may block on that basis alone. If the
open questions below are still unanswered when T10's Ceremony 1 runs, that
does not license a fourth roll-forward: it licenses exit (b), which forces the
reasoning and the new trigger to be written down again by whoever chooses it.

## Open questions escalated to the user — not resolved by this ADR

Both of these are product/legal decisions, not engineering ones.
`docs/agent-operating-handbook.md` B5 requires the Product Owner to escalate
decisions "that require a product/legal decision ... explicitly out of scope
for an engineering session" rather than guess, and the Business Analyst
dossier says the same for trade-off forks. They are raised here as questions
with the decision each one blocks, and they are open.

**Q1 — How is the Player Level formula weighted?**
The external design handoff's own Assumptions & Open Questions section states
it: *"Player Level formula (tenure + win rate) needs a product decision on
weighting before auto-matching logic is finalized"*
(`docs/design/handoff-2026-08/README.md:83`; the handoff is reconciled into
this project's design workstream by
`docs/design/v1-external-reference-reconciliation.md`). A related question sits
one level above it, unanswered since round 1 of the v1 design review
(`docs/design/v1-system-design.md` §5, §7 Q6): whether "Level" is a
player-facing tenure+wins score, a restatement of the internal DUPR-style
rating, or **both** (a public skill tier plus a private precise rating, which
is what many real systems do).
*Blocks:* the shape of `PlayerRating` and `Level` themselves — whether they
are one field or two, and what `Match` results have to record to feed them.
Engineering can build the storage without this answer; it cannot build the
formula, and building storage shaped by a guessed formula is how the wrong
shape becomes load-bearing.

**Q2 — Is gender-mix matching in scope at all?**
Requirement #15 (`docs/design/v1-system-design.md:64`) asks for "automatic
matchmaking based on level **and gender**," and Flows 3/4 of the design
handoff show a gender-mix matching control. Mixed doubles is a real, standard
pickleball format, so the feature is domain-accurate — that is not the
question. The question is that implementing it means **collecting a protected
attribute and algorithmically acting on it**, which in most jurisdictions this
platform could launch in is a product-and-legal decision, not a design one.

Round 1 of the v1 design review did answer the *design-shape* question
(`docs/design/v1-review-round-1.md`, Q7: additive design confirmed — a
declared `Gender` attribute plus a per-Game matchmaking-mode flag, with the
attribute self-reported, optional, and carrying a "prefer not to say" path).
That answer is a good design *if the feature ships*. It is an internal review
resolution, not user sign-off on whether to collect the attribute at all, and
this ADR does not treat it as one. The comparison class is deliberate: this is
the same product/legal category as the two items still explicitly awaiting the
user's sign-off in `docs/design/v1-system-design.md`'s top blockquote —
`Registration.GuestCount` being set-once, and the camera-link
consent/signage attestation checkbox — both of which are recorded there as
working assumptions "put to the user directly and not yet confirmed."
*Blocks:* whether Identity/Users carries a `Gender` attribute at all, whether
`Game`/`Competition` gets a matchmaking-mode flag that can require
gender-balanced pairing, and what the signup form asks. An answer of "no"
removes work; an answer of "yes" adds a consent/optionality surface that has
to be designed before it is built, not after.

Neither question blocks *this* decision — auto-matching is deferred to the
Identity sprint regardless of how they are answered. They block the sprint
that builds it, which is why they are being raised now rather than discovered
at T10's Ceremony 1.

## What would change this decision

`docs/roles/principal-engineer.md` §5 requires locked and deferred decisions
to be defended against re-litigation *absent new evidence*, while remaining
genuinely open *given* new evidence — "the T4 phase's concurrency proof is the
kind of evidence that would justify revisiting a locked decision; a preference
is not." Stated concretely, so a future ceremony can check itself rather than
argue:

**Would change it (build auto-matching earlier than the Identity sprint):**
1. **A real, ticketed call site appears sooner.** If bracket seeding, or any
   other feature that genuinely requires a rating to function, gets pulled
   into a sprint before Identity/Users, the (c) argument is gone and the
   tradeoff genuinely changes.
2. **Identity/Users slips indefinitely.** If T10's re-scope (which §A5 already
   predicts, calling T10 a "three-way overloaded slot") pushes Identity past
   T11 while matching becomes the sprint goal, the sequencing premise fails
   and the choice becomes a real one — with a new ADR, not silence.
3. **The user answers Q1 and Q2 and asks for matching sooner.** A direct
   product instruction is new information, not a preference; the escalations
   exist precisely to make that possible.

**Would not change it:**
- A preference that matching "feels overdue," or that it has now rolled
  forward four sprints. Elapsed time is not evidence; it is the reason this
  ADR has a trigger instead of an intention.
- A minimal/temporary implementation offered as low-risk. `games.facility_id`
  was also minimal, and its cleanup is *still* not finished — see the
  deprecated artifacts listed in (b).
- A design document, mockup, or UI that shows matching controls. The design is
  not in dispute; the data owner and the sequencing are.

## Consequences

**Pros.** No Identity concept is parked in Social Play or Competitions, so the
T8.3 reconciliation cost is not paid a second time knowingly. The deferral is
bounded by a trigger with exactly two exits, so it cannot roll forward a fifth
time by default — which is the specific failure this ADR was written to stop.
Two product/legal decisions that were sitting inside an engineering backlog
are now written as questions addressed to the person who can actually answer
them. Ratings and brackets stay consistent with each other by being deferred
together.

**Cons.** Two shipped sprints of UI (T8.8, T8.9) and T9's Competitions
screens carry honest "matching isn't available yet" notes rather than the
feature; that is a real product gap, disclosed rather than disguised.
Competitions ships without seeding, so its first bracket implementation will
have to introduce seeding and rating semantics at the same time. And T10 is
already an overloaded slot (§A5 names five workstreams competing for it), so
this ADR's trigger lands on a sprint that will have to re-scope — which is
exactly why the trigger's second exit is "supersede with a new ADR," not
"skip."

**Alternative considered and rejected: build a minimal `PlayerRating` in
Social Play now, migrate it to Identity later.** Rejected on (b): this is not
a hypothetical migration cost, it is a measured one this project has already
paid once, and the artifacts from the last time are still sitting in the
schema, the domain, and the proto. Rejected also on (c): a minimal rating with
no call site is speculative generality, the first anti-pattern
`docs/roles/principal-engineer.md` §5 lists — an abstraction whose cost is
paid by every future reader while it has zero real callers.

**Alternative considered and rejected: defer again without an ADR, as the
previous three sprints did.** Rejected on `docs/roles/principal-engineer.md`
§5's technical-debt standard and `HANDOFF.md`'s Definition of Done. The
difference between this ADR and the previous three deferrals is not the
outcome — it is that this one names the owning context, names the sprint,
names the trigger, names what would change it, and sends the two questions it
cannot answer to the person who can.
