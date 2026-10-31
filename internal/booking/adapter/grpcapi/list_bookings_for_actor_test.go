// T67.1 — handler-level tests for ListBookingsForActor, the player's own "my
// bookings" read (part of issue #353, "Nothing you book or join is visible
// afterwards").
//
// These run the real grpcapi.Handler over the real app.Service and domain,
// with in-memory fakes for the Postgres adapters — the same shape, and for
// the same stated reason, as discount_regression_test.go and
// recurring_hire_regression_test.go in this package: what is under test here
// (where the actor comes from, which rows are returned, which gRPC code a
// missing principal gets) lives entirely in the handler/app/domain layers, so
// a Postgres round trip would add infrastructure rather than proof. The SQL's
// own half of the contract — that cancelled rows are not filtered out and
// that ORDER BY really orders — is pinned separately by
// internal/booking/adapter/postgres's integration test, because a fake cannot
// prove anything about a query.
package grpcapi_test

import (
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	bookingv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/booking/v1"
)

// bookAs creates an individual booking through the real handler as the given
// subject, on court, starting at the given RFC3339 time for an hour.
//
// It goes through CreateBooking rather than seeding the repository directly
// because the owner is the thing under test: CreateBookingRequest has no
// owner field, so the only way a booking can come to belong to this subject
// is the handler's actor funnel — the same funnel ListBookingsForActor uses
// to read it back. Seeding would prove the read filters on a column; this
// proves the write and the read agree on whose booking it is.
func bookAs(t *testing.T, h *recurringHarness, subject, court, startsAt string) *bookingv1.Booking {
	t.Helper()
	start, err := time.Parse(time.RFC3339, startsAt)
	if err != nil {
		t.Fatalf("bad fixture time %q: %v", startsAt, err)
	}
	resp, err := h.handler.CreateBooking(ctxAs(subject), &bookingv1.CreateBookingRequest{
		CourtId:  court,
		Source:   bookingv1.Source_SOURCE_INDIVIDUAL,
		StartsAt: timestamppb.New(start),
		EndsAt:   timestamppb.New(start.Add(time.Hour)),
	})
	if err != nil {
		t.Fatalf("CreateBooking as %s: %v", subject, err)
	}
	return resp.GetBooking()
}

func listAs(t *testing.T, h *recurringHarness, subject string) []*bookingv1.ActorBooking {
	t.Helper()
	resp, err := h.handler.ListBookingsForActor(ctxAs(subject), &bookingv1.ListBookingsForActorRequest{})
	if err != nil {
		t.Fatalf("ListBookingsForActor as %s: %v", subject, err)
	}
	return resp.GetBookings()
}

// TestListBookingsForActor_ScopesToThePrincipal is the headline property: the
// result set is determined by the verified token and by nothing else.
func TestListBookingsForActor_ScopesToThePrincipal(t *testing.T) {
	h := newRecurringHandler()

	mine := bookAs(t, h, subjectOf(playerUser), courtID(1), "2026-03-01T09:00:00Z")
	theirs := bookAs(t, h, subjectOf(attackerUser), courtID(1), "2026-03-01T11:00:00Z")

	got := listAs(t, h, subjectOf(playerUser))
	if len(got) != 1 {
		t.Fatalf("got %d bookings, want 1 (the caller's own); %q belongs to another user", len(got), theirs.GetId())
	}
	if got[0].GetBooking().GetId() != mine.GetId() {
		t.Errorf("got booking %q, want %q", got[0].GetBooking().GetId(), mine.GetId())
	}
	if owner := got[0].GetBooking().GetOwnerUserId(); owner != userID(playerUser) {
		t.Errorf("OwnerUserId = %q, want the principal's resolved User.ID %q", owner, userID(playerUser))
	}

	// The same call, a different token, a different answer — which is what
	// "the actor is the scope" means operationally.
	other := listAs(t, h, subjectOf(attackerUser))
	if len(other) != 1 || other[0].GetBooking().GetId() != theirs.GetId() {
		t.Errorf("the other principal got %v, want exactly their own booking %q", other, theirs.GetId())
	}
}

