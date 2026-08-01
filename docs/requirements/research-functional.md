# Functional Requirements Research — Verification Against Industry Practice

**Purpose:** verify and strengthen the functional requirements in
`docs/pickleball-platform-spec.md` (§3, §6, §7, §8) against real-world
court-booking platforms, DUPR's actual rating system, real pickleball
formats, and documented scheduling-system edge cases. Research only — no
code or spec changes made here.

**Method:** web search + fetch against product docs/help centers of
comparable platforms (Playtomic, CourtReserve, PlayByPoint, Skedda,
AllBooked), DUPR's own materials and third-party explainers, pickleball
tournament-format references, and scheduling-software DST/timezone
literature. Each finding below is tagged **Aligned** / **Gap** /
**Reconsider** against the current spec.

---

## 1. Competitor/comparable platform research

### 1.1 Cancellation windows are club-configurable, not a single global rule
Playtomic does not enforce one cancellation policy platform-wide — each club
sets its own deadline (some allow cancellation any time, some require 24h
notice, some allow none at all), enforced by the app at cancel time. Some
clubs use a 4-hour free-cancellation cutoff.

- **Verdict: Gap.** Spec §3.4 says only "Cancellation windows, refunds,
  no-show handling, waitlists" with no owner about who sets the window or at
  what granularity. Real platforms make the cancellation window a
  **per-facility (sometimes per-court-type) configurable policy**, not a
  platform constant. The spec's data model (§8) has no field for this — it
  should live on `facilities` (or a `booking_policy` sub-entity) alongside
  pricing rules, and `CancelBooking` needs to check it before allowing a
  free cancellation vs. a penalized one.

### 1.2 No-show and late-cancellation fees are real, standard, and separate from refund logic
CourtReserve-based clubs commonly charge a full court fee for no-shows and
cancellations inside the penalty window (e.g., <4h), and unpaid penalty fees
block the member's *next* reservation until settled. Cancelled-in-time
prepaid time is returned as **account credit**, not always a cash refund.

- **Verdict: Gap.** The spec's payment model (§5) has "paid/unpaid" as the
  only states. There is no `no_show` outcome, no penalty-fee concept, and
  no account-credit concept distinct from a straight refund. Real facilities
  rely on no-show fees to control the single biggest complaint host/owners
  have (empty booked courts). Recommend adding a `no_show` booking outcome
  and considering an account-credit ledger as a lighter-weight alternative
  to Stripe refunds for in-window cancellations.

### 1.3 Waitlists auto-fill from cancellations in real systems
CourtReserve explicitly automates this: "When a slot becomes free, it
immediately goes to the waitlist, turning a potential loss into a recovered
booking," combined with automatic cancellation-window/no-show enforcement.
Playtomic's Open Match model instead auto-cancels under-filled matches
(1–3 players) 1 hour before start and does not clearly document an
auto-promote waitlist — cancellations there just reopen a spot for direct
re-request rather than a queued auto-fill.

- **Verdict: Gap.** The spec mentions "waitlists" once (§3.4, listed
  alongside cancellation/refunds/no-show with no design). It isn't in the
  data model (§8) at all — no `waitlist_entries` table, no rule for
  auto-promotion order (FIFO? skill-matched for games?) or for what happens
  when a promoted player doesn't confirm in time. This needs its own
  mini-spec before implementation: at minimum a table keyed to
  (court/slot or game), an ordered queue, and a promotion policy with a
  response-timeout so one unresponsive waitlisted player doesn't block the
  slot indefinitely — CourtReserve's real-time backfill implies exactly this
  timeout problem exists in practice.

### 1.4 Recurring hire / membership booking uses passes and reservation *rules*, not just a cron-like recurrence
PlayByPoint's "Reservation Rules" model lets a club configure how far in
advance booking opens (e.g., a slot becomes visible/bookable exactly 24h
ahead), guest-count limits per booking, and membership "booking passes"
that are redeemed per reservation (discounted/free courts during specific
shifts). This is materially richer than a plain weekly-recurrence template.

- **Verdict: Reconsider.** Spec §3.5 describes recurring hire as "weekly
  recurrence rules" only. Real platforms couple recurring/membership access
  to **booking-window rules** (how far ahead a slot opens) and to
  **entitlement passes** (X free/discounted bookings per period), both of
  which interact with pricing (§3.3) and payments (§5). Not necessarily a
  v1 requirement, but worth a note in §3.5/§8 that the recurring-hire
  template will eventually need a booking-window field and possibly a
  passes/entitlements concept, so the schema doesn't have to be reshaped
  later.

