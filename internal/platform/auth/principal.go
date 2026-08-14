// Package auth is this backend's verified-caller identity: the platform
// capability that turns a bearer token on the wire into a Principal the
// handlers are entitled to trust.
//
// # Why this is platform code and not a bounded context
//
// Authentication is a cross-cutting concern every context consumes.
// Identity/Users is a *domain* context that owns the User aggregate; folding
// token verification into internal/identity would make five other contexts
// import a domain context in order to authenticate, inverting the dependency
// rule (CLAUDE.md rule 3). internal/platform already holds exactly this class
// of code — grpcrecovery, idgen, pg — and this package sits beside them. It
// imports nothing from any bounded context, and must not start to. See
// ADR-0013 and the T12 sprint plan's A11 Ruling 1.
//
// # What this package does today, and what it deliberately does not
//
// The interceptors in this package are **observe-only**. They resolve a token
// when one is present, put the resulting Principal in the request context, and
// enforce nothing: a request with no token, or an expired one, or a forged
// one, proceeds exactly as it did before this package existed. That is a
// deliberate acceptance criterion of the ticket that introduced it (T12.2),
// not a half-finished state — it lets the whole platform be exercised end to
// end before any handler's correctness depends on it, which means introducing
// it cannot break a single already-shipped flow.
//
// Enforcement is turned on per-RPC by the migration tickets that follow
// (T12.7/T12.8/T12.9), each context declaring which of its own RPCs require a
// principal. When that happens the two gRPC codes below must stay distinct,
// because conflating them is the classic authorization bug:
//
//   - codes.Unauthenticated — "I do not know who you are." A missing,
//     expired, malformed, or otherwise unverifiable token.
//   - codes.PermissionDenied — "I know who you are, and you may not do this."
//     A perfectly valid token whose Principal fails an object-level ownership
//     check (this booking is not yours, this game is not yours to cancel).
//
// One nuance the sentinel errors below encode: ErrKeyUnavailable is neither.
// A JWKS we cannot reach is our outage, not the caller's forgery, and should
// map to codes.Unavailable. Use IsTokenRejection to tell the two apart rather
// than treating every Verify error as an authentication failure.
//
// # What "auth exists" does and does not mean
//
// This package is the verification and enforcement half. Provisioning an
// identity provider tenant, registering the application, and publishing a
// JWKS endpoint is server-side infrastructure work no coding session in this
// project can perform — the same honesty HANDOFF.md applies to the Jenkins
// server. Everything here is tested against locally-minted RSA key material
// with no network and no Docker. Treat "auth exists" as "verification and
// enforcement machinery exists and is tested", not "a production IdP is wired
// up".
package auth

import (
	"context"
	"time"
)

// Principal is a caller whose identity has been cryptographically verified —
// the value that replaces the "claimed actor" pattern (an actor_user_id field
// on a request DTO, trusted because the caller said so) that every
// authorization check in this codebase rested on before this package existed.
//
// # A Principal is NOT a User
//
// This distinction is load-bearing and will be tempting to blur, because the
// two types end up side by side once Identity mints User rows from verified
// subjects (T12.9). They are two concepts, not two names for one:
//
//   - A Principal is the platform's *verified-caller* value. It is derived
//     entirely from a signed token, it exists for the life of one request, it
//     has no persistence and no lifecycle, and this backend does not author
//     it — the identity provider does.
//   - A User is the Identity bounded context's *aggregate*: a row this
//     backend owns, with a lifecycle (created, updated, anonymized on
//     deletion per ADR-0007), a profile, and domain rules of its own.
//
// The relationship between them belongs at exactly one seam: Identity maps a
// verified Subject to a User. Every other context translates a Principal into
// whatever its own domain already understands — an owner ID, an actor ID — at
// its grpcapi boundary, and its domain and app packages keep their existing
// signatures and must not import this package (T12 sprint plan, A11 Ruling 3).
//
// # Only Subject is authoritative
//
// Subject is the token's `sub` claim: globally unique within the issuer, and
// stable for the life of the identity. Everything else on this struct is
// context for logging, scope checks, and diagnosis. Authorization decisions
// key off Subject.
type Principal struct {
	// Subject is the verified `sub` claim — the single authoritative field.
	// A Principal with an empty Subject is not an identity; the constructors
	// and context helpers in this package refuse to produce or store one.
	Subject string

	// Issuer is the verified `iss` claim: which identity provider vouched for
	// this caller. Retained because a multi-tenant or migrated deployment can
	// legitimately trust more than one issuer, and "which one said so" is the
	// first question asked when a token is disputed.
	Issuer string

	// Audience is the verified `aud` claim, kept whole rather than reduced to
	// the one entry that matched: a token minted for several APIs is normal,
	// and the untruncated list is what makes an audit answer "what else was
	// this token good for?".
	Audience []string

	// Scopes is the `scope` claim split on whitespace (RFC 9068 §2.2.3 /
	// RFC 6749 §3.3). Empty when the token carries no scope claim. Present
	// for later coarse capability checks; object-level ownership checks are
	// not scope checks and must not be written as one.
	Scopes []string

	// IssuedAt and ExpiresAt are the verified `iat` and `exp` claims. The
	// verifier has already enforced expiry — these are here for logging and
	// for handlers that need to reason about token age, not for re-checking.
	IssuedAt  time.Time
	ExpiresAt time.Time
}

// HasScope reports whether the token carried the given OAuth2 scope.
//
// This is a capability check ("may this client class do this kind of thing"),
// never an ownership check ("is this particular booking yours"). Ownership
// stays in the domain, compared against Subject.
func (p Principal) HasScope(scope string) bool {
	for _, s := range p.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

// principalContextKey is an unexported struct type, so no other package can
// construct the key and forge a principal into a context from outside.
type principalContextKey struct{}

// ContextWithPrincipal returns ctx carrying p.
//
// It is exported because the migration tickets' handler tests need to build a
// context representing an authenticated caller without standing up a gRPC
// server and minting a token. Production code should not call it directly —
// the interceptors in this package are the only intended producer.
//
// A Principal with no Subject is not stored. That is not defensive
// tidiness: a zero-Subject principal would let a later ownership check compare
// "" against an empty owner column and succeed, which is precisely the class
// of bug this package exists to remove.
func ContextWithPrincipal(ctx context.Context, p Principal) context.Context {
	if p.Subject == "" {
		return ctx
	}
	return context.WithValue(ctx, principalContextKey{}, p)
}

// PrincipalFromContext returns the verified caller for this request, if there
// is one.
//
// The boolean is the whole contract: false means this request has no verified
// caller. In observe-only mode that is an entirely ordinary situation (nobody
// is sending tokens yet), so a handler reading this today must treat false as
// "anonymous", not as an error. Once its RPC is migrated to enforce, the
// interceptor guarantees false can no longer reach the handler, and the
// handler may treat true as an invariant.
func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalContextKey{}).(Principal)
	return p, ok
}
