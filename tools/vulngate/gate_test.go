package vulngate

import (
	"strings"
	"testing"
)

// --- govulncheck fixtures -------------------------------------------------
//
// govulncheck -format json emits a stream of JSON objects, each carrying
// exactly one of "config" / "progress" / "osv" / "finding". A finding's
// `trace` is ordered vulnerable-symbol-first; whether the deepest frame
// carries a `function` is what distinguishes a vuln this code actually
// CALLS from one merely imported or required.
//
// IMPORTANT (honesty note, CLAUDE.md rule 10): these fixtures are
// hand-written from the documented govulncheck JSON schema, NOT captured
// from a live run — vuln.go.dev is blocked by this environment's egress
// policy, so no real govulncheck output could be obtained here. The npm
// fixtures below ARE the real schema, captured from `npm audit --json`.

const govulnCalled = `
{"config":{"protocol_version":"v1.0.0","scanner_name":"govulncheck"}}
{"progress":{"message":"Scanning your code and 123 packages..."}}
{"osv":{"id":"GO-2021-0113","summary":"Out-of-bounds read in golang.org/x/text"}}
{"finding":{"osv":"GO-2021-0113","fixed_version":"v0.3.7","trace":[
  {"module":"golang.org/x/text","version":"v0.3.5","package":"golang.org/x/text/language","function":"Parse"},
  {"module":"github.com/nhuthuynh/white-label","package":"github.com/nhuthuynh/white-label/internal/booking/app","function":"CreateBooking"}
]}}
`

const govulnImportedOnly = `
{"osv":{"id":"GO-2022-0999","summary":"Something in an imported-but-uncalled package"}}
{"finding":{"osv":"GO-2022-0999","fixed_version":"v1.2.3","trace":[
  {"module":"example.com/dep","version":"v1.0.0","package":"example.com/dep/sub"}
]}}
`

const govulnRequiredOnly = `
{"osv":{"id":"GO-2023-0001","summary":"Module required but never imported"}}
{"finding":{"osv":"GO-2023-0001","trace":[{"module":"example.com/other","version":"v0.1.0"}]}}
`

// --- npm audit fixtures ---------------------------------------------------
//
// Real `npm audit --json` v2 shape, confirmed by running it against web/.

const npmClean = `{"auditReportVersion":2,"vulnerabilities":{},"metadata":{"vulnerabilities":{"info":0,"low":0,"moderate":0,"high":0,"critical":0,"total":0}}}`

const npmMixed = `{
 "auditReportVersion": 2,
 "vulnerabilities": {
   "bad-pkg": {"name":"bad-pkg","severity":"critical","via":[{"source":1,"name":"bad-pkg","title":"RCE","url":"https://github.com/advisories/GHSA-aaaa-bbbb-cccc","severity":"critical"}],"range":"<1.0.1","fixAvailable":true},
   "meh-pkg": {"name":"meh-pkg","severity":"high","via":[{"source":2,"name":"meh-pkg","title":"Prototype pollution","url":"https://github.com/advisories/GHSA-dddd-eeee-ffff","severity":"high"}],"range":"<2.0.0","fixAvailable":true},
   "minor-pkg": {"name":"minor-pkg","severity":"moderate","via":[{"source":3,"name":"minor-pkg","title":"ReDoS","url":"https://github.com/advisories/GHSA-gggg-hhhh-iiii","severity":"moderate"}],"range":"<3.0.0","fixAvailable":true},
   "chained-pkg": {"name":"chained-pkg","severity":"high","via":["bad-pkg"],"range":"*","fixAvailable":false}
 },
 "metadata": {"vulnerabilities":{"info":0,"low":0,"moderate":1,"high":2,"critical":1,"total":4}}
}`

