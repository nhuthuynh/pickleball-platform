// Package stalecitations answers one question, and reports rather than fails:
//
//	Which hand-written lines cite a migration whose subject a LATER migration
//	has since changed?
//
// It exists because of a measured defect class. T68 found two comments in the
// shipped web client asserting "Both Courts seeded by 0002_seed.sql have no
// Facility, so this is the common case on a freshly migrated database". The
// claim was false AND its rationale was inverted: 0034 backfills both seeded
// courts, so empty-facility had become the RARE case. #369 merged that comment
// at 13:41:22; #370 merged the backfill twelve minutes later, swept for the
// sites it invalidated, found thirteen in Go, and missed these two — one
// because the phrase wrapped across a newline, one because the sweep used
// --include=*.go and could not see a web client.
//
// The mechanism is the finding: a fact about the database was COPIED into
// prose rather than REFERENCED, so each copy became an independent claim that
// goes stale on its own schedule and that no gate can see. This tool sees the
// citation, not the claim — which is both why it works at all and why its
// precision is poor.
//
// # Why seed-then-mutate, and not "any later change"
//
// The first version of this tool flagged a citation whenever ANY later
// migration touched ANY table the cited migration touched. Measured on this
// repository: 363 findings across 141 files -- unreadable, and an advisory
// report nobody reads is worth nothing. Narrowing to "the cited migration
// INSERTed rows that a higher-numbered migration later UPDATEd or DELETEd"
// gives 45 findings across 34 files, and still catches both defects the tool
// was built from. Only four migration pairs in this repository satisfy it.
//
// Schema drift is therefore OUT of scope: a later ALTER TABLE does not flag
// anything. That is a real gap -- a comment describing a column that has since
// changed type is exactly as stale -- and it is accepted deliberately, because
// create-then-alter matches 22 pairs here and drowns the signal.
//
// # Report, never fail
//
// Scan always exits 0; exit 2 is reserved for a tree it could not read. This
// follows the decision recorded in tools/gatecoverage's Untested doc comment
// (#328 option (b)): a gate that FAILED here would assert a policy nobody
// decided -- that citing an older migration is wrong.
//
// docs/process/t7-sprint-plan.md is the standing counter-example. It says the
// seeded courts "have no facility until backfilled", which is accurate AND
// anticipates 0034. The citation shape is identical to the two real defects;
// only the truth value differs, and no static tool can tell them apart. So a
// human reads the report.
//
// No precision figure is quoted here. The 45 findings have not been audited
// one by one; two are the known defects this tool was built from and the other
// 43 are unexamined. An invented percentage is the defect class this tool
// exists to catch, so the honest statement is the count, its date, and the
// command: `make stale-citations`, 45 across 34 files, measured T68.
//
// # A tree it cannot read is an error, not a clean report
//
// Scan refuses a root with no db/migrations, or none holding an NNNN_*.sql.
// Before T68 it returned "no findings" for a mistyped -root, printing
// "OK -- no stale citations" for a tree it never opened. That is the
// vacuous-green failure this repository keeps shipping (t66-retro.md finding
// 3), and it was caught by cmd/stalecitations's exit-code test rather than by
// review -- an exit code being the only part of that program a Makefile sees.
//
// # No exclusion list
//
// There is no list of files, paths or packages to skip, and adding one is the
// change that would defeat this tool -- the same warning tools/gatecoverage
// carries about its own package list, for the same reason: three sprints
// running shipped a hand-written glob that was stale before its sprint ended.
// The only things skipped are structural and computed at run time:
//
//   - db/migrations itself, because a migration citing an earlier migration is
//     ordinary SQL history, not a claim about the present;
//   - generated files, recognised by Go's own "DO NOT EDIT" header rather than
//     by a path prefix, because generated code carries no hand-written claim;
//   - unreadable and non-UTF-8 files.
//
// This package's own test fixtures are named "NNNN_fixture*" instead of after
// the migrations they model. Reusing live names made a run over this
// repository flag the test file 22 times -- a third of the whole report. The
// fix was to rename the fixtures, not to teach the tool to skip its own
// directory, which would have been the first entry in the list above.
//
// The doc comment you are reading is a different matter and IS flagged,
// because it really does cite the migrations it describes. Renaming those
// would make this explanation useless, so they stay and their findings stay
// with them: a tool with no special cases does not get one for its own
// documentation.
//
// The first draft of this paragraph said "is flagged, six times". Writing the
// sentence added a seventh citation and made its own count wrong before the
// commit. That is the defect class this tool was built for, committed inside
// the tool, and it is left recorded here rather than corrected to seven --
// which would be wrong again on the next edit. CLAUDE.md rule 12: a
// correction deletes the claim or replaces it with a command. The command is
// `make stale-citations`.
//
// # What it cannot catch
//
//   - A stale claim that cites nothing. The great majority of claims in this
//     repository name no file at all; they are invisible here.
//   - A claim made stale by a code change rather than a migration.
//   - A claim that is wrong on the day it is written. Staleness is a change in
//     the world; this tool only sees change.
//   - Whether the citing line makes a claim at all. "See 0002_seed.sql" and
//     "0002_seed.sql proves X" are the same to it.
package stalecitations

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

