package competitions_test

import (
	"context"
	"errors"
	"testing"

	competitionsapp "github.com/nhuthuynh/white-label/internal/competitions/app"
	competitionsdomain "github.com/nhuthuynh/white-label/internal/competitions/domain"
	paymentscompetitions "github.com/nhuthuynh/white-label/internal/payments/adapter/competitions"
	"github.com/nhuthuynh/white-label/internal/payments/adapter/stripestub"
	paymentsapp "github.com/nhuthuynh/white-label/internal/payments/app"
	paymentsdomain "github.com/nhuthuynh/white-label/internal/payments/domain"
)

// This file is T17.1's own headline mutation check (closes #198), the
// CreateOnlinePayment mirror of authorization_boundary_test.go's
// TestRecordOfflinePayment_RealCompetitionsSeam_EntrantSucceeds_AdminAssignThenRevoke
// — same real seam (the REAL competitionsapp.Service, through the REAL
// paymentscompetitions.EntryLookup/CompetitionAdminReader adapters, per
// T14.8/T15.5's cross-context-fake warning), same shared boundary*
// fixtures/constants from that file (same package, deliberately reused
// rather than re-declared — this ticket's own instructions require reusing
// T16.2's test infrastructure, not inventing a parallel harness), only the
// RPC under test differs: CreateOnlinePayment (the online/Stripe-intent
// path) instead of RecordOfflinePayment (the offline/cash path). The two
// RPCs now share the identical authorization method — Service.
// authorizeOnlineCreation delegates to authorizeCompetitionEntryRecording
// unchanged (see internal/payments/app/service.go's doc comment) — so this
// test is this ticket's proof that the delegation actually holds against the
// real Competitions store, not just against unit-level fakes
// (service_test.go already covers those).

// onlineBoundaryPaymentIDs mirrors boundaryPaymentIDs (authorization_boundary_test.go)
// but with its own three ids: CreateOnlinePayment persists an unpaid intent
// per call, just like RecordOfflinePayment persists a paid record per call,
// so this file needs its own distinct sequence rather than sharing one
// generator with a test that runs in a different (parallel) fakeIDs scope.
var onlineBoundaryPaymentIDs = []string{
	"6ba7b810-0000-4000-8000-0000000000fa",
	"6ba7b810-0000-4000-8000-0000000000fb",
	"6ba7b810-0000-4000-8000-0000000000fc",
}

// TestCreateOnlinePayment_RealCompetitionsSeam_EntrantSucceeds_AdminAssignThenRevoke
// is T17.1's required mutation table (instruction 5), run against the real
// competitionsapp.Service end to end through
// paymentsapp.Service.CreateOnlinePayment:
//
//  1. The entry's real entrant succeeds, resolved via EntryLookup (this
//     test's CreateOnlinePaymentInput never sets an entrant_player_id — the
//     field no longer exists on the Go struct at all, T17.1's own deletion).
//  2. An as-yet-unassigned user is refused.
//  3. That SAME user, assigned as a Competition Admin through the real
//     competitionsapp.Service.AssignCompetitionAdmin (by boundaryHostID, the
//     Competition's own real Host — AssignCompetitionAdmin is a Host-only
//     action, distinct from the entrant), now succeeds — resolved via
//     CompetitionAdminReader.
//  4. The identical user, revoked through the real
//     competitionsapp.Service.RevokeCompetitionAdmin, is refused again —
//     against a fresh CompetitionEntry.
func TestCreateOnlinePayment_RealCompetitionsSeam_EntrantSucceeds_AdminAssignThenRevoke(t *testing.T) {
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
		IDs:                    &boundaryPaymentIDs{ids: onlineBoundaryPaymentIDs},
		Processor:              stripestub.NewProcessor(),
		EntryLookup:            paymentscompetitions.NewEntryLookup(competitionsSvc),
		CompetitionAdminReader: paymentscompetitions.NewCompetitionAdminReader(competitionsSvc),
	})

	ctx := context.Background()

	// 1. The real entrant succeeds.
	if _, err := paymentsSvc.CreateOnlinePayment(ctx, paymentsapp.CreateOnlinePaymentInput{
		PayableType: paymentsdomain.PayableTypeCompetitionEntry,
		PayableID:   boundaryEntry1ID,
		Amount:      boundaryAmount(),
		ActorUserID: boundaryEntrantID,
	}); err != nil {
		t.Fatalf("entrant CreateOnlinePayment: unexpected err: %v", err)
	}

	// 2. Not-yet-assigned admin is refused, against a different entry
	// (boundaryEntry1ID already has an intent — a second attempt against it
	// would hit ErrPaymentAlreadyRecorded, proving nothing about
	// authorization).
	if _, err := paymentsSvc.CreateOnlinePayment(ctx, paymentsapp.CreateOnlinePaymentInput{
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
	if _, err := paymentsSvc.CreateOnlinePayment(ctx, paymentsapp.CreateOnlinePaymentInput{
		PayableType: paymentsdomain.PayableTypeCompetitionEntry,
		PayableID:   boundaryEntry2ID,
		Amount:      boundaryAmount(),
		ActorUserID: boundaryAdminID,
	}); err != nil {
		t.Fatalf("assigned admin CreateOnlinePayment: unexpected err: %v", err)
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
	if _, err := paymentsSvc.CreateOnlinePayment(ctx, paymentsapp.CreateOnlinePaymentInput{
		PayableType: paymentsdomain.PayableTypeCompetitionEntry,
		PayableID:   boundaryEntry3ID,
		Amount:      boundaryAmount(),
		ActorUserID: boundaryAdminID,
	}); !errors.Is(err, paymentsdomain.ErrNotPaymentRecorder) {
		t.Fatalf("after revocation: got err %v, want %v", err, paymentsdomain.ErrNotPaymentRecorder)
	}
}
