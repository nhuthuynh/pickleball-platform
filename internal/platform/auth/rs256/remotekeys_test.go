package rs256_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nhuthuynh/white-label/internal/platform/auth"
	"github.com/nhuthuynh/white-label/internal/platform/auth/rs256"
)

// Everything in this file runs against an httptest server holding
// locally-minted keys. There is no external identity provider, no tenant to
// provision, and no network egress — which is what makes the *remote* half of
// auth testable in the same gate as the rest (issue #137, T15.7).
//
// The server is TLS because RemoteKeys refuses a plaintext URL: a JWKS fetched
// over http:// can be replaced in flight by anyone on the path, and a verifier
// that trusts an attacker-supplied key set verifies attacker-minted tokens. So
// the tests exercise the same scheme production must use.

const (
	remoteTTL         = 15 * time.Minute
	remoteMinRefresh  = time.Minute
	remoteStaleWindow = time.Hour
)

// jwksServer is a stand-in for a provider's /.well-known/jwks.json: it counts
// requests (so "did this refetch?" is an assertion rather than a guess), and
// its document and status can be changed mid-test to model a key rotation or
// an outage.
type jwksServer struct {
	t *testing.T

	mu       sync.Mutex
	document []byte
	status   int
	requests int

	srv *httptest.Server
}

func newJWKSServer(t *testing.T, document []byte) *jwksServer {
	t.Helper()

	s := &jwksServer{t: t, document: document, status: http.StatusOK}
	s.srv = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		s.mu.Lock()
		status, body := s.status, s.document
		s.requests++
		s.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write(body)
	}))
	t.Cleanup(s.srv.Close)
	return s
}

func (s *jwksServer) url() string { return s.srv.URL + "/.well-known/jwks.json" }

// client trusts this server's throwaway certificate and nothing else.
func (s *jwksServer) client() *http.Client { return s.srv.Client() }

func (s *jwksServer) serve(document []byte, status int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.document, s.status = document, status
}

func (s *jwksServer) hits() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.requests
}

// stop makes the endpoint unreachable, the way a provider outage or a DNS
// failure would. Calling it twice is a test bug, so it is not idempotent by
// accident — t.Cleanup's second Close is tolerated by httptest itself.
func (s *jwksServer) stop() { s.srv.Close() }

// testClock is the clock seam TTL and rate-limit behaviour are asserted
// against. Wall-clock sleeps would make these tests slow and flaky; the
// property under test is "what happens when 16 minutes pass", not "what
// happens when the machine is busy".
type testClock struct {
	mu  sync.Mutex
	now time.Time
}

func newTestClock() *testClock { return &testClock{now: fixedNow} }

