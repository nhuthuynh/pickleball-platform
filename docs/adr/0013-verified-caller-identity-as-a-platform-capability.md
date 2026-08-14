# ADR-0013: Verified caller identity is a platform capability, observe-only first

- **Status:** Accepted
- **Date:** 2026-08-14
- **Ticket:** T12.2 (`docs/process/t12-sprint-plan.md`)
- **Relates to:** ADR-0007 (User deletion is anonymize, not cascade-delete),
  ADR-0011 (CI pipeline and security gating), and
  `docs/requirements/research-security-compliance.md` §3, which pinned the
  shape of this port two sprints before it was built

## Context

Every authorization check in this codebase rested on a **claimed** actor. A
request carried `actor_user_id` or `actor_player_id` as an ordinary field, the
handler read it, and the domain compared it against an owner. Nothing anywhere
established that the caller was who the field said. The T12 ceremony counted
the surface rather than estimating it: **54 occurrences across all six protos**
(booking 13, identity 11, socialplay 9, facilities 8, payments 7,
competitions 6).

The consequence is not theoretical. `HANDOFF.md`'s T10.2 bullet records a live,
disclosed defect: `CreateUser` takes the caller's own chosen UUID as the row's
permanent primary key, so an anonymous caller can permanently occupy an
identity a real person will later need — "a persistent, targeted denial of
service, not a rejected mutation." That bullet names its own closure condition:
*"Must close the moment real auth exists."*

Before this ADR there was **no auth precedent anywhere in the repo** — verified
by grep, and recorded in the sprint plan as a checked negative rather than an
assumption. `internal/platform/` held exactly `grpcrecovery`, `idgen`, and
`pg`. There was no auth package, no token verification, no principal type.

## Decisions

### 1. Auth lives in `internal/platform/auth`, not in a bounded context

Authentication is a cross-cutting concern that every context consumes.
Identity/Users is a *domain* context that owns the `User` aggregate.

Putting token verification inside `internal/identity` would make five other
contexts import a domain context in order to authenticate. That inverts the
dependency rule (CLAUDE.md rule 3, "the dependency rule points inward") and
recreates the `games.facility_id` mistake in a new place: a cross-cutting
concern parked in whichever context happened to need it first, which then
becomes a dependency magnet for everybody else.

A seventh bounded context was also rejected. A bounded context owns a piece of
the business's language and rules; token verification owns neither. It is
infrastructure that sits *below* the contexts, which is precisely the class
`internal/platform` already holds — `grpcrecovery` (a gRPC interceptor pair),
`idgen`, `pg`. `internal/platform/auth` is a sibling of those, not a peer of
`internal/booking`.

**Auth and Identity do meet — at exactly one seam.** Identity maps a verified
subject to a `User` (T12.9). It does not verify tokens.

**Constraint this imposes:** `internal/platform/auth` imports nothing from any
bounded context, now or later. Verified mechanically:
`go list -deps ./internal/platform/auth/...` resolves to only the two auth
packages themselves, plus the standard library and the JWT library.

### 2. `Principal` is not `User`

These are two concepts that will sit next to each other once T12.9 mints
`User.ID` from a verified subject, and blurring them would be a genuine
CLAUDE.md rule 7 (one ubiquitous language) violation — not because two words
are used, but because they would name one thing when they name two:

| | `auth.Principal` | `identity.domain.User` |
|---|---|---|
| What it is | The platform's verified-caller value | The Identity context's aggregate |
| Where it comes from | Derived from a signed token; **authored by the identity provider** | A row this backend owns and authors |
| Lifetime | One request | Persistent, with a lifecycle (created, updated, anonymized per ADR-0007) |
| Has domain rules | No | Yes |

Only `Principal.Subject` (the `sub` claim) is authoritative. Authorization
decisions key off it; everything else on the struct is context for logging,
scope checks, and diagnosis.

Per the sprint plan's A11 Ruling 3, contexts must **not** grow a shared-kernel
dependency on `Principal`. Handlers translate a `Principal` into whatever their
own domain already understands (an owner ID, an actor ID) **at the grpcapi
boundary**; `domain` and `app` packages keep their existing signatures and must
not import this package. That is what keeps the domain pure (rule 2) and keeps
every existing domain-level authorization test valid unchanged.

### 3. The port/adapter split, mirroring Stripe

