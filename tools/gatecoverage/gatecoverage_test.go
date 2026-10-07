package gatecoverage

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- Makefile parsing -----------------------------------------------------

func TestParseMakefile(t *testing.T) {
	const src = "" +
		".PHONY: test-domain test-platform \\\n" +
		"        ci-checks ci\n" +
		"\n" +
		"GOFLAGS := -mod=readonly\n" +
		"\n" +
		"# a comment line\n" +
		"test-domain:\n" +
		"\tgo test ./internal/.../domain/... -race -count=1\n" +
		"\n" +
		"test-platform:\n" +
		"\tgo test ./internal/platform/... -race -count=1\n" +
		"\n" +
		"vet-integration: generate\n" +
		"\tgo vet -tags=integration ./...\n" +
		"\n" +
		"generate:\n" +
		"\tbuf generate\n" +
		"\n" +
		"ci-checks: generate test-domain test-platform vet-integration\n" +
		"\tgo build ./...\n" +
		"\n" +
		"ci: ci-checks\n" +
		"\t$(MAKE) security\n" +
		"\n" +
		"security:\n" +
		"ifeq ($(SKIP),1)\n" +
		"\t@echo skipping\n" +
		"else\n" +
		"\tgo run ./cmd/vulngate\n" +
		"endif\n"

	mf, err := ParseMakefile(strings.NewReader(src))
	if err != nil {
		t.Fatalf("ParseMakefile: %v", err)
	}

	tests := []struct {
		name    string
		target  string
		wantOK  bool
		prereqs []string
		recipe  []string
	}{
		{
			name:    "simple target with recipe",
			target:  "test-domain",
			wantOK:  true,
			prereqs: nil,
			recipe:  []string{"go test ./internal/.../domain/... -race -count=1"},
		},
		{
			name:    "target with a prerequisite",
			target:  "vet-integration",
			wantOK:  true,
			prereqs: []string{"generate"},
			recipe:  []string{"go vet -tags=integration ./..."},
		},
		{
			name:    "target with several prerequisites",
			target:  "ci-checks",
			wantOK:  true,
			prereqs: []string{"generate", "test-domain", "test-platform", "vet-integration"},
			recipe:  []string{"go build ./..."},
		},
		{
			name:   "line-continued .PHONY is one rule, not two",
			target: "ci",
			wantOK: true,
			// ci must survive the continued .PHONY line above it.
			prereqs: []string{"ci-checks"},
			recipe:  []string{"$(MAKE) security"},
		},
		{
			name:   "conditional directives do not terminate the recipe",
			target: "security",
			wantOK: true,
			recipe: []string{"@echo skipping", "go run ./cmd/vulngate"},
		},
		{
			name:   "a := assignment is not a rule",
			target: "GOFLAGS",
			wantOK: false,
		},
		{
			name:   "an unknown target is absent",
			target: "nope",
			wantOK: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rule, ok := mf.Rule(tc.target)
			if ok != tc.wantOK {
				t.Fatalf("Rule(%q) present = %v, want %v", tc.target, ok, tc.wantOK)
			}
			if !ok {
				return
			}
			if got := strings.Join(rule.Prereqs, ","); got != strings.Join(tc.prereqs, ",") {
				t.Errorf("prereqs = %q, want %q", got, strings.Join(tc.prereqs, ","))
			}
			if tc.recipe != nil {
				if got := strings.Join(rule.Recipe, "|"); got != strings.Join(tc.recipe, "|") {
					t.Errorf("recipe = %q, want %q", got, strings.Join(tc.recipe, "|"))
				}
			}
		})
	}

	// .PHONY is itself parsed as a rule; the closure walk must never be
	// rooted there (it would pull in every target in the file).
	if _, ok := mf.Rule(".PHONY"); !ok {
		t.Errorf(".PHONY should parse as a rule like any other")
	}
}

func TestReachableTargets(t *testing.T) {
	const src = "" +
		"a: b c\n" +
		"\techo a\n" +
		"b: d\n" +
		"\techo b\n" +
		"c:\n" +
		"\t$(MAKE) e\n" +
		"d:\n" +
		"\techo d\n" +
		"e:\n" +
		"\techo e\n" +
		"orphan:\n" +
		"\techo orphan\n"

	mf, err := ParseMakefile(strings.NewReader(src))
	if err != nil {
		t.Fatalf("ParseMakefile: %v", err)
	}

	tests := []struct {
		name string
		root string
		want string
	}{
		{name: "transitive prerequisites and $(MAKE) recursion", root: "a", want: "a,b,c,d,e"},
		{name: "a leaf reaches only itself", root: "d", want: "d"},
		{name: "orphans are not reachable from a", root: "orphan", want: "orphan"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := mf.Reachable(tc.root)
			if err != nil {
				t.Fatalf("Reachable(%q): %v", tc.root, err)
			}
			if strings.Join(got, ",") != tc.want {
				t.Errorf("Reachable(%q) = %v, want %v", tc.root, got, tc.want)
			}
		})
	}

	if _, err := mf.Reachable("missing"); err == nil {
		t.Errorf("Reachable on an undefined target should error, so a renamed gate root fails loudly")
	}
}

