package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The exit code is the only part of this program a Makefile can see, and the
// whole design rests on it being 0 even with findings. That cannot be checked
// in-process, so it is checked in a subprocess -- the shape cmd/docsindex uses.
func build(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "stalecitations")
	out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput()
	if err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	return bin
}

func fixture(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, body := range files {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func run(t *testing.T, bin, root string) (string, int) {
	t.Helper()
	cmd := exec.Command(bin, "-root", root)
	out, err := cmd.CombinedOutput()
	code := 0
	if ee, ok := err.(*exec.ExitError); ok {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatalf("run: %v", err)
	}
	return string(out), code
}

func TestExitsZeroWithFindings(t *testing.T) {
	bin := build(t)
	root := fixture(t, map[string]string{
		"db/migrations/0002_fixtureseed.sql": "INSERT INTO courts (id) VALUES ('1');\n",
		"db/migrations/0034_fixtureback.sql": "UPDATE courts\nSET facility_id = '3';\n",
		"web/a.go":                           "// see 0002_fixtureseed.sql\n",
	})
	out, code := run(t, bin, root)
	if code != 0 {
		t.Errorf("exit code = %d, want 0 — this gate reports, it never fails", code)
	}
	if !strings.Contains(out, "NOTE") || !strings.Contains(out, "web/a.go:1") {
		t.Errorf("expected a finding for web/a.go:1, got:\n%s", out)
	}
}

func TestExitsZeroWithNoFindings(t *testing.T) {
	bin := build(t)
	root := fixture(t, map[string]string{
		"db/migrations/0002_fixtureseed.sql": "INSERT INTO courts (id) VALUES ('1');\n",
		"web/a.go":                           "// see 0002_fixtureseed.sql\n",
	})
	out, code := run(t, bin, root)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(out, "OK") {
		t.Errorf("expected OK, got:\n%s", out)
	}
}

func TestExitsTwoOnUnreadableTree(t *testing.T) {
	bin := build(t)
	out, code := run(t, bin, filepath.Join(t.TempDir(), "does-not-exist"))
	if code != 2 {
		t.Errorf("exit code = %d, want 2 for a broken run; output:\n%s", code, out)
	}
}
