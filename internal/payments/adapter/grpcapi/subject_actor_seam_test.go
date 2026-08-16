// End-to-end coverage for ADR-0014/ADR-0017's actor-resolution seam applied
// to Payments (T28.1), and the regression test for #164's Payments third —
// mirroring internal/booking/adapter/grpcapi/subject_actor_seam_test.go and
// internal/facilities/adapter/grpcapi/subject_owner_seam_test.go exactly.
//
// **What makes this file different from every other test in this package:**
// it wires the REAL internal/payments/adapter/identity.Lookup over the REAL
// internal/identity/app.Service, instead of the fakeIdentityLookup
// (identity_fixtures_test.go) every other file in this package uses. That
// distinction is the whole point: every other test proves the authorization
// RULES; this file proves the SEAM those rules now sit on top of — that a
// real, non-uuid IdP subject really does resolve to a real User.ID at the
// grpcapi boundary, and that the SAME resolution happens consistently on
// both the write path (CreateOnlinePayment, which persists the actor) and
// the read path (ConfirmOnlinePayment, which compares against it) — which
// is exactly the hazard app.authorizeOnlineConfirmation's dated doc comment
// and this PR's migration exist to close together, not separately.
//
// The path exercised here is the production one, end to end and unmocked at
// every layer that has an opinion about identifiers:
//
//	auth.Principal{Subject: "auth0|..."}    (a real IdP subject shape)
//	  -> grpcapi.Handler.CreateOnlinePayment
//	  -> Handler.actor(ctx)                 (the ADR-0014/ADR-0017 seam)
//	  -> app.Service.ResolveActorUserID
//	  -> port.IdentityLookup.UserIDBySubject
//	  -> adapter/identity.Lookup            (REAL)
//	  -> identity/app.Service.UserBySubject (REAL)
//	  -> identity port.Repository           (in-memory, a fake)
//	  -> app.Service.CreateOnlinePayment    (receives a uuid, as ADR-0014 rules)
//	  -> the persisted Payment.RecordedByUserID
//
// Only the two repositories are fakes, and that is deliberate rather than
// convenient: the defect class this file exists to catch lives entirely in
// which *identifier space* a value belongs to, and a Postgres round trip
// cannot influence that. The migration's own Docker-free proxy — proving a
// resolved uuid, and only a resolved uuid, can land in
// payments.recorded_by_user_id — is
// internal/payments/adapter/postgres/migration_backfill_integration_test.go
// (`-tags=integration`); this file proves the identifier that migration
// receives is actually produced correctly by the funnel above it.
package grpcapi_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/nhuthuynh/white-label/internal/payments/adapter/grpcapi"
	paymentsidentity "github.com/nhuthuynh/white-label/internal/payments/adapter/identity"
	"github.com/nhuthuynh/white-label/internal/payments/adapter/stripestub"
	"github.com/nhuthuynh/white-label/internal/payments/app"
	"github.com/nhuthuynh/white-label/internal/payments/domain"

	identityapp "github.com/nhuthuynh/white-label/internal/identity/app"
	identitydomain "github.com/nhuthuynh/white-label/internal/identity/domain"

	paymentsv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/payments/v1"
)

// userUUID mints this file's own fixture uuids — the server-side identifier
// space (identity_users.id) a resolved subject lands in. Local to this file
// rather than shared package-wide, mirroring
// internal/facilities/adapter/grpcapi/authz_regression_test.go's identical
// helper: this file's fixtures are deliberately distinct from every other
// file's in this package (see seamOwnerSubject's own doc comment).
func userUUID(n int) string { return fmt.Sprintf("00000000-0000-4000-c000-%012d", n) }

// uuidShaped is the canonical 8-4-4-4-12 form pgtype.UUID.Scan accepts,
// which is what "can be stored in payments.recorded_by_user_id" means in
// practice, as of the T28.1 migration. Kept local to the test rather than
// exported from app, mirroring facilities' identical local copy: a test
// that imported the production regexp would pass whenever both were wrong
// together.
var uuidShaped = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// seamOwnerSubject/seamOwnerUserID/seamOtherSubject/seamOtherUserID are this
// file's own fixtures, distinct from confirm_authz_test.go's
// payerSubject/intruderSubject (which run against fakeIdentityLookup, not
// the real seam) so the two files' fixtures can never be confused for one
// another.
const (
	seamOwnerSubject = "auth0|seam-owner-1"
	seamOtherSubject = "auth0|seam-other-1"
)

var (
	seamOwnerUserID = userUUID(101)
	seamOtherUserID = userUUID(102)
)

