> **Round 1 resolutions (see `v1-review-round-1.md` for full reasoning):** all
> seven §7 questions answered; two of them are provisional defaults pending
> explicit user sign-off, not fully settled — `Registration.GuestCount` is
> assumed **set-once** (change of mind = cancel and re-register, not an
> in-place edit) because a mutable guest count would silently bypass the
> capacity-guard trigger's UPDATE-path guard; and the camera-link field is
> assumed to require a **facility-onboarding consent/signage attestation
> checkbox**, not just an informational note. Both were put to the user
> directly and not yet confirmed — treat as the working assumption for round
> 2's artifact and this doc's §3, revisit if the user answers differently.
> Also: the discount `EndCondition` stays the three mutually-exclusive
> variants in §3.2 (no compound "date OR occurrence count" condition for
> v1) — the round-1 artifact's compound example was a mockup-only mismatch,
> now fixed. And: **T5's Social Play code is not yet merged into this
> branch** — anything below describing it as "already built"/"shipped"
> means "built on an unmerged sprint branch," not "on `HEAD`."

# Pickleball Platform — System Design v1 (round 0, pre-review draft)

**Status:** draft, round 0 — written before Designer/PM/Principal Engineer/PO
review. Produced from a detailed requirements list the user gave directly in
chat (reproduced/organized below); the referenced external design doc
(`claude.ai/design/...`) was not accessible (HTTP 403 — needs a claude.ai
login this session's fetch tool doesn't carry), so this is a fresh design
reconciled against this project's existing locked decisions, not a review of
that doc. **Purpose of this file:** give the review team one place to
brainstorm/verify against, not a finished spec — round 1+ is expected to
change it.

**Method:** every requirement below is mapped to (a) an existing spec
section/data model entity it extends, (b) a **new** entity/concept this
project doesn't have yet, or (c) a **conflict** with an existing locked
decision or a previously-identified risk — conflicts are flagged explicitly,
not silently resolved, per this project's "don't manufacture consensus"
discipline.

---

## 1. Requirements as given, organized

1. Facility owners let users book courts.
2. Hosts (owners or others) add: facility description, photos, address,
   link to security cameras.
3. Court rental config: price by date/time; monthly rent price; discount
   price for specific periods, with **end-after conditions**.
4. Hosts can rent courts to organise social games; owners can also host
   social games (owner is a kind of host).
5. Hosts define game rules and payment method (online or cash).
6. Users can also be hosts; users can join social games; users/players can
   find social games.
7. A host can add a new facility if it doesn't exist on the platform yet.
8. A club can hire courts for specific days/times at a discount, depending
   on the facility owner's config.
9. A host can organize competitions and advertise them via WhatsApp,
   Facebook groups, X, Instagram.
10. A host can link their social media accounts to post ads for social
    games; **users can reply on those social platforms to register**.
11. A user can register for friends — how many depends on the host's setup.
12. Host game setup: cancellation-without-payment cutoff; capacity per
    court.
13. A social game can have multiple users/players.
14. Player skill level depends on how long they've been playing and how
    many wins they have.
15. Automatic matchmaking based on level **and gender**; host can override.
16. Platform targets: web, mobile, iPhone, iPad.

---

## 2. Mapping against the existing spec/data model

