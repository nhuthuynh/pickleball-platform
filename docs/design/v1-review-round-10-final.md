# Design Review — Round 10 (Final)

**Reviewing:** round 9's two findings (hardcoded `color:#fff` on the selected
payment-method control; changelog misattributing the camera-consent
checkbox's addition to round 8) against the companion visual artifact (same
URL, republished — `https://claude.ai/code/artifact/bcfb62bc-d157-44dc-95da-5166f4480ab9`),
fetched fresh (827-line HTML source saved to disk, full `<style>` block and
full `<body>` markup read end to end, not inferred from the artifact's own
changelog callout) — plus the most thorough single-pass audit this review has
run, combining every category checked across rounds 1–9 into one pass:
systematic hardcoded-hex contrast checking (not just `var()`-based pairings),
every interactive element's real operability, full cross-flow factual
consistency, and a doc-vs-artifact cross-check — plus a fresh, independent
read looking for anything none of the first nine passes' specific checklists
would have caught.
**Roles played:** UX/UI Designer, Product Manager, Principal Engineer,
Product Owner — same four dossiers as rounds 1–9.
**Round:** 10 of up to 10 — **the hard cap. This is the last round. The
review workstream ends here, on this round's outcome, whatever it is.**

---

## 0. State of the design at cap — read this first

Ten rounds in, the honest picture is **good but not clean**, and the
distinction matters for anyone about to hand this to T7 ticket-writing.

