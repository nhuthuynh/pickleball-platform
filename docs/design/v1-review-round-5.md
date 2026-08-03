# Design Review — Round 5

**Reviewing:** round 4's two assigned fixes (facility/pricing screen-caption
still asserting the pre-fix "inherits" framing; "Bracket preview — Pool A"
label on a round-robin format) against the companion visual artifact (same
URL, republished — `https://claude.ai/code/artifact/bcfb62bc-d157-44dc-95da-5166f4480ab9`),
verified against the artifact's actual HTML/CSS source (fetched directly —
full `<style>` block and full `<body>` markup for every section, saved and
grepped, not inferred from its own "Round 1–5" changelog callout, per the
task's standing instruction), plus a fresh end-to-end read of all five flows
looking specifically for the recurring "fix on one surface, stale claim
survives on another" failure pattern, plus an independent cross-check of the
two still-open product/legal items across all five places they're mentioned.
**Roles played:** UX/UI Designer, Product Manager, Principal Engineer,
Product Owner — same four dossiers as rounds 1–4.
**Round:** 5 of up to 10.

---

## 1. Verdicts on the two assigned checks

| # | Check | Verdict | Reason |
|---|---|---|---|
| 1 | Facility/pricing screen-caption accurately describes the club discount as its own rule, not inherited | **PASS** | The `screen-caption` paragraph under Flow 1 now reads: "...and — **as of round 4** — club recurring hire as its own discount rule with its own rate and end condition, **not inherited from the individual discount above it**." The word "inherit" appears exactly once in the entire artifact, in this corrected form (`grep`-verified) — the pre-fix "inheriting the facility's discount config" sentence round 3 and round 4 both flagged is gone, and nothing else on the page repeats the old framing. |
| 2 | "Bracket preview — Pool A" relabeled to avoid claiming round robin produces a bracket | **PASS, with a caveat** | The `divider-label` above the seeded-entries list now reads "Pool A entries — seeded by level" — no "Bracket preview" anywhere in the markup. But see §2: the word "bracket" survives one paragraph lower, in the same section, describing this same round-robin competition, which is exactly the pattern that motivated this fix in the first place. |

---

## 2. New finding — the competition screen-caption still calls a round-robin competition "a bracket on top"

The round-4 finding, verbatim from that round's review: *"A round-robin pool
doesn't produce a bracket (no elimination pairing)."* Round 5 fixed the
label that made this claim directly (`Bracket preview` → `Pool A entries —
seeded by level`). But the `screen-caption` paragraph closing out the same
Flow-4 section — written before this round's own changelog entry explains
the fix — still reads:

```html
<p class="screen-caption">
  Reuses the game-creation screen's advertising pattern exactly (same
  shareable-link, in-app-RSVP model, same §7.05 resolution) rather than
  inventing a second one — a competition is a Booking/Registration-shaped
  thing with a bracket on top, not a different kind of advertising problem.
</p>
```

This screen depicts one specific competition — "Riverside Summer Ladder,"
format "Round robin, 4 pools of 4" — and the caption's own sentence is
describing *that* competition ("a competition is a ... thing with a bracket
on top"), immediately below a divider label this same round just renamed
specifically because round robin doesn't produce a bracket. The artifact's
own intro changelog, two sections above, states the fix rationale in those
exact words: "renamed 'Bracket preview' to 'Pool A entries — seeded by
level' on the competition screen, **since round robin doesn't produce a
bracket**." The screen-caption directly below the fixed label still asserts
the opposite claim about the same on-screen competition.

This is the same failure shape the task brief asked round 5 to hunt for,
and the same shape round 2 (bare "12/16" surviving outside the `.pill`
component) and round 4 (the stale "inherits" caption) already found: a fix
lands on the component/label that was named as broken, and a second,
unrelated-looking piece of copy nearby — written earlier, not touched by the
fix — keeps repeating the pre-fix claim. It's a smaller instance than round
4's (one clause, not a full contradicting paragraph, and "a bracket on top"
could charitably be read as a general statement about the `Competition`
aggregate rather than this specific pool-play instance) — but the artifact
gives no signal that it's speaking generally rather than about the
round-robin ladder pictured two inches above, and a reader skimming the
caption for "what does this screen prove" would come away thinking this
round-robin competition has a bracket, which round 5's own label change
says it does not.

