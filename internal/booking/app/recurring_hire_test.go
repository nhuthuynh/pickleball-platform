// T11.5 — app-layer tests for the RecurringHireTemplate request/approve/
// reject use cases built on T11.4's domain model.
//
// Two of these tests exist specifically because asserting a returned error is
// not enough evidence:
//   - the authorization tests assert an untouched repository call counter, the
//     same way T11.2's TestCreateDiscountRule_NonOwnerIsRejectedBeforeTheRepository
//     does — a check that ran *after* the write would still return the right
//     error;
//   - TestApproveRecurringHire_PartialConflictStillApproves asserts every
//     per-occurrence outcome individually, because an implementation that
//     aborted the whole approval on the first conflict would still return an
//     approved-looking template if only the status were checked.
package app_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nhuthuynh/white-label/internal/booking/app"
	"github.com/nhuthuynh/white-label/internal/booking/domain"
)

// --- fakes -------------------------------------------------------------

// fakeRecurringHireRepo is an in-memory port.RecurringHireRepository.
//
// createCalls/updateCalls exist so a test can prove a rejected request never
// reached the repository at all — the point of the ordering AC, which a
// returned error alone cannot demonstrate.
type fakeRecurringHireRepo struct {
	templates map[string]domain.RecurringHireTemplate

	createCalls int
	updateCalls int
	listCalls   int
}

func newFakeRecurringHireRepo() *fakeRecurringHireRepo {
	return &fakeRecurringHireRepo{templates: make(map[string]domain.RecurringHireTemplate)}
}

func (r *fakeRecurringHireRepo) Create(_ context.Context, t domain.RecurringHireTemplate) (domain.RecurringHireTemplate, error) {
	r.createCalls++
	r.templates[t.ID] = t
	return t, nil
}

func (r *fakeRecurringHireRepo) GetByID(_ context.Context, id string) (domain.RecurringHireTemplate, error) {
	t, ok := r.templates[id]
	if !ok {
		return domain.RecurringHireTemplate{}, domain.ErrRecurringHireTemplateNotFound
	}
	return t, nil
}

func (r *fakeRecurringHireRepo) UpdateStatus(_ context.Context, t domain.RecurringHireTemplate) (domain.RecurringHireTemplate, error) {
	r.updateCalls++
	if _, ok := r.templates[t.ID]; !ok {
		return domain.RecurringHireTemplate{}, domain.ErrRecurringHireTemplateNotFound
	}
	r.templates[t.ID] = t
	return t, nil
}

func (r *fakeRecurringHireRepo) ListForCourts(_ context.Context, courtIDs []string) ([]domain.RecurringHireTemplate, error) {
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

func (r *fakeRecurringHireRepo) ListForRequester(_ context.Context, requestedByUserID string) ([]domain.RecurringHireTemplate, error) {
	r.listCalls++
	out := make([]domain.RecurringHireTemplate, 0, len(r.templates))
	for _, t := range r.templates {
		if t.RequestedByUserID == requestedByUserID {
			out = append(out, t)
		}
	}
	return out, nil
}

// fakeIdentityLookup is an in-memory port.IdentityLookup: the set of users
// that exist, and which of them hold the `club` role. That is exactly the
// answer the real adapter derives from identityapp.Service.GetUser's Roles —
// the point being that the role is a server-side fact, never a field the
// caller sends.
type fakeIdentityLookup struct {
	clubs  map[string]bool
	known  map[string]bool
	checks int
}

func (l *fakeIdentityLookup) EnsureClubRole(_ context.Context, actorUserID string) error {
	l.checks++
	if !l.known[actorUserID] {
		return domain.ErrUserNotFound
	}
	if !l.clubs[actorUserID] {
		return domain.ErrNotClub
	}
	return nil
}

// --- fixtures ----------------------------------------------------------

const (
	clubUser    = 10 // holds the `club` role
	playerUser  = 11 // a real user who does NOT hold `club`
	ownerUser   = 12 // owns facilityID(1)
	strangerUsr = 13 // a real user who owns nothing
)

// mondayNineToTen is the schedule every fixture template uses: Mondays,
// 09:00-10:00, starting Monday 2026-01-05.
func mondayNineToTen(t *testing.T) (domain.ClockTime, domain.ClockTime, time.Time) {
	t.Helper()
	start, err := domain.NewClockTime(9, 0)
	if err != nil {
		t.Fatalf("fixture clock time: %v", err)
	}
	end, err := domain.NewClockTime(10, 0)
	if err != nil {
		t.Fatalf("fixture clock time: %v", err)
	}
	return start, end, time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)
}

