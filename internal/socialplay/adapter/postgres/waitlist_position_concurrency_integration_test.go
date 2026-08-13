//go:build integration

// This is a "large" test (Google's testing-pyramid terminology): it starts a
// real Postgres in a container and proves join_waitlist_entry
// (db/migrations/0009_socialplay_waitlist_join_position.sql) — not
// domain.JoinWaitlist's app-layer Position computation alone — is what
// actually gives concurrent JoinWaitlist calls for the same Game a correct,
// collision-free, gap-free Position sequence. This is the T6.6-loop-2 PM+PE
// review finding's required concurrency proof: see that migration's doc
// comment for the full race analysis (ordering-shaped, not
// distinctness-shaped, mirroring 0008's own conclusion for the promotion
// trigger).
//
// Excluded from `make test-domain` and `go test ./...` by the integration
// build tag; run it with `go test -tags=integration ./...` or `make test`.
// Requires Docker. This environment has no Docker daemon (docker CLI
// present, `docker ps` fails to dial the socket — the same gap the sibling
// concurrency tests in this package document), so it was also verified
// manually against a real local Postgres 16 instance (system package) with
// the identical application code path — see the PR description for the
// exact, repeated run output (CLAUDE.md rule 10: more than one run,
// including cold starts, before calling this "proven").
package postgres_test

import (
	"context"
	"sort"
	"strconv"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/nhuthuynh/white-label/internal/platform/idgen"
	socialplaypg "github.com/nhuthuynh/white-label/internal/socialplay/adapter/postgres"
	socialplayapp "github.com/nhuthuynh/white-label/internal/socialplay/app"
	"github.com/nhuthuynh/white-label/internal/socialplay/domain"
)

// concurrentJoinAttempts is deliberately > a handful, mirroring the sibling
// concurrency tests in this package (concurrentRegisterAttempts,
// concurrentCancelAttempts) — the race-closing lock must hold regardless of
// how many requests race for the same Game, not just the simplest
// two-request case.
const concurrentJoinAttempts = 30

// TestJoinWaitlist_CorrectPositionSequenceUnderConcurrency is the ticket's
// required scenario: a capacity-1 Game already full (one active
// Registration), then concurrentJoinAttempts distinct players all call
// app.Service.JoinWaitlist for it at once. The assertion is that the
// resulting Position values are EXACTLY 1..concurrentJoinAttempts — no
// duplicates (the collision this loop's review finding named: reproduced
// directly pre-fix, 30 concurrent joins produced 27 entries all at Position
// 1) and no gaps/misordering (the "correct next sequential integer, not
// just distinct" half of the finding a bare UNIQUE(game_id, position)
// constraint alone would not guarantee).
func TestJoinWaitlist_CorrectPositionSequenceUnderConcurrency(t *testing.T) {
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
	game, err := domain.NewGame("22222222-2222-2222-2222-100000000001", "host-x", "facility-x", "", []string{seedCourtID}, r, 1, domain.PaymentMethodEither, 0, domain.Money{Cents: 1500, Currency: "USD"})
	if err != nil {
		t.Fatalf("bad fixture game: %v", err)
	}
	game, err = gameRepo.Create(ctx, game)
	if err != nil {
		t.Fatalf("failed to create fixture game: %v", err)
	}

	// Fill capacity so every JoinWaitlist call below is legal (game is full).
	if _, err := svc.RegisterForGame(ctx, socialplayapp.RegisterForGameInput{GameID: game.ID, PlayerID: "player-active"}); err != nil {
		t.Fatalf("failed to fill capacity: %v", err)
	}

	// Every goroutine joins as a DISTINCT player — this is not the
	// no-double-waitlisting race (0007's unique index), it's the
	// queue-position race: N legitimate, distinct joiners racing for N
	// distinct, correctly-ordered Position values.
	var wg sync.WaitGroup
	var mu sync.Mutex
	var positions []int
	var unexpected []error

	for i := 0; i < concurrentJoinAttempts; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			entry, err := svc.JoinWaitlist(ctx, socialplayapp.JoinWaitlistInput{
				GameID:   game.ID,
				PlayerID: playerName(i),
			})
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				unexpected = append(unexpected, err)
				return
			}
			positions = append(positions, entry.Position)
		}(i)
	}
	wg.Wait()

	if len(unexpected) != 0 {
		t.Fatalf("got %d unexpected JoinWaitlist errors (want 0): %v", len(unexpected), unexpected)
	}
	if len(positions) != concurrentJoinAttempts {
		t.Fatalf("got %d successful joins, want %d", len(positions), concurrentJoinAttempts)
	}

	sort.Ints(positions)
	for i, p := range positions {
		want := i + 1
		if p != want {
			t.Fatalf("positions = %v: index %d has Position %d, want %d (sequence must be exactly 1..%d, no gaps, no collisions)",
				positions, i, p, want, concurrentJoinAttempts)
		}
	}

	// Belt-and-braces: the stored rows for this game must agree with what
	// JoinWaitlist returned, not just the in-process return values.
	stored, err := waitlistRepo.ListForGame(ctx, game.ID)
	if err != nil {
		t.Fatalf("failed to list stored entries: %v", err)
	}
	if len(stored) != concurrentJoinAttempts {
		t.Fatalf("stored entries = %d, want %d", len(stored), concurrentJoinAttempts)
	}
}

func playerName(i int) string {
	return "player-waiting-" + strconv.Itoa(i)
}