func (c *testClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *testClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

// remoteFixture wires a RemoteKeys against a jwksServer with the clock under
// the test's control.
type remoteFixture struct {
	server *jwksServer
	clock  *testClock
	keys   *rs256.RemoteKeys
}

func newRemoteFixture(t *testing.T, document []byte) remoteFixture {
	t.Helper()

	server := newJWKSServer(t, document)
	clock := newTestClock()

	keys, err := rs256.NewRemoteKeys(rs256.RemoteConfig{
		URL:                server.url(),
		HTTPClient:         server.client(),
		TTL:                remoteTTL,
		MinRefreshInterval: remoteMinRefresh,
		StaleIfUnreachable: remoteStaleWindow,
		Now:                clock.Now,
	})
	if err != nil {
		t.Fatalf("NewRemoteKeys: %v", err)
	}
	return remoteFixture{server: server, clock: clock, keys: keys}
}

func TestNewRemoteKeysRejectsUnusableConfig(t *testing.T) {
	t.Parallel()

	const valid = "https://tenant.example.com/.well-known/jwks.json"

	tests := []struct {
		name    string
		cfg     rs256.RemoteConfig
		wantErr string
	}{
		{
			name:    "empty URL",
			cfg:     rs256.RemoteConfig{},
			wantErr: "URL is required",
		},
		{
			// The security case for this is the whole reason the field is
			// validated rather than passed to http.Get: over plaintext, the
			// key set is attacker-writable and so is every token it verifies.
			name:    "plaintext http URL",
			cfg:     rs256.RemoteConfig{URL: "http://tenant.example.com/.well-known/jwks.json"},
			wantErr: "https",
		},
		{
			name:    "unparseable URL",
			cfg:     rs256.RemoteConfig{URL: "https://tenant.example.com/%zz"},
			wantErr: "not a valid URL",
		},
		{
			name:    "no host",
			cfg:     rs256.RemoteConfig{URL: "https:///.well-known/jwks.json"},
			wantErr: "host",
		},
		{
			name:    "negative TTL",
			cfg:     rs256.RemoteConfig{URL: valid, TTL: -time.Second},
			wantErr: "TTL",
		},
		{
			name:    "negative MinRefreshInterval",
			cfg:     rs256.RemoteConfig{URL: valid, MinRefreshInterval: -time.Second},
			wantErr: "MinRefreshInterval",
		},
		{
			name:    "negative StaleIfUnreachable",
			cfg:     rs256.RemoteConfig{URL: valid, StaleIfUnreachable: -time.Second},
			wantErr: "StaleIfUnreachable",
		},
		{
			name:    "negative MaxDocumentBytes",
			cfg:     rs256.RemoteConfig{URL: valid, MaxDocumentBytes: -1},
			wantErr: "MaxDocumentBytes",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := rs256.NewRemoteKeys(tc.cfg)
			if err == nil {
				t.Fatalf("NewRemoteKeys(%+v) succeeded; want an error", tc.cfg)
			}
			if got != nil {
				t.Error("NewRemoteKeys returned a non-nil KeySource alongside an error")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error %q does not mention %q", err.Error(), tc.wantErr)
			}
		})
	}

	t.Run("a well-formed config is accepted", func(t *testing.T) {
		t.Parallel()

		// Construction must not perform I/O: this URL resolves to nothing.
		got, err := rs256.NewRemoteKeys(rs256.RemoteConfig{URL: valid})
		if err != nil {
			t.Fatalf("NewRemoteKeys: %v", err)
		}
		if got == nil {
			t.Fatal("NewRemoteKeys returned nil with no error")
		}
	})
}

func TestRemoteKeysFetchesAndCaches(t *testing.T) {
	t.Parallel()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	t.Run("serves the key the provider publishes", func(t *testing.T) {
		t.Parallel()

		f := newRemoteFixture(t, jwksFor(t, map[string]*rsa.PublicKey{"kid-a": &priv.PublicKey}))

		got, err := f.keys.PublicKey(context.Background(), "kid-a")
		if err != nil {
			t.Fatalf("PublicKey: %v", err)
		}
		if got.N.Cmp(priv.N) != 0 || got.E != priv.E {
			t.Error("PublicKey returned a key that is not the one the endpoint published")
		}
		if f.server.hits() != 1 {
			t.Errorf("provider was called %d times for the first key, want 1", f.server.hits())
		}
	})

	t.Run("does not call the provider again while the cache is fresh", func(t *testing.T) {
		t.Parallel()

		f := newRemoteFixture(t, jwksFor(t, map[string]*rsa.PublicKey{"kid-a": &priv.PublicKey}))

		// The point of the cache: without it every RPC on this platform would
		// take a dependency on the provider's uptime and latency.
		for i := range 20 {
			if _, err := f.keys.PublicKey(context.Background(), "kid-a"); err != nil {
				t.Fatalf("PublicKey call %d: %v", i, err)
			}
			f.clock.advance(time.Second)
		}
		if f.server.hits() != 1 {
			t.Errorf("provider was called %d times across 20 verifications, want 1", f.server.hits())
		}
	})

	t.Run("refetches once the TTL has expired", func(t *testing.T) {
		t.Parallel()

		rotated, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatalf("generate rotated key: %v", err)
		}

		f := newRemoteFixture(t, jwksFor(t, map[string]*rsa.PublicKey{"kid-a": &priv.PublicKey}))
		if _, err := f.keys.PublicKey(context.Background(), "kid-a"); err != nil {
			t.Fatalf("priming PublicKey: %v", err)
		}

		// The provider rotates: same kid, new key material.
		f.server.serve(jwksFor(t, map[string]*rsa.PublicKey{"kid-a": &rotated.PublicKey}), http.StatusOK)
		f.clock.advance(remoteTTL + time.Second)

		got, err := f.keys.PublicKey(context.Background(), "kid-a")
		if err != nil {
			t.Fatalf("PublicKey after TTL: %v", err)
		}
		if got.N.Cmp(rotated.N) != 0 {
			t.Error("PublicKey still returns the pre-rotation key after the TTL expired")
		}
		if f.server.hits() != 2 {
			t.Errorf("provider was called %d times, want 2 (once to prime, once after the TTL)", f.server.hits())
		}
	})
}

