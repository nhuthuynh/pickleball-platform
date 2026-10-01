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
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
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

// --- T60.2 (closes issue #308) -------------------------------------------
//
// TestResolvedUserIDIsInjective guards the property every authorization test
// in this package silently depends on, and derives its own inputs rather
// than carrying a list.
//
// # What depends on it
//
// These tests work by authenticating as one subject, naming or seeding
// another, and asserting the two are treated differently. That proves
// something only if distinct subjects resolve to distinct User.IDs. If
// resolvedUserID ever became degenerate — a constant, or a passthrough —
// every one of them would STILL PASS while proving nothing. A total, silent
// loss of coverage on the BOLA and privilege-field regressions, which are
// the most security-load-bearing tests here.
//
// That is not a hypothetical class of failure. Social Play's equivalent fake
// returned the subject UNCHANGED for dozens of sprints (#305, fixed at T60),
// and its own thirty-plus authorization assertions detected it zero times —
// the only detections came accidentally, from a guard in another context.
//
// # Why the inputs are derived and not listed
//
// The inputs are every distinct string literal in this package's own test
// files, parsed at run time. There is deliberately NO list of subjects here,
// for the reason CLAUDE.md gives about `make gate-coverage`: "there is no
// package list in the tool, and adding one is the one change that would
// defeat it — three sprints running shipped a hand-written glob that was
// stale before its sprint ended."
//
// A hand-maintained subject list has exactly that failure mode. Someone adds
// a fixture subject, does not add it here, and the guard silently narrows.
// Deriving means a new subject is covered the moment it is written, and the
// check cannot rot.
//
// # Why over-inclusive inputs are the right trade
//
// This sweeps in strings that are not subjects at all — error messages,
// UUIDs, format strings. That is fine and deliberate: a degenerate resolver
// is caught by ANY two distinct inputs, so a wider net strictly dominates,
// and a narrower one buys only the ability to go stale. What it does not
// buy is a claim about sha256's collision resistance, which is not what is
// being tested — the failure being guarded is an edit to this file, not a
// cryptographic break.
func TestResolvedUserIDIsInjective(t *testing.T) {
	t.Parallel()

	inputs := stringLiteralsInPackageTests(t)
	if len(inputs) < 50 {
		t.Fatalf("derived only %d string literals from this package's tests — the derivation is broken, "+
			"which would make this guard silently vacuous", len(inputs))
	}

	byResolved := make(map[string]string, len(inputs))
	for _, in := range inputs {
		got := resolvedUserID(in)

		if got == in {
			t.Fatalf("resolvedUserID(%q) returned its own input — the resolver is a passthrough, which is "+
				"precisely the #305 defect: every principal-not-wire assertion would still pass while "+
				"comparing a subject against itself", in)
		}
		if got == "" {
			t.Fatalf("resolvedUserID(%q) = \"\", which port.IdentityLookup reserves for an unregistered "+
				"subject — a resolvable subject must never produce it", in)
		}
		if !uuidShapedFixture(got) {
			t.Fatalf("resolvedUserID(%q) = %q, which is not uuid-shaped — the real identity_users.id is a "+
				"uuid, and a fixture that is not makes this package model a state the schema forbids", in, got)
		}
		if prev, dup := byResolved[got]; dup {
			t.Fatalf("collision: %q and %q both resolve to %q — every assertion in this package that "+
				"distinguishes two principals would still pass while proving nothing", prev, in, got)
		}
		byResolved[got] = in
	}

	// The empty subject keeps its documented meaning. Checked separately
	// because the derivation above deliberately excludes it.
	if got := resolvedUserID(""); got != "" {
		t.Fatalf("resolvedUserID(\"\") = %q, want \"\" — an empty subject is unregistered, never resolvable", got)
	}
}

// fixtureUUIDShape is the canonical 8-4-4-4-12 hex form, duplicated here
// rather than imported from a context's app layer: this is a test fixture
// asserting on another test fixture, and reaching into production code for
// it would couple the two for no gain.
//
// Compiled once at package level rather than per call — the injective test
// above runs it over several hundred derived inputs.
var fixtureUUIDShape = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func uuidShapedFixture(s string) bool {
	return fixtureUUIDShape.MatchString(s)
}

// stringLiteralsInPackageTests returns every distinct non-empty string
// literal in this package's *_test.go files, by parsing them.
//
// Reads the package's own source at run time — the same technique
// tools/gatecoverage uses, and this package's own
// TestSentinelToCodeTableCoversEveryDomainSentinel, which parses
// domain/errors.go to prove the mapping table is complete. A test that
// derives its inputs from the tree cannot disagree with the tree.
func stringLiteralsInPackageTests(t *testing.T) []string {
	t.Helper()

	files, err := filepath.Glob("*_test.go")
	if err != nil {
		t.Fatalf("globbing this package's test files: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("found no *_test.go files in this package — the derivation cannot work from this directory")
	}

	seen := map[string]bool{}
	fset := token.NewFileSet()
	for _, f := range files {
		parsed, err := parser.ParseFile(fset, f, nil, 0)
		if err != nil {
			t.Fatalf("parsing %s: %v", f, err)
		}
		ast.Inspect(parsed, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			if v, err := strconv.Unquote(lit.Value); err == nil && v != "" {
				seen[v] = true
			}
			return true
		})
	}

	out := make([]string, 0, len(seen))
	for s := range seen {
		out = append(out, s)
	}
	sort.Strings(out) // deterministic failure messages
	return out
}