// TestListBookingsForActor_RequiresAPrincipal pins codes.Unauthenticated
// specifically, not merely "an error": ADR-0013 §5 keeps "I do not know who
// you are" distinct from "I know you and you may not" (PermissionDenied), and
// a client distinguishes "sign in" from "you cannot do this" by exactly this
// code. The RPC's membership in AuthenticatedMethods() is what makes the
// server-wide interceptor refuse it too — authenticated_test.go proves the
// lists classify every RPC; this proves the handler itself does not serve an
// anonymous caller even if it were reached directly.
func TestListBookingsForActor_RequiresAPrincipal(t *testing.T) {
	h := newRecurringHandler()
	bookAs(t, h, subjectOf(playerUser), courtID(1), "2026-03-01T09:00:00Z")

	_, err := h.handler.ListBookingsForActor(anonymous(), &bookingv1.ListBookingsForActorRequest{})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("code = %v (err %v), want %v", status.Code(err), err, codes.Unauthenticated)
	}
}

// TestListBookingsForActor_UnregisteredSubjectIsPermissionDenied is
// ADR-0014 §6's rule at this endpoint: a caller whose token verified but
// whose subject maps to no User is PermissionDenied, never NotFound — the
// caller is known and simply may not act, and NotFound would make the
// endpoint an enumeration oracle for which subjects have accounts.
func TestListBookingsForActor_UnregisteredSubjectIsPermissionDenied(t *testing.T) {
	h := newRecurringHandler()

	_, err := h.handler.ListBookingsForActor(ctxAs("auth0|never-registered"),
		&bookingv1.ListBookingsForActorRequest{})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("code = %v (err %v), want %v", status.Code(err), err, codes.PermissionDenied)
	}
}

// TestListBookingsForActorRequest_HasNoForgeableActorField is this ticket's
// "a forged actor field changes nothing" requirement, in the strongest form
// available: the field cannot be sent at all.
//
// The older ListRecurringHireTemplatesForActorRequest keeps an
// `actor_user_id` marked deprecated and ignored by the server, because it
// predates T12.7's authentication and removing a proto field is a breaking
// wire change — so that endpoint needs a test proving the server ignores what
// a client can still send. A message born after that fix has no such field to
// ignore, and this test is what stops one being added back: a field naming
// the actor on a read whose entire scope IS the actor would be a field for
// reading another player's booking history, and because the handler takes the
// actor from h.actor(ctx) it would be a field that silently did nothing until
// someone "wired it up".
//
// Read off the generated descriptor rather than the Go struct, so it is the
// wire contract being asserted — the thing the Vue, Swift and Kotlin clients
// actually see.
func TestListBookingsForActorRequest_HasNoForgeableActorField(t *testing.T) {
	fields := (&bookingv1.ListBookingsForActorRequest{}).ProtoReflect().Descriptor().Fields()

	for i := 0; i < fields.Len(); i++ {
		name := string(fields.Get(i).Name())
		for _, forbidden := range []string{"actor", "owner", "user", "subject"} {
			if strings.Contains(name, forbidden) {
				t.Errorf("ListBookingsForActorRequest has field %q. The actor on this read comes from "+
					"the verified principal via Handler.actor and from nowhere else; a request field "+
					"naming a user is a field for reading somebody else's bookings.", name)
			}
		}
	}

	// Stated as its own assertion because it is the current contract and a
	// reader of the proto should find it pinned: the message is empty.
	if fields.Len() != 0 {
		t.Errorf("ListBookingsForActorRequest now has %d field(s); it is documented as empty. "+
			"If a filter was added deliberately, update booking.proto's comment and this test together.",
			fields.Len())
	}
}

