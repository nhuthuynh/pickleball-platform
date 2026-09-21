package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nhuthuynh/white-label/internal/payments/adapter/stripestub"
	"github.com/nhuthuynh/white-label/internal/payments/app"
	"github.com/nhuthuynh/white-label/internal/payments/domain"
)

// T56.2 (issue #297) — a payment's amount is checked against what the
// payable actually owes, instead of being believed.
//
// Before this, `CreateOnlinePayment` took the `Money` straight off the wire
// and nothing compared it to anything: Payments had read-side ports for
// *ownership* facts but none for price. A $25 Game could be paid for with
// one cent, and `reconcileRegistrationPaymentStatus` would then mark the
// Registration paid in full.
//
// The expected figure is the Registration's **frozen** `AmountOwed`
// (T56.1), not a live recomputation from the Game — validating against a
// number the Host can change afterwards would turn a player's correct
// payment into a wrong one the moment the price moved.

// fakeAmountLookup stands in for Social Play's side of the read port.
type fakeAmountLookup struct {
	owed map[string]domain.Money
	err  error
}

func (f fakeAmountLookup) ExpectedAmountForRegistration(_ context.Context, registrationID string) (domain.Money, error) {
	if f.err != nil {
		return domain.Money{}, f.err
	}
	m, ok := f.owed[registrationID]
	if !ok {
		return domain.Money{}, domain.ErrPayableNotFound
	}
	return m, nil
}

func amountCheckService(repo *fakeRepository, owed domain.Money) *app.Service {
	regs, games, admins := newGameAuthzFixtures(fixtureGameHostID)
	return app.NewService(app.ServiceOptions{
		Payments:           repo,
		IDs:                &fixedIDs{ids: []string{fixtureRegistrationPaymentID}},
		Processor:          stripestub.NewProcessor(),
		RegistrationLookup: regs,
		GameLookup:         games,
		GameAdminReader:    admins,
		PayableAmounts: fakeAmountLookup{
			owed: map[string]domain.Money{fixtureRegistrationID: owed},
		},
	})
}

// The defect, stated as a test: underpaying must be refused.
func TestCreateOnlinePayment_RejectsAnAmountThatIsNotWhatIsOwed(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	svc := amountCheckService(repo, domain.Money{Cents: 2500, Currency: "USD"})

	_, err := svc.CreateOnlinePayment(context.Background(), app.CreateOnlinePaymentInput{
		PayableType: domain.PayableTypeRegistration,
		PayableID:   fixtureRegistrationID,
		Amount:      domain.Money{Cents: 1, Currency: "USD"},
		ActorUserID: fixtureGameHostID,
	})
	if !errors.Is(err, domain.ErrAmountMismatch) {
		t.Fatalf("paying 1 cent for a 2500-cent registration = %v, want %v", err, domain.ErrAmountMismatch)
	}

	// And nothing was recorded — a refused payment must leave no Payment
	// behind for a later reconciliation to find and act on.
	if _, getErr := repo.GetByID(context.Background(), fixtureRegistrationPaymentID); !errors.Is(getErr, domain.ErrPaymentNotFound) {
		t.Fatalf("a refused payment created a Payment row: %v", getErr)
	}
}

// Overpaying is refused too. Exact match, not a minimum: a minimum would
// permit silent overcharge, and this check exists precisely because the
// caller's figure is not trusted.
func TestCreateOnlinePayment_RejectsOverpayment(t *testing.T) {
	t.Parallel()

	svc := amountCheckService(newFakeRepository(), domain.Money{Cents: 2500, Currency: "USD"})

	_, err := svc.CreateOnlinePayment(context.Background(), app.CreateOnlinePaymentInput{
		PayableType: domain.PayableTypeRegistration,
		PayableID:   fixtureRegistrationID,
		Amount:      domain.Money{Cents: 9900, Currency: "USD"},
		ActorUserID: fixtureGameHostID,
	})
	if !errors.Is(err, domain.ErrAmountMismatch) {
		t.Fatalf("overpaying = %v, want %v", err, domain.ErrAmountMismatch)
	}
}