### 1.5 Split/shared courts are a real, common facility pattern this spec doesn't model
Multiple vendors (AllBooked, SmartHealthClubs) explicitly support courts
that are physically one space bookable as either a whole (e.g., one tennis
court) or as independent halves (e.g., two pickleball courts painted over
it) — "booking a tennis court blocks the entire space... booking a
pickleball court blocks only that portion and leaves the others open."

- **Verdict: Gap.** Spec §3.2/§8 models `courts` as flat, independent units
  under a facility with no parent/child or overlap relationship between
  them. This is extremely common in the real pickleball market specifically
  (pickleball lines painted on tennis/badminton courts). If any pilot
  facility has shared/split courts — plausible for a new pickleball venue —
  the current `EXCLUDE (court_id WITH =, during WITH &&)` constraint in §6
  **will not catch a conflict between a full-tennis-court booking and a
  half-court pickleball booking on the same physical space**, because
  they'd be different `court_id`s with no relationship recorded. This is
  the same shape of bug F1 already caught in the design review (Booking/Game
  not sharing an invariant) — worth flagging explicitly before a facility
  with shared courts onboards, even if deferred past v1.

### 1.6 Multi-court single booking is standard
PlayByPoint explicitly supports booking multiple courts at once in one
reservation (e.g., Court 1 and 2, 5–6 PM) — relevant since Games/Competitions
in this spec book "court(s)" (plural) per §3.4/§8.

- **Verdict: Aligned.** The polymorphic Booking model (D3b) with a
  game/competition able to reserve multiple court-rows already accommodates
  this if each court gets its own Booking row sharing a `game_id`/
  `competition_id` — worth confirming the domain layer actually creates N
  booking rows for an N-court game rather than assuming one.

---

## 2. DUPR verification

### 2.1 DUPR is a modified Elo system, not open-source
DUPR (Dynamic Universal Pickleball Rating) computes an **expected score**
before a match from the combined ratings of the two sides, compares it to
the **actual score** after, and moves ratings up or down based on
performance vs. expectation — the same Elo/chess-rating family of
algorithm, adapted for pickleball scoring. As of the most recent rule
update, a rating can now rise even on a loss (or fall on a win) if the
score outperforms/underperforms what was expected. The precise formula
(K-factor, weighting curve) is **not published** — described consistently
across sources as proprietary.

- **Verdict: Aligned, with a caveat.** The spec's D2 design ("automated
  from historic data — games played, wins, losses, and rating," feeding a
  "DUPR-style internal rating") is a reasonable simplified analog in
  *category* (skill rating from match outcomes) but the spec as written
  (§3.7) only mentions win/loss counts and rating, not **margin/score vs.
  expectation**, which is DUPR's actual core mechanic — a straight Elo on
  win/loss alone is a materially cruder system than DUPR (closer to
  classic chess Elo pre-margin-of-victory adjustments). Recommend the spec
  note explicitly that "rating" updates should be a function of the
  *score margin against the opponent's rating-implied expectation*, not
  just win/loss, if it wants players to recognize the mechanic as
  DUPR-like. This is a genuinely missing detail, not just a phrasing nit —
  it changes what data `matches` (§8) needs to store (both teams' scores,
  not just a winner flag).

### 2.2 DUPR weights match sources differently — a detail the spec's cold-start design doesn't yet capture
DUPR gives **self-reported** recreational match scores materially less
weight than scores **submitted by a club/tournament director** — clubs'
submissions count roughly 5x a self-reported match, letting a rating
stabilize in 5–10 games via verified submission vs. "a dozen or so"
self-reported games. New ratings are also flagged **provisional** (shown
with an asterisk) until enough matches accumulate — commonly cited as
roughly 10–20 matches before it stabilizes, with single-match swings of
0.2–0.4 possible before that.

