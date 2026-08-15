package port

import "context"

// GameAdminReader is Payments' inbound read port onto Social Play's durable
// Game-Admin store (T14.4's game_admins table, reached via
// socialplay/port.GameAdminRepository) — the read half of T15.5's §A12 GAP
// C. internal/payments/adapter/{socialplay,competitions} already push OUT to
// their respective contexts (RegistrationUpdater/EntryUpdater); this is the
// first port in this package that reads IN, and
// internal/payments/adapter/socialplay.GameAdminReader implements it against
// the real socialplayapp.Service.ListGameAdmins (exported at
// service.go:1089 specifically so a second context could consume it later —
// see that method's own doc comment and the sprint plan's §A12 GAP A/B).
//
// Returns bare subject strings, not socialplaydomain.GameAdmin values: this
// port answers exactly the membership question Payments' authorization would
// need (the shape domain.HasGameAdmin already tests against a []GameAdmin on
// the Social Play side) and nothing about AssignedBy/AssignedAt, which are
// Social Play's own audit facts, not an entitlement fact Payments consumes.
// Keeping the return type free of socialplaydomain also keeps this package
// (internal/payments/port, which otherwise imports only
// internal/payments/domain) from needing a second context's domain import —
// internal/payments/adapter/socialplay is where that import belongs, and
// already has it.
//
// # Wired into internal/payments/app.Service as of T16.2 — history below
//
// This port and internal/payments/adapter/socialplay.GameAdminReader (the
// implementation against it) were built and tested against Social Play's
// real app.Service in T15.5 (instruction 1, §A12 GAP C), but were
// deliberately left NOT constructor-wired into app.Service and NOT consulted
// by authorizeOfflineRecording — T15.5 instruction 2 required stopping and
// reporting a finding, rather than forcing a workaround, the moment
// consuming an already-exported read turned out to need more than that
// sprint plan's dependency-completeness check had verified, and it did:
// ListGameAdmins(ctx, gameID) needs a gameID, and neither
// RecordOfflinePaymentInput nor RefundPaymentInput carried one for a
// PayableTypeRegistration/PayableTypeNoShowFee payable — only PayableID (the
// Registration's own id) and the caller-supplied GameHostID (a *subject*,
// not a Game id, and one that would not have resolved to a single Game even
// if looked up, since one Host can host many Games).
//
// T16.2 closes exactly that gap: internal/payments/port.RegistrationLookup
// resolves a Registration's own id to its GameID
// (socialplayapp.Service.GetRegistrationByID), and this port then resolves
// that GameID to its current admin set — the two-step read issue #149's
// "what closing it looks like" section named as the harder, still-open half
// of Payments' fact-fabrication gap. Both steps are now wired into
// authorizeOfflineRecording via internal/payments/app.Service, for
// PayableTypeRegistration and PayableTypeNoShowFee payables. #149 stays open
// after T16.2 for one remaining fact only: PayableTypeBooking's
// BookingHostID, which is blocked on ADR-0015's still-open D1 (a Booking's
// Host is not resolvable the same way a Game's is until that decision
// lands), not on a missing read.
type GameAdminReader interface {
	// ListGameAdmins returns the user ids currently holding Game-Admin
	// authority over gameID, oldest assignment first — the order
	// socialplay/port.GameAdminRepository.ListGameAdmins documents; this
	// port is a thin read-through, not a new contract.
	//
	// An unknown or malformed gameID returns an empty slice, not an error —
	// deliberately more permissive than socialplayapp.Service.ListGameAdmins
	// itself (which answers domain.ErrGameNotFound for those cases, since it
	// is also the Assign/Revoke write path's existence check). A membership
	// question asked as an authorization pre-check has one safe answer for
	// both "no such Game" and "this Game has no admins": not authorized —
	// and this port's only intended caller is exactly that kind of
	// pre-check, never a response body an end user reads. See
	// internal/payments/adapter/socialplay.GameAdminReader.ListGameAdmins
	// for where that translation happens.
	ListGameAdmins(ctx context.Context, gameID string) ([]string, error)
}
