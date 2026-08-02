# Design Review — Round 2

**Reviewing:** round 1's five fixes to the companion visual artifact (same
URL, republished — `https://claude.ai/code/artifact/bcfb62bc-d157-44dc-95da-5166f4480ab9`),
verified against the artifact's actual HTML/CSS source (fetched directly,
not inferred from its own "Changed since round 1" callout), plus a second
pass over `docs/design/pickleball-system-design-v1.md` and a
cross-reference check against `docs/process/t6-sprint-plan.md`.
**Roles played:** UX/UI Designer, Product Manager, Principal Engineer,
Product Owner — same four dossiers as round 1.
**Round:** 2 of up to 10.

---

## 1. Verification of round 1's five fixes

Each item was checked against the artifact's actual `<style>` block and
markup (fetched in full), not the "Changed since round 1" summary box,
per the task's instruction. Two of the five checks required computing
approximate WCAG contrast ratios by hand from the CSS custom-property
hex values, since "eyeball it" alone would have missed the theme-specific
failure below.

| # | Check | Verdict | Reason |
|---|---|---|---|
| 1 | Status pills solid-fill, plausibly WCAG 1.4.3 | **PARTIAL/FAIL** | Solid fills confirmed (`.pill.paid{background:var(--success)}` etc., no more `color-mix` tint-on-saturated-text). But the text color on `.paid`/`.unpaid`/`.full`/`.flag` is a **hardcoded hex** (`#06251b`, `#2b1600`, `#2c0a05`), not a token, and it doesn't move between themes. Computed against light-theme tokens: `.paid` ≈**4.10:1**, `.full`/`.flag` ≈**3.87:1** — both under the 4.5:1 floor for this size text (0.64rem/600-weight, well short of the "large text" 18pt/14pt-bold exemption); `.unpaid` ≈**4.50:1**, sitting right on the line. Against dark-theme tokens the same hardcoded text computes to ≈**7.8:1** / **7.8:1** / **6.6:1** — comfortably passing. See §2. |
| 2 | Toggle tap target + keyboard operability | **PASS** | `.toggle-hit` is a real `<button type="button" role="switch" aria-checked="..." aria-label="...">` at **44×24 CSS px**, wrapping a visually slimmer `<span class="toggle" aria-hidden="true">` track — exactly the "bigger hit area, slimmer visible track" fix round 1 asked for, and it's keyboard-focusable/operable by construction (real `<button>`, not a bare `<span>`). Meets WCAG 2.5.8's 24×24 floor exactly on height. New sub-note: still short of Apple HIG's 44×44pt and Material's 48×48dp, which the Designer dossier itself names as the real bar for native — fine as a web mockup, but this exact token can't be copy-pasted verbatim into the Swift/Kotlin component spec. |
| 3 | Roster capacity pill reads "12/16 seats" | **PARTIAL** | The `.pill` component itself now says "12/16 seats" everywhere it's used as a pill (host tablet roster, iPhone discovery card). But the same figure recurs as a **bare "12/16"** with no unit in the iPad split-view's condensed list: `<p class="cond">Riverside · 4–6pm · 12/16</p>`. The fix landed on the component, not on every surface that repeats the same information — see §3. |
| 4 | Discount card shows one end-condition | **PASS** | Host console discount card now reads "Weekday off-peak (2–5pm) · **ends after 10 bookings**" — a single condition, matching `EndAfterOccurrences`. The round-1 artifact's "...or Sep 30" compound language is gone; no residual mismatch against the domain model's mutually-exclusive `EndCondition` variant. |
| 5 | Genuine light **and** dark theme parity | **FAIL** | Same underlying defect as #1, called out separately because it's exactly the failure mode the task brief predicted in advance: a fixed hex text color against a themed (`var()`-based) background looks right in the theme it was eyeballed against (dark: 6.6–7.8:1) and silently fails in the other (light: 3.87–4.50:1). Every other color in the stylesheet is token-based and theme-aware (`--ink`, `--ink-soft`, `--paper`, `--paper-raised`, `--line`, `--court`, `--court-soft`, `--accent`, `--accent-ink`); only these four pill text values are the exception, and they're the exact four the round-1 fix touched. |

