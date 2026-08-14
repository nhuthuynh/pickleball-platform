package rs256_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/nhuthuynh/white-label/internal/platform/auth"
	"github.com/nhuthuynh/white-label/internal/platform/auth/rs256"
)

// All key material in this file is minted in-process, per test run. There is
// no network call, no Docker, and no external identity provider anywhere in
// this package's tests — which is the whole point: the verification half of
// auth is testable without the IdP half existing (T12 sprint plan, A1).
const (
	testIssuer   = "https://pickleball-test.example.com/"
	testAudience = "https://api.pickleball-test.example.com"
	testKeyID    = "test-key-1"
	testSubject  = "auth0|abc123"
)

// fixedNow is the instant every table case is evaluated at, so "expired" and
// "not yet valid" are properties of the token rather than of how long the
// test suite took to run.
var fixedNow = time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)

type testKeys struct {
	private *rsa.PrivateKey
	other   *rsa.PrivateKey // a second, unrelated keypair — for bad-signature cases
	source  rs256.KeySource
}

func newTestKeys(t *testing.T) testKeys {
	t.Helper()

	// 2048 is the smallest modulus RFC 8725 / current guidance accepts for
	// RS256. Generating a deliberately weak-but-fast key here would make the
	// test cheaper and the thing it proves different from the thing that ships.
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}
	other, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate second rsa key: %v", err)
	}

	return testKeys{
		private: priv,
		other:   other,
		source:  rs256.NewStaticKeys(map[string]*rsa.PublicKey{testKeyID: &priv.PublicKey}),
	}
}

// claims is the smallest struct that can express every malformed shape the
// table needs, including a `scope` claim the RegisteredClaims set has no room
// for (RFC 9068 §2.2.3).
type claims struct {
	jwt.RegisteredClaims
	Scope string `json:"scope,omitempty"`
}

type mintOpts struct {
	subject   string
	issuer    string
	audience  []string
	scope     string
	issuedAt  time.Time
	notBefore time.Time
	expiresAt time.Time
	omitExp   bool
	keyID     string
	signWith  *rsa.PrivateKey
	method    jwt.SigningMethod
	hmacKey   []byte
}

func mint(t *testing.T, keys testKeys, o mintOpts) string {
	t.Helper()

	c := claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: o.subject,
			Issuer:  o.issuer,
		},
		Scope: o.scope,
	}
	if len(o.audience) > 0 {
		c.Audience = jwt.ClaimStrings(o.audience)
	}
	if !o.issuedAt.IsZero() {
		c.IssuedAt = jwt.NewNumericDate(o.issuedAt)
	}
	if !o.notBefore.IsZero() {
		c.NotBefore = jwt.NewNumericDate(o.notBefore)
	}
	if !o.omitExp && !o.expiresAt.IsZero() {
		c.ExpiresAt = jwt.NewNumericDate(o.expiresAt)
	}

	method := o.method
	if method == nil {
		method = jwt.SigningMethodRS256
	}

	tok := jwt.NewWithClaims(method, c)
	if o.keyID != "" {
		tok.Header["kid"] = o.keyID
	}

	var (
		signed string
		err    error
	)
	switch method.Alg() {
	case "none":
		signed, err = tok.SignedString(jwt.UnsafeAllowNoneSignatureType)
	case "HS256":
		signed, err = tok.SignedString(o.hmacKey)
	default:
		key := o.signWith
		if key == nil {
			key = keys.private
		}
		signed, err = tok.SignedString(key)
	}
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}

// validOpts is the happy-path token every negative case mutates exactly one
// field of, so each row proves one check rather than a soup of them.
func validOpts() mintOpts {
	return mintOpts{
		subject:   testSubject,
		issuer:    testIssuer,
		audience:  []string{testAudience},
		scope:     "bookings:read bookings:write",
		issuedAt:  fixedNow.Add(-1 * time.Minute),
		expiresAt: fixedNow.Add(10 * time.Minute),
		keyID:     testKeyID,
	}
}

func newVerifier(t *testing.T, keys testKeys) *rs256.Verifier {
	t.Helper()

	v, err := rs256.NewVerifier(rs256.Config{
		Issuer:   testIssuer,
		Audience: testAudience,
		Keys:     keys.source,
		Now:      func() time.Time { return fixedNow },
	})
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}
	return v
}