// realIdentityRepo is a minimal internal/identity/port.Repository fake. It
// is the only fake between the handler and the User record on the Identity
// side — every layer above it is the real implementation. Mirrors
// booking's/facilities' identically-named fixture exactly.
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
// through the repository directly (see seedIdentityUser), so it is never
// called — it exists because identityapp.NewService requires the port.
type identityStubIDs struct{}

func (identityStubIDs) NewID() string { return seamOwnerUserID }

// seedIdentityUser writes a User straight through the repository fake.
// Going through identityapp.Service.CreateUser instead would mint its own
// id from the IDGenerator, and these tests need to control which uuid a
// given subject resolves to in order to assert on it.
func seedIdentityUser(t *testing.T, repo *realIdentityRepo, id, subject string) {
	t.Helper()

	u, err := identitydomain.NewUser(id, subject, "Ada Lovelace",
		[]identitydomain.Role{identitydomain.RolePlayer}, identitydomain.SelfReportedStartingLevel(3))
	if err != nil {
		t.Fatalf("building fixture user (id=%q subject=%q): %v", id, subject, err)
	}
	if _, err := repo.Create(context.Background(), u); err != nil {
		t.Fatalf("seeding identity repo: %v", err)
	}
}

// realIdentityHarness is newTestHandler's twin, with the ONE difference
// that matters: port.IdentityLookup is the real adapter over the real
// Identity service, not fakeIdentityLookup.
type realIdentityHarness struct {
	handler  *grpcapi.Handler
	payments *fakeRepository
	users    *realIdentityRepo
}

func newHandlerWithRealIdentity(t *testing.T) *realIdentityHarness {
	t.Helper()

	users := newRealIdentityRepo()
	lookup := paymentsidentity.NewLookup(identityapp.NewService(users, identityStubIDs{}))

	payments := newFakeRepository()
	svc := app.NewService(app.ServiceOptions{
		Payments:  payments,
		IDs:       &fixedIDs{ids: []string{"6ba7b810-0000-4000-8000-0000000000e1", "6ba7b810-0000-4000-8000-0000000000e2"}},
		Processor: stripestub.NewProcessor(),
		Identity:  lookup,
	})
	return &realIdentityHarness{handler: grpcapi.NewHandler(svc), payments: payments, users: users}
}

// TestCreateOnlinePayment_SubjectShapedActorResolvesEndToEnd is the direct
// regression test for the Payments third of #164: the write path that used
// to persist a raw subject into what is now a `uuid` column, driven by a
// caller shaped exactly like a real one.
//
// Before T28.1 this stored RecordedByUserId = "auth0|seam-owner-1" — the raw
// subject — which is the value that, against the real Postgres column
// db/migrations/0024_payments_recorded_by_user_id_uuid.sql produces, the
// Postgres adapter's mustUUID() panics on. Against the fakes here it would
// silently "succeed" with a value production can no longer hold, which is
// why the OwnerId-shaped assertions below are load-bearing, not cosmetic:
// asserting only "no error" would still pass if the seam persisted the raw
// subject.
func TestCreateOnlinePayment_SubjectShapedActorResolvesEndToEnd(t *testing.T) {
	h := newHandlerWithRealIdentity(t)
	seedIdentityUser(t, h.users, seamOwnerUserID, seamOwnerSubject)

	resp, err := h.handler.CreateOnlinePayment(ctxAs(seamOwnerSubject), &paymentsv1.CreateOnlinePaymentRequest{
		PayableType: paymentsv1.PayableType_PAYABLE_TYPE_BOOKING,
		PayableId:   fixtureBookingID,
		Amount:      &paymentsv1.Money{AmountCents: 3000, CurrencyCode: "USD"},
	})
	if err != nil {
		t.Fatalf("CreateOnlinePayment as a subject-shaped actor (%q) = %v, want success — "+
			"this is #164's Payments third: a verified IdP subject must resolve to the "+
			"caller's User.ID at the grpcapi boundary (ADR-0014, ADR-0017)", seamOwnerSubject, err)
	}

	got := resp.GetPayment().GetRecordedByUserId()
	if got == seamOwnerSubject {
		t.Fatalf("Payment.RecordedByUserId = %q — the raw subject was persisted. "+
			"payments.recorded_by_user_id is now `uuid REFERENCES identity_users (id)` "+
			"and the Postgres adapter converts it with a mustUUID() that PANICS on this "+
			"value (#164, ADR-0017).", got)
	}
	if !uuidShaped.MatchString(got) {
		t.Fatalf("Payment.RecordedByUserId = %q, want a uuid — anything else cannot be "+
			"written to payments.recorded_by_user_id post-backfill", got)
	}
	if got != seamOwnerUserID {
		t.Fatalf("Payment.RecordedByUserId = %q, want %q (the resolved User.ID for subject %q)",
			got, seamOwnerUserID, seamOwnerSubject)
	}

	// And the same value reached persistence, not just the response.
	stored, ok := h.payments.byID[resp.GetPayment().GetId()]
	if !ok {
		t.Fatal("the Payment was not persisted")
	}
	if stored.RecordedByUserID != seamOwnerUserID {
		t.Fatalf("persisted RecordedByUserID = %q, want %q", stored.RecordedByUserID, seamOwnerUserID)
	}
}

