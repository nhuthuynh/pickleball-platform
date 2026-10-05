// Package docsindex answers one question mechanically, every time it runs:
// **does HANDOFF.md's Docs index still describe the documents that exist?**
//
// # Why this exists
//
// `sprint-process.md` assigns Docs-index correction to the *next* sprint's
// Ceremony 1, correctly — a row must cite the retro's own merge PR number,
// which does not exist until that PR merges, so a retro PR is structurally
// incapable of writing its own row. What that rule does not survive is a
// successor that holds no Ceremony 1.
//
// T62's Ceremony 1 was the first since T59. Running a derived check rather
// than re-reading the table found eight sprints of accumulated debt
// (`docs/process/t62-sprint-plan.md` §3):
//
//	T54 | Retro cell read "not yet written"; the file had existed since T54 | 8 sprints
//	T55 | same                                                             | 7 sprints
//	T58 | FOUR cells against a six-column header                           | 4 sprints
//	T59 | stale Retro cell AND eight cells                                 | 3 sprints
//	T60 | no row at all                                                    | 2 sprints
//	T61 | no row at all                                                    | 1 sprint
//	T42 | narrative said "retro not yet written"; the file exists           | 20 sprints
//
// A markdown table with inconsistent cell counts drops or empties cells when
// rendered, so the artifact CLAUDE.md instructs every session to read first —
// "it's the map" — had been mis-rendering for its four most recent sprints,
// with two sprints missing from it entirely.
//
// `docs/process/t60-retro.md` recommendation 4 proposed a firmer prose rule:
// a sprint that holds no planning ceremony still writes the previous sprint's
// row as its first act. T62's §3 argues that is not enough, and the number is
// the argument: the debt is eight sprints across six missed ceremonies, and
// the T42 note had been stale for twenty. **A rule six consecutive sprints did
// not follow will not be followed by the seventh for having been written down
// more firmly.** So this is a gate instead.
//
// # The design constraint, which is the whole point
//
// Both sides are computed at run time and neither is a list:
//
//   - Side A — the documents that exist — is a directory scan of
//     `docs/process` and `docs/adr`. A retro written next sprint appears in it
//     with no edit here.
//   - Side B — what `HANDOFF.md` claims — is parsed from `HANDOFF.md` itself.
//
// **There is no phase list in this tool, and adding one is the one change that
// would defeat it** — exactly as `tools/gatecoverage` says of its own package
// list. Three sprints running (T11, T12, T13/#157) shipped a hand-written glob
// that was stale before its sprint ended; the Docs index then went stale for
// eight. Fix a failure by correcting `HANDOFF.md`, never by adding an
// exclusion.
//
// # Assert the positive, never the absence of a phrase
//
// This constraint was discovered three times inside T62's single ceremony, and
// it is the reason this package never greps for a stale phrase:
//
//  1. ADR-0015 and ADR-0016 preserve the words "Escalated — awaiting product
//     decision" verbatim beneath a supersession notice, so a literal grep for
//     "Escalated" over their prose hits two resolved decisions.
//
//  2. T61's sweep deliberately left 17 stale "no Docker daemon" clauses in
//     place beneath their refutations, so a grep for the clause still matches
//     all 17.
//
//  3. This check's own first draft keyed on the ABSENCE of "not yet written"
//     and immediately false-positived on T54's and T42's own *corrections* —
//     because a correction quotes the phrase it retires.
//
//     A correction that preserves the claim it corrects will match any
//     checker that greps for the claim.
//
// So every check here is a positive assertion about what a cell or section
// *names*: "the Retro cell names a file that exists", never "the Retro cell
// does not say X". `TestCorrectionQuotingTheStalePhrasePasses` pins exactly
// that, and it is the test most worth keeping.
//
// # ADR status, and why it lives here too
//
// T62's Ceremony 1 also found its own escalation sweep unable to read six of
// seventeen ADRs, because **two status conventions coexist**: eleven ADRs use a
// `## Status` heading, six use a `- **Status:**` front-matter bullet. The
// sweep keyed on the heading and silently skipped the rest — which is how
// #314, a decision genuinely awaiting an answer, stayed invisible to the step
// built to surface decisions awaiting answers.
//
// The fix is to read both forms and require the value to *begin* with a
// canonical token, not to rewrite eleven files into one convention. An ADR this
// package cannot classify is a **failure**, never a silent skip: a silent skip
// is precisely what made the gap invisible.
package docsindex

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Finding is one thing wrong, phrased so the message alone says what to do.
type Finding struct {
	Where string // the file, and the phase or ADR within it
	What  string
}

func (f Finding) String() string { return fmt.Sprintf("%s: %s", f.Where, f.What) }

// Report is everything one run found.
type Report struct {
	Findings []Finding
	// Checked is what the run actually looked at, printed so a run that
	// silently examined nothing cannot be mistaken for a clean one.
	PhasesInIndex int
	DocsOnDisk    int
	ADRsChecked   int
}

