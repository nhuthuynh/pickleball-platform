package enumconformance_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nhuthuynh/white-label/tools/enumconformance"
)

// Two renderings of pg_get_constraintdef output.
//
// textColumnDef is COPIED VERBATIM from a real Postgres 16 run against this
// repo's own migrations (captured by a throwaway probe test at T62.5). That
// provenance is the point: T62.5's first draft pinned an INVENTED string of
// the varcharColumnDef shape, with a comment claiming it was real. The parser
// written against it matched 0 of the 56 CHECK constraints the live database
// holds, and **the unit tests did not catch that, because they asserted
// against the same invention.** What caught it was the integration test's
// vacuity guard — "found only 0 enum-shaped constraints among 56".
//
// The lesson is narrower than "test against reality": a fixture and the code
// under test, both written from the same wrong assumption, agree perfectly.
// Only something that touches the real system can break the tie.
const (
	textColumnDef    = `CHECK ((source = ANY (ARRAY['recurring_hire'::text, 'individual'::text, 'game'::text, 'competition'::text])))`
	varcharColumnDef = `CHECK (((payable_type)::text = ANY ((ARRAY['booking'::character varying, 'registration'::character varying])::text[])))`
)

func TestParseConstraintDefReadsBothPostgresRenderings(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name, def, wantCol string
		wantVals           []string
	}{
		{
			name: "text column, rendered bare — the form this schema actually uses",
			def:  textColumnDef, wantCol: "source",
			wantVals: []string{"competition", "game", "individual", "recurring_hire"},
		},
		{
			name: "varchar column, rendered with a cast",
			def:  varcharColumnDef, wantCol: "payable_type",
			wantVals: []string{"booking", "registration"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			col, vals, ok := enumconformance.ParseConstraintDef(tc.def)
			if !ok {
				t.Fatalf("failed to parse: %s", tc.def)
			}
			if col != tc.wantCol {
				t.Fatalf("column = %q, want %q", col, tc.wantCol)
			}
			if strings.Join(vals, ",") != strings.Join(tc.wantVals, ",") {
				t.Fatalf("values = %v, want %v (sorted)", vals, tc.wantVals)
			}
		})
	}
}

// A CHECK that is not an enum list must report "not an enum", never an empty
// set — an empty set would silently compare equal to a domain type with no
// constants and pass.
func TestNonEnumConstraintsAreNotMistakenForEmptySets(t *testing.T) {
	t.Parallel()

	for _, def := range []string{
		`CHECK ((amount_cents > 0))`,
		`CHECK ((cardinality(court_ids) > 0))`,
		`CHECK ((guest_count >= 0))`,
	} {
		if _, _, ok := enumconformance.ParseConstraintDef(def); ok {
			t.Fatalf("%q was parsed as an enum constraint", def)
		}
	}
}

func TestDeclaredConstantsReadsTypedStringConstants(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "x.go")
	src := `package domain

type Status string

const (
	StatusUnpaid   Status = "unpaid"
	StatusPaid     Status = "paid"
	StatusRefunded Status = "refunded"
)

type Other string

const OtherThing Other = "ignored"
`
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := enumconformance.DeclaredConstants(path, "Status")
	if err != nil {
		t.Fatalf("DeclaredConstants: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d constants, want 3 (a constant of another type must not leak in): %v", len(got), got)
	}
	if got["StatusPaid"] != "paid" {
		t.Fatalf("StatusPaid = %q, want paid", got["StatusPaid"])
	}
}

func TestDeclaredConstantsFailsLoudlyWhenItFindsNothing(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "x.go")
	if err := os.WriteFile(path, []byte("package domain\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := enumconformance.DeclaredConstants(path, "Status"); err == nil {
		t.Fatal("finding no constants must be an error — a silent empty set would compare equal to an empty CHECK and pass")
	}
}

func decl(m map[string]map[string]string) func(enumconformance.Pair) (map[string]string, error) {
	return func(p enumconformance.Pair) (map[string]string, error) { return m[p.TypeName], nil }
}

