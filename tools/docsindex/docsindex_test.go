package docsindex_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nhuthuynh/white-label/tools/docsindex"
)

// fixture builds a throwaway tree: HANDOFF.md plus whatever docs/process and
// docs/adr files each case needs. Writing a real tree rather than faking the
// filesystem is deliberate — this tool's whole job is to compare a document
// against a directory, so a fake directory would test the wrong half.
type fixture struct {
	handoff string
	process map[string]string // filename -> contents (contents are irrelevant)
	adrs    map[string]string
}

func (f fixture) write(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "HANDOFF.md"), []byte(f.handoff), 0o644); err != nil {
		t.Fatal(err)
	}
	for dir, files := range map[string]map[string]string{
		filepath.Join("docs", "process"): f.process,
		filepath.Join("docs", "adr"):     f.adrs,
	} {
		full := filepath.Join(root, dir)
		if err := os.MkdirAll(full, 0o755); err != nil {
			t.Fatal(err)
		}
		for name, body := range files {
			if err := os.WriteFile(filepath.Join(full, name), []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	return root
}

const header = "| Phase | Sprint plan | Retro | Reviews | Key ADRs | Design |\n|---|---|---|---|---|---|\n"

// okADR is the minimum an ADR needs to classify, in the front-matter form.
const okADR = "# ADR-0001: Something\n\n- **Status:** Accepted\n- **Date:** 2026-01-01\n"

func check(t *testing.T, f fixture) docsindex.Report {
	t.Helper()
	rep, err := docsindex.Check(f.write(t))
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	return rep
}

func findingsContaining(rep docsindex.Report, substr string) int {
	n := 0
	for _, f := range rep.Findings {
		if strings.Contains(f.String(), substr) {
			n++
		}
	}
	return n
}

// TestCorrectionQuotingTheStalePhrasePasses is the test this package exists to
// keep, and the one most worth reading.
//
// T62's Ceremony 1 hit the same trap three times in one sitting: ADR-0015/0016
// preserve the word "Escalated" beneath a supersession notice; T61's sweep left
// 17 stale "no Docker daemon" clauses beneath their refutations; and this
// check's own first draft keyed on the ABSENCE of "not yet written" and
// false-positived on T54's and T42's own corrections.
//
//	A correction that preserves the claim it corrects will match any checker
//	that greps for the claim.
//
// So a row whose Retro cell BOTH names the real file AND quotes the retired
// phrase inside its correction must PASS. If this test ever fails, someone has
// rewritten a check as "the cell must not say X", and that is the regression.
func TestCorrectionQuotingTheStalePhrasePasses(t *testing.T) {
	t.Parallel()

	rep := check(t, fixture{
		handoff: header +
			"| T54 | `docs/process/t54-sprint-plan.md` | " +
			"`docs/process/t54-retro.md` (**this cell read \"not yet written\" from T54 until " +
			"T62's Ceremony 1 — eight sprints**) | PR #290 | — | — |\n" +
			"| T42 | plan | `docs/process/t42-retro.md` | — | — | — |\n" +
			"\n**T42 — a narrative.** Retro not yet written — corrected at T62: it is " +
			"`docs/process/t42-retro.md`, and has existed since T42.\n",
		process: map[string]string{
			"t54-sprint-plan.md": "x",
			"t54-retro.md":       "x",
			"t42-retro.md":       "x",
		},
		adrs: map[string]string{"0001-a.md": okADR},
	})

	if !rep.OK() {
		t.Fatalf("a correction that quotes the phrase it retires must pass, got %d finding(s):\n%v\n\n"+
			"This is the positive-assertion constraint from tools/docsindex's doc comment. "+
			"A check phrased as \"the cell must not say X\" fails here by construction.",
			len(rep.Findings), rep.Findings)
	}
}

// A retro that exists and a row that does not name it is the T54/T55 defect.
func TestRetroExistsButRowDoesNotNameIt(t *testing.T) {
	t.Parallel()

	rep := check(t, fixture{
		handoff: header + "| T54 | `docs/process/t54-sprint-plan.md` | not yet written | PR #290 | — | — |\n",
		process: map[string]string{"t54-sprint-plan.md": "x", "t54-retro.md": "x"},
		adrs:    map[string]string{"0001-a.md": okADR},
	})

	if rep.OK() {
		t.Fatal("a Retro cell that does not name an existing retro must fail — this is the T54 defect, stale for eight sprints")
	}
	if got := findingsContaining(rep, "docs/process/t54-retro.md exists but the Retro cell does not name it"); got != 1 {
		t.Fatalf("want one finding naming the unnamed retro, got %d: %v", got, rep.Findings)
	}
}

// A document with no row at all is the T60/T61 defect.
func TestDocumentWithNoRow(t *testing.T) {
	t.Parallel()

	rep := check(t, fixture{
		handoff: header + "| T59 | `docs/process/t59-sprint-plan.md` | `docs/process/t59-retro.md` | — | — | — |\n",
		process: map[string]string{
			"t59-sprint-plan.md": "x", "t59-retro.md": "x",
			"t60-retro.md": "x", // exists, no row
		},
		adrs: map[string]string{"0001-a.md": okADR},
	})

	if got := findingsContaining(rep, "T60"); got != 1 {
		t.Fatalf("a retro with no Docs-index row must fail, naming the phase; got %d finding(s): %v", got, rep.Findings)
	}
}

// The cell-count check is the T58 (four cells) and T59 (eight cells) defect.
// Both directions, because the T58 and T59 rows were wrong opposite ways and a
// check that caught only one would have reported the index clean at T62.
func TestCellCountMismatchBothDirections(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name, row string
		want      string
	}{
		{
			name: "too few cells, as T58 had",
			row:  "| T58 | plan | `docs/process/t58-retro.md` | PR #301 |\n",
			want: "row has 4 cells, header has 6",
		},
		{
			name: "too many cells, as T59 had",
			row:  "| T59 | plan | `docs/process/t59-retro.md` | PR #306 | — | — | none new | — |\n",
			want: "row has 8 cells, header has 6",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			phase := "t58"
			if strings.Contains(tc.row, "T59") {
				phase = "t59"
			}
			rep := check(t, fixture{
				handoff: header + tc.row,
				process: map[string]string{phase + "-retro.md": "x"},
				adrs:    map[string]string{"0001-a.md": okADR},
			})
			if got := findingsContaining(rep, tc.want); got != 1 {
				t.Fatalf("want a finding saying %q, got: %v", tc.want, rep.Findings)
			}
		})
	}
}

