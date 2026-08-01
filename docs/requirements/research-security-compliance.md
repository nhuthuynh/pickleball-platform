# Security & Compliance Research — Non-Functional Requirements

Research to verify/extend `pickleball-platform-spec.md` §4 (Non-functional
requirements) and §5 (Payment design) against real external standards, for a
Go/gRPC/REST backend on Postgres+PostGIS, with Stripe (Connect) payments and
a planned Auth0 integration (currently a stubbed anti-corruption layer per
`HANDOFF.md` T6 / CLAUDE.md rule 3's ports-and-adapters pattern).

Scope note: web fetches to primary sources (owasp.org, stripe.com,
gdpr-info.eu, etc.) returned HTTP 403 in this environment (likely bot
defenses on those origins, not a proxy fault — `curl
$HTTPS_PROXY/__agentproxy/status` showed no relay failures). Findings below
are sourced via web search result snippets that quote the primary documents
directly (titles/URLs listed per section and in Sources), not paraphrased
secondary blog summaries where a primary quote was available. Where only a
secondary source was available, that's noted.

---

## 1. OWASP API Security Top 10 (2023) — mapped to this system

The current list (OWASP API Security Project, `owasp.org/API-Security/editions/2023/en/`):

| # | Risk | Relevance to this system | Concrete mitigation implied |
|---|------|---------------------------|------------------------------|
| API1 | **Broken Object Level Authorization (BOLA)** | **Highest relevance.** Every booking, registration, and payment record is fetched/mutated by ID (`CancelBooking`, `ListCourtBookings`, future `GetPayment`/registration endpoints). A player could pass another player's `booking_id` and cancel or view it. | Every app-layer method that takes an object ID must check the caller (from the auth context, once real) owns/is-authorized-for that object *before* acting — not just that the object exists. This has to be enforced in `internal/<context>/app`, not left to the DB or to REST-layer conventions. `CancelBooking` (T3, already shipped) is a concrete audit target: confirm it checks requester == booking owner (or Game Admin/Owner/Admin scope) and add a regression test for "user A cannot cancel user B's booking" if that test doesn't already exist. |
| API2 | **Broken Authentication** | High relevance once Auth0 lands — token validation is the front door for every gRPC/REST call. | Validate JWTs per RFC 9068/8725 (signature, `iss`, `aud`, expiry — see §3 below) at the gRPC interceptor layer, not per-handler. No endpoint should be reachable unauthenticated except explicitly public discovery endpoints (search, facility listing). |
| API3 | **Broken Object Property Level Authorization (BOPLA)** | Relevant to any endpoint returning a full row (e.g., a `Booking` or `User` object) that includes fields the caller shouldn't see (payment references, another user's contact info) or shouldn't be able to write (marking `paid` without Host/Game Admin role, per D3a). | Response DTOs (proto messages) should be shaped per-caller-role rather than returning the full domain object; mutation paths must allow-list which fields a given role can set (e.g., only Host/Game Admin/Owner/Admin can flip `payment.status`). |
| API4 | **Unrestricted Resource Consumption** | Booking search/list endpoints and "near me" geo search are natural targets for scraping/DoS given no rate limiting is mentioned in the spec. | Add per-endpoint rate limiting and pagination caps, especially on `ListCourtBookings` and geo search, before those are internet-facing. |
| API5 | **Broken Function Level Authorization (BFLA)** | Relevant to the role model: Game Admin, Host, Owner, platform Admin are all distinct scopes (§3.1, D3a). A player token must not be able to call `CreatePricingRule`, mark payments as offline-paid, or manage another owner's facility. | Function-level checks (role/scope, not just object ownership) belong in the app layer per RPC method; this is a second, independent check from API1's object-level check — both are needed. |
| API6 | **Unrestricted Access to Sensitive Business Flows** | The booking-creation flow itself is a sensitive business flow (a scripted client could mass-book/hold slots to scalp them, or hammer `CreateBooking` to grief the EXCLUDE constraint). | Consider CAPTCHA/velocity checks or per-user booking-attempt rate limits on `CreateBooking`, on top of the DB-level invariant (T4) which stops double-booking but not abuse of the booking flow itself. |
| API7 | **Server-Side Request Forgery (SSRF)** | Lower but non-zero relevance: any future feature that fetches a URL server-side (e.g., validating a facility's website link, fetching an avatar from a URL, webhook callbacks from Stripe) is an SSRF vector. | Validate/allow-list outbound URLs; Stripe webhooks should be *inbound* (verified by signature), not the server fetching attacker-supplied URLs. |
| API8 | **Security Misconfiguration** | Applies to gRPC-gateway CORS config, error verbosity (domain errors must not leak Postgres internals — CLAUDE.md rule 5 already asks for this), and Docker/Jenkins config maturing past prototype. | Keep the CLAUDE.md rule 5 error-translation discipline; add a config lint/checklist before any non-prototype deploy (CORS allow-list, no debug endpoints in prod, TLS termination). |
| API9 | **Improper Inventory Management** | Directly relevant to this repo's own practice: proto is the single source of truth (CLAUDE.md), which is the correct mitigation — but only as long as every deployed endpoint traces back to a `.proto` and nothing is hand-added to `cmd/server` outside codegen. | Keep "no hand-edited generated code" (rule 6) enforced; consider an API inventory check in CI that diffs deployed routes against the OpenAPI output. |
| API10 | **Unsafe Consumption of APIs** | Relevant once Stripe and Auth0 are live: this backend *consumes* Stripe's and Auth0's APIs/webhooks. | Validate Stripe webhook signatures, treat Auth0 JWKS/discovery responses as untrusted input to parse defensively, and don't trust unvalidated third-party payloads into domain logic. |

**Verdict: spec has a gap.** §4 says "Row Level Security on every table" and "PII minimised" but says nothing about API-layer authorization design (BOLA/BFLA/BOPLA — API1/3/5, three of the current top five risks). Given D3a's multi-role model (player/Host/Game Admin/Owner/Admin) and a booking/payment system where object-level authorization bugs are the single most common real-world API vulnerability class per OWASP's own ranking, this should be named explicitly as a non-functional requirement, with per-endpoint object- and function-level authorization tests as a Definition-of-Done item (this already has a natural home: CLAUDE.md's "Definition of Done" checklist and HANDOFF.md's cross-cutting "Auth (JWT) + per-context authorization" line item — recommend making BOLA/BFLA regression tests an explicit AC there, not just "wire it in").

---

## 2. PCI-DSS scope for the Stripe integration

**Findings:**
- Stripe Checkout (hosted payment page) and Stripe Elements (embedded fields) both submit card data directly from the client to Stripe's servers — the merchant's backend never receives, processes, or stores raw card data. Per Stripe's own PCI compliance guidance, this qualifies the merchant for **SAQ A**, the simplest self-assessment questionnaire, "since cardholder data never touches your servers."
- The merchant is **not exempt from all PCI obligations** even on SAQ A: because the payment page is served from the merchant's own domain, the merchant remains responsible for the security of scripts loaded on that page (e.g., not injecting a compromised third-party script that could skim card fields) and still files an Attestation of Compliance (AoC).
- If the design ever changes so that raw card data is submitted to *this* backend (e.g., a custom card form posting to a Go endpoint instead of Stripe.js/Elements tokenizing client-side), the applicable questionnaire jumps to **SAQ A-EP** (if the backend only orchestrates the payment page/flow but never stores/processes/transmits the actual PAN) or **SAQ D** (the most comprehensive questionnaire, for merchants that do store/process/transmit cardholder data directly) — both are materially heavier compliance burdens than SAQ A.
- **Stripe Connect specifically**: for a marketplace/platform model (D3, this project's model — Connect with a platform + connected owner/host accounts), Stripe's Connect documentation states Connect "tokenizes card data to help with PCI compliance" and separately handles identity verification/KYC/sanctions checks during connected-account onboarding — i.e., the platform doesn't need to separately solve KYC for owners/hosts receiving payouts, which matches the "Owner verification/onboarding... before receiving payouts" line in spec §3.1.

**Verdict: spec is aligned, with one addition worth making explicit.** §4's claim — "payment data never touches your servers (Stripe-hosted)" — is accurate *as a design intent* and is achievable at SAQ A, provided the actual implementation uses Stripe.js/Elements or Checkout (client-side tokenization) end-to-end, including in the Swift/Kotlin native clients (Stripe's mobile SDKs follow the same non-touching pattern). This should be written down as an explicit constraint — e.g., "the backend must never accept a raw PAN/CVC field on any request DTO, in proto or REST" — so a future contributor doesn't accidentally add a `card_number` field to a request message during T6 (Payments context) and silently move the system to SAQ A-EP/D scope. Recommend adding this as a one-line rule alongside the existing "Golden rules" in CLAUDE.md when T6 starts, and as a proto-review checklist item (no PAN/CVV/track-data fields, ever).

---

## 3. Auth0 / OAuth2 / OIDC best practices for a backend API

**Findings, from RFC 9068 (JWT Profile for OAuth 2.0 Access Tokens), RFC 8725 (JWT Best Current Practices), RFC 8252 (OAuth 2.0 for Native Apps), and Auth0's own token/RBAC docs:**

- **Token validation (resource-server side, i.e. this Go backend):** the resource server MUST (a) verify the JWT signature per RFC 7515 and reject `alg: none`; (b) verify `iss` exactly matches the expected authorization server identifier; (c) verify `aud` contains an identifier for *this* API specifically — RFC 8725 calls out "cross-JWT confusion" as a named attack class this guards against (a token minted for a different API/audience being replayed against this one). This maps directly onto a `grpc.UnaryInterceptor`/`StreamInterceptor` that does signature+iss+aud+exp validation once, centrally, rather than per-handler.
- **Refresh tokens:** rotation (a new refresh token issued — and the old one invalidated — on every use) is described as "the gold standard"/"non-negotiable" by current guidance, with reuse-detection triggering revocation of the whole token family. Access tokens should be short-lived (commonly 5–15 min, ceiling ~1 hour); refresh tokens capped around 30 days. Refresh tokens are opaque to the resource server — this backend should never need to see or handle refresh tokens at all; that's strictly an Auth0-token-endpoint concern.
- **Native apps (the Swift/Kotlin clients in this project):** RFC 8252 specifies the Authorization Code flow **with PKCE**, run in the system browser (not an embedded webview), as the required pattern for public/native clients, since they can't hold a client secret; the Implicit flow is explicitly NOT RECOMMENDED for native apps because it can't refresh without full re-auth. This constrains how the iOS/Android apps must integrate with Auth0 regardless of backend design.
- **Scopes/RBAC:** Auth0's RBAC-for-APIs model puts authorization data in the JWT's `scope`/permissions claims (an intersection of what the client requested and what's been assigned to the user), with the recommendation to keep permission sets minimal ("only remove scopes which are not authorized... refrain from adding scope") and to put access-control data that must survive across calls in durable claims (e.g., `app_metadata`) rather than re-querying an external API per-request.

**Design implication for the stubbed ACL (this project's actual open question):** to make the "stub now, real Auth0 later, interface unchanged" plan (locked decision, HANDOFF T6/CLAUDE.md rule 3) actually hold, the `port` interface this project defines should be shaped around the **IdP-agnostic OAuth2/OIDC concepts above**, not around any Auth0 SDK type:
- A `TokenVerifier` (or similarly named) port whose method takes a raw bearer token string and returns a small, backend-owned claims struct: subject/user ID, roles or scope list, expiry — i.e., exactly the `sub`/`scope`/`iss`/`aud`/`exp` claims RFC 9068 standardizes, not an Auth0-specific payload shape.
- The stub adapter (Auth0 not wired yet) implements this port by decoding a fixture/test JWT (or a fixed claims map) without hitting a network; the real adapter later implements the *same* port by fetching Auth0's JWKS and doing real signature/iss/aud verification. Because both satisfy the same domain-facing interface, swapping is an adapter change only — consistent with CLAUDE.md rule 3 (dependency rule points inward) and rule 5 (adapters translate infra errors into domain errors: an Auth0 network/JWKS error should map to a domain-level `ErrUnauthenticated`, not leak an HTTP/JWKS error upward).
- Role/scope checks (API5/BFLA above) should be written against the backend's own role vocabulary (player/host/game_admin/owner/admin — the ubiquitous language already defined in CLAUDE.md rule 7), populated from the verified claims — not against raw Auth0 permission strings — so the app/domain layers stay Auth0-agnostic per rule 2 ("keep the domain pure").

**Verdict: spec has a gap, but the project's own locked decision already points at the right shape.** §4 doesn't mention authN/authZ at all (it's implied by "comply with your market's data-protection law" at best). The good news: the "stubbed ACL, Auth0 later" decision already referenced in this task's brief is exactly the right pattern per the ports-and-adapters rule already in CLAUDE.md — the concrete recommendation is to pin the *shape* of that port (claims struct + role vocabulary, above) now, before the stub is built, so the later Auth0 swap really is adapter-only.

---

## 4. GDPR / CCPA — data handling for PII, geolocation, and payment references

**Findings:**
- **GDPR right to erasure (Art. 17, "right to be forgotten"):** applies when data is no longer necessary for the purpose it was collected for, when consent is withdrawn and there's no other legal basis, or when the subject objects to processing — all plausible for a player who deletes their account. Exceptions exist for legal-obligation compliance, public-interest archiving, and freedom-of-expression grounds, none of which obviously apply to routine player data (payment/tax records may need retention under a *different* legal basis — see below).
- **Data minimisation (Art. 5(1)(c)) and storage limitation (Art. 5(1)(e))** are described as the two "quantity-control principles": minimisation governs *how much* is collected (only what's necessary for the stated purpose), storage limitation governs *how long* it's kept (no longer than necessary) — these are two distinct obligations, not one.
- **CCPA (California, enforced by the CA Attorney General / CPPA):** grants a parallel right to delete, plus explicit purpose-limitation/data-minimization rules: collection, use, and retention must be "reasonably necessary and proportionate" to purposes a consumer would reasonably expect, that are disclosed, or that they agreed to.
- **Geolocation data specifically:** the UK ICO (the actual regulator, not a blog) treats geolocation/location data as personal data under the Art. 4 definition requiring a lawful basis to process, and separately flags it as *practically* sensitive even though it isn't automatically "special category" data — frequent/continuous location tracking can reveal patterns (routines, places visited) that go well beyond a single "near me" query. This is directly relevant to this spec's "near me" search (§3.10): a one-off, user-initiated proximity query against static facility coordinates is low-risk, but *storing* a history of user-location pings over time would raise the sensitivity bar considerably.
- **Payment references:** neither GDPR nor CCPA erasure rights are absolute where a separate legal obligation requires retention — financial/tax/accounting record-keeping law (jurisdiction-dependent, typically several years) is a standard, legitimate basis to retain payment records (e.g., `payments` rows, Stripe charge/customer IDs) even after a user requests account deletion, so long as the retained data is limited to what that obligation requires and is otherwise put beyond ordinary use (the common pattern: anonymize/detach the PII, keep the transaction/statement records).

**Verdict: spec is aligned in principle, has one gap.**
- **Aligned:** §4's "PII minimised... comply with your market's data-protection law" and §4's consent-based, non-silent-profiling data-collection principle (F6) and §7's "no scraping / no silent profiling" stance are directionally consistent with GDPR Art. 5 minimisation and the "reasonably necessary and proportionate" CCPA standard. No conflict found.
- **Gap:** the spec doesn't mention a **deletion/erasure mechanism** or **retention policy** anywhere in §4, §5, or the data model (§8) — no `deleted_at`/anonymization strategy for `users`, no stated retention period for `payments`/`analytics_events`, no accounting for the fact that erasing a user must NOT cascade-delete `payments` rows that a court booking/tax record needs to keep (this is where GDPR erasure and financial-retention law collide and need an explicit resolution, not silence). Recommend adding: (a) a documented retention period per table in §8, especially `payments`, `analytics_events`, and any future location-history table; (b) an explicit "delete vs anonymize" decision for `users` on account deletion (likely: anonymize PII fields, retain the row/id for FK integrity in `payments`/`bookings`/`matches`); (c) treating `payments`'s existing "single source of truth" design (§5) as the natural place to enforce field-level minimisation (store Stripe *references*, e.g. `charge_id`/`customer_id`, never card data — already implied but not stated as a rule) and to hang the retention/anonymization policy off, since every payable action already funnels through one table.

---

## Summary of verdicts

| Area | Verdict |
|---|---|
| OWASP API Security Top 10 | **Gap** — API-layer authZ (BOLA/BFLA/BOPLA) isn't named as a non-functional requirement despite being the top current API risk class and directly applicable to this role/booking model. |
| PCI-DSS / Stripe scope | **Aligned** — SAQ A claim is accurate for Checkout/Elements/mobile SDK client-side tokenization; needs one explicit guardrail (no PAN/CVV fields ever, in proto or REST) to stay true through T6. |
| Auth0 / OAuth2 / OIDC | **Gap, but the locked "stub now, adapter-swap later" decision is the right shape** — needs the port's claims/role interface pinned down now (IdP-agnostic, RFC 9068-shaped) so the later swap is truly adapter-only. |
| GDPR/CCPA | **Aligned in principle, gap in mechanism** — minimisation/consent principles match; no stated erasure mechanism, retention policy, or delete-vs-anonymize decision for `users`/`payments`/`analytics_events`. |

---

## Sources

- OWASP API Security Project, Top 10 2023: https://owasp.org/API-Security/editions/2023/en/0x11-t10/ and table of contents https://owasp.org/API-Security/editions/2023/en/0x00-toc/ ; per-risk pages e.g. API1 https://owasp.org/API-Security/editions/2023/en/0xa1-broken-object-level-authorization/ , API3 https://owasp.org/API-Security/editions/2023/en/0xa3-broken-object-property-level-authorization/ ; OWASP Foundation project page https://owasp.org/www-project-api-security/ ; OWASP announcement https://owasp.org/blog/2023/07/03/owasp-api-top10-2023
- Stripe, "What is PCI DSS compliance?": https://stripe.com/guides/pci-compliance
- Stripe Connect documentation (platforms/marketplaces, connected accounts, PCI/KYC): https://docs.stripe.com/connect and https://stripe.com/connect
- Stripe Terminal PCI/SAQ P2PE support doc: https://support.stripe.com/questions/stripe-terminal-payments-and-pci-compliance
- PCI Security Standards Council, SAQ A-EP (v4.0) and SAQ D (v4.0) forms: https://listings.pcisecuritystandards.org/documents/PCI-DSS-v4-0-SAQ-A-EP.pdf , https://listings.pcisecuritystandards.org/documents/PCI-DSS-v4-0-SAQ-D-Merchant.pdf
- IETF RFC 9068, "JSON Web Token (JWT) Profile for OAuth 2.0 Access Tokens": https://datatracker.ietf.org/doc/html/rfc9068
- IETF RFC 8725, "JSON Web Token Best Current Practices": https://datatracker.ietf.org/doc/html/rfc8725
- IETF RFC 8252, "OAuth 2.0 for Native Apps": https://www.rfc-editor.org/rfc/rfc8252.html
- Auth0 Docs, "Token Best Practices": https://auth0.com/docs/secure/tokens/token-best-practices
- Auth0 Docs, "Role-Based Access Control": https://auth0.com/docs/manage-users/access-control/rbac and "Enable Role-Based Access Control for APIs": https://dev.auth0.com/docs/get-started/apis/enable-role-based-access-control-for-apis
- Auth0 blog, "A Guide to Auth0 Authorization" (scopes/roles/client grants): https://auth0.com/blog/auth0-authorization-guide-scopes-roles-client-grants/
- GDPR Art. 17 (right to erasure), full text/commentary: https://gdpr-info.eu/art-17-gdpr/
- GDPR.eu, "Right to be Forgotten": https://gdpr.eu/right-to-be-forgotten/
- Ireland Data Protection Commission, "The right to erasure (Articles 17 & 19 GDPR)": https://dataprotection.ie/en/individuals/know-your-rights/right-erasure-articles-17-19-gdpr
- California Office of the Attorney General, CCPA page: https://oag.ca.gov/privacy/ccpa and CCPA Fact Sheet PDF: https://www.oag.ca.gov/system/files/attachments/press_releases/CCPA%20Fact%20Sheet%20(00000002).pdf
- California Privacy Protection Agency (CPPA) FAQ: https://cppa.ca.gov/faq.html
- UK Information Commissioner's Office (ICO), geolocation guidance (Age Appropriate Design Code, §10 Geolocation — general geolocation-as-personal-data reasoning applies beyond the children's-code context it's published under): https://ico.org.uk/for-organisations/uk-gdpr-guidance-and-resources/childrens-information/childrens-code-guidance-and-resources/age-appropriate-design-a-code-of-practice-for-online-services/10-geolocation/
- ICO glossary, "Data type" (location data definition): https://ico.org.uk/action-weve-taken/complaints-and-concerns-data-sets/data-security-incident-trends/glossary-of-terms/data-type/

**Note on method:** direct `WebFetch` to primary sources (owasp.org, stripe.com, gdpr-info.eu, en.wikipedia.org) returned HTTP 403 throughout this research session — confirmed not a proxy relay failure via `$HTTPS_PROXY/__agentproxy/status`. All findings above are drawn from `WebSearch` result snippets, which in most cases directly quote the primary document (RFCs, OWASP risk pages, Stripe's own guide, ICO's own pages, oag.ca.gov). Where a quote couldn't be attributed to the primary source, it's presented as a synthesis across the listed secondary sources for that line. If exact-wording citations are needed for a compliance filing (as opposed to engineering NFRs), re-fetch the primary URLs directly rather than relying on this document.