| # | Requirement | Maps to |
|---|---|---|
| 1 | Court booking by facility owner | **Existing.** `pickleball-platform-spec.md` §3.2/§3.4, already built (Booking context, T0-T4). |
| 2 (desc/photos/address) | Facility metadata | **Existing.** §3.2. |
| 2 (security camera link) | — | **New.** No entity today. See §3 below. |
| 3 (date/time + monthly price) | Pricing rules | **Existing.** §3.3, `pricing_rules` table, T1. |
| 3 (discount + end-after condition) | — | **New.** See §3 below — this is materially different from a pricing *band*, it's a time-bounded promotion. |
| 4, 5, 6, 13 | Social games, hosts, rules, payment method | **Existing, in progress.** §3.6, D2/D3, `Game`/`Registration` (T5, just built this session) already cover host-owned games, multi-player, paid/unpaid. Owner-as-host is already implicit (any user can hold multiple roles, §3.1) — worth an explicit test, not just an assumption (see §6). |
| 7 | Host creates a new facility if missing | **Existing.** §3.10 ("If a facility isn't listed, a user can create it... queued for owner claim/verification"). Not yet built (no Facilities CRUD ticket exists in HANDOFF.md T0-T6). |
| 8 | Club recurring hire at a discount | **Existing + New.** §3.5 (recurring hire) exists as a concept but has no discount mechanism — depends on the same new discount concept as #3. |
| 9 | Competitions + social media advertising | **Existing + Flagged conflict.** §3.8 (competitions) exists conceptually (not built). §7 of the spec already addresses "social promotion" — see the **conflict in §4 below**, this is the important one. **Round 4 correction:** rounds 1-3's visual artifact claimed this requirement was covered but had no competition-scoped screen at all — every mockup was Game-scoped. Round 4 added a dedicated competition-setup screen (level-seeded pool entries + the same shareable-link advertising pattern as a social game), so the artifact now actually shows what this row describes rather than asserting it. (Round 8: corrected this row's own wording from "bracket preview" to match round 5's fix in the artifact itself — the shown format is round robin, which doesn't produce a bracket.) |
| 10 (link accounts to post) | — | Half-new: posting an ad via a linked account's official API is a reasonable extension of §7's "shareable link... posts to any platform manually or via official share APIs." |
| 10 (**reply-to-register**) | — | **Direct conflict with an existing, deliberate risk call.** See §4. |
| 11 | Friend/plus-one registration | — | **New.** `Registration` (T5.2) has no guest-count field today. |
| 12 (cancel-without-payment cutoff) | Per-facility cancellation window | **Already identified as a gap, not yet built.** This is P1 finding #12 from `docs/requirements/README.md` (per-facility configurable cancellation window) — the user is independently re-requesting something already on the books. Good corroboration, not a new idea. |
| 12 (capacity per court) | — | **Existing**, but worth a precise reading: capacity today (T5.1) is *per Game*, not *per court*. If a Game spans multiple courts, does capacity mean "total across all courts" or "per court"? Not ambiguous in the current implementation (it's a single `Game.Capacity` int, i.e. total) but the user's phrasing ("how many players for a certain courts") suggests they may be picturing per-court capacity. **Flagged for PM/PO in round 1**, not assumed either way. |
| 14 (skill from tenure + wins) | Matchmaking rating (D2) | **Simplifies, and arguably conflicts with, the existing locked design.** See §5. |
| 15 (level + **gender**) | Matchmaking (D2, §3.7) | **New axis.** Gender isn't mentioned anywhere in §3.7's automated-matchmaking description today. This is domain-accurate (mixed doubles is a real, common pickleball format) — see §5. |
| 16 | Web + iPhone + iPad | **Existing, needs a precision.** D4a already locks Go backend + Vue web + Swift/SwiftUI iOS. "iPad" isn't previously called out explicitly — see §6. |

---

## 3. New concepts this design needs to add

### 3.1 Security camera link (requirement #2)
Simplest honest model: a `facilities.security_camera_url` (or a small
`facility_cameras` table if a facility can have more than one, e.g. one per
court) storing a link/embed reference to a third-party camera feed the
owner already has (e.g. a RTSP/HLS stream URL, or a link to a vendor's own
viewing page — Nest, Ring, a court-specific vendor like PlaySight/SwingVision
are real examples in this exact market). **This platform does not host or
process video** — it stores a reference and, at most, embeds a
vendor-provided player/link. Flag for round 1: is this "store a URL" (cheap,
matches the "managed services, not your own plumbing" principle in spec §1)
or does the user want camera *footage* surfaced inside the app (materially
bigger scope — storage, privacy/consent implications under the GDPR/CCPA
findings already in `docs/requirements/research-security-compliance.md`)?
Defaulting to "store a URL/reference, embed the vendor's own player" as the
v1 assumption — **confirmed in round 1** by all four review roles.

**Round 1 addition:** storing a reference instead of footage sidesteps this
platform's own storage/processing obligations, but not the facility
operator's obligation to have lawful basis and adequate notice (signage,
sometimes explicit consent depending on jurisdiction) for recording real,
identifiable people at their venue. **Working assumption (put to the user,
not yet confirmed): facility onboarding requires a consent/signage
attestation checkbox** ("I confirm this facility has appropriate
signage/consent for any security cameras linked here") before the
camera-URL field can be saved — cheap now, expensive to retrofit once many
facilities have live links. This is additive to the "store a URL" answer,
not a change to it. Also: the camera field is host-facing only in this
design (facility settings), with no player-facing surface — stated here as
an explicit requirement, not left implicit in what a mockup happens to
show.

### 3.2 Discount rules with end-after conditions (requirements #3, #8)
A new `discount_rules` concept layered on top of the existing `pricing_rules`
resolution (§3.3): `DiscountRule{ID, CourtID or FacilityID, DiscountType
(percent|fixed_amount), AppliesTo (individual|recurring_hire|club), StartsAt,
EndCondition}`. `EndCondition` is itself a small variant, matching how real
comparable platforms model promotions: `EndAfterDate(date)`,
`EndAfterOccurrences(n)` (e.g. "first 10 bookings"), or `NoEnd` (open-ended,
e.g. a permanent club rate). This composes with `pricing_rules` the same way
`ResolvePrice` already composes weekday/peak/weekend bands — round 1 should
decide whether a discount *replaces* the resolved price or *modifies* it
(percent-off the resolved band price is almost certainly right, but write it
down, don't assume it). **Reuses ADR-0002's precedent** (an ambiguous
pricing/discount match is a domain error, not silently resolved by
priority) — two overlapping discount rules on the same slot should behave
the same way overlapping pricing rules already do.

### 3.3 Friend / plus-one registration (requirement #11)
Extend `Registration` (T5.2) with a `GuestCount int` field, bounded by a new
`Game.MaxGuestsPerRegistration int` the host sets (default 0). Capacity
counting (built on an unmerged T5 sprint branch, T5.4's DB-enforced guard —
not yet on `HEAD`) must count `1 + GuestCount` per registration, not 1 —
**this is a direct, material change to the capacity invariant T5 just built
and verified**, not a cosmetic addition.

**Round 1 correction:** the fix is *not* a one-line `count(*)` →
`sum(1 + guest_count)` swap, as this section originally (wrongly) claimed.
The actual trigger (`enforce_game_capacity()`) only re-checks capacity on
`INSERT` or a `cancelled → active` `UPDATE` — an `UPDATE` that leaves
`status` unchanged is deliberately skipped today, correctly, because
nothing else about a registration can currently change its seat
consumption after the fact. Once `GuestCount` exists, that stops being
safe **if** it's editable post-registration: an existing registrant
raising their own guest count is exactly the skipped `UPDATE` path, so a
naive aggregate-function swap would miss a real capacity overrun.
**Working assumption (put to the user, not yet confirmed): `GuestCount` is
set-once at registration time** — change of mind means cancel and
re-register, the same pattern already used elsewhere in this domain —
which keeps the fix scoped to the aggregate-function swap, because the
only paths that ever consume new seats stay `INSERT` and
`cancelled → active`, both already guarded. If the answer turns out to be
"must be editable in place," the trigger's UPDATE-branching logic needs to
change too, not just its aggregate function — a materially bigger ticket
than this section originally implied. Log as a concrete T7+ ticket
candidate either way, not something to silently patch in this design doc.

### 3.4 Skill level and gender in matchmaking (requirements #14, #15)
See §5 — deliberately separated out because it interacts with an existing
locked decision (D2) and a research finding (docs/requirements/
research-functional.md §2.1/§2.2), not a clean net-new addition.

---

## 4. Conflict to resolve, not silently build around: social-media reply-to-register

The existing spec (§7, "Social promotion & RSVP — the feature to rethink")
already examined exactly this idea and rejected the "auto-monitor public
platform replies" approach for two concrete reasons still true today:

1. **Platform terms of service**: WhatsApp's Business API only sees messages
   sent *to your own number*, not public group chatter; the Facebook Groups
   API is closed to third parties; X and Instagram's APIs for reading public
   replies are restricted and/or paid, and scraping public posts for replies
   is against most of these platforms' terms.
2. **Reliability**: even where technically possible, parsing free-text
   replies ("I'm in", "put me down for 2", a reply-to-a-reply thread) into a
   structured registration is fragile compared to an in-app RSVP.

The user's requirement #10 ("users can reply into these social media
platforms to register") is functionally the same idea the spec already
flagged as high-risk. **This is not something to quietly reinterpret away**
— it's exactly the kind of requirement this project's process exists to
surface and verify before building, not after. Recorded here as an explicit
question for the round-1 review team (Designer + PM + Principal Engineer +
PO), with the spec's own already-designed alternative restated as the
likely answer:

- Each game/competition gets a shareable link/card a host posts manually (or
  via an official, ToS-compliant share API) to any platform.
- RSVPs land **in-app** through that link, always an accurate, structured
  count — this satisfies "advertise via social media" (requirement #9)
  fully, and gets most of the way to requirement #10's *intent* (let social
  media drive registrations) without the ToS/reliability problems.
- Optionally, a bot on a channel **the platform controls** (a WhatsApp
  Business number people message directly, a Telegram/Discord group) can
  parse a simple, constrained format (e.g. `IN 2` for "me + 1 guest",
  connecting directly to requirement #11's friend-registration count) —
  this is meaningfully different from monitoring public replies on
  Facebook/X/Instagram, because it's a channel the operator owns and
  controls, not a public platform's API/ToS surface.

**This design does not implement generic social-reply parsing** pending
round-1 confirmation that the above is an acceptable resolution — it is the
existing, already-reasoned position in this project's own spec, being
carried forward rather than re-litigated silently.

---

## 5. Matchmaking: reconciling requirement #14/#15 against D2 and existing research

D2 (locked decision) already specifies matchmaking as "automated from each
player's historic data (games/wins/losses/rating)... new players seeded by a
self-reported starting level." The user's new phrasing — "level... depends
on how long they've been playing and how many wins they have" — reads as a
simpler formula (tenure + win count) than D2's rating-based design, and is
also simpler than what `docs/requirements/research-functional.md` §2.1
recommended (DUPR-style rating should move on **score margin vs.
expectation**, not just win/loss counts).

**Not silently resolving this either way.** Three readings, for round 1 to
pick from explicitly:
1. The user is describing an intuitive/lay explanation of what a rating
   *already* correlates with (more games + more wins → higher rating), not
   proposing to replace D2's rating system with a raw tenure+win-count
   formula. Most consistent with the existing locked decision and the
   research finding — likely the right reading, but should be **confirmed
   with PM/PO**, not assumed.
2. The user genuinely wants a simpler, transparent formula (e.g. `level =
   f(games_played, win_rate)`) instead of an Elo-style rating — a real,
   valid product choice (easier for players to understand "why is my level
   X"), but it would need to formally reopen D2, which CLAUDE.md's "Locked
   decisions — do NOT reopen" section says shouldn't happen without an
   explicit decision, not a design-doc assumption.
3. Both exist: a simple, player-facing "level" (tenure+wins, easy to
   understand) *and* an internal rating (Elo-style, used for actual
   matchmaking precision) — common in real systems (a public-facing "skill
   tier" vs. an internal precise rating players don't see directly).

**Gender as a matchmaking axis (requirement #15)** is a genuine,
domain-accurate addition — mixed doubles (2 men + 2 women per pairing) is a
standard, common pickleball format this project's research hadn't
previously named as a matchmaking input. Recommend adding `Gender` as a
declared user attribute (self-reported, same pattern as skill level) and a
`Game.MatchmakingMode` that can require gender-balanced pairing, alongside
the existing skill-balance requirement — additive to D2, not a conflict.

---

## 6. Platform/UI targets (requirement #16)

D4a already locks: Go backend, Vue 3 web (SSG public pages + SPA app), and
Swift/SwiftUI **iOS**. Two precisions this design adds, not conflicts:

- **"iPhone, iPad" means one SwiftUI codebase with adaptive layouts**
  (size classes / `NavigationSplitView` on iPad vs. a single-column stack on
  iPhone), not two separate apps — this is standard SwiftUI practice and
  doesn't change the D4a decision, just clarifies it explicitly for the
  design-review round, since "iPhone, iPad" as a requirement could be
  misread as needing a distinct iPad app.
- **Android** is not named in the user's latest requirements list, and D4a's
  own text already flags it as an "open sub-decision... whether Android (a
  later Kotlin app) is in scope." Not resolved here — carried forward as
  still-open, not silently decided either way.

The visual/UX mockups produced alongside this document (round 1 artifact)
target: a responsive Vue web layout (desktop + tablet + mobile web
breakpoints) and an iOS/iPadOS-adaptive treatment of the same core flows,
consistent with the above.

---

## 7. Open questions for round 1 (Designer + PM + Principal Engineer + PO)

**All seven answered in round 1 — see `v1-review-round-1.md` for full
reasoning.** Short answers: (1) store-a-URL, confirmed, plus a new
consent-attestation requirement; (2) modify (percent/fixed-off), confirmed;
(3) per-Game total, confirmed; (4) proceed, but the fix is bigger than
originally described here (see §3.3's round-1 correction above); (5) the
in-app-RSVP alternative confirmed accurate against the spec; (6) additive to
D2 (a public "Level" label backed by the existing rating), not a reopening;
(7) additive, confirmed, self-reported/optional. Two of these (1 and 4) have
a provisional-default answer pending explicit user sign-off — see the note
at the top of this document.

1. Security camera integration: store-a-reference-URL (cheap) vs.
   surface-footage-in-app (real scope, real privacy implications) — §3.1.
2. Does a discount *replace* or *modify* (percent-off) the pricing-rule
   result? — §3.2.
3. Per-court vs. per-game capacity — is the current T5 model (capacity is
   per-Game, spanning however many courts the Game books) actually what's
   wanted, or does "capacity per court" mean something more granular? — §2
   row 12.
4. Friend/plus-one registration changes the capacity-counting invariant
   T5.4 just built and verified (row-count → seat-count) — confirm this is
   wanted before it's ticketed, since it touches a recently-hardened
   concurrency-sensitive trigger. — §3.3.
5. Social-media "reply to register" — confirm the spec's existing
   ToS-safe alternative (in-app RSVP via shared link + optional owned-channel
   bot) is the accepted resolution, not generic reply-parsing. — §4.
6. Skill "level": confirm which of the three readings in §5 is intended —
   does this reopen D2, or is it a restatement/addition to it?
7. Gender-based matchmaking: confirm the proposed additive design (a
   declared `Gender` attribute + a per-game matchmaking-mode flag) matches
   intent.

These seven questions are the round-1 review's primary agenda — the visual
mockups (companion artifact) illustrate the UI assuming the "likely" answer
noted above for each, so round 1 can react concretely rather than
abstractly.
