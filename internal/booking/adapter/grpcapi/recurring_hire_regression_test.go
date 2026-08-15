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
//
// knownCourts MODELS THE FOREIGN KEY, deliberately and explicitly (T15.6,
// issue #185). bookings.court_id is `uuid NOT NULL REFERENCES courts (id)`
// (db/migrations/0001_init.sql:24), so a well-formed UUID naming no courts
// row passes app.Service.CreateBooking's uuidShape guard — a shape check and
// nothing more — and is rejected only at the INSERT, as Postgres 23503, which
// adapter/postgres.translateErr turns into domain.ErrInvalidCourtReference
// (proved against that code path in
// internal/booking/adapter/postgres/foreign_key_test.go, and against a real
// Postgres in that package's unknown_court_integration_test.go).
//
// A fake without this check accepts every well-formed court id and reports
// green whether or not the bug exists — the fixture infidelity that hid #185
// for two sprints (docs/LESSONS.md' T9 entry). knownCourts is nil by default
// here, which means "every court exists", because most tests in this package
// are about something else entirely and courtID(1..n) are all legitimate
// fixtures to them; unknown_court_test.go opts in via seedKnownCourts. That
// default is a deliberate, documented choice and NOT an oversight: see
// seedKnownCourts' comment for why the opt-in direction is safe here and the
// opt-out direction was chosen in the socialplay/competitions twins.
type storingBookingRepo struct {
	bookings    map[string]domain.Booking
	knownCourts map[string]bool
}

func newStoringBookingRepo() *storingBookingRepo {
	return &storingBookingRepo{bookings: make(map[string]domain.Booking)}
}

// seedKnownCourts switches this fake from "every court exists" to modelling
// the bookings.court_id foreign key against exactly the listed courts.
//
// The opt-IN direction is safe in this package specifically because the
// mapping under test at this layer is domain sentinel -> gRPC code, and every
// other test here supplies courtID(n) fixtures that stand for real courts. In
// internal/socialplay and internal/competitions' grpcapi twins the FK is
// modelled by DEFAULT instead, because those packages hold the RPCs #185 was
// actually reported against and a new test there must not be able to get the
// naive behaviour by forgetting.
func (r *storingBookingRepo) seedKnownCourts(ids ...string) *storingBookingRepo {
	r.knownCourts = make(map[string]bool, len(ids))
	for _, id := range ids {
		r.knownCourts[id] = true
	}
	return r
}

func (r *storingBookingRepo) Create(_ context.Context, b domain.Booking) (domain.Booking, error) {
	// The FK, modelled — see knownCourts' comment on the struct. A nil map
	// means the FK is not modelled and every court is treated as real.
	if r.knownCourts != nil && !r.knownCourts[b.CourtID] {
		return domain.Booking{}, domain.ErrInvalidCourtReference
	}
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
//
// It models BOTH identifier spaces ADR-0014 distinguishes, because a fake
// that collapsed them would reintroduce the blind spot #146 and #152 came
// through: `subjects` maps subjectOf(n) -> userID(n) for UserIDBySubject, and
// `clubs`/`known` are keyed on the uuid for EnsureClubRole. A subject handed
// to EnsureClubRole here resolves to nothing, exactly as the real adapter's
// GetUser refuses it.
//
// The genuinely unmocked end-to-end coverage of this seam is
// subject_actor_seam_test.go, which drives the real adapter over the real
// Identity service. This fake exists so the ~40 authorization cases in this
// package do not each need an Identity fixture.
type fakeIdentityLookup struct {
	clubs    map[string]bool
	known    map[string]bool
	subjects map[string]string
	calls    int
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
		subjects: map[string]string{
			subjectOf(clubUser):     userID(clubUser),
			subjectOf(playerUser):   userID(playerUser),
			subjectOf(ownerUser):    userID(ownerUser),
			subjectOf(attackerUser): userID(attackerUser),
		},
	}
}

