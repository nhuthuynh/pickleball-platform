//go:build integration

// This is a "large" test (Google's testing-pyramid terminology): it starts a
// real Postgres in a container and proves the
// competition_entries_capacity_guard trigger (db/migrations/
// 0014_competitions.sql) enforces the WEIGHTED capacity invariant —
// sum(1 + guest_count), not a plain row COUNT(*) — under real concurrency.
//
// This is T9.4's non-negotiable proof standard (CLAUDE.md rule 10, QA
// dossier §5 item 7): N simultaneous EnterCompetition calls with varying
// guest counts against a Competition sized so only one combination fits,
// asserting the DB-level guard admits exactly the set domain.Enter would.
//
// WHY THE WEIGHTING NEEDS ITS OWN PROOF, not just a capacity test: a fixture
// where every entry has guest_count 0 gives every row weight exactly 1, and
// in that case a COUNT(*) trigger and a weighted-SUM trigger are
// indistinguishable — both produce identical results. The fixtures below
// deliberately give entries weights above 1, which is the exact case T8.7
// found Social Play's original count-based trigger got wrong (see
// db/migrations/0012_socialplay_guest_capacity.sql): COUNT(*) compares only
// row count to capacity, so up to `capacity` rows could land regardless of
// how many guests each brought, silently overbooking well past the real
// capacity. Competitions has never had a count-based version; this test is
// what keeps it that way.
//
// Excluded from `make test-domain` and plain `go test ./...` by the
// integration build tag; run it with `go test -tags=integration ./...` or
// `make test`. Requires Docker.
//
// ---------------------------------------------------------------------------
// HOW THE T9.4 EVIDENCE WAS ACTUALLY PRODUCED — read this before citing it
// ---------------------------------------------------------------------------
// No Docker daemon was available in this ticket's sandbox (the same gap
// HANDOFF.md records for T4/T5.4/T6.4/T6.5/T7/T8.3/T8.7/T9.2 — `docker ps`
// fails with "dial unix /var/run/docker.sock: connect: no such file or
// directory"), so testcontainers could not run here and THIS FILE WAS NOT
// EXECUTED BY ITS AUTHOR.
//
// It is committed anyway, deliberately: it is the portable, CI-runnable
// version of the proof, and T6.4's own review finding was that an
// uncommitted throwaway verification leaves nothing guarding the invariant
// against regression.
//
// The evidence behind T9.4's concurrency claim came instead from the
// documented manual-Postgres fallback (the methodology T6.4/T6.5/T8.3/T9.2
// used): db/migrations/*.sql applied to a real local Postgres 16 via psql,
// and a throwaway program running the same app.Service.EnterCompetition call
// path against it with the same fixtures as the tests below, repeated
// multiple times including cold starts (database dropped, migrations
// re-applied, Postgres restarted). The T9.4 PR description states plainly
// which of the two produced the evidence and includes the captured output of
// each run. Do not read a green CI run of this file as the same event, and
// do not describe the manual run as if this file executed.
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

	competitionspg "github.com/nhuthuynh/white-label/internal/competitions/adapter/postgres"
	competitionsapp "github.com/nhuthuynh/white-label/internal/competitions/app"
	"github.com/nhuthuynh/white-label/internal/competitions/domain"
	"github.com/nhuthuynh/white-label/internal/platform/idgen"
)

// seedCourtID is the court db/migrations/0002_seed.sql inserts. Sessions
// store court_ids as a plain uuid[] with no FK (Postgres can't express an
// array-element FK), so this only needs to be a well-formed uuid — but using
// the real seeded one keeps the fixture honest.
const seedCourtID = "11111111-1111-1111-1111-111111111111"

// --- stub ports -----------------------------------------------------------
//
// EnterCompetition touches only the repository and the ID generator, but
// app.NewService requires every dependency (none is optional — see
// app.ServiceOptions). These stubs satisfy the rest without affecting what
// is under test: the fixtures below are persisted through the repository
// directly, so no court is ever reserved during these tests.

type stubReservation struct{}

func (stubReservation) ReserveCourt(_ context.Context, _ string, _, _ time.Time, _ string) (string, error) {
	return "", nil
}
func (stubReservation) ReleaseCourt(_ context.Context, _ string) error { return nil }

type stubFacilities struct{}

func (stubFacilities) FacilityExists(_ context.Context, _ string) error { return nil }

type stubShareTokens struct{}

func (stubShareTokens) NewShareToken() (string, error) { return "unused", nil }

