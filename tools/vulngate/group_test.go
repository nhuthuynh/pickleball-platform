package vulngate_test

import (
	"testing"

	"github.com/nhuthuynh/white-label/tools/vulngate"
)

// #348: the gate's summary counted CALL SITES, so eight advisories were
// reported as "155 new gating finding(s)" — 60-odd identical copies of one
// line. These tests pin the grouped shape.

func goFinding(id string, reach vulngate.Reach, detail string) vulngate.Finding {
	return vulngate.Finding{Scanner: vulngate.ScannerGo, ID: id, Reach: reach, Detail: detail}
}

func TestGroupByID_CollapsesCallSitesAndCountsThem(t *testing.T) {
	findings := []vulngate.Finding{
		goFinding("GO-2026-6603", vulngate.ReachCalled, "HTTP/2 Trailer headers"),
		goFinding("GO-2026-6603", vulngate.ReachCalled, "HTTP/2 Trailer headers"),
		goFinding("GO-2026-6603", vulngate.ReachCalled, "HTTP/2 Trailer headers"),
	}

	got := vulngate.GroupByID(findings)

	if len(got) != 1 {
		t.Fatalf("got %d group(s), want 1: %+v", len(got), got)
	}
	if got[0].Sites != 3 {
		t.Errorf("Sites = %d, want 3", got[0].Sites)
	}
	if got[0].ID != "GO-2026-6603" {
		t.Errorf("ID = %q", got[0].ID)
	}
	if got[0].Detail != "HTTP/2 Trailer headers" {
		t.Errorf("Detail = %q", got[0].Detail)
	}
}

// Distinct advisories must stay distinct — a grouping that collapsed them
// would hide findings, which is far worse than over-counting them.
func TestGroupByID_KeepsDistinctAdvisoriesApart(t *testing.T) {
	findings := []vulngate.Finding{
		goFinding("GO-2026-6603", vulngate.ReachCalled, "a"),
		goFinding("GO-2026-6617", vulngate.ReachCalled, "b"),
		goFinding("GO-2026-6603", vulngate.ReachCalled, "a"),
	}

	got := vulngate.GroupByID(findings)

	if len(got) != 2 {
		t.Fatalf("got %d group(s), want 2: %+v", len(got), got)
	}
	// First-appearance order, so the report reads in the order the scanner
	// produced — the same property Evaluate preserves.
	if got[0].ID != "GO-2026-6603" || got[1].ID != "GO-2026-6617" {
		t.Errorf("order: got %q then %q", got[0].ID, got[1].ID)
	}
	if got[0].Sites != 2 || got[1].Sites != 1 {
		t.Errorf("Sites: got %d and %d, want 2 and 1", got[0].Sites, got[1].Sites)
	}
}

// An ID is only unique WITHIN a scanner: npm findings are keyed by package
// name, and nothing stops a package sharing a string with a Go advisory id.
// Baseline.has is scanner-scoped for the same reason.
func TestGroupByID_IsScannerScoped(t *testing.T) {
	findings := []vulngate.Finding{
		{Scanner: vulngate.ScannerGo, ID: "shared-name", Reach: vulngate.ReachCalled},
		{Scanner: vulngate.ScannerNPM, ID: "shared-name", Severity: vulngate.SevHigh},
	}

	got := vulngate.GroupByID(findings)

	if len(got) != 2 {
		t.Fatalf("got %d group(s), want 2 — a Go advisory and an npm package must not merge: %+v", len(got), got)
	}
}

// Within one ID the reach can differ per site: GO-2026-6809 arrived as both
// `imported` and `called` in the run that prompted #348. The group must
// report the STRONGEST, because that is what Gating() keys on — reporting
// the weakest would describe a called vulnerability as merely imported.
func TestGroupByID_ReportsTheStrongestReach(t *testing.T) {
	for _, tc := range []struct {
		name  string
		order []vulngate.Reach
	}{
		{"weakest first", []vulngate.Reach{vulngate.ReachRequired, vulngate.ReachImported, vulngate.ReachCalled}},
		{"strongest first", []vulngate.Reach{vulngate.ReachCalled, vulngate.ReachImported, vulngate.ReachRequired}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var findings []vulngate.Finding
			for _, r := range tc.order {
				findings = append(findings, goFinding("GO-2026-6809", r, "crypto/x509"))
			}

			got := vulngate.GroupByID(findings)

			if len(got) != 1 {
				t.Fatalf("got %d group(s), want 1", len(got))
			}
			if got[0].Reach != vulngate.ReachCalled {
				t.Errorf("Reach = %q, want %q", got[0].Reach, vulngate.ReachCalled)
			}
		})
	}
}

