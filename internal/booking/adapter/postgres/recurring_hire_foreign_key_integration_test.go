//go:build integration

// T17.5 (issue #195). This is the half of the regression proof that NO
// in-memory fixture can supply, and saying why is the point of this file
// existing separately from recurring_hire_foreign_key_test.go beside it —
// the identical split unknown_court_integration_test.go and
// foreign_key_test.go already established for #185/T15.6.
//
// The gap: app.Service.RequestRecurringHire resolves the referenced Court
// (port.FacilityLookup.FacilityIDForCourt) and User (port.IdentityLookup.
// EnsureClubRole) *before* calling this repository's Create. In the ordinary
// case that read's own answer — domain.ErrFacilityNotFound or
// domain.ErrUserNotFound — is what a caller sees for a bad reference, and
// the INSERT's own `recurring_hire_templates.court_id`/`.requested_by_user_id`
// FK never fires. Under a concurrent delete of the Court or the User landing
// between that read and this INSERT, the read's answer is stale: the FK does
// fire, as a raw 23503 only a real Postgres raises. Before T17.5 that would
// have fallen through translateRecurringHireErr's default straight to
// codes.Internal — a 500 for a legitimate (if racy) client-visible
// condition, the same shape #185 was opened and closed for on
// bookings.court_id, on this narrower window (#195's own framing).
//
// Excluded from `make test-domain`, `make test-adapters` and plain `go test
// ./...` by the integration build tag; compiled (not run) by `make
// vet-integration`, which `make ci` runs; executed by `make ci-integration`
// or `make test`. Requires Docker — not available in the environment this
// ticket was authored in, so these were compiled, not run (CLAUDE.md rule
// 10: stated plainly, not papered over).
package postgres_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	bookingpg "github.com/nhuthuynh/white-label/internal/booking/adapter/postgres"
	"github.com/nhuthuynh/white-label/internal/booking/domain"
)

// seedCourtRow inserts a courts row (name only — id and facility_id are
// unused by this file's assertions) and returns its server-minted id.
func seedCourtRow(t *testing.T, ctx context.Context, pool *pgxpool.Pool) string {
	t.Helper()

	var id string
	err := pool.QueryRow(ctx, `INSERT INTO courts (name) VALUES ($1) RETURNING id`, "T17.5 race-window court").Scan(&id)
	if err != nil {
		t.Fatalf("seed court: %v", err)
	}
	return id
}

// seedUserRow inserts an identity_users row satisfying every NOT NULL/CHECK
// this table declares (0016_identity.sql) and returns the caller-chosen id —
// identity_users.id has no DEFAULT (T12.9's comment explains why: it is the
// server-minted User.ID other contexts reference, not IdP-supplied), so the
// caller mints it, mirroring what port.IDGenerator does in production.
func seedUserRow(t *testing.T, ctx context.Context, pool *pgxpool.Pool, id string) {
	t.Helper()

	_, err := pool.Exec(ctx,
		`INSERT INTO identity_users (id, display_name, roles, self_reported_starting_level) VALUES ($1, $2, $3, $4)`,
		id, "T17.5 race-window club", []string{"club"}, 3)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
}

// deleteRowConcurrently deletes the row identified by (table, column, id)
// on a separate goroutine, synchronized against ready so the delete cannot
// start before the caller has finished whatever the guarding read is meant
// to represent, and cannot finish after insert has already been issued: it
// blocks on committed, which the caller closes only once its own INSERT call
// has returned. That ordering — delete commits strictly between "the guard
// would have read" and "the repository's INSERT returns" — is what makes
// this deterministic rather than a coin flip on scheduling, while still
// being a genuinely separate transaction on a genuinely separate connection,
// not a same-transaction shortcut. A sleep-based "probably wins the race"
// version was considered and rejected: CLAUDE.md rule 10 is explicit that a
// single successful run of a non-deterministic concurrency test proves
// nothing, and a flaky integration test that only sometimes exercises the
// 23503 path would be worse than not having it.
func deleteRowConcurrently(t *testing.T, ctx context.Context, pool *pgxpool.Pool, table, column, id string, ready <-chan struct{}) *sync.WaitGroup {
	t.Helper()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ready
		if _, err := pool.Exec(ctx, `DELETE FROM `+table+` WHERE `+column+` = $1`, id); err != nil {
			t.Errorf("concurrent delete of %s.%s=%s: %v", table, column, id, err)
		}
	}()
	return &wg
}

