// End-to-end coverage for the actor path issue #146 exposed: a verified IdP
// subject leaving the grpcapi boundary, crossing Booking's app layer, and
// reaching the REAL Identity context — no faked port.IdentityLookup anywhere.
//
// This file lives under adapter/ rather than beside the app-layer tests
// because wiring the real Identity service means importing an adapter, which
// internal/booking/app's own tests must never do (CLAUDE.md rule 3: the
// dependency rule points inward). Here it is legal: this IS the adapter.
package identity_test

import (
	"context"
	"errors"
	"testing"
	"time"

	bookingidentity "github.com/nhuthuynh/white-label/internal/booking/adapter/identity"
	bookingapp "github.com/nhuthuynh/white-label/internal/booking/app"
	bookingdomain "github.com/nhuthuynh/white-label/internal/booking/domain"
	identityapp "github.com/nhuthuynh/white-label/internal/identity/app"
	identitydomain "github.com/nhuthuynh/white-label/internal/identity/domain"
)

const (
	fixtureCourtID    = "3f2504e0-4f89-11d3-9a0c-0305e82c3301"
	fixtureFacilityID = "3f2504e0-4f89-11d3-9a0c-0305e82c3302"
	fixtureTemplateID = "3f2504e0-4f89-11d3-9a0c-0305e82c3303"
)

// --- minimal fakes for the collaborators this path actually touches ------

type fakeFacilityLookup struct{}

func (fakeFacilityLookup) EnsureFacilityOwner(_ context.Context, _, _ string) error { return nil }
func (fakeFacilityLookup) FacilityIDForCourt(_ context.Context, _ string) (string, error) {
	return fixtureFacilityID, nil
}
func (fakeFacilityLookup) CourtIDsForFacility(_ context.Context, _ string) ([]string, error) {
	return []string{fixtureCourtID}, nil
}

type fakeRecurringRepo struct {
	created     []bookingdomain.RecurringHireTemplate
	createCalls int
}

func (r *fakeRecurringRepo) Create(_ context.Context, t bookingdomain.RecurringHireTemplate) (bookingdomain.RecurringHireTemplate, error) {
	r.createCalls++
	r.created = append(r.created, t)
	return t, nil
}

func (r *fakeRecurringRepo) GetByID(_ context.Context, _ string) (bookingdomain.RecurringHireTemplate, error) {
	return bookingdomain.RecurringHireTemplate{}, bookingdomain.ErrRecurringHireTemplateNotFound
}

func (r *fakeRecurringRepo) UpdateStatus(_ context.Context, t bookingdomain.RecurringHireTemplate) (bookingdomain.RecurringHireTemplate, error) {
	return t, nil
}

func (r *fakeRecurringRepo) ListForCourts(_ context.Context, _ []string) ([]bookingdomain.RecurringHireTemplate, error) {
	return nil, nil
}

func (r *fakeRecurringRepo) ListForRequester(_ context.Context, _ string) ([]bookingdomain.RecurringHireTemplate, error) {
	return nil, nil
}

type stubBookingIDs struct{}

func (stubBookingIDs) NewID() string { return fixtureTemplateID }

// newServiceWithRealIdentity wires Booking's app.Service with the REAL
// Identity chain behind port.IdentityLookup. The three repositories
// RequestRecurringHire never reaches on this path are left nil deliberately:
// if a future change makes this use case touch them, the resulting nil panic
// is a louder and more honest signal than a fake quietly absorbing the call.
func newServiceWithRealIdentity(t *testing.T, seed identitydomain.User) (*bookingapp.Service, *fakeRecurringRepo) {
	t.Helper()

	repo := newInMemoryRepo()
	if _, err := repo.Create(context.Background(), seed); err != nil {
		t.Fatalf("seeding identity repo: %v", err)
	}
	lookup := bookingidentity.NewLookup(identityapp.NewService(repo, stubIDs{}))

	recurringRepo := &fakeRecurringRepo{}
	svc := bookingapp.NewService(
		nil, // port.Repository        — unused by RequestRecurringHire
		nil, // PricingRuleRepository  — unused
		nil, // DiscountRuleRepository — unused
		recurringRepo,
		fakeFacilityLookup{},
		lookup,
		stubBookingIDs{},
	)
	return svc, recurringRepo
}

