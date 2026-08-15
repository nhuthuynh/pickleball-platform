//go:build integration

// This is a "large" test (Google's testing-pyramid terminology): it starts a
// real Postgres in a container and proves the REWRITTEN
// registrations_capacity_guard trigger (db/migrations/
// 0012_socialplay_guest_capacity.sql, rewriting the T5.4-loop-2 trigger from
// db/migrations/0006_socialplay_capacity_guard.sql) enforces the *weighted*
// capacity invariant — sum(1 + guest_count), not a plain row COUNT(*) — under
// real concurrency, the same standard capacity_concurrency_integration_test.go
// held the original (headcount-only) trigger to.
//
// Why this needs its own test, not just an extension of
// TestRegisterForGame_CapacityHoldsUnderConcurrency: that test's fixture
// never sets GuestCount (every registration has weight exactly 1), so it
// cannot distinguish a COUNT(*)-based trigger from a weighted-SUM one — both
// produce the identical result when every row's weight is 1. The tests in
// this file specifically construct fixtures where a registration's guests
// push its weight above 1, which is exactly the case the old trigger got
// wrong (see 0012's own migration doc comment): COUNT(*) only ever compares
// row count to capacity, so up to `capacity` rows could land regardless of
// how many guests each one brought, silently overbooking the venue past its
// real capacity.
//
// Excluded from `make test-domain` and `go test ./...` by the integration
// build tag; run it with `go test -tags=integration ./...` or `make test`.
// Requires Docker.
//
// No Docker daemon was available in this ticket's (T8.7) sandbox (same gap
// T4/T7/T8.3 documented — `docker info` fails with "failed to connect to the
// docker API ... dial unix /var/run/docker.sock: connect: no such file or
// directory"). This file is still committed as the portable, CI-runnable
// version of the proof; it was manually verified in this environment against
// a real local Postgres 16 instance (not a container) instead, running the
// exact same app.Service.RegisterForGame call path against
// db/migrations/*.sql applied fresh, repeated multiple times including two
// full "cold start" runs (dropped database, freshly re-applied migrations)
// per CLAUDE.md rule 10 — see the T8.7 PR description for the captured
// output of each run.
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

