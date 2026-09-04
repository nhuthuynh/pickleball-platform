package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nhuthuynh/white-label/internal/booking/app"
	"github.com/nhuthuynh/white-label/internal/booking/domain"
)

// T55.1 (DECISION D1, ADR-0015 option (a); closes #144) — the app-level
// proof that a Booking is owned and that only its owner can cancel it.
//
// #144's report, in its own words, was: "anyone holding a booking id can
// cancel it". The test that actually closes it is therefore not "the owner
// can cancel" (a happy path that passed before this ticket for the wrong
// reason — nobody was checked at all) but "a stranger holding the real
// booking id cannot", which is what TestCancelBooking_RejectsNonOwner
// asserts against a genuinely valid id.

const (
	ownerA   = "11111111-1111-1111-1111-111111111111"
	ownerB   = "33333333-3333-3333-3333-333333333333"
	ownerNil = ""
)

func ownershipService(repo *inMemoryRepo) *app.Service {
	return app.NewService(app.ServiceOptions{
		Bookings:       repo,
		PricingRules:   &fakePricingRepo{},
		DiscountRules:  newFakeDiscountRepo(),
		RecurringHires: newFakeRecurringHireRepo(),
		Facilities:     &fakeFacilityLookup{},
		Identity:       &fakeIdentityLookup{},
		IDs:            &sequentialIDs{},
	})
}

// CreateBooking must refuse to mint an unowned Booking. Before D1 this was
// the normal case; after it, it is an error at the app boundary, not merely
// a NOT NULL violation discovered at the database.
func TestCreateBooking_RequiresOwner(t *testing.T) {
	t.Parallel()

	svc := ownershipService(newInMemoryRepo())
	ctx := context.Background()
	rng := mustTimeRange(t, "2026-09-04T09:00:00Z", "2026-09-04T10:00:00Z")

	_, err := svc.CreateBooking(ctx, app.CreateBookingInput{
		CourtID:     courtID(1),
		Source:      domain.SourceIndividual,
		Range:       rng,
		OwnerUserID: ownerNil,
	})
	if !errors.Is(err, domain.ErrEmptyOwnerUserID) {
		t.Fatalf("CreateBooking with no owner = %v, want %v", err, domain.ErrEmptyOwnerUserID)
	}
}

// The booking records the owner it was created with, and hands it back.
func TestCreateBooking_RecordsOwner(t *testing.T) {
	t.Parallel()

	svc := ownershipService(newInMemoryRepo())
	ctx := context.Background()
	rng := mustTimeRange(t, "2026-09-04T09:00:00Z", "2026-09-04T10:00:00Z")

	b, err := svc.CreateBooking(ctx, app.CreateBookingInput{
		CourtID:     courtID(1),
		Source:      domain.SourceIndividual,
		Range:       rng,
		OwnerUserID: ownerA,
	})
	if err != nil {
		t.Fatalf("CreateBooking: %v", err)
	}
	if b.OwnerUserID != ownerA {
		t.Fatalf("OwnerUserID = %q, want %q", b.OwnerUserID, ownerA)
	}
}

// The test that closes #144.
func TestCancelBooking_RejectsNonOwner(t *testing.T) {
	t.Parallel()

	svc := ownershipService(newInMemoryRepo())
	ctx := context.Background()
	rng := mustTimeRange(t, "2026-09-04T09:00:00Z", "2026-09-04T10:00:00Z")

	created, err := svc.CreateBooking(ctx, app.CreateBookingInput{
		CourtID:     courtID(1),
		Source:      domain.SourceIndividual,
		Range:       rng,
		OwnerUserID: ownerA,
	})
	if err != nil {
		t.Fatalf("CreateBooking: %v", err)
	}

	tests := []struct {
		name  string
		actor string
		want  error
	}{
		{
			// The #144 scenario exactly: a real, valid booking id in the
			// hands of somebody who is not its owner.
			name:  "a stranger holding the real booking id is refused",
			actor: ownerB,
			want:  domain.ErrNotBookingOwner,
		},
		{
			name:  "an unauthenticated caller is refused",
			actor: ownerNil,
			want:  domain.ErrNotBookingOwner,
		},
		{
			name:  "the owner succeeds",
			actor: ownerA,
			want:  nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.CancelBooking(ctx, created.ID, tc.actor)
			if !errors.Is(err, tc.want) {
				t.Fatalf("CancelBooking(actor=%q) = %v, want %v", tc.actor, err, tc.want)
			}
		})
	}
}

// A refused cancellation must not have cancelled anything. Asserting only
// the returned error would pass even if the service cancelled the booking
// and then reported a permission failure, so the slot is re-checked: it is
// still held, which is only true if the booking is still confirmed.
func TestCancelBooking_RefusedCancellationLeavesBookingConfirmed(t *testing.T) {
	t.Parallel()

	svc := ownershipService(newInMemoryRepo())
	ctx := context.Background()
	rng := mustTimeRange(t, "2026-09-04T09:00:00Z", "2026-09-04T10:00:00Z")

	created, err := svc.CreateBooking(ctx, app.CreateBookingInput{
		CourtID:     courtID(1),
		Source:      domain.SourceIndividual,
		Range:       rng,
		OwnerUserID: ownerA,
	})
	if err != nil {
		t.Fatalf("CreateBooking: %v", err)
	}

	if _, err := svc.CancelBooking(ctx, created.ID, ownerB); !errors.Is(err, domain.ErrNotBookingOwner) {
		t.Fatalf("CancelBooking by stranger = %v, want %v", err, domain.ErrNotBookingOwner)
	}

	// The slot must still be held — the strongest available proof that the
	// refused cancel did not persist, since a cancelled booking frees its
	// slot (T3).
	_, err = svc.CreateBooking(ctx, app.CreateBookingInput{
		CourtID:     courtID(1),
		Source:      domain.SourceGame,
		Range:       rng,
		ReferenceID: "game-1",
		OwnerUserID: ownerB,
	})
	if !errors.Is(err, domain.ErrCourtDoubleBooked) {
		t.Fatalf("slot after refused cancel = %v, want %v (booking must still be confirmed)", err, domain.ErrCourtDoubleBooked)
	}
}

// An unknown booking id must answer NotFound for everyone, including a
// caller who owns no bookings at all — the ownership check must not run
// before the existence check in a way that turns a miss into a permission
// answer, nor vice versa. Ordering matters: GetByID first (NotFound), then
// EnsureOwner (PermissionDenied).
func TestCancelBooking_UnknownIDIsNotFoundNotPermissionDenied(t *testing.T) {
	t.Parallel()

	svc := ownershipService(newInMemoryRepo())
	ctx := context.Background()

	_, err := svc.CancelBooking(ctx, "44444444-4444-4444-4444-444444444444", ownerA)
	if !errors.Is(err, domain.ErrBookingNotFound) {
		t.Fatalf("CancelBooking(unknown id) = %v, want %v", err, domain.ErrBookingNotFound)
	}
}
