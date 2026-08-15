package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/nhuthuynh/white-label/internal/platform/auth"
)

// T13.5 / #136 — these tests exist because internal/platform/auth's own tests
// cannot reach the question that actually matters.
//
// EnsureVerifierConfigured being correct is worth nothing if nobody calls it,
// and it is worth nothing if the set it is called with turns out to be empty
// in this binary. Both of those are properties of *this* package's wiring.
// run() cannot be called from a test — it wants a database, a listener and six
// handler constructions before it reaches auth — so the composition and the
// check were pulled out into authenticationPolicy, which is exactly as much of
// run() as can be exercised without infrastructure and exactly the part that
// decides whether the process may start.

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// stubVerifier is a TokenVerifier that never has to work: authenticationPolicy
// only asks whether one is present.
type stubVerifier struct{}

func (stubVerifier) Verify(context.Context, string) (auth.Principal, error) {
	return auth.Principal{}, auth.ErrNoToken
}

// TestAuthenticationPolicyRefusesToStartWithoutAVerifier is the ticket's
// headline behaviour, asserted against the real composed method set rather
// than a fixture. Before T13.5 this call logged a warning and returned a
// perfectly usable MethodSet.
func TestAuthenticationPolicyRefusesToStartWithoutAVerifier(t *testing.T) {
	_, err := authenticationPolicy(nil, discardLogger())

	if err == nil {
		t.Fatal("authenticationPolicy(nil) returned no error; a server that enforces authentication with no verifier must not start")
	}
	if !errors.Is(err, auth.ErrVerifierRequired) {
		t.Fatalf("errors.Is(err, auth.ErrVerifierRequired) = false for %v", err)
	}
	// The remedy belongs in this binary's vocabulary — see the doc comment on
	// authenticationPolicy. An operator hitting a hard startup failure at
	// 03:00 should not have to read source to learn which variables to set.
	// AUTH_JWKS_URL joins the list in T15.7: an operator reading this error is
	// being told how to satisfy it, and "point it at your provider's JWKS
	// endpoint" is now one of the two honest answers.
	for _, want := range []string{"AUTH_ISSUER", "AUTH_AUDIENCE", "AUTH_JWKS_FILE", "AUTH_JWKS_URL"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("startup error %q does not name %s", err.Error(), want)
		}
	}
}

// TestAuthenticationPolicyStartsWithAVerifier is the other half: the check
// refuses a misconfiguration, not every configuration. A rule that rejected
// the intended deployment too would be caught here rather than in production.
func TestAuthenticationPolicyStartsWithAVerifier(t *testing.T) {
	set, err := authenticationPolicy(stubVerifier{}, discardLogger())
	if err != nil {
		t.Fatalf("authenticationPolicy with a configured verifier returned %v, want nil", err)
	}
	if set.Len() == 0 {
		t.Fatal("composed MethodSet is empty; the policy this server enforces resolved to nothing")
	}
}

// TestComposedPolicyIsNonEmpty is what stops the #136 fix from being vacuous.
//
// EnsureVerifierConfigured is a no-op when nothing is enforced, so if every
// context's AuthenticatedMethods() ever returned an empty list — a refactor
// that dropped a line from the composition, a context whose list moved and was
// not re-wired — the startup check would silently stop firing and this
// server would go back to booting with no verifier and no complaint. That
// failure is invisible to the test above, which only proves the check runs.
// This one proves it has something to bite on, and names the count so the
// diff is legible when a context is added or removed.
func TestComposedPolicyIsNonEmpty(t *testing.T) {
	set, err := authenticationPolicy(stubVerifier{}, discardLogger())
	if err != nil {
		t.Fatalf("authenticationPolicy: %v", err)
	}
	if set.Len() < 6 {
		t.Errorf("composed policy covers %d methods; all six bounded contexts declare authenticated RPCs, so this should not be near zero", set.Len())
	}
	t.Logf("composed authentication policy covers %d RPCs", set.Len())
}

