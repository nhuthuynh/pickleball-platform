// End-to-end tests for the gate binary, driving it the way the Makefile
// does: real report files in, exit code and printed report out.
//
// This package had NO test at all — `make gate-coverage`'s "no test
// function" NOTE listed it (#328's blind spot). The grouped report (#348)
// is the right occasion to add one, because the thing that went wrong was
// in the OUTPUT rather than the verdict: the gate's pass/fail was correct
// throughout, while its summary counted call sites and reported eight
// advisories as "155 new gating finding(s)". tools/vulngate's own tests
// cover GroupByID; only a test at this level can see what the binary
// actually prints, and the exit code is something only a subprocess can
// observe — the same reason cmd/docsindex's test builds and runs itself.
package main_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// buildGate compiles the gate to a scratch path. `-o` is not optional:
// `go build ./cmd/vulngate` with no `-o` drops a 3 MB executable at the
// repository root named after the package, which is how a compiled binary
// got committed at T65.4 (see `make binary-check`).
func buildGate(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "vulngate")
	out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput()
	if err != nil {
		t.Fatalf("building the gate: %v\n%s", err, out)
	}
	return bin
}

func write(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("writing %s: %v", name, err)
	}
	return path
}

const emptyBaseline = `{"accepted": []}`

// govulncheckReport builds a report in the streaming-JSON shape the real
// tool emits: one `osv` object describing the advisory, then one `finding`
// object per call site. `sites` of them, all for the same advisory — which
// is exactly the shape that produced #348's "155".
func govulncheckReport(id, summary string, sites int) string {
	var b strings.Builder
	b.WriteString(`{"osv":{"id":"` + id + `","summary":"` + summary + `"}}` + "\n")
	for i := 0; i < sites; i++ {
		b.WriteString(`{"finding":{"osv":"` + id + `","trace":[{"module":"stdlib","package":"net/http","function":"Handler"}]}}` + "\n")
	}
	return b.String()
}

func runGate(t *testing.T, args ...string) (output string, exitCode int) {
	t.Helper()
	out, err := exec.Command(buildGate(t), args...).CombinedOutput()
	if err == nil {
		return string(out), 0
	}
	var exit *exec.ExitError
	if !errors.As(err, &exit) {
		// Not the program exiting non-zero — it could not be run at all.
		t.Fatalf("running the gate: %v\n%s", err, out)
	}
	return string(out), exit.ExitCode()
}

// The headline #348 is about: one advisory reported at many call sites is
// ONE line and ONE advisory, with the site count beside it rather than as
// the number.
func TestGate_ReportsOneLinePerAdvisoryNotPerCallSite(t *testing.T) {
	dir := t.TempDir()
	base := write(t, dir, "baseline.json", emptyBaseline)
	report := write(t, dir, "govulncheck.json",
		govulncheckReport("GO-2026-6603", "HTTP/2 server memory exhaustion", 61))

	out, code := runGate(t, "-baseline", base, "-go", report)

	if code != 1 {
		t.Fatalf("exit code = %d, want 1 (a called advisory must fail the gate)\n%s", code, out)
	}
	if n := strings.Count(out, "GO-2026-6603"); n != 1 {
		t.Errorf("the advisory is printed %d times, want 1 — this is #348:\n%s", n, out)
	}
	if !strings.Contains(out, "1 advisory, 61 call sites") {
		t.Errorf("summary does not count advisories and sites separately:\n%s", out)
	}
	// The old wording, which is what made the number unreadable.
	if strings.Contains(out, "61 new gating finding") {
		t.Errorf("the summary still counts call sites as findings:\n%s", out)
	}
}

// Distinct advisories must still be listed separately — a grouping that hid
// findings would be far worse than one that over-counted them.
func TestGate_KeepsDistinctAdvisoriesVisible(t *testing.T) {
	dir := t.TempDir()
	base := write(t, dir, "baseline.json", emptyBaseline)
	report := write(t, dir, "govulncheck.json",
		govulncheckReport("GO-2026-6603", "Trailer headers", 3)+
			govulncheckReport("GO-2026-6617", "HPACK encoder race", 2))

	out, code := runGate(t, "-baseline", base, "-go", report)

	if code != 1 {
		t.Fatalf("exit code = %d, want 1\n%s", code, out)
	}
	for _, id := range []string{"GO-2026-6603", "GO-2026-6617"} {
		if n := strings.Count(out, id); n != 1 {
			t.Errorf("%s printed %d times, want 1:\n%s", id, n, out)
		}
	}
	if !strings.Contains(out, "2 advisories, 5 call sites") {
		t.Errorf("summary wrong:\n%s", out)
	}
}

// One site per advisory must not grow a redundant "1 call sites" tail —
// that is the same noise in a different costume.
func TestGate_OmitsTheSiteCountWhenItAddsNothing(t *testing.T) {
	dir := t.TempDir()
	base := write(t, dir, "baseline.json", emptyBaseline)
	report := write(t, dir, "govulncheck.json", govulncheckReport("GO-2026-6603", "Trailer headers", 1))

	out, _ := runGate(t, "-baseline", base, "-go", report)

	if !strings.Contains(out, "1 advisory.") {
		t.Errorf("want a bare advisory count:\n%s", out)
	}
	if strings.Contains(out, "call sites") {
		t.Errorf("site count printed when every advisory had one site:\n%s", out)
	}
}

// The verdict must not change: grouping is a reporting change, and a gate
// that started passing because of it would be the worst possible outcome.
func TestGate_PassesWhenNothingGates(t *testing.T) {
	dir := t.TempDir()
	base := write(t, dir, "baseline.json", emptyBaseline)
	// A trace with no function is `required`, which does not gate.
	report := write(t, dir, "govulncheck.json",
		`{"osv":{"id":"GO-2026-6866","summary":"x/net/html"}}`+"\n"+
			`{"finding":{"osv":"GO-2026-6866","trace":[{"module":"golang.org/x/net"}]}}`+"\n")

	out, code := runGate(t, "-baseline", base, "-go", report)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 — a `required` finding does not gate\n%s", code, out)
	}
	if !strings.Contains(out, "PASS") {
		t.Errorf("want PASS:\n%s", out)
	}
}

// Running the gate with no report at all must fail loudly rather than
// report success while checking nothing — the binary's own doc comment
// calls that worse than not running it, and nothing tested it.
func TestGate_RefusesToRunWithNoReports(t *testing.T) {
	dir := t.TempDir()
	base := write(t, dir, "baseline.json", emptyBaseline)

	out, code := runGate(t, "-baseline", base)

	if code == 0 {
		t.Fatalf("exit code = 0 with no reports; the gate reported success while checking nothing:\n%s", out)
	}
}
