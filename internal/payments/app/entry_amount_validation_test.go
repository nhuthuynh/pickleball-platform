package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nhuthuynh/white-label/internal/payments/adapter/stripestub"
	"github.com/nhuthuynh/white-label/internal/payments/app"
	"github.com/nhuthuynh/white-label/internal/payments/domain"
)

// T57.2 — the Competitions half of the amount check, the half T56.2 (issue
// #297) explicitly left out.
//
// T56.2 made `CreateOnlinePayment` validate a Registration payment against
// what the Registration owes, and named `competition_entry` as out of
// scope because Competitions had no frozen owed amount to compare with.
// T57.1 gives it one, so the exclusion has nothing left to stand on.
//
// What that exclusion meant in the meantime is worth stating plainly: a
// $50.00 Competition entry could be paid for with **one cent**, and
// `reconcileCompetitionEntryPaymentStatus` would then mark the entry paid.
// The identical payment against a Game registration was refused. Two
// payable types, the same shape, opposite behaviour, with nothing but a
// ticket boundary explaining the difference.
//
// The expected figure is the entry's **frozen** AmountOwed (T57.1), not a
// live recomputation from the Competition — validating against a number the
// Host can change afterwards would turn an entrant's correct payment into a
// wrong one the moment the price moved.

// Actor fixtures local to this file. UUID-shaped, per service_test.go's own
// fixture-fidelity note (T10.7): these are User.IDs as of T28.1's
// resolution seam, and a mnemonic string is a shape internal/platform/idgen
// never produces.
const (
	fixtureEntrantPlayerID    = "6ba7b810-0000-4000-8000-0000000000e1"
	fixtureCompetitionAdminID = "6ba7b810-0000-4000-8000-0000000000e2"
)

// fakeEntryAmountLookup stands in for Competitions' side of the read port.
type fakeEntryAmountLookup struct {
	owed map[string]domain.Money
	err  error
}

func (f fakeEntryAmountLookup) ExpectedAmountForCompetitionEntry(_ context.Context, entryID string) (domain.Money, error) {
	if f.err != nil {
		return domain.Money{}, f.err
	}
	m, ok := f.owed[entryID]
	if !ok {
		return domain.Money{}, domain.ErrPayableNotFound
	}
	return m, nil
}

// entryAmountCheckService wires the entry amount lookup alongside the
// authorization fixtures CreateOnlinePayment needs for a competition_entry
// payable — unlike the Registration branch, this one IS authorized
// (authorizeOnlineCreation delegates to
// authorizeCompetitionEntryRecording), so the actor must be the entrant or
// a Competition Admin before the amount is ever reached.
func entryAmountCheckService(repo *fakeRepository, owed domain.Money) *app.Service {
	entries, admins := newEntryAuthzFixtures(fixtureEntrantPlayerID)
	return app.NewService(app.ServiceOptions{
		Payments:               repo,
		IDs:                    &fixedIDs{ids: []string{fixtureCompetitionEntryPaymentID}},
		Processor:              stripestub.NewProcessor(),
		EntryLookup:            entries,
		CompetitionAdminReader: admins,
		EntryAmounts: fakeEntryAmountLookup{
			owed: map[string]domain.Money{fixtureCompetitionEntryID: owed},
		},
	})
}

// The defect, stated as a test: underpaying for an entry must be refused,
// exactly as underpaying for a registration already is.
func TestCreateOnlinePayment_Entry_RejectsAnAmountThatIsNotWhatIsOwed(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	svc := entryAmountCheckService(repo, domain.Money{Cents: 5000, Currency: "GBP"})

	_, err := svc.CreateOnlinePayment(context.Background(), app.CreateOnlinePaymentInput{
		PayableType: domain.PayableTypeCompetitionEntry,
		PayableID:   fixtureCompetitionEntryID,
		Amount:      domain.Money{Cents: 1, Currency: "GBP"},
		ActorUserID: fixtureEntrantPlayerID,
	})
	if !errors.Is(err, domain.ErrAmountMismatch) {
		t.Fatalf("paying 1 cent for a 5000-cent entry = %v, want %v", err, domain.ErrAmountMismatch)
	}

	// And nothing was recorded — a refused payment must leave no Payment
	// behind for a later reconciliation to find and act on.
	if _, getErr := repo.GetByID(context.Background(), fixtureCompetitionEntryPaymentID); !errors.Is(getErr, domain.ErrPaymentNotFound) {
		t.Fatalf("a refused payment created a Payment row: %v", getErr)
	}
}

