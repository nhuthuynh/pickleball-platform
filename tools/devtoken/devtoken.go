// Package devtoken mints RS256 JWTs against the repository's committed
// local-development key fixture, and derives the JWKS document cmd/server
// verifies them with.
//
// # Why this exists (issue #160)
//
// T13.5 made a nil auth.TokenVerifier a startup failure, which is correct: a
// process that declares authenticated RPCs and holds nothing that can verify a
// token must not start. The cost, disclosed at the time, is that `make up` and
// a bare `go run ./cmd/server` stopped working, because neither sets
// AUTH_ISSUER / AUTH_AUDIENCE / AUTH_JWKS_FILE and this project has no
// identity provider to point them at.
//
// The fix #160 asks for — and the only fix T13.5's own reasoning leaves open —
// is *real key material a real verifier really verifies*, never an opt-out
// flag. An AUTH_DISABLED-style escape hatch would re-create the fail-open path
// the hardening ticket existed to close, with one extra step and a chance of
// travelling to production in a copied env file. So: a committed keypair, the
// JWKS derived from it, and this package to sign tokens with it. Local
// development then exercises the same verification code the deployed path
// does, rather than a bypass of it.
//
// # Why it lives under tools/ and not internal/platform/auth
//
// internal/platform/auth is the security spine; nothing that signs tokens
// belongs inside it, and T14.9 is explicitly forbidden from changing its
// behaviour. tools/ is where build-and-development tooling that carries real
// logic already lives (tools/vulngate), and `make test-tools` — part of
// `make ci-checks` — already runs its tests. Putting the logic here and a
// thin flag-parsing main in cmd/devtoken mirrors tools/vulngate + cmd/vulngate
// exactly, and means these tests run in CI rather than in no gate at all.
//
// # What stops this being a production foothold
//
//   - The key material is committed, therefore public, therefore worthless.
//     It is named so in the filename, in the file, and in dev/auth/README.md.
//   - LoadPrivateJWK refuses any key whose `kid` does not carry the
//     DevKeyIDPrefix marker, so this command cannot be repurposed to sign with
//     a real provider's signing key that happens to be lying on disk.
//   - The fixture issuer and audience are under RFC 2606's reserved
//     `.invalid` TLD, which can never resolve and can never be a real
//     provider's issuer URL.
//   - cmd/server has no fallback key path at all: it trusts exactly the JWKS
//     that AUTH_JWKS_FILE names. A deployment pointed at a real provider never
//     loads this key, so a token minted here cannot be presented to it. T14.9
//     changed nothing in cmd/server, by design (the ticket scopes it out) and
//     by necessity (there is nothing there to weaken).
package devtoken

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// The fixture's identity, in one place. docker-compose.yml's AUTH_* values,
// cmd/devtoken's flag defaults, dev/auth/README.md and the committed fixture
// files all have to agree; TestDockerComposeWiresTheFixture fails if the
// compose file drifts from these constants.
const (
	// Issuer is the fixture's `iss`. `.invalid` is reserved by RFC 2606 and
	// is guaranteed never to resolve, so this string cannot collide with a
	// real identity provider's issuer URL — including by a copy-paste into a
	// deployment's configuration, which would then verify nothing but this
	// fixture's own tokens.
	Issuer = "https://dev-auth.pickleball.invalid/"

	// Audience is the fixture's `aud`: the API these dev tokens are minted
	// for, and no other. Same `.invalid` reasoning as Issuer.
	Audience = "https://api.pickleball.invalid/dev"

	// KeyID is the fixture key's `kid`. It appears in every token this
	// package mints and in the JWKS cmd/server loads, so "which key signed
	// this?" is answerable from a log line alone.
	KeyID = "dev-only-insecure-do-not-trust"

	// DevKeyIDPrefix is the marker LoadPrivateJWK requires. See the package
	// doc: it is what keeps this tool pointed at fixtures.
	DevKeyIDPrefix = "dev-only-"

	// Subject is the default `sub`. It is shaped like a real identity
	// provider's subject (Auth0 renders subjects as `<connection>|<id>`) on
	// purpose: a dev token whose subject were a bare "user-1" would let a
	// handler that mishandles provider-shaped subjects pass locally and fail
	// against a real provider.
	Subject = "dev|local-user-1"

	// DefaultTTL is how long a minted token lives. Long enough to work
	// through a manual curl session, short enough that one pasted into a
	// scratch file stops working the same day.
	DefaultTTL = 8 * time.Hour

	// PrivateJWKFile and JWKSFile are the committed fixture paths, relative
	// to the repository root.
	PrivateJWKFile = "dev/auth/dev-only-insecure.private-jwk.json"
	JWKSFile       = "dev/auth/dev-only-insecure.jwks.json"
)

