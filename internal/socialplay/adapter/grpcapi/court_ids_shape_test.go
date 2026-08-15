// End-to-end reproduction and fix for issue #156 (T14.8): CreateGame with a
// malformed entry in court_ids answered codes.Internal (HTTP 500) while an
// empty court_ids correctly answered codes.InvalidArgument (HTTP 400), even
// though both are pure client input errors.
//
// Why this file wires the REAL cross-context Booking adapter instead of
// entry_fee_test.go's fakeReservation: the 500 was produced by the port
// boundary itself, not by Social Play. bookingapp.CreateBooking's own
// uuidShape guard returns bookingdomain.ErrInvalidCourtReference; the adapter
// strips the sentinel with %s rather than %w — correctly, per CLAUDE.md rule
// 5 — so nothing survived for toStatus to match and the error fell into its
// default codes.Internal arm. A fake reservation cannot reproduce that: it
// would only prove whatever the fake was told to return. These tests drive
// the real bookingapp.Service over an in-memory booking repository, the same
// choice internal/socialplay/adapter/booking's own tests (T13.1) make.
package grpcapi_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	bookingapp "github.com/nhuthuynh/white-label/internal/booking/app"
	bookingdomain "github.com/nhuthuynh/white-label/internal/booking/domain"
	socialplayv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/socialplay/v1"
	socialplaybooking "github.com/nhuthuynh/white-label/internal/socialplay/adapter/booking"
	"github.com/nhuthuynh/white-label/internal/socialplay/adapter/grpcapi"
	"github.com/nhuthuynh/white-label/internal/socialplay/app"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	shapeFixtureCourtA = "3f2504e0-4f89-11d3-9a0c-0305e82c3301"
	shapeFixtureCourtB = "3f2504e0-4f89-11d3-9a0c-0305e82c3302"
	shapeMintedBooking = "3f2504e0-4f89-11d3-9a0c-0305e82c3401"
)

// courtID mints a deterministic, UUID-shaped court id for this package's
// other test files, which used to seed "court-1" — a value that never
// existed in any courts table and that bookingapp.CreateBooking rejects
// outright. Those fixtures passed only because they never reached the real
// Booking context (see this file's package comment), which is precisely why
// #156 survived every gate until T13.1 went looking at the seam.
func courtID(n int) string {
	return fmt.Sprintf("00000000-0000-4000-c000-%012d", n)
}

// --- real Booking context, in-memory persistence --------------------------

type shapeBookingRepo struct {
	bookings map[string]bookingdomain.Booking
}

func newShapeBookingRepo() *shapeBookingRepo {
	return &shapeBookingRepo{bookings: make(map[string]bookingdomain.Booking)}
}

func (r *shapeBookingRepo) Create(_ context.Context, b bookingdomain.Booking) (bookingdomain.Booking, error) {
	r.bookings[b.ID] = b
	return b, nil
}

func (r *shapeBookingRepo) ListActiveForCourt(_ context.Context, courtID string, rng bookingdomain.TimeRange) ([]bookingdomain.Booking, error) {
	var out []bookingdomain.Booking
	for _, b := range r.bookings {
		if b.CourtID != courtID || b.Status == bookingdomain.StatusCancelled {
			continue
		}
		if b.Range.Overlaps(rng) {
			out = append(out, b)
		}
	}
	return out, nil
}

func (r *shapeBookingRepo) GetByID(_ context.Context, id string) (bookingdomain.Booking, error) {
	b, ok := r.bookings[id]
	if !ok {
		return bookingdomain.Booking{}, bookingdomain.ErrBookingNotFound
	}
	return b, nil
}

func (r *shapeBookingRepo) Update(_ context.Context, b bookingdomain.Booking) (bookingdomain.Booking, error) {
	if _, ok := r.bookings[b.ID]; !ok {
		return bookingdomain.Booking{}, bookingdomain.ErrBookingNotFound
	}
	r.bookings[b.ID] = b
	return b, nil
}

// shapeBookingStubs satisfies the bookingapp dependencies CreateBooking never
// reaches (pricing, discounts, recurring hires, facilities, identity) in one
// type — reserving a court touches none of them.
type shapeBookingStubs struct{}

func (shapeBookingStubs) ListForCourt(context.Context, string) ([]bookingdomain.PricingRule, error) {
	return nil, nil
}

func (shapeBookingStubs) Create(_ context.Context, r bookingdomain.DiscountRule) (bookingdomain.DiscountRule, error) {
	return r, nil
}

func (shapeBookingStubs) ListForFacility(context.Context, string) ([]bookingdomain.DiscountRule, error) {
	return nil, nil
}

func (shapeBookingStubs) EnsureFacilityOwner(context.Context, string, string) error { return nil }

func (shapeBookingStubs) FacilityIDForCourt(context.Context, string) (string, error) {
	return "", bookingdomain.ErrFacilityNotFound
}

func (shapeBookingStubs) CourtIDsForFacility(context.Context, string) ([]string, error) {
	return nil, nil
}

func (shapeBookingStubs) EnsureClubRole(context.Context, string) error { return nil }

func (shapeBookingStubs) UserIDBySubject(context.Context, string) (string, error) {
	return "", bookingdomain.ErrUserNotFound
}

type shapeRecurringStub struct{}

func (shapeRecurringStub) Create(_ context.Context, tmpl bookingdomain.RecurringHireTemplate) (bookingdomain.RecurringHireTemplate, error) {
	return tmpl, nil
}

func (shapeRecurringStub) GetByID(context.Context, string) (bookingdomain.RecurringHireTemplate, error) {
	return bookingdomain.RecurringHireTemplate{}, bookingdomain.ErrRecurringHireTemplateNotFound
}

