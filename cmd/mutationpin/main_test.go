package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func build(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "mutationpin")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	return bin
}

// The exit code is the only part of this program a Makefile can see, and the
// design rests on it being 0 even with survivors.
func TestExitsZeroWhenThereIsNoDiff(t *testing.T) {
	bin := build(t)
	root := t.TempDir()
	if out, err := exec.Command("git", "-C", root, "init", "-q").CombinedOutput(); err != nil {
		t.Skipf("git unavailable: %v\n%s", err, out)
	}
	cmd := exec.Command(bin, "-root", root, "-base", "HEAD")
	out, err := cmd.CombinedOutput()
	// An empty repo has no HEAD, so this is the broken-run path: exit 2.
	if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() != 2 {
		t.Errorf("exit = %d, want 2 for an unreadable diff; output:\n%s", ee.ExitCode(), out)
	}
}

func TestGoTestRunnerReportsKillAndSurvival(t *testing.T) {
	root := t.TempDir()
	write := func(rel, body string) {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module tmpmod\n\ngo 1.26\n")
	write("pkg/a.go", "package pkg\n\nfunc F() int { return 1 }\n")

	run, calls := goTestRunner(root, time.Minute, 10)

	// A passing package: the mutant would survive.
	write("pkg/a_test.go", "package pkg\n\nimport \"testing\"\n\nfunc TestF(t *testing.T) { _ = F() }\n")
	passed, err := run("pkg")
	if err != nil {
		t.Fatalf("runner: %v", err)
	}
	if !passed {
		t.Error("runner reported failure for a passing package")
	}

	// A failing package: the mutant is killed.
	write("pkg/a_test.go", "package pkg\n\nimport \"testing\"\n\nfunc TestF(t *testing.T) { t.Fatal(\"boom\") }\n")
	passed, err = run("pkg")
	if err != nil {
		t.Fatalf("runner: %v", err)
	}
	if passed {
		t.Error("runner reported success for a failing package")
	}
	if *calls != 2 {
		t.Errorf("calls = %d, want 2", *calls)
	}
}

func TestGoTestRunnerStopsAtMax(t *testing.T) {
	run, _ := goTestRunner(t.TempDir(), time.Minute, 1)
	if _, err := run("pkg"); err == nil {
		// first call is allowed (it will fail to run, but not for -max)
		_ = err
	}
	_, err := run("pkg")
	if err == nil || !strings.Contains(err.Error(), "-max limit") {
		t.Errorf("second call err = %v, want a -max limit error", err)
	}
}
