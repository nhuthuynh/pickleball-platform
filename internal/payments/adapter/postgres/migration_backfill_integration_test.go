//go:build integration

// This is a "large" test (Google's testing-pyramid terminology): it starts a
// real Postgres in a container, applies every migration UP TO 0023 (the
// schema as it existed before this ticket), seeds payments rows directly
// against that pre-T28.1 text-column shape, applies
// db/migrations/0024_payments_recorded_by_user_id_uuid.sql on top, and
// asserts against the raw column afterward — not through app.Service or
// port.Repository, both of which only understand the POST-migration shape,
// so this file is the one place in this package that talks to the payments
// table directly.
//
// T28.1 (closes the Payments third of #164) instruction 7: this is CLAUDE.md
// rule 10 ("a single successful run is not proof of reliability... re-run
// non-deterministic tests") applied to a migration specifically — a
// mutation-check, not an assertion that merely runs the migration and
// checks it didn't error. Both seeded rows are read back and their EXACT
// values are compared: a resolvable subject must map to the correct
// identity_users.id (proving the join key is right, not just that some
// value landed), and an orphaned subject must be NULL — specifically NULL,
// checked via the driver's pgtype.UUID.Valid flag, not empty-string, not
// silently dropped (row count is asserted too, so an orphan being DELETEd
// instead of nulled would also fail this test).
//
// Excluded from `make test-domain` and plain `go test ./...` by the
// integration build tag; run it with `go test -tags=integration ./...` or
// `make test`. Requires Docker — this authoring environment has none (the
// standing T4 LESSONS.md gap every other `-tags=integration` file in this
// repo already discloses); see the PR description for how this was
// verified in this environment instead.
package postgres_test

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

// resolvableSubject is seeded into identity_users.subject AND, pre-migration,
// directly into a payments row's recorded_by_user_id text column — the
// "actor who called CreateUser before recording a Payment" case the backfill
// is supposed to resolve.
const resolvableSubject = "auth0|resolvable-actor"

// resolvableUserID is the identity_users.id resolvableSubject maps to. The
// test asserts the backfilled column equals EXACTLY this value, not merely
// "some non-null uuid" — proving the join key, not just that a join ran.
const resolvableUserID = "11111111-1111-1111-1111-111111111111"

// orphanSubject is written into a second payments row's recorded_by_user_id
// text column but is NEVER seeded into identity_users.subject — the "actor
// recorded a Payment before ever calling CreateUser" case ADR-0017 Decision
// 3 rules on: the backfilled column must be NULL, not dropped, not "".
const orphanSubject = "auth0|never-registered"

