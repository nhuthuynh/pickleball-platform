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

func (f *fakeReservation) ReserveCourt(_ context.Context, _ string, _, _ time.Time, _, _ string) (string, error) {
	f.n++
	return "booking-1", nil
}

func (f *fakeReservation) ReleaseCourt(_ context.Context, _, _ string) error { return nil }

func newEntryFeeHandler() *grpcapi.Handler {
	svc := app.NewService(app.ServiceOptions{
		Identity:      fakeIdentityLookup{},
		IDs:           &fakeIDs{},
		Games:         newFakeGameRepo(),
		Registrations: newFakeRegistrationRepo(),
		Waitlist:      newFakeWaitlistRepo(),
		Matches:       newFakeMatchRepo(),
		GameAdmins:    newFakeGameAdminRepo(),
	})
	return grpcapi.NewHandler(svc, &fakeReservation{}, nil, noopRefunder{})
}

func createGameReq(fee *socialplayv1.Money) *socialplayv1.CreateGameRequest {
	start := time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC)
	return &socialplayv1.CreateGameRequest{
		HostId:        "host-1",
		FacilityId:    "facility-1",
		CourtIds:      []string{courtID(1)},
		StartsAt:      timestamppb.New(start),
		EndsAt:        timestamppb.New(start.Add(time.Hour)),
		Capacity:      8,
		PaymentMethod: socialplayv1.PaymentMethod_PAYMENT_METHOD_EITHER,
		// T56.1: a guest allowance, so the per-head amount_owed tests below
		// can actually register a party. 0 (the previous implicit value)
		// made every guest-bringing registration a rejection before it
		// reached the field under test.
		GuestAllowance: 3,
		EntryFee:       fee,
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

// ReleaseCourtsForReference implements port.CourtReservation for T55.2's
// #124 cascade. This fake records nothing: the tests that exercise the
// cascade drive it through a real Reservation over an in-memory booking
// repository, so a counting stub here would only assert that a call
// happened, not that a court was actually freed.
func (f *fakeReservation) ReleaseCourtsForReference(context.Context, string, string) (int, error) {
	return 0, nil
}

// T56.1 (issue #126) — Registration.amount_owed survives the full
// wire -> app -> domain -> wire path.
//
// This test exists because of what its absence allowed, which was measured
// rather than assumed: deleting the single `AmountOwed:` line from
// toProtoRegistration left `make test-domain`, `make test-adapters` and
// `make test-cmd` ALL GREEN. Every Go gate passed while the feature was
// entirely broken — a client would read 0, send 0, and T56.2's validation
// would then refuse every payment it was supposed to permit. The failure
// mode is the same one TestCreateGame_EntryFeeRoundTrip above was written
// for, and it recurred on the very next field added to this message.
//
// The per-head arithmetic itself is proven in the domain
// (internal/socialplay/domain/amount_owed_test.go). What is proven HERE, and
// only here, is that the number reaches the client at all.
func TestRegisterForGame_AmountOwedRoundTrip(t *testing.T) {
	h := newEntryFeeHandler()

	gameResp, err := h.CreateGame(ctxAs("host-1"), createGameReq(&socialplayv1.Money{
		AmountCents:  2500,
		CurrencyCode: "USD",
	}))
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}

	// Two guests: three heads at 2500 = 7500. A response carrying 2500
	// would mean the per-head rule never reached the wire; one carrying 0
	// would mean the field did not.
	regResp, err := h.RegisterForGame(ctxAs("player-1"), &socialplayv1.RegisterForGameRequest{
		GameId:     gameResp.GetGame().GetId(),
		GuestCount: 2,
	})
	if err != nil {
		t.Fatalf("RegisterForGame: %v", err)
	}

	got := regResp.GetRegistration().GetAmountOwed()
	if got == nil {
		t.Fatal("amount_owed is absent from the wire — the client cannot pay what it cannot see")
	}
	if got.GetAmountCents() != 7500 {
		t.Fatalf("amount_owed cents = %d, want 7500 (2500 × 3 heads)", got.GetAmountCents())
	}
	if got.GetCurrencyCode() != "USD" {
		t.Fatalf("amount_owed currency = %q, want USD", got.GetCurrencyCode())
	}
}

// A free Game emits the message with a zero amount rather than omitting it,
// so a client can tell "this is free" from "the server never populated
// this" — the same contract Game.entry_fee carries, and the distinction
// GameCheckout.vue's free-game branch depends on.
func TestRegisterForGame_FreeGameStillEmitsAmountOwed(t *testing.T) {
	h := newEntryFeeHandler()

	gameResp, err := h.CreateGame(ctxAs("host-1"), createGameReq(&socialplayv1.Money{}))
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}

	regResp, err := h.RegisterForGame(ctxAs("player-1"), &socialplayv1.RegisterForGameRequest{
		GameId:     gameResp.GetGame().GetId(),
		GuestCount: 2,
	})
	if err != nil {
		t.Fatalf("RegisterForGame: %v", err)
	}

	got := regResp.GetRegistration().GetAmountOwed()
	if got == nil {
		t.Fatal("a free Game must still emit amount_owed, so 'free' is distinguishable from 'unset'")
	}
	if got.GetAmountCents() != 0 {
		t.Fatalf("amount_owed cents = %d, want 0 — a free game is free for any party size", got.GetAmountCents())
	}
}