func TestGoTestPatterns(t *testing.T) {
	tests := []struct {
		name   string
		recipe string
		want   string
	}{
		{
			name:   "plain go test with flags after the pattern",
			recipe: "go test ./internal/.../domain/... -race -count=1",
			want:   "./internal/.../domain/...",
		},
		{
			name:   "several patterns on one line",
			recipe: "go test ./internal/.../adapter/... ./cmd/... -race -count=1",
			want:   "./internal/.../adapter/...,./cmd/...",
		},
		{
			name:   "flags before the pattern",
			recipe: "go test -race -count=1 ./tools/...",
			want:   "./tools/...",
		},
		{
			name:   "gotestsum passes patterns after the -- separator",
			recipe: "gotestsum --junitfile build/junit.xml -- -race -tags=integration ./...",
			want:   "./...",
		},
		{
			name:   "a recipe prefixed with @ still counts",
			recipe: "@go test ./internal/platform/... -count=1",
			want:   "./internal/platform/...",
		},
		{
			name:   "go vet is not go test",
			recipe: "go vet -tags=integration ./...",
			want:   "",
		},
		{
			name:   "go build is not go test",
			recipe: "go build ./...",
			want:   "",
		},
		{
			name:   "go run is not go test",
			recipe: "go run ./cmd/vulngate -baseline security/vuln-baseline.json",
			want:   "",
		},
		{
			name:   "a -run filter value is not mistaken for a pattern",
			recipe: "go test -run TestFoo ./internal/booking/app",
			want:   "./internal/booking/app",
		},
		{
			name:   "npm test is not go test",
			recipe: "npm --prefix web run test",
			want:   "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := goTestPatterns(tc.recipe)
			if strings.Join(got, ",") != tc.want {
				t.Errorf("goTestPatterns(%q) = %v, want %q", tc.recipe, got, tc.want)
			}
		})
	}
}

func TestExecutedPatternsWalksTheGateClosure(t *testing.T) {
	const src = "" +
		"test-domain:\n" +
		"\tgo test ./internal/.../domain/... -race -count=1\n" +
		"test-tools:\n" +
		"\tgo test ./tools/... -race -count=1\n" +
		"generate:\n" +
		"\tbuf generate\n" +
		"lint:\n" +
		"\tgolangci-lint run ./...\n" +
		"ci-checks: generate lint test-domain test-tools\n" +
		"\tgo build ./...\n" +
		"detached:\n" +
		"\tgo test ./internal/orphan/... -count=1\n"

	mf, err := ParseMakefile(strings.NewReader(src))
	if err != nil {
		t.Fatalf("ParseMakefile: %v", err)
	}

	inv, err := mf.TestInvocations("ci-checks")
	if err != nil {
		t.Fatalf("TestInvocations: %v", err)
	}

	var got []string
	for _, i := range inv {
		got = append(got, i.Target+"="+strings.Join(i.Patterns, "+"))
	}
	want := "test-domain=./internal/.../domain/...,test-tools=./tools/..."
	if strings.Join(got, ",") != want {
		t.Errorf("TestInvocations(ci-checks) = %v, want %v", got, want)
	}

	// The load-bearing property: a target that runs `go test` but is NOT
	// reachable from the gate root contributes nothing. This is what makes
	// "some Makefile target somewhere runs it" insufficient — only the
	// standing gate counts.
	for _, i := range inv {
		if i.Target == "detached" {
			t.Errorf("a target unreachable from the gate root must not count as coverage")
		}
	}
}

// --- Source scan (side A) -------------------------------------------------

// writeTree materialises a synthetic module so the scanner can be exercised
// against shapes the real repo does not (yet) contain.
func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, body := range files {
		full := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", rel, err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	return root
}

