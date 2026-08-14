// T11.5 — authorization, error-code and partial-success regression tests for
// Booking's RecurringHireTemplate endpoints, run through the real gRPC handler
// rather than only the app-layer unit tests in internal/booking/app. Same
// "does the guarantee survive the full stack" pattern as
// discount_regression_test.go (T11.2), which this file shares fixtures with.
//
// Handler-level (real grpcapi.Handler + real app.Service + real domain, with
// in-memory fakes standing in for the Postgres adapters) rather than a
// -tags=integration testcontainers test, for the reasons that file's own
// header gives: the checks under test live entirely in the app/domain/port
// layers a real Postgres round trip would not influence, and this environment
// has no Docker daemon, so a test that cannot run here would prove nothing
// (CLAUDE.md rule 10).
package grpcapi_test

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/nhuthuynh/white-label/internal/booking/adapter/grpcapi"
	"github.com/nhuthuynh/white-label/internal/booking/app"
	"github.com/nhuthuynh/white-label/internal/booking/domain"
	bookingv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/booking/v1"
)

const (
	clubUser   = 3 // holds the `club` role in Identity
	playerUser = 4 // a real user who does not
)

// --- port fakes --------------------------------------------------------

// storingBookingRepo is a real-enough port.Repository: unlike
// discount_regression_test.go's fakeBookingRepo (which stores nothing, because
// no discount test needs it to), it keeps bookings and answers
// ListActiveForCourt with genuine overlap matching. That is what lets
// app.Service.CreateBooking's domain.EnsureNoConflict pre-check produce a real
// ErrCourtDoubleBooked, which the partial-success test depends on.
type storingBookingRepo struct {
	bookings map[string]domain.Booking
}

func newStoringBookingRepo() *storingBookingRepo {
	return &storingBookingRepo{bookings: make(map[string]domain.Booking)}
}

func (r *storingBookingRepo) Create(_ context.Context, b domain.Booking) (domain.Booking, error) {
	r.bookings[b.ID] = b
	return b, nil
}

func (r *storingBookingRepo) ListActiveForCourt(_ context.Context, cID string, rng domain.TimeRange) ([]domain.Booking, error) {
	var out []domain.Booking
	for _, b := range r.bookings {
		if b.CourtID != cID || b.Status == domain.StatusCancelled {
			continue
		}
		if b.Range.Overlaps(rng) {
			out = append(out, b)
		}
	}
	return out, nil
}

func (r *storingBookingRepo) GetByID(_ context.Context, id string) (domain.Booking, error) {
	b, ok := r.bookings[id]
	if !ok {
		return domain.Booking{}, domain.ErrBookingNotFound
	}
	return b, nil
}

func (r *storingBookingRepo) Update(_ context.Context, b domain.Booking) (domain.Booking, error) {
	r.bookings[b.ID] = b
	return b, nil
}

type fakeRecurringRepo struct {
	templates   map[string]domain.RecurringHireTemplate
	createCalls int
	updateCalls int
	listCalls   int
}

func newFakeRecurringRepo() *fakeRecurringRepo {
	return &fakeRecurringRepo{templates: make(map[string]domain.RecurringHireTemplate)}
}

func (r *fakeRecurringRepo) Create(_ context.Context, t domain.RecurringHireTemplate) (domain.RecurringHireTemplate, error) {
	r.createCalls++
	r.templates[t.ID] = t
	return t, nil
}

func (r *fakeRecurringRepo) GetByID(_ context.Context, id string) (domain.RecurringHireTemplate, error) {
	t, ok := r.templates[id]
	if !ok {
		return domain.RecurringHireTemplate{}, domain.ErrRecurringHireTemplateNotFound
	}
	return t, nil
}

func (r *fakeRecurringRepo) UpdateStatus(_ context.Context, t domain.RecurringHireTemplate) (domain.RecurringHireTemplate, error) {
	r.updateCalls++
	r.templates[t.ID] = t
	return t, nil
}

func (r *fakeRecurringRepo) ListForCourts(_ context.Context, courtIDs []string) ([]domain.RecurringHireTemplate, error) {
	r.listCalls++
	wanted := make(map[string]bool, len(courtIDs))
	for _, id := range courtIDs {
		wanted[id] = true
	}
	out := make([]domain.RecurringHireTemplate, 0, len(r.templates))
	for _, t := range r.templates {
		if wanted[t.CourtID] {
			out = append(out, t)
		}
	}
	return out, nil
}

