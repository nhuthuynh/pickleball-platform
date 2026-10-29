// Package vulngate turns raw dependency-scanner output into a build
// decision: fail on NEW high-severity findings, tolerate explicitly
// baselined ones, ignore sub-threshold noise.
//
// Why this exists as real (tested) code rather than a shell one-liner:
// the "fail on new but not on pre-existing" rule is genuine logic with
// genuine edge cases (scanner-scoped baselines, malformed reports that
// must not read as "clean"), and CLAUDE.md rule 1 wants logic like that
// under test. The pipeline shells out to `vulngate` in the Security scan
// stage; see the Jenkinsfile and `make security`.
//
// # Severity model
//
// npm audit reports a real severity per advisory (info/low/moderate/
// high/critical), so npm gating is literal: high and critical gate.
//
// govulncheck does NOT: the Go vulnerability database does not carry a
// CVSS score per entry, so there is no "HIGH/CRITICAL" field to read.
// What govulncheck reports instead — and what actually matters — is
// REACHABILITY: whether your code calls the vulnerable symbol, merely
// imports the package, or just requires the module. A called
// vulnerability is the actionable equivalent of high/critical (and is a
// stricter, better-targeted signal than CVSS, which knows nothing about
// whether you reach the affected code path). So: called gates; imported
// and required are reported but do not gate.
//
// This mapping is a deliberate, documented choice, not an oversight —
// see docs/adr/0011-ci-pipeline-and-security-gating.md.
package vulngate

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
)

// Scanner identifies which tool produced a finding. Findings are always
// scanner-scoped: a baseline entry for one scanner never silences a
// same-named finding from the other.
type Scanner string

const (
	ScannerGo  Scanner = "govulncheck"
	ScannerNPM Scanner = "npm-audit"
)

// Severity is npm audit's advisory severity. Empty for Go findings,
// which use Reach instead (see the package doc).
type Severity string

const (
	SevCritical Severity = "critical"
	SevHigh     Severity = "high"
	SevModerate Severity = "moderate"
	SevLow      Severity = "low"
	SevInfo     Severity = "info"
)

// Reach is govulncheck's reachability verdict. Empty for npm findings.
type Reach string

const (
	// ReachCalled means the scan traced a call into the vulnerable
	// symbol — this code actually executes the affected path.
	ReachCalled Reach = "called"
	// ReachImported means the vulnerable package is imported but no call
	// into the affected symbol was traced.
	ReachImported Reach = "imported"
	// ReachRequired means the module is in the dependency graph but its
	// vulnerable package is never imported.
	ReachRequired Reach = "required"
)

// Finding is one vulnerability from one scanner, normalised across the
// two report formats.
type Finding struct {
	Scanner  Scanner
	ID       string // GO-YYYY-NNNN for Go; the npm package name for npm
	Severity Severity
	Reach    Reach
	Detail   string
}

// Gating reports whether this finding is severe enough to fail a build
// if it is not baselined. See the package doc for the severity model.
func (f Finding) Gating() bool {
	switch f.Scanner {
	case ScannerNPM:
		return f.Severity == SevHigh || f.Severity == SevCritical
	case ScannerGo:
		return f.Reach == ReachCalled
	default:
		// Unknown scanner: gate rather than silently pass. A finding we
		// cannot classify is not a finding we may ignore.
		return true
	}
}

func (f Finding) String() string {
	grade := string(f.Severity)
	if grade == "" {
		grade = string(f.Reach)
	}
	s := fmt.Sprintf("[%s] %s (%s)", f.Scanner, f.ID, grade)
	if f.Detail != "" {
		s += ": " + f.Detail
	}
	return s
}

// --- govulncheck ----------------------------------------------------------

// govulnMessage is one line of `govulncheck -format json` output. Each
// object carries exactly one populated field.
type govulnMessage struct {
	OSV     *govulnOSV     `json:"osv"`
	Finding *govulnFinding `json:"finding"`
}

type govulnOSV struct {
	ID      string `json:"id"`
	Summary string `json:"summary"`
}

type govulnFinding struct {
	OSV          string        `json:"osv"`
	FixedVersion string        `json:"fixed_version"`
	Trace        []govulnFrame `json:"trace"`
}

type govulnFrame struct {
	Module   string `json:"module"`
	Version  string `json:"version"`
	Package  string `json:"package"`
	Function string `json:"function"`
}

// ParseGovulncheck reads the newline-delimited JSON stream emitted by
// `govulncheck -format json`.
//
// A malformed stream is an error, never an empty result — "the scanner
// output did not parse" must not be indistinguishable from "there are no
// vulnerabilities".
func ParseGovulncheck(r io.Reader) ([]Finding, error) {
	dec := json.NewDecoder(r)
	summaries := map[string]string{}
	var findings []Finding

	for {
		var msg govulnMessage
		if err := dec.Decode(&msg); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("parsing govulncheck JSON: %w", err)
		}
		if msg.OSV != nil {
			summaries[msg.OSV.ID] = msg.OSV.Summary
		}
		if msg.Finding != nil {
			findings = append(findings, Finding{
				Scanner: ScannerGo,
				ID:      msg.Finding.OSV,
				Reach:   reachOf(msg.Finding.Trace),
			})
		}
	}

	// Attach summaries in a second pass: the `osv` line for a given ID can
	// legally arrive after the `finding` that references it.
	for i := range findings {
		findings[i].Detail = summaries[findings[i].ID]
	}
	return findings, nil
}

