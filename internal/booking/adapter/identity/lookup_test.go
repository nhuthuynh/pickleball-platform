// Package identity_test covers internal/booking/adapter/identity, the one
// place Booking is allowed to call into the Identity context.
//
// These tests deliberately exercise the REAL identityapp.Service over a real
// port.Repository fake, NOT a fake port.IdentityLookup. That distinction is
// the entire point of this file: issue #146 shipped because every existing
// Booking test fakes the IdentityLookup port wholesale, so the real
// GetUser/UserBySubject behaviour was never once exercised with a real,
// non-uuid IdP subject — the only actor value this adapter is ever given in
// production since T12.7 made the actor `auth.RequireSubject(ctx)`.
package identity_test

import (
	"context"
	"errors"
	"testing"

	bookingidentity "github.com/nhuthuynh/white-label/internal/booking/adapter/identity"
	bookingdomain "github.com/nhuthuynh/white-label/internal/booking/domain"
	identityapp "github.com/nhuthuynh/white-label/internal/identity/app"
	identitydomain "github.com/nhuthuynh/white-label/internal/identity/domain"
)

// fixtureSubject is deliberately NOT uuid-shaped: an IdP subject is an opaque
// provider string, and that is exactly why passing it to GetUser (which
// guards on uuidShape) could never resolve. Mirrors the fixture in
// internal/identity/app's own tests.
const fixtureSubject = "auth0|abc123"

// fixtureUserID is the server-minted uuid primary key. Since T12.9 the ID and
// the Subject are two different identifier spaces, which is the whole subject
// of issue #146.
const fixtureUserID = "6ba7b810-9dad-11d1-80b4-00c04fd430c8"

// inMemoryRepo is a minimal identity port.Repository fake. Only the two read
// methods this adapter can reach are meaningfully implemented; the writes
// exist to satisfy the interface.
type inMemoryRepo struct {
	users map[string]identitydomain.User
}

func newInMemoryRepo() *inMemoryRepo {
	return &inMemoryRepo{users: make(map[string]identitydomain.User)}
}

func (r *inMemoryRepo) Create(_ context.Context, u identitydomain.User) (identitydomain.User, error) {
	for _, existing := range r.users {
		if existing.Subject == u.Subject {
			return identitydomain.User{}, identitydomain.ErrUserAlreadyExists
		}
	}
	r.users[u.ID] = u
	return u, nil
}

func (r *inMemoryRepo) GetByID(_ context.Context, id string) (identitydomain.User, error) {
	u, ok := r.users[id]
	if !ok {
		return identitydomain.User{}, identitydomain.ErrUserNotFound
	}
	return u, nil
}

func (r *inMemoryRepo) GetBySubject(_ context.Context, subject string) (identitydomain.User, error) {
	for _, u := range r.users {
		if u.Subject == subject {
			return u, nil
		}
	}
	return identitydomain.User{}, identitydomain.ErrUserNotFound
}

func (r *inMemoryRepo) UpdateSelfReportedLevel(_ context.Context, id string, level identitydomain.SelfReportedStartingLevel) (identitydomain.User, error) {
	u, ok := r.users[id]
	if !ok {
		return identitydomain.User{}, identitydomain.ErrUserNotFound
	}
	u.SelfReportedStartingLevel = level
	r.users[id] = u
	return u, nil
}

// stubIDs is a deterministic port.IDGenerator. Nothing in this file creates a
// User through the service (CreateUser only ever accepts RolePlayer, and
// these tests need RoleClub), so it is never actually called — it exists
// because NewService requires the port.
type stubIDs struct{}

func (stubIDs) NewID() string { return fixtureUserID }

// newLookup builds the adapter over a REAL identityapp.Service, seeded by
// writing the User straight through the repository fake. Seeding via the repo
// rather than via Service.CreateUser is required, not a shortcut: CreateUser
// rejects any role other than RolePlayer (self-registration must never mint a
// privileged role), and every case here needs a User holding RoleClub.
func newLookup(t *testing.T, u identitydomain.User) *bookingidentity.Lookup {
	t.Helper()

	repo := newInMemoryRepo()
	if u.ID != "" {
		if _, err := repo.Create(context.Background(), u); err != nil {
			t.Fatalf("seeding repo: %v", err)
		}
	}
	return bookingidentity.NewLookup(identityapp.NewService(repo, stubIDs{}))
}

