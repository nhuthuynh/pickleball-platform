//go:build integration

// T61 — the regression test for the defect 0031 fixes, and the standing guard
// against the next instance of it.
//
// # The bug this pins
//
// `domain.PayableTypeCompetitionEntry` ("competition_entry") was added at T10.6
// with the entire Competitions payment path, and `payments.payable_type`'s
// CHECK constraint was not widened to accept it. The domain said the value was
// valid; Postgres rejected it with 23514. Every Competition-entry payment
// failed against a real database for 22 sprints, while every unit-level test
// passed — an in-memory Payments repository has no CHECK constraint to violate,
// so the fixture was more permissive than the database it stood in for.
//
// # Why this test parses the source instead of listing the types
//
// The obvious version of this test is a table of the four payable types. That
// test would have been written at T10.6 by the ticket that already forgot the
// migration, and it would be the fifth type that broke — a hand-maintained
// list cannot catch a value being added without the list being updated, because
// forgetting to update the list is the same act as forgetting the migration.
//
// So the set under test is read out of `internal/payments/domain/payment.go`
// itself, by parsing it: every `X PayableType = "..."` const declaration. A new
// payable type therefore enters this test's table the moment it is declared,
// with no edit here, and fails until the schema accepts it. That is the only
// arrangement in which this test is a guard rather than a second place to
// forget. (The AST-parsing convention is this project's own, from
// `stringLiteralsInPackageTests` in the three grpcapi identity-fixture tests.)
//
// Requires Docker. Excluded from every Docker-free gate by the build tag —
// which is the whole reason the defect above survived, so: this file proves
// nothing until `make ci-integration` runs it.
package postgres_test

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/nhuthuynh/white-label/internal/payments/domain"
)

// declaredPayableTypes returns every string value declared as a PayableType
// const in internal/payments/domain, read from the source rather than from a
// list maintained here. See this file's header for why.
func declaredPayableTypes(t *testing.T) map[string]string {
	t.Helper()

	path := filepath.Join("..", "..", "domain", "payment.go")
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatalf("parsing %s: %v", path, err)
	}

	found := map[string]string{}
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
			// Only `name PayableType = "literal"`. A const in an untyped
			// group, or one of another type, is not ours.
			ident, ok := vs.Type.(*ast.Ident)
			if !ok || ident.Name != "PayableType" {
				continue
			}
			for i, name := range vs.Names {
				if i >= len(vs.Values) {
					continue
				}
				lit, ok := vs.Values[i].(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					t.Fatalf("%s is declared as a PayableType but its value is not a string literal — "+
						"this test reads the constants from source and cannot evaluate an expression", name.Name)
				}
				// lit.Value carries its quotes.
				found[name.Name] = lit.Value[1 : len(lit.Value)-1]
			}
		}
	}

	if len(found) == 0 {
		t.Fatalf("parsed %s and found no PayableType constants — the parse, not the schema, is what broke", path)
	}
	return found
}

// TestPayableTypeCheckConstraintAcceptsEveryDomainPayableType is the assertion
// that failed to exist for 22 sprints: CLAUDE.md rule 4's two halves, compared
// to each other rather than each to its own author's intent.
func TestPayableTypeCheckConstraintAcceptsEveryDomainPayableType(t *testing.T) {
	ctx := context.Background()
	pool := newPayableTypeTestPool(t, ctx)

	declared := declaredPayableTypes(t)

	for constName, value := range declared {
		t.Run(constName, func(t *testing.T) {
			// A fresh payable_id per case: payments_payable_unique_idx is on
			// (payable_type, payable_id), and a collision there would fail
			// this test with 23505 for a reason that has nothing to do with
			// the CHECK under test.
			if _, err := pool.Exec(ctx, `
				INSERT INTO payments (payable_type, payable_id, amount_cents, currency_code, method, status)
				VALUES ($1, $2, 2500, 'AUD', 'offline', 'paid')
			`, value, uuid.NewString()); err != nil {
				t.Fatalf("domain declares %s = %q as a payable type, but Postgres refuses it: %v\n\n"+
					"The domain and the schema have diverged (CLAUDE.md rule 4). Widen "+
					"payments_payable_type_check in a new migration — see "+
					"db/migrations/0031_payments_competition_entry_payable_type.sql for the "+
					"last time this happened.", constName, value, err)
			}

			// And the value the domain declared is the value the domain
			// accepts: a const whose literal `IsValid()` rejects would make
			// the assertion above meaningless.
			if !domain.PayableType(value).IsValid() {
				t.Fatalf("%s = %q is declared as a PayableType but IsValid() rejects it — "+
					"the constant and the switch in payment.go disagree", constName, value)
			}
		})
	}
}

// TestPayableTypeCheckConstraintRejectsAnUndeclaredType is the other direction,
// and the control for the test above: a CHECK that accepted anything would pass
// it. 'subscription' is deliberately chosen — `IsValid()`'s own test cases name
// it as a type this codebase does NOT recognise, so the schema must not either.
func TestPayableTypeCheckConstraintRejectsAnUndeclaredType(t *testing.T) {
	ctx := context.Background()
	pool := newPayableTypeTestPool(t, ctx)

	const undeclared = "subscription"
	if domain.PayableType(undeclared).IsValid() {
		t.Fatalf("this test assumes %q is not a payable type, and the domain now says it is — "+
			"if it was genuinely added, it needs a migration and this control needs a different value", undeclared)
	}

	_, err := pool.Exec(ctx, `
		INSERT INTO payments (payable_type, payable_id, amount_cents, currency_code, method, status)
		VALUES ($1, $2, 2500, 'AUD', 'offline', 'paid')
	`, undeclared, uuid.NewString())
	if err == nil {
		t.Fatalf("Postgres accepted payable_type = %q, which the domain rejects — "+
			"the CHECK constraint is permitting values no code path can produce, "+
			"which makes the test above prove nothing", undeclared)
	}
}

// newPayableTypeTestPool boots a Postgres container with the schema applied,
// mirroring every other integration test in this package. It reuses
// smoke_integration_test.go's waitForReady/applyMigrations rather than
// re-declaring them.
func newPayableTypeTestPool(t *testing.T, ctx context.Context) *pgxpool.Pool {
	t.Helper()

	container, err := tcpostgres.Run(ctx, "postgres:16-alpine",
		tcpostgres.WithDatabase("pickleball"),
		tcpostgres.WithUsername("pickleball"),
		tcpostgres.WithPassword("pickleball"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(context.Background()); err != nil {
			t.Logf("failed to terminate postgres container: %v", err)
		}
	})

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("failed to open pool: %v", err)
	}
	t.Cleanup(pool.Close)

	waitForReady(t, ctx, pool)
	applyMigrations(t, ctx, pool)

	return pool
}