func TestMigration0024_BackfillsResolvableSubjectAndNullsOrphan(t *testing.T) {
	ctx := context.Background()

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
	defer pool.Close()

	waitForReady(t, ctx, pool)

	// Step 1: apply every migration BEFORE 0024 — the schema exactly as it
	// existed prior to this ticket, with payments.recorded_by_user_id still
	// `text`.
	applyMigrationsBefore(t, ctx, pool, "0024_payments_recorded_by_user_id_uuid.sql")

	// Step 2: seed identity_users with the ONE resolvable actor, and
	// payments with two rows — one whose recorded_by_user_id matches it, one
	// whose recorded_by_user_id matches nobody. Both are Booking-payable
	// rows (this table has no live join to Booking, so any payable_id shape
	// is fine) with distinct payable_id values so payments_payable_unique_idx
	// does not collide them.
	if _, err := pool.Exec(ctx, `
		INSERT INTO identity_users (id, display_name, roles, self_reported_starting_level, subject)
		VALUES ($1, 'Resolvable Actor', ARRAY['player'], 3, $2)
	`, resolvableUserID, resolvableSubject); err != nil {
		t.Fatalf("seeding identity_users: %v", err)
	}

	const resolvablePaymentID = "22222222-2222-2222-2222-222222222222"
	const orphanPaymentID = "33333333-3333-3333-3333-333333333333"
	if _, err := pool.Exec(ctx, `
		INSERT INTO payments (id, payable_type, payable_id, amount_cents, currency_code, method, status, recorded_by_user_id)
		VALUES
			($1, 'booking', '44444444-4444-4444-4444-444444444444', 2500, 'USD', 'offline', 'paid', $2),
			($3, 'booking', '55555555-5555-5555-5555-555555555555', 2500, 'USD', 'offline', 'paid', $4)
	`, resolvablePaymentID, resolvableSubject, orphanPaymentID, orphanSubject); err != nil {
		t.Fatalf("seeding payments: %v", err)
	}

	// Step 3: apply ONLY 0024 — the migration under test.
	applyMigrationFile(t, ctx, pool, "0024_payments_recorded_by_user_id_uuid.sql")

	// Step 4: read the raw column back (not through Repository/app.Service —
	// this test is about the migration's SQL, not the Go adapter above it)
	// and check both rows by exact value.
	var resolvedCol pgtype.UUID
	if err := pool.QueryRow(ctx,
		`SELECT recorded_by_user_id FROM payments WHERE id = $1`, resolvablePaymentID,
	).Scan(&resolvedCol); err != nil {
		t.Fatalf("reading back resolvable row: %v", err)
	}
	if !resolvedCol.Valid {
		t.Fatalf("resolvable row: recorded_by_user_id is NULL, want %s — the backfill "+
			"failed to resolve a subject that WAS registered in identity_users", resolvableUserID)
	}
	if got := resolvedCol.String(); got != resolvableUserID {
		t.Fatalf("resolvable row: recorded_by_user_id = %s, want %s — backfilled to the "+
			"WRONG identity_users.id, not merely a non-null one", got, resolvableUserID)
	}

	var orphanCol pgtype.UUID
	if err := pool.QueryRow(ctx,
		`SELECT recorded_by_user_id FROM payments WHERE id = $1`, orphanPaymentID,
	).Scan(&orphanCol); err != nil {
		t.Fatalf("reading back orphan row: %v", err)
	}
	if orphanCol.Valid {
		t.Fatalf("orphan row: recorded_by_user_id = %s, want NULL — ADR-0017 Decision 3 "+
			"requires an unresolvable subject to backfill to NULL, not a fabricated or "+
			"default uuid", orphanCol.String())
	}

	// Step 5: prove the orphan row was not silently DROPPED instead of
	// nulled — a migration that DELETEd unresolvable rows would also leave
	// orphanCol looking like "no error, no row", which Scan above would have
	// reported as pgx.ErrNoRows, not a passing test. This is the row-count
	// half of the mutation-check: both rows the test seeded must still exist.
	var rowCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM payments WHERE id IN ($1, $2)`,
		resolvablePaymentID, orphanPaymentID,
	).Scan(&rowCount); err != nil {
		t.Fatalf("counting rows: %v", err)
	}
	if rowCount != 2 {
		t.Fatalf("payments row count for the two seeded rows = %d, want 2 — the orphan (or "+
			"the resolvable row) was dropped by the migration rather than backfilled", rowCount)
	}

	// Step 6: the column-type assertion — this migration's whole point is
	// that recorded_by_user_id is now uuid, not text. A future edit that
	// reverted the ALTER TABLE...RENAME COLUMN step but left the backfill
	// UPDATE in place would still pass every check above (a text column can
	// hold a uuid-shaped string) without actually delivering ADR-0017's
	// ruling, so the type itself is checked directly.
	var dataType string
	if err := pool.QueryRow(ctx, `
		SELECT data_type FROM information_schema.columns
		WHERE table_name = 'payments' AND column_name = 'recorded_by_user_id'
	`).Scan(&dataType); err != nil {
		t.Fatalf("reading column type: %v", err)
	}
	if dataType != "uuid" {
		t.Fatalf("payments.recorded_by_user_id data_type = %q, want %q", dataType, "uuid")
	}
}

// waitForReady is smoke_integration_test.go's package-level helper — reused
// here rather than redeclared, since both files build under the same
// `integration` tag into the same postgres_test package and a second
// definition would be a duplicate-symbol compile error.

// migrationFiles returns db/migrations/*.sql in filename order.
func migrationFiles(t *testing.T) (dir string, files []string) {
	t.Helper()
	dir = filepath.Join("..", "..", "..", "..", "db", "migrations")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read migrations dir: %v", err)
	}
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".sql" {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)
	return dir, files
}

// applyMigrationsBefore applies every migration whose filename sorts before
// stopBefore, in order — the schema as it existed immediately prior to the
// named migration landing. Fails the test if stopBefore does not appear in
// the directory at all, since that would silently apply every migration
// (including the one under test) and defeat the "seed against the OLD
// shape" step this test depends on.
func applyMigrationsBefore(t *testing.T, ctx context.Context, pool *pgxpool.Pool, stopBefore string) {
	t.Helper()
	dir, files := migrationFiles(t)

	found := false
	for _, name := range files {
		if name == stopBefore {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("stopBefore migration %q not found in %s — cannot verify pre-migration schema", stopBefore, dir)
	}

	for _, name := range files {
		if name == stopBefore {
			break
		}
		execMigrationFile(t, ctx, pool, dir, name)
	}
}

// applyMigrationFile applies exactly one named migration.
func applyMigrationFile(t *testing.T, ctx context.Context, pool *pgxpool.Pool, name string) {
	t.Helper()
	dir, files := migrationFiles(t)
	for _, f := range files {
		if f == name {
			execMigrationFile(t, ctx, pool, dir, name)
			return
		}
	}
	t.Fatalf("migration %q not found in %s", name, dir)
}

func execMigrationFile(t *testing.T, ctx context.Context, pool *pgxpool.Pool, dir, name string) {
	t.Helper()
	sqlBytes, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("failed to read migration %s: %v", name, err)
	}
	if _, err := pool.Exec(ctx, string(sqlBytes)); err != nil {
		t.Fatalf("failed to apply migration %s: %v", name, err)
	}
}