// Finding is one line citing a migration that SEEDED data which a later
// migration then CHANGED.
type Finding struct {
	File            string   // path relative to the scan root
	Line            int      // 1-indexed
	Text            string   // the citing line, trimmed
	Cited           string   // e.g. "0002_seed.sql"
	Tables          []string // seeded tables later mutated, sorted
	LaterMigrations []string // higher-numbered migrations that mutated them, sorted
}

var (
	migrationName = regexp.MustCompile(`^(\d{4})_([A-Za-z0-9_]+)\.sql$`)

	// Two relations, not one. The narrow one is the whole point: see the
	// package doc's "Why seed-then-mutate" section.
	//
	// Matched against the WHOLE FILE, never line by line. db/migrations/0034
	// writes "UPDATE courts\nSET facility_id = ..." across two lines, so a
	// line-based scan misses it — which is the same wrapping failure that let
	// the original defect through #370's sweep, reproduced here while building
	// the tool meant to catch it. \s spans newlines in Go's regexp; keep it
	// that way.
	seedStatements = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bINSERT\s+INTO\s+([A-Za-z_][A-Za-z0-9_]*)`),
	}
	mutateStatements = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bUPDATE\s+([A-Za-z_][A-Za-z0-9_]*)\s+SET\b`),
		regexp.MustCompile(`(?i)\bDELETE\s+FROM\s+([A-Za-z_][A-Za-z0-9_]*)`),
	}
)