func TestScanTests(t *testing.T) {
	root := writeTree(t, map[string]string{
		"go.mod": "module example.com/synthetic\n\ngo 1.25.0\n",

		// Plain, runnable tests.
		"internal/plain/plain.go":      "package plain\n",
		"internal/plain/plain_test.go": "package plain\n\nimport \"testing\"\n\nfunc TestOne(t *testing.T) {}\nfunc TestTwo(t *testing.T) {}\n",

		// Only build-tagged tests: compiled by vet-integration, executed by
		// nothing — the deliberately-accepted middle state.
		"internal/tagged/tagged.go":                 "package tagged\n",
		"internal/tagged/thing_integration_test.go": "//go:build integration\n\npackage tagged\n\nimport \"testing\"\n\nfunc TestTagged(t *testing.T) {}\n",

		// Both: the package is runnable because at least one unconstrained
		// test func exists, even though it also holds tagged ones.
		"internal/mixed/mixed.go":                  "package mixed\n",
		"internal/mixed/unit_test.go":              "package mixed\n\nimport \"testing\"\n\nfunc TestUnit(t *testing.T) {}\n",
		"internal/mixed/heavy_integration_test.go": "//go:build integration\n\npackage mixed\n\nimport \"testing\"\n\nfunc TestHeavy(t *testing.T) {}\n",

		// Production code with no test files at all.
		"internal/untested/untested.go": "package untested\n",

		// Non-test funcs whose names merely begin with the letters "Test",
		// plus TestMain, which is a harness hook and not a test.
		"internal/lookalike/lookalike.go":      "package lookalike\n\nfunc Testing() {}\n",
		"internal/lookalike/lookalike_test.go": "package lookalike\n\nimport (\n\t\"os\"\n\t\"testing\"\n)\n\nfunc TestMain(m *testing.M) { os.Exit(m.Run()) }\nfunc Testify() {}\nfunc testHelper() {}\n",

		// An external test package (package foo_test) counts for its dir.
		"internal/external/external.go":      "package external\n",
		"internal/external/external_test.go": "package external_test\n\nimport \"testing\"\n\nfunc TestExternal(t *testing.T) {}\n",

		// go list ignores testdata/, _underscore and .dot dirs — so must the
		// scan, or side A would report packages side B can never contain.
		"internal/plain/testdata/fixture_test.go": "package fixture\n\nimport \"testing\"\n\nfunc TestFixture(t *testing.T) {}\n",
		"_scratch/scratch_test.go":                "package scratch\n\nimport \"testing\"\n\nfunc TestScratch(t *testing.T) {}\n",
		".hidden/hidden_test.go":                  "package hidden\n\nimport \"testing\"\n\nfunc TestHidden(t *testing.T) {}\n",
		"node_modules/pkg/pkg_test.go":            "package pkg\n\nimport \"testing\"\n\nfunc TestNodeModules(t *testing.T) {}\n",
	})

	pkgs, err := ScanTests(root)
	if err != nil {
		t.Fatalf("ScanTests: %v", err)
	}

	byDir := map[string]TestPackage{}
	for _, p := range pkgs {
		byDir[p.Dir] = p
	}

	tests := []struct {
		name            string
		dir             string
		wantPresent     bool
		wantRunnable    []string
		wantConstrained []string
	}{
		{
			name:         "unconstrained test funcs are runnable",
			dir:          "internal/plain",
			wantPresent:  true,
			wantRunnable: []string{"TestOne", "TestTwo"},
		},
		{
			name:            "a build-tagged-only package is constrained, never runnable",
			dir:             "internal/tagged",
			wantPresent:     true,
			wantConstrained: []string{"TestTagged"},
		},
		{
			name:            "a mixed package is runnable and records its tagged tests too",
			dir:             "internal/mixed",
			wantPresent:     true,
			wantRunnable:    []string{"TestUnit"},
			wantConstrained: []string{"TestHeavy"},
		},
		{
			name:         "an external _test package counts for its directory",
			dir:          "internal/external",
			wantPresent:  true,
			wantRunnable: []string{"TestExternal"},
		},
		{name: "a package with no test files is absent", dir: "internal/untested"},
		{name: "TestMain and Test-prefixed non-tests do not make a package", dir: "internal/lookalike"},
		{name: "testdata is skipped, exactly as go list skips it", dir: "internal/plain/testdata"},
		{name: "_underscore dirs are skipped", dir: "_scratch"},
		{name: "dot dirs are skipped", dir: ".hidden"},
		{name: "node_modules is skipped", dir: "node_modules/pkg"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := byDir[tc.dir]
			if ok != tc.wantPresent {
				t.Fatalf("dir %q present = %v, want %v (scanned: %v)", tc.dir, ok, tc.wantPresent, byDir)
			}
			if !ok {
				return
			}
			if strings.Join(got.RunnableTests, ",") != strings.Join(tc.wantRunnable, ",") {
				t.Errorf("runnable = %v, want %v", got.RunnableTests, tc.wantRunnable)
			}
			if strings.Join(got.ConstrainedTests, ",") != strings.Join(tc.wantConstrained, ",") {
				t.Errorf("constrained = %v, want %v", got.ConstrainedTests, tc.wantConstrained)
			}
		})
	}
}

