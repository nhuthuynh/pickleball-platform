package competitions_test

import (
	"context"
	"errors"
	"testing"

	competitionsapp "github.com/nhuthuynh/white-label/internal/competitions/app"
	competitionsdomain "github.com/nhuthuynh/white-label/internal/competitions/domain"
	paymentscompetitions "github.com/nhuthuynh/white-label/internal/payments/adapter/competitions"
	paymentsapp "github.com/nhuthuynh/white-label/internal/payments/app"
	paymentsdomain "github.com/nhuthuynh/white-label/internal/payments/domain"
)

// This file is competitions_test's mirror of
// internal/payments/adapter/socialplay/authorization_boundary_test.go —
// T16.2's headline mutation check (closing #168) for the
// PayableTypeCompetitionEntry branch, driven through the REAL Competitions
// app.Service via the REAL paymentscompetitions.EntryLookup/
// CompetitionAdminReader adapters, per T14.8/T15.5's cross-context-fake
// warning. See that file's own doc comment for the full reasoning, repeated
// here only where it differs.

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

func (r *boundaryFakePaymentsRepo) Update(_ context.Context, p paymentsdomain.Payment) (paymentsdomain.Payment, error) {
	if _, ok := r.byID[p.ID]; !ok {
		return paymentsdomain.Payment{}, paymentsdomain.ErrPaymentNotFound
	}
	r.byID[p.ID] = p
	return p, nil
}

// boundaryPaymentIDs is a deterministic payments/port.IDGenerator — distinct
// from this package's own fakeIDs (competitionsport.IDGenerator's stub),
// because this file needs three distinct Payment ids across three
// RecordOfflinePayment calls.
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
	boundaryCompetitionID = "6ba7b810-0000-4000-8000-0000000000d1"
	boundaryHostID        = "host-subject-t16-2-boundary"
	boundaryEntrantID     = "entrant-subject-t16-2-boundary"
	boundaryAdminID       = "admin-subject-t16-2-boundary"
	boundaryEntry1ID      = "6ba7b810-0000-4000-8000-0000000000d2"
	boundaryEntry2ID      = "6ba7b810-0000-4000-8000-0000000000d3"
	boundaryEntry3ID      = "6ba7b810-0000-4000-8000-0000000000d4"
)

func boundaryAmount() paymentsdomain.Money {
	return paymentsdomain.Money{Cents: 1000, Currency: "USD"}
}