---

## 2. New finding — the round-1 contrast fix was only verified in dark theme

Round 1 flagged the pre-fix pill CSS as a likely WCAG 1.4.3 fail and
recommended a solid-fill treatment. The implementer did switch to solid
fills, but picked **one fixed near-black hex per status** (`#06251b` for
paid, `#2b1600` for unpaid, `#2c0a05` for full/flag) rather than a
token that resolves differently per theme. Worked contrast ratios,
computed from the actual `--success`/`--warning`/`--critical` hex values
in each theme block of the artifact's own `<style>`:

| Pill | Light bg | Text | Light ratio | Dark bg | Text | Dark ratio |
|---|---|---|---|---|---|---|
| `.paid` | `#2f8f6f` | `#06251b` | **≈4.10:1 — fail** | `#59c79f` | `#06251b` | ≈7.84:1 — pass |
| `.unpaid` | `#b8722a` | `#2b1600` | **≈4.50:1 — borderline** | `#e0a250` | `#2b1600` | ≈7.76:1 — pass |
| `.full` / `.flag` | `#c14f3f` | `#2c0a05` | **≈3.87:1 — fail** | `#e28174` | `#2c0a05` | ≈6.61:1 — pass |

(`.waitlist` uses `background: var(--ink-soft); color: var(--paper-raised)`
— both tokens, and it computes comfortably above 4.5:1 in both themes, so
it isn't part of this finding; it's the model the other three should
follow.)

Three of four status pills — the exact component round 1 asked to be
fixed, and the one the Designer dossier's own §3.2 flags as the piece
most likely to get copy-pasted into three clients (Vue/Swift/Kotlin) as
a shared badge component — read fine in dark theme and read wrong (two
of them clearly failing, one sitting exactly on the line) in light
theme. This is not a new category of problem; it's the *same* problem
round 1 found, not actually closed, discoverable only by checking both
themes rather than trusting the "solid fill" description. **The fix:
swap the four hardcoded hex values for either (a) two token-aware values
per status (a light-mode-safe dark ink, a dark-mode-safe light ink,
switched the same way `--ink`/`--paper` already are), or (b) a single
sufficiently-dark ink chosen against the *lighter* of the two theme
backgrounds (light theme's `--success`/`--warning`/`--critical` are all
lighter/more saturated than dark theme's), which would pass both.**

---

## 3. Other second-pass findings

- **iPad condensed list repeats the bare-ratio bug the pill fix was
  supposed to close** (§1 item 3). The underlying data point ("12/16")
  recurs in the iPad split-view's list card outside the `.pill` markup
  and wasn't touched by the round-1 fix. Recommend fixing this as a
  copy/token issue, not a second design decision — the same "X/Y seats"
  string should be the one source of truth for this figure everywhere
  it's rendered, not a `.pill`-only convention.
- **"Verify link" (camera field) is a `<span>`, not an interactive
  element at all** — no `role`, `tabindex`, or keyboard handler, so it's
  not reachable or operable by keyboard, a stronger failure than round
  1's "borderline target size" framing implied. Not a new regression —
  round 1 checked its *size*, not whether it's a real control — and it's
  consistent with every other CTA in this artifact (`Publish game`,
  `Save court settings`, `Confirm & pay`, `Register` are all
  `<span class="btn ...">`, not `<button>`/`<a>`). Given the toggle *was*
  correctly upgraded to a real `<button role="switch">` this round, the
  artifact's convention is inconsistent: interactive semantics were
  added exactly where round 1 named the gap and nowhere else. Worth a
  one-line note in the artifact (or the doc) that CTA markup here is
  illustrative, not a preview of final Vue/Swift markup, so a T7
  implementer doesn't copy `<span>` buttons verbatim.
- **The "Changed since round 1" callout is legible in both themes**
  (computed ≈7.24:1 light, comfortably higher dark) — no accessibility
  problem in the callout itself. **Recorded disagreement (Designer vs.
  PM), not resolved:** Designer would trim it to a one-line pointer
  ("4 fixes landed, 2 open — see `review-round-1.md`") since it adds
  ~80 words above the fold before the actual open-questions grid, which
  is exactly the kind of "extra unit of information competing with the
  relevant units" the Designer dossier's §3.1 (*Refactoring UI*)
  principle warns about. PM sees no cost here — this is an internal
  review artifact, not a shipped product surface, and a reviewer
  skimming for what changed benefits from the detail being inline rather
  than one hop away. Neither treats this as blocking.
- **Round-0 doc's provisional-decision framing (GuestCount, camera
  consent) reads clearly enough to act on.** Each hedge in the top
  blockquote names a specific working assumption and its trigger
  condition ("revisit if the user answers differently") rather than just
  expressing uncertainty — that's what a T7 ticket-writer actually needs
  to proceed. PO's one polish suggestion, not a blocker: the blockquote
  currently folds two unrelated open decisions into one paragraph; a
  two-row table (`Decision | Working assumption | Status`) would be
  faster to scan at ticket-writing time than prose, but the current
  version is not actually unclear, just denser than necessary.
