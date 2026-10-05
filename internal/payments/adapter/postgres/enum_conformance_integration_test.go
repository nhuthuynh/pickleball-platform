//go:build integration

// T62.5 (closes #311) — every enum-shaped CHECK constraint in the schema,
// checked against the domain type that governs it, against a real database.
//
// # What this replaces, and why it is not what #311 asked for
//
// #311 asked for four more tests in the shape of
// `payable_type_conformance_integration_test.go` — one per pair, from the list
// of five pairs the issue named.
//
// **The schema has twenty-two enum-shaped CHECK constraints, not five.**
// Counted at T62.5 by enumerating them; the mapping below is the full set. So
// building #311 as filed would have shipped a hand-maintained list that was
// already incomplete by a factor of four — the identical failure the issue was
// filed about, and the third consecutive instance of it in this project
// (#308's subject lists said 12 and 8 where 22 and 21 existed; #311's pair
// list said 5 where 22 existed).
//
// `CLAUDE.md` says the same thing of `make gate-coverage`: *"there is no
// package list in the tool, and adding one is the one change that would defeat
// it."* So this test derives side A from the **live database** and inverts the
// burden of proof on side B: **an enum constraint absent from the mapping
// fails this test.** The mapping may be incomplete; it cannot be incomplete
// and quiet.
//
// # Why the live database and not db/migrations
//
// Parsing the SQL would have been easier and would have been wrong. A static
// parse of `CREATE TABLE payments` reports
// `payable_type IN ('booking','registration','no_show_fee')` — T6.4's
// definition — and misses that `0031` dropped and re-added that constraint
// with `competition_entry`. What the application actually meets is the
// post-migration state, which only Postgres can report.
//
// # Why this file lives in internal/payments/adapter/postgres
//
// It is cross-cutting — it covers `bookings`, `games`, `registrations`,
// `competitions`, `payments`, `pricing_rules`, `discount_rules` and
// `recurring_hire_templates` — and no bounded context owns it. It sits here
// because this package already applies every migration in `db/migrations`,
// already boots a container, and already holds T61's single-pair precedent,
// so the alternative was a new package whose only purpose was to own one test.
// It imports no other context: the domain sets are read by **parsing source
// files**, so the dependency rule (CLAUDE.md rule 3) is untouched — nothing
// here imports another context's package.
//
// The parsing and comparison halves live in `tools/enumconformance` and are
// Docker-free and unit-tested under `make test-tools`. This file supplies only
// the database and the mapping.
//
// Requires Docker. Excluded from every Docker-free gate by the build tag, which
// is how `payable_type`'s divergence survived from T10.6 to T61 — so: this
// proves nothing until `make ci-integration` runs it.
package postgres_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/nhuthuynh/white-label/tools/enumconformance"
)

// enumPairs maps every enum-shaped CHECK constraint in the schema to the
// domain type that governs it.
//
// **Adding a row here is the fix when this test fails on an unmapped column.
// Removing the failing column from the check is not.** A column with no single
// governing domain type gets a row with Why set, so an exemption is visible
// rather than absent.
//
// Paths are relative to this file. The repo's convention is one Go const block
// per enum, in the context that owns it.
var enumPairs = []enumconformance.Pair{
	// Booking
	{Table: "bookings", Column: "source", File: "booking/domain/booking.go", TypeName: "Source"},
	{Table: "bookings", Column: "status", File: "booking/domain/booking.go", TypeName: "Status"},
	{Table: "pricing_rules", Column: "band", File: "booking/domain/pricing.go", TypeName: "Band"},
	{Table: "discount_rules", Column: "discount_type", File: "booking/domain/discount.go", TypeName: "DiscountType"},
	{Table: "discount_rules", Column: "end_condition_kind", File: "booking/domain/discount.go", TypeName: "EndConditionKind"},
	{Table: "recurring_hire_templates", Column: "end_condition_kind", File: "booking/domain/discount.go", TypeName: "EndConditionKind"},
	{Table: "recurring_hire_templates", Column: "status", File: "booking/domain/recurring_hire_template.go", TypeName: "RecurringHireStatus"},

	// Social Play
	{Table: "games", Column: "status", File: "socialplay/domain/game.go", TypeName: "Status"},
	// Added after the first real run: this column was absent from the mapping
	// that a static parse of db/migrations produced, and the unmapped-is-a-
	// failure inversion is what surfaced it. Exactly the case the inversion
	// exists for.
	{Table: "games", Column: "payment_method", File: "socialplay/domain/payment_method.go", TypeName: "PaymentMethod"},
	{Table: "registrations", Column: "source", File: "socialplay/domain/registration.go", TypeName: "RegistrationSource"},
	{Table: "registrations", Column: "status", File: "socialplay/domain/registration.go", TypeName: "RegistrationStatus"},
	{Table: "registrations", Column: "payment_status", File: "socialplay/domain/registration.go", TypeName: "PaymentStatus"},
	{Table: "waitlist_entries", Column: "status", File: "socialplay/domain/waitlist.go", TypeName: "WaitlistStatus"},

	// Competitions
	{Table: "competitions", Column: "status", File: "competitions/domain/competition.go", TypeName: "Status"},
	{Table: "competitions", Column: "format", File: "competitions/domain/format.go", TypeName: "Format"},
	{Table: "competitions", Column: "payment_method", File: "competitions/domain/payment_method.go", TypeName: "PaymentMethod"},
	{Table: "competition_entries", Column: "source", File: "competitions/domain/entry.go", TypeName: "EntrySource"},
	{Table: "competition_entries", Column: "status", File: "competitions/domain/entry.go", TypeName: "EntryStatus"},
	{Table: "competition_entries", Column: "payment_status", File: "competitions/domain/entry.go", TypeName: "PaymentStatus"},

	// Payments
	{Table: "payments", Column: "payable_type", File: "payments/domain/payment.go", TypeName: "PayableType"},
	{Table: "payments", Column: "method", File: "payments/domain/payment.go", TypeName: "Method"},
	{Table: "payments", Column: "status", File: "payments/domain/payment.go", TypeName: "Status"},
}

