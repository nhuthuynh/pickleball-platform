// T10.6 (closes #96) — object-level authorization regression tests for the
// new PAYABLE_TYPE_COMPETITION_ENTRY payable, run through the real gRPC
// handler (not just the app-level unit tests
// internal/payments/app/service_test.go already has for
// authorizeOfflineRecording/authorizeOnlineCreation). This is the "does the
// guarantee survive the full stack" test the ticket's own instructions
// require, mirroring T8.5's
// authz_regression_test.go (internal/payments/adapter/grpcapi/authz_regression_test.go)
// exactly, one payable type over: RecordOfflinePayment's Registration-payable
// suite there is the direct template for the RecordOfflinePayment cases
// below, and this file adds the CreateOnlinePayment half T8.5 didn't need
// (the online path carried no actor at all before this ticket).
//
// Both RPCs under test here are covered by the same two sentinels toStatus
// already maps: domain.ErrNotPaymentRecorder -> PermissionDenied and
// domain.ErrInvalidPayableType -> InvalidArgument — no new status-code
// mapping was needed for this ticket, only new callers of the existing
// mapping.
package grpcapi_test

import (
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/nhuthuynh/white-label/internal/payments/adapter/grpcapi"
	"github.com/nhuthuynh/white-label/internal/payments/adapter/stripestub"
	"github.com/nhuthuynh/white-label/internal/payments/app"

	paymentsv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/payments/v1"
)

func competitionEntryFixtureAmount() *paymentsv1.Money {
	return &paymentsv1.Money{AmountCents: 2500, CurrencyCode: "USD"}
}

// newTestHandlerWithProcessor extends newTestHandler (authz_regression_test.go)
// with a stripestub.Processor, the one extra dependency CreateOnlinePayment
// needs that RecordOfflinePayment doesn't — no PermissionDenied case in
// this file needs a working capture (a rejected CreateOnlinePayment never
// reaches the processor at all), but NewService still requires a non-nil
// Processor for the online path's happy-path cases below. Wires the
// identical resolver defaults newTestHandler does (T16.2) — see that
// function's own doc comment.
func newTestHandlerWithProcessor(seedIDs ...string) (*grpcapi.Handler, *fakeRepository) {
	repo := newFakeRepository()
	// T28.1: seeded with resolvedUserID(subject), not the raw subject — see
	// newTestHandler's identical comment in authz_regression_test.go.
	regs, games, gameAdmins := newAuthzResolverFixtures(resolvedUserID("host-1"), resolvedUserID("admin-1"), resolvedUserID("admin-2"))
	entries, compAdmins := newEntryAuthzResolverFixtures(resolvedUserID("player-1"), resolvedUserID("admin-1"), resolvedUserID("admin-2"))
	svc := app.NewService(app.ServiceOptions{
		Payments:               repo,
		IDs:                    &fixedIDs{ids: seedIDs},
		Processor:              stripestub.NewProcessor(),
		RegistrationLookup:     regs,
		GameLookup:             games,
		GameAdminReader:        gameAdmins,
		EntryLookup:            entries,
		CompetitionAdminReader: compAdmins,
		Identity:               newFakeIdentityLookup(),
	})
	return grpcapi.NewHandler(svc), repo
}

// --- RecordOfflinePayment / CompetitionEntry-payable: object-level (BOLA) --

