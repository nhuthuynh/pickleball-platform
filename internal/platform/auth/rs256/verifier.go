// Package rs256 verifies RS256-signed JWT access tokens against a JWKS-style
// key source, and is the real implementation behind auth.TokenVerifier.
//
// It stands to internal/platform/auth as internal/payments/adapter/stripestub
// stands to internal/payments/port: the interface next door is the seam, this
// is one replaceable implementation of it. The dependency points inward only —
// this package imports auth, auth does not import this — so a second
// implementation (a different algorithm family, or token introspection against
// a remote endpoint) drops in beside it without any caller changing.
//
// The algorithm is in the package name deliberately. RS256 is not a detail of
// the port; it is a security decision with an allowlist attached, and a
// package named for it cannot quietly grow support for a weaker one.
package rs256

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/nhuthuynh/white-label/internal/platform/auth"
)

// defaultLeeway is the clock skew tolerated on exp/nbf when Config.Leeway is
// unset. Small enough that an expired token is not usefully extended, large
// enough to absorb ordinary NTP drift between the identity provider's clock
// and this process's.
const defaultLeeway = 30 * time.Second

// Config describes the one issuer/audience pair a Verifier accepts.
//
// Every field except Leeway and Now is required, and NewVerifier refuses to
// build a Verifier without them. That refusal is the point: an empty Issuer or
// Audience would make the corresponding check vacuously pass, producing a
// verifier that checks the signature and nothing else — which looks like a
// real check in code review and in logs, and is not one. Failing at
// construction turns that into a startup error instead of a silent downgrade.
type Config struct {
	// Issuer is the exact `iss` claim value to require, as the provider
	// publishes it (including any trailing slash — this is an exact string
	// comparison, not a URL normalization).
	Issuer string

	// Audience is the exact `aud` value identifying this API. A token is
	// accepted when its audience list contains this value; tokens minted for
	// a different API of the same issuer are rejected, which is the check
	// that stops a token from one service being replayed at this one.
	Audience string

	// Keys supplies public keys by `kid`.
	Keys KeySource

	// Leeway is the clock skew tolerated on exp and nbf. Zero means
	// defaultLeeway; it must not be negative.
	Leeway time.Duration

	// Now is a clock seam for tests. Nil means time.Now.
	Now func() time.Time
}

// Verifier is a configured RS256 token verifier. It is immutable after
// construction and safe for concurrent use.
type Verifier struct {
	parser   *jwt.Parser
	keys     KeySource
	audience string
}

// claims is the claim set this verifier reads: the RFC 7519 registered claims
// plus RFC 9068's `scope`.
type claims struct {
	jwt.RegisteredClaims
	Scope string `json:"scope,omitempty"`
}

// NewVerifier builds a Verifier, or returns an error explaining which part of
// the configuration would have made verification weaker than it looks.
func NewVerifier(cfg Config) (*Verifier, error) {
	if strings.TrimSpace(cfg.Issuer) == "" {
		return nil, errors.New("rs256: Issuer is required (an empty issuer would make the iss check vacuous)")
	}
	if strings.TrimSpace(cfg.Audience) == "" {
		return nil, errors.New("rs256: Audience is required (an empty audience would make the aud check vacuous)")
	}
	if cfg.Keys == nil {
		return nil, errors.New("rs256: Keys is required (there is nothing to verify signatures against)")
	}
	if cfg.Leeway < 0 {
		return nil, fmt.Errorf("rs256: Leeway must not be negative, got %s", cfg.Leeway)
	}

	leeway := cfg.Leeway
	if leeway == 0 {
		leeway = defaultLeeway
	}
	now := cfg.Now
	if now == nil {
		now = time.Now
	}

	return &Verifier{
		parser: jwt.NewParser(
			// The algorithm allowlist, and the single most important option
			// here. Without it the parser honours the token's own `alg`
			// header, which lets an attacker present an HS256 token signed
			// with the RSA *public* key — a value a JWKS publishes to the
			// world — and have it verify. Pinning RS256 also rejects
			// `alg: none` outright.
			jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}),
			jwt.WithIssuer(cfg.Issuer),
			jwt.WithAudience(cfg.Audience),
			// A token with no exp is a credential that never stops working.
			jwt.WithExpirationRequired(),
			jwt.WithLeeway(leeway),
			jwt.WithTimeFunc(now),
		),
		keys:     cfg.Keys,
		audience: cfg.Audience,
	}, nil
}