func requestInput(t *testing.T, actor, court string, occurrences int) app.RequestRecurringHireInput {
	t.Helper()
	start, end, startsAt := mondayNineToTen(t)
	endCondition, err := domain.EndRecurringHireAfterOccurrences(occurrences)
	if err != nil {
		t.Fatalf("fixture end condition: %v", err)
	}
	return app.RequestRecurringHireInput{
		ActorUserID:  actor,
		CourtID:      court,
		Weekday:      time.Monday,
		StartTime:    start,
		EndTime:      end,
		StartsAt:     startsAt,
		EndCondition: endCondition,
	}
}

type recurringFixture struct {
	svc        *app.Service
	templates  *fakeRecurringHireRepo
	bookings   *inMemoryRepo
	identity   *fakeIdentityLookup
	facilities *fakeFacilityLookup
}

// newRecurringSvc wires a Service whose court courtID(1) belongs to
// facilityID(1), owned by userID(ownerUser); userID(clubUser) holds the club
// role and userID(playerUser)/userID(strangerUsr) do not.
func newRecurringSvc() *recurringFixture {
	templates := newFakeRecurringHireRepo()
	bookings := newInMemoryRepo()
	identity := &fakeIdentityLookup{
		clubs: map[string]bool{userID(clubUser): true},
		known: map[string]bool{
			userID(clubUser):    true,
			userID(playerUser):  true,
			userID(ownerUser):   true,
			userID(strangerUsr): true,
		},
	}
	facilities := &fakeFacilityLookup{
		ownerByFacility: map[string]string{facilityID(1): userID(ownerUser)},
		facilityByCourt: map[string]string{courtID(1): facilityID(1)},
	}
	svc := app.NewService(bookings, &fakePricingRepo{}, newFakeDiscountRepo(), templates,
		facilities, identity, &sequentialIDs{})
	return &recurringFixture{
		svc: svc, templates: templates, bookings: bookings,
		identity: identity, facilities: facilities,
	}
}

// mustRequest submits a valid request as the club user and returns the
// resulting template.
func mustRequest(t *testing.T, f *recurringFixture, occurrences int) domain.RecurringHireTemplate {
	t.Helper()
	tpl, err := f.svc.RequestRecurringHire(context.Background(), requestInput(t, userID(clubUser), courtID(1), occurrences))
	if err != nil {
		t.Fatalf("RequestRecurringHire: %v", err)
	}
	return tpl
}

// --- RequestRecurringHire ----------------------------------------------

// TestRequestRecurringHire_ClubActorCanRequest is the happy path, and also
// the creation-RPC checklist's item 1 in test form (T10 retro finding 3,
// sprint plan A4): the ID is minted server-side by the IDGenerator port, never
// taken from the caller — RequestRecurringHireInput has no ID field at all, so
// there is no identifier for an anonymous caller to squat.
func TestRequestRecurringHire_ClubActorCanRequest(t *testing.T) {
	t.Parallel()

	f := newRecurringSvc()

	tpl, err := f.svc.RequestRecurringHire(context.Background(), requestInput(t, userID(clubUser), courtID(1), 4))
	if err != nil {
		t.Fatalf("RequestRecurringHire: %v", err)
	}
	if tpl.ID != seqID(1) {
		t.Errorf("ID = %q, want the server-generated %q", tpl.ID, seqID(1))
	}
	if tpl.Status != domain.RecurringHireStatusRequested {
		t.Errorf("Status = %q, want %q", tpl.Status, domain.RecurringHireStatusRequested)
	}
	if tpl.RequestedByUserID != userID(clubUser) {
		t.Errorf("RequestedByUserID = %q, want %q", tpl.RequestedByUserID, userID(clubUser))
	}
	if f.templates.createCalls != 1 {
		t.Errorf("repository Create called %d times, want 1", f.templates.createCalls)
	}
}

