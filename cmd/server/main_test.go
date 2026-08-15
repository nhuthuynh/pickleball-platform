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
	for _, want := range []string{"AUTH_ISSUER", "AUTH_AUDIENCE", "AUTH_JWKS_FILE"} {
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
