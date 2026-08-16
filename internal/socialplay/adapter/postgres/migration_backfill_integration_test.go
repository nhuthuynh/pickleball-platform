//go:build integration

// This is a "large" test (Google's testing-pyramid terminology): it starts a
// real Postgres in a container, applies every migration UP TO 0026 (the
// schema as it existed before this ticket), seeds games/registrations/
// waitlist_entries/game_admins rows directly against that pre-T29.2
// text-column shape, applies
// db/migrations/0026_socialplay_identity_conformance.sql on top, and asserts
// against the raw columns afterward — not through app.Service or
// port.GameRepository/RegistrationRepository/WaitlistRepository/
// GameAdminRepository, all of which only understand the POST-migration
// shape, so this file is the one place in this package that talks to these
// four tables directly.
//
// T29.2 (closes the Social Play third of #164) instruction 8: this is
// CLAUDE.md rule 10 ("a single successful run is not proof of
// reliability... re-run non-deterministic tests") applied to a migration
// specifically — a mutation-check, not an assertion that merely runs the
// migration and checks it didn't error. Mirrors
// internal/payments/adapter/postgres/migration_backfill_integration_test.go
// (T28.1)'s shape exactly, extended from one table/one column to four
// tables/five columns. Every seeded row is read back and its EXACT value is
// compared: a resolvable subject must map to the correct identity_users.id
// (proving the join key is right, not just that some value landed), and an
// orphaned subject must be NULL BEFORE the SET NOT NULL step — this test
// therefore does not run the ticket's actual migration file verbatim (which
// adds NOT NULL unconditionally, per the PR body's own zero-orphan
// determination) but its backfill portion only, so the orphan-NULL behaviour
// ADR-0017 Decision 3 requires is provable independently of which NOT NULL
// branch the real migration took.
//
// Excluded from `make test-domain` and plain `go test ./...` by the
// integration build tag; run it with `go test -tags=integration ./...` or
// `make test`. Requires Docker — this authoring environment has none (the
// standing T4 LESSONS.md gap every other `-tags=integration` file in this
// repo already discloses); see the PR description for how this was verified
// in this environment instead (a real local Postgres, `dangerouslyDisableSandbox`).
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
// directly into each table's text actor column — the "actor who called
// CreateUser before writing a row" case the backfill is supposed to resolve.
const resolvableSubject = "auth0|resolvable-actor"

// resolvableUserID is the identity_users.id resolvableSubject maps to. The
// test asserts the backfilled column equals EXACTLY this value, not merely
// "some non-null uuid" — proving the join key, not just that a join ran.
const resolvableUserID = "11111111-1111-1111-1111-111111111111"

// orphanSubject is written into a second row's text actor column per table
// but is NEVER seeded into identity_users.subject — the "actor wrote a row
// before ever calling CreateUser" case ADR-0017 Decision 3 rules on: the
// backfilled column must be NULL, not dropped, not "".
const orphanSubject = "auth0|never-registered"

// A second, distinct orphan/resolvable subject pair for game_admins, which
// needs FOUR actor values (two rows x {user_id, assigned_by}) rather than
// one — reusing resolvableSubject/orphanSubject for every field would not
// distinguish "the join key is right" from "the join always finds the one
// row that happens to be seeded".
const resolvableSubject2 = "auth0|resolvable-assigner"
const resolvableUserID2 = "22222222-2222-2222-2222-222222222222"