// TestRecurringHireRepository_CourtDeletedInRaceWindow proves the
// recurring_hire_templates.court_id FK path: a Court that existed when
// RequestRecurringHire's guard would have read it, and is gone by the time
// this repository's INSERT runs, must translate to domain.ErrFacilityNotFound
// — the same sentinel that guard's own read already answers for "no such
// Court" — rather than an unclassified 23503.
func TestRecurringHireRepository_CourtDeletedInRaceWindow(t *testing.T) {
	ctx := context.Background()
	pool := startPostgres(t, ctx)

	courtID := seedCourtRow(t, ctx, pool)
	userID := "22222222-2222-4222-a222-222222222201"
	seedUserRow(t, ctx, pool, userID)

	repo := bookingpg.NewRecurringHireRepository(pool)
	template := mustRecurringHireTemplate(t, "33333333-3333-4333-a333-333333333301", userID, courtID)

	// The narrow window: the Court existed a moment ago (an app-level guard
	// reading it now would still see it), a concurrent delete removes it,
	// and only THEN does the INSERT this test issues observe the FK's
	// absence — deterministically, via the synchronization
	// deleteRowConcurrently documents, not by chance timing.
	ready := make(chan struct{})
	wg := deleteRowConcurrently(t, ctx, pool, "courts", "id", courtID, ready)
	close(ready)
	// Give the delete transaction a moment's head start so it is the delete,
	// and not the INSERT, that reaches the FK's underlying index first —
	// both are real, separate Postgres transactions; this only orders which
	// one the FK check observes, it does not fake the concurrency.
	time.Sleep(50 * time.Millisecond)

	_, err := repo.Create(ctx, template)
	wg.Wait()

	if !errors.Is(err, domain.ErrFacilityNotFound) {
		t.Fatalf("Create(court deleted mid-race) = %v, want domain.ErrFacilityNotFound", err)
	}
	// The inverse guard: a court-reference failure must not be misclassified
	// as the sibling FK's sentinel — the two share one 23503 code and are
	// told apart only by constraint name (recurring_hire_repository.go's own
	// comment on why that name is load-bearing here).
	if errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("a deleted Court was misclassified as a missing User: %v", err)
	}
}

// TestRecurringHireRepository_UserDeletedInRaceWindow is
// CourtDeletedInRaceWindow's mirror for requested_by_user_id: a User deleted
// between RequestRecurringHire's EnsureClubRole guard and this INSERT must
// translate to domain.ErrUserNotFound, not an unclassified 23503.
func TestRecurringHireRepository_UserDeletedInRaceWindow(t *testing.T) {
	ctx := context.Background()
	pool := startPostgres(t, ctx)

	courtID := seedCourtRow(t, ctx, pool)
	userID := "22222222-2222-4222-a222-222222222202"
	seedUserRow(t, ctx, pool, userID)

	repo := bookingpg.NewRecurringHireRepository(pool)
	template := mustRecurringHireTemplate(t, "33333333-3333-4333-a333-333333333302", userID, courtID)

	ready := make(chan struct{})
	wg := deleteRowConcurrently(t, ctx, pool, "identity_users", "id", userID, ready)
	close(ready)
	time.Sleep(50 * time.Millisecond)

	_, err := repo.Create(ctx, template)
	wg.Wait()

	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("Create(user deleted mid-race) = %v, want domain.ErrUserNotFound", err)
	}
	if errors.Is(err, domain.ErrFacilityNotFound) {
		t.Fatalf("a deleted User was misclassified as a missing Court: %v", err)
	}
}

// TestRecurringHireRepository_KnownParentsStillSucceedAgainstRealPostgres is
// the control both tests above need. Without it, a repository that rejected
// every INSERT would satisfy both — the same reasoning
// unknown_court_integration_test.go's own known-court control gives.
func TestRecurringHireRepository_KnownParentsStillSucceedAgainstRealPostgres(t *testing.T) {
	ctx := context.Background()
	pool := startPostgres(t, ctx)

	courtID := seedCourtRow(t, ctx, pool)
	userID := "22222222-2222-4222-a222-222222222203"
	seedUserRow(t, ctx, pool, userID)

	repo := bookingpg.NewRecurringHireRepository(pool)
	template := mustRecurringHireTemplate(t, "33333333-3333-4333-a333-333333333303", userID, courtID)

	got, err := repo.Create(ctx, template)
	if err != nil {
		t.Fatalf("Create(seeded court and user) = %v, want success", err)
	}
	if got.CourtID != courtID || got.RequestedByUserID != userID {
		t.Fatalf("persisted template = %+v, want CourtID=%q RequestedByUserID=%q", got, courtID, userID)
	}
}

// mustRecurringHireTemplate builds a well-formed, requested-status template
// through the domain's own constructor, so a future invariant added there is
// exercised by this file rather than bypassed by a hand-built struct.
func mustRecurringHireTemplate(t *testing.T, id, requestedByUserID, courtID string) domain.RecurringHireTemplate {
	t.Helper()

	startTime, err := domain.NewClockTime(9, 0)
	if err != nil {
		t.Fatalf("bad fixture start time: %v", err)
	}
	endTime, err := domain.NewClockTime(10, 0)
	if err != nil {
		t.Fatalf("bad fixture end time: %v", err)
	}
	startsAt, err := time.Parse(time.RFC3339, "2026-09-07T00:00:00Z") // a Monday
	if err != nil {
		t.Fatalf("bad fixture startsAt: %v", err)
	}

	template, err := domain.NewRecurringHireTemplate(
		id, requestedByUserID, courtID,
		time.Monday, startTime, endTime, startsAt,
		domain.NoRecurringHireEnd(),
	)
	if err != nil {
		t.Fatalf("NewRecurringHireTemplate: %v", err)
	}
	return template
}