// --- The diff (sides A vs B) ---------------------------------------------

func TestAnalyze(t *testing.T) {
	pkgs := []TestPackage{
		{Dir: "internal/booking/domain", RunnableTests: []string{"TestA"}},
		{Dir: "internal/booking/adapter/postgres", RunnableTests: []string{"TestB"}, ConstrainedTests: []string{"TestC"}},
		{Dir: "internal/socialplay/adapter/grpcapi", RunnableTests: []string{"TestD"}},
		{Dir: "internal/onlytagged", ConstrainedTests: []string{"TestE"}},
	}

	tests := []struct {
		name             string
		gated            []string
		wantOK           bool
		wantUngated      []string
		wantCompiledOnly []string
	}{
		{
			name:             "every runnable package gated is a pass",
			gated:            []string{"internal/booking/domain", "internal/booking/adapter/postgres", "internal/socialplay/adapter/grpcapi"},
			wantOK:           true,
			wantCompiledOnly: []string{"internal/onlytagged"},
		},
		{
			name:             "a runnable package no gate executes fails and is named",
			gated:            []string{"internal/booking/domain"},
			wantOK:           false,
			wantUngated:      []string{"internal/booking/adapter/postgres", "internal/socialplay/adapter/grpcapi"},
			wantCompiledOnly: []string{"internal/onlytagged"},
		},
		{
			name:             "gating the tagged-only package does not change its classification",
			gated:            []string{"internal/booking/domain", "internal/booking/adapter/postgres", "internal/socialplay/adapter/grpcapi", "internal/onlytagged"},
			wantOK:           true,
			wantCompiledOnly: []string{"internal/onlytagged"},
		},
		{
			name:             "no gates at all fails with every runnable package named",
			gated:            nil,
			wantOK:           false,
			wantUngated:      []string{"internal/booking/adapter/postgres", "internal/booking/domain", "internal/socialplay/adapter/grpcapi"},
			wantCompiledOnly: []string{"internal/onlytagged"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rep := Analyze(pkgs, tc.gated)
			if rep.OK() != tc.wantOK {
				t.Errorf("OK() = %v, want %v", rep.OK(), tc.wantOK)
			}
			if got := dirsOf(rep.Ungated); strings.Join(got, ",") != strings.Join(tc.wantUngated, ",") {
				t.Errorf("ungated = %v, want %v", got, tc.wantUngated)
			}
			if got := dirsOf(rep.CompiledOnly); strings.Join(got, ",") != strings.Join(tc.wantCompiledOnly, ",") {
				t.Errorf("compiled-only = %v, want %v", got, tc.wantCompiledOnly)
			}
		})
	}
}

func dirsOf(f []Finding) []string {
	var out []string
	for _, x := range f {
		out = append(out, x.Dir)
	}
	return out
}

// TestReportNamesTheOffendingPackage is the mutation check in test form: the
// failure output has to be actionable on its own, because the whole point of
// this gate is that whoever trips it should not need to re-derive the answer.
func TestReportNamesTheOffendingPackage(t *testing.T) {
	rep := Analyze([]TestPackage{
		{Dir: "internal/booking/domain", RunnableTests: []string{"TestA"}},
		{Dir: "internal/facilities/adapter/identity", RunnableTests: []string{"TestOwnerOf", "TestOwnerOfMissing"}},
	}, []string{"internal/booking/domain"})

	out := rep.String()

	for _, want := range []string{
		"FAIL",
		"internal/facilities/adapter/identity",
		"2 test func",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("failure output missing %q; got:\n%s", want, out)
		}
	}
	if strings.Contains(out, "internal/booking/domain") {
		t.Errorf("a covered package must not be listed as a failure; got:\n%s", out)
	}
}

// TestPackageAddedAfterTheGateWasWritten is the ticket's own dual-coverage
// question (T14 plan A5), as a test. Three consecutive sprints shipped a gate
// whose package set was fixed at authoring time, and each was stale before the
// sprint ended — #157's own list went stale inside one sprint when T13.3
// created internal/facilities/adapter/identity.
//
// The property that has to hold is that a package created LATER, by a ticket
// nobody had written when this check was authored, is classified purely by
// what is on disk at run time. There is no list to update, so the only way for
// this to regress is for the scan itself to break — which the rest of these
// tests cover.
func TestPackageAddedAfterTheGateWasWritten(t *testing.T) {
	// The gate's globs, frozen as of "authoring time".
	gated := []string{
		"internal/booking/domain",
		"internal/booking/app",
	}

	before := []TestPackage{
		{Dir: "internal/booking/domain", RunnableTests: []string{"TestA"}},
		{Dir: "internal/booking/app", RunnableTests: []string{"TestB"}},
	}
	if rep := Analyze(before, gated); !rep.OK() {
		t.Fatalf("baseline should be green, got:\n%s", rep)
	}

	// A later sprint adds a package the frozen globs do not match — T13.3's
	// internal/facilities/adapter/identity is the real instance of this.
	after := append(before, TestPackage{
		Dir:           "internal/facilities/adapter/identity",
		RunnableTests: []string{"TestOwnerOf"},
	})
	rep := Analyze(after, gated)
	if rep.OK() {
		t.Fatalf("a package added after the gate was written must fail the check; got:\n%s", rep)
	}
	if dirs := dirsOf(rep.Ungated); strings.Join(dirs, ",") != "internal/facilities/adapter/identity" {
		t.Errorf("ungated = %v, want exactly the newly-added package", dirs)
	}
}

