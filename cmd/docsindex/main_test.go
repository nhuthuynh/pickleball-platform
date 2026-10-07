package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// #328: `-statuses` and its non-zero exit shipped at T64.5 with no test
// anywhere — rules 1 and 8 were not met, in the one `cmd` package whose
// output a ceremony is now told to rely on. This file is that debt paid.
//
// Every test here drives the real entry point: either listStatuses against a
// throwaway tree, or the built command in a subprocess so the EXIT CODE
// itself is asserted rather than the error value that is supposed to produce
// it. An exit code is the only part of this program a Makefile can see.

// adrTree writes a throwaway repository root holding just docs/adr, which is
// all ADRStatuses reads. Named files, because the listing is sorted by
// filename and the order is part of what is asserted.
func adrTree(t *testing.T, files map[string]string) string {
	t.Helper()

	root := t.TempDir()
	dir := filepath.Join(root, "docs", "adr")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

const (
	headingAccepted  = "# ADR-0001: A\n\n## Status\n**Accepted (T1, 2026-01-01)** — with prose after it.\n"
	frontSuperseded  = "# ADR-0002: B\n\n- **Status:** Superseded by ADR-0003\n- **Date:** 2026-01-02\n"
	headingEscalated = "# ADR-0003: C\n\n## Status\nEscalated — awaiting the user's decision.\n"
	noStatusAtAll    = "# ADR-0004: D\n\nThis document never says what its status is.\n"
	unknownToken     = "# ADR-0005: E\n\n## Status\nPending a chat with someone.\n"
)

func listInto(t *testing.T, root string) (string, error) {
	t.Helper()

	var buf bytes.Buffer
	err := listStatuses(root, &buf)
	return buf.String(), err
}

// TestListStatusesPrintsARowPerADRAndATally is the listing itself: one row
// per ADR, in filename order, each naming the canonical token and which of
// the two legitimate status forms carried it, then a tally.
func TestListStatusesPrintsARowPerADRAndATally(t *testing.T) {
	t.Parallel()

	root := adrTree(t, map[string]string{
		"0001-a.md": headingAccepted,
		"0002-b.md": frontSuperseded,
		"0003-c.md": headingEscalated,
	})

	out, err := listInto(t, root)
	if err != nil {
		t.Fatalf("unexpected err: %v\n%s", err, out)
	}

	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 5 { // three rows, a blank line, the tally
		t.Fatalf("got %d line(s), want 5:\n%s", len(lines), out)
	}
	for i, want := range []struct{ token, form, file string }{
		{"Accepted", "## heading", "0001-a.md"},
		{"Superseded", "front-matter", "0002-b.md"},
		{"Escalated", "## heading", "0003-c.md"},
	} {
		fields := strings.Fields(lines[i])
		if len(fields) < 3 {
			t.Fatalf("line %d = %q, want three columns", i, lines[i])
		}
		if fields[0] != want.token {
			t.Fatalf("line %d token = %q, want %q", i, fields[0], want.token)
		}
		if got := strings.Join(fields[1:len(fields)-1], " "); got != want.form {
			t.Fatalf("line %d form = %q, want %q", i, got, want.form)
		}
		if fields[len(fields)-1] != want.file {
			t.Fatalf("line %d file = %q, want %q", i, fields[len(fields)-1], want.file)
		}
	}

	tally := lines[len(lines)-1]
	for _, want := range []string{"3 ADR(s):", "Accepted=1", "Superseded=1", "Escalated=1"} {
		if !strings.Contains(tally, want) {
			t.Fatalf("tally = %q, want it to contain %q", tally, want)
		}
	}
}

// TestTheTallyAccountsForEveryRow guards the shape of the tally rather than
// its contents: it is printed by walking a fixed token list, so a row whose
// token is not on that list would be listed and never counted. The sum has to
// equal the row count for every tree.
func TestTheTallyAccountsForEveryRow(t *testing.T) {
	t.Parallel()

	root := adrTree(t, map[string]string{
		"0001-a.md": headingAccepted,
		"0002-b.md": frontSuperseded,
		"0003-c.md": headingEscalated,
		"0005-e.md": unknownToken,
	})

	out, _ := listInto(t, root) // an unknown token is unclassifiable: err is expected
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	rows := len(lines) - 2
	tally := lines[len(lines)-1]

	total := 0
	for _, part := range strings.Fields(tally) {
		if i := strings.IndexByte(part, '='); i > 0 {
			n, err := strconv.Atoi(part[i+1:])
			if err != nil {
				t.Fatalf("tally fragment %q: %v", part, err)
			}
			total += n
		}
	}
	if total != rows {
		t.Fatalf("tally sums to %d across %d row(s): %q\n%s", total, rows, tally, out)
	}
}

// TestAnUnclassifiableStatusIsListedAndThenFails is the behaviour #328 names
// as the one most worth pinning: an ADR the parser cannot classify must be
// SHOWN (so a reader knows which file to fix) and must still fail (so a
// ceremony cannot act on a listing that quietly omitted one). Both halves, or
// this is the silent skip T62.4 was written to stop, reintroduced in the tool
// built to display the values.
func TestAnUnclassifiableStatusIsListedAndThenFails(t *testing.T) {
	t.Parallel()

	for name, body := range map[string]string{
		"no status section at all":         noStatusAtAll,
		"a status with no canonical token": unknownToken,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			root := adrTree(t, map[string]string{
				"0001-a.md":   headingAccepted,
				"0009-bad.md": body,
			})

			out, err := listInto(t, root)
			if err == nil {
				t.Fatalf("err = nil for an unclassifiable status; listing was:\n%s", out)
			}
			if !strings.Contains(err.Error(), "cannot classify") {
				t.Fatalf("err = %v, want it to say the status cannot be classified", err)
			}
			if !strings.Contains(out, "UNCLASSIFIABLE") || !strings.Contains(out, "0009-bad.md") {
				t.Fatalf("the offending ADR is not in the listing:\n%s", out)
			}
			if !strings.Contains(out, "UNCLASSIFIABLE=1") {
				t.Fatalf("the tally does not count the unclassifiable ADR:\n%s", out)
			}
			if !strings.Contains(out, "0001-a.md") {
				t.Fatalf("the classifiable ADRs were dropped once one failed:\n%s", out)
			}
		})
	}
}

