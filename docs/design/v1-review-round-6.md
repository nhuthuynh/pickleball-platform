# Design Review — Round 6

**Reviewing:** round 5's assigned fix (competition screen-caption's "a bracket
on top" clause) against the companion visual artifact (same URL, republished
— `https://claude.ai/code/artifact/bcfb62bc-d157-44dc-95da-5166f4480ab9`),
verified against the artifact's actual HTML/CSS source (fetched in full —
53KB, every `<style>` rule and every line of `<body>` markup read end to end,
not inferred from the artifact's own "Round 1–6" changelog callout, per the
task's standing instruction), plus a genuinely fresh, from-scratch sweep of
every `<p>`, `.hint`, `.cond`, and `.screen-caption` across all six flows
(not a re-run of round 5's checklist), plus a deliberately different
big-picture question about the artifact as a whole after six rounds of edits.
**Roles played:** UX/UI Designer, Product Manager, Principal Engineer,
Product Owner — same four dossiers as rounds 1–5.
**Round:** 6 of up to 10.

---

## 1. Verdict on round 5's assigned fix

**PASS.** The Flow-4 (competition) screen-caption no longer says "a bracket
on top." It now reads:

```html
<p class="screen-caption">
  Reuses the game-creation screen's advertising pattern exactly (same
  shareable-link, in-app-RSVP model, same §7.05 resolution) rather than
  inventing a second one — a competition is a Booking/Registration-shaped
  thing with a format (round robin, ladder, or a true bracket, depending
  on what the host picks) layered on top, not a different kind of
  advertising problem.
</p>
```

A `grep`-equivalent check of the full source confirms the word "bracket"
appears exactly once in the entire artifact, in this corrected form, inside
a list that also includes "round robin" and "ladder" as sibling options.

**On whether this is a genuinely good fix or technically-true-but-evasive
generality (task item 1's follow-up question):** it's a genuinely good fix,
not an evasion. A purely evasive fix would have dropped all format words and
said something contentless like "a structure layered on top." Instead the
new sentence explicitly names round robin — the exact format this screen's
own "Format" field shows ("Round robin, 4 pools of 4") — as one of three
concrete, named options, correctly acknowledging that *this* screen depicts
round robin while the sentence itself is making a general claim about the
`Competition` aggregate (which can also be a ladder or a true bracket
depending on what a future host picks). That's the right level of
generality: general where the claim is genuinely about the aggregate, and
specific enough that a reader isn't left wondering what the sentence means.
No finding here.

---

## 2. Fresh, independent sweep — three new findings, none a repeat of rounds 2/4/5's pattern

Read every `<p>`, `.hint`, `.cond`, and `.screen-caption` in the full
53KB source end to end (not the changelog's own description of what
changed), then cross-referenced every factual/numeric claim against the
structured UI elements on the same screen and on other screens describing
the same entity — starting from zero assumptions about what was likely to
still be broken, per the task's instruction not to reuse round 5's specific
checklist. This found three real, independently verifiable inconsistencies.
None of them is the "component gets fixed, an old caption elsewhere still
asserts the pre-fix claim" shape rounds 2/4/5 kept finding — these are new
failure shapes, found by literally checking the numbers instead of the prose.

### 2.1 — The footer still claims this is "round 1," contradicting the masthead's "Round 6"

The masthead status line (top of page) reads:

```html
<div class="status">... Round 6 of up to 10 — draft, not implemented</div>
```

The footer, unchanged since the artifact's first publish, reads:

```html
<p>
  This artifact and its companion doc (...) are round 1 of an iterative
  review — Designer, PM, Principal Engineer, and PO react to what's here,
  findings get folded in, and the next round republishes to this same page.
</p>
```

The footer describes the artifact as "round 1" while the masthead — and the
entire intro changelog directly above it — describes it as round 6. This is
the exact failure class the task asked to hunt for: a factual claim ("what
round is this") stated in two places, one updated five times over, one never
touched. It's low-stakes (it doesn't affect the actual design content under
review), but it is a genuine, unambiguous, `grep`-verifiable contradiction
that survived all five prior rounds' sweeps, because none of those sweeps
checked the footer — they were checking the screens.

### 2.2 — The player-facing cancellation cutoff doesn't match the game's own start time or the host's own stated policy

Three data points, all in the same flow, describing the same game
("Sat mixed doubles ladder," Riverside):

- Flow 2 (host console): `Date & time: Sat, Aug 8 · 4:00–6:00pm`, and
  `Cancel without charge until: 4 hours before start` (hint: "Facility's
  default policy — hosts can tighten, not loosen").
- Flow 3 (iPhone, Register screen, same game by name and facility):
  `Cancel free until Wed 6:00pm (4hrs before)`.

"4 hours before" a Saturday 4:00pm start is Saturday 12:00pm (noon) — not
Wednesday 6:00pm, which is roughly **70 hours** before the start, under any
reading of which Wednesday/Saturday pair is meant. The parenthetical
`(4hrs before)` is a numeric claim about the relationship between "Wed
6:00pm" and the game's start time, and it's false against every other
number on the page describing this same game and this same policy.

This is not a new bug introduced this round — this exact sentence is quoted
verbatim in round 1's §4 as an example of the artifact doing cancellation
policy *well* ("the hint text is doing real work, not decoration"). Every
round since has praised or ignored it without checking the arithmetic. It's
a real finding regardless of when it was introduced, per the task's framing
that this sweep should be genuinely independent of prior rounds' specific
checklists — and it's squarely inside item 2(b)'s explicit ask ("do the...
dates quoted in prose match the structured UI elements on the same screen").

### 2.3 — The same game's level requirement is described two different ways on iPhone vs. iPad

- Flow 3 (iPhone, "Find a game" list card, `.cond` text): `Riverside Courts
  2 & 3 · 4:00–6:00pm · Level 3.5+ · Gender-balanced pairs` — an
  **open-ended** floor, no stated ceiling.
- iPad adaptive flow (dedicated `.field` labeled "Level range" in the
  detail panel for the same game, identified by the same app-title "Sat
  mixed doubles ladder"): `3.5–4.0` — a **closed range** with an explicit
  4.0 ceiling.

These aren't two views of different information (like the iPad showing
"Format: King of the court" that the iPhone card simply omits, which is
fine — less detail, not conflicting detail). "3.5+" and "3.5–4.0" are
different, incompatible claims about who's eligible: under the iPhone's
wording a 4.3-level player qualifies; under the iPad's, they don't. This is
the same game, named identically, at the same facility, same time, same
"gender-balanced pairs" descriptor on both screens — only the level
eligibility disagrees. (Checked and ruled out a false lead first: the
iPhone screen also shows a *search filter* reading "Level 3.5–4.0" above
the list — that's the player's own search filter, not the game's property,
and comparing a filter to a game property would have been the wrong
comparison. The actual game-property fields, `.cond` on iPhone vs. `.field`
on iPad, are what disagree.)

**No other inconsistency found.** Capacity ("12/16 seats") is consistent
everywhere it recurs (tablet roster pill, roster caption-block, iPhone
discovery card, iPad condensed list) — re-checked fresh, not assumed from
round 5. Discount math checks out (`$18 × 0.75 = $13.50`, `$18 × 0.80 =
$14.40`). All seven `§7.0N` cross-references in screen-captions and hints
point to the matching open question. The pool-of-4 seeding (3 named entries
+ 1 open slot) still matches "4 pools of 4." Considered and rejected as
non-findings: the roster rows shown (5 of "12/16") don't individually sum
to 12 — this is an ordinary illustrative-sample truncation, not a claim of
completeness, the same reasoning round 1 already recorded as a
non-blocking Designer/PE disagreement, not a new defect. The competition's
proper-noun name "Riverside Summer Ladder" alongside a "Round robin" format
field was considered and set aside — "ladder" as an event brand name
distinct from its technical format is a normal real-world convention (the
same way a "ladder league" is commonly organized as round-robin play), and
two full rounds of scrutiny on this exact screen (4 and 5) never flagged it,
which is weak evidence it isn't actually confusing in context.

---

## 3. Stepping back — does the artifact as a whole still work, after 6 rounds?

This is deliberately not a line-item table. Honest answer, checked against
what's actually on the page rather than assumed:

**Partially eroded, in one concrete and fixable way — the front matter, not
the screens.** The six flows themselves are still coherent: consistent
tokens, consistent component patterns (pills, buttons, toggles all
converged to one shared treatment by round 3–4), a believable narrative
(the same illustrative players — Mara T., Devon K., Priya S. — reused
sensibly across host, player, and competition views, which round 5 already
confirmed is intentional reuse, not drift). A Designer/PM/PE/PO team could
still open this and react to the actual design. That part hasn't degraded.

What *has* degraded is the reading path before you reach any of that. The
intro `caption-block` is now roughly 300 words of round-by-round changelog
prose — five rounds' worth of "round 2 caught that round 1's fix was only
verified in dark theme... round 3 replaces the fixed hex with theme-aware
tokens... round 4... round 5... round 6..." — sitting between the hero
title and the actual "seven things to settle" section a reviewer is
supposed to react to. This is exactly the failure mode the Designer dossier
warns about under NN/g heuristic #8 ("every extra unit of information
competes with the relevant units and diminishes their visibility") and
under *Refactoring UI*'s "most interface problems are hierarchy problems"
— the changelog has grown, edit by edit, into the single largest block of
text above the fold, ahead of the content it's a changelog *for*.

**One concrete instance where this stopped being cosmetic and started
replacing actual content:** every flow's subtitle (the `<p>` under its
`<h2>`) describes what the screen is *for*, e.g. Flow 1's "Desktop-first:
an owner adding a facility, a camera reference, and a discount rule with an
end condition." Flow 4's subtitle is the one exception:

```html
<p>Requirement #9 — added in round 4; the round-1 through round-3 mockups
never actually included this screen despite the intro copy claiming it was
covered.</p>
```

This is entirely review-history narration — it says nothing about what a
competition-setup screen does or why it's designed this way, unlike every
other flow's subtitle. A reader hitting Flow 4 without having read the prior
rounds' docs gets no design description at all here, only an audit trail.
That's a real regression in the artifact's usefulness *as a design sketch*,
caused directly by the iteration process itself, not by any single round's
fix being wrong.

**Honest verdict:** no, the iteration process has not broken the artifact,
but it has measurably crowded it, and crowded it asymmetrically — the
screens are still clean, the surrounding narration about the screens is not.
Recommend, before round 7 or whenever this stops being actively edited:
move the round-by-round changelog out of the primary reading path (a
collapsed/appendix block, or purely into the `review-round-N.md` files
where it already lives in full detail — the artifact doesn't need to
duplicate it), and rewrite Flow 4's subtitle to describe the competition
screen the way every other flow's subtitle describes its screen. Neither
fix is expensive; both are overdue given how long the changelog has been
allowed to grow unchecked.

---

## 4. Round 6 verdict

**Not converged — proceeds to round 7.** Item 1's assigned check passes
cleanly, and it's a good fix on its own merits, not evasive generality. But
§2's genuinely fresh sweep — deliberately not a re-run of round 5's
checklist — found three real, independently verifiable inconsistencies
that no prior round caught: the footer's stale "round 1" self-description
contradicting the masthead's "Round 6," the player-facing cancellation
cutoff's arithmetic not matching the game's own start time or the host's
own stated policy, and the same game's level-eligibility being described
two incompatible ways on iPhone vs. iPad. None of these is the narrow
"fix landed on one surface, stale text survived on another" shape rounds
2/4/5 kept finding — they're a different failure mode (numbers and dates
that were simply never cross-checked against each other), which is exactly
what a genuinely independent sweep, rather than a repeat of the same
checklist, is supposed to surface.

Per the convergence rule ("two consecutive rounds find nothing material"):
round 6 is material. **This makes zero consecutive clean rounds, not
one** — the counter resets to zero exactly as it has every round since
round 3 first found something. Round 7 would need to find nothing material,
and round 8 after it, before this converges.

§3's answer is additive, not blocking: the artifact's core design content
is intact, but the accumulated changelog and one instance of review-history
narration displacing actual screen description are real, worth fixing
before this goes much further, and are the kind of thing that will keep
getting worse, not better, with more rounds of "add another changelog
paragraph" fixes — worth a structural fix (move the changelog out of the
primary path) rather than another line-edit.

Unchanged from rounds 1–5, still independently open regardless of this
round's findings: **`Registration.GuestCount` mutability** and **the
camera-consent attestation wording** remain explicit product/legal
decisions pending the user's direct sign-off — not resolved by any design
round, and not blocking round 7 from proceeding on the same
working-assumption basis established in round 1.
