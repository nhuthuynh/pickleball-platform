package domain

// Status is a Competition's lifecycle state. Mirrors
// socialplay.Status/booking.Status's shape: a cancelled Competition no
// longer holds its slots (see Cancel's doc comment for the cascade that is
// a known, deliberate gap).
type Status string

const (
	StatusScheduled Status = "scheduled"
	StatusCancelled Status = "cancelled"
)

// Session is one sitting of a Competition: a time range and the courts it
// reserves for that range. A Competition runs across dates (spec Flow 5:
// "reserves courts across dates"), which is the one structural difference
// from a Game — a Game has a single Range and CourtIDs, a Competition has a
// list of (Range, CourtIDs) pairs.
//
// Sessions are a value type inside the Competition aggregate, not an
// aggregate of their own: they have no identity, no lifecycle, and are only
// ever meaningful as part of the Competition that owns them. T9.3 reserves
// one `competition`-source Booking per (session, court) pair through
// Competitions' own port.CourtReservation, inheriting the no-double-booking
// invariant with no change to Booking.
type Session struct {
	Range    TimeRange
	CourtIDs []string
}

// Competition is the aggregate a Host creates to run a tournament: a
// fixed-capacity event reserving courts across one or more sessions.
//
// Deliberately absent in T9: brackets, rounds, seeding, match scheduling
// and results (docs/process/t9-sprint-plan.md §A4). This ticket models the
// court-reservation and entry spine only — a complete user loop (create →
// advertise → enter → roster) without them. They are deferred to T10+, not
// dropped, and their absence is a scoping decision rather than an
// oversight.
type Competition struct {
	ID     string
	HostID string
	Name   string
	// VenueFacilityID is a real Facilities-context Facility ID, with the
	// same semantics and the same reasoning as
	// socialplay.Game.VenueFacilityID (T8.3): NewCompetition only stores
	// it, because checking that it refers to a real Facility requires the
	// Facilities context, which this pure domain package must never import
	// (CLAUDE.md rule 2/3). That existence check is
	// app.Service.ScheduleCompetition's job via port.FacilityLookup (T9.3),
	// before any court is reserved. Optional: an empty string means this
	// Competition has no known venue Facility yet, and the lookup is
	// skipped entirely in that case.
	VenueFacilityID string
	// Sessions is one or more sittings; never empty (NewCompetition
	// rejects that) and never self-overlapping on a shared court.
	Sessions []Session
	// Capacity is the total number of *places*, not entries: an entry and
	// each of its guests occupy one place each. See Enter for the weighted
	// rule this implies.
	Capacity int
	// GuestAllowance is the maximum number of guests a single entry may
	// bring; 0 means no guests permitted. CompetitionEntry.GuestCount is
	// validated against this field by Enter.
	GuestAllowance int
	PaymentMethod  PaymentMethod
	// EntryFee is what one entry costs. A zero amount is a real value
	// meaning a free competition, not "unset" — see Money.
	EntryFee Money
	// Format is descriptive only and enforces nothing — see format.go.
	Format Format
	Status Status
	// ShareToken is the opaque token behind a Competition's shareable
	// registration link (T9.5), stored as given. Generating it is
	// port.ShareTokenGenerator's job at the app layer (T9.3) — this
	// package has no business choosing a token format, and a pure domain
	// constructor must not reach for randomness. An empty token is legal
	// here so a Competition can be constructed in tests and in flows that
	// don't publish a link.
	ShareToken string
}

// NewCompetition constructs a scheduled Competition, validating every
// invariant that doesn't require knowledge of other Competitions or
// Bookings:
//
//   - Capacity must be positive;
//   - at least one Session is required (ErrEmptySessions), each with at
//     least one court (ErrEmptyCourtIDs) and a valid range
//     (ErrInvalidTimeRange). The range is re-validated here rather than
//     trusting the caller went through NewTimeRange, so a Competition built
//     from a zero-duration or inverted range is rejected regardless of how
//     the TimeRange value was produced — the same reasoning
//     socialplay.NewGame documents;
//   - no two sessions may reserve the same court at overlapping times
//     (ErrOverlappingSessions, see ensureNoSessionOverlap);
//   - PaymentMethod and Format must each be one of their closed enum
//     values;
//   - GuestAllowance must not be negative;
//   - EntryFee must be a well-formed Money (a zero fee is legal).
//
// venueFacilityID and shareToken are stored as-is, with no validation here
// — see their field doc comments for why each check belongs to the app
// layer instead.
func NewCompetition(
	id, hostID, name, venueFacilityID string,
	sessions []Session,
	capacity, guestAllowance int,
	paymentMethod PaymentMethod,
	entryFee Money,
	format Format,
	shareToken string,
) (Competition, error) {
	if capacity <= 0 {
		return Competition{}, ErrInvalidCapacity
	}
	if len(sessions) == 0 {
		return Competition{}, ErrEmptySessions
	}
	for _, s := range sessions {
		if len(s.CourtIDs) == 0 {
			return Competition{}, ErrEmptyCourtIDs
		}
		if !s.Range.End.After(s.Range.Start) {
			return Competition{}, ErrInvalidTimeRange
		}
	}
	if err := ensureNoSessionOverlap(sessions); err != nil {
		return Competition{}, err
	}
	if !paymentMethod.IsValid() {
		return Competition{}, ErrInvalidPaymentMethod
	}
	if !format.IsValid() {
		return Competition{}, ErrInvalidFormat
	}
	if guestAllowance < 0 {
		return Competition{}, ErrInvalidGuestAllowance
	}
	if err := entryFee.Validate(); err != nil {
		return Competition{}, err
	}

	return Competition{
		ID:              id,
		HostID:          hostID,
		Name:            name,
		VenueFacilityID: venueFacilityID,
		Sessions:        sessions,
		Capacity:        capacity,
		GuestAllowance:  guestAllowance,
		PaymentMethod:   paymentMethod,
		EntryFee:        entryFee,
		Format:          format,
		Status:          StatusScheduled,
		ShareToken:      shareToken,
	}, nil
}

