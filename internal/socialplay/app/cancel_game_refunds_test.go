package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nhuthuynh/white-label/internal/socialplay/app"
	"github.com/nhuthuynh/white-label/internal/socialplay/domain"
)

// T55.3 (issue #124's refund half) — cancelling a Game refunds the players
// who had paid for it.
//
// The Product Owner answered #124's second question on 2026-09-04:
// host-initiated cancellation refunds automatically, because the players did
// nothing wrong. T16.3 already cancelled their Registrations; nothing
// reversed their money, and RefundPayment existed but was simply never
// called.
//
// Whether a given payment is actually refundable is Payments' decision (see
// app.Service.RefundForPayable) — what these tests pin is that Social Play
// asks about every registration, as the right actor, and handles a partial
// failure correctly.

func registerPlayers(t *testing.T, svc *app.Service, gameID string, players ...string) []string {
	t.Helper()

	ids := make([]string, 0, len(players))
	for _, p := range players {
		reg, err := svc.RegisterForGame(context.Background(), app.RegisterForGameInput{
			GameID:   gameID,
			PlayerID: p,
		})
		if err != nil {
			t.Fatalf("seed registration for %s: %v", p, err)
		}
		ids = append(ids, reg.ID)
	}
	return ids
}

func TestCancelGame_RefundsEveryRegistration(t *testing.T) {
	svc, g, _, _ := newMatchTestService(t)
	ctx := context.Background()

	ids := registerPlayers(t, svc, g.ID, "player-1", "player-2", "player-3")
	refunds := &fakeRefunder{}

	if _, err := svc.CancelGame(ctx, g.ID, g.HostID, &fakeReservation{}, refunds); err != nil {
		t.Fatalf("CancelGame: %v", err)
	}

	if len(refunds.refunded) != len(ids) {
		t.Fatalf("attempted %d refunds, want %d (one per registration)", len(refunds.refunded), len(ids))
	}
	for _, id := range ids {
		found := false
		for _, got := range refunds.refunded {
			if got == id {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("registration %s was never offered for refund", id)
		}
	}

	// The actor matters: Payments authorizes the refund against the Game's
	// Host, so passing anything else would be refused in production while
	// still passing a test that only counted calls.
	for _, actor := range refunds.actors {
		if actor != g.HostID {
			t.Fatalf("refund attempted as %q, want the Game's host %q", actor, g.HostID)
		}
	}
}

// The partial-failure rule, stated in CancelGame's doc comment: one failing
// refund must not stop the others. Stopping early would leave the remaining
// players unrefunded because somebody else's refund failed — the worse
// outcome when what is being distributed is money.
func TestCancelGame_OneFailingRefundDoesNotStopTheRest(t *testing.T) {
	svc, g, _, _ := newMatchTestService(t)
	ctx := context.Background()

	ids := registerPlayers(t, svc, g.ID, "player-1", "player-2", "player-3")

	boom := errors.New("processor unavailable")
	refunds := &fakeRefunder{failFor: map[string]error{ids[0]: boom}}

	_, err := svc.CancelGame(ctx, g.ID, g.HostID, &fakeReservation{}, refunds)
	if !errors.Is(err, boom) {
		t.Fatalf("CancelGame = %v, want it to wrap the refund failure %v", err, boom)
	}

	// Every registration was still attempted — this is the actual assertion.
	if len(refunds.refunded) != len(ids) {
		t.Fatalf("attempted %d refunds after a failure, want all %d — the cascade must not stop early",
			len(refunds.refunded), len(ids))
	}
}

// The Game is still reported cancelled even when refunds fail: the status
// write and the registration cascade committed before the refunds ran, and
// that ordering is deliberate.
func TestCancelGame_RefundFailureStillLeavesTheGameCancelled(t *testing.T) {
	svc, g, games, _ := newMatchTestService(t)
	ctx := context.Background()

	ids := registerPlayers(t, svc, g.ID, "player-1")
	refunds := &fakeRefunder{failFor: map[string]error{
		ids[0]: errors.New("processor unavailable"),
	}}

	cancelled, err := svc.CancelGame(ctx, g.ID, g.HostID, &fakeReservation{}, refunds)
	if err == nil {
		t.Fatal("expected the refund failure to be surfaced")
	}
	if cancelled.Status != domain.StatusCancelled {
		t.Fatalf("returned Status = %q, want cancelled even though refunds failed", cancelled.Status)
	}
	// And persisted — the caller must be able to trust that the Game really
	// is cancelled and only the money is outstanding.
	stored, err := games.GetByID(ctx, g.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if stored.Status != domain.StatusCancelled {
		t.Fatalf("persisted Status = %q, want cancelled", stored.Status)
	}
}

// A refused cancellation refunds nobody: authorization runs first.
func TestCancelGame_NonHostRefundsNobody(t *testing.T) {
	svc, g, _, _ := newMatchTestService(t)
	ctx := context.Background()

	registerPlayers(t, svc, g.ID, "player-1")
	refunds := &fakeRefunder{}

	if _, err := svc.CancelGame(ctx, g.ID, "not-the-host", &fakeReservation{}, refunds); !errors.Is(err, domain.ErrNotGameHost) {
		t.Fatalf("CancelGame(non-host) = %v, want %v", err, domain.ErrNotGameHost)
	}
	if len(refunds.refunded) != 0 {
		t.Fatalf("attempted %d refunds for a refused cancellation, want 0", len(refunds.refunded))
	}
}
