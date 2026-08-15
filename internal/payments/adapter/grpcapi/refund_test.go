// T12.3 — handler-level tests for the RefundPayment RPC, proving the gRPC
// status-code mapping the ticket specifies survives the real
// grpcapi.Handler -> app.Service -> domain path (not just the app-layer
// unit tests in internal/payments/app/refund_test.go).
//
// Handler-level with an in-memory port.Repository fake rather than a
// `-tags=integration` testcontainers-go test, for the reason this package's
// authz_regression_test.go already documents at length: nothing under test
// here is influenced by port.Repository's implementation — a real Postgres
// round trip would add infrastructure, not proof — and this environment has
// no Docker daemon, so CLAUDE.md rule 10 means the test has to actually be
// runnable where it was written.
package grpcapi_test

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/nhuthuynh/white-label/internal/payments/adapter/grpcapi"
	"github.com/nhuthuynh/white-label/internal/payments/adapter/stripestub"
	"github.com/nhuthuynh/white-label/internal/payments/app"
	"github.com/nhuthuynh/white-label/internal/payments/domain"

	paymentsv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/payments/v1"
)

// UUID-shaped Payment id fixtures — RefundPayment resolves the Payment
// through app.Service.GetPayment's uuidShape guard, so a "pay-1"-shaped id
// would be rejected as malformed before the repository is consulted.
const (
	refundBookingPaymentID     = "6ba7b810-0000-4000-8000-0000000000b1"
	refundRegistrationPayentID = "6ba7b810-0000-4000-8000-0000000000b2"
	refundNoShowPaymentID      = "6ba7b810-0000-4000-8000-0000000000b3"
	refundUnknownPaymentID     = "6ba7b810-0000-4000-8000-0000000000bf"

	refundBookingHostID    = "host-booking-9"
	refundGameHostID       = "host-game-9"
	refundOtherPlayerID    = "player-outsider-9"
	refundEntrantPlayerID  = "player-entrant-9"
	refundCompAdminUserID  = "admin-competition-9"
	refundCompetitionPayID = "6ba7b810-0000-4000-8000-0000000000b4"

	// seededPaymentOwnerID is the actor the seed helpers below record as the
	// Payment's own (RecordedByUserID) — the value CreateOnlinePayment would
	// have written from the verified principal in production (T13.7).
	// RefundPayment does not consult it (it authorizes against the
	// caller-supplied Host/Game-Admin facts), but ConfirmOnlinePayment does,
	// so a seeded Payment without one is a Payment nobody may capture — which
	// would silently turn any confirm-driven test into a PermissionDenied test.
	seededPaymentOwnerID = "payer-online-9"
)

// newRefundTestHandler wires the real app.Service and real grpcapi.Handler
// against an in-memory repository and the deterministic stub processor —
// the same production code cmd/server wires, with only the persistence and
// processor boundaries faked.
//
// T16.2 (closes #168): also wires RegistrationLookup/GameLookup/
// GameAdminReader, with refundGameHostID pre-wired as fixtureRegistrationID's
// Game's resolved Host (mirroring newAuthzResolverFixtures'
// authz_regression_test.go pattern) — game_host_id on the wire is now
// ignored, so this is what makes TestRefundPayment_Handler_NonRecorderActorPermissionDenied
// below a genuine "the real Host would succeed, this actor doesn't match"
// negative test rather than a fail-closed-because-nothing-is-wired one.
//
// T16.4 (closes the corrected #125): also wires EntryLookup/
// CompetitionAdminReader via newEntryAuthzResolverFixtures, with
// refundEntrantPlayerID pre-wired as fixtureCompetitionEntryID's resolved
// entrant — the Competitions mirror of the Registration wiring above, needed
// now that RefundPayment admits PayableTypeCompetitionEntry. Without this, a
// competition_entry refund test would fail closed for the wrong reason (no
// resolver wired) rather than proving the real authorized-actor set.
func newRefundTestHandler() (*grpcapi.Handler, *fakeRepository, *stripestub.Processor) {
	repo := newFakeRepository()
	proc := stripestub.NewProcessor()
	regs, games, admins := newAuthzResolverFixtures(refundGameHostID)
	entries, compAdmins := newEntryAuthzResolverFixtures(refundEntrantPlayerID)
	svc := app.NewService(app.ServiceOptions{
		Payments:               repo,
		IDs:                    &fixedIDs{},
		Processor:              proc,
		RegistrationLookup:     regs,
		GameLookup:             games,
		GameAdminReader:        admins,
		EntryLookup:            entries,
		CompetitionAdminReader: compAdmins,
	})
	return grpcapi.NewHandler(svc), repo, proc
}

