// Package gatecoverage answers one question mechanically, for the whole
// module, every time it runs: **which packages hold tests that no gate
// executes?**
//
// # Why this exists
//
// Three consecutive sprints found the same class of hole, and each fix closed
// only the instance someone happened to trip over (docs/process/t13-retro.md
// finding 2):
//
//	T11 | //go:build integration files ran in no gate | T12.1's vet-integration COMPILES them | still never executed
//	T12 | internal/platform/**, incl. the auth spine  | T13.4's test-platform                 | left 22 adapter packages
//	T13 | 22 internal/*/adapter/* packages (#157)     | this check                            | -
//
// Every one of those fixes was a hand-written glob covering exactly the
// packages named in that sprint's issue. #157's own list went stale inside a
// single sprint: T13.3 created internal/facilities/adapter/identity after #157
// was filed, and any gate built from #157's list would have shipped already
// missing a package its own sprint sibling created.
//
// So the deliberate design constraint, from the T14 plan's A5, is that **both
// sides of the diff are computed at run time and neither is a list**:
//
//   - Side A — packages holding tests — is a source scan of the module. A
//     package created next sprint appears in it with no edit here.
//   - Side B — packages a gate executes — is derived from the Makefile itself:
//     the transitive target closure of the gate root (`ci-checks`), filtered to
//     the `go test` invocations inside it, with the resulting patterns expanded
//     by `go list`. A gate whose pattern changes cannot desynchronise from this
//     check, because this check reads that pattern rather than a copy of it.
//
// # The three states, kept distinct
//
// Conflating these is what turns this check back into a fourth glob widening:
//
//	OK      executed by a Docker-free gate.
//	NOTE    compiled but never executed — the //go:build integration files.
//	        make vet-integration typechecks them; executing them needs Docker.
//	        This is T11's finding, known and deliberately accepted, so it is
//	        REPORTED and does not fail.
//	FAIL    executed by nothing at all. The state this check exists to catch.
//
// A package whose only test funcs sit behind a build tag is NOTE, not FAIL:
// no Docker-free gate can run it, and pretending otherwise would either fail
// the build forever or require an exclusion list — the exact artifact this
// package refuses to have.
package gatecoverage

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

// DefaultGateRoot is the Makefile target treated as "the standing gate": the
// one command CI runs and a PR is judged by. Rooting the walk here — rather
// than scanning every target in the file — is what makes `make test` (which
// runs everything but hard-requires a Docker daemon) correctly NOT count as
// coverage, without naming it anywhere.
const DefaultGateRoot = "ci-checks"

// --- Side A: the source scan ---------------------------------------------

// TestPackage is one directory holding at least one test function.
//
// The unit is the directory, not the Go package name, because that is the unit
// `go test` patterns select — an external `package foo_test` in the same
// directory is executed by the same invocation as `package foo`.
type TestPackage struct {
	// Dir is slash-separated and relative to the module root.
	Dir string
	// RunnableTests are test funcs in files the default build context
	// selects — i.e. tests a plain `go test` would execute.
	RunnableTests []string
	// ConstrainedTests are test funcs in files build constraints exclude
	// from the default context (in this repo: //go:build integration).
	ConstrainedTests []string
}

// skipDir reports directories the scan must not descend into.
//
// The first three mirror `go list ./...` exactly (it ignores testdata and any
// element beginning with "." or "_"). That correspondence is load-bearing: side
// B is produced by `go list`, so a directory side A counts but `go list` can
// never return would be reported as permanently ungated with no possible fix.
// node_modules is a scan-cost guard for the Vue app, which holds no Go.
func skipDir(name string) bool {
	switch name {
	case "testdata", "node_modules", "vendor":
		return true
	}
	return strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_")
}

// isTestFunc applies the go test naming rule: a test is `func TestXxx(*testing.T)`
// where Xxx does not begin with a lower-case letter. TestMain is excluded — it
// is the harness hook, and a package holding only TestMain executes no tests.
func isTestFunc(fn *ast.FuncDecl) bool {
	if fn.Recv != nil || fn.Name == nil {
		return false
	}
	name := fn.Name.Name
	if name == "TestMain" || !strings.HasPrefix(name, "Test") {
		return false
	}
	rest := []rune(name[len("Test"):])
	if len(rest) > 0 && unicode.IsLower(rest[0]) {
		return false
	}
	// A test takes exactly one parameter (*testing.T).
	return fn.Type.Params != nil && len(fn.Type.Params.List) == 1
}