`auth.TokenVerifier` is the seam; `auth/rs256` is one replaceable
implementation. This is the same split the project already applies to payments
(`internal/payments/port.PaymentProcessor` +
`internal/payments/adapter/stripestub`), and the shape is not invented here:
`docs/requirements/research-security-compliance.md` §3 specified it in advance,
from RFC 9068 and RFC 8725 — "a `TokenVerifier` port whose method takes a raw
bearer token string and returns a small, backend-owned claims struct."

`rs256.KeySource` is a **second, deliberate seam** inside the implementation.
Verifying a token and obtaining the keys to verify it with have completely
different failure modes and lifetimes: verification is pure and instant, key
retrieval is remote, cached, and subject to rotation. Splitting them means the
network half can be added later without touching the verification logic that
the platform's security actually rests on.

**A verifier must verify, not merely parse.** A verifier that checks the
signature and stops is worse than no verifier, because it looks like a check in
review and in logs. The implementation checks signature (against a key selected
by `kid`, under an explicit RS256-only allowlist), `exp`, `nbf`, `iss`, `aud`,
and the presence of `sub`. `rs256.NewVerifier` **refuses to construct** a
verifier with an empty issuer or audience, because either would make the
corresponding check vacuously pass — turning a silent downgrade into a startup
error.

### 4. Observe-only now, per-context enforcement later

The interceptors ship **observe-only**: they populate the request context when
a valid token is present and enforce nothing. A request with no token, or an
expired, forged, or wrong-audience one, proceeds to its handler exactly as it
did before, with no principal and no error.

This is an acceptance criterion, not an unfinished state. Two reasons:

1. **It makes the platform exercisable before anything depends on it.** The
   whole path — client metadata, gateway header forwarding, key selection,
   signature and claim checking, context plumbing — can be run against a live
   platform while every handler's correctness still rests on what it rested on
   yesterday.
2. **It is structurally incapable of breaking a shipped flow.** No code path
   introduced here can turn a request that previously succeeded into one that
   fails. (One residual exception is named under "Known gaps" below, and
   tracked.)

Enforcement is turned on per-RPC by T12.7/T12.8/T12.9, each context exporting
its own `AuthenticatedMethods()` (A11 Ruling 2) — the knowledge of which of a
context's RPCs are public belongs with that context's handlers, next to the
code that breaks if it is wrong.

The sprint plan records an unresolved PdE/PE disagreement about whether
observe-only is as strong a proving mechanism as a merged, reviewed consumer
(A14). This ADR does not claim to settle it. It is scoreable at T12's retro: if
T12.7, T12.8, and T12.9 each independently hit the same defect in this package,
PdE was right.

### 5. `Unauthenticated` vs `PermissionDenied` — gRPC codes only

The convention every later ticket follows. Conflating these two is the classic
auth bug, and it is worth being pedantic about now, before nine handlers each
invent an answer:

- **`codes.Unauthenticated`** — *"I do not know who you are."* A missing,
  expired, malformed, wrong-issuer, wrong-audience, or badly-signed token.
- **`codes.PermissionDenied`** — *"I know who you are, and you may not do
  this."* A perfectly valid token whose `Principal` fails an **object-level**
  ownership check: this booking is not yours, this game is not yours to cancel.

Every ownership check that exists in this codebase today keeps returning
`PermissionDenied` when it lands. Migration to a verified principal changes
*where the actor identity comes from*, not *what the failure means*.

**One nuance, encoded in the error vocabulary rather than left to memory.**
`auth.ErrKeyUnavailable` is neither of the above. A JWKS we cannot reach is our
outage, not the caller's forgery, and reporting it as `Unauthenticated` would
send an entire fleet of clients into a pointless re-authentication loop during
an incident. It should map to `codes.Unavailable`. `auth.IsTokenRejection(err)`
exists so each migration ticket asks this question in one place instead of
writing its own `errors.Is` chain and getting this case wrong.

No HTTP status codes are named anywhere in this decision. The gateway derives
them from the gRPC code; naming both invites the two to drift.

### 6. The JWT library, rather than hand-rolled verification

`github.com/golang-jwt/jwt/v5` — the first third-party auth dependency in this
repo. It has no transitive dependencies of its own.