// --- The real repository --------------------------------------------------

// TestRepoMakefileIsStillParseable is the desync guard. Side B is derived from
// the Makefile rather than copied from it, so a Makefile restructure cannot
// silently desynchronise the check — but it COULD silently blind the parser,
// leaving side B empty. An empty side B fails loudly (every package becomes
// ungated) rather than passing vacuously, and this test says so up front.
func TestRepoMakefileIsStillParseable(t *testing.T) {
	f, err := os.Open(filepath.Join("..", "..", "Makefile"))
	if err != nil {
		t.Fatalf("open repo Makefile: %v", err)
	}
	defer f.Close()

	mf, err := ParseMakefile(f)
	if err != nil {
		t.Fatalf("ParseMakefile on the repo's own Makefile: %v", err)
	}

	inv, err := mf.TestInvocations(DefaultGateRoot)
	if err != nil {
		t.Fatalf("TestInvocations(%q): %v", DefaultGateRoot, err)
	}
	if len(inv) == 0 {
		t.Fatalf("no `go test` invocation is reachable from %q — either the gate lost its "+
			"test targets or the parser has been blinded by a Makefile restructure", DefaultGateRoot)
	}

	// Named deliberately: these are the targets whose absence from the gate
	// root would silently shrink side B. This is an assertion ABOUT the
	// Makefile, not a copy of it — side B is still derived, and a target
	// renamed here fails this test instead of quietly reducing coverage.
	seen := map[string]bool{}
	for _, i := range inv {
		seen[i.Target] = true
	}
	for _, want := range []string{"test-domain", "test-platform", "test-tools"} {
		if !seen[want] {
			t.Errorf("%q runs no `go test` reachable from %q (reachable: %v)", want, DefaultGateRoot, seen)
		}
	}
}

// --- The zero-test blind spot (#328, decided at T65.4) --------------------

// TestUntestedNamesPackagesHoldingNoTestFunction is the pure half of the
// decision #328 asked for. This check's question is "which tests does no gate
// run?", and a package with no tests has none to run — so it was invisible
// here by construction, and `gate-coverage: OK` was silent about four of five
// `cmd` packages. The decision was to REPORT them, never to fail on them: a
// gate that failed would be satisfied by a stub test, which is worse than the
// silence it replaced.
func TestUntestedNamesPackagesHoldingNoTestFunction(t *testing.T) {
	all := []string{
		"cmd/devtoken",
		"cmd/docsindex",
		"cmd/server",
		"internal/booking/domain",
		"internal/platform/pg",
	}
	pkgs := []TestPackage{
		{Dir: "cmd/docsindex", RunnableTests: []string{"TestA"}},
		{Dir: "cmd/server", RunnableTests: []string{"TestB"}},
		{Dir: "internal/booking/domain", RunnableTests: []string{"TestC"}},
	}

	got := Untested(all, pkgs)
	want := []string{"cmd/devtoken", "internal/platform/pg"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("Untested() = %v, want %v", got, want)
	}
}

// TestUntestedCountsATaggedOnlyPackageAsTested: a package whose every test is
// behind a build tag HAS tests — it is the CompiledOnly case, which this check
// already reports under its own heading. Double-reporting it would make the
// new note's count meaningless.
func TestUntestedCountsATaggedOnlyPackageAsTested(t *testing.T) {
	got := Untested(
		[]string{"internal/onlytagged"},
		[]TestPackage{{Dir: "internal/onlytagged", ConstrainedTests: []string{"TestE"}}},
	)
	if len(got) != 0 {
		t.Fatalf("Untested() = %v, want none — a tagged-only package is CompiledOnly, not untested", got)
	}
}

