package app

import (
	"context"

	"github.com/nhuthuynh/white-label/internal/booking/domain"
)

// ActorBooking is one row of ListBookingsForActor's answer: a Booking,
// together with the id of the Facility that owns its Court.
//
// **Why the FacilityID is not a field on domain.Booking.** It is not a fact
// about a Booking — it is a fact about the Court a Booking references, owned
// by the Facilities context. Putting it on the aggregate would mean either
// denormalising another context's data into this one's table or leaving a
// field that is empty on every path that does not happen to resolve it; this
// type exists so the composition lives where composition belongs (the app
// layer) and internal/booking/domain stays a model of Booking alone.
//
// **Why the read carries it at all.** Issue #342 — closed — was that a user
// was shown a court's uuid where its name belonged. A client cannot resolve a
// court's name from a Booking alone, because Booking does not and must not
// read the courts table (that is the Facilities context's). Handing back the
// Facility id lets the client ask Facilities for the name, which is what
// keeps #342 closed on this new screen rather than reintroduced on it.
type ActorBooking struct {
	Booking    domain.Booking
	FacilityID string
}

// ListBookingsForActor returns every Booking the actor owns — the
// player-facing "my bookings" read (T67.1, part of issue #353: "Nothing you
// book or join is visible afterwards").
//
// # Authorization: the actor's identity IS the scope
//
// The read filters on owner_user_id = actorUserID, so a caller can only ever
// be answered with bookings they themselves own; there is no other party's
// data in reach for an ownership check to protect. That makes it
// structurally the same shape as ListRecurringHireTemplatesForActor and
// deliberately unlike ListCourtBookings, which is public precisely because
// it answers an availability question about a court rather than a question
// about a person.
//
// What makes the scope trustworthy is not this method: it is that its only
// caller obtains actorUserID from the verified token via
// grpcapi.Handler.actor (ADR-0014's seam — auth.RequireSubject, then
// ResolveActorUserID), and that the RPC is listed in
// grpcapi.AuthenticatedMethods() so an unauthenticated call is refused with
// Unauthenticated before reaching here. The request message carries no actor
// field for a caller to forge, which is the property that distinguishes this
// from the pre-T12.7 shape where the actor was a caller-supplied claim.
//
// # Cancelled bookings are included
//
// Deliberately the opposite of ListCourtBookings, and the single most
// important behaviour here. A cancelled Booking is a thing that happened to
// the user, and a list that omitted it would leave them unable to tell "I
// cancelled that" from "my booking was lost" — the second being the complaint
// #353 was filed about. Reporting a cancellation as a cancellation is the
// answer; omitting the row is a different, worse bug wearing the fix's
// clothes. Both the SQL and port.Repository.ListForOwner's contract say so.
//
// # A Court with no Facility is an honest empty, not an error
//
// courts.facility_id is nullable and both Courts seeded by 0002_seed.sql
// still have NULL there (0010_facilities.sql's own note), so on a freshly
// migrated database this is the common case rather than a corner. Such a
// Booking yields an ActorBooking with an empty FacilityID — the booking is
// real and the user must see it — and emphatically not an error that fails
// the whole list. Note that port.FacilityLookup.FacilityIDForCourt, the
// single-court method, answers domain.ErrFacilityNotFound for exactly this
// case; that is why this method uses the batch method instead, whose contract
// omits unresolvable courts rather than failing.
//
// An error *from the lookup itself* is different and is returned: a blank
// facility id on screen is indistinguishable from #342's uuid-instead-of-name
// defect, so a lookup that failed must not be reported as a court with no
// facility.
//
// # No pagination, deliberately not invented here
//
// This list is unbounded — a player with years of history gets all of it in
// one response. That is a real limitation, recorded rather than silently
// accepted, and it is not fixed here because no endpoint in this API has
// pagination and inventing a scheme for one RPC would either prejudge the
// shape for all of them or leave two conventions in one proto. See this
// ticket's PR for the finding.
func (s *Service) ListBookingsForActor(ctx context.Context, actorUserID string) ([]ActorBooking, error) {
	// A malformed actorUserID is answered exactly like an unknown one — an
	// empty list, matching ListCourtBookings' convention for a list-shaped
	// read rather than an error this method never otherwise returns. The
	// guard has to run regardless of how unreachable it looks: the Postgres
	// adapter's mustUUID panics on anything pgtype.UUID.Scan cannot parse,
	// bookings.owner_user_id is a `uuid` column, and grpc installs no
	// recover() — so an unguarded value off this path takes the process down
	// (T10.7, issue #97).
	//
	// Post-ADR-0014 a caller whose subject resolves to no User never reaches
	// this method at all, so this branch is now reachable only by a
	// programming error. It is kept because what it prevents is a crash, not
	// a wrong answer.
	if !uuidShape.MatchString(actorUserID) {
		return []ActorBooking{}, nil
	}

	bookings, err := s.repo.ListForOwner(ctx, actorUserID)
	if err != nil {
		return nil, err
	}
	if len(bookings) == 0 {
		// An allocated empty slice, not nil: the REST mapping must emit []
		// rather than null for "you have no bookings", and skipping the
		// facility lookup for an empty page is the point of checking here.
		return []ActorBooking{}, nil
	}

	facilityByCourt, err := s.facilities.FacilityIDsForCourts(ctx, distinctCourtIDs(bookings))
	if err != nil {
		return nil, err
	}

	out := make([]ActorBooking, 0, len(bookings))
	for _, b := range bookings {
		// A court absent from the map resolves to "" — the port's contract
		// is that absence means "no Facility", which is exactly the value
		// this row should carry.
		out = append(out, ActorBooking{Booking: b, FacilityID: facilityByCourt[b.CourtID]})
	}
	return out, nil
}

// distinctCourtIDs reduces a page of bookings to the set of courts it
// mentions, preserving first-appearance order so the argument passed to the
// lookup is deterministic (a map's iteration order is not, and a
// non-deterministic argument makes a failing test's output unreadable).
//
// De-duplicating matters beyond tidiness: a player's bookings cluster on the
// courts they play on, so the same handful of ids would otherwise be asked
// for once per booking.
func distinctCourtIDs(bookings []domain.Booking) []string {
	seen := make(map[string]struct{}, len(bookings))
	out := make([]string, 0, len(bookings))
	for _, b := range bookings {
		if _, ok := seen[b.CourtID]; ok {
			continue
		}
		seen[b.CourtID] = struct{}{}
		out = append(out, b.CourtID)
	}
	return out
}