// seedPaidOnline puts a paid, online Payment in the repository carrying an
// intent reference the stub processor really issued and captured, so a
// refund against it succeeds at the processor for the right reason.
func seedPaidOnline(t *testing.T, repo *fakeRepository, proc *stripestub.Processor, paymentID string, payableType domain.PayableType, payableID string) {
	t.Helper()

	ctx := context.Background()
	amount := domain.Money{Cents: 3000, Currency: "USD"}
	ref, err := proc.CreateIntent(ctx, amount, payableID)
	if err != nil {
		t.Fatalf("seed: CreateIntent: %v", err)
	}
	if err := proc.CapturePayment(ctx, ref); err != nil {
		t.Fatalf("seed: CapturePayment: %v", err)
	}

	if _, err := repo.Create(ctx, domain.Payment{
		ID:               paymentID,
		PayableType:      payableType,
		PayableID:        payableID,
		Amount:           amount,
		Method:           domain.MethodOnline,
		Status:           domain.StatusPaid,
		StripeReference:  ref,
		RecordedByUserID: seededPaymentOwnerID,
	}); err != nil {
		t.Fatalf("seed: Create: %v", err)
	}
}

func wantCode(t *testing.T, err error, want codes.Code) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected an error with code %v, got nil", want)
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("error %v is not a gRPC status", err)
	}
	if st.Code() != want {
		t.Fatalf("status code = %v, want %v (err: %v)", st.Code(), want, err)
	}
}