func mustUser(t *testing.T, id, subject string, roles []identitydomain.Role) identitydomain.User {
	t.Helper()

	u, err := identitydomain.NewUser(id, subject, "Ada Lovelace", roles, identitydomain.SelfReportedStartingLevel(3))
	if err != nil {
		t.Fatalf("building fixture user: %v", err)
	}
	return u
}

// TestEnsureClubRole_ResolvesBySubject is the issue #146 regression test.
//
// The first case is the one that fails against the pre-fix adapter: a real
// club User exists, the caller presents the very subject that User is
// registered under, and EnsureClubRole must succeed. Before the fix the
// adapter called GetUser, whose uuidShape guard rejected "auth0|abc123"
// before it ever reached the repository, so this returned
// bookingdomain.ErrUserNotFound and RequestRecurringHire answered NotFound to
// every caller alive.
func TestEnsureClubRole_ResolvesBySubject(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		seed    identitydomain.User
		actor   string
		wantErr error
	}{
		{
			name:    "club user resolved by non-uuid IdP subject",
			seed:    mustUser(t, fixtureUserID, fixtureSubject, []identitydomain.Role{identitydomain.RoleClub}),
			actor:   fixtureSubject,
			wantErr: nil,
		},
		{
			name:    "club role among several roles",
			seed:    mustUser(t, fixtureUserID, fixtureSubject, []identitydomain.Role{identitydomain.RolePlayer, identitydomain.RoleClub}),
			actor:   fixtureSubject,
			wantErr: nil,
		},
		{
			name:    "registered subject without the club role",
			seed:    mustUser(t, fixtureUserID, fixtureSubject, []identitydomain.Role{identitydomain.RolePlayer}),
			actor:   fixtureSubject,
			wantErr: bookingdomain.ErrNotClub,
		},
		{
			name:    "unregistered subject",
			seed:    mustUser(t, fixtureUserID, fixtureSubject, []identitydomain.Role{identitydomain.RoleClub}),
			actor:   "auth0|nobody",
			wantErr: bookingdomain.ErrUserNotFound,
		},
		{
			name:    "empty subject",
			seed:    mustUser(t, fixtureUserID, fixtureSubject, []identitydomain.Role{identitydomain.RoleClub}),
			actor:   "",
			wantErr: bookingdomain.ErrUserNotFound,
		},
		{
			// The User's uuid primary key must NOT resolve: the actor
			// crossing this boundary is a subject, and the two identifier
			// spaces are deliberately distinct since T12.9. This case is what
			// stops a future "just accept both" patch from quietly
			// reintroducing the ambiguity.
			name:    "server-minted user id is not an actor subject",
			seed:    mustUser(t, fixtureUserID, fixtureSubject, []identitydomain.Role{identitydomain.RoleClub}),
			actor:   fixtureUserID,
			wantErr: bookingdomain.ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			lookup := newLookup(t, tt.seed)

			err := lookup.EnsureClubRole(context.Background(), tt.actor)

			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("EnsureClubRole(%q) = %v, want nil", tt.actor, err)
				}
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("EnsureClubRole(%q) = %v, want %v", tt.actor, err, tt.wantErr)
			}
		})
	}
}

// TestEnsureClubRole_NeverLeaksIdentitySentinels holds CLAUDE.md rule 5 at
// this boundary: whatever Identity returns, a Booking-side caller must only
// ever be able to errors.Is() against Booking's own sentinels.
func TestEnsureClubRole_NeverLeaksIdentitySentinels(t *testing.T) {
	t.Parallel()

	lookup := newLookup(t, mustUser(t, fixtureUserID, fixtureSubject, []identitydomain.Role{identitydomain.RolePlayer}))

	err := lookup.EnsureClubRole(context.Background(), "auth0|nobody")

	if errors.Is(err, identitydomain.ErrUserNotFound) {
		t.Fatalf("EnsureClubRole leaked the identity sentinel: %v", err)
	}
	if !errors.Is(err, bookingdomain.ErrUserNotFound) {
		t.Fatalf("EnsureClubRole = %v, want bookingdomain.ErrUserNotFound", err)
	}
}
