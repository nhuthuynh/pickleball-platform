//go:build integration

// T17.4 (issue #195). The real-Postgres half of the regression proof for
// discount_rules.facility_id that no in-memory fixture can supply — mirrors
// unknown_court_integration_test.go's identical split for T15.6/#185, and
// internal/facilities/adapter/postgres/
// facility_deleted_race_integration_test.go's twin proof for this ticket's
// other two FK paths (courts.facility_id, facility_camera_links.facility_id).
//
// The defect: CreateDiscountRule is reached only after app.Service's own
// EnsureFacilityOwner guard (Booking's port.FacilityLookup, over the real
// Facilities context — see app/service.go) already confirmed the target
// Facility exists. In the non-racing case that guard's own
// domain.ErrFacilityNotFound wins and the FK never fires. Under a concurrent
// delete of the Facility landing between that guard's read and this
// package's INSERT, discount_rules.facility_id's "REFERENCES facilities
// (id)" (db/migrations/0017_booking_discount_rules.sql) instead raises a raw
// 23503 that — before this ticket — translateDiscountErr had no arm for (see
// its own doc comment for why this file, not
// internal/facilities/adapter/postgres, is where #195's table's
// discount_rules.facility_id row actually lands).
//
// This test reproduces the race window deterministically, the same way
// facility_deleted_race_integration_test.go does and for the identical
// reason stated in that file's own package comment: insert the Facility
// directly with SQL (this package owns no Facilities repository — Booking's
// only view of a Facility is port.FacilityLookup's two-answer interface),
// confirm it exists, delete it, THEN call DiscountRuleRepository.Create —
// guaranteeing the exact interleaving #195 describes rather than hoping a
// goroutine race lands there.
//
// Reuses startPostgres/waitForReady/applyMigrations from
// unknown_court_integration_test.go/concurrency_integration_test.go (same
// package, same schema every other integration test in this package runs
// against).
//
// Excluded from `make test-domain`, `make test-adapters` and plain
// `go test ./...` by the integration build tag; compiled (not run) by `make
// vet-integration`, which `make ci` runs; executed by `make ci-integration`
// or `make test`. Requires Docker.
package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	bookingpg "github.com/nhuthuynh/white-label/internal/booking/adapter/postgres"
	"github.com/nhuthuynh/white-label/internal/booking/domain"
)

// TestCreateDiscountRule_FacilityDeletedInRaceWindowAgainstRealPostgres is
// the load-bearing claim: against a REAL database, a DiscountRule naming a
// Facility that existed moments ago but was deleted before this write
// reached Postgres produces a 23503 foreign-key violation that the adapter
// translates into domain.ErrFacilityNotFound — not a raw, wrapped infra
// error that would answer codes.Internal.
func TestCreateDiscountRule_FacilityDeletedInRaceWindowAgainstRealPostgres(t *testing.T) {
	ctx := context.Background()
	pool := startPostgres(t, ctx)
	repo := bookingpg.NewDiscountRuleRepository(pool)

	const facilityID = "00000000-0000-4000-c000-000000000091"
	if _, err := pool.Exec(ctx,
		`INSERT INTO facilities (id, owner_id, name, address) VALUES ($1, $2, $3, $4)`,
		facilityID, "00000000-0000-4000-b000-000000000001", "Race Window Facility", "1 Test Way"); err != nil {
		t.Fatalf("failed to seed fixture facility: %v", err)
	}

	// The guarding read app.Service.CreateDiscountRule's EnsureFacilityOwner
	// performs before its own write — proving it genuinely succeeds while
	// the Facility exists, not merely asserting that it would.
	var exists bool
	if err := pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM facilities WHERE id = $1)", facilityID).Scan(&exists); err != nil {
		t.Fatalf("failed to confirm fixture facility exists: %v", err)
	}
	if !exists {
		t.Fatal("fixture facility does not exist immediately after insert — test setup is broken")
	}

	// The concurrent delete landing in the window: by the time Create below
	// reaches Postgres, the row the guard just confirmed is gone.
	if _, err := pool.Exec(ctx, "DELETE FROM facilities WHERE id = $1", facilityID); err != nil {
		t.Fatalf("failed to delete fixture facility to open the race window: %v", err)
	}

	rule, err := domain.NewDiscountRule(
		"00000000-0000-4000-a000-000000000091",
		facilityID,
		domain.DiscountTypePercent,
		domain.DiscountAmount{Percent: 10},
		[]domain.Source{domain.SourceIndividual},
		time.Now().UTC(),
		domain.NoEnd(),
	)
	if err != nil {
		t.Fatalf("failed to build fixture discount rule: %v", err)
	}

	_, err = repo.Create(ctx, rule)
	if !errors.Is(err, domain.ErrFacilityNotFound) {
		t.Fatalf("Create(deleted facility) = %v, want domain.ErrFacilityNotFound", err)
	}
}

// TestCreateDiscountRule_KnownFacilityStillSucceedsAgainstRealPostgres is the
// control. Without it, an adapter that rejected every INSERT would satisfy
// the test above.
func TestCreateDiscountRule_KnownFacilityStillSucceedsAgainstRealPostgres(t *testing.T) {
	ctx := context.Background()
	pool := startPostgres(t, ctx)
	repo := bookingpg.NewDiscountRuleRepository(pool)

	const facilityID = "00000000-0000-4000-c000-000000000092"
	if _, err := pool.Exec(ctx,
		`INSERT INTO facilities (id, owner_id, name, address) VALUES ($1, $2, $3, $4)`,
		facilityID, "00000000-0000-4000-b000-000000000001", "Control Facility", "2 Test Way"); err != nil {
		t.Fatalf("failed to seed fixture facility: %v", err)
	}

	rule, err := domain.NewDiscountRule(
		"00000000-0000-4000-a000-000000000092",
		facilityID,
		domain.DiscountTypePercent,
		domain.DiscountAmount{Percent: 10},
		[]domain.Source{domain.SourceIndividual},
		time.Now().UTC(),
		domain.NoEnd(),
	)
	if err != nil {
		t.Fatalf("failed to build fixture discount rule: %v", err)
	}

	got, err := repo.Create(ctx, rule)
	if err != nil {
		t.Fatalf("Create(existing facility) = %v, want success", err)
	}
	if got.FacilityID != facilityID {
		t.Fatalf("persisted rule facility = %q, want %q", got.FacilityID, facilityID)
	}
}