// TestRecordOfflinePayment_CompetitionEntryPayable_RejectsMismatchedActor
// is this ticket's required test: a Player who is neither the entrant nor
// an assigned Competition Admin attempts RecordOfflinePayment against
// another player's CompetitionEntry, through the real handler -> app ->
// domain path, and the request is rejected with the correctly mapped
// status — not a 500, not a silent success.
func TestRecordOfflinePayment_CompetitionEntryPayable_RejectsMismatchedActor(t *testing.T) {
	h, repo := newTestHandler("pay-1")

	// The BOLA attempt: "random-player", neither fixtureCompetitionEntryID's entrant nor one
	// of its assigned Competition Admins, tries to record a Payment against
	// it.
	_, err := h.RecordOfflinePayment(ctxAs("random-player"), &paymentsv1.RecordOfflinePaymentRequest{
		PayableType:                     paymentsv1.PayableType_PAYABLE_TYPE_COMPETITION_ENTRY,
		PayableId:                       fixtureCompetitionEntryID,
		Amount:                          competitionEntryFixtureAmount(),
		EntrantPlayerId:                 "player-1",
		AssignedCompetitionAdminUserIds: []string{"admin-1"},
	})
	if err == nil {
		t.Fatal("RecordOfflinePayment(random-player) succeeded silently — a Player who is neither the entrant nor an assigned Competition Admin was able to record a payment against another player's CompetitionEntry (BOLA regression)")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("RecordOfflinePayment(random-player) returned a non-gRPC-status error: %v (a client can't map this to a clean HTTP status)", err)
	}
	if st.Code() == codes.Internal {
		t.Fatalf("RecordOfflinePayment(random-player) mapped to Internal (500-shaped) — want PermissionDenied (403-shaped): %v", err)
	}
	if st.Code() != codes.PermissionDenied {
		t.Fatalf("RecordOfflinePayment(random-player) status code = %v, want PermissionDenied (403-shaped)", st.Code())
	}

	// Belt-and-braces, per the ticket's "not a silent success": prove no
	// Payment was persisted, not just that an error came back on the wire.
	if len(repo.byID) != 0 {
		t.Errorf("repo has %d Payments after a rejected RecordOfflinePayment, want 0 (the attacker's rejected attempt must not have any side effect)", len(repo.byID))
	}
}

// TestRecordOfflinePayment_CompetitionEntryPayable_AllowsEntrant is the
// symmetric positive-path case: the entrant recording their own
// competition-entry payment succeeds through the same handler path.
func TestRecordOfflinePayment_CompetitionEntryPayable_AllowsEntrant(t *testing.T) {
	h, repo := newTestHandler("pay-1")

	resp, err := h.RecordOfflinePayment(ctxAs("player-1"), &paymentsv1.RecordOfflinePaymentRequest{
		PayableType:     paymentsv1.PayableType_PAYABLE_TYPE_COMPETITION_ENTRY,
		PayableId:       fixtureCompetitionEntryID,
		Amount:          competitionEntryFixtureAmount(),
		EntrantPlayerId: "player-1",
	})
	if err != nil {
		t.Fatalf("RecordOfflinePayment(player-1) (the entrant) should succeed, got: %v", err)
	}
	if resp.GetPayment().GetStatus() != paymentsv1.PaymentStatus_PAYMENT_STATUS_PAID {
		t.Errorf("Payment.Status = %v, want PAYMENT_STATUS_PAID", resp.GetPayment().GetStatus())
	}
	if len(repo.byID) != 1 {
		t.Errorf("repo has %d Payments, want 1 after the entrant's successful RecordOfflinePayment", len(repo.byID))
	}
}

// TestRecordOfflinePayment_CompetitionEntryPayable_AllowsAssignedCompetitionAdmin
// mirrors TestRecordOfflinePayment_CompetitionEntryPayable_AllowsEntrant for
// the assigned-Competition-Admin path: an admin who is not the entrant may
// still record the offline payment on the entrant's behalf.
func TestRecordOfflinePayment_CompetitionEntryPayable_AllowsAssignedCompetitionAdmin(t *testing.T) {
	h, _ := newTestHandler("pay-1")

	resp, err := h.RecordOfflinePayment(ctxAs("admin-2"), &paymentsv1.RecordOfflinePaymentRequest{
		PayableType:                     paymentsv1.PayableType_PAYABLE_TYPE_COMPETITION_ENTRY,
		PayableId:                       fixtureCompetitionEntryID,
		Amount:                          competitionEntryFixtureAmount(),
		EntrantPlayerId:                 "player-1",
		AssignedCompetitionAdminUserIds: []string{"admin-1", "admin-2"},
	})
	if err != nil {
		t.Fatalf("RecordOfflinePayment(admin-2) (an assigned Competition Admin) should succeed, got: %v", err)
	}
	if want := resolvedUserID("admin-2"); resp.GetPayment().GetRecordedByUserId() != want {
		t.Errorf("Payment.RecordedByUserId = %q, want %q (admin-2's resolved User.ID)", resp.GetPayment().GetRecordedByUserId(), want)
	}
}

