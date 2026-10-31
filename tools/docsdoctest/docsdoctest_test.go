package docsdoctest

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fixture reads one testdata document. The fixtures are real .md files rather
// than string literals because the thing under test is markdown parsing, and a
// Go string literal silently normalises exactly what trips an extractor up:
// fence indentation, tabs, and trailing whitespace.
func fixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return string(b)
}

func TestExtractMarkdown(t *testing.T) {
	cases := []struct {
		name string
		file string
		want []Block // Line, Command, Expected, Marked
	}{
		{
			name: "plain, indented and multi-command marked fences",
			file: "marked.md",
			want: []Block{
				{Line: 6, Marked: true, Command: `printf 'one\ntwo\n'`, Expected: []string{"one", "two"}},
				{Line: 17, Marked: true, Command: `printf 'indented\n'`, Expected: []string{"indented"}},
				{Line: 25, Marked: true, Command: `printf 'first\n'`, Expected: []string{"first", ""}},
				{Line: 28, Marked: true, Command: `printf 'second\n'`, Expected: []string{"second"}},
			},
		},
		{
			// The blind spot, deliberately: a prompt-bearing fence with no
			// marker is extracted and reported, never run.
			name: "an unmarked prompt block is found but not marked",
			file: "unmarked.md",
			want: []Block{
				{Line: 6, Marked: false,
					Command:  `curl -H "Authorization: Bearer $TOKEN" localhost:8080/v1/bookings`,
					Expected: []string{`{"bookings": []}`}},
			},
		},
		{
			name: "backslash continuations are joined into one command",
			file: "continuation.md",
			want: []Block{
				{Line: 7, Marked: true,
					Command:  "for w in a b; do \\\n    printf '%s\\n' \"$w\"; \\\n  done",
					Expected: []string{"a", "b"}},
			},
		},
		{
			name: "a command with no output lines claims empty output",
			file: "nooutput.md",
			want: []Block{
				{Line: 7, Marked: true,
					Command:  `printf 'x\n' | grep -i "nothing matches this"     # no output`,
					Expected: nil},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ExtractMarkdown(fixture(t, tc.file), tc.file)
			if len(got) != len(tc.want) {
				t.Fatalf("got %d block(s), want %d:\n%s", len(got), len(tc.want), dump(got))
			}
			for i, w := range tc.want {
				g := got[i]
				if g.File != tc.file {
					t.Errorf("block %d: File = %q, want %q", i, g.File, tc.file)
				}
				if g.Line != w.Line {
					t.Errorf("block %d: Line = %d, want %d", i, g.Line, w.Line)
				}
				if g.Command != w.Command {
					t.Errorf("block %d: Command =\n%q\nwant\n%q", i, g.Command, w.Command)
				}
				if g.Marked != w.Marked {
					t.Errorf("block %d: Marked = %v, want %v", i, g.Marked, w.Marked)
				}
				if strings.Join(g.Expected, "\n") != strings.Join(w.Expected, "\n") {
					t.Errorf("block %d: Expected = %q, want %q", i, g.Expected, w.Expected)
				}
			}
		})
	}
}

func dump(bs []Block) string {
	var sb strings.Builder
	for _, b := range bs {
		sb.WriteString(b.String())
		sb.WriteString("\n")
	}
	return sb.String()
}

// A fence with no "$ " line is not a candidate at all. CLAUDE.md's
// architecture tree and its npm comparison table are both of this shape, and
// a gate that reported them as an unclassified blind spot would be crying
// wolf about prose that makes no reproducible claim.
func TestExtractIgnoresFencesWithNoPrompt(t *testing.T) {
	src := "# Doc\n\n```\nproto/   API contract\ninternal/<context>/\n```\n\n" +
		"```\nlockfile present -> {high: 0}, deps 368   # correct\n```\n"
	if got := ExtractMarkdown(src, "d.md"); len(got) != 0 {
		t.Fatalf("got %d block(s), want 0:\n%s", len(got), dump(got))
	}
}