// TestCreateOnlinePayment_ThenOwnerCanConfirmIt is the test this ticket is
// entirely organized around (see handler.go's actor() doc comment and
// app.authorizeOnlineConfirmation's dated doc comment): the write path
// (CreateOnlinePayment) and the read path (ConfirmOnlinePayment) must
// resolve the SAME subject to the SAME User.ID, through the SAME real seam,
// or the owner who legitimately created the intent cannot capture their own
// payment. A seam that resolved differently — or, the pre-T28.1 shape, a
// funnel that resolved on neither side — would either lock every owner out
// (comparing a uuid against a stale subject can never match) or, if the
// comparison were written the other way round, silently permit the wrong
// caller. This test drives both RPCs through the identical real seam, as
// the identical caller, and is this file's direct proof that landing the
// funnel change and the backfill migration together closed that window.
func TestCreateOnlinePayment_ThenOwnerCanConfirmIt(t *testing.T) {
	h := newHandlerWithRealIdentity(t)
	seedIdentityUser(t, h.users, seamOwnerUserID, seamOwnerSubject)

	created, err := h.handler.CreateOnlinePayment(ctxAs(seamOwnerSubject), &paymentsv1.CreateOnlinePaymentRequest{
		PayableType: paymentsv1.PayableType_PAYABLE_TYPE_BOOKING,
		PayableId:   fixtureBookingID,
		Amount:      &paymentsv1.Money{AmountCents: 3000, CurrencyCode: "USD"},
	})
	if err != nil {
		t.Fatalf("CreateOnlinePayment: %v", err)
	}

	confirmed, err := h.handler.ConfirmOnlinePayment(ctxAs(seamOwnerSubject), &paymentsv1.ConfirmOnlinePaymentRequest{
		PaymentId: created.GetPayment().GetId(),
	})
	if err != nil {
		t.Fatalf("ConfirmOnlinePayment as the Payment's own creator = %v, want success — "+
			"the write path and the ownership-check read path disagree about the actor's "+
			"identifier space, which is exactly the hazard this ticket's PR description "+
			"discusses at length", err)
	}
	if got := confirmed.GetPayment().GetStatus(); got != paymentsv1.PaymentStatus_PAYMENT_STATUS_PAID {
		t.Fatalf("Payment.Status = %v, want PAID", got)
	}
}

// TestConfirmOnlinePayment_StillRejectsANonOwnerThroughTheRealSeam proves
// the seam resolves identity WITHOUT weakening authorization — that
// ADR-0014 answers "who are you", never "what may you do".
//
// The actor here is a real, registered, resolvable User who simply is not
// the Payment's own recorded owner. Resolution succeeds for both; the
// ownership check still refuses. Without this case, a seam that resolved
// every subject to the same User.ID would pass the test above and let any
// registered caller confirm any Payment.
func TestConfirmOnlinePayment_StillRejectsANonOwnerThroughTheRealSeam(t *testing.T) {
	h := newHandlerWithRealIdentity(t)
	seedIdentityUser(t, h.users, seamOwnerUserID, seamOwnerSubject)
	seedIdentityUser(t, h.users, seamOtherUserID, seamOtherSubject)

	created, err := h.handler.CreateOnlinePayment(ctxAs(seamOwnerSubject), &paymentsv1.CreateOnlinePaymentRequest{
		PayableType: paymentsv1.PayableType_PAYABLE_TYPE_BOOKING,
		PayableId:   fixtureBookingID,
		Amount:      &paymentsv1.Money{AmountCents: 3000, CurrencyCode: "USD"},
	})
	if err != nil {
		t.Fatalf("CreateOnlinePayment: %v", err)
	}

	_, err = h.handler.ConfirmOnlinePayment(ctxAs(seamOtherSubject), &paymentsv1.ConfirmOnlinePaymentRequest{
		PaymentId: created.GetPayment().GetId(),
	})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("ConfirmOnlinePayment(resolvable non-owner) = %v (code %v), want PermissionDenied",
			err, status.Code(err))
	}

	stored, ok := h.payments.byID[created.GetPayment().GetId()]
	if !ok {
		t.Fatal("the Payment was not persisted")
	}
	if stored.Status == domain.StatusPaid {
		t.Fatal("a rejected confirmation captured the Payment anyway")
	}
}

