package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nhuthuynh/white-label/internal/payments/app"
	"github.com/nhuthuynh/white-label/internal/payments/domain"
)

// T58 (closes issue #299) — RecordOfflinePayment checks the amount too.
//
// # The decision this implements, and the premise that had to be corrected
//
// #299 was filed asking whether the offline path should validate, on the
// reasoning that "a Game Admin recording cash may legitimately record a
// part-payment, and refusing one would mean the system cannot represent
// money that genuinely changed hands."
//
// That reasoning described a capability this system does not have.
// `payments_payable_unique_idx` (db/migrations/0005_payments.sql) enforces
// ONE Payment per (payable_type, payable_id) in Postgres. So a part-payment
// was never supported-but-unvalidated — it was, and is, impossible:
//
//   - a Host takes $20.00 of a $30.00 debt and records it;
//   - reconcileRegistrationPaymentStatus marks the Registration PAID IN
//     FULL, because it keys off the existence of a Payment and never looks
//     at its amount;
//   - the remaining $10.00 can never be recorded, because a second Payment
//     for the same payable is refused with ErrPaymentAlreadyRecorded.
//
// The part-payment case was therefore already broken: mis-recorded, then
// locked. Given that, exact-match is the only rule consistent with the
// one-Payment-per-payable design already enforced in the database. If you
// may record exactly one Payment, it had better be the whole amount, or
// the record is a lie. (Product Owner decision, 2026-09-22.)
//
// Refusing a $20.00 record against a $30.00 debt is a real cost, and it is
// the honest failure: today that record is accepted and the system then
// reports the debt settled. Properly supporting part-payments means
// dropping the unique index, tracking amount-paid against amount-owed, and
// deriving payment status across Social Play AND Competitions — its own
// sprint, not this ticket.
//
// # Why this shares the online path's machinery rather than copying it
//
// The same ports, the same sentinel, the same scope rules. Two
// near-identical amount checks that could drift is exactly the failure
// CLAUDE.md rule 7 is about — see expectedAmountFor, which both paths now
// call.

func offlineRegistrationAmountService(repo *fakeRepository, owed domain.Money) *app.Service {
	regs, games, admins := newGameAuthzFixtures(fixtureGameHostID)
	return app.NewService(app.ServiceOptions{
		Payments:           repo,
		IDs:                &fixedIDs{ids: []string{fixtureRegistrationPaymentID}},
		RegistrationLookup: regs,
		GameLookup:         games,
		GameAdminReader:    admins,
		RegistrationAmounts: fakeAmountLookup{
			owed: map[string]domain.Money{fixtureRegistrationID: owed},
		},
	})
}

func offlineEntryAmountService(repo *fakeRepository, owed domain.Money) *app.Service {
	entries, admins := newEntryAuthzFixtures(fixtureEntrantPlayerID)
	return app.NewService(app.ServiceOptions{
		Payments:               repo,
		IDs:                    &fixedIDs{ids: []string{fixtureCompetitionEntryPaymentID}},
		EntryLookup:            entries,
		CompetitionAdminReader: admins,
		EntryAmounts: fakeEntryAmountLookup{
			owed: map[string]domain.Money{fixtureCompetitionEntryID: owed},
		},
	})
}

// The defect, stated as a test: a Host recording $1.00 of cash against a
// $30.00 registration must be refused.
//
// Note what made this worse than the online equivalent: RecordOfflinePayment
// marks the Payment PAID immediately (there is no confirm step), so the
// bogus figure reached reconciliation in the same call.
func TestRecordOfflinePayment_RejectsAnAmountThatIsNotWhatIsOwed(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	svc := offlineRegistrationAmountService(repo, domain.Money{Cents: 3000, Currency: "USD"})

	_, err := svc.RecordOfflinePayment(context.Background(), app.RecordOfflinePaymentInput{
		PayableType: domain.PayableTypeRegistration,
		PayableID:   fixtureRegistrationID,
		Amount:      domain.Money{Cents: 100, Currency: "USD"},
		ActorUserID: fixtureGameHostID,
	})
	if !errors.Is(err, domain.ErrAmountMismatch) {
		t.Fatalf("recording 100 cents against a 3000-cent registration = %v, want %v", err, domain.ErrAmountMismatch)
	}

	// Nothing recorded. This matters more here than on the online path: a
	// Payment written by this method is already StatusPaid, so a row left
	// behind would not merely await confirmation — it would be a settled
	// debt, and the unique index would then block the correct record.
	if _, getErr := repo.GetByID(context.Background(), fixtureRegistrationPaymentID); !errors.Is(getErr, domain.ErrPaymentNotFound) {
		t.Fatalf("a refused offline payment created a Payment row: %v", getErr)
	}
}

