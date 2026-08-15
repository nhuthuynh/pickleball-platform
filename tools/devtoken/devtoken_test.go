package devtoken_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nhuthuynh/white-label/internal/platform/auth"
	"github.com/nhuthuynh/white-label/internal/platform/auth/rs256"
	"github.com/nhuthuynh/white-label/tools/devtoken"
)

// repoFile reads a path relative to the repository root.
//
// The fixture's whole point is that the *committed* files work, so these tests
// read them from disk rather than generating a keypair in memory: an in-memory
// round trip would pass just as happily against a fixture that was never
// committed, or was committed half-rotated.
func repoFile(t *testing.T, path string) []byte {
	t.Helper()
	full := filepath.Join("..", "..", filepath.FromSlash(path))
	data, err := os.ReadFile(full)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return data
}

func loadFixtureKey(t *testing.T) *devtoken.PrivateKey {
	t.Helper()
	key, err := devtoken.LoadPrivateJWK(repoFile(t, devtoken.PrivateJWKFile))
	if err != nil {
		t.Fatalf("LoadPrivateJWK(%s): %v", devtoken.PrivateJWKFile, err)
	}
	return key
}

// fixtureVerifier builds the verifier cmd/server builds when pointed at the
// committed JWKS: same parser, same key source, same code path. Nothing in
// this test constructs a key by hand.
func fixtureVerifier(t *testing.T, issuer, audience string) *rs256.Verifier {
	t.Helper()
	keys, err := rs256.NewStaticKeysFromJWKS(repoFile(t, devtoken.JWKSFile))
	if err != nil {
		t.Fatalf("NewStaticKeysFromJWKS(%s): %v", devtoken.JWKSFile, err)
	}
	verifier, err := rs256.NewVerifier(rs256.Config{Issuer: issuer, Audience: audience, Keys: keys})
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}
	return verifier
}

// TestMintedTokenVerifiesAgainstTheCommittedJWKS is the ticket in one test:
// the committed private half mints a token that the real RS256 verifier,
// loaded from the committed public half, accepts — and the principal it
// resolves is the one a handler will see.
func TestMintedTokenVerifiesAgainstTheCommittedJWKS(t *testing.T) {
	token, err := loadFixtureKey(t).Mint(devtoken.MintParams{
		Issuer:   devtoken.Issuer,
		Audience: devtoken.Audience,
		Subject:  devtoken.Subject,
		Scope:    "bookings:read bookings:write",
	})
	if err != nil {
		t.Fatalf("Mint: %v", err)
	}

	principal, err := fixtureVerifier(t, devtoken.Issuer, devtoken.Audience).Verify(context.Background(), token)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}

	if principal.Subject != devtoken.Subject {
		t.Errorf("Subject = %q, want %q", principal.Subject, devtoken.Subject)
	}
	if principal.Issuer != devtoken.Issuer {
		t.Errorf("Issuer = %q, want %q", principal.Issuer, devtoken.Issuer)
	}
	if len(principal.Audience) != 1 || principal.Audience[0] != devtoken.Audience {
		t.Errorf("Audience = %v, want [%q]", principal.Audience, devtoken.Audience)
	}
	if got, want := strings.Join(principal.Scopes, " "), "bookings:read bookings:write"; got != want {
		t.Errorf("Scopes = %q, want %q", got, want)
	}
	if principal.ExpiresAt.Sub(principal.IssuedAt) != devtoken.DefaultTTL {
		t.Errorf("token lifetime = %s, want DefaultTTL %s",
			principal.ExpiresAt.Sub(principal.IssuedAt), devtoken.DefaultTTL)
	}
}

// TestFixtureSubjectIsProviderShaped pins the `sub` shape. A bare "user-1"
// would let a handler that mishandles a real provider's `<connection>|<id>`
// subject pass locally and fail in a deployment — which is exactly the class
// of bug a local fixture exists to catch early.
func TestFixtureSubjectIsProviderShaped(t *testing.T) {
	before, after, found := strings.Cut(devtoken.Subject, "|")
	if !found || before == "" || after == "" {
		t.Fatalf("Subject = %q, want a provider-shaped <connection>|<id> subject", devtoken.Subject)
	}
}

