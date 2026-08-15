// T15.3 — the pure rules behind a durable Competition-Admin assignment (#168,
// partial fix). Table-driven, dependency-free: nothing here touches a
// repository, a database or a wire type, which is the whole point of putting
// the rules in domain rather than in app.Service.
//
// The exact mirror of internal/socialplay/domain/game_admin_test.go (T14.4),
// case for case — T15.3's instruction 1 is "copy T14.4's shape deliberately;
// do not redesign it", and a test file that diverged from its template would
// make the two contexts' guarantees quietly different rather than
// deliberately identical.
//
// The rule this file exists to pin hardest is instruction 2's: **an admin must
// not be able to appoint an admin.** #168 states why in one line — "or the
// Host-only distinction ErrNotCompetitionHost exists to protect is worthless"
// — and TestAssignCompetitionAdmin_AnAdminCannotAppointAnAdmin below is the
// assertion that fails if a future change ever routes assignment through a
// host-or-admin check.
package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/nhuthuynh/white-label/internal/competitions/domain"
)

const (
	compAdminHostID    = "host-subject"
	compAdminUserID    = "admin-subject"
	compAdminStranger  = "stranger-subject"
	compAdminSecondary = "second-admin-subject"
)

var compAdminAssignedAt = time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)

// compAdminFixtureCompetition builds a scheduled Competition hosted by
// compAdminHostID. It goes through domain.NewCompetition rather than a struct
// literal so the fixture can never drift from a Competition the constructor
// would actually accept.
func compAdminFixtureCompetition(t *testing.T) domain.Competition {
	t.Helper()

	start := time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC)
	rng, err := domain.NewTimeRange(start, start.Add(time.Hour))
	if err != nil {
		t.Fatalf("bad fixture range: %v", err)
	}
	c, err := domain.NewCompetition(
		"11111111-1111-4111-8111-111111111111",
		compAdminHostID,
		"Autumn Open",
		"",
		[]domain.Session{{
			Range:    rng,
			CourtIDs: []string{"22222222-2222-4222-8222-222222222222"},
		}},
		8,
		0,
		domain.PaymentMethodEither,
		domain.Money{},
		domain.FormatDoubles,
		"",
	)
	if err != nil {
		t.Fatalf("bad fixture competition: %v", err)
	}
	return c
}

func TestAssignCompetitionAdmin(t *testing.T) {
	t.Parallel()

	competition := compAdminFixtureCompetition(t)
	alreadyAssigned := []domain.CompetitionAdmin{{
		CompetitionID: competition.ID,
		UserID:        compAdminUserID,
		AssignedBy:    compAdminHostID,
		AssignedAt:    compAdminAssignedAt,
	}}

	tests := []struct {
		name        string
		existing    []domain.CompetitionAdmin
		actorUserID string
		adminUserID string
		wantErr     error
		why         string
	}{
		{
			name:        "the host assigns a new admin",
			actorUserID: compAdminHostID,
			adminUserID: compAdminUserID,
			why:         "the whole point of the aggregate: the Host, and only the Host, delegates",
		},
		{
			name:        "the host assigns a second admin",
			existing:    alreadyAssigned,
			actorUserID: compAdminHostID,
			adminUserID: compAdminSecondary,
			why:         "an existing assignment does not close the Competition to further ones",
		},
		{
			name:        "an assigned admin tries to appoint another admin",
			existing:    alreadyAssigned,
			actorUserID: compAdminUserID,
			adminUserID: compAdminSecondary,
			wantErr:     domain.ErrNotCompetitionHost,
			why: "#168's own words: an admin must not appoint an admin, " +
				"or the Host-only distinction ErrNotCompetitionHost exists to protect is worthless",
		},
		{
			name:        "a stranger tries to assign",
			actorUserID: compAdminStranger,
			adminUserID: compAdminSecondary,
			wantErr:     domain.ErrNotCompetitionHost,
		},
		{
			name:        "an unidentified caller tries to assign",
			actorUserID: "",
			adminUserID: compAdminUserID,
			wantErr:     domain.ErrNotCompetitionHost,
			why:         "EnsureHost rejects an empty actor even against a Competition with an empty HostID",
		},
		{
			name:        "the host assigns an empty user id",
			actorUserID: compAdminHostID,
			adminUserID: "",
			wantErr:     domain.ErrEmptyCompetitionAdminUserID,
			why:         "a blank admin row would match a blank actor at read time — the exact accident HasCompetitionAdmin's blank-entry guard exists to prevent",
		},
		{
			name:        "the host assigns themselves",
			actorUserID: compAdminHostID,
			adminUserID: compAdminHostID,
			wantErr:     domain.ErrHostCannotBeCompetitionAdmin,
			why:         "the Host is already entitled; a row implying delegation to themselves would make a later revoke look like it removed Host authority, which it cannot",
		},
		{
			name:        "the host assigns someone already assigned",
			existing:    alreadyAssigned,
			actorUserID: compAdminHostID,
			adminUserID: compAdminUserID,
			wantErr:     domain.ErrAlreadyCompetitionAdmin,
			why:         "mirrors domain.Enter's ErrAlreadyEntered pre-check; the composite primary key is the authoritative guard (CLAUDE.md rule 4)",
		},
		{
			name:        "authorization is checked before the input",
			existing:    alreadyAssigned,
			actorUserID: compAdminStranger,
			adminUserID: "",
			wantErr:     domain.ErrNotCompetitionHost,
			why: "an unauthorized actor must not learn whether the input would otherwise have been valid — " +
				"the ordering internal/payments/app.authorizeOfflineRecording established",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := domain.AssignCompetitionAdmin(competition, tt.existing, tt.actorUserID, tt.adminUserID, compAdminAssignedAt)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("AssignCompetitionAdmin() error = %v, want %v — %s", err, tt.wantErr, tt.why)
				}
				if got != (domain.CompetitionAdmin{}) {
					t.Errorf("AssignCompetitionAdmin() returned %+v alongside an error, want the zero value", got)
				}
				return
			}

			if err != nil {
				t.Fatalf("AssignCompetitionAdmin() unexpected error: %v — %s", err, tt.why)
			}
			if got.CompetitionID != competition.ID {
				t.Errorf("CompetitionID = %q, want %q", got.CompetitionID, competition.ID)
			}
			if got.UserID != tt.adminUserID {
				t.Errorf("UserID = %q, want %q", got.UserID, tt.adminUserID)
			}
			if got.AssignedBy != tt.actorUserID {
				t.Errorf("AssignedBy = %q, want %q — the assignment records who made it, not merely that it exists", got.AssignedBy, tt.actorUserID)
			}
			if !got.AssignedAt.Equal(compAdminAssignedAt) {
				t.Errorf("AssignedAt = %v, want %v", got.AssignedAt, compAdminAssignedAt)
			}
		})
	}
}

