# Design Review — Pickleball Platform Spec

**What this is.** A simulated roundtable in which the six roles from the Operating Handbook review the platform spec, argue, and surface decisions for you. It's role-play in one pass, not six independent agents — so treat the *findings and decisions* as the product, not the theatre. Each role was pointed at the part of the spec it's paid to be skeptical about.

**Cast.** PE = Principal Engineer · PM = Product Manager · PdE = Product Engineer (your builder hat) · QA · PO = Product Owner · BA = Business Analyst.

---

## Roundtable

### Topic 1 — "Both loops in v1" (your stated choice)

**PM:** I want to challenge this. Booking and Social Play are *both* core contexts. As a solo dev, shipping both means you validate neither quickly. Which side of the marketplace can you actually seed first — do you have a facility that'll list, or organisers who'll host?

**PE:** Agreed on the risk, but I'll defend partial overlap: both loops sit on Facilities, Courts, Pricing, and — critically — Booking. If we model that shared base once, the second loop is cheaper than it looks. It's not two products, it's one spine with two heads.

**PdE:** That's my instinct too. I'd rather build the spine and one head fully, then bolt on the second head, than build two half-heads.

**PO:** Then the honest version of "both" is *sequenced*: spine + one complete loop shippable, second loop as the next increment. Not both in parallel.

**BA:** One caution — the two loops don't share as cleanly as it sounds unless we fix a modelling gap first (Topic 2). If we don't, "reuse" becomes "double-booking bug."

*Lean:* build the shared spine, ship **one** loop first, sequence the other. Which loop goes first depends on which side you can seed — an open question for you.

### Topic 2 — The Booking / Game modelling gap (High severity)

**BA:** In the spec, a direct **Booking** reserves a Court/Slot, and a **Game** is "hosted at a Facility on booked Court(s)." But nowhere does it say a Game's court usage *is* a Booking. If Games hold courts through a separate mechanism, the no-overlap constraint in §6 won't see them.

**PE:** That's a real hole, and it's the dangerous kind — invisible until launch day. Decision: a Game reserving a court must create Booking rows (or the same exclusion-constrained range table). One reservation concept, one invariant, no exceptions. Recurring Club hires too.

**QA:** Confirming with a test: two requests at the same instant — one direct booking, one game — for the same court/slot. Exactly one must win. If the spec keeps them separate, that test fails and I can't sign off.

**PdE:** Cheap to fix now, expensive later. I'll take it.

*Lean:* **not really optional** — represent every court reservation (direct, game, recurring hire, competition) as the same Booking aggregate so one invariant covers them all.

### Topic 3 — The matchmaking engine and rating cold-start

**PM:** Auto-matchmaking is a differentiator; I don't want to gut it.

**QA:** Then tell me how I test "fair." Winner-vs-winner and loser-vs-loser pairing is deterministic and testable. "Balanced mix of new vs experienced" is a judgement call — I can test the code runs, not that players feel it's fair.

**BA:** Worse: the whole engine keys off wins/losses and rating, and a brand-new player has *none*. On day one every player is unrated, so "match by skill" is noise. We need a self-reported starting level, or the engine has nothing to work with.

**PE:** So split it. v1 = manual match entry + score capture + the two deterministic pairings (W-v-W, L-v-L), seeded by a self-reported level. The clever balancing and custom rules come after there's real rating history to make them meaningful.

**PM:** I can live with that if manual entry ships first — hosts get value immediately and the engine improves as data accumulates.

*Lean:* ship **manual matches + scores + simple deterministic pairing + self-reported starting level** first; defer the sophisticated auto-balancing until ratings have history.

### Topic 4 — Optional payments + Stripe Connect timing (Med–High)

**QA:** "Optional in-app pay *and* manual offline marking" doubles the payment state space. My edge list: offline-marked-then-Stripe-captured, refund-after-paid, capacity-change-mid-payment, game cancelled with some paid. Each is a real bug waiting to happen.

**PE:** And taking money is a one-way door. Stripe Connect means KYC on payees, payout timing, refund liability, tax — before you've proven anyone wants the product. That's a lot of gravity for an unvalidated marketplace.

**PM:** Counter: paid games reduce no-shows, which is a top host pain. Payment isn't a nice-to-have, it's part of the value.

**PO:** Compromise that keeps both happy: v1 tracks paid/unpaid with **manual marking only** (offline). The roster and statements work identically. Add real Stripe Connect capture as a fast-follow once a facility is actually live. Same data model, half the launch risk.

**BA:** That also lets me get the refund/cancellation rules right on paper before real money makes mistakes costly.