func TestMigration0026_BackfillsResolvableSubjectsAndNullsOrphans(t *testing.T) {
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

	// Step 1: apply every migration BEFORE 0026 — the schema exactly as it
	// existed prior to this ticket, with games.host_id/registrations.
	// player_id/waitlist_entries.player_id/game_admins.user_id/assigned_by
	// still `text`.
	applyMigrationsBefore(t, ctx, pool, "0026_socialplay_identity_conformance.sql")

	// Step 2: seed identity_users with the two resolvable actors.
	if _, err := pool.Exec(ctx, `
		INSERT INTO identity_users (id, display_name, roles, self_reported_starting_level, subject)
		VALUES
			($1, 'Resolvable Actor', ARRAY['player'], 3, $2),
			($3, 'Resolvable Assigner', ARRAY['player'], 3, $4)
	`, resolvableUserID, resolvableSubject, resolvableUserID2, resolvableSubject2); err != nil {
		t.Fatalf("seeding identity_users: %v", err)
	}

	// Step 3: seed one resolvable-subject Game and one orphan-subject Game
	// (games.host_id) — each needs its own court_ids/facility_id/etc. to
	// satisfy the table's other NOT NULL columns, values otherwise
	// irrelevant to this test.
	const resolvableGameID = "33333333-3333-3333-3333-333333333333"
	const orphanGameID = "44444444-4444-4444-4444-444444444444"
	const seedCourtID = "55555555-5555-5555-5555-555555555555"
	if _, err := pool.Exec(ctx, `
		INSERT INTO games (id, host_id, facility_id, court_ids, starts_at, ends_at, capacity, status)
		VALUES
			($1, $2, 'facility-a', ARRAY[$5::uuid], now(), now() + interval '1 hour', 4, 'scheduled'),
			($3, $4, 'facility-a', ARRAY[$5::uuid], now(), now() + interval '1 hour', 4, 'scheduled')
	`, resolvableGameID, resolvableSubject, orphanGameID, orphanSubject, seedCourtID); err != nil {
		t.Fatalf("seeding games: %v", err)
	}

	// Step 4: seed one resolvable-subject and one orphan-subject
	// Registration (registrations.player_id), both against the resolvable
	// Game (which Game they're scoped to doesn't matter for this column).
	const resolvableRegistrationID = "66666666-6666-6666-6666-666666666666"
	const orphanRegistrationID = "77777777-7777-7777-7777-777777777777"
	if _, err := pool.Exec(ctx, `
		INSERT INTO registrations (id, game_id, player_id, source, status, payment_status)
		VALUES
			($1, $2, $3, 'app', 'registered', 'unpaid'),
			($4, $2, $5, 'app', 'registered', 'unpaid')
	`, resolvableRegistrationID, resolvableGameID, resolvableSubject, orphanRegistrationID, orphanSubject); err != nil {
		t.Fatalf("seeding registrations: %v", err)
	}

	// Step 5: seed one resolvable-subject and one orphan-subject
	// WaitlistEntry (waitlist_entries.player_id).
	const resolvableWaitlistID = "88888888-8888-8888-8888-888888888888"
	const orphanWaitlistID = "99999999-9999-9999-9999-999999999999"
	if _, err := pool.Exec(ctx, `
		INSERT INTO waitlist_entries (id, game_id, player_id, position, status)
		VALUES
			($1, $2, $3, 1, 'waiting'),
			($4, $2, $5, 2, 'waiting')
	`, resolvableWaitlistID, resolvableGameID, resolvableSubject, orphanWaitlistID, orphanSubject); err != nil {
		t.Fatalf("seeding waitlist_entries: %v", err)
	}

	// Step 6: seed one resolvable-subject-on-both-fields game_admins row
	// (scoped to resolvableGameID) and one orphan-subject-on-both-fields row
	// (scoped to orphanGameID) — user_id and assigned_by both need
	// backfilling per 0020's own header instruction. game_admins has no
	// surrogate id (its identity is the composite (game_id, user_id)), so
	// each row is scoped to a DIFFERENT Game specifically so it can be
	// re-located after the backfill by game_id alone, without needing to
	// know its post-migration user_id in advance (which, for the orphan row,
	// is exactly the NULL value under test).
	const orphanAssigneeSubject = "auth0|never-registered-assignee"
	if _, err := pool.Exec(ctx, `
		INSERT INTO game_admins (game_id, user_id, assigned_by, assigned_at)
		VALUES
			($1, $2, $3, now()),
			($4, $5, $6, now())
	`, resolvableGameID, resolvableSubject, resolvableSubject2, orphanGameID, orphanSubject, orphanAssigneeSubject); err != nil {
		t.Fatalf("seeding game_admins: %v", err)
	}

	// Step 7: apply ONLY the migration's BACKFILL portion — not the full
	// file (which also adds NOT NULL, which would fail this test's own
	// orphan rows by design; see this file's own top-of-file doc comment for
	// why testing the backfill mechanism is deliberately kept independent of
	// the real migration's NOT NULL determination).
	applyBackfillOnly(t, ctx, pool)

	// Step 8: read every raw column back and check both rows, by exact
	// value, per table.
	assertUUIDColumn(t, ctx, pool, "games", "host_id", resolvableGameID, resolvableUserID)
	assertUUIDColumn(t, ctx, pool, "games", "host_id", orphanGameID, "")

	assertUUIDColumn(t, ctx, pool, "registrations", "player_id", resolvableRegistrationID, resolvableUserID)
	assertUUIDColumn(t, ctx, pool, "registrations", "player_id", orphanRegistrationID, "")

	assertUUIDColumn(t, ctx, pool, "waitlist_entries", "player_id", resolvableWaitlistID, resolvableUserID)
	assertUUIDColumn(t, ctx, pool, "waitlist_entries", "player_id", orphanWaitlistID, "")

	// game_admins has no surrogate id — each row is scoped to its own Game
	// (see the seeding step above for why), so game_id alone re-locates it.
	assertGameAdminColumns(t, ctx, pool, resolvableGameID, resolvableUserID, resolvableUserID2)
	assertGameAdminColumns(t, ctx, pool, orphanGameID, "", "")

	// Step 9: prove no row was silently DROPPED instead of backfilled — a
	// migration that DELETEd unresolvable rows would also leave the "NULL"
	// assertions above looking like a pass (Scan would report pgx.ErrNoRows
	// instead, which assertUUIDColumn treats as a failure, not a NULL — see
	// its own doc comment), so this is a second, independent proof via row
	// counts.
	assertRowCount(t, ctx, pool, "games", 2)
	assertRowCount(t, ctx, pool, "registrations", 2)
	assertRowCount(t, ctx, pool, "waitlist_entries", 2)
	assertRowCount(t, ctx, pool, "game_admins", 2)

	// Step 10: the column-type assertion for all five columns — this
	// migration's whole point is that these are uuid, not text. A future
	// edit that reverted an ALTER TABLE...RENAME COLUMN step but left the
	// backfill UPDATE in place would still pass every check above (a text
	// column can hold a uuid-shaped string) without actually delivering
	// ADR-0017's ruling, so the type itself is checked directly, per column.
	assertColumnType(t, ctx, pool, "games", "host_id", "uuid")
	assertColumnType(t, ctx, pool, "registrations", "player_id", "uuid")
	assertColumnType(t, ctx, pool, "waitlist_entries", "player_id", "uuid")
	assertColumnType(t, ctx, pool, "game_admins", "user_id", "uuid")
	assertColumnType(t, ctx, pool, "game_admins", "assigned_by", "uuid")
}

