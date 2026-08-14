package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nhuthuynh/white-label/internal/payments/adapter/stripestub"
	"github.com/nhuthuynh/white-label/internal/payments/app"
	"github.com/nhuthuynh/white-label/internal/payments/domain"
	socialplaydomain "github.com/nhuthuynh/white-label/internal/socialplay/domain"
)

// T12.3 — app-layer tests for Service.RefundPayment, the method that closes
// the gap open since T6.5: domain.Payment.Refund() and
// port.PaymentProcessor.RefundPayment both already existed, but nothing
// called either, so `refunded` was a status no request could reach.
//
// Payment-id fixtures are UUID-shaped because RefundPayment resolves the
// Payment through Service.GetPayment, which applies the same uuidShape
// boundary guard (PR #89's Layer 2, generalized by T11.9) every other
// get-shaped read in this codebase has — a "pay-1"-shaped id would be
// rejected as malformed before the repository is ever consulted, exactly
// the fixture infidelity docs/LESSONS.md's T9 entry describes.
const (
	fixtureBookingPaymentID          = "6ba7b810-0000-4000-8000-0000000000a1"
	fixtureRegistrationPaymentID     = "6ba7b810-0000-4000-8000-0000000000a2"
	fixtureNoShowFeePaymentID        = "6ba7b810-0000-4000-8000-0000000000a3"
	fixtureCompetitionEntryPaymentID = "6ba7b810-0000-4000-8000-0000000000a4"

	fixtureBookingHostID  = "host-booking-1"
	fixtureGameHostID     = "host-game-1"
	fixtureGameAdminID    = "admin-game-1"
	fixtureOutsidePlayer  = "player-outsider-1"
	fixtureEntrantPlayer  = "player-entrant-1"
	fixtureCompAdminUser  = "admin-competition-1"
	fixtureRefundCurrency = "USD"
)

// seedPaidOnlinePayment drives the *real* online path (CreateOnlinePayment
// then ConfirmOnlinePayment) to arrive at a paid, online Payment carrying a
// stripestub intent reference the stub will actually recognise on refund.
// Built this way rather than hand-constructing a domain.Payment so the
// refund tests exercise a Payment that a real caller could actually have
// produced — including a StripeReference the processor knows about.
func seedPaidOnlinePayment(t *testing.T, svc *app.Service, paymentID string, payableType domain.PayableType, payableID string) domain.Payment {
	t.Helper()

	created, err := svc.CreateOnlinePayment(context.Background(), app.CreateOnlinePaymentInput{
		PayableType: payableType,
		PayableID:   payableID,
		Amount:      domain.Money{Cents: 3000, Currency: fixtureRefundCurrency},
	})
	if err != nil {
		t.Fatalf("seed: CreateOnlinePayment: %v", err)
	}
	if created.ID != paymentID {
		t.Fatalf("seed: payment id = %q, want %q (check the seeded fixedIDs)", created.ID, paymentID)
	}

	paid, err := svc.ConfirmOnlinePayment(context.Background(), created)
	if err != nil {
		t.Fatalf("seed: ConfirmOnlinePayment: %v", err)
	}
	if paid.Status != domain.StatusPaid {
		t.Fatalf("seed: status = %v, want paid", paid.Status)
	}
	return paid
}

// --- happy paths ----------------------------------------------------------