func (r *fakeRecurringRepo) ListForRequester(_ context.Context, requestedByUserID string) ([]domain.RecurringHireTemplate, error) {
	r.listCalls++
	out := make([]domain.RecurringHireTemplate, 0, len(r.templates))
	for _, t := range r.templates {
		if t.RequestedByUserID == requestedByUserID {
			out = append(out, t)
		}
	}
	return out, nil
}

// fakeIdentityLookup stands in for internal/booking/adapter/identity,
// returning the same two Booking-local sentinels that adapter translates
// Identity's errors into. userID(clubUser) holds `club`; userID(playerUser)
// and the two facility actors do not.
type fakeIdentityLookup struct {
	clubs map[string]bool
	known map[string]bool
	calls int
}

func newFakeIdentityLookup() *fakeIdentityLookup {
	return &fakeIdentityLookup{
		clubs: map[string]bool{userID(clubUser): true},
		known: map[string]bool{
			userID(clubUser):     true,
			userID(playerUser):   true,
			userID(ownerUser):    true,
			userID(attackerUser): true,
		},
	}
}

func (l *fakeIdentityLookup) EnsureClubRole(_ context.Context, actorUserID string) error {
	l.calls++
	if !l.known[actorUserID] {
		return domain.ErrUserNotFound
	}
	if !l.clubs[actorUserID] {
		return domain.ErrNotClub
	}
	return nil
}

// --- harness -----------------------------------------------------------

type recurringHarness struct {
	handler   *grpcapi.Handler
	templates *fakeRecurringRepo
	bookings  *storingBookingRepo
	identity  *fakeIdentityLookup
}

// newRecurringHandler wires the real app.Service and the real grpcapi.Handler
// — exactly what cmd/server wires in production — against the fakes above.
// courtID(1) belongs to facilityID(1), owned by userID(ownerUser).
func newRecurringHandler() *recurringHarness {
	templates := newFakeRecurringRepo()
	bookings := newStoringBookingRepo()
	identity := newFakeIdentityLookup()
	svc := app.NewService(bookings, &fakePricingRepo{}, &fakeDiscountRepo{byFacility: map[string][]domain.DiscountRule{}},
		templates, fakeFacilityLookup{}, identity, &fakeIDs{})
	return &recurringHarness{
		handler: grpcapi.NewHandler(svc), templates: templates,
		bookings: bookings, identity: identity,
	}
}

// validRequest is a Monday 09:00-10:00 standing slot for four weeks from
// Monday 2026-01-05.
func validRequest(occurrences int32) *bookingv1.RequestRecurringHireRequest {
	return &bookingv1.RequestRecurringHireRequest{
		ActorUserId: userID(clubUser),
		CourtId:     courtID(1),
		Weekday:     int32(time.Monday),
		StartMinute: 9 * 60,
		EndMinute:   10 * 60,
		StartsAt:    timestamppb.New(time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)),
		EndCondition: &bookingv1.RecurringHireEndCondition{
			Kind:        bookingv1.RecurringHireEndConditionKind_RECURRING_HIRE_END_CONDITION_KIND_END_AFTER_OCCURRENCES,
			Occurrences: occurrences,
		},
	}
}

func mustRequestTemplate(t *testing.T, h *recurringHarness, occurrences int32) *bookingv1.RecurringHireTemplate {
	t.Helper()
	resp, err := h.handler.RequestRecurringHire(context.Background(), validRequest(occurrences))
	if err != nil {
		t.Fatalf("RequestRecurringHire: %v", err)
	}
	return resp.GetTemplate()
}

// --- RequestRecurringHire: the creation-RPC checklist -------------------

