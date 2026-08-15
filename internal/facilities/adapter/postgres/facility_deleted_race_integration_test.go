//go:build integration

// T17.4 (issue #195). This is the half of the regression proof that NO
// in-memory fixture can supply, and saying why is the point of this file
// existing separately from foreign_key_test.go beside it — mirroring
// internal/booking/adapter/postgres/unknown_court_integration_test.go's
// identical split for T15.6/#185.
//
// The defect: AddCourt and AddCameraLink are both reached only after
// app.Service's own GetFacilityByID guard already confirmed the target
// Facility exists (internal/facilities/app/service.go). In the non-racing
// case that guard's own domain.ErrFacilityNotFound wins and the FK never
// fires. Under a concurrent delete of the Facility landing between that
// guard's read and this package's INSERT, courts.facility_id or
// facility_camera_links.facility_id's "REFERENCES facilities (id)"
// (db/migrations/0010_facilities.sql) instead raises a raw Postgres 23503
// that — before this ticket — translateErr had no arm for, so it fell
// through to the wrapped default and would have answered codes.Internal: a
// 500 for what is, at the moment it happens, the legitimate client-visible
// fact "no such Facility". Issue #195's own table names this exact pair as
// "guarded by a read, not a translation" — narrower than #185's window, not
// absent.
//
// This file reproduces that race window deterministically rather than with
// real concurrent goroutines: create the Facility, let a guarding read
// against it succeed (proving the "guard passed" half is genuine, not
// assumed), delete the Facility, THEN issue the write this package's own
// Repository.AddCourt/AddCameraLink perform. A goroutine-based race against
// the same single-process Postgres connection could land on either side of
// the window nondeterministically and would make this test flaky for no
// added proof; sequencing the delete between the confirmed-successful read
// and the write instead *guarantees* the exact interleaving #195 describes,
// which is the scenario under test, not "sometimes reproduce it".
//
// The unit tests in foreign_key_test.go pin translateErr's 23503 -> sentinel
// mapping with a synthetic *pgconn.PgError, and
// adapter/grpcapi/foreign_key_error_mapping_test.go pins sentinel ->
// codes.NotFound with a fake — but BOTH assume Postgres raises 23503 here at
// all. This file is where that assumption is checked against a real
// database rather than asserted.
//
// Excluded from `make test-domain`, `make test-adapters` and plain
// `go test ./...` by the integration build tag; compiled (not run) by `make
// vet-integration`, which `make ci` runs; executed by `make ci-integration`
// or `make test`. Requires Docker.
package postgres_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	facilitiespg "github.com/nhuthuynh/white-label/internal/facilities/adapter/postgres"
	"github.com/nhuthuynh/white-label/internal/facilities/domain"
)

// startFacilitiesPostgres brings up a real Postgres with the repo's own
// migrations applied — the same schema a real deploy gets, not a hand-rolled
// test schema that could drift. Mirrors
// internal/booking/adapter/postgres/concurrency_integration_test.go's
// startPostgres-equivalent helpers; duplicated locally (rather than shared)
// because they are unexported and this is a different package.
func startFacilitiesPostgres(t *testing.T, ctx context.Context) *pgxpool.Pool {
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

	waitForFacilitiesPostgresReady(t, ctx, pool)
	applyFacilitiesMigrations(t, ctx, pool)
	return pool
}

func waitForFacilitiesPostgresReady(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for {
		if err := pool.Ping(ctx); err == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("postgres did not become ready in time")
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func applyFacilitiesMigrations(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	dir := filepath.Join("..", "..", "..", "..", "db", "migrations")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read migrations dir: %v", err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".sql" {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	for _, name := range files {
		sqlBytes, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("failed to read migration %s: %v", name, err)
		}
		if _, err := pool.Exec(ctx, string(sqlBytes)); err != nil {
			t.Fatalf("failed to apply migration %s: %v", name, err)
		}
	}
}

// seedRaceFacility creates a Facility, confirms a guarding read against it
// succeeds (the "guard passed" half of the race window), then deletes it
// directly with SQL — port.Repository has no delete method, matching every
// other context's own read/write surface — and returns its (now-dangling)
// ID for a subsequent write to reference.
func seedRaceFacility(t *testing.T, ctx context.Context, pool *pgxpool.Pool, repo *facilitiespg.Repository, id string) string {
	t.Helper()

	f, err := domain.NewFacility(id, "00000000-0000-4000-b000-000000000001",
		"Race Window Courts", "", "1 Test Way", nil)
	if err != nil {
		t.Fatalf("failed to build fixture facility: %v", err)
	}
	if _, err := repo.CreateFacility(ctx, f); err != nil {
		t.Fatalf("failed to seed fixture facility: %v", err)
	}

	// The guarding read app.Service.AddCourt/AddCameraLink call before their
	// own write — proving it genuinely succeeds while the Facility exists,
	// not merely asserting that it would.
	if _, err := repo.GetFacilityByID(ctx, id); err != nil {
		t.Fatalf("guarding read GetFacilityByID(%s) failed before the race window even opened: %v", id, err)
	}

	// The concurrent delete landing in the window: by the time the write
	// below reaches Postgres, the row the guard just confirmed is gone.
	if _, err := pool.Exec(ctx, "DELETE FROM facilities WHERE id = $1", id); err != nil {
		t.Fatalf("failed to delete fixture facility to open the race window: %v", err)
	}
	return id
}