**Recommendation:** drop "with a bracket on top" from the screen-caption
(the sentence's actual point — reuse of the advertising pattern — doesn't
need it) or replace it with format-neutral language ("a Booking/
Registration-shaped thing with pools or a bracket depending on format, not
a different kind of advertising problem"), then re-grep the whole artifact
for "bracket" to confirm no other surface repeats the claim — which this
round did (§ nothing else found, see below).

---

## 3. Full end-to-end consistency sweep, all five flows

Read the entire artifact fresh (facility/pricing, game creation + roster,
player discovery/registration/level on iPhone, iPad adaptive, competition),
cross-checking every mention of capacity, discount/club terms, and the
club/individual distinction:

- **Capacity ("X/Y seats").** Appears four times — Flow 2 tablet roster
  pill, Flow 2's `caption-block` prose, Flow 3 iPhone discovery card pill,
  and the iPad condensed-list `cond` text. All four read "12/16 seats,"
  none bare. Consistent with rounds 2–3's fix; no regression found.
- **Discount / club vs. individual.** The only two discount cards
  (individual "25% off... ends after 10 bookings," club "20% off... no end
  date") both appear once, only on Flow 1, with a divider label ("a
  separate discount rule, not the same one"), a field hint (independent
  rate/end-condition, no stacking), and now a corrected screen-caption —
  all four say the same thing, all four checked. No other flow mentions a
  discount or a club rate, so there's no second surface for this specific
  claim to drift on.
- **"Bracket."** Exactly one stale instance — §2 above. No other flow uses
  the word.
- **Sample player data reused across flows** (Mara T. appears in Flow 2's
  roster as "+1 guest, Paid" and in Flow 4's Pool A entries at level 3.7;
  the level card in Flow 3 shows "3.7 · 27 games played · 18 wins")  is
  internally consistent, not contradictory — same name, same level number,
  plausible as the same illustrative player reused across mockups rather
  than a data conflict.
- **Result:** one new instance found (§2), the same class as rounds 2 and
  4's findings, smaller in severity. No other capacity, discount, or
  club/individual inconsistency across the five flows.

---

## 4. Cross-check — the two open product/legal items, all five mentions

Checked `Registration.GuestCount` mutability and the camera-consent
attestation wording everywhere both the companion doc and the artifact
state them:

| Location | GuestCount | Camera consent | Still "provisional, pending user"? |
|---|---|---|---|
| Doc top blockquote (lines 1–16) | "assumed **set-once**... put to the user directly and not yet confirmed" | "assumed to require a **facility-onboarding consent/signage attestation checkbox**... put to the user directly and not yet confirmed" | **Yes, both** |
| Doc §3.1 (camera) | — | "**Working assumption (put to the user, not yet confirmed)**: facility onboarding requires a consent/signage attestation checkbox" | **Yes** |
| Doc §3.3 (GuestCount) | "**Working assumption (put to the user, not yet confirmed)**: `GuestCount` is set-once at registration time" | — | **Yes** |
| Doc §7 | "Two of these (1 and 4) have a **provisional-default answer pending explicit user sign-off** — see the note at the top of this document." | same sentence, item 1 | **Yes, both** |
| Artifact's intro `caption-block` | "whether guest count is editable after registering... **pending the user's product/legal call**" | "the exact camera-consent attestation wording — **pending the user's product/legal call**" | **Yes, both** |

All five locations agree: both items are still explicitly framed as
provisional, working assumptions, pending the user's direct sign-off — none
has drifted toward implying either is settled. No finding here.

---

## 5. Round 5 verdict

**Not converged — proceeds to round 6.** Items 1 and 2 (the two fixes round
5 was assigned to verify) both pass on their own terms. Item 4's five-way
cross-check of the two open user-decision items found no drift — all five
locations still agree the items are provisional.

But §2's sweep found one more instance of the exact failure class this
review keeps finding: the Flow-4 screen-caption's "a bracket on top" clause
survives, unedited, one paragraph below the label this same round renamed
specifically because round robin doesn't produce a bracket — the artifact's
own changelog states the fix rationale two sections above the sentence that
still contradicts it. Smaller in severity than round 4's fully-contradicting
paragraph (a clause, not a paragraph; arguably general phrasing rather than
a claim about this specific screen), but real, concrete, and grep-verified,
not hypothetical.

Per the convergence rule ("two consecutive rounds find nothing material"):
round 4 was material. Round 5 is also material (§2). **This makes zero
consecutive clean rounds, not one** — round 5 does not reset toward
convergence, it restarts the counter at zero exactly as round 4 did to
round 3's counter. Round 6 would need to find nothing material, and round 7
after it, before this converges — one clean round is never sufficient on
its own.

Unchanged from rounds 1–4, still independently open regardless of this
round's findings: **`Registration.GuestCount` mutability** and **the
camera-consent attestation wording** remain explicit product/legal
decisions pending the user's direct sign-off — confirmed still accurately
and consistently described as provisional in all five places checked (§4),
not resolved by any design round, and not blocking round 6 from proceeding
on the same working-assumption basis established in round 1.