func subjectActorInput(t *testing.T, actor string) bookingapp.RequestRecurringHireInput {
	t.Helper()

	start, err := bookingdomain.NewClockTime(9, 0)
	if err != nil {
		t.Fatalf("fixture clock time: %v", err)
	}
	end, err := bookingdomain.NewClockTime(10, 0)
	if err != nil {
		t.Fatalf("fixture clock time: %v", err)
	}
	endCondition, err := bookingdomain.EndRecurringHireAfterOccurrences(4)
	if err != nil {
		t.Fatalf("fixture end condition: %v", err)
	}
	return bookingapp.RequestRecurringHireInput{
		ActorUserID:  actor,
		CourtID:      fixtureCourtID,
		Weekday:      time.Monday,
		StartTime:    start,
		EndTime:      end,
		StartsAt:     time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC),
		EndCondition: endCondition,
	}
}

// TestRequestRecurringHire_SubjectActorStillBlockedByAppLayerUUIDGuard pins
// the SECOND half of issue #146, which the issue's own diagnosis does not
// mention and which this adapter fix does not close.
//
// app.Service.RequestRecurringHire opens with
// `uuidShape.MatchString(in.ActorUserID)` (internal/booking/app/
// recurring_hire.go) and returns ErrUserNotFound when it fails. Since T12.7
// the handler supplies auth.RequireSubject(ctx) — a subject, never a uuid —
// so that guard rejects every real caller BEFORE port.IdentityLookup is
// consulted at all. Fixing the adapter to resolve subjects (the change this
// file ships alongside) is necessary but not sufficient: the RPC is still
// broken end-to-end, it just no longer fails for the reason issue #146 named.
//
// This test asserts the defect rather than the desired behaviour on purpose.
// The remaining fix is not a one-liner and is not this change's to make: the
// actor value is persisted as recurring_hire_templates.requested_by_user_id,
// declared `uuid NOT NULL REFERENCES identity_users (id)`, and written via
// the postgres adapter's mustUUID(), which PANICS on a non-uuid. So simply
// deleting the guard would convert a clean NotFound into a server panic and
// an FK violation. Closing this properly means translating subject -> User.ID
// at the boundary and carrying the uuid inward, which changes
// port.IdentityLookup's contract (it deliberately exposes no method that
// returns a User) and touches all six of the handler's actor call sites.
// That is a design decision, escalated in the PR rather than guessed at here.
//
// When that follow-up lands, this test should FLIP: the wantErr expectation
// becomes a successful template creation, asserting createCalls == 1 and the
// persisted RequestedByUserID being the User's uuid rather than the subject.
func TestRequestRecurringHire_SubjectActorStillBlockedByAppLayerUUIDGuard(t *testing.T) {
	t.Parallel()

	seed := mustUser(t, fixtureUserID, fixtureSubject, []identitydomain.Role{identitydomain.RoleClub})
	svc, recurringRepo := newServiceWithRealIdentity(t, seed)

	_, err := svc.RequestRecurringHire(context.Background(), subjectActorInput(t, fixtureSubject))

	if !errors.Is(err, bookingdomain.ErrUserNotFound) {
		t.Fatalf("RequestRecurringHire(subject actor) = %v, want %v (app-layer uuid guard); "+
			"if this now succeeds, the subject->uuid follow-up has landed and this test must be flipped",
			err, bookingdomain.ErrUserNotFound)
	}
	if recurringRepo.createCalls != 0 {
		t.Fatalf("createCalls = %d, want 0 — a rejected request must never reach the repository", recurringRepo.createCalls)
	}
}

// TestRequestRecurringHire_UUIDActorReachesRealIdentityAndIsRejected proves
// the adapter fix is genuinely wired into the app service — that the real
// Identity chain, not a fake, is answering.
//
// A uuid-shaped actor clears the app-layer guard and reaches the real
// EnsureClubRole, which (correctly, post-fix) refuses to resolve a uuid as a
// subject. The distinction from the test above matters: that one fails
// BEFORE the lookup, this one fails INSIDE it, so together they show the
// guard and the lookup are two separate blockers rather than one.
func TestRequestRecurringHire_UUIDActorReachesRealIdentityAndIsRejected(t *testing.T) {
	t.Parallel()

	seed := mustUser(t, fixtureUserID, fixtureSubject, []identitydomain.Role{identitydomain.RoleClub})
	svc, recurringRepo := newServiceWithRealIdentity(t, seed)

	_, err := svc.RequestRecurringHire(context.Background(), subjectActorInput(t, fixtureUserID))

	if !errors.Is(err, bookingdomain.ErrUserNotFound) {
		t.Fatalf("RequestRecurringHire(uuid actor) = %v, want %v", err, bookingdomain.ErrUserNotFound)
	}
	if recurringRepo.createCalls != 0 {
		t.Fatalf("createCalls = %d, want 0", recurringRepo.createCalls)
	}
}
