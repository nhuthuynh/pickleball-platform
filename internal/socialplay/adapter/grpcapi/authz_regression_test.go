// T5.5 — object-level authorization regression tests, run through the real
// gRPC handler (not just the domain-level unit test T5.2 already has in
// internal/socialplay/domain/registration_test.go). This is the "does the
// guarantee survive the full stack" test the T5 sprint plan's kickoff note
// asks for (docs/process/t5-sprint-plan.md, T5.5).
//
// This is a handler-level test (real grpcapi.Handler + real app.Service +
// real domain, with in-memory fakes standing in for port.GameRepository/
// port.RegistrationRepository) rather than a `-tags=integration`
// testcontainers-go test like T4's concurrency_integration_test.go or
// T5.4's capacity_concurrency_integration_test.go. Two independent reasons:
//
//  1. The ticket (docs/process/t5-sprint-plan.md, T5.5) explicitly allows
//     "a lighter handler-level test if a full Postgres round trip isn't
//     needed to prove the point" — the object-level check under test lives
//     entirely in domain.Registration.Cancel (see that file), which
//     port.RegistrationRepository doesn't influence; a real Postgres round
//     trip would add infrastructure, not proof.
//  2. This environment has no Docker daemon (docker CLI is present, `docker
//     ps` fails to dial the socket) — the same gap
//     concurrency_integration_test.go's package comment already documents
//     for this context. CLAUDE.md rule 10 ("prove it, don't assume it")
//     means this ticket's own regression test needs to actually run in an
//     environment we can execute it in, not just be plausible in CI. A
//     testcontainers-based version of this test can be added later
//     following the existing pattern in internal/socialplay/adapter/postgres
//     if the team wants Postgres-round-trip coverage of the same
//     assertion; it would not exercise any code this file doesn't already
//     cover, since the ownership check has no SQL involved.
package grpcapi_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/nhuthuynh/white-label/internal/socialplay/adapter/grpcapi"
	"github.com/nhuthuynh/white-label/internal/socialplay/app"
	"github.com/nhuthuynh/white-label/internal/socialplay/domain"

	socialplayv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/socialplay/v1"
)

// --- in-memory port fakes -------------------------------------------------
//
// These stand in for internal/socialplay/adapter/postgres for this test
// only. They implement the exact same port.GameRepository/
// port.RegistrationRepository interfaces the real Postgres adapter does, so
// app.Service and grpcapi.Handler run unmodified, real production code —
// only the persistence boundary is faked.

type fakeGameRepo struct {
	games map[string]domain.Game
}

func newFakeGameRepo() *fakeGameRepo {
	return &fakeGameRepo{games: map[string]domain.Game{}}
}

func (f *fakeGameRepo) Create(_ context.Context, g domain.Game) (domain.Game, error) {
	f.games[g.ID] = g
	return g, nil
}

func (f *fakeGameRepo) GetByID(_ context.Context, id string) (domain.Game, error) {
	g, ok := f.games[id]
	if !ok {
		return domain.Game{}, domain.ErrGameNotFound
	}
	return g, nil
}

type fakeRegistrationRepo struct {
	regs map[string]domain.Registration
}

func newFakeRegistrationRepo() *fakeRegistrationRepo {
	return &fakeRegistrationRepo{regs: map[string]domain.Registration{}}
}

func (f *fakeRegistrationRepo) Create(_ context.Context, r domain.Registration) (domain.Registration, error) {
	f.regs[r.ID] = r
	return r, nil
}

func (f *fakeRegistrationRepo) GetByID(_ context.Context, id string) (domain.Registration, error) {
	r, ok := f.regs[id]
	if !ok {
		return domain.Registration{}, domain.ErrRegistrationNotFound
	}
	return r, nil
}

func (f *fakeRegistrationRepo) ListActiveForGame(_ context.Context, gameID string) ([]domain.Registration, error) {
	var out []domain.Registration
	for _, r := range f.regs {
		if r.GameID == gameID && r.Status != domain.RegistrationStatusCancelled {
			out = append(out, r)
		}
	}
	return out, nil
}

func (f *fakeRegistrationRepo) Update(_ context.Context, r domain.Registration) (domain.Registration, error) {
	if _, ok := f.regs[r.ID]; !ok {
		return domain.Registration{}, domain.ErrRegistrationNotFound
	}
	f.regs[r.ID] = r
	return r, nil
}

