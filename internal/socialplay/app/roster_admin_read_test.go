// T14.5 — the roster read is Host **or** assigned Game Admin (partial fix for
// #147, and the consumption half of #168).
//
// # What changed, and why it could not have changed earlier
//
// T13.6 shipped ListRegistrationsForGame as Host-only and said so in its own
// doc comment: an assigned Game Admin *should* be able to read a roster, but
// the only expression of "assigned Game Admin" this codebase had was
// RecordMatchResultRequest.assigned_game_admin_user_ids — a repeated string the
// caller supplied. A rule honouring it would have admitted any caller willing
// to type their own id, which is strictly worse than Host-only because it reads
// as authorization while enforcing nothing. T14.4 built the durable store
// (domain.GameAdmin, port.GameAdminRepository, the game_admins table), so the
// entitled set is now a server fact and the rule can finally widen.
//
// # Every test in this file drives the REAL store
//
// There is no "pretend this user is an admin" seam anywhere below. The only way
// a subject becomes an admin here is Service.AssignGameAdmin — the real
// Host-only write path, through the real domain rule — and the only way the
// read learns about it is Service.ListRegistrationsForGame resolving the set
// through port.GameAdminRepository.ListGameAdmins. A test that stubbed the
// membership answer would prove the widened check calls *something*, not that
// it calls the store; that distinction is the entire content of this ticket.
package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nhuthuynh/white-label/internal/socialplay/app"
	"github.com/nhuthuynh/white-label/internal/socialplay/domain"
)

const (
	rosterHost       = "roster-host-subject"
	rosterAdmin      = "roster-admin-subject"
	rosterRegistrant = "roster-registrant-subject"
	rosterStranger   = "roster-stranger-subject"
)

// rosterFixture seeds a Game hosted by rosterHost carrying one active
// Registration, and returns the Service alongside it. The registration matters:
// without it every "the roster was withheld" assertion would also pass against
// a check that authorized correctly and returned nothing.
func rosterFixture(t *testing.T) (*app.Service, domain.Game) {
	t.Helper()

	svc := app.NewService(app.ServiceOptions{
		IDs:           &sequentialIDs{},
		Games:         newFakeGameRepository(),
		Registrations: newFakeRegistrationRepository(),
		Waitlist:      newFakeWaitlistRepository(),
		Matches:       newFakeMatchRepository(),
		GameAdmins:    newFakeGameAdminRepository(),
	})

	in := validInput(courtID(1))
	in.HostID = rosterHost
	in.Range = mustRange(t, "2026-10-01T09:00:00Z", "2026-10-01T10:00:00Z")
	game, err := svc.ScheduleGame(context.Background(), in, newFakeReservation(), newFakeFacilityLookup())
	if err != nil {
		t.Fatalf("seeding the fixture Game: %v", err)
	}

	if _, err := svc.RegisterForGame(context.Background(), app.RegisterForGameInput{
		GameID:   game.ID,
		PlayerID: rosterRegistrant,
	}); err != nil {
		t.Fatalf("seeding a registration: %v", err)
	}

	return svc, game
}

// assignAdmin grants adminUserID Game-Admin authority over game through the
// real Host-only write path.
func assignAdmin(t *testing.T, svc *app.Service, game domain.Game, adminUserID string) {
	t.Helper()

	if _, err := svc.AssignGameAdmin(context.Background(), app.AssignGameAdminInput{
		GameID:      game.ID,
		ActorUserID: rosterHost,
		AdminUserID: adminUserID,
	}); err != nil {
		t.Fatalf("the Host assigning %q as a Game Admin: %v", adminUserID, err)
	}
}

