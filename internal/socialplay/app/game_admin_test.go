// T14.4 — app-layer tests for the durable Game-Admin store (#168, partial
// fix), against an in-memory port.GameAdminRepository.
//
// The assertion the ticket names as non-optional (instruction 5, §A13 GAP A) is
// TestGameAdminStore_ReadObservesWhatTheWriteWrote: a store nothing can query
// would reproduce T12's finding-1 shape — work that looks done and is not —
// one sprint after the dependency-completeness check that exists to prevent it.
package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nhuthuynh/white-label/internal/socialplay/app"
	"github.com/nhuthuynh/white-label/internal/socialplay/domain"
)

const (
	gaHost      = "ga-host-subject"
	gaAdmin     = "ga-admin-subject"
	gaSecond    = "ga-second-admin-subject"
	gaStranger  = "ga-stranger-subject"
	gaUnknownID = "00000000-0000-4000-d000-000000009999"
)

// gameAdminFixture seeds a Game hosted by gaHost into a fresh Service and
// returns both, so each test starts from an independent store.
func gameAdminFixture(t *testing.T) (*app.Service, *fakeGameAdminRepository, domain.Game) {
	t.Helper()

	games := newFakeGameRepository()
	admins := newFakeGameAdminRepository()
	svc := app.NewService(app.ServiceOptions{
		IDs:           &sequentialIDs{},
		Games:         games,
		Registrations: newFakeRegistrationRepository(),
		Waitlist:      newFakeWaitlistRepository(),
		Matches:       newFakeMatchRepository(),
		GameAdmins:    admins,
	})

	in := validInput(courtID(1))
	in.HostID = gaHost
	in.Range = mustRange(t, "2026-09-01T09:00:00Z", "2026-09-01T10:00:00Z")
	game, err := svc.ScheduleGame(context.Background(), in, newFakeReservation(), newFakeFacilityLookup())
	if err != nil {
		t.Fatalf("seeding fixture game: %v", err)
	}
	return svc, admins, game
}

// TestGameAdminStore_ReadObservesWhatTheWriteWrote is instruction 5's required
// proof: the store is queryable, and the query returns what the write put in —
// including the assigner and the Game scoping, not merely the user id.
func TestGameAdminStore_ReadObservesWhatTheWriteWrote(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svc, _, game := gameAdminFixture(t)

	before, err := svc.ListGameAdmins(ctx, game.ID)
	if err != nil {
		t.Fatalf("ListGameAdmins before assigning: %v", err)
	}
	if len(before) != 0 {
		t.Fatalf("a fresh Game reports %d admins, want 0", len(before))
	}

	assigned, err := svc.AssignGameAdmin(ctx, app.AssignGameAdminInput{
		GameID:      game.ID,
		ActorUserID: gaHost,
		AdminUserID: gaAdmin,
	})
	if err != nil {
		t.Fatalf("AssignGameAdmin: %v", err)
	}

	after, err := svc.ListGameAdmins(ctx, game.ID)
	if err != nil {
		t.Fatalf("ListGameAdmins after assigning: %v", err)
	}
	if len(after) != 1 {
		t.Fatalf("after one assignment the store reports %d admins, want 1 — a store nothing can read back is not a store", len(after))
	}

	got := after[0]
	if got.UserID != gaAdmin {
		t.Errorf("read back UserID %q, want %q", got.UserID, gaAdmin)
	}
	if got.GameID != game.ID {
		t.Errorf("read back GameID %q, want %q — an assignment that is not Game-scoped is not per-game", got.GameID, game.ID)
	}
	if got.AssignedBy != gaHost {
		t.Errorf("read back AssignedBy %q, want %q", got.AssignedBy, gaHost)
	}
	if got.AssignedAt.IsZero() {
		t.Error("read back a zero AssignedAt; the app layer supplies the clock")
	}
	if !got.AssignedAt.Equal(assigned.AssignedAt) {
		t.Errorf("read back AssignedAt %v, but the write returned %v", got.AssignedAt, assigned.AssignedAt)
	}

	// The resolution helper T14.5 consumes, driven off the same read.
	if !domain.HasGameAdmin(after, gaAdmin) {
		t.Error("HasGameAdmin does not resolve the user the Host just assigned — this is the query T14.5 consumes")
	}
	if domain.HasGameAdmin(after, gaStranger) {
		t.Error("HasGameAdmin resolved a stranger as an admin")
	}
}

