//go:build integration

// This is a "large" test (Google's testing-pyramid terminology): it starts a
// real Postgres in a container, applies the real migrations for both
// contexts, and proves T10.6's end-to-end wiring through the real adapters
// on both sides of the boundary — not fakes, and not just "the port was
// called" (internal/payments/app/service_test.go's
// fakeCompetitionEntryUpdater tests already prove that at the app layer). A
// live Competition/CompetitionEntry is created via the real Competitions
// stack, an offline Payment is recorded via the real Payments stack, and the
// CompetitionEntry's PaymentStatus is re-read from Postgres via the real
// Competitions stack again — proving the full round trip:
// RecordOfflinePayment -> EntryUpdater -> Competitions app.Service ->
// Postgres -> GetEntryByID observes "paid". Mirrors
// internal/payments/adapter/socialplay/cross_context_integration_test.go
// (T6.5) exactly, one context over. Excluded from `make test-domain` and
// plain `go test ./...` by the integration build tag; run it with
// `go test -tags=integration ./...` or `make test`. Requires Docker.
//
// NOT EXECUTED BY ITS AUTHOR. No Docker daemon was available in this
// ticket's sandbox (`docker ps` fails with "dial unix
// /var/run/docker.sock: connect: no such file or directory") — the same
// gap HANDOFF.md records for T4/T5.4/T6.4/T6.5/T7/T8.3/T8.7/T9.2/T9.4/
// T9.5. It was type-checked under its build tag (`go vet -tags=integration
// ./internal/payments/adapter/competitions/...`) and no further. It is
// committed anyway for the same reason those precedents were: it is the
// portable, CI-runnable guard against the exact routing regression #96
// exists to prevent, and CLAUDE.md rule 10 means this claim must not be
// overstated as "proven" or "reliable" until a run with a real Docker
// daemon says so.
package competitions_test

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	competitionspg "github.com/nhuthuynh/white-label/internal/competitions/adapter/postgres"
	competitionsapp "github.com/nhuthuynh/white-label/internal/competitions/app"
	competitionsdomain "github.com/nhuthuynh/white-label/internal/competitions/domain"
	"github.com/nhuthuynh/white-label/internal/platform/idgen"

	paymentscompetitions "github.com/nhuthuynh/white-label/internal/payments/adapter/competitions"
	paymentspg "github.com/nhuthuynh/white-label/internal/payments/adapter/postgres"
	paymentsapp "github.com/nhuthuynh/white-label/internal/payments/app"
	paymentsdomain "github.com/nhuthuynh/white-label/internal/payments/domain"
)

// stubReservation/stubFacilities/stubShareTokens stand in for Competitions'
// two cross-context ports (Booking, Facilities) and its token generator —
// this test's scope is the Payments <-> Competitions boundary
// (EntryUpdater), not court reservation, so a real booking round trip would
// add infrastructure, not proof, mirroring
// internal/competitions/adapter/postgres/capacity_concurrency_integration_test.go's
// identical choice.
type stubReservation struct{}

func (stubReservation) ReserveCourt(_ context.Context, _ string, _, _ time.Time, _ string) (string, error) {
	return "booking-fixture", nil
}
func (stubReservation) ReleaseCourt(_ context.Context, _ string) error { return nil }

type stubFacilities struct{}

func (stubFacilities) FacilityExists(_ context.Context, _ string) error { return nil }

type stubShareTokens struct{}

func (stubShareTokens) NewShareToken() (string, error) { return "unused", nil }

