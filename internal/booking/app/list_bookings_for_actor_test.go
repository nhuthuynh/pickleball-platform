package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nhuthuynh/white-label/internal/booking/app"
	"github.com/nhuthuynh/white-label/internal/booking/domain"
)

// --- ListBookingsForActor (T67.1, part of issue #353) ------------------
//
// The player-facing counterpart to ListCourtBookings: "which courts have I
// booked?" rather than "who has booked this court?". Issue #353 is that
// nothing a player books or joins is visible afterwards — a player could
// create a Booking, reload, and be told nothing at all, because no proto in
// this repository had a player-scoped read of any kind.
//
// Three properties below are the ones worth stating in a test rather than
// leaving to a reader of the SQL, because each is a deliberate choice that
// the obvious implementation gets wrong:
//
//   - **Cancelled bookings are INCLUDED.** This is the exact opposite of
//     port.Repository.ListActiveForCourt, which exists to answer an
//     availability question and so must exclude them. See the app method's
//     own doc comment for why: a list that drops cancelled rows cannot
//     distinguish "I cancelled that" from "my booking was lost".
//   - **The actor is the scope, and comes from nowhere but the token.** The
//     app method takes it as a parameter; what makes that safe is the
//     handler funnel (grpcapi.Handler.actor), pinned by this ticket's
//     handler tests. Here the property under test is that the method filters
//     on it at all.
//   - **A Court belonging to no Facility is an honest empty FacilityID, not
//     an error.** courts.facility_id is nullable and both courts seeded by
//     0002_seed.sql still have NULL there, so this is the common case on a
//     fresh database, not a corner.
//
// Ordering is asserted here too, but note what that does and does not prove:
// the app layer must not re-order what the repository returns, and
// inMemoryRepo.ListForOwner mirrors the real query's ORDER BY. That the
// *SQL* orders correctly is a different claim, pinned by
// internal/booking/adapter/postgres's integration test, not by this file.

// actorBookingFixture builds a Service whose repository already holds
// bookings, plus a court->facility map, and hands back the facility fake so a
// test can count how many times the batch lookup was called.
func actorBookingFixture(t *testing.T) (*app.Service, *inMemoryRepo, *fakeFacilityLookup) {
	t.Helper()

	repo := newInMemoryRepo()
	lookup := &fakeFacilityLookup{
		ownerByFacility: map[string]string{
			facilityID(1): userID(ownerUser),
			facilityID(2): userID(ownerUser),
		},
		facilityByCourt: map[string]string{
			courtID(1): facilityID(1),
			courtID(2): facilityID(2),
			// courtID(3) is deliberately absent: a Court belonging to no
			// Facility, the shape both courts of 0002_seed.sql have.
		},
	}
	svc := app.NewService(app.ServiceOptions{
		Bookings:       repo,
		PricingRules:   &fakePricingRepo{},
		DiscountRules:  newFakeDiscountRepo(),
		RecurringHires: newFakeRecurringHireRepo(),
		Facilities:     lookup,
		Identity:       &fakeIdentityLookup{},
		IDs:            &sequentialIDs{},
	})
	return svc, repo, lookup
}

// seedBooking puts a booking straight into the repository, bypassing
// CreateBooking. Deliberate: CreateBooking enforces the no-double-booking
// invariant, and several cases below need bookings that overlap in time (on
// different courts, or a cancelled one on top of a live one) to exercise the
// read. The read under test applies no such rule.
func seedBooking(t *testing.T, repo *inMemoryRepo, id, court, owner string, status domain.Status, from, to string) domain.Booking {
	t.Helper()
	b := domain.Booking{
		ID:          id,
		CourtID:     court,
		Source:      domain.SourceIndividual,
		Status:      status,
		Range:       mustTimeRange(t, from, to),
		OwnerUserID: owner,
	}
	repo.bookings[b.ID] = b
	return b
}

