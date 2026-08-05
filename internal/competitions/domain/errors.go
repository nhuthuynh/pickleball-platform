package domain

import "errors"

// Domain errors. Adapters must translate infrastructure failures into these
// — upper layers only ever see errors from this file. Mirrors
// internal/socialplay/domain/errors.go's sentinel-error pattern, and
// deliberately does not share symbols with it: each bounded context owns
// its own errors, so internal/competitions/domain never needs to import
// internal/socialplay/domain or internal/booking/domain (CLAUDE.md rule 2).
// Every message is prefixed "competitions:" so a wrapped error read in a
// log says which context produced it.
var (
	// Construction-time validation (NewCompetition).
	ErrInvalidTimeRange      = errors.New("competitions: invalid time range")
	ErrInvalidCapacity       = errors.New("competitions: capacity must be greater than zero")
	ErrEmptySessions         = errors.New("competitions: at least one session is required")
	ErrEmptyCourtIDs         = errors.New("competitions: at least one court id is required per session")
	ErrInvalidPaymentMethod  = errors.New("competitions: invalid payment method")
	ErrInvalidFormat         = errors.New("competitions: invalid format")
	ErrInvalidGuestAllowance = errors.New("competitions: guest allowance must not be negative")

	// ErrInvalidMoney rejects a Money value that couples an amount with no
	// currency (or a malformed currency code) — ADR-0005's "amount and
	// currency are coupled, never a bare cents value" applied to
	// Competition.EntryFee. A *zero* amount with no currency is explicitly
	// legal: that is a free competition, a real value rather than a
	// placeholder (see Money's doc comment).
	ErrInvalidMoney = errors.New("competitions: invalid money value")

	// ErrOverlappingSessions rejects a Competition whose own sessions
	// double-book one of its courts (T9.1). Caught at construction rather
	// than left to the Booking-level invariant, which would only notice
	// mid-way through T9.3's per-(session, court) reservation loop and
	// force a rollback for a mistake that was detectable up front. Ranges
	// are half-open [start, end), so back-to-back sessions on one court
	// are not an overlap — same rule as Booking.
	ErrOverlappingSessions = errors.New("competitions: sessions overlap on the same court")

	// ErrIllegalStatusTransition guards Competition.Cancel — cancelling an
	// already-cancelled Competition is rejected, not silently accepted
	// (mirrors booking.Booking.Cancel / socialplay.Game.Cancel).
	ErrIllegalStatusTransition = errors.New("competitions: illegal status transition")

	// Entry-time validation and rejections (Enter). Each is a distinct,
	// stable sentinel rather than a generic validation error because
	// T9.4's gRPC/REST layer maps each to its own status code
	// (docs/process/t9-sprint-plan.md T9.4: ErrCompetitionFull ->
	// FailedPrecondition, ErrGuestAllowanceExceeded -> InvalidArgument,
	// ErrNotCompetitionHost -> PermissionDenied, never Internal).
	ErrEmptyPlayerID          = errors.New("competitions: player id is required")
	ErrInvalidEntrySource     = errors.New("competitions: invalid entry source")
	ErrGuestAllowanceExceeded = errors.New("competitions: guest count exceeds this competition's guest allowance")
	ErrCompetitionFull        = errors.New("competitions: competition is at capacity")
	ErrAlreadyEntered         = errors.New("competitions: player has already entered this competition")
	ErrCompetitionCancelled   = errors.New("competitions: competition is cancelled")

	// ErrNotCompetitionHost is Competitions' object-level (BOLA-shaped)
	// authorization rejection, mirroring
	// facilities.Facility.EnsureOwner's ErrNotFacilityOwner,
	// socialplay.ErrNotRegistrationOwner, and
	// payments.ErrNotPaymentRecorder. Introduced in T9.1 — the same sprint
	// as the write path that consumes it (T9.3's CancelCompetition) — per
	// HANDOFF.md's Cross-cutting process decision, after four sprints in
	// which authorization arrived as an afterthought. As with those
	// precedents, actorUserID is a caller-supplied claim, not a verified
	// identity: real authentication is HANDOFF.md's open Auth item and
	// this sentinel must not be read as closing it.
	ErrNotCompetitionHost = errors.New("competitions: only the host may perform this action on this competition")
)

// Deliberately NOT declared here yet: the port-level sentinels this context
// will need once it has ports and adapters (a not-found error per
// aggregate; a context-local ErrCourtUnavailable translating
// bookingdomain.ErrCourtDoubleBooked; a context-local ErrFacilityNotFound
// translating the Facilities context's own — both mirroring
// internal/socialplay/domain/errors.go). They belong to T9.3, which
// introduces the code that produces and translates them; declaring them
// here would ship exported sentinels no code path can return and no test
// can exercise.