// minModulusBits mirrors internal/platform/auth/rs256's floor. Duplicated
// rather than imported because the two enforce it for different reasons —
// there it is a trust decision about a key from an untrusted JWKS, here it is
// a guard against regenerating the fixture too weak for the verifier to accept
// — and because tools/ must not become a reason internal/platform/auth cannot
// change. The test suite mints against the real verifier, so a drift between
// them surfaces as a failing test, not as a silently unusable fixture.
const minModulusBits = 2048

// warning is stamped into both fixture files. The files are the first thing a
// reader opens after seeing AUTH_JWKS_FILE in docker-compose.yml, so the
// "this is not a secret, and must never be used anywhere real" statement lives
// in them rather than only in a README beside them.
var warning = []string{
	"DEV-ONLY FIXTURE — NOT A SECRET. This key material is committed to a",
	"public repository on purpose, which is exactly why it is worthless.",
	"NEVER use it in any environment that matters: anyone who has read this",
	"repository can mint a token that this key verifies.",
	"Regenerate with: go run ./cmd/devtoken -regenerate",
	"See dev/auth/README.md and issue #160.",
}

// PrivateKey is the fixture signing key: an RSA private key plus the `kid`
// that names it in the JWKS.
type PrivateKey struct {
	keyID string
	key   *rsa.PrivateKey
}

// KeyID returns the key's `kid`.
func (k *PrivateKey) KeyID() string { return k.keyID }

// GenerateKeyPair mints a fresh fixture key. It is how the committed fixture
// was produced and how it is rotated (`go run ./cmd/devtoken -regenerate`);
// nothing in the serving path calls it.
func GenerateKeyPair(bits int) (*PrivateKey, error) {
	if bits < minModulusBits {
		return nil, fmt.Errorf("devtoken: %d-bit key requested, the verifier requires at least %d", bits, minModulusBits)
	}
	key, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, fmt.Errorf("devtoken: generating key: %w", err)
	}
	return &PrivateKey{keyID: KeyID, key: key}, nil
}

// privateJWK is the on-disk shape of the private half: an RFC 7517 JWK with
// RFC 7518 §6.3.2's private members.
//
// A JWK rather than a PEM, deliberately. The public half beside it is already
// a JWK Set — that is the format cmd/server parses — so one format covers
// both, and a reader who understands one understands the other. It also keeps
// a deliberately-public fixture from tripping every secret scanner that
// pattern-matches "-----BEGIN RSA PRIVATE KEY-----": #160's own caveat warns
// that this repo's security gate and GitHub's push protection would flag a PEM
// here, and a standing false positive on a file that is meant to be public
// trains people to wave real alerts through. The key material is identical
// either way; only the encoding differs, and this is the encoding the rest of
// the auth path already speaks.
type privateJWK struct {
	Warning []string `json:"_WARNING"`
	Kty     string   `json:"kty"`
	Use     string   `json:"use"`
	Alg     string   `json:"alg"`
	Kid     string   `json:"kid"`
	N       string   `json:"n"`
	E       string   `json:"e"`
	D       string   `json:"d"`
	P       string   `json:"p"`
	Q       string   `json:"q"`
	Dp      string   `json:"dp"`
	Dq      string   `json:"dq"`
	Qi      string   `json:"qi"`
}

