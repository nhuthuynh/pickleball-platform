package domain

// Source identifies which of the four reservation kinds produced a Booking.
// D3b (locked): every court reservation — however it originates — becomes a
// Booking with one of these sources, so the no-double-booking invariant
// covers all of them. There is no separate reservation concept per source.
type Source string

const (
	SourceRecurringHire Source = "recurring_hire"
	SourceIndividual    Source = "individual"
	SourceGame          Source = "game"
	SourceCompetition   Source = "competition"
)

func (s Source) IsValid() bool {
	switch s {
	case SourceRecurringHire, SourceIndividual, SourceGame, SourceCompetition:
		return true
	default:
		return false
	}
}

// Status is a Booking's lifecycle state. A cancelled Booking no longer
// occupies its court — the EXCLUDE constraint mirrors this by scoping to
// `status <> 'cancelled'`, and EnsureNoConflict does the same.
type Status string

const (
	StatusConfirmed Status = "confirmed"
	StatusCancelled Status = "cancelled"
)

// ReferenceID optionally links a Booking to the Game, Competition, or Club
// recurring-hire template it belongs to. Empty for a plain individual booking.
type Booking struct {
	ID          string
	CourtID     string
	Source      Source
	Status      Status
	Range       TimeRange
	ReferenceID string
}

// NewBooking constructs a confirmed Booking, validating the invariants that
// don't require knowledge of other bookings (time-range validity and source
// validity). Conflict checking against existing bookings is a separate step
// (EnsureNoConflict) because it needs the court's other bookings, which the
// domain doesn't hold — that's the app layer's job via the repository port.
func NewBooking(id, courtID string, source Source, r TimeRange, referenceID string) (Booking, error) {
	if courtID == "" {
		return Booking{}, ErrEmptyCourtID
	}
	if !source.IsValid() {
		return Booking{}, ErrInvalidSource
	}
	return Booking{
		ID:          id,
		CourtID:     courtID,
		Source:      source,
		Status:      StatusConfirmed,
		Range:       r,
		ReferenceID: referenceID,
	}, nil
}

// Cancel transitions a Booking to cancelled. The only legal transition is
// confirmed -> cancelled; cancelling an already-cancelled booking is
// rejected rather than silently accepted, so double-cancel bugs surface.
func (b *Booking) Cancel() error {
	if b.Status != StatusConfirmed {
		return ErrIllegalStatusTransition
	}
	b.Status = StatusCancelled
	return nil
}

// EnsureNoConflict is the domain-side mirror of the Postgres EXCLUDE
// constraint: candidate must not overlap any non-cancelled existing booking
// on the same court, regardless of source (game vs competition vs individual
// vs recurring hire all count). It's a fast pre-check for tests and for
// giving a clear error before hitting the DB — the DB constraint remains
// authoritative under concurrency (see T4).
func EnsureNoConflict(candidate Booking, existing []Booking) error {
	for _, other := range existing {
		if other.CourtID != candidate.CourtID {
			continue
		}
		if other.Status == StatusCancelled {
			continue
		}
		if other.ID == candidate.ID {
			continue
		}
		if candidate.Range.Overlaps(other.Range) {
			return ErrCourtDoubleBooked
		}
	}
	return nil
}
