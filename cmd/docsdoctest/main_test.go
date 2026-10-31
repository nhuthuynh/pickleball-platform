package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The exit code is the only part of this program a Makefile can see, so it is
// the part worth testing — and testing it needs a subprocess. Same shape as
// cmd/docsindex/main_test.go, which exists because cmd/docsdoctest's sibling
// shipped its -statuses listing with no test at all (#328), which rules 1 and
// 8 do not allow.
//
// `go run` rather than a pre-built binary: building to a path inside the
// repository is what put a 3 MB executable at the root and caused
// `make binary-check` to exist.
func runGate(t *testing.T, root string, extra ...string) (stdout string, code int) {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	args := append([]string{"run", ".", "-root", root}, extra...)
	cmd := exec.Command("go", args...)
	cmd.Dir = wd

	var out, errOut strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	_ = cmd.Run()
	if cmd.ProcessState == nil {
		t.Fatalf("go run did not run: %s", errOut.String())
	}
	if c := cmd.ProcessState.ExitCode(); c == 2 {
		t.Fatalf("the gate's own machinery failed (exit 2): %s", errOut.String())
	}
	return out.String(), cmd.ProcessState.ExitCode()
}

func writeDoc(t *testing.T, body string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "doc.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestExitsZeroWhenEveryDocumentedCommandReproduces(t *testing.T) {
	root := writeDoc(t, "```console doctest\n$ printf '26\\n'\n26\n```\n")

	out, code := runGate(t, root)
	if code != 0 {
		t.Fatalf("exit = %d, want 0\n%s", code, out)
	}
	if !strings.Contains(out, "docs-doctest: OK") {
		t.Errorf("output does not report OK:\n%s", out)
	}
}

func TestExitsOneOnAStaleFigure(t *testing.T) {
	root := writeDoc(t, "```console doctest\n$ printf '26\\n'\n27\n```\n")

	out, code := runGate(t, root)
	if code != 1 {
		t.Fatalf("exit = %d, want 1\n%s", code, out)
	}
	for _, want := range []string{"doc.md:2", "FAIL", "27", "26"} {
		if !strings.Contains(out, want) {
			t.Errorf("output does not mention %q:\n%s", want, out)
		}
	}
}

// The guard that stops this gate being switched off silently: with every
// marker removed it must go red, not print OK over an empty run.
func TestExitsOneWhenNothingWasChecked(t *testing.T) {
	root := writeDoc(t, "```\n$ curl localhost:8080/v1/bookings\n{}\n```\n")

	out, code := runGate(t, root)
	if code != 1 {
		t.Fatalf("exit = %d, want 1\n%s", code, out)
	}
	if !strings.Contains(out, "no block was checked") {
		t.Errorf("output does not say it checked nothing:\n%s", out)
	}
}

// The blind spot must be visible in the gate's own stdout, not just in its
// source — following gate-coverage's untested-package NOTE.
func TestReportsUnmarkedBlocksWithoutFailing(t *testing.T) {
	root := writeDoc(t, "```console doctest\n$ printf 'x\\n'\nx\n```\n\n"+
		"```\n$ docker ps\nCONTAINER ID\n```\n")

	out, code := runGate(t, root)
	if code != 0 {
		t.Fatalf("exit = %d, want 0 — an unmarked block must not fail\n%s", code, out)
	}
	if !strings.Contains(out, "NOTE") || !strings.Contains(out, "1 documented command(s)") {
		t.Errorf("the blind spot is not counted:\n%s", out)
	}
	// Counted by default, listed on request — and the default must NOT list,
	// or on the real repository ~190 lines bury the findings.
	if strings.Contains(out, "docker ps") {
		t.Errorf("the default output lists the blind spot instead of counting it:\n%s", out)
	}

	listed, code := runGate(t, root, "-blind-spots")
	if code != 0 {
		t.Fatalf("exit = %d, want 0\n%s", code, listed)
	}
	if !strings.Contains(listed, "docker ps") {
		t.Errorf("-blind-spots does not list the unmarked command:\n%s", listed)
	}
}

func TestTimeoutFlagIsHonoured(t *testing.T) {
	root := writeDoc(t, "```console doctest\n$ sleep 30\n```\n")

	out, code := runGate(t, root, "-timeout", "200ms")
	if code != 1 {
		t.Fatalf("exit = %d, want 1\n%s", code, out)
	}
	if !strings.Contains(out, "timed out") {
		t.Errorf("output does not report a timeout:\n%s", out)
	}
}
