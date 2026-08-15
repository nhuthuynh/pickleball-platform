package grpcapi_test

import (
	"context"
	"testing"
	"time"

	socialplayv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/socialplay/v1"
	"github.com/nhuthuynh/white-label/internal/socialplay/adapter/grpcapi"
	"github.com/nhuthuynh/white-label/internal/socialplay/app"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// fakeReservation is a port.CourtReservation stand-in that always succeeds —
// these tests are about the entry fee travelling across the wire, not about
// court availability (which authz_regression_test.go and the postgres
// integration tests already cover).
type fakeReservation struct{ n int }

func (f *fakeReservation) ReserveCourt(_ context.Context, _ string, _, _ time.Time, _ string) (string, error) {
	f.n++
	return "booking-1", nil
}

func (f *fakeReservation) ReleaseCourt(_ context.Context, _ string) error { return nil }

func newEntryFeeHandler() *grpcapi.Handler {
	svc := app.NewService(app.ServiceOptions{
		IDs:           &fakeIDs{},
		Games:         newFakeGameRepo(),
		Registrations: newFakeRegistrationRepo(),
		Waitlist:      newFakeWaitlistRepo(),
		Matches:       newFakeMatchRepo(),
	})
	return grpcapi.NewHandler(svc, &fakeReservation{}, nil)
}

func createGameReq(fee *socialplayv1.Money) *socialplayv1.CreateGameRequest {
	start := time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC)
	return &socialplayv1.CreateGameRequest{
		HostId:        "host-1",
		FacilityId:    "facility-1",
		CourtIds:      []string{"court-1"},
		StartsAt:      timestamppb.New(start),
		EndsAt:        timestamppb.New(start.Add(time.Hour)),
		Capacity:      4,
		PaymentMethod: socialplayv1.PaymentMethod_PAYMENT_METHOD_EITHER,
		EntryFee:      fee,
	}
}

// TestCreateGame_EntryFeeRoundTrip proves the T9.2 field survives the full
// wire -> app -> domain -> wire path, rather than being silently dropped by
// a translation layer (the failure mode the T4 row-type mismatch showed is
// entirely possible for a newly added field).
func TestCreateGame_EntryFeeRoundTrip(t *testing.T) {
	h := newEntryFeeHandler()

	resp, err := h.CreateGame(ctxAs("host-1"), createGameReq(&socialplayv1.Money{
		AmountCents:  2500,
		CurrencyCode: "USD",
	}))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	got := resp.GetGame().GetEntryFee()
	if got.GetAmountCents() != 2500 || got.GetCurrencyCode() != "USD" {
		t.Fatalf("entry fee = %+v, want 2500 USD", got)
	}
}

// TestCreateGame_FreeGame proves a zero fee is accepted as a real value —
// a free Game — and comes back as an explicit zero-amount Money rather than
// an absent field, so a client can tell "free" from "the server didn't say".
func TestCreateGame_FreeGame(t *testing.T) {
	h := newEntryFeeHandler()

	resp, err := h.CreateGame(ctxAs("host-1"), createGameReq(&socialplayv1.Money{
		AmountCents:  0,
		CurrencyCode: "USD",
	}))
	if err != nil {
		t.Fatalf("a free game must be accepted, got err: %v", err)
	}
	fee := resp.GetGame().GetEntryFee()
	if fee == nil {
		t.Fatal("entry fee message must be present even for a free game")
	}
	if fee.GetAmountCents() != 0 {
		t.Fatalf("amount = %d, want 0", fee.GetAmountCents())
	}
}

// TestCreateGame_AbsentEntryFeeIsFree proves an old/unaware client that
// sends no entry_fee at all gets a free Game rather than an error — the
// same value db/migrations/0013_socialplay_entry_fee.sql backfilled onto
// every pre-existing row.
func TestCreateGame_AbsentEntryFeeIsFree(t *testing.T) {
	h := newEntryFeeHandler()

	resp, err := h.CreateGame(ctxAs("host-1"), createGameReq(nil))
	if err != nil {
		t.Fatalf("an absent entry fee must be accepted as free, got err: %v", err)
	}
	if got := resp.GetGame().GetEntryFee().GetAmountCents(); got != 0 {
		t.Fatalf("amount = %d, want 0", got)
	}
}

// TestCreateGame_InvalidEntryFeeIsInvalidArgument proves domain.ErrInvalidMoney
// is mapped to 400 INVALID_ARGUMENT, not a 500 — a malformed price is a bad
// request, not a server fault.
func TestCreateGame_InvalidEntryFeeIsInvalidArgument(t *testing.T) {
	tests := []struct {
		name string
		fee  *socialplayv1.Money
	}{
		{"negative amount", &socialplayv1.Money{AmountCents: -1, CurrencyCode: "USD"}},
		{"non-zero amount with no currency", &socialplayv1.Money{AmountCents: 1500}},
		{"non-zero amount with a malformed currency", &socialplayv1.Money{AmountCents: 1500, CurrencyCode: "dollars"}},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			h := newEntryFeeHandler()

			_, err := h.CreateGame(ctxAs("host-1"), createGameReq(tt.fee))
			if err == nil {
				t.Fatal("expected an error, got nil")
			}
			if got := status.Code(err); got != codes.InvalidArgument {
				t.Fatalf("code = %v, want %v", got, codes.InvalidArgument)
			}
		})
	}
}