// TestAnEmptyStatusFormIsLabelled: an ADR with no status section has no form
// either, and an empty column would read as a formatting bug rather than as
// the finding it is.
func TestAnEmptyStatusFormIsLabelled(t *testing.T) {
	t.Parallel()

	root := adrTree(t, map[string]string{"0004-d.md": noStatusAtAll})

	out, _ := listInto(t, root)
	if !strings.Contains(out, "no status") {
		t.Fatalf("an ADR with no status section is not labelled as such:\n%s", out)
	}
}

// TestAMissingDocsADRDirectoryIsAnError: pointed at a tree that is not a
// repository, the listing must say so rather than report a clean zero.
func TestAMissingDocsADRDirectoryIsAnError(t *testing.T) {
	t.Parallel()

	out, err := listInto(t, t.TempDir())
	if err == nil {
		t.Fatalf("err = nil for a root with no docs/adr; listing was:\n%s", out)
	}
	if !strings.Contains(err.Error(), "docs/adr") {
		t.Fatalf("err = %v, want it to name docs/adr", err)
	}
}

// ---------------------------------------------------------------------
// The exit codes, asserted by running the real command
// ---------------------------------------------------------------------

// runStatuses builds and runs this command against root, returning its exit
// code and combined output. Building the binary is what makes this an
// assertion about `go run ./cmd/docsindex -statuses` — the form the Makefile
// and CLAUDE.md both use — rather than about a function that happens to
// return an error.
func runStatuses(t *testing.T, root string) (int, string) {
	t.Helper()

	bin := filepath.Join(t.TempDir(), "docsindex")
	build := exec.Command("go", "build", "-o", bin, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("building the command: %v\n%s", err, out)
	}

	out, err := exec.Command(bin, "-statuses", "-root", root).CombinedOutput()
	if err == nil {
		return 0, string(out)
	}
	var exit *exec.ExitError
	if !errors.As(err, &exit) {
		t.Fatalf("running the command: %v\n%s", err, out)
	}
	return exit.ExitCode(), string(out)
}

// TestExitZeroWhenEveryStatusClassifies.
func TestExitZeroWhenEveryStatusClassifies(t *testing.T) {
	t.Parallel()

	code, out := runStatuses(t, adrTree(t, map[string]string{
		"0001-a.md": headingAccepted,
		"0003-c.md": headingEscalated,
	}))
	if code != 0 {
		t.Fatalf("exit = %d, want 0:\n%s", code, out)
	}
	if !strings.Contains(out, "Escalated=1") {
		t.Fatalf("output does not carry the tally:\n%s", out)
	}
}

// TestExitTwoWhenAStatusCannotBeClassified is the assertion #328 asks for in
// so many words. Exit 2 rather than 1 is deliberate in main: 1 means "the
// index and the tree disagree", 2 means "this tool could not do its job", and
// a ceremony that cannot tell those apart would treat an unreadable corpus as
// a clean one.
func TestExitTwoWhenAStatusCannotBeClassified(t *testing.T) {
	t.Parallel()

	code, out := runStatuses(t, adrTree(t, map[string]string{
		"0001-a.md":   headingAccepted,
		"0009-bad.md": noStatusAtAll,
	}))
	if code != 2 {
		t.Fatalf("exit = %d, want 2:\n%s", code, out)
	}
	if !strings.Contains(out, "UNCLASSIFIABLE") {
		t.Fatalf("the failing ADR is not named in the output:\n%s", out)
	}
}

// TestTheRealRepositoryClassifiesCleanly runs the command against this
// checkout, which is what `make docs-index-check`'s sibling command does on
// every ceremony. A fixture-only test would pass with the real corpus red.
func TestTheRealRepositoryClassifiesCleanly(t *testing.T) {
	t.Parallel()

	code, out := runStatuses(t, "../..")
	if code != 0 {
		t.Fatalf("exit = %d against the real repository, want 0:\n%s", code, out)
	}
	if !strings.Contains(out, "ADR(s):") {
		t.Fatalf("no tally in the output:\n%s", out)
	}
}
