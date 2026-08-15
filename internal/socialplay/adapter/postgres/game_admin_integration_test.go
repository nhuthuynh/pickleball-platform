//go:build integration

// T14.4 — a "large" test (Google's testing-pyramid terminology) for the
// durable Game-Admin store (partial fix for #168): it starts a real Postgres
// and proves the two things the in-memory fakes in
// internal/socialplay/{app,adapter/grpcapi} structurally cannot.
//
//  1. **The read observes what the write wrote, through real SQL** — the
//     sprint plan's §A13 GAP A requirement (T14.4 instruction 5). A fake
//     round-trips a Go value; this round-trips through the actual
//     game_admins columns, so a wrong column mapping, a dropped assigned_by,
//     or a timestamp lost to the timestamptz conversion fails here and
//     nowhere else.
//  2. **game_admins' composite primary key is the authoritative duplicate
//     guard under concurrency**, not domain.AssignGameAdmin's pre-check
//     (CLAUDE.md rule 4). The pre-check is a get-then-insert across two
//     statements, so it cannot close a cross-goroutine race on its own; the
//     constraint can, and translateGameAdminErr is what turns its 23505 into
//     the same domain.ErrAlreadyGameAdmin the pre-check returns.
//
// Excluded from `make test-domain` and `go test ./...` by the integration
// build tag; run it with `go test -tags=integration ./...` or `make test`.
// Requires Docker — `make vet-integration` (T12.1) compiles it without.
// Shares waitForReady/applyMigrations/mustRange/seedCourtID with
// concurrency_integration_test.go in this same package.
package postgres_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	socialplaypg "github.com/nhuthuynh/white-label/internal/socialplay/adapter/postgres"
	"github.com/nhuthuynh/white-label/internal/socialplay/domain"
)

// concurrentAssignAttempts is deliberately well above 2: the guard must hold
// however many requests race, not merely in the simplest pair.
const concurrentAssignAttempts = 20