func TestRemoteKeysRotation(t *testing.T) {
	t.Parallel()

	oldKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	newKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	t.Run("an unknown kid triggers exactly one refresh and resolves the new key", func(t *testing.T) {
		t.Parallel()

		f := newRemoteFixture(t, jwksFor(t, map[string]*rsa.PublicKey{"kid-old": &oldKey.PublicKey}))
		if _, err := f.keys.PublicKey(context.Background(), "kid-old"); err != nil {
			t.Fatalf("priming PublicKey: %v", err)
		}

		// A rotation mid-TTL: tokens start arriving with a kid this process
		// has never seen. Waiting out the TTL would reject every one of them,
		// which is the redeploy-or-outage failure #137 describes.
		f.server.serve(jwksFor(t, map[string]*rsa.PublicKey{
			"kid-old": &oldKey.PublicKey,
			"kid-new": &newKey.PublicKey,
		}), http.StatusOK)
		// Past the rate-limit floor, which is anchored on the last fetch
		// attempt — see the window subtest below for what happens inside it.
		f.clock.advance(remoteMinRefresh + time.Second)

		got, err := f.keys.PublicKey(context.Background(), "kid-new")
		if err != nil {
			t.Fatalf("PublicKey for the rotated kid: %v", err)
		}
		if got.N.Cmp(newKey.N) != 0 {
			t.Error("PublicKey returned the wrong key for the rotated kid")
		}
		if f.server.hits() != 2 {
			t.Errorf("provider was called %d times, want 2 (prime + one rotation refresh)", f.server.hits())
		}

		// And the refreshed set is now cached like any other.
		if _, err := f.keys.PublicKey(context.Background(), "kid-new"); err != nil {
			t.Fatalf("second PublicKey for the rotated kid: %v", err)
		}
		if f.server.hits() != 2 {
			t.Errorf("provider was called %d times after a repeat lookup, want 2", f.server.hits())
		}
	})

	t.Run("an unknown kid inside the rate-limit window is denied without calling the provider", func(t *testing.T) {
		t.Parallel()

		f := newRemoteFixture(t, jwksFor(t, map[string]*rsa.PublicKey{"kid-old": &oldKey.PublicKey}))
		if _, err := f.keys.PublicKey(context.Background(), "kid-old"); err != nil {
			t.Fatalf("priming PublicKey: %v", err)
		}

		// The floor is anchored on the last fetch *attempt*, so a kid nobody
		// published, arriving seconds after a fetch, costs the provider
		// nothing at all. The cost of that choice is bounded and stated: a
		// genuine rotation is picked up within one MinRefreshInterval.
		f.clock.advance(remoteMinRefresh / 2)
		if _, err := f.keys.PublicKey(context.Background(), "kid-new"); err == nil {
			t.Fatal("PublicKey resolved a kid that is in no fetched key set")
		}
		if f.server.hits() != 1 {
			t.Errorf("provider was called %d times inside the rate-limit window, want 1 (the prime only)", f.server.hits())
		}
	})

	t.Run("a kid the provider does not have is denied and rate-limits the refresh", func(t *testing.T) {
		t.Parallel()

		f := newRemoteFixture(t, jwksFor(t, map[string]*rsa.PublicKey{"kid-old": &oldKey.PublicKey}))
		if _, err := f.keys.PublicKey(context.Background(), "kid-old"); err != nil {
			t.Fatalf("priming PublicKey: %v", err)
		}
		f.clock.advance(remoteMinRefresh + time.Second)

		// An unauthenticated caller chooses the kid header. Without a floor
		// between refreshes, that is an amplified DoS against the identity
		// provider, aimed by anyone who can reach this API.
		for i := range 50 {
			key, err := f.keys.PublicKey(context.Background(), "kid-attacker-chosen")
			if err == nil {
				t.Fatalf("call %d: PublicKey returned a key for a kid the provider never published", i)
			}
			if key != nil {
				t.Fatalf("call %d: PublicKey returned a non-nil key alongside an error", i)
			}
			if !errors.Is(err, auth.ErrKeyUnavailable) {
				t.Fatalf("call %d: errors.Is(err, auth.ErrKeyUnavailable) = false for %v", i, err)
			}
			f.clock.advance(time.Second)
		}

		// 50 requests, one refresh: the first unknown kid is a plausible
		// rotation signal, the other 49 arrive inside the rate-limit window.
		if f.server.hits() != 2 {
			t.Errorf("provider was called %d times for 50 unknown-kid lookups, want 2 (prime + one refresh)", f.server.hits())
		}
	})

	t.Run("the rate limit lifts once the interval has passed", func(t *testing.T) {
		t.Parallel()

		f := newRemoteFixture(t, jwksFor(t, map[string]*rsa.PublicKey{"kid-old": &oldKey.PublicKey}))
		if _, err := f.keys.PublicKey(context.Background(), "kid-old"); err != nil {
			t.Fatalf("priming PublicKey: %v", err)
		}
		if _, err := f.keys.PublicKey(context.Background(), "kid-new"); err == nil {
			t.Fatal("PublicKey resolved a kid the provider has not published yet")
		}

		// The provider finishes its rotation a few minutes later. A rate limit
		// that never lifted would be a permanent outage rather than a bound.
		f.server.serve(jwksFor(t, map[string]*rsa.PublicKey{"kid-new": &newKey.PublicKey}), http.StatusOK)
		f.clock.advance(remoteMinRefresh + time.Second)

		got, err := f.keys.PublicKey(context.Background(), "kid-new")
		if err != nil {
			t.Fatalf("PublicKey after the rate-limit window: %v", err)
		}
		if got.N.Cmp(newKey.N) != 0 {
			t.Error("PublicKey returned the wrong key after the rate limit lifted")
		}
	})

	t.Run("a concurrent burst on an unknown kid refreshes once, not once per request", func(t *testing.T) {
		t.Parallel()

		f := newRemoteFixture(t, jwksFor(t, map[string]*rsa.PublicKey{"kid-old": &oldKey.PublicKey}))
		if _, err := f.keys.PublicKey(context.Background(), "kid-old"); err != nil {
			t.Fatalf("priming PublicKey: %v", err)
		}
		f.server.serve(jwksFor(t, map[string]*rsa.PublicKey{
			"kid-old": &oldKey.PublicKey,
			"kid-new": &newKey.PublicKey,
		}), http.StatusOK)
		f.clock.advance(remoteMinRefresh + time.Second)

		// A rotation on a busy server hits every in-flight request at once.
		const callers = 20
		var wg sync.WaitGroup
		errs := make([]error, callers)
		for i := range callers {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, errs[i] = f.keys.PublicKey(context.Background(), "kid-new")
			}()
		}
		wg.Wait()

		for i, err := range errs {
			if err != nil {
				t.Errorf("caller %d: %v", i, err)
			}
		}
		if f.server.hits() != 2 {
			t.Errorf("provider was called %d times for a %d-caller burst, want 2 (prime + one shared refresh)", f.server.hits(), callers)
		}
	})
}