// reachOf derives reachability from the trace's most specific frame. The
// trace is ordered vulnerable-symbol-first, so frame 0 is the deepest.
func reachOf(trace []govulnFrame) Reach {
	if len(trace) == 0 {
		return ReachRequired
	}
	switch {
	case trace[0].Function != "":
		return ReachCalled
	case trace[0].Package != "":
		return ReachImported
	default:
		return ReachRequired
	}
}

// --- npm audit ------------------------------------------------------------

// npmAuditReport is the `npm audit --json` v2 document. Confirmed against
// real output from web/ rather than assumed.
type npmAuditReport struct {
	AuditReportVersion int                       `json:"auditReportVersion"`
	Vulnerabilities    map[string]npmVulnerablty `json:"vulnerabilities"`
}

type npmVulnerablty struct {
	Name     string          `json:"name"`
	Severity string          `json:"severity"`
	Via      json.RawMessage `json:"via"`
	Range    string          `json:"range"`
}

// npmVia is one entry of the `via` array when it is an object (a direct
// advisory). `via` entries may also be plain strings naming another
// package, i.e. "vulnerable only because of a transitive dep".
type npmVia struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

// ParseNPMAudit reads `npm audit --json` output. Findings come back
// sorted by package name so the report is deterministic (npm keys
// vulnerabilities by package in a JSON object, and Go map iteration
// order is deliberately random).
func ParseNPMAudit(r io.Reader) ([]Finding, error) {
	var report npmAuditReport
	if err := json.NewDecoder(r).Decode(&report); err != nil {
		return nil, fmt.Errorf("parsing npm audit JSON: %w", err)
	}

	var findings []Finding
	for name, v := range report.Vulnerabilities {
		id := v.Name
		if id == "" {
			id = name
		}
		findings = append(findings, Finding{
			Scanner:  ScannerNPM,
			ID:       id,
			Severity: Severity(v.Severity),
			Detail:   npmDetail(v),
		})
	}
	sort.Slice(findings, func(i, j int) bool { return findings[i].ID < findings[j].ID })
	return findings, nil
}

// npmDetail summarises why a package is flagged: the advisory title for a
// direct advisory, or the upstream package name for a transitive one.
func npmDetail(v npmVulnerablty) string {
	if len(v.Via) == 0 {
		return ""
	}
	var objs []npmVia
	if err := json.Unmarshal(v.Via, &objs); err == nil {
		var titles []string
		for _, o := range objs {
			if o.Title != "" {
				titles = append(titles, o.Title)
			}
		}
		if len(titles) > 0 {
			return strings.Join(titles, "; ")
		}
	}
	var names []string
	if err := json.Unmarshal(v.Via, &names); err == nil && len(names) > 0 {
		return "via " + strings.Join(names, ", ")
	}
	return ""
}

// --- baseline -------------------------------------------------------------

// BaselineEntry is one explicitly accepted finding. Reason is mandatory:
// the point of a baseline is that every suppression is justified in
// writing and reviewable in a diff.
type BaselineEntry struct {
	Scanner Scanner `json:"scanner"`
	ID      string  `json:"id"`
	Reason  string  `json:"reason"`
	AddedOn string  `json:"added_on"`
}

// Baseline is the set of findings that are known, accepted, and must not
// fail the build.
type Baseline struct {
	Accepted []BaselineEntry `json:"accepted"`
}

func (b Baseline) has(f Finding) bool {
	for _, e := range b.Accepted {
		if e.Scanner == f.Scanner && e.ID == f.ID {
			return true
		}
	}
	return false
}

// LoadBaseline reads and validates the baseline document.
//
// Validation is strict on purpose. An entry with no reason is a silent
// suppression, and an entry naming an unknown scanner can never match
// anything — both look like protection while providing none, so both are
// hard errors rather than warnings.
func LoadBaseline(r io.Reader) (Baseline, error) {
	var b Baseline
	if err := json.NewDecoder(r).Decode(&b); err != nil {
		return Baseline{}, fmt.Errorf("parsing baseline: %w", err)
	}
	for i, e := range b.Accepted {
		if e.ID == "" {
			return Baseline{}, fmt.Errorf("baseline entry %d: id is required", i)
		}
		if strings.TrimSpace(e.Reason) == "" {
			return Baseline{}, fmt.Errorf("baseline entry %d (%s): a reason is required — baselining must be explicit, never a silent suppression", i, e.ID)
		}
		if e.Scanner != ScannerGo && e.Scanner != ScannerNPM {
			return Baseline{}, fmt.Errorf("baseline entry %d (%s): unknown scanner %q, want %q or %q", i, e.ID, e.Scanner, ScannerGo, ScannerNPM)
		}
	}
	return b, nil
}

