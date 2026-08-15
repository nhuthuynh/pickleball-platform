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

	// ErrInvalidMoney (T9.2) is returned by Money.Validate — and so by
	// NewGame for its entryFee argument — for a negative amount, or for a
	// non-zero amount whose currency code is missing or malformed. A ZERO
	// amount is not an error: it means a free Game, a real product state
	// (see Money.IsFree). One sentinel covers both the amount and the
	// currency half of the check because callers have the same recourse for
	// either ("that isn't a well-formed price, fix the input"), unlike
	// internal/payments/domain's split ErrInvalidAmount/ErrInvalidCurrency,
	// whose gRPC layer distinguishes the two.
	ErrInvalidMoney = errors.New("socialplay: invalid money amount")

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

	// T10.3 Match sentinels (docs/process/t10-sprint-plan.md T10.3). Match is
	// a Social Play fact — a recorded result against an existing Game — not
	// a PlayerRating computation; see match.go's own doc comment and
	// ADR-0012 (docs/adr/0012-identity-users-and-match-built-rating-and-
	// matching-algorithm-blocked-on-escalated-decisions.md) for why no
	// rating field or computation exists anywhere near this type this
	// sprint.
	//
	//   - ErrEmptyGameID: RecordMatch's gameID argument is empty.
	//   - ErrEmptyPlayers: RecordMatch's players argument is nil or empty —
	//     kept distinct from ErrTooFewPlayers (below) rather than folded
	//     into one generic error, mirroring T10.1/T10.3's own instruction
	//     that each named validation condition gets its own sentinel.
	//   - ErrTooFewPlayers: RecordMatch's players argument holds exactly one
	//     entry — a Match needs at least two participants to mean anything.
	//     Checked only once ErrEmptyPlayers has already ruled out zero, so
	//     each condition fires for exactly the input that describes it.
	ErrEmptyGameID   = errors.New("socialplay: game id is required")
	ErrEmptyPlayers  = errors.New("socialplay: at least one player is required")
	ErrTooFewPlayers = errors.New("socialplay: a match requires at least two players")

	// T10.4 Match-wiring sentinels (docs/process/t10-sprint-plan.md T10.4).
	//
	//   - ErrEmptyScore: RecordMatch's score argument is nil or empty.
	//     T10.3's original doc comment on Match.Score said RecordMatch
	//     "does not validate its contents beyond accepting whatever is
	//     given" — true of *which* players Score covers, but T10.4's own
	//     error-handling instructions require rejecting a wholesale empty
	//     score with InvalidArgument, so this ticket adds exactly that
	//     check (see RecordMatch's updated doc comment) without going back
	//     on T10.3's "doesn't have to cover every player" decision.
	//   - ErrNotGameHostOrAdmin: Game.EnsureHostOrGameAdmin's rejection —
	//     the actor recording a Match result is neither the Game's Host
	//     nor one of its caller-supplied assigned Game Admins. Mirrors
	//     competitions.ErrNotCompetitionHost/facilities.ErrNotFacilityOwner's
	//     flat, single-sentinel shape (a caller does not get to distinguish
	//     "wrong actor" from "not an admin either" from this error alone).
	//   - ErrGameCancelled: app.Service.RecordMatchResult's precondition
	//     check — a cancelled Game can no longer have match results
	//     recorded against it. Named distinctly from
	//     ErrIllegalStatusTransition (Game.Cancel's own transition guard)
	//     because this is a precondition on a *different* operation acting
	//     on an already-cancelled Game, the same relationship
	//     competitions.ErrCompetitionCancelled has with Competition's own
	//     status field (internal/competitions/domain/entry.go's Enter).
	ErrEmptyScore         = errors.New("socialplay: match score is required")
	ErrNotGameHostOrAdmin = errors.New("socialplay: only the game's host or an assigned game admin may record a match result")
	ErrGameCancelled      = errors.New("socialplay: game is cancelled")

	// ErrNotGameHost is Game.EnsureHost's rejection (T12.4): the actor
	// attempting a Host-only action on a Game is not that Game's Host.
	// Mirrors competitions.ErrNotCompetitionHost/facilities.ErrNotFacilityOwner's
	// flat, single-sentinel shape.
	//
	// T13.6 widened the message from "may cancel the game" to the general
	// form, because CancelGame is no longer the only Host-only rule: the
	// roster read (ListRegistrationsForGame, partial fix for #147) is the
	// second, and it is a *read* rather than a destructive write. The
	// sentinel and its PermissionDenied mapping are unchanged — only the
	// human-readable text, which had narrowed to its first caller. Both
	// callers use Game.EnsureHost, so the permitted-actor set they encode is
	// identical and one sentinel is correct for both.
	//
	// DO NOT UNIFY THIS WITH ErrNotGameHostOrAdmin ABOVE. The two sentinels
	// look near-identical and gate near-identical-looking checks, but they
	// encode two genuinely different authorization rules on the same
	// aggregate, and collapsing them would silently widen one of them:
	//
	//   - ErrNotGameHostOrAdmin gates RecordMatchResult (T10.4), which the
	//     Host *or* any assigned Game Admin may perform. Recording a score
	//     is routine scorekeeping — CLAUDE.md's locked decision that
	//     "per-game Game Admins can record offline payments" is the same
	//     delegation shape.
	//   - ErrNotGameHost gates CancelGame (T12.4) and the roster read
	//     (T13.6), both of which are **Host-only**. Cancelling is destructive
	//     and irreversible (Game.Cancel permits no transition back out of
	//     cancelled), so it is deliberately NOT delegated to Game Admins. A
	//     Game Admin who may legitimately record a match result must still be
	//     refused here — see domain.TestGame_EnsureHost_GameAdminIsRejected,
	//     which exists precisely to fail if the two are ever merged. The
	//     roster read lands on this stricter sentinel for a different reason:
	//     a Game Admin arguably *should* be entitled to read a roster, but
	//     the assigned-admin list is caller-supplied and persisted nowhere
	//     (#168), so honouring it would let any caller name themselves an
	//     admin. Host-only is the narrower, honest answer until that store
	//     exists — recorded here because it is a product limitation, not a
	//     domain-modelling preference.
	//
	// Both map to codes.PermissionDenied at the gRPC boundary; that shared
	// mapping is not a reason to share a sentinel, since the callers'
	// permitted-actor sets differ.
	ErrNotGameHost = errors.New("socialplay: only the game's host may perform this action on this game")
)
