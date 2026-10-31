// T67.2 (part of issue #353) — ListRegistrationsForActor: a Player reads back
// the Games they joined.
//
// # Why this read has to exist, and why no existing one serves it
//
// Joining a Game works today, but the outcome lives in a Vue `ref` and nothing
// can recover it after a reload. The two reads that already exist cannot be
// pressed into service:
//
//   - ListRegistrationsForGame is Host-or-Game-Admin gated
//     (app.Service.ListRegistrationsForGame → domain.Game.EnsureHostOrGameAdmin).
//     A Player asking "which Games am I in?" is neither, and widening that check
//     would hand them every *other* registrant's player_id and payment_status —
//     the exact leak #147/T13.6 closed.
//   - ListGames filters `WHERE g.status = 'scheduled'`
//     (db/queries/socialplay.sql), and **there is no GetGame RPC at all**
//     (verified against the full rpc list in proto/, not assumed). So a Game the
//     Host cancelled is unreachable through every read on this service — which is
//     precisely the one fact a Player most needs to be told.
//
// That last point is why the Game facts are part of THIS read rather than left
// to a second call: there is no second call to make.
// TestListRegistrationsForActor_CancelledGameIsStillReturned is therefore the
// headline case, not an edge case.
package app_test

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/nhuthuynh/white-label/internal/socialplay/app"
	"github.com/nhuthuynh/white-label/internal/socialplay/domain"
)

// Actor ids here are uuid-shaped deliberately. registrations.player_id is a
// `uuid NOT NULL REFERENCES identity_users (id)` column as of
// db/migrations/0026_socialplay_identity_conformance.sql, and the Postgres
// adapter's mustUUID panics on anything else — so a fixture using
// "player-1"-style strings would be the exact fixture infidelity LESSONS.md's
// T9 entry names, passing here and panicking in production.
const (
	actorRegPlayer = "00000000-0000-4000-c000-000000000001"
	actorRegOther  = "00000000-0000-4000-c000-000000000002"
	actorRegHost   = "00000000-0000-4000-c000-000000000003"
)

// actorRegFixture wires a Service whose Registration repository can answer the
// joined read, and seeds Games through the REAL ScheduleGame/RegisterForGame
// write paths rather than by poking the fakes' maps. That matters for the same
// reason roster_admin_read_test.go spells out: a fixture that injected the
// answer would prove the method calls *something*, not that it reads what
// registration and Game writes actually left behind.
func actorRegFixture(t *testing.T) (*app.Service, *fakeGameRepository) {
	t.Helper()

	games := newFakeGameRepository()
	svc := app.NewService(app.ServiceOptions{
		Identity:      fakeIdentityLookup{},
		IDs:           &sequentialIDs{},
		Games:         games,
		Registrations: newFakeRegistrationRepositoryJoining(games),
		Waitlist:      newFakeWaitlistRepository(),
		Matches:       newFakeMatchRepository(),
		GameAdmins:    newFakeGameAdminRepository(),
	})
	return svc, games
}

// seedGame schedules one Game hosted by actorRegHost over the given window.
func seedGame(t *testing.T, svc *app.Service, start, end string) domain.Game {
	t.Helper()

	in := validInput(courtID(1))
	in.HostID = actorRegHost
	in.Range = mustRange(t, start, end)
	game, err := svc.ScheduleGame(context.Background(), in, newFakeReservation(), newFakeFacilityLookup())
	if err != nil {
		t.Fatalf("seeding a Game: %v", err)
	}
	return game
}

// join registers playerID for game through the real write path.
func join(t *testing.T, svc *app.Service, gameID, playerID string) domain.Registration {
	t.Helper()

	reg, err := svc.RegisterForGame(context.Background(), app.RegisterForGameInput{
		GameID:   gameID,
		PlayerID: playerID,
	})
	if err != nil {
		t.Fatalf("seeding a Registration for %q: %v", playerID, err)
	}
	return reg
}