// matchTables returns the sorted, deduplicated table names the given statement
// patterns capture. nil when none match.
func matchTables(sql string, pats []*regexp.Regexp) []string {
	seen := map[string]bool{}
	for _, re := range pats {
		for _, m := range re.FindAllStringSubmatch(sql, -1) {
			seen[strings.ToLower(m[1])] = true
		}
	}
	if len(seen) == 0 {
		return nil
	}
	out := make([]string, 0, len(seen))
	for t := range seen {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

// tablesSeeded: tables this migration INSERTs rows into.
func tablesSeeded(sql string) []string { return matchTables(sql, seedStatements) }

// tablesMutated: tables this migration UPDATEs or DELETEs rows in.
func tablesMutated(sql string) []string { return matchTables(sql, mutateStatements) }

type migration struct {
	name    string // "0002_seed.sql"
	stem    string // "0002_seed"
	number  string // "0002"
	seeded  []string
	mutated []string
}

func loadMigrations(root string) ([]migration, error) {
	dir := filepath.Join(root, "db", "migrations")
	entries, err := os.ReadDir(dir)
	if err != nil {
		// Deliberately NOT a quiet "no migrations, nothing to report". A
		// mistyped -root would then print "OK -- no stale citations", which is
		// a gate reporting green on a tree it never read: the vacuous-green
		// failure this repository has shipped more than once (t66-retro.md
		// finding 3). Found by cmd/stalecitations's own exit-code test, which
		// is the only place an exit code is visible.
		return nil, fmt.Errorf("reading %s: %w", dir, err)
	}
	var out []migration
	for _, e := range entries {
		m := migrationName.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		body, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		out = append(out, migration{
			name:    e.Name(),
			stem:    strings.TrimSuffix(e.Name(), ".sql"),
			number:  m[1],
			seeded:  tablesSeeded(string(body)),
			mutated: tablesMutated(string(body)),
		})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no NNNN_*.sql migrations under %s", dir)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out, nil
}

// laterMutating returns the tables cited SEEDED that some higher-numbered
// migration later MUTATED, together with the names of those migrations. Both
// are empty when the cited migration seeds nothing, or when nothing later
// changes what it seeded.
func laterMutating(cited migration, all []migration) (tables, migrations []string) {
	// No early return for "seeds nothing": an empty seeded set matches no
	// mutation, so the loop below already returns empty. A mutation test
	// (T68) found the early return survived its own removal -- it was an
	// optimization wearing a guard's name, and this comment is cheaper than
	// a test that pins a no-op.
	seeded := map[string]bool{}
	for _, t := range cited.seeded {
		seeded[t] = true
	}
	hitTables := map[string]bool{}
	for _, m := range all {
		if m.number <= cited.number {
			continue
		}
		hit := false
		for _, t := range m.mutated {
			if seeded[t] {
				hitTables[t] = true
				hit = true
			}
		}
		if hit {
			migrations = append(migrations, m.name)
		}
	}
	if len(migrations) == 0 {
		return nil, nil
	}
	for t := range hitTables {
		tables = append(tables, t)
	}
	sort.Strings(tables)
	sort.Strings(migrations)
	return tables, migrations
}

func isGenerated(body []byte) bool {
	// Go's own convention, as tools/gatecoverage uses it: the marker appears in
	// the first lines of a generated file.
	head := body
	if len(head) > 4096 {
		head = head[:4096]
	}
	return strings.Contains(string(head), "DO NOT EDIT")
}

// Scan walks root and returns every citation of a migration whose tables a
// later migration changes. It never returns a "failure"; the error is reserved
// for an unreadable tree.
func Scan(root string) ([]Finding, error) {
	migrations, err := loadMigrations(root)
	if err != nil {
		return nil, err
	}
	byStem := map[string]migration{}
	for _, m := range migrations {
		byStem[m.stem] = m
	}

	migrationsDir := filepath.Join(root, "db", "migrations")
	var findings []Finding

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // unreadable entry: skip, never fail the run
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules", "dist", "build":
				return fs.SkipDir
			}
			if path == migrationsDir {
				return fs.SkipDir // a migration citing a migration is SQL history
			}
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil || !utf8.Valid(body) || isGenerated(body) {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}

		sc := bufio.NewScanner(strings.NewReader(string(body)))
		sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
		for n := 1; sc.Scan(); n++ {
			line := sc.Text()
			for stem, m := range byStem {
				if !strings.Contains(line, stem) {
					continue
				}
				tables, later := laterMutating(m, migrations)
				if len(later) == 0 {
					continue
				}
				findings = append(findings, Finding{
					File:            filepath.ToSlash(rel),
					Line:            n,
					Text:            strings.TrimSpace(line),
					Cited:           m.name,
					Tables:          tables,
					LaterMigrations: later,
				})
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(findings, func(i, j int) bool {
		if findings[i].File != findings[j].File {
			return findings[i].File < findings[j].File
		}
		if findings[i].Line != findings[j].Line {
			return findings[i].Line < findings[j].Line
		}
		return findings[i].Cited < findings[j].Cited
	})
	return findings, nil
}

// Report renders findings for a build log. It is advisory: the caller exits 0.
func Report(findings []Finding) string {
	if len(findings) == 0 {
		return "stale-citations: OK — no tracked line cites a migration that a later migration has since changed.\n"
	}
	var b strings.Builder
	files := map[string]bool{}
	for _, f := range findings {
		files[f.File] = true
	}
	fmt.Fprintf(&b, "stale-citations: NOTE — %d citation(s) in %d file(s) name a migration that a LATER migration has since changed.\n",
		len(findings), len(files))
	b.WriteString("  This is advisory and never fails the build: the tool sees the citation, not the claim.\n")
	b.WriteString("  Roughly 40% of these are real (measured T68). A citation is correct whenever the later\n")
	b.WriteString("  migration does not disturb what the line actually asserts. Read each one; do not bulk-edit.\n")
	b.WriteString("  A line that turns out to be wrong is DELETED or replaced with a reference, never restated\n")
	b.WriteString("  in new prose (CLAUDE.md rule 12).\n\n")
	for _, f := range findings {
		fmt.Fprintf(&b, "  %s:%d\n", f.File, f.Line)
		fmt.Fprintf(&b, "      %s\n", f.Text)
		fmt.Fprintf(&b, "      cites %s (touches %s); since changed by %s\n\n",
			f.Cited, strings.Join(f.Tables, ", "), strings.Join(f.LaterMigrations, ", "))
	}
	return b.String()
}