// Verify implements auth.TokenVerifier.
//
// On any failure it returns the zero auth.Principal and an error wrapping
// exactly one of the auth package's sentinels, so callers never see a
// jwt-library error type (CLAUDE.md rule 5).
func (v *Verifier) Verify(ctx context.Context, rawToken string) (auth.Principal, error) {
	var parsed claims

	// keyErr captures the key source's own error, because the jwt library
	// renders a keyfunc failure into jwt.ErrTokenUnverifiable's *message*
	// with %v rather than wrapping it — so by the time ParseWithClaims
	// returns, errors.Is can no longer see the underlying cause. Holding onto
	// it here is what lets an operator distinguish "the JWKS endpoint is
	// down" from "this caller named a kid we do not have", which are an
	// incident and a non-event respectively.
	var keyErr error

	// keyFunc runs after the header is parsed and before the signature is
	// checked, which is the only point at which the token's kid is known.
	keyFunc := func(token *jwt.Token) (any, error) {
		// Belt and braces alongside WithValidMethods: if that option were
		// ever dropped, this keeps an HMAC-signed token from being handed an
		// *rsa.PublicKey to verify against.
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			keyErr = fmt.Errorf("%w: unexpected signing method %q", auth.ErrTokenSignature, token.Method.Alg())
			return nil, keyErr
		}

		kid, _ := token.Header["kid"].(string)
		key, err := v.keys.PublicKey(ctx, kid)
		if err != nil {
			keyErr = err
			return nil, err
		}
		return key, nil
	}

	if _, err := v.parser.ParseWithClaims(rawToken, &parsed, keyFunc); err != nil {
		if keyErr != nil {
			return auth.Principal{}, translateKeyErr(keyErr)
		}
		return auth.Principal{}, translate(err)
	}

	// jwt validates `sub`'s syntax but never its presence — a token whose
	// subject is absent or empty parses cleanly. It is also the one claim
	// this platform treats as authoritative, so an empty one is not a
	// degraded principal, it is not a principal at all.
	subject := strings.TrimSpace(parsed.Subject)
	if subject == "" {
		return auth.Principal{}, fmt.Errorf("%w: sub", auth.ErrTokenClaimMissing)
	}

	return auth.Principal{
		Subject:   subject,
		Issuer:    parsed.Issuer,
		Audience:  []string(parsed.Audience),
		Scopes:    strings.Fields(parsed.Scope),
		IssuedAt:  claimTime(parsed.IssuedAt),
		ExpiresAt: claimTime(parsed.ExpiresAt),
	}, nil
}

// translateKeyErr maps a key-selection failure, preserving the key source's
// own error in the chain so errors.Is can still reach it.
//
// A source that already speaks this vocabulary (StaticKeys reporting an
// unknown kid, or the signing-method guard) is passed through as-is rather
// than re-wrapped, so the message stays one sentence instead of two.
func translateKeyErr(err error) error {
	if errors.Is(err, auth.ErrKeyUnavailable) || errors.Is(err, auth.ErrTokenSignature) {
		return fmt.Errorf("rs256: %w", err)
	}
	return fmt.Errorf("%w: %w", auth.ErrKeyUnavailable, err)
}

// translate maps a jwt-library error onto this platform's error vocabulary.
//
// jwt v5 joins multiple validation failures into one error, so the order here
// is a priority order, not a switch. It is deliberately most-specific-first:
// an expired token that *also* has the wrong audience should be reported as
// the audience mismatch, because a token minted for another service is a
// materially more interesting event for an operator than an old one.
func translate(err error) error {
	switch {
	// A key-source failure is checked first because it is the one case that
	// is not the caller's fault, and translating it to a token error would
	// lose that distinction entirely (see auth.IsTokenRejection).
	case errors.Is(err, auth.ErrKeyUnavailable):
		return fmt.Errorf("rs256: %w", err)
	case errors.Is(err, auth.ErrTokenSignature):
		return fmt.Errorf("rs256: %w", err)

	case errors.Is(err, jwt.ErrTokenMalformed):
		return fmt.Errorf("%w: %v", auth.ErrTokenMalformed, err)
	case errors.Is(err, jwt.ErrTokenSignatureInvalid):
		return fmt.Errorf("%w: %v", auth.ErrTokenSignature, err)
	case errors.Is(err, jwt.ErrTokenInvalidAudience):
		return fmt.Errorf("%w: %v", auth.ErrTokenAudience, err)
	case errors.Is(err, jwt.ErrTokenInvalidIssuer):
		return fmt.Errorf("%w: %v", auth.ErrTokenIssuer, err)
	case errors.Is(err, jwt.ErrTokenExpired):
		return fmt.Errorf("%w: %v", auth.ErrTokenExpired, err)
	case errors.Is(err, jwt.ErrTokenNotValidYet):
		return fmt.Errorf("%w: %v", auth.ErrTokenNotYetValid, err)
	case errors.Is(err, jwt.ErrTokenRequiredClaimMissing):
		return fmt.Errorf("%w: %v", auth.ErrTokenClaimMissing, err)

	// jwt.ErrTokenUnverifiable means the parser could not even attempt a
	// signature check — in this package that is always a key-selection
	// failure, since keyFunc is the only thing that can cause it.
	case errors.Is(err, jwt.ErrTokenUnverifiable):
		return fmt.Errorf("%w: %v", auth.ErrKeyUnavailable, err)

	default:
		// Anything unrecognised is treated as a rejection rather than passed
		// through: an unclassified verification failure must never be
		// mistaken for a success, and must not leak a library error type past
		// this boundary.
		return fmt.Errorf("%w: %v", auth.ErrTokenMalformed, err)
	}
}

func claimTime(d *jwt.NumericDate) time.Time {
	if d == nil {
		return time.Time{}
	}
	return d.Time
}

// compile-time proof that this package satisfies the port it exists to
// implement — the same check internal/payments/adapter/stripestub carries.
var _ auth.TokenVerifier = (*Verifier)(nil)
