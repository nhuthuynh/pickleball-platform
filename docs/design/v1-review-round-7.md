# Design Review — Round 7

**Reviewing:** round 6's four assigned fixes (footer's stale "round 1"
self-description; the cancellation-cutoff arithmetic mismatch; the
iPhone-vs-iPad level-range disagreement; the ~300-word changelog crowding the
primary reading path, including Flow 4's subtitle being replaced by
review-history narration) against the companion visual artifact (same URL,
republished — `https://claude.ai/code/artifact/bcfb62bc-d157-44dc-95da-5166f4480ab9`),
verified against the artifact's actual HTML/CSS source (fetched in full —
791 lines saved to disk and grepped for every load-bearing string, not
inferred from the artifact's own changelog callout, per the task's standing
instruction), plus an independent, non-deferential judgment on whether the
round-7 trim actually improved the artifact as a stand-alone design sketch,
plus a fresh sweep for the same recurring bug class rounds 2/4/5/6 each
found, specifically re-checking whether the *trimmed* changelog's own claim
about what round 7 fixed is itself accurate.
**Roles played:** UX/UI Designer, Product Manager, Principal Engineer,
Product Owner — same four dossiers as rounds 1–6.
**Round:** 7 of up to 10.

---

## 1. Verdicts on the four assigned checks

| # | Check | Verdict | Reason |
|---|---|---|---|
| 1 | Footer reflects an ongoing multi-round review, consistent with the masthead | **PASS** | The footer no longer states any round number at all. It now reads: "This artifact and its companion doc ... are under iterative review — Designer, PM, Principal Engineer, and PO react to what's here each round, findings get folded in, and the next round republishes to this same page. See the masthead for the current round and `docs/design/review-round-*.md` for full history." `grep -n "round 1"` against the full 791-line source returns **zero matches** anywhere in the document. This is a stronger fix than a one-line number update: by deferring to "the masthead" instead of restating a round number, the footer structurally cannot drift out of sync with the masthead again — there is now exactly one place in the artifact that states the current round number, not two that have to be kept in lockstep by hand. |
| 2 | Cancellation cutoff matches the game's stated start time | **PASS** | Flow 3's Register screen (`Sat mixed doubles ladder`) now reads: `Cancel free until` → `Sat 12:00pm (4hrs before the 4:00pm start)`. The game's own `Date & time` field (Flow 2, host console, same game) is `Sat, Aug 8 · 4:00–6:00pm`. 4 hours before a 4:00pm Saturday start is 12:00pm the same day — the arithmetic now holds, and the parenthetical states its own basis explicitly ("4hrs before the 4:00pm start") rather than leaving the reader to compute it. Flow 2's own cancellation field (`Cancel without charge until: 4 hours before start`) is unchanged and still correctly general (a policy statement, not a date), so there's nothing there to conflict with the now-corrected specific instance. |
| 3 | Level range for the mixed-doubles ladder game reads "3.5–4.0" everywhere | **PASS** | Checked all three surfaces that state this game's level eligibility as a structured field: iPhone discovery card `.cond` text (`Riverside Courts 2 & 3 · 4:00–6:00pm · Level 3.5–4.0 · Gender-balanced pairs`), iPad split-view "Level range" `.field` (`3.5–4.0`), and the iPhone search filter above the list (`Near me · This weekend · Level 3.5–4.0`, unchanged since round 6 — it was already correct, the game-property field was the one that was wrong). `grep -o "3\.5[^ <"]*"` finds no remaining `3.5+` anywhere in the source. The iPhone Register screen (the fourth candidate surface named in the task) does not restate a level range at all — it shows guest count, payment, and cancellation only — so there is no fourth instance to check for drift, and its absence isn't a gap: it's the same "less detail, not conflicting detail" pattern round 6 already ruled non-problematic for the iPad's extra "Format" field. |
| 4 | Intro changelog substantially trimmed, and the competition screen's subtitle describes the screen again | **PASS** | Measured, not eyeballed: the intro `caption-block` is now **116 words** (counted programmatically from the rendered text), against round 6's flagged ~300 — a 61% reduction, not a reword at the same length. It states what changed (round 7's three fixes, named specifically), points to the full history files instead of restating them, and states the two still-open product/legal items and the one recorded disagreement in one sentence each — same substantive content as before, in roughly a third of the words. Flow 4's subtitle no longer reads as review-history narration; it now reads "Format, entry fee, level-seeded pool entries, and the same shareable-link advertising a social game uses" — a genuine description of the screen's content, in the same style as every other flow's subtitle (Flow 1: "an owner adding a facility, a camera reference, and a discount rule with an end condition"; Flow 2: "Rules, payment method, capacity, friend registration, and social-media advertising in one setup flow"). The word "round" and a bare round number no longer appear in any flow's `<h2>` subtitle — only in the deeper, per-screen `.screen-caption` paragraphs below the mockups, which is a materially different position in the reading path than the subtitle round 6 flagged. |

---

## 2. Independent clarity judgment — did the trim actually help?

Asked as the task requires: a genuine judgment, not a box to check because a
word count went down.

**Yes, this is a real improvement, not a cosmetic one.** The mechanism
matters as much as the result. Round 6's complaint had two parts: (a) the
changelog block had grown in raw size to the point of competing with the
actual content above the fold, and (b) one screen's subtitle had stopped
describing its screen at all. Round 7 didn't just shorten prose — it
restructured the *kind* of information in each spot. The footer no longer
carries a fact that can go stale (it defers instead of restating); the
changelog block names *what* changed in one clause per round-7 fix instead
of narrating *how* the process reached each fix; and Flow 4's subtitle went
back to being about the screen, full stop. A reader who has never seen
rounds 1–6 can now read every flow's subtitle and get an accurate one-line
description of what that screen is for — that was untrue of Flow 4 as of
round 6, and it's true again now.

**It is not spotless, and I'd rather say so than round up.** Two small
residues of review-history language remain, both pre-existing (not
reintroduced this round, and not among the four things round 7 was asked to
fix), and both worth naming honestly rather than let a future round
rediscover them as if new:

- Flow 1's `.screen-caption` still carries the parenthetical "— **as of
  round 4** — club recurring hire as its own discount rule..." This is a
  meaningfully smaller problem than Flow 4's old subtitle was: the sentence
  still fully describes the screen's actual content (a club discount rule,
  independently rated and end-conditioned) and the round-4 callout is an
  aside inside that sentence, not a replacement of it. But it's still a
  place where a reader has to know what "round 4" refers to for one clause
  to fully parse, which is exactly the kind of dependency-on-review-history
  the round-6 finding was about in miniature.
