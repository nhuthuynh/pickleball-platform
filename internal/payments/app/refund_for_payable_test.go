package app_test

import (
	"context"
	"testing"

	"github.com/nhuthuynh/white-label/internal/payments/app"
	"github.com/nhuthuynh/white-label/internal/payments/domain"
)

// T55.3 (issue #124's refund half) — RefundForPayable is the entry point a
// cancellation cascade uses: it knows which Registration it just cancelled,
// not which Payment paid for it.
//
// The load-bearing behaviour here is not the happy path (RefundPayment
// already has thorough coverage) but the two "nothing to do" answers. A
// cancelled Game's roster is a mix of paid, unpaid and already-refunded
// registrations, and a cascade that errored on the last two would fail on
// almost every real cancellation.

func refundForPayableService(repo *fakeRepository, paymentID string) *app.Service {
	regs, games, admins := newGameAuthzFixtures(fixtureGameHostID)
	return app.NewService(app.ServiceOptions{
		Payments:           repo,
		IDs:                &fixedIDs{ids: []string{paymentID}},
		RegistrationLookup: regs,
		GameLookup:         games,
		GameAdminReader:    admins,
	})
}

func TestRefundForPayable_RefundsAPaidRegistration(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	svc := refundForPayableService(repo, fixtureRegistrationPaymentID)

	if _, err := svc.RecordOfflinePayment(context.Background(), app.RecordOfflinePaymentInput{
		PayableType: domain.PayableTypeRegistration,
		PayableID:   fixtureRegistrationID,
		Amount:      offlineFixtureAmount(),
		ActorUserID: fixtureGameHostID,
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	refunded, err := svc.RefundForPayable(context.Background(), app.RefundForPayableInput{
		PayableType: domain.PayableTypeRegistration,
		PayableID:   fixtureRegistrationID,
		ActorUserID: fixtureGameHostID,
	})
	if err != nil {
		t.Fatalf("RefundForPayable: %v", err)
	}
	if !refunded {
		t.Fatal("refunded = false, want true")
	}

	// Persisted, not just reported.
	stored, err := repo.GetByID(context.Background(), fixtureRegistrationPaymentID)
	if err != nil {
		t.Fatalf("payment should still exist: %v", err)
	}
	if stored.Status != domain.StatusRefunded {
		t.Fatalf("persisted Status = %v, want refunded", stored.Status)
	}
}

// An unpaid Registration — no Payment row at all — is the commonest case on
// a cancelled Game's roster and must be a silent no-op.
func TestRefundForPayable_NoPaymentIsANoOpNotAnError(t *testing.T) {
	t.Parallel()

	svc := refundForPayableService(newFakeRepository(), fixtureRegistrationPaymentID)

	refunded, err := svc.RefundForPayable(context.Background(), app.RefundForPayableInput{
		PayableType: domain.PayableTypeRegistration,
		PayableID:   fixtureRegistrationID,
		ActorUserID: fixtureGameHostID,
	})
	if err != nil {
		t.Fatalf("RefundForPayable with no payment = %v, want nil", err)
	}
	if refunded {
		t.Fatal("refunded = true, want false — there was nothing to refund")
	}
}

// An already-refunded Payment must also be a no-op, so re-running a cascade
// (a retry, a repair sweep) neither double-refunds nor fails.
func TestRefundForPayable_AlreadyRefundedIsANoOpNotAnError(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	svc := refundForPayableService(repo, fixtureRegistrationPaymentID)

	if _, err := svc.RecordOfflinePayment(context.Background(), app.RecordOfflinePaymentInput{
		PayableType: domain.PayableTypeRegistration,
		PayableID:   fixtureRegistrationID,
		Amount:      offlineFixtureAmount(),
		ActorUserID: fixtureGameHostID,
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	in := app.RefundForPayableInput{
		PayableType: domain.PayableTypeRegistration,
		PayableID:   fixtureRegistrationID,
		ActorUserID: fixtureGameHostID,
	}

	if refunded, err := svc.RefundForPayable(context.Background(), in); err != nil || !refunded {
		t.Fatalf("first refund = (%v, %v), want (true, nil)", refunded, err)
	}

	refunded, err := svc.RefundForPayable(context.Background(), in)
	if err != nil {
		t.Fatalf("second refund = %v, want nil (idempotent)", err)
	}
	if refunded {
		t.Fatal("second refund reported true, want false — it was already refunded")
	}
}

// Authorization is NOT relaxed by going through this entry point: a caller
// who could not refund the Payment directly cannot refund it via the
// payable either. Without this, the cascade would be a privilege-escalation
// path around authorizeOfflineRecording.
func TestRefundForPayable_DoesNotBypassAuthorization(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	svc := refundForPayableService(repo, fixtureRegistrationPaymentID)

	if _, err := svc.RecordOfflinePayment(context.Background(), app.RecordOfflinePaymentInput{
		PayableType: domain.PayableTypeRegistration,
		PayableID:   fixtureRegistrationID,
		Amount:      offlineFixtureAmount(),
		ActorUserID: fixtureGameHostID,
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	refunded, err := svc.RefundForPayable(context.Background(), app.RefundForPayableInput{
		PayableType: domain.PayableTypeRegistration,
		PayableID:   fixtureRegistrationID,
		ActorUserID: "not-the-host-or-an-admin",
	})
	if err == nil {
		t.Fatal("an unauthorized actor was allowed to refund via the payable path")
	}
	if refunded {
		t.Fatal("refunded = true for an unauthorized actor")
	}

	// And nothing changed.
	stored, err := repo.GetByID(context.Background(), fixtureRegistrationPaymentID)
	if err != nil {
		t.Fatalf("payment should still exist: %v", err)
	}
	if stored.Status != domain.StatusPaid {
		t.Fatalf("persisted Status = %v, want paid — the refused refund must not have persisted", stored.Status)
	}
}