// TestRequestRecurringHire_NonClubActorIsRejectedBeforeTheRepository is the
// creation-RPC checklist's item 2 — the one that actually fires for this
// ticket (sprint plan A4). Whether the actor may request a recurring hire as a
// Club is resolved server-side from Identity's real Roles; there is no
// is_club/role field on the input for a caller to self-declare. The untouched
// call counter is the part that matters: a role check that ran after the
// insert would still return the right error.
func TestRequestRecurringHire_NonClubActorIsRejectedBeforeTheRepository(t *testing.T) {
	t.Parallel()

	f := newRecurringSvc()

	_, err := f.svc.RequestRecurringHire(context.Background(), requestInput(t, userID(playerUser), courtID(1), 4))
	if !errors.Is(err, domain.ErrNotClub) {
		t.Fatalf("error = %v, want %v", err, domain.ErrNotClub)
	}
	if f.templates.createCalls != 0 {
		t.Fatalf("a non-club actor's template reached the repository (%d calls); the role check must run first", f.templates.createCalls)
	}
}

// TestRequestRecurringHire_UnresolvableActorIsRejected covers the actor ids
// that cannot be resolved to a real User at all. They are answered the same
// way a known non-club actor is (PermissionDenied at the gRPC boundary, see
// the handler test): an actor who cannot be resolved has not proven the club
// role, and answering NotFound here would turn an unauthenticated endpoint
// into a user-enumeration oracle.
//
// The malformed values additionally prove the app-layer uuidShape guard fires
// before the lookup — identity_users.id is a `uuid` column.
func TestRequestRecurringHire_UnresolvableActorIsRejected(t *testing.T) {
	t.Parallel()

	t.Run("unknown but well-formed actor", func(t *testing.T) {
		t.Parallel()
		f := newRecurringSvc()

		_, err := f.svc.RequestRecurringHire(context.Background(), requestInput(t, userID(99), courtID(1), 4))
		if !errors.Is(err, domain.ErrUserNotFound) {
			t.Fatalf("error = %v, want %v", err, domain.ErrUserNotFound)
		}
		if f.identity.checks != 1 {
			t.Errorf("identity lookup called %d times, want 1 — a well-formed id must still be resolved", f.identity.checks)
		}
		if f.templates.createCalls != 0 {
			t.Errorf("an unknown actor's template reached the repository (%d calls)", f.templates.createCalls)
		}
	})

	for _, id := range malformedFacilityIDs() {
		id := id
		t.Run("malformed/"+id, func(t *testing.T) {
			t.Parallel()
			f := newRecurringSvc()

			_, err := f.svc.RequestRecurringHire(context.Background(), requestInput(t, id, courtID(1), 4))
			if !errors.Is(err, domain.ErrUserNotFound) {
				t.Fatalf("RequestRecurringHire(actor=%q) error = %v, want %v", id, err, domain.ErrUserNotFound)
			}
			if f.identity.checks != 0 {
				t.Errorf("a malformed actor id reached the Identity lookup (%d calls) — against real Postgres that hits a `uuid` column", f.identity.checks)
			}
			if f.templates.createCalls != 0 {
				t.Errorf("a malformed actor id reached the repository (%d calls)", f.templates.createCalls)
			}
		})
	}
}

// TestRequestRecurringHire_UnresolvableCourtIsNotFound covers the ticket's
// NotFound mapping for a Court that names no Facility — including a Court that
// exists but belongs to none (courts.facility_id is nullable). Both are
// rejected rather than accepted, because a template on such a Court could
// never be approved: there is no Facility Owner to approve it.
func TestRequestRecurringHire_UnresolvableCourtIsNotFound(t *testing.T) {
	t.Parallel()

	ids := append([]string{courtID(77)}, malformedFacilityIDs()...)
	for _, id := range ids {
		id := id
		t.Run(id, func(t *testing.T) {
			t.Parallel()
			f := newRecurringSvc()

			_, err := f.svc.RequestRecurringHire(context.Background(), requestInput(t, userID(clubUser), id, 4))
			if !errors.Is(err, domain.ErrFacilityNotFound) {
				t.Fatalf("RequestRecurringHire(court=%q) error = %v, want %v", id, err, domain.ErrFacilityNotFound)
			}
			if f.templates.createCalls != 0 {
				t.Errorf("an unresolvable court's template reached the repository (%d calls)", f.templates.createCalls)
			}
		})
	}
}