// --- evaluation -----------------------------------------------------------

// Result partitions a scan into the three buckets a reviewer cares
// about: what is new and must be fixed, what is known and accepted, and
// what is below the gating threshold.
type Result struct {
	New       []Finding
	Baselined []Finding
	Ignored   []Finding
}

// Failed reports whether the build should fail.
func (r Result) Failed() bool { return len(r.New) > 0 }

// Evaluate partitions findings against the baseline, preserving input
// order within each bucket.
func Evaluate(findings []Finding, b Baseline) Result {
	var res Result
	for _, f := range findings {
		switch {
		case !f.Gating():
			res.Ignored = append(res.Ignored, f)
		case b.has(f):
			res.Baselined = append(res.Baselined, f)
		default:
			res.New = append(res.New, f)
		}
	}
	return res
}

// --- grouping -------------------------------------------------------------

// Group is one advisory, with however many call sites reported it (#348).
//
// Why this type exists: govulncheck emits one record per call site, so the
// gate's first real run reported eight advisories as "155 new gating
// finding(s)" — sixty-odd identical copies of one line, which had to be
// deduplicated by eye to learn that four of the eight were a single
// toolchain bump. The number was also useless as a trend signal, because it
// moved when call sites moved rather than when exposure changed.
//
// Nothing is lost by collapsing them. A Finding carries Scanner, ID,
// Severity, Reach and Detail, and for one ID every one of those is
// identical across sites except Reach — the trace that distinguishes the
// sites is read by reachOf and then discarded, never stored. So there is no
// per-site detail to keep behind a flag, which is what #348's "keep the
// per-site detail available" would otherwise have asked for.
type Group struct {
	Scanner  Scanner
	ID       string
	Severity Severity
	Reach    Reach
	Detail   string
	// Sites is how many findings collapsed into this group: call sites for
	// govulncheck, advisories-per-package for npm.
	Sites int
}

func (g Group) String() string {
	grade := string(g.Severity)
	if grade == "" {
		grade = string(g.Reach)
	}
	s := fmt.Sprintf("[%s] %s (%s)", g.Scanner, g.ID, grade)
	if g.Detail != "" {
		s += ": " + g.Detail
	}
	// Only when there is something to say. "(1 site)" on every line is the
	// noise this type exists to remove, in a different costume.
	if g.Sites > 1 {
		s += fmt.Sprintf(" [%d call sites]", g.Sites)
	}
	return s
}

// reachRank orders reachability by how much it should worry a reader.
// Gating() keys on ReachCalled, so a group covering both an imported and a
// called site must report `called` — describing a called vulnerability as
// merely imported would understate exactly the thing the gate acts on.
func reachRank(r Reach) int {
	switch r {
	case ReachCalled:
		return 3
	case ReachImported:
		return 2
	case ReachRequired:
		return 1
	default:
		return 0
	}
}

// severityRank orders npm severities the same way, for the same reason.
func severityRank(s Severity) int {
	switch s {
	case SevCritical:
		return 5
	case SevHigh:
		return 4
	case SevModerate:
		return 3
	case SevLow:
		return 2
	case SevInfo:
		return 1
	default:
		return 0
	}
}

// GroupByID collapses findings that describe the same advisory, counting
// how many collapsed. Order follows first appearance, so a grouped report
// reads in the order the scanner produced — the property Evaluate also
// preserves.
//
// Keyed by (Scanner, ID), never ID alone: an npm finding's ID is a package
// name, and nothing stops one sharing a string with a Go advisory id.
// Baseline.has is scanner-scoped for the same reason.
func GroupByID(findings []Finding) []Group {
	type key struct {
		scanner Scanner
		id      string
	}
	index := make(map[key]int, len(findings))
	groups := make([]Group, 0, len(findings))

	for _, f := range findings {
		k := key{f.Scanner, f.ID}
		i, seen := index[k]
		if !seen {
			index[k] = len(groups)
			groups = append(groups, Group{
				Scanner:  f.Scanner,
				ID:       f.ID,
				Severity: f.Severity,
				Reach:    f.Reach,
				Detail:   f.Detail,
				Sites:    1,
			})
			continue
		}

		g := &groups[i]
		g.Sites++
		if reachRank(f.Reach) > reachRank(g.Reach) {
			g.Reach = f.Reach
		}
		if severityRank(f.Severity) > severityRank(g.Severity) {
			g.Severity = f.Severity
		}
		// govulncheck's `osv` line can arrive after the `finding` that
		// references it, and a report truncated mid-stream can leave some
		// blank — so take the first non-empty rather than the first.
		if g.Detail == "" {
			g.Detail = f.Detail
		}
	}
	return groups
}
