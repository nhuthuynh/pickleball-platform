//go:build integration

// T67.1 (part of issue #353). The half of ListBookingsForActor's contract no
// in-memory fixture can prove, and the reason this file exists beside the
// app- and handler-level tests rather than instead of them.
//
// Three claims this ticket makes are claims about SQL and schema, not about
// Go:
//
//  1. **Cancelled bookings are returned.** Every other list query in
//     db/queries/booking.sql carries `status <> 'cancelled'`;
//     ListBookingsForOwner deliberately does not. A fake repository "proves"
//     whichever behaviour its own loop was written with, so it cannot witness
//     a filter being added to the query.
//  2. **The rows come back ordered by starts_at.** `ORDER BY` is the
//     database's, and a fake that sorts in Go proves only that the fake
//     sorts. Without a real query, a dropped ORDER BY would surface as rows
//     in heap order — which, on a small table, often looks sorted anyway.
//  3. **A Court with facility_id NULL resolves to no Facility rather than
//     failing.** The two Courts 0002_seed.sql creates are inserted as
//     (id, name) only, and 0010_facilities.sql adds facility_id as nullable
//     with no backfill — so on any freshly migrated database both seeded
//     courts have NULL there. That is the common case for this read, and the
//     in-memory fake cannot reproduce a NULL column.
//
// CLAUDE.md's own lesson for this shape: "A DB-level guard with no DB-level
// test is a comment", and "an in-memory fake is more permissive than
// Postgres, and that asymmetry hides storage bugs".
//
// Excluded from `make test-domain`, `make test-adapters` and plain
// `go test ./...` by the integration build tag; compiled (not run) by `make
// vet-integration`; executed by `make ci-integration` or `make test`.
// Requires Docker.
package postgres_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	bookingpg "github.com/nhuthuynh/white-label/internal/booking/adapter/postgres"
	"github.com/nhuthuynh/white-label/internal/booking/domain"
	facilitiespg "github.com/nhuthuynh/white-label/internal/facilities/adapter/postgres"
)

// secondSeedCourtID is the other court 0002_seed.sql creates (seedCourtID,
// in concurrency_integration_test.go, is the first). Both are needed here:
// the ordering claim has to be witnessed across courts, since two bookings on
// one court cannot overlap and a single-court fixture would let a
// chronological insert order masquerade as an ORDER BY.
// Read off db/migrations/0002_seed.sql rather than guessed from the first
// one's shape — the first draft of this file assumed ...111112 and the seed
// really says ...222222, which a Docker-free gate could never have caught.
const secondSeedCourtID = "22222222-2222-2222-2222-222222222222"

