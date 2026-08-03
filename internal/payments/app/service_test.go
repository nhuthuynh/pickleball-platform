package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nhuthuynh/white-label/internal/payments/adapter/stripestub"
	"github.com/nhuthuynh/white-label/internal/payments/app"
	"github.com/nhuthuynh/white-label/internal/payments/domain"
)

func fixtureAmount() domain.Money {
	return domain.Money{Cents: 3000, Currency: "USD"}
}

func offlineFixtureAmount() domain.Money {
	return domain.Money{Cents: 2500, Currency: "USD"}
}

// --- T6.2: online path (stripestub processor) -----------------------------

// TestCreateOnlinePayment_Succeeds proves the happy path: an online
// Payment is built (unpaid), an intent is created against the stub
// processor, the returned intent reference is stored as StripeReference —
// while the Payment itself stays unpaid, because creating an intent is not
// the same as capturing funds — and the result is persisted (T6.4).
func TestCreateOnlinePayment_Succeeds(t *testing.T) {
	t.Parallel()

	proc := stripestub.NewProcessor()
	repo := newFakeRepository()
	svc := app.NewService(app.ServiceOptions{
		Payments:  repo,
		IDs:       &fixedIDs{ids: []string{"pay-1"}},
		Processor: proc,
	})

	p, err := svc.CreateOnlinePayment(context.Background(), app.CreateOnlinePaymentInput{
		PayableType: domain.PayableTypeBooking,
		PayableID:   "booking-1",
		Amount:      fixtureAmount(),
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if p.ID != "pay-1" {
		t.Fatalf("ID = %q, want pay-1", p.ID)
	}
	if p.Status != domain.StatusUnpaid {
		t.Fatalf("Status = %v, want unpaid", p.Status)
	}
	if p.Method != domain.MethodOnline {
		t.Fatalf("Method = %v, want online", p.Method)
	}
	if p.StripeReference == "" {
		t.Fatal("StripeReference must be populated from CreateIntent")
	}

	stored, err := repo.GetByID(context.Background(), "pay-1")
	if err != nil {
		t.Fatalf("expected the Payment to be persisted: %v", err)
	}
	if stored != p {
		t.Fatalf("persisted Payment = %+v, want %+v", stored, p)
	}
}

// TestCreateOnlinePayment_InvalidInputRejectedByDomain proves
// CreateOnlinePayment delegates validation to domain.NewPayment rather
// than duplicating it — an invalid amount is rejected before the
// processor is ever called.
func TestCreateOnlinePayment_InvalidInputRejectedByDomain(t *testing.T) {
	t.Parallel()

	proc := stripestub.NewProcessor()
	svc := app.NewService(app.ServiceOptions{
		Payments:  newFakeRepository(),
		IDs:       &fixedIDs{ids: []string{"pay-1"}},
		Processor: proc,
	})

	_, err := svc.CreateOnlinePayment(context.Background(), app.CreateOnlinePaymentInput{
		PayableType: domain.PayableTypeBooking,
		PayableID:   "booking-1",
		Amount:      domain.Money{Cents: 0, Currency: "USD"},
	})
	if !errors.Is(err, domain.ErrInvalidAmount) {
		t.Fatalf("got err %v, want %v", err, domain.ErrInvalidAmount)
	}
}

// TestCreateOnlinePayment_ProcessorUnavailable proves a processor failure
// at intent-creation time surfaces as the port's sentinel, not a panic or
// a partially-built Payment being returned (or persisted) as if it
// succeeded.
func TestCreateOnlinePayment_ProcessorUnavailable(t *testing.T) {
	t.Parallel()

	proc := stripestub.NewProcessor()
	proc.SeedUnavailable("booking-1")
	repo := newFakeRepository()
	svc := app.NewService(app.ServiceOptions{
		Payments:  repo,
		IDs:       &fixedIDs{ids: []string{"pay-1"}},
		Processor: proc,
	})

	_, err := svc.CreateOnlinePayment(context.Background(), app.CreateOnlinePaymentInput{
		PayableType: domain.PayableTypeBooking,
		PayableID:   "booking-1",
		Amount:      fixtureAmount(),
	})
	if !errors.Is(err, domain.ErrPaymentProcessorUnavailable) {
		t.Fatalf("got err %v, want %v", err, domain.ErrPaymentProcessorUnavailable)
	}
	if _, err := repo.GetByID(context.Background(), "pay-1"); !errors.Is(err, domain.ErrPaymentNotFound) {
		t.Fatalf("expected nothing persisted on processor failure, got err %v", err)
	}
}

// TestConfirmOnlinePayment_Succeeds proves the happy path: capturing a
// created intent transitions the Payment to paid via MarkPaid, and the
// transition is persisted (T6.4).
func TestConfirmOnlinePayment_Succeeds(t *testing.T) {
	t.Parallel()

	proc := stripestub.NewProcessor()
	repo := newFakeRepository()
	svc := app.NewService(app.ServiceOptions{
		Payments:  repo,
		IDs:       &fixedIDs{ids: []string{"pay-1"}},
		Processor: proc,
	})

	p, err := svc.CreateOnlinePayment(context.Background(), app.CreateOnlinePaymentInput{
		PayableType: domain.PayableTypeBooking,
		PayableID:   "booking-1",
		Amount:      fixtureAmount(),
	})
	if err != nil {
		t.Fatalf("unexpected err creating: %v", err)
	}

	confirmed, err := svc.ConfirmOnlinePayment(context.Background(), p)
	if err != nil {
		t.Fatalf("unexpected err confirming: %v", err)
	}
	if confirmed.Status != domain.StatusPaid {
		t.Fatalf("Status = %v, want paid", confirmed.Status)
	}
	if confirmed.StripeReference != p.StripeReference {
		t.Fatalf("StripeReference = %q, want unchanged %q", confirmed.StripeReference, p.StripeReference)
	}

	stored, err := repo.GetByID(context.Background(), "pay-1")
	if err != nil {
		t.Fatalf("expected the Payment to still be persisted: %v", err)
	}
	if stored.Status != domain.StatusPaid {
		t.Fatalf("persisted Status = %v, want paid", stored.Status)
	}
}

// TestConfirmOnlinePayment_Declined is the ticket's required test: a
// declined card must leave the Payment unchanged (still unpaid), not
// error out as an illegal state transition, and must not persist any
// change. This is the case most likely to be implemented wrong by reaching
// for MarkPaid's error path instead of just not calling MarkPaid at all.
func TestConfirmOnlinePayment_Declined(t *testing.T) {
	t.Parallel()

	proc := stripestub.NewProcessor()
	proc.SeedDecline("booking-1")
	repo := newFakeRepository()
	svc := app.NewService(app.ServiceOptions{
		Payments:  repo,
		IDs:       &fixedIDs{ids: []string{"pay-1"}},
		Processor: proc,
	})

	p, err := svc.CreateOnlinePayment(context.Background(), app.CreateOnlinePaymentInput{
		PayableType: domain.PayableTypeBooking,
		PayableID:   "booking-1",
		Amount:      fixtureAmount(),
	})
	if err != nil {
		t.Fatalf("unexpected err creating: %v", err)
	}

	before := p

	result, err := svc.ConfirmOnlinePayment(context.Background(), p)
	if !errors.Is(err, domain.ErrPaymentDeclined) {
		t.Fatalf("got err %v, want %v", err, domain.ErrPaymentDeclined)
	}
	if result.Status != domain.StatusUnpaid {
		t.Fatalf("Status = %v, want unchanged (unpaid)", result.Status)
	}
	if result != before {
		t.Fatalf("Payment mutated on decline: got %+v, want unchanged %+v", result, before)
	}

	stored, err := repo.GetByID(context.Background(), "pay-1")
	if err != nil {
		t.Fatalf("unexpected err reloading: %v", err)
	}
	if stored.Status != domain.StatusUnpaid {
		t.Fatalf("persisted Status = %v, want unchanged (unpaid)", stored.Status)
	}
}

// TestConfirmOnlinePayment_ProcessorUnavailable proves a processor-level
// failure (distinct from a decline) also leaves the Payment unchanged and
// unpersisted, not just the decline path.
func TestConfirmOnlinePayment_ProcessorUnavailable(t *testing.T) {
	t.Parallel()

	proc := stripestub.NewProcessor()
	repo := newFakeRepository()
	svc := app.NewService(app.ServiceOptions{
		Payments:  repo,
		IDs:       &fixedIDs{ids: []string{"pay-1"}},
		Processor: proc,
	})

	p, err := svc.CreateOnlinePayment(context.Background(), app.CreateOnlinePaymentInput{
		PayableType: domain.PayableTypeBooking,
		PayableID:   "booking-1",
		Amount:      fixtureAmount(),
	})
	if err != nil {
		t.Fatalf("unexpected err creating: %v", err)
	}
	// Corrupt the reference so the stub treats it as unknown, simulating
	// a processor-side failure distinct from a decline.
	p.StripeReference = "pi_does_not_exist"

	result, err := svc.ConfirmOnlinePayment(context.Background(), p)
	if !errors.Is(err, domain.ErrPaymentProcessorUnavailable) {
		t.Fatalf("got err %v, want %v", err, domain.ErrPaymentProcessorUnavailable)
	}
	if result.Status != domain.StatusUnpaid {
		t.Fatalf("Status = %v, want unchanged (unpaid)", result.Status)
	}
}

// TestConfirmOnlinePayment_AlreadyPaidIsIllegal proves ConfirmOnlinePayment
// still routes through Payment.MarkPaid's own transition guard for a
// Payment that is not unpaid — confirming an already-paid Payment again is
// an illegal transition, not a silent no-op or a second successful
// capture.
func TestConfirmOnlinePayment_AlreadyPaidIsIllegal(t *testing.T) {
	t.Parallel()

	proc := stripestub.NewProcessor()
	svc := app.NewService(app.ServiceOptions{
		Payments:  newFakeRepository(),
		IDs:       &fixedIDs{ids: []string{"pay-1"}},
		Processor: proc,
	})

	p, err := svc.CreateOnlinePayment(context.Background(), app.CreateOnlinePaymentInput{
		PayableType: domain.PayableTypeBooking,
		PayableID:   "booking-1",
		Amount:      fixtureAmount(),
	})
	if err != nil {
		t.Fatalf("unexpected err creating: %v", err)
	}
	paid, err := svc.ConfirmOnlinePayment(context.Background(), p)
	if err != nil {
		t.Fatalf("unexpected err confirming: %v", err)
	}

	_, err = svc.ConfirmOnlinePayment(context.Background(), paid)
	if !errors.Is(err, domain.ErrIllegalStatusTransition) {
		t.Fatalf("got err %v, want %v", err, domain.ErrIllegalStatusTransition)
	}
}

// --- T6.3: offline recording path ------------------------------------------

// TestRecordOfflinePayment_BookingPayable_HostSucceeds proves the happy
// path for a booking-payable: the actor matching the caller-supplied
// BookingHostID may record the offline payment, and the result is built
// via domain.NewPayment (Method: offline, RecordedByUserID: ActorUserID)
// and immediately transitioned to paid — an offline recording *is* the
// payment event, there is no separate intent step — and persisted (T6.4).
func TestRecordOfflinePayment_BookingPayable_HostSucceeds(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	svc := app.NewService(app.ServiceOptions{
		Payments: repo,
		IDs:      &fixedIDs{ids: []string{"pay-1"}},
	})

	p, err := svc.RecordOfflinePayment(context.Background(), app.RecordOfflinePaymentInput{
		PayableType:   domain.PayableTypeBooking,
		PayableID:     "booking-1",
		Amount:        offlineFixtureAmount(),
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

	if _, err := repo.GetByID(context.Background(), "pay-1"); err != nil {
		t.Fatalf("expected the Payment to be persisted: %v", err)
	}
}

// TestRecordOfflinePayment_BookingPayable_WrongActorRejected proves a user
// who is not the Booking's owning Host cannot record the offline payment —
// ErrNotPaymentRecorder, not a silent success and not
// ErrIllegalStatusTransition.
func TestRecordOfflinePayment_BookingPayable_WrongActorRejected(t *testing.T) {
	t.Parallel()

	svc := app.NewService(app.ServiceOptions{
		Payments: newFakeRepository(),
		IDs:      &fixedIDs{ids: []string{"pay-1"}},
	})

	_, err := svc.RecordOfflinePayment(context.Background(), app.RecordOfflinePaymentInput{
		PayableType:   domain.PayableTypeBooking,
		PayableID:     "booking-1",
		Amount:        offlineFixtureAmount(),
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

	svc := app.NewService(app.ServiceOptions{
		Payments: newFakeRepository(),
		IDs:      &fixedIDs{ids: []string{"pay-1"}},
	})

	_, err := svc.RecordOfflinePayment(context.Background(), app.RecordOfflinePaymentInput{
		PayableType:   domain.PayableTypeBooking,
		PayableID:     "booking-1",
		Amount:        offlineFixtureAmount(),
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

	svc := app.NewService(app.ServiceOptions{
		Payments: newFakeRepository(),
		IDs:      &fixedIDs{ids: []string{"pay-1"}},
	})

	p, err := svc.RecordOfflinePayment(context.Background(), app.RecordOfflinePaymentInput{
		PayableType: domain.PayableTypeRegistration,
		PayableID:   "reg-1",
		Amount:      offlineFixtureAmount(),
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

	svc := app.NewService(app.ServiceOptions{
		Payments: newFakeRepository(),
		IDs:      &fixedIDs{ids: []string{"pay-1"}},
	})

	p, err := svc.RecordOfflinePayment(context.Background(), app.RecordOfflinePaymentInput{
		PayableType:              domain.PayableTypeRegistration,
		PayableID:                "reg-1",
		Amount:                   offlineFixtureAmount(),
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

	svc := app.NewService(app.ServiceOptions{
		Payments: newFakeRepository(),
		IDs:      &fixedIDs{ids: []string{"pay-1"}},
	})

	_, err := svc.RecordOfflinePayment(context.Background(), app.RecordOfflinePaymentInput{
		PayableType:              domain.PayableTypeRegistration,
		PayableID:                "reg-1",
		Amount:                   offlineFixtureAmount(),
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

	svc := app.NewService(app.ServiceOptions{
		Payments: newFakeRepository(),
		IDs:      &fixedIDs{ids: []string{"pay-1"}},
	})

	p, err := svc.RecordOfflinePayment(context.Background(), app.RecordOfflinePaymentInput{
		PayableType:              domain.PayableTypeNoShowFee,
		PayableID:                "reg-1",
		Amount:                   offlineFixtureAmount(),
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

	svc := app.NewService(app.ServiceOptions{
		Payments: newFakeRepository(),
		IDs:      &fixedIDs{ids: []string{"pay-1"}},
	})

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

	svc := app.NewService(app.ServiceOptions{
		Payments: newFakeRepository(),
		IDs:      &fixedIDs{ids: []string{"pay-1"}},
	})

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

// --- T6.4: persistence (duplicate-recording AC) ----------------------------

// TestRecordOfflinePayment_DuplicateForSamePayableRejected proves the T6.4
// AC directly at the app layer using fakeRepository's distinctness check
// (standing in for the Postgres UNIQUE (payable_type, payable_id)
// constraint): a second offline payment for the same payable action is
// rejected with domain.ErrPaymentAlreadyRecorded, not silently accepted as
// a second row.
func TestRecordOfflinePayment_DuplicateForSamePayableRejected(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	svc := app.NewService(app.ServiceOptions{
		Payments: repo,
		IDs:      &fixedIDs{ids: []string{"pay-1", "pay-2"}},
	})

	in := app.RecordOfflinePaymentInput{
		PayableType: domain.PayableTypeRegistration,
		PayableID:   "reg-1",
		Amount:      offlineFixtureAmount(),
		ActorUserID: "host-1",
		GameHostID:  "host-1",
	}

	if _, err := svc.RecordOfflinePayment(context.Background(), in); err != nil {
		t.Fatalf("unexpected err on first recording: %v", err)
	}

	_, err := svc.RecordOfflinePayment(context.Background(), in)
	if !errors.Is(err, domain.ErrPaymentAlreadyRecorded) {
		t.Fatalf("got err %v, want %v", err, domain.ErrPaymentAlreadyRecorded)
	}
}
