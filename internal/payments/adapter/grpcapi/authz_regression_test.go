// T8.5 (closes issue #53) — object-level authorization regression tests for
// Payments' RecordOfflinePayment, run through the real gRPC handler (not
// just the app-level unit tests internal/payments/app/service_test.go
// already has for authorizeOfflineRecording). This is the "does the
// guarantee survive the full stack" test that T6.7 was supposed to add
// (docs/process/t6-sprint-plan.md, T6.7) but never did —
// HANDOFF.md's "Not yet built" section names it explicitly ("QA
// object-level-auth regression tests for Payments (T6.7) — no PR, not
// started"). This is the third instance of the pattern after T5.5 (Social
// Play, internal/socialplay/adapter/grpcapi/authz_regression_test.go) and
// T7.7 (Facilities, internal/facilities/adapter/grpcapi/authz_regression_test.go).
//
// Scope-check performed first, per this ticket's own instructions: the
// object-level check does NOT need to be added here. It already exists —
// app.authorizeOfflineRecording (internal/payments/app/service.go, T6.3)
// and the domain.ErrNotPaymentRecorder sentinel it returns, mapped by this
// package's toStatus to codes.PermissionDenied — was built and unit-tested
// at the app layer in the same T6.3 PR (see
// internal/payments/app/service_test.go's
// TestRecordOfflinePayment_BookingPayable_WrongActorRejected and
// TestRecordOfflinePayment_RegistrationPayable_UnassignedActorRejected).
// What was missing, and what T6.7/this ticket adds, is proof that the
// check survives the real grpcapi.Handler -> app.Service path end to end,
// including the gRPC status-code mapping — HANDOFF.md's own T7.7 entry
// already reflects this ("unlike T5.5/T6.7, which only needed a regression
// test proving an existing check").
//
// This is a handler-level test (real grpcapi.Handler + real app.Service +
// real domain, with an in-memory fake standing in for
// internal/payments/adapter/postgres) rather than a `-tags=integration`
// testcontainers-go test, mirroring T5.5's and T7.7's reasoning exactly:
//
//  1. The check under test lives entirely in app.authorizeOfflineRecording
//     (internal/payments/app/service.go), which port.Repository doesn't
//     influence — a real Postgres round trip would add infrastructure, not
//     proof.
//  2. This environment has no Docker daemon (the same gap
//     concurrency_integration_test.go's, T5.5's, and T7.7's own package
//     comments already document for this repo). CLAUDE.md rule 10 means
//     this ticket's own regression test needs to actually run in an
//     environment we can execute it in, not just be plausible in CI.
package grpcapi_test

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/nhuthuynh/white-label/internal/payments/adapter/grpcapi"
	"github.com/nhuthuynh/white-label/internal/payments/app"
	"github.com/nhuthuynh/white-label/internal/payments/domain"

	paymentsv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/payments/v1"
)

// fixtureBookingID/fixtureRegistrationID/fixtureCompetitionEntryID are
// deterministic, UUID-shaped PayableId fixtures shared by every test in this
// package (grpcapi_test) — including competition_entry_authz_regression_test.go.
// They used to be the literals "booking-1"/"reg-1"/"entry-1", a shape
// internal/platform/idgen never produces and T10.7's uuidShape guard on
// CreateOnlinePayment/RecordOfflinePayment correctly rejects (the same
// fixture-infidelity class docs/LESSONS.md's T9 entry already documents,
// and the identical fix already applied once in
// internal/payments/app/service_test.go — this package's own test fixtures
// were a separate, undiscovered instance of the same bug).
// fixtureGameID/fixtureCompetitionID (T16.2, closing #168) are the parent
// ids fixtureRegistrationID/fixtureCompetitionEntryID resolve to through the
// new resolver ports — see newAuthzResolverFixtures/newEntryAuthzResolverFixtures
// below, this package's mirror of internal/payments/app/service_test.go's
// identically-named fixtures and helpers.
const (
	fixtureBookingID          = "6ba7b810-0000-4000-8000-000000000001"
	fixtureRegistrationID     = "6ba7b810-0000-4000-8000-000000000002"
	fixtureCompetitionEntryID = "6ba7b810-0000-4000-8000-000000000003"
	fixtureGameID             = "6ba7b810-0000-4000-8000-000000000004"
	fixtureCompetitionID      = "6ba7b810-0000-4000-8000-000000000005"
)