// newGameAdminTestPool boots a Postgres container with the schema applied and
// returns a live pool, mirroring the setup every other integration test in
// this package performs inline.
func newGameAdminTestPool(t *testing.T, ctx context.Context) *pgxpool.Pool {
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

// seedGameAdminFixtureGame persists a Game whose HostID is a SUBJECT rather
// than a uuid — which is the point, not a shortcut. ADR-0014 §5a rules that
// this context's actor columns are text subjects, and
// db/migrations/0020_socialplay_game_admins.sql follows games.host_id
// deliberately; a fixture using uuids everywhere would let a uuid column type
// slip into game_admins unnoticed.
func seedGameAdminFixtureGame(t *testing.T, ctx context.Context, pool *pgxpool.Pool, id, hostID string) domain.Game {
	t.Helper()

	r := mustRange(t, "2026-09-01T09:00:00Z", "2026-09-01T10:00:00Z")
	game, err := domain.NewGame(id, hostID, "facility-x", "", []string{seedCourtID}, r, 4, domain.PaymentMethodEither, 0, domain.Money{Cents: 1500, Currency: "USD"})
	if err != nil {
		t.Fatalf("bad fixture game: %v", err)
	}
	game, err = socialplaypg.NewGameRepository(pool).Create(ctx, game)
	if err != nil {
		t.Fatalf("failed to create fixture game: %v", err)
	}
	return game
}

// TestGameAdminRepository_ReadObservesWhatTheWriteWrote is instruction 5's
// requirement against real SQL, plus the revoke's own round trip.
func TestGameAdminRepository_ReadObservesWhatTheWriteWrote(t *testing.T) {
	ctx := context.Background()
	pool := newGameAdminTestPool(t, ctx)

	const (
		hostSubject  = "auth0|host-subject"
		adminSubject = "auth0|admin-subject"
	)
	game := seedGameAdminFixtureGame(t, ctx, pool, "22222222-2222-2222-2222-200000000001", hostSubject)
	repo := socialplaypg.NewGameAdminRepository(pool)

	before, err := repo.ListGameAdmins(ctx, game.ID)
	if err != nil {
		t.Fatalf("ListGameAdmins before assigning: %v", err)
	}
	if len(before) != 0 {
		t.Fatalf("a fresh Game has %d admin rows, want 0", len(before))
	}

	assignedAt := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	written, err := repo.Assign(ctx, domain.GameAdmin{
		GameID:     game.ID,
		UserID:     adminSubject,
		AssignedBy: hostSubject,
		AssignedAt: assignedAt,
	})
	if err != nil {
		t.Fatalf("Assign: %v", err)
	}
	if written.UserID != adminSubject || written.AssignedBy != hostSubject {
		t.Fatalf("Assign returned %+v, want the values it was given", written)
	}

	after, err := repo.ListGameAdmins(ctx, game.ID)
	if err != nil {
		t.Fatalf("ListGameAdmins after assigning: %v", err)
	}
	if len(after) != 1 {
		t.Fatalf("after one assignment the table holds %d rows, want 1", len(after))
	}

	got := after[0]
	if got.GameID != game.ID {
		t.Errorf("read back GameID %q, want %q", got.GameID, game.ID)
	}
	if got.UserID != adminSubject {
		t.Errorf("read back UserID %q, want %q — a subject must survive the text column unchanged (ADR-0014 §5a)", got.UserID, adminSubject)
	}
	if got.AssignedBy != hostSubject {
		t.Errorf("read back AssignedBy %q, want %q", got.AssignedBy, hostSubject)
	}
	if !got.AssignedAt.Equal(assignedAt) {
		t.Errorf("read back AssignedAt %v, want %v — the caller-supplied timestamp must survive the timestamptz round trip", got.AssignedAt, assignedAt)
	}

	// domain.HasGameAdmin over the real read is exactly what T14.5 consumes.
	if !domain.HasGameAdmin(after, adminSubject) {
		t.Error("HasGameAdmin does not resolve the admin the real store just recorded")
	}

	if err := repo.Revoke(ctx, game.ID, adminSubject); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	remaining, err := repo.ListGameAdmins(ctx, game.ID)
	if err != nil {
		t.Fatalf("ListGameAdmins after revoking: %v", err)
	}
	if len(remaining) != 0 {
		t.Fatalf("after the revoke the table holds %d rows, want 0", len(remaining))
	}
	if err := repo.Revoke(ctx, game.ID, adminSubject); !errors.Is(err, domain.ErrGameAdminNotFound) {
		t.Fatalf("revoking an already-revoked assignment = %v, want ErrGameAdminNotFound — "+
			"a DELETE that matched nothing must not report success", err)
	}
}

// TestGameAdminRepository_ExactlyOneAssignWinsUnderConcurrency proves the
// composite primary key, not the app-layer pre-check, is what makes duplicate
// assignment impossible — and that the adapter translates its violation into
// a domain error rather than leaking a pgconn.PgError upward (CLAUDE.md
// rule 5).
//
// Every goroutine writes the SAME (game_id, user_id), so this is the
// distinctness-shaped race the PK closes, unlike the capacity guard's
// ordering-shaped one.
//
// Per CLAUDE.md rule 10 a single green run is not proof: this test is written
// to be re-run (and to be run cold), and the PR body records how many times it
// actually was.
func TestGameAdminRepository_ExactlyOneAssignWinsUnderConcurrency(t *testing.T) {
	ctx := context.Background()
	pool := newGameAdminTestPool(t, ctx)

	const (
		hostSubject  = "auth0|race-host"
		adminSubject = "auth0|race-admin"
	)
	game := seedGameAdminFixtureGame(t, ctx, pool, "22222222-2222-2222-2222-200000000002", hostSubject)
	repo := socialplaypg.NewGameAdminRepository(pool)

	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		wins     int
		conflict int
		others   []error
	)
	start := make(chan struct{})

	for i := 0; i < concurrentAssignAttempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start

			_, err := repo.Assign(ctx, domain.GameAdmin{
				GameID:     game.ID,
				UserID:     adminSubject,
				AssignedBy: hostSubject,
				AssignedAt: time.Now(),
			})

			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				wins++
			case errors.Is(err, domain.ErrAlreadyGameAdmin):
				conflict++
			default:
				others = append(others, err)
			}
		}()
	}

	close(start)
	wg.Wait()

	if len(others) > 0 {
		t.Fatalf("%d assignment(s) failed with something other than ErrAlreadyGameAdmin: %v — "+
			"the adapter must translate the constraint violation into a domain error, never leak the driver's (CLAUDE.md rule 5)",
			len(others), others)
	}
	if wins != 1 {
		t.Fatalf("%d of %d concurrent assignments succeeded, want exactly 1 — "+
			"game_admins' composite primary key is the authoritative guard, not the app-layer pre-check",
			wins, concurrentAssignAttempts)
	}
	if conflict != concurrentAssignAttempts-1 {
		t.Fatalf("%d assignments were rejected as duplicates, want %d", conflict, concurrentAssignAttempts-1)
	}

	rows, err := repo.ListGameAdmins(ctx, game.ID)
	if err != nil {
		t.Fatalf("ListGameAdmins: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("the table holds %d rows after the race, want 1", len(rows))
	}
}

