// Package docsdoctest answers one question mechanically, for the whole
// repository, every time it runs: **does every documented command still
// produce the output printed beside it?**
//
// # Why this exists
//
// This project's single most-repeated failure is a doc asserting a quantity
// nobody measured, or prescribing a command that does not reproduce the figure
// beside it. docs/process/sprint-process.md already requires that every
// quantity carry the command that produced it. Nothing enforced it, and the
// consequence was measured at T67.5: CLAUDE.md's gate-coverage block printed
// its own verification commands and asserted 27, 14 and "three" where the tree
// said 26, 13 and two. That file had been corrected line by line by three
// adversarial review passes 24 days earlier and drifted into three measurable
// falsehoods anyway, because nothing re-checked it.
//
// So the check is not "is there a command beside the number?". That is a
// presence check, satisfiable by pasting any command at all, and this
// repository already has a documented case of a gate that `touch` could
// silence. The check is **run the command and diff the output**.
//
// # The design constraint, inherited from tools/gatecoverage
//
// Both sides are derived at run time and **neither is a list**:
//
//   - Side A — the documented claims — is a scan of every markdown file in the
//     tree for fenced blocks carrying a shell prompt. A doc written next sprint
//     is covered with no edit here.
//   - Side B — the actual output — is produced by running the command. There
//     is no recorded "golden" output anywhere in this package.
//
// There is deliberately **no list of files, no list of commands, and no
// exclusion list**. gatecoverage's doc comment states the principle and the
// evidence: "there is no package list in the tool, and adding one is the one
// change that would defeat it" — three sprints running shipped a hand-written
// glob that was stale before its sprint ended. If a block cannot be handled,
// the fix is to improve the extractor or to skip by a **property of the
// block**, never by naming the file it lives in.
//
// # The classification rule: an opt-in marker on the fence
//
// A block is checked when its fence's info string carries the whole token
// `doctest` (so “```console doctest”, which GitHub still renders as a console
// block). Everything else is extracted, counted and reported, never run.
//
// WHY OPT-IN AND NOT A CLASSIFIER. The alternative is to inspect the command
// and decide whether it looks safe and deterministic — and every version of
// that idea collapses into the artifact this package refuses to have. "Only
// run `make`, `go`, `grep`, `find`" is a command list. "Skip anything
// mentioning docker or curl" is an exclusion list. Both are stale the moment
// someone documents a tool nobody thought of, and both fail in the dangerous
// direction: a classifier that guesses "safe" wrongly runs `rm -rf` or a
// `curl` against production.
//
// The decisive argument, though, is that the author's INTENT is not
// recoverable from the text. This repository contains two blocks that are
// textually near-identical and want opposite treatment:
//
//   - `$ make gate-coverage | grep -iE "ONLY build-tagged|compiled"  # no
//     output` — a real claim ("this prints nothing"), and checkable.
//   - `$ for f in $(find . -name '*_test.go'); do …` — a command offered so a
//     reader can derive a figure the doc deliberately does NOT paste. There is
//     nothing to compare, and asserting empty output would be simply wrong.
//
// No classifier can tell those apart. A marker can, because the author says
// which they meant. Preferring precision over coverage is also the ticket's
// instruction: a gate that checks fewer blocks correctly beats one that is
// switched off because it cries wolf.
//
// WHAT THE MARKER LETS THROUGH — the blind spot, stated plainly. An unmarked
// block is never checked, so a wrong figure in one survives, and the gate can
// be switched off for any block by deleting four characters. Two things make
// that visible rather than silent:
//
//   - Every prompt-bearing unmarked block is COUNTED AND LOCATED in the
//     report, on every run, clean or not. The blind spot is a number a reader
//     sees without reading this source — following gatecoverage's "report,
//     never fail" precedent for its own untested-package NOTE (#328).
//   - A run that checked NO block FAILS. "docs-doctest: OK" can therefore
//     never mean "I checked nothing", which is the failure mode an opt-in
//     mechanism invites.
//
// # Normalisation, and the two deliberate non-normalisations
//
// Applied to both sides before comparison: CRLF to LF, ANSI escape sequences
// removed, trailing whitespace stripped per line, trailing blank lines
// stripped. Each is invisible in a rendered doc or in a terminal, so a
// mismatch caused by one would be a false alarm nobody could see.
//
// Leading whitespace is KEPT — it is how gate-coverage indents its package
// list, and stripping it would stop the gate noticing a change in that shape.
//
// Interior blank lines are KEPT. (Trailing ones must go: a fence holding two
// commands separated by a blank line gives the first command a trailing blank
// that no terminal would show.)
//
// NOT normalised, both on purpose:
//
//   - **Timings.** `ok pkg 0.312s` is left alone. A timing-stripping regexp is
//     a guess about output shape that can mask a real change, and the better
//     remedy is already available to the doc author: pipe the command through
//     `grep`, `wc` or `sed`, which also makes the block fast. That the marker
//     is opt-in is what makes this a reasonable thing to ask.
//   - **Absolute paths.** Commands run with the repository root as the working
//     directory, so relative paths are stable; a doc that hardcodes one
//     machine's absolute path is wrong on every other machine, and
//     normalising it away would hide that rather than fix it.
//
// # Streams and exit status
//
// Comparison is against **stdout**. A block that means to assert what a
// command writes to stderr says `2>&1`, which is the shell's own way of saying
// it; merging the streams unconditionally would make comparison depend on
// interleaving, which is not deterministic. On a mismatch the captured stderr
// IS printed, so an author is never left guessing.
//
// **Exit status is reported and never judged.** `grep` exits 1 when it matches
// nothing, and "this matches nothing" is a documented claim in this very
// repository. A block's claim is about its output.
//
// # Slow blocks, Docker and the network
//
// Each command gets a timeout (DefaultTimeout); exceeding it is a FAILURE,
// named and located, never a silent skip. There is deliberately no Docker
// probe and no network probe — either would be a command classifier, i.e. the
// list this package refuses to have. A block that needs Docker or the network
// must simply not be marked: this gate runs inside `make ci-checks`, which is
// Docker-free and offline, and an unmarked block is counted in the blind-spot
// report where a reader can see it.
//
// # Honest limits — what this gate cannot catch
//
// It is a narrow instrument and overselling it would be the same failure in a
// new place:
//
//   - **A rule nobody follows that has no figure in it.** "Write a failing
//     test first" has nothing to run.
//   - **A statement that is true and is the wrong statement.** A command that
//     reproduces its output exactly while answering a question that does not
//     matter passes here, forever.
//   - **Prose that is wrong with no number in it.** T64's review found a
//     quote attributed to the wrong file; both files existed, nothing was
//     countable, and this gate would have said OK.
//   - **A question nobody thought to ask.** The gate checks the claims that
//     are written down. The expensive defects in this project's history —
//     enforce_game_capacity()'s dropped clauses, payable_type's unwidened
//     CHECK — were absences, and an absence has no fenced block.
//   - **An unmarked block**, per the blind spot above.
//
// It catches exactly one thing: a documented command whose pasted output is no
// longer what it prints. That was three live falsehoods in the rulebook on the
// day it shipped.
package docsdoctest

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Marker is the opt-in token looked for in a fence's info string. A whole
// whitespace-separated token, not a substring: a language named "doctestish"
// is not an opt-in.
const Marker = "doctest"

