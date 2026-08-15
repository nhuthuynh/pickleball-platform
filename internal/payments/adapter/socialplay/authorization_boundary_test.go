package socialplay_test

import (
	"context"
	"errors"
	"testing"

	paymentssocialplay "github.com/nhuthuynh/white-label/internal/payments/adapter/socialplay"
	paymentsapp "github.com/nhuthuynh/white-label/internal/payments/app"
	paymentsdomain "github.com/nhuthuynh/white-label/internal/payments/domain"
	socialplayapp "github.com/nhuthuynh/white-label/internal/socialplay/app"
	socialplaydomain "github.com/nhuthuynh/white-label/internal/socialplay/domain"
)

// This file is T16.2's headline mutation check (closing #168), delivered at
// the seam T15.5's game_admin_reader_test.go could not reach: it drives
// internal/payments/app.Service.RecordOfflinePayment itself — not just
// GameAdminReader.ListGameAdmins in isolation — through the REAL Social Play
// app.Service, via the REAL paymentssocialplay.RegistrationLookup/GameLookup/
// GameAdminReader adapters this ticket built. Per T14.8/T15.5's
// cross-context-fake warning (repeated by this ticket's own instruction 9):
// a fake standing in for socialplayapp.Service would prove nothing about
// this seam, since the seam's whole job is translating a REAL
// GetRegistrationByID/GetGame/ListGameAdmins call correctly.

// boundaryFakePaymentsRepo is a minimal in-memory payments/port.Repository —
// just enough for this file's own tests, mirroring
// internal/payments/app/fixtures_test.go's fakeRepository without its
// call-counting instrumentation (not needed here).
type boundaryFakePaymentsRepo struct {
	byID map[string]paymentsdomain.Payment
}

func newBoundaryFakePaymentsRepo() *boundaryFakePaymentsRepo {
	return &boundaryFakePaymentsRepo{byID: map[string]paymentsdomain.Payment{}}
}

func (r *boundaryFakePaymentsRepo) Create(_ context.Context, p paymentsdomain.Payment) (paymentsdomain.Payment, error) {
	r.byID[p.ID] = p
	return p, nil
}

func (r *boundaryFakePaymentsRepo) GetByID(_ context.Context, id string) (paymentsdomain.Payment, error) {
	p, ok := r.byID[id]
	if !ok {
		return paymentsdomain.Payment{}, paymentsdomain.ErrPaymentNotFound
	}
	return p, nil
}

// GetByStripeReference implements port.Repository (T18.1, closes #167) —
// unused by this file's own scenarios, but required for
// *boundaryFakePaymentsRepo to keep satisfying the interface.
func (r *boundaryFakePaymentsRepo) GetByStripeReference(_ context.Context, ref string) (paymentsdomain.Payment, error) {
	for _, p := range r.byID {
		if p.StripeReference == ref {
			return p, nil
		}
	}
	return paymentsdomain.Payment{}, paymentsdomain.ErrPaymentNotFound
}

func (r *boundaryFakePaymentsRepo) Update(_ context.Context, p paymentsdomain.Payment) (paymentsdomain.Payment, error) {
	if _, ok := r.byID[p.ID]; !ok {
		return paymentsdomain.Payment{}, paymentsdomain.ErrPaymentNotFound
	}
	r.byID[p.ID] = p
	return p, nil
}

// boundaryPaymentIDs is a deterministic payments/port.IDGenerator — distinct
// from this package's own fakeIDs (socialplayport.IDGenerator's stub, which
// always returns the constant "unused"), because this file needs three
// distinct Payment ids across three RecordOfflinePayment calls.
type boundaryPaymentIDs struct {
	ids  []string
	next int
}

func (f *boundaryPaymentIDs) NewID() string {
	id := f.ids[f.next]
	f.next++
	return id
}

const (
	boundaryGameID  = "6ba7b810-0000-4000-8000-0000000000e1"
	boundaryHostID  = "host-subject-t16-2-boundary"
	boundaryAdminID = "admin-subject-t16-2-boundary"
	boundaryReg1ID  = "6ba7b810-0000-4000-8000-0000000000e2"
	boundaryReg2ID  = "6ba7b810-0000-4000-8000-0000000000e3"
	boundaryReg3ID  = "6ba7b810-0000-4000-8000-0000000000e4"
)

func boundaryAmount() paymentsdomain.Money {
	return paymentsdomain.Money{Cents: 1000, Currency: "USD"}
}

