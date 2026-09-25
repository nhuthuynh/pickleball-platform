package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nhuthuynh/white-label/internal/booking/app"
	"github.com/nhuthuynh/white-label/internal/booking/domain"
)

// T59.1 (closes issue #296) — a malformed OwnerUserID is rejected with a
// domain error rather than reaching the adapter's mustUUID and panicking.
//
// # The asymmetry this closes
//
// `CreateBooking` shape-checks one of its two caller-ish ids and not the
// other. `CourtID` has a `uuidShape` guard, added by T10.7 after issue #97,
// because a malformed one reached `mustUUID` and took the process down.
// `OwnerUserID` — added by T55.1, persisted as
// `bookings.owner_user_id uuid NOT NULL REFERENCES identity_users (id)` and
// written by the adapter with the same `mustUUID` — had no equivalent.
//
// The empty case was already covered: `domain.NewBooking` rejects `""` with
// `ErrEmptyOwnerUserID`. It is the malformed-but-non-empty case that reached
// the panic.
//
// # Why this was not reachable, and why it is still worth closing
//
// Every current supplier is structurally a uuid: the grpcapi handler resolves
// it from the verified principal, Social Play passes `Game.HostID`,
// Competitions passes `Competition.HostID`, and `ApproveRecurringHire`
// passes the template's `RequestedByUserID` — all uuid columns. So no test
// here can demonstrate a production panic; what it demonstrates is that the
// *guard* exists, which is the property that survives the next supplier
// being added.
//
// #97 is the precedent for why that matters: `CourtID` was "obviously"
// well-formed until it wasn't.

// Fixtures local to this file. ownerA (booking_ownership_test.go) is the
// package's well-formed owner; these two are ids this file seeds directly.
const (
	ownerShapeBookingID   = "3f2504e0-4f89-11d3-9a0c-0305e82c3f01"
	ownerShapeReferenceID = "3f2504e0-4f89-11d3-9a0c-0305e82c3f02"
)

// newOwnerShapeSvc wires the full ServiceOptions this package's constructor
// validates, so a test failure here is about the guard and never about a
// missing dependency.
func newOwnerShapeSvc(repo *inMemoryRepo) *app.Service {
	return app.NewService(app.ServiceOptions{
		Bookings:       repo,
		PricingRules:   &fakePricingRepo{},
		DiscountRules:  newFakeDiscountRepo(),
		RecurringHires: newFakeRecurringHireRepo(),
		Facilities:     &fakeFacilityLookup{},
		Identity:       &fakeIdentityLookup{},
		IDs:            &sequentialIDs{},
	})
}

// A malformed owner is refused, with the sentinel that says what is wrong.
func TestCreateBooking_MalformedOwnerUserIDIsRejected(t *testing.T) {
	t.Parallel()

	svc := newOwnerShapeSvc(newInMemoryRepo())

	_, err := svc.CreateBooking(context.Background(), app.CreateBookingInput{
		CourtID:     courtID(1),
		Source:      domain.SourceIndividual,
		Range:       mustTimeRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z"),
		OwnerUserID: "not-a-uuid",
	})
	if !errors.Is(err, domain.ErrInvalidOwnerReference) {
		t.Fatalf("malformed OwnerUserID = %v, want %v", err, domain.ErrInvalidOwnerReference)
	}
}

// The empty case keeps its own, more specific sentinel. Two different
// answers because they are two different problems: an empty owner is a
// caller that supplied nothing, a malformed one is a caller that supplied
// something wrong. Pinned so a future tidy-up does not collapse them.
func TestCreateBooking_EmptyOwnerUserIDKeepsItsOwnSentinel(t *testing.T) {
	t.Parallel()

	svc := newOwnerShapeSvc(newInMemoryRepo())

	_, err := svc.CreateBooking(context.Background(), app.CreateBookingInput{
		CourtID:     courtID(1),
		Source:      domain.SourceIndividual,
		Range:       mustTimeRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z"),
		OwnerUserID: "",
	})
	if !errors.Is(err, domain.ErrEmptyOwnerUserID) {
		t.Fatalf("empty OwnerUserID = %v, want %v", err, domain.ErrEmptyOwnerUserID)
	}
	if errors.Is(err, domain.ErrInvalidOwnerReference) {
		t.Fatal("an empty owner must not answer the malformed-owner sentinel — they are different faults")
	}
}

