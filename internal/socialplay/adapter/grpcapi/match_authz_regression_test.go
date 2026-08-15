// T10.4 (docs/process/t10-sprint-plan.md T10.4) — object-level authorization
// regression tests for RecordMatchResult, run through the real gRPC handler
// (not just the app-level unit tests internal/socialplay/app/service_test.go
// already has for domain.Game.EnsureHostOrGameAdmin/Service.RecordMatchResult).
// This is the "does the guarantee survive the full stack" test the ticket's
// own instructions require, mirroring
// internal/payments/adapter/grpcapi/competition_entry_authz_regression_test.go's
// shape exactly — same reasoning, one context and one payable-vs-Game-Admin
// concept over.
//
// Also covers this ticket's malformed-ID boundary guard and the
// FailedPrecondition/cancelled-Game case at the handler level, mirroring
// authz_regression_test.go/waitlist_test.go's existing handler-level smoke
// coverage in this package.
package grpcapi_test

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/nhuthuynh/white-label/internal/socialplay/adapter/grpcapi"
	"github.com/nhuthuynh/white-label/internal/socialplay/app"
	"github.com/nhuthuynh/white-label/internal/socialplay/domain"

	socialplayv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/socialplay/v1"
)

// fakeMatchRepo is a minimal in-memory port.MatchRepository fake, mirroring
// fakeGameRepo/fakeRegistrationRepo's shape in authz_regression_test.go.
type fakeMatchRepo struct {
	matches map[string]domain.Match
}

func newFakeMatchRepo() *fakeMatchRepo {
	return &fakeMatchRepo{matches: map[string]domain.Match{}}
}

func (f *fakeMatchRepo) Create(_ context.Context, m domain.Match) (domain.Match, error) {
	f.matches[m.ID] = m
	return m, nil
}

func (f *fakeMatchRepo) ListForGame(_ context.Context, gameID string) ([]domain.Match, error) {
	var out []domain.Match
	for _, m := range f.matches {
		if m.GameID == gameID {
			out = append(out, m)
		}
	}
	return out, nil
}

// newTestHandlerWithMatches mirrors newTestHandler (authz_regression_test.go)
// but also returns the fakeMatchRepo, since this file's tests need to seed a
// fixture Game and then inspect match-repository state after a rejected
// RecordMatchResult call.
func newTestHandlerWithMatches() (*grpcapi.Handler, *fakeGameRepo, *fakeMatchRepo) {
	gameRepo := newFakeGameRepo()
	matchRepo := newFakeMatchRepo()
	svc := app.NewService(app.ServiceOptions{
		IDs:           &fakeIDs{},
		Games:         gameRepo,
		Registrations: newFakeRegistrationRepo(),
		Waitlist:      newFakeWaitlistRepo(),
		Matches:       matchRepo,
		GameAdmins:    newFakeGameAdminRepo(),
	})
	return grpcapi.NewHandler(svc, nil, nil), gameRepo, matchRepo
}

// --- RecordMatchResult: object-level (BOLA) regression ---------------------

// TestRecordMatchResult_RejectsNonHostNonAdmin is the ticket's required
// test: a Player who is neither the Game's Host nor an assigned Game Admin
// attempts RecordMatchResult against someone else's Game, through the real
// handler -> app -> domain path, and the request is rejected with the
// correctly mapped status — not a 500, not a silent success.
func TestRecordMatchResult_RejectsNonHostNonAdmin(t *testing.T) {
	h, gameRepo, matchRepo := newTestHandlerWithMatches()

	game := seedGame(t, gameRepo, "game-1", 4)

	_, err := h.RecordMatchResult(ctxAs("random-player"), &socialplayv1.RecordMatchResultRequest{
		GameId:                   game.ID,
		Players:                  []string{"player-1", "player-2"},
		Score:                    map[string]int32{"player-1": 11, "player-2": 7},
		AssignedGameAdminUserIds: []string{"admin-1"},
	})
	if err == nil {
		t.Fatal("RecordMatchResult(random-player) succeeded silently — a Player who is neither the Game's Host nor an assigned Game Admin was able to record a match result (BOLA regression)")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("RecordMatchResult(random-player) returned a non-gRPC-status error: %v (a client can't map this to a clean HTTP status)", err)
	}
	if st.Code() == codes.Internal {
		t.Fatalf("RecordMatchResult(random-player) mapped to Internal (500-shaped) — want PermissionDenied (403-shaped): %v", err)
	}
	if st.Code() != codes.PermissionDenied {
		t.Fatalf("RecordMatchResult(random-player) status code = %v, want PermissionDenied (403-shaped)", st.Code())
	}

	// Belt-and-braces, per the ticket's "not a silent success": prove no
	// Match was persisted, not just that an error came back on the wire.
	if len(matchRepo.matches) != 0 {
		t.Errorf("match repo has %d Matches after a rejected RecordMatchResult, want 0 (the rejected attempt must not have any side effect)", len(matchRepo.matches))
	}
}

// TestRecordMatchResult_AllowsHost is the symmetric positive-path case: the
// Game's own Host recording a result succeeds through the same handler
// path.
func TestRecordMatchResult_AllowsHost(t *testing.T) {
	h, gameRepo, matchRepo := newTestHandlerWithMatches()

	game := seedGame(t, gameRepo, "game-2", 4)

	resp, err := h.RecordMatchResult(ctxAs("host-1"), &socialplayv1.RecordMatchResultRequest{
		GameId:  game.ID,
		Players: []string{"player-1", "player-2"},
		Score:   map[string]int32{"player-1": 11, "player-2": 7},
	})
	if err != nil {
		t.Fatalf("RecordMatchResult(host-1) (the Game's own Host) should succeed, got: %v", err)
	}
	if resp.GetMatch().GetId() == "" {
		t.Errorf("Match.Id is empty, want a generated id")
	}
	if len(matchRepo.matches) != 1 {
		t.Errorf("match repo has %d Matches, want 1 after the Host's successful RecordMatchResult", len(matchRepo.matches))
	}
}