// Prompt is the prefix that makes a line inside a fence a command. The
// trailing space is load-bearing — without it `$HOME/bin` and `TOKEN=$(…)`,
// both of which appear in this repository's docs, would be read as commands.
const Prompt = "$ "

// DefaultTimeout bounds a single documented command. Generous because
// CLAUDE.md's own marked block runs `make generate`; bounded because a gate
// that hangs is worse than one that fails.
const DefaultTimeout = 5 * time.Minute

// --- Side A: extraction ---------------------------------------------------

// Block is one documented command together with the output printed beneath it.
type Block struct {
	// File is slash-separated and relative to the scan root.
	File string
	// Line is the 1-based line number of the command's first line, so an
	// editor jump from the failure message lands on it.
	Line int
	// Info is the fence's info string, kept for the report.
	Info string
	// Marked reports whether Info carried Marker.
	Marked bool
	// Command is the shell command, with backslash continuations joined
	// and the fence's indentation removed.
	Command string
	// Expected is the output lines printed beneath the command. Nil means
	// the block claims the command prints nothing — a real claim, not a
	// missing one.
	Expected []string
}

func (b Block) String() string {
	return fmt.Sprintf("%s:%d: $ %s", b.File, b.Line, firstLine(b.Command))
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i] + " …"
	}
	return s
}