// TestRequestRecurringHire_ClubActorSucceeds is the positive control the
// rejection tests below need (without it, a handler that rejected everyone
// would pass every authz assertion), and it pins checklist item 1: the id on
// the response is the server-minted one, because
// RequestRecurringHireRequest has no id field to squat.
func TestRequestRecurringHire_ClubActorSucceeds(t *testing.T) {
	h := newRecurringHandler()

	resp, err := h.handler.RequestRecurringHire(context.Background(), validRequest(4))
	if err != nil {
		t.Fatalf("RequestRecurringHire(club): %v", err)
	}
	tpl := resp.GetTemplate()
	if tpl.GetId() != "00000000-0000-4000-a000-000000000001" {
		t.Errorf("id = %q, want the server-minted id — RequestRecurringHireRequest has no id field to squat", tpl.GetId())
	}
	if tpl.GetStatus() != bookingv1.RecurringHireStatus_RECURRING_HIRE_STATUS_REQUESTED {
		t.Errorf("status = %v, want REQUESTED", tpl.GetStatus())
	}
	if tpl.GetStartMinute() != 9*60 || tpl.GetEndMinute() != 10*60 {
		t.Errorf("start/end minute = %d/%d, want 540/600", tpl.GetStartMinute(), tpl.GetEndMinute())
	}
	if h.templates.createCalls != 1 {
		t.Errorf("repository Create called %d times, want 1", h.templates.createCalls)
	}
}

// TestRequestRecurringHire_NonClubActorIsPermissionDenied is the
// creation-RPC checklist's item 2 at the wire — **the item that fires for
// this ticket** (T10 retro finding 3, sprint plan A4). A real, known user who
// does not hold `club` is refused, and the refusal happens before anything is
// persisted. The companion fact this test relies on is structural rather than
// assertable: RequestRecurringHireRequest has no is_club/role field at all, so
// there is nothing for this caller to self-declare in order to pass.
func TestRequestRecurringHire_NonClubActorIsPermissionDenied(t *testing.T) {
	h := newRecurringHandler()

	req := validRequest(4)
	req.ActorUserId = userID(playerUser)

	_, err := h.handler.RequestRecurringHire(context.Background(), req)
	requireCode(t, err, codes.PermissionDenied)
	if h.templates.createCalls != 0 {
		t.Fatalf("a non-club actor's template reached the repository (%d calls) — the role check must run before anything is persisted", h.templates.createCalls)
	}
}

// TestRequestRecurringHire_UnresolvableActorIsPermissionDenied pins the
// deliberate choice not to answer NotFound for an actor id that resolves to no
// User: on an unauthenticated endpoint that would be a user-enumeration
// oracle. Both shapes — well-formed-but-unknown and malformed — get the same
// code as a known non-club actor, so the response reveals nothing about which
// ids exist.
func TestRequestRecurringHire_UnresolvableActorIsPermissionDenied(t *testing.T) {
	for _, id := range []string{
		userID(99),
		"",
		"not-a-uuid",
		"{6ba7b810-9dad-11d1-80b4-00c04fd430c8}",
		"urn:uuid:6ba7b810-9dad-11d1-80b4-00c04fd430c8",
	} {
		t.Run(id, func(t *testing.T) {
			h := newRecurringHandler()

			req := validRequest(4)
			req.ActorUserId = id

			_, err := h.handler.RequestRecurringHire(context.Background(), req)
			requireCode(t, err, codes.PermissionDenied)
			if h.templates.createCalls != 0 {
				t.Fatalf("an unresolvable actor's template reached the repository (%d calls)", h.templates.createCalls)
			}
		})
	}
}

// TestRequestRecurringHire_UnresolvableCourtIsNotFound covers the ticket's
// NotFound mapping for the CourtID, including the malformed shapes that must
// never reach a `uuid` column.
func TestRequestRecurringHire_UnresolvableCourtIsNotFound(t *testing.T) {
	for _, id := range []string{
		courtID(77),
		"",
		"not-a-uuid",
		"'; DROP TABLE recurring_hire_templates;--",
		"{6ba7b810-9dad-11d1-80b4-00c04fd430c8}",
	} {
		t.Run(id, func(t *testing.T) {
			h := newRecurringHandler()

			req := validRequest(4)
			req.CourtId = id

			_, err := h.handler.RequestRecurringHire(context.Background(), req)
			requireCode(t, err, codes.NotFound)
			if h.templates.createCalls != 0 {
				t.Fatalf("an unresolvable court's template reached the repository (%d calls)", h.templates.createCalls)
			}
		})
	}
}