// OK reports whether the tree is consistent.
func (r Report) OK() bool { return len(r.Findings) == 0 }

// canonicalStatuses are the only ADR status tokens this project uses. A value
// must BEGIN with one; everything after it is free prose, which is how
// "Accepted (T0)" and "Accepted — D1 answered on 2026-09-04: option (a)" both
// classify without either being rewritten.
var canonicalStatuses = []string{"Accepted", "Superseded", "Proposed", "Escalated", "Rejected"}

var (
	rowRe         = regexp.MustCompile(`^\|\s*T(\d+)\s*\|`)
	headerRe      = regexp.MustCompile(`^\|\s*Phase\s*\|`)
	docRe         = regexp.MustCompile(`^t(\d+)-(retro|sprint-plan)\.md$`)
	narrativeRe   = regexp.MustCompile(`^\*\*T(\d+)`)
	stalePhraseRe = regexp.MustCompile(`(?i)retro not yet written`)
	adrFrontRe    = regexp.MustCompile(`(?m)^-\s*\*\*Status:\*\*\s*(.+)$`)
	adrHeadingRe  = regexp.MustCompile(`(?m)^##\s*Status\s*\n+(.+)$`)
)

// Check runs every check against the tree rooted at root.
func Check(root string) (Report, error) {
	var rep Report

	handoffPath := filepath.Join(root, "HANDOFF.md")
	handoff, err := os.ReadFile(handoffPath)
	if err != nil {
		return rep, fmt.Errorf("reading HANDOFF.md: %w", err)
	}
	lines := strings.Split(string(handoff), "\n")

	// --- side A: what exists on disk ---
	processDir := filepath.Join(root, "docs", "process")
	entries, err := os.ReadDir(processDir)
	if err != nil {
		return rep, fmt.Errorf("reading docs/process: %w", err)
	}
	retroExists := map[int]string{} // phase -> repo-relative path
	planExists := map[int]string{}
	for _, e := range entries {
		m := docRe.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		n, _ := strconv.Atoi(m[1])
		rel := filepath.ToSlash(filepath.Join("docs", "process", e.Name()))
		if m[2] == "retro" {
			retroExists[n] = rel
		} else {
			planExists[n] = rel
		}
	}
	rep.DocsOnDisk = len(retroExists) + len(planExists)

	// --- side B: what HANDOFF.md's Docs index claims ---
	headerCells := -1
	rows := map[int][]string{}
	rowOrder := []int{}
	for _, l := range lines {
		if headerCells < 0 && headerRe.MatchString(l) {
			headerCells = len(splitRow(l))
			continue
		}
		m := rowRe.FindStringSubmatch(l)
		if m == nil {
			continue
		}
		n, _ := strconv.Atoi(m[1])
		if _, dup := rows[n]; dup {
			rep.Findings = append(rep.Findings, Finding{
				Where: fmt.Sprintf("HANDOFF.md T%d", n),
				What:  "two Docs-index rows for the same phase",
			})
			continue
		}
		rows[n] = splitRow(l)
		rowOrder = append(rowOrder, n)
	}
	rep.PhasesInIndex = len(rows)

	if headerCells < 0 {
		rep.Findings = append(rep.Findings, Finding{
			Where: "HANDOFF.md",
			What:  "no Docs-index header row (`| Phase | ... |`) found — cannot check cell counts",
		})
	}

	// --- check 1: every document on disk has a row ---
	for _, n := range sortedKeys(retroExists, planExists) {
		if _, ok := rows[n]; !ok {
			rep.Findings = append(rep.Findings, Finding{
				Where: fmt.Sprintf("HANDOFF.md T%d", n),
				What: fmt.Sprintf("no Docs-index row, but %s exists — add the row",
					firstNonEmpty(retroExists[n], planExists[n])),
			})
		}
	}

	// --- checks 2 and 3: per row ---
	for _, n := range rowOrder {
		cells := rows[n]

		if headerCells > 0 && len(cells) != headerCells {
			rep.Findings = append(rep.Findings, Finding{
				Where: fmt.Sprintf("HANDOFF.md T%d", n),
				What: fmt.Sprintf("row has %d cells, header has %d — a markdown table with "+
					"inconsistent cell counts drops cells when rendered", len(cells), headerCells),
			})
		}

		// The Retro cell is column index 2 (Phase, Sprint plan, Retro, ...).
		if len(cells) < 3 {
			continue
		}
		cell := cells[2]
		path, exists := retroExists[n]
		switch {
		case exists && !strings.Contains(cell, path):
			rep.Findings = append(rep.Findings, Finding{
				Where: fmt.Sprintf("HANDOFF.md T%d", n),
				What: fmt.Sprintf("%s exists but the Retro cell does not name it — "+
					"name the path in the cell", path),
			})
		case !exists && mentionsSomeRetroPath(cell, n):
			rep.Findings = append(rep.Findings, Finding{
				Where: fmt.Sprintf("HANDOFF.md T%d", n),
				What:  "the Retro cell names a retro file that does not exist",
			})
		}
	}

	// --- check 4: narrative sections ---
	//
	// Positive form: a narrative section that still carries the stale phrase
	// must ALSO name the retro file, so the correction sits beside the claim it
	// retires rather than below it. A section that quotes the phrase inside its
	// own correction therefore passes, which is the point — see this package's
	// doc comment and TestCorrectionQuotingTheStalePhrasePasses.
	section := -1
	sectionText := map[int][]string{}
	for _, l := range lines {
		if m := narrativeRe.FindStringSubmatch(l); m != nil {
			n, _ := strconv.Atoi(m[1])
			section = n
		}
		if section >= 0 {
			sectionText[section] = append(sectionText[section], l)
		}
	}
	for _, n := range sortedKeys(sectionText) {
		body := strings.Join(sectionText[n], "\n")
		if !stalePhraseRe.MatchString(body) {
			continue
		}
		path, exists := retroExists[n]
		if exists && !strings.Contains(body, path) {
			rep.Findings = append(rep.Findings, Finding{
				Where: fmt.Sprintf("HANDOFF.md, T%d narrative", n),
				What: fmt.Sprintf("says a retro is not yet written and never names %s, "+
					"which exists — put the correction beside the claim, not below it", path),
			})
		}
	}

	// --- check 5: every ADR carries a classifiable status ---
	adrDir := filepath.Join(root, "docs", "adr")
	adrs, err := os.ReadDir(adrDir)
	if err != nil {
		return rep, fmt.Errorf("reading docs/adr: %w", err)
	}
	for _, e := range adrs {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		rep.ADRsChecked++
		src, err := os.ReadFile(filepath.Join(adrDir, e.Name()))
		if err != nil {
			return rep, fmt.Errorf("reading %s: %w", e.Name(), err)
		}
		status, form := adrStatus(string(src))
		switch {
		case form == "":
			rep.Findings = append(rep.Findings, Finding{
				Where: "docs/adr/" + e.Name(),
				What: "no status in either accepted form (`## Status` heading or " +
					"`- **Status:**` front matter) — an ADR this check cannot classify is a " +
					"failure, never a silent skip",
			})
		case !beginsWithCanonicalStatus(status):
			rep.Findings = append(rep.Findings, Finding{
				Where: "docs/adr/" + e.Name(),
				What: fmt.Sprintf("status (%s form) is %q, which does not begin with one of %s — "+
					"keep the prose, but lead with the token so a sweep can read it",
					form, truncate(status, 60), strings.Join(canonicalStatuses, "/")),
			})
		}
	}

	sort.Slice(rep.Findings, func(i, j int) bool { return rep.Findings[i].Where < rep.Findings[j].Where })
	return rep, nil
}

