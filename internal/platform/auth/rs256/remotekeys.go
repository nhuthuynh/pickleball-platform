package rs256

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/nhuthuynh/white-label/internal/platform/auth"
)

// Defaults for RemoteConfig. Each is a security or availability bound, not a
// tuning knob, so each says what breaks if it is set wrong.
const (
	// defaultTTL is how long a fetched key set is served without re-checking
	// the provider. Short enough that a rotation is picked up on its own
	// within the quarter hour even if no token ever names the new kid; long
	// enough that the provider sees a trickle of requests rather than one per
	// RPC. The unknown-kid path below is what makes a *fast* rotation work;
	// this is the backstop for the case that path cannot see.
	defaultTTL = 15 * time.Minute

	// defaultMinRefreshInterval is the floor between two fetch attempts.
	//
	// This is the bound issue #137 asks for by name, and it is a security
	// bound rather than politeness: the `kid` header is chosen by whoever
	// sends the token, and no authentication has happened yet at the moment it
	// is read. Without a floor, an anonymous caller can turn one cheap request
	// to this API into one request to the identity provider, at whatever rate
	// they like — an amplified DoS aimed at the provider, with this backend as
	// the amplifier.
	//
	// The floor is anchored on the last fetch *attempt*, successful or not, so
	// a failing provider is not retried per request either. The cost is stated
	// rather than hidden: a genuine rotation whose new kid appears within this
	// window of the previous fetch is picked up at the end of it, so tokens
	// naming a brand-new kid can be rejected for up to one interval. Providers
	// publish a new key before signing with it, which is what makes that
	// bound acceptable; removing it is not.
	defaultMinRefreshInterval = time.Minute

	// defaultStaleIfUnreachable is how far past the TTL a cached key set may
	// still be served *when refreshing fails*. See the discussion on
	// RemoteKeys about why this is not fail-open, and why it is bounded.
	defaultStaleIfUnreachable = time.Hour

	// defaultMaxDocumentBytes caps the JWKS document read from the wire. A
	// JWKS is untrusted input (research-security-compliance.md §1, API10), and
	// an unbounded io.ReadAll against a hostile or broken endpoint is a memory
	// exhaustion bug that needs no credentials to trigger. Real key sets are a
	// few kilobytes.
	defaultMaxDocumentBytes = 1 << 20 // 1 MiB

	// defaultRequestTimeout bounds a single fetch. Without it, a provider that
	// accepts connections and never answers holds the refresh lock — and so
	// every verification that needs a refresh — for as long as it likes.
	defaultRequestTimeout = 10 * time.Second
)

// errRefreshRateLimited is the internal signal that a refresh was declined by
// the rate limiter rather than attempted and failed. It never escapes this
// package: callers see auth.ErrKeyUnavailable with this as explanation.
var errRefreshRateLimited = errors.New("refresh declined: minimum interval between fetches has not elapsed")

// RemoteConfig describes the identity provider's JWKS endpoint and the two
// bounds that keep fetching it safe.
//
// Every duration may be left zero to take the documented default. Negative
// values are a construction error rather than a silent clamp: a negative TTL
// means "always refetch", which is exactly the per-RPC dependency on the
// provider that this type exists to avoid, and it should not be reachable by a
// typo in a deployment's configuration.
type RemoteConfig struct {
	// URL is the provider's JWK Set endpoint, e.g.
	// https://your-tenant.auth0.com/.well-known/jwks.json
	//
	// It must be https. A JWKS fetched over plaintext can be replaced in
	// flight by anyone on the path, and a verifier that trusts an
	// attacker-supplied key set will happily verify attacker-minted tokens —
	// which is worse than having no verifier at all, because it looks like a
	// check in review and in logs. Deployments that genuinely need a
	// non-TLS or file-backed source use AUTH_JWKS_FILE and StaticKeys.
	URL string

	// HTTPClient fetches the document. Nil means a client with no ambient
	// state (no cookie jar, no shared transport surprises) and the default
	// request timeout.
	HTTPClient *http.Client

	// TTL is how long a fetched key set is served before a refresh is
	// attempted. Zero means defaultTTL.
	TTL time.Duration

	// MinRefreshInterval is the floor between two fetch attempts, successful
	// or not. Zero means defaultMinRefreshInterval.
	MinRefreshInterval time.Duration

	// StaleIfUnreachable is how far past TTL a cached set may still be served
	// while the provider cannot be reached. Zero means
	// defaultStaleIfUnreachable.
	StaleIfUnreachable time.Duration

	// MaxDocumentBytes caps the JWKS read from the wire. Zero means
	// defaultMaxDocumentBytes.
	MaxDocumentBytes int64

	// RequestTimeout bounds one fetch. Zero means defaultRequestTimeout.
	RequestTimeout time.Duration

	// Now is a clock seam for tests. Nil means time.Now.
	Now func() time.Time
}