func (f *fakeRegistrationRepo) UpdatePaymentStatus(_ context.Context, id string, status domain.PaymentStatus) (domain.Registration, error) {
	r, ok := f.regs[id]
	if !ok {
		return domain.Registration{}, domain.ErrRegistrationNotFound
	}
	r.PaymentStatus = status
	f.regs[id] = r
	return r, nil
}

// fakeWaitlistRepo is a minimal, empty-always port.WaitlistRepository stand-in
// — neither RegisterForGame nor CancelRegistration's BOLA behaviour under
// test here depends on waitlist state, so this only needs to satisfy the
// interface (PromoteNext always reports no waiting entries, matching every
// fixture in this file never seeding one).
type fakeWaitlistRepo struct {
	entries map[string]domain.WaitlistEntry
}

func newFakeWaitlistRepo() *fakeWaitlistRepo {
	return &fakeWaitlistRepo{entries: map[string]domain.WaitlistEntry{}}
}

func (f *fakeWaitlistRepo) Create(_ context.Context, e domain.WaitlistEntry) (domain.WaitlistEntry, error) {
	f.entries[e.ID] = e
	return e, nil
}

func (f *fakeWaitlistRepo) GetByID(_ context.Context, id string) (domain.WaitlistEntry, error) {
	e, ok := f.entries[id]
	if !ok {
		return domain.WaitlistEntry{}, domain.ErrWaitlistEntryNotFound
	}
	return e, nil
}

func (f *fakeWaitlistRepo) ListForGame(_ context.Context, gameID string) ([]domain.WaitlistEntry, error) {
	var out []domain.WaitlistEntry
	for _, e := range f.entries {
		if e.GameID == gameID {
			out = append(out, e)
		}
	}
	return out, nil
}

func (f *fakeWaitlistRepo) PromoteNext(_ context.Context, gameID string, now time.Time) (domain.WaitlistEntry, error) {
	return domain.WaitlistEntry{}, domain.ErrNoWaitingEntries
}

func (f *fakeWaitlistRepo) ExpirePromotion(_ context.Context, id string, now time.Time) (domain.WaitlistEntry, error) {
	return domain.WaitlistEntry{}, domain.ErrWaitlistEntryNotFound
}

// fakeIDs is a deterministic, dependency-free port.IDGenerator stand-in
// (mirrors why internal/platform/idgen exists for production — tests get a
// predictable sequence instead of random UUIDs).
type fakeIDs struct{ n int }

func (f *fakeIDs) NewID() string {
	f.n++
	return fmt.Sprintf("id-%d", f.n)
}

// --- test setup ------------------------------------------------------------

// newTestHandler wires the real app.Service and the real grpcapi.Handler —
// exactly what cmd/server wires in production — against the in-memory
// fakes above. The reservation port is nil: neither RegisterForGame nor
// CancelRegistration (the two RPCs under test) touch it, only CreateGame
// does.
func newTestHandler() (*grpcapi.Handler, *fakeGameRepo, *fakeRegistrationRepo) {
	gameRepo := newFakeGameRepo()
	regRepo := newFakeRegistrationRepo()
	svc := app.NewService(&fakeIDs{}, gameRepo, regRepo, newFakeWaitlistRepo())
	return grpcapi.NewHandler(svc, nil), gameRepo, regRepo
}

func seedGame(t *testing.T, gameRepo *fakeGameRepo, id string, capacity int) domain.Game {
	t.Helper()
	start := time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	rng, err := domain.NewTimeRange(start, end)
	if err != nil {
		t.Fatalf("bad fixture range: %v", err)
	}
	g, err := domain.NewGame(id, "host-1", "facility-1", []string{"court-1"}, rng, capacity, domain.PaymentMethodEither, 0)
	if err != nil {
		t.Fatalf("bad fixture game: %v", err)
	}
	g, err = gameRepo.Create(context.Background(), g)
	if err != nil {
		t.Fatalf("failed to seed fixture game: %v", err)
	}
	return g
}

// --- CancelRegistration: object-level (BOLA) regression -------------------

