package competitions_test

import (
	"context"
	"errors"
	"testing"

	competitionsdomain "github.com/nhuthuynh/white-label/internal/competitions/domain"
	paymentscompetitions "github.com/nhuthuynh/white-label/internal/payments/adapter/competitions"
	paymentsdomain "github.com/nhuthuynh/white-label/internal/payments/domain"
	paymentsport "github.com/nhuthuynh/white-label/internal/payments/port"
)

// T57.2 — the Competitions side of the amount check, proven against the
// REAL competitionsapp.Service rather than a fake of it, per T14.8/T15.5's
// cross-context-fake warning and the precedent entry_lookup_test.go sets in
// this same package.

// Compile-time proof the adapter satisfies the port.
var _ paymentsport.EntryAmountLookup = (*paymentscompetitions.EntryAmountLookup)(nil)

const (
	amountLookupEntryID       = "6ba7b810-0000-4000-8000-0000000000fa"
	amountLookupCompetitionID = "6ba7b810-0000-4000-8000-0000000000fb"
)

// The happy path, and the field mapping that is easy to get silently wrong:
// Competitions' Money names its fields AmountCents/CurrencyCode and
// Payments' uses Cents/Currency, so the translation is hand-written and a
// swapped or dropped field would compile.
func TestExpectedAmountForCompetitionEntry_ReturnsTheStoredAmount(t *testing.T) {
	t.Parallel()

	repo := newEntryLookupFakeRepository(competitionsdomain.CompetitionEntry{
		ID:            amountLookupEntryID,
		CompetitionID: amountLookupCompetitionID,
		PlayerID:      "player-1",
		GuestCount:    2,
		Status:        competitionsdomain.EntryStatusEntered,
		PaymentStatus: competitionsdomain.PaymentStatusUnpaid,
		AmountOwed:    competitionsdomain.Money{AmountCents: 6000, CurrencyCode: "GBP"},
	})
	lookup := paymentscompetitions.NewEntryAmountLookup(newEntryLookupTestService(repo))

	owed, err := lookup.ExpectedAmountForCompetitionEntry(context.Background(), amountLookupEntryID)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if owed.Cents != 6000 {
		t.Fatalf("cents = %d, want 6000", owed.Cents)
	}
	if owed.Currency != "GBP" {
		t.Fatalf("currency = %q, want GBP", owed.Currency)
	}
}

// The property the whole design rests on: what comes back is the entry's
// STORED figure, not EntryFee × (1 + GuestCount) recomputed from the
// Competition. Seeded here with an AmountOwed that deliberately disagrees
// with any such recomputation — an entry made before a price change — so an
// implementation that re-derived instead of reading could not pass.
func TestExpectedAmountForCompetitionEntry_DoesNotRecomputeFromTheCompetition(t *testing.T) {
	t.Parallel()

	repo := newEntryLookupFakeRepository(competitionsdomain.CompetitionEntry{
		ID:            amountLookupEntryID,
		CompetitionID: amountLookupCompetitionID,
		PlayerID:      "player-1",
		GuestCount:    1,
		Status:        competitionsdomain.EntryStatusEntered,
		PaymentStatus: competitionsdomain.PaymentStatusUnpaid,
		// Two heads at the OLD price of 2000. Any recomputation against a
		// Competition that has since moved to 9900 would answer 19800.
		AmountOwed: competitionsdomain.Money{AmountCents: 4000, CurrencyCode: "GBP"},
	})
	lookup := paymentscompetitions.NewEntryAmountLookup(newEntryLookupTestService(repo))

	owed, err := lookup.ExpectedAmountForCompetitionEntry(context.Background(), amountLookupEntryID)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if owed.Cents != 4000 {
		t.Fatalf("cents = %d, want the stored 4000 — the frozen figure must not be re-derived", owed.Cents)
	}
}

// A free entry resolves to a real zero rather than an error or an empty
// Money, so "this entry owes nothing" is a fact the caller can act on.
func TestExpectedAmountForCompetitionEntry_FreeEntryIsZeroNotAnError(t *testing.T) {
	t.Parallel()

	repo := newEntryLookupFakeRepository(competitionsdomain.CompetitionEntry{
		ID:            amountLookupEntryID,
		CompetitionID: amountLookupCompetitionID,
		PlayerID:      "player-1",
		Status:        competitionsdomain.EntryStatusEntered,
		PaymentStatus: competitionsdomain.PaymentStatusUnpaid,
		AmountOwed:    competitionsdomain.Money{},
	})
	lookup := paymentscompetitions.NewEntryAmountLookup(newEntryLookupTestService(repo))

	owed, err := lookup.ExpectedAmountForCompetitionEntry(context.Background(), amountLookupEntryID)
	if err != nil {
		t.Fatalf("a free entry must resolve, got %v", err)
	}
	if owed.Cents != 0 {
		t.Fatalf("cents = %d, want 0", owed.Cents)
	}
}

// An unknown entry is an explicit refusal, translated to Payments' own
// sentinel.
//
// This is the one place this adapter deliberately differs from EntryLookup
// in the same package, which swallows the same miss into a nil error: there
// a missing entry correctly means "nobody is authorized", here it must
// refuse, because answering "no amount, no error" would make the amount
// check bypassable with an entry id that does not exist.
func TestExpectedAmountForCompetitionEntry_UnknownEntryIsRefused(t *testing.T) {
	t.Parallel()

	lookup := paymentscompetitions.NewEntryAmountLookup(newEntryLookupTestService(newEntryLookupFakeRepository()))

	_, err := lookup.ExpectedAmountForCompetitionEntry(context.Background(), amountLookupEntryID)
	if !errors.Is(err, paymentsdomain.ErrPayableNotFound) {
		t.Fatalf("unknown entry = %v, want %v", err, paymentsdomain.ErrPayableNotFound)
	}
}

// No competitionsdomain sentinel may cross this boundary (CLAUDE.md rule
// 5) — the miss above must arrive as Payments' own error, never as
// Competitions'.
func TestExpectedAmountForCompetitionEntry_DoesNotLeakCompetitionsSentinels(t *testing.T) {
	t.Parallel()

	lookup := paymentscompetitions.NewEntryAmountLookup(newEntryLookupTestService(newEntryLookupFakeRepository()))

	_, err := lookup.ExpectedAmountForCompetitionEntry(context.Background(), amountLookupEntryID)
	if errors.Is(err, competitionsdomain.ErrCompetitionEntryNotFound) {
		t.Fatalf("leaked the competitionsdomain sentinel across the boundary: %v", err)
	}
}
