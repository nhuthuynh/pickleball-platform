package port

import (
	"context"

	"github.com/nhuthuynh/white-label/internal/payments/domain"
)

// This file holds Payments' inbound read ports for PRICE — what a payable
// actually owes — one per context that owns a payable type.
//
// They are separate interfaces rather than one multi-method port, matching
// the shape this package already uses for the ownership reads
// (RegistrationLookup over Social Play, EntryLookup over Competitions): an
// implementation lives in an adapter over exactly one context's
// app.Service, and a single interface spanning both could not be
// implemented by either of them.
//
// RegistrationAmountLookup is Payments' inbound read port resolving a
// Registration to the amount it actually owes (T56.2, issue #297).
//
// It was called PayableAmountLookup when T56.2 introduced it, and was
// renamed in T57.2 when the Competitions half arrived: a name promising to
// resolve any *payable* while only answering for Registrations is exactly
// the kind of overpromise that gets a reader wondering why entries are
// missing. One port per context, each named for what it answers.
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
type RegistrationAmountLookup interface {
	// ExpectedAmountForRegistration returns what the Registration
	// identified by registrationID owes.
	//
	// Implementations return an error satisfying
	// errors.Is(err, domain.ErrPayableNotFound) when registrationID
	// resolves to no Registration. Callers must not depend on any other
	// error type crossing this boundary for that case.
	ExpectedAmountForRegistration(ctx context.Context, registrationID string) (domain.Money, error)
}

// EntryAmountLookup is RegistrationAmountLookup's Competitions mirror
// (T57.2): it resolves a CompetitionEntry to what it owes.
//
// # Why a second port rather than a second method
//
// See this file's header. The two are implemented in different adapter
// packages over different app.Services; merging them would force one type
// to depend on both contexts.
//
// # Why this exists at all, a sprint after the Registration half
//
// T56.2 validated online payments for Registrations and named
// CompetitionEntry as explicitly out of scope. That left the two payable
// types with the same shape and opposite behaviour — a $50 Competition
// entry could still be paid for with one cent while an equivalent Game
// registration could not — with nothing but a scope note explaining the
// difference. A validation gap that exists because a ticket ended is not a
// design; this closes it.
type EntryAmountLookup interface {
	// ExpectedAmountForCompetitionEntry returns what the CompetitionEntry
	// identified by entryID owes: the Competition's EntryFee once per head,
	// frozen when the entrant entered (T57.1).
	//
	// Implementations return an error satisfying
	// errors.Is(err, domain.ErrPayableNotFound) when entryID resolves to no
	// entry — the SAME sentinel RegistrationAmountLookup uses for its own
	// miss, because "the payable this payment names does not exist" is one
	// fact with one answer, whichever context owns the payable (CLAUDE.md
	// rule 7). Callers must not depend on any other error type crossing
	// this boundary for that case.
	ExpectedAmountForCompetitionEntry(ctx context.Context, entryID string) (domain.Money, error)
}
