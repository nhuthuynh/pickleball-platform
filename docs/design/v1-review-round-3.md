# Design Review — Round 3

**Reviewing:** round 2's findings against the companion visual artifact (same
URL, republished — `https://claude.ai/code/artifact/bcfb62bc-d157-44dc-95da-5166f4480ab9`),
verified against the artifact's actual HTML/CSS source (fetched directly via
its raw markup, not inferred from its own "Round 1/2/3" changelog callout,
per the task's standing instruction), plus a fresh open-ended pass over
`docs/design/v1-system-design.md` §1's 16 requirements against
every screen in the artifact.
**Roles played:** UX/UI Designer, Product Manager, Principal Engineer,
Product Owner — same four dossiers as rounds 1–2.
**Round:** 3 of up to 10.

---

## 1. Verification of round 2's four findings

| # | Check | Verdict | Reason |
|---|---|---|---|
| 1 | Pill contrast fixed in **both** themes | **PASS** | Hardcoded hex text is gone. The fix is exactly what round 2 recommended: dedicated `--pill-success-bg`/`--pill-warning-bg`/`--pill-critical-bg`/`--pill-fg` tokens, redefined per theme, consumed by `.pill.paid/.unpaid/.full/.flag`. All 6 status × theme combinations now clear WCAG 1.4.3's 4.5:1 floor — see §2 for the worked ratios. No theme-parity gap remains for this component. |
| 2 | iPad condensed-view bare "12/16" now reads "12/16 seats" | **PASS** | `<p class="cond">Riverside · 4–6pm · 12/16 seats</p>` — the unit is now present at every surface the figure appears (host tablet roster pill, iPhone discovery card pill, iPad condensed list), not just inside `.pill` markup. No remaining bare-ratio instance found anywhere in the source. |
| 3 | CTAs are real `<button>` elements, keyboard-operable | **PASS** | All six named CTAs — Save court settings, Add another court, Publish game, Confirm & pay, Register, Verify link — are now `<button type="button" class="btn ...">`, matching the toggle's existing `<button role="switch">` pattern. Native `<button>` elements are keyboard-focusable and Enter/Space-activatable without additional script, so this closes round 2's stronger finding (not just target size — actual operability). |
| 4 | Any regression from these changes | **PARTIAL** | No token collisions and the theme-override cascade is sound (see §3). One small, real gap found: neither `.btn` nor its variants set `appearance: none` / `-webkit-appearance: none`, so on browsers that don't fully suppress native chrome from explicit `background`/`border` alone (older/some WebKit builds) the real `<button>` elements could retain a faint native bevel/shadow the `<span>` mockup never had. Low severity, not a functional or contrast defect — see §3. |

---

## 2. Worked contrast ratios — the round-2 finding, re-verified in both themes

Computed from the actual hex values in the artifact's `<style>` block
(WCAG relative-luminance formula, not eyeballed):

| Pill | Light bg | Text | Light ratio | Dark bg | Text | Dark ratio |
|---|---|---|---|---|---|---|
| `.paid` | `#1f6f4f` (`--pill-success-bg`) | `#ffffff` | **≈6.09:1 — pass** | `#59c79f` (`--pill-success-bg`, = `--success`) | `#06251b` | **≈7.84:1 — pass** |
| `.unpaid` | `#8f5416` (`--pill-warning-bg`) | `#ffffff` | **≈6.09:1 — pass** | `#e0a250` (`--pill-warning-bg`, = `--warning`) | `#06251b` | **≈7.34:1 — pass** |
| `.full` / `.flag` | `#9c3a2c` (`--pill-critical-bg`) | `#ffffff` | **≈6.88:1 — pass** | `#e28174` (`--pill-critical-bg`, = `--critical`) | `#06251b` | **≈5.92:1 — pass** |

All six combinations clear the 4.5:1 AA floor for this size text, several
with real margin. Mechanism, confirmed rather than assumed: light theme
gets **new, purpose-built darker backgrounds** (`--pill-*-bg`, distinct
from the base `--success`/`--warning`/`--critical` tokens used elsewhere
for text-on-`--paper`) paired with solid white text; dark theme reuses the
base `--success`/`--warning`/`--critical` values (already light/bright) as
the pill background, paired with the single dark ink `#06251b`. `.waitlist`
(`background: var(--ink-soft); color: var(--paper-raised)`) is unchanged
and was already passing in both themes per round 2 — not re-verified here
since nothing about it changed.

