package auth

import (
	"errors"
	"fmt"
	"reflect"
)

// ErrVerifierRequired reports that this process declares authentication on at
// least one RPC while holding no TokenVerifier to perform it.
//
// It is a sentinel so cmd/server can identify this specific misconfiguration —
// and add the environment-variable remedy, which is its vocabulary and not
// this package's — rather than matching on message text.
var ErrVerifierRequired = errors.New("auth: authentication is enforced but no token verifier is configured")

// EnsureVerifierConfigured reports whether this process may start.
//
// # The rule
//
// A process that lists RPCs in its MethodSet is telling its operators that
// those RPCs admit only verified callers. With no TokenVerifier that claim is
// unperformable: no token can ever resolve into a Principal, so every one of
// those RPCs rejects every caller forever. Startup is where that has to be
// caught, because every later moment reports it as something else — a fleet of
// Unauthenticated responses that look like a client-side credential problem.
//
// # Why this is not merely a warning (issue #136)
//
// Until enforcement landed, a nil verifier meant "resolve nothing", which was
// byte-for-byte the pre-auth behavior and therefore safe; cmd/server logged a
// warning and continued, and ADR-0013 recorded that this must invert once any
// RPC enforced. All six bounded contexts now declare an authenticated set, so
// the inversion is due. A warning is the wrong instrument for it: warnings are
// emitted once, into a log nobody reads at 03:00, by a process that then
// proceeds to serve traffic while advertising a guarantee it cannot honour.
// The whole value of "this endpoint requires authentication" is that it is a
// property an operator can rely on rather than hope for, and a property that
// degrades silently to its opposite under a misconfiguration is not one.
//
// # What stays legal
//
// An empty MethodSet with no verifier — the observe-only deployment T12.2
// shipped. That process claims nothing, so it can honour what it claims. The
// rule is about *claimed* enforcement, not about auth being configured
// everywhere, which is what keeps it from becoming a rule people route around.
func EnsureVerifierConfigured(verifier TokenVerifier, set MethodSet) error {
	if set.Len() == 0 {
		return nil
	}
	if !verifierIsNil(verifier) {
		return nil
	}
	return fmt.Errorf("%w: %d RPC(s) require an authenticated caller and none of them can ever admit one", ErrVerifierRequired, set.Len())
}

// verifierIsNil reports whether verifier is unusable, covering both an
// unset interface and an interface holding a nil pointer.
//
// The second case is the one a plain `verifier == nil` misses and the reason
// this is not a one-liner: a constructor that returns `(*rs256.Verifier)(nil)`
// alongside its error, and a caller that assigns it to a TokenVerifier before
// checking, produces a non-nil interface whose every method call panics. That
// is precisely the fail-open shape this function exists to stop, so catching
// only the easy half would leave the check looking complete while the more
// insidious case walked past it.
//
// (Belt and braces: a typed-nil verifier that reached the interceptors would
// now panic into their recover and deny the request — see interceptor.go — so
// the two halves of this ticket back each other up. Denying every request at
// runtime is still far worse than refusing to start, which is why this check
// is the primary one.)
func verifierIsNil(verifier TokenVerifier) bool {
	if verifier == nil {
		return true
	}
	switch v := reflect.ValueOf(verifier); v.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan:
		return v.IsNil()
	default:
		// A non-pointer implementation (rs256's is a value type in some
		// configurations, and the test fakes in this package are structs)
		// cannot be nil.
		return false
	}
}
