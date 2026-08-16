// T12.8 — the verified-principal half of this context's authorization
// regression suite. authz_regression_test.go (T5.5) and
// match_authz_regression_test.go (T10.4) proved the *ownership rules*: a
// caller who is not the registering player, the Host, or a Game Admin is
// rejected. This file proves the thing those rules were resting on is no
// longer a lie: **where the acting identity comes from**.
//
// Before this ticket the actor was a string the caller put in the request
// body. "Only the Host may cancel a Game" was therefore only ever "only a
// caller willing to type the Host's ID may cancel a Game" — a check that
// rejects nobody, since Game.host_id is returned on the public ListGames
// browse response. These tests assert the four cases the T12.8 ticket
// requires, for every now-enforced RPC:
//
//	(a) a valid principal for the real owner            -> succeeds
//	(b) a valid principal for a different user          -> PermissionDenied
//	(c) no principal at all                             -> Unauthenticated
//	(d) a wire actor_player_id/actor_user_id claiming to
//	    be the owner, with a non-owner principal or none -> rejected
//
// (c) and (b) must stay distinct codes: "I do not know who you are" versus "I
// know who you are and you may not do this" (ADR-0013 §5). (d) is the one that
// would have been silently broken by a naive migration — a handler that read
// the principal but fell back to req.GetActorPlayerId() when there wasn't one
// would pass (a), (b) and (c) and still be exactly as exploitable as before.
// It was mutation-checked rather than assumed; see the PR body.
package grpcapi_test

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/nhuthuynh/white-label/internal/platform/auth"
	"github.com/nhuthuynh/white-label/internal/socialplay/adapter/grpcapi"
	"github.com/nhuthuynh/white-label/internal/socialplay/app"
	"github.com/nhuthuynh/white-label/internal/socialplay/domain"

	socialplayv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/socialplay/v1"
)

// ctxAs builds the context the auth interceptors produce for a caller whose
// bearer token verified as subject. It goes through the real exported
// auth.ContextWithPrincipal rather than stuffing a context key directly, so
// these tests break if that contract changes.
func ctxAs(subject string) context.Context {
	return auth.ContextWithPrincipal(context.Background(), auth.Principal{
		Subject: subject,
		Issuer:  "https://issuer.test/",
	})
}

// anonymous is the context of a caller who presented no token, or one the
// verifier rejected — the observe-only interceptor leaves the context
// untouched in both cases, so they are indistinguishable here by design.
func anonymous() context.Context { return context.Background() }

// fakeIdentityLookup implements port.IdentityLookup for every test in this
// package (T29.2, closing the Social Play third of #164): a deterministic
// subject == User.ID passthrough. This is deliberately NOT a no-op stand-in
// the way internal/socialplay/app's own package-private fake is (that
// package's tests never reach the funnel at all) — every test file in THIS
// package drives real Handler.actor()/resolveUserID calls through ctxAs, so
// this fake's resolution rule is genuinely exercised on every RPC call, not
// merely present to satisfy construction. Passthrough is what keeps every
// existing fixture's ownership comparisons (Game.HostID seeded as the raw
// literal "host-1", compared against ctxAs("host-1")'s resolved actor) true
// exactly as they were before T29.2 — see internal/socialplay/domain.Game.
// EnsureHost's own doc comment for why the comparison logic itself is
// unchanged by this ticket, only what feeds both sides.
//
// An empty subject, or unregisteredSubject specifically, resolves to
// domain.ErrUserNotFound — mirroring the real adapter/identity.Lookup's
// documented behaviour for an unregistered/empty subject (see that package's
// own doc comment). The blank case is needed by anonymous()-shaped tests
// that reach resolveUserID with a blank caller-supplied admin target
// (AssignGameAdmin/RevokeGameAdmin's own blank-skips-resolution guard means
// this path is normally unreached, but a correct fake should not silently
// accept blank as a real identity either). unregisteredSubject exists so
// error_mapping_test.go's mutation-check table (every domain sentinel must
// have a mapping row) has a real, well-formed-but-unresolvable subject to
// drive AssignGameAdmin/RevokeGameAdmin's target-user resolution through —
// see that file's own ErrUserNotFound case.
type fakeIdentityLookup struct{}

