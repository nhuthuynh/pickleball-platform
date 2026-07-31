package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nhuthuynh/white-label/internal/booking/app"
	"github.com/nhuthuynh/white-label/internal/booking/domain"
)

// TestCancelBooking_FreesTheSlotForRebooking is the app-level proof
// (HANDOFF.md T3 AC: "cancelling then re-booking the same slot succeeds")
// that a cancelled booking no longer holds the no-double-booking invariant
// — CreateBooking's pre-check (domain.EnsureNoConflict) already ignores
// cancelled bookings (proven at the domain level in T0); this test proves
// the wiring that gets a booking into that state actually works end to end.
func TestCancelBooking_FreesTheSlotForRebooking(t *testing.T) {
	t.Parallel()

	repo := newInMemoryRepo()
	svc := app.NewService(repo, &fakePricingRepo{}, &sequentialIDs{})
	ctx := context.Background()
	rng := mustTimeRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")

	first, err := svc.CreateBooking(ctx, app.CreateBookingInput{CourtID: "court-1", Source: domain.SourceIndividual, Range: rng})
	if err != nil {
		t.Fatalf("unexpected err creating first booking: %v", err)
	}

	// Before cancelling, the slot is genuinely taken.
	_, err = svc.CreateBooking(ctx, app.CreateBookingInput{CourtID: "court-1", Source: domain.SourceGame, Range: rng, ReferenceID: "game-1"})
	if !errors.Is(err, domain.ErrCourtDoubleBooked) {
		t.Fatalf("got err %v, want %v (slot should still be taken)", err, domain.ErrCourtDoubleBooked)
	}

	cancelled, err := svc.CancelBooking(ctx, first.ID)
	if err != nil {
		t.Fatalf("unexpected err cancelling: %v", err)
	}
	if cancelled.Status != domain.StatusCancelled {
		t.Fatalf("status = %v, want cancelled", cancelled.Status)
	}

	// After cancelling, the same slot can be re-booked.
	rebooked, err := svc.CreateBooking(ctx, app.CreateBookingInput{CourtID: "court-1", Source: domain.SourceGame, Range: rng, ReferenceID: "game-1"})
	if err != nil {
		t.Fatalf("re-booking the freed slot should succeed, got %v", err)
	}
	if rebooked.Status != domain.StatusConfirmed {
		t.Fatalf("rebooked status = %v, want confirmed", rebooked.Status)
	}
}

func TestCancelBooking_UnknownBookingReturnsNotFound(t *testing.T) {
	t.Parallel()

	repo := newInMemoryRepo()
	svc := app.NewService(repo, &fakePricingRepo{}, &sequentialIDs{})
	ctx := context.Background()

	_, err := svc.CancelBooking(ctx, "does-not-exist")
	if !errors.Is(err, domain.ErrBookingNotFound) {
		t.Fatalf("got err %v, want %v", err, domain.ErrBookingNotFound)
	}
}

func TestCancelBooking_AlreadyCancelledIsRejected(t *testing.T) {
	t.Parallel()

	repo := newInMemoryRepo()
	svc := app.NewService(repo, &fakePricingRepo{}, &sequentialIDs{})
	ctx := context.Background()
	rng := mustTimeRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")

	b, err := svc.CreateBooking(ctx, app.CreateBookingInput{CourtID: "court-1", Source: domain.SourceIndividual, Range: rng})
	if err != nil {
		t.Fatalf("unexpected err creating fixture: %v", err)
	}
	if _, err := svc.CancelBooking(ctx, b.ID); err != nil {
		t.Fatalf("first cancel should succeed, got %v", err)
	}

	if _, err := svc.CancelBooking(ctx, b.ID); !errors.Is(err, domain.ErrIllegalStatusTransition) {
		t.Fatalf("double-cancel should be rejected, got %v", err)
	}
}