// TestListBookingsForActor_IncludesCancelled is the behaviour this endpoint
// exists to get right, and the one most likely to be "fixed" into a bug:
// a cancelled booking must come back, reported as cancelled.
func TestListBookingsForActor_IncludesCancelled(t *testing.T) {
	h := newRecurringHandler()

	live := bookAs(t, h, subjectOf(playerUser), courtID(1), "2026-03-01T09:00:00Z")
	doomed := bookAs(t, h, subjectOf(playerUser), courtID(1), "2026-03-02T09:00:00Z")

	if _, err := h.handler.CancelBooking(ctxAs(subjectOf(playerUser)),
		&bookingv1.CancelBookingRequest{BookingId: doomed.GetId()}); err != nil {
		t.Fatalf("CancelBooking: %v", err)
	}

	got := listAs(t, h, subjectOf(playerUser))
	if len(got) != 2 {
		t.Fatalf("got %d bookings, want 2 — the cancelled one must still be listed, or a user "+
			"cannot tell a booking they cancelled from one that was lost (#353)", len(got))
	}

	byID := map[string]bookingv1.Status{}
	for _, row := range got {
		byID[row.GetBooking().GetId()] = row.GetBooking().GetStatus()
	}
	if s := byID[live.GetId()]; s != bookingv1.Status_STATUS_CONFIRMED {
		t.Errorf("live booking status = %v, want %v", s, bookingv1.Status_STATUS_CONFIRMED)
	}
	if s := byID[doomed.GetId()]; s != bookingv1.Status_STATUS_CANCELLED {
		t.Errorf("cancelled booking status = %v, want %v — reported as itself, not omitted",
			s, bookingv1.Status_STATUS_CANCELLED)
	}
}

// TestListBookingsForActor_CarriesFacilityID covers both halves of the
// facility field: resolved where there is a Facility, and an honest empty
// where the Court belongs to none. In this package's fixture courtID(1)
// belongs to facilityID(1) and every other court belongs to no Facility —
// which is also the real shape of both Courts seeded by 0002_seed.sql.
func TestListBookingsForActor_CarriesFacilityID(t *testing.T) {
	h := newRecurringHandler()

	withFacility := bookAs(t, h, subjectOf(playerUser), courtID(1), "2026-03-01T09:00:00Z")
	without := bookAs(t, h, subjectOf(playerUser), courtID(2), "2026-03-02T09:00:00Z")

	got := listAs(t, h, subjectOf(playerUser))
	if len(got) != 2 {
		t.Fatalf("got %d bookings, want 2 — a Court belonging to no Facility must still be listed", len(got))
	}

	byID := map[string]string{}
	for _, row := range got {
		byID[row.GetBooking().GetId()] = row.GetFacilityId()
	}
	if id := byID[withFacility.GetId()]; id != facilityID(1) {
		t.Errorf("FacilityId = %q, want %q — a client needs it to show the court's NAME rather than "+
			"its uuid (#342)", id, facilityID(1))
	}
	if id := byID[without.GetId()]; id != "" {
		t.Errorf("FacilityId = %q for a Court with no Facility, want empty", id)
	}
}

// TestListBookingsForActor_OrdersByStart pins the ordering the proto promises
// three generated clients. What it proves is that nothing between the
// repository and the wire re-orders the rows; that the SQL's ORDER BY is
// right is the integration test's claim, not this one's.
func TestListBookingsForActor_OrdersByStart(t *testing.T) {
	h := newRecurringHandler()

	// Created latest-first, so passing cannot be an accident of insertion
	// order.
	third := bookAs(t, h, subjectOf(playerUser), courtID(1), "2026-03-03T09:00:00Z")
	first := bookAs(t, h, subjectOf(playerUser), courtID(1), "2026-03-01T09:00:00Z")
	second := bookAs(t, h, subjectOf(playerUser), courtID(1), "2026-03-02T09:00:00Z")

	got := listAs(t, h, subjectOf(playerUser))
	want := []string{first.GetId(), second.GetId(), third.GetId()}
	if len(got) != len(want) {
		t.Fatalf("got %d bookings, want %d", len(got), len(want))
	}
	for i, id := range want {
		if got[i].GetBooking().GetId() != id {
			t.Fatalf("position %d = %q, want %q (ascending starts_at)", i, got[i].GetBooking().GetId(), id)
		}
	}
}

// TestListBookingsForActor_EmptyIsNotAnError: a player who has booked nothing
// gets an empty list and a 200, not a 404. "You have no bookings" is an
// answer, and a NotFound here would also tell a caller something about
// whether a user exists.
func TestListBookingsForActor_EmptyIsNotAnError(t *testing.T) {
	h := newRecurringHandler()

	resp, err := h.handler.ListBookingsForActor(ctxAs(subjectOf(playerUser)),
		&bookingv1.ListBookingsForActorRequest{})
	if err != nil {
		t.Fatalf("ListBookingsForActor: %v, want nil", err)
	}
	if n := len(resp.GetBookings()); n != 0 {
		t.Fatalf("got %d bookings, want 0", n)
	}
}