// TestRecordMatchResult_AllowsAssignedGameAdmin mirrors
// TestRecordMatchResult_AllowsHost for the assigned-Game-Admin path: an
// admin who is not the Host may still record the match result.
func TestRecordMatchResult_AllowsAssignedGameAdmin(t *testing.T) {
	h, gameRepo, _ := newTestHandlerWithMatches()

	game := seedGame(t, gameRepo, "game-3", 4)

	resp, err := h.RecordMatchResult(ctxAs("admin-2"), &socialplayv1.RecordMatchResultRequest{
		GameId:                   game.ID,
		Players:                  []string{"player-1", "player-2"},
		Score:                    map[string]int32{"player-1": 11, "player-2": 7},
		AssignedGameAdminUserIds: []string{"admin-1", "admin-2"},
	})
	if err != nil {
		t.Fatalf("RecordMatchResult(admin-2) (an assigned Game Admin) should succeed, got: %v", err)
	}
	if resp.GetMatch().GetGameId() != game.ID {
		t.Errorf("Match.GameId = %q, want %q", resp.GetMatch().GetGameId(), game.ID)
	}
}

// --- RecordMatchResult: cancelled-Game precondition -------------------------

// TestRecordMatchResult_CancelledGameRejected proves the FailedPrecondition
// mapping at the handler level: even the Game's own Host may not record a
// result against a cancelled Game.
func TestRecordMatchResult_CancelledGameRejected(t *testing.T) {
	h, gameRepo, matchRepo := newTestHandlerWithMatches()

	game := seedGame(t, gameRepo, "game-4", 4)
	cancelled := game
	if err := cancelled.Cancel(); err != nil {
		t.Fatalf("fixture cancel should succeed: %v", err)
	}
	gameRepo.games[cancelled.ID] = cancelled

	_, err := h.RecordMatchResult(ctxAs("host-1"), &socialplayv1.RecordMatchResultRequest{
		GameId:  cancelled.ID,
		Players: []string{"player-1", "player-2"},
		Score:   map[string]int32{"player-1": 11, "player-2": 7},
	})
	if err == nil {
		t.Fatal("RecordMatchResult against a cancelled Game succeeded silently, want FailedPrecondition")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("returned a non-gRPC-status error: %v", err)
	}
	if st.Code() != codes.FailedPrecondition {
		t.Fatalf("status code = %v, want FailedPrecondition", st.Code())
	}
	if len(matchRepo.matches) != 0 {
		t.Errorf("match repo has %d Matches after a rejected RecordMatchResult, want 0", len(matchRepo.matches))
	}
}

// --- RecordMatchResult / ListMatchesForGame: malformed-id boundary guard ---

// TestRecordMatchResult_MalformedGameIDIsNotFound and
// TestListMatchesForGame_MalformedGameIDIsNotFound prove T10.4's own
// malformed-ID guard (reusing app's uuidShape helper, T10.7's precedent)
// survives the full handler stack — the exhaustive per-input-string sweep
// already lives at the app layer (internal/socialplay/app/malformed_id_test.go);
// this is the "one representative malformed id reaches NotFound through the
// real handler too" smoke check.
func TestRecordMatchResult_MalformedGameIDIsNotFound(t *testing.T) {
	h, _, _ := newTestHandlerWithMatches()

	_, err := h.RecordMatchResult(ctxAs("host-1"), &socialplayv1.RecordMatchResultRequest{
		GameId:  "not-a-uuid",
		Players: []string{"player-1", "player-2"},
		Score:   map[string]int32{"player-1": 11, "player-2": 7},
	})
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("returned a non-gRPC-status error: %v", err)
	}
	if st.Code() != codes.NotFound {
		t.Fatalf("status code = %v, want NotFound", st.Code())
	}
}

func TestListMatchesForGame_MalformedGameIDIsNotFound(t *testing.T) {
	ctx := context.Background()
	h, _, _ := newTestHandlerWithMatches()

	_, err := h.ListMatchesForGame(ctx, &socialplayv1.ListMatchesForGameRequest{GameId: "not-a-uuid"})
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("returned a non-gRPC-status error: %v", err)
	}
	if st.Code() != codes.NotFound {
		t.Fatalf("status code = %v, want NotFound", st.Code())
	}
}

// --- ListMatchesForGame: happy path -----------------------------------------

// TestListMatchesForGame_ReturnsRecordedMatches proves the read path end to
// end through the real handler.
func TestListMatchesForGame_ReturnsRecordedMatches(t *testing.T) {
	ctx := context.Background()
	h, gameRepo, _ := newTestHandlerWithMatches()

	game := seedGame(t, gameRepo, "game-5", 4)
	if _, err := h.RecordMatchResult(ctxAs("host-1"), &socialplayv1.RecordMatchResultRequest{
		GameId: game.ID, Players: []string{"player-1", "player-2"},
		Score: map[string]int32{"player-1": 11, "player-2": 7},
	}); err != nil {
		t.Fatalf("fixture RecordMatchResult failed: %v", err)
	}

	resp, err := h.ListMatchesForGame(ctx, &socialplayv1.ListMatchesForGameRequest{GameId: game.ID})
	if err != nil {
		t.Fatalf("ListMatchesForGame err: %v", err)
	}
	if len(resp.GetMatches()) != 1 || resp.GetMatches()[0].GetGameId() != game.ID {
		t.Fatalf("ListMatchesForGame(%s) = %+v, want one match for this game", game.ID, resp.GetMatches())
	}
}