// Same argument for npm severity: report the worst, never the first seen.
func TestGroupByID_ReportsTheHighestSeverity(t *testing.T) {
	findings := []vulngate.Finding{
		{Scanner: vulngate.ScannerNPM, ID: "undici", Severity: vulngate.SevModerate},
		{Scanner: vulngate.ScannerNPM, ID: "undici", Severity: vulngate.SevHigh},
		{Scanner: vulngate.ScannerNPM, ID: "undici", Severity: vulngate.SevLow},
	}

	got := vulngate.GroupByID(findings)

	if len(got) != 1 {
		t.Fatalf("got %d group(s), want 1", len(got))
	}
	if got[0].Severity != vulngate.SevHigh {
		t.Errorf("Severity = %q, want %q", got[0].Severity, vulngate.SevHigh)
	}
}

// govulncheck's `osv` line can arrive after the `finding` that references
// it, so ParseGovulncheck fills Detail in a second pass — but a report
// truncated mid-stream can leave some blank. Take the first non-empty
// rather than whichever happened to be first.
func TestGroupByID_TakesTheFirstNonEmptyDetail(t *testing.T) {
	findings := []vulngate.Finding{
		goFinding("GO-2026-6603", vulngate.ReachCalled, ""),
		goFinding("GO-2026-6603", vulngate.ReachCalled, "the real summary"),
	}

	got := vulngate.GroupByID(findings)

	if got[0].Detail != "the real summary" {
		t.Errorf("Detail = %q, want %q", got[0].Detail, "the real summary")
	}
}

func TestGroupByID_EmptyInput(t *testing.T) {
	if got := vulngate.GroupByID(nil); len(got) != 0 {
		t.Errorf("got %d group(s), want 0", len(got))
	}
}

// The property the headline number exists for: it must move with exposure,
// not with how many places happen to call the vulnerable symbol. #348's
// complaint was that "155" moved when call sites moved.
func TestGroupByID_CountIsStableAsCallSitesChange(t *testing.T) {
	one := []vulngate.Finding{goFinding("GO-2026-6603", vulngate.ReachCalled, "x")}
	many := []vulngate.Finding{
		goFinding("GO-2026-6603", vulngate.ReachCalled, "x"),
		goFinding("GO-2026-6603", vulngate.ReachCalled, "x"),
		goFinding("GO-2026-6603", vulngate.ReachCalled, "x"),
		goFinding("GO-2026-6603", vulngate.ReachCalled, "x"),
	}

	if got, want := len(vulngate.GroupByID(one)), len(vulngate.GroupByID(many)); got != want {
		t.Errorf("advisory count changed with call-site count: %d vs %d", want, got)
	}
	if sites := vulngate.GroupByID(many)[0].Sites; sites != 4 {
		t.Errorf("Sites = %d, want 4 — the call-site count is still reported, just not as the headline", sites)
	}
}

// The grouped line is what a reviewer reads, so it has to say the things
// #348 asked for: the id, the reach/severity, the summary, and how many
// sites it collapsed.
func TestGroupString_SaysWhatItCollapsed(t *testing.T) {
	g := vulngate.GroupByID([]vulngate.Finding{
		goFinding("GO-2026-6603", vulngate.ReachCalled, "HTTP/2 server memory exhaustion"),
		goFinding("GO-2026-6603", vulngate.ReachCalled, "HTTP/2 server memory exhaustion"),
	})[0]

	s := g.String()
	for _, want := range []string{"GO-2026-6603", "called", "HTTP/2 server memory exhaustion", "2"} {
		if !contains(s, want) {
			t.Errorf("String() = %q, missing %q", s, want)
		}
	}
}

// A single-site group must NOT claim a count — "(1 site)" on every line is
// the noise this ticket is about, in a different costume.
func TestGroupString_OmitsTheCountForOneSite(t *testing.T) {
	g := vulngate.GroupByID([]vulngate.Finding{
		goFinding("GO-2026-6617", vulngate.ReachCalled, "HPACK"),
	})[0]

	if s := g.String(); contains(s, "site") {
		t.Errorf("String() = %q, should not mention sites for a single-site group", s)
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (func() bool {
		for i := 0; i+len(needle) <= len(haystack); i++ {
			if haystack[i:i+len(needle)] == needle {
				return true
			}
		}
		return false
	})()
}