// Overpaying is refused too. Exact match, not a minimum: a minimum would
// permit silent overcharge, and this check exists precisely because the
// caller's figure is not trusted.
func TestCreateOnlinePayment_Entry_RejectsOverpayment(t *testing.T) {
	t.Parallel()

	svc := entryAmountCheckService(newFakeRepository(), domain.Money{Cents: 5000, Currency: "GBP"})

	_, err := svc.CreateOnlinePayment(context.Background(), app.CreateOnlinePaymentInput{
		PayableType: domain.PayableTypeCompetitionEntry,
		PayableID:   fixtureCompetitionEntryID,
		Amount:      domain.Money{Cents: 9900, Currency: "GBP"},
		ActorUserID: fixtureEntrantPlayerID,
	})
	if !errors.Is(err, domain.ErrAmountMismatch) {
		t.Fatalf("overpaying = %v, want %v", err, domain.ErrAmountMismatch)
	}
}

// The currency is part of the comparison, not just the number.
func TestCreateOnlinePayment_Entry_RejectsRightNumberWrongCurrency(t *testing.T) {
	t.Parallel()

	svc := entryAmountCheckService(newFakeRepository(), domain.Money{Cents: 5000, Currency: "GBP"})

	_, err := svc.CreateOnlinePayment(context.Background(), app.CreateOnlinePaymentInput{
		PayableType: domain.PayableTypeCompetitionEntry,
		PayableID:   fixtureCompetitionEntryID,
		Amount:      domain.Money{Cents: 5000, Currency: "USD"},
		ActorUserID: fixtureEntrantPlayerID,
	})
	if !errors.Is(err, domain.ErrAmountMismatch) {
		t.Fatalf("right number wrong currency = %v, want %v", err, domain.ErrAmountMismatch)
	}
}

// The control: the correct amount still goes through. Without this, an
// implementation that rejected every entry payment would pass everything
// above.
func TestCreateOnlinePayment_Entry_AcceptsTheExactAmountOwed(t *testing.T) {
	t.Parallel()

	svc := entryAmountCheckService(newFakeRepository(), domain.Money{Cents: 5000, Currency: "GBP"})

	p, err := svc.CreateOnlinePayment(context.Background(), app.CreateOnlinePaymentInput{
		PayableType: domain.PayableTypeCompetitionEntry,
		PayableID:   fixtureCompetitionEntryID,
		Amount:      domain.Money{Cents: 5000, Currency: "GBP"},
		ActorUserID: fixtureEntrantPlayerID,
	})
	if err != nil {
		t.Fatalf("paying exactly what is owed = %v, want success", err)
	}
	if p.Amount.Cents != 5000 {
		t.Fatalf("recorded amount = %d, want 5000", p.Amount.Cents)
	}
}

// An unresolvable entry is refused rather than waved through. Failing open
// here would make the check trivially bypassable by sending an entry id
// that does not exist.
func TestCreateOnlinePayment_Entry_UnresolvablePayableIsRefused(t *testing.T) {
	t.Parallel()

	// Authorized as a Competition Admin, so the refusal that arrives is the
	// amount lookup's and not an authorization answer that would mask it.
	entries, admins := newEntryAuthzFixtures(fixtureEntrantPlayerID, fixtureCompetitionAdminID)
	svc := app.NewService(app.ServiceOptions{
		Payments:               newFakeRepository(),
		IDs:                    &fixedIDs{ids: []string{fixtureCompetitionEntryPaymentID}},
		Processor:              stripestub.NewProcessor(),
		EntryLookup:            entries,
		CompetitionAdminReader: admins,
		EntryAmounts:           fakeEntryAmountLookup{owed: map[string]domain.Money{}},
	})

	_, err := svc.CreateOnlinePayment(context.Background(), app.CreateOnlinePaymentInput{
		PayableType: domain.PayableTypeCompetitionEntry,
		PayableID:   fixtureCompetitionEntryID,
		Amount:      domain.Money{Cents: 5000, Currency: "GBP"},
		ActorUserID: fixtureEntrantPlayerID,
	})
	if err == nil {
		t.Fatal("an unresolvable entry was allowed to be paid for")
	}
}