// TestCommittedJWKSIsDerivedFromTheCommittedPrivateKey is the anti-drift
// check. Two files rotated out of step would leave a JWKS nobody can mint
// against — the server would start and every token would be rejected, which
// is the confusing half-broken state #160 exists to prevent.
func TestCommittedJWKSIsDerivedFromTheCommittedPrivateKey(t *testing.T) {
	derived, err := loadFixtureKey(t).MarshalJWKS()
	if err != nil {
		t.Fatalf("MarshalJWKS: %v", err)
	}
	if committed := repoFile(t, devtoken.JWKSFile); !bytes.Equal(derived, committed) {
		t.Errorf("%s is not the public half of %s.\n"+
			"Regenerate both with: go run ./cmd/devtoken -regenerate\ngot:\n%s\nwant:\n%s",
			devtoken.JWKSFile, devtoken.PrivateJWKFile, committed, derived)
	}
}

// TestCommittedFixtureIsMarkedDevOnly checks the property that keeps this
// safe to commit: every file says so, loudly, in itself.
func TestCommittedFixtureIsMarkedDevOnly(t *testing.T) {
	for _, path := range []string{devtoken.PrivateJWKFile, devtoken.JWKSFile} {
		body := string(repoFile(t, path))
		for _, marker := range []string{"DEV-ONLY", "NOT A SECRET", "NEVER use it"} {
			if !strings.Contains(body, marker) {
				t.Errorf("%s does not contain the marker %q — a reader must not be able to "+
					"mistake this file for a real secret", path, marker)
			}
		}
		if !strings.Contains(body, devtoken.KeyID) {
			t.Errorf("%s does not carry kid %q", path, devtoken.KeyID)
		}
	}
}

// TestPrivateFixtureIsNotAPEM guards the deliberate encoding choice. A PEM
// private key here would trip GitHub's push protection and every secret
// scanner pointed at this repository, producing a permanent false positive on
// a file that is public on purpose. See the privateJWK doc comment.
func TestPrivateFixtureIsNotAPEM(t *testing.T) {
	if body := string(repoFile(t, devtoken.PrivateJWKFile)); strings.Contains(body, "-----BEGIN") {
		t.Error("the private fixture is PEM-encoded; it must be a JWK so a deliberately-public " +
			"fixture does not trip secret scanners (see dev/auth/README.md)")
	}
}

// TestDockerComposeWiresTheFixture is what makes `make up` staying fixed a
// tested property rather than a claim in a PR body. The compose file and this
// package have to agree on all three AUTH_* values and on the mounted JWKS
// path; nothing else in the build would notice if they stopped agreeing —
// `make up` would simply fail again, and only for whoever ran it next.
func TestDockerComposeWiresTheFixture(t *testing.T) {
	compose := string(repoFile(t, "docker-compose.yml"))

	for _, want := range []string{
		"AUTH_ISSUER: " + devtoken.Issuer,
		"AUTH_AUDIENCE: " + devtoken.Audience,
	} {
		if !strings.Contains(compose, want) {
			t.Errorf("docker-compose.yml does not set %q — `make up` will fail startup again", want)
		}
	}

	// The JWKS must be bind-mounted from the fixture directory and named by
	// AUTH_JWKS_FILE at its in-container path. Checking both halves matters:
	// either one alone starts nothing.
	const mount = "./dev/auth:/etc/pickleball/dev-auth:ro"
	const jwksEnv = "AUTH_JWKS_FILE: /etc/pickleball/dev-auth/"
	if !strings.Contains(compose, mount) {
		t.Errorf("docker-compose.yml does not bind-mount the fixture (%q)", mount)
	}
	if !strings.Contains(compose, jwksEnv+filepath.Base(devtoken.JWKSFile)) {
		t.Errorf("docker-compose.yml's AUTH_JWKS_FILE does not point at %s inside the mount",
			filepath.Base(devtoken.JWKSFile))
	}
}