// TestRecordOfflinePayment_RealSocialPlaySeam_HostSucceeds_AdminAssignThenRevoke
// is the full-stack mutation table T16.2's instruction 9 requires, run
// against the real socialplayapp.Service end to end through
// paymentsapp.Service.RecordOfflinePayment:
//
//  1. The Game's real Host succeeds, resolved via the newly-built
//     RegistrationLookup -> GameLookup chain (not a caller-supplied
//     game_host_id — this test's RecordOfflinePaymentInput never sets one,
//     because the field no longer exists).
//  2. An as-yet-unassigned user is refused.
//  3. That SAME user, assigned as a Game Admin through the real
//     socialplayapp.Service.AssignGameAdmin, now succeeds — resolved via
//     GameAdminReader.
//  4. The identical user, revoked through the real
//     socialplayapp.Service.RevokeGameAdmin, is refused again — against a
//     fresh Registration (a second Payment for the same PayableID would hit
//     ErrPaymentAlreadyRecorded and prove nothing about authorization).
//
// This is genuinely a positive-then-negative control on a REAL mutation
// (AssignGameAdmin/RevokeGameAdmin), not two independently-configured fakes
// that happen to differ — the same distinction
// TestListGameAdmins_AssignThenRevoke draws for the reader alone, extended
// one level up to the authorization decision it feeds.
func TestRecordOfflinePayment_RealSocialPlaySeam_HostSucceeds_AdminAssignThenRevoke(t *testing.T) {
	t.Parallel()

	games := newAdminFakeGames(socialplaydomain.Game{ID: boundaryGameID, HostID: boundaryHostID})
	gameAdmins := newAdminFakeGameAdmins()
	regs := newFakeRegistrations(
		socialplaydomain.Registration{ID: boundaryReg1ID, GameID: boundaryGameID, PlayerID: "player-a", Status: socialplaydomain.RegistrationStatusRegistered, PaymentStatus: socialplaydomain.PaymentStatusUnpaid},
		socialplaydomain.Registration{ID: boundaryReg2ID, GameID: boundaryGameID, PlayerID: "player-b", Status: socialplaydomain.RegistrationStatusRegistered, PaymentStatus: socialplaydomain.PaymentStatusUnpaid},
		socialplaydomain.Registration{ID: boundaryReg3ID, GameID: boundaryGameID, PlayerID: "player-c", Status: socialplaydomain.RegistrationStatusRegistered, PaymentStatus: socialplaydomain.PaymentStatusUnpaid},
	)
	socialplaySvc := socialplayapp.NewService(socialplayapp.ServiceOptions{
		IDs:           fakeIDs{},
		Games:         games,
		Registrations: regs,
		Waitlist:      fakeWaitlist{},
		Matches:       fakeMatches{},
		GameAdmins:    gameAdmins,
	})

	paymentsSvc := paymentsapp.NewService(paymentsapp.ServiceOptions{
		Payments:           newBoundaryFakePaymentsRepo(),
		IDs:                &boundaryPaymentIDs{ids: []string{"6ba7b810-0000-4000-8000-0000000000f1", "6ba7b810-0000-4000-8000-0000000000f2", "6ba7b810-0000-4000-8000-0000000000f3"}},
		RegistrationLookup: paymentssocialplay.NewRegistrationLookup(socialplaySvc),
		GameLookup:         paymentssocialplay.NewGameLookup(socialplaySvc),
		GameAdminReader:    paymentssocialplay.NewGameAdminReader(socialplaySvc),
	})

	ctx := context.Background()

	// 1. The real Host succeeds.
	if _, err := paymentsSvc.RecordOfflinePayment(ctx, paymentsapp.RecordOfflinePaymentInput{
		PayableType: paymentsdomain.PayableTypeRegistration,
		PayableID:   boundaryReg1ID,
		Amount:      boundaryAmount(),
		ActorUserID: boundaryHostID,
	}); err != nil {
		t.Fatalf("Host RecordOfflinePayment: unexpected err: %v", err)
	}

	// 2. Not-yet-assigned admin is refused.
	if _, err := paymentsSvc.RecordOfflinePayment(ctx, paymentsapp.RecordOfflinePaymentInput{
		PayableType: paymentsdomain.PayableTypeRegistration,
		PayableID:   boundaryReg2ID,
		Amount:      boundaryAmount(),
		ActorUserID: boundaryAdminID,
	}); !errors.Is(err, paymentsdomain.ErrNotPaymentRecorder) {
		t.Fatalf("before assignment: got err %v, want %v", err, paymentsdomain.ErrNotPaymentRecorder)
	}

	// 3. Assign via the REAL Social Play app.Service — the mutation this
	// test's positive-then-negative control turns on.
	if _, err := socialplaySvc.AssignGameAdmin(ctx, socialplayapp.AssignGameAdminInput{
		GameID:      boundaryGameID,
		ActorUserID: boundaryHostID,
		AdminUserID: boundaryAdminID,
	}); err != nil {
		t.Fatalf("AssignGameAdmin: unexpected err: %v", err)
	}

	// 4. Now the assigned admin succeeds — resolved via GameAdminReader
	// against the store AssignGameAdmin just wrote to, not a cached answer.
	if _, err := paymentsSvc.RecordOfflinePayment(ctx, paymentsapp.RecordOfflinePaymentInput{
		PayableType: paymentsdomain.PayableTypeRegistration,
		PayableID:   boundaryReg2ID,
		Amount:      boundaryAmount(),
		ActorUserID: boundaryAdminID,
	}); err != nil {
		t.Fatalf("assigned admin RecordOfflinePayment: unexpected err: %v", err)
	}

	// 5. Revoke via the REAL Social Play app.Service.
	if err := socialplaySvc.RevokeGameAdmin(ctx, socialplayapp.RevokeGameAdminInput{
		GameID:      boundaryGameID,
		ActorUserID: boundaryHostID,
		AdminUserID: boundaryAdminID,
	}); err != nil {
		t.Fatalf("RevokeGameAdmin: unexpected err: %v", err)
	}

	// 6. The identical actor is refused again, against a fresh Registration
	// (boundaryReg3ID — a second attempt against boundaryReg2ID would hit
	// ErrPaymentAlreadyRecorded, proving nothing about authorization).
	if _, err := paymentsSvc.RecordOfflinePayment(ctx, paymentsapp.RecordOfflinePaymentInput{
		PayableType: paymentsdomain.PayableTypeRegistration,
		PayableID:   boundaryReg3ID,
		Amount:      boundaryAmount(),
		ActorUserID: boundaryAdminID,
	}); !errors.Is(err, paymentsdomain.ErrNotPaymentRecorder) {
		t.Fatalf("after revocation: got err %v, want %v", err, paymentsdomain.ErrNotPaymentRecorder)
	}
}