// TestEveryEnumCheckAgreesWithItsDomainType is CLAUDE.md rule 4's two halves
// compared to each other rather than each to its own author's intent.
func TestEveryEnumCheckAgreesWithItsDomainType(t *testing.T) {
	ctx := context.Background()
	pool := newPayableTypeTestPool(t, ctx)

	// Side A: every CHECK constraint the database actually holds, rendered by
	// Postgres itself after all migrations have applied.
	rows, err := pool.Query(ctx, `
		SELECT rel.relname, pg_get_constraintdef(con.oid)
		FROM pg_constraint con
		JOIN pg_class rel ON rel.oid = con.conrelid
		JOIN pg_namespace nsp ON nsp.oid = rel.relnamespace
		WHERE con.contype = 'c' AND nsp.nspname = 'public'
	`)
	if err != nil {
		t.Fatalf("listing CHECK constraints: %v", err)
	}
	defer rows.Close()

	var constraints []enumconformance.Constraint
	total := 0
	for rows.Next() {
		var table, def string
		if err := rows.Scan(&table, &def); err != nil {
			t.Fatalf("scanning constraint: %v", err)
		}
		total++
		col, vals, ok := enumconformance.ParseConstraintDef(def)
		if !ok {
			continue // not enum-shaped: amount_cents > 0, cardinality(...) > 0, ...
		}
		constraints = append(constraints, enumconformance.Constraint{Table: table, Column: col, Values: vals})
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterating constraints: %v", err)
	}

	// A run that found no constraints would otherwise pass silently, which is
	// the failure mode of every check in this repo that went stale.
	if len(constraints) < len(enumPairs) {
		t.Fatalf("found only %d enum-shaped CHECK constraints among %d total — "+
			"expected at least the %d this test maps. Either the query or "+
			"ParseConstraintDef has stopped matching what Postgres renders, which would make "+
			"this test vacuous rather than green", len(constraints), total, len(enumPairs))
	}
	t.Logf("checked %d enum-shaped CHECK constraints (of %d CHECKs) against %d mapped domain types",
		len(constraints), total, len(enumPairs))

	// Side B: the domain's declared constants, parsed from source.
	declared := func(p enumconformance.Pair) (map[string]string, error) {
		return enumconformance.DeclaredConstants(filepath.Join("..", "..", "..", p.File), p.TypeName)
	}

	// T63.4 — check the mapping itself before comparing anything. This half is
	// Docker-free and is what would have caught T62's five wrong rows (three
	// named a file that does not exist, two a type that does not exist).
	// Ambiguous type names are reported, not failed: `Status` is declared in
	// four bounded contexts, so the name alone cannot identify the right one,
	// and only a human can confirm a given row's choice.
	mapFindings, notes := enumconformance.ValidateMapping(enumPairs, declared)
	for _, n := range notes {
		t.Logf("mapping note: %s", n)
	}
	for _, f := range mapFindings {
		t.Errorf("mapping: %s", f)
	}
	if len(mapFindings) > 0 {
		t.FailNow() // comparing against a broken mapping would report noise
	}

	findings, err := enumconformance.Compare(constraints, enumPairs, declared)
	if err != nil {
		t.Fatalf("comparing: %v", err)
	}
	for _, f := range findings {
		t.Errorf("%s", f)
	}
	if len(findings) > 0 {
		t.Logf("\n%d enum constraint(s) disagree with their domain type, or are unmapped.\n"+
			"Fix by adding a migration, correcting the domain, or adding a row to enumPairs —\n"+
			"never by removing a column from enumPairs. #311 listed 5 of the 22 that exist,\n"+
			"which is the failure this test's unmapped-is-a-failure inversion prevents.", len(findings))
	}
}