// ScanTests walks the module rooted at root and returns every directory
// holding at least one test function, sorted by directory.
func ScanTests(root string) ([]TestPackage, error) {
	ctx := build.Default
	ctx.UseAllFiles = false

	byDir := map[string]*TestPackage{}
	fset := token.NewFileSet()

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != root && skipDir(d.Name()) {
				return fs.SkipDir
			}
			return nil
		}
		name := d.Name()
		if !strings.HasSuffix(name, "_test.go") {
			return nil
		}

		dir := filepath.Dir(path)

		// MatchFile is go/build's own answer to "would the default build
		// context select this file?" — build constraints and GOOS/GOARCH
		// name suffixes alike. Using it, rather than hand-parsing
		// //go:build lines, means this check classifies a file exactly as
		// the toolchain does.
		selected, matchErr := ctx.MatchFile(dir, name)
		if matchErr != nil {
			// A file go/build cannot read is not evidence of coverage.
			// Treat it as constrained-out and keep going rather than
			// failing the whole scan on one unparseable file.
			selected = false
		}

		f, parseErr := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if parseErr != nil {
			return fmt.Errorf("parse %s: %w", path, parseErr)
		}

		var names []string
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if ok && isTestFunc(fn) {
				names = append(names, fn.Name.Name)
			}
		}
		if len(names) == 0 {
			return nil
		}

		rel, relErr := filepath.Rel(root, dir)
		if relErr != nil {
			return relErr
		}
		key := filepath.ToSlash(rel)

		pkg := byDir[key]
		if pkg == nil {
			pkg = &TestPackage{Dir: key}
			byDir[key] = pkg
		}
		if selected {
			pkg.RunnableTests = append(pkg.RunnableTests, names...)
		} else {
			pkg.ConstrainedTests = append(pkg.ConstrainedTests, names...)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	out := make([]TestPackage, 0, len(byDir))
	for _, p := range byDir {
		sort.Strings(p.RunnableTests)
		sort.Strings(p.ConstrainedTests)
		out = append(out, *p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Dir < out[j].Dir })
	return out, nil
}

// --- Side B: the Makefile ------------------------------------------------

// Rule is one Makefile rule: its prerequisites and its recipe lines.
type Rule struct {
	Name    string
	Prereqs []string
	Recipe  []string
}

// Makefile is the subset of make's grammar this check needs: rules, their
// prerequisites, and their recipes. Variable expansion is deliberately NOT
// implemented — see goTestPatterns.
type Makefile struct {
	rules map[string]*Rule
}

// Rule returns the named rule.
func (m *Makefile) Rule(name string) (*Rule, bool) {
	r, ok := m.rules[name]
	return r, ok
}

// assignmentOps are the operators that make a `name OP value` line a variable
// assignment rather than a rule. `:=` in particular starts with the same colon
// a rule does.
var assignmentOps = []string{":=", "::=", ":::=", "+=", "?=", "!="}

// isDirective reports lines that appear at column 0 inside a recipe without
// ending it — make's conditionals. Treating them as rule-terminators would
// silently truncate a recipe and lose the `go test` line after an `else`.
func isDirective(line string) bool {
	word, _, _ := strings.Cut(strings.TrimSpace(line), " ")
	switch word {
	case "ifeq", "ifneq", "ifdef", "ifndef", "else", "endif", "include", "-include", "export", "unexport", "define", "endef":
		return true
	}
	return false
}