// TestUntestedIsSortedAndDeduplicated: the output is read by humans and
// compared run to run, so churn in it would be indistinguishable from a real
// change.
func TestUntestedIsSortedAndDeduplicated(t *testing.T) {
	got := Untested([]string{"z/pkg", "a/pkg", "z/pkg", "m/pkg"}, nil)
	if want := "a/pkg,m/pkg,z/pkg"; strings.Join(got, ",") != want {
		t.Fatalf("Untested() = %v, want %v", got, want)
	}
}

// TestUntestedPackagesAreReportedAndDoNotFailTheCheck is the decision itself,
// asserted on the rendered report: the packages are named, the output says in
// so many words that this is not a failure, and OK() is unaffected.
func TestUntestedPackagesAreReportedAndDoNotFailTheCheck(t *testing.T) {
	rep := Analyze([]TestPackage{
		{Dir: "internal/booking/domain", RunnableTests: []string{"TestA"}},
	}, []string{"internal/booking/domain"})
	rep.GateRoot = "ci-checks"
	rep.Untested = []string{"cmd/devtoken", "cmd/vulngate"}

	if !rep.OK() {
		t.Fatalf("an untested package must not fail the check; got:\n%s", rep)
	}

	out := rep.String()
	for _, want := range []string{
		"no test function at all",
		"cmd/devtoken",
		"cmd/vulngate",
		"2 package(s)",
		"OK",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("report missing %q; got:\n%s", want, out)
		}
	}
	if strings.Contains(out, "FAIL") {
		t.Fatalf("the untested note must not read as a failure; got:\n%s", out)
	}
}

// TestNoUntestedPackagesPrintsNoNote: a note that prints "0 package(s)" every
// run is noise, and noise is what gets skimmed.
func TestNoUntestedPackagesPrintsNoNote(t *testing.T) {
	rep := Analyze([]TestPackage{
		{Dir: "internal/booking/domain", RunnableTests: []string{"TestA"}},
	}, []string{"internal/booking/domain"})
	rep.GateRoot = "ci-checks"

	if out := rep.String(); strings.Contains(out, "no test function at all") {
		t.Fatalf("the note printed with nothing to report:\n%s", out)
	}
}

// TestRunReportsTheRealRepositorysUntestedPackages is the end-to-end half,
// against this checkout. #328's premise — four of five `cmd` packages hold
// zero tests — was true when filed; T65.4 tested one of them, so the figure
// the test asserts is derived here rather than written down.
func TestRunReportsTheRealRepositorysUntestedPackages(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	rep, err := Run(root, filepath.Join(root, "Makefile"), DefaultGateRoot, GoList(root))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(rep.Untested) == 0 {
		t.Fatalf("this repository has packages with no tests (cmd/devtoken for one) and none were reported")
	}
	var found bool
	for _, d := range rep.Untested {
		if d == "cmd/devtoken" {
			found = true
		}
		for _, p := range rep.Scanned {
			if p.Dir == d {
				t.Fatalf("%s is reported as untested and also holds test functions", d)
			}
		}
	}
	if !found {
		t.Fatalf("cmd/devtoken holds no test function and is not in Untested = %v", rep.Untested)
	}
	if !rep.OK() {
		t.Fatalf("the real repository must still pass:\n%s", rep)
	}
}

// TestSplitGeneratedUsesGosOwnMarkerRatherThanAPathList is the other half of
// T65.4's decision. The real repository has 27 packages with no tests and 13
// of them are `internal/gen/**`, which is generated and gitignored (CLAUDE.md
// rule 6: never hand-edit it). Printing those alongside the hand-written ones
// buries the list a reader can act on under one they cannot.
//
// The split is derived from Go's own documented generated-file marker — a
// property of the files — and NOT from a path prefix. A path list in this tool
// is the single change that would defeat it, and "internal/gen" would be one.
func TestSplitGeneratedUsesGosOwnMarkerRatherThanAPathList(t *testing.T) {
	root := writeTree(t, map[string]string{
		"gen/sqlc/db.go":    "// Code generated by sqlc. DO NOT EDIT.\n\npackage sqlcgen\n",
		"gen/proto/x.pb.go": "// Code generated by protoc-gen-go. DO NOT EDIT.\n// versions:\n\npackage protogen\n",
		"handwritten/a.go":  "package handwritten\n\nfunc A() {}\n",
		"mixed/gen.go":      "// Code generated by thing. DO NOT EDIT.\n\npackage mixed\n",
		"mixed/hand.go":     "package mixed\n\nfunc B() {}\n",
		"mentions/m.go":     "package mentions\n\n// This file talks about Code generated by sqlc. DO NOT EDIT. in prose.\nfunc C() {}\n",
		"notatop/n.go":      "package notatop\n\n// Code generated by sqlc. DO NOT EDIT.\nfunc D() {}\n",
	})

	hand, generated := SplitGenerated(root, []string{
		"gen/proto", "gen/sqlc", "handwritten", "mentions", "mixed", "notatop",
	})

	if want := "gen/proto,gen/sqlc,mixed"; strings.Join(generated, ",") != want {
		t.Fatalf("generated = %v, want %v", generated, want)
	}
	if want := "handwritten,mentions,notatop"; strings.Join(hand, ",") != want {
		t.Fatalf("handWritten = %v, want %v — the marker must be a header, not a mention", hand, want)
	}
}

