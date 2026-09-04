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
	ID      string
	CourtID string
	Source  Source
	Status  Status
	Range   TimeRange
	// OwnerUserID is the User.ID (uuid — never an IdP subject, ADR-0014) of
	// whoever this Booking belongs to. It is required: DECISION D1
	// (ADR-0015, option (a)) settled that every Booking is owned by a
	// verified user, so there is no such thing as an anonymous Booking any
	// more. Which user it is depends on the Source — the booker for
	// `individual`, the Game's host for `game`, the Competition's host for
	// `competition`, the template's requester for `recurring_hire` — but
	// there is always exactly one, and EnsureOwner below is what it is for.
	//
	// Mirrored in Postgres as bookings.owner_user_id, declared
	// `uuid NOT NULL REFERENCES identity_users (id)` (migration 0027),
	// following RecurringHireTemplate.RequestedByUserID's established shape.
	OwnerUserID string
	ReferenceID string
}

// NewBooking constructs a confirmed Booking, validating the invariants that
// don't require knowledge of other bookings (time-range validity and source
// validity). Conflict checking against existing bookings is a separate step
// (EnsureNoConflict) because it needs the court's other bookings, which the
// domain doesn't hold — that's the app layer's job via the repository port.
func NewBooking(id, courtID string, source Source, r TimeRange, referenceID, ownerUserID string) (Booking, error) {
	if courtID == "" {
		return Booking{}, ErrEmptyCourtID
	}
	if !source.IsValid() {
		return Booking{}, ErrInvalidSource
	}
	// Rejected here rather than left to the FK, for the same reason
	// NewRecurringHireTemplate rejects an empty RequestedByUserID: an
	// unowned Booking is precisely the hole #144 reported, and the domain
	// should refuse to construct one even in a test or an in-memory
	// repository that never reaches Postgres.
	if ownerUserID == "" {
		return Booking{}, ErrEmptyOwnerUserID
	}
	return Booking{
		ID:          id,
		CourtID:     courtID,
		Source:      source,
		Status:      StatusConfirmed,
		Range:       r,
		OwnerUserID: ownerUserID,
		ReferenceID: referenceID,
	}, nil
}

// EnsureOwner returns ErrNotBookingOwner unless actorUserID is exactly this
// Booking's owner. Both sides are User.IDs (uuids), never IdP subjects —
// ADR-0014's seam translates the subject once, at the grpcapi boundary.
//
// An empty actorUserID never matches, even against a Booking whose
// OwnerUserID is somehow also empty (a hand-built zero value; NewBooking
// cannot produce one). That guard is the same one Game.EnsureHost and
// Facility.EnsureOwner carry, and it is load-bearing rather than defensive:
// without it an unauthenticated caller and an unowned row would authorize
// each other, which is the exact shape of the bug D1 was raised to close.
func (b *Booking) EnsureOwner(actorUserID string) error {
	if actorUserID == "" || actorUserID != b.OwnerUserID {
		return ErrNotBookingOwner
	}
	return nil
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