// TestRecordOfflinePayment_RealCompetitionsSeam_EntrantSucceeds_AdminAssignThenRevoke
// is the CompetitionEntry-branch mutation table T16.2's instruction 9
// requires, run against the real competitionsapp.Service end to end through
// paymentsapp.Service.RecordOfflinePayment:
//
//  1. The entry's real entrant succeeds, resolved via the newly-built
//     EntryLookup (this test's RecordOfflinePaymentInput never sets an
//     entrant_player_id — the field no longer exists).
//  2. An as-yet-unassigned user is refused.
//  3. That SAME user, assigned as a Competition Admin through the real
//     competitionsapp.Service.AssignCompetitionAdmin (by boundaryHostID,
//     the Competition's own real Host — AssignCompetitionAdmin is a
//     Host-only action, distinct from the entrant), now succeeds — resolved
//     via CompetitionAdminReader.
//  4. The identical user, revoked through the real
//     competitionsapp.Service.RevokeCompetitionAdmin, is refused again —
//     against a fresh CompetitionEntry.
func TestRecordOfflinePayment_RealCompetitionsSeam_EntrantSucceeds_AdminAssignThenRevoke(t *testing.T) {
	t.Parallel()

	repo := newEntryLookupFakeRepository(
		competitionsdomain.CompetitionEntry{ID: boundaryEntry1ID, CompetitionID: boundaryCompetitionID, PlayerID: boundaryEntrantID, Status: competitionsdomain.EntryStatusEntered, PaymentStatus: competitionsdomain.PaymentStatusUnpaid},
		competitionsdomain.CompetitionEntry{ID: boundaryEntry2ID, CompetitionID: boundaryCompetitionID, PlayerID: "player-other-1", Status: competitionsdomain.EntryStatusEntered, PaymentStatus: competitionsdomain.PaymentStatusUnpaid},
		competitionsdomain.CompetitionEntry{ID: boundaryEntry3ID, CompetitionID: boundaryCompetitionID, PlayerID: "player-other-2", Status: competitionsdomain.EntryStatusEntered, PaymentStatus: competitionsdomain.PaymentStatusUnpaid},
	)
	if _, err := repo.Create(context.Background(), competitionsdomain.Competition{ID: boundaryCompetitionID, HostID: boundaryHostID}); err != nil {
		t.Fatalf("fixture: seed Competition: %v", err)
	}
	competitionAdmins := newAdminFakeCompetitionAdmins()
	competitionsSvc := competitionsapp.NewService(competitionsapp.ServiceOptions{
		Competitions:      repo,
		IDs:               fakeIDs{},
		Reservation:       fakeReservation{},
		Facilities:        fakeFacilityLookup{},
		ShareTokens:       fakeShareTokens{},
		CompetitionAdmins: competitionAdmins,
	})

	paymentsSvc := paymentsapp.NewService(paymentsapp.ServiceOptions{
		Payments:               newBoundaryFakePaymentsRepo(),
		IDs:                    &boundaryPaymentIDs{ids: []string{"6ba7b810-0000-4000-8000-0000000000f6", "6ba7b810-0000-4000-8000-0000000000f7", "6ba7b810-0000-4000-8000-0000000000f8"}},
		EntryLookup:            paymentscompetitions.NewEntryLookup(competitionsSvc),
		CompetitionAdminReader: paymentscompetitions.NewCompetitionAdminReader(competitionsSvc),
	})

	ctx := context.Background()

	// 1. The real entrant succeeds.
	if _, err := paymentsSvc.RecordOfflinePayment(ctx, paymentsapp.RecordOfflinePaymentInput{
		PayableType: paymentsdomain.PayableTypeCompetitionEntry,
		PayableID:   boundaryEntry1ID,
		Amount:      boundaryAmount(),
		ActorUserID: boundaryEntrantID,
	}); err != nil {
		t.Fatalf("entrant RecordOfflinePayment: unexpected err: %v", err)
	}

	// 2. Not-yet-assigned admin is refused, against a different entry
	// (boundaryEntry1ID is now paid — a second attempt against it would hit
	// ErrPaymentAlreadyRecorded, proving nothing about authorization).
	if _, err := paymentsSvc.RecordOfflinePayment(ctx, paymentsapp.RecordOfflinePaymentInput{
		PayableType: paymentsdomain.PayableTypeCompetitionEntry,
		PayableID:   boundaryEntry2ID,
		Amount:      boundaryAmount(),
		ActorUserID: boundaryAdminID,
	}); !errors.Is(err, paymentsdomain.ErrNotPaymentRecorder) {
		t.Fatalf("before assignment: got err %v, want %v", err, paymentsdomain.ErrNotPaymentRecorder)
	}

	// 3. Assign via the REAL Competitions app.Service — the mutation this
	// test's positive-then-negative control turns on. The assignor must be
	// the Competition's real Host, not the entrant (AssignCompetitionAdmin
	// is a Host-only action).
	if _, err := competitionsSvc.AssignCompetitionAdmin(ctx, competitionsapp.AssignCompetitionAdminInput{
		CompetitionID: boundaryCompetitionID,
		ActorUserID:   boundaryHostID,
		AdminUserID:   boundaryAdminID,
	}); err != nil {
		t.Fatalf("AssignCompetitionAdmin: unexpected err: %v", err)
	}

	// 4. Now the assigned admin succeeds — resolved via CompetitionAdminReader
	// against the store AssignCompetitionAdmin just wrote to, not a cached
	// answer.
	if _, err := paymentsSvc.RecordOfflinePayment(ctx, paymentsapp.RecordOfflinePaymentInput{
		PayableType: paymentsdomain.PayableTypeCompetitionEntry,
		PayableID:   boundaryEntry2ID,
		Amount:      boundaryAmount(),
		ActorUserID: boundaryAdminID,
	}); err != nil {
		t.Fatalf("assigned admin RecordOfflinePayment: unexpected err: %v", err)
	}

	// 5. Revoke via the REAL Competitions app.Service.
	if err := competitionsSvc.RevokeCompetitionAdmin(ctx, competitionsapp.RevokeCompetitionAdminInput{
		CompetitionID: boundaryCompetitionID,
		ActorUserID:   boundaryHostID,
		AdminUserID:   boundaryAdminID,
	}); err != nil {
		t.Fatalf("RevokeCompetitionAdmin: unexpected err: %v", err)
	}

	// 6. The identical actor is refused again, against a fresh
	// CompetitionEntry.
	if _, err := paymentsSvc.RecordOfflinePayment(ctx, paymentsapp.RecordOfflinePaymentInput{
		PayableType: paymentsdomain.PayableTypeCompetitionEntry,
		PayableID:   boundaryEntry3ID,
		Amount:      boundaryAmount(),
		ActorUserID: boundaryAdminID,
	}); !errors.Is(err, paymentsdomain.ErrNotPaymentRecorder) {
		t.Fatalf("after revocation: got err %v, want %v", err, paymentsdomain.ErrNotPaymentRecorder)
	}
}