// TestRemoteKeysFailsClosed is the ticket's security property: a source that
// cannot reach a conclusion denies, and never admits.
//
// Each row breaks the provider in a different way and asserts the same two
// things — no key comes back, and the error is auth.ErrKeyUnavailable, which
// ADR-0013 §5 distinguishes from a token rejection so an operator can tell
// "our provider is down" from "this caller's token is bad".
func TestRemoteKeysFailsClosed(t *testing.T) {
	t.Parallel()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	good := jwksFor(t, map[string]*rsa.PublicKey{"kid-a": &priv.PublicKey})

	tests := []struct {
		name string
		// break_ mutates the provider before the first lookup. There is no
		// cached key set at that point, so there is nothing to fall back to.
		break_ func(f remoteFixture)
	}{
		{
			name:   "provider is unreachable",
			break_: func(f remoteFixture) { f.server.stop() },
		},
		{
			name:   "provider returns HTML instead of a JWKS",
			break_: func(f remoteFixture) { f.server.serve([]byte("<html>502 Bad Gateway</html>"), http.StatusOK) },
		},
		{
			name:   "provider returns truncated JSON",
			break_: func(f remoteFixture) { f.server.serve([]byte(`{"keys":[{"kty":"RSA"`), http.StatusOK) },
		},
		{
			name:   "provider returns an empty key set",
			break_: func(f remoteFixture) { f.server.serve([]byte(`{"keys":[]}`), http.StatusOK) },
		},
		{
			name:   "provider returns 500",
			break_: func(f remoteFixture) { f.server.serve(good, http.StatusInternalServerError) },
		},
		{
			name:   "provider returns 404",
			break_: func(f remoteFixture) { f.server.serve([]byte("not found"), http.StatusNotFound) },
		},
		{
			// A JWKS is untrusted input: an unbounded read is a memory DoS
			// with no authentication required to trigger it.
			name: "provider returns a document larger than the cap",
			break_: func(f remoteFixture) {
				f.server.serve([]byte(`{"keys":[`+strings.Repeat(" ", 2<<20)+`]}`), http.StatusOK)
			},
		},
		{
			// Parsed defensively by NewStaticKeysFromJWKS — asserted here too
			// because the remote path must not acquire a second, laxer parser.
			name: "provider publishes an undersized key",
			break_: func(f remoteFixture) {
				weak, err := rsa.GenerateKey(rand.Reader, 1024)
				if err != nil {
					t.Fatalf("generate weak key: %v", err)
				}
				f.server.serve(jwksFor(t, map[string]*rsa.PublicKey{"kid-a": &weak.PublicKey}), http.StatusOK)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			f := newRemoteFixture(t, good)
			tc.break_(f)

			key, err := f.keys.PublicKey(context.Background(), "kid-a")
			if err == nil {
				t.Fatal("PublicKey returned no error; a key source that cannot reach the provider must deny, not admit")
			}
			if key != nil {
				t.Error("PublicKey returned a non-nil key alongside an error")
			}
			if !errors.Is(err, auth.ErrKeyUnavailable) {
				t.Errorf("errors.Is(err, auth.ErrKeyUnavailable) = false for %v", err)
			}
		})
	}
}

