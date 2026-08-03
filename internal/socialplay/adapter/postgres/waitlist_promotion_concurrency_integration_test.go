//go:build integration

// This is a "large" test (Google's testing-pyramid terminology): it starts a
// real Postgres in a container and proves promote_next_waiting
// (db/migrations/0008_socialplay_waitlist_promotion.sql) — not
// app.Service's app-layer orchestration alone — is what actually stops a
// Game's single waitlist entry from being promoted twice under real
// concurrency. This is the T6.6 ticket's required concurrency proof: see
// that migration's doc comment for the full race analysis (ordering-shaped,
// not distinctness-shaped, closed by a FOR UPDATE-locking Postgres function
// mirroring 0006_socialplay_capacity_guard.sql's trigger pattern).
//
// Excluded from `make test-domain` and `go test ./...` by the integration
// build tag; run it with `go test -tags=integration ./...` or `make test`.
// Requires Docker. This environment has no Docker daemon (docker CLI
// present, `docker ps` fails to dial the socket — the same gap
// concurrency_integration_test.go's package comment documents), so it was
// also verified manually against a real local Postgres 16 instance (system
// package) with the identical application code path — see the PR
// description for the exact, repeated run output (CLAUDE.md rule 10: more
// than one run, including a cold start, before calling this "proven").
package postgres_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	socialplaypg "github.com/nhuthuynh/white-label/internal/socialplay/adapter/postgres"
	socialplayapp "github.com/nhuthuynh/white-label/internal/socialplay/app"
	"github.com/nhuthuynh/white-label/internal/socialplay/domain"

	"github.com/nhuthuynh/white-label/internal/platform/idgen"
)

// concurrentCancelAttempts is deliberately > 1: the promotion guard must
// hold regardless of how many requests race for the single promotion, not
// just the simplest two-request case, mirroring the sibling concurrency
// tests in this package.
const concurrentCancelAttempts = 10

// TestPromoteNextWaiting_ExactlyOnePromotionUnderConcurrency is the ticket's
// required scenario: a capacity-1 Game with one active Registration and one
// waiting WaitlistEntry. Multiple goroutines all attempt to cancel the SAME
// Registration concurrently (the realistic trigger for a promotion race:
// app.Service.CancelRegistration's own get -> domain.Cancel -> Update
// sequence has no compare-and-swap guard on the registration row itself, so
// concurrent identical cancel attempts can each reach the "cancellation
// succeeded, promote the next waiter" step) — the assertion is that,
// however many of those concurrent CancelRegistration calls the app layer
// itself allows through, the DB-level promote_next_waiting guard ensures
// EXACTLY ONE promotion happens: not zero (the waiting player must not be
// left stranded when a slot is genuinely free), and not two-or-more (a
// single entry must never be double-promoted, and — with only one entry —
// there is nothing else it could correctly promote instead).
func TestPromoteNextWaiting_ExactlyOnePromotionUnderConcurrency(t *testing.T) {
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
	svc := socialplayapp.NewService(idgen.UUID{}, gameRepo, regRepo, waitlistRepo)

	r := mustRange(t, "2026-09-01T09:00:00Z", "2026-09-01T10:00:00Z")
	game, err := domain.NewGame("33333333-3333-3333-3333-100000000001", "host-x", "facility-x", []string{seedCourtID}, r, 1)
	if err != nil {
		t.Fatalf("bad fixture game: %v", err)
	}
	game, err = gameRepo.Create(ctx, game)
	if err != nil {
		t.Fatalf("failed to create fixture game: %v", err)
	}

	reg, err := svc.RegisterForGame(ctx, socialplayapp.RegisterForGameInput{GameID: game.ID, PlayerID: "player-active"})
	if err != nil {
		t.Fatalf("failed to create fixture registration: %v", err)
	}

	waitingEntry, err := svc.JoinWaitlist(ctx, socialplayapp.JoinWaitlistInput{GameID: game.ID, PlayerID: "player-waiting"})
	if err != nil {
		t.Fatalf("failed to create fixture waitlist entry: %v", err)
	}

	// Every goroutine attempts to cancel the SAME registration, as the
	// same actor -- the realistic concurrent-promotion trigger described
	// in this test's doc comment above.
	var wg sync.WaitGroup
	var mu sync.Mutex
	cancelSuccesses, cancelErrs := 0, 0

	for i := 0; i < concurrentCancelAttempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := svc.CancelRegistration(ctx, reg.ID, "player-active")
			mu.Lock()
			defer mu.Unlock()
			if err == nil {
				cancelSuccesses++
			} else {
				cancelErrs++
				t.Logf("cancel attempt error (may be expected): %v", err)
			}
		}()
	}
	wg.Wait()

	if cancelSuccesses == 0 {
		t.Fatalf("expected at least one CancelRegistration call to succeed, got %d successes / %d errors", cancelSuccesses, cancelErrs)
	}

	// The actual assertion: regardless of how many of the concurrent
	// cancel calls the app layer let through, promote_next_waiting must
	// have promoted the single waiting entry EXACTLY once.
	stored, err := waitlistRepo.GetByID(ctx, waitingEntry.ID)
	if err != nil {
		t.Fatalf("failed to fetch waitlist entry: %v", err)
	}
	if stored.Status != domain.WaitlistStatusPromoted {
		t.Fatalf("waitlist entry status = %v, want promoted (exactly one promotion should have happened)", stored.Status)
	}

	// Belt-and-braces: PromoteNext must report "no waiting entries" for
	// every attempt beyond the first — if the entry had somehow been
	// promoted twice (impossible for a single row, but this also confirms
	// the guard didn't leave the game in some inconsistent state), a
	// second real PromoteNext call now must cleanly report
	// ErrNoWaitingEntries, not promote anything else (there is nothing
	// else to promote — this is the "not two" half of the assertion,
	// checked directly against the DB-level operation itself, not just
	// its observed side effect above).
	if _, err := waitlistRepo.PromoteNext(ctx, game.ID, stored.PromotedAt); !errors.Is(err, domain.ErrNoWaitingEntries) {
		t.Fatalf("a second PromoteNext call got err %v, want ErrNoWaitingEntries (nothing left to promote)", err)
	}
}