func newTestService(pool *pgxpool.Pool) (*competitionsapp.Service, *competitionspg.Repository) {
	repo := competitionspg.NewRepository(pool)
	svc := competitionsapp.NewService(competitionsapp.ServiceOptions{
		Competitions: repo,
		IDs:          idgen.UUID{},
		Reservation:  stubReservation{},
		Facilities:   stubFacilities{},
		ShareTokens:  stubShareTokens{},
	})
	return svc, repo
}

// seedCompetition persists a fixture Competition directly through the
// repository, bypassing ScheduleCompetition so no court reservation is
// involved — the capacity guard, not the booking path, is what's under test.
func seedCompetition(t *testing.T, ctx context.Context, repo *competitionspg.Repository, id string, capacity, guestAllowance int, shareToken string) domain.Competition {
	t.Helper()

	competition, err := domain.NewCompetition(
		id, "host-x", "Concurrency Fixture", "",
		[]domain.Session{{
			Range:    mustRange(t, "2026-09-01T09:00:00Z", "2026-09-01T12:00:00Z"),
			CourtIDs: []string{seedCourtID},
		}},
		capacity, guestAllowance,
		domain.PaymentMethodEither,
		domain.Money{AmountCents: 2500, CurrencyCode: "AUD"},
		domain.FormatDoubles,
		shareToken,
	)
	if err != nil {
		t.Fatalf("bad fixture competition: %v", err)
	}

	persisted, err := repo.Create(ctx, competition)
	if err != nil {
		t.Fatalf("failed to create fixture competition: %v", err)
	}
	return persisted
}

// attempt is one concurrent EnterCompetition call.
type attempt struct {
	playerID   string
	guestCount int
}

// runConcurrently fires every attempt simultaneously through the REAL
// app.Service.EnterCompetition path and tallies the outcomes.
//
// Going through app.Service (rather than the repository directly) is
// deliberate and is what makes this a proof about the DB guard: the service
// runs domain.Enter's pre-check first, and under concurrency that pre-check
// routinely PASSES for several racers at once — they all read the same
// pre-insert snapshot and all conclude they fit. Whatever stops the extras
// after that point can only be the trigger.
func runConcurrently(t *testing.T, ctx context.Context, svc *competitionsapp.Service, competitionID string, attempts []attempt) (successes, full, unexpected int) {
	t.Helper()

	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, a := range attempts {
		wg.Add(1)
		go func(a attempt) {
			defer wg.Done()
			_, err := svc.EnterCompetition(ctx, competitionsapp.EnterCompetitionInput{
				CompetitionID: competitionID,
				PlayerID:      a.playerID,
				GuestCount:    a.guestCount,
				Source:        domain.EntrySourceApp,
			})
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				successes++
			case errors.Is(err, domain.ErrCompetitionFull):
				full++
			default:
				unexpected++
				t.Logf("unexpected error for %s (guestCount=%d): %v", a.playerID, a.guestCount, err)
			}
		}(a)
	}
	wg.Wait()
	return successes, full, unexpected
}

// occupancyOf sums the WEIGHTED occupancy actually persisted — the quantity a
// COUNT(*)-based trigger never checks.
func occupancyOf(t *testing.T, ctx context.Context, repo *competitionspg.Repository, competitionID string) (occupied int, entries []domain.CompetitionEntry) {
	t.Helper()
	entries, err := repo.ListActiveEntriesForCompetition(ctx, competitionID)
	if err != nil {
		t.Fatalf("failed to list active entries: %v", err)
	}
	for _, e := range entries {
		occupied += 1 + e.GuestCount
	}
	return occupied, entries
}

