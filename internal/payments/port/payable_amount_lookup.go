package port

import (
	"context"

	"github.com/nhuthuynh/white-label/internal/payments/domain"
)

// PayableAmountLookup is Payments' inbound read port resolving a payable to
// the amount it actually owes (T56.2, issue #297).
//
// # Why this exists
//
// Before it, the amount on `CreateOnlinePayment` was **caller-supplied and
// never checked against anything**. Payments had read-side ports for
// *ownership* facts — `RegistrationLookup` (Registration -> Game) and
// `GameLookup` (Game -> Host) — but nothing exposed a price, so a $25 Game
// could be paid for with one cent and the Registration would then be marked
// paid in full. Harmless only because the processor is stubbed; a live
// revenue hole the day a real one is wired.
//
// This is the same family of defect as #149 (Payments trusting a fact the
// caller asserts) but a different fact, with a different fix: #149 is about
// *who owns what*, this is about *what it costs*.
//
// # What it returns, and why that is the frozen figure
//
// The expected amount is the Registration's own stored `AmountOwed`, frozen
// when the player registered (T56.1, issue #126) — NOT a live recomputation
// from the Game's current EntryFee. That distinction is the whole value of
// the check: validating against a number the Host can change afterwards
// would mean a player's correct payment becomes "wrong" the moment the
// price moves. What is validated is what was agreed.
//
// # Shape
//
// Mirrors RegistrationLookup exactly — a primitive-ish inbound port,
// implemented in `internal/payments/adapter/socialplay` against Social
// Play's real app.Service, returning Payments' OWN domain.Money rather than
// Social Play's, so no socialplaydomain type crosses the boundary
// (CLAUDE.md rules 2/3/5).
type PayableAmountLookup interface {
	// ExpectedAmountForRegistration returns what the Registration
	// identified by registrationID owes.
	//
	// Implementations return an error satisfying
	// errors.Is(err, domain.ErrPayableNotFound) when registrationID
	// resolves to no Registration. Callers must not depend on any other
	// error type crossing this boundary for that case.
	ExpectedAmountForRegistration(ctx context.Context, registrationID string) (domain.Money, error)
}
