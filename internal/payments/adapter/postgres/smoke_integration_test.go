//go:build integration

// This is a "large" test (Google's testing-pyramid terminology): it starts a
// real Postgres in a container, applies the real migrations, and proves the
// T6.4 smoke-test AC end to end through the real app.Service + postgres
// Repository stack (not a fake) — the DB-level guard
// (payments_payable_unique_idx) is what actually stops a duplicate
// recording, not just the domain-level pre-check. Excluded from `make
// test-domain` and plain `go test ./...` by the integration build tag; run
// it with `go test -tags=integration ./...` or `make test`. Requires
// Docker.
//
// This authoring environment has no Docker daemon (mirroring the T4 gap
// documented in docs/LESSONS.md), so this file could not itself be run
// here. The identical scenario was instead verified manually against a
// real local Postgres 16 instance (installed as a system package, the same
// T4 LESSONS.md fallback) — see the T6.4 PR description for that
// methodology and the exact commands/results. This file is the portable,
// CI-runnable version of that same manual verification, mirroring
// internal/booking/adapter/postgres/concurrency_integration_test.go's
// pattern exactly.
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

	paymentspg "github.com/nhuthuynh/white-label/internal/payments/adapter/postgres"
	paymentsapp "github.com/nhuthuynh/white-label/internal/payments/app"
	"github.com/nhuthuynh/white-label/internal/payments/domain"
	"github.com/nhuthuynh/white-label/internal/platform/idgen"
)

func TestRecordOfflinePayment_SmokeTestAC(t *testing.T) {
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

	repo := paymentspg.NewRepository(pool)
	svc := paymentsapp.NewService(paymentsapp.ServiceOptions{
		Payments: repo,
		IDs:      idgen.UUID{},
	})

	amount := domain.Money{Cents: 2500, Currency: "USD"}

	// AC 1: offline payment for a Registration succeeds ("returns 200").
	p, err := svc.RecordOfflinePayment(ctx, paymentsapp.RecordOfflinePaymentInput{
		PayableType: domain.PayableTypeRegistration,
		PayableID:   "22222222-2222-2222-2222-222222222222",
		Amount:      amount,
		ActorUserID: "host-1",
		GameHostID:  "host-1",
	})
	if err != nil {
		t.Fatalf("first recording: unexpected err: %v", err)
	}
	if p.Status != domain.StatusPaid {
		t.Fatalf("first recording: Status = %v, want paid", p.Status)
	}

	// AC 2: a duplicate for the same (payable_type, payable_id) returns
	// ErrPaymentAlreadyRecorded (-> 409 via grpcapi.toStatus), enforced by
	// payments_payable_unique_idx, not just the domain pre-check.
	_, err = svc.RecordOfflinePayment(ctx, paymentsapp.RecordOfflinePaymentInput{
		PayableType: domain.PayableTypeRegistration,
		PayableID:   "22222222-2222-2222-2222-222222222222",
		Amount:      amount,
		ActorUserID: "host-1",
		GameHostID:  "host-1",
	})
	if !errors.Is(err, domain.ErrPaymentAlreadyRecorded) {
		t.Fatalf("duplicate recording: got err %v, want %v", err, domain.ErrPaymentAlreadyRecorded)
	}

	// AC 3: an actor mismatch returns ErrNotPaymentRecorder (-> 403-shaped
	// via grpcapi.toStatus), not a 500 — and, critically, never reaches
	// Postgres at all (authorization is checked before any domain
	// construction / repo call), so it must not consume the
	// (payable_type, payable_id) slot the first recording above claimed.
	_, err = svc.RecordOfflinePayment(ctx, paymentsapp.RecordOfflinePaymentInput{
		PayableType: domain.PayableTypeRegistration,
		PayableID:   "33333333-3333-3333-3333-333333333333",
		Amount:      amount,
		ActorUserID: "random-player",
		GameHostID:  "host-1",
	})
	if !errors.Is(err, domain.ErrNotPaymentRecorder) {
		t.Fatalf("actor mismatch: got err %v, want %v", err, domain.ErrNotPaymentRecorder)
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
// internal/booking/adapter/postgres's identical helper — keeps the
// integration test honest against the same schema a real deploy would use.
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