func TestListBookingsForActor(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	const (
		me        = 40
		otherUser = 41
	)

	t.Run("returns only the actor's own bookings, ordered by start", func(t *testing.T) {
		t.Parallel()
		svc, repo, _ := actorBookingFixture(t)

		later := seedBooking(t, repo, seqID(2), courtID(1), userID(me), domain.StatusConfirmed,
			"2026-03-02T10:00:00Z", "2026-03-02T11:00:00Z")
		earlier := seedBooking(t, repo, seqID(1), courtID(2), userID(me), domain.StatusConfirmed,
			"2026-03-01T09:00:00Z", "2026-03-01T10:00:00Z")
		// Same court, same time as `later`, a different owner: proves the
		// filter is the owner and not anything about the slot.
		seedBooking(t, repo, seqID(3), courtID(1), userID(otherUser), domain.StatusCancelled,
			"2026-03-02T10:00:00Z", "2026-03-02T11:00:00Z")

		got, err := svc.ListBookingsForActor(ctx, userID(me))
		if err != nil {
			t.Fatalf("ListBookingsForActor: %v", err)
		}

		if len(got) != 2 {
			t.Fatalf("got %d bookings, want 2 (the actor's own): %+v", len(got), got)
		}
		if got[0].Booking.ID != earlier.ID || got[1].Booking.ID != later.ID {
			t.Errorf("order = [%s %s], want [%s %s] (ascending starts_at)",
				got[0].Booking.ID, got[1].Booking.ID, earlier.ID, later.ID)
		}
		for _, row := range got {
			if row.Booking.OwnerUserID != userID(me) {
				t.Errorf("booking %s is owned by %s, not by the actor %s — the read leaked another user's row",
					row.Booking.ID, row.Booking.OwnerUserID, userID(me))
			}
		}
	})

	// The deliberate inversion of ListActiveForCourt. If this ever starts
	// failing because someone "fixed" the read to hide cancelled rows, the
	// user-visible bug is that a cancelled booking is indistinguishable from
	// one that was silently lost.
	t.Run("includes cancelled bookings", func(t *testing.T) {
		t.Parallel()
		svc, repo, _ := actorBookingFixture(t)

		cancelled := seedBooking(t, repo, seqID(1), courtID(1), userID(me), domain.StatusCancelled,
			"2026-03-01T09:00:00Z", "2026-03-01T10:00:00Z")

		got, err := svc.ListBookingsForActor(ctx, userID(me))
		if err != nil {
			t.Fatalf("ListBookingsForActor: %v", err)
		}
		if len(got) != 1 || got[0].Booking.ID != cancelled.ID {
			t.Fatalf("got %+v, want the cancelled booking %q — a cancelled booking a user cannot see "+
				"is indistinguishable from one that was lost", got, cancelled.ID)
		}
		if got[0].Booking.Status != domain.StatusCancelled {
			t.Errorf("status = %q, want %q reported as itself", got[0].Booking.Status, domain.StatusCancelled)
		}
	})

	t.Run("each row carries its court's facility id, and NULL is an honest empty", func(t *testing.T) {
		t.Parallel()
		svc, repo, _ := actorBookingFixture(t)

		seedBooking(t, repo, seqID(1), courtID(1), userID(me), domain.StatusConfirmed,
			"2026-03-01T09:00:00Z", "2026-03-01T10:00:00Z")
		seedBooking(t, repo, seqID(2), courtID(2), userID(me), domain.StatusConfirmed,
			"2026-03-02T09:00:00Z", "2026-03-02T10:00:00Z")
		// courtID(3) belongs to no Facility — the 0002_seed.sql shape.
		seedBooking(t, repo, seqID(3), courtID(3), userID(me), domain.StatusConfirmed,
			"2026-03-03T09:00:00Z", "2026-03-03T10:00:00Z")

		got, err := svc.ListBookingsForActor(ctx, userID(me))
		if err != nil {
			t.Fatalf("ListBookingsForActor: %v — a Court with no Facility must not fail the whole list", err)
		}
		if len(got) != 3 {
			t.Fatalf("got %d bookings, want 3", len(got))
		}

		want := map[string]string{
			courtID(1): facilityID(1),
			courtID(2): facilityID(2),
			courtID(3): "", // honest empty, not an error and not a guess
		}
		for _, row := range got {
			if row.FacilityID != want[row.Booking.CourtID] {
				t.Errorf("booking on court %s: FacilityID = %q, want %q",
					row.Booking.CourtID, row.FacilityID, want[row.Booking.CourtID])
			}
		}
	})

	// The reason port.FacilityLookup gained a batch method rather than having
	// the app loop over FacilityIDForCourt: one lookup for the whole page, no
	// matter how many bookings or how many distinct courts.
	t.Run("resolves facilities in one batch lookup, not one per booking", func(t *testing.T) {
		t.Parallel()
		svc, repo, lookup := actorBookingFixture(t)

		for i, court := range []string{courtID(1), courtID(2), courtID(1), courtID(2), courtID(1)} {
			seedBooking(t, repo, seqID(i+1), court, userID(me), domain.StatusConfirmed,
				"2026-03-01T09:00:00Z", "2026-03-01T10:00:00Z")
		}

		if _, err := svc.ListBookingsForActor(ctx, userID(me)); err != nil {
			t.Fatalf("ListBookingsForActor: %v", err)
		}

		if n := lookup.facilityIDsForCourtsCalls.Load(); n != 1 {
			t.Errorf("FacilityIDsForCourts called %d times for 5 bookings, want exactly 1 — "+
				"a per-booking lookup is the N+1 the batch port exists to prevent", n)
		}
		if n := lookup.facilityIDsForCourtsArgLen.Load(); n != 2 {
			t.Errorf("batch lookup was asked for %d court ids, want 2 distinct — "+
				"the 5 bookings sit on 2 courts", n)
		}
	})

	t.Run("an actor with no bookings gets an empty list and no error", func(t *testing.T) {
		t.Parallel()
		svc, repo, lookup := actorBookingFixture(t)
		seedBooking(t, repo, seqID(1), courtID(1), userID(otherUser), domain.StatusConfirmed,
			"2026-03-01T09:00:00Z", "2026-03-01T10:00:00Z")

		got, err := svc.ListBookingsForActor(ctx, userID(me))
		if err != nil {
			t.Fatalf("ListBookingsForActor: %v, want nil", err)
		}
		if len(got) != 0 {
			t.Fatalf("got %+v, want an empty list", got)
		}
		if got == nil {
			t.Error("got a nil slice; the other list reads in this package return an " +
				"allocated empty slice so a JSON encoder emits [] rather than null")
		}
		if n := lookup.facilityIDsForCourtsCalls.Load(); n != 0 {
			t.Errorf("FacilityIDsForCourts called %d times with no bookings to resolve, want 0", n)
		}
	})

	// Same T10.7 guard every other caller-supplied id in this package carries:
	// the Postgres adapter's mustUUID panics on anything pgtype.UUID.Scan
	// rejects, and bookings.owner_user_id is a `uuid` column. A malformed
	// actor is answered exactly like an unknown one, which for a list read
	// means an empty list rather than an error this method never otherwise
	// returns — ListCourtBookings' identical convention.
	//
	// Post-ADR-0014 a caller whose subject resolves to no User never reaches
	// this method at all (the handler's actor() funnel refuses them), so this
	// branch is reachable only by a programming error. The guard still has to
	// exist, because what it prevents is a process-wide panic.
	t.Run("a malformed actor id is an empty list that never reaches the repository", func(t *testing.T) {
		t.Parallel()

		for _, malformed := range []string{
			"",
			"not-a-uuid",
			"00000000-0000-4000-b000",
			"{00000000-0000-4000-b000-000000000040}",
			"urn:uuid:00000000-0000-4000-b000-000000000040",
		} {
			svc, repo, lookup := actorBookingFixture(t)
			seedBooking(t, repo, seqID(1), courtID(1), userID(me), domain.StatusConfirmed,
				"2026-03-01T09:00:00Z", "2026-03-01T10:00:00Z")

			got, err := svc.ListBookingsForActor(ctx, malformed)
			if err != nil {
				t.Fatalf("ListBookingsForActor(%q) err = %v, want nil", malformed, err)
			}
			if len(got) != 0 {
				t.Fatalf("ListBookingsForActor(%q) = %+v, want an empty list", malformed, got)
			}
			if n := repo.listForOwnerCalls.Load(); n != 0 {
				t.Errorf("ListBookingsForActor(%q) reached the repository %d times; the shape guard "+
					"exists because the Postgres adapter's mustUUID panics on this input", malformed, n)
			}
			if n := lookup.facilityIDsForCourtsCalls.Load(); n != 0 {
				t.Errorf("ListBookingsForActor(%q) reached the facility lookup %d times, want 0", malformed, n)
			}
		}
	})

	// A failure resolving facilities is a real failure, not a reason to serve
	// a half-populated page: the client uses facility_id to show a court's
	// NAME (issue #342, closed), so silently blanking it would reintroduce
	// the uuid-on-screen bug on a new screen. "This court has no Facility"
	// (the subtest above) and "I could not reach Facilities" are different
	// answers and only the first is an empty FacilityID.
	t.Run("a facility lookup failure is reported, not papered over", func(t *testing.T) {
		t.Parallel()
		svc, repo, lookup := actorBookingFixture(t)
		seedBooking(t, repo, seqID(1), courtID(1), userID(me), domain.StatusConfirmed,
			"2026-03-01T09:00:00Z", "2026-03-01T10:00:00Z")

		boom := errors.New("facilities unreachable")
		lookup.batchErr = boom

		if _, err := svc.ListBookingsForActor(ctx, userID(me)); !errors.Is(err, boom) {
			t.Fatalf("err = %v, want %v", err, boom)
		}
	})
}