// TestRemoteKeysStaleFallbackIsBounded pins the one place this package accepts
// a cached answer the provider has not reconfirmed — and pins where that stops.
//
// Serving a key past its TTL during an outage is not fail-open: the key was
// fetched from the provider over TLS and the provider published it. But it is
// unbounded staleness that turns a revoked key into a permanent one, so the
// window has a ceiling, and past the ceiling every token is denied.
func TestRemoteKeysStaleFallbackIsBounded(t *testing.T) {
	t.Parallel()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	newOutage := func(t *testing.T) remoteFixture {
		t.Helper()
		f := newRemoteFixture(t, jwksFor(t, map[string]*rsa.PublicKey{"kid-a": &priv.PublicKey}))
		if _, err := f.keys.PublicKey(context.Background(), "kid-a"); err != nil {
			t.Fatalf("priming PublicKey: %v", err)
		}
		f.server.stop()
		return f
	}

	t.Run("a fresh cache is unaffected by the outage", func(t *testing.T) {
		t.Parallel()

		f := newOutage(t)
		f.clock.advance(remoteTTL / 2)

		if _, err := f.keys.PublicKey(context.Background(), "kid-a"); err != nil {
			t.Fatalf("PublicKey during an outage inside the TTL: %v", err)
		}
	})

	t.Run("a cache past its TTL is still served inside the stale window", func(t *testing.T) {
		t.Parallel()

		f := newOutage(t)
		f.clock.advance(remoteTTL + remoteStaleWindow/2)

		if _, err := f.keys.PublicKey(context.Background(), "kid-a"); err != nil {
			t.Fatalf("PublicKey inside the stale window: %v; a brief provider outage must not take the whole API down", err)
		}
	})

	t.Run("past the stale ceiling every token is denied", func(t *testing.T) {
		t.Parallel()

		f := newOutage(t)
		f.clock.advance(remoteTTL + remoteStaleWindow + time.Second)

		key, err := f.keys.PublicKey(context.Background(), "kid-a")
		if err == nil {
			t.Fatal("PublicKey served a key past the stale ceiling; the fallback is bounded by design")
		}
		if key != nil {
			t.Error("PublicKey returned a non-nil key alongside an error")
		}
		if !errors.Is(err, auth.ErrKeyUnavailable) {
			t.Errorf("errors.Is(err, auth.ErrKeyUnavailable) = false for %v", err)
		}
	})

	t.Run("an unknown kid is never served from a stale cache", func(t *testing.T) {
		t.Parallel()

		f := newOutage(t)
		f.clock.advance(remoteTTL + remoteStaleWindow/2)

		// The stale window relaxes *when* a key was confirmed, never *which*
		// keys are trusted. A kid nobody published stays unavailable.
		if _, err := f.keys.PublicKey(context.Background(), "kid-unknown"); err == nil {
			t.Fatal("PublicKey resolved a kid that was never in any fetched key set")
		}
	})
}