// adrStatus returns the status value and which form carried it. Both forms are
// read deliberately: eleven ADRs use the heading, six use the front-matter
// bullet, and T62's Ceremony 1 demonstrated that a sweep keyed on one silently
// skips the other.
func adrStatus(src string) (value, form string) {
	if m := adrFrontRe.FindStringSubmatch(src); m != nil {
		return clean(m[1]), "front-matter"
	}
	if m := adrHeadingRe.FindStringSubmatch(src); m != nil {
		return clean(m[1]), "## heading"
	}
	return "", ""
}

func beginsWithCanonicalStatus(v string) bool {
	for _, s := range canonicalStatuses {
		if strings.HasPrefix(v, s) {
			return true
		}
	}
	return false
}

// splitRow returns a markdown row's cells, excluding the empty strings either
// side of the leading and trailing pipes.
func splitRow(line string) []string {
	parts := strings.Split(strings.TrimSpace(line), "|")
	if len(parts) >= 2 {
		parts = parts[1 : len(parts)-1]
	}
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		out = append(out, strings.TrimSpace(p))
	}
	return out
}

// mentionsSomeRetroPath reports whether a cell names this phase's retro path,
// used only to catch a cell pointing at a file that is not there.
func mentionsSomeRetroPath(cell string, phase int) bool {
	return strings.Contains(cell, fmt.Sprintf("docs/process/t%d-retro.md", phase))
}

func clean(s string) string {
	s = strings.ReplaceAll(s, "**", "")
	return strings.TrimSpace(s)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func firstNonEmpty(vs ...string) string {
	for _, v := range vs {
		if v != "" {
			return v
		}
	}
	return ""
}

func sortedKeys[V any](ms ...map[int]V) []int {
	seen := map[int]bool{}
	for _, m := range ms {
		for k := range m {
			seen[k] = true
		}
	}
	out := make([]int, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}