func (shapeRecurringStub) UpdateStatus(_ context.Context, tmpl bookingdomain.RecurringHireTemplate) (bookingdomain.RecurringHireTemplate, error) {
	return tmpl, nil
}

func (shapeRecurringStub) ListForCourts(context.Context, []string) ([]bookingdomain.RecurringHireTemplate, error) {
	return nil, nil
}

func (shapeRecurringStub) ListForRequester(context.Context, string) ([]bookingdomain.RecurringHireTemplate, error) {
	return nil, nil
}

type shapeBookingIDs struct{}

func (shapeBookingIDs) NewID() string { return shapeMintedBooking }

// newBookingBackedHandler wires grpcapi.Handler -> app.Service -> the real
// socialplaybooking.Reservation -> the real bookingapp.Service, i.e. the
// exact chain cmd/server builds and the exact chain #156 walked.
func newBookingBackedHandler() (*grpcapi.Handler, *fakeGameRepo, *shapeBookingRepo) {
	bookingRepo := newShapeBookingRepo()
	bookingSvc := bookingapp.NewService(bookingapp.ServiceOptions{
		Bookings:       bookingRepo,
		PricingRules:   shapeBookingStubs{},
		DiscountRules:  shapeBookingStubs{},
		RecurringHires: shapeRecurringStub{},
		Facilities:     shapeBookingStubs{},
		Identity:       shapeBookingStubs{},
		IDs:            shapeBookingIDs{},
	})

	gameRepo := newFakeGameRepo()
	svc := app.NewService(app.ServiceOptions{
		IDs:           &fakeIDs{},
		Games:         gameRepo,
		Registrations: newFakeRegistrationRepo(),
		Waitlist:      newFakeWaitlistRepo(),
		Matches:       newFakeMatchRepo(),
		GameAdmins:    newFakeGameAdminRepo(),
	})

	return grpcapi.NewHandler(svc, socialplaybooking.NewReservation(bookingSvc), nil), gameRepo, bookingRepo
}

func shapeCreateGameReq(courtIDs ...string) *socialplayv1.CreateGameRequest {
	start := time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC)
	return &socialplayv1.CreateGameRequest{
		FacilityId:    "facility-1",
		CourtIds:      courtIDs,
		StartsAt:      timestamppb.New(start),
		EndsAt:        timestamppb.New(start.Add(time.Hour)),
		Capacity:      4,
		PaymentMethod: socialplayv1.PaymentMethod_PAYMENT_METHOD_EITHER,
	}
}

// TestCreateGame_MalformedCourtIDIsInvalidArgumentNotInternal is issue #156's
// literal reported shape: court_ids: ["not-a-uuid"] returned 500, court_ids:
// [] returned 400, and both are the client's mistake.
func TestCreateGame_MalformedCourtIDIsInvalidArgumentNotInternal(t *testing.T) {
	h, _, _ := newBookingBackedHandler()

	_, err := h.CreateGame(ctxAs("host-1"), shapeCreateGameReq("not-a-uuid"))
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("CreateGame(court_ids=[\"not-a-uuid\"]) code = %v (err: %v), want InvalidArgument", status.Code(err), err)
	}
}

// TestCreateGame_EmptyCourtIDsIsInvalidArgument pins #156's own baseline —
// the behaviour the malformed case is being brought into line with. If this
// ever regresses, the comparison the ticket rests on is gone.
func TestCreateGame_EmptyCourtIDsIsInvalidArgument(t *testing.T) {
	h, _, _ := newBookingBackedHandler()

	_, err := h.CreateGame(ctxAs("host-1"), shapeCreateGameReq())
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("CreateGame(court_ids=[]) code = %v (err: %v), want InvalidArgument", status.Code(err), err)
	}
}

// TestCreateGame_MalformedCourtIDLeavesNoBookingBehind is the partial-state
// proof at the seam that matters: the good court is listed FIRST, so a guard
// applied per-court inside the reservation loop (rather than over the whole
// input up front) would leave a real Booking in the Booking context's
// repository and a rejected Game in Social Play's.
func TestCreateGame_MalformedCourtIDLeavesNoBookingBehind(t *testing.T) {
	h, games, bookings := newBookingBackedHandler()

	_, err := h.CreateGame(ctxAs("host-1"), shapeCreateGameReq(shapeFixtureCourtA, "not-a-uuid"))
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("code = %v (err: %v), want InvalidArgument", status.Code(err), err)
	}
	if len(bookings.bookings) != 0 {
		t.Fatalf("Booking context holds %d bookings, want 0 — the valid court must never have been reserved", len(bookings.bookings))
	}
	if len(games.games) != 0 {
		t.Fatalf("persisted %d games, want 0", len(games.games))
	}
}

// TestCreateGame_WellFormedCourtIDsStillReserveThroughRealBooking is the
// too-strict rail across the real port: a guard that rejected genuine court
// ids would take CreateGame down entirely, and no fake would notice.
func TestCreateGame_WellFormedCourtIDsStillReserveThroughRealBooking(t *testing.T) {
	h, games, bookings := newBookingBackedHandler()

	if _, err := h.CreateGame(ctxAs("host-1"), shapeCreateGameReq(shapeFixtureCourtB)); err != nil {
		t.Fatalf("well-formed court id must still schedule, got err: %v", err)
	}
	if len(bookings.bookings) != 1 {
		t.Fatalf("Booking context holds %d bookings, want 1", len(bookings.bookings))
	}
	if len(games.games) != 1 {
		t.Fatalf("persisted %d games, want 1", len(games.games))
	}
}