// --- CreateOnlinePayment / CompetitionEntry-payable: object-level (BOLA) --
//
// Unlike RecordOfflinePayment, CreateOnlinePayment carried no actor field
// at all before T10.6 — this is the first authorization check the online
// path has ever had, scoped to PAYABLE_TYPE_COMPETITION_ENTRY only (see
// app.CreateOnlinePaymentInput's doc comment for why the pre-existing
// payable types are deliberately left as they were).
//
// entrant_player_id/assigned_competition_admin_user_ids on the requests
// below are now [deprecated = true] and ignored by the server (T17.1, closes
// #198) — newTestHandlerWithProcessor's real entrant/admin set ("player-1"/
// "admin-1"/"admin-2", per newEntryAuthzResolverFixtures) is what actually
// decides these tests now, not these wire values. Left set anyway in the
// positive-path cases below purely to prove they have no effect (an
// un-migrated client sending them must not behave differently) — see
// TestCreateOnlinePayment_Handler_ForgedEntrantPlayerIdIgnored below for the
// case where they'd matter if honored.

// TestCreateOnlinePayment_CompetitionEntryPayable_RejectsMismatchedActor is
// this ticket's required test for the online path: a Player who is
// neither the entrant nor an assigned Competition Admin attempts
// CreateOnlinePayment against another player's CompetitionEntry, through
// the real handler -> app -> domain path, and the request is rejected with
// the correctly mapped status — not a 500, not a silent success.
func TestCreateOnlinePayment_CompetitionEntryPayable_RejectsMismatchedActor(t *testing.T) {
	h, repo := newTestHandlerWithProcessor("pay-1")

	_, err := h.CreateOnlinePayment(ctxAs("random-player"), &paymentsv1.CreateOnlinePaymentRequest{
		PayableType: paymentsv1.PayableType_PAYABLE_TYPE_COMPETITION_ENTRY,
		PayableId:   fixtureCompetitionEntryID,
		Amount:      competitionEntryFixtureAmount(),
		// nolint:staticcheck // SA1019: deprecated (T17.1) — set here to prove it has no effect.
		EntrantPlayerId: "player-1",
		// nolint:staticcheck // SA1019: deprecated (T17.1) — set here to prove it has no effect.
		AssignedCompetitionAdminUserIds: []string{"admin-1"},
	})
	if err == nil {
		t.Fatal("CreateOnlinePayment(random-player) succeeded silently — a Player who is neither the entrant nor an assigned Competition Admin was able to start an online payment against another player's CompetitionEntry (BOLA regression)")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("CreateOnlinePayment(random-player) returned a non-gRPC-status error: %v (a client can't map this to a clean HTTP status)", err)
	}
	if st.Code() == codes.Internal {
		t.Fatalf("CreateOnlinePayment(random-player) mapped to Internal (500-shaped) — want PermissionDenied (403-shaped): %v", err)
	}
	if st.Code() != codes.PermissionDenied {
		t.Fatalf("CreateOnlinePayment(random-player) status code = %v, want PermissionDenied (403-shaped)", st.Code())
	}

	if len(repo.byID) != 0 {
		t.Errorf("repo has %d Payments after a rejected CreateOnlinePayment, want 0 (the attacker's rejected attempt must not have any side effect)", len(repo.byID))
	}
}

// TestCreateOnlinePayment_CompetitionEntryPayable_AllowsEntrant is the
// symmetric positive-path case: the entrant starting an online payment for
// their own competition entry succeeds through the same handler path.
func TestCreateOnlinePayment_CompetitionEntryPayable_AllowsEntrant(t *testing.T) {
	h, repo := newTestHandlerWithProcessor("pay-1")

	resp, err := h.CreateOnlinePayment(ctxAs("player-1"), &paymentsv1.CreateOnlinePaymentRequest{
		PayableType: paymentsv1.PayableType_PAYABLE_TYPE_COMPETITION_ENTRY,
		PayableId:   fixtureCompetitionEntryID,
		Amount:      competitionEntryFixtureAmount(),
		// nolint:staticcheck // SA1019: deprecated (T17.1) — set here to prove it has no effect.
		EntrantPlayerId: "player-1",
	})
	if err != nil {
		t.Fatalf("CreateOnlinePayment(player-1) (the entrant) should succeed, got: %v", err)
	}
	if resp.GetPayment().GetStatus() != paymentsv1.PaymentStatus_PAYMENT_STATUS_UNPAID {
		t.Errorf("Payment.Status = %v, want PAYMENT_STATUS_UNPAID (creating an intent does not capture funds)", resp.GetPayment().GetStatus())
	}
	if len(repo.byID) != 1 {
		t.Errorf("repo has %d Payments, want 1 after the entrant's successful CreateOnlinePayment", len(repo.byID))
	}
}

