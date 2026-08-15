// T13.6 — the roster read stopped being public (partial fix for #147).
//
// **Superseded in part by T14.5**, which widened the entitled set from
// Host-only to Host-or-assigned-Game-Admin once T14.4's durable store made
// "assigned Game Admin" a server fact. The cases below are unchanged and still
// hold — the Host reads, a stranger and a registrant are refused, an anonymous
// caller is Unauthenticated, an unknown Game is an empty roster. What is no
// longer true is this file's original claim that the *admin* is refused too;
// roster_admin_authz_test.go carries that half now. The two files are kept
// separate rather than merged so the T13.6 proof stays legible as the thing it
// was: the fix for the leak itself, which is independent of who the entitled
// set eventually grew to include.
//
// Before this ticket ListRegistrationsForGame had no authorization check of
// any kind: it was in PublicMethods(), took no actor, and returned every
// registrant's player_id, payment_status and guest_count to anyone holding a
// game_id. game_id is readable straight off the *public* ListGames response,
// so the ids were never secret and the leak was enumerable — #147's exact
// complaint.
//
// This file is the four-case proof #147 asks for, in the shape
// principal_authz_test.go established for the write RPCs:
//
//	(a) the Game's Host                       -> succeeds, sees the roster
//	(b) a verified caller who is not the Host -> PermissionDenied
//	(c) no principal at all                   -> Unauthenticated
//	(d) an unknown or malformed game_id       -> empty roster, unchanged
//
// (b) and (c) must stay distinct codes — "I know who you are and you may not"
// versus "I do not know who you are" (ADR-0013 §5). (b) is asserted as
// PermissionDenied specifically, never Internal: a 500-shaped answer to an
// authorization question means the error escaped toStatus.
//
// # The registrant case is a deliberate, disclosed limitation
//
// TestListRegistrationsForGame_NonHostPrincipalIsPermissionDenied covers a
// player who is *in* the Game and is still refused. #147 explicitly leaves
// "should a registrant see the roster of a Game they are in?" open as a
// product question, and no ticket since has answered it — T14.5 widened the
// entitled set to assigned Game Admins and deliberately did not touch the
// registrant case, because that would be a product decision smuggled in as an
// authorization change. Asserted here rather than left untested so the
// remaining narrowness is a checked fact, not an accident.
//
// The T13.6-era reasoning for why *admins* were refused here — the assigned
// list was caller-supplied and persisted nowhere (#168), so honouring it would
// have let any caller name themselves — is now history rather than policy. See
// roster_admin_authz_test.go.
package grpcapi_test

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"

	socialplayv1 "github.com/nhuthuynh/white-label/internal/gen/pickleball/socialplay/v1"
)

// --- (a) the Host reads the roster ---------------------------------------

func TestListRegistrationsForGame_HostPrincipalReadsTheRoster(t *testing.T) {
	h, _, _ := newPrincipalTestHandler()
	game := seedHostedGame(t, h)

	if _, err := h.RegisterForGame(ctxAs(playerSubject), &socialplayv1.RegisterForGameRequest{
		GameId: game.GetId(),
	}); err != nil {
		t.Fatalf("seeding a registration should succeed: %v", err)
	}

	resp, err := h.ListRegistrationsForGame(ctxAs(hostSubject), &socialplayv1.ListRegistrationsForGameRequest{
		GameId: game.GetId(),
	})
	if err != nil {
		t.Fatalf("ListRegistrationsForGame as the Game's Host should succeed, got: %v", err)
	}

	// The roster must actually arrive. A check that authorized correctly but
	// returned nothing would pass a bare error-is-nil assertion while breaking
	// the Host pending-cash-payments dashboard this RPC exists to serve
	// (T8.10).
	regs := resp.GetRegistrations()
	if len(regs) != 1 {
		t.Fatalf("Host read %d registrations, want 1 — the Host's own dashboard read is broken", len(regs))
	}
	if got := regs[0].GetPlayerId(); got != playerSubject {
		t.Errorf("Registration.PlayerId = %q, want %q", got, playerSubject)
	}
}

// --- (b) and (c) the caller who is not the Host --------------------------

// TestListRegistrationsForGame_NonHostPrincipalIsPermissionDenied is the
// regression for #147 itself. Before T13.6 every row of this table returned
// OK plus the full roster.
func TestListRegistrationsForGame_NonHostPrincipalIsPermissionDenied(t *testing.T) {
	tests := []struct {
		name string
		ctx  func() context.Context
		want codes.Code
	}{
		{
			// The enumerating stranger #147 describes: a verified account,
			// a game_id read off the public ListGames response.
			name: "a verified caller who is not the Host",
			ctx:  func() context.Context { return ctxAs(attackerSubject) },
			want: codes.PermissionDenied,
		},
		{
			// Registered *in* the Game and still refused — the disclosed
			// Host-only narrowness, asserted rather than assumed. See this
			// file's header.
			name: "a registrant of this very Game",
			ctx:  func() context.Context { return ctxAs(playerSubject) },
			want: codes.PermissionDenied,
		},
		{
			// Distinct code from the two above, per ADR-0013 §5: the fix for
			// an anonymous caller is to authenticate, and PermissionDenied
			// does not say so.
			name: "no principal at all",
			ctx:  anonymous,
			want: codes.Unauthenticated,
		},
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

			resp, err := h.ListRegistrationsForGame(tc.ctx(), &socialplayv1.ListRegistrationsForGameRequest{
				GameId: game.GetId(),
			})
			requireCode(t, "ListRegistrationsForGame", err, tc.want)

			// Belt and braces: a handler that returned both an error and a
			// populated body would still leak through a gateway that logged
			// or forwarded the message.
			if len(resp.GetRegistrations()) != 0 {
				t.Errorf("a rejected roster read still returned %d registrations", len(resp.GetRegistrations()))
			}
		})
	}
}

// --- (d) the unknown-Game answer is unchanged ----------------------------

// TestListRegistrationsForGame_UnknownGameIsStillAnEmptyRoster pins the one
// behaviour T13.6 deliberately does NOT change.
//
// app.Service.ListRegistrationsForGame has always answered an unknown or
// malformed game_id with an empty roster rather than NotFound (the invariant
// malformed_id_test.go exists to protect: "malformed is indistinguishable
// from unknown"). Adding the Host check means loading the Game, which puts a
// tempting ErrGameNotFound on the path. Propagating it would be a second,
// unrelated behaviour change smuggled into an authorization ticket — and it
// buys nothing, because a Game that does not exist has no roster to leak and
// Game existence is public via ListGames regardless.
func TestListRegistrationsForGame_UnknownGameIsStillAnEmptyRoster(t *testing.T) {
	for _, gameID := range []string{
		"11111111-2222-3333-4444-555555555555", // well-formed, no such Game
		"not-a-uuid",                           // malformed
		"",                                     // absent
	} {
		t.Run(gameID, func(t *testing.T) {
			h, _, _ := newPrincipalTestHandler()

			resp, err := h.ListRegistrationsForGame(ctxAs(attackerSubject), &socialplayv1.ListRegistrationsForGameRequest{
				GameId: gameID,
			})
			if err != nil {
				t.Fatalf("ListRegistrationsForGame(%q) error = %v, want an empty roster", gameID, err)
			}
			if len(resp.GetRegistrations()) != 0 {
				t.Errorf("ListRegistrationsForGame(%q) returned %d registrations, want 0", gameID, len(resp.GetRegistrations()))
			}
		})
	}
}
