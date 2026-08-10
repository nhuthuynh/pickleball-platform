# T10 Sprint Plan — Identity/Users + Match, and three T9 follow-ups

Ceremonies 1 and 2 per `docs/process/sprint-process.md`, six-role team
(briefs: `docs/agent-operating-handbook.md` Part B), against `HANDOFF.md`'s
T9 entry, `docs/adr/0010-auto-matching-deferred-to-identity-context.md`'s
binding trigger, `docs/process/t9-retro.md`'s adopted process changes, and
`docs/design/handoff-2026-08/README.md`'s open questions.

---

# Part A — Ceremony 1 (backlog refinement)

**Product Manager + Principal Engineer**, per `sprint-process.md`. PM drove
scope/value framing; PE drove technical sequencing/feasibility. Both sign
off on every ticket below before it was opened.

## A0. What ADR-0010's binding trigger actually requires, and why this isn't exit (a) as first assumed

ADR-0010's trigger is explicit: T10's Ceremony 1 must either (a) build
auto-matching in full, or (b) supersede the ADR with a new, numbered ADR
carrying its own trigger — "it may not defer it again in prose," and if
the two escalated questions (Q1: Player Level formula weighting, Q2:
whether gender-mix matching is in scope) are still unanswered when this
ceremony runs, the ADR itself says that "does not license a fourth
roll-forward: it licenses exit (b)."

