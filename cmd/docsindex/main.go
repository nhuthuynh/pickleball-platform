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
	statuses := flag.Bool("statuses", false,
		"list every ADR's status instead of checking the index (same parser the check uses)")
	flag.Parse()

	if *statuses {
		if err := listStatuses(*root); err != nil {
			fmt.Fprintf(os.Stderr, "docs-index-check: %v\n", err)
			os.Exit(2)
		}
		return
	}

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

// listStatuses answers the question Ceremony 1's escalation sweep actually asks
// — "which ADRs are Escalated?" — from the same parser the gate uses.
//
// Before this existed the gate could refuse an unreadable status and could not
// print a readable one, so T64's own ceremony copied tools/docsindex into a
// scratch directory to avoid re-implementing the parser as a grep. See
// docsindex.ADRStatuses for why that is the one thing worth not doing.
func listStatuses(root string) error {
	adrs, err := docsindex.ADRStatuses(root)
	if err != nil {
		return err
	}
	counts := map[string]int{}
	for _, a := range adrs {
		token, form := a.Token, a.Form
		if !a.Classifiable() {
			token = "UNCLASSIFIABLE"
			if form == "" {
				form = "no status"
			}
		}
		counts[token]++
		fmt.Printf("%-14s %-13s %s\n", token, form, a.File)
	}
	fmt.Printf("\n%d ADR(s):", len(adrs))
	for _, t := range append(docsindex.CanonicalStatuses(), "UNCLASSIFIABLE") {
		if counts[t] > 0 {
			fmt.Printf(" %s=%d", t, counts[t])
		}
	}
	fmt.Println()

	// An unclassifiable status is a failure here too, not just in the check —
	// a listing that quietly omitted one would be the silent skip T62.4 was
	// written to stop, reintroduced in the tool built to show the values.
	if n := counts["UNCLASSIFIABLE"]; n > 0 {
		return fmt.Errorf("%d ADR(s) carry a status this parser cannot classify — "+
			"`go run ./cmd/docsindex` names them", n)
	}
	return nil
}