// A "$" that is not a prompt — mid-line, or "$VAR" with no space — must not
// start a command. Shell snippets in docs are full of both.
func TestExtractRequiresADollarSpacePrefix(t *testing.T) {
	src := "```doctest\nTOKEN=$(make -s dev-token)\n$HOME/bin\necho \"cost: $ 5\"\n```\n"
	if got := ExtractMarkdown(src, "d.md"); len(got) != 0 {
		t.Fatalf("got %d block(s), want 0:\n%s", len(got), dump(got))
	}
}

func TestMarkerIsAWholeTokenInTheInfoString(t *testing.T) {
	cases := []struct {
		info string
		want bool
	}{
		{"console doctest", true},
		{"doctest", true},
		{"sh   doctest", true},
		{"", false},
		{"console", false},
		// Substring matches must not count: a language named "doctestish" is
		// not an opt-in, and neither is prose in the info string.
		{"doctestish", false},
		{"nodoctest", false},
	}
	for _, tc := range cases {
		src := "```" + tc.info + "\n$ printf 'x\\n'\nx\n```\n"
		got := ExtractMarkdown(src, "d.md")
		if len(got) != 1 {
			t.Fatalf("info %q: got %d block(s), want 1", tc.info, len(got))
		}
		if got[0].Marked != tc.want {
			t.Errorf("info %q: Marked = %v, want %v", tc.info, got[0].Marked, tc.want)
		}
	}
}

// Tilde fences and longer backtick runs are both CommonMark, and a doc using
// one would otherwise be invisible — a silent hole, which is worse than a
// loud one.
func TestExtractHandlesTildeAndLongerFences(t *testing.T) {
	src := "~~~doctest\n$ printf 'a\\n'\na\n~~~\n\n````doctest\n$ printf 'b\\n'\nb\n````\n"
	got := ExtractMarkdown(src, "d.md")
	if len(got) != 2 {
		t.Fatalf("got %d block(s), want 2:\n%s", len(got), dump(got))
	}
	if got[0].Command != `printf 'a\n'` || got[1].Command != `printf 'b\n'` {
		t.Errorf("commands = %q, %q", got[0].Command, got[1].Command)
	}
}

