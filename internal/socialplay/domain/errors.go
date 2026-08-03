package domain

import "errors"

// Domain errors. Adapters must translate infrastructure failures into these
// — upper layers only ever see errors from this file. Mirrors
// internal/booking/domain/errors.go's sentinel-error pattern; deliberately
// not shared with that package (CLAUDE.md: socialplay/domain must not
// import internal/booking/domain).
var (
	ErrInvalidTimeRange        = errors.New("socialplay: invalid time range")
	ErrInvalidCapacity         = errors.New("socialplay: capacity must be greater than zero")
	ErrEmptyCourtIDs           = errors.New("socialplay: at least one court id is required")
	ErrIllegalStatusTransition = errors.New("socialplay: illegal status transition")
	ErrEmptyPlayerID           = errors.New("socialplay: player id is required")

	// ErrGameFull, ErrAlreadyRegistered, and ErrNotRegistrationOwner are
	// deliberately distinct, stable sentinels (not a generic validation
	// error) — T5.4's gRPC/REST layer maps each to its own status code, and
	// T6's waitlist promotion trigger needs a clean hook on ErrGameFull
	// specifically (see docs/process/t5-sprint-plan.md T5.2 kickoff note).
	ErrGameFull             = errors.New("socialplay: game is at capacity")
	ErrAlreadyRegistered    = errors.New("socialplay: player is already registered for this game")
	ErrNotRegistrationOwner = errors.New("socialplay: only the registering player may cancel this registration")

	// ErrGameNotFound and ErrRegistrationNotFound are returned by
	// port.GameRepository/port.RegistrationRepository implementations
	// (T5.4) when the requested aggregate doesn't exist — mirrors
	// bookingdomain.ErrBookingNotFound's role.
	ErrGameNotFound         = errors.New("socialplay: game not found")
	ErrRegistrationNotFound = errors.New("socialplay: registration not found")

	// ErrCourtUnavailable is Social Play's own, context-local conflict
	// error (T5.3) — the translated equivalent of
	// bookingdomain.ErrCourtDoubleBooked, but never that type itself, so
	// internal/socialplay/domain and internal/socialplay/app never need to
	// import internal/booking/domain (CLAUDE.md rule 2 / T5 sprint plan
	// kickoff note). port.CourtReservation implementations (the T5.4
	// adapter, or an in-memory test fake) return an error satisfying
	// errors.Is(err, ErrCourtUnavailable) when the requested court/time
	// already has a confirmed Booking of any source.
	ErrCourtUnavailable = errors.New("socialplay: court unavailable for the requested time range")

	// ErrInvalidPaymentStatus is returned by Registration.MarkPaymentStatus
	// (T6.5) for a status value outside PaymentStatus's closed enum — the
	// domain-side guard keeping Registration.PaymentStatus a projection of
	// internal/payments/domain.Status's own values, never arbitrary input.
	ErrInvalidPaymentStatus = errors.New("socialplay: invalid payment status")

	// T6.6 waitlist sentinels. ErrGameNotFull and ErrAlreadyOnWaitlist are
	// JoinWaitlist's own rejections, deliberately distinct from
	// ErrGameFull/ErrAlreadyRegistered (which JoinWaitlist also propagates
	// unchanged in the cases they apply — see JoinWaitlist's doc comment).
	//
	//   - ErrGameNotFull: a player tried to join the waitlist for a Game that
	//     still has an open registration slot — they should register
	//     directly instead.
	//   - ErrAlreadyOnWaitlist: a player with an existing waiting/promoted
	//     waitlist entry for this Game tried to join a second time.
	ErrGameNotFull           = errors.New("socialplay: game is not full, register directly instead of joining the waitlist")
	ErrAlreadyOnWaitlist     = errors.New("socialplay: player is already on this game's waitlist")
	ErrWaitlistEntryNotFound = errors.New("socialplay: waitlist entry not found")

	// ErrNoWaitingEntries is returned by
	// port.WaitlistRepository.PromoteNext when a Game currently has no
	// entry in "waiting" status to promote — an expected, non-error outcome
	// for app.Service (a cancellation on a Game with an empty waitlist is
	// not a failure), kept as a sentinel rather than a (WaitlistEntry, bool,
	// error) return so the interface stays symmetric with the rest of this
	// package's repositories.
	ErrNoWaitingEntries = errors.New("socialplay: no waiting waitlist entries for this game")

	// ErrWaitlistPromotionNotExpired guards ExpireWaitlistPromotion
	// (app.Service): calling it on a promoted entry whose response window
	// (WaitlistEntry.PromotionResponseWindow) hasn't actually elapsed yet is
	// a caller error, not a silent no-op — mirrors this package's existing
	// "reject, don't guess" stance on illegal/premature transitions.
	ErrWaitlistPromotionNotExpired = errors.New("socialplay: waitlist promotion response window has not expired yet")

	// ErrNotWaitlistEntryOwner is WaitlistEntry.Cancel's BOLA-shaped
	// rejection, kept distinct from ErrNotRegistrationOwner even though the
	// shape is identical (CLAUDE.md rule 7 — a WaitlistEntry is not a
	// Registration, so its own-scope check gets its own sentinel).
	ErrNotWaitlistEntryOwner = errors.New("socialplay: only the waiting player may cancel this waitlist entry")

	// T8.6 sentinels: Game.PaymentMethod, Game.GuestAllowance, and
	// Registration.GuestCount (docs/process/t8-sprint-plan.md T8.6).
	//
	//   - ErrInvalidPaymentMethod: NewGame's PaymentMethod argument is not one
	//     of PaymentMethod's closed enum values (online/cash/either) — mirrors
	//     PaymentStatus.IsValid's "reject garbage input" role, but for a
	//     Game-level field, deliberately distinct from
	//     ErrInvalidPaymentStatus (Registration-level; see PaymentMethod's own
	//     doc comment for why the two must not be conflated).
	//   - ErrInvalidGuestAllowance: NewGame's GuestAllowance argument is
	//     negative.
	//   - ErrGuestAllowanceExceeded: Register's guestCount argument is
	//     negative, or exceeds game.GuestAllowance.
	ErrInvalidPaymentMethod   = errors.New("socialplay: invalid payment method")
	ErrInvalidGuestAllowance  = errors.New("socialplay: guest allowance must not be negative")
	ErrGuestAllowanceExceeded = errors.New("socialplay: guest count exceeds this game's guest allowance")

	// ErrFacilityNotFound is Social Play's own, context-local sentinel
	// (T8.3) for a Game.VenueFacilityID that doesn't refer to any real
	// Facility — the translated equivalent of
	// facilitiesdomain.ErrFacilityNotFound, but never that type itself, so
	// internal/socialplay/domain and internal/socialplay/app never need to
	// import internal/facilities/domain (CLAUDE.md rule 2/3, mirrors
	// ErrCourtUnavailable's relationship with bookingdomain.
	// ErrCourtDoubleBooked). A distinct symbol from facilities' own
	// same-named sentinel is intentional and safe: each bounded context
	// owns its own errors (this file's own top-of-block comment) and the
	// two are never compared against each other — only
	// internal/socialplay/adapter/facilities ever sees both, and its job
	// is precisely to translate one into the other, never let
	// facilitiesdomain.ErrFacilityNotFound cross this boundary directly.
	// app.Service.ScheduleGame returns this (via port.FacilityLookup) when
	// VenueFacilityID is non-empty but unknown; grpcapi.toStatus maps it to
	// codes.NotFound (404), not a 500 or a silent accept.
	ErrFacilityNotFound = errors.New("socialplay: venue facility not found")
)