// ensureNoSessionOverlap rejects a Competition that double-books one of its
// own courts: any two (session, court) reservations sharing a court whose
// half-open ranges overlap return ErrOverlappingSessions. Back-to-back
// sessions on one court are fine — [start, end) means one ending exactly
// when the next starts do not share an instant, the same rule
// booking.EnsureNoConflict applies.
//
// A court listed twice within a single session's own CourtIDs is caught by
// the same walk (a range always overlaps itself), which is correct: that
// would ask T9.3's reservation loop to book the same court twice for the
// same slot and hit the Postgres EXCLUDE constraint on the second attempt.
//
// The comparison is a plain O(n²) walk over (session, court) pairs. That is
// deliberate: a Competition has a handful of sessions and courts, this runs
// once at construction, and an index-based scheme would trade real clarity
// for imaginary speed.
func ensureNoSessionOverlap(sessions []Session) error {
	type reservation struct {
		courtID string
		r       TimeRange
	}
	var seen []reservation
	for _, s := range sessions {
		for _, courtID := range s.CourtIDs {
			for _, prev := range seen {
				if prev.courtID != courtID {
					continue
				}
				if prev.r.Overlaps(s.Range) {
					return ErrOverlappingSessions
				}
			}
			seen = append(seen, reservation{courtID: courtID, r: s.Range})
		}
	}
	return nil
}

// EnsureHost returns ErrNotCompetitionHost unless actorUserID matches
// c.HostID exactly (an empty actorUserID is always rejected, even against a
// Competition with an empty HostID — an unidentified caller is never the
// host). This is Competitions' object-level (BOLA) authorization check,
// mirroring facilities.Facility.EnsureOwner and
// socialplay.Registration.Cancel's actorPlayerID check applied to a
// Competition's own ownership fact.
//
// Built in T9.1, the same sprint as the write path that uses it (T9.3's
// CancelCompetition), rather than retrofitted afterwards — HANDOFF.md's
// Cross-cutting process decision, after four sprints in which authorization
// arrived late. As with every precedent here, actorUserID is a
// caller-supplied claim, not a verified identity: real authentication
// remains HANDOFF.md's open Auth item, and this method must not be read as
// closing it.
func (c Competition) EnsureHost(actorUserID string) error {
	if actorUserID == "" || actorUserID != c.HostID {
		return ErrNotCompetitionHost
	}
	return nil
}

// Cancel transitions a Competition to cancelled. The only legal transition
// is scheduled -> cancelled; cancelling an already-cancelled Competition is
// rejected rather than silently accepted, mirroring
// booking.Booking.Cancel()/socialplay.Game.Cancel().
//
// KNOWN GAP, stated so it doesn't look accidental: cancelling a Competition
// does NOT cascade to the Bookings its sessions reserved or to its entries.
// This method flips this aggregate's own status and nothing else, exactly
// as socialplay.Game.Cancel has since T5.1 — the cascade needs the app
// layer's ports (releasing each reserved court, cancelling each entry, and
// deciding what a cancelled entry means for a paid entry fee), and
// inventing half of it in a pure domain method would be worse than leaving
// it visibly undone. Entering a cancelled Competition IS already blocked
// (see Enter), so the gap is "courts stay reserved and existing entries
// keep their status", not "a cancelled Competition still takes entries".
func (c *Competition) Cancel() error {
	if c.Status != StatusScheduled {
		return ErrIllegalStatusTransition
	}
	c.Status = StatusCancelled
	return nil
}
