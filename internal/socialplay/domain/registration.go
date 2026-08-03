package domain

// RegistrationSource identifies how a Registration came to exist. "app" is
// a Player registering directly through the platform; "social" is reserved
// for a future social-invite flow (e.g. a friend registering on someone's
// behalf) and is not produced by anything in T5.2 — Register always
// produces RegistrationSourceApp today. Modelled now (per the glossary,
// agent-operating-handbook.md A2) so the field doesn't need a breaking
// change when that flow is built.
type RegistrationSource string

const (
	RegistrationSourceApp    RegistrationSource = "app"
	RegistrationSourceSocial RegistrationSource = "social"
)

// RegistrationStatus is a Registration's lifecycle state. Named distinctly
// from Game's Status (both live in this package) rather than reusing that
// type.
type RegistrationStatus string

const (
	RegistrationStatusRegistered RegistrationStatus = "registered"
	RegistrationStatusCancelled  RegistrationStatus = "cancelled"
)

// PaymentStatus tracks whether a Registration's spot has been paid for.
// T5 shipped this as modelling only ("a later app-layer method lets a Game
// Admin flip it once offline/Stripe payment recording exists"); T6.5 is
// that method (Service.MarkRegistrationPaymentStatus, called only through
// port.RegistrationPaymentUpdater by internal/payments/adapter/socialplay).
//
// PaymentStatusRefunded (T6.5) mirrors internal/payments/domain.Status's
// three-state unpaid -> paid -> refunded machine exactly, rather than
// collapsing "refunded" back into "unpaid": Registration.PaymentStatus is
// documented (docs/process/t6-sprint-plan.md T6.5) as becoming "a
// projection of" the real Payment aggregate, and a projection that can't
// represent one of its source's states loses real information a Game
// Admin needs — "this player never paid" and "this player paid, then got
// refunded" are different facts a support/admin view must be able to tell
// apart, even though Social Play itself never decides when either applies
// (Payments is the only writer, via UpdatePaymentStatus).
type PaymentStatus string

const (
	PaymentStatusUnpaid   PaymentStatus = "unpaid"
	PaymentStatusPaid     PaymentStatus = "paid"
	PaymentStatusRefunded PaymentStatus = "refunded"
)

// IsValid reports whether s is one of the PaymentStatus values Social Play
// recognises. MarkPaymentStatus uses this to keep the field a closed enum;
// mirrors internal/payments/domain.PayableType.IsValid's role.
func (s PaymentStatus) IsValid() bool {
	switch s {
	case PaymentStatusUnpaid, PaymentStatusPaid, PaymentStatusRefunded:
		return true
	default:
		return false
	}
}

// Registration is a Player's claim on one of a Game's capacity slots.
type Registration struct {
	ID            string
	GameID        string
	PlayerID      string
	Source        RegistrationSource
	Status        RegistrationStatus
	PaymentStatus PaymentStatus
}

// Register builds a new Registration for playerID against game, enforcing
// the two invariants that only need the Game and its other registrations
// (not infrastructure) to check:
//
//   - capacity: counts existing, non-cancelled registrations scoped to
//     game.ID (mirroring EnsureNoConflict's own-scope filtering, so a
//     caller can safely pass an unfiltered registrations slice); at
//     capacity, returns the exact, stable ErrGameFull sentinel.
//   - no double registration: a player with an existing active
//     registration for this game returns ErrAlreadyRegistered. This check
//     runs before the capacity check, since it's the more specific,
//     actionable error — a player who is already in should never be told
//     the game is full because of their own registration.
//
// The returned Registration's ID is intentionally left empty: Register is
// a pure domain constructor and, unlike NewGame/NewBooking, does not take
// an id parameter (see T5.2's ticket-specified signature) — assigning a
// durable ID is the app/adapter layer's job at persistence time.
func Register(game Game, existing []Registration, playerID string) (Registration, error) {
	if playerID == "" {
		return Registration{}, ErrEmptyPlayerID
	}

	activeCount, playerAlreadyActive := countActiveRegistrations(game.ID, existing, playerID)
	if playerAlreadyActive {
		return Registration{}, ErrAlreadyRegistered
	}
	if activeCount >= game.Capacity {
		return Registration{}, ErrGameFull
	}

	return Registration{
		GameID:        game.ID,
		PlayerID:      playerID,
		Source:        RegistrationSourceApp,
		Status:        RegistrationStatusRegistered,
		PaymentStatus: PaymentStatusUnpaid,
	}, nil
}

// countActiveRegistrations scans existing for non-cancelled registrations
// scoped to gameID, returning both the active count and whether playerID
// already holds one of them. Extracted (T6.6) from Register's own body so
// domain.JoinWaitlist can derive "is this Game actually full for this
// player" from the exact same counting rule Register uses, rather than a
// second, independently-maintained copy of it (CLAUDE.md rule 4's spirit:
// one counting rule, not two that can drift) — see JoinWaitlist's doc
// comment in waitlist.go.
func countActiveRegistrations(gameID string, existing []Registration, playerID string) (activeCount int, playerAlreadyActive bool) {
	for _, r := range existing {
		if r.GameID != gameID {
			continue
		}
		if r.Status == RegistrationStatusCancelled {
			continue
		}
		if r.PlayerID == playerID {
			playerAlreadyActive = true
		}
		activeCount++
	}
	return activeCount, playerAlreadyActive
}

// Cancel transitions a Registration to cancelled, but only for its owner.
// A mismatched actorPlayerID returns ErrNotRegistrationOwner rather than
// silently succeeding or falling through to ErrIllegalStatusTransition —
// this is the object-level (BOLA-shaped) check T5.2 requires: Player A
// must never be able to cancel Player B's registration. Ownership is
// checked before the status transition so a wrong actor gets a consistent
// answer regardless of the registration's current state, and the
// registration is left untouched on rejection.
//
// Once ownership is established, the only legal transition is
// registered -> cancelled; cancelling an already-cancelled registration is
// rejected (ErrIllegalStatusTransition) rather than silently accepted,
// mirroring booking.Booking.Cancel()/Game.Cancel().
func (r *Registration) Cancel(actorPlayerID string) error {
	if actorPlayerID != r.PlayerID {
		return ErrNotRegistrationOwner
	}
	if r.Status != RegistrationStatusRegistered {
		return ErrIllegalStatusTransition
	}
	r.Status = RegistrationStatusCancelled
	return nil
}

// MarkPaymentStatus sets PaymentStatus to a new value reported by the
// Payments context (T6.5), via port.RegistrationPaymentUpdater ->
// internal/payments/adapter/socialplay ->
// Service.MarkRegistrationPaymentStatus. This is the only path, besides
// Register's initial "unpaid" default, that may change PaymentStatus — see
// that method's doc comment for why the field must not have any other
// writer once T6.5 merges.
//
// Deliberately does NOT re-enforce Payment's own unpaid -> paid -> refunded
// transition legality (contrast with Cancel, which does own its
// transition): domain.Payment (internal/payments/domain) is the single
// source of truth for whether a transition is legal, and duplicating that
// state machine here would let the two contexts silently disagree about
// what's legal — if a future bug ever allowed Payments to report an
// impossible transition, that is a Payments-side bug to fix at the source,
// not something Social Play should also validate against. The only check
// here is that status is a recognised value at all (IsValid), so garbage
// input can never be written to the field.
func (r *Registration) MarkPaymentStatus(status PaymentStatus) error {
	if !status.IsValid() {
		return ErrInvalidPaymentStatus
	}
	r.PaymentStatus = status
	return nil
}