// Authorization runs BEFORE the amount check, so an unauthorized caller
// never learns what an entry costs.
//
// This matters more here than on the Registration branch: that branch has
// no authorization at all (authorizeOnlineCreation only gates
// competition_entry), so this is the one payable type where the ordering is
// observable — and getting it backwards would turn CreateOnlinePayment into
// a price oracle for competitions the caller has nothing to do with.
func TestCreateOnlinePayment_Entry_AuthorizationPrecedesTheAmountCheck(t *testing.T) {
	t.Parallel()

	svc := entryAmountCheckService(newFakeRepository(), domain.Money{Cents: 5000, Currency: "GBP"})

	// A stranger, sending a wrong amount. Both checks would refuse; the
	// question is which answer comes back.
	_, err := svc.CreateOnlinePayment(context.Background(), app.CreateOnlinePaymentInput{
		PayableType: domain.PayableTypeCompetitionEntry,
		PayableID:   fixtureCompetitionEntryID,
		Amount:      domain.Money{Cents: 1, Currency: "GBP"},
		ActorUserID: "6ba7b810-0000-4000-8000-00000000beef",
	})
	if errors.Is(err, domain.ErrAmountMismatch) {
		t.Fatal("an unauthorized caller was told the amount was wrong — that discloses what the entry costs")
	}
	if !errors.Is(err, domain.ErrNotPaymentRecorder) {
		t.Fatalf("unauthorized entry payment = %v, want %v", err, domain.ErrNotPaymentRecorder)
	}
}

// The port is optional, and a Service built without one skips the check
// rather than panicking — the same treatment RegistrationAmounts gets, and
// for the same reason: the existing tests in this package, none of which
// are about amounts, must keep reaching the behaviour they are actually
// about. cmd/server always wires it.
func TestCreateOnlinePayment_Entry_WithoutTheLookupTheCheckIsSkipped(t *testing.T) {
	t.Parallel()

	entries, admins := newEntryAuthzFixtures(fixtureEntrantPlayerID)
	svc := app.NewService(app.ServiceOptions{
		Payments:               newFakeRepository(),
		IDs:                    &fixedIDs{ids: []string{fixtureCompetitionEntryPaymentID}},
		Processor:              stripestub.NewProcessor(),
		EntryLookup:            entries,
		CompetitionAdminReader: admins,
		// EntryAmounts deliberately omitted.
	})

	if _, err := svc.CreateOnlinePayment(context.Background(), app.CreateOnlinePaymentInput{
		PayableType: domain.PayableTypeCompetitionEntry,
		PayableID:   fixtureCompetitionEntryID,
		Amount:      domain.Money{Cents: 1, Currency: "GBP"},
		ActorUserID: fixtureEntrantPlayerID,
	}); err != nil {
		t.Fatalf("without an EntryAmounts port the check must be skipped, got %v", err)
	}
}

// The two lookups are independent: a Service wired with only the
// Registration half must not start refusing entry payments, and vice
// versa. Pinned because the obvious refactor — one nil check guarding
// both branches — would silently couple them.
func TestCreateOnlinePayment_TheTwoAmountLookupsAreIndependent(t *testing.T) {
	t.Parallel()

	entries, admins := newEntryAuthzFixtures(fixtureEntrantPlayerID)
	regs, games, gameAdmins := newGameAuthzFixtures(fixtureGameHostID)

	// Registration half wired, entry half not: a wrong-amount ENTRY payment
	// goes through, a wrong-amount REGISTRATION payment does not.
	svc := app.NewService(app.ServiceOptions{
		Payments:               newFakeRepository(),
		IDs:                    &fixedIDs{ids: []string{fixtureCompetitionEntryPaymentID, fixtureRegistrationPaymentID}},
		Processor:              stripestub.NewProcessor(),
		EntryLookup:            entries,
		CompetitionAdminReader: admins,
		RegistrationLookup:     regs,
		GameLookup:             games,
		GameAdminReader:        gameAdmins,
		RegistrationAmounts: fakeAmountLookup{
			owed: map[string]domain.Money{fixtureRegistrationID: {Cents: 2500, Currency: "USD"}},
		},
	})

	if _, err := svc.CreateOnlinePayment(context.Background(), app.CreateOnlinePaymentInput{
		PayableType: domain.PayableTypeCompetitionEntry,
		PayableID:   fixtureCompetitionEntryID,
		Amount:      domain.Money{Cents: 1, Currency: "GBP"},
		ActorUserID: fixtureEntrantPlayerID,
	}); err != nil {
		t.Fatalf("an unwired entry lookup must not refuse an entry payment, got %v", err)
	}

	_, err := svc.CreateOnlinePayment(context.Background(), app.CreateOnlinePaymentInput{
		PayableType: domain.PayableTypeRegistration,
		PayableID:   fixtureRegistrationID,
		Amount:      domain.Money{Cents: 1, Currency: "USD"},
		ActorUserID: fixtureGameHostID,
	})
	if !errors.Is(err, domain.ErrAmountMismatch) {
		t.Fatalf("the wired registration lookup must still refuse, got %v", err)
	}
}