// RemoteKeys is a KeySource backed by an identity provider's JWKS endpoint: it
// fetches the key set over HTTPS, caches it, refreshes it on a TTL, and
// refreshes it early — under a rate limit — when a token names a `kid` the
// cached set does not hold, which is what a key rotation looks like from here.
//
// It is the implementation ADR-0013 §3 predicted when it split KeySource out
// of the verifier, and it changes no line of the verification logic.
//
// # Fail-closed, and where the one exception stops
//
// Every path that cannot produce a key the provider published returns an error
// wrapping auth.ErrKeyUnavailable, and no path ever returns a key it did not
// get from a successfully parsed, TLS-fetched document. In particular:
//
//   - No cached set and the fetch failed → deny. There is nothing to fall back
//     to, and admitting the token would mean trusting a signature nobody
//     checked.
//   - Unknown kid, before or after a refresh → deny. Never "the only key we
//     have"; guessing here would defeat rotation entirely.
//   - Malformed, oversized, non-200, or wrong-shaped document → deny, and the
//     previously cached set is left untouched rather than replaced by a
//     partial parse.
//
// The single exception is deliberate and bounded: while the provider is
// unreachable, a cached set is served past its TTL for StaleIfUnreachable.
// That is not fail-open — those keys were fetched from the provider over TLS
// and are the keys it published; nothing new is admitted, and an unknown kid
// is still denied. What it buys is that a provider blip does not take every
// authenticated RPC on this platform down with it. What it costs is that a key
// the provider *revokes* during an outage keeps verifying tokens until the
// window closes, which is why the window has a ceiling at all. Past
// TTL+StaleIfUnreachable, every token is denied until a fetch succeeds.
//
// RemoteKeys is safe for concurrent use. A burst of requests arriving during a
// rotation produces one fetch, not one per request.
type RemoteKeys struct {
	url            string
	client         *http.Client
	ttl            time.Duration
	minRefresh     time.Duration
	staleTolerance time.Duration
	maxBytes       int64
	timeout        time.Duration
	now            func() time.Time

	// fetchMu serializes fetches, so a burst collapses into one request. It is
	// separate from mu because a fetch is slow and reading the cache must not
	// wait behind one.
	fetchMu sync.Mutex

	mu sync.Mutex
	// keys is the last successfully parsed key set, or nil if none has ever
	// been fetched. Replaced wholesale, never mutated.
	keys *StaticKeys
	// fetchedAt is when keys was fetched; lastAttempt is when a fetch was last
	// *attempted*, successfully or not — the rate limiter reads the latter, so
	// a failing provider is not retried on every request.
	fetchedAt   time.Time
	lastAttempt time.Time
}

