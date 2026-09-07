package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nhuthuynh/white-label/internal/booking/app"
	"github.com/nhuthuynh/white-label/internal/booking/domain"
)

// T55.2 (issue #124's court-Bookings half) — CancelBookingsForReference is
// the Booking-side capability a cancelled Game or Competition needs in order
// to release the courts it was holding.
//
// #124's report: a cancelled Game leaves its `game`-source Bookings holding
// the courts, because only a *cancelled* Booking frees a slot. The half of
// that fix which lives in this context is "cancel every active Booking made
// against this reference, and prove the slot is genuinely free afterwards".
//
// This was blocked until DECISION D1 was answered, because the cascade calls
// CancelBooking, whose signature D1 might change. D1 chose option (a), so
// CancelBooking takes an actor and checks ownership — and a game-source
// Booking is owned by the Game's host, who is also the only party allowed to
// cancel the Game. The ownership check therefore passes for the legitimate
// caller without widening anyone's permissions, which is what this file
// pins.

func referenceTestService(repo *inMemoryRepo) *app.Service {
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

// The core of #124: after the cascade, the court is genuinely re-bookable.
// This is the T3 standard the issue's own suggested shape asks for — prove
// the slot is free, not merely that a status field flipped.
func TestCancelBookingsForReference_FreesTheCourts(t *testing.T) {
	t.Parallel()

	svc := referenceTestService(newInMemoryRepo())
	ctx := context.Background()
	morning := mustTimeRange(t, "2026-09-10T09:00:00Z", "2026-09-10T10:00:00Z")
	evening := mustTimeRange(t, "2026-09-10T18:00:00Z", "2026-09-10T19:00:00Z")

	// A Game holding two courts across two slots, exactly as ScheduleGame
	// would leave them.
	for _, tc := range []struct {
		court string
		rng   domain.TimeRange
	}{
		{courtID(1), morning},
		{courtID(2), morning},
		{courtID(1), evening},
	} {
		if _, err := svc.CreateBooking(ctx, app.CreateBookingInput{
			CourtID:     tc.court,
			Source:      domain.SourceGame,
			Range:       tc.rng,
			ReferenceID: "game-1",
			OwnerUserID: ownerA,
		}); err != nil {
			t.Fatalf("seed booking on %s: %v", tc.court, err)
		}
	}

	// A booking belonging to a DIFFERENT game, on a court the cascade must
	// not touch. Without this, a cascade that cancelled everything would
	// pass the assertions below.
	if _, err := svc.CreateBooking(ctx, app.CreateBookingInput{
		CourtID:     courtID(3),
		Source:      domain.SourceGame,
		Range:       morning,
		ReferenceID: "game-2",
		OwnerUserID: ownerA,
	}); err != nil {
		t.Fatalf("seed other game's booking: %v", err)
	}

	n, err := svc.CancelBookingsForReference(ctx, "game-1", ownerA)
	if err != nil {
		t.Fatalf("CancelBookingsForReference: %v", err)
	}
	if n != 3 {
		t.Fatalf("cancelled %d bookings, want 3", n)
	}

	// Every slot the game held is re-bookable — the actual #124 fix.
	for _, tc := range []struct {
		court string
		rng   domain.TimeRange
	}{
		{courtID(1), morning},
		{courtID(2), morning},
		{courtID(1), evening},
	} {
		if _, err := svc.CreateBooking(ctx, app.CreateBookingInput{
			CourtID:     tc.court,
			Source:      domain.SourceIndividual,
			Range:       tc.rng,
			OwnerUserID: ownerB,
		}); err != nil {
			t.Fatalf("re-booking %s after cascade = %v, want the slot to be free", tc.court, err)
		}
	}

	// The other game's court is untouched — still held.
	_, err = svc.CreateBooking(ctx, app.CreateBookingInput{
		CourtID:     courtID(3),
		Source:      domain.SourceIndividual,
		Range:       morning,
		OwnerUserID: ownerB,
	})
	if !errors.Is(err, domain.ErrCourtDoubleBooked) {
		t.Fatalf("other game's court = %v, want %v (the cascade must be scoped to its reference)", err, domain.ErrCourtDoubleBooked)
	}
}

// The cascade is owner-checked like every other cancellation. A caller who
// does not own the bookings gets ErrNotBookingOwner and cancels nothing —
// this method must not become a way around D1's answer.
func TestCancelBookingsForReference_RejectsNonOwner(t *testing.T) {
	t.Parallel()

	svc := referenceTestService(newInMemoryRepo())
	ctx := context.Background()
	rng := mustTimeRange(t, "2026-09-10T09:00:00Z", "2026-09-10T10:00:00Z")

	if _, err := svc.CreateBooking(ctx, app.CreateBookingInput{
		CourtID:     courtID(1),
		Source:      domain.SourceGame,
		Range:       rng,
		ReferenceID: "game-1",
		OwnerUserID: ownerA,
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if _, err := svc.CancelBookingsForReference(ctx, "game-1", ownerB); !errors.Is(err, domain.ErrNotBookingOwner) {
		t.Fatalf("CancelBookingsForReference(stranger) = %v, want %v", err, domain.ErrNotBookingOwner)
	}

	// And nothing was cancelled — the slot is still held. A partial cascade
	// that cancelled some bookings before hitting the one it did not own
	// would fail this.
	_, err := svc.CreateBooking(ctx, app.CreateBookingInput{
		CourtID:     courtID(1),
		Source:      domain.SourceIndividual,
		Range:       rng,
		OwnerUserID: ownerB,
	})
	if !errors.Is(err, domain.ErrCourtDoubleBooked) {
		t.Fatalf("slot after refused cascade = %v, want %v (nothing should have been cancelled)", err, domain.ErrCourtDoubleBooked)
	}
}

// Idempotent: a reference with no active bookings is not an error. This
// matters because CancelGame surfaces a cascade failure to its caller, so a
// second cancel attempt (a retry, a repair sweep) must not look like one.
func TestCancelBookingsForReference_IsIdempotent(t *testing.T) {
	t.Parallel()

	svc := referenceTestService(newInMemoryRepo())
	ctx := context.Background()
	rng := mustTimeRange(t, "2026-09-10T09:00:00Z", "2026-09-10T10:00:00Z")

	if _, err := svc.CreateBooking(ctx, app.CreateBookingInput{
		CourtID:     courtID(1),
		Source:      domain.SourceGame,
		Range:       rng,
		ReferenceID: "game-1",
		OwnerUserID: ownerA,
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if n, err := svc.CancelBookingsForReference(ctx, "game-1", ownerA); err != nil || n != 1 {
		t.Fatalf("first cascade = (%d, %v), want (1, nil)", n, err)
	}

	// Second run: nothing left to do, and that is a success, not a failure.
	n, err := svc.CancelBookingsForReference(ctx, "game-1", ownerA)
	if err != nil {
		t.Fatalf("second cascade = %v, want nil (idempotent)", err)
	}
	if n != 0 {
		t.Fatalf("second cascade cancelled %d, want 0", n)
	}
}

// An unknown reference is likewise a no-op rather than NotFound: Booking has
// no way to tell "a Game that never reserved a court" from "a reference that
// never existed", and both mean the same actionable thing here — there is
// nothing to release.
func TestCancelBookingsForReference_UnknownReferenceIsNoOp(t *testing.T) {
	t.Parallel()

	svc := referenceTestService(newInMemoryRepo())

	n, err := svc.CancelBookingsForReference(context.Background(), "game-nonexistent", ownerA)
	if err != nil {
		t.Fatalf("unknown reference = %v, want nil", err)
	}
	if n != 0 {
		t.Fatalf("unknown reference cancelled %d, want 0", n)
	}
}

// An empty reference must never match. bookings.reference_id is nullable and
// empty for plain individual bookings, so a cascade keyed on "" would cancel
// every unreferenced booking in the system — the worst possible bug in this
// method, and cheap to pin.
func TestCancelBookingsForReference_EmptyReferenceMatchesNothing(t *testing.T) {
	t.Parallel()

	svc := referenceTestService(newInMemoryRepo())
	ctx := context.Background()
	rng := mustTimeRange(t, "2026-09-10T09:00:00Z", "2026-09-10T10:00:00Z")

	// A plain individual booking: no reference at all.
	if _, err := svc.CreateBooking(ctx, app.CreateBookingInput{
		CourtID:     courtID(1),
		Source:      domain.SourceIndividual,
		Range:       rng,
		OwnerUserID: ownerA,
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	n, err := svc.CancelBookingsForReference(ctx, "", ownerA)
	if err != nil {
		t.Fatalf("empty reference = %v, want nil", err)
	}
	if n != 0 {
		t.Fatalf("empty reference cancelled %d bookings, want 0 — it must never match unreferenced bookings", n)
	}

	// Proof the booking really is still held.
	_, err = svc.CreateBooking(ctx, app.CreateBookingInput{
		CourtID:     courtID(1),
		Source:      domain.SourceIndividual,
		Range:       rng,
		OwnerUserID: ownerB,
	})
	if !errors.Is(err, domain.ErrCourtDoubleBooked) {
		t.Fatalf("unreferenced booking after empty-reference cascade = %v, want it untouched", err)
	}
}