// A refused record must leave the payable's slot FREE, so the Host can
// immediately record the correct amount. Without this, the first mistake
// would permanently block the real payment via the unique index — turning a
// validation into a denial of service on the payable.
func TestRecordOfflinePayment_ARefusedRecordDoesNotBlockTheCorrectOne(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	svc := offlineRegistrationAmountService(repo, domain.Money{Cents: 3000, Currency: "USD"})

	if _, err := svc.RecordOfflinePayment(context.Background(), app.RecordOfflinePaymentInput{
		PayableType: domain.PayableTypeRegistration,
		PayableID:   fixtureRegistrationID,
		Amount:      domain.Money{Cents: 100, Currency: "USD"},
		ActorUserID: fixtureGameHostID,
	}); !errors.Is(err, domain.ErrAmountMismatch) {
		t.Fatalf("setup: expected the wrong amount to be refused, got %v", err)
	}

	p, err := svc.RecordOfflinePayment(context.Background(), app.RecordOfflinePaymentInput{
		PayableType: domain.PayableTypeRegistration,
		PayableID:   fixtureRegistrationID,
		Amount:      domain.Money{Cents: 3000, Currency: "USD"},
		ActorUserID: fixtureGameHostID,
	})
	if err != nil {
		t.Fatalf("recording the correct amount after a refusal = %v, want success", err)
	}
	if p.Amount.Cents != 3000 {
		t.Fatalf("recorded amount = %d, want 3000", p.Amount.Cents)
	}
}

// Overpaying is refused too — a Host cannot record $50.00 of cash against a
// $30.00 debt. Exact match, not a floor.
func TestRecordOfflinePayment_RejectsOverpayment(t *testing.T) {
	t.Parallel()

	svc := offlineRegistrationAmountService(newFakeRepository(), domain.Money{Cents: 3000, Currency: "USD"})

	_, err := svc.RecordOfflinePayment(context.Background(), app.RecordOfflinePaymentInput{
		PayableType: domain.PayableTypeRegistration,
		PayableID:   fixtureRegistrationID,
		Amount:      domain.Money{Cents: 5000, Currency: "USD"},
		ActorUserID: fixtureGameHostID,
	})
	if !errors.Is(err, domain.ErrAmountMismatch) {
		t.Fatalf("overpaying = %v, want %v", err, domain.ErrAmountMismatch)
	}
}

// The control: the correct amount is still recorded and still reconciles.
func TestRecordOfflinePayment_AcceptsTheExactAmountOwed(t *testing.T) {
	t.Parallel()

	svc := offlineRegistrationAmountService(newFakeRepository(), domain.Money{Cents: 3000, Currency: "USD"})

	p, err := svc.RecordOfflinePayment(context.Background(), app.RecordOfflinePaymentInput{
		PayableType: domain.PayableTypeRegistration,
		PayableID:   fixtureRegistrationID,
		Amount:      domain.Money{Cents: 3000, Currency: "USD"},
		ActorUserID: fixtureGameHostID,
	})
	if err != nil {
		t.Fatalf("recording exactly what is owed = %v, want success", err)
	}
	if p.Status != domain.StatusPaid {
		t.Fatalf("status = %v, want paid — an offline record settles immediately", p.Status)
	}
}

// The Competitions half, which had the identical hole.
func TestRecordOfflinePayment_Entry_RejectsAnAmountThatIsNotWhatIsOwed(t *testing.T) {
	t.Parallel()

	svc := offlineEntryAmountService(newFakeRepository(), domain.Money{Cents: 5000, Currency: "GBP"})

	_, err := svc.RecordOfflinePayment(context.Background(), app.RecordOfflinePaymentInput{
		PayableType: domain.PayableTypeCompetitionEntry,
		PayableID:   fixtureCompetitionEntryID,
		Amount:      domain.Money{Cents: 100, Currency: "GBP"},
		ActorUserID: fixtureEntrantPlayerID,
	})
	if !errors.Is(err, domain.ErrAmountMismatch) {
		t.Fatalf("recording 100 cents against a 5000-cent entry = %v, want %v", err, domain.ErrAmountMismatch)
	}
}

