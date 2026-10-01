// T28.1 (closes the Payments third of #164) — this package's shared
// port.IdentityLookup test double, plus the deterministic subject->User.ID
// mapping every other *_test.go file in grpcapi_test needs once
// handler.go's actor() funnel starts resolving subjects instead of passing
// them through unchanged.
//
// Every existing test in this package authenticated as a literal string
// ("host-1", "admin-2", "attacker", ...) and, before this ticket, that SAME
// string flowed unchanged into RecordedByUserId, into BookingHostId
// comparisons, and into the T16.2 resolver fakes' (fakeGameLookup,
// fakeGameAdminReader, fakeEntryLookup, fakeCompetitionAdminReader) seeded
// host/admin/player values — because actor(ctx) returned the raw subject.
// As of T28.1 it returns a resolved User.ID, so every one of those
// call sites needs the SAME resolved value, not the raw subject, to keep
// comparing like with like.
//
// resolvedUserID is a pure, deterministic function rather than a shared
// stateful fake instance threaded through every constructor's return
// values, specifically so each test file's existing helper signatures don't
// all need a new return value: any test body can compute "what will subject
// X resolve to" independently, and get the same answer fakeIdentityLookup's
// own UserIDBySubject does, because both call this same function.
package grpcapi_test

import (
	"context"
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

	"github.com/nhuthuynh/white-label/internal/payments/domain"
)

// resolvedUserID deterministically maps a non-empty subject to a fixed,
// uuidShape-matching User.ID — the value a real port.IdentityLookup would
// resolve it to, if that subject had been registered through Identity's
// CreateUser. Deterministic (not random, not counter-based) so it can be
// called independently, any number of times, from any test file, and always
// agree with itself and with fakeIdentityLookup.UserIDBySubject below,
// without any shared mutable state.
//
// An empty subject maps to "" — mirrors port.IdentityLookup's documented
// convention that an empty subject is unregistered, never a valid input to
// resolve.
func resolvedUserID(subject string) string {
	if subject == "" {
		return ""
	}
	sum := sha256.Sum256([]byte("payments-grpcapi-test-fixture:" + subject))
	b := sum[:16]
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// fakeIdentityLookup stands in for internal/payments/adapter/identity,
// returning the same context-local sentinel that adapter translates
// Identity's ErrUserNotFound into (T28.1). It resolves EVERY non-empty,
// non-blocked subject via resolvedUserID above — auto-registration was
// chosen over an explicit subjects map (contrast
// internal/facilities/adapter/grpcapi/authz_regression_test.go's
// fakeIdentityLookup) because this package's *_test.go files between them
// authenticate as dozens of distinct literal subjects, and requiring each
// one to be enumerated in one central map here would be exactly the kind of
// change-amplification CLAUDE.md's simplicity bar warns against — every new
// test author would have to remember to register their fixture subject in a
// file they may not think to look at.
//
// unregistered names subjects that must resolve to domain.ErrUserNotFound —
// a verified-but-not-yet-a-User caller — for the one test in this package
// that specifically proves that path (see confirm_authz_test.go's
// TestConfirmOnlinePayment_UnregisteredActorIsPermissionDenied).
type fakeIdentityLookup struct {
	unregistered map[string]bool
}

func newFakeIdentityLookup(unregisteredSubjects ...string) *fakeIdentityLookup {
	l := &fakeIdentityLookup{unregistered: make(map[string]bool, len(unregisteredSubjects))}
	for _, s := range unregisteredSubjects {
		l.unregistered[s] = true
	}
	return l
}

func (l *fakeIdentityLookup) UserIDBySubject(_ context.Context, subject string) (string, error) {
	if subject == "" || l.unregistered[subject] {
		return "", domain.ErrUserNotFound
	}
	return resolvedUserID(subject), nil
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