The alternative was implementing RS256 verification on the standard library
(`crypto/rsa`, `encoding/base64`, `encoding/json`), which is roughly 200 lines
and entirely possible. It was rejected: hand-rolled JWT parsing is a
well-populated graveyard, and the failure mode of getting it subtly wrong is
silent acceptance of a forged token. JWKS *parsing* is still written here
(RFC 7517 is not in the library's scope), which is why the JWKS parser is
defensive by construction — see "untrusted input" below.

The library's own footgun is guarded explicitly: it honours the token's `alg`
header unless told otherwise, so `jwt.WithValidMethods` pins RS256, and the
keyfunc re-asserts the signing method independently. Both `alg: none` and an
HS256 token signed with the RSA *public key* (the classic algorithm-confusion
attack, using a value a JWKS publishes to the world) are covered by tests.

### 7. A JWKS is untrusted input; malformed means startup failure

`research-security-compliance.md` §1 (API10) requires treating IdP discovery
responses as untrusted input parsed defensively. Every malformed shape is a
**construction error**, never a silently-skipped key.

This matters more than it first looks: skipping a bad entry would yield a key
set missing the key everyone's tokens are signed with, and **in observe-only
mode the resulting total verification failure is invisible** — no caller is
told, because nothing is enforced. A startup error is loud; a quietly empty key
set is not. Keys below 2048 bits are rejected at parse time (RFC 8725 §3.5), as
are duplicate `kid`s, since ambiguous key selection produces a verification
result nobody can reason about afterwards.

## What "auth exists" does and does not mean after this ticket

Stated with the same honesty `HANDOFF.md` has applied to the Jenkins server
since SCRUM-6, because this is the claim most likely to be over-read:

> **Provisioning an identity provider tenant, registering the application, and
> publishing a JWKS endpoint is server-side infrastructure work that no coding
> session in this project can perform.** No session in this project's history
> has had a reachable external identity provider.

So, precisely:

- **What exists:** the verification and enforcement machinery — a principal
  type, a verifier port, a real RS256/JWKS-backed implementation, an
  interceptor pair, and an error vocabulary — tested against locally-minted
  RSA key material, with no network, no Docker, and no external tenant.
- **What does not exist:** a production identity provider, any real token
  issuance, and any client that sends a token. `cmd/server` starts with **no
  verifier configured** unless `AUTH_ISSUER`, `AUTH_AUDIENCE`, and
  `AUTH_JWKS_FILE` are all set, and logs that it is doing so.

"Auth exists" after this ticket means *the half this project can build is built
and tested*, not *a production IdP is wired up*.

## Known gaps, disclosed and tracked

Per the sprint plan's A5 standing rule, a disclosed gap gets a durable record,
not a paragraph. Each of these was opened as a GitHub issue by the T12.2 PR
that disclosed it — issues #135, #136, #137 and #138 respectively:

1. **(#135) A panicking verifier is the one way this ticket could affect a shipped
   flow.** The interceptor deliberately does not recover panics — recovery sits
   in front of it and catches them, which is tested. But a verifier that panics
   turns requests into `Internal` errors rather than passing them through, so
   the "structurally cannot break a shipped flow" property has this single
   exception. Recovering locally would be fail-open, which is *worse* once
   enforcement is on. That is a deliberate decision the enforcement tickets
   must make, not one to make silently here.
2. **(#136) A nil verifier is fail-open under enforcement.** Today it means "resolve
   nothing", which is exactly today's behavior and therefore safe. Once any RPC
   enforces, a nil verifier must become a startup failure instead.
3. **(#137) No remote `KeySource`.** Keys come from a JWKS document on disk. There is
   no HTTP client, no cache, and no rotation handling, so this platform cannot
   yet verify a token from a live provider even once one exists. The seam is in
   place; the implementation is not, and no T12 ticket owns it.
4. **(#138) `internal/platform/**` tests do not run in `make ci`.** `make test-domain`
   globs `./internal/.../domain/... ./internal/.../app/...`, which does not
   match `internal/platform/auth`, and `make ci` runs no broader `go test`.
   This package's tests — and `grpcrecovery`'s, pre-existing — currently gate
   only in `make test`, which requires Docker. The security spine's tests not
   running in the default gate is worth fixing; the `Makefile` is T12.1's file
   this sprint, so this ADR flags it rather than editing it.

## Consequences

- Six contexts gain a trustworthy identity to migrate to, without any of them
  changing yet.
- The T10.2 `CreateUser` identity-squatting DoS becomes closable (T12.9).
- One new third-party dependency, with no transitive dependencies.
- `cmd/server` gains three optional environment variables and one startup
  failure mode (partial auth configuration), reachable only by someone who set
  at least one of them.
- Later contexts must resist two temptations this ADR forbids: importing
  `internal/platform/auth` from a `domain` or `app` package, and treating
  `Principal` as a `User`.
