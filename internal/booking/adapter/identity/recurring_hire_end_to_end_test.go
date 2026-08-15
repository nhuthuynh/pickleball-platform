// End-to-end coverage for the actor path issues #146 and #152 exposed: a
// verified IdP subject crossing into the REAL Identity context and being
// translated to a User.ID, then that User.ID crossing Booking's app layer —
// no faked port.IdentityLookup anywhere.
//
// This file lives under adapter/ rather than beside the app-layer tests
// because wiring the real Identity service means importing an adapter, which
// internal/booking/app's own tests must never do (CLAUDE.md rule 3: the
// dependency rule points inward). Here it is legal: this IS the adapter.
//
// **Scope, after T13.2.** ADR-0014 put the translation in the grpcapi
// handler's actor() funnel, so the full production path now starts one layer
// above this package. What these tests drive is everything from
// app.Service.ResolveActorUserID down, over the real Identity chain — which
// is the half that lives here. The handler-inclusive path (real
// auth.Principal -> real Handler -> this seam) is covered by
// internal/booking/adapter/grpcapi/subject_actor_seam_test.go. Neither file
// mocks port.IdentityLookup; between them the seam has no unexercised layer.
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
// RequestRecurringHire never reaches on this path are supplied as unused*
// stubs whose every method panics: if a future change makes this use case
// touch them, that panic is a louder and more honest signal than a fake
// quietly absorbing the call.
//
// **These were literal nils until T13.8**, for exactly that reason — a nil
// deref was the loud signal. ServiceOptions now rejects a nil required
// dependency at construction, so the nils could no longer stand; the unused*
// types below preserve the intent and sharpen it, since they name the port
// that was unexpectedly reached instead of reporting a bare nil-pointer
// dereference. This mirrors the unusedRecurringHireRepository /
// unusedFacilityLookup / unusedIdentityLookup stubs
// internal/booking/adapter/postgres/concurrency_integration_test.go has used
// for the same purpose since T4.
func newServiceWithRealIdentity(t *testing.T, seed identitydomain.User) (*bookingapp.Service, *fakeRecurringRepo) {
	t.Helper()

	repo := newInMemoryRepo()
	if _, err := repo.Create(context.Background(), seed); err != nil {
		t.Fatalf("seeding identity repo: %v", err)
	}
	lookup := bookingidentity.NewLookup(identityapp.NewService(repo, stubIDs{}))

	recurringRepo := &fakeRecurringRepo{}
	svc := bookingapp.NewService(bookingapp.ServiceOptions{
		Bookings:       unusedBookingRepository{},
		PricingRules:   unusedPricingRuleRepository{},
		DiscountRules:  unusedDiscountRuleRepository{},
		RecurringHires: recurringRepo,
		Facilities:     fakeFacilityLookup{},
		Identity:       lookup,
		IDs:            stubBookingIDs{},
	})
	return svc, recurringRepo
}

// --- ports RequestRecurringHire must never reach on this path -------------
//
// Every method panics rather than returning a zero value: a silent zero would
// let a future regression change what this file proves while every assertion
// below still passed.

type unusedBookingRepository struct{}

func (unusedBookingRepository) Create(_ context.Context, _ bookingdomain.Booking) (bookingdomain.Booking, error) {
	panic("port.Repository.Create is not part of the RequestRecurringHire path")
}

func (unusedBookingRepository) ListActiveForCourt(_ context.Context, _ string, _ bookingdomain.TimeRange) ([]bookingdomain.Booking, error) {
	panic("port.Repository.ListActiveForCourt is not part of the RequestRecurringHire path")
}

func (unusedBookingRepository) GetByID(_ context.Context, _ string) (bookingdomain.Booking, error) {
	panic("port.Repository.GetByID is not part of the RequestRecurringHire path")
}

func (unusedBookingRepository) Update(_ context.Context, _ bookingdomain.Booking) (bookingdomain.Booking, error) {
	panic("port.Repository.Update is not part of the RequestRecurringHire path")
}

type unusedPricingRuleRepository struct{}

func (unusedPricingRuleRepository) ListForCourt(_ context.Context, _ string) ([]bookingdomain.PricingRule, error) {
	panic("port.PricingRuleRepository.ListForCourt is not part of the RequestRecurringHire path")
}

type unusedDiscountRuleRepository struct{}

func (unusedDiscountRuleRepository) Create(_ context.Context, _ bookingdomain.DiscountRule) (bookingdomain.DiscountRule, error) {
	panic("port.DiscountRuleRepository.Create is not part of the RequestRecurringHire path")
}

func (unusedDiscountRuleRepository) ListForFacility(_ context.Context, _ string) ([]bookingdomain.DiscountRule, error) {
	panic("port.DiscountRuleRepository.ListForFacility is not part of the RequestRecurringHire path")
}