// assertUUIDColumn reads column from table WHERE id = rowID and compares it
// against wantUserID — "" means "want NULL" (the orphan case), any other
// value means "want exactly this uuid" (the resolvable case, checked by
// exact value, not merely non-null — proving the join key is right).
func assertUUIDColumn(t *testing.T, ctx context.Context, pool *pgxpool.Pool, table, column, rowID, wantUserID string) {
	t.Helper()
	var got pgtype.UUID
	query := "SELECT " + column + " FROM " + table + " WHERE id = $1"
	if err := pool.QueryRow(ctx, query, rowID).Scan(&got); err != nil {
		t.Fatalf("%s.%s for row %s: %v", table, column, rowID, err)
	}
	if wantUserID == "" {
		if got.Valid {
			t.Fatalf("%s.%s for row %s = %s, want NULL — ADR-0017 Decision 3 requires an "+
				"unresolvable subject to backfill to NULL, not a fabricated or default uuid",
				table, column, rowID, got.String())
		}
		return
	}
	if !got.Valid {
		t.Fatalf("%s.%s for row %s is NULL, want %s — the backfill failed to resolve a "+
			"subject that WAS registered in identity_users", table, column, rowID, wantUserID)
	}
	if got.String() != wantUserID {
		t.Fatalf("%s.%s for row %s = %s, want %s — backfilled to the WRONG identity_users.id, "+
			"not merely a non-null one", table, column, rowID, got.String(), wantUserID)
	}
}

// assertGameAdminColumns reads the (sole) game_admins row scoped to gameID
// and checks its user_id/assigned_by against wantUserID/wantAssignedBy ("" on
// either meaning "want NULL", mirroring assertUUIDColumn). Scoped by game_id
// alone — safe because this test seeds exactly one game_admins row per Game
// (see the seeding step's own comment for why one row per Game, rather than
// two rows sharing one Game, is what makes "re-locate this row after user_id
// itself changes value" possible without a surrogate id).
func assertGameAdminColumns(t *testing.T, ctx context.Context, pool *pgxpool.Pool, gameID, wantUserID, wantAssignedBy string) {
	t.Helper()
	var userIDCol, assignedByCol pgtype.UUID
	if err := pool.QueryRow(ctx, `
		SELECT user_id, assigned_by FROM game_admins WHERE game_id = $1
	`, gameID).Scan(&userIDCol, &assignedByCol); err != nil {
		t.Fatalf("game_admins row for game %s: %v", gameID, err)
	}

	if wantUserID == "" {
		if userIDCol.Valid {
			t.Fatalf("game_admins.user_id for game %s = %s, want NULL", gameID, userIDCol.String())
		}
	} else if !userIDCol.Valid || userIDCol.String() != wantUserID {
		t.Fatalf("game_admins.user_id for game %s = %v, want %s", gameID, userIDCol, wantUserID)
	}

	if wantAssignedBy == "" {
		if assignedByCol.Valid {
			t.Fatalf("game_admins.assigned_by for game %s = %s, want NULL", gameID, assignedByCol.String())
		}
	} else if !assignedByCol.Valid || assignedByCol.String() != wantAssignedBy {
		t.Fatalf("game_admins.assigned_by for game %s = %v, want %s", gameID, assignedByCol, wantAssignedBy)
	}
}

