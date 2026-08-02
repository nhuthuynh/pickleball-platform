//go:build integration

// This is a "large" test (Google's testing-pyramid terminology): it starts a
// real Postgres in a container and proves the
// registrations_active_player_per_game_idx unique constraint — not the
// domain-level pre-check — is what actually stops a double registration
// under real concurrency, the same relationship T4 proved for bookings'
// EXCLUDE constraint. Excluded from `make test-domain` and `go test ./...`
// by the integration build tag; run it with `go test -tags=integration
// ./...` or `make test`. Requires Docker.
//
// Manually verified in this environment (no Docker daemon, so
// testcontainers itself couldn't run here — same gap T4 documented) via a
// throwaway program that called RegistrationRepository.Create directly
// against a real local Postgres 16 instance, bypassing the domain-level
// pre-check entirely: the second same-player insert was rejected by the DB
// constraint and correctly translated to domain.ErrAlreadyRegistered, not
// leaked as a raw pgconn.PgError. See the T5.4 PR description for the exact
// output. This file is the portable, CI-runnable, concurrency version of
// that same proof.
package postgres_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	socialplaypg "github.com/nhuthuynh/white-label/internal/socialplay/adapter/postgres"
	"github.com/nhuthuynh/white-label/internal/socialplay/domain"
)

// concurrentRegisterAttempts is deliberately > a handful: the unique
// constraint must hold regardless of how many requests race, not just for
// the simplest two-request case, mirroring T4's booking concurrency test.
const concurrentRegisterAttempts = 20

func TestCreateRegistration_ExactlyOneWinsUnderConcurrency(t *testing.T) {
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
	applyMigrations(t, ctx, pool)

	gameRepo := socialplaypg.NewGameRepository(pool)
	regRepo := socialplaypg.NewRegistrationRepository(pool)

	r := mustRange(t, "2026-09-01T09:00:00Z", "2026-09-01T10:00:00Z")
	game, err := domain.NewGame("11111111-1111-1111-1111-100000000001", "host-x", "facility-x", []string{seedCourtID}, r, concurrentRegisterAttempts)
	if err != nil {
		t.Fatalf("bad fixture game: %v", err)
	}
	game, err = gameRepo.Create(ctx, game)
	if err != nil {
		t.Fatalf("failed to create fixture game: %v", err)
	}

	// Every goroutine registers the SAME player for the SAME game —
	// bypassing domain.Register's own pre-check entirely by calling the
	// repository directly, so only the DB constraint can be what stops the
	// duplicates. This deliberately does not go through app.Service, the
	// same way T4's booking test calls the repo layer where the invariant
	// actually lives under concurrency.
	var wg sync.WaitGroup
	var mu sync.Mutex
	successes, conflicts, unexpected := 0, 0, 0

	for i := 0; i < concurrentRegisterAttempts; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			reg := domain.Registration{
				ID:            uuidLike(n),
				GameID:        game.ID,
				PlayerID:      "dup-player",
				Source:        domain.RegistrationSourceApp,
				Status:        domain.RegistrationStatusRegistered,
				PaymentStatus: domain.PaymentStatusUnpaid,
			}
			_, err := regRepo.Create(ctx, reg)
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				successes++
			case errors.Is(err, domain.ErrAlreadyRegistered):
				conflicts++
			default:
				unexpected++
				t.Logf("unexpected error: %v", err)
			}
		}(i)
	}
	wg.Wait()

	if successes != 1 {
		t.Errorf("successes = %d, want exactly 1", successes)
	}
	if conflicts != concurrentRegisterAttempts-1 {
		t.Errorf("conflicts = %d, want %d", conflicts, concurrentRegisterAttempts-1)
	}
	if unexpected != 0 {
		t.Errorf("unexpected errors = %d, want 0", unexpected)
	}
}

const seedCourtID = "11111111-1111-1111-1111-111111111111"

// uuidLike deterministically builds a distinct valid UUID string per
// goroutine index, avoiding a real UUID dependency in this test file.
func uuidLike(n int) string {
	return fmt.Sprintf("00000000-0000-0000-0000-%012d", n)
}

func waitForReady(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
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

// applyMigrations runs db/migrations/*.sql in filename order, mirroring what
// docker-compose's initdb.d does for local dev (see CLAUDE.md gotchas) —
// this keeps the integration test honest against the same schema a real
// deploy would use, rather than a hand-rolled test schema that could drift.
func applyMigrations(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
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

func mustRange(t *testing.T, start, end string) domain.TimeRange {
	t.Helper()
	s, err := time.Parse(time.RFC3339, start)
	if err != nil {
		t.Fatalf("bad fixture time: %v", err)
	}
	e, err := time.Parse(time.RFC3339, end)
	if err != nil {
		t.Fatalf("bad fixture time: %v", err)
	}
	r, err := domain.NewTimeRange(s, e)
	if err != nil {
		t.Fatalf("bad fixture range: %v", err)
	}
	return r
}
