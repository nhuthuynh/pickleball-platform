# Requirements Research — Consolidated Synthesis

Synthesis of the four sourced research passes below, produced per the
user-approved "requirements gathering first" sequencing before sprint
mechanics (Scrum board setup, per-role knowledge expansion, sprint-loop
rules) begins. Each underlying doc verifies part of
`docs/pickleball-platform-spec.md` against real external sources — this
file cross-references their findings, resolves overlaps, and ranks what
needs a decision before it becomes backlog/ticket work.

- `research-functional.md` — competitor platforms, DUPR, pickleball formats, booking edge cases (18 findings, 61 sources)
- `research-security-compliance.md` — OWASP API Top 10, PCI-DSS/Stripe, OAuth2/OIDC/Auth0, GDPR/CCPA (~20 findings, 20 sources)
- `research-performance-availability.md` — SRE SLOs, Postgres EXCLUDE/GiST under load, k6 methodology, observability (4 sections, 13 sources)
- `research-accessibility-i18n.md` — WCAG 2.2 AA, timezone/DST, currency/i18n, mobile accessibility (~14 findings, code-cross-checked)

No new research was done in this file — it only cross-references and
prioritizes the findings already sourced in the four docs above.

---

## Cross-cutting pattern noticed across three of the four docs

Three independent research passes converged on the **same shape of bug**
without being asked to look for it: an invariant or resolution keyed to
the wrong identity.

- **Functional research (§1.5/§4.4):** the `EXCLUDE (court_id WITH =, ...)`
  constraint can't see a conflict between a whole court and its split
  pickleball-line halves — different `court_id`s, same physical space.
- **Accessibility/i18n research (§2):** `ClockTimeOf` resolves a pricing
  band (and, later, a recurring-hire weekday) from whatever `time.Time`
  the caller passes, with nothing pinning that to the court's actual IANA
  timezone — no `timezone` column exists on `courts` at all.
- Both are variations of **F1 from the original design review** (Booking
  and Game not sharing an invariant) — an entity assumed to be a single
  flat unit turns out to need a relationship (to another court, or to a
  timezone) the schema doesn't record yet.

This is worth naming explicitly because it's cheaper to add a
`courts.timezone` column and a `courts.parent_court_id`/overlap-group
concept now, before more code depends on `court_id` being a sufficient
key, than to retrofit both later.

---

## Priority-ranked findings

### P0 — schema/architecture decisions worth making before more code lands on top

