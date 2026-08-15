package grpcapi_test

import (
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// shapeUnknownCourt is uuid-shaped and seeded into no repository — the exact
// input issue #185 is about. It passes every shape guard on the way down
// (competitions' own T14.8 court_ids check, then bookingapp.CreateBooking's
// uuidShape guard) and is caught only by the FK on bookings.court_id, which
// newShapeBookingRepo models. Distinct from shapeFixtureCourtA/B, which that
// fake seeds as real courts.
const shapeUnknownCourt = "3f2504e0-4f89-11d3-9a0c-0305e82c33ff"

// TestCreateCompetition_UnknownCourtIsNotFoundNotInternal is issue #185's
// reported shape on the Competitions RPC, and the direct sibling of this
// file's neighbour
// TestCreateCompetition_MalformedCourtIDIsInvalidArgumentNotInternal (#156,
// T14.8) — the half that fix did not reach.
//
// It runs over newBookingBackedHandler: the REAL chain cmd/server wires,
// grpcapi.Handler -> app.Service -> the real competitionsbooking.Reservation
// -> the real bookingapp.Service, with only the Postgres repository faked.
// Every layer in that chain is load-bearing for this defect, so a fake
// reservation could only prove what the fake was told to say.
//
// Before T15.6 this request answered codes.Internal — HTTP 500 — for a
// caller naming a court that does not exist.
func TestCreateCompetition_UnknownCourtIsNotFoundNotInternal(t *testing.T) {
	h, repo, bookingRepo := newBookingBackedHandler()

	_, err := h.CreateCompetition(ctxAs("host-1"), shapeCreateCompetitionReq(
		protoSession("2026-09-01T09:00:00Z", "2026-09-01T12:00:00Z", shapeUnknownCourt),
	))

	if got := status.Code(err); got == codes.Internal {
		t.Fatalf("CreateCompetition(court_ids=[unknown uuid]) still answers Internal — issue #185 is not fixed (err: %v)", err)
	} else if got != codes.NotFound {
		t.Fatalf("CreateCompetition(court_ids=[unknown uuid]) code = %v (err: %v), want NotFound", got, err)
	}

	if n := len(repo.competitions); n != 0 {
		t.Fatalf("an unknown court id persisted %d competition(s), want 0", n)
	}
	if n := len(bookingRepo.bookings); n != 0 {
		t.Fatalf("an unknown court id left %d booking(s) behind, want 0", n)
	}
}

// TestCreateCompetition_UnknownCourtAfterAValidOneRollsBack is the
// multi-court case. ScheduleCompetition reserves across sessions x courts in
// order, so an unknown id after a good one must release the Booking already
// taken — otherwise a rejected request silently holds a real court, which is
// worse than the 500 this ticket removes.
func TestCreateCompetition_UnknownCourtAfterAValidOneRollsBack(t *testing.T) {
	h, repo, bookingRepo := newBookingBackedHandler()

	_, err := h.CreateCompetition(ctxAs("host-1"), shapeCreateCompetitionReq(
		protoSession("2026-09-01T09:00:00Z", "2026-09-01T12:00:00Z", shapeFixtureCourtA, shapeUnknownCourt),
	))
	if status.Code(err) != codes.NotFound {
		t.Fatalf("CreateCompetition(court_ids=[valid, unknown]) code = %v (err: %v), want NotFound", status.Code(err), err)
	}

	if n := len(repo.competitions); n != 0 {
		t.Fatalf("persisted %d competition(s) after a failed reservation, want 0", n)
	}
	for id, b := range bookingRepo.bookings {
		if b.Status != "cancelled" {
			t.Fatalf("booking %s on court %s was left %s, want it released", id, b.CourtID, b.Status)
		}
	}
}

// TestCreateCompetition_KnownCourtStillSucceeds is the control: without it,
// an over-broad change would make every test above pass for the wrong reason.
func TestCreateCompetition_KnownCourtStillSucceeds(t *testing.T) {
	h, repo, bookingRepo := newBookingBackedHandler()

	_, err := h.CreateCompetition(ctxAs("host-1"), shapeCreateCompetitionReq(
		protoSession("2026-09-01T09:00:00Z", "2026-09-01T12:00:00Z", shapeFixtureCourtA),
	))
	if err != nil {
		t.Fatalf("CreateCompetition(court_ids=[known uuid]) = %v, want success", err)
	}
	if n := len(repo.competitions); n != 1 {
		t.Fatalf("persisted %d competition(s), want 1", n)
	}
	if n := len(bookingRepo.bookings); n != 1 {
		t.Fatalf("wrote %d booking(s), want 1", n)
	}
}