// TestEnterCompetition_UniformGuestWeightHoldsUnderConcurrency is the
// starkest demonstration that the guard weights guests rather than counting
// rows: every one of the N concurrent attempts brings the SAME number of
// guests, so the count-based and weighted answers differ by a single,
// easily-checked number.
//
//   - capacity 7, guestCount 3 => weight 4 per attempt (entrant + 3 guests).
//   - A COUNT(*) trigger would accept up to 7 of the 20 attempts (whichever
//     commit first): 7 rows x weight 4 = 28 people in a 7-place Competition.
//   - The weighted trigger accepts only while the running sum stays <= 7. The
//     first acceptance brings the total to 4; a second would need 8 > 7, so
//     it is rejected. Exactly 1 of 20 can ever succeed, regardless of arrival
//     order — deterministic precisely because every attempt has the identical
//     weight, so there is no order-dependent partial packing to reason about.
func TestEnterCompetition_UniformGuestWeightHoldsUnderConcurrency(t *testing.T) {
	ctx := context.Background()
	pool := newTestPool(t, ctx)
	svc, repo := newTestService(pool)

	const (
		capacity          = 7
		guestCount        = 3 // weight 4
		concurrentEntries = 20
		expectedSuccesses = 1 // floor(7/4) = 1; a 2nd (weight 4) would need 8 > 7
	)

	competition := seedCompetition(t, ctx, repo, "44444444-4444-4444-4444-100000000001", capacity, guestCount, "concurrency-token-uniform")

	attempts := make([]attempt, 0, concurrentEntries)
	for i := 0; i < concurrentEntries; i++ {
		attempts = append(attempts, attempt{playerID: fmt.Sprintf("uniform-player-%02d", i), guestCount: guestCount})
	}

	successes, full, unexpected := runConcurrently(t, ctx, svc, competition.ID, attempts)

	if unexpected != 0 {
		t.Errorf("unexpected errors = %d, want 0 (every attempt must resolve to success or ErrCompetitionFull — a leaked pgconn.PgError would mean CLAUDE.md rule 5 was violated)", unexpected)
	}
	if successes != expectedSuccesses {
		t.Errorf("successes = %d, want exactly %d — the weighted guard must cap at floor(capacity/weight), NOT (incorrectly) at `capacity` rows", successes, expectedSuccesses)
	}
	if full != concurrentEntries-expectedSuccesses {
		t.Errorf("full = %d, want %d", full, concurrentEntries-expectedSuccesses)
	}

	occupied, entries := occupancyOf(t, ctx, repo, competition.ID)
	if occupied > capacity {
		t.Errorf("persisted weighted occupancy = %d, must never exceed capacity %d — this is the exact quantity a COUNT(*) trigger never checks", occupied, capacity)
	}
	if len(entries) != expectedSuccesses {
		t.Errorf("persisted active entries = %d, want exactly %d", len(entries), expectedSuccesses)
	}
}

// TestEnterCompetition_VaryingGuestCountsFillExactlyToCapacity covers the
// ticket's "varying guest_count" requirement directly: zero-guest and
// heavy-guest attempts race for the same Competition.
//
// WHICH attempts win is inherently order-dependent (the trigger's row lock
// serializes racers in whatever order their transactions acquire it, not by
// any bin-packing optimum), so the fixture is sized so that the TOTAL
// occupancy is deterministic regardless:
//
//   - capacity 7.
//   - "large" attempts: guestCount 3 (weight 4). At most one can ever
//     succeed, since two would need 8 > 7.
//   - "small" attempts: guestCount 0 (weight 1), and there are more of them
//     (10) than the largest possible remaining gap (7). A weight-1 attempt
//     succeeds whenever ANY room remains, and occupancy never decreases, so
//     with 10 of them available the running total is provably walked all the
//     way to exactly capacity no matter how the interleaving falls.
//
// So while the winning SET is non-deterministic, the guard is proven run
// after run to land on exactly 7/7 — never over, never short. That is
// precisely "admits exactly the set the domain check would": domain.Enter
// applies the identical (1 + GuestCount) weighted rule, so any state this
// trigger allows is one domain.Enter would have accepted sequentially, and
// anything it rejects is what domain.Enter would also reject.
func TestEnterCompetition_VaryingGuestCountsFillExactlyToCapacity(t *testing.T) {
	ctx := context.Background()
	pool := newTestPool(t, ctx)
	svc, repo := newTestService(pool)

	const (
		capacity        = 7
		largeGuestCount = 3 // weight 4
		smallGuestCount = 0 // weight 1
		largeAttempts   = 10
		smallAttempts   = 10
	)

	competition := seedCompetition(t, ctx, repo, "44444444-4444-4444-4444-100000000002", capacity, largeGuestCount, "concurrency-token-varying")

	attempts := make([]attempt, 0, largeAttempts+smallAttempts)
	for i := 0; i < largeAttempts; i++ {
		attempts = append(attempts, attempt{playerID: fmt.Sprintf("large-player-%02d", i), guestCount: largeGuestCount})
	}
	for i := 0; i < smallAttempts; i++ {
		attempts = append(attempts, attempt{playerID: fmt.Sprintf("small-player-%02d", i), guestCount: smallGuestCount})
	}

	successes, full, unexpected := runConcurrently(t, ctx, svc, competition.ID, attempts)

	if unexpected != 0 {
		t.Errorf("unexpected errors = %d, want 0", unexpected)
	}
	if successes+full != len(attempts) {
		t.Errorf("successes(%d) + full(%d) = %d, want %d (every attempt must resolve to exactly one of these outcomes)", successes, full, successes+full, len(attempts))
	}

	occupied, entries := occupancyOf(t, ctx, repo, competition.ID)
	if len(entries) != successes {
		t.Fatalf("persisted active entries = %d, want %d (must match the successes app.Service reported)", len(entries), successes)
	}
	if occupied > capacity {
		t.Fatalf("persisted weighted occupancy = %d, must never exceed capacity %d", occupied, capacity)
	}
	if occupied != capacity {
		t.Errorf("persisted weighted occupancy = %d, want exactly %d — a unit-weight attempt was always available to fill any remaining gap, so leftover room means the guard rejected something it should have accepted", occupied, capacity)
	}

	largeWinners := 0
	for _, e := range entries {
		if e.GuestCount == largeGuestCount {
			largeWinners++
		}
	}
	if largeWinners > 1 {
		t.Errorf("large (weight %d) winners = %d, want at most 1 — capacity %d cannot hold two", 1+largeGuestCount, largeWinners, capacity)
	}

	// The guard and the domain rule must agree on what remains. This is the
	// "admits exactly the set the domain check would" assertion made
	// explicit: domain.SpotsLeft, computed over the ACTUAL persisted rows,
	// must say the Competition is full — and a further entry must in fact be
	// refused.
	if free := domain.SpotsLeft(competition, entries); free != 0 {
		t.Errorf("domain.SpotsLeft over the persisted entries = %d, want 0 — the DB guard and the domain rule disagree about whether this Competition is full", free)
	}
	if _, err := svc.EnterCompetition(ctx, competitionsapp.EnterCompetitionInput{
		CompetitionID: competition.ID, PlayerID: "late-arrival", GuestCount: 0, Source: domain.EntrySourceApp,
	}); !errors.Is(err, domain.ErrCompetitionFull) {
		t.Errorf("a further entry into a full Competition returned %v, want ErrCompetitionFull", err)
	}
}