- **Verdict: Gap.** The spec's cold-start design (§3.1, §3.7) has exactly
  one mechanism: a self-reported starting level, with no distinction
  afterward between a self-reported/organiser-recorded match and one a
  Game Admin verified on the day. Given this platform already has a
  **Game Admin** role (D3a) who "manages and directs players on the day,"
  it is well positioned to mirror DUPR's verified-vs-self-reported
  weighting: matches scored by a Game Admin present at the game should
  weight more than a player self-reporting after the fact. Recommend
  adding (a) a `recorded_by`/`verified` flag on `matches`, (b) a
  `rating_provisional` flag on `player_ratings` that clears after N
  matches, mirroring DUPR's asterisk convention so players who already
  know DUPR recognize the UX.

### 2.3 DUPR is a single global cross-format rating, not per-competition
DUPR "recognizes pickleball tournaments and recreational play" on one 2.0–8.0
scale and updates from *every* match a player logs, in contrast to
USA Pickleball's UTPR, which counts **only** results from USA
Pickleball–sanctioned tournaments (1.0–6.0+ scale) and does not move from
recreational/club play at all. Many competitive players carry both: DUPR
for everyday tracking, UTPR for tournament seeding eligibility.

- **Verdict: Aligned, note for later.** The spec's single internal
  `player_ratings` per user (§8) matches DUPR's "one global dynamic
  rating fed by everything" model, which is the right choice for a
  platform whose main use case is recreational/social games rather than
  sanctioned tournament seeding. Worth an explicit note that this product
  is *not* attempting to be a UTPR-equivalent tournament-sanctioning
  rating — if Competitions (§3.8) ever need external tournament-standard
  seeding, that would be a genuinely different, additional rating concept,
  not a variant of the internal one.

---

## 3. Pickleball rules/formats verification

### 3.1 §3.7's "winner-vs-winner / loser-vs-loser... king-of-the-court / Swiss style" pairing is a real, named, common format
King of the Court (aka Queen of the Court / Up-River-Down-River / Ladder /
Waterfall) is a widely used recreational rotation format: winners move up a
court, losers move down, with points typically awarded per win on the top
("king") court over a fixed session (60–90 minutes). This lines up with the
spec's description.

- **Verdict: Aligned.** §3.7's automated pairing design correctly names and
  targets a real, standard recreational format.

### 3.2 Round Robin and Ladder are two additional standard formats the spec doesn't name for competitions
Beyond King of the Court, real pickleball play/competition commonly also
uses:
- **Round robin** — every player/team plays every other exactly once, no
  eliminations, standings by win/loss or points scored; used for smaller
  club and casual tournaments where participation matters more than speed.
