# Technology Stack — Locked (Go API · Vue web · Swift iOS)

The stack is now chosen. Domain logic lives in **one place — the Go backend** — and the web (Vue) and native (Swift) apps are thin clients over its API. That's a clean architecture: the booking rules, pricing, matchmaking, and payment state machine are written and tested once, server-side, and never duplicated across clients.

**Locked**
- **Backend / API:** Go (Chi or stdlib `net/http`), pgx + sqlc, Postgres + PostGIS, stripe-go.
- **Web:** Go serves a **Vue 3** frontend. Public discovery pages are pre-rendered (SSG) for SEO; the logged-in app is a Vue SPA.
- **Native (iOS):** **Swift / SwiftUI**, calling the Go API. (Optional `gomobile bind` only if you want on-device Go logic — see §2.)
- **Deploy:** single Go binary + managed Postgres; Vue built assets served by the Go binary.

**Two open sub-decisions** (small, but they change the build): which Go↔Swift pattern (§2), and whether Android is in scope (Swift is Apple-only).

---

## 1. The stack, layer by layer

**Backend (Go) — the core.** Goroutines suit the concurrency-heavy parts (simultaneous booking attempts, live rosters, live scores). Single static binary → trivial deploys. `go test` + table-driven tests make TDD tight for pricing and matchmaking. The **no-double-booking invariant is enforced in Postgres** (spec §6 exclusion constraint), not in Go — the database owns it regardless of language.

**Web (Go + Vue).** Go is the API and static-asset host; **Vue 3** is the UI. The SEO trap to avoid: a bare client-rendered SPA doesn't rank, and discovery pages must rank. Resolution:
- **Pre-render / SSG the public pages** (court/facility/public-game pages) so they ship as static, crawlable HTML — served straight from the Go binary. Single deploy, SEO intact.
- Keep the **logged-in app as a Vue SPA** (SEO doesn't matter behind auth).
- Alternative if you prefer framework-level SSR: **Nuxt** — but it needs a Node runtime beside the Go service, adding a second deploy. The SSG-public + SPA-app split avoids that.

**Native (Go + Swift).** Swift/SwiftUI for a genuinely native iOS app. See §2 for how Go and Swift combine.

---

## 2. How Go and Swift combine (pick one)

| Pattern | What it is | When to choose |
|---|---|---|
| **(a) Swift app on the Go API** (recommended default) | Native Swift/SwiftUI client calling the Go REST/JSON API over HTTP, exactly like the web. No `gomobile`. | Almost always — you keep Go where it's strong and get a fully native app with the least complexity |
| **(b) `gomobile bind`** | Compile shared Go domain logic into an `.xcframework` and call it from Swift, running booking/matchmaking logic on-device | Only if you need offline or client-side domain logic; boundary types are limited, tooling is niche — verify current maintenance status first |

**Recommendation:** **(a)** unless on-device Go logic is a real requirement. It's simpler, more maintainable solo, and loses nothing given the domain already lives in the Go API.

**Apple-only caveat:** Swift covers iOS (and macOS). **Android is a separate Kotlin app** if/when you want it — it can share the same Go core via `gomobile bind` (path b) or just call the same API (path a). Decide whether iOS-first is intentional.

---

## 3. Languages

- **Go** — backend/API and shared domain logic.
- **SQL** — schema, the no-overlap invariant, and queries via **sqlc** (type-safe Go from SQL; SQL-first suits DDD).
- **TypeScript / JavaScript** — the Vue web frontend.
- **Swift** — the native iOS app.

Reuse across these three is the **API contract only** — so make that contract first-class (see §4).

---

## 4. Frameworks & libraries

**Backend (Go)**
- HTTP: stdlib `net/http` (1.22+) or **Chi**.
- Data: **pgx** + **sqlc**; **Postgres + PostGIS**; migrations via **goose** / golang-migrate.
- Payments: **stripe-go** + Stripe Connect, behind an anti-corruption layer.
- Auth: your own JWT/sessions, or a managed auth service; realtime via SSE or `coder/websocket`.
- Background jobs: goroutines + a queue (River on Postgres, or asynq on Redis) for statements, webhooks, RSVP parsing.

**Web (Vue)**
- **Vue 3** + **Vite**; **Pinia** for state; **Vue Router**.
- **Vite SSG** (or Nuxt if you go SSR) for the public discovery pages.
- **Tailwind** for styling.

**Native (Swift)**
- **SwiftUI** + `URLSession` (or a light client like Alamofire).

**The contract (do this early)**
- Define the API as an **OpenAPI spec** and generate typed clients: for Vue (e.g., `openapi-typescript` / orval) and for Swift (e.g., swift-openapi-generator). One source of truth keeps three stacks in sync and claws back most of what you lose by not sharing a language.

---

## 5. Coding style & architecture (DDD + TDD)

- **Domain lives in the Go backend**, framework-free: pricing resolution, matchmaking pairing, the payment-status state machine, booking rules as plain Go — trivially unit-testable and the heart of the TDD pyramid. Vue and Swift stay thin (UI + API calls), so there's one authoritative model.
- **Ports & adapters:** the Go domain declares interfaces; pgx/Stripe/notifications are adapters. Swapping a provider touches an adapter, not the domain.
- **Package per bounded context** (booking, socialplay, payments, pricing, facilities…), translating at the seams.
- **sqlc** keeps queries type-safe and the schema authoritative.
- **Ubiquitous language** in names across DB, Go domain, API, Vue, and Swift — a `Booking` is a `Booking` everywhere.
- **TDD tooling:** `go test` + table-driven tests (ideal for pricing bands and matchmaking permutations); **testcontainers-go** or a disposable Postgres for integration; **k6** to prove no-double-booking under concurrency. Front-end: **Vitest** + component tests (Vue), **XCTest** (Swift). E2E: **Playwright** (web), **XCUITest** (iOS).
- **Discipline:** `gofmt` + golangci-lint; ESLint/Prettier for Vue; SwiftLint; Conventional Commits; trunk-based development.

---

## 6. Deployment

- **Go API + Vue assets:** one Go binary serves the JSON API and the built Vue static/SSG bundle → minimal container (distroless/scratch) → **Fly.io**, **Cloud Run**, **Render**, **Railway**, or any VPS. Single deploy. (If you choose Nuxt SSR instead, deploy the web app separately on a Node host.)
- **Database:** managed Postgres with PostGIS — **Supabase**, **Neon**, or **Fly Postgres**.
- **iOS:** Xcode build → **TestFlight** → App Store; you'll need a paid Apple Developer account and to budget for review cycles.
- **CI/CD:** **GitHub Actions** — `go test` + lint + Vue tests on every PR; build binary/container and iOS app on merge; migrations applied through the pipeline; Stripe in test mode outside prod.
- **Environments:** `dev → staging → prod`, separate Postgres per environment.
- **Observability from day one:** Sentry (Go + Vue + iOS), structured logs (slog), uptime monitor.

---

## 7. The trade-off in one paragraph

This is the widest surface you can pick solo — three stacks (Go, Vue, Swift) with reuse limited to the API contract, and iOS-only until you add a Kotlin Android app. You accept that in exchange for a concurrency-native Go backend that fits this domain unusually well, a mainstream and pleasant Vue frontend, and a fully native iOS experience. The two moves that keep it sane: keep **all domain logic in the Go backend** so it's written and tested once, and make the **OpenAPI contract first-class** so the Vue and Swift clients generate from one source instead of drifting. Get the Vue public pages pre-rendered for SEO and pick the Go↔Swift pattern (§2) deliberately, and this is a coherent, buildable architecture.
