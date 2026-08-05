package sharetoken_test

import (
	"strings"
	"testing"

	"github.com/nhuthuynh/white-label/internal/competitions/adapter/sharetoken"
	"github.com/nhuthuynh/white-label/internal/competitions/port"
)

// Compile-time proof that Generator satisfies the port the app layer
// depends on — a mismatch should fail the build, not a test run.
var _ port.ShareTokenGenerator = sharetoken.Generator{}

// TestNewShareToken_IsURLSafe pins the encoding contract the port documents
// ("URL-safe"): the token goes into a shareable link, so any character that
// would need percent-encoding — or, worse, silently change meaning in a
// query string — is a bug. base64url's alphabet is [A-Za-z0-9-_]; padding
// ('=') is explicitly excluded because it is not URL-safe unescaped.
func TestNewShareToken_IsURLSafe(t *testing.T) {
	const urlSafeAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"

	g := sharetoken.Generator{}
	for i := 0; i < 100; i++ {
		token, err := g.NewShareToken()
		if err != nil {
			t.Fatalf("NewShareToken() unexpected error: %v", err)
		}
		if token == "" {
			t.Fatal("NewShareToken() returned an empty token")
		}
		if strings.Contains(token, "=") {
			t.Errorf("token %q contains base64 padding '=', which is not URL-safe unescaped", token)
		}
		for _, r := range token {
			if !strings.ContainsRune(urlSafeAlphabet, r) {
				t.Fatalf("token %q contains non-URL-safe character %q", token, r)
			}
		}
	}
}

// TestNewShareToken_IsUnguessablyLong guards the security property that
// actually matters. A share token is the ONLY thing standing between an
// unpublished Competition and anyone who tries to guess one, so a short or
// sequential token would make every Competition enumerable — precisely the
// risk port.ShareTokenGenerator's doc comment calls out.
//
// 32 random bytes is 256 bits of entropy; base64url-encoded without padding
// that is 43 characters. The assertion is a floor, not an exact match, so
// increasing entropy later doesn't fail this test — but decreasing it below
// a defensible threshold does.
func TestNewShareToken_IsUnguessablyLong(t *testing.T) {
	const minLength = 43 // 32 bytes of entropy, base64url, unpadded

	g := sharetoken.Generator{}
	token, err := g.NewShareToken()
	if err != nil {
		t.Fatalf("NewShareToken() unexpected error: %v", err)
	}
	if len(token) < minLength {
		t.Errorf("token %q is %d chars, want at least %d — a short token makes unpublished Competitions enumerable", token, len(token), minLength)
	}
}

// TestNewShareToken_IsDistinctAcrossCalls is the non-vacuous half of the
// entropy claim: a generator that returned the same long constant every time
// would pass both tests above. It would also collide on the share_token
// UNIQUE constraint the moment a second Competition was created, so this
// catches a real, immediate failure mode and not just a theoretical one.
func TestNewShareToken_IsDistinctAcrossCalls(t *testing.T) {
	const iterations = 1000

	g := sharetoken.Generator{}
	seen := make(map[string]struct{}, iterations)
	for i := 0; i < iterations; i++ {
		token, err := g.NewShareToken()
		if err != nil {
			t.Fatalf("NewShareToken() unexpected error on call %d: %v", i, err)
		}
		if _, dup := seen[token]; dup {
			t.Fatalf("NewShareToken() returned a duplicate token %q within %d calls — tokens must be unguessable AND unique (share_token is UNIQUE in the schema)", token, iterations)
		}
		seen[token] = struct{}{}
	}
}
