//go:build integration

// This is a "large" test (Google's testing-pyramid terminology): it starts a
// real Postgres in a container and proves the
// registrations_capacity_guard trigger (db/migrations/
// 0006_socialplay_capacity_guard.sql) — not app.Service.RegisterForGame's
// in-process get -> list -> insert pre-check — is what actually stops a
// Game's registrations exceeding its capacity under real concurrency. This
// is the fix for the PM+PE loop-1 review finding on PR #14: the
// no-double-registration race (TestCreateRegistration_ExactlyOneWinsUnderConcurrency,
// same file set) was already closed by a unique index, but nothing
// previously enforced the capacity invariant itself in Postgres — two
// concurrent registrations for the last open slot targeted *different*
// (game_id, player_id) keys, so the unique index didn't help, and the
// in-process count check alone can't close a cross-goroutine race.
//
// Excluded from `make test-domain` and `go test ./...` by the integration
// build tag; run it with `go test -tags=integration ./...` or `make test`.
// Requires Docker. Shares waitForReady/applyMigrations/mustRange/
// seedCourtID/uuidLike with concurrency_integration_test.go in this same
// package.
package postgres_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	socialplaypg "github.com/nhuthuynh/white-label/internal/socialplay/adapter/postgres"
	socialplayapp "github.com/nhuthuynh/white-label/internal/socialplay/app"
	"github.com/nhuthuynh/white-label/internal/socialplay/domain"

	"github.com/nhuthuynh/white-label/internal/platform/idgen"
)

// gameCapacity is deliberately small and concurrentRegisterAttemptsForCapacity
// deliberately > it by a wide margin: the guard must hold regardless of how
// many requests race for the few remaining slots, not just the simplest
// one-slot case.
const (
	gameCapacity                          = 5
	concurrentRegisterAttemptsForCapacity = 20
)

func TestRegisterForGame_CapacityHoldsUnderConcurrency(t *testing.T) {
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
	waitlistRepo := socialplaypg.NewWaitlistRepository(pool)
	matchRepo := socialplaypg.NewMatchRepository(pool)
	svc := socialplayapp.NewService(idgen.UUID{}, gameRepo, regRepo, waitlistRepo, matchRepo)

	r := mustRange(t, "2026-09-01T09:00:00Z", "2026-09-01T10:00:00Z")
	game, err := domain.NewGame("22222222-2222-2222-2222-100000000001", "host-x", "facility-x", "", []string{seedCourtID}, r, gameCapacity, domain.PaymentMethodEither, 0, domain.Money{Cents: 1500, Currency: "USD"})
	if err != nil {
		t.Fatalf("bad fixture game: %v", err)
	}
	game, err = gameRepo.Create(ctx, game)
	if err != nil {
		t.Fatalf("failed to create fixture game: %v", err)
	}

	// Every goroutine registers a DIFFERENT player for the SAME game,
	// through app.Service.RegisterForGame — the real call path (get -> list
	// -> domain.Register's pre-check -> insert) — so distinct
	// (game_id, player_id) keys mean the unique index can't be what closes
	// this race; only the capacity guard trigger can.
	var wg sync.WaitGroup
	var mu sync.Mutex
	successes, full, unexpected := 0, 0, 0

	for i := 0; i < concurrentRegisterAttemptsForCapacity; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_, err := svc.RegisterForGame(ctx, socialplayapp.RegisterForGameInput{
				GameID:   game.ID,
				PlayerID: fmt.Sprintf("player-%02d", n),
			})
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				successes++
			case errors.Is(err, domain.ErrGameFull):
				full++
			default:
				unexpected++
				t.Logf("unexpected error: %v", err)
			}
		}(i)
	}
	wg.Wait()

	if successes != gameCapacity {
		t.Errorf("successes = %d, want exactly %d (game capacity)", successes, gameCapacity)
	}
	if full != concurrentRegisterAttemptsForCapacity-gameCapacity {
		t.Errorf("full = %d, want %d", full, concurrentRegisterAttemptsForCapacity-gameCapacity)
	}
	if unexpected != 0 {
		t.Errorf("unexpected errors = %d, want 0", unexpected)
	}

	// Belt-and-braces: the actual persisted row count must never exceed
	// capacity, regardless of what the in-process counters above say.
	active, err := regRepo.ListActiveForGame(ctx, game.ID)
	if err != nil {
		t.Fatalf("failed to list active registrations: %v", err)
	}
	if len(active) != gameCapacity {
		t.Errorf("persisted active registrations = %d, want exactly %d", len(active), gameCapacity)
	}
}
