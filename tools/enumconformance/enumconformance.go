// Package enumconformance answers one question mechanically: **does every
// enum-shaped CHECK constraint in the schema accept exactly the values its
// domain type declares?**
//
// # Why this exists
//
// `payments.payable_type`'s CHECK was never widened for `competition_entry`,
// a value `internal/payments/domain` had accepted since T10.6. Every
// Competition-entry payment therefore failed against a real database with
// SQLSTATE 23514, from the sprint the feature shipped until T61 found it —
// and every unit-level test passed throughout, because the in-memory Payments
// repository has no CHECK constraint to violate. The fixture was more
// permissive than the database it stood in for, so the tests proved the
// routing and hid the storage (`docs/process/t61-retro.md` §3, `CLAUDE.md`'s
// gotcha of the same name).
//
// T61 closed that one pair with an insert-based test. Issue #311 asked for the
// same treatment on the pairs it listed.
//
// # Why this is a derivation and not #311's list
//
// **#311 named five pairs. The schema has twenty-two.** Counted at T62.5 by
// enumerating every column-level `CHECK (col IN (...))` across
// `db/migrations`, which is the check this package performs.
//
// That is the identical failure the issue itself was filed about — and the
// third time in three sprints that a hand-written list in this project was
// incomplete on the day it was written (#308's subject lists: 12 where 22
// existed, 8 where 21 existed; #311's pair list: 5 where 22 existed). It is
// also what `CLAUDE.md` says of `make gate-coverage`: *"there is no package
// list in the tool, and adding one is the one change that would defeat it —
// three sprints running shipped a hand-written glob that was stale before its
// sprint ended."*
//
// So side A is **derived from the live database**, never from a list and never
// from parsing SQL:
//
//   - Parsing `db/migrations/*.sql` would have been easier and would have been
//     wrong. A static parse of `CREATE TABLE payments` reports
//     `payable_type IN ('booking','registration','no_show_fee')` — the T6.4
//     definition — and misses that `0031` dropped and re-added the constraint
//     with `competition_entry`. The authoritative set is what Postgres holds
//     after every migration has applied, which is what
//     `pg_get_constraintdef()` returns.
//
// Side B is the domain's declared constants, parsed from source.
//
// # The mapping, and why it cannot go stale silently
//
// `table.column` to Go type is the one thing that cannot be derived: nothing
// in the schema says `registrations.status` corresponds to
// `socialplay/domain.RegistrationStatus`. So the mapping is explicit — and
// **an enum-shaped constraint the mapping does not cover is a failure**, not a
// skip. A column added next sprint fails this check until someone maps it or
// records why it is exempt. That inversion is the whole design: the list may
// be incomplete, but it cannot be incomplete *and quiet*.
package enumconformance

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"sort"
	"strings"
)

// Pair is one schema column and the domain type that is supposed to govern it.
type Pair struct {
	Table    string
	Column   string
	File     string // repo-relative path to the Go file declaring the type
	TypeName string
	// Why, when non-empty, records a deliberate exemption: the column is
	// enum-shaped but has no single governing domain type. It is still
	// reported, so an exemption is visible rather than absent.
	Why string
}

func (p Pair) Col() string { return p.Table + "." + p.Column }

// Constraint is one enum-shaped CHECK as the database actually holds it.
type Constraint struct {
	Table  string
	Column string
	Values []string
}

func (c Constraint) Col() string { return c.Table + "." + c.Column }

// These pull the column and value list out of a constraint definition as
// pg_get_constraintdef renders it. Postgres normalises the `IN (...)` written
// in db/migrations into `= ANY (ARRAY[...])`, so these match the rendered form
// and never the source form. A definition they cannot parse is reported, never
// ignored — see Compare, and see colRe for the two renderings and the mistake
// that made matching both necessary.
var (
	anyArrayRe = regexp.MustCompile(`ARRAY\[(.*?)\]`)
	quotedRe   = regexp.MustCompile(`'([^']*)'`)
	// Both renderings, because Postgres casts the column only when it needs
	// to. A `text` column renders bare —
	//
	//	CHECK ((source = ANY (ARRAY['game'::text, ...])))
	//
	// while a `varchar` one renders cast —
	//
	//	CHECK (((kind)::text = ANY ((ARRAY['a'::character varying, ...])::text[])))
	//
	// Every enum column in this schema is `text` today, so only the first form
	// occurs. T62.5's first draft matched only the SECOND, having been written
	// against an invented example rather than a real one, and parsed 0 of 56
	// CHECK constraints — caught not by the unit tests, which pinned the same
	// invented string, but by the integration test's vacuity guard. Both forms
	// are matched here and both are pinned in the unit tests.
	colRe = regexp.MustCompile(`\(+\(?([a-z_][a-z0-9_]*)\)?(?:::[a-z ]+)?\s*=\s*ANY`)
)

