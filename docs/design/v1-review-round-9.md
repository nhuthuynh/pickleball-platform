# Design Review — Round 9

**Reviewing:** round 8's two findings (missing camera-consent checkbox;
`--success`/`--warning`/`--critical` used directly as text color in
`.eyebrow`, `.qcard .qn`, `.hint.flag`, `.tag.new`, `.tag.open`,
`.tag.conflict`) against the companion visual artifact (same URL,
republished — `https://claude.ai/code/artifact/bcfb62bc-d157-44dc-95da-5166f4480ab9`),
fetched fresh via `WebFetch` and saved to disk (826 lines, full `<style>`
block and full `<body>` markup), not inferred from the artifact's own
changelog callout, per the task's standing instruction — plus a hand-worked
WCAG 1.4.3 contrast computation (relative-luminance formula, not eyeballed)
for the new `--ink-success`/`--ink-warning`/`--ink-critical` tokens in both
themes, plus an exhaustive `grep`-driven sweep of every CSS rule and inline
`style=` attribute referencing `--success`/`--warning`/`--critical`/
`--court`/`--accent`, plus a fresh full read of all six flows, the intro,
and the footer for the recurring "fix on one surface, stale claim survives
on another" pattern this review has found in rounds 2, 4, 5, 6, and 8.
**Roles played:** UX/UI Designer, Product Manager, Principal Engineer,
Product Owner — same four dossiers as rounds 1–8.
**Round:** 9 of up to 10.

---

## 1. Verdicts on the two assigned checks

| # | Check | Verdict | Reason |
|---|---|---|---|
| 1 | Camera-consent checkbox exists, correctly associated, correct copy | **PASS** | Flow 1 now has a real checkbox: `<input type="checkbox" id="camera-consent" checked ...>` paired with `<label for="camera-consent">I confirm this facility has appropriate signage/consent for any security cameras linked here.</label>` — a genuine `id`/`for` pair, not visually-adjacent text (`grep`-verified: exactly one `id="camera-consent"` and one `for="camera-consent"` in the source, on the input and its label respectively). The copy is a verbatim match, word for word including punctuation placement, to `docs/design/v1-system-design.md` line 117–118's quoted attestation text ("I confirm this facility has appropriate signage/consent for any security cameras linked here"). A supporting hint ("Required before the camera link above can be saved.") makes the field's purpose and gating behavior explicit. This closes round 8's §2.1 finding cleanly — the doc and the artifact now agree, and the checkbox is a real form control, not a caption promising one. |
| 2 | `--ink-success`/`--ink-warning`/`--ink-critical` clear 4.5:1 on `--paper`/`--paper-raised` for `.eyebrow` and `.qcard .qn`, both themes; `.tag.*` now use solid-fill `--pill-*-bg`/`--pill-fg` like `.pill`, both themes | **PASS** | See §2 for worked numbers. All four consumers (`.eyebrow`, `.qcard .qn`, `.hint.flag`, `.tag.new`/`.open`/`.conflict`) now route through either the new `--ink-*` tokens or the round-3 `--pill-*-bg`/`--pill-fg` tokens — never the raw `--success`/`--warning`/`--critical` values — and every combination checked clears 4.5:1 with real margin in both themes. |

---

## 2. Worked contrast — the round-8 fix, computed rather than trusted