// TestAssignGameAdmin_OnlyTheHostMayAssign is instruction 6's property at the
// app layer: the assigned admin is refused, so no chain of appointments can
// start from a delegation.
func TestAssignGameAdmin_OnlyTheHostMayAssign(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svc, _, game := gameAdminFixture(t)

	if _, err := svc.AssignGameAdmin(ctx, app.AssignGameAdminInput{
		GameID: game.ID, ActorUserID: gaHost, AdminUserID: gaAdmin,
	}); err != nil {
		t.Fatalf("seeding an admin: %v", err)
	}

	actors := []struct {
		name  string
		actor string
	}{
		{"an assigned game admin", gaAdmin},
		{"a stranger", gaStranger},
		{"an unidentified caller", ""},
	}

	for _, tt := range actors {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.AssignGameAdmin(ctx, app.AssignGameAdminInput{
				GameID: game.ID, ActorUserID: tt.actor, AdminUserID: gaSecond,
			})
			if !errors.Is(err, domain.ErrNotGameHost) {
				t.Fatalf("AssignGameAdmin as %s = %v, want ErrNotGameHost", tt.name, err)
			}

			admins, listErr := svc.ListGameAdmins(ctx, game.ID)
			if listErr != nil {
				t.Fatalf("ListGameAdmins: %v", listErr)
			}
			if domain.HasGameAdmin(admins, gaSecond) {
				t.Fatalf("%s was refused but the assignment persisted anyway — the rejection must leave no trace", tt.name)
			}
		})
	}
}

func TestRevokeGameAdmin(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("the host revokes an assignment", func(t *testing.T) {
		svc, _, game := gameAdminFixture(t)
		if _, err := svc.AssignGameAdmin(ctx, app.AssignGameAdminInput{
			GameID: game.ID, ActorUserID: gaHost, AdminUserID: gaAdmin,
		}); err != nil {
			t.Fatalf("seeding an admin: %v", err)
		}

		if err := svc.RevokeGameAdmin(ctx, app.RevokeGameAdminInput{
			GameID: game.ID, ActorUserID: gaHost, AdminUserID: gaAdmin,
		}); err != nil {
			t.Fatalf("RevokeGameAdmin: %v", err)
		}

		admins, err := svc.ListGameAdmins(ctx, game.ID)
		if err != nil {
			t.Fatalf("ListGameAdmins: %v", err)
		}
		if domain.HasGameAdmin(admins, gaAdmin) {
			t.Fatal("the revoked user still resolves as an admin — the read must observe the revoke as well as the assign")
		}
	})

	t.Run("an assigned admin cannot revoke", func(t *testing.T) {
		svc, _, game := gameAdminFixture(t)
		if _, err := svc.AssignGameAdmin(ctx, app.AssignGameAdminInput{
			GameID: game.ID, ActorUserID: gaHost, AdminUserID: gaAdmin,
		}); err != nil {
			t.Fatalf("seeding an admin: %v", err)
		}
		if _, err := svc.AssignGameAdmin(ctx, app.AssignGameAdminInput{
			GameID: game.ID, ActorUserID: gaHost, AdminUserID: gaSecond,
		}); err != nil {
			t.Fatalf("seeding a second admin: %v", err)
		}

		err := svc.RevokeGameAdmin(ctx, app.RevokeGameAdminInput{
			GameID: game.ID, ActorUserID: gaAdmin, AdminUserID: gaSecond,
		})
		if !errors.Is(err, domain.ErrNotGameHost) {
			t.Fatalf("RevokeGameAdmin as an assigned admin = %v, want ErrNotGameHost", err)
		}

		admins, listErr := svc.ListGameAdmins(ctx, game.ID)
		if listErr != nil {
			t.Fatalf("ListGameAdmins: %v", listErr)
		}
		if !domain.HasGameAdmin(admins, gaSecond) {
			t.Fatal("the refused revoke removed the assignment anyway")
		}
	})

	t.Run("revoking a user who holds no assignment", func(t *testing.T) {
		svc, _, game := gameAdminFixture(t)

		err := svc.RevokeGameAdmin(ctx, app.RevokeGameAdminInput{
			GameID: game.ID, ActorUserID: gaHost, AdminUserID: gaStranger,
		})
		if !errors.Is(err, domain.ErrGameAdminNotFound) {
			t.Fatalf("RevokeGameAdmin for an unassigned user = %v, want ErrGameAdminNotFound — "+
				"answering \"done\" would confirm a belief about who holds authority that the store does not support", err)
		}
	})
}