**What's solid:** all six flows are internally coherent, cover all 16 of the
user's requirements (including the two — competitions, club discount — that
took until round 4 to actually land), use one consistent token system and one
consistent component vocabulary (buttons, toggles, pills, checkboxes are all
real, keyboard-operable controls, not `<span>` mockups), and the front matter
no longer drowns the content the way it did by round 6. Every WCAG contrast
defect this review has ever found — three separate defect classes, found in
rounds 1–2, 8, and 9 respectively — has been fixed and re-verified with
worked numbers in both themes, not eyeballed. The two intentionally-deferred
product/legal calls (`Registration.GuestCount` mutability; the camera-consent
attestation's exact copy) are exactly where they've been since round 1:
named, consistent across every place they're mentioned, and correctly framed
as the user's decision, not this review's to make.

**What isn't clean:** this round's audit — the most exhaustive one this
review has run, specifically because there's no round 11 to defer to — found
a new, real, previously-uncaught defect (§3.5 below): the camera-consent
checkbox added in round 9 ships **pre-checked**. A "required attestation"
that defaults to affirmed defeats its own stated purpose — a host who never
touches the field still submits a checked box. This is not the same category
as the two deferred items (it isn't a product/legal judgment call; it's an
implementation detail that inverts the control's meaning) and it is not
cosmetic the way the lingering "as of round 4" caption or the missing
`appearance: none` are. **A T7 ticket-writer taking this mockup literally
would ship an invalid consent mechanism.** That is a real gap this review is
closing out with, not a nitpick.

Net: **round 10 is material, not clean.** Items 1 and 2 (this round's
assigned verification) both pass. The exhaustive audit is what makes this
round material anyway — which is the right outcome for a last round whose
job is to be thorough, not to manufacture a clean exit.

---

## 1. Verdicts on the two assigned checks

| # | Check | Verdict | Reason |
|---|---|---|---|
| 1 | Selected payment-method control uses `color:var(--paper)` instead of hardcoded `color:#fff`, and actually fixes the contrast in both themes | **PASS** | Line 564: `style="justify-content:center;background:var(--court);color:var(--paper);border-color:var(--court);font-weight:650;"` — `grep`-verified, no `color:#fff` or `color: #fff` remains anywhere in the artifact outside token definitions (`--paper-raised`/`--pill-fg`, both legitimate). Computed WCAG relative-luminance contrast, not eyeballed: **light theme** — `--paper` (`#eef2ec`) text on `--court` (`#1c463d`) bg ≈ **9.31:1 — pass**; **dark theme** — `--paper` (`#0f1c17`) text on `--court` (`#9fe0c4`) bg ≈ **11.62:1 — pass**. This exactly mirrors the roster-capacity-pill pattern (`background:var(--court); color:var(--paper)`) round 9 already validated at ≈9.30:1/≈11.62:1 for other consumers of the same pairing — the payment-method control now uses the identical, already-proven-safe token pair instead of a one-off hardcoded value. |
| 2 | Changelog credits round 9, not round 8, for adding the camera-consent checkbox | **PASS** | Intro `caption-block` now reads: "a camera-consent checkbox added (**round 9**, after round 8 found the doc described it as settled while this mockup never actually had one)... the camera-consent checkbox's existence is no longer in question, only its exact copy is still provisional." `grep`-verified: "round 8" appears exactly twice in the body text, both correctly describing what round 8 *found* (a CSS-comment aside about the `.eyebrow`/`.qcard .qn` measurement, and this attribution clause correctly naming round 8 as the round that found the gap, not the round that closed it). "round 9" appears twice, both correctly crediting round 9 for the two things round 9 actually did (added the checkbox; found the payment-method contrast bug). No stray "round 8 added the checkbox" language anywhere. |

---

## 2. Full audit — organized by the four sub-categories

### 2a. WCAG contrast — every hardcoded hex, not just `var()` pairings

Every inline `style=` attribute containing `color` was extracted (17 total)
and every hardcoded hex literal in the file was catalogued. Result: **zero**
inline `color:` values remain hardcoded — all 17 use `var(--ink)`,
`var(--ink-soft)`, or `var(--paper)`. The only hardcoded hex colors outside
`:root` token blocks are the four `.social-chip .sw` swatches
(`#25D366`/`#1877F2`/`#111`/`#E1306C`, WhatsApp/Facebook/X/Instagram brand
colors) — these are `background` on decorative 8px dots sitting next to
platform-name text that already carries the same information redundantly
(WCAG 1.4.1 is satisfied by the text label; these dots aren't a text/
foreground-background pairing WCAG 1.4.3 governs). No live contrast failure
found in this sweep — the round-9 fix on the payment-method control was the
last hardcoded-text-color instance in the file.

### 2b. Every interactive element — real and keyboard-operable

All 10 CTA `<button type="button" class="btn ...">` elements, all 3
`<button role="switch" aria-checked aria-label>` toggles, and the one
`<input type="checkbox" id=... >` / `<label for=...>` pair are real,
keyboard-focusable, semantically correct controls. Zero `<span class="btn">`
or non-interactive lookalikes remain (`grep`-verified). No `<a href>` tags
exist, so no dead/fake links to check.

**One new, real finding — the checkbox ships pre-checked:**

```html
<input type="checkbox" id="camera-consent" checked
  style="margin-top:2px; width:18px; height:18px; flex:0 0 auto; accent-color:var(--court);">
```

The `checked` attribute means this "required attestation" defaults to
affirmed. No prior round (1–9) flagged this — round 9's verification checked
that the `id`/`for` pairing and the copy matched the doc verbatim, which it
does, but never checked the control's default *state*. A required consent
checkbox that starts checked is a well-documented anti-pattern (the Designer
dossier's own §6 cites deceptive.design's catalog of exactly this shape of
issue under "trick questions"/assumed consent) — under most jurisdictions'
consent standards a pre-ticked box is not treated as an affirmative action.
As written, a host who never touches this field still submits an attestation
they didn't actively make. **Recommendation: remove `checked`, or make this
an explicit acceptance-criterion line in whichever ticket builds this field**
("consent checkbox defaults to unchecked; saving is blocked until the user
actively checks it").

**Second, minor observation on the same element:** the checkbox's own
visible box is `18×18` CSS px — under WCAG 2.5.8's 24×24 AA floor, and
smaller than the deliberate `44×24` hit-area wrapper (`.toggle-hit`) built
for the toggles elsewhere on this exact page. It likely satisfies WCAG
2.5.8's "Equivalent" exception (the paired `<label for="camera-consent">` is
a large, click-activates-the-same-control target, several lines of wrapped
text wide), so this is not flagged as a standalone failure — but it's a real
inconsistency against this artifact's own established pattern of giving
small controls a generous hit area, worth a one-line fix for consistency
before this becomes the shared form-control spec.

**Unchanged, still-open, low-severity items from prior rounds** (checked
fresh, not newly found, not re-litigated as new defects): `.btn` still has
no `appearance: none` (round 3, cosmetic-only); `.dot-av` and
`.social-chip .sw` still lack `aria-hidden` (round 8, functionally inert
since screen readers don't announce empty non-text elements regardless);
`.field label` elements still have no `for`/`id` pairing to any real input
because every other `.control` in the mockup is still a static `<div>`, not
a form control (round 8, correctly non-blocking for a mockup).

### 2c. Internal consistency of every factual claim, all six flows + intro/footer

Fresh, from-scratch check (not a re-run of any prior round's specific
targets):

- **Round number.** Masthead and `<title>` both say "Round 10 of up to 10
  (final)." Footer states no round number at all (round 7's structural fix
  holds — it defers to the masthead rather than restating a number that
  could go stale). No stray "round 1"–"round 7" claim anywhere in body text.
- **Capacity ("12/16 seats").** Four occurrences (tablet roster pill, roster
  caption-block, iPhone discovery card, iPad condensed list) — all
  identical, unit included every time.
- **Discount math.** `$18 × 0.75 = $13.50/hr` ✓, `$18 × 0.80 = $14.40/hr` ✓.
  Both still correct.
- **"Bracket."** Exactly one occurrence, still the round-6-corrected,
  format-neutral phrasing ("round robin, ladder, or a true bracket,
  depending on what the host picks").
- **Level range.** "3.5–4.0" on all three surfaces that state it (iPhone
  search filter, iPhone game-card `.cond`, iPad "Level range" field) — no
  stray "3.5+".
- **Cancellation cutoff.** Flow 2's general policy ("4 hours before start")
  and Flow 3's specific instance ("Sat 12:00pm (4hrs before the 4:00pm
  start)") remain mutually consistent — the round-7 fix holds.
- **"Inherit."** Two legitimate uses (CSS `a{color:inherit}`; the
  cancellation-policy hint correctly describing the *policy* as inherited
  from the facility, which is the accurate, intended meaning) and the
  correct negation ("not inherited from the individual discount above it").
  No recurrence of the pre-round-4 discount-inheritance bug.
- **"Sat mixed doubles ladder" naming vs. iPad's "King of the court"
  format field** — considered and set aside on the same basis round 6 set
  aside "Riverside Summer Ladder" naming a round-robin competition: a
  colloquial game/event name and its technical format field are allowed to
  differ (a "ladder" league commonly runs as rotation-style play), and this
  specific pairing has been present, unflagged, since at least round 6
  without producing real confusion in context. Not counted as a new finding.
- **One pre-existing, already-non-blocking residue, unchanged:** Flow 1's
  screen-caption still carries "— **as of round 4** —" (flagged non-blocking
  in rounds 7 and 8; still the only screen-caption on the page with any
  review-process language in it). Not re-litigated as new.

**No new "fix on one surface, stale claim on another" instance found** — the
one class of bug this review has now found in six separate rounds (2, 4, 5,
6, 8, 9) does not have a seventh instance this round.

### 2d. Companion doc vs. artifact

Read `docs/design/v1-system-design.md` fully against the
round-10 artifact. The one live mismatch this review has ever found across
the two documents (round 8 §2.1 — doc described the consent checkbox as
settled-except-wording while the mockup had none) is resolved: the doc's
§3.1 and top blockquote describe the checkbox as a *design decision*
("working assumption... requires a consent/signage attestation checkbox"),
which was always true independent of mockup status, and the artifact's Flow
1 now shows a real checkbox with copy that is a verbatim match to the doc's
quoted text (`grep`-confirmed word-for-word). The doc's row-9 "bracket
preview" cross-reference, stale as of round 8, was corrected that same round
and remains correct. No new doc-vs-artifact mismatch found. **One nuance
worth naming, not a defect:** the doc's language never made a claim about
the checkbox's default-checked state either way, so §3.5's finding (the
`checked` attribute) is not a doc-vs-artifact contradiction — it's purely an
artifact-implementation gap the doc is silent on.

---

## 3. Round 10 verdict

**Material — not clean.** Items 1 and 2 (the two assigned checks) both pass
cleanly on independent, computed verification. But §2b's exhaustive
interactive-element pass — done specifically because this is the last round
and the task called for combining every prior category into one final
pass — found a real, previously-uncaught defect: the camera-consent
checkbox ships pre-checked, which inverts the meaning of a "required
attestation." This is not a wording nit or a cosmetic residue; it is the
kind of gap that would ship wrong if a T7 ticket-writer copied this mockup's
behavior literally, and no round from 1 through 9 ever checked the control's
default state.

**Consecutive clean rounds at cap: 0.** Round 7 was this review's only clean
round in ten; round 8 reset it, round 9 reset it again, and round 10 —
this round — is material for the reason above. Per the convergence rule
("two consecutive rounds find nothing material"), **this review never
achieves convergence within its budget.** That was already mathematically
foreclosed as of round 9 (round 9 itself said so), and round 10's own
finding confirms it rather than being a missed opportunity — round 10 could
not have converged the review on its own even if it had found nothing,
since there is no round 11 to serve as the second consecutive clean round.

**This is the hard cap. There is no round 11. The review workstream ends
here, on this outcome**, per the task's explicit instruction — not because
round 10 happened to find something, but because ten rounds was always the
budget regardless of outcome.

---

## 4. Closing assessment — the honest state of ten rounds of work

**Could a real Designer+PM+PE+PO team hand this to T7 ticket-writing today?
Mostly yes, with one explicit exception carried forward as a finding, not a
silent gap.**

The six flows are a genuinely usable design sketch: every one of the 16
original requirements has a screen: the domain model (polymorphic Booking,
per-Game capacity, discount rules with mutually-exclusive end conditions,
guest counts, level/gender matchmaking) is drawn faithfully and consistently
across web, iPhone, and iPad; every component that recurs (pills, buttons,
toggles, form fields) resolved to one shared, theme-aware, accessible
pattern by round 3–9, not four screens each reinventing their own; and the
front-matter, which genuinely degraded by round 6 under the weight of its
own changelog, was structurally fixed in round 7 and has stayed fixed since.
A ticket-writer opening this today gets an accurate, coherent picture of
what to build, not an artifact that misleads about its own state.

**But "mostly clean" and "fully clean" are different claims, and this
review should not blur them at the finish line.** Beyond the two
explicitly-flagged, correctly-deferred product/legal calls —
`Registration.GuestCount` mutability and the camera-consent attestation's
exact wording, both genuinely requiring the user's direct call, not this
review's — this final round surfaced one more concrete item a T7
ticket-writer would trip over if it isn't carried into the ticket
explicitly: **the consent checkbox must default to unchecked.** This is not
a third product/legal open question — it doesn't need the user's judgment,
it's simply wrong as drawn, the same way the round-1 contrast fix was simply
wrong before round 3 actually closed it. Unlike the two deferred items, it
has an obvious correct answer and should be written into the ticket's
acceptance criteria directly, not escalated.

**Ten rounds also taught a structural lesson worth stating plainly, since
there's no round 11 to act on it in-artifact:** every round from 2 through 9
found something a previous round's "verified fix" missed — a stale caption,
an unchecked sibling component, a defect that didn't match the literal
search pattern used to hunt for its category, and now, in round 10, a
control's default *state* rather than its markup or its color. The pattern
across all ten rounds is not that this artifact is unusually sloppy — it's
that "verify the assigned fix" and "the artifact is now correct" are
different claims, and each round's exhaustive-sounding sweep still had a
blind spot the next round's differently-framed question could find. A team
picking this up for T7 should read that as a caution about mockups
generally (a fixed-scope visual sketch will always have another angle worth
checking) rather than as unfinished business specific to this one — the
categories of defect this review's ten rounds actually found (contrast,
operability, cross-surface factual drift, doc/artifact sync, and now
default-state correctness) are a reasonably complete taxonomy for a design
review to have exercised, and diminishing, not zero, is the realistic bar
for round *n* of a mockup review, not a failure of process.

**Final state, plainly:**
- Two remaining gaps require the user's product/legal call, unchanged since
  round 1: **`Registration.GuestCount` mutability** and **the exact
  camera-consent attestation copy**.
- One remaining gap does not require the user's call and should go straight
  into T7's acceptance criteria: **the consent checkbox must not default to
  checked.**
- No remaining WCAG contrast failure, no remaining fake/non-interactive
  control, no remaining cross-flow factual contradiction, and no remaining
  doc-vs-artifact mismatch were found in this round's exhaustive pass.
- The review closes at the round-10 cap without two consecutive clean
  rounds, as was already mathematically certain after round 9. That is an
  accurate description of what ten rounds actually found, not a failure to
  reach a bar the process was still capable of clearing.