Both questions are still unanswered — this ceremony has no authority to
answer them (`docs/agent-operating-handbook.md` B5, B6; this sprint's own
instructions bar any role from inventing an answer on the user's behalf).
So exit (a) in full is not honestly available: ADR-0010 itself states Q1
"blocks... the shape of `PlayerRating` and `Level` themselves," and that
guessing here reproduces the exact mistake the ADR exists to prevent.

**PE's ruling, PM concurring:** exit (b) does not mean "defer again with a
new number" — the ADR's own text forecloses that reading. It means:
build everything that does not require Q1/Q2's answer, name precisely
what remains blocked and why, and set a trigger tied to the user
answering, not to another ceremony's judgment call. **`docs/adr/
0012-identity-users-and-match-built-rating-and-matching-algorithm-blocked-on-escalated-decisions.md`**
is that ADR, written and merged as part of this ceremony (not a ticket for
later). Full reasoning, the piece-by-piece decomposition of "auto-matching"
into what's blocked vs. buildable, and the new trigger are there in full;
not restated here beyond the summary below.

**Resolution, in one line:** Identity/Users and `Match` are built this
sprint (T10.1–T10.4). `PlayerRating`, the rating/matching algorithm, and
any `Gender`/gender-mix matching are **not** built, named as blocked on
Q1/Q2 specifically (not blocked on "matchmaking generally"), with a
trigger tied to the user's answers landing.

**QA's condition for signing off on this reading:** the ticket text for
T10.1–T10.4 must say, explicitly, what is *not* being built and why —
not just what is. Confirmed present in every ticket below (see each
ticket's "Explicitly not built" line).

## A1. Cross-context dependency check (T9 retro finding 5, applied)

For every ticket below: which existing contexts does it call into, and
does the enum/port/type it needs actually have a member for this context?
Checked at planning time, not discovered at the UI layer this time.

| Ticket | Calls into | Member/port exists? |
|---|---|---|
| T10.1 (Identity domain) | none — new context | n/a |
| T10.2 (Identity wiring) | none new | n/a |
| T10.3 (Match domain) | Social Play's own `Game` (existing) | Yes — `domain.Game` already has an ID `Match` can reference |
| T10.4 (Match wiring) | Social Play's own Host/Game-Admin authorization (existing) | Yes — `EnsureHost`-shaped pattern already established (T5.1/T5.4) |
| T10.5 (profile + disclosure UI) | `identity` (T10.2's `GetUser`), `socialplay` (T10.4's match list, read-only) | Both exist by the time this ticket starts (dependency order below) |
| T10.6 (Competitions `PayableType`) | `payments.PayableType` enum, `internal/payments/adapter/competitions` (new) | **Checked, not assumed**: enum genuinely has no member yet (verified by re-reading `internal/payments/domain/payment.go` this ceremony) — this ticket adds it, mirroring T6.5's Social Play precedent exactly |
| T10.7 (write-handler guard) | five existing contexts' write handlers | No new member needed — pure boundary validation reusing the existing UUID-shape helper from PR #89's Layer 2 |
| T10.8 (display-name join) | `facilities.GetFacility` (existing, has `Name`), `identity.GetUser` (T10.2, needs `DisplayName`) | Facilities: yes, already returns `Name` (T8.2). Identity: **only after T10.1/T10.2 add `DisplayName`** — confirmed this ceremony that the field does not exist yet anywhere in this codebase, so T10.1's Instructions require it explicitly rather than leaving T10.8 to discover its absence mid-sprint |

No gap found that mirrors the T9.6/T9.7 `PayableType` surprise — the one
enum genuinely missing a member (`payments.PayableType` for Competitions)
was already known going into this ceremony (it's issue #96, opened this
ceremony) rather than discovered by an implementer.

## A2. The roadmap-carryover check — "gated on real auth" does not mean gated on everything

`HANDOFF.md`'s T9 entry describes the former T9 roadmap items — now T10 —
as "pricing/discount UI, Club rentals, WCAG hardening — both gated on
real auth per ADR-0009/ADR-0010's triggers." Taking that at face value
would mean none of the three is available this sprint. **Checked, not
assumed:**

- **Pricing/discount UI (Flow 2, `discount_rules` backend).** Not
  actually gated on real auth. A `CreatePricingRule`/discount-period write
  path needs the same claimed-`actor_user_id` object-level check every
  other write path in this codebase already has (T5.5/T6.7→T8.5/T7.7
  precedent) — it does not need JWT/Auth0 to exist first, any more than
  `AddCourt` did. **Independently buildable now.**
- **Club rentals (Flow 7).** Partially coupled, not gated. The
  *recurring-booking mechanics* (a `recurring_hire`-source Booking
  generated from a template) need no more auth than any existing Booking
  path. What genuinely benefits from sequencing after Identity/Users:
  "Club" is named in `docs/agent-operating-handbook.md` A1 as one of
  Identity's roles (`player, host/organiser, game admin, facility owner,
  club, platform admin`) — building the Club account type before
  Identity/Users exists would risk the exact `games.facility_id` mistake
  ADR-0010 (b) already paid for once: a role concept parked in the wrong
  place because its real home didn't exist yet. T10.1 adds a `club` role
  value to `User.Roles` for exactly this reason (see T10.1's AC), so this
  is no longer a blocker once T10.1 merges — but the full rental flow
  remains real, non-trivial work.
- **WCAG 2.2 AA audit.** Not gated on anything. Zero auth dependency —
  pure accessibility work across existing shipped screens.

**PM/PE's actual scoping decision, not just the analysis above:** none of
the three is taken this sprint anyway. Reasoning: T10 as accumulated
already carries a new bounded context (Identity/Users), an extension to
an existing one (Social Play's `Match`), and three follow-up tickets —
37 points of real, coupled work (dependency order in Part B). Adding
three more independent workstreams on top would repeat the exact
"three-way overloaded slot" mistake `docs/process/t9-sprint-plan.md` §A5
named and warned the next ceremony about, this time turning it into a
five-way one. **Recorded as a scoping decision with the blocking analysis
made explicit, not as "these are blocked" — the distinction the task
asked this ceremony to check.** All three roll to T11, where pricing/
discount UI and the WCAG audit can start immediately (no dependency on
anything T10 ships) and Club rentals can build on T10.1's new `club` role.

## A3. The three untracked T9 follow-ups — opened as real issues this ceremony

Per `docs/process/t9-retro.md` finding 5, the PO's "second time of
asking": all three are now real GitHub issues, opened before any new
ticket text was written, not after.

- **#96** — Competitions-shaped `payments.PayableType` value + port/
  adapter pair (T9.6/T9.7 finding). Ticketed as **T10.6**.
- **#97** — Extend the write-handler malformed-ID boundary guard (PR #89's
  disclosed gap, still open after PR #94 closed only the two public-read
  instances). Ticketed as **T10.7**.
- **#98** — The host/venue display-name join gap both T9.6/T9.7 UI
  reviews flagged. Ticketed as **T10.8**.

## A4. What Identity/Users is, and is not, in T10

Mirrors `booking` exactly per `CLAUDE.md`: `internal/identity/{domain,app,
port,adapter}` + `proto/pickleball/identity/v1`.

**In:** a `User` aggregate — `ID`, `DisplayName` (required non-empty, so
T10.8 has something real to join against), `Roles` (`[]Role`, closed enum
`player | host_organiser | game_admin | facility_owner | club |
platform_admin`, per A1), `SelfReportedStartingLevel` (the raw value a
player sets at signup — a simple bounded value, e.g. a 1–5 self-assessment
scale; **not** a computed score). This is the one field the locked
`CLAUDE.md` decision names directly ("new players seeded by a
self-reported starting level") — building it requires no answer to Q1,
which is about the *derived, weighted* Level, a different thing.

**Out, explicitly, per ADR-0012:** `PlayerRating` (derived value or
update algorithm), any computed "Level," `Gender`, any matching-mode flag,
any matchmaking RPC or UI control implying matching happens. See ADR-0012
for the full piece-by-piece reasoning; T10.1/T10.2's own Instructions
restate the "explicitly not built" line so an implementer can't miss it.

## A5. What `Match` is, and is not, in T10

Per `docs/agent-operating-handbook.md` A1, `Match` and `PlayerRating`
belong to **Social Play**, not Identity — only the self-reported level is
Identity's. So `Match` is added to the existing `internal/socialplay`
context, connected to Identity via a new `port.IdentityLookup` (mirrors
T8.3's `port.FacilityLookup` pattern: the pure domain stays free of the
Identity context per `CLAUDE.md` rules 2/3, and the dependency arrow
points the direction the context map requires).

**In:** a `Match` value recording a real result — players, score,
`RecordedAt`, the `GameID` it belongs to. Requires the recording actor to
be the Game's Host or an assigned Game Admin (mirrors the existing
Host/Game-Admin authorization pattern, T5.1/T5.4/T6.3).

**Out, explicitly:** any field or computation deriving `PlayerRating` from
recorded matches. A `Match` is a fact; nothing yet reads it back into a
rating. This is deliberate, not an oversight — see ADR-0012.

## A6. Ticket-writing rule adopted this ceremony (T9 retro finding 4)

Every ticket's error-handling instructions below name a **gRPC code
only** — never an HTTP status alongside it, since the gateway's code→HTTP
mapping is the single authority and restating both is exactly how T9.4's
`FailedPrecondition`/409 contradiction shipped. Where a REST-specific
shape is genuinely part of the requirement, it's named *instead of* the
gRPC code, not next to it — none of T10's tickets need that exception.

## A7. Dispatch isolation (T9 retro finding 1, now a Ceremony 2 checklist item — not a per-dispatch judgment call)

Explicit for this sprint, decided at planning time:

- **Wave 1 — up to 4 implementers in parallel, each on its own isolated
  git worktree (`isolation: "worktree"`), never a shared unisolated
  checkout.** T10.1, T10.3, T10.6, T10.7 touch disjoint packages
  (`internal/identity` is new; `internal/socialplay/domain` for T10.3;
  `internal/payments`+`internal/competitions` for T10.6; five contexts'
  handler files for T10.7 — checked against T10.1/T10.3's file lists,
  no overlap).
- **Wave 2 — up to 2 implementers in parallel, worktree-isolated.**
  T10.2 (depends on T10.1), T10.4 (depends on T10.3). Different
  contexts, no shared files expected, but both touch `cmd/server`
  wiring (`main.go`) — **flag this explicitly**: whichever merges second
  should expect and resolve a `main.go` conflict on its own source
  branch, never on the shared branch (the T9.6/T9.7 `router/index.ts`
  precedent — sequence or resolve-on-source-branch, and this time the
  choice and its one-line reasoning goes in the first affected PR's body
  per T9 retro finding 7's adopted fix).
- **Wave 3 — up to 2 implementers in parallel, worktree-isolated.**
  T10.5 (depends on T10.2, benefits from T10.4), T10.8 (depends on
  T10.2). Both touch `web/src/router/index.ts` and possibly
  `GameDetailPanel.vue`/`CompetitionDetailPanel.vue` — **same flag as
  above, recorded in advance rather than rediscovered**: T10.8 touches
  display components T10.5 might also touch (profile-linked name
  rendering). Sequence T10.8 after T10.5 if both are dispatched the same
  day; if genuinely parallel, whichever merges second resolves on its
  own source branch.

## A8. Recorded disagreements (not manufactured consensus)

**PM vs. PE — does this sprint deliver enough user-visible value?**
- **PM:** T9 shipped two clickable flows end to end. T10 as scoped ships
  no new user-visible capability — a profile field nobody can act on yet,
  and a match-recording endpoint no UI surfaces meaningfully (T10.5 is a
  disclosure update, not a feature). That's a real cost, and it should be
  named, not absorbed silently into "the responsible engineering call."
- **PE:** the alternative is guessing at Q1/Q2, which this sprint's own
  instructions forbid and which ADR-0010's own reasoning says produces a
  more expensive mistake than shipping nothing new. The value this sprint
  produces is real but structural — a context this project has needed
  since T5 exists for the first time, without repeating the
  `games.facility_id` cost. That is worth a sprint even with no new
  screen a player would notice.
- **PO's tiebreak:** both are right and the tension doesn't need resolving
  to act — T10.5's honest, upgraded disclosure ("Identity now tracks your
  starting level; automated matching is pending two decisions we've asked
  the platform owner to make") is the one user-visible thing this sprint
  owes, and it ships. Recorded as unresolved on the larger question.

**QA vs. PdE — is 8 tickets across a new context, an existing-context
extension, and three hardening tickets too much for one sprint?**
- **QA:** T9's retro finding 8 flagged that "5-loop-cap has never bound"
  as an *untested* number, not a calibrated one — this sprint is a
  reasonable place to find out, given its size (37 points, comparable to
  T9's 49) and the genuine cross-context wiring involved (T10.2/T10.4
  both touch `cmd/server`).
- **PdE:** the four Wave-1 tickets are each small and self-contained
  (3–5 points, pure domain or pure boundary-guard work); the risk is
  concentrated in T10.2 and T10.4's `cmd/server` collision, already
  flagged in A7, not in overall ticket count.
- **Unresolved**, recorded rather than smoothed — both agree the A7
  mitigation (worktree isolation + the pre-flagged conflict point) is the
  right response regardless of who's right about the aggregate risk.

**BA note, not a disagreement:** the BA dossier's contradiction check was
run against T10.1's `SelfReportedStartingLevel` type choice (a bounded
1–5 self-assessment) against Q1's still-open "one field or two" question.
**Finding:** no contradiction — `SelfReportedStartingLevel` is explicitly
the *raw input*, not `Level` or `PlayerRating`, and ADR-0012 states this
distinction is why it's buildable pre-Q1. Confirmed the ticket text
(T10.1) states this explicitly so a future reader can't mistake the
1–5 scale for an answer to Q1.

---

# Part B — Ceremony 2 (sprint planning)

## Sprint goal

> Identity/Users exists as a real bounded context — a User with roles and
> a self-reported starting level — and Social Play can record real Match
> results against it, both wired end to end through Postgres/proto/gRPC
> with authorization baked in from the start; three untracked T9
> correctness/UX gaps (a missing Competitions payable type, an incomplete
> malformed-ID guard, and raw-ID display names) are closed; and
> `PlayerRating`, the matching algorithm, and gender-mix matching are
> named — precisely, in an ADR and in-product — as blocked on two
> decisions only the platform owner can make, not deferred a fifth time
> in prose.

## In-scope tickets

### T10.1 — Add the Identity/Users `User` domain model (pure domain, TDD)

**Story:** As the platform, I want a real `User` aggregate with roles and
a self-reported starting level, so that every other context that
currently treats `actor_user_id`/`host_id`/`player_id` as an opaque
string has a real concept to eventually reference, and so cold-start
matchmaking has a real input once it's built.

**Description:** First ticket in a brand-new bounded context — no prior
Identity code exists anywhere in this repo (confirmed by the same grep
ADR-0010 ran, re-run this ceremony with the same empty result). Mirrors
`internal/booking/domain`'s shape exactly per `CLAUDE.md`.

**Instructions:**
1. `internal/identity/domain.User`: `ID` (non-empty), `DisplayName`
   (required non-empty — T10.8 depends on this existing), `Roles`
   (`[]Role`, non-empty, closed enum `player | host_organiser |
   game_admin | facility_owner | club | platform_admin` per
   `docs/agent-operating-handbook.md` A1), `SelfReportedStartingLevel`
   (bounded value, e.g. `1..5`; reject out-of-range).
2. Constructor validation: empty `ID`/`DisplayName` rejected, empty
   `Roles` rejected, unrecognized `Role` value rejected, out-of-range
   level rejected — each with a distinct sentinel error.
3. **Explicitly not built, and the doc comment on `User` must say so
   directly, citing ADR-0012:** no `PlayerRating` field, no derived
   `Level`, no `Gender` field, no matching-mode flag. `SelfReportedStartingLevel`
   is the raw self-reported input the locked `CLAUDE.md` decision
   specifies for cold-start seeding — it is not an answer to Q1 (the
   tenure+win-rate weighting question), and the doc comment must state
   that distinction so a future reader doesn't conflate them.
4. Non-functional: framework-free (`CLAUDE.md` rule 2) — no pgx/grpc
   imports.

**Cross-context dependency check:** none — first entity in a new context,
calls nothing.

**Story points:** 5. **Role:** principal-engineer. **Labels:**
`sprint:t10`, `role:principal-engineer`, `type:story`, `points:5`.

---

### T10.2 — Wire Identity/Users to Postgres + proto + gRPC/REST

**Story:** As a client, I want to create a User and read one back by ID,
so that Identity/Users is a real, callable service other contexts and
the web app can depend on.

**Description:** Depends on T10.1. Mirrors T7.3/T8.2's Postgres+proto+gRPC
wiring pattern. Authorization and the malformed-ID boundary guard are
**baked into this ticket**, not a follow-up — per T9's adopted process
decision ("authorization is baked in, not a follow-up ticket," T9 kickoff
note) and per this sprint's own issue #97, which exists specifically
because that discipline wasn't applied retroactively to older write
paths. This ticket does not get to create that gap a second time.

**Instructions:**
1. `identity_users` table + migration; sqlc queries; `port.Repository`
   (`Create`, `GetByID`); Postgres adapter.
2. `identity.proto`: `CreateUser`, `GetUser`, `UpdateSelfReportedLevel`
   RPCs. `CreateUser`/`UpdateSelfReportedLevel` require an
   `actor_user_id` field (same claimed-actor pattern as every other
   write path in this codebase — same caveat, not re-litigated: object-
   level check given a claimed actor, not authentication).
3. `UpdateSelfReportedLevel` requires `actor_user_id == target User.ID`
   (`domain.User.EnsureSelf`, mirrors `EnsureOwner`/`EnsureHost`). A
   handler-level regression test proves a non-matching actor is rejected
   (T5.5/T7.7 pattern: disable the check, confirm the new test fails,
   restore it, confirm green — CLAUDE.md rule 10).
4. **Malformed-ID boundary guard, on `GetUser`, from day one:** reuse the
   existing UUID-shape validator from PR #89's Layer 2 (do not re-derive
   it — the postmortem in `docs/LESSONS.md` records why a validator wider
   than what the adapter accepts is not a validator). A malformed ID
   returns the identical `NotFound` an unknown-but-well-formed ID would.
5. Error handling — **gRPC codes only:**
   - Unknown/malformed `GetUser` ID → `NotFound`.
   - Invalid role/level/empty `DisplayName` on `CreateUser` →
     `InvalidArgument`.
   - `EnsureSelf` mismatch on `UpdateSelfReportedLevel` →
     `PermissionDenied`.

**Cross-context dependency check:** none new — this ticket is the
producer other contexts (T10.5, T10.8) will consume via a port later,
not a consumer itself.

**Story points:** 8. **Role:** principal-engineer. **Labels:**
`sprint:t10`, `role:principal-engineer`, `type:story`, `points:8`.

---

### T10.3 — Social Play: Add `Match` domain model (pure domain, TDD)

**Story:** As a Host or Game Admin, I want to record a Match's result
against a Game, so that Social Play has real match history — the input
any future rating/matching algorithm will need — without inventing a
rating this sprint.

**Description:** Extends `internal/socialplay/domain`, not a new context
— `Match` belongs to Social Play per `docs/agent-operating-handbook.md`
A1. No dependency on T10.1/T10.2; can run in parallel.

**Instructions:**
1. `domain.Match`: `ID`, `GameID` (references an existing `Game`),
   `Players` (`[]PlayerID`, at least 2), `Score` (a simple value — e.g.
   per-player point totals or a winner/loser pair; PE's call, keep it
   minimal since nothing downstream consumes it yet), `RecordedAt`.
2. Constructor validation: empty `GameID`/`Players` rejected, fewer than
   2 players rejected.
3. **Explicitly not built, doc comment must say so, citing ADR-0012:** no
   `PlayerRating` field or computation anywhere in this ticket. `Match`
   is a recorded fact; nothing reads it back into a rating yet.

**Cross-context dependency check:** calls into Social Play's own existing
`Game` (same context, not cross-context) — confirmed `domain.Game`
already has a stable `ID` to reference. No call to Identity/Users in this
ticket (rating computation, which would need it, is out of scope here).

**Story points:** 3. **Role:** principal-engineer. **Labels:**
`sprint:t10`, `role:principal-engineer`, `type:story`, `points:3`.

---

### T10.4 — Social Play: Wire `Match` to Postgres + proto + gRPC/REST

**Story:** As a Host or Game Admin, I want to submit a Match result
through the real API, so that match history is durable and queryable.

**Description:** Depends on T10.3. Authorization baked in from the start
(same T9 process rule as T10.2).

**Instructions:**
1. `matches` table + migration; sqlc queries; `port.MatchRepository`;
   Postgres adapter.
2. `socialplay.proto` gains `RecordMatchResult`, `ListMatchesForGame`
   RPCs.
3. `RecordMatchResult` requires the caller to be the Game's Host or an
   assigned Game Admin (`domain.Game.EnsureHostOrGameAdmin`-shaped check,
   mirroring the existing pattern payments/T6.3 established for offline
   recording). Handler-level regression test proving a non-Host/non-
   Game-Admin actor is rejected, verified non-vacuously (CLAUDE.md rule
   10, same discipline as T10.2's item 3).
4. Malformed-ID boundary guard on both RPCs from day one, same as T10.2
   item 4 (this ticket does not get to be a second instance of issue #97).
5. Error handling — **gRPC codes only:**
   - Unknown `GameID` on `RecordMatchResult`/`ListMatchesForGame` →
     `NotFound`.
   - Non-Host/non-Game-Admin actor → `PermissionDenied`.
   - Fewer than 2 players / empty score → `InvalidArgument`.
   - Recording a match against a cancelled Game → `FailedPrecondition`.

**Cross-context dependency check:** none new — confirmed Social Play's
existing Host/Game-Admin concept is sufficient for this ticket's
authorization; no call to Identity/Users or Payments.

**Story points:** 5. **Role:** principal-engineer. **Labels:**
`sprint:t10`, `role:principal-engineer`, `type:story`, `points:5`.

---

### T10.5 — Vue: Profile screen (self-reported level) + upgraded "matching not available" disclosures

**Story:** As a Player, I want to see and set my self-reported starting
level on a real profile, and understand precisely why automated matching
isn't available yet, so that the product tells me the truth about both
what exists and what's still pending.

**Description:** Depends on T10.2 (`GetUser`/`UpdateSelfReportedLevel`);
benefits from T10.4 (a Match-history read, if capacity allows — not
required for AC). Updates the existing T8.8/T8.9 "matching isn't
available yet" notes with the more precise, upgraded explanation ADR-0012
requires, rather than leaving the old generic copy in place now that it's
stale (Identity/Users existing changes what's true to say).

**Instructions:**
1. A minimal profile view/edit for `DisplayName` (read-only in this
   ticket — no rename flow scoped) and `SelfReportedStartingLevel`
   (editable, calls `UpdateSelfReportedLevel`).
2. Update every existing "matching isn't available yet" note (T8.8's
   Social Game creation flow, T8.9's Discover/Join flow, T9.6/T9.7's
   Competition screens) to the ADR-0012-precise version: name that
   Identity/Users now exists, and that the two things still pending are
   specific, named, escalated product decisions — not "coming soon."
   Copy must not imply a timeline the team doesn't control.
3. **Explicitly not built:** no matching UI control of any kind (no
   level-range slider, no gender-mix selector) — same absence-assertion
   discipline T8.8/T8.9/T9.6/T9.7 already established (grep the whole
   diff for `connect|match|gender|level range` per those PRs' own
   verification method; assert absence in tests, not just by omission).
4. No fabricated fields — if `SelfReportedStartingLevel` is unset,
   render an honest empty state, not a default value implying the
   player chose one.

**Cross-context dependency check:** calls `identity` (T10.2, `GetUser`/
`UpdateSelfReportedLevel` — confirmed to exist by dependency order below)
and, optionally, `socialplay` (T10.4, `ListMatchesForGame`, read-only,
not required for AC).

**Story points:** 5. **Role:** ux-ui-designer. **Labels:** `sprint:t10`,
`role:ux-ui-designer`, `type:story`, `points:5`.

---

### T10.6 — Add a Competitions-shaped `payments.PayableType` + port/adapter (closes #96)

**Story:** As a Player entering a Competition, I want to pay online, so
that I'm not limited to cash — and as the platform, I want that payment
routed to the right context's table, not Social Play's by accident.

**Description:** Closes issue #96 (T9.6/T9.7 finding). Mirrors T6.5's
Social Play precedent exactly. No dependency on T10.1–T10.5.

**Instructions:**
1. Add `PAYABLE_TYPE_COMPETITION_ENTRY` to `payments.PayableType` (proto
   + `internal/payments/domain.PayableType`), in the same PR that first
   produces a caller for it (per the type's own extension policy).
2. Add a `CompetitionEntryPaymentUpdater` port on the Competitions side
   (mirrors `internal/socialplay/port.RegistrationPaymentUpdater`);
   implement it in a new `internal/payments/adapter/competitions` adapter
   (mirrors `internal/payments/adapter/socialplay`).
3. Wire `reconcileCompetitionEntryPaymentStatus`, called only for
   `PAYABLE_TYPE_COMPETITION_ENTRY`, alongside the existing
   `reconcileRegistrationPaymentStatus`.
4. Vue: replace `CompetitionEntryPanel.vue`'s "online payments aren't
   available yet" disclosure with the real checkout hop, reusing
   `GameCheckout.vue`'s pattern per T8.10 (one checkout component, not a
   fork — same reuse discipline T9.6's `UnpaidCashAmount.vue` extraction
   already established).
5. Error handling — **gRPC codes only:** unknown/invalid `PayableType`
   on the new path → `InvalidArgument`; authorization mismatch (actor is
   neither the entrant nor an assigned Game/Competition Admin) →
   `PermissionDenied`. Authorization baked into this ticket per the T9
   process rule.

**Cross-context dependency check:** calls into `internal/payments`
(new enum member, confirmed absent by inspection this ceremony — not
assumed) and `internal/competitions` (new port implementation).
Confirmed `competitions.domain.Entry`'s `ID` type is a plain string,
matching what `PayableID` expects — checked before ticketing, since
T6.5's own build hit exactly this class of mismatch.

**Story points:** 5. **Role:** principal-engineer. **Labels:**
`sprint:t10`, `role:principal-engineer`, `type:story`, `points:5`.

---

### T10.7 — Extend the malformed-ID boundary guard to the remaining write handlers (closes #97)

**Story:** As the platform, I want every public write handler — not just
the read handlers PR #89/#94 already covered — to answer a malformed ID
the same way an unknown-but-well-formed one would, so that an
unauthenticated caller can't drive attacker-controlled log volume via
repeated panics even though the process no longer crashes.

**Description:** Closes issue #97. No dependency on any other T10 ticket.
Pure hardening pass across five contexts' existing write handlers.

**Instructions:**
1. Add the Layer-2-equivalent UUID shape guard (reuse the existing
   helper from PR #89 — do not re-derive it) to: `CancelCompetition`,
   `EnterCompetition`, `AddCourt`, `RecordOfflinePayment`,
   `CreateOnlinePayment`, `ConfirmOnlinePayment`, and any other public
   write handler taking a caller-supplied ID found by inspection (don't
   assume this list is exhaustive without re-checking against current
   `main.go` routing).
2. Each handler's malformed-ID answer must match its *existing*
   not-found/precondition semantics — **gRPC codes only:** the same
   `NotFound` (or context-appropriate equivalent, e.g.
   `FailedPrecondition` where that's what an unknown-but-valid ID
   already gets) an unknown-but-well-formed ID already returns from that
   same handler. No new status shape invented per handler.
3. Regression tests must avoid the fake-fidelity trap named in
   `docs/process/t9-retro.md` finding 3: assert the malformed ID **never
   reaches the repository call** (a property a fake can't satisfy
   identically whether the guard exists or not), not a return-value
   assertion alone. State in the PR which verification shape was used.
4. Verify non-vacuously (CLAUDE.md rule 10): disable the guard, confirm
   the new tests fail, restore it, confirm green.

**Cross-context dependency check:** touches all five existing contexts'
handler files — horizontal, not vertical. No enum/port gap involved;
pure boundary validation reusing an existing helper.

**Story points:** 3. **Role:** principal-engineer. **Labels:**
`sprint:t10`, `role:principal-engineer`, `type:bug`, `points:3`.

---

### T10.8 — Resolve the host/venue display-name join gap (closes #98)

**Story:** As a Player who followed a shared Competition link, I want to
see a real host name and venue name instead of a raw ID, so that the
screen means something to a stranger with no other context.

**Description:** Closes issue #98. Two independently-blocked halves —
Venue name is buildable now; Host/player display name depends on T10.2's
`DisplayName` field existing.

**Instructions:**
1. **Venue name (no Identity dependency):** join `venue_facility_id` to
   `facilities.GetFacility`'s existing `Name` field in
   `GameDetailPanel.vue`, `CompetitionDetailPanel.vue`, and
   `CompetitionManage.vue`. PE's call whether this is a client-side
   composition against the existing read or a small server-side
   denormalization — either is acceptable, state which and why in the PR.
2. **Host/player display name (depends on T10.2):** join `hostId`/
   `playerId` to `identity.GetUser`'s `DisplayName` in the same three
   components plus `CompetitionManage.vue`'s roster (`Player {{
   entry.playerId }}` → the resolved name).
3. Apply consistently across both Games and Competitions — this is a
   shared gap across both, not a Competitions-only fix (both T9.6's and
   T9.7's reviews named it independently).
4. No fabricated data: a User with no set `DisplayName` (shouldn't occur
   given T10.1 requires it non-empty at construction, but handle a
   lookup failure honestly) degrades to a named empty state, never a
   placeholder implying real data.

**Cross-context dependency check:** calls `internal/facilities`
(existing `GetFacility`, no new member) and `internal/identity` (T10.2's
`GetUser`, requires `DisplayName` — confirmed present by dependency
order below, not assumed).

**Story points:** 3. **Role:** ux-ui-designer. **Labels:** `sprint:t10`,
`role:ux-ui-designer`, `type:story`, `points:3`.

---

## Dependency order (see A7 for dispatch-isolation waves)

```
Wave 1 (parallel, ≤4 implementers, worktree-isolated):
  T10.1 (Identity domain) ─┐
  T10.3 (Match domain)     ─┼─→ Wave 2
  T10.6 (Competitions PayableType) ─┘   (independent, ships whenever)
  T10.7 (malformed-ID guard)             (independent, ships whenever)

Wave 2 (parallel, ≤2 implementers, worktree-isolated):
  T10.2 (Identity wiring, needs T10.1)
  T10.4 (Match wiring, needs T10.3)
  ⚠ both touch cmd/server/main.go — second to merge resolves on its
    own source branch, one-line reasoning in that PR's body (T9 retro
    finding 7's adopted fix)

Wave 3 (parallel, ≤2 implementers, worktree-isolated):
  T10.5 (profile + disclosures, needs T10.2)
  T10.8 (display-name join, needs T10.2)
  ⚠ both may touch GameDetailPanel.vue/CompetitionDetailPanel.vue —
    sequence T10.8 after T10.5 if dispatched same-day; otherwise
    resolve on source branch per the same rule
```

Total: **37 points** across 8 tickets (comparable to T9's 49 across 10 —
smaller because two brand-new-context tickets, T10.1/T10.2, carry more
irreducible sequencing risk than raw point count suggests, per the QA/PdE
disagreement in A8).

## Definition of Done (sprint-level)

All 8 tickets merged per `sprint-process.md`'s per-ticket DoD (PR-only,
CLAUDE.md rule 9 — no exception for this plan, the ADR, or the issues
above, all of which land the same way); sprint goal met (Identity/Users +
Match real and wired, three follow-ups closed, blocked pieces named in
ADR-0012 rather than silently dropped); retro held
(`docs/process/t10-retro.md`) with findings indexed from
`docs/LESSONS.md`'s `## T10 sprint retro`; `HANDOFF.md` updated for T11 to
resume from, including the A2 re-scope decision (pricing/discount UI,
Club rentals, WCAG audit — none blocked, all deliberately sequenced to
T11) and ADR-0012's trigger (fires on the user answering Q1/Q2).
