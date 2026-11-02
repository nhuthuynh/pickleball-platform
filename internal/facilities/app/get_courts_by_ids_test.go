package app_test

import (
	"context"
	"testing"

	"github.com/nhuthuynh/white-label/internal/facilities/app"
)

// TestGetCourtsByIDs covers T67.1's batch read — the counterpart to GetCourt,
// added so another bounded context can resolve many Courts to their owning
// Facilities in one call instead of N.
//
// Its caller is Booking's player-facing "my bookings" read, through
// port.FacilityLookup.FacilityIDsForCourts: a player's bookings span
// arbitrarily many courts, each of which the client needs resolved to show a
// court's NAME rather than its uuid (issue #342). Resolving them one at a time
// would be an N+1 whose N is set by how much the player has booked.
//
// Two properties here are deliberately NOT GetCourt's, and each has a subtest:
//
//   - **A miss is absence, not domain.ErrCourtNotFound.** GetCourt fails for
//     an unknown id because its caller wants one specific Court. A batch read
//     that failed on one miss would let a single deleted court hide every
//     other court's answer from its caller.
//   - **Malformed ids are filtered out rather than rejected.** This is the
//     same T10.7 guard GetCourt applies (a malformed id is answered exactly
//     like an unknown one), translated into the batch world where "unknown"
//     means "absent". It is load-bearing, not cosmetic: the Postgres
//     adapter's mustUUID panics on anything pgtype.UUID.Scan rejects, and the
//     in-memory fake cannot reproduce that, so the guard is proved by the
//     repository call never happening.
func TestGetCourtsByIDs(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := newInMemoryRepo()
	svc := app.NewService(repo, stubIdentity{}, &sequentialIDs{})

	owner := seqUUID(910)
	f, err := svc.CreateFacility(ctx, app.CreateFacilityInput{
		OwnerID: owner, Name: "Lakeside Courts", Address: "9 Lake Rd",
	})
	if err != nil {
		t.Fatalf("CreateFacility: %v", err)
	}
	first, err := svc.AddCourt(ctx, f.ID, owner, "Court 1")
	if err != nil {
		t.Fatalf("AddCourt: %v", err)
	}
	second, err := svc.AddCourt(ctx, f.ID, owner, "Court 2")
	if err != nil {
		t.Fatalf("AddCourt: %v", err)
	}

	t.Run("resolves every known court in one repository call", func(t *testing.T) {
		before := repo.getCourtsByIDsCalls.Load()

		got, err := svc.GetCourtsByIDs(ctx, []string{first.ID, second.ID})
		if err != nil {
			t.Fatalf("GetCourtsByIDs: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("got %d courts, want 2: %+v", len(got), got)
		}
		for _, c := range got {
			if c.FacilityID != f.ID {
				t.Errorf("court %s: FacilityID = %q, want %q", c.ID, c.FacilityID, f.ID)
			}
		}
		if n := repo.getCourtsByIDsCalls.Load() - before; n != 1 {
			t.Errorf("the repository was called %d times, want exactly 1 — resolving a set of "+
				"courts one at a time is the N+1 this method exists to replace", n)
		}
	})

	t.Run("an unknown id contributes no row rather than failing the call", func(t *testing.T) {
		got, err := svc.GetCourtsByIDs(ctx, []string{first.ID, seqUUID(4343)})
		if err != nil {
			t.Fatalf("GetCourtsByIDs: %v, want nil — one miss must not hide the hits", err)
		}
		if len(got) != 1 || got[0].ID != first.ID {
			t.Fatalf("got %+v, want only the known court %q", got, first.ID)
		}
	})

	// The guard: a malformed id must never reach the repository, where
	// mustUUID would panic and take the process down with it.
	t.Run("malformed ids are dropped before the repository", func(t *testing.T) {
		malformed := []string{
			"",
			"not-a-uuid",
			"3f2504e0-4f89-11d3-9a0c",
			"{3f2504e0-4f89-11d3-9a0c-0305e82c3301}",
			"urn:uuid:3f2504e0-4f89-11d3-9a0c-0305e82c3301",
		}

		t.Run("alongside a well-formed one", func(t *testing.T) {
			got, err := svc.GetCourtsByIDs(ctx, append(malformed, first.ID))
			if err != nil {
				t.Fatalf("GetCourtsByIDs: %v", err)
			}
			if len(got) != 1 || got[0].ID != first.ID {
				t.Fatalf("got %+v, want only the well-formed court %q", got, first.ID)
			}
			if arg := repo.lastGetCourtsByIDsArg; len(arg) != 1 || arg[0] != first.ID {
				t.Errorf("the repository was asked for %v; only the well-formed id may reach it, "+
					"because the Postgres adapter's mustUUID panics on the others", arg)
			}
		})

		t.Run("and on their own, without touching the repository", func(t *testing.T) {
			before := repo.getCourtsByIDsCalls.Load()

			got, err := svc.GetCourtsByIDs(ctx, malformed)
			if err != nil {
				t.Fatalf("GetCourtsByIDs: %v, want nil", err)
			}
			if len(got) != 0 {
				t.Fatalf("got %+v, want empty", got)
			}
			if n := repo.getCourtsByIDsCalls.Load() - before; n != 0 {
				t.Errorf("the repository was called %d times with nothing well-formed to ask for, want 0", n)
			}
		})
	})

	t.Run("no ids is an empty result and no repository call", func(t *testing.T) {
		for _, ids := range [][]string{nil, {}} {
			before := repo.getCourtsByIDsCalls.Load()

			got, err := svc.GetCourtsByIDs(ctx, ids)
			if err != nil {
				t.Fatalf("GetCourtsByIDs(%v): %v, want nil", ids, err)
			}
			if len(got) != 0 {
				t.Fatalf("GetCourtsByIDs(%v) = %+v, want empty", ids, got)
			}
			if n := repo.getCourtsByIDsCalls.Load() - before; n != 0 {
				t.Errorf("GetCourtsByIDs(%v) called the repository %d times, want 0", ids, n)
			}
		}
	})
}
