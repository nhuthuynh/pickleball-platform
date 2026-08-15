//go:build integration

// This is a "large" test (Google's testing-pyramid terminology): it starts a
// real Postgres in a container and proves the EXCLUDE constraint — not the
// domain-level pre-check — is what actually stops a double-booking under
// real concurrency. Excluded from `make test-domain` and `go test ./...` by
// the integration build tag; run it with `go test -tags=integration ./...`
// or `make test`. Requires Docker.
//
// Manually verified against a real local Postgres 16 instance during T4
// development (20 concurrent CreateBooking calls on an identical slot -> 1
// success, 19 domain.ErrCourtDoubleBooked, 0 unexpected errors) in an
// environment where testcontainers itself couldn't run (no Docker daemon) —
// see docs/reviews/04-t4-concurrency-invariant.md for that methodology and
// result. This file is the portable, CI-runnable version of that same test.
package postgres_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	bookingpg "github.com/nhuthuynh/white-label/internal/booking/adapter/postgres"
	bookingapp "github.com/nhuthuynh/white-label/internal/booking/app"
	"github.com/nhuthuynh/white-label/internal/booking/domain"
	"github.com/nhuthuynh/white-label/internal/platform/idgen"
)

// concurrentCreateAttempts is deliberately > a handful: the EXCLUDE
// constraint must hold regardless of how many requests race, not just for
// the simplest two-request case.
const concurrentCreateAttempts = 20

func TestCreateBooking_ExactlyOneWinsUnderConcurrency(t *testing.T) {
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

	repo := bookingpg.NewRepository(pool)
	pricingRepo := bookingpg.NewPricingRuleRepository(pool)
	// T11.2's and T11.5's new dependencies. This test exercises CreateBooking
	// only, which touches none of them — the real Postgres
	// DiscountRuleRepository is wired anyway (it is what production uses,
	// and it costs nothing here), while FacilityLookup/RecurringHireRepository/
	// IdentityLookup are nil-free stand-ins rather than the real
	// Facilities/Identity-backed adapters: wiring those would pull two
	// entire other contexts' app.Services into a test about one EXCLUDE
	// constraint, and a nil interface would panic the moment anything did
	// reach it.
	discountRepo := bookingpg.NewDiscountRuleRepository(pool)
	svc := bookingapp.NewService(bookingapp.ServiceOptions{
		Bookings:       repo,
		PricingRules:   pricingRepo,
		DiscountRules:  discountRepo,
		RecurringHires: unusedRecurringHireRepository{},
		Facilities:     unusedFacilityLookup{},
		Identity:       unusedIdentityLookup{},
		IDs:            idgen.UUID{},
	})

	rng := mustRange(t, "2026-09-01T09:00:00Z", "2026-09-01T10:00:00Z")

	var wg sync.WaitGroup
	var mu sync.Mutex
	successes, conflicts, unexpected := 0, 0, 0

	for i := 0; i < concurrentCreateAttempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := svc.CreateBooking(ctx, bookingapp.CreateBookingInput{
				CourtID: seedCourtID,
				Source:  domain.SourceIndividual,
				Range:   rng,
			})
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				successes++
			case errors.Is(err, domain.ErrCourtDoubleBooked):
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
	if conflicts != concurrentCreateAttempts-1 {
		t.Errorf("conflicts = %d, want %d", conflicts, concurrentCreateAttempts-1)
	}
	if unexpected != 0 {
		t.Errorf("unexpected errors = %d, want 0", unexpected)
	}
}

const seedCourtID = "11111111-1111-1111-1111-111111111111"

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
// docker-compose's initdb.d does for local dev (see CLAUDE.md gotchas) —
// this keeps the integration test honest against the same schema a real
// deploy would use, rather than a hand-rolled test schema that could drift.
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

// unusedFacilityLookup satisfies port.FacilityLookup without reaching the
// Facilities context. This test's CreateBooking path never calls it (only
// GetQuote and CreateDiscountRule do), so its methods report the same
// "no Facility resolved" answer the real adapter gives for an unknown
// reference rather than pretending an ownership check succeeded — if a
// future change did route through here, it would fail closed.
type unusedFacilityLookup struct{}

func (unusedFacilityLookup) EnsureFacilityOwner(_ context.Context, _, _ string) error {
	return domain.ErrFacilityNotFound
}

func (unusedFacilityLookup) FacilityIDForCourt(_ context.Context, _ string) (string, error) {
	return "", domain.ErrFacilityNotFound
}

func (unusedFacilityLookup) CourtIDsForFacility(_ context.Context, _ string) ([]string, error) {
	return nil, domain.ErrFacilityNotFound
}

// unusedRecurringHireRepository and unusedIdentityLookup are this test's
// T11.5 equivalents of unusedFacilityLookup above, for the same reason: this
// test exercises CreateBooking only, which reaches neither RecurringHire
// approval nor an Identity lookup, so a real Postgres-backed
// RecurringHireRepository or Identity-backed adapter would be dead weight,
// and a nil interface would panic the moment anything unexpectedly did
// reach it. Each fails closed rather than silently succeeding.
type unusedRecurringHireRepository struct{}

func (unusedRecurringHireRepository) Create(_ context.Context, _ domain.RecurringHireTemplate) (domain.RecurringHireTemplate, error) {
	return domain.RecurringHireTemplate{}, domain.ErrRecurringHireTemplateNotFound
}

func (unusedRecurringHireRepository) GetByID(_ context.Context, _ string) (domain.RecurringHireTemplate, error) {
	return domain.RecurringHireTemplate{}, domain.ErrRecurringHireTemplateNotFound
}

func (unusedRecurringHireRepository) UpdateStatus(_ context.Context, _ domain.RecurringHireTemplate) (domain.RecurringHireTemplate, error) {
	return domain.RecurringHireTemplate{}, domain.ErrRecurringHireTemplateNotFound
}

func (unusedRecurringHireRepository) ListForCourts(_ context.Context, _ []string) ([]domain.RecurringHireTemplate, error) {
	return nil, nil
}

// ListForRequester was added to port.RecurringHireRepository in T11.6. It is
// listed here for the same reason as the four above — and because this file is
// behind `//go:build integration`, a plain `go vet ./...` would NOT have
// noticed it missing: T11.5 broke exactly this check by growing the Service's
// wiring without updating this file, so `go vet -tags=integration ./...` is
// part of this ticket's verification list, not an optional extra.
func (unusedRecurringHireRepository) ListForRequester(_ context.Context, _ string) ([]domain.RecurringHireTemplate, error) {
	return nil, nil
}

type unusedIdentityLookup struct{}

func (unusedIdentityLookup) EnsureClubRole(_ context.Context, _ string) error {
	return domain.ErrUserNotFound
}

// UserIDBySubject joined port.IdentityLookup in T13.2 (ADR-0014). It fails
// closed like every other method here — this test reaches no actor path at
// all, and a stub that returned a plausible id would be a stub that could
// hide a wiring mistake instead of exposing one.
//
// That this file needed touching at all is the point of `make
// vet-integration`: the build tag hides it from `go test ./...` and
// `go vet ./...`, so the port change compiled clean everywhere a machine
// without Docker can look, and only the T12.1 gate caught it (see CLAUDE.md's
// gotcha on the 11 integration-gated files).
func (unusedIdentityLookup) UserIDBySubject(_ context.Context, _ string) (string, error) {
	return "", domain.ErrUserNotFound
}