- **New — T6/discount cross-reference gap, not previously raised.**
  `docs/process/t6-sprint-plan.md` T6.1 defines `domain.Payment.Amount`
  as a plain caller-supplied `Money{Cents, Currency}` value with no
  opinion on how the amount was derived — structurally this is
  compatible with a discounted price once `discount_rules` (still
  design-doc-only, §3.2 of the system-design doc; not on any sprint's
  backlog yet) ships, since `Payment` doesn't compute prices, it only
  records them. T6 is correctly silent on discounts today — there is no
  discount-aware pricing code path yet for a Payment to be wrong about.
  But nothing in T6.1–T6.4's acceptance criteria states, as a forward
  pointer, that **a Payment for a discounted booking/registration must
  carry the discount-resolved amount, not the base `pricing_rules`
  amount** — so when `discount_rules` is eventually scoped (T7+), the
  natural failure mode is a PR that wires `CreateOnlinePayment`/
  `RecordOfflinePayment` straight off `ResolvePrice`'s un-discounted
  result, because nothing currently forces the discount step into that
  path. Recommend adding this as an explicit acceptance-criterion line
  to whichever ticket builds `discount_rules`, not to T6 (T6 has no
  discount concept to be wrong about, so this is not a T6 defect) — but
  log it now so it isn't rediscovered the hard way after Payments and
  discounts are both live.

---

## 4. Round 2 verdict

**Proceeds to round 3 — not converged.** The task brief's convergence
rule is "two consecutive rounds find nothing material." Round 1 found
several material things; round 2 found one more: the round-1 contrast
fix does not actually hold in light theme for three of four status
pills (two clear fails, one on the line), which is the same underlying
defect round 1 raised, not yet closed, just newly precise about *why*
it isn't closed. That is a concrete, fixable CSS-token issue (§2's
recommendation), not a direction problem, and it should not require a
third design pass to agree on — but it does need to actually be fixed
and re-verified in both themes before this badge pattern is treated as
settled and copied into the Vue/Swift/Kotlin component set, per the
Designer dossier's own warning against accessibility-as-afterthought.

Also carrying forward, unchanged from round 1: the two provisional
answers (`Registration.GuestCount` mutability; camera-consent
attestation wording) are still explicitly pending the user's direct
sign-off, not resolved by this round, and don't block round 3 from
proceeding on the same working-assumption basis round 1 established.

One recorded disagreement this round (§3, Designer vs. PM on the
changelog callout's length) — scoped and non-blocking, not a
direction-level split.