// TestFixtureTokenIsRejectedElsewhere is the negative that matters. The
// committed key is public, so the property worth proving is that a token it
// signs opens nothing beyond a server explicitly configured with this fixture.
func TestFixtureTokenIsRejectedElsewhere(t *testing.T) {
	key := loadFixtureKey(t)

	// A separately-generated key stands in for "a real provider's key set" —
	// the deployed path, where AUTH_JWKS_FILE names someone else's JWKS.
	otherKey, err := devtoken.GenerateKeyPair(2048)
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}
	otherJWKS, err := otherKey.MarshalJWKS()
	if err != nil {
		t.Fatalf("MarshalJWKS: %v", err)
	}
	otherKeys, err := rs256.NewStaticKeysFromJWKS(otherJWKS)
	if err != nil {
		t.Fatalf("NewStaticKeysFromJWKS: %v", err)
	}

	tests := []struct {
		name     string
		verifier func(t *testing.T) *rs256.Verifier
		mint     devtoken.MintParams
		wantErr  error
	}{
		{
			// The stand-in key deliberately carries the SAME kid as the
			// fixture, which is the worst case: it removes key *selection*
			// from the argument and leaves the signature check alone to
			// reject the token. A real provider publishing different kids is
			// the easier case (an unknown kid never reaches a signature
			// check at all), so proving the hard one covers both.
			name: "a server pointed at another key set rejects it, even under the same kid",
			verifier: func(t *testing.T) *rs256.Verifier {
				t.Helper()
				v, err := rs256.NewVerifier(rs256.Config{
					Issuer: devtoken.Issuer, Audience: devtoken.Audience, Keys: otherKeys,
				})
				if err != nil {
					t.Fatalf("NewVerifier: %v", err)
				}
				return v
			},
			mint:    devtoken.MintParams{Issuer: devtoken.Issuer, Audience: devtoken.Audience, Subject: devtoken.Subject},
			wantErr: auth.ErrTokenSignature,
		},
		{
			name:     "a token minted for another issuer is rejected",
			verifier: func(t *testing.T) *rs256.Verifier { return fixtureVerifier(t, devtoken.Issuer, devtoken.Audience) },
			mint: devtoken.MintParams{
				Issuer: "https://real-idp.example.com/", Audience: devtoken.Audience, Subject: devtoken.Subject,
			},
			wantErr: auth.ErrTokenIssuer,
		},
		{
			name:     "a token minted for another audience is rejected",
			verifier: func(t *testing.T) *rs256.Verifier { return fixtureVerifier(t, devtoken.Issuer, devtoken.Audience) },
			mint: devtoken.MintParams{
				Issuer: devtoken.Issuer, Audience: "https://api.example.com/prod", Subject: devtoken.Subject,
			},
			wantErr: auth.ErrTokenAudience,
		},
		{
			name:     "an expired token is rejected",
			verifier: func(t *testing.T) *rs256.Verifier { return fixtureVerifier(t, devtoken.Issuer, devtoken.Audience) },
			mint: devtoken.MintParams{
				Issuer: devtoken.Issuer, Audience: devtoken.Audience, Subject: devtoken.Subject,
				TTL: time.Minute,
				Now: func() time.Time { return time.Now().Add(-24 * time.Hour) },
			},
			wantErr: auth.ErrTokenExpired,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			token, err := key.Mint(tc.mint)
			if err != nil {
				t.Fatalf("Mint: %v", err)
			}
			_, err = tc.verifier(t).Verify(context.Background(), token)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Verify error = %v, want one wrapping %v", err, tc.wantErr)
			}
		})
	}
}

