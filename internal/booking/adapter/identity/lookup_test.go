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

// TestUserIDBySubject_TranslatesTheVerifiedSubject is the issue #146
// regression test, moved to the method that now owns the subject space
// (ADR-0014). The first case is the one that fails against the pre-#151
// adapter: a real club User exists, the caller presents the very subject that
// User is registered under, and the translation must produce that User's uuid.
//
// **The returned id is asserted, not just the absence of an error.** A
// translation that "succeeded" while returning the subject unchanged would
// satisfy an error-only assertion and then panic the Postgres adapter's
// mustUUID one layer down — the exact outcome #152 names as worse than the
// bug it reports.
func TestUserIDBySubject_TranslatesTheVerifiedSubject(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		seed    identitydomain.User
		subject string
		wantID  string
		wantErr error
	}{
		{
			name:    "non-uuid IdP subject resolves to the server-minted uuid",
			seed:    mustUser(t, fixtureUserID, fixtureSubject, []identitydomain.Role{identitydomain.RoleClub}),
			subject: fixtureSubject,
			wantID:  fixtureUserID,
		},
		{
			// Resolution answers "who are you", never "what may you do": a
			// Player resolves exactly as readily as a Club. The role check is
			// a separate question, asked separately by EnsureClubRole.
			name:    "resolution is independent of the User's roles",
			seed:    mustUser(t, fixtureUserID, fixtureSubject, []identitydomain.Role{identitydomain.RolePlayer}),
			subject: fixtureSubject,
			wantID:  fixtureUserID,
		},
		{
			name:    "unregistered subject",
			seed:    mustUser(t, fixtureUserID, fixtureSubject, []identitydomain.Role{identitydomain.RoleClub}),
			subject: "auth0|nobody",
			wantErr: bookingdomain.ErrUserNotFound,
		},
		{
			// identityapp.Service.UserBySubject short-circuits the empty
			// subject rather than querying for it, and answers it exactly
			// like an unregistered one.
			name:    "empty subject",
			seed:    mustUser(t, fixtureUserID, fixtureSubject, []identitydomain.Role{identitydomain.RoleClub}),
			subject: "",
			wantErr: bookingdomain.ErrUserNotFound,
		},
		{
			// The User's uuid primary key must NOT resolve as a subject. The
			// two identifier spaces are deliberately distinct since T12.9,
			// and this case is what stops a future "just accept both" patch
			// from quietly reintroducing the ambiguity that produced #146.
			name:    "server-minted user id is not a subject",
			seed:    mustUser(t, fixtureUserID, fixtureSubject, []identitydomain.Role{identitydomain.RoleClub}),
			subject: fixtureUserID,
			wantErr: bookingdomain.ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			lookup := newLookup(t, tt.seed)

			got, err := lookup.UserIDBySubject(context.Background(), tt.subject)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("UserIDBySubject(%q) error = %v, want %v", tt.subject, err, tt.wantErr)
				}
				if got != "" {
					t.Fatalf("UserIDBySubject(%q) = %q on the error path, want the zero value", tt.subject, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("UserIDBySubject(%q) = %v, want nil", tt.subject, err)
			}
			if got != tt.wantID {
				t.Fatalf("UserIDBySubject(%q) = %q, want %q — a subject returned unchanged "+
					"would panic the Postgres adapter's mustUUID (#152)", tt.subject, got, tt.wantID)
			}
		})
	}
}

// TestEnsureClubRole_KeysOnUserIDNotSubject pins the OTHER half of ADR-0014's
// identifier-space split, and is the test that stops the two methods on this
// adapter from drifting back into ambiguity.
//
// EnsureClubRole is reached only after the handler's actor() funnel has
// resolved the subject, so what it receives is a User.ID. The subject case
// below is the important one: it asserts that a subject reaching this method
// is REFUSED rather than resolved. That is what makes a bypass of the seam
// fail closed and loudly, instead of silently authorizing whoever happens to
// match.
func TestEnsureClubRole_KeysOnUserIDNotSubject(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		seed    identitydomain.User
		actor   string
		wantErr error
	}{
		{
			name:  "club user resolved by their User.ID",
			seed:  mustUser(t, fixtureUserID, fixtureSubject, []identitydomain.Role{identitydomain.RoleClub}),
			actor: fixtureUserID,
		},
		{
			name:  "club role among several roles",
			seed:  mustUser(t, fixtureUserID, fixtureSubject, []identitydomain.Role{identitydomain.RolePlayer, identitydomain.RoleClub}),
			actor: fixtureUserID,
		},
		{
			name:    "registered user without the club role",
			seed:    mustUser(t, fixtureUserID, fixtureSubject, []identitydomain.Role{identitydomain.RolePlayer}),
			actor:   fixtureUserID,
			wantErr: bookingdomain.ErrNotClub,
		},
		{
			name:    "unknown User.ID",
			seed:    mustUser(t, fixtureUserID, fixtureSubject, []identitydomain.Role{identitydomain.RoleClub}),
			actor:   "6ba7b810-9dad-11d1-80b4-00c04fd430ff",
			wantErr: bookingdomain.ErrUserNotFound,
		},
		{
			// The seam's contract, asserted from the other side: a subject is
			// not an actor id below the grpcapi boundary, and this method
			// must never resolve one. GetUser's uuidShape guard is what makes
			// this fail closed.
			name:    "a raw subject is refused, never resolved",
			seed:    mustUser(t, fixtureUserID, fixtureSubject, []identitydomain.Role{identitydomain.RoleClub}),
			actor:   fixtureSubject,
			wantErr: bookingdomain.ErrUserNotFound,
		},
		{
			name:    "empty actor",
			seed:    mustUser(t, fixtureUserID, fixtureSubject, []identitydomain.Role{identitydomain.RoleClub}),
			actor:   "",
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

// TestLookup_NeverLeaksIdentitySentinels holds CLAUDE.md rule 5 at this
// boundary, for BOTH methods: whatever Identity returns, a Booking-side caller
// must only ever be able to errors.Is() against Booking's own sentinels.
// Covering both matters because they take different routes into translate() —
// UserBySubject and GetUser — and only one of them was covered before T13.2.
func TestLookup_NeverLeaksIdentitySentinels(t *testing.T) {
	t.Parallel()

	lookup := newLookup(t, mustUser(t, fixtureUserID, fixtureSubject, []identitydomain.Role{identitydomain.RolePlayer}))

	t.Run("EnsureClubRole", func(t *testing.T) {
		t.Parallel()

		err := lookup.EnsureClubRole(context.Background(), "6ba7b810-9dad-11d1-80b4-00c04fd430ff")

		if errors.Is(err, identitydomain.ErrUserNotFound) {
			t.Fatalf("EnsureClubRole leaked the identity sentinel: %v", err)
		}
		if !errors.Is(err, bookingdomain.ErrUserNotFound) {
			t.Fatalf("EnsureClubRole = %v, want bookingdomain.ErrUserNotFound", err)
		}
	})

	t.Run("UserIDBySubject", func(t *testing.T) {
		t.Parallel()

		_, err := lookup.UserIDBySubject(context.Background(), "auth0|nobody")

		if errors.Is(err, identitydomain.ErrUserNotFound) {
			t.Fatalf("UserIDBySubject leaked the identity sentinel: %v", err)
		}
		if !errors.Is(err, bookingdomain.ErrUserNotFound) {
			t.Fatalf("UserIDBySubject = %v, want bookingdomain.ErrUserNotFound", err)
		}
	})
}