// TestEnterCompetition_ExactlyOneCombinationFits is the ticket's "sized so
// exactly one combination fits" case in its purest form: capacity 5 with
// every attempt weighing exactly 5 (entrant + 4 guests). One attempt fills
// the Competition completely and no second attempt of any size can follow.
//
// A COUNT(*) trigger would accept 5 of these — 25 people in a 5-place
// Competition, a 5x overbooking.
func TestEnterCompetition_ExactlyOneCombinationFits(t *testing.T) {
	ctx := context.Background()
	pool := newTestPool(t, ctx)
	svc, repo := newTestService(pool)

	const (
		capacity          = 5
		guestCount        = 4 // weight 5 — exactly fills the Competition
		concurrentEntries = 16
	)

	competition := seedCompetition(t, ctx, repo, "44444444-4444-4444-4444-100000000003", capacity, guestCount, "concurrency-token-exact")

	attempts := make([]attempt, 0, concurrentEntries)
	for i := 0; i < concurrentEntries; i++ {
		attempts = append(attempts, attempt{playerID: fmt.Sprintf("exact-player-%02d", i), guestCount: guestCount})
	}

	successes, full, unexpected := runConcurrently(t, ctx, svc, competition.ID, attempts)

	if unexpected != 0 {
		t.Errorf("unexpected errors = %d, want 0", unexpected)
	}
	if successes != 1 {
		t.Errorf("successes = %d, want exactly 1 — only one weight-%d entry fits in a capacity-%d Competition", successes, 1+guestCount, capacity)
	}
	if full != concurrentEntries-1 {
		t.Errorf("full = %d, want %d", full, concurrentEntries-1)
	}

	occupied, entries := occupancyOf(t, ctx, repo, competition.ID)
	if occupied != capacity {
		t.Errorf("persisted weighted occupancy = %d, want exactly %d", occupied, capacity)
	}
	if len(entries) != 1 {
		t.Errorf("persisted active entries = %d, want exactly 1 (a COUNT(*) trigger would have allowed %d rows here, a %dx overbooking)", len(entries), capacity, capacity)
	}
}

// --- container / schema helpers -------------------------------------------

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
// docker-compose's initdb.d does for local dev (see CLAUDE.md gotchas) — so
// this test runs against the same schema a real deploy would use, rather
// than a hand-rolled test schema that could drift from it.
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