*Lean:* keep the **paid/unpaid tracking** (it's cheap and valuable), but consider launching with **manual/offline marking first** and switching Stripe Connect on once validated — your call, since you explicitly want in-app pay.

### Topic 5 — Web + native mobile, solo

**PE:** Expo shares code, but web + iOS + Android still means app-store review cycles, push infrastructure, and device testing — three release surfaces for one person.

**QA:** And Detox mobile e2e is the flakiest, highest-maintenance tier in the pyramid. Solo, it'll rot.

**PdE:** I feel the pull to ship native, though — it feels more "real."

**PO:** The spec already offers PWA-first as the scope-reducer. I'd take it: installable web app at launch, native apps as a later phase once the loops are validated. Reach barely suffers; bandwidth roughly halves.

*Lean:* **PWA/web first**, native apps as a later phase.

### Topic 6 — Social RSVP bot + data collection (Low–Med)

**BA:** The bot is already de-risked to Phase 4 in the spec (own-channel, in-app links) — no objection, just keep it out of the critical path.

**PE:** On "collect data about paddle use, revenues, social play" — keep it first-party product analytics with consent, and don't quietly build a profile store. That's a privacy and trust liability that buys little early on.

*Lean:* fine as scoped; add an explicit consent/data-minimisation note.

---

## Consolidated findings

| # | Finding | Raised by | Severity | Recommendation |
|---|---------|-----------|----------|----------------|
| F1 | Game/recurring-hire court usage not modelled as Bookings → double-booking hole | BA, PE, QA | **High** | Unify all court reservations under one Booking aggregate + the §6 invariant |
| F2 | "Both loops in v1" is high-risk for a solo dev | PM, PO | High | Build shared spine; ship one loop first, sequence the other |
| F3 | Matchmaking keys off ratings that don't exist at launch (cold-start) | BA, QA | Med | Self-reported starting level; ship manual + simple pairing first |
| F4 | Optional-pay + offline-marking doubles payment edge cases; Stripe is a one-way door pre-validation | QA, PE | Med–High | Keep tracking; consider offline-marking first, Stripe as fast-follow |
| F5 | Three release surfaces (web/iOS/Android) overload a solo dev | PE, QA | Med | PWA/web first; native later |
| F6 | Data collection lacks consent/minimisation framing | PE | Low–Med | First-party analytics with consent; no silent profiling |

---

## Decisions for you to make

These are genuine forks; the roundtable's lean is noted but the call is yours.

- **D1 — Which loop ships first?** Depends on the side you can seed. *Blocked on your answer:* do you have any pilot facilities or organisers lined up?
- **D2 — Matchmaking scope in v1.** Manual + simple pairing now (roundtable lean), or hold out for the full auto engine?
- **D3 — Payments at launch.** Offline paid/unpaid tracking first with Stripe as fast-follow (lean), or Stripe Connect from day one as you originally wanted?
- **D4 — Platforms.** PWA/web first (lean), or web + native together?

## Recommended spec changes (adopt if you agree)

1. **F1 (strongly recommend):** add a rule to §8 that *every* court reservation — direct booking, game, recurring hire, competition — is represented as a Booking, so §6's invariant covers all of them.
2. **F3:** add a self-reported starting skill level to the player profile and to matchmaking inputs.
3. **F6:** add a one-line data-consent/minimisation principle to §4.
4. Re-order §9 phases to reflect whichever loop wins D1, and to put manual matches (D2) and offline payment tracking (D3) ahead of their fancier versions.

Approve any of these and I'll fold them into `pickleball-platform-spec.md`.

---

## Resolution note (bootstrap session, 2026-07-31)

The locked decisions in `CLAUDE.md` / `pickleball-platform-spec.md` §Revision 2 show how the open forks above were actually resolved: **F1 was adopted in full** (D3b, the polymorphic Booking aggregate — this is now a locked invariant, not a suggestion). **F2/D1** was resolved as "full scope is v1, but the *build order* is spine-first, one context at a time" (§9) — the backlog in `HANDOFF.md` (T1–T6) is that sequencing in practice: Booking spine lands before Social Play (T5) and Payments (T6). **D2/F3** was resolved toward the roundtable's lean at the domain-modelling level (self-reported starting level is a locked field) but the *product* decision kept the full automated-matchmaking ambition for later; T5's AC explicitly scopes T5 itself to the Game aggregate + booking integration and defers matchmaking to a follow-up task, which is the F3 lean applied to sequencing rather than to cutting scope. **D3/F4** was resolved as "both modes from the data-model's point of view" (paid/unpaid is one source of truth regardless of mode) with T6 sequencing the Stripe adapter behind an anti-corruption layer so the offline path can ship and be tested before real Stripe credentials are wired in — a middle path between the roundtable's offline-first lean and the original online-first ask. **D4/F5** was resolved as web + native together (not PWA-first) per D4a's stack lock — accepted as a conscious risk given the solo-dev constraint; mitigated by keeping all domain logic server-side so Vue/Swift/Kotlin stay thin.