func TestRecordOfflinePayment_Entry_AcceptsTheExactAmountOwed(t *testing.T) {
	t.Parallel()

	svc := offlineEntryAmountService(newFakeRepository(), domain.Money{Cents: 5000, Currency: "GBP"})

	if _, err := svc.RecordOfflinePayment(context.Background(), app.RecordOfflinePaymentInput{
		PayableType: domain.PayableTypeCompetitionEntry,
		PayableID:   fixtureCompetitionEntryID,
		Amount:      domain.Money{Cents: 5000, Currency: "GBP"},
		ActorUserID: fixtureEntrantPlayerID,
	}); err != nil {
		t.Fatalf("recording exactly what is owed = %v, want success", err)
	}
}

// A no-show fee is NOT checked, and this is the test that keeps that
// deliberate rather than accidental.
//
// There is no "correct" figure for a no-show fee to match: the amount is
// whatever the Game Admin decided to charge, and no owed-amount exists
// anywhere to compare it against. It shares a payable id with a
// Registration (the fee is levied against that registration), so a check
// that keyed off the id rather than the TYPE would compare the fee to the
// entry price and refuse every one of them — which is precisely the bug
// this test exists to catch.
func TestRecordOfflinePayment_NoShowFeeIsNotAmountChecked(t *testing.T) {
	t.Parallel()

	// The registration this fee is levied against owes 3000; the fee is 500.
	svc := offlineRegistrationAmountService(newFakeRepository(), domain.Money{Cents: 3000, Currency: "USD"})

	if _, err := svc.RecordOfflinePayment(context.Background(), app.RecordOfflinePaymentInput{
		PayableType: domain.PayableTypeNoShowFee,
		PayableID:   fixtureRegistrationID,
		Amount:      domain.Money{Cents: 500, Currency: "USD"},
		ActorUserID: fixtureGameHostID,
	}); err != nil {
		t.Fatalf("a no-show fee must not be amount-checked against the registration's price, got %v", err)
	}
}

// Authorization still runs before the amount check, so an unauthorized
// actor never learns what a payable costs.
func TestRecordOfflinePayment_AuthorizationPrecedesTheAmountCheck(t *testing.T) {
	t.Parallel()

	svc := offlineRegistrationAmountService(newFakeRepository(), domain.Money{Cents: 3000, Currency: "USD"})

	_, err := svc.RecordOfflinePayment(context.Background(), app.RecordOfflinePaymentInput{
		PayableType: domain.PayableTypeRegistration,
		PayableID:   fixtureRegistrationID,
		Amount:      domain.Money{Cents: 100, Currency: "USD"},
		ActorUserID: "6ba7b810-0000-4000-8000-00000000beef",
	})
	if errors.Is(err, domain.ErrAmountMismatch) {
		t.Fatal("an unauthorized actor was told the amount was wrong — that discloses what the payable costs")
	}
	if !errors.Is(err, domain.ErrNotPaymentRecorder) {
		t.Fatalf("unauthorized offline record = %v, want %v", err, domain.ErrNotPaymentRecorder)
	}
}

// The port stays optional here too: a Service without it records without
// checking, which is what keeps the many tests in this package that are not
// about amounts working unchanged. cmd/server always wires it.
func TestRecordOfflinePayment_WithoutTheLookupTheCheckIsSkipped(t *testing.T) {
	t.Parallel()

	regs, games, admins := newGameAuthzFixtures(fixtureGameHostID)
	svc := app.NewService(app.ServiceOptions{
		Payments:           newFakeRepository(),
		IDs:                &fixedIDs{ids: []string{fixtureRegistrationPaymentID}},
		RegistrationLookup: regs,
		GameLookup:         games,
		GameAdminReader:    admins,
		// RegistrationAmounts deliberately omitted.
	})

	if _, err := svc.RecordOfflinePayment(context.Background(), app.RecordOfflinePaymentInput{
		PayableType: domain.PayableTypeRegistration,
		PayableID:   fixtureRegistrationID,
		Amount:      domain.Money{Cents: 100, Currency: "USD"},
		ActorUserID: fixtureGameHostID,
	}); err != nil {
		t.Fatalf("without a RegistrationAmounts port the check must be skipped, got %v", err)
	}
}