// A narrative that still claims the retro is unwritten, and never names the
// file, is the T42 defect — stale for twenty sprints.
func TestNarrativeKeepsStaleClaimWithoutNamingTheFile(t *testing.T) {
	t.Parallel()

	rep := check(t, fixture{
		handoff: header + "| T42 | plan | `docs/process/t42-retro.md` | — | — | — |\n" +
			"\n**T42 — Ceremony 1/2 only.** Retro not yet written.\n",
		process: map[string]string{"t42-retro.md": "x"},
		adrs:    map[string]string{"0001-a.md": okADR},
	})

	if got := findingsContaining(rep, "T42 narrative"); got != 1 {
		t.Fatalf("want one finding about the T42 narrative, got %d: %v", got, rep.Findings)
	}
}

// Both ADR status conventions must be read. T62's Ceremony 1 keyed on the
// heading alone and silently skipped the six front-matter ADRs — which is how
// #314, a decision genuinely awaiting an answer, stayed invisible to the sweep
// built to surface decisions awaiting answers.
func TestBothADRStatusFormsAreRead(t *testing.T) {
	t.Parallel()

	rep := check(t, fixture{
		handoff: header,
		adrs: map[string]string{
			"0001-heading-form.md":      "# ADR-0001: x\n\n## Status\nAccepted (T0)\n\n## Context\n",
			"0014-front-matter-form.md": "# ADR-0014: x\n\n- **Status:** Accepted\n- **Date:** 2026-08-15\n",
			"0015-token-then-prose.md": "# ADR-0015: x\n\n- **Status:** **Accepted — D1 answered on 2026-09-04: " +
				"option (a).** The question below is preserved unedited, including the words " +
				"\"Escalated — awaiting product decision\".\n",
		},
	})

	if !rep.OK() {
		t.Fatalf("all three status shapes must classify, got: %v", rep.Findings)
	}
	if rep.ADRsChecked != 3 {
		t.Fatalf("ADRsChecked = %d, want 3 — a status form this tool cannot read must fail, never be skipped", rep.ADRsChecked)
	}
}

// An ADR whose status this tool cannot classify is a failure, never a silent
// skip. The silent skip is what made T62's escalation gap invisible.
func TestUnclassifiableADRFails(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct{ name, body, want string }{
		{
			name: "no status at all",
			body: "# ADR-0099: x\n\n## Context\nSomething.\n",
			want: "no status in either accepted form",
		},
		{
			name: "status that does not lead with a token",
			body: "# ADR-0006: x\n\n## Status\nGame waitlists shipped in T6.6.\n",
			want: "does not begin with one of",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rep := check(t, fixture{handoff: header, adrs: map[string]string{"0099-x.md": tc.body}})
			if got := findingsContaining(rep, tc.want); got != 1 {
				t.Fatalf("want a finding saying %q, got: %v", tc.want, rep.Findings)
			}
		})
	}
}

// The control. Without it, a tool that failed everything would satisfy every
// test above.
func TestConsistentTreePasses(t *testing.T) {
	t.Parallel()

	rep := check(t, fixture{
		handoff: header +
			"| T60 | **No sprint-plan document.** | `docs/process/t60-retro.md` | PR #307 | none new | — |\n" +
			"| T61 | **No sprint-plan document.** | `docs/process/t61-retro.md` | PR #310 | none new | — |\n" +
			"\n**T61 — a narrative.** Retro: `docs/process/t61-retro.md`.\n",
		process: map[string]string{"t60-retro.md": "x", "t61-retro.md": "x"},
		adrs:    map[string]string{"0001-a.md": okADR},
	})

	if !rep.OK() {
		t.Fatalf("a consistent tree must pass, got: %v", rep.Findings)
	}
	if rep.PhasesInIndex != 2 || rep.DocsOnDisk != 2 {
		t.Fatalf("PhasesInIndex=%d DocsOnDisk=%d, want 2 and 2 — a run that examined nothing "+
			"must not be mistakable for a clean one", rep.PhasesInIndex, rep.DocsOnDisk)
	}
}

// A duplicated row would let one correct row mask a stale one.
func TestDuplicateRowFails(t *testing.T) {
	t.Parallel()

	rep := check(t, fixture{
		handoff: header +
			"| T60 | plan | not yet written | — | — | — |\n" +
			"| T60 | plan | `docs/process/t60-retro.md` | — | — | — |\n",
		process: map[string]string{"t60-retro.md": "x"},
		adrs:    map[string]string{"0001-a.md": okADR},
	})

	if got := findingsContaining(rep, "two Docs-index rows for the same phase"); got != 1 {
		t.Fatalf("want one duplicate-row finding, got: %v", rep.Findings)
	}
}
