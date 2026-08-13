// T6.6 — JoinWaitlist handler-level smoke test, mirroring
// authz_regression_test.go's shape and reasoning: this proves the wire path
// (grpcapi.Handler -> app.Service -> domain) end to end and that domain
// errors map to the right gRPC status, not the domain logic itself (already
// covered exhaustively in internal/socialplay/domain/waitlist_test.go and
// internal/socialplay/app/service_test.go).
package grpcapi_test

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/nhuthuynh/white-label/internal/socialplay/adapter/grpcapi"
	"github.com/nhuthuynh/white-label/internal/socialplay/app"

	socialplayv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/socialplay/v1"
)

// newTestHandlerWithWaitlist mirrors newTestHandler (authz_regression_test.go)
// but also returns the fakeWaitlistRepo, since these tests need to seed/
// inspect waitlist state that newTestHandler's fixed set of return values
// doesn't expose.
func newTestHandlerWithWaitlist() (*fakeGameRepo, *fakeRegistrationRepo, *fakeWaitlistRepo, *app.Service) {
	gameRepo := newFakeGameRepo()
	regRepo := newFakeRegistrationRepo()
	waitlistRepo := newFakeWaitlistRepo()
	svc := app.NewService(&fakeIDs{}, gameRepo, regRepo, waitlistRepo, newFakeMatchRepo())
	return gameRepo, regRepo, waitlistRepo, svc
}

// TestJoinWaitlist_Valid proves the happy path through the real handler: a
// full Game accepts a waitlist join over the wire.
func TestJoinWaitlist_Valid(t *testing.T) {
	ctx := context.Background()
	gameRepo, _, waitlistRepo, svc := newTestHandlerWithWaitlist()
	h := grpcapi.NewHandler(svc, nil, nil)

	game := seedGame(t, gameRepo, "game-wl-1", 1)
	if _, err := h.RegisterForGame(ctx, &socialplayv1.RegisterForGameRequest{GameId: game.ID, PlayerId: "player-a"}); err != nil {
		t.Fatalf("fixture registration should succeed: %v", err)
	}

	resp, err := h.JoinWaitlist(ctx, &socialplayv1.JoinWaitlistRequest{GameId: game.ID, PlayerId: "player-b"})
	if err != nil {
		t.Fatalf("JoinWaitlist should succeed on a full game: %v", err)
	}
	if resp.GetEntry().GetStatus() != socialplayv1.WaitlistStatus_WAITLIST_STATUS_WAITING {
		t.Fatalf("entry status = %v, want WAITLIST_STATUS_WAITING", resp.GetEntry().GetStatus())
	}
	if resp.GetEntry().GetPosition() != 1 {
		t.Fatalf("entry position = %d, want 1", resp.GetEntry().GetPosition())
	}

	if _, err := waitlistRepo.GetByID(ctx, resp.GetEntry().GetId()); err != nil {
		t.Fatalf("entry should be persisted: %v", err)
	}
}

// TestJoinWaitlist_GameNotFull_MapsToInvalidArgument proves
// domain.ErrGameNotFull maps to a 400-shaped status over the wire, not a
// 500 and not a silent success.
func TestJoinWaitlist_GameNotFull_MapsToInvalidArgument(t *testing.T) {
	ctx := context.Background()
	gameRepo, _, _, svc := newTestHandlerWithWaitlist()
	h := grpcapi.NewHandler(svc, nil, nil)

	game := seedGame(t, gameRepo, "game-wl-2", 4)

	_, err := h.JoinWaitlist(ctx, &socialplayv1.JoinWaitlistRequest{GameId: game.ID, PlayerId: "player-a"})
	if err == nil {
		t.Fatal("expected an error joining the waitlist of a game that isn't full")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("got a non-gRPC-status error: %v", err)
	}
	if st.Code() != codes.InvalidArgument {
		t.Fatalf("status code = %v, want InvalidArgument", st.Code())
	}
}

// TestJoinWaitlist_AlreadyOnWaitlist_MapsToAlreadyExists proves
// domain.ErrAlreadyOnWaitlist maps to a 409-shaped status, mirroring
// ErrAlreadyRegistered's own mapping.
func TestJoinWaitlist_AlreadyOnWaitlist_MapsToAlreadyExists(t *testing.T) {
	ctx := context.Background()
	gameRepo, _, _, svc := newTestHandlerWithWaitlist()
	h := grpcapi.NewHandler(svc, nil, nil)

	game := seedGame(t, gameRepo, "game-wl-3", 1)
	if _, err := h.RegisterForGame(ctx, &socialplayv1.RegisterForGameRequest{GameId: game.ID, PlayerId: "player-a"}); err != nil {
		t.Fatalf("fixture registration should succeed: %v", err)
	}
	if _, err := h.JoinWaitlist(ctx, &socialplayv1.JoinWaitlistRequest{GameId: game.ID, PlayerId: "player-b"}); err != nil {
		t.Fatalf("fixture join should succeed: %v", err)
	}

	_, err := h.JoinWaitlist(ctx, &socialplayv1.JoinWaitlistRequest{GameId: game.ID, PlayerId: "player-b"})
	if err == nil {
		t.Fatal("expected an error on a duplicate waitlist join")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("got a non-gRPC-status error: %v", err)
	}
	if st.Code() != codes.AlreadyExists {
		t.Fatalf("status code = %v, want AlreadyExists", st.Code())
	}
}