// TestRequestRecurringHire_InvalidScheduleIsInvalidArgument walks every way a
// schedule can be malformed and requires InvalidArgument for each — none may
// fall through to the default Internal branch, which would report a caller's
// mistake as a server fault.
func TestRequestRecurringHire_InvalidScheduleIsInvalidArgument(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*bookingv1.RequestRecurringHireRequest)
	}{
		{
			name: "start minute not before end minute",
			mutate: func(r *bookingv1.RequestRecurringHireRequest) {
				r.StartMinute, r.EndMinute = r.EndMinute, r.StartMinute
			},
		},
		{
			name:   "start equals end",
			mutate: func(r *bookingv1.RequestRecurringHireRequest) { r.EndMinute = r.StartMinute },
		},
		{
			name:   "minute of day out of range",
			mutate: func(r *bookingv1.RequestRecurringHireRequest) { r.EndMinute = 24 * 60 },
		},
		{
			name:   "negative minute of day",
			mutate: func(r *bookingv1.RequestRecurringHireRequest) { r.StartMinute = -1 },
		},
		{
			name:   "weekday out of range",
			mutate: func(r *bookingv1.RequestRecurringHireRequest) { r.Weekday = 9 },
		},
		{
			name: "end after zero occurrences",
			mutate: func(r *bookingv1.RequestRecurringHireRequest) {
				r.EndCondition.Occurrences = 0
			},
		},
		{
			name: "end after negative occurrences",
			mutate: func(r *bookingv1.RequestRecurringHireRequest) {
				r.EndCondition.Occurrences = -5
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			h := newRecurringHandler()

			req := validRequest(4)
			tt.mutate(req)

			_, err := h.handler.RequestRecurringHire(context.Background(), req)
			requireCode(t, err, codes.InvalidArgument)
			if h.templates.createCalls != 0 {
				t.Fatalf("an invalid template reached the repository (%d calls)", h.templates.createCalls)
			}
		})
	}
}

// --- ApproveRecurringHire ----------------------------------------------

// TestApproveRecurringHire_PartialConflictIsReportedNotFatal is this ticket's
// headline AC at the wire. One of four weeks is already booked; approval must
// still succeed, the template must still come back APPROVED, and the response
// must say which weeks were booked and which was skipped and why.
//
// An implementation that aborted on the first conflict would fail on the error
// alone; one that silently dropped the conflicted week would pass that but
// fail the per-occurrence assertions below. Both are checked, because a UI
// that showed a Club a confirmed standing booking for a week it does not hold
// is the actual harm here.
func TestApproveRecurringHire_PartialConflictIsReportedNotFatal(t *testing.T) {
	ctx := context.Background()
	h := newRecurringHandler()
	tpl := mustRequestTemplate(t, h, 4)

	// Occupy the third occurrence (Monday 2026-01-19 09:00-10:00) with an
	// ordinary individual booking, through the real CreateBooking RPC.
	if _, err := h.handler.CreateBooking(ctx, &bookingv1.CreateBookingRequest{
		CourtId:  courtID(1),
		Source:   bookingv1.Source_SOURCE_INDIVIDUAL,
		StartsAt: timestamppb.New(time.Date(2026, 1, 19, 9, 0, 0, 0, time.UTC)),
		EndsAt:   timestamppb.New(time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC)),
	}); err != nil {
		t.Fatalf("fixture CreateBooking: %v", err)
	}

	resp, err := h.handler.ApproveRecurringHire(ctx, &bookingv1.ApproveRecurringHireRequest{
		TemplateId:  tpl.GetId(),
		ActorUserId: userID(ownerUser),
	})
	if err != nil {
		t.Fatalf("ApproveRecurringHire returned %v; one already-booked week must not fail the whole approval", err)
	}

	if got := resp.GetTemplate().GetStatus(); got != bookingv1.RecurringHireStatus_RECURRING_HIRE_STATUS_APPROVED {
		t.Errorf("template status = %v, want APPROVED", got)
	}

	occ := resp.GetOccurrences()
	if len(occ) != 4 {
		t.Fatalf("got %d occurrences, want 4 — every occurrence must be reported, booked or not", len(occ))
	}

	want := []bookingv1.RecurringHireOccurrenceOutcome{
		bookingv1.RecurringHireOccurrenceOutcome_RECURRING_HIRE_OCCURRENCE_OUTCOME_BOOKED,
		bookingv1.RecurringHireOccurrenceOutcome_RECURRING_HIRE_OCCURRENCE_OUTCOME_BOOKED,
		bookingv1.RecurringHireOccurrenceOutcome_RECURRING_HIRE_OCCURRENCE_OUTCOME_SKIPPED_CONFLICT,
		bookingv1.RecurringHireOccurrenceOutcome_RECURRING_HIRE_OCCURRENCE_OUTCOME_BOOKED,
	}
	for i, w := range want {
		if occ[i].GetOutcome() != w {
			t.Errorf("occurrence %d (%s) outcome = %v, want %v",
				i, occ[i].GetStartsAt().AsTime().Format(time.RFC3339), occ[i].GetOutcome(), w)
		}
	}

	if got := occ[2].GetStartsAt().AsTime(); !got.Equal(time.Date(2026, 1, 19, 9, 0, 0, 0, time.UTC)) {
		t.Errorf("skipped occurrence starts at %s, want the already-booked 2026-01-19T09:00Z", got)
	}
	if occ[2].GetReason() == "" {
		t.Error("the skipped occurrence carries no reason — the response must say why a week was skipped")
	}
	if occ[2].GetBookingId() != "" {
		t.Errorf("the skipped occurrence carries booking id %q, want none", occ[2].GetBookingId())
	}
	for _, i := range []int{0, 1, 3} {
		if occ[i].GetBookingId() == "" {
			t.Errorf("occurrence %d was booked but carries no booking id", i)
		}
		if occ[i].GetReason() != "" {
			t.Errorf("occurrence %d was booked but carries reason %q", i, occ[i].GetReason())
		}
	}

	// The successful weeks are real SOURCE_RECURRING_HIRE Bookings on the
	// court, readable through the ordinary list endpoint — not just entries
	// in an approval response.
	list, err := h.handler.ListCourtBookings(ctx, &bookingv1.ListCourtBookingsRequest{
		CourtId: courtID(1),
		From:    timestamppb.New(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)),
		To:      timestamppb.New(time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)),
	})
	if err != nil {
		t.Fatalf("ListCourtBookings: %v", err)
	}
	recurring := 0
	for _, b := range list.GetBookings() {
		if b.GetSource() != bookingv1.Source_SOURCE_RECURRING_HIRE {
			continue
		}
		recurring++
		if b.GetReferenceId() != tpl.GetId() {
			t.Errorf("recurring booking %q references %q, want the template %q", b.GetId(), b.GetReferenceId(), tpl.GetId())
		}
	}
	if recurring != 3 {
		t.Errorf("got %d recurring-hire bookings on the court, want 3", recurring)
	}
}

