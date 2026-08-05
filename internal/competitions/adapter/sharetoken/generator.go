// Package sharetoken implements port.ShareTokenGenerator with a
// cryptographically secure random source.
//
// Why this ships in T9.4 rather than waiting for T9.5, which owns the
// share-link flow: app.Service.ScheduleCompetition generates a token for
// every Competition at construction, and T9.4 is the ticket that makes
// CreateCompetition reachable over gRPC/REST for the first time. From the
// moment that endpoint is live, real tokens are being minted and stored in a
// NOT NULL UNIQUE column — so the entropy decision is due NOW, not when the
// links are first resolved. Wiring a placeholder (a counter, a UUID, a
// timestamp) and "hardening it in T9.5" would mean every Competition created
// in between carries a weak token, and port.ShareTokenGenerator's own doc
// comment is explicit that a predictable token is worse than no Competition.
//
// T9.5 remains free to extend this (token rotation, expiry, a resolve-by-
// token read path); what it must not have to do is retrofit the randomness.
package sharetoken

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/nhuthuynh/white-label/internal/competitions/port"
)

// tokenBytes is the number of random bytes behind each token: 32 bytes =
// 256 bits of entropy, encoded as 43 unpadded base64url characters.
//
// The threat model is enumeration, not brute-force of a single target: a
// share token is the only thing protecting an unpublished Competition from
// anyone who tries guessing, so the bar is "no feasible number of guesses
// finds any valid token", which 256 bits clears by an enormous margin. It is
// also the same order of magnitude as a session token, which is the right
// comparison — this value grants read access to a resource.
const tokenBytes = 32

// Generator implements port.ShareTokenGenerator using crypto/rand.
//
// A zero-size struct with no configuration, mirroring idgen.UUID: there is
// nothing to tune here that wouldn't be a way to make the token weaker.
type Generator struct{}

// Compile-time assertion that this satisfies the port, so a signature drift
// fails the build rather than surfacing as a nil-interface panic in
// cmd/server.
var _ port.ShareTokenGenerator = Generator{}

// NewShareToken returns a fresh, URL-safe, unguessable token.
//
// base64.RawURLEncoding (not StdEncoding): the URL-safe alphabet replaces
// '+' and '/' with '-' and '_', and "Raw" omits '=' padding — all three of
// the excluded characters require percent-encoding in a URL, and a token
// that changes shape depending on whether it was escaped is a token that
// will eventually be looked up with the wrong value.
//
// A read failure from crypto/rand is returned, never swallowed and never
// fallen back on: the port's contract is explicit that callers must treat it
// as fatal to the operation rather than degrade to a weaker token. In
// practice this means app.Service.ScheduleCompetition aborts before touching
// any other port, so a Competition is never created with a token that isn't
// fully random.
func (Generator) NewShareToken() (string, error) {
	buf := make([]byte, tokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("competitions sharetoken: reading %d random bytes: %w", tokenBytes, err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