// TestListRegistrationsForActor is the table-driven core: each case seeds a
// world, asks for actorRegPlayer's list, and states exactly which Games should
// come back.
func TestListRegistrationsForActor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		// seed builds the world and returns the Game IDs expected back, in
		// the order the read should produce them.
		seed func(t *testing.T, svc *app.Service, games *fakeGameRepository) []string
	}{
		{
			name: "the caller's own active registration comes back",
			seed: func(t *testing.T, svc *app.Service, _ *fakeGameRepository) []string {
				g := seedGame(t, svc, "2026-11-01T09:00:00Z", "2026-11-01T10:00:00Z")
				join(t, svc, g.ID, actorRegPlayer)
				return []string{g.ID}
			},
		},
		{
			name: "another player's registration is not the caller's",
			seed: func(t *testing.T, svc *app.Service, _ *fakeGameRepository) []string {
				g := seedGame(t, svc, "2026-11-02T09:00:00Z", "2026-11-02T10:00:00Z")
				join(t, svc, g.ID, actorRegOther)
				return nil
			},
		},
		{
			name: "a registration the caller cancelled leaves the list",
			seed: func(t *testing.T, svc *app.Service, _ *fakeGameRepository) []string {
				g := seedGame(t, svc, "2026-11-03T09:00:00Z", "2026-11-03T10:00:00Z")
				reg := join(t, svc, g.ID, actorRegPlayer)
				if _, err := svc.CancelRegistration(context.Background(), reg.ID, actorRegPlayer); err != nil {
					t.Fatalf("the caller cancelling their own Registration: %v", err)
				}
				return nil
			},
		},
		{
			name: "a player with no registrations gets an empty list, not an error",
			seed: func(t *testing.T, svc *app.Service, _ *fakeGameRepository) []string {
				seedGame(t, svc, "2026-11-04T09:00:00Z", "2026-11-04T10:00:00Z")
				return nil
			},
		},
		{
			name: "several registrations come back ordered by the Game's start",
			seed: func(t *testing.T, svc *app.Service, _ *fakeGameRepository) []string {
				later := seedGame(t, svc, "2026-11-20T09:00:00Z", "2026-11-20T10:00:00Z")
				sooner := seedGame(t, svc, "2026-11-06T09:00:00Z", "2026-11-06T10:00:00Z")
				join(t, svc, later.ID, actorRegPlayer)
				join(t, svc, sooner.ID, actorRegPlayer)
				return []string{sooner.ID, later.ID}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			svc, games := actorRegFixture(t)
			want := tc.seed(t, svc, games)

			got, err := svc.ListRegistrationsForActor(ctx, actorRegPlayer)
			if err != nil {
				t.Fatalf("ListRegistrationsForActor: %v", err)
			}
			if got == nil {
				t.Fatalf("ListRegistrationsForActor returned a nil slice; " +
					"an empty result must be an empty list, never nil")
			}
			if len(got) != len(want) {
				t.Fatalf("ListRegistrationsForActor returned %d listings, want %d (%v)",
					len(got), len(want), want)
			}
			for i, listing := range got {
				if listing.Registration.GameID != want[i] {
					t.Errorf("listing[%d].Registration.GameID = %q, want %q",
						i, listing.Registration.GameID, want[i])
				}
				if listing.Registration.PlayerID != actorRegPlayer {
					t.Errorf("listing[%d].Registration.PlayerID = %q, want the caller %q",
						i, listing.Registration.PlayerID, actorRegPlayer)
				}
			}
		})
	}
}