// actorInput builds a valid RequestRecurringHireInput for the given actor.
//
// The parameter is named `actor` rather than `subject` deliberately: post
// ADR-0014 the value app.Service expects here is a User.ID, and the tests
// below pass a *subject* only in the one case that asserts such a value is
// refused. Naming it `subject` (as this helper was named through PR #151)
// would state the opposite of the contract it now feeds.
func actorInput(t *testing.T, actor string) bookingapp.RequestRecurringHireInput {
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

// TestRequestRecurringHire_SubjectActorResolvedAtTheSeamThenSucceeds is the
// FLIPPED form of the test this file shipped with in PR #151, which was named
// ...SubjectActorStillBlockedByAppLayerUUIDGuard and asserted the defect on
// purpose. Its own comment specified how to flip it once the follow-up landed:
// "the wantErr expectation becomes a successful template creation, asserting
// createCalls == 1 and the persisted RequestedByUserID being the User's uuid
// rather than the subject". That is what this asserts. T13.2 is that
// follow-up, and #152 is closed.
//
// **The flip has one honest amendment to what PR #151 predicted, and it is
// the substance of ADR-0014.** #151 could not know where the translation
// would land. It landed at the grpcapi boundary, not inside app.Service — so
// the flipped test does not simply pass fixtureSubject to
// RequestRecurringHire and expect success. It calls the seam first, exactly
// as the handler's actor() funnel does, and then the use case. Both halves
// are the real implementations over the real Identity service; nothing here
// is mocked but the repositories.
//
// The companion assertion that the app layer still REFUSES a raw subject
// lives below, in TestRequestRecurringHire_RawSubjectIsRefusedByTheAppLayer.
// Together they say what ADR-0014 rules: subjects stop at the boundary.
func TestRequestRecurringHire_SubjectActorResolvedAtTheSeamThenSucceeds(t *testing.T) {
	t.Parallel()

	seed := mustUser(t, fixtureUserID, fixtureSubject, []identitydomain.Role{identitydomain.RoleClub})
	svc, recurringRepo := newServiceWithRealIdentity(t, seed)

	// The seam, called exactly as grpcapi.Handler.actor does.
	actorUserID, err := svc.ResolveActorUserID(context.Background(), fixtureSubject)
	if err != nil {
		t.Fatalf("ResolveActorUserID(%q) = %v, want the caller's User.ID — this is #146/#152",
			fixtureSubject, err)
	}
	if actorUserID != fixtureUserID {
		t.Fatalf("ResolveActorUserID(%q) = %q, want %q", fixtureSubject, actorUserID, fixtureUserID)
	}

	template, err := svc.RequestRecurringHire(context.Background(), actorInput(t, actorUserID))
	if err != nil {
		t.Fatalf("RequestRecurringHire(resolved actor) = %v, want success", err)
	}

	if recurringRepo.createCalls != 1 {
		t.Fatalf("createCalls = %d, want 1", recurringRepo.createCalls)
	}
	if template.RequestedByUserID != fixtureUserID {
		t.Fatalf("RequestedByUserID = %q, want the User's uuid %q, not the subject %q",
			template.RequestedByUserID, fixtureUserID, fixtureSubject)
	}
	if got := recurringRepo.created[0].RequestedByUserID; got != fixtureUserID {
		t.Fatalf("persisted RequestedByUserID = %q, want %q — the value that reaches the "+
			"repository is the one recurring_hire_templates.requested_by_user_id "+
			"(uuid NOT NULL REFERENCES identity_users) actually receives", got, fixtureUserID)
	}
}

// TestRequestRecurringHire_RawSubjectIsRefusedByTheAppLayer is the preserved
// intent of the pre-flip test: a raw subject handed to app.Service is refused.
//
// What CHANGED is why that is the right answer. Before T13.2 it was the bug —
// the handler had no other value to pass, so this rejected every real caller
// (#152). After ADR-0014 it is the contract: the boundary resolves subjects,
// so a subject arriving here means the seam was bypassed, and failing closed
// is what keeps that bypass loud instead of silently authorizing the wrong
// person. The assertion is unchanged; only its meaning is, which is precisely
// why the test is kept rather than deleted.
func TestRequestRecurringHire_RawSubjectIsRefusedByTheAppLayer(t *testing.T) {
	t.Parallel()

	seed := mustUser(t, fixtureUserID, fixtureSubject, []identitydomain.Role{identitydomain.RoleClub})
	svc, recurringRepo := newServiceWithRealIdentity(t, seed)

	_, err := svc.RequestRecurringHire(context.Background(), actorInput(t, fixtureSubject))

	if !errors.Is(err, bookingdomain.ErrUserNotFound) {
		t.Fatalf("RequestRecurringHire(raw subject) = %v, want %v — below the grpcapi "+
			"boundary an actor is always a User.ID (ADR-0014)", err, bookingdomain.ErrUserNotFound)
	}
	if recurringRepo.createCalls != 0 {
		t.Fatalf("createCalls = %d, want 0 — a rejected request must never reach the repository", recurringRepo.createCalls)
	}
}

// TestResolveActorUserID_UnregisteredSubjectIsUserNotFound proves the seam
// fails closed against the REAL Identity service, not just a fake: a verified
// caller who has never registered resolves to nothing, and gets the sentinel
// grpcapi maps to PermissionDenied (ADR-0014 §6).
func TestResolveActorUserID_UnregisteredSubjectIsUserNotFound(t *testing.T) {
	t.Parallel()

	seed := mustUser(t, fixtureUserID, fixtureSubject, []identitydomain.Role{identitydomain.RoleClub})
	svc, recurringRepo := newServiceWithRealIdentity(t, seed)

	got, err := svc.ResolveActorUserID(context.Background(), "auth0|never-registered")

	if !errors.Is(err, bookingdomain.ErrUserNotFound) {
		t.Fatalf("ResolveActorUserID(unregistered) = %v, want %v", err, bookingdomain.ErrUserNotFound)
	}
	if got != "" {
		t.Fatalf("ResolveActorUserID(unregistered) = %q, want the zero value", got)
	}
	if recurringRepo.createCalls != 0 {
		t.Fatalf("createCalls = %d, want 0", recurringRepo.createCalls)
	}
}
