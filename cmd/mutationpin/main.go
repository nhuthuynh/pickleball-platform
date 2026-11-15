// Command mutationpin reports which `if` guards a change ADDS can be removed
// without any test failing.
//
// Advisory: it ALWAYS EXITS 0 with findings. Exit 2 is a broken run (git or
// the tree could not be read), which is not a finding. See
// tools/mutationpin's package doc for why this reports rather than gates, and
// for the list of things a surviving mutant legitimately is.
//
// It MUTATES FILES IN THE WORKING TREE, one at a time, restoring each before
// the next. Run it on a clean tree: an interrupted run (SIGKILL, a dead
// container) can leave one file mutated, and `git diff` is how you would see
// that. The restore is deferred and covered by a test, but a process that
// never returns cannot run a defer.
//
// Build to a scratch path, never the repository root:
//
//	go build -o /tmp/mutationpin ./cmd/mutationpin
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/nhuthuynh/white-label/tools/mutationpin"
)

func main() {
	base := flag.String("base", "origin/claude/go-backend-pickleball-7up34j",
		"git ref to diff against; the shared integration branch, not main (CLAUDE.md rule 14)")
	root := flag.String("root", ".", "repository root")
	max := flag.Int("max", 40, "stop after this many mutants; each one costs a package test run")
	perTest := flag.Duration("timeout", 5*time.Minute, "timeout for one package's test run")
	flag.Parse()

	diff, err := gitDiff(*root, *base)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mutation-pin: could not read the diff against %s: %v\n", *base, err)
		os.Exit(2)
	}

	added := mutationpin.ParseAddedLines(diff)

	// A NEW file is invisible to `git diff` until it is staged, and a new file
	// is exactly where new guards live. Without this, adding a whole package
	// of guards reported "no `if` guard to mutate" -- the second vacuous green
	// this tool produced before anyone looked, after the base...HEAD one above.
	untracked, err := untrackedGoFiles(*root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mutation-pin: could not list untracked files: %v\n", err)
		os.Exit(2)
	}
	for _, rel := range untracked {
		lines, err := allLines(filepath.Join(*root, rel))
		if err != nil {
			continue
		}
		added[rel] = lines
	}
	if len(added) == 0 {
		fmt.Print(mutationpin.Report(nil))
		return
	}

	run, calls := goTestRunner(*root, *perTest, *max)
	results, err := mutationpin.Run(*root, added, run)
	if err != nil {
		// A restore failure lands here, and it is the one thing worth shouting
		// about: the working tree may hold a mutant.
		fmt.Fprintf(os.Stderr, "mutation-pin: %v\n", err)
		os.Exit(2)
	}
	fmt.Print(mutationpin.Report(results))
	if *calls >= *max {
		fmt.Printf("  (stopped at the -max limit of %d mutants; the diff has more)\n", *max)
	}
}

// gitDiff returns the change against the MERGE BASE, compared to the WORKING
// TREE -- committed and uncommitted alike.
//
// The first version used `git diff base...HEAD`, which cannot see uncommitted
// work. Running it on a branch whose changes were not yet committed printed
// "OK -- this diff adds no `if` guard to mutate", which is a vacuous green:
// the exact failure tools/stalecitations hit with a mistyped -root, found the
// same way, by running the thing and disbelieving a clean answer. Since the
// most useful moment to run this is BEFORE committing, that default was
// precisely backwards.
//
// Two commands, because `git diff <base>` with a branch name compares against
// that branch's tip rather than the point of divergence; merge-base is what
// makes "what this change adds" mean what it says.
// untrackedGoFiles lists .go files git does not track yet. Every line of such
// a file is new, so every guard in it is in scope.
func untrackedGoFiles(root string) ([]string, error) {
	cmd := exec.Command("git", "ls-files", "--others", "--exclude-standard", "--", "*.go")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var files []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}

// allLines marks every line of a file as added.
func allLines(abs string) (map[int]bool, error) {
	body, err := os.ReadFile(abs)
	if err != nil {
		return nil, err
	}
	out := map[int]bool{}
	for i := range strings.Split(string(body), "\n") {
		out[i+1] = true
	}
	return out, nil
}

func gitDiff(root, base string) (string, error) {
	mb := exec.Command("git", "merge-base", base, "HEAD")
	mb.Dir = root
	out, err := mb.Output()
	if err != nil {
		return "", fmt.Errorf("merge-base %s HEAD: %w", base, err)
	}
	mergeBase := strings.TrimSpace(string(out))

	cmd := exec.Command("git", "diff", "--unified=0", mergeBase, "--", "*.go")
	cmd.Dir = root
	d, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git diff %s: %w", mergeBase, err)
	}
	return string(d), nil
}

// goTestRunner returns a TestRunner and a pointer to its call count, so main
// can tell whether the -max limit was reached.
func goTestRunner(root string, timeout time.Duration, max int) (mutationpin.TestRunner, *int) {
	calls := 0
	return func(pkgDir string) (bool, error) {
		if calls >= max {
			return false, fmt.Errorf("-max limit of %d mutants reached", max)
		}
		calls++
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		cmd := exec.CommandContext(ctx, "go", "test", "./"+pkgDir+"/", "-count=1")
		cmd.Dir = root
		err := cmd.Run()
		if ctx.Err() != nil {
			return false, fmt.Errorf("package %s: test run timed out after %s", pkgDir, timeout)
		}
		// A non-zero exit means a test failed: the mutant was killed.
		return err == nil, nil
	}, &calls
}