// TestAssignCompetitionAdmin_AnAdminCannotAppointAnAdmin states instruction
// 2's headline property on its own, outside the table, because it is the one a
// future "simplification" is most likely to break: routing assignment through
// a host-or-admin check instead of Competition.EnsureHost would still pass
// every happy-path row above while silently making every admin an appointer.
//
// Note what makes the property hold *by construction* rather than by a second
// check: an admin is, by ErrHostCannotBeCompetitionAdmin above, never the
// Host, so the Host-only gate is the whole enforcement. There is no separate
// "is the actor merely an admin" branch that could be forgotten.
//
// This is the mutation-check target named by instruction 7 — the twin of
// T14.4's TestAssignGameAdmin_AnAdminCannotAppointAnAdmin.
func TestAssignCompetitionAdmin_AnAdminCannotAppointAnAdmin(t *testing.T) {
	t.Parallel()

	competition := compAdminFixtureCompetition(t)
	existing := []domain.CompetitionAdmin{{
		CompetitionID: competition.ID,
		UserID:        compAdminUserID,
		AssignedBy:    compAdminHostID,
		AssignedAt:    compAdminAssignedAt,
	}}

	if _, err := domain.AssignCompetitionAdmin(competition, existing, compAdminUserID, compAdminSecondary, compAdminAssignedAt); !errors.Is(err, domain.ErrNotCompetitionHost) {
		t.Fatalf("an assigned Competition Admin appointing another admin got %v, want ErrNotCompetitionHost — "+
			"a Competition Admin who can appoint Competition Admins makes the Host-only rule unenforceable (#168)", err)
	}
	if err := domain.EnsureMayRevokeCompetitionAdmin(competition, compAdminUserID); !errors.Is(err, domain.ErrNotCompetitionHost) {
		t.Fatalf("an assigned Competition Admin revoking an admin got %v, want ErrNotCompetitionHost — "+
			"appointing and un-appointing are the same authority and must be gated identically", err)
	}
}

func TestEnsureMayRevokeCompetitionAdmin(t *testing.T) {
	t.Parallel()

	competition := compAdminFixtureCompetition(t)

	tests := []struct {
		name        string
		actorUserID string
		wantErr     error
	}{
		{name: "the host revokes", actorUserID: compAdminHostID},
		{name: "an assigned admin revokes", actorUserID: compAdminUserID, wantErr: domain.ErrNotCompetitionHost},
		{name: "a stranger revokes", actorUserID: compAdminStranger, wantErr: domain.ErrNotCompetitionHost},
		{name: "an unidentified caller revokes", actorUserID: "", wantErr: domain.ErrNotCompetitionHost},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := domain.EnsureMayRevokeCompetitionAdmin(competition, tt.actorUserID)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("EnsureMayRevokeCompetitionAdmin() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// TestHasCompetitionAdmin covers the pure membership helper T15.4 resolves its
// entitled set with. The blank-entry row is the one that matters: it mirrors
// domain.HasGameAdmin's identical guard, so an unidentified caller can never
// match a malformed row.
func TestHasCompetitionAdmin(t *testing.T) {
	t.Parallel()

	admins := []domain.CompetitionAdmin{
		{UserID: compAdminUserID},
		{UserID: compAdminSecondary},
	}

	tests := []struct {
		name   string
		admins []domain.CompetitionAdmin
		userID string
		want   bool
	}{
		{name: "an assigned admin", admins: admins, userID: compAdminUserID, want: true},
		{name: "a second assigned admin", admins: admins, userID: compAdminSecondary, want: true},
		{name: "a stranger", admins: admins, userID: compAdminStranger, want: false},
		{name: "an empty user id against real rows", admins: admins, userID: "", want: false},
		{name: "an empty user id against a blank row", admins: []domain.CompetitionAdmin{{UserID: ""}}, userID: "", want: false},
		{name: "no admins at all", admins: nil, userID: compAdminUserID, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := domain.HasCompetitionAdmin(tt.admins, tt.userID); got != tt.want {
				t.Fatalf("HasCompetitionAdmin(%q) = %v, want %v", tt.userID, got, tt.want)
			}
		})
	}
}