- **Ladder** — an ongoing, longer-running ranked list where players
  challenge opponents a few rungs above them; a win swaps ranks; updates
  span weeks/months rather than a single session (distinct from
  King-of-the-Court's single-session version of the same "climb" idea).
- **Double elimination** — the dominant bracket format at USA
  Pickleball–sanctioned events: a single loss drops a player/team to a
  losers bracket rather than eliminating them; they're out only after a
  second loss; winners- and losers-bracket winners meet in a final
  (sometimes requiring the losers-bracket finalist to beat the
  winners-bracket finalist twice).

- **Verdict: Gap.** Spec §3.8 (Competitions) says only "Organiser sets up a
  competition with a format" — it never enumerates which formats the
  platform must support, and §3.7's automated-pairing description covers
  King-of-the-Court/Swiss-style social play but not competition brackets at
  all. Given double elimination is described as "the dominant format at
  most USA Pickleball-sanctioned events," a competitions feature that can't
  produce a double-elimination bracket would be missing what players
  competitively active in pickleball expect by default. Recommend §3.8
  explicitly list the formats v1 (or a near follow-up) must support:
  **round robin, single elimination, double elimination**, at minimum,
  with King-of-the-Court/ladder treated as the *social-game* (§3.6/§3.7)
  format rather than a competition format — these are documented as
  distinct use cases in the sources (recreational rotation vs. tournament
  bracket) and the spec currently blurs them into one "matchmaking &
  scoring" section.

### 3.3 Swiss-style pairing is real but is a distinct format from King-of-the-Court, not a synonym
Sources describe King-of-the-Court/ladder-style rotation and true Swiss
tournament pairing (pair players/teams with similar running records each
round, no elimination, fixed number of rounds) as related but separate
concepts; the spec's §3.7 phrase "king-of-the-court / Swiss style" treats
them as interchangeable.

- **Verdict: Reconsider.** Minor terminology issue, but worth fixing before
  it becomes a naming inconsistency in code/proto: King-of-the-Court is a
  *court-rotation* mechanic (winners/losers physically move courts within a
  single session); Swiss is a *pairing-by-record* mechanic typically used
  across a multi-round tournament without court rotation. The spec's
  winner-vs-winner/loser-vs-loser description is closer to Swiss pairing
  logic applied within King-of-the-Court's court-movement structure — fine
  as a design, but the spec should pick one term or clearly describe it as
  a hybrid, since a builder or reviewer familiar with pickleball will read
  "king-of-the-court / Swiss style" as two different named formats mashed
  into one label.

---

## 4. Booking/scheduling edge cases from real systems

### 4.1 Store UTC, render local — and this specific project already commits to it, but the "why" is a documented recurring bug class
Scheduling-literature consensus (and the DST bug reports found) is
consistent: storing times in a local/offset format instead of UTC causes
recurring bookings to silently shift by an hour after a DST transition
(e.g., a 10am Monday recurring meeting becomes 11am on the first Monday
after clocks change), and this is cited as one of the most common failure
modes in scheduling systems. Root cause in some real bug reports: a client
sends only a UTC *offset*, not a real IANA timezone name, so the server
can't tell whether that offset should shift across a DST boundary.

- **Verdict: Aligned, strengthen the test list.** Spec §4 already says
  "timezone-correct everywhere (store UTC, render local)," which is the
  correct baseline decision. The concrete gap is in **§3.5 recurring
  hire**: the spec has no explicit note that a weekly recurrence rule must
  be defined in terms of a **facility-local wall-clock time + IANA zone
  name** (e.g., "Mondays 18:00 America/Los_Angeles"), re-resolved to UTC
  per-occurrence at generation time — not as a fixed UTC offset baked in
  once. Recommend HANDOFF.md/T5-or-later add an explicit test case:
  generate a recurring-hire booking series spanning a DST transition and
  assert the local start time stays fixed while the UTC instant shifts.
  This is exactly the bug class the sources call out as common and
  easy to miss until the first DST boundary in production.

### 4.2 Buffer time between bookings is a standard, separately-configurable rule in comparable systems — absent from this spec entirely
Skedda documents "buffer rules" as a first-class venue setting: a
configurable gap enforced *before and/or after* a booking, settable
per-space, for turnover/cleaning/changeover — distinct from, and layered on
top of, the plain no-overlap check.

- **Verdict: Gap.** Spec §6's EXCLUDE constraint only prevents *overlapping*
  bookings — it says nothing about back-to-back bookings with zero gap,
  which is fine for many uses but not universal (e.g., a facility wanting
  5 minutes to clear a court, or an owner wanting to prevent players
  camping past their slot into the next booking with no turnover). HANDOFF.md
  itself notes T0's domain tests cover "overlap logic (incl. back-to-back
  edges)" — worth confirming that means back-to-back is *currently allowed*
  (zero-gap adjacency), and if so, flagging that a configurable buffer is a
  reasonable near-future addition to `pricing_rules`-style
  per-court/facility config, not a v1 blocker.

