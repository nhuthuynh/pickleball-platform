// Command docsdoctest fails the build when a documented command no longer
// produces the output printed beside it.
//
// Both sides are derived at run time — the claims from a scan of every
// markdown file in the tree, the actual output by running the command — so a
// doc written next sprint is covered with no edit here. See tools/docsdoctest
// for the opt-in classification rule, the normalisation policy, the gate's
// blind spot, and an honest list of what it cannot catch.
//
// Usage:
//
//	go run ./cmd/docsdoctest [-root .] [-timeout 5m]
//
// Exit status is 1 when some documented command does not reproduce its stated
// output, or when no block was checked at all.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nhuthuynh/white-label/tools/docsdoctest"
)

func main() {
	root := flag.String("root", ".", "repository root to scan for documented commands")
	timeout := flag.Duration("timeout", docsdoctest.DefaultTimeout, "per-command time limit")
	blindSpots := flag.Bool("blind-spots", false,
		"also list every unmarked documented command, i.e. everything this gate does not check")
	flag.Parse()

	abs, err := filepath.Abs(*root)
	if err != nil {
		fail(err)
	}

	rep, err := docsdoctest.Run(abs, *timeout, docsdoctest.ShellRunner(abs))
	if err != nil {
		fail(err)
	}

	if *blindSpots {
		fmt.Print(rep.BlindSpots())
		fmt.Println()
	}
	fmt.Print(rep)
	if !rep.OK() {
		os.Exit(1)
	}
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "docs-doctest: ERROR — %v\n", err)
	// A check that could not run has NOT passed. Exiting non-zero here is
	// the difference between this gate and one that reports green because
	// its own machinery broke.
	os.Exit(2)
}