func newTestPool(t *testing.T, ctx context.Context) *pgxpool.Pool {
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

// TestRegisterForGame_UniformGuestWeightHoldsUnderConcurrency is the
// starkest possible demonstration of the bug db/migrations/
// 0012_socialplay_guest_capacity.sql fixes: every one of the N concurrent
// attempts brings the SAME number of guests, so the old (COUNT(*)) trigger
// and the new (weighted SUM) trigger diverge in a single, easily-checked
// number.
//
//   - gameCapacityUniform = 7, guestCountUniform = 3 (weight 4 per
//     registration: the registering player + 3 guests).
//   - Old trigger: COUNT(*) against capacity 7 would accept up to 7 of the
//     20 concurrent attempts (whichever 7 commit first) — 7 rows × weight 4
//     = 28 people occupying a 7-person Game. A severe overbooking bug.
//   - New trigger: only accepts while the running weighted SUM stays
//     <= capacity. The first acceptance brings the total to 4; a second
//     would require 8 > 7, so it's rejected. Exactly 1 of the 20 attempts
//     can ever succeed, regardless of arrival order — this is deterministic
//     specifically because every attempt has the identical weight (4), so
//     there is no ordering-dependent partial packing to reason about.
func TestRegisterForGame_UniformGuestWeightHoldsUnderConcurrency(t *testing.T) {
	ctx := context.Background()
	pool := newTestPool(t, ctx)

	const (
		gameCapacityUniform       = 7
		guestCountUniform         = 3 // weight 4 per attempt (1 player + 3 guests)
		concurrentAttemptsUniform = 20
		expectedSuccessesUniform  = 1 // floor(7/4) = 1; a 2nd (weight 4) would need 8 > 7
	)

	gameRepo := socialplaypg.NewGameRepository(pool)
	regRepo := socialplaypg.NewRegistrationRepository(pool)
	waitlistRepo := socialplaypg.NewWaitlistRepository(pool)
	matchRepo := socialplaypg.NewMatchRepository(pool)
	svc := socialplayapp.NewService(socialplayapp.ServiceOptions{
		IDs:           idgen.UUID{},
		Games:         gameRepo,
		Registrations: regRepo,
		Waitlist:      waitlistRepo,
		Matches:       matchRepo,
		GameAdmins:    socialplaypg.NewGameAdminRepository(pool),
	})

	r := mustRange(t, "2026-09-01T09:00:00Z", "2026-09-01T10:00:00Z")
	game, err := domain.NewGame(
		"33333333-3333-3333-3333-100000000001", "host-x", "facility-x", "",
		[]string{seedCourtID}, r, gameCapacityUniform,
		domain.PaymentMethodEither, guestCountUniform, // GuestAllowance must cover guestCountUniform
		domain.Money{Cents: 1500, Currency: "USD"},
	)
	if err != nil {
		t.Fatalf("bad fixture game: %v", err)
	}
	game, err = gameRepo.Create(ctx, game)
	if err != nil {
		t.Fatalf("failed to create fixture game: %v", err)
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	successes, full, unexpected := 0, 0, 0

	for i := 0; i < concurrentAttemptsUniform; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_, err := svc.RegisterForGame(ctx, socialplayapp.RegisterForGameInput{
				GameID:     game.ID,
				PlayerID:   fmt.Sprintf("uniform-player-%02d", n),
				GuestCount: guestCountUniform,
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

	if successes != expectedSuccessesUniform {
		t.Errorf("successes = %d, want exactly %d — the weighted trigger must cap at floor(capacity/weight), not (incorrectly) at capacity rows", successes, expectedSuccessesUniform)
	}
	if full != concurrentAttemptsUniform-expectedSuccessesUniform {
		t.Errorf("full = %d, want %d", full, concurrentAttemptsUniform-expectedSuccessesUniform)
	}
	if unexpected != 0 {
		t.Errorf("unexpected errors = %d, want 0", unexpected)
	}

	// Belt-and-braces: sum the ACTUAL persisted occupancy (1 + guest_count
	// per row), not just the row count, and confirm it never exceeds
	// capacity — this is the exact quantity the old COUNT(*) trigger never
	// checked.
	active, err := regRepo.ListActiveForGame(ctx, game.ID)
	if err != nil {
		t.Fatalf("failed to list active registrations: %v", err)
	}
	occupied := 0
	for _, reg := range active {
		occupied += 1 + reg.GuestCount
	}
	if occupied > gameCapacityUniform {
		t.Errorf("persisted occupied capacity = %d, must never exceed game capacity %d", occupied, gameCapacityUniform)
	}
	if len(active) != expectedSuccessesUniform {
		t.Errorf("persisted active registrations = %d, want exactly %d", len(active), expectedSuccessesUniform)
	}
}

// TestRegisterForGame_VaryingGuestCountsFillExactlyToCapacityUnderConcurrency
// covers "varying guest_count values" directly: a mix of zero-guest and
// heavy-guest attempts race for the same Game. The specific WINNING
// combination is inherently order-dependent under real concurrency (the
// trigger's row lock serializes attempts in whatever order their
// transactions happen to acquire it, not by any bin-packing optimum) — but
// the fixture is deliberately sized so the *total occupied capacity* is
// deterministic regardless of that order:
//
//   - gameCapacityVarying = 7.
//   - "large" attempts: guestCount = 3 (weight 4). At most one can ever
//     succeed, since two would need 8 > 7 — this alone proves the same
//     weighted-vs-headcount point as the uniform test above.
//   - "small" attempts: guestCount = 0 (weight 1), and there are more of
//     them (10) than the largest possible remaining gap (7). Because a
//     weight-1 attempt always succeeds whenever ANY room remains (a fact
//     provable from the trigger's own monotonic-occupancy behaviour: total
//     occupancy never decreases, so if a size-1 gap ever exists while an
//     untried small attempt remains, that attempt is guaranteed to close
//     it), having 10 of them guarantees the running total is walked all the
//     way up to exactly capacity by the time every goroutine has been
//     processed, regardless of interleaving. So while WHICH specific
//     players win is non-deterministic, the DB-level guard is proven, run
//     after run, to always land on exactly 7/7 occupied — never over, never
//     short — which is precisely "the same set the domain-level check would
//     accept": domain.Register (see registration.go) uses the identical (1 +
//     GuestCount) weighted-sum rule, so any persisted state this trigger
//     allows is one domain.Register would also have accepted from a
//     sequential caller's point of view, and anything it rejects
//     (ErrGameFull) is exactly what domain.Register would also reject.
func TestRegisterForGame_VaryingGuestCountsFillExactlyToCapacityUnderConcurrency(t *testing.T) {
	ctx := context.Background()
	pool := newTestPool(t, ctx)

	const (
		gameCapacityVarying = 7
		largeGuestCount     = 3 // weight 4
		smallGuestCount     = 0 // weight 1
		largeAttempts       = 10
		smallAttempts       = 10
	)

	gameRepo := socialplaypg.NewGameRepository(pool)
	regRepo := socialplaypg.NewRegistrationRepository(pool)
	waitlistRepo := socialplaypg.NewWaitlistRepository(pool)
	matchRepo := socialplaypg.NewMatchRepository(pool)
	svc := socialplayapp.NewService(socialplayapp.ServiceOptions{
		IDs:           idgen.UUID{},
		Games:         gameRepo,
		Registrations: regRepo,
		Waitlist:      waitlistRepo,
		Matches:       matchRepo,
		GameAdmins:    socialplaypg.NewGameAdminRepository(pool),
	})

	r := mustRange(t, "2026-09-01T09:00:00Z", "2026-09-01T10:00:00Z")
	game, err := domain.NewGame(
		"33333333-3333-3333-3333-100000000002", "host-x", "facility-x", "",
		[]string{seedCourtID}, r, gameCapacityVarying,
		domain.PaymentMethodEither, largeGuestCount, // GuestAllowance must cover the largest guestCount used
		domain.Money{Cents: 1500, Currency: "USD"},
	)
	if err != nil {
		t.Fatalf("bad fixture game: %v", err)
	}
	game, err = gameRepo.Create(ctx, game)
	if err != nil {
		t.Fatalf("failed to create fixture game: %v", err)
	}

	type attempt struct {
		playerID   string
		guestCount int
	}
	var attempts []attempt
	for i := 0; i < largeAttempts; i++ {
		attempts = append(attempts, attempt{playerID: fmt.Sprintf("large-player-%02d", i), guestCount: largeGuestCount})
	}
	for i := 0; i < smallAttempts; i++ {
		attempts = append(attempts, attempt{playerID: fmt.Sprintf("small-player-%02d", i), guestCount: smallGuestCount})
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	successes, full, unexpected := 0, 0, 0

	for _, a := range attempts {
		wg.Add(1)
		go func(a attempt) {
			defer wg.Done()
			_, err := svc.RegisterForGame(ctx, socialplayapp.RegisterForGameInput{
				GameID:     game.ID,
				PlayerID:   a.playerID,
				GuestCount: a.guestCount,
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
				t.Logf("unexpected error for %s (guestCount=%d): %v", a.playerID, a.guestCount, err)
			}
		}(a)
	}
	wg.Wait()

	if unexpected != 0 {
		t.Errorf("unexpected errors = %d, want 0", unexpected)
	}
	if successes+full != largeAttempts+smallAttempts {
		t.Errorf("successes(%d) + full(%d) = %d, want %d (every attempt must resolve to exactly one of these two outcomes)", successes, full, successes+full, largeAttempts+smallAttempts)
	}

	active, err := regRepo.ListActiveForGame(ctx, game.ID)
	if err != nil {
		t.Fatalf("failed to list active registrations: %v", err)
	}
	if len(active) != successes {
		t.Fatalf("persisted active registrations = %d, want %d (must match the successes app.Service reported)", len(active), successes)
	}

	occupied := 0
	largeWinners := 0
	for _, reg := range active {
		occupied += 1 + reg.GuestCount
		if reg.GuestCount == largeGuestCount {
			largeWinners++
		}
	}

	// The core invariant: occupied capacity must never exceed the Game's
	// real capacity, regardless of which specific attempts won the race.
	if occupied > gameCapacityVarying {
		t.Fatalf("persisted occupied capacity = %d, must never exceed game capacity %d", occupied, gameCapacityVarying)
	}
	// The stronger, deterministic invariant this fixture is specifically
	// sized to prove (see doc comment above): with 10 unit-weight attempts
	// available -- more than the largest possible gap (7) -- the guard is
	// provably walked all the way to exactly full, never leaving room a
	// small attempt could still have used.
	if occupied != gameCapacityVarying {
		t.Errorf("persisted occupied capacity = %d, want exactly %d (a unit-weight attempt was available to fill any remaining gap; leftover room means the guard rejected something it should have accepted, or vice versa)", occupied, gameCapacityVarying)
	}
	// At most one "large" (weight 4) registration can ever fit in a
	// capacity-7 game (two would need 8 > 7) -- this is the same
	// weighted-vs-headcount point as the uniform test, restated here against
	// a mixed-weight fixture.
	if largeWinners > 1 {
		t.Errorf("large (weight %d) winners = %d, want at most 1 — capacity %d cannot hold two", 1+largeGuestCount, largeWinners, gameCapacityVarying)
	}
}
