// Package mutationpin answers one question about a change, and only about the
// change:
//
//	Which guards this diff ADDS can be removed without any test noticing?
//
// A guard whose deletion nothing notices is not a guard. It is a comment that
// happens to compile. This repository has the incident to prove it: 0007 added
// a waitlist-promotion reservation at the database level with no test on
// either side of the boundary, 0012 rebuilt the function body from an older
// version and silently dropped it, and nothing failed for seventeen
// migrations. T61 found it by reading history, not by running anything.
//
// # Why diff-scoped
//
// Whole-repository mutation testing is the right idea at an impossible price:
// every mutant costs a test run, and this tree has thousands of candidate
// sites. Scoping to the lines a change ADDS keeps the cost proportional to the
// change and puts the finding where someone can still act on it -- in review,
// on code they just wrote, not in a backlog report about code from T19.
//
// That scoping is also the honest limit. This tool says nothing about existing
// code, so a clean run means "this diff adds no unpinned guard", never "this
// package's guards are pinned".
//
// # Two mutants per guard, and why both
//
// For each `if` statement on an added line:
//
//	AlwaysFalse  the guard never fires
//	AlwaysTrue   the guard always fires
//
// Both matter, and a survivor of each means something different. If
// AlwaysFalse survives, no test exercises the case the guard was written for.
// If AlwaysTrue survives, no test exercises the ordinary path past it. If BOTH
// survive, the condition is inert as far as the suite is concerned -- which is
// the shape of the T68 finding that a "guard" in tools/stalecitations was an
// optimization wearing a guard's name, and was deleted rather than tested.
//
// # What a surviving mutant is, and is not
//
// It is evidence that nothing fails when the guard is removed. It is NOT proof
// of a bug. Legitimate survivors exist and are common enough to make this
// advisory rather than a gate:
//
//   - defensive checks against states the type system already prevents;
//   - error branches for failures no test can provoke (a disk filling, a
//     context cancelled at one exact instant);
//   - a guard whose absence changes only performance.
//
// So the tool reports and exits 0, like tools/gatecoverage and
// tools/stalecitations before it, for the reason recorded in gatecoverage's
// Untested doc comment (#328 option (b)): a gate that failed here would assert
// a policy nobody decided.
//
// # What it cannot catch
//
//   - An unpinned anything that is not an `if`. Switch cases, boolean
//     sub-expressions, loop bounds, error wrapping and arithmetic are all out
//     of scope. This is deliberately the narrowest useful mutation set, not a
//     general mutation-testing engine.
//   - A guard added on a line the diff does not touch (a guard moved by a
//     refactor, for instance, reads as unchanged).
//   - A test that fails for the wrong reason. A mutant is "killed" when the
//     package's tests go red, and this tool does not check that they went red
//     for a related reason.
//   - Anything in a package whose tests do not run, or do not exist. A package
//     with no tests kills no mutants, and every mutant there survives
//     trivially -- which is correct, and is also what `make gate-coverage`
//     already reports more cheaply.
package mutationpin

import (
	"bufio"
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Mutation kinds.
const (
	AlwaysFalse = "if-always-false"
	AlwaysTrue  = "if-always-true"
)

// Mutation is one guard removal to try.
type Mutation struct {
	File string
	Line int    // 1-indexed line of the `if` keyword
	Kind string // AlwaysFalse or AlwaysTrue
	Cond string // the original condition, as written
}

func (m Mutation) String() string {
	return fmt.Sprintf("%s:%d %s (was `%s`)", m.File, m.Line, m.Kind, m.Cond)
}

// Result pairs a mutation with what the test run said about it.
type Result struct {
	Mutation
	Survived bool   // true means no test failed: nothing pins this guard
	Note     string // why it could not be run, when applicable
}

func parse(filename string, src []byte) (*token.FileSet, *ast.File, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return nil, nil, err
	}
	return fset, f, nil
}