// TestApproveRecurringHire_NonOwnerIsPermissionDenied is the BOLA regression:
// only the owner of the Facility that owns the template's Court may approve.
// The Club that submitted the request is included on purpose — being the
// requester is not the same as being the approver.
func TestApproveRecurringHire_NonOwnerIsPermissionDenied(t *testing.T) {
	ctx := context.Background()

	for _, actor := range []string{userID(attackerUser), userID(clubUser)} {
		t.Run(actor, func(t *testing.T) {
			h := newRecurringHandler()
			tpl := mustRequestTemplate(t, h, 4)

			_, err := h.handler.ApproveRecurringHire(ctx, &bookingv1.ApproveRecurringHireRequest{
				TemplateId:  tpl.GetId(),
				ActorUserId: actor,
			})
			requireCode(t, err, codes.PermissionDenied)
			if h.templates.updateCalls != 0 {
				t.Errorf("a non-owner's approval was persisted (%d update calls) — the ownership check must run first", h.templates.updateCalls)
			}
			if got := len(h.bookings.bookings); got != 0 {
				t.Errorf("a non-owner's approval created %d bookings, want 0", got)
			}
		})
	}
}

// TestApproveRecurringHire_UnknownTemplateIsNotFound covers the ticket's
// NotFound mapping for the template id, both shapes.
func TestApproveRecurringHire_UnknownTemplateIsNotFound(t *testing.T) {
	for _, id := range []string{
		"00000000-0000-4000-a000-000000000077",
		"",
		"not-a-uuid",
		"urn:uuid:6ba7b810-9dad-11d1-80b4-00c04fd430c8",
	} {
		t.Run(id, func(t *testing.T) {
			h := newRecurringHandler()

			_, err := h.handler.ApproveRecurringHire(context.Background(), &bookingv1.ApproveRecurringHireRequest{
				TemplateId:  id,
				ActorUserId: userID(ownerUser),
			})
			requireCode(t, err, codes.NotFound)
		})
	}
}

