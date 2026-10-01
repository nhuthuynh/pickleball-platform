//go:build integration

// This is a "large" test (Google's testing-pyramid terminology): it starts a
// real Postgres in a container, applies the real migrations for both
// contexts, and proves T6.5's end-to-end wiring through the real adapters
// on both sides of the boundary — not fakes, and not just "the port was
// called" (internal/payments/app/service_test.go's fakeRegistrationUpdater
// tests already prove that at the app layer). A live Game/Registration is
// created via the real Social Play stack, an offline Payment is recorded
// via the real Payments stack, and the Registration's PaymentStatus is
// re-read from Postgres via the real Social Play stack again — proving the
// full round trip: RecordOfflinePayment -> RegistrationUpdater ->
// Social Play app.Service -> Postgres -> GetRegistrationByID observes
// "paid". Excluded from `make test-domain` and plain `go test ./...` by the
// integration build tag; run it with `go test -tags=integration ./...` or
// `make test`. Requires Docker.
//
// T61 NOTE: this file's original header said no Docker daemon was available
// in the authoring environment and that the scenario had been verified
// manually against a local Postgres instead. T61 ran it for real, via
// testcontainers, in `make ci-integration` — where it failed immediately, on
// fixtures that migration 0026 had made invalid. The manual verification was
// genuine at T6.5; what it could not do is stay true, and nothing re-checked
// it for 32 sprints because no Docker-free gate executes this file and
// `make vet-integration` only COMPILES it.
package socialplay_test

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	paymentspg "github.com/nhuthuynh/white-label/internal/payments/adapter/postgres"
	paymentssocialplay "github.com/nhuthuynh/white-label/internal/payments/adapter/socialplay"
	paymentsapp "github.com/nhuthuynh/white-label/internal/payments/app"
	paymentsdomain "github.com/nhuthuynh/white-label/internal/payments/domain"
	"github.com/nhuthuynh/white-label/internal/platform/idgen"
	socialplaypg "github.com/nhuthuynh/white-label/internal/socialplay/adapter/postgres"
	socialplayapp "github.com/nhuthuynh/white-label/internal/socialplay/app"
	socialplaydomain "github.com/nhuthuynh/white-label/internal/socialplay/domain"
)