// TestRefundPayment_Handler_BookingHostSucceeds is the happy path through
// the real handler: the response carries the Payment with the refunded
// status, mapped onto the proto enum.
func TestRefundPayment_Handler_BookingHostSucceeds(t *testing.T) {
	t.Parallel()

	h, repo, proc := newRefundTestHandler()
	seedPaidOnline(t, repo, proc, refundBookingPaymentID, domain.PayableTypeBooking, fixtureBookingID)

	resp, err := h.RefundPayment(ctxAs(refundBookingHostID), &paymentsv1.RefundPaymentRequest{
		PaymentId:     refundBookingPaymentID,
		BookingHostId: refundBookingHostID,
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got := resp.GetPayment().GetStatus(); got != paymentsv1.PaymentStatus_PAYMENT_STATUS_REFUNDED {
		t.Fatalf("status = %v, want PAYMENT_STATUS_REFUNDED", got)
	}
}

// TestRefundPayment_Handler_NonRecorderActorPermissionDenied is the
// object-level authorization regression test at the handler boundary: a
// player who is neither Host nor assigned Game Admin gets PermissionDenied
// — not Internal, and certainly not a silent success.
func TestRefundPayment_Handler_NonRecorderActorPermissionDenied(t *testing.T) {
	t.Parallel()

	h, repo, proc := newRefundTestHandler()
	seedPaidOnline(t, repo, proc, refundRegistrationPayentID, domain.PayableTypeRegistration, fixtureRegistrationID)

	_, err := h.RefundPayment(ctxAs(refundOtherPlayerID), &paymentsv1.RefundPaymentRequest{
		PaymentId:  refundRegistrationPayentID,
		GameHostId: refundGameHostID,
	})
	wantCode(t, err, codes.PermissionDenied)

	stored, getErr := repo.GetByID(context.Background(), refundRegistrationPayentID)
	if getErr != nil {
		t.Fatalf("unexpected err: %v", getErr)
	}
	if stored.Status != domain.StatusPaid {
		t.Fatalf("persisted status = %v, want paid — a denied refund must have no side effect", stored.Status)
	}
}

// TestRefundPayment_Handler_RegistrationPayable_GameHostSucceeds is T16.2's
// headline positive control at the handler boundary (instruction 9,
// mirroring TestRefundPayment_Handler_NonRecorderActorPermissionDenied's
// negative case above): the real, resolver-established Host of
// fixtureRegistrationID's Game (refundGameHostID, per newRefundTestHandler)
// succeeds — proving the two tests together distinguish "the check correctly
// rejects a mismatched actor" from "RefundPayment is broken and rejects
// everyone", the same pairing authz_regression_test.go's Allows/Rejects
// tests already establish for RecordOfflinePayment.
func TestRefundPayment_Handler_RegistrationPayable_GameHostSucceeds(t *testing.T) {
	t.Parallel()

	h, repo, proc := newRefundTestHandler()
	seedPaidOnline(t, repo, proc, refundRegistrationPayentID, domain.PayableTypeRegistration, fixtureRegistrationID)

	resp, err := h.RefundPayment(ctxAs(refundGameHostID), &paymentsv1.RefundPaymentRequest{
		PaymentId: refundRegistrationPayentID,
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got := resp.GetPayment().GetStatus(); got != paymentsv1.PaymentStatus_PAYMENT_STATUS_REFUNDED {
		t.Fatalf("status = %v, want PAYMENT_STATUS_REFUNDED", got)
	}
}

// TestRefundPayment_Handler_ForgedGameHostIdIgnored is T16.2's headline
// forgery mutation check at the wire boundary (instruction 9): a caller
// naming themselves via the deprecated, now-ignored game_host_id field is
// refused — the request below is byte-for-byte what would have SUCCEEDED
// before this ticket (an attacker who is neither the real Host nor an
// assigned Game Admin simply claiming game_host_id = their own subject), and
// must now fail exactly like
// TestRefundPayment_Handler_NonRecorderActorPermissionDenied's unrelated-actor
// case, because there is no live path left from this field to the
// authorization decision.
func TestRefundPayment_Handler_ForgedGameHostIdIgnored(t *testing.T) {
	t.Parallel()

	h, repo, proc := newRefundTestHandler()
	seedPaidOnline(t, repo, proc, refundRegistrationPayentID, domain.PayableTypeRegistration, fixtureRegistrationID)

	const attacker = "attacker-forging-game-host-id"
	_, err := h.RefundPayment(ctxAs(attacker), &paymentsv1.RefundPaymentRequest{
		PaymentId: refundRegistrationPayentID,
		// nolint:staticcheck // SA1019: setting the deprecated field IS the test.
		GameHostId: attacker, // the forgery: claiming to be the Game's own Host
	})
	wantCode(t, err, codes.PermissionDenied)

	stored, getErr := repo.GetByID(context.Background(), refundRegistrationPayentID)
	if getErr != nil {
		t.Fatalf("unexpected err: %v", getErr)
	}
	if stored.Status != domain.StatusPaid {
		t.Fatalf("persisted status = %v, want paid — a forged refund must have no side effect", stored.Status)
	}
}

// TestRefundPayment_Handler_UnknownAndMalformedIDsNotFound proves both an
// unknown and a malformed payment_id answer NotFound — the malformed one
// without ever reaching the repository (and so without reaching the
// Postgres adapter's mustUUID, which panics on non-UUID input).
func TestRefundPayment_Handler_UnknownAndMalformedIDsNotFound(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		paymentID string
	}{
		{name: "well-formed but unknown", paymentID: refundUnknownPaymentID},
		{name: "malformed", paymentID: "not-a-uuid"},
		{name: "empty", paymentID: ""},
		{name: "sql-injection-shaped", paymentID: "' OR 1=1 --"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h, _, _ := newRefundTestHandler()

			_, err := h.RefundPayment(ctxAs(refundBookingHostID), &paymentsv1.RefundPaymentRequest{
				PaymentId:     tc.paymentID,
				BookingHostId: refundBookingHostID,
			})
			wantCode(t, err, codes.NotFound)
		})
	}
}

// TestRefundPayment_Handler_IllegalTransitionFailedPrecondition covers both
// illegal transitions the ticket names — already refunded, and never paid —
// and pins them to FailedPrecondition rather than InvalidArgument.
//
// FailedPrecondition is the semantically right code here: the request is
// perfectly well-formed, it's the *system state* that makes the operation
// illegal, and the caller shouldn't retry until that state changes.
//
// T13.9 (closing #131) resolved the divergence this comment used to
// describe: ErrIllegalStatusTransition now maps to FailedPrecondition for
// *every* RPC in this service, not just RefundPayment, so this test no
// longer pins a per-RPC exception. The service-wide version of this
// assertion lives in error_mapping_test.go.
func TestRefundPayment_Handler_IllegalTransitionFailedPrecondition(t *testing.T) {
	t.Parallel()

	t.Run("already refunded", func(t *testing.T) {
		t.Parallel()

		h, repo, proc := newRefundTestHandler()
		seedPaidOnline(t, repo, proc, refundBookingPaymentID, domain.PayableTypeBooking, fixtureBookingID)

		req := &paymentsv1.RefundPaymentRequest{
			PaymentId:     refundBookingPaymentID,
			BookingHostId: refundBookingHostID,
		}
		if _, err := h.RefundPayment(ctxAs(refundBookingHostID), req); err != nil {
			t.Fatalf("first refund should succeed: %v", err)
		}

		_, err := h.RefundPayment(ctxAs(refundBookingHostID), req)
		wantCode(t, err, codes.FailedPrecondition)
	})

	t.Run("never paid", func(t *testing.T) {
		t.Parallel()

		h, repo, _ := newRefundTestHandler()
		if _, err := repo.Create(context.Background(), domain.Payment{
			ID:          refundBookingPaymentID,
			PayableType: domain.PayableTypeBooking,
			PayableID:   fixtureBookingID,
			Amount:      domain.Money{Cents: 3000, Currency: "USD"},
			Method:      domain.MethodOffline,
			Status:      domain.StatusUnpaid,
		}); err != nil {
			t.Fatalf("seed: %v", err)
		}

		_, err := h.RefundPayment(ctxAs(refundBookingHostID), &paymentsv1.RefundPaymentRequest{
			PaymentId:     refundBookingPaymentID,
			BookingHostId: refundBookingHostID,
		})
		wantCode(t, err, codes.FailedPrecondition)
	})
}

// TestRefundPayment_Handler_ProcessorFailureUnavailableAndNotRefunded is
// T12.3's required processor-failure assertion at the handler boundary, with
// T13.9's corrected code: the caller sees Unavailable, and — the part that
// actually matters, and the part T13.9 did not touch — the Payment is still
// paid, not silently advanced to refunded. A platform that marks a refund it
// never made is one that believes it returned money it still holds.
//
// T12.3 pinned Internal here. T13.9 (closing #131) changed it to Unavailable
// service-wide: ErrPaymentProcessorUnavailable means the processor could not
// complete the request — in both the capture and refund directions nothing
// moved, so a retry is safe, which is Unavailable's contract. Internal is for
// broken invariants in our own system. The persistence assertion below is
// what actually guarantees a failed refund is not mistaken for a completed
// one; the status code is how the caller is told, not what protects the money.
func TestRefundPayment_Handler_ProcessorFailureUnavailableAndNotRefunded(t *testing.T) {
	t.Parallel()

	h, repo, _ := newRefundTestHandler()

	// A paid online Payment whose intent reference the stub processor never
	// issued — exactly what a real adapter reports for an unknown or stale
	// intent.
	if _, err := repo.Create(context.Background(), domain.Payment{
		ID:              refundBookingPaymentID,
		PayableType:     domain.PayableTypeBooking,
		PayableID:       fixtureBookingID,
		Amount:          domain.Money{Cents: 3000, Currency: "USD"},
		Method:          domain.MethodOnline,
		Status:          domain.StatusPaid,
		StripeReference: "pi_stub_never_issued",
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	_, err := h.RefundPayment(ctxAs(refundBookingHostID), &paymentsv1.RefundPaymentRequest{
		PaymentId:     refundBookingPaymentID,
		BookingHostId: refundBookingHostID,
	})
	wantCode(t, err, codes.Unavailable)

	stored, getErr := repo.GetByID(context.Background(), refundBookingPaymentID)
	if getErr != nil {
		t.Fatalf("unexpected err: %v", getErr)
	}
	if stored.Status != domain.StatusPaid {
		t.Fatalf("persisted status = %v, want paid — a failed processor refund must not mark the Payment refunded", stored.Status)
	}
}

// TestRefundPayment_Handler_OutOfScopePayableTypeInvalidArgument pins the
// remaining scope boundary at the wire: no_show_fee (issue #130, which
// carries a genuinely open product question, unlike #125) is rejected
// rather than silently refunded.
//
// competition_entry moved OUT of this table at T16.4 (closes the corrected
// #125) — see TestRefundPayment_Handler_CompetitionEntryPayable_EntrantSucceeds
// below for its accepted-case counterpart. Moved, not deleted: a case that
// simply vanished here would prove nothing had been checked.
func TestRefundPayment_Handler_OutOfScopePayableTypeInvalidArgument(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		payableType domain.PayableType
		payableID   string
	}{
		{name: "no_show_fee (#130)", payableType: domain.PayableTypeNoShowFee, payableID: fixtureRegistrationID},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h, repo, proc := newRefundTestHandler()
			seedPaidOnline(t, repo, proc, refundNoShowPaymentID, tc.payableType, tc.payableID)

			_, err := h.RefundPayment(ctxAs(refundGameHostID), &paymentsv1.RefundPaymentRequest{
				PaymentId:  refundNoShowPaymentID,
				GameHostId: refundGameHostID,
			})
			wantCode(t, err, codes.InvalidArgument)

			stored, getErr := repo.GetByID(context.Background(), refundNoShowPaymentID)
			if getErr != nil {
				t.Fatalf("unexpected err: %v", getErr)
			}
			if stored.Status != domain.StatusPaid {
				t.Fatalf("persisted status = %v, want paid", stored.Status)
			}
		})
	}
}

// TestRefundPayment_Handler_CompetitionEntryPayable_EntrantSucceeds is
// T16.4's accepted-case counterpart to the now-removed "competition_entry
// (#125)" case in TestRefundPayment_Handler_OutOfScopePayableTypeInvalidArgument
// above — the same payable type, run through the real handler, now expected
// to succeed rather than InvalidArgument: RefundPayment admits
// PayableTypeCompetitionEntry (closes the corrected #125), reusing
// authorizeOfflineRecording's existing PayableTypeCompetitionEntry branch,
// so the resolved entrant (refundEntrantPlayerID, per
// newRefundTestHandler's EntryLookup/CompetitionAdminReader wiring) may
// refund their own entry's Payment exactly as a Registration's Game Host
// already can.
func TestRefundPayment_Handler_CompetitionEntryPayable_EntrantSucceeds(t *testing.T) {
	t.Parallel()

	h, repo, proc := newRefundTestHandler()
	seedPaidOnline(t, repo, proc, refundCompetitionPayID, domain.PayableTypeCompetitionEntry, fixtureCompetitionEntryID)

	resp, err := h.RefundPayment(ctxAs(refundEntrantPlayerID), &paymentsv1.RefundPaymentRequest{
		PaymentId: refundCompetitionPayID,
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got := resp.GetPayment().GetStatus(); got != paymentsv1.PaymentStatus_PAYMENT_STATUS_REFUNDED {
		t.Fatalf("status = %v, want PAYMENT_STATUS_REFUNDED", got)
	}

	stored, getErr := repo.GetByID(context.Background(), refundCompetitionPayID)
	if getErr != nil {
		t.Fatalf("unexpected err: %v", getErr)
	}
	if stored.Status != domain.StatusRefunded {
		t.Fatalf("persisted status = %v, want refunded", stored.Status)
	}
}

// TestRefundPayment_Handler_CompetitionEntryPayable_AssignedAdminSucceeds
// proves the refund reuses authorizeCompetitionEntryRecording's existing
// authorized-actor set exactly (entrant OR assigned Competition Admin, not
// entrant only) — mirroring
// TestRefundPayment_Handler_RegistrationPayable_GameHostSucceeds's own
// Host-or-Admin pairing on the Registration side.
func TestRefundPayment_Handler_CompetitionEntryPayable_AssignedAdminSucceeds(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	proc := stripestub.NewProcessor()
	regs, games, admins := newAuthzResolverFixtures(refundGameHostID)
	entries, compAdmins := newEntryAuthzResolverFixtures(refundEntrantPlayerID, refundCompAdminUserID)
	svc := app.NewService(app.ServiceOptions{
		Payments:               repo,
		IDs:                    &fixedIDs{},
		Processor:              proc,
		RegistrationLookup:     regs,
		GameLookup:             games,
		GameAdminReader:        admins,
		EntryLookup:            entries,
		CompetitionAdminReader: compAdmins,
	})
	h := grpcapi.NewHandler(svc)
	seedPaidOnline(t, repo, proc, refundCompetitionPayID, domain.PayableTypeCompetitionEntry, fixtureCompetitionEntryID)

	resp, err := h.RefundPayment(ctxAs(refundCompAdminUserID), &paymentsv1.RefundPaymentRequest{
		PaymentId: refundCompetitionPayID,
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got := resp.GetPayment().GetStatus(); got != paymentsv1.PaymentStatus_PAYMENT_STATUS_REFUNDED {
		t.Fatalf("status = %v, want PAYMENT_STATUS_REFUNDED", got)
	}
}

// TestRefundPayment_Handler_CompetitionEntryPayable_NonEntrantNonAdminRejected
// is the negative control paired with the success test immediately above,
// the same pairing authz_regression_test.go's Allows/Rejects tests already
// establish elsewhere: an actor who is neither the resolved entrant nor an
// assigned Competition Admin is refused, proving the check actually
// discriminates rather than admitting anyone once the payable type gate
// opened.
func TestRefundPayment_Handler_CompetitionEntryPayable_NonEntrantNonAdminRejected(t *testing.T) {
	t.Parallel()

	h, repo, proc := newRefundTestHandler()
	seedPaidOnline(t, repo, proc, refundCompetitionPayID, domain.PayableTypeCompetitionEntry, fixtureCompetitionEntryID)

	_, err := h.RefundPayment(ctxAs(refundOtherPlayerID), &paymentsv1.RefundPaymentRequest{
		PaymentId: refundCompetitionPayID,
	})
	wantCode(t, err, codes.PermissionDenied)

	stored, getErr := repo.GetByID(context.Background(), refundCompetitionPayID)
	if getErr != nil {
		t.Fatalf("unexpected err: %v", getErr)
	}
	if stored.Status != domain.StatusPaid {
		t.Fatalf("persisted status = %v, want paid — a denied refund must have no side effect", stored.Status)
	}
}