func TestCompareFindsADivergenceInEitherDirection(t *testing.T) {
	t.Parallel()

	cs := []enumconformance.Constraint{{Table: "payments", Column: "payable_type", Values: []string{"booking", "registration"}}}
	ps := []enumconformance.Pair{{Table: "payments", Column: "payable_type", File: "p.go", TypeName: "PayableType"}}

	// domain has a value the schema refuses — the T10.6 defect.
	f, err := enumconformance.Compare(cs, ps, decl(map[string]map[string]string{
		"PayableType": {"A": "booking", "B": "registration", "C": "competition_entry"},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if len(f) != 1 || !strings.Contains(f[0].What, "disagree") {
		t.Fatalf("want one disagreement finding, got %v", f)
	}

	// schema accepts a value the domain does not declare — the other direction.
	f, err = enumconformance.Compare(cs, ps, decl(map[string]map[string]string{
		"PayableType": {"A": "booking"},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if len(f) != 1 {
		t.Fatalf("want one finding for the reverse direction, got %v", f)
	}
}

// The inversion that keeps the mapping honest, and the test most worth
// keeping: #311 listed 5 of the 22 enum constraints that exist, so an unmapped
// column must FAIL rather than be skipped.
func TestUnmappedEnumConstraintFails(t *testing.T) {
	t.Parallel()

	cs := []enumconformance.Constraint{
		{Table: "payments", Column: "payable_type", Values: []string{"booking"}},
		{Table: "pricing_rules", Column: "band", Values: []string{"peak", "weekday", "weekend"}},
	}
	ps := []enumconformance.Pair{{Table: "payments", Column: "payable_type", File: "p.go", TypeName: "PayableType"}}

	f, err := enumconformance.Compare(cs, ps, decl(map[string]map[string]string{"PayableType": {"A": "booking"}}))
	if err != nil {
		t.Fatal(err)
	}
	if len(f) != 1 || f[0].Col != "pricing_rules.band" {
		t.Fatalf("an enum constraint absent from the mapping must fail, naming the column; got %v", f)
	}
	if !strings.Contains(f[0].What, "5 of the 22") {
		t.Fatalf("the finding should say why this is a failure by design; got %q", f[0].What)
	}
}

// A mapping entry whose constraint has vanished is its own failure — otherwise
// dropping a CHECK would silently make this check pass.
func TestMappedPairWithNoConstraintFails(t *testing.T) {
	t.Parallel()

	ps := []enumconformance.Pair{{Table: "payments", Column: "gone", File: "p.go", TypeName: "T"}}
	f, err := enumconformance.Compare(nil, ps, decl(nil))
	if err != nil {
		t.Fatal(err)
	}
	if len(f) != 1 || !strings.Contains(f[0].What, "no enum-shaped CHECK") {
		t.Fatalf("want a finding for the missing constraint, got %v", f)
	}
}

// An exemption is honoured but must still be declared explicitly.
func TestExemptPairIsNotCompared(t *testing.T) {
	t.Parallel()

	cs := []enumconformance.Constraint{{Table: "t", Column: "c", Values: []string{"x"}}}
	ps := []enumconformance.Pair{{Table: "t", Column: "c", Why: "no single governing domain type"}}

	f, err := enumconformance.Compare(cs, ps, decl(nil))
	if err != nil {
		t.Fatal(err)
	}
	if len(f) != 0 {
		t.Fatalf("an explicitly exempt pair must not be compared, got %v", f)
	}
}

// The control.
func TestAgreeingSetsPass(t *testing.T) {
	t.Parallel()

	cs := []enumconformance.Constraint{{Table: "payments", Column: "method", Values: []string{"offline", "online"}}}
	ps := []enumconformance.Pair{{Table: "payments", Column: "method", File: "p.go", TypeName: "Method"}}

	f, err := enumconformance.Compare(cs, ps, decl(map[string]map[string]string{
		"Method": {"MethodOnline": "online", "MethodOffline": "offline"},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if len(f) != 0 {
		t.Fatalf("agreeing sets must pass, got %v", f)
	}
}

// --- T63.4: the mapping checked without a database ---

func declOK(m map[string]map[string]string) func(enumconformance.Pair) (map[string]string, error) {
	return func(p enumconformance.Pair) (map[string]string, error) {
		v, ok := m[p.TypeName]
		if !ok {
			return nil, errors.New("found no " + p.TypeName + " constants")
		}
		return v, nil
	}
}

// The same column mapped twice is a hard error: Compare indexes by column, so
// one correct row would silently mask an incorrect one.
func TestValidateMappingRejectsADuplicateColumn(t *testing.T) {
	t.Parallel()

	f, _ := enumconformance.ValidateMapping([]enumconformance.Pair{
		{Table: "payments", Column: "status", File: "a.go", TypeName: "Status"},
		{Table: "payments", Column: "status", File: "b.go", TypeName: "Other"},
	}, declOK(map[string]map[string]string{"Status": {"A": "a"}, "Other": {"B": "b"}}))

	if len(f) != 1 || !strings.Contains(f[0].What, "mapped twice") {
		t.Fatalf("a duplicated column must fail, got %v", f)
	}
}

// A row whose file or type does not resolve fails here, without a database —
// which is the shape all five of T62's real mapping errors had.
func TestValidateMappingRejectsAnUnresolvableRow(t *testing.T) {
	t.Parallel()

	f, _ := enumconformance.ValidateMapping([]enumconformance.Pair{
		{Table: "games", Column: "status", File: "socialplay/domain/game.go", TypeName: "GameStatus"},
	}, declOK(nil)) // nothing resolves

	if len(f) != 1 || !strings.Contains(f[0].What, "does not resolve") {
		t.Fatalf("an unresolvable row must fail, got %v", f)
	}
}

// An ambiguous type name is REPORTED, not failed. `Status` is declared in four
// bounded contexts, so the name alone cannot identify the right one — but
// whether a given row chose correctly is not machine-decidable, and failing on
// ambiguity would fail the 22-row mapping this repo legitimately has.
func TestValidateMappingReportsAnAmbiguousTypeNameWithoutFailing(t *testing.T) {
	t.Parallel()

	pairs := []enumconformance.Pair{
		{Table: "bookings", Column: "status", File: "booking/domain/booking.go", TypeName: "Status"},
		{Table: "payments", Column: "status", File: "payments/domain/payment.go", TypeName: "Status"},
	}
	f, notes := enumconformance.ValidateMapping(pairs,
		declOK(map[string]map[string]string{"Status": {"A": "a"}}))

	if len(f) != 0 {
		t.Fatalf("ambiguity must not fail — this repo has 22 legitimate rows including four Status types; got %v", f)
	}
	if len(notes) != 1 || !strings.Contains(notes[0], "ambiguous") {
		t.Fatalf("ambiguity must be reported so a human can check the choice; got %v", notes)
	}
	if !strings.Contains(notes[0], "booking/domain/booking.go") || !strings.Contains(notes[0], "payments/domain/payment.go") {
		t.Fatalf("the note must name each row's File, since that is what disambiguates; got %q", notes[0])
	}
}

// An explicitly exempt row is not resolved, but is still checked for duplication.
func TestValidateMappingSkipsResolutionForAnExemptRow(t *testing.T) {
	t.Parallel()

	f, _ := enumconformance.ValidateMapping([]enumconformance.Pair{
		{Table: "t", Column: "c", Why: "no single governing domain type"},
	}, declOK(nil))

	if len(f) != 0 {
		t.Fatalf("an exempt row must not be resolved, got %v", f)
	}
}

// The control.
func TestValidateMappingPassesAGoodMapping(t *testing.T) {
	t.Parallel()

	f, notes := enumconformance.ValidateMapping([]enumconformance.Pair{
		{Table: "payments", Column: "method", File: "payments/domain/payment.go", TypeName: "Method"},
	}, declOK(map[string]map[string]string{"Method": {"MethodOnline": "online"}}))

	if len(f) != 0 || len(notes) != 0 {
		t.Fatalf("a good unambiguous mapping must be silent, got findings=%v notes=%v", f, notes)
	}
}