// unregisteredSubject is a well-formed subject fakeIdentityLookup never
// resolves — the "asked to grant/revoke Game-Admin authority for a subject
// nobody has ever registered" case (T29.2).
const unregisteredSubject = "auth0|never-registered"

func (fakeIdentityLookup) UserIDBySubject(_ context.Context, subject string) (string, error) {
	if subject == "" || subject == unregisteredSubject {
		return "", domain.ErrUserNotFound
	}
	return subject, nil
}

// newPrincipalTestHandler wires the real app.Service and the real
// grpcapi.Handler against this package's in-memory fakes, plus the
// fakeReservation entry_fee_test.go already defines — unlike
// authz_regression_test.go's newTestHandler, these tests drive CreateGame,
// which is the one RPC that actually calls the CourtReservation port.
func newPrincipalTestHandler() (*grpcapi.Handler, *fakeGameRepo, *fakeRegistrationRepo) {
	gameRepo := newFakeGameRepo()
	regRepo := newFakeRegistrationRepo()
	svc := app.NewService(app.ServiceOptions{
		Identity:      fakeIdentityLookup{},
		IDs:           &fakeIDs{},
		Games:         gameRepo,
		Registrations: regRepo,
		Waitlist:      newFakeWaitlistRepo(),
		Matches:       newFakeMatchRepo(),
		GameAdmins:    newFakeGameAdminRepo(),
	})
	return grpcapi.NewHandler(svc, &fakeReservation{}, nil), gameRepo, regRepo
}

// principalGameReq is createGameReq with a caller-chosen capacity. host_id is
// deliberately left populated with a value that is NOT the acting principal,
// so every CreateGame call in this file doubles as a check that the wire field
// is ignored.
func principalGameReq(capacity int32) *socialplayv1.CreateGameRequest {
	req := createGameReq(nil)
	req.Capacity = capacity
	return req
}

// seedHostedGameWithCapacity is seedHostedGame with an explicit capacity, for
// the waitlist case that needs a Game it can actually fill.
func seedHostedGameWithCapacity(t *testing.T, h *grpcapi.Handler, capacity int32) *socialplayv1.Game {
	t.Helper()
	resp, err := h.CreateGame(ctxAs(hostSubject), principalGameReq(capacity))
	if err != nil {
		t.Fatalf("seeding a game as its host should succeed, got: %v", err)
	}
	return resp.GetGame()
}

const (
	hostSubject     = "auth0|host-1"
	playerSubject   = "auth0|player-1"
	attackerSubject = "auth0|attacker-9"
)

// requireCode asserts the exact gRPC code, and reports Internal separately — a
// 500-shaped answer to an authorization question is its own bug class (it
// means an error escaped toStatus), and the T5.5 tests this file joins already
// call it out that way.
func requireCode(t *testing.T, what string, err error, want codes.Code) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s succeeded, want %v", what, want)
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("%s returned a non-gRPC-status error: %v", what, err)
	}
	if st.Code() == codes.Internal && want != codes.Internal {
		t.Fatalf("%s mapped to Internal (500-shaped), want %v: %v", what, want, err)
	}
	if st.Code() != want {
		t.Fatalf("%s code = %v, want %v (%v)", what, st.Code(), want, st.Message())
	}
}

// seedHostedGame creates a Game hosted by hostSubject through the real
// CreateGame handler. Note there is no HostId on the request: the Host is
// minted from the principal now, which is itself part of what is under test —
// see TestCreateGame_HostComesFromPrincipalNotWire.
func seedHostedGame(t *testing.T, h *grpcapi.Handler) *socialplayv1.Game {
	t.Helper()
	resp, err := h.CreateGame(ctxAs(hostSubject), principalGameReq(4))
	if err != nil {
		t.Fatalf("seeding a game as its host should succeed, got: %v", err)
	}
	return resp.GetGame()
}

