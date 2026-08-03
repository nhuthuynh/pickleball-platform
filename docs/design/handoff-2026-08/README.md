# Handoff: Pickleball Booking Platform — System Design

## Overview
System design for a pickleball marketplace: facility owners list courts, hosts (who can be owners) run social games and competitions, players discover and join games, and clubs book recurring discounted slots. Covers roles/permissions, domain model, facility onboarding, court pricing/discounts, social game creation with auto-matching, competitions with social-media advertising, payments, and club rentals. Target platforms: responsive web, iPad, iPhone.

## About the Design Files
The bundled HTML file is a **design reference** — an interactive prototype/document built to communicate structure, flows, and layout. It is not production code to copy directly. The task is to **recreate this design in the target codebase's existing environment** (React, Vue, SwiftUI, native, etc.) using its established component patterns, state management, and libraries — or, if no environment exists yet, choose the most appropriate framework and implement there.

## Fidelity
**Low-to-mid fidelity.** This is a system-design document with schematic flow diagrams and illustrative wireframes (placeholder data, simplified controls) — not pixel-perfect final UI. Use it as a guide for information architecture, entity relationships, and screen structure. Apply the target codebase's existing design system (or a newly chosen one) for final visual styling, spacing, and typography rather than matching this file's exact pixel values.

## Roles & Permissions
Role is contextual per account, not a fixed account type:
- **Player**: books courts, finds/joins social games, brings guests, has a Level derived from tenure + win record.
- **Host**: rents courts to run social games/competitions, sets game rules and payment method, advertises via linked social accounts. A Player can become a Host.
- **Facility Owner**: lists a facility, configures court pricing/discounts, links security cameras, approves club rental requests, and can also host games directly.
- **Club**: an organization account that books recurring days/times at an owner-configured discount tier.

See the permission matrix in the design file (Section 01) for the full capability-by-role breakdown.

## Domain Model
Entity clusters (high-level, no schema):
- **Identity**: Account (profile with contextual Player/Host/Owner capability), Player Level (derived, feeds auto-matching).
- **Facility & Courts**: Facility (description, photos, address, camera links, owner), Court (belongs to a Facility), Pricing Rule (date/time price, monthly price, discount period + end condition — per court).
- **Games & Bookings**: Booking (court reserved for a date/time by a player, host, or club), Social Game (rules: cancellation window, capacity, payment method; matching config; sits on a Booking), Participant (a player + guest count joined to a game), Payment (online or cash, tied to a booking/game).
- **Growth**: Competition (organized by a host, reserves courts across dates), Social Account Link (WhatsApp/Facebook Group/X/Instagram, connected once per host), Ad Post (auto-generated promo published to linked accounts; replies become registrations), Club (recurring bookings at owner-configured discount).

Core relationship spine: `Account (Owner) → Facility → Court → Booking → Social Game or Payment`.

## Screens / Flows

### 1. Facility Onboarding (Owner/Host)
- **Purpose**: Add a new facility if it doesn't exist yet, then make it bookable.
- **Flow**: Search & add (create if not found) → Details (name, description, address) → Photos → Cameras (optional security camera link) → Courts & go live.
- **Key fields**: facility name, description, address (map pin), camera connections list (+ add), photo gallery grid, courts list with per-court pricing status ("Priced" / "Needs pricing"), "+ Add court" action.

### 2. Court & Pricing Configuration (Owner)
- **Purpose**: Define what each court costs and when discounts apply.
- **Flow**: Base pricing (price by date & time-of-day block) → Monthly rate (optional flat monthly access price) → Discount period (date range + price, with an end condition: on a specific date, after N bookings, or never/ongoing) → Preview calendar → Publish.
- **Key fields**: list of discount periods (name, date range or ongoing, price, end condition badge), monthly rate field, calendar preview color-coded by standard/discounted/monthly-holder.

### 3. Social Game Creation (Host/Owner)
- **Purpose**: Configure and publish a social game on a rented or owned court.
- **Flow**: Court & time → Capacity & format (max players, singles/doubles) → Rules (cancellation cutoff, payment method: online/cash/either) → Guests (max friends per registrant) → Matching & publish (auto-match by level/gender toggle, host can override anytime).
- **Key fields**: cancellation cutoff text field, payment-method 3-way selector, guest allowance stepper, auto-match toggle, level-range slider, gender-mix selector.