// ParseConstraintDef extracts the column and accepted values from one
// pg_get_constraintdef() string. ok is false when the definition is not an
// enum-shaped list, which callers must treat as "not an enum constraint"
// rather than as an empty set.
func ParseConstraintDef(def string) (column string, values []string, ok bool) {
	arr := anyArrayRe.FindStringSubmatch(def)
	if arr == nil {
		return "", nil, false
	}
	for _, m := range quotedRe.FindAllStringSubmatch(arr[1], -1) {
		values = append(values, m[1])
	}
	if len(values) == 0 {
		return "", nil, false
	}
	if c := colRe.FindStringSubmatch(def); c != nil {
		column = c[1]
	}
	sort.Strings(values)
	return column, values, column != ""
}

// DeclaredConstants returns every string constant declared with the named type
// in a Go source file, keyed by constant name.
//
// Parsing the source rather than reflecting over a package is deliberate and is
// the same technique `payable_type_conformance_integration_test.go` and the
// three grpcapi identity-fixture tests already use: a test that imports the
// package would need one import per context and would drag five domain
// packages into one test binary, and a reflective enumeration of typed string
// constants is not available at run time anyway.
func DeclaredConstants(path, typeName string) (map[string]string, error) {
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	out := map[string]string{}
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		for _, spec := range gen.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			ident, ok := vs.Type.(*ast.Ident)
			if !ok || ident.Name != typeName {
				continue
			}
			for i, name := range vs.Names {
				if i >= len(vs.Values) {
					continue
				}
				lit, ok := vs.Values[i].(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					return nil, fmt.Errorf("%s is declared %s but its value is not a string literal", name.Name, typeName)
				}
				out[name.Name] = strings.Trim(lit.Value, `"`)
			}
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("found no %s constants in %s", typeName, path)
	}
	return out, nil
}

// ValidateMapping checks the mapping itself, before any database is consulted.
//
// # Why this is narrower than T63.4 first proposed, and why that is deliberate
//
// T63's plan asked for a guard against "a plausible-but-wrong row", on the
// evidence that five of the first twenty-one rows were wrong. **Re-verifying
// that finding before building to it — per `t62-retro.md` recommendation 4 —
// showed the five were not of that shape.** Three named a file that does not
// exist and two named a type that does not exist, and both already fail loudly:
// DeclaredConstants errors rather than returning an empty set, so a wrong path
// or a wrong type name cannot pass silently.
//
// The plan also asserted that a context-ownership check would catch "the shape
// four of the five errors had". That was wrong, and the count that shows it is
// zero: every mapped row currently declares its type inside the bounded context
// that owns its table.
//
// So a context-ownership check would be machinery for a failure that has never
// occurred — and it would need a hand-maintained `table -> context` list, which
// is the exact artifact this package exists to avoid. **Declined, with the
// reasoning recorded** rather than built.
//
// # What is genuinely uncovered, and is checked here
//
// One case survives and is real: **`Status` is declared in four different
// bounded contexts** (booking, socialplay, competitions, payments), several
// with overlapping values. A row for `games.status` pointing at Competitions'
// `Status` would find real constants — `scheduled`, `cancelled` — that happen
// to match, and nothing would notice.
//
// That cannot be ruled out without knowing which context owns which table, so
// it is **reported rather than failed**: an ambiguous type name is surfaced in
// the test's output, naming the file the row chose, so a reader can check the
// one thing a machine cannot. The two hard errors below do fail.
func ValidateMapping(pairs []Pair, declared func(Pair) (map[string]string, error)) (findings []Finding, notes []string) {
	seen := map[string]Pair{}
	for _, p := range pairs {
		// Hard error: the same column mapped twice. One correct row would
		// otherwise mask an incorrect one, since Compare indexes by column.
		if prev, dup := seen[p.Col()]; dup {
			findings = append(findings, Finding{p.Col(), fmt.Sprintf(
				"mapped twice — to %s and to %s. One row would mask the other",
				prev.TypeName, p.TypeName)})
			continue
		}
		seen[p.Col()] = p

		if p.Why != "" {
			continue
		}

		// Hard error: the row's file and type must resolve. This is asserted
		// here, not only at comparison time, so a mapping is checkable without
		// a database — which is what makes this function Docker-free.
		if _, err := declared(p); err != nil {
			findings = append(findings, Finding{p.Col(), fmt.Sprintf(
				"mapping does not resolve: %v", err)})
		}
	}

	// Note, not error: type names used by more than one row, which is the
	// signature of an ambiguous name like Status.
	byType := map[string][]string{}
	for _, p := range pairs {
		if p.TypeName != "" {
			byType[p.TypeName] = append(byType[p.TypeName], p.Col()+" <- "+p.File)
		}
	}
	for ty, uses := range byType {
		if len(uses) > 1 {
			sort.Strings(uses)
			notes = append(notes, fmt.Sprintf(
				"%q is mapped by %d rows, so the name alone is ambiguous — each row's File "+
					"is what disambiguates it, and only a human can confirm the choice: %s",
				ty, len(uses), strings.Join(uses, "; ")))
		}
	}
	sort.Strings(notes)
	sort.Slice(findings, func(i, j int) bool { return findings[i].Col < findings[j].Col })
	return findings, notes
}

