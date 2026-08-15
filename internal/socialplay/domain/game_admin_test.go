// T14.4 — the pure rules behind a durable Game-Admin assignment (#168,
// partial fix). Table-driven, dependency-free: nothing here touches a
// repository, a database or a wire type, which is the whole point of putting
// the rules in domain rather than in app.Service.
//
// The rule this file exists to pin hardest is instruction 6's: **an admin must
// not be able to appoint an admin.** #168 states why in one line — "or the
// Host-only distinction ErrNotGameHost exists to protect is worthless" — and
// TestAssignGameAdmin_AnAdminCannotAppointAnAdmin below is the assertion that
// fails if a future change ever routes assignment through
// EnsureHostOrGameAdmin.
package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/nhuthuynh/white-label/internal/socialplay/domain"
)

const (
	adminHostID    = "host-subject"
	adminUserID    = "admin-subject"
	adminStranger  = "stranger-subject"
	adminSecondary = "second-admin-subject"
)

var adminAssignedAt = time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)

// adminFixtureGame builds a scheduled Game hosted by adminHostID. It goes
// through domain.NewGame rather than a struct literal so the fixture can never
// drift from a Game the constructor would actually accept.
func adminFixtureGame(t *testing.T) domain.Game {
	t.Helper()

	start := time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC)
	rng, err := domain.NewTimeRange(start, start.Add(time.Hour))
	if err != nil {
		t.Fatalf("bad fixture range: %v", err)
	}
	g, err := domain.NewGame(
		"11111111-1111-4111-8111-111111111111",
		adminHostID,
		"facility-1",
		"",
		[]string{"22222222-2222-4222-8222-222222222222"},
		rng,
		4,
		domain.PaymentMethodEither,
		0,
		domain.Money{},
	)
	if err != nil {
		t.Fatalf("bad fixture game: %v", err)
	}
	return g
}

func TestAssignGameAdmin(t *testing.T) {
	t.Parallel()

	game := adminFixtureGame(t)
	alreadyAssigned := []domain.GameAdmin{{
		GameID:     game.ID,
		UserID:     adminUserID,
		AssignedBy: adminHostID,
		AssignedAt: adminAssignedAt,
	}}

	tests := []struct {
		name        string
		existing    []domain.GameAdmin
		actorUserID string
		adminUserID string
		wantErr     error
		why         string
	}{
		{
			name:        "the host assigns a new admin",
			actorUserID: adminHostID,
			adminUserID: adminUserID,
			why:         "the whole point of the aggregate: the Host, and only the Host, delegates",
		},
		{
			name:        "the host assigns a second admin",
			existing:    alreadyAssigned,
			actorUserID: adminHostID,
			adminUserID: adminSecondary,
			why:         "an existing assignment does not close the Game to further ones",
		},
		{
			name:        "an assigned admin tries to appoint another admin",
			existing:    alreadyAssigned,
			actorUserID: adminUserID,
			adminUserID: adminSecondary,
			wantErr:     domain.ErrNotGameHost,
			why: "#168's own words: an admin must not appoint an admin, " +
				"or the Host-only distinction ErrNotGameHost exists to protect is worthless",
		},
		{
			name:        "a stranger tries to assign",
			actorUserID: adminStranger,
			adminUserID: adminSecondary,
			wantErr:     domain.ErrNotGameHost,
		},
		{
			name:        "an unidentified caller tries to assign",
			actorUserID: "",
			adminUserID: adminUserID,
			wantErr:     domain.ErrNotGameHost,
			why:         "EnsureHost rejects an empty actor even against a Game with an empty HostID",
		},
		{
			name:        "the host assigns an empty user id",
			actorUserID: adminHostID,
			adminUserID: "",
			wantErr:     domain.ErrEmptyGameAdminUserID,
			why:         "a blank admin row would match a blank actor at read time — the exact accident EnsureHostOrGameAdmin's blank-entry guard exists to prevent",
		},
		{
			name:        "the host assigns themselves",
			actorUserID: adminHostID,
			adminUserID: adminHostID,
			wantErr:     domain.ErrHostCannotBeGameAdmin,
			why:         "the Host is already entitled; a row implying delegation to themselves would make a later revoke look like it removed Host authority, which it cannot",
		},
		{
			name:        "the host assigns someone already assigned",
			existing:    alreadyAssigned,
			actorUserID: adminHostID,
			adminUserID: adminUserID,
			wantErr:     domain.ErrAlreadyGameAdmin,
			why:         "mirrors domain.Register's ErrAlreadyRegistered pre-check; the composite primary key is the authoritative guard (CLAUDE.md rule 4)",
		},
		{
			name:        "authorization is checked before the input",
			existing:    alreadyAssigned,
			actorUserID: adminStranger,
			adminUserID: "",
			wantErr:     domain.ErrNotGameHost,
			why: "an unauthorized actor must not learn whether the input would otherwise have been valid — " +
				"the ordering internal/payments/app.authorizeOfflineRecording established",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := domain.AssignGameAdmin(game, tt.existing, tt.actorUserID, tt.adminUserID, adminAssignedAt)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("AssignGameAdmin() error = %v, want %v — %s", err, tt.wantErr, tt.why)
				}
				if got != (domain.GameAdmin{}) {
					t.Errorf("AssignGameAdmin() returned %+v alongside an error, want the zero value", got)
				}
				return
			}

			if err != nil {
				t.Fatalf("AssignGameAdmin() unexpected error: %v — %s", err, tt.why)
			}
			if got.GameID != game.ID {
				t.Errorf("GameID = %q, want %q", got.GameID, game.ID)
			}
			if got.UserID != tt.adminUserID {
				t.Errorf("UserID = %q, want %q", got.UserID, tt.adminUserID)
			}
			if got.AssignedBy != tt.actorUserID {
				t.Errorf("AssignedBy = %q, want %q — the assignment records who made it, not merely that it exists", got.AssignedBy, tt.actorUserID)
			}
			if !got.AssignedAt.Equal(adminAssignedAt) {
				t.Errorf("AssignedAt = %v, want %v", got.AssignedAt, adminAssignedAt)
			}
		})
	}
}

