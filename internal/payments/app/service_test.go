package app_test

import (
	"context"
	"errors"
	"testing"

	competitionsdomain "github.com/nhuthuynh/white-label/internal/competitions/domain"
	"github.com/nhuthuynh/white-label/internal/payments/adapter/stripestub"
	"github.com/nhuthuynh/white-label/internal/payments/app"
	"github.com/nhuthuynh/white-label/internal/payments/domain"
	socialplaydomain "github.com/nhuthuynh/white-label/internal/socialplay/domain"
)

// fixtureBookingID/fixtureRegistrationID are deterministic, UUID-shaped
// PayableID fixtures. They used to be the literals "booking-1"/"reg-1" — a
// shape internal/platform/idgen never produces and the Postgres adapter's
// mustUUID panics on (db/migrations/0005_payments.sql's payable_id column is
// `uuid NOT NULL`, un-FK'd but still typed). T10.7 (closing issue #97) adds a
// uuidShape guard on PayableID for exactly the class of bug LESSONS.md's T9
// entry describes ("the test fixtures made the bug invisible"): every
// existing test in this file used a PayableID no real request could ever
// send, which would have made the new guard's own regression tests the only
// ones exercising a realistic value. Renamed here instead, mirroring how
// internal/booking, internal/competitions, and internal/facilities app
// tests already fixed the identical fixture infidelity on their own IDs.
const (
	fixtureBookingID          = "6ba7b810-0000-4000-8000-000000000001"
	fixtureRegistrationID     = "6ba7b810-0000-4000-8000-000000000002"
	fixtureCompetitionEntryID = "6ba7b810-0000-4000-8000-000000000003"
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
		PayableID:   fixtureBookingID,
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
		PayableID:   fixtureBookingID,
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
	proc.SeedUnavailable(fixtureBookingID)
	repo := newFakeRepository()
	svc := app.NewService(app.ServiceOptions{
		Payments:  repo,
		IDs:       &fixedIDs{ids: []string{"pay-1"}},
		Processor: proc,
	})

	_, err := svc.CreateOnlinePayment(context.Background(), app.CreateOnlinePaymentInput{
		PayableType: domain.PayableTypeBooking,
		PayableID:   fixtureBookingID,
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
		PayableID:   fixtureBookingID,
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
	proc.SeedDecline(fixtureBookingID)
	repo := newFakeRepository()
	svc := app.NewService(app.ServiceOptions{
		Payments:  repo,
		IDs:       &fixedIDs{ids: []string{"pay-1"}},
		Processor: proc,
	})

	p, err := svc.CreateOnlinePayment(context.Background(), app.CreateOnlinePaymentInput{
		PayableType: domain.PayableTypeBooking,
		PayableID:   fixtureBookingID,
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
		PayableID:   fixtureBookingID,
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
		PayableID:   fixtureBookingID,
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
		PayableID:     fixtureBookingID,
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
		PayableID:     fixtureBookingID,
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
		PayableID:     fixtureBookingID,
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
		PayableID:   fixtureRegistrationID,
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
		PayableID:                fixtureRegistrationID,
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
		PayableID:                fixtureRegistrationID,
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
		PayableID:                fixtureRegistrationID,
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
		PayableID:     fixtureBookingID,
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
		PayableID:     fixtureBookingID,
		Amount:        domain.Money{Cents: 0, Currency: "USD"},
		ActorUserID:   "wrong-actor",
		BookingHostID: "host-1",
	})
	if !errors.Is(err, domain.ErrNotPaymentRecorder) {
		t.Fatalf("got err %v, want %v", err, domain.ErrNotPaymentRecorder)
	}
}

// --- T10.6: competition_entry payable authorization (closes #96) ----------
//
// Mirrors the registration-payable authorization tests above, with one
// deliberate difference: for a competition_entry payable, the entrant
// themself (the CompetitionEntry's own PlayerID) is an authorized actor,
// not just a Host/assigned Admin — the ticket's own AC ("as a Player
// entering a Competition, I want to pay online"). An assigned Competition
// Admin may also record on the entrant's behalf, mirroring the
// Registration/no_show_fee branch's Host-or-Admin shape.

// TestRecordOfflinePayment_CompetitionEntryPayable_EntrantSucceeds proves
// the entrant may record their own competition entry payment offline.
func TestRecordOfflinePayment_CompetitionEntryPayable_EntrantSucceeds(t *testing.T) {
	t.Parallel()

	svc := app.NewService(app.ServiceOptions{
		Payments: newFakeRepository(),
		IDs:      &fixedIDs{ids: []string{"pay-1"}},
	})

	p, err := svc.RecordOfflinePayment(context.Background(), app.RecordOfflinePaymentInput{
		PayableType:     domain.PayableTypeCompetitionEntry,
		PayableID:       fixtureCompetitionEntryID,
		Amount:          offlineFixtureAmount(),
		ActorUserID:     "player-1",
		EntrantPlayerID: "player-1",
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if p.Status != domain.StatusPaid {
		t.Fatalf("Status = %v, want paid", p.Status)
	}
}

// TestRecordOfflinePayment_CompetitionEntryPayable_AssignedCompetitionAdminSucceeds
// proves a Competition Admin assigned to the specific Competition may record
// an offline payment against one of its entries on the entrant's behalf,
// even though they are not the entrant.
func TestRecordOfflinePayment_CompetitionEntryPayable_AssignedCompetitionAdminSucceeds(t *testing.T) {
	t.Parallel()

	svc := app.NewService(app.ServiceOptions{
		Payments: newFakeRepository(),
		IDs:      &fixedIDs{ids: []string{"pay-1"}},
	})

	p, err := svc.RecordOfflinePayment(context.Background(), app.RecordOfflinePaymentInput{
		PayableType:                     domain.PayableTypeCompetitionEntry,
		PayableID:                       fixtureCompetitionEntryID,
		Amount:                          offlineFixtureAmount(),
		ActorUserID:                     "admin-2",
		EntrantPlayerID:                 "player-1",
		AssignedCompetitionAdminUserIDs: []string{"admin-1", "admin-2"},
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if p.RecordedByUserID != "admin-2" {
		t.Fatalf("RecordedByUserID = %q, want admin-2", p.RecordedByUserID)
	}
}

// TestRecordOfflinePayment_CompetitionEntryPayable_UnauthorizedActorRejected
// proves an actor who is neither the entrant nor an assigned Competition
// Admin cannot record an offline payment against a CompetitionEntry —
// ErrNotPaymentRecorder (the ticket's required BOLA regression at the app
// layer; the handler-level proof lives in
// internal/payments/adapter/grpcapi).
func TestRecordOfflinePayment_CompetitionEntryPayable_UnauthorizedActorRejected(t *testing.T) {
	t.Parallel()

	svc := app.NewService(app.ServiceOptions{
		Payments: newFakeRepository(),
		IDs:      &fixedIDs{ids: []string{"pay-1"}},
	})

	_, err := svc.RecordOfflinePayment(context.Background(), app.RecordOfflinePaymentInput{
		PayableType:                     domain.PayableTypeCompetitionEntry,
		PayableID:                       fixtureCompetitionEntryID,
		Amount:                          offlineFixtureAmount(),
		ActorUserID:                     "random-player",
		EntrantPlayerID:                 "player-1",
		AssignedCompetitionAdminUserIDs: []string{"admin-1"},
	})
	if !errors.Is(err, domain.ErrNotPaymentRecorder) {
		t.Fatalf("got err %v, want %v", err, domain.ErrNotPaymentRecorder)
	}
}

// TestCreateOnlinePayment_CompetitionEntryPayable_EntrantSucceeds proves the
// online-checkout half of the same authorization rule: the entrant supplies
// actor_user_id claiming to be themself and CreateOnlinePayment succeeds.
func TestCreateOnlinePayment_CompetitionEntryPayable_EntrantSucceeds(t *testing.T) {
	t.Parallel()

	proc := stripestub.NewProcessor()
	repo := newFakeRepository()
	svc := app.NewService(app.ServiceOptions{
		Payments:  repo,
		IDs:       &fixedIDs{ids: []string{"pay-1"}},
		Processor: proc,
	})

	p, err := svc.CreateOnlinePayment(context.Background(), app.CreateOnlinePaymentInput{
		PayableType:     domain.PayableTypeCompetitionEntry,
		PayableID:       fixtureCompetitionEntryID,
		Amount:          fixtureAmount(),
		ActorUserID:     "player-1",
		EntrantPlayerID: "player-1",
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if p.Status != domain.StatusUnpaid {
		t.Fatalf("Status = %v, want unpaid (creating an intent does not capture funds)", p.Status)
	}
}

// TestCreateOnlinePayment_CompetitionEntryPayable_MissingActorRejected
// proves an empty actor_user_id is rejected outright for a competition_entry
// payable — a caller must claim to be someone before the online path will
// authorize a Payment against another Player's entry.
func TestCreateOnlinePayment_CompetitionEntryPayable_MissingActorRejected(t *testing.T) {
	t.Parallel()

	proc := stripestub.NewProcessor()
	svc := app.NewService(app.ServiceOptions{
		Payments:  newFakeRepository(),
		IDs:       &fixedIDs{ids: []string{"pay-1"}},
		Processor: proc,
	})

	_, err := svc.CreateOnlinePayment(context.Background(), app.CreateOnlinePaymentInput{
		PayableType:     domain.PayableTypeCompetitionEntry,
		PayableID:       fixtureCompetitionEntryID,
		Amount:          fixtureAmount(),
		EntrantPlayerID: "player-1",
	})
	if !errors.Is(err, domain.ErrNotPaymentRecorder) {
		t.Fatalf("got err %v, want %v", err, domain.ErrNotPaymentRecorder)
	}
}

// TestCreateOnlinePayment_CompetitionEntryPayable_UnauthorizedActorRejected
// proves an actor who is neither the entrant nor an assigned Competition
// Admin cannot start an online payment against another Player's
// CompetitionEntry — the ticket's required BOLA regression for the online
// path.
func TestCreateOnlinePayment_CompetitionEntryPayable_UnauthorizedActorRejected(t *testing.T) {
	t.Parallel()

	proc := stripestub.NewProcessor()
	repo := newFakeRepository()
	svc := app.NewService(app.ServiceOptions{
		Payments:  repo,
		IDs:       &fixedIDs{ids: []string{"pay-1"}},
		Processor: proc,
	})

	_, err := svc.CreateOnlinePayment(context.Background(), app.CreateOnlinePaymentInput{
		PayableType:                     domain.PayableTypeCompetitionEntry,
		PayableID:                       fixtureCompetitionEntryID,
		Amount:                          fixtureAmount(),
		ActorUserID:                     "random-player",
		EntrantPlayerID:                 "player-1",
		AssignedCompetitionAdminUserIDs: []string{"admin-1"},
	})
	if !errors.Is(err, domain.ErrNotPaymentRecorder) {
		t.Fatalf("got err %v, want %v", err, domain.ErrNotPaymentRecorder)
	}
	if _, err := repo.GetByID(context.Background(), "pay-1"); !errors.Is(err, domain.ErrPaymentNotFound) {
		t.Fatalf("expected nothing persisted on a rejected authorization attempt, got err %v", err)
	}
}

// TestCreateOnlinePayment_BookingPayable_NoActorFieldsRequired is a
// regression guard: the new competition_entry-only authorization check must
// not start requiring actor_user_id for the pre-existing payable types this
// ticket does not touch — CreateOnlinePayment for a booking (or a
// registration) stays exactly as open as it was before T10.6.
func TestCreateOnlinePayment_BookingPayable_NoActorFieldsRequired(t *testing.T) {
	t.Parallel()

	proc := stripestub.NewProcessor()
	svc := app.NewService(app.ServiceOptions{
		Payments:  newFakeRepository(),
		IDs:       &fixedIDs{ids: []string{"pay-1"}},
		Processor: proc,
	})

	if _, err := svc.CreateOnlinePayment(context.Background(), app.CreateOnlinePaymentInput{
		PayableType: domain.PayableTypeBooking,
		PayableID:   fixtureBookingID,
		Amount:      fixtureAmount(),
	}); err != nil {
		t.Fatalf("unexpected err: %v", err)
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
		PayableID:   fixtureRegistrationID,
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

// --- T6.5: Registration.PaymentStatus reconciliation -----------------------
//
// These are the ticket's required tests: an in-memory fake
// RegistrationPaymentUpdater proves ConfirmOnlinePayment/RecordOfflinePayment
// call it exactly once with the right registration id and status on
// success, and not at all when PayableType == booking.

// TestConfirmOnlinePayment_RegistrationPayable_UpdatesRegistration proves a
// successful online capture for a registration-payable Payment calls
// RegistrationUpdater.UpdatePaymentStatus exactly once, with PayableID as
// the registration id and PaymentStatusPaid as the status.
func TestConfirmOnlinePayment_RegistrationPayable_UpdatesRegistration(t *testing.T) {
	t.Parallel()

	proc := stripestub.NewProcessor()
	repo := newFakeRepository()
	updater := &fakeRegistrationUpdater{}
	svc := app.NewService(app.ServiceOptions{
		Payments:            repo,
		IDs:                 &fixedIDs{ids: []string{"pay-1"}},
		Processor:           proc,
		RegistrationUpdater: updater,
	})

	p, err := svc.CreateOnlinePayment(context.Background(), app.CreateOnlinePaymentInput{
		PayableType: domain.PayableTypeRegistration,
		PayableID:   fixtureRegistrationID,
		Amount:      fixtureAmount(),
	})
	if err != nil {
		t.Fatalf("unexpected err creating: %v", err)
	}

	if _, err := svc.ConfirmOnlinePayment(context.Background(), p); err != nil {
		t.Fatalf("unexpected err confirming: %v", err)
	}

	if len(updater.calls) != 1 {
		t.Fatalf("RegistrationUpdater called %d times, want exactly 1: %+v", len(updater.calls), updater.calls)
	}
	if updater.calls[0].registrationID != fixtureRegistrationID {
		t.Fatalf("registrationID = %q, want reg-1", updater.calls[0].registrationID)
	}
	if updater.calls[0].status != socialplaydomain.PaymentStatusPaid {
		t.Fatalf("status = %v, want paid", updater.calls[0].status)
	}
}

// TestConfirmOnlinePayment_BookingPayable_DoesNotUpdateRegistration is the
// ticket's required negative case: a booking-payable Payment must never
// trigger a Registration update.
func TestConfirmOnlinePayment_BookingPayable_DoesNotUpdateRegistration(t *testing.T) {
	t.Parallel()

	proc := stripestub.NewProcessor()
	repo := newFakeRepository()
	updater := &fakeRegistrationUpdater{}
	svc := app.NewService(app.ServiceOptions{
		Payments:            repo,
		IDs:                 &fixedIDs{ids: []string{"pay-1"}},
		Processor:           proc,
		RegistrationUpdater: updater,
	})

	p, err := svc.CreateOnlinePayment(context.Background(), app.CreateOnlinePaymentInput{
		PayableType: domain.PayableTypeBooking,
		PayableID:   fixtureBookingID,
		Amount:      fixtureAmount(),
	})
	if err != nil {
		t.Fatalf("unexpected err creating: %v", err)
	}

	if _, err := svc.ConfirmOnlinePayment(context.Background(), p); err != nil {
		t.Fatalf("unexpected err confirming: %v", err)
	}

	if len(updater.calls) != 0 {
		t.Fatalf("RegistrationUpdater called %d times, want 0 for a booking-payable Payment: %+v", len(updater.calls), updater.calls)
	}
}

// TestConfirmOnlinePayment_Declined_DoesNotUpdateRegistration proves a
// declined capture (MarkPaid never called) never reaches the
// RegistrationUpdater either — only a genuinely successful MarkPaid may
// trigger the reconciliation call.
func TestConfirmOnlinePayment_Declined_DoesNotUpdateRegistration(t *testing.T) {
	t.Parallel()

	proc := stripestub.NewProcessor()
	proc.SeedDecline(fixtureRegistrationID)
	repo := newFakeRepository()
	updater := &fakeRegistrationUpdater{}
	svc := app.NewService(app.ServiceOptions{
		Payments:            repo,
		IDs:                 &fixedIDs{ids: []string{"pay-1"}},
		Processor:           proc,
		RegistrationUpdater: updater,
	})

	p, err := svc.CreateOnlinePayment(context.Background(), app.CreateOnlinePaymentInput{
		PayableType: domain.PayableTypeRegistration,
		PayableID:   fixtureRegistrationID,
		Amount:      fixtureAmount(),
	})
	if err != nil {
		t.Fatalf("unexpected err creating: %v", err)
	}

	if _, err := svc.ConfirmOnlinePayment(context.Background(), p); !errors.Is(err, domain.ErrPaymentDeclined) {
		t.Fatalf("got err %v, want %v", err, domain.ErrPaymentDeclined)
	}

	if len(updater.calls) != 0 {
		t.Fatalf("RegistrationUpdater called %d times, want 0 on a decline: %+v", len(updater.calls), updater.calls)
	}
}

// TestRecordOfflinePayment_RegistrationPayable_UpdatesRegistration mirrors
// TestConfirmOnlinePayment_RegistrationPayable_UpdatesRegistration for the
// offline path: RecordOfflinePayment's immediate MarkPaid must also
// reconcile the Registration.
func TestRecordOfflinePayment_RegistrationPayable_UpdatesRegistration(t *testing.T) {
	t.Parallel()

	updater := &fakeRegistrationUpdater{}
	svc := app.NewService(app.ServiceOptions{
		Payments:            newFakeRepository(),
		IDs:                 &fixedIDs{ids: []string{"pay-1"}},
		RegistrationUpdater: updater,
	})

	_, err := svc.RecordOfflinePayment(context.Background(), app.RecordOfflinePaymentInput{
		PayableType: domain.PayableTypeRegistration,
		PayableID:   fixtureRegistrationID,
		Amount:      offlineFixtureAmount(),
		ActorUserID: "host-1",
		GameHostID:  "host-1",
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	if len(updater.calls) != 1 {
		t.Fatalf("RegistrationUpdater called %d times, want exactly 1: %+v", len(updater.calls), updater.calls)
	}
	if updater.calls[0].registrationID != fixtureRegistrationID {
		t.Fatalf("registrationID = %q, want reg-1", updater.calls[0].registrationID)
	}
	if updater.calls[0].status != socialplaydomain.PaymentStatusPaid {
		t.Fatalf("status = %v, want paid", updater.calls[0].status)
	}
}

// TestRecordOfflinePayment_BookingPayable_DoesNotUpdateRegistration is the
// ticket's required negative case for the offline path.
func TestRecordOfflinePayment_BookingPayable_DoesNotUpdateRegistration(t *testing.T) {
	t.Parallel()

	updater := &fakeRegistrationUpdater{}
	svc := app.NewService(app.ServiceOptions{
		Payments:            newFakeRepository(),
		IDs:                 &fixedIDs{ids: []string{"pay-1"}},
		RegistrationUpdater: updater,
	})

	_, err := svc.RecordOfflinePayment(context.Background(), app.RecordOfflinePaymentInput{
		PayableType:   domain.PayableTypeBooking,
		PayableID:     fixtureBookingID,
		Amount:        offlineFixtureAmount(),
		ActorUserID:   "host-1",
		BookingHostID: "host-1",
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	if len(updater.calls) != 0 {
		t.Fatalf("RegistrationUpdater called %d times, want 0 for a booking-payable Payment: %+v", len(updater.calls), updater.calls)
	}
}

// TestRecordOfflinePayment_NoShowFeePayable_DoesNotUpdateRegistration
// proves PayableTypeNoShowFee — despite also targeting a Registration's
// Game — deliberately does NOT trigger a Registration.PaymentStatus update
// either: the ticket's instructions scope the reconciliation to "PayableType
// == registration" specifically (see the PR description's judgment-call
// note), since a no-show fee is a separate charge, not the seat's own
// payment status.
func TestRecordOfflinePayment_NoShowFeePayable_DoesNotUpdateRegistration(t *testing.T) {
	t.Parallel()

	updater := &fakeRegistrationUpdater{}
	svc := app.NewService(app.ServiceOptions{
		Payments:            newFakeRepository(),
		IDs:                 &fixedIDs{ids: []string{"pay-1"}},
		RegistrationUpdater: updater,
	})

	_, err := svc.RecordOfflinePayment(context.Background(), app.RecordOfflinePaymentInput{
		PayableType: domain.PayableTypeNoShowFee,
		PayableID:   fixtureRegistrationID,
		Amount:      offlineFixtureAmount(),
		ActorUserID: "host-1",
		GameHostID:  "host-1",
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	if len(updater.calls) != 0 {
		t.Fatalf("RegistrationUpdater called %d times, want 0 for a no_show_fee-payable Payment: %+v", len(updater.calls), updater.calls)
	}
}

// TestConfirmOnlinePayment_NilRegistrationUpdater_DoesNotPanic proves a
// Service constructed without a RegistrationUpdater (e.g. every pre-T6.5
// test above, and any caller that only exercises the Booking-payable path)
// still works — RegistrationUpdater is optional, not a hard dependency
// every ServiceOptions must supply.
func TestConfirmOnlinePayment_NilRegistrationUpdater_DoesNotPanic(t *testing.T) {
	t.Parallel()

	proc := stripestub.NewProcessor()
	svc := app.NewService(app.ServiceOptions{
		Payments:  newFakeRepository(),
		IDs:       &fixedIDs{ids: []string{"pay-1"}},
		Processor: proc,
	})

	p, err := svc.CreateOnlinePayment(context.Background(), app.CreateOnlinePaymentInput{
		PayableType: domain.PayableTypeRegistration,
		PayableID:   fixtureRegistrationID,
		Amount:      fixtureAmount(),
	})
	if err != nil {
		t.Fatalf("unexpected err creating: %v", err)
	}

	if _, err := svc.ConfirmOnlinePayment(context.Background(), p); err != nil {
		t.Fatalf("unexpected err confirming with a nil RegistrationUpdater: %v", err)
	}
}

// --- T10.6: CompetitionEntry.PaymentStatus reconciliation (closes #96) ----
//
// Mirrors the T6.5 Registration reconciliation tests above exactly: an
// in-memory fake CompetitionEntryPaymentUpdater proves
// ConfirmOnlinePayment/RecordOfflinePayment call it exactly once with the
// right entry id and status on success, and not at all when
// PayableType != competition_entry — and, symmetrically, that a
// competition_entry-payable Payment never calls the RegistrationUpdater.

// TestConfirmOnlinePayment_CompetitionEntryPayable_UpdatesEntry proves a
// successful online capture for a competition_entry-payable Payment calls
// CompetitionEntryUpdater.UpdatePaymentStatus exactly once, with PayableID
// as the entry id and PaymentStatusPaid as the status.
func TestConfirmOnlinePayment_CompetitionEntryPayable_UpdatesEntry(t *testing.T) {
	t.Parallel()

	proc := stripestub.NewProcessor()
	repo := newFakeRepository()
	updater := &fakeCompetitionEntryUpdater{}
	svc := app.NewService(app.ServiceOptions{
		Payments:                repo,
		IDs:                     &fixedIDs{ids: []string{"pay-1"}},
		Processor:               proc,
		CompetitionEntryUpdater: updater,
	})

	p, err := svc.CreateOnlinePayment(context.Background(), app.CreateOnlinePaymentInput{
		PayableType:     domain.PayableTypeCompetitionEntry,
		PayableID:       fixtureCompetitionEntryID,
		Amount:          fixtureAmount(),
		ActorUserID:     "player-1",
		EntrantPlayerID: "player-1",
	})
	if err != nil {
		t.Fatalf("unexpected err creating: %v", err)
	}

	if _, err := svc.ConfirmOnlinePayment(context.Background(), p); err != nil {
		t.Fatalf("unexpected err confirming: %v", err)
	}

	if len(updater.calls) != 1 {
		t.Fatalf("CompetitionEntryUpdater called %d times, want exactly 1: %+v", len(updater.calls), updater.calls)
	}
	if updater.calls[0].entryID != fixtureCompetitionEntryID {
		t.Fatalf("entryID = %q, want entry-1", updater.calls[0].entryID)
	}
	if updater.calls[0].status != competitionsdomain.PaymentStatusPaid {
		t.Fatalf("status = %v, want paid", updater.calls[0].status)
	}
}

// TestConfirmOnlinePayment_RegistrationPayable_DoesNotUpdateCompetitionEntry
// is the symmetric negative case: a registration-payable Payment must never
// trigger a CompetitionEntry update.
func TestConfirmOnlinePayment_RegistrationPayable_DoesNotUpdateCompetitionEntry(t *testing.T) {
	t.Parallel()

	proc := stripestub.NewProcessor()
	repo := newFakeRepository()
	entryUpdater := &fakeCompetitionEntryUpdater{}
	svc := app.NewService(app.ServiceOptions{
		Payments:                repo,
		IDs:                     &fixedIDs{ids: []string{"pay-1"}},
		Processor:               proc,
		CompetitionEntryUpdater: entryUpdater,
	})

	p, err := svc.CreateOnlinePayment(context.Background(), app.CreateOnlinePaymentInput{
		PayableType: domain.PayableTypeRegistration,
		PayableID:   fixtureRegistrationID,
		Amount:      fixtureAmount(),
	})
	if err != nil {
		t.Fatalf("unexpected err creating: %v", err)
	}

	if _, err := svc.ConfirmOnlinePayment(context.Background(), p); err != nil {
		t.Fatalf("unexpected err confirming: %v", err)
	}

	if len(entryUpdater.calls) != 0 {
		t.Fatalf("CompetitionEntryUpdater called %d times, want 0 for a registration-payable Payment: %+v", len(entryUpdater.calls), entryUpdater.calls)
	}
}

// TestConfirmOnlinePayment_CompetitionEntryPayable_DoesNotUpdateRegistration
// is the mirror-image negative case: a competition_entry-payable Payment
// must never trigger a Registration update — the exact routing mistake #96
// exists to prevent (a Competition entry ID must never reach Social Play's
// RegistrationPaymentUpdater).
func TestConfirmOnlinePayment_CompetitionEntryPayable_DoesNotUpdateRegistration(t *testing.T) {
	t.Parallel()

	proc := stripestub.NewProcessor()
	repo := newFakeRepository()
	regUpdater := &fakeRegistrationUpdater{}
	entryUpdater := &fakeCompetitionEntryUpdater{}
	svc := app.NewService(app.ServiceOptions{
		Payments:                repo,
		IDs:                     &fixedIDs{ids: []string{"pay-1"}},
		Processor:               proc,
		RegistrationUpdater:     regUpdater,
		CompetitionEntryUpdater: entryUpdater,
	})

	p, err := svc.CreateOnlinePayment(context.Background(), app.CreateOnlinePaymentInput{
		PayableType:     domain.PayableTypeCompetitionEntry,
		PayableID:       fixtureCompetitionEntryID,
		Amount:          fixtureAmount(),
		ActorUserID:     "player-1",
		EntrantPlayerID: "player-1",
	})
	if err != nil {
		t.Fatalf("unexpected err creating: %v", err)
	}

	if _, err := svc.ConfirmOnlinePayment(context.Background(), p); err != nil {
		t.Fatalf("unexpected err confirming: %v", err)
	}

	if len(regUpdater.calls) != 0 {
		t.Fatalf("RegistrationUpdater called %d times, want 0 for a competition_entry-payable Payment: %+v", len(regUpdater.calls), regUpdater.calls)
	}
	if len(entryUpdater.calls) != 1 {
		t.Fatalf("CompetitionEntryUpdater called %d times, want exactly 1: %+v", len(entryUpdater.calls), entryUpdater.calls)
	}
	if entryUpdater.calls[0].entryID != fixtureCompetitionEntryID {
		t.Fatalf("entryID = %q, want entry-1", entryUpdater.calls[0].entryID)
	}
}

// TestRecordOfflinePayment_CompetitionEntryPayable_UpdatesEntry mirrors
// TestConfirmOnlinePayment_CompetitionEntryPayable_UpdatesEntry for the
// offline path: RecordOfflinePayment's immediate MarkPaid must also
// reconcile the CompetitionEntry.
func TestRecordOfflinePayment_CompetitionEntryPayable_UpdatesEntry(t *testing.T) {
	t.Parallel()

	updater := &fakeCompetitionEntryUpdater{}
	svc := app.NewService(app.ServiceOptions{
		Payments:                newFakeRepository(),
		IDs:                     &fixedIDs{ids: []string{"pay-1"}},
		CompetitionEntryUpdater: updater,
	})

	_, err := svc.RecordOfflinePayment(context.Background(), app.RecordOfflinePaymentInput{
		PayableType:     domain.PayableTypeCompetitionEntry,
		PayableID:       fixtureCompetitionEntryID,
		Amount:          offlineFixtureAmount(),
		ActorUserID:     "player-1",
		EntrantPlayerID: "player-1",
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	if len(updater.calls) != 1 {
		t.Fatalf("CompetitionEntryUpdater called %d times, want exactly 1: %+v", len(updater.calls), updater.calls)
	}
	if updater.calls[0].entryID != fixtureCompetitionEntryID {
		t.Fatalf("entryID = %q, want entry-1", updater.calls[0].entryID)
	}
	if updater.calls[0].status != competitionsdomain.PaymentStatusPaid {
		t.Fatalf("status = %v, want paid", updater.calls[0].status)
	}
}

// TestConfirmOnlinePayment_NilCompetitionEntryUpdater_DoesNotPanic proves a
// Service constructed without a CompetitionEntryUpdater still works —
// CompetitionEntryUpdater is optional, not a hard dependency every
// ServiceOptions must supply, mirroring RegistrationUpdater's own
// optionality.
func TestConfirmOnlinePayment_NilCompetitionEntryUpdater_DoesNotPanic(t *testing.T) {
	t.Parallel()

	proc := stripestub.NewProcessor()
	svc := app.NewService(app.ServiceOptions{
		Payments:  newFakeRepository(),
		IDs:       &fixedIDs{ids: []string{"pay-1"}},
		Processor: proc,
	})

	p, err := svc.CreateOnlinePayment(context.Background(), app.CreateOnlinePaymentInput{
		PayableType:     domain.PayableTypeCompetitionEntry,
		PayableID:       fixtureCompetitionEntryID,
		Amount:          fixtureAmount(),
		ActorUserID:     "player-1",
		EntrantPlayerID: "player-1",
	})
	if err != nil {
		t.Fatalf("unexpected err creating: %v", err)
	}

	if _, err := svc.ConfirmOnlinePayment(context.Background(), p); err != nil {
		t.Fatalf("unexpected err confirming with a nil CompetitionEntryUpdater: %v", err)
	}
}