// Nothing was persisted. The guard must run before the repository is
// touched, for the reason the CourtID guard documents: the adapter method
// downstream is the one that panics, so a refusal that happens after the
// call is no refusal at all.
func TestCreateBooking_MalformedOwnerPersistsNothing(t *testing.T) {
	t.Parallel()

	repo := newInMemoryRepo()
	svc := newOwnerShapeSvc(repo)

	if _, err := svc.CreateBooking(context.Background(), app.CreateBookingInput{
		CourtID:     courtID(1),
		Source:      domain.SourceIndividual,
		Range:       mustTimeRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z"),
		OwnerUserID: "not-a-uuid",
	}); !errors.Is(err, domain.ErrInvalidOwnerReference) {
		t.Fatalf("setup: want %v, got %v", domain.ErrInvalidOwnerReference, err)
	}

	if got := len(repo.bookings); got != 0 {
		t.Fatalf("repository holds %d bookings, want 0 — the guard must precede any write", got)
	}
}

// The control: a well-formed owner still goes through. Without this, a guard
// that rejected every owner would pass everything above.
func TestCreateBooking_WellFormedOwnerStillSucceeds(t *testing.T) {
	t.Parallel()

	svc := newOwnerShapeSvc(newInMemoryRepo())

	b, err := svc.CreateBooking(context.Background(), app.CreateBookingInput{
		CourtID:     courtID(1),
		Source:      domain.SourceIndividual,
		Range:       mustTimeRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z"),
		OwnerUserID: ownerA,
	})
	if err != nil {
		t.Fatalf("well-formed owner = %v, want success", err)
	}
	if b.OwnerUserID != ownerA {
		t.Fatalf("OwnerUserID = %q, want %q", b.OwnerUserID, ownerA)
	}
}

// CourtID's guard still fires, and still first. Ordering is observable: a
// request malformed in BOTH ids gets the CourtID answer, because that guard
// runs first and this ticket did not reorder it.
//
// Pinned because the alternative — the new guard jumping ahead of the
// existing one — would silently change what an existing client sees for an
// input that has always been rejected.
func TestCreateBooking_CourtIDGuardStillRunsFirst(t *testing.T) {
	t.Parallel()

	svc := newOwnerShapeSvc(newInMemoryRepo())

	_, err := svc.CreateBooking(context.Background(), app.CreateBookingInput{
		CourtID:     "also-not-a-uuid",
		Source:      domain.SourceIndividual,
		Range:       mustTimeRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z"),
		OwnerUserID: "not-a-uuid",
	})
	if !errors.Is(err, domain.ErrInvalidCourtReference) {
		t.Fatalf("both ids malformed = %v, want the CourtID sentinel %v (guard order unchanged)", err, domain.ErrInvalidCourtReference)
	}
}

// T59.1 instruction 4 — the symmetry question, answered as a test rather
// than as prose.
//
// #296 asks whether `CancelBookingsForReference`'s `actorUserID` (T55.2)
// wants the same guard. **It does not, and this test is why.** That value is
// only ever *compared* — `EnsureOwner` checks it against a stored
// `OwnerUserID` — and never written, so it cannot reach `mustUUID` and
// cannot panic. A malformed actor simply matches no booking's owner and is
// refused as not-the-owner, which is the correct answer: a caller who
// supplies a malformed id is not the owner.
//
// Adding a shape guard there would convert a correct `PermissionDenied` into
// an `InvalidArgument`, telling an unauthorized caller that their id was
// *shaped* wrong — a small disclosure, for no safety gain, on the endpoint
// #144 was filed about. So the asymmetry between the two fields is
// deliberate, and pinned here so it is not "fixed" later for consistency's
// own sake.
func TestCancelBookingsForReference_MalformedActorIsRefusedNotErrored(t *testing.T) {
	t.Parallel()

	repo := newInMemoryRepo()
	repo.bookings[ownerShapeBookingID] = domain.Booking{
		ID:          ownerShapeBookingID,
		CourtID:     courtID(1),
		Source:      domain.SourceGame,
		Status:      domain.StatusConfirmed,
		Range:       mustTimeRange(t, "2026-08-03T09:00:00Z", "2026-08-03T10:00:00Z"),
		ReferenceID: ownerShapeReferenceID,
		OwnerUserID: ownerA,
	}
	svc := newOwnerShapeSvc(repo)

	_, err := svc.CancelBookingsForReference(context.Background(), ownerShapeReferenceID, "not-a-uuid")
	if !errors.Is(err, domain.ErrNotBookingOwner) {
		t.Fatalf("malformed actor = %v, want %v — a compared-only value needs no shape guard", err, domain.ErrNotBookingOwner)
	}
	// And the booking is untouched: a refused cascade cancels nothing.
	if got := repo.bookings[ownerShapeBookingID].Status; got != domain.StatusConfirmed {
		t.Fatalf("booking status = %q, want it untouched (confirmed)", got)
	}
}