// TestGameAdminMethods_MalformedAndUnknownGameIDs pins the T10.7-shaped
// boundary guard these three new methods inherit: a malformed game id must be
// answered exactly as an unknown-but-well-formed one, and never reach the
// Postgres adapter's mustUUID (which panics, and grpc installs no recover()).
func TestGameAdminMethods_MalformedAndUnknownGameIDs(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	ids := []struct {
		name   string
		gameID string
	}{
		{"malformed", "not-a-uuid"},
		{"empty", ""},
		{"unknown but well formed", gaUnknownID},
	}

	for _, tt := range ids {
		t.Run(tt.name, func(t *testing.T) {
			svc, _, _ := gameAdminFixture(t)

			if _, err := svc.AssignGameAdmin(ctx, app.AssignGameAdminInput{
				GameID: tt.gameID, ActorUserID: gaHost, AdminUserID: gaAdmin,
			}); !errors.Is(err, domain.ErrGameNotFound) {
				t.Errorf("AssignGameAdmin with a %s game id = %v, want ErrGameNotFound", tt.name, err)
			}

			if err := svc.RevokeGameAdmin(ctx, app.RevokeGameAdminInput{
				GameID: tt.gameID, ActorUserID: gaHost, AdminUserID: gaAdmin,
			}); !errors.Is(err, domain.ErrGameNotFound) {
				t.Errorf("RevokeGameAdmin with a %s game id = %v, want ErrGameNotFound", tt.name, err)
			}

			if _, err := svc.ListGameAdmins(ctx, tt.gameID); !errors.Is(err, domain.ErrGameNotFound) {
				t.Errorf("ListGameAdmins with a %s game id = %v, want ErrGameNotFound", tt.name, err)
			}
		})
	}
}

// TestAssignGameAdmin_DuplicateIsRejected proves the pre-check reaches the
// caller as ErrAlreadyGameAdmin rather than as a repository error, and that a
// rejected duplicate does not disturb the assignment already in place.
func TestAssignGameAdmin_DuplicateIsRejected(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svc, _, game := gameAdminFixture(t)

	in := app.AssignGameAdminInput{GameID: game.ID, ActorUserID: gaHost, AdminUserID: gaAdmin}
	first, err := svc.AssignGameAdmin(ctx, in)
	if err != nil {
		t.Fatalf("first AssignGameAdmin: %v", err)
	}

	if _, err := svc.AssignGameAdmin(ctx, in); !errors.Is(err, domain.ErrAlreadyGameAdmin) {
		t.Fatalf("second AssignGameAdmin = %v, want ErrAlreadyGameAdmin", err)
	}

	admins, err := svc.ListGameAdmins(ctx, game.ID)
	if err != nil {
		t.Fatalf("ListGameAdmins: %v", err)
	}
	if len(admins) != 1 {
		t.Fatalf("the store holds %d assignments after a rejected duplicate, want 1", len(admins))
	}
	if !admins[0].AssignedAt.Equal(first.AssignedAt) {
		t.Error("the rejected duplicate overwrote the original assignment's AssignedAt")
	}
}

// TestAssignGameAdmin_RejectsTheHostAndBlankUsers keeps the two domain input
// rules reachable through the app layer, so a future refactor that skipped
// domain.AssignGameAdmin would fail here and not only in the domain package.
func TestAssignGameAdmin_RejectsTheHostAndBlankUsers(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := []struct {
		name        string
		adminUserID string
		wantErr     error
	}{
		{"the host themselves", gaHost, domain.ErrHostCannotBeGameAdmin},
		{"a blank user id", "", domain.ErrEmptyGameAdminUserID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _, game := gameAdminFixture(t)

			_, err := svc.AssignGameAdmin(ctx, app.AssignGameAdminInput{
				GameID: game.ID, ActorUserID: gaHost, AdminUserID: tt.adminUserID,
			})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("AssignGameAdmin with %s = %v, want %v", tt.name, err, tt.wantErr)
			}

			admins, listErr := svc.ListGameAdmins(ctx, game.ID)
			if listErr != nil {
				t.Fatalf("ListGameAdmins: %v", listErr)
			}
			if len(admins) != 0 {
				t.Fatalf("a rejected assignment persisted %d rows, want 0", len(admins))
			}
		})
	}
}

// TestListGameAdmins_IsScopedToOneGame is the negative control for the read:
// an assignment on one Game must not resolve on another, or "per-game Game
// Admins" (CLAUDE.md's locked decision) would be a platform-wide role.
func TestListGameAdmins_IsScopedToOneGame(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svc, _, gameA := gameAdminFixture(t)

	other := validInput(courtID(2))
	other.HostID = gaHost
	other.Range = mustRange(t, "2026-09-02T09:00:00Z", "2026-09-02T10:00:00Z")
	gameB, err := svc.ScheduleGame(ctx, other, newFakeReservation(), newFakeFacilityLookup())
	if err != nil {
		t.Fatalf("seeding a second game: %v", err)
	}

	if _, err := svc.AssignGameAdmin(ctx, app.AssignGameAdminInput{
		GameID: gameA.ID, ActorUserID: gaHost, AdminUserID: gaAdmin,
	}); err != nil {
		t.Fatalf("AssignGameAdmin on game A: %v", err)
	}

	adminsB, err := svc.ListGameAdmins(ctx, gameB.ID)
	if err != nil {
		t.Fatalf("ListGameAdmins on game B: %v", err)
	}
	if domain.HasGameAdmin(adminsB, gaAdmin) {
		t.Fatal("an admin assigned to game A resolves as an admin of game B — the assignment must be per-Game")
	}
}
