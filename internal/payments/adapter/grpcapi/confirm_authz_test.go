// T13.7 (closes issue #148) — object-level authorization for
// ConfirmOnlinePayment, the RPC that captures the money.
//
// The gap this file exists to close, in the issue's own words: "anyone holding
// a payment_id can capture the intent". Before this ticket ConfirmOnlinePayment
// was the one RPC on this service in PublicMethods() — no principal required,
// no actor read, and no ownership fact recorded anywhere for it to check, since
// CreateOnlinePayment wrote an empty RecordedByUserID for every online Payment.
// A payment_id is not a secret: it comes back in the CreateOnlinePayment
// response, travels through client logs and URLs, and is UUID-shaped but not
// unguessable-by-policy. Holding one was the whole of the authorization.
//
// The four cases below are the same four T12.8's principal_authz_test.go
// established for the other three RPCs, applied to this one:
//
//	(a) the principal that created the intent      -> succeeds
//	(b) a different verified principal             -> PermissionDenied
//	(c) no principal at all                        -> Unauthenticated
//	(d) a Payment with no recorded owner           -> PermissionDenied (fail closed)
//
// (b) and (c) stay distinct codes for ADR-0013 §5's reason: "I know who you are
// and you may not do this" is not "I do not know who you are". (d) is issue
// #148's explicitly-open question — reject ownerless legacy rows, or grandfather
// them — answered by rejecting, since grandfathering would leave exactly the
// hole this ticket closes open on every row created before it.
//
// Identifier space: both sides of the comparison are subjects. The context
// carries a subject (auth.Principal.Subject), and payments.recorded_by_user_id
// is a text column holding whatever actor(ctx) returned at creation time — so
// no resolution step is involved, per ADR-0014 §5a's explicit ruling for this
// ticket. The fixtures below are auth0|-shaped for that reason: they are
// subjects, not User uuids.
//
// Handler-level with the in-memory port.Repository fake and the deterministic
// stub processor, for the reason authz_regression_test.go, refund_test.go and
// error_mapping_test.go already document: nothing under test here is influenced
// by which port.Repository sits behind it, and this environment has no Docker
// daemon, so CLAUDE.md rule 10 means the test has to actually run where it was
// written.
package grpcapi_test

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"

	"github.com/nhuthuynh/white-label/internal/payments/adapter/grpcapi"
	"github.com/nhuthuynh/white-label/internal/payments/domain"

	paymentsv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/payments/v1"
)

const (
	// payerSubject is the caller who creates the intent, and therefore the
	// only caller entitled to capture it. auth0|-shaped because it is a
	// subject: ADR-0014 §5a rules that this context compares actor(ctx)'s
	// value against the stored one unchanged.
	payerSubject = "auth0|payer-7"
	// intruderSubject is a perfectly valid principal — a real, authenticated
	// user of the platform — who simply is not the payer. Case (b) is about
	// authorization, not authentication, which is why this is a verified
	// caller rather than a bogus token.
	intruderSubject = "auth0|intruder-7"

	confirmPaymentID = "6ba7b810-0000-4000-8000-0000000000d1"
	// confirmLegacyPaymentID stands for a row written before this ticket:
	// an online Payment with an empty recorded_by_user_id.
	confirmLegacyPaymentID = "6ba7b810-0000-4000-8000-0000000000d2"
)

// createIntentAs drives the real CreateOnlinePayment RPC as subject and returns
// the resulting Payment id. Going through the RPC rather than seeding the
// repository directly is deliberate: the ownership fact this ticket's check
// reads has to be *produced* by production code, not planted by the test.
func createIntentAs(t *testing.T, h *grpcapi.Handler, subject string) string {
	t.Helper()

	resp, err := h.CreateOnlinePayment(ctxAs(subject), &paymentsv1.CreateOnlinePaymentRequest{
		PayableType: paymentsv1.PayableType_PAYABLE_TYPE_BOOKING,
		PayableId:   fixtureBookingID,
		Amount:      &paymentsv1.Money{AmountCents: 3000, CurrencyCode: "USD"},
	})
	if err != nil {
		t.Fatalf("seed: CreateOnlinePayment as %s: %v", subject, err)
	}
	return resp.GetPayment().GetId()
}

// TestCreateOnlinePayment_RecordsTheVerifiedPrincipalAsOwner is the fact the
// whole ticket rests on: before it, CreateOnlinePayment passed "" as the
// Payment's RecordedByUserID, so there was no ownership fact for
// ConfirmOnlinePayment to check at all — which is why issue #148 reads "there
// is no ownership fact recorded anywhere for it to check".
//
// The value must come from the principal, never from req.ActorUserId, for the
// reason T12.8 already proved for RecordOfflinePayment's copy of this field: a
// wire-sourced owner would let a caller record the intent under someone else's
// name, and here it would additionally let them *choose* who may capture it.
func TestCreateOnlinePayment_RecordsTheVerifiedPrincipalAsOwner(t *testing.T) {
	t.Parallel()

	h, _, _ := newMappingHandler(confirmPaymentID)

	req := &paymentsv1.CreateOnlinePaymentRequest{
		PayableType: paymentsv1.PayableType_PAYABLE_TYPE_BOOKING,
		PayableId:   fixtureBookingID,
		Amount:      &paymentsv1.Money{AmountCents: 3000, CurrencyCode: "USD"},
	}
	// nolint:staticcheck // SA1019: setting the deprecated field IS the test.
	req.ActorUserId = intruderSubject // the lie

	resp, err := h.CreateOnlinePayment(ctxAs(payerSubject), req)
	if err != nil {
		t.Fatalf("CreateOnlinePayment as %s should succeed: %v", payerSubject, err)
	}

	if got := resp.GetPayment().GetRecordedByUserId(); got != payerSubject {
		t.Fatalf("Payment.RecordedByUserId = %q, want %q — an online Payment must record the "+
			"verified principal that created the intent, or ConfirmOnlinePayment has nothing to "+
			"check and anyone holding the payment_id can capture it (issue #148)", got, payerSubject)
	}
}