// TestCancelRegistration_RejectsMismatchedActor is the ticket's required
// test: register Player A for a Game, then attempt CancelRegistration as
// Player B, through the real handler -> app -> domain path, and assert the
// request is rejected with the correctly mapped status — not a 500, not a
// silent success.
func TestCancelRegistration_RejectsMismatchedActor(t *testing.T) {
	ctx := context.Background()
	h, gameRepo, regRepo := newTestHandler()

	game := seedGame(t, gameRepo, "game-1", 4)

	regResp, err := h.RegisterForGame(ctx, &socialplayv1.RegisterForGameRequest{
		GameId:   game.ID,
		PlayerId: "player-A",
	})
	if err != nil {
		t.Fatalf("RegisterForGame(player-A) failed: %v", err)
	}
	registrationID := regResp.GetRegistration().GetId()

	// The BOLA attempt: Player B, a different actor_player_id than the one
	// that owns this registration, tries to cancel it.
	_, err = h.CancelRegistration(ctx, &socialplayv1.CancelRegistrationRequest{
		RegistrationId: registrationID,
		ActorPlayerId:  "player-B",
	})
	if err == nil {
		t.Fatal("CancelRegistration(player-B) succeeded silently — Player B was able to cancel Player A's registration (BOLA regression)")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("CancelRegistration(player-B) returned a non-gRPC-status error: %v (a client can't map this to a clean HTTP status)", err)
	}
	if st.Code() == codes.Internal {
		t.Fatalf("CancelRegistration(player-B) mapped to Internal (500-shaped) — want PermissionDenied (403-shaped): %v", err)
	}
	if st.Code() != codes.PermissionDenied {
		t.Fatalf("CancelRegistration(player-B) status code = %v, want PermissionDenied (403-shaped)", st.Code())
	}

	// Belt-and-braces, per the ticket's "not a silent success": prove the
	// registration itself was left untouched, not just that an error came
	// back on the wire.
	stored, err := regRepo.GetByID(ctx, registrationID)
	if err != nil {
		t.Fatalf("failed to re-fetch registration: %v", err)
	}
	if stored.Status != domain.RegistrationStatusRegistered {
		t.Errorf("registration status after rejected cancel = %v, want still %v (Player B's rejected attempt must not have any side effect)",
			stored.Status, domain.RegistrationStatusRegistered)
	}
}

// TestCancelRegistration_AllowsOwningActor is the symmetric positive-path
// case: without it, TestCancelRegistration_RejectsMismatchedActor alone
// couldn't tell "the ownership check correctly rejects a mismatched actor"
// apart from "CancelRegistration is broken and rejects everyone" — this
// pins down that the real owner's cancellation still succeeds through the
// same handler path.
func TestCancelRegistration_AllowsOwningActor(t *testing.T) {
	ctx := context.Background()
	h, gameRepo, _ := newTestHandler()

	game := seedGame(t, gameRepo, "game-2", 4)

	regResp, err := h.RegisterForGame(ctx, &socialplayv1.RegisterForGameRequest{
		GameId:   game.ID,
		PlayerId: "player-A",
	})
	if err != nil {
		t.Fatalf("RegisterForGame(player-A) failed: %v", err)
	}

	cancelResp, err := h.CancelRegistration(ctx, &socialplayv1.CancelRegistrationRequest{
		RegistrationId: regResp.GetRegistration().GetId(),
		ActorPlayerId:  "player-A",
	})
	if err != nil {
		t.Fatalf("CancelRegistration(player-A) (the owner) should succeed, got: %v", err)
	}
	if cancelResp.GetRegistration().GetStatus() != socialplayv1.RegistrationStatus_REGISTRATION_STATUS_CANCELLED {
		t.Errorf("registration status = %v, want REGISTRATION_STATUS_CANCELLED", cancelResp.GetRegistration().GetStatus())
	}
}

// --- CreateGame / Game.Cancel(): explicitly out of scope for this ticket --
//
// The ticket also asks for the equivalent regression test on CreateGame
// (only a Game's HostID may Cancel() it, T5.1) "if it fits within this
// ticket's points; if not, split into a follow-up rather than silently
// skip it."
//
// It does not fit: there is currently no CancelGame RPC at all.
// domain.Game.Cancel() (internal/socialplay/domain/game.go) takes no actor
// parameter — it isn't even ownership-checked at the domain level yet,
// unlike Registration.Cancel — and proto/pickleball/socialplay/v1/
// socialplay.proto exposes no CancelGame method, so there is no
// app.Service method or grpcapi.Handler RPC to test through the stack in
// the first place. Building one is proto + app + handler + regression-test
// work (closer to a T5.1/T5.4-shaped ticket than a T5.5-shaped regression
// test), not an extension of an existing pattern. Per the loop-mechanics
// rule on mis-scoped tickets, this is split into a follow-up rather than
// silently dropped — see the PR description and the new HANDOFF.md
// cross-cutting note this PR adds.
