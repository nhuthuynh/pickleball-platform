package domain_test

import (
	"errors"
	"testing"

	"github.com/nhuthuynh/white-label/internal/identity/domain"
)

func validRoles() []domain.Role {
	return []domain.Role{domain.RolePlayer}
}

// TestNewUser_Valid proves the happy path: a well-formed User is
// constructed with every field round-tripped as given.
func TestNewUser_Valid(t *testing.T) {
	t.Parallel()

	u, err := domain.NewUser("user-1", "Ada Lovelace", validRoles(), domain.SelfReportedStartingLevel(3))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if u.ID != "user-1" {
		t.Fatalf("ID = %q, want %q", u.ID, "user-1")
	}
	if u.DisplayName != "Ada Lovelace" {
		t.Fatalf("DisplayName = %q, want %q", u.DisplayName, "Ada Lovelace")
	}
	if len(u.Roles) != 1 || u.Roles[0] != domain.RolePlayer {
		t.Fatalf("Roles = %v, want [player]", u.Roles)
	}
	if u.SelfReportedStartingLevel != 3 {
		t.Fatalf("SelfReportedStartingLevel = %d, want 3", u.SelfReportedStartingLevel)
	}
}

// TestNewUser_MultipleRoles proves a User may hold more than one Role at
// once (e.g. a Host who is also a Player) — Roles is a slice, not a single
// value, precisely to allow this.
func TestNewUser_MultipleRoles(t *testing.T) {
	t.Parallel()

	roles := []domain.Role{domain.RolePlayer, domain.RoleHostOrganiser}
	u, err := domain.NewUser("user-1", "Ada Lovelace", roles, domain.SelfReportedStartingLevel(3))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(u.Roles) != 2 {
		t.Fatalf("Roles = %v, want 2 roles", u.Roles)
	}
}

// TestNewUser_Validation is the required table-driven boundary coverage:
// empty ID, empty DisplayName, empty Roles, an unrecognized Role value, and
// an out-of-range SelfReportedStartingLevel — each rejected with its own
// distinct sentinel error, mirroring booking.NewBooking's and
// socialplay.NewGame's "one sentinel per rejected invariant" convention.
func TestNewUser_Validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		id          string
		displayName string
		roles       []domain.Role
		level       domain.SelfReportedStartingLevel
		wantErr     error
	}{
		{"empty id is rejected", "", "Ada Lovelace", validRoles(), 3, domain.ErrEmptyID},
		{"empty display name is rejected", "user-1", "", validRoles(), 3, domain.ErrEmptyDisplayName},
		{"empty roles is rejected", "user-1", "Ada Lovelace", []domain.Role{}, 3, domain.ErrEmptyRoles},
		{"nil roles is rejected", "user-1", "Ada Lovelace", nil, 3, domain.ErrEmptyRoles},
		{"unrecognized role is rejected", "user-1", "Ada Lovelace", []domain.Role{domain.Role("coach")}, 3, domain.ErrInvalidRole},
		{"one valid role among an invalid one is still rejected", "user-1", "Ada Lovelace", []domain.Role{domain.RolePlayer, domain.Role("coach")}, 3, domain.ErrInvalidRole},
		{"level zero is rejected", "user-1", "Ada Lovelace", validRoles(), 0, domain.ErrInvalidSelfReportedStartingLevel},
		{"level below range is rejected", "user-1", "Ada Lovelace", validRoles(), -1, domain.ErrInvalidSelfReportedStartingLevel},
		{"level above range is rejected", "user-1", "Ada Lovelace", validRoles(), 6, domain.ErrInvalidSelfReportedStartingLevel},
		{"level at minimum is accepted", "user-1", "Ada Lovelace", validRoles(), 1, nil},
		{"level at maximum is accepted", "user-1", "Ada Lovelace", validRoles(), 5, nil},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := domain.NewUser(tt.id, tt.displayName, tt.roles, tt.level)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got err %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// TestRole_IsValid proves Role is a closed enum: exactly the six roles per
// docs/agent-operating-handbook.md A1 are valid, everything else — including
// the empty string — is not.
func TestRole_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		role domain.Role
		want bool
	}{
		{"player is valid", domain.RolePlayer, true},
		{"host_organiser is valid", domain.RoleHostOrganiser, true},
		{"game_admin is valid", domain.RoleGameAdmin, true},
		{"facility_owner is valid", domain.RoleFacilityOwner, true},
		{"club is valid", domain.RoleClub, true},
		{"platform_admin is valid", domain.RolePlatformAdmin, true},
		{"empty role is invalid", domain.Role(""), false},
		{"unknown role is invalid", domain.Role("coach"), false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.role.IsValid(); got != tt.want {
				t.Fatalf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestSelfReportedStartingLevel_IsValid proves the bounded 1..5 range
// directly, independent of NewUser, so a future caller of the value type on
// its own (e.g. an UpdateSelfReportedLevel RPC handler, T10.2) can rely on
// it too.
func TestSelfReportedStartingLevel_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		level domain.SelfReportedStartingLevel
		want  bool
	}{
		{"zero is invalid", 0, false},
		{"negative is invalid", -1, false},
		{"one is valid", 1, true},
		{"three is valid", 3, true},
		{"five is valid", 5, true},
		{"six is invalid", 6, false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.level.IsValid(); got != tt.want {
				t.Fatalf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}
