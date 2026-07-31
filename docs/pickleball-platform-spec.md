# Pickleball Courts & Community Platform — Requirements & Architecture Spec

**Build context:** solo developer who can code.
**v1 goal:** both loops — court discovery/booking *and* social games with matchmaking.
**Payments:** optional in-app payment, with paid/unpaid tracking surfaced to hosts (social games) and owners (monthly/recurring).

**Assumptions to confirm:** (1) single launch market, built i18n/currency-ready for later expansion; (2) revenue = commission on in-app payments + optional owner subscription. Correct these and the spec shifts accordingly.

---

## Revision 2 — locked decisions

- **D1 — Scope:** v1 includes the full set — court management, hosting games, joining games (both loops, not sequenced). The build *order* still starts with the shared spine (see §9); "everything in v1" is the scope, not a claim that it all ships on day one.
- **D2 — Matchmaking:** automated from each player's historic data (games/wins/losses/rating), always overridable manually by a Host, **Game Admin**, or platform Admin. New players are seeded with a self-reported starting level so the automation has an input from game one.
- **D3 — Payments:** both modes. Online via Stripe Connect (pay in app or on the website); or offline in person, where a Host / **Game Admin** / Owner / Admin records the amount. Paid/unpaid tracking is the single source of truth regardless of mode.
- **D3a — New role, Game Admin:** each Game (and Competition) can have one or more **Game Admins** who manage and direct players on the day and may record offline payments. Distinct from the Host/Organiser (who owns the Game) and the platform Admin.
- **D3b — Booking is polymorphic:** four reservation types all resolve to one **Booking** aggregate so the no-double-booking invariant covers them all — (1) Club monthly recurring hire, (2) individual court booking, (3) Host booking court(s) to run a Game, (4) Host booking court(s) for a Tournament/Competition.
- **D4 — Platforms:** native mobile apps **and** a web app. Technology options for this are explored in the companion doc `technology-options.md`.
- **D4a — Language direction (locked):** **Go** API backend (Chi/stdlib, pgx, sqlc); **Vue 3** web frontend (public pages pre-rendered/SSG for SEO, app as SPA); **Swift/SwiftUI** native **iOS** app on the Go API. Domain logic lives once in the Go backend; Vue and Swift are thin clients. This supersedes the TypeScript-everywhere baseline in §2. *Open sub-decisions:* Go↔Swift pattern (native Swift on the API vs `gomobile bind` for on-device Go logic), and whether Android (a later Kotlin app) is in scope — see `technology-options.md` §2.

*Still open:* do you have a pilot facility or organiser lined up? It doesn't change the architecture, but it decides which side of the marketplace you court first at launch.

---

## 1. Guiding principle for a solo build

You cannot hand-build auth, file storage, realtime, payouts, and infra alone and still ship. The stack below pushes all of that onto managed services so you write product code, not plumbing. Everything is TypeScript so there's one language across web, mobile, and backend logic.

---

## 2. Recommended stack

> **Superseded by `technology-options.md` (Go-centric revision).** Per your decision to centre on Go, the backend + web stack is now Go (Chi/stdlib + pgx + sqlc + templ/HTMX) over managed Postgres/PostGIS, with the native app handled by a separate pattern (see that doc). The table below is the earlier TypeScript-everywhere baseline, kept for comparison.

| Layer | Choice | Why |
|---|---|---|
| Backend backbone | **Supabase** (Postgres + PostGIS, Auth, Storage, Realtime, Row Level Security, Edge Functions) | Removes ~60% of solo backend work; Postgres is the right DB for this relational, geo-heavy data |
| Web | **Next.js** (React, TypeScript) on **Vercel** | SSR gives you SEO on court/facility pages — critical for discovery |
| Mobile | **Expo / React Native** (TypeScript) | Shares types and logic with web; one codebase for iOS + Android |
| Monorepo | **Turborepo** + shared `packages/` (types, API client, validation with Zod) | Write a booking type once, use it everywhere |
| Payments | **Stripe Connect** (destination charges) | Handles KYC, split payments, payouts, refunds; your commission = application fee |
| Geo search | **PostGIS** ("courts near me", radius filters) | Built into Supabase Postgres |
| Bots / jobs | **Supabase Edge Functions** + a small **Telegram/Discord** or WhatsApp Business bot | For RSVP capture on channels you control (see §7) |
| Notifications | **Expo Push** / FCM, **Postmark** (email), **Twilio** (SMS/WhatsApp) | |
| Guide videos | Embed **YouTube/Vimeo** | Never self-host video |
| Product analytics | **PostHog** | Covers your data-collection goal; add a warehouse (BigQuery) only if you outgrow it |
| Testing | **Vitest** (unit), **Playwright** (web e2e), **Detox** (mobile), **k6** (load) | |
| CI/CD | **GitHub Actions**; web → Vercel, mobile → **EAS** | |

**Solo-friendly option worth considering:** ship the player/organiser experience as a **PWA (installable web app) first**, add native apps in a later phase. It halves your surface area at launch. You lose some native polish and push reliability, which you can add when the product is validated.

---