// TestListRegistrationsForActor_CancelledGameIsStillReturned is this ticket's
// headline requirement and the whole reason the Game fields are in this read.
//
// ListGames filters `WHERE g.status = 'scheduled'` and there is no GetGame RPC,
// so if this read dropped the cancelled Game — or returned the row without its
// Game status — "the Host cancelled your game" would be unreportable to the one
// person it concerns. Note what the Player's own Registration still says:
// app.Service.CancelGame cascades a bulk cancel over the Game's Registrations
// (port.RegistrationRepository.CancelAllActiveForGame, T16.3), so the row is
// cancelled too. The assertion below is therefore specifically that a Game
// cancelled by its HOST still surfaces — the Registration's own status is no
// longer the discriminator, and the Game status is.
func TestListRegistrationsForActor_CancelledGameIsStillReturned(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svc, _ := actorRegFixture(t)

	game := seedGame(t, svc, "2026-11-10T09:00:00Z", "2026-11-10T10:00:00Z")
	join(t, svc, game.ID, actorRegPlayer)

	if _, err := svc.CancelGame(ctx, game.ID, actorRegHost,
		newFakeReservation(), &fakeRefunder{}); err != nil {
		t.Fatalf("the Host cancelling the Game: %v", err)
	}

	got, err := svc.ListRegistrationsForActor(ctx, actorRegPlayer)
	if err != nil {
		t.Fatalf("ListRegistrationsForActor after the Host cancelled: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("the caller read %d listings after the Host cancelled their Game, want 1 — "+
			"a cancelled Game is unreachable through ListGames and there is no GetGame RPC, "+
			"so dropping it here makes the cancellation unreportable", len(got))
	}
	if got[0].GameStatus != domain.StatusCancelled {
		t.Errorf("listing.GameStatus = %q, want %q", got[0].GameStatus, domain.StatusCancelled)
	}
	if got[0].Registration.GameID != game.ID {
		t.Errorf("listing.Registration.GameID = %q, want %q", got[0].Registration.GameID, game.ID)
	}
}

// TestListRegistrationsForActor_CarriesTheGameFactsTheScreenNeeds pins the
// projection itself. A read that returned the Registration alone would compile,
// pass every scoping assertion above, and still leave the screen unable to say
// when or where the Game is — so each field is asserted against the Game that
// was actually scheduled, not merely checked for non-emptiness.
func TestListRegistrationsForActor_CarriesTheGameFactsTheScreenNeeds(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svc, _ := actorRegFixture(t)

	in := validInput(courtID(1), courtID(2))
	in.HostID = actorRegHost
	in.FacilityID = "facility-1"
	in.Range = mustRange(t, "2026-11-15T09:00:00Z", "2026-11-15T11:00:00Z")
	game, err := svc.ScheduleGame(ctx, in, newFakeReservation(), newFakeFacilityLookup())
	if err != nil {
		t.Fatalf("seeding a Game: %v", err)
	}
	join(t, svc, game.ID, actorRegPlayer)

	got, err := svc.ListRegistrationsForActor(ctx, actorRegPlayer)
	if err != nil {
		t.Fatalf("ListRegistrationsForActor: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d listings, want 1", len(got))
	}
	l := got[0]

	if !l.GameStartsAt.Equal(game.Range.Start) {
		t.Errorf("GameStartsAt = %v, want %v", l.GameStartsAt, game.Range.Start)
	}
	if !l.GameEndsAt.Equal(game.Range.End) {
		t.Errorf("GameEndsAt = %v, want %v", l.GameEndsAt, game.Range.End)
	}
	if l.GameVenueFacilityID != game.VenueFacilityID {
		t.Errorf("GameVenueFacilityID = %q, want %q", l.GameVenueFacilityID, game.VenueFacilityID)
	}
	if l.GameStatus != domain.StatusScheduled {
		t.Errorf("GameStatus = %q, want %q", l.GameStatus, domain.StatusScheduled)
	}
	if len(l.GameCourtIDs) != len(game.CourtIDs) {
		t.Fatalf("GameCourtIDs = %v, want %v", l.GameCourtIDs, game.CourtIDs)
	}
	for i, id := range game.CourtIDs {
		if l.GameCourtIDs[i] != id {
			t.Errorf("GameCourtIDs[%d] = %q, want %q", i, l.GameCourtIDs[i], id)
		}
	}

	// The Registration half of the projection, which the screen needs
	// alongside the Game facts (status, payment_status, guest_count,
	// amount_owed).
	if l.Registration.Status != domain.RegistrationStatusRegistered {
		t.Errorf("Registration.Status = %q, want %q",
			l.Registration.Status, domain.RegistrationStatusRegistered)
	}
	if l.Registration.PaymentStatus != domain.PaymentStatusUnpaid {
		t.Errorf("Registration.PaymentStatus = %q, want %q",
			l.Registration.PaymentStatus, domain.PaymentStatusUnpaid)
	}
}