func TestVerifierVerify(t *testing.T) {
	t.Parallel()

	keys := newTestKeys(t)

	tests := []struct {
		name string
		// token is built per-case so each row can sign with different key
		// material; a pre-built string would force one signer for all.
		token   func(t *testing.T) string
		wantErr error
		// check runs only when wantErr is nil.
		check func(t *testing.T, p auth.Principal)
	}{
		{
			name:  "valid token yields a populated principal",
			token: func(t *testing.T) string { return mint(t, keys, validOpts()) },
			check: func(t *testing.T, p auth.Principal) {
				if p.Subject != testSubject {
					t.Errorf("Subject = %q, want %q", p.Subject, testSubject)
				}
				if p.Issuer != testIssuer {
					t.Errorf("Issuer = %q, want %q", p.Issuer, testIssuer)
				}
				if len(p.Audience) != 1 || p.Audience[0] != testAudience {
					t.Errorf("Audience = %v, want [%q]", p.Audience, testAudience)
				}
				if len(p.Scopes) != 2 || p.Scopes[0] != "bookings:read" || p.Scopes[1] != "bookings:write" {
					t.Errorf("Scopes = %v, want [bookings:read bookings:write]", p.Scopes)
				}
				if !p.ExpiresAt.Equal(fixedNow.Add(10 * time.Minute)) {
					t.Errorf("ExpiresAt = %v, want %v", p.ExpiresAt, fixedNow.Add(10*time.Minute))
				}
			},
		},
		{
			name: "multi-valued audience containing ours is accepted",
			token: func(t *testing.T) string {
				o := validOpts()
				o.audience = []string{"https://other.example.com", testAudience}
				return mint(t, keys, o)
			},
			check: func(t *testing.T, p auth.Principal) {
				if len(p.Audience) != 2 {
					t.Errorf("Audience = %v, want both entries preserved", p.Audience)
				}
			},
		},
		{
			name: "expired token is rejected",
			token: func(t *testing.T) string {
				o := validOpts()
				o.issuedAt = fixedNow.Add(-2 * time.Hour)
				o.expiresAt = fixedNow.Add(-1 * time.Hour)
				return mint(t, keys, o)
			},
			wantErr: auth.ErrTokenExpired,
		},
		{
			name: "token whose nbf is in the future is rejected",
			token: func(t *testing.T) string {
				o := validOpts()
				o.notBefore = fixedNow.Add(30 * time.Minute)
				o.expiresAt = fixedNow.Add(60 * time.Minute)
				return mint(t, keys, o)
			},
			wantErr: auth.ErrTokenNotYetValid,
		},
		{
			name: "wrong issuer is rejected",
			token: func(t *testing.T) string {
				o := validOpts()
				o.issuer = "https://attacker.example.com/"
				return mint(t, keys, o)
			},
			wantErr: auth.ErrTokenIssuer,
		},
		{
			name: "wrong audience is rejected",
			token: func(t *testing.T) string {
				o := validOpts()
				o.audience = []string{"https://some-other-api.example.com"}
				return mint(t, keys, o)
			},
			wantErr: auth.ErrTokenAudience,
		},
		{
			// Rejected as a *missing* claim rather than a mismatched one:
			// both are rejections and both become Unauthenticated, but the
			// distinction is what an operator reads (see ErrTokenAudience's
			// doc comment).
			name: "absent audience is rejected",
			token: func(t *testing.T) string {
				o := validOpts()
				o.audience = nil
				return mint(t, keys, o)
			},
			wantErr: auth.ErrTokenClaimMissing,
		},
		{
			name: "signature from an unrelated keypair is rejected",
			token: func(t *testing.T) string {
				o := validOpts()
				o.signWith = keys.other
				return mint(t, keys, o)
			},
			wantErr: auth.ErrTokenSignature,
		},
		{
			name:    "malformed token is rejected",
			token:   func(t *testing.T) string { return "this.is.not-a-jwt" },
			wantErr: auth.ErrTokenMalformed,
		},
		{
			name:    "empty token is rejected",
			token:   func(t *testing.T) string { return "" },
			wantErr: auth.ErrTokenMalformed,
		},
		{
			name: "alg=none is rejected",
			token: func(t *testing.T) string {
				o := validOpts()
				o.method = jwt.SigningMethodNone
				return mint(t, keys, o)
			},
			wantErr: auth.ErrTokenSignature,
		},
		{
			name: "HS256 signed with the RSA public key is rejected (algorithm confusion)",
			token: func(t *testing.T) string {
				o := validOpts()
				o.method = jwt.SigningMethodHS256
				// The classic attack: the "secret" is the verifier's own
				// public key, which an attacker has by definition because a
				// JWKS publishes it.
				pub, err := publicKeyPEM(&keys.private.PublicKey)
				if err != nil {
					t.Fatalf("encode public key: %v", err)
				}
				o.hmacKey = pub
				return mint(t, keys, o)
			},
			wantErr: auth.ErrTokenSignature,
		},
		{
			name: "unknown kid is rejected",
			token: func(t *testing.T) string {
				o := validOpts()
				o.keyID = "some-key-we-have-never-seen"
				return mint(t, keys, o)
			},
			wantErr: auth.ErrKeyUnavailable,
		},
		{
			name: "missing kid is rejected",
			token: func(t *testing.T) string {
				o := validOpts()
				o.keyID = ""
				return mint(t, keys, o)
			},
			wantErr: auth.ErrKeyUnavailable,
		},
		{
			name: "token without exp is rejected",
			token: func(t *testing.T) string {
				o := validOpts()
				o.omitExp = true
				return mint(t, keys, o)
			},
			wantErr: auth.ErrTokenClaimMissing,
		},
		{
			name: "token without sub is rejected",
			token: func(t *testing.T) string {
				o := validOpts()
				o.subject = ""
				return mint(t, keys, o)
			},
			wantErr: auth.ErrTokenClaimMissing,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			v := newVerifier(t, keys)
			got, err := v.Verify(context.Background(), tt.token(t))

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Verify() error = %v, want errors.Is(..., %v)", err, tt.wantErr)
				}
				// Every rejection must yield the zero Principal. A partially
				// populated principal alongside an error is how "we rejected
				// this token" turns into "we authenticated this token" one
				// careless caller later.
				if !reflect.DeepEqual(got, auth.Principal{}) {
					t.Errorf("Verify() principal = %+v on error, want zero value", got)
				}
				return
			}

			if err != nil {
				t.Fatalf("Verify() unexpected error: %v", err)
			}
			if tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}