**One margin note, not a failure:** round 2's *old* hardcoded scheme
happened to use a separately-tuned dark ink per status (`#2c0a05` for
full/flag specifically) that produced a slightly higher dark-theme ratio
(≈6.61:1) than the *new* shared `--pill-fg: #06251b` produces for the same
pill (≈5.92:1) — because one ink now serves all three hues, `.full`/`.flag`
gave up some of its old margin to gain theme-awareness. Still comfortably
above 4.5:1, so not a finding on its own, but worth naming since "still
passes" and "as much margin as before" are different claims and only the
former is true here.

---

## 3. Regression check, in the same rigor as rounds 1–2's "don't trust a plausible mechanism"

- **Token collisions:** `--pill-success-bg`, `--pill-warning-bg`,
  `--pill-critical-bg`, `--pill-fg` are declared exactly three times
  (`:root`, `:root[data-theme="dark"]`, and the `@media (prefers-color-scheme:
  dark) { :root:not([data-theme="light"]) }` block) and consumed only by
  the four `.pill.*` rules. No shadowing of `--success`/`--warning`/
  `--critical` (those tokens are untouched and still used elsewhere for
  text-on-`--paper` contexts, e.g. `.eyebrow`, `.qcard .qn`). **No
  regression.**
- **Native `<button>` default styling:** `.btn` explicitly sets
  `font-family`, `font-weight`, `font-size`, `padding`, `border-radius`,
  `border: 1px solid transparent`, and `cursor: pointer`; `.btn.primary`/
  `.btn.ghost` each set an explicit `background`. That's enough to prevent
  the classic "gray gradient button" in evergreen Chrome/Firefox, but
  **no rule anywhere in the stylesheet sets `appearance: none` /
  `-webkit-appearance: none`**, which is the one property that
  unconditionally suppresses native platform chrome (a residual inset
  shadow/bevel some WebKit versions still render under explicit
  background+border, per known cross-browser button-reset gotchas). `.toggle-hit`
  does this correctly (`background: none; border: none; padding: 0;` — a
  full manual reset), so the pattern exists in the file, just wasn't
  applied to `.btn`. Low-severity, cosmetic-only, real. Worth a one-line
  fix (`appearance: none;` on `.btn`) before this becomes the shared
  button component three clients copy.
- **`:root[data-theme="light"]` override path:** no explicit block with
  that selector exists. Traced the cascade instead of assuming a gap:
  `:root[data-theme="dark"]` only matches `data-theme="dark"`; the dark
  media-query rule's selector is `:root:not([data-theme="light"])`, which
  explicitly excludes elements carrying `data-theme="light"`. So when a
  user with an OS dark preference stamps `data-theme="light"` (the
  artifact viewer's theme control does this, per the runtime script's
  `X(e)` function), **neither** dark rule matches, and the base `:root`
  block — which already holds the light hex values as its unconditional
  default — is what's left in effect. Both `:root[data-theme="dark"]` and
  the media-query rule tie at specificity (0,2,0), so there's no
  specificity trap either. **This is sufficient and correct**, not a gap
  papered over by convention — a separate `:root[data-theme="light"]`
  block would be redundant with the base `:root`, not missing coverage.

---

## 4. Open-ended pass — design-direction gaps, not implementation details

Per the task brief's framing that round 3 should look past mechanics at
whether the artifact's screens actually cover the 16 requirements in
`v1-system-design.md` §1:

**Requirement #9 (competitions + social-media advertising) has no screen,
despite the artifact's own intro claiming it's covered.** The intro
paragraph states this pass covers "...host-run social games with
cash/online payment, friend registration, level- and gender-aware
matchmaking, **and competitions advertised through social media**." But
every screen in the artifact is scoped to a **Game** (`New social game`,
`Find a game`, `Sat mixed doubles ladder` roster) — the word
"Competition" does not appear anywhere in the markup, and the "Advertise"
block (WhatsApp/Facebook/X/Instagram shareable-link chips) is nested
inside Flow 2's *game*-creation screen, not a separate competition-setup
flow. Per this project's own glossary (`agent-operating-handbook.md` §A2),
`Competition` is a distinct aggregate from `Game` — "an Organiser-run
tournament/bracket format" with its own brackets/rounds — not a
reskin of a Game. Requirement #9 was flagged in the source doc itself as
"one of the more elaborate asks" (§2 row 9, "Existing + Flagged conflict"),
so its complete absence from every screen is a real gap between what the
artifact's own copy claims and what it shows, not a minor omission.

**Requirement #8 (club recurring-hire discount) is shown as a distinct
field, but not as a distinct, configurable discount.** Flow 1 does have a
separate "Club recurring hire" row below the "Discount rule" price-card —
so it isn't literally the same UI element as requirement #3's individual
discount. But the club row shows only an outcome pill ("Club rate
applied") with no visible percentage, amount, or end condition, and the
screen's own caption states the club rate is shown "**inheriting the
facility's discount config** rather than a separate price a club
negotiates ad hoc" — i.e., by the artifact's own description, it reuses
the same `DiscountRule` instance (same 25%-off, same "ends after 10
bookings") drawn two paragraphs above for the individual/off-peak case,
rather than depicting its own club-specific rule. That's a real design gap
against `v1-system-design.md` §3.2's own model, which defines
`AppliesTo (individual|recurring_hire|club)` as a discriminator precisely
because these are meant to be independently configurable discount rule
instances, not one rule wearing two labels. Not fully conflated (there's a
labeled field), but not distinctly *configurable* either — worth a
dedicated club-rate control (its own rate + end condition) before this
becomes a ticket.

No other requirement from the 16 is completely unaddressed by every
screen: #1 (booking), #2 desc/photos/address (already built, correctly
not re-illustrated), #4/5/6/13 (game hosting, rules, payment method,
multi-player), #10 (shareable-link advertising, resolved per §7.05), #11
(friend registration), #12 (cancellation cutoff, capacity), #14/15 (level,
gender-aware matchmaking), and #16 (web/iPhone/iPad) all have at least one
screen. #7 (host adds a facility that doesn't exist yet) also has no
screen, but — unlike #9 — the source doc's own mapping already marks it
"not yet built, no ticket" as existing, un-illustrated scope rather than
new-requirement scope this round is meant to sketch, so it doesn't carry
the same "the copy claims coverage that doesn't exist" problem #9 does.

---

## 5. Round 3 verdict

**The accessibility/interactivity sub-loop opened by round 1 and kept open
by round 2 is now closed.** All four items round 2 flagged — pill contrast
in both themes, the bare "12/16," non-interactive CTAs, and (implicitly)
whether the fix mechanism actually holds up rather than just sounding
right — are verified fixed against the artifact's real source, with worked
numbers, not assumed from "it's token-based now." §3's regression check
found one minor, real, easily-fixed gap (`appearance: none` missing from
`.btn`) and confirmed the theme-override cascade is sound rather than
merely plausible. **Safe to stop re-litigating this specific set of
issues** — nothing here should require a round 4 pass on its own.

**But round 3 is not a clean "nothing material" round**, so per the task's
own convergence rule ("two consecutive rounds find nothing material"),
**the overall review does not converge yet.** §4's open-ended pass found
that requirement #9 (competitions) has no screen anywhere in the artifact
despite the intro copy claiming it's covered, and requirement #8 (club
discount) is a labeled-but-unconfigurable field that inherits, rather than
distinctly specifies, its rate. Both are direction-level gaps, not CSS
issues, and both should be closed with an actual screen/field before this
artifact is treated as a complete sketch of the 16 requirements.

Unchanged from rounds 1–2, still independently open regardless of this
sub-loop's closure: **`Registration.GuestCount` mutability** and **the
camera-consent attestation wording** remain explicit product/legal
decisions pending the user's direct sign-off — not resolved by any design
round, and not blocking further rounds from proceeding on the same
working-assumption basis established in round 1.