- The changelog's closing sentence about the pill-reuse disagreement ends
  "...or split into a dedicated rating variant (round 4)" — same pattern,
  same low severity.

Neither of these crowds out content the way the pre-round-7 changelog or the
old Flow 4 subtitle did, and neither was in scope for round 7's assigned
fixes, so I'm not scoring them as a failure of this round's work. But if the
standard is "reads as a clean, stand-alone design sketch to someone who
opens it cold," these two phrases are the last visible seam. **Honest
verdict: substantially improved, no longer reads as patched-together at the
structural level round 6 flagged, but not fully purged of review-process
language at the sentence level.** A team opening this artifact today would
not be distracted or misled by it, which is the bar that actually matters —
but "not distracted" and "zero remaining trace of seven rounds of edits" are
different claims, and only the first is true.

---

## 3. Fresh sweep for the recurring bug class (rounds 2, 4, 5, 6), plus the changelog-accuracy check

Read every `.cond`, `.hint`, `.field`, `.price-card`, and `.screen-caption`
across all six flows again, independent of the three items already
prescribed by items 1–3, looking specifically for a fix landing on one
surface while a second surface keeps the pre-fix claim, or two surfaces
stating the same fact differently.

- **Capacity ("X/Y seats")** — four occurrences (tablet roster pill, roster
  `.caption-block` prose, iPhone discovery card pill, iPad condensed list),
  all read "12/16 seats." No regression, no drift.
- **Discount math** — individual card `$18 × 0.75 = $13.50/hr` ✓, club card
  `$18 × 0.80 = $14.40/hr` ✓. Both correct, both stable since round 4.