// Skip reports whether a file must not be mutated: a test file (mutating the
// tests that do the killing is meaningless) or generated code (recognised by
// Go's own DO NOT EDIT header rather than by path, so a generated file outside
// internal/gen is caught too).
func Skip(filename string, src []byte) bool {
	if strings.HasSuffix(filename, "_test.go") {
		return true
	}
	head := src
	if len(head) > 4096 {
		head = head[:4096]
	}
	return bytes.Contains(head, []byte("DO NOT EDIT"))
}

// Mutants returns the mutations applicable to the `if` statements whose
// keyword falls on one of addedLines.
func Mutants(filename string, src []byte, addedLines map[int]bool) ([]Mutation, error) {
	if len(addedLines) == 0 || Skip(filename, src) {
		return nil, nil
	}
	fset, f, err := parse(filename, src)
	if err != nil {
		return nil, err
	}
	var out []Mutation
	ast.Inspect(f, func(n ast.Node) bool {
		ifs, ok := n.(*ast.IfStmt)
		if !ok || ifs.Cond == nil {
			return true
		}
		line := fset.Position(ifs.If).Line
		if !addedLines[line] {
			return true
		}
		var buf bytes.Buffer
		if err := printer.Fprint(&buf, fset, ifs.Cond); err != nil {
			return true
		}
		cond := buf.String()
		out = append(out, Mutation{File: filename, Line: line, Kind: AlwaysFalse, Cond: cond})
		out = append(out, Mutation{File: filename, Line: line, Kind: AlwaysTrue, Cond: cond})
		return true
	})
	sort.Slice(out, func(i, j int) bool {
		if out[i].Line != out[j].Line {
			return out[i].Line < out[j].Line
		}
		return out[i].Kind < out[j].Kind
	})
	return out, nil
}

// Apply returns src with the one targeted condition replaced by a literal.
// Other `if` statements, including ones with an identical condition, are left
// alone: the target is identified by position, not by text.
func Apply(src []byte, m Mutation) ([]byte, error) {
	fset, f, err := parse(m.File, src)
	if err != nil {
		return nil, err
	}
	lit := "false"
	if m.Kind == AlwaysTrue {
		lit = "true"
	}
	replaced := false
	ast.Inspect(f, func(n ast.Node) bool {
		ifs, ok := n.(*ast.IfStmt)
		if !ok || ifs.Cond == nil || replaced {
			return true
		}
		if fset.Position(ifs.If).Line != m.Line {
			return true
		}
		ifs.Cond = &ast.Ident{NamePos: ifs.Cond.Pos(), Name: lit}
		replaced = true
		return false
	})
	if !replaced {
		return nil, fmt.Errorf("no if-statement at %s:%d", m.File, m.Line)
	}
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fset, f); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

var (
	diffFileRe = regexp.MustCompile(`^\+\+\+ b/(.+)$`)
	diffHunkRe = regexp.MustCompile(`^@@ -\d+(?:,\d+)? \+(\d+)(?:,(\d+))? @@`)
)

// ParseAddedLines reads `git diff --unified=0` output and returns, per Go
// source file, the set of line numbers the diff adds. Test files and files
// with no added lines are omitted.
func ParseAddedLines(diff string) map[string]map[int]bool {
	out := map[string]map[int]bool{}
	var file string
	sc := bufio.NewScanner(strings.NewReader(diff))
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if m := diffFileRe.FindStringSubmatch(line); m != nil {
			file = m[1]
			continue
		}
		if !strings.HasSuffix(file, ".go") || strings.HasSuffix(file, "_test.go") {
			continue
		}
		m := diffHunkRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		start, err := strconv.Atoi(m[1])
		if err != nil {
			continue
		}
		count := 1
		if m[2] != "" {
			if count, err = strconv.Atoi(m[2]); err != nil {
				continue
			}
		}
		if count == 0 {
			continue // a pure deletion hunk adds nothing
		}
		if out[file] == nil {
			out[file] = map[int]bool{}
		}
		for i := 0; i < count; i++ {
			out[file][start+i] = true
		}
	}
	return out
}

// TestRunner runs the tests that could kill a mutant in pkgDir, relative to
// the repository root. It reports whether they PASSED. A passing run means the
// mutant survived.
type TestRunner func(pkgDir string) (passed bool, err error)

