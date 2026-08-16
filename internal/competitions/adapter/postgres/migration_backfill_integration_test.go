//go:build integration

// This is a "large" test (Google's testing-pyramid terminology): it starts a
// real Postgres in a container, applies every migration UP TO 0024 (the
// schema as it existed before this ticket), seeds competitions/
// competition_entries/competition_admins rows directly against that
// pre-T29.1 text-column shape, applies db/migrations/
// 0025_competitions_identity_conformance.sql on top, and asserts against the
// raw columns afterward — not through app.Service or port.Repository, both
// of which only understand the POST-migration shape, so this file is the one
// place in this package that talks to these three tables directly. Mirrors
// internal/payments/adapter/postgres/migration_backfill_integration_test.go
// (T28.1) exactly, one context over, extended from one column/one table to
// four columns/three tables.
//
// T29.1 (closes the Competitions third of #164) instruction 8: this is
// CLAUDE.md rule 10 ("a single successful run is not proof of
// reliability... re-run non-deterministic tests") applied to a migration
// specifically — a mutation-check, not an assertion that merely runs the
// migration and checks it didn't error. Every seeded row is read back and
// its EXACT value is compared: a resolvable subject must map to the correct
// identity_users.id (proving the join key is right, not just that some
// value landed), and an orphaned subject must be NULL — specifically NULL,
// checked via the driver's pgtype.UUID.Valid flag, not empty-string, not
// silently dropped (row counts are asserted too, so an orphan being DELETEd
// instead of nulled would also fail this test).
//
// Excluded from `make test-domain` and plain `go test ./...` by the
// integration build tag; run it with `go test -tags=integration ./...` or
// `make test`. Requires Docker — this authoring environment has none (the
// standing T4 LESSONS.md gap every other `-tags=integration` file in this
// repo already discloses); see the PR description for how this was verified
// in this environment instead (a real local Postgres 16 system service, the
// same T4 LESSONS.md fallback every other such file cites).
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

// --- fixtures ---------------------------------------------------------
//
// One resolvable identity per role, one orphaned subject per role — mirrors
// T28.1's resolvableSubject/orphanSubject shape, extended to four roles
// (host, player, admin, assigner) since this migration backfills four
// columns across three tables in one pass.

const (
	resolvableHostSubject   = "auth0|resolvable-host"
	resolvableHostUserID    = "11111111-1111-1111-1111-111111111111"
	resolvablePlayerSubject = "auth0|resolvable-player"
	resolvablePlayerUserID  = "22222222-2222-2222-2222-222222222222"
	resolvableAdminSubject  = "auth0|resolvable-admin"
	resolvableAdminUserID   = "33333333-3333-3333-3333-333333333333"

	// orphanSubject is shared across all four columns — a single "actor who
	// never called CreateUser" fact is sufficient to prove the orphan
	// (Decision 3) branch for every column, since the join logic is
	// identical per column and the mutation-check's point is the JOIN
	// behaviour, not that different orphans exist.
	orphanSubject = "auth0|never-registered"

	competitionResolvableID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	competitionOrphanID     = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	entryResolvableID       = "cccccccc-cccc-cccc-cccc-cccccccccccc"
	entryOrphanID           = "dddddddd-dddd-dddd-dddd-dddddddddddd"
)

