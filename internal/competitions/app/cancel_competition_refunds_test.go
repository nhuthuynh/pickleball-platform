package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nhuthuynh/white-label/internal/competitions/app"
	"github.com/nhuthuynh/white-label/internal/competitions/domain"
)

// T55.3 (issue #124's refund half, mirrored onto Competitions) — cancelling
// a Competition refunds the entrants who had paid.
//
// #124 was raised against Social Play, but T16.3 found the identical gap
// here and fixed the entries half in lockstep; the refund half mirrors the
// same way. See internal/socialplay/app/cancel_game_refunds_test.go for the
// Game-side equivalents — these assert the same properties on the same
// reasoning.

func enterPlayers(t *testing.T, svc *app.Service, competitionID string, players ...string) []string {
	t.Helper()

	ids := make([]string, 0, len(players))
	for _, p := range players {
		e, err := svc.EnterCompetition(context.Background(), app.EnterCompetitionInput{
			CompetitionID: competitionID,
			PlayerID:      p,
			Source:        domain.EntrySourceApp,
		})
		if err != nil {
			t.Fatalf("seed entry for %s: %v", p, err)
		}
		ids = append(ids, e.ID)
	}
	return ids
}

func TestCancelCompetition_RefundsEveryEntry(t *testing.T) {
	t.Parallel()

	_, svc, c := scheduleFixture(t, 16, 0)
	ids := enterPlayers(t, svc, c.ID, "player-1", "player-2", "player-3")
	refunds := &fakeEntryRefunder{}

	if _, err := svc.CancelCompetition(context.Background(), c.ID, c.HostID, refunds); err != nil {
		t.Fatalf("CancelCompetition: %v", err)
	}

	if len(refunds.refunded) != len(ids) {
		t.Fatalf("attempted %d refunds, want %d (one per entry)", len(refunds.refunded), len(ids))
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
			t.Fatalf("entry %s was never offered for refund", id)
		}
	}

	// The actor matters: Payments authorizes against the Competition's Host,
	// so passing anything else would be refused in production while still
	// passing a test that only counted calls.
	for _, actor := range refunds.actors {
		if actor != c.HostID {
			t.Fatalf("refund attempted as %q, want the Competition's host %q", actor, c.HostID)
		}
	}
}

// The partial-failure rule: one failing refund must not stop the others.
func TestCancelCompetition_OneFailingRefundDoesNotStopTheRest(t *testing.T) {
	t.Parallel()

	_, svc, c := scheduleFixture(t, 16, 0)
	ids := enterPlayers(t, svc, c.ID, "player-1", "player-2", "player-3")

	boom := errors.New("processor unavailable")
	refunds := &fakeEntryRefunder{failFor: map[string]error{ids[0]: boom}}

	_, err := svc.CancelCompetition(context.Background(), c.ID, c.HostID, refunds)
	if !errors.Is(err, boom) {
		t.Fatalf("CancelCompetition = %v, want it to wrap the refund failure %v", err, boom)
	}
	if len(refunds.refunded) != len(ids) {
		t.Fatalf("attempted %d refunds after a failure, want all %d — the cascade must not stop early",
			len(refunds.refunded), len(ids))
	}
}

// The Competition is still cancelled even when refunds fail: the status
// write and the entry cascade committed before the refunds ran.
func TestCancelCompetition_RefundFailureStillLeavesItCancelled(t *testing.T) {
	t.Parallel()

	repo, svc, c := scheduleFixture(t, 16, 0)
	ids := enterPlayers(t, svc, c.ID, "player-1")
	refunds := &fakeEntryRefunder{failFor: map[string]error{
		ids[0]: errors.New("processor unavailable"),
	}}

	cancelled, err := svc.CancelCompetition(context.Background(), c.ID, c.HostID, refunds)
	if err == nil {
		t.Fatal("expected the refund failure to be surfaced")
	}
	if cancelled.Status != domain.StatusCancelled {
		t.Fatalf("returned Status = %q, want cancelled even though refunds failed", cancelled.Status)
	}
	if repo.competitions[c.ID].Status != domain.StatusCancelled {
		t.Fatalf("persisted Status = %q, want cancelled", repo.competitions[c.ID].Status)
	}
}

// A refused cancellation refunds nobody: authorization runs first.
func TestCancelCompetition_NonHostRefundsNobody(t *testing.T) {
	t.Parallel()

	_, svc, c := scheduleFixture(t, 16, 0)
	enterPlayers(t, svc, c.ID, "player-1")
	refunds := &fakeEntryRefunder{}

	if _, err := svc.CancelCompetition(context.Background(), c.ID, "not-the-host", refunds); !errors.Is(err, domain.ErrNotCompetitionHost) {
		t.Fatalf("CancelCompetition(non-host) = %v, want %v", err, domain.ErrNotCompetitionHost)
	}
	if len(refunds.refunded) != 0 {
		t.Fatalf("attempted %d refunds for a refused cancellation, want 0", len(refunds.refunded))
	}
}
