package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nhuthuynh/white-label/internal/payments/app"
	"github.com/nhuthuynh/white-label/internal/payments/domain"
)

// T55.4 (closes issue #130) — RefundPayment admits `no_show_fee` payables.
//
// #130's report: T12.3's scope sentence named `registration` and `booking`,
// and `no_show_fee` appeared in NEITHER half of it — not included, and
// unlike `competition_entry` (#125) not excluded either. It was therefore
// left out rather than silently admitted, and rejected with
// ErrInvalidPayableType. The practical consequence: a Game Admin could
// charge a no-show fee and had no way to reverse one charged in error.
//
// The Product Owner confirmed on 2026-09-04 that reversing one is the
// wanted behaviour, and that the refund projects NOTHING — the option that
// would also clear the no-show mark was considered and rejected, because it
// would couple payment state to attendance state, which this design
// deliberately keeps apart.
//
// This file is the accepted-case counterpart to the `no_show_fee` case
// removed from TestRefundPayment_OutOfScopePayableTypesRejected, following
// exactly the move-don't-delete precedent T16.4 set for `competition_entry`.

// The happy path: a Game Host may reverse a no-show fee they charged.
// Authorization is not new work — authorizeOfflineRecording already routes
// PayableTypeNoShowFee through authorizeGameRecording, the same
// Host-or-assigned-Game-Admin rule a registration payable already uses.
func TestRefundPayment_OfflineNoShowFeePayable_GameHostSucceeds(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	regs, games, admins := newGameAuthzFixtures(fixtureGameHostID)
	svc := app.NewService(app.ServiceOptions{
		Payments:           repo,
		IDs:                &fixedIDs{ids: []string{fixtureNoShowFeePaymentID}},
		RegistrationLookup: regs,
		GameLookup:         games,
		GameAdminReader:    admins,
	})

	if _, err := svc.RecordOfflinePayment(context.Background(), app.RecordOfflinePaymentInput{
		PayableType: domain.PayableTypeNoShowFee,
		PayableID:   fixtureRegistrationID,
		Amount:      offlineFixtureAmount(),
		ActorUserID: fixtureGameHostID,
	}); err != nil {
		t.Fatalf("seed: RecordOfflinePayment: %v", err)
	}

	refunded, err := svc.RefundPayment(context.Background(), app.RefundPaymentInput{
		PaymentID:   fixtureNoShowFeePaymentID,
		ActorUserID: fixtureGameHostID,
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if refunded.Status != domain.StatusRefunded {
		t.Fatalf("Status = %v, want refunded", refunded.Status)
	}

	stored, err := repo.GetByID(context.Background(), fixtureNoShowFeePaymentID)
	if err != nil {
		t.Fatalf("expected the Payment to still exist: %v", err)
	}
	if stored.Status != domain.StatusRefunded {
		t.Fatalf("persisted Status = %v, want refunded — the transition must be persisted, not just returned", stored.Status)
	}
}

// THE load-bearing test of this ticket, and the reason #130 was a decision
// rather than a one-line gate change.
//
// A no-show fee is a SEPARATE charge from the Registration's own seat —
// reconcileRegistrationPaymentStatus has always excluded it deliberately
// (see its doc comment). So refunding one must NOT push `refunded` through
// RegistrationPaymentUpdater and flip the Registration's own payment status,
// which would tell Social Play the player never paid for their spot.
//
// Without this negative assertion a later refactor could quietly start
// projecting and every other test here would still pass.
func TestRefundPayment_NoShowFeeRefundProjectsNothing(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	updater := &fakeRegistrationUpdater{}
	regs, games, admins := newGameAuthzFixtures(fixtureGameHostID)
	svc := app.NewService(app.ServiceOptions{
		Payments:            repo,
		IDs:                 &fixedIDs{ids: []string{fixtureNoShowFeePaymentID}},
		RegistrationUpdater: updater,
		RegistrationLookup:  regs,
		GameLookup:          games,
		GameAdminReader:     admins,
	})

	if _, err := svc.RecordOfflinePayment(context.Background(), app.RecordOfflinePaymentInput{
		PayableType: domain.PayableTypeNoShowFee,
		PayableID:   fixtureRegistrationID,
		Amount:      offlineFixtureAmount(),
		ActorUserID: fixtureGameHostID,
	}); err != nil {
		t.Fatalf("seed: RecordOfflinePayment: %v", err)
	}

	// Recording the fee must not have projected either — proving the
	// assertion below is about the refund and not inherited from a clean
	// slate that was never dirty.
	if len(updater.calls) != 0 {
		t.Fatalf("recording a no-show fee projected %d status update(s), want 0", len(updater.calls))
	}

	if _, err := svc.RefundPayment(context.Background(), app.RefundPaymentInput{
		PaymentID:   fixtureNoShowFeePaymentID,
		ActorUserID: fixtureGameHostID,
	}); err != nil {
		t.Fatalf("RefundPayment: %v", err)
	}

	if len(updater.calls) != 0 {
		t.Fatalf("refunding a no-show fee projected %d status update(s) to the Registration, want 0 — "+
			"a no-show fee is a separate charge, not the Registration's own payment status (calls: %+v)",
			len(updater.calls), updater.calls)
	}
}

// The gate is a whitelist, not a passthrough. With all four recognised
// payable types now in scope, this is what stops RefundPayment silently
// admitting a fifth one added later: a new payable type must opt in
// deliberately, exactly as no_show_fee and competition_entry each had to.
//
// The bogus Payment is written straight into the repository because
// domain.NewPayment validates PayableType — which is the point: this
// asserts the refund gate's own behaviour, not the constructor's.
func TestRefundPayment_UnrecognisedPayableTypeStillRejected(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	regs, games, admins := newGameAuthzFixtures(fixtureGameHostID)
	svc := app.NewService(app.ServiceOptions{
		Payments:           repo,
		IDs:                &fixedIDs{ids: []string{fixtureNoShowFeePaymentID}},
		RegistrationLookup: regs,
		GameLookup:         games,
		GameAdminReader:    admins,
	})

	bogus := domain.Payment{
		ID:               fixtureNoShowFeePaymentID,
		PayableType:      domain.PayableType("season_pass"),
		PayableID:        fixtureRegistrationID,
		Amount:           offlineFixtureAmount(),
		Method:           domain.MethodOffline,
		Status:           domain.StatusPaid,
		RecordedByUserID: fixtureGameHostID,
	}
	if _, err := repo.Create(context.Background(), bogus); err != nil {
		t.Fatalf("seed bogus payment: %v", err)
	}

	_, err := svc.RefundPayment(context.Background(), app.RefundPaymentInput{
		PaymentID:   fixtureNoShowFeePaymentID,
		ActorUserID: fixtureGameHostID,
	})
	if !errors.Is(err, domain.ErrInvalidPayableType) {
		t.Fatalf("RefundPayment(unrecognised payable type) = %v, want %v", err, domain.ErrInvalidPayableType)
	}

	stored, getErr := repo.GetByID(context.Background(), fixtureNoShowFeePaymentID)
	if getErr != nil {
		t.Fatalf("unexpected err: %v", getErr)
	}
	if stored.Status != domain.StatusPaid {
		t.Fatalf("persisted Status = %v, want paid — a rejected refund must not transition anything", stored.Status)
	}
}
