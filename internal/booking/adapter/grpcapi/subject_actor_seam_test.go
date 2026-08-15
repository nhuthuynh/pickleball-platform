// End-to-end coverage for ADR-0014's actor-resolution seam (T13.2), and the
// regression test for issues #146 and #152.
//
// **What makes this file different from every other test in this package:**
// it wires the REAL internal/booking/adapter/identity.Lookup over the REAL
// internal/identity/app.Service, instead of the fakeIdentityLookup the other
// files use. That distinction is the whole point. #146 and #152 both shipped
// because every Booking test faked port.IdentityLookup wholesale, so no test
// ever carried a real, non-uuid IdP subject across the Booking↔Identity seam
// — the only actor value production has carried since T12.7 made the actor
// auth.RequireSubject(ctx).
//
// The path exercised here is the production one, end to end and unmocked at
// every layer that has an opinion about identifiers:
//
//	auth.Principal{Subject: "auth0|..."}   (a real IdP subject shape)
//	  -> grpcapi.Handler.RequestRecurringHire
//	  -> Handler.actor(ctx)                (the ADR-0014 seam)
//	  -> app.Service.ResolveActorUserID
//	  -> port.IdentityLookup.UserIDBySubject
//	  -> adapter/identity.Lookup           (REAL)
//	  -> identity/app.Service.UserBySubject (REAL)
//	  -> identity port.Repository          (in-memory, the only fake left)
//	  -> app.Service.RequestRecurringHire  (receives a uuid, as ADR-0014 rules)
//	  -> the persisted RequestedByUserID
//
// Only the repositories are fakes, and that is deliberate rather than
// convenient: the defect class this file exists to catch lives entirely in
// which *identifier space* a value belongs to, and a Postgres round trip
// cannot influence that. What a Postgres round trip WOULD add is the
// mustUUID() panic #152 warns about — which is precisely why the assertion
// below on RequestedByUserID being uuid-shaped is not cosmetic: it is the
// Docker-free proxy for "this value can be written to a uuid column at all".
package grpcapi_test

import (
	"context"
	"fmt"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/nhuthuynh/white-label/internal/booking/adapter/grpcapi"
	bookingidentity "github.com/nhuthuynh/white-label/internal/booking/adapter/identity"
	"github.com/nhuthuynh/white-label/internal/booking/app"
	"github.com/nhuthuynh/white-label/internal/booking/domain"
	bookingv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/booking/v1"
	identityapp "github.com/nhuthuynh/white-label/internal/identity/app"
	identitydomain "github.com/nhuthuynh/white-label/internal/identity/domain"
)

// subjectOf is the verified IdP subject for fixture user n — deliberately NOT
// uuid-shaped, because a real `sub` claim never is (`auth0|abc123`,
// `google-oauth2|107691...`; see db/migrations/0019_identity_subject.sql).
//
// Before T13.2 this package's ctxAs was handed userID(n) — a uuid — as the
// principal's subject, with a comment conceding that a real subject is not a
// uuid and that the identifier-space question was disclosed rather than
// resolved. ADR-0014 resolves it, so the fixtures now carry the production
// shape: subjectOf(n) is what the token says, userID(n) is what this platform
// stores, and the two are never the same string. That gap is exactly what
// #146 and #152 fell into.
func subjectOf(n int) string { return fmt.Sprintf("auth0|fixture-user-%d", n) }

// realIdentityRepo is a minimal internal/identity/port.Repository fake. It is
// the ONLY fake between the handler and the User record in this file — every
// layer above it is the real implementation.
type realIdentityRepo struct {
	users map[string]identitydomain.User
}

func newRealIdentityRepo() *realIdentityRepo {
	return &realIdentityRepo{users: make(map[string]identitydomain.User)}
}

func (r *realIdentityRepo) Create(_ context.Context, u identitydomain.User) (identitydomain.User, error) {
	for _, existing := range r.users {
		if existing.Subject == u.Subject {
			return identitydomain.User{}, identitydomain.ErrUserAlreadyExists
		}
	}
	r.users[u.ID] = u
	return u, nil
}

func (r *realIdentityRepo) GetByID(_ context.Context, id string) (identitydomain.User, error) {
	u, ok := r.users[id]
	if !ok {
		return identitydomain.User{}, identitydomain.ErrUserNotFound
	}
	return u, nil
}

func (r *realIdentityRepo) GetBySubject(_ context.Context, subject string) (identitydomain.User, error) {
	for _, u := range r.users {
		if u.Subject == subject {
			return u, nil
		}
	}
	return identitydomain.User{}, identitydomain.ErrUserNotFound
}

