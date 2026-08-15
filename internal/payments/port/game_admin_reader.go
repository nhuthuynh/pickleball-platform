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
// service.go:1009 specifically so a second context could consume it later —
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
// # NOT wired into internal/payments/app.Service as of T15.5 — read this
// before wiring it in
//
// This port and internal/payments/adapter/socialplay.GameAdminReader (the
// implementation against it) are built and tested against Social Play's real
// app.Service (T15.5 instruction 1, §A12 GAP C), but are deliberately NOT
// constructor-wired into app.Service and NOT consulted by
// authorizeOfflineRecording. That is not an oversight: T15.5 instruction 2
// requires stopping and reporting a finding, rather than forcing a
// workaround, the moment consuming an already-exported read turns out to
// need more than the sprint plan's dependency-completeness check verified —
// and it does here.
//
// ListGameAdmins(ctx, gameID) needs a gameID. RecordOfflinePaymentInput and
// RefundPaymentInput never carry one for a PayableTypeRegistration or
// PayableTypeNoShowFee payable — only PayableID (the Registration's own id)
// and the caller-supplied GameHostID (a *subject*, not a Game id — and one
// Host can host many Games, so it would not resolve to a single Game even if
// it were looked up). Resolving "the Game this Registration belongs to" is a
// read no context exports today: socialplay/app.Service has no
// GetRegistration-shaped method, only write paths
// (RegisterForGame/CancelRegistration) and gameID-scoped reads
// (ListRegistrationsForGame, which additionally runs its own Host-or-admin
// authorization check — using it here to discover a Registration's GameID
// would require already knowing the answer this port exists to compute, a
// circularity, not a shortcut).
//
// Issue #149 already names this exact resolution step as the harder,
// still-open half of Payments' fact-fabrication gap: its "what closing it
// looks like" section says closing it means "RecordOfflinePayment looks up
// the Registration's Game's Host instead of being told it" — a read that
// does not exist yet, independent of whether the Game-Admin store does.
// #149 was open before T15.5 and stays open after it for exactly this
// reason (see the T15.5 PR description for the full writeup, and the
// follow-up issue this finding was filed as). Accepting a second,
// caller-supplied id here (e.g. a new game_id field, trusted without
// verification against the real Registration) would not close the gap — it
// would relocate it: a genuine Game Admin of some *other* Game could name
// that Game's id and pass this port's check for a Registration that is not
// theirs to administer, which is the same shape of forgery this ticket
// exists to close, one level up.
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
