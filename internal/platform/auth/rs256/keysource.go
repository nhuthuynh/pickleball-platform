package rs256

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/nhuthuynh/white-label/internal/platform/auth"
)

// minModulusBits is the smallest RSA modulus this package will verify against.
//
// RFC 8725 §3.5 and current guidance put the floor for RS256 at 2048 bits.
// Enforcing it at parse time rather than at verification time means a key too
// weak to trust never enters the key set at all, so no token signed by one can
// reach a signature check.
const minModulusBits = 2048

// KeySource supplies the public key a token's `kid` header names.
//
// It is a separate seam from auth.TokenVerifier on purpose. Verifying a token
// and *obtaining the keys to verify it with* have completely different failure
// modes and lifetimes: verification is pure and instant, key retrieval is
// remote, cached, and subject to rotation. Splitting them means the network
// half can be replaced — with an HTTP JWKS client, a cache, a rotation policy
// — without touching a line of the verification logic that the security of
// this platform actually rests on.
//
// Implementations return an error wrapping auth.ErrKeyUnavailable when they
// cannot supply a key, whether because the kid is unknown or because the
// source itself failed.
type KeySource interface {
	PublicKey(ctx context.Context, keyID string) (*rsa.PublicKey, error)
}

// StaticKeys is an in-process, immutable KeySource.
//
// It is what the tests use, and what cmd/server uses when AUTH_JWKS_FILE names
// a JWKS on disk — including the committed local-dev fixture in dev/auth/
// (T14.9). What it deliberately does not do is fetch: there is no HTTP client,
// no cache, and no rotation handling here. That is RemoteKeys' job (T15.7,
// issue #137), which fetches AUTH_JWKS_URL and slots in behind this same
// interface without either of them knowing about the other.
//
// A file source cannot see a rotation: the document is read once at startup,
// so rotating the provider's keys under a deployment pointed at a file
// invalidates every token until someone re-exports the file and restarts.
// That is correct for a fixture and wrong for a live provider, which is the
// whole reason both implementations exist.
//
// Being immutable after construction makes it safe for concurrent use by every
// in-flight request without a lock.
type StaticKeys struct {
	keys map[string]*rsa.PublicKey
}

// NewStaticKeys builds a KeySource from keys already in hand, indexed by key
// ID. The map is copied, so a later mutation by the caller cannot change what
// this verifier trusts.
func NewStaticKeys(keys map[string]*rsa.PublicKey) *StaticKeys {
	copied := make(map[string]*rsa.PublicKey, len(keys))
	for kid, key := range keys {
		copied[kid] = key
	}
	return &StaticKeys{keys: copied}
}

// jwks and jwk model the subset of RFC 7517 this backend needs: RSA signing
// keys. Fields this package does not use are intentionally absent rather than
// parsed-and-ignored.
type jwks struct {
	Keys []jwk `json:"keys"`
}

type jwk struct {
	Kty string `json:"kty"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	Kid string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// NewStaticKeysFromJWKS parses an RFC 7517 JWK Set — the document a real
// identity provider serves at /.well-known/jwks.json — into a KeySource.
//
// A JWKS is untrusted input (docs/requirements/research-security-compliance.md
// §1, API10: "treat JWKS/discovery responses as untrusted input to parse
// defensively"), so every malformed shape is a construction error rather than
// a silently-skipped entry. That choice matters more than it looks: skipping a
// bad entry would yield a key set missing the key everyone's tokens are signed
// with, and in observe-only mode the resulting total verification failure is
// invisible — no caller is told, because nothing is enforced. A startup error
// is loud; a quietly empty key set is not.
func NewStaticKeysFromJWKS(document []byte) (*StaticKeys, error) {
	var set jwks
	if err := json.Unmarshal(document, &set); err != nil {
		return nil, fmt.Errorf("rs256: parsing jwks: %w", err)
	}
	if len(set.Keys) == 0 {
		return nil, fmt.Errorf("rs256: jwks contains no keys")
	}

	keys := make(map[string]*rsa.PublicKey, len(set.Keys))
	for i, k := range set.Keys {
		if k.Kty != "RSA" {
			return nil, fmt.Errorf("rs256: jwks key %d: unsupported kty %q, want RSA", i, k.Kty)
		}
		// `use` and `alg` are optional in RFC 7517, but when a provider does
		// state them, honouring the statement is the point of publishing it:
		// an encryption key must never be accepted as a signing key.
		if k.Use != "" && k.Use != "sig" {
			return nil, fmt.Errorf("rs256: jwks key %d (kid %q): use %q is not a signing key", i, k.Kid, k.Use)
		}
		if k.Alg != "" && k.Alg != "RS256" {
			return nil, fmt.Errorf("rs256: jwks key %d (kid %q): alg %q is not RS256", i, k.Kid, k.Alg)
		}
		if k.Kid == "" {
			return nil, fmt.Errorf("rs256: jwks key %d: missing kid", i)
		}
		if _, duplicate := keys[k.Kid]; duplicate {
			// Two keys under one kid makes key selection ambiguous, and an
			// ambiguous selection is a verification result nobody can reason
			// about after the fact.
			return nil, fmt.Errorf("rs256: jwks contains duplicate kid %q", k.Kid)
		}

		pub, err := rsaPublicKey(k)
		if err != nil {
			return nil, fmt.Errorf("rs256: jwks key %d (kid %q): %w", i, k.Kid, err)
		}
		keys[k.Kid] = pub
	}

	return &StaticKeys{keys: keys}, nil
}

// rsaPublicKey rebuilds an RSA public key from a JWK's base64url modulus and
// exponent (RFC 7518 §6.3.1).
func rsaPublicKey(k jwk) (*rsa.PublicKey, error) {
	// RawURLEncoding: JWK members are base64url without padding.
	modulus, err := base64.RawURLEncoding.DecodeString(k.N)
	if err != nil {
		return nil, fmt.Errorf("modulus is not base64url: %w", err)
	}
	if len(modulus) == 0 {
		return nil, fmt.Errorf("modulus is empty")
	}

	exponent, err := base64.RawURLEncoding.DecodeString(k.E)
	if err != nil {
		return nil, fmt.Errorf("exponent is not base64url: %w", err)
	}
	if len(exponent) == 0 {
		return nil, fmt.Errorf("exponent is empty")
	}

	n := new(big.Int).SetBytes(modulus)
	if n.BitLen() < minModulusBits {
		return nil, fmt.Errorf("modulus is %d bits, want at least %d", n.BitLen(), minModulusBits)
	}

	e := new(big.Int).SetBytes(exponent)
	// rsa.PublicKey.E is an int; a public exponent that does not fit is not a
	// key any real provider publishes, and big.Int.IsInt64 is the honest check.
	if !e.IsInt64() || e.Int64() < 3 || e.Int64() > (1<<31-1) {
		return nil, fmt.Errorf("public exponent %s is out of range", e)
	}

	return &rsa.PublicKey{N: n, E: int(e.Int64())}, nil
}

// PublicKey returns the key registered under keyID.
//
// An unknown or empty keyID is an error, never a fallback to "the only key we
// have". Guessing here would defeat key rotation: during a rotation the whole
// point is that a token naming the retired kid must stop verifying.
func (s *StaticKeys) PublicKey(_ context.Context, keyID string) (*rsa.PublicKey, error) {
	key, ok := s.keys[keyID]
	if !ok {
		return nil, fmt.Errorf("%w: no key registered for kid %q", auth.ErrKeyUnavailable, keyID)
	}
	return key, nil
}