// TestRemoteKeysRefresh covers the startup warm-up cmd/server performs, which
// is what turns a typo in AUTH_JWKS_URL into a startup error instead of a
// server that runs and denies every request.
func TestRemoteKeysRefresh(t *testing.T) {
	t.Parallel()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	t.Run("primes the cache from a reachable provider", func(t *testing.T) {
		t.Parallel()

		f := newRemoteFixture(t, jwksFor(t, map[string]*rsa.PublicKey{"kid-a": &priv.PublicKey}))
		if err := f.keys.Refresh(context.Background()); err != nil {
			t.Fatalf("Refresh: %v", err)
		}
		if f.server.hits() != 1 {
			t.Errorf("provider was called %d times, want 1", f.server.hits())
		}

		// Primed: the first verification costs no request.
		if _, err := f.keys.PublicKey(context.Background(), "kid-a"); err != nil {
			t.Fatalf("PublicKey after Refresh: %v", err)
		}
		if f.server.hits() != 1 {
			t.Errorf("provider was called %d times after a primed lookup, want 1", f.server.hits())
		}
	})

	t.Run("reports an unreachable provider", func(t *testing.T) {
		t.Parallel()

		f := newRemoteFixture(t, jwksFor(t, map[string]*rsa.PublicKey{"kid-a": &priv.PublicKey}))
		f.server.stop()

		err := f.keys.Refresh(context.Background())
		if err == nil {
			t.Fatal("Refresh against an unreachable provider returned nil")
		}
		if !errors.Is(err, auth.ErrKeyUnavailable) {
			t.Errorf("errors.Is(err, auth.ErrKeyUnavailable) = false for %v", err)
		}
	})
}

// TestVerifierWithRemoteKeys is the end-to-end proof, and the mutation check
// rule 10 asks for: a real RS256 token, minted locally, verified through
// rs256.Verifier against keys fetched over HTTPS — and then denied once the
// provider has been unreachable past the ceiling.
//
// It is deliberately written at the Verify boundary rather than the KeySource
// boundary, because "the fetch failed" only matters if it becomes "this token
// is rejected" by the time a caller sees it.
func TestVerifierWithRemoteKeys(t *testing.T) {
	t.Parallel()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	keys := testKeys{private: priv}

	f := newRemoteFixture(t, jwksFor(t, map[string]*rsa.PublicKey{testKeyID: &priv.PublicKey}))
	verifier, err := rs256.NewVerifier(rs256.Config{
		Issuer:   testIssuer,
		Audience: testAudience,
		Keys:     f.keys,
		Now:      f.clock.Now,
	})
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}

	token := mint(t, keys, validOpts())

	principal, err := verifier.Verify(context.Background(), token)
	if err != nil {
		t.Fatalf("Verify against a live provider: %v", err)
	}
	if principal.Subject != testSubject {
		t.Errorf("Principal.Subject = %q, want %q", principal.Subject, testSubject)
	}

	// The provider goes away and stays away. Everything below is the same
	// token that just verified: the only thing that changed is our ability to
	// confirm the key it was signed with.
	f.server.stop()

	// Long-lived token, so expiry cannot be what rejects it and mask the
	// property under test.
	longLived := validOpts()
	longLived.expiresAt = fixedNow.Add(remoteTTL + remoteStaleWindow + time.Hour)
	token = mint(t, keys, longLived)

	f.clock.advance(remoteTTL + remoteStaleWindow + time.Second)

	got, err := verifier.Verify(context.Background(), token)
	if err == nil {
		t.Fatal("Verify accepted a token after the provider had been unreachable past the stale ceiling; " +
			"a verifier that cannot obtain the signing key must deny")
	}
	if !errors.Is(err, auth.ErrKeyUnavailable) {
		t.Errorf("errors.Is(err, auth.ErrKeyUnavailable) = false for %v", err)
	}
	if auth.IsTokenRejection(err) {
		t.Error("a provider outage was reported as a token rejection; ADR-0013 §5 keeps these distinct")
	}
	if got.Subject != "" {
		t.Errorf("Verify returned a non-zero Principal (%+v) alongside an error", got)
	}
}

// compile-time proof the remote source satisfies the seam ADR-0013 §3 split
// out for it, so it drops in wherever StaticKeys does.
var _ rs256.KeySource = (*rs256.RemoteKeys)(nil)