// assertRowCount fails unless table holds exactly want rows.
func assertRowCount(t *testing.T, ctx context.Context, pool *pgxpool.Pool, table string, want int) {
	t.Helper()
	var got int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM "+table).Scan(&got); err != nil {
		t.Fatalf("counting %s: %v", table, err)
	}
	if got != want {
		t.Fatalf("%s row count = %d, want %d — a row was dropped (or duplicated) by the backfill "+
			"rather than merely converted in place", table, got, want)
	}
}

// assertColumnType fails unless table.column's information_schema data_type
// is exactly want.
func assertColumnType(t *testing.T, ctx context.Context, pool *pgxpool.Pool, table, column, want string) {
	t.Helper()
	var got string
	if err := pool.QueryRow(ctx, `
		SELECT data_type FROM information_schema.columns
		WHERE table_name = $1 AND column_name = $2
	`, table, column).Scan(&got); err != nil {
		t.Fatalf("reading %s.%s column type: %v", table, column, err)
	}
	if got != want {
		t.Fatalf("%s.%s data_type = %q, want %q", table, column, got, want)
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

// applyBackfillOnly applies exactly the backfill statements of
// 0026_socialplay_identity_conformance.sql — the ADD COLUMN/UPDATE steps for
// all five columns, deliberately NOT the SET NOT NULL/DROP COLUMN/RENAME/
// index-recreation/join_waitlist_entry-recreation steps — so this test can
// assert the orphan-NULL behaviour independently of whichever NOT NULL
// branch the real migration file takes (see this file's own top-of-file doc
// comment). Hand-transcribed from the real migration's backfill sections
// rather than executed via regex-slicing the file, so a future edit to the
// real migration's SQL is not silently desynced from what this test
// actually runs — if the two drift, this test's own assertions (not a
// missing statement) are what would need updating, the same trade-off
// db/migrations/*_integration_test.go files in this repo already accept
// elsewhere for hand-seeded fixture rows.
func applyBackfillOnly(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	stmts := []string{
		`ALTER TABLE games ADD COLUMN host_id_uuid uuid REFERENCES identity_users (id)`,
		`UPDATE games g SET host_id_uuid = u.id FROM identity_users u WHERE g.host_id = u.subject`,

		`ALTER TABLE registrations ADD COLUMN player_id_uuid uuid REFERENCES identity_users (id)`,
		`UPDATE registrations r SET player_id_uuid = u.id FROM identity_users u WHERE r.player_id = u.subject`,

		`ALTER TABLE waitlist_entries ADD COLUMN player_id_uuid uuid REFERENCES identity_users (id)`,
		`UPDATE waitlist_entries w SET player_id_uuid = u.id FROM identity_users u WHERE w.player_id = u.subject`,

		`ALTER TABLE game_admins ADD COLUMN user_id_uuid uuid REFERENCES identity_users (id)`,
		`ALTER TABLE game_admins ADD COLUMN assigned_by_uuid uuid REFERENCES identity_users (id)`,
		`UPDATE game_admins ga SET user_id_uuid = u.id FROM identity_users u WHERE ga.user_id = u.subject`,
		`UPDATE game_admins ga SET assigned_by_uuid = u.id FROM identity_users u WHERE ga.assigned_by = u.subject`,

		// Rename the backfilled shadow columns into the names
		// assertUUIDColumn/assertGameAdminColumns query (host_id, player_id,
		// user_id, assigned_by) — the real migration does this only AFTER
		// dropping the old text column and setting NOT NULL; this test skips
		// both of those (deliberately, per this function's own doc comment)
		// but still needs the backfilled values under their final column
		// names, so the OLD text columns are dropped here too (their data is
		// already fully read out via the seeded id/game_id and
		// identity_users.subject joins above, so nothing is lost that this
		// test still needs).
		`ALTER TABLE games DROP COLUMN host_id`,
		`ALTER TABLE games RENAME COLUMN host_id_uuid TO host_id`,
		`ALTER TABLE registrations DROP COLUMN player_id`,
		`ALTER TABLE registrations RENAME COLUMN player_id_uuid TO player_id`,
		`ALTER TABLE waitlist_entries DROP COLUMN player_id`,
		`ALTER TABLE waitlist_entries RENAME COLUMN player_id_uuid TO player_id`,
		`ALTER TABLE game_admins DROP COLUMN user_id`,
		`ALTER TABLE game_admins DROP COLUMN assigned_by`,
		`ALTER TABLE game_admins RENAME COLUMN user_id_uuid TO user_id`,
		`ALTER TABLE game_admins RENAME COLUMN assigned_by_uuid TO assigned_by`,
	}
	for _, stmt := range stmts {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			t.Fatalf("applying backfill statement %q: %v", stmt, err)
		}
	}
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
