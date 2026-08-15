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
	ErrInvalidTimeRange = errors.New("competitions: invalid time range")
	ErrInvalidCapacity  = errors.New("competitions: capacity must be greater than zero")
	ErrEmptySessions    = errors.New("competitions: at least one session is required")
	ErrEmptyCourtIDs    = errors.New("competitions: at least one court id is required per session")

	// ErrMalformedCourtID is ScheduleCompetition's shape guard on the
	// entries of every session's caller-supplied court_ids list (T14.8,
	// closing issue #156). Twin of
	// internal/socialplay/domain.ErrMalformedCourtID — see that sentinel's
	// doc comment for why this is InvalidArgument rather than the
	// not-found answer T10.7 gives a malformed Competition ID, and why it is
	// a distinct sentinel from ErrEmptyCourtIDs rather than a reuse of it.
	//
	// Kept context-local rather than shared with Social Play's copy for the
	// same reason uuidShape itself is duplicated per context (CLAUDE.md rule
	// 2 / the write-up on this package's uuidShape): a domain package does
	// not import another context's domain package.
	ErrMalformedCourtID      = errors.New("competitions: court id is not a valid court reference")
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
	// (see adapter/grpcapi.toStatus: ErrCompetitionFull -> AlreadyExists
	// per PR #87 review, matching Social Play's ErrGameFull precedent;
	// ErrGuestAllowanceExceeded -> InvalidArgument; ErrNotCompetitionHost
	// -> PermissionDenied; never Internal).
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

	// Port-level sentinels, added in T9.3 alongside the ports and adapters
	// that produce them — T9.1 deliberately left them undeclared rather
	// than shipping exported sentinels no code path could return (see the
	// note this block replaces).

	// ErrCompetitionNotFound is port.Repository's not-found answer for a
	// Competition ID that doesn't resolve. Mirrors
	// socialplay.ErrGameNotFound / booking.ErrBookingNotFound.
	ErrCompetitionNotFound = errors.New("competitions: competition not found")

	// ErrCourtUnavailable is this context's local translation of the
	// Booking context's ErrCourtDoubleBooked, produced by
	// internal/competitions/adapter/booking (CLAUDE.md rule 5). It means a
	// court a Competition's session needs is already held by an
	// overlapping Booking of *any* source — another competition, a game, a
	// recurring hire, or an individual booking. Competitions code above
	// the adapter never sees a bookingdomain error type, exactly as
	// socialplay.ErrCourtUnavailable does for Social Play.
	ErrCourtUnavailable = errors.New("competitions: court is unavailable for the requested time")

	// ErrFacilityNotFound is this context's local translation of the
	// Facilities context's own not-found error, produced by
	// internal/competitions/adapter/facilities. Returned by
	// app.Service.ScheduleCompetition when a non-empty VenueFacilityID
	// doesn't refer to a real Facility; T9.4 maps it to a 404-shaped
	// status.
	ErrFacilityNotFound = errors.New("competitions: facility not found")

	// ErrCompetitionEntryNotFound is port.Repository's not-found answer for
	// a CompetitionEntry ID that doesn't resolve (T10.6, GetEntryByID).
	// Mirrors socialplay.ErrRegistrationNotFound.
	ErrCompetitionEntryNotFound = errors.New("competitions: competition entry not found")

	// ErrInvalidPaymentStatus is returned by CompetitionEntry.
	// MarkPaymentStatus (T10.6) when asked to write a value outside
	// PaymentStatus's closed enum. Mirrors
	// socialplay.ErrInvalidPaymentStatus exactly — same role, same
	// boundary (the only thing checked is "is this a recognised value at
	// all"; the Payments context, not this one, owns whether the
	// unpaid -> paid -> refunded transition itself is legal, see
	// MarkPaymentStatus's doc comment).
	ErrInvalidPaymentStatus = errors.New("competitions: invalid payment status")

	// Competition-Admin sentinels (T15.3, partial fix for #168) — the exact
	// mirror of Social Play's Game-Admin set in
	// internal/socialplay/domain/errors.go.
	//
	// AssignCompetitionAdmin / EnsureMayRevokeCompetitionAdmin reuse
	// **ErrNotCompetitionHost** above for their authorization rejection rather
	// than declaring an admin-specific one, and that reuse is the point: the
	// rule they enforce IS "only the Host", identical to CancelCompetition's.
	// #168 requires an assigned admin to be refused exactly as a stranger is,
	// so giving the two callers different sentinels would invite a handler or
	// a future check to treat them differently.
	//
	//   - ErrEmptyCompetitionAdminUserID: AssignCompetitionAdmin was given a
	//     blank user id. A blank row would match a blank actor at read time,
	//     which is what HasCompetitionAdmin's blank-entry guard defends the
	//     read end against; rejecting it here closes the write end too.
	//     InvalidArgument at the handler.
	//   - ErrHostCannotBeCompetitionAdmin: the Host was named as an admin of
	//     their own Competition. The row would grant nothing (the Host is
	//     already entitled) while making a later revoke look like it stripped
	//     Host authority. InvalidArgument at the handler.
	//   - ErrAlreadyCompetitionAdmin: this user already holds an assignment
	//     for this Competition. Returned both by the domain's fail-fast
	//     pre-check and by the Postgres adapter translating the composite
	//     primary key's 23505, which is the authoritative guard under
	//     concurrency (CLAUDE.md rules 4 and 5). AlreadyExists at the handler.
	//   - ErrCompetitionAdminNotFound: RevokeCompetitionAdmin named a user
	//     holding no assignment. An error rather than a silent success — see
	//     port.CompetitionAdminRepository.Revoke's doc comment. NotFound at
	//     the handler.
	ErrEmptyCompetitionAdminUserID  = errors.New("competitions: competition admin user id is required")
	ErrHostCannotBeCompetitionAdmin = errors.New("competitions: a competition's host cannot also be one of its competition admins")
	ErrAlreadyCompetitionAdmin      = errors.New("competitions: user is already a competition admin for this competition")
	ErrCompetitionAdminNotFound     = errors.New("competitions: competition admin assignment not found")
)