// TestRecurringHire_AlreadyDecidedIsFailedPrecondition pins the ticket's
// FailedPrecondition mapping across all four already-decided combinations, and
// proves re-approval does not quietly generate a second set of Bookings for
// the same weeks.
func TestRecurringHire_AlreadyDecidedIsFailedPrecondition(t *testing.T) {
	ctx := context.Background()

	t.Run("re-approving an approved template", func(t *testing.T) {
		h := newRecurringHandler()
		tpl := mustRequestTemplate(t, h, 2)

		if _, err := h.handler.ApproveRecurringHire(ctx, &bookingv1.ApproveRecurringHireRequest{
			TemplateId: tpl.GetId(), ActorUserId: userID(ownerUser),
		}); err != nil {
			t.Fatalf("first approval: %v", err)
		}
		afterFirst := len(h.bookings.bookings)

		_, err := h.handler.ApproveRecurringHire(ctx, &bookingv1.ApproveRecurringHireRequest{
			TemplateId: tpl.GetId(), ActorUserId: userID(ownerUser),
		})
		requireCode(t, err, codes.FailedPrecondition)
		if got := len(h.bookings.bookings); got != afterFirst {
			t.Errorf("re-approving created %d extra bookings, want 0", got-afterFirst)
		}
	})

	t.Run("rejecting an approved template", func(t *testing.T) {
		h := newRecurringHandler()
		tpl := mustRequestTemplate(t, h, 2)

		if _, err := h.handler.ApproveRecurringHire(ctx, &bookingv1.ApproveRecurringHireRequest{
			TemplateId: tpl.GetId(), ActorUserId: userID(ownerUser),
		}); err != nil {
			t.Fatalf("approval: %v", err)
		}

		_, err := h.handler.RejectRecurringHire(ctx, &bookingv1.RejectRecurringHireRequest{
			TemplateId: tpl.GetId(), ActorUserId: userID(ownerUser),
		})
		requireCode(t, err, codes.FailedPrecondition)
	})

	t.Run("approving a rejected template", func(t *testing.T) {
		h := newRecurringHandler()
		tpl := mustRequestTemplate(t, h, 2)

		if _, err := h.handler.RejectRecurringHire(ctx, &bookingv1.RejectRecurringHireRequest{
			TemplateId: tpl.GetId(), ActorUserId: userID(ownerUser),
		}); err != nil {
			t.Fatalf("rejection: %v", err)
		}

		_, err := h.handler.ApproveRecurringHire(ctx, &bookingv1.ApproveRecurringHireRequest{
			TemplateId: tpl.GetId(), ActorUserId: userID(ownerUser),
		})
		requireCode(t, err, codes.FailedPrecondition)
		if got := len(h.bookings.bookings); got != 0 {
			t.Errorf("approving a rejected template created %d bookings, want 0", got)
		}
	})
}

// --- RejectRecurringHire -----------------------------------------------

func TestRejectRecurringHire(t *testing.T) {
	ctx := context.Background()

	t.Run("owner rejects and no bookings are created", func(t *testing.T) {
		h := newRecurringHandler()
		tpl := mustRequestTemplate(t, h, 4)

		resp, err := h.handler.RejectRecurringHire(ctx, &bookingv1.RejectRecurringHireRequest{
			TemplateId: tpl.GetId(), ActorUserId: userID(ownerUser),
		})
		if err != nil {
			t.Fatalf("RejectRecurringHire: %v", err)
		}
		if got := resp.GetTemplate().GetStatus(); got != bookingv1.RecurringHireStatus_RECURRING_HIRE_STATUS_REJECTED {
			t.Errorf("status = %v, want REJECTED", got)
		}
		if got := len(h.bookings.bookings); got != 0 {
			t.Errorf("rejection created %d bookings, want 0", got)
		}
	})

	t.Run("non-owner is permission denied", func(t *testing.T) {
		h := newRecurringHandler()
		tpl := mustRequestTemplate(t, h, 4)

		_, err := h.handler.RejectRecurringHire(ctx, &bookingv1.RejectRecurringHireRequest{
			TemplateId: tpl.GetId(), ActorUserId: userID(attackerUser),
		})
		requireCode(t, err, codes.PermissionDenied)
		if h.templates.updateCalls != 0 {
			t.Errorf("a non-owner's rejection was persisted (%d update calls)", h.templates.updateCalls)
		}
	})
}