## 3. Functional requirements by domain

### 3.1 Users & roles
- Roles: player, host/organiser, **game admin**, facility owner, club, platform admin. A user can hold several (a player can also host).
- **Game Admin** (D3a): assigned per Game/Competition by the Host; manages and directs players on the day, and may record offline payments. Scoped to the games they're assigned to — not a platform-wide admin.
- Auth (email + social login), profile, **self-reported starting skill level** (seeds matchmaking before any history exists), play history.
- Owner verification/onboarding (identity + Stripe Connect onboarding before receiving payouts).

### 3.2 Facilities & courts
- Owners create a facility: name, facility number, address (geocoded), website, images, floor plan, entry instructions, "how to get in".
- Courts within a facility, each numbered, with attributes (indoor/outdoor, surface).
- Court map per facility (positions/layout).

### 3.3 Pricing
- Pricing rules per court (or inherited from facility): by day-of-week, time window, weekend rate. Rules resolve to a price for any given slot.

### 3.4 Booking & availability
- **One Booking, four sources (D3b):** every court reservation is the *same* Booking underneath — (1) club monthly recurring hire, (2) individual court booking, (3) host booking court(s) for a Game, (4) host booking court(s) for a Tournament/Competition. This is what lets the no-double-booking invariant cover all of them.
- **Hard requirement: no double-booking** across all four sources (see §6).
- Payment per booking is online (Stripe) or offline (recorded by Host/Game Admin/Owner/Admin) — see §5.
- Cancellation windows, refunds, no-show handling, waitlists.
- Owner availability/blackout management; iCal export.

### 3.5 Clubs & recurring hire
- Owners let a club hire specific courts on recurring days (weekly recurrence rules).
- Club sees its schedule; owner sees the recurring revenue and payment status.

### 3.6 Social games (the community core)
- An organiser creates a game: facility, courts, date/time, capacity, skill rules, price (optional).
- Players register in-app or via a social link (see §7); each registration tracks source and payment status.
- Host view: roster with **paid / unpaid** per player.
- Host can see, from confirmed numbers, how many courts to hire.

### 3.7 Matchmaking & scoring (D2)
- **Automated from historic data** — matches generated from each player's games played, wins, losses, and rating:
  - Winner-vs-winner and loser-vs-loser random pairings (king-of-the-court / Swiss style).
  - Balanced mixes of new vs experienced players.
  - Custom rule sets per game.
- **Cold-start:** a player with no history is seeded by their self-reported starting level (§3.1) so the automation always has an input.
- **Manual override always available:** a Host, Game Admin, or platform Admin can add, change, or re-pair matches and enter scores by hand at any time.
- Scores feed each player's rating and W/L record (DUPR-style internal rating), which feeds the next round of automation.

### 3.8 Competitions
- Organiser sets up a competition with a format, opens registration, manages brackets/rounds, records results.

### 3.9 Statements & analytics
- Owner: monthly per-facility statement — earnings and expenses.
- Host: per-game statement — expenses and earnings.
- Platform data collection: court utilisation, paddles used, hosting activity, revenue, social-play participation, competition stats.

### 3.10 Discovery & content
- Search courts/games by location, date, price, indoor/outdoor, skill level; "near me".
- If a facility isn't listed, a user can create it (address, photos, court numbers, hours, entry instructions) — queued for owner claim/verification.
- Guide videos (how to play, choosing a paddle).
- Paddle reviews with affiliate links, sourced from reputable reviewers with ratings.

---

## 4. Non-functional requirements

- **Security & privacy:** Row Level Security on every table; PII minimised; payment data never touches your servers (Stripe-hosted); comply with your market's data-protection law.
- **Data collection with consent (F6):** the paddle/revenue/social-play data you want is collected as first-party product analytics with user consent and clear purpose — no silent profiling; minimise and anonymise where the insight doesn't need identity.
- **Correctness under load:** booking is bursty and concurrency-sensitive — see §6.
- **Availability & observability:** error tracking (Sentry), structured logs, uptime monitoring.
- **Performance:** fast geo search (spatial index), cached pricing resolution.
- **Accessibility:** WCAG AA.
- **Localisation:** timezone-correct everywhere (store UTC, render local), currency-ready.
- **Maintainability:** shared types, schema-validated inputs (Zod), migrations in version control.

---

## 5. Payment design (matches your "optional pay + paid tracking" requirement)

- Every bookable thing (court booking, game registration, recurring hire, subscription) can optionally accept online payment.
- **If online pay is enabled:** payer checks out via Stripe → status set to `paid` automatically. Your commission is the Connect application fee; the rest routes to the owner/host's connected account.
- **If offline:** status starts `unpaid`; a Host, **Game Admin**, Owner, or platform Admin records the amount and marks `paid` when cash/transfer is received. Same status field, one source of truth regardless of mode.
- **Host view (social game):** roster showing paid/unpaid per registrant.
- **Owner view:** monthly payers (club hires, subscriptions, recurring bookings) with paid/unpaid and outstanding totals.
- A single `payment` record links to Stripe references, payer, payee, type, and status so statements (§3.9) are just aggregations over it.