// TestGameAdminRepository_RejectsAnUnknownGame pins the FK translation: a
// game_id naming no Game is domain.ErrGameNotFound, not a raw driver error.
// app.Service checks existence first, so this path is defensive — which is
// exactly why it needs a test, since nothing else exercises it.
func TestGameAdminRepository_RejectsAnUnknownGame(t *testing.T) {
	ctx := context.Background()
	pool := newGameAdminTestPool(t, ctx)

	repo := socialplaypg.NewGameAdminRepository(pool)
	_, err := repo.Assign(ctx, domain.GameAdmin{
		GameID:     "22222222-2222-2222-2222-2000000000ff",
		UserID:     "auth0|nobody",
		AssignedBy: "auth0|nobody-else",
		AssignedAt: time.Now(),
	})
	if !errors.Is(err, domain.ErrGameNotFound) {
		t.Fatalf("assigning against a nonexistent Game = %v, want ErrGameNotFound", err)
	}
}

// TestGameAdmins_AreScopedToOneGame is the negative control for the read
// against real SQL: an assignment on one Game must not appear on another, or
// "per-game Game Admins" (CLAUDE.md's locked decision) would be a
// platform-wide role granted by any one Host.
func TestGameAdmins_AreScopedToOneGame(t *testing.T) {
	ctx := context.Background()
	pool := newGameAdminTestPool(t, ctx)

	const (
		hostSubject  = "auth0|scope-host"
		adminSubject = "auth0|scope-admin"
	)
	gameA := seedGameAdminFixtureGame(t, ctx, pool, "22222222-2222-2222-2222-200000000003", hostSubject)
	gameB := seedGameAdminFixtureGame(t, ctx, pool, "22222222-2222-2222-2222-200000000004", hostSubject)
	repo := socialplaypg.NewGameAdminRepository(pool)

	if _, err := repo.Assign(ctx, domain.GameAdmin{
		GameID: gameA.ID, UserID: adminSubject, AssignedBy: hostSubject, AssignedAt: time.Now(),
	}); err != nil {
		t.Fatalf("Assign on game A: %v", err)
	}

	adminsB, err := repo.ListGameAdmins(ctx, gameB.ID)
	if err != nil {
		t.Fatalf("ListGameAdmins on game B: %v", err)
	}
	if domain.HasGameAdmin(adminsB, adminSubject) {
		t.Fatal("an admin assigned to game A resolves as an admin of game B")
	}

	// And the same user CAN hold an assignment on both — the PK is composite,
	// not a global unique on user_id.
	if _, err := repo.Assign(ctx, domain.GameAdmin{
		GameID: gameB.ID, UserID: adminSubject, AssignedBy: hostSubject, AssignedAt: time.Now(),
	}); err != nil {
		t.Fatalf("assigning the same user to a second Game: %v — the uniqueness guard is per (game_id, user_id), not per user", err)
	}
}