func TestNormalize(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{"trailing whitespace per line is invisible and is stripped",
			"a   \nb\t\n", "a\nb"},
		{"trailing blank lines are stripped from both sides",
			"a\n\n\n", "a"},
		{"ANSI colour sequences are stripped",
			"\x1b[32mok\x1b[0m pkg", "ok pkg"},
		{"CRLF is normalised to LF",
			"a\r\nb\r\n", "a\nb"},
		{"interior blank lines are significant and are kept",
			"a\n\nb\n", "a\n\nb"},
		{"leading whitespace is significant — it is how a list is indented",
			"    cmd/devtoken\n", "    cmd/devtoken"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Normalize(tc.in); got != tc.want {
				t.Errorf("Normalize(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// The decisions NOT to normalise, pinned so that "let's just strip timings"
// has to argue with a test. Both are documented in the package comment:
// a timing-stripping regexp guesses at output shape and can mask a real
// change, and normalising an absolute path hides a doc that hardcoded one
// machine's filesystem.
func TestNormalizeLeavesTimingsAndAbsolutePathsAlone(t *testing.T) {
	for _, s := range []string{"ok  \tpkg\t0.312s", "/home/someone/checkout/x"} {
		if got := Normalize(s); got != s {
			t.Errorf("Normalize(%q) = %q — it must not rewrite this", s, got)
		}
	}
}

// --- running -------------------------------------------------------------

// scriptedRunner answers from a table, so the report-level tests do not
// depend on a shell, a PATH, or a timer.
func scriptedRunner(out map[string]Execution) Runner {
	return func(_ context.Context, command string) Execution {
		e, ok := out[command]
		if !ok {
			return Execution{Stderr: "no such scripted command: " + command, Exit: 127}
		}
		return e
	}
}

func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, body := range files {
		full := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", rel, err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	return root
}

func TestRunPassesWhenTheDocumentedOutputReproduces(t *testing.T) {
	root := writeTree(t, map[string]string{
		"CLAUDE.md": "```doctest\n$ count\n26\n```\n",
	})
	rep, err := Run(root, time.Minute, scriptedRunner(map[string]Execution{
		"count": {Stdout: "26\n"},
	}))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !rep.OK() {
		t.Fatalf("want OK:\n%s", rep)
	}
	if len(rep.Checked) != 1 || len(rep.Mismatches) != 0 {
		t.Fatalf("Checked=%d Mismatches=%d", len(rep.Checked), len(rep.Mismatches))
	}
}

// The core assertion of the whole ticket: a figure that no longer reproduces
// fails, and the failure names everything a reader needs to fix it without
// re-deriving anything.
func TestRunFailsOnAStaleFigureAndNamesEverything(t *testing.T) {
	root := writeTree(t, map[string]string{
		"CLAUDE.md": "intro\n\n```console doctest\n$ count-packages\n27\n```\n",
	})
	rep, err := Run(root, time.Minute, scriptedRunner(map[string]Execution{
		"count-packages": {Stdout: "26\n"},
	}))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if rep.OK() {
		t.Fatalf("a stale figure must fail the gate:\n%s", rep)
	}
	if len(rep.Mismatches) != 1 {
		t.Fatalf("Mismatches = %d, want 1", len(rep.Mismatches))
	}
	out := rep.String()
	for _, want := range []string{"CLAUDE.md", ":4", "count-packages", "27", "26"} {
		if !strings.Contains(out, want) {
			t.Errorf("report does not mention %q:\n%s", want, out)
		}
	}
}

// grep exits 1 when it matches nothing, and CLAUDE.md documents a command of
// exactly that shape ("# no output"). Gating on exit status would fail it.
func TestRunDoesNotJudgeExitStatus(t *testing.T) {
	root := writeTree(t, map[string]string{
		"d.md": "```doctest\n$ grepish     # no output\n```\n",
	})
	rep, err := Run(root, time.Minute, scriptedRunner(map[string]Execution{
		"grepish     # no output": {Exit: 1},
	}))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !rep.OK() {
		t.Fatalf("a non-zero exit with the documented (empty) output must pass:\n%s", rep)
	}
}

// An empty expectation is a real claim, so output where the doc promised none
// is a mismatch — otherwise "# no output" would be unfalsifiable.
func TestRunFailsWhenACommandDocumentedAsSilentPrintsSomething(t *testing.T) {
	root := writeTree(t, map[string]string{
		"d.md": "```doctest\n$ quiet     # no output\n```\n",
	})
	rep, err := Run(root, time.Minute, scriptedRunner(map[string]Execution{
		"quiet     # no output": {Stdout: "surprise\n"},
	}))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if rep.OK() {
		t.Fatalf("unexpected output must fail:\n%s", rep)
	}
}

// Comparison is against stdout. A doc that means to assert stderr says "2>&1",
// which is the shell's own way of saying it; stderr is surfaced in the failure
// text so an author is never left guessing why.
func TestRunComparesStdoutAndSurfacesStderrOnFailure(t *testing.T) {
	root := writeTree(t, map[string]string{
		"d.md": "```doctest\n$ noisy\nwanted\n```\n",
	})
	rep, err := Run(root, time.Minute, scriptedRunner(map[string]Execution{
		"noisy": {Stdout: "other\n", Stderr: "make: *** missing separator\n"},
	}))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if rep.OK() {
		t.Fatalf("want failure:\n%s", rep)
	}
	if !strings.Contains(rep.String(), "missing separator") {
		t.Errorf("stderr is not surfaced:\n%s", rep)
	}
}

func TestRunReportsUnmarkedBlocksWithoutFailing(t *testing.T) {
	root := writeTree(t, map[string]string{
		"a.md": "```doctest\n$ ok\nfine\n```\n",
		"b.md": "```\n$ curl localhost:8080/v1/x\n{}\n```\n\n```\n$ docker ps\nID\n```\n",
	})
	rep, err := Run(root, time.Minute, scriptedRunner(map[string]Execution{
		"ok": {Stdout: "fine\n"},
	}))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !rep.OK() {
		t.Fatalf("unmarked blocks must not fail the gate:\n%s", rep)
	}
	if len(rep.Unchecked) != 2 {
		t.Fatalf("Unchecked = %d, want 2", len(rep.Unchecked))
	}
	// The summary counts the blind spot — both how deep (2 commands) and how
	// wide (1 file) — on every run.
	if out := rep.String(); !strings.Contains(out, "2 documented command(s) in 1 file(s)") {
		t.Errorf("the blind spot must be counted in the summary:\n%s", out)
	}
	// The listing locates each one, behind the flag. ~190 lines on the real
	// repository, which printed unconditionally would bury the mismatches.
	spots := rep.BlindSpots()
	for _, want := range []string{"b.md:2", "b.md:7", "curl", "docker ps"} {
		if !strings.Contains(spots, want) {
			t.Errorf("BlindSpots() does not mention %q:\n%s", want, spots)
		}
	}
}

// "docs-doctest: OK" must never be reachable by checking nothing. Without
// this the whole gate is switched off by deleting one marker.
func TestRunFailsWhenItCheckedNothing(t *testing.T) {
	root := writeTree(t, map[string]string{
		"d.md": "```\n$ curl localhost:8080\n{}\n```\n",
	})
	rep, err := Run(root, time.Minute, scriptedRunner(nil))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if rep.OK() {
		t.Fatalf("a run that checked no block has not passed:\n%s", rep)
	}
	if !strings.Contains(rep.String(), "no block") {
		t.Errorf("the report must say it checked nothing:\n%s", rep)
	}
}

func TestRunTimesOutRatherThanHanging(t *testing.T) {
	root := writeTree(t, map[string]string{
		"d.md": "```doctest\n$ slow\nnever\n```\n",
	})
	rep, err := Run(root, 10*time.Millisecond, func(ctx context.Context, _ string) Execution {
		<-ctx.Done()
		return Execution{TimedOut: true}
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if rep.OK() {
		t.Fatalf("a timed-out block must fail:\n%s", rep)
	}
	if !strings.Contains(rep.String(), "timed out") {
		t.Errorf("the report must say it timed out:\n%s", rep)
	}
}

// Fixtures live in testdata precisely so the live gate cannot read them: this
// package's own testdata holds a deliberately-stale "27" that must never fail
// the repository's gate. Same property gatecoverage's scan uses, same reason.
func TestScanSkipsTestdataHiddenAndVendorDirectories(t *testing.T) {
	root := writeTree(t, map[string]string{
		"real.md":                   "```doctest\n$ a\nx\n```\n",
		"tools/x/testdata/stale.md": "```doctest\n$ printf '27'\n26\n```\n",
		".github/notes.md":          "```doctest\n$ b\ny\n```\n",
		"_scratch/notes.md":         "```doctest\n$ c\nz\n```\n",
		"node_modules/p/readme.md":  "```doctest\n$ d\nw\n```\n",
		"vendor/v/readme.md":        "```doctest\n$ e\nv\n```\n",
		"web/src/app.ts":            "// not markdown\n",
	})
	blocks, files, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if files != 1 {
		t.Errorf("FilesScanned = %d, want 1", files)
	}
	if len(blocks) != 1 || blocks[0].File != "real.md" {
		t.Fatalf("blocks = %s", dump(blocks))
	}
}

func TestScanFindsMarkdownByExtensionCaseInsensitively(t *testing.T) {
	root := writeTree(t, map[string]string{
		"a.md":            "```doctest\n$ a\nx\n```\n",
		"docs/B.MD":       "```doctest\n$ b\ny\n```\n",
		"docs/c.markdown": "```doctest\n$ c\nz\n```\n",
		"d.txt":           "```doctest\n$ d\nw\n```\n",
	})
	blocks, _, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(blocks) != 3 {
		t.Fatalf("got %d block(s), want 3:\n%s", len(blocks), dump(blocks))
	}
}

// TestFixturesThroughTheRealShell closes the loop the scripted Runner leaves
// open: extraction, `bash -c`, capture, normalisation and comparison, all of
// it, against the fixture documents. The commands are printf/grep/for only, so
// it needs no repository toolchain and no network.
//
// Each fixture is copied into a temp root because Scan deliberately skips
// testdata — the property that stops the live gate reading mismatch.md's
// deliberately-stale "27" as a claim about this repository.
func TestFixturesThroughTheRealShell(t *testing.T) {
	cases := []struct {
		fixture string
		wantOK  bool
		checked int
		why     string
	}{
		{"marked.md", true, 4, "plain, indented and two-command fences all reproduce"},
		{"continuation.md", true, 1, "a backslash-continued command reaches bash intact"},
		{"nooutput.md", true, 1, "a grep that matches nothing satisfies an empty claim"},
		{"exitstatus.md", true, 1, "exit status 1 with the documented output is a pass"},
		{"mismatch.md", false, 1, "a stale figure fails"},
	}

	for _, tc := range cases {
		t.Run(tc.fixture, func(t *testing.T) {
			root := writeTree(t, map[string]string{tc.fixture: fixture(t, tc.fixture)})

			rep, err := Run(root, 30*time.Second, ShellRunner(root))
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if len(rep.Checked) != tc.checked {
				t.Errorf("Checked = %d, want %d", len(rep.Checked), tc.checked)
			}
			if rep.OK() != tc.wantOK {
				t.Fatalf("OK() = %v, want %v (%s):\n%s", rep.OK(), tc.wantOK, tc.why, rep)
			}
		})
	}
}

// --- against the real tree -----------------------------------------------

// TestTheRealRepositorysDocsAreCheckedAndPass is the end-to-end half, and it
// skips rather than fails when a prerequisite is missing: `make test-tools` is
// deliberately codegen-free and Docker-free, and CLAUDE.md's own marked block
// shells out to `make generate`, which needs buf and sqlc on PATH. Following
// tools/gatecoverage's real-repository test, which skips when internal/gen is
// absent rather than blaming its header scan for a fresh clone.
func TestTheRealRepositorysDocsAreCheckedAndPass(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}

	// The structural half always runs: it needs no tools and it is what
	// catches an extractor that has stopped seeing this repository's docs.
	blocks, files, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if files == 0 {
		t.Fatal("no markdown scanned in this repository")
	}
	var marked int
	for _, b := range blocks {
		if !b.Marked {
			continue
		}
		marked++
		if b.Command == "" {
			t.Errorf("%s: a marked block has an empty command", b)
		}
	}
	if marked == 0 {
		t.Fatal("this repository documents commands with pasted output and none is marked for " +
			"checking — the gate would be switched off")
	}

	// The executing half needs the executables this repository's currently
	// marked blocks shell out to. This IS a short list — in the test, which
	// is where gatecoverage's real-repository test puts its own prerequisite
	// probe (it stats internal/gen). The rule the tool obeys is that the
	// GATE holds no list; a test-side prerequisite guard is a different
	// thing, and it fails in the safe direction: a future marked block that
	// needs some other tool makes this test go RED here rather than skip,
	// which is what a drift-catching test should do.
	for _, tool := range []string{"make", "buf", "sqlc"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%q is not on PATH, so this repository's marked blocks cannot be executed "+
				"here (`make tools-install` installs it); `make docs-doctest` is where they run", tool)
		}
	}

	rep, err := Run(root, 5*time.Minute, ShellRunner(root))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !rep.OK() {
		t.Fatalf("this repository's own documented commands do not reproduce:\n%s", rep)
	}
}
