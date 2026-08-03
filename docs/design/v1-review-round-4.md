# Design Review — Round 4

**Reviewing:** round 3's two open-ended findings (no competition screen despite
claimed coverage; club discount not distinctly configurable) against the
companion visual artifact (same URL, republished —
`https://claude.ai/code/artifact/bcfb62bc-d157-44dc-95da-5166f4480ab9`),
verified against the artifact's actual HTML/CSS source (fetched directly —
full `<style>` block and full `<body>` markup for every section — not
inferred from its own "Round 1–4" changelog callout, per the task's standing
instruction), plus a fresh open-ended pass over
`docs/design/v1-system-design.md` §1's 16 requirements against
every screen in the artifact.
**Roles played:** UX/UI Designer, Product Manager, Principal Engineer,
Product Owner — same four dossiers as rounds 1–3.
**Round:** 4 of up to 10.

---

## 1. Verdicts on the four assigned checks

| # | Check | Verdict | Reason |
|---|---|---|---|
| 1 | Competition-scoped screen exists and holds up on its own merits | **PASS** (one naming nit) | A new section, "Host console — competition setup & advertising" (Flow 4), exists with `<h2>` copy that names it as the round-4 fix for requirement #9. Bracket preview seeds by level in descending order (Mara 3.7, Devon 3.6, Priya 3.5), matches the declared "3.0–4.5, self-seeded" level range, and the fourth slot is shown honestly as `<div class="roster-row">4. Open slot <span class="pill waitlist">Unfilled</span></div>` — an admitted gap, not faked as full. The advertising block is the *same* markup pattern as the game-creation screen: identical four `.social-chip` rows (WhatsApp/Facebook/X/Instagram, same brand-color swatches), identical "shareable link" copy, and a hint line that explicitly says "Same rule as a social game's ad card... we don't read replies posted on these platforms" — a real, stated reuse, not a second inconsistent pattern independently invented. **Naming nit, not a functional defect:** the format field reads "Round robin, 4 pools of 4," but the preview above it is labeled "Bracket preview — Pool A." A round-robin pool doesn't produce a bracket (no elimination pairing) — the numbers are internally consistent (3 seeded + 1 open = pool of 4), only the section label borrows tournament-bracket language for a pool-play format. Cheap fix, worth doing before this label reaches a ticket. |
| 2 | Club discount is genuinely distinct from the individual discount | **PARTIAL** | The club discount card now shows its own numbers — 20% off → $14.40/hr vs. the individual card's 25% off → $13.50/hr — and its own end condition (`NoEnd`, "no end date — standing club rate") vs. the individual rule's `EndAfterOccurrences` ("ends after 10 bookings"). The field's own hint line explains non-stacking clearly: "a club member booking outside the recurring block pays the regular (or individual-discount) rate, not the club rate." That fully satisfies the literal ask. **But** the `screen-caption` paragraph at the bottom of the *same* Flow-1 screen was not updated and still reads: "...club recurring hire shown as **inheriting the facility's discount config** rather than a separate price a club negotiates ad hoc" — word-for-word the framing round 3 flagged as the actual bug. See §2 — a screen that shows one thing and, four inches lower, describes the opposite thing is not a clean fix. |
| 3 | No regression in rounds 1–3's fixes (pill tokens, real `<button>`) | **PASS** | The new screen's only interactive control, "Publish competition," is `<button type="button" class="btn primary block">` — a real button, matching every other CTA in the artifact, not a reintroduced `<span>`. Its pills use only already-established, theme-aware patterns: the "Unfilled" pill is `class="pill waitlist"` (the same `--ink-soft`/`--paper-raised` pairing verified passing in round 3, unchanged), and the level pills use `style="background:var(--court); color:var(--paper);"` — themed CSS custom properties, not a hardcoded hex and not the old `color-mix(...)` tint pattern. No token collision, no hex regression. |
| 4 | Bracket-seeding pills (3.7/3.6/3.5) reusing the `var(--court)` treatment | **Genuine split — recorded, not resolved** | See §3. |

---

## 2. New finding — the club-discount screen contradicts itself

Round 3's exact complaint was that the artifact's own caption described the
club rate as "inheriting the facility's discount config rather than a
separate price a club negotiates ad hoc." That sentence is **still present,
unedited**, in round 4's HTML:

```html
<p class="screen-caption">
  <strong>Reads the requirements literally:</strong> ... and club recurring
  hire shown as inheriting the facility's discount config rather than a
  separate price a club negotiates ad hoc.
</p>
```

Four inches above it, the actual fixed markup now reads:

```html
<div class="divider-label">Club recurring hire — a separate discount rule, not the same one</div>
...
<div class="field">
  <label>Club discount rule</label>
  <div class="price-card">
    ... <span class="pill" ...>20% off</span>
    <p class="cond">Recurring block only (Tue/Thu 6–8pm) · no end date — standing club rate</p>
  </div>
  <p class="hint">Its own rate and end condition, set independently of the weekday
  off-peak discount above...</p>
</div>
```

The divider label and the field hint say "separate rule." The
screen-caption paragraph below still says "inheriting... rather than a
separate price." **These directly contradict each other on the same
screen.** This reads as exactly what it probably is — the discount card and
its inline hint were edited to fix round 3's finding, but the
summary caption at the bottom of the section (written before the fix, and
not part of the component that changed) was never revisited. A host or a
future reviewer skimming only the caption — which is plausible, it's the
one-paragraph summary of "what this screen proves" — would walk away with
the round-3-era, *wrong* understanding of how the club rate works, even
though the UI directly above it is now correct.

**This is a real, if small and cheap-to-fix, defect**, not a nitpick about
prose style: it's the same failure mode this whole review loop exists to
catch — a fix that's correct in one place and unverified everywhere the same
claim is restated (the same shape as round 2's "12/16" bare-ratio finding,
which was also a fix landing on one surface and not on a second surface
repeating the same fact). Recommend: delete or rewrite the trailing clause
of the screen-caption to match the divider label and field hint, then
re-verify no other surface in the artifact repeats the pre-fix framing.

---

## 3. Item 4, in full — the pill-reuse question, with a real split recorded

The task asks for a genuine verdict, not "could go either way." Read
carefully, the four roles do not converge, and manufacturing agreement here
would misrepresent the review.

**First, a fact that changes the framing of the question as posed:** the
`style="background:var(--court); color:var(--paper);"` treatment is **not**
newly borrowed from the roster-capacity pill for this screen. It is already
the artifact's general-purpose "neutral highlight" pill, in use since round
1 for at least three distinct kinds of information before round 4 added a
fourth: the discount-percentage pill ("25% off," "20% off"), the
capacity-ratio pill ("12/16 seats"), and now the player-level pill ("3.7,"
"3.6," "3.5"). It sits deliberately apart from the four *semantic-status*
pills (`--pill-success-bg`/`--pill-warning-bg`/`--pill-critical-bg`/
`waitlist`), which carry real meaning (paid/unpaid/full/waitlist) and were
the subject of rounds 1–3's contrast work. So the real question isn't "was a
capacity-count style reused for a rating," it's "should this one shared
'neutral info tag' component keep absorbing more distinct data types," which
is a slightly different, and slightly less alarming, question than the task
brief poses.

