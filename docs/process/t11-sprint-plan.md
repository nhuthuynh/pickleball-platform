# T11 Sprint Plan — Pricing/discount UI, Club rentals, WCAG audit

Ceremonies 1 and 2 per `docs/process/sprint-process.md`, six-role team
(briefs: `docs/agent-operating-handbook.md` Part B), against `HANDOFF.md`'s
T10 entry and Cross-cutting section, `docs/process/t10-sprint-plan.md`'s
A2 roadmap-carryover decision, and `docs/process/t10-retro.md`'s six
findings (five adopted, threaded into this plan below — not just cited).

---

# Part A — Ceremony 1 (backlog refinement)

**Product Manager + Principal Engineer**, per `sprint-process.md`. PM drove
scope/value framing; PE drove technical sequencing/feasibility. Both sign
off on every ticket below before it was opened.

## A0. What this ceremony inherits

T10's own Ceremony 1 (§A2) already did the "is this actually gated on real
auth" analysis for the three roadmapped items T7's plan first named as a
future "T9" and T8/T9's plans rolled forward to T10, then to T11:
pricing/discount UI (Flow 2), Club rentals (Flow 7), and a WCAG 2.2 AA
audit. Its conclusion: **none of the three is gated on real auth** — each
needs, at most, the same claimed-`actor_user_id` object-level check every
other write path in this codebase already has (T5.5/T6.7→T8.5/T7.7
precedent), the same way `AddCourt` never needed JWT/Auth0 to exist first.
T10 deliberately took none of them anyway, reasoning that T10's own 37
points (a new context, an existing-context extension, three follow-ups)
was already comparable to T9's 49, and adding three more independent
workstreams would repeat the "overloaded slot" mistake T9's own retro
warned the next ceremony about.

## A1. Re-verifying the "gated on real auth" analysis still holds

Per this sprint's own instructions: checked, not assumed. T10 shipped
Identity/Users (`User.Roles` now includes `club`) and `Match`; neither
changes the analysis, because neither pricing/discount rules nor the
recurring-hire booking mechanics ever depended on Identity/Users existing
— T10.1's A2 explicitly named `club` as the one piece Club rentals was
*waiting on*, and it now exists. So the analysis doesn't just "still
hold," one of its own stated preconditions is now satisfied:

- **Pricing/discount UI (Flow 2).** Unchanged conclusion: independently
  buildable. `discount_rules` (`docs/design/v1-system-design.md` §3.2) is
  a new concept layered on top of the existing `pricing_rules` resolution
  (`ResolvePrice`, T1) — it needs an owner-scoped create path, not
  authentication.
- **Club rentals (Flow 7).** The one piece that *was* genuinely coupled —
  "Club" needing a real home before the account type gets built, so as
  not to repeat the `games.facility_id` mistake — is now resolved: T10.1
  added `club` to `User.Roles` for exactly this reason. **No longer even
  partially blocked.**
- **WCAG 2.2 AA audit.** Unchanged: zero auth dependency, pure
  accessibility work across existing shipped screens.

**PM/PE's scoping decision:** all three are taken this sprint. T10's own
plan named this as the expected outcome ("all three roll to T11, where
pricing/discount UI and the WCAG audit can start immediately... and Club
rentals can build on T10.1's new `club` role") — deferring a second time
without a new blocking fact would repeat exactly the "prose deferral"
pattern this project's retros keep calling out (T9 retro finding 5, ADR-
0010's own "not a fourth roll-forward" language). Total scope, priced
below: **47 points across 9 tickets** — larger than T10's 37, comparable
in shape to T9's 49 (three real workstreams plus small, well-specified
hardening/chore tickets, not one giant one).

## A2. Threading T10 retro finding 1 into ticket-writing guidance (staged-diff verification)

**Adopted, stated here so every T11 implementer sees it, not just cited:**
any commit that follows a manual edit made during merge-conflict
resolution must be checked with `git diff --cached` (or `git show
--stat`/`git log -p -1` on the resulting commit) **before pushing** —
verifying the working tree alone is not sufficient evidence of what will
actually ship. This is not a suggestion to remember; it is now standing
ticket-writing guidance, restated in every T11 ticket whose Instructions
involve a merge (T11.2/T11.5, T11.3/T11.6 — see A7's dispatch waves) via
a one-line reminder in that ticket's own text, the same way T10's plan
threaded T9 retro findings into its own tickets rather than leaving them
as prose above the ticket list. (T10 retro finding 1's harder question —
whether a full three-way staged-diff/local-suite/fresh-worktree
verification becomes *mandatory* on every conflict-touching PR, not just
the cheap `git diff --cached` floor — was left as a recorded PE/QA
disagreement and stays unresolved; this ceremony does not have standing
to resolve a disagreement the retro itself left open, so it is restated
as unresolved below in A8, not silently picked one way.)

## A3. Threading T10 retro finding 2 into a real ticket (fixture-infidelity, out-of-scope-of-`make test-domain`)

**Adopted, and given a ticket, not left as a disclosure norm.** T10 hit
this bug class twice in one sprint (four times project-wide, across T9 and
T10). The retro's own recommendation was two structural changes: (1) a
shared, UUID-shaped fixture-ID helper per context — PR #107 already did
this for `payments`' two `grpcapi` test files; **T11.9 generalizes that
pattern to every context**, not just the one PR #107 happened to touch —
and (2) any disclosed-but-out-of-scope regression must produce a tracked
follow-up (issue or `HANDOFF.md` bullet) **at the moment it's found**, not
left for a future reviewer to rediscover by chance. (2) is not a ticket by
itself — it's a standing rule, restated in T11.9's own Instructions and in
the general ticket-writing guidance below (A6), since it applies to every
ticket in this sprint and every sprint after it, not just T11.9's own
diff.

## A4. Threading T10 retro finding 3 into T11's creation-RPC tickets

**Adopted, applied concretely.** Two tickets in this sprint add a real
creation RPC where the caller-supplied value could become or gate a
resource's own permanent state (not a mutation on an object that already
exists) — the exact shape T10.2's `CreateUser` review found unprepared
for: **T11.2** (`CreateDiscountRule`) and **T11.5**
(`RequestRecurringHire`). Both tickets' Instructions apply the retro's
adopted checklist explicitly:
1. Can an anonymous/unauthenticated caller choose the resource's own
   permanent identifier, and permanently occupy one a legitimate future
   caller will need? — **Checked for both: no.** Neither RPC accepts a
   caller-supplied ID; both mint the ID server-side (`ids.NewID()`,
   mirroring every ID-generating create path in this codebase since T1).
   No `CreateUser`-shaped squatting is possible because there is nothing
   to squat.