- **"Bracket"** — appears exactly once in the full source, in the
  round-6-corrected, format-neutral form ("round robin, ladder, or a true
  bracket, depending on what the host picks"). No new instance reintroduced.
- **Club vs. individual discount framing** — the Flow 1 `.screen-caption`'s
  "not inherited from the individual discount above it" still matches the
  price card and hint text above it. No regression from round 4/5's fix.
- **The three round-7 fixes themselves, cross-checked against every other
  surface that could restate the same fact:** footer (only place that ever
  stated a round number besides the masthead — now doesn't), cancellation
  cutoff (only two surfaces state a cancellation time at all — Flow 2's
  general policy and Flow 3's specific instance — and they're consistent by
  construction, one is general and one is a correct specific case of it),
  level range (three surfaces checked in §1 item 3, all consistent). No
  fourth or fifth surface anywhere in the source restates any of these three
  facts in a way that could have been missed.

**No new instance of the recurring bug class found.** This is itself
notable given the pattern of rounds 2, 4, 5, and 6 each finding at least one
— either the fixes this round were unusually clean, or seven rounds of
grep-based verification have genuinely exhausted the easy-to-miss surfaces.
Can't distinguish those two explanations from inside a single round, which
is exactly why the convergence rule requires two consecutive clean rounds,
not one.

**Changelog-accuracy check (the task's specific final ask):** does the
trimmed 116-word changelog's own claim about round 7 accurately describe
what round 7 actually changed, now that items 1–3 have been independently
verified rather than assumed? The relevant clause: *"(round 7) a footer
that had stopped tracking the round number, a cancellation time that didn't
match its own game's start time, and a level range shown two different ways
for the same game."* Checked against the independently-verified facts
above:

- "a footer that had stopped tracking the round number" — accurate
  description of the round-6 finding and directionally accurate about the
  fix (the footer no longer states a stale number); the fix technically
  makes the footer stop *stating* a round number rather than resume
  *tracking* it, but that's a fair, non-misleading gloss of the same
  outcome, not an inaccurate claim.
- "a cancellation time that didn't match its own game's start time" —
  accurate; matches the verified fix in §1 item 2 exactly.
- "a level range shown two different ways for the same game" — accurate;
  matches the verified fix in §1 item 3 exactly.

**All three claims in the trimmed changelog hold up against independent
verification.** The changelog is not just shorter — it's still telling the
truth, which is the harder property to get right when the fix itself is
under active editing.

---

## 4. Round 7 verdict

**Not converged — proceeds to round 8. Round 7 is clean.** All four
assigned checks pass on independent verification, not on trusting the
artifact's own description of itself. The fresh sweep for the recurring
"fix on one surface, stale claim survives on another" bug class — the same
class rounds 2, 4, 5, and 6 each found a live instance of — found nothing
new this round. The trimmed changelog's own claims about what round 7 fixed
were independently checked against §1's verified findings and hold up
exactly, not approximately.

**Consecutive clean rounds: 1 (this round).** Per the convergence rule
("two consecutive rounds find nothing material"), round 6 was material
(three new findings), which reset the counter to zero. Round 7 finding
nothing material makes this the **first** clean round, not proof of
convergence — round 8 must also find nothing material before this review
converges. Do not stop at round 8's start assuming round 7's cleanliness
carries forward; verify round 8 independently, the same way every round
back to round 1 was told to.

§2's independent clarity judgment is additive, not a pass/fail gate: the
artifact is genuinely better-organized after round 7's trim, not just
shorter, but two small review-history residues remain (Flow 1's "as of
round 4" and the disagreement note's "(round 4)") — both pre-existing, both
low severity, both worth a future round's one-line cleanup but neither
blocking convergence on their own.

Unchanged from rounds 1–6, still independently open regardless of this
round's findings: **`Registration.GuestCount` mutability** and **the
camera-consent attestation wording** remain explicit product/legal
decisions pending the user's direct sign-off — not resolved by any design
round, and not blocking round 8 from proceeding on the same
working-assumption basis established in round 1.