// --- (a) and (b): the principal decides ---------------------------------

func TestEnforcedRPCs_OwnerPrincipalSucceeds(t *testing.T) {
	h, _, _ := newPrincipalTestHandler()
	game := seedHostedGame(t, h)

	// The registering player is a different human from the Host, and each
	// acts under their own principal.
	regResp, err := h.RegisterForGame(ctxAs(playerSubject), &socialplayv1.RegisterForGameRequest{
		GameId: game.GetId(),
	})
	if err != nil {
		t.Fatalf("RegisterForGame under the player's own principal should succeed, got: %v", err)
	}
	if got := regResp.GetRegistration().GetPlayerId(); got != playerSubject {
		t.Errorf("Registration.PlayerId = %q, want %q — the registering player was not taken from the principal", got, playerSubject)
	}

	if _, err := h.RecordMatchResult(ctxAs(hostSubject), &socialplayv1.RecordMatchResultRequest{
		GameId:  game.GetId(),
		Players: []string{playerSubject, hostSubject},
		Score:   map[string]int32{playerSubject: 11, hostSubject: 7},
	}); err != nil {
		t.Errorf("RecordMatchResult as the Host's principal should succeed, got: %v", err)
	}

	if _, err := h.CancelRegistration(ctxAs(playerSubject), &socialplayv1.CancelRegistrationRequest{
		RegistrationId: regResp.GetRegistration().GetId(),
	}); err != nil {
		t.Errorf("CancelRegistration under the registering player's own principal should succeed, got: %v", err)
	}

	if _, err := h.CancelGame(ctxAs(hostSubject), &socialplayv1.CancelGameRequest{
		GameId: game.GetId(),
	}); err != nil {
		t.Errorf("CancelGame as the Host's principal should succeed, got: %v", err)
	}
}

func TestEnforcedRPCs_NonOwnerPrincipalIsPermissionDenied(t *testing.T) {
	h, _, _ := newPrincipalTestHandler()
	game := seedHostedGame(t, h)

	regResp, err := h.RegisterForGame(ctxAs(playerSubject), &socialplayv1.RegisterForGameRequest{
		GameId: game.GetId(),
	})
	if err != nil {
		t.Fatalf("seeding a registration should succeed: %v", err)
	}

	ctx := ctxAs(attackerSubject)

	_, err = h.CancelRegistration(ctx, &socialplayv1.CancelRegistrationRequest{
		RegistrationId: regResp.GetRegistration().GetId(),
	})
	requireCode(t, "CancelRegistration with a non-owner principal", err, codes.PermissionDenied)

	_, err = h.RecordMatchResult(ctx, &socialplayv1.RecordMatchResultRequest{
		GameId:  game.GetId(),
		Players: []string{playerSubject, hostSubject},
		Score:   map[string]int32{playerSubject: 11, hostSubject: 7},
	})
	requireCode(t, "RecordMatchResult with a non-Host principal", err, codes.PermissionDenied)

	_, err = h.CancelGame(ctx, &socialplayv1.CancelGameRequest{GameId: game.GetId()})
	requireCode(t, "CancelGame with a non-Host principal", err, codes.PermissionDenied)
}

// --- (c): no principal is Unauthenticated, NOT PermissionDenied ---------