// T15.7 / #137 — the configuration matrix.
//
// tokenVerifierFromEnv is the only place that decides what this process trusts
// to verify tokens, and until now nothing tested it. It gained a second key
// source (AUTH_JWKS_URL, a live provider's JWKS endpoint) alongside
// AUTH_JWKS_FILE, which makes three questions worth pinning: that the
// file path T14.9 built still works exactly as it did, that the two sources
// are mutually exclusive rather than one silently winning, and that a
// misconfigured URL is a startup failure rather than a server that runs and
// denies every request.
//
// t.Setenv rules out t.Parallel here, which is the correct trade: these cases
// are about process-wide state.
func TestTokenVerifierFromEnv(t *testing.T) {
	// The committed local-dev fixture (T14.9, #160). Using the real file
	// rather than a temp one means this test fails if that path is ever
	// renamed or its shape drifts — `make up` depends on both.
	const devJWKS = "../../dev/auth/dev-only-insecure.jwks.json"

	// Nothing listens here, so the warm-up fetch fails immediately rather
	// than waiting out a timeout.
	const unreachableJWKS = "https://127.0.0.1:1/.well-known/jwks.json"

	tests := []struct {
		name         string
		issuer       string
		audience     string
		jwksFile     string
		jwksURL      string
		wantVerifier bool
		wantErr      bool
		// wantErrMentions are substrings an operator needs in order to fix the
		// misconfiguration without reading this source file.
		wantErrMentions []string
	}{
		{
			// A build of this server that enforces nothing may still run
			// without auth; authenticationPolicy, not this function, decides
			// whether that is acceptable.
			name: "nothing configured is not an error here",
		},
		{
			name:         "AUTH_JWKS_FILE still builds a verifier (T14.9 dev path)",
			issuer:       "https://dev-auth.pickleball.invalid/",
			audience:     "https://api.pickleball.invalid/dev",
			jwksFile:     devJWKS,
			wantVerifier: true,
		},
		{
			name:            "both sources set is a startup error, not a precedence rule",
			issuer:          "https://dev-auth.pickleball.invalid/",
			audience:        "https://api.pickleball.invalid/dev",
			jwksFile:        devJWKS,
			jwksURL:         "https://tenant.example.com/.well-known/jwks.json",
			wantErr:         true,
			wantErrMentions: []string{"AUTH_JWKS_FILE", "AUTH_JWKS_URL"},
		},
		{
			name:            "issuer and audience with no key source",
			issuer:          "https://dev-auth.pickleball.invalid/",
			audience:        "https://api.pickleball.invalid/dev",
			wantErr:         true,
			wantErrMentions: []string{"AUTH_JWKS_FILE", "AUTH_JWKS_URL"},
		},
		{
			name:            "a URL with no issuer is partial configuration",
			jwksURL:         "https://tenant.example.com/.well-known/jwks.json",
			wantErr:         true,
			wantErrMentions: []string{"AUTH_ISSUER"},
		},
		{
			name:            "a plaintext AUTH_JWKS_URL is refused",
			issuer:          "https://dev-auth.pickleball.invalid/",
			audience:        "https://api.pickleball.invalid/dev",
			jwksURL:         "http://tenant.example.com/.well-known/jwks.json",
			wantErr:         true,
			wantErrMentions: []string{"https"},
		},
		{
			// The warm-up. Without it this process would start, look healthy,
			// and reject every authenticated RPC until someone read the logs.
			name:            "an unreachable AUTH_JWKS_URL fails at startup",
			issuer:          "https://dev-auth.pickleball.invalid/",
			audience:        "https://api.pickleball.invalid/dev",
			jwksURL:         unreachableJWKS,
			wantErr:         true,
			wantErrMentions: []string{"AUTH_JWKS_URL"},
		},
		{
			name:            "a missing AUTH_JWKS_FILE fails at startup",
			issuer:          "https://dev-auth.pickleball.invalid/",
			audience:        "https://api.pickleball.invalid/dev",
			jwksFile:        "../../dev/auth/does-not-exist.json",
			wantErr:         true,
			wantErrMentions: []string{"AUTH_JWKS_FILE"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("AUTH_ISSUER", tc.issuer)
			t.Setenv("AUTH_AUDIENCE", tc.audience)
			t.Setenv("AUTH_JWKS_FILE", tc.jwksFile)
			t.Setenv("AUTH_JWKS_URL", tc.jwksURL)

			verifier, err := tokenVerifierFromEnv(discardLogger())

			if tc.wantErr {
				if err == nil {
					t.Fatal("tokenVerifierFromEnv returned no error; a misconfigured verifier must not reach a running server")
				}
				if verifier != nil {
					t.Error("tokenVerifierFromEnv returned a verifier alongside an error")
				}
				for _, want := range tc.wantErrMentions {
					if !strings.Contains(err.Error(), want) {
						t.Errorf("error %q does not name %s", err.Error(), want)
					}
				}
				return
			}

			if err != nil {
				t.Fatalf("tokenVerifierFromEnv: %v", err)
			}
			if got := verifier != nil; got != tc.wantVerifier {
				t.Errorf("verifier != nil = %t, want %t", got, tc.wantVerifier)
			}
		})
	}
}
