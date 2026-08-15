package booking_test

import (
	"context"
	"errors"
	"testing"

	bookingdomain "github.com/nhuthuynh/white-label/internal/booking/domain"
	"github.com/nhuthuynh/white-label/internal/socialplay/domain"
)

// unknownCourtID is uuid-shaped and seeded into no repo — the exact input
// issue #185 is about. It passes bookingapp.CreateBooking's uuidShape guard
// and is caught only by the FK that inMemoryRepo models.
const unknownCourtID = "3f2504e0-4f89-11d3-9a0c-0305e82c33ff"

// TestReserveCourt_UnknownCourtIsTranslated is Social Play's half of T15.6
// (issue #185). Before this ticket, a well-formed but unknown court id
// produced an untranslated Postgres FK violation, which this adapter — quite
// correctly, per CLAUDE.md rule 5 — stripped of its type with %s rather than
// %w, leaving nothing for toStatus to classify. It landed in
// `default: codes.Internal`, so ScheduleGame answered 500 for a caller naming
// a court that does not exist.
//
// The fix has two halves and this test covers the second: Booking's Postgres
// adapter now translates 23503 to bookingdomain.ErrInvalidCourtReference, and
// this boundary translates THAT into Social Play's own
// domain.ErrCourtNotFound — never the bookingdomain type itself, exactly as
// ErrCourtDoubleBooked -> ErrCourtUnavailable already works beside it.
func TestReserveCourt_UnknownCourtIsTranslated(t *testing.T) {
	t.Parallel()

	res, repo := newReservation(t)
	rng := slot(t, 10, 11)

	_, err := res.ReserveCourt(context.Background(), unknownCourtID, rng.Start, rng.End, fixtureGameID)

	if !errors.Is(err, domain.ErrCourtNotFound) {
		t.Fatalf("ReserveCourt(unknown court) = %v, want socialplay domain.ErrCourtNotFound", err)
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

// TestReserveCourt_KnownCourtStillSucceeds is the control for the test above.
// Without it, a translation that rejected *every* court id would pass — the
// failure mode a fake that models an FK invites, and worth one assertion to
// close off.
func TestReserveCourt_KnownCourtStillSucceeds(t *testing.T) {
	t.Parallel()

	res, repo := newReservation(t)
	rng := slot(t, 10, 11)

	bookingID, err := res.ReserveCourt(context.Background(), fixtureCourtID, rng.Start, rng.End, fixtureGameID)
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