// TestAssignGameAdmin_AnAdminCannotAppointAnAdmin states instruction 6's
// property on its own, outside the table, because it is the one a future
// "simplification" is most likely to break: routing assignment through
// Game.EnsureHostOrGameAdmin instead of Game.EnsureHost would still pass every
// happy-path row above while silently making every admin an appointer.
//
// Note what makes the property hold *by construction* rather than by a second
// check: an admin is, by ErrHostCannotBeGameAdmin above, never the Host, so
// the Host-only gate is the whole enforcement. There is no separate
// "is the actor merely an admin" branch that could be forgotten.
func TestAssignGameAdmin_AnAdminCannotAppointAnAdmin(t *testing.T) {
	t.Parallel()

	game := adminFixtureGame(t)
	existing := []domain.GameAdmin{{GameID: game.ID, UserID: adminUserID, AssignedBy: adminHostID, AssignedAt: adminAssignedAt}}

	if _, err := domain.AssignGameAdmin(game, existing, adminUserID, adminSecondary, adminAssignedAt); !errors.Is(err, domain.ErrNotGameHost) {
		t.Fatalf("an assigned Game Admin appointing another admin got %v, want ErrNotGameHost — "+
			"a Game Admin who can appoint Game Admins makes the Host-only rule unenforceable (#168)", err)
	}
	if err := domain.EnsureMayRevokeGameAdmin(game, adminUserID); !errors.Is(err, domain.ErrNotGameHost) {
		t.Fatalf("an assigned Game Admin revoking an admin got %v, want ErrNotGameHost — "+
			"appointing and un-appointing are the same authority and must be gated identically", err)
	}
}

func TestEnsureMayRevokeGameAdmin(t *testing.T) {
	t.Parallel()

	game := adminFixtureGame(t)

	tests := []struct {
		name        string
		actorUserID string
		wantErr     error
	}{
		{name: "the host revokes", actorUserID: adminHostID},
		{name: "an assigned admin revokes", actorUserID: adminUserID, wantErr: domain.ErrNotGameHost},
		{name: "a stranger revokes", actorUserID: adminStranger, wantErr: domain.ErrNotGameHost},
		{name: "an unidentified caller revokes", actorUserID: "", wantErr: domain.ErrNotGameHost},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := domain.EnsureMayRevokeGameAdmin(game, tt.actorUserID)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("EnsureMayRevokeGameAdmin() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// TestHasGameAdmin covers the pure membership helper T14.5 resolves its
// entitled set with. The blank-entry row is the one that matters: it mirrors
// Game.EnsureHostOrGameAdmin's existing `adminID != ""` guard, so an
// unidentified caller can never match a malformed row.
func TestHasGameAdmin(t *testing.T) {
	t.Parallel()

	admins := []domain.GameAdmin{
		{UserID: adminUserID},
		{UserID: adminSecondary},
	}

	tests := []struct {
		name   string
		admins []domain.GameAdmin
		userID string
		want   bool
	}{
		{name: "an assigned admin", admins: admins, userID: adminUserID, want: true},
		{name: "a second assigned admin", admins: admins, userID: adminSecondary, want: true},
		{name: "a stranger", admins: admins, userID: adminStranger, want: false},
		{name: "an empty user id against real rows", admins: admins, userID: "", want: false},
		{name: "an empty user id against a blank row", admins: []domain.GameAdmin{{UserID: ""}}, userID: "", want: false},
		{name: "no admins at all", admins: nil, userID: adminUserID, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := domain.HasGameAdmin(tt.admins, tt.userID); got != tt.want {
				t.Fatalf("HasGameAdmin(%q) = %v, want %v", tt.userID, got, tt.want)
			}
		})
	}
}