// TestListRegistrationsForGame_AssignedGameAdminReadsTheRoster is this
// ticket's headline assertion, and the one that fails red against T13.6's
// Host-only check. It is deliberately not satisfied by a bare error-is-nil
// assertion: the roster has to actually arrive, because a widened check that
// authorized and then returned an empty list would break the very dashboard the
// widening exists to serve.
func TestListRegistrationsForGame_AssignedGameAdminReadsTheRoster(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svc, game := rosterFixture(t)
	assignAdmin(t, svc, game, rosterAdmin)

	got, err := svc.ListRegistrationsForGame(ctx, game.ID, rosterAdmin)
	if err != nil {
		t.Fatalf("an assigned Game Admin reading the roster: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("the assigned Game Admin read %d registrations, want 1", len(got))
	}
	if got[0].PlayerID != rosterRegistrant {
		t.Errorf("Registration.PlayerID = %q, want %q", got[0].PlayerID, rosterRegistrant)
	}
}

// TestListRegistrationsForGame_HostStillReadsTheRoster is the too-strict guard.
// Widening a check is exactly as capable of breaking the previously-entitled
// actor as of admitting the new one, and the Host's own dashboard read (T8.10)
// is what this RPC was built for.
func TestListRegistrationsForGame_HostStillReadsTheRoster(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svc, game := rosterFixture(t)
	assignAdmin(t, svc, game, rosterAdmin)

	got, err := svc.ListRegistrationsForGame(ctx, game.ID, rosterHost)
	if err != nil {
		t.Fatalf("the Game's own Host reading the roster: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("the Host read %d registrations, want 1", len(got))
	}
}

// TestListRegistrationsForGame_UnentitledActorsAreStillRefused is the negative
// half, and the reason the widening is not a regression of #147 itself.
//
// The rows are chosen so that each one fails a *different* way to be an admin.
// "Named themselves" is the row that matters most: it is the caller-supplied
// list T13.6 refused to honour, re-expressed as the closest thing the new API
// still permits — an actor id the caller controls, with no store row behind it.
func TestListRegistrationsForGame_UnentitledActorsAreStillRefused(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		actor string
		// setup runs against the fixture before the read, so each row can put
		// the store into the state its case is about.
		setup func(t *testing.T, svc *app.Service, game domain.Game)
	}{
		{
			// #147's enumerating stranger: a game_id off the public ListGames
			// response and nothing else.
			name:  "a stranger with no assignment",
			actor: rosterStranger,
		},
		{
			// The disclosed product limitation T13.6 recorded, unchanged by this
			// ticket: #147 leaves "may a registrant see the roster of a Game
			// they are in" open, so until it is answered they may not.
			name:  "a registrant of this very Game",
			actor: rosterRegistrant,
		},
		{
			// The caller-supplied-list attack, in the only form the widened API
			// leaves available. There is no wire field to forge any more, so
			// this row asserts the property that replaced it: an actor id with
			// no row in the store buys nothing at all.
			name:  "a caller who named themselves an admin but has no stored assignment",
			actor: "roster-self-appointed-subject",
		},
		{
			// Authority is per-Game by construction (CLAUDE.md's locked
			// "per-game Game Admins"). An admin of some *other* Game is a real
			// admin holding a real row — just not one scoped here.
			name:  "an admin of a different Game",
			actor: "roster-other-game-admin-subject",
			setup: func(t *testing.T, svc *app.Service, _ domain.Game) {
				t.Helper()
				other := validInput(courtID(2))
				other.HostID = rosterHost
				other.Range = mustRange(t, "2026-10-02T09:00:00Z", "2026-10-02T10:00:00Z")
				otherGame, err := svc.ScheduleGame(context.Background(), other, newFakeReservation(), newFakeFacilityLookup())
				if err != nil {
					t.Fatalf("seeding the second Game: %v", err)
				}
				assignAdmin(t, svc, otherGame, "roster-other-game-admin-subject")
			},
		},
		{
			// A revoked admin is refused, which is what makes the read resolve
			// the set at call time rather than caching an answer. Without this
			// row a Host could withdraw authority and the reader would keep it.
			name:  "an admin whose assignment was revoked",
			actor: "roster-revoked-admin-subject",
			setup: func(t *testing.T, svc *app.Service, game domain.Game) {
				t.Helper()
				assignAdmin(t, svc, game, "roster-revoked-admin-subject")
				if err := svc.RevokeGameAdmin(context.Background(), app.RevokeGameAdminInput{
					GameID:      game.ID,
					ActorUserID: rosterHost,
					AdminUserID: "roster-revoked-admin-subject",
				}); err != nil {
					t.Fatalf("the Host revoking: %v", err)
				}
			},
		},
		{
			// An unidentified caller is never the Host and never an admin, even
			// against a store that somehow held a blank row — domain.
			// HasGameAdmin's blank guard, reached from here.
			name:  "no actor at all",
			actor: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			svc, game := rosterFixture(t)
			assignAdmin(t, svc, game, rosterAdmin)
			if tc.setup != nil {
				tc.setup(t, svc, game)
			}

			got, err := svc.ListRegistrationsForGame(ctx, game.ID, tc.actor)
			if !errors.Is(err, domain.ErrNotGameHostOrAdmin) {
				t.Fatalf("ListRegistrationsForGame as %s: err = %v, want %v", tc.name, err, domain.ErrNotGameHostOrAdmin)
			}
			if len(got) != 0 {
				t.Errorf("a refused roster read still returned %d registrations", len(got))
			}
		})
	}
}

// TestListRegistrationsForGame_UnknownGameIsStillAnEmptyRoster pins the one
// behaviour this ticket must NOT change while adding a second repository read
// to the path.
//
// "Malformed is indistinguishable from unknown" is malformed_id_test.go's
// invariant, and the widened check now reaches port.GameAdminRepository as well
// as GameRepository — a naive implementation that consulted the admin store
// before establishing the Game exists could easily turn either case into an
// error. A Game that does not exist has no roster to withhold.
func TestListRegistrationsForGame_UnknownGameIsStillAnEmptyRoster(t *testing.T) {
	t.Parallel()

	for _, gameID := range []string{
		"11111111-2222-3333-4444-555555555555", // well-formed, no such Game
		"not-a-uuid",                           // malformed
		"",                                     // absent
	} {
		t.Run(gameID, func(t *testing.T) {
			t.Parallel()

			svc, _ := rosterFixture(t)

			got, err := svc.ListRegistrationsForGame(context.Background(), gameID, rosterStranger)
			if err != nil {
				t.Fatalf("ListRegistrationsForGame(%q) = %v, want an empty roster", gameID, err)
			}
			if len(got) != 0 {
				t.Errorf("ListRegistrationsForGame(%q) returned %d registrations, want 0", gameID, len(got))
			}
		})
	}
}