// fenceOpen matches a CommonMark fenced-code-block opener: up to three spaces
// of indentation (CLAUDE.md nests fences five deep inside list items, so more
// is allowed here), a run of at least three backticks or tildes, then the info
// string.
var fenceOpen = regexp.MustCompile("^([ \t]*)(`{3,}|~{3,})[ \t]*(.*)$")

// ExtractMarkdown returns every command block in one markdown document, in
// document order. file is used only for reporting.
//
// The fence scan is deliberately small rather than a full CommonMark parser:
// the only constructs that matter are the fence delimiters, the info string
// and the fence's indentation (which is stripped from content lines, because
// this repository indents fences inside list items and the output lines'
// relative indentation is significant).
func ExtractMarkdown(src, file string) []Block {
	lines := strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n")

	var out []Block
	for i := 0; i < len(lines); i++ {
		m := fenceOpen.FindStringSubmatch(lines[i])
		if m == nil {
			continue
		}
		indent, delim, info := m[1], m[2], strings.TrimSpace(m[3])

		// Collect the fence body, up to a closing fence of the same
		// character and at least the same length, or end of document
		// (CommonMark allows an unclosed fence).
		var body []string
		var bodyLine []int
		j := i + 1
		for ; j < len(lines); j++ {
			if isFenceClose(lines[j], delim) {
				break
			}
			body = append(body, stripIndent(lines[j], indent))
			bodyLine = append(bodyLine, j+1)
		}
		out = append(out, blocksInFence(body, bodyLine, file, info)...)
		i = j
	}
	return out
}

// isFenceClose reports whether line closes a fence opened with delim: a run of
// the same character, at least as long, optionally indented, and nothing else
// on the line.
func isFenceClose(line, delim string) bool {
	t := strings.TrimRight(strings.TrimLeft(line, " \t"), " \t")
	if len(t) < len(delim) || t[0] != delim[0] {
		return false
	}
	return strings.Count(t, delim[:1]) == len(t)
}

// stripIndent removes the opening fence's exact indentation prefix from a
// content line, so that output indentation inside the block stays relative to
// the fence rather than to the page.
func stripIndent(line, indent string) string {
	// A blank line inside a fence carries no information and is often not
	// padded out to the fence's indentation by an editor.
	if strings.TrimSpace(line) == "" {
		return ""
	}
	if indent == "" {
		return line
	}
	if strings.HasPrefix(line, indent) {
		return line[len(indent):]
	}
	// Indented less than its own fence: malformed, so fall back to
	// dropping the leading blank run rather than keeping an indentation
	// that is meaningless relative to the fence.
	return strings.TrimLeft(line, " \t")
}