func TestParseGovulncheck(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantIDs   []string
		wantReach map[string]Reach
	}{
		{
			name:      "called vulnerability is parsed with reach=called",
			input:     govulnCalled,
			wantIDs:   []string{"GO-2021-0113"},
			wantReach: map[string]Reach{"GO-2021-0113": ReachCalled},
		},
		{
			name:      "imported-but-not-called yields reach=imported",
			input:     govulnImportedOnly,
			wantIDs:   []string{"GO-2022-0999"},
			wantReach: map[string]Reach{"GO-2022-0999": ReachImported},
		},
		{
			name:      "module-only trace yields reach=required",
			input:     govulnRequiredOnly,
			wantIDs:   []string{"GO-2023-0001"},
			wantReach: map[string]Reach{"GO-2023-0001": ReachRequired},
		},
		{
			name:    "empty input yields no findings",
			input:   "",
			wantIDs: nil,
		},
		{
			name:    "config and progress lines alone yield no findings",
			input:   `{"config":{"scanner_name":"govulncheck"}}` + "\n" + `{"progress":{"message":"scanning"}}`,
			wantIDs: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseGovulncheck(strings.NewReader(tc.input))
			if err != nil {
				t.Fatalf("ParseGovulncheck() error = %v, want nil", err)
			}
			if len(got) != len(tc.wantIDs) {
				t.Fatalf("got %d findings %v, want %d %v", len(got), ids(got), len(tc.wantIDs), tc.wantIDs)
			}
			for i, want := range tc.wantIDs {
				if got[i].ID != want {
					t.Errorf("finding[%d].ID = %q, want %q", i, got[i].ID, want)
				}
				if got[i].Scanner != ScannerGo {
					t.Errorf("finding[%d].Scanner = %q, want %q", i, got[i].Scanner, ScannerGo)
				}
				if wantReach, ok := tc.wantReach[want]; ok && got[i].Reach != wantReach {
					t.Errorf("finding[%d].Reach = %q, want %q", i, got[i].Reach, wantReach)
				}
			}
		})
	}
}

func TestParseGovulncheckRejectsMalformedJSON(t *testing.T) {
	if _, err := ParseGovulncheck(strings.NewReader("{not json")); err == nil {
		t.Fatal("ParseGovulncheck() error = nil, want a parse error — a malformed scanner report must fail loudly, never be read as 'no vulnerabilities'")
	}
}

// TestParseGovulncheckTruncatedRunLooksLikeZeroFindings documents a real
// incident (PR #95 review): when govulncheck can't reach vuln.go.dev, it
// still writes a well-formed `config`/SBOM preamble to stdout before it
// dies — no findings, no OSV entries, syntactically valid NDJSON. Parsed in
// isolation, that is genuinely indistinguishable from "the scan ran and
// found nothing", and this test pins that ParseGovulncheck does exactly
// that (no error, zero findings) rather than guessing.
//
// That is why this is NOT a bug in ParseGovulncheck to "fix": nothing in
// the byte stream says the process died partway through — only the
// process's own exit code carries that information, and this function
// never sees it. The Jenkinsfile's Security scan stage is what has to
// carry that signal (build/govulncheck.exit, checked before this file's
// output is ever trusted) — this test exists so a future reader doesn't
// try to solve the problem at the wrong layer a second time.
func TestParseGovulncheckTruncatedRunLooksLikeZeroFindings(t *testing.T) {
	truncated := `{"config":{"protocol_version":"v1","scanner_name":"govulncheck","scanner_version":"1.1.3","db":"https://vuln.go.dev","db_last_modified":"2026-08-01T00:00:00Z","go_version":"go1.25.0"}}
{"progress":{"message":"Scanning your code and 42 packages across 6 dependent modules for known vulnerabilities..."}}
`
	got, err := ParseGovulncheck(strings.NewReader(truncated))
	if err != nil {
		t.Fatalf("ParseGovulncheck() error = %v, want nil — a truncated-but-well-formed stream is not malformed JSON", err)
	}
	if len(got) != 0 {
		t.Fatalf("got %d findings, want 0 — a config/progress-only stream genuinely has none to report", len(got))
	}
}

