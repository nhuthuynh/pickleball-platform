// Command docsindex fails the build when HANDOFF.md's Docs index stops
// describing the documents that actually exist.
//
// Both sides of that comparison are computed at run time — the documents from a
// scan of docs/process and docs/adr, the claims from HANDOFF.md itself — so a
// retro written next sprint is covered with no edit here. See tools/docsindex
// for why that constraint is the whole point, and for the three times T62's
// Ceremony 1 proved that a check must assert what a cell *names* rather than
// grep for a phrase it should not contain.
//
// Usage:
//
//	go run ./cmd/docsindex [-root .]
//
// Exit status is 1 when the index and the tree disagree.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/nhuthuynh/white-label/tools/docsindex"
)

func main() {
	root := flag.String("root", ".", "repository root to check")
	flag.Parse()

	rep, err := docsindex.Check(*root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "docs-index-check: %v\n", err)
		os.Exit(2)
	}

	// Printed on every run, clean or not: a check that silently examined
	// nothing must not be mistakable for a check that found nothing.
	fmt.Printf("docs-index-check: %d phase row(s) in HANDOFF.md, %d process doc(s) on disk, %d ADR(s).\n",
		rep.PhasesInIndex, rep.DocsOnDisk, rep.ADRsChecked)

	if rep.OK() {
		fmt.Println("docs-index-check: OK — the Docs index, the narratives and every ADR status agree with the tree.")
		return
	}

	fmt.Printf("\ndocs-index-check: %d problem(s):\n\n", len(rep.Findings))
	for _, f := range rep.Findings {
		fmt.Printf("  %s\n", f)
	}
	fmt.Println()
	fmt.Println("Fix these by correcting HANDOFF.md or the ADR, never by adding an exclusion")
	fmt.Println("to tools/docsindex — there is no phase list in that tool, and adding one is")
	fmt.Println("the single change that would defeat it.")
	os.Exit(1)
}