// --- T16.2: authorization resolver port fakes (closes #168) ---------------
//
// Package-shared across every *_test.go file in grpcapi_test — these fake
// the five NEW ports at the same level authz_regression_test.go's own
// fakeRepository fakes port.Repository: a plain in-memory double of
// internal/payments' own port interfaces, not a fake of another context's
// app.Service. This package proves the handler -> app.Service wiring and the
// gRPC status-code mapping survive the real stack; it does not re-prove that
// the ADAPTERS correctly translate a real socialplayapp.Service/
// competitionsapp.Service call — that proof lives in
// internal/payments/adapter/socialplay and .../competitions (T15.5's
// existing game_admin_reader_test.go and T16.2's new
// registration_lookup_test.go/game_lookup_test.go/entry_lookup_test.go/
// authorization_boundary_test.go), per T14.8/T15.5's cross-context-fake
// warning.
type fakeRegistrationLookup struct{ gameByRegistration map[string]string }

func (f *fakeRegistrationLookup) GameIDForRegistration(_ context.Context, registrationID string) (string, error) {
	return f.gameByRegistration[registrationID], nil
}

type fakeGameLookup struct{ hostByGame map[string]string }

func (f *fakeGameLookup) HostIDForGame(_ context.Context, gameID string) (string, error) {
	return f.hostByGame[gameID], nil
}

type fakeGameAdminReader struct{ adminsByGame map[string][]string }

func (f *fakeGameAdminReader) ListGameAdmins(_ context.Context, gameID string) ([]string, error) {
	return f.adminsByGame[gameID], nil
}

type fakeEntryLookup struct {
	competitionByEntry map[string]string
	playerByEntry      map[string]string
}

func (f *fakeEntryLookup) CompetitionIDAndPlayerIDForEntry(_ context.Context, entryID string) (string, string, error) {
	return f.competitionByEntry[entryID], f.playerByEntry[entryID], nil
}

type fakeCompetitionAdminReader struct{ adminsByCompetition map[string][]string }

func (f *fakeCompetitionAdminReader) ListCompetitionAdmins(_ context.Context, competitionID string) ([]string, error) {
	return f.adminsByCompetition[competitionID], nil
}

// newAuthzResolverFixtures builds the three resolver fakes
// authorizeGameRecording (RecordOfflinePayment/RefundPayment's
// PayableTypeRegistration/PayableTypeNoShowFee branch) needs, pre-wired so
// fixtureRegistrationID resolves to fixtureGameID, hostID is that Game's
// Host, and admins is its current Game-Admin set — the handler-level
// replacement for what a request used to establish directly via the
// now-ignored game_host_id/assigned_game_admin_user_ids wire fields.
func newAuthzResolverFixtures(hostID string, admins ...string) (*fakeRegistrationLookup, *fakeGameLookup, *fakeGameAdminReader) {
	regs := &fakeRegistrationLookup{gameByRegistration: map[string]string{fixtureRegistrationID: fixtureGameID}}
	games := &fakeGameLookup{hostByGame: map[string]string{fixtureGameID: hostID}}
	gameAdmins := &fakeGameAdminReader{adminsByGame: map[string][]string{fixtureGameID: admins}}
	return regs, games, gameAdmins
}

// newEntryAuthzResolverFixtures is newAuthzResolverFixtures' Competitions
// mirror: pre-wires fixtureCompetitionEntryID to resolve to
// fixtureCompetitionID and playerID in one call, and admins as that
// Competition's current Competition-Admin set.
func newEntryAuthzResolverFixtures(playerID string, admins ...string) (*fakeEntryLookup, *fakeCompetitionAdminReader) {
	entries := &fakeEntryLookup{
		competitionByEntry: map[string]string{fixtureCompetitionEntryID: fixtureCompetitionID},
		playerByEntry:      map[string]string{fixtureCompetitionEntryID: playerID},
	}
	competitionAdmins := &fakeCompetitionAdminReader{adminsByCompetition: map[string][]string{fixtureCompetitionID: admins}}
	return entries, competitionAdmins
}

// --- in-memory port.Repository fake -----------------------------------
//
// Stands in for internal/payments/adapter/postgres for this test only.
// Implements the exact same port.Repository interface the real Postgres
// adapter does, so app.Service and grpcapi.Handler run unmodified, real
// production code — only the persistence boundary is faked. Mirrors
// internal/payments/app/fixtures_test.go's fakeRepository and
// internal/facilities/adapter/grpcapi/authz_regression_test.go's fakeRepo.
type fakeRepository struct {
	byID      map[string]domain.Payment
	byPayable map[string]string // "payableType:payableID" -> payment id
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{
		byID:      map[string]domain.Payment{},
		byPayable: map[string]string{},
	}
}