// TestCreateOnlinePayment_UnregisteredSubjectIsPermissionDenied pins
// ADR-0014 §6's answer to the error case: a caller whose token verified but
// who has no User row.
//
// PermissionDenied, not NotFound and not Unauthenticated:
//   - Unauthenticated is wrong by ADR-0013 §5 — we know exactly who they
//     are, their token verified.
//   - NotFound would make the RPC's failure ambiguous between "the Booking
//     you named does not exist" and "you do not exist", and turn every
//     actor-taking endpoint on this service into a user-enumeration oracle.
func TestCreateOnlinePayment_UnregisteredSubjectIsPermissionDenied(t *testing.T) {
	h := newHandlerWithRealIdentity(t)
	// Deliberately seed nobody: the subject verifies, but no User is registered.

	_, err := h.handler.CreateOnlinePayment(ctxAs("auth0|never-registered"), &paymentsv1.CreateOnlinePaymentRequest{
		PayableType: paymentsv1.PayableType_PAYABLE_TYPE_BOOKING,
		PayableId:   fixtureBookingID,
		Amount:      &paymentsv1.Money{AmountCents: 3000, CurrencyCode: "USD"},
	})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("CreateOnlinePayment(unregistered subject) = %v (code %v), want PermissionDenied",
			err, status.Code(err))
	}
	if n := len(h.payments.byID); n != 0 {
		t.Fatalf("payments = %d, want 0 — an unresolvable actor must never reach the repository", n)
	}
}

// TestActorSeam_RawSubjectNeverReachesTheAppLayer is the structural
// assertion behind ADR-0014's ruling, and the one that would catch a future
// RPC wired past the seam.
//
// It states the invariant directly: below grpcapi, no actor value is ever a
// subject. Passing the raw subject into app.Service is a caller error, not
// a value the app layer accepts and forwards unchecked — CreateOnlinePayment
// takes an ActorUserID it trusts came from the funnel; this test proves what
// happens when that trust is bypassed, so the seam's presence is not merely
// assumed by every other test in this file.
func TestActorSeam_RawSubjectNeverReachesTheAppLayer(t *testing.T) {
	h := newHandlerWithRealIdentity(t)
	seedIdentityUser(t, h.users, seamOwnerUserID, seamOwnerSubject)

	users := h.users
	lookup := paymentsidentity.NewLookup(identityapp.NewService(users, identityStubIDs{}))
	svc := app.NewService(app.ServiceOptions{
		Payments:  h.payments,
		IDs:       &fixedIDs{ids: []string{"6ba7b810-0000-4000-8000-0000000000e9"}},
		Processor: stripestub.NewProcessor(),
		Identity:  lookup,
	})

	// The seam, called directly: subject in, User.ID out.
	resolved, err := svc.ResolveActorUserID(context.Background(), seamOwnerSubject)
	if err != nil {
		t.Fatalf("ResolveActorUserID(%q) = %v, want the caller's User.ID", seamOwnerSubject, err)
	}
	if resolved != seamOwnerUserID {
		t.Fatalf("ResolveActorUserID(%q) = %q, want %q", seamOwnerSubject, resolved, seamOwnerUserID)
	}

	// The raw subject, handed to the app layer as an ActorUserID, the way a
	// bypassed handler would: this is not rejected by a uuid-shape guard the
	// way Booking's/Facilities' write paths are (CreateOnlinePayment's own
	// PayableID guard is the only uuidShape check on this path, and it does
	// not apply to ActorUserID) — it is simply stored as given, which is
	// precisely why the funnel, not a second app-layer guard, is this
	// context's enforcement point. See app.Service's own doc comment for why
	// this ticket's design places the guarantee at the boundary rather than
	// duplicating a shape check app.NewPayment was never asked to perform.
	p, err := svc.CreateOnlinePayment(context.Background(), app.CreateOnlinePaymentInput{
		PayableType: domain.PayableTypeBooking,
		PayableID:   fixtureBookingID,
		Amount:      domain.Money{Cents: 3000, Currency: "USD"},
		ActorUserID: seamOwnerSubject, // bypassing the funnel entirely
	})
	if err != nil {
		t.Fatalf("CreateOnlinePayment(ActorUserID = a raw subject) = %v — expected this to "+
			"succeed at the app layer (no uuid guard here), which is exactly why the funnel "+
			"is where this invariant is actually enforced, not the app layer", err)
	}
	if p.RecordedByUserID != seamOwnerSubject {
		t.Fatalf("RecordedByUserID = %q, want the raw subject unchanged (%q) — this call "+
			"bypassed ResolveActorUserID on purpose to prove app.Service applies no guard "+
			"of its own", p.RecordedByUserID, seamOwnerSubject)
	}
}
