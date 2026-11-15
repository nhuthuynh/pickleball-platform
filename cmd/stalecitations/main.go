// Command stalecitations reports every hand-written line that cites a
// migration which SEEDED data a later migration then CHANGED.
//
// It is advisory and ALWAYS EXITS 0 — see tools/stalecitations's package doc
// for why a failing gate here would assert a policy nobody decided. Exit 2 is
// reserved for an unreadable tree, which is a broken run, not a finding.
//
// Build to a scratch path, never to the repository root:
//
//	go build -o /tmp/stalecitations ./cmd/stalecitations
//
// `go build ./cmd/stalecitations` with no -o drops a binary named after the
// package at the root, where `git add -A` tracks it — the incident `make
// binary-check` exists to catch.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/nhuthuynh/white-label/tools/stalecitations"
)

func main() {
	root := flag.String("root", ".", "repository root to scan")
	flag.Parse()

	findings, err := stalecitations.Scan(*root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "stale-citations: could not scan %s: %v\n", *root, err)
		os.Exit(2)
	}
	fmt.Print(stalecitations.Report(findings))
}
