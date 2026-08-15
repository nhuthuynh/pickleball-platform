// T14.5 — the roster read admits an assigned Game Admin (partial fix for
// #147), proved through the real handler -> app -> domain path.
//
// roster_authz_test.go (T13.6) is the four-case proof that this RPC stopped
// being public. This file is the fifth case that ticket could not write: the
// Game Admin who *should* have been entitled all along and was refused because
// "assigned Game Admin" was, at the time, whatever the caller typed. T14.4 made
// it a stored fact; these tests are what that fact bought.
//
// Everything here goes through AssignGameAdmin, the real Host-only write RPC.
// There is no seam for declaring someone an admin, which is the property under
// test as much as the read itself — a test that reached into the fake
// repository to plant a row would still pass if the handler had been wired to
// some other, forgeable source of admin identity.
package grpcapi_test

import (
	"testing"

	"google.golang.org/grpc/codes"

	"github.com/nhuthuynh/white-label/internal/socialplay/adapter/grpcapi"

	socialplayv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/socialplay/v1"
)

// assignRosterAdmin has the Game's Host grant adminSubject Game-Admin authority
// over gameID, through the real RPC.
func assignRosterAdmin(t *testing.T, h *grpcapi.Handler, gameID, adminSubject string) {
	t.Helper()

	if _, err := h.AssignGameAdmin(ctxAs(hostSubject), &socialplayv1.AssignGameAdminRequest{
		GameId: gameID,
		UserId: adminSubject,
	}); err != nil {
		t.Fatalf("the Host assigning %q as a Game Admin: %v", adminSubject, err)
	}
}

// adminSubject is the Game Admin these tests assign. Deliberately distinct from
// playerSubject and attackerSubject (principal_authz_test.go) so no row can
// pass by accidentally sharing an identity with an already-entitled actor.
const adminSubject = "auth0|game-admin-1"

// TestListRegistrationsForGame_AssignedAdminPrincipalReadsTheRoster is the
// headline case. It asserts the roster actually arrives rather than merely that
// no error came back: an authorization change that admitted the admin and then
// returned nothing would break the pending-cash-payments reconciliation this
// widening exists to enable, while passing a bare error-is-nil check.
func TestListRegistrationsForGame_AssignedAdminPrincipalReadsTheRoster(t *testing.T) {
	h, _, _ := newPrincipalTestHandler()
	game := seedHostedGame(t, h)

	if _, err := h.RegisterForGame(ctxAs(playerSubject), &socialplayv1.RegisterForGameRequest{
		GameId: game.GetId(),
	}); err != nil {
		t.Fatalf("seeding a registration should succeed: %v", err)
	}

	assignRosterAdmin(t, h, game.GetId(), adminSubject)

	resp, err := h.ListRegistrationsForGame(ctxAs(adminSubject), &socialplayv1.ListRegistrationsForGameRequest{
		GameId: game.GetId(),
	})
	if err != nil {
		t.Fatalf("ListRegistrationsForGame as an assigned Game Admin should succeed, got: %v", err)
	}

	regs := resp.GetRegistrations()
	if len(regs) != 1 {
		t.Fatalf("the assigned Game Admin read %d registrations, want 1", len(regs))
	}
	if got := regs[0].GetPlayerId(); got != playerSubject {
		t.Errorf("Registration.PlayerId = %q, want %q", got, playerSubject)
	}
}