// TestRefundPayment_OnlineBookingPayable_HostSucceeds is the online happy
// path: the Booking's Host refunds a paid online Payment, the processor is
// asked to refund the stored intent reference, the domain state machine
// moves the Payment to refunded, and the transition is persisted.
func TestRefundPayment_OnlineBookingPayable_HostSucceeds(t *testing.T) {
	t.Parallel()

	proc := stripestub.NewProcessor()
	repo := newFakeRepository()
	svc := app.NewService(app.ServiceOptions{
		Payments:  repo,
		IDs:       &fixedIDs{ids: []string{fixtureBookingPaymentID}},
		Processor: proc,
	})
	seedPaidOnlinePayment(t, svc, fixtureBookingPaymentID, domain.PayableTypeBooking, fixtureBookingID)

	refunded, err := svc.RefundPayment(context.Background(), app.RefundPaymentInput{
		PaymentID:     fixtureBookingPaymentID,
		ActorUserID:   fixtureBookingHostID,
		BookingHostID: fixtureBookingHostID,
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if refunded.Status != domain.StatusRefunded {
		t.Fatalf("Status = %v, want refunded", refunded.Status)
	}

	stored, err := repo.GetByID(context.Background(), fixtureBookingPaymentID)
	if err != nil {
		t.Fatalf("expected the Payment to still exist: %v", err)
	}
	if stored.Status != domain.StatusRefunded {
		t.Fatalf("persisted Status = %v, want refunded — the transition must be persisted, not just returned", stored.Status)
	}
}

// TestRefundPayment_OfflineRegistrationPayable_GameHostSucceeds is the
// offline happy path: an offline Payment has no processor involvement at
// all (there is no intent to refund — the money moved as cash), so the Game
// Host recording the refund is purely a domain transition plus a persist.
func TestRefundPayment_OfflineRegistrationPayable_GameHostSucceeds(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	svc := app.NewService(app.ServiceOptions{
		Payments: repo,
		IDs:      &fixedIDs{ids: []string{fixtureRegistrationPaymentID}},
	})

	if _, err := svc.RecordOfflinePayment(context.Background(), app.RecordOfflinePaymentInput{
		PayableType: domain.PayableTypeRegistration,
		PayableID:   fixtureRegistrationID,
		Amount:      offlineFixtureAmount(),
		ActorUserID: fixtureGameHostID,
		GameHostID:  fixtureGameHostID,
	}); err != nil {
		t.Fatalf("seed: RecordOfflinePayment: %v", err)
	}

	refunded, err := svc.RefundPayment(context.Background(), app.RefundPaymentInput{
		PaymentID:   fixtureRegistrationPaymentID,
		ActorUserID: fixtureGameHostID,
		GameHostID:  fixtureGameHostID,
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if refunded.Status != domain.StatusRefunded {
		t.Fatalf("Status = %v, want refunded", refunded.Status)
	}
}

// TestRefundPayment_RegistrationPayable_AssignedGameAdminSucceeds proves
// the refund reuses authorizeOfflineRecording's *existing* authorized-actor
// set rather than a second, refund-specific authorization concept: an
// assigned Game Admin — already allowed to record an offline payment for
// this Game — is allowed to refund one too.
func TestRefundPayment_RegistrationPayable_AssignedGameAdminSucceeds(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	svc := app.NewService(app.ServiceOptions{
		Payments: repo,
		IDs:      &fixedIDs{ids: []string{fixtureRegistrationPaymentID}},
	})

	if _, err := svc.RecordOfflinePayment(context.Background(), app.RecordOfflinePaymentInput{
		PayableType: domain.PayableTypeRegistration,
		PayableID:   fixtureRegistrationID,
		Amount:      offlineFixtureAmount(),
		ActorUserID: fixtureGameHostID,
		GameHostID:  fixtureGameHostID,
	}); err != nil {
		t.Fatalf("seed: RecordOfflinePayment: %v", err)
	}

	refunded, err := svc.RefundPayment(context.Background(), app.RefundPaymentInput{
		PaymentID:                fixtureRegistrationPaymentID,
		ActorUserID:              fixtureGameAdminID,
		GameHostID:               fixtureGameHostID,
		AssignedGameAdminUserIDs: []string{fixtureGameAdminID},
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if refunded.Status != domain.StatusRefunded {
		t.Fatalf("Status = %v, want refunded", refunded.Status)
	}
}

// --- authorization --------------------------------------------------------

// TestRefundPayment_WrongActorRejected proves a Player who is neither the
// Game's Host nor an assigned Game Admin cannot refund someone else's
// Payment, and that the rejection reuses the existing
// ErrNotPaymentRecorder sentinel (not a new refund-specific one).
func TestRefundPayment_WrongActorRejected(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	svc := app.NewService(app.ServiceOptions{
		Payments: repo,
		IDs:      &fixedIDs{ids: []string{fixtureRegistrationPaymentID}},
	})

	if _, err := svc.RecordOfflinePayment(context.Background(), app.RecordOfflinePaymentInput{
		PayableType: domain.PayableTypeRegistration,
		PayableID:   fixtureRegistrationID,
		Amount:      offlineFixtureAmount(),
		ActorUserID: fixtureGameHostID,
		GameHostID:  fixtureGameHostID,
	}); err != nil {
		t.Fatalf("seed: RecordOfflinePayment: %v", err)
	}

	_, err := svc.RefundPayment(context.Background(), app.RefundPaymentInput{
		PaymentID:   fixtureRegistrationPaymentID,
		ActorUserID: fixtureOutsidePlayer,
		GameHostID:  fixtureGameHostID,
	})
	if !errors.Is(err, domain.ErrNotPaymentRecorder) {
		t.Fatalf("got err %v, want %v", err, domain.ErrNotPaymentRecorder)
	}

	stored, getErr := repo.GetByID(context.Background(), fixtureRegistrationPaymentID)
	if getErr != nil {
		t.Fatalf("unexpected err: %v", getErr)
	}
	if stored.Status != domain.StatusPaid {
		t.Fatalf("persisted Status = %v, want paid — a rejected actor must have no side effect", stored.Status)
	}
}

// TestRefundPayment_MissingActorRejected proves an empty actor is rejected
// rather than treated as a match against an equally-empty ownership fact.
func TestRefundPayment_MissingActorRejected(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	svc := app.NewService(app.ServiceOptions{
		Payments: repo,
		IDs:      &fixedIDs{ids: []string{fixtureRegistrationPaymentID}},
	})

	if _, err := svc.RecordOfflinePayment(context.Background(), app.RecordOfflinePaymentInput{
		PayableType: domain.PayableTypeRegistration,
		PayableID:   fixtureRegistrationID,
		Amount:      offlineFixtureAmount(),
		ActorUserID: fixtureGameHostID,
		GameHostID:  fixtureGameHostID,
	}); err != nil {
		t.Fatalf("seed: RecordOfflinePayment: %v", err)
	}

	_, err := svc.RefundPayment(context.Background(), app.RefundPaymentInput{
		PaymentID: fixtureRegistrationPaymentID,
	})
	if !errors.Is(err, domain.ErrNotPaymentRecorder) {
		t.Fatalf("got err %v, want %v", err, domain.ErrNotPaymentRecorder)
	}
}

// --- domain state machine -------------------------------------------------

// TestRefundPayment_AlreadyRefundedRejectedByDomain proves the second
// refund is rejected by domain.Payment.Refund()'s own state machine — this
// method must not re-implement the transition check in the app layer
// (T12.3 instruction 2). `refunded` is terminal.
func TestRefundPayment_AlreadyRefundedRejectedByDomain(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	svc := app.NewService(app.ServiceOptions{
		Payments: repo,
		IDs:      &fixedIDs{ids: []string{fixtureRegistrationPaymentID}},
	})

	if _, err := svc.RecordOfflinePayment(context.Background(), app.RecordOfflinePaymentInput{
		PayableType: domain.PayableTypeRegistration,
		PayableID:   fixtureRegistrationID,
		Amount:      offlineFixtureAmount(),
		ActorUserID: fixtureGameHostID,
		GameHostID:  fixtureGameHostID,
	}); err != nil {
		t.Fatalf("seed: RecordOfflinePayment: %v", err)
	}

	in := app.RefundPaymentInput{
		PaymentID:   fixtureRegistrationPaymentID,
		ActorUserID: fixtureGameHostID,
		GameHostID:  fixtureGameHostID,
	}
	if _, err := svc.RefundPayment(context.Background(), in); err != nil {
		t.Fatalf("first refund should succeed: %v", err)
	}

	_, err := svc.RefundPayment(context.Background(), in)
	if !errors.Is(err, domain.ErrIllegalStatusTransition) {
		t.Fatalf("got err %v, want %v", err, domain.ErrIllegalStatusTransition)
	}
}

// TestRefundPayment_NeverPaidRejectedByDomain proves refunding an unpaid
// Payment is rejected — again by the domain, not an app-level pre-check.
// An intent was created (the Payment exists, with a StripeReference) but
// funds were never captured, so there is nothing to refund.
func TestRefundPayment_NeverPaidRejectedByDomain(t *testing.T) {
	t.Parallel()

	proc := stripestub.NewProcessor()
	repo := newFakeRepository()
	svc := app.NewService(app.ServiceOptions{
		Payments:  repo,
		IDs:       &fixedIDs{ids: []string{fixtureBookingPaymentID}},
		Processor: proc,
	})

	// Created (intent authorized) but deliberately never confirmed, so the
	// Payment is still unpaid.
	if _, err := svc.CreateOnlinePayment(context.Background(), app.CreateOnlinePaymentInput{
		PayableType: domain.PayableTypeBooking,
		PayableID:   fixtureBookingID,
		Amount:      fixtureAmount(),
	}); err != nil {
		t.Fatalf("seed: CreateOnlinePayment: %v", err)
	}

	_, err := svc.RefundPayment(context.Background(), app.RefundPaymentInput{
		PaymentID:     fixtureBookingPaymentID,
		ActorUserID:   fixtureBookingHostID,
		BookingHostID: fixtureBookingHostID,
	})
	if !errors.Is(err, domain.ErrIllegalStatusTransition) {
		t.Fatalf("got err %v, want %v", err, domain.ErrIllegalStatusTransition)
	}

	stored, getErr := repo.GetByID(context.Background(), fixtureBookingPaymentID)
	if getErr != nil {
		t.Fatalf("unexpected err: %v", getErr)
	}
	if stored.Status != domain.StatusUnpaid {
		t.Fatalf("persisted Status = %v, want unpaid", stored.Status)
	}
}

// --- lookup ---------------------------------------------------------------

// TestRefundPayment_UnknownAndMalformedIDsBothAnswerNotFound proves the
// lookup goes through Service.GetPayment's uuidShape guard: a malformed id
// is answered exactly like an unknown one, and neither reaches the
// repository in a shape the Postgres adapter's mustUUID would panic on.
func TestRefundPayment_UnknownAndMalformedIDsBothAnswerNotFound(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name          string
		paymentID     string
		wantRepoCalls int64
	}{
		{
			name:          "well-formed but unknown reaches the repository and misses",
			paymentID:     "6ba7b810-0000-4000-8000-0000000000ff",
			wantRepoCalls: 1,
		},
		{
			name:          "malformed is rejected before the repository is consulted",
			paymentID:     "not-a-uuid",
			wantRepoCalls: 0,
		},
		{
			name:          "empty is rejected before the repository is consulted",
			paymentID:     "",
			wantRepoCalls: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			repo := newFakeRepository()
			svc := app.NewService(app.ServiceOptions{
				Payments: repo,
				IDs:      &fixedIDs{ids: []string{fixtureRegistrationPaymentID}},
			})

			_, err := svc.RefundPayment(context.Background(), app.RefundPaymentInput{
				PaymentID:   tc.paymentID,
				ActorUserID: fixtureGameHostID,
				GameHostID:  fixtureGameHostID,
			})
			if !errors.Is(err, domain.ErrPaymentNotFound) {
				t.Fatalf("got err %v, want %v", err, domain.ErrPaymentNotFound)
			}
			if got := repo.getByIDCalls.Load(); got != tc.wantRepoCalls {
				t.Fatalf("repository GetByID calls = %d, want %d", got, tc.wantRepoCalls)
			}
		})
	}
}

// --- processor failure ----------------------------------------------------

// TestRefundPayment_ProcessorFailure_DoesNotMarkRefunded is T12.3's
// explicitly-required test (instruction 6): when the processor fails, the
// Payment must NOT be marked refunded — not in the value returned to the
// caller, and not in the store. A refund that didn't happen at the
// processor must not silently advance the Payment's status, which would
// leave the platform believing it returned money it still holds.
//
// The failure is produced honestly rather than by a bespoke fake: the stub
// processor returns ErrPaymentProcessorUnavailable for an intent reference
// it doesn't recognise (stripestub.Processor.RefundPayment), which is
// exactly what a real adapter does for an unknown/stale intent.
func TestRefundPayment_ProcessorFailure_DoesNotMarkRefunded(t *testing.T) {
	t.Parallel()

	proc := stripestub.NewProcessor()
	repo := newFakeRepository()
	updater := &fakeRegistrationUpdater{}
	svc := app.NewService(app.ServiceOptions{
		Payments:            repo,
		IDs:                 &fixedIDs{ids: []string{fixtureRegistrationPaymentID}},
		Processor:           proc,
		RegistrationUpdater: updater,
	})
	seedPaidOnlinePayment(t, svc, fixtureRegistrationPaymentID, domain.PayableTypeRegistration, fixtureRegistrationID)

	// Rewrite the stored Payment's processor reference to one the stub has
	// never issued, so the refund call fails at the processor while
	// everything else about the Payment stays valid and refundable.
	stored, err := repo.GetByID(context.Background(), fixtureRegistrationPaymentID)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	stored.StripeReference = "pi_stub_never_issued"
	if _, err := repo.Update(context.Background(), stored); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	// Seeding a *paid* registration Payment legitimately pushes `paid`
	// through the updater (T6.5's existing behaviour). Reset here so the
	// assertion below is specifically "the failed refund pushed nothing",
	// not a count that happens to include the seed.
	updater.calls = nil

	returned, err := svc.RefundPayment(context.Background(), app.RefundPaymentInput{
		PaymentID:   fixtureRegistrationPaymentID,
		ActorUserID: fixtureGameHostID,
		GameHostID:  fixtureGameHostID,
	})
	if !errors.Is(err, domain.ErrPaymentProcessorUnavailable) {
		t.Fatalf("got err %v, want %v", err, domain.ErrPaymentProcessorUnavailable)
	}
	if returned.Status == domain.StatusRefunded {
		t.Fatal("returned Payment must not be marked refunded when the processor failed")
	}

	after, err := repo.GetByID(context.Background(), fixtureRegistrationPaymentID)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if after.Status != domain.StatusPaid {
		t.Fatalf("persisted Status = %v, want paid — a failed refund must leave the Payment untouched", after.Status)
	}

	// And the Social Play projection must not be told about a refund that
	// never happened.
	if len(updater.calls) != 0 {
		t.Fatalf("registration updater calls = %d, want 0 on processor failure", len(updater.calls))
	}
}

// --- persistence failure --------------------------------------------------

// TestRefundPayment_PersistFailure_ReportsError is T12.3's other explicitly-
// required failure-path test (instruction 7): a refund that succeeds at the
// processor but then fails to persist the status change must not silently
// report success to the caller. The money really did move at the processor,
// so this is the genuinely dangerous case — the caller has to learn that
// the record is now inconsistent with the processor rather than be told
// everything is fine.
func TestRefundPayment_PersistFailure_ReportsError(t *testing.T) {
	t.Parallel()

	proc := stripestub.NewProcessor()
	repo := newFakeRepository()
	updater := &fakeRegistrationUpdater{}
	svc := app.NewService(app.ServiceOptions{
		Payments:            repo,
		IDs:                 &fixedIDs{ids: []string{fixtureRegistrationPaymentID}},
		Processor:           proc,
		RegistrationUpdater: updater,
	})
	seedPaidOnlinePayment(t, svc, fixtureRegistrationPaymentID, domain.PayableTypeRegistration, fixtureRegistrationID)

	// As above: drop the seed's legitimate `paid` push so the assertion
	// below is specifically about what the failed refund did.
	updater.calls = nil

	wantErr := errors.New("payments: simulated persistence failure")
	repo.updateErr = wantErr

	_, err := svc.RefundPayment(context.Background(), app.RefundPaymentInput{
		PaymentID:   fixtureRegistrationPaymentID,
		ActorUserID: fixtureGameHostID,
		GameHostID:  fixtureGameHostID,
	})
	if err == nil {
		t.Fatal("a refund that failed to persist must not report success")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("got err %v, want it to wrap %v", err, wantErr)
	}

	// The projection must not be pushed forward off the back of a
	// transition that was never persisted.
	if len(updater.calls) != 0 {
		t.Fatalf("registration updater calls = %d, want 0 when the persist failed", len(updater.calls))
	}
}

// --- Social Play projection ----------------------------------------------

// TestRefundPayment_RegistrationPayable_PushesRefundedToSocialPlay is
// T12.3's instruction 3: on success, PaymentStatusRefunded goes out through
// the *existing* socialplayport.RegistrationPaymentUpdater — the port built
// in T6.5 for exactly this call site, which has had no caller pushing
// `refunded` since. No second updater path.
func TestRefundPayment_RegistrationPayable_PushesRefundedToSocialPlay(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	updater := &fakeRegistrationUpdater{}
	svc := app.NewService(app.ServiceOptions{
		Payments:            repo,
		IDs:                 &fixedIDs{ids: []string{fixtureRegistrationPaymentID}},
		RegistrationUpdater: updater,
	})

	if _, err := svc.RecordOfflinePayment(context.Background(), app.RecordOfflinePaymentInput{
		PayableType: domain.PayableTypeRegistration,
		PayableID:   fixtureRegistrationID,
		Amount:      offlineFixtureAmount(),
		ActorUserID: fixtureGameHostID,
		GameHostID:  fixtureGameHostID,
	}); err != nil {
		t.Fatalf("seed: RecordOfflinePayment: %v", err)
	}
	// The recording itself pushes `paid` (T6.5's existing behaviour).
	if len(updater.calls) != 1 || updater.calls[0].status != socialplaydomain.PaymentStatusPaid {
		t.Fatalf("seed: registration updater calls = %+v, want one paid call", updater.calls)
	}

	if _, err := svc.RefundPayment(context.Background(), app.RefundPaymentInput{
		PaymentID:   fixtureRegistrationPaymentID,
		ActorUserID: fixtureGameHostID,
		GameHostID:  fixtureGameHostID,
	}); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	if len(updater.calls) != 2 {
		t.Fatalf("registration updater calls = %d, want 2 (the seed's paid, then this refund)", len(updater.calls))
	}
	got := updater.calls[1]
	if got.status != socialplaydomain.PaymentStatusRefunded {
		t.Fatalf("pushed status = %v, want %v", got.status, socialplaydomain.PaymentStatusRefunded)
	}
	if got.registrationID != fixtureRegistrationID {
		t.Fatalf("pushed registration id = %q, want %q", got.registrationID, fixtureRegistrationID)
	}
}

// TestRefundPayment_BookingPayable_DoesNotUpdateRegistration is the
// negative half: a booking-payable refund must never write into Social
// Play's Registration table, mirroring the equivalent guarantee T6.5
// already proves for the paid direction.
func TestRefundPayment_BookingPayable_DoesNotUpdateRegistration(t *testing.T) {
	t.Parallel()

	proc := stripestub.NewProcessor()
	repo := newFakeRepository()
	updater := &fakeRegistrationUpdater{}
	svc := app.NewService(app.ServiceOptions{
		Payments:            repo,
		IDs:                 &fixedIDs{ids: []string{fixtureBookingPaymentID}},
		Processor:           proc,
		RegistrationUpdater: updater,
	})
	seedPaidOnlinePayment(t, svc, fixtureBookingPaymentID, domain.PayableTypeBooking, fixtureBookingID)

	if _, err := svc.RefundPayment(context.Background(), app.RefundPaymentInput{
		PaymentID:     fixtureBookingPaymentID,
		ActorUserID:   fixtureBookingHostID,
		BookingHostID: fixtureBookingHostID,
	}); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	if len(updater.calls) != 0 {
		t.Fatalf("registration updater calls = %+v, want none for a booking payable", updater.calls)
	}
}

// TestRefundPayment_NilRegistrationUpdater_DoesNotPanic mirrors the
// existing nil-updater guarantees for the paid direction: the updater is
// optional (ServiceOptions' doc comment), so a Service wired without one
// skips the projection rather than panicking on a nil interface call.
func TestRefundPayment_NilRegistrationUpdater_DoesNotPanic(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	svc := app.NewService(app.ServiceOptions{
		Payments: repo,
		IDs:      &fixedIDs{ids: []string{fixtureRegistrationPaymentID}},
	})

	if _, err := svc.RecordOfflinePayment(context.Background(), app.RecordOfflinePaymentInput{
		PayableType: domain.PayableTypeRegistration,
		PayableID:   fixtureRegistrationID,
		Amount:      offlineFixtureAmount(),
		ActorUserID: fixtureGameHostID,
		GameHostID:  fixtureGameHostID,
	}); err != nil {
		t.Fatalf("seed: RecordOfflinePayment: %v", err)
	}

	refunded, err := svc.RefundPayment(context.Background(), app.RefundPaymentInput{
		PaymentID:   fixtureRegistrationPaymentID,
		ActorUserID: fixtureGameHostID,
		GameHostID:  fixtureGameHostID,
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if refunded.Status != domain.StatusRefunded {
		t.Fatalf("Status = %v, want refunded", refunded.Status)
	}
}

// --- scope boundary -------------------------------------------------------

// TestRefundPayment_OutOfScopePayableTypesRejected pins T12.3's stated
// scope as a test rather than a paragraph: this ticket refunds
// `registration`- and `booking`-payable Payments only.
//
//   - competition_entry: Competitions remains cash-only for refund purposes
//     (issue #125 tracks its PayableType gap). Deliberately not routed
//     through competitionsport.CompetitionEntryPaymentUpdater here.
//   - no_show_fee: named in neither half of the ticket's scope sentence, so
//     it is treated as out of scope rather than silently included — issue
//     #130 tracks it.
//
// Both answer with the existing ErrInvalidPayableType sentinel (no new
// domain error invented for a scope boundary), and neither reaches the
// authorization check or the domain transition.
func TestRefundPayment_OutOfScopePayableTypesRejected(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		paymentID   string
		payableType domain.PayableType
		payableID   string
		in          app.RecordOfflinePaymentInput
	}{
		{
			name:        "competition_entry is out of scope (issue #125)",
			paymentID:   fixtureCompetitionEntryPaymentID,
			payableType: domain.PayableTypeCompetitionEntry,
			payableID:   fixtureCompetitionEntryID,
			in: app.RecordOfflinePaymentInput{
				PayableType:     domain.PayableTypeCompetitionEntry,
				PayableID:       fixtureCompetitionEntryID,
				ActorUserID:     fixtureEntrantPlayer,
				EntrantPlayerID: fixtureEntrantPlayer,
			},
		},
		{
			name:        "no_show_fee is out of scope (issue #130)",
			paymentID:   fixtureNoShowFeePaymentID,
			payableType: domain.PayableTypeNoShowFee,
			payableID:   fixtureRegistrationID,
			in: app.RecordOfflinePaymentInput{
				PayableType: domain.PayableTypeNoShowFee,
				PayableID:   fixtureRegistrationID,
				ActorUserID: fixtureGameHostID,
				GameHostID:  fixtureGameHostID,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			repo := newFakeRepository()
			svc := app.NewService(app.ServiceOptions{
				Payments: repo,
				IDs:      &fixedIDs{ids: []string{tc.paymentID}},
			})

			seed := tc.in
			seed.Amount = offlineFixtureAmount()
			if _, err := svc.RecordOfflinePayment(context.Background(), seed); err != nil {
				t.Fatalf("seed: RecordOfflinePayment: %v", err)
			}

			_, err := svc.RefundPayment(context.Background(), app.RefundPaymentInput{
				PaymentID:                tc.paymentID,
				ActorUserID:              fixtureGameHostID,
				GameHostID:               fixtureGameHostID,
				AssignedGameAdminUserIDs: []string{fixtureCompAdminUser},
			})
			if !errors.Is(err, domain.ErrInvalidPayableType) {
				t.Fatalf("got err %v, want %v", err, domain.ErrInvalidPayableType)
			}

			stored, getErr := repo.GetByID(context.Background(), tc.paymentID)
			if getErr != nil {
				t.Fatalf("unexpected err: %v", getErr)
			}
			if stored.Status != domain.StatusPaid {
				t.Fatalf("persisted Status = %v, want paid — an out-of-scope refund must have no side effect", stored.Status)
			}
		})
	}
}
