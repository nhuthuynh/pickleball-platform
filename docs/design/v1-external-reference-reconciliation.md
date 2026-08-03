# Reconciling the external design handoff against the v1 design workstream

**Source:** `docs/design/handoff-2026-08/` — a design-session export ("Court&Play
— System Design · V1 · Aug 2026") the user attached, covering the same scope
this project's own 10-round design review (`docs/design/v1-system-design.md`,
`docs/design/v1-review-round-*.md`) already worked through. This note
reconciles the two rather than treating the attachment as a second, competing
design — same workstream tag (`v1`), same month, overlapping scope.

## What confirms the existing design (no action needed)
Facility onboarding with camera links, court pricing with a discount period +
end condition (date / booking count / ongoing), social game creation with
cancellation cutoff + payment method + guest allowance + level/gender
auto-matching, discover/join flows, competitions advertised via linked
social accounts, club recurring rentals at a discount, online vs. cash
payment, and a responsive web/iPad/iPhone target — all match what the
10-round review already worked through. Concrete breakpoints from the
attachment (`Platform Notes`, screenshot 09) are worth adopting verbatim
since nothing in the existing docs specified numbers: **web ≥1280px**
(persistent sidebar nav), **iPad 768–1180px** (two-column list+detail,
collapsible nav, ≥44px touch targets), **iPhone <600px** (single-column,
bottom tab bar: Discover / Bookings / Games / Profile, full-screen modal
forms).

## One direct, load-bearing conflict — same as before, now doubly confirmed
The attachment's own `DESIGN_CONTEXT.md` lists as an open question: "whether
social-media reply registration is fully automated or needs host
confirmation (platform API/ToS dependent)." This is the identical tension
`docs/pickleball-platform-spec.md` §7 and the v1 design review's §4 already
flagged and resolved toward: **in-app RSVP via a shareable link is the
answer, not generic reply-parsing** — a bot on a channel the platform
controls (WhatsApp Business number, Telegram/Discord) can parse a
constrained format, but scraping/monitoring public replies on
WhatsApp/Facebook/X/Instagram is not being built. Two independent passes
landing on the same "this needs to stay in-app, not become reply-scraping"
answer is corroboration, not new information — carried forward as the
sprint's working assumption below, not re-litigated.

## New: things the attachment raises that the prior 10 rounds didn't
`DESIGN_CONTEXT.md`'s "Where brainstorming is most valuable" section names
five UX questions the prior design rounds never asked, because the prior
rounds were reviewing screens, not open product/UX questions. Recorded here
as **inputs to sprint planning below**, not resolved unilaterally:

1. **Role-switching UX.** The same account moves between Player, Host,
   Owner, and (per this attachment, more explicitly than before) **Club**
   constantly. Does the UI need a persistent role toggle, a per-action
   prompt, or fully contextual controls (the same booking screen just gains
   host controls when you're the host)? This is a real information-
   architecture decision the sprint needs to make, not defer.
2. **Auto-matching transparency.** How much visibility should a player get
   into *why* they were matched/not matched by the level/gender algorithm?
3. **Cash-vs-online payment surfacing.** A cash booking is "pending" until
   a host marks it paid — how prominently should unpaid/pending cash
   bookings surface to hosts (dashboard badge? notification?) so nothing is
   silently forgotten?
4. **Pricing-conflict UI.** The v1 design review already decided a discount
   *modifies* the resolved price and settled the domain-model shape
   (`EndCondition`: date / occurrence-count / ongoing, mutually exclusive).
   Still open: does the *UI* need its own conflict-resolution surface if an
   owner stacks two overlapping discount periods, or is preventing the
   overlap at entry time (a validation error, not a runtime resolution UI)
   sufficient? Leans toward the latter (simpler, matches ADR-0002's
   "ambiguity is a domain error, not silently prioritized" precedent for
   pricing rules) — recommended, not yet decided by a review round.
5. **Club as an explicit account type.** The attachment's domain model
   names **Club** as a fourth contextual role/account type (alongside
   Player/Host/Owner), not just "a recurring-hire relationship" as the
   existing docs implied. Worth adopting explicitly — it clarifies who can
   *request* a recurring discounted slot (a Club account) versus who
   *approves* it (the Facility Owner), which the existing design left
   implicit.

## Where this goes next
These five items are exactly the kind of open question a PM+PE backlog
refinement ceremony resolves into ticket-shaped decisions, not something
this reconciliation note should decide alone. See the T7 sprint plan
(`docs/process/t7-sprint-plan.md`) for how each was actually scoped.