// TestAddCourt_FacilityDeletedInRaceWindowAgainstRealPostgres is the
// load-bearing claim for courts.facility_id: against a REAL database, a
// Court naming a Facility that existed moments ago but was deleted before
// this write reached Postgres produces a 23503 foreign-key violation that
// the adapter translates into domain.ErrFacilityNotFound — not a raw,
// wrapped infra error that would answer codes.Internal.
func TestAddCourt_FacilityDeletedInRaceWindowAgainstRealPostgres(t *testing.T) {
	ctx := context.Background()
	pool := startFacilitiesPostgres(t, ctx)
	repo := facilitiespg.NewRepository(pool)

	deletedFacilityID := seedRaceFacility(t, ctx, pool, repo,
		"00000000-0000-4000-a000-000000000001")

	_, err := repo.AddCourt(ctx, domain.Court{
		ID:         "00000000-0000-4000-a000-000000000011",
		FacilityID: deletedFacilityID,
		Name:       "Court 1",
	})
	if !errors.Is(err, domain.ErrFacilityNotFound) {
		t.Fatalf("AddCourt(deleted facility) = %v, want domain.ErrFacilityNotFound", err)
	}
}

// TestAddCourt_KnownFacilityStillSucceedsAgainstRealPostgres is the control.
// Without it, an adapter that rejected every INSERT would satisfy the test
// above.
func TestAddCourt_KnownFacilityStillSucceedsAgainstRealPostgres(t *testing.T) {
	ctx := context.Background()
	pool := startFacilitiesPostgres(t, ctx)
	repo := facilitiespg.NewRepository(pool)

	f, err := domain.NewFacility("00000000-0000-4000-a000-000000000002",
		"00000000-0000-4000-b000-000000000001", "Control Courts", "", "2 Test Way", nil)
	if err != nil {
		t.Fatalf("failed to build fixture facility: %v", err)
	}
	if _, err := repo.CreateFacility(ctx, f); err != nil {
		t.Fatalf("failed to seed fixture facility: %v", err)
	}

	c, err := repo.AddCourt(ctx, domain.Court{
		ID:         "00000000-0000-4000-a000-000000000012",
		FacilityID: f.ID,
		Name:       "Court 1",
	})
	if err != nil {
		t.Fatalf("AddCourt(existing facility) = %v, want success", err)
	}
	if c.FacilityID != f.ID {
		t.Fatalf("persisted court facility = %q, want %q", c.FacilityID, f.ID)
	}
}

// TestAddCameraLink_FacilityDeletedInRaceWindowAgainstRealPostgres mirrors
// TestAddCourt_FacilityDeletedInRaceWindowAgainstRealPostgres for
// facility_camera_links.facility_id, the second of this ticket's three FK
// paths.
func TestAddCameraLink_FacilityDeletedInRaceWindowAgainstRealPostgres(t *testing.T) {
	ctx := context.Background()
	pool := startFacilitiesPostgres(t, ctx)
	repo := facilitiespg.NewRepository(pool)

	deletedFacilityID := seedRaceFacility(t, ctx, pool, repo,
		"00000000-0000-4000-a000-000000000003")

	_, err := repo.AddCameraLink(ctx, deletedFacilityID, domain.CameraLink{
		URL: "https://example.com/cam1.m3u8",
	})
	if !errors.Is(err, domain.ErrFacilityNotFound) {
		t.Fatalf("AddCameraLink(deleted facility) = %v, want domain.ErrFacilityNotFound", err)
	}
}

// TestAddCameraLink_KnownFacilityStillSucceedsAgainstRealPostgres is
// AddCameraLink's control, mirroring AddCourt's above.
func TestAddCameraLink_KnownFacilityStillSucceedsAgainstRealPostgres(t *testing.T) {
	ctx := context.Background()
	pool := startFacilitiesPostgres(t, ctx)
	repo := facilitiespg.NewRepository(pool)

	f, err := domain.NewFacility("00000000-0000-4000-a000-000000000004",
		"00000000-0000-4000-b000-000000000001", "Control Courts 2", "", "4 Test Way", nil)
	if err != nil {
		t.Fatalf("failed to build fixture facility: %v", err)
	}
	if _, err := repo.CreateFacility(ctx, f); err != nil {
		t.Fatalf("failed to seed fixture facility: %v", err)
	}

	link, err := repo.AddCameraLink(ctx, f.ID, domain.CameraLink{
		URL: "https://example.com/cam2.m3u8",
	})
	if err != nil {
		t.Fatalf("AddCameraLink(existing facility) = %v, want success", err)
	}
	if link.URL != "https://example.com/cam2.m3u8" {
		t.Fatalf("persisted camera link URL = %q, want the seeded URL", link.URL)
	}
}