- **Designer's position:** still a real problem, even reframed. A discount
  percentage, a seat ratio, and a player skill rating are three different
  *kinds* of number a user has to disambiguate by content alone, because
  the pill gives no differentiating affordance (no icon, no label prefix,
  same shape, same color) — this is precisely NN/g heuristic #4
  ("consistency and standards... don't make users wonder if different
  situations mean the same thing") cutting *against* reuse here, not for
  it: visual consistency across genuinely different meanings teaches users
  the wrong lesson (that pill = "this number is like that other number").
  The bracket-preview context (a numbered list, "1. Mara T." next to "3.7")
  makes the meaning locally inferable today, but that's exactly the kind of
  "reads fine in the one place we designed it" reasoning that produced the
  round-1→round-2 pill-contrast miss. Recommend a `.pill.rating` variant —
  same tokens, a small label change (e.g., a leading "Lvl" or a distinct
  border treatment) — before this becomes the shared badge component three
  clients copy.
- **Principal Engineer's position:** this is a mockup, and the component
  already has exactly the right shape for this: one reusable "neutral
  info tag" versus four purpose-built "status" tokens. Inventing a fifth
  variant for one more data type, at this stage, before three real contexts
  (Social Play, Competitions, Pricing) have all landed and shown whether
  this is actually a recurring collision, is the over-engineering anti-
  pattern the PE dossier explicitly names — "functionality that isn't
  presently needed by the system." If real usage testing later shows users
  actually confuse a level number for a capacity ratio, add the variant
  then, backed by that evidence, not preemptively.
- **PM's position:** low product risk either way — this is a review
  artifact, and in the actual shipped screen a rating pill will always sit
  next to a player name in a roster/bracket row, never next to a bare
  number with no label, so the realistic confusion scenario Designer
  describes doesn't obviously occur in the real IA. Not worth spending a
  round-5 cycle on.
- **PO's position:** agrees with PE/PM that this doesn't block ticket-
  writing, but flags it as exactly the kind of item that belongs in a
  `.pill` component's *acceptance criteria* once Competitions actually gets
  a ticket (a one-line AC: "rating pills are visually distinguishable from
  count/ratio pills") rather than being silently dropped — cheap to write
  down now, easy to forget once the mockup phase is behind everyone.

**Recorded, not resolved:** Designer treats this as a real, if minor,
mixed-signal problem worth a distinct component variant before three
clients copy the pattern; PE/PM/QA-adjacent reasoning (PM, PO) treat it as
reasonable, low-cost reuse of an already-established "neutral tag" pattern
that doesn't warrant a new variant without evidence of actual user
confusion. Both readings are defensible from the same source; this review
does not manufacture a single verdict where the roles genuinely differ.

---

## 4. Open-ended pass — requirement coverage, full 16-item re-check

Re-checked every row of `v1-system-design.md` §1 against every
screen in the round-4 artifact (not just the two rows round 3 flagged):

| # | Requirement | Status this round |
|---|---|---|
| 1 | Court booking | Existing, correctly not re-illustrated. |
| 2 | Facility desc/photos/address, camera link | Covered — Flow 1. |
| 3 | Date/time + monthly + discount w/ end condition | Covered — Flow 1. |
| 4/5/6/13 | Hosts, rules, payment method, multi-player games | Covered — Flow 2. |
| 7 | Host adds a facility that doesn't exist yet | **Still no screen** — but this is the same, previously-acknowledged status as round 3 (the source doc's own mapping already marks this "not yet built, no ticket" as existing un-illustrated scope, not new-requirement scope this round claims to cover). Not a new gap, not a regression. |
| 8 | Club recurring-hire discount | **Now covered**, distinctly configurable — see §1 item 2 and §2's caveat. |
| 9 | Competitions + social-media advertising | **Now covered** — Flow 4, new this round. See §1 item 1. |
| 10 (post ads) | Link social accounts to advertise | Covered, unchanged — shareable-link pattern, now used identically on two screens. |
| 10 (reply-to-register) | Social-reply RSVP | Explicitly flagged as the resolved conflict (§7.05 resolution), unchanged. |
| 11 | Friend/plus-one registration | Covered — Flow 2, Flow 3. |
| 12 | Cancellation cutoff, capacity | Covered — Flow 2 (per-Game total, as before). |
| 14 | Skill level from tenure+wins | Covered — Flow 3 "Your level." |
| 15 | Level- and gender-aware matchmaking | Covered — Flow 2 toggles. |
| 16 | Web / iPhone / iPad | Covered — Flow 3 iPhone, adaptive iPad section. |

**Result: full coverage of all 16 requirements achieved**, modulo
requirement #7, whose absence is deliberate, already-acknowledged, pre-
existing scope (not a claim-vs-reality gap the way #9 was) and was
explicitly carved out as such in round 3. No new "claims coverage it
doesn't have" gap was found this round — the intro copy's claim of covering
"competitions advertised through social media" is now actually true against
the markup, which it was not in round 3.

---

## 5. Round 4 verdict

**Not converged — proceeds to round 5.** The convergence rule is "two
consecutive rounds find nothing material." Round 3 found two material,
direction-level gaps (no competition screen despite claimed coverage; club
discount not distinctly configurable). Both are now substantively fixed —
item 1 and the core of item 2 are clean passes, and the bracket/advertising
screen holds up well on its own merits, not just as "a screen exists now."

But round 4 is **not** a clean round either: §2's finding (the Flow-1
screen-caption still asserts the pre-fix "inherits the facility's discount
config" framing, directly contradicting the corrected discount card four
inches above it, on the same screen) is a real, concrete, easily-verified
defect — the same *shape* of miss as round 2's bare-"12/16" finding (a fix
landing on one surface, not on every surface repeating the same claim), not
a hypothetical. That resets the "two consecutive clean rounds" counter to
zero rather than starting it at one.

§3's split on the rating-pill reuse question is recorded, not resolved, per
this project's standing "don't manufacture consensus" rule — it should not
by itself block progress, but should be carried into whichever ticket
defines the shared `.pill` component, per PO's suggested acceptance
criterion.

Unchanged from rounds 1–3, still independently open regardless of this
round's findings: **`Registration.GuestCount` mutability** and **the
camera-consent attestation wording** remain explicit product/legal
decisions pending the user's direct sign-off — not resolved by any design
round, and not blocking round 5 from proceeding on the same
working-assumption basis established in round 1.