// TestCreateOnlinePayment_Handler_ForgedEntrantPlayerIdIgnored is T17.1's
// headline forgery mutation check at the wire boundary (closes #198,
// instruction 5), the CreateOnlinePayment mirror of
// TestRefundPayment_Handler_ForgedGameHostIdIgnored (refund_test.go): a
// caller naming themselves via the deprecated, now-ignored
// entrant_player_id field is refused — the request below is byte-for-byte
// what would have SUCCEEDED before this ticket (an attacker who is neither
// the real entrant ("player-1") nor an assigned Competition Admin
// ("admin-1"/"admin-2", per newTestHandlerWithProcessor's resolver wiring)
// simply claiming entrant_player_id = their own subject), and must now fail
// exactly like TestCreateOnlinePayment_CompetitionEntryPayable_RejectsMismatchedActor's
// unrelated-actor case, because there is no live path left from this field
// to the authorization decision.
func TestCreateOnlinePayment_Handler_ForgedEntrantPlayerIdIgnored(t *testing.T) {
	h, repo := newTestHandlerWithProcessor("pay-1")

	const attacker = "attacker-forging-entrant-player-id"
	_, err := h.CreateOnlinePayment(ctxAs(attacker), &paymentsv1.CreateOnlinePaymentRequest{
		PayableType: paymentsv1.PayableType_PAYABLE_TYPE_COMPETITION_ENTRY,
		PayableId:   fixtureCompetitionEntryID,
		Amount:      competitionEntryFixtureAmount(),
		// nolint:staticcheck // SA1019: setting the deprecated field IS the test.
		EntrantPlayerId: attacker, // the forgery: claiming to be the entry's own entrant
	})
	if err == nil {
		t.Fatal("CreateOnlinePayment(attacker forging entrant_player_id) succeeded silently — the deprecated field still decided authorization (T17.1 regression)")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("CreateOnlinePayment(attacker) returned a non-gRPC-status error: %v (a client can't map this to a clean HTTP status)", err)
	}
	if st.Code() == codes.Internal {
		t.Fatalf("CreateOnlinePayment(attacker) mapped to Internal (500-shaped) — want PermissionDenied (403-shaped): %v", err)
	}
	if st.Code() != codes.PermissionDenied {
		t.Fatalf("CreateOnlinePayment(attacker) status code = %v, want PermissionDenied (403-shaped)", st.Code())
	}

	if len(repo.byID) != 0 {
		t.Errorf("repo has %d Payments after a forged CreateOnlinePayment, want 0 — a forged attempt must have no side effect", len(repo.byID))
	}
}

// TestCreateOnlinePayment_BookingPayable_StillNeedsNoActor is a regression
// guard mirroring TestRecordOfflinePayment_RejectionNotConflatedWithOtherErrors's
// spirit: the new competition_entry-only authorization check in
// CreateOnlinePayment must not start requiring an actor for the payable
// types this ticket does not touch.
// T12.8 narrowed this test's claim without changing what it is for. A booking
// payable still has no object-level actor check to satisfy — that is the T10.6
// property under test, and it is unchanged. What changed is that the RPC now
// requires the caller to be *authenticated* at all, so the request travels
// under a principal even though no authorization branch reads it. The two are
// different questions ("who are you" versus "may you"), and this test still
// answers only the second.
func TestCreateOnlinePayment_BookingPayable_StillNeedsNoActor(t *testing.T) {
	h, _ := newTestHandlerWithProcessor("pay-1")

	if _, err := h.CreateOnlinePayment(ctxAs("auth0|any-authenticated-caller"), &paymentsv1.CreateOnlinePaymentRequest{
		PayableType: paymentsv1.PayableType_PAYABLE_TYPE_BOOKING,
		PayableId:   fixtureBookingID,
		Amount:      competitionEntryFixtureAmount(),
	}); err != nil {
		t.Fatalf("CreateOnlinePayment for a booking payable with no actor fields should still succeed (unaffected by T10.6), got: %v", err)
	}
}