// TestListRegistrationsForGame_UnentitledPrincipalsStillDenied is the negative
// half at the handler boundary: widening the entitled set must not have widened
// it past the store.
//
// Every row asserts PermissionDenied specifically, never Internal — a
// 500-shaped answer to an authorization question means the error escaped
// toStatus, which is its own bug class this package's tests have called out
// since T5.5.
func TestListRegistrationsForGame_UnentitledPrincipalsStillDenied(t *testing.T) {
	tests := []struct {
		name  string
		actor string
		// revokeFirst assigns the actor and then revokes them, so the row
		// proves the read resolves authority at call time.
		revokeFirst bool
	}{
		{name: "a stranger with no assignment", actor: attackerSubject},
		{name: "a registrant of this very Game", actor: playerSubject},
		{name: "an admin whose assignment was revoked", actor: "auth0|revoked-admin-1", revokeFirst: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, _, _ := newPrincipalTestHandler()
			game := seedHostedGame(t, h)

			// A registration must exist, so a passing test proves the roster
			// was *withheld* rather than merely empty.
			if _, err := h.RegisterForGame(ctxAs(playerSubject), &socialplayv1.RegisterForGameRequest{
				GameId: game.GetId(),
			}); err != nil {
				t.Fatalf("seeding a registration should succeed: %v", err)
			}

			// A real admin exists throughout, so these rows prove the caller
			// was refused rather than that the check simply denies everyone.
			assignRosterAdmin(t, h, game.GetId(), adminSubject)

			if tc.revokeFirst {
				assignRosterAdmin(t, h, game.GetId(), tc.actor)
				if _, err := h.RevokeGameAdmin(ctxAs(hostSubject), &socialplayv1.RevokeGameAdminRequest{
					GameId: game.GetId(),
					UserId: tc.actor,
				}); err != nil {
					t.Fatalf("the Host revoking %q: %v", tc.actor, err)
				}
			}

			resp, err := h.ListRegistrationsForGame(ctxAs(tc.actor), &socialplayv1.ListRegistrationsForGameRequest{
				GameId: game.GetId(),
			})
			requireCode(t, "ListRegistrationsForGame", err, codes.PermissionDenied)

			if len(resp.GetRegistrations()) != 0 {
				t.Errorf("a rejected roster read still returned %d registrations", len(resp.GetRegistrations()))
			}
		})
	}
}

// TestListRegistrationsForGame_AdminAuthorityIsPerGame pins CLAUDE.md's locked
// "per-game Game Admins" decision at the RPC boundary.
//
// A Game Admin of one Game is a legitimate admin holding a legitimate stored
// row — the strongest available shape of the wrong answer. If the resolution
// query ever lost its game scoping, every other test in this file would still
// pass and this one would fail, which is exactly why it is separate.
func TestListRegistrationsForGame_AdminAuthorityIsPerGame(t *testing.T) {
	h, _, _ := newPrincipalTestHandler()

	first := seedHostedGame(t, h)
	second := seedHostedGameWithCapacity(t, h, 6)
	if first.GetId() == second.GetId() {
		t.Fatalf("fixture precondition: the two Games must be distinct, both are %q", first.GetId())
	}

	if _, err := h.RegisterForGame(ctxAs(playerSubject), &socialplayv1.RegisterForGameRequest{
		GameId: first.GetId(),
	}); err != nil {
		t.Fatalf("seeding a registration on the first Game: %v", err)
	}

	// The admin is assigned to the SECOND Game only.
	assignRosterAdmin(t, h, second.GetId(), adminSubject)

	resp, err := h.ListRegistrationsForGame(ctxAs(adminSubject), &socialplayv1.ListRegistrationsForGameRequest{
		GameId: first.GetId(),
	})
	requireCode(t, "an admin of another Game reading this Game's roster", err, codes.PermissionDenied)
	if len(resp.GetRegistrations()) != 0 {
		t.Errorf("a rejected cross-Game roster read still returned %d registrations", len(resp.GetRegistrations()))
	}

	// And the same principal reading the Game they actually administer
	// succeeds — without which the row above would be satisfied by an admin
	// who is simply broken everywhere.
	if _, err := h.ListRegistrationsForGame(ctxAs(adminSubject), &socialplayv1.ListRegistrationsForGameRequest{
		GameId: second.GetId(),
	}); err != nil {
		t.Fatalf("the admin reading the Game they administer should succeed, got: %v", err)
	}
}

// TestListRegistrationsForGame_AnonymousIsStillUnauthenticated re-pins
// ADR-0013 §5 at this RPC after the widening. "I do not know who you are" must
// stay a different code from "I know who you are and you may not": the fix for
// the former is to authenticate, and PermissionDenied does not say so.
//
// roster_authz_test.go already covers this for the Host-only rule; it is
// repeated here because the check it guards is a different one now, and a
// widened check that resolved admins *before* demanding a principal could
// plausibly have started answering PermissionDenied to anonymous callers.
func TestListRegistrationsForGame_AnonymousIsStillUnauthenticated(t *testing.T) {
	h, _, _ := newPrincipalTestHandler()
	game := seedHostedGame(t, h)
	assignRosterAdmin(t, h, game.GetId(), adminSubject)

	_, err := h.ListRegistrationsForGame(anonymous(), &socialplayv1.ListRegistrationsForGameRequest{
		GameId: game.GetId(),
	})
	requireCode(t, "an anonymous roster read", err, codes.Unauthenticated)
}