// ParseMakefile reads the rules out of a Makefile.
func ParseMakefile(r io.Reader) (*Makefile, error) {
	mf := &Makefile{rules: map[string]*Rule{}}

	var (
		lines   []string
		pending strings.Builder
	)
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := sc.Text()
		// Join backslash continuations before anything else — the repo's
		// own .PHONY line is continued, and a parser that missed that would
		// read the second half as a rule.
		if strings.HasSuffix(line, `\`) {
			pending.WriteString(strings.TrimSuffix(line, `\`))
			pending.WriteString(" ")
			continue
		}
		if pending.Len() > 0 {
			pending.WriteString(line)
			line = pending.String()
			pending.Reset()
		}
		lines = append(lines, line)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if pending.Len() > 0 {
		lines = append(lines, pending.String())
	}

	var current *Rule
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "\t"):
			if current != nil {
				if body := strings.TrimSpace(line); body != "" {
					current.Recipe = append(current.Recipe, body)
				}
			}
			continue
		case strings.TrimSpace(line) == "":
			continue
		case strings.HasPrefix(strings.TrimSpace(line), "#"):
			continue
		case isDirective(line):
			// Keeps `current` alive: an ifeq/else/endif around a recipe is
			// still that recipe.
			continue
		}

		head, tail, ok := strings.Cut(line, ":")
		if !ok {
			current = nil
			continue
		}
		isAssignment := false
		for _, op := range assignmentOps {
			if strings.HasPrefix(strings.TrimSpace(line[len(head):]), op) {
				isAssignment = true
				break
			}
		}
		if isAssignment || strings.HasPrefix(strings.TrimSpace(tail), "=") {
			current = nil
			continue
		}

		current = nil
		for _, name := range strings.Fields(head) {
			rule := mf.rules[name]
			if rule == nil {
				rule = &Rule{Name: name}
				mf.rules[name] = rule
			}
			rule.Prereqs = append(rule.Prereqs, strings.Fields(tail)...)
			// A multi-target rule shares one recipe; the last name wins as
			// the recipe's owner, which is enough for this check because it
			// only ever reads recipes of targets it reached by name.
			current = rule
		}
	}
	return mf, nil
}

// makeRecursionTarget extracts `foo` from a `$(MAKE) foo` recipe line.
func makeRecursionTarget(recipe string) []string {
	fields := strings.Fields(recipe)
	var out []string
	for i, f := range fields {
		if f == "$(MAKE)" || f == "${MAKE}" || f == "make" {
			for _, arg := range fields[i+1:] {
				if !strings.HasPrefix(arg, "-") && !strings.Contains(arg, "=") {
					out = append(out, arg)
				}
			}
		}
	}
	return out
}

// Reachable returns root plus every target reachable from it through
// prerequisites and `$(MAKE) x` recursion, sorted.
//
// An undefined root is an error rather than an empty set: a gate root that has
// been renamed must fail loudly, not quietly report that nothing is gated (or,
// worse, that everything is).
func (m *Makefile) Reachable(root string) ([]string, error) {
	if _, ok := m.rules[root]; !ok {
		return nil, fmt.Errorf("makefile defines no target %q", root)
	}
	seen := map[string]bool{}
	var walk func(string)
	walk = func(name string) {
		if seen[name] {
			return
		}
		rule, ok := m.rules[name]
		if !ok {
			return // a file prerequisite, not a target
		}
		seen[name] = true
		for _, p := range rule.Prereqs {
			walk(p)
		}
		for _, line := range rule.Recipe {
			for _, t := range makeRecursionTarget(line) {
				walk(t)
			}
		}
	}
	walk(root)

	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}
	sort.Strings(out)
	return out, nil
}

// TestInvocation is one `go test` command found inside the gate.
type TestInvocation struct {
	Target   string
	Recipe   string
	Patterns []string
}

// isPackagePattern reports whether a recipe token is a Go package pattern.
//
// Deliberately narrow: only relative patterns count. That is what this repo
// writes, and it means a flag VALUE that happens to follow a flag (`-run
// TestFoo`) can never be mistaken for a package. A pattern this misses is
// invisible to side B, which makes the check fail LOUDLY (its packages look
// ungated) rather than pass vacuously — the safe direction for a gate.
//
// The same asymmetry holds one level down: `go list -e` exits 0 and prints
// nothing for a pattern matching no directory (verified), so a gate pattern
// that has rotted contributes no coverage and its packages surface as
// ungated, rather than the rot passing unnoticed.
func isPackagePattern(tok string) bool {
	return tok == "." || strings.HasPrefix(tok, "./") || strings.HasPrefix(tok, "../")
}

// goTestPatterns pulls the package patterns out of one recipe line, or returns
// nil if the line does not execute Go tests.
//
// `go vet`, `go build` and `go run` are all `go <verb>` lines that must not
// count: vet-integration COMPILING a package is precisely the state this check
// distinguishes from executing it.
func goTestPatterns(recipe string) []string {
	line := strings.TrimLeft(recipe, "@-+")
	fields := strings.Fields(line)

	start := -1
	for i, f := range fields {
		if f == "go" && i+1 < len(fields) && fields[i+1] == "test" {
			start = i + 2
			break
		}
		// gotestsum is `go test` with a JUnit writer around it; its own
		// flags precede a `--` separator, after which the arguments are go
		// test's.
		if strings.HasSuffix(f, "gotestsum") {
			for j := i + 1; j < len(fields); j++ {
				if fields[j] == "--" {
					start = j + 1
					break
				}
			}
			break
		}
	}
	if start < 0 {
		return nil
	}

	var out []string
	for _, tok := range fields[start:] {
		if tok == ";" || tok == "&&" || tok == "|" || strings.HasPrefix(tok, ">") {
			break
		}
		if isPackagePattern(tok) {
			out = append(out, tok)
		}
	}
	return out
}

// TestInvocations returns every `go test` invocation reachable from the gate
// root, in target order.
func (m *Makefile) TestInvocations(root string) ([]TestInvocation, error) {
	targets, err := m.Reachable(root)
	if err != nil {
		return nil, err
	}
	var out []TestInvocation
	for _, name := range targets {
		rule := m.rules[name]
		for _, line := range rule.Recipe {
			if pats := goTestPatterns(line); len(pats) > 0 {
				out = append(out, TestInvocation{Target: name, Recipe: line, Patterns: pats})
			}
		}
	}
	return out, nil
}

// --- Expanding side B's patterns -----------------------------------------

// Lister expands Go package patterns to module-relative directories. It is an
// interface seam only so the diff logic is testable without a toolchain; the
// real implementation is GoList.
type Lister func(patterns []string) ([]string, error)

// GoList expands patterns with `go list`, which is the authoritative answer to
// what a pattern selects — reimplementing `...` matching here would be a second
// source of truth, and a wrong one.
//
// `-e` matters: internal/gen is gitignored (CLAUDE.md gotchas), so on a clean
// checkout the codegen-dependent packages do not type-check. Without -e, go
// list would fail and this check would report those packages as ungated for a
// reason that has nothing to do with gating. With -e they are still listed —
// which is correct, because `go test` would still ATTEMPT them, and whether
// they compile is `make generate`'s business, not this check's.
func GoList(moduleRoot string) Lister {
	return func(patterns []string) ([]string, error) {
		if len(patterns) == 0 {
			return nil, nil
		}
		args := append([]string{"list", "-e", "-f", "{{.Dir}}"}, patterns...)
		cmd := exec.Command("go", args...)
		cmd.Dir = moduleRoot
		out, err := cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("go list %s: %w", strings.Join(patterns, " "), err)
		}
		var dirs []string
		for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			rel, relErr := filepath.Rel(moduleRoot, line)
			if relErr != nil {
				continue
			}
			dirs = append(dirs, filepath.ToSlash(rel))
		}
		return dirs, nil
	}
}