// publicJWK and jwksDocument are the public half: exactly the subset
// internal/platform/auth/rs256 parses, plus the warning banner.
type publicJWK struct {
	Kty string `json:"kty"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	Kid string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type jwksDocument struct {
	Warning []string    `json:"_WARNING"`
	Keys    []publicJWK `json:"keys"`
}

// MarshalPrivateJWK renders the private half as the committed fixture file.
func (k *PrivateKey) MarshalPrivateJWK() ([]byte, error) {
	k.key.Precompute()
	if len(k.key.Primes) != 2 {
		return nil, fmt.Errorf("devtoken: expected a two-prime RSA key, got %d primes", len(k.key.Primes))
	}
	p, q := k.key.Primes[0], k.key.Primes[1]

	doc := privateJWK{
		Warning: warning,
		Kty:     "RSA",
		Use:     "sig",
		Alg:     "RS256",
		Kid:     k.keyID,
		N:       encode(k.key.N),
		E:       encode(big.NewInt(int64(k.key.E))),
		D:       encode(k.key.D),
		P:       encode(p),
		Q:       encode(q),
		Dp:      encode(k.key.Precomputed.Dp),
		Dq:      encode(k.key.Precomputed.Dq),
		Qi:      encode(k.key.Precomputed.Qinv),
	}
	return marshalFixture(doc)
}

// MarshalJWKS renders the public half: the document AUTH_JWKS_FILE points at.
//
// Deriving it here rather than maintaining it by hand is what makes the two
// committed files provably halves of one keypair — TestCommittedJWKSIsDerived
// FromTheCommittedPrivateKey regenerates it and compares bytes, so a fixture
// rotation that updated one file and not the other fails a test instead of
// producing a JWKS nobody can mint against.
func (k *PrivateKey) MarshalJWKS() ([]byte, error) {
	doc := jwksDocument{
		Warning: warning,
		Keys: []publicJWK{{
			Kty: "RSA",
			Use: "sig",
			Alg: "RS256",
			Kid: k.keyID,
			N:   encode(k.key.N),
			E:   encode(big.NewInt(int64(k.key.E))),
		}},
	}
	return marshalFixture(doc)
}

func marshalFixture(v any) ([]byte, error) {
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("devtoken: encoding fixture: %w", err)
	}
	return append(out, '\n'), nil
}

// LoadPrivateJWK parses a fixture private key.
//
// Every rejection below is a refusal to sign, never a warning: this command's
// entire output is a credential, so a key it is unsure about must produce no
// token at all.
func LoadPrivateJWK(document []byte) (*PrivateKey, error) {
	var doc privateJWK
	if err := json.Unmarshal(document, &doc); err != nil {
		return nil, fmt.Errorf("devtoken: parsing private jwk: %w", err)
	}
	if doc.Kty != "RSA" {
		return nil, fmt.Errorf("devtoken: unsupported kty %q, want RSA", doc.Kty)
	}
	if doc.Alg != "" && doc.Alg != "RS256" {
		return nil, fmt.Errorf("devtoken: alg %q is not RS256", doc.Alg)
	}
	// The guard that keeps this a fixture tool. Without it, `-key` turns
	// cmd/devtoken into a general-purpose token forger pointed at whatever
	// private key is on the machine — including a real provider's, if one
	// ever leaks onto a developer's disk. With it, the tool signs dev
	// fixtures and says so when asked to do anything else.
	if !strings.HasPrefix(doc.Kid, DevKeyIDPrefix) {
		return nil, fmt.Errorf(
			"devtoken: refusing to sign with kid %q: this tool mints DEV FIXTURE tokens only, "+
				"and a fixture key's kid must start with %q", doc.Kid, DevKeyIDPrefix)
	}

	n, err := decode(doc.N, "n")
	if err != nil {
		return nil, err
	}
	if n.BitLen() < minModulusBits {
		return nil, fmt.Errorf("devtoken: modulus is %d bits, the verifier requires at least %d", n.BitLen(), minModulusBits)
	}
	e, err := decode(doc.E, "e")
	if err != nil {
		return nil, err
	}
	if !e.IsInt64() || e.Int64() < 3 || e.Int64() > (1<<31-1) {
		return nil, fmt.Errorf("devtoken: public exponent %s is out of range", e)
	}
	d, err := decode(doc.D, "d")
	if err != nil {
		return nil, err
	}
	p, err := decode(doc.P, "p")
	if err != nil {
		return nil, err
	}
	q, err := decode(doc.Q, "q")
	if err != nil {
		return nil, err
	}

	key := &rsa.PrivateKey{
		PublicKey: rsa.PublicKey{N: n, E: int(e.Int64())},
		D:         d,
		Primes:    []*big.Int{p, q},
	}
	// dp/dq/qi are written to the file for RFC 7518 completeness and
	// recomputed here rather than trusted: Precompute derives them from p, q
	// and d, so a file whose CRT members were edited to disagree with its
	// primes cannot influence a signature. Validate then confirms the whole
	// key is internally consistent.
	key.Precompute()
	if err := key.Validate(); err != nil {
		return nil, fmt.Errorf("devtoken: private jwk is not a consistent RSA key: %w", err)
	}

	return &PrivateKey{keyID: doc.Kid, key: key}, nil
}

// MintParams describes one token to mint.
type MintParams struct {
	// Issuer, Audience and Subject become `iss`, `aud` and `sub`. All three
	// are required: the verifier checks issuer and audience exactly, and
	// treats an empty subject as no principal at all, so a token missing any
	// of them is a token that cannot succeed.
	Issuer   string
	Audience string
	Subject  string

	// Scope becomes the space-separated `scope` claim (RFC 9068). Optional —
	// no handler in this repo requires a scope today.
	Scope string

	// TTL is how long the token is valid for. Zero means DefaultTTL;
	// negative is rejected rather than quietly clamped, because "mint me an
	// already-expired token" is a plausible test intent that the caller
	// should express with Now instead, where it is legible.
	TTL time.Duration

	// Now is a clock seam for tests. Nil means time.Now.
	Now func() time.Time
}

// Mint signs a token with this key.
func (k *PrivateKey) Mint(p MintParams) (string, error) {
	if strings.TrimSpace(p.Issuer) == "" {
		return "", errors.New("devtoken: Issuer is required")
	}
	if strings.TrimSpace(p.Audience) == "" {
		return "", errors.New("devtoken: Audience is required")
	}
	if strings.TrimSpace(p.Subject) == "" {
		return "", errors.New("devtoken: Subject is required (a token with no sub carries no principal)")
	}
	if p.TTL < 0 {
		return "", fmt.Errorf("devtoken: TTL must not be negative, got %s", p.TTL)
	}

	ttl := p.TTL
	if ttl == 0 {
		ttl = DefaultTTL
	}
	now := p.Now
	if now == nil {
		now = time.Now
	}
	issuedAt := now().UTC()

	claims := struct {
		jwt.RegisteredClaims
		Scope string `json:"scope,omitempty"`
	}{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    p.Issuer,
			Subject:   p.Subject,
			Audience:  jwt.ClaimStrings{p.Audience},
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			NotBefore: jwt.NewNumericDate(issuedAt),
			ExpiresAt: jwt.NewNumericDate(issuedAt.Add(ttl)),
		},
		Scope: strings.TrimSpace(p.Scope),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	// The verifier selects its key by `kid` and never falls back to "the
	// only key we have", so a token without this header cannot verify.
	token.Header["kid"] = k.keyID

	signed, err := token.SignedString(k.key)
	if err != nil {
		return "", fmt.Errorf("devtoken: signing: %w", err)
	}
	return signed, nil
}

// encode/decode handle JWK's base64url-without-padding big-endian integers
// (RFC 7518 §6.3).
func encode(i *big.Int) string {
	return base64.RawURLEncoding.EncodeToString(i.Bytes())
}

func decode(value, member string) (*big.Int, error) {
	if value == "" {
		return nil, fmt.Errorf("devtoken: private jwk is missing member %q", member)
	}
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("devtoken: member %q is not base64url: %w", member, err)
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("devtoken: member %q is empty", member)
	}
	return new(big.Int).SetBytes(raw), nil
}