2. Can an anonymous caller self-assign any field that gates authorization
   or privilege elsewhere in the system, given there's no pre-existing
   owner fact to check the request against? — **Checked for both, and
   this is the one that actually fires for T11.5**: `RequestRecurringHire`
   takes only `actor_user_id`, never an `is_club`/`role` field on the
   request DTO — whether the actor may request a recurring hire as a Club
   is resolved server-side via a real `identity.GetUser` lookup checking
   `Roles` contains `club`, never a self-declared claim. `CreateDiscountRule`
   has no analogous field (`DiscountType`/`AppliesTo` describe the
   discount, not the actor) — checked, not assumed, and stated as a
   negative finding in T11.2's own ticket text rather than silently
   skipped.

## A5. Threading T10 retro finding 4 into the migration-namespace check (extending A7's dispatch-isolation table)

**Adopted, and applied before any collision can happen this time** — this
is the second occurrence of the identical collision shape
(`0005_payments.sql`/`0005_socialplay.sql` at T6, `0015` claimed by both
T10.2 and T10.4), and T10's retro named it a checklist gap, not just an
incident. Current tip: `db/migrations/0016_identity.sql` (confirmed by
listing the directory this ceremony, not assumed from the last plan's
count). Two T11 tickets add a migration file: **T11.2 is pre-assigned
`0017`, T11.5 is pre-assigned `0018`** — recorded here, in the plan both
implementers will read, not left for either session to discover the other
claimed the same number. (PdE/QA's T10 retro disagreement on whether to
generalize this check to proto field numbers too is unresolved — see A8;
this ceremony applies the adopted part, migration numbers, and does not
independently decide the contested generalization.)

## A6. Threading T10 retro finding 5 into T11's absence-assertion ticket