func payableKey(t domain.PayableType, id string) string {
	return string(t) + ":" + id
}

func (r *fakeRepository) Create(_ context.Context, p domain.Payment) (domain.Payment, error) {
	key := payableKey(p.PayableType, p.PayableID)
	if _, exists := r.byPayable[key]; exists {
		return domain.Payment{}, domain.ErrPaymentAlreadyRecorded
	}
	r.byPayable[key] = p.ID
	r.byID[p.ID] = p
	return p, nil
}

func (r *fakeRepository) GetByID(_ context.Context, id string) (domain.Payment, error) {
	p, ok := r.byID[id]
	if !ok {
		return domain.Payment{}, domain.ErrPaymentNotFound
	}
	return p, nil
}

func (r *fakeRepository) Update(_ context.Context, p domain.Payment) (domain.Payment, error) {
	if _, ok := r.byID[p.ID]; !ok {
		return domain.Payment{}, domain.ErrPaymentNotFound
	}
	r.byID[p.ID] = p
	return p, nil
}

// fixedIDs is a deterministic, dependency-free port.IDGenerator stand-in,
// mirroring internal/socialplay/adapter/grpcapi/authz_regression_test.go's
// fakeIDs and internal/payments/app/fixtures_test.go's fixedIDs.
type fixedIDs struct {
	ids  []string
	next int
}

func (f *fixedIDs) NewID() string {
	if f.next >= len(f.ids) {
		panic("fixedIDs: ran out of seeded ids")
	}
	id := f.ids[f.next]
	f.next++
	return id
}

// newTestHandler wires the real app.Service and the real grpcapi.Handler —
// exactly what cmd/server wires in production — against an in-memory
// fakeRepository. No PaymentProcessor/RegistrationUpdater is needed: the
// RPC under test, RecordOfflinePayment, doesn't touch either (offline
// recording has no processor step, and the reconciliation call to Social
// Play is a side effect that fires only on success — irrelevant to proving
// a rejected actor gets no side effects at all).
//
// T16.2 (closes #168): the five resolver ports ARE wired, with a fixed
// default every existing test in this file and competition_entry_authz_regression_test.go
// already assumes — "host-1"/"admin-1"/"admin-2" as fixtureGameID's Host/
// admin set, "player-1" as fixtureCompetitionEntryID's entrant with the same
// admin set — because game_host_id/assigned_game_admin_user_ids/
// entrant_player_id/assigned_competition_admin_user_ids on the wire request
// are now ignored (see grpcapi.Handler.RecordOfflinePayment's doc comment):
// the equivalent facts must come from these resolvers instead, or every
// positive-path test below would start failing closed. Tests that need a
// DIFFERENT resolved Host/entrant (principal_authz_test.go, whose subjects
// don't match these literals) use their own dedicated constructor —
// newPrincipalAuthzHandler — rather than overloading this one.
func newTestHandler(seedIDs ...string) (*grpcapi.Handler, *fakeRepository) {
	repo := newFakeRepository()
	regs, games, gameAdmins := newAuthzResolverFixtures("host-1", "admin-1", "admin-2")
	entries, compAdmins := newEntryAuthzResolverFixtures("player-1", "admin-1", "admin-2")
	svc := app.NewService(app.ServiceOptions{
		Payments:               repo,
		IDs:                    &fixedIDs{ids: seedIDs},
		RegistrationLookup:     regs,
		GameLookup:             games,
		GameAdminReader:        gameAdmins,
		EntryLookup:            entries,
		CompetitionAdminReader: compAdmins,
	})
	return grpcapi.NewHandler(svc), repo
}

func offlineFixtureAmount() *paymentsv1.Money {
	return &paymentsv1.Money{AmountCents: 2500, CurrencyCode: "USD"}
}

// --- RecordOfflinePayment / Registration-payable: object-level (BOLA) -----