WCAG relative-luminance formula, computed by hand from the artifact's own
hex values (not eyeballed, not assumed from the CSS comment's own claim):

| Element | Pairing | Light theme | Dark theme |
|---|---|---|---|
| `.eyebrow` | `--ink-success` (`#1f6f4f`/`#59c79f`) on `--paper` (`#eef2ec`/`#0f1c17`) | **≈5.38:1 — pass** | **≈8.42:1 — pass** |
| `.qcard .qn` | `--ink-success` on `--paper-raised` (`#ffffff`/`#172822`) | **≈6.09:1 — pass** | **≈7.40:1 — pass** |
| `.hint.flag` | `--ink-warning` (`#8f5416`/`#e0a250`) on `--paper-raised` | **≈6.09:1 — pass** | (unchanged from round 8's warning-token math; not re-derived, same token pair) |

(`.qcard .qn` sits on `--paper-raised` because `.qcard`'s own background is
`--paper-raised`; `.eyebrow` sits directly on the page's `--paper`
background inside `.intro` — the two are genuinely different backgrounds,
which is why their light-theme ratios differ slightly, 5.38 vs. 6.09, even
though both draw from the same `--ink-success` value. Both clear the floor
regardless.)

`.tag.new`/`.tag.open`/`.tag.conflict` (lines 176–178) no longer use
`color-mix(...)` tint-on-saturated-text at all — they now read
`background: var(--pill-*-bg); color: var(--pill-fg);`, the identical rule
shape as `.pill.paid`/`.unpaid`/`.full` (lines 229–233). These tokens were
already validated in round 3 (`v1-review-round-3.md` §2): light theme
6.09–6.88:1, dark theme 5.92–7.84:1, all six combinations passing. Since
`.tag.*` now consumes the exact same tokens rather than a parallel
same-looking-but-different implementation, no new computation is needed —
but the *mechanism* was checked, not assumed: `grep` confirms `.tag.new`,
`.tag.open`, and `.tag.conflict` each reference `--pill-success-bg`,
`--pill-warning-bg`, `--pill-critical-bg` and `--pill-fg` respectively, with
no leftover `color-mix` anywhere in the stylesheet. `.tag.conflict`, the one
that failed in *both* themes per round 8, is confirmed fixed in both.

**Item 2 verdict: full PASS**, computed rather than trusted, matching the
rigor round 2 was faulted for skipping the first time this exact defect
class was "fixed."

---

## 3. Item 3 — exhaustive sweep, one live finding

Every CSS rule and inline `style=` attribute in the 826-line source was
checked for `var(--success)`, `var(--warning)`, `var(--critical)`,
`var(--court)`, or `var(--accent)`. Full result set:

| Location | Usage | Verdict |
|---|---|---|
| `.status .dot { background: var(--warning); }` (masthead status indicator) | Background of a 6px decorative dot, `aria-hidden="true"` on its parent span | **(b)** not a text-color risk — background of a decorative, hidden element |
| `.eyebrow`, `.qcard .qn` | `color: var(--ink-success)` | **(a)** validated safe token — see §2 |
| `.hint.flag` | `color: var(--ink-warning)` | **(a)** validated safe token — see §2 |
| `.tag.new`/`.open`/`.conflict`, `.pill.paid`/`.unpaid`/`.full`/`.flag` | `background: var(--pill-*-bg); color: var(--pill-fg)` | **(a)** validated safe token pair — see §2/round 3 |
| `:focus-visible { outline: ... var(--court) }` | Outline, not text | **(b)** border/outline use |
| `.app-topbar .avatar` background, `.toggle.on` background | `background: var(--court)` on decorative/aria-hidden or non-text UI elements | **(b)** background-color use |
| checkbox `accent-color: var(--court)` (camera-consent input) | Native checkbox tick color, not text | **(b)** not a text-color property |
| `.price-card` `border-color: var(--court)` (iPad selected-game card) | Border, not text | **(b)** border use |
| 6× inline `style="background:var(--court); color:var(--paper);"` (discount-%, capacity-ratio, and level-rating "neutral info" pills) | `--court` as background, `--paper` as text — a pairing never explicitly computed before | **(b)**, verified rather than assumed: `#12241f`-family `--paper` text on `--court` background computes to **≈9.30:1 (light)** and **≈11.62:1 (dark)** — both pass with large margin |
| `::selection`, `.brand .mark`, `.btn.primary` — 3× `var(--accent)` | Always background, always paired with the dedicated `--accent-ink` token, never raw `--accent` as text | **(a)**-equivalent, but computed for the first time in 9 rounds since no prior round had: `--accent-ink` on `--accent` (the live `.btn.primary` CTA text — "Publish game," "Save court settings," etc.) computes to **≈8.86:1 (light)** and **≈13.27:1 (dark)** — passes with large margin in both |

**One live, previously-undetected failure, found outside the literal
search pattern:**

**`.field .control` inline style on the "Online (Stripe)" selected
payment-method button (Flow 2, line 562):**
```html
<div class="control" style="justify-content:center;background:var(--court);
     color:#fff;border-color:var(--court);font-weight:650;">Online (Stripe)</div>
```
This is `background: var(--court)` paired with a **hardcoded** `color:#fff`
— not `color: var(...)` at all, which is exactly why a literal grep for
"`color:var(--court)`" (the pattern the task names) would not surface it.
It is the same underlying defect *class* the task is hunting — a fixed
color paired against a background that moves between themes, never
verified in both — just inverted: earlier instances (rounds 1–2's `.pill`)
had a fixed *text* color against a themed background; this one has a fixed
*text* color too, but discovered by reasoning about the pairing rather than
the literal string match.

Computed: `#ffffff` text on `--court`:
- **Light theme** (`--court: #1c463d`, a dark green): **≈10.53:1 — pass**, comfortably.
- **Dark theme** (`--court: #9fe0c4`, a pale mint — deliberately *inverted* from typical dark-theme convention, since `--court` is used as an accent/brand color, not a surface color): **≈1.51:1 — fail**, badly. White text on a near-white pastel green is close to unreadable.

This is a real, severe, unambiguous WCAG 1.4.3 failure on a live
interactive control (the selected state of the payment-method toggle,
Flow 2's host game-creation screen), present in every round's HTML back to
round 1 (the payment-method field itself has never been touched by any
round's fix), and never caught because every prior round's contrast sweep
searched for `color: var(...)`, not for a hardcoded color sitting next to a
`var()` background. It is exactly the "systematic version" the task asked
for finding a case a spot-check would miss — this one would have escaped
even a literal grep for the task's own named search pattern.

**Recommendation:** either give `--court` a dedicated `--court-ink`
companion token (mirroring the `--accent`/`--accent-ink` and
`--pill-*-bg`/`--pill-fg` pattern already proven twice in this stylesheet)
and use it here instead of the hardcoded `#fff`, or keep the payment-method
selected state on the existing `--pill-*-bg`/`--pill-fg` treatment rather
than inventing a fourth ad hoc "colored control" pattern.

---

## 4. Item 4 — fresh full-artifact read: a sixth instance of the recurring pattern

Read the intro, all six flows, and the footer end to end, independent of
items 1–3, looking specifically for the failure shape rounds 2, 4, 5, 6,
and 8 each found: a fix landing on one surface while a stale or incorrect
claim about it survives on another.

**Found: the intro caption-block misattributes the camera-consent
checkbox's existence to round 8, when round 8's own review explicitly
established the checkbox did not exist at that point.**

```html
<div class="caption-block" style="max-width:760px;">
  <strong>9 rounds of review folded in</strong> — accessibility fixes,
  a missing competition/advertising screen, several stale-text fixes,
  and (round 8) a camera-consent checkbox that the doc described as
  settled but the mockup never actually had, ...
  ... the camera-consent checkbox
  itself now exists in this mockup (round 8), only its exact copy is
  still provisional. ...
</div>
```

The first "(round 8)" is a fair, accurate description of round 8's
*finding* (the checkbox was flagged missing). The second is a claim about
the *current state of the mockup* — "the checkbox itself now exists... —
and it attributes that existence to round 8. `v1-review-round-8.md` itself is
unambiguous on this point: *"It doesn't exist. Flow 1 ... shows the full
security-camera field verbatim: [...] No checkbox."* Round 8 was
explicitly review-only (CLAUDE.md golden rule 9's process discipline
applies here too — a reviewing round reports, it doesn't fix), and this
artifact's own established convention for round labels — see Flow 1's
surviving "**as of round 4**" caption and the disagreement note's
"...(round 4)" — is to name the round in which a change was *made*, not
the round that *found* the gap (round 4 is correctly credited because round
4 is the round that actually added the club-discount distinction and the
competition screen). By that same convention, the checkbox's existence
should read "(round 9)" — this is the round that added it, per this
review's own task framing ("republished... with round-9 fixes") and per
the fact that "round 9" never once appears in the artifact's body text
(`grep`-verified: only the `<title>` and the masthead status line say
"Round 9" — nowhere in the prose does the artifact credit anything to the
round it is currently in).

This is the same failure shape rounds 2 (bare "12/16"), 4 (stale "inherits"
caption), 5 (stale "bracket" clause), 6 (footer's stale round number), and
8 (doc-vs-artifact mismatch on the checkbox's existence) each found — a
fix landing on the actual markup (the checkbox is real, correctly built,
verified in §1) while the *changelog's own account of when that fix
happened* is wrong. It is lower-severity than round 8's finding (a reader
is not misled about whether the checkbox exists — it does, and is
described accurately as existing — only about which round built it), but
it is concrete, `grep`-verifiable, and not a matter of interpretation: round
8's own document is the primary source proving the claim false.

**No other instance found.** Checked fresh, independent of prior rounds'
specific checklists: capacity ("12/16 seats," four occurrences, all
consistent), discount math ($13.50/$14.40, both correct), the "bracket"
word (exactly one occurrence, still the round-6-corrected format-neutral
form), the cancellation-cutoff arithmetic (Flow 2's general policy and Flow
3's specific "Sat 12:00pm (4hrs before the 4:00pm start)" still agree), the
level range ("3.5–4.0" on all three surfaces, no stray "3.5+"), and the
footer (still round-number-agnostic, defers to the masthead, no "round 1"
or any other stale digit). Flow 1's pre-existing "as of round 4" residue
(flagged non-blocking in rounds 7 and 8) is unchanged and still the correct,
accurate attribution — not a new problem, and not re-litigated here.

---

## 5. Round 9 verdict

**Not converged — proceeds to round 10.** Two independent findings, each
sufficient on its own to make this round material:

1. **§3 — a genuine, severe WCAG 1.4.3 failure** on the "Online (Stripe)"
   selected payment-method control (Flow 2): hardcoded `color:#fff` against
   `background: var(--court)` computes to ≈1.51:1 in dark theme (pass in
   light, ≈10.53:1). This is the same underlying defect class rounds 1–2
   and round 8 already found and fixed elsewhere in the stylesheet, on a
   component nobody had checked because it doesn't match the literal
   `color: var(...)` search pattern — found here by reasoning about the
   background/foreground pairing rather than by string-matching alone,
   which is exactly the standard the task asked this round to apply.
2. **§4 — a sixth instance of the recurring "fix on one surface, stale/
   wrong claim on another" pattern**: the intro caption-block attributes
   the camera-consent checkbox's existence to round 8, contradicted by
   round 8's own review document, and inconsistent with this same
   artifact's established convention (round label = the round that made
   the change) used correctly elsewhere on the same page.

Items 1 and 2 (the two assigned checks) both pass cleanly on independent,
worked verification — the camera-consent checkbox is real and correctly
built, and the round-8 contrast defect is genuinely fixed with margin in
both themes for every consumer it was found in. Those are not in question.

**Consecutive clean rounds: 0.** Round 8 was material (two findings), and
round 9 is also material (two more, found by design under this round's own
mandate to check exhaustively and to hunt for a sixth instance of the
recurring pattern — succeeding at both). Per the convergence rule, this
does not reset toward convergence; it stays at zero, exactly as it has
every round a finding landed.

**Honest statement of where this leaves the review, given the round cap:**
this is round 9 of 10. Even if round 10 finds nothing material, that
would be the **first** clean round, not proof of convergence — the task's
own two-consecutive-clean-round rule cannot be satisfied by round 10 alone,
since there is no round 11 to serve as a second consecutive clean round.
**This review will reach the round-10 cap without ever having achieved
two consecutive clean rounds**, regardless of round 10's outcome, unless
round 9 is retroactively treated as clean — it should not be, given §3 and
§4 above are both concrete and independently verified. Round 10 should
verify: the `--court`/hardcoded-`#fff` payment-method fix (in both themes,
computed, not eyeballed) and the checkbox's round-attribution correction
(or removal of the incorrect parenthetical), then do one more from-scratch
full read — but it should be reported as "the last round in the budget,"
not "the round that converges this review," since convergence by this
review's own rule is no longer mathematically reachable within the
remaining budget.

Unchanged from rounds 1–8, still independently open regardless of this
round's findings: **`Registration.GuestCount` mutability** remains an
explicit product/legal decision pending the user's direct sign-off — the
camera-consent attestation checkbox itself is now resolved (§1), but its
*exact copy* was already noted by round 8 as the one remaining open
sub-question, and the copy shown (verbatim from the design doc) is a
reasonable default still awaiting the user's explicit confirmation, not a
new open item.
