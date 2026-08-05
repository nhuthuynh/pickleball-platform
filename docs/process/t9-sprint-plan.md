# T9 Sprint Plan — Competitions context + growth (shareable links, not OAuth)

Produced by Ceremony 1 (Backlog refinement) + Ceremony 2 (Sprint planning)
per `docs/process/sprint-process.md`. Ceremony 1 played by **Product
Manager** (handbook B2) and **Principal Engineer** (handbook B1 +
`docs/roles/principal-engineer.md`); Ceremony 2 by the full six-role team
(PE, PM, PdE, QA, PO, BA, UX/UI Designer — each loading its handbook brief
and `docs/roles/*.md` dossier).

> **Scope notice, read this first.** T9's scope is inherited, not
> re-derived: `docs/process/t8-sprint-plan.md`'s Part A re-scope notice is
> a **locked decision** and is not re-litigated here. It named four things
> for T9: (c) a new **Competitions** bounded context plus its
> Discover/Advertise UI (Flow 5), (d) social-account-linking + shareable-
> registration-links, (e) a scoped WhatsApp-or-Zalo owned-channel-bot
> spike, and (f) a **real decision** — not a fourth deferral — on
> whether/when auto-matching gets built.
>
> This ceremony executes that scope. Two of the four survive intact
> (Competitions, shareable-registration-links). One is **split on a
> genuine, newly-found blocker** (social-account-linking), and one is
> **decided, with an ADR and a trigger condition** (auto-matching).
> Both are argued in Part A rather than asserted, per this project's
> "don't manufacture consensus" discipline.

---

# Part A — Scope findings and roadmap update

## A0. What the code actually says (verified this ceremony, not assumed)

T8's own re-scope notice exists because the T7 roadmap made assumptions
about the code that turned out to be half true. This ceremony ran the same
check first. Findings, all confirmed by inspection on
`claude/go-backend-pickleball-7up34j` at `cb9cdb9`:

1. **`internal/competitions` does not exist.** Confirmed — `internal/`
   contains `booking`, `facilities`, `gen`, `payments`, `platform`,
   `socialplay`. There is no `proto/pickleball/competitions/v1` either.
   This is a genuine 0→1 bounded context, the fifth in the repo.
2. **The Booking spine is already ready to carry it.**
   `internal/booking/domain/booking.go:13` defines
   `SourceCompetition Source = "competition"`, `ValidSource` accepts it,
   `db/migrations/0001_init.sql:25`'s `CHECK (source IN (...))` already
   lists `'competition'`, and `booking_test.go` already proves the
   no-double-booking invariant holds **across** sources (`game vs
   competition same court overlapping conflicts`, both directions). So
   Competitions does not need any change to Booking's schema, domain, or
   tested invariant — it needs its own `port.CourtReservation` +
   `adapter/booking` pair that passes `SourceCompetition` where Social
   Play's existing one hardcodes `SourceGame`
   (`internal/socialplay/adapter/booking`). This is the single biggest
   de-risking fact in this sprint and it is why Competitions can be a
   5-point domain ticket rather than a T5-sized 27-point build.
3. **There is no Identity/Users context, no `User` entity, no `Level`, no
   `PlayerRating`, and no `Match` anywhere in the repo.** Confirmed by
   grepping `--include=*.go --include=*.proto --include=*.sql
   --include=*.ts --include=*.vue` for
   `PlayerRating|AutoMatch|SocialAccount|OAuth|Competition` — the only
   non-`node_modules`, non-`internal/gen` hits are `SOURCE_COMPETITION` in
   `booking.proto` and T8.8's own comments explaining why it *omitted* the
   matching controls. `agent-operating-handbook.md` A1 assigns `Level` /
   "self-reported starting level" to the **Identity/Users** context, which
   has never been built. This is load-bearing for the auto-matching
   decision (§A2) and was not previously stated this precisely.
4. **`domain.RegistrationSource` already has an unused `social` value.**
   `internal/socialplay/domain/registration.go:13-14` defines
   `RegistrationSourceApp` and `RegistrationSourceSocial`, and its own doc
   comment admits "produces `RegistrationSourceApp` today" — i.e. `social`
   has been modelled since T5.2 with **no producer**. Shareable
   registration links are exactly the missing producer. BA finding, see
   §A3.
5. **T8.10's placeholder fee is live in the shipped UI.**
   `PLACEHOLDER_REGISTRATION_FEE_CENTS` is still the amount every Payments
   screen sends, because `domain.Game` has no price field —
   `HANDOFF.md`'s Cross-cutting section already recommends a real
   per-Game price field as a "T9/T10 backlog" follow-up. Since T9.1 has to
   model a Competition's **entry fee** anyway (Flow 5 names it as a
   create-competition field), the cheapest moment to model money on a
   playable session correctly, once, for both contexts, is this sprint —
   not after Competitions has shipped its own independent shape. See
   T9.2 and §A4.

## A1. The social-account-linking split (genuine blocker, not a scope cut)

T8's roadmap line for (d) reads: "Social-account-linking (an OAuth token
per host per platform) and shareable-registration-links (in-app RSVP —
**still** not reply-scraping, this remains a locked decision)."

The locked half is untouched: **shareable-registration-links ship this
sprint** (T9.5), and nothing in this plan reads, scrapes, or parses a
public reply on any platform. `docs/design/v1-system-design.md` §4 and
`docs/design/v1-external-reference-reconciliation.md`'s "one direct,
load-bearing conflict" section are carried forward as-is, including the
reconciliation note's explicit instruction that a wireframe label reading
"via WhatsApp reply" must be drawn as "via WhatsApp — owned-channel bot"
or similar, never copied verbatim.

The OAuth half is where this ceremony found a real problem, raised by PE
and independently seconded by QA:

> **Storing a third-party OAuth access/refresh token keyed to a claimed,
> unverified `actor_user_id` is a categorically worse failure than the
> object-level-authorization caveat this project already carries.**

`HANDOFF.md`'s Cross-cutting section documents that caveat three times over
(T5.5 Social Play, T6.7/T8.5 Payments, T7.7 Facilities) and each time the
recorded conclusion is the same: "this proves the *object-level* check
given a claimed `actor_user_id`; it does not and cannot prove that identity
itself" until real auth lands. Every prior instance of that caveat bounds
the blast radius to *this platform's own data* — a bad actor claiming
someone else's ID could cancel a registration or add a court. An OAuth
token store changes the blast radius in kind, not degree: the same
unverified claim would hand the caller a live credential that posts to a
real person's WhatsApp Business number, Facebook Group, X account, or
Instagram — on a system with no authentication, no encryption-at-rest
story, and no token-revocation path. That is not "the same caveat again";
it is the first time the caveat would guard someone else's property
outside this system.

PE classifies this as a **one-way door** in the Bezos sense (PE dossier §3
heuristic 1): once real hosts have connected real accounts, a token store
built without auth cannot be un-shipped, and a leak cannot be un-leaked.
The correct response to a one-way door with an unresolved prerequisite is
to not walk through it yet.

**What ships instead, and why it is not a consolation prize.** Flow 5's
user-visible value is: *a host publishes an auto-formatted promo with a
registration link, and the replies become registrations*. Decomposed:

| Flow 5 capability | Needs OAuth? | T9 status |
|---|---|---|
| Auto-formatted promo copy (name, format, dates, venue, entry fee, link) | No | **Ships** (T9.6, composed client-side from data the client already has) |
| A shareable registration link that lands in-app | No | **Ships** (T9.5) |
| Tapping the link registers you, with guest count, attributed to its channel | No | **Ships** (T9.5 + T9.7) |
| Roster showing name, guest count, and source channel | No | **Ships** (T9.6) |
| The platform *posting on the host's behalf* without the host pasting | **Yes** | **Deferred** (ADR-0009, T9.8) |

Only the last row needs a credential. Everything else is a share payload
plus a token-addressed read — which is what the locked "in-app RSVP via a
shareable link" decision has always described. PE's framing: this is the
**boring** implementation of Flow 5, and it happens to be the one with zero
credential custody.

**The UX consequence is recorded, not smoothed over.** The design
attachment's wireframe (`docs/design/handoff-2026-08/README.md` screen 5)
draws a connected-accounts list with per-platform "Connected / Connect"
buttons. T9 must **not** draw that, because a "Connect" button that opens
no OAuth flow is a deceptive pattern in the precise, catalogued sense
(UX dossier §6, Brignull) and fails NN/g heuristic #1 (visibility of system
status). The Designer's ruling, agreed by PM: the Advertise step is
labelled for what it does — "Copy your post" / "Share" (Web Share API where
available) — with an honest one-line note that automatic posting to linked
accounts isn't available yet, mirroring exactly how T8.8 handled the
omitted matching step rather than shipping a dead control.

## A2. Auto-matching: the decision (not a fourth deferral)

T8's plan flagged this as the one item that "has **no scheduled sprint at
all** in any prior roadmap," and required T9's Ceremony 1 to "explicitly
decide whether Competitions' bracket/seeding logic creates enough shared
need with matchmaking to justify building a minimal `PlayerRating` this
sprint, or whether it stays deferred again with a real reason recorded (not
silently dropped a fourth time)."

**Decision: auto-matching is NOT built in T9. It is scheduled for T10 as
part of standing up the Identity/Users context, and the decision is
recorded as ADR-0010 with an explicit trigger condition (T9.9) so it cannot
roll forward a fifth time by default.**

This is a decision, not a deferral, because three things are now settled
that were previously open:

1. **The blocker is named and it is structural, not "no screen yet."**
   `Level` / "self-reported starting level" belongs to **Identity/Users**
   per `agent-operating-handbook.md` A1. That context does not exist —
   there is no `User`, no profile, no place a self-reported level can
   live. Building `Level` in T9 means putting an Identity concept inside
   Social Play or Competitions. This project has already paid that exact
   bill once: `games.facility_id text` was a Facilities concept parked in
   Social Play, and reconciling it cost T8.3 a migration, a new
   `port.FacilityLookup`, a new adapter, a deprecated column that still
   sits in the schema today, and a deprecated proto field that will sit on
   the wire indefinitely. PE's position, unopposed on the technical
   merits: doing it a second time knowingly is not a tradeoff, it is a
   repeat.
2. **A product decision is genuinely outstanding, and it is not
   engineering's to make.** The design attachment's own Open Questions
   say: "Player Level formula (tenure + win rate) needs a product decision
   on weighting before auto-matching logic is finalized"
   (`docs/design/handoff-2026-08/README.md`). Separately, Flow 3/Flow 4
   name a **gender-mix** matching control; gender is a protected attribute
   in most jurisdictions this platform could launch in, and collecting and
   algorithmically acting on it is a product-and-legal decision of the
   same class as the two items already awaiting user sign-off in
   `docs/design/v1-system-design.md`'s top blockquote. PO's brief (B5)
   says escalate product/legal decisions rather than guess; BA's dossier
   §6 says the same for trade-off forks. Both are escalated here, in
   writing, rather than resolved by an engineering ceremony.
3. **Competitions does not create the shared need T8 asked us to test
   for.** T8's instruction was specifically to check whether bracket/
   seeding logic justifies a minimal `PlayerRating`. It would — seeding a
   bracket by rating is the canonical use — but **T9 does not build
   brackets** (§A4), so there is no seeding call site. And the causality
   runs the other way round from what would justify building it: brackets
   without ratings are a weaker feature, so building brackets first and
   ratings later would ship a bracket that seeds arbitrarily and then
   silently change how it seeds. Deferring both together, in the same
   sprint that owns Identity, keeps them consistent.