// TestLoadPrivateJWKRefusals covers the guard that keeps cmd/devtoken a
// fixture tool: it signs with dev fixtures, and says no to anything else.
func TestLoadPrivateJWKRefusals(t *testing.T) {
	valid := repoFile(t, devtoken.PrivateJWKFile)

	tests := []struct {
		name        string
		document    []byte
		wantMessage string
	}{
		{
			name:        "not json",
			document:    []byte("{"),
			wantMessage: "parsing private jwk",
		},
		{
			name:        "a real provider's key id",
			document:    bytes.ReplaceAll(valid, []byte(devtoken.KeyID), []byte("prod-signing-key-2026")),
			wantMessage: "mints DEV FIXTURE tokens only",
		},
		{
			name:        "wrong key type",
			document:    bytes.Replace(valid, []byte(`"kty": "RSA"`), []byte(`"kty": "EC"`), 1),
			wantMessage: "unsupported kty",
		},
		{
			name:        "missing private exponent",
			document:    mustReplaceMember(t, valid, "d", ""),
			wantMessage: `missing member "d"`,
		},
		{
			name:        "modulus below the verifier's floor",
			document:    mustReplaceMember(t, valid, "n", "AQAB"),
			wantMessage: "the verifier requires at least 2048",
		},
		{
			name:        "primes that do not match the modulus",
			document:    mustReplaceMember(t, valid, "p", "AQAB"),
			wantMessage: "not a consistent RSA key",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := devtoken.LoadPrivateJWK(tc.document)
			if err == nil {
				t.Fatal("LoadPrivateJWK succeeded, want an error")
			}
			if !strings.Contains(err.Error(), tc.wantMessage) {
				t.Fatalf("error = %q, want it to contain %q", err, tc.wantMessage)
			}
		})
	}
}

// mustReplaceMember rewrites one JWK member's value, so a table case can name
// the member it is corrupting rather than a byte pattern.
func mustReplaceMember(t *testing.T, document []byte, member, value string) []byte {
	t.Helper()
	var fields map[string]any
	if err := json.Unmarshal(document, &fields); err != nil {
		t.Fatalf("decoding fixture: %v", err)
	}
	if _, ok := fields[member]; !ok {
		t.Fatalf("fixture has no member %q", member)
	}
	if value == "" {
		delete(fields, member)
	} else {
		fields[member] = value
	}
	out, err := json.Marshal(fields)
	if err != nil {
		t.Fatalf("encoding fixture: %v", err)
	}
	return out
}

// TestMintRejectsIncompleteParams: a token missing any of these cannot verify,
// so refusing to mint it turns a confusing 401 into an immediate error.
func TestMintRejectsIncompleteParams(t *testing.T) {
	key := loadFixtureKey(t)

	tests := []struct {
		name        string
		params      devtoken.MintParams
		wantMessage string
	}{
		{
			name:        "no issuer",
			params:      devtoken.MintParams{Audience: devtoken.Audience, Subject: devtoken.Subject},
			wantMessage: "Issuer is required",
		},
		{
			name:        "no audience",
			params:      devtoken.MintParams{Issuer: devtoken.Issuer, Subject: devtoken.Subject},
			wantMessage: "Audience is required",
		},
		{
			name:        "no subject",
			params:      devtoken.MintParams{Issuer: devtoken.Issuer, Audience: devtoken.Audience},
			wantMessage: "Subject is required",
		},
		{
			name: "negative ttl",
			params: devtoken.MintParams{
				Issuer: devtoken.Issuer, Audience: devtoken.Audience, Subject: devtoken.Subject, TTL: -time.Hour,
			},
			wantMessage: "TTL must not be negative",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := key.Mint(tc.params)
			if err == nil {
				t.Fatal("Mint succeeded, want an error")
			}
			if !strings.Contains(err.Error(), tc.wantMessage) {
				t.Fatalf("error = %q, want it to contain %q", err, tc.wantMessage)
			}
		})
	}
}

// TestGenerateKeyPairRefusesWeakKeys keeps a fixture rotation from producing
// key material the verifier's RFC 8725 floor would reject at parse time —
// which would look like "the fixture stopped working" rather than "the key is
// too weak".
func TestGenerateKeyPairRefusesWeakKeys(t *testing.T) {
	if _, err := devtoken.GenerateKeyPair(1024); err == nil {
		t.Fatal("GenerateKeyPair(1024) succeeded, want a refusal")
	}
}
