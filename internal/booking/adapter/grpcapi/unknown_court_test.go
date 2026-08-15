package grpcapi_test

import (
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/nhuthuynh/white-label/internal/booking/adapter/grpcapi"
	"github.com/nhuthuynh/white-label/internal/booking/app"
	"github.com/nhuthuynh/white-label/internal/booking/domain"
)

// unknownCourt is uuid-shaped and named by no courts row — the exact input
// issue #185 is about. courtID(1) is the court seedKnownCourts registers as
// real, so the two differ only in whether the FK would accept them, which is
// the whole point.
var unknownCourt = courtID(999)

// newFKHandler builds the CreateBooking handler over a repository fake that
// MODELS THE FOREIGN KEY on bookings.court_id (see storingBookingRepo's
// knownCourts comment in recurring_hire_regression_test.go). Only courtID(1)
// exists.
func newFKHandler() (*grpcapi.Handler, *storingBookingRepo) {
	bookings := newStoringBookingRepo().seedKnownCourts(courtID(1))
	svc := app.NewService(app.ServiceOptions{
		Bookings:       bookings,
		PricingRules:   &fakePricingRepo{},
		DiscountRules:  &fakeDiscountRepo{byFacility: map[string][]domain.DiscountRule{}},
		RecurringHires: newFakeRecurringRepo(),
		Facilities:     fakeFacilityLookup{},
		Identity:       newFakeIdentityLookup(),
		IDs:            &fakeIDs{},
	})
	return grpcapi.NewHandler(svc), bookings
}

// TestCreateBooking_UnknownCourtIsNotFoundNotInternal is issue #185's
// reported shape on Booking's own RPC — the first of the three the issue
// names, and the one the other two reach through.
//
// Before T15.6, a syntactically valid UUID naming no courts row passed
// app.Service.CreateBooking's uuidShape guard (a shape check and nothing
// more), reached the INSERT, failed the FK as Postgres 23503, and — with no
// 23503 arm in adapter/postgres.translateErr — arrived at toStatus as a
// wrapped, unclassified infra error. It answered codes.Internal: HTTP 500 for
// a caller naming a court that does not exist.
//
// The fake behind this handler models that FK; without that, the request
// would succeed and this test would pass whether or not the fix exists (the
// #185 testing note, and what hid the defect through T14.8).
func TestCreateBooking_UnknownCourtIsNotFoundNotInternal(t *testing.T) {
	t.Parallel()

	h, repo := newFKHandler()

	req := mapCreateBookingRequest()
	req.CourtId = unknownCourt
	_, err := h.CreateBooking(anonymous(), req)

	if got := status.Code(err); got == codes.Internal {
		t.Fatalf("CreateBooking(unknown court) still answers Internal — issue #185 is not fixed (err: %v)", err)
	} else if got != codes.NotFound {
		t.Fatalf("CreateBooking(unknown court) code = %v (err: %v), want NotFound", got, err)
	}
	if n := len(repo.bookings); n != 0 {
		t.Fatalf("an unknown court id persisted %d booking(s), want 0", n)
	}
}

// TestCreateBooking_MalformedAndUnknownCourtAnswerAlike pins the decision
// this ticket had to make explicitly (#185's question 2): both halves share
// one sentinel, domain.ErrInvalidCourtReference, so they answer with one
// code. The codes are asserted equal TO EACH OTHER rather than to a constant,
// so a future change that re-splits them fails here even if someone updates
// the constant in the mapping table to match — the same technique
// error_mapping_test.go's TestBookingService_SameSentinelSameCodeAcrossRPCs
// uses against #131's one-concept-two-codes shape.
func TestCreateBooking_MalformedAndUnknownCourtAnswerAlike(t *testing.T) {
	t.Parallel()

	h, _ := newFKHandler()

	malformedReq := mapCreateBookingRequest()
	malformedReq.CourtId = "not-a-uuid"
	_, malformedErr := h.CreateBooking(anonymous(), malformedReq)

	unknownReq := mapCreateBookingRequest()
	unknownReq.CourtId = unknownCourt
	_, unknownErr := h.CreateBooking(anonymous(), unknownReq)

	if status.Code(malformedErr) != status.Code(unknownErr) {
		t.Fatalf("malformed court id -> %v but unknown court id -> %v; one sentinel must give one code",
			status.Code(malformedErr), status.Code(unknownErr))
	}
	if status.Code(unknownErr) == codes.Internal {
		t.Fatalf("both halves agree on Internal, which is the defect rather than the fix: %v", unknownErr)
	}
}

// TestCreateBooking_KnownCourtStillSucceeds is the control: without it, a
// change that rejected every court id — or an FK-modelling fake seeded with
// nothing — would satisfy both tests above for the wrong reason.
func TestCreateBooking_KnownCourtStillSucceeds(t *testing.T) {
	t.Parallel()

	h, repo := newFKHandler()

	if _, err := h.CreateBooking(anonymous(), mapCreateBookingRequest()); err != nil {
		t.Fatalf("CreateBooking(known court) = %v, want success", err)
	}
	if n := len(repo.bookings); n != 1 {
		t.Fatalf("wrote %d booking(s), want 1", n)
	}
}
