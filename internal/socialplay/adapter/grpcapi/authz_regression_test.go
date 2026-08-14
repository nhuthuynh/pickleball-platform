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
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/nhuthuynh/white-label/internal/socialplay/adapter/grpcapi"
	"github.com/nhuthuynh/white-label/internal/socialplay/app"
	"github.com/nhuthuynh/white-label/internal/socialplay/domain"
	"github.com/nhuthuynh/white-label/internal/socialplay/port"

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

// ListGames is a minimal stub (T8.9): this suite's tests don't exercise the
// Discover & Join Games browse/filter read, but port.GameRepository now
// requires it.
func (f *fakeGameRepo) ListGames(_ context.Context, _ port.GameListingFilter) ([]port.GameListing, error) {
	return nil, nil
}

// UpdateStatus is the in-memory port.GameRepository.UpdateStatus stand-in
// (T12.4), backing CancelGame. Like the real single-column query, it writes
// only Status and leaves every other field on the stored Game untouched.
func (f *fakeGameRepo) UpdateStatus(_ context.Context, id string, status domain.Status) (domain.Game, error) {
	g, ok := f.games[id]
	if !ok {
		return domain.Game{}, domain.ErrGameNotFound
	}
	g.Status = status
	f.games[id] = g
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

// NewID mints UUID-shaped ids. It used to return "id-1", "id-2", ... — a
// shape uuidShape (internal/socialplay/app/service.go) rejects, so once
// T10.7 added its GameID/RegistrationID boundary guards, every test in this
// package that registers/joins-waitlist/cancels/records-a-match against a
// Game or Registration id minted by this generator started failing that
// guard (the identical class of pre-existing regression seedGameUUID's doc
// comment above describes and fixes for Game ids — see this PR's
// description). Mirrors internal/socialplay/app's sequentialIDs fix
// (service_test.go), which T10.7 already applied to that package's own
// fake generator but not to this one.
func (f *fakeIDs) NewID() string {
	f.n++
	return fmt.Sprintf("00000000-0000-4000-d000-%012d", f.n)
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
	svc := app.NewService(&fakeIDs{}, gameRepo, regRepo, newFakeWaitlistRepo(), newFakeMatchRepo())
	return grpcapi.NewHandler(svc, nil, nil), gameRepo, regRepo
}

// seedGameUUID deterministically maps a human-readable fixture label (e.g.
// "game-1") onto a UUID-shaped id, so seedGame's callers can keep using
// short, readable, distinguishable labels while every id actually stored
// in the fake repository (and sent over the wire in a handler-level test)
// satisfies uuidShape (internal/socialplay/app/service.go) — see seedGame's
// own doc comment for why this mapping exists at all. md5 is used purely
// as a deterministic 16-byte hash (not for anything security-sensitive):
// its 32 hex digits fit uuidShape's 8-4-4-4-12 grouping exactly, with no
// truncation needed, and distinct labels this package actually uses never
// collide.
func seedGameUUID(label string) string {
	sum := md5.Sum([]byte(label))
	h := hex.EncodeToString(sum[:])
	return fmt.Sprintf("%s-%s-%s-%s-%s", h[0:8], h[8:12], h[12:16], h[16:20], h[20:32])
}

// seedGame constructs and persists a fixture Game against the given
// in-memory fakeGameRepo, returning the persisted Game so callers can read
// back its (real, UUID-shaped) ID.
//
// id is a human-readable fixture label ("game-1", "game-2", ...), not the
// literal persisted ID: T10.7 added a uuidShape boundary guard to every
// public write handler taking a caller-supplied GameID (RegisterForGame,
// CancelRegistration's registration lookup, JoinWaitlist, and — this
// ticket — RecordMatchResult/ListMatchesForGame), so a non-UUID-shaped
// fixture id like the ones this file originally used would now be rejected
// at the boundary before ever reaching the fake repository, breaking every
// test that seeds one (a real regression this ticket found already present
// on this branch, via seedGameUUID's deterministic mapping below — see this
// PR's description). Every existing call site keeps passing its original
// short label unchanged; only this helper's internal ID assignment
// changed, since callers only ever consume the *returned* Game's ID field,
// never the raw label string.
func seedGame(t *testing.T, gameRepo *fakeGameRepo, id string, capacity int) domain.Game {
	t.Helper()
	start := time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	rng, err := domain.NewTimeRange(start, end)
	if err != nil {
		t.Fatalf("bad fixture range: %v", err)
	}
	g, err := domain.NewGame(seedGameUUID(id), "host-1", "facility-1", "venue-1", []string{"court-1"}, rng, capacity, domain.PaymentMethodEither, 0, domain.Money{Cents: 1500, Currency: "USD"})
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

// --- CancelGame: object-level (BOLA) regression, T12.4 --------------------
//
// This block replaces the placeholder T5.5 left here. That comment recorded
// why the CreateGame/Game.Cancel() half of T5.5 was split into a follow-up:
// there was no CancelGame RPC at all, domain.Game.Cancel() took no actor
// parameter, and socialplay.proto exposed no method to reach it — so there
// was nothing to test through the stack. T12.4 built all three (the
// ErrNotGameHost sentinel, Game.EnsureHost, app.Service.CancelGame, the
// RPC and the handler), so the follow-up is now the tests below.
//
// Same shape and same justification as the CancelRegistration tests above,
// and as T7.7's Facilities equivalent: real grpcapi.Handler -> real
// app.Service -> real domain.Game, with only the persistence boundary
// faked. No Docker, no Postgres — the check under test lives entirely in
// domain.Game.EnsureHost, which port.GameRepository does not influence, so
// a real database round trip would add infrastructure, not proof.
//
// CAVEAT, stated here and in the PR rather than left implicit: this is an
// object-level check given a *claimed* actor, not authentication.
// actor_player_id is caller-supplied; nothing yet verifies the caller is
// who they say they are. T12.8 (this sprint, not this ticket) migrates
// Social Play to a verified principal, at which point these checks become
// trustworthy end to end. Until then, what is proven below is exactly:
// "given a claimed actor, the Host-only rule is enforced through the full
// stack and maps to the right gRPC code."

// seedGameWithHost is seedGame's sibling for the CancelGame tests: the
// same fixture, but with a caller-chosen HostID, since every assertion
// below turns on who the Host actually is. seedGame hard-codes "host-1".
func seedGameWithHost(t *testing.T, gameRepo *fakeGameRepo, label, hostID string) domain.Game {
	t.Helper()
	start := time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	rng, err := domain.NewTimeRange(start, end)
	if err != nil {
		t.Fatalf("bad fixture range: %v", err)
	}
	g, err := domain.NewGame(seedGameUUID(label), hostID, "facility-1", "venue-1", []string{"court-1"}, rng, 4, domain.PaymentMethodEither, 0, domain.Money{Cents: 1500, Currency: "USD"})
	if err != nil {
		t.Fatalf("bad fixture game: %v", err)
	}
	g, err = gameRepo.Create(context.Background(), g)
	if err != nil {
		t.Fatalf("failed to seed fixture game: %v", err)
	}
	return g
}

// TestCancelGame_RejectsNonHostActor is this ticket's required proof:
// seed a Game hosted by player-A, attempt CancelGame as player-B through
// the real handler -> app -> domain path, and assert it is rejected with
// the correctly mapped status — not a 500, not a silent success.
func TestCancelGame_RejectsNonHostActor(t *testing.T) {
	ctx := context.Background()
	h, gameRepo, _ := newTestHandler()

	game := seedGameWithHost(t, gameRepo, "cancel-game-1", "player-A")

	// The BOLA attempt: player-B, who does not host this Game, tries to
	// cancel it.
	_, err := h.CancelGame(ctx, &socialplayv1.CancelGameRequest{
		GameId:        game.ID,
		ActorPlayerId: "player-B",
	})
	if err == nil {
		t.Fatal("CancelGame(player-B) succeeded silently — player B was able to cancel player A's Game (BOLA regression)")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("CancelGame(player-B) returned a non-gRPC-status error: %v (a client can't map this to a clean status)", err)
	}
	if st.Code() == codes.Internal {
		t.Fatalf("CancelGame(player-B) mapped to Internal — want PermissionDenied: %v", err)
	}
	if st.Code() != codes.PermissionDenied {
		t.Fatalf("CancelGame(player-B) status code = %v, want PermissionDenied", st.Code())
	}

	// Per the "not a silent success" standard the CancelRegistration test
	// above sets: prove the Game itself was left untouched, not merely that
	// an error came back on the wire.
	stored, err := gameRepo.GetByID(ctx, game.ID)
	if err != nil {
		t.Fatalf("failed to re-fetch game: %v", err)
	}
	if stored.Status != domain.StatusScheduled {
		t.Errorf("game status after rejected cancel = %v, want still %v (player B's rejected attempt must have no side effect)",
			stored.Status, domain.StatusScheduled)
	}
}

// TestCancelGame_AllowsHost is the symmetric positive-path case: without
// it, the rejection test alone couldn't distinguish "the ownership check
// correctly rejects a non-Host" from "CancelGame is broken and rejects
// everyone."
// It also deliberately calls through the generated
// socialplayv1.SocialPlayServiceServer *interface* rather than the concrete
// *grpcapi.Handler. grpcapi.Handler embeds
// UnimplementedSocialPlayServiceServer, so a CancelGame method with a
// subtly wrong signature would shadow the promoted stub by name, leave the
// real RPC answering codes.Unimplemented over the wire, and still let every
// direct-call test in this file pass. Binding the handler to the interface
// here is what makes that impossible: it is the same assertion
// cmd/server's RegisterSocialPlayServiceServer makes at wiring time.
func TestCancelGame_AllowsHost(t *testing.T) {
	ctx := context.Background()
	h, gameRepo, _ := newTestHandler()

	var srv socialplayv1.SocialPlayServiceServer = h

	game := seedGameWithHost(t, gameRepo, "cancel-game-2", "player-A")

	resp, err := srv.CancelGame(ctx, &socialplayv1.CancelGameRequest{
		GameId:        game.ID,
		ActorPlayerId: "player-A",
	})
	if err != nil {
		t.Fatalf("CancelGame(player-A) (the host) should succeed, got: %v", err)
	}
	if resp.GetGame().GetStatus() != socialplayv1.GameStatus_GAME_STATUS_CANCELLED {
		t.Errorf("game status = %v, want GAME_STATUS_CANCELLED", resp.GetGame().GetStatus())
	}

	stored, err := gameRepo.GetByID(ctx, game.ID)
	if err != nil {
		t.Fatalf("failed to re-fetch game: %v", err)
	}
	if stored.Status != domain.StatusCancelled {
		t.Errorf("persisted game status = %v, want %v (the response must reflect a real write)", stored.Status, domain.StatusCancelled)
	}
}

// TestCancelGame_RejectsGameAdminActor is the test that keeps ErrNotGameHost
// and ErrNotGameHostOrAdmin from being "simplified" into one sentinel. The
// actor here is a legitimate assigned Game Admin — RecordMatchResult
// accepts them, asserted directly below so the fixture cannot silently rot
// into a test of nothing — and CancelGame must still refuse them, because
// cancelling a Game is Host-only.
func TestCancelGame_RejectsGameAdminActor(t *testing.T) {
	ctx := context.Background()
	h, gameRepo, _ := newTestHandler()

	game := seedGameWithHost(t, gameRepo, "cancel-game-3", "player-A")
	const gameAdmin = "player-admin"

	// Fixture precondition, asserted rather than assumed: this actor really
	// is authorized for the Host-or-Admin operation.
	if _, err := h.RecordMatchResult(ctx, &socialplayv1.RecordMatchResultRequest{
		GameId:                   game.ID,
		Players:                  []string{"player-A", "player-B"},
		Score:                    map[string]int32{"player-A": 11, "player-B": 7},
		ActorUserId:              gameAdmin,
		AssignedGameAdminUserIds: []string{gameAdmin},
	}); err != nil {
		t.Fatalf("fixture precondition: %q must be accepted as an assigned game admin by RecordMatchResult, got: %v", gameAdmin, err)
	}

	// ... and is still refused the Host-only operation.
	_, err := h.CancelGame(ctx, &socialplayv1.CancelGameRequest{
		GameId:        game.ID,
		ActorPlayerId: gameAdmin,
	})
	if err == nil {
		t.Fatal("CancelGame(game admin) succeeded — cancelling a Game is Host-only; a Game Admin must not be able to cancel it")
	}
	st, _ := status.FromError(err)
	if st.Code() != codes.PermissionDenied {
		t.Fatalf("CancelGame(game admin) status code = %v, want PermissionDenied", st.Code())
	}

	stored, err := gameRepo.GetByID(ctx, game.ID)
	if err != nil {
		t.Fatalf("failed to re-fetch game: %v", err)
	}
	if stored.Status != domain.StatusScheduled {
		t.Errorf("game status after rejected cancel = %v, want still %v", stored.Status, domain.StatusScheduled)
	}
}

// TestCancelGame_ErrorMapping covers the rest of T12.4's error table
// through the same real handler: an empty actor is rejected like any other
// non-Host, an unknown or malformed game id answers NotFound (never a
// panic, never Internal — the T10.7-shaped boundary guard), and a second
// cancel by the rightful Host answers FailedPrecondition rather than
// silently succeeding.
func TestCancelGame_ErrorMapping(t *testing.T) {
	ctx := context.Background()

	t.Run("empty actor -> PermissionDenied", func(t *testing.T) {
		h, gameRepo, _ := newTestHandler()
		game := seedGameWithHost(t, gameRepo, "cancel-game-4", "player-A")

		_, err := h.CancelGame(ctx, &socialplayv1.CancelGameRequest{GameId: game.ID})
		st, _ := status.FromError(err)
		if st.Code() != codes.PermissionDenied {
			t.Fatalf("status code = %v, want PermissionDenied (an unidentified caller is never the host)", st.Code())
		}
	})

	t.Run("unknown game id -> NotFound", func(t *testing.T) {
		h, _, _ := newTestHandler()

		_, err := h.CancelGame(ctx, &socialplayv1.CancelGameRequest{
			GameId:        seedGameUUID("never-seeded"),
			ActorPlayerId: "player-A",
		})
		st, _ := status.FromError(err)
		if st.Code() != codes.NotFound {
			t.Fatalf("status code = %v, want NotFound", st.Code())
		}
	})

	t.Run("malformed game id -> NotFound", func(t *testing.T) {
		h, _, _ := newTestHandler()

		_, err := h.CancelGame(ctx, &socialplayv1.CancelGameRequest{
			GameId:        "not-a-uuid",
			ActorPlayerId: "player-A",
		})
		st, _ := status.FromError(err)
		if st.Code() != codes.NotFound {
			t.Fatalf("status code = %v, want NotFound (a malformed id must answer exactly as an unknown one does, not Internal or a panic)", st.Code())
		}
	})

	t.Run("already-cancelled game -> FailedPrecondition", func(t *testing.T) {
		h, gameRepo, _ := newTestHandler()
		game := seedGameWithHost(t, gameRepo, "cancel-game-5", "player-A")

		req := &socialplayv1.CancelGameRequest{GameId: game.ID, ActorPlayerId: "player-A"}
		if _, err := h.CancelGame(ctx, req); err != nil {
			t.Fatalf("first cancel should succeed, got: %v", err)
		}

		_, err := h.CancelGame(ctx, req)
		if err == nil {
			t.Fatal("a second cancel succeeded silently; it must be rejected")
		}
		st, _ := status.FromError(err)
		if st.Code() != codes.FailedPrecondition {
			t.Fatalf("status code = %v, want FailedPrecondition", st.Code())
		}
	})
}