func TestParseNPMAudit(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantIDs  []string
		wantSevs map[string]Severity
	}{
		{
			name:    "clean report yields no findings",
			input:   npmClean,
			wantIDs: nil,
		},
		{
			name:    "each advisory is one finding, keyed by package",
			input:   npmMixed,
			wantIDs: []string{"bad-pkg", "chained-pkg", "meh-pkg", "minor-pkg"},
			wantSevs: map[string]Severity{
				"bad-pkg":     SevCritical,
				"meh-pkg":     SevHigh,
				"minor-pkg":   SevModerate,
				"chained-pkg": SevHigh,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseNPMAudit(strings.NewReader(tc.input))
			if err != nil {
				t.Fatalf("ParseNPMAudit() error = %v, want nil", err)
			}
			if len(got) != len(tc.wantIDs) {
				t.Fatalf("got %d findings %v, want %d %v", len(got), ids(got), len(tc.wantIDs), tc.wantIDs)
			}
			// Parser sorts by ID so the output is deterministic (npm's
			// `vulnerabilities` is a map, and Go map order is random).
			for i, want := range tc.wantIDs {
				if got[i].ID != want {
					t.Errorf("finding[%d].ID = %q, want %q", i, got[i].ID, want)
				}
				if got[i].Scanner != ScannerNPM {
					t.Errorf("finding[%d].Scanner = %q, want %q", i, got[i].Scanner, ScannerNPM)
				}
				if wantSev, ok := tc.wantSevs[want]; ok && got[i].Severity != wantSev {
					t.Errorf("finding[%d].Severity = %q, want %q", i, got[i].Severity, wantSev)
				}
			}
		})
	}
}

func TestParseNPMAuditRejectsMalformedJSON(t *testing.T) {
	if _, err := ParseNPMAudit(strings.NewReader("{not json")); err == nil {
		t.Fatal("ParseNPMAudit() error = nil, want a parse error — a malformed scanner report must fail loudly, never be read as 'no vulnerabilities'")
	}
}