// blocksInFence splits one fence body into Blocks: each Prompt line starts a
// command, backslash-continuation lines extend it, and everything up to the
// next prompt is that command's expected output.
func blocksInFence(body []string, lineOf []int, file, info string) []Block {
	marked := hasMarker(info)

	var out []Block
	for i := 0; i < len(body); i++ {
		if !strings.HasPrefix(body[i], Prompt) {
			continue
		}
		b := Block{File: file, Line: lineOf[i], Info: info, Marked: marked}

		// The command, plus any backslash continuations. A trailing "\"
		// means the shell reads the next line as part of the same
		// command; treating those lines as expected output would
		// mis-parse CLAUDE.md's integration-test census entirely.
		cmd := []string{strings.TrimPrefix(body[i], Prompt)}
		for strings.HasSuffix(cmd[len(cmd)-1], `\`) && i+1 < len(body) {
			i++
			cmd = append(cmd, body[i])
		}
		b.Command = strings.Join(cmd, "\n")

		for i+1 < len(body) && !strings.HasPrefix(body[i+1], Prompt) {
			i++
			b.Expected = append(b.Expected, body[i])
		}
		out = append(out, b)
	}
	return out
}

// hasMarker reports whether an info string carries Marker as a whole
// whitespace-separated token.
func hasMarker(info string) bool {
	for _, f := range strings.Fields(info) {
		if f == Marker {
			return true
		}
	}
	return false
}

// markdownExt reports whether a filename is a markdown document, by extension
// and case-insensitively. A property of the name, not a list of documents.
func markdownExt(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".md", ".markdown":
		return true
	}
	return false
}

// skipDir reports directories the scan must not descend into.
//
// Identical to tools/gatecoverage's, deliberately and for a reason that
// matters here too: THIS package's own fixtures live in testdata and contain a
// deliberately-stale "27". Skipping testdata is what stops the gate reading
// its own test material as a claim about the repository. The dot- and
// underscore-prefixed rule mirrors `go list ./...`; node_modules is the Vue
// app's dependency tree, which holds thousands of vendored READMEs.
func skipDir(name string) bool {
	switch name {
	case "testdata", "node_modules", "vendor":
		return true
	}
	return strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_")
}

// Scan walks the tree rooted at root and returns every command block in every
// markdown document, in path order, plus the number of documents scanned.
func Scan(root string) (blocks []Block, filesScanned int, err error) {
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != root && skipDir(d.Name()) {
				return fs.SkipDir
			}
			return nil
		}
		if !markdownExt(d.Name()) {
			return nil
		}
		src, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		filesScanned++
		blocks = append(blocks, ExtractMarkdown(string(src), filepath.ToSlash(rel))...)
		return nil
	})
	if err != nil {
		return nil, 0, err
	}
	return blocks, filesScanned, nil
}

// --- Normalisation --------------------------------------------------------

// ansi matches the CSI escape sequences tools use for colour. Stripped because
// a doc records what a human read, not the bytes a TTY received.
var ansi = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]`)

// Normalize applies the comparison policy to one side. See the package comment
// for each rule and for the two deliberate non-normalisations.
func Normalize(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = ansi.ReplaceAllString(s, "")
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t\r")
	}
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return strings.Join(lines, "\n")
}

// --- Side B: running ------------------------------------------------------

// Execution is what running one documented command produced.
type Execution struct {
	Stdout   string
	Stderr   string
	Exit     int
	TimedOut bool
	// Err is a failure to run the command at all, as distinct from a
	// command that ran and exited non-zero.
	Err error
}

// Runner runs one shell command and reports what it produced. An interface
// seam so the report-level tests need no shell, no PATH and no clock.
type Runner func(ctx context.Context, command string) Execution

// ShellRunner runs commands through `bash -c` with dir as the working
// directory, which is what makes the relative paths in a documented command
// mean what the doc says.
//
// bash specifically, not `sh`: this repository's docs use `$(…)`, `[[`,
// pipelines and `for … done`, and the Makefile already depends on bash
// features. Stdout and stderr are captured separately — see the package
// comment on streams.
func ShellRunner(dir string) Runner {
	return func(ctx context.Context, command string) Execution {
		cmd := exec.CommandContext(ctx, "bash", "-c", command)
		cmd.Dir = dir
		// Keep the environment, so a command finds the same PATH a
		// developer or CI runner would. TERM is emptied so tools that
		// colour conditionally do not, which is a cheaper and more
		// honest fix than widening the ANSI regexp.
		cmd.Env = append(os.Environ(), "TERM=dumb", "NO_COLOR=1")

		var stdout, stderr strings.Builder
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		runErr := cmd.Run()
		e := Execution{Stdout: stdout.String(), Stderr: stderr.String()}
		if cmd.ProcessState != nil {
			e.Exit = cmd.ProcessState.ExitCode()
		}
		if ctx.Err() != nil {
			e.TimedOut = true
			return e
		}
		// A command that RAN and exited non-zero is not an error here —
		// exit status is never judged. A command that could not be
		// started at all is, so that a missing bash is never mistaken
		// for a stale figure.
		var exitErr *exec.ExitError
		if runErr != nil && !errors.As(runErr, &exitErr) {
			e.Err = runErr
		}
		return e
	}
}

// Outcome is one checked Block together with what running it produced.
type Outcome struct {
	Block
	Execution
}

// Mismatched reports whether this outcome is a failure: the output differed,
// the command timed out, or it could not be run at all. Exit status alone is
// never a failure — see the package comment.
func (o Outcome) Mismatched() bool {
	if o.TimedOut || o.Err != nil {
		return true
	}
	return Normalize(strings.Join(o.Expected, "\n")) != Normalize(o.Stdout)
}

// Report is one run of the gate.
type Report struct {
	Root         string
	FilesScanned int
	Timeout      time.Duration

	// Checked is every marked block that was run, Mismatches the subset
	// that failed.
	Checked    []Outcome
	Mismatches []Outcome

	// Unchecked is every prompt-bearing block that carried no marker —
	// the blind spot, reported on every run and never failed on.
	Unchecked []Block
}

// OK reports whether the gate passes.
//
// Note the second clause: a run that checked NOTHING has not passed. Without
// it, an opt-in gate is switched off by deleting one marker and still prints
// OK — the exact shape of "a gate that `touch` could silence".
func (r *Report) OK() bool {
	return len(r.Mismatches) == 0 && len(r.Checked) > 0
}

// Run scans root, executes every marked block, and reports.
func Run(root string, timeout time.Duration, run Runner) (*Report, error) {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	blocks, files, err := Scan(root)
	if err != nil {
		return nil, err
	}
	rep := &Report{Root: root, FilesScanned: files, Timeout: timeout}

	for _, b := range blocks {
		if !b.Marked {
			rep.Unchecked = append(rep.Unchecked, b)
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		o := Outcome{Block: b, Execution: run(ctx, b.Command)}
		cancel()

		rep.Checked = append(rep.Checked, o)
		if o.Mismatched() {
			rep.Mismatches = append(rep.Mismatches, o)
		}
	}
	return rep, nil
}

// --- Reporting ------------------------------------------------------------

// String renders the whole report. The counts print on every run, clean or
// not: a check that silently examined nothing must not be mistakable for a
// check that found nothing.
func (r *Report) String() string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "docs-doctest: %d markdown file(s) scanned, %d documented command(s) executed.\n",
		r.FilesScanned, len(r.Checked))

	if n := len(r.Unchecked); n > 0 {
		fmt.Fprintf(&sb, "\ndocs-doctest: NOTE — %d documented command(s) in %d file(s) carry no `%s` marker\n",
			n, r.uncheckedFiles(), Marker)
		sb.WriteString("  and were NOT run. Not a failure: an unmarked block is one whose author did not\n")
		sb.WriteString("  claim its output reproduces here — an illustrative curl, a `git show` of a commit\n")
		sb.WriteString("  recording what a past sprint measured, a command offered so a reader can derive a\n")
		sb.WriteString("  figure the doc deliberately does not paste, or one needing Docker or the network.\n")
		sb.WriteString("  This is this gate's blind spot, counted on every run so it is visible without\n")
		sb.WriteString("  reading the source. Counted and not listed, as gate-coverage does with its\n")
		sb.WriteString("  generated packages: the list is long enough to bury the findings below it.\n")
		sb.WriteString("  List them with `go run ./cmd/docsdoctest -blind-spots`.\n")
	}

	if len(r.Mismatches) > 0 {
		fmt.Fprintf(&sb, "\ndocs-doctest: FAIL — %d documented command(s) do not reproduce their stated output:\n",
			len(r.Mismatches))
		for _, o := range r.Mismatches {
			sb.WriteString("\n")
			sb.WriteString(o.describe())
		}
		sb.WriteString("\nFix these by correcting the doc to what the command prints, or by correcting the\n")
		sb.WriteString("command. Never by removing the marker: an unmarked wrong figure is the defect this\n")
		sb.WriteString("gate exists to catch, and there is no exclusion list in tools/docsdoctest to add to.\n")
		return sb.String()
	}

	if len(r.Checked) == 0 {
		sb.WriteString("\ndocs-doctest: FAIL — no block was checked, so nothing was verified.\n")
		fmt.Fprintf(&sb, "  Mark a fence with its language and `%s` (e.g. ```console %s) on a block whose\n",
			Marker, Marker)
		sb.WriteString("  pasted output should be reproducible. A gate that checked nothing has not passed.\n")
		return sb.String()
	}

	sb.WriteString("docs-doctest: OK — every marked command reproduces the output printed beside it.\n")
	return sb.String()
}