// TestSplitGeneratedReportsAnUnreadableDirectoryAsHandWritten: the
// conservative direction. A directory this cannot read is one it knows nothing
// about, and the note is supposed to surface packages, not hide them.
func TestSplitGeneratedReportsAnUnreadableDirectoryAsHandWritten(t *testing.T) {
	hand, generated := SplitGenerated(t.TempDir(), []string{"does/not/exist"})
	if strings.Join(hand, ",") != "does/not/exist" || len(generated) != 0 {
		t.Fatalf("hand = %v, generated = %v; want the unreadable directory reported", hand, generated)
	}
}

// TestTheGeneratedNoteIsACountAndNotAList: thirteen generated package names
// every run is noise, and noise is what gets skimmed. The hand-written ones
// are named; the generated ones are counted.
func TestTheGeneratedNoteIsACountAndNotAList(t *testing.T) {
	rep := Analyze([]TestPackage{
		{Dir: "internal/booking/domain", RunnableTests: []string{"TestA"}},
	}, []string{"internal/booking/domain"})
	rep.GateRoot = "ci-checks"
	rep.Untested = []string{"cmd/devtoken"}
	rep.UntestedGenerated = []string{"internal/gen/bookingdb", "internal/gen/identitydb"}

	out := rep.String()
	if !strings.Contains(out, "cmd/devtoken") {
		t.Fatalf("the hand-written package is not named:\n%s", out)
	}
	if !strings.Contains(out, "2 generated") {
		t.Fatalf("the generated packages are not counted:\n%s", out)
	}
	if strings.Contains(out, "internal/gen/bookingdb") {
		t.Fatalf("a generated package is listed by name:\n%s", out)
	}
	if !rep.OK() {
		t.Fatalf("neither note may fail the check:\n%s", out)
	}
}

// TestRunSplitsTheRealRepositorysUntestedPackages asserts the PROPERTY, not
// the tree.
//
// Its first version required every generated-side entry to start with
// "internal/gen/", which T65.4's review correctly called the path list the
// tool is forbidden to have, relocated into its gate: a correctly-marked
// `wire`, `mockgen` or `stringer` package anywhere else turned `ci-checks`
// red for doing its job, and the fix a session under time pressure reaches
// for is editing the hardcoded prefix. So the assertion is now the one the
// implementation claims — every reported package is on the side its own
// files put it on — checked by re-reading those files here rather than by
// calling the function under test.
func TestRunSplitsTheRealRepositorysUntestedPackages(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	rep, err := Run(root, filepath.Join(root, "Makefile"), DefaultGateRoot, GoList(root))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	for _, d := range rep.Untested {
		if marked, _ := anyFileCarriesTheMarker(t, filepath.Join(root, d)); marked {
			t.Errorf("%s is reported as hand-written but one of its files carries the "+
				"generated-code marker", d)
		}
	}
	for _, d := range rep.UntestedGenerated {
		marked, files := anyFileCarriesTheMarker(t, filepath.Join(root, d))
		if !marked {
			t.Errorf("%s is reported as generated but none of its %d Go file(s) carries the marker",
				d, files)
		}
	}

	// The generated side being non-empty depends on `internal/gen` existing,
	// which is gitignored — and `test-tools` has no `generate` prerequisite,
	// so on a fresh clone this test runs before any codegen. T65.4's review
	// found that version red on a clean checkout, blaming the header scan
	// for a missing directory. Skipped with the reason rather than asserted.
	if len(rep.UntestedGenerated) == 0 {
		if _, statErr := os.Stat(filepath.Join(root, "internal", "gen")); os.IsNotExist(statErr) {
			t.Skip("internal/gen is absent (run `make generate`); nothing in this tree is generated, " +
				"so there is no generated side to check")
		}
		t.Errorf("internal/gen exists and no package in it was recognised as generated; " +
			"sqlc and protoc both emit Go's DO NOT EDIT header")
	}
}