// TestRequestRecurringHire_InvalidScheduleIsRejected pins the ticket's
// InvalidArgument cases. An end-after-zero-occurrences condition is rejected
// by T11.4's own EndRecurringHireAfterOccurrences constructor before it can
// reach the app layer, so it is exercised here directly rather than through a
// half-built input.
func TestRequestRecurringHire_InvalidScheduleIsRejected(t *testing.T) {
	t.Parallel()

	t.Run("start time not before end time", func(t *testing.T) {
		t.Parallel()
		f := newRecurringSvc()

		in := requestInput(t, userID(clubUser), courtID(1), 4)
		in.StartTime, in.EndTime = in.EndTime, in.StartTime

		_, err := f.svc.RequestRecurringHire(context.Background(), in)
		if !errors.Is(err, domain.ErrInvalidRecurringHireTimeRange) {
			t.Fatalf("error = %v, want %v", err, domain.ErrInvalidRecurringHireTimeRange)
		}
		if f.templates.createCalls != 0 {
			t.Errorf("an invalid template reached the repository (%d calls)", f.templates.createCalls)
		}
	})

	t.Run("non-positive end-after-occurrences", func(t *testing.T) {
		t.Parallel()

		for _, n := range []int{0, -1} {
			if _, err := domain.EndRecurringHireAfterOccurrences(n); !errors.Is(err, domain.ErrInvalidRecurringHireEndAfterOccurrences) {
				t.Fatalf("EndRecurringHireAfterOccurrences(%d) error = %v, want %v", n, err, domain.ErrInvalidRecurringHireEndAfterOccurrences)
			}
		}
	})
}

// --- ApproveRecurringHire ----------------------------------------------