**Trigger condition, recorded in ADR-0010 so the debt is not open-ended**
(PE dossier §5, "silent technical-debt accrual with no trigger to pay it
down" — an undocumented "later" is a decision to never): auto-matching is
built **in the sprint that stands up the Identity/Users context, and not
before**; that sprint is now named as T10 in §A5. Ceremony 1 of T10 must
either build it or supersede ADR-0010 with a new ADR — it may not defer it
in prose.

**What T9 still owes the deferral:** T8.8 and T8.9 both shipped honest
inline notes where the matching controls would have been. T9's
Competitions UI must do the same rather than silently omitting them
(T9.6/T9.7 Instructions), so the product tells the same story in both
places.

## A3. BA finding: give `RegistrationSourceSocial` a definition and a producer

`internal/socialplay/domain/registration.go` has carried
`RegistrationSourceSocial` with **no producer** since T5.2, and the
glossary defines Registration's source only as "(`app | social`)" with no
statement of what `social` means. Shareable registration links are the
producer it was modelled for.

BA's ruling, agreed by PE: **Competitions must use the same two values,
`app | social`, not a new `shared_link` vocabulary** — CLAUDE.md rule 7
(one ubiquitous language) and BA dossier §5's contradiction check both
point the same way, and inventing a third word for the same fact is
precisely the "two different terms for what should be the same concept"
failure that checklist exists to catch. T9.5's PR must add a glossary line
to `agent-operating-handbook.md` A2 defining `social` precisely — *"the
registrant arrived via a shareable registration link the Host published to
a channel outside the app"* — which finally makes the value mean something
verifiable rather than gesturing at social media generally.

**Deliberately not done in T9, flagged so it isn't mistaken for an
oversight:** Games do not get shareable links this sprint (only
Competitions do), so `RegistrationSourceSocial` still has no producer on
the Social Play side after T9. Extending share links to Games is a small
follow-up ticket once the Competitions shape is proven — raise at T10
refinement.

## A4. What Competitions is, and is not, in T9

PE + PdE scoped the aggregate deliberately, because "a new bounded context"
is not self-limiting.

**In:** the court-reservation and registration spine. A `Competition` has a
Host, a real venue Facility (via `port.FacilityLookup`, T8.3's pattern),
one or more **sessions** (a competition runs across dates — Flow 5's own
words are "reserves courts across dates"), a capacity, a guest allowance, a
payment method, an entry fee, a status, and a share token. Entries mirror
Registrations. Courts are reserved as `competition`-source Bookings through
Competitions' own `port.CourtReservation`, inheriting the no-double-booking
invariant with **zero change to Booking** (§A0 finding 2).

**Out, explicitly:** brackets, rounds, seeding, match scheduling, and
results. PdE's argument, accepted: those are the part of Competitions with
no analog anywhere in this codebase to mirror, they are the part that most
wants a `PlayerRating` that isn't being built (§A2), and shipping the
reservation+registration spine without them still produces a complete user
loop (create → advertise → enter → roster). Shipping brackets *with* them
would be a second 0→1 subsystem in the same sprint as a 0→1 context —
the "innovation tokens" pattern T8's own re-scope notice was written to
stop. Deferred to T10+, named in §A5, not dropped.

**A consistency question BA raised and the team resolved rather than let
slide:** T8's kickoff note dropped `CancellationCutoff` from Social Play on
the rule *"shipping a cosmetic text field that enforces nothing would be
actively misleading."* Does `Competition.Format` (singles/doubles) violate
that same rule, given it enforces nothing? **No, and the distinction is
worth stating so it doesn't read as an inconsistency:** a cancellation
cutoff is a *rule the Host is asserting over other people's behavior* — a
Host who types one reasonably expects late cancellations to be blocked, and
a field that silently doesn't is a broken promise. A format label is
*descriptive information for players* ("this is a doubles competition"),
which is complete and honest the moment it is displayed. The test is
whether the field implies enforcement, not whether it has code behind it.
T9.1's Instructions require `Format`'s doc comment to say this explicitly.

**Money, modelled once for both contexts.** T9.1 gives `Competition` an
`EntryFee`, and T9.2 gives `Game` one, both as a context-local `Money`
value type mirroring `internal/payments/domain.Money` (ADR-0005: amount and
currency are coupled, never a bare cents value). PE's note for reviewers,
so nobody "fixes" it: each context defining its own small `Money` value
type is correct DDD, not a DRY violation — `proto/pickleball/payments/v1/
payments.proto:36-38` already documents this precedent for the proto side
("Money mirrors internal/payments/domain.Money"), and a shared cross-context
money type would be exactly the shared-kernel coupling
`agent-operating-handbook.md` A1 forbids.

## A5. Roadmap update (supersedes T8 plan's T9/T10 lines)

**T9 (this document, fully ticketed in Part B) — Competitions context +
growth via shareable links.** As scoped above.

**T10 — Identity/Users + auto-matching + CI.** T8's plan named T10 as
"Pricing/discount UI + Club rentals + hardening." That is now a **three-way
overloaded slot**, and saying so here is the point: T10 as currently
accumulated would carry (a) the Identity/Users context and auto-matching
(ADR-0010's trigger, §A2), (b) Jenkins/CI wiring (§A6), (c) the
`discount_rules` backend + pricing UI (Flow 2), (d) the Club account type +
recurring-rental flow (Flow 7), and (e) the cross-sprint WCAG 2.2 AA audit.
That is more than one sprint, and T10's Ceremony 1 will have to re-scope,
the same way T8's did. **This ceremony's recommendation on ordering, for
T10's PM+PE to start from rather than rediscover:** Identity/Users first
(it unblocks auto-matching *and* is the prerequisite for the real
authentication that ADR-0009's deferred OAuth work and three sprints of
`actor_user_id` caveats are all waiting on — one context unblocks four
open items), CI second (§A6), pricing/Club/WCAG after. Recorded as a
recommendation, not a decision — T10's own ceremony owns it.

**T11+ — Competition brackets/rounds/results, share links for Games,
automated social posting (post-auth), pricing/discount UI, Club rentals,
WCAG audit, native Swift/Kotlin clients.** Nothing here is new; it is the
overflow from T10's slot plus this sprint's own explicit deferrals, listed
so none of it lives only in prose.

## A6. The two three-sprint-old chores T8 pre-committed to re-raising

T8's plan closed with an instruction to this ceremony: the `npm install
--legacy-peer-deps` friction and Jenkins CI wiring are "still open — flag
again at T9 refinement if a third sprint passes without them, per the same
T5-retro-finding-6 discipline this sprint applied to T6.7." Both have now
passed a third sprint. Ignoring that instruction would be the exact
"flagged in prose, not tracked as a real ticket" failure T5's retro
finding 6 named, which is the reason the instruction exists.

- **`npm install --legacy-peer-deps` (T9.10, 2 points): taken this
  sprint.** It is small, well-understood (16 tickets' worth of builds have
  hit it), and touches nothing else in T9's dependency graph.
- **CI: named for T10, not taken here.** See the recorded disagreement in
  §B4 — this was not a unanimous call.

---

# Part B — T9 Sprint Plan

## Sprint goal

> A Host can create a Competition at a real Facility that reserves its
> courts as `competition`-source Bookings under the same no-double-booking
> invariant a Game does, publish a shareable registration link for it, and
> a Player can discover that Competition, follow the link, and enter with
> guests at a real entry fee — all through the real Vue app against real
> gRPC-gateway REST APIs, with no OAuth credential custody, no
> reply-scraping, and no fabricated or dead-end fields anywhere in the
> flow.

## Kickoff note

### Scope decisions against the open UX questions (carried from T7's and T8's kickoff notes)

1. **Role-switching UX.** Decided in T7 (contextual indicator, UI-state
   only), extended in T8 to recognize "Host" after a Game is created. T9
   extends the same `RoleIndicator` mechanism to recognize Host after a
   **Competition** is created (T9.6) — no new mechanism. Same caveat,
   unchanged and not re-litigated: client-side session state, not a
   server-verified permission, until the JWT/Auth0 item lands.
2. **Auto-matching transparency.** **Decided this sprint, not deferred** —
   see §A2 and T9.9 (ADR-0010). The transparency *UI* question stays moot
   until the feature exists; what changed is that the feature now has a
   named sprint, a named blocker, a named escalation, and an ADR.
3. **Cash-vs-online payment surfacing prominence.** Decided and
   implemented in T8 (Host dashboard badge + list, no notifications).
   T9's Competitions entries carry `PaymentMethod` and `EntryFee` in the
   same shape, so the same surfacing applies — T9.6's roster shows a
   Competition's unpaid-cash entries using the pattern
   `HostPayments.vue` already established. No new decision needed.
4. **Pricing-conflict UI.** Unchanged — still T10+ with T7's recorded
   direction (validation error at entry time, not a runtime resolution
   surface). Not touched in T9.
5. **Club as a fourth account type.** Unchanged — adopted as domain
   direction, not built. Not touched in T9.

### Process decision: authorization is baked in, not a follow-up ticket

Four sprints running (T5.5, T6.7→T8.5, T7.7, and T8.5 again) have shipped
object-level authorization as its **own ticket after the write path
existed** — and T7.7 found that in one case (`AddCourt`/`AddCameraLink`)
the check did not exist at all, meaning a shipped write path was
unprotected until a later sprint noticed. T5's retro finding 1 already
states the rule: bake an invariant's guard into the ticket that introduces
it, not a follow-up.

**T9 applies it.** There is deliberately **no separate authz ticket this
sprint.** `Competition.EnsureHost(actorUserID)` is part of T9.3 (the ticket
that introduces the write path), and the handler-level regression proof
plus the `toStatus` → `PermissionDenied` mapping are acceptance criteria of
T9.4 (the ticket that introduces the handler). QA signed off on this only
on the condition that T9.4's Instructions name the mutation-test
verification explicitly (disable the check, watch the test fail, restore
it — QA dossier §5 item 2, CLAUDE.md rule 10) rather than leaving "add a
regression test" as an unfalsifiable line item. That condition is in T9.4.

Same caveat as every prior instance, stated once and not re-litigated per
`HANDOFF.md`'s own instruction: this proves the *object-level* check given
a claimed `actor_user_id`, not authentication.

### Scope decisions against the design handoff's Flow 5

- The connected-accounts list with "Connected / Connect" buttons is **not
  built** — see §A1. The Advertise step is a share-payload composer with
  copy / Web Share, honestly labelled.
- The registrations list showing "name, guest count, and source channel"
  **is** built (T9.6), with source rendered per the reconciliation note's
  instruction — never the wireframe's verbatim "via WhatsApp reply".
- "Manage roster (host confirms, schedules, brackets)" ships as **roster
  visibility + entry management**; scheduling and brackets are out (§A4).

### Toolchain / environment note (same gotcha class as T4's, T7's, and T8's)

This authoring environment was not verified for `node`/`npm`/Docker/buf/
sqlc availability as part of this ceremony (planning-only session, no
`make test` run). Whoever implements T9's tickets should run
`HANDOFF.md`'s "First actions on resume" toolchain check before assuming
availability, rather than trusting this note either way. Note specifically
that `HANDOFF.md` records **no Docker daemon** in several prior sandboxes,
which is what forced T4/T5.5/T6.4/T6.5/T6.6/T8.3's manual-Postgres
verification fallback — T9.4's concurrency requirement should plan for that
fallback rather than discover it.

---

## Tickets

### T9.1 — Add the Competition and CompetitionEntry domain model (pure domain, TDD)

**Story:** As a Host, I want to define a competition — its venue, its
sessions across one or more dates, its capacity, guest allowance, payment
method, and entry fee — so that a competition is a real modelled thing this
platform can reserve courts for, rather than a Game with a different label.

**Description:** The 0→1 domain ticket for this repo's fifth bounded
context, `internal/competitions/domain`. Mirrors Social Play's shape
closely and deliberately (CLAUDE.md: "Add a new bounded context... mirror
the `booking` context exactly"; Social Play is the closest sibling since a
Competition reserves courts the same way a Game does). Pure domain,
framework-free, TDD-first — no proto, no adapters, no app service in this
ticket (T9.3/T9.4 wire it). This ticket must land T8.6's hard-won capacity
lesson **from the start**, not rediscover it.

**Instructions:**
1. Functional requirements:
   - Write failing table-driven tests first (CLAUDE.md rule 1).
   - `internal/competitions/domain` with: `Competition`, `Session`,
     `CompetitionEntry`, `Status`, `Format`, `PaymentMethod`, `Money`,
     `EntrySource`, `TimeRange`, and a sentinel-error file — mirroring
     `internal/socialplay/domain`'s file layout so a reader who knows one
     context can navigate the other.
   - `Competition` fields: `ID`, `HostID`, `Name`, `VenueFacilityID`
     (optional, same semantics and doc-comment reasoning as
     `domain.Game.VenueFacilityID` — stored as-is here, existence checked
     by the app layer via a port, since this package may never import
     Facilities), `Sessions []Session`, `Capacity int`, `GuestAllowance
     int`, `PaymentMethod`, `EntryFee Money`, `Format`, `Status`,
     `ShareToken string`.
   - `Session` fields: `Range TimeRange`, `CourtIDs []string`. A
     Competition has **one or more** sessions (Flow 5: "reserves courts
     across dates"); `NewCompetition` rejects zero sessions
     (`ErrEmptySessions`), a session with zero courts
     (`ErrEmptyCourtIDs`), and an invalid/inverted range
     (`ErrInvalidTimeRange`) — re-validating the range itself rather than
     trusting the caller went through `NewTimeRange`, same reasoning
     `domain.NewGame` documents.
   - **Overlapping sessions within one Competition on the same court are
     rejected at construction** (`ErrOverlappingSessions`). Given/When/
     Then: *Given* a Competition whose session A reserves court C for
     10:00–12:00, *When* session B also reserves court C for 11:00–13:00,
     *Then* `NewCompetition` returns `ErrOverlappingSessions` — because
     the Booking-level invariant would reject the second reservation
     mid-way through T9.3's loop and force a rollback for a mistake that
     was detectable up front. Back-to-back sessions on the same court
     (one ends exactly when the next starts) are **not** an overlap —
     ranges are half-open `[start, end)`, same rule as Booking.
   - `CompetitionEntry` fields: `ID`, `CompetitionID`, `PlayerID`,
     `GuestCount int`, `Source EntrySource`, `PaymentStatus`, `Status`.
   - `EntrySource` is `app | social` — **the same two values Social Play's
     `RegistrationSource` already uses, not a new vocabulary** (CLAUDE.md
     rule 7; see §A3). `social` means the entrant arrived via a shareable
     registration link.
   - **The capacity invariant is weighted from day one.** `Enter(...)`'s
     rejection condition is `sum(1 + e.GuestCount for e in active
     entries) + (1 + guestCount) > Capacity` — never a plain entry
     headcount. Required test, phrased so it proves the weighting rather
     than the arithmetic: a `Capacity: 3` Competition with one existing
     entry at `GuestCount: 2` (weight 3, already full) rejects a second
     entry at `GuestCount: 0` with `ErrCompetitionFull`. This is the test
     that would *pass* under a count-based check and must *fail* it. (T8.6
     had to retrofit this into Social Play; T5's retro finding 1 says bake
     it into the ticket that introduces the invariant.)
   - `GuestCount` is validated `0 <= GuestCount <= GuestAllowance`
     (`ErrGuestAllowanceExceeded`); `GuestAllowance >= 0` with 0 meaning
     no guests permitted.
   - `Money` is a context-local value type `{AmountCents int64;
     CurrencyCode string}` mirroring `internal/payments/domain.Money` per
     ADR-0005 (amount and currency coupled, never a bare cents value).
     `EntryFee` of zero amount is legal and means a free competition —
     it is a real value, not a placeholder. Reject a non-empty amount with
     an empty currency code (`ErrInvalidMoney`). Add a doc comment
     explaining that the duplication across contexts is deliberate (§A4)
     so a future reviewer doesn't extract a shared money package.
   - `Format` is `singles | doubles`. Its doc comment must state
     explicitly that it is **descriptive, not enforcing** — nothing in the
     domain validates entry counts against it — and why that is
     acceptable where T8's dropped `CancellationCutoff` was not (§A4: a
     format label informs, a cutoff implies enforcement).
   - `Competition.EnsureHost(actorUserID string) error` returning
     `ErrNotCompetitionHost` on mismatch — mirroring
     `facilities.Facility.EnsureOwner` /
     `socialplay.ErrNotRegistrationOwner` / `payments.ErrNotPaymentRecorder`.
     Introduced here, in the same sprint as the write path, per the
     kickoff note's process decision.
   - `Cancel()` transitions `scheduled → cancelled`; cancelling twice is
     rejected, not silently idempotent. Same known gap as Social Play's
     (a cancelled Competition does not yet cascade to its Bookings or
     entries) — state it in the doc comment and in the PR, don't let it
     look accidental.
2. Non-functional requirements:
   - **Zero non-stdlib imports** in `internal/competitions/domain` —
     verify by inspection and state the result in the PR (CLAUDE.md
     rule 2).
   - Table-driven boundary tests, at minimum: capacity exactly reached
     vs. exceeded (both directions — QA dossier §5 item 4: the shadow rule
     "exactly at capacity still succeeds" must also be tested);
     `GuestAllowance == 0` with `GuestCount > 0` rejected; `GuestCount`
     exactly at the allowance boundary allowed, one over rejected;
     back-to-back sessions on one court allowed; overlapping sessions
     rejected; invalid `Format`/`PaymentMethod`/`Status` values; zero-fee
     and non-empty-amount-with-empty-currency `Money` cases.
   - Add `Competition`, `Session`, `Competition Entry`, `Entry Source`,
     and `Format` to `agent-operating-handbook.md` A2's glossary **in this
     ticket's PR** (CLAUDE.md rule 7), explicit about how a Competition
     differs from a Game (multi-session; brackets deliberately absent in
     T9 — §A4).

**Story points:** 5

**Labels:** `sprint:t9`, `role:principal-engineer`, `type:story`, `points:5`

---

### T9.2 — Give a Game a real `EntryFee`, retiring T8.10's disclosed placeholder

**Story:** As a Player, I want to see and pay the actual price a Host set
for their Game, so that the amount I am charged is a real number the Host
chose rather than a placeholder the UI labels as fake.

**Description:** `HANDOFF.md`'s Cross-cutting section records that T8.10
found `domain.Game`/`domain.Registration` have no price field at all and
shipped `PLACEHOLDER_REGISTRATION_FEE_CENTS` ($10.00), visibly labelled
"placeholder" everywhere — reviewed as an acceptable *disclosed* workaround
with the explicit recommendation of "a follow-up ticket (T9/T10 backlog) to
add a real per-Game price field, the same way T8.6/T8.7 did for
`PaymentMethod`." This is that ticket, taken in T9 rather than T10 for a
specific reason (§A0 finding 5): T9.1 models a Competition's entry fee this
sprint regardless, and modelling money on a playable session once, for both
contexts, in the same sprint is strictly cheaper and more consistent than
modelling it twice a sprint apart. Independent of every other T9 ticket —
can run in parallel with T9.1.

**Instructions:**
1. Functional requirements:
   - Failing tests first. `domain.Money` in `internal/socialplay/domain`
     (context-local, mirroring `internal/payments/domain.Money` per
     ADR-0005 — same reasoning and same doc comment as T9.1's; coordinate
     wording with T9.1 if both are in flight).
   - `domain.Game` gains `EntryFee Money`; `NewGame(...)` gains it as a
     parameter and validates it (`ErrInvalidMoney` for a non-empty amount
     with an empty currency code; a negative amount is rejected). **A zero
     amount is legal and means a free Game** — a real value, not a
     sentinel.
   - `db/migrations/0013_socialplay_entry_fee.sql`: `ALTER TABLE games ADD
     COLUMN entry_fee_cents bigint NOT NULL DEFAULT 0`, `ADD COLUMN
     entry_fee_currency text NOT NULL DEFAULT 'USD'`, with a `CHECK
     (entry_fee_cents >= 0)`. Document the defaults in the migration file:
     0 = free, which is the only safe default for pre-existing rows —
     backfilling any non-zero figure would invent a charge no Host ever
     set. (`'USD'` mirrors ADR-0005's existing currency-column precedent;
     check that ADR before picking a different default and say in the PR
     which one you followed.)
   - `CreateGameRequest`/`Game` proto messages gain an `entry_fee` field
     using a context-local `Money` message mirroring
     `proto/pickleball/payments/v1/payments.proto`'s (do **not** import
     the payments proto package — §A4). Run `make generate`; never
     hand-edit `internal/gen/**` (CLAUDE.md rule 6).
   - sqlc queries + `internal/socialplay/adapter/postgres` updated for the
     new columns, following the existing per-query `fromFields` pattern
     (see `HANDOFF.md`'s gotcha on distinct `...Row` structs per query).
   - **Delete `PLACEHOLDER_REGISTRATION_FEE_CENTS` and every "placeholder"
     label it drove**, in `GameCreation.vue` (add a fee input to the
     existing form), `GameJoinPanel.vue`/`GameDetailPanel.vue` (show the
     real fee, or "Free" at zero), and `GameCheckout.vue`/
     `HostPayments.vue` (send the real amount). Grep for the constant to
     be sure every call site is caught, and state in the PR that the
     constant no longer exists anywhere.
2. Non-functional requirements:
   - **PCI guardrail (CLAUDE.md rule 11):** this ticket changes a proto
     message on a payment path, so review it against
     `docs/checklists/proto-review.md` before merging and say so in the
     PR. An entry fee is an amount, never card data — no PAN/CVV/track-data
     field may appear on any request DTO added here.
   - WCAG 1.4.1 (Use of Color) and 3.3.1 (Error Identification) on the new
     fee input: a validation error ("Entry fee can't be negative") renders
     as text next to the field, same rule T7.4/T8.8 already applied.
   - "Free" must render as the word, not an empty space or "$0.00 " with a
     hidden meaning — NN/g heuristic #2 (match between system and the real
     world); a zero-price game is a real product state, not a missing
     value.
   - `make test` green, including this context. Existing T8.6/T8.7 tests
     that construct a `Game` must be updated, not deleted, and their
     substance must not change (this ticket adds a field; it must not
     silently alter capacity or payment-method behavior).

**Story points:** 3

**Labels:** `sprint:t9`, `role:principal-engineer`, `type:story`, `points:3`

---

### T9.3 — Competitions application service: reserve courts, validate the venue, enforce Host ownership

**Story:** As a Host, I want scheduling a competition to actually reserve
its courts, so that nobody else can book those courts for those times — and
as the platform, I want a competition at a venue that doesn't exist to be
rejected before any court is held.

**Description:** Depends on T9.1. Mirrors T5.3's shape (app service +
`port.CourtReservation` + all-or-nothing rollback) and T8.3's
(`port.FacilityLookup` validation before reserving), one level deeper
because a Competition reserves across `sessions × courts` rather than a
single range. The `competition` Booking source already exists and is
already covered by Booking's cross-source conflict tests (§A0 finding 2),
so **this ticket must not change anything under `internal/booking/`** —
verify and state that in the PR.

**Instructions:**
1. Functional requirements:
   - Failing tests first, against in-memory fakes for every port.
   - `internal/competitions/port`: `Repository`, `IDGenerator`,
     `CourtReservation`, `FacilityLookup`, and `ShareTokenGenerator`
     (T9.5 consumes the last one; introduce it here so `ScheduleCompetition`
     can populate `ShareToken` at construction rather than needing a second
     write path later).
   - `port.CourtReservation` is expressed **entirely in primitives** (court
     ID, start/end `time.Time`, reference ID) — copy the interface-shape
     reasoning from `internal/socialplay/port/court_reservation.go`'s doc
     comment, including that every reservation through this port is
     implicitly `competition`-source and the source is therefore not a
     caller-chosen parameter.
   - `internal/competitions/adapter/booking` implements it against the real
     `bookingapp.Service.CreateBooking` with
     `bookingdomain.SourceCompetition`, translating
     `bookingdomain.ErrCourtDoubleBooked` into a context-local
     `domain.ErrCourtUnavailable` (CLAUDE.md rule 5). This adapter package
     is the **only** place `internal/competitions` may import
     `internal/booking/*`.
   - `internal/competitions/adapter/facilities` implements `FacilityLookup`
     against the real `facilitiesapp.Service.GetFacility`, mirroring
     `internal/socialplay/adapter/facilities` (T8.3).
   - `app.Service.ScheduleCompetition`:
     1. validates `VenueFacilityID` (when non-empty) via `FacilityLookup`,
        returning `domain.ErrFacilityNotFound` → 404-shaped status,
        **before any court is reserved**;
     2. constructs the `Competition` via `domain.NewCompetition`;
     3. reserves every `(session, court)` pair through `CourtReservation`;
     4. on any failure, releases every reservation already made, in
        reverse order, best-effort, and surfaces the **original**
        reservation failure — not the rollback's — as the actionable
        error (same reasoning `port.CourtReservation.ReleaseCourt`'s
        existing doc comment gives).
   - `app.Service.EnterCompetition` applies the weighted capacity check via
     the domain and persists the entry.
   - `app.Service.CancelCompetition` calls `Competition.EnsureHost` before
     `Cancel()`. `EnterCompetition` needs no ownership check (any Player
     may enter a published Competition — same reasoning as
     `RegisterForGame`); say so explicitly in the PR rather than leaving
     the asymmetry unexplained.
   - Constructor takes an **options struct**, not positional args.
     `HANDOFF.md`'s Cross-cutting section already records that
     `socialplayapp.NewService` reached 4 positional args in T6.6 and that
     "the next dependency added... should switch it to an options struct
     instead of growing further." A brand-new service starting at five
     dependencies must not repeat that; follow
     `paymentsapp.ServiceOptions`'s existing shape.
2. Non-functional requirements:
   - `internal/competitions/domain` and `internal/competitions/app` must
     **not** import `internal/booking/*` or `internal/facilities/*`.
     Verify by import inspection and state the result in the PR
     (CLAUDE.md rule 3; PE dossier §3 heuristic 7 — the first PR that
     quietly violates this is the one to catch).
   - Required test — **partial-state rollback**: a Competition with two
     sessions where the *second* session's court is already booked leaves
     **zero** Bookings behind and **no** persisted Competition. Assert on
     the observable state (no bookings, no competition), not on the fake's
     call sequence — QA dossier §6 warns against change-detector tests
     that assert a mock's call log instead of the outcome.
   - Required test — venue validation ordering: an unknown
     `VenueFacilityID` reserves **nothing** (mirrors T8.3's
     `TestScheduleGame_UnknownVenueFacilityRejectedBeforeReservingCourts`).
   - Required test — `EnsureHost`: a non-Host actor cannot cancel a
     Competition, and the error is `ErrNotCompetitionHost` specifically,
     not merely non-nil (QA dossier §5 item 3).
   - `make test-domain` green.

**Story points:** 5

**Labels:** `sprint:t9`, `role:principal-engineer`, `type:story`, `points:5`

---

### T9.4 — Wire Competitions to Postgres + proto + gRPC/REST, with a weighted DB capacity guard and the authz regression proof

**Story:** As a client (Vue this sprint), I want to create, read, list, and
enter Competitions over the real REST API, so that T9.1/T9.3's rules are
reachable outside Go tests — and as the platform, I want the database to
enforce the same weighted capacity rule the domain does, so two
simultaneous entries can't overfill a Competition.

**Description:** Depends on T9.3. Mirrors T5.4 + T8.7 combined: schema,
sqlc, proto, gRPC/REST, plus a DB-level capacity guard that must be
weighted from the start (T8.7 had to *rewrite* Social Play's count-based
trigger; `db/migrations/0012_socialplay_guest_capacity.sql` is the worked
example to copy). This is the ticket where the invariant either becomes
real or quietly doesn't.

**Instructions:**
1. Functional requirements:
   - `db/migrations/0014_competitions.sql`: `competitions`,
     `competition_sessions`, and `competition_entries` tables.
     `competitions.venue_facility_id uuid REFERENCES facilities (id)`
     nullable (T8.3's precedent);
     `competitions.share_token text NOT NULL UNIQUE`;
     `entry_fee_cents bigint NOT NULL CHECK (entry_fee_cents >= 0)` +
     `entry_fee_currency text NOT NULL`;
     `competition_entries.guest_count integer NOT NULL DEFAULT 0 CHECK
     (guest_count >= 0)`;
     `status`/`format`/`payment_method`/`source` as `text` with `CHECK
     (... IN (...))` constraints listing the domain's closed value sets
     (the `bookings.source` precedent in `0001_init.sql:25`).
     Pick the next free migration number at implementation time — T9.2 also
     adds one; whichever lands second renumbers rather than colliding.
     (`HANDOFF.md` records the existing duplicate-`0005` collision as a
     known, tolerated wart — do not add a second one.)
   - **DB-level weighted capacity guard**, modelled on
     `db/migrations/0012_socialplay_guest_capacity.sql`'s trigger and
     `promote_next_waiting`'s `FOR UPDATE` locking pattern: lock the owning
     `competitions` row, sum `1 + guest_count` across non-cancelled
     entries, compare against `capacity`, reject on overflow. The adapter
     translates that rejection into `domain.ErrCompetitionFull` (CLAUDE.md
     rule 5) — no new error type reaches the wire.
   - **Proof standard for the guard, non-negotiable** (CLAUDE.md rule 10;
     QA dossier §5 item 7): N simultaneous `EnterCompetition` calls with
     varying `guest_count` against a Competition sized so exactly one
     combination fits; assert the DB-level guard admits exactly the set the
     domain check would. Run it **more than once, including a cold start**.
     A single green run is not evidence. Commit it as a
     `-tags=integration` testcontainers test alongside the existing
     `concurrency_integration_test.go` files. If this sandbox has no Docker
     daemon (`HANDOFF.md` records that gap repeatedly), fall back to the
     documented manual-Postgres methodology T6.4/T6.5/T8.3 used, **still
     commit the integration test**, and state plainly in the PR which of
     the two produced the evidence — do not describe a manual run as if the
     committed test executed. (T6.4's own review finding: an uncommitted
     throwaway proof leaves nothing guarding the invariant against
     regression.)
   - New sqlc entry in `sqlc.yaml` → package `competitionsdb`, its own
     generated package like every other context's, plus
     `db/queries/competitions.sql`.
   - `internal/competitions/adapter/postgres` implementing `port.Repository`,
     following the per-query `fromFields` pattern (`HANDOFF.md`'s gotcha:
     sqlc generates a distinct `...Row` struct per query — don't assume one
     shared row type).
   - `proto/pickleball/competitions/v1/competitions.proto` with
     `CreateCompetition`, `GetCompetition`, `ListCompetitions`,
     `EnterCompetition`, `CancelCompetition`, and
     `ListEntriesForCompetition` — mirroring `socialplay.proto`'s existing
     message/RPC/HTTP-annotation style, including a server-computed
     `spots_left` on the listing message that uses the **weighted** formula
     (T8.9's `ListGames` is the precedent; a boundary test must prove the
     weighting, since an unweighted `spots_left` would be a visible lie the
     moment anyone brings a guest). Run `make generate`.
   - `internal/competitions/adapter/grpcapi` + `cmd/server` wiring.
     `toStatus` maps `ErrCompetitionFull` → `FailedPrecondition`/409-shaped,
     `ErrFacilityNotFound` → `NotFound`, `ErrNotCompetitionHost` →
     `PermissionDenied`, `ErrGuestAllowanceExceeded` →
     `InvalidArgument` — **never** `Internal` for any of them.
2. Non-functional requirements:
   - **Authorization regression proof, per the kickoff note's process
     decision** (this replaces what would previously have been a separate
     authz ticket): a handler-level test through the real
     `grpcapi.Handler → app.Service → domain.Competition` path proving a
     non-Host actor's `CancelCompetition` is rejected as
     `PermissionDenied` (not `Internal`, not a silent success) and that no
     state changed. **Verify it is non-vacuous by temporarily
     short-circuiting `EnsureHost` to return `nil`, confirming the test
     fails, then restoring it** — state the observed failure output in the
     PR (CLAUDE.md rule 10; QA dossier §5 item 2; the exact method T7.7 and
     T8.5 used).
   - Same caveat, stated once, not re-litigated: object-level check given a
     claimed `actor_user_id`, not authentication.
   - This ticket must not modify `internal/booking/**` or its migrations.
     Verify and state it.
   - `make down && make up` after the schema change (docker initdb.d only
     applies migrations on a fresh volume — `HANDOFF.md` gotcha); confirm
     existing T5–T8 integration tests still pass unchanged.
   - `make test` green including the `-tags=integration` suite.

**Story points:** 8

**Labels:** `sprint:t9`, `role:principal-engineer`, `type:story`, `points:8`

---

### T9.5 — Shareable registration links: token-addressed public read + in-app RSVP with source attribution

**Story:** As a Host, I want a link I can post anywhere that lets people
register for my competition in the app, and as a Player, I want tapping
that link to take me straight to entering — so that social promotion
produces accurate, structured registrations without the platform reading
anyone's replies.

**Description:** Depends on T9.4. This is the **locked** half of T8's
sub-scope (d) — `docs/design/v1-system-design.md` §4 and
`docs/design/v1-external-reference-reconciliation.md` both resolve
reply-to-register to "in-app RSVP via a shareable link, not reply
parsing," and two independent passes landing there is treated as
corroboration, not re-opened. The OAuth half is split out (§A1, ADR-0009).
This ticket also finally gives `RegistrationSourceSocial`'s Competitions
equivalent a real producer (§A3).

**Instructions:**
1. Functional requirements:
   - Failing tests first.
   - `port.ShareTokenGenerator` (declared in T9.3) implemented in
     `internal/platform` or a context-local adapter using
     **`crypto/rand`**, not `math/rand`, not a sequential counter, not the
     Competition's own ID. Minimum 128 bits of entropy, URL-safe encoding.
     A guessable token would make every unpublished Competition
     enumerable — state the entropy choice and its reasoning in the PR.
   - `GetCompetitionByShareToken` RPC: a **public, unauthenticated read**
     keyed by token, returning the same public projection
     `GetCompetition` returns — explicitly **no** additional fields.
     Required test: the token-addressed response is field-for-field the
     same shape as the ID-addressed one, so a link never discloses more
     than the app already shows. (BA dossier §5, silent scope narrowing —
     inverted: this is silent scope *widening*, and it's the likelier bug.)
   - An unknown or malformed token returns `NotFound` — the same status,
     with the same message, as a well-formed token for a Competition that
     doesn't exist, so the endpoint isn't an oracle for which tokens are
     real.
   - `EnterCompetitionRequest` gains a `source` field. When the entry
     arrives via a share link the client sends `social`; otherwise `app`.
     The server **validates** it against the closed enum and stores it —
     it does not infer it.
   - `ListEntriesForCompetition` returns each entry's `source` so T9.6's
     roster can show the channel.
   - A cancelled Competition's token still resolves, and returns the
     Competition with `status: cancelled`; entering it is rejected.
     Given/When/Then: *Given* a Host cancelled a Competition after posting
     its link, *When* a Player follows that link, *Then* they see an
     honest "this competition was cancelled" state rather than a 404 that
     looks like a broken link (NN/g heuristic #9 — diagnose and recover;
     a dead link and a cancelled event are different facts).
2. Non-functional requirements:
   - **No reply-scraping, no platform API reads, no webhook ingestion of
     third-party messages anywhere in this ticket.** The link is outbound
     only; every registration enters through this platform's own API. If an
     implementer finds themselves adding an inbound integration, stop —
     that is ADR-0009's deferred scope, not this ticket's.
   - **Revocation is out of scope and must be named, not silently
     omitted:** there is no rotate/revoke-token path in T9. Document it in
     the PR and in the proto's doc comment as a known gap with its trigger
     (build it when a Host can be authenticated, i.e. alongside real auth)
     rather than leaving a reader to assume tokens are revocable.
   - Add `Shareable Registration Link` and a precise definition of the
     `social` entry/registration source to `agent-operating-handbook.md`
     A2's glossary in this PR (CLAUDE.md rule 7; §A3's exact wording).
   - `make test` green.

**Story points:** 5

**Labels:** `sprint:t9`, `role:principal-engineer`, `type:story`, `points:5`

---

### T9.6 — Vue: Create, advertise, and manage a Competition (Host)

**Story:** As a Host, I want to create a competition at a real facility,
get a ready-to-post promo with its registration link, and see who has
entered and how they got there, so that "organize a competition and promote
it" (Flow 5) is a real, working flow rather than a wireframe.

**Description:** Depends on T9.4, T9.5, and T9.2 (fee-input pattern).
Implements Flow 5's "Create competition → [link accounts] → Publish ad →
Manage roster," with the accounts step replaced per §A1 by an honest share
composer. Reuses T8.8's `GameCreation.vue` multi-step-form patterns rather
than inventing a second wizard idiom — one component family with variants,
not two bespoke flows (UX dossier §1 synthesis).

**Instructions:**
1. Functional requirements:
   - Route `/competitions/new` and `/competitions/:id/manage`, added to
     `web/src/router/index.ts` and to `AppNav.vue`'s existing nav structure
     (the nav's tab set is fixed by T8.1's design-handoff mapping —
     Competitions belongs under the existing Games area rather than
     inventing a fifth top-level tab; state the choice in the PR).
   - Multi-step create form: venue & courts (reuse T7.5/T8.8's facility +
     courts pickers against the real `ListFacilities` /
     `GetFacilityResponse.courts`) → **sessions** (add/remove one or more
     date+time+courts rows; a Competition spans dates, so this step is
     genuinely different from a Game's single range and is the one place
     this form must not just copy `GameCreation.vue`) → capacity, guest
     allowance, format → payment method & entry fee (reuse T9.2's fee
     input) → review & publish, calling `CreateCompetition`.
   - Client-side validation mirroring T9.1's domain rules as a UX nicety —
     overlapping sessions on the same court are flagged **at entry**, next
     to the offending row, before submit (NN/g heuristic #5, error
     prevention over correction). The server check remains authoritative;
     this is not a substitute for it and the PR must say so.
   - **Advertise step:** an auto-formatted promo (name, format, dates,
     venue, entry fee or "Free", spots, and the share URL from T9.5) with
     a **Copy** button and the Web Share API where available. Honest
     labelling per §A1: no "Connect account" buttons, no per-platform
     Connected/Connect state, and a one-line note that automatic posting to
     linked accounts isn't available yet — the same convention T8.8 used
     for the omitted matching step, not a dead control.
   - Copy success is announced in an ARIA live region (`role="status"`) —
     WCAG 4.1.3, and NN/g heuristic #1: a clipboard write with no feedback
     is invisible.
   - **Roster view:** entries with name, guest count, payment status, and
     **source channel**. Source renders as "In app" or "Via shared link" —
     never the design attachment's verbatim "via WhatsApp reply", per
     `docs/design/v1-external-reference-reconciliation.md`'s explicit
     instruction.
   - Unpaid-cash entries surface using the pattern `HostPayments.vue`
     already established (T8.10), reusing its components rather than
     duplicating the badge/list.
   - Once a Competition is created, `RoleIndicator` recognizes "Host" for
     the session (kickoff note decision #1) — extend the existing
     mechanism, don't add a second one.
   - Matching controls are **absent**, with the same honest inline note
     T8.8 uses ("Automated matching isn't available yet — players enter
     directly"), per §A2.
2. Non-functional requirements:
   - Responsive across all three breakpoints (web ≥1280 sidebar, iPad
     768–1180 two-column, iPhone <600 single-column) per
     `docs/design/v1-external-reference-reconciliation.md`'s adopted
     numbers; touch targets ≥44px on iPad/iPhone (Apple HIG, stricter than
     WCAG 2.5.8's 24px floor — UX dossier §3.3).
   - WCAG 1.4.1: payment status and entry source carry text, never color
     alone. WCAG 3.3.1: field-level error text next to the field.
   - Component tests (Vitest): the sessions step adds/removes rows and
     retains state across forward/back navigation; overlapping-session
     validation fires at entry; the share payload contains the real token
     URL (not a placeholder); the copy action announces success; **assert
     the absence** of any matching control and of any "Connect account"
     affordance, not merely the presence of what's there (T8.8's precedent
     — absence assertions are the only ones that catch a well-meaning
     future re-addition).
   - No fabricated data anywhere: every field displayed must come from a
     real API response. If something has no backend home, it does not
     render.

**Story points:** 8

**Labels:** `sprint:t9`, `role:ux-ui-designer`, `type:story`, `points:8`

---

### T9.7 — Vue: Discover a Competition, and enter via link or in-app (Player)

**Story:** As a Player, I want to browse competitions or open a link a host
posted, see what it costs and how many spots are left, and enter with my
guests, so that "find and enter a competition" is a real, working flow
whether I arrived from the app or from a shared link.

**Description:** Depends on T9.4 and T9.5. Mirrors T8.9's
Discover-&-Join-Games structure (list + detail + join panel) and reuses its
components where the shape overlaps. The distinguishing surface is the
**deep-link landing path** — a first-time visitor arriving on a token URL,
who may have no prior context in the app at all.

**Instructions:**
1. Functional requirements:
   - Route `/competitions` (list, filterable by facility and date range
     against `ListCompetitions`) and `/competitions/:id` (detail: host,
     venue, every session's date/time/courts, format, spots left, entry
     fee or "Free", payment method).
   - **Deep-link route `/c/:shareToken`** calling
     `GetCompetitionByShareToken` and rendering the detail view with the
     entry action prominent — this is the path a shared link lands on, and
     an entry started from it sends `source: social`; one started from
     `/competitions` sends `source: app`. Required test: the two paths
     produce different `source` values on the wire.
   - Entry flow: guest-count stepper bounded by `GuestAllowance`
     (client-side nicety only — the server check stays authoritative, say
     so in the PR), calling `EnterCompetition`. A capacity rejection
     (`ErrCompetitionFull`, 409) surfaces a specific, actionable message
     ("This competition just filled up") — WCAG 3.3.3 Error Suggestion,
     the same rule T7.6 and T8.9 applied. **There is no waitlist for
     Competitions in T9** (Social Play's waitlist is Game-scoped, T6.6 /
     ADR-0006); do not offer one, and say in the PR that the absence is
     deliberate so a reviewer doesn't read it as a missed reuse.
   - Payment follows T8.10's established paths unchanged: online → the
     existing checkout step; cash → "pending, pay at facility". Reuse
     those components; do not build a second checkout.
   - **Cancelled-competition state:** a share link to a cancelled
     Competition renders an honest cancelled state with a route onward to
     `/competitions`, not a 404 and not a silently disabled button
     (T9.5's Given/When/Then; NN/g heuristic #9).
   - Empty / loading / error states for list, detail, and the deep-link
     landing (unknown token) — real designed UI for each, never a blank
     screen (UX dossier §5, empty-state guidance). The unknown-token state
     says the link is invalid or expired and offers browsing instead.
   - Skeleton screens for the full-page loads, spinners only for isolated
     components (UX dossier §5, NN/g loading guidance) — matching whatever
     T8.9 already established rather than introducing a third loading
     idiom.
2. Non-functional requirements:
   - Responsive across all three breakpoints; ≥44px touch targets on
     iPad/iPhone.
   - WCAG 1.4.1: spots-left urgency and "Cash at facility" vs. price carry
     text, not color alone. WCAG 2.1.1/2.4.7: the deep-link landing is
     fully keyboard operable with a visible focus indicator — it is the
     one screen a brand-new user may hit first, so it cannot rely on
     prior navigation state.
   - Component tests (Vitest): deep-link vs. in-app `source` attribution;
     guest stepper bounded by `GuestAllowance`; the competition-full
     rejection path; the cancelled-competition landing; unknown-token
     landing; empty/loading/error states.
   - No fabricated fields — same rule as T9.6.

**Story points:** 8

**Labels:** `sprint:t9`, `role:ux-ui-designer`, `type:story`, `points:8`

---

### T9.8 — Spike: owned-channel messaging bot + social-account-link credential custody (ADR-0009)

**Story:** As the team, I want a written, decided position on how (and
when) this platform integrates an owned messaging channel and stores hosts'
social credentials, so that the next sprint that touches it starts from a
decision rather than re-opening the question a fourth time.

**Description:** T8's roadmap named "a scoped WhatsApp-or-Zalo
owned-channel-bot spike." **T7's research is not repeated** — 
`docs/process/t7-sprint-plan.md`'s "Social platform integration: WhatsApp
vs. Zalo, researched" section was re-checked for staleness by T8's ceremony
and found **not stale** (the follow-based vs. message-based opt-in
difference, Zalo OA's tiered verification cost, and the
Vietnam-vs-global market-scope question are facts about the platforms, not
project state). This ticket **cites** it and adds only what T7 explicitly
left as a T8/T9 scoping input, plus the credential-custody finding this
ceremony raised (§A1). A spike, not a feature: **it produces an ADR and
nothing else** — no vendor account, no credentials, no adapter package, no
proto.

**Instructions:**
1. Functional requirements:
   - Write `docs/adr/0009-social-channel-integration-deferred.md` recording:
     - **Decision:** no OAuth token storage and no inbound messaging
       integration until real authentication exists. Shareable registration
       links (T9.5) are the shipped mechanism for social-driven
       registration in the meantime.
     - **Context:** the credential-custody argument in §A1 — that an OAuth
       token keyed to an unverified `actor_user_id` differs *in kind*, not
       degree, from the three existing object-level-authz caveats
       `HANDOFF.md` records, because the asset guarded is a third party's
       account rather than this platform's own data.
     - **The `port.MessagingChannel` shape**, designed on paper only —
       `SendMessage` / `ParseInboundReply`, per T7's recommendation, behind
       the same ACL pattern `port.PaymentProcessor` (T6.2) already
       established. Name it, sketch its Go interface in the ADR, and
       explicitly do **not** create the package.
     - **Platform choice recommendation** with the tradeoff stated once
       (cite T7's section; do not re-derive): prototype **one** platform,
       not both, and the choice is gated on the Vietnam-vs-global
       market-scope question T7 already escalated to PM/PO and which
       remains unanswered — record it as still-open, addressed to the
       user, rather than picking for them.
     - **Trigger condition:** revisited in the sprint that lands real
       authentication (currently recommended as T10 — §A5), with a named
       list of what must exist first (verified identity, encrypted
       token-at-rest story, revocation path).
   - Update `HANDOFF.md`'s Docs index and Cross-cutting section to point at
     the ADR, so the decision is discoverable from the map, not only from
     this sprint plan.
2. Non-functional requirements:
   - **Timebox: the ADR only.** If the spike starts producing an adapter,
     a proto, or a vendor signup, it has left its scope — stop and flag
     (PdE dossier §3 heuristic 5, circuit breaker).
   - The ADR must not contradict `docs/design/v1-system-design.md` §4 or
     `docs/design/v1-external-reference-reconciliation.md`; it extends the
     already-locked "channel you control, not public reply scraping"
     position with a *timing* decision, and must say so explicitly so it
     doesn't read as a new position on a locked one.
   - No credentials, API keys, or vendor accounts are created or committed.

**Story points:** 3

**Labels:** `sprint:t9`, `role:principal-engineer`, `type:spike`, `points:3`

---

### T9.9 — ADR-0010: record the auto-matching decision and its trigger condition

**Story:** As the team, I want the auto-matching decision written down as
an ADR with an explicit trigger, so that a fifth sprint cannot defer it by
default and the two outstanding product decisions reach the person who can
actually make them.

**Description:** T8's plan required T9's Ceremony 1 to "explicitly decide
whether... to justify building a minimal `PlayerRating` this sprint, or
whether it stays deferred again with a real reason recorded (not silently
dropped a fourth time)." §A2 is that decision. This ticket is the artifact
that makes it durable — per CLAUDE.md's Definition of Done ("ADR written if
an architectural decision was made") and PE dossier §5 (a debt item with no
stated trigger is a decision never to fix it). Independent of every other
T9 ticket.

**Instructions:**
1. Functional requirements:
   - Write `docs/adr/0010-auto-matching-deferred-to-identity-context.md`
     recording, per §A2:
     - **Decision:** auto-matching / `PlayerRating` / `Level` is not built
       in T9; it is built in the sprint that stands up **Identity/Users**,
       and not before.
     - **Context, three parts:** (a) `agent-operating-handbook.md` A1
       assigns `Level` to Identity/Users, which does not exist — verified
       this ceremony by grep, with the result stated; (b) this project
       already paid the cost of parking a cross-context field in the wrong
       context (`games.facility_id`, reconciled at real expense by T8.3 —
       a migration, a new port, a new adapter, and two deprecated
       artifacts still in the tree), and knowingly repeating it is a
       repeat, not a tradeoff; (c) T9 builds no brackets (§A4), so the
       seeding call site that would have justified a minimal rating this
       sprint does not exist.
     - **Two escalations, addressed to the user, not resolved here:** the
       Level formula's weighting (the design handoff's own open question —
       "needs a product decision on weighting before auto-matching logic
       is finalized") and whether gender-mix matching is in scope at all,
       given it means collecting and algorithmically acting on a protected
       attribute — the same product/legal class as the two items already
       awaiting sign-off in `docs/design/v1-system-design.md`'s top
       blockquote. Both must be stated as questions with the decision each
       one blocks, not as background.
     - **Trigger:** T10's Ceremony 1 must either build it or supersede
       this ADR with a new one. It may not defer it in prose.
   - Update `HANDOFF.md`'s Docs index (T9 row) and Cross-cutting section to
     reference ADR-0010, and remove/replace any Cross-cutting wording that
     still describes matchmaking as deferred without a home.
2. Non-functional requirements:
   - The ADR must state what would *change* the decision (new evidence,
     not a preference — PE dossier §5 on defending locked decisions while
     staying open to evidence).
   - It must not silently supersede `CLAUDE.md`'s locked decision that
     matchmaking is in v1 scope ("automated from history, always manually
     overridable; new players seeded by a self-reported starting level").
     This is a **sequencing** decision, not a scope reversal — say so
     explicitly, since an ADR that reads as a cancellation of a locked
     decision would be reopening one, which the PO brief forbids.

**Story points:** 2

**Labels:** `sprint:t9`, `role:product-owner`, `type:chore`, `points:2`

---

### T9.10 — Chore: remove the `npm install --legacy-peer-deps` requirement

**Story:** As a developer or CI job building the web client, I want `npm
install` to work without a workaround flag, so that a fresh checkout builds
the same way for everyone and CI (when it lands) doesn't need a special-case
install step.

**Description:** `HANDOFF.md`'s Cross-cutting section: every T7 and T8
implementer sandbox — 16 tickets' worth of builds — needed `npm install
--legacy-peer-deps` because of `typescript@~6.0.0` versus
`openapi-typescript`'s `^5.x` peer range. Confirmed working across all of
them, so it is friction rather than a real incompatibility, but T8's plan
pre-committed to raising it as "a real, small T9/T10 ticket rather than
logging it a fourth time" (§A6). Independent of every other T9 ticket.

**Instructions:**
1. Functional requirements:
   - Determine which side to move: check whether a newer
     `openapi-typescript` widens its TypeScript peer range, or whether
     pinning `typescript` to a version inside the existing range is
     cheaper. Prefer whichever needs no change to generated output.
   - Verify `npm ci` and `npm install` both succeed **without** the flag
     from a clean checkout (delete `node_modules` and the lockfile's
     resolution cache as appropriate), and that `npm run build`,
     `npm run test`, and `npm run generate:client` all still pass.
   - If the generated client output changes at all, treat that as a
     blocking finding and stop — this ticket is not authorized to change
     API surface as a side effect of a dependency bump. Report it and
     split.
   - Remove the `--legacy-peer-deps` instruction from every doc that
     carries it, and add a line to `HANDOFF.md`'s Cross-cutting section
     recording that this three-sprint-old item is closed (strike-through
     style, matching how the T8.2/T8.3/T8.4 closures were recorded).
2. Non-functional requirements:
   - No unrelated dependency upgrades bundled in. A dependency PR that
     also bumps five other packages is unreviewable — keep the diff to the
     one problem.
   - State the before/after `npm install` output in the PR, so the claim
     is checkable rather than asserted (there is still no CI signal —
     §A6).

**Story points:** 2

**Labels:** `sprint:t9`, `role:product-engineer`, `type:chore`, `points:2`

---

## Ticket index (GitHub issues)

| Ticket | Issue | Title | Points | Primary role |
|---|---|---|---|---|
| T9.1 | [#70](https://github.com/nhuthuynh/pickleball-platform/issues/70) | Add the Competition and CompetitionEntry domain model (pure domain, TDD) | 5 | Principal Engineer |
| T9.2 | [#71](https://github.com/nhuthuynh/pickleball-platform/issues/71) | Give a Game a real `EntryFee`, retiring T8.10's disclosed placeholder | 3 | Principal Engineer |
| T9.3 | [#72](https://github.com/nhuthuynh/pickleball-platform/issues/72) | Competitions application service: reserve courts, validate the venue, enforce Host ownership | 5 | Principal Engineer |
| T9.4 | [#73](https://github.com/nhuthuynh/pickleball-platform/issues/73) | Wire Competitions to Postgres + proto + gRPC/REST, with a weighted DB capacity guard and the authz regression proof | 8 | Principal Engineer |
| T9.5 | [#74](https://github.com/nhuthuynh/pickleball-platform/issues/74) | Shareable registration links: token-addressed public read + in-app RSVP with source attribution | 5 | Principal Engineer |
| T9.6 | [#75](https://github.com/nhuthuynh/pickleball-platform/issues/75) | Vue: Create, advertise, and manage a Competition (Host) | 8 | UX/UI Designer |
| T9.7 | [#76](https://github.com/nhuthuynh/pickleball-platform/issues/76) | Vue: Discover a Competition, and enter via link or in-app (Player) | 8 | UX/UI Designer |
| T9.8 | [#77](https://github.com/nhuthuynh/pickleball-platform/issues/77) | Spike: owned-channel messaging bot + social-account-link credential custody (ADR-0009) | 3 | Principal Engineer |
| T9.9 | [#78](https://github.com/nhuthuynh/pickleball-platform/issues/78) | ADR-0010: record the auto-matching decision and its trigger condition | 2 | Product Owner |
| T9.10 | [#79](https://github.com/nhuthuynh/pickleball-platform/issues/79) | Chore: remove the `npm install --legacy-peer-deps` requirement | 2 | Product Engineer |

## Dependency order

Implementers should be dispatched in this order. Tickets on the same line
are independent of each other and may run in parallel.

```
T9.1 (Competition domain)  ·  T9.2 (Game entry fee)  ·  T9.8 (ADR-0009)  ·  T9.9 (ADR-0010)  ·  T9.10 (npm chore)
   ↓
T9.3 (app service, ports, adapters, EnsureHost)
   ↓
T9.4 (postgres + proto + gRPC + capacity guard + authz regression)
   ↓
T9.5 (shareable registration links)
   ↓
T9.6 (Host UI)  ·  T9.7 (Player UI)          ← T9.6 also depends on T9.2
```

T9.6 and T9.7 both touch `web/src/router/index.ts` and will conflict there
if run in parallel — T8.8↔T8.9 hit exactly this (`HANDOFF.md` records the
`router/index.ts` + duplicate-client conflict). Either sequence them, or
have whichever lands second resolve on its own source branch and re-verify
(`npm run build` + `npm run test`) before merge. Never resolve on the
shared branch.

## Sprint totals

- **Tickets:** 10 (T9.1–T9.10)
- **Total story points:** 49 (5 + 3 + 5 + 8 + 5 + 8 + 8 + 3 + 2 + 2)
- **Composition, for calibration against T5 (27 points / 5 tickets), T6
  (37/7), T7 (40/7), T8 (51/10):** 26 points (T9.1–T9.5) are backend —
  a new bounded context's full domain→app→port→adapter stack plus the
  share-link path; 16 points (T9.6, T9.7) are user-facing screens; 7
  points (T9.8–T9.10) are decisions and debt paydown. Slightly under T8's
  total, which is deliberate: T8's 51 was inflated by carrying four
  separate pre-existing gaps, whereas T9's headline item is a 0→1 bounded
  context, historically this project's most expensive shape (T5's Social
  Play was 27 points on its own). The reason T9 can carry a fifth context
  *and* a UI pair *and* three side items is §A0 finding 2 — Booking
  already accepts `competition`-source bookings and already has the
  cross-source conflict tests, so the riskiest part of a new
  court-reserving context is already proven.
- **Sequencing discipline:** T9 ships backend before UI within the sprint
  (T9.1→T9.5 before T9.6/T9.7), the same domain→app→port→adapter→UI order
  CLAUDE.md rule 3 requires and the trap PE flagged in T8's re-scope
  notice (don't build UI against fields or RPCs that don't exist yet).
  Every field T9.6/T9.7 renders has a backend home landed by an earlier
  T9 ticket — verified ticket by ticket at Ceremony 2.
- **Open questions resolved this ceremony:** auto-matching (#2 from the
  reconciliation note) is **decided**, with an ADR and a trigger, ending a
  four-sprint deferral chain; the social-account-linking scope is decided
  and split with a written blocker; the `RegistrationSourceSocial`
  never-produced-value gap is given a definition and a producer; the
  `Format`-vs-`CancellationCutoff` consistency question is resolved with a
  stated test ("does the field imply enforcement?").
- **Explicitly deferred, not forgotten:** OAuth social-account-linking and
  automated posting (→ post-auth, ADR-0009); competition brackets/rounds/
  seeding/results (→ T11+, §A4); shareable links for Games (→ T10
  refinement, §A3); share-token revocation (→ with real auth, T9.5);
  competition waitlists (→ no ticket, ADR-0006 is Game-scoped);
  Jenkins/CI wiring (→ **T10, named** — §A6 and the disagreement below);
  Identity/Users context (→ T10, §A5); `CancellationCutoff` enforcement
  (→ still no concrete driver, unchanged from T8); pricing/discount UI,
  Club account type, WCAG 2.2 AA audit (→ T10/T11, unchanged); native
  Swift/Kotlin clients (→ unchanged).

## Genuine disagreements recorded (not manufactured consensus)

**1. PM vs. PE — cutting the OAuth half of social-account-linking.**
PM's position: T8's roadmap named social-account-linking for T9, hosts
advertising through their own audience *is* the growth loop, and cutting
the automated half leaves Flow 5 shipping a copy-paste step that the
wireframe doesn't draw — the risk being that "advertise via social" becomes
a permanently half-built feature. PE's position: §A1's credential-custody
argument — an OAuth token keyed to an unverified actor is a one-way door
guarding *someone else's* account, categorically unlike the three
object-level-authz caveats already on the books. QA seconded PE
independently, on the narrower ground that there is no test that could
demonstrate the token store is safe, because the thing that would make it
safe (authentication) doesn't exist to be tested.
**Resolution: PE's position prevailed, with PM's condition attached.**
The decomposition table in §A1 is what settled it — once the team listed
which Flow 5 capabilities actually need a credential, only "post without
the host pasting" did, and PM accepted that four of five rows shipping is
not a half-built feature. PM's condition, accepted and ticketed: the
Advertise UI must be honest about the gap rather than drawing a dead
"Connect" button (T9.6), and the deferral must carry a trigger condition in
an ADR rather than living in prose (T9.8). The Designer supported the
condition on independent grounds (a "Connect" button that connects nothing
is a catalogued deceptive pattern, UX dossier §6).

**2. PM vs. PE — whether to ship at least the self-reported starting level
in T9.** PM's position: the self-reported starting level is the *cold-start
seed*, explicitly named in CLAUDE.md's locked decisions, requires no Match
history, no algorithm, and no product decision on weighting — and shipping
it would let Flow 4's level filter chips (already drawn in the design)
finally do something, converting a four-sprint deferral into partial
progress. PE's position: a `Level` on a Player, in a codebase with no
Player entity, has to be parked in Social Play or Competitions, which is
an Identity concept in the wrong context — and this project has a priced
receipt for that exact mistake (`games.facility_id`, §A2 part 1), not a
hypothetical one. PdE sided with PE on cost: the field is cheap, the
reconciliation is not, and T8.3 is the evidence.
**Resolution: PE's position prevailed, with PM's condition attached.**
PM accepted on two conditions, both ticketed: T10's headline scope is
**named now** as Identity/Users (§A5) so this cannot slip by default a
fifth time, and ADR-0010 (T9.9) must carry a trigger obliging T10's
Ceremony 1 to either build it or supersede the ADR — it may not defer it
in prose. PO ratified that framing as the difference between a decision
and a deferral.

**3. PE vs. PdE — whether CI lands in T9 or T10.** PE's position: four
sprints of "tests pass" claims with zero independently-checkable check
runs is now the oldest unaddressed risk in the repo, and every sprint that
passes makes the eventual first-CI-run more likely to surface a backlog of
failures at once. PdE's position: T9 already carries a 0→1 bounded context
plus two UI screens, and CI is not the small chore it looks like here —
`CLAUDE.md` locks **Jenkins** as the stack decision while the
check-run signal this repo actually lacks is a GitHub one, so choosing
between them touches a locked decision and needs PO/user input before an
implementer starts, not during. QA sided with PE on the risk and with PdE
on the sequencing.
**Resolution: PdE's position prevailed on timing; PE's on treatment.** CI
does not enter T9, but it is **named as T10 scope with a stated ordering
priority** (§A5) rather than flagged in prose for a fourth sprint, and the
locked-decision question (Jenkins pipeline vs. GitHub check runs, or both)
is escalated to the user now — in this document — so T10's Ceremony 1
starts with an answer instead of the same debate. PE's condition, accepted:
the smaller of the two three-sprint-old chores (`--legacy-peer-deps`) is
taken **this** sprint (T9.10), so the pattern of both aging together is
broken rather than deferred wholesale.

**4. BA vs. the room — `Format` as a non-enforcing field.** BA raised it as
a possible inconsistency with T8's own reasoning for dropping
`CancellationCutoff` ("a cosmetic text field that enforces nothing would be
actively misleading"). Recorded because the challenge was correct to make
even though it did not change the outcome: the team's answer (§A4 — the
test is whether the field *implies enforcement*, not whether it has code
behind it) is now written into T9.1's Instructions as a required doc
comment, so the next reader hits the reasoning rather than the apparent
contradiction. This is the disagreement-resolution outcome the process is
supposed to produce: not a reversal, but a documented rule that stops the
question being re-asked from scratch.

## Escalations to the user (not resolvable in an engineering ceremony)

Listed together because three of the items above end in the same place, and
burying them in ticket text would repeat the "flagged in prose" failure
mode this project keeps naming:

1. **Level formula weighting** and **whether gender-mix matching is in
   scope at all** — blocks auto-matching's design, not just its schedule
   (T9.9 / ADR-0010). The second is a product *and* legal question:
   collecting and algorithmically acting on a protected attribute.
2. **Market scope: Vietnam-concentrated or global?** — T7 escalated this
   and it is still unanswered; it decides WhatsApp vs. Zalo and therefore
   whether the messaging-channel work is worth starting at all
   (T9.8 / ADR-0009).
3. **CI: Jenkins pipeline, GitHub check runs, or both?** — `CLAUDE.md`
   locks Jenkins for the stack; the missing signal is a GitHub one. T10's
   scope depends on the answer (§A6, disagreement 3).
