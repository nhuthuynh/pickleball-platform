package grpcapi_test

import (
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// shapeUnknownCourt is uuid-shaped and seeded into no repository — the exact
// input issue #185 is about. It passes every shape guard on the way down
// (socialplay's own T14.8 court_ids check, then bookingapp.CreateBooking's
// uuidShape guard) and is caught only by the FK on bookings.court_id, which
// newShapeBookingRepo models. Distinct from shapeFixtureCourtA/B, which that
// fake seeds as real courts.
const shapeUnknownCourt = "3f2504e0-4f89-11d3-9a0c-0305e82c33ff"

// TestCreateGame_UnknownCourtIsNotFoundNotInternal is issue #185's reported
// shape on the Social Play RPC, and the direct sibling of this file's
// neighbour TestCreateGame_MalformedCourtIDIsInvalidArgumentNotInternal
// (#156, T14.8) — the half that fix did not reach.
//
// It runs over newBookingBackedHandler: the REAL chain cmd/server wires,
// grpcapi.Handler -> app.Service -> the real socialplaybooking.Reservation ->
// the real bookingapp.Service, with only the Postgres repository faked. That
// matters here more than anywhere, because every layer in that chain is
// load-bearing for this defect: bookingapp propagates the repository's error
// unchanged, the Reservation adapter strips its type with %s (correctly, per
// CLAUDE.md rule 5), and toStatus classifies whatever survives. A fake
// reservation could only prove what the fake was told to say.
//
// Before T15.6 this request answered codes.Internal — HTTP 500 — for a
// caller naming a court that does not exist.
func TestCreateGame_UnknownCourtIsNotFoundNotInternal(t *testing.T) {
	h, gameRepo, bookingRepo := newBookingBackedHandler()

	_, err := h.CreateGame(ctxAs("host-1"), shapeCreateGameReq(shapeUnknownCourt))

	if got := status.Code(err); got == codes.Internal {
		t.Fatalf("CreateGame(court_ids=[unknown uuid]) still answers Internal — issue #185 is not fixed (err: %v)", err)
	} else if got != codes.NotFound {
		t.Fatalf("CreateGame(court_ids=[unknown uuid]) code = %v (err: %v), want NotFound", got, err)
	}

	// Nothing may be left behind: ScheduleGame reserves every court before
	// persisting, so a rejected reservation must produce neither a Game nor a
	// Booking.
	if n := len(gameRepo.games); n != 0 {
		t.Fatalf("an unknown court id persisted %d game(s), want 0", n)
	}
	if n := len(bookingRepo.bookings); n != 0 {
		t.Fatalf("an unknown court id left %d booking(s) behind, want 0", n)
	}
}

// TestCreateGame_UnknownCourtAfterAValidOneRollsBack is the multi-court case.
// ScheduleGame reserves courts in order, so an unknown id in second position
// must roll back the Booking the first court already took — otherwise a
// rejected request silently holds a real court, which is worse than the 500
// this ticket is removing.
//
// The malformed twin of this test already exists beside it
// (TestCreateGame_MalformedCourtIDLeavesNoReservation, T14.8); that one
// passes trivially because the shape guard rejects the whole list before any
// reservation is attempted. This one is the case where a reservation really
// is taken first and really must be released.
func TestCreateGame_UnknownCourtAfterAValidOneRollsBack(t *testing.T) {
	h, gameRepo, bookingRepo := newBookingBackedHandler()

	_, err := h.CreateGame(ctxAs("host-1"), shapeCreateGameReq(shapeFixtureCourtA, shapeUnknownCourt))
	if status.Code(err) != codes.NotFound {
		t.Fatalf("CreateGame(court_ids=[valid, unknown]) code = %v (err: %v), want NotFound", status.Code(err), err)
	}

	if n := len(gameRepo.games); n != 0 {
		t.Fatalf("persisted %d game(s) after a failed reservation, want 0", n)
	}
	for id, b := range bookingRepo.bookings {
		if b.Status != "cancelled" {
			t.Fatalf("booking %s on court %s was left %s, want it released", id, b.CourtID, b.Status)
		}
	}
}

// TestCreateGame_KnownCourtStillSucceeds is the control. Without it, an
// over-broad change — or an FK-modelling fake seeded with nothing — would
// make every test above pass for the wrong reason.
func TestCreateGame_KnownCourtStillSucceeds(t *testing.T) {
	h, gameRepo, bookingRepo := newBookingBackedHandler()

	if _, err := h.CreateGame(ctxAs("host-1"), shapeCreateGameReq(shapeFixtureCourtA)); err != nil {
		t.Fatalf("CreateGame(court_ids=[known uuid]) = %v, want success", err)
	}
	if n := len(gameRepo.games); n != 1 {
		t.Fatalf("persisted %d game(s), want 1", n)
	}
	if n := len(bookingRepo.bookings); n != 1 {
		t.Fatalf("wrote %d booking(s), want 1", n)
	}
}
