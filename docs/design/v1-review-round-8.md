# Design Review — Round 8

**Reviewing:** the companion visual artifact (same URL, unchanged since round
7's republish — `https://claude.ai/code/artifact/bcfb62bc-d157-44dc-95da-5166f4480ab9`,
title and masthead both still read "Round 7"), fetched fresh via `WebFetch`
(not assumed from round 7's saved copy) and diffed byte-for-byte against
round 7's own quoted HTML — confirmed identical, so this is a genuinely
independent re-read of the same content, not a reaction to any new edit —
plus a from-scratch, whole-page pass rather than a re-run of any prior
round's checklist, plus a full fresh end-to-end read of
`docs/design/v1-system-design.md` cross-checked against the
artifact specifically for doc-vs-artifact mismatches (a class this review
has repeatedly found *within* the artifact but never checked across the two
documents), plus a dedicated accessibility pass deliberately scoped away
from the pills/toggles/buttons every prior round already covered.
**Roles played:** UX/UI Designer, Product Manager, Principal Engineer,
Product Owner — same four dossiers as rounds 1–7.
**Round:** 8 of up to 10.

---

## 1. Verdict on round 7's two noted residues

Round 7 flagged two low-severity items and called both non-blocking: Flow
1's `.screen-caption` parenthetical "— **as of round 4** —", and the
changelog's closing disagreement note ending "...(round 4)". Re-examined
independently, not deferentially:

**Verdict: round 7's call holds, for a reason worth stating precisely
rather than just re-affirming it.** These two instances are not
equivalent, and treating them identically (as round 7 implicitly did)
slightly understates the difference:

- The disagreement note's "(round 4)" sits **inside the intro
  `caption-block`** — a section whose entire stated purpose is to be a
  changelog ("7 rounds of review folded in — accessibility fixes, a
  missing competition/advertising screen, several stale-text fixes, and
  (round 7) a footer that had stopped tracking the round number..."). Round
  references are the expected content of this block, not an intrusion into
  it — this "(round 4)" reads no differently than the "(round 7)" three
  sentences earlier in the same paragraph. This isn't really a residue at
  all; it's correctly-placed meta-commentary in the one part of the page
  designed to carry meta-commentary, consistent with the footer's own
  framing ("See the masthead for the current round and
  `docs/design/review-round-*.md` for full history").
- Flow 1's "as of round 4" sits inside a `.screen-caption` — the one
  paragraph type this artifact otherwise uses **exclusively** to describe
  screen content, never review process. Checked all four other
  `screen-caption`s (Flow 2, Flow 4, and the two others) fresh: zero
  contain a round number or the word "round." Flow 1's is the only
  screen-caption where process language leaks into content description,
  which is a real, if extremely minor, pattern break — a reader gets one
  clause of one sentence on one of six screens that requires knowing what
  "round 4" refers to, where every other screen-caption on the page needs
  no such context.

Neither is **material** by the bar this review has actually applied for
seven rounds — a factual contradiction, a missing requirement, a broken
invariant, a real accessibility failure. Both are wording/placement nits.
**Confirmed non-blocking**, with Flow 1's instance the (very slightly) more
legitimate of the two to eventually clean up, and the disagreement note's
instance arguably not a defect at all once you look at where it actually
sits.

---

## 2. Doc-vs-artifact cross-check — two real findings, one material

This review has repeatedly found the "fix lands on one surface, stale
claim survives on another" bug *within* the artifact (rounds 2, 4, 5) but
had never checked it **across** `v1-system-design.md` and the
artifact until this round, per the task's explicit instruction. Read the
doc fully, end to end, then checked every claim it makes about what the
artifact shows against the artifact's actual markup.

### 2.1 — Material: the camera-consent attestation checkbox the doc treats as "wording pending" doesn't exist anywhere in the artifact

The doc's top blockquote and §3.1 are unambiguous: the camera-link
requirement (§7.01) was **augmented in round 1**, with all four roles'
agreement, to require "a required attestation checkbox at facility
onboarding ('I confirm this facility has appropriate signage/consent for
any security cameras linked here') alongside the camera-URL field." §3.1
frames this as a **working assumption already adopted**, with only its
exact wording still open: *"facility onboarding requires a
consent/signage attestation checkbox... Working assumption (put to the
user, not yet confirmed): facility onboarding requires a
consent/signage attestation checkbox."*

The artifact's own intro `caption-block` restates this the same way —
as a settled requirement with only phrasing left open: *"the exact
camera-consent attestation wording (§3.1)"* is listed as one of two items
"pending the user's product/legal call" — language that only makes sense
if the checkbox itself exists and merely awaits final copy.

**It doesn't exist.** Flow 1 — the screen explicitly captioned "Reads the
requirements literally... a camera link stored as a reference (not hosted
footage, §7.01)" — shows the full security-camera field verbatim:

```html
<div class="field">
  <label>Security camera <span class="pill open" ...>(optional)</span></label>
  <div class="control">
    <span>stream.swingvision.app/riverside/court-3</span>
    <button type="button" class="btn ghost" ...>Verify link</button>
  </div>
  <p class="hint">We link to your existing camera provider — we don't host or store footage.</p>
</div>
```

No checkbox. No attestation copy, draft or otherwise. No mention of
consent or signage anywhere in Flow 1's markup or its screen-caption. A
reader relying on the artifact's own summary ("only the wording is open")
would reasonably expect to find *a* checkbox with placeholder text to
judge — there is nothing to judge. This is exactly the class of bug this
review exists to catch, just crossing a document boundary instead of
staying inside one screen: the doc and the artifact's own changelog both
describe this as one short step from done, and the actual mockup never
took the first step.

**Recommendation:** either add the attestation checkbox to Flow 1 (even
with placeholder wording, consistent with round 1's original suggested
copy) so the artifact actually shows what the doc and its own intro text
claim is settled, or change the intro text to accurately say the
*checkbox itself*, not just its wording, remains unbuilt in this mockup.

### 2.2 — Minor: the doc's requirement-9 row still describes the round-4 screen as "bracket preview," which round 5 itself renamed for being misleading

Doc §2, row 9: *"Round 4 added a dedicated competition-setup screen
(**bracket preview** + the same shareable-link advertising pattern as a
social game), so the artifact now actually shows what this row
describes."* That parenthetical was accurate the day round 4 shipped. It
is stale today: round 5's own review found that "Bracket preview" was a
wrong label (a round-robin pool doesn't produce a bracket) and the
artifact was fixed to read "Pool A entries — seeded by level" — confirmed
still true in this round's fresh read of the artifact (the word "bracket"
appears exactly once in the whole page, inside format-neutral list
language, never as a section label). The doc's cross-reference to that
screen was never revisited after round 5's fix, so a reader checking the
doc's description of "what round 4 built" against the current artifact
would go looking for "bracket preview" and not find it. Lower severity
than §2.1 (it's phrased as historical narration of what round 4 did, not
a present-tense claim, and the underlying screen itself is not
misdescribed in any way that matters to a reader who doesn't cross-check
word-for-word) — but real, and the same underlying failure mode.

**Both are genuine findings against item 2's specific mandate.** §2.1 is
material on its own: it's not a wording nit, it's the artifact's own
summary overstating how resolved a real product/legal requirement is,
which is precisely the kind of misleading "coverage" claim round 3
originally flagged for requirement #9's missing screen.

---

## 3. Fresh accessibility pass — deliberately away from pills/toggles/buttons

### 3.1 Heading hierarchy — clean

Exactly one `<h1>` on the page ("Booking, hosting, and social play — one
platform, three screens, seven open questions."). Six `<h2>`s, one per
`section.block`, all siblings at the same level — no skipped levels. Seven
`<h3>`s exist, all inside `.qcard` elements within the first `<h2>`
section only, correctly nested one level below their parent `<h2>` and
never appearing before the page's first `<h2>`. No `<h4>`–`<h6>` anywhere.
**No finding.**

### 3.2 Non-text content — no real failures, one small inconsistency

No `<img>` tags exist anywhere in the artifact, so there's no alt-text
gap to find. Every decorative graphical element is an empty `<span>`;
most (`.mark`, the masthead `.dot`, device-chrome `.dot`s, `.avatar`,
`.notch`) are correctly `aria-hidden="true"`. Two are not:
`.dot-av` (the roster-row avatar placeholder, used ~9 times) and
`.social-chip .sw` (the platform-color swatch, used 8 times). Because
both are genuinely empty elements with no text content and no ARIA role,
most screen readers won't announce them regardless — so this isn't a live
WCAG failure — but it's an inconsistency worth naming: every other
decorative dot/circle on the page was deliberately marked
`aria-hidden`, and these two were not, for no apparent reason. Cheap
one-line fix, not urgent.

### 3.3 Form-label association — no live defect, but a real gap for whoever builds this next

Every `.field label` (30+ instances) lacks a `for` attribute and does not
wrap its corresponding `.control` — label and control are always
siblings. Checked whether this is a live WCAG 1.3.1/4.1.2 failure: it
is not, **today**, because none of the `.control` elements are real form
controls — they're `<div>`s with static text (`<div class="control
stacked">Riverside Courts</div>`), not `<input>`/`<select>`/`<textarea>`.
A screen reader reads the label text and the value text as ordinary
adjacent content either way, so nothing is silently inaccessible right
now. But this is exactly the kind of gap round 2 already flagged for the
old `<span>`-as-button pattern: if a T7+ implementer copies this markup
shape and swaps `<div class="control">` for a real `<input>` (the natural
next step once this becomes a build), nothing here creates the `for`/`id`
pairing that input would need, and — unlike the CTAs, which *were*
upgraded to real `<button>`s with correct semantics in round 3 — no round
has ever done the equivalent pass for these `<label>`s. Worth a note for
whichever ticket turns this mockup into real form markup; not a defect in
the mockup itself.

### 3.4 Color contrast — body/caption text is fine; **the base --success/--warning/--critical tokens fail as direct text color, a defect that survived all seven prior rounds**

`--ink-soft` (used for `.cond`, `.hint`, `.screen-caption`,
`.caption-block`, `.field label`, and most secondary text on the page) was
computed against every background it appears on, both themes:

| Context | Light | Dark |
|---|---|---|
| ink-soft on `--paper` | 7.65:1 | 8.63:1 |
| ink-soft on `--paper-raised` | 8.66:1 (light, computed) | 6.91:1 |
| ink-soft on `--court-soft` | 7.23:1 | 6.21:1 |

All comfortably clear 4.5:1. **No finding on body/caption text generally.**

But rounds 1–3's contrast work only ever touched the **`.pill`** component
(`paid`/`unpaid`/`full`/`waitlist`), and round 3 explicitly noted — without
computing it — that the *base* `--success`/`--warning`/`--critical` tokens
are "still used elsewhere for text-on-`--paper` contexts, e.g. `.eyebrow`,
`.qcard .qn`," treating that as a non-finding because it wasn't a token
*collision*. Nobody in seven rounds computed the actual contrast for those
other consumers. This round did, precisely (WCAG relative-luminance
formula, script-verified, not eyeballed):

| Element | Uses | Light theme | Dark theme |
|---|---|---|---|
| `.eyebrow` (page eyebrow label) | `--success` text on `--paper` | **3.51:1 — fail** | 8.42:1 — pass |
| `.qcard .qn` ("01"–"07" labels, 7 instances) | `--success` text on `--paper-raised` | **3.51:1 — fail** | 8.42:1 — pass |
| `.hint.flag` (the RSVP/social-reply disclaimer, Flow 2 *and* Flow 4) | `--warning` text on `--paper` | **3.39:1 — fail** | 7.89:1 — pass |
| `.tag.new` (7 qcards, "New"/"Touches shipped code") | `--success` text on 16%-tint bg | **3.29:1 — fail** | 5.35:1 — pass |
| `.tag.open` (7 qcards, "Needs confirmation") | `--warning` text on 18%-tint bg | **3.11:1 — fail** | 4.95:1 — pass |
| `.tag.conflict` ("Conflicts with existing decision") | `--critical` text on 16%-tint bg | **3.80:1 — fail** | **4.39:1 — fail** |

All six sit well under the 4.5:1 AA floor for normal text (none of this
text is large — 0.62–0.72rem, i.e. ~10–11.5px, nowhere near the
18px/14px-bold large-text exemption). Five of six fail only in light
theme — the exact same asymmetric miss rounds 1–2 found and fixed for
`.pill`, reappearing in sibling components that reuse the *unremediated*
base tokens directly instead of the `--pill-*-bg`/`--pill-fg` pair built
specifically to be theme-safe. `.tag.conflict` is worse: it fails in
**both** themes, because `--critical`'s dark-theme value isn't shifted
light enough to clear 4.5:1 even through the same 16% tint that clears
for `.tag.new`.

This is not a hypothetical or a near-miss — it's the identical
CSS anti-pattern round 1 named on day one ("same-hue tint-on-saturated-text
pairings are a common way to quietly fail WCAG 1.4.3 even though they
read as on-brand"), unfixed in a different component that happens to sit
right next to the one that got fixed, on the same page, using the same
underlying color tokens, for all seven rounds this review has run. It
touches visible, substantive content — the RSVP/social-reply disclaimer
text is the one place the artifact explains why the platform won't
auto-monitor social replies, and it's under-contrast in light mode on
both screens it appears — and it touches all seven of the "things to
settle" cards' status tags, i.e. a component on the very first screen a
reader sees.

**Recommendation:** extend the round-3 remediation pattern
(`--pill-*-bg`/`--pill-fg`) to a parallel `--tag-*-fg` set of theme-aware
tokens for `.tag.new`/`.tag.open`/`.tag.conflict`, and give `.eyebrow`/
`.qcard .qn`/`.hint.flag` a dedicated theme-aware ink rather than the raw
`--success`/`--warning` tokens — the same fix shape already proven to
work for `.pill`, just not yet applied to its neighbors.

---

## 4. Convergence-trend assessment — honest, not momentum-driven

Rounds 5, 6, and 7's findings did shrink in scope: a one-clause stale
claim (round 5), three independent-but-contained factual/numeric
mismatches (round 6), then nothing material (round 7, with two
wording-level residues). It would be easy to read that as a smooth
glide path to convergence. **This round's findings say that reading was
wrong, or at least premature.**

The reason isn't that the artifact got worse — it didn't; nothing changed
since round 7. It's that rounds 5–7's "fresh sweeps" kept re-scanning the
same *kind* of surface: prose claims, numbers, and dates restated across
screens. That surface really has been thoroughly gone over, and rounds
6–7's shrinking yield from it is real evidence of *that specific
surface's* convergence. But §3.4's finding came from checking a
*different* surface — non-pill consumers of the same color tokens the
pill fix used — that no round had ever actually computed, despite round 3
explicitly naming two of the six failing elements in passing three rounds
ago. And §2.1's finding came from checking a surface (doc-vs-artifact
claims, rather than within-artifact claims) that literally had never been
checked at all before this round, per the task's own framing.

**Honest assessment: the shrinking-severity trend in rounds 5–7 reflected
a narrowing of what was being checked, not the artifact running out of
real problems.** §3.4's contrast failure is arguably comparable in scope
and severity to round 1's original pill-contrast finding — six failing
element/theme combinations across a component used on the artifact's
first screen — not a "smaller" finding than round 6's. A fresh set of
eyes, or the same eyes asking a different question, plainly could and did
find something this round as significant as an earlier round's headline
findings. Ten rounds is a budget, not a proof that round *n* is
necessarily smaller than round *n−1*; this round is direct evidence
against assuming otherwise.

---

## 5. Round 8 verdict

**Not converged — proceeds to round 9. Round 8 found material issues; the
consecutive-clean-round counter resets to 0** (down from round 7's 1).

Two independent, real findings, either one alone sufficient to make this
round material:

1. **§3.4 — a genuine WCAG 1.4.3 contrast failure**, previously
   undetected across seven rounds of pill/toggle/button-focused
   accessibility review: `--success`/`--warning`/`--critical` used
   directly as text color (not through the theme-aware `--pill-*-bg`/
   `--pill-fg` tokens rounds 1–3 built) fails 4.5:1 in `.eyebrow`,
   `.qcard .qn`, `.hint.flag`, `.tag.new`, and `.tag.open` in light theme,
   and `.tag.conflict` fails in **both** themes. This is the same defect
   *class* rounds 1–2 already found and fixed once — just never checked
   for these other consumers of the same base tokens.
2. **§2.1 — a doc-vs-artifact mismatch**: the companion doc and the
   artifact's own intro text both frame the camera-consent attestation
   checkbox as a settled requirement with only its wording left open, but
   no such checkbox — placeholder wording or otherwise — exists anywhere
   in the artifact's Flow 1 mockup. This is the first time this review has
   checked doc-claims against artifact-reality rather than
   artifact-against-itself, and it found a real gap on the first pass.

§2.2 (doc's stale "bracket preview" cross-reference) and §1 (the two round-7
residues) are confirmed low-severity/non-blocking on their own and don't
change this round's verdict — the verdict rests on §3.4 and §2.1, which
are unambiguous findings by this review's own established bar (a real,
computed accessibility failure; a real, verified claim-vs-reality gap),
not judgment calls.

Unchanged from rounds 1–7, still independently open regardless of this
round's findings: **`Registration.GuestCount` mutability** and **the
camera-consent attestation wording** (now compounded by §2.1: not just
the wording, but the checkbox itself, is unbuilt in the artifact) remain
explicit product/legal decisions pending the user's direct sign-off — not
resolved by any design round, and not blocking round 9 from proceeding on
the same working-assumption basis established in round 1.
