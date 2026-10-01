//go:build integration

// This is a "large" test (Google's testing-pyramid terminology): it starts a
// real Postgres in a container and proves payments_payable_unique_idx
// (UNIQUE (payable_type, payable_id)) — not just the domain-level pre-check
// — is what actually stops a duplicate offline-payment recording under real
// concurrency. Excluded from `make test-domain` and plain `go test ./...` by
// the integration build tag; run it with `go test -tags=integration ./...`
// or `make test`. Requires Docker.
//
// T19.2 (closes #213): this invariant was disclosed at T6.4 and only ever
// proven once, manually, via an uncommitted throwaway program — see
// smoke_integration_test.go's own header for that same fallback history.
// This file is the committed, CI-runnable proof that was missing, mirroring
// internal/booking/adapter/postgres/concurrency_integration_test.go's and
// internal/socialplay/adapter/postgres/capacity_concurrency_integration_test.go's
// pattern exactly: fire N=20 concurrent RecordOfflinePayment calls at the
// same (payable_type, payable_id) pair and assert exactly 1 success and
// exactly N-1 domain.ErrPaymentAlreadyRecorded — the precise count, not
// "at least one."
//
// This authoring environment has no Docker daemon, so this committed test
// could not itself be executed here (same standing gap as every prior
// sprint's `-tags=integration` files, CLAUDE.md's own gotcha). The identical
// 20-concurrent-call scenario was instead verified manually against a
// throwaway, uncommitted program — see this ticket's PR description for
// whether a local Postgres was actually reachable in this environment, and
// the exact run count and outcomes if it was. This file changes no
// production code: payments_payable_unique_idx already correctly enforces
// the invariant under test; only the committed regression proof was
// missing.
package postgres_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	paymentspg "github.com/nhuthuynh/white-label/internal/payments/adapter/postgres"
	paymentsapp "github.com/nhuthuynh/white-label/internal/payments/app"
	"github.com/nhuthuynh/white-label/internal/payments/domain"
	"github.com/nhuthuynh/white-label/internal/platform/idgen"

	"github.com/jackc/pgx/v5/pgxpool"
)

// concurrentRecordAttempts matches T6.4's original claim (20 concurrent
// RecordOfflinePayment calls -> 1 success, 19 conflicts), the same count
// instruction 2 of T19.2 asks be reproduced exactly, not just "several."
const concurrentRecordAttempts = 20

func TestRecordOfflinePayment_ExactlyOneWinsUnderConcurrency(t *testing.T) {
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

	// waitForReady/applyMigrations are shared, package-private helpers
	// already defined in smoke_integration_test.go (same package,
	// postgres_test) — reused here rather than duplicated, per this
	// ticket's own instruction 1.
	waitForReady(t, ctx, pool)
	applyMigrations(t, ctx, pool)

	repo := paymentspg.NewRepository(pool)
	svc := paymentsapp.NewService(paymentsapp.ServiceOptions{
		Payments: repo,
		IDs:      idgen.UUID{},
	})

	// PayableTypeBooking, same reasoning as smoke_integration_test.go's own
	// AC1/AC3: this package's scope is the Postgres Repository — the
	// payable_type/payable_id unique index under test — not cross-context
	// resolution, so it does not need to wire RegistrationLookup/
	// GameLookup/GameAdminReader. ActorUserID == BookingHostID on every one
	// of the 20 calls so every attempt passes authorization identically and
	// the only thing distinguishing a winner from a loser is the race for
	// the unique index, not an authorization difference between attempts.
	// T61: actorID is a uuid with a real identity_users row behind it.
	//
	// payments.recorded_by_user_id became `uuid REFERENCES identity_users (id)`
	// at T28.1 (migration 0024). This fixture predated that and still used
	// "host-concurrency-1", which mustUUID panics on — taking the whole test
	// binary down and marking every other test in this package failed. It was
	// invisible because no Docker-free gate runs these tests, and
	// `make vet-integration` only COMPILES them.
	const (
		payableID = "44444444-4444-4444-4444-444444444444"
		actorID   = "55555555-5555-5555-5555-555555555555"
	)
	seedPaymentsActor(t, ctx, pool, actorID)
	amount := domain.Money{Cents: 1500, Currency: "USD"}

	var wg sync.WaitGroup
	var mu sync.Mutex
	successes, conflicts, unexpected := 0, 0, 0

	for i := 0; i < concurrentRecordAttempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p, err := svc.RecordOfflinePayment(ctx, paymentsapp.RecordOfflinePaymentInput{
				PayableType:   domain.PayableTypeBooking,
				PayableID:     payableID,
				Amount:        amount,
				ActorUserID:   actorID,
				BookingHostID: actorID,
			})
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				if p.Status != domain.StatusPaid {
					unexpected++
					t.Logf("unexpected: success but Status = %v, want %v", p.Status, domain.StatusPaid)
					return
				}
				successes++
			case errors.Is(err, domain.ErrPaymentAlreadyRecorded):
				conflicts++
			default:
				unexpected++
				t.Logf("unexpected error: %v", err)
			}
		}()
	}
	wg.Wait()

	if successes != 1 {
		t.Errorf("successes = %d, want exactly 1", successes)
	}
	if conflicts != concurrentRecordAttempts-1 {
		t.Errorf("conflicts = %d, want %d", conflicts, concurrentRecordAttempts-1)
	}
	if unexpected != 0 {
		t.Errorf("unexpected outcomes = %d, want 0", unexpected)
	}
}

// seedPaymentsActor inserts the identity_users row
// payments.recorded_by_user_id's FK requires (T61). `subject` is NOT NULL
// UNIQUE as of migration 0019, so it is derived from the id.
func seedPaymentsActor(t *testing.T, ctx context.Context, pool *pgxpool.Pool, userID string) {
	t.Helper()

	if _, err := pool.Exec(ctx, `
		INSERT INTO identity_users (id, display_name, roles, self_reported_starting_level, subject)
		VALUES ($1, 'T61 Concurrency Actor', ARRAY['player'], 3, $2)
	`, userID, "auth0|t61-concurrency-"+userID); err != nil {
		t.Fatalf("seeding the concurrency actor: %v", err)
	}
}