// TestRecordOfflinePayment_RegistrationPayable_RejectsMismatchedActor is the
// ticket's required test: a Player who is neither the Game's Host nor an
// assigned Game Admin attempts RecordOfflinePayment against another
// player's Registration, through the real handler -> app -> domain path,
// and the request is rejected with the correctly mapped status — not a
// 500, not a silent success.
func TestRecordOfflinePayment_RegistrationPayable_RejectsMismatchedActor(t *testing.T) {
	h, repo := newTestHandler("pay-1")

	// The BOLA attempt: "random-player", neither game-1's Host nor one of
	// its assigned Game Admins, tries to record a Payment against fixtureRegistrationID.
	_, err := h.RecordOfflinePayment(ctxAs("random-player"), &paymentsv1.RecordOfflinePaymentRequest{
		PayableType:              paymentsv1.PayableType_PAYABLE_TYPE_REGISTRATION,
		PayableId:                fixtureRegistrationID,
		Amount:                   offlineFixtureAmount(),
		GameHostId:               "host-1",
		AssignedGameAdminUserIds: []string{"admin-1"},
	})
	if err == nil {
		t.Fatal("RecordOfflinePayment(random-player) succeeded silently — a Player who is neither Host nor an assigned Game Admin was able to record a payment against another player's Registration (BOLA regression)")
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

// TestRecordOfflinePayment_RegistrationPayable_AllowsGameHost is the
// symmetric positive-path case for the Game's Host: without it,
// TestRecordOfflinePayment_RegistrationPayable_RejectsMismatchedActor alone
// couldn't tell "the ownership check correctly rejects a mismatched actor"
// apart from "RecordOfflinePayment is broken and rejects everyone" — this
// pins down that the real Host's recording still succeeds through the same
// handler path.
func TestRecordOfflinePayment_RegistrationPayable_AllowsGameHost(t *testing.T) {
	h, repo := newTestHandler("pay-1")

	resp, err := h.RecordOfflinePayment(ctxAs("host-1"), &paymentsv1.RecordOfflinePaymentRequest{
		PayableType: paymentsv1.PayableType_PAYABLE_TYPE_REGISTRATION,
		PayableId:   fixtureRegistrationID,
		Amount:      offlineFixtureAmount(),
		GameHostId:  "host-1",
	})
	if err != nil {
		t.Fatalf("RecordOfflinePayment(host-1) (the Game's Host) should succeed, got: %v", err)
	}
	if resp.GetPayment().GetStatus() != paymentsv1.PaymentStatus_PAYMENT_STATUS_PAID {
		t.Errorf("Payment.Status = %v, want PAYMENT_STATUS_PAID", resp.GetPayment().GetStatus())
	}
	if len(repo.byID) != 1 {
		t.Errorf("repo has %d Payments, want 1 after the Host's successful RecordOfflinePayment", len(repo.byID))
	}
}

// TestRecordOfflinePayment_RegistrationPayable_AllowsAssignedGameAdmin
// mirrors TestRecordOfflinePayment_RegistrationPayable_AllowsGameHost for
// the assigned-Game-Admin path (the glossary's Game Admin scope,
// docs/agent-operating-handbook.md A2): a Game Admin who is not the Host
// may still record the offline payment.
func TestRecordOfflinePayment_RegistrationPayable_AllowsAssignedGameAdmin(t *testing.T) {
	h, _ := newTestHandler("pay-1")

	resp, err := h.RecordOfflinePayment(ctxAs("admin-2"), &paymentsv1.RecordOfflinePaymentRequest{
		PayableType:              paymentsv1.PayableType_PAYABLE_TYPE_REGISTRATION,
		PayableId:                fixtureRegistrationID,
		Amount:                   offlineFixtureAmount(),
		GameHostId:               "host-1",
		AssignedGameAdminUserIds: []string{"admin-1", "admin-2"},
	})
	if err != nil {
		t.Fatalf("RecordOfflinePayment(admin-2) (an assigned Game Admin) should succeed, got: %v", err)
	}
	if resp.GetPayment().GetRecordedByUserId() != "admin-2" {
		t.Errorf("Payment.RecordedByUserId = %q, want admin-2", resp.GetPayment().GetRecordedByUserId())
	}
}

// --- RecordOfflinePayment / Booking-payable: object-level (BOLA) ----------

// TestRecordOfflinePayment_BookingPayable_RejectsMismatchedActor is the
// ticket's required test for the Booking-payable path: a user who is not
// the Booking's owning Host attempts RecordOfflinePayment against it,
// through the real handler -> app -> domain path, and the request is
// rejected with the correctly mapped status — not a 500, not a silent
// success.
func TestRecordOfflinePayment_BookingPayable_RejectsMismatchedActor(t *testing.T) {
	h, repo := newTestHandler("pay-1")

	// The BOLA attempt: "some-other-player", not fixtureBookingID's owning Host,
	// tries to record a Payment against it.
	_, err := h.RecordOfflinePayment(ctxAs("some-other-player"), &paymentsv1.RecordOfflinePaymentRequest{
		PayableType:   paymentsv1.PayableType_PAYABLE_TYPE_BOOKING,
		PayableId:     fixtureBookingID,
		Amount:        offlineFixtureAmount(),
		BookingHostId: "host-1",
	})
	if err == nil {
		t.Fatal("RecordOfflinePayment(some-other-player) succeeded silently — a non-Host was able to record a payment against fixtureBookingID (BOLA regression)")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("RecordOfflinePayment(some-other-player) returned a non-gRPC-status error: %v (a client can't map this to a clean HTTP status)", err)
	}
	if st.Code() == codes.Internal {
		t.Fatalf("RecordOfflinePayment(some-other-player) mapped to Internal (500-shaped) — want PermissionDenied (403-shaped): %v", err)
	}
	if st.Code() != codes.PermissionDenied {
		t.Fatalf("RecordOfflinePayment(some-other-player) status code = %v, want PermissionDenied (403-shaped)", st.Code())
	}

	if len(repo.byID) != 0 {
		t.Errorf("repo has %d Payments after a rejected RecordOfflinePayment, want 0 (the attacker's rejected attempt must not have any side effect)", len(repo.byID))
	}
}

// TestRecordOfflinePayment_BookingPayable_AllowsOwningHost is the symmetric
// positive-path case for the Booking-payable path: the real owning Host's
// recording still succeeds through the same handler path.
func TestRecordOfflinePayment_BookingPayable_AllowsOwningHost(t *testing.T) {
	h, repo := newTestHandler("pay-1")

	resp, err := h.RecordOfflinePayment(ctxAs("host-1"), &paymentsv1.RecordOfflinePaymentRequest{
		PayableType:   paymentsv1.PayableType_PAYABLE_TYPE_BOOKING,
		PayableId:     fixtureBookingID,
		Amount:        offlineFixtureAmount(),
		BookingHostId: "host-1",
	})
	if err != nil {
		t.Fatalf("RecordOfflinePayment(host-1) (the Booking's owning Host) should succeed, got: %v", err)
	}
	if resp.GetPayment().GetStatus() != paymentsv1.PaymentStatus_PAYMENT_STATUS_PAID {
		t.Errorf("Payment.Status = %v, want PAYMENT_STATUS_PAID", resp.GetPayment().GetStatus())
	}
	if len(repo.byID) != 1 {
		t.Errorf("repo has %d Payments, want 1 after the Host's successful RecordOfflinePayment", len(repo.byID))
	}
}

// TestRecordOfflinePayment_RejectionNotConflatedWithOtherErrors is a
// belt-and-braces check that toStatus's PermissionDenied mapping is
// specific to domain.ErrNotPaymentRecorder — not, say, every non-nil error
// from RecordOfflinePayment. Combined with the two Rejects tests above (BOLA
// -> PermissionDenied) and the two Allows tests (owner -> success), this
// closes the loop that the ticket's "not a 500, not a silent success" bar
// asks for on both payable paths.
func TestRecordOfflinePayment_RejectionNotConflatedWithOtherErrors(t *testing.T) {
	h, _ := newTestHandler("pay-1")

	_, err := h.RecordOfflinePayment(ctxAs("host-1"), &paymentsv1.RecordOfflinePaymentRequest{
		PayableType:   paymentsv1.PayableType_PAYABLE_TYPE_BOOKING,
		PayableId:     fixtureBookingID,
		Amount:        &paymentsv1.Money{AmountCents: 0, CurrencyCode: "USD"}, // invalid amount
		BookingHostId: "host-1",
	})
	if err == nil {
		t.Fatal("expected an error for a zero amount")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("returned a non-gRPC-status error: %v", err)
	}
	if st.Code() == codes.PermissionDenied {
		t.Fatalf("an authorized actor's invalid-amount request mapped to PermissionDenied, want InvalidArgument — the ownership check must not be conflated with unrelated validation errors: %v", err)
	}
	if st.Code() != codes.InvalidArgument {
		t.Fatalf("status code = %v, want InvalidArgument", st.Code())
	}
}
