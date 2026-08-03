package domain

// Status is a Game's lifecycle state. Mirrors booking.Status's shape: a
// cancelled Game no longer holds its slot (later tickets: a cancelled Game
// should eventually cancel its Bookings/Registrations too, but that cascade
// is explicitly out of scope for T5.1 — see PR description).
type Status string

const (
	StatusScheduled Status = "scheduled"
	StatusCancelled Status = "cancelled"
)

// Game is the Social Play aggregate a Host creates: a fixed-capacity game
// session reserving one or more courts for a time range. CourtIDs is a
// slice (not a single CourtID like Booking) because a Game can span
// multiple courts; T5.3 reserves one Booking per court via
// port.CourtReservation.
type Game struct {
	ID         string
	HostID     string
	// FacilityID is the pre-Facilities-context opaque free-text venue
	// identifier (db/migrations/0005_socialplay.sql). DEPRECATED (T8.3):
	// Facilities didn't exist when Social Play was originally built, so
	// this field has no real relationship to anything — prefer
	// VenueFacilityID, a real FK into the Facilities context. Left in
	// place, unreferenced by any new code path, until nothing depends on
	// it anymore (see db/migrations/0011_socialplay_facility_fk.sql).
	FacilityID string
	// VenueFacilityID is a real Facilities-context Facility ID (T8.3),
	// reconciling the opaque FacilityID above with facilities.id uuid.
	// Validated for existence (not merely format) by
	// app.Service.ScheduleGame via port.FacilityLookup — NewGame itself
	// only stores it, the same way it never validated FacilityID's
	// existence either; existence requires the Facilities context, which
	// this pure domain package must never import (CLAUDE.md rule 2/3).
	// Optional: an empty string means this Game has no known venue
	// Facility yet (db/migrations/0011_socialplay_facility_fk.sql adds it
	// as a nullable column precisely so this is legal), and
	// ScheduleGame skips the FacilityLookup check entirely in that case.
	VenueFacilityID string
	CourtIDs        []string
	Range           TimeRange
	Capacity        int
	Status          Status
	// PaymentMethod is the Host's declared accepted payment method(s) for
	// this Game (T8.6). See payment_method.go's doc comment for why this is
	// deliberately distinct from Registration.PaymentStatus.
	PaymentMethod PaymentMethod
	// GuestAllowance is the maximum number of guests a single Registration
	// against this Game may bring (T8.6); 0 means no guests permitted.
	// Registration.GuestCount is validated against this field by Register.
	GuestAllowance int
}

// NewGame constructs a scheduled Game, validating the invariants that don't
// require knowledge of other Games or Bookings: Capacity must be positive,
// CourtIDs must be non-empty, the time range must be valid, PaymentMethod
// must be one of its closed enum values, and GuestAllowance must not be
// negative. It re-validates the range itself (rather than trusting the
// caller already went through NewTimeRange) so a Game built from a
// zero-duration or inverted range is rejected regardless of how the
// TimeRange value was produced.
//
// venueFacilityID (T8.3) is stored as-is, with no validation here: whether
// it refers to a real Facility requires calling out to the Facilities
// context, which this pure domain constructor cannot do (CLAUDE.md rule 2)
// — that existence check is app.Service.ScheduleGame's job, via
// port.FacilityLookup, before this Game is allowed to reserve any courts.
// An empty venueFacilityID is legal (see VenueFacilityID's doc comment).
func NewGame(id, hostID, facilityID, venueFacilityID string, courtIDs []string, r TimeRange, capacity int, paymentMethod PaymentMethod, guestAllowance int) (Game, error) {
	if capacity <= 0 {
		return Game{}, ErrInvalidCapacity
	}
	if len(courtIDs) == 0 {
		return Game{}, ErrEmptyCourtIDs
	}
	if !r.End.After(r.Start) {
		return Game{}, ErrInvalidTimeRange
	}
	if !paymentMethod.IsValid() {
		return Game{}, ErrInvalidPaymentMethod
	}
	if guestAllowance < 0 {
		return Game{}, ErrInvalidGuestAllowance
	}
	return Game{
		ID:              id,
		HostID:          hostID,
		FacilityID:      facilityID,
		VenueFacilityID: venueFacilityID,
		CourtIDs:        courtIDs,
		Range:           r,
		Capacity:        capacity,
		Status:          StatusScheduled,
		PaymentMethod:   paymentMethod,
		GuestAllowance:  guestAllowance,
	}, nil
}

// Cancel transitions a Game to cancelled. The only legal transition is
// scheduled -> cancelled; cancelling an already-cancelled Game is rejected
// rather than silently accepted, mirroring booking.Booking.Cancel().
//
// Cancelling a Game cascading to its Bookings/Registrations (freeing the
// reserved courts and cancelling player registrations) is explicitly out of
// scope for T5.1 — this method only flips the Game's own status. See the
// PR description's deferred-work note.
func (g *Game) Cancel() error {
	if g.Status != StatusScheduled {
		return ErrIllegalStatusTransition
	}
	g.Status = StatusCancelled
	return nil
}