// anyFileCarriesTheMarker re-reads a package directory and reports whether
// any .go file directly in it carries Go's generated-code header, plus how
// many it looked at. Deliberately a second, independent reading: asserting
// SplitGenerated's output by calling SplitGenerated would assert nothing.
func anyFileCarriesTheMarker(t *testing.T, dir string) (bool, int) {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, 0
	}
	seen := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		seen++
		src, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		for _, line := range strings.Split(strings.TrimPrefix(string(src), "\ufeff"), "\n") {
			line = strings.TrimRight(line, "\r")
			if strings.HasPrefix(line, "package ") {
				break
			}
			if strings.HasPrefix(line, "// Code generated ") && strings.HasSuffix(line, "DO NOT EDIT.") {
				return true, seen
			}
		}
	}
	return false, seen
}

// TestSplitGeneratedHandlesCRLFAndABOM: the marker regexp anchors on
// "EDIT.$", so a trailing \r defeats it and a CRLF-terminated generated file
// read as hand-written. Found by T65.4's review. Low real-world risk here —
// codegen runs on Linux into a gitignored directory — and one TrimRight.
func TestSplitGeneratedHandlesCRLFAndABOM(t *testing.T) {
	root := writeTree(t, map[string]string{
		"crlf/db.go": "// Code generated by sqlc. DO NOT EDIT.\r\n\r\npackage crlf\r\n",
		"bom/db.go":  "\ufeff// Code generated by sqlc. DO NOT EDIT.\n\npackage bom\n",
		"tagged/x.go": "//go:build !ignore\n\n// Code generated by protoc-gen-go. DO NOT EDIT.\n\n" +
			"package tagged\n",
		"hand/a.go": "package hand\n",
	})

	hand, generated := SplitGenerated(root, []string{"bom", "crlf", "hand", "tagged"})
	if want := "bom,crlf,tagged"; strings.Join(generated, ",") != want {
		t.Fatalf("generated = %v, want %v", generated, want)
	}
	if want := "hand"; strings.Join(hand, ",") != want {
		t.Fatalf("handWritten = %v, want %v", hand, want)
	}
}

// TestRunIgnoresSkippedDirectories: `go list ./...` does not honour
// ScanTests' skip rule, so a Go package inside node_modules would land in
// this project's own untested list. T65.4's review planted one and watched
// it appear.
func TestRunIgnoresSkippedDirectories(t *testing.T) {
	// Driven through Run, not through withoutSkipped alone: testing the
	// helper in isolation left "Run stops calling it" a surviving mutant.
	// The listing is injected, so this needs no planted directory on disk.
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	realLister := GoList(root)
	planted := "web/node_modules/some-pkg/go"
	withPlant := func(patterns []string) ([]string, error) {
		dirs, err := realLister(patterns)
		if err != nil {
			return nil, err
		}
		if len(patterns) == 1 && patterns[0] == "./..." {
			dirs = append(dirs, planted, "_scratch/pkg", "internal/vendor/thing")
		}
		return dirs, nil
	}

	rep, err := Run(root, filepath.Join(root, "Makefile"), DefaultGateRoot, Lister(withPlant))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	for _, d := range append(append([]string{}, rep.Untested...), rep.UntestedGenerated...) {
		for _, bad := range []string{planted, "_scratch/pkg", "internal/vendor/thing"} {
			if d == bad {
				t.Fatalf("Run reported %q, which the test scan's own skip rule excludes", d)
			}
		}
	}

	for _, dir := range []string{
		"web/node_modules/some-pkg/go",
		"internal/vendor/thing",
		"_scratch/pkg",
		".tooling/pkg",
		"internal/booking/testdata/pkg",
	} {
		if got := withoutSkipped([]string{"internal/booking/domain", dir}); len(got) != 1 {
			t.Errorf("withoutSkipped kept %q: got %v", dir, got)
		}
	}
	if got := withoutSkipped([]string{"cmd/server", "internal/gen/bookingdb"}); len(got) != 2 {
		t.Errorf("withoutSkipped dropped a real package: %v", got)
	}
}

// TestRunFailsWhenTheModuleCannotBeListed is the one new guard T65.4's
// review found untested: dropping Run's error check left the suite green,
// and a failing `./...` listing would then empty Untested, remove the NOTE,
// and still print OK — the vacuous green this tool exists to prevent, in
// the branch that reports it.
func TestRunFailsWhenTheModuleCannotBeListed(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	realLister := GoList(root)
	failing := func(patterns []string) ([]string, error) {
		if len(patterns) == 1 && patterns[0] == "./..." {
			return nil, errors.New("go list refused")
		}
		return realLister(patterns)
	}

	_, err = Run(root, filepath.Join(root, "Makefile"), DefaultGateRoot, Lister(failing))
	if err == nil {
		t.Fatalf("err = nil when the module could not be listed; the report would print OK with no NOTE")
	}
	if !strings.Contains(err.Error(), "listing every package") {
		t.Fatalf("err = %v, want it to say which step failed", err)
	}
}