func TestRecordOfflinePayment_ReconcilesRegistrationPaymentStatus_CrossContext(t *testing.T) {
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

	// Real Social Play stack (T5.4's Postgres adapter + app.Service), the
	// same shape cmd/server wires.
	gameRepo := socialplaypg.NewGameRepository(pool)
	regRepo := socialplaypg.NewRegistrationRepository(pool)
	waitlistRepo := socialplaypg.NewWaitlistRepository(pool)
	matchRepo := socialplaypg.NewMatchRepository(pool)
	socialplaySvc := socialplayapp.NewService(socialplayapp.ServiceOptions{
		Identity:      fakeSocialplayIdentityLookup{},
		IDs:           idgen.UUID{},
		Games:         gameRepo,
		Registrations: regRepo,
		Waitlist:      waitlistRepo,
		Matches:       matchRepo,
		GameAdmins:    socialplaypg.NewGameAdminRepository(pool),
	})

	// Real Payments stack (T6.4's Postgres adapter + app.Service), wired
	// with the real RegistrationUpdater (T6.5) against the same
	// socialplaySvc instance — not a fake, and not a second Social Play
	// stack that happens to point at the same DB. RegistrationLookup/
	// GameLookup/GameAdminReader (T16.2, closes #168) are wired the
	// identical way, against the same real socialplaySvc/real Postgres —
	// this is what lets RecordOfflinePayment below authorize the Registration's
	// real Host without a caller-supplied game_host_id.
	paymentsRepo := paymentspg.NewRepository(pool)
	registrationUpdater := paymentssocialplay.NewRegistrationUpdater(socialplaySvc)
	registrationLookup := paymentssocialplay.NewRegistrationLookup(socialplaySvc)
	gameLookup := paymentssocialplay.NewGameLookup(socialplaySvc)
	gameAdminReader := paymentssocialplay.NewGameAdminReader(socialplaySvc)
	paymentsSvc := paymentsapp.NewService(paymentsapp.ServiceOptions{
		Payments:            paymentsRepo,
		IDs:                 idgen.UUID{},
		RegistrationUpdater: registrationUpdater,
		RegistrationLookup:  registrationLookup,
		GameLookup:          gameLookup,
		GameAdminReader:     gameAdminReader,
	})

	// Set up a live Game + Registration through the real Social Play stack
	// — no fixtures inserted by raw SQL, so this test also proves the
	// upstream Game/Registration creation path is compatible with the
	// reconciliation wiring, not just that a pre-seeded row can be updated.
	rng, err := socialplaydomain.NewTimeRange(
		time.Date(2026, 8, 3, 9, 0, 0, 0, time.UTC),
		time.Date(2026, 8, 3, 10, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("failed to build fixture time range: %v", err)
	}
	// T61: hostUserID, the court id and the player below all have to be uuids
	// with real rows behind them. games.host_id and registrations.player_id
	// became `uuid NOT NULL REFERENCES identity_users (id)` at migration 0026
	// and games.court_ids has been `uuid[]` since 0005 — the old "host-1" /
	// "player-1" / "court-1" fixtures panicked in the adapter's mustUUID. The
	// court is 0002_seed.sql's Court 1 rather than a seeded one, matching what
	// every other integration test in the tree uses.
	hostUserID := seedCrossContextUser(t, ctx, pool, "cross-context-host")
	game, err := socialplaydomain.NewGame("", hostUserID, "facility-1", "", []string{"11111111-1111-1111-1111-111111111111"}, rng, 4, socialplaydomain.PaymentMethodEither, 0, socialplaydomain.Money{Cents: 1500, Currency: "USD"})
	if err != nil {
		t.Fatalf("failed to build fixture game: %v", err)
	}
	game.ID = "11111111-1111-1111-1111-111111111111"
	if _, err := gameRepo.Create(ctx, game); err != nil {
		t.Fatalf("failed to persist fixture game: %v", err)
	}

	reg, err := socialplaySvc.RegisterForGame(ctx, socialplayapp.RegisterForGameInput{
		GameID:   game.ID,
		PlayerID: seedCrossContextUser(t, ctx, pool, "cross-context-player"),
	})
	if err != nil {
		t.Fatalf("failed to persist fixture registration: %v", err)
	}
	if reg.PaymentStatus != socialplaydomain.PaymentStatusUnpaid {
		t.Fatalf("fixture PaymentStatus = %v, want unpaid before any Payment is recorded", reg.PaymentStatus)
	}

	// The T6.5 AC: recording an offline payment for the live Registration
	// through the real Payments stack. No game_host_id (T16.2, closes #168):
	// hostUserID is authorized because it is genuinely game.HostID, resolved
	// end to end through real Postgres via RegistrationLookup -> GameLookup,
	// not because the caller claims it.
	_, err = paymentsSvc.RecordOfflinePayment(ctx, paymentsapp.RecordOfflinePaymentInput{
		PayableType: paymentsdomain.PayableTypeRegistration,
		PayableID:   reg.ID,
		Amount:      paymentsdomain.Money{Cents: 2500, Currency: "USD"},
		ActorUserID: hostUserID,
	})
	if err != nil {
		t.Fatalf("RecordOfflinePayment: unexpected err: %v", err)
	}

	// Observable via a fresh GetByID through the real Social Play stack —
	// not the in-memory value RecordOfflinePayment happened to return, a
	// second, independent read proving the Postgres round trip actually
	// persisted the projection (the exact gap a pre-T6.5
	// UpdateRegistrationStatus-only query would have silently missed — see
	// the PR description).
	stored, err := regRepo.GetByID(ctx, reg.ID)
	if err != nil {
		t.Fatalf("GetByID: unexpected err: %v", err)
	}
	if stored.PaymentStatus != socialplaydomain.PaymentStatusPaid {
		t.Fatalf("PaymentStatus = %v, want paid", stored.PaymentStatus)
	}
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

// applyMigrations runs db/migrations/*.sql in filename order, mirroring the
// identical helper duplicated per-package by
// internal/payments/adapter/postgres and internal/socialplay/adapter/
// postgres's own integration tests — this test needs both contexts' schema
// (payments, games, registrations), so it uses the same full directory.
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

// seedCrossContextUser inserts an identity_users row and returns its uuid id.
//
// T61. Duplicated per package rather than shared, like every other
// container-boot/migrate helper in this tree — see applyMigrations above,
// which this package already duplicates for the same reason.
func seedCrossContextUser(t *testing.T, ctx context.Context, pool *pgxpool.Pool, label string) string {
	t.Helper()

	id := uuid.NewString()
	if _, err := pool.Exec(ctx, `
		INSERT INTO identity_users (id, display_name, roles, self_reported_starting_level, subject)
		VALUES ($1, $2, ARRAY['player'], 3, $3)
	`, id, "T61 "+label, "auth0|t61-payments-socialplay-"+id); err != nil {
		t.Fatalf("seeding identity_users for %s: %v", label, err)
	}
	return id
}