func TestFindingGating(t *testing.T) {
	tests := []struct {
		name    string
		finding Finding
		want    bool
	}{
		{"npm critical gates", Finding{Scanner: ScannerNPM, Severity: SevCritical}, true},
		{"npm high gates", Finding{Scanner: ScannerNPM, Severity: SevHigh}, true},
		{"npm moderate does not gate", Finding{Scanner: ScannerNPM, Severity: SevModerate}, false},
		{"npm low does not gate", Finding{Scanner: ScannerNPM, Severity: SevLow}, false},
		{"npm info does not gate", Finding{Scanner: ScannerNPM, Severity: SevInfo}, false},
		// Go's vuln DB does not carry CVSS severity, so reachability is the
		// gating signal: a vulnerability this code actually calls is the
		// actionable equivalent of HIGH/CRITICAL. See gate.go's doc comment.
		{"go called gates", Finding{Scanner: ScannerGo, Reach: ReachCalled}, true},
		{"go imported does not gate", Finding{Scanner: ScannerGo, Reach: ReachImported}, false},
		{"go required does not gate", Finding{Scanner: ScannerGo, Reach: ReachRequired}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.finding.Gating(); got != tc.want {
				t.Errorf("Gating() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestEvaluate(t *testing.T) {
	baseline := Baseline{Accepted: []BaselineEntry{
		{Scanner: ScannerNPM, ID: "bad-pkg", Reason: "no fix published upstream", AddedOn: "2026-08-05"},
		{Scanner: ScannerGo, ID: "GO-2021-0113", Reason: "tracked in HANDOFF", AddedOn: "2026-08-05"},
	}}

	tests := []struct {
		name          string
		findings      []Finding
		baseline      Baseline
		wantNew       []string
		wantBaselined []string
		wantIgnored   []string
		wantFailed    bool
	}{
		{
			name:       "no findings passes",
			findings:   nil,
			baseline:   baseline,
			wantFailed: false,
		},
		{
			name:        "sub-threshold findings are ignored, not failed",
			findings:    []Finding{{Scanner: ScannerNPM, ID: "minor-pkg", Severity: SevModerate}},
			baseline:    baseline,
			wantIgnored: []string{"minor-pkg"},
			wantFailed:  false,
		},
		{
			name:          "a baselined gating finding does not fail the build",
			findings:      []Finding{{Scanner: ScannerNPM, ID: "bad-pkg", Severity: SevCritical}},
			baseline:      baseline,
			wantBaselined: []string{"bad-pkg"},
			wantFailed:    false,
		},
		{
			name:       "a NEW gating finding fails the build",
			findings:   []Finding{{Scanner: ScannerNPM, ID: "brand-new-pkg", Severity: SevHigh}},
			baseline:   baseline,
			wantNew:    []string{"brand-new-pkg"},
			wantFailed: true,
		},
		{
			name: "baseline is scanner-scoped — same ID under a different scanner is still new",
			// Guards against a baseline entry silently covering a
			// same-named finding from the other scanner.
			findings:   []Finding{{Scanner: ScannerGo, ID: "bad-pkg", Reach: ReachCalled}},
			baseline:   baseline,
			wantNew:    []string{"bad-pkg"},
			wantFailed: true,
		},
		{
			name: "mixed report partitions correctly and fails on the new one",
			findings: []Finding{
				{Scanner: ScannerNPM, ID: "bad-pkg", Severity: SevCritical},
				{Scanner: ScannerNPM, ID: "brand-new-pkg", Severity: SevHigh},
				{Scanner: ScannerNPM, ID: "minor-pkg", Severity: SevModerate},
				{Scanner: ScannerGo, ID: "GO-2021-0113", Reach: ReachCalled},
				{Scanner: ScannerGo, ID: "GO-2022-0999", Reach: ReachImported},
			},
			baseline:      baseline,
			wantNew:       []string{"brand-new-pkg"},
			wantBaselined: []string{"bad-pkg", "GO-2021-0113"},
			wantIgnored:   []string{"minor-pkg", "GO-2022-0999"},
			wantFailed:    true,
		},
		{
			name:       "an empty baseline lets every gating finding through as new",
			findings:   []Finding{{Scanner: ScannerNPM, ID: "bad-pkg", Severity: SevCritical}},
			baseline:   Baseline{},
			wantNew:    []string{"bad-pkg"},
			wantFailed: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Evaluate(tc.findings, tc.baseline)
			assertIDs(t, "New", got.New, tc.wantNew)
			assertIDs(t, "Baselined", got.Baselined, tc.wantBaselined)
			assertIDs(t, "Ignored", got.Ignored, tc.wantIgnored)
			if got.Failed() != tc.wantFailed {
				t.Errorf("Failed() = %v, want %v", got.Failed(), tc.wantFailed)
			}
		})
	}
}

func TestLoadBaseline(t *testing.T) {
	const doc = `{"accepted":[
      {"scanner":"npm-audit","id":"bad-pkg","reason":"no upstream fix","added_on":"2026-08-05"}
    ]}`
	got, err := LoadBaseline(strings.NewReader(doc))
	if err != nil {
		t.Fatalf("LoadBaseline() error = %v, want nil", err)
	}
	if len(got.Accepted) != 1 {
		t.Fatalf("got %d accepted entries, want 1", len(got.Accepted))
	}
	if got.Accepted[0].ID != "bad-pkg" || got.Accepted[0].Reason != "no upstream fix" {
		t.Errorf("got %+v, want id=bad-pkg reason=%q", got.Accepted[0], "no upstream fix")
	}
}

func TestLoadBaselineRequiresAReason(t *testing.T) {
	// "Baseline them explicitly, don't just suppress silently" — an entry
	// with no reason is a silent suppression, so it is rejected outright.
	const doc = `{"accepted":[{"scanner":"npm-audit","id":"bad-pkg","added_on":"2026-08-05"}]}`
	if _, err := LoadBaseline(strings.NewReader(doc)); err == nil {
		t.Fatal("LoadBaseline() error = nil, want an error for a baseline entry with no reason")
	}
}

func TestLoadBaselineRejectsUnknownScanner(t *testing.T) {
	const doc = `{"accepted":[{"scanner":"nope","id":"x","reason":"typo in scanner name"}]}`
	if _, err := LoadBaseline(strings.NewReader(doc)); err == nil {
		t.Fatal("LoadBaseline() error = nil, want an error for an unknown scanner — a typo'd scanner name would silently never match, making the entry dead weight")
	}
}

// --- helpers --------------------------------------------------------------

func ids(fs []Finding) []string {
	out := make([]string, 0, len(fs))
	for _, f := range fs {
		out = append(out, f.ID)
	}
	return out
}

func assertIDs(t *testing.T, label string, got []Finding, want []string) {
	t.Helper()
	gotIDs := ids(got)
	if len(gotIDs) != len(want) {
		t.Errorf("%s = %v, want %v", label, gotIDs, want)
		return
	}
	for i := range want {
		if gotIDs[i] != want[i] {
			t.Errorf("%s = %v, want %v", label, gotIDs, want)
			return
		}
	}
}