// TestEnforcedRPCs_NoPrincipalIsUnauthenticated pins the distinction ADR-0013
// §5 establishes. Asserting merely "an error" here would let a handler that
// answered PermissionDenied pass — and PermissionDenied tells an anonymous
// caller "you are known and refused", which is both wrong and unactionable
// (the fix is to authenticate, and that answer does not say so).
func TestEnforcedRPCs_NoPrincipalIsUnauthenticated(t *testing.T) {
	h, _, _ := newPrincipalTestHandler()

	// A Game and a Registration must exist for the ownership checks to be
	// reachable at all, so that this test proves the principal check runs
	// *first* rather than the call failing for lack of a fixture.
	game := seedHostedGame(t, h)
	regResp, err := h.RegisterForGame(ctxAs(playerSubject), &socialplayv1.RegisterForGameRequest{
		GameId: game.GetId(),
	})
	if err != nil {
		t.Fatalf("seeding a registration should succeed: %v", err)
	}

	ctx := anonymous()

	_, err = h.CreateGame(ctx, principalGameReq(4))
	requireCode(t, "CreateGame with no principal", err, codes.Unauthenticated)

	_, err = h.RegisterForGame(ctx, &socialplayv1.RegisterForGameRequest{GameId: game.GetId()})
	requireCode(t, "RegisterForGame with no principal", err, codes.Unauthenticated)

	_, err = h.JoinWaitlist(ctx, &socialplayv1.JoinWaitlistRequest{GameId: game.GetId()})
	requireCode(t, "JoinWaitlist with no principal", err, codes.Unauthenticated)

	_, err = h.CancelRegistration(ctx, &socialplayv1.CancelRegistrationRequest{
		RegistrationId: regResp.GetRegistration().GetId(),
	})
	requireCode(t, "CancelRegistration with no principal", err, codes.Unauthenticated)

	_, err = h.RecordMatchResult(ctx, &socialplayv1.RecordMatchResultRequest{
		GameId:  game.GetId(),
		Players: []string{playerSubject, hostSubject},
		Score:   map[string]int32{playerSubject: 11, hostSubject: 7},
	})
	requireCode(t, "RecordMatchResult with no principal", err, codes.Unauthenticated)

	_, err = h.CancelGame(ctx, &socialplayv1.CancelGameRequest{GameId: game.GetId()})
	requireCode(t, "CancelGame with no principal", err, codes.Unauthenticated)
}

// --- (d) the load-bearing case: the wire field is ignored ----------------

// TestEnforcedRPCs_WireActorClaimingOwnershipIsIgnored is the whole point of
// the sprint. Each request below carries the deprecated actor field set to the
// real owner — the exact bytes that succeeded before this ticket — while the
// verified principal is either a different user or absent entirely.
//
// If any of these succeeds, the migration is cosmetic: the wire field is still
// what authorizes the call, and an attacker who can read host_id off the
// public ListGames response still controls every Game on the platform.
func TestEnforcedRPCs_WireActorClaimingOwnershipIsIgnored(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
		want codes.Code
	}{
		{
			// The principal says attacker, the body says owner. The body must
			// lose, and the answer must be "you may not", not "who are you".
			name: "non-owner principal, wire field claims the owner",
			ctx:  ctxAs(attackerSubject),
			want: codes.PermissionDenied,
		},
		{
			// No principal at all, body claims the owner. This is the precise
			// fallback the ticket forbids: a handler that trusted the wire
			// field when there was no principal would return OK here.
			name: "no principal, wire field claims the owner",
			ctx:  anonymous(),
			want: codes.Unauthenticated,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, gameRepo, regRepo := newPrincipalTestHandler()
			game := seedHostedGame(t, h)
			regResp, err := h.RegisterForGame(ctxAs(playerSubject), &socialplayv1.RegisterForGameRequest{
				GameId: game.GetId(),
			})
			if err != nil {
				t.Fatalf("seeding a registration should succeed: %v", err)
			}
			registrationID := regResp.GetRegistration().GetId()

			_, err = h.CancelRegistration(tc.ctx, &socialplayv1.CancelRegistrationRequest{
				RegistrationId: registrationID,
				ActorPlayerId:  playerSubject, // the lie
			})
			requireCode(t, "CancelRegistration", err, tc.want)

			_, err = h.CancelGame(tc.ctx, &socialplayv1.CancelGameRequest{
				GameId:        game.GetId(),
				ActorPlayerId: hostSubject, // the lie
			})
			requireCode(t, "CancelGame", err, tc.want)

			_, err = h.RecordMatchResult(tc.ctx, &socialplayv1.RecordMatchResultRequest{
				GameId:      game.GetId(),
				Players:     []string{playerSubject, hostSubject},
				Score:       map[string]int32{playerSubject: 11, hostSubject: 7},
				ActorUserId: hostSubject, // the lie
			})
			requireCode(t, "RecordMatchResult", err, tc.want)

			// A rejected call must have no side effect: the Game is still
			// scheduled and the Registration still active.
			if got := gameRepo.games[game.GetId()].Status; got != domain.StatusScheduled {
				t.Errorf("Game.Status = %v after rejected CancelGame — a rejected call wrote anyway", got)
			}
			if got := regRepo.regs[registrationID].Status; got != domain.RegistrationStatusRegistered {
				t.Errorf("Registration.Status = %v after rejected CancelRegistration — a rejected call wrote anyway", got)
			}
		})
	}
}