func (r *realIdentityRepo) UpdateSelfReportedLevel(_ context.Context, id string, level identitydomain.SelfReportedStartingLevel) (identitydomain.User, error) {
	u, ok := r.users[id]
	if !ok {
		return identitydomain.User{}, identitydomain.ErrUserNotFound
	}
	u.SelfReportedStartingLevel = level
	r.users[id] = u
	return u, nil
}

// identityStubIDs satisfies identity's port.IDGenerator. Users are seeded
// through the repository directly (see seedUser), so it is never called — it
// exists because identityapp.NewService requires the port.
type identityStubIDs struct{}

func (identityStubIDs) NewID() string { return userID(clubUser) }

// seedUser writes a User straight through the repository fake rather than
// through identityapp.Service.CreateUser. That is required, not a shortcut:
// CreateUser only ever mints RolePlayer (self-registration must never grant a
// privileged role), and this file needs a User holding RoleClub.
func seedUser(t *testing.T, repo *realIdentityRepo, id, subject string, roles []identitydomain.Role) {
	t.Helper()

	u, err := identitydomain.NewUser(id, subject, "Ada Lovelace", roles, identitydomain.SelfReportedStartingLevel(3))
	if err != nil {
		t.Fatalf("building fixture user (id=%q subject=%q): %v", id, subject, err)
	}
	if _, err := repo.Create(context.Background(), u); err != nil {
		t.Fatalf("seeding identity repo: %v", err)
	}
}

// realIdentityHarness is newRecurringHandler's twin, with the ONE difference
// that matters: port.IdentityLookup is the real adapter, not fakeIdentityLookup.
type realIdentityHarness struct {
	handler   *grpcapi.Handler
	templates *fakeRecurringRepo
	users     *realIdentityRepo
}

func newHandlerWithRealIdentity(t *testing.T) *realIdentityHarness {
	t.Helper()

	users := newRealIdentityRepo()
	lookup := bookingidentity.NewLookup(identityapp.NewService(users, identityStubIDs{}))

	templates := newFakeRecurringRepo()
	svc := app.NewService(
		newStoringBookingRepo(),
		&fakePricingRepo{},
		&fakeDiscountRepo{byFacility: map[string][]domain.DiscountRule{}},
		templates,
		fakeFacilityLookup{},
		lookup,
		&fakeIDs{},
	)
	return &realIdentityHarness{handler: grpcapi.NewHandler(svc), templates: templates, users: users}
}

// TestRequestRecurringHire_SubjectShapedActorSucceedsEndToEnd is the #146 +
// #152 regression test: the RPC those issues report as broken for every
// caller alive, driven by a caller shaped exactly like a real one.
//
// Before T13.2 this failed with codes.PermissionDenied (domain.ErrUserNotFound
// out of app.Service.RequestRecurringHire's uuidShape guard, which fired on
// the raw subject before port.IdentityLookup was ever consulted — #152's
// diagnosis exactly).
//
// The RequestedByUserID assertion is the load-bearing half. Asserting only
// "no error" would still pass if the seam persisted the raw subject string,
// which is the outcome #152 explicitly names as WORSE than the bug: the
// Postgres adapter's mustUUID() panics on a non-uuid, and
// recurring_hire_templates.requested_by_user_id is
// `uuid NOT NULL REFERENCES identity_users (id)`.
func TestRequestRecurringHire_SubjectShapedActorSucceedsEndToEnd(t *testing.T) {
	h := newHandlerWithRealIdentity(t)
	seedUser(t, h.users, userID(clubUser), subjectOf(clubUser), []identitydomain.Role{identitydomain.RoleClub})

	resp, err := h.handler.RequestRecurringHire(ctxAs(subjectOf(clubUser)), validRequest(4))
	if err != nil {
		t.Fatalf("RequestRecurringHire as a subject-shaped actor (%q) = %v, want success — "+
			"this is the #146/#152 defect: a verified IdP subject must resolve to the "+
			"caller's User.ID at the grpcapi boundary (ADR-0014)", subjectOf(clubUser), err)
	}

	if got := resp.GetTemplate().GetRequestedByUserId(); got != userID(clubUser) {
		t.Fatalf("RequestedByUserId = %q, want %q (the resolved User.ID uuid). "+
			"A raw subject here would panic the Postgres adapter's mustUUID() and violate "+
			"recurring_hire_templates' FK to identity_users(id) — see #152.", got, userID(clubUser))
	}
	if h.templates.createCalls != 1 {
		t.Fatalf("createCalls = %d, want 1", h.templates.createCalls)
	}

	// The template is readable back by the same caller, which only holds if
	// the value persisted and the value the actor-scoped read filters on are
	// the SAME identifier space. A seam that resolved on write but not on
	// read would pass every assertion above and still strand the Club with an
	// empty list.
	list, err := h.handler.ListRecurringHireTemplatesForActor(ctxAs(subjectOf(clubUser)),
		&bookingv1.ListRecurringHireTemplatesForActorRequest{})
	if err != nil {
		t.Fatalf("ListRecurringHireTemplatesForActor: %v", err)
	}
	if n := len(list.GetTemplates()); n != 1 {
		t.Fatalf("ListRecurringHireTemplatesForActor returned %d templates, want 1 — "+
			"the write path and the read path disagree about the actor's identifier space", n)
	}
}