---

## 6. Preventing double-booking

Use a Postgres exclusion constraint so the database itself refuses overlapping bookings on the same court — safer than app-level checks under concurrency:

```sql
-- bookings.during is a tstzrange of [start, end)
ALTER TABLE bookings
  ADD CONSTRAINT no_overlap
  EXCLUDE USING gist (
    court_id WITH =,
    during   WITH &&
  ) WHERE (status <> 'cancelled');
```

Wrap booking creation in a transaction; a conflicting insert fails cleanly and you show "just taken, pick another slot".

---

## 7. Social promotion & RSVP (the feature to rethink)

Auto-monitoring public WhatsApp/Facebook/Instagram/X chats to count RSVPs is largely against those platforms' terms and often technically impossible (WhatsApp's API only sees messages to your own number; the Facebook Groups API is closed to third parties; X/Instagram APIs are restricted and paid). **Verify current terms before relying on any of this.**

Design that gets the same outcome safely:
- Each game generates a shareable link/card the organiser posts to any platform manually or via official share APIs.
- RSVPs land **in-app** through that link — always an accurate count.
- Optionally run a bot on a channel **you control** (a Telegram/Discord group, or a WhatsApp Business number people message) that parses a simple format like `IN 2` and updates the roster.
- The host's "courts needed" number is then computed from confirmed in-app RSVPs.

---

## 8. Core data model (starter entities)

`users` · `facilities` · `courts` · `pricing_rules` · `bookings` · `game_admins` · `games` · `game_registrations` · `competitions` · `matches` · `player_ratings` · `payments` · `statements` · `paddle_reviews` · `social_posts` · `analytics_events`.

Key relationships:
- A facility has many courts; courts have pricing rules.
- **`bookings` is polymorphic (D3b):** a booking has a `source` of `recurring_hire | individual | game | competition` and (where relevant) a nullable link to the game/competition/club it belongs to. There is no separate reservation table — this is what makes the §6 no-overlap invariant cover every reservation type. (A recurring hire is a template that generates bookings; a game/competition holds its courts *as* bookings.)
- Games belong to a facility, hold their courts as `game`-source bookings, and have zero or more **`game_admins`**.
- Registrations belong to games and carry payment status and source (app/social).
- `users` carry a self-reported starting level; `matches` belong to a game or competition and update `player_ratings`.
- Every payable action writes one `payments` row (online or offline); statements are aggregations over it.

---

## 9. Suggested delivery phases

Per D1 the whole thing is v1 scope. This is the **build order** that keeps the shared spine first so nothing is built twice — not a gate that drops later loops. Each phase is independently shippable so you can put something in real users' hands early.

- **Phase 0 — Foundations:** auth, roles (incl. Game Admin), monorepo, the polymorphic Booking model, facility + court CRUD, geo search, Stripe Connect onboarding.
- **Phase 1 — Booking spine:** availability, pricing rules, the four booking sources, no-double-book constraint, online (Stripe) + offline payment recording.
- **Phase 2 — Social games:** create/join games, registration + paid tracking, automated matchmaking with manual override, score entry, ratings.
- **Phase 3 — Clubs, competitions, statements:** recurring hire, competition brackets, owner/host statements, analytics dashboards.
- **Phase 4 — Content & growth:** social share links + RSVP bot on your own channel, guide videos, paddle reviews + affiliate links.

---

## 10. Testing & deployment

- **Unit:** pricing resolution, matchmaking algorithm, statement aggregation (Vitest).
- **Integration:** booking + payment flows against a test Supabase branch and Stripe test mode.
- **E2E:** Playwright (web), Detox (mobile) for the book-a-court and join-a-game happy paths.
- **Load:** k6 on the booking endpoint to prove the double-booking constraint holds under concurrency.
- **CI/CD:** GitHub Actions runs tests on PR; merge deploys web to Vercel and mobile via EAS; DB changes go through versioned migrations.

---

## 11. Open risks to keep in view

- **Marketplace chicken-and-egg:** you need owners *and* players. Decide which side you can seed first (e.g., partner one or two facilities before launch) — this shapes go-to-market, though the build order in §9 stays the same either way.
- **Legal:** liability waivers for games, insurance expectations, and marketplace/money-transmission rules depend on your market — get local advice before taking payments (relevant from Phase 1, since Stripe is in scope).
- **Solo bandwidth (native + web, full scope):** you chose the widest surface, so the biggest lever now is *code reuse* — share the domain layer across web/native/backend (see `technology-options.md`) so features are written once. Ship phase by phase to keep it real.
- **Social APIs:** re-verify platform terms before building anything that touches them.

---

## 12. Decisions

**Resolved (Revision 2):** scope = full/both loops (D1); matchmaking = automated + manual override (D2); payments = Stripe + offline recording (D3); platforms = native + web (D4).

**Still open:**
1. Launch market / currency (drives Stripe setup and legal).
2. Commission model and rough percentage; owner subscription yes/no.
3. Pilot facility or organiser lined up? (decides which side of the marketplace you court first — go-to-market, not architecture).