### 4.3 Booking-window visibility (how far ahead a slot can even be seen/booked) is a standard, separate control from cancellation windows
PlayByPoint's reservation rules separately control **how far in advance
booking opens** (e.g., a court becomes visible/bookable exactly 24 hours
before the slot) from the cancellation-window rule — two different knobs
answering two different questions ("when can I book this" vs "when can I
un-book this").

- **Verdict: Gap.** Spec §3.4 lists "Cancellation windows, refunds, no-show
  handling, waitlists" but never mentions a forward booking-open window at
  all — implicitly, the spec assumes any future slot is bookable
  immediately, which is a reasonable v1 simplification but should be a
  conscious choice, not an oversight, since real competitors treat it as a
  standard configurable rule (useful for e.g. giving club/recurring-hire
  members first access before opening a slot to the general public — which
  directly matters given §3.5's club recurring-hire feature already
  implies some form of priority access).

### 4.4 Partial-court / split-court booking (see §1.5 above) is also a scheduling-edge-case, not just a facilities-modeling gap
Restated from the competitor-research section because it is exactly the
class of edge case §6's own EXCLUDE-constraint literature warns about:
an invariant defined purely on `court_id` equality cannot express a
conflict between two *different* `court_id`s that physically overlap
(a whole court vs. its two split halves). The general pattern in
interval-scheduling literature for this is a **resource hierarchy/graph**
(parent-child or overlap-group), not a flat resource ID, feeding the
exclusion check.

- **Verdict: Gap** (cross-referenced with §1.5). Not urgent for the current
  single-facility, flat-court T0–T4 slice, but should be logged as a known
  limitation before any pilot facility with shared/split courts is
  onboarded, since it's the same *category* of invisible invariant hole
  the design review already found once (F1, Booking/Game not sharing an
  invariant) and fixed at cost. Cheaper to design the `courts` parent/child
  relationship now than to retrofit the EXCLUDE constraint later.

---

## Summary of gaps by priority (for triage against HANDOFF.md backlog)

| # | Area | Verdict | Priority note |
|---|------|---------|----------------|
| 1.2 | No-show fees / account credit, not just refund | Gap | Directly extends T6 (Payments) design before it's built |
| 1.3 | Waitlist data model + auto-promotion + timeout | Gap | §8 has no waitlist entity at all; needed before §3.4 waitlist claim is true |
| 2.1 | Rating should use score margin vs. expectation, not just W/L | Gap | Affects `matches` schema in T5 follow-up (matchmaking task) |
| 2.2 | Verified (Game Admin) vs self-reported match weighting + provisional flag | Gap | Natural fit with existing Game Admin role; affects `player_ratings`/`matches` |
| 3.2 | Competitions needs named bracket formats (round robin/single/double elim) | Gap | §3.8 currently has zero named formats; affects Phase 3 competitions work |
| 1.1 | Per-facility configurable cancellation window | Gap | Needed before CancelBooking's real-world policy logic (T3 built the mechanism, not the policy) |
| 1.5 / 4.4 | Split/shared court modeling | Gap (deferred OK) | Same invariant-hole class as F1; log as known limitation |
| 4.1 | DST-safe recurring-hire generation | Aligned direction, needs explicit test | Add to T5+/recurring-hire task AC |
| 4.2 | Configurable buffer time between bookings | Gap (minor, deferrable) | Note as future `pricing_rules`-adjacent config |
| 4.3 | Booking-open window (visibility lead time) | Gap (minor, deferrable) | Note as future config, relevant once §3.5 club priority access is built |
| 1.4 | Recurring hire booking-window + entitlement passes | Reconsider | Note for later schema evolution |
| 1.6 | Multi-court single booking | Aligned | Confirm domain creates N rows per game, not 1 |
| 2.3 | Single global rating (not per-tournament) | Aligned | No action, just documented confirmation |
| 3.1 | King-of-the-Court pairing design | Aligned | No action |
| 3.3 | "King-of-the-court / Swiss style" conflates two formats | Reconsider | Terminology fix in §3.7 |

---

## Sources

1. [FAQ Players Playtomic – Playtomic Help](https://playerhelp.playtomic.com/hc/en-gb/articles/19831519179281-FAQ-Players-Playtomic)
2. [How to cancel a reservation – Playtomic Manager](https://helpmanager.playtomic.com/hc/en-gb/articles/20535256445969-How-to-cancel-a-reservation)
3. [Cancellation Policy for Open Matches – Playtomic Help](https://playerhelp.playtomic.com/hc/en-gb/articles/19831672824465-Cancellation-Policy-for-Open-Matches)
4. [How to cancel a reservation – Playtomic Help](https://playerhelp.playtomic.com/hc/en-gb/articles/19832121593873-How-to-cancel-a-reservation)
5. [How to configure "Open Matches" at your Club – Playtomic Manager](https://helpmanager.playtomic.com/hc/en-gb/articles/20535035123473-How-to-configure-Open-Matches-at-your-Club)
6. [Request a spot in an Open Match – Playtomic Manager](https://helpmanager.playtomic.com/hc/en-gb/articles/20535718052241-Request-a-spot-in-an-Open-Match)
7. [10 Features for Complex Pickleball Court Scheduling — CourtReserve](https://courtreserve.com/pickleball-court-scheduling-software-features/)
8. [Waitlist Reporting, Bypass Restriction Setting + More! — CourtReserve Release Notes](https://courtreserve.releasenotes.io/release/1eL8N-waitlist-reporting-bypass-restriction-setting-more)
9. [Auto-Refunding Cancellations: What Racquet and Paddle Clubs Need to Know — CourtReserve](https://courtreserve.com/auto-refund-cancellations-racquet-club/)
10. [Account Credit — CourtReserve Help Center](https://help.courtreserve.com/en/articles/9788173-account-credit)
11. [Court Reservations & Policies — Lifetime Activities (Santa Clara)](https://www.lifetimeactivities.com/santa-clara/court-reservations-policies/)
12. [Court Reservations & Cancelations — WaterColor Community, FL](https://www.mywatercolorcommunity.com/340/Court-Reservations-Cancelations)
13. [Rules and Policies — Pickleballerz](https://www.pickleballerzusa.com/rules-and-policies)
14. [All Reservation Rules Explained — Playbypoint Help Center](https://help.playbypoint.com/en/articles/11395832-all-reservation-rules-explained)
15. [How Player Booking Passes Work — Playbypoint Help Center](https://help.playbypoint.com/hc/en-us/articles/27706625149979-How-Player-Booking-Passes-Work)
16. [Court Reservation Basics — Playbypoint Help Center](https://help.playbypoint.com/en/articles/11412867-court-reservation-basics)
17. [Buffer time — Skedda Updates](https://updates.skedda.com/buffer-time-134373)
18. [Buffer time — Skedda Support](https://support.skedda.com/en/articles/3653032-buffer-time)
19. [Free Online Sports Court Booking System — SuperSaaS](https://www.supersaas.com/info/sports-courts-booking-system)
20. [Setting priority of Courts on what can be booked — CourtReserve Idea Board](https://feedback.courtreserve.com/communities/1/topics/192-setting-priority-of-courts-on-what-can-be-booked)
21. [Pickleball Court Booking Software — AllBooked](https://www.allbooked.com/solutions/pickleball-court-booking-software)
22. [Court Booking — SmartHealthClubs](https://smarthealthclubs.com/court-booking/)
23. [DUPR Blog | How to Get a Pickleball Rating?](https://www.dupr.com/post/how-to-get-a-pickleball-rating-2)
24. [Understanding DUPR: Dynamic Universal Pickleball Rating for Players and Managers — Playbypoint Help Center](https://help.playbypoint.com/en/articles/11024893-understanding-dupr-dynamic-universal-pickleball-rating-for-players-and-managers)
25. [DUPR Blog | Cracking the DUPR Code: How Pickleball's Rating System Shapes the Game](https://www.dupr.com/post/cracking-the-dupr-code-how-pickleballs-rating-system-shapes-the-game)
26. [DUPR Blog | Pickleball Ratings Explained: How Skill Levels Are Calculated](https://www.dupr.com/post/pickleball-ratings-explained-how-skill-levels-are-calculated)
27. [Understanding DUPR: Your Questions Answered — Pickleball.com](https://pickleball.com/blogs/understanding-dupr-your-questions-answered-1)
28. [How the DUPR Rating Algorithm Works — Pickleheads](https://www.pickleheads.com/guides/how-dupr-works)
29. [DUPR Overview — Pictona](https://pictona.org/play/dupr/what-is-dupr/)
30. [DUPR Pickleball Rating Explained: The Ultimate Beginner's Guide — TeachMe.To](https://teachme.to/blog/how-does-dupr-rating-work)
31. [Understanding The Pickleball DUPR Rating System — Pickleland](https://pickleland.com/pickleball-dupr-score/)
32. [DUPR Rating Explained: How Pickleball Ratings Work — Rosterlytic](https://rosterlytic.com/sideline/explainers/dupr-rating-explained)
33. [DUPR Blog | DUPR Integration, Integrity, and Info: Your Questions Answered](https://www.dupr.com/post/upa-integration-and-dupr-algorithm-faqs---all-you-need-to-know)
34. [Pickleball DUPR Rating: How It Works & How to Get Yours — The Dink Pickleball](https://www.thedinkpickleball.com/pickleball-dupr-rating-how-it-works-how-to-get-yours/)
35. [Pickleball Player Rating — USA Pickleball](https://usapickleball.org/skill-level/ratings/)
36. [Understanding DUPR: Your Questions Answered — PickleballTournaments.com](https://pickleballtournaments.com/blog/understanding-dupr-your-questions-answered)
37. [Pickleball518 + DUPR FAQ](https://www.pickleball518.com/pickleball518-dupr-faq/)
38. [Pickleball Rating Guide: Find Your Skill Level in 3 Steps — Paddletek](https://www.paddletek.com/blogs/news/pickleball-ratings-guide)
39. [UTPR vs. DUPR: Decisions for Pickleball Tournament Owners — Sports Destination Management](https://www.sportsdestinations.com/sports/pickleball/utpr-vs-dupr-decisions-pickleball-tournament-33556)
40. [DUPR Blog | Understanding All Pickleball Ratings](https://www.dupr.com/post/understanding-all-pickleball-ratings)
41. [DUPR Pickleball Rating Converter — Bounce](https://www.bounce.game/pickleball-rating-converter)
42. [Pickleball Ratings Explained: DUPR vs UTPR — Helios](https://heliospickleball.com/blogs/news/pickleball-rating-system-explained)
43. [UTPR Rating Explained — Pickleball Athletic Club](https://playatpac.com/utpr-rating/)
44. [Pickleball Round Robin: Schedules, Scoring & Free Live Leaderboard — Leaderboarded](https://leaderboarded.com/blog/posts/pickleball-round-robin/)
45. [Men's King of the Court — Pickleball Tournament Rules & Details — Global Pickleball Network](https://www.globalpickleball.network/pickleball-tournaments/tournaments/pickleball-tournament-page/details/296-men-s-king-of-the-court)
46. [Types of pickleball rec play: Round Robin, King of the Court, open play, and more — PlayPickleball](https://www.playpickleball.com/types-of-pickleball-rec-play/)
47. [Pickleball Tournament Formats — Round Robin, Double Elim & More — PickleBoard](https://pickleboard.co/guides/pickleball-tournament-formats)
48. [Pickleball rotation formats: round robin, king of the court, and fair play for groups — Pickleball Court Scheduler](https://pickleballcourtscheduler.com/wiki/organizing/rotation-formats-explained/)
49. [Round Robin Pickleball: Master The Most Social Tournament Format — PlayAtPAC](https://playatpac.com/round-robin-pickleball-guide/)
50. [Pickleball Brackets Explained: Understand Tournament Rules — The Dink Pickleball](https://www.thedinkpickleball.com/pickleball-brackets-explained-understand-tournament-rules/)
51. [What Are the Different Pickleball Tournament Formats? — Pickleball.com](https://pickleball.com/docs/en/article/what-are-the-different-pickleball-tournament-formats)
52. [Pickleball Brackets: Complete Guide — PaceCourt](https://pacecourt.com/pickleball-brackets-complete-guide-types-rules-formats-how-to-create-a-winning-tournament-bracket/)
53. [Types of Pickleball Tournaments — The Picklr](https://thepicklr.com/types-of-pickleball-tournaments/)
54. [Pickleball Tournament Formats Explained: Round Robin, Double Elimination, and Pool Play — DinkVision](https://dinkvision.com/pickleball-tournament-formats-bracket-types/)
55. [Double-elimination tournament — Wikipedia](https://en.wikipedia.org/wiki/Double-elimination_tournament)
56. [Pickleball Tournament Formats: The Simple, Complete Guide — Regan Family Pickleball](https://reganfamilypickleball.com/pickleball-tournament-formats/)
57. [How Pickleball Double Elimination Tournaments Work? — Ramsports](https://ramsports.com/blogs/news/how-pickleball-double-elimination-tournaments-work)
58. [Scheduling API comparison guide — Nylas](https://www.nylas.com/blog/scheduling-api-comparison-guide/)
59. [Daylight saving on interval trigger · Issue #372 — agronholm/apscheduler (GitHub)](https://github.com/agronholm/apscheduler/issues/372)
60. [Troubleshooting: schedules with Daylight saving time (DST) — Talend/Qlik](https://help.qlik.com/talend/en-US/management-console-user-guide/Cloud/scheduling-with-dst-daylight-saving-time)
61. [Scheduler Times off by 1 Hour with Daylight Saving Time Adjusted — Patterson Support](https://pattersonsupport.custhelp.com/app/answers/detail/a_id/22216/~/scheduler-times-off-by-1-hour-with-daylight-saving-time-adjusted)

*(Additional pages returned 403 to automated fetch — e.g. dupr.com's own
long-form blog posts and the CourtReserve booking-settings help article —
and are cited above via the search-result excerpts that were retrievable;
where a claim rests on a search snippet rather than a fully fetched page,
it is corroborated by at least one other independent source in the list
above.)*
