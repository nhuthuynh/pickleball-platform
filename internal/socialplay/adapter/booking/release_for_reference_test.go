package booking_test

import (
	"context"
	"errors"
	"testing"

	bookingdomain "github.com/nhuthuynh/white-label/internal/booking/domain"
)

// T55.2 — the cross-context proof for issue #124's court half, driven
// through the REAL socialplay booking adapter over a real
// bookingapp.Service. Nothing is stubbed at the seam under test.
//
// #124's report: a cancelled Game leaves its `game`-source Bookings holding
// the courts. This file proves the release works at the boundary Social Play
// actually crosses, and holds it to the T3 standard the issue asked for —
// the court must be genuinely RE-RESERVABLE afterwards, not merely marked
// cancelled.
//
// The CancelGame wiring itself is proven separately in
// internal/socialplay/app; this file is about the port doing what the port
// says.
//
// # Why these tests seed the repo directly instead of calling ReserveCourt
//
// This package's harness mints a constant booking id (fixedIDs.NewID returns
// mintedBookingID), and domain.EnsureNoConflict deliberately self-excludes a
// booking with the candidate's own id — so two ReserveCourt calls through
// this harness produce one map entry, not two, and never conflict with each
// other. A multi-booking cascade therefore cannot be built through the port
// here. Seeding distinct ids directly is the honest way to exercise it; the
// re-reservability assertion below still runs through the real port, which
// is the half that matters.

// seedGameBooking puts a game-source Booking carrying referenceID into the
// repo, which is what ScheduleGame leaves behind and what the cascade has to
// find. seedBooking (reservation_test.go) deliberately seeds an unreferenced
// individual booking instead, so both shapes are available.
func seedGameBooking(repo *inMemoryRepo, id, referenceID string, rng bookingdomain.TimeRange) {
	repo.bookings[id] = bookingdomain.Booking{
		ID:          id,
		CourtID:     fixtureCourtID,
		Source:      bookingdomain.SourceGame,
		Status:      bookingdomain.StatusConfirmed,
		Range:       rng,
		ReferenceID: referenceID,
		OwnerUserID: testBookingOwner,
	}
}

func TestReleaseCourtsForReference_FreesEveryCourtTheGameHeld(t *testing.T) {
	res, repo := newReservation(t)
	ctx := context.Background()
	morning := slot(t, 10, 11)
	afternoon := slot(t, 14, 15)

	const (
		bookingA = "3f2504e0-4f89-11d3-9a0c-0305e82c3401"
		bookingB = "3f2504e0-4f89-11d3-9a0c-0305e82c3402"
		bookingC = "3f2504e0-4f89-11d3-9a0c-0305e82c3403"
	)

	// Two bookings for the Game being cancelled...
	seedGameBooking(repo, bookingA, fixtureGameID, morning)
	seedGameBooking(repo, bookingB, fixtureGameID, afternoon)
	// ...and one for a DIFFERENT game, which the cascade must not touch.
	// Without this, a cascade that cancelled everything would pass.
	seedGameBooking(repo, bookingC, "some-other-game", slot(t, 18, 19))

	released, err := res.ReleaseCourtsForReference(ctx, fixtureGameID, testBookingOwner)
	if err != nil {
		t.Fatalf("ReleaseCourtsForReference: %v", err)
	}
	if released != 2 {
		t.Fatalf("released %d courts, want 2", released)
	}

	for _, id := range []string{bookingA, bookingB} {
		if got := repo.bookings[id].Status; got != bookingdomain.StatusCancelled {
			t.Fatalf("booking %s status = %q, want cancelled", id, got)
		}
	}
	if got := repo.bookings[bookingC].Status; got != bookingdomain.StatusConfirmed {
		t.Fatalf("other game's booking status = %q, want it untouched (confirmed)", got)
	}

	// The #124 assertion, through the real port: the freed slot is
	// genuinely re-reservable, not merely flagged.
	if _, err := res.ReserveCourt(ctx, fixtureCourtID, morning.Start, morning.End, "other-game", testBookingOwner); err != nil {
		t.Fatalf("re-reserving the released slot = %v, want nil", err)
	}
}

// The slot really was held beforehand — otherwise the assertion above would
// pass against a harness that never enforced conflicts at all.
func TestReleaseCourtsForReference_SlotWasGenuinelyHeldBeforeRelease(t *testing.T) {
	res, repo := newReservation(t)
	ctx := context.Background()
	morning := slot(t, 10, 11)

	seedGameBooking(repo, "3f2504e0-4f89-11d3-9a0c-0305e82c3404", fixtureGameID, morning)

	if _, err := res.ReserveCourt(ctx, fixtureCourtID, morning.Start, morning.End, "other-game", testBookingOwner); err == nil {
		t.Fatal("the slot should be held before release, but a competing reservation succeeded")
	}

	if _, err := res.ReleaseCourtsForReference(ctx, fixtureGameID, testBookingOwner); err != nil {
		t.Fatalf("ReleaseCourtsForReference: %v", err)
	}

	if _, err := res.ReserveCourt(ctx, fixtureCourtID, morning.Start, morning.End, "other-game", testBookingOwner); err != nil {
		t.Fatalf("re-reserving after release = %v, want nil", err)
	}
}

// No Booking sentinel may cross this boundary — CLAUDE.md rule 5, the same
// discipline every other method on this adapter follows.
func TestReleaseCourtsForReference_DoesNotLeakBookingSentinels(t *testing.T) {
	res, repo := newReservation(t)
	ctx := context.Background()

	seedGameBooking(repo, "3f2504e0-4f89-11d3-9a0c-0305e82c3405", fixtureGameID, slot(t, 10, 11))

	// A caller who is not the owner is refused (DECISION D1), and the
	// refusal must arrive as this adapter's own wrapped error rather than as
	// bookingdomain.ErrNotBookingOwner itself.
	_, err := res.ReleaseCourtsForReference(ctx, fixtureGameID, "22222222-2222-2222-2222-222222222222")
	if err == nil {
		t.Fatal("expected a non-owner release to be refused")
	}
	if errors.Is(err, bookingdomain.ErrNotBookingOwner) {
		t.Fatalf("leaked the bookingdomain sentinel across the boundary: %v", err)
	}
}

// Idempotent, and an unknown reference is a no-op rather than an error — the
// contract port.CourtReservation states, which CancelGame relies on because
// it surfaces a cascade failure to its caller.
func TestReleaseCourtsForReference_IdempotentAndUnknownReferenceIsNoOp(t *testing.T) {
	res, repo := newReservation(t)
	ctx := context.Background()

	seedGameBooking(repo, "3f2504e0-4f89-11d3-9a0c-0305e82c3406", fixtureGameID, slot(t, 10, 11))

	if n, err := res.ReleaseCourtsForReference(ctx, fixtureGameID, testBookingOwner); err != nil || n != 1 {
		t.Fatalf("first release = (%d, %v), want (1, nil)", n, err)
	}
	if n, err := res.ReleaseCourtsForReference(ctx, fixtureGameID, testBookingOwner); err != nil || n != 0 {
		t.Fatalf("second release = (%d, %v), want (0, nil) — must be idempotent", n, err)
	}
	if n, err := res.ReleaseCourtsForReference(ctx, "no-such-game", testBookingOwner); err != nil || n != 0 {
		t.Fatalf("unknown reference = (%d, %v), want (0, nil)", n, err)
	}
}
