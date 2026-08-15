// Command gatecoverage fails the build when a package holds test functions
// that no gate executes.
//
// Both sides of that comparison are computed at run time — the packages from a
// scan of the module, the gates from the Makefile itself — so a package added
// next sprint is covered with no edit here. See tools/gatecoverage for why
// that constraint is the whole point.
//
// Usage:
//
//	go run ./cmd/gatecoverage [-root .] [-makefile Makefile] [-gate ci-checks]
//
// Exit status is 1 when some package's tests are executed by nothing.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nhuthuynh/white-label/tools/gatecoverage"
)

func main() {
	root := flag.String("root", ".", "module root to scan")
	makefile := flag.String("makefile", "", "path to the Makefile (default <root>/Makefile)")
	gate := flag.String("gate", gatecoverage.DefaultGateRoot, "Makefile target treated as the standing gate")
	flag.Parse()

	abs, err := filepath.Abs(*root)
	if err != nil {
		fail(err)
	}
	mkPath := *makefile
	if mkPath == "" {
		mkPath = filepath.Join(abs, "Makefile")
	}

	rep, err := gatecoverage.Run(abs, mkPath, *gate, gatecoverage.GoList(abs))
	if err != nil {
		fail(err)
	}

	fmt.Print(rep)
	if !rep.OK() {
		os.Exit(1)
	}
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "gate-coverage: ERROR — %v\n", err)
	// A check that cannot run has NOT passed. Exiting non-zero here is the
	// difference between this gate and one that reports green because its
	// own machinery broke.
	os.Exit(2)
}