// TestListForOwner_AgainstRealPostgres drives the real repository against the
// real schema. It asserts all three claims above in one container, because
// starting Postgres is the expensive part and the claims share a fixture.
func TestListForOwner_AgainstRealPostgres(t *testing.T) {
	ctx := context.Background()
	pool := startPostgres(t, ctx)
	repo := bookingpg.NewRepository(pool)

	owner := seedBookingOwner(t, ctx, pool)
	stranger := seedBookingOwner(t, ctx, pool)

	// Inserted deliberately out of chronological order, and across both
	// seeded courts, so a passing ordering assertion cannot be an artifact of
	// insertion order.
	third := createBookingRow(t, ctx, repo, seedCourtID, owner, "2026-04-03T09:00:00Z", "2026-04-03T10:00:00Z")
	first := createBookingRow(t, ctx, repo, secondSeedCourtID, owner, "2026-04-01T09:00:00Z", "2026-04-01T10:00:00Z")
	second := createBookingRow(t, ctx, repo, seedCourtID, owner, "2026-04-02T09:00:00Z", "2026-04-02T10:00:00Z")

	// The stranger's booking sits in the same table, on the same court, in
	// the middle of the owner's date range.
	strangers := createBookingRow(t, ctx, repo, secondSeedCourtID, stranger, "2026-04-02T09:00:00Z", "2026-04-02T10:00:00Z")

	// Cancel one of the owner's, through the real UPDATE path.
	cancelled := second
	cancelled.Status = domain.StatusCancelled
	if _, err := repo.Update(ctx, cancelled); err != nil {
		t.Fatalf("Update(cancel): %v", err)
	}

	got, err := repo.ListForOwner(ctx, owner)
	if err != nil {
		t.Fatalf("ListForOwner: %v", err)
	}

	// Claim 1 + the owner scope: three rows, the cancelled one among them,
	// the stranger's absent.
	if len(got) != 3 {
		t.Fatalf("got %d bookings, want 3 (all of this owner's, cancelled included): %+v", len(got), got)
	}
	for _, b := range got {
		if b.ID == strangers.ID {
			t.Errorf("another user's booking %q came back; the query is not scoped to owner_user_id", b.ID)
		}
		if b.OwnerUserID != owner {
			t.Errorf("booking %q is owned by %q, want %q", b.ID, b.OwnerUserID, owner)
		}
	}

	// Claim 2: the order is the database's.
	wantOrder := []string{first.ID, second.ID, third.ID}
	for i, id := range wantOrder {
		if got[i].ID != id {
			t.Fatalf("position %d = %q, want %q — the rows are not in ORDER BY starts_at order",
				i, got[i].ID, id)
		}
	}

	// Claim 1, stated as its own assertion: the cancelled row is present AND
	// reported as cancelled. A query that returned it with a stale status
	// would be a different bug with the same row count.
	var sawCancelled bool
	for _, b := range got {
		if b.ID != second.ID {
			continue
		}
		sawCancelled = true
		if b.Status != domain.StatusCancelled {
			t.Errorf("booking %q status = %q, want %q", b.ID, b.Status, domain.StatusCancelled)
		}
	}
	if !sawCancelled {
		t.Errorf("the cancelled booking %q is missing. db/queries/booking.sql's ListBookingsForOwner "+
			"must NOT filter on status: a player who cannot see a booking they cancelled cannot tell "+
			"it from one that was lost, which is the bug #353 reports", second.ID)
	}

	// An owner with nothing is an empty list, not an error, against the real
	// query too.
	empty, err := repo.ListForOwner(ctx, seedBookingOwner(t, ctx, pool))
	if err != nil {
		t.Fatalf("ListForOwner(owner with no bookings): %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("got %+v, want an empty list", empty)
	}

	// A well-formed uuid belonging to no identity_users row: there is no FK
	// to violate on a SELECT, so this must simply be empty rather than an
	// error — and must not be distinguishable from "that user has no
	// bookings", which would make the read an enumeration oracle.
	unknown, err := repo.ListForOwner(ctx, uuid.NewString())
	if err != nil {
		t.Fatalf("ListForOwner(unknown owner): %v", err)
	}
	if len(unknown) != 0 {
		t.Fatalf("got %+v, want an empty list", unknown)
	}
}

// TestGetCourtsByIDs_AgainstRealPostgres is claim 3: the batch court read
// T67.1 added to the Facilities context, run against the real `courts` table
// including its NULL facility_id rows.
//
// It also pins the schema fact the whole "honest empty facility id" design
// rests on, rather than trusting the migration headers: **both courts seeded
// by 0002_seed.sql have facility_id NULL.** If a future migration backfilled
// them, this assertion is where that becomes visible, instead of in a
// surprised reading of ListBookingsForActor's output.
func TestGetCourtsByIDs_AgainstRealPostgres(t *testing.T) {
	ctx := context.Background()
	pool := startPostgres(t, ctx)
	repo := facilitiespg.NewRepository(pool)

	courts, err := repo.GetCourtsByIDs(ctx, []string{seedCourtID, secondSeedCourtID})
	if err != nil {
		t.Fatalf("GetCourtsByIDs: %v", err)
	}
	if len(courts) != 2 {
		t.Fatalf("got %d courts, want the 2 seeded by 0002_seed.sql: %+v", len(courts), courts)
	}
	for _, c := range courts {
		if c.FacilityID != "" {
			t.Errorf("seeded court %q has FacilityID %q, want empty — 0002_seed.sql inserts "+
				"(id, name) only and 0010_facilities.sql adds facility_id as nullable with no "+
				"backfill, so this read's empty-facility path is the common case on a fresh "+
				"database. If a migration now backfills these, ListBookingsForActor's "+
				"empty-facility behaviour needs a different fixture to stay covered.", c.ID, c.FacilityID)
		}
	}

	// A miss contributes no row rather than failing the call — the property
	// that lets a Booking on an unresolvable court still be listed.
	mixed, err := repo.GetCourtsByIDs(ctx, []string{seedCourtID, uuid.NewString()})
	if err != nil {
		t.Fatalf("GetCourtsByIDs(one unknown): %v, want nil — one miss must not fail the batch", err)
	}
	if len(mixed) != 1 || mixed[0].ID != seedCourtID {
		t.Fatalf("got %+v, want only the known court", mixed)
	}

	// An empty array is a legal query, not a syntax error: `= ANY('{}')`
	// matches nothing. The app layer short-circuits before reaching here, so
	// this asserts the adapter is safe if that guard is ever moved.
	none, err := repo.GetCourtsByIDs(ctx, nil)
	if err != nil {
		t.Fatalf("GetCourtsByIDs(nil): %v", err)
	}
	if len(none) != 0 {
		t.Fatalf("got %+v, want empty", none)
	}
}

// createBookingRow persists one booking through the real repository, minting
// the id the way production does.
func createBookingRow(t *testing.T, ctx context.Context, repo *bookingpg.Repository, courtID, owner, from, to string) domain.Booking {
	t.Helper()

	b, err := domain.NewBooking(uuid.NewString(), courtID, domain.SourceIndividual,
		mustRange(t, from, to), "", owner)
	if err != nil {
		t.Fatalf("building fixture booking: %v", err)
	}
	stored, err := repo.Create(ctx, b)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	return stored
}
