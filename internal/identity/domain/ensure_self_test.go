package domain_test

import (
	"errors"
	"testing"

	"github.com/nhuthuynh/white-label/internal/identity/domain"
)

// TestUser_EnsureSelf is T10.2's object-level (BOLA) authorization proof at
// the domain layer, mirroring internal/facilities/domain.Facility.EnsureOwner
// and internal/socialplay/domain.Registration's actorPlayerID-vs-PlayerID
// check applied to User's own identity fact: UpdateSelfReportedLevel must
// only be permitted when the caller-claimed actor_user_id matches the target
// User's own ID exactly. As with those precedents, actorUserID is a
// caller-supplied claim, not a verified identity (no JWT/Auth0 exists yet).
func TestUser_EnsureSelf(t *testing.T) {
	t.Parallel()

	u, err := domain.NewUser("user-1", "auth0|ada", "Ada Lovelace", []domain.Role{domain.RolePlayer}, domain.SelfReportedStartingLevel(3))
	if err != nil {
		t.Fatalf("unexpected err constructing fixture: %v", err)
	}

	tests := []struct {
		name        string
		actorUserID string
		wantErr     error
	}{
		{"matching actor is allowed", "user-1", nil},
		{"mismatched actor is rejected", "user-2", domain.ErrNotSelf},
		{"empty actor is rejected", "", domain.ErrNotSelf},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := u.EnsureSelf(tt.actorUserID)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("EnsureSelf(%q) err = %v, want %v", tt.actorUserID, err, tt.wantErr)
			}
		})
	}
}