// TestListRegistrationsForActor_MalformedActorNeverReachesTheRepository is the
// T10.7-shaped guard every other actor-scoped read in this codebase carries
// (see booking's app.Service.ListRecurringHireTemplatesForActor). It is not
// cosmetic: registrations.player_id is a `uuid` column, and the Postgres
// adapter's mustUUID PANICS on anything pgtype.UUID.Scan cannot parse. The
// assertion is on the call count, not merely on the empty answer — a guard
// that returned the right value after reaching the repository would not have
// prevented the panic.
func TestListRegistrationsForActor_MalformedActorNeverReachesTheRepository(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	games := newFakeGameRepository()
	regs := newFakeRegistrationRepositoryJoining(games)
	svc := app.NewService(app.ServiceOptions{
		Identity:      fakeIdentityLookup{},
		IDs:           &sequentialIDs{},
		Games:         games,
		Registrations: regs,
		Waitlist:      newFakeWaitlistRepository(),
		Matches:       newFakeMatchRepository(),
		GameAdmins:    newFakeGameAdminRepository(),
	})

	for _, malformed := range []string{"", "not-a-uuid", "auth0|subject", "   "} {
		got, err := svc.ListRegistrationsForActor(ctx, malformed)
		if err != nil {
			t.Fatalf("ListRegistrationsForActor(%q) error = %v, want nil — "+
				"a malformed actor is answered exactly like an unknown one", malformed, err)
		}
		if len(got) != 0 {
			t.Errorf("ListRegistrationsForActor(%q) = %+v, want an empty list", malformed, got)
		}
	}

	if n := regs.listForPlayerCalls.Load(); n != 0 {
		t.Errorf("the repository's ListForPlayer was called %d times for malformed ids, want 0 — "+
			"the Postgres adapter's mustUUID panics on a non-uuid player_id", n)
	}
}

// TestListRegistrationsForActor_IdentityPortIsNeverConsulted pins the absence
// of a role check, the same way booking's actor-scoped read does. The actor's
// identity IS the scope here: the read filters on player_id = actor, so there
// is no other party's data in reach for an ownership or role check to protect.
// Requiring a role would remove no exposure and would cost a Player who lost
// one the ability to read back Games they already joined.
func TestListRegistrationsForActor_IdentityPortIsNeverConsulted(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	games := newFakeGameRepository()
	counting := &countingIdentityLookup{}
	svc := app.NewService(app.ServiceOptions{
		Identity:      counting,
		IDs:           &sequentialIDs{},
		Games:         games,
		Registrations: newFakeRegistrationRepositoryJoining(games),
		Waitlist:      newFakeWaitlistRepository(),
		Matches:       newFakeMatchRepository(),
		GameAdmins:    newFakeGameAdminRepository(),
	})

	if _, err := svc.ListRegistrationsForActor(ctx, actorRegPlayer); err != nil {
		t.Fatalf("ListRegistrationsForActor: %v", err)
	}
	if n := counting.calls.Load(); n != 0 {
		t.Errorf("port.IdentityLookup was consulted %d times, want 0 — "+
			"this read deliberately applies no role check", n)
	}
}

// countingIdentityLookup is a port.IdentityLookup that records whether it was
// consulted at all. It exists so the absence of a role/identity check on this
// read is PINNED rather than incidental — mirroring the equivalent assertion
// booking's actor-scoped read carries.
type countingIdentityLookup struct {
	calls atomic.Int64
}

func (c *countingIdentityLookup) UserIDBySubject(_ context.Context, subject string) (string, error) {
	c.calls.Add(1)
	if subject == "" {
		return "", domain.ErrUserNotFound
	}
	return subject, nil
}

// newFakeRegistrationRepositoryJoining returns a fakeRegistrationRepository
// wired to read Games out of the given fakeGameRepository, so its ListForPlayer
// can model the real query's single JOIN rather than inventing the Game facts.
//
// The wiring is explicit (rather than a field every caller must remember to
// set) because an unwired fake would answer ListForPlayer with an empty list
// and look like a correctly-scoped read returning nothing — the failure mode
// that is indistinguishable from success.
func newFakeRegistrationRepositoryJoining(games *fakeGameRepository) *fakeRegistrationRepository {
	r := newFakeRegistrationRepository()
	r.gameSource = games
	return r
}