// NewRemoteKeys validates the configuration and returns a KeySource. It
// performs no I/O: a process that cannot reach its provider at startup is a
// decision for the caller (cmd/server warms the cache and fails startup if it
// cannot), not something this constructor should make by blocking.
func NewRemoteKeys(cfg RemoteConfig) (*RemoteKeys, error) {
	raw := strings.TrimSpace(cfg.URL)
	if raw == "" {
		return nil, errors.New("rs256: RemoteConfig.URL is required (there is no endpoint to fetch keys from)")
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("rs256: RemoteConfig.URL is not a valid URL: %w", err)
	}
	if parsed.Scheme != "https" {
		return nil, fmt.Errorf(
			"rs256: RemoteConfig.URL must use https, got %q "+
				"(a JWKS fetched over a plaintext connection can be replaced in flight, "+
				"which would let an attacker choose the keys this process trusts)", parsed.Scheme)
	}
	if parsed.Host == "" {
		return nil, fmt.Errorf("rs256: RemoteConfig.URL %q has no host", raw)
	}

	if cfg.TTL < 0 {
		return nil, fmt.Errorf("rs256: RemoteConfig.TTL must not be negative, got %s", cfg.TTL)
	}
	if cfg.MinRefreshInterval < 0 {
		return nil, fmt.Errorf("rs256: RemoteConfig.MinRefreshInterval must not be negative, got %s", cfg.MinRefreshInterval)
	}
	if cfg.StaleIfUnreachable < 0 {
		return nil, fmt.Errorf("rs256: RemoteConfig.StaleIfUnreachable must not be negative, got %s", cfg.StaleIfUnreachable)
	}
	if cfg.MaxDocumentBytes < 0 {
		return nil, fmt.Errorf("rs256: RemoteConfig.MaxDocumentBytes must not be negative, got %d", cfg.MaxDocumentBytes)
	}
	if cfg.RequestTimeout < 0 {
		return nil, fmt.Errorf("rs256: RemoteConfig.RequestTimeout must not be negative, got %s", cfg.RequestTimeout)
	}

	r := &RemoteKeys{
		url:            raw,
		client:         cfg.HTTPClient,
		ttl:            orDuration(cfg.TTL, defaultTTL),
		minRefresh:     orDuration(cfg.MinRefreshInterval, defaultMinRefreshInterval),
		staleTolerance: orDuration(cfg.StaleIfUnreachable, defaultStaleIfUnreachable),
		maxBytes:       cfg.MaxDocumentBytes,
		timeout:        orDuration(cfg.RequestTimeout, defaultRequestTimeout),
		now:            cfg.Now,
	}
	if r.maxBytes == 0 {
		r.maxBytes = defaultMaxDocumentBytes
	}
	if r.client == nil {
		r.client = &http.Client{}
	}
	if r.now == nil {
		r.now = time.Now
	}
	return r, nil
}

func orDuration(configured, fallback time.Duration) time.Duration {
	if configured == 0 {
		return fallback
	}
	return configured
}

// URL reports the endpoint this source fetches from. Useful for startup logs,
// where an operator wants to see which tenant a process is actually pointed at.
func (r *RemoteKeys) URL() string { return r.url }

// Refresh fetches the key set now, regardless of TTL, and reports whether that
// succeeded. cmd/server calls it once at startup so a typo in AUTH_JWKS_URL is
// a startup failure rather than a server that runs and denies every request.
//
// It is still subject to the rate limit, so calling it in a loop cannot be
// used to hammer the provider.
func (r *RemoteKeys) Refresh(ctx context.Context) error {
	if err := r.refresh(ctx, r.snapshot()); err != nil {
		return fmt.Errorf("%w: fetching jwks from %s: %w", auth.ErrKeyUnavailable, r.url, err)
	}
	return nil
}

// PublicKey implements KeySource.
func (r *RemoteKeys) PublicKey(ctx context.Context, keyID string) (*rsa.PublicKey, error) {
	snap := r.snapshot()

	// The common case by a wide margin: a fresh cache holding the kid.
	if !snap.expired(r.now(), r.ttl) {
		if key := snap.lookup(ctx, keyID); key != nil {
			return key, nil
		}
	}

	// Otherwise one of three things is true — nothing has ever been fetched,
	// the set is past its TTL, or the kid is unknown (the rotation signal) —
	// and all three want the same thing: a refresh, if the rate limiter allows
	// one. refresh reports why it could not, which is the difference between
	// "the provider is down" (an incident) and "this caller named a kid that
	// does not exist" (a non-event).
	refreshErr := r.refresh(ctx, snap)

	snap = r.snapshot()
	key := snap.lookup(ctx, keyID)
	if key == nil {
		if refreshErr != nil && snap.keys == nil {
			// Nothing cached and nothing fetched: say so, rather than
			// reporting an unknown kid, which would blame the caller for our
			// outage.
			return nil, fmt.Errorf("%w: fetching jwks from %s: %w", auth.ErrKeyUnavailable, r.url, refreshErr)
		}
		return nil, fmt.Errorf("%w: no key for kid %q in the key set from %s", auth.ErrKeyUnavailable, keyID, r.url)
	}

	now := r.now()
	if !snap.expired(now, r.ttl) {
		return key, nil
	}
	// The key is cached but past its TTL, which means the refresh above did
	// not succeed. Bounded stale-serving, as documented on RemoteKeys.
	if age := now.Sub(snap.fetchedAt); age <= r.ttl+r.staleTolerance {
		return key, nil
	}
	return nil, fmt.Errorf(
		"%w: cached key set from %s is %s old, past the %s ceiling, and the provider is unreachable (last error: %v)",
		auth.ErrKeyUnavailable, r.url, now.Sub(snap.fetchedAt).Round(time.Second), r.ttl+r.staleTolerance, refreshErr)
}

