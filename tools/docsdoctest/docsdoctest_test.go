package docsdoctest

import (
	"context"
	"errors"
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

// OK()'s "checked nothing" clause catches a TOTAL switch-off and cannot see a
// PARTIAL one — unmark a single fence and paste a false figure, and only two
// counters move. Naming each executed block puts that loss in CI's own output
// rather than only in a diff.
func TestReportNamesEachExecutedBlockNotJustTheCount(t *testing.T) {
	root := writeTree(t, map[string]string{
		"a.md": "```doctest\n$ printf 'x\\n'\nx\n```\n",
		"b.md": "intro\n\n```console doctest\n$ printf 'y\\n'\ny\n```\n",
	})
	rep, err := Run(root, time.Minute, ShellRunner(root))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !rep.OK() {
		t.Fatalf("want OK:\n%s", rep)
	}
	out := rep.String()
	for _, want := range []string{"a.md:2", "b.md:4", "printf 'x", "printf 'y"} {
		if !strings.Contains(out, want) {
			t.Errorf("the summary does not name %q:\n%s", want, out)
		}
	}

	// And the point of naming them: unmarking one is visible in the output,
	// not only in the diff.
	root2 := writeTree(t, map[string]string{
		"a.md": "```doctest\n$ printf 'x\\n'\nx\n```\n",
		"b.md": "intro\n\n```console\n$ printf 'y\\n'\ny\n```\n",
	})
	rep2, err := Run(root2, time.Minute, ShellRunner(root2))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if strings.Contains(rep2.String(), "b.md:4") {
		t.Errorf("an unmarked block is still listed as executed:\n%s", rep2)
	}
}

// Each prompt line is its own `bash -c`, so shell state does not carry between
// them. README.md's ghcr.io probe is exactly this shape. Pinned because the
// failure direction matters: such a block fails LOUDLY rather than passing
// wrongly, and a future change to share one shell per fence would flip that.
func TestEachPromptLineGetsItsOwnShell(t *testing.T) {
	root := writeTree(t, map[string]string{
		"d.md": "```doctest\n$ V=set\n$ printf '[%s]\\n' \"$V\"\n[set]\n```\n",
	})
	rep, err := Run(root, time.Minute, ShellRunner(root))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if rep.OK() {
		t.Fatalf("a variable must NOT carry to the next prompt line:\n%s", rep)
	}
	if !strings.Contains(rep.String(), "[]") {
		t.Errorf("the failure should show the empty expansion:\n%s", rep)
	}

	// The documented remedy: one command, so one shell.
	joined := writeTree(t, map[string]string{
		"d.md": "```doctest\n$ V=set; printf '[%s]\\n' \"$V\"\n[set]\n```\n",
	})
	rep2, err := Run(joined, time.Minute, ShellRunner(joined))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !rep2.OK() {
		t.Fatalf("joining into one command must work:\n%s", rep2)
	}
}

// The honest-limits entry this pins: an empty-output claim is satisfied when
// the output moves to stderr, which is where a reader would look for it.
// Asserted so the limitation is a known property rather than a surprise, and
// so that `2>&1` is shown to close it.
func TestAnEmptyOutputClaimIsBlindToStderr(t *testing.T) {
	// The inner quoting keeps the command's own ">&2" separate from the
	// doc's outer "2>&1". Written as `printf … >&2 2>&1` on one line both
	// redirections apply left to right and the output still lands on
	// stderr, so the pair below would prove nothing — which is exactly what
	// the first draft of this test did.
	const writesToStderr = `sh -c 'printf "surprise\n" >&2'`

	root := writeTree(t, map[string]string{
		"d.md": "```doctest\n$ " + writesToStderr + "     # no output\n```\n",
	})
	rep, err := Run(root, time.Minute, ShellRunner(root))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !rep.OK() {
		t.Fatalf("documented limit: stdout is empty, so this PASSES:\n%s", rep)
	}

	withRedirect := writeTree(t, map[string]string{
		"d.md": "```doctest\n$ " + writesToStderr + " 2>&1     # no output\n```\n",
	})
	rep2, err := Run(withRedirect, time.Minute, ShellRunner(withRedirect))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if rep2.OK() {
		t.Fatalf("with 2>&1 the same command must FAIL — that is how a doc closes this:\n%s", rep2)
	}
}

// TestTimeoutSurvivesAGrandchildHoldingTheOutputPipe is the regression test
// for the one defect the PE review of PR #362 found, and it has to use a
// BACKGROUNDED GRANDCHILD: exec.CommandContext's default cancellation kills
// only bash, and with a strings.Builder sink Wait blocks until every process
// holding the pipe's write end closes it. Before the fix this document hung
// forever printing nothing (measured: 25s and killed by an outer `timeout`,
// against 2s and a correct report for the single-child case), while
// TestTimeoutFlagIsHonoured passed throughout — it exercises only the child
// case, which cannot fail.
//
// Run through the REAL ShellRunner, because the defect was in ShellRunner's
// process handling and a scripted Runner cannot have it. Asserted as "returns
// within a bound", which is the property; the watchdog is what fails, rather
// than the test hanging and taking the package's whole run with it.
func TestTimeoutSurvivesAGrandchildHoldingTheOutputPipe(t *testing.T) {
	root := writeTree(t, map[string]string{
		// The subshell backgrounds a sleep that outlives bash, then
		// bash itself blocks. Killing bash alone leaves the grandchild
		// holding the pipe.
		"d.md": "```console doctest\n$ (sleep 45 &) ; sleep 45\nnever\n```\n",
	})

	type result struct {
		rep     *Report
		err     error
		elapsed time.Duration
	}
	done := make(chan result, 1)
	go func() {
		start := time.Now()
		rep, err := Run(root, 500*time.Millisecond, ShellRunner(root))
		done <- result{rep, err, time.Since(start)}
	}()

	// Generously above the 500ms limit plus WaitDelay, so this fails only
	// on a genuine hang and not on a slow machine.
	const bound = 30 * time.Second
	select {
	case got := <-done:
		if got.err != nil {
			t.Fatalf("Run: %v", got.err)
		}
		if got.rep.OK() {
			t.Fatalf("a timed-out command must fail the gate:\n%s", got.rep)
		}
		if !strings.Contains(got.rep.String(), "timed out") {
			t.Errorf("the report must say it timed out:\n%s", got.rep)
		}
		if got.elapsed > bound {
			t.Errorf("Run took %s, want well under %s", got.elapsed, bound)
		}
	case <-time.After(bound):
		t.Fatalf("ShellRunner HUNG: after %s the per-command timeout had not returned. "+
			"A surviving grandchild is holding the output pipe open — set cmd.WaitDelay "+
			"and kill the process group, not just the child (see killGroup)", bound)
	}
}

// The other half of the fix: the group kill, so a timed-out command's
// grandchildren are reaped rather than orphaned. WaitDelay alone would unblock
// Wait and leave an orphaned `go build` burning a CI runner's CPU for the rest
// of the job, which no assertion about elapsed time would catch.
func TestTimeoutReapsTheWholeProcessGroup(t *testing.T) {
	if _, err := exec.LookPath("pgrep"); err != nil {
		t.Skip("pgrep is not on PATH, so surviving processes cannot be looked for")
	}

	// A distinctive duration, so pgrep cannot match another test's process,
	// and LONG — long enough that the grandchild cannot be reaped by simply
	// finishing. A first draft of this test used `sleep 47` and so PASSED
	// against the unfixed runner: Run blocked for the full 47s, by which
	// point the grandchild had exited on its own and there was nothing left
	// to find. A guard that cannot fail is the defect this ticket is about,
	// so the duration is now far beyond the watchdog below.
	const marker = "sleep 293"
	root := writeTree(t, map[string]string{
		"d.md": "```console doctest\n$ (" + marker + " &) ; " + marker + "\n```\n",
	})

	// Best-effort cleanup, so a FAILING run does not leave a five-minute
	// sleep behind on a CI runner.
	t.Cleanup(func() { _ = exec.Command("pkill", "-f", marker).Run() })

	type result struct {
		rep *Report
		err error
	}
	done := make(chan result, 1)
	go func() {
		rep, err := Run(root, 500*time.Millisecond, ShellRunner(root))
		done <- result{rep, err}
	}()

	const bound = 30 * time.Second
	select {
	case got := <-done:
		if got.err != nil {
			t.Fatalf("Run: %v", got.err)
		}
		if got.rep.OK() {
			t.Fatalf("want a timeout failure:\n%s", got.rep)
		}
	case <-time.After(bound):
		t.Fatalf("ShellRunner did not return within %s of a 500ms limit", bound)
	}

	deadline := time.Now().Add(10 * time.Second)
	for {
		out, _ := exec.Command("pgrep", "-f", marker).Output()
		if len(strings.TrimSpace(string(out))) == 0 {
			return // reaped
		}
		if time.Now().After(deadline) {
			t.Fatalf("a grandchild of the timed-out command survived: pgrep still reports %q — "+
				"the process GROUP must be killed, not just the direct child (see killGroup). "+
				"WaitDelay alone unblocks Wait and leaves this orphan running",
				strings.TrimSpace(string(out)))
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// TestWaitDelayBoundsWaitWhenAProcessEscapesTheGroup pins cmd.WaitDelay, the
// half of the timeout fix that the group kill cannot cover.
//
// WHY IT NEEDS setsid. The two tests above are both satisfied by Setpgid +
// Cancel alone: cutting the WaitDelay line leaves them passing in 0.50s and
// 0.52s. So "WaitDelay is the guarantee", which this package's own doc comment
// says, was production code no test demanded — found by the re-review of
// PR #362 building the full mutation matrix rather than trusting the claim.
// A process that calls setsid() leaves the process group, so killGroup cannot
// reach it, and it keeps holding the output pipe; WaitDelay is then the only
// thing that lets Wait return. Measured, 2s limit, outer `timeout 25`:
//
//	WaitDelay cut      (no output at all)                ELAPSED=25s  RC=124
//	WaitDelay present  "timed out after the per-command" ELAPSED=7s   RC=1
//
// The elapsed-time assertion below is what stops this test going vacuous. If
// setsid ever stopped escaping, or stopped inheriting the pipe, Run would
// return in ~500ms and the test would pass for entirely the wrong reason —
// the same trap that made this file's first reaping test pass against the
// unfixed runner. Requiring that it took at least about WaitDelay proves the
// pipe really was held open and that WaitDelay is what released it.
func TestWaitDelayBoundsWaitWhenAProcessEscapesTheGroup(t *testing.T) {
	if _, err := exec.LookPath("setsid"); err != nil {
		t.Skip("setsid is not on PATH, so a process that escapes its process group " +
			"cannot be created here")
	}

	// Distinctive and long, so pgrep/pkill cannot hit another test's
	// process and so the escapee cannot be reaped by merely finishing.
	const marker = "sleep 291"
	root := writeTree(t, map[string]string{
		"d.md": "```console doctest\n$ (setsid " + marker + " &) ; " + marker + "\nnever\n```\n",
	})

	// This escapee survives by design — killGroup cannot reach it — so
	// cleaning up after it is this test's own responsibility.
	t.Cleanup(func() { _ = exec.Command("pkill", "-f", marker).Run() })

	const limit = 500 * time.Millisecond
	type result struct {
		rep     *Report
		err     error
		elapsed time.Duration
	}
	done := make(chan result, 1)
	go func() {
		start := time.Now()
		rep, err := Run(root, limit, ShellRunner(root))
		done <- result{rep, err, time.Since(start)}
	}()

	// Comfortably above limit+WaitDelay, so this fails only on a real hang.
	bound := WaitDelay + 25*time.Second
	select {
	case got := <-done:
		if got.err != nil {
			t.Fatalf("Run: %v", got.err)
		}
		if got.rep.OK() {
			t.Fatalf("a command whose output pipe is held open must fail:\n%s", got.rep)
		}
		// Slack below WaitDelay only for scheduling, not enough to let a
		// 500ms return through.
		if floor := WaitDelay - time.Second; got.elapsed < floor {
			t.Fatalf("Run returned in %s, before WaitDelay (%s) could have released it — "+
				"the setsid child is not holding the output pipe, so this test is not "+
				"exercising WaitDelay at all", got.elapsed, WaitDelay)
		}
	case <-time.After(bound):
		t.Fatalf("ShellRunner HUNG: after %s a %s limit had not returned. A process that "+
			"escaped the group via setsid is holding the output pipe and only cmd.WaitDelay "+
			"can bound Wait — the group kill cannot reach it", bound, limit)
	}
}

// A command that finishes with exactly the right output but leaves a
// background child holding the pipe used to be reported as "could not be run:
// exec: WaitDelay expired before I/O complete" — which says the opposite of
// what happened. It WAS run, its stdout was exactly right, and TimedOut is
// false because the context never fired, so the report was truthful about
// neither thing. Introduced by the WaitDelay fix and found by the re-review.
func TestABackgroundChildHoldingThePipeIsReportedTruthfully(t *testing.T) {
	root := writeTree(t, map[string]string{
		// A generous limit, so this is NOT the timeout path: the
		// command exits at once and its output is correct.
		"d.md": "```console doctest\n$ (sleep 23 &) ; printf 'hello\\n'\nhello\n```\n",
	})
	t.Cleanup(func() { _ = exec.Command("pkill", "-f", "sleep 23").Run() })

	rep, err := Run(root, time.Minute, ShellRunner(root))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if rep.OK() {
		t.Fatalf("a lingering background child truncates the stream, so this must fail:\n%s", rep)
	}

	out := rep.String()
	if strings.Contains(out, "could not be run") {
		t.Errorf("the report claims the command could not be run, but it ran and printed "+
			"the documented output:\n%s", out)
	}
	for _, want := range []string{"ran and exited", "held the output pipe open", "hello"} {
		if !strings.Contains(out, want) {
			t.Errorf("the report does not mention %q:\n%s", want, out)
		}
	}
	// And the genuine "could not be run" case must still say so.
	if o := (Outcome{Execution: Execution{Err: errors.New("exec: \"bash\": not found")}}); //
	!strings.Contains(o.describe(), "could not be run") {
		t.Errorf("a command that truly could not be run must still say so:\n%s", o.describe())
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
