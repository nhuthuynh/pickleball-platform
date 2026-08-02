package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nhuthuynh/white-label/internal/payments/app"
	"github.com/nhuthuynh/white-label/internal/payments/domain"
)

func fixtureAmount() domain.Money {
	return domain.Money{Cents: 2500, Currency: "USD"}
}

// TestRecordOfflinePayment_BookingPayable_HostSucceeds proves the happy
// path for a booking-payable: the actor matching the caller-supplied
// BookingHostID may record the offline payment, and the result is built
// via domain.NewPayment (Method: offline, RecordedByUserID: ActorUserID)
// and immediately transitioned to paid — an offline recording *is* the
// payment event, there is no separate intent step.
func TestRecordOfflinePayment_BookingPayable_HostSucceeds(t *testing.T) {
	t.Parallel()

	svc := app.NewService(&fixedIDs{ids: []string{"pay-1"}})

	p, err := svc.RecordOfflinePayment(context.Background(), app.RecordOfflinePaymentInput{
		PayableType:   domain.PayableTypeBooking,
		PayableID:     "booking-1",
		Amount:        fixtureAmount(),
		ActorUserID:   "host-1",
		BookingHostID: "host-1",
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if p.ID != "pay-1" {
		t.Fatalf("ID = %q, want pay-1", p.ID)
	}
	if p.PayableType != domain.PayableTypeBooking {
		t.Fatalf("PayableType = %v, want booking", p.PayableType)
	}
	if p.Method != domain.MethodOffline {
		t.Fatalf("Method = %v, want offline", p.Method)
	}
	if p.Status != domain.StatusPaid {
		t.Fatalf("Status = %v, want paid", p.Status)
	}
	if p.RecordedByUserID != "host-1" {
		t.Fatalf("RecordedByUserID = %q, want host-1", p.RecordedByUserID)
	}
}

// TestRecordOfflinePayment_BookingPayable_WrongActorRejected proves a user
// who is not the Booking's owning Host cannot record the offline payment —
// ErrNotPaymentRecorder, not a silent success and not
// ErrIllegalStatusTransition.
func TestRecordOfflinePayment_BookingPayable_WrongActorRejected(t *testing.T) {
	t.Parallel()

	svc := app.NewService(&fixedIDs{ids: []string{"pay-1"}})

	_, err := svc.RecordOfflinePayment(context.Background(), app.RecordOfflinePaymentInput{
		PayableType:   domain.PayableTypeBooking,
		PayableID:     "booking-1",
		Amount:        fixtureAmount(),
		ActorUserID:   "some-other-player",
		BookingHostID: "host-1",
	})
	if !errors.Is(err, domain.ErrNotPaymentRecorder) {
		t.Fatalf("got err %v, want %v", err, domain.ErrNotPaymentRecorder)
	}
}

// TestRecordOfflinePayment_BookingPayable_NoHostRejected proves the T6.3
// scope cut: a booking-payable with no owning Host (a direct
// individual/recurring-hire court hire, per BookingHostID left empty) is
// out of scope for offline recording in T6 and is rejected, not silently
// authorized to whoever happens to call it.
func TestRecordOfflinePayment_BookingPayable_NoHostRejected(t *testing.T) {
	t.Parallel()

	svc := app.NewService(&fixedIDs{ids: []string{"pay-1"}})

	_, err := svc.RecordOfflinePayment(context.Background(), app.RecordOfflinePaymentInput{
		PayableType:   domain.PayableTypeBooking,
		PayableID:     "booking-1",
		Amount:        fixtureAmount(),
		ActorUserID:   "",
		BookingHostID: "",
	})
	if !errors.Is(err, domain.ErrNotPaymentRecorder) {
		t.Fatalf("got err %v, want %v", err, domain.ErrNotPaymentRecorder)
	}
}

// TestRecordOfflinePayment_RegistrationPayable_GameHostSucceeds proves a
// registration-payable can be recorded by the Registration's Game's Host.
func TestRecordOfflinePayment_RegistrationPayable_GameHostSucceeds(t *testing.T) {
	t.Parallel()

	svc := app.NewService(&fixedIDs{ids: []string{"pay-1"}})

	p, err := svc.RecordOfflinePayment(context.Background(), app.RecordOfflinePaymentInput{
		PayableType: domain.PayableTypeRegistration,
		PayableID:   "reg-1",
		Amount:      fixtureAmount(),
		ActorUserID: "host-1",
		GameHostID:  "host-1",
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if p.Status != domain.StatusPaid {
		t.Fatalf("Status = %v, want paid", p.Status)
	}
}

// TestRecordOfflinePayment_RegistrationPayable_AssignedGameAdminSucceeds
// proves the glossary's Game Admin scope (agent-operating-handbook.md
// A2): a Game Admin assigned to the specific Game may record an offline
// payment against one of its Registrations, even though they are not the
// Host.
func TestRecordOfflinePayment_RegistrationPayable_AssignedGameAdminSucceeds(t *testing.T) {
	t.Parallel()

	svc := app.NewService(&fixedIDs{ids: []string{"pay-1"}})

	p, err := svc.RecordOfflinePayment(context.Background(), app.RecordOfflinePaymentInput{
		PayableType:              domain.PayableTypeRegistration,
		PayableID:                "reg-1",
		Amount:                   fixtureAmount(),
		ActorUserID:              "admin-2",
		GameHostID:               "host-1",
		AssignedGameAdminUserIDs: []string{"admin-1", "admin-2"},
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if p.RecordedByUserID != "admin-2" {
		t.Fatalf("RecordedByUserID = %q, want admin-2", p.RecordedByUserID)
	}
}

// TestRecordOfflinePayment_RegistrationPayable_UnassignedActorRejected
// proves a Player who is neither the Game's Host nor an assigned Game
// Admin cannot record an offline payment against a Registration —
// ErrNotPaymentRecorder.
func TestRecordOfflinePayment_RegistrationPayable_UnassignedActorRejected(t *testing.T) {
	t.Parallel()

	svc := app.NewService(&fixedIDs{ids: []string{"pay-1"}})

	_, err := svc.RecordOfflinePayment(context.Background(), app.RecordOfflinePaymentInput{
		PayableType:              domain.PayableTypeRegistration,
		PayableID:                "reg-1",
		Amount:                   fixtureAmount(),
		ActorUserID:              "random-player",
		GameHostID:               "host-1",
		AssignedGameAdminUserIDs: []string{"admin-1"},
	})
	if !errors.Is(err, domain.ErrNotPaymentRecorder) {
		t.Fatalf("got err %v, want %v", err, domain.ErrNotPaymentRecorder)
	}
}

// TestRecordOfflinePayment_NoShowFee_GameAdmin_SameCodePath is the ticket's
// required test (P1 #8): a Game Admin records a PayableType: no_show_fee
// payment against a Registration through the exact same
// RecordOfflinePayment function as an ordinary registration payment — same
// code path, only PayableType differs — proving the extensible-enum design
// from T6.1 pays off with zero new code paths for the manual no-show-fee
// recording case.
func TestRecordOfflinePayment_NoShowFee_GameAdmin_SameCodePath(t *testing.T) {
	t.Parallel()

	svc := app.NewService(&fixedIDs{ids: []string{"pay-1"}})

	p, err := svc.RecordOfflinePayment(context.Background(), app.RecordOfflinePaymentInput{
		PayableType:              domain.PayableTypeNoShowFee,
		PayableID:                "reg-1",
		Amount:                   fixtureAmount(),
		ActorUserID:              "admin-2",
		GameHostID:               "host-1",
		AssignedGameAdminUserIDs: []string{"admin-1", "admin-2"},
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if p.PayableType != domain.PayableTypeNoShowFee {
		t.Fatalf("PayableType = %v, want no_show_fee", p.PayableType)
	}
	if p.Status != domain.StatusPaid {
		t.Fatalf("Status = %v, want paid", p.Status)
	}
	if p.RecordedByUserID != "admin-2" {
		t.Fatalf("RecordedByUserID = %q, want admin-2", p.RecordedByUserID)
	}
}

// TestRecordOfflinePayment_InvalidAmountRejectedByDomain proves
// RecordOfflinePayment delegates amount/payable validation to
// domain.NewPayment rather than duplicating it — an authorized actor with
// an invalid amount is still rejected, by the domain sentinel.
func TestRecordOfflinePayment_InvalidAmountRejectedByDomain(t *testing.T) {
	t.Parallel()

	svc := app.NewService(&fixedIDs{ids: []string{"pay-1"}})

	_, err := svc.RecordOfflinePayment(context.Background(), app.RecordOfflinePaymentInput{
		PayableType:   domain.PayableTypeBooking,
		PayableID:     "booking-1",
		Amount:        domain.Money{Cents: 0, Currency: "USD"},
		ActorUserID:   "host-1",
		BookingHostID: "host-1",
	})
	if !errors.Is(err, domain.ErrInvalidAmount) {
		t.Fatalf("got err %v, want %v", err, domain.ErrInvalidAmount)
	}
}

// TestRecordOfflinePayment_AuthorizationCheckedBeforeDomainConstruction
// proves authorization is checked even when other input would also be
// invalid — an unauthorized actor gets ErrNotPaymentRecorder, not a
// domain validation error that would leak which fields are wrong to a
// caller who was never allowed to attempt the call.
func TestRecordOfflinePayment_AuthorizationCheckedBeforeDomainConstruction(t *testing.T) {
	t.Parallel()

	svc := app.NewService(&fixedIDs{ids: []string{"pay-1"}})

	_, err := svc.RecordOfflinePayment(context.Background(), app.RecordOfflinePaymentInput{
		PayableType:   domain.PayableTypeBooking,
		PayableID:     "booking-1",
		Amount:        domain.Money{Cents: 0, Currency: "USD"},
		ActorUserID:   "wrong-actor",
		BookingHostID: "host-1",
	})
	if !errors.Is(err, domain.ErrNotPaymentRecorder) {
		t.Fatalf("got err %v, want %v", err, domain.ErrNotPaymentRecorder)
	}
}
