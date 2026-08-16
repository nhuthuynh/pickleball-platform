// End-to-end reproduction and fix for issue #156 (T14.8) on the Competitions
// side: CreateCompetition with a malformed entry in a session's court_ids
// answered codes.Internal (HTTP 500) while an empty court_ids correctly
// answered codes.InvalidArgument (HTTP 400).
//
// Like its Social Play twin, this file wires the REAL cross-context Booking
// adapter rather than authz_regression_test.go's fakeReservation: the 500 was
// manufactured by the port boundary — bookingapp.CreateBooking's uuidShape
// guard returning a sentinel that competitions/adapter/booking correctly
// strips with %s (CLAUDE.md rule 5) — so a fake reservation could only ever
// prove what the fake was told to say.
package grpcapi_test

import (
	"context"
	"fmt"
	"testing"

	bookingapp "github.com/nhuthuynh/white-label/internal/booking/app"
	bookingdomain "github.com/nhuthuynh/white-label/internal/booking/domain"
	competitionsbooking "github.com/nhuthuynh/white-label/internal/competitions/adapter/booking"
	"github.com/nhuthuynh/white-label/internal/competitions/adapter/grpcapi"
	"github.com/nhuthuynh/white-label/internal/competitions/app"
	competitionsv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/competitions/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	shapeFixtureCourtA = "3f2504e0-4f89-11d3-9a0c-0305e82c3301"
	shapeFixtureCourtB = "3f2504e0-4f89-11d3-9a0c-0305e82c3302"
)

// --- real Booking context, in-memory persistence --------------------------

// knownCourts MODELS THE FOREIGN KEY, deliberately and explicitly (T15.6,
// issue #185) — the twin of the identical field in
// internal/socialplay/adapter/grpcapi/court_ids_shape_test.go. Read this
// before using this fake for anything court-shaped.
//
// bookings.court_id is `uuid NOT NULL REFERENCES courts (id)`
// (db/migrations/0001_init.sql:24). A well-formed UUID naming no courts row
// passes bookingapp.CreateBooking's uuidShape guard — a shape check and
// nothing more — and is rejected only at the INSERT, as Postgres 23503, which
// bookingpg.translateErr turns into bookingdomain.ErrInvalidCourtReference
// (proved against that code path in
// internal/booking/adapter/postgres/foreign_key_test.go, and against a real
// Postgres in that package's unknown_court_integration_test.go).
//
// Without this check the fake accepts any well-formed court id and passes
// whether or not the bug exists. That is precisely what hid #185: #156's fix
// reached only the malformed half, because the fixture backing it could not
// represent the unknown half at all. It is the fixture-infidelity trap
// docs/LESSONS.md' T9 entry names, on the very file that closed #156.
// newShapeBookingRepo seeds both fixture courts, so the FK is modelled by
// default and a test must opt out on purpose to get the naive behaviour.
type shapeBookingRepo struct {
	bookings    map[string]bookingdomain.Booking
	knownCourts map[string]bool
}

func newShapeBookingRepo() *shapeBookingRepo {
	return &shapeBookingRepo{
		bookings: make(map[string]bookingdomain.Booking),
		knownCourts: map[string]bool{
			shapeFixtureCourtA: true,
			shapeFixtureCourtB: true,
		},
	}
}