// cacheState is an immutable read of the cache, taken under the lock so the
// decision logic above never sees a half-updated cache and never holds a lock
// across a network call.
type cacheState struct {
	keys        *StaticKeys
	fetchedAt   time.Time
	lastAttempt time.Time
}

func (s cacheState) expired(now time.Time, ttl time.Duration) bool {
	return s.keys == nil || now.Sub(s.fetchedAt) > ttl
}

func (s cacheState) lookup(ctx context.Context, keyID string) *rsa.PublicKey {
	if s.keys == nil {
		return nil
	}
	key, err := s.keys.PublicKey(ctx, keyID)
	if err != nil {
		return nil
	}
	return key
}

func (r *RemoteKeys) snapshot() cacheState {
	r.mu.Lock()
	defer r.mu.Unlock()
	return cacheState{keys: r.keys, fetchedAt: r.fetchedAt, lastAttempt: r.lastAttempt}
}

// refresh fetches and installs a new key set, unless another goroutine already
// did the work or the rate limiter declines.
//
// seen is the cache state the caller made its decision against. If the cache
// moved on while this call waited for fetchMu, another goroutine has already
// answered the same question — that is what collapses a rotation burst into a
// single request.
func (r *RemoteKeys) refresh(ctx context.Context, seen cacheState) error {
	r.fetchMu.Lock()
	defer r.fetchMu.Unlock()

	current := r.snapshot()
	if current.lastAttempt.After(seen.lastAttempt) || current.fetchedAt.After(seen.fetchedAt) {
		return nil
	}

	now := r.now()
	if !current.lastAttempt.IsZero() && now.Sub(current.lastAttempt) < r.minRefresh {
		return errRefreshRateLimited
	}

	r.mu.Lock()
	r.lastAttempt = now
	r.mu.Unlock()

	document, err := r.fetch(ctx)
	if err != nil {
		return err
	}
	// The same defensive parser the file-backed source uses. A second, laxer
	// parser here is exactly the two-shapes failure this project keeps naming
	// — and the lax one would be the one facing the network.
	keys, err := NewStaticKeysFromJWKS(document)
	if err != nil {
		return err
	}

	r.mu.Lock()
	r.keys = keys
	r.fetchedAt = r.now()
	r.mu.Unlock()
	return nil
}

// fetch performs one bounded HTTP GET and returns the raw document.
func (r *RemoteKeys) fetch(ctx context.Context) ([]byte, error) {
	// WithoutCancel: this fetch fills a cache shared by every in-flight
	// request, so one caller hanging up must not cancel the refresh the others
	// are waiting on — and must not burn the rate-limit window on a request
	// that never really happened. The timeout below is what bounds it instead.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), r.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.url, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("endpoint returned HTTP %s", resp.Status)
	}

	// Read one byte past the cap so "exactly at the cap" and "too large" are
	// distinguishable rather than silently truncated into a parse error.
	document, err := io.ReadAll(io.LimitReader(resp.Body, r.maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	if int64(len(document)) > r.maxBytes {
		return nil, fmt.Errorf("document exceeds the %d byte limit", r.maxBytes)
	}
	return document, nil
}

// compile-time proof this satisfies the seam it was written for.
var _ KeySource = (*RemoteKeys)(nil)