// ExecutedDirs expands every gate invocation's patterns into the set of
// directories the gate actually runs tests in.
func ExecutedDirs(inv []TestInvocation, list Lister) ([]string, error) {
	seen := map[string]bool{}
	for _, i := range inv {
		dirs, err := list(i.Patterns)
		if err != nil {
			return nil, err
		}
		for _, d := range dirs {
			seen[d] = true
		}
	}
	out := make([]string, 0, len(seen))
	for d := range seen {
		out = append(out, d)
	}
	sort.Strings(out)
	return out, nil
}

// --- The diff ------------------------------------------------------------

// Finding is one package plus why it was flagged.
type Finding struct {
	Dir   string
	Tests []string
}

// Report is the outcome of diffing side A against side B.
type Report struct {
	// Scanned is every package holding tests (side A).
	Scanned []TestPackage
	// GateRoot and Invocations record how side B was derived, so the output
	// shows its own working.
	GateRoot    string
	Invocations []TestInvocation
	// Covered, Ungated and CompiledOnly are the three states, disjoint.
	Covered      []Finding
	Ungated      []Finding
	CompiledOnly []Finding
}

// Analyze diffs packages holding tests against the directories the gate runs.
func Analyze(pkgs []TestPackage, gatedDirs []string) Report {
	gated := make(map[string]bool, len(gatedDirs))
	for _, d := range gatedDirs {
		gated[d] = true
	}

	rep := Report{Scanned: pkgs}
	for _, p := range pkgs {
		switch {
		case len(p.RunnableTests) == 0:
			// Every test func is behind a build tag. No Docker-free gate
			// can execute it; vet-integration compiles it. Reported, never
			// failed — see the package doc.
			rep.CompiledOnly = append(rep.CompiledOnly, Finding{Dir: p.Dir, Tests: p.ConstrainedTests})
		case gated[p.Dir]:
			rep.Covered = append(rep.Covered, Finding{Dir: p.Dir, Tests: p.RunnableTests})
		default:
			rep.Ungated = append(rep.Ungated, Finding{Dir: p.Dir, Tests: p.RunnableTests})
		}
	}
	// Sorted so the report is byte-identical run to run regardless of the
	// order the scan happened to yield — a gate whose output churns is a
	// gate people learn to skim.
	for _, set := range [][]Finding{rep.Covered, rep.Ungated, rep.CompiledOnly} {
		sort.Slice(set, func(i, j int) bool { return set[i].Dir < set[j].Dir })
	}
	return rep
}