// uncheckedFiles counts the distinct documents the blind spot spans, so the
// NOTE says how wide it is as well as how deep.
func (r *Report) uncheckedFiles() int {
	seen := map[string]bool{}
	for _, b := range r.Unchecked {
		seen[b.File] = true
	}
	return len(seen)
}

// BlindSpots lists every unmarked command, located. Behind a flag rather than
// in String: on this repository it is ~190 lines, which printed on every run
// would bury the mismatches the gate exists to report — and a gate whose
// findings are buried gets switched off.
func (r *Report) BlindSpots() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "docs-doctest: %d unmarked documented command(s) in %d file(s):\n",
		len(r.Unchecked), r.uncheckedFiles())
	for _, b := range r.Unchecked {
		fmt.Fprintf(&sb, "  %s\n", b)
	}
	return sb.String()
}

// describe renders one mismatch with everything needed to act on it and
// nothing to re-derive: where it is, what ran, both sides of the diff, and the
// stderr that usually explains it.
func (o Outcome) describe() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "  %s:%d\n", o.File, o.Line)
	for _, l := range strings.Split(o.Command, "\n") {
		fmt.Fprintf(&sb, "    $ %s\n", l)
	}

	switch {
	case o.TimedOut:
		fmt.Fprintf(&sb, "    timed out after the per-command limit. Either make the documented command\n")
		fmt.Fprintf(&sb, "    fast (pipe it through grep or wc) or stop marking it.\n")
		return sb.String()
	case o.Err != nil:
		fmt.Fprintf(&sb, "    could not be run: %v\n", o.Err)
		return sb.String()
	}

	fmt.Fprintf(&sb, "    expected (from the doc):\n%s", indentBlock(Normalize(strings.Join(o.Expected, "\n"))))
	fmt.Fprintf(&sb, "    actual (from running it):\n%s", indentBlock(Normalize(o.Stdout)))
	if s := Normalize(o.Stderr); s != "" {
		fmt.Fprintf(&sb, "    stderr (not compared — add 2>&1 to the documented command to assert it):\n%s",
			indentBlock(s))
	}
	fmt.Fprintf(&sb, "    exit status %d (reported, never judged: a grep that matches nothing exits 1)\n", o.Exit)
	return sb.String()
}

// indentBlock renders one side of a diff, making an empty side explicit —
// "expected nothing" is a claim, and a blank gap would read as a bug here.
func indentBlock(s string) string {
	if s == "" {
		return "      (no output)\n"
	}
	var sb strings.Builder
	for _, l := range strings.Split(s, "\n") {
		fmt.Fprintf(&sb, "      | %s\n", l)
	}
	return sb.String()
}