func TestRecordOfflinePayment_ReconcilesCompetitionEntryPaymentStatus_CrossContext(t *testing.T) {
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

	// Real Competitions stack (T9.4's Postgres adapter + app.Service), the
	// same shape cmd/server wires.
	competitionsRepo := competitionspg.NewRepository(pool)
	competitionsSvc := competitionsapp.NewService(competitionsapp.ServiceOptions{
		Competitions: competitionsRepo,
		IDs:          idgen.UUID{},
		Reservation:  stubReservation{},
		Facilities:   stubFacilities{},
		ShareTokens:  stubShareTokens{},
	})

	// Real Payments stack (T6.4's Postgres adapter + app.Service), wired
	// with the real EntryUpdater (T10.6) against the same competitionsSvc
	// instance — not a fake, and not a second Competitions stack that
	// happens to point at the same DB. EntryLookup/CompetitionAdminReader
	// (T16.2, closes #168) are wired the identical way, against the same
	// real competitionsSvc/real Postgres — this is what lets
	// RecordOfflinePayment below authorize the entry's real entrant without
	// a caller-supplied entrant_player_id.
	paymentsRepo := paymentspg.NewRepository(pool)
	entryUpdater := paymentscompetitions.NewEntryUpdater(competitionsSvc)
	entryLookup := paymentscompetitions.NewEntryLookup(competitionsSvc)
	competitionAdminReader := paymentscompetitions.NewCompetitionAdminReader(competitionsSvc)
	paymentsSvc := paymentsapp.NewService(paymentsapp.ServiceOptions{
		Payments:                paymentsRepo,
		IDs:                     idgen.UUID{},
		CompetitionEntryUpdater: entryUpdater,
		EntryLookup:             entryLookup,
		CompetitionAdminReader:  competitionAdminReader,
	})

	// Set up a live Competition + CompetitionEntry through the real
	// Competitions stack — no fixtures inserted by raw SQL, so this test
	// also proves the upstream Competition/Entry creation path is
	// compatible with the reconciliation wiring, not just that a
	// pre-seeded row can be updated.
	rng, err := competitionsdomain.NewTimeRange(
		time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC),
		time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("failed to build fixture time range: %v", err)
	}
	competition, err := competitionsdomain.NewCompetition(
		"", "host-1", "Spring Doubles Open", "",
		[]competitionsdomain.Session{{Range: rng, CourtIDs: []string{"court-1"}}},
		16, 2,
		competitionsdomain.PaymentMethodEither,
		competitionsdomain.Money{AmountCents: 2500, CurrencyCode: "AUD"},
		competitionsdomain.FormatDoubles,
		"share-token-t106",
	)
	if err != nil {
		t.Fatalf("failed to build fixture competition: %v", err)
	}
	competition.ID = "22222222-2222-2222-2222-222222222222"
	if _, err := competitionsRepo.Create(ctx, competition); err != nil {
		t.Fatalf("failed to persist fixture competition: %v", err)
	}

	entry, err := competitionsSvc.EnterCompetition(ctx, competitionsapp.EnterCompetitionInput{
		CompetitionID: competition.ID,
		PlayerID:      "player-1",
		Source:        competitionsdomain.EntrySourceApp,
	})
	if err != nil {
		t.Fatalf("failed to persist fixture entry: %v", err)
	}
	if entry.PaymentStatus != competitionsdomain.PaymentStatusUnpaid {
		t.Fatalf("fixture PaymentStatus = %v, want unpaid before any Payment is recorded", entry.PaymentStatus)
	}

	// The T10.6 AC: recording an offline payment for the live
	// CompetitionEntry through the real Payments stack. No
	// entrant_player_id (T16.2, closes #168): "player-1" is authorized
	// because it is genuinely entry.PlayerID, resolved end to end through
	// real Postgres via EntryLookup, not because the caller claims it.
	_, err = paymentsSvc.RecordOfflinePayment(ctx, paymentsapp.RecordOfflinePaymentInput{
		PayableType: paymentsdomain.PayableTypeCompetitionEntry,
		PayableID:   entry.ID,
		Amount:      paymentsdomain.Money{Cents: 2500, Currency: "AUD"},
		ActorUserID: "player-1",
	})
	if err != nil {
		t.Fatalf("RecordOfflinePayment: unexpected err: %v", err)
	}

	// Observable via a fresh GetEntryByID through the real Competitions
	// stack — not the in-memory value RecordOfflinePayment happened to
	// return, a second, independent read proving the Postgres round trip
	// actually persisted the projection.
	stored, err := competitionsRepo.GetEntryByID(ctx, entry.ID)
	if err != nil {
		t.Fatalf("GetEntryByID: unexpected err: %v", err)
	}
	if stored.PaymentStatus != competitionsdomain.PaymentStatusPaid {
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

// applyMigrations runs db/migrations/*.sql in filename order, mirroring
// internal/payments/adapter/socialplay/cross_context_integration_test.go's
// identical helper — this test needs both contexts' schema (payments,
// competitions), so it uses the same full directory.
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