func (r *shapeBookingRepo) Create(_ context.Context, b bookingdomain.Booking) (bookingdomain.Booking, error) {
	// The FK, modelled — see knownCourts' comment on the struct.
	if !r.knownCourts[b.CourtID] {
		return bookingdomain.Booking{}, bookingdomain.ErrInvalidCourtReference
	}
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
// reaches (pricing, discounts, facilities, identity) in one type.
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

// shapeBookingIDs mints a distinct uuid per Booking — a Competition reserves
// across sessions × courts, so a fixed id would collapse several Bookings
// onto one map key and hide exactly the partial state these tests assert on.
type shapeBookingIDs struct{ n int }

func (g *shapeBookingIDs) NewID() string {
	g.n++
	return fmt.Sprintf("3f2504e0-4f89-11d3-9a0c-%012d", g.n)
}

// newBookingBackedHandler wires grpcapi.Handler -> app.Service -> the real
// competitionsbooking.Reservation -> the real bookingapp.Service: the chain
// cmd/server builds, and the chain #156 walked.
func newBookingBackedHandler() (*grpcapi.Handler, *fakeRepo, *shapeBookingRepo) {
	bookingRepo := newShapeBookingRepo()
	bookingSvc := bookingapp.NewService(bookingapp.ServiceOptions{
		Bookings:       bookingRepo,
		PricingRules:   shapeBookingStubs{},
		DiscountRules:  shapeBookingStubs{},
		RecurringHires: shapeRecurringStub{},
		Facilities:     shapeBookingStubs{},
		Identity:       shapeBookingStubs{},
		IDs:            &shapeBookingIDs{},
	})

	repo := newFakeRepo()
	svc := app.NewService(app.ServiceOptions{
		Competitions: repo,
		IDs:          &fakeIDs{},
		Reservation:  competitionsbooking.NewReservation(bookingSvc),
		Facilities:   fakeFacilities{},
		ShareTokens:  &fakeShareTokens{},
		Identity:     newFakeIdentityLookup(),
	})
	return grpcapi.NewHandler(svc), repo, bookingRepo
}

func shapeCreateCompetitionReq(sessions ...*competitionsv1.CompetitionSession) *competitionsv1.CreateCompetitionRequest {
	return &competitionsv1.CreateCompetitionRequest{
		Name:          "Spring Doubles Open",
		Sessions:      sessions,
		Capacity:      16,
		PaymentMethod: competitionsv1.PaymentMethod_PAYMENT_METHOD_EITHER,
		EntryFee:      &competitionsv1.Money{AmountCents: 2500, CurrencyCode: "AUD"},
		Format:        competitionsv1.CompetitionFormat_COMPETITION_FORMAT_DOUBLES,
	}
}

// TestCreateCompetition_MalformedCourtIDIsInvalidArgumentNotInternal is
// issue #156's reported shape on the Competitions RPC.
func TestCreateCompetition_MalformedCourtIDIsInvalidArgumentNotInternal(t *testing.T) {
	h, _, _ := newBookingBackedHandler()

	_, err := h.CreateCompetition(ctxAs("host-1"), shapeCreateCompetitionReq(
		protoSession("2026-09-01T09:00:00Z", "2026-09-01T12:00:00Z", "not-a-uuid"),
	))
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("CreateCompetition(court_ids=[\"not-a-uuid\"]) code = %v (err: %v), want InvalidArgument", status.Code(err), err)
	}
}

// TestCreateCompetition_EmptyCourtIDsIsInvalidArgument pins the baseline the
// malformed case is being brought into line with.
func TestCreateCompetition_EmptyCourtIDsIsInvalidArgument(t *testing.T) {
	h, _, _ := newBookingBackedHandler()

	_, err := h.CreateCompetition(ctxAs("host-1"), shapeCreateCompetitionReq(
		protoSession("2026-09-01T09:00:00Z", "2026-09-01T12:00:00Z"),
	))
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("CreateCompetition(court_ids=[]) code = %v (err: %v), want InvalidArgument", status.Code(err), err)
	}
}

// TestCreateCompetition_MalformedCourtIDLeavesNoBookingBehind hides the
// malformed id in the SECOND session, behind a first session that would
// reserve cleanly — the nested-loop partial-state case.
func TestCreateCompetition_MalformedCourtIDLeavesNoBookingBehind(t *testing.T) {
	h, repo, bookings := newBookingBackedHandler()

	_, err := h.CreateCompetition(ctxAs("host-1"), shapeCreateCompetitionReq(
		protoSession("2026-09-01T09:00:00Z", "2026-09-01T12:00:00Z", shapeFixtureCourtA),
		protoSession("2026-09-02T09:00:00Z", "2026-09-02T12:00:00Z", shapeFixtureCourtB, "not-a-uuid"),
	))
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("code = %v (err: %v), want InvalidArgument", status.Code(err), err)
	}
	if len(bookings.bookings) != 0 {
		t.Fatalf("Booking context holds %d bookings, want 0 — session 1 must never have been reserved", len(bookings.bookings))
	}
	if len(repo.competitions) != 0 {
		t.Fatalf("persisted %d competitions, want 0", len(repo.competitions))
	}
}

// TestCreateCompetition_WellFormedCourtIDsStillReserveThroughRealBooking is
// the too-strict rail across the real port.
func TestCreateCompetition_WellFormedCourtIDsStillReserveThroughRealBooking(t *testing.T) {
	h, repo, bookings := newBookingBackedHandler()

	if _, err := h.CreateCompetition(ctxAs("host-1"), shapeCreateCompetitionReq(
		protoSession("2026-09-01T09:00:00Z", "2026-09-01T12:00:00Z", shapeFixtureCourtA, shapeFixtureCourtB),
	)); err != nil {
		t.Fatalf("well-formed court ids must still schedule, got err: %v", err)
	}
	if len(bookings.bookings) != 2 {
		t.Fatalf("Booking context holds %d bookings, want 2", len(bookings.bookings))
	}
	if len(repo.competitions) != 1 {
		t.Fatalf("persisted %d competitions, want 1", len(repo.competitions))
	}
}

// courtID mints a deterministic, UUID-shaped court id for this package's
// other test files, which used to seed "court-1" — a value no courts table
// ever held and that bookingapp.CreateBooking rejects outright. See the twin
// helper in internal/socialplay/adapter/grpcapi.
func courtID(n int) string {
	return fmt.Sprintf("00000000-0000-4000-c000-%012d", n)
}
