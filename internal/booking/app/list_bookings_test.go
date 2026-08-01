package app_test

import (
	"context"
	"testing"

	"github.com/nhuthuynh/white-label/internal/booking/app"
	"github.com/nhuthuynh/white-label/internal/booking/domain"
)

// TestListCourtBookings_ReturnsIntersectingBookings is the app-level proof
// (HANDOFF.md T2 AC: "REST GET returns bookings intersecting the range")
// that ListCourtBookings is wired to the repository and actually filters by
// court and overlap, not just "all bookings ever made".
func TestListCourtBookings_ReturnsIntersectingBookings(t *testing.T) {
	t.Parallel()

	repo := newInMemoryRepo()
	svc := app.NewService(repo, &fakePricingRepo{}, &sequentialIDs{})
	ctx := context.Background()

	// court-1: a morning booking and an evening booking.
	morning := mustTimeRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")
	evening := mustTimeRange(t, "2026-08-03T18:00:00Z", "2026-08-03T19:00:00Z")
	if _, err := svc.CreateBooking(ctx, app.CreateBookingInput{CourtID: "court-1", Source: domain.SourceIndividual, Range: morning}); err != nil {
		t.Fatalf("unexpected err creating fixture: %v", err)
	}
	if _, err := svc.CreateBooking(ctx, app.CreateBookingInput{CourtID: "court-1", Source: domain.SourceGame, Range: evening, ReferenceID: "game-1"}); err != nil {
		t.Fatalf("unexpected err creating fixture: %v", err)
	}
	// court-2: overlaps the same window but is a different court.
	if _, err := svc.CreateBooking(ctx, app.CreateBookingInput{CourtID: "court-2", Source: domain.SourceIndividual, Range: morning}); err != nil {
		t.Fatalf("unexpected err creating fixture: %v", err)
	}

	wholeDay := mustTimeRange(t, "2026-08-03T00:00:00Z", "2026-08-04T00:00:00Z")
	got, err := svc.ListCourtBookings(ctx, "court-1", wholeDay)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d bookings, want 2 (both court-1 bookings intersecting the day)", len(got))
	}

	morningOnly, err := svc.ListCourtBookings(ctx, "court-1", morning)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(morningOnly) != 1 || morningOnly[0].Range != morning {
		t.Fatalf("got %+v, want exactly the morning booking", morningOnly)
	}

	none, err := svc.ListCourtBookings(ctx, "court-9", wholeDay)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(none) != 0 {
		t.Fatalf("got %d bookings for a court with none, want 0", len(none))
	}
}

func TestListCourtBookings_ExcludesCancelled(t *testing.T) {
	t.Parallel()

	repo := newInMemoryRepo()
	svc := app.NewService(repo, &fakePricingRepo{}, &sequentialIDs{})
	ctx := context.Background()

	rng := mustTimeRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z")
	created, err := svc.CreateBooking(ctx, app.CreateBookingInput{CourtID: "court-1", Source: domain.SourceIndividual, Range: rng})
	if err != nil {
		t.Fatalf("unexpected err creating fixture: %v", err)
	}
	created.Status = domain.StatusCancelled
	if _, err := repo.Update(ctx, created); err != nil {
		t.Fatalf("unexpected err cancelling fixture: %v", err)
	}

	got, err := svc.ListCourtBookings(ctx, "court-1", rng)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("got %d bookings, want 0 (cancelled booking must not be listed)", len(got))
	}
}
