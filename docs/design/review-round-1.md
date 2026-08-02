# Design Review — Round 1

**Reviewing:** `docs/design/pickleball-system-design-v1.md` (round 0) + its
companion visual artifact (host facility/pricing console, host game-creation
flow, player discovery/join/registration on iPhone, iPad adaptive layout).
**Roles played:** UX/UI Designer, Product Manager, Principal Engineer,
Product Owner — per the four dossiers in `docs/roles/` +
`docs/agent-operating-handbook.md` Part B. Per this project's standing rule,
disagreements are recorded, not resolved into false consensus.
**Round:** 1 of up to 10.

---

## 1. Answers to round-0 §7's seven open questions

Each answer is this round's genuine recommendation, not a restatement of the
question. Full reasoning below the table; disagreements are called out where
they're real, not implied by silence.

| # | Question | Round-1 answer |
|---|---|---|
| 1 | Security camera: link vs. footage? | **Store a reference URL only** (round-0's default), *plus* a new requirement round-0 missed: a facility-onboarding consent/signage attestation — see §2. |
| 2 | Discount: replace or modify price? | **Modify** (percent/fixed-off the resolved pricing-rule band) — confirmed. But the mockup depicts a compound end-condition round-0's domain model doesn't support; fix one or the other (§3). |
| 3 | Capacity: per-game or per-court? | **Per-Game total**, matching the already-built `Game.Capacity int` and what the mockup itself depicts (16 total across 2 courts, not 8+8). |
| 4 | Friend/plus-one changes the capacity invariant — proceed? | **Yes, proceed**, but the fix is *not* the one-line `count(*)`→`sum(1+guest_count)` swap round-0 described — see §4, a materially bigger PE finding grounded in the actual trigger code. |
| 5 | Social-reply-to-register — accept the in-app-RSVP alternative? | **Yes**, confirmed against the actual spec text (§7 of `pickleball-platform-spec.md`) — round-0's characterization is accurate, not paraphrased loosely. |
| 6 | "Level" from tenure+wins — reopen D2 or restate it? | **Reading 3 (both)** — a simple public-facing "Level" label backed by the existing Elo-style internal rating. This is additive to D2, not a reopening; the mockup already draws it this way. |
| 7 | Gender as a matchmaking input — additive design OK? | **Yes**, additive and confirmed — with one addition: it must be self-reported, optional, and have a "prefer not to say" path, same as skill level's cold-start pattern, given gender is more sensitive profile data than a skill number. |

### On Q1 — security camera
All four roles converge on round-0's default (store a reference URL, embed
the vendor's own player, never host footage) — it's the cheap, correct
reading of "managed services, not your own plumbing" (spec §1) and avoids
opening a real video-storage/GDPR-footage scope round-0 already flagged as
the expensive alternative. **What round-0 missed**: even the cheap version
puts a live camera feed of real people at a physical venue one click away
inside the app. See §2 for the new finding.

### On Q2 — discount replace vs. modify
Uncontroversial — percent/fixed-off the resolved band price is the only
reading consistent with ADR-0002's precedent (ambiguity is a domain error,
not silently prioritized) and with how every comparable platform in
`research-functional.md` models promotions. The artifact, however, shows a
discount ending on **whichever of two conditions comes first** ("ends after
10 bookings, or Sep 30"), which round-0's `EndCondition` variant
(`EndAfterDate | EndAfterOccurrences | NoEnd`, mutually exclusive) cannot
express. PE recommends narrowing the mockup to a single end-condition per
rule for v1 (cheaper, and nothing in the original requirements asked for a
compound condition) rather than adding a fourth variant to the domain model
to match the mockup. PM has no objection — a compound end-condition is a
nice-to-have, not a differentiator.

### On Q3 — capacity granularity
Per-Game, matching the shipped `Game.Capacity int` (T5.1, on the sprint
branches — see §4's caveat about merge status) and the mockup's own numbers.
PM adds: this is also the right user mental model — a host thinks "how many
players fit in my session," not "how many fit on court 2 specifically."

### On Q4 — friend/plus-one and the capacity invariant
See §4 below — this is this round's most substantial finding, not a one-line
confirmation.

### On Q5 — social-reply-to-register
Confirmed accurate. Cross-checked round-0's summary against
`docs/pickleball-platform-spec.md` §7 verbatim: the spec already rejects
auto-monitoring public replies for the same two reasons round-0 cites (ToS
restrictions per-platform, and free-text-reply parsing being unreliable
compared to structured in-app RSVP), and already proposes the same
alternative (shareable link → in-app RSVP, optional bot on a
platform-owned channel parsing a constrained format like `IN 2`). Round-0
did not misquote or soften this — it is a faithful restatement, and the
artifact's "we don't read replies posted on these platforms" copy on the
game-creation screen is the right user-facing framing of the resolution.
No disagreement across roles here.

### On Q6 — skill level
The mockup already draws the resolution: a player-facing "Level 3.7 · 27
games played · 18 wins · trending up" card, with copy noting the level
"moves with score margin, not just wins — a close loss to a strong pair
moves it less than a blowout loss to an even one." That is reading 3 from
round-0's three options (a simple public label backed by an internal
Elo-style rating) — and it happens to also resolve `research-functional.md`
P1 #10 (rating should weight score margin vs. expectation, not just W/L) in
the same stroke, which round-0 treated as a separate, deferred finding.
**Recommendation: fold P1 #10 into whichever ticket builds this**, since the
mockup already commits to the margin-weighted design — don't let it drift
back to a plain win-count formula at implementation time. This is additive
to D2 (automated from history, self-reported cold-start, manual override
intact); nothing here reopens the locked decision, and PO confirms no
user sign-off is needed to proceed on this reading.

### On Q7 — gender in matchmaking
Additive, confirmed, matches a real pickleball format (mixed doubles).
Designer's addition: model it with the same self-reported,
optional-at-signup pattern as skill level (spec §3.1), with an explicit
"prefer not to say" option — games that don't request gender-balanced
pairing should simply never read the field. This isn't optional politeness;
`research-security-compliance.md` §4's GDPR minimization finding applies
here as directly as it does to geolocation: collect it only where a game
actually uses it, and don't make it a signup blocker.

---

## 2. New finding — camera-link consent/signage isn't modeled anywhere

Round-0's Q1 already asks the right cheap-vs-expensive question about
*storage* (URL reference vs. hosted footage) and gets the right answer. But
neither round-0 nor the four requirements-research docs asks the adjacent
*consent* question: a security-camera link is a reference to a live or
recent video feed of **real, identifiable people at a physical venue** —
players, not just courts. Storing a URL instead of footage sidesteps the
platform's own data-processing/storage obligations, but it does **not**
sidestep the facility operator's obligation to have lawful basis and
adequate notice (typically signage, sometimes explicit consent depending on
jurisdiction) for recording people who show up to play. `ico.org.uk`'s
geolocation guidance (already cited in `research-security-compliance.md`
§4) makes the general point this generalizes from: a data type doesn't need
to be hosted by *this* platform to be a privacy concern the platform's own
onboarding flow should account for, if the platform is the thing surfacing
a link to it.

**Concrete, low-cost recommendation:** add a required attestation checkbox
at facility onboarding ("I confirm this facility has appropriate signage/
consent for any security cameras linked here") alongside the camera-URL
field — a product/legal decision (PO territory, not an engineering one),
cheap to add now, expensive to retrofit once camera links are live across
many facilities. This does not change round-0's "store a URL" resolution to
Q1; it's an addition, not a reopening.

Separately: the mockup places the camera field only on the **host-facing**
facility-settings screen, with no player-facing surface shown. That's the
right default (don't expose a raw stream URL to end users) — worth stating
explicitly as a requirement rather than leaving it implicit in "that's just
what the mockup happened to show."

---

## 3. Grounded finding — the capacity-guard trigger fix is bigger than round-0 said (Q4, detailed)

Round-0 §3.3 states the fix as: *"the trigger's `count(*)` needs to become
`sum(1 + guest_count)`"* — implying a one-line aggregate-function swap. This
round read the actual code, per the task brief's instruction, rather than
trusting that description. **Important caveat first: this code is not on
the branch this review's `HEAD` is checked out on.** `internal/socialplay/`
and `db/migrations/0006_socialplay_capacity_guard.sql` exist only on
unmerged sprint branches (`sprint/t5.4-postgres-proto-grpc`,
`t54-loop2-work`, and related worktrees) — `HANDOFF.md`'s own "Not yet
built" list on this branch still reads "Social Play, Payments, Facilities,
Competitions, Statements contexts," and there is no merge commit bringing
T5 into `claude/go-backend-pickleball-7up34j` or `master`. Round-0's framing
("the just-shipped, freshly-reviewed capacity-guard trigger") describes
sprint-branch work as if it's already landed on the shared branch, which
per CLAUDE.md rule 9 it isn't yet. This may simply be a sequencing/timing
gap (the merge is presumably coming), but it's worth naming plainly so
nobody schedules a T7 ticket against a trigger that turns out to still be
mid-review on PR #14's follow-on branches when the ticket starts.

With that caveat, here's what the actual trigger
(`enforce_game_capacity()`, on the sprint branch) does that round-0's
one-line description misses:

```sql
IF TG_OP = 'UPDATE' AND OLD.status <> 'cancelled' THEN
    RETURN NEW;  -- skip the capacity check entirely
END IF;
...
SELECT count(*) INTO active_count
FROM registrations
WHERE game_id = NEW.game_id AND status <> 'cancelled' AND id <> NEW.id;
IF active_count >= game_capacity THEN RAISE EXCEPTION ...
```

The trigger only re-checks capacity on **INSERT** or a **cancelled→active
UPDATE**. An UPDATE that leaves `status` unchanged (e.g. a payment-status
flip) is explicitly skipped, on purpose — that's correct today because
nothing else about a registration can change its seat consumption after the
fact.

**Once `guest_count` exists, that stops being true if guest_count is
mutable after registration.** A Player who registers alone and later edits
their registration to add a friend ("bringing 2 guests now, up from 0") is
exactly the `TG_OP = 'UPDATE' AND OLD.status <> 'cancelled'` case the
trigger currently, deliberately, skips — so a naive `count(*)` →
`sum(1+guest_count)` swap on the existing branches would **not** catch a
capacity overrun caused by an existing registrant increasing their own
guest count, only overruns from new registrations. That's a real, silent
gap in the invariant, not a hypothetical — it's the same shape of bug this
project's own T5.4-loop-2 fix was created to close (an UPDATE path the
original design didn't consider).

**Recommendation, framed as a decision PO should make now, not an
engineering detail to improvise later:** make `GuestCount` **immutable
after registration creation** (change-of-mind means cancel and
re-register, same UX pattern already used for booking changes elsewhere in
this domain) — this keeps the trigger fix to the swap round-0 described,
because the only paths that ever consume new seats stay INSERT and
cancelled→active UPDATE, both already guarded. If the product actually
wants in-place guest-count edits, say so explicitly, because it changes
the trigger's UPDATE-branching logic, not just its aggregate function, and
should be pointed as a materially bigger ticket than round-0's framing
implies.

**Disagreement recorded (Designer vs. Principal Engineer), in the format
this project already uses (see `docs/process/t5-sprint-plan.md`'s kickoff
note):**
- **Designer's position:** the roster view needs to show a seats/
  registrations breakdown explicitly ("12/16 seats · 9 registrations · 3
  guests"), not just a bare ratio, both for NN/g heuristic #6
  (recognition over recall — a host shouldn't have to infer from
  scanning "+1 guest" tags) and because a screen-reader user gets no
  equivalent of the visual "+1 guest" tag scan unless the breakdown is
  explicit, labeled text. This should be written into acceptance criteria
  now, not left to a client-side afterthought, per the dossier's standing
  warning against "accessibility as an afterthought."
- **Principal Engineer's position:** agrees the breakdown is cheap to
  compute client-side from `GuestCount` per registration and doesn't
  require new backend fields (no `total_seats`/`total_guests` computed
  columns needed) — but pushes back on treating this as an API-shape
  decision. The backend's job is to expose `GuestCount` per registration
  and enforce the invariant; the breakdown text is a presentation concern
  that belongs in the ticket's UI acceptance criteria, not its domain/API
  acceptance criteria, and shouldn't gate T7's backend scope.
- **Resolution:** not full consensus, but a scoped one — both agree the
  breakdown belongs in T7's UI acceptance criteria explicitly (Designer's
  ask), and both agree it requires no new backend fields beyond
  `GuestCount` (PE's ask). Recorded, not silently merged into agreement,
  because the underlying tension (when does a display concern get to
  block ticket scope) will recur.

---

## 4. Other UI/UX findings against the artifact

- **Roster capacity pill is ambiguous.** The tablet/iPad roster shows a bare
  `12/16` pill with no unit. The review-only caption block explaining "12/16
  seats, not registrations" is not part of the actual product UI — a real
  host or player sees only `12/16` next to rows that say "+1 guest," which
  is suggestive but not explicit. Recommend the pill itself read `12/16
  seats` (or carry an accessible label to that effect), not rely on the
  reader inferring the unit from adjacent rows.
- **Status pills likely fail contrast at their current size.** The pill CSS
  uses `color-mix(in srgb, var(--success) 18%, transparent)` as background
  with the same hue at full saturation as the text color, at `0.64rem`
  (~10px) — same-hue tint-on-saturated-text pairings are a common way to
  quietly fail WCAG 1.4.3 (4.5:1 for text this small) even though they read
  as "on-brand." This needs an actual contrast-ratio check before this
  pattern becomes the shared badge component (per the Designer dossier's
  own §3.2 recommendation to define this once, not per-screen) — flag as a
  build-time check, not a blocker to the design direction.
- **Toggle switches are under every applicable target-size floor.** The
  mockup's `.toggle` is 34×19 CSS px. WCAG 2.5.8's AA floor is 24×24; both
  Apple HIG (44×44pt) and Material (48×48dp) are stricter still. 19px
  height fails all three. This is a real, fixable spacing issue (add
  padding/hit-area, not just the visible track), not a fundamental design
  problem — flag for the component-token pass before three clients
  (Vue/Swift/Kotlin) each independently reimplement a too-small toggle.
- **Color is not the sole signal for paid/unpaid/full/waitlist status** —
  every pill in the mockup carries a text label ("Paid," "Unpaid," "Full,"
  "Waitlist #1," "Paid — cash"), satisfying WCAG 1.4.1. No finding here;
  called out because it's exactly the anti-pattern the Designer dossier
  warns about, and the artifact avoids it correctly.
- **Cancellation-policy inheritance is shown clearly.** Both the host
  game-creation screen ("Facility's default policy — hosts can tighten,
  not loosen") and the player registration screen ("Cancel free until Wed
  6:00pm (4hrs before)") surface the policy and its source in plain text
  at the point of decision. No finding — this is a good pattern, worth
  preserving as the ticket gets built (the hint text is doing real
  work, not decoration).
- **Discount-pricing card communicates the end-condition well**, modulo the
  compound-condition mismatch already noted in §1/Q2 — the struck-through
  original price, percent-off badge, and plain-language condition line
  ("Weekday off-peak (2–5pm) · ends after 10 bookings...") together satisfy
  NN/g's error-prevention-by-clarity standard better than most real
  competitor promo UIs reviewed in `research-functional.md`.
- **"Verify link" button on the camera field is a borderline target size**
  (`padding: 4px 10px` at `0.72rem`) — likely just under the 24×24 CSS px
  WCAG floor depending on line-height. Minor, but same category as the
  toggle finding — worth a pass across the whole component set rather than
  fixing this one instance.

---

## 5. Round 1 verdict

**Proceeds to round 2.** No part of this design direction requires
reopening a `CLAUDE.md` "do NOT reopen" locked decision — Q6 (skill level)
was the one candidate for that, and this round's answer (reading 3,
additive to D2) is consistent with all four roles' read of the mockup and
the existing research, not a reopening.

Two items are **product/legal decisions, not design-review decisions**, and
should go to the user (matching the Product Owner dossier's own
"escalates when... requires a product/legal decision... explicitly out of
scope for an engineering session" standard) before they're turned into
ticket acceptance criteria, though neither blocks round 2 from proceeding:

1. **Is `Registration.GuestCount` mutable after registration, or
   set-once?** (§3) — this materially changes T7's scope and story
   points; recommend immutable, but it's the user's call.
2. **Facility camera-link consent attestation wording** (§2) — cheap to
   add, but the actual attestation copy/requirement is a legal-adjacent
   product decision, not something this review should draft unilaterally.

One process item worth surfacing before more work stacks on top: **confirm
T5's actual merge status** into the shared branch before scheduling any
ticket that edits `0006_socialplay_capacity_guard.sql` — this review had to
read that file off an unmerged sprint branch, not `HEAD`, to do the grounded
analysis in §3, and round-0's own framing describes it as already shipped.

No fundamental disagreement blocks progress; the two recorded disagreements
(§3) are scoped and resolved enough to carry into round 2's ticket-writing
without re-litigation.
