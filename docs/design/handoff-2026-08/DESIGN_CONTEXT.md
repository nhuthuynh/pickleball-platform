# Design Context — for Claude Code subagents

This file is written for AI agents (not just human developers) to load into context when brainstorming UI/UX for this project. Read this alongside `README.md` (screen-by-screen spec) and the `.dc.html` design file (visual reference) and `screenshots/` (quick visuals without opening HTML).

## Product framing
A pickleball marketplace with four contextual roles on one account system: Player, Host, Facility Owner, Club. The central tension to brainstorm around: **the same person moves between roles constantly** (a Player becomes a Host of their own game; an Owner also Hosts games on their own courts). UI/UX should minimize friction when switching context rather than forcing separate "modes" or logins.

## Where brainstorming is most valuable
1. **Role-switching UX** — how does someone go from "I'm joining a game" to "I'm hosting one" without feeling like a different app? Is it a persistent toggle, a per-action prompt, or contextual (the same booking screen just gains host controls)?
2. **Auto-matching transparency** — host sets level/gender matching rules; players should understand why they were matched/not matched. Open design question: how much visibility into the matching algorithm should players get?
3. **Social-media ad → registration loop** — a host posts to WhatsApp/Facebook/X/Instagram, replies become registrations. This is the least conventional flow here. Brainstorm: how does the host reconcile registrations arriving from 4 different channels plus in-app joins into one roster view without confusion (see wireframe in `06-competitions-social.png`)? What happens with ambiguous replies (e.g., a reply that isn't clearly a registration)?
4. **Pricing complexity surfaced simply** — an owner can stack base price + monthly rate + multiple discount periods (each with a different end condition: date, booking count, or ongoing) per court. The wireframe (`04-pricing-config.png`) shows a list + calendar preview. Brainstorm whether that's enough, or whether conflict resolution (two overlapping discounts) needs its own UI.
5. **Cash vs online payment signaling** — cash bookings are "pending" until a host marks them paid. Brainstorm how prominently pending/unpaid cash bookings should surface to hosts (dashboard? notification?) so nothing falls through the cracks.

## Constraints already decided (don't re-litigate these)
- Roles are contextual on one account — not separate account types.
- A facility can be added by any Host/Owner if it doesn't already exist on the platform.
- Payment method (online/cash/either) is set per game/booking by the host/owner.
- Auto-matching considers level and gender; host can override/update at any time.
- Guest allowance (how many friends a registrant can bring) is host-configured per game.
- Platforms: responsive web + iPad + iPhone from one shared design system — no platform-specific redesigns.

## Open questions worth a subagent's attention
- Player Level formula (tenure + win rate) — no weighting decided yet.
- Whether social-media reply registration is fully automated or needs host confirmation (platform API/ToS dependent).
- Online payment processor and payout mechanics to owners/hosts — unspecified.
- Conflict handling when a club's recurring request overlaps an existing booking.

## How to use this in a Claude Code session
Drop this whole `design_handoff_pickleball_platform/` folder into the repo (e.g. `docs/design/`). Point subagents at `DESIGN_CONTEXT.md` first for the framing and open questions, then `README.md` for per-screen detail, then the screenshots/HTML for visual reference. Treat the HTML file's visuals as illustrative, not final pixel spec — see Fidelity note in README.
