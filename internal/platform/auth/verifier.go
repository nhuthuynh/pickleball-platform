package auth

import (
	"context"
	"errors"
)

// TokenVerifier turns a raw bearer token into a verified Principal.
//
// This is the seam, and it is shaped the way this project already shapes its
// outbound ports: internal/payments/port.PaymentProcessor describes the
// vendor-agnostic lifecycle a card payment has, and
// internal/payments/adapter/stripestub implements it, so swapping in a real
// Stripe adapter later is an adapter-only change. The same split applies here
// — the interface is expressed in this backend's own primitives (a raw token
// string in, a Principal out) rather than any identity provider's SDK types,
// so a real Auth0-backed implementation and the RS256 one in the rs256
// subpackage are interchangeable without a single caller changing.
//
// The shape is not invented here: docs/requirements/research-security-compliance.md
// §3 pinned it in advance, from RFC 9068 (JWT Profile for OAuth 2.0 Access
// Tokens) and RFC 8725 (JWT Best Current Practices) — "a TokenVerifier port
// whose method takes a raw bearer token string and returns a small,
// backend-owned claims struct: subject/user ID, roles or scope list, expiry".
// Treat this signature as a promise from day one.
//
// # Implementations must verify, not merely parse
//
// An implementation that checks the signature and stops is worse than no
// verifier at all, because it looks like a check. A conforming implementation
// checks, at minimum:
//
//   - the signature, against a key it selected by the token's `kid` from a
//     trusted key source, restricted to an explicit algorithm allowlist (an
//     implementation that honours the token's own `alg` header unrestricted is
//     vulnerable to algorithm confusion);
//   - `exp`, and `nbf` when present, against a clock, with bounded leeway;
//   - `iss`, against the exact issuer it was configured for;
//   - `aud`, against the exact audience it was configured for;
//   - that `sub` is present and non-empty.
//
// # Error contract
//
// Implementations return exactly one of the sentinel errors below, wrapped
// with whatever detail helps an operator (CLAUDE.md rule 5: adapters translate
// infra errors into this package's errors, so callers never see a JWT
// library's or an HTTP client's error type). Callers distinguish them with
// errors.Is, and use IsTokenRejection to separate "the caller's token is bad"
// from "our key source is broken" — those map to different gRPC codes.
//
// Verify must never return a partially populated Principal alongside an
// error; on any failure the Principal is the zero value.
type TokenVerifier interface {
	Verify(ctx context.Context, rawToken string) (Principal, error)
}

// The sentinel errors a TokenVerifier (or the interceptor's own token
// extraction) reports. They are deliberately fine-grained: the *caller* almost
// always collapses them to a single codes.Unauthenticated, but an operator
// reading logs needs to know whether the fleet is seeing expired tokens
// (clock skew, or an IdP session-length change) or bad signatures (a key
// rotation nobody propagated), and those are very different incidents.
var (
	// ErrNoToken means no bearer token was presented at all.
	ErrNoToken = errors.New("auth: no bearer token presented")

	// ErrTokenMalformed means the value presented is not a well-formed JWT.
	ErrTokenMalformed = errors.New("auth: token is malformed")

	// ErrTokenSignature means the signature did not verify, or the token was
	// signed with an algorithm this verifier does not accept. The two are
	// deliberately one error: both mean "this token was not minted by the
	// issuer we trust", and distinguishing them for the caller only helps an
	// attacker probe the allowlist.
	ErrTokenSignature = errors.New("auth: token signature is invalid")

	// ErrTokenExpired means `exp` is in the past, beyond any configured leeway.
	ErrTokenExpired = errors.New("auth: token has expired")

	// ErrTokenNotYetValid means `nbf` is in the future, beyond any leeway.
	ErrTokenNotYetValid = errors.New("auth: token is not valid yet")

	// ErrTokenIssuer means `iss` is not the issuer this verifier trusts.
	ErrTokenIssuer = errors.New("auth: token issuer is not trusted")

	// ErrTokenAudience means `aud` is present but does not include this API's
	// audience — i.e. a token minted for a different service was replayed at
	// this one. A token carrying no `aud` at all is ErrTokenClaimMissing
	// rather than this: both are rejections, and both become
	// codes.Unauthenticated, but "the claim is absent" and "the claim names
	// someone else" are different enough that an operator should not have to
	// guess which one the fleet is producing.
	ErrTokenAudience = errors.New("auth: token audience does not match this api")

	// ErrTokenClaimMissing means a claim this verifier requires is absent:
	// `exp` (a token that never expires is not acceptable), `sub` (a token
	// that identifies nobody), or `aud` (a token scoped to nothing).
	ErrTokenClaimMissing = errors.New("auth: token is missing a required claim")

	// ErrKeyUnavailable means the verifier could not obtain the key needed to
	// check the signature: an unknown `kid`, an absent `kid`, or a key source
	// that failed.
	//
	// This one is not like the others. An unknown kid is the caller's problem
	// (their token was signed by something we do not trust), but a key source
	// that is *down* is ours, and reporting our outage to every caller as
	// "your credentials are invalid" sends an entire fleet of clients into a
	// pointless re-authentication loop during an incident. IsTokenRejection
	// therefore reports false for it; see ADR-0013.
	ErrKeyUnavailable = errors.New("auth: signing key unavailable")

	// ErrVerifierPanicked means a TokenVerifier implementation panicked
	// instead of returning, so verification reached no conclusion at all.
	//
	// Implementations do not return this — by definition they cannot. The
	// interceptors synthesize it when they recover a panic out of Verify (see
	// interceptor.go, issue #135), so that "the verifier is broken" is a case
	// the request path handles explicitly rather than a goroutine unwinding
	// past it.
	//
	// Like ErrKeyUnavailable it is *our* failure and not the caller's, so
	// IsTokenRejection reports false for it. Unlike ErrKeyUnavailable it is a
	// bug rather than an outage, which is why the interceptors answer it with
	// codes.Internal rather than the codes.Unavailable ADR-0013 §5 assigns to
	// an unreachable key source.
	ErrVerifierPanicked = errors.New("auth: token verifier panicked")
)

// tokenRejections is the set of errors that mean "the token the caller
// presented is not acceptable" — the set that maps to codes.Unauthenticated
// once enforcement is turned on.
var tokenRejections = []error{
	ErrNoToken,
	ErrTokenMalformed,
	ErrTokenSignature,
	ErrTokenExpired,
	ErrTokenNotYetValid,
	ErrTokenIssuer,
	ErrTokenAudience,
	ErrTokenClaimMissing,
}

// IsTokenRejection reports whether err means the caller's token was rejected,
// as opposed to this service failing to do its job.
//
// It exists so the migration tickets that turn enforcement on have one place
// to ask the question, rather than each writing its own errors.Is chain and
// each getting the ErrKeyUnavailable case subtly wrong:
//
//	if err != nil {
//	    if auth.IsTokenRejection(err) {
//	        return status.Error(codes.Unauthenticated, "...")
//	    }
//	    return status.Error(codes.Unavailable, "...")
//	}
//
// It reports false for nil, so it cannot be misread as "did verification
// fail".
func IsTokenRejection(err error) bool {
	if err == nil {
		return false
	}
	for _, sentinel := range tokenRejections {
		if errors.Is(err, sentinel) {
			return true
		}
	}
	return false
}
