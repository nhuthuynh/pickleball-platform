package competitions

import (
	"context"
	"errors"
	"fmt"

	competitionsapp "github.com/nhuthuynh/white-label/internal/competitions/app"
	competitionsdomain "github.com/nhuthuynh/white-label/internal/competitions/domain"
	paymentsdomain "github.com/nhuthuynh/white-label/internal/payments/domain"
)

// EntryAmountLookup implements payments/port.EntryAmountLookup over the
// Competitions context's real app.Service (T57.2) — the Competitions mirror
// of internal/payments/adapter/socialplay.RegistrationAmountLookup, and the
// price counterpart to EntryLookup in this same package.
type EntryAmountLookup struct {
	competitionsSvc *competitionsapp.Service
}

// NewEntryAmountLookup builds an EntryAmountLookup against the real, shared
// Competitions app.Service instance, mirroring NewEntryLookup's identical
// constructor shape.
func NewEntryAmountLookup(competitionsSvc *competitionsapp.Service) *EntryAmountLookup {
	return &EntryAmountLookup{competitionsSvc: competitionsSvc}
}

// ExpectedAmountForCompetitionEntry returns what the CompetitionEntry owes.
//
// # It returns the STORED figure, not a recomputation
//
// This reads `CompetitionEntry.AmountOwed`, frozen when the entrant entered
// (T57.1), rather than re-deriving `EntryFee × (1 + GuestCount)` from the
// Competition. That is the whole point of freezing it: a Host who raises
// the entry fee afterwards must not retroactively make an entrant's correct
// payment wrong. Re-deriving here would reintroduce exactly the drift the
// frozen column exists to prevent — invisibly, since both paths return a
// plausible number.
//
// # Error translation
//
// competitionsdomain.ErrCompetitionEntryNotFound becomes Payments' own
// ErrPayableNotFound: no Competitions error type crosses this boundary
// (CLAUDE.md rule 5). Anything else is wrapped with %s rather than %w, so
// no competitionsdomain sentinel leaks by accident either.
//
// Note this deliberately does NOT follow EntryLookup's swallow-the-miss
// convention. That method answers an *authorization* question, where a
// missing entry resolving to ("", "", nil) correctly means "nobody is
// authorized"; here a missing entry must be an explicit refusal, because
// answering "no amount, no error" would make the whole check bypassable by
// sending an entry id that does not exist. Two methods on the same context,
// opposite handling of the same condition, because the condition means
// different things to their callers — see port.EntryAmountLookup and
// port.EntryLookup's own doc comments.
func (l *EntryAmountLookup) ExpectedAmountForCompetitionEntry(ctx context.Context, entryID string) (paymentsdomain.Money, error) {
	entry, err := l.competitionsSvc.GetEntryByID(ctx, entryID)
	if err != nil {
		if errors.Is(err, competitionsdomain.ErrCompetitionEntryNotFound) {
			return paymentsdomain.Money{}, paymentsdomain.ErrPayableNotFound
		}
		return paymentsdomain.Money{}, fmt.Errorf("payments competitions adapter: resolving amount owed for entry %s: %s", entryID, err)
	}

	// Money is rebuilt field by field rather than converted wholesale. The
	// two types are not even the same shape here — Competitions' Money
	// names its fields AmountCents/CurrencyCode while Payments' uses
	// Cents/Currency — which makes the point competitions/domain.Money's
	// own doc comment argues: these are context-local value types free to
	// diverge, never a shared kernel.
	return paymentsdomain.Money{
		Cents:    entry.AmountOwed.AmountCents,
		Currency: entry.AmountOwed.CurrencyCode,
	}, nil
}