**Adopted, applied concretely, and also stated as checked-but-not-firing
where it doesn't apply.** T11.6 (Club rental UI) has a real "must not
show control X to actor Y" requirement — the rental-request control must
be absent for a non-Club actor — so its AC requires the
`findGenderControls()`-shaped semantic-signal check (`aria-label`,
`<label>` association, ARIA `role`, not just native `id`/`name`), reusing
the existing helper's shape rather than a screen-local narrower
reimplementation. **Checked and does not fire for T11.3** (pricing UI has
no "must be absent for some actor" requirement — a Player simply sees
whatever discounted price resolves, there's no control being hidden) or
**T11.7** (the WCAG audit's job is finding present-but-broken controls,
not proving a control's absence) — stated explicitly per this finding's
own "checked, not assumed" discipline rather than left for a reader to
wonder why only one ticket cites it.

## A7. Ticket-writing rule carried forward unchanged (gRPC codes only)

Continuing T10's A6 rule: every ticket below names a **gRPC code only** in
its error-handling instructions, never an HTTP status alongside it — the
gateway's code→HTTP mapping is the single authority. No T11 ticket needs
a REST-specific-shape exception.

## A8. Cross-context dependency check (T9 retro finding 5 lineage, continued from T10's A1)

| Ticket | Calls into | Member/port exists? |
|---|---|---|
| T11.1 (DiscountRule domain) | none — extends `internal/booking/domain`, same context as `pricing_rules` | n/a |
| T11.2 (DiscountRule wiring) | `internal/facilities` (new `port.FacilityLookup` on the Booking side — **first time Booking calls into Facilities at all**, confirmed by inspection this ceremony: `domain.NewBooking`'s `CourtID` is today an unvalidated opaque string, no existing port) | Facilities: `GetFacility`'s `OwnerID` already exists (T7.2/T7.3) — the port is new, the data it reads is not |
| T11.3 (pricing/discount UI) | `booking` (T11.2's `CreateDiscountRule`/quote-with-discount, read+write) | Exists by dependency order below |
| T11.4 (RecurringHireTemplate domain) | none — extends `internal/booking/domain`, references `domain.Source.SourceRecurringHire` (already exists since T0, unused until now — confirmed by grep this ceremony) | n/a |
| T11.5 (RecurringHireTemplate wiring) | `internal/facilities` (reuses T11.2's new `port.FacilityLookup`, does not re-derive it), `internal/identity` (new `internal/booking/port.IdentityLookup`, mirrors T10.4's `socialplay/port.IdentityLookup` pattern exactly — **first time Booking calls into Identity**) | Facilities: yes, via T11.2. Identity: `GetUser`/`Roles` exist since T10.2 — confirmed `club` is a real enum value, not assumed |
| T11.6 (Club rental UI) | `booking` (T11.5's request/approve RPCs, read+write) | Exists by dependency order below |
| T11.7 (WCAG audit) | no new backend calls — audits existing shipped Vue screens | n/a |
| T11.8 (retroactive issue-closing chore) | GitHub API only (via `issue_write`), plus a `sprint-process.md` doc edit — no application code | n/a |
| T11.9 (fixture-ID helper generalization) | touches existing `adapter/grpcapi`/`adapter/postgres` test files across all five existing contexts (booking, socialplay, payments, facilities, competitions) plus `identity` — no new port/member, pure test-infra refactor | n/a |

No gap found that mirrors the T9.6/T9.7 `PayableType` surprise or T10's
`CreateUser` surprise: the two genuinely new cross-context calls this
sprint introduces (Booking → Facilities, Booking → Identity) are both
named and checked here, at planning time, rather than discovered by an
implementer mid-ticket.

## A9. Two PE rulings on where new code lives (stated once, not re-litigated per ticket)

**Ruling 1 — `DiscountRule` stays inside `internal/booking/domain`,
alongside `pricing_rules`, not a new `internal/pricing` context.**
`HANDOFF.md`'s own Cross-cutting note on `GetQuote` says Pricing's
extraction trigger is "if/when Pricing grows real CRUD and its own
lifecycle" — `CreateDiscountRule` is arguably that trigger firing. **PE's
call, PM concurring:** extracting Pricing into its own bounded context now
would mean moving `pricing_rules`' sqlc queries, `ResolvePrice`, and
`GetQuote`'s call sites out of a stable, well-tested area of the codebase
for no AC this sprint actually needs — a real but separable refactor with
its own one-way-door cost (proto namespace, migration ownership), not a
prerequisite for shipping a discount on top of the existing resolution.
Flagged here as a live candidate for a future sprint, not silently
dropped; not done this sprint because `CreateDiscountRule`'s own AC does
not require it.

**Ruling 2 — `RecurringHireTemplate` also stays inside
`internal/booking/domain`, not a new context and not folded into
Identity/Users.** It directly manipulates and generates `Booking`
aggregates (its entire purpose is producing `recurring_hire`-source
Bookings) and has no other consumer — the same reasoning that kept
Social Play's `Match` inside `internal/socialplay` rather than spinning up
a sixth context for one addition (T10 A5). `Club` the *role* belongs to
Identity (T10.1, already shipped); `RecurringHireTemplate` the *request/
approval workflow* belongs to Booking, the same way `Registration`
belongs to Social Play even though a Player's identity doesn't.

**Scope narrowing, stated explicitly (BA discipline, not silent):**
`DiscountRule` in T11 is `FacilityID`-scoped only, not the design doc's
full "`CourtID` or `FacilityID`" either/or. PE's call: a facility-wide
discount covers the real Flow-2 use case (a Host running a seasonal
promotion) without the extra branching a court-level option would add to
both the domain constructor and the UI; court-level granularity is a
cheap, additive follow-up if a real product need for it shows up, not a
capability being foreclosed.

## A10. Dispatch isolation (T9 retro finding 1's discipline, T10 retro finding 4's namespace check applied)

- **Wave 1 — up to 5 implementers in parallel, each on its own isolated
  git worktree (`isolation: "worktree"`).** T11.1, T11.4 (both new files
  in `internal/booking/domain`, disjoint filenames — `discount_rule.go`
  vs. `recurring_hire_template.go` — checked against each other's
  expected file list, no overlap expected but flagged as the same package
  as a minor watch item), T11.7 (`web/` only), T11.8 (GitHub API +
  `sprint-process.md` only), T11.9 (existing `adapter/**` test files
  across contexts — none of which T11.1/T11.4/T11.7/T11.8 touch).
- **Wave 2 — 1 implementer.** T11.2 (needs T11.1; also the ticket that
  builds `internal/booking/port.FacilityLookup` + `internal/booking/
  adapter/facilities` — the shared infrastructure T11.5 depends on, so
  T11.5 cannot start until this lands, not just until T11.4 does).
  Migration `0017` pre-assigned (A5). Reminder per A2: if this ticket's
  own PR includes a merge-conflict resolution, verify with `git diff
  --cached` before pushing, not just a working-tree test run.
- **Wave 3 — up to 2 implementers in parallel, worktree-isolated.** T11.5
  (needs T11.4 for the domain type, and T11.2 for the `FacilityLookup`
  port it reuses rather than re-deriving — **do not let this ticket build
  a second `FacilityLookup` port**, the exact "two lineages independently
  invent the same thing" shape T6.5's dual-`0005` migration and T10's
  dual-`0015` claim both already are). Migration `0018` pre-assigned.
  T11.3 (needs T11.2 only, not T11.5 — pricing UI doesn't depend on
  Club rentals). T11.3 is a Vue ticket, T11.5 is Go backend — no file
  overlap expected, safe to run genuinely in parallel. Same A2 reminder
  applies to T11.5 if its merge includes a hand-resolved conflict against
  T11.2's shared port.
- **Wave 4 — 1 implementer.** T11.6 (needs T11.5). Likely touches
  `web/src/router/index.ts` and may reuse `CourtBookingFlow.vue`/checkout
  components T11.3 also touches — **flagged in advance**: if T11.3 and
  T11.6 end up dispatched close together despite the wave gap (e.g. T11.3
  finishes late and both land the same day), whichever merges second
  resolves the shared-file conflict on its own source branch with a
  one-line reason in that PR's body, same rule T9.6/T9.7 and T10.5/T10.8
  established.

## A11. Recorded disagreements (not manufactured consensus)

**PM vs. PdE — is 47 points across 9 tickets, with a 4-deep wave chain in
Booking alone (T11.1→T11.2→T11.5→T11.6), too much sequencing risk for one
sprint?**
- **PM:** three real, independently-valuable workstreams ship this
  sprint (a Host can finally run a discount, a Club can finally request a
  recurring slot, and every shipped screen gets an accessibility pass) —
  more user-visible value than T10 shipped, which is worth the extra
  points given T10's own retro's PM/PE disagreement (A8 there) named "no
  new user-visible capability" as a real, named cost.
- **PdE:** the 4-deep wave chain (T11.1→T11.2→T11.5→T11.6) is real serial
  risk this sprint's own dependency structure creates, not an artifact of
  over-caution — T11.2 building a shared port T11.5 must wait for is a
  genuine bottleneck, unlike T10's wider, shallower 3-wave shape where
  Wave 1 had four independent tickets. A cheaper path exists: ship
  Club rentals' domain+wiring (T11.4/T11.5) without T11.2's discount
  dependency by having T11.5 build its own minimal `FacilityLookup`
  read-only helper instead of waiting on T11.2's — but that reintroduces
  exactly the "two lineages invent the same port" duplication A10
  explicitly warns against, so PdE does not recommend taking the cheaper
  path, only names that it exists.
- **Unresolved**, recorded rather than smoothed. Both agree the wave
  structure in A10 is correct given the shared-port constraint; the
  disagreement is about whether that constraint was worth accepting
  versus a design that avoided creating it (e.g., two independent,
  duplicated `FacilityLookup` reads) in the first place.

**QA vs. PE — should T11.9 (fixture-ID helper) be scoped to a shared
constant block per context (mechanical, low-risk, matches PR #107's
precedent exactly) or a stronger structural guard (e.g. a lint rule or a
test-helper that panics on a non-UUID-shaped literal at test-run time)?**
- **QA:** T10 retro finding 2 counted this bug class at four occurrences
  across three sprints with no structural fix landing — a shared constant
  block is still just convention, not enforcement; nothing stops a sixth
  occurrence in a brand-new test file that doesn't happen to reuse the
  constant.
- **PE:** a runtime/lint guard is real, unscoped extra work (deciding
  where it lives, whether it's a `go vet` check or a test-helper
  assertion, whether it false-positives on legitimately non-UUID test
  data) that this ticket's own point budget (3) doesn't carry, and this
  project's own convention (cited in PR #108's review, per T10 retro
  finding 4) argues against building for a problem beyond what's been
  demonstrated. Fix the four demonstrated instances' pattern now; revisit
  enforcement if a fifth instance appears in a file the shared constant
  block doesn't already cover.
- **Unresolved**, recorded per this project's standing convention. T11.9
  as ticketed below takes PE's narrower scope; QA's stronger version is
  named as a candidate follow-up, not silently adopted or dismissed.

**BA note, not a disagreement.** Ran the same contradiction check T10's
BA note ran (T11.1's `DiscountRule` type choices against every locked
decision it touches): no contradiction found. `AppliesTo
(individual|recurring_hire|club)` correctly references `Source`'s already-
locked four values (`recurring_hire`/`individual`/`game`/`competition`,
D3b) without inventing a fifth; `FacilityID`-only scoping (A9) doesn't
contradict the design doc's "`CourtID` or `FacilityID`" language since
that was phrased as an either/or from the start, not a requirement for
both.

---

# Part B — Ceremony 2 (sprint planning)

## Sprint goal

> A Host can configure a real discount on top of an existing pricing
> band and a Player sees the discounted price in a live quote; a Club can
> request a recurring court slot and a Facility Owner can approve or
> reject it, generating real `recurring_hire`-source Bookings under the
> existing no-double-booking invariant; every screen this platform has
> shipped since T7 passes a real WCAG 2.2 AA audit against the criteria
> already named as load-bearing for this app's two flagship flows; and
> three T10-retro-adopted process fixes (staged-diff verification,
> per-context fixture-ID fidelity, retroactive issue closing) are threaded
> into how this sprint itself is planned and executed, not just recorded
> as findings to remember later.

## In-scope tickets

### T11.1 — Add the `DiscountRule` domain model (pure domain, TDD)

**Story:** As a Facility Owner, I want to define a discount on top of my
existing pricing, so that I can run a promotion (a percent-off launch
rate, a fixed-amount club rate, a time-boxed discount) without the
platform silently ignoring it or resolving it ambiguously against another
overlapping discount.

**Description:** Extends `internal/booking/domain`, alongside the
existing `pricing_rules`/`ResolvePrice` (T1). Per A9 Ruling 1, this does
**not** create a new `internal/pricing` context. No dependency on any
other T11 ticket — pure domain, can start immediately.

**Instructions:**
1. `domain.DiscountRule`: `ID`, `FacilityID` (non-empty — A9's scope
   narrowing: facility-wide only, not court-level, this sprint),
   `DiscountType` (`percent | fixed_amount`, closed enum), `Amount`
   (bounded: `percent` in `(0, 100]`, `fixed_amount` a positive `Money`,
   reusing `booking`'s existing `Money`-shaped value — do **not** import a
   cross-context `Money` type; T9's own ruling on this (A4 there) already
   forecloses that as shared-kernel coupling), `AppliesTo`
   (`[]domain.Source`, non-empty, must be a subset of the four already-
   locked `Source` values — reject an unrecognized value), `StartsAt`
   (`time.Time`), `EndCondition` (a small variant per
   `docs/design/v1-system-design.md` §3.2: `EndAfterDate(time.Time) |
   EndAfterOccurrences(int) | NoEnd` — this project's own precedent, not
   a new pattern).
2. `ResolveDiscount(rules []DiscountRule, facilityID string, source
   Source, at time.Time) (DiscountRule, error)`: mirrors
   `ResolvePrice`'s own shape. **Reuses ADR-0002's precedent explicitly**
   (design doc's own instruction): two rules matching the same
   `(FacilityID, Source, time)` triple is `domain.ErrAmbiguousDiscountRule`,
   a domain error — never silently resolved by priority/insertion order.
   No matching rule is not an error (a quote with no discount is a valid,
   common outcome) — return a "no discount" zero value, distinct from the
   ambiguous-match error.
3. Constructor validation: empty `FacilityID` rejected, invalid
   `DiscountType`/out-of-range `Amount` rejected, empty `AppliesTo`
   rejected, an `AppliesTo` value outside the four locked `Source` values
   rejected, `EndAfterOccurrences` with `n <= 0` rejected — each a
   distinct sentinel error.
4. Non-functional: framework-free (`CLAUDE.md` rule 2).

**Cross-context dependency check:** none (A8).

**Story points:** 3. **Role:** principal-engineer. **Labels:**
`sprint:t11`, `role:principal-engineer`, `type:story`, `points:3`.

---

### T11.2 — Wire `DiscountRule` to Postgres + proto + gRPC/REST, and apply it in `GetQuote`

**Story:** As a Facility Owner, I want to create a discount through the
real API and have it actually affect the price a Player is quoted, so
that the promotion I configure is real, not decorative.

**Description:** Depends on T11.1. Migration number pre-assigned:
**`0017_booking_discount_rules.sql`** (A5). This is also the ticket that
builds `internal/booking/port.FacilityLookup` + `internal/booking/
adapter/facilities` — the **first** call from Booking into Facilities
(confirmed absent by inspection this ceremony, A8) — which T11.5 will
reuse, not re-derive. Authorization is baked in from the start (T9's
adopted process rule, carried through T10 and restated here).

**Instructions:**
1. `discount_rules` table + `0017_booking_discount_rules.sql`; sqlc
   queries; `port.DiscountRuleRepository` (`Create`, `ListForFacility`);
   Postgres adapter. `AppliesTo` persists as a Postgres array/enum column
   (PE's call which, state the choice and why in the PR — mirrors how
   `pricing_rules.weekdays` already made this exact call, `HANDOFF.md`
   Cross-cutting notes it uses `time.Weekday` numbering directly).
2. `internal/booking/port.FacilityLookup` (mirrors `socialplay/
   port.FacilityLookup`/`competitions/port.FacilityLookup` exactly — same
   shape, new implementation, not shared code across contexts) +
   `internal/booking/adapter/facilities` implementing it against the real
   `facilitiesapp.Service.GetFacility`.
3. `booking.proto` gains `CreateDiscountRule`, `ListDiscountRulesForFacility`
   RPCs. `CreateDiscountRule` requires `actor_user_id`; the handler
   resolves the target `Facility` via `FacilityLookup` and calls
   `Facility.EnsureOwner(actor_user_id)` (T7.7 pattern) before
   constructing/persisting anything — no code path to the repository for
   a rejected actor, mirroring `AddCourt`'s own shape exactly.
4. `GetQuote` (existing RPC, T1) is extended: after resolving the price
   band via `ResolvePrice`, resolve a discount via `ResolveDiscount` for
   the same `(FacilityID, Source, time)` and apply it to the band price
   before returning. **Requires threading `FacilityID` through the quote
   path for the first time** (`GetQuote` currently keys off `CourtID`
   only, per `HANDOFF.md`'s note that `CourtID` is an unvalidated opaque
   string) — resolve `FacilityID` via the same `FacilityLookup` this
   ticket already built, do not add a second lookup mechanism.
5. **Creation-RPC checklist (T10 retro finding 3, A4 above) — both items
   checked, recorded in the PR:** (1) `ID` is server-generated
   (`ids.NewID()`), never caller-supplied — no squatting surface exists.
   (2) No field on `CreateDiscountRuleRequest` gates authorization or
   privilege on the actor themselves (`DiscountType`/`AppliesTo`/`Amount`
   describe the discount, not the caller) — checked and does not fire,
   stated as a negative finding, not silently omitted.
6. Error handling — **gRPC codes only:**
   - Non-owner actor on `CreateDiscountRule` → `PermissionDenied`.
   - Unknown/malformed `FacilityID` → `NotFound` (reuse the existing
     UUID-shape helper from PR #89's Layer 2, do not re-derive it).
   - Invalid `DiscountType`/`Amount`/`AppliesTo`/`EndCondition` →
     `InvalidArgument`.
   - Ambiguous discount match surfaced through `GetQuote` →
     `FailedPrecondition` (mirrors how `ErrAmbiguousPricingRule` is
     already mapped, per ADR-0002 — check the existing mapping before
     inventing a new one).
7. **Reminder (A2):** if this PR's merge includes a hand-resolved
   conflict, verify with `git diff --cached` before pushing.

**Cross-context dependency check:** calls into `internal/facilities`
(new port, existing data — A8). No call to Identity/Users in this ticket.

**Story points:** 8. **Role:** principal-engineer. **Labels:**
`sprint:t11`, `role:principal-engineer`, `type:story`, `points:8`.

---

### T11.3 — Vue: Pricing/discount UI (Flow 2)

**Story:** As a Facility Owner, I want to see and create discounts for my
facility, and as a Player, I want the quote I see to reflect any active
discount honestly, so that neither side of the flow is looking at a
number the backend isn't actually enforcing.

**Description:** Depends on T11.2. Builds the Flow-2 UI the design
review's own round-10 mockups scoped ("a discount rule with an end
condition") but that never had a backend to point at until now.

**Instructions:**
1. Facility Owner-facing: a discount create/list panel (reuses this
   project's existing form/panel component patterns — `FacilityOnboarding.vue`'s
   step shape, not a new one-off layout), calling `CreateDiscountRule`/
   `ListDiscountRulesForFacility`.
2. Player-facing: `CourtBookingFlow.vue`'s existing quote display now
   shows the discounted price when one applies, with the original band
   price struck through/labeled — never silently replacing one number
   with another with no indication a discount was applied (an honest-UI
   requirement, same discipline as T8.10's placeholder-fee labeling).
3. No fabricated fields: an `EndCondition` of `NoEnd` renders as
   "no end date," never a fabricated date. `EndAfterOccurrences` renders
   the remaining count if the read path returns one, otherwise an honest
   "ends after N total uses" label rather than inventing a live counter
   this ticket doesn't build.
4. Non-functional: this is new UI, so it is built to the WCAG 2.2 AA
   criteria T11.7 audits everything else against from the start (form
   labels, error identification/suggestion for invalid discount input —
   3.3.1/3.3.3) rather than being retrofitted by that audit later.

**Cross-context dependency check:** calls `booking` (T11.2's
`CreateDiscountRule`/`ListDiscountRulesForFacility`/extended `GetQuote` —
confirmed to exist by dependency order below).

**Story points:** 5. **Role:** ux-ui-designer. **Labels:** `sprint:t11`,
`role:ux-ui-designer`, `type:story`, `points:5`.

---

### T11.4 — Add the `RecurringHireTemplate` domain model (pure domain, TDD)

**Story:** As a Club, I want to request a recurring weekly slot at a
Facility, so that I don't have to book the same court one week at a time
— and as the platform, I want that request to generate real Bookings
under the same no-double-booking invariant every other Booking source
already obeys.

**Description:** Extends `internal/booking/domain`. Per A9 Ruling 2, not
a new context, not folded into Identity/Users. References the already-
locked `domain.SourceRecurringHire` value (exists since T0, unused until
this ticket — confirmed by grep this ceremony). No dependency on any
other T11 ticket — pure domain, can start immediately, in parallel with
T11.1.

**Instructions:**
1. `domain.RecurringHireTemplate`: `ID`, `RequestedByUserID`,
   `CourtID`, `Weekday` (reuses `time.Weekday`, same convention
   `pricing_rules.weekdays` already established — `HANDOFF.md`'s own
   Cross-cutting note already flags this as a logged, accepted
   convention, not something to silently change here), `StartTime`/
   `EndTime` (time-of-day, mirrors `Booking`'s own range shape),
   `StartsAt` (first occurrence date), `EndCondition` (**a
   Booking-context-local type, same shape as T11.1's — not shared code
   across the two domain packages**, per A9's shared-kernel note),
   `Status` (`requested | approved | rejected | cancelled`, closed enum).
2. `GenerateOccurrences(t RecurringHireTemplate, upTo time.Time)
   []TimeRange`: pure function computing the concrete date/time slots a
   template implies, bounded by `EndCondition` and `upTo` (a safety cap —
   `NoEnd` must never be asked to generate an unbounded slice). No I/O,
   no `Booking` construction here — that's T11.5's job once the template
   is approved.
3. Constructor validation: empty `RequestedByUserID`/`CourtID` rejected,
   `StartTime >= EndTime` rejected, `EndAfterOccurrences` with `n <= 0`
   rejected — each a distinct sentinel error. `Approve`/`Reject` state
   transitions: only `requested → approved`/`requested → rejected` legal;
   any other transition (e.g. re-approving an already-approved template)
   is `domain.ErrInvalidRecurringHireStatusTransition`.
4. Non-functional: framework-free (`CLAUDE.md` rule 2).

**Cross-context dependency check:** none — references `domain.Source`
(same package, not cross-context).

**Story points:** 5. **Role:** principal-engineer. **Labels:**
`sprint:t11`, `role:principal-engineer`, `type:story`, `points:5`.

---

### T11.5 — Wire `RecurringHireTemplate` to Postgres + proto + gRPC/REST: request, approve/reject, generate

**Story:** As a Club, I want to submit a recurring-hire request through
the real API, and as a Facility Owner, I want to approve or reject it and
have approval actually create the Bookings, so that a Club rental is a
real, durable commitment, not a conversation that happens outside the
platform.

**Description:** Depends on T11.4 (domain) **and** T11.2 (reuses its
`FacilityLookup` port — do not build a second one, A10). Migration number
pre-assigned: **`0018_booking_recurring_hire_templates.sql`** (A5). First
call from Booking into Identity/Users (new `internal/booking/
port.IdentityLookup`, mirrors T10.4's `socialplay/port.IdentityLookup`
exactly). Authorization baked in from the start (same T9 process rule).

**Instructions:**
1. `recurring_hire_templates` table + `0018_booking_recurring_hire_templates.sql`;
   sqlc queries; `port.RecurringHireRepository`; Postgres adapter.
2. `internal/booking/port.IdentityLookup` (new) + `internal/booking/
   adapter/identity` implementing it against the real
   `identityapp.Service.GetUser`.
3. `booking.proto` gains `RequestRecurringHire`, `ApproveRecurringHire`,
   `RejectRecurringHire`, `ListRecurringHireTemplatesForFacility` RPCs.
   - `RequestRecurringHire` requires `actor_user_id`. The handler
     resolves the actor via `IdentityLookup.GetUser` and checks `Roles`
     contains `club` — **never a self-declared `is_club` field on the
     request DTO** (T10 retro finding 3's checklist, A4 above — this is
     the item that actually fires for this ticket).
   - `ApproveRecurringHire`/`RejectRecurringHire` require `actor_user_id`;
     the handler resolves the template's `CourtID` → `Facility` via the
     **reused** `FacilityLookup` from T11.2 and calls
     `Facility.EnsureOwner(actor_user_id)`.
4. On `ApproveRecurringHire`: call `GenerateOccurrences` (T11.4), then
   attempt `booking.app.Service.CreateBooking` for each occurrence with
   `Source = SourceRecurringHire`. **A per-occurrence conflict
   (`ErrCourtDoubleBooked`) does not fail the whole approval** — that
   occurrence is recorded as skipped/conflicted and reported back in the
   response (which occurrences succeeded, which were skipped and why);
   the template itself still transitions to `approved`. State this
   explicitly in the ticket so an implementer doesn't default to
   "abort the whole approval on the first conflict," which would make a
   single already-booked Tuesday block an entire 12-week request.
5. **Creation-RPC checklist (A4) — both items checked, recorded in the
   PR:** (1) `ID` server-generated, no squatting surface. (2) Role
   self-assignment — the one that fires here, covered by item 3 above;
   state in the PR that this was the specific scenario the checklist item
   was written for.
6. Error handling — **gRPC codes only:**
   - Non-`club` actor on `RequestRecurringHire` → `PermissionDenied`.
   - Non-owner actor on `ApproveRecurringHire`/`RejectRecurringHire` →
     `PermissionDenied`.
   - Unknown/malformed `CourtID` on request, or unknown template ID on
     approve/reject → `NotFound`.
   - Approving/rejecting an already-approved/rejected/cancelled template
     → `FailedPrecondition`.
   - Invalid schedule (`StartTime >= EndTime`, non-positive
     `EndAfterOccurrences`) → `InvalidArgument`.
7. **Reminder (A2):** if this PR's merge includes a hand-resolved
   conflict against T11.2's shared `FacilityLookup` port, verify with
   `git diff --cached` before pushing.

**Cross-context dependency check:** calls into `internal/facilities`
(reused port, A8) and `internal/identity` (new port, `GetUser`/`Roles`
confirmed to exist since T10.2, A8).

**Story points:** 8. **Role:** principal-engineer. **Labels:**
`sprint:t11`, `role:principal-engineer`, `type:story`, `points:8`.

---

### T11.6 — Vue: Club rental request/approval flow (Flow 7)

**Story:** As a Club, I want to request a recurring slot and see its
status, and as a Facility Owner, I want to review and approve or reject
incoming requests, so that Club rentals are a real, usable flow rather
than an API only a script could drive.

**Description:** Depends on T11.5.

**Instructions:**
1. Club-facing: a request form (Court, weekday, time range, start date,
   end condition) calling `RequestRecurringHire`, plus a status view
   (`requested | approved | rejected`) reading back the actor's own
   templates.
2. Facility Owner-facing: an incoming-requests panel (reuses
   `FacilityOnboarding.vue`'s owner-console patterns) calling
   `ApproveRecurringHire`/`RejectRecurringHire`, showing the
   per-occurrence generation result T11.5's approval response returns
   (which weeks were booked, which were skipped and why) — never a bare
   "approved" with no visibility into partial conflicts.
3. **Absence assertion (T10 retro finding 5, A6 above — this is the
   ticket it fires for):** the "Request a recurring rental" control must
   be absent for a logged-in actor whose `Roles` do not include `club`.
   The check must cover semantic signals — `aria-label`, `<label>`
   association, ARIA `role` — not just native `id`/`name`, reusing
   `findGenderControls()`'s shape (`web/src/test-support/
   genderControlAssertions.ts`) as the reference pattern, not a
   screen-local reimplementation that only checks one signal type.
4. No fabricated data: a rejected request shows the rejection as a real,
   terminal state, never implying it can still be approved later without
   a fresh request.

**Cross-context dependency check:** calls `booking` (T11.5's
`RequestRecurringHire`/`ApproveRecurringHire`/`RejectRecurringHire`/
`ListRecurringHireTemplatesForFacility`, read+write — confirmed to exist
by dependency order below) and, read-only, `identity` (to determine
whether the current actor has the `club` role, for item 3's absence
check).

**Story points:** 5. **Role:** ux-ui-designer. **Labels:** `sprint:t11`,
`role:ux-ui-designer`, `type:story`, `points:5`.

---

### T11.7 — WCAG 2.2 AA audit + remediation across shipped screens

**Story:** As a Player or Host using assistive technology, I want every
screen this platform has shipped to actually meet the accessibility bar
this project has claimed since `docs/pickleball-platform-spec.md`'s own
"Accessibility: WCAG AA" line, so that the bar is verified, not assumed.

**Description:** No dependency on any other T11 ticket — audits
**existing** shipped screens (T7 through T10; T11.3/T11.6's *new* screens
build to the same bar from the start per their own Instructions, per A6 —
this ticket does not need to wait for them and does not re-audit them).
Scope: the criteria `docs/requirements/research-accessibility-i18n.md`
§1 already named as load-bearing for this app's two flagship flows —
3.3.1 (Error Identification), 3.3.3 (Error Suggestion), 3.3.4 (Error
Prevention — Legal/Financial/Data, directly relevant to Payments/
Booking's cancellation and payment flows), 4.1.3 (Status Messages) — plus
a general WCAG 2.2 AA pass, not limited to only those four. Builds on,
does not re-litigate, the design review's own round-8 contrast findings
(`docs/design/v1-review-round-8.md`) — check they still hold rather than
re-deriving them from scratch.

**Instructions:**
1. Automated pass (axe-core or equivalent) across every route
   `web/src/router/index.ts` currently registers, plus a manual keyboard-
   only and screen-reader spot check of the two flagship flows (booking a
   court, joining/paying for a Social Game — spec's own "flagship flows"
   framing).
2. For each of the four named criteria specifically: confirm every form
   (booking, registration, payment, Facility onboarding, Competition
   entry, and this sprint's new discount/rental forms if they land first)
   identifies invalid input in text, not color alone (3.3.1); offers a
   correction suggestion, not just "invalid" (3.3.3); and that any
   irreversible/financial action (cancel a paid booking, confirm a
   payment) has a confirm-before-commit step or an undo window
   (3.3.4). Confirm async state changes (payment confirmation, waitlist
   promotion, the T10.8 display-name resolution) announce via
   `aria-live`, extending — not re-deriving — the convention T10.8's own
   review found missing on two new components and fixed
   (`GameJoinPanel.vue`/`GameCheckout.vue`/`CompetitionCheckout.vue`/
   `CourtBookingFlow.vue`/`DisplayName.vue`/`VenueName.vue` already carry
   it; audit whether any other shipped async-state component doesn't).
3. Fix what's found, don't just report it — this is a hardening pass, not
   a spike. Findings that are genuinely out of scope (e.g. a finding that
   implies a product decision, not a technical fix) are named explicitly
   and deferred with reasoning, not silently dropped or silently fixed
   with a guess.
4. Every fix gets a regression test (an axe-core assertion, a semantic-
   query test, or both) — a fixed-then-reverted contrast/label bug with
   no test is exactly the "assumed, not proven" shape QA's brief warns
   against.

**Cross-context dependency check:** none — audits existing Vue screens
only, no new backend calls.

**Story points:** 8. **Role:** ux-ui-designer. **Labels:** `sprint:t11`,
`role:ux-ui-designer`, `type:bug`, `points:8`.

---

### T11.8 — Chore: retroactively close T5–T10 issues; make issue-closing a manual merge-flow step (closes #111)

**Story:** As the team, I want the GitHub issue list to actually reflect
what's shipped, so that "open issues" means "not yet done," not "done
five sprints ago and nobody told GitHub."

**Description:** Closes issue #111 (opened this ceremony — full context,
including the discovery that #70/#96/#97/#98 all show
`closed_by_pull_requests: {"total_count": 0}` despite merged implementing
PRs, and the root cause (this project never merges into GitHub's default
branch, so `Closes #N` structurally cannot fire), is in the issue body,
not restated here). Per T10 retro finding 6. No dependency on any other
T11 ticket.

**Instructions:**
1. For every currently-open issue from #6 through #98 (T5.1 through the
   T10 follow-ups), confirm its implementing PR is actually merged
   (cross-check `HANDOFF.md`'s per-phase merge-order notes, not just the
   issue list going stale) and close it with `state_reason: completed`
   plus a comment naming the merged PR(s).
2. Update `docs/process/sprint-process.md`'s Ceremony 1 section: the
   sentence "GitHub does this automatically via 'Closes #N' in the PR
   body, or manual linking if the PR doesn't close the issue outright" is
   simply wrong on this repo's branch topology — replace it with wording
   that states closing is always a manual step here, and tighten the
   per-ticket DoD's step 5 ("The GitHub issue is linked to the merged PR
   and closed") to say explicitly *how* — a manual `issue_write` close
   call after merge — rather than leaving it passively phrased in a way
   that reads as automatic.
3. **Disclosed-but-out-of-scope note, per A3's standing rule:** if, while
   closing issues, this ticket's implementer notices T10.1–T10.8 were
   never opened as individual GitHub issues at all (no issue numbers
   between #79 and #96 correspond to them — flagged in issue #111's body
   as a secondary, related gap), that observation goes in this ticket's
   own PR body as a named, tracked note — not silently backfilled (out of
   this ticket's scope) and not silently dropped either.

**Cross-context dependency check:** none — GitHub API + one doc edit.

**Story points:** 2. **Role:** business-analyst. **Labels:**
`sprint:t11`, `role:business-analyst`, `type:chore`, `points:2`.

---

### T11.9 — Chore: generalize per-context UUID-shaped fixture-ID helpers

**Story:** As the team, I want every context's test fixtures to use
UUID-shaped IDs by default, so that the `uuidShape` guard's own test
suite stops being the thing that discovers non-UUID fixtures by
accident, sprint after sprint.

**Description:** Per T10 retro finding 2 (this project's third and
fourth instance of the identical bug class in one sprint alone, fourth
and fifth+ project-wide). PR #107 already fixed `internal/payments/
adapter/grpcapi`'s two files with one shared UUID-shaped fixture constant
block — this ticket generalizes that exact pattern, not a new one, to
every other context. No dependency on any other T11 ticket.

**Instructions:**
1. Audit every `adapter/grpcapi` and `adapter/postgres` test file across
   all six contexts (booking, socialplay, payments, facilities,
   competitions, identity) for non-UUID-shaped literal IDs (`"entry-1"`,
   `"id-1"`, `"booking-1"`, etc. — the exact shapes PR #89's root cause
   and PR #107's fix both already catalogued).
2. Where found, replace with a shared `fixture*ID` constant block per
   package (one per context, reused by every test file in that package —
   PR #107's shape, generalized, not reinvented per context).
3. Where **not** found (a context's fixtures are already UUID-shaped),
   state that explicitly in the PR as a checked-clean result, not silent
   omission — this project's own "checked, not assumed" discipline
   applied to a negative finding.
4. This does **not** include the stronger enforcement mechanism (a lint
   rule or runtime guard preventing a *future* non-UUID literal) — A11's
   recorded PE/QA disagreement leaves that out of scope for this ticket
   specifically; note it as a named, deferred candidate in the PR, not
   silently declined.
5. Verify each fixed file's suite still passes **and** still exercises
   the `uuidShape` guard's negative path somewhere in the suite (a
   fixture fix that accidentally removes the malformed-ID test case
   entirely would be a regression in coverage, not just a fix) —
   CLAUDE.md rule 10 discipline applied to a refactor, not just new code.

**Cross-context dependency check:** touches existing test files across
all six contexts — horizontal, not vertical; no enum/port gap involved.

**Story points:** 3. **Role:** qa. **Labels:** `sprint:t11`, `role:qa`,
`type:chore`, `points:3`.

---

## Dependency order (see A10 for dispatch-isolation waves)

```
Wave 1 (parallel, ≤5 implementers, worktree-isolated):
  T11.1 (DiscountRule domain)            ─┐
  T11.4 (RecurringHireTemplate domain)   ─┤
  T11.7 (WCAG audit)                     ─┼─→ independent, ships whenever
  T11.8 (issue-closing chore)            ─┤
  T11.9 (fixture-ID helper)              ─┘

Wave 2 (1 implementer):
  T11.2 (DiscountRule wiring, needs T11.1)
  — builds internal/booking/port.FacilityLookup, migration 0017

Wave 3 (parallel, ≤2 implementers, worktree-isolated):
  T11.5 (RecurringHireTemplate wiring, needs T11.4 + T11.2's FacilityLookup)
  — builds internal/booking/port.IdentityLookup, migration 0018
  T11.3 (Pricing/discount UI, needs T11.2 only)
  ⚠ no expected file overlap (Go backend vs. Vue) — genuinely parallel

Wave 4 (1 implementer):
  T11.6 (Club rental UI, needs T11.5)
  ⚠ may share files with T11.3 (router, checkout components) if timing
    overlaps despite the wave gap — whichever merges second resolves on
    its own source branch, one-line reasoning in that PR's body
```

Total: **47 points** across 9 tickets (T10: 37/8, T9: 49/10 — comparable
in shape to T9: three real workstreams plus small, well-specified
hardening/chore tickets, not one overloaded slot).

## Explicitly deferred, not silently dropped (HANDOFF Cross-cutting review)

Per this ceremony's instruction to check `HANDOFF.md`'s Cross-cutting
section beyond what T10's retro already pointed at:

- **Write-handler malformed-ID guard extension** — already closed, T10.7
  (issue #97). No T11 action needed; `HANDOFF.md`'s Cross-cutting bullet
  describing it as "not yet closed" is now stale and gets corrected in
  this sprint's `HANDOFF.md` update (see below), not left to mislead a
  future reader.
- **`RefundPayment` wiring** (open since T6.5, `HANDOFF.md`'s own
  proposed-follow-up text already fully specifies it: online via
  `PaymentProcessor.RefundPayment`, offline as a Host/Game-Admin action,
  pushing `PaymentStatusRefunded` through `RegistrationPaymentUpdater`).
  **Not taken this sprint.** Reasoning: well-specified, self-contained,
  genuinely T11-sized (5-ish points) — but T11 as scoped above is already
  47 points across a 4-deep Booking-context wave chain; adding a fourth
  independent workstream repeats the exact overload pattern this
  ceremony's own A1 argued against taking a fourth time. Real candidate
  for T12's Ceremony 1 to pick up first, not re-deferred in prose without
  a reason — the reason is named here.
- **`CancelGame` with `HostID`-scoped authorization** (open since T5.5,
  similarly fully specified already). **Not taken this sprint**, same
  reasoning as `RefundPayment` — both are aging, well-specified, and
  real, but this ceremony chose the three roadmap items + two process
  chores over a fifth independent workstream. T12 candidate.
- **Observability (Sentry + slog + uptime)** — **not taken**, lower
  urgency than the above (no specific incident or aging-issue pressure
  named anywhere in `HANDOFF.md` the way `RefundPayment`/`CancelGame`
  have); genuinely T12+ or later.
- **Real auth/JWT** — **deliberately not taken**, and not because it's
  unimportant: it's the opposite, the reason Identity/Users was built
  first (T9's own roadmap ordering recommendation, carried out at T10).
  It is large enough — every context's `actor_user_id`/`actor_player_id`
  claimed-actor pattern (four contexts now) needs to migrate to a
  verified principal, `CreateUser`'s ID-squatting gap needs to close per
  its own stated closure condition — to deserve its own sprint's full
  Ceremony 1/2 scoping, not a ticket squeezed into T11 alongside three
  other workstreams. Candidate for its own dedicated near-term sprint;
  named here so it isn't mistaken for forgotten.
- **CI server-side wiring** (Jenkins job/webhook/branch protection —
  SCRUM-6 shipped the repo-side pipeline definition only). **Not
  ticketed**, same reason it hasn't been ticketed in any sprint since
  SCRUM-6: it requires access to a real, reachable Jenkins instance and
  admin credentials no session in this project's history has had —
  infra-provisioning work outside what a coding session can perform, not
  a scoping choice.
- **Swap docker `initdb.d` for `golang-migrate`/`goose`; ISO-8601
  weekday numbering** — unchanged, low-urgency infra debt, no new
  pressure this sprint. Left as-is.

## `HANDOFF.md` updates this ceremony's PR carries

Since `HANDOFF.md`'s "Current state" narrative never got a `**T10 — ...**`
paragraph the way T5–T9 each did (confirmed by grep this ceremony — only
the Docs-index table row exists for T10), this PR adds the missing
paragraph (summarizing T10's outcome: Identity/Users + `Match` shipped,
ADR-0012, the three T9 follow-ups closed, the retro's six findings) so
`HANDOFF.md` accurately resumes from T10 before T11 work starts, plus a
new Docs-index row for T11 pointing at this plan and issue #111, plus the
one-line correction to the write-handler-guard Cross-cutting bullet noted
above.

## Definition of Done (sprint-level)

All 9 tickets merged per `sprint-process.md`'s per-ticket DoD (PR-only,
CLAUDE.md rule 9 — no exception for this plan or issue #111, both of
which land the same way); sprint goal met (discount UI live end to end,
Club rental request/approval live end to end generating real Bookings,
WCAG 2.2 AA audit complete with fixes, T5–T10 issues actually closed);
retro held (`docs/process/t11-retro.md`) with findings indexed from
`docs/LESSONS.md`'s `## T11 sprint retro`; `HANDOFF.md` updated for T12
to resume from, including which of `RefundPayment`/`CancelGame`/
observability/real-auth (if any) T12's own Ceremony 1 picks up first.