// UserIDBySubject is ADR-0014's translation, faked: the subject the token
// carries in, the User.ID this platform stores out. An unregistered subject
// is domain.ErrUserNotFound, which the handler maps to PermissionDenied.
func (l *fakeIdentityLookup) UserIDBySubject(_ context.Context, subject string) (string, error) {
	id, ok := l.subjects[subject]
	if !ok {
		return "", domain.ErrUserNotFound
	}
	return id, nil
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
	svc := app.NewService(app.ServiceOptions{
		Bookings:       bookings,
		PricingRules:   &fakePricingRepo{},
		DiscountRules:  &fakeDiscountRepo{byFacility: map[string][]domain.DiscountRule{}},
		RecurringHires: templates,
		Facilities:     fakeFacilityLookup{},
		Identity:       identity,
		IDs:            &fakeIDs{},
	})
	return &recurringHarness{
		handler: grpcapi.NewHandler(svc), templates: templates,
		bookings: bookings, identity: identity,
	}
}

// validRequest is a Monday 09:00-10:00 standing slot for four weeks from
// Monday 2026-01-05.
func validRequest(occurrences int32) *bookingv1.RequestRecurringHireRequest {
	return &bookingv1.RequestRecurringHireRequest{
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
	resp, err := h.handler.RequestRecurringHire(ctxAs(subjectOf(clubUser)), validRequest(occurrences))
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

	resp, err := h.handler.RequestRecurringHire(ctxAs(subjectOf(clubUser)), validRequest(4))
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

	// T12.7: the non-club actor is now a *verified* non-club user. Before
	// this ticket the identity was req.ActorUserId, which the handler took at
	// face value; the role check it feeds is unchanged.
	_, err := h.handler.RequestRecurringHire(ctxAs(subjectOf(playerUser)), req)
	requireCode(t, err, codes.PermissionDenied)
	if h.templates.createCalls != 0 {
		t.Fatalf("a non-club actor's template reached the repository (%d calls) — the role check must run before anything is persisted", h.templates.createCalls)
	}
}

// TestRequestRecurringHire_UnresolvableActorIsPermissionDenied pins the
// deliberate choice not to answer NotFound for an actor that resolves to no
// User: that would be a user-enumeration oracle, and it would make a 404
// ambiguous between "your Court does not exist" and "you do not exist". Every
// shape gets the same code as a known non-club actor, so the response reveals
// nothing about which callers are registered.
//
// T12.7 note: the empty-string case this loop used to carry has moved. An
// empty subject is not an identity — auth.ContextWithPrincipal refuses to
// store one — so that caller is now Unauthenticated, not PermissionDenied, and
// is asserted in authz_regression_test.go's no-principal test.
//
// T13.2/ADR-0014 note: these values are now VERIFIED SUBJECTS that no User is
// registered to, and the rejection moved one layer up — from the app's
// uuidShape guard to the handler's actor() funnel (ADR-0014 §6). The uuid- and
// urn-shaped entries are kept deliberately even though a real `sub` claim
// would rarely look like that: they pin that the funnel refuses anything not
// in identity_users.subject rather than anything not uuid-shaped, which are
// two very different checks and only one of them is the right one.
func TestRequestRecurringHire_UnresolvableActorIsPermissionDenied(t *testing.T) {
	for _, id := range []string{
		subjectOf(99),
		userID(99),
		"not-a-uuid",
		"{6ba7b810-9dad-11d1-80b4-00c04fd430c8}",
		"urn:uuid:6ba7b810-9dad-11d1-80b4-00c04fd430c8",
	} {
		t.Run(id, func(t *testing.T) {
			h := newRecurringHandler()

			req := validRequest(4)

			_, err := h.handler.RequestRecurringHire(ctxAs(id), req)
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

			_, err := h.handler.RequestRecurringHire(ctxAs(subjectOf(clubUser)), req)
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

			_, err := h.handler.RequestRecurringHire(ctxAs(subjectOf(clubUser)), req)
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

	resp, err := h.handler.ApproveRecurringHire(ctxAs(subjectOf(ownerUser)), &bookingv1.ApproveRecurringHireRequest{
		TemplateId: tpl.GetId(),
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
	for _, actor := range []string{subjectOf(attackerUser), subjectOf(clubUser)} {
		t.Run(actor, func(t *testing.T) {
			h := newRecurringHandler()
			tpl := mustRequestTemplate(t, h, 4)

			_, err := h.handler.ApproveRecurringHire(ctxAs(actor), &bookingv1.ApproveRecurringHireRequest{
				TemplateId: tpl.GetId(),
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

			_, err := h.handler.ApproveRecurringHire(ctxAs(subjectOf(ownerUser)), &bookingv1.ApproveRecurringHireRequest{
				TemplateId: id,
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
	t.Run("re-approving an approved template", func(t *testing.T) {
		h := newRecurringHandler()
		tpl := mustRequestTemplate(t, h, 2)

		if _, err := h.handler.ApproveRecurringHire(ctxAs(subjectOf(ownerUser)), &bookingv1.ApproveRecurringHireRequest{
			TemplateId: tpl.GetId(),
		}); err != nil {
			t.Fatalf("first approval: %v", err)
		}
		afterFirst := len(h.bookings.bookings)

		_, err := h.handler.ApproveRecurringHire(ctxAs(subjectOf(ownerUser)), &bookingv1.ApproveRecurringHireRequest{
			TemplateId: tpl.GetId(),
		})
		requireCode(t, err, codes.FailedPrecondition)
		if got := len(h.bookings.bookings); got != afterFirst {
			t.Errorf("re-approving created %d extra bookings, want 0", got-afterFirst)
		}
	})

	t.Run("rejecting an approved template", func(t *testing.T) {
		h := newRecurringHandler()
		tpl := mustRequestTemplate(t, h, 2)

		if _, err := h.handler.ApproveRecurringHire(ctxAs(subjectOf(ownerUser)), &bookingv1.ApproveRecurringHireRequest{
			TemplateId: tpl.GetId(),
		}); err != nil {
			t.Fatalf("approval: %v", err)
		}

		_, err := h.handler.RejectRecurringHire(ctxAs(subjectOf(ownerUser)), &bookingv1.RejectRecurringHireRequest{
			TemplateId: tpl.GetId(),
		})
		requireCode(t, err, codes.FailedPrecondition)
	})

	t.Run("approving a rejected template", func(t *testing.T) {
		h := newRecurringHandler()
		tpl := mustRequestTemplate(t, h, 2)

		if _, err := h.handler.RejectRecurringHire(ctxAs(subjectOf(ownerUser)), &bookingv1.RejectRecurringHireRequest{
			TemplateId: tpl.GetId(),
		}); err != nil {
			t.Fatalf("rejection: %v", err)
		}

		_, err := h.handler.ApproveRecurringHire(ctxAs(subjectOf(ownerUser)), &bookingv1.ApproveRecurringHireRequest{
			TemplateId: tpl.GetId(),
		})
		requireCode(t, err, codes.FailedPrecondition)
		if got := len(h.bookings.bookings); got != 0 {
			t.Errorf("approving a rejected template created %d bookings, want 0", got)
		}
	})
}

// --- RejectRecurringHire -----------------------------------------------

func TestRejectRecurringHire(t *testing.T) {
	t.Run("owner rejects and no bookings are created", func(t *testing.T) {
		h := newRecurringHandler()
		tpl := mustRequestTemplate(t, h, 4)

		resp, err := h.handler.RejectRecurringHire(ctxAs(subjectOf(ownerUser)), &bookingv1.RejectRecurringHireRequest{
			TemplateId: tpl.GetId(),
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

		_, err := h.handler.RejectRecurringHire(ctxAs(subjectOf(attackerUser)), &bookingv1.RejectRecurringHireRequest{
			TemplateId: tpl.GetId(),
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
	t.Run("owner sees the queue", func(t *testing.T) {
		h := newRecurringHandler()
		tpl := mustRequestTemplate(t, h, 4)

		resp, err := h.handler.ListRecurringHireTemplatesForFacility(ctxAs(subjectOf(ownerUser)), &bookingv1.ListRecurringHireTemplatesForFacilityRequest{
			FacilityId: facilityID(1),
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

		_, err := h.handler.ListRecurringHireTemplatesForFacility(ctxAs(subjectOf(attackerUser)), &bookingv1.ListRecurringHireTemplatesForFacilityRequest{
			FacilityId: facilityID(1),
		})
		requireCode(t, err, codes.PermissionDenied)
		if h.templates.listCalls != 0 {
			t.Errorf("a non-owner's list reached the repository (%d calls)", h.templates.listCalls)
		}
	})

	t.Run("unknown or malformed facility is not found", func(t *testing.T) {
		for _, id := range []string{facilityID(77), "", "not-a-uuid"} {
			h := newRecurringHandler()
			_, err := h.handler.ListRecurringHireTemplatesForFacility(ctxAs(subjectOf(ownerUser)), &bookingv1.ListRecurringHireTemplatesForFacilityRequest{
				FacilityId: id,
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
	t.Run("actor sees their own templates and nobody else's", func(t *testing.T) {
		h := newRecurringHandler()
		mine := mustRequestTemplate(t, h, 4)

		resp, err := h.handler.ListRecurringHireTemplatesForActor(ctxAs(subjectOf(clubUser)), &bookingv1.ListRecurringHireTemplatesForActorRequest{})
		if err != nil {
			t.Fatalf("ListRecurringHireTemplatesForActor: %v", err)
		}
		if len(resp.GetTemplates()) != 1 || resp.GetTemplates()[0].GetId() != mine.GetId() {
			t.Fatalf("got %v, want exactly the actor's own template %q", resp.GetTemplates(), mine.GetId())
		}

		// The Facility Owner requested nothing, so their list is empty even
		// though they can see this same template through the owner queue.
		ownerResp, err := h.handler.ListRecurringHireTemplatesForActor(ctxAs(subjectOf(ownerUser)), &bookingv1.ListRecurringHireTemplatesForActorRequest{})
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

		if _, err := h.handler.RejectRecurringHire(ctxAs(subjectOf(ownerUser)), &bookingv1.RejectRecurringHireRequest{
			TemplateId: tpl.GetId(),
		}); err != nil {
			t.Fatalf("RejectRecurringHire: %v", err)
		}

		resp, err := h.handler.ListRecurringHireTemplatesForActor(ctxAs(subjectOf(clubUser)), &bookingv1.ListRecurringHireTemplatesForActorRequest{})
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

	// **This case CHANGED in T13.2, deliberately — see ADR-0014 §6.** It used
	// to assert that an unknown or malformed actor got an ordinary empty list
	// rather than an error, because the app layer's uuidShape guard answered
	// an unresolvable actor the same way it answered an actor with no
	// templates.
	//
	// Under ADR-0014 an unresolvable caller no longer reaches the app layer at
	// all: the handler's actor() funnel refuses them with PermissionDenied
	// before the read is dispatched. That is the better answer — an
	// unregistered caller has no templates and no basis to ask — but it is
	// observable, so it is asserted here rather than left to be discovered.
	//
	// The app layer's empty-list branch is unchanged and still covered by
	// internal/booking/app's own tests; it is simply now reachable only by a
	// programming error rather than by a real caller.
	t.Run("a subject registered to no User is PermissionDenied, not an empty list", func(t *testing.T) {
		// The empty id is gone for the reason above: with no storable
		// principal the call is Unauthenticated, which
		// authz_regression_test.go asserts.
		for _, id := range []string{subjectOf(99), userID(99), "not-a-uuid", "urn:uuid:6ba7b810-9dad-11d1-80b4-00c04fd430c8"} {
			h := newRecurringHandler()
			mustRequestTemplate(t, h, 4)

			_, err := h.handler.ListRecurringHireTemplatesForActor(ctxAs(id), &bookingv1.ListRecurringHireTemplatesForActorRequest{})
			requireCode(t, err, codes.PermissionDenied)
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
	h := newRecurringHandler()

	req := validRequest(0)
	req.EndCondition = &bookingv1.RecurringHireEndCondition{
		Kind: bookingv1.RecurringHireEndConditionKind_RECURRING_HIRE_END_CONDITION_KIND_NO_END,
	}
	created, err := h.handler.RequestRecurringHire(ctxAs(subjectOf(clubUser)), req)
	if err != nil {
		t.Fatalf("RequestRecurringHire: %v", err)
	}

	resp, err := h.handler.ApproveRecurringHire(ctxAs(subjectOf(ownerUser)), &bookingv1.ApproveRecurringHireRequest{
		TemplateId: created.GetTemplate().GetId(),
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