func TestMigration0025_BackfillsResolvableSubjectsAndNullsOrphans(t *testing.T) {
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

	// Step 1: apply every migration BEFORE 0025 — the schema exactly as it
	// existed prior to this ticket, with all four columns still `text`.
	applyMigrationsBefore(t, ctx, pool, "0025_competitions_identity_conformance.sql")

	// Step 2: seed identity_users with the three resolvable actors.
	if _, err := pool.Exec(ctx, `
		INSERT INTO identity_users (id, display_name, roles, self_reported_starting_level, subject)
		VALUES
			($1, 'Resolvable Host', ARRAY['host_organiser'], 3, $2),
			($3, 'Resolvable Player', ARRAY['player'], 3, $4),
			($5, 'Resolvable Admin', ARRAY['player'], 3, $6)
	`, resolvableHostUserID, resolvableHostSubject,
		resolvablePlayerUserID, resolvablePlayerSubject,
		resolvableAdminUserID, resolvableAdminSubject); err != nil {
		t.Fatalf("seeding identity_users: %v", err)
	}

	// Step 3: seed competitions (one resolvable host, one orphan host),
	// competition_entries (one resolvable player, one orphan player, both
	// against the RESOLVABLE competition so the FK holds either way), and
	// competition_admins (one resolvable admin/assigner pair, one orphan
	// admin/assigner pair, both against the resolvable competition).
	if _, err := pool.Exec(ctx, `
		INSERT INTO competitions (id, host_id, name, capacity, payment_method, format, entry_fee_cents, entry_fee_currency, share_token)
		VALUES
			($1, $2, 'Resolvable Open', 8, 'either', 'singles', 0, 'USD', 'tok-resolvable-0025'),
			($3, $4, 'Orphan Open', 8, 'either', 'singles', 0, 'USD', 'tok-orphan-0025')
	`, competitionResolvableID, resolvableHostSubject, competitionOrphanID, orphanSubject); err != nil {
		t.Fatalf("seeding competitions: %v", err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO competition_entries (id, competition_id, player_id, guest_count, source, status, payment_status)
		VALUES
			($1, $2, $3, 0, 'app', 'entered', 'unpaid'),
			($4, $2, $5, 0, 'app', 'entered', 'unpaid')
	`, entryResolvableID, competitionResolvableID, resolvablePlayerSubject, entryOrphanID, orphanSubject); err != nil {
		t.Fatalf("seeding competition_entries: %v", err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO competition_admins (competition_id, user_id, assigned_by, assigned_at)
		VALUES
			($1, $2, $3, now()),
			($1, $4, $5, now())
	`, competitionResolvableID, resolvableAdminSubject, resolvableHostSubject,
		orphanSubject, orphanSubject); err != nil {
		t.Fatalf("seeding competition_admins: %v", err)
	}

	// Step 4: apply ONLY 0025 — the migration under test.
	applyMigrationFile(t, ctx, pool, "0025_competitions_identity_conformance.sql")

	// Step 5: read every raw column back and check EXACT values.

	assertResolvedUUID(t, ctx, pool, "competitions", "host_id", "id", competitionResolvableID, resolvableHostUserID)
	assertNullUUID(t, ctx, pool, "competitions", "host_id", "id", competitionOrphanID)

	assertResolvedUUID(t, ctx, pool, "competition_entries", "player_id", "id", entryResolvableID, resolvablePlayerUserID)
	assertNullUUID(t, ctx, pool, "competition_entries", "player_id", "id", entryOrphanID)

	assertResolvedAdminColumn(t, ctx, pool, "user_id", resolvableAdminUserID)
	assertResolvedAdminColumn(t, ctx, pool, "assigned_by", resolvableHostUserID)

	// Step 6: row-count checks — an orphan being DELETEd instead of nulled
	// would leave every check above looking like "no error, no row" via
	// pgx.ErrNoRows, which the helpers below already fail on, but the counts
	// are asserted directly too so the failure message names the real cause.
	assertRowCount(t, ctx, pool, "competitions", []string{competitionResolvableID, competitionOrphanID}, 2)
	assertRowCount(t, ctx, pool, "competition_entries", []string{entryResolvableID, entryOrphanID}, 2)
	var adminRowCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM competition_admins WHERE competition_id = $1`,
		competitionResolvableID).Scan(&adminRowCount); err != nil {
		t.Fatalf("counting competition_admins rows: %v", err)
	}
	if adminRowCount != 2 {
		t.Fatalf("competition_admins row count = %d, want 2 — a row was dropped by the migration rather than backfilled", adminRowCount)
	}

	// Step 7: the column-type assertions — this migration's whole point is
	// that all four columns are now uuid, not text. A future edit that
	// reverted the ALTER TABLE...RENAME COLUMN step but left the backfill
	// UPDATE in place would still pass every check above (a text column can
	// hold a uuid-shaped string) without actually delivering ADR-0017's
	// ruling, so the type itself is checked directly, per column.
	assertColumnType(t, ctx, pool, "competitions", "host_id", "uuid")
	assertColumnType(t, ctx, pool, "competition_entries", "player_id", "uuid")
	assertColumnType(t, ctx, pool, "competition_admins", "user_id", "uuid")
	assertColumnType(t, ctx, pool, "competition_admins", "assigned_by", "uuid")

	// Step 8: nullability — this migration deliberately ships all four
	// columns NULLABLE (see 0025's own header for the full reasoning: the
	// orphan seeded above must leave the migration succeeding, which a
	// NOT NULL constraint recreated in the same pass would have prevented).
	// Asserted directly so a future edit that tightened one of these to
	// NOT NULL without also removing this file's own orphan seeding (which
	// would then make this test itself fail to apply the migration) is
	// caught by an assertion that names the mismatch, not just a migration
	// apply error.
	assertNullable(t, ctx, pool, "competitions", "host_id")
	assertNullable(t, ctx, pool, "competition_entries", "player_id")
	assertNullable(t, ctx, pool, "competition_admins", "user_id")
	assertNullable(t, ctx, pool, "competition_admins", "assigned_by")

	// Step 9: competition_admins' uniqueness guard survived the PK ->
	// UNIQUE-constraint change 0025 makes (see its own header) — a second
	// assignment of the SAME (competition, resolved-admin) pair is still
	// rejected.
	if _, err := pool.Exec(ctx, `
		INSERT INTO competition_admins (competition_id, user_id, assigned_by, assigned_at)
		VALUES ($1, $2, $3, now())
	`, competitionResolvableID, resolvableAdminUserID, resolvableHostUserID); err == nil {
		t.Fatal("a duplicate (competition_id, user_id) assignment was accepted after 0025 — " +
			"the composite PRIMARY KEY's replacement UNIQUE constraint did not carry the guard forward")
	}
}

func assertResolvedUUID(t *testing.T, ctx context.Context, pool *pgxpool.Pool, table, column, idColumn, rowID, wantUserID string) {
	t.Helper()
	var got pgtype.UUID
	if err := pool.QueryRow(ctx,
		"SELECT "+column+" FROM "+table+" WHERE "+idColumn+" = $1", rowID,
	).Scan(&got); err != nil {
		t.Fatalf("reading back %s.%s for %s: %v", table, column, rowID, err)
	}
	if !got.Valid {
		t.Fatalf("%s.%s for %s is NULL, want %s — the backfill failed to resolve a subject that WAS registered in identity_users",
			table, column, rowID, wantUserID)
	}
	if got.String() != wantUserID {
		t.Fatalf("%s.%s for %s = %s, want %s — backfilled to the WRONG identity_users.id, not merely a non-null one",
			table, column, rowID, got.String(), wantUserID)
	}
}

func assertNullUUID(t *testing.T, ctx context.Context, pool *pgxpool.Pool, table, column, idColumn, rowID string) {
	t.Helper()
	var got pgtype.UUID
	if err := pool.QueryRow(ctx,
		"SELECT "+column+" FROM "+table+" WHERE "+idColumn+" = $1", rowID,
	).Scan(&got); err != nil {
		t.Fatalf("reading back %s.%s for %s: %v", table, column, rowID, err)
	}
	if got.Valid {
		t.Fatalf("%s.%s for %s = %s, want NULL — ADR-0017 Decision 3 requires an unresolvable "+
			"subject to backfill to NULL, not a fabricated or default uuid", table, column, rowID, got.String())
	}
}

// assertResolvedAdminColumn reads BOTH competition_admins rows for
// competitionResolvableID and confirms exactly one has the given column
// equal to wantUserID (the resolvable row) — competition_admins has no
// single-row primary key addressable by a simple id column the way
// competitions/competition_entries do (its identity is the composite
// (competition_id, user_id) pair, which is exactly the column under test),
// so this walks both seeded rows rather than querying by id.
func assertResolvedAdminColumn(t *testing.T, ctx context.Context, pool *pgxpool.Pool, column, wantUserID string) {
	t.Helper()
	rows, err := pool.Query(ctx, "SELECT "+column+" FROM competition_admins WHERE competition_id = $1", competitionResolvableID)
	if err != nil {
		t.Fatalf("reading back competition_admins.%s: %v", column, err)
	}
	defer rows.Close()

	found := false
	sawNull := false
	for rows.Next() {
		var got pgtype.UUID
		if err := rows.Scan(&got); err != nil {
			t.Fatalf("scanning competition_admins.%s: %v", column, err)
		}
		if !got.Valid {
			sawNull = true
			continue
		}
		if got.String() == wantUserID {
			found = true
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterating competition_admins.%s: %v", column, err)
	}
	if !found {
		t.Fatalf("no competition_admins row has %s = %s — the backfill failed to resolve a subject that WAS registered in identity_users", column, wantUserID)
	}
	if !sawNull {
		t.Fatalf("no competition_admins row has %s = NULL — the seeded orphan row's column was not left NULL as ADR-0017 Decision 3 requires", column)
	}
}

func assertRowCount(t *testing.T, ctx context.Context, pool *pgxpool.Pool, table string, ids []string, want int) {
	t.Helper()
	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM "+table+" WHERE id = ANY($1)", ids).Scan(&count); err != nil {
		t.Fatalf("counting %s rows: %v", table, err)
	}
	if count != want {
		t.Fatalf("%s row count for the seeded rows = %d, want %d — a row was dropped by the migration rather than backfilled", table, count, want)
	}
}

func assertColumnType(t *testing.T, ctx context.Context, pool *pgxpool.Pool, table, column, want string) {
	t.Helper()
	var dataType string
	if err := pool.QueryRow(ctx, `
		SELECT data_type FROM information_schema.columns
		WHERE table_name = $1 AND column_name = $2
	`, table, column).Scan(&dataType); err != nil {
		t.Fatalf("reading %s.%s column type: %v", table, column, err)
	}
	if dataType != want {
		t.Fatalf("%s.%s data_type = %q, want %q", table, column, dataType, want)
	}
}

func assertNullable(t *testing.T, ctx context.Context, pool *pgxpool.Pool, table, column string) {
	t.Helper()
	var isNullable string
	if err := pool.QueryRow(ctx, `
		SELECT is_nullable FROM information_schema.columns
		WHERE table_name = $1 AND column_name = $2
	`, table, column).Scan(&isNullable); err != nil {
		t.Fatalf("reading %s.%s nullability: %v", table, column, err)
	}
	if isNullable != "YES" {
		t.Fatalf("%s.%s is_nullable = %q, want %q — 0025 ships this column nullable, see its own header for why", table, column, isNullable, "YES")
	}
}

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
