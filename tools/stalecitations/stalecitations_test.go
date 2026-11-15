package stalecitations

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Fixture migrations are named "NNNN_fixture*" rather than after the real
// migrations they model. A fixture that reused a live name would be flagged by
// a run of this tool over this repository -- 22 times, measured -- burying the
// findings under the tool's own test data. The alternative, teaching the tool
// to skip its own directory, is the first entry in the exclusion list the
// package doc says must never exist.

// writeTree materialises a map of relative path -> contents under a temp root.
func writeTree(t *testing.T, files map[string]string) string {
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

func TestTablesSeededAndMutated(t *testing.T) {
	tests := []struct {
		name       string
		sql        string
		wantSeed   []string
		wantMutate []string
	}{
		{"insert seeds", "INSERT INTO courts (id) VALUES ('x');", []string{"courts"}, nil},
		{"update mutates", "UPDATE courts SET facility_id = 'y';", nil, []string{"courts"}},
		{"delete mutates", "DELETE FROM courts WHERE id = 'z';", nil, []string{"courts"}},
		{"lowercase", "update courts set facility_id = 'y';", nil, []string{"courts"}},
		{"create is neither", "CREATE TABLE courts (id uuid);", nil, nil},
		{"alter is neither", "ALTER TABLE courts ADD COLUMN facility_id uuid;", nil, nil},
		{"both, deduplicated and sorted", "INSERT INTO facilities (id) VALUES ('f');\nINSERT INTO courts (id) VALUES ('c');\nUPDATE courts SET x = 1;", []string{"courts", "facilities"}, []string{"courts"}},
		{"none", "-- just a comment\n", nil, nil},

		// The regression that matters: 0034 writes "UPDATE courts" and "SET"
		// on separate lines. A line-based scan misses it, which is the same
		// wrapping failure that let the original defect past #370's sweep.
		{"UPDATE and SET on separate lines", "UPDATE courts\nSET facility_id = '3333'\nWHERE id IN ('1111');", nil, []string{"courts"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tablesSeeded(tt.sql); strings.Join(got, ",") != strings.Join(tt.wantSeed, ",") {
				t.Errorf("tablesSeeded() = %v, want %v", got, tt.wantSeed)
			}
			if got := tablesMutated(tt.sql); strings.Join(got, ",") != strings.Join(tt.wantMutate, ",") {
				t.Errorf("tablesMutated() = %v, want %v", got, tt.wantMutate)
			}
		})
	}
}

// The real defect this tool exists to catch, reduced to its skeleton:
// a comment cites 0002_fixtureseed and asserts a property of `courts`, and a LATER
// migration changes `courts`.
func TestScanFindsTheRealShape(t *testing.T) {
	root := writeTree(t, map[string]string{
		"db/migrations/0002_fixtureseed.sql":     "INSERT INTO courts (id) VALUES ('1111');\n",
		"db/migrations/0034_fixturebackfill.sql": "UPDATE courts\nSET facility_id = '3333';\n",
		"web/src/views/MyBookings.vue":           "// Both Courts seeded\n// by `0002_fixtureseed.sql` have no Facility.\n",
	})

	findings, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 {
		t.Fatalf("got %d findings, want 1: %+v", len(findings), findings)
	}
	f := findings[0]
	if f.File != "web/src/views/MyBookings.vue" {
		t.Errorf("File = %q", f.File)
	}
	if f.Line != 2 {
		t.Errorf("Line = %d, want 2 (the line carrying the citation)", f.Line)
	}
	if f.Cited != "0002_fixtureseed.sql" {
		t.Errorf("Cited = %q", f.Cited)
	}
	if strings.Join(f.Tables, ",") != "courts" {
		t.Errorf("Tables = %v", f.Tables)
	}
	if strings.Join(f.LaterMigrations, ",") != "0034_fixturebackfill.sql" {
		t.Errorf("LaterMigrations = %v", f.LaterMigrations)
	}
}

func TestScanQuietCases(t *testing.T) {
	tests := []struct {
		name  string
		files map[string]string
	}{
		{
			// Nothing later touches the cited table.
			name: "no later migration mutates the seeded table",
			files: map[string]string{
				"db/migrations/0002_fixtureseed.sql":  "INSERT INTO courts (id) VALUES ('1');\n",
				"db/migrations/0034_fixtureother.sql": "UPDATE pricing_rules SET x = 1;\n",
				"web/a.go":                            "// see 0002_fixtureseed.sql\n",
			},
		},
		{
			// Mutation by an EARLIER migration is not a staleness risk.
			name: "only an earlier migration mutates the table",
			files: map[string]string{
				"db/migrations/0001_fixtureearlyfix.sql": "UPDATE courts SET x = 1;\n",
				"db/migrations/0002_fixtureseed.sql":     "INSERT INTO courts (id) VALUES ('1');\n",
				"web/a.go":                               "// see 0002_fixtureseed.sql\n",
			},
		},
		{
			// Schema drift is deliberately OUT of scope: a later ALTER is not
			// a mutation of seeded rows. Narrow by design; see the package doc.
			name: "a later migration only ALTERs the table",
			files: map[string]string{
				"db/migrations/0002_fixtureseed.sql":  "INSERT INTO courts (id) VALUES ('1');\n",
				"db/migrations/0034_fixturealter.sql": "ALTER TABLE courts ADD COLUMN tz text;\n",
				"web/a.go":                            "// see 0002_fixtureseed.sql\n",
			},
		},
		{
			// The cited migration seeds nothing, so it has no data to go stale.
			name: "cited migration seeds nothing",
			files: map[string]string{
				"db/migrations/0001_fixtureearlyfixtureinit.sql": "CREATE TABLE courts (id uuid);\n",
				"db/migrations/0034_fixturelater.sql":            "UPDATE courts SET x = 1;\n",
				"web/a.go":                                       "// see 0001_fixtureearlyfixtureinit.sql\n",
			},
		},
		{
			// A migration citing another migration is ordinary SQL history.
			name: "citations inside db/migrations are not scanned",
			files: map[string]string{
				"db/migrations/0002_fixtureseed.sql":  "INSERT INTO courts (id) VALUES ('1');\n",
				"db/migrations/0034_fixturelater.sql": "-- supersedes 0002_fixtureseed.sql\nUPDATE courts SET x = 1;\n",
			},
		},
		{
			// Generated code is not hand-written prose; it cannot carry a claim.
			name: "generated code is skipped",
			files: map[string]string{
				"db/migrations/0002_fixtureseed.sql":  "INSERT INTO courts (id) VALUES ('1');\n",
				"db/migrations/0034_fixturelater.sql": "UPDATE courts SET x = 1;\n",
				"internal/gen/x.go":                   "// Code generated by protoc. DO NOT EDIT.\n// see 0002_fixtureseed.sql\n",
			},
		},
		{
			// A citation of a migration that does not exist is not this tool's business.
			name: "citation of an unknown migration",
			files: map[string]string{
				"db/migrations/0002_fixtureseed.sql": "INSERT INTO courts (id) VALUES ('1');\n",
				"web/a.go":                           "// see 9999_fixtureabsent.sql\n",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings, err := Scan(writeTree(t, tt.files))
			if err != nil {
				t.Fatal(err)
			}
			if len(findings) != 0 {
				t.Errorf("got %d findings, want 0: %+v", len(findings), findings)
			}
		})
	}
}

func TestReportIsStableAndNamesTheCause(t *testing.T) {
	findings := []Finding{{
		File:            "web/src/views/MyBookings.vue",
		Line:            2,
		Text:            "// by `0002_fixtureseed.sql` have no Facility.",
		Cited:           "0002_fixtureseed.sql",
		Tables:          []string{"courts"},
		LaterMigrations: []string{"0034_fixturebackfill.sql"},
	}}
	out := Report(findings)
	for _, want := range []string{
		"web/src/views/MyBookings.vue:2",
		"0002_fixtureseed.sql",
		"courts",
		"0034_fixturebackfill.sql",
		"advisory",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("report does not mention %q:\n%s", want, out)
		}
	}
}

func TestReportOnNoFindings(t *testing.T) {
	out := Report(nil)
	if !strings.Contains(out, "OK") {
		t.Errorf("empty report should say OK, got:\n%s", out)
	}
}

// A tree with no migrations is a broken run, not a clean one. Reporting
// "OK -- no stale citations" for a path that was never read is the
// vacuous-green failure mode; Scan must refuse instead.
func TestScanRefusesATreeItCannotRead(t *testing.T) {
	tests := []struct {
		name  string
		files map[string]string
	}{
		{"no db/migrations directory", map[string]string{"web/a.go": "// see 0002_fixtureseed.sql\n"}},
		{"directory present but holds no migration", map[string]string{"db/migrations/README.md": "not a migration\n"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Scan(writeTree(t, tt.files)); err == nil {
				t.Error("Scan returned nil error; a tree with no migrations must be refused, not reported OK")
			}
		})
	}
}
