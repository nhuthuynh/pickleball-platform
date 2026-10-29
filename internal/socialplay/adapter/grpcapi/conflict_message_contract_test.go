// The exact gRPC messages RegisterForGame returns for its two distinct
// conflicts — pinned because the WEB CLIENT MATCHES ON THEM (#345).
//
// error_mapping_test.go already asserts that these sentinels map to
// AlreadyExists. That is not enough for the client: five sentinels share
// AlreadyExists on registration paths (ErrGameFull, ErrAlreadyRegistered,
// ErrCourtUnavailable, ErrAlreadyOnWaitlist, ErrAlreadyGameAdmin), and the
// grouping is deliberate and documented in toStatus. So HTTP 409 alone
// cannot tell a browser which happened, and for a while the client assumed
// every 409 meant "full": a player who was already registered was told the
// game had just filled up and offered a waitlist for a game they were
// already in.
//
// web/src/composables/useJoinGame.ts now reads the message to tell them
// apart. That makes these strings an API contract rather than prose, and a
// reword would silently return the UI to the wrong message — silently,
// because every assertion in the web suite would still pass against its own
// fixtures. This test is what makes that reword fail here instead.
//
// If you need to change one of these strings: change it in
// internal/socialplay/domain/errors.go, change the matcher in
// useJoinGame.ts, and change this test, in the same commit.
package grpcapi_test

import (
	"testing"

	"google.golang.org/grpc/status"

	socialplayv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/socialplay/v1"
)

func TestRegisterForGame_ConflictMessagesAreTheClientsContract(t *testing.T) {
	// Exactly the strings useJoinGame.ts matches. Written out rather than
	// referenced from domain.Err* on purpose: a test that reads the
	// sentinel's own text cannot notice the text changing, which is the
	// thing this test exists to notice.
	const (
		wantGameFull          = "socialplay: game is at capacity"
		wantAlreadyRegistered = "socialplay: player is already registered for this game"
	)

	t.Run("a full game", func(t *testing.T) {
		h, gameRepo, _ := newTestHandler()
		game := seedGame(t, gameRepo, "contract-full", 1)
		// Capacity 1, taken by another player.
		if _, err := h.RegisterForGame(ctxAs(mapHostID), &socialplayv1.RegisterForGameRequest{GameId: game.ID}); err != nil {
			t.Fatalf("seed RegisterForGame: %v", err)
		}

		_, err := h.RegisterForGame(ctxAs(mapPlayerID), &socialplayv1.RegisterForGameRequest{GameId: game.ID})
		if err == nil {
			t.Fatal("registering for a full game succeeded")
		}
		if got := status.Convert(err).Message(); got != wantGameFull {
			t.Errorf("message changed:\n got %q\nwant %q\n\n"+
				"web/src/composables/useJoinGame.ts matches this string to decide "+
				"whether to offer the waitlist. Update both, or the UI silently "+
				"reports the wrong thing.", got, wantGameFull)
		}
	})

	t.Run("already registered", func(t *testing.T) {
		h, gameRepo, _ := newTestHandler()
		game := seedGame(t, gameRepo, "contract-dup", 4)
		req := &socialplayv1.RegisterForGameRequest{GameId: game.ID}
		if _, err := h.RegisterForGame(ctxAs(mapPlayerID), req); err != nil {
			t.Fatalf("seed RegisterForGame: %v", err)
		}

		_, err := h.RegisterForGame(ctxAs(mapPlayerID), req)
		if err == nil {
			t.Fatal("registering twice succeeded")
		}
		if got := status.Convert(err).Message(); got != wantAlreadyRegistered {
			t.Errorf("message changed:\n got %q\nwant %q\n\n"+
				"web/src/composables/useJoinGame.ts matches this string to show "+
				"\"you are already registered\" instead of \"this game just filled "+
				"up\". Update both.", got, wantAlreadyRegistered)
		}
	})

	// The two must stay DIFFERENT, which is the property the client actually
	// depends on. If a refactor ever collapsed both onto one message, the
	// assertions above could both be updated to that message and still pass.
	t.Run("and they are distinguishable", func(t *testing.T) {
		if wantGameFull == wantAlreadyRegistered {
			t.Fatal("the two conflict messages are identical; the client cannot tell them apart")
		}
	})
}
