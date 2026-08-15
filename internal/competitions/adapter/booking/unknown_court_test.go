package booking_test

import (
	"context"
	"errors"
	"testing"

	bookingdomain "github.com/nhuthuynh/white-label/internal/booking/domain"
	"github.com/nhuthuynh/white-label/internal/competitions/domain"
)

// unknownCourtID is uuid-shaped and seeded into no repo — the exact input
// issue #185 is about. It passes bookingapp.CreateBooking's uuidShape guard
// and is caught only by the FK that inMemoryRepo models.
const unknownCourtID = "3f2504e0-4f89-11d3-9a0c-0305e82c33ff"

// TestReserveCourt_UnknownCourtIsTranslated is Competitions' half of T15.6
// (issue #185) — the twin of the identically named test in
// internal/socialplay/adapter/booking. Before this ticket, a well-formed but
// unknown court id produced an untranslated Postgres FK violation, which this
// adapter correctly stripped of its type with %s rather than %w (CLAUDE.md
// rule 5), leaving toStatus nothing to classify: ScheduleCompetition answered
// 500 for a caller naming a court that does not exist.
func TestReserveCourt_UnknownCourtIsTranslated(t *testing.T) {
	t.Parallel()

	res, repo := newReservation(t)
	rng := slot(t, 10, 11)

	_, err := res.ReserveCourt(context.Background(), unknownCourtID, rng.Start, rng.End, fixtureCompetitionID)

	if !errors.Is(err, domain.ErrCourtNotFound) {
		t.Fatalf("ReserveCourt(unknown court) = %v, want competitions domain.ErrCourtNotFound", err)
	}
	if errors.Is(err, bookingdomain.ErrInvalidCourtReference) {
		t.Fatalf("leaked the bookingdomain sentinel across the boundary: %v", err)
	}
	// An unknown court must not be confused with a busy one: the two carry
	// different fixes for the caller (correct the id vs. pick another slot).
	if errors.Is(err, domain.ErrCourtUnavailable) {
		t.Fatalf("misclassified an unknown court as a court conflict: %v", err)
	}
	if len(repo.bookings) != 0 {
		t.Fatalf("an unknown court left a booking behind: %v", repo.bookings)
	}
}

// TestReserveCourt_KnownCourtStillSucceeds is the control for the test above:
// without it, a translation that rejected EVERY court id would pass.
func TestReserveCourt_KnownCourtStillSucceeds(t *testing.T) {
	t.Parallel()

	res, repo := newReservation(t)
	rng := slot(t, 10, 11)

	bookingID, err := res.ReserveCourt(context.Background(), fixtureCourtID, rng.Start, rng.End, fixtureCompetitionID)
	if err != nil {
		t.Fatalf("ReserveCourt(known court) = %v, want success", err)
	}
	if bookingID != mintedBookingID {
		t.Fatalf("ReserveCourt returned booking id %q, want %q", bookingID, mintedBookingID)
	}
	if len(repo.bookings) != 1 {
		t.Fatalf("want exactly 1 booking written, got %d", len(repo.bookings))
	}
}