// TestApproveRecurringHire_PartialConflictStillApproves is this ticket's
// headline AC: a single already-booked Tuesday must not block an entire
// multi-week request. One of the four occurrences is pre-booked by an
// unrelated individual booking, so approval hits ErrCourtDoubleBooked on that
// week and only that week.
//
// Every outcome is asserted individually, not just the template's status: an
// implementation that aborted on the first conflict, or one that silently
// dropped the conflicted week from the response, would both still leave the
// template approved.
func TestApproveRecurringHire_PartialConflictStillApproves(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	f := newRecurringSvc()
	tpl := mustRequest(t, f, 4)

	// Occupy the third occurrence (Monday 2026-01-19, 09:00-10:00) with an
	// unrelated individual booking, exactly as a Player booking that slot
	// through the normal court-hire flow would.
	conflicted := domain.TimeRange{
		Start: time.Date(2026, 1, 19, 9, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 1, 19, 10, 0, 0, 0, time.UTC),
	}
	if _, err := f.svc.CreateBooking(ctx, app.CreateBookingInput{
		CourtID: courtID(1), Source: domain.SourceIndividual, Range: conflicted,
	}); err != nil {
		t.Fatalf("fixture CreateBooking: %v", err)
	}

	result, err := f.svc.ApproveRecurringHire(ctx, tpl.ID, userID(ownerUser))
	if err != nil {
		t.Fatalf("ApproveRecurringHire returned %v; a per-occurrence conflict must not fail the whole approval", err)
	}

	if result.Template.Status != domain.RecurringHireStatusApproved {
		t.Errorf("template status = %q, want %q", result.Template.Status, domain.RecurringHireStatusApproved)
	}
	// The approval must be persisted, not just returned.
	stored, err := f.templates.GetByID(ctx, tpl.ID)
	if err != nil {
		t.Fatalf("reading back the template: %v", err)
	}
	if stored.Status != domain.RecurringHireStatusApproved {
		t.Errorf("persisted status = %q, want %q", stored.Status, domain.RecurringHireStatusApproved)
	}

	if len(result.Occurrences) != 4 {
		t.Fatalf("got %d occurrence results, want 4 — every occurrence must be reported, booked or not", len(result.Occurrences))
	}

	wantOutcomes := []app.OccurrenceOutcome{
		app.OccurrenceBooked,
		app.OccurrenceBooked,
		app.OccurrenceSkippedConflict,
		app.OccurrenceBooked,
	}
	for i, want := range wantOutcomes {
		got := result.Occurrences[i]
		if got.Outcome != want {
			t.Errorf("occurrence %d (%s) outcome = %q, want %q", i, got.Range.Start.Format(time.RFC3339), got.Outcome, want)
		}
		switch want {
		case app.OccurrenceBooked:
			if got.BookingID == "" {
				t.Errorf("occurrence %d was booked but carries no booking id", i)
			}
			if got.Reason != "" {
				t.Errorf("occurrence %d was booked but carries reason %q", i, got.Reason)
			}
		case app.OccurrenceSkippedConflict:
			if got.BookingID != "" {
				t.Errorf("occurrence %d was skipped but carries booking id %q", i, got.BookingID)
			}
			if got.Reason == "" {
				t.Error("a skipped occurrence must say why it was skipped")
			}
		}
	}

	// The skipped week is the one that was already taken, not an arbitrary one.
	if got := result.Occurrences[2].Range.Start; !got.Equal(conflicted.Start) {
		t.Errorf("skipped occurrence starts at %s, want the already-booked %s", got, conflicted.Start)
	}

	// And the three that succeeded are real, recurring-hire-sourced Bookings
	// on the court — not just entries in a response.
	all, err := f.svc.ListCourtBookings(ctx, courtID(1), domain.TimeRange{
		Start: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("ListCourtBookings: %v", err)
	}
	recurring := 0
	for _, b := range all {
		if b.Source != domain.SourceRecurringHire {
			continue
		}
		recurring++
		if b.ReferenceID != tpl.ID {
			t.Errorf("recurring booking %q references %q, want the template %q", b.ID, b.ReferenceID, tpl.ID)
		}
	}
	if recurring != 3 {
		t.Errorf("got %d recurring-hire bookings on the court, want 3", recurring)
	}
}

// TestApproveRecurringHire_AllOccurrencesBooked is the positive control the
// partial-success test needs: without it, an implementation that skipped
// *everything* would still satisfy "approval succeeds despite conflicts".
func TestApproveRecurringHire_AllOccurrencesBooked(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	f := newRecurringSvc()
	tpl := mustRequest(t, f, 3)

	result, err := f.svc.ApproveRecurringHire(ctx, tpl.ID, userID(ownerUser))
	if err != nil {
		t.Fatalf("ApproveRecurringHire: %v", err)
	}
	if len(result.Occurrences) != 3 {
		t.Fatalf("got %d occurrences, want 3", len(result.Occurrences))
	}
	for i, occ := range result.Occurrences {
		if occ.Outcome != app.OccurrenceBooked {
			t.Errorf("occurrence %d outcome = %q, want %q on an unconflicted court", i, occ.Outcome, app.OccurrenceBooked)
		}
	}
}

// TestApproveRecurringHire_NonOwnerIsRejectedBeforeAnythingChanges is the
// object-level (BOLA) AC, mirroring T11.2's
// TestCreateDiscountRule_NonOwnerIsRejectedBeforeTheRepository: ownership is
// checked before the template's status is touched and before any Booking is
// created. Asserting only the returned error would pass even if the approval
// had already been written.
func TestApproveRecurringHire_NonOwnerIsRejectedBeforeAnythingChanges(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	for _, actor := range []string{userID(strangerUsr), userID(clubUser)} {
		actor := actor
		t.Run(actor, func(t *testing.T) {
			t.Parallel()
			f := newRecurringSvc()
			tpl := mustRequest(t, f, 4)

			_, err := f.svc.ApproveRecurringHire(ctx, tpl.ID, actor)
			if !errors.Is(err, domain.ErrNotFacilityOwner) {
				t.Fatalf("error = %v, want %v", err, domain.ErrNotFacilityOwner)
			}
			if f.templates.updateCalls != 0 {
				t.Errorf("a non-owner's approval was persisted (%d update calls)", f.templates.updateCalls)
			}
			stored, err := f.templates.GetByID(ctx, tpl.ID)
			if err != nil {
				t.Fatalf("reading back the template: %v", err)
			}
			if stored.Status != domain.RecurringHireStatusRequested {
				t.Errorf("template status = %q after a non-owner approval, want it untouched at %q", stored.Status, domain.RecurringHireStatusRequested)
			}
			if got := len(f.bookings.bookings); got != 0 {
				t.Errorf("a non-owner's approval created %d bookings, want 0", got)
			}
		})
	}
}

// TestApproveRecurringHire_UnknownOrMalformedTemplateIsNotFound covers the
// ticket's NotFound mapping for the template id, including the malformed
// shapes the app-layer guard must catch before they reach a `uuid` column.
func TestApproveRecurringHire_UnknownOrMalformedTemplateIsNotFound(t *testing.T) {
	t.Parallel()

	ids := append([]string{seqID(77)}, malformedFacilityIDs()...)
	for _, id := range ids {
		id := id
		t.Run(id, func(t *testing.T) {
			t.Parallel()
			f := newRecurringSvc()

			_, err := f.svc.ApproveRecurringHire(context.Background(), id, userID(ownerUser))
			if !errors.Is(err, domain.ErrRecurringHireTemplateNotFound) {
				t.Fatalf("ApproveRecurringHire(%q) error = %v, want %v", id, err, domain.ErrRecurringHireTemplateNotFound)
			}
		})
	}
}

// TestApproveRecurringHire_AlreadyDecidedIsRejected pins the
// FailedPrecondition case: T11.4's Approve/Reject are legal only from
// `requested`, and re-approving must not silently re-generate a second set of
// Bookings for the same weeks.
func TestApproveRecurringHire_AlreadyDecidedIsRejected(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	f := newRecurringSvc()
	tpl := mustRequest(t, f, 2)

	if _, err := f.svc.ApproveRecurringHire(ctx, tpl.ID, userID(ownerUser)); err != nil {
		t.Fatalf("first ApproveRecurringHire: %v", err)
	}
	bookingsAfterFirst := len(f.bookings.bookings)

	_, err := f.svc.ApproveRecurringHire(ctx, tpl.ID, userID(ownerUser))
	if !errors.Is(err, domain.ErrInvalidRecurringHireStatusTransition) {
		t.Fatalf("second approval error = %v, want %v", err, domain.ErrInvalidRecurringHireStatusTransition)
	}
	if got := len(f.bookings.bookings); got != bookingsAfterFirst {
		t.Errorf("re-approving created %d extra bookings, want 0", got-bookingsAfterFirst)
	}

	if _, err := f.svc.RejectRecurringHire(ctx, tpl.ID, userID(ownerUser)); !errors.Is(err, domain.ErrInvalidRecurringHireStatusTransition) {
		t.Errorf("rejecting an approved template error = %v, want %v", err, domain.ErrInvalidRecurringHireStatusTransition)
	}
}

// --- RejectRecurringHire -----------------------------------------------

// TestRejectRecurringHire covers the reject path: the owner may reject, a
// non-owner may not (and the template is left untouched when they try), and a
// rejection never creates Bookings.
func TestRejectRecurringHire(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("owner rejects", func(t *testing.T) {
		t.Parallel()
		f := newRecurringSvc()
		tpl := mustRequest(t, f, 4)

		rejected, err := f.svc.RejectRecurringHire(ctx, tpl.ID, userID(ownerUser))
		if err != nil {
			t.Fatalf("RejectRecurringHire: %v", err)
		}
		if rejected.Status != domain.RecurringHireStatusRejected {
			t.Errorf("status = %q, want %q", rejected.Status, domain.RecurringHireStatusRejected)
		}
		if got := len(f.bookings.bookings); got != 0 {
			t.Errorf("rejection created %d bookings, want 0", got)
		}
	})

	t.Run("non-owner is rejected before anything changes", func(t *testing.T) {
		t.Parallel()
		f := newRecurringSvc()
		tpl := mustRequest(t, f, 4)

		_, err := f.svc.RejectRecurringHire(ctx, tpl.ID, userID(strangerUsr))
		if !errors.Is(err, domain.ErrNotFacilityOwner) {
			t.Fatalf("error = %v, want %v", err, domain.ErrNotFacilityOwner)
		}
		if f.templates.updateCalls != 0 {
			t.Errorf("a non-owner's rejection was persisted (%d update calls)", f.templates.updateCalls)
		}
	})

	t.Run("unknown template is not found", func(t *testing.T) {
		t.Parallel()
		f := newRecurringSvc()

		_, err := f.svc.RejectRecurringHire(ctx, seqID(77), userID(ownerUser))
		if !errors.Is(err, domain.ErrRecurringHireTemplateNotFound) {
			t.Fatalf("error = %v, want %v", err, domain.ErrRecurringHireTemplateNotFound)
		}
	})
}

// --- ListRecurringHireTemplatesForFacility -----------------------------

// TestListRecurringHireTemplatesForFacility covers the owner-facing read.
// Unlike T11.2's ListDiscountRulesForFacility (deliberately unauthenticated,
// because a discounted price is something a Player is quoted anyway), this
// list is a queue of other parties' pending business requests, so it is
// owner-only — see the method's own doc comment.
func TestListRecurringHireTemplatesForFacility(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("owner sees the facility's templates", func(t *testing.T) {
		t.Parallel()
		f := newRecurringSvc()
		tpl := mustRequest(t, f, 4)

		got, err := f.svc.ListRecurringHireTemplatesForFacility(ctx, facilityID(1), userID(ownerUser))
		if err != nil {
			t.Fatalf("ListRecurringHireTemplatesForFacility: %v", err)
		}
		if len(got) != 1 || got[0].ID != tpl.ID {
			t.Fatalf("got %+v, want exactly the template %q", got, tpl.ID)
		}
	})

	t.Run("non-owner is rejected before the repository", func(t *testing.T) {
		t.Parallel()
		f := newRecurringSvc()
		mustRequest(t, f, 4)

		_, err := f.svc.ListRecurringHireTemplatesForFacility(ctx, facilityID(1), userID(strangerUsr))
		if !errors.Is(err, domain.ErrNotFacilityOwner) {
			t.Fatalf("error = %v, want %v", err, domain.ErrNotFacilityOwner)
		}
		if f.templates.listCalls != 0 {
			t.Errorf("a non-owner's list reached the repository (%d calls)", f.templates.listCalls)
		}
	})

	t.Run("unknown or malformed facility is not found", func(t *testing.T) {
		t.Parallel()

		for _, id := range append([]string{facilityID(77)}, malformedFacilityIDs()...) {
			f := newRecurringSvc()
			_, err := f.svc.ListRecurringHireTemplatesForFacility(ctx, id, userID(ownerUser))
			if !errors.Is(err, domain.ErrFacilityNotFound) {
				t.Fatalf("ListRecurringHireTemplatesForFacility(%q) error = %v, want %v", id, err, domain.ErrFacilityNotFound)
			}
		}
	})
}

// --- ListRecurringHireTemplatesForActor --------------------------------

// TestListRecurringHireTemplatesForActor covers the CLUB-facing read T11.6
// adds: "show me the templates I asked for, and what happened to them".
//
// T11.5 shipped only the owner-facing, Facility-scoped queue and flagged the
// absence of this one as a known gap (see
// ListRecurringHireTemplatesForFacility's own doc comment); T11.6's Club
// status view is the consumer that closes it.
//
// The authorization shape is deliberately different from the Facility read's,
// and each subtest below pins one half of why:
//   - there is no ownership check to make, because the actor's own identity IS
//     the entire scope — the query filters on requested_by_user_id, so a
//     caller can only ever be answered with rows they themselves created
//     ("actor sees only their own templates");
//   - there is deliberately no `club`-role check either, and the untouched
//     identity call counter is what proves it is absent rather than merely
//     passing for this fixture ("no role check is performed"). A role check
//     would gate a User's access to their OWN request history on a role they
//     could later lose, while removing no exposure: T11.5 already gates
//     *creation* on the club role, so a non-club actor's list is empty anyway.
func TestListRecurringHireTemplatesForActor(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("actor sees only their own templates", func(t *testing.T) {
		t.Parallel()
		f := newRecurringSvc()
		// A second club, on the same court, whose template must not leak into
		// the first club's list.
		const otherClubUser = 14
		f.identity.clubs[userID(otherClubUser)] = true
		f.identity.known[userID(otherClubUser)] = true

		mine := mustRequest(t, f, 4)
		theirs, err := f.svc.RequestRecurringHire(ctx, requestInput(t, userID(otherClubUser), courtID(1), 4))
		if err != nil {
			t.Fatalf("RequestRecurringHire (other club): %v", err)
		}

		got, err := f.svc.ListRecurringHireTemplatesForActor(ctx, userID(clubUser))
		if err != nil {
			t.Fatalf("ListRecurringHireTemplatesForActor: %v", err)
		}
		if len(got) != 1 || got[0].ID != mine.ID {
			t.Fatalf("got %+v, want exactly the actor's own template %q (the other club's %q must not appear)",
				got, mine.ID, theirs.ID)
		}
	})

	// The status view's whole point: a rejected request must come back as a
	// real, terminal state, not vanish from the list as though it were still
	// open or had never been made.
	t.Run("every status is readable, including the terminal ones", func(t *testing.T) {
		t.Parallel()
		f := newRecurringSvc()

		requested := mustRequest(t, f, 4)
		toApprove := mustRequest(t, f, 4)
		toReject := mustRequest(t, f, 4)

		if _, err := f.svc.ApproveRecurringHire(ctx, toApprove.ID, userID(ownerUser)); err != nil {
			t.Fatalf("ApproveRecurringHire: %v", err)
		}
		if _, err := f.svc.RejectRecurringHire(ctx, toReject.ID, userID(ownerUser)); err != nil {
			t.Fatalf("RejectRecurringHire: %v", err)
		}

		got, err := f.svc.ListRecurringHireTemplatesForActor(ctx, userID(clubUser))
		if err != nil {
			t.Fatalf("ListRecurringHireTemplatesForActor: %v", err)
		}

		statuses := make(map[string]domain.RecurringHireStatus, len(got))
		for _, tpl := range got {
			statuses[tpl.ID] = tpl.Status
		}
		want := map[string]domain.RecurringHireStatus{
			requested.ID: domain.RecurringHireStatusRequested,
			toApprove.ID: domain.RecurringHireStatusApproved,
			toReject.ID:  domain.RecurringHireStatusRejected,
		}
		for id, wantStatus := range want {
			if statuses[id] != wantStatus {
				t.Errorf("template %q status = %q, want %q", id, statuses[id], wantStatus)
			}
		}
	})

	// The negative finding, asserted rather than assumed: this read performs
	// no role check at all. An identity call counter of zero is the evidence —
	// a check that happened to pass for a club fixture would look identical
	// from the returned value alone.
	t.Run("no role check is performed", func(t *testing.T) {
		t.Parallel()
		f := newRecurringSvc()
		f.identity.checks = 0

		got, err := f.svc.ListRecurringHireTemplatesForActor(ctx, userID(playerUser))
		if err != nil {
			t.Fatalf("ListRecurringHireTemplatesForActor: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("got %+v, want an empty list — a non-club actor can hold no templates", got)
		}
		if f.identity.checks != 0 {
			t.Errorf("identity was consulted %d times; this read is scoped by identity, not gated by role", f.identity.checks)
		}
	})

	// A malformed actor id is answered exactly like an unknown one — which,
	// for this read, means an empty list rather than an error (the T10.7
	// guard's rule applied faithfully, not a new error invented for it).
	// Contrast ListRecurringHireTemplatesForFacility, where a malformed id IS
	// an error: there, an actor check must not silently pass for a Facility
	// that does not exist. Here there is no such check to skip.
	//
	// The guard still has to exist: the Postgres adapter's mustUUID panics on
	// anything pgtype.UUID.Scan cannot parse, so an unguarded malformed id off
	// the wire would take the process down.
	t.Run("unknown or malformed actor gets an empty list, never a panic", func(t *testing.T) {
		t.Parallel()

		for _, id := range append([]string{userID(99)}, malformedFacilityIDs()...) {
			f := newRecurringSvc()
			mustRequest(t, f, 4)

			got, err := f.svc.ListRecurringHireTemplatesForActor(ctx, id)
			if err != nil {
				t.Fatalf("ListRecurringHireTemplatesForActor(%q) error = %v, want nil", id, err)
			}
			if len(got) != 0 {
				t.Errorf("ListRecurringHireTemplatesForActor(%q) = %+v, want an empty list", id, got)
			}
		}
	})

	// A malformed id must not even reach the repository — that is what keeps
	// mustUUID from ever seeing it.
	t.Run("a malformed actor never reaches the repository", func(t *testing.T) {
		t.Parallel()
		f := newRecurringSvc()

		if _, err := f.svc.ListRecurringHireTemplatesForActor(ctx, "not-a-uuid"); err != nil {
			t.Fatalf("ListRecurringHireTemplatesForActor: %v", err)
		}
		if f.templates.listCalls != 0 {
			t.Errorf("a malformed actor id reached the repository (%d calls)", f.templates.listCalls)
		}
	})
}