// Run applies each mutant to the working tree one at a time, runs the owning
// package's tests, and restores the file. The original bytes are restored on
// every path, including a panic: a tool that can leave a mutated source file
// behind is worse than no tool, because the next command silently tests the
// mutant.
func Run(root string, added map[string]map[int]bool, run TestRunner) ([]Result, error) {
	var files []string
	for f := range added {
		files = append(files, f)
	}
	sort.Strings(files)

	var results []Result
	for _, rel := range files {
		abs := filepath.Join(root, rel)
		orig, err := os.ReadFile(abs)
		if err != nil {
			continue // deleted or moved by the diff; nothing to mutate
		}
		muts, err := Mutants(rel, orig, added[rel])
		if err != nil || len(muts) == 0 {
			continue // unparseable or nothing to do; not this tool's business
		}
		pkgDir := filepath.Dir(rel)
		for _, m := range muts {
			res, err := runOne(abs, orig, m, pkgDir, run)
			if err != nil {
				return nil, err
			}
			results = append(results, res)
		}
	}
	return results, nil
}

// runOne mutates, tests, and restores. Split out so the restore is a deferred
// call with nothing after it that could skip.
func runOne(abs string, orig []byte, m Mutation, pkgDir string, run TestRunner) (res Result, err error) {
	mutated, err := Apply(orig, m)
	if err != nil {
		return Result{Mutation: m, Survived: false, Note: "could not apply: " + err.Error()}, nil
	}
	info, err := os.Stat(abs)
	if err != nil {
		return Result{}, err
	}
	if err := os.WriteFile(abs, mutated, info.Mode().Perm()); err != nil {
		return Result{}, err
	}
	defer func() {
		if rerr := os.WriteFile(abs, orig, info.Mode().Perm()); rerr != nil && err == nil {
			err = fmt.Errorf("RESTORE FAILED for %s -- the working tree now holds a mutant: %w", abs, rerr)
		}
	}()

	passed, runErr := run(pkgDir)
	if runErr != nil {
		return Result{Mutation: m, Note: "test run failed to execute: " + runErr.Error()}, nil
	}
	return Result{Mutation: m, Survived: passed}, nil
}

// Report renders results for a build log. Advisory: the caller exits 0.
func Report(results []Result) string {
	var survivors, notes []Result
	for _, r := range results {
		switch {
		case r.Note != "":
			notes = append(notes, r)
		case r.Survived:
			survivors = append(survivors, r)
		}
	}
	var b strings.Builder
	if len(results) == 0 {
		b.WriteString("mutation-pin: OK — this diff adds no `if` guard to mutate.\n")
		return b.String()
	}
	if len(survivors) == 0 {
		fmt.Fprintf(&b, "mutation-pin: OK — all %d mutant(s) of this diff's added guards were killed by a test.\n", len(results))
	} else {
		fmt.Fprintf(&b, "mutation-pin: NOTE — %d of %d mutant(s) SURVIVED: the guard was removed and no test failed.\n",
			len(survivors), len(results))
		b.WriteString("  Advisory, never fails the build. A survivor is evidence, not a verdict: a defensive\n")
		b.WriteString("  check the type system already prevents, or an error branch no test can provoke, will\n")
		b.WriteString("  survive legitimately. What it rules out is the guard being pinned.\n")
		b.WriteString("  Both directions surviving on one line means the condition is inert to the suite.\n\n")
		for _, r := range survivors {
			fmt.Fprintf(&b, "  %s:%d  %s\n      condition: %s\n", r.File, r.Line, r.Kind, r.Cond)
		}
		b.WriteString("\n")
	}
	if len(notes) > 0 {
		fmt.Fprintf(&b, "  %d mutant(s) could not be evaluated:\n", len(notes))
		for _, r := range notes {
			fmt.Fprintf(&b, "    %s:%d %s — %s\n", r.File, r.Line, r.Kind, r.Note)
		}
	}
	return b.String()
}
