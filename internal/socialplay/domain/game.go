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
	ID     string
	HostID string
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
	// EntryFee is the price the Host set for one player to join this Game
	// (T9.2). A zero EntryFee is legal and means a free Game — a real
	// value the Host chose, never a sentinel for "unset" (see
	// Money.IsFree). This field replaces T8.10's
	// PLACEHOLDER_REGISTRATION_FEE_CENTS, a flat $10.00 the web client
	// invented and labelled "placeholder" in the UI precisely because the
	// domain had no price field at all.
	EntryFee Money
}

// NewGame constructs a scheduled Game, validating the invariants that don't
// require knowledge of other Games or Bookings: Capacity must be positive,
// CourtIDs must be non-empty, the time range must be valid, PaymentMethod
// must be one of its closed enum values, GuestAllowance must not be
// negative, and entryFee must be well-formed (T9.2 — negative amounts and
// non-zero amounts with a missing/malformed currency code are rejected with
// ErrInvalidMoney; a zero amount is accepted and means a free Game, see
// Money.Validate). It re-validates the range itself (rather than trusting the
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
func NewGame(id, hostID, facilityID, venueFacilityID string, courtIDs []string, r TimeRange, capacity int, paymentMethod PaymentMethod, guestAllowance int, entryFee Money) (Game, error) {
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
	if err := entryFee.Validate(); err != nil {
		return Game{}, err
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
		EntryFee:        entryFee,
	}, nil
}

// EnsureHostOrGameAdmin returns ErrNotGameHostOrAdmin unless actorUserID
// matches g.HostID exactly, or holds a Game-Admin assignment in assigned — the
// object-level (BOLA) authorization check T10.4's RecordMatchResult requires
// and T14.5's roster read now shares, mirroring
// competitions.Competition.EnsureHost and facilities.Facility.EnsureOwner's
// shape (an empty actorUserID is always rejected, even against a Game with an
// empty HostID — an unidentified caller is never the host or an admin).
//
// # assigned is a resolved set, and its type is the enforcement
//
// Until T14.5 this parameter was `assignedGameAdminUserIDs []string`, taken
// **straight off the wire**: this codebase had no durable Game-Admin store, so
// the caller asserted who the admins were and this method believed them. Any
// caller could name themselves and pass. T14.4 built the store
// (port.GameAdminRepository, the game_admins table), and this signature is what
// makes using it non-optional: []GameAdmin is a type the app layer can only
// obtain by reading the repository, so no request DTO can be handed to this
// method however carelessly a future call site is written. Changing it back to
// []string would silently restore the forgeable check — the parameter type is
// load-bearing, not stylistic.
//
// The membership test itself is HasGameAdmin (game_admin.go), including its
// blank-entry guard, so the roster read and the match-recording path cannot
// drift apart by re-deriving it. Note this method takes the whole set rather
// than a pre-computed bool for the same reason: a caller that computed
// "isAdmin" itself would be holding the rule, and the rule belongs here.
//
// Keeping the domain pure (CLAUDE.md rule 2) is why the *resolution* — asking
// the repository for a Game's assignments — happens in app.Service and not
// here. This package imports nothing outside the standard library and gains no
// port import from this change.
//
// actorUserID is a verified subject as of T12.8 (the handler passes the bearer
// token's principal, never a request field), in the same identifier space
// g.HostID and GameAdmin.UserID hold — see domain.GameAdmin's doc comment and
// ADR-0014 §5a. Nothing is resolved or converted on either side.
func (g Game) EnsureHostOrGameAdmin(actorUserID string, assigned []GameAdmin) error {
	if actorUserID == "" {
		return ErrNotGameHostOrAdmin
	}
	if actorUserID == g.HostID {
		return nil
	}
	if HasGameAdmin(assigned, actorUserID) {
		return nil
	}
	return ErrNotGameHostOrAdmin
}

// EnsureHost returns ErrNotGameHost unless actorPlayerID matches g.HostID
// exactly (an empty actorPlayerID is always rejected, even against a Game
// with an empty HostID — an unidentified caller is never the host). This is
// the object-level (BOLA) authorization check T12.4's CancelGame requires,
// mirroring competitions.Competition.EnsureHost and
// facilities.Facility.EnsureOwner's shape.
//
// Deliberately a *separate* method from EnsureHostOrGameAdmin above,
// returning a separate sentinel, because the two gate different rules on
// this same aggregate: a match result may be recorded by the Host or an
// assigned Game Admin, but cancelling the Game is Host-only. See
// ErrNotGameHost's doc comment in errors.go for the full reasoning, and do
// not "simplify" by routing Cancel through EnsureHostOrGameAdmin with an
// empty admin list — that would make the difference an accident of the
// caller's arguments rather than a stated rule.
//
// As with every actor-scoped check in this codebase, actorPlayerID is a
// caller-supplied claim, not a verified identity: real authentication is
// HANDOFF.md's open Auth item (T12.8 this sprint), and this method must not
// be read as closing it.
func (g Game) EnsureHost(actorPlayerID string) error {
	if actorPlayerID == "" || actorPlayerID != g.HostID {
		return ErrNotGameHost
	}
	return nil
}

// EnsureNotCancelled returns ErrGameCancelled when g.Status is cancelled —
// T10.4's RecordMatchResult precondition ("recording a match against a
// cancelled Game -> FailedPrecondition"). Mirrors
// competitions.domain.Enter's identical "the Competition must still be
// scheduled" check (internal/competitions/domain/entry.go), lifted into its
// own method here rather than inlined into a constructor the way Enter does
// it: match.go's RecordMatch (T10.3) already shipped, merged, with a
// signature taking a bare gameID string — not a Game value — and its own
// doc comment explicitly assigns "whether gameID refers to a real,
// non-cancelled Game" to app.Service at wiring time (T10.4). Changing
// RecordMatch's signature now would relitigate a already-merged, deliberate
// T10.3 design choice for no real benefit; this method gives
// app.Service.RecordMatchResult the identical pure, reusable check without
// that churn.
func (g Game) EnsureNotCancelled() error {
	if g.Status == StatusCancelled {
		return ErrGameCancelled
	}
	return nil
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