// Finding is one disagreement, phrased so the message alone says what to do.
type Finding struct {
	Col  string
	What string
}

func (f Finding) String() string { return fmt.Sprintf("%s: %s", f.Col, f.What) }

// Compare checks every constraint against the mapping.
//
// It reports three distinct problems, and the third is the one that keeps the
// mapping honest:
//
//  1. a mapped pair whose sets disagree — the T10.6 defect, in whichever
//     direction it occurs;
//  2. a mapped pair with no constraint in the database at all;
//  3. **an enum-shaped constraint the mapping does not cover** — so a column
//     added next sprint fails until it is mapped or explicitly exempted.
func Compare(constraints []Constraint, pairs []Pair, declared func(Pair) (map[string]string, error)) ([]Finding, error) {
	byCol := map[string]Constraint{}
	for _, c := range constraints {
		byCol[c.Col()] = c
	}
	mapped := map[string]Pair{}
	for _, p := range pairs {
		mapped[p.Col()] = p
	}

	var findings []Finding

	for _, p := range pairs {
		c, ok := byCol[p.Col()]
		if !ok {
			findings = append(findings, Finding{p.Col(), "mapped to " + p.TypeName +
				" but the database has no enum-shaped CHECK on this column — " +
				"either the constraint was dropped or the mapping names the wrong column"})
			continue
		}
		if p.Why != "" {
			continue // exempt, but reported by the caller
		}
		consts, err := declared(p)
		if err != nil {
			return nil, err
		}
		want := make([]string, 0, len(consts))
		for _, v := range consts {
			want = append(want, v)
		}
		sort.Strings(want)
		if !equal(want, c.Values) {
			findings = append(findings, Finding{p.Col(), fmt.Sprintf(
				"the domain and the schema disagree (CLAUDE.md rule 4)\n"+
					"      domain %s declares: %v\n"+
					"      the CHECK accepts:  %v\n"+
					"      → add a migration widening the CHECK, or remove the value from %s",
				p.TypeName, want, c.Values, p.File)})
		}
	}

	for _, c := range constraints {
		if _, ok := mapped[c.Col()]; !ok {
			findings = append(findings, Finding{c.Col(), fmt.Sprintf(
				"enum-shaped CHECK accepting %v is not in this test's mapping — "+
					"map it to its domain type, or add a Pair with Why set to record why it has none. "+
					"An unmapped enum column is a failure by design: #311 listed 5 of the 22 that exist, "+
					"which is the failure mode this inversion prevents", c.Values)})
		}
	}

	sort.Slice(findings, func(i, j int) bool { return findings[i].Col < findings[j].Col })
	return findings, nil
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