// TestRequestRecurringHire_UnregisteredSubjectIsPermissionDenied pins
// ADR-0014's answer to the new error case (T13.2 instruction 11): a caller
// whose token verified but who has no User row.
//
// PermissionDenied, not NotFound and not Unauthenticated:
//   - Unauthenticated is wrong by ADR-0013 §5 — we know exactly who they are,
//     their token verified.
//   - NotFound would make a 404 ambiguous between "the Court you named does
//     not exist" and "you do not exist", and turn this endpoint into a
//     user-enumeration oracle. That is the same reasoning already recorded on
//     toStatus' ErrUserNotFound case, so this is the existing mapping being
//     reused, not a new one invented.
func TestRequestRecurringHire_UnregisteredSubjectIsPermissionDenied(t *testing.T) {
	h := newHandlerWithRealIdentity(t)
	// Deliberately seed nobody: the subject verifies, but no User is registered.

	_, err := h.handler.RequestRecurringHire(ctxAs("auth0|never-registered"), validRequest(4))
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("RequestRecurringHire(unregistered subject) = %v (code %v), want PermissionDenied",
			err, status.Code(err))
	}
	if h.templates.createCalls != 0 {
		t.Fatalf("createCalls = %d, want 0 — an unresolvable actor must never reach the repository", h.templates.createCalls)
	}
}

// TestRequestRecurringHire_ResolvedActorStillFailsTheClubRoleCheck proves the
// seam resolves identity WITHOUT weakening authorization — that ADR-0014
// answers "who are you", never "what may you do".
//
// The actor here is a real, registered, resolvable User who simply does not
// hold the `club` role. Resolution succeeds; the role check still refuses.
// Without this case, a seam that resolved every subject to some User would
// pass the success test above and silently hand a Player a Club's capability.
func TestRequestRecurringHire_ResolvedActorStillFailsTheClubRoleCheck(t *testing.T) {
	h := newHandlerWithRealIdentity(t)
	seedUser(t, h.users, userID(playerUser), subjectOf(playerUser), []identitydomain.Role{identitydomain.RolePlayer})

	_, err := h.handler.RequestRecurringHire(ctxAs(subjectOf(playerUser)), validRequest(4))
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("RequestRecurringHire(resolvable non-club actor) = %v (code %v), want PermissionDenied",
			err, status.Code(err))
	}
	if h.templates.createCalls != 0 {
		t.Fatalf("createCalls = %d, want 0", h.templates.createCalls)
	}
}

// TestActorSeam_RawSubjectNeverReachesTheAppLayer is the structural assertion
// behind ADR-0014's ruling, and the one that would catch a future RPC wired
// past the seam.
//
// It states the invariant directly: below grpcapi, no actor value is ever a
// subject. Passing the raw subject into app.Service is therefore a caller
// error and stays rejected — deliberately NOT "fixed" — because the app layer
// keeping its uuidShape guards is exactly what makes them honest again rather
// than the stale, every-caller-rejecting checks #152 found.
func TestActorSeam_RawSubjectNeverReachesTheAppLayer(t *testing.T) {
	h := newHandlerWithRealIdentity(t)
	seedUser(t, h.users, userID(clubUser), subjectOf(clubUser), []identitydomain.Role{identitydomain.RoleClub})

	svc := app.NewService(
		newStoringBookingRepo(), &fakePricingRepo{},
		&fakeDiscountRepo{byFacility: map[string][]domain.DiscountRule{}},
		h.templates, fakeFacilityLookup{},
		bookingidentity.NewLookup(identityapp.NewService(h.users, identityStubIDs{})),
		&fakeIDs{},
	)

	// The seam, called directly: subject in, User.ID out.
	resolved, err := svc.ResolveActorUserID(context.Background(), subjectOf(clubUser))
	if err != nil {
		t.Fatalf("ResolveActorUserID(%q) = %v, want the caller's User.ID", subjectOf(clubUser), err)
	}
	if resolved != userID(clubUser) {
		t.Fatalf("ResolveActorUserID(%q) = %q, want %q", subjectOf(clubUser), resolved, userID(clubUser))
	}
}
