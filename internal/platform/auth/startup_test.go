package auth_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/nhuthuynh/white-label/internal/platform/auth"
)

// T13.5 / #136 — a process that enforces authentication on at least one RPC
// and holds no TokenVerifier is claiming a guarantee it cannot perform. Before
// this ticket cmd/server logged a warning and started anyway; these tests pin
// the rule that replaces that warning.
//
// The check lives in this package rather than in cmd/server's run() so that it
// is testable at all: run() needs a database, a listener and six handler
// constructions before it reaches the auth wiring, so a rule expressed only
// there is a rule no test can reach. cmd/server calls it in one line — see
// TestEnsureVerifierConfigured_IsWiredIntoServerStartup's comment.

// nilVerifier is a typed nil implementing TokenVerifier. Assigned to a
// TokenVerifier variable it produces a *non-nil interface holding a nil
// pointer*, which `verifier == nil` does not catch — the classic Go footgun,
// and one a future `tokenVerifierFromEnv` returning `(*rs256.Verifier)(nil)`
// on some error path would walk straight into.
type nilVerifier struct{}

func (*nilVerifier) Verify(_ context.Context, _ string) (auth.Principal, error) {
	return auth.Principal{}, auth.ErrNoToken
}

func TestEnsureVerifierConfigured(t *testing.T) {
	t.Parallel()

	enforced := auth.NewMethodSet([]string{authedMethod})
	nothingEnforced := auth.NewMethodSet()

	tests := []struct {
		name     string
		verifier auth.TokenVerifier
		set      auth.MethodSet
		wantErr  bool
	}{
		{
			// The observe-only deployment T12.2 shipped. Nothing is enforced,
			// so "no verifier" claims nothing and breaks nothing. Keeping this
			// legal is what makes the rule about *claimed enforcement* rather
			// than about auth being configured everywhere.
			name:     "no verifier and nothing enforced is a legal deployment",
			verifier: nil,
			set:      nothingEnforced,
			wantErr:  false,
		},
		{
			// #136 itself.
			name:     "no verifier while methods are enforced is fatal",
			verifier: nil,
			set:      enforced,
			wantErr:  true,
		},
		{
			name:     "typed-nil verifier while methods are enforced is fatal",
			verifier: (*nilVerifier)(nil),
			set:      enforced,
			wantErr:  true,
		},
		{
			name:     "configured verifier with enforced methods is the intended deployment",
			verifier: &stubVerifier{token: "t"},
			set:      enforced,
			wantErr:  false,
		},
		{
			name:     "configured verifier with nothing enforced is legal",
			verifier: &stubVerifier{token: "t"},
			set:      nothingEnforced,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := auth.EnsureVerifierConfigured(tt.verifier, tt.set)

			if tt.wantErr && err == nil {
				t.Fatal("EnsureVerifierConfigured returned nil; want a fatal configuration error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("EnsureVerifierConfigured returned %v; want nil", err)
			}
		})
	}
}

// TestEnsureVerifierConfiguredErrorIsIdentifiableAndCounted pins the two
// properties an operator-facing startup failure needs beyond "non-nil".
//
// It is matchable with errors.Is, so cmd/server can distinguish "nobody
// configured a verifier" from "the JWKS file was unreadable" and attach the
// right remedy to each. The environment-variable names are deliberately *not*
// asserted here: AUTH_ISSUER and friends are cmd/server's vocabulary, and this
// package would be reaching across a layer to know them (CLAUDE.md rule 3).
// cmd/server/main_test.go asserts the remedy text where it belongs.
//
// And it reports how many RPCs are affected, which is the operator's
// "how much is broken" signal — two enforced methods with no verifier is a
// different incident from ninety.
func TestEnsureVerifierConfiguredErrorIsIdentifiableAndCounted(t *testing.T) {
	t.Parallel()

	err := auth.EnsureVerifierConfigured(nil, auth.NewMethodSet([]string{authedMethod, publicMethod}))
	if err == nil {
		t.Fatal("EnsureVerifierConfigured returned nil; want a fatal configuration error")
	}
	if !errors.Is(err, auth.ErrVerifierRequired) {
		t.Fatalf("errors.Is(err, ErrVerifierRequired) = false for %v; the sentinel is how callers identify this", err)
	}
	if !strings.Contains(err.Error(), "2") {
		t.Errorf("error %q does not report how many methods are enforced", err.Error())
	}
}