// TestConfirmOnlinePayment_OwnerCheck is issue #148 stated as a table: who may
// capture an intent, and what every other caller is told.
func TestConfirmOnlinePayment_OwnerCheck(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		// confirmAs builds the context the capture is attempted from.
		confirmAs func() context.Context
		// wantCode is codes.OK for the one caller entitled to capture.
		wantCode codes.Code
		why      string
	}{
		{
			name:      "the principal that created the intent captures it",
			confirmAs: func() context.Context { return ctxAs(payerSubject) },
			wantCode:  codes.OK,
			why: "without this row the table cannot tell 'the owner check rejects the wrong caller' " +
				"apart from 'ConfirmOnlinePayment is broken and rejects everyone'",
		},
		{
			name:      "a different verified principal is refused",
			confirmAs: func() context.Context { return ctxAs(intruderSubject) },
			wantCode:  codes.PermissionDenied,
			why: "a verified principal failing an object-level (BOLA) check is 403-shaped — " +
				"not Unauthenticated (we know exactly who they are) and not NotFound " +
				"(ADR-0013 §5, ADR-0014 §6)",
		},
		{
			name:      "no principal at all is Unauthenticated",
			confirmAs: anonymous,
			wantCode:  codes.Unauthenticated,
			why: "'I do not know who you are' must stay distinguishable from 'I know who you are " +
				"and you may not do this' — the anonymous caller's fix is to authenticate, and " +
				"PermissionDenied does not say so (ADR-0013 §5)",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h, repo, _ := newMappingHandler(confirmPaymentID)
			paymentID := createIntentAs(t, h, payerSubject)

			resp, err := h.ConfirmOnlinePayment(tc.confirmAs(), &paymentsv1.ConfirmOnlinePaymentRequest{
				PaymentId: paymentID,
			})

			if tc.wantCode == codes.OK {
				if err != nil {
					t.Fatalf("ConfirmOnlinePayment: %v — %s", err, tc.why)
				}
				if got := resp.GetPayment().GetStatus(); got != paymentsv1.PaymentStatus_PAYMENT_STATUS_PAID {
					t.Fatalf("Payment.Status = %v, want PAID", got)
				}
				return
			}

			requireCode(t, "ConfirmOnlinePayment ("+tc.name+", "+tc.why+")", err, tc.wantCode)

			// Not a silent success: prove the money was not captured, not
			// merely that an error came back on the wire.
			if got := repo.byID[paymentID].Status; got != domain.StatusUnpaid {
				t.Fatalf("stored Payment.Status = %v after a refused capture, want %v — "+
					"the refused call captured anyway", got, domain.StatusUnpaid)
			}
		})
	}
}

// TestConfirmOnlinePayment_OwnerlessPaymentIsRefused answers issue #148's own
// open question about rows created before this ticket: "reject confirmation on
// ownerless legacy rows, or grandfather them".
//
// Rejecting is the only answer that closes the issue. Grandfathering would keep
// "anyone holding a payment_id can capture the intent" true for every row
// already in the table, on the one path where nobody would notice — and an
// empty owner is also what a future code path that forgets to set one produces,
// so failing closed here makes that bug loud instead of exploitable.
func TestConfirmOnlinePayment_OwnerlessPaymentIsRefused(t *testing.T) {
	t.Parallel()

	h, repo, proc := newMappingHandler()

	ref, err := proc.CreateIntent(context.Background(), domain.Money{Cents: 3000, Currency: "USD"}, fixtureBookingID)
	if err != nil {
		t.Fatalf("seed: CreateIntent: %v", err)
	}
	// Seeded straight into the repository with no RecordedByUserID — the
	// shape every online Payment had before this ticket.
	seedOnline(t, repo, confirmLegacyPaymentID, domain.PayableTypeBooking, fixtureBookingID, ref, domain.StatusUnpaid, "")

	_, err = h.ConfirmOnlinePayment(ctxAs(payerSubject), &paymentsv1.ConfirmOnlinePaymentRequest{
		PaymentId: confirmLegacyPaymentID,
	})
	requireCode(t, "ConfirmOnlinePayment against an ownerless Payment", err, codes.PermissionDenied)

	if got := repo.byID[confirmLegacyPaymentID].Status; got != domain.StatusUnpaid {
		t.Fatalf("stored Payment.Status = %v after a refused capture, want %v", got, domain.StatusUnpaid)
	}
}