// OK reports whether every package holding runnable tests is executed by the
// gate.
func (r Report) OK() bool { return len(r.Ungated) == 0 }

// sample names a few of a package's tests. The package NAME is what makes the
// failure actionable; a grpcapi package holds 40-odd test funcs and printing
// all of them buries the list of packages under the list of tests.
func sample(names []string) string {
	const max = 3
	if len(names) <= max {
		return strings.Join(names, ", ")
	}
	return strings.Join(names[:max], ", ") + ", …"
}

// String renders the report. The failure half names every offending package
// and its test count, because the point of a standing gate is that whoever
// trips it can act on the message without re-deriving the analysis.
func (r Report) String() string {
	var b strings.Builder

	fmt.Fprintf(&b, "gate-coverage: %d package(s) hold test functions.\n", len(r.Scanned))
	fmt.Fprintf(&b, "gate-coverage: side B derived from the Makefile — %d `go test` invocation(s) reachable from %q:\n",
		len(r.Invocations), r.GateRoot)
	for _, i := range r.Invocations {
		fmt.Fprintf(&b, "    %-16s %s\n", i.Target, strings.Join(i.Patterns, " "))
	}

	if len(r.CompiledOnly) > 0 {
		fmt.Fprintf(&b, "\ngate-coverage: NOTE — %d package(s) hold ONLY build-tagged tests. `make vet-integration`\n", len(r.CompiledOnly))
		b.WriteString("  compiles these; executing them needs Docker (`make ci-integration`). Known and accepted:\n")
		for _, f := range r.CompiledOnly {
			fmt.Fprintf(&b, "    %s  (%d test func(s))\n", f.Dir, len(f.Tests))
		}
	}

	if len(r.Ungated) == 0 {
		fmt.Fprintf(&b, "\ngate-coverage: OK — all %d package(s) with runnable tests are executed by %q.\n",
			len(r.Covered), r.GateRoot)
		return b.String()
	}

	fmt.Fprintf(&b, "\ngate-coverage: FAIL — %d package(s) hold tests that NO gate executes:\n", len(r.Ungated))
	for _, f := range r.Ungated {
		fmt.Fprintf(&b, "    %s  (%d test func(s), e.g. %s)\n", f.Dir, len(f.Tests), sample(f.Tests))
	}
	b.WriteString("\n" +
		"These packages' tests exist but nothing runs them, so they cannot fail a build.\n" +
		"Fix by widening a `go test` pattern in a target reachable from " + r.GateRoot + " —\n" +
		"NOT by adding an exclusion here. See docs/process/t13-retro.md finding 2.\n")
	return b.String()
}

// Run performs the whole check: scan the module, derive the gate from the
// Makefile, expand, diff.
func Run(moduleRoot, makefilePath, gateRoot string, list Lister) (Report, error) {
	pkgs, err := ScanTests(moduleRoot)
	if err != nil {
		return Report{}, fmt.Errorf("scanning for test functions: %w", err)
	}

	f, err := os.Open(makefilePath)
	if err != nil {
		return Report{}, fmt.Errorf("opening the Makefile: %w", err)
	}
	defer f.Close()

	mf, err := ParseMakefile(f)
	if err != nil {
		return Report{}, fmt.Errorf("parsing %s: %w", makefilePath, err)
	}
	inv, err := mf.TestInvocations(gateRoot)
	if err != nil {
		return Report{}, err
	}
	dirs, err := ExecutedDirs(inv, list)
	if err != nil {
		return Report{}, err
	}

	rep := Analyze(pkgs, dirs)
	rep.GateRoot = gateRoot
	rep.Invocations = inv
	return rep, nil
}
