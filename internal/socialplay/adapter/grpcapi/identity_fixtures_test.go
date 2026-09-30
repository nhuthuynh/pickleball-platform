// T60 (closes issue #305) — this package's subject -> User.ID resolution for
// tests, the pass Payments got at T28.1 and Competitions got at T29.1 and
// Social Play did not.
//
// # What was wrong
//
// The shared fakeIdentityLookup returned the subject UNCHANGED, so
// `ctxAs("host-1")` produced `Game.HostID == "host-1"`. A real
// port.IdentityLookup cannot return that: `games.host_id` is
// `uuid NOT NULL REFERENCES identity_users (id)` as of
// db/migrations/0026_socialplay_identity_conformance.sql, which explicitly
// drops the old `text` column. Every test in this package that authenticated
// as a subject was therefore asserting against a state the schema forbids.
//
// # How it surfaced, and why it had hidden so long
//
// T59.1 (#296) added a uuidShape guard on Booking.OwnerUserID. Social Play
// passes Game.HostID as that owner when reserving courts, so five tests went
// red at once — all through newBookingBackedHandler, the one harness driving
// the REAL bookingapp.Service. They had passed because that harness's
// booking repository is in-memory and never reaches the Postgres adapter's
// mustUUID, which is what actually panics on a non-uuid.
//
// Production was never affected: the real adapter resolves a subject to an
// identity_users.id. This was a test-fidelity defect only — and precisely
// the class docs/LESSONS.md's T9 entry names, recurring in a fake of the one
// seam whose whole job is to translate a subject into something else.
//
// # Why the salt differs per package
//
// Each context salts its own hash ("socialplay-grpcapi-test-fixture:" here,
// "competitions-..." and "payments-..." in the others). A subject therefore
// resolves to a DIFFERENT User.ID in each package's tests, which is correct
// and deliberate: nothing cross-context may assume the two agree, and a
// shared value would invite exactly that assumption. Within one package the
// mapping is stable, which is all any assertion here needs.
package grpcapi_test

import (
	"crypto/sha256"
	"fmt"
	"testing"
)

// resolvedUserID deterministically maps a non-empty subject to a fixed,
// uuidShape-matching User.ID — the value a real port.IdentityLookup would
// resolve it to, had that subject been registered through Identity's
// CreateUser.
//
// Deterministic (not random, not counter-based) so any test file can compute
// "what will subject X resolve to" independently and always agree with
// itself and with fakeIdentityLookup.UserIDBySubject, without shared mutable
// state. Copied in shape from internal/payments/adapter/grpcapi and
// internal/competitions/adapter/grpcapi, which solved this identically.
//
// An empty subject maps to "" — mirroring port.IdentityLookup's convention
// that an empty subject is unregistered, never a valid input to resolve.
func resolvedUserID(subject string) string {
	if subject == "" {
		return ""
	}
	sum := sha256.Sum256([]byte("socialplay-grpcapi-test-fixture:" + subject))
	b := sum[:16]
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// TestResolvedUserIDIsInjectiveAcrossFixtureSubjects guards the property the
// rest of this package's security assertions quietly depend on.
//
// Tests like TestCreateGame_HostComesFromPrincipalNotWire prove that a
// caller cannot mint a Game hosted by someone else. They do that by
// authenticating as one subject, naming another on the wire, and asserting
// the result carries the FIRST one. If two fixture subjects ever resolved to
// the same User.ID, every one of those assertions would still pass and prove
// nothing at all — a silent, total loss of coverage on the checks that matter
// most here.
//
// Nothing about a sha256-derived value makes a collision plausible; the risk
// is a future edit to the salt or the truncation that accidentally makes the
// function constant, or returns its input. Both are caught here.
//
// Neither Payments' nor Competitions' copy of this fixture has this test.
// They should — the same assertions rest on the same property there — and
// that is noted rather than fixed from this package, which cannot reach
// their unexported helpers.
func TestResolvedUserIDIsInjectiveAcrossFixtureSubjects(t *testing.T) {
	t.Parallel()

	// Every subject this package authenticates as, gathered by reading the
	// _test.go files rather than guessed. A new fixture subject should be
	// added here; the cost of forgetting is only that it goes unchecked.
	subjects := []string{
		"host-1", "player-1", "player-A", "player-B", "admin-2",
		"auth0|player-1", "auth0|attacker-9", "auth0|never-registered",
		"ga-host", "ga-admin", "map-host", "wire-vs-store-admin",
		"player-admin", "some-other-player",
	}

	byResolved := make(map[string]string, len(subjects))
	for _, s := range subjects {
		got := resolvedUserID(s)

		if got == s {
			t.Errorf("resolvedUserID(%q) returned its own input — the resolver is a passthrough, which is the #305 defect", s)
			continue
		}
		if got == "" {
			t.Errorf("resolvedUserID(%q) = \"\", which port.IdentityLookup reserves for an unregistered subject", s)
			continue
		}
		if prev, dup := byResolved[got]; dup {
			t.Errorf("collision: %q and %q both resolve to %q — every principal-not-wire assertion in this package "+
				"would still pass while proving nothing", prev, s, got)
			continue
		}
		byResolved[got] = s
	}

	// And the empty subject keeps its documented meaning.
	if got := resolvedUserID(""); got != "" {
		t.Errorf("resolvedUserID(\"\") = %q, want \"\" — an empty subject is unregistered, never resolvable", got)
	}
}
