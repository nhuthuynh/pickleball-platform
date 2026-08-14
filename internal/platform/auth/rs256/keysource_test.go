package rs256_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"math/big"
	"testing"

	"github.com/nhuthuynh/white-label/internal/platform/auth"
	"github.com/nhuthuynh/white-label/internal/platform/auth/rs256"
)

// publicKeyPEM renders a public key the way an attacker would obtain it from a
// published JWKS — used by the algorithm-confusion case in verifier_test.go.
func publicKeyPEM(pub *rsa.PublicKey) ([]byte, error) {
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}), nil
}

// jwksFor renders a real RFC 7517 JWKS document for the given keys, which is
// what a production IdP's /.well-known/jwks.json actually serves.
func jwksFor(t *testing.T, keys map[string]*rsa.PublicKey) []byte {
	t.Helper()

	type jwk struct {
		Kty string `json:"kty"`
		Use string `json:"use"`
		Alg string `json:"alg"`
		Kid string `json:"kid"`
		N   string `json:"n"`
		E   string `json:"e"`
	}
	doc := struct {
		Keys []jwk `json:"keys"`
	}{}

	for kid, pub := range keys {
		e := big.NewInt(int64(pub.E)).Bytes()
		doc.Keys = append(doc.Keys, jwk{
			Kty: "RSA",
			Use: "sig",
			Alg: "RS256",
			Kid: kid,
			N:   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
			E:   base64.RawURLEncoding.EncodeToString(e),
		})
	}

	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal jwks: %v", err)
	}
	return raw
}

func TestNewStaticKeysFromJWKS(t *testing.T) {
	t.Parallel()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	t.Run("round-trips a real JWKS document", func(t *testing.T) {
		t.Parallel()

		src, err := rs256.NewStaticKeysFromJWKS(jwksFor(t, map[string]*rsa.PublicKey{"kid-a": &priv.PublicKey}))
		if err != nil {
			t.Fatalf("NewStaticKeysFromJWKS: %v", err)
		}

		got, err := src.PublicKey(context.Background(), "kid-a")
		if err != nil {
			t.Fatalf("PublicKey: %v", err)
		}
		if got.N.Cmp(priv.N) != 0 || got.E != priv.E {
			t.Error("PublicKey returned a key that is not the one the JWKS described")
		}
	})

	t.Run("unknown kid is a key-unavailable error", func(t *testing.T) {
		t.Parallel()

		src, err := rs256.NewStaticKeysFromJWKS(jwksFor(t, map[string]*rsa.PublicKey{"kid-a": &priv.PublicKey}))
		if err != nil {
			t.Fatalf("NewStaticKeysFromJWKS: %v", err)
		}
		if _, err := src.PublicKey(context.Background(), "kid-b"); !errors.Is(err, auth.ErrKeyUnavailable) {
			t.Fatalf("PublicKey(unknown kid) error = %v, want ErrKeyUnavailable", err)
		}
	})

	// A JWKS is untrusted input parsed at startup (research-security-compliance
	// §1, API10). Every malformed shape must be a construction error, never a
	// silently-empty key set — an empty key set would make every token fail
	// verification, which in observe-only mode is invisible.
	t.Run("rejects malformed documents", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			doc  string
		}{
			{"not json", `{`},
			{"no keys array", `{"keys":[]}`},
			{"missing kid", `{"keys":[{"kty":"RSA","n":"AQAB","e":"AQAB"}]}`},
			{"non-rsa key type", `{"keys":[{"kty":"EC","kid":"k","n":"AQAB","e":"AQAB"}]}`},
			{"modulus not base64url", `{"keys":[{"kty":"RSA","kid":"k","n":"!!!","e":"AQAB"}]}`},
			{"empty modulus", `{"keys":[{"kty":"RSA","kid":"k","n":"","e":"AQAB"}]}`},
			{"empty exponent", `{"keys":[{"kty":"RSA","kid":"k","n":"AQAB","e":""}]}`},
			{"duplicate kid", `{"keys":[{"kty":"RSA","kid":"k","n":"AQAB","e":"AQAB"},{"kty":"RSA","kid":"k","n":"AQAB","e":"AQAB"}]}`},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				src, err := rs256.NewStaticKeysFromJWKS([]byte(tt.doc))
				if err == nil {
					t.Fatalf("NewStaticKeysFromJWKS(%s) = %v, nil; want an error", tt.doc, src)
				}
				if src != nil {
					t.Error("NewStaticKeysFromJWKS returned a key source alongside an error")
				}
			})
		}
	})

	// An RSA key too small for RS256 is a real-world downgrade risk, and a
	// JWKS is attacker-influenced input in the threat model where the IdP's
	// discovery response is spoofed. Rejecting at parse time means no token
	// signed by such a key ever reaches signature verification.
	t.Run("rejects an undersized modulus", func(t *testing.T) {
		t.Parallel()

		weak, err := rsa.GenerateKey(rand.Reader, 1024)
		if err != nil {
			t.Fatalf("generate weak key: %v", err)
		}
		if _, err := rs256.NewStaticKeysFromJWKS(jwksFor(t, map[string]*rsa.PublicKey{"weak": &weak.PublicKey})); err == nil {
			t.Fatal("NewStaticKeysFromJWKS accepted a 1024-bit modulus, want rejection")
		}
	})
}

func TestNewStaticKeys(t *testing.T) {
	t.Parallel()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	src := rs256.NewStaticKeys(map[string]*rsa.PublicKey{"kid-a": &priv.PublicKey})
	if _, err := src.PublicKey(context.Background(), "kid-a"); err != nil {
		t.Fatalf("PublicKey: %v", err)
	}
	if _, err := src.PublicKey(context.Background(), ""); !errors.Is(err, auth.ErrKeyUnavailable) {
		t.Fatalf("PublicKey(\"\") error = %v, want ErrKeyUnavailable", err)
	}
}