// TestCreateGame_HostComesFromPrincipalNotWire is case (d) for the RPC that
// *writes* Host-ship rather than checking it.
//
// A caller sends host_id = the victim. If that field were still honored, the
// attacker would have minted a Game hosted by someone else — who could then
// neither cancel it nor record its results, while the attacker could do both.
func TestCreateGame_HostComesFromPrincipalNotWire(t *testing.T) {
	h, _, _ := newPrincipalTestHandler()

	req := principalGameReq(4)
	// nolint:staticcheck // SA1019: setting the deprecated field IS the test.
	req.HostId = hostSubject // the lie

	resp, err := h.CreateGame(ctxAs(attackerSubject), req)
	if err != nil {
		t.Fatalf("CreateGame with a valid principal should succeed: %v", err)
	}

	if got := resp.GetGame().GetHostId(); got != attackerSubject {
		t.Errorf("Game.HostId = %q, want %q — host_id was taken from the wire, so a caller can create a Game hosted by someone else", got, attackerSubject)
	}
}

// TestJoinWaitlist_PlayerComesFromPrincipalNotWire is the same assertion for
// the waitlist entry point. A WaitlistEntry's player becomes a Registration's
// player on promotion, so a claimed value here propagates into the aggregate
// CancelRegistration guards.
func TestJoinWaitlist_PlayerComesFromPrincipalNotWire(t *testing.T) {
	h, _, _ := newPrincipalTestHandler()
	// Capacity 1, filled by the Host, so the next joiner goes to the waitlist.
	game := seedHostedGameWithCapacity(t, h, 1)

	if _, err := h.RegisterForGame(ctxAs(hostSubject), &socialplayv1.RegisterForGameRequest{
		GameId: game.GetId(),
	}); err != nil {
		t.Fatalf("filling the game should succeed: %v", err)
	}

	resp, err := h.JoinWaitlist(ctxAs(attackerSubject), &socialplayv1.JoinWaitlistRequest{
		GameId:   game.GetId(),
		PlayerId: playerSubject, // the lie
	})
	if err != nil {
		t.Fatalf("JoinWaitlist with a valid principal should succeed: %v", err)
	}

	if got := resp.GetEntry().GetPlayerId(); got != attackerSubject {
		t.Errorf("WaitlistEntry.PlayerId = %q, want %q — player_id was taken from the wire", got, attackerSubject)
	}
}

// TestRegisterForGame_PlayerComesFromPrincipalNotWire is the same assertion
// for RegisterForGame: a Registration minted for a claimed player_id could
// never be cancelled by the verified human it names.
func TestRegisterForGame_PlayerComesFromPrincipalNotWire(t *testing.T) {
	h, _, _ := newPrincipalTestHandler()
	game := seedHostedGame(t, h)

	resp, err := h.RegisterForGame(ctxAs(attackerSubject), &socialplayv1.RegisterForGameRequest{
		GameId:   game.GetId(),
		PlayerId: playerSubject, // the lie
	})
	if err != nil {
		t.Fatalf("RegisterForGame with a valid principal should succeed: %v", err)
	}

	if got := resp.GetRegistration().GetPlayerId(); got != attackerSubject {
		t.Errorf("Registration.PlayerId = %q, want %q — player_id was taken from the wire", got, attackerSubject)
	}
}