// --- ListRecurringHireTemplatesForFacility -----------------------------

// TestListRecurringHireTemplatesForFacility reads back what
// RequestRecurringHire wrote, and pins this endpoint's owner-only
// authorization — the deliberate difference from T11.2's unauthenticated
// ListDiscountRulesForFacility, recorded in the RPC's own proto comment.
func TestListRecurringHireTemplatesForFacility(t *testing.T) {
	ctx := context.Background()

	t.Run("owner sees the queue", func(t *testing.T) {
		h := newRecurringHandler()
		tpl := mustRequestTemplate(t, h, 4)

		resp, err := h.handler.ListRecurringHireTemplatesForFacility(ctx, &bookingv1.ListRecurringHireTemplatesForFacilityRequest{
			FacilityId: facilityID(1), ActorUserId: userID(ownerUser),
		})
		if err != nil {
			t.Fatalf("ListRecurringHireTemplatesForFacility: %v", err)
		}
		if len(resp.GetTemplates()) != 1 || resp.GetTemplates()[0].GetId() != tpl.GetId() {
			t.Fatalf("got %v, want exactly the template %q", resp.GetTemplates(), tpl.GetId())
		}
	})

	t.Run("non-owner is permission denied", func(t *testing.T) {
		h := newRecurringHandler()
		mustRequestTemplate(t, h, 4)

		_, err := h.handler.ListRecurringHireTemplatesForFacility(ctx, &bookingv1.ListRecurringHireTemplatesForFacilityRequest{
			FacilityId: facilityID(1), ActorUserId: userID(attackerUser),
		})
		requireCode(t, err, codes.PermissionDenied)
		if h.templates.listCalls != 0 {
			t.Errorf("a non-owner's list reached the repository (%d calls)", h.templates.listCalls)
		}
	})

	t.Run("unknown or malformed facility is not found", func(t *testing.T) {
		for _, id := range []string{facilityID(77), "", "not-a-uuid"} {
			h := newRecurringHandler()
			_, err := h.handler.ListRecurringHireTemplatesForFacility(ctx, &bookingv1.ListRecurringHireTemplatesForFacilityRequest{
				FacilityId: id, ActorUserId: userID(ownerUser),
			})
			requireCode(t, err, codes.NotFound)
		}
	})
}

// --- ListRecurringHireTemplatesForActor --------------------------------