| # | Finding | Source | Why it's P0 |
|---|---|---|---|
| 1 | No IANA timezone column on `courts`; `ClockTimeOf` silently assumes correct locale | a11y/i18n §2 | Already affects today's pricing resolution near local midnight, not just future recurring-hire work |
| 2 | No `currency_code` column alongside `price_cents` | a11y/i18n §3 | One column, trivial now; a schema-and-every-callsite migration later |
| 3 | No waitlist data model despite §3.4 claiming the feature | functional §1.3 | Spec currently asserts a feature the data model can't support at all |
| 4 | No deletion/anonymization mechanism or retention policy (GDPR erasure vs. payment-retention conflict unresolved) | security §4 | Needs a policy decision, not just code; blocks a clean `users` deletion path once real accounts exist |
| 5 | Split/shared court modeling absent (`court_id`-keyed EXCLUDE can't see it) | functional §1.5/§4.4 | Same invariant-hole class as F1; cheap to design for now, log as known limitation if deferred |

### P1 — needed before the specific backlog item that depends on it ships

| # | Finding | Source | Affects |
|---|---|---|---|
| 6 | BOLA/BFLA/BOPLA authorization not named as an NFR; `CancelBooking` (T3, already shipped) needs a "user A can't cancel user B's booking" regression test as an audit target | security §1 | Auth work (HANDOFF cross-cutting), retroactive T3 audit |
| 7 | Auth0 port should be shaped around IdP-agnostic OAuth2/OIDC claims (RFC 9068), not an Auth0 SDK type, so the stub→real swap is adapter-only | security §3 | The stubbed ACL decision already agreed — pin the interface before building the stub |
| 8 | No-show fee / account-credit concept missing from payment model (only paid/unpaid) | functional §1.2 | T6 Payments context design |
| 9 | Competitions (§3.8) names zero concrete bracket formats; double elimination is the dominant real format | functional §3.2 | Phase 3 competitions work |
| 10 | Internal rating should weight score margin vs. expectation, not just win/loss, to be genuinely "DUPR-style" | functional §2.1 | T5 matchmaking follow-up, `matches` schema |
| 11 | Verified (Game Admin-recorded) vs. self-reported match weighting + provisional-rating flag | functional §2.2 | Same schema area as #10; natural fit with existing Game Admin role |
| 12 | Per-facility configurable cancellation window (not a platform constant) | functional §1.1 | CancelBooking's real-world policy logic (T3 built the mechanism, not the policy) |
| 13 | PCI guardrail: never accept a raw PAN/CVV field on any request DTO, in proto or REST | security §2 | T6 Payments — add as a proto-review checklist item before Payments proto is drafted |

### P2 — deferrable, log now so it isn't forgotten

- DST-safe recurring-hire generation test (direction is already correct; needs an explicit AC when the generator is built) — functional §4.1, a11y/i18n §2
- Configurable buffer time between bookings — functional §4.2
- Booking-open visibility window (how far ahead a slot can be seen/booked, separate from cancellation window) — functional §4.3
- Recurring-hire booking-window + entitlement/pass concept — functional §1.4
- Terminology: "king-of-the-court / Swiss style" conflates two distinct formats — functional §3.3

### NFR posture (performance/availability + a11y — process changes, not backlog tickets)

- **SLOs:** adopt loose, provisional targets now (e.g. 99% monthly API uptime; track — don't target — the rate of untranslated booking-write errors), per the SRE book's "start loose, tighten from real data" guidance. Revisit once there's a first cohort of real bookings. Not a ticket — a line to add to spec §4.
- **Postgres EXCLUDE/GiST:** current bounded-retry mitigation (T4) is confirmed correct by an independent public bug report, not just this project's own experience. Explicit revisit trigger: re-evaluate (advisory lock / hot-slot sharding) only once sustained concurrent writes on a *single* court/slot regularly exceed ~20–30 simultaneous transactions — below that, retry is sufficient.
- **Load testing:** script T4's existing manual 20-concurrent-`CreateBooking` proof as a repeatable k6 spike test run from a **cold** container start in CI — this is the one recommendation two independent research passes (performance, and this project's own T4 correction) converge on.
- **Observability:** Grafana Cloud free tier (metrics/logs/traces/uptime) + Sentry free tier (error tracking) + OpenTelemetry instrumentation from day one, so the backend stays swappable later.
- **WCAG version:** state the spec's "WCAG AA" commitment as **WCAG 2.2 AA** specifically (current W3C Recommendation, strict superset of 2.1). §3.3.4 (Error Prevention — Legal/Financial/Data) applies directly to booking-with-payment and to `CancelBooking`; §4.1.3 (Status Messages) is the concrete AA requirement behind every async "just taken, pick another slot" / payment-recorded toast needing an ARIA live region, not just visual rendering.
- **Cross-platform error shape:** WCAG 3.3.1/3.3.3, Apple HIG, and Android TalkBack guidance all converge on the same API-design requirement — every error response needs a precise, human-readable message per error case (not just a domain error code), so Vue/Swift/Kotlin clients can each feed it into their respective accessible-announcement mechanism without re-inventing copy or leaking a raw `23P01`/error code to an end user.

---

## What this changes about the existing spec/backlog

Nothing has been edited in `docs/pickleball-platform-spec.md` or
`HANDOFF.md` yet — this is a research synthesis, not a spec revision.
Aligned findings (multi-court single booking, single global rating,
King-of-the-Court design, PCI SAQ-A claim, GDPR minimisation principle,
UTC-storage design) need no action and aren't repeated here; see each
source doc's verdict tables for the full list.

The P0/P1 items above are natural inputs to **Ceremony 1 (Backlog
refinement)** in `docs/process/sprint-process.md` — PM + Principal
Engineer would fold the relevant ones into whichever sprint's tickets they
land in, rather than this document prescribing tickets unilaterally.