// The currency is part of the comparison, not just the number. Paying
// "2500" in the wrong currency is a different amount of money.
func TestCreateOnlinePayment_RejectsRightNumberWrongCurrency(t *testing.T) {
	t.Parallel()

	svc := amountCheckService(newFakeRepository(), domain.Money{Cents: 2500, Currency: "USD"})

	_, err := svc.CreateOnlinePayment(context.Background(), app.CreateOnlinePaymentInput{
		PayableType: domain.PayableTypeRegistration,
		PayableID:   fixtureRegistrationID,
		Amount:      domain.Money{Cents: 2500, Currency: "EUR"},
		ActorUserID: fixtureGameHostID,
	})
	if !errors.Is(err, domain.ErrAmountMismatch) {
		t.Fatalf("right number wrong currency = %v, want %v", err, domain.ErrAmountMismatch)
	}
}

// The control: the correct amount still goes through. Without this, an
// implementation that rejected everything would pass all the tests above.
func TestCreateOnlinePayment_AcceptsTheExactAmountOwed(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	svc := amountCheckService(repo, domain.Money{Cents: 2500, Currency: "USD"})

	p, err := svc.CreateOnlinePayment(context.Background(), app.CreateOnlinePaymentInput{
		PayableType: domain.PayableTypeRegistration,
		PayableID:   fixtureRegistrationID,
		Amount:      domain.Money{Cents: 2500, Currency: "USD"},
		ActorUserID: fixtureGameHostID,
	})
	if err != nil {
		t.Fatalf("paying exactly what is owed = %v, want success", err)
	}
	if p.Amount.Cents != 2500 {
		t.Fatalf("recorded amount = %d, want 2500", p.Amount.Cents)
	}
}

// A payable that resolves to nothing is refused rather than waved through.
// Failing open here would make the check trivially bypassable by sending a
// payable id that does not exist.
func TestCreateOnlinePayment_UnresolvablePayableIsRefused(t *testing.T) {
	t.Parallel()

	svc := amountCheckService(newFakeRepository(), domain.Money{Cents: 2500, Currency: "USD"})

	_, err := svc.CreateOnlinePayment(context.Background(), app.CreateOnlinePaymentInput{
		PayableType: domain.PayableTypeRegistration,
		PayableID:   "00000000-0000-4000-a000-0000000000ff",
		Amount:      domain.Money{Cents: 2500, Currency: "USD"},
		ActorUserID: fixtureGameHostID,
	})
	if err == nil {
		t.Fatal("an unresolvable payable was allowed to be paid for")
	}
}

// The port is optional, and a Service built without one skips the check
// rather than panicking — matching RegistrationUpdater's established
// optional-port treatment. This is NOT a security hole by omission: it is
// how the ~40 existing tests in this package, none of which are about
// amounts, keep working. cmd/server always wires it.
//
// Pinned as a test so the behaviour is deliberate and visible, rather than
// an accident of a nil check somebody might "tidy up" later.
func TestCreateOnlinePayment_WithoutTheLookupTheCheckIsSkipped(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	regs, games, admins := newGameAuthzFixtures(fixtureGameHostID)
	svc := app.NewService(app.ServiceOptions{
		Payments:           repo,
		IDs:                &fixedIDs{ids: []string{fixtureRegistrationPaymentID}},
		Processor:          stripestub.NewProcessor(),
		RegistrationLookup: regs,
		GameLookup:         games,
		GameAdminReader:    admins,
		// PayableAmounts deliberately omitted.
	})

	if _, err := svc.CreateOnlinePayment(context.Background(), app.CreateOnlinePaymentInput{
		PayableType: domain.PayableTypeRegistration,
		PayableID:   fixtureRegistrationID,
		Amount:      domain.Money{Cents: 1, Currency: "USD"},
		ActorUserID: fixtureGameHostID,
	}); err != nil {
		t.Fatalf("without a PayableAmounts port the check must be skipped, got %v", err)
	}
}