// TestListRecurringHireTemplatesForActor pins the Club-facing status view
// T11.6 adds at the wire boundary: the actor's own templates, in every
// status, with no error surfaced for an actor who simply has none.
//
// The rejected-template subtest is the "no fabricated data" AC in test form:
// a rejected request must come back present and REJECTED. Omitting it — the
// obvious alternative implementation, filtering the list to open requests —
// would let the Club's screen imply a decided request is still pending.
func TestListRecurringHireTemplatesForActor(t *testing.T) {
	ctx := context.Background()

	t.Run("actor sees their own templates and nobody else's", func(t *testing.T) {
		h := newRecurringHandler()
		mine := mustRequestTemplate(t, h, 4)

		resp, err := h.handler.ListRecurringHireTemplatesForActor(ctx, &bookingv1.ListRecurringHireTemplatesForActorRequest{
			ActorUserId: userID(clubUser),
		})
		if err != nil {
			t.Fatalf("ListRecurringHireTemplatesForActor: %v", err)
		}
		if len(resp.GetTemplates()) != 1 || resp.GetTemplates()[0].GetId() != mine.GetId() {
			t.Fatalf("got %v, want exactly the actor's own template %q", resp.GetTemplates(), mine.GetId())
		}

		// The Facility Owner requested nothing, so their list is empty even
		// though they can see this same template through the owner queue.
		ownerResp, err := h.handler.ListRecurringHireTemplatesForActor(ctx, &bookingv1.ListRecurringHireTemplatesForActorRequest{
			ActorUserId: userID(ownerUser),
		})
		if err != nil {
			t.Fatalf("ListRecurringHireTemplatesForActor (owner): %v", err)
		}
		if len(ownerResp.GetTemplates()) != 0 {
			t.Errorf("owner's own-request list = %v, want empty", ownerResp.GetTemplates())
		}
	})

	t.Run("a rejected template is returned as REJECTED, not omitted", func(t *testing.T) {
		h := newRecurringHandler()
		tpl := mustRequestTemplate(t, h, 4)

		if _, err := h.handler.RejectRecurringHire(ctx, &bookingv1.RejectRecurringHireRequest{
			TemplateId: tpl.GetId(), ActorUserId: userID(ownerUser),
		}); err != nil {
			t.Fatalf("RejectRecurringHire: %v", err)
		}

		resp, err := h.handler.ListRecurringHireTemplatesForActor(ctx, &bookingv1.ListRecurringHireTemplatesForActorRequest{
			ActorUserId: userID(clubUser),
		})
		if err != nil {
			t.Fatalf("ListRecurringHireTemplatesForActor: %v", err)
		}
		if len(resp.GetTemplates()) != 1 {
			t.Fatalf("got %d templates, want the rejected one still listed", len(resp.GetTemplates()))
		}
		if got := resp.GetTemplates()[0].GetStatus(); got != bookingv1.RecurringHireStatus_RECURRING_HIRE_STATUS_REJECTED {
			t.Errorf("status = %v, want REJECTED", got)
		}
	})

	// An actor with no templates — unknown, malformed, or simply new — is an
	// ordinary empty list, not an error and not a panic (the malformed values
	// would otherwise reach the adapter's mustUUID).
	t.Run("unknown or malformed actor is an empty list, not an error", func(t *testing.T) {
		for _, id := range []string{userID(99), "", "not-a-uuid", "urn:uuid:6ba7b810-9dad-11d1-80b4-00c04fd430c8"} {
			h := newRecurringHandler()
			mustRequestTemplate(t, h, 4)

			resp, err := h.handler.ListRecurringHireTemplatesForActor(ctx, &bookingv1.ListRecurringHireTemplatesForActorRequest{
				ActorUserId: id,
			})
			if err != nil {
				t.Fatalf("ListRecurringHireTemplatesForActor(%q): %v", id, err)
			}
			if len(resp.GetTemplates()) != 0 {
				t.Errorf("ListRecurringHireTemplatesForActor(%q) = %v, want empty", id, resp.GetTemplates())
			}
		}
	})
}

// --- horizon -----------------------------------------------------------

// TestApproveRecurringHire_OpenEndedTemplateIsBoundedByTheHorizon pins the
// generation cap. An open-ended ("permanent club rate") template must not
// produce an unbounded set of Bookings on approval; the app layer applies a
// one-year horizon from the template's StartsAt. 2026-01-05 is a Monday, and
// the horizon runs to 2027-01-05 exclusive; 2026-01-05 plus 52 weeks is
// 2027-01-04, which still falls inside it, so a full year yields 53 Mondays
// rather than 52.
//
// This is the one behaviour T11.5's ticket text did not specify, so it is
// pinned here rather than left to be discovered by whoever first approves an
// open-ended request in production.
func TestApproveRecurringHire_OpenEndedTemplateIsBoundedByTheHorizon(t *testing.T) {
	ctx := context.Background()
	h := newRecurringHandler()

	req := validRequest(0)
	req.EndCondition = &bookingv1.RecurringHireEndCondition{
		Kind: bookingv1.RecurringHireEndConditionKind_RECURRING_HIRE_END_CONDITION_KIND_NO_END,
	}
	created, err := h.handler.RequestRecurringHire(ctx, req)
	if err != nil {
		t.Fatalf("RequestRecurringHire: %v", err)
	}

	resp, err := h.handler.ApproveRecurringHire(ctx, &bookingv1.ApproveRecurringHireRequest{
		TemplateId: created.GetTemplate().GetId(), ActorUserId: userID(ownerUser),
	})
	if err != nil {
		t.Fatalf("ApproveRecurringHire: %v", err)
	}

	if got := len(resp.GetOccurrences()); got != 53 {
		t.Errorf("open-ended template generated %d occurrences, want 53 (the Mondays from 2026-01-05 up to the 2027-01-05 horizon)", got)
	}
	last := resp.GetOccurrences()[len(resp.GetOccurrences())-1].GetStartsAt().AsTime()
	if !last.Before(time.Date(2027, 1, 5, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("last occurrence %s is not inside the one-year horizon", last)
	}
	if !last.Equal(time.Date(2027, 1, 4, 9, 0, 0, 0, time.UTC)) {
		t.Errorf("last occurrence = %s, want 2027-01-04T09:00Z — the last Monday inside the horizon", last)
	}
}