### 4. Discover & Join Games (Player)
- **Purpose**: Find and join social games.
- **Flow**: Browse & filter (location, date, level, gender mix) → Review game (host, court, level range, spots left, price) → Join + guests (pay online or mark cash) → Play & level up (result updates Level for future matching).
- **Key fields**: filter chips (location, date, level range, gender mix), game cards (title, spots-left badge, host/court/time/level line, price or "Cash at facility", Join button).

### 5. Competitions & Social Media Advertising (Host)
- **Purpose**: Organize a competition and promote it through linked social accounts; replies register players.
- **Flow**: Create competition (name, format, dates, courts, entry fee) → Link accounts (one-time connect, reused for every post) → Publish ad (auto-formatted summary + registration link) → Replies register (reply or tap link, with guest count) → Manage roster (host confirms, schedules, brackets).
- **Key fields**: connected-accounts list (WhatsApp Group, Facebook Group, X, Instagram — each Connected/Connect), "Publish ad" action, registrations list showing name, guest count, and source channel (WhatsApp reply / App / Facebook reply).

### 6. Payments
- **Purpose**: Apply and settle the payment method chosen per game/booking.
- **Two methods**: Online (charged at booking/join time, instant confirm, auto-refund within free-cancel window) and Cash (reservation held pending, settled in person, host marks paid after the game).
- **Flow**: Booking/game created → Payment method applied → Settled & recorded in host ledger.

### 7. Club Rentals
- **Purpose**: A club requests a recurring discounted slot across one or more facilities.
- **Flow**: Request recurring slot (facility, courts, days, time, date range) → Discount applied (per owner's club/bulk pricing config) → Owner approves (or auto-approves within policy) → Bookings generated (recurring, discounted, club manages its own roster).
- **Key fields**: weekday picker, time & date-range field, price summary (standard rate, club discount %, resulting recurring rate), approval-status badge.

## Design Tokens (from the reference file — treat as illustrative, not final)
- **Colors**: background `oklch(98% 0.006 95)`, surface `#ffffff`, text `oklch(22% 0.01 95)`, muted text `oklch(48% 0.012 95)`, border `oklch(89% 0.012 95)`, accent (green) `oklch(56% 0.135 155)` with soft tint `oklch(94% 0.035 155)`, accent2 (orange) `oklch(58% 0.135 45)` with soft tint `oklch(94% 0.035 45)`.
- **Typography**: Inter (400/500/600/700/800) for UI text; JetBrains Mono for placeholder/code-style labels.
- **Radius**: 6–14px depending on component size (chips use full pill radius).
- **Role badges**: pill-shaped, color-coded per role (Player: neutral grey, Host: green, Owner: orange, Club: outlined).

## Platform Notes
One shared design system and component set across breakpoints, not divergent designs:
- **Web (≥1280px)**: multi-column dashboards, calendar + list side-by-side, persistent sidebar nav for hosts/owners.
- **iPad (768–1180px)**: two-column list + detail where space allows, collapsible nav, touch targets ≥44px.
- **iPhone (<600px)**: single-column stacked flows, bottom tab bar (Discover, Bookings, Games, Profile), full-screen modal forms.

## Assumptions & Open Questions
- Camera integration assumes facilities connect an existing system (partner API or stream link) — not a platform-owned camera product.
- Social platforms may restrict automated reading of replies; some ad flows may need the host to confirm registrations manually rather than fully automatically.
- Payment processor for "online" is unspecified — assumed a standard card/wallet processor with platform payout to owners/hosts.
- Player Level formula (tenure + win rate) needs a product decision on weighting before auto-matching logic is finalized.

## Files
- `Pickleball Platform System Design.dc.html` — full system design document (this handoff's source of truth for flows, entities, and wireframes).
- `screenshots/` — section-by-section captures of the design document, for quick visual reference without opening the HTML file:
  - `01-overview-roles.png` — overview + roles/permissions matrix
  - `02-domain-model.png` — domain model / entity clusters
  - `03-facility-onboarding.png` — facility onboarding flow + wireframe
  - `04-pricing-config.png` — court & pricing configuration flow + wireframe
  - `05-social-games.png` — social game creation, auto-matching, discover/join wireframes
  - `06-competitions-social.png` — competitions & social media advertising flow + wireframe
  - `07-payments.png` — payments (online vs cash)
  - `08-club-rentals.png` — club recurring rental flow + wireframe
  - `09-platform-notes.png` — web/iPad/iPhone responsive notes