// TestVerifierLeewayToleratesClockSkew proves the leeway knob is wired to the
// clock checks rather than merely stored: the same token is rejected with no
// leeway and accepted with enough of it.
func TestVerifierLeewayToleratesClockSkew(t *testing.T) {
	t.Parallel()

	keys := newTestKeys(t)
	o := validOpts()
	o.expiresAt = fixedNow.Add(-30 * time.Second) // expired half a minute ago
	token := mint(t, keys, o)

	strict, err := rs256.NewVerifier(rs256.Config{
		Issuer:   testIssuer,
		Audience: testAudience,
		Keys:     keys.source,
		Now:      func() time.Time { return fixedNow },
	})
	if err != nil {
		t.Fatalf("NewVerifier(strict): %v", err)
	}
	if _, err := strict.Verify(context.Background(), token); !errors.Is(err, auth.ErrTokenExpired) {
		t.Fatalf("strict verifier error = %v, want ErrTokenExpired", err)
	}

	lenient, err := rs256.NewVerifier(rs256.Config{
		Issuer:   testIssuer,
		Audience: testAudience,
		Keys:     keys.source,
		Leeway:   2 * time.Minute,
		Now:      func() time.Time { return fixedNow },
	})
	if err != nil {
		t.Fatalf("NewVerifier(lenient): %v", err)
	}
	if _, err := lenient.Verify(context.Background(), token); err != nil {
		t.Fatalf("lenient verifier error = %v, want nil", err)
	}
}

// TestNewVerifierRejectsIncompleteConfig is the guard against the failure mode
// the ticket names directly: a verifier that looks like a check but isn't. An
// empty Issuer or Audience would make jwt's iss/aud validation vacuous, so the
// constructor refuses rather than silently degrading to signature-only.
func TestNewVerifierRejectsIncompleteConfig(t *testing.T) {
	t.Parallel()

	keys := newTestKeys(t)

	tests := []struct {
		name string
		cfg  rs256.Config
	}{
		{
			name: "no issuer",
			cfg:  rs256.Config{Audience: testAudience, Keys: keys.source},
		},
		{
			name: "no audience",
			cfg:  rs256.Config{Issuer: testIssuer, Keys: keys.source},
		},
		{
			name: "no key source",
			cfg:  rs256.Config{Issuer: testIssuer, Audience: testAudience},
		},
		{
			name: "negative leeway",
			cfg:  rs256.Config{Issuer: testIssuer, Audience: testAudience, Keys: keys.source, Leeway: -time.Second},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			v, err := rs256.NewVerifier(tt.cfg)
			if err == nil {
				t.Fatalf("NewVerifier(%+v) = %v, nil; want an error", tt.cfg, v)
			}
			if v != nil {
				t.Errorf("NewVerifier returned a non-nil verifier alongside an error: %v", v)
			}
		})
	}
}

// TestVerifierSurfacesKeySourceFailure proves an infrastructure failure in the
// key source stays distinguishable from a caller presenting a bad token. The
// two map to different gRPC codes once enforcement is on (ADR-0013): a JWKS we
// cannot reach is our outage, not the caller's forgery.
func TestVerifierSurfacesKeySourceFailure(t *testing.T) {
	t.Parallel()

	keys := newTestKeys(t)
	boom := errors.New("jwks endpoint unreachable")

	v, err := rs256.NewVerifier(rs256.Config{
		Issuer:   testIssuer,
		Audience: testAudience,
		Keys:     keySourceFunc(func(context.Context, string) (*rsa.PublicKey, error) { return nil, boom }),
		Now:      func() time.Time { return fixedNow },
	})
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}

	_, err = v.Verify(context.Background(), mint(t, keys, validOpts()))
	if !errors.Is(err, auth.ErrKeyUnavailable) {
		t.Fatalf("Verify() error = %v, want errors.Is(..., ErrKeyUnavailable)", err)
	}
	if !errors.Is(err, boom) {
		t.Errorf("Verify() error = %v, want it to wrap the key source's own error for operator diagnosis", err)
	}
	if auth.IsTokenRejection(err) {
		t.Error("IsTokenRejection(key-source failure) = true, want false: an unreachable JWKS is our failure, not the caller's")
	}
}

type keySourceFunc func(ctx context.Context, keyID string) (*rsa.PublicKey, error)

func (f keySourceFunc) PublicKey(ctx context.Context, keyID string) (*rsa.PublicKey, error) {
	return f(ctx, keyID)
}
