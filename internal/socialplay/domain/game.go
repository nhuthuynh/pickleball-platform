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
	FacilityID string
	CourtIDs   []string
	Range      TimeRange
	Capacity   int
	Status     Status
}

// NewGame constructs a scheduled Game, validating the invariants that don't
// require knowledge of other Games or Bookings: Capacity must be positive,
// CourtIDs must be non-empty, and the time range must be valid. It
// re-validates the range itself (rather than trusting the caller already
// went through NewTimeRange) so a Game built from a zero-duration or
// inverted range is rejected regardless of how the TimeRange value was
// produced.
func NewGame(id, hostID, facilityID string, courtIDs []string, r TimeRange, capacity int) (Game, error) {
	if capacity <= 0 {
		return Game{}, ErrInvalidCapacity
	}
	if len(courtIDs) == 0 {
		return Game{}, ErrEmptyCourtIDs
	}
	if !r.End.After(r.Start) {
		return Game{}, ErrInvalidTimeRange
	}
	return Game{
		ID:         id,
		HostID:     hostID,
		FacilityID: facilityID,
		CourtIDs:   courtIDs,
		Range:      r,
		Capacity:   capacity,
		Status:     StatusScheduled,
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
