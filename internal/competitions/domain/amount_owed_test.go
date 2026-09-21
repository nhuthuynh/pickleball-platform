package domain_test

import (
	"errors"
	"testing"

	"github.com/nhuthuynh/white-label/internal/competitions/domain"
)

// T57.1 — Competitions' mirror of T56.1 (issue #126): a CompetitionEntry
// owes the Competition's EntryFee once per HEAD, not once per entry.
//
// This is the same defect T56.1 fixed in Social Play, in the context that
// was explicitly left out of it. Competition.EntryFee's own doc comment
// says it is "what one entry costs", and until this ticket that was taken
// literally: an entrant bringing three guests paid for one, because nothing
// multiplied the fee by the party size. The weighted capacity rule has
// always known that each guest occupies a place — Enter sums
// (1 + GuestCount) — so the Competition was correctly sold out by four
// heads and correctly billed for one.
//
// The asymmetry between the two contexts is the thing worth naming: after
// T56.1 a Game charged per head and a Competition did not, and nothing in
// either context said why. That is not a ubiquitous language (CLAUDE.md
// rule 7); it is one rule with an undocumented exception.

func competitionWithFee(t *testing.T, cents int64, currency string, guestAllowance int) domain.Competition {
	t.Helper()

	c, err := domain.NewCompetition(
		"comp-1", "host-1", "Autumn Doubles Open", "venue-1",
		validSessions(t), 16, guestAllowance,
		domain.PaymentMethodEither,
		domain.Money{AmountCents: cents, CurrencyCode: currency},
		domain.FormatDoubles, "tok-1",
	)
	if err != nil {
		t.Fatalf("NewCompetition: %v", err)
	}
	return c
}

// The arithmetic, including the boundary the whole ticket turns on: an
// entrant alone is 1 head, not 0.
func TestExpectedAmount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		feeCents   int64
		currency   string
		guestCount int
		wantCents  int64
	}{
		{name: "no guests pays for one head", feeCents: 2000, currency: "GBP", guestCount: 0, wantCents: 2000},
		{name: "one guest pays for two heads", feeCents: 2000, currency: "GBP", guestCount: 1, wantCents: 4000},
		{name: "three guests pays for four heads", feeCents: 2000, currency: "GBP", guestCount: 3, wantCents: 8000},
		{
			// The regression this ticket exists to prevent: a free
			// Competition stays free for a whole party.
			name: "a free competition is free for any party size", feeCents: 0, currency: "", guestCount: 3, wantCents: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			c := competitionWithFee(t, tc.feeCents, tc.currency, 5)
			got := domain.ExpectedAmount(c, tc.guestCount)

			if got.AmountCents != tc.wantCents {
				t.Fatalf("ExpectedAmount cents = %d, want %d", got.AmountCents, tc.wantCents)
			}
			// The currency rides along with the amount — ADR-0005's whole
			// point is that a bare cents figure never exists in the domain.
			if got.CurrencyCode != tc.currency {
				t.Fatalf("ExpectedAmount currency = %q, want %q", got.CurrencyCode, tc.currency)
			}
		})
	}
}

// Enter freezes the owed amount onto the entry rather than leaving it to be
// recomputed later — the same choice the Product Owner made for Social Play
// on 2026-09-04 ("freeze, and lock guests once paid"), applied here for the
// same reasons: a Host who raises the entry fee afterwards must not change
// what an entrant already agreed to pay, and T57.2's payment validation is
// only meaningful against the figure actually agreed.
func TestEnterFreezesAmountOwed(t *testing.T) {
	t.Parallel()

	c := competitionWithFee(t, 2000, "GBP", 3)

	e, err := domain.Enter(c, nil, "player-1", 2, domain.EntrySourceApp)
	if err != nil {
		t.Fatalf("Enter: %v", err)
	}

	if e.AmountOwed.AmountCents != 6000 {
		t.Fatalf("AmountOwed cents = %d, want 6000 (2000 × 3 heads)", e.AmountOwed.AmountCents)
	}
	if e.AmountOwed.CurrencyCode != "GBP" {
		t.Fatalf("AmountOwed currency = %q, want GBP", e.AmountOwed.CurrencyCode)
	}
}

// The frozen figure must not drift when the Competition's fee later
// changes. This is the property that makes T57.2's server-side validation
// mean something: it compares a payment against what was actually agreed,
// not against a number that moved underneath the entrant.
func TestEnterAmountOwedDoesNotFollowALaterFeeChange(t *testing.T) {
	t.Parallel()

	c := competitionWithFee(t, 2000, "GBP", 3)

	e, err := domain.Enter(c, nil, "player-1", 1, domain.EntrySourceApp)
	if err != nil {
		t.Fatalf("Enter: %v", err)
	}
	if e.AmountOwed.AmountCents != 4000 {
		t.Fatalf("AmountOwed cents = %d, want 4000", e.AmountOwed.AmountCents)
	}

	// The Host raises the price after this entrant entered.
	c.EntryFee = domain.Money{AmountCents: 9900, CurrencyCode: "GBP"}

	if e.AmountOwed.AmountCents != 4000 {
		t.Fatalf("AmountOwed changed to %d after the Competition's fee changed — it must be frozen at entry", e.AmountOwed.AmountCents)
	}
}

// A free Competition produces a free entry, and IsZero is what says so —
// not a bare `AmountCents == 0` comparison, per Money's own doc comment
// warning against the "zero means unset" misreading.
func TestEnterFreeCompetitionOwesNothing(t *testing.T) {
	t.Parallel()

	c := competitionWithFee(t, 0, "", 3)

	e, err := domain.Enter(c, nil, "player-1", 3, domain.EntrySourceApp)
	if err != nil {
		t.Fatalf("Enter: %v", err)
	}
	if !e.AmountOwed.IsZero() {
		t.Fatalf("AmountOwed = %+v, want free", e.AmountOwed)
	}
}

// Every existing rejection still runs before the amount is computed, so a
// refused entry is never priced. Guest allowance is the case worth pinning:
// it is the check that shares an input with the pricing rule.
func TestEnterRejectsOverAllowanceBeforePricing(t *testing.T) {
	t.Parallel()

	c := competitionWithFee(t, 2000, "GBP", 1)

	_, err := domain.Enter(c, nil, "player-1", 2, domain.EntrySourceApp)
	if !errors.Is(err, domain.ErrGuestAllowanceExceeded) {
		t.Fatalf("Enter with too many guests = %v, want %v", err, domain.ErrGuestAllowanceExceeded)
	}
}
